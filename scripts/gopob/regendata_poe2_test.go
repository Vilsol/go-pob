package main

import (
	"strings"
	"testing"
)

func TestRegenDataGameFlag(t *testing.T) {
	for _, tc := range []struct {
		argv []string
		game string
		ok   bool
	}{
		{[]string{"--game", "poe2", "--check"}, "poe2", true},
		{[]string{"--game", "poe1"}, "poe1", true},
		{[]string{}, "", true},
		{[]string{"--game", "poe3"}, "", false},
		{[]string{"--game"}, "", false},
	} {
		t.Run(strings.Join(tc.argv, " "), func(t *testing.T) {
			var stderr strings.Builder
			defer rdStderrForTest(&stderr)()
			r := &rdRun{}
			if _, ok := rdParseArgs(r, tc.argv, ""); ok != tc.ok || r.game != tc.game {
				t.Fatalf("regen-data %v: ok %v game %q, want ok %v game %q", tc.argv, ok, r.game, tc.ok, tc.game)
			}
		})
	}
}

func TestPoE2GeneratorsAreOwnedAndClassified(t *testing.T) {
	poe1 := map[string]bool{}
	for _, g := range rdGenerators {
		poe1[g.name], poe1[g.out] = true, true
	}
	if len(rdGeneratorsPoE2) == 0 {
		t.Fatal("no PoE2 generators")
	}
	for _, g := range rdGeneratorsPoE2 {
		if poe1[g.name] || poe1[g.out] {
			t.Errorf("%s (%s) collides with a PoE1 generator", g.name, g.out)
		}
		if !strings.HasPrefix(g.out, "data/poe2/") {
			t.Errorf("%s writes %s, outside the PoE2 data package", g.name, g.out)
		}
		switch g.source {
		case rdGameDataFromPoB, rdPoBAuthored:
		case rdGameData:
			if g.lua != "" || g.dump == "" {
				t.Errorf("%s is game data from go-pob-data but has lua %q, dump %q", g.name, g.lua, g.dump)
			}
		case rdGameDataOverlay:
			if g.lua == "" || g.dump == "" {
				t.Errorf("%s overlays go-pob-data on a lua dumper but has lua %q, dump %q", g.name, g.lua, g.dump)
			}
		default:
			t.Errorf("%s is not classified as game data or a PoB-authored table read from the PoE2 pin", g.name)
		}
	}
}

func TestPoE2DataEmitRefuses(t *testing.T) {
	for _, tc := range []struct {
		name, table, body, want string
	}{
		{"unknown field", "bases", `[{"name":"A","type":"Ring","tags":["ring"],"newField":1}]`, "newField"},
		{"present zero quality", "bases", `[{"name":"A","type":"Ring","quality":0}]`, "quality = 0"},
		{"empty implicit", "bases", `[{"name":"A","type":"Ring","implicit":""}]`, `implicit = ""`},
		{"unsorted", "bases", `[{"name":"B","type":"Ring"},{"name":"A","type":"Ring"}]`, "not sorted"},
		{"empty", "gems", `[]`, "no gems"},
		{"empty legion", "legion", `[]`, "no legion nodes"},
		{"legion unknown field", "legion", `[{"id":"a","dn":"A","sa":1}]`, "sa"},
		{"legion no id", "legion", `[{"dn":"A"}]`, "no id"},
		{"gem id gap", "gems", `[{"id":"g","additionalGrantedEffectId2":"x"}]`, "gap"},
		{"item mods too few", "item-mods", `[{"category":"Item","id":"A","lines":["x"],"statOrder":[1]}]`, "did not load"},
		{"rune unsorted slot", "runes", `[{"rune":"R","slot":"b"},{"rune":"R","slot":"a"}]`, "not sorted"},
		{"constants string", "constants", `{"misc":{"a":"x"}}`, "holds 1 tables"},
		{"constants pair shape", "constants", `{"misc":{"a":1},"gameConstants":{"a":1},"characterConstants":{"a":1},"monsterConstants":{"a":1},"playerMinionIntrinsicStats":{"a":1},"monsterEvasionTable":[1],"monsterAccuracyTable":[1],"monsterLifeTable":[1],"monsterAllyLifeTable":[1],"monsterDamageTable":[1],"monsterAllyDamageTable":[1],"monsterArmourTable":[1],"monsterAilmentThresholdTable":[1],"monsterPoiseThresholdTable":[1],"minionLevelTable":[1],"hollowPalmAddedPhys":[[1,2,3]],"totemLifeMult":{"1":1}}`, "not 2"},
		{"constants totem key", "constants", `{"misc":{"a":1},"gameConstants":{"a":1},"characterConstants":{"a":1},"monsterConstants":{"a":1},"playerMinionIntrinsicStats":{"a":1},"monsterEvasionTable":[1],"monsterAccuracyTable":[1],"monsterLifeTable":[1],"monsterAllyLifeTable":[1],"monsterDamageTable":[1],"monsterAllyDamageTable":[1],"monsterArmourTable":[1],"monsterAilmentThresholdTable":[1],"monsterPoiseThresholdTable":[1],"minionLevelTable":[1],"hollowPalmAddedPhys":[[1,2]],"totemLifeMult":{"x":1}}`, "not a skill totem id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			switch tc.table {
			case "bases":
				_, err = p2eBases([]byte(tc.body))
			case "gems":
				_, err = p2eGems([]byte(tc.body))
			case "runes":
				_, err = p2eRunes([]byte(tc.body))
			case "constants":
				_, err = p2eConstants([]byte(tc.body))
			case "legion":
				_, err = p2eLegion([]byte(tc.body))
			case "item-mods":
				_, err = p2eItemMods([]byte(tc.body))
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want an error containing %q", err, tc.want)
			}
		})
	}
}

// TestPoE2DataEmitBoundary is the companion: a present non-zero and an absent field both emit.
func TestPoE2DataEmitBoundary(t *testing.T) {
	out, err := p2eBases([]byte(`[{"name":"A","type":"Ring","quality":20,"armour":{"Evasion":30}},{"name":"B","type":"Ring"}]`))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`{Name: "A", Type: "Ring", Quality: 20, Armour: &BaseArmour{Evasion: 30}},`, `{Name: "B", Type: "Ring"},`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in:\n%s", want, out)
		}
	}
}
