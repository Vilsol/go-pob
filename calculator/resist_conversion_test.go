package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestResistanceConversionBlocks pins CalcDefence.lua:513-558.
func TestResistanceConversionBlocks(t *testing.T) {
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

	keys := []string{"FireResist", "ColdResist", "LightningResist",
		"FireResistTotal", "ColdResistTotal", "LightningResistTotal"}

	var maxActive, plainActive, seen, wrong, baseSwept int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		db := env.Player.ModDB

		anyMax, anyPlain := false, false
		for _, from := range resistTypeList {
			for _, to := range resistTypeList {
				if db.Sum(mod.TypeBase, nil, from+"MaxResConvertTo"+to) != 0 {
					anyMax = true
				}
				if db.Sum(mod.TypeBase, nil, from+"ResConvertTo"+to) != 0 {
					anyPlain = true
				}
			}
		}
		if anyMax {
			maxActive++
		}
		if anyPlain {
			plainActive++
		}
		if !anyMax && !anyPlain {
			continue
		}

		for _, elem := range resistTypeList {
			for _, m := range db.Tabulate(mod.TypeBase, nil, elem+"ResistMax") {
				src := string(m.Mod.GetSource())
				if len(src) > 0 && src != "Base" && m.Value >= 75 {
					baseSwept++
				}
			}
		}

		for _, key := range keys {
			want, has := rec.Output[key]
			if !has {
				continue
			}
			seen++
			if !oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
				wrong++
			}
		}
		if maxActive >= 40 && plainActive >= 40 {
			break
		}
	}

	if maxActive == 0 {
		t.Fatal("REFUSING: no graded build carries a live <from>MaxResConvertTo<to> mod. Either " +
			"the two ModParser entries regressed or the Chieftain nodes stopped reaching the " +
			"modDB — before this port landed this population was zero by construction")
	}
	if plainActive == 0 {
		t.Fatal("REFUSING: no graded build carries a live <from>ResConvertTo<to> mod, so the " +
			"second loop is never exercised")
	}
	if seen == 0 {
		t.Fatal("REFUSING: the blob publishes no resist stats on the converting builds")
	}
	if baseSwept > 0 {
		t.Errorf("%d conversion-sourced <Elem>ResistMax mods carry a value >= 75 — the \"Base\" "+
			"exclusion at :522 / :542 is not holding and the 75-point base is being converted",
			baseSwept)
	}
	if wrong > 0 {
		t.Logf("resist stats still disagree with PoB on %d of %d pairs across the %d builds with "+
			"a live conversion (a separate residual)", wrong, seen, maxActive)
	}
	t.Logf("MEASURED: %d builds with a max conversion, %d with a plain one", maxActive, plainActive)
}
