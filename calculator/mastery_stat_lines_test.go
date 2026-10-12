package calculator

import (
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestMasteryStatLinesParseWhole: Mastery stat lines are DROPPED WHOLE when the parser leaves leftover Extra.
func TestMasteryStatLinesParseWhole(t *testing.T) {
	for _, c := range []struct{ line, wantName string }{
		{"100% increased Energy Shield from Equipped Helmet", "EnergyShield"},
		{"20% increased Armour from Equipped Body Armour", "Armour"},
		{"20% increased Armour from your Equipped Body Armour", "Armour"},
		{"10% increased Evasion Rating from Equipped Armour Items", "Evasion"},
		{"10% increased Armour from Equipped Boots and Gloves", "Armour"},
		{"Your Chance to Suppress Spell Damage is Lucky", "SpellSuppressionChanceIsLucky"},
		{"Corrupted Blood cannot be inflicted on you", "CorruptedBloodImmune"},
		{"100% increased Armour and Energy Shield from Equipped Helmet and Gloves", "ArmourAndEnergyShield"},
		{"10% increased Armour from Equipped Gloves and Boots", "Armour"},
	} {
		t.Run(c.line, func(t *testing.T) {
			e := ParseMod(c.line, false)
			if strings.TrimSpace(e.Extra) != "" {
				t.Fatalf("%q left Extra=%q -- processModLine DROPS the whole line", c.line, e.Extra)
			}
			if len(e.ModList) == 0 {
				t.Fatalf("%q yielded no mods at all", c.line)
			}
			found := false
			for _, m := range e.ModList {
				if m.Name() == c.wantName {
					found = true
				}
			}
			if !found {
				t.Errorf("%q did not yield %q", c.line, c.wantName)
			}
		})
	}
}

var restoredPhrases = []string{
	"from equipped helmet",
	"from equipped body armour",
	"from your equipped body armour",
	"from equipped armour items",
	"from equipped boots and gloves",
	"chance to suppress spell damage is lucky",
	"corrupted blood cannot be inflicted on you",
}

func TestSelectedMasteryLinesAreNotDropped(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	builds := []string{
		"0001-necromancer-unknown.xml",
		"0002-pathfinder-unknown.xml",
		"0009-trickster-cobra-lash.xml",
	}
	total, dropped := 0, 0
	var droppedLines []string
	for _, b := range builds {
		env := corpusEnvForLoadedData(t, b)
		for id, n := range env.AllocatedNodes {
			if n.IsMastery == nil || !*n.IsMastery {
				continue
			}
			nid, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				continue
			}
			eff, sel := env.Spec.MasterySelections[nid]
			if !sel {
				continue
			}
			for _, e := range n.MasteryEffects {
				if e.Effect != eff {
					continue
				}
				for _, line := range e.Stats {
					total++
					if pe := ParseMod(line, false); strings.TrimSpace(pe.Extra) != "" {
						dropped++
						droppedLines = append(droppedLines, line)
					}
				}
			}
		}
	}
	if total == 0 {
		t.Fatal("REFUSING: no selected mastery effect on any fixture; this test no longer " +
			"exercises the mechanism it was written for")
	}
	for _, phrase := range restoredPhrases {
		for _, line := range droppedLines {
			if strings.Contains(strings.ToLower(line), phrase) {
				t.Errorf("a line carrying the restored phrase %q is being dropped again: %q", phrase, line)
			}
		}
	}
	t.Logf("selected-mastery stat lines: %d, still dropped: %d (ItemCondition and others remain "+
		"unported -- see the report)", total, dropped)
}

// TestEnergyShieldFromHelmetReachesTheOracle grades the largest restored line against PoB.
func TestEnergyShieldFromHelmetReachesTheOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	const build = "0001-necromancer-unknown.xml"
	rec, ok := oracle[build]
	if !ok {
		t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
	}
	want, ok := rec.Output["EffectiveSpellSuppressionChance"]
	if !ok || want == 0 {
		t.Skipf("PoB reports no nonzero EffectiveSpellSuppressionChance for %s", build)
	}
	env := corpusEnvForLoadedData(t, build)
	if got := env.Player.Output["EffectiveSpellSuppressionChance"]; math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
		t.Errorf("%s: EffectiveSpellSuppressionChance = %v, PoB reports %v", build, got, want)
	}
}
