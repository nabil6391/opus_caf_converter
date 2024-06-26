package caf

import (
	"encoding/binary"
	"io"
)

// OggReader is used to read Ogg files and return page payloads
type OggReader struct {
	stream io.Reader
}

// OggHeader is the metadata from the first two pages in the file (ID and Comment)
type OggHeader struct {
	ChannelMap uint8
	Channels   uint8
	OutputGain uint16
	PreSkip    uint16
	SampleRate uint32
	Version    uint8
}

// OggPageHeader is the metadata for a Page
type OggPageHeader struct {
	GranulePosition uint64
	sig             [4]byte
	version         uint8
	headerType      uint8
	serial          uint32
	index           uint32
	segmentsCount   uint8
}

func (o *OggReader) readHeaders() (*OggHeader, error) {
	segments, pageHeader, err := o.ParseNextPage()
	if err != nil {
		return nil, err
	}

	header := &OggHeader{}
	if string(pageHeader.sig[:]) != pageHeaderSignature {
		return nil, errBadIDPageSignature
	}

	if pageHeader.headerType != pageHeaderTypeBeginningOfStream {
		return nil, errBadIDPageType
	}

	if len(segments[0]) != idPagePayloadLength {
		return nil, errBadIDPageLength
	}

	if s := string(segments[0][:8]); s != idPageSignature {
		return nil, errBadIDPagePayloadSignature
	}

	header.Version = segments[0][8]
	header.Channels = segments[0][9]
	header.PreSkip = binary.LittleEndian.Uint16(segments[0][10:12])
	header.SampleRate = binary.LittleEndian.Uint32(segments[0][12:16])
	header.OutputGain = binary.LittleEndian.Uint16(segments[0][16:18])
	header.ChannelMap = segments[0][18]

	return header, nil
}

// ParseNextPage reads from stream and returns Ogg page segments, header,
// and an error if there is incomplete page data.
func (o *OggReader) ParseNextPage() ([][]byte, *OggPageHeader, error) {
	h := make([]byte, pageHeaderLen)

	n, err := io.ReadFull(o.stream, h)
	if err != nil {
		return nil, nil, err
	} else if n < len(h) {
		return nil, nil, errShortPageHeader
	}

	pageHeader := &OggPageHeader{
		sig:           [4]byte{h[0], h[1], h[2], h[3]},
		version:       h[4],
		headerType:    h[5],
		GranulePosition: binary.LittleEndian.Uint64(h[6:14]),
		serial:        binary.LittleEndian.Uint32(h[14:18]),
		index:         binary.LittleEndian.Uint32(h[18:22]),
		segmentsCount: h[26],
	}

	sizeBuffer := make([]byte, pageHeader.segmentsCount)
	if _, err = io.ReadFull(o.stream, sizeBuffer); err != nil {
		return nil, nil, err
	}

	segments := make([][]byte, 0, pageHeader.segmentsCount)
	var currentSegment []byte
	var segmentSize int

	for _, size := range sizeBuffer {
		segmentSize += int(size)
		if size < 255 {
			currentSegment = make([]byte, segmentSize)
			if _, err = io.ReadFull(o.stream, currentSegment); err != nil {
				return nil, nil, err
			}
			segments = append(segments, currentSegment)
			segmentSize = 0
		}
	}

	return segments, pageHeader, nil
}
