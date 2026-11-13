package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/loader"
	"github.com/andybalholm/brotli"
)

const fixtureVersion = "gamefile-fixture"

type fixtureCache map[string][]byte

func (c fixtureCache) Get(key string) ([]byte, error) { return c[key], nil }
func (c fixtureCache) Set(string, []byte) error       { return nil }
func (c fixtureCache) Exists(key string) bool         { _, ok := c[key]; return ok }

// manifestOnly answers the manifest, which LoadManifest fetches past the asset cache, and 404s everything else.
type manifestOnly []byte

func (m manifestOnly) RoundTrip(req *http.Request) (*http.Response, error) {
	code, body := http.StatusNotFound, []byte(nil)
	if req.URL.String() == loader.ManifestURL(fixtureVersion) {
		code, body = http.StatusOK, m
	}
	return &http.Response{StatusCode: code, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{}, Request: req}, nil
}

func brotliBytes(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := brotli.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// serveFixtureGameFiles publishes testdata/gamefiles both as per-file assets and as one bundle per extension
// listed in the manifest, as go-pob-data does.
func serveFixtureGameFiles(t *testing.T) (fixtureCache, map[string][]byte) {
	t.Helper()
	files := map[string][]byte{}
	root := os.DirFS("testdata/gamefiles")
	if err := fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if files[p], err = fs.ReadFile(root, p); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	cache := fixtureCache{}
	byExt := map[string]map[string][]byte{}
	manifest := loader.Manifest{FileBundles: map[string]string{}}
	for p, text := range files {
		cache[loader.FileURL(fixtureVersion, p)] = brotliBytes(t, text)
		ext := path.Ext(p)
		if byExt[ext] == nil {
			byExt[ext] = map[string][]byte{}
		}
		byExt[ext][p] = text
		manifest.Files = append(manifest.Files, p)
	}
	versionURL := strings.TrimSuffix(loader.ManifestURL(fixtureVersion), "manifest.json")
	for ext, group := range byExt {
		encoded := loader.EncodeFileBundle(group)
		bundle := loader.FileBundlePath(ext, encoded)
		manifest.FileBundles[ext] = bundle
		cache[versionURL+bundle] = brotliBytes(t, encoded)
	}
	slices.Sort(manifest.Files)

	b, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	previous := http.DefaultClient.Transport
	http.DefaultClient.Transport = manifestOnly(b)
	clear(fileBundles)
	t.Cleanup(func() {
		http.DefaultClient.Transport = previous
		clear(fileBundles)
	})
	return cache, files
}

func TestGameFilesMatchPerFileAssets(t *testing.T) {
	cache, files := serveFixtureGameFiles(t)
	for _, ext := range []string{".it", ".ot", ".txt"} {
		bundle, err := gameFiles(fixtureVersion, ext, cache)
		if err != nil {
			t.Fatal(err)
		}
		var want []string
		for p := range files {
			if path.Ext(p) == ext {
				want = append(want, p)
			}
		}
		got := slices.Sorted(maps.Keys(bundle))
		slices.Sort(want)
		if len(want) == 0 || !slices.Equal(want, got) {
			t.Errorf("%s bundle holds %v, want %v", ext, got, want)
		}
		for _, p := range want {
			perFile, err := loader.LoadFile(gameDataContext(), fixtureVersion, p, cache)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(perFile, bundle[p]) {
				t.Errorf("%s: bundle %q, per-file %q", p, bundle[p], perFile)
			}
		}
	}

	if _, err := gameFiles(fixtureVersion, ".dat", cache); !errors.Is(err, loader.ErrAssetNotFound) {
		t.Errorf("a missing bundle: got %v, want ErrAssetNotFound", err)
	}
}

func TestCallSitesReadBundles(t *testing.T) {
	cache, _ := serveFixtureGameFiles(t)
	its, err := gameFiles(fixtureVersion, ".it", cache)
	if err != nil {
		t.Fatal(err)
	}
	b := &basesExport{tags: map[string][]string{}, its: its}
	if got := b.baseItemTags("Metadata/Items/Weapons/Sword"); !slices.Equal(got, []string{"sword"}) {
		t.Errorf("Sword tags %v, want [sword]", got)
	}
	if got := b.baseItemTags("Metadata/Items/Absent"); got != nil {
		t.Errorf("a missing .it file: tags %v, want nil", got)
	}

	if _, err := gameFiles(fixtureVersion, ".ot", cache); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := otConstants(fixtureVersion, "Metadata/Characters/Character.ot", []string{"Stats"}, &out); err != nil {
		t.Fatal(err)
	}
	if want := "\t[\"base_maximum_life\"] = 38,\n\t[\"base_movement_velocity\"] = 100,\n"; out.String() != want {
		t.Errorf("Character.ot constants %q, want %q", out.String(), want)
	}
	if err := otConstants(fixtureVersion, "Metadata/Monsters/Absent.ot", []string{"Stats"}, &out); !errors.Is(err, loader.ErrAssetNotFound) {
		t.Errorf("a missing .ot file: got %v, want ErrAssetNotFound", err)
	}
}
