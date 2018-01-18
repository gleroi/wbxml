package wbxml

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeMultibyteInteger(t *testing.T) {
	tests := []struct {
		input    []byte
		expected uint64
	}{
		{[]byte{0x81, 0x20}, 0xA0},
		{[]byte{0x60}, 0x60},
	}

	for testID, test := range tests {
		result, err := mbUint(bytes.NewReader(test.input), 8)

		if err != nil {
			t.Errorf("case %d: unexpected error: %s", testID, err)
			continue
		}

		if result != test.expected {
			t.Errorf("case %d: expected %d, got %d", testID, test.expected, result)
		}
	}
}

var encodingExamples = [][]byte{
	[]byte{
		0x01, 0x01, 0x03, 0x00, 0x47, 0x46, 0x03, ' ', 'X', ' ', ' ', '&', ' ', 'Y', 0x00, 0x05, 0x03, ' ', 'X', 0x00, 0x02, 0x81, 0x20, 0x03, '=', 0x00, 0x02, 0x81, 0x20, 0x03, '1', ' ', 0x00, 0x01, 0x01},
	[]byte{
		0x01, 0x01, 0x6A, 0x12, 'a', 'b', 'c', 0x00, ' ', 'E', 'n', 't', 'e', 'r', ' ', 'n',
		'a', 'm', 'e', ':', ' ', 0x00, 0x47, 0xC5, 0x09, 0x03, 0x00, 0x05, 0x01, 0x88, 0x06,
		0x86, 0x08, 0x03, 'x', 'y', 'z', 0x00, 0x85, 0x03, '/', 's', 0x00, 0x01, 0x83, 0x04,
		0x01, 0x83, 0x04, 0x86, 0x07, 0x0A, 0x03, 'N', 0x00, 0x01, 0x01, 0x01,
	},
}

func TestReadHeader(t *testing.T) {
	tests := []Header{
		Header{
			Version:     1,
			PublicID:    1,
			Charset:     3,
			StringTable: []byte{},
		},
		Header{
			Version:     1,
			PublicID:    1,
			Charset:     106,
			StringTable: []byte{'a', 'b', 'c', 0x00, ' ', 'E', 'n', 't', 'e', 'r', ' ', 'n', 'a', 'm', 'e', ':', ' ', 0x00},
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
