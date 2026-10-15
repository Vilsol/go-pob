package importproxy

import "regexp"

type Post struct {
	URL     string
	Fields  string
	CodeOut string
}

type Site struct {
	Label    string
	ID       string
	Match    string
	Regex    string
	Download string
	Link     string
	Post     *Post
}

var SITES = []Site{
	{
		Label:    "Maxroll",
		ID:       "Maxroll",
		Match:    `^https:\/\/maxroll\.gg\/poe\/pob\/.*`,
		Regex:    `maxroll\.gg\/poe\/pob\/(.+)\s*$`,
		Download: "maxroll.gg/poe/api/pob/$1",
		Link:     "https://maxroll.gg/poe/pob/",
		Post:     &Post{URL: "https://maxroll.gg/poe/api/pob", Fields: "pobCode=", CodeOut: "https://maxroll.gg/poe/pob/"},
	},
	{
		Label:    "pob.codes",
		ID:       "PoBCodes",
		Match:    `^https:\/\/pob\.codes\/b\/.+`,
		Regex:    `pob\.codes\/b\/(.+)\s*$`,
		Download: "api.pob.codes/$1/raw",
		Link:     "https://pob.codes/b/",
		Post:     &Post{URL: "https://api.pob.codes/pob/plain", Fields: "", CodeOut: "https://pob.codes/b/"},
	},
	{
		Label:    "pobb.in",
		ID:       "POBBin",
		Match:    `^https:\/\/pobb\.in\/.+`,
		Regex:    `pobb\.in\/(.+)\s*$`,
		Download: "pobb.in/pob/$1",
		Link:     "https://pobb.in/",
		Post:     &Post{URL: "https://pobb.in/pob/", Fields: "", CodeOut: "https://pobb.in/"},
	},
	{
		Label:    "PoeNinja",
		ID:       "PoeNinja",
		Match:    `^https:\/\/poe\.ninja\/?p?o?e?1?\/pob\/\w+`,
		Regex:    `poe\.ninja\/?p?o?e?1?\/pob\/(\w+)\s*$`,
		Download: "poe.ninja/poe1/pob/raw/$1",
		Link:     "https://poe.ninja/poe1/pob/",
		Post:     &Post{URL: "https://poe.ninja/poe1/pob/api/upload", Fields: "code=", CodeOut: ""},
	},
	{Label: "Pastebin.com", ID: "pastebin", Match: `^https:\/\/pastebin\.com\/\w+`, Regex: `pastebin\.com\/(\w+)\s*$`, Download: "pastebin.com/raw/$1"},
	{
		Label:    "PastebinP.com",
		ID:       "pastebinProxy",
		Match:    `^https:\/\/pastebinp\.com\/\w+`,
		Regex:    `pastebinp\.com\/(\w+)\s*$`,
		Download: "pastebinp.com/raw/$1",
	},
	{
		Label:    "poedb.tw",
		ID:       "PoEDB",
		Match:    `^https:\/\/poedb\.tw\/.+`,
		Regex:    `poedb\.tw\/pob\/(.+)\s*$`,
		Download: "poedb.tw/pob/$1/raw",
		Link:     "https://poedb.tw/pob/",
		Post:     &Post{URL: "https://poedb.tw/pob/api/gen", Fields: "", CodeOut: ""},
	},
}

type compiledSite struct {
	*Site
	match *regexp.Regexp
	id    *regexp.Regexp
}

var sites = func() map[string]compiledSite {
	m := make(map[string]compiledSite, len(SITES))
	for i := range SITES {
		s := &SITES[i]
		m[s.ID] = compiledSite{Site: s, match: regexp.MustCompile(s.Match), id: idPattern(s.Regex)}
	}
	return m
}()
