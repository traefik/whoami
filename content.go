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
	if r.cur >= r.size {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}

	var n int
	for n < len(p) && r.cur < r.size {
		if r.cur == 0 || r.cur == r.size-1 {
			p[n] = '|'
		} else {
			p[n] = contentCharset[int(r.cur)%len(contentCharset)]
		}
		r.cur++
		n++
	}

	return n, nil
}

// Seek implements the io.Seek interface.
func (r *contentReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, errors.New("seek: invalid whence")
	case io.SeekStart:
		// offset is unchanged
	case io.SeekCurrent:
		offset += r.cur
	case io.SeekEnd:
		offset += r.size
	}

	if offset < 0 {
		return 0, errors.New("seek: invalid offset")
	}

	r.cur = offset

	return offset, nil
}
