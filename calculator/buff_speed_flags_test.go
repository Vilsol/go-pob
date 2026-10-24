package calculator

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestBuffSpeedModsAreFlagged pins upstream 12953654f's rule for Adrenaline.
func TestBuffSpeedModsAreFlagged(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS!=true")
	}
	requireGameData(t, "3.29")
	const build = "0176-juggernaut-vigilant-strike-4.xml"
	// PoB's dumped modDB.Speed rows sourced from Adrenaline on this build.
	want := []string{"Attack INC 25", "Cast INC 25"}

	env := corpusEnvForLoadedData(t, build)
	if !env.ModDB.Flag(nil, "Adrenaline") {
		t.Fatalf("REFUSING: %s does not set Adrenaline, so it cannot see this defect — wrong fixture", build)
	}
	var got []string
	for _, m := range env.ModDB.Mods["Speed"] {
		if m.GetSource() != "Adrenaline" {
			continue
		}
		flag := "-"
		switch m.Flags() {
		case mod.MFlagAttack:
			flag = "Attack"
		case mod.MFlagCast:
			flag = "Cast"
		}
		got = append(got, fmt.Sprintf("%s %s %g", flag, m.Type(), m.Value().Float()))
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("%s: Adrenaline Speed mods %v, PoB %v. An unflagged buff mod matches every cfg, so it "+
			"leaks into passes the game never applies it to.", build, got, want)
	}
}
