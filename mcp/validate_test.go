package mcp

import (
	"strings"
	"testing"
)

func hasWarning(list any, sub string) bool {
	for _, w := range asList(list) {
		if s, _ := w.(string); strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func asList(v any) []any {
	l, _ := v.([]any)
	return l
}

func TestValidateWarnsOnTooFewLinks(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "craft", "base": "Iron Gauntlets", "slot": "Gloves", "sockets": "R-R R-R"})})
	if !hasWarning(res["warnings_added"], "largest link group in Gloves is 2") {
		t.Fatalf("warnings_added = %v", res["warnings_added"])
	}
	if hasWarning(res["warnings_added"], "too many gems in your Gloves slot") {
		t.Fatalf("four sockets for four gems flagged as overflow: %v", res["warnings_added"])
	}
	res = call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "craft", "base": "Iron Gauntlets", "slot": "Gloves", "sockets": "R"})})
	if !hasWarning(res["warnings_added"], "too many gems in your Gloves slot") {
		t.Fatalf("one socket for four gems: warnings_added = %v", res["warnings_added"])
	}
}

func TestValidateWarnsOnPointsOverLevel(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	before := call(t, s, "get_build", map[string]any{"handle": h, "section": "warnings"})
	if hasWarning(before["warnings"], "passive points need character level") {
		t.Fatalf("the fixture already warns: %v", before["warnings"])
	}
	res := call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_level", "level": 60})})
	if !hasWarning(res["warnings_added"], "passive points need character level") {
		t.Fatalf("warnings_added = %v", res["warnings_added"])
	}
	after := call(t, s, "get_build", map[string]any{"handle": h, "section": "warnings"})
	if !hasWarning(after["warnings"], "(level is 60)") {
		t.Fatalf("get_build warnings = %v", after["warnings"])
	}
}

func TestValidateWarnsOnChargeOverrideOverMaximum(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "set_config", "var": "usePowerCharges", "value": true},
		map[string]any{"op": "set_config", "var": "overridePowerCharges", "value": 2})})
	if hasWarning(res["warnings_added"], "Power Charges") {
		t.Fatalf("an override under the maximum warned: %v", res["warnings_added"])
	}
	res = call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "overridePowerCharges", "value": 9})})
	if !hasWarning(res["warnings_added"], "overridePowerCharges is 9, above your maximum of 3 Power Charges") {
		t.Fatalf("warnings_added = %v", res["warnings_added"])
	}
}

func TestSocketMapSection(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := assertSize(t, s, "get_build", map[string]any{"handle": h, "section": "sockets"}, 2500)
	var body map[string]any
	for _, r := range asList(res["sockets"]) {
		if row := r.(map[string]any); row["slot"] == "Body Armour" {
			body = row
		}
	}
	if body == nil || body["sockets"] != "G-G-R-R-R-R" || body["links"] != float64(6) || body["gems"] != float64(6) {
		t.Fatalf("Body Armour socket row = %v (all %v)", body, res["sockets"])
	}
}
