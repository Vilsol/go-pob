//go:build !js

package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
)

// TestSettingsOutliveTheCacheExpiry: App-wide settings (PoB's Settings.xml, e.g.
func TestSettingsOutliveTheCacheExpiry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := Get()

	got, err := s.GetSetting("shared-item-sets.json")
	testza.AssertNoError(t, err)
	testza.AssertNil(t, got, "an unset setting reads as nothing")

	testza.AssertNoError(t, s.SetSetting("shared-item-sets.json", []byte(`[{"title":"a"}]`)))
	dir, err := os.UserConfigDir()
	testza.AssertNoError(t, err)
	old := time.Now().Add(-2 * cacheExpiry)
	testza.AssertNoError(t, os.Chtimes(filepath.Join(dir, "go-pob", "settings", "shared-item-sets.json"), old, old))

	got, err = s.GetSetting("shared-item-sets.json")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, `[{"title":"a"}]`, string(got))
}
