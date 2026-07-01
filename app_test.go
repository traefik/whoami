package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"
)

func TestApp_addHeaderFilter(t *testing.T) {
	tests := []struct {
		desc     string
		params   map[string][]string
		expected http.Header
	}{
		{
			desc: "single header",
			params: map[string][]string{
				"Content-Type": {"application/json"},
			},
			expected: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			desc: "multiple headers",
			params: map[string][]string{
				"Content-Type":  {"application/json"},
				"Cache-Control": {"no-store"},
			},
			expected: http.Header{
				"Content-Type":  []string{"application/json"},
				"Cache-Control": []string{"no-store"},
			},
		},
		{
			desc: "duplicate headers",
			params: map[string][]string{
				"Content-Type":  {"application/json"},
				"Cache-Control": {"no-store", "no-cache"},
			},
			expected: http.Header{
				"Content-Type":  []string{"application/json"},
				"Cache-Control": []string{"no-store", "no-cache"},
			},
		},
		{
			desc: "special encoded headers",
			params: map[string][]string{
				"X-Custom-Field": {"equal=one"},
			},
			expected: http.Header{
				"X-Custom-Field": []string{"equal=one"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			uri, err := url.Parse("/random-path")
			if err != nil {
				t.Fatal(err)
			}

			query := uri.Query()

			for k, values := range test.params {
				for _, v := range values {
					query.Add("header", fmt.Sprintf("%s=%s", k, v))
				}
			}

			uri.RawQuery = query.Encode()

			req := httptest.NewRequest(http.MethodGet, uri.String(), http.NoBody)
			w := httptest.NewRecorder()
			addHeaderFilter(w, req)

			for k, v := range w.Header() {
				if slices.Compare(test.expected.Values(k), v) != 0 {
					t.Errorf("header %s doesn't match: got %v, expected %v", k, v, test.expected.Values(k))
				}
			}
		})
	}
}
