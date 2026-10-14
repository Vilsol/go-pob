package calculator

import (
	"os"
	"strconv"
	"testing"
)

// TestFoulbornKitavaTriggersHelmetSpells grades CalcTriggers.lua:1095-1108 on 15421, whose helmet spells are triggered by Foulborn Kitava's Thirst.
func TestFoulbornKitavaTriggersHelmetSpells(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const name = "15421-inquisitor-crackling-lance-of-disintegration.xml"
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	env := corpusEnvForLoadedData(t, name)

	if want := rec.Output["FullDPS"]; want == 0 || !oracleClose(env.Player.Output["FullDPS"], want, oracleDumpTolerance) {
		t.Errorf("FullDPS = %g, PoB %g", env.Player.Output["FullDPS"], want)
	}

	saved := env.Build.Build.FullDPSSkills
	if len(saved) != 4 {
		t.Fatalf("REFUSING: %s saves %d FullDPSSkill rows, want 4", name, len(saved))
	}
	ours := map[string]SkillDPS{}
	for _, s := range env.Player.SkillDPS {
		ours[s.Name] = s
	}
	for _, row := range saved {
		want, err := strconv.ParseFloat(row.Value, 64)
		if err != nil {
			t.Fatalf("REFUSING: %s value %q: %v", row.Stat, row.Value, err)
		}
		got, ok := ours[row.Stat]
		if !ok {
			t.Errorf("%s: no SkillDPS row", row.Stat)
			continue
		}
		if !oracleClose(got.DPS*got.Count, want, oracleDumpTolerance) {
			t.Errorf("%s DPS = %g, PoB %g (ratio %.4f)", row.Stat, got.DPS*got.Count, want, got.DPS*got.Count/want)
		}
		source := got.Source
		if source == "" {
			source = got.Trigger
		}
		if source != row.Source {
			t.Errorf("%s source = %q, PoB %q", row.Stat, source, row.Source)
		}
	}
}
