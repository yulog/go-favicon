// MIT License
//
// Copyright (c) 2024 yulog
//
// Copyright (c) 2020 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT
// Created on 2020-11-09

package favicon

import (
	"fmt"
	"io"
	"iter"
	urls "net/url"
	"path/filepath"
	"strings"

	gq "github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// entry point for URLs
func (p *parser) parseURL(url string) ([]*Icon, error) {
	u, err := urls.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	p.baseURL = u

	rc, err := p.find.fetchURL(url)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}
	defer rc.Close()

	doc, err := gq.NewDocumentFromReader(rc)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}
	return p.parse(doc)
}

// entry point for io.Reader
func (p *parser) parseReader(r io.Reader) ([]*Icon, error) {
	doc, err := gq.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}
	return p.parse(doc)
}

// entry point for html.Node
func (p *parser) parseNode(n *html.Node) ([]*Icon, error) {
	doc := gq.NewDocumentFromNode(n)
	return p.parse(doc)
}

// entry point for gq.Document
func (p *parser) parseGoQueryDocument(doc *gq.Document) ([]*Icon, error) {
	return p.parse(doc)
}

// main parser function
func (p *parser) parse(doc *gq.Document) ([]*Icon, error) {
	var (
		manifestURL = p.absURL("/manifest.json")
	)
	iconsIter := make([]func(func(*Icon) bool), 0, 7)

	// icons described in <link../> tags
	for _, sel := range doc.Find("link").EachIter() {
		rel, _ := sel.Attr("rel")
		rel = strings.ToLower(rel)
		switch rel {
		// all cases are handled the same way for now
		case "icon", "alternate icon", "shortcut icon":
			iconsIter = append(iconsIter, p.parseLinkIter(sel))
		case "apple-touch-icon", "apple-touch-icon-precomposed":
			iconsIter = append(iconsIter, p.parseLinkIter(sel))
		// site-specific browser apps (https://fluidapp.com/)
		case "fluid-icon":
			iconsIter = append(iconsIter, p.parseLinkIter(sel))
		case "manifest":
			url, _ := sel.Attr("href")
			url = p.absURL(url)
			if url != "" {
				manifestURL = url
			}
		}
	}

	// OpenGraph (og:) and Twitter <meta../> tags
	var (
		// k, v, k, v sequences
		opengraph []string
		twitter   []string
	)
	for _, sel := range doc.Find("meta").EachIter() {
		if s, ok := sel.Attr("charset"); ok && s != "" {
			p.charset = s
			continue
		}

		var (
			name, _ = sel.Attr("name")
			prop, _ = sel.Attr("property")
			val, _  = sel.Attr("content")
		)

		if prop == "" && name != "" {
			prop = name
		}

		if prop == "" || val == "" {
			continue
		}

		prop = strings.ToLower(prop)
		if strings.HasPrefix(prop, "og:image") {
			opengraph = append(opengraph, prop, val)
		}
		if strings.HasPrefix(prop, "twitter:image") {
			twitter = append(twitter, prop, val)
		}
	}

	// find icons in k, v sequences
	iconsIter = append(iconsIter, p.parseOpenGraphIter(opengraph), p.parseTwitterIter(twitter))

	// retrieve and parse JSON manifest
	if !p.find.ignoreManifest {
		iconsIter = append(iconsIter, p.parseManifestIter(manifestURL))
	}
	// check for existence of URLs like /favicon.ico
	if !p.find.ignoreWellKnown {
		iconsIter = append(iconsIter, p.findWellKnownIconsIter())
	}

	return p.postProcessIcons(concat(iconsIter...)), nil
}

// extract icons defined in <link../> tags
func (p *parser) parseLinkIter(sel *gq.Selection) iter.Seq[*Icon] {
	var (
		href, _ = sel.Attr("href")
		typ, _  = sel.Attr("type")
		size, _ = sel.Attr("sizes")
		sizes   = false
		icon    = &Icon{}
	)
	return func(yield func(*Icon) bool) {
		if href = p.absURL(href); href == "" {
			return
		}

		icon.URL = href
		// icon.FileExt = fileExt(href)
		if typ != "" {
			icon.MimeType = typ
		}
		if size != "" {
			for sz := range parseSizesIter(size) {
				i := icon.Copy()
				i.Width, i.Height = sz.w, sz.h
				sizes = true
				if !yield(i) {
					return
				}
			}
		}
		if !sizes { // no sizes understood
			yield(icon)
		}

		p.find.log.Printf("(link) %s", icon.URL)
	}
}

// extract file extension from a URL
func fileExt(url string) string {
	u, err := urls.Parse(url)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(filepath.Ext(u.Path), ".")
}
