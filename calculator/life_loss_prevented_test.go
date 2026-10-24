package calculator

import (
	"os"
	"sort"
	"sync"
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

var _ = registerCorpusCheck("TestLifeLossPreventedReachesTheHitPool", func() corpusCheck { return &lifeLossPreventedCheck{} })

type lifeLossPreventedCarrier struct {
	name               string
	exceedsRecoverable bool
	seen, wrong        int
}

type lifeLossPreventedCheck struct {
	mu       sync.Mutex
	carriers []lifeLossPreventedCarrier
}

func (c *lifeLossPreventedCheck) visit(name string, rec oracleRecord, graded bool, env *Environment) {
	if !graded || env.Player.ModDB.Sum(mod.TypeBase, nil, "LifeLossPrevented") <= 0 {
		return
	}
	carrier := lifeLossPreventedCarrier{
		name:               name,
		exceedsRecoverable: env.Player.Output["LifeHitPool"] > env.Player.Output["LifeRecoverable"],
	}
	for _, key := range []string{"LifeHitPool", "sharedMoMHitPool"} {
		want, has := rec.Output[key]
		if !has || want == 0 {
			continue
		}
		carrier.seen++
		if !oracleClose(env.Player.Output[key], want, oracleDumpTolerance) {
			carrier.wrong++
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.carriers = append(c.carriers, carrier)
}

// TestLifeLossPreventedReachesTheHitPool is the activation half: the mod must be live on real builds and LifeHitPool must move with it.
func TestLifeLossPreventedReachesTheHitPool(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	c := corpusWalkCheck[*lifeLossPreventedCheck](t)
	sort.Slice(c.carriers, func(i, j int) bool { return c.carriers[i].name < c.carriers[j].name })
	var live, exceedsRecoverable, seen, wrong int
	for _, carrier := range c.carriers {
		live++
		if carrier.exceedsRecoverable {
			exceedsRecoverable++
		}
		seen += carrier.seen
		wrong += carrier.wrong
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
