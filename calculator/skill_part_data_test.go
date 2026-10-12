package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestBladeVortexBaseModsCarryEveryPart: Blade Vortex writes hitTimeOverride once per part.
func TestBladeVortexBaseModsCarryEveryPart(t *testing.T) {
	list := moddb.NewModList()
	for _, m := range raw.SkillBaseMods["BladeVortex"] {
		list.AddMod(m)
	}

	for _, tc := range []struct {
		part   int
		hit    float64
		blades float64
	}{
		{1, 0.6, 0},
		{2, 0.6 / 2.75, 5},
		{3, 0.6 / 4.5, 10},
	} {
		cfg := &moddb.ListCfg{SkillPart: tc.part}

		var hit float64
		for _, v := range list.List(cfg, "SkillData") {
			if sd, ok := v.(mod.SkillData); ok && sd.Key == "HitTimeOverride" {
				hit = sd.Value
			}
		}
		if math.Abs(hit-tc.hit) > 1e-9 {
			t.Errorf("part %d: HitTimeOverride = %v, PoB act_dex.lua:945-947 gives %v", tc.part, hit, tc.hit)
		}

		blades := list.Sum(mod.TypeBase, cfg, "Multiplier:BladeVortexBlade")
		if blades != tc.blades {
			t.Errorf("part %d: Multiplier:BladeVortexBlade = %v, PoB act_dex.lua:943-944 gives %v", tc.part, blades, tc.blades)
		}
	}
}

// TestEarthquakeStatMapCarriesPart2: Earthquake's aftershock damage is a part-2 statMap row (src/Data/Skills/act_str.lua).
func TestEarthquakeStatMapCarriesPart2(t *testing.T) {
	entry := raw.SkillStatMaps["Earthquake"]["earthquake_aftershock_minimum_added_physical_damage"]
	if entry == nil {
		t.Fatal("Earthquake has no earthquake_aftershock_minimum_added_physical_damage entry")
	}
	if len(entry.Mods) == 0 {
		t.Fatal("Earthquake aftershock entry has no mods; PoB maps it to PhysicalMin tagged SkillPart 2")
	}
	for _, m := range entry.Mods {
		for _, tag := range m.Tags() {
			if sp, ok := tag.(*mod.SkillPartTag); ok && sp.Part == 2 {
				return
			}
		}
	}
	t.Fatal("Earthquake aftershock mod is not tagged SkillPart 2")
}

// TestElementalHitStatMapCarriesPartLists: PoB's `skillPartList` form: Elemental Hit suppresses fire on parts 3-6.
func TestElementalHitStatMapCarriesPartLists(t *testing.T) {
	entry := raw.SkillStatMaps["ElementalHit"]["elemental_hit_no_physical_chaos_damage"]
	if entry == nil {
		t.Fatal("ElementalHit has no elemental_hit_no_physical_chaos_damage entry")
	}

	list := moddb.NewModList()
	for _, m := range entry.Mods {
		list.AddMod(m)
	}
	for part, wantNoFire := range map[int]bool{1: false, 2: false, 3: true, 4: true, 5: true, 6: true} {
		got := list.Flag(&moddb.ListCfg{SkillPart: part}, "DealNoFire")
		if got != wantNoFire {
			t.Errorf("part %d: DealNoFire = %v, PoB gives %v", part, got, wantNoFire)
		}
	}
}
