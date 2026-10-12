package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestBearTrapEnchantLandsOnTheEnemy pins ModParser.lua:5703.
func TestBearTrapEnchantLandsOnTheEnemy(t *testing.T) {
	e := ParseMod("Enemies affected by Bear Trap take 30% increased Damage from Trap or Mine Hits", false)
	if e == nil || len(e.ModList) != 1 {
		t.Fatalf("ParseMod produced %v, want exactly 1 mod", e)
	}
	if e.Extra != "" {
		t.Fatalf("leftover Extra %q — the line does not parse whole", e.Extra)
	}

	outer, ok := e.ModList[0].Value().ValueList.(mod.ExtraSkillMod)
	if !ok {
		t.Fatalf("outer mod is %T, want mod.ExtraSkillMod", e.ModList[0].Value().ValueList)
	}

	if got := outer.Mod.Name(); got != "EnemyModifier" {
		t.Fatalf("ExtraSkillMod wraps %q, want %q; the debuff is landing on the PLAYER, not the enemy", got, "EnemyModifier")
	}

	inner, ok := outer.Mod.Value().ValueList.(mod.EnemyModifier)
	if !ok {
		t.Fatalf("EnemyModifier payload is %T", outer.Mod.Value().ValueList)
	}
	if got := inner.Mod.Name(); got != "TrapMineDamageTaken" {
		t.Errorf("enemy mod = %q, want TrapMineDamageTaken", got)
	}
	if got := inner.Mod.Value().Float(); got != 30 {
		t.Errorf("enemy mod value = %v, want 30", got)
	}
}
