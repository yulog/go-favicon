// MIT License
//
// Copyright (c) 2025 yulog
//
// Copyright (c) 2020 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT
// Created on 2020-11-09

package favicon

import (
	"iter"
	"strconv"
)

func (p *parser) parseTwitterIter(kv []string) iter.Seq[*Icon] {
	return func(yield func(*Icon) bool) {
		var (
			icon *Icon
		)
		for i := 0; i < len(kv)-1; i += 2 {
			k, v := kv[i], kv[i+1]
			switch k {
			case "twitter:image:src", "twitter:image":
				if icon != nil {
					if !yield(icon) {
						return
					}
				}
				icon = &Icon{URL: v}
				p.find.log.Printf("(twitter) %s", icon.URL)
			case "twitter:image:width":
				if icon != nil {
					if n, err := strconv.ParseInt(v, 10, 32); err == nil {
						icon.Width = int(n)
					}
				}
			case "twitter:image:height":
				if icon != nil {
					if n, err := strconv.ParseInt(v, 10, 32); err == nil {
						icon.Height = int(n)
					}
				}
			}
		}
		if icon != nil {
			yield(icon)
		}
	}
}
