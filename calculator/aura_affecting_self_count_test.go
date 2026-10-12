package calculator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
	_, names := corpusNamesForLoadedData(t)

	graded, agree, oursHigh, oursLow := 0, 0, 0, 0
	for _, b := range names {
		raw, err := os.ReadFile(filepath.Join(dir, b[:len(b)-len(".xml")]+".json"))
		if err != nil {
			continue
		}
		var d struct {
			Multipliers map[string]float64 `json:"multipliers"`
		}
		if json.Unmarshal(raw, &d) != nil || d.Multipliers == nil {
			continue
		}
		want, ok := d.Multipliers["AuraAffectingSelf"]
		if !ok {
			continue
		}
		env := corpusEnvForLoadedData(t, b)
		if env == nil || env.Player == nil {
			continue
		}
		graded++
		switch got := env.Player.ModDB.Multipliers["AuraAffectingSelf"]; {
		case got == want:
			agree++
		case got > want:
			oursHigh++
		default:
			oursLow++
		}
	}

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
