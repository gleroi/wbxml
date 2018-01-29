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
		result, err := mbUint(&Decoder{r: bytes.NewReader(test.input)}, 8)

		if err != nil {
			t.Errorf("case %d: unexpected error: %s", testID, err)
			continue
		}

		if result != test.expected {
			t.Errorf("case %d: expected %d, got %d", testID, test.expected, result)
		}
	}
}

func TestEncodeMultibyteInteger(t *testing.T) {
	tests := []struct {
		expected []byte
		input    uint64
	}{
		{[]byte{0x81, 0x20}, 0xA0},
		{[]byte{0x60}, 0x60},
	}

	for testID, test := range tests {
		w := bytes.NewBuffer(nil)
		err := writeMbUint(&Encoder{w: w}, test.input, 8)

		if err != nil {
			t.Errorf("case %d: unexpected error: %s", testID, err)
			continue
		}

		assert.Equal(t, test.expected, w.Bytes(), "case %d", testID)
	}
}
