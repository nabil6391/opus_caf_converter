package caf

import (
	"bufio"
	"io"
)

type CAFFileData struct {
	CAFFileHeader CAFFileHeader
	Chunks        []CAFChunk
}

func (cf *CAFFileData) Decode(r io.Reader) error {
	bufferedReader := bufio.NewReader(r)
	var fileHeader CAFFileHeader
	if err := fileHeader.Decode(bufferedReader); err != nil {
		return err
	}
	cf.CAFFileHeader = fileHeader
	for {
		var c CAFChunk
		if err := c.decode(bufferedReader); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		cf.Chunks = append(cf.Chunks, c)
	}
	return nil
}

func (cf *CAFFileData) Encode(w io.Writer) error {
	if err := cf.CAFFileHeader.Encode(w); err != nil {
		return err
	}
	for _, c := range cf.Chunks {
		if err := c.Encode(w); err != nil {
			return err
		}
	}
	return nil
}
