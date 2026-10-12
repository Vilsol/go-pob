package datafields

import (
	"testing"

	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob-data/raw"
)

// TestAssetURLIsTheLoadersURL: project and sizes find the full assets in the cache under the URL the loader requested them by.
func TestAssetURLIsTheLoadersURL(t *testing.T) {
	for path, want := range map[string]string{
		"raw/Mods":                               loader.RawURL[*poe.Mod]("3.29", "Mods"),
		"raw/AlternateTreeVersions":              loader.RawURL[*raw.AlternateTreeVersion]("3.29", "AlternateTreeVersions"),
		"stat_translations/en/stat_descriptions": loader.TranslationURL("3.29", "en", "stat_descriptions"),
	} {
		got, err := AssetURL("3.29", path)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s: %s, the loader requests %s", path, got, want)
		}
	}
	if _, err := AssetURL("3.29", "raw/NoSuchTable"); err == nil {
		t.Error("an asset the loader does not read must have no URL")
	}
}
