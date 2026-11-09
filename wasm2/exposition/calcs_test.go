package exposition

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type calcsOracle struct {
	MainSkill string                     `json:"mainSkill"`
	Output    map[string]json.RawMessage `json:"output"`
	Mods      map[string]float64         `json:"mods"`
	Entries   map[string]struct {
		ModName   []string `json:"modName"`
		ModType   string   `json:"modType"`
		Cfg       string   `json:"cfg"`
		Enemy     bool     `json:"enemy"`
		ModSource string   `json:"modSource"`
		Actor     string   `json:"actor"`
	} `json:"entries"`
}

// calcsFixtures are the builds oracle/poe2_calcs.lua records (`mise run calc-sections-poe2`): a spell, and an attack
// dual wielding with damage over time.
var calcsFixtures = []string{"sorceress_spark", "warrior_dual_mace"}

// calcsInputFixture is an oracle/poe2_calcs.lua recording after the Calcs tab's own inputs moved off the sidebar's.
type calcsInputFixture struct {
	name, xml       string
	skillNumber     int
	buffMode        string
	mainActiveSkill int
}

var calcsInputFixtures = []calcsInputFixture{
	{name: "sorceress_three_spells_arc_unbuffed", xml: "sorceress_three_spells", skillNumber: 2, buffMode: "UNBUFFED"},
	{name: "sorceress_three_spells_fireball_combat", xml: "sorceress_three_spells", skillNumber: 1, buffMode: "COMBAT", mainActiveSkill: 2},
	{name: "warrior_dual_mace_buffed", xml: "warrior_dual_mace", buffMode: "BUFFED"},
}

