package caf

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
)

// Opus Decoding
const (
	pageHeaderTypeBeginningOfStream = 0x02
	pageHeaderSignature             = "OggS"

	idPageSignature = "OpusHead"

	pageHeaderLen       = 27
	idPagePayloadLength = 19
)

var (
	errNilStream                 = errors.New("stream is nil")
	errBadIDPageSignature        = errors.New("bad header signature")
	errBadIDPageType             = errors.New("wrong header, expected beginning of stream")
	errBadIDPageLength           = errors.New("payload for id page must be 19 bytes")
	errBadIDPagePayloadSignature = errors.New("bad payload signature")
	errShortPageHeader           = errors.New("not enough data for payload header")
)

var ChunkeAudioDescription = NewFourByteStr("desc")
var ChunkChannelLayout = NewFourByteStr("chan")
var ChunkInformation = NewFourByteStr("info")
var ChunkAudioData = NewFourByteStr("data")
var ChunkPacketTable = NewFourByteStr("pakt")
var ChunkMidi = NewFourByteStr("midi")

func ConvertOpusToCaf(inputFile string, outputFile string) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Println("error closing the file", err)
		}
	}()

	ogg, header, err := newWith(file)
	if err != nil {
		return err
	}

	audioData, trailingData, frameSize := readOpusData(ogg)
	cf := buildCafFile(header, audioData, trailingData, frameSize)

	outputBuffer := &bytes.Buffer{}
	if cf.Encode(outputBuffer) != nil {
		return errors.New("error encoding")
	}

	output := outputBuffer.Bytes()
	outfile, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	defer func() {
		if err := outfile.Close(); err != nil {
			log.Println("error closing the file", err)
		}
	}()

	_, err = outfile.Write(output)
	if err != nil {
		return err
	}

	return nil
}

// newWith returns a new Ogg reader and Ogg header with an io.Reader input
func newWith(in io.Reader) (*OggReader, *OggHeader, error) {

	if in == nil {
		return nil, nil, errNilStream
	}

	reader := &OggReader{
		stream: in,
	}

	header, err := reader.readHeaders()
	if err != nil {
		return nil, nil, err
	}

	return reader, header, nil
}

func readOpusData(ogg *OggReader) ([]byte, []uint64, int) {
	audioData := []byte{}
	trailingData := make([]uint64, 0)
	frameSize := 0

	for {
		segments, header, err := ogg.ParseNextPage()

		if errors.Is(err, io.EOF) {
			break
		} else if bytes.HasPrefix(segments[0], []byte("OpusTags")) {
			continue
		}

		if err != nil {
			log.Printf("Error parsing Ogg page: %v", err)
			break
		}

		for _, segment := range segments {
			if err != nil {
				log.Printf("Error parsing Opus packet: %v", err)
				continue
			}

			trailingData = append(trailingData, uint64(len(segment)))
			audioData = append(audioData, segment...)
		}

		if header.index == 2 {
			tmpPacket := segments[0]
			if len(tmpPacket) > 0 {
				tmptoc := int(tmpPacket[0] & 255)
				tocConfig := tmptoc >> 3

				switch {
				case tocConfig < 12:
					frameSize = 960 * (tocConfig&3 + 1)
				case tocConfig < 16:
					frameSize = 480 << (tocConfig & 1)
				default:
					frameSize = 120 << (tocConfig & 3)
				}
			}
		}
	}

	return audioData, trailingData, frameSize
}

func calculatePacketTableLength(trailing_data []uint64) int {
	packetTableLength := 24

	// // Check how much chunk size is needed each segment by BER encoding
	for i := 0; i < len(trailing_data); i++ {
		value := uint32(trailing_data[i])
		numBytes := 0
		if (value & 0x7f) == value {
			numBytes = 1
		} else if (value & 0x3fff) == value {
			numBytes = 2
		} else if (value & 0x1fffff) == value {
			numBytes = 3
		} else if (value & 0x0fffffff) == value {
			numBytes = 4
		} else {
			numBytes = 5
		}
		packetTableLength += numBytes
	}
	return packetTableLength
}

func buildCafFile(header *OggHeader, audioData []byte, trailingData []uint64, frameSize int) *CAFFileData {
	lenAudio := len(audioData)
	packets := len(trailingData)
	frames := frameSize * packets

	packetTableLength := calculatePacketTableLength(trailingData)

	cf := &CAFFileData{}
	cf.CAFFileHeader = CAFFileHeader{FileType: FourByteString{99, 97, 102, 102}, FileVersion: 1, FileFlags: 0}

	c := CAFChunk{
		Header:   CAFChunkHeader{ChunkType: ChunkeAudioDescription, ChunkSize: 32},
		Contents: &CAFAudioFormat{SampleRate: 48000, FormatID: FourByteString{111, 112, 117, 115}, FormatFlags: 0x00000000, BytesPerPacket: 0, FramesPerPacket: uint32(frameSize), BitsPerChannel: 0, ChannelsPerPacket: uint32(header.Channels)},
	}

	cf.Chunks = append(cf.Chunks, c)

	c1 := CAFChunk{
		Header: CAFChunkHeader{ChunkType: ChunkChannelLayout, ChunkSize: 12},
		Contents: &CAFChannelLayout{ChannelLayoutTag: GetChannelLayoutForChannels(uint32(header.Channels)), ChannelBitmap: 0x0, NumberChannelDescriptions: 0,
			Channels: []CAFChannelDescription{},
		},
	}

	cf.Chunks = append(cf.Chunks, c1)

	c2 := CAFChunk{
		Header:   CAFChunkHeader{ChunkType: ChunkInformation, ChunkSize: 26},
		Contents: &CAFStringsChunk{NumEntries: 1, Strings: []Information{{Key: "encoder\x00", Value: "Lavf59.27.100\x00"}}},
	}

	cf.Chunks = append(cf.Chunks, c2)
	c3 := CAFChunk{
		Header:   CAFChunkHeader{ChunkType: ChunkAudioData, ChunkSize: int64(lenAudio + 4)},
		Contents: &DataX{Bytes: audioData},
	}

	cf.Chunks = append(cf.Chunks, c3)

	c4 := CAFChunk{
		Header: CAFChunkHeader{ChunkType: ChunkPacketTable, ChunkSize: int64(packetTableLength)},
		Contents: &CAFPacketTable{
			Header: CAFPacketTableHeader{NumberPackets: int64(packets), NumberValidFrames: int64(frames), PrimingFrames: 0, RemainderFrames: 0},
			Entry:  trailingData,
		},
	}

	cf.Chunks = append(cf.Chunks, c4)

	return cf
}

