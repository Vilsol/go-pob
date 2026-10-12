package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func repeatDamageMoreFor(t *testing.T, env *Environment, source string) (float64, bool) {
	t.Helper()
	ms := env.Player.MainSkill
	if ms == nil || ms.SkillModList == nil {
		t.Fatal("REFUSING: fixture has no main skill")
	}
	for _, v := range ms.SkillModList.Tabulate(mod.TypeMore, ms.SkillCfg, "Damage") {
		if string(v.Mod.GetSource()) != source {
			continue
		}
		for _, tag := range v.Mod.Tags() {
			if tag.Type() == mod.TypeModFlagOr {
				return v.Mod.Value().Float(), true
			}
		}
	}
	return 0, false
}

// TestRepeatModeDefaultsToAverage pins the resolution of the `repeatMode` config.
func TestRepeatModeDefaultsToAverage(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const multistrike = "Skill:SupportGreaterMultistrike"

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}

	t.Run("an unset repeatMode averages the repeat damage", func(t *testing.T) {
		const name = "3773-ladder-theMope_2750.Juggernerd"
		env := corpusEnvForLoadedData(t, name)

		if got := env.Player.Output["Repeats"]; got != 4 {
			t.Fatalf("fixture is useless: Repeats = %v, want 4 — the averaging divides by it", got)
		}
		got, ok := repeatDamageMoreFor(t, env, multistrike)
		if !ok {
			t.Fatalf("fixture is useless: no re-emitted Damage MORE from %s", multistrike)
		}
		if got != 33 {
			t.Errorf("re-emitted Damage MORE = %v, want 33 — PoB averages (0+22+44+66)/4 over the "+
				"initial hit and its three repeats (CalcOffence.lua:842-860); %v is the FINAL "+
				"repeat's value, which is the arm taken when repeatMode is not AVERAGE", got, got)
		}

		rec, hit := oracle[name+".xml"]
		if !hit {
			t.Fatalf("REFUSING: %s is not in the oracle blob", name)
		}
		mine := env.Player.OutputTable[OutTableMainHand]
		for _, k := range []string{"TotalMin", "TotalMax", "AverageDamage"} {
			want := rec.MainHand[k]
			if want == 0 {
				t.Fatalf("REFUSING: PoB reports 0 for MainHand.%s; a 0-vs-0 agreement proves nothing", k)
			}
			if g := mine[k]; math.Abs(g-want) > 0.005*math.Abs(want)+1e-9 {
				t.Errorf("MainHand.%s = %v, want %v", k, g, want)
			}
		}
	})

	t.Run("an explicit FINAL_DPS still takes the final repeat", func(t *testing.T) {
		const name = "0787-juggernaut-molten-strike-of-the-zenith-2"
		env := corpusEnvForLoadedData(t, name)

		if got := env.Build.GetStringOption("repeatMode"); got != "FINAL_DPS" {
			t.Fatalf("control is useless: the build no longer saves repeatMode=FINAL_DPS (got %q), "+
				"so it no longer distinguishes a saved input from the default", got)
		}
		if got := env.Player.Output["Repeats"]; got != 4 {
			t.Fatalf("control is useless: Repeats = %v, want 4", got)
		}
		got, ok := repeatDamageMoreFor(t, env, multistrike)
		if !ok {
			t.Fatalf("control is useless: no re-emitted Damage MORE from %s", multistrike)
		}
		if got != 66 {
			t.Errorf("re-emitted Damage MORE = %v, want 66 — the build asks for FINAL_DPS, so "+
				"CalcOffence.lua:850-853 emits the final repeat's value unaveraged", got)
		}
	})
}
