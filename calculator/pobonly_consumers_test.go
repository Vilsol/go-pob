package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

// TestLifeRegenAppliesToEnergyShieldConsumer: CalcDefence.lua:1286-1292 adds floor(LifeRegen * conv, 2) and floor(LifeRegenPercent * conv, 2) as ES regen.
func TestLifeRegenAppliesToEnergyShieldConsumer(t *testing.T) {
	t.Parallel()
	build := parseCorpusBuild(t, "0011-saboteur-cyclone.xml")
	build.SetConfigOption(pob.Input{Name: "customMods", String: utils.Ptr("18% of Life Regeneration also applies to Energy Shield")})
	db := GetMiscCalculator(build).Base().ModDB
	lifePercent := db.Sum(mod.TypeBase, nil, "LifeRegenPercent")
	lifeBase := db.Sum(mod.TypeBase, nil, "LifeRegen")
	if lifePercent <= 0 && lifeBase <= 0 {
		t.Fatalf("REFUSING: the build has no life regeneration, so the conversion cannot show")
	}
	want := map[string]float64{
		"EnergyShieldRegen":        math.Floor(lifeBase*0.18*100+0.0001) / 100,
		"EnergyShieldRegenPercent": math.Floor(lifePercent*0.18*100+0.0001) / 100,
	}
	for name, w := range want {
		got, n := 0.0, 0
		for _, r := range db.Tabulate(mod.TypeBase, nil, name) {
			if r.Mod.GetSource() == "Life Regen to ES Regen" {
				got += r.Value
				n++
			}
		}
		if n != 1 || got != w {
			t.Errorf("%s from Life Regen to ES Regen: %d mods summing %v, want 1 mod of %v", name, n, got, w)
		}
	}
}

// TestMagicFlaskNoAdjacentEffectConsumer: CalcSetup.lua:1022-1037 maps each flask to its slot.
func TestMagicFlaskNoAdjacentEffectConsumer(t *testing.T) {
	t.Parallel()
	const line = "Equipped Magic Flasks have 30% increased effect on you if no Flasks are Adjacent to them"
	speed := func(emptyNeighbours, withLine bool) float64 {
		build := parseCorpusBuild(t, "0006-elementalist-storm-burst.xml")
		set := &build.Items.ItemSets[0]
		for i := range set.Slots {
			switch set.Slots[i].Name {
			case "Flask 2":
				if set.Slots[i].ItemID != 16 {
					t.Fatalf("REFUSING: Flask 2 holds item %d, not the magic Quicksilver Flask 16", set.Slots[i].ItemID)
				}
				set.Slots[i].Active = true
			case "Flask 1", "Flask 3":
				if emptyNeighbours {
					set.Slots[i].ItemID = 0
				}
			}
		}
		if withLine {
			build.SetConfigOption(pob.Input{Name: "customMods", String: utils.Ptr(line)})
		}
		return GetMiscCalculator(build).Base().Player.Output["MovementSpeedMod"]
	}
	if base, active := speed(true, false), speed(false, false); base != active || base <= 1 {
		t.Fatalf("REFUSING: MovementSpeedMod %v (empty neighbours) / %v (full rack) without the line — the active Quicksilver Flask does not show", base, active)
	}
	if without, with := speed(true, false), speed(true, true); with <= without {
		t.Errorf("isolated magic flask: MovementSpeedMod %v with the line, %v without — the bonus did not apply", with, without)
	}
	if without, with := speed(false, false), speed(false, true); with != without {
		t.Errorf("flask with neighbours: MovementSpeedMod %v with the line, %v without — the bonus applied despite adjacency", with, without)
	}
}
