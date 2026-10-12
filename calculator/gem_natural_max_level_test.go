package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestNaturalMaxLevelIsPerGem pins calcLib.validateGemLevel's SECOND fallback (CalcTools.lua:51-53), `gemInstance.gemData.naturalMaxLevel`.
func TestNaturalMaxLevelIsPerGem(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		baseID string
		want   int
	}{
		{"Metadata/Items/Gems/SkillGemArc", 20},
		{"Metadata/Items/Gems/SupportGemMultistrike", 20},
		{"Metadata/Items/Gems/SupportGemMultistrikePlus", 5},
	} {
		t.Run(c.baseID, func(t *testing.T) {
			base := poe.BaseItemTypeByIDMap[c.baseID]
			if base == nil {
				t.Fatalf("REFUSING: %q is not a BaseItemType — wrong fixture", c.baseID)
			}
			gem := base.SkillGem()
			if gem == nil {
				t.Fatalf("REFUSING: %q is not a skill gem — wrong fixture", c.baseID)
			}

			if got := naturalMaxLevelOf(gem); got != c.want {
				t.Errorf("naturalMaxLevelOf = %d, PoB's gem.naturalMaxLevel is %d", got, c.want)
			}

			if got := gem.DefaultLevel(); got != 19 {
				t.Fatalf("REFUSING: poe.SkillGem.DefaultLevel() answered %d rather than the "+
					"constant 19 this replaces — re-measure the data and retire the workaround "+
					"instead of trusting this test", got)
			}
		})
	}

	if got := naturalMaxLevelOf(nil); got != 0 {
		t.Errorf("naturalMaxLevelOf(nil) = %d, want 0", got)
	}
}
