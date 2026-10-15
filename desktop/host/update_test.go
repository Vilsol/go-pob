package host

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

type updaterHost struct{}

func (updaterHost) Emit(string, ...any) bool                              { return true }
func (updaterHost) OnEvent(string, func(any)) func()                      { return func() {} }
func (updaterHost) OpenWindow(updater.WindowOptions) updater.WindowHandle { return nil }
func (updaterHost) Quit()                                                 {}

type release struct {
	manifest map[string]any
	artifact []byte
}

func signedRelease(t *testing.T, priv ed25519.PrivateKey, version string, artifact []byte) *release {
	t.Helper()
	digest := sha512.Sum512(artifact)
	sig, err := priv.Sign(nil, digest[:], &ed25519.Options{Hash: crypto.SHA512})
	if err != nil {
		t.Fatal(err)
	}
	return &release{artifact: artifact, manifest: map[string]any{
		"schemaVersion": 1, "version": version, "name": "go-pob " + version, "metadata": map[string]any{"page": "https://example.invalid/releases/v" + version},
		"artifacts": []map[string]any{{
			"url": "go-pob-x86_64.AppImage", "platform": "linux", "arch": "amd64", "size": len(artifact),
			"digestAlgo": "sha512", "digest": base64.StdEncoding.EncodeToString(digest[:]),
			"signatureAlgo": "ed25519ph", "signature": base64.StdEncoding.EncodeToString(sig),
		}},
	}}
}

func (r *release) artifactField(k string, v any) {
	r.manifest["artifacts"].([]map[string]any)[0][k] = v
}

func (r *release) serve(t *testing.T) string {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /update.json", func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(r.manifest) })
	mux.HandleFunc("GET /go-pob-x86_64.AppImage", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(r.artifact) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL + "/update.json"
}

func newUpdates(t *testing.T, url, current string, pub ed25519.PublicKey) http.Handler {
	t.Helper()
	p, err := endpoint.New(endpoint.Config{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	u := updater.New(updaterHost{})
	if err := u.Init(updater.Config{CurrentVersion: current, Providers: []updater.Provider{SignedOnly{p}}, PublicKey: pub, Platform: "linux", Arch: "amd64", Window: updater.WindowNone}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if path := u.DownloadedPath(); path != "" {
			_ = os.RemoveAll(path)
		}
	})
	mux := http.NewServeMux()
	(&Updates{Updater: u}).Register(mux)
	return mux
}

func call(t *testing.T, h http.Handler, method, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, nil))
	return rec.Code, rec.Body.String()
}

func TestUpdates(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	newer := []byte("go-pob 1.1.0 AppImage")
	for _, tc := range []struct {
		name, current string
		tamper        func(r *release)
		check         string
		install       string
	}{
		{name: "up to date", current: "1.1.0", check: `{"enabled":true,"current":"1.1.0","install":false}`},
		{name: "signed newer", current: "1.0.0", check: `"version":"1.1.0","name":"go-pob 1.1.0","page":"https://example.invalid/releases/v1.1.0"`, install: `{"ready":true}`},
		{
			name: "artifact swapped", current: "1.0.0", check: `"version":"1.1.0"`, install: "digest mismatch",
			tamper: func(r *release) { r.artifact = []byte("go-pob 1.1.0 AppImagE") },
		},
		{
			name: "artifact swapped, digest rewritten", current: "1.0.0", check: `"version":"1.1.0"`, install: "ed25519ph signature did not verify",
			tamper: func(r *release) {
				r.artifact = []byte("go-pob 1.1.0 AppImagE")
				d := sha512.Sum512(r.artifact)
				r.artifactField("digest", base64.StdEncoding.EncodeToString(d[:]))
			},
		},
		{
			name: "signature stripped", current: "1.0.0", check: "release 1.1.0 is not signed",
			tamper: func(r *release) { r.artifactField("signature", ""); r.artifactField("signatureAlgo", "") },
		},
		{
			name: "signed by another key", current: "1.0.0", check: `"version":"1.1.0"`, install: "ed25519ph signature did not verify",
			tamper: func(r *release) {
				_, other, _ := ed25519.GenerateKey(rand.Reader)
				*r = *signedRelease(t, other, "1.1.0", r.artifact)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rel := signedRelease(t, priv, "1.1.0", newer)
			if tc.tamper != nil {
				tc.tamper(rel)
			}
			h := newUpdates(t, rel.serve(t), tc.current, pub)
			code, body := call(t, h, http.MethodGet, "/update/check")
			if !strings.Contains(body, tc.check) {
				t.Fatalf("check %d %s, want %s", code, body, tc.check)
			}
			if tc.install == "" {
				return
			}
			code, body = call(t, h, http.MethodPost, "/update/install")
			if !strings.Contains(body, tc.install) || (tc.install == `{"ready":true}`) != (code == http.StatusOK) {
				t.Fatalf("install %d %s, want %s", code, body, tc.install)
			}
		})
	}
}

