package utils

import (
	"io"
)

func DecodeLength(bytes []byte) (int, int, error) {
	ln := 0
	size := 0
	for {
		if len(bytes) == 0 {
			return 0, 0, io.ErrUnexpectedEOF
		}
		elem := int(bytes[0])
		bytes = bytes[1:]
		ln |= (elem & 0x7f) << (size * 7)
		size += 1
		if (elem & 0x80) == 0 {
			break
		}
	}
	return ln, size, nil
}

func EncodeLength(bytes *[]byte, length int) {
	remLen := length
	for {
		elem := byte(remLen & 0x7f)
		remLen >>= 7
		if remLen == 0 {
			*bytes = append(*bytes, elem)
			break
		} else {
			elem |= 0x80
			*bytes = append(*bytes, elem)
		}
	}
}
