// Copyright (c) 2020 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT
// Created on 2020-11-10

package favicon

import (
	"net/http"
	"net/http/httptest"
	urls "net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParserAbsURL tests resolution of URLs.
func TestParserAbsURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		in, x string
		base  *urls.URL
	}{
		{"empty", "", "", nil},
		{"onlyBaseURL", "", "", mustURL("https://github.com")},
		{"noBaseURL", "/root", "/root", nil},
		{"baseURL", "/root", "https://github.com/root", mustURL("https://github.com")},
		{"absURL", "https://github.com/root", "https://github.com/root", mustURL("https://github.com")},
		// absolute URLs returned as-is
		{"absURLDifferentBase", "https://github.com/root", "https://github.com/root", mustURL("https://google.com")},
		{"absURLNoBase", "https://github.com/root", "https://github.com/root", nil},
	}

	for _, td := range tests {
		td := td
		t.Run(td.name, func(t *testing.T) {
			p := parser{baseURL: td.base}
			v := p.absURL(td.in)
			assert.Equal(t, td.x, v, "unexpected URL")
		})
	}
}

// TestParseSize tests the extraction and parsing of image sizes.
func TestParseSize(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		i             int
		width, height int
		square        bool
	}{
		// size read from manifest & markup
		{"kuli-0", "./testdata/kuli", 0, 512, 512, true}, // manifest
		{"kuli-1", "./testdata/kuli", 1, 400, 400, true}, // markup
		{"kuli-2", "./testdata/kuli", 2, 192, 192, true}, // manifest
		{"kuli-3", "./testdata/kuli", 3, 180, 180, true}, // markup

		// size read from manifest
		{"manifest-only-0", "./testdata/manifest-only", 0, 512, 512, true},
		{"manifest-only-1", "./testdata/manifest-only", 1, 192, 192, true},

		// size parsed from WxH in URL
		{"mozilla-0", "./testdata/mozilla", 0, 196, 196, true},
		{"mozilla-1", "./testdata/mozilla", 1, 180, 180, true},

		// size parsed from <link>
		{"multisize-0", "./testdata/multisize", 0, 48, 48, true},
		{"multisize-1", "./testdata/multisize", 1, 24, 24, true},
		{"multisize-2", "./testdata/multisize", 2, 16, 16, true},
	}

	for _, td := range tests {
		td := td
		t.Run(td.name, func(t *testing.T) {
			t.Parallel()
			ts := httptest.NewServer(http.FileServer(http.Dir(td.path)))
			defer ts.Close()

			f := New(WithLogger(debugLogger{}))
			icons, err := f.Find(ts.URL + "/index.html")
			require.Nil(t, err, "unexpected error")
			require.Greater(t, len(icons), td.i, "too few icons found")
			icon := icons[td.i]
			assert.Equal(t, td.width, icon.Width, "unexpected width")
			assert.Equal(t, td.height, icon.Height, "unexpected height")
			assert.Equal(t, td.square, icon.IsSquare(), "unexpected square")
		})
	}
}

func mustURL(s string) *urls.URL {
	u, err := urls.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
