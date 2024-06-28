package caf

import (
	"encoding/binary"
	"errors"
	"io"
)

// Constants
const (
	pageHeaderTypeBeginningOfStream = 0x02
	pageHeaderSignature             = "OggS"
	idPageSignature                 = "OpusHead"
	pageHeaderLen                   = 27
	idPagePayloadLength             = 19
	maxPageSize                     = 65307 // Maximum Ogg page size
)

// Errors
var (
	errNilStream                 = errors.New("stream is nil")
	errBadIDPageSignature        = errors.New("bad header signature")
	errBadIDPageType             = errors.New("wrong header, expected beginning of stream")
	errBadIDPageLength           = errors.New("payload for id page must be 19 bytes")
	errBadIDPagePayloadSignature = errors.New("bad payload signature")
	errShortPageHeader           = errors.New("not enough data for payload header")
)

// OggReader is used to read Ogg files and return page payloads
type OggReader struct {
	stream     io.Reader
	pageBuffer []byte
}

// OggHeader is the metadata from the first two pages in the file (ID and Comment)
type OggHeader struct {
	Version    uint8
	Channels   uint8
	PreSkip    uint16
	SampleRate uint32
	OutputGain uint16
	ChannelMap uint8
}

// OggPageHeader is the metadata for a Page
type OggPageHeader struct {
	GranulePosition uint64
	Signature       [4]byte
	Version         uint8
	HeaderType      uint8
	Serial          uint32
	Index           uint32
	SegmentsCount   uint8
}

func (o *OggReader) readHeaders() (*OggHeader, error) {
	segments, pageHeader, err := o.ParseNextPage()
	if err != nil {
		return nil, err
	}

	if string(pageHeader.Signature[:]) != pageHeaderSignature {
		return nil, errBadIDPageSignature
	}

	if pageHeader.HeaderType != pageHeaderTypeBeginningOfStream {
		return nil, errBadIDPageType
	}

	if len(segments[0]) != idPagePayloadLength {
		return nil, errBadIDPageLength
	}

	if string(segments[0][:8]) != idPageSignature {
		return nil, errBadIDPagePayloadSignature
	}

	return &OggHeader{
		Version:    segments[0][8],
		Channels:   segments[0][9],
		PreSkip:    binary.LittleEndian.Uint16(segments[0][10:12]),
		SampleRate: binary.LittleEndian.Uint32(segments[0][12:16]),
		OutputGain: binary.LittleEndian.Uint16(segments[0][16:18]),
		ChannelMap: segments[0][18],
	}, nil
}

// ParseNextPage reads from stream and returns Ogg page segments, header,
// and an error if there is incomplete page data.
func (o *OggReader) ParseNextPage() ([][]byte, *OggPageHeader, error) {
	if _, err := io.ReadFull(o.stream, o.pageBuffer[:pageHeaderLen]); err != nil {
		return nil, nil, err
	}

	pageHeader := &OggPageHeader{
		Signature:       [4]byte{o.pageBuffer[0], o.pageBuffer[1], o.pageBuffer[2], o.pageBuffer[3]},
		Version:         o.pageBuffer[4],
		HeaderType:      o.pageBuffer[5],
		GranulePosition: binary.LittleEndian.Uint64(o.pageBuffer[6:14]),
		Serial:          binary.LittleEndian.Uint32(o.pageBuffer[14:18]),
		Index:           binary.LittleEndian.Uint32(o.pageBuffer[18:22]),
		SegmentsCount:   o.pageBuffer[26],
	}

	sizeBuffer := o.pageBuffer[pageHeaderLen : pageHeaderLen+int(pageHeader.SegmentsCount)]
	if _, err := io.ReadFull(o.stream, sizeBuffer); err != nil {
		return nil, nil, err
	}

	segments := make([][]byte, 0, pageHeader.SegmentsCount)
	offset := pageHeaderLen + int(pageHeader.SegmentsCount)
	segmentSize := 0

	for _, size := range sizeBuffer {
		segmentSize += int(size)
		if size < 255 {
			if offset+segmentSize > len(o.pageBuffer) {
				return nil, nil, errors.New("segment size exceeds buffer capacity")
			}
			if _, err := io.ReadFull(o.stream, o.pageBuffer[offset:offset+segmentSize]); err != nil {
				return nil, nil, err
			}
			segments = append(segments, o.pageBuffer[offset:offset+segmentSize])
			offset += segmentSize
			segmentSize = 0
		}
	}

	return segments, pageHeader, nil
}

func CalculateFrameSize(tocByte int) uint32 {
	tocConfig := tocByte >> 3
	switch {
	case tocConfig < 12:
		return 960 * (uint32(tocConfig)&3 + 1)
	case tocConfig < 16:
		return 480 << (tocConfig & 1)
	default:
		return 120 << (tocConfig & 3)
	}
}

// NewWith returns a new Ogg reader and Ogg header with an io.Reader input
func NewWith(in io.Reader) (*OggReader, *OggHeader, error) {
	if in == nil {
		return nil, nil, errNilStream
	}

	reader := &OggReader{
		stream:     in,
		pageBuffer: make([]byte, maxPageSize),
	}
	header, err := reader.readHeaders()
	if err != nil {
		return nil, nil, err
	}

	return reader, header, nil
}