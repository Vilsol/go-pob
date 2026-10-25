package calculator2

import (
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob2"
)

// The expected values are PoB-PoE2's for the same file, loaded headless with loadBuildFromXML:
// calcsTab.mainOutput and mainEnv.modDB sums. The tree-only warrior wears a rare ring with a
// repeated line under 30% left-ring bonus effect, Kalandra's Touch under 10% right-ring bonus effect,
// and a Mageblood with a duplicated legacy.
func TestPerformRingBonusMagebloodFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/ring_mageblood.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{
		"Str": 59, "Life": 622, "FireResist": 81, "FireResistTotal": 131, "ColdResist": 6, "LightningResist": 6, "Evasion": 3109,
	} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9 {
			t.Errorf("output %s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
	for stat, want := range map[string]float64{
		"Str": 59, "Life": 592, "FireResist": 75, "FireResistMax": 81, "ElementalResist": 56, "Evasion": 2507,
	} {
		testza.AssertEqual(t, want, env.ModDB.Sum("BASE", nil, stat), "BASE "+stat)
	}
	testza.AssertEqual(t, 24.0, env.ModDB.Sum("INC", nil, "Evasion"), "INC Evasion")
	testza.AssertEqual(t, 5.0, env.ModDB.Sum("INC", nil, "Life"), "INC Life")
}

// As above for the tree-only warrior with custom Onslaught, Adrenaline and "You are Blind" lines.
func TestPerformActorMiscFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/actor_misc.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{"Evasion": 16, "MovementSpeedMod": 1.35, "Life": 489} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9 {
			t.Errorf("output %s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
	for _, c := range []struct {
		modType mod.Type
		name    string
		want    float64
	}{
		{mod.TypeBase, "PhysicalDamageReduction", 10}, {mod.TypeMore, "Accuracy", -20}, {mod.TypeIncrease, "Damage", 100}, {mod.TypeIncrease, "MovementSpeed", 35},
	} {
		testza.AssertEqual(t, c.want, env.ModDB.Sum(c.modType, nil, c.name), string(c.modType)+" "+c.name)
	}
}

// As above for the tree-only warrior wielding a normal Quarterstaff, with custom lines conditioned on
// the weapon-type conditions.
func TestPerformWeaponConditionsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/weapon_conditions.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{"Life": 583, "Armour": 140, "Evasion": 37, "EnergyShield": 0} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9 {
			t.Errorf("output %s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
	var conds []string
	for k, v := range env.ModDB.Conditions {
		if v && (strings.HasPrefix(k, "Using") || strings.Contains(k, "Unarmed") || strings.Contains(k, "Wield")) {
			conds = append(conds, k)
		}
	}
	sort.Strings(conds)
	testza.AssertEqual(t, []string{"UsingMartialWeapon", "UsingMeleeWeapon", "UsingStaff", "UsingTwoHandedWeapon"}, conds)
}
