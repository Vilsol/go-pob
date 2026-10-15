package host

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestFindPoBBuilds(t *testing.T) {
	home := t.TempDir()
	mk := func(p string) string {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(home, p), 0o755); err != nil {
			t.Fatal(err)
		}
		return filepath.Join(home, p)
	}
	write := func(p, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(home, p), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	docs := mk("Documents")
	wineBuilds := mk("Documents/Path of Building/Builds")
	mk(".wine/drive_c/users/me")
	if err := os.Symlink(docs, filepath.Join(home, ".wine/drive_c/users/me/Documents")); err != nil {
		t.Fatal(err)
	}
	pfx := ".steam/steam/steamapps/compatdata/123/pfx"
	mk(pfx + "/drive_c/users/steamuser/Documents/Path of Building")
	custom := mk(pfx + "/drive_c/PoB Builds")
	write(pfx+"/drive_c/users/steamuser/Documents/Path of Building/Settings.xml",
		`<?xml version="1.0"?><PathOfBuilding><Mode mode="LIST"/><Misc buildPath="C:\PoB Builds/" showThousandsSeparators="true"/></PathOfBuilding>`)
	mk(".local/share/bottles/bottles/pob/drive_c/users/me/Documents/Path of Building")
	mk("Games/other/drive_c/users/me/Documents")

	old := time.Now().Add(-time.Hour)
	_ = os.Chtimes(wineBuilds, old, old)

	got := FindPoBBuilds(home, "", filepath.Join(home, "Documents"))
	if len(got) != 2 {
		t.Fatalf("found %+v", got)
	}
	if got[0].Dir != filepath.Join(custom, "") || got[0].Settings == "" {
		t.Errorf("first is the newest, Proton's custom path: %+v", got[0])
	}
	if got[1].Dir != wineBuilds && got[1].Dir != filepath.Join(home, ".wine/drive_c/users/me/Documents/Path of Building/Builds") {
		t.Errorf("second is Wine's: %+v", got[1])
	}
	empty := t.TempDir()
	if FindPoBBuilds(empty, "", filepath.Join(empty, "Documents")) != nil {
		t.Error("an empty home has none")
	}
}

func TestWineToUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Wine prefixes are a Linux host's")
	}
	pfx := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pfx, "dosdevices"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/", filepath.Join(pfx, "dosdevices", "z:")); err != nil {
		t.Fatal(err)
	}
	for in, want := range map[string]string{
		`C:\PoB\Builds/`:        filepath.Join(pfx, "drive_c", "PoB", "Builds"),
		`Z:\home\me\pob builds`: "/home/me/pob builds",
		`D:\x`:                  "",
		"/already/unix":         "/already/unix",
		"relative":              "",
	} {
		if got := wineToUnix(in, pfx); got != want {
			t.Errorf("%s: %q, want %q", in, got, want)
		}
	}
}

// TestFindPoBBuildsNative: The host's own PoB (native Windows.
func TestFindPoBBuildsNative(t *testing.T) {
	docs := t.TempDir()
	user := filepath.Join(docs, "Path of Building")
	custom := filepath.Join(t.TempDir(), "PoB Builds")
	for _, d := range []string{user, custom} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	settings := `<?xml version="1.0"?><PathOfBuilding><Misc buildPath="` + custom + `/"/></PathOfBuilding>`
	if err := os.WriteFile(filepath.Join(user, "Settings.xml"), []byte(settings), 0o600); err != nil {
		t.Fatal(err)
	}
	got := FindPoBBuilds(t.TempDir(), "", docs)
	if len(got) != 1 || got[0].Dir != custom {
		t.Fatalf("found %+v, want %s", got, custom)
	}
}

func TestWineToUnixNative(t *testing.T) {
	cases := map[string]string{"relative": "", `Builds\`: ""}
	switch runtime.GOOS {
	case "windows":
		cases[`D:\PoB Builds/`] = `D:\PoB Builds`
		cases[`C:/Users/me/OneDrive/Documents/Path of Building/Builds/`] = `C:\Users\me\OneDrive\Documents\Path of Building\Builds`
		cases[`\\nas\share\builds\`] = `\\nas\share\builds`
	default:
		cases[`D:\x`] = ""
		cases["/home/me/pob builds/"] = "/home/me/pob builds"
	}
	for in, want := range cases {
		if got := wineToUnix(in, ""); got != want {
			t.Errorf("%s: %q, want %q", in, got, want)
		}
	}
}

func TestUserDocuments(t *testing.T) {
	home := t.TempDir()
	got := UserDocuments(home)
	if !filepath.IsAbs(got) || filepath.Base(got) != "Documents" {
		t.Errorf("UserDocuments: %q", got)
	}
	if runtime.GOOS == "windows" && got == filepath.Join(home, "Documents") {
		t.Errorf("UserDocuments: %q is the fallback, not the known folder", got)
	}
	t.Log(got)
}
