package calculator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestChanceCombineDoubleHitsArmIsLive pins CalcOffence.lua:2020-2039 — combineStat's CHANCE mode has TWO arms and this tree had only the second.
//
//	:2020-2023  if skillData.doubleHitsWhenDualWielding then
func TestChanceCombineDoubleHitsArmIsLive(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	carriers := doubleHitsCarrierBuilds(t)
	if len(carriers) == 0 {
		t.Fatal("REFUSING: no dump carries doubleHitsWhenDualWielding at all — either the dump " +
			"set moved or this probe is reading the wrong field. A 0 here is not a finding.")
	}

	live := make([]string, 0, 8)
	for _, build := range carriers {
		env := corpusEnvForLoadedData(t, build)
		skill := env.Player.MainSkill
		if skill == nil || skill.SkillData == nil {
			continue
		}
		if skill.SkillFlags[SkillFlagBothWeaponAttack] && skill.SkillData.DoubleHitsWhenDualWielding {
			live = append(live, build)
		}
	}

	t.Logf("doubleHitsWhenDualWielding in a dump: %d; reaching the arm in the built environment: %d",
		len(carriers), len(live))

	if len(live) == 0 {
		t.Error("no corpus build reaches combineStat's CHANCE doubleHits arm in the built " +
			"environment, yet porting it moved the gate +4 on ImpaleModifier, PoisonDPS and " +
			"PoisonDamage. Those two facts cannot both be true — this probe is reading the " +
			"wrong flags, or the arm is being reached by another route. Do not delete the arm " +
			"on the strength of this count.")
	}
}

func doubleHitsCarrierBuilds(t *testing.T) []string {
	t.Helper()
	dir := dumpsDirForLoadedData(t)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("REFUSING: cannot read the dump directory %s: %v", dir, err)
	}

	out := make([]string, 0, 128)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("REFUSING: cannot read dump %s: %v", e.Name(), err)
		}
		if bytes.Contains(b, []byte("doubleHitsWhenDualWielding")) {
			out = append(out, strings.TrimSuffix(e.Name(), ".json")+".xml")
		}
	}
	return out
}
