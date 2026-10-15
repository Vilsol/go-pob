package importproxy

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var (
	tsSites = regexp.MustCompile(`(?s)export const SITES: Site\[\] = \[(.*?)\n\];`)
	tsField = regexp.MustCompile(`(\w+): ('(?:[^'\\]|\\.)*'|/(?:[^/\\\n]|\\.)+/([a-z]*))`)
)

// TestSitesMatchShareTS: SITES is share.ts's table; a site, field or regex source that differs fails here.
func TestSitesMatchShareTS(t *testing.T) {
	src, err := os.ReadFile("../../frontend/src/lib/design/share.ts")
	if err != nil {
		t.Fatal(err)
	}
	block := tsSites.FindSubmatch(src)
	if block == nil {
		t.Fatal("no SITES table in share.ts")
	}
	var ts []Site
	for _, m := range tsField.FindAllStringSubmatch(string(block[1]), -1) {
		key, lit := m[1], m[2]
		if m[3] != "" {
			t.Fatalf("%s: regex flags %q have no RE2 counterpart here", key, m[3])
		}
		val := strings.ReplaceAll(lit[1:len(lit)-len(m[3])-1], `\'`, `'`)
		if key == "label" {
			ts = append(ts, Site{})
		}
		if len(ts) == 0 {
			t.Fatalf("%s before the first label", key)
		}
		s := &ts[len(ts)-1]
		post := func() *Post {
			if s.Post == nil {
				s.Post = &Post{}
			}
			return s.Post
		}
		switch key {
		case "label":
			s.Label = val
		case "id":
			s.ID = val
		case "match":
			s.Match = val
		case "regex":
			s.Regex = val
		case "download":
			s.Download = val
		case "link":
			s.Link = val
		case "url":
			post().URL = val
		case "fields":
			post().Fields = val
		case "codeOut":
			post().CodeOut = val
		default:
			t.Fatalf("share.ts's SITES has a field %q the Go table lacks", key)
		}
	}
	if len(ts) != len(SITES) {
		t.Fatalf("share.ts has %d sites, Go %d", len(ts), len(SITES))
	}
	for i := range ts {
		if !reflect.DeepEqual(ts[i], SITES[i]) {
			t.Errorf("site %d: share.ts %+v %+v, Go %+v %+v", i, ts[i], ts[i].Post, SITES[i], SITES[i].Post)
		}
	}
}
