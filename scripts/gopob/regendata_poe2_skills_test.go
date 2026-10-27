package main

import (
	"strings"
	"testing"
)

func TestPoE2SkillsEmitOverlay(t *testing.T) {
	mod := `{"name":"Damage","type":"MORE","value":null,"flags":0,"keywordFlags":0,"tags":[{"type":"SkillType","skillType":"Attack"}]}`
	out, err := p2sOverlay([]byte(`[
{"id":"A","hidden":true,"minionList":["X","Y"],"statSets":[{"id":"S1","baseFlags":[],"baseMods":[` + mod + `]},{"statMap":[
 {"stat":"a","items":[` + mod + `,{"group":[` + mod + `],"value":null,"mult":null,"div":2,"base":null}],"value":0,"mult":null,"div":null,"base":null,"skillFlag":"f"},
 {"stat":"b","items":[],"value":null,"mult":null,"div":null,"base":null,"skillFlag":null}]}]},
{"id":"B","preDamageFunc":true,"noTemplate":true,"name":"Bee","baseTypeName":"Bee","color":4,"skillTypes":["Area","Damage"],"castTime":0,
 "cannotBeSupported":true,"hasQualityStats":true,"levels":[{"level":1,"extra":{"levelRequirement":1,"baseMultiplier":1}}],
 "statSets":[{"label":"L","incrementalEffectiveness":0,"statDescriptionScope":"s","constantStats":[{"stat":"c","value":4}],"stats":[],"levels":[1],"baseFlags":[]}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`{ID: "A", Hidden: true, MinionList: []string{"X", "Y"}, StatSets: []StatSetOverlay{{ID: "S1", BaseFlags: []string{}, HasBaseFlags: true, BaseMods: []string{`,
		`{Stat: "a", Value: 0, HasValue: true, SkillFlag: "f", Items: []StatMapItem{{Mod: `,
		`Div: 2, HasDiv: true}}}, {Stat: "b"}}}}}`,
		`{ID: "B", PreDamageFunc: true, NoTemplate: true, Name: "Bee", BaseTypeName: "Bee", HasBaseTypeName: true, SkillTypes: []string{"Area", "Damage"}, ` +
			`Levels: []LevelOverlay{{Level: 1, Extra: map[string]float64{"baseMultiplier": 1, "levelRequirement": 1}}}, Color: 4, CastTime: 0, HasCastTime: true, ` +
			`CannotBeSupported: true, HasQualityStats: true, StatSets: []StatSetOverlay{{Label: "L", HasLabel: true, StatDescriptionScope: "s", HasStatDescriptionScope: true, ` +
			`ConstantStats: []ConstantStatOverlay{{Stat: "c", Value: 4}}, HasConstantStats: true, Stats: []string{}, HasStats: true, Levels: []int{1}, ` +
			`IncrementalEffectiveness: 0, HasIncrementalEffectiveness: true, BaseFlags: []string{}, HasBaseFlags: true}}},`,
		`granted effects sorted by id; the 1 with`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in:\n%s", want, out)
		}
	}
	for _, tc := range []struct{ name, body, want string }{
		{"unknown field", `[{"id":"A","x":1}]`, "schema"},
		{"unsorted", `[{"id":"B"},{"id":"A"}]`, "not sorted"},
		{"duplicate", `[{"id":"A"},{"id":"A"}]`, "twice"},
		{"empty", `[]`, "no granted effects"},
		{"game fields on a templated effect", `[{"id":"A","name":"A"}]`, "has a template but carries game fields"},
		{"no template without a name", `[{"id":"A","noTemplate":true}]`, "no template and no name"},
		{"fractional colour", `[{"id":"A","noTemplate":true,"name":"A","color":1.5}]`, "not an integer"},
		{"group of non-mods", `[{"id":"A","statSets":[{"statMap":[{"stat":"a","items":[{"group":[1]}]}]}]}]`, "not a mod"},
		{"unsorted stat map", `[{"id":"A","statSets":[{"statMap":[{"stat":"b","items":[]},{"stat":"a","items":[]}]}]}]`, "not sorted by stat"},
		{"unknown group field", `[{"id":"A","statSets":[{"statMap":[{"stat":"a","items":[{"group":[],"scalar":1}]}]}]}]`, "schema"},
	} {
		if _, err := p2sOverlay([]byte(tc.body)); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want an error containing %q", tc.name, err, tc.want)
		}
	}
}

func TestPoE2SkillsEmitGlobals(t *testing.T) {
	out, err := p2sGlobals([]byte(`{"skillType":[{"name":"Attack","value":1},{"name":"Spell","value":2}],` +
		`"modFlag":[{"name":"WeaponMask","value":66035056640}],"keywordFlag":[{"name":"Aura","value":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"Attack": 1,`, `"WeaponMask": 0x0000000F5FFF0000,`, `"Aura": 0x0000000000000001,`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in:\n%s", want, out)
		}
	}
	for _, tc := range []struct{ name, body, want string }{
		{"fraction", `{"skillType":[{"name":"A","value":1.5}],"modFlag":[{"name":"A","value":1}],"keywordFlag":[{"name":"A","value":1}]}`, "integer"},
		{"empty", `{"skillType":[],"modFlag":[{"name":"A","value":1}],"keywordFlag":[{"name":"A","value":1}]}`, "empty"},
		{"unsorted", `{"skillType":[{"name":"B","value":1},{"name":"A","value":2}],"modFlag":[{"name":"A","value":1}],"keywordFlag":[{"name":"A","value":1}]}`, "sorted"},
	} {
		if _, err := p2sGlobals([]byte(tc.body)); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want an error containing %q", tc.name, err, tc.want)
		}
	}
}

func TestPoE2SkillsEmitStatMap(t *testing.T) {
	out, err := p2sStatMap([]byte(`[{"stat":"x","items":[{"name":"SkillData","type":"LIST","value":{"key":"duration"},"flags":0,"keywordFlags":0,"tags":[]}],"value":null,"mult":null,"div":1000,"base":null,"skillFlag":null}]`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `{Stat: "x", Div: 1000, HasDiv: true, Items: []StatMapItem{{Mod: "{\"name\":\"SkillData\"`) {
		t.Errorf("unexpected output:\n%s", out)
	}
	if _, err := p2sStatMap([]byte(`[]`)); err == nil {
		t.Error("an empty stat map was accepted")
	}
}
