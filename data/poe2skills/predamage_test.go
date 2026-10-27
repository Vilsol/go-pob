package poe2skills

import (
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data/poe2"
)

type fakeSkill struct {
	sums          map[string]float64
	skillData     map[string]float64
	explodeSource *[3]string
	list          []ExplodeMod
	tab           []TabulatedExplodeMod
	newMods       []string
	statSetIndex  int
}

func (f *fakeSkill) SkillData(key string) (float64, bool) {
	v, ok := f.skillData[key]
	return v, ok
}

func (f *fakeSkill) SetSkillData(key string, v float64, ok bool) {
	if ok {
		f.skillData[key] = v
	} else {
		delete(f.skillData, key)
	}
}

func (f *fakeSkill) Sum(modType, name string) float64 { return f.sums[modType+":"+name] }

func (f *fakeSkill) NewMod(name, modType string, value any, source string) {
	v := ""
	switch x := value.(type) {
	case float64:
		v = caseNum(x)
	case bool:
		v = strconv.FormatBool(x)
	}
	if source == "" {
		source = "nil"
	}
	f.newMods = append(f.newMods, name+"|"+modType+"|"+v+"|"+source)
}

func (f *fakeSkill) StatSetIndex() int                          { return f.statSetIndex }
func (f *fakeSkill) ExplodeMods() []ExplodeMod                  { return f.list }
func (f *fakeSkill) TabulateExplodeMods() []TabulatedExplodeMod { return f.tab }
func (f *fakeSkill) ExplodeSource() (string, bool, string) {
	if f.explodeSource == nil {
		return "", false, ""
	}
	return f.explodeSource[0], f.explodeSource[0] != "", f.explodeSource[1]
}

type fakeOutput map[string]float64

func (o fakeOutput) Get(key string) (float64, bool) {
	v, ok := o[key]
	return v, ok
}

func (o fakeOutput) Set(key string, v float64) { o[key] = v }

func caseNum(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "+Inf"
	case math.IsInf(v, -1):
		return "-Inf"
	case math.IsNaN(v):
		return "NaN"
	}
	return strconv.FormatFloat(v, 'g', 17, 64)
}

type preDamageCase struct {
	sums, skillData, output map[string]float64
	explodeSource           *[3]string
	name, id                string
	list                    []ExplodeMod
	tab                     []TabulatedExplodeMod
	statSetIndex            int
}

// The same cases as testdata/predamage_cases.lua.
var preDamageCases = []preDamageCase{
	{name: "DetonatingArrow", id: "DetonatingArrowPlayer", sums: map[string]float64{"BASE:Multiplier:DetonatingArrowStage": 4}},
	{name: "Snipe", id: "SnipePlayer", skillData: map[string]float64{"channelPercentOfAttackTime": 0.35, "hitTimeMultiplier": 9}},
	{name: "SnipeNil", id: "SnipePlayer", skillData: map[string]float64{"hitTimeMultiplier": 9, "channelTimeMultiplier": 9}},
	{name: "Bonestorm", id: "BonestormPlayer", sums: map[string]float64{"BASE:Multiplier:BonestormStage": 7}, output: map[string]float64{"ProjectileCount": 3}},
	{name: "BonestormZero", id: "BonestormPlayer", sums: map[string]float64{"BASE:Multiplier:BonestormStage": 7}, output: map[string]float64{"ProjectileCount": 0}},
	{name: "BonestormNil", id: "BonestormPlayer", sums: map[string]float64{"BASE:Multiplier:BonestormStage": 7}},
	{name: "Flameblast", id: "FlameblastPlayer", sums: map[string]float64{"BASE:Multiplier:FlameblastStage": 10}},
	{name: "FlickerStrike", id: "FlickerStrikePlayer", sums: map[string]float64{"BASE:AdditionalFlickersPerPowerCharge": 2}},
	{name: "GatheringStorm", id: "GatheringStormPlayer", skillData: map[string]float64{"channelPercentOfAttackTime": 1.5}},
	{name: "Incinerate", id: "IncineratePlayer", sums: map[string]float64{"BASE:Multiplier:IncinerateStage": 3}},
	{name: "PlasmaBlast", id: "PlasmaBlastPlayer", skillData: map[string]float64{"channelTimeMultiplier": 4}},
	{name: "PerfectStrike", id: "PerfectStrikePlayer", skillData: map[string]float64{"channelPercentOfAttackTime": 0.8}},
	{name: "ShieldCharge", id: "ShieldChargePlayer"},
	{name: "SuperchargedSlam", id: "SuperchargedSlamPlayer", sums: map[string]float64{"BASE:Multiplier:SuperchargedSlamStage": 5}},
	{name: "MinionInstability", id: "MinionInstability", skillData: map[string]float64{"selfFireExplosionLifeMultiplier": 0.01}, output: map[string]float64{"Life": 1234}},
	{name: "MinionInstabilityNil", id: "MinionInstability", skillData: map[string]float64{"selfFireExplosionLifeMultiplier": 0.01}},
	{name: "TemperWeapon", id: "TemperWeaponPlayer", sums: map[string]float64{"BASE:Multiplier:TemperWeaponStage": 6}},
	{name: "ChaosFrogInactive", id: "ChaosFrogExplosionPlayer"},
	{name: "ChaosFrog", id: "ChaosFrogExplosionPlayer", sums: map[string]float64{"BASE:ToadSpawnInterval": 750}},
	{name: "EnemyExplodeSource", id: "EnemyExplode", statSetIndex: 1, explodeSource: &[3]string{"Item:7:Ring", ""}, tab: []TabulatedExplodeMod{
		{Source: "Item:7:Ring", Value: ExplodeMod{Type: "RandomElement", Amount: 9, Value: 25}},
		{Source: "Tree:55", Value: ExplodeMod{Type: "Fire", Amount: 4, Value: 100}},
		{Source: "Item:7:Ring", Value: ExplodeMod{Type: "Cold", Amount: 6, Value: 40}},
	}},
	{name: "EnemyExplodeTree", id: "EnemyExplode", statSetIndex: 2, explodeSource: &[3]string{"", "55"}, tab: []TabulatedExplodeMod{
		{Source: "Tree:55", Value: ExplodeMod{Type: "Fire", Amount: 4, Value: 30}},
	}},
	{name: "EnemyExplodeAll", id: "EnemyExplode", statSetIndex: 3, list: []ExplodeMod{
		{Type: "Physical", Amount: 10, Value: 30},
		{Type: "Physical", Amount: 10, Value: 20},
		{Type: "Fire", Amount: 5, Value: 50},
	}},
	{name: "EnemyExplodeRandom", id: "EnemyExplode", statSetIndex: 3, list: []ExplodeMod{
		{Type: "RandomElement", Amount: 12, Value: 60},
	}},
}

func sortedFloatKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func runPreDamageCase(t *testing.T, c preDamageCase) []string {
	fn := preDamageFuncs[c.id]
	if fn == nil {
		t.Fatalf("%s has no preDamageFunc", c.id)
	}
	skill := &fakeSkill{sums: c.sums, skillData: map[string]float64{}, explodeSource: c.explodeSource, list: c.list, tab: c.tab, statSetIndex: c.statSetIndex}
	for k, v := range c.skillData {
		skill.skillData[k] = v
	}
	output := fakeOutput{}
	for k, v := range c.output {
		output[k] = v
	}
	var rows []string
	if err := fn(skill, output); err != nil {
		rows = append(rows, c.name+"\terror\tnil arithmetic")
	}
	for _, k := range sortedFloatKeys(skill.skillData) {
		rows = append(rows, c.name+"\tskillData."+k+"\t"+caseNum(skill.skillData[k]))
	}
	for _, k := range sortedFloatKeys(output) {
		rows = append(rows, c.name+"\toutput."+k+"\t"+caseNum(output[k]))
	}
	for i, m := range skill.newMods {
		rows = append(rows, c.name+"\tnewMod."+strconv.Itoa(i+1)+"\t"+m)
	}
	return rows
}

// TestPreDamageFuncs grades each port against PoB-PoE2's own function over the same cases.
func TestPreDamageFuncs(t *testing.T) {
	b, err := os.ReadFile("testdata/predamage_expected.tsv")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Split(strings.TrimSpace(string(b)), "\n")
	got := make([]string, 0, len(want))
	for _, c := range preDamageCases {
		got = append(got, runPreDamageCase(t, c)...)
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("preDamageFunc results differ from PoB-PoE2's\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// Every effect the overlay marks with a preDamageFunc has a port, and no port names another.
func TestPreDamageFuncsCoverOverlay(t *testing.T) {
	marked := map[string]bool{}
	for _, o := range poe2.SkillOverlays {
		if o.PreDamageFunc {
			marked[o.ID] = true
			if preDamageFuncs[o.ID] == nil {
				t.Errorf("%s has a preDamageFunc in PoB and no port", o.ID)
			}
		}
	}
	for id := range preDamageFuncs {
		if !marked[id] {
			t.Errorf("%s has a port but no preDamageFunc in PoB", id)
		}
	}
	covered := map[string]bool{}
	for _, c := range preDamageCases {
		covered[c.id] = true
	}
	for id := range preDamageFuncs {
		if !covered[id] {
			t.Errorf("%s has no case", id)
		}
	}
}

// The expected file is what PoB-PoE2 prints today.
func TestPreDamageExpectedIsCurrent(t *testing.T) {
	var rows []string
	for _, l := range strings.Split(string(runPoBLua(t, "predamage_cases.lua")), "\n") {
		if strings.Count(l, "\t") == 2 {
			rows = append(rows, l)
		}
	}
	b, err := os.ReadFile("testdata/predamage_expected.tsv")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(rows, "\n") != strings.TrimSpace(string(b)) {
		t.Errorf("testdata/predamage_expected.tsv is stale; PoB-PoE2 prints:\n%s", strings.Join(rows, "\n"))
	}
}
