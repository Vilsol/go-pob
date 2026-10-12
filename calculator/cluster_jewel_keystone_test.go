package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestClusterJewelKeystonesParse covers ModParser.lua:5881-5883.
func TestClusterJewelKeystonesParse(t *testing.T) {
	if len(data.ClusterJewels.Keystones) == 0 {
		t.Fatal("REFUSING: data.ClusterJewels.Keystones is empty, so this test asserts nothing")
	}

	for _, name := range data.ClusterJewels.Keystones {
		t.Run(name, func(t *testing.T) {
			e := ParseMod(name, false)
			if e == nil || len(e.ModList) == 0 {
				t.Fatalf("ParseMod(%q) produced no mods", name)
			}
			if e.Extra != "" {
				t.Errorf("ParseMod(%q) left Extra %q", name, e.Extra)
			}
			if len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %d mods, want exactly 1 -- the cluster-jewel loop "+
					"sets a Keystone and nothing else", name, len(e.ModList))
			}
			if got := e.ModList[0].Name(); got != "Keystone" {
				t.Errorf("ParseMod(%q) produced %q, want Keystone", name, got)
			}
			if got := e.ModList[0].Value().ValueList; got != name {
				t.Errorf("ParseMod(%q) Keystone value = %v, want %q", name, got, name)
			}
		})
	}
}

// TestOrdinaryKeystonesStillCarryTheHaveFlag is the two-sided control.
func TestOrdinaryKeystonesStillCarryTheHaveFlag(t *testing.T) {
	const name = "Resolute Technique"

	e := ParseMod(name, false)
	if e == nil || len(e.ModList) != 2 {
		t.Fatalf("ParseMod(%q) produced %v, want a Keystone AND a Condition:Have flag", name, e)
	}
	if got := e.ModList[1].Name(); got != "Condition:HaveResoluteTechnique" {
		t.Errorf("ParseMod(%q) second mod = %q, want Condition:HaveResoluteTechnique", name, got)
	}
}
