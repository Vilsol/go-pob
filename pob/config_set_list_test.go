package pob

import (
	"testing"
)

func configSetTitles(c Config) []string {
	out := make([]string, 0, len(c.ConfigSets))
	for _, s := range c.ConfigSets {
		t := "Default"
		if s.Title != nil {
			t = *s.Title
		}
		out = append(out, t)
	}
	return out
}

// TestConfigSetManageList: ConfigSetListControl.lua:13-110 / ConfigTab.lua:1224-1244.
func TestConfigSetManageList(t *testing.T) {
	b := &PathOfBuilding{}
	level := 80.0
	b.SetConfigOption(Input{Name: "enemyLevel", Number: &level})
	if id := b.NewConfigSet("Boss"); id != 2 {
		t.Fatalf("new id %d", id)
	}
	if n := len(b.Config.ConfigSets[1].Inputs); n != 0 {
		t.Fatalf("a new set carries %d inputs", n)
	}
	copied := b.CopyConfigSet(1, "Boss copy")
	if copied != 3 || len(b.Config.ConfigSets[2].Inputs) != 1 || b.Config.ConfigSets[2].Inputs[0].Name != "enemyLevel" {
		t.Fatalf("copy %d: %+v", copied, b.Config.ConfigSets[2])
	}
	b.RenameConfigSet(2, "Mapping")
	if got := configSetTitles(b.Config); got[1] != "Mapping" {
		t.Fatalf("titles %v", got)
	}

	b.SetActiveConfigSet(3)
	b.DeleteConfigSet(3)
	if b.Config.ActiveConfigSet != 2 || len(b.Config.ConfigSets) != 2 {
		t.Fatalf("after deleting the active set: active %d, %v", b.Config.ActiveConfigSet, configSetTitles(b.Config))
	}
	b.MoveConfigSet(1, 0)
	if got := configSetTitles(b.Config); got[0] != "Mapping" || got[1] != "Default" {
		t.Fatalf("after the move: %v", got)
	}
	b.DeleteConfigSet(2)
	b.DeleteConfigSet(1)
	if len(b.Config.ConfigSets) != 1 || b.Config.ActiveConfigSet != 1 {
		t.Fatalf("the last set went or the active one is wrong: %v, active %d", configSetTitles(b.Config), b.Config.ActiveConfigSet)
	}
	if b.GetNumberOption("enemyLevel") != 80 {
		t.Fatalf("enemyLevel %v", b.GetNumberOption("enemyLevel"))
	}
}
