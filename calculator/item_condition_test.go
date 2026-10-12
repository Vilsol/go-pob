package calculator

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func TestItemConditionPatternsParse(t *testing.T) {
	for _, c := range []struct {
		line   string
		slot   string
		search string
		neg    bool
		both   bool
		all    bool
	}{
		{"15% increased maximum Life if there are no Life Modifiers on Equipped Body Armour",
			"Body Armour", "life", true, false, false},
		{"10% increased Armour if Equipped Helmet has an Armour modifier",
			"Helmet", "armour", false, false, false},
		{"5% increased Damage if both equipped Rings have a Life modifier",
			"Ring", "life", false, true, false},
		{"5% increased Damage if there are no Life modifiers on other equipped items",
			"{SlotName}", "life", true, false, true},
	} {
		t.Run(c.line, func(t *testing.T) {
			e := ParseMod(c.line, false)
			if strings.TrimSpace(e.Extra) != "" {
				t.Fatalf("%q left Extra=%q -- processModLine DROPS the whole line", c.line, e.Extra)
			}
			var tag *mod.ItemConditionTag
			for _, m := range e.ModList {
				for _, tg := range m.Tags() {
					if ic, ok := tg.(*mod.ItemConditionTag); ok {
						tag = ic
					}
				}
			}
			if tag == nil {
				t.Fatalf("%q produced no ItemCondition tag; the mod would apply UNCONDITIONALLY, "+
					"which is worse than dropping it", c.line)
			}
			if got := strings.TrimSpace(tag.ItemSlot); got != c.slot {
				t.Errorf("itemSlot = %q, want %q", got, c.slot)
			}
			if got := strings.ToLower(strings.TrimSpace(tag.SearchCond)); got != c.search {
				t.Errorf("searchCond = %q, want %q", got, c.search)
			}
			if tag.Neg != c.neg {
				t.Errorf("neg = %v, want %v", tag.Neg, c.neg)
			}
			if tag.BothSlots != c.both {
				t.Errorf("bothSlots = %v, want %v", tag.BothSlots, c.both)
			}
			if tag.AllSlots != c.all {
				t.Errorf("allSlots = %v, want %v", tag.AllSlots, c.all)
			}
		})
	}
}

// TestFindModifierSubstringHonoursTheTables pins the two hand-maintained tables.
func TestFindModifierSubstringHonoursTheTables(t *testing.T) {
	for _, c := range []struct {
		name     string
		explicit []string
		want     bool
	}{
		{"plain life line matches", []string{"+80 to maximum life"}, true},
		{"no life line", []string{"+30% to fire resistance"}, false},
		{"excluded: life as extra maximum energy shield",
			[]string{"10% of life as extra maximum energy shield"}, false},
		{"excluded: when on full life", []string{"20% increased damage when on full life"}, false},
		{"special: blood magic counts", []string{"blood magic"}, true},
		{"line naming life as a modifier type", []string{"cannot roll life modifier"}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := findModifierSubstring(c.explicit, nil, "life", "body armour")
			if got != c.want {
				t.Errorf("findModifierSubstring(%v, \"life\", \"body armour\") = %v, want %v",
					c.explicit, got, c.want)
			}
		})
	}
}

// TestItemTagTablesUseOnlyAnchors guards the promise luaFindPattern makes: the transcribed tables contain only plain substrings, optionally anchored with ^ or $.
func TestItemTagTablesUseOnlyAnchors(t *testing.T) {
	checked := 0
	for _, table := range []map[string]map[string][]string{itemTagSpecial, itemTagSpecialExclusionPattern} {
		for cond, slots := range table {
			for slot, patterns := range slots {
				for _, p := range patterns {
					checked++
					body := strings.TrimSuffix(strings.TrimPrefix(p, "^"), "$")
					if strings.ContainsAny(body, "%[]()*+-?") {
						t.Errorf("%s/%s: pattern %q contains Lua pattern metacharacters; "+
							"luaFindPattern would match it literally", cond, slot, p)
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("REFUSING: the tables are empty, so this guard checked nothing")
	}
	t.Logf("checked %d patterns", checked)
}

// TestLifeMasteryReachesTheOracle is the acceptance test: the Life Mastery line must now reach Life on a build that selects it.
func TestLifeMasteryReachesTheOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	for _, build := range []string{
		"0001-necromancer-unknown.xml",
		"0002-pathfinder-unknown.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok {
				t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
			}
			env := corpusEnvForLoadedData(t, build)

			if len(env.Spec.MasterySelections) == 0 {
				t.Fatalf("premise broken: %s selects no mastery effects at all", build)
			}
			want := rec.Output["Life"]
			if want == 0 {
				t.Fatalf("REFUSING: PoB reports Life 0 for %s", build)
			}
			if got := env.Player.Output["Life"]; math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
				t.Errorf("%s: Life = %v, PoB reports %v", build, got, want)
			}
		})
	}
}
