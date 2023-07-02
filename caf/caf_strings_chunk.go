package caf

import (
	"encoding/binary"
	"io"
)

type CAFStringsChunk struct {
	NumEntries uint32
	Strings    []Information
}

func (c *CAFStringsChunk) decode(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &c.NumEntries); err != nil {
		return err
	}
	for i := uint32(0); i < c.NumEntries; i++ {
		var info Information
		if err := info.decode(r); err != nil {
			return err
		}
		c.Strings = append(c.Strings, info)
	}
	return nil
}

func (c *CAFStringsChunk) encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, &c.NumEntries); err != nil {
		return err
	}
	for i := uint32(0); i < c.NumEntries; i++ {
		if err := c.Strings[i].encode(w); err != nil {
			return err
		}
	}
	return nil
}
