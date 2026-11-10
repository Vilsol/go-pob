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

	"github.com/Vilsol/go-pob/calculator2"
)

// skillSelectOracle is RefreshSkillSelectControls' state as oracle/poe2_calcs.lua records it; Lua's nil is false.
type skillSelectOracle struct {
	Skills          []string `json:"skills"`
	StatSets        []string `json:"statSets"`
	StageCount      string   `json:"stageCount"`
	MineCount       string   `json:"mineCount"`
	MainActiveSkill int      `json:"mainActiveSkill"`
	StatSet         int      `json:"statSet"`
	SkillShown      bool     `json:"skillShown"`
	StatSetShown    bool     `json:"statSetShown"`
	StatSetEnabled  bool     `json:"statSetEnabled"`
	PartShown       bool     `json:"partShown"`
	StageShown      bool     `json:"stageShown"`
	MineShown       bool     `json:"mineShown"`
	HaveMinion      bool     `json:"haveMinion"`

	Minions                   []string `json:"minions"`
	MinionSkills              []string `json:"minionSkills"`
	MinionSkillStatSets       []string `json:"minionSkillStatSets"`
	Minion                    int      `json:"minion"`
	MinionSkill               int      `json:"minionSkill"`
	MinionSkillStatSet        int      `json:"minionSkillStatSet"`
	MinionShown               bool     `json:"minionShown"`
	MinionEnabled             bool     `json:"minionEnabled"`
	MinionLibraryShown        bool     `json:"minionLibraryShown"`
	BeastLibraryShown         bool     `json:"beastLibraryShown"`
	MinionSkillShown          bool     `json:"minionSkillShown"`
	MinionSkillEnabled        bool     `json:"minionSkillEnabled"`
	MinionSkillStatSetShown   bool     `json:"minionSkillStatSetShown"`
	MinionSkillStatSetEnabled bool     `json:"minionSkillStatSetEnabled"`
}

// shownOnly drops what a hidden dropdown keeps from an earlier refresh.
func (o skillSelectOracle) shownOnly() skillSelectOracle {
	if !o.MinionShown {
		o.MinionEnabled = false
	}
	if !o.MinionSkillShown {
		o.MinionSkillEnabled = false
	}
	if !o.MinionSkillStatSetShown {
		o.MinionSkillStatSetEnabled = false
	}
	return o
}

