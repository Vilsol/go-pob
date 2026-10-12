package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"

	"github.com/Vilsol/go-pob/internal/oraclebin"
)

func TestPortableName(t *testing.T) {
	for _, name := range []string{"a.xml", "sub/c.xml", "COM10.xml", "CONSOLE.xml", "x..y.xml", ".hidden", "a_b.xml", "Hi-5_3284.WEWILLBETHERE.xml"} {
		if err := portableName(name); err != nil {
			t.Errorf("refused %q: %v", name, err)
		}
	}
	for _, name := range []string{"a|b.xml", "a:b.xml", "a<b", "a>b", `a"b`, `a\b`, "a?b", "a*b", "a\x01b", "a\x1fb",
		"trail.", "trail ", "sub./c.xml", "CON", "con.xml", "NUL.txt", "COM1", "lpt9.xml", "AUX"} {
		if err := portableName(name); err == nil {
			t.Errorf("accepted %q", name)
		}
	}
}

func TestPortableRenames(t *testing.T) {
	got, err := portableRenames([]string{"a.xml", "7428-ladder-i|c00ff00ffHi-5_3284.WEWILLBETHERE.xml", "b<>.xml"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"7428-ladder-i|c00ff00ffHi-5_3284.WEWILLBETHERE.xml": "7428-ladder-i_c00ff00ffHi-5_3284.WEWILLBETHERE.xml",
		"b<>.xml": "b__.xml",
	}
	if len(got) != len(want) {
		t.Fatalf("renames %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s -> %q, want %q", k, got[k], v)
		}
	}
	for _, names := range [][]string{{"A.xml", "a.xml"}, {"a|b.xml", "a_b.xml"}, {"x.xml", "CON.xml"}, {"y.xml", "x."}} {
		if _, err := portableRenames(names); err == nil {
			t.Errorf("accepted %v", names)
		}
	}
}

func writeGz(t *testing.T, p string, lines ...string) {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(strings.Join(lines, "\n") + "\n"))
	_ = zw.Close()
	if err := os.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPackRenamesTheCorpusAndItsOracleKeys(t *testing.T) {
	root := t.TempDir()
	corpus, oracle := filepath.Join(root, "corpus-x"), filepath.Join(root, "oracle-x")
	for _, d := range []string{corpus, oracle} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []string{"a|b.xml", "c.xml"} {
		if err := os.WriteFile(filepath.Join(corpus, n), []byte("<"+n+"/>"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	jsonl := filepath.Join(oracle, "output-x.jsonl.gz")
	writeGz(t, jsonl, `{"build":"a|b.xml","output":{"Life":1}}`, `{"build":"c.xml","output":{"Life":2}}`)
	if _, _, err := oraclebin.Regenerate(jsonl); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oracle, "picks.tsv"), []byte("c\t1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	names, err := listFiles(corpus)
	if err != nil {
		t.Fatal(err)
	}
	renames, err := portableRenames(names)
	if err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	for _, dir := range []string{"corpus-x", "oracle-x"} {
		archive := filepath.Join(root, dir+archiveExt)
		if _, _, err := packDir(filepath.Join(root, dir), dir, archive, nil, renames); err != nil {
			t.Fatal(err)
		}
		if err := unpack(archive, stage, dir); err != nil {
			t.Fatal(err)
		}
	}

	got, err := listFiles(filepath.Join(stage, "corpus-x"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "a_b.xml c.xml" {
		t.Fatalf("corpus entries %v", got)
	}
	if b, _ := os.ReadFile(filepath.Join(stage, "corpus-x", "a_b.xml")); string(b) != "<a|b.xml/>" {
		t.Fatalf("renamed entry holds %q", b)
	}

	out := filepath.Join(stage, "oracle-x", "output-x.jsonl.gz")
	fromJSONL, err := oraclebin.DecodeJSONL(out)
	if err != nil {
		t.Fatal(err)
	}
	bin, digest, err := oraclebin.DecodeFile(oraclebin.BinPath(out))
	if err != nil {
		t.Fatal(err)
	}
	if want, _ := oraclebin.DigestFile(out); digest != want {
		t.Fatal("the packed side-car is not keyed by the packed JSONL, so every reader falls back")
	}
	for _, recs := range []map[string]oraclebin.Record{fromJSONL, bin} {
		keys := make([]string, 0, len(recs))
		for k := range recs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if strings.Join(keys, " ") != "a_b.xml c.xml" || recs["a_b.xml"].Output["Life"] != 1 {
			t.Fatalf("oracle keys %v", keys)
		}
	}

	if err := os.WriteFile(filepath.Join(oracle, "picks.tsv"), []byte("a|b\t1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := packDir(oracle, "oracle-x", filepath.Join(root, "o.tar.br"), nil, renames); err == nil {
		t.Fatal("packed an oracle file naming a renamed build without rewriting it")
	}
}

func TestUnpackRefusesWindowsNames(t *testing.T) {
	for _, name := range []string{"corpus-x/a|b.xml", "corpus-x/CON.xml", "corpus-x/sub./a.xml"} {
		root := t.TempDir()
		archive := filepath.Join(root, "a.tar.br")
		var buf bytes.Buffer
		bw := brotli.NewWriter(&buf)
		tw := tar.NewWriter(bw)
		if err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: name, Mode: 0o644, Size: 1}); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write([]byte("x"))
		_ = tw.Close()
		_ = bw.Close()
		if err := os.WriteFile(archive, buf.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := unpack(archive, filepath.Join(root, "stage"), "corpus-x"); err == nil {
			t.Errorf("unpacked %q", name)
		}
	}
}

// TestInstalledTestdataIsPortable: The installed release is what a Windows or macOS checkout unpacks.
func TestInstalledTestdataIsPortable(t *testing.T) {
	root := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(root, "testdata", installedFile)); err != nil {
		t.Skip("no fetched test data release here (in-tree corpus)")
	}
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(manifestPath)))
	if err != nil {
		t.Fatal(err)
	}
	var m manifest
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	n := 0
	for asset := range m.Assets {
		dir, err := assetDir(asset)
		if err != nil {
			t.Fatal(err)
		}
		names, err := listFiles(filepath.Join(root, "testdata", dir))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := portableRenames(names); err != nil {
			t.Error(err)
		}
		for _, name := range names {
			if err := portableName(name); err != nil {
				t.Error(err)
			}
		}
		n += len(names)
	}
	if n == 0 {
		t.Fatal("the installed release holds no files")
	}
}
