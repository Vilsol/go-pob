package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestExplodeModParse pins ModParser.lua:2106-2185.
func TestExplodeModParse(t *testing.T) {
	cases := []struct {
		line string
		want ExplodeMod
		tag  string
	}{
		{"Enemies you Kill during Effect have a 24% chance to Explode, dealing a tenth of their maximum Life as Damage of a Random Element",
			ExplodeMod{Type: "RandomElement", Chance: 0.24, Amount: 10, KeyOfScaledMod: "chance"}, "Condition"},
		{"Enemies you or your Totems Kill have 10% chance to Explode, dealing 250% of their maximum Life as Fire Damage",
			ExplodeMod{Type: "Fire", Chance: 0.1, Amount: 250, KeyOfScaledMod: "chance"}, ""},
		{"Cursed Enemies you or your Minions Kill have a 50% chance to Explode, dealing a quarter of their maximum Life as Chaos Damage",
			ExplodeMod{Type: "Chaos", Chance: 0.5, Amount: 25, KeyOfScaledMod: "chance"}, "ActorCondition"},
		{"Burning Enemies you Kill have a 5% chance to Explode, dealing a tenth of their maximum Life as Fire Damage",
			ExplodeMod{Type: "Tenth", Chance: 1, Amount: 5, KeyOfScaledMod: "chance"}, "ActorCondition"},
	}
	for _, c := range cases {
		mods, extra := parseMod(c.line, 1)
		if extra != "" || len(mods) != 2 {
			t.Errorf("%q: extra %q, %d mods, want 2", c.line, extra, len(mods))
			continue
		}
		if mods[0].Name() != "ExplodeMod" || mods[1].Name() != "CanExplode" || mods[1].Type() != mod.TypeFlag {
			t.Errorf("%q: got %s/%s", c.line, mods[0].Name(), mods[1].Name())
			continue
		}
		got, ok := mods[0].Value().ValueList.(ExplodeMod)
		if !ok || got != c.want {
			t.Errorf("%q: payload %#v, want %#v", c.line, mods[0].Value().ValueList, c.want)
		}
		switch c.tag {
		case "":
			if len(mods[0].Tags()) != 0 {
				t.Errorf("%q: %d tags, want none", c.line, len(mods[0].Tags()))
			}
		case "Condition":
			if len(mods[0].Tags()) != 1 {
				t.Errorf("%q: %d tags, want 1", c.line, len(mods[0].Tags()))
			} else if _, ok := mods[0].Tags()[0].(*mod.ConditionTag); !ok {
				t.Errorf("%q: tag %#v, want Condition", c.line, mods[0].Tags()[0])
			}
		case "ActorCondition":
			if len(mods[0].Tags()) != 1 {
				t.Errorf("%q: %d tags, want 1", c.line, len(mods[0].Tags()))
			} else if _, ok := mods[0].Tags()[0].(*mod.ActorConditionTag); !ok {
				t.Errorf("%q: tag %#v, want ActorCondition", c.line, mods[0].Tags()[0])
			}
		}
	}

	if mods, _ := parseMod("Enemies you Kill have a 10% chance to Explode, dealing a third of their maximum Life as Fire Damage", 1); len(mods) != 0 {
		t.Errorf("unreadable amount produced %d mods, want none", len(mods))
	}
}
