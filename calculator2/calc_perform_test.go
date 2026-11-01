package calculator2

import (
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
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

func loadChargesCursesFixture(t *testing.T) *Environment {
	t.Helper()
	raw, err := os.ReadFile("testdata/charges_curses.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)
	return env
}

func assertPoE2Outputs(t *testing.T, env *Environment, want map[string]float64) {
	t.Helper()
	for stat, w := range want {
		if got, ok := env.Player.Output[stat]; !ok || math.Abs(got-w) > 1e-9 {
			t.Errorf("output %s: PoB-PoE2 %v, ours %v (set %v)", stat, w, got, ok)
		}
	}
}

// As above for the tree-only warrior with custom maximum charge, charge duration, Blood Charge, curse and
// mark limit, rarity and attribute requirement lines.
func TestPerformChargesFixture(t *testing.T) {
	env := loadChargesCursesFixture(t)
	assertPoE2Outputs(t, env, map[string]float64{
		"PowerChargesMax": 4, "PowerChargesDuration": 18, "FrenzyChargesMax": 5, "FrenzyChargesDuration": 18,
		"EnduranceChargesMax": 5, "EnduranceChargesDuration": 21, "BloodChargesMax": 7, "BloodCharges": 7,
		"InspirationCharges": 5, "InspirationChargesMax": 5, "PowerCharges": 0, "TotalCharges": 0,
	})
	testza.AssertEqual(t, 7.0, env.ModDB.Multipliers["BloodCharge"])
}

func TestPerformCurseLimitFixture(t *testing.T) {
	env := loadChargesCursesFixture(t)
	assertPoE2Outputs(t, env, map[string]float64{"EnemyCurseLimit": 4, "EnemyMarkLimit": 2})
}

func TestPerformStatOutputsFixture(t *testing.T) {
	env := loadChargesCursesFixture(t)
	assertPoE2Outputs(t, env, map[string]float64{
		"EffectiveLootRarityMod": 1.3, "LootQuantityNormalEnemies": 0, "LootRarityMagicEnemies": 0,
		"GlobalItemAttributeRequirements": 0.9, "GlobalGemAttributeRequirements": 0.9, "GlobalWeaponAttributeRequirements": 0.9,
		"CharmLimit": 0, "CharmEffect": 20, "FlaskEffect": 10, "WarcryPower": 20,
		"FlaskChargeGen": 0.25, "LifeFlaskChargeGen": 0.45, "ManaFlaskChargeGen": 0.25, "UtilityFlaskChargeGen": 0.25,
	})
}

// As above for the tree-only warrior with a Chill maximum override, stacking Shocks and a Shock maximum line.
func TestPerformAilmentLimitsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/ailment_limits.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)
	assertPoE2Outputs(t, env, map[string]float64{
		"MaximumChill": 35, "MaximumShock": 110, "CurrentChill": 0, "CurrentShock": 0, "ShockStackCount": 1,
	})
}

// As above for the tree-only warrior wearing items with attribute requirements.
func TestPerformAttributeRequirementsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/warrior_items.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)
	assertPoE2Outputs(t, env, map[string]float64{
		"ReqStr": 104, "ReqStrString": 104, "ReqDex": 44, "ReqInt": 92,
		"StrRequirementsOnBody Armour": 60, "StrRequirementsOnBoots": 101, "StrRequirementsOnWeapon 1": 104,
		"DexRequirementsOnGloves": 44, "IntRequirementsOnGloves": 44, "IntRequirementsOnWeapon 2": 92,
	})
}

// As above, for the base damage and local accuracy of the same warrior's weapon.
func TestPerformWeaponBaseStatsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/warrior_items.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)
	assertPoE2Outputs(t, env, map[string]float64{
		"AccuracyOnWeapon 1": 150, "PhysicalMinOnWeapon 1": 157, "PhysicalMaxOnWeapon 1": 231,
		"LightningMinOnWeapon 1": 5, "LightningMaxOnWeapon 1": 50,
	})
	if _, ok := env.Player.Output["FireMinOnWeapon 1"]; ok {
		t.Error("FireMinOnWeapon 1 is set; PoB-PoE2 leaves it unset")
	}
}

