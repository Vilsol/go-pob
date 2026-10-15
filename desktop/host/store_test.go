package host

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStore(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	s := &Store{Roots: func(bucket string) (string, bool) { return root, bucket == "builds" }}
	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
		return rec
	}

	if c := do(http.MethodHead, "/builds/a.xml", "").Code; c != 404 {
		t.Fatalf("exists before write: %d", c)
	}
	if c := do(http.MethodPut, "/builds/a.xml", "<PathOfBuilding/>").Code; c != 204 {
		t.Fatalf("write: %d", c)
	}
	if c := do(http.MethodHead, "/builds/a.xml", "").Code; c != 204 {
		t.Fatalf("exists after write: %d", c)
	}
	if rec := do(http.MethodGet, "/builds/a.xml", ""); rec.Code != 200 || rec.Body.String() != "<PathOfBuilding/>" {
		t.Fatalf("get: %d %q", rec.Code, rec.Body.String())
	}
	if c := do(http.MethodPut, "/builds/missing/b.xml", "x").Code; c != 404 {
		t.Fatalf("write into a missing folder: %d, want 404 as zenfs fails", c)
	}
	if c := do(http.MethodPost, "/builds/Folder", "").Code; c != 204 {
		t.Fatalf("mkdir: %d", c)
	}
	if c := do(http.MethodPut, "/builds/Folder/b.xml", "x").Code; c != 204 {
		t.Fatalf("write into folder: %d", c)
	}

	rec := do(http.MethodGet, "/builds/", "")
	var entries []Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("list: %v %q", err, rec.Body.String())
	}
	if len(entries) != 2 || entries[0].Name != "Folder" || !entries[0].Dir || entries[1].Name != "a.xml" || entries[1].Dir {
		t.Fatalf("list: %+v", entries)
	}

	if c := do(http.MethodDelete, "/builds/a.xml", "").Code; c != 204 {
		t.Fatalf("delete: %d", c)
	}
	if _, err := os.Stat(filepath.Join(root, "a.xml")); !os.IsNotExist(err) {
		t.Fatalf("a.xml still there: %v", err)
	}

	for _, p := range []string{"/builds/../outside.txt", "/builds/Folder/../../outside.txt"} {
		if c := do(http.MethodPut, p, "x").Code; c != 400 {
			t.Errorf("%s: %d, want 400", p, c)
		}
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("wrote outside the bucket: %v", err)
	}
	if c := do(http.MethodGet, "/nope/a", "").Code; c != 404 {
		t.Errorf("unknown bucket: %d", c)
	}
	if c := do(http.MethodDelete, "/builds/", "").Code; c != 400 {
		t.Errorf("delete the bucket: %d", c)
	}
}

// TestStoreBuildSuffix: builds are <name>.xml on disk, as PoB writes and lists them, and bare names to go-pob.
func TestStoreBuildSuffix(t *testing.T) {
	root := t.TempDir()
	s := &Store{Roots: func(string) (string, bool) { return root, true }, Suffix: map[string]string{"builds": ".xml"}}
	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
		return rec
	}
	if c := do(http.MethodPut, "/builds/My%20build", "<PathOfBuilding/>").Code; c != 204 {
		t.Fatalf("write: %d", c)
	}
	if _, err := os.Stat(filepath.Join(root, "My build.xml")); err != nil {
		t.Fatalf("not written as My build.xml: %v", err)
	}
	if rec := do(http.MethodGet, "/builds/My%20build", ""); rec.Code != 200 || rec.Body.String() != "<PathOfBuilding/>" {
		t.Fatalf("get: %d %q", rec.Code, rec.Body.String())
	}
	for _, f := range []string{"notes.txt", "Folder/x.xml"} {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(root, f)), 0o755)
		_ = os.WriteFile(filepath.Join(root, f), []byte("x"), 0o600)
	}
	if c := do(http.MethodHead, "/builds/Folder", "").Code; c != 204 {
		t.Fatalf("a folder exists: %d", c)
	}
	var entries []Entry
	_ = json.Unmarshal(do(http.MethodGet, "/builds/", "").Body.Bytes(), &entries)
	if len(entries) != 2 || entries[0].Name != "Folder" || entries[1].Name != "My build" {
		t.Fatalf("list: %+v", entries)
	}
	if c := do(http.MethodPut, "/cache/abc", "data").Code; c != 204 {
		t.Fatalf("cache write: %d", c)
	}
	if _, err := os.Stat(filepath.Join(root, "abc")); err != nil {
		t.Fatalf("cache keys carry no suffix: %v", err)
	}
	if c := do(http.MethodDelete, "/builds/My%20build", "").Code; c != 204 {
		t.Fatalf("delete: %d", c)
	}
	if _, err := os.Stat(filepath.Join(root, "My build.xml")); !os.IsNotExist(err) {
		t.Fatalf("My build.xml still there: %v", err)
	}
}

// TestStoreSize: Settings › Cached data: GET <bucket>/?size is the bytes of every file under the bucket's folder.
func TestStoreSize(t *testing.T) {
	root := t.TempDir()
	s := &Store{Roots: func(bucket string) (string, bool) { return root, bucket == "cache" }}
	if err := os.MkdirAll(filepath.Join(root, "data", "3.29"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, n := range map[string]int{"a.bin": 1000, "data/3.29/b.json": 24} {
		if err := os.WriteFile(filepath.Join(root, name), make([]byte, n), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/cache/?size", nil))
	var got struct{ Bytes int64 }
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != 200 || got.Bytes != 1024 {
		t.Fatalf("size: %d %q %v", rec.Code, rec.Body.String(), err)
	}
}
