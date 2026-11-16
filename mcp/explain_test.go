package mcp

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func explain(t *testing.T, s *Server, h, key string) map[string]any {
	t.Helper()
	res := assertSize(t, s, "get_stats", map[string]any{"handle": h, "explain": key}, 4096)
	return res["explain"].(map[string]any)
}

func TestExplainUsesPoBsBreakdown(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	for key, want := range map[string]string{
		"TotalEHP":                "total damage you can take",
		"PhysicalMaximumHitTaken": "maximum survivable enemy damage",
		"ManaReserved":            "Vitality",
	} {
		ex := explain(t, s, h, key)
		if !strings.Contains(strings.Join(toStrings(ex["breakdown"]), "\n"), want) {
			t.Fatalf("%s breakdown = %v", key, ex["breakdown"])
		}
	}
}

func TestExplainAccuracyReadsTheMainHand(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	ex := explain(t, s, h, "Accuracy")
	if ex["value"] == float64(0) || !strings.Contains(strings.Join(toStrings(ex["breakdown"]), "\n"), "= ") {
		t.Fatalf("Accuracy explain = %v", ex)
	}
}

func TestExplainCritMultiplierSourcesAddUpToTheValue(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	ex := explain(t, s, h, "CritMultiplier")
	v, _ := ex["hand_value"].(float64)
	if v == 0 {
		v = ex["value"].(float64)
	}
	if got := 1 + ex["base"].(float64)/100; math.Abs(got*ex["more"].(float64)-v) > 0.006 {
		t.Fatalf("1 + base %v / 100 = %v, value %v: %v", ex["base"], got, v, ex)
	}
	if lines := strings.Join(toStrings(ex["breakdown"]), "\n"); !strings.Contains(lines, "extra damage") || !strings.Contains(lines, "= 1 + ") {
		t.Fatalf("CritMultiplier breakdown = %v", ex["breakdown"])
	}
}

func toStrings(v any) []string {
	var out []string
	for _, x := range asList(v) {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func TestExplainResistanceListsAllElementalSources(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch"})["handle"].(string)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "customMods",
		"value": "+17% to all Elemental Resistances\n+5% to Fire Resistance"})})
	for _, key := range []string{"FireResist", "FireResistTotal"} {
		ex := explain(t, s, h, key)
		if ex["base"] != float64(17+5-60) {
			t.Fatalf("%s base %v (want -60 + 17 + 5): %v", key, ex["base"], ex["sources"])
		}
	}
}

// TestExplainDurationListsSupports: Duration's sources come from the main skill's list, as CalcSections.lua:742-747
// tabulates them, so a duration support shows up and the more total matches the skill's duration modifier.
func TestExplainDurationListsSupports(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch"})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Blade Vortex", "More Duration"}, "slot": "Body Armour", "level": 20})})
	mod := explain(t, s, h, "DurationMod")["value"].(float64)
	for _, key := range []string{"Duration", "DurationMod"} {
		ex := explain(t, s, h, key)
		if !strings.Contains(fmt.Sprint(ex["sources"]), "More Duration") || math.Abs(ex["more"].(float64)-mod) > 0.05 {
			t.Fatalf("%s explain (duration modifier %v) = %v", key, mod, ex)
		}
	}
}

// TestExplainAvoidanceListsCalcSectionsRows: an avoidance stat reads the mod names CalcSections lists for it (not a mod
// named after the stat), its immunity flags included; the fixture's bleed avoidance is Steelskin's guard and a flask's
// bleed immunity.
func TestExplainAvoidanceListsCalcSectionsRows(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	ex := explain(t, s, h, "BleedAvoidChance")
	got := make([]string, 0, len(asList(ex["sources"])))
	for _, src := range asList(ex["sources"]) {
		m := src.(map[string]any)
		got = append(got, fmt.Sprint(m["source"], " = ", m["value"]))
	}
	joined := strings.Join(got, "\n")
	if ex["value"] != float64(100) || !strings.Contains(joined, "Skill: Steelskin = 100 base") || !strings.Contains(joined, "Flask of the Iguana = True (Bleed Immune)") {
		t.Fatalf("BleedAvoidChance explain = %v", ex)
	}
}
