package wbxml

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

var encodingExamples = [][]byte{
	[]byte{
		0x01, 0x01, 0x03, 0x00, 0x47, 0x46, 0x03, ' ', 'X', ' ', '&', ' ', 'Y', 0x00, 0x05, 0x03, ' ', 'X', 0x00, 0x02, 0x81, 0x20, 0x03, '=', 0x00, 0x02, 0x81, 0x20, 0x03, '1', ' ', 0x00, 0x01, 0x01},
	[]byte{
		0x01, 0x01, 0x6A, 0x12, 'a', 'b', 'c', 0x00, ' ', 'E', 'n', 't', 'e', 'r', ' ', 'n',
		'a', 'm', 'e', ':', ' ', 0x00, 0x47, 0xC5, 0x09, 0x83, 0x00, 0x05, 0x01, 0x88, 0x06,
		0x86, 0x08, 0x03, 'x', 'y', 'z', 0x00, 0x85, 0x03, '/', 's', 0x00, 0x01, 0x83, 0x04,
		0x86, 0x07, 0x0A, 0x03, 'N', 0x00, 0x01, 0x01, 0x01,
	},
}

var tagSpaceExamples = []struct {
	tags  CodeSpace
	attrs CodeSpace
}{
	{
		tags: CodeSpace{
			pages: map[byte]CodePage{
				0: CodePage{
					5: "BR",
					6: "CARD",
					7: "XYZ",
				},
			},
		},
	},
	{
		tags: CodeSpace{
			pages: map[byte]CodePage{
				0: CodePage{
					5: "CARD",
					6: "INPUT",
					7: "XYZ",
					8: "DO",
				},
			},
		},
		attrs: CodeSpace{
			pages: map[byte]CodePage{
				0: CodePage{
					0x05: "STYLE",
					0x06: "TYPE",
					0x07: "TYPE",
					0x08: "URL",
					0x09: "NAME",
					0x0A: "KEY",
					0x85: ".org",
					0x86: "ACCEPT",
				},
			},
		},
	},
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
		[]Token{
			StartElement{Name: "XYZ"},
			StartElement{
				Name: "CARD",
				Attr: []Attr{
					Attr{"NAME", "abc"},
					Attr{"STYLE", ""},
				}},
			StartElement{
				Name: "DO",
				Attr: []Attr{
					Attr{"TYPE", "ACCEPT"},
					Attr{"URL", "xyz.org/s"},
				},
			},
			EndElement{Name: "DO"},
			CharData(" Enter name: "),
			StartElement{
				Name: "INPUT",
				Attr: []Attr{
					Attr{"TYPE", ""},
					Attr{"KEY", "N"},
				},
			},
			EndElement{Name: "INPUT"},
			EndElement{Name: "CARD"},
			EndElement{Name: "XYZ"},
			nil,
		},
	}

	for testID, expected := range tests {
		input := encodingExamples[testID]
		space := tagSpaceExamples[testID]

		r := bytes.NewReader(input)
		d := NewDecoder(r, space.tags, space.attrs)

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
