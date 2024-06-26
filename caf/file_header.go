package caf

import (
	"encoding/binary"
	"errors"
	"io"
)

type FileHeader struct {
	FileType    FourByteString
	FileVersion int16
	FileFlags   int16
}

func (h *FileHeader) Decode(r io.Reader) error {
	err := binary.Read(r, binary.BigEndian, h)
	if err != nil {
		return err
	}
	if h.FileType != NewFourByteStr("caff") {
		return errors.New("invalid caff header")
	}
	return nil
}

func (h *FileHeader) Encode(w io.Writer) error {
	err := binary.Write(w, binary.BigEndian, h)
	if err != nil {
		return err
	}
	return nil
}
