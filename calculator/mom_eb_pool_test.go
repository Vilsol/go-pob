package calculator

import (
	"math"
	"os"
	"testing"
)

// TestCalcMoMEBPoolF pins CalcDefence.lua:2307-2319.
//
//	local maxMoMPool    = MoMEffect < 1 and lifePool / (1 - MoMEffect) - lifePool or m_huge
//	local maxManaUsable = m_floor(m_min(mana, maxMoMPool))
//	local maxESUsable   = flag and esBypass < 1 and m_floor(m_min(EnergyShieldRecoveryCap,
//	local manaUsed      = m_floor(m_min(maxMoMPool - maxESUsable, maxManaUsable))
func TestCalcMoMEBPoolF(t *testing.T) {
	cases := []struct {
		name                    string
		lifePool, mom, esBypass float64
		mana, esCap             float64
		protects                bool
		want                    float64
	}{
		{"mana capped by headroom", 2974, 0.4, 0, 6585, 0, false, 4956},
		{"negative unreserved mana, ES only", 2846, 0.4, 0, -655, 645, true, 3491},
		{"mana and ES both contribute", 2868, 0.4, 0, 28, 1480, true, 4376},
		{"ES only, corpus-verifiable build", 2467, 0.4, 0, -893, 1439, true, 3906},
		{"ES cap ignored without the flag", 2868, 0.4, 0, 28, 1480, false, 2896},
		{"fully bypassed ES contributes nothing", 2868, 0.4, 1, 28, 1480, true, 2896},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _, _, _ := calcMoMEBPoolF(tc.lifePool, tc.mom, tc.esBypass, tc.mana, tc.esCap, tc.protects)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("calcMoMEBPoolF(life=%v, mom=%v, bypass=%v, mana=%v, esCap=%v, protects=%v) = %v, PoB %v",
					tc.lifePool, tc.mom, tc.esBypass, tc.mana, tc.esCap, tc.protects, got, tc.want)
			}
		})
	}
}

// TestMoMHitPoolIsPublished covers CalcDefence.lua:2320-2377 — the wiring around the function above, on the corpus build whose inputs come closest to PoB's.
func TestMoMHitPoolIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const relTolerance = 0.005
	closeEnough := func(got, want float64) bool {
		if want == 0 {
			return math.Abs(got) < 1e-9
		}
		return math.Abs(got-want)/math.Abs(want) <= relTolerance
	}

	const build = "0504-assassin-lancing-steel-of-spraying-4.xml"
	env := corpusEnvForLoadedData(t, build)
	out := env.Player.Output

	for _, p := range []struct {
		name string
		want float64
	}{
		{"LifeRecoverable", 2467},
		{"ManaUnreserved", -893},
		{"EnergyShieldRecoveryCap", 1439},
		{"sharedMindOverMatter", 40},
	} {
		if got := out[p.name]; !closeEnough(got, p.want) {
			t.Fatalf("premise broken: %s = %v, PoB %v — this build has MoM's inputs closer than "+
				"any other, so re-derive the fixture rather than tuning the pool", p.name, got, p.want)
		}
	}

	if got := out["sharedManaEffectiveLife"]; !closeEnough(got, 3906) {
		t.Errorf("sharedManaEffectiveLife = %v, PoB 3906", got)
	}
	for _, damageType := range []string{"Fire", "Cold", "Lightning", "Physical"} {
		if got := out[damageType+"ManaEffectiveLife"]; !closeEnough(got, 3906) {
			t.Errorf("%sManaEffectiveLife = %v, PoB 3906", damageType, got)
		}
	}

	if out["LifeHitPool"] != out["LifeRecoverable"] {
		t.Skipf("LifeHitPool (%v) has diverged from LifeRecoverable (%v) — a prevention producer "+
			"landed; re-pin the hit pool against PoB directly", out["LifeHitPool"], out["LifeRecoverable"])
	}
	if got, ok := out["sharedMoMHitPool"]; !ok {
		t.Error("sharedMoMHitPool absent; PoB publishes it on every corpus build")
	} else if !closeEnough(got, 3906) {
		t.Errorf("sharedMoMHitPool = %v, PoB 3906", got)
	}
	for _, damageType := range []string{"Fire", "Cold", "Lightning", "Physical", "Chaos"} {
		if _, ok := out[damageType+"MoMHitPool"]; !ok {
			t.Errorf("%sMoMHitPool absent; PoB publishes it on every corpus build", damageType)
		}
	}
}
