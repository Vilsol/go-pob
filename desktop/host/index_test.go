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
	if bytes.Contains(out, []byte(`rel="manifest"`)) {
		t.Error("still has the manifest link")
	}
	if !bytes.Equal(out, manifestLink.ReplaceAll(app, nil)) || !bytes.Contains(out, []byte(`/fonts/fonts.css" rel="stylesheet"`)) {
		t.Error("removed more than the manifest link")
	}
	if _, err := DesktopIndex([]byte("<html></html>")); err == nil {
		t.Error("an index without the manifest link must fail")
	}
}

// TestAppFonts: the web app and the desktop build draw from frontend/static/fonts alone; every file fonts.css or a preload names is there.
func TestAppFonts(t *testing.T) {
	const dir = "../../frontend/static/fonts"
	app, err := os.ReadFile("../../frontend/src/app.html")
	if err != nil {
		t.Fatal(err)
	}
	css, err := os.ReadFile(filepath.Join(dir, "fonts.css"))
	if err != nil {
		t.Fatal(err)
	}
	for _, host := range []string{"fonts.googleapis.com", "fonts.gstatic.com"} {
		if bytes.Contains(app, []byte(host)) {
			t.Errorf("app.html still reaches %s", host)
		}
	}
	urls := regexp.MustCompile(`url\(([^)]+)\)`).FindAllStringSubmatch(string(css), -1)
	if len(urls) == 0 {
		t.Fatal("fonts.css names no font file")
	}
	named := map[string]bool{}
	for _, m := range urls {
		named[m[1]] = true
		if _, err := os.Stat(filepath.Join(dir, m[1])); err != nil {
			t.Errorf("fonts.css names %s: %v", m[1], err)
		}
	}
	preloads := regexp.MustCompile(`<link rel="preload" href="%sveltekit\.assets%/fonts/([^"]+)" as="font" type="font/woff2" crossorigin />`).FindAllSubmatch(app, -1)
	if len(preloads) == 0 {
		t.Error("app.html preloads no font")
	}
	for _, m := range preloads {
		if !named[string(m[1])] {
			t.Errorf("app.html preloads %s, which fonts.css does not name", m[1])
		}
	}
	if !strings.Contains(string(css), "font-display: swap") {
		t.Error("fonts.css lost font-display: swap")
	}
}
