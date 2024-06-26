package caf

import (
	"bufio"
	"encoding/binary"
	"io"
)

type CAFPacketTable struct {
	Header CAFPacketTableHeader
	Entry  []uint64
}

type CAFPacketTableHeader struct {
	NumberPackets     int64
	NumberValidFrames int64
	PrimingFrames    int32
	RemainderFrames   int32
}


func (c *CAFPacketTable) decode(r *bufio.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &c.Header); err != nil {
		return err
	}
	for i := 0; i < int(c.Header.NumberPackets); i++ {
		if val, err := decodeInt(r); err != nil {
			return err
		} else {
			c.Entry = append(c.Entry, val)
		}
	}
	return nil
}

func (c *CAFPacketTable) encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, c.Header); err != nil {
		return err
	}
	for i := 0; i < int(c.Header.NumberPackets); i++ {
		if err := encodeInt(w, c.Entry[i]); err != nil {
			return err
		}
	}
	return nil
}
