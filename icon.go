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
type Icon struct {
	URL    string `json:"url"`    // Never empty
	Format string `json:"format"` // MIME type of icon; never empty
	// Dimensions are extracted from markup/manifest, falling back to
	// searching for numbers in the URL.
	Width  int `json:"width"`
	Height int `json:"height"`
	// Hash of URL and dimensions to uniquely identify icon.
	Hash string `json:"hash"`
}

// IsSquare returns true if image has equally-long sides.
func (i Icon) IsSquare() bool { return i.Width == i.Height }

// Icons is a collection of icons for a URL.
type Icons []Icon

// Implement sort.Interface
func (v Icons) Len() int           { return len(v) }
func (v Icons) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v Icons) Less(i, j int) bool { return v[i].Width < v[j].Width }

// Check missing values, remove duplicates, sort.
func (p *parser) postProcessIcons(icons Icons) Icons {
	tidied := map[string]Icon{}
	for _, icon := range icons {
		icon.URL = p.absURL(icon.URL)

		if icon.Format == "" {
			icon.Format = mimeTypeURL(icon.URL)
		}

		if icon.URL == "" || icon.Format == "" {
			continue
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

	sort.Sort(sort.Reverse(icons))
	return icons
}

// returns a hash of icon's URL and size.
func iconHash(i Icon) string {
	s := fmt.Sprintf("%s-%dx%d", i.URL, i.Width, i.Height)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}
