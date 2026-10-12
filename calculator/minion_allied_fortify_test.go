package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMinionInheritsParentFortificationFromFortitude is the CONSUMER half of the Fortitude chain, and it exists because neither half can be measured alone.
func TestMinionInheritsParentFortificationFromFortitude(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	build := loadCorpusBuild(t, "0682-gladiator-herald-of-agony.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
	if env.Minion == nil || env.Minion.Actor == nil {
		t.Fatal("REFUSING: build has no minion actor, so this fixture proves nothing")
	}

	if !env.Minion.ModDB.Flag(nil, "YourFortifyEqualToParent") {
		t.Fatal("the minion never received YourFortifyEqualToParent: the ExtraAura ally arm " +
			"did not deliver it, so the alliedFortify term below is untested")
	}

	parentStacks := env.Player.Output["FortificationStacks"]
	if parentStacks <= 0 {
		t.Fatalf("REFUSING: the parent has %v fortification stacks; a zero cannot be "+
			"distinguished from an uninherited one", parentStacks)
	}
	if got := env.Minion.Output["FortificationStacks"]; got != parentStacks {
		t.Errorf("Minion.FortificationStacks = %v, want the parent's %v", got, parentStacks)
	}

	if got := env.Minion.ModDB.More(nil, "DamageTakenWhenHit"); math.Abs(got-0.8) > 1e-9 {
		t.Errorf("minion DamageTakenWhenHit More = %v, want 0.8 (PoB's -20 MORE)", got)
	}

	if !env.Minion.ModDB.Flag(nil, "Condition:HaveMaximumFortification") {
		t.Error("minion has no Condition:HaveMaximumFortification, so stacks < maxStacks")
	}

	for key, want := range map[string]float64{
		"PhysicalTakenHitMult":  0.104,
		"LightningTakenHitMult": 0.824,
		"ColdTakenHitMult":      0.824,
		"FireTakenHitMult":      0.824,
		"ChaosTakenHitMult":     0.8,
		"SpellTakenHitMult":     0.8,
		"AttackTakenHitMult":    0.8,
	} {
		if got := env.Minion.Output[key]; math.Abs(got-want) > want*1e-9 {
			t.Errorf("Minion.%s = %v, want %v (PoB's value)", key, got, want)
		}
	}

	if env.Player.ModDB.Flag(nil, "YourFortifyEqualToParent") {
		t.Error("the PLAYER received YourFortifyEqualToParent: the ExtraAura self arm ran " +
			"despite onlyAllies")
	}
	if env.Player.Parent != nil {
		t.Error("the player has a parent actor, so the guard above proves less than it reads")
	}
}

// TestMinionFortificationExceedsItsOwnMaximum is the fixture for :636's `m_max(MaximumFortification, alliedFortify)` arm.
func TestMinionFortificationExceedsItsOwnMaximum(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	build := loadCorpusBuild(t, "5850-ladder-REDZIxPL_7670.GownianyDeszcz.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
	if env.Minion == nil || env.Minion.Actor == nil {
		t.Fatal("REFUSING: build has no minion actor, so this fixture proves nothing")
	}
	if !env.Minion.ModDB.Flag(nil, "YourFortifyEqualToParent") {
		t.Fatal("the minion never received YourFortifyEqualToParent: the arm below is untested")
	}

	if own := env.Minion.ModDB.Sum(mod.TypeBase, nil, "MaximumFortification"); own != 20 {
		t.Fatalf("REFUSING: the minion's own MaximumFortification is %v, not the base 20 — "+
			"this fixture no longer isolates the m_max arm", own)
	}
	if got := env.Minion.Output["MaximumFortification"]; got != 38 {
		t.Errorf("Minion.MaximumFortification = %v, want 38 (the parent's, through m_max)", got)
	}
	if got := env.Minion.Output["FortificationStacks"]; got != 38 {
		t.Errorf("Minion.FortificationStacks = %v, want 38", got)
	}
	for _, key := range []string{"AttackTakenHitMult", "SpellTakenHitMult"} {
		if got := env.Minion.Output[key]; math.Abs(got-0.62) > 0.62*1e-9 {
			t.Errorf("Minion.%s = %v, want 0.62 (PoB's value)", key, got)
		}
	}
}
