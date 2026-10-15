package host

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Store struct {
	Roots  func(bucket string) (string, bool)
	Suffix map[string]string
}

type Entry struct {
	Name     string `json:"name"`
	Dir      bool   `json:"dir"`
	LastEdit string `json:"lastEdit"`
}

func (s *Store) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bucket, rel, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	dir, ok := s.Roots(bucket)
	if !ok {
		http.Error(w, "unknown bucket "+bucket, http.StatusNotFound)
		return
	}
	rel = strings.Trim(rel, "/")
	if rel != "" && !filepath.IsLocal(rel) {
		http.Error(w, "path escapes the bucket", http.StatusBadRequest)
		return
	}
	name := rel
	if name == "" {
		name = "."
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		fail(w, err)
		return
	}
	defer root.Close()
	suffix := s.Suffix[bucket]
	file := name
	if suffix != "" && rel != "" && !strings.HasSuffix(strings.ToLower(name), suffix) {
		file = name + suffix
	}
	existing := name
	if _, err := root.Stat(file); err == nil {
		existing = file
	}

	switch r.Method {
	case http.MethodGet:
		st, err := root.Stat(existing)
		if err != nil {
			fail(w, err)
			return
		}
		if st.IsDir() && r.URL.Query().Has("size") {
			size(w, root, existing)
			return
		}
		if st.IsDir() {
			list(w, root, existing, suffix)
			return
		}
		f, err := root.Open(existing)
		if err != nil {
			fail(w, err)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeContent(w, r, "", st.ModTime(), f)
	case http.MethodHead:
		if _, err := root.Stat(existing); err != nil {
			fail(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodPut:
		if rel == "" {
			http.Error(w, "no key", http.StatusBadRequest)
			return
		}
		if err := writeAtomic(root, file, r.Body); err != nil {
			fail(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodPost:
		if err := root.MkdirAll(name, 0o755); err != nil {
			fail(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if rel == "" {
			http.Error(w, "refusing to delete a bucket", http.StatusBadRequest)
			return
		}
		if err := root.Remove(existing); err != nil {
			fail(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func list(w http.ResponseWriter, root *os.Root, name, suffix string) {
	entries, err := fs.ReadDir(root.FS(), name)
	if err != nil {
		fail(w, err)
		return
	}
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), tmpPrefix) {
			continue
		}
		key := e.Name()
		if suffix != "" && !e.IsDir() {
			if !strings.HasSuffix(strings.ToLower(key), suffix) {
				continue
			}
			key = key[:len(key)-len(suffix)]
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Entry{Name: key, Dir: e.IsDir(), LastEdit: info.ModTime().UTC().Format(time.RFC3339Nano)})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func size(w http.ResponseWriter, root *os.Root, name string) {
	var n int64
	err := fs.WalkDir(root.FS(), name, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("size %s: %w", d.Name(), err)
		}
		n += info.Size()
		return nil
	})
	if err != nil {
		fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Bytes int64 `json:"bytes"`
	}{n})
}

const tmpPrefix = ".tmp-"

func writeAtomic(root *os.Root, name string, body io.Reader) error {
	tmp := path.Join(path.Dir(filepath.ToSlash(name)), tmpPrefix+rand.Text())
	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	_, err = io.Copy(f, body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = root.Rename(tmp, name)
	}
	if err != nil {
		_ = root.Remove(tmp)
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, fs.ErrExist):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
