package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestSoulEaterSpeedRows pins CalcPerform.lua:935-939.
func TestSoulEaterSpeedRows(t *testing.T) {
	requireGameData(t, "3.29")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("needs the corpus")
	}

	const build = "2650-ladder-drayvol721_4210.DavyJonesSlut"
	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	pob, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	env := (&Calculator{PoB: pob}).BuildOutput(OutputModeMain)

	if !env.ModDB.Flag(nil, "Condition:CanHaveSoulEater") {
		t.Fatal("REFUSING: " + build + " no longer sets Condition:CanHaveSoulEater")
	}

	want := map[mod.MFlag]bool{mod.MFlagAttack: false, mod.MFlagCast: false}
	for _, m := range env.ModDB.Mods["Speed"] {
		if m.Type() != mod.TypeIncrease || m.Value().Float() != 5 {
			continue
		}
		for _, tag := range m.Tags() {
			mt, ok := tag.(*mod.MultiplierTag)
			if !ok || len(mt.VariableList) != 1 || mt.VariableList[0] != "SoulEaterStack" {
				continue
			}
			if mt.TagLimit == nil {
				t.Errorf("SoulEaterStack multiplier has no limit, want PoB's 54")
			} else if *mt.TagLimit != 54 {
				t.Errorf("SoulEaterStack multiplier limit = %v, want PoB's 54", *mt.TagLimit)
			}
			if _, tracked := want[m.Flags()]; !tracked {
				t.Errorf("unexpected mod flag %v on a SoulEaterStack Speed row", m.Flags())
				continue
			}
			want[m.Flags()] = true
		}
	}
	for flag, seen := range want {
		if !seen {
			t.Errorf("no Speed INC 5 row tagged Multiplier:SoulEaterStack for flag %v", flag)
		}
	}
}
