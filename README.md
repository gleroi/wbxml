wbxml
=====

package wbxml `// import "github.com/gleroi/wbxml"` 

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

```golang
func MbUint(r io.Reader, max int) (uint64, error)
func XML(w io.Writer, wb *Decoder, indent string) (finalError error)
type Attr struct{ ... }
type CharData []byte
type CodePage map[byte]string
type CodeSpace map[byte]CodePage
type Decoder struct{ ... }
    func NewDecoder(r io.Reader, tags CodeSpace, attrs CodeSpace) *Decoder
type Encoder struct{ ... }
    func NewEncoder(w io.Writer, tags CodeSpace, attrs CodeSpace) *Encoder
type EndElement struct{ ... }
type Entity uint32
type Header struct{ ... }
type Marshaler interface{ ... }
type Opaque []byte
type ProcInst struct{ ... }
type StartElement struct{ ... }
type Tag byte
type Token interface{}
type Unmarshaler interface{ ... }
```