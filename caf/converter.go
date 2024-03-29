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

	audioData, trailing_data, frame_size := readOpusData(ogg)
	cf := buildCafFile(header, audioData, trailing_data, frame_size)

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
	frame_size := 0
	trailing_data := make([]uint64, 0)

	for {
		segments, header, err := ogg.ParseNextPage()

		if errors.Is(err, io.EOF) {
			break
		} else if bytes.HasPrefix(segments[0], []byte("OpusTags")) {
			continue
		}

		if err != nil {
			panic(err)
		}

		for i := range segments {
			trailing_data = append(trailing_data, uint64(len(segments[i])))
			audioData = append(audioData, segments[i]...)
		}

		if header.index == 2 {
			tmpPacket := segments[0]
			if len(tmpPacket) > 0 {
				tmptoc := int(tmpPacket[0] & 255)
				tocConfig := tmptoc >> 3

				switch {
				case tocConfig < 12:
					frame_size = 960 * (tocConfig&3 + 1)
				case tocConfig < 16:
					frame_size = 480 << (tocConfig & 1)
				default:
					frame_size = 120 << (tocConfig & 3)
				}
			}
		}
	}

	return audioData, trailing_data, frame_size
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

func buildCafFile(header *OggHeader, audioData []byte, trailing_data []uint64, frame_size int) *FileData {
	len_audio := len(audioData)
	packets := len(trailing_data)
	frames := frame_size * packets

	packetTableLength := calculatePacketTableLength(trailing_data)

	cf := &FileData{}
	cf.FileHeader = FileHeader{FileType: FourByteString{99, 97, 102, 102}, FileVersion: 1, FileFlags: 0}

	c := Chunk{
		Header:   ChunkHeader{ChunkType: ChunkeAudioDescription, ChunkSize: 32},
		Contents: &AudioFormat{SampleRate: 48000, FormatID: FourByteString{111, 112, 117, 115}, FormatFlags: 0x00000000, BytesPerPacket: 0, FramesPerPacket: uint32(frame_size), BitsPerChannel: 0, ChannelsPerPacket: uint32(header.Channels)},
	}

	cf.Chunks = append(cf.Chunks, c)
	var channelLayoutTag uint32
	if header.Channels == 2 {
		channelLayoutTag = 6619138 // kAudioChannelLayoutTag_Stereo
	} else {
		channelLayoutTag = 6553601 // kAudioChannelLayoutTag_Mono
	}

	c1 := Chunk{
		Header: ChunkHeader{ChunkType: ChunkChannelLayout, ChunkSize: 12},
		Contents: &ChannelLayout{ChannelLayoutTag: channelLayoutTag, ChannelBitmap: 0x0, NumberChannelDescriptions: 0,
			Channels: []ChannelDescription{},
		},
	}

	cf.Chunks = append(cf.Chunks, c1)

	c2 := Chunk{
		Header:   ChunkHeader{ChunkType: ChunkInformation, ChunkSize: 26},
		Contents: &CAFStringsChunk{NumEntries: 1, Strings: []Information{{Key: "encoder\x00", Value: "Lavf59.27.100\x00"}}},
	}

	cf.Chunks = append(cf.Chunks, c2)
	c3 := Chunk{
		Header:   ChunkHeader{ChunkType: ChunkAudioData, ChunkSize: int64(len_audio + 4)},
		Contents: &DataX{Bytes: audioData},
	}

	cf.Chunks = append(cf.Chunks, c3)

	c4 := Chunk{
		Header: ChunkHeader{ChunkType: ChunkPacketTable, ChunkSize: int64(packetTableLength)},
		Contents: &PacketTable{
			Header: PacketTableHeader{NumberPackets: int64(packets), NumberValidFrames: int64(frames), PrimingFramess: 0, RemainderFrames: 0},
			Entry:  trailing_data,
		},
	}

	cf.Chunks = append(cf.Chunks, c4)

	return cf
}
