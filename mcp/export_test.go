package mcp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

func TestExportBuildToPathReturnsOnlyPaths(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	for _, format := range []string{"xml", "code"} {
		path := filepath.Join(t.TempDir(), "out."+format)
		res := assertSize(t, s, "export_build", map[string]any{"handle": h, "format": format, "path": path}, 2000)
		if res["xml"] != nil || res["code"] != nil {
			t.Fatalf("%s export to a path returned the body: keys %v", format, keys(res))
		}
		written, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		files := res["files"].([]any)
		if len(files) != 1 || files[0].(map[string]any)["path"] != path || files[0].(map[string]any)["bytes"] != float64(len(written)) {
			t.Fatalf("%s files = %v", format, files)
		}
		if format == "code" {
			if _, err := pob.DecodeDecompress(string(written)); err != nil {
				t.Fatalf("the code file does not decode: %v", err)
			}
		} else if !strings.HasPrefix(string(written), "<PathOfBuilding") {
			t.Fatalf("the xml file starts %q", string(written[:20]))
		}
	}
}

func TestExportBuildWritesXMLAndCodeTogether(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	dir := t.TempDir()
	res := call(t, s, "export_build", map[string]any{"handle": h, "format": "both", "path": filepath.Join(dir, "b.xml")})
	files := res["files"].([]any)
	if len(files) != 2 {
		t.Fatalf("files = %v", files)
	}
	for _, want := range []string{"b.xml", "b.txt"} {
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Fatalf("%s not written: %v", want, err)
		}
	}
	inline := call(t, s, "export_build", map[string]any{"handle": h, "format": "both"})
	if inline["xml"] == nil || inline["code"] == nil {
		t.Fatalf("both without a path = keys %v", keys(inline))
	}
}

func TestExportDirNamesDifferAcrossServers(t *testing.T) {
	a, b := testServer(t), testServer(t)
	dir := t.TempDir()
	a.opts.ExportDir, b.opts.ExportDir = dir, dir
	ha, hb := importFixture(t, a), importFixture(t, b)
	fa := call(t, a, "export_build", map[string]any{"handle": ha})["export_file"].(string)
	fb := call(t, b, "export_build", map[string]any{"handle": hb})["export_file"].(string)
	name := regexp.MustCompile(`^.+-0001-b1-r0\.xml$`)
	if !name.MatchString(filepath.Base(fa)) || !name.MatchString(filepath.Base(fb)) || filepath.Base(fa) == filepath.Base(fb) {
		t.Fatalf("export names %s and %s", filepath.Base(fa), filepath.Base(fb))
	}
}
