package mcp

import (
	"slices"
	"strings"
	"testing"
)

// TestEditFlagsDPSGainsThatLowerSpeed: an edit that raises DPS while the main skill attacks more slowly says so, as a
// ranked row does; one that raises speed does not.
func TestEditFlagsDPSGainsThatLowerSpeed(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Marauder", "level": 70})["handle"].(string)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "craft", "base": "Siege Axe", "slot": "Weapon 1"})})
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Heavy Strike"}, "slot": "Weapon 1"},
		map[string]any{"op": "set_main", "group": 1})})
	for gem, want := range map[string]bool{"Melee Physical Damage": true, "Faster Attacks": false} {
		dry := call(t, s, "edit_skills", map[string]any{"handle": h, "dry_run": true, "ops": ops(map[string]any{"op": "add_gem", "group": 1, "gem": gem})})
		w, _ := dry["warnings_added"].([]any)
		if got := slices.ContainsFunc(w, func(x any) bool { return strings.HasPrefix(x.(string), "slower: ") }); got != want {
			t.Errorf("%s: slower flag %v: %v", gem, got, w)
		}
	}
}

// TestSlowerFlagNamesEverySpeedLoss: a swap like Ruthless for Pulverise lowers both DPS and speed; the speed loss is
// named too, since nothing else in the result shows it unless Speed is watched.
func TestSlowerFlagNamesEverySpeedLoss(t *testing.T) {
	base := outputEnv(map[string]float64{"CombinedDPS": 100, "Speed": 1.64})
	if f := slowerFlag(base, outputEnv(map[string]float64{"CombinedDPS": 78, "Speed": 1.4}), mainDPS); len(f) != 1 || f[0] != "slower: Speed 1.64 -> 1.4 while MainDPS falls" {
		t.Fatalf("flag %v", f)
	}
	if f := slowerFlag(base, outputEnv(map[string]float64{"CombinedDPS": 104, "Speed": 1.4}), mainDPS); len(f) != 1 || f[0] != "slower: Speed 1.64 -> 1.4 while MainDPS rises" {
		t.Fatalf("flag %v", f)
	}
	if f := slowerFlag(base, outputEnv(map[string]float64{"CombinedDPS": 78, "Speed": 1.64}), mainDPS); len(f) != 0 {
		t.Fatalf("flag %v with speed kept", f)
	}
}
