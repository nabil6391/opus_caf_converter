package caf

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"os"
)

var ChunkeAudioDescription = NewFourByteStr("desc")
var ChunkChannelLayout = NewFourByteStr("chan")
var ChunkInformation = NewFourByteStr("info")
var ChunkAudioData = NewFourByteStr("data")
var ChunkPacketTable = NewFourByteStr("pakt")
var ChunkMidi = NewFourByteStr("midi")

func ConvertOpusToCaf(inputFile string, outputFile string) error {
	inFile, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer inFile.Close()

	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	bufferedReader := bufio.NewReaderSize(inFile, 32*1024) // Increased buffer size
	bufferedWriter := bufio.NewWriterSize(outFile, 32*1024) // Increased buffer size
	
	ogg, header, err := NewWith(bufferedReader)
	if err != nil {
		return err
	}

	// Write CAF file header
	cafHeader := CAFFileHeader{
		FileType:    NewFourByteStr("caff"),
		FileVersion: 1,
		FileFlags:   0,
	}
	if err := cafHeader.Encode(bufferedWriter); err != nil {
		return err
	}

	frameSize := uint32(0)

	// Write audio description chunk
	descChunk := CAFChunk{
		Header: CAFChunkHeader{ChunkType: ChunkeAudioDescription, ChunkSize: 32},
		Contents: &CAFAudioFormat{
			SampleRate:        48000,
			FormatID:          NewFourByteStr("opus"),
			FormatFlags:       0x00000000,
			BytesPerPacket:    0,
			FramesPerPacket:   frameSize,
			BitsPerChannel:    0,
			ChannelsPerPacket: uint32(header.Channels),
		},
	}
	if err := descChunk.Encode(bufferedWriter); err != nil {
		return err
	}

	// Write channel layout chunk
	chanChunk := CAFChunk{
		Header: CAFChunkHeader{ChunkType: ChunkChannelLayout, ChunkSize: 12},
		Contents: &CAFChannelLayout{
			ChannelLayoutTag:          GetChannelLayoutForChannels(uint32(header.Channels)),
			ChannelBitmap:             0x0,
			NumberChannelDescriptions: 0,
		},
	}
	if err := chanChunk.Encode(bufferedWriter); err != nil {
		return err
	}

	// Write information chunk
	infoChunk := CAFChunk{
		Header:   CAFChunkHeader{ChunkType: ChunkInformation, ChunkSize: 25},
		Contents: &CAFStringsChunk{NumEntries: 1, Strings: []Information{{Key: "encoder\x00", Value: "Lavf60.3.100\x00"}}},
	}
	if err := infoChunk.Encode(bufferedWriter); err != nil {
		return err
	}

	dataOffset := bufferedWriter.Buffered()
	// Write audio data chunk header
	dataChunkHeader := CAFChunkHeader{ChunkType: ChunkAudioData, ChunkSize: -1}
	if err := binary.Write(bufferedWriter, binary.BigEndian, &dataChunkHeader); err != nil {
		return err
	}

	// Write edit count
	var editCount uint32 = 0
	if err := binary.Write(bufferedWriter, binary.BigEndian, &editCount); err != nil {
		return err
	}

	// Process audio data
	var totalBytes int64
	packetSizes := make([]uint64, 0, 1024) // Pre-allocate slice

	for {
		segments, pageHeader, err := ogg.ParseNextPage()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		segment := segments[0]
		index := pageHeader.Index
		if index == 2 && len(segment) > 0 {
			tmptoc := int(segment[0] & 255)
			frameSize = CalculateFrameSize(tmptoc)
		}

		if index == 1 && bytes.HasPrefix(segment, []byte("OpusTags")) {
			continue
		}

		if index <= 2 && bytes.HasPrefix(segment, []byte("OpusHead")) {
			continue
		}

		for _, segment := range segments {
			segmentLen := len(segment)
			totalBytes += int64(segmentLen)
			packetSizes = append(packetSizes, uint64(segmentLen))

			if _, err := bufferedWriter.Write(segment); err != nil {
				return err
			}
		}
	}

	packetTableLength := calculatePacketTableLength(packetSizes)

	// Write packet table chunk
	paktChunk := CAFChunk{
		Header: CAFChunkHeader{ChunkType: ChunkPacketTable, ChunkSize: int64(packetTableLength)},
		Contents: &CAFPacketTable{
			Header: CAFPacketTableHeader{
				NumberPackets:     int64(len(packetSizes)),
				NumberValidFrames: int64(int(frameSize) * len(packetSizes)),
				PrimingFrames:     0,
				RemainderFrames:   0,
			},
			Entry: packetSizes,
		},
	}
	if err := paktChunk.Encode(bufferedWriter); err != nil {
		return err
	}

	// Flush the buffered writer
	if err := bufferedWriter.Flush(); err != nil {
		return err
	}

	// Update frame size in audio description chunk
	if _, err := outFile.Seek(40, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(outFile, binary.BigEndian, frameSize); err != nil {
		return err
	}

	// Update data chunk size
	if _, err := outFile.Seek(int64(dataOffset + 4), io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(outFile, binary.BigEndian, totalBytes+4); err != nil {
		return err
	}

	return nil
}

func calculatePacketTableLength(trailing_data []uint64) int {
	packetTableLength := 24

	for _, value := range trailing_data {
		if value < 0x80 {
			packetTableLength++
		} else if value < 0x4000 {
			packetTableLength += 2
		} else if value < 0x200000 {
			packetTableLength += 3
		} else if value < 0x10000000 {
			packetTableLength += 4
		} else {
			packetTableLength += 5
		}
	}
	return packetTableLength
}