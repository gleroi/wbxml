// Package wbxml implements a simple WBXML parser, based on encoding/xml API.
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
package wbxml

import (
	"fmt"
	"io"
)

// CodeSpace represents the mapping of a tag or attribute to its code.
type CodeSpace struct {
	pages map[byte]CodePage
}

func (space CodeSpace) Name(pageId byte, id byte) (string, error) {
	page, ok := space.pages[pageId]
	if !ok {
		return "", fmt.Errorf("Unknown page %d", pageId)
	}
	name, ok := page[id]
	if !ok {
		return "", fmt.Errorf("Unknown code %d in page %d", id, pageId)
	}
	return name, nil
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

type Opaque []byte

type Entity uint32

// Header represents the header of a wbxml document.
type Header struct {
	Version     uint8
	PublicID    uint32
	Charset     uint32
	StringTable []byte
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

func (d *Decoder) panicErr(err error) {
	if err != nil {
		if err == io.EOF {
			panic(err)
		}
		panic(fmt.Errorf("position %d: %s", d.offset, err))
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
