package calculator2

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// TestDotEffMultBreakdown: Flame Wall's burning ground against a resisting enemy shows its effective DPS
// modifier as PoB-PoE2 does (CalcOffence.lua:6102-6105; the lines are oracle/poe2_breakdowns.lua's for the fixture).
func TestDotEffMultBreakdown(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join("testdata", "dot_eff_mult.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Enemy resistance: 40%", "Effective DPS modifier:", "x 0.60 ^8(resistance)", "x 1.10 ^8(increased/reduced damage taken)", "= 0.660"}
	e := env.Player.Breakdown.Entries["FireDotEffMult"]
	if e == nil {
		t.Fatal("no FireDotEffMult breakdown")
	}
	if !slices.Equal(e.Lines, want) {
		t.Errorf("FireDotEffMult\n got  %q\n want %q", e.Lines, want)
	}
}

// TestDropsBurningGroundSource: burning ground dropped while moving is not ignite's, so PoB-PoE2 writes
// BurningGroundFromIgnite = false beside BurningGroundDPS 66 (CalcOffence.lua:6032-6035).
func TestDropsBurningGroundSource(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join("testdata", "dot_eff_mult.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	block := &build.Config.ConfigSets[0].CustomModifierBlocks[0]
	block.Text += "\nDrops Burning Ground while moving, dealing 100 Fire Damage per second for 3 seconds"
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	if err != nil {
		t.Fatal(err)
	}
	if got := env.Player.Output["BurningGroundDPS"]; got != 66 {
		t.Errorf("BurningGroundDPS %v, PoB-PoE2 66", got)
	}
	if v, ok := env.Player.OutputFlags["BurningGroundFromIgnite"]; !ok || v {
		t.Errorf("BurningGroundFromIgnite %v (set %v), PoB-PoE2 false (set)", v, ok)
	}
	if _, ok := env.Player.OutputFlags["CausticGroundFromPoison"]; ok {
		t.Error("CausticGroundFromPoison set, PoB-PoE2 nil")
	}
}
