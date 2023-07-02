package caf

// CAF Encoding and Decoding
type FourByteString [4]byte

func NewFourByteStr(str string) FourByteString {
	if len(str) != 4 {
		panic("FourByteString must be 4 bytes")
	}
	res := FourByteString{}
	for i, v := range str {
		res[i] = byte(v)
	}
	return res
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

type PacketTableHeader struct {
	NumberPackets     int64
	NumberValidFrames int64
	PrimingFramess    int32
	RemainderFrames   int32
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
