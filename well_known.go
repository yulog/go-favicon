// Copyright (c) 2020 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT
// Created on 2020-11-09

package favicon

// IconNames are common names of icon files hosted in server roots.
var IconNames = []string{
	"favicon.ico",
	"apple-touch-icon.png",
}

func (p *parser) findWellKnownIcons() []Icon {
	if p.baseURL == nil {
		return nil
	}

	var (
		icons []Icon
		root  = p.baseURL.Scheme + "://" + p.baseURL.Host + "/"
	)
	for _, name := range IconNames {
		u := root + name
		r, err := p.find.fetchURL(u)
		if err != nil {
			continue
		}
		r.Close()

		p.find.log.Printf("(well-known) %s", u)
		icons = append(icons, Icon{URL: u})
	}

	return icons
}
