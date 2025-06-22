// MIT License
//
// Copyright (c) 2025 yulog
//
// Copyright (c) 2020 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT
// Created on 2020-11-09

package favicon

import (
	"encoding/json"
	"io"
	"iter"
	urls "net/url"
	"path/filepath"
	"regexp"
	"strconv"
)

// Manifest is the relevant parts of a manifest.json file.
type Manifest struct {
	Icons []ManifestIcon `json:"icons"`
}

// ManifestIcon is an icon from a manifest.json file.
type ManifestIcon struct {
	URL      string `json:"src"`
	Type     string `json:"type"`
	RawSizes string `json:"sizes"`
}

type size struct {
	w, h int
}

func (p *parser) parseManifestIter(url string) iter.Seq[*Icon] {
	return func(yield func(*Icon) bool) {
		p.find.log.Printf("loading manifest %q ...", url)
		rc, err := p.find.fetchURL(url)
		if err != nil {
			p.find.log.Printf("[ERROR] parse manifest: %v", err)
			return
		}
		defer rc.Close()

		for icon := range p.parseManifestReaderIter(rc) {
			yield(icon)
		}
	}
}

func (p *parser) parseManifestReaderIter(r io.Reader) iter.Seq[*Icon] {
	var (
		man = Manifest{}
	)

	if err := json.NewDecoder(r).Decode(&man); err != nil {
		p.find.log.Printf("[ERROR] parse manifest: %v", err)
	}
	return func(yield func(*Icon) bool) {
		for _, mi := range man.Icons {
			// TODO: make URL relative to manifest, not page
			mi.URL = p.absURL(mi.URL)
			p.find.log.Printf("(manifest) %s", mi.URL)
			for sz := range parseSizesIter(mi.RawSizes) {
				icon := &Icon{
					URL:    mi.URL,
					Width:  sz.w,
					Height: sz.h,
				}
				if !yield(icon) {
					return
				}
			}
		}
	}
}

var (
	rxSize  = regexp.MustCompile(`(\d+)x(\d+)`)
	rxWidth = regexp.MustCompile(`-(\d+)$`)
)

func parseSizesIter(s string) iter.Seq[size] {
	m := rxSize.FindAllStringSubmatch(s, -1)
	return func(yield func(size) bool) {
		if m == nil {
			return
		}
		for _, l := range m {
			for i := 1; i < len(l)-1; i += 2 {
				w, _ := strconv.ParseInt(l[i], 10, 32)
				h, _ := strconv.ParseInt(l[i+1], 10, 32)
				if !yield(size{w: int(w), h: int(h)}) {
					return
				}
			}
		}
	}
}

// find dimensions in URL
func extractSizeFromURL(url string) *size {
	// try to find WxH pattern
	for v := range parseSizesIter(url) {
		return &v
	}

	// look for -NNN at end of filename
	u, err := urls.Parse(url)
	if err != nil {
		return nil
	}

	var (
		name = filepath.Base(u.Path)
		ext  = filepath.Ext(name)
	)

	if m := rxWidth.FindStringSubmatch(name[:len(name)-len(ext)]); m != nil {
		n, _ := strconv.ParseInt(m[1], 10, 32)
		return &size{w: int(n), h: int(n)}
	}

	return nil
}
