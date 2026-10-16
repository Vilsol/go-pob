package host

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

type Updater interface {
	CurrentVersion() string
	Check(ctx context.Context) (*updater.Release, error)
	DownloadAndInstall(ctx context.Context) error
	Restart(ctx context.Context) error
}

type Updates struct {
	Updater    Updater
	CanRestart bool
	Manager    string
}

type UpdateCheck struct {
	Enabled bool   `json:"enabled"`
	Manager string `json:"manager,omitempty"`
	Current string `json:"current,omitempty"`
	Version string `json:"version,omitempty"`
	Name    string `json:"name,omitempty"`
	Notes   string `json:"notes,omitempty"`
	Page    string `json:"page,omitempty"`
	Install bool   `json:"install"`
	Error   string `json:"error,omitempty"`
}

func (u *Updates) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /update/check", u.check)
	mux.HandleFunc("POST /update/install", u.install)
	mux.HandleFunc("POST /update/restart", u.restart)
}

func (u *Updates) check(w http.ResponseWriter, r *http.Request) {
	if u.Updater == nil {
		writeJSON(w, http.StatusOK, UpdateCheck{Manager: u.Manager})
		return
	}
	c := UpdateCheck{Enabled: true, Current: u.Updater.CurrentVersion(), Install: u.CanRestart}
	rel, err := u.Updater.Check(r.Context())
	switch {
	case err != nil:
		c.Error = err.Error()
		writeJSON(w, http.StatusBadGateway, c)
		return
	case rel != nil:
		c.Version, c.Name, c.Notes = rel.Version, rel.Name, rel.Notes
		c.Page, _ = rel.Metadata["page"].(string)
	}
	writeJSON(w, http.StatusOK, c)
}

func (u *Updates) install(w http.ResponseWriter, r *http.Request) {
	if u.Updater == nil {
		msg := "this build has no update source"
		if u.Manager != "" {
			msg = "updates for this build come from " + u.Manager
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": msg})
		return
	}
	if err := u.Updater.DownloadAndInstall(r.Context()); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ready": true})
}

func (u *Updates) restart(w http.ResponseWriter, r *http.Request) {
	if u.Updater == nil || !u.CanRestart {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "this install cannot replace itself"})
		return
	}
	if err := u.Updater.Restart(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type SignedOnly struct{ updater.Provider }

func (s SignedOnly) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	rel, err := s.Provider.Check(ctx, req)
	if err != nil || rel == nil {
		return rel, err //nolint:wrapcheck // the updater names the provider in its own error
	}
	if rel.Verification == nil || len(rel.Verification.Signature) == 0 {
		return nil, fmt.Errorf("release %s is not signed", rel.Version)
	}
	return rel, nil
}

func ReplaceDirs(exe, goos string) []string {
	dirs := []string{filepath.Dir(exe)}
	if goos != "darwin" {
		return dirs
	}
	for dir := filepath.Dir(exe); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if strings.HasSuffix(dir, ".app") {
			return append(dirs, filepath.Dir(dir))
		}
	}
	return dirs
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
