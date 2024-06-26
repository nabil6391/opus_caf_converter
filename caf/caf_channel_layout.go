package caf

import (
	"encoding/binary"
	"io"
)

const (
	kCAFChannelLayoutTag_Mono   = 100<<16 | 1
	kCAFChannelLayoutTag_Stereo = 101<<16 | 2
)

type CAFChannelLayout struct {
	ChannelLayoutTag          uint32
	ChannelBitmap             uint32
	NumberChannelDescriptions uint32
	Channels                  []CAFChannelDescription
}

type CAFChannelDescription struct {
	ChannelLabel uint32
	ChannelFlags uint32
	Coordinates  [3]float32
}

func (c *CAFChannelLayout) decode(r io.Reader) error {
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
		var channelDesc CAFChannelDescription
		if err := binary.Read(r, binary.BigEndian, &channelDesc); err != nil {
			return err
		}
		c.Channels = append(c.Channels, channelDesc)
	}
	return nil
}

func (c *CAFChannelLayout) encode(w io.Writer) error {
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

func GetChannelLayoutForChannels(channels uint32) uint32 {
	switch channels {
	case 1:
		return kCAFChannelLayoutTag_Mono
	case 2:
		return kCAFChannelLayoutTag_Stereo
	// Add more cases as needed
	default:
		return 0 // Unknown layout
	}
}
