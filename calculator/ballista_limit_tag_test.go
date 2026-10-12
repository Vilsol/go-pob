package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/storage"
)

// TestBallistaLimitIsTaggedTotemsAreBallistae pins ModParser.lua:600: "maximum number of summoned ballista totems" is gated on SkillType TotemsAreBallistae.
func TestBallistaLimitIsTaggedTotemsAreBallistae(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := poe.ActiveSkillTypesByID["TotemsAreBallistae"]; !ok {
		t.Fatalf("TotemsAreBallistae does not resolve in the loaded game data")
	}

	e := ParseMod("Attack Skills have +1 to maximum number of Summoned Ballista Totems", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "ActiveBallistaLimit" {
		t.Fatalf("want one ActiveBallistaLimit mod parsed whole, got %+v", e)
	}
	found := false
	for _, tag := range e.ModList[0].Tags() {
		if st, ok := tag.(*mod.SkillTypeTag); ok {
			for _, id := range st.SkillTypeList {
				if id == "TotemsAreBallistae" {
					found = true
				}
				if id == "RangedAttack" {
					t.Errorf("ballista limit is still gated on RangedAttack")
				}
			}
		}
	}
	if !found {
		t.Errorf("ballista limit carries no SkillType TotemsAreBallistae tag: %#v", e.ModList[0].Tags())
	}

	for _, c := range []struct {
		build string
		want  float64
	}{
		{"5971-ladder-SHAKKKA_1869.PIMPY_SON_OPP.xml", 2},
		{"10055-ladder-Funkomatic_5313.JingakhJr.xml", 6},
		{"13539-ladder-fishhhy_1795.Ratshits.xml", 18},
	} {
		build := loadCorpusBuild(t, c.build)
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if got := env.Player.Output["ActiveTotemLimit"]; math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: ActiveTotemLimit = %v, PoB %v", c.build, got, c.want)
		}
	}
}
