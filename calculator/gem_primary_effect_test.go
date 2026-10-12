package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestPrimaryGrantedEffectIsPerGem pins the replacement for poe.SkillGem.GetGrantedEffect().
func TestPrimaryGrantedEffectIsPerGem(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	gemByBaseID := func(id string) *poe.SkillGem {
		base := poe.BaseItemTypeByIDMap[id]
		if base == nil {
			t.Fatalf("REFUSING: %q is not a BaseItemType in the loaded data — wrong fixture", id)
		}
		gem := base.SkillGem()
		if gem == nil {
			t.Fatalf("REFUSING: %q is not a skill gem — wrong fixture", id)
		}
		return gem
	}

	for _, c := range []struct{ baseID, wantPrimary, wantSecondary string }{
		{"Metadata/Items/Gems/SkillGemArc", "Arc", ""},
		{"Metadata/Items/Gems/SupportGemMultistrike", "SupportMultistrike", ""},
		{"Metadata/Items/Gems/SkillGemVaalArcticArmour", "VaalArcticArmour", "ArcticArmour"},
	} {
		t.Run(c.baseID, func(t *testing.T) {
			gem := gemByBaseID(c.baseID)

			got := primaryGrantedEffectOf(gem)
			if got == nil {
				t.Fatalf("primaryGrantedEffectOf = nil, want %q", c.wantPrimary)
			}
			if got.ID != c.wantPrimary {
				t.Errorf("primaryGrantedEffectOf = %q, want %q", got.ID, c.wantPrimary)
			}

			sec := secondaryGrantedEffectOf(gem, "", nil)
			switch {
			case c.wantSecondary == "" && sec != nil:
				t.Errorf("secondaryGrantedEffectOf = %q, want none", sec.ID)
			case c.wantSecondary != "" && sec == nil:
				t.Errorf("secondaryGrantedEffectOf = nil, want %q", c.wantSecondary)
			case sec != nil && sec.ID != c.wantSecondary:
				t.Errorf("secondaryGrantedEffectOf = %q, want %q", sec.ID, c.wantSecondary)
			}

			if up := gem.GetGrantedEffect(); up == nil || up.ID != "Fireball" {
				id := "<nil>"
				if up != nil {
					id = up.ID
				}
				t.Fatalf("REFUSING: poe.SkillGem.GetGrantedEffect() answered %q rather than the "+
					"constant \"Fireball\" this helper exists to replace — re-measure the data "+
					"and retire the workaround instead of trusting this test", id)
			}
		})
	}

	if primaryGrantedEffectOf(nil) != nil {
		t.Error("primaryGrantedEffectOf(nil) is non-nil")
	}
}
