package wbxml

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

var multiByteExamples = []struct {
	data   []byte
	mbuint uint64
}{
	{[]byte{0x81, 0x20}, 0xA0},
	{[]byte{0x60}, 0x60},
	{[]byte{0x83, 0x74}, 500},
}

func TestDecodeMultibyteInteger(t *testing.T) {
	tests := multiByteExamples

	for testID, test := range tests {
		result, err := mbUint(&Decoder{r: bytes.NewReader(test.data)}, 8)

		if err != nil {
			t.Errorf("case %d: unexpected error: %s", testID, err)
			continue
		}

		if result != test.mbuint {
			t.Errorf("case %d: expected %d, got %d", testID, test.mbuint, result)
		}
	}
}

func TestEncodeMultibyteInteger(t *testing.T) {
	tests := multiByteExamples

	for testID, test := range tests {
		w := bytes.NewBuffer(nil)
		err := writeMbUint(&Encoder{w: w}, test.mbuint, 4)

		if err != nil {
			t.Errorf("case %d: unexpected error: %s", testID, err)
			continue
		}

		assert.Equal(t, test.data, w.Bytes(), "case %d", testID)
	}
}
