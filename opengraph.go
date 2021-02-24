// Copyright (c) 2020 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT
// Created on 2020-11-09

package favicon

import "strconv"

func (p *parser) parseOpenGraph(kv []string) []*Icon {
	var (
		icons []*Icon
		icon  *Icon
	)

	for i := 0; i < len(kv)-1; i += 2 {
		k, v := kv[i], kv[i+1]
		switch k {
		case "og:image":
			if icon != nil {
				icons = append(icons, icon)
			}
			icon = &Icon{URL: v}
			p.find.log.Printf("(opengraph) %s", icon.URL)
		case "og:image:type":
			if icon != nil {
				icon.MimeType = v
			}
		case "og:image:width":
			if icon != nil {
				if n, err := strconv.ParseInt(v, 10, 32); err == nil {
					icon.Width = int(n)
				}
			}
		case "og:image:height":
			if icon != nil {
				if n, err := strconv.ParseInt(v, 10, 32); err == nil {
					icon.Height = int(n)
				}
			}
		}
	}
	if icon != nil {
		icons = append(icons, icon)
	}
	return icons
}
