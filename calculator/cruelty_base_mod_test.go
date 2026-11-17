package calculator

import (
	"os"
	"testing"
)

// TestCrueltyGrantsItsMultiplier: Cruelty's baseMods flag (sup_str.lua:1538-1540) reaches the
// main skill, so CalcPerform.lua:648-650 sets Multiplier:Cruelty and ignites carry its more.
func TestCrueltyGrantsItsMultiplier(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	for _, build := range []string{"0078-elementalist-penance-brand.xml", "0255-elementalist-penance-brand-9.xml"} {
		t.Run(build, func(t *testing.T) {
			oracles, err := oracleRecordsForBuilds(build)
			if err != nil {
				t.Fatal(err)
			}
			env := corpusEnvPinnedForLoadedData(t, build)
			for _, p := range oracleTablePairs(oracles[build], env) {
				if p.prefix != "" {
					continue
				}
				for _, stat := range []string{"IgniteFireMin", "IgniteLightningMax", "IgniteDPS"} {
					want, ok := p.exp[stat]
					if !ok {
						t.Fatalf("oracle has no %s", stat)
					}
					if got := p.got[stat]; !oracleClose(got, want, oracleDumpTolerance) {
						t.Errorf("%s = %v, PoB %v", stat, got, want)
					}
				}
			}
		})
	}
}
