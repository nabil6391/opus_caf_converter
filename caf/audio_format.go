package caf

import (
	"encoding/binary"
	"io"
)

type AudioFormat struct {
	SampleRate        float64
	FormatID          FourByteString
	FormatFlags       uint32
	BytesPerPacket    uint32
	FramesPerPacket   uint32
	ChannelsPerPacket uint32
	BitsPerChannel    uint32
}

func (c *AudioFormat) decode(r io.Reader) error {
	return binary.Read(r, binary.BigEndian, c)
}

func (c *AudioFormat) encode(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, c)
}
