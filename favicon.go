// MIT License
//
// Copyright (c) 2024 yulog
//
// Copyright (c) 2020 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT
// Created on 2020-11-09

// Package favicon finds icons for websites. It can find icons in HTML (favicons
// in <link> elements, Open Graph or Twitter images) and in JSON manifests, or
// check common paths on the server (e.g. /favicon.ico).
//
// Package-level functions call the corresponding methods on a default Finder.
// For customised Finder behaviour, pass appropriate options to New().
package favicon

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	urls "net/url"
	"path/filepath"
	"sort"

	gq "github.com/PuerkitoBio/goquery"
	"github.com/friendsofgo/errors"
	"golang.org/x/net/html"
)

// UserAgent is sent in the User-Agent HTTP header.
var UserAgent = "go-favicon/0.1"

// Logger describes the logger used by Finder.
type Logger interface {
	Printf(string, ...interface{})
}

// black hole logger
type nullLogger struct{}

func (l nullLogger) Printf(format string, arg ...interface{}) {}

var (
	finder *Finder          // used by package-level functions
	client = &http.Client{} // default client used by Finder
)

func init() {
	finder = New()
}

// Filter accepts/rejects/modifies Icons. If if returns nil, the Icon is ignored.
// Set a Finder's filters by passing WithFilter(...) to New().
type Filter func(*Icon) *Icon

// Sorter Icons.
// Set a Finder's sorter by passing WithSorter(...) to New().
type Sorter func([]*Icon) sort.Interface

// Option configures Finder. Pass Options to New().
type Option func(*Finder)

// WithLogger sets the logger used by Finder.
func WithLogger(logger Logger) Option {
	return func(f *Finder) {
		f.log = logger
	}
}

// WithClient configures Finder to use the given HTTP client.
func WithClient(client *http.Client) Option {
	return func(f *Finder) {
		f.client = client
	}
}

// WithFilter only returns Icons accepted by Filter functions.
func WithFilter(filter ...Filter) Option {
	return func(f *Finder) {
		f.filters = append(f.filters, filter...)
	}
}

// WithSorter configures Finder to use the given Sorter.
func WithSorter(sorter Sorter) Option {
	return func(f *Finder) {
		f.sorter = sorter
	}
}

// OnlyMimeType only finds Icons that have one of the specified MIME types,
// e.g. "image/png" or "image/jpeg".
func OnlyMimeType(mimeType ...string) Option {
	return WithFilter(func(i *Icon) *Icon {
		for _, s := range mimeType {
			if i.MimeType == s {
				return i
			}
		}
		return nil
	})
}

// MinWidth ignores icons smaller than the given width.
func MinWidth(width int) Option {
	return WithFilter(func(icon *Icon) *Icon {
		if icon.Width < width {
			return nil
		}
		return icon
	})
}

// MaxWidth ignores icons larger than the given width.
func MaxWidth(width int) Option {
	return WithFilter(func(icon *Icon) *Icon {
		if icon.Width > width {
			return nil
		}
		return icon
	})
}

// MinHeight ignores icons smaller than the given height.
func MinHeight(height int) Option {
	return WithFilter(func(icon *Icon) *Icon {
		if icon.Height < height {
			return nil
		}
		return icon
	})
}

// MaxHeight ignores icons larger than the given height.
func MaxHeight(height int) Option {
	return WithFilter(func(icon *Icon) *Icon {
		if icon.Height > height {
			return nil
		}
		return icon
	})
}

var (
	// IgnoreWellKnown ignores common locations like /favicon.ico.
	IgnoreWellKnown Option = func(f *Finder) { f.ignoreWellKnown = true }

	// IgnoreManifest ignores manifest.json files.
	IgnoreManifest Option = func(f *Finder) { f.ignoreManifest = true }

	// IgnoreNoSize ignores icons with no specified size.
	IgnoreNoSize Option = WithFilter(func(icon *Icon) *Icon {
		if icon.Width == 0 || icon.Height == 0 {
			return nil
		}
		return icon
	})

	// OnlyPNG ignores non-PNG files.
	OnlyPNG Option = OnlyMimeType("image/png")

	// OnlyICO ignores non-ICO files.
	OnlyICO Option = WithFilter(func(icon *Icon) *Icon {
		if icon.MimeType == "image/x-icon" || icon.MimeType == "image/vnd.microsoft.icon" {
			return icon
		}
		return nil
	})

	// OnlySquare ignores non-square files. NOTE: Icons without a known size are also returned.
	OnlySquare Option = WithFilter(func(icon *Icon) *Icon {
		if !icon.IsSquare() {
			return nil
		}
		return icon
	})

	// SortByWidth sorts icons by width (largest first), and then by image type
	// (PNG > JPEG > SVG > ICO).
	SortByWidth Option = WithSorter(func(icons []*Icon) sort.Interface {
		return ByWidth(icons)
	})

	// NopSort represents a no operation sorting.
	NopSort Option = WithSorter(nil)
)

