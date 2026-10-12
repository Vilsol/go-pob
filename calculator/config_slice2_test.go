package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

var slice2Handlers = map[string]string{
	"conditionAgainstDamageOverTime":                "814-816",
	"conditionAttackedRecently":                     "1021-1024",
	"conditionBeenCritRecently":                     "931-933",
	"conditionBeenHitByAttackRecently":              "928-930",
	"conditionBeenSavageHitRecently":                "943-946",
	"conditionBleeding":                             "805-807",
	"conditionBlinded":                              "780-782",
	"conditionBlockedAttackRecently":                "970-973",
	"conditionBlockedHitFromUniqueEnemyInPast10Sec": "1121-1123",
	"conditionBlockedSpellRecently":                 "974-977",
	"conditionBurning":                              "783-785",
	"conditionCastSpellRecently":                    "1025-1028",
	"conditionChilled":                              "789-791",
	"conditionChilledEnemyRecently":                 "897-899",
	"conditionConsumed12SteelShardsRecently":        "934-936",
	"conditionConsumedCorpseInPast2Sec":             "1098-1100",
	"conditionCritWithHeraldSkillRecently":          "837-839",
	"conditionCursedEnemyRecently":                  "1086-1088",
	"conditionFrozenEnemyRecently":                  "894-896",
	"conditionFullEnergyShield":                     "102-104",
	"conditionHaveTotem":                            "736-738",
	"conditionHitBySpellDamageRecently":             "959-962",
	"conditionHitRecently":                          "827-829",
	"conditionHitRecentlyWithWeapon":                "849-851",
	"conditionIgnited":                              "786-788",
	"conditionIgnitedEnemyRecently":                 "903-905",
	"conditionImpaledRecently":                      "1127-1129",
	"conditionKilledAffectedByDoT":                  "888-890",
	"conditionKilledLast3Seconds":                   "859-861",
	"conditionKilledUniqueEnemy":                    "1140-1142",
	"conditionLeechingLife":                         "721-724",
	"conditionLowEnergyShield":                      "105-107",
	"conditionLowMana":                              "99-101",
	"conditionNonCritRecently":                      "843-845",
	"conditionOnBurningGround":                      "768-771",
	"conditionOnFungalGround":                       "765-767",
	"conditionSelfChill":                            "795-797",
	"conditionShocked":                              "801-804",
	"conditionSkillCritRecently":                    "834-836",
	"conditionSoulGainPrevention":                   "1066-1068",
	"conditionStoppedTakingDamageOverTimeRecently":  "981-983",
	"conditionStunnedEnemyRecently":                 "909-911",
	"conditionSummonedTotemRecently":                "739-741",
	"conditionTakenFireDamageFromEnemyHitRecently":  "963-966",
	"conditionTauntedEnemyRecently":                 "1111-1113",
	"conditionThrownTrapOrMineRecently":             "1083-1085",
	"conditionUsedMovementSkillRecently":            "1058-1061",
	"conditionUsedSkillRecently":                    "1014-1016",
	"conditionUsedWarcryInPast8Seconds":             "1074-1076",
	"conditionUsedWarcryRecently":                   "1069-1073",
	"conditionChilledEffect":                        "792-794",
	"multiplierBeenHitRecently":                     "924-927",
	"multiplierBleedsOnEnemy":                       "1133-1136",
	"multiplierCastLast8Seconds":                    "1032-1034",
	"multiplierCorpseConsumedRecently":              "1101-1104",
	"multiplierCorrosionStackCount":                 "1214-1218",
	"multiplierDefiance":                            "712-714",
	"multiplierEnemyAffectedByGraspingVines":        "1345-1347",
	"multiplierEnsnaredStackCount":                  "1219-1222",
	"multiplierFragileRegrowth":                     "1137-1139",
	"multiplierFreezeShockIgniteOnEnemy":            "1305-1307",
	"multiplierHexDoom":                             "275-277",
	"multiplierKilledRecently":                      "855-858",
	"multiplierLifeSpentRecently":                   "915-917",
	"multiplierManaSpentRecently":                   "918-920",
	"multiplierMineDetonatedRecently":               "1077-1079",
	"multiplierNearbyAlly":                          "752-754",
	"multiplierNearbyCorpse":                        "755-757",
	"multiplierPoisonAppliedRecently":               "912-914",
	"multiplierPoisonOnSelf":                        "811-813",
	"multiplierRampage":                             "659-661",
	"multiplierTrapTriggeredRecently":               "1080-1082",
	"multiplierWarcryUsedRecently":                  "1105-1110",
	"multiplierImpalesOnEnemy":                      "1130-1132",
	"multiplierNearbyRareOrUniqueEnemies":           "821-826",
}

