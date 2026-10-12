package calculator

import (
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestOathOfSpringShockRowsParse pins ModParser.lua:3028 and :3029, the two lines on the Warden notable Oath of Spring (tree node 31364).
func TestOathOfSpringShockRowsParse(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line failed to parse; this test cannot attribute a failure")
	}

	parse := func(t *testing.T, line string) []mod.Mod {
		t.Helper()
		e := ParseMod(line, false)
		if e == nil {
			t.Fatalf("no entry: %s", line)
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Fatalf("leftover %q (the line is DROPPED WHOLE): %s", extra, line)
		}
		return e.ModList
	}

	find := func(t *testing.T, mods []mod.Mod, name string) mod.Mod {
		t.Helper()
		for _, m := range mods {
			if m.Name() == name {
				return m
			}
		}
		t.Fatalf("no %s among %v", name, modNamesOf(mods))
		return nil
	}

	t.Run("ShockMaxOverride", func(t *testing.T) {
		mods := parse(t, "Maximum Effect of Shock is 2% increased Damage taken")
		m := find(t, mods, "ShockMax")
		if m.Type() != mod.TypeOverride {
			t.Errorf("ShockMax type is %v, want OVERRIDE — a BASE here ADDS 2 to the default "+
				"cap of 50 instead of REPLACING it", m.Type())
		}
	})

	t.Run("ShockStacksMaxOverride", func(t *testing.T) {
		mods := parse(t, "You can apply up to 50 Shocks to each Enemy")
		find(t, mods, "ShockCanStack")
		m := find(t, mods, "ShockStacksMax")
		if m.Type() != mod.TypeOverride {
			t.Errorf("ShockStacksMax type is %v, want OVERRIDE — ailmentStacksMax reads the "+
				"override arm, and a BASE would be summed with CalcSetup's seed of 1", m.Type())
		}
	})

	t.Run("neighbours", func(t *testing.T) {
		base := find(t, parse(t, "+40% to Maximum Effect of Shock"), "ShockMax")
		if base.Type() != mod.TypeBase {
			t.Errorf("ModParser.lua:3591 now emits %v, want BASE — the new OVERRIDE row has "+
				"swallowed it", base.Type())
		}
		other := find(t, parse(t, "Your Shocks can increase Damage taken by up to a Maximum of 60%"), "ShockMax")
		if other.Type() != mod.TypeOverride {
			t.Errorf("ModParser.lua:3590 now emits %v, want OVERRIDE", other.Type())
		}
	})
}

func modNamesOf(mods []mod.Mod) []string {
	out := make([]string, 0, len(mods))
	for _, m := range mods {
		out = append(out, m.Name())
	}
	return out
}

// TestOathOfSpringShockCapReachesTheOutput is the end-to-end half: a row that parses is not a row that ARRIVES.
func TestOathOfSpringShockCapReachesTheOutput(t *testing.T) {
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

	var seen, wrong int
	var wrongNames []string
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		if env.Player.ModDB.Override(nil, "ShockMax") == nil {
			continue
		}
		seen++
		if want, has := rec.Output["MaximumShock"]; has &&
			!oracleClose(env.Player.Output["MaximumShock"], want, oracleDumpTolerance) {
			wrong++
			if len(wrongNames) < 5 {
				wrongNames = append(wrongNames, name)
			}
		}
		if seen >= 40 {
			break
		}
	}
	if seen == 0 {
		t.Fatal("REFUSING: no graded corpus build carries a ShockMax OVERRIDE, so the row " +
			"reaches nothing and this test measured nothing")
	}
	if wrong > 0 {
		t.Errorf("MaximumShock disagrees with PoB on %d of %d builds carrying a ShockMax "+
			"OVERRIDE: %v", wrong, seen, wrongNames)
	}
}
