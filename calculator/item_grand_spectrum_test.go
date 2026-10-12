package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestGrandSpectrumMultiplierIsItemLevel pins Item.lua:2413-2418: the Multiplier:GrandSpectrum producer belongs to the JEWEL, not to any one of its mod lines.
//
//	if self.name:find("Grand Spectrum") then
//	  local spectrumMod = modLib.createMod("Multiplier:GrandSpectrum", "BASE", 1, self.name)
//	  modList:AddMod(spectrumMod)
//	  modList:NewMod("MinionModifier", "LIST", { mod = spectrumMod }, self.name)
//	end
func TestGrandSpectrumMultiplierIsItemLevel(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		name string
		raw  string
		line string
	}{
		{"charges", grandSpectrumRaw("Crimson Jewel", "+1 to Minimum Endurance Charges per Grand Spectrum"), "EnduranceChargesMin"},
		{"life", grandSpectrumRaw("Crimson Jewel", "5% increased Maximum Life per Grand Spectrum"), "Life"},
		{"minion crit multi", grandSpectrumRaw("Viridian Jewel", "Minions have +10% to Critical Strike Multiplier per Grand Spectrum"), "MinionModifier"},
		{"armour", grandSpectrumRaw("Crimson Jewel", "Gain 200 Armour per Grand Spectrum"), "Armour"},
		{"elemental damage", grandSpectrumRaw("Cobalt Jewel", "15% increased Elemental Damage per Grand Spectrum"), "ElementalDamage"},
		{"resistances", grandSpectrumRaw("Cobalt Jewel", "+7% to all Elemental Resistances per Grand Spectrum"), "ElementalResist"},
		{"mana", grandSpectrumRaw("Cobalt Jewel", "Gain 30 Mana per Grand Spectrum"), "Mana"},
		{"crit chance", grandSpectrumRaw("Viridian Jewel", "25% increased Critical Strike Chance per Grand Spectrum"), "CritChance"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := ParseItem(tc.raw)
			if item == nil || item.Base == nil {
				t.Fatalf("REFUSING: the fixture item did not resolve a base")
			}
			item.BuildModList()
			mods := item.Mods

			var spectrum, minionSpectrum, payload int
			for _, m := range mods {
				switch m.Name() {
				case "Multiplier:GrandSpectrum":
					spectrum++
				case tc.line:
					payload++
				}
				if m.Name() == "MinionModifier" && m.Value().Type() == mod.ModValueMultiTypeList {
					if mm, ok := m.Value().List().(mod.MinionModifier); ok && mm.Mod != nil &&
						mm.Mod.Name() == "Multiplier:GrandSpectrum" {
						minionSpectrum++
					}
				}
			}
			if spectrum != 1 {
				t.Errorf("Multiplier:GrandSpectrum appears %d times, want exactly 1", spectrum)
			}
			if minionSpectrum != 1 {
				t.Errorf("MinionModifier carrying Multiplier:GrandSpectrum appears %d times, want exactly 1", minionSpectrum)
			}
			if payload == 0 {
				t.Errorf("the jewel's own line produced no %s", tc.line)
			}
		})
	}

	item := ParseItem(grandSpectrumRaw("Cobalt Jewel", "10% increased Elemental Damage"))
	item.Name = "Hypnotic Eye Jewel"
	item.Title = "Hypnotic Eye Jewel"
	item.BuildModList()
	for _, m := range item.Mods {
		if m.Name() == "Multiplier:GrandSpectrum" {
			t.Errorf("a non-Grand-Spectrum jewel gained Multiplier:GrandSpectrum")
		}
	}
}

func grandSpectrumRaw(base, line string) string {
	return "Rarity: UNIQUE\nGrand Spectrum\n" + base + "\nLimited to: 3\nItem Level: 84\n" + line + "\n"
}
