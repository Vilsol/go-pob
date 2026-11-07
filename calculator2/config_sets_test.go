package calculator2

import (
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

func TestConfigSets(t *testing.T) {
	n := 3.0
	build := &pob2.PathOfBuilding2{}
	build.Config.Inputs = []pob2.Input{{Name: "enemyLevel", Number: &n}}

	// Load reads a pre-ConfigSet build as set 1 (ConfigTab.lua:1409-1412)
	sets, active := ConfigSetList(build)
	if len(sets) != 1 || sets[0] != (ConfigSetView{ID: 1, Title: "Default"}) || active != 1 || len(build.Config.ConfigSets[0].Inputs) != 1 || build.Config.Inputs != nil {
		t.Fatalf("legacy layout: %+v active %d, %+v", sets, active, build.Config)
	}

	// ConfigSetService:NewConfigSet and CopyConfigSet make the new set active
	if id := NewConfigSet(build, "Boss"); id != 2 || activeConfigSetID(build) != 2 || len(*activeConfigInputs(build)) != 0 {
		t.Fatalf("new set %d active %d", id, activeConfigSetID(build))
	}
	if blocks := customModifierBlocks(build); len(blocks) != 1 || *blocks[0].Title != "Default" || !*blocks[0].Enabled || blocks[0].Text != "" {
		t.Fatalf("new set's custom modifier blocks: %+v", blocks)
	}
	if id := CopyConfigSet(build, 1, "Mapping"); id != 3 || activeConfigSetID(build) != 3 || len(*activeConfigInputs(build)) != 1 {
		t.Fatalf("copied set %d active %d", id, activeConfigSetID(build))
	}
	RenameConfigSet(build, 1, "Base")
	MoveConfigSet(build, 2, 0)
	sets, _ = ConfigSetList(build)
	if want := []ConfigSetView{{3, "Mapping"}, {1, "Base"}, {2, "Boss"}}; !equalSets(sets, want) {
		t.Fatalf("after rename and move: %+v", sets)
	}

	// deleting the active set activates the one before it in the order (ConfigSetService.lua:37-46)
	SetActiveConfigSet(build, 2)
	DeleteConfigSet(build, 2)
	if sets, active = ConfigSetList(build); active != 1 || len(sets) != 2 {
		t.Fatalf("after deleting the active set: %+v active %d", sets, active)
	}
	// the free id is reused
	if id := NewConfigSet(build, "Again"); id != 2 {
		t.Fatalf("reused id %d", id)
	}
	// the last set cannot be deleted
	DeleteConfigSet(build, 3)
	DeleteConfigSet(build, 2)
	DeleteConfigSet(build, 1)
	if sets, _ = ConfigSetList(build); len(sets) != 1 || sets[0].ID != 1 {
		t.Fatalf("after deleting every set: %+v", sets)
	}
}

func equalSets(a, b []ConfigSetView) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
