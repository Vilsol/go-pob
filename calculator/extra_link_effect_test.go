package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestExtraLinkEffectRowsParse: Fixture text is quoted from the pin's own data (Data/ModCache.lua:8313, :10435, :10653), never from our parser table.
func TestExtraLinkEffectRowsParse(t *testing.T) {
	cases := []struct {
		line         string
		inner        string
		globalEffect bool
	}{
		{"Curses are inflicted on you instead of Linked targets", "CurseImmune", false},
		{"Non-Unique Utility Flasks you Use apply to Linked Targets", "ParentNonUniqueFlasksAppliedToYou", true},
	}
	for _, c := range cases {
		e := ParseMod(c.line, false)
		if e.Extra != "" {
			t.Fatalf("%q: unparsed remainder %q — the line must parse WHOLE", c.line, e.Extra)
		}
		if len(e.ModList) != 1 {
			t.Fatalf("%q: got %d mods, want 1", c.line, len(e.ModList))
		}
		m := e.ModList[0]
		if m.Name() != "ExtraLinkEffect" || m.Type() != mod.TypeList {
			t.Fatalf("%q: got %s %s, want ExtraLinkEffect LIST", c.line, m.Name(), m.Type())
		}
		payload, ok := m.Value().List().(mod.MinionModifier)
		if !ok {
			t.Fatalf("%q: payload is %T, want mod.MinionModifier", c.line, m.Value().List())
		}
		if payload.Mod.Name() != c.inner || payload.Mod.Type() != mod.TypeFlag {
			t.Errorf("%q: nested %s %s, want %s FLAG", c.line, payload.Mod.Name(), payload.Mod.Type(), c.inner)
		}
		hasGE := false
		for _, tag := range payload.Mod.Tags() {
			if ge, ok := tag.(*mod.GlobalEffectTag); ok && ge.UnscalableTag {
				hasGE = true
			}
		}
		if hasGE != c.globalEffect {
			t.Errorf("%q: nested GlobalEffect/unscalable = %v, want %v", c.line, hasGE, c.globalEffect)
		}
	}

	qs := ParseMod("Quicksilver Flasks you Use also apply to nearby Allies", false)
	if qs.Extra != "" || len(qs.ModList) != 1 || qs.ModList[0].Name() != "QuickSilverAppliesToAllies" ||
		qs.ModList[0].Type() != mod.TypeFlag {
		t.Errorf("quicksilver row: remainder %q, mods %v", qs.Extra, qs.ModList)
	}

	fam := ParseMod("Flasks you Use apply to your Raised Zombies and Spectres", false)
	if fam.Extra != "" || len(fam.ModList) != 1 || fam.ModList[0].Name() != "FlasksApplyToMinion" ||
		len(fam.ModList[0].Tags()) != 1 {
		t.Errorf("umbilicus row: remainder %q, mods %v", fam.Extra, fam.ModList)
	} else if sn, ok := fam.ModList[0].Tags()[0].(*mod.SkillNameTag); !ok || !sn.IncludeTransfigured {
		t.Errorf("umbilicus row: tag %#v, want SkillName{Raise Zombie, Raise Spectre} includeTransfigured", fam.ModList[0].Tags()[0])
	}

	n := ParseMod("Your Linked Minions take 70% less Damage", false)
	if n.Extra != "" || len(n.ModList) != 1 || n.ModList[0].Name() != "MinionModifier" {
		t.Errorf("neighbour :4324 no longer parses to one MinionModifier: remainder %q, mods %v", n.Extra, n.ModList)
	}
}
