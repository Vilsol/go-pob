package calculator

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/moddb"
)

// TestEverlastingSacrificeConfigDispatches pins ConfigOptions.lua:1676-1679.
func TestEverlastingSacrificeConfigDispatches(t *testing.T) {
	if !slices.Contains(configOrder, "EverlastingSacrifice") {
		t.Fatal("EverlastingSacrifice missing from configOrder: the handler would never run")
	}
	apply, ok := configurations["EverlastingSacrifice"]
	if !ok {
		t.Fatal("EverlastingSacrifice has no handler in configurations")
	}

	on := moddb.NewModList()
	apply(true, on, moddb.NewModList())
	on.Conditions["Combat"] = true
	if got := on.Sum("BASE", nil, "ElementalResistMax"); got != 0 {
		t.Errorf("without Condition:EverlastingSacrifice ElementalResistMax = %v, want 0", got)
	}
	on.Conditions["EverlastingSacrifice"] = true
	if got := on.Sum("BASE", nil, "ElementalResistMax"); got != 5 {
		t.Errorf("ElementalResistMax = %v, want 5", got)
	}
	if got := on.Sum("BASE", nil, "ChaosResistMax"); got != 5 {
		t.Errorf("ChaosResistMax = %v, want 5", got)
	}

	off := moddb.NewModList()
	apply(false, off, moddb.NewModList())
	off.Conditions["Combat"] = true
	off.Conditions["EverlastingSacrifice"] = true
	if got := off.Sum("BASE", nil, "ElementalResistMax"); got != 0 {
		t.Errorf("unchecked must add nothing, ElementalResistMax = %v", got)
	}
}

// TestLinkedRecentlyConfigDispatches pins ConfigOptions.lua:1499-1501.
func TestLinkedRecentlyConfigDispatches(t *testing.T) {
	if !slices.Contains(configOrder, "conditionLinkedRecently") {
		t.Fatal("conditionLinkedRecently missing from configOrder: the handler would never run")
	}
	apply, ok := configurations["conditionLinkedRecently"]
	if !ok {
		t.Fatal("conditionLinkedRecently has no handler in configurations")
	}
	on := moddb.NewModList()
	apply(true, on, moddb.NewModList())
	if !on.Flag(nil, "Condition:LinkedRecently") {
		t.Error("checked must add Condition:LinkedRecently, ungated")
	}
	off := moddb.NewModList()
	apply(false, off, moddb.NewModList())
	if off.Flag(nil, "Condition:LinkedRecently") {
		t.Error("unchecked must add nothing")
	}
}
