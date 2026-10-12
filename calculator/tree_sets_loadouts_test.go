package calculator

import (
	"slices"
	"testing"
)

// TestMoveSpecReordersLoadouts: Reordering trees (PassiveSpecListControl.lua:82-87.
func TestMoveSpecReordersLoadouts(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	NewLoadout(build, "Mapping")
	NewLoadout(build, "Bossing")
	before := Loadouts(build).Loadouts
	i, j := slices.Index(before, "Mapping"), slices.Index(before, "Bossing")
	if i < 0 || j < 0 || i > j {
		t.Fatalf("REFUSING: loadouts %v", before)
	}
	if !SelectLoadout(build, "Mapping") {
		t.Fatal("not selected")
	}
	var mapping, bossing int
	for k, s := range build.Tree.Specs {
		switch orDefault(s.Title) {
		case "Mapping":
			mapping = k
		case "Bossing":
			bossing = k
		}
	}
	build.MoveSpec(bossing+1, mapping+1)
	after := Loadouts(build)
	if slices.Index(after.Loadouts, "Bossing") > slices.Index(after.Loadouts, "Mapping") {
		t.Errorf("after the move: %v", after.Loadouts)
	}
	if after.Selected < 0 || after.Loadouts[after.Selected] != "Mapping" {
		t.Errorf("selected %d of %v", after.Selected, after.Loadouts)
	}
}
