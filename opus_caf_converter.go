package caf

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// CAF Encoding and Decoding
type FourByteString [4]byte

var ChunkTypeAudioDescription = stringToChunkType("desc")
var ChunkTypeChannelLayout = stringToChunkType("chan")
var ChunkTypeInformation = stringToChunkType("info")
var ChunkTypeAudioData = stringToChunkType("data")
var ChunkTypePacketTable = stringToChunkType("pakt")
var ChunkTypeMidi = stringToChunkType("midi")

func stringToChunkType(str string) (result FourByteString) {
	for i, v := range str {
		result[i] = byte(v)
	}
	return
}

type FileHeader struct {
	FileType    FourByteString
	FileVersion int16
	FileFlags   int16
}

type ChunkHeader struct {
	ChunkType FourByteString
	ChunkSize int64
}

type Data struct {
	EditCount uint32
	Data      []byte
}

type AudioFormat struct {
	SampleRate        float64
	FormatID          FourByteString
	FormatFlags       uint32
	BytesPerPacket    uint32
	FramesPerPacket   uint32
	ChannelsPerPacket uint32
	BitsPerChannel    uint32
}

type PacketTableHeader struct {
	NumberPackets     int64
	NumberValidFrames int64
	PrimingFramess    int32
	RemainderFrames   int32
}

type PacketTable struct {
	Header PacketTableHeader
	Entry  []uint64
}

func encodeInt(w io.Writer, i uint64) error {
	var byts []byte
	var cur = i
	for {
		val := byte(cur & 127)
		cur = cur >> 7
		byts = append(byts, val)
		if cur == 0 {
			break
		}
	}
	for i := len(byts) - 1; i >= 0; i-- {
		var val = byts[i]
		if i > 0 {
			val = val | 0x80
		}
		if w != nil {
			if n, err := w.Write([]byte{val}); err != nil {
				return err
			} else {
				if n != 1 {
					return errors.New("error writing")
				}
			}
		}
	}
	return nil
}

func decodeInt(r *bufio.Reader) (uint64, error) {
	var res uint64 = 0
	var bytesRead = 0
	for {
		byt, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		bytesRead += 1
		res = res << 7
		res = res | uint64(byt&127)
		if byt&128 == 0 || bytesRead >= 8 {
			return res, nil
		}
	}
}

func (c *PacketTable) decode(r *bufio.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &c.Header); err != nil {
		return err
	}
	for i := 0; i < int(c.Header.NumberPackets); i++ {
		if val, err := decodeInt(r); err != nil {
			return err
		} else {
			c.Entry = append(c.Entry, val)
		}
	}
	return nil
}

func (c *PacketTable) encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, c.Header); err != nil {
		return err
	}
	for i := 0; i < int(c.Header.NumberPackets); i++ {
		if err := encodeInt(w, c.Entry[i]); err != nil {
			return err
		}
	}
	return nil
}

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

type UnknownContents struct {
	Data []byte
}

type Midi = []byte

type File struct {
	FileHeader FileHeader
	Chunks     []Chunk
}

func (cf *File) Decode(r io.Reader) error {
	bufferedReader := bufio.NewReader(r)
	var fileHeader FileHeader
	if err := fileHeader.Decode(bufferedReader); err != nil {
		return err
	}
	cf.FileHeader = fileHeader
	for {
		var c Chunk
		if err := c.decode(bufferedReader); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		cf.Chunks = append(cf.Chunks, c)
	}
	return nil
}

func (cf *File) Encode(w io.Writer) error {
	if err := cf.FileHeader.Encode(w); err != nil {
		return err
	}
	for _, c := range cf.Chunks {
		if err := c.Encode(w); err != nil {
			return err
		}
	}
	return nil
}

type Information struct {
	Key   string
	Value string
}

func readString(r io.Reader) (string, error) {
	var bs []byte
	var b = make([]byte, 1)
	for {
		if _, err := r.Read(b); err != nil {
			return "", err
		} else {
			bs = append(bs, b[0])
			if b[0] == 0 {
				break
			}
		}
	}
	return string(bs), nil
}

func writeString(w io.Writer, s string) error {
	byteString := []byte(s)
	_, err := w.Write(byteString)
	return err
}

func (c *Information) decode(r io.Reader) error {
	if key, err := readString(r); err != nil {
		return err
	} else {
		c.Key = key
	}
	if value, err := readString(r); err != nil {
		return err
	} else {
		c.Value = value
	}

	return nil
}

func (c *Information) encode(w io.Writer) error {
	if err := writeString(w, c.Key); err != nil {
		return err
	}
	return writeString(w, c.Value)
}

type CAFStringsChunk struct {
	NumEntries uint32
	Strings    []Information
}

func (c *CAFStringsChunk) decode(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &c.NumEntries); err != nil {
		return err
	}
	for i := uint32(0); i < c.NumEntries; i++ {
		var info Information
		if err := info.decode(r); err != nil {
			return err
		}
		c.Strings = append(c.Strings, info)
	}
	return nil
}

func (c *CAFStringsChunk) encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, &c.NumEntries); err != nil {
		return err
	}
	for i := uint32(0); i < c.NumEntries; i++ {
		if err := c.Strings[i].encode(w); err != nil {
			return err
		}
	}
	return nil
}

type Chunk struct {
	Header   ChunkHeader
	Contents interface{}
}

func (c *AudioFormat) decode(r io.Reader) error {
	return binary.Read(r, binary.BigEndian, c)
}

func (c *AudioFormat) encode(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, c)
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

