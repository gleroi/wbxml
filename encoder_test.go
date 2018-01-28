package wbxml

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncoderEncodeTag(t *testing.T) {
	tests := []struct {
		tag      StartElement
		content  bool
		expected []byte
	}{
		{
			tag:      StartElement{Name: "SyncML"},
			content:  false,
			expected: []byte{0x2D},
		},
		{
			tag:      StartElement{Name: "SyncML"},
			content:  true,
			expected: []byte{0x6D},
		},
		{
			tag:      StartElement{Name: "SyncML", Attr: []Attr{Attr{}}},
			content:  true,
			expected: []byte{0xeD},
		},
		{
			tag:      StartElement{Name: "CS"},
			content:  false,
			expected: []byte{0x00, 0x08, 0x05},
		},
	}

	for testID, test := range tests {
		buf := bytes.NewBuffer(nil)
		{
			e := NewEncoder(buf, SyncMLTags, CodeSpace{})
			err := e.EncodeTag(test.tag, test.content)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
				continue
			}
			assert.Equal(t, test.expected, buf.Bytes(), "case %d", testID)
		}
	}
}
