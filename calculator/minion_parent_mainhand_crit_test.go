package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMinionAttackCritIsEqualToParentMainHand pins ModParser.lua:4475 and its consumer at CalcOffence.lua:3002-3003.
func TestMinionAttackCritIsEqualToParentMainHand(t *testing.T) {
	e := ParseMod("Minions' Base Attack Critical Strike Chance is equal to the Critical Strike Chance of your Main Hand Weapon", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 {
		t.Fatalf("got %v", e)
	}
	m := e.ModList[0]
	if m.Name() != "MinionModifier" || m.Type() != mod.TypeList {
		t.Fatalf("got %s %s, want MinionModifier LIST", m.Name(), m.Type())
	}
	mm, ok := m.Value().List().(mod.MinionModifier)
	if !ok {
		t.Fatalf("value = %T, want mod.MinionModifier", m.Value().List())
	}
	inner := mm.Mod
	if inner.Name() != "AttackCritIsEqualToParentMainHand" || inner.Type() != mod.TypeFlag || !inner.Value().Flag() {
		t.Fatalf("inner = %s %s %v, want AttackCritIsEqualToParentMainHand FLAG true",
			inner.Name(), inner.Type(), inner.Value())
	}
	if inner.Flags()&mod.MFlagAttack == 0 {
		t.Errorf("inner flags = %d, want MFlagAttack set", inner.Flags())
	}
}

// TestDestructiveLinkMainHandCritIsEqualToParent pins CalcOffence.lua:2991-2992 on the one corpus build whose dump carries MainHandCritIsEqualToParent.
func TestDestructiveLinkMainHandCritIsEqualToParent(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1549-luminary-animate-guardian-of-smiting-5.xml")
	if env.Minion == nil {
		t.Fatal("no minion")
	}
	const want = 14.45
	if got := env.Minion.Output["CritChance"]; math.Abs(got-want) > want*1e-6 {
		t.Errorf("minion CritChance = %v, PoB has %v", got, want)
	}
}
