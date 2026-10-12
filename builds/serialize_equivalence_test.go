package builds

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func serializeReference(t *testing.T, xmlText []byte) []byte {
	t.Helper()
	build, err := ParseBuild(xmlText)
	if err != nil {
		t.Fatal(err)
	}
	b, err := xml.Marshal(build)
	if err != nil {
		t.Fatal(err)
	}
	r := strings.NewReplacer("&#xA;", "\n", "&#x9;", "\t", "&#34;", "&quot;", "&#39;", "&apos;")
	return []byte(r.Replace(string(b)))
}

// TestSerializeBuildMatchesTheReference: on every 25th corpus build, largest first so the size hint is exercised shrinking.
func TestSerializeBuildMatchesTheReference(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "testdata", "corpus-3.29", "*.xml"))
	if err != nil || len(files) < 1000 {
		t.Fatalf("REFUSING: %d corpus files (%v)", len(files), err)
	}
	picked := []string{filepath.Join("..", "testdata", "corpus-3.29", "0001-necromancer-unknown.xml")}
	for i := 0; i < len(files); i += 25 {
		picked = append(picked, files[i])
	}
	entities := 0
	for _, f := range picked {
		x, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		want := serializeReference(t, x)
		build, err := ParseBuild(x)
		if err != nil {
			t.Fatal(err)
		}
		got, err := SerializeBuild(build)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s: serialisation differs from the reference", filepath.Base(f))
		}
		entities += bytes.Count(want, []byte("&quot;")) + bytes.Count(want, []byte("&apos;")) + bytes.Count(want, []byte("\n"))
	}
	if entities == 0 {
		t.Fatal("REFUSING: no build carried an entity the rewrite changes")
	}
}

func TestReplacePoBEntities(t *testing.T) {
	in := `a&#xA;b&#x9;c&#34;d&#39;e&amp;f&#xD;g&#3&#34`
	want := "a\nb\tc&quot;d&apos;e&amp;f&#xD;g&#3&#34"
	if got := string(replacePoBEntities([]byte(in))); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestPoBEntityWriterSplits: The writer the encoder writes through rewrites as replacePoBEntities does on the whole document.
func TestPoBEntityWriterSplits(t *testing.T) {
	for _, in := range []string{
		`a&#xA;b&#x9;c&#34;d&#39;e&amp;f&#xD;g&#3&#34`,
		`&#&#xA;&&#34;&#3&#39;&#x9&#xA`,
		`&#39;&#39;&&#34;x&#34;`,
		`&`,
		``,
	} {
		want := string(replacePoBEntities([]byte(in)))
		for i := 0; i <= len(in); i++ {
			for j := i; j <= len(in); j++ {
				w := &pobEntityWriter{}
				for _, part := range []string{in[:i], in[i:j], in[j:]} {
					if _, err := w.Write([]byte(part)); err != nil {
						t.Fatal(err)
					}
				}
				if got := string(w.finish()); got != want {
					t.Fatalf("%q split at %d, %d: got %q, want %q", in, i, j, got, want)
				}
			}
		}
	}
}
