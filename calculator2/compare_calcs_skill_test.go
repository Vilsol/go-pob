package calculator2

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// TestCompareSkillBreakdownMatchesPoB grades the Compare page Calcs header's SkillBuffs and SkillDebuffs panels against
// the rows PoB-PoE2's DrawSkillBreakdownPanel draws (oracle/poe2_compare_skill_breakdown.lua).
func TestCompareSkillBreakdownMatchesPoB(t *testing.T) {
	tree := loadTestTree(t)
	rows := 0
	for _, name := range []string{"skill_buffs", "minion_skill_debuffs"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "testdata", "oracle", "poe2_compare_skill_breakdown_"+name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var want map[string][]CompareSkillBreakdownRow
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatal(err)
			}
			xml, err := os.ReadFile(filepath.Join("testdata", name+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			build, err := pob2.ParseBuild(xml)
			if err != nil {
				t.Fatal(err)
			}
			mainEnv, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
			if err != nil {
				t.Fatal(err)
			}
			env := BuildCalcsOutput(mainEnv, BuildOutputUsed(mainEnv))
			for key, w := range want {
				rows += len(w)
				got := env.CompareSkillBreakdown(key)
				if len(w) == 0 && len(got) == 0 {
					continue
				}
				if !reflect.DeepEqual(got, w) {
					t.Errorf("%s:\n ours %+v\n PoB  %+v", key, got, w)
				}
			}
		})
	}
	if rows < 5 {
		t.Fatalf("only %d rows in the fixtures; the oracle is not exercising the port", rows)
	}
}
