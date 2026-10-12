package calculator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestNonVaalSummonedMinionCensus answers the ONE question no missing-thing instrument here can: which DIRECTION does Multiplier:NonVaalSummonedMinion diverge in.
func TestNonVaalSummonedMinionCensus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" || os.Getenv("NONVAAL_CENSUS") != "1" {
		t.Skip("set CORPUS=true NONVAAL_CENSUS=1 to run the direction census")
	}
	const name = "Multiplier:NonVaalSummonedMinion"
	dir := diagDumpsDir(t)
	names := diagCorpusNames(t)
	limit := len(names)
	if v := os.Getenv("CENSUS_LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			if n < limit {
				limit = n
			}
		} else {
			t.Fatalf("REFUSING: CENSUS_LIMIT=%q is not a positive integer — a defaulted limit that "+
				"happens to be the maximum is the worst failure mode a sampling knob has", v)
		}
	}

	var considered, agree, oursLow, oursHigh, pobOverride, pobAbsentOursPresent int
	var examplesLow, examplesHigh, examplesAbsent []string
	for _, b := range names[:limit] {
		d, healthy := diagLoadDump(dir, b)
		if d == nil || !healthy {
			continue
		}
		entries := d.ModDB[name]
		var pobVal float64
		var hasOverride bool
		for _, e := range entries {
			var f float64
			if json.Unmarshal(e.Value, &f) != nil {
				continue
			}
			switch e.Type {
			case "OVERRIDE":
				pobVal, hasOverride = f, true
			case "BASE":
				if !hasOverride {
					pobVal += f
				}
			}
		}
		if hasOverride {
			pobOverride++
		}

		xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), b+".xml"))
		if err != nil {
			continue
		}
		parsed, err := builds.ParseBuild(xml)
		if err != nil {
			continue
		}
		env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)
		ours := env.Player.ModDB.Sum(mod.TypeBase, nil, name)
		if o := env.Player.ModDB.Override(nil, name); o != nil {
			ours = o.Float()
		}

		if len(entries) == 0 {
			if ours == 0 {
				continue
			}
			pobAbsentOursPresent++
			if len(examplesAbsent) < 5 {
				examplesAbsent = append(examplesAbsent, b)
			}
		}
		considered++
		switch {
		case ours == pobVal:
			agree++
		case ours < pobVal:
			oursLow++
			if len(examplesLow) < 5 {
				examplesLow = append(examplesLow, b)
			}
		default:
			oursHigh++
			if len(examplesHigh) < 5 {
				examplesHigh = append(examplesHigh, b)
			}
		}
	}

	if considered == 0 {
		t.Fatalf("REFUSING: no build carried %s — an empty result here would read as agreement", name)
	}
	t.Logf("=== %s DIRECTION CENSUS === %d builds carry it (of %d scanned)", name, considered, limit)
	t.Logf("  agree        %d (%.1f%%)", agree, 100*float64(agree)/float64(considered))
	t.Logf("  ours LOW     %d   e.g. %v", oursLow, examplesLow)
	t.Logf("  ours HIGH    %d   e.g. %v", oursHigh, examplesHigh)
	t.Logf("  PoB emits an OVERRIDE (ConfigOptions.lua:1131) on %d — go-pob implements none of these", pobOverride)
	t.Logf("  PoB ABSENT / ours PRESENT %d   e.g. %v   <- the pure over-application case", pobAbsentOursPresent, examplesAbsent)
}
