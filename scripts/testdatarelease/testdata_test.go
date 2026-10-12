package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
)

func TestPackIsDeterministicAndRoundTrips(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "corpus-x")
	for name, body := range map[string]string{"b.xml": "<b/>", "a.xml": "<a/>", "sub/c.xml": "<c/>"} {
		p := filepath.Join(src, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	one, two := filepath.Join(root, "1.tar.br"), filepath.Join(root, "2.tar.br")
	sum1, n, err := packDir(src, "corpus-x", one, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("packed %d files, want 3", n)
	}
	if err := os.Chtimes(filepath.Join(src, "a.xml"), archiveTime.AddDate(3, 0, 0), archiveTime.AddDate(3, 0, 0)); err != nil {
		t.Fatal(err)
	}
	sum2, _, err := packDir(src, "corpus-x", two, nil)
	if err != nil {
		t.Fatal(err)
	}
	b1, _ := os.ReadFile(one)
	b2, _ := os.ReadFile(two)
	if sum1 != sum2 || !bytes.Equal(b1, b2) {
		t.Fatal("the same tree packed to different bytes")
	}

	stage := filepath.Join(root, "stage")
	if err := unpack(one, stage, "corpus-x"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(stage, "corpus-x", "sub", "c.xml"))
	if err != nil || string(got) != "<c/>" {
		t.Fatalf("round trip: %q, %v", got, err)
	}
}

func TestPackRefusesLFSPointers(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "a.xml"), append(lfsPointer, "\noid sha256:00\n"...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := packDir(src, "corpus-x", filepath.Join(t.TempDir(), "o.tar.br"), nil); err == nil || !strings.Contains(err.Error(), "LFS") {
		t.Fatalf("packed an LFS pointer: %v", err)
	}
}

func TestUnpackRefusesEntriesOutsideItsDir(t *testing.T) {
	for _, name := range []string{"../evil.xml", "corpus-x/../../evil.xml", "oracle-x/a.xml", "/abs.xml", "corpus-x"} {
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

func TestDownloadRefusesAChecksumMismatch(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "a.tar.br")
	if err := os.WriteFile(src, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	u := "file://" + filepath.ToSlash(src)
	if !strings.HasPrefix(filepath.ToSlash(src), "/") {
		u = "file:///" + filepath.ToSlash(src)
	}
	dst := filepath.Join(root, "out")
	const good = "239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5"
	if err := download(u, dst, good); err != nil {
		t.Fatalf("refused the right checksum: %v", err)
	}
	if err := download(u, dst, strings.Repeat("0", 64)); err == nil {
		t.Fatal("accepted a wrong checksum")
	}
	if _, err := os.Stat(dst); err == nil {
		t.Fatal("left the mismatched download behind")
	}
}

func TestExportRulesRewriteAndRefuseLeaks(t *testing.T) {
	r := &exportRules{
		rewrites: [][2]string{{"see the secret notes", "see the notes"}},
		leaks:    regexp.MustCompile(`(?i)(?:secret)|(?:\blane\b)`),
		allow:    regexp.MustCompile(`(?:Lanes of Blight)`),
	}
	if got := r.rewrite("testdata/corpus-x/MANIFEST.md", []byte("see the secret notes")); string(got) != "see the notes" {
		t.Fatalf("rewrite: %q", got)
	}
	if got := r.rewrite("testdata/corpus-x/a.xml", []byte("see the secret notes")); string(got) != "see the secret notes" {
		t.Fatalf("rewrote corpus XML: %q", got)
	}

	cases := []struct {
		name, body string
		leaks      bool
	}{
		{"MANIFEST.md", "a\nsee the secret notes\n", false},
		{"a.xml", "<x>see the secret notes</x>", true},
		{"b.xml", "<x>Lanes of Blight</x>\n<y>one lane</y>", true},
		{"c.xml", "<x>Lanes of Blight</x>", false},
		{"secret.xml", "<x/>", true},
		{"o.orb", "\x00secret", false},
	}
	for _, c := range cases {
		src := t.TempDir()
		if err := os.WriteFile(filepath.Join(src, c.name), []byte(c.body), 0o600); err != nil {
			t.Fatal(err)
		}
		names := make([]string, 1, 21)
		names[0] = c.name
		for i := range 20 {
			clean := fmt.Sprintf("clean-%02d.xml", i)
			if err := os.WriteFile(filepath.Join(src, clean), []byte("<x/>"), 0o600); err != nil {
				t.Fatal(err)
			}
			names = append(names, clean)
		}
		if err := r.checkAll(src, "corpus-x", names); (err != nil) != c.leaks {
			t.Errorf("%s: leaks=%v, checkAll returned %v", c.name, c.leaks, err)
		}
	}
}
