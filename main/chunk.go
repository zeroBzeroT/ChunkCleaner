package main

// Big credits to https://github.com/Tnze/go-mc/blob/fe34dbd8bb6a93e3c9dfd33670fd7bc350915860/save/chunk.go and hub for fixing it

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"io"

	"github.com/Tnze/go-mc/nbt"
)

type Chunk struct {
	Level struct {
		InhabitedTime int64
	}
	XPos int32 `nbt:"xPos"`
	YPos int32 `nbt:"yPos"`
	ZPos int32 `nbt:"zPos"`
}

func (c *Chunk) Load(data []byte) (err error) {
	var reader io.Reader = bytes.NewReader(data[1:])

	switch data[0] {
	case 1:
		reader, err = gzip.NewReader(reader)
	case 2:
		reader, err = zlib.NewReader(reader)
	case 3:
		// no compression
	default:
		err = errors.New("unknown compression")
	}
	if err != nil {
		return err
	}

	_, err = nbt.NewDecoder(reader).Decode(&c)
	if err != nil {
		return err
	}
	return
}
