package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestProfaneGroundRowsParseWhole pins ModParser.lua:5544-5549.
func TestProfaneGroundRowsParseWhole(t *testing.T) {
	parseWhole := func(t *testing.T, line string) mod.Mod {
		t.Helper()
		e := ParseMod(line, false)
		if e == nil {
			t.Fatalf("ParseMod(%q) returned nothing", line)
		}
		testza.AssertEqual(t, "", e.Extra,
			"%q left a remainder, and an unparsed remainder discards the WHOLE line", line)
		if len(e.ModList) != 1 {
			t.Fatalf("%q produced %d mods, want 1", line, len(e.ModList))
		}
		return e.ModList[0]
	}

	condVars := func(m mod.Mod) []string {
		var out []string
		for _, tag := range m.Tags() {
			if c, ok := tag.(*mod.ConditionTag); ok {
				out = append(out, c.VarList...)
			}
		}
		return out
	}

	t.Run("profane ground on crit, int highest", func(t *testing.T) {
		m := parseWhole(t, "25% chance to create Profane Ground on Critical Strike "+
			"if Intelligence is your highest Attribute")
		testza.AssertEqual(t, "Condition:CreateProfaneGround", m.Name())
		testza.AssertEqual(t, []string{"IntHighestAttribute"}, condVars(m),
			"the pin gates this on IntHighestAttribute; an untagged flag would apply to every build")
	})

	t.Run("consecrated path and purifying flame", func(t *testing.T) {
		m := parseWhole(t, "Consecrated Path and Purifying Flame create Profane Ground "+
			"instead of Consecrated Ground")
		testza.AssertEqual(t, "Condition:CreateProfaneGround", m.Name())
	})

	t.Run("neighbour: consecrated ground, str highest", func(t *testing.T) {
		m := parseWhole(t, "You have Consecrated Ground around you while stationary "+
			"if Strength is your highest Attribute")
		testza.AssertEqual(t, "Condition:OnConsecratedGround", m.Name(),
			"the new Profane row must not swallow its Consecrated neighbour")
		testza.AssertContains(t, condVars(m), "StrHighestAttribute")
	})
}
