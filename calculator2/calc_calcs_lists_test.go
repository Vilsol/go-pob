package calculator2

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

type calcsListsRow struct {
	BuffList   string `json:"BuffList"`
	CombatList string `json:"CombatList"`
	CurseList  string `json:"CurseList"`
}

// TestCalcsBuffListsMatchPoB grades the Calcs tab's buff, combat buff and curse lists over the fixture builds
// against PoB-PoE2's CALCS calculation (oracle/poe2_calcs_buff_lists.lua).
func TestCalcsBuffListsMatchPoB(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "testdata", "oracle", "poe2_calcs_buff_lists.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tree := loadTestTree(t)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	var combat int
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
		t.Run(row.Build, func(t *testing.T) {
			if row.Error != "" {
				t.Fatalf("PoB-PoE2 failed: %s", row.Error)
			}
			if row.MainSkill != row.CalcsSkill {
				t.Skipf("PoB-PoE2's Calcs tab shows %s, its main calculation %s", row.CalcsSkill, row.MainSkill)
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
			env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
			if err != nil {
				t.Fatal(err)
			}
			BuildOutputUsed(env)
			check := func(who string, got map[string]string, want calcsListsRow) {
				for key, w := range map[string]string{"BuffList": want.BuffList, "CombatList": want.CombatList, "CurseList": want.CurseList} {
					if got[key] != w {
						t.Errorf("%s %s %q, PoB %q", who, key, got[key], w)
					}
				}
			}
			check("player", env.Player.OutputStrings, row.calcsListsRow)
			if (row.Minion != nil) != (env.Minion != nil) {
				t.Fatalf("minion: ours %v, PoB %v", env.Minion != nil, row.Minion != nil)
			}
			if row.Minion != nil {
				check("minion", env.Minion.OutputStrings, *row.Minion)
			}
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if combat < 2 {
		t.Fatalf("only %d fixture builds have a combat buff; the oracle is not exercising the port", combat)
	}
}
