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
