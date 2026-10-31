package exposition

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

type calcsOracle struct {
	Mods    map[string]float64 `json:"mods"`
	Entries map[string]struct {
		ModName   []string `json:"modName"`
		ModType   string   `json:"modType"`
		Cfg       string   `json:"cfg"`
		Enemy     bool     `json:"enemy"`
		ModSource string   `json:"modSource"`
		Actor     string   `json:"actor"`
	} `json:"entries"`
}

// The values are PoB-PoE2's Calcs tab entries over its main calculation (oracle/poe2_calcs.lua, `task
// calc-sections-poe2`).
func TestCalcsModsMatchPoB(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_calcs_sorceress_spark.json"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle calcsOracle
	if err := json.Unmarshal(raw, &oracle); err != nil {
		t.Fatal(err)
	}
	xml, err := os.ReadFile(filepath.Join("testdata", "sorceress_spark.xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.CalcsMods(CalcsModQuery{}); err == nil {
		t.Error("CalcsMods before Calculate: no error")
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	query := CalcsModQuery{Entries: map[string]CalcsModEntry{}}
	for k, e := range oracle.Entries {
		query.Entries[k] = CalcsModEntry{Cfg: e.Cfg, ModSource: e.ModSource, Enemy: e.Enemy, ModName: e.ModName, ModType: e.ModType, Actor: e.Actor}
	}
	got, err := calc.CalcsMods(query)
	if err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(query.Entries))
	for k := range query.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	nonzero := 0
	for _, k := range keys {
		want, inPoB := oracle.Mods[k]
		v, inOurs := got[k]
		if inPoB != inOurs {
			t.Errorf("%s %v: PoB-PoE2 has a value %v, ours %v", k, query.Entries[k].ModName, inPoB, inOurs)
			continue
		}
		if want != 0 {
			nonzero++
		}
		assertClose(t, "sorceress_spark.xml", k, v, want)
	}
	if nonzero == 0 {
		t.Error("no entry has a nonzero PoB-PoE2 value; the fixture tests nothing")
	}
}
