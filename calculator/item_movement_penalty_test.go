package calculator

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestArmourMovementPenaltySign locks the sign of ItemBaseArmour.MovementPenalty against PoB's own item base dump.
func TestArmourMovementPenaltySign(t *testing.T) {
	var expected []expectedBase
	raw, err := os.ReadFile(itemBasesExpected)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &expected))

	checked := 0
	for _, want := range expected {
		penalty, ok := want.Armour["MovementPenalty"]
		if !ok || penalty == 0 {
			continue
		}
		checked++
		got := ItemBaseByName(want.Name)
		testza.AssertNotNil(t, got, "no item base for %q", want.Name)
		if got == nil || got.Armour == nil {
			continue
		}
		testza.AssertEqual(t, int(penalty), got.Armour.MovementPenalty, want.Name+" MovementPenalty")
	}
	testza.AssertGreater(t, checked, 5, "oracle carried no MovementPenalty bases")
}

// TestArmourMovementPenaltyMod is the end-to-end half: the mod an armour base contributes must REDUCE movement speed.
func TestArmourMovementPenaltyMod(t *testing.T) {
	item := ParseItem("Rarity: NORMAL\nAstral Plate\nAstral Plate\n")
	testza.AssertNotNil(t, item)
	if item == nil {
		return
	}

	var found []float64
	for _, m := range item.Mods {
		if m.Name() == "MovementSpeed" && m.Type() == mod.TypeIncrease {
			found = append(found, m.Value().Float())
		}
	}
	testza.AssertEqual(t, 1, len(found), "expected exactly one base MovementSpeed mod")
	if len(found) == 1 {
		testza.AssertEqual(t, float64(-5), found[0], "Astral Plate movement penalty")
	}
}
