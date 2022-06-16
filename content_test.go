package main

import (
	"io"
	"testing"
)

func Test_contentReader_Read(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		content string
	}{
		{
			name:    "simple",
			size:    40,
			content: "|ABCDEFGHIJKLMNOPQRSTUVWXYZ-ABCDEFGHIJK|",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			d := &contentReader{size: test.size}

			b, err := io.ReadAll(d)
			if err != nil {
				t.Errorf("contentReader.Read() error = %v", err)
				return
			}

			if string(b) != test.content {
				t.Errorf("return content does not match expected value: got %s want %s", b, test.content)
			}
		})
	}
}

func Test_contentReader_ReadSeek(t *testing.T) {
	tests := []struct {
		name       string
		offset     int64
		seekWhence int
		size       int64
		content    string
	}{
		{
			name:       "simple",
			offset:     10,
			seekWhence: io.SeekCurrent,
			size:       40,
			content:    "JKLMNOPQRSTUVWXYZ-ABCDEFGHIJK|",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			d := &contentReader{size: test.size}

			_, err := d.Seek(test.offset, test.seekWhence)
			if err != nil {
				t.Errorf("contentReader.Seek() error = %v", err)
				return
			}

			b, err := io.ReadAll(d)
			if err != nil {
				t.Errorf("contentReader.Read() error = %v", err)
				return
			}

			if string(b) != test.content {
				t.Errorf("return content does not match expected value: got %s want %s", b, test.content)
			}
		})
	}
}
