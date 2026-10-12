package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

//nolint:gosec // G101 false positive: these are ConfigOptions.lua line ranges, not credentials.
var slice6Handlers = map[string]string{
	"lifeRegenMode":                   "119-125 list",
	"armourCalculationMode":           "128-134 list",
	"warcryMode":                      "135-139 list",
	"bannerPlanted":                   "173-175 check",
	"elementalArmyExposureType":       "230-238 list",
	"flameWallAddedDamage":            "255-257 check",
	"frostboltExposure":               "259-261 check",
	"meatShieldEnemyNearYou":          "299-301 check",
	"plagueBearerState":               "303-309 list",
	"summonLightningGolemEnableWrath": "399-401 check",
	"critChanceLucky":                 "1168-1170 check",
}

func TestSlice6HandlersAreRegistered(t *testing.T) {
	for name := range slice6Handlers {
		_, ok := configurations[name]
		testza.AssertTrue(t, ok, "%s must have a live handler", name)
	}
}

// TestSlice6ListOptionsSelectByValue: The "list" options emit a different flag per value and NOTHING for an unrecognised one.
func TestSlice6ListOptionsSelectByValue(t *testing.T) {
	for _, tc := range []struct {
		option string
		value  string
		flag   string
	}{
		{"lifeRegenMode", "AVERAGE", "Condition:LifeRegenBurstAvg"},
		{"lifeRegenMode", "FULL", "Condition:LifeRegenBurstFull"},
		{"armourCalculationMode", "MAX", "Condition:ArmourMax"},
		{"armourCalculationMode", "AVERAGE", "Condition:ArmourAvg"},
		{"warcryMode", "MAX", "Condition:WarcryMaxHit"},
		{"plagueBearerState", "INC", "Condition:PlagueBearerIncubating"},
		{"plagueBearerState", "INF", "Condition:PlagueBearerInfecting"},
	} {
		list := moddb.NewModList()
		configurations[tc.option](tc.value, list, moddb.NewModList())

		var found bool
		for _, m := range list.Mods() {
			if m.Name() == tc.flag {
				found = true
			}
		}
		testza.AssertTrue(t, found, "%s=%q must set %s", tc.option, tc.value, tc.flag)
	}

	for _, option := range []string{"lifeRegenMode", "armourCalculationMode", "warcryMode", "plagueBearerState"} {
		list := moddb.NewModList()
		configurations[option]("NotAValue", list, moddb.NewModList())
		testza.AssertEqual(t, 0, len(list.Mods()), "%s must emit nothing for an unlisted value", option)
	}
}

// TestSlice6CheckOptionsRespectFalse: The "check" options must respect configChecked.
func TestSlice6CheckOptionsRespectFalse(t *testing.T) {
	for _, option := range []string{
		"bannerPlanted", "flameWallAddedDamage", "frostboltExposure",
		"meatShieldEnemyNearYou", "summonLightningGolemEnableWrath", "critChanceLucky",
	} {
		off := moddb.NewModList()
		configurations[option](false, off, moddb.NewModList())
		testza.AssertEqual(t, 0, len(off.Mods()), "%s must emit nothing when unchecked", option)

		on := moddb.NewModList()
		configurations[option](true, on, moddb.NewModList())
		testza.AssertTrue(t, len(on.Mods()) > 0, "%s must emit when checked", option)
	}
}

// TestSlice6ExposureChances: Frostbolt grants 20% cold exposure chance and Elemental Army 100% of the chosen element.
func TestSlice6ExposureChances(t *testing.T) {
	for _, tc := range []struct {
		option string
		value  interface{}
		name   string
		want   float64
	}{
		{"frostboltExposure", true, "ColdExposureChance", 20},
		{"elementalArmyExposureType", "Fire", "FireExposureChance", 100},
		{"elementalArmyExposureType", "Cold", "ColdExposureChance", 100},
		{"elementalArmyExposureType", "Lightning", "LightningExposureChance", 100},
	} {
		list := moddb.NewModList()
		configurations[tc.option](tc.value, list, moddb.NewModList())

		var got float64
		for _, m := range list.Mods() {
			if m.Name() == tc.name && m.Type() == mod.TypeBase {
				got = m.Value().Float()
			}
		}
		testza.AssertEqual(t, tc.want, got, "%s=%v -> %s", tc.option, tc.value, tc.name)
	}
}

// TestSlice6CritChanceLuckyIsEffectiveOnly: critChanceLucky carries a Condition:Effective tag (ConfigOptions.lua:1169), so it applies only in effective mode.
func TestSlice6CritChanceLuckyIsEffectiveOnly(t *testing.T) {
	list := moddb.NewModList()
	configurations["critChanceLucky"](true, list, moddb.NewModList())

	for _, m := range list.Mods() {
		if m.Name() == "CritChanceLucky" {
			testza.AssertEqual(t, 1, len(m.Tags()), "CritChanceLucky carries exactly its Condition tag")
		}
	}
}
