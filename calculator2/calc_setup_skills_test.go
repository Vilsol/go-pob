package calculator2

import (
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/poe2skills"
	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob2"
	"github.com/Vilsol/go-pob/utils"
)

func readTSV(t *testing.T, path string, cols int) [][]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rows [][]string
	for _, line := range strings.Split(strings.TrimRight(string(raw), "\n"), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != cols {
			t.Fatalf("%s: %q has %d columns, want %d", path, line, len(f), cols)
		}
		rows = append(rows, f)
	}
	if len(rows) == 0 {
		t.Fatalf("%s: no rows", path)
	}
	return rows
}

// TestGetGemStatRequirement grades the gem attribute requirement formula against PoB-PoE2's own.
func TestGetGemStatRequirement(t *testing.T) {
	for _, f := range readTSV(t, "testdata/gem_stat_requirement.tsv", 4) {
		level, _ := strconv.ParseFloat(f[0], 64)
		multi, _ := strconv.ParseFloat(f[1], 64)
		want, _ := strconv.ParseFloat(f[3], 64)
		if got := getGemStatRequirement(level, multi, f[2] == "1"); got != want {
			t.Errorf("getGemStatRequirement(%v, %v, %v) = %v, PoB %v", level, multi, f[2], got, want)
		}
	}
}

// TestDoesTypeExpressionMatch grades the skill type postfix evaluation against PoB-PoE2's own.
func TestDoesTypeExpressionMatch(t *testing.T) {
	set := func(s string) map[string]bool {
		if s == "" {
			return nil
		}
		m := map[string]bool{}
		for _, w := range strings.Fields(s) {
			m[w] = true
		}
		return m
	}
	for _, f := range readTSV(t, "testdata/type_expression_match.tsv", 4) {
		got := doesTypeExpressionMatch(strings.Fields(f[0]), set(f[1]), set(f[2]))
		if strconv.FormatBool(got) != f[3] {
			t.Errorf("doesTypeExpressionMatch(%q, %q, %q) = %v, PoB %s", f[0], f[1], f[2], got, f[3])
		}
	}
}

// CalcSetup.lua 564-568: a gemRequirements entry matches only a gem whose requirement exceeds it.
func TestApplyGemModsGemRequirements(t *testing.T) {
	prop := mod.GemProperty{Keyword: utils.Ptr("all"), Key: "level", Value: 2, KeyOfScaledMod: "value", GemRequirements: map[string]float64{"reqInt": 1}}
	for _, c := range []struct {
		reqInt float64
		want   int
	}{{100, 3}, {0, 1}, {1, 1}} {
		effect := &GemEffect{Level: 1, GemData: &poe2skills.Gem{Data: &poe2.Gem{Name: "Probe", ReqInt: c.reqInt}}}
		applyGemMods(effect, []any{prop})
		if effect.Level != c.want {
			t.Errorf("reqInt %v: level %d, want %d", c.reqInt, effect.Level, c.want)
		}
	}
}

