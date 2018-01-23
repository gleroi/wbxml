package wbxml

import (
	"encoding/xml"
	"fmt"
	"io"
)

// XML pretty print WBXML to textual XML
func XML(w io.Writer, wb *Decoder) error {
	x := xml.NewEncoder(w)

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
		case EndElement:
			x.EncodeToken(xml.EndElement{Name: xml.Name{Local: t.Name}})
		default:
			return fmt.Errorf("unknown token %T:\n  %+v", t, t)
		}
	}
	return nil
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