// TestSlice2HandlersAreRegistered is the shape gate for the block.
func TestSlice2HandlersAreRegistered(t *testing.T) {
	for name, line := range slice2Handlers {
		if _, ok := configurations[name]; !ok {
			t.Errorf("config handler %q (ConfigOptions.lua:%s) is not registered", name, line)
		}
	}
}

func slice2Apply(t *testing.T, name string, val interface{}) (*moddb.ModList, *moddb.ModList) {
	t.Helper()

	apply, ok := configurations[name]
	if !ok {
		t.Fatalf("config handler %q is not registered", name)
	}

	modList := moddb.NewModList()
	enemyModList := moddb.NewModList()
	apply(val, modList, enemyModList)
	return modList, enemyModList
}

func slice2Names(list *moddb.ModList) []string {
	names := make([]string, 0, len(list.Mods()))
	for _, m := range list.Mods() {
		names = append(names, m.Name())
	}
	return names
}

func slice2Find(t *testing.T, list *moddb.ModList, name string) mod.Mod {
	t.Helper()

	for _, m := range list.Mods() {
		if m.Name() == name {
			return m
		}
	}
	t.Fatalf("no mod named %q in %v", name, slice2Names(list))
	return nil
}

func slice2Flag(t *testing.T, list *moddb.ModList, name string) bool {
	t.Helper()

	return slice2Find(t, list, name).Value().Flag()
}

