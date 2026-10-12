package calculator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

// TestParseFormattedSourceModRefusesWhatItCannotHold pins trap 1 of the party-tab port.
func TestParseFormattedSourceModRefusesWhatItCannotHold(t *testing.T) {
	for _, line := range []string{
		"1|s|X|BASE|-|-|type=SkillType/skillType=5",
		"1|s|X|BASE|-|-|type=Condition/var=A/bogus=1",
		"1|s|X|BASE|-|-|type=GlobalEffect/effectType=Aura/div=x",
		"true|s|X|BASE|-|-|-",
		"1|s|X|LIST|-|-|-",
	} {
		if pm, err := parseFormattedSourceMod(line); err == nil {
			t.Errorf("%q: parsed to %v, want an error", line, pm)
		}
	}

	pm, err := parseFormattedSourceMod("-9|Skill:Enfeeble|Damage|MORE|Spell|Attack|type=GlobalEffect/effectType=Curse,type=Condition/neg=true/var=RareOrUnique")
	if err != nil {
		t.Fatal(err)
	}
	if pm.Name() != "Damage" || pm.Type() != mod.TypeMore || pm.Value().Float() != -9 ||
		pm.GetSource() != "Skill:Enfeeble" || pm.Flags() != mod.MFlagSpell || pm.KeywordFlags() != mod.KeywordFlagAttack {
		t.Errorf("mod fields: %s %s %v %s %v %v", pm.Name(), pm.Type(), pm.Value().Float(), pm.GetSource(), pm.Flags(), pm.KeywordFlags())
	}
	tags := pm.Tags()
	if len(tags) != 2 {
		t.Fatalf("tags: %d, want 2", len(tags))
	}
	if ge, ok := tags[0].(*mod.GlobalEffectTag); !ok || ge.GlobalEffectList[0] != "Curse" {
		t.Errorf("tag 1: %#v", tags[0])
	}
	if c, ok := tags[1].(*mod.ConditionTag); !ok || !c.Negative || c.VarList[0] != "RareOrUnique" {
		t.Errorf("tag 2: %#v", tags[1])
	}

	pm, err = parseFormattedSourceMod("1|s|X|BASE|Attack,Hit|-|-")
	if err != nil || pm.Flags() != 0 {
		t.Errorf("multi-flag name: flags %v err %v, want 0", pm.Flags(), err)
	}
}

func partyCarriers(t *testing.T) (string, []string) {
	t.Helper()
	dir := corpusDirForLoadedData(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var names []string
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "<ImportedBuffs") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatal("REFUSING: no corpus build carries <ImportedBuffs>; the fixture set is gone")
	}
	return dir, names
}

// TestPartyTabLoadsEveryCorpusCarrier asserts every party line in the corpus is one the port can represent.
func TestPartyTabLoadsEveryCorpusCarrier(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	dir, names := partyCarriers(t)
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(build.Party.ImportedBuffs) == 0 {
			t.Errorf("%s: <ImportedBuffs> present in the XML but not read into build.Party", name)
		}
		if tab := loadPartyTab(build.Party); len(tab.skipped) != 0 {
			t.Errorf("%s: skipped lines PoB reads: %v", name, tab.skipped)
		}
	}
}

