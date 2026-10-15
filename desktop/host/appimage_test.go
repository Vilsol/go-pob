package host

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const testEntry = `[Desktop Entry]
Type=Application
Name=go-pob
Exec=go-pob %u
Icon=go-pob
MimeType=x-scheme-handler/pob;application/xml;
`

func TestAppImageEntry(t *testing.T) {
	for _, tc := range []struct{ path, exec string }{
		{"/opt/go-pob-x86_64.AppImage", "Exec=/opt/go-pob-x86_64.AppImage %u"},
		{"/home/a b/go-pob.AppImage", `Exec="/home/a b/go-pob.AppImage" %u`},
		{`/x/$a"b\c/100%.AppImage`, `Exec="/x/\\$a\\"b\\\\c/100%%.AppImage" %u`},
	} {
		got := string(AppImageEntry([]byte(testEntry), tc.path))
		if !strings.Contains(got, "\n"+tc.exec+"\n") || !strings.Contains(got, "\nTryExec="+tc.path+"\n") {
			t.Errorf("%s:\n%s", tc.path, got)
		}
		if strings.Contains(got, "Exec=go-pob") || !strings.Contains(got, "MimeType=x-scheme-handler/pob;application/xml;\n") {
			t.Errorf("%s: other keys changed:\n%s", tc.path, got)
		}
	}
}

func TestIntegrateAppImage(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	data := t.TempDir()
	notes, err := IntegrateAppImage(t.Context(), "/opt/go-pob.AppImage", data, []byte(testEntry), []byte("png"))
	if err != nil {
		t.Fatal(err)
	}
	if notes != "update-desktop-database not found\nxdg-mime not found" {
		t.Errorf("notes: %q", notes)
	}
	entry, err := os.ReadFile(filepath.Join(data, "applications", "go-pob.desktop"))
	if err != nil || !strings.Contains(string(entry), "Exec=/opt/go-pob.AppImage %u") {
		t.Errorf("entry: %v\n%s", err, entry)
	}
	if icon, err := os.ReadFile(filepath.Join(data, "icons", "hicolor", "512x512", "apps", "go-pob.png")); err != nil || string(icon) != "png" {
		t.Errorf("icon: %v %q", err, icon)
	}
}

// TestIntegrateAppImageXdgMime needs xdg-utils and desktop-file-utils.
func TestIntegrateAppImageXdgMime(t *testing.T) {
	for _, tool := range []string{"xdg-mime", "update-desktop-database"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " not installed")
		}
	}
	home := t.TempDir()
	for k, v := range map[string]string{
		"HOME": home, "XDG_CONFIG_HOME": filepath.Join(home, ".config"), "XDG_DATA_HOME": filepath.Join(home, ".local", "share"),
		"XDG_DATA_DIRS": filepath.Join(home, "system"), "XDG_CONFIG_DIRS": filepath.Join(home, "system"), "XDG_CURRENT_DESKTOP": "",
		"KDE_FULL_SESSION": "", "GNOME_DESKTOP_SESSION_ID": "", "DESKTOP_SESSION": "",
	} {
		t.Setenv(k, v)
	}
	query := func() string {
		out, _ := exec.CommandContext(t.Context(), "xdg-mime", "query", "default", "x-scheme-handler/pob").Output()
		return strings.TrimSpace(string(out))
	}
	if got := query(); got != "" {
		t.Fatalf("before: %q", got)
	}
	notes, err := IntegrateAppImage(t.Context(), "/opt/go-pob.AppImage", os.Getenv("XDG_DATA_HOME"), []byte(testEntry), []byte("png"))
	if err != nil || notes != "" {
		t.Fatalf("%v %q", err, notes)
	}
	if got := query(); got != "go-pob.desktop" {
		t.Fatalf("after: %q", got)
	}
}
