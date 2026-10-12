package calculator

import (
	"strings"
	"testing"
)

// TestBodyArmourESAndArmourHasBothHalves pins ModParser.lua:2478 together with its consumer, CalcDefence.lua:900 and :908.
func TestBodyArmourESAndArmourHasBothHalves(t *testing.T) {
	requireGameData(t, "3.29")

	const line = "100% increased Armour and Energy Shield from Equipped Body Armour if Equipped Helmet, Gloves and Boots all have Armour and Energy Shield"
	e := ParseMod(line, false)
	if e == nil {
		t.Fatalf("no entry: %s", line)
	}
	if extra := strings.TrimSpace(modExtra(e)); extra != "" {
		t.Fatalf("leftover %q (line is DROPPED WHOLE): %s", extra, line)
	}
	got := make([]string, 0, len(e.ModList))
	for _, m := range e.ModList {
		got = append(got, m.Name())
	}
	if !contains(got, "Body ArmourESAndArmour") {
		t.Fatalf("produced %v, want a %q mod — the name the consumer reads as slot+\"ESAndArmour\"",
			got, "Body ArmourESAndArmour")
	}
}
