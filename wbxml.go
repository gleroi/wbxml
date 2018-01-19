// Package wbxml implements a simple WBXML parser, based on encoding/xml API.
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

func NewDecoder(r io.Reader, tags CodeSpace) *Decoder {
	d := &Decoder{
		r:       r,
		tags:    tags,
		tokChan: make(chan Token),
	}

	go d.run()
	return d
}

// CodeSpace represents the mapping of a tag or attribute to its code.
type CodeSpace struct {
	pages map[byte]CodePage
	page  byte
}

func (space CodeSpace) Name(id byte) string {
	page, ok := space.pages[space.page]
	if !ok {
		panic(fmt.Errorf("Unknown page %d", space.page))
	}
	name, ok := page[id]
	if !ok {
		panic(fmt.Errorf("Unknown code %d in page %d", id, space.page))
	}
	return name
}

// CodePage represents a mapping between code and tag/attribute.
type CodePage map[byte]string

// Token is an interface holding one of the token types:
// StartElement, EndElement, CharData, Comment, ProcInst, or Directive.
type Token interface{}

type StartElement struct {
	Name string
	Attr []Attr
}

type Attr struct {
	Name  string
	Value string
}

type EndElement struct {
	Name string
}

type ProcInst struct {
	Target string
	Inst   []byte
}

type CharData []byte

// Header represents the header of a wbxml document.
type Header struct {
	Version     uint8
	PublicID    uint32
	Charset     uint32
	StringTable [][]byte
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

const (
	switchPage = 0x0  // 	Change the code page for the current token state. Followed by a single u_int8 indicating the new code page number.
	end        = 0x1  // 	Indicates the end of an attribute list or the end of an element.
	entity     = 0x2  // 	A character entity. Followed by a mb_u_int32 encoding the character entity number.
	strI       = 0x3  // 	Inline string. Followed by a termstr.
	literal    = 0x4  // 	An unknown tag or attribute name. Followed by an mb_u_int32 that encodes an offset into the string table.
	extI0      = 0x40 // 	Inline string document-type-specific extension token. Token is followed by a termstr.
	extI1      = 0x41 // 	Inline string document-type-specific extension token. Token is followed by a termstr.
	extI2      = 0x42 // 	Inline string document-type-specific extension token. Token is followed by a termstr.
	pi         = 0x43 // 	Processing instruction.
	literalC   = 0x44 // 	Unknown tag, with content.
	extT0      = 0x80 // 	Inline integer document-type-specific extension token. Token is followed by a mb_uint_32.
	extT1      = 0x81 // 	Inline integer document-type-specific extension token. Token is followed by a mb_uint_32.
	extT2      = 0x82 // 	Inline integer document-type-specific extension token. Token is followed by a mb_uint_32.
	strT       = 0x83 // 	String table reference. Followed by a mb_u_int32 encoding a byte offset from the beginning of the string table.
	literalA   = 0x84 // 	Unknown tag, with attributes.
	ext0       = 0xC0 // 	Single-byte document-type-specific extension token.
	ext1       = 0xC1 // 	Single-byte document-type-specific extension token.
	ext2       = 0xC2 // 	Single-byte document-type-specific extension token.
	opaque     = 0xC3 // 	Opaque document-type-specific data.
	literalAc  = 0xC4 // 	Unknown tag, with content and attributes.
)

// ReadHeader reads the wbxml header.
func ReadHeader(r io.Reader) (Header, error) {
	var h Header
	var err error

	h.Version, err = readByte(r)
	if err != nil {
		return h, err
	}

	h.PublicID, err = mbUint32(r)
	if err != nil {
		return h, err
	}
	if h.PublicID == 0 {
		h.PublicID, err = mbUint32(r)
	}

	h.Charset, err = mbUint32(r)
	if err != nil {
		return h, err
	}

	length, err := mbUint32(r)
	if err != nil {
		return h, err
	}
	buf := make([]byte, length)
	_, err = r.Read(buf)
	if err != nil {
		return h, err
	}

	h.StringTable = make([][]byte, 0, 8)
	start := 0
	for end, b := range buf {
		if b == 0 {
			h.StringTable = append(h.StringTable, buf[start:end])
			start = end + 1
		}
	}
	return h, nil
}

func (d *Decoder) GetString(i uint32) ([]byte, error) {
	if i >= uint32(len(d.Header.StringTable)) {
		return nil, fmt.Errorf("%d is not a valid string reference (max %d)", i, len(d.Header.StringTable))
	}
	return d.Header.StringTable[i], nil
}

func (d *Decoder) run() {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				d.err = err
			}
			close(d.tokChan)
		}
	}()

	d.body()
	close(d.tokChan)
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
		*cdata = make([]byte, 0)
	}
}

func (d *Decoder) charData(cdata *CharData, b byte) {
	switch b {
	case strI:
		str, err := readString(d.r)
		panicErr(err)
		*cdata = append(*cdata, str...)
	case strT:
		index, err := mbUint32(d.r)
		panicErr(err)
		str, err := d.GetString(index)
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

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

type Tag byte

func (t Tag) Attr() bool {
	return t&0x80 == 0x80
}

func (t Tag) Content() bool {
	return t&0x40 == 0x40
}

func (t Tag) ID() byte {
	return byte(t & 0x03F)
}