// Finder discovers favicons for a URL.
// By default, a Finder looks in the following places:
//
// The HTML page at the given URL for...
//   - icons in <link> tags
//   - Open Graph images
//   - Twitter images
//
// The manifest file...
//   - defined in the HTML page
//     -- or --
//   - /manifest.json
//
// Standard favicon paths
//   - /favicon.ico
//   - /apple-touch-icon.png
//
// Pass the IgnoreManifest and/or IgnoreWellKnown Options to New() to
// reduce the number of requests made to webservers.
type Finder struct {
	ignoreManifest  bool
	ignoreWellKnown bool
	log             Logger
	client          *http.Client
	filters         []Filter
	sorter          Sorter
}

// New creates a new Finder configured with the given options.
func New(option ...Option) *Finder {
	f := &Finder{
		log:     nullLogger{},
		client:  client,
		filters: []Filter{},
	}
	SortByWidth(f) // Default sort option
	for _, fn := range option {
		fn(f)
	}
	return f
}

// Find finds favicons for URL.
func Find(url string) ([]*Icon, error) { return finder.Find(url) }

// Find finds favicons for URL.
func (f *Finder) Find(url string) ([]*Icon, error) {
	return f.newParser().parseURL(url)
}

// FindReader finds a favicon in HTML. It accepts an optional base URL, which
// is used to resolve relative links.
func FindReader(r io.Reader, baseURL ...string) ([]*Icon, error) {
	return finder.FindReader(r, baseURL...)
}

// FindReader finds a favicon in HTML.
func (f *Finder) FindReader(r io.Reader, baseURL ...string) ([]*Icon, error) {
	p := f.newParser()
	if len(baseURL) > 0 {
		u, err := urls.Parse(baseURL[0])
		if err != nil {
			return nil, errors.Wrap(err, "reader base URL")
		}
		p.baseURL = u
	}
	return p.parseReader(r)
}

// FindNode finds a favicon in HTML Node. It accepts an optional base URL, which
// is used to resolve relative links.
func FindNode(n *html.Node, baseURL ...string) ([]*Icon, error) {
	return finder.FindNode(n, baseURL...)
}

// FindNode finds a favicon in HTML Node.
func (f *Finder) FindNode(n *html.Node, baseURL ...string) ([]*Icon, error) {
	p := f.newParser()
	if len(baseURL) > 0 {
		u, err := urls.Parse(baseURL[0])
		if err != nil {
			return nil, errors.Wrap(err, "node base URL")
		}
		p.baseURL = u
	}
	return p.parseNode(n)
}

// FindGoQueryDocument finds a favicon in GoQueryDocument. It accepts an optional base URL, which
// is used to resolve relative links.
func FindGoQueryDocument(doc *gq.Document, baseURL ...string) ([]*Icon, error) {
	return finder.FindGoQueryDocument(doc, baseURL...)
}

// FindGoQueryDocument finds a favicon in GoQueryDocument.
func (f *Finder) FindGoQueryDocument(doc *gq.Document, baseURL ...string) ([]*Icon, error) {
	p := f.newParser()
	if len(baseURL) > 0 {
		u, err := urls.Parse(baseURL[0])
		if err != nil {
			return nil, errors.Wrap(err, "node base URL")
		}
		p.baseURL = u
	}
	return p.parseGoQueryDocument(doc)
}

// Retrieve a URL and return response body. Returns an error if response status >= 300.
func (f *Finder) fetchURL(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "request URL")
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "retrieve URL")
	}
	f.log.Printf("[%d] %s", resp.StatusCode, url)

	if resp.StatusCode > 299 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("[%d] %s", resp.StatusCode, resp.Status)
	}

	return resp.Body, nil
}

type parser struct {
	baseURL *urls.URL
	charset string

	find *Finder
}

func (f *Finder) newParser() *parser {
	return &parser{find: f}
}

func (p *parser) absURL(url string) string {
	if url == "" || p.baseURL == nil {
		return url
	}

	u, err := urls.Parse(url)
	if err != nil {
		return ""
	}
	if p.baseURL != nil {
		return p.baseURL.ResolveReference(u).String()
	}
	return url
}

// return MIME type based on file extension in URL
func mimeTypeURL(url string) string {
	u, err := urls.Parse(url)
	if err != nil {
		return ""
	}
	return mime.TypeByExtension(filepath.Ext(u.Path))
}
