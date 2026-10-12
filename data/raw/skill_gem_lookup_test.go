package raw

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/storage"
)

// TestSkillGemForGrantedEffectIDResolves pins the granted effect -> skill gem link.
func TestSkillGemForGrantedEffectIDResolves(t *testing.T) {
	if err := poe.InitializeAll(DataContext(t.Context()), LatestVersion, AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct{ effectID, wantBaseName string }{
		{"Fireball", "Fireball"},
		{"SupportBarrage", "Barrage Support"},
		{"Barrage", "Barrage"},
	} {
		t.Run(c.effectID, func(t *testing.T) {
			gem := SkillGemForGrantedEffectID(c.effectID)
			if gem == nil {
				t.Fatalf("granted effect %q resolves to no skill gem", c.effectID)
			}
			base := gem.GetBaseItemType()
			if base == nil {
				t.Fatalf("%q resolved to a gem with no base item type", c.effectID)
			}
			if base.Name != c.wantBaseName {
				t.Errorf("%q resolved to base %q, want %q", c.effectID, base.Name, c.wantBaseName)
			}
		})
	}

	if gem := SkillGemForGrantedEffectID("MonsterLightningStrike"); gem != nil {
		base := "?"
		if b := gem.GetBaseItemType(); b != nil {
			base = b.Name
		}
		t.Errorf("MonsterLightningStrike resolved to gem %q; PoB ships no gem for it, so the "+
			"lookup is matching something it should not", base)
	}
	if gem := SkillGemForGrantedEffectID("ZzNotARealGrantedEffect"); gem != nil {
		t.Error("a fabricated granted effect id resolved to a gem; the lookup cannot answer NO")
	}

	resolved := 0
	for _, ge := range poe.GrantedEffects {
		if SkillGemForGrantedEffectID(ge.ID) != nil {
			resolved++
		}
	}
	t.Logf("granted effects resolving to a gem: %d of %d", resolved, len(poe.GrantedEffects))
	if resolved < 500 {
		t.Errorf("only %d granted effects resolve to a gem; PoB ships 608 gem rows, so this is "+
			"the broken-index shape (it resolved exactly 1) rather than PoB's own coverage", resolved)
	}
}
