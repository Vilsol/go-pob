package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
)

// TestMinionLifeScalingTables pins CalcActiveSkill.lua:717.
func TestMinionLifeScalingTables(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		name  string
		table []float64
		first float64
		last  float64
	}{
		{"MonsterLifeTable2", data.MonsterLifeTable2, 10, 47841},
		{"MonsterLifeTable3", data.MonsterLifeTable3, 13, 50962},
	} {
		if len(tc.table) == 0 {
			t.Errorf("%s is empty; the emitter did not produce it", tc.name)
			continue
		}
		if tc.table[1] != tc.first {
			t.Errorf("%s[1] = %v, want %v (Misc.lua's first entry)", tc.name, tc.table[1], tc.first)
		}
		if got := tc.table[len(tc.table)-1]; got != tc.last {
			t.Errorf("%s[last] = %v, want %v", tc.name, got, tc.last)
		}
	}

	var altLife2 string
	for id, m := range raw2.Spectres {
		if m.LifeScaling == "AltLife2" {
			altLife2 = id
			break
		}
	}
	if altLife2 == "" {
		t.Fatal("REFUSING: no spectre carries LifeScaling AltLife2; the branch under test has no " +
			"population and this assertion would be vacuous")
	}
	if got := minionLifeTable("AltLife1", false); &got[0] != &data.MonsterLifeTable2[0] {
		t.Errorf("AltLife1 selected the wrong table")
	}
	if got := minionLifeTable("AltLife2", false); &got[0] != &data.MonsterLifeTable3[0] {
		t.Errorf("AltLife2 selected the wrong table")
	}
	if got := minionLifeTable("AltLife1", true); &got[0] != &data.MonsterLifeTable2[0] {
		t.Errorf("AltLife1 must win over isSpectre, as the Lua `or` chain orders it")
	}
	if got := minionLifeTable("", true); &got[0] != &data.MonsterLifeTable[0] {
		t.Errorf("isSpectre with no scaling must still select MonsterLifeTable")
	}
	if got := minionLifeTable("", false); &got[0] != &data.MonsterAllyLifeTable[0] {
		t.Errorf("no scaling and not a spectre must still select MonsterAllyLifeTable")
	}
}
