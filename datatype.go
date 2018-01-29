package wbxml

import (
	"fmt"
	"io"
)

func readByte(d *Decoder) (byte, error) {
	var b [1]byte
	n, err := d.r.Read(b[:])
	d.offset += n
	return b[0], err
}

func rreadByte(d io.Reader) (byte, error) {
	var b [1]byte
	_, err := d.Read(b[:])
	return b[0], err
}

func writeByte(e *Encoder, b byte) error {
	buf := [1]byte{b}
	_, err := e.w.Write(buf[:])
	return err
}

func MbUint(r io.Reader, max int) (uint64, error) {
	var result uint64

	for i := 0; i < max; i++ {
		b, err := rreadByte(r)
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

func mbUint(d *Decoder, max int) (uint64, error) {
	var result uint64

	for i := 0; i < max; i++ {
		b, err := readByte(d)
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

func mbUint32(d *Decoder) (uint32, error) {
	u, err := mbUint(d, 4)
	if err != nil {
		return 0, err
	}
	return uint32(u), nil
}

func writeMbUint(d *Encoder, v uint64, max int) error {
	const M = 8
	var buf [M]byte

	b := byte(v & 0xFF)
	buf[M-1] = b & 0x7f
	v >>= 7

	i := 2
	for ; i <= max && v > 0; i++ {
		b := byte(v & 0xFF)
		buf[M-i] = 0x80 | (b & 0x7f)
		v >>= 7
	}
	if v != 0 {
		return fmt.Errorf("%d is to big to fit on %d bytes", v, max)
	}
	return writeSlice(d, buf[M-i+1:M])
}

func writeMbUint32(d *Encoder, v uint32) error {
	return writeMbUint(d, uint64(v), 4)
}

func readString(d *Decoder) ([]byte, error) {
	result := make([]byte, 0, 8)
	for {
		b, err := readByte(d)
		if err != nil {
			return nil, err
		}
		if b == 0 {
			return result, nil
		}
		result = append(result, b)
	}
}

func writeString(d *Encoder, str []byte) error {
	for _, b := range str {
		err := writeByte(d, b)
		if err != nil {
			return err
		}
	}
	return writeByte(d, 0)
}

func readSlice(d *Decoder, length uint32) ([]byte, error) {
	result := make([]byte, length)
	n, err := d.r.Read(result)
	if err != nil {
		return nil, err
	}
	d.offset += n
	if uint32(n) != length {
		return result[:n], fmt.Errorf("expected %d bytes, got %d", length, n)
	}
	return result, nil
}

func writeSlice(d *Encoder, buf []byte) error {
	n, err := d.w.Write(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("expected %d bytes, got %d", len(buf), n)
	}
	return nil
}

func writeOpaque(d *Encoder, buf Opaque) error {
	err := writeByte(d, gloOpaque)
	if err != nil {
		return err
	}
	err = writeMbUint32(d, uint32(len(buf)))
	if err != nil {
		return err
	}
	return writeSlice(d, buf)
}
