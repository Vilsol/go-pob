package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestMultiThresholdKeepsSeparateAccumulators pins ModParser.lua:6522-6531.
func TestMultiThresholdKeepsSeparateAccumulators(t *testing.T) {
	build := func() radiusJewelFunc {
		return multiThreshold([]radiusJewelFunc{
			getThreshold([]string{"Int"}, MOD("A", mod.TypeBase, 1)),
			getThreshold([]string{"Int"}, MOD("B", mod.TypeBase, 1)),
		})
	}

	run := func(t *testing.T, intPerNode float64, nodes int) []string {
		t.Helper()
		fn := build()
		d := newRadiusJewelData("Tree:1")
		out := moddb.NewModList()
		for i := 0; i < nodes; i++ {
			nodeList := moddb.NewModList()
			nodeList.AddMod(mod.NewFloat("Int", mod.TypeBase, intPerNode))
			work := moddb.NewModList()
			work.AddMod(mod.NewFloat("Int", mod.TypeBase, intPerNode))
			fn(&radiusJewelNode{id: "n", modList: nodeList}, work, d)
		}
		fn(nil, out, d)

		found := out.Tabulate("", nil, "A", "B")
		names := make([]string, 0, len(found))
		for _, m := range found {
			names = append(names, m.Mod.Name())
		}
		return names
	}

	if got := run(t, 40, 1); len(got) != 2 {
		t.Errorf("at 40 Int got %v, want both A and B", got)
	}
	if got := run(t, 20, 1); len(got) != 0 {
		t.Errorf("at 20 Int got %v, want neither -- the two funcs are sharing one accumulator", got)
	}
}

// TestJewelThresholdRowsMatchThePin is the name-set half.
func TestJewelThresholdRowsMatchThePin(t *testing.T) {
	for _, line := range []string{
		"With at least 40 Intelligence in Radius, Raised Spectres have a 50% chance to gain Soul Eater for 20 seconds on Kill",
		"With at least 40 Strength in Radius, Cleave has +0.1 metres to Radius per Nearby Enemy, up to a maximum of +1 metre",
		"With at least 40 Intelligence in Radius, 12% of Damage taken Recouped as Mana if you've Warcried Recently",
		"With at least 40 Intelligence in Radius, Fireball Projectiles gain Radius as they travel farther, up to +6 Radius",
		"With at least 40 Intelligence in Radius, Projectiles gain radius as they travel farther, up to a maximum of +0.6 metres to radius",
	} {
		t.Run(line[:40], func(t *testing.T) {
			entry := ParseMod(line, false)
			if strings.Trim(entry.Extra, " ") != "" {
				t.Fatalf("unparsed remainder %q -- the whole line is discarded", entry.Extra)
			}
			if len(entry.ModList) != 1 || entry.ModList[0].Name() != "JewelFunc" {
				t.Fatalf("got %v, want one JewelFunc mod", entry.ModList)
			}
		})
	}
}