// TestSlice2CheckHandlersEmitTheirCondition walks every `type = "check"` option in the block and pins the exact set of Condition flags PoB's apply function emits.
func TestSlice2CheckHandlersEmitTheirCondition(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags []string
	}{
		{"conditionAgainstDamageOverTime", []string{"Condition:AgainstDamageOverTime"}},
		{"conditionAttackedRecently", []string{"Condition:AttackedRecently", "Condition:UsedSkillRecently"}},
		{"conditionBeenCritRecently", []string{"Condition:BeenCritRecently"}},
		{"conditionBeenHitByAttackRecently", []string{"Condition:BeenHitByAttackRecently"}},
		{"conditionBeenSavageHitRecently", []string{"Condition:BeenSavageHitRecently", "Condition:BeenHitRecently"}},
		{"conditionBleeding", []string{"Condition:Bleeding"}},
		{"conditionBlinded", []string{"Condition:Blinded"}},
		{"conditionBlockedAttackRecently", []string{"Condition:BlockedAttackRecently", "Condition:BlockedRecently"}},
		{"conditionBlockedHitFromUniqueEnemyInPast10Sec", []string{"Condition:BlockedHitFromUniqueEnemyInPast10Sec"}},
		{"conditionBlockedSpellRecently", []string{"Condition:BlockedSpellRecently", "Condition:BlockedRecently"}},
		{"conditionBurning", []string{"Condition:Burning"}},
		{"conditionCastSpellRecently", []string{"Condition:CastSpellRecently", "Condition:UsedSkillRecently"}},
		{"conditionChilled", []string{"Condition:Chilled"}},
		{"conditionChilledEnemyRecently", []string{"Condition:ChilledEnemyRecently"}},
		{"conditionConsumed12SteelShardsRecently", []string{"Condition:Consumed12SteelShardsRecently"}},
		{"conditionConsumedCorpseInPast2Sec", []string{"Condition:ConsumedCorpseInPast2Sec"}},
		{"conditionCritWithHeraldSkillRecently", []string{"Condition:CritWithHeraldSkillRecently"}},
		{"conditionCursedEnemyRecently", []string{"Condition:CursedEnemyRecently"}},
		{"conditionFrozenEnemyRecently", []string{"Condition:FrozenEnemyRecently"}},
		{"conditionFullEnergyShield", []string{"Condition:FullEnergyShield"}},
		{"conditionHaveTotem", []string{"Condition:HaveTotem"}},
		{"conditionHitBySpellDamageRecently", []string{"Condition:HitBySpellDamageRecently", "Condition:BeenHitRecently"}},
		{"conditionHitRecently", []string{"Condition:HitRecently"}},
		{"conditionHitRecentlyWithWeapon", []string{"Condition:HitRecentlyWithWeapon"}},
		{"conditionIgnited", []string{"Condition:Ignited"}},
		{"conditionIgnitedEnemyRecently", []string{"Condition:IgnitedEnemyRecently"}},
		{"conditionImpaledRecently", []string{"Condition:ImpaledRecently"}},
		{"conditionKilledAffectedByDoT", []string{"Condition:KilledAffectedByDotRecently"}},
		{"conditionKilledLast3Seconds", []string{"Condition:KilledLast3Seconds"}},
		{"conditionKilledUniqueEnemy", []string{"Condition:KilledUniqueEnemy"}},
		{"conditionLeechingLife", []string{"Condition:LeechingLife", "Condition:Leeching"}},
		{"conditionLowEnergyShield", []string{"Condition:LowEnergyShield"}},
		{"conditionLowMana", []string{"Condition:LowMana"}},
		{"conditionNonCritRecently", []string{"Condition:NonCritRecently"}},
		{"conditionOnBurningGround", []string{"Condition:OnBurningGround", "Condition:Burning"}},
		{"conditionOnFungalGround", []string{"Condition:OnFungalGround"}},
		{"conditionSelfChill", []string{"Condition:ChilledSelf"}},
		{"conditionSkillCritRecently", []string{"Condition:SkillCritRecently"}},
		{"conditionSoulGainPrevention", []string{"Condition:SoulGainPrevention"}},
		{"conditionStoppedTakingDamageOverTimeRecently", []string{"Condition:StoppedTakingDamageOverTimeRecently"}},
		{"conditionStunnedEnemyRecently", []string{"Condition:StunnedEnemyRecently"}},
		{"conditionSummonedTotemRecently", []string{"Condition:SummonedTotemRecently"}},
		{"conditionTakenFireDamageFromEnemyHitRecently", []string{"Condition:TakenFireDamageFromEnemyHitRecently", "Condition:BeenHitRecently"}},
		{"conditionTauntedEnemyRecently", []string{"Condition:TauntedEnemyRecently"}},
		{"conditionThrownTrapOrMineRecently", []string{"Condition:TrapOrMineThrownRecently"}},
		{"conditionUsedMovementSkillRecently", []string{"Condition:UsedMovementSkillRecently", "Condition:UsedSkillRecently"}},
		{"conditionUsedSkillRecently", []string{"Condition:UsedSkillRecently"}},
		{"conditionUsedWarcryInPast8Seconds", []string{"Condition:UsedWarcryInPast8Seconds"}},
		{"conditionUsedWarcryRecently", []string{"Condition:UsedWarcryRecently", "Condition:UsedWarcryInPast8Seconds", "Condition:UsedSkillRecently"}},
	} {
		modList, enemyModList := slice2Apply(t, tc.name, true)

		testza.AssertEqual(t, tc.flags, slice2Names(modList), "%s player mods", tc.name)
		testza.AssertEqual(t, 0, len(enemyModList.Mods()), "%s must not touch the enemy", tc.name)
	}
}

// TestSlice2CheckHandlersRespectFalse pins ConfigTab.lua:444-447.
func TestSlice2CheckHandlersRespectFalse(t *testing.T) {
	for name := range slice2Handlers {
		if name == "conditionChilledEffect" || len(name) < 9 || name[:9] != "condition" {
			continue
		}

		modList, enemyModList := slice2Apply(t, name, false)

		testza.AssertEqual(t, 0, len(modList.Mods()), "%s must emit nothing when unticked", name)
		testza.AssertEqual(t, 0, len(enemyModList.Mods()), "%s must emit nothing when unticked", name)
	}
}

