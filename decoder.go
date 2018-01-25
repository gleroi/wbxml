package wbxml

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"unicode/utf8"
)

type Decoder struct {
	r io.Reader

	tagPage  byte
	tags     CodeSpace
	attrPage byte
	attrs    CodeSpace

	offset  int
	tokChan chan Token
	err     error
	Header  Header
}

func NewDecoder(r io.Reader, tags CodeSpace, attrs CodeSpace) *Decoder {
	d := &Decoder{
		r: r,

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

func (d *Decoder) Decode(v interface{}) error {
	return d.DecodeElement(v, nil)
}

func (d *Decoder) DecodeElement(v interface{}, start *StartElement) error {
	val := reflect.ValueOf(v)

	if val.Kind() == reflect.Interface {
		val = val.Elem()
	}
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if start == nil {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		if st, ok := tok.(StartElement); ok {
			start = &st
		} else {
			return fmt.Errorf("expected a StartElement, got %t", tok)
		}
	}

	switch t := val.Type(); val.Kind() {
	case reflect.Struct:
		for {
			tok, err := d.Token()
			if err != nil {
				return err
			}
			if end, ok := tok.(EndElement); ok {
				if end.Name == start.Name {
					return nil
				}
				return fmt.Errorf("expected end element %s, got %s", start.Name, end.Name)
			}
			if st, ok := tok.(StartElement); ok {
				if _, ok := t.FieldByName(st.Name); ok {
					fld := val.FieldByName(st.Name)
					if fld.CanAddr() {
						fld = fld.Addr()
					}
					if fld.CanInterface() {
						err := d.DecodeElement(fld.Interface(), &st)
						if err != nil {
							return err
						}
					} else {
						return fmt.Errorf("tag %s: type %s can't be used as interface{}", st.Name, t.Name())
					}
				} else {
					return fmt.Errorf("struct %s has no field %s", t.Name(), st.Name)
				}
			}
		}
	case reflect.String:
		tok, err := d.Token()
		if err != nil {
			return err
		}
		if cdata, ok := tok.(CharData); ok {
			val.SetString(string(cdata))
			tok, err := d.Token()
			if err != nil {
				return err
			}
			if end, ok := tok.(EndElement); !ok || end.Name != start.Name {
				return fmt.Errorf("expected end element %s, got %s", start.Name, end.Name)
			}
			return nil
		}
		return fmt.Errorf("expected a CharData, got %t", tok)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch itok := tok.(type) {
		case Entity:
			val.SetUint(uint64(itok))
		case CharData:
			i, err := strconv.ParseUint(string(itok), 10, 8)
			if err != nil {
				return fmt.Errorf("field %s: %s", start.Name, err)
			}
			val.SetUint(i)
		default:
			return fmt.Errorf("expected a number, got %T", tok)
		}
		tok, err = d.Token()
		if err != nil {
			return err
		}
		if end, ok := tok.(EndElement); !ok || end.Name != start.Name {
			return fmt.Errorf("expected end element %s, got %+v", start.Name, tok)
		}
		return nil
	default:
		return fmt.Errorf("%s not implemented", t.Name())
	}
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

func (d *Decoder) tagName(code byte) string {
	name, err := d.tags.Name(d.tagPage, code)
	if err != nil {
		d.panicErr(err)
	}
	return name
}

func (d *Decoder) attrName(code byte) string {
	name, err := d.attrs.Name(d.attrPage, code)
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
		if b != gloPi {
			break
		}
		d.piStar()
	}

	d.element(b)

	for {
		b, err = readByte(d)
		d.panicErr(err)
		if b != gloPi {
			break
		}
		d.piStar()
	}
}

func (d *Decoder) piStar() {
}

func (d *Decoder) element(b byte) {
	switch b {
	case gloSwitchPage:
		index, err := readByte(d)
		d.panicErr(err)
		d.tagPage = index
	case gloLiteral, gloLiteralA, gloLiteralC, gloLiteralAC:
		panic(fmt.Errorf("literal tag not implemented"))
	default:
		tag := Tag(b)
		tagName := d.tagName(tag.ID())
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
	b, err := readByte(d)
	d.panicErr(err)

	for {
		switch b {
		case gloSwitchPage:
			index, err := readByte(d)
			d.panicErr(err)
			d.attrPage = index
		case gloLiteral:
			var attr Attr
			index, err := mbUint32(d)
			d.panicErr(err)
			name, err := d.GetString(index)
			d.panicErr(err)
			attr.Name = string(name)
			attr.Value, b = d.readAttrValue()
			elt.Attr = append(elt.Attr, attr)
		case gloEnd:
			return
		default:
			if b >= 128 {
				panic(fmt.Errorf("unexpected attribute value"))
			}
			var attr Attr
			attr.Name = d.attrName(b)
			attr.Value, b = d.readAttrValue()
			elt.Attr = append(elt.Attr, attr)
		}
	}
}

func (d *Decoder) readAttrValue() (string, byte) {
	var cdata CharData
	for {
		b, err := readByte(d)
		d.panicErr(err)

		switch b {
		case gloStrI, gloStrT, gloEntity:
			d.charData(&cdata, b)
		case gloExt0, gloExt1, gloExt2,
			gloExtI0, gloExtI1, gloExtI2,
			gloExtT0, gloExtT1, gloExtT2:
			panic(fmt.Errorf("extension token unimplemented (token %d)", b))
		case gloEnd:
			return string(cdata), b
		default:
			if b < 128 {
				return string(cdata), b
				//panic(fmt.Errorf("unexpected attribute tag name %d", b))
			}
			cdata = append(cdata, []byte(d.attrName(b))...)
		}
	}
}

func (d *Decoder) content() {
	// content() accumulate adjacent CharData in a unique instance until END or ELEMENT is
	// encountered

	var cdata CharData = nil
	for {
		b, err := readByte(d)
		d.panicErr(err)

		switch b {
		case gloStrI, gloStrT, gloEntity:
			d.charData(&cdata, b)
		case gloOpaque:
			d.sendCharData(&cdata)
			length, err := mbUint32(d)
			d.panicErr(err)
			data, err := readSlice(d, length)
			d.panicErr(err)
			d.tokChan <- Opaque(data)
		case gloExt0, gloExt1, gloExt2,
			gloExtI0, gloExtI1, gloExtI2,
			gloExtT0, gloExtT1, gloExtT2:
			panic(fmt.Errorf("extension token unimplemented (token %d)", b))
		case gloEnd:
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
	if cdata == nil {
		*cdata = make([]byte, 0)
	}
	switch b {
	case gloStrI:
		str, err := readString(d)
		d.panicErr(err)
		*cdata = append(*cdata, str...)
	case gloStrT:
		index, err := mbUint32(d)
		d.panicErr(err)
		str, err := d.GetString(index)
		d.panicErr(err)
		*cdata = append(*cdata, str...)
	case gloEntity:
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
