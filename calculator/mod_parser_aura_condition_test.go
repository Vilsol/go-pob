package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"
	raw2 "github.com/Vilsol/go-pob/data/raw"
)

// TestAuraConditionPatternsCoverEveryAuraGem pins the `while affected by <aura>` / `while using <aura>` grammar that ModParser.lua:6059-6062 GENERATES.
func TestAuraConditionPatternsCoverEveryAuraGem(t *testing.T) {
	requireGameData(t, "3.29")

	var auraNames []string
	for _, gem := range raw2.Gems {
		if !gem.Tags["aura"] && !gem.Tags["herald"] {
			continue
		}
		ge := poe.GrantedEffectByID(gem.GrantedEffectID)
		if ge == nil || ge.IsSupport || ge.GetActiveSkill() == nil {
			continue
		}
		auraNames = append(auraNames, gem.Name)
	}
	if len(auraNames) < 20 {
		t.Fatalf("REFUSING: raw.Gems reports %d aura/herald gems; this test would pass "+
			"vacuously over an empty population", len(auraNames))
	}

	var missing []string
	for _, name := range auraNames {
		key := "while affected by " + strings.ToLower(name)
		if _, ok := skillNameListCompiled[key]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d of %d aura/herald gems have no `while affected by` pattern: %v",
			len(missing), len(auraNames), missing)
	}
}

// TestWatchersEyeLinesParseWhole is the end-to-end half: the grammar gap above costs the whole modifier, not just its condition.
func TestWatchersEyeLinesParseWhole(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line failed to parse; this test cannot attribute a failure")
	}

	for _, line := range []string{
		"Gain 25 Energy Shield per Enemy Hit while affected by Discipline",
		"Damage Penetrates 15% Lightning Resistance while affected by Wrath",
		"+23% to Critical Strike Multiplier while affected by Precision",
		"8% additional Physical Damage Reduction while affected by Determination",
		"+1.79% to Critical Strike Chance while affected by Hatred",
		"Gain 28 Life per Enemy Hit while affected by Vitality",
		"+20% to Damage over Time Multiplier while affected by Malevolence",
		"You have Phasing while affected by Haste",
		"+39% to Chaos Resistance while affected by Purity of Elements",
		"Gain 10% of Maximum Mana as Extra Maximum Energy Shield while affected by Clarity",
		"Impales you inflict last 2 additional Hits while using Pride",
	} {
		e := ParseMod(line, false)
		if e == nil {
			t.Errorf("no entry: %s", line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q (line is DROPPED WHOLE): %s", extra, line)
			continue
		}
		if len(e.ModList) == 0 {
			t.Errorf("parsed but produced 0 mods: %s", line)
		}
	}
}
