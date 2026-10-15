package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMinionWitherArmApplies covers CalcPerform.lua:2704's SECOND DISJUNCT.
func TestMinionWitherArmApplies(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, name := range []string{
		"0632-necromancer-animate-weapon-of-ranged-arms-15.xml",
		"0478-necromancer-animate-weapon-of-ranged-arms-10.xml",
	} {
		rec, ok := oracle[name]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob; this test has lost its subject", name)
		}
		env := corpusEnvForLoadedData(t, name)

		if env.Minion == nil || env.Minion.ModDB == nil {
			t.Fatalf("REFUSING: %s builds no minion actor", name)
		}
		if !env.Minion.ModDB.Flag(nil, "Condition:CanWither") {
			t.Fatalf("PREMISE BROKEN on %s: the minion does not carry Condition:CanWither, so the "+
				"second disjunct cannot be what this build exercises", name)
		}
		if env.EnemyModDB.Sum(mod.TypeBase, nil, "Multiplier:WitheredStack") == 0 {
			t.Fatalf("PREMISE BROKEN on %s: the enemy carries no WitheredStack multiplier, so "+
				"WitherEffectStack reaches no graded number and this build cannot show the fix", name)
		}
		if got := env.ModDB.Max(nil, "WitherEffectStack"); got != 6 {
			t.Errorf("%s WitherEffectStack = %v, want 6", name, got)
		}

		graded := 0
		for _, key := range []string{"Minion.CombinedDPS", "Minion.AverageDamage", "FullDPS"} {
			want, ok := rec.Output[key]
			if !ok || want == 0 {
				continue
			}
			graded++
			got := env.Minion.Output[keyAfterMinionPrefix(key)]
			if keyAfterMinionPrefix(key) == key {
				got = env.Player.Output[key]
			}
			if !oracleClose(got, want, oracleDumpTolerance) {
				t.Errorf("%s %s = %g, PoB %g (ratio %.4f)", name, key, got, want, got/want)
			}
		}
		if graded == 0 {
			t.Errorf("REFUSING on %s: none of Minion.CombinedDPS / Minion.AverageDamage / FullDPS "+
				"is present and nonzero in the oracle, so this build asserted nothing about the "+
				"graded consequence", name)
		}
	}
}

// TestMinionWitherReachesTheEnemy grades the multiplier ITSELF rather than a DPS aggregate downstream of it.
//
//pob:block:src/Modules/CalcPerform.lua:2756-2763
func TestMinionWitherReachesTheEnemy(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	builds := []string{
		"0478-necromancer-animate-weapon-of-ranged-arms-10.xml",
		"0632-necromancer-animate-weapon-of-ranged-arms-15.xml",
		"0801-necromancer-animate-guardians-weapon-2.xml",
		"0903-necromancer-animate-weapon-of-ranged-arms-19.xml",
		"1081-necromancer-summon-carrion-golem-of-scavenging-2.xml",
		"1551-necromancer-vaal-animate-weapon-of-ranged-arms.xml",
	}

	graded := 0
	for _, build := range builds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob; this test has lost a subject", build)
		}
		want, ok := rec.Minion["PoisonEffMult"]
		if !ok || want == 1 || want == 0 {
			t.Fatalf("REFUSING: PoB reports Minion.PoisonEffMult = %v for %s — a fixture at "+
				"exactly 1 is the DEFECT's own value and cannot see the disjunct being dropped",
				want, build)
		}
		graded++

		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			if env.Minion == nil || env.Minion.Actor == nil {
				t.Fatalf("REFUSING: %s built no minion", build)
			}
			got := env.Minion.Output["PoisonEffMult"]
			if !oracleClose(got, want, oracleDumpTolerance) {
				t.Errorf("%s: Minion.PoisonEffMult = %v, PoB reports %v (ratio %.5f)",
					build, got, want, got/want)
			}
		})
	}

	if graded != len(builds) {
		t.Fatalf("REFUSING: graded %d of %d fixtures", graded, len(builds))
	}
}

func keyAfterMinionPrefix(key string) string {
	const p = "Minion."
	if len(key) > len(p) && key[:len(p)] == p {
		return key[len(p):]
	}
	return key
}
