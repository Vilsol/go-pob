package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestEverlastingSacrificeRowsParse pins ModParser.lua:2429-2431 and :3249-3251, the producers of Condition:EverlastingSacrifice.
func TestEverlastingSacrificeRowsParse(t *testing.T) {
	requireGameData(t, "3.29")

	e := ParseMod("Everlasting Sacrifice", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "Condition:EverlastingSacrifice" || len(e.ModList[0].Tags()) != 0 {
		t.Errorf("bare row: got %+v", e)
	}

	e = ParseMod("You have Everlasting Sacrifice if at least 6 Corrupted Items are Equipped", false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "Condition:EverlastingSacrifice" || len(e.ModList[0].Tags()) != 1 {
		t.Fatalf("threshold row: got %+v", e)
	}
	mt, ok := e.ModList[0].Tags()[0].(*mod.MultiplierThresholdTag)
	if !ok || mt.Variable != "CorruptedItem" || mt.TagThreshold == nil || *mt.TagThreshold != 6 {
		t.Errorf("threshold row: tag %#v, want MultiplierThreshold CorruptedItem >= 6", e.ModList[0].Tags()[0])
	}
}
