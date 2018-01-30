/*
Package wbxml implements a simple WBXML parser, based on encoding/xml API.

Specifications of the standard are available at https://www.w3.org/TR/wbxml.

This package supports decoding most WBXML construct, except:
  - Process Instruction (PI)
  - Literal tag and attribute (LITERAL)
  - Extension are not supported (EXT*)

When decoding, some restrictions apply:
  - Attribute values decode to string or []byte only
  - Entity, string and  are aggregated to one CharData if they are consecutive

When encoding a struct, some restrictions apply:
  - Fields cannot be mapped to attributes
  - slice other than []byte are not supported

Golang attributes are not supported for now.

WBXML grammar is:
  start		= version publicid charset strtbl body
  strtbl		= length *byte
  body		= *pi element *pi
  element	= stag [ 1*attribute END ] [ *content END ]

  content	= element | string | extension | entity | pi | opaque

  stag		= TAG | ( LITERAL index )
  attribute	= attrStart *attrValue
  attrStart	= ATTRSTART | ( LITERAL index )
  attrValue	= ATTRVALUE | string | extension | entity

  extension	= ( EXT_I termstr ) | ( EXT_T index ) | EXT

  string		= inline | tableref
  inline		= STR_I termstr
  tableref	= STR_T index

  entity		= ENTITY entcode
  entcode	= mb_u_int32			// UCS-4 character code

  pi			= PI attrStart *attrValue END

  opaque		= OPAQUE length *byte

  version	= u_int8 containing WBXML version number
  publicid	= mb_u_int32 | ( zero index )
  charset	= mb_u_int32
  termstr	= charset-dependent string with termination
  index		= mb_u_int32			// integer index into string table.
  length		= mb_u_int32			// integer length.
  zero		= u_int8			// containing the value zero (0).
*/
package wbxml

import (
	"fmt"
	"io"
	"unicode/utf8"
)

// CodeSpace represents the mapping of a tag or attribute to its code, organized in pages
// of overlapping code to tag mapping.
type CodeSpace map[byte]CodePage

// Name return the name of tag encoded by (pageID, code).
func (space CodeSpace) Name(pageID byte, code byte) (string, error) {
	page, ok := space[pageID]
	if !ok {
		return "", fmt.Errorf("Unknown page %d", pageID)
	}
	name, ok := page[code]
	if !ok {
		return "", fmt.Errorf("Unknown code %d in page %d", code, pageID)
	}
	return name, nil
}

// CodePage represents a mapping between code and tag/attribute.
type CodePage map[byte]string

// Token is an interface holding one of the token types:
// StartElement, EndElement, CharData, Entity, Opaque, ProcInst.
type Token interface{}

// StartElement represent the start tag of an WBXML element.
type StartElement struct {
	Name    string
	Attr    []Attr
	Content bool
	Offset  int
}

// Attr represents an attribute of WBXML element.
type Attr struct {
	Name  string
	Value string
}

// EndElement represents the end tag of an WBXML element.
type EndElement struct {
	Name   string
	Offset int
}

// ProcInst represents a processor instruction (PI) in a WBXML document.
type ProcInst struct {
	Target string
	Inst   []byte
}

// CharData represents multiple of adjacent string (inline or tableref) and entity.
type CharData []byte

// Opaque represents an Opaque string of data.
type Opaque []byte

// Entity represents a WBXML entity, used only when alone, else it is concatenated to the previous
// CharData.
type Entity uint32

// UTF8 converts an entity to a valid utf sequence.
func (ent Entity) UTF8() []byte {
	var buf [4]byte
	rlen := utf8.RuneLen(rune(ent))
	utf8.EncodeRune(buf[:rlen], rune(ent))
	return buf[:rlen]
}

// Header represents the header of a wbxml document.
type Header struct {
	Version     uint8
	PublicID    uint32
	Charset     uint32
	StringTable []byte
}

const (
	gloSwitchPage = 0x0  // 	Change the code page for the current token state. Followed by a single u_int8 indicating the new code page number.
	gloEnd        = 0x1  // 	Indicates the end of an attribute list or the end of an element.
	gloEntity     = 0x2  // 	A character entity. Followed by a mb_u_int32 encoding the character entity number.
	gloStrI       = 0x3  // 	Inline string. Followed by a termstr.
	gloLiteral    = 0x4  // 	An unknown tag or attribute name. Followed by an mb_u_int32 that encodes an offset into the string table.
	gloExtI0      = 0x40 // 	Inline string document-type-specific extension token. Token is followed by a termstr.
	gloExtI1      = 0x41 // 	Inline string document-type-specific extension token. Token is followed by a termstr.
	gloExtI2      = 0x42 // 	Inline string document-type-specific extension token. Token is followed by a termstr.
	gloPi         = 0x43 // 	Processing instruction.
	gloLiteralC   = 0x44 // 	Unknown tag, with content.
	gloExtT0      = 0x80 // 	Inline integer document-type-specific extension token. Token is followed by a mb_uint_32.
	gloExtT1      = 0x81 // 	Inline integer document-type-specific extension token. Token is followed by a mb_uint_32.
	gloExtT2      = 0x82 // 	Inline integer document-type-specific extension token. Token is followed by a mb_uint_32.
	gloStrT       = 0x83 // 	String table reference. Followed by a mb_u_int32 encoding a byte offset from the beginning of the string table.
	gloLiteralA   = 0x84 // 	Unknown tag, with attributes.
	gloExt0       = 0xC0 // 	Single-byte document-type-specific extension token.
	gloExt1       = 0xC1 // 	Single-byte document-type-specific extension token.
	gloExt2       = 0xC2 // 	Single-byte document-type-specific extension token.
	gloOpaque     = 0xC3 // 	Opaque document-type-specific data.
	gloLiteralAC  = 0xC4 // 	Unknown tag, with content and attributes.
)

func (d *Decoder) panicErr(err error) {
	if err != nil {
		if err == io.EOF {
			panic(err)
		}
		panic(fmt.Errorf("position %d: %s", d.offset, err))
	}
}

// Tag represents a non global tag in a WBXML document.
type Tag byte

const tagAttrMask = 0x80
const tagContentMask = 0x40

// Attr returns if a Tag has some attribute following it.
func (t Tag) Attr() bool {
	return t&tagAttrMask == tagAttrMask
}

// Content return if a Tag has some content followinf it.
func (t Tag) Content() bool {
	return t&tagContentMask == tagContentMask
}

// ID return the code identifying a Tag in its code space.
func (t Tag) ID() byte {
	return byte(t & 0x03F)
}
