package calculator

import "testing"

// TestFoulbornUniqueItemCount: a unique with a mutated explicit line is foulborn.
func TestFoulbornUniqueItemCount(t *testing.T) {
	requireGameData(t, "3.29")
	for build, want := range map[string]map[string]float64{
		"11388-ladder-Panther34_4123.WingedGuardianAngel_CZE.xml": {"SpellBlockChance": 75},
		"6151-ladder-SoloT1Witch_1235.AllFlameOfAT.xml":           {"GemLevel": 34},
	} {
		env := corpusEnvForLoadedData(t, build)
		if got := env.ItemModDB.Multipliers["FoulbornUniqueItem"]; got < 4 {
			t.Errorf("%s Multiplier:FoulbornUniqueItem = %v, want >= 4", build, got)
		}
		for k, w := range want {
			if got := env.Player.Output[k]; got != w {
				t.Errorf("%s %s = %v, want %v (PoB)", build, k, got, w)
			}
		}
	}
}

func TestFoulbornGemLevelRowParses(t *testing.T) {
	e := ParseMod("+2 to Level of all Lightning Skill Gems if at least 4 Foulborn Unique Items are Equipped", false)
	if e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "GemProperty" {
		t.Fatalf("got %d mods, extra %q", len(e.ModList), e.Extra)
	}
}
