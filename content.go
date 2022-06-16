package main

import (
	"errors"
	"io"
)

const contentCharset = "-ABCDEFGHIJKLMNOPQRSTUVWXYZ"

type contentReader struct {
	size int64
	cur  int64
}

// Read implements the io.Read interface.
func (d *contentReader) Read(p []byte) (n int, err error) {
	length := d.size - 1
	if d.cur >= length {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}

	if d.cur == 0 {
		p[n] = '|'
		d.cur++
		n++
	}

	for n < len(p) && d.cur <= length {

		p[n] = contentCharset[int(d.cur)%len(contentCharset)]
		d.cur++
		n++
	}

	if d.cur >= length {
		p[n-1] = '|'
	}

	return
}

// Seek implements the io.Seek interface.
func (d *contentReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, errors.New("Seek: invalid whence")
	case io.SeekStart:
		offset += 0
	case io.SeekCurrent:
		offset += d.cur
	case io.SeekEnd:
		offset += d.size - 1
	}
	if offset < 0 {
		return 0, errors.New("Seek: invalid offset")
	}
	d.cur = offset
	return offset, nil
}
