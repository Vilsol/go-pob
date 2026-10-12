package calculator

import (
	"math"
	"os"
	"testing"
)

// TestMirageMoreRoundsInItsOwnList pins CalcActiveSkill.lua:181 on the mirage copy: `newSkill.skillModList = new("ModList", newSkill.baseSkillModList)`.
func TestMirageMoreRoundsInItsOwnList(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	const name = "1235-necromancer-kinetic-blast-of-clustering-5.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	want := rec.Output["MirageDPS"]
	if want == 0 {
		t.Fatalf("REFUSING: %s has no MirageDPS in the oracle", name)
	}

	env := corpusEnvForLoadedData(t, name)
	got := env.Player.Output["MirageDPS"]
	if math.Abs(got-want) > 1e-6*math.Abs(want) {
		t.Errorf("%s MirageDPS = %.6f, PoB %.6f (ratio %.6f)", name, got, want, got/want)
	}
}

// TestMirageSkillChainsToTheOuterActor pins CalcActiveSkill.lua:178/:482 on the mirage copy.
func TestMirageSkillChainsToTheOuterActor(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	for _, name := range []string{
		"4399-ladder-CrisCringe_2334.StarterBanBumBum.xml",
		"6280-ladder-Teralog_4039.DivineSorcière.xml",
		"0270-warden-elemental-hit-of-the-spectrum.xml",
	} {
		rec, ok := oracle[name]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", name)
		}
		want := rec.Output["MirageDPS"]
		if want == 0 {
			t.Fatalf("REFUSING: %s has no MirageDPS in the oracle", name)
		}
		env := corpusEnvForLoadedData(t, name)
		got := env.Player.Output["MirageDPS"]
		if math.Abs(got-want) > 1e-6*math.Abs(want) {
			t.Errorf("%s MirageDPS = %.6f, PoB %.6f (ratio %.6f)", name, got, want, got/want)
		}
	}
}

// TestMirageCopyHasNoSlotName pins that PoB's mirage copy has no slotName.
func TestMirageCopyHasNoSlotName(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	for _, name := range []string{"1424-juggernaut-kinetic-bolt.xml", "1615-juggernaut-kinetic-blast.xml"} {
		want := oracle[name].Output["MirageDPS"]
		if want == 0 {
			t.Fatalf("REFUSING: %s has no MirageDPS in the oracle", name)
		}
		got := corpusEnvForLoadedData(t, name).Player.Output["MirageDPS"]
		if math.Abs(got-want) > 1e-6*math.Abs(want) {
			t.Errorf("%s MirageDPS = %.6f, PoB %.6f (ratio %.6f)", name, got, want, got/want)
		}
	}
}
