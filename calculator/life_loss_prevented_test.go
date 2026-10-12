package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestLifeLossPreventedIsParsed pins ModParser.lua:5082 — Progenesis.
func TestLifeLossPreventedIsParsed(t *testing.T) {
	t.Parallel()

	const line = "When Hit during effect, 25% of Life Loss from Damage taken occurs over 4 seconds instead"

	entry := ParseMod(line, false)
	if entry.ModList == nil {
		t.Fatalf("%q did not parse at all", line)
	}
	found := false
	for _, m := range entry.ModList {
		if m.Name() != "LifeLossPrevented" {
			continue
		}
		found = true
		if got := m.Value().Float(); got != 25 {
			t.Errorf("LifeLossPrevented = %v, want 25", got)
		}
	}
	if !found {
		t.Errorf("%q produced no LifeLossPrevented mod; got %d mod(s)", line, len(entry.ModList))
	}
}

// TestLifeLossPreventedReachesTheHitPool is the activation half: the mod must be live on real builds and LifeHitPool must move with it.
func TestLifeLossPreventedReachesTheHitPool(t *testing.T) {
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

	var live, exceedsRecoverable, seen, wrong int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		if env.Player.ModDB.Sum(mod.TypeBase, nil, "LifeLossPrevented") <= 0 {
			continue
		}
		live++
		if env.Player.Output["LifeHitPool"] > env.Player.Output["LifeRecoverable"] {
			exceedsRecoverable++
		}
		for _, key := range []string{"LifeHitPool", "sharedMoMHitPool"} {
			want, has := rec.Output[key]
			if !has || want == 0 {
				continue
			}
			seen++
			if !oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
				wrong++
			}
		}
		if live >= 60 {
			break
		}
	}

	if live == 0 {
		t.Fatal("REFUSING: no graded build carries a live LifeLossPrevented mod. That was the " +
			"state before this parser entry — the population was zero by construction, so an " +
			"empty one here means the entry regressed or the flask stopped reaching the modDB")
	}
	if exceedsRecoverable == 0 {
		t.Errorf("across %d builds with LifeLossPrevented live, NOT ONE has LifeHitPool above "+
			"LifeRecoverable — the value is reaching the modDB but not the hit pool", live)
	}
	if seen == 0 {
		t.Fatal("REFUSING: the blob publishes no LifeHitPool/sharedMoMHitPool on those builds")
	}
	if wrong > 0 {
		t.Logf("LifeHitPool/sharedMoMHitPool still disagree on %d of %d pairs across %d builds "+
			"(a separate residual)", wrong, seen, live)
	}
	t.Logf("MEASURED: %d builds carry a live LifeLossPrevented, %d show a raised hit pool",
		live, exceedsRecoverable)
}
