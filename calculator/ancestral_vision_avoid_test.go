package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestAncestralVisionSuppressionLineParses: Ancestral Vision — ModParser.lua:3813-3816 and CalcDefence.lua:1556-1560.
func TestAncestralVisionSuppressionLineParses(t *testing.T) {
	const line = "Modifiers to Chance to Suppress Spell Damage also apply to Chance to Avoid " +
		"Elemental Ailments at 50% of their Value"

	e := ParseMod(line, false)
	if e == nil {
		t.Fatalf("ParseMod(%q) returned nothing", line)
	}
	testza.AssertEqual(t, "", e.Extra,
		"leftover Extra — processModLine discards the whole line, so the mods below never land")
	testza.AssertEqual(t, 2, len(e.ModList), "the row emits a BASE percent and a FLAG")

	var sawPercent, sawFlag bool
	for _, m := range e.ModList {
		switch m.Name() {
		case "SpellSuppressionAppliesToAilmentAvoidancePercent":
			sawPercent = true
			testza.AssertEqual(t, float64(50), m.Value().Float())
		case "SpellSuppressionAppliesToAilmentAvoidance":
			sawFlag = true
		}
	}
	testza.AssertTrue(t, sawPercent, "no SpellSuppressionAppliesToAilmentAvoidancePercent mod")
	testza.AssertTrue(t, sawFlag,
		"no SpellSuppressionAppliesToAilmentAvoidance flag — without it the three calc_perform "+
			"consumers and the calc_defence one all stay dead")
}

// TestAncestralVisionUsesRawSuppressionChance: The consumer half: the flag plus a suppression chance must produce an AvoidElementalAilments BASE mod.
func TestAncestralVisionUsesRawSuppressionChance(t *testing.T) {
	for _, tc := range []struct {
		name      string
		suppress  float64
		percent   float64
		wantAvoid float64
	}{
		{"half of 100", 100, 50, 50},
		{"above the cap", 120, 50, 60},
		{"floors rather than rounds", 55, 50, 27},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := moddb.NewModDB()
			db.AddMod(mod.NewFloat("SpellSuppressionChance", mod.TypeBase, tc.suppress).Source("Test"))
			db.AddMod(mod.NewFloat("SpellSuppressionAppliesToAilmentAvoidancePercent",
				mod.TypeBase, tc.percent).Source("Test"))
			db.AddMod(mod.NewFlag("SpellSuppressionAppliesToAilmentAvoidance", true).Source("Test"))

			got := ancestralVisionAvoid(db)
			testza.AssertEqual(t, tc.wantAvoid, got)
		})
	}
}
