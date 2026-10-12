package calculator

import (
	"fmt"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestBlindAndImpaleAvoidChancePinCalcDefence1548 pins CalcDefence.lua:1548-1549.
//
//	output.BlindAvoidChance  = modDB:Flag(nil, "BlindImmune")  and 100 or m_min(modDB:Sum("BASE", nil, "AvoidBlind"), 100)
//	output.ImpaleAvoidChance = modDB:Flag(nil, "ImpaleImmune") and 100 or m_min(modDB:Sum("BASE", nil, "AvoidImpale"), 100)
func TestBlindAndImpaleAvoidChancePinCalcDefence1548(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	type arm struct {
		key       string
		flag      string
		sumName   string
		seen      int
		wrong     int
		witnesses []string
	}
	arms := []*arm{
		{key: "BlindAvoidChance", flag: "BlindImmune", sumName: "AvoidBlind"},
		{key: "ImpaleAvoidChance", flag: "ImpaleImmune", sumName: "AvoidImpale"},
	}

	builds := 0
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		db := env.Player.ModDB
		builds++

		for _, a := range arms {
			if !db.Flag(nil, a.flag) && db.Sum(mod.TypeBase, nil, a.sumName) == 0 {
				continue
			}
			a.seen++
			want, has := rec.Output[a.key]
			if !has {
				continue
			}
			if got := env.Player.Output[a.key]; !oracleClose(got, want, oracleDumpTolerance) {
				a.wrong++
				if len(a.witnesses) < 5 {
					a.witnesses = append(a.witnesses, fmt.Sprintf("%s(got %.4f want %.4f)", name, got, want))
				}
			}
		}

		done := true
		for _, a := range arms {
			if a.seen < 20 {
				done = false
			}
		}
		if done {
			break
		}
	}

	if builds == 0 {
		t.Fatal("REFUSING: no graded build was built; this test measured nothing")
	}
	for _, a := range arms {
		if a.seen == 0 {
			t.Fatalf("REFUSING: over %d builds not one carries %s or a nonzero %s, so %s is "+
				"decided by nothing and this arm no longer measures its mechanism",
				builds, a.flag, a.sumName, a.key)
		}
		if a.wrong > 0 {
			t.Errorf("%s disagrees with PoB on %d of %d builds carrying %s or %s: %v",
				a.key, a.wrong, a.seen, a.flag, a.sumName, a.witnesses)
		}
	}
}
