package wbxml

import (
	"fmt"
	"io"
)

func readByte(r io.Reader) (byte, error) {
	var b [1]byte
	_, err := r.Read(b[:])
	return b[0], err
}

func mbUint(r io.Reader, max int) (uint64, error) {
	var result uint64

	for i := 0; i < max; i++ {
		b, err := readByte(r)
		if err != nil {
			return 0, err
		}

		result = (result << 7) | (uint64(b) & 0x7f)

		if b&0x80 == 0x00 { // final byte
			return result, nil
		}
	}
	return 0, fmt.Errorf("multi-byte integer is longer than expected %d bytes", max)
}

func mbUint32(r io.Reader) (uint32, error) {
	u, err := mbUint(r, 4)
	if err != nil {
		return 0, err
	}
	return uint32(u), nil
}