type calcsOracle struct {
	MainSkill       string                     `json:"mainSkill"`
	Controls        skillSelectOracle          `json:"controls"`
	SidebarControls skillSelectOracle          `json:"sidebarControls"`
	Output          map[string]json.RawMessage `json:"output"`
	Mods            map[string]float64         `json:"mods"`
	Entries         map[string]struct {
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
	statSet         int
	stageCount      int
	showMinion      bool
	// minion, minionSkill and minionSkillStatSet are picked in that order, each after the calculation that listed it.
	minion             int
	minionSkill        int
	minionSkillStatSet int
}

func (f calcsInputFixture) minionPicks() bool {
	return f.minion != 0 || f.minionSkill != 0 || f.minionSkillStatSet != 0
}

var calcsInputFixtures = []calcsInputFixture{
	{name: "sorceress_three_spells_arc_unbuffed", xml: "sorceress_three_spells", skillNumber: 2, buffMode: "UNBUFFED"},
	{name: "sorceress_three_spells_fireball_combat", xml: "sorceress_three_spells", skillNumber: 1, buffMode: "COMBAT", mainActiveSkill: 2},
	{name: "warrior_dual_mace_buffed", xml: "warrior_dual_mace", buffMode: "BUFFED"},
	{name: "sorceress_three_spells_fireball_explosion", xml: "sorceress_three_spells", skillNumber: 1, mainActiveSkill: 2, statSet: 2},
	{name: "sorceress_flameblast_stages", xml: "sorceress_flameblast", stageCount: 5},
	{name: "sorceress_raise_zombie_minion", xml: "sorceress_raise_zombie", showMinion: true},
	{name: "sorceress_skeletal_brute_minion_skill", xml: "sorceress_skeletal_brute", minionSkill: 2, minionSkillStatSet: 2},
	{name: "sorceress_spectres_second_minion", xml: "sorceress_spectres", showMinion: true, minion: 2},
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
	// the stat set and stage controls write the skill the calculation above listed
	if f.statSet == 0 && f.stageCount == 0 && !f.showMinion && !f.minionPicks() {
		return oracle, calc
	}
	if f.statSet != 0 {
		if err := calc.SetSkillStatSet(true, f.statSet); err != nil {
			t.Fatal(err)
		}
	}
	if f.stageCount != 0 {
		if err := calc.SetSkillStageCount(true, &f.stageCount); err != nil {
			t.Fatal(err)
		}
	}
	if f.showMinion {
		calc.SetCalcsShowMinion(true)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	for _, pick := range []struct {
		index int
		set   func(bool, int) error
	}{{f.minion, calc.SetSkillMinion}, {f.minionSkill, calc.SetSkillMinionSkill}, {f.minionSkillStatSet, calc.SetSkillMinionSkillStatSet}} {
		if pick.index == 0 {
			continue
		}
		if err := pick.set(true, pick.index); err != nil {
			t.Fatal(err)
		}
		if _, err := calc.Calculate(); err != nil {
			t.Fatal(err)
		}
	}
	return oracle, calc
}

func checkSkillSelect(t *testing.T, what string, want skillSelectOracle, got calculator2.SkillSelectControls) {
	t.Helper()
	ours := skillSelectOracle{
		Skills: got.Skills, StatSets: got.StatSets, StageCount: got.StageCount, MineCount: got.MineCount,
		MainActiveSkill: got.MainActiveSkill, StatSet: got.StatSet, SkillShown: got.SkillShown, StatSetShown: got.StatSetShown,
		StatSetEnabled: got.StatSetEnabled, PartShown: got.PartShown, StageShown: got.StageShown, MineShown: got.MineShown,
		HaveMinion: want.HaveMinion,

		Minions: got.Minions, MinionSkills: got.MinionSkills, MinionSkillStatSets: got.MinionSkillStatSets,
		Minion: got.Minion, MinionSkill: got.MinionSkill, MinionSkillStatSet: got.MinionSkillStatSet,
		MinionShown: got.MinionShown, MinionEnabled: got.MinionEnabled, MinionLibraryShown: got.MinionLibraryShown,
		BeastLibraryShown: got.BeastLibraryShown, MinionSkillShown: got.MinionSkillShown,
		MinionSkillEnabled: got.MinionSkillEnabled, MinionSkillStatSetShown: got.MinionSkillStatSetShown,
		MinionSkillStatSetEnabled: got.MinionSkillStatSetEnabled,
	}
	ours, want = ours.shownOnly(), want.shownOnly()
	if !reflect.DeepEqual(ours, want) {
		t.Errorf("%s controls:\n PoB-PoE2 %+v\n ours     %+v", what, want, ours)
	}
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
			checkSkillSelect(t, "Calcs", oracle.Controls, got.Controls)
			if got.HaveMinion != oracle.Controls.HaveMinion {
				t.Errorf("haveMinion %v, PoB-PoE2 %v", got.HaveMinion, oracle.Controls.HaveMinion)
			}
			out, err := calc.Calculate()
			if err != nil {
				t.Fatal(err)
			}
			checkSkillSelect(t, "sidebar", oracle.SidebarControls, out.MainSkill)
			calcsEnv, err := calc.calcs()
			if err != nil {
				t.Fatal(err)
			}
			if name := calc.calcsActor(calcsEnv).MainSkill.ActiveEffect.GrantedEffect.Name; name != oracle.MainSkill {
				t.Errorf("main skill %s, PoB-PoE2 %s", name, oracle.MainSkill)
			}
			if main := calc.env.Player.MainSkill.ActiveEffect.GrantedEffect.Name; main == oracle.MainSkill && f.buffMode == "" && f.statSet == 0 && f.stageCount == 0 && !f.minionPicks() {
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

// A section's collapsed state round-trips through the saved build as CalcsTab:Save and Load write and read it.
func TestCalcsSectionCollapsedRoundTrip(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join("testdata", "sorceress_spark.xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, saved := calc.CalcsSectionCollapsed("HitDamage", "SkillHitDamage"); saved {
		t.Fatal("a build without Section elements has a saved state")
	}
	calc.SetCalcsSectionCollapsed("HitDamage", "SkillHitDamage", true)
	calc.SetCalcsSectionCollapsed("Life", "Life", true)
	calc.SetCalcsSectionCollapsed("Life", "Life", false)
	out, err := calc.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	again, err := ParseBuild(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		id, sub   string
		collapsed bool
	}{{"HitDamage", "SkillHitDamage", true}, {"Life", "Life", false}} {
		if got, saved := again.CalcsSectionCollapsed(c.id, c.sub); !saved || got != c.collapsed {
			t.Errorf("%s/%s: saved %v collapsed %v, want collapsed %v", c.id, c.sub, saved, got, c.collapsed)
		}
	}
	if n := strings.Count(out, `<Section `); n != 2 {
		t.Errorf("%d Section elements, want 2:\n%s", n, out)
	}
}

// CompareTab.lua:556-573 (PoB-PoE2): the Compare tab's Calcs header Minion dropdown moves only the Calcs tab's minion;
// the sidebar keeps its own, and the gem keeps its name.
func TestSetCompareCalcsMinionWritesOnlyCalcs(t *testing.T) {
	_, calc := readCalcsOracle(t, "sorceress_spectres_second_minion", "sorceress_spectres")
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	if err := calc.SetCompareCalcsMinion(2); err != nil {
		t.Fatal(err)
	}
	out, err := calc.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	page, err := calc.CalcsOutputs()
	if err != nil {
		t.Fatal(err)
	}
	if page.Controls.Minion != 2 || out.MainSkill.Minion != 1 {
		t.Errorf("Calcs minion %d, sidebar minion %d; want 2 and 1", page.Controls.Minion, out.MainSkill.Minion)
	}
	if got := out.MainSkill.Skills; len(got) != 1 || got[0] != "Spectre: Vaal Skeletal Priest" {
		t.Errorf("sidebar skills %v; the gem was renamed", got)
	}
}
