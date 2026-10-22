package host

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestArt(t *testing.T) {
	hits := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch r.URL.Path {
		case "/image/Art/2DItems/Belts/Headhunter.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("png"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	dir := t.TempDir()
	art := &Art{Dir: dir, Upstream: upstream.URL + "/image/"}
	get := func(p string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		art.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		return rec
	}

	if rec := get("/Art/2DItems/Belts/Headhunter.png"); rec.Code != 200 || rec.Body.String() != "png" || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("first fetch: %d %q %q", rec.Code, rec.Body.String(), rec.Header().Get("Content-Type"))
	}
	if b, err := os.ReadFile(filepath.Join(dir, "Art", "2DItems", "Belts", "Headhunter.png")); err != nil || string(b) != "png" {
		t.Fatalf("not kept on disk: %q %v", b, err)
	}
	upstream.Close()
	if rec := get("/Art/2DItems/Belts/Headhunter.png"); rec.Code != 200 || rec.Body.String() != "png" {
		t.Fatalf("offline, from disk: %d %q", rec.Code, rec.Body.String())
	}
	if rec := get("/Art/2DItems/Belts/Other.png"); rec.Code != 404 {
		t.Fatalf("offline, not on disk: %d, want 404", rec.Code)
	}
	if hits != 1 {
		t.Fatalf("upstream hit %d times, want 1", hits)
	}
}

func TestArtRefuses(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/image/Art/Page.png" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html>"))
			return
		}
		http.NotFound(w, r)
	}))
	defer upstream.Close()
	dir := t.TempDir()
	art := &Art{Dir: dir, Upstream: upstream.URL + "/image/"}
	for _, p := range []string{"/Art/Missing.png", "/Art/Page.png", "/Other/x.png", "/Art/x.jpg", "/Art/../../etc/passwd.png", "/../Art/x.png"} {
		rec := httptest.NewRecorder()
		art.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != 404 {
			t.Errorf("%s: %d, want 404", p, rec.Code)
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("kept %d entries for refused requests", len(entries))
	}
}
