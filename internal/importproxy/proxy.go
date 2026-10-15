package importproxy

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const USER_AGENT = "go-pob (+https://pob.gg)"

const BUILD_CAP = 1 << 20

const day = 24 * 60 * 60

var immutable = map[string]bool{"Maxroll": true, "PoBCodes": true, "POBBin": true, "PoeNinja": true}

const IMMUTABLE_TTL = 7 * day

const EDITABLE_TTL = 5 * 60

type Entry struct {
	Status  int
	Header  http.Header
	Body    []byte
	Expires time.Time
}

type Cache interface {
	Get(key string) (Entry, bool)
	Put(key string, e Entry)
}

type Deps struct {
	Client *http.Client
	Now    func() time.Time
	Cache  Cache
}

type response struct {
	status int
	header http.Header
	body   []byte
}

func respond(status int, contentType string, body string, kv ...string) *response {
	h := http.Header{"Content-Type": {contentType}}
	for i := 0; i < len(kv); i += 2 {
		h.Set(kv[i], kv[i+1])
	}
	return &response{status: status, header: h, body: []byte(body)}
}

func text(status int, body string, kv ...string) *response {
	return respond(status, "text/plain; charset=utf-8", body, kv...)
}

func readCapped(contentLength int64, body io.ReadCloser, limit int) (string, bool, error) {
	defer body.Close()
	if contentLength > int64(limit) {
		return "", false, nil
	}
	b, err := io.ReadAll(io.LimitReader(body, int64(limit)+1))
	if err != nil {
		return "", false, fmt.Errorf("read: %w", err)
	}
	if len(b) > limit {
		return "", false, nil
	}
	return decodeUTF8(b, false), true, nil
}

var sMaxAge = regexp.MustCompile(`s-maxage=(\d+)`)

func cached(key string, d Deps, produce func() *response) *response {
	if d.Cache != nil {
		if hit, ok := d.Cache.Get(key); ok && d.Now().Before(hit.Expires) {
			return &response{status: hit.Status, header: hit.Header.Clone(), body: hit.Body}
		}
	}
	res := produce()
	if m := sMaxAge.FindStringSubmatch(res.header.Get("Cache-Control")); d.Cache != nil && m != nil {
		secs, _ := strconv.ParseInt(m[1], 10, 64)
		d.Cache.Put(key, Entry{Status: res.status, Header: res.header.Clone(), Body: res.body, Expires: d.Now().Add(time.Duration(secs) * time.Second)})
	}
	return res
}

func upstreamError(who string, res *http.Response) *response {
	msg := fmt.Sprintf("%s answered %d", who, res.StatusCode)
	if cf := res.Header.Get("cf-mitigated"); cf != "" {
		msg += " (" + latin1(cf) + ")"
	}
	return text(res.StatusCode, msg)
}

type request struct {
	method  string
	url     string
	headers map[string]string
	body    string
}

func (d Deps) fetch(r *http.Request, req request, manual bool) (*http.Response, error) {
	var body io.Reader
	if req.method != http.MethodGet {
		body = strings.NewReader(req.body)
	}
	up, err := http.NewRequestWithContext(r.Context(), req.method, req.url, body)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	for k, v := range req.headers {
		up.Header.Set(k, v)
	}
	client := d.Client
	if manual {
		c := *client
		c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		client = &c
	}
	res, err := client.Do(up)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	return res, nil
}

func get(url string) request {
	return request{method: http.MethodGet, url: url, headers: map[string]string{"User-Agent": USER_AGENT}}
}

var idCapture = regexp.MustCompile(`\((.+?)\)\\s\*\$$`)

func idPattern(source string) *regexp.Regexp {
	inner := "[^\\x00-\\x{10FFFF}]"
	if m := idCapture.FindStringSubmatch(source); m != nil {
		inner = m[1]
	}
	return regexp.MustCompile(`^(?:` + inner + `)$`)
}

var safeID = regexp.MustCompile(`^[\w-]{1,128}$`)

const ninjaCharacter = "overview/code"

var (
	ninjaFields = []string{"account", "name", "overview", "type"}
	ninjaField  = regexp.MustCompile(`^[^` + jsSpace + `&?#/\\=]{1,64}$`)
)

