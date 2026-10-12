package calculator

import (
	"os"
	"testing"

	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
)

// TestSpectreAllyAndPlayerModifiersReachThePlayer covers CalcPerform.lua:2155-2218.
func TestSpectreAllyAndPlayerModifiersReachThePlayer(t *testing.T) {
	t.Run("data", func(t *testing.T) {
		for _, tc := range []struct {
			spectre string
			outer   string
			name    string
			modType mod.Type
			value   float64
			flag    bool
		}{
			{"Metadata/Monsters/LeagueAzmeri/SpecialCorpses/TurtleHigh", "PlayerModifier",
				"PhysicalDamageReduction", mod.TypeBase, 5, false},
			{"Metadata/Monsters/LeagueAzmeri/SpecialCorpses/TurtleMid_", "PlayerModifier",
				"PhysicalDamageReduction", mod.TypeBase, 3, false},
			{"Metadata/Monsters/LeagueAzmeri/SpecialCorpses/CasterDemonHigh", "AllyModifier",
				"CritMultiplier", mod.TypeBase, 30, false},
			{"Metadata/Monsters/LeagueAzmeri/SpecialCorpses/KudukuHigh", "AllyModifier",
				"LightningLuckHits", mod.TypeBase, 0, true},
			{"Metadata/Monsters/LeagueAzmeri/SpecialCorpses/VikingHigh", "AllyModifier",
				"Condition:Onslaught", mod.TypeBase, 0, true},
		} {
			info := raw2.Minions[tc.spectre]
			if info == nil {
				t.Fatalf("REFUSING: %s is not in raw.Minions at all", tc.spectre)
			}
			found := false
			for _, m := range info.ModList {
				if m.Name() != tc.outer {
					continue
				}
				nested, ok := m.Value().List().(mod.MinionModifier)
				if !ok {
					t.Errorf("%s: %s payload is %#v, want mod.MinionModifier",
						tc.spectre, tc.outer, m.Value().List())
					continue
				}
				if nested.Mod.Name() != tc.name {
					continue
				}
				found = true
				if tc.flag {
					if nested.Mod.Value().Type() != mod.ModValueMultiTypeFlag || !nested.Mod.Value().Flag() {
						t.Errorf("%s: %s %s is %#v, want a true flag",
							tc.spectre, tc.outer, tc.name, nested.Mod.Value())
					}
					continue
				}
				if nested.Mod.Type() != tc.modType || nested.Mod.Value().Float() != tc.value {
					t.Errorf("%s: %s %s is %v %v, want %v %v", tc.spectre, tc.outer, tc.name,
						nested.Mod.Type(), nested.Mod.Value().Float(), tc.modType, tc.value)
				}
			}
			if !found {
				t.Errorf("%s carries no %s row for %s; PoB's src/Data/Spectres.lua does",
					tc.spectre, tc.outer, tc.name)
			}
		}
	})

	t.Run("consumer", func(t *testing.T) {
		if os.Getenv("CORPUS") != "true" {
			t.Skip("set CORPUS=true")
		}
		requireGameData(t, "3.29")

		for _, tc := range []struct {
			build string
			stat  string
			want  float64
		}{
			{"0034-elementalist-storm-burst-of-repulsion-2", "BasePhysicalDamageReduction", 30},
			{"0034-elementalist-storm-burst-of-repulsion-2", "BasePhysicalDamageReductionWhenHit", 30},
			{"0337-occultist-lightning-tendrils-of-escalation", "CritMultiplier", 3.98},
		} {
			env := corpusEnvForLoadedData(t, tc.build)
			got, ok := env.Player.Output[tc.stat]
			if !ok {
				t.Errorf("%s: %s absent", tc.build, tc.stat)
				continue
			}
			if got != tc.want {
				t.Errorf("%s: %s = %v, want %v (the pinned PoB's value)", tc.build, tc.stat, got, tc.want)
			}
		}
	})
}
