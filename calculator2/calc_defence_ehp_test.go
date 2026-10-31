package calculator2

import (
	"math"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob2"
)

// ehpTestActor is an actor whose output holds what numberOfHitsToDie and reducePoolsByDamage read.
func ehpTestActor() *ehpActor {
	output := map[string]float64{
		"Life": 3000, "LifeRecoverable": 2500, "EnergyShieldRecoveryCap": 1200, "Ward": 150, "ManaUnreserved": 800,
		"sharedMindOverMatter": 30, "preventedLifeLoss": 0, "preventedLifeLossBelowHalf": 0, "preventedLifeLossTotal": 0,
		"FrostShieldLife": 0, "FrostShieldDamageMitigation": 0, "sharedAegis": 0, "sharedElementalAegis": 200,
		"VaalArcticArmourLife": 0, "sharedGuardAbsorb": 0, "sharedGuardAbsorbRate": 0,
	}
	for _, t := range dmgTypeList {
		output[t+"EnergyShieldBypass"] = 0
		output[t+"MindOverMatter"] = 0
		output[t+"Aegis"] = 0
		output[t+"GuardAbsorb"] = 0
		output[t+"GuardAbsorbRate"] = 0
	}
	output["ChaosEnergyShieldBypass"] = 25
	output["FireMindOverMatter"] = 10
	return &ehpActor{Actor: &Actor{ModDB: moddb.NewModDB(), Output: output}}
}

// The expected values are PoB-PoE2's numberOfHitsToDie and calcs.reducePoolsByDamage
// (CalcDefence.lua:427-699, 2054-2220) run under LuaJIT on the same inputs.
func TestNumberOfHitsToDie(t *testing.T) {
	actor := ehpTestActor()
	in := newEHPDamageIn()
	in.damage = map[string]float64{"Physical": 900, "Fire": 300.5, "Cold": 0, "Lightning": 120, "Chaos": 210}
	if got, want := numberOfHitsToDie(actor.Output, actor, in), 3.0659915060437766; math.Abs(got-want) > 1e-9 {
		t.Errorf("numberOfHitsToDie = %.17g, PoB %.17g", got, want)
	}
}

func TestReducePoolsByDamage(t *testing.T) {
	actor := ehpTestActor()
	pools := reducePoolsByDamage(nil, map[string]float64{"Physical": 1500, "Fire": 700, "Chaos": 400}, actor)
	for name, c := range map[string][2]float64{
		"Life": {pools.life, 1450}, "Mana": {pools.mana, 350}, "EnergyShield": {pools.energyShield, 0},
		"Ward": {pools.ward, 150}, "OverkillDamage": {pools.overkillDamage, 0}, "hitPoolRemaining": {pools.hitPoolRemaining, 1950},
	} {
		if math.Abs(c[0]-c[1]) > 1e-9 {
			t.Errorf("%s = %.17g, PoB %.17g", name, c[0], c[1])
		}
	}
}

// The expected damages are ConfigOptions.lua's enemyIsBoss placeholders computed under LuaJIT from
// PoB-PoE2's data.monsterDamageTable; 965 / 386 is the Pinnacle placeholder every saved corpus build holds.
func TestEnemyConfigPlaceholders(t *testing.T) {
	for _, c := range []struct {
		boss         string
		level, input float64
		phys, chaos  float64
		pen          float64
		hasPen       bool
	}{
		{boss: "Pinnacle", level: 90, phys: 965, chaos: 386, pen: 3, hasPen: true},
		{boss: "Boss", level: 70, phys: 338, chaos: 135},
		{boss: "Uber", level: 90, input: 84, phys: 1322, chaos: 331, pen: 8, hasPen: true},
	} {
		build := &pob2.PathOfBuilding2{}
		build.Build.Level = int(c.level)
		env := &Environment{Build: build, ConfigInput: map[string]ConfigValue{"enemyIsBoss": {String: c.boss, IsString: true}}}
		if c.input > 0 {
			env.ConfigInput["enemyLevel"] = ConfigValue{Number: c.input, IsNumber: true}
		}
		p := enemyConfigPlaceholders(env)
		if p["enemyPhysicalDamage"] != c.phys || p["enemyFireDamage"] != c.phys || p["enemyChaosDamage"] != c.chaos {
			t.Errorf("%s: physical %v fire %v chaos %v, PoB %v / %v", c.boss, p["enemyPhysicalDamage"], p["enemyFireDamage"], p["enemyChaosDamage"], c.phys, c.chaos)
		}
		if pen, ok := p["enemyColdPen"]; ok != c.hasPen || pen != c.pen {
			t.Errorf("%s: cold pen %v (set %v), PoB %v (set %v)", c.boss, pen, ok, c.pen, c.hasPen)
		}
	}
}

// The expected values are PoB-PoE2's for the same file, loaded headless: a chance to avoid a damage
// type sets output.specificTypeAvoidance, so the EHP pass averages each type's avoidance.
func TestEHPSpecificTypeAvoidanceFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/avoid_damage_types.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, env.Player.SpecificTypeAvoidance)
	for stat, want := range map[string]float64{"ConfiguredDamageChance": 89.4, "TotalEHP": 668.63585774864, "NumberOfMitigatedDamagingHits": 0.15747429527759} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-6*math.Max(1, want) {
			t.Errorf("%s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
}