func handleBuild(r *http.Request, d Deps) *response {
	if r.Method != http.MethodGet {
		return text(405, "GET only", "Allow", "GET")
	}
	params := parseSearchParams(r.URL.RawQuery)
	site, ok := sites[params.value("site")]
	if !ok {
		return text(400, "unknown site")
	}
	id := params.value("id")
	ttl := EDITABLE_TTL
	if immutable[site.ID] {
		ttl = IMMUTABLE_TTL
	}
	if site.ID == "PoeNinja" && id == ninjaCharacter {
		query := make([]string, len(ninjaFields))
		for i, f := range ninjaFields {
			v := params.value(f)
			if !ninjaField.MatchString(v) {
				return text(400, "malformed "+site.Label+" character")
			}
			query[i] = f + "=" + encodeURIComponent(v)
		}
		id = ninjaCharacter + "?" + strings.Join(query, "&")
		ttl = EDITABLE_TTL
	} else if !safeID.MatchString(id) || !site.id.MatchString(id) {
		return text(400, "malformed "+site.Label+" id")
	}
	url := "https://" + strings.Replace(site.Download, "$1", id, 1)
	return cached("/api/build?site="+site.ID+"&id="+id, d, func() *response {
		res, err := d.fetch(r, get(url), false)
		if err != nil {
			return text(502, site.Label+" is unreachable")
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			res.Body.Close()
			return upstreamError(site.Label, res)
		}
		body, ok, err := readCapped(res.ContentLength, res.Body, BUILD_CAP)
		if err != nil {
			return text(502, site.Label+" is unreachable")
		}
		if !ok {
			return text(502, fmt.Sprintf("%s answered more than %d bytes", site.Label, BUILD_CAP))
		}
		cc := fmt.Sprintf("public, max-age=%d, s-maxage=%d", ttl, ttl)
		return text(200, body, "Cache-Control", cc)
	})
}

const LINK_CAP = 1 << 10

var link = regexp.MustCompile(`^https:\/\/[\w\-./]+$`)

const XML_CAP = 32 << 20

var buildCodeChars = regexp.MustCompile(`^[\w\-+/]+={0,2}$`)

func atob(s string) ([]byte, bool) {
	if len(s)%4 == 0 {
		s = strings.TrimSuffix(s, "=")
		s = strings.TrimSuffix(s, "=")
	}
	if len(s)%4 == 1 || strings.ContainsRune(s, '=') {
		return nil, false
	}
	b, err := base64.RawStdEncoding.DecodeString(s)
	return b, err == nil
}

func isBuildCode(code string) bool {
	if !buildCodeChars.MatchString(code) {
		return false
	}
	b, ok := atob(strings.NewReplacer("-", "+", "_", "/").Replace(code))
	if !ok {
		return false
	}
	z, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return false
	}
	xml, ok, err := readCapped(-1, z, XML_CAP)
	return err == nil && ok && xml != "" && strings.Contains(xml, "<PathOfBuilding")
}

func handleShare(r *http.Request, d Deps) *response {
	if r.Method != http.MethodPost {
		return text(405, "POST only", "Allow", "POST")
	}
	site, ok := sites[parseSearchParams(r.URL.RawQuery).value("site")]
	if !ok || site.Post == nil {
		return text(400, "unknown site")
	}
	body, ok, err := readCapped(r.ContentLength, r.Body, BUILD_CAP)
	if err != nil {
		return text(400, "not a build code")
	}
	if !ok {
		return text(413, fmt.Sprintf("a build code is at most %d bytes", BUILD_CAP))
	}
	code := jsTrim(body)
	if !isBuildCode(code) {
		return text(400, "not a build code")
	}
	res, err := d.fetch(r, request{
		method:  http.MethodPost,
		url:     site.Post.URL,
		headers: map[string]string{"User-Agent": USER_AGENT, "Content-Type": "application/x-www-form-urlencoded"},
		body:    site.Post.Fields + code,
	}, false)
	if err != nil {
		return text(502, site.Label+" is unreachable")
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return upstreamError(site.Label, res)
	}
	answer, ok, err := readCapped(res.ContentLength, res.Body, LINK_CAP)
	l := site.Post.CodeOut + jsTrim(answer)
	if err != nil || !ok || !link.MatchString(l) || !site.match.MatchString(l) {
		return text(502, site.Label+" answered no link")
	}
	return text(200, l, "Cache-Control", "no-store")
}