// applyEnemyModifiers copies each EnemyModifier's mod into the enemy's modDB once per cache, with the
// outer mod's source when the inner one has none.
func TestApplyEnemyModifiers(t *testing.T) {
	player := &Actor{ModDB: moddb.NewModDB()}
	enemy := &Actor{ModDB: moddb.NewModDB()}
	player.Enemy, enemy.Enemy = enemy, player
	player.ModDB.AddMod(mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Blinded", true)}).Source("Item:Ring"))

	applyEnemyModifiers(player, true)
	applyEnemyModifiers(player, false)
	testza.AssertTrue(t, enemy.ModDB.Flag(nil, "Condition:Blinded"))
	testza.AssertLen(t, enemy.ModDB.Mods["Condition:Blinded"], 1)
	testza.AssertEqual(t, mod.Source("Item:Ring"), enemy.ModDB.Mods["Condition:Blinded"][0].GetSource())

	applyEnemyModifiers(player, true)
	testza.AssertLen(t, enemy.ModDB.Mods["Condition:Blinded"], 2)
}

// The expected values are PoB-PoE2's output for the same file, loaded headless: an aura, a buff, a
// curse and two persistent buffs, plus a disabled herald that must not count.
func TestPerformSkillBuffsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/skill_buffs.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{
		"Evasion": 7, "FireResist": -11, "FireResistTotal": -11, "MaximumRage": 30, "InherentRageLoss": 10,
		"ManaRegen": 11.9, "SpiritReserved": 160, "SpiritUnreserved": 340, "EnemyCurseLimit": 1,
		"MovementSpeedMod": 1,
	} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9*math.Max(1, math.Abs(want)) {
			t.Errorf("%s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
	for name, want := range map[string]float64{"BuffOnSelf": 3, "CurseOnEnemy": 1, "AuraAffectingSelf": 1} {
		if got := env.ModDB.Multipliers[name]; got != want {
			t.Errorf("Multiplier:%s: PoB-PoE2 %v, ours %v", name, want, got)
		}
	}
}

// The expected values are PoB-PoE2's output for the same file, loaded headless: a ballista, a grenade
// and a crossbow ammo skill, each counted once by type.
func TestPerformActiveSkillTypesFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/skill_perform.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	for name, want := range map[string]float64{"GrenadeTypes": 1, "AmmoTypes": 1, "PersistentMinionTypes": 0} {
		if got, ok := env.ModDB.Multipliers[name]; !ok || got != want {
			t.Errorf("Multiplier:%s: PoB-PoE2 %v, ours %v (set %v)", name, want, got, ok)
		}
	}
	for _, stat := range []string{"ActiveTotemLimit", "TotemsSummoned"} {
		if got, ok := env.Player.Output[stat]; ok {
			t.Errorf("%s: PoB-PoE2 leaves it unset, ours %v", stat, got)
		}
	}
}

// The expected values are PoB-PoE2's output for the same file, loaded headless: Eternal Rage with
// increased mana regeneration that also applies to rage regeneration.
func TestPerformManaRegenAppliesToRageFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/mana_regen_to_rage.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	for stat, want := range map[string]float64{"RageRegen": 5.9, "RageRegenRecovery": 5.9, "ManaRegen": 15.5, "MaximumRage": 30} {
		if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9*math.Max(1, math.Abs(want)) {
			t.Errorf("%s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
}

// CalcPerform.lua:193-197: a skill's own exposure chance makes the player able to apply that exposure,
// even when the player's modDB carries none.
func TestHasExposureSourceActiveSkill(t *testing.T) {
	modDB := moddb.NewModDB()
	skillModList := newModList()
	skillModList.Parent = modDB
	skillModList.AddMod(newMod("FireExposureChance", mod.TypeBase, 25, "Skill:Test"))
	env := &Environment{Player: &Actor{ModDB: modDB, ActiveSkillList: []*ActiveSkill{{SkillModList: skillModList, BaseSkillModList: skillModList, SkillCfg: &moddb.ListCfg{}}}}}
	testza.AssertTrue(t, hasExposureSource(modDB, env, "Fire"))
	testza.AssertFalse(t, hasExposureSource(modDB, env, "Cold"))
}

// A support's InflictExposure sits in baseSkillModList, the parent of perform's skillModList;
// ModList:HasMod reads the parent, so config exposure takes the skill's exposure effect:
// floor(20 * (1 + 20/100) * 0.5) = 12 (CalcPerform.lua:3492).
func TestApplyExposureConfigReadsBaseSkillModList(t *testing.T) {
	env := &Environment{ModDB: moddb.NewModDB(), Player: &Actor{}, Enemy: &Actor{ModDB: moddb.NewModDB()}}
	base := newModList()
	base.AddMod(newFlagMod("InflictExposure", "Skill:SupportFireExposurePlayer", mod.ActorCondition("enemy", "Ignited")))
	base.AddMod(newMod("FireExposureEffect", mod.TypeIncrease, 20, "Skill:SupportPotentExposurePlayer"))
	skillModList := newModList()
	skillModList.Parent = base
	flags := mod.MFlag(0)
	env.Player.ActiveSkillList = []*ActiveSkill{{BaseSkillModList: base, SkillModList: skillModList, SkillCfg: &moddb.ListCfg{Flags: &flags}}}
	env.Enemy.ModDB.AddMod(newMod("FireExposure", mod.TypeBase, 20, "Config"))
	env.Enemy.ModDB.AddMod(newMod("ExposureEffectOnSelf", mod.TypeMore, -50, "Unique"))

	applyExposure(env)
	testza.AssertEqual(t, -12.0, env.Enemy.ModDB.Sum(mod.TypeBase, nil, "FireResist"))
}

// "targets affected by abyssal wasting you inflict are blinded" on an Abyssal Wasted enemy: the enemy
// gets both the bare condition flag and Condition:Blinded (CalcPerform.lua:1264-1276); a line without
// applyToEnemy lands on the player.
func TestApplyAbyssalWastingConditions(t *testing.T) {
	env := &Environment{ModDB: moddb.NewModDB(), Player: &Actor{}, Enemy: &Actor{ModDB: moddb.NewModDB()}}
	env.Enemy.ModDB.AddMod(newFlagMod("AbyssalWasted", "Item:Ring"))
	env.ModDB.AddMod(mod.NewList("AbyssalWastingImpliesCondition", LuaTable{"condition": "Blinded", "applyToEnemy": true}).Source("Item:Ring"))
	env.ModDB.AddMod(mod.NewList("AbyssalWastingImpliesCondition", LuaTable{"condition": "Condition:CanWither"}).Source("Item:Ring"))

	applyAbyssalWastingConditions(env)
	testza.AssertTrue(t, env.Enemy.ModDB.Flag(nil, "Condition:Blinded"))
	testza.AssertTrue(t, env.Enemy.ModDB.Flag(nil, "Blinded"))
	testza.AssertTrue(t, env.ModDB.Flag(nil, "Condition:CanWither"))
	testza.AssertFalse(t, env.Enemy.ModDB.Flag(nil, "Condition:CanWither"))
}

// The expected values are PoB-PoE2's enemy modDB for the same file, loaded headless: Pounce's wolf
// skills carry the group's Blind II, so the enemy holds three BlindEffect debuffs, one from Pounce and
// two from the wolf's skills.
func TestPerformMinionSkillDebuffsFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/minion_skill_debuffs.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	testza.AssertLen(t, env.Enemy.ModDB.Mods["BlindEffect"], 3)
	testza.AssertEqual(t, 60.0, env.Enemy.ModDB.Sum(mod.TypeIncrease, nil, "BlindEffect"))
}

// The expected values are PoB-PoE2's output and modDB for the same file, loaded headless: Mace Strike
// with Infernal Cry and Fortifying Cry at 20 warcry power. Only Infernal Cry empowers attacks.
func TestPerformWarcryBuffsFixture(t *testing.T) {
	env := offenceFixtureEnv(t, "testdata/warcry_buffs.xml")
	testza.AssertEqual(t, "Mace Strike", env.Player.MainSkill.ActiveEffect.GrantedEffect.Name)
	assertPoE2Outputs(t, env, map[string]float64{
		"InfernalEmpoweredCount": 2, "InfernalCryCooldown": 8.019, "InfernalCryCastTime": 0.8, "InfernalCryDuration": 8,
		"InfernalCryUptimeRatio": 15.640212550489, "GlobalWarcryUptimeRatio": 15.640212550489,
		"ExertedAttackUptimeRatio": 15.640212550489, "ExertedAttackHitEffect": 0.15640212550489,
		"ExertedAttackAvgDmg": 1, "WarcryPower": 20, "Speed": 1.45,
	})
	for name, want := range map[string]float64{"NumInfernalEmpowers": 2, "Multiplier:EmpoweringWarcryCount": 1, "InfernalExtraFireDamage": 41} {
		if got := env.ModDB.Sum(mod.TypeBase, nil, name); math.Abs(got-want) > 1e-9 {
			t.Errorf("%s: PoB-PoE2 %v, ours %v", name, want, got)
		}
	}
	for _, cond := range []string{"AffectedByInfernal", "AffectedByFortifying"} {
		if !env.ModDB.Conditions[cond] {
			t.Errorf("condition %s: PoB-PoE2 true, ours false", cond)
		}
	}
}

// The expected values are PoB-PoE2's for the same file, loaded headless: Unearth's Bone Construct is
// env.minion, so the enemy's Fully Broken Armour comes from the player's and the minion's misc pass, and
// the minion gets its own defence and offence passes.
func TestPerformMinionActorFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/unearth_armour_broken.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	testza.AssertNotNil(t, env.Minion)
	testza.AssertEqual(t, "UnearthBoneConstruct", env.Minion.Type)
	hit := ModFlagHit
	testza.AssertEqual(t, 40.0, env.Enemy.ModDB.Sum(mod.TypeIncrease, &moddb.ListCfg{Flags: &hit}, "PhysicalDamageTaken"))
	for stat, want := range map[string]float64{"Life": 731, "Armour": 317, "Evasion": 306, "StunDuration": 2.937,
		"TotalDPS": 163.641509434, "AverageDamage": 173.46, "Speed": 0.943396226415, "HitChance": 100} {
		if got := env.Minion.Output[stat]; math.Abs(got-want) > 1e-3 {
			t.Errorf("minion %s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
}

// The expected values are PoB-PoE2's for the same file, loaded headless: the Bone Construct's
// minionData.damage scales the added fire damage its attack gets from the player (AddedDamage MORE),
// not its own weapon damage.
func TestPerformMinionDamageMultiplierFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/unearth_added_fire.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	env, err := BuildOutput(build, Options{Tree: loadTestTree(t)})
	testza.AssertNoError(t, err)

	testza.AssertNotNil(t, env.Minion)
	for stat, want := range map[string]float64{"PhysicalMinBase": 82, "FireMinBase": 9, "FireMaxBase": 13.5,
		"TotalMin": 119.3, "TotalMax": 222.6} {
		if got := env.Minion.OutputMainHand[stat]; math.Abs(got-want) > 1e-6 {
			t.Errorf("minion MainHand %s: PoB-PoE2 %v, ours %v", stat, want, got)
		}
	}
	if got, want := env.Minion.Output["TotalDPS"], 169.337264150943; math.Abs(got-want) > 1e-6 {
		t.Errorf("minion TotalDPS: PoB-PoE2 %v, ours %v", want, got)
	}
}

// The expected values are PoB-PoE2's for the same files, loaded headless: Dark Effigy (skillTotemId 24,
// life multiplier 1) at gem level 17 is a level 34 totem; Ancestral Warrior Totem (skillTotemId 15,
// multiplier 1.2) at gem level 11 is a level 22 totem.
func TestPerformTotemLifeFixtures(t *testing.T) {
	for _, tc := range []struct {
		file             string
		totemLevel, life float64
	}{
		{"testdata/totem_life_effigy.xml", 34, 2384},
		{"testdata/totem_life_ancestral_warrior.xml", 22, 1087},
	} {
		raw, err := os.ReadFile(tc.file)
		testza.AssertNoError(t, err)
		build, err := pob2.ParseBuild(raw)
		testza.AssertNoError(t, err)
		tree := loadTestTree(t)
		env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
		testza.AssertNoError(t, err)

		if got, _ := sdNum(env.Player.MainSkill.SkillData, "totemLevel"); got != tc.totemLevel {
			t.Errorf("%s totemLevel: PoB-PoE2 %v, ours %v", tc.file, tc.totemLevel, got)
		}
		if got := env.Player.Output["TotemLife"]; got != tc.life {
			t.Errorf("%s TotemLife: PoB-PoE2 %v, ours %v", tc.file, tc.life, got)
		}
	}
}
