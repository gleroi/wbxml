package wbxml

import (
	"bytes"
	"fmt"
	"io"
)

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

func (d *Encoder) GetIndex(str []byte) (uint32, bool) {
	start := 0
	for end, b := range d.Header.StringTable {
		if b == 0 {
			if bytes.Equal(str, d.Header.StringTable[start:end]) {
				return uint32(start), true
			}
			start = end + 1
		}
	}
	return 0, false
}

func (d *Encoder) EncodeHeader(h Header) error {
	d.Header = h

	err := writeByte(d, h.Version)
	if err != nil {
		return err
	}

	err = writeMbUint32(d, h.PublicID)
	if err != nil {
		return err
	}
	if h.PublicID == 0 {
		err = writeMbUint32(d, h.PublicID)
		if err != nil {
			return err
		}
	}

	err = writeMbUint32(d, h.Charset)
	if err != nil {
		return err
	}

	return writeOpaque(d, h.StringTable)
}

func (d *Encoder) EncodeToken(tok Token) error {
	switch tok := tok.(type) {
	case StartElement:
		return d.encodeTag(tok)
	case EndElement:
		ilen := len(d.ignoreEnd)
		if ilen > 0 && tok.Name == d.ignoreEnd[ilen-1] {
			d.ignoreEnd = d.ignoreEnd[:ilen-1]
			return nil
		}
		return writeByte(d, gloEnd)
	case ProcInst:
		return fmt.Errorf("not implemented")
	case CharData:
		return d.writeString(tok)
	case Opaque:
		return writeOpaque(d, tok)
	case Entity:
		return writeMbUint32(d, uint32(tok))
	default:
		return fmt.Errorf("unknown token %T", tok)
	}
}

// tag return the tag code, page or and error.
// tag is -1 if no switch page is needed
func (d *Encoder) tag(tag string) (byte, byte, error) {
	return findCodePage(d.tags, tag)
}

func (d *Encoder) attribute(tag string) (byte, byte, error) {
	return findCodePage(d.attrs, tag)
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

func (d *Encoder) encodeTag(tok StartElement) error {
	code, page, err := d.tag(tok.Name)
	if err != nil {
		return err
	}
	err = d.switchTagPage(page)
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
		d.ignoreEnd = append(d.ignoreEnd, tok.Name)
	}
	err = writeByte(d, finalCode)
	if err != nil {
		return err
	}

	return d.encodeAttrs(tok.Attr)
}

func (d *Encoder) encodeAttrs(attrs []Attr) error {
	if len(attrs) == 0 {
		return nil
	}
	for _, attr := range attrs {
		code, page, err := d.attribute(attr.Name)
		if err != nil {
			return err
		}

		err = d.switchTagPage(page)
		if err != nil {
			return err
		}

		err = writeByte(d, code)
		if err != nil {
			return err
		}

		code, page, err = d.attribute(attr.Value)
		if err == nil {
			err := d.switchTagPage(page)
			if err != nil {
				return err
			}
			err = writeByte(d, code)
			if err != nil {
				return err
			}
		} else {
			d.writeString([]byte(attr.Value))
		}
	}
	return writeByte(d, gloEnd)
}

func (d *Encoder) switchTagPage(p byte) error {
	if p == d.tagPage {
		return nil
	}
	d.tagPage = byte(p)
	err := writeByte(d, gloSwitchPage)
	if err != nil {
		return err
	}
	return writeByte(d, byte(p))
}

func (d *Encoder) switchAttrPage(p byte) error {
	if p == d.attrPage {
		return nil
	}
	d.attrPage = byte(p)
	err := writeByte(d, gloSwitchPage)
	if err != nil {
		return err
	}
	return writeByte(d, byte(p))
}

func (d *Encoder) writeString(cdata CharData) error {
	if index, ok := d.GetIndex(cdata); ok {
		err := writeByte(d, gloStrT)
		if err != nil {
			return err
		}
		err = writeMbUint32(d, index)
		if err != nil {
			return err
		}
		return nil
	}

	if len(cdata) == 0 {
		return nil
	}
	err := writeByte(d, gloStrI)
	if err != nil {
		return err
	}
	return writeString(d, cdata)
}
