package calculator

import (
	"math"
	"os"
	"testing"
)

// for _, blockType in ipairs({"BlockChance","ProjectileBlockChance","SpellBlockChance","SpellProjectileBlockChance"}) do
//
//	output["Effective"..blockType] = env.mode_effective
//	    and m_max(output[blockType] - enemyDB:Sum("BASE", nil, "reduceEnemyBlock"), 0)
//
// end
var effectiveBlockBuilds = []string{
	"0001-necromancer-unknown.xml",
	"0003-guardian-unknown.xml",
	"0004-trickster-lightning-tendrils-of-eccentricity.xml",
	"0006-elementalist-storm-burst.xml",
}

func TestEffectiveBlockChanceIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	keys := []string{
		"EffectiveBlockChance",
		"EffectiveProjectileBlockChance",
		"EffectiveSpellBlockChance",
		"EffectiveSpellProjectileBlockChance",
		"EffectiveAverageBlockChance",
	}

	for _, build := range effectiveBlockBuilds {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			env := corpusEnvForLoadedData(t, build)

			for _, k := range keys {
				want, ok := rec.Output[k]
				if !ok {
					t.Fatalf("REFUSING: PoB reports no %s for %s", k, build)
				}
				if want == 0 {
					t.Fatalf("REFUSING: PoB reports %s = 0 for %s; an unwritten key agrees with "+
						"that, so the case cannot discriminate", k, build)
				}
			}
			if rec.Output["EffectiveBlockChance"] == rec.Output["EffectiveSpellBlockChance"] {
				t.Fatalf("REFUSING: %s has equal attack and spell block (%v); a port that computed "+
					"one value for all four types would pass this fixture",
					build, rec.Output["EffectiveBlockChance"])
			}

			for _, k := range keys {
				want := rec.Output[k]
				got, present := env.Player.Output[k]
				if !present {
					t.Errorf("%s: we never write %s at all; PoB reports %v", build, k, want)
					continue
				}
				if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
					t.Errorf("%s: %s = %v, PoB reports %v", build, k, got, want)
				}
			}

			mean := (env.Player.Output["EffectiveBlockChance"] +
				env.Player.Output["EffectiveProjectileBlockChance"] +
				env.Player.Output["EffectiveSpellBlockChance"] +
				env.Player.Output["EffectiveSpellProjectileBlockChance"]) / 4
			if got := env.Player.Output["EffectiveAverageBlockChance"]; math.Abs(got-mean) > 1e-9 {
				t.Errorf("%s: EffectiveAverageBlockChance = %v but the mean of the four is %v",
					build, got, mean)
			}
		})
	}
}

var effectiveSuppressionBuilds = []string{
	"0005-occultist-vaal-double-strike.xml",
	"0006-elementalist-storm-burst.xml",
	"0008-necromancer-vaal-summon-skeletons.xml",
	"0001-necromancer-unknown.xml",
}

func TestEffectiveSpellSuppressionChanceIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	sawSmall := false
	for _, build := range effectiveSuppressionBuilds {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			want, ok := rec.Output["EffectiveSpellSuppressionChance"]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: PoB reports no nonzero EffectiveSpellSuppressionChance for %s; "+
					"an unwritten key agrees with 0 and the case cannot discriminate", build)
			}
			if want < 100 {
				sawSmall = true
			}

			env := corpusEnvForLoadedData(t, build)
			got, present := env.Player.Output["EffectiveSpellSuppressionChance"]
			if !present {
				t.Errorf("%s: we never write EffectiveSpellSuppressionChance at all; PoB reports %v",
					build, want)
				return
			}
			if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: EffectiveSpellSuppressionChance = %v, PoB reports %v", build, got, want)
			}

			if wantBase, ok := rec.Output["SpellSuppressionChance"]; ok {
				if gotBase := env.Player.Output["SpellSuppressionChance"]; math.Abs(gotBase-wantBase) > math.Abs(wantBase)*oracleDumpTolerance {
					t.Errorf("%s: SpellSuppressionChance = %v, PoB reports %v -- the Effective value "+
						"must be derived, not written back over the base", build, gotBase, wantBase)
				}
			}
		})
	}
	if !sawSmall {
		t.Fatal("REFUSING: every fixture reports 100%, where a clamping or squaring port would also " +
			"agree; the set has lost its discriminating cases")
	}
}
