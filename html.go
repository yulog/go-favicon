// Copyright (c) 2020 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT
// Created on 2020-11-09

package favicon

import (
	"io"
	"net/url"
	"path/filepath"
	"strings"

	gq "github.com/PuerkitoBio/goquery"
	"github.com/friendsofgo/errors"
)

func (p *parser) parseURL(u string) (Icons, error) {
	URL, err := url.Parse(u)
	if err != nil {
		return nil, errors.Wrap(err, "invalid URL")
	}
	p.baseURL = URL

	rc, err := p.f.fetchURL(u)
	if err != nil {
		return nil, errors.Wrap(err, "fetch page")
	}

	doc, err := gq.NewDocumentFromReader(rc)
	if err != nil {
		return nil, errors.Wrap(err, "parse HTML")
	}
	return p.parse(doc)
}

func (p *parser) parseReader(r io.Reader) (Icons, error) {
	doc, err := gq.NewDocumentFromReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "parse HTML")
	}
	return p.parse(doc)
}

// main parser function
func (p *parser) parse(doc *gq.Document) (Icons, error) {
	var (
		icons       Icons
		manifestURL = p.absURL("/manifest.json")
	)
	doc.Find("link").Each(func(i int, sel *gq.Selection) {
		rel, _ := sel.Attr("rel")
		rel = strings.ToLower(rel)
		switch rel {
		case "icon", "alternate icon", "shortcut icon":
			icons = append(icons, p.parseLink(sel)...)
		case "apple-touch-icon", "apple-touch-icon-precomposed":
			icons = append(icons, p.parseLink(sel)...)
		// for site-specific browser apps
		// https://fluidapp.com/
		case "fluid-icon":
			icons = append(icons, p.parseLink(sel)...)
		case "manifest":
			u, _ := sel.Attr("href")
			u = p.absURL(u)
			if u != "" {
				manifestURL = u
			}
		}
	})

	var (
		opengraph []string
		twitter   []string
	)
	doc.Find("meta").Each(func(i int, sel *gq.Selection) {
		if s, ok := sel.Attr("charset"); ok && s != "" {
			p.charset = s
			return
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
			return
		}

		prop = strings.ToLower(prop)
		if strings.HasPrefix(prop, "og:image") {
			opengraph = append(opengraph, prop, val)
		}
		if strings.HasPrefix(prop, "twitter:image") {
			twitter = append(twitter, prop, val)
		}
	})

	icons = append(icons, p.parseOpenGraph(opengraph)...)
	icons = append(icons, p.parseTwitter(twitter)...)
	if !p.f.ignoreManifest {
		icons = append(icons, p.parseManifest(manifestURL)...)
	}
	if !p.f.ignoreWellKnown {
		icons = append(icons, p.findWellKnownIcons()...)
	}

	icons = p.postProcessIcons(icons)

	return icons, nil
}

func (p *parser) parseLink(sel *gq.Selection) []Icon {
	var (
		href, _ = sel.Attr("href")
		typ, _  = sel.Attr("type")
		size, _ = sel.Attr("sizes")
		icons   []Icon
		icon    Icon
	)

	if href = p.absURL(href); href == "" {
		return nil
	}

	icon.URL = href
	// icon.FileExt = fileExt(href)
	if typ != "" {
		icon.MimeType = typ
	}
	if size != "" {
		for _, sz := range parseSizes(size) {
			i := icon.Copy()
			i.Width, i.Height = sz.w, sz.h
			icons = append(icons, i)
		}
	}
	if len(icons) == 0 { // no sizes understood
		icons = append(icons, icon)
	}

	p.f.log.Printf("(link) %s", icon.URL)
	return icons
}

// extract file extension from a URL.
func fileExt(u string) string {
	p, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(filepath.Ext(p.Path), ".")
}
