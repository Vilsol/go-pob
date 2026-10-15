package calculator

import (
	"strings"
	"testing"
)

// TestSidebarCritAndHitBreakdowns pins CalcOffence.lua:2542-2555.
func TestSidebarCritAndHitBreakdowns(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0215-slayer-cyclone-of-tumult.xml")
	b := env.Player.Breakdown
	if b == nil || !env.Player.MainSkill.SkillFlags[SkillFlagAttack] {
		t.Fatal("CONTROL FAILED: no breakdown or not an attack")
	}
	for _, key := range []string{"PreEffectiveCritChance", "CritChance"} {
		e := b.Data[key]
		if e == nil || len(e.Lines) == 0 {
			t.Errorf("%s breakdown missing", key)
			continue
		}
		if e.BreakdownSource != "MainHand.CritChance" && e.BreakdownSource != "OffHand.CritChance" {
			t.Errorf("%s breakdownSource = %q", key, e.BreakdownSource)
		}
	}
	if pre, crit := b.Data["PreEffectiveCritChance"], b.Data["CritChance"]; pre != nil && crit != nil && len(pre.Lines) > len(crit.Lines) {
		t.Errorf("PreEffectiveCritChance has more lines than CritChance: %v", strings.Join(pre.Lines, " | "))
	}
	if b.Data["HitChance"] == nil {
		t.Error("HitChance breakdown missing")
	}

	dual := corpusEnvForLoadedData(t, "0021-assassin-blade-flurry.xml").Player.Breakdown
	for _, key := range []string{"CritChance", "HitChance"} {
		if e := dual.Data[key]; e == nil || len(e.Lines) == 0 || e.Lines[0] != "Both weapons:" {
			t.Errorf("dual wield %s breakdown does not combine both weapons: %+v", key, e)
		}
	}
}
