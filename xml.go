package wbxml

import (
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

// XML pretty print WBXML to textual XML
func XML(w io.Writer, wb *Decoder) (finalError error) {
	x := xml.NewEncoder(w)
	x.Indent("", "  ")
	defer func() {
		x.Flush()
	}()

	for {
		tok, err := wb.Token()
		if err != nil {
			return err
		}

		switch t := tok.(type) {
		case StartElement:
			x.EncodeToken(xml.StartElement{
				Name: xml.Name{Local: t.Name},
				Attr: mapAttrToXml(t.Attr),
			})
		case CharData:
			x.EncodeToken(xml.CharData(t))
		case Opaque:
			x.EncodeToken(xml.CharData(hex.EncodeToString(t)))
		case Entity:
			x.EncodeToken(xml.CharData(strconv.FormatInt(int64(t), 10)))
		case EndElement:
			x.EncodeToken(xml.EndElement{Name: xml.Name{Local: t.Name}})
		default:
			return fmt.Errorf("unknown token %T:\n  %+v", t, t)
		}
	}
}

func mapAttrToXml(attrs []Attr) []xml.Attr {
	x := make([]xml.Attr, len(attrs))
	for i, attr := range attrs {
		x[i] = xml.Attr{
			Name:  xml.Name{Local: attr.Name},
			Value: attr.Value,
		}
	}
	return x
}
