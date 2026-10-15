package host

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDesktopIndex(t *testing.T) {
	app, err := os.ReadFile("../../frontend/src/app.html")
	if err != nil {
		t.Fatal(err)
	}
	out, err := DesktopIndex(app)
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"fonts.googleapis.com", "fonts.gstatic.com", `rel="manifest"`} {
		if bytes.Contains(out, []byte(gone)) {
			t.Errorf("still has %s", gone)
		}
	}
	if bytes.Count(out, []byte(FontsCSS)) != 1 {
		t.Errorf("want one %s link", FontsCSS)
	}
	if !bytes.Contains(out, []byte("%sveltekit.head%")) || !bytes.Contains(out, []byte(`<link rel="apple-touch-icon"`)) {
		t.Error("removed more than the font and manifest links")
	}
	if _, err := DesktopIndex([]byte("<html></html>")); err == nil {
		t.Error("an index without the links must fail")
	}
}

// TestBundledFontsMatchAppHTML: fonts/fetch.sh records the links it fetched on fonts.css's first line; app.html linking anything else means the bundle is stale.
func TestBundledFontsMatchAppHTML(t *testing.T) {
	app, err := os.ReadFile("../../frontend/src/app.html")
	if err != nil {
		t.Fatal(err)
	}
	css, err := os.ReadFile("../fonts/fonts.css")
	if err != nil {
		t.Fatal(err)
	}
	links := regexp.MustCompile(`https://fonts\.googleapis\.com/css2\?[^"]*`).FindAllString(string(app), -1)
	first, _, _ := strings.Cut(string(css), "\n")
	if want := "/* from: " + strings.Join(links, " ") + " */"; first != want {
		t.Fatalf("fonts.css was fetched from\n%s\nbut app.html links\n%s\nre-run desktop/fonts/fetch.sh", first, want)
	}
	for _, m := range regexp.MustCompile(`url\(([^)]+)\)`).FindAllStringSubmatch(string(css), -1) {
		if _, err := os.Stat(filepath.Join("../fonts", m[1])); err != nil {
			t.Errorf("fonts.css names %s: %v", m[1], err)
		}
	}
}
