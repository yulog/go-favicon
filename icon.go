// Copyright (c) 2020 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT
// Created on 2020-11-09

package favicon

import (
	"crypto/sha256"
	"fmt"
	"sort"
)

// Icon is a favicon parsed from an HTML file or JSON manifest.
//
// TODO: Use *Icon everywhere to be consistent with higher-level APIs that return nil for "not found".
type Icon struct {
	URL      string `json:"url"`       // Never empty
	MimeType string `json:"mimetype"`  // MIME type of icon; never empty
	FileExt  string `json:"extension"` // File extension; may be empty
	// Dimensions are extracted from markup/manifest, falling back to
	// searching for numbers in the URL.
	Width  int `json:"width"`
	Height int `json:"height"`
	// Hash of URL and dimensions to uniquely identify icon.
	Hash string `json:"hash"`
}

// IsSquare returns true if image has equally-long sides.
func (i Icon) IsSquare() bool { return i.Width == i.Height }

// Copy returns a new Icon with the same values as this one.
func (i Icon) Copy() Icon {
	return Icon{
		URL:      i.URL,
		MimeType: i.MimeType,
		FileExt:  i.FileExt,
		Width:    i.Width,
		Height:   i.Height,
		Hash:     i.Hash,
	}
}

// ByWidth sorts icons by width (largest first), and then by image type
// (PNG > JPEG > SVG > ICO).
type ByWidth []Icon

// Implement sort.Interface
func (v ByWidth) Len() int      { return len(v) }
func (v ByWidth) Swap(i, j int) { v[i], v[j] = v[j], v[i] }

// used for sorting icons
// higher number = higher priority
var formatRank = map[string]int{
	"image/png":                10,
	"image/jpeg":               9,
	"image/svg":                8,
	"image/x-icon":             7, // .ico
	"image/vnd.microsoft.icon": 7, // .ico
}

func (v ByWidth) Less(i, j int) bool {
	a, b := v[i], v[j]
	if a.Width != b.Width {
		return a.Width > b.Width
	}
	fa, fb := formatRank[a.MimeType], formatRank[b.MimeType]
	if fa != fb {
		return fa > fb
	}
	return a.URL < b.URL
}

// Check missing values, remove duplicates, sort.
func (p *parser) postProcessIcons(icons ByWidth) ByWidth {
	tidied := map[string]Icon{}
	for _, icon := range icons {
		icon.URL = p.absURL(icon.URL)

		if icon.MimeType == "" {
			icon.MimeType = mimeTypeURL(icon.URL)
		}

		if icon.URL == "" || icon.MimeType == "" {
			continue
		}

		if icon.FileExt == "" {
			icon.FileExt = fileExt(icon.URL)
		}

		if icon.Width == 0 {
			if sz := extractSizeFromURL(icon.URL); sz != nil {
				icon.Width, icon.Height = sz.w, sz.h
			}
		}
		icon.Hash = iconHash(icon)
		tidied[icon.Hash] = icon
	}

	icons = make([]Icon, len(tidied))

	var i int
	for _, icon := range tidied {
		icons[i] = icon
		i++
	}

	sort.Sort(icons)
	return icons
}

// returns a hash of icon's URL and size.
func iconHash(i Icon) string {
	s := fmt.Sprintf("%s-%dx%d", i.URL, i.Width, i.Height)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}
