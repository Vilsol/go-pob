package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestEnemiesYouveHitRecentlyPrefixes pins ModParser.lua:1261-1269, three preFlagList rows that had no counterpart here.
func TestEnemiesYouveHitRecentlyPrefixes(t *testing.T) {
	const suffix = "50% reduced Life Regeneration rate"

	requireEnemyLifeRegen := func(t *testing.T, line string) mod.Mod {
		t.Helper()
		e := ParseMod(line, false)
		if e == nil || e.Extra != "" {
			t.Fatalf("%q did not parse whole: %v", line, e)
		}
		if len(e.ModList) != 1 {
			t.Fatalf("%q: got %d mods, want 1", line, len(e.ModList))
		}
		m := e.ModList[0]
		if m.Name() != "EnemyModifier" {
			t.Fatalf("%q: got %s, want EnemyModifier (applyToEnemy rewraps)", line, m.Name())
		}
		em, ok := m.Value().List().(mod.EnemyModifier)
		if !ok {
			t.Fatalf("%q: payload is %T, want mod.EnemyModifier", line, m.Value().List())
		}
		if em.Mod.Name() != "LifeRegen" || em.Mod.Type() != mod.TypeIncrease || em.Mod.Value().Float() != -50 {
			t.Fatalf("%q: nested mod is %s %s %v, want LifeRegen INC -50",
				line, em.Mod.Name(), em.Mod.Type(), em.Mod.Value())
		}
		return m
	}

	t.Run("suffix alone was never broken", func(t *testing.T) {
		e := ParseMod(suffix, false)
		if e == nil || e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "LifeRegen" {
			t.Fatalf("got %v, want a single LifeRegen mod", e)
		}
	})

	t.Run("bare prefix", func(t *testing.T) {
		m := requireEnemyLifeRegen(t, "Enemies you've Hit Recently have "+suffix)
		var conds []string
		for _, tg := range m.Tags() {
			if c, ok := tg.(*mod.ConditionTag); ok {
				conds = append(conds, c.VarList...)
			}
		}
		if len(conds) != 1 || conds[0] != "HitRecently" {
			t.Errorf("Condition tags = %v, want exactly [HitRecently]", conds)
		}
	})

	for _, tc := range []struct{ line, actorCond string }{
		{"While a Pinnacle Atlas Boss is in your presence, Enemies you've Hit Recently have " + suffix, "PinnacleBoss"},
		{"While a unique Enemy is in your presence, Enemies you've Hit Recently have " + suffix, "RareOrUnique"},
	} {
		t.Run(tc.actorCond, func(t *testing.T) {
			m := requireEnemyLifeRegen(t, tc.line)
			var hitRecently, actorVar bool
			for _, tg := range m.Tags() {
				switch tag := tg.(type) {
				case *mod.ActorConditionTag:
					for _, v := range tag.VariableList {
						if v == tc.actorCond {
							actorVar = true
						}
					}
				case *mod.ConditionTag:
					for _, v := range tag.VarList {
						if v == "HitRecently" {
							hitRecently = true
						}
					}
				}
			}
			if !hitRecently || !actorVar {
				t.Errorf("tags = %v; want BOTH Condition:HitRecently and enemy %s. "+
					"One tag alone applies the mod where upstream withholds it.",
					m.Tags(), tc.actorCond)
			}
		})
	}
}
