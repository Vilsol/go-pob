package host

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const OpenedPrefix = "gopob-file:"

type Dialogs interface {
	SaveFile(suggested string) (string, error)
	OpenBuild() (string, error)
}

type Files struct {
	Dialogs Dialogs

	mu     sync.Mutex
	opened map[string]string
}

func (f *Files) Save(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.URL.Query().Get("name"))
	path, err := f.Dialogs.SaveFile(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if path == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var body bytes.Buffer
	if _, err := body.ReadFrom(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := os.WriteFile(path, body.Bytes(), 0o644); err != nil { //nolint:gosec // the path the user chose in the dialog
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(path)) //nolint:gosec // text/plain, read by fetch
}

func (f *Files) Opened(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	code, ok := f.opened[r.PathValue("token")]
	f.mu.Unlock()
	if !ok {
		http.Error(w, "this build file is no longer open; open it again", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(code))
}

func (f *Files) Open(path string) (string, error) {
	b, err := os.ReadFile(path) //nolint:gosec // a file the user opened
	if err != nil {
		return "", fmt.Errorf("open build file: %w", err)
	}
	code, err := BuildCode(b)
	if err != nil {
		return "", fmt.Errorf("open build file %s: %w", path, err)
	}
	token := rand.Text()
	f.mu.Lock()
	if f.opened == nil {
		f.opened = map[string]string{}
	}
	f.opened[token] = code
	f.mu.Unlock()
	return OpenedPrefix + token, nil
}

func BuildCode(b []byte) (string, error) {
	text := strings.TrimSpace(strings.TrimPrefix(string(b), "\ufeff"))
	if !strings.HasPrefix(text, "<") {
		return text, nil
	}
	var z bytes.Buffer
	zw := zlib.NewWriter(&z)
	if _, err := zw.Write([]byte(text)); err != nil {
		return "", fmt.Errorf("compress: %w", err)
	}
	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("compress: %w", err)
	}
	return base64.URLEncoding.EncodeToString(z.Bytes()), nil
}

func LaunchImport(args []string, workDir string, files *Files) string {
	for _, a := range args {
		if strings.HasPrefix(strings.ToLower(a), "pob:") {
			return a
		}
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if !filepath.IsAbs(a) && workDir != "" {
			a = filepath.Join(workDir, a)
		}
		if st, err := os.Stat(a); err != nil || st.IsDir() { //nolint:gosec // a path the launch named
			continue
		}
		if text, err := files.Open(a); err == nil {
			return text
		}
	}
	return ""
}
