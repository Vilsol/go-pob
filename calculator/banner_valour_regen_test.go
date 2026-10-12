package calculator

import (
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestBannerValourRegenRows pins ModParser.lua:2589-2594.
func TestBannerValourRegenRows(t *testing.T) {
	for _, line := range []string{
		"You and Allies near your Banner Regenerate 0.1% of Life per second for each Valour consumed for that Banner",
		"You and Allies affected by your placed Banners Regenerate 0.1% of Life per second for each Stage",
	} {
		e := ParseMod(strings.ToLower(line), false)
		if e == nil || e.Extra != "" || len(e.ModList) != 1 {
			t.Fatalf("%q: got %+v, want one mod parsed whole", line, e)
		}
		outer := e.ModList[0]
		aura, ok := outer.Value().List().(mod.ExtraAura)
		if outer.Name() != "ExtraAura" || !ok {
			t.Fatalf("%q: got %s %T, want ExtraAura LIST", line, outer.Name(), outer.Value().List())
		}
		inner := aura.Mod
		if inner.Name() != "LifeRegenPercent" || inner.Value().Float() != 0.1 {
			t.Errorf("%q: inner %s %v", line, inner.Name(), inner.Value())
		}
		var conds, mults []string
		for _, tag := range inner.Tags() {
			switch tg := tag.(type) {
			case *mod.ConditionTag:
				conds = append(conds, tg.VarList...)
			case *mod.MultiplierTag:
				mults = append(mults, tg.VariableList...)
			}
		}
		if !slices.Contains(conds, "AffectedByPlacedBanner") || !slices.Contains(mults, "BannerValour") {
			t.Errorf("%q: conditions %v multipliers %v, want AffectedByPlacedBanner + BannerValour", line, conds, mults)
		}
	}

	e := ParseMod(strings.ToLower("If you've Impaled an Enemy Recently, you and nearby Allies have +1000 to Armour"), false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "ExtraAura" {
		t.Errorf("impale control: got %+v", e)
	}
}
