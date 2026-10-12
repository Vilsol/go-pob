package calculator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

var flaskChargeGenOutputs = []string{
	"FlaskChargeGen", "LifeFlaskChargeGen", "ManaFlaskChargeGen",
	"UtilityFlaskChargeGen", "FlaskEffect", "FlaskChargeOnCritChance",
}

// TestFlaskChargeGenIsCalcsOnly pins the CONDITION that would retire a declared omission, rather than the omission itself.
func TestFlaskChargeGenIsCalcsOnly(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true required (reads PoB's dumps)")
	}
	dir := diagDumpsDir(t)
	names := diagCorpusNames(t)
	if len(names) > 150 {
		names = names[:150]
	}

	var scanned, inCalcs int
	inMain := map[string][]string{}
	for _, b := range names {
		raw, err := os.ReadFile(filepath.Join(dir, b+".json"))
		if err != nil {
			continue
		}
		var d struct {
			Output      map[string]json.RawMessage `json:"output"`
			CalcsOutput map[string]json.RawMessage `json:"calcsOutput"`
		}
		if json.Unmarshal(raw, &d) != nil {
			continue
		}
		scanned++
		for _, k := range flaskChargeGenOutputs {
			if _, ok := d.CalcsOutput[k]; ok {
				inCalcs++
				break
			}
		}
		for _, k := range flaskChargeGenOutputs {
			if _, ok := d.Output[k]; ok {
				inMain[k] = append(inMain[k], b)
			}
		}
	}

	if scanned == 0 {
		t.Fatalf("REFUSING: read 0 dumps from %s — an empty scan would report this property as "+
			"holding when nothing was examined", dir)
	}
	if inCalcs != scanned {
		t.Fatalf("REFUSING: only %d of %d dumps carry any flask-charge key in calcsOutput. PoB "+
			"writes all six there on every build; if it no longer does, this test's premise is "+
			"gone and the CALCS-only claim needs re-deriving, not re-asserting", inCalcs, scanned)
	}
	if len(inMain) > 0 {
		keys := make([]string, 0, len(inMain))
		for k := range inMain {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		t.Fatalf("CalcPerform.lua:1579's `if breakdown then` gate has MOVED: %v now appear in "+
			"MAIN-mode output (e.g. %s). The flask-charge family is gradeable and the reader that "+
			"was deliberately not ported now has to be — see TestUnreadProducerSweep's header.",
			keys, inMain[keys[0]][0])
	}
	t.Logf("%d dumps: all carry the flask-charge outputs in calcsOutput, none in output — "+
		"display-only, correctly absent here", scanned)
}