// TestSlice2CountHandlersRespectZero pins the other half of ConfigTab.lua:449.
func TestSlice2CountHandlersRespectZero(t *testing.T) {
	for _, name := range []string{
		"conditionChilledEffect", "multiplierBeenHitRecently", "multiplierBleedsOnEnemy",
		"multiplierCastLast8Seconds", "multiplierCorpseConsumedRecently",
		"multiplierCorrosionStackCount", "multiplierDefiance",
		"multiplierEnemyAffectedByGraspingVines", "multiplierEnsnaredStackCount",
		"multiplierFragileRegrowth", "multiplierFreezeShockIgniteOnEnemy", "multiplierHexDoom",
		"multiplierKilledRecently", "multiplierLifeSpentRecently", "multiplierManaSpentRecently",
		"multiplierMineDetonatedRecently", "multiplierNearbyAlly", "multiplierNearbyCorpse",
		"multiplierPoisonAppliedRecently", "multiplierPoisonOnSelf", "multiplierRampage",
		"multiplierTrapTriggeredRecently", "multiplierWarcryUsedRecently",
	} {
		modList, enemyModList := slice2Apply(t, name, float64(0))

		testza.AssertEqual(t, 0, len(modList.Mods()), "%s must emit nothing at zero", name)
		testza.AssertEqual(t, 0, len(enemyModList.Mods()), "%s must emit nothing at zero", name)
	}
}

// TestSlice2CountAllowZeroHandlersApplyAtZero is the inverse.
func TestSlice2CountAllowZeroHandlersApplyAtZero(t *testing.T) {
	_, enemyModList := slice2Apply(t, "multiplierImpalesOnEnemy", float64(0))
	testza.AssertEqual(t, []string{"Multiplier:ImpaleStacks"}, slice2Names(enemyModList))

	modList, enemyModList := slice2Apply(t, "multiplierNearbyRareOrUniqueEnemies", float64(0))
	testza.AssertEqual(t, []string{
		"Multiplier:NearbyRareOrUniqueEnemies", "Multiplier:NearbyEnemies",
		"Condition:AtMostOneNearbyRareOrUniqueEnemy",
	}, slice2Names(modList))
	testza.AssertTrue(t, slice2Flag(t, modList, "Condition:AtMostOneNearbyRareOrUniqueEnemy"))
	testza.AssertEqual(t, []string{"Condition:NearbyRareOrUniqueEnemy"}, slice2Names(enemyModList))
	testza.AssertFalse(t, slice2Flag(t, enemyModList, "Condition:NearbyRareOrUniqueEnemy"))
}

// TestSlice2CountHandlersEmitTheirMultiplier pins the plain `Multiplier:X BASE val` options.
func TestSlice2CountHandlersEmitTheirMultiplier(t *testing.T) {
	for _, tc := range []struct {
		name string
		stat string
	}{
		{"multiplierCastLast8Seconds", "Multiplier:CastLast8Seconds"},
		{"multiplierEnemyAffectedByGraspingVines", "Multiplier:GraspingVinesAffectingEnemy"},
		{"multiplierFreezeShockIgniteOnEnemy", "Multiplier:FreezeShockIgniteOnEnemy"},
		{"multiplierHexDoom", "Multiplier:HexDoomStack"},
		{"multiplierLifeSpentRecently", "Multiplier:LifeSpentRecently"},
		{"multiplierManaSpentRecently", "Multiplier:ManaSpentRecently"},
		{"multiplierMineDetonatedRecently", "Multiplier:MineDetonatedRecently"},
		{"multiplierNearbyAlly", "Multiplier:NearbyAlly"},
		{"multiplierNearbyCorpse", "Multiplier:NearbyCorpse"},
		{"multiplierPoisonAppliedRecently", "Multiplier:PoisonAppliedRecently"},
		{"multiplierPoisonOnSelf", "Multiplier:PoisonStack"},
		{"multiplierRampage", "Multiplier:Rampage"},
		{"multiplierTrapTriggeredRecently", "Multiplier:TrapTriggeredRecently"},
	} {
		modList, enemyModList := slice2Apply(t, tc.name, float64(7))

		testza.AssertEqual(t, []string{tc.stat}, slice2Names(modList), "%s player mods", tc.name)
		testza.AssertEqual(t, mod.TypeBase, slice2Find(t, modList, tc.stat).Type(), "%s mod type", tc.name)
		testza.AssertEqual(t, float64(7), slice2Find(t, modList, tc.stat).Value().Float(), "%s value", tc.name)
		testza.AssertEqual(t, 0, len(enemyModList.Mods()), "%s must not touch the enemy", tc.name)
	}
}

