package wbxml

import (
	"fmt"
	"io"
)

type Encoder struct {
	w io.Writer

	tagPage  byte
	tags     CodeSpace
	attrPage byte
	attrs    CodeSpace

	offset  int
	tokChan chan Token
	err     error
	Header  Header
}

func NewEncoder(w io.Writer, tags CodeSpace, attrs CodeSpace) *Encoder {
	e := &Encoder{
		w:       w,
		tags:    tags,
		attrs:   attrs,
		tokChan: make(chan Token),
	}

	return e
}

func (d *Encoder) EncodeToken(tok Token) error {
	switch tok := tok.(type) {
	case StartElement:
		panic(fmt.Errorf("use EncodeTag instead"))
	case EndElement:
	case ProcInst:
	case CharData:
	case Opaque:
	case Entity:
	default:
		return fmt.Errorf("unknown token %T", tok)
	}
	return nil
}

// tag return the tag code, page or and error.
// tag is -1 if no switch page is needed
func (d *Encoder) tag(tag string) (byte, int, error) {
	for page, p := range d.tags {
		for code, name := range p {
			if name == tag {
				if page == d.tagPage {
					return code, -1, nil
				}
				return code, int(page), nil
			}
		}
	}
	return 0, 0, fmt.Errorf("unknown tag %s", tag)
}

func (d *Encoder) EncodeTag(tok StartElement, hasContent bool) error {
	code, page, err := d.tag(tok.Name)
	if err != nil {
		return err
	}
	if page != -1 {
		err := d.switchPage(byte(page))
		if err != nil {
			return err
		}
	}
	finalCode := code
	if len(tok.Attr) != 0 {
		finalCode |= tagAttrMask
	}
	if hasContent {
		finalCode |= tagContentMask
	}
	return writeByte(d, finalCode)
}

func (d *Encoder) switchPage(p byte) error {
	err := writeByte(d, gloSwitchPage)
	if err != nil {
		return err
	}
	return writeByte(d, p)
}
