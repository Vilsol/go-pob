package host

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestSPA(t *testing.T) {
	assets := fstest.MapFS{
		"index.html":                 {Data: []byte("unrewritten")},
		"go-pob.wasm.gz":             {Data: []byte("gz")},
		"_app/immutable/entry/a.js":  {Data: []byte("js")},
		"tree-rings/ring.png":        {Data: []byte("png")},
		"_app/immutable/nodes/0.css": {Data: []byte("css")},
	}
	h := SPA(assets, []byte("shell"))
	for _, c := range []struct {
		path, body string
		code       int
		ctype      string
	}{
		{"/", "shell", 200, "text/html; charset=utf-8"},
		{"/tree", "shell", 200, "text/html; charset=utf-8"},
		{"/items/", "shell", 200, "text/html; charset=utf-8"},
		{"/tree-rings", "shell", 200, "text/html; charset=utf-8"},
		{"/_app/immutable/entry/a.js", "js", 200, "text/javascript; charset=utf-8"},
		{"/go-pob.wasm.gz", "gz", 200, "application/gzip"},
		{"/missing.js", "", 404, ""},
		{"/../index.html", "shell", 200, ""},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, c.path, nil))
		if rec.Code != c.code {
			t.Errorf("%s: code %d, want %d", c.path, rec.Code, c.code)
			continue
		}
		if c.code != 200 {
			continue
		}
		if rec.Body.String() != c.body {
			t.Errorf("%s: body %q, want %q", c.path, rec.Body.String(), c.body)
		}
		if c.ctype != "" && rec.Header().Get("Content-Type") != c.ctype {
			t.Errorf("%s: Content-Type %q, want %q", c.path, rec.Header().Get("Content-Type"), c.ctype)
		}
		if enc := rec.Header().Get("Content-Encoding"); enc != "" {
			t.Errorf("%s: Content-Encoding %q", c.path, enc)
		}
	}
}

func TestRoutes(t *testing.T) {
	tag := func(name string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(name + " " + r.URL.Path)) }) //nolint:gosec // a test echo
	}
	h := Routes(tag("api"), tag("native"))(tag("assets"))
	for path, want := range map[string]string{
		"/api/build":     "api /api/build",
		"/desktop/fs/x":  "native /fs/x",
		"/tree":          "assets /tree",
		"/apiary":        "assets /apiary",
		"/desktopx/a.js": "assets /desktopx/a.js",
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		if rec.Body.String() != want {
			t.Errorf("%s: %q, want %q", path, rec.Body.String(), want)
		}
	}
}