// TestSlice2CountHandlersAreClamped pins the three `m_min` caps in the block.
func TestSlice2CountHandlersAreClamped(t *testing.T) {
	for _, tc := range []struct {
		name string
		stat string
		in   float64
		want float64
	}{
		{"multiplierDefiance", "Multiplier:Defiance", 10, 10},
		{"multiplierDefiance", "Multiplier:Defiance", 25, 10},
		{"multiplierFragileRegrowth", "Multiplier:FragileRegrowthCount", 5, 5},
		{"multiplierFragileRegrowth", "Multiplier:FragileRegrowthCount", 40, 10},
		{"multiplierWarcryUsedRecently", "Multiplier:WarcryUsedRecently", 7, 7},
		{"multiplierWarcryUsedRecently", "Multiplier:WarcryUsedRecently", 250, 100},
	} {
		modList, _ := slice2Apply(t, tc.name, tc.in)

		testza.AssertEqual(t, tc.want, slice2Find(t, modList, tc.stat).Value().Float(),
			"%s(%v)", tc.name, tc.in)
	}
}

// TestSlice2WarcryUsedRecentlyImpliesThreeConditions pins ConfigOptions.lua:1105-1110.
func TestSlice2WarcryUsedRecentlyImpliesThreeConditions(t *testing.T) {
	modList, enemyModList := slice2Apply(t, "multiplierWarcryUsedRecently", float64(3))

	testza.AssertEqual(t, []string{
		"Multiplier:WarcryUsedRecently", "Condition:UsedWarcryRecently",
		"Condition:UsedWarcryInPast8Seconds", "Condition:UsedSkillRecently",
	}, slice2Names(modList))
	testza.AssertEqual(t, 0, len(enemyModList.Mods()))
}

// TestSlice2BeenHitRecentlyConditionTracksTheCount pins the `1 <= val` guard of ConfigOptions.lua:926.
func TestSlice2BeenHitRecentlyConditionTracksTheCount(t *testing.T) {
	modList, _ := slice2Apply(t, "multiplierBeenHitRecently", float64(4))

	testza.AssertEqual(t, []string{"Multiplier:BeenHitRecently", "Condition:BeenHitRecently"},
		slice2Names(modList))
	testza.AssertEqual(t, float64(4), slice2Find(t, modList, "Multiplier:BeenHitRecently").Value().Float())
	testza.AssertTrue(t, slice2Flag(t, modList, "Condition:BeenHitRecently"))
}

// TestSlice2KilledRecentlyImpliesKilled pins ConfigOptions.lua:855-858.
func TestSlice2KilledRecentlyImpliesKilled(t *testing.T) {
	modList, _ := slice2Apply(t, "multiplierKilledRecently", float64(5))

	testza.AssertEqual(t, []string{"Multiplier:EnemyKilledRecently", "Condition:KilledRecently"},
		slice2Names(modList))
	testza.AssertEqual(t, float64(5), slice2Find(t, modList, "Multiplier:EnemyKilledRecently").Value().Float())
	testza.AssertTrue(t, slice2Flag(t, modList, "Condition:KilledRecently"))
}

// TestSlice2CorpseConsumedRecentlyImpliesConsumed pins ConfigOptions.lua:1101-1104.
func TestSlice2CorpseConsumedRecentlyImpliesConsumed(t *testing.T) {
	modList, _ := slice2Apply(t, "multiplierCorpseConsumedRecently", float64(12))

	testza.AssertEqual(t, []string{"Multiplier:CorpseConsumedRecently", "Condition:ConsumedCorpseRecently"},
		slice2Names(modList))
	testza.AssertEqual(t, float64(12), slice2Find(t, modList, "Multiplier:CorpseConsumedRecently").Value().Float())
}

