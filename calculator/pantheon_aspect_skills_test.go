package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func aspectModSum(env *Environment, db map[string][]mod.Mod, stat, source string, modType mod.Type) (float64, bool) {
	sum, found := 0.0, false
	for _, m := range db[stat] {
		if string(m.GetSource()) == source && m.Type() == modType {
			sum += ourModValue(m)
			found = true
		}
	}
	return sum, found
}

// TestPantheonAspectSkills pins upstream #10205: the aspect skills' buffs land.
func TestPantheonAspectSkills(t *testing.T) {
	t.Run("Brine King chills with its own buff's chill effect", func(t *testing.T) {
		env := warcryFixtureEnv(t, "13791-ladder-illdyking_0423.daymanILLDAN.xml")
		v, ok := aspectModSum(env, env.Player.ModDB.Mods, "ChillOverride", "Aspect of the Brine King", mod.TypeBase)
		if !ok {
			t.Fatal("no ChillOverride from Aspect of the Brine King")
		}
		if v < 15 {
			t.Errorf("ChillOverride = %v, want >= 15 (the buff's +50%% chill effect missing)", v)
		}
	})
	t.Run("Solaris Sunscald scales inflicted exposure", func(t *testing.T) {
		env := warcryFixtureEnv(t, "0475-elementalist-divine-blast-of-radiance.xml")
		if got := env.EnemyModDB.Sum(mod.TypeBase, nil, "FireExposure"); got != -20 {
			t.Fatalf("CONTROL FAILED: enemy FireExposure = %v, want two -10 sources (the build's and Solaris's)", got)
		}
		if got := env.Player.ModDB.Sum(mod.TypeBase, nil, "ExtraExposure", "ExtraFireExposure"); got != -30 {
			t.Fatalf("CONTROL FAILED: ExtraExposure = %v, want -30", got)
		}
		sources := map[mod.Source]bool{}
		for _, m := range env.EnemyModDB.Mods["FireExposure"] {
			sources[m.GetSource()] = true
		}
		got, found := 0.0, false
		for _, m := range env.EnemyModDB.Mods["FireResist"] {
			if sources[m.GetSource()] && m.Type() == mod.TypeBase {
				got, found = got+ourModValue(m), true
			}
		}
		if !found || got != -70 {
			t.Errorf("FireResist from exposure = %v (found %v), want -70", got, found)
		}
	})
	t.Run("Solaris inflicts Fire Exposure", func(t *testing.T) {
		env := warcryFixtureEnv(t, "1036-elementalist-vaal-cold-snap-6.xml")
		if got, ok := aspectModSum(env, env.EnemyModDB.Mods, "FireExposure", "Skill:AspectOfSolaris", mod.TypeBase); !ok || got != -10 {
			t.Errorf("enemy FireExposure from Aspect of Solaris = %v (%v), want -10", got, ok)
		}
	})
	t.Run("Lunaris grants damage taken per Lunar Protection", func(t *testing.T) {
		env := warcryFixtureEnv(t, "0820-hierophant-forbidden-rite-of-soul-sacrifice-2.xml")
		if got := env.Player.ModDB.Sum(mod.TypeBase, nil, "Multiplier:LunarProtection"); got != 8 {
			t.Errorf("Multiplier:LunarProtection = %v, want the placeholder 8", got)
		}
		if _, ok := aspectModSum(env, env.Player.ModDB.Mods, "PhysicalDamageTaken", "Skill:AspectOfLunaris", mod.TypeMore); !ok {
			t.Error("no PhysicalDamageTaken MORE from Aspect of Lunaris")
		}
	})
}
