package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestMinionLifeAppliesToPlayer pins ModParser.lua:3408 against its consumer at CalcPerform.lua:1531-1541 — Rigwald's Hunt.
func TestMinionLifeAppliesToPlayer(t *testing.T) {
	requireGameData(t, "3.29")

	t.Run("the row parses whole", func(t *testing.T) {
		const line = "Increases and Reductions to Minion Maximum Life also apply to you at 15% of their value"
		e := ParseMod(line, false)
		if e.Extra != "" {
			t.Fatalf("unparsed remainder %q -- the WHOLE line is discarded", e.Extra)
		}
		want := map[string]struct {
			typ mod.Type
			val float64
		}{
			"MinionLifeAppliesToPlayer":         {mod.TypeFlag, 0},
			"ImprovedMinionLifeAppliesToPlayer": {mod.TypeMAX, 15},
		}
		if len(e.ModList) != len(want) {
			t.Fatalf("got %d mods, want %d", len(e.ModList), len(want))
		}
		for _, m := range e.ModList {
			w, ok := want[m.Name()]
			if !ok {
				t.Fatalf("unexpected mod %s", m.Name())
			}
			if m.Type() != w.typ {
				t.Errorf("%s is %s, want %s", m.Name(), m.Type(), w.typ)
			}
			if w.typ != mod.TypeFlag && m.Value().Float() != w.val {
				t.Errorf("%s = %v, want %v", m.Name(), m.Value().Float(), w.val)
			}
		}
	})

	if os.Getenv("CORPUS") != "true" {
		return
	}

	for _, tc := range []struct {
		build string
		want  float64
		role  string
	}{
		{"14525-ladder-razowiec2_6009.tardivediskinesia", 230, "six converted nodes, each floored"},
		{"15175-ladder-xxlegendxx_5325.PvA_fortifyRelic", 212, "a converted node above the 20% step"},
		{"11745-ladder-Roige_4445.DaddyDommy", 154, "80 MinionModifier rows, only the Life INC ones convert"},
		{"10145-ladder-Grayeye_1799.Sichuanchelyidae", 128, "no minion Life nodes on the tree: the block must add nothing"},
		{"7449-ladder-jirkamcz_7919.Jirkamcz_Allflame_Dom", 169, "the fifth and last carrier in the corpus"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), tc.build+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			build, err := builds.ParseBuild(xml)
			if err != nil {
				t.Fatal(err)
			}
			env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)
			got := float64(0)
			for _, m := range env.ModDB.Mods["Life"] {
				if m.Type() == mod.TypeIncrease {
					got += m.Value().Float()
				}
			}
			if got != tc.want {
				t.Errorf("summed INC Life = %v, want %v (%s)", got, tc.want, tc.role)
			}
		})
	}
}
