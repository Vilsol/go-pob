package calculator

import (
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestAllyExtraAuraRowsParseWithTheirGating pins ModParser.lua:2585 and :5086.
func TestAllyExtraAuraRowsParseWithTheirGating(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line failed to parse; this test cannot attribute a failure")
	}

	extraAuraOf := func(t *testing.T, line string) mod.ExtraAura {
		t.Helper()
		e := ParseMod(line, false)
		if e == nil {
			t.Fatalf("no entry: %s", line)
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Fatalf("leftover %q (the line is DROPPED WHOLE): %s", extra, line)
		}
		var found []mod.ExtraAura
		for _, m := range e.ModList {
			if m.Name() != "ExtraAura" {
				continue
			}
			if ea, ok := m.Value().List().(mod.ExtraAura); ok {
				found = append(found, ea)
			}
		}
		if len(found) != 1 {
			t.Fatalf("expected exactly 1 ExtraAura mod, got %d: %s", len(found), line)
		}
		return found[0]
	}

	t.Run("Fortitude", func(t *testing.T) {
		ea := extraAuraOf(t, "Nearby Allies count as having Fortification equal to yours")

		if !ea.OnlyAllies {
			t.Errorf("OnlyAllies is false: the flag would land on the player as well as its allies")
		}
		if ea.Mod == nil || ea.Mod.Name() != "YourFortifyEqualToParent" {
			t.Fatalf("payload mod is %v, want YourFortifyEqualToParent", ea.Mod)
		}
		if ea.Mod.Type() != mod.TypeFlag {
			t.Errorf("payload type is %v, want FLAG", ea.Mod.Type())
		}
		var unscalable bool
		for _, raw := range ea.Mod.Tags() {
			if tag, ok := raw.(*mod.GlobalEffectTag); ok && tag.UnscalableTag {
				unscalable = true
			}
		}
		if !unscalable {
			t.Errorf("payload carries no unscalable GlobalEffect tag")
		}
	})

	t.Run("BitterHeresy", func(t *testing.T) {
		ea := extraAuraOf(t, "Profane Ground you create also affects you and your Allies, granting Chaotic Might")

		if ea.OnlyAllies {
			t.Errorf("OnlyAllies is true: the player would not receive its own Profane Ground buff")
		}
		if ea.Mod == nil || ea.Mod.Name() != "Condition:ChaoticMight" {
			t.Fatalf("payload mod is %v, want Condition:ChaoticMight", ea.Mod)
		}
		var onProfaneGround bool
		for _, raw := range ea.Mod.Tags() {
			if tag, ok := raw.(*mod.ConditionTag); ok && slices.Contains(tag.VarList, "OnProfaneGround") {
				onProfaneGround = true
			}
		}
		if !onProfaneGround {
			t.Errorf("payload carries no Condition(OnProfaneGround) tag: the buff would be unconditional")
		}
	})

	for _, line := range []string{
		"You have your maximum Fortification",
		"Profane Ground you create inflicts Malediction on Enemies",
	} {
		e := ParseMod(line, false)
		if e == nil || len(e.ModList) == 0 {
			t.Errorf("neighbour row stopped parsing: %s", line)
			continue
		}
		for _, m := range e.ModList {
			if m.Name() == "ExtraAura" {
				t.Errorf("neighbour row was swallowed by an ExtraAura row: %s", line)
			}
		}
	}
}
