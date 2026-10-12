package calculator

import (
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestAvoidElementalAilmentsIsOneNameNotSeven pins ModParser.lua:407-408.
func TestAvoidElementalAilmentsIsOneNameNotSeven(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("+10 to Strength", false); e == nil || len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the control line failed to parse; this test cannot attribute a failure")
	}

	namesOf := func(t *testing.T, line string) []string {
		t.Helper()
		e := ParseMod(line, false)
		if e == nil {
			t.Fatalf("no entry: %s", line)
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Fatalf("leftover %q (the line is DROPPED WHOLE): %s", extra, line)
		}
		out := make([]string, 0, len(e.ModList))
		for _, m := range e.ModList {
			out = append(out, m.Name())
		}
		return out
	}

	for _, line := range []string{
		"20% chance to Avoid Elemental Ailments",
		"20% chance to Avoid Elemental Status Ailments",
	} {
		got := namesOf(t, line)
		if len(got) != 1 || got[0] != "AvoidElementalAilments" {
			t.Errorf("%q -> %v, want exactly [AvoidElementalAilments] (:407-408 map to ONE name; "+
				"the seven-name expansion double-counts under Stormshroud)", line, got)
		}
	}

	nonDamaging := namesOf(t, "20% chance to Avoid non-Damaging Ailments on you")
	if len(nonDamaging) != 6 {
		t.Errorf("%q -> %v, want the SIX-name expansion ModParser.lua:405 still has",
			"chance to Avoid non-Damaging Ailments on you", nonDamaging)
	}
	for _, want := range []string{"AvoidShock", "AvoidFreeze", "AvoidChill", "AvoidSap", "AvoidBrittle", "AvoidScorch"} {
		found := false
		for _, g := range nonDamaging {
			if g == want {
				found = true
			}
		}
		if !found {
			t.Errorf("non-damaging row lost %s: %v", want, nonDamaging)
		}
	}

	for line, want := range map[string]string{
		"20% chance to Avoid being Shocked": "AvoidShock",
		"20% chance to Avoid being Frozen":  "AvoidFreeze",
		"20% chance to Avoid being Chilled": "AvoidChill",
		"20% chance to Avoid being Ignited": "AvoidIgnite",
	} {
		got := namesOf(t, line)
		if len(got) != 1 || got[0] != want {
			t.Errorf("%q -> %v, want [%s]", line, got, want)
		}
	}
}

// TestAvoidElementalAilmentsHasAProducer is the other half.
func TestAvoidElementalAilmentsHasAProducer(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	_, names := corpusNamesForLoadedData(t)
	var seen, withStormshroud int
	for _, name := range names {
		env := corpusEnvForLoadedData(t, name)
		db := env.Player.ModDB
		if db.Sum(mod.TypeBase, nil, "AvoidElementalAilments") > 0 {
			seen++
			if db.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
				withStormshroud++
			}
		}
		if seen >= 60 && withStormshroud >= 5 {
			break
		}
	}
	if seen == 0 {
		t.Fatal("REFUSING: no corpus build carries a nonzero AvoidElementalAilments, so the " +
			"name has no producer and its four consumers are a dead branch")
	}
	if withStormshroud == 0 {
		t.Fatalf("REFUSING: %d builds carry AvoidElementalAilments and NONE also carries "+
			"ShockAvoidAppliesToElementalAilments, so CalcDefence.lua:1584's AvoidShock term "+
			"decides nothing and this fix is unmeasurable here", seen)
	}
	t.Logf("AvoidElementalAilments has a producer on %d builds, %d of them with Stormshroud",
		seen, withStormshroud)
}
