package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestWorklistParserRowsParseWhole pins six rows the pin parses WHOLE and go-pob dropped whole.
func TestWorklistParserRowsParseWhole(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 ||
		strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line does not parse")
	}

	for _, tc := range []struct {
		line string
		want []string
	}{
		{"Gain 1% of Cold Damage as Extra Fire Damage per 1% Chill Effect on Enemy", []string{"ColdDamageGainAsFire"}},
		{"Gain 1% of Lightning Damage as Extra Cold Damage per 2% Shock Effect on Enemy", []string{"LightningDamageGainAsCold"}},
		{"Minions have +5% to Critical Strike Multiplier per Withered Debuff on Enemy", []string{"MinionModifier"}},
		{"40% increased Effect of Non-Curse Auras from your Skills while you have a Linked Target", []string{"AuraEffect"}},
		{"Socketed Support Gems can also Support Skills from Equipped Body Armour", []string{"LinkedSupport"}},
		{"Hits with Prismatic Skills always inflict Brittle", []string{"EnemyBrittleChance"}},
		{"Hits with Prismatic Skills always Scorch", []string{"EnemyScorchChance"}},
		{"Hits with Prismatic Skills always Sap", []string{"EnemySapChance"}},
		{"Unholy Might during Effect", []string{"Condition:UnholyMight", "Condition:CanWither"}},
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
		for _, want := range tc.want {
			if !contains(names, want) {
				t.Errorf("%q -> %v, want %q", tc.line, names, want)
			}
		}
	}

	for _, tc := range []struct {
		line   string
		multi  string
		actor  string
		thresh bool
	}{
		{"Gain 1% of Cold Damage as Extra Fire Damage per 1% Chill Effect on Enemy", "ChillEffect", "enemy", false},
		{"Minions have +5% to Critical Strike Multiplier per Withered Debuff on Enemy", "WitheredStack", "enemy", false},
		{"40% increased Effect of Non-Curse Auras from your Skills while you have a Linked Target", "LinkedTargets", "", true},
	} {
		e := ParseMod(tc.line, false)
		if e == nil || len(e.ModList) == 0 {
			continue
		}
		var got []string
		for _, m := range e.ModList {
			for _, tag := range allTagsDeep(m) {
				switch v := tag.(type) {
				case *mod.MultiplierTag:
					if !tc.thresh && v.TagActor == tc.actor {
						got = append(got, v.VariableList...)
					}
				case *mod.MultiplierThresholdTag:
					if tc.thresh {
						got = append(got, v.Variable)
					}
				}
			}
		}
		if !contains(got, tc.multi) {
			t.Errorf("%q -> multipliers %v, want %q on actor %q", tc.line, got, tc.multi, tc.actor)
		}
	}
}

func allTagsDeep(m mod.Mod) []mod.Tag {
	tags := m.Tags()
	if m.Value().Type() == mod.ModValueMultiTypeList {
		if mm, ok := m.Value().List().(mod.MinionModifier); ok && mm.Mod != nil {
			tags = append(tags, mm.Mod.Tags()...)
		}
	}
	return tags
}
