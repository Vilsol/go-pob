package calculator

import (
	"reflect"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestTreeStatMarkupParsesAsPinnedText: go-pob-data keeps the game's keyword markup.
func TestTreeStatMarkupParsesAsPinnedText(t *testing.T) {
	for id, c := range map[string]struct{ ours, pob string }{
		"16287": {"Triggers Level 20 [BoneFeeders|Summon Bone Feeders] when Allocated", "Triggers Level 20 Summon Bone Feeders when Allocated"},
		"5502":  {"Gain [SpiritInfusion|Spirit Infusion] every 0.5 seconds while Channelling a Spell", "Gain Spirit Infusion every 0.5 seconds while Channelling a Spell"},
		"1945":  {"Spell Critical Strike Chance [BifurcatedCrit|Bifurcates]", "Spell Critical Strike Chance Bifurcates"},
	} {
		want := buildModListForNode(masteryTestEnv(nil), id, data.Node{Stats: []string{c.pob}}).Mods()
		got := buildModListForNode(masteryTestEnv(nil), id, data.Node{Stats: []string{c.ours}}).Mods()
		if len(want) == 0 {
			t.Errorf("node %s: PoB's text %q parses to no mods; this case has no subject", id, c.pob)
			continue
		}
		if len(got) != len(want) {
			t.Errorf("node %s: %q gives %d mods, PoB's text gives %d", id, c.ours, len(got), len(want))
			continue
		}
		for i := range want {
			if !reflect.DeepEqual(got[i], want[i]) {
				t.Errorf("node %s mod %d: got %s %s %v, want %s %s %v", id, i,
					got[i].Name(), got[i].Type(), got[i].Value(), want[i].Name(), want[i].Type(), want[i].Value())
			}
		}
	}
}

// TestForbiddenGameWordingParses: the game words two Forbidden nodes differently from the pin (src/TreeData/3_29/tree.lua:89995, :90084).
func TestForbiddenGameWordingParses(t *testing.T) {
	for _, c := range []struct {
		ours, pob string
		mods      int
	}{
		{"25% of Damage Taken Recouped as Life, Mana and Energy Shield", "25% of Damage taken Recouped as Life, Mana and Energy Shield", 3},
		{"Non-Damaging Elemental Ailments you inflict spread to other enemies within 2 metres", "Non-Damaging Elemental Ailments you inflict spread to nearby enemies within 2 metres", 0},
	} {
		want := buildModListForNode(masteryTestEnv(nil), "1", data.Node{Stats: []string{c.pob}}).Mods()
		got := buildModListForNode(masteryTestEnv(nil), "1", data.Node{Stats: []string{c.ours}}).Mods()
		if len(want) != c.mods || !reflect.DeepEqual(got, want) {
			t.Errorf("%q: got %d mods, PoB's text gives %d, want %d", c.ours, len(got), len(want), c.mods)
		}
	}
}
