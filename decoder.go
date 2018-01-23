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
// The input stream is limited to the body part, the header is read on initialization or
// by yourself using ReadHeader.
//
// Grammar is:
//   start		= version publicid charset strtbl body
//   strtbl		= length *byte
//   body		= *pi element *pi
//   element	= stag [ 1*attribute END ] [ *content END ]
//
//   content	= element | string | extension | entity | pi | opaque
//
//   stag		= TAG | ( LITERAL index )
//   attribute	= attrStart *attrValue
//   attrStart	= ATTRSTART | ( LITERAL index )
//   attrValue	= ATTRVALUE | string | extension | entity
//
//   extension	= ( EXT_I termstr ) | ( EXT_T index ) | EXT
//
//   string		= inline | tableref
//   inline		= STR_I termstr
//   tableref	= STR_T index
//
//   entity		= ENTITY entcode
//   entcode	= mb_u_int32			// UCS-4 character code
//
//   pi			= PI attrStart *attrValue END
//
//   opaque		= OPAQUE length *byte
//
//   version	= u_int8 containing WBXML version number
//   publicid	= mb_u_int32 | ( zero index )
//   charset	= mb_u_int32
//   termstr	= charset-dependent string with termination
//   index		= mb_u_int32			// integer index into string table.
//   length		= mb_u_int32			// integer length.
//   zero		= u_int8			// containing the value zero (0).
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
	panicErr(err)
	d.Header = h
	d.body()
	close(d.tokChan)
}

// readHeader reads the wbxml header.
func (d *Decoder) readHeader() (Header, error) {
	var h Header
	var err error

	h.Version, err = readByte(d.r)
	if err != nil {
		return h, err
	}

	h.PublicID, err = mbUint32(d.r)
	if err != nil {
		return h, err
	}
	if h.PublicID == 0 {
		h.PublicID, err = mbUint32(d.r)
	}

	h.Charset, err = mbUint32(d.r)
	if err != nil {
		return h, err
	}

	length, err := mbUint32(d.r)
	if err != nil {
		return h, err
	}
	buf := make([]byte, length)
	_, err = d.r.Read(buf)
	if err != nil {
		return h, err
	}
	h.StringTable = buf
	return h, nil
}

func (d *Decoder) body() {
	var b byte
	var err error

	for {
		b, err = readByte(d.r)
		panicErr(err)
		if b != pi {
			break
		}
		d.piStar()
	}

	d.element(b)

	for {
		b, err = readByte(d.r)
		panicErr(err)
		if b != pi {
			break
		}
		d.piStar()
	}
}

func (d *Decoder) piStar() {
}

func (d *Decoder) element(b byte) {
	switch b {
	case switchPage:
		panic(fmt.Errorf("unexpected token switchPage"))
	case literal:
		panic(fmt.Errorf("unexpected token literal"))
	default:
		tag := Tag(b)
		tagName := d.tags.Name(tag.ID())
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
	b, err := readByte(d.r)
	panicErr(err)

	for {
		switch b {
		case literal:
			var attr Attr
			index, err := mbUint32(d.r)
			panicErr(err)
			name, err := d.GetString(index)
			panicErr(err)
			attr.Name = string(name)
			attr.Value, b = d.readAttrValue()
			elt.Attr = append(elt.Attr, attr)
		case end:
			return
		default:
			if b >= 128 {
				panic(fmt.Errorf("unexpected attribute value"))
			}
			var attr Attr
			attr.Name = d.attrs.Name(b)
			attr.Value, b = d.readAttrValue()
			elt.Attr = append(elt.Attr, attr)
		}
	}
}

func (d *Decoder) readAttrValue() (string, byte) {

	var cdata CharData
	for {
		b, err := readByte(d.r)
		panicErr(err)

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
			cdata = append(cdata, []byte(d.attrs.Name(b))...)
		}
	}
}

func (d *Decoder) content() {
	// content() accumulate adjacent CharData in a unique instance until END or ELEMENT is
	// encountered

	var cdata CharData
	for {
		b, err := readByte(d.r)
		panicErr(err)

		switch b {
		case strI, strT, entity:
			d.charData(&cdata, b)
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
		str, err := readString(d.r)
		panicErr(err)
		*cdata = append(*cdata, str...)
	case strT:
		index, err := mbUint32(d.r)
		panicErr(err)
		str, err := d.GetString(index)
		panicErr(err)
		*cdata = append(*cdata, str...)
	case entity:
		entcode, err := mbUint32(d.r)
		panicErr(err)
		var buf [4]byte
		rlen := utf8.RuneLen(rune(entcode))
		utf8.EncodeRune(buf[:rlen], rune(entcode))
		panicErr(err)
		*cdata = append(*cdata, buf[:rlen]...)
	default:
		panic(fmt.Errorf("Unknown char data tag %d", b))
	}
}
