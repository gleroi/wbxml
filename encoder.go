package wbxml

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
)

type Marshaler interface {
	MarshalWBXML(e *Encoder, st StartElement) error
}

type Encoder struct {
	w io.Writer

	tagPage  byte
	tags     CodeSpace
	attrPage byte
	attrs    CodeSpace

	offset    int
	tokChan   chan Token
	ignoreEnd []string
	err       error
	Header    Header
}

func NewEncoder(w io.Writer, tags CodeSpace, attrs CodeSpace) *Encoder {
	e := &Encoder{
		w:         w,
		tags:      tags,
		attrs:     attrs,
		tokChan:   make(chan Token),
		ignoreEnd: make([]string, 0, 8),
	}

	return e
}

func (e *Encoder) GetIndex(str []byte) (uint32, bool) {
	start := 0
	for end, b := range e.Header.StringTable {
		if b == 0 {
			if bytes.Equal(str, e.Header.StringTable[start:end]) {
				return uint32(start), true
			}
			start = end + 1
		}
	}
	return 0, false
}

func (e *Encoder) EncodeHeader(h Header) error {
	e.Header = h

	err := writeByte(e, h.Version)
	if err != nil {
		return err
	}

	err = writeMbUint32(e, h.PublicID)
	if err != nil {
		return err
	}
	if h.PublicID == 0 {
		err = writeMbUint32(e, h.PublicID)
		if err != nil {
			return err
		}
	}

	err = writeMbUint32(e, h.Charset)
	if err != nil {
		return err
	}

	err = writeMbUint32(e, uint32(len(h.StringTable)))
	if err != nil {
		return err
	}

	return writeSlice(e, h.StringTable)
}

func (e *Encoder) EncodeElement(v interface{}, start StartElement) error {
	val := reflect.ValueOf(v)

	if v == nil {
		return nil
	}

	if val.Kind() == reflect.Interface && !val.IsNil() {
		if marsh, ok := val.Interface().(Marshaler); ok {
			return marsh.MarshalWBXML(e, start)
		}
		val = val.Elem()
	}
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		if marsh, ok := val.Interface().(Marshaler); ok {
			return marsh.MarshalWBXML(e, start)
		}
		val = val.Elem()
	}
	if !val.IsValid() {
		return nil
	}
	if marsh, ok := val.Interface().(Marshaler); ok {
		return marsh.MarshalWBXML(e, start)
	}

	return e.marshalValue(val, start)
}

