package exposition

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

// The cases are oracle/poe2_build_planner.lua's record of BuildExportPoE2.BuildTable over each fixture's trees,
// skill sets and item sets; PoB-PoE2 walks the passives and the slot map with
// pairs(), so both lists are compared sorted.
var buildPlannerFixtures = []string{"build_planner", "tree_specs", "sorceress_three_spells", "warrior_dual_mace"}

var buildPlannerMetas = []BuildPlannerMeta{
	{Name: "Planner build", Author: "Author", Description: "", UseGeneratedItemText: true},
	{Name: "Planner build", Author: "Someone", Description: "Line one {braces}", UseGeneratedItemText: false},
}

func sortPlannerLists(t *testing.T, build map[string]any) {
	t.Helper()
	for _, field := range []string{"passives", "inventory_slots"} {
		list, _ := build[field].([]any)
		key := func(v any) string {
			m := v.(map[string]any)
			id, _ := m["id"].(string)
			if inv, ok := m["inventory_id"].(string); ok {
				id = inv
			}
			slot, _ := m["slot_x"].(float64)
			return id + ":" + strconv.Itoa(int(slot))
		}
		sort.SliceStable(list, func(i, j int) bool { return key(list[i]) < key(list[j]) })
		build[field] = list
	}
}

func TestBuildPlannerExportMatchesPoB(t *testing.T) {
	for _, name := range buildPlannerFixtures {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_build_planner_"+name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var oracle struct {
				Cases []struct {
					SpecIndex  int            `json:"specIndex"`
					SkillSetID int            `json:"skillSetId"`
					ItemSetID  int            `json:"itemSetId"`
					Meta       int            `json:"meta"`
					Path       string         `json:"path"`
					Build      map[string]any `json:"build"`
				} `json:"cases"`
			}
			if err := json.Unmarshal(raw, &oracle); err != nil {
				t.Fatal(err)
			}
			xml, err := os.ReadFile(filepath.Join("testdata", name+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			calc, err := ParseBuild(string(xml))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := calc.BuildPlannerExport(buildPlannerMetas[0], 1, 1, 1); err == nil {
				t.Error("BuildPlannerExport before Calculate: no error")
			}
			if _, err := calc.Calculate(); err != nil {
				t.Fatal(err)
			}
			sets := calc.BuildPlannerSets()
			if len(oracle.Cases) != len(sets.Trees)*len(sets.SkillSets)*len(sets.ItemSets)*len(buildPlannerMetas) {
				t.Fatalf("%d recorded cases for %d trees, %d skill sets and %d item sets", len(oracle.Cases), len(sets.Trees), len(sets.SkillSets), len(sets.ItemSets))
			}
			for _, tc := range oracle.Cases {
				file, err := calc.BuildPlannerExport(buildPlannerMetas[tc.Meta-1], tc.SpecIndex, tc.SkillSetID, tc.ItemSetID)
				if err != nil {
					t.Fatal(err)
				}
				if file.FileName != tc.Path {
					t.Errorf("tree %d: file name %q, PoB-PoE2 %q", tc.SpecIndex, file.FileName, tc.Path)
				}
				var got map[string]any
				if err := json.Unmarshal([]byte(file.JSON), &got); err != nil {
					t.Fatal(err)
				}
				sortPlannerLists(t, got)
				sortPlannerLists(t, tc.Build)
				for _, field := range []string{"name", "ascendancy", "author", "description", "passives", "skills", "inventory_slots"} {
					if !reflect.DeepEqual(got[field], tc.Build[field]) {
						ours, _ := json.MarshalIndent(got[field], "", " ")
						want, _ := json.MarshalIndent(tc.Build[field], "", " ")
						t.Errorf("tree %d, skill set %d, item set %d, meta %d: %s\n ours %s\n PoB-PoE2 %s", tc.SpecIndex, tc.SkillSetID, tc.ItemSetID, tc.Meta, field, ours, want)
					}
				}
			}
		})
	}
}

func TestBuildPlannerFileName(t *testing.T) {
	// BuildExportPoE2.BuildPath's file name: unsafe characters become dashes, an empty name is Unnamed.
	for in, want := range map[string]string{
		"":              "Unnamed [0.5].build",
		`a/b\c:d*e?f"g`: "a-b-c-d-e-f-g [0.5].build",
		"x<y>z|\tw":     "x-y-z--w [0.5].build",
	} {
		if got := BuildPlannerFileName(in, "0_5"); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}
