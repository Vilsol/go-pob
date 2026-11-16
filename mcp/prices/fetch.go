package prices

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// DefaultURL is go-pob's price service: a pass-through to poe.ninja's economy API under poe.ninja's own paths,
	// which asks desktop clients to proxy through their own backend rather than call it directly.
	DefaultURL = "https://prices.pob.gg"
	apiPath    = "/poe1/api/economy"
	userAgent  = "go-pob (+https://github.com/Vilsol/go-pob)"
	// DefaultTTL is how long a fetched overview is used before it is fetched again.
	DefaultTTL = 6 * time.Hour
	maxBody    = 32 << 20
)

// Live fetches poe.ninja's overviews through the price service at Base for the current challenge league (the first
// league poe.ninja lists), one request per category per TTL, and keeps them in Dir. A stale copy answers when a refetch
// fails.
type Live struct {
	Base   string
	Dir    string
	TTL    time.Duration
	Client *http.Client
	Now    func() time.Time
}

// NewLive is a book over Live reading the price service at serviceURL (DefaultURL when empty); dir is where responses
// are cached, per service host (os.UserCacheDir()/go-pob/prices when empty).
func NewLive(dir, serviceURL string) (*Book, error) {
	if serviceURL == "" {
		serviceURL = DefaultURL
	}
	u, err := url.Parse(serviceURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("price service URL %q: want http(s)://host[:port][/path]", serviceURL)
	}
	if dir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("price cache: %w", err)
		}
		dir = filepath.Join(base, "go-pob", "prices")
	}
	dir = filepath.Join(dir, safeName(u.Host))
	return newBook(&Live{Base: strings.TrimSuffix(serviceURL, "/"), Dir: dir, TTL: DefaultTTL, Client: http.DefaultClient, Now: time.Now}), nil
}

func (f *Live) league() (string, error) {
	body, _, err := f.cached("leagues.json", f.Base+apiPath+"/leagues")
	if err != nil {
		return "", fmt.Errorf("leagues: %w", err)
	}
	var leagues []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &leagues); err != nil {
		return "", fmt.Errorf("leagues: %w", err)
	}
	if len(leagues) == 0 || leagues[0].ID == "" {
		return "", fmt.Errorf("leagues: none listed")
	}
	return leagues[0].ID, nil
}

func (f *Live) overview(league string, c Category) ([]byte, time.Time, error) {
	path, typ := "stash/current/item", string(c)
	switch c {
	case Currency:
		path = "stash/current/currency"
	case Exchange:
		path, typ = "exchange/current", string(Currency)
	}
	u := fmt.Sprintf("%s%s/%s/overview?league=%s&type=%s", f.Base, apiPath, path, url.QueryEscape(league), typ)
	return f.cached(filepath.Join(safeName(league), string(c)+".json"), u)
}

func safeName(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == '.' {
			return '_'
		}
		return r
	}, s)
}

// cached answers from the cache file while it is younger than the TTL; otherwise it fetches, sending the stored ETag
// so an unchanged overview costs nothing to send. The fetch time is the file's modification time.
func (f *Live) cached(name, u string) ([]byte, time.Time, error) {
	path := filepath.Join(f.Dir, name)
	st, statErr := os.Stat(path)
	if statErr == nil && f.Now().Sub(st.ModTime()) < f.TTL {
		body, err := os.ReadFile(path)
		return body, st.ModTime(), err
	}
	body, err := f.fetch(path, u)
	if err == nil {
		return body, f.Now(), nil
	}
	if statErr == nil {
		if stale, rerr := os.ReadFile(path); rerr == nil {
			return stale, st.ModTime(), nil
		}
	}
	return nil, time.Time{}, err
}

func (f *Live) fetch(path, u string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil) //nolint:noctx // a price lookup runs inside a tool call that takes no context
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	etag, _ := os.ReadFile(path + ".etag")
	if _, err := os.Stat(path); err == nil && len(etag) > 0 {
		req.Header.Set("If-None-Match", string(etag))
	}
	res, err := f.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer res.Body.Close()
	now := f.Now()
	if res.StatusCode == http.StatusNotModified {
		_ = os.Chtimes(path, now, now)
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("cache: %w", err)
		}
		return body, nil
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the price service answered %s", res.Status)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("the price service answered with invalid JSON")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("cache: %w", err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return nil, fmt.Errorf("cache: %w", err)
	}
	_ = os.Chtimes(path, now, now)
	if tag := res.Header.Get("ETag"); tag != "" {
		_ = os.WriteFile(path+".etag", []byte(tag), 0o600)
	} else {
		_ = os.Remove(path + ".etag")
	}
	return body, nil
}
