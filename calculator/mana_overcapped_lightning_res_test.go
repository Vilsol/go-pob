package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestManaIncreasedByOvercappedLightningResParses pins ModParser.lua:5872, the only member of the overcapped-resistance family that had no row here.
func TestManaIncreasedByOvercappedLightningResParses(t *testing.T) {
	e := ParseMod("Mana is increased by 50% of Overcapped Lightning Resistance", false)
	if e == nil {
		t.Fatal("ParseMod returned nil")
	}
	if x := strings.TrimSpace(modExtra(e)); x != "" {
		t.Fatalf("leftover Extra %q — processModLine discards the WHOLE line for this", x)
	}
	if len(e.ModList) != 2 {
		names := make([]string, len(e.ModList))
		for i, m := range e.ModList {
			names[i] = m.Name()
		}
		t.Fatalf("got %d mods %v, want 2 (the flag and the Mana INC)", len(e.ModList), names)
	}

	var sawFlag bool
	var manaMod mod.Mod
	for _, m := range e.ModList {
		switch m.Name() {
		case "ManaIncreasedByOvercappedLightningRes":
			sawFlag = true
		case "Mana":
			manaMod = m
		}
	}
	if !sawFlag {
		t.Error("no ManaIncreasedByOvercappedLightningRes flag: CalcPerform.lua:2040's disjunction stays dead")
	}
	if manaMod == nil {
		t.Fatal("no Mana mod")
	}
	if got := manaMod.Value().Float(); got != 0.5 {
		t.Errorf("Mana INC = %v, want 0.5 (num/100)", got)
	}
	var perStat *mod.PerStatTag
	for _, tag := range manaMod.Tags() {
		if p, ok := tag.(*mod.PerStatTag); ok {
			perStat = p
		}
	}
	if perStat == nil {
		t.Fatal("Mana mod carries no PerStat tag — an untagged INC 0.5 applies unconditionally, " +
			"which is over-application replacing under-application")
	}
	if len(perStat.StatList) != 1 || perStat.StatList[0] != "LightningResistOverCap" {
		t.Errorf("PerStat stats = %v, want [LightningResistOverCap]", perStat.StatList)
	}

	c := ParseMod("Critical Strike Chance is increased by Overcapped Lightning Resistance", false)
	if c == nil || len(c.ModList) != 1 || c.ModList[0].Name() != "CritChance" {
		t.Errorf("the crit sibling no longer yields CritChance alone: %+v", c)
	}
	l := ParseMod("Your Maximum Energy Shield is equal to 20% of your Maximum Life", false)
	if l == nil || len(l.ModList) != 1 || l.ModList[0].Name() != "EnergyShield" {
		t.Errorf("the maximum-life row above it no longer yields EnergyShield alone: %+v", l)
	}
}