func TestUpdatesDisabled(t *testing.T) {
	mux := http.NewServeMux()
	(&Updates{}).Register(mux)
	if code, body := call(t, mux, http.MethodGet, "/update/check"); code != http.StatusOK || strings.TrimSpace(body) != `{"enabled":false,"install":false}` {
		t.Fatalf("check %d %s", code, body)
	}
	if code, _ := call(t, mux, http.MethodPost, "/update/restart"); code != http.StatusConflict {
		t.Fatalf("restart %d", code)
	}
}

func TestReplaceDirs(t *testing.T) {
	for _, c := range []struct {
		exe, goos string
		want      []string
	}{
		{"/Applications/go-pob.app/Contents/MacOS/go-pob", "darwin", []string{"/Applications/go-pob.app/Contents/MacOS", "/Applications"}},
		{"/Users/me/Apps/go-pob.app/Contents/MacOS/go-pob", "darwin", []string{"/Users/me/Apps/go-pob.app/Contents/MacOS", "/Users/me/Apps"}},
		{"/Users/me/bin/go-pob", "darwin", []string{"/Users/me/bin"}},
		{"/opt/x.app/go-pob", "linux", []string{"/opt/x.app"}},
		{"/home/me/bin/go-pob", "linux", []string{"/home/me/bin"}},
	} {
		if got := ReplaceDirs(c.exe, c.goos); !slices.Equal(got, c.want) {
			t.Errorf("%s (%s): %q, want %q", c.exe, c.goos, got, c.want)
		}
	}
}

// TestUpdatesPerPlatform: The release job's update.json names one artifact per platform the updater can replace.
func TestUpdatesPerPlatform(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	r := signedRelease(t, priv, "1.1.0", []byte("x"))
	base := r.manifest["artifacts"].([]map[string]any)[0]
	artifacts := make([]map[string]any, 0, 4)
	for _, a := range [][3]string{
		{"go-pob-linux-amd64", "linux", "amd64"},
		{"go-pob-windows-amd64.zip", "windows", "amd64"},
		{"go-pob-macos-arm64.zip", "darwin", "arm64"},
		{"go-pob-macos-amd64.zip", "darwin", "amd64"},
	} {
		art := maps.Clone(base)
		art["url"], art["platform"], art["arch"] = a[0], a[1], a[2]
		artifacts = append(artifacts, art)
	}
	r.manifest["artifacts"] = artifacts
	p, err := endpoint.New(endpoint.Config{URL: r.serve(t)})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range [][3]string{
		{"linux", "amd64", "go-pob-linux-amd64"},
		{"windows", "amd64", "go-pob-windows-amd64.zip"},
		{"darwin", "arm64", "go-pob-macos-arm64.zip"},
		{"darwin", "amd64", "go-pob-macos-amd64.zip"},
	} {
		rel, err := SignedOnly{p}.Check(t.Context(), updater.CheckRequest{CurrentVersion: "1.0.0", Platform: c[0], Arch: c[1]})
		if err != nil || rel == nil || rel.Artifact.Filename != c[2] {
			t.Errorf("%s/%s: %+v, %v; want %s", c[0], c[1], rel, err, c[2])
		}
	}
	if rel, err := (SignedOnly{p}).Check(t.Context(), updater.CheckRequest{CurrentVersion: "1.0.0", Platform: "windows", Arch: "arm64"}); err == nil {
		t.Errorf("windows/arm64 has no artifact: %+v", rel)
	}
}
