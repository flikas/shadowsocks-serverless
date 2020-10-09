package main

import (
	"bufio"
	"encoding/binary"
	"io"
)

type Link struct {
	Writer *io.PipeWriter
	Reader *io.PipeReader
}

func makeLink() *Link {
	link := new(Link)
	link.Reader, link.Writer = io.Pipe()
	return link
}

func (link *Link) Read() ([]byte, error) {
	l, err := binary.ReadUvarint(bufio.NewReader(link.Reader))
	if err != nil {
		return nil, err
	}
	pkg := make([]byte, l)
	_, err = io.ReadFull(link.Reader, pkg)
	if err != nil {
		return nil, err
	}
	return pkg, nil
}

func (link *Link) Write(buf []byte) error {
	lenbuf := make([]byte, binary.MaxVarintLen64)
	l := binary.PutUvarint(lenbuf, uint64(len(buf)))
	_, err := link.Writer.Write(lenbuf[:l])
	if err != nil {
		return err
	}
	_, err = link.Writer.Write(buf)
	return err
}

func (link *Link) SetError(err error) {
	//TODO Implement this
}
