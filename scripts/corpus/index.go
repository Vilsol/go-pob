package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Link struct {
	URL       string `json:"url"`
	Host      string `json:"host"`
	Source    string `json:"source"`
	PostedAt  string `json:"posted_at"`
	Context   string `json:"context"`
	FirstSeen string `json:"first_seen,omitempty"`
}

type hostSpec struct {
	host       string
	pattern    *regexp.Regexp
	validID    func(string) bool
	canonical  func(string) string
	rawAllowed bool
	raw        func(string) string
}

var (
	rePastebin   = regexp.MustCompile(`(?i)pastebin\.com/(?:raw/)?([A-Za-z0-9]+)`)
	pobbinIDPart = `(?:[A-Za-z0-9_-]|\\+[_-])`
	rePobbin     = regexp.MustCompile(`(?i)pobb\.in/(?:pob/)?(?:u/` + pobbinIDPart + `+/)?(` + pobbinIDPart + `+)`)
	rePobCodes   = regexp.MustCompile(`(?i)pob\.codes/b/(` + pobbinIDPart + `+)`)
)

var hostSpecs = []hostSpec{
	{
		host:      "pastebin.com",
		pattern:   rePastebin,
		validID:   func(id string) bool { return len(id) == 8 },
		canonical: func(id string) string { return "https://pastebin.com/" + id },
		raw:       func(id string) string { return "https://pastebin.com/raw/" + id },
	},
	{
		host:       "pob.codes",
		pattern:    rePobCodes,
		validID:    func(id string) bool { return len(id) == 11 },
		canonical:  func(id string) string { return "https://pob.codes/b/" + id },
		rawAllowed: true,
		raw:        func(id string) string { return "https://api.pob.codes/" + id + "/raw" },
	},
	{
		host:      "pobb.in",
		pattern:   rePobbin,
		validID:   func(id string) bool { return len(id) >= 6 && len(id) <= 24 && id != "pob" },
		canonical: func(id string) string { return "https://pobb.in/" + id },
		raw:       func(id string) string { return "https://pobb.in/" + id + "/raw" },
	},
}

func hostSpecFor(host string) *hostSpec {
	for i := range hostSpecs {
		if hostSpecs[i].host == host {
			return &hostSpecs[i]
		}
	}
	return nil
}

func unescapeID(id string) string {
	if !strings.Contains(id, `\`) {
		return id
	}
	return strings.ReplaceAll(id, `\`, "")
}

func extractLinks(text string) ([]struct{ Host, ID string }, map[string]int) {
	var found []struct{ Host, ID string }
	rejected := map[string]int{}
	seen := map[string]bool{}
	for i := range hostSpecs {
		spec := &hostSpecs[i]
		for _, m := range spec.pattern.FindAllStringSubmatch(text, -1) {
			id := unescapeID(m[1])
			if !spec.validID(id) {
				rejected[spec.host]++
				continue
			}
			key := spec.host + "/" + id
			if seen[key] {
				continue
			}
			seen[key] = true
			found = append(found, struct{ Host, ID string }{spec.host, id})
		}
	}
	return found, rejected
}

type linkIndex struct {
	path  string
	byID  map[string]Link
	order []string
}

func loadIndex(work string) (*linkIndex, error) {
	idx := &linkIndex{path: filepath.Join(work, "index.jsonl"), byID: map[string]Link{}}
	f, err := os.Open(idx.path)
	if os.IsNotExist(err) {
		return idx, nil
	}
	if err != nil {
		return nil, fmt.Errorf("loadIndex: %w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for ln := 1; sc.Scan(); ln++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var l Link
		if err := json.Unmarshal([]byte(line), &l); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", idx.path, ln, err)
		}
		if _, ok := idx.byID[l.URL]; !ok {
			idx.order = append(idx.order, l.URL)
		}
		idx.byID[l.URL] = l
	}
	return idx, wrapScan(sc.Err())
}

func (idx *linkIndex) add(l Link) bool {
	if _, ok := idx.byID[l.URL]; ok {
		return false
	}
	idx.byID[l.URL] = l
	idx.order = append(idx.order, l.URL)
	return true
}

func (idx *linkIndex) save() error {
	if err := os.MkdirAll(filepath.Dir(idx.path), 0o755); err != nil {
		return fmt.Errorf("save: %w", err)
	}
	tmp := idx.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for _, u := range idx.order {
		if err := enc.Encode(idx.byID[u]); err != nil {
			f.Close()
			return fmt.Errorf("save: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return fmt.Errorf("save: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("save: %w", err)
	}
	if err := os.Rename(tmp, idx.path); err != nil {
		return fmt.Errorf("renaming %s: %w", tmp, err)
	}
	return nil
}

func (idx *linkIndex) links() []Link {
	out := make([]Link, 0, len(idx.order))
	for _, u := range idx.order {
		out = append(out, idx.byID[u])
	}
	return out
}

func countBy(links []Link, key func(Link) string) []struct {
	Key string
	N   int
} {
	m := map[string]int{}
	for _, l := range links {
		m[key(l)]++
	}
	out := make([]struct {
		Key string
		N   int
	}, 0, len(m))
	for k, v := range m {
		out = append(out, struct {
			Key string
			N   int
		}{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].Key < out[j].Key
	})
	return out
}

func utc(sec int64) string { return time.Unix(sec, 0).UTC().Format(time.RFC3339) }

func wrapScan(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("scanning: %w", err)
}
