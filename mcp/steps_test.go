package mcp

import (
	"fmt"
	"strings"
	"testing"
)

func TestCompareBuildsStepsListsEachRevision(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_level", "level": 80})})
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "craft", "base": "Two-Stone Ring", "slot": "Ring 2", "affixes": []string{"+# to maximum Life"}})})
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "enemyIsBoss", "value": "None"})})
	res := assertSize(t, s, "compare_builds", map[string]any{"a": h + "@0", "b": h + "@3", "steps": true}, capCompareSteps)
	steps := asList(res["steps"])
	if len(steps) != 3 {
		t.Fatalf("steps = %v", res["steps"])
	}
	want := []string{"level 90 -> 80", "Ring 2:", "enemyIsBoss"}
	for i, st := range steps {
		row := st.(map[string]any)
		if row["revision"] != float64(i+1) || !strings.Contains(row["changes"].(string), want[i]) {
			t.Fatalf("step %d = %v", i+1, row)
		}
	}
	if d, ok := steps[0].(map[string]any)["delta"].(map[string]any); !ok || d["Life"] == nil {
		t.Fatalf("level step delta = %v", steps[0])
	}
	other := importFixture(t, s)
	if e := callErr(t, s, "compare_builds", map[string]any{"a": h + "@0", "b": other, "steps": true}); e.Code != CodeInvalidArgument {
		t.Fatalf("steps across handles: %s", e.Code)
	}
}

func TestCompareStepsNamesItemModAndGemLevelChanges(t *testing.T) {
	s := testServer(t)
	imp := call(t, s, "import_build", map[string]any{"source": fixture("3.xml")})
	h := imp["handle"].(string)
	craft := func(affix string) {
		call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "craft", "base": "Two-Stone Ring", "slot": "Ring 2", "affixes": []string{affix}})})
	}
	craft("+# to maximum Life")
	craft("+# to maximum Mana")
	main := imp["summary"].(map[string]any)["main_skill"].(map[string]any)
	gem := main["name"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_gem_level", "group": main["group"], "gem": gem, "level": 1, "quality": 7})})
	res := call(t, s, "compare_builds", map[string]any{"a": h + "@1", "b": h + "@3", "steps": true})
	steps := asList(res["steps"])
	item := steps[0].(map[string]any)["changes"].(string)
	if !strings.Contains(item, "Ring 2:") || !strings.Contains(item, "maximum Mana") || !strings.Contains(item, "maximum Life") {
		t.Fatalf("item step = %q", item)
	}
	gems := steps[1].(map[string]any)["changes"].(string)
	if !strings.Contains(gems, gem) || !strings.Contains(gems, "level") || !strings.Contains(gems, "-> 1") || !strings.Contains(gems, "-> 7") {
		t.Fatalf("gem step = %q", gems)
	}
}

// TestCompareStepsSkipReverted: after a revert the next edit builds on the revision reverted to, so the steps from 0
// follow that line and leave the reverted revision out.
func TestCompareStepsSkipReverted(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_level", "level": 80})})
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_level", "level": 70})})
	call(t, s, "revert_build", map[string]any{"handle": h})
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "enemyIsBoss", "value": "None"})})
	res := call(t, s, "compare_builds", map[string]any{"a": h + "@0", "b": h + "@3", "steps": true})
	steps := asList(res["steps"])
	if len(steps) != 2 {
		t.Fatalf("steps = %v", res)
	}
	last := steps[1].(map[string]any)
	if last["revision"] != float64(3) || strings.Contains(last["changes"].(string), "level") || !strings.Contains(last["changes"].(string), "enemyIsBoss") {
		t.Fatalf("step after the revert = %v", last)
	}
	if !strings.Contains(fmt.Sprint(res["reverted"]), "2") {
		t.Fatalf("reverted = %v", res["reverted"])
	}
}
