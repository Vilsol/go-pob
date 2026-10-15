package importproxy

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type spec struct {
	plain  *string
	Prefix string `json:"prefix"`
	Repeat string `json:"repeat"`
	Count  int    `json:"count"`
	B64    string `json:"b64"`
	Code   *spec  `json:"code"`
	Suffix string `json:"suffix"`
}

func (s *spec) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var p string
		if err := json.Unmarshal(b, &p); err != nil {
			return fmt.Errorf("spec: %w", err)
		}
		s.plain = &p
		return nil
	}
	type raw spec
	if err := json.Unmarshal(b, (*raw)(s)); err != nil {
		return fmt.Errorf("spec: %w", err)
	}
	return nil
}

func (s *spec) bytes(t *testing.T) []byte {
	if s.plain != nil {
		return []byte(*s.plain)
	}
	out := []byte(s.Prefix)
	switch {
	case s.Repeat != "":
		out = append(out, strings.Repeat(s.Repeat, s.Count)...)
	case s.B64 != "":
		b, err := base64.StdEncoding.DecodeString(s.B64)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, b...)
	case s.Code != nil:
		var z bytes.Buffer
		w := zlib.NewWriter(&z)
		_, _ = w.Write(s.Code.bytes(t))
		_ = w.Close()
		out = append(out, base64.URLEncoding.EncodeToString(z.Bytes())...)
	}
	return append(out, s.Suffix...)
}

type step struct {
	Method  string             `json:"method"`
	Path    spec               `json:"path"`
	Body    *spec              `json:"body"`
	Status  int                `json:"status"`
	Text    *spec              `json:"text"`
	JSON    any                `json:"json"`
	Headers map[string]*string `json:"headers"`
	Fetches *int               `json:"fetches"`
	Cached  *int               `json:"cached"`
}

