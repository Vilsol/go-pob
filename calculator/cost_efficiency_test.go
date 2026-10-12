package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestCostEfficiencyLinesParseWhole pins the CostEfficiency name family (ModParser.lua:198-218).
func TestCostEfficiencyLinesParseWhole(t *testing.T) {
	for _, c := range []struct{ line, wantName string }{
		{"15% increased Mana Cost Efficiency", "ManaCostEfficiency"},
		{"15% increased Cost Efficiency of Attacks", "CostEfficiency"},
		{"25% increased Cost Efficiency of Attacks", "CostEfficiency"},
		{"5% increased Cost Efficiency", "CostEfficiency"},
		{"20% increased Mana Cost Efficiency of Curse Skills", "ManaCostEfficiency"},
		{"20% increased Mana Cost Efficiency of Link Skills", "ManaCostEfficiency"},
		{"10% increased Life Cost Efficiency", "LifeCostEfficiency"},
		{"10% increased Energy Shield Cost Efficiency", "ESCostEfficiency"},
		{"10% increased Rage Cost Efficiency", "RageCostEfficiency"},
		{"10% increased Soul Cost Efficiency", "SoulCostEfficiency"},
		{"Skills Cost Life instead of 15% of Mana Cost", "HybridManaAndLifeCost_Life"},
		{"Attack Skills Cost Life instead of 30% of Mana Cost", "HybridManaAndLifeCost_Life"},
	} {
		t.Run(c.line, func(t *testing.T) {
			e := ParseMod(c.line, false)
			if strings.TrimSpace(e.Extra) != "" {
				t.Fatalf("%q left Extra=%q -- processStats DROPS the whole line", c.line, e.Extra)
			}
			if len(e.ModList) == 0 {
				t.Fatalf("%q yielded no mods at all", c.line)
			}
			found := false
			for _, m := range e.ModList {
				if m.Name() == c.wantName {
					found = true
				}
			}
			if !found {
				names := make([]string, 0, len(e.ModList))
				for _, m := range e.ModList {
					names = append(names, m.Name())
				}
				t.Errorf("%q did not yield %q, got %v", c.line, c.wantName, names)
			}
		})
	}
}

// TestRetaliationCostEfficiencyStaysDropped ports an ABSENCE.
func TestRetaliationCostEfficiencyStaysDropped(t *testing.T) {
	const line = "100% increased Cost Efficiency of Retaliation Skills"
	e := ParseMod(line, false)
	if strings.TrimSpace(e.Extra) == "" {
		t.Errorf("%q now parses whole (Extra empty). PoB has no \"of retaliation skills\" tag "+
			"phrase at this pin, so we would be emitting a modifier the oracle does not.", line)
	}
	const control = "100% increased Cost Efficiency of Link Skills"
	if c := ParseMod(control, false); strings.TrimSpace(c.Extra) != "" {
		t.Errorf("control: %q left Extra=%q -- this test cannot tell a ported absence from a "+
			"broken parser", control, c.Extra)
	}
}

// TestCostEfficiencyCarriesItsSkillTypeTag is the §3.1 guard.
func TestCostEfficiencyCarriesItsSkillTypeTag(t *testing.T) {
	for _, c := range []struct {
		line     string
		wantName string
		wantTag  bool
	}{
		{"15% increased Cost Efficiency of Attacks", "CostEfficiency", true},
		{"15% increased Cost Efficiency of Spells", "CostEfficiency", true},
		{"15% increased Mana Cost Efficiency of Attacks", "ManaCostEfficiency", true},
		{"5% increased Cost Efficiency", "CostEfficiency", false},
		{"15% increased Mana Cost Efficiency", "ManaCostEfficiency", false},
	} {
		t.Run(c.line, func(t *testing.T) {
			e := ParseMod(c.line, false)
			if strings.TrimSpace(e.Extra) != "" {
				t.Fatalf("%q left Extra=%q", c.line, e.Extra)
			}
			var target []mod.Tag
			found := false
			for _, m := range e.ModList {
				if m.Name() != c.wantName {
					continue
				}
				found = true
				target = m.Tags()
			}
			if !found {
				t.Fatalf("%q did not yield %q at all", c.line, c.wantName)
			}
			hasSkillType := false
			for _, tg := range target {
				if tg.Type() == mod.TypeSkillType {
					hasSkillType = true
				}
			}
			if hasSkillType != c.wantTag {
				t.Errorf("%q: %q carries a SkillType tag = %v, want %v (tags: %v)",
					c.line, c.wantName, hasSkillType, c.wantTag, target)
			}
		})
	}
}
