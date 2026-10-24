package calculator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var _ = registerCorpusCheck("TestAuraAffectingSelfMatchesPoB", func() corpusCheck {
	dir, skip := dumpsDirCandidate()
	if skip != "" {
		dir = ""
	}
	return &auraAffectingSelfCheck{dir: dir}
})

type auraAffectingSelfCheck struct {
	mu                               sync.Mutex
	dir                              string
	graded, agree, oursHigh, oursLow int
}

func (c *auraAffectingSelfCheck) visit(b string, _ oracleRecord, _ bool, env *Environment) {
	if c.dir == "" {
		return
	}
	raw, err := os.ReadFile(filepath.Join(c.dir, b[:len(b)-len(".xml")]+".json"))
	if err != nil {
		return
	}
	var d struct {
		Multipliers map[string]float64 `json:"multipliers"`
	}
	if json.Unmarshal(raw, &d) != nil || d.Multipliers == nil {
		return
	}
	want, ok := d.Multipliers["AuraAffectingSelf"]
	if !ok {
		return
	}
	if env == nil || env.Player == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.graded++
	switch got := env.Player.ModDB.Multipliers["AuraAffectingSelf"]; {
	case got == want:
		c.agree++
	case got > want:
		c.oursHigh++
	default:
		c.oursLow++
	}
}

// TestAuraAffectingSelfMatchesPoB: `Multiplier:AuraAffectingSelf` is a COUNT, and it was over-counted on a third of the corpus.
//
//	if     skillTypes[Herald] and not heraldList[name]        then Herald++
//	elseif skillTypes[Aura] and not skillTypes[AuraAffectsEnemies]
//	       and not auraList[name]                             then AuraAffectingSelf++
func TestAuraAffectingSelfMatchesPoB(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	dir := dumpsDirForLoadedData(t)
	c := corpusWalkCheck[*auraAffectingSelfCheck](t)
	if c.dir != dir {
		t.Fatalf("REFUSING: the shared walk read dumps from %s, this test resolved %s", c.dir, dir)
	}
	graded, agree, oursHigh, oursLow := c.graded, c.agree, c.oursHigh, c.oursLow

	if graded == 0 {
		t.Fatal("REFUSING: no dump carries an AuraAffectingSelf multiplier, so this sweep is not " +
			"reaching the herald/aura block at all and its counts mean nothing")
	}
	if agree == 0 {
		t.Fatal("REFUSING: the count agrees on NO build — that reads as the multiplier having " +
			"stopped being written rather than as a worse port")
	}

	t.Logf("Multiplier:AuraAffectingSelf: %d graded, %d agree (%.1f%%), %d ours HIGH, %d ours LOW",
		graded, agree, 100*float64(agree)/float64(graded), oursHigh, oursLow)

	const floor = 7413
	if agree < floor {
		t.Errorf("AuraAffectingSelf agrees on %d of %d builds, floor %d — the herald/aura loop at "+
			"CalcPerform.lua:2116-2125 is one if/elseif chain, and splitting it double-counts any "+
			"skill that is both", agree, graded, floor)
	}
	const highCeiling = 0
	if oursHigh > highCeiling {
		t.Errorf("AuraAffectingSelf is over-counted on %d builds (ceiling %d)", oursHigh, highCeiling)
	}
}
