package caf

import (
	"encoding/binary"
	"io"
)

type ChannelLayout struct {
	ChannelLayoutTag          uint32
	ChannelBitmap             uint32
	NumberChannelDescriptions uint32
	Channels                  []ChannelDescription
}

type ChannelDescription struct {
	ChannelLabel uint32
	ChannelFlags uint32
	Coordinates  [3]float32
}

func (c *ChannelLayout) decode(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &c.ChannelLayoutTag); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &c.ChannelBitmap); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &c.NumberChannelDescriptions); err != nil {
		return err
	}
	for i := uint32(0); i < c.NumberChannelDescriptions; i++ {
		var channelDesc ChannelDescription
		if err := binary.Read(r, binary.BigEndian, &channelDesc); err != nil {
			return err
		}
		c.Channels = append(c.Channels, channelDesc)
	}
	return nil
}

func (c *ChannelLayout) encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, &c.ChannelLayoutTag); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, &c.ChannelBitmap); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, &c.NumberChannelDescriptions); err != nil {
		return err
	}
	for i := uint32(0); i < c.NumberChannelDescriptions; i++ {
		if err := binary.Write(w, binary.BigEndian, &c.Channels[i]); err != nil {
			return err
		}
	}
	return nil
}
