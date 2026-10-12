package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestPatternTranscriptionAgainstPin pins the rows a mechanical sweep of THIS FILE's table keys against the pinned ModParser.lua found to differ from upstream in OPTIONALITY or CARDINALITY.
func TestPatternTranscriptionAgainstPin(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 ||
		strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line does not parse")
	}

	for _, tc := range []struct {
		line string
		want string
	}{
		{"Intimidate Enemies for 12 seconds on Block while holding a Shield", "EnemyModifier"},
		{"+10 to maximum Virulence", "Multiplier:VirulenceStacksMax"},
		{"Minions deal 5% increased Damage per Minion, up to a maximum of 50%", "MinionModifier"},
		{"+10% chance to Suppress Spell Damage if your equipped Boots, Helmet and Gloves have Evasion", "SpellSuppressionChance"},
		{"Summoned Skeleton Warriors and Soldiers deal Triple Damage with this Weapon if you've Hit with this Weapon Recently", "MinionModifier"},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || len(e.ModList) == 0 {
			t.Errorf("DROPPED WHOLE (no mods): %s", tc.line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q, so the line is discarded: %s", extra, tc.line)
			continue
		}
		var names []string
		for _, m := range e.ModList {
			names = append(names, m.Name())
		}
		if !contains(names, tc.want) {
			t.Errorf("%q -> %v, want %q", tc.line, names, tc.want)
		}
	}

	for _, tc := range []struct {
		line string
		want string
	}{
		{"35% chance to avoid being Stunned for each Herald Buff affecting you", "AvoidStun"},
		{"35% chance to avoid being Stunned for each Herald Skill affecting you", "AvoidStun"},
		{"Gain Life and Mana from Leech instantly during Flask Effect", "InstantLifeLeech"},
		{"Gain Life and Mana from Leech instantly during Effect", "InstantLifeLeech"},
		{"Summoned Skeleton Warriors deal Triple Damage with this Weapon if you've Hit with this Weapon Recently", "MinionModifier"},
		{"Minions deal 5% increased Damage per Minion", "MinionModifier"},
		{"+10% chance to Suppress Spell Damage if your Boots, Helmet and Gloves have Evasion", "SpellSuppressionChance"},
		{"Intimidate Enemies for 4 seconds on Block while holding a Shield", "EnemyModifier"},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
			t.Errorf("CONTROL MOVED: the other spelling no longer parses whole: %s", tc.line)
			continue
		}
		var names []string
		for _, m := range e.ModList {
			names = append(names, m.Name())
		}
		if !contains(names, tc.want) {
			t.Errorf("CONTROL MOVED: %q -> %v, want %q", tc.line, names, tc.want)
		}
	}

	e := ParseMod("35% chance to avoid being Stunned for each Herald Buff affecting you", false)
	if e != nil && len(e.ModList) > 0 {
		var got []string
		for _, tag := range e.ModList[0].Tags() {
			if m, ok := tag.(*mod.MultiplierTag); ok {
				got = append(got, m.VariableList...)
			}
		}
		if !contains(got, "Herald") {
			t.Errorf("herald buff line -> multipliers %v, want Herald", got)
		}
	}
}
