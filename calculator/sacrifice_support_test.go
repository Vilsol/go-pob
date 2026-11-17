package calculator

import (
	"os"
	"testing"
)

// TestSacrificeAddsChaosFromLife: CalcOffence.lua:1914-1924 adds a share of unreserved (or totem)
// life as flat chaos damage to skills Sacrifice supports.
func TestSacrificeAddsChaosFromLife(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	for _, build := range []string{"1087-gladiator-divine-retribution.xml", "1110-gladiator-divine-retribution-2.xml"} {
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
				for _, stat := range []string{"ChaosMinBase", "ChaosMaxBase", "AverageHit"} {
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
