package raw

import (
	"testing"

	"github.com/Vilsol/go-pob-data/loader"

	"github.com/Vilsol/go-pob/storage"
)

// TestTimelessJewelSourcesLoad pins the three data sources the timeless-jewel transform needs and that poe.InitializeAll does NOT provide.
func TestTimelessJewelSourcesLoad(t *testing.T) {
	if err := InitializeAll(LatestVersion, nil); err != nil {
		t.Fatalf("REFUSING: game data %s did not initialise: %v", LatestVersion, err)
	}
	if err := EnsureTimelessSources(); err != nil {
		t.Fatalf("REFUSING: timeless sources did not load: %v", err)
	}

	t.Run("AlternateTreeVersions", func(t *testing.T) {
		if len(AlternateTreeVersions) == 0 {
			t.Fatal("empty — the timeless-jewel transform selects a tree version per jewel type")
		}
		for want, key := range map[string]int{
			"vaal": 1, "karui": 2, "maraketh": 3, "templar": 4, "eternal": 5, "kalguuran": 6,
		} {
			v := AlternateTreeVersionByConqueror(want)
			if v == nil {
				t.Errorf("no AlternateTreeVersion for conqueror type %q", want)
				continue
			}
			if v.Key != key {
				t.Errorf("conqueror %q has jewel type %d, want %d — the enum join is broken", want, v.Key, key)
			}
		}
		if len(AlternateTreeVersions) < 8 {
			t.Errorf("only %d tree versions; 3.29 ships the Abyss jewels at key 7+", len(AlternateTreeVersions))
		}
	})

	t.Run("PassiveSkills", func(t *testing.T) {
		if len(PassiveSkills) == 0 {
			t.Fatal("empty — Calculate() is keyed on the passive skill INDEX, not the tree node id")
		}
		ps := PassiveSkillByGraphID(34171)
		if ps == nil {
			t.Fatal("no PassiveSkill for graph id 34171")
		}
		if ps.Hash != 34171 {
			t.Errorf("graph id round-trip broken: got %d", ps.Hash)
		}
		if ps.Key == 34171 {
			t.Errorf("Key %d equals the graph id — the two tables are almost certainly conflated", ps.Key)
		}
	})

	t.Run("StatDescriptions", func(t *testing.T) {
		desc, err := loader.LoadTranslation(DataContext(t.Context()), LatestVersion, "en", "stat_descriptions", AssetLoaderWrapper{Storage: storage.Get()})
		if err != nil {
			t.Fatal(err)
		}
		if len(desc.Descriptors) == 0 {
			t.Fatal("empty — rendering a replaced node's stat line needs the translation file")
		}
	})
}
