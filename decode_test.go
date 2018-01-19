package wbxml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var encodingExamples = [][]byte{
	[]byte{
		0x01, 0x01, 0x03, 0x00, 0x47, 0x46, 0x03, ' ', 'X', ' ', '&', ' ', 'Y', 0x00, 0x05, 0x03, ' ', 'X', 0x00, 0x02, 0x81, 0x20, 0x03, '=', 0x00, 0x02, 0x81, 0x20, 0x03, '1', ' ', 0x00, 0x01, 0x01},
	[]byte{
		0x01, 0x01, 0x6A, 0x12, 'a', 'b', 'c', 0x00, ' ', 'E', 'n', 't', 'e', 'r', ' ', 'n',
		'a', 'm', 'e', ':', ' ', 0x00, 0x47, 0xC5, 0x09, 0x03, 0x00, 0x05, 0x01, 0x88, 0x06,
		0x86, 0x08, 0x03, 'x', 'y', 'z', 0x00, 0x85, 0x03, '/', 's', 0x00, 0x01, 0x83, 0x04,
		0x01, 0x83, 0x04, 0x86, 0x07, 0x0A, 0x03, 'N', 0x00, 0x01, 0x01, 0x01,
	},
}

var tagSpaceExamples = []CodeSpace{
	CodeSpace{
		pages: map[byte]CodePage{
			0: CodePage{
				5: "BR",
				6: "CARD",
				7: "XYZ",
			},
		},
	},
	CodeSpace{
		pages: map[byte]CodePage{
			0: CodePage{
				5: "CARD",
				6: "INPUT",
				7: "XYZ",
				8: "DO",
			},
		},
	},
}

var xmlExamples []string = []string{
	`<XYZ>
<CARD>
X &amp; Y<BR/>
X&nbsp;=&nbsp;1
</CARD>
</XYZ>`,
}

func TestReadHeader(t *testing.T) {
	tests := []Header{
		Header{
			Version:     1,
			PublicID:    1,
			Charset:     3,
			StringTable: [][]byte{},
		},
		Header{
			Version:  1,
			PublicID: 1,
			Charset:  106,
			StringTable: [][]byte{
				{'a', 'b', 'c'},
				{' ', 'E', 'n', 't', 'e', 'r', ' ', 'n', 'a', 'm', 'e', ':', ' '},
			},
		},
	}

	for testID, expected := range tests {
		input := encodingExamples[testID]
		result, err := ReadHeader(bytes.NewReader(input))
		if err != nil {
			t.Errorf("case %d: unexpected error: %s", testID, err)
			continue
		}

		assert.Equal(t, expected, result)
	}
}

func TestXmlString(t *testing.T) {
	t.SkipNow()

	for testID, input := range xmlExamples {
		d := xml.NewDecoder(strings.NewReader(input))
		d.Entity = xml.HTMLEntity

		fmt.Fprintf(os.Stdout, "DEB %d\n", testID)

		for {
			tok, err := d.Token()

			if err != nil {
				if err != io.EOF {
					t.Errorf("case %d: unexpected error: %s", testID, err)
				}
				break
			}
			fmt.Fprintf(os.Stdout, "%T : %+v\n", tok, tok)
			if cdata, ok := tok.(xml.CharData); ok {
				fmt.Fprintf(os.Stdout, "%s\n", string(cdata))
			}
		}
		fmt.Fprintf(os.Stdout, "FIN %d\n", testID)
	}
}

func TestDecoderToken(t *testing.T) {
	tests := [][]Token{
		[]Token{
			StartElement{Name: "XYZ"},
			StartElement{Name: "CARD"},
			CharData(" X & Y"),
			StartElement{Name: "BR"},
			EndElement{Name: "BR"},
			CharData(" X\u00A0=\u00A01 "),
			EndElement{Name: "CARD"},
			EndElement{Name: "XYZ"},
			nil,
		},
	}

	for testID, expected := range tests {
		input := encodingExamples[testID]
		tagspace := tagSpaceExamples[testID]

		r := bytes.NewReader(input)
		ReadHeader(r)
		d := NewDecoder(r, tagspace)

		result := make([]Token, 0, len(expected))
		var err error
		var tok Token
		for range expected {
			tok, err = d.Token()
			if err != nil {
				if err != io.EOF {
					t.Errorf("case %d: unexpected error: %s", testID, err)
					break
				}
			}
			result = append(result, tok)
		}
		if err != io.EOF {
			t.Errorf("case %d: EOF not meet", testID)
		}
		assert.Equal(t, expected, result)
	}
}
