package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSocketThresholdRowsAreTwoSided pins ModParser.lua:3059-3062 (Precursor's Release) against PoB's own state, in BOTH directions and for BOTH conjuncts.
func TestSocketThresholdRowsAreTwoSided(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	cases := []struct {
		build    string
		why      string
		wantFlag map[string]bool
	}{
		{
			build: "1415-hierophant-lightning-tendrils-of-eccentricity-12",
			why:   "Helmet 0, Body Armour 6 -- the helmet row fires and the body-armour row does not",
			wantFlag: map[string]bool{
				"StunImmune":             true,
				"ElementalAilmentImmune": false,
			},
		},
		{
			build: "4508-ladder-Dexinin_4356.FRFDEXININ",
			why:   "Helmet 4, Body Armour 0 -- the SAME two rows, the OTHER way round",
			wantFlag: map[string]bool{
				"StunImmune":             false,
				"ElementalAilmentImmune": true,
			},
		},
		{
			build: "7988-ladder-podcerepnikovivan_2165.Rokoah",
			why:   "Helmet 4, Body Armour 6 -- neither fires",
			wantFlag: map[string]bool{
				"StunImmune":             false,
				"ElementalAilmentImmune": false,
			},
		},
		{
			build: "4879-ladder-Harwinnam_4002.AFNiwrah",
			why:   "NO helmet, and a body armour with no gems: the threshold passes on BOTH (an absent multiplier reads 0) and only Condition:UsingHelmet separates them",
			wantFlag: map[string]bool{
				"StunImmune":             false,
				"ElementalAilmentImmune": true,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, c.build)
			if env.Player == nil || env.Player.ModDB == nil {
				t.Fatalf("REFUSING: %s built no player ModDB", c.build)
			}
			for name, want := range c.wantFlag {
				got := env.Player.ModDB.Flag(nil, name)
				if got != want {
					t.Errorf("%s = %v, want %v (%s)", name, got, want, c.why)
				}
			}
		})
	}
}

// TestSocketThresholdRowsCarryBothConjuncts is the structural half: every one of the seven rows must carry the MultiplierThreshold AND the Condition.
func TestSocketThresholdRowsCarryBothConjuncts(t *testing.T) {
	cases := map[string]struct {
		multiplier string
		condition  string
		modName    string
	}{
		"Action Speed cannot be modified to below Base Value if you have Equipped Boots with no Socketed Gems":    {"SocketedGemsInBoots", "UsingBoots", "MinimumActionSpeed"},
		"Cannot be Stunned if you have an Equipped Helmet with no Socketed Gems":                                  {"SocketedGemsInHelmet", "UsingHelmet", "StunImmune"},
		"Elemental Ailments cannot be inflicted on you if you have an Equipped Body Armour with no Socketed Gems": {"SocketedGemsInBody Armour", "UsingBody Armour", "ElementalAilmentImmune"},
		"Take no Extra Damage from Critical Strikes if you have Equipped Gloves with no Socketed Gems":            {"SocketedGemsInGloves", "UsingGloves", "ReduceCritExtraDamage"},
		"Defences from Equipped Body Armour are Doubled if it has no Socketed Gems":                               {"SocketedGemsInBody Armour", "UsingBody Armour", "Defences"},
		"+25% to all Elemental Resistances if you have an Equipped Helmet with no Socketed Gems":                  {"SocketedGemsInHelmet", "UsingHelmet", "ElementalResist"},
		"10% increased maximum Life if you have Equipped Gloves with no Socketed Gems":                            {"SocketedGemsInGloves", "UsingGloves", "Life"},
		"20% increased Movement Speed if you have Equipped Boots with no Socketed Gems":                           {"SocketedGemsInBoots", "UsingBoots", "MovementSpeed"},
	}

	for line, want := range cases {
		t.Run(want.modName, func(t *testing.T) {
			entry := ParseMod(line, false)
			if entry.Extra != "" {
				t.Fatalf("unparsed remainder %q -- the whole line is discarded", entry.Extra)
			}
			var target mod.Mod
			for _, m := range entry.ModList {
				if m.Name() == want.modName {
					target = m
				}
			}
			if target == nil {
				t.Fatalf("no %s mod in %v", want.modName, entry.ModList)
			}
			var sawThreshold, sawCondition bool
			for _, tg := range target.Tags() {
				switch tt := tg.(type) {
				case *mod.MultiplierThresholdTag:
					if tt.Variable != want.multiplier {
						t.Errorf("threshold var %q, want %q", tt.Variable, want.multiplier)
					}
					if tt.TagThreshold == nil || *tt.TagThreshold != 0 || !tt.TagUpper {
						t.Errorf("threshold %v upper %v, want 0 / true", tt.TagThreshold, tt.TagUpper)
					}
					sawThreshold = true
				case *mod.ConditionTag:
					for _, v := range tt.VarList {
						if v == want.condition {
							sawCondition = true
						}
					}
				}
			}
			if !sawThreshold {
				t.Error("no MultiplierThreshold tag")
			}
			if !sawCondition {
				t.Errorf("no Condition %q tag -- without it the row fires on an EMPTY slot", want.condition)
			}
		})
	}
}
