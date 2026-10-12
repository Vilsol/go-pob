package calculator

import (
	"sort"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func modValuesBySource(actor *Actor, stat string) []string {
	out := make([]string, 0, len(actor.ModDB.Mods[stat]))
	for _, m := range actor.ModDB.Mods[stat] {
		out = append(out, string(m.GetSource())+"="+formatModValue(ourModValue(m)))
	}
	sort.Strings(out)
	return out
}

func formatModValue(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// TestWarcryV218ResidueIsGone pins the deletion of the v2.18.1 warcry block at CalcPerform.lua:1429 and the move of its exert half to the 3.29 site at :2316-2325.
func TestWarcryV218ResidueIsGone(t *testing.T) {
	gone := []string{
		"AlreadyGlobalWarcryCooldown", "GlobalWarcryCooldown", "GlobalWarcryCount",
		"AncestralActive", "EnduringActive", "InfernalActive", "BattlemageActive",
		"IntimidatingActive",
	}

	t.Run("0018-berserker-heavy-strike", func(t *testing.T) {
		env := warcryFixtureEnv(t, "0018-berserker-heavy-strike.xml")
		for _, name := range gone {
			if got := modValuesBySource(env.Player, name); len(got) != 0 {
				t.Errorf("%s: %d mod(s) %v — PoB's dump carries none", name, len(got), got)
			}
		}
		if got := modValuesBySource(env.Player, "SpellDamageAppliesToAttacks"); len(got) != 0 {
			t.Errorf("SpellDamageAppliesToAttacks: %v — PoB's dump carries none", got)
		}
		if got := modValuesBySource(env.Player, "ImprovedSpellDamageAppliesToAttacks"); len(got) != 0 {
			t.Errorf("ImprovedSpellDamageAppliesToAttacks: %v — PoB's dump carries none", got)
		}
		assertModSum(t, env, "NumAncestralExerts", 18)
		assertModSum(t, env, "NumBattlemageExerts", 12)
		if got := modValuesBySource(env.Player, "Multiplier:ExertingWarcryCount"); len(got) != 2 {
			t.Errorf("Multiplier:ExertingWarcryCount = %v, want 2 entries (Ancestral Cry, Battlemage's Cry)", got)
		}
		for _, name := range []string{"NumInfernalExerts", "NumIntimidatingExerts", "NumRallyingExerts", "NumSeismicExerts"} {
			if got := modValuesBySource(env.Player, name); len(got) != 0 {
				t.Errorf("%s: %v — PoB's dump carries none on this build", name, got)
			}
		}
	})

	t.Run("0065-chieftain-tectonic-slam", func(t *testing.T) {
		env := warcryFixtureEnv(t, "0065-chieftain-tectonic-slam.xml")
		for _, name := range gone {
			if got := modValuesBySource(env.Player, name); len(got) != 0 {
				t.Errorf("%s: %d mod(s) %v — PoB's dump carries none", name, len(got), got)
			}
		}
		if !env.ModDB.Flag(nil, "RallyingActive") {
			t.Errorf("RallyingActive is not set; CalcPerform.lua:1432's arm did not run")
		}
		if !env.ModDB.Flag(nil, "SeismicActive") {
			t.Errorf("SeismicActive is not set; CalcPerform.lua:1435's arm did not run")
		}
		assertModSum(t, env, "RallyingExertMoreDamagePerAlly", 5)
		assertModSum(t, env, "SeismicMoreAoE", 50)
		assertModSum(t, env, "NumBattlemageExerts", 8)
		assertModSum(t, env, "NumInfernalExerts", 9)
		assertModSum(t, env, "NumIntimidatingExerts", 5)
		assertModSum(t, env, "NumRallyingExerts", 8)
		assertModSum(t, env, "NumSeismicExerts", 9)
		if got := modValuesBySource(env.Player, "Multiplier:ExertingWarcryCount"); len(got) != 5 {
			t.Errorf("Multiplier:ExertingWarcryCount = %v, want 5 entries", got)
		}
	})
}

func assertModSum(t *testing.T, env *Environment, stat string, want float64) {
	t.Helper()
	if got := env.Player.ModDB.Sum(mod.TypeBase, nil, stat); got != want {
		t.Errorf("%s = %v, want %v (PoB's dump); mods: %v", stat, got, want, modValuesBySource(env.Player, stat))
	}
}
