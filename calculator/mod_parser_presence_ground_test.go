package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestPresenceGroundRows pins ModParser.lua:3740-3747: the Eater/Exarch "while a unique enemy / pinnacle atlas boss is in your presence, drops <X> ground" rows.
func TestPresenceGroundRows(t *testing.T) {
	for _, c := range []struct{ line, name, ground, presence string }{
		{"While a Unique Enemy is in your Presence, Drops Brittle Ground while moving, lasting 5 seconds", "BrittleBase", "OnBrittleGround", "RareOrUnique"},
		{"While a Unique Enemy is in your Presence, Drops Shocked Ground while moving, lasting 5 seconds", "ShockBase", "OnShockedGround", "RareOrUnique"},
		{"While a Unique Enemy is in your Presence, Drops Scorched Ground while moving, lasting 5 seconds", "ScorchBase", "OnScorchedGround", "RareOrUnique"},
		{"While a Unique Enemy is in your Presence, Drops Sapped Ground while moving, lasting 5 seconds", "SapBase", "OnSappedGround", "RareOrUnique"},
		{"While a Pinnacle Atlas Boss is in your Presence, Drops Brittle Ground while moving, lasting 5 seconds", "BrittleBase", "OnBrittleGround", "PinnacleBoss"},
		{"While a Pinnacle Atlas Boss is in your Presence, Drops Shocked Ground while moving, lasting 5 seconds", "ShockBase", "OnShockedGround", "PinnacleBoss"},
		{"While a Pinnacle Atlas Boss is in your Presence, Drops Scorched Ground while moving, lasting 5 seconds", "ScorchBase", "OnScorchedGround", "PinnacleBoss"},
		{"While a Pinnacle Atlas Boss is in your Presence, Drops Sapped Ground while moving, lasting 5 seconds", "SapBase", "OnSappedGround", "PinnacleBoss"},
	} {
		entry := ParseMod(c.line, false)
		if strings.TrimSpace(entry.Extra) != "" || len(entry.ModList) != 1 {
			t.Errorf("%q: extra %q, %d mods", c.line, entry.Extra, len(entry.ModList))
			continue
		}
		m := entry.ModList[0]
		if m.Name() != c.name || m.Type() != mod.TypeBase {
			t.Errorf("%q: got %s %s", c.line, m.Name(), m.Type())
		}
		var vars []string
		for _, tag := range m.Tags() {
			ac, ok := tag.(*mod.ActorConditionTag)
			if !ok || ac.Actor == nil || *ac.Actor != "enemy" {
				t.Errorf("%q: unexpected tag %#v", c.line, tag)
				continue
			}
			vars = append(vars, ac.VariableList...)
		}
		if strings.Join(vars, ",") != c.ground+","+c.presence {
			t.Errorf("%q: actor conditions %v, want [%s %s]", c.line, vars, c.ground, c.presence)
		}
	}
}
