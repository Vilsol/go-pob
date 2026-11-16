package mcp

import (
	"slices"
	"strings"
	"testing"
)

// TestEditFlagsDPSGainsThatShrinkTheArea: Concentrated Effect raises Ice Nova's DPS and shrinks its area; the edit says
// so, as a ranked row does; a support that leaves the area alone does not.
func TestEditFlagsDPSGainsThatShrinkTheArea(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Ice Nova"}}, map[string]any{"op": "set_main", "group": 1})})
	for gem, want := range map[string]bool{"Concentrated Effect": true, "Controlled Destruction": false} {
		dry := call(t, s, "edit_skills", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "add_gem", "group": 1, "gem": gem})})
		w, _ := dry["warnings_added"].([]any)
		if got := slices.ContainsFunc(w, func(x any) bool { return strings.HasPrefix(x.(string), "area: ") }); got != want {
			t.Errorf("%s: area flag %v: %v", gem, got, w)
		}
	}
}

// TestAreaFlagNamesAShrinkingArea: a DPS gain with a smaller area of effect is flagged; a loss, or a gain with the area
// kept, is not.
func TestAreaFlagNamesAShrinkingArea(t *testing.T) {
	base := outputEnv(map[string]float64{"CombinedDPS": 100, "AreaOfEffectMod": 1.5, "AreaOfEffectRadius": 22})
	if f := areaFlag(base, outputEnv(map[string]float64{"CombinedDPS": 140, "AreaOfEffectMod": 1.05, "AreaOfEffectRadius": 18}), mainDPS); len(f) != 1 || f[0] != "area: AreaOfEffectMod 1.5 -> 1.05 (radius 22 -> 18) while MainDPS rises" {
		t.Fatalf("flag %v", f)
	}
	if f := areaFlag(base, outputEnv(map[string]float64{"CombinedDPS": 90, "AreaOfEffectMod": 1.05}), mainDPS); len(f) != 0 {
		t.Fatalf("flag %v on a loss", f)
	}
	if f := areaFlag(base, outputEnv(map[string]float64{"CombinedDPS": 140, "AreaOfEffectMod": 1.5, "AreaOfEffectRadius": 22}), mainDPS); len(f) != 0 {
		t.Fatalf("flag %v with the area kept", f)
	}
}
