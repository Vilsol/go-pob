package calculator

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestReducePoolsByDamageMatchesOracle grades ReducePoolsByDamage against PoB'S OWN RETURN VALUES.
func TestReducePoolsByDamageMatchesOracle(t *testing.T) {
	dumpDir := os.Getenv("REDUCE_POOLS_DUMP")
	if dumpDir == "" {
		t.Skip("set REDUCE_POOLS_DUMP=<dir> (task reduce-pools BUILDS=<list> OUT=<dir> NICE=nice)")
	}
	requireGameData(t, "3.29")

	files, err := filepath.Glob(filepath.Join(dumpDir, "*.json"))
	if err != nil {
		t.Fatalf("globbing %s: %v", dumpDir, err)
	}
	if len(files) == 0 {
		t.Fatalf("REFUSING: %s holds no .json dumps. Generate them with\n"+
			"  task reduce-pools BUILDS=<list file> OUT=%s NICE=nice", dumpDir, dumpDir)
	}

	type oraclePools struct {
		Ward                          *float64                      `json:"Ward"`
		WardActiveChance              *float64                      `json:"WardActiveChance"`
		EnergyShield                  *float64                      `json:"EnergyShield"`
		Mana                          *float64                      `json:"Mana"`
		Life                          *float64                      `json:"Life"`
		LifeLossLostOverTime          *float64                      `json:"LifeLossLostOverTime"`
		LifeBelowHalfLossLostOverTime *float64                      `json:"LifeBelowHalfLossLostOverTime"`
		OverkillDamage                *float64                      `json:"OverkillDamage"`
		HitPoolRemaining              *float64                      `json:"hitPoolRemaining"`
		ResourcesLost                 map[string]map[string]float64 `json:"resourcesLostToTypeDamage"`
	}
	type oracleEntry struct {
		Build    string             `json:"build"`
		Error    string             `json:"error"`
		NoDamage bool               `json:"noDamage"`
		DamageIn map[string]float64 `json:"damageIn"`
		OneHit   *oraclePools       `json:"oneHit"`
		TwoHits  *oraclePools       `json:"twoHits"`
		Inputs   map[string]any     `json:"inputs"`
	}

	cmp := func(t *testing.T, label string, got float64, want *float64) {
		t.Helper()
		if want == nil {
			return
		}
		if got != *want {
			t.Errorf("%s = %v, want %v (PoB)", label, got, *want)
		}
	}

	checked := 0
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		var e oracleEntry
		if err := json.Unmarshal(raw, &e); err != nil {
			t.Fatalf("parsing %s: %v", f, err)
		}
		if e.Error != "" {
			t.Errorf("%s: the DUMP itself errored: %s", e.Build, e.Error)
			continue
		}
		if e.NoDamage || e.OneHit == nil {
			continue
		}

		t.Run(e.Build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, e.Build)
			actor := env.Player

			mkDamage := func() map[data.DamageType]float64 {
				d := make(map[data.DamageType]float64, len(e.DamageIn))
				for k, v := range e.DamageIn {
					d[data.DamageType(k)] = v
				}
				return d
			}

			inputMismatch := false
			for _, in := range []struct {
				key  string
				want any
			}{
				{"Life", e.Inputs["Life"]},
				{"LifeRecoverable", e.Inputs["LifeRecoverable"]},
				{"ManaUnreserved", e.Inputs["ManaUnreserved"]},
				{"EnergyShieldRecoveryCap", e.Inputs["EnergyShieldRecoveryCap"]},
				{"Ward", e.Inputs["Ward"]},
				{"preventedLifeLoss", e.Inputs["preventedLifeLoss"]},
				{"preventedLifeLossBelowHalf", e.Inputs["preventedLifeLossBelowHalf"]},
			} {
				wf, ok := in.want.(float64)
				if !ok {
					continue
				}
				if gf := actor.Output[in.key]; math.Abs(gf-wf) > 1e-6 {
					t.Errorf("INPUT %s = %v, PoB has %v — the pool comparison below is VOID on this "+
						"build: the two sides start from different numbers, and that is an upstream "+
						"defect, not a ReducePoolsByDamage defect", in.key, gf, wf)
					inputMismatch = true
				}
			}
			if inputMismatch {
				return
			}

			got := ReducePoolsByDamage(nil, mkDamage(), actor)
			cmp(t, "oneHit.Life", deref(got.Life), e.OneHit.Life)
			cmp(t, "oneHit.Mana", deref(got.Mana), e.OneHit.Mana)
			cmp(t, "oneHit.EnergyShield", deref(got.EnergyShield), e.OneHit.EnergyShield)
			cmp(t, "oneHit.Ward", deref(got.Ward), e.OneHit.Ward)
			cmp(t, "oneHit.OverkillDamage", got.OverkillDamage, e.OneHit.OverkillDamage)
			cmp(t, "oneHit.hitPoolRemaining", got.HitPoolRemaining, e.OneHit.HitPoolRemaining)
			cmp(t, "oneHit.LifeLossLostOverTime", got.LifeLossLostOverTime, e.OneHit.LifeLossLostOverTime)
			cmp(t, "oneHit.LifeBelowHalfLossLostOverTime", got.LifeBelowHalfLossLostOverTime,
				e.OneHit.LifeBelowHalfLossLostOverTime)

			for dtName, want := range e.OneHit.ResourcesLost {
				dt := data.DamageType(dtName)
				gotRow := got.ResourcesLost[dt]
				for key, wv := range want {
					if math.Abs(gotRow[key]-wv) > 1e-6 {
						t.Errorf("oneHit.resourcesLost[%s][%s] = %v, want %v (PoB)",
							dtName, key, gotRow[key], wv)
					}
				}
				for key, gv := range gotRow {
					if _, ok := want[key]; !ok {
						t.Errorf("oneHit.resourcesLost[%s][%s] = %v, PoB has NO such key",
							dtName, key, gv)
					}
				}
			}

			if e.TwoHits != nil {
				first := ReducePoolsByDamage(nil, mkDamage(), actor)
				second := ReducePoolsByDamage(first, mkDamage(), actor)
				cmp(t, "twoHits.Life", deref(second.Life), e.TwoHits.Life)
				cmp(t, "twoHits.Mana", deref(second.Mana), e.TwoHits.Mana)
				cmp(t, "twoHits.EnergyShield", deref(second.EnergyShield), e.TwoHits.EnergyShield)
				cmp(t, "twoHits.Ward", deref(second.Ward), e.TwoHits.Ward)
				cmp(t, "twoHits.OverkillDamage", second.OverkillDamage, e.TwoHits.OverkillDamage)
				cmp(t, "twoHits.hitPoolRemaining", second.HitPoolRemaining, e.TwoHits.HitPoolRemaining)
			}
		})
		checked++
	}

	if checked == 0 {
		t.Fatalf("REFUSING: %d dump files and NONE carried a comparable hit. That is a broken "+
			"dump, not a passing test.", len(files))
	}
	t.Logf("compared %d build(s) against PoB's own reducePoolsByDamage", checked)
}

func deref(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