const poeURL = "http://poeurl.com/"

var (
	poeURLID   = regexp.MustCompile(`^\w{1,32}$`)
	treeLink   = regexp.MustCompile(`^https:\/\/www\.pathofexile\.com\/(?:fullscreen-)?passive-skill-tree\/[\w\-./]+$`)
	treePrefix = regexp.MustCompile(`^https:\/\/www\.pathofexile\.com\/(?:fullscreen-)?passive-skill-tree\/`)
	redirectTo = regexp.MustCompile(`^https?:\/\/[^` + jsSpace + `]+$`)
	poeURLHost = regexp.MustCompile(`poeurl\.com\/`)
)

func within(s string, prefix *regexp.Regexp, n int) bool {
	return len([]rune(s))-len([]rune(prefix.FindString(s))) <= n
}

var redirectPrefix = regexp.MustCompile(`^https?:\/\/`)

func handlePoeURL(r *http.Request, d Deps) *response {
	if r.Method != http.MethodGet {
		return text(405, "GET only", "Allow", "GET")
	}
	params := parseSearchParams(r.URL.RawQuery)
	id, hasID := params.get("id")
	shrink, hasShrink := params.get("shrink")
	if hasID && !poeURLID.MatchString(id) {
		return text(400, "malformed PoEURL id")
	}
	if !hasID && (!hasShrink || !treeLink.MatchString(shrink) || !within(shrink, treePrefix, 4096)) {
		return text(400, "not a passive tree link")
	}
	var res *http.Response
	var err error
	if hasID {
		res, err = d.fetch(r, get(poeURL+id), true)
	} else {
		res, err = d.fetch(r, get(poeURL+"shrink.php?url="+encodeURIComponent(shrink)), false)
	}
	if err != nil {
		return text(502, "PoEURL is unreachable")
	}
	if hasID {
		res.Body.Close()
		to := latin1(res.Header.Get("Location"))
		if !redirectTo.MatchString(to) || !within(to, redirectPrefix, 4096) || poeURLHost.MatchString(to) {
			return text(502, "Failed to resolve PoEURL link")
		}
		cc := fmt.Sprintf("public, max-age=%d, s-maxage=%d", IMMUTABLE_TTL, IMMUTABLE_TTL)
		return text(200, to, "Cache-Control", cc)
	}
	short := ""
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		body, ok, err := readCapped(res.ContentLength, res.Body, LINK_CAP)
		if err == nil && ok {
			short = jsTrim(body)
		}
	} else {
		res.Body.Close()
	}
	if !poeURLID.MatchString(short) {
		return text(502, "Failed to get PoEURL link")
	}
	return text(200, poeURL+short, "Cache-Control", "no-store")
}

var REALMS = []string{"pc", "xbox", "sony"}

const poeHost = "https://www.pathofexile.com/"

const POE_CAP = 4 << 20
const POE_TTL = 60

type endpoint struct {
	path      func(a, c, r string) string
	character bool
}

var endpoints = map[string]endpoint{
	"characters": {path: func(a, _, r string) string { return "character-window/get-characters?accountName=" + a + "&realm=" + r }},
	"profile":    {path: func(a, _, _ string) string { return "account/view-profile/" + a }},
	"items": {path: func(a, c, r string) string {
		return "character-window/get-items?accountName=" + a + "&character=" + c + "&realm=" + r
	}, character: true},
	"passives": {path: func(a, c, r string) string {
		return "character-window/get-passive-skills?accountName=" + a + "&character=" + c + "&realm=" + r
	}, character: true},
}

var (
	account   = regexp.MustCompile(`^[^` + jsSpace + `#&?/\\%=]{1,64}(?:#\d{1,8})?$`)
	character = regexp.MustCompile(`^[^` + jsSpace + `#&?/\\%=]{1,64}$`)
)

func queryValue(v string) string { return strings.ReplaceAll(encodeURIComponent(v), "%2B", "+") }

