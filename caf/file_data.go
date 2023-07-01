package caf

import (
	"bufio"
	"io"
)

type FileData struct {
	FileHeader FileHeader
	Chunks     []Chunk
}

func (cf *FileData) Decode(r io.Reader) error {
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

func (cf *FileData) Encode(w io.Writer) error {
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
