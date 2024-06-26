package caf

import (
	"bufio"
	"errors"
	"io"
)

func encodeInt(w io.Writer, i uint64) error {
	var byts []byte
	var cur = i
	for {
		val := byte(cur & 127)
		cur = cur >> 7
		byts = append(byts, val)
		if cur == 0 {
			break
		}
	}
	for i := len(byts) - 1; i >= 0; i-- {
		var val = byts[i]
		if i > 0 {
			val = val | 0x80
		}
		if w != nil {
			if n, err := w.Write([]byte{val}); err != nil {
				return err
			} else {
				if n != 1 {
					return errors.New("error writing")
				}
			}
		}
	}
	return nil
}

func decodeInt(r *bufio.Reader) (uint64, error) {
	var res uint64 = 0
	var bytesRead = 0
	for {
		byt, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		bytesRead += 1
		res = res << 7
		res = res | uint64(byt&127)
		if byt&128 == 0 || bytesRead >= 8 {
			return res, nil
		}
	}
}

func readString(r io.Reader) (string, error) {
	var bs []byte
	var b = make([]byte, 1)
	for {
		if _, err := r.Read(b); err != nil {
			return "", err
		} else {
			bs = append(bs, b[0])
			if b[0] == 0 {
				break
			}
		}
	}
	return string(bs), nil
}

func writeString(w io.Writer, s string) error {
	byteString := []byte(s)
	_, err := w.Write(byteString)
	return err
}

// CAF Encoding and Decoding
type FourByteString [4]byte

func NewFourByteStr(str string) FourByteString {
	if len(str) != 4 {
		panic("FourByteString must be 4 bytes")
	}
	res := FourByteString{}
	for i, v := range str {
		res[i] = byte(v)
	}
	return res
}