package calculator

import "testing"

// TestTotemLifeMultMatchesTheGeneratedLuaTable is the assertion calc_offence.go:483 has claimed for this rebuild all along.
func TestTotemLifeMultMatchesTheGeneratedLuaTable(t *testing.T) {
	requireGameData(t, "3.29")

	want := map[int]float64{
		1: 1, 2: 1, 3: 1, 4: 1.2, 5: 1, 6: 1.2, 7: 1.2, 8: 1.2, 9: 1, 10: 1,
		11: 1, 12: 1, 13: 1.2, 15: 1.2, 16: 7.44, 17: 1.2, 18: 1, 19: 1, 20: 1.2, 21: 1.2,
	}

	for id := 0; id <= 22; id++ {
		got, ok := totemLifeMult(id)
		exp, expOK := want[id]
		if ok != expOK {
			t.Errorf("totemLifeMult(%d): ok = %v, want %v (Misc.lua:214 %s this key)",
				id, ok, expOK, map[bool]string{true: "has", false: "does not have"}[expOK])
			continue
		}
		if ok && got != exp {
			t.Errorf("totemLifeMult(%d) = %v, want %v", id, got, exp)
		}
	}

	if v, ok := totemLifeMult(16); !ok || v != 7.44 {
		t.Fatalf("control: totemLifeMult(16) = %v, %v — want 7.44, true; the .dat tables did not load", v, ok)
	}
}
