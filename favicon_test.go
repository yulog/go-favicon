// Copyright (c) 2020 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT
// Created on 2020-11-09

package favicon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var verboseTest bool

func init() {
	if s := os.Getenv("TEST_VERBOSE"); s == "true" || s == "1" {
		verboseTest = true
	}
}

type debugLogger struct{}

func (l debugLogger) Printf(format string, v ...interface{}) {
	if verboseTest {
		fmt.Printf(format+"\n", v...)
	}
}

// TestBaseURL verifies absolute links.
func TestBaseURL(t *testing.T) {
	t.Parallel()
	file, err := os.Open("testdata/kuli/index.html")
	require.Nil(t, err, "unexpected error")
	defer file.Close()

	f := New(
		WithLogger(debugLogger{}),
		OnlyMimeType("image/x-icon"),
		IgnoreWellKnown,
		IgnoreManifest,
	)
	require.Nil(t, err, "unexpected error")

	var (
		baseURL = "https://www.kulturliste-duesseldorf.de"
		x       = "https://www.kulturliste-duesseldorf.de/favicon-rot.ico"
		icons   []*Icon
	)
	icons, err = f.FindReader(file, baseURL)
	require.Nil(t, err, "unexpected error")
	// for _, i := range icons {
	// 	fmt.Println(i)
	// }
	assert.Equal(t, 1, len(icons), "unexpected favicon count")
	assert.Equal(t, x, icons[0].URL, "unexpected favicon URL")
}

// TestFindHTML parses HTML only.
func TestFindHTML(t *testing.T) {
	t.Parallel()
	file, err := os.Open("testdata/github/index.html")
	require.Nil(t, err, "unexpected error")
	defer file.Close()

	f := New(WithLogger(debugLogger{}))
	require.Nil(t, err, "unexpected error")

	var icons []*Icon
	icons, err = f.FindReader(file)
	require.Nil(t, err, "unexpected error")
	assert.Equal(t, 6, len(icons), "unexpected favicon count")
}

// TestFindManifest finds favicons in manifest.
func TestFindManifest(t *testing.T) {
	t.Parallel()
	file, err := os.Open("testdata/github/manifest.json")
	require.Nil(t, err, "unexpected error")
	defer file.Close()

	f := New(WithLogger(debugLogger{}))
	require.Nil(t, err, "unexpected error")
	p := f.newParser()
	p.baseURL = mustURL("https://github.com")

	icons := p.parseManifestReader(file)
	assert.Equal(t, 11, len(icons), "unexpected favicon count")
}

// TestHTTP tests fetching via HTTP.
func TestHTTP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, path string
		xcount     int
	}{
		{"github", "./testdata/github", 17},
		{"kuli", "./testdata/kuli", 7},
		{"mozilla", "./testdata/mozilla", 4},
		{"no-markup", "./testdata/no-markup", 3},
	}

	for _, td := range tests {
		td := td
		t.Run(td.name, func(t *testing.T) {
			t.Parallel()
			ts := httptest.NewServer(http.FileServer(http.Dir(td.path)))
			defer ts.Close()

			f := New(WithClient(ts.Client()), WithLogger(debugLogger{}))
			icons, err := f.Find(ts.URL + "/index.html")
			require.Nil(t, err, "unexpected error")
			assert.Equal(t, td.xcount, len(icons), "unexpected favicon count")
		})
	}
}

// TestIgnore verifies Ignore* Options.
func TestIgnore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, path      string
		ignoreWellKnown bool
		ignoreManifest  bool
		xcount          int
	}{
		// ignore well-known
		{"github-ignore-well-known", "./testdata/github", true, false, 17},
		{"kuli-ignore-well-known", "./testdata/kuli", true, false, 7},
		{"mozilla-ignore-well-known", "./testdata/mozilla", true, false, 4},
		{"no-markup-ignore-well-known", "./testdata/no-markup", true, false, 2},
		{"manifest-only-ignore-well-known", "./testdata/manifest-only", true, false, 2},

		// ignore manifest
		{"no-markup-ignore-manifest", "./testdata/no-markup", false, true, 1},
		{"manifest-only-ignore-manifest", "./testdata/manifest-only", false, true, 0},

		// ignore well-known & manifest
		{"github-ignore-both", "./testdata/github", true, true, 6},
		{"kuli-ignore-both", "./testdata/kuli", true, true, 5},
		{"mozilla-ignore-both", "./testdata/mozilla", true, true, 4},
		{"no-markup-ignore-both", "./testdata/no-markup", true, true, 0},
		{"manifest-only-both", "./testdata/manifest-only", true, true, 0},
	}

	for _, td := range tests {
		td := td
		t.Run(td.name, func(t *testing.T) {
			t.Parallel()
			ts := httptest.NewServer(http.FileServer(http.Dir(td.path)))
			defer ts.Close()

			opts := []Option{
				WithClient(ts.Client()),
				WithLogger(debugLogger{}),
			}

			if td.ignoreWellKnown {
				opts = append(opts, IgnoreWellKnown)
			}
			if td.ignoreManifest {
				opts = append(opts, IgnoreManifest)
			}

			f := New(opts...)
			icons, err := f.Find(ts.URL + "/index.html")
			require.Nil(t, err, "unexpected error")
			assert.Equal(t, td.xcount, len(icons), "unexpected favicon count")
		})
	}
}

// TestFilter verifies filtering Options.
func TestFilter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, path string
		opts       []Option
		xcount     int
	}{
		{"no-options", "./testdata/multiformat", []Option{}, 9},
		{"only-square", "./testdata/multiformat", []Option{OnlySquare}, 6},
		{"ignore-nosize", "./testdata/multiformat", []Option{IgnoreNoSize}, 8},
		{"only-ico", "./testdata/multiformat", []Option{OnlyICO}, 1},
		{"only-png", "./testdata/multiformat", []Option{OnlyPNG}, 7},
		{"only-square-png", "./testdata/multiformat", []Option{OnlyPNG, OnlySquare}, 4},
		{"only-jpeg", "./testdata/multiformat", []Option{OnlyMimeType("image/jpeg")}, 1},
		{"only-square-sized", "./testdata/multiformat", []Option{OnlySquare, IgnoreNoSize}, 5},
		{"only-400", "./testdata/multiformat", []Option{MinWidth(400), MaxWidth(400)}, 1},
		{"width-100-and-200", "./testdata/multiformat", []Option{MinWidth(100), MaxWidth(200)}, 4},
		{"width+height-100-and-200", "./testdata/multiformat", []Option{MinWidth(100), MaxWidth(200), MinHeight(100), MaxHeight(200)}, 2},
	}

	for _, td := range tests {
		td := td
		t.Run(td.name, func(t *testing.T) {
			t.Parallel()
			ts := httptest.NewServer(http.FileServer(http.Dir(td.path)))
			defer ts.Close()

			opts := []Option{
				WithClient(ts.Client()),
				WithLogger(debugLogger{}),
			}
			opts = append(opts, td.opts...)
			f := New(opts...)
			icons, err := f.Find(ts.URL + "/index.html")
			require.Nil(t, err, "unexpected error")
			for _, i := range icons {
				fmt.Println(i)
			}
			assert.Equal(t, td.xcount, len(icons), "unexpected favicon count")
		})
	}
}
