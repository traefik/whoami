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
func (r *contentReader) Read(p []byte) (int, error) {
	length := r.size - 1

	if r.cur >= length {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}

	var n int
	if r.cur == 0 {
		p[n] = '|'
		r.cur++
		n++
	}

	for n < len(p) && r.cur <= length {
		p[n] = contentCharset[int(r.cur)%len(contentCharset)]
		r.cur++
		n++
	}

	if r.cur >= length {
		p[n-1] = '|'
	}

	return n, nil
}

// Seek implements the io.Seek interface.
func (r *contentReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, errors.New("seek: invalid whence")
	case io.SeekStart:
		offset += 0
	case io.SeekCurrent:
		offset += r.cur
	case io.SeekEnd:
		offset += r.size - 1
	}

	if offset < 0 {
		return 0, errors.New("seek: invalid offset")
	}

	r.cur = offset

	return offset, nil
}
