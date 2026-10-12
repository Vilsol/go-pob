package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSelfCastCurseRowsParse pins the Balance of Terror family: every upstream row whose tag is `Condition:SelfCast<Curse>`.
func TestSelfCastCurseRowsParse(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 ||
		strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line does not parse")
	}

	for _, tc := range []struct {
		line string
		want string
		cond string
	}{
		{"Inflict Withered for 2 seconds on Hit if you've cast Despair in the past 10 seconds", "Condition:CanWither", "SelfCastDespair"},
		{"Gain 40% of Physical Damage as a Random Element if you've cast Elemental Weakness in the past 10 seconds", "PhysicalDamageGainAsRandom", "SelfCastElementalWeakness"},
		{"25% increased Cooldown Recovery Rate if you've cast Temporal Chains in the past 10 seconds", "CooldownRecovery", "SelfCastTemporalChains"},
		{"Immune to Curses if you've cast Despair in the past 10 seconds", "CurseImmune", "SelfCastDespair"},
		{"Inflict Fire Exposure on Hit if you've cast Flammability in the past 10 seconds", "EnemyModifier", "SelfCastFlammability"},
		{"Take no Extra Damage from Critical Strikes if you've cast Enfeeble in the past 10 seconds", "ReduceCritExtraDamage", "SelfCastEnfeeble"},
		{"You are Unaffected by Bleeding if you've cast Vulnerability in the past 10 seconds", "SelfBleedingEffect", "SelfCastVulnerability"},
		{"You are Unaffected by Ignite if you've cast Flammability in the past 10 seconds", "SelfIgniteEffect", "SelfCastFlammability"},
		{"Immune to Exposure if you've cast Elemental Weakness in the past 10 seconds", "ExposureImmune", "SelfCastElementalWeakness"},
		{"Intimidate Enemies on Hit if you've cast Punishment in the past 10 seconds", "EnemyModifier", "SelfCastPunishment"},
		{"You are Unaffected by Freeze if you've cast Frostbite in the past 10 seconds", "SelfFreezeEffect", "SelfCastFrostbite"},
		{"Cold Exposure on Hit if you've cast Frostbite in the past 10 seconds", "EnemyModifier", "SelfCastFrostbite"},
		{"You are Unaffected by Shock if you've cast Conductivity in the past 10 seconds", "SelfShockEffect", "SelfCastConductivity"},
		{"Inflict Lightning Exposure on Hit if you've cast Conductivity in the past 10 seconds", "EnemyModifier", "SelfCastConductivity"},
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
		var names, conds []string
		for _, m := range e.ModList {
			names = append(names, m.Name())
			for _, tag := range m.Tags() {
				if c, ok := tag.(*mod.ConditionTag); ok {
					conds = append(conds, c.VarList...)
				}
			}
		}
		if !contains(names, tc.want) {
			t.Errorf("produced %v, want %q: %s", names, tc.want, tc.line)
		}
		if !contains(conds, tc.cond) {
			t.Errorf("conditions %v, want %q: %s", conds, tc.cond, tc.line)
		}
	}
}
