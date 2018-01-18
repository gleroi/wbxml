// Package wbxml implements a simple WBXML parser, based on encoding/xml API.
package wbxml

import (
	"fmt"
	"io"
)

type Decoder struct {
	r io.ByteReader

	Header Header
}

// Token is an interface holding one of the token types:
// StartElement, EndElement, CharData, Comment, ProcInst, or Directive.
type Token interface{}

// Header represents the header of a wbxml document.
type Header struct {
	Version     uint8
	PublicID    uint32
	Charset     uint32
	StringTable []byte
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
	return nil, fmt.Errorf("not implemented")
}

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
	h.StringTable = make([]byte, length)
	_, err = r.Read(h.StringTable)
	if err != nil {
		return h, err
	}
	return h, nil
}
