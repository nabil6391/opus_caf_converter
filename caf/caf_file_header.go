package caf

import (
	"encoding/binary"
	"errors"
	"io"
)

type CAFFileHeader struct {
	FileType    FourByteString
	FileVersion int16
	FileFlags   int16
}

func (h *CAFFileHeader) Decode(r io.Reader) error {
	err := binary.Read(r, binary.BigEndian, h)
	if err != nil {
		return err
	}
	if h.FileType != NewFourByteStr("caff") {
		return errors.New("invalid caff header")
	}
	return nil
}

func (h *CAFFileHeader) Encode(w io.Writer) error {
	err := binary.Write(w, binary.BigEndian, h)
	if err != nil {
		return err
	}
	return nil
}
