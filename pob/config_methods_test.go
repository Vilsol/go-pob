package pob

import "testing"

func strPtr(s string) *string { return &s }

func TestGetCustomModifierBlocksMigratesLegacyText(t *testing.T) {
	b := &PathOfBuilding{}
	b.SetConfigOption(Input{Name: "customMods", String: strPtr("+10 to maximum Life")})
	b.Config.CustomModifierBlocks = []CustomModifierBlock{{Text: ""}}

	got := b.GetCustomModifierBlocks()
	if len(got) != 1 || got[0].Text != "+10 to maximum Life" || got[0].BlockTitle() != "Default" || !got[0].IsEnabled() {
		t.Fatalf("legacy text not migrated into a Default block: %+v", got)
	}

	b.Config.CustomModifierBlocks = []CustomModifierBlock{{Title: strPtr("A"), Text: "+1 to Strength"}}
	if got := b.GetCustomModifierBlocks(); len(got) != 1 || got[0].Text != "+1 to Strength" {
		t.Fatalf("a block with text must win over the legacy input: %+v", got)
	}

	if got := (&PathOfBuilding{}).GetCustomModifierBlocks(); len(got) != 1 || got[0].Text != "" || got[0].BlockTitle() != "Default" {
		t.Fatalf("no blocks must read as one empty Default block: %+v", got)
	}
}

func TestSetCustomModifierBlocksDropsLegacyInput(t *testing.T) {
	b := &PathOfBuilding{}
	b.SetConfigOption(Input{Name: "customMods", String: strPtr("+10 to maximum Life")})
	b.SetCustomModifierBlocks([]CustomModifierBlock{{Title: strPtr("A"), Enabled: strPtr("false"), Text: "x"}})
	if b.GetStringOption("customMods") != "" {
		t.Error("legacy customMods input survived a block write")
	}
	if len(b.Config.CustomModifierBlocks) != 1 || b.Config.CustomModifierBlocks[0].IsEnabled() {
		t.Errorf("blocks not stored as given: %+v", b.Config.CustomModifierBlocks)
	}

	b.SetCustomModifierBlocks(nil)
	if got := b.Config.CustomModifierBlocks; len(got) != 1 || got[0].BlockTitle() != "Default" {
		t.Errorf("removing the last block must leave a Default one: %+v", got)
	}
}

func TestResetConfigOptionsKeepsCustomMods(t *testing.T) {
	b := &PathOfBuilding{}
	yes := true
	b.SetConfigOption(Input{Name: "conditionOnslaught", Boolean: &yes})
	b.SetConfigOption(Input{Name: "customMods", String: strPtr("+10 to maximum Life")})
	b.ResetConfigOptions()
	if len(b.Config.Inputs) != 1 || b.Config.Inputs[0].Name != "customMods" {
		t.Errorf("inputs after reset = %+v, want only customMods", b.Config.Inputs)
	}
}
