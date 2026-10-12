package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestCountAsHavingMaximumCharges pins ModParser.lua:5219-5221 and CalcPerform.lua:1072-1080.
func TestCountAsHavingMaximumCharges(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		line string
		flag string
	}{
		{"Count as having maximum number of Power Charges", "HaveMaximumPowerCharges"},
		{"Count as having maximum number of Frenzy Charges", "HaveMaximumFrenzyCharges"},
		{"Count as having maximum number of Endurance Charges", "HaveMaximumEnduranceCharges"},
	} {
		e := ParseMod(tc.line, false)
		if e == nil {
			t.Fatalf("no entry: %s", tc.line)
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q (line is DROPPED WHOLE): %s", extra, tc.line)
			continue
		}
		var got []string
		for _, m := range e.ModList {
			got = append(got, m.Name())
		}
		if !contains(got, tc.flag) {
			t.Errorf("produced %v, want %s: %s", got, tc.flag, tc.line)
		}
	}

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)
	const name = "0339-reliquarian-shield-crush-of-the-chieftain-2.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	file, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(strings.ToLower(string(file)), "maximum number of endurance charges") {
		t.Fatalf("REFUSING: %s no longer carries the modifier under test", name)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	for _, k := range []string{"BasePhysicalDamageReduction", "Armour"} {
		want := rec.Output[k]
		got := env.Player.Output[k]
		if math.Abs(got-want) > 0.005*math.Abs(want) {
			t.Errorf("%s: %s = %v, want %v", name, k, got, want)
		}
	}
}
