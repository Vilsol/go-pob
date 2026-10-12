package calculator

import (
	"slices"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestEscalationWarcryBuffDurationPair pins BOTH ends of Escalation.
func TestEscalationWarcryBuffDurationPair(t *testing.T) {
	mods, extra := parseMod("10% increased Melee Damage for each second you've been affected by a Warcry Buff, up to a maximum of 60%", 1)
	testza.AssertEqual(t, "", extra, "the line must parse completely")
	if len(mods) != 1 || len(mods[0].Tags()) != 1 {
		t.Fatalf("got %v, want one Damage INC with one Multiplier tag", mods)
	}
	testza.AssertEqual(t, "Damage", mods[0].Name())
	testza.AssertEqual(t, mod.TypeIncrease, mods[0].Type())
	testza.AssertEqual(t, 10.0, mods[0].Value().ValueFloat)
	m, ok := mods[0].Tags()[0].(*mod.MultiplierTag)
	if !ok || !slices.Equal(m.VariableList, []string{"AffectedByWarcryBuffDuration"}) || m.TagLimit == nil || *m.TagLimit != 60 || !m.TagLimitTotal {
		t.Errorf("tag = %#v, want Multiplier AffectedByWarcryBuffDuration limit=60 limitTotal", mods[0].Tags()[0])
	}

	control, extra := parseMod("10% increased Damage for each of your Mines Detonated Recently, up to 60%", 1)
	testza.AssertEqual(t, "", extra, "the neighbour line must still parse completely")
	if len(control) != 1 || len(control[0].Tags()) != 1 {
		t.Fatalf("neighbour produced %v", control)
	}
	if c, ok := control[0].Tags()[0].(*mod.MultiplierTag); !ok || !slices.Equal(c.VariableList, []string{"MineDetonatedRecently"}) {
		t.Errorf("neighbour tag = %#v, want Multiplier MineDetonatedRecently", control[0].Tags()[0])
	}

	if !slices.Contains(configOrder, "multiplierAffectedByWarcryBuffDuration") {
		t.Fatal("multiplierAffectedByWarcryBuffDuration missing from configOrder: the handler would never run")
	}
	apply, ok := configurations["multiplierAffectedByWarcryBuffDuration"]
	if !ok {
		t.Fatal("no handler in configurations")
	}
	list := moddb.NewModList()
	apply(8.0, list, moddb.NewModList())
	var found bool
	for _, x := range list.Mods() {
		found = found || (x.Name() == "Multiplier:AffectedByWarcryBuffDuration" && x.Type() == mod.TypeBase && x.Value().ValueFloat == 8)
	}
	if !found {
		t.Errorf("handler added %v, want Multiplier:AffectedByWarcryBuffDuration BASE 8", list.Mods())
	}
}

// TestRiteOfRuinDegenPerRageEffect pins ModParser.lua:2564 (ModCache.lua:9824): the degen scales with RageEffect, not the raw Rage stack.
func TestRiteOfRuinDegenPerRageEffect(t *testing.T) {
	mods, extra := parseMod("Lose 0.1% of Life per second per Rage while you are not losing Rage", 1)
	testza.AssertEqual(t, "", extra, "the line must parse completely")
	if len(mods) != 1 || len(mods[0].Tags()) != 2 {
		t.Fatalf("got %v, want one LifeDegen with PercentStat + Multiplier", mods)
	}
	m, ok := mods[0].Tags()[1].(*mod.MultiplierTag)
	if !ok || !slices.Equal(m.VariableList, []string{"RageEffect"}) {
		t.Errorf("tag 2 = %#v, want Multiplier RageEffect", mods[0].Tags()[1])
	}
}
