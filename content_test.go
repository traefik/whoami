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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &contentReader{
				size: tt.size,
			}

			b, err := io.ReadAll(d)
			if err != nil {
				t.Errorf("contentReader.Read() error = %v", err)
				return
			}

			if string(b) != tt.content {
				t.Errorf("return content does not match expected value: %s vs %s", tt.content, b)
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &contentReader{
				size: tt.size,
			}
			_, err := d.Seek(tt.offset, tt.seekWhence)
			if err != nil {
				t.Errorf("contentReader.Seek() error = %v", err)
				return
			}

			b, err := io.ReadAll(d)
			if err != nil {
				t.Errorf("contentReader.Read() error = %v", err)
				return
			}

			if string(b) != tt.content {
				t.Errorf("return content does not match expected value: %s vs %s", tt.content, b)
			}
		})
	}
}