func (c *Data) decode(r *bufio.Reader, h ChunkHeader) error {
	if err := binary.Read(r, binary.BigEndian, &c.EditCount); err != nil {
		return err
	}
	if h.ChunkSize == -1 {
		// read until end
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		c.Data = data
	} else {
		dataLength := h.ChunkSize - 4 /* for edit count*/
		data, err := io.ReadAll(io.LimitReader(r, dataLength))
		if err != nil {
			return err
		}
		c.Data = data
	}
	return nil
}

func (c *Data) encode(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, &c.EditCount); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, &c.Data); err != nil {
		return err
	}
	return nil
}

func (c *Chunk) decode(r *bufio.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &c.Header); err != nil {
		return err
	}
	switch c.Header.ChunkType {
	case ChunkTypeAudioDescription:
		{
			var cc AudioFormat
			if err := cc.decode(r); err != nil {
				return err
			}
			c.Contents = &cc
			break
		}
	case ChunkTypeChannelLayout:
		{
			var cc ChannelLayout
			if err := cc.decode(r); err != nil {
				return err
			}
			c.Contents = &cc
			break
		}
	case ChunkTypeInformation:
		{
			var cc CAFStringsChunk
			if err := cc.decode(r); err != nil {
				return err
			}
			c.Contents = &cc
			break
		}
	case ChunkTypeAudioData:
		{
			var cc Data
			if err := cc.decode(r, c.Header); err != nil {
				return err
			}
			c.Contents = &cc
		}
	case ChunkTypePacketTable:
		{
			var cc PacketTable
			if err := cc.decode(r); err != nil {
				return err
			}
			c.Contents = &cc
		}
	case ChunkTypeMidi:
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
	case ChunkTypeAudioDescription:
		{
			cc := c.Contents.(*AudioFormat)
			if err := cc.encode(w); err != nil {
				return err
			}
			break
		}
	case ChunkTypeChannelLayout:
		{
			cc := c.Contents.(*ChannelLayout)
			if err := cc.encode(w); err != nil {
				return err
			}
			break
		}
	case ChunkTypeInformation:
		{
			cc := c.Contents.(*CAFStringsChunk)
			if err := cc.encode(w); err != nil {
				return err
			}
			break
		}
	case ChunkTypeAudioData:
		{
			cc := c.Contents.(*Data)
			if err := cc.encode(w); err != nil {
				return err
			}
			c.Contents = &cc
		}
	case ChunkTypePacketTable:
		{
			cc := c.Contents.(*PacketTable)
			if err := cc.encode(w); err != nil {
				return err
			}
			c.Contents = &cc
		}
	case ChunkTypeMidi:
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

func (h *FileHeader) Decode(r io.Reader) error {
	err := binary.Read(r, binary.BigEndian, h)
	if err != nil {
		return err
	}
	if h.FileType != stringToChunkType("caff") {
		return errors.New("invalid caff header")
	}
	return nil
}

func (h *FileHeader) Encode(w io.Writer) error {
	err := binary.Write(w, binary.BigEndian, h)
	if err != nil {
		return err
	}
	return nil
}

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

// NewWith returns a new Ogg reader and Ogg header with an io.Reader input
func NewWith(in io.Reader) (*OggReader, *OggHeader, error) {

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

func buildCafFile(header *OggHeader, audioData []byte, trailing_data []uint64, frame_size int) *File {
	len_audio := len(audioData)
	packets := len(trailing_data)
	frames := frame_size * packets

	packetTableLength := calculatePacketTableLength(trailing_data)

	cf := &File{}
	cf.FileHeader = FileHeader{FileType: FourByteString{99, 97, 102, 102}, FileVersion: 1, FileFlags: 0}

	c := Chunk{
		Header:   ChunkHeader{ChunkType: ChunkTypeAudioDescription, ChunkSize: 32},
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
		Header: ChunkHeader{ChunkType: ChunkTypeChannelLayout, ChunkSize: 12},
		Contents: &ChannelLayout{ChannelLayoutTag: channelLayoutTag, ChannelBitmap: 0x0, NumberChannelDescriptions: 0,
			Channels: []ChannelDescription{},
		},
	}

	cf.Chunks = append(cf.Chunks, c1)

	c2 := Chunk{
		Header:   ChunkHeader{ChunkType: ChunkTypeInformation, ChunkSize: 26},
		Contents: &CAFStringsChunk{NumEntries: 1, Strings: []Information{{Key: "encoder\x00", Value: "Lavf59.27.100\x00"}}},
	}

	cf.Chunks = append(cf.Chunks, c2)
	c3 := Chunk{
		Header:   ChunkHeader{ChunkType: ChunkTypeAudioData, ChunkSize: int64(len_audio + 4)},
		Contents: &Data{Data: audioData},
	}

	cf.Chunks = append(cf.Chunks, c3)

	c4 := Chunk{
		Header: ChunkHeader{ChunkType: ChunkTypePacketTable, ChunkSize: int64(packetTableLength)},
		Contents: &PacketTable{
			Header: PacketTableHeader{NumberPackets: int64(packets), NumberValidFrames: int64(frames), PrimingFramess: 0, RemainderFrames: 0},
			Entry:  trailing_data,
		},
	}

	cf.Chunks = append(cf.Chunks, c4)

	return cf
}

func ConvertOpusToCaf(inputFile string, outputFile string) {
	file, err := os.Open(inputFile)
	if err != nil {
		panic(err)
	}

	ogg, header, err := NewWith(file)
	if err != nil {
		panic(err)
	}

	audioData, trailing_data, frame_size := readOpusData(ogg)
	cf := buildCafFile(header, audioData, trailing_data, frame_size)

	outputBuffer := &bytes.Buffer{}
	if cf.Encode(outputBuffer) != nil {
		return
	}
	output := outputBuffer.Bytes()
	outfile, err := os.Create(outputFile)
	if err != nil {
		return
	}
	defer outfile.Close()
	_, err = outfile.Write(output)
	if err != nil {
		return
	}
}
