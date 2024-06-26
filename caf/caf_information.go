package caf

import "io"

type Information struct {
	Key   string
	Value string
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
