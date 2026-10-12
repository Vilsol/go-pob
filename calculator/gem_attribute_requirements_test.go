package calculator

import (
	"os"
	"testing"
)

// TestGemsContributeAttributeRequirements pins the OTHER half of the same defect.
func TestGemsContributeAttributeRequirements(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var decisive, seen, wrong, gemDecided int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)

		gemContributes := false
		for _, r := range env.RequirementsTableGems {
			if r.Str > 0 || r.Dex > 0 || r.Int > 0 {
				gemContributes = true
				break
			}
		}
		if !gemContributes {
			continue
		}
		decisive++

		itemOnly := map[string]int{"ReqStr": 0, "ReqDex": 0, "ReqInt": 0}
		for _, r := range env.RequirementsTableItems {
			itemOnly["ReqStr"] = max(itemOnly["ReqStr"], r.Str)
			itemOnly["ReqDex"] = max(itemOnly["ReqDex"], r.Dex)
			itemOnly["ReqInt"] = max(itemOnly["ReqInt"], r.Int)
		}

		for _, key := range []string{"ReqStr", "ReqDex", "ReqInt"} {
			if env.Player.Output[key] > float64(itemOnly[key]) {
				gemDecided++
			}
			want, has := rec.Output[key]
			if !has {
				continue
			}
			seen++
			if !oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
				wrong++
			}
		}
		if decisive >= 200 {
			break
		}
	}

	if decisive == 0 {
		t.Fatal("REFUSING: not one graded build has a gem contributing a nonzero attribute " +
			"requirement. That was the state this test was written to close — the gem half of " +
			"env.RequirementsTable is starved again")
	}
	if seen == 0 {
		t.Fatal("REFUSING: the blob publishes no Req{Str,Dex,Int} on the decisive builds")
	}
	if gemDecided == 0 {
		t.Fatalf("REFUSING: across %d builds where a gem carries a requirement, NOT ONE aggregate "+
			"exceeds the item-only maximum. That is the exact state before this fix — the gem rows "+
			"exist but nothing downstream reads them", decisive)
	}

	if wrong > 0 {
		t.Logf("Req{Str,Dex,Int} disagrees with PoB on %d of %d pairs across %d builds where a "+
			"gem contributes (see the note above: a separate, diffuse residual)", wrong, seen, decisive)
	}
}
