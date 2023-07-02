package caf

import (
	"encoding/binary"
	"io"
)

// OggReader is used to read Ogg files and return page payloads
type OggReader struct {
	stream io.Reader
}

// OggHeader is the metadata from the first two pages
// in the file (ID and Comment)
//
// https://tools.ietf.org/html/rfc7845.html#section-3
type OggHeader struct {
	ChannelMap uint8
	Channels   uint8
	OutputGain uint16
	PreSkip    uint16
	SampleRate uint32
	Version    uint8
}

// OggPageHeader is the metadata for a Page
// Pages are the fundamental unit of multiplexing in an Ogg stream
//
// https://tools.ietf.org/html/rfc7845.html#section-1
type OggPageHeader struct {
	GranulePosition uint64

	sig           [4]byte
	version       uint8
	headerType    uint8
	serial        uint32
	index         uint32
	segmentsCount uint8
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
		sig: [4]byte{h[0], h[1], h[2], h[3]},
	}

	pageHeader.version = h[4]
	pageHeader.headerType = h[5]
	pageHeader.GranulePosition = binary.LittleEndian.Uint64(h[6 : 6+8])
	pageHeader.serial = binary.LittleEndian.Uint32(h[14 : 14+4])
	pageHeader.index = binary.LittleEndian.Uint32(h[18 : 18+4])
	pageHeader.segmentsCount = h[26]

	sizeBuffer := make([]byte, pageHeader.segmentsCount)
	if _, err = io.ReadFull(o.stream, sizeBuffer); err != nil {
		return nil, nil, err
	}

	newArr := make([]int, 0)
	// Iteraring throug all segments to check if there are lacing packets.
	//  If a segment is 255 bytes long, it means that there must be a following segment for the same packet (which may be again 255 bytes long)
	for i := 0; i < len(sizeBuffer); i++ {
		if sizeBuffer[i] == 255 {
			sum := int(sizeBuffer[i])
			i++
			for i < len(sizeBuffer) && sizeBuffer[i] == 255 {
				sum += int(sizeBuffer[i])
				i++
			}
			if i < len(sizeBuffer) {
				sum += int(sizeBuffer[i])
			}
			newArr = append(newArr, sum)
		} else {
			newArr = append(newArr, int(sizeBuffer[i]))
		}
	}

	segments := make([][]byte, 0, len(newArr))

	for _, s := range newArr {
		segment := make([]byte, int(s))
		if _, err = io.ReadFull(o.stream, segment); err != nil {
			return nil, nil, err
		}

		segments = append(segments, segment)
	}

	return segments, pageHeader, nil
}
