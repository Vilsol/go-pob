package host

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestFlatpakID(t *testing.T) {
	info := filepath.Join(t.TempDir(), ".flatpak-info")
	if got := flatpakID(info, "io.example.App"); got != "" {
		t.Fatalf("no .flatpak-info: %q", got)
	}
	if err := os.WriteFile(info, []byte("[Application]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := flatpakID(info, "io.example.App"); got != "io.example.App" {
		t.Fatalf("in a sandbox: %q", got)
	}
}

func TestRequestPath(t *testing.T) {
	if got := RequestPath(":1.42", "gopob7"); got != "/org/freedesktop/portal/desktop/request/1_42/gopob7" {
		t.Fatal(got)
	}
}

func TestPortalResult(t *testing.T) {
	uris := func(u ...string) map[string]dbus.Variant { return map[string]dbus.Variant{"uris": dbus.MakeVariant(u)} }
	for _, c := range []struct {
		name string
		body []any
		want string
		err  bool
	}{
		{"picked", []any{uint32(0), uris("file:///run/user/1000/doc/ab12/My%20Build.xml")}, "/run/user/1000/doc/ab12/My Build.xml", false},
		{"cancelled", []any{uint32(1), map[string]dbus.Variant{}}, "", false},
		{"failed", []any{uint32(2), map[string]dbus.Variant{}}, "", true},
		{"no uris", []any{uint32(0), map[string]dbus.Variant{}}, "", false},
		{"not a file", []any{uint32(0), uris("https://example.com/x.xml")}, "", true},
		{"short", []any{uint32(0)}, "", true},
	} {
		got, err := portalResult(c.body)
		if got != c.want || (err != nil) != c.err {
			t.Errorf("%s: %q, %v", c.name, got, err)
		}
	}
}