// TestSlice2ShockedEmitsOnlyTheCondition pins ConfigOptions.lua:1199-1201, whose whole apply is the flag.
func TestSlice2ShockedEmitsOnlyTheCondition(t *testing.T) {
	modList, _ := slice2Apply(t, "conditionShocked", true)

	testza.AssertEqual(t, []string{"Condition:Shocked"}, slice2Names(modList))

	playerShockEffect, _ := slice2Apply(t, "conditionPlayerShockEffect", float64(30))
	shockVal := slice2Find(t, playerShockEffect, "ShockVal")
	testza.AssertEqual(t, mod.TypeOverride, shockVal.Type(), "PoB uses OVERRIDE (ConfigOptions.lua:1203)")
	testza.AssertEqual(t, float64(30), shockVal.Value().Float())
	testza.AssertEqual(t, mod.Source("Shock"), shockVal.GetSource())
}

// TestSlice2ChilledEffectOverridesChillVal pins ConfigOptions.lua:792-794.
func TestSlice2ChilledEffectOverridesChillVal(t *testing.T) {
	modList, enemyModList := slice2Apply(t, "conditionChilledEffect", float64(30))

	chillVal := slice2Find(t, modList, "ChillVal")
	testza.AssertEqual(t, mod.TypeOverride, chillVal.Type())
	testza.AssertEqual(t, float64(30), chillVal.Value().Float())
	testza.AssertEqual(t, mod.Source("Chill"), chillVal.GetSource())
	testza.AssertEqual(t, 0, len(enemyModList.Mods()))
}

// TestSlice2EnemySideCountHandlers pins the three options in the block that write to the ENEMY list.
func TestSlice2EnemySideCountHandlers(t *testing.T) {
	modList, enemyModList := slice2Apply(t, "multiplierCorrosionStackCount", float64(4))
	testza.AssertEqual(t, 0, len(modList.Mods()), "corrosion is enemy-side only")
	testza.AssertEqual(t, []string{"Multiplier:CorrosionStack", "Armour", "Evasion"},
		slice2Names(enemyModList))
	testza.AssertEqual(t, float64(4), slice2Find(t, enemyModList, "Multiplier:CorrosionStack").Value().Float())
	testza.AssertEqual(t, float64(-5000), slice2Find(t, enemyModList, "Armour").Value().Float())
	testza.AssertEqual(t, float64(-1000), slice2Find(t, enemyModList, "Evasion").Value().Float())

	modList, enemyModList = slice2Apply(t, "multiplierBleedsOnEnemy", float64(2))
	testza.AssertEqual(t, 0, len(modList.Mods()), "bleeds-on-enemy is enemy-side only")
	testza.AssertEqual(t, []string{"Multiplier:BleedStacks", "Condition:Bleeding"},
		slice2Names(enemyModList))
	testza.AssertEqual(t, float64(2), slice2Find(t, enemyModList, "Multiplier:BleedStacks").Value().Float())

	modList, enemyModList = slice2Apply(t, "multiplierImpalesOnEnemy", float64(6))
	testza.AssertEqual(t, 0, len(modList.Mods()), "impales-on-enemy is enemy-side only")
	testza.AssertEqual(t, []string{"Multiplier:ImpaleStacks"}, slice2Names(enemyModList))
	testza.AssertEqual(t, float64(6), slice2Find(t, enemyModList, "Multiplier:ImpaleStacks").Value().Float())
}

// TestSlice2EnsnareStackCountMakesTheEnemyMove pins ConfigOptions.lua:1219-1222.
func TestSlice2EnsnareStackCountMakesTheEnemyMove(t *testing.T) {
	modList, enemyModList := slice2Apply(t, "multiplierEnsnaredStackCount", float64(1))

	testza.AssertEqual(t, []string{"Multiplier:EnsnareStackCount"}, slice2Names(modList))
	testza.AssertEqual(t, float64(1), slice2Find(t, modList, "Multiplier:EnsnareStackCount").Value().Float())

	testza.AssertEqual(t, []string{"Condition:Moving"}, slice2Names(enemyModList))
	tags := slice2Find(t, enemyModList, "Condition:Moving").Tags()
	testza.AssertEqual(t, 1, len(tags))
	threshold, ok := tags[0].(*mod.MultiplierThresholdTag)
	testza.AssertTrue(t, ok, "Condition:Moving must carry a MultiplierThreshold tag")
	testza.AssertEqual(t, "EnsnareStackCount", threshold.Variable)
	testza.AssertEqual(t, "enemy", threshold.TagActor)
	testza.AssertEqual(t, float64(1), *threshold.TagThreshold)
}

