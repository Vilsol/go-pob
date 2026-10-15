package host

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type scripted struct{ save, open string }

func (s scripted) SaveFile(string) (string, error) { return s.save, nil }
func (s scripted) OpenBuild() (string, error)      { return s.open, nil }

func decode(t *testing.T, code string) string {
	t.Helper()
	code = strings.NewReplacer("-", "+", "_", "/").Replace(code)
	r, err := zlib.NewReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(code)))
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBuildCode(t *testing.T) {
	xml := "<?xml version=\"1.0\"?>\n<PathOfBuilding><Build level=\"90\"/></PathOfBuilding>"
	code, err := BuildCode([]byte("\ufeff" + xml + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(code, "+/") {
		t.Errorf("not URL-safe: %s", code)
	}
	if got := decode(t, code); got != xml {
		t.Errorf("round trip: %q", got)
	}
	if code2, _ := BuildCode([]byte("  eNoLcvVz8QQABIYBug==\n")); code2 != "eNoLcvVz8QQABIYBug==" {
		t.Errorf("a code file is its trimmed text: %q", code2)
	}
}

func TestFilesSaveAndOpen(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "export.zip")
	f := &Files{Dialogs: scripted{save: target}}

	rec := httptest.NewRecorder()
	f.Save(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/save?name=../x.zip", bytes.NewReader([]byte("zip bytes"))))
	if rec.Code != 200 || rec.Body.String() != target {
		t.Fatalf("save: %d %q", rec.Code, rec.Body.String())
	}
	if b, _ := os.ReadFile(target); string(b) != "zip bytes" {
		t.Fatalf("saved %q", b)
	}

	f.Dialogs = scripted{}
	rec = httptest.NewRecorder()
	f.Save(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/save?name=x.zip", bytes.NewReader([]byte("other"))))
	if rec.Code != 204 {
		t.Fatalf("cancelled save: %d", rec.Code)
	}

	build := filepath.Join(dir, "My Build.xml")
	if err := os.WriteFile(build, []byte("<PathOfBuilding/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	text, err := f.Open(build)
	if err != nil || !strings.HasPrefix(text, OpenedPrefix) {
		t.Fatalf("open: %q %v", text, err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /opened/{token}", f.Opened)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/opened/"+strings.TrimPrefix(text, OpenedPrefix), nil))
	if rec.Code != 200 || decode(t, rec.Body.String()) != "<PathOfBuilding/>" {
		t.Fatalf("opened: %d %q", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/opened/nope", nil))
	if rec.Code != 404 {
		t.Fatalf("unknown token: %d", rec.Code)
	}

	if got := LaunchImport([]string{"--flag", "My Build.xml"}, dir, f); !strings.HasPrefix(got, OpenedPrefix) {
		t.Errorf("relative file launch: %q", got)
	}
	if got := LaunchImport([]string{build, "pob://pobbin/abc"}, "", f); got != "pob://pobbin/abc" {
		t.Errorf("a link wins: %q", got)
	}
	if got := LaunchImport([]string{"missing.xml", dir}, dir, f); got != "" {
		t.Errorf("nothing to open: %q", got)
	}
}
