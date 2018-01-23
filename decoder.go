package wbxml

import (
	"fmt"
	"io"
	"unicode/utf8"
)

type Decoder struct {
	r     io.Reader
	tags  CodeSpace
	attrs CodeSpace

	offset  int
	tokChan chan Token
	err     error
	Header  Header
}

func NewDecoder(r io.Reader, tags CodeSpace, attrs CodeSpace) *Decoder {
	d := &Decoder{
		r:       r,
		tags:    tags,
		attrs:   attrs,
		tokChan: make(chan Token),
	}

	go d.run()
	return d
}

// Token returns the next token in the input stream, or nil and io.EOF at the end.
func (d *Decoder) Token() (Token, error) {
	tok := <-d.tokChan
	if tok == nil {
		return tok, d.err
	}
	return tok, nil
}

func (d *Decoder) GetString(i uint32) ([]byte, error) {
	if i >= uint32(len(d.Header.StringTable)) {
		return nil, fmt.Errorf("%d is not a valid string reference (max %d)", i, len(d.Header.StringTable))
	}
	for end, b := range d.Header.StringTable[i:] {
		if b == 0 {
			return d.Header.StringTable[i : i+uint32(end)], nil
		}
	}
	return nil, fmt.Errorf("StringTable: no NULL terminator found")
}

func (d *Decoder) tagName(page, code byte) string {
	name, err := d.tags.Name(page, code)
	if err != nil {
		d.panicErr(err)
	}
	return name
}

func (d *Decoder) attrName(page, code byte) string {
	name, err := d.attrs.Name(page, code)
	if err != nil {
		d.panicErr(err)
	}
	return name
}

func (d *Decoder) run() {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				if err == io.EOF {
					d.err = err
				} else {
					panic(err)
				}
			}
			close(d.tokChan)
		}
	}()

	h, err := d.readHeader()
	d.panicErr(err)
	d.Header = h
	d.body()
	close(d.tokChan)
}

// readHeader reads the wbxml header.
func (d *Decoder) readHeader() (Header, error) {
	var h Header
	var err error

	h.Version, err = readByte(d)
	if err != nil {
		return h, err
	}

	h.PublicID, err = mbUint32(d)
	if err != nil {
		return h, err
	}
	if h.PublicID == 0 {
		h.PublicID, err = mbUint32(d)
	}

	h.Charset, err = mbUint32(d)
	if err != nil {
		return h, err
	}

	length, err := mbUint32(d)
	if err != nil {
		return h, err
	}
	buf := make([]byte, length)
	n, err := d.r.Read(buf)
	if err != nil {
		return h, err
	}
	d.offset += n
	h.StringTable = buf
	return h, nil
}

func (d *Decoder) body() {
	var b byte
	var err error

	for {
		b, err = readByte(d)
		d.panicErr(err)
		if b != pi {
			break
		}
		d.piStar()
	}

	d.element(b)

	for {
		b, err = readByte(d)
		d.panicErr(err)
		if b != pi {
			break
		}
		d.piStar()
	}
}

func (d *Decoder) piStar() {
}

func (d *Decoder) element(b byte) {
	var page byte

	switch b {
	case switchPage:
		index, err := readByte(d)
		d.panicErr(err)
		page = index
	case literal:
		panic(fmt.Errorf("literal tag not implemented"))
	default:
		tag := Tag(b)
		tagName := d.tagName(page, tag.ID())
		tok := StartElement{Name: tagName}
		if tag.Attr() {
			d.attributes(&tok)
		}
		d.tokChan <- tok
		if tag.Content() {
			d.content()
		}
		d.tokChan <- EndElement{Name: tagName}
	}
}

func (d *Decoder) attributes(elt *StartElement) {
	var page byte
	b, err := readByte(d)
	d.panicErr(err)

	for {
		switch b {
		case switchPage:
			index, err := readByte(d)
			d.panicErr(err)
			page = index
		case literal:
			var attr Attr
			index, err := mbUint32(d)
			d.panicErr(err)
			name, err := d.GetString(index)
			d.panicErr(err)
			attr.Name = string(name)
			attr.Value, b = d.readAttrValue(page)
			elt.Attr = append(elt.Attr, attr)
		case end:
			return
		default:
			if b >= 128 {
				panic(fmt.Errorf("unexpected attribute value"))
			}
			var attr Attr
			attr.Name = d.attrName(page, b)
			attr.Value, b = d.readAttrValue(page)
			elt.Attr = append(elt.Attr, attr)
		}
	}
}

func (d *Decoder) readAttrValue(page byte) (string, byte) {

	var cdata CharData
	for {
		b, err := readByte(d)
		d.panicErr(err)

		switch b {
		case strI, strT, entity:
			d.charData(&cdata, b)
		case ext0, ext1, ext2,
			extI0, extI1, extI2,
			extT0, extT1, extT2:
			panic(fmt.Errorf("extension token unimplemented (token %d)", b))
		case end:
			return string(cdata), b
		default:
			if b < 128 {
				return string(cdata), b
				//panic(fmt.Errorf("unexpected attribute tag name %d", b))
			}
			cdata = append(cdata, []byte(d.attrName(page, b))...)
		}
	}
}

func (d *Decoder) content() {
	// content() accumulate adjacent CharData in a unique instance until END or ELEMENT is
	// encountered

	var cdata CharData
	for {
		b, err := readByte(d)
		d.panicErr(err)

		switch b {
		case strI, strT, entity:
			d.charData(&cdata, b)
		case opaque:
			d.sendCharData(&cdata)
			length, err := mbUint32(d)
			d.panicErr(err)
			data, err := readSlice(d, length)
			d.panicErr(err)
			d.tokChan <- Opaque(data)
		case ext0, ext1, ext2,
			extI0, extI1, extI2,
			extT0, extT1, extT2:
			panic(fmt.Errorf("extension token unimplemented (token %d)", b))
		case end:
			d.sendCharData(&cdata)
			return
		default:
			d.sendCharData(&cdata)
			d.element(b)
		}
	}
}

func (d *Decoder) sendCharData(cdata *CharData) {
	if *cdata != nil {
		d.tokChan <- *cdata
		*cdata = nil
	}
}

func (d *Decoder) charData(cdata *CharData, b byte) {
	if *cdata == nil {
		*cdata = make([]byte, 0)
	}
	switch b {
	case strI:
		str, err := readString(d)
		d.panicErr(err)
		*cdata = append(*cdata, str...)
	case strT:
		index, err := mbUint32(d)
		d.panicErr(err)
		str, err := d.GetString(index)
		d.panicErr(err)
		*cdata = append(*cdata, str...)
	case entity:
		entcode, err := mbUint32(d)
		d.panicErr(err)
		if len(*cdata) > 0 {
			var buf [4]byte
			rlen := utf8.RuneLen(rune(entcode))
			utf8.EncodeRune(buf[:rlen], rune(entcode))
			d.panicErr(err)
			*cdata = append(*cdata, buf[:rlen]...)
		} else {
			d.tokChan <- Entity(entcode)
		}
	default:
		d.panicErr(fmt.Errorf("Unknown char data tag %d", b))
	}
}