// TestWeaponSetContext: a group only on the other weapon set is rebuilt in that set's context, so its
// skill reserves nothing; values from PoB-PoE2 headless.
func TestWeaponSetContext(t *testing.T) {
	raw, err := os.ReadFile("testdata/weapon_set_context.xml")
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	if err != nil {
		t.Fatal(err)
	}
	for stat, want := range map[string]float64{
		"Spirit": 350, "SpiritReserved": 0, "SpiritReservedPercent": 0, "SpiritUnreserved": 350,
		"SpiritUnreservedPercent": 100, "SpiritCancellableReservation": 100, "FireResist": -10, "ManaRegen": 11.9,
	} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9*math.Max(1, math.Abs(want)) {
			t.Errorf("%s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
	if env.WeaponSet != 1 || env.WeaponSetEnvs[2] == nil {
		t.Fatalf("weapon set %d, contexts %v", env.WeaponSet, env.WeaponSetEnvs)
	}
}

// TestActorWeaponData: actor.weaponData1/2's numbers for an equipped mace and an empty off hand;
// values from PoB-PoE2 headless.
func TestActorWeaponData(t *testing.T) {
	raw, err := os.ReadFile("testdata/warrior_items.xml")
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{
		"AttackRate": 1.62, "AttackSpeedInc": 12, "CritChance": 6.5, "ElementalDPS": 44.55, "LightningDPS": 44.55,
		"LightningMax": 50, "LightningMin": 5, "PhysicalDPS": 314.28, "PhysicalMax": 231, "PhysicalMin": 157,
		"TotalDPS": 358.83, "range": 13, "rangeBonus": 0,
	}
	got := env.Player.WeaponData1
	if len(got) != len(want) {
		t.Errorf("weaponData1 has %d numbers, PoB-PoE2 %d: %v", len(got), len(want), got)
	}
	for k, w := range want {
		if g, ok := got[k]; !ok || math.Abs(g-w) > 1e-9*math.Max(1, math.Abs(w)) {
			t.Errorf("weaponData1.%s: PoB-PoE2 %v, ours %v", k, w, g)
		}
	}
	if env.Player.WeaponData1Type != "One Hand Mace" || env.Player.WeaponData2 == nil || len(env.Player.WeaponData2) != 0 {
		t.Errorf("type %q, weaponData2 %v", env.Player.WeaponData1Type, env.Player.WeaponData2)
	}
}

// TestExplodeSources: each item that can explode gets one EnemyExplode gem in the Explode group;
// sources from PoB-PoE2 headless (its gem order follows pairs()).
func TestExplodeSources(t *testing.T) {
	raw, err := os.ReadFile("testdata/explode_sources.xml")
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"Item:1:Doom Knuckle, Wooden Club": true, "Item:2:Storm Loop, Iron Ring": true}
	var group *SocketGroup
	for _, g := range env.SocketGroupList {
		if g.Source == "Explode" {
			group = g
		}
	}
	if group == nil || len(group.GemList) != len(want) {
		t.Fatalf("explode group %+v, want %d gems", group, len(want))
	}
	for _, gem := range group.GemList {
		if key := explodeSourceKey(gem.ExplodeSource); !want[key] || gem.SkillID != "EnemyExplode" || gem.Level != 1 || !gem.Triggered || !gem.Enabled {
			t.Errorf("gem %q %s level %d triggered %v enabled %v", key, gem.SkillID, gem.Level, gem.Triggered, gem.Enabled)
		}
	}
}

// TestNodeExplodeSource: a passive whose mods can explode is an explode source (PoB-PoE2 lists node
// 47097 among env.explodeSources).
func TestNodeExplodeSource(t *testing.T) {
	n := loadTestTree(t).Nodes[47097]
	if n == nil {
		t.Fatal("no node 47097")
	}
	node := &tree2.SpecNode{Node: n, Dn: n.Name, Sd: n.Stats}
	env := &Environment{modSource: WithParser(ParseMod), nodeModLists: map[int]*moddb.ModList{}, AllocNodes: map[int]*tree2.SpecNode{}}
	buildModListForNode(env, node, 0, false)
	if len(env.explodeSources) != 1 || explodeSourceKey(env.explodeSources[0]) != "47097" {
		t.Errorf("explode sources %v", env.explodeSources)
	}
}

// Rows are SkillsTab:ApplySocketGroupWeaponSetValidity's three branches (SkillsTab.lua 1458-1470).
func TestApplySocketGroupWeaponSetValidity(t *testing.T) {
	cases := []struct {
		set1, set2, valid1, valid2 bool
		wantSet1, wantSet2         bool
		wantChanged                bool
	}{
		{true, true, false, true, false, true, true},
		{true, false, false, true, false, true, true},
		{false, true, false, true, false, true, false},
		{true, false, false, false, true, true, true},
		{true, true, false, false, true, true, false},
		{true, false, true, true, true, false, false},
		{false, false, true, true, true, true, true},
	}
	for _, c := range cases {
		g := &SocketGroup{Set1: c.set1, Set2: c.set2}
		changed := applySocketGroupWeaponSetValidity(g, c.valid1, c.valid2)
		if g.Set1 != c.wantSet1 || g.Set2 != c.wantSet2 || changed != c.wantChanged {
			t.Errorf("sets %v/%v valid %v/%v: got %v/%v changed %v, want %v/%v changed %v", c.set1, c.set2, c.valid1, c.valid2, g.Set1, g.Set2, changed, c.wantSet1, c.wantSet2, c.wantChanged)
		}
	}
}

// warrior_items.xml wields a Bandit Mace beside a Sceptre (no weapon data, so "Unarmed") and swaps to
// a Twisted Wand beside a Tower Shield. PoB-PoE2's CharacterMeleeSkills.lua maps One Hand Mace /
// Unarmed to SkillGemPlayerDefault1HMace (Melee1HMacePlayer) and has no Wand row.
//
//pob2:block:src/Modules/CalcSetup.lua:1845-1858
func TestDefaultAttackGroups(t *testing.T) {
	raw, err := os.ReadFile("testdata/warrior_items.xml")
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, g := range env.SocketGroupList {
		if g.Source == "Default Attack" {
			got = append(got, g.Slot+":"+g.GemList[0].SkillID+":"+strconv.FormatBool(g.GemList[0].FromItem))
		}
	}
	if strings.Join(got, ",") != "Weapon 1:Melee1HMacePlayer:true" {
		t.Fatalf("Default Attack groups %v", got)
	}
}

// The expected values are PoB-PoE2's for the same file, loaded headless: with no weapon the Default
// Attack group is removed, the main group index lands on a group assigned only to the other weapon set,
// and initEnv restarts in that set's context, so the main skill is one of the active skills.
func TestMainGroupWeaponSetRestartFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/main_group_other_weapon_set.xml")
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	for _, withItems := range []bool{false, true} {
		build, err := pob2.ParseBuild(raw)
		testza.AssertNoError(t, err)
		opts := Options{Tree: tree}
		if withItems {
			opts.Items = NewItemsTab(build, tree, nil)
		}
		env, err := BuildOutput(build, opts)
		testza.AssertNoError(t, err)

		names := make([]string, 0, len(env.Player.ActiveSkillList))
		inList := false
		for _, s := range env.Player.ActiveSkillList {
			names = append(names, s.ActiveEffect.GrantedEffect.Name)
			inList = inList || s == env.Player.MainSkill
		}
		testza.AssertEqual(t, []string{"Twister", "Skeletal Sniper Minion"}, names)
		testza.AssertEqual(t, "Twister", env.Player.MainSkill.ActiveEffect.GrantedEffect.Name)
		testza.AssertTrue(t, inList, "main skill is in the active skill list")
		testza.AssertEqual(t, 1, env.MainSocketGroup)
		testza.AssertEqual(t, 1, env.WeaponSet)
		testza.AssertEqual(t, 2, len(env.SocketGroupList))
	}
}

// The expected values are PoB-PoE2's for the same file, loaded headless and run through
// calcs.initEnv(build, "MAIN", { mainSocketGroup = 2 }): the restart check resolves the override's
// group, not build.mainSocketGroup, so it does not restart back onto group 1.
func TestMainGroupRestartReadsOverrideMainSocketGroup(t *testing.T) {
	raw, err := os.ReadFile("testdata/main_group_other_weapon_set.xml")
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	base, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)
	env := newWeaponSetEnv(base, "MAIN", calcOverride{MainSocketGroup: 2})
	initEnv(env)
	testza.AssertEqual(t, 2, env.MainSocketGroup)
	testza.AssertEqual(t, 1, env.WeaponSet)
	testza.AssertEqual(t, "Skeletal Sniper Minion", env.Player.MainSkill.ActiveEffect.GrantedEffect.Name)
	testza.AssertEqual(t, 1, build.Build.MainSocketGroup)
}