type call struct {
	URL      string            `json:"url"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	Body     *spec             `json:"body"`
	Redirect string            `json:"redirect"`
}

type contractCase struct {
	Name     string `json:"name"`
	Upstream struct {
		Status      int               `json:"status"`
		Headers     map[string]string `json:"headers"`
		Body        *spec             `json:"body"`
		Unreachable bool              `json:"unreachable"`
	} `json:"upstream"`
	Cache *bool  `json:"cache"`
	Steps []step `json:"steps"`
	Fetch []call `json:"fetch"`
}

func loadContract(t *testing.T) (map[string]any, []contractCase) {
	b, err := os.ReadFile("../../testdata/import_proxy/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Constants map[string]any `json:"constants"`
		Cases     []contractCase `json:"cases"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return f.Constants, f.Cases
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type seen struct {
	url, method, body string
	header            http.Header
}

func TestContractConstants(t *testing.T) {
	constants, _ := loadContract(t)
	got := map[string]any{"USER_AGENT": USER_AGENT, "BUILD_CAP": float64(BUILD_CAP), "LINK_CAP": float64(LINK_CAP), "POE_CAP": float64(POE_CAP),
		"IMMUTABLE_TTL": float64(IMMUTABLE_TTL), "EDITABLE_TTL": float64(EDITABLE_TTL), "POE_TTL": float64(POE_TTL)}
	if !reflect.DeepEqual(got, constants) {
		t.Fatalf("constants %v, contract %v", got, constants)
	}
}

func TestContract(t *testing.T) {
	_, cases := loadContract(t)
	if len(cases) == 0 {
		t.Fatal("no cases")
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			var calls []seen
			up := c.Upstream
			client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
				var body []byte
				if r.Body != nil {
					body, _ = io.ReadAll(r.Body)
				}
				calls = append(calls, seen{url: r.URL.String(), method: r.Method, body: string(body), header: r.Header})
				if up.Unreachable {
					return nil, errors.New("fetch failed")
				}
				res := &http.Response{StatusCode: up.Status, Header: http.Header{}, ContentLength: -1, Request: r}
				if res.StatusCode == 0 {
					res.StatusCode = 200
				}
				for k, v := range up.Headers {
					res.Header.Set(k, strings.TrimSpace(v))
				}
				if n, err := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64); err == nil {
					res.ContentLength = n
				}
				var b []byte
				if up.Body != nil {
					b = up.Body.bytes(t)
				}
				res.Body = io.NopCloser(bytes.NewReader(b))
				return res, nil
			})}
			cache := NewMemoryCache()
			d := Deps{Client: client, Now: func() time.Time { return time.Unix(0, 0) }, Cache: cache}
			if c.Cache != nil && !*c.Cache {
				d.Cache = nil
			}
			h := Handler(d)
			for i, s := range c.Steps {
				method := s.Method
				if method == "" {
					method = http.MethodGet
				}
				var body io.Reader
				if s.Body != nil {
					body = bytes.NewReader(s.Body.bytes(t))
				}
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, "https://pob.gg"+string(s.Path.bytes(t)), body))
				if rec.Code != s.Status {
					t.Fatalf("step %d: status %d, want %d (%q)", i, rec.Code, s.Status, trim(rec.Body.String()))
				}
				for k, v := range s.Headers {
					got, ok := rec.Header()[http.CanonicalHeaderKey(k)]
					if v == nil && ok || v != nil && (!ok || strings.Join(got, ", ") != *v) {
						t.Errorf("step %d: header %s = %q, want %v", i, k, got, deref(v))
					}
				}
				if s.Text != nil {
					if want := string(s.Text.bytes(t)); rec.Body.String() != want {
						t.Errorf("step %d: text %q, want %q", i, trim(rec.Body.String()), trim(want))
					}
				} else if s.JSON != nil {
					var got any
					if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || !reflect.DeepEqual(got, s.JSON) {
						t.Errorf("step %d: json %s, want %v", i, rec.Body.String(), s.JSON)
					}
				}
				if s.Fetches != nil && len(calls) != *s.Fetches {
					t.Errorf("step %d: %d fetches, want %d", i, len(calls), *s.Fetches)
				}
				if s.Cached != nil && len(cache.m) != *s.Cached {
					t.Errorf("step %d: %d cached, want %d", i, len(cache.m), *s.Cached)
				}
			}
			if c.Fetch == nil {
				return
			}
			if len(calls) != len(c.Fetch) {
				t.Fatalf("%d fetches, want %d", len(calls), len(c.Fetch))
			}
			for i, want := range c.Fetch {
				got := calls[i]
				method := want.Method
				if method == "" {
					method = http.MethodGet
				}
				wantBody := ""
				if want.Body != nil {
					wantBody = string(want.Body.bytes(t))
				}
				header := map[string]string{}
				for k := range got.header {
					header[k] = got.header.Get(k)
				}
				wantHeader := map[string]string{}
				for k, v := range want.Headers {
					wantHeader[http.CanonicalHeaderKey(k)] = v
				}
				if got.url != want.URL || got.method != method || got.body != wantBody || !reflect.DeepEqual(header, wantHeader) {
					t.Errorf("fetch %d: %s %s %v %q, want %s %s %v %q", i, got.method, got.url, header, trim(got.body), method, want.URL, wantHeader, trim(wantBody))
				}
			}
		})
	}
}

func trim(s string) string {
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}

func deref(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// TestCacheExpires: the Workers cache drops an answer at s-maxage; MemoryCache keeps it, so the handler checks Expires against Now.
func TestCacheExpires(t *testing.T) {
	fetches := 0
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		fetches++
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, ContentLength: -1, Body: io.NopCloser(strings.NewReader("[]")), Request: r}, nil
	})}
	now := time.Unix(0, 0)
	h := Handler(Deps{Client: client, Now: func() time.Time { return now }, Cache: NewMemoryCache()})
	ask := func() {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/poe/characters?realm=pc&account=a%231", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}
	}
	ask()
	now = now.Add(POE_TTL*time.Second - 1)
	ask()
	if fetches != 1 {
		t.Fatalf("%d fetches within the TTL, want 1", fetches)
	}
	now = now.Add(1)
	ask()
	if fetches != 2 {
		t.Fatalf("%d fetches at the TTL, want 2", fetches)
	}
}

// TestHandlerRoutes: the Pages Functions answer only these four routes; desk-host mounts the handler at /api/.
func TestHandlerRoutes(t *testing.T) {
	h := Handler(Deps{})
	for _, path := range []string{"/api/", "/api/poe/", "/api/poe/items/x", "/api/builds", "/other"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		if rec.Code != 404 || rec.Body.String() != "404 page not found\n" {
			t.Errorf("%s: %d %q, want the router's 404", path, rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/build?site=nope", nil))
	if rec.Code != 400 {
		t.Errorf("/api/build: status %d, want 400", rec.Code)
	}
}
