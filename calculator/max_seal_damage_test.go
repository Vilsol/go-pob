package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestMaxSealDamageParses pins ModParser.lua:3075 (The Primal Owl, TreeData/3_29/tree.lua:13619).
func TestMaxSealDamageParses(t *testing.T) {
	e := ParseMod("Unsealed Spells gain 5% more Damage each time their effects Reoccur", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 {
		t.Fatalf("ParseMod = %v, want 1 mod", e)
	}
	if m := e.ModList[0]; m.Name() != "MaxSealDamage" || m.Type() != mod.TypeMore || m.Value().Float() != 5 {
		t.Errorf("got %s %s %v, want MaxSealDamage MORE 5", m.Name(), m.Type(), m.Value().Float())
	}
	c := ParseMod("Skills supported by Unleash have +3 to maximum number of Seals", false)
	if c == nil || c.Extra != "" || len(c.ModList) != 1 || c.ModList[0].Name() != "SealCount" {
		t.Errorf("control :13620: ParseMod = %v", c)
	}
}

// TestMaxSealDamageReachesDamage pins CalcOffence.lua:923-926: with UseMaxUnleash, each MaxSealDamage MORE becomes a Damage MORE of value*SealMax.
func TestMaxSealDamageReachesDamage(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0810-reliquarian-arc-of-oscillating-2.xml")
	out := env.Player.Output
	if out["SealMax"] != 8 {
		t.Fatalf("premise broken: SealMax = %v, want 8", out["SealMax"])
	}
	if got, want := out["AverageHit"], 2180106.0373292; math.Abs(got-want) > 1e-6*want {
		t.Errorf("AverageHit = %v, want %v (PoB)", got, want)
	}
}