func readCalcsOracle(t *testing.T, name, xmlName string) (calcsOracle, *Calculator) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_calcs_"+name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle calcsOracle
	if err := json.Unmarshal(raw, &oracle); err != nil {
		t.Fatal(err)
	}
	xml, err := os.ReadFile(filepath.Join("testdata", xmlName+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	return oracle, calc
}

func readCalcsFixture(t *testing.T, name string) (calcsOracle, *Calculator) {
	t.Helper()
	return readCalcsOracle(t, name, name)
}

// readCalcsInputFixture sets the inputs as the tab's controls do, between calculations as the oracle does.
func readCalcsInputFixture(t *testing.T, f calcsInputFixture) (calcsOracle, *Calculator) {
	t.Helper()
	oracle, calc := readCalcsOracle(t, f.name, f.xml)
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	if f.skillNumber != 0 {
		calc.SetCalcsSkillNumber(f.skillNumber)
	}
	if f.buffMode != "" {
		calc.SetCalcsBuffMode(f.buffMode)
	}
	if f.mainActiveSkill != 0 {
		if err := calc.SetMainActiveSkillCalcs(f.mainActiveSkill); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	return oracle, calc
}

// The values are PoB-PoE2's Calcs tab entries over its main calculation.
func TestCalcsModsMatchPoB(t *testing.T) {
	for _, name := range calcsFixtures {
		t.Run(name, func(t *testing.T) {
			oracle, calc := readCalcsFixture(t, name)
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
			nonzero, dot := 0, 0
			for _, k := range keys {
				want, inPoB := oracle.Mods[k]
				v, inOurs := got[k]
				if inPoB != inOurs {
					t.Errorf("%s %v: PoB-PoE2 has a value %v, ours %v", k, query.Entries[k].ModName, inPoB, inOurs)
					continue
				}
				if want != 0 {
					nonzero++
					if c := query.Entries[k].Cfg; c != "" && c != "skill" {
						dot++
					}
				}
				assertClose(t, name, k, v, want)
			}
			if nonzero == 0 {
				t.Error("no entry has a nonzero PoB-PoE2 value; the fixture tests nothing")
			}
			if name == "warrior_dual_mace" && dot == 0 {
				t.Error("no damage over time cfg entry has a nonzero PoB-PoE2 value")
			}
		})
	}
}

// PoB-PoE2's true and string outputs and its MainHand and OffHand tables, as the Calcs page reads them.
func TestCalcsOutputsMatchPoB(t *testing.T) {
	for _, name := range calcsFixtures {
		t.Run(name, func(t *testing.T) {
			oracle, calc := readCalcsFixture(t, name)
			if _, err := calc.CalcsOutputs(); err == nil {
				t.Error("CalcsOutputs before Calculate: no error")
			}
			if _, err := calc.Calculate(); err != nil {
				t.Fatal(err)
			}
			got, err := calc.CalcsOutputs()
			if err != nil {
				t.Fatal(err)
			}
			var wantFlags, gotFlags []string
			for k, raw := range oracle.Output {
				if string(raw) == "true" {
					wantFlags = append(wantFlags, k)
				}
			}
			for k := range got.Flags {
				gotFlags = append(gotFlags, k)
			}
			sort.Strings(wantFlags)
			sort.Strings(gotFlags)
			if strings.Join(gotFlags, " ") != strings.Join(wantFlags, " ") {
				t.Errorf("true outputs:\n PoB-PoE2 %v\n ours     %v", wantFlags, gotFlags)
			}
			wantStrings := map[string]string{}
			for k, raw := range oracle.Output {
				var s string
				if json.Unmarshal(raw, &s) == nil {
					wantStrings[k] = s
				}
			}
			checkCalcsNumbers(t, name, oracle, got.Output)
			gotStrings := maps.Clone(got.Strings)
			if len(wantStrings) == 0 || !reflect.DeepEqual(gotStrings, wantStrings) {
				t.Errorf("string outputs:\n PoB-PoE2 %v\n ours     %v", wantStrings, gotStrings)
			}
			for _, hand := range []struct {
				ns   string
				ours map[string]float64
			}{{"MainHand", got.MainHand}, {"OffHand", got.OffHand}} {
				raw, inPoB := oracle.Output[hand.ns]
				if !inPoB {
					if hand.ours != nil {
						t.Errorf("%s: PoB-PoE2 has none, ours has %d entries", hand.ns, len(hand.ours))
					}
					continue
				}
				var want map[string]any
				if err := json.Unmarshal(raw, &want); err != nil {
					t.Fatal(err)
				}
				nonzero := 0
				for k, w := range want {
					f, ok := w.(float64)
					if !ok {
						continue
					}
					v, has := hand.ours[k]
					if !has {
						if f != 0 {
							t.Errorf("%s.%s: PoB-PoE2 %v, ours absent", hand.ns, k, f)
						}
						continue
					}
					if f != 0 {
						nonzero++
					}
					assertClose(t, name, hand.ns+"."+k, v, f)
				}
				if nonzero == 0 {
					t.Errorf("%s: no nonzero PoB-PoE2 value", hand.ns)
				}
			}
		})
	}
}

// The Calcs tab's own socket group, active skill and calculation mode give PoB-PoE2's CALCS calculation: every
// number output, the main skill and the {p:mod:n} entries.
func TestCalcsInputsMatchPoB(t *testing.T) {
	for _, f := range calcsInputFixtures {
		t.Run(f.name, func(t *testing.T) {
			oracle, calc := readCalcsInputFixture(t, f)
			got, err := calc.CalcsOutputs()
			if err != nil {
				t.Fatal(err)
			}
			if got.Inputs.SkillNumber != max(f.skillNumber, 1) {
				t.Errorf("skill number %d, set %d", got.Inputs.SkillNumber, f.skillNumber)
			}
			if name := got.Skills[got.MainActiveSkill-1]; name != oracle.MainSkill {
				t.Errorf("main skill %s, PoB-PoE2 %s", name, oracle.MainSkill)
			}
			if main := calc.env.Player.MainSkill.ActiveEffect.GrantedEffect.Name; main == oracle.MainSkill && f.buffMode == "" {
				t.Errorf("the sidebar's main skill %s is the Calcs tab's; the fixture tests no selection", main)
			}
			checkCalcsNumbers(t, f.name, oracle, got.Output)
			query := CalcsModQuery{Entries: map[string]CalcsModEntry{}}
			for k, e := range oracle.Entries {
				query.Entries[k] = CalcsModEntry{Cfg: e.Cfg, ModSource: e.ModSource, Enemy: e.Enemy, ModName: e.ModName, ModType: e.ModType, Actor: e.Actor}
			}
			mods, err := calc.CalcsMods(query)
			if err != nil {
				t.Fatal(err)
			}
			for k, want := range oracle.Mods {
				assertClose(t, f.name, k, mods[k], want)
			}
		})
	}
}

// checkCalcsNumbers grades every number output PoB-PoE2 writes; one it does not write must be 0 on ours.
func checkCalcsNumbers(t *testing.T, name string, oracle calcsOracle, got map[string]float64) {
	t.Helper()
	differ := 0
	for k, raw := range oracle.Output {
		var want float64
		if json.Unmarshal(raw, &want) != nil {
			continue
		}
		v, has := got[k]
		if !has && want != 0 {
			t.Errorf("%s: PoB-PoE2 %v, ours absent", k, want)
			continue
		}
		if !has {
			continue
		}
		if want != 0 {
			differ++
		}
		assertClose(t, name, k, v, want)
	}
	if differ == 0 {
		t.Error("no nonzero PoB-PoE2 number; the fixture tests nothing")
	}
}
