package calculator2

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

type calcsListsRow struct {
	BuffList     string        `json:"BuffList"`
	CombatList   string        `json:"CombatList"`
	CurseList    string        `json:"CurseList"`
	SkillBuffs   []calcsModRow `json:"SkillBuffs"`
	SkillDebuffs []calcsModRow `json:"SkillDebuffs"`
}

type calcsModRow struct {
	Value      string `json:"value"`
	Name       string `json:"name"`
	Flags      string `json:"flags"`
	Tags       string `json:"tags"`
	Source     string `json:"source"`
	SourceName string `json:"sourceName"`
}

// TestCalcsBuffListsMatchPoB grades the Calcs tab's buff, combat buff and curse lists over the fixture builds
// against PoB-PoE2's CALCS calculation (oracle/poe2_calcs_buff_lists.lua), with the SkillBuffs and SkillDebuffs
// breakdowns' mod tables.
func TestCalcsBuffListsMatchPoB(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "testdata", "oracle", "poe2_calcs_buff_lists.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tree := loadTestTree(t)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	var combat, modRows int
	for sc.Scan() {
		var row struct {
			calcsListsRow
			Build      string         `json:"build"`
			Error      string         `json:"error"`
			MainSkill  string         `json:"mainSkill"`
			CalcsSkill string         `json:"calcsSkill"`
			Minion     *calcsListsRow `json:"Minion"`
		}
		if err := json.Unmarshal(sc.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		if row.CombatList != "" {
			combat++
		}
		modRows += len(row.SkillBuffs) + len(row.SkillDebuffs)
		t.Run(row.Build, func(t *testing.T) {
			if row.Error != "" {
				t.Fatalf("PoB-PoE2 failed: %s", row.Error)
			}
			path := fullDPSFixturePath(row.Build)
			if path == "" {
				t.Fatalf("no fixture %s", row.Build)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			build, err := pob2.ParseBuild(raw)
			if err != nil {
				t.Fatal(err)
			}
			mainEnv, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
			if err != nil {
				t.Fatal(err)
			}
			env := BuildCalcsOutput(mainEnv, BuildOutputUsed(mainEnv))
			// PoB-PoE2 names a skill by its granted effect, which a companion's renames for every instance
			main, calcs := getActiveSkillDisplayName(mainEnv.Player.MainSkill), getActiveSkillDisplayName(env.Player.MainSkill)
			if (main == calcs) != (row.MainSkill == row.CalcsSkill) {
				t.Errorf("Calcs tab skill %s, main %s; PoB %s, %s", calcs, main, row.CalcsSkill, row.MainSkill)
			}
			check := func(who string, got map[string]string, want calcsListsRow, minion bool) {
				for key, w := range map[string]string{"BuffList": want.BuffList, "CombatList": want.CombatList, "CurseList": want.CurseList} {
					if got[key] != w {
						t.Errorf("%s %s %q, PoB %q", who, key, got[key], w)
					}
				}
				for key, w := range map[string][]calcsModRow{"SkillBuffs": want.SkillBuffs, "SkillDebuffs": want.SkillDebuffs} {
					var rows []calcsModRow
					for _, r := range env.CalcsBreakdownModList(key, minion) {
						rows = append(rows, calcsModRow{Value: r.Value, Name: r.Name, Flags: r.Flags, Tags: r.Tags, Source: r.Source, SourceName: r.SourceName})
					}
					if !reflect.DeepEqual(rows, w) {
						t.Errorf("%s %s:\n ours %+v\n PoB  %+v", who, key, rows, w)
					}
				}
			}
			check("player", env.Player.OutputStrings, row.calcsListsRow, false)
			if (row.Minion != nil) != (env.Minion != nil) {
				t.Fatalf("minion: ours %v, PoB %v", env.Minion != nil, row.Minion != nil)
			}
			if row.Minion != nil {
				check("minion", env.Minion.OutputStrings, *row.Minion, true)
			}
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if modRows < 10 {
		t.Fatalf("only %d SkillBuffs / SkillDebuffs rows in the fixture; the oracle is not exercising the port", modRows)
	}
	if combat < 2 {
		t.Fatalf("only %d fixture builds have a combat buff; the oracle is not exercising the port", combat)
	}
}
