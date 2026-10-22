package host

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Art serves the game's item art from Dir, fetching what it lacks from Upstream and keeping it; the page
// shows its text fallback for anything it answers 404. It keeps at most MaxFiles images, dropping the least
// recently served, and DELETE on its root empties it.
type Art struct {
	Dir      string
	Upstream string
	Client   *http.Client
	MaxFiles int
	mu       sync.Mutex
}

const artMaxBytes = 4 << 20

// ArtMaxFiles matches the web app's art cache.
const ArtMaxFiles = 200

func (a *Art) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if r.Method == http.MethodDelete && rel == "" {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, strconv.Itoa(a.Clear()))
		return
	}
	if r.Method != http.MethodGet || !strings.HasPrefix(rel, "Art/") || !strings.HasSuffix(rel, ".png") || !filepath.IsLocal(rel) {
		http.NotFound(w, r)
		return
	}
	file := filepath.Join(a.Dir, filepath.FromSlash(rel))
	if b, err := os.ReadFile(file); err == nil {
		now := time.Now()
		_ = os.Chtimes(file, now, now)
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
		a.prune()
	}
	serveArt(w, b)
}

type artFile struct {
	path string
	mod  time.Time
}

func (a *Art) files() []artFile {
	var out []artFile
	_ = filepath.WalkDir(a.Dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".png") {
			return nil
		}
		if info, err := d.Info(); err == nil {
			out = append(out, artFile{p, info.ModTime()})
		}
		return nil
	})
	return out
}

func (a *Art) prune() {
	limit := a.MaxFiles
	if limit <= 0 {
		limit = ArtMaxFiles
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	files := a.files()
	if len(files) <= limit {
		return
	}
	slices.SortFunc(files, func(x, y artFile) int { return x.mod.Compare(y.mod) })
	for _, f := range files[:len(files)-limit] {
		_ = os.Remove(f.path)
	}
}

// Clear removes every kept image and returns how many there were.
func (a *Art) Clear() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := len(a.files())
	_ = os.RemoveAll(a.Dir)
	return n
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
