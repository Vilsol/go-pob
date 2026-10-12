package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestRepeatPerRepeatDamageReachesTheSkill pins the repeat-multiplier block, CalcOffence.lua:786-869, which had no counterpart here at all.
func TestRepeatPerRepeatDamageReachesTheSkill(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0221-reliquarian-ethereal-knives.xml")
	ms := env.Player.MainSkill
	if ms == nil || ms.SkillModList == nil {
		t.Fatal("fixture is useless: no main skill")
	}

	src := ms.SkillModList.Tabulate(mod.TypeMore, ms.SkillCfg, "RepeatPerRepeatDamage")
	if len(src) == 0 {
		t.Fatal("fixture is useless: no RepeatPerRepeatDamage mod on this skill")
	}
	if env.Player.Output["Repeats"] <= 1 {
		t.Fatalf("fixture is useless: Repeats = %v, so (Repeats-1) is 0 and the emitted mod is 0 "+
			"whether or not the block ran", env.Player.Output["Repeats"])
	}

	want := src[0].Mod.GetSource()
	for _, m := range ms.SkillModList.Tabulate(mod.TypeMore, ms.SkillCfg, "Damage") {
		if m.Mod.GetSource() == want {
			return
		}
	}
	t.Errorf("no Damage MORE carrying source %q — CalcOffence.lua:824-838 re-emits every "+
		"RepeatPerRepeatDamage mod as a Damage MORE scaled by (Repeats-1); Repeats = %v",
		want, env.Player.Output["Repeats"])
}
