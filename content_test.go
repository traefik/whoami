package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_contentReader_Read(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		content string
	}{
		{
			name:    "empty",
			size:    0,
			content: "",
		},
		{
			name:    "size 1",
			size:    1,
			content: "|",
		},
		{
			name:    "size 2",
			size:    2,
			content: "||",
		},
		{
			name:    "size 3",
			size:    3,
			content: "|A|",
		},
		{
			name:    "simple",
			size:    40,
			content: "|ABCDEFGHIJKLMNOPQRSTUVWXYZ-ABCDEFGHIJK|",
		},
	}

	for _, test := range tests {
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

func Test_contentReader_Read_Chunked(t *testing.T) {
	d := &contentReader{size: 40}
	expected := "|ABCDEFGHIJKLMNOPQRSTUVWXYZ-ABCDEFGHIJK|"

	var buf bytes.Buffer
	p := make([]byte, 7) // arbitrary chunk size
	for {
		n, err := d.Read(p)
		if n > 0 {
			buf.Write(p[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if buf.String() != expected {
		t.Errorf("chunked read result mismatch: got %s want %s", buf.String(), expected)
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
			name:       "seek current",
			offset:     10,
			seekWhence: io.SeekCurrent,
			size:       40,
			content:    "JKLMNOPQRSTUVWXYZ-ABCDEFGHIJK|",
		},
		{
			name:       "seek start to last byte",
			offset:     39,
			seekWhence: io.SeekStart,
			size:       40,
			content:    "|",
		},
		{
			name:       "seek end offset 0 is EOF",
			offset:     0,
			seekWhence: io.SeekEnd,
			size:       40,
			content:    "",
		},
		{
			name:       "seek end offset -1 is last byte",
			offset:     -1,
			seekWhence: io.SeekEnd,
			size:       40,
			content:    "|",
		},
		{
			name:       "seek end offset -5 is last 5 bytes",
			offset:     -5,
			seekWhence: io.SeekEnd,
			size:       40,
			content:    "HIJK|",
		},
	}

	for _, test := range tests {
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

func Test_contentReader_ServeContent(t *testing.T) {
	const totalSize = 40
	reader := &contentReader{size: totalSize}

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	rr := httptest.NewRecorder()

	http.ServeContent(rr, req, "data.txt", time.Now(), reader)

	resp := rr.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if resp.ContentLength != totalSize {
		t.Errorf("expected Content-Length %d, got %d", totalSize, resp.ContentLength)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed reading body: %v", err)
	}

	expected := "|ABCDEFGHIJKLMNOPQRSTUVWXYZ-ABCDEFGHIJK|"
	if string(body) != expected {
		t.Errorf("expected body %s, got %s", expected, string(body))
	}
}

func Test_dataHandler(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		rangeHeader    string
		expectedStatus int
		expectedBody   string
		expectedLength int64
	}{
		{
			name:           "default size 1",
			url:            "/data",
			expectedStatus: http.StatusOK,
			expectedBody:   "|",
			expectedLength: 1,
		},
		{
			name:           "explicit size 0",
			url:            "/data?size=0",
			expectedStatus: http.StatusOK,
			expectedBody:   "",
			expectedLength: 0,
		},
		{
			name:           "attachment size 40",
			url:            "/data?size=40&attachment=true",
			expectedStatus: http.StatusOK,
			expectedBody:   "|ABCDEFGHIJKLMNOPQRSTUVWXYZ-ABCDEFGHIJK|",
			expectedLength: 40,
		},
		{
			name:           "attachment size 40 range last 5 bytes",
			url:            "/data?size=40&attachment=true",
			rangeHeader:    "bytes=-5",
			expectedStatus: http.StatusPartialContent,
			expectedBody:   "HIJK|",
			expectedLength: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			if tc.rangeHeader != "" {
				req.Header.Set("Range", tc.rangeHeader)
			}
			rr := httptest.NewRecorder()

			dataHandler(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed reading body: %v", err)
			}

			if string(body) != tc.expectedBody {
				t.Errorf("expected body %q, got %q", tc.expectedBody, string(body))
			}

			if int64(len(body)) != tc.expectedLength {
				t.Errorf("expected body length %d, got %d", tc.expectedLength, len(body))
			}
		})
	}
}
