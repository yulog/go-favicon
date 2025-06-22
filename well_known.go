// MIT License
//
// Copyright (c) 2025 yulog
//
// Copyright (c) 2020 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT
// Created on 2020-11-09

package favicon

import "iter"

// IconNames are common names of icon files hosted in server roots.
var IconNames = []string{
	"favicon.ico",
	"apple-touch-icon.png",
}

func (p *parser) findWellKnownIconsIter() iter.Seq[*Icon] {
	return func(yield func(*Icon) bool) {
		if p.baseURL == nil {
			return
		}

		var (
			root = p.baseURL.Scheme + "://" + p.baseURL.Host + "/"
		)
		for _, name := range IconNames {
			u := root + name
			r, err := p.find.fetchURL(u)
			if err != nil {
				continue
			}
			r.Close()

			p.find.log.Printf("(well-known) %s", u)
			if !yield(&Icon{URL: u}) {
				return
			}
		}
	}
}
