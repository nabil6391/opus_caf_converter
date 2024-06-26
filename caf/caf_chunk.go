package caf

import (
	"bufio"
	"encoding/binary"
	"io"

	"github.com/sirupsen/logrus"
)

type CAFChunk struct {
	Header   CAFChunkHeader
	Contents any
}

type CAFChunkHeader struct {
	ChunkType FourByteString
	ChunkSize int64
}

type UnknownContents struct {
	Data []byte
}

type Midi = []byte

func (c *CAFChunk) decode(r *bufio.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &c.Header); err != nil {
		return err
	}
	switch c.Header.ChunkType {
	case ChunkeAudioDescription:
		var cc CAFAudioFormat
		if err := cc.decode(r); err != nil {
			return err
		}
		c.Contents = &cc
	case ChunkChannelLayout:
		var cc CAFChannelLayout
		if err := cc.decode(r); err != nil {
			return err
		}
		c.Contents = &cc
	case ChunkInformation:
		var cc CAFStringsChunk
		if err := cc.decode(r); err != nil {
			return err
		}
		c.Contents = &cc
	case ChunkAudioData:
		var cc DataX
		if err := cc.decode(r, c.Header); err != nil {
			return err
		}
		c.Contents = &cc
	case ChunkPacketTable:
		var cc CAFPacketTable
		if err := cc.decode(r); err != nil {
			return err
		}
		c.Contents = &cc
	case ChunkMidi:
		var cc Midi
		ba := make([]byte, c.Header.ChunkSize)
		if err := binary.Read(r, binary.BigEndian, &ba); err != nil {
			return err
		}
		cc = ba
		c.Contents = cc
	default:
		logrus.Debugf("Got unknown chunk type")
		ba := make([]byte, c.Header.ChunkSize)
		if err := binary.Read(r, binary.BigEndian, &ba); err != nil {
			return err
		}
		c.Contents = &UnknownContents{Data: ba}
	}
	return nil
}

func (c *CAFChunk) Encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, &c.Header); err != nil {
		return err
	}
	switch c.Header.ChunkType {
	case ChunkeAudioDescription:
		cc := c.Contents.(*CAFAudioFormat)
		if err := cc.encode(w); err != nil {
			return err
		}
	case ChunkChannelLayout:
		cc := c.Contents.(*CAFChannelLayout)
		if err := cc.encode(w); err != nil {
			return err
		}
	case ChunkInformation:
		cc := c.Contents.(*CAFStringsChunk)
		if err := cc.encode(w); err != nil {
			return err
		}
	case ChunkAudioData:
		cc := c.Contents.(*DataX)
		if err := cc.encode(w); err != nil {
			return err
		}
	case ChunkPacketTable:
		cc := c.Contents.(*CAFPacketTable)
		if err := cc.encode(w); err != nil {
			return err
		}
	case ChunkMidi:
		midi := c.Contents.(Midi)
		if _, err := w.Write(midi); err != nil {
			return err
		}
	default:
		data := c.Contents.(*UnknownContents).Data
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	return nil
}