// TestSlice2NearbyRareOrUniqueEnemiesFansOut pins ConfigOptions.lua:821-826.
func TestSlice2NearbyRareOrUniqueEnemiesFansOut(t *testing.T) {
	modList, enemyModList := slice2Apply(t, "multiplierNearbyRareOrUniqueEnemies", float64(3))

	testza.AssertEqual(t, []string{
		"Multiplier:NearbyRareOrUniqueEnemies", "Multiplier:NearbyEnemies",
		"Condition:AtMostOneNearbyRareOrUniqueEnemy",
	}, slice2Names(modList))
	testza.AssertEqual(t, float64(3), slice2Find(t, modList, "Multiplier:NearbyRareOrUniqueEnemies").Value().Float())
	testza.AssertEqual(t, float64(3), slice2Find(t, modList, "Multiplier:NearbyEnemies").Value().Float())
	testza.AssertFalse(t, slice2Flag(t, modList, "Condition:AtMostOneNearbyRareOrUniqueEnemy"))

	testza.AssertEqual(t, []string{"Condition:NearbyRareOrUniqueEnemy"}, slice2Names(enemyModList))
	testza.AssertTrue(t, slice2Flag(t, enemyModList, "Condition:NearbyRareOrUniqueEnemy"))
}

// TestSlice2ConditionTagsMatchPoB spot-checks the `{ type = "Condition", var = ...
func TestSlice2ConditionTagsMatchPoB(t *testing.T) {
	for _, tc := range []struct {
		name  string
		stat  string
		val   interface{}
		enemy bool
		cond  string
	}{
		{"conditionHaveTotem", "Condition:HaveTotem", true, false, "Combat"},
		{"conditionFullEnergyShield", "Condition:FullEnergyShield", true, false, ""},
		{"conditionLowMana", "Condition:LowMana", true, false, ""},
		{"conditionLowEnergyShield", "Condition:LowEnergyShield", true, false, ""},
		{"conditionCursedEnemyRecently", "Condition:CursedEnemyRecently", true, false, "Effective"},
		{"multiplierPoisonOnSelf", "Multiplier:PoisonStack", float64(3), false, "Effective"},
		{"multiplierFreezeShockIgniteOnEnemy", "Multiplier:FreezeShockIgniteOnEnemy", float64(3), false, "Effective"},
		{"multiplierEnemyAffectedByGraspingVines", "Multiplier:GraspingVinesAffectingEnemy", float64(3), false, "Effective"},
		{"multiplierEnsnaredStackCount", "Multiplier:EnsnareStackCount", float64(3), false, "Effective"},
		{"multiplierCorrosionStackCount", "Multiplier:CorrosionStack", float64(3), true, "Effective"},
		{"multiplierBleedsOnEnemy", "Multiplier:BleedStacks", float64(3), true, "Combat"},
		{"multiplierImpalesOnEnemy", "Multiplier:ImpaleStacks", float64(3), true, "Combat"},
	} {
		modList, enemyModList := slice2Apply(t, tc.name, tc.val)
		list := modList
		if tc.enemy {
			list = enemyModList
		}

		tags := slice2Find(t, list, tc.stat).Tags()
		if tc.cond == "" {
			testza.AssertEqual(t, 0, len(tags), "%s must be untagged", tc.name)
			continue
		}

		testza.AssertEqual(t, 1, len(tags), "%s tag count", tc.name)
		condition, ok := tags[0].(*mod.ConditionTag)
		testza.AssertTrue(t, ok, "%s must carry a Condition tag", tc.name)
		testza.AssertEqual(t, []string{tc.cond}, condition.VarList, "%s condition var", tc.name)
	}
}
