package exposition

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator2"
)

type modSectionOracle struct {
	Multi bool   `json:"multi"`
	Error string `json:"error"`
	Rows  []struct {
		Value      string   `json:"value"`
		Name       string   `json:"name"`
		Flags      string   `json:"flags"`
		Tags       string   `json:"tags"`
		Source     string   `json:"source"`
		SourceName string   `json:"sourceName"`
		Totals     []string `json:"totals"`
	} `json:"rows"`
}

// modSectionOf is the entry an oracle key names: modName list|modType|cfg|modSource|enemy|actor.
func modSectionOf(key string, multi bool) calculator2.ModSectionData {
	p := strings.Split(key, "|")
	return calculator2.ModSectionData{
		ModName: strings.Split(p[0], ","), Multi: multi, ModType: p[1], Cfg: p[2], ModSource: p[3], Enemy: p[4] == "true", Actor: p[5],
	}
}

// Every CalcSections.lua mod table is the one PoB-PoE2's CalcBreakdownControl:AddModSection builds over its main
// calculation (oracle/poe2_mod_sections.lua, `mise run breakdowns-poe2`).
func TestCalcsModSectionsMatchPoB(t *testing.T) {
	for _, name := range calcsFixtures {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_mod_sections_"+name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var oracle map[string]modSectionOracle
			if err := json.Unmarshal(raw, &oracle); err != nil {
				t.Fatal(err)
			}
			_, calc := readCalcsFixture(t, name)
			if _, err := calc.CalcsModSection(calculator2.ModSectionData{ModName: []string{"Life"}}); err == nil {
				t.Error("CalcsModSection before Calculate: no error")
			}
			if _, err := calc.Calculate(); err != nil {
				t.Fatal(err)
			}
			keys := make([]string, 0, len(oracle))
			for k := range oracle {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			withRows := 0
			for _, k := range keys {
				o := oracle[k]
				if o.Error != "" {
					continue
				}
				got, err := calc.CalcsModSection(modSectionOf(k, o.Multi))
				if err != nil {
					t.Fatal(err)
				}
				var want []calculator2.ModSectionRow
				for _, r := range o.Rows {
					want = append(want, calculator2.ModSectionRow{
						Value: r.Value, Name: r.Name, Flags: r.Flags, Tags: r.Tags, Source: r.Source, SourceName: r.SourceName, Totals: r.Totals,
					})
				}
				if len(want) > 0 {
					withRows++
				}
				shown := make([]calculator2.ModSectionRow, len(got))
				for i, r := range got {
					r.ItemID, r.NodeID = 0, 0
					shown[i] = r
				}
				if len(shown) == 0 {
					shown = nil
				}
				if !reflect.DeepEqual(shown, want) {
					g, _ := json.Marshal(shown)
					w, _ := json.Marshal(want)
					t.Errorf("%s:\n PoB-PoE2 %s\n ours     %s", k, w, g)
				}
			}
			if withRows < 50 {
				t.Errorf("only %d entries with rows", withRows)
			}
		})
	}
}
