package wbxml

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncoderEncodeTag(t *testing.T) {
	tests := []struct {
		tag      StartElement
		expected []byte
	}{
		{
			tag:      StartElement{Name: "SyncML"},
			expected: []byte{0x2D},
		},
		{
			tag:      StartElement{Name: "SyncML", Content: true},
			expected: []byte{0x6D},
		},
		{
			tag: StartElement{Name: "SyncML", Content: true,
				Attr: []Attr{Attr{Name: "A"}}},
			expected: []byte{0xED, 05, 01},
		},
		{
			tag:      StartElement{Name: "CS"},
			expected: []byte{0x00, 0x08, 0x05},
		},
	}

	for testID, test := range tests {
		buf := bytes.NewBuffer(nil)
		{
			e := NewEncoder(buf, SyncMLTags, CodeSpace{0: CodePage{5: "A"}})
			err := e.encodeTag(test.tag)
			if err != nil {
				t.Errorf("case %d: unexpected error: %s", testID, err)
				continue
			}
			assert.Equal(t, test.expected, buf.Bytes(), "case %d", testID)
		}
	}
}

func TestEncoderToken(t *testing.T) {
	for testID, tokens := range tokensExamples {
		expected := encodingExamples[testID]
		space := tagSpaceExamples[testID]

		w := bytes.NewBuffer(nil)
		e := NewEncoder(w, space.tags, space.attrs)

		err := e.EncodeHeader(headerExamples[testID])
		if err != nil {
			t.Fatalf("case %d: encoding header: %s", testID, err)
		}

		for _, tok := range tokens {
			if tok == nil {
				break
			}
			err := e.EncodeToken(tok)
			if err != nil {
				t.Errorf("error: token %v: %s", tok, err)
			}
		}

		assert.Equal(t, expected, w.Bytes(), "case %d", testID)
	}
}
