package calculator

import (
	"testing"
)

// TestOptimiseSockets: SkillsTab.lua:242-288: 0005's body armour holds Vaal Double Strike with Void Manipulation.
func TestOptimiseSockets(t *testing.T) {
	build := loadCorpusBuild(t, "0005-occultist-vaal-double-strike.xml")
	id := setSlotItemID(build.Items.ItemSets[activeItemSetIndex(build.Items)], "Body Armour")
	text, ok := buildItemText(build, id)
	if !ok {
		t.Fatal("no body armour")
	}
	scrambled, err := EditItem(text, ItemEdit{Kind: "sockets", Text: "R R R R R R"})
	if err != nil || !build.SetItemText(id, scrambled) {
		t.Fatalf("scramble: %v", err)
	}
	if !OptimiseSockets(build, "Body Armour") {
		t.Fatal("OptimiseSockets refused the body armour")
	}
	text, _ = buildItemText(build, id)
	if got := socketsText(ParseItem(text).Sockets); got != "G-G-G-B-R-B" {
		t.Errorf("sockets %q, want G-G-G-B-R-B", got)
	}
	if OptimiseSockets(build, "Flask 1") {
		t.Error("a flask slot has no socket limit")
	}
}

// TestOptimiseSocketsUseful: SkillsTab.lua:294-313 (#10165): the button is enabled while a gem it would socket is not in its colour.
func TestOptimiseSocketsUseful(t *testing.T) {
	build := loadCorpusBuild(t, "0005-occultist-vaal-double-strike.xml")
	id := setSlotItemID(build.Items.ItemSets[activeItemSetIndex(build.Items)], "Body Armour")
	text, ok := buildItemText(build, id)
	if !ok {
		t.Fatal("no body armour")
	}
	scrambled, err := EditItem(text, ItemEdit{Kind: "sockets", Text: "R R R R R R"})
	if err != nil || !build.SetItemText(id, scrambled) {
		t.Fatalf("scramble: %v", err)
	}
	if !OptimiseSocketsUseful(build, "Body Armour") {
		t.Error("scrambled sockets: Optimise Sockets is not offered")
	}
	if !OptimiseSockets(build, "Body Armour") {
		t.Fatal("OptimiseSockets refused the body armour")
	}
	if OptimiseSocketsUseful(build, "Body Armour") {
		t.Error("optimised sockets: Optimise Sockets is still offered")
	}
	if OptimiseSocketsUseful(build, "Flask 1") {
		t.Error("a flask slot is offered")
	}
}
