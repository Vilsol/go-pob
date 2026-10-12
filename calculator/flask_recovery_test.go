package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

func flaskRecoveryFixture() *Item {
	return &Item{
		Name:     "Test Life Flask",
		BaseName: "Divine Life Flask",
		Rarity:   RarityMagic,
		Base:     &ItemBase{Flask: &ItemBaseFlask{Life: 1000, Duration: 5}},
		FlaskData: map[string]float64{
			"lifeBase":    1000,
			"duration":    5,
			"instantPerc": 0,
			"effectInc":   0,
		},
	}
}

func flaskRecoveryEnv(mods ...mod.Mod) *Environment {
	modDB := moddb.NewModDB()
	for _, m := range mods {
		modDB.AddMod(m)
	}
	return &Environment{ModDB: modDB}
}

// TestFlaskRecoveryIsGatedOnEffectNotRemoved pins CalcPerform.lua:1621, the gate that decides whether a flask's recovery becomes a per-second modifier at all.
//
//	if not item.flaskData[lType.."EffectNotRemoved"] and not modDB:Flag(nil, type.."FlaskEffectNotRemoved") then
//	end
func TestFlaskRecoveryIsGatedOnEffectNotRemoved(t *testing.T) {
	item := flaskRecoveryFixture()

	testza.AssertNil(t, calcFlaskRecovery(flaskRecoveryEnv(), "Life", item, 0, 0, 0),
		"with neither source of the flag, :1621 returns an empty list")

	fromDB := calcFlaskRecovery(
		flaskRecoveryEnv(mod.NewFlag("LifeFlaskEffectNotRemoved", true).Source("Test")),
		"Life", item, 0, 0, 0)
	testza.AssertLen(t, fromDB, 1, "the modDB half of :1621 alone must open the gate")
	testza.AssertEqual(t, "LifeRecovery", fromDB[0].Name())
	testza.AssertEqual(t, 200.0, fromDB[0].Value().Float(),
		"1000 life over a 5s duration is 200/s")

	item.FlaskData["lifeEffectNotRemoved"] = 1
	fromItem := calcFlaskRecovery(flaskRecoveryEnv(), "Life", item, 0, 0, 0)
	testza.AssertLen(t, fromItem, 1, "the flaskData half of :1621 alone must open the gate")
}

// TestFlaskRecoveryInstantShareIsNotSpreadOverTime pins :1614 and :1632's `(1 - instPerc/100)` together.
func TestFlaskRecoveryInstantShareIsNotSpreadOverTime(t *testing.T) {
	item := flaskRecoveryFixture()
	item.FlaskData["instantPerc"] = 25

	out := calcFlaskRecovery(
		flaskRecoveryEnv(mod.NewFlag("LifeFlaskEffectNotRemoved", true).Source("Test")),
		"Life", item, 0, 0, 0)
	testza.AssertLen(t, out, 1)
	testza.AssertEqual(t, 150.0, out[0].Value().Float(),
		"1000 * (1 - 25/100) / 5 = 150")

	item.FlaskData["instantLowLifePerc"] = 60
	item.FlaskData["instantPerc"] = 60
	capped := calcFlaskRecovery(
		flaskRecoveryEnv(
			mod.NewFlag("LifeFlaskEffectNotRemoved", true).Source("Test"),
			mod.NewFlag("Condition:LowLife", true).Source("Test")),
		"Life", item, 0, 0, 0)
	testza.AssertLen(t, capped, 1)
	testza.AssertEqual(t, 0.0, capped[0].Value().Float(),
		"instPerc clamps to 100, so nothing is left to recover gradually")
}

// TestFlaskRecoveryLowLifeMoreIsPreDividedByFlaskEffect pins :1635-1641 and the reason upstream gives for it.
func TestFlaskRecoveryLowLifeMoreIsPreDividedByFlaskEffect(t *testing.T) {
	env := flaskRecoveryEnv(
		mod.NewFlag("LifeFlaskEffectNotRemoved", true).Source("Test"),
		mod.NewFloat("FlaskLifeRecoveryLowLife", mod.TypeMore, 100).Source("Test"))

	life := calcFlaskRecovery(env, "Life", flaskRecoveryFixture(), 100, 0, 0)
	testza.AssertLen(t, life, 1)
	testza.AssertEqual(t, 300.0, life[0].Value().Float(),
		"(more-1)/(1+effectInc/100) + 1 = 1.5, applied to 200/s")

	mana := flaskRecoveryFixture()
	mana.Base.Flask = &ItemBaseFlask{Mana: 1000, Duration: 5}
	mana.FlaskData["manaBase"] = 1000
	manaEnv := flaskRecoveryEnv(
		mod.NewFlag("ManaFlaskEffectNotRemoved", true).Source("Test"),
		mod.NewFloat("FlaskLifeRecoveryLowLife", mod.TypeMore, 100).Source("Test"))
	manaOut := calcFlaskRecovery(manaEnv, "Mana", mana, 100, 0, 0)
	testza.AssertLen(t, manaOut, 1)
	testza.AssertEqual(t, 200.0, manaOut[0].Value().Float(),
		"the low-life multiplier is life-only; the mana arm of :1640 is a bare 1")
}

// TestFlaskRecoveryRedirectArmsAreAdditional pins :1645-1651.
func TestFlaskRecoveryRedirectArmsAreAdditional(t *testing.T) {
	out := calcFlaskRecovery(
		flaskRecoveryEnv(
			mod.NewFlag("LifeFlaskEffectNotRemoved", true).Source("Test"),
			mod.NewFlag("LifeFlaskAppliesToEnergyShield", true).Source("Test")),
		"Life", flaskRecoveryFixture(), 0, 0, 0)

	testza.AssertLen(t, out, 2, ":1643's mod is kept alongside :1646's")
	testza.AssertEqual(t, "LifeRecovery", out[0].Name())
	testza.AssertEqual(t, "EnergyShieldRecovery", out[1].Name())
	testza.AssertEqual(t, out[0].Value().Float(), out[1].Value().Float())
}

// TestFlaskRecoveryMergeScalesByFlaskEffectAndDoesNotStack pins the two properties the merge adds on top of calcFlaskRecovery.
func TestFlaskRecoveryMergeScalesByFlaskEffectAndDoesNotStack(t *testing.T) {
	env := flaskRecoveryEnv(mod.NewFlag("LifeFlaskEffectNotRemoved", true).Source("Test"))

	weak := flaskRecoveryFixture()
	strong := flaskRecoveryFixture()
	strong.FlaskData["lifeBase"] = 2000

	dest := make(map[string]*moddb.ModList)
	order := mergeFlaskRecovery(env, strong, "Life", "LifeFlask", 50, 0, 0, dest, nil)
	order = mergeFlaskRecovery(env, weak, "Life", "LifeFlask", 50, 0, 0, dest, order)

	testza.AssertLen(t, order, 1, "both flasks merge under the one key, so the order list is 1")
	testza.AssertEqual(t, 600.0, dest["LifeFlask"].Sum(mod.TypeBase, nil, "LifeRecovery"),
		"2000/5 = 400/s scaled by a 50% flask effect = 600/s, and the weaker flask does not add to it")
}
