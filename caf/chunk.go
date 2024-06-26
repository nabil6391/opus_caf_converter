package caf

import (
	"bufio"
	"encoding/binary"
	"io"

	"github.com/sirupsen/logrus"
)

type Chunk struct {
	Header   ChunkHeader
	Contents any
}

type ChunkHeader struct {
	ChunkType FourByteString
	ChunkSize int64
}

type UnknownContents struct {
	Data []byte
}

type Midi = []byte


func (c *Chunk) decode(r *bufio.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &c.Header); err != nil {
		return err
	}
	switch c.Header.ChunkType {
	case ChunkeAudioDescription:
		{
			var cc AudioFormat
			if err := cc.decode(r); err != nil {
				return err
			}
			c.Contents = &cc
			break
		}
	case ChunkChannelLayout:
		{
			var cc ChannelLayout
			if err := cc.decode(r); err != nil {
				return err
			}
			c.Contents = &cc
			break
		}
	case ChunkInformation:
		{
			var cc CAFStringsChunk
			if err := cc.decode(r); err != nil {
				return err
			}
			c.Contents = &cc
			break
		}
	case ChunkAudioData:
		{
			var cc DataX
			if err := cc.decode(r, c.Header); err != nil {
				return err
			}
			c.Contents = &cc
		}
	case ChunkPacketTable:
		{
			var cc PacketTable
			if err := cc.decode(r); err != nil {
				return err
			}
			c.Contents = &cc
		}
	case ChunkMidi:
		{
			var cc Midi
			ba := make([]byte, c.Header.ChunkSize)
			if err := binary.Read(r, binary.BigEndian, &ba); err != nil {
				return err
			}
			cc = ba
			c.Contents = cc
		}
	default:
		{
			logrus.Debugf("Got unknown chunk type")
			ba := make([]byte, c.Header.ChunkSize)
			if err := binary.Read(r, binary.BigEndian, &ba); err != nil {
				return err
			}
			c.Contents = &UnknownContents{Data: ba}
		}
	}
	return nil
}

func (c *Chunk) Encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, &c.Header); err != nil {
		return err
	}
	switch c.Header.ChunkType {
	case ChunkeAudioDescription:
		{
			cc := c.Contents.(*AudioFormat)
			if err := cc.encode(w); err != nil {
				return err
			}
			break
		}
	case ChunkChannelLayout:
		{
			cc := c.Contents.(*ChannelLayout)
			if err := cc.encode(w); err != nil {
				return err
			}
			break
		}
	case ChunkInformation:
		{
			cc := c.Contents.(*CAFStringsChunk)
			if err := cc.encode(w); err != nil {
				return err
			}
			break
		}
	case ChunkAudioData:
		{
			cc := c.Contents.(*DataX)
			if err := cc.encode(w); err != nil {
				return err
			}
			c.Contents = &cc
		}
	case ChunkPacketTable:
		{
			cc := c.Contents.(*PacketTable)
			if err := cc.encode(w); err != nil {
				return err
			}
			c.Contents = &cc
		}
	case ChunkMidi:
		{
			midi := c.Contents.(Midi)
			if _, err := w.Write(midi); err != nil {
				return err
			}

		}
	default:
		{
			data := c.Contents.(*UnknownContents).Data
			if _, err := w.Write(data); err != nil {
				return err
			}
		}
	}
	return nil
}
