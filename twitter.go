// Copyright (c) 2020 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT
// Created on 2020-11-09

package favicon

import "strconv"

func (p *parser) parseTwitter(kv []string) []Icon {
	var (
		icons []Icon
		icon  *Icon
	)
	for i := 0; i < len(kv)-1; i += 2 {
		k, v := kv[i], kv[i+1]
		switch k {
		case "twitter:image:src", "twitter:image":
			if icon != nil {
				icons = append(icons, *icon)
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
		icons = append(icons, *icon)
	}
	return icons
}