// TestPartyEnemyModsReachEnemyDB pins CalcSetup.lua:583 against PoB's own dumps.
func TestPartyEnemyModsReachEnemyDB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	dumps := dumpsDirForLoadedData(t)
	dir, names := partyCarriers(t)

	checked := 0
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dumps, strings.TrimSuffix(name, ".xml")+".json"))
		if err != nil {
			t.Logf("%s: no dump (%v)", name, err)
			continue
		}
		var d struct {
			EnemyModDB map[string][]struct {
				Source string `json:"source"`
				Type   string `json:"type"`
				Value  any    `json:"value"`
				Flags  string `json:"flags"`
				Tags   []string
			} `json:"enemyModDB"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var want []string
		for modName, mods := range d.EnemyModDB {
			for _, m := range mods {
				if strings.HasPrefix(m.Source, "Party") {
					want = append(want, fmt.Sprintf("%s|%s|%v|%s|%s|%s", modName, m.Type, m.Value, m.Source, m.Flags, strings.Join(m.Tags, ",")))
				}
			}
		}

		xml, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			t.Fatal(err)
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.EnemyModDB == nil {
			t.Fatalf("%s: no environment", name)
		}
		var got []string
		for modName, mods := range env.EnemyModDB.Mods {
			for _, m := range mods {
				if !strings.HasPrefix(string(m.GetSource()), "Party") {
					continue
				}
				var value any = m.Value().Float()
				if m.Type() == mod.TypeFlag {
					value = m.Value().Flag()
				}
				flags := ""
				if m.Flags() == mod.MFlagAttack {
					flags = "Attack"
				} else if m.Flags() == mod.MFlagSpell {
					flags = "Spell"
				}
				var tagTypes []string
				for _, tag := range m.Tags() {
					tagTypes = append(tagTypes, string(tag.Type()))
				}
				got = append(got, fmt.Sprintf("%s|%s|%v|%s|%s|%s", modName, m.Type(), value, m.GetSource(), flags, strings.Join(tagTypes, ",")))
			}
		}
		sort.Strings(want)
		sort.Strings(got)
		if strings.Join(want, "\n") != strings.Join(got, "\n") {
			t.Errorf("%s: party enemy mods differ\n PoB:\n  %s\n ours:\n  %s", name, strings.Join(want, "\n  "), strings.Join(got, "\n  "))
		}
		if len(want) > 0 {
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("REFUSING: no carrier's dump holds a Party-sourced enemy mod; nothing was compared")
	}
	t.Logf("compared %d carriers with party enemy mods", checked)
}

// TestPartyBuffsReachTheOracle pins the party auras.
func TestPartyBuffsReachTheOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	for _, c := range []struct{ build, stat string }{
		{"1631-luminary-ghostly-artillery.xml", "FireResist"},
		{"0758-necromancer-vaal-summon-skeletons-of-archers.xml", "CullPercent"},
		{"1051-luminary-explosive-concoction.xml", "Evasion"},
		{"0581-luminary-unknown.xml", "BleedChance"},
	} {
		rec, ok := oracle[c.build]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle", c.build)
		}
		exp, ok := rec.Output[c.stat]
		if !ok || exp == 0 {
			t.Fatalf("REFUSING: the oracle has no nonzero %s for %s", c.stat, c.build)
		}
		build := loadCorpusBuild(t, c.build)
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil {
			t.Fatalf("%s: no environment", c.build)
		}
		got := env.Player.Output[c.stat]
		if math.Abs(got-exp) > 1e-3*math.Abs(exp) {
			t.Errorf("%s: %s = %v, PoB %v", c.build, c.stat, got, exp)
		}
	}
}

// TestPartyAllyStrongerThanIsStrict pins the comparison CalcPerform.lua:2413 / :2431 / :2804 / :2822 negate.
func TestPartyAllyStrongerThanIsStrict(t *testing.T) {
	tab := newPartyTab()
	tab.parseBuffs(tab.actor.Aura, "Wrath\n150\n1|Skill:Wrath|LightningMin|BASE|-|-|type=GlobalEffect/effectType=Aura\n---\nVaal Grace\n200\n1|Skill:VaalGrace|Evasion|BASE|-|-|type=GlobalEffect/effectType=Aura\n---", "Aura")
	if len(tab.skipped) != 0 {
		t.Fatal(tab.skipped)
	}
	l := tab.actor.Aura
	if !l.strongerThan("Aura", "Wrath", 1.49) {
		t.Error("ally 150% vs own 1.49: ally should win")
	}
	if l.strongerThan("Aura", "Wrath", 1.5) {
		t.Error("ally 150% vs own 1.5: a tie keeps the player's")
	}
	if l.strongerThan("Aura", "Vaal Grace", 0) || l.strongerThan("AuraDebuff", "Wrath", 0) {
		t.Error("a Vaal aura or another type must not compare")
	}
	if l.byType["Aura"].vaal["Vaal Grace"] == nil {
		t.Error("Vaal Grace should nest under the Vaal key (PartyTab.lua:839-842)")
	}
}

// TestPartyTabSkipsALineItCannotRead.
func TestPartyTabSkipsALineItCannotRead(t *testing.T) {
	party := pob.Party{ImportedBuffs: []pob.PartyBuffs{{
		Name: "EnemyMods",
		Text: "-10|Base|FireResist|BASE|None|None|-\n" +
			"5|Base|ColdResist|BASE|None|None|notatag\n" +
			"-20|Base|LightningResist|BASE|None|None|-",
	}}}
	tab := loadPartyTab(party)
	for _, name := range []string{"FireResist", "LightningResist"} {
		if len(tab.enemyModList.Mods()) == 0 || tab.enemyModList.Sum(mod.TypeBase, nil, name) == 0 {
			t.Errorf("%s from a readable line did not land", name)
		}
	}
	if got := tab.enemyModList.Sum(mod.TypeBase, nil, "ColdResist"); got != 0 {
		t.Errorf("ColdResist from the unreadable line landed (%v) — it must be skipped whole", got)
	}
	if len(tab.skipped) != 1 {
		t.Errorf("skipped = %v, want exactly the one unreadable line recorded", tab.skipped)
	}
}

// TestDisablePartyEffectsReachesTheCalculation: Disable Party Effects takes the party out of the calculation.
func TestDisablePartyEffectsReachesTheCalculation(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	const name, stat = "1631-luminary-ghostly-artillery.xml", "FireResist"
	exp := oracle[name].Output[stat]
	build := loadCorpusBuild(t, name)
	out := func(b *pob.PathOfBuilding) float64 {
		env := BuildOutputWithCache(&Calculator{PoB: b}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil {
			t.Fatalf("%s: no environment", name)
		}
		return env.Player.Output[stat]
	}
	if got := out(build.WithoutPartyEffects()); math.Abs(got-exp) <= 1e-3*math.Abs(exp) {
		t.Errorf("%s with the party disabled = %v, still PoB's party value %v", stat, got, exp)
	}
	build.DisablePartyEffects()
	build.RebuildParty()
	if got := out(build); math.Abs(got-exp) > 1e-3*math.Abs(exp) {
		t.Errorf("%s after Rebuild All = %v, PoB %v", stat, got, exp)
	}
}
