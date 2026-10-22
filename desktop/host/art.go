package host

import (
	"context"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Art serves the game's item art from Dir, fetching what it lacks from Upstream and keeping it; the page
// shows its text fallback for anything it answers 404.
type Art struct {
	Dir      string
	Upstream string
	Client   *http.Client
}

const artMaxBytes = 4 << 20

func (a *Art) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if r.Method != http.MethodGet || !strings.HasPrefix(rel, "Art/") || !strings.HasSuffix(rel, ".png") || !filepath.IsLocal(rel) {
		http.NotFound(w, r)
		return
	}
	file := filepath.Join(a.Dir, filepath.FromSlash(rel))
	if b, err := os.ReadFile(file); err == nil {
		serveArt(w, b)
		return
	}
	b, ok := a.fetch(r.Context(), rel)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err == nil {
		tmp := file + ".part"
		if os.WriteFile(tmp, b, 0o644) == nil {
			_ = os.Rename(tmp, file)
		}
	}
	serveArt(w, b)
}

func (a *Art) fetch(ctx context.Context, rel string) ([]byte, bool) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.Upstream+rel, nil)
	if err != nil {
		return nil, false
	}
	client := a.Client
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, false
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(res.Header.Get("Content-Type"), "image/") {
		return nil, false
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, artMaxBytes+1))
	if err != nil || len(b) > artMaxBytes {
		return nil, false
	}
	return b, true
}

func serveArt(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(b)
}
