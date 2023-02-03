package caf

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"os"

	"github.com/sirupsen/logrus"
)

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

type Information struct {
	Key   string
	Value string
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

type CAFStringsChunk struct {
	NumEntries uint32
	Strings    []Information
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

const DEFAULT_BUFFER_FOR_PLAYBACK_MS = 2500

// OpusReader is used to take an OGG file and write RTP packets
type OpusReader struct {
	stream                  io.Reader
	fd                      *os.File
	SampleRate              uint32
	Channels                uint8
	serial                  uint32
	pageIndex               uint32
	checksumTable           *crc32.Table
	previousGranulePosition uint64
	currentSampleLen        float32
	CurrentSampleLen        uint32
	CurrentFrames           uint32
	currentSamples          uint32
	currentSegment          uint8
	payloadLen              uint32
	segments                uint8
	currentSample           uint8
	segmentMap              map[uint8]uint8
}

type OpusSamples struct {
	Payload  []byte
	Frames   uint8
	Samples  uint32
	Duration uint32
}

// New builds a new OGG Opus reader
func NewFile(fileName string) (*OpusReader, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	reader := &OpusReader{}
	//  reader, err := NewWith(f, sampleRate, channelCount)
	//if err != nil {
	//        return nil, err
	//}
	reader.fd = f
	reader.segmentMap = make(map[uint8]uint8)
	reader.stream = bufio.NewReader(f)
	err = reader.getPage()
	if err != nil {
		return reader, err
	}
	err = reader.getPage()
	if err != nil {
		return reader, err
	}
	return reader, nil
}

func (i *OpusReader) readOpusHead() error {
	var version uint8
	magic := make([]byte, 8)
	if err := binary.Read(i.stream, binary.LittleEndian, &magic); err != err {
		return err
	}
	if bytes.Compare(magic, []byte("OpusHead")) != 0 {
		return errors.New("Wrong Opus Head")
	}

	if err := binary.Read(i.stream, binary.LittleEndian, &version); err != err {
		return err
	}
	if err := binary.Read(i.stream, binary.LittleEndian, &i.Channels); err != err {
		return err
	}
	var preSkip uint16
	if err := binary.Read(i.stream, binary.LittleEndian, &preSkip); err != err {
		return err
	}
	if err := binary.Read(i.stream, binary.LittleEndian, &i.SampleRate); err != err {
		return err
	}
	//Skipping OutputGain
	io.CopyN(io.Discard, i.stream, 2)
	var channelMap uint8
	if err := binary.Read(i.stream, binary.LittleEndian, &channelMap); err != err {
		return err
	}
	//if channelMap (Mapping family) is different than 0, next 4 bytes contain channel mapping configuration
	if channelMap != 0 {
		io.CopyN(io.Discard, i.stream, 4)
	}
	return nil
}

func (i *OpusReader) readOpusTags() (uint32, error) {
	var plen uint32
	var vendorLen uint32
	magic := make([]byte, 8)
	if err := binary.Read(i.stream, binary.LittleEndian, &magic); err != err {
		return 0, err
	}
	if bytes.Compare(magic, []byte("OpusTags")) != 0 {
		return 0, errors.New("Wrong Opus Tags")
	}

	if err := binary.Read(i.stream, binary.LittleEndian, &vendorLen); err != err {
		return 0, err
	}
	vendorName := make([]byte, vendorLen)
	if err := binary.Read(i.stream, binary.LittleEndian, &vendorName); err != err {
		return 0, err
	}

	var userCommentLen uint32
	if err := binary.Read(i.stream, binary.LittleEndian, &userCommentLen); err != err {
		return 0, err
	}
	userComment := make([]byte, userCommentLen)
	if err := binary.Read(i.stream, binary.LittleEndian, &userComment); err != err {
		return 0, err
	}
	plen = 16 + vendorLen + userCommentLen
	return plen, nil

}

func (i *OpusReader) getPageHead() error {
	head := make([]byte, 4)
	if err := binary.Read(i.stream, binary.LittleEndian, &head); err != err {
		return err
	}
	if bytes.Compare(head, []byte("OggS")) != 0 {
		return fmt.Errorf("Incorrect page. Does not start with \"OggS\" : %s %v", string(head), hex.EncodeToString(head))
	}
	//Skipping Version
	io.CopyN(io.Discard, i.stream, 1)
	var headerType uint8
	if err := binary.Read(i.stream, binary.LittleEndian, &headerType); err != err {
		return err
	}
	var granulePosition uint64
	if err := binary.Read(i.stream, binary.LittleEndian, &granulePosition); err != err {
		return err
	}
	if err := binary.Read(i.stream, binary.LittleEndian, &i.serial); err != err {
		return err
	}
	if err := binary.Read(i.stream, binary.LittleEndian, &i.pageIndex); err != err {
		return err
	}
	//skipping checksum
	io.CopyN(io.Discard, i.stream, 4)

	if err := binary.Read(i.stream, binary.LittleEndian, &i.segments); err != err {
		return err
	}
	var x uint8
	// building a map of all segments
	i.payloadLen = 0
	for x = 1; x <= i.segments; x++ {
		var segSize uint8
		if err := binary.Read(i.stream, binary.LittleEndian, &segSize); err != err {
			return err
		}
		i.segmentMap[x] = segSize
		i.payloadLen += uint32(segSize)
	}

	return nil
}

func (i *OpusReader) getPage() error {
	err := i.getPageHead()
	if err != nil {
		return err
	}
	if i.pageIndex == 0 {
		err := i.readOpusHead()
		if err != nil {
			return err
		}
	} else if i.pageIndex == 1 {
		plen, err := i.readOpusTags()
		if err != nil {
			return err
		}
		// we are not interested in tags (metadata?)
		io.CopyN(io.Discard, i.stream, int64(i.payloadLen-plen))

	} else {
		io.CopyN(io.Discard, i.stream, int64(i.payloadLen))
	}

	return nil
}

func (i *OpusReader) GetSample() (*OpusSamples, error) {
	opusSamples := new(OpusSamples)
	if i.currentSegment == 0 {

		err := i.getPageHead()
		if err != nil {
			return opusSamples, err

		}
	}
	var currentPacketSize uint32
	// Iteraring throug all segments to check if there are lacing packets. If a segment is 255 bytes long, it means that there must be a following segment for the same packet (which may be again 255 bytes long)
	for i.segmentMap[i.currentSegment] == 255 {
		currentPacketSize += 255
		i.currentSegment += 1

	}
	// Adding either the last segments of lacing ones or a packet that fits only in one segment
	currentPacketSize += uint32(i.segmentMap[i.currentSegment])
	if i.currentSegment < i.segments {
		i.currentSegment += 1
	} else {
		i.currentSegment = 0
	}
	tmpPacket := make([]byte, currentPacketSize)
	opusSamples.Payload = tmpPacket
	binary.Read(i.stream, binary.LittleEndian, &tmpPacket)
	//Reading the TOC byte - we need to know  the frame duration.
	if len(tmpPacket) > 0 {
		tmptoc := tmpPacket[0] & 255
		var frames uint8
		switch tmptoc & 3 {
		case 0:
			frames = 1
			break
		case 1:
		case 2:
			frames = 2
			break
		default:
			frames = tmpPacket[1] & 63
			break
		}
		opusSamples.Frames = frames
		tocConfig := tmptoc >> 3

		var length uint32
		length = uint32(tocConfig & 3)
		if tocConfig >= 16 {
			length = DEFAULT_BUFFER_FOR_PLAYBACK_MS << length
		} else if tocConfig >= 12 {
			length = 10000 << (length & 1)
		} else if length == 3 {
			length = 60000
		} else {
			length = 10000 << length
		}
		opusSamples.Duration = length
		opusSamples.Samples = (i.SampleRate * length) / 1000000
	}
	return opusSamples, nil
}

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
	errChecksumMismatch          = errors.New("expected and actual checksum do not match")
)

// OggReader is used to read Ogg files and return page payloads
type OggReader struct {
	stream               io.Reader
	bytesReadSuccesfully int64
	checksumTable        *[256]uint32
	doChecksum           bool
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

// NewWith returns a new Ogg reader and Ogg header
// with an io.Reader input
func NewWith(in io.Reader) (*OggReader, *OggHeader, error) {
	return newWith(in /* doChecksum */, true)
}

func newWith(in io.Reader, doChecksum bool) (*OggReader, *OggHeader, error) {
	if in == nil {
		return nil, nil, errNilStream
	}

	reader := &OggReader{
		stream:        in,
		checksumTable: generateChecksumTable(),
		doChecksum:    doChecksum,
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

	segments := [][]byte{}

	for _, s := range newArr {
		segment := make([]byte, int(s))
		if _, err = io.ReadFull(o.stream, segment); err != nil {
			return nil, nil, err
		}

		segments = append(segments, segment)
	}

	return segments, pageHeader, nil
}

// ResetReader resets the internal stream of OggReader. This is useful
// for live streams, where the end of the file might be read without the
// data being finished.
func (o *OggReader) ResetReader(reset func(bytesRead int64) io.Reader) {
	o.stream = reset(o.bytesReadSuccesfully)
}

func generateChecksumTable() *[256]uint32 {
	var table [256]uint32
	const poly = 0x04c11db7

	for i := range table {
		r := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if (r & 0x80000000) != 0 {
				r = (r << 1) ^ poly
			} else {
				r <<= 1
			}
			table[i] = (r & 0xffffffff)
		}
	}
	return &table
}

func ConvertOpusToCaf(i string, o string) {
	file, err := os.Open(i)
	if err != nil {
		panic(err)
	}

	ogg, header, err := NewWith(file)
	if err != nil {
		panic(err)
	}

	audioData := []byte{}
	frame_size := 0
	trailing_data := make([]uint64, 0)
	packetTableLength := 24

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

			// packets issues
			audioData = append(audioData, segments[i]...)
		}

		if header.index == 2 {
			tmpPacket := segments[0]
			if len(tmpPacket) > 0 {
				tmptoc := tmpPacket[0] & 255

				tocConfig := tmptoc >> 3

				length := uint32(tocConfig & 3)

				if tocConfig < 12 {
					frame_size = int(math.Max(480, float64(960*length)))
				} else if tocConfig < 16 {
					frame_size = 480 << (tocConfig & 1)
				} else {
					frame_size = 120 << (tocConfig & 3)
				}
			}
		}
	}
	len_audio := len(audioData)
	packets := len(trailing_data)
	frames := frame_size * packets

	// Check how much chunk size is needed each segment by BER encoding
	for i := 0; i < packets; i++ {
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

	outputBuffer := &bytes.Buffer{}
	if cf.Encode(outputBuffer) != nil {
		return
	}
	output := outputBuffer.Bytes()
	outfile, err := os.Create(o)
	if err != nil {
		return
	}
	defer outfile.Close()
	_, err = outfile.Write(output)
	if err != nil {
		return
	}
}
