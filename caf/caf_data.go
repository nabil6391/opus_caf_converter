package caf

import (
	"bufio"
	"encoding/binary"
	"io"
)

type DataX struct {
	EditCount uint32
	Bytes     []byte
}

func (c *DataX) decode(r *bufio.Reader, h CAFChunkHeader) error {
	if err := binary.Read(r, binary.BigEndian, &c.EditCount); err != nil {
		return err
	}
	if h.ChunkSize == -1 {
		// read until end
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		c.Bytes = data
	} else {
		dataLength := h.ChunkSize - 4 /* for edit count*/
		data, err := io.ReadAll(io.LimitReader(r, dataLength))
		if err != nil {
			return err
		}
		c.Bytes = data
	}
	return nil
}

func (c *DataX) encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, &c.EditCount); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, &c.Bytes); err != nil {
		return err
	}
	return nil
}