var viewProfile = regexp.MustCompile(`\/view-profile\/([^/]+)\/characters`)

func profileName(html string) string {
	m := viewProfile.FindStringSubmatch(html)
	if m == nil {
		return ""
	}
	if name, ok := decodeURIComponent(m[1]); ok {
		return name
	}
	return m[1]
}

func rateLimited(res *http.Response) *response {
	retry := strconv.Itoa(POE_TTL)
	if v, ok := res.Header["Retry-After"]; ok {
		retry = strings.Join(v, ", ")
	}
	wait := jsNumber(retry)
	if wait == 0 || math.IsNaN(wait) {
		wait = POE_TTL
	}
	out := text(429, "rate limited by pathofexile.com", "Retry-After", retry)
	for k, v := range res.Header {
		if strings.HasPrefix(strings.ToLower(k), "x-rate-limit") {
			out.header.Set(k, strings.Join(v, ", "))
		}
	}
	out.header.Set("Cache-Control", "s-maxage="+jsNumberString(wait))
	return out
}

func handlePoe(name string, r *http.Request, d Deps) *response {
	if r.Method != http.MethodGet {
		return text(405, "GET only", "Allow", "GET")
	}
	ep, ok := endpoints[name]
	if !ok {
		return text(404, "unknown endpoint")
	}
	params := parseSearchParams(r.URL.RawQuery)
	realm := params.value("realm")
	if !contains(REALMS, realm) {
		return text(400, "unknown realm")
	}
	acc := params.value("account")
	if !account.MatchString(acc) {
		return text(400, "malformed account name")
	}
	char := ""
	if ep.character {
		char = params.value("character")
		if !character.MatchString(char) {
			return text(400, "malformed character name")
		}
	}
	path := ep.path(queryValue(acc), queryValue(char), realm)
	key := "/api/poe/" + name + "?realm=" + realm + "&account=" + queryValue(acc) + "&character=" + queryValue(char)
	return cached(key, d, func() *response {
		res, err := d.fetch(r, get(poeHost+path), false)
		if err != nil {
			return text(502, "pathofexile.com is unreachable")
		}
		if res.StatusCode == http.StatusTooManyRequests {
			res.Body.Close()
			return rateLimited(res)
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			res.Body.Close()
			return upstreamError("pathofexile.com", res)
		}
		body, ok, err := readCapped(res.ContentLength, res.Body, POE_CAP)
		if err != nil {
			return text(502, "pathofexile.com is unreachable")
		}
		if !ok {
			return text(502, fmt.Sprintf("pathofexile.com answered more than %d bytes", POE_CAP))
		}
		cc := fmt.Sprintf("public, max-age=%d, s-maxage=%d", POE_TTL, POE_TTL)
		if name != "profile" {
			return respond(200, "application/json", body, "Cache-Control", cc)
		}
		pn := profileName(body)
		if pn == "" {
			return text(502, "the profile page names no account")
		}
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(map[string]string{"accountName": pn})
		return respond(200, "application/json", strings.TrimSuffix(b.String(), "\n"), "Cache-Control", cc)
	})
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func Handler(d Deps) http.Handler {
	if d.Client == nil {
		d.Client = http.DefaultClient
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var res *response
		switch p := r.URL.Path; {
		case p == "/api/build":
			res = handleBuild(r, d)
		case p == "/api/share":
			res = handleShare(r, d)
		case p == "/api/poeurl":
			res = handlePoeURL(r, d)
		case strings.HasPrefix(p, "/api/poe/") && len(p) > len("/api/poe/") && !strings.Contains(p[len("/api/poe/"):], "/"):
			res = handlePoe(p[len("/api/poe/"):], r, d)
		default:
			http.NotFound(w, r)
			return
		}
		for k, v := range res.header {
			w.Header()[k] = v
		}
		w.WriteHeader(res.status)
		_, _ = w.Write(res.body)
	})
}

type MemoryCache struct {
	mu sync.Mutex
	m  map[string]Entry
}

func NewMemoryCache() *MemoryCache { return &MemoryCache{m: map[string]Entry{}} }

func (c *MemoryCache) Get(key string) (Entry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	return e, ok
}

func (c *MemoryCache) Put(key string, e Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = e
}