func (e *Encoder) marshalValue(val reflect.Value, start StartElement) error {
	kind := val.Kind()
	typ := val.Type()

	switch kind {
	case reflect.Struct:
		start.Content = false
		for i := 0; i < val.NumField(); i++ {
			fld := val.Field(i)
			if fld.IsValid() {
				start.Content = true
				break
			}
		}
		err := e.EncodeToken(start)
		if err != nil {
			return err
		}
		for i := 0; i < val.NumField() && start.Content; i++ {
			fld := val.Field(i)
			if fld.IsValid() {
				err := e.EncodeElement(fld.Interface(), StartElement{Name: typ.Field(i).Name})
				if err != nil {
					return fmt.Errorf("%s.%s: %s", typ.Name(), typ.Field(i).Name, err)
				}
			}
		}
		return e.EncodeToken(EndElement{Name: start.Name})
	case reflect.Slice:
		start.Content = val.Len() > 0
		err := e.EncodeToken(start)
		if err != nil {
			return err
		}
		if start.Content {
			if typ.Elem().Kind() == reflect.Uint8 {
				err := e.EncodeToken(Opaque(val.Bytes()))
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("SLICE!")
			}
		}
		return e.EncodeToken(EndElement{Name: start.Name})
	case reflect.String:
		start.Content = val.Len() > 0
		err := e.EncodeToken(start)
		if err != nil {
			return err
		}
		if start.Content {
			err := e.EncodeToken(CharData(val.String()))
			if err != nil {
				return err
			}
		}
		return e.EncodeToken(EndElement{Name: start.Name})
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		start.Content = true
		err := e.EncodeToken(start)
		if err != nil {
			return err
		}
		err = e.EncodeToken(Entity(val.Uint()))
		if err != nil {
			return err
		}
		return e.EncodeToken(EndElement{Name: start.Name})
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		start.Content = true
		err := e.EncodeToken(start)
		if err != nil {
			return err
		}
		err = e.EncodeToken(Entity(val.Int()))
		if err != nil {
			return err
		}
		return e.EncodeToken(EndElement{Name: start.Name})
	case reflect.Bool:
		if val.Bool() {
			start.Content = true
			err := e.EncodeToken(start)
			if err != nil {
				return err
			}
			err = e.EncodeToken(EndElement{Name: start.Name})
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%s (%s) not supported", val.Kind(), typ.Name())
	}
	return nil
}

func (e *Encoder) EncodeToken(tok Token) error {
	switch tok := tok.(type) {
	case StartElement:
		return e.encodeTag(tok)
	case EndElement:
		return e.encodeEnd(tok)
	case ProcInst:
		return fmt.Errorf("not implemented")
	case CharData:
		return e.writeString(tok)
	case Opaque:
		return writeOpaque(e, tok)
	case Entity:
		return e.writeEntity(tok)
	default:
		return fmt.Errorf("unknown token %T", tok)
	}
}

// tag return the tag code, page or and error.
// tag is -1 if no switch page is needed
func (e *Encoder) tag(tag string) (byte, byte, error) {
	return findCodePage(e.tags, tag)
}

func (e *Encoder) attribute(tag string) (byte, byte, error) {
	return findCodePage(e.attrs, tag)
}

// findCodePage return the a code, page or and error.
// page is -1 if no switch page is needed
func findCodePage(space CodeSpace, tag string) (byte, byte, error) {
	for page, p := range space {
		for code, name := range p {
			if name == tag {
				return code, page, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("unknown tag %s", tag)
}

func (e *Encoder) encodeTag(tok StartElement) error {
	code, page, err := e.tag(tok.Name)
	if err != nil {
		return err
	}
	err = e.switchTagPage(page)
	if err != nil {
		return err
	}
	finalCode := code
	if len(tok.Attr) != 0 {
		finalCode |= tagAttrMask
	}
	if tok.Content {
		finalCode |= tagContentMask
	} else {
		// no content, remember to not write end for this tag
		e.ignoreEnd = append(e.ignoreEnd, tok.Name)
	}
	err = writeByte(e, finalCode)
	if err != nil {
		return err
	}

	return e.encodeAttrs(tok.Attr)
}

func (e *Encoder) encodeAttrs(attrs []Attr) error {
	if len(attrs) == 0 {
		return nil
	}
	for _, attr := range attrs {
		code, page, err := e.attribute(attr.Name)
		if err != nil {
			return err
		}

		err = e.switchTagPage(page)
		if err != nil {
			return err
		}

		err = writeByte(e, code)
		if err != nil {
			return err
		}

		code, page, err = e.attribute(attr.Value)
		if err == nil {
			err := e.switchTagPage(page)
			if err != nil {
				return err
			}
			err = writeByte(e, code)
			if err != nil {
				return err
			}
		} else {
			e.writeString([]byte(attr.Value))
		}
	}
	return writeByte(e, gloEnd)
}

func (e *Encoder) encodeEnd(tok EndElement) error {
	ilen := len(e.ignoreEnd)
	if ilen > 0 && tok.Name == e.ignoreEnd[ilen-1] {
		e.ignoreEnd = e.ignoreEnd[:ilen-1]
		return nil
	}
	_, page, err := e.tag(tok.Name)
	if err != nil {
		return err
	}
	err = writeByte(e, gloEnd)
	if err != nil {
		return err
	}
	return e.switchTagPage(page)
}

func (e *Encoder) switchTagPage(p byte) error {
	if p == e.tagPage {
		return nil
	}
	e.tagPage = byte(p)
	err := writeByte(e, gloSwitchPage)
	if err != nil {
		return err
	}
	return writeByte(e, byte(p))
}

func (e *Encoder) switchAttrPage(p byte) error {
	if p == e.attrPage {
		return nil
	}
	e.attrPage = byte(p)
	err := writeByte(e, gloSwitchPage)
	if err != nil {
		return err
	}
	return writeByte(e, byte(p))
}

func (e *Encoder) writeString(cdata CharData) error {
	if index, ok := e.GetIndex(cdata); ok {
		err := writeByte(e, gloStrT)
		if err != nil {
			return err
		}
		err = writeMbUint32(e, index)
		if err != nil {
			return err
		}
		return nil
	}

	if len(cdata) == 0 {
		return nil
	}
	err := writeByte(e, gloStrI)
	if err != nil {
		return err
	}
	return writeString(e, cdata)
}

func (e *Encoder) writeEntity(tok Entity) error {
	err := writeByte(e, gloEntity)
	if err != nil {
		return err
	}
	return writeMbUint32(e, uint32(tok))
}
