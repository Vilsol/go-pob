package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

// TestGazeOfTheLichRows pins ModParser.lua:2429 and :3072-3101 (upstream #10232).
func TestGazeOfTheLichRows(t *testing.T) {
	for _, tc := range []struct {
		line  string
		names []string
	}{
		{"Arcane Surge also grants 10% increased Life Regeneration rate to you", []string{"ArcaneSurgeAlsoLifeRegen/INC/10"}},
		{"While you have at least 2 Hypnotic Eye Jewels socketed: Arcane Surge also grants 5% of Damage taken Recouped as Mana to you Arcane Surge also grants 10% increased Mana Cost Efficiency to you",
			[]string{"ArcaneSurgeAlsoManaRecoup/BASE/5", "ArcaneSurgeAlsoManaCostEfficiency/INC/10"}},
		{"While you have at least 2 Ghastly Eye Jewels socketed: Unholy Might you grant also causes target's Damage to Penetrate 5% Chaos Resistance Unholy Might you grant also applies 20% increased effect of Withered",
			[]string{"UnholyMightAlsoChaosPenetration/BASE/5", "UnholyMightAlsoWitherEffect/INC/20"}},
		{"While you have at least 2 Searching Eye Jewels socketed: Targets affected by Maim you inflict cannot deal Critical Strikes Maim you inflict causes Hits against the target to have 30% more Critical Strike Chance",
			[]string{"EnemyModifier/LIST/0", "EnemyModifier/LIST/0"}},
		{"While you have at least 2 Murderous Eye Jewels socketed: Intimidate you inflict applies an extra +10% increased Attack Damage taken Intimidate you inflict causes targets to deal 10% less Damage",
			[]string{"EnemyModifier/LIST/0", "EnemyModifier/LIST/0"}},
	} {
		entry := ParseMod(tc.line, false)
		if strings.TrimSpace(entry.Extra) != "" {
			t.Errorf("%q: unparsed %q", tc.line, entry.Extra)
			continue
		}
		var got []string
		for _, m := range entry.ModList {
			v := 0.0
			if m.Type() != mod.TypeList {
				v = m.Value().Float()
			}
			got = append(got, fmt.Sprintf("%s/%s/%g", m.Name(), m.Type(), v))
		}
		if strings.Join(got, ",") != strings.Join(tc.names, ",") {
			t.Errorf("%q: got %v, want %v", tc.line, got, tc.names)
		}
	}
	entry := ParseMod("While you have at least 2 Murderous Eye Jewels socketed: Intimidate you inflict applies an extra +10% increased Attack Damage taken Intimidate you inflict causes targets to deal 10% less Damage", false)
	if len(entry.ModList) == 2 {
		inner := entry.ModList[0].Value().List().(mod.EnemyModifier).Mod
		if inner.Name() != "DamageTaken" || inner.Flags() != mod.MFlagAttack || inner.KeywordFlags() != 0 {
			t.Errorf("intimidate DamageTaken: %s flags=%#x kw=%#x, want PoB's flags Attack and keywordFlags 0", inner.Name(), inner.Flags(), inner.KeywordFlags())
		}
	}
}

// TestArcaneSurgeAndUnholyMightAlsoGrant pins CalcPerform.lua:726-733, :750-758 and :2756-2763 (upstream #10232).
func TestArcaneSurgeAndUnholyMightAlsoGrant(t *testing.T) {
	t.Run("Arcane Surge also grants mana recoup and cost efficiency", func(t *testing.T) {
		env := warcryFixtureEnv(t, "15344-hierophant-spark-of-the-nova.xml")
		if env.Player.ModDB.Sum(mod.TypeBase, nil, "ArcaneSurgeAlsoManaRecoup") != 20 {
			t.Fatal("CONTROL FAILED: ArcaneSurgeAlsoManaRecoup is not 20")
		}
		recoup, ok1 := aspectModSum(env, env.Player.ModDB.Mods, "ManaRecoup", "Arcane Surge", mod.TypeBase)
		efficiency, ok2 := aspectModSum(env, env.Player.ModDB.Mods, "ManaCostEfficiency", "Arcane Surge", mod.TypeIncrease)
		if !ok1 || !ok2 || recoup <= 20 || recoup != efficiency {
			t.Errorf("ManaRecoup %v (%v), ManaCostEfficiency %v (%v): want both 20 * the Arcane Surge effect", recoup, ok1, efficiency, ok2)
		}
	})
	t.Run("Unholy Might also applies wither effect", func(t *testing.T) {
		requireGameData(t, "3.29")
		if os.Getenv("CORPUS") != "true" {
			t.Skip("set CORPUS=true")
		}
		corpusDir, _ := corpusNamesForLoadedData(t)
		file, err := os.ReadFile(filepath.Join(corpusDir, "0903-necromancer-animate-weapon-of-ranged-arms-19.xml"))
		if err != nil {
			t.Fatal(err)
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			t.Fatal(err)
		}
		on := true
		build.Config.Inputs = append(build.Config.Inputs, pob.Input{Name: "buffUnholyMight", Boolean: &on})
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		also := env.Player.ModDB.Sum(mod.TypeIncrease, nil, "UnholyMightAlsoWitherEffect")
		if also != 20 || !env.Player.ModDB.Flag(nil, "UnholyMight") {
			t.Fatalf("CONTROL FAILED: UnholyMight %v, UnholyMightAlsoWitherEffect %v", env.Player.ModDB.Flag(nil, "UnholyMight"), also)
		}
		base := math.Floor(6 * (1 + env.Player.ModDB.Sum(mod.TypeIncrease, nil, "WitherEffect")/100))
		if got := env.Player.ModDB.Max(nil, "WitherEffectStack"); got <= base {
			t.Errorf("WitherEffectStack = %v, want above %v (+%v%% from Unholy Might)", got, base, also)
		}
	})
	t.Run("Searching Eye maim adds enemy SelfCritChance MORE to crit chance", func(t *testing.T) {
		requireGameData(t, "3.29")
		if os.Getenv("CORPUS") != "true" {
			t.Skip("set CORPUS=true")
		}
		corpusDir, _ := corpusNamesForLoadedData(t)
		file, err := os.ReadFile(filepath.Join(corpusDir, "3784-ladder-tueurmaxee_5628.Vdh_Allflame.xml"))
		if err != nil {
			t.Fatal(err)
		}
		crit := func(maimed bool) (float64, float64) {
			build, err := builds.ParseBuild(file)
			if err != nil {
				t.Fatal(err)
			}
			build.Config.Inputs = append(build.Config.Inputs, pob.Input{Name: "conditionEnemyMaimed", Boolean: &maimed})
			env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
			return env.Player.Output["CritChance"], env.EnemyModDB.More(nil, "SelfCritChance")
		}
		base, _ := crit(false)
		maimed, more := crit(true)
		if more <= 1 {
			t.Fatalf("CONTROL FAILED: enemy SelfCritChance MORE = %v with the enemy maimed", more)
		}
		if base <= 0 || base >= 100 || math.Abs(maimed-math.Min(base*more, 100)) > 0.005*more+0.005 {
			t.Errorf("CritChance maimed = %v, want %v * %v (unmaimed %v)", maimed, base, more, base)
		}
	})
}
