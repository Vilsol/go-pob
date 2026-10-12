package calculator

import (
	"maps"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

//pob:src/Classes/ConfigTab.lua:493-524
func buildConfigModLists(env *Environment, build *pob.PathOfBuilding) (*moddb.ModList, *moddb.ModList) {
	modList := moddb.NewModList()
	enemyModList := moddb.NewModList()

	inputs := make(map[string][]pob.Input, len(build.Config.Inputs))
	suppliedInputs := make(map[string]bool, len(build.Config.Inputs))
	for _, input := range build.Config.Inputs {
		if input.Name == "customMods" {
			continue
		}
		inputs[input.Name] = append(inputs[input.Name], input)
		if input.Number == nil || *input.Number != 0 || configCountAllowZero[input.Name] {
			suppliedInputs[input.Name] = true
		}
	}

	for _, name := range configOrder {
		config, ok := configurations[name]
		if withBuild, found := configurationsWithBuild[name]; !ok && found {
			config, ok = func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {
				withBuild(val, modList, enemyModList, build)
			}, true
		}
		if !ok {
			continue
		}
		for _, input := range inputs[name] {
			var value interface{}
			if input.String != nil {
				value = *input.String
			} else if input.Boolean != nil {
				value = *input.Boolean
			} else if input.Number != nil {
				value = *input.Number
			}
			config(value, modList, enemyModList)
		}
		if suppliedInputs[name] {
			continue
		}
		if isConfigDefault(name) {
			config(true, modList, enemyModList)
		}
		if def, ok := configIndexDefaults[name]; ok {
			config(def, modList, enemyModList)
		}
		if isConfigPlaceholderVar(name) {
			applyConfigPlaceholder(env, name, modList, enemyModList)
		}
	}

	return applyCustomModifierBlocks(build, modList), enemyModList
}

//pob:block:src/Modules/CalcSetup.lua:503-506
func addBaseResistancePenalty(db *moddb.ModDB, build *pob.PathOfBuilding) {
	penalty := float64(-60)
	if value, ok := build.GetNumberOptionOK("resistancePenalty"); ok {
		penalty = value
	}

	for _, name := range []string{"FireResist", "ColdResist", "LightningResist", "ChaosResist"} {
		db.AddMod(mod.NewFloat(name, mod.TypeBase, penalty).Source("Base"))
	}
}

//pob:src/Modules/CalcSetup.lua:263-1151
func InitEnv(build *pob.PathOfBuilding, envCache *EnvironmentCache, mode OutputMode) (*Environment, moddb.ModStoreFuncs, moddb.ModStoreFuncs, moddb.ModStoreFuncs) {
	return initEnvWith(build, envCache, mode, false, nil)
}

//pob:src/Modules/CalcSetup.lua:376-400
func InitEnvOverride(build *pob.PathOfBuilding, envCache *EnvironmentCache, mode OutputMode, override *CalcOverride) (*Environment, moddb.ModStoreFuncs, moddb.ModStoreFuncs, moddb.ModStoreFuncs) {
	return initEnvWith(build, envCache, mode, false, override)
}

//pob:block:src/Classes/ConfigTab.lua:1062-1071
func resolveEnemyLevel(input, placeholder, characterLevel float64) int {
	level := characterLevel
	switch {
	case input > 0:
		level = input
	case placeholder > 0:
		level = placeholder
	}
	return max(1, min(data.MaxEnemyLevel, int(level)))
}

func initEnvWith(build *pob.PathOfBuilding, envCache *EnvironmentCache, mode OutputMode, alignCalcs bool, override *CalcOverride) (*Environment, moddb.ModStoreFuncs, moddb.ModStoreFuncs, moddb.ModStoreFuncs) {
	return initEnvSpec(build, envCache, mode, alignCalcs, override, nil)
}

//pob:block:src/Modules/Calcs.lua:328-335
type calcAccelerate struct {
	nodeAlloc         bool
	requirementsItems bool
	requirementsGems  bool
	skills            bool
	everything        bool
}

//pob:block:src/Modules/CalcSetup.lua:379-383
type calcSpecEnv struct {
	cachedPlayerDB  moddb.ModStoreFuncs
	cachedEnemyDB   moddb.ModStoreFuncs
	cachedMinionDB  moddb.ModStoreFuncs
	env             *Environment
	accelerate      calcAccelerate
	flatDB          bool
	clusterNodes    map[string]data.Node
	clusterNodesSet bool
}

func wipeModDB(db *moddb.ModDB) {
	clear(db.Mods)
	clear(db.Conditions)
	clear(db.Multipliers)
}

func copyModDB(dst *moddb.ModDB, src moddb.ModStoreFuncs) {
	cached := src.(*moddb.ModDB)
	dst.AddDB(cached)
	maps.Copy(dst.Multipliers, cached.Multipliers)
	maps.Copy(dst.Conditions, cached.Conditions)
}

//pob:src/Modules/CalcSetup.lua:216-290
func wipeEnv(env *Environment, accelerate calcAccelerate) {
	env.itemText = nil
	env.itemsByName = nil
	wipeModDB(env.ModDB)
	wipeModDB(env.EnemyModDB)
	if env.Minion != nil && env.Minion.ModDB != nil {
		wipeModDB(env.Minion.ModDB)
	}

	if accelerate.everything {
		return
	}

	if !accelerate.nodeAlloc {
		clear(env.AllocatedNodes)
		clear(env.GrantedPassives)
		clear(env.GrantedSkillsNodes)
	}

	if !accelerate.requirementsItems {
		wipeModDB(env.ItemModDB)
		clear(env.RadiusJewelList)
		clear(env.ExtraRadiusNodeList)
		clear(env.Player.ItemList)
		clear(env.GrantedSkillsItems)
		env.Flasks = env.Flasks[:0]
		env.Tinctures = env.Tinctures[:0]
		env.RequirementsTableItems = env.RequirementsTableItems[:0]
	}

	if !accelerate.requirementsGems {
		env.RequirementsTableGems = env.RequirementsTableGems[:0]
	}

	if !accelerate.skills {
		env.Player.ActiveSkillList = env.Player.ActiveSkillList[:0]
		env.AuxSkillList = env.AuxSkillList[:0]
	}
}

//pob:block:src/Modules/CalcSetup.lua:1843-1863
func resetAcceleratedSkillData(env *Environment, activeSkill *ActiveSkill) {
	skillData := activeSkill.SkillData
	activeSkill.SkillData = &SkillData{}
	for _, value := range utils.CastSlice[mod.SkillData](env.ModDB.List(activeSkill.SkillCfg, "SkillData")) {
		utils.Set(activeSkill.SkillData, value.Key, value.Value)
	}
	for _, value := range utils.CastSlice[mod.SkillData](activeSkill.SkillModList.List(activeSkill.SkillCfg, "SkillData")) {
		utils.Set(activeSkill.SkillData, value.Key, value.Value)
	}
	activeSkill.SkillData.ManaReservationPercent = skillData.ManaReservationPercent
	activeSkill.SkillData.Cooldown = skillData.Cooldown
	activeSkill.SkillData.StoredUses = skillData.StoredUses
	activeSkill.SkillData.CritChance = skillData.CritChance
	activeSkill.SkillData.AttackTime = skillData.AttackTime
	activeSkill.SkillData.AttackSpeedMultiplier = skillData.AttackSpeedMultiplier
	activeSkill.SkillData.SoulPreventionDuration = 0
	activeSkill.SkillData.TotemLevel = skillData.TotemLevel
	activeSkill.SkillData.DamageEffectiveness = skillData.DamageEffectiveness
	activeSkill.SkillData.StagesMax = skillData.StagesMax
}

type allocatedNodeCounts struct {
	notable      int
	mastery      int
	keystone     int
	masteryTypes map[string]int
}

func countAllocatedNodes(env *Environment, nodes map[string]data.Node, masteryOverrides map[string]*data.LegionNode) allocatedNodeCounts {
	counts := allocatedNodeCounts{masteryTypes: make(map[string]int, len(env.Spec.MasterySelections))}
	for _, node := range nodes {
		if data.IsTreeNotable(node) {
			counts.notable++
		}
		if node.IsKeystone != nil && *node.IsKeystone {
			counts.keystone++
		}
	}
	for nodeID, effectID := range env.Spec.MasterySelections {
		node, ok := nodes[strconv.FormatInt(nodeID, 10)]
		if !ok {
			continue
		}
		hasEffect := false
		for _, effect := range node.MasteryEffects {
			if effect.Effect == effectID {
				hasEffect = true
				break
			}
		}
		if !hasEffect {
			continue
		}
		counts.mastery++

		name := ""
		if node.Name != nil {
			name = *node.Name
		}
		if override, overridden := masteryOverrides[strconv.FormatInt(nodeID, 10)]; overridden &&
			override != nil && strings.HasPrefix(override.DN, "Runegraft ") {
			name = "Runegraft Mastery"
		}
		if name != "" {
			counts.masteryTypes[name]++
		}
	}
	return counts
}

//pob:src/Modules/CalcSetup.lua:376-460
func initEnvSpec(build *pob.PathOfBuilding, envCache *EnvironmentCache, mode OutputMode, alignCalcs bool, override *CalcOverride, specEnv *calcSpecEnv) (*Environment, moddb.ModStoreFuncs, moddb.ModStoreFuncs, moddb.ModStoreFuncs) {
	return initEnvSpecKeep(build, envCache, mode, alignCalcs, override, specEnv, true)
}

func initEnvOnly(build *pob.PathOfBuilding, envCache *EnvironmentCache, mode OutputMode, override *CalcOverride) *Environment {
	env, _, _, _ := initEnvSpecKeep(build, envCache, mode, false, override, nil, false)
	return env
}

func initEnvSpecKeep(build *pob.PathOfBuilding, envCache *EnvironmentCache, mode OutputMode, alignCalcs bool, override *CalcOverride, specEnv *calcSpecEnv, keepCached bool) (*Environment, moddb.ModStoreFuncs, moddb.ModStoreFuncs, moddb.ModStoreFuncs) {
	var (
		cachedPlayerDB, cachedEnemyDB, cachedMinionDB moddb.ModStoreFuncs
		env                                           *Environment
		accelerate                                    calcAccelerate
	)
	if specEnv != nil {
		cachedPlayerDB, cachedEnemyDB, cachedMinionDB = specEnv.cachedPlayerDB, specEnv.cachedEnemyDB, specEnv.cachedMinionDB
		env = specEnv.env
		accelerate = specEnv.accelerate
	}
	if override == nil {
		override = &CalcOverride{}
	}

	currentTreeVersion := data.LatestTreeVersion

	if env != nil {
		wipeEnv(env, accelerate)
		env.accelerated = accelerate.everything
	} else {
		env = &Environment{}
		env.Cache = envCache

		env.Cache.useTreeVersion(currentTreeVersion)

		env.DebugErrors = make([]string, 0)
		env.Build = build
		env.Mode = mode
		env.override = override
		env.Spec = NewPassiveSpec(build, data.LatestTreeVersion)

		env.ModDB = moddb.NewModDBSized(384)
		env.EnemyModDB = moddb.NewModDBSized(192)
		env.ItemModDB = moddb.NewModDBSized(96)

		env.ConfigPlaceholder = enemyConfigPlaceholders(build)

		env.EnemyLevel = resolveEnemyLevel(
			build.GetNumberOption("enemyLevel"),
			env.ConfigPlaceholder["enemyLevel"],
			float64(build.Build.Level))

		env.Player = &Actor{
			ModDB:           env.ModDB,
			Level:           build.Build.Level,
			ActiveSkillList: make([]*ActiveSkill, 0),
		}

		env.Enemy = &Actor{
			ModDB: env.EnemyModDB,
			Level: env.EnemyLevel,
		}

		env.ModDB.Actor = env.Player
		env.EnemyModDB.Actor = env.Enemy
		env.Player.Enemy = env.Enemy
		env.Enemy.Enemy = env.Player
		env.Enemy.Player = env.Player
		env.Player.Player = env.Player

		env.RequirementsTableItems = make([]*RequirementsTable, 0)
		env.RequirementsTableGems = make([]*RequirementsTable, 0)

		env.RadiusJewelList = make(map[string]interface{})
		env.ExtraRadiusNodeList = make(map[string]interface{})
		env.GrantedSkills = make(map[string]interface{})
		env.GrantedSkillsNodes = make(map[string]interface{})
		env.GrantedSkillsItems = make(map[string]interface{})
		env.Flasks = make([]*Item, 0)

		env.GrantedPassives = make(map[string]interface{})
		env.GrantedAscendancyNodes = make(map[string]bool)

		env.AuxSkillList = make([]*ActiveSkill, 0)
	}

	//pob:block:src/Modules/CalcSetup.lua:462-468
	buffMode := BuffModeEffective
	if mode == OutputModeCalcs {
		buffMode = BuffMode(build.CalcsInputString("misc_buffMode", string(BuffModeEffective)))
	}

	switch buffMode {
	case BuffModeEffective:
		env.ModeEffective = true
		fallthrough
	case BuffModeCombat:
		env.ModeCombat = true
		fallthrough
	case BuffModeBuffed:
		env.ModeBuffs = true
	}

	if env.partyTab == nil {
		env.partyTab = loadPartyTab(build.Party)
	}

	//pob:block:src/Modules/CalcSetup.lua:488-592
	if cachedPlayerDB == nil {
		classStats := env.Spec.Class()

		env.ModDB.AddMod(mod.NewFloat("Str", mod.TypeBase, float64(classStats.BaseStr)).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("Dex", mod.TypeBase, float64(classStats.BaseDex)).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("Int", mod.TypeBase, float64(classStats.BaseInt)).Source("Base"))

		env.ModDB.Multipliers["Level"] = max(1, min(100, float64(build.Build.Level)))
		initModDB(env, env.ModDB)

		env.ModDB.AddMod(mod.NewFloat("Life", mod.TypeBase, 12).Source("Base").Tag(mod.Multiplier("Level").Base(38)))
		env.ModDB.AddMod(mod.NewFloat("Mana", mod.TypeBase, 6).Source("Base").Tag(mod.Multiplier("Level").Base(34)))
		env.ModDB.AddMod(mod.NewFloat("ManaRegen", mod.TypeBase, 0.0175).Source("Base").Tag(mod.PerStat(1, "Mana")))
		env.ModDB.AddMod(mod.NewFloat("Devotion", mod.TypeBase, 0).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("Evasion", mod.TypeBase, 15).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("Accuracy", mod.TypeBase, 2).Source("Base").Tag(mod.Multiplier("Level").Base(-2)))
		env.ModDB.AddMod(mod.NewFloat("CritMultiplier", mod.TypeBase, 50).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("DotMultiplier", mod.TypeBase, 50).Source("Base").Tag(mod.Condition("CriticalStrike")))
		addBaseResistancePenalty(env.ModDB, build)
		env.ModDB.AddMod(mod.NewFloat("TotemFireResist", mod.TypeBase, 40).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("TotemColdResist", mod.TypeBase, 40).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("TotemLightningResist", mod.TypeBase, 40).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("TotemChaosResist", mod.TypeBase, 20).Source("Base"))
		//pob:block:src/Modules/CalcSetup.lua:511
		env.ModDB.AddMod(mod.NewFloat("CritChance", mod.TypeIncrease, data.CriticalStrikeChancePlusPercentPerPowerCharge).Source("Base").Tag(mod.Multiplier("PowerCharge").Base(0)))
		//pob:block:src/Modules/CalcSetup.lua:512-513
		env.ModDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, data.BaseAttackSpeedPlusPercentPerFrenzyCharge).Source("Base").Flag(mod.MFlagAttack).Tag(mod.Multiplier("FrenzyCharge").Base(0)))
		env.ModDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, data.BaseCastSpeedPlusPercentPerFrenzyCharge).Source("Base").Flag(mod.MFlagCast).Tag(mod.Multiplier("FrenzyCharge").Base(0)))
		env.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, 4).Source("Base").Tag(mod.Multiplier("FrenzyCharge").Base(0)))
		//pob:block:src/Modules/CalcSetup.lua:515-516
		env.ModDB.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, data.PhysicalDamageReductionPercentPerEnduranceCharge).Source("Base").Tag(mod.Multiplier("EnduranceCharge").Base(0)))
		env.ModDB.AddMod(mod.NewFloat("ElementalDamageReduction", mod.TypeBase, data.ElementalDamageReductionPercentPerEnduranceCharge).Source("Base").Tag(mod.Multiplier("EnduranceCharge").Base(0)))
		//pob:block:src/Modules/CalcSetup.lua:517
		env.ModDB.AddMod(mod.NewFloat("MaximumRage", mod.TypeBase, data.MaximumRage).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("Multiplier:GaleForce", mod.TypeBase, 0).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("MaximumGaleForce", mod.TypeBase, 10).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("MaximumFortification", mod.TypeBase, 20).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("MaximumValour", mod.TypeBase, 50).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("Multiplier:IntensityLimit", mod.TypeBase, 3).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 2).Source("Base").Tag(mod.Multiplier("Rampage").Base(0).Limit(50).Div(20)))
		env.ModDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, 1).Source("Base").Tag(mod.Multiplier("Rampage").Base(0).Limit(50).Div(20)))
		env.ModDB.AddMod(mod.NewFloat("ActiveTrapLimit", mod.TypeBase, 15).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("ActiveMineLimit", mod.TypeBase, 15).Source("Base"))
		//pob:block:src/Modules/CalcSetup.lua:527-528
		env.ModDB.AddMod(mod.NewFloat("MineThrowCount", mod.TypeBase, 1).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("TrapThrowCount", mod.TypeBase, 1).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("ActiveBrandLimit", mod.TypeBase, 3).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("EnemyCurseLimit", mod.TypeBase, 1).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("SocketedCursesHexLimitValue", mod.TypeBase, 1).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("ProjectileCount", mod.TypeBase, 1).Source("Base"))
		//pob:block:src/Modules/CalcSetup.lua:533-536
		env.ModDB.AddMod(mod.NewFloat("Speed", mod.TypeMore, data.DualWieldInherentAttackSpeedPlusPercentFinal).Source("Base").Flag(mod.MFlagAttack).Tag(mod.Condition("DualWielding")).Tag(mod.Condition("DoubledInherentDualWieldingSpeed").Neg(true)))
		env.ModDB.AddMod(mod.NewFloat("Speed", mod.TypeMore, 2*data.DualWieldInherentAttackSpeedPlusPercentFinal).Source("Base").Flag(mod.MFlagAttack).Tag(mod.Condition("DualWielding")).Tag(mod.Condition("DoubledInherentDualWieldingSpeed")))
		env.ModDB.AddMod(mod.NewFloat("BlockChance", mod.TypeBase, data.InherentBlockWhileDualWieldingPercent).Source("Base").Tag(mod.Condition("DualWielding")).Tag(mod.Condition("NoInherentBlock").Neg(true)).Tag(mod.Condition("DoubledInherentDualWieldingBlock").Neg(true)))
		env.ModDB.AddMod(mod.NewFloat("BlockChance", mod.TypeBase, 2*data.InherentBlockWhileDualWieldingPercent).Source("Base").Tag(mod.Condition("DualWielding")).Tag(mod.Condition("NoInherentBlock").Neg(true)).Tag(mod.Condition("DoubledInherentDualWieldingBlock")))
		env.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, 200).Source("Base").KeywordFlag(mod.KeywordFlagBleed).Tag(mod.ActorCondition("enemy", "Moving")).Tag(mod.Condition("NoExtraBleedDamageToMovingEnemy").Neg(true)))
		env.ModDB.AddMod(mod.NewFlag("Condition:BloodStance", true).Source("Base").Tag(mod.Condition("SandStance").Neg(true)))
		env.ModDB.AddMod(mod.NewFlag("Condition:PrideMinEffect", true).Source("Base").Tag(mod.Condition("PrideMaxEffect").Neg(true)))
		env.ModDB.AddMod(mod.NewFloat("PerBrutalTripleDamageChance", mod.TypeBase, 3).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("PerAfflictionAilmentDamage", mod.TypeBase, 8).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("PerAfflictionNonDamageEffect", mod.TypeBase, 8).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("PerAbsorptionElementalEnergyShieldRecoup", mod.TypeBase,
			data.ElementalDamageTakenGoesToEnergyShieldOver4SecondsPercentPerAbsorptionCharge).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("TinctureLimit", mod.TypeBase, 1).Source("Base"))
		env.ModDB.AddMod(mod.NewFloat("ManaDegenPercentTincture", mod.TypeBase, 1).Source("Base").
			Tag(mod.Multiplier("EffectiveManaBurnStacks")))
		env.ModDB.AddMod(mod.NewFloat("LifeDegenPercentTincture", mod.TypeBase, 1).Source("Base").
			Tag(mod.Multiplier("WeepingWoundsStacks")))
		env.ModDB.AddMod(mod.NewFloat("PresenceRadius", mod.TypeBase, data.BasePresenceRadius).Source("Base"))

		//	if     bandit == "Alira"   then modDB:NewMod("ElementalResist", "BASE", 15, "Bandit")
		//	elseif bandit == "Kraityn" then modDB:NewMod("MovementSpeed",   "INC",   8, "Bandit")
		//	elseif bandit == "Oak"     then modDB:NewMod("Life",            "BASE", 40, "Bandit")
		//	else                            modDB:NewMod("ExtraPoints",     "BASE",  1, "Bandit")
		//	end
		switch build.Build.Bandit {
		case "Alira":
			env.ModDB.AddMod(mod.NewFloat("ElementalResist", mod.TypeBase, 15).Source("Bandit"))
		case "Kraityn":
			env.ModDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, 8).Source("Bandit"))
		case "Oak":
			env.ModDB.AddMod(mod.NewFloat("Life", mod.TypeBase, 40).Source("Bandit"))
		default:
			env.ModDB.AddMod(mod.NewFloat("ExtraPoints", mod.TypeBase, 1).Source("Bandit"))
		}

		// Add Pantheon mods
		applySoulMod(env, build.Build.PantheonMajorGod)
		applySoulMod(env, build.Build.PantheonMinorGod)

		// Initialise enemy modifier database
		initModDB(env, env.EnemyModDB)
		//pob:block:src/Modules/CalcSetup.lua:575
		env.EnemyModDB.AddMod(mod.NewFloat("Accuracy", mod.TypeBase, data.MonsterAccuracyTable[env.EnemyLevel]).Source("Base"))
		env.EnemyModDB.AddMod(mod.NewFlag("Condition:AgainstDamageOverTime", true).Source("Base").
			Flag(mod.MFlagDot).Tag(mod.ActorCondition("player", "Combat")))

		// Add mods from the config tab
		confModList, confEnemyModList := buildConfigModLists(env, build)
		env.ModDB.AddList(confModList)
		env.EnemyModDB.AddList(confEnemyModList)

		// Add mods from the party tab
		//pob:block:src/Modules/CalcSetup.lua:583
		env.EnemyModDB.AddList(env.partyTab.enemyModList)

		if keepCached {
			cachedPlayerDB = env.ModDB.Clone()
			cachedEnemyDB = env.EnemyModDB.Clone()
			if env.Minion != nil {
				cachedMinionDB = env.Minion.ModDB.Clone()
			}
		}
	} else if specEnv.flatDB {
		copyModDB(env.ModDB, cachedPlayerDB)
		copyModDB(env.EnemyModDB, cachedEnemyDB)
		if cachedMinionDB != nil && env.Minion != nil {
			copyModDB(env.Minion.ModDB, cachedMinionDB)
		}
	} else {
		env.ModDB.Parent = cachedPlayerDB
		env.EnemyModDB.Parent = cachedEnemyDB
		if cachedMinionDB != nil && env.Minion != nil {
			env.Minion.ModDB.Parent = cachedMinionDB
		}
	}

	//pob:block:src/Modules/CalcSetup.lua:594-598
	for _, condition := range override.Conditions {
		env.ModDB.Conditions[condition] = true
	}

	//pob:block:src/Modules/CalcSetup.lua:600-605
	counts := env.specAllocCounts
	//pob:block:src/Modules/CalcSetup.lua:609-674
	if !accelerate.nodeAlloc {
		var tree = data.TreeVersions[data.LatestTreeVersion].Tree()
		env.AllocatedNodes = make(map[string]data.Node, len(env.Build.Build.PassiveNodes)+len(override.AddNodes))
		var clusterNodes map[string]data.Node
		if specEnv != nil && specEnv.clusterNodesSet {
			clusterNodes = specEnv.clusterNodes
		} else {
			clusterNodes = buildClusterJewelGraphs(env, data.LatestTreeVersion)
		}
		env.clusterNodes = clusterNodes
		resolveNodes := func(nodeIDs []int64, into map[string]data.Node) {
			for _, id := range nodeIDs {
				var strId = strconv.FormatInt(id, 10)
				if node, ok := clusterNodes[strId]; ok {
					into[strId] = node
					continue
				}
				into[strId] = resolveTreeNode(tree, strId)
			}
		}

		//pob:block:src/Modules/CalcSetup.lua:612-672
		nodeIDs := env.Build.Build.PassiveNodes
		overridesNodes := len(override.AddNodes) > 0 || len(override.RemoveNodes) > 0
		if overridesNodes {
			nodeIDs = make([]int64, 0, len(override.AddNodes)+len(env.Build.Build.PassiveNodes))
			nodeIDs = append(nodeIDs, override.AddNodes...)
			for _, id := range env.Build.Build.PassiveNodes {
				if !override.removesNode(id) {
					nodeIDs = append(nodeIDs, id)
				}
			}
		}
		resolveNodes(nodeIDs, env.AllocatedNodes)

		env.Spec.DistanceToClassStart = buildDistanceToClassStart(
			env.AllocatedNodes, classStartNodeID(tree, env.Spec.ClassID))

		masteryOverrides := tattooedNodes(env)
		counts = countAllocatedNodes(env, env.AllocatedNodes, masteryOverrides)
		env.Spec.AllocatedNotableCount = counts.notable
		env.Spec.AllocatedMasteryCount = counts.mastery

		env.specAllocCounts = counts
		if overridesNodes {
			specNodes := make(map[string]data.Node, len(env.Build.Build.PassiveNodes))
			resolveNodes(env.Build.Build.PassiveNodes, specNodes)
			env.specAllocCounts = countAllocatedNodes(env, specNodes, masteryOverrides)
		}
	}

	//pob:block:src/Modules/CalcSetup.lua:676-690
	if counts.notable > 0 {
		env.ModDB.AddMod(mod.NewFloat("Multiplier:AllocatedNotable", mod.TypeBase, float64(counts.notable)))
	}
	if counts.mastery > 0 {
		env.ModDB.AddMod(mod.NewFloat("Multiplier:AllocatedMastery", mod.TypeBase, float64(counts.mastery)))
	}
	if counts.keystone > 0 {
		env.ModDB.AddMod(mod.NewFloat("Multiplier:AllocatedKeystone", mod.TypeBase, float64(counts.keystone)))
	}
	if n := len(counts.masteryTypes); n > 0 {
		env.ModDB.AddMod(mod.NewFloat("Multiplier:AllocatedMasteryType", mod.TypeBase, float64(n)))
	}
	if n := counts.masteryTypes["Life Mastery"]; n > 0 {
		env.ModDB.AddMod(mod.NewFloat("Multiplier:AllocatedLifeMastery", mod.TypeBase, float64(n)))
	}

	/*
		TODO -- Build and merge item modifiers, and create list of radius jewels
		for _, slot in pairs(build.itemsTab.orderedSlots) do
			local slotName = slot.slotName
			local item
			if slotName == override.repSlotName then
				item = override.repItem
			elseif override.repItem and override.repSlotName:match("^Weapon 1") and slotName:match("^Weapon 2") and
			(override.repItem.base.type == "Staff" or override.repItem.base.type == "Two Handed Sword" or override.repItem.base.type == "Two Handed Axe" or override.repItem.base.type == "Two Handed Mace"
			or (override.repItem.base.type == "Bow" and item and item.base.type ~= "Quiver")) then
				item = nil
			elseif slot.nodeId and override.spec then
				item = build.itemsTab.items[env.spec.jewels[slot.nodeId]]
			else
				item = build.itemsTab.items[slot.selItemId]
			end
			if item then
				-- Find skills granted by this item
				for _, skill in ipairs(item.grantedSkills) do
					local grantedSkill = copyTable(skill)
					grantedSkill.sourceItem = item
					grantedSkill.slotName = slotName
					t_insert(env.grantedSkillsItems, grantedSkill)
				end
			end
			if slot.weaponSet and slot.weaponSet ~= (build.itemsTab.activeItemSet.useSecondWeaponSet and 2 or 1) then
				item = nil
			end
			if slot.weaponSet == 2 and build.itemsTab.activeItemSet.useSecondWeaponSet then
				slotName = slotName:gsub(" Swap","")
			end
			-- (the radius-jewel half of this block is ported below, as
			--  buildRadiusJewelList)
			if item and item.type == "Flask" then
				if slot.active then
					env.flasks[item] = true
				end
				if item.base.subType == "Life" then
					local highestLifeRecovery = env.itemModDB.multipliers["LifeFlaskRecovery"] or 0
					if item.flaskData.lifeTotal > highestLifeRecovery then
						env.itemModDB.multipliers["LifeFlaskRecovery"] = item.flaskData.lifeTotal
					end
				end
				item = nil
			end
			local scale = 1
			if item and item.type == "Jewel" and item.base.subType == "Abyss" and slot.parentSlot then
				-- Check if the item in the parent slot has enough Abyssal Sockets
				local parentItem = env.player.itemList[slot.parentSlot.slotName]
				if not parentItem or parentItem.abyssalSocketCount < slot.slotNum then
					item = nil
				else
					scale = parentItem.socketedJewelEffectModifier
				end
			end
			if slot.nodeId and item and item.type == "Jewel" and item.jewelData and item.jewelData.jewelIncEffectFromClassStart then
				local node = env.spec.nodes[slot.nodeId]
				if node and node.distanceToClassStart then
					scale = scale + node.distanceToClassStart * (item.jewelData.jewelIncEffectFromClassStart / 100)
				end
			end
			if item then
				env.player.itemList[slotName] = item
				-- Merge mods for this item
				local srcList = item.modList or item.slotModList[slot.slotNum]
				if item.requirements and not accelerate.requirementsItems then
					t_insert(env.requirementsTableItems, {
						source = "Item",
						sourceItem = item,
						sourceSlot = slotName,
						Str = item.requirements.strMod,
						Dex = item.requirements.dexMod,
						Int = item.requirements.intMod,
					})
				end
				if item.type == "Jewel" and item.base.subType == "Abyss" then
					-- Update Abyss Jewel conditions/multipliers
					local cond = "Have"..item.baseName:gsub(" ","")
					if not env.itemModDB.conditions[cond] then
						env.itemModDB.conditions[cond] = true
						env.itemModDB.multipliers["AbyssJewelType"] = (env.itemModDB.multipliers["AbyssJewelType"] or 0) + 1
					end
					if slot.parentSlot then
						env.itemModDB.conditions[cond.."In"..slot.parentSlot.slotName] = true
					end
					env.itemModDB.multipliers["AbyssJewel"] = (env.itemModDB.multipliers["AbyssJewel"] or 0) + 1
					env.itemModDB.multipliers[item.baseName:gsub(" ","")] = (env.itemModDB.multipliers[item.baseName:gsub(" ","")] or 0) + 1
				end
				if item.type == "Shield" and env.allocNodes[45175] and env.allocNodes[45175].dn == "Necromantic Aegis" then
					-- Special handling for Necromantic Aegis
					env.aegisModList = new("ModList")
					for _, mod in ipairs(srcList) do
						-- Filter out mods that apply to socketed gems, or which add supports
						local add = true
						for _, tag in ipairs(mod) do
							if tag.type == "SocketedIn" then
								add = false
								break
							end
						end
						if add then
							env.aegisModList:ScaleAddMod(mod, scale)
						else
							env.itemModDB:ScaleAddMod(mod, scale)
						end
					end
				elseif (slotName == "Weapon 1" or slotName == "Weapon 2") and modDB:Flag(nil, "Condition:EnergyBladeActive") then
					local type = env.player.itemList[slotName] and env.player.itemList[slotName].weaponData and env.player.itemList[slotName].weaponData[1].type
					local info = env.data.weaponTypeInfo[type]
					if info and type ~= "Bow" then
						local name = info.oneHand and "Energy Blade One Handed" or "Energy Blade Two Handed"
						local item = new("Item")
						item.name = name
						item.base = data.itemBases[name]
						item.baseName = name
						item.classRequirementModLines = { }
						item.buffModLines = { }
						item.enchantModLines = { }
						item.scourgeModLines = { }
						item.implicitModLines = { }
						item.explicitModLines = { }
						item.quality = 0
						item.rarity = "NORMAL"
						if item.baseName.implicit then
							local implicitIndex = 1
							for line in item.baseName.implicit:gmatch("[^\n]+") do
								local modList, extra = modLib.parseMod(line)
								t_insert(item.implicitModLines, { line = line, extra = extra, modList = modList or { }, modTags = item.baseName.implicitModTypes and item.baseName.implicitModTypes[implicitIndex] or { } })
								implicitIndex = implicitIndex + 1
							end
						end
						item:NormaliseQuality()
						item:BuildAndParseRaw()
						env.player.itemList[slotName] = item
					else
						env.itemModDB:ScaleAddList(srcList, scale)
					end
				elseif slotName == "Weapon 1" and item.name == "The Iron Mass, Gladius" then
					-- Special handling for The Iron Mass
					env.theIronMass = new("ModList")
					for _, mod in ipairs(srcList) do
						-- Filter out mods that apply to socketed gems, or which add supports
						local add = true
						for _, tag in ipairs(mod) do
							if tag.type == "SocketedIn" then
								add = false
								break
							end
						end
						if add then
							env.theIronMass:ScaleAddMod(mod, scale)
						end
						-- Add all the stats to player as well
						env.itemModDB:ScaleAddMod(mod, scale)
					end
				elseif slotName == "Weapon 1" and item.grantedSkills[1] and item.grantedSkills[1].skillId == "UniqueAnimateWeapon" then
					-- Special handling for The Dancing Dervish
					env.weaponModList1 = new("ModList")
					for _, mod in ipairs(srcList) do
						-- Filter out mods that apply to socketed gems, or which add supports
						local add = true
						for _, tag in ipairs(mod) do
							if tag.type == "SocketedIn" then
								add = false
								break
							end
						end
						if add then
							env.weaponModList1:ScaleAddMod(mod, scale)
						else
							env.itemModDB:ScaleAddMod(mod, scale)
						end
					end
				else
					env.itemModDB:ScaleAddList(srcList, scale)
				end
				-- set conditions on restricted items
				if item.classRestriction then
					env.itemModDB.conditions[item.title:gsub(" ", "")] = item.classRestriction
				end
				if item.type ~= "Jewel" and item.type ~= "Flask" then
					-- Update item counts
					local key
					if item.rarity == "UNIQUE" or item.rarity == "RELIC" then
						key = "UniqueItem"
					elseif item.rarity == "RARE" then
						key = "RareItem"
					elseif item.rarity == "MAGIC" then
						key = "MagicItem"
					else
						key = "NormalItem"
					end
					env.itemModDB.multipliers[key] = (env.itemModDB.multipliers[key] or 0) + 1
					env.itemModDB.conditions[key .. "In" .. slotName] = true
					if item.corrupted then
						env.itemModDB.multipliers.CorruptedItem = (env.itemModDB.multipliers.CorruptedItem or 0) + 1
					else
						env.itemModDB.multipliers.NonCorruptedItem = (env.itemModDB.multipliers.NonCorruptedItem or 0) + 1
					end
					if item.shaper then
						env.itemModDB.multipliers.ShaperItem = (env.itemModDB.multipliers.ShaperItem or 0) + 1
						env.itemModDB.conditions["ShaperItemIn"..slotName] = true
					else
						env.itemModDB.multipliers.NonShaperItem = (env.itemModDB.multipliers.NonShaperItem or 0) + 1
					end
					if item.elder then
						env.itemModDB.multipliers.ElderItem = (env.itemModDB.multipliers.ElderItem or 0) + 1
						env.itemModDB.conditions["ElderItemIn"..slotName] = true
					else
						env.itemModDB.multipliers.NonElderItem = (env.itemModDB.multipliers.NonElderItem or 0) + 1
					end
					if item.shaper or item.elder then
						env.itemModDB.multipliers.ShaperOrElderItem = (env.itemModDB.multipliers.ShaperOrElderItem or 0) + 1
					end
				end
			end
		end
	*/

	/*
		TODO -- Merge env.itemModDB with env.ModDB
		mergeDB(env.modDB, env.itemModDB)
	*/

	if !accelerate.nodeAlloc {
		env.InitialNodeModDB, _ = buildModListForNodeList(env, env.AllocatedNodes, true)
		for _, m := range override.AddMods {
			env.InitialNodeModDB.AddMod(m)
		}
		//pob:src/Modules/CalcSetup.lua:673
		mergeKeystones(env, env.InitialNodeModDB)
	}

	var itemExplodeSources []ExplodeSource
	//pob:block:src/Modules/CalcSetup.lua:698
	if !accelerate.requirementsItems {
		mergeEquippedItems(env)
		env.Flasks = toggleItem(env.Flasks, override.ToggleFlask)
		env.Tinctures = toggleItem(env.Tinctures, override.ToggleTincture)
		itemExplodeSources = collectItemExplodeSources(env, currentTreeVersion)
	}
	mergeDB(env.ModDB, env.ItemModDB)

	//pob:block:src/Modules/CalcSetup.lua:1286
	if !accelerate.nodeAlloc {
		allocateGrantedPassives(env, currentTreeVersion)
	}
	allocateGrantedAscendancyNodes(env, currentTreeVersion)

	explodeSources := itemExplodeSources
	if !accelerate.everything || env.nodeModList == nil {
		nodeModList, nodeExplodeSources := buildModListForNodeList(env, env.AllocatedNodes, true)
		for _, m := range override.AddMods {
			nodeModList.AddMod(m)
		}
		env.nodeModList = nodeModList
		//pob:block:src/Modules/CalcSetup.lua:1321
		explodeSources = append(nodeExplodeSources, itemExplodeSources...)
	}
	env.ModDB.AddList(env.nodeModList)

	/*
		TODO -- Find skills granted by tree nodes
		for _, node in pairs(env.allocNodes) do
			for _, skill in ipairs(node.grantedSkills) do
				local grantedSkill = copyTable(skill)
				grantedSkill.sourceNode = node
				t_insert(env.grantedSkillsNodes, grantedSkill)
			end
		end
	*/

	/*
		TODO -- Merge Granted Skills Tables
		env.grantedSkills = tableConcat(env.grantedSkillsNodes, env.grantedSkillsItems)
	*/

	//pob:block:src/Modules/CalcSetup.lua:1349
	if !accelerate.skills {
		processExplodeSocketGroup(build, explodeSources)

		if env.Mode == OutputModeMain {
			/*
				TODO -- Process extra skills granted by items or tree nodes
				local markList = wipeTable(tempTable1)
				for _, grantedSkill in ipairs(env.grantedSkills) do
					TODO -- Check if a matching group already exists
					local group
					for index, socketGroup in pairs(build.skillsTab.socketGroupList) do
						if socketGroup.source == grantedSkill.source and socketGroup.slot == grantedSkill.slotName then
							if socketGroup.gemList[1] and socketGroup.gemList[1].skillId == grantedSkill.skillId and socketGroup.gemList[1].level == grantedSkill.level then
								group = socketGroup
								markList[socketGroup] = true
								break
							end
						end
					end
					if not group then
						TODO -- Create a new group for this skill
						group = { label = "", enabled = true, gemList = { }, source = grantedSkill.source, slot = grantedSkill.slotName }
						t_insert(build.skillsTab.socketGroupList, group)
						markList[group] = true
					end

					TODO -- Update the group
					group.sourceItem = grantedSkill.sourceItem
					group.sourcacceleeNode = grantedSkill.sourceNode
					local activeGemInstance = group.gemList[1] or {
						skillId = grantedSkill.skillId,
						quality = 0,
						enabled = true,
					}
					activeGemInstance.gemId = nil
					activeGemInstance.level = grantedSkill.level
					activeGemInstance.enableGlobal1 = true
					if grantedSkill.triggered then
						activeGemInstance.triggered = grantedSkill.triggered
					end
					wipeTable(group.gemList)
					t_insert(group.gemList, activeGemInstance)
					if grantedSkill.noSupports then
						group.noSupports = true
					else
						for _, socketGroup in pairs(build.skillsTab.socketGroupList) do
							TODO -- Look for other groups that are socketed in the item
							if socketGroup.slot == grantedSkill.slotName and not socketGroup.source then
								TODO -- Add all support gems to the skill's group
								for _, gemInstance in ipairs(socketGroup.gemList) do
									if gemInstance.gemData and gemInstance.gemData.grantedEffect.support then
										t_insert(group.gemList, gemInstance)
									end
								end
							end
						end
					end
					build.skillsTab:ProcessSocketGroup(group)
				end

				TODO -- Remove any socket groups that no longer have a matching item
				local i = 1
				while build.skillsTab.socketGroupList[i] do
					local socketGroup = build.skillsTab.socketGroupList[i]
					if socketGroup.source and not markList[socketGroup] then
						t_remove(build.skillsTab.socketGroupList, i)
						if build.skillsTab.displayGroup == socketGroup then
							build.skillsTab.displayGroup = nil
						end
					else
						i = i + 1
					end
				end
			*/
		}

		assignActorWeaponData(env)

		if env.Mode == OutputModeCalcs && !alignCalcs {
			//pob:block:src/Modules/CalcSetup.lua:1483-1485
			selectedSkillSet := max(activeSkillSetIndex(build.Skills), 0)
			skillCount := 0
			if len(build.Skills.SkillSets) > selectedSkillSet {
				skillCount = len(build.Skills.SkillSets[selectedSkillSet].Skills)
			}
			env.MainSocketGroup = min(max(skillCount, 1), int(build.CalcsInputNumber("skill_number", 1))) - 1
		} else {
			selectedSkillSet := activeSkillSetIndex(build.Skills)
			if selectedSkillSet < 0 {
				selectedSkillSet = 0
			}

			skillCount := 0
			if len(build.Skills.SkillSets) > selectedSkillSet {
				skillCount = len(build.Skills.SkillSets[selectedSkillSet].Skills)
			}
			env.MainSocketGroup = min(max(skillCount, 1), build.Build.MainSocketGroup) - 1
		}

		/*
			TODO -- Determine main skill group
			if env.mode == "CALCS" then
				env.calcsInput.skill_number = m_min(m_max(#build.skillsTab.socketGroupList, 1), env.calcsInput.skill_number or 1)
				env.mainSocketGroup = env.calcsInput.skill_number
			else
				build.mainSocketGroup = m_min(m_max(#build.skillsTab.socketGroupList, 1), build.mainSocketGroup or 1)
				env.mainSocketGroup = build.mainSocketGroup
			end
		*/

		groupCfg := &moddb.ListCfg{}

		selectedSkillSet := activeSkillSetIndex(build.Skills)
		if selectedSkillSet < 0 {
			selectedSkillSet = 0
		}

		//pob:block:src/Modules/CalcSetup.lua:998-1000
		crossLinkedSupportGroups := make(map[string]bool)
		//pob:block:src/Modules/CalcSetup.lua:1492-1496
		crossLinkedSupportSources := make(map[string][]string)
		for _, linkedMod := range env.ModDB.Mods["LinkedSupport"] {
			for _, tag := range linkedMod.Tags() {
				if socketedIn, ok := tag.(*mod.SocketedInTag); ok && socketedIn.SlotName != "" {
					crossLinkedSupportGroups[socketedIn.SlotName] = true
					if value, ok := linkedMod.Value().List().(mod.LinkedSupport); ok {
						crossLinkedSupportSources[value.TargetSlotName] = append(crossLinkedSupportSources[value.TargetSlotName], socketedIn.SlotName)
					}
				}
			}
		}

		var indexOrder []int
		if selectedSkillSet < len(build.Skills.SkillSets) {
			indexOrder = make([]int, 0)
			for i, socketGroup := range build.Skills.SkillSets[selectedSkillSet].Skills {
				if crossLinkedSupportGroups[strings.ReplaceAll(socketGroup.Slot, " Swap", "")] {
					indexOrder = append([]int{i}, indexOrder...)
				} else {
					indexOrder = append(indexOrder, i)
				}
			}
		}

		activeWeaponSet := 1
		useSecondWeaponSet := build.Items.UseSecondWeaponSet
		for _, itemSet := range build.Items.ItemSets {
			if itemSet.ID == strconv.Itoa(build.Items.ActiveItemSet) && itemSet.UseSecondWeaponSet != nil {
				useSecondWeaponSet = itemSet.UseSecondWeaponSet
			}
		}
		if useSecondWeaponSet != nil && *useSecondWeaponSet {
			activeWeaponSet = 2
		}

		var matchesSocket map[int]map[int]bool
		if selectedSkillSet < len(build.Skills.SkillSets) {
			matchesSocket = buildMatchesSocket(env, build.Skills.SkillSets[selectedSkillSet].Skills)
		}

		imbuedSupportBySlot := buildImbuedSupportBySlot(build.Skills.SkillSets[selectedSkillSet].Skills)

		slotSupportLists := collectSlotSupportLists(env, build.Skills.SkillSets[selectedSkillSet].Skills,
			indexOrder, env.MainSocketGroup, activeWeaponSet, matchesSocket, imbuedSupportBySlot)

		processedSockets := make(map[*pob.Gem]bool)
		reqOverride := make(map[*pob.Gem]*int)

		//pob:src/Modules/CalcSetup.lua:996
		for _, index := range indexOrder {
			socketGroupIndex := index
			socketGroup := &build.Skills.SkillSets[selectedSkillSet].Skills[index]
			socketGroupSkillList := make([]*ActiveSkill, 0)

			socketGroup.SlotEnabled = true
			if ws := slotWeaponSet(socketGroup.Slot); ws != 0 {
				socketGroup.SlotEnabled = ws == activeWeaponSet
			}
			if index == env.MainSocketGroup || (socketGroup.Enabled && socketGroup.SlotEnabled) {
				if socketGroup.Slot != "" {
					groupCfg.SlotName = strings.Replace(socketGroup.Slot, " Swap", "", -1)
				} else {
					groupCfg.SlotName = ""
				}

				propertyModList := gemPropertyList(env, groupCfg)

				supportList := make([]*GemEffect, 0)
				if !socketGroupIsSourced(socketGroup) {
					// Add extra supports from the item this group is socketed in
					for _, value := range utils.CastSlice[mod.ExtraSupport](env.ModDB.List(groupCfg, "ExtraSupport")) {
						grantedEffect := poe.GrantedEffectByID(value.SkillID)
						// Some skill gems share the same name as support gems, e.g. Barrage.
						// Since a support gem is expected here, if the first lookup returns a skill, then
						// prepending "Support" to the skillId will find the support version of the gem.
						if grantedEffect != nil && !grantedEffect.IsSupport {
							grantedEffect = poe.GrantedEffectByID("Support" + value.SkillID)
						}
						if grantedEffect != nil {
							env.markItemSupport(grantedEffect.ID)
							supportList = append(supportList, &GemEffect{
								GrantedEffect: &GrantedEffect{
									Raw:      grantedEffect,
									fromItem: true,
								},
								Level:   value.Level,
								Quality: 0,
							})
						}
					}
				}

				//pob:src/Modules/CalcSetup.lua:1557-1565
				if socketGroup.ImbuedSupport != nil && groupCfg.SlotName != "" {
					for _, effect := range imbuedSupportBySlot[groupCfg.SlotName] {
						supportList = append(supportList, &GemEffect{
							GrantedEffect: &GrantedEffect{Raw: effect},
							Level:         1,
							Quality:       0,
						})
					}
				}

				for gemIndex, gemInstance := range socketGroup.Gems {
					socketQuality := 0
					if matchesSocket[socketGroupIndex][gemIndex] {
						socketQuality = data.MatchingSocketQualityBonus
					}
					// Add support gems from this group
					if env.Mode == OutputModeMain {
					}

					if gemInstance.Enabled {
						gemData, bareGrantedEffect := resolveGemInstance(gemInstance)
						if gemData == nil && bareGrantedEffect == nil {
							continue
						}

						processGrantedEffect := func(grantedEffect *poe.GrantedEffect) {
							if grantedEffect == nil || !grantedEffect.IsSupport {
								return
							}

							supportEffect := newSupportEffect(env, gemInstance, gemData, grantedEffect, socketQuality,
								gemPropertyModListFor(env, groupCfg.SlotName, gemIndex, propertyModList))
							reqOverride[&socketGroup.Gems[gemIndex]] = supportEffect.req
							//pob:block:src/Modules/CalcSetup.lua:1592-1595
							if env.Mode == OutputModeMain {
								env.setDisplayEffect(socketGroupIndex, gemIndex, supportEffect)
							}
							if gemData != nil && !processedSockets[&socketGroup.Gems[gemIndex]] {
								processedSockets[&socketGroup.Gems[gemIndex]] = true
								applySocketMods(env, gemData, groupCfg, gemIndex+1)
							}
							supportList = addBestSupport(supportEffect, supportList, env.Mode)
						}

						_ = processGrantedEffect

						if bareGrantedEffect != nil {
							processGrantedEffect(bareGrantedEffect)
						} else if gemData != nil {
							processGrantedEffect(primaryGrantedEffectOf(gemData))
						}

						//pob:block:src/Modules/CalcSetup.lua:1620-1625
						processGrantedEffect(secondaryGrantedEffectOf(gemData, gemInstance.SkillID, gemInstance.VariantID))
					}
				}

				//pob:block:src/Modules/CalcSetup.lua:1692-1709
				if socketGroupIsSourced(socketGroup) {
					for _, slotSupport := range slotSupportLists[groupCfg.SlotName] {
						supportList = addBestSupport(slotSupport, supportList, env.Mode)
					}
				}

				//pob:block:src/Modules/CalcSetup.lua:1711-1730
				for _, crossLinkedSupportSlot := range crossLinkedSupportSources[groupCfg.SlotName] {
					for _, supportEffect := range slotSupportLists[crossLinkedSupportSlot] {
						supportList = addBestSupport(supportEffect, supportList, env.Mode)
					}
				}

				// Create active skills
				for gemIndex, gemInstance := range socketGroup.Gems {
					socketQuality := 0
					if matchesSocket[socketGroupIndex][gemIndex] {
						socketQuality = data.MatchingSocketQualityBonus
					}
					gemData, bareGrantedEffect := resolveGemInstance(gemInstance)
					if gemData == nil && bareGrantedEffect == nil {
						continue
					}

					var grantedEffectList []*poe.GrantedEffect
					if bareGrantedEffect != nil {
						grantedEffectList = []*poe.GrantedEffect{bareGrantedEffect}
					} else if gemData != nil {
						grantedEffectList = raw2.GrantedEffectsForGem(gemData)
					}

					//pob:block:src/Modules/Data.lua:1078-1081
					if secondary := secondaryGrantedEffectOf(gemData, gemInstance.SkillID, gemInstance.VariantID); secondary != nil {
						present := false
						for _, existing := range grantedEffectList {
							if existing == secondary {
								present = true
								break
							}
						}
						if !present {
							grantedEffectList = append(grantedEffectList, secondary)
						}
					}

					if gemInstance.Enabled && grantedEffectList != nil && len(grantedEffectList) > 0 {
						for index, grantedEffect := range grantedEffectList {
							globalEnable := gemInstance.EnableGlobal1
							if index == 1 {
								globalEnable = gemInstance.EnableGlobal2
							}

							if !grantedEffect.IsSupport && (!raw2.SkillHasGlobalEffect(grantedEffect.ID) || globalEnable) {
								skillTypes := grantedEffectSkillTypes(grantedEffect)

								temp := gemInstance
								activeEffect := &GemEffect{
									GrantedEffect: &GrantedEffect{
										Raw:        grantedEffect,
										Parts:      skillPartsFor(grantedEffect.ID),
										SkillTypes: skillTypes,
										Funcs:      skillFuncsFor(grantedEffect.ID),
									},
									Level:         gemInstance.Level,
									Quality:       gemInstance.Quality + socketQuality,
									SocketQuality: socketQuality,
									QualityID:     gemInstance.QualityID,
									SrcInstance:   &temp,
									GemData:       gemData,
									CostEmptied:   itemGrantedSkillTriggered(env, socketGroup, gemInstance),
								}
								activeEffect.triggerChance = itemGrantedSkillTriggerChance(env, socketGroup, gemInstance)
								//pob:block:src/Modules/CalcSetup.lua:1433-1440
								if socketGroup.Source == "Explode" && gemIndex < len(explodeSources) {
									activeEffect.ExplodeSource = &explodeSources[gemIndex]
									activeEffect.CostEmptied = true
								}

								if gemData != nil && !processedSockets[&socketGroup.Gems[gemIndex]] {
									processedSockets[&socketGroup.Gems[gemIndex]] = true
									applySocketMods(env, gemData, groupCfg, gemIndex+1)
								}

								if gemData != nil {
									//pob:block:src/Modules/CalcSetup.lua:1669-1671
									gemCfg := *groupCfg
									if item := env.Player.ItemList[groupCfg.SlotName]; item != nil && gemIndex < len(item.SocketColors) {
										gemCfg.SocketColor = item.SocketColors[gemIndex]
									}
									gemCfg.SocketNum = gemIndex + 1
									activeEffect.gemCfg = &gemCfg
									for _, value := range gemPropertyModListFor(env, groupCfg.SlotName, gemIndex, propertyModList) {
										match := false
										if len(value.KeywordList) > 0 {
											match = true
											for _, keyword := range value.KeywordList {
												if !CalcGemIsTypeVariant(activeEffect.GemData, grantedEffect.ID, keyword) {
													match = false
													break
												}
											}
										} else {
											match = CalcGemIsTypeVariant(activeEffect.GemData, grantedEffect.ID, *value.Keyword)
										}

										if match {
											applyGemProperty(activeEffect, value)
											//pob:block:src/Modules/CalcSetup.lua:323-324
											activeEffect.gemPropertyInfo = append(activeEffect.gemPropertyInfo, value)
										}
									}
									reqOverride[&socketGroup.Gems[gemIndex]] = activeEffect.req
								}

								//pob:block:src/Modules/CalcSetup.lua:1683-1685
								if env.Mode == OutputModeMain {
									env.setDisplayEffect(socketGroupIndex, gemIndex, activeEffect)
								}

								appliedSupportList := supportList
								//pob:block:src/Modules/CalcSetup.lua:1686-1688
								if itemGrantedSkillNoSupports(env, socketGroup) {
									appliedSupportList = []*GemEffect{}
								}
								activeSkill := CreateActiveSkill(activeEffect, appliedSupportList, env.Player, socketGroup, nil)
								activeSkill.SocketGroupIndex = socketGroupIndex + 1
								if gemData != nil {
									activeSkill.SlotName = groupCfg.SlotName
								}

								socketGroupSkillList = append(socketGroupSkillList, activeSkill)
								env.Player.ActiveSkillList = append(env.Player.ActiveSkillList, activeSkill)
							}
						}

						if gemData != nil {
							grantedEffect := bareGrantedEffect
							if grantedEffect == nil {
								grantedEffect = primaryGrantedEffectOf(gemData)
							}
							reqLevel := float64(0)
							isSupport := false
							if grantedEffect != nil {
								isSupport = grantedEffect.IsSupport
								if level, ok := raw2.GetCalculatedGrantedEffect(grantedEffect).GetCalculatedLevels()[gemInstance.Level]; ok {
									reqLevel = float64(level.LevelRequirement)
								}
							}
							reqStr, reqDex, reqInt := 0.0, 0.0, 0.0
							if pobGem, ok := pobGemForVariant(gemData, gemInstance.SkillID); ok {
								reqStr = float64(pobGem.ReqStr)
								reqDex = float64(pobGem.ReqDex)
								reqInt = float64(pobGem.ReqInt)
							}
							req := &RequirementsTable{
								Source:    "Gem",
								SourceGem: utils.Ptr(gemInstance),
								Str:       int(getGemStatRequirement(reqLevel, isSupport, reqStr)),
								Dex:       int(getGemStatRequirement(reqLevel, isSupport, reqDex)),
								Int:       int(getGemStatRequirement(reqLevel, isSupport, reqInt)),
							}
							//pob:block:src/Modules/CalcSetup.lua:1745-1747
							if override := reqOverride[&socketGroup.Gems[gemIndex]]; override != nil {
								req.Str, req.Dex, req.Int = *override, *override, *override
							}
							env.RequirementsTableGems = append(env.RequirementsTableGems, req)
						}
					}
				}

				if index == env.MainSocketGroup && len(socketGroupSkillList) > 0 {
					// Select the main skill from this socket group
					activeSkillIndex := 0
					if env.Mode == OutputModeCalcs && !alignCalcs {
						activeSkillIndex = min(len(socketGroupSkillList)-1, max(1, socketGroup.MainActiveSkillCalcs)-1)
						socketGroup.MainActiveSkillCalcs = activeSkillIndex + 1
					} else {
						activeSkillIndex = min(len(socketGroupSkillList)-1, max(1, socketGroup.MainActiveSkill)-1)
						if env.Mode == OutputModeMain {
							socketGroup.MainActiveSkill = activeSkillIndex + 1
						}
					}
					env.Player.MainSkill = socketGroupSkillList[activeSkillIndex]
				}
			}

			if env.Mode == OutputModeMain {
				// Create display label for the socket group if the user didn't specify one
				if socketGroup.Label != "" {
					socketGroup.DisplayLabel = socketGroup.Label
				} else {
					DisplayLabel := ""
					for _, gemInstance := range socketGroup.Gems {
						baseItem := poe.BaseItemTypeByIDMap[gemInstance.GemID]
						if baseItem == nil {
							continue
						}
						gemData := baseItem.SkillGem()
						grantedEffect := primaryGrantedEffectOf(gemData)
						if grantedEffect != nil && !grantedEffect.IsSupport && gemInstance.Enabled {
							if activeSkill := grantedEffect.GetActiveSkill(); activeSkill != nil {
								if DisplayLabel != "" {
									DisplayLabel += ", "
								}
								DisplayLabel += activeSkill.DisplayedName
							}
						}
					}
					if DisplayLabel == "" {
						DisplayLabel = "<No active skills>"
					}
					socketGroup.DisplayLabel = DisplayLabel
				}

				// Save the active skill list for display in the socket group tooltip
				socketGroup.SetDisplaySkillList(socketGroupSkillList)
			} else if env.Mode == OutputModeCalcs {
				socketGroup.SetDisplaySkillListCalcs(socketGroupSkillList)
			}

			// Check for enabled energy blade to see if we need to regenerate everything.
			//pob:block:src/Modules/CalcSetup.lua:1812-1822
			if !env.ModDB.Conditions[conditionAffectedByEnergyBlade] && socketGroup.Enabled && socketGroup.SlotEnabled {
				for _, gemInstance := range socketGroup.Gems {
					_, grantedEffect := resolveGemInstance(gemInstance)
					if grantedEffect != nil && !grantedEffect.IsSupport && gemInstance.Enabled &&
						(&GrantedEffect{Raw: grantedEffect}).Name() == energyBlade {
						withBlade := *override
						withBlade.Conditions = append(append([]string{}, override.Conditions...), conditionAffectedByEnergyBlade)
						return initEnvSpecKeep(build, envCache, mode, alignCalcs, &withBlade, specEnv, keepCached)
					}
				}
			}
		}

		if env.Player.MainSkill == nil {
			// Add a default main skill if none are specified
			playerMelee := poe.GrantedEffectByID("PlayerMelee")
			skillTypes := grantedEffectSkillTypes(playerMelee)
			defaultEffect := &GemEffect{
				GrantedEffect: &GrantedEffect{
					Raw:        playerMelee,
					Parts:      skillPartsFor(playerMelee.ID),
					SkillTypes: skillTypes,
					Funcs:      skillFuncsFor(playerMelee.ID),
				},
			}
			env.Player.MainSkill = CreateActiveSkill(defaultEffect, []*GemEffect{}, env.Player, nil, nil)
			env.Player.ActiveSkillList = append(env.Player.ActiveSkillList, env.Player.MainSkill)
		}

		// Build skill modifier lists
		for _, activeSkill := range env.Player.ActiveSkillList {
			CalcBuildActiveSkillModList(env, activeSkill)
		}
	} else {
		//pob:block:src/Modules/CalcSetup.lua:1841-1865
		for _, activeSkill := range env.Player.ActiveSkillList {
			resetAcceleratedSkillData(env, activeSkill)
		}
	}

	env.RequirementsTable = nil
	env.RequirementsTable = append(env.RequirementsTable, env.RequirementsTableItems...)
	env.RequirementsTable = append(env.RequirementsTable, env.RequirementsTableGems...)

	return env, cachedPlayerDB, cachedEnemyDB, cachedMinionDB
}

func socketGroupIsSourced(socketGroup *pob.Skill) bool {
	return socketGroup.SourceAttr != nil
}

// local gemId = data.gemForSkill[gemInstance.skillId]
//
//pob:block:src/Classes/SkillsTab.lua:1151-1160
func resolveGemInstance(gemInstance pob.Gem) (*poe.SkillGem, *poe.GrantedEffect) {
	if gemInstance.SkillID != "" {
		if grantedEffect := poe.GrantedEffectByID(gemInstance.SkillID); grantedEffect != nil {
			if baseItemType := poe.BaseItemTypeByIDMap[gemInstance.GemID]; baseItemType != nil {
				if sg := baseItemType.SkillGem(); sg != nil {
					return sg, grantedEffect
				}
			}
			return nil, grantedEffect
		}
		if shell := pobOnlySkillEffect(gemInstance.SkillID); shell != nil {
			return nil, shell
		}
	}

	if baseItemType := poe.BaseItemTypeByIDMap[gemInstance.GemID]; baseItemType != nil {
		return baseItemType.SkillGem(), nil
	}

	//pob:block:src/Classes/SkillsTab.lua:1166-1174
	if gemInstance.GemID == "" && gemInstance.SkillID == "" && strings.TrimSpace(gemInstance.NameSpec) != "" {
		id, isVariant := findSkillGem(gemInstance.NameSpec)
		switch {
		case id == "":
		case isVariant:
			return resolveGemInstance(pob.Gem{SkillID: id})
		default:
			baseItemType := poe.BaseItemTypeByIDMap[id]
			if baseItemType == nil {
				baseItemType = poe.BaseItemTypeByIDMap[strings.Replace(id, "/SkillGemSupport", "/SupportGem", 1)]
			}
			if baseItemType != nil {
				return baseItemType.SkillGem(), nil
			}
		}
	}

	return nil, nil
}

//pob:src/Classes/SkillsTab.lua:1076-1100
func findSkillGem(nameSpec string) (string, bool) {
	r := findSkillGemCached(nameSpec)
	return r.id, r.variant
}

func FindSkillGemError(nameSpec string) string {
	return findSkillGemCached(nameSpec).msg
}

func findSkillGemCached(nameSpec string) findSkillGemResult {
	if hit, ok := findSkillGemCache.Load(nameSpec); ok {
		return hit.(findSkillGemResult)
	}
	r := findSkillGemUncached(nameSpec)
	findSkillGemCache.Store(nameSpec, r)
	return r
}

type findSkillGemResult struct {
	id      string
	variant bool
	msg     string
}

var findSkillGemCache sync.Map

func findSkillGemUncached(nameSpec string) findSkillGemResult {
	caseless := func(a string) string { return "[" + strings.ToUpper(a) + strings.ToLower(a) + "]" }
	noSpace := strings.ReplaceAll(nameSpec, " ", "")
	patternList := []string{
		"^ " + luaGsubLetters(nameSpec, isLuaAlpha, caseless) + "$",
		"^" + luaGsubLetters(nameSpec, isLuaAlpha, func(a string) string { return " " + a + "%l+" }) + "$",
		"^ " + luaGsubLetters(noSpace, isLuaLower, func(a string) string { return "%l*" + a }) + "%l+$",
		"^" + luaGsubLetters(noSpace, isLuaAlpha, func(a string) string { return ".*" + a }),
		"^" + luaGsubLetters(noSpace, isLuaAlpha, func(a string) string { return ".*" + caseless(a) }),
	}
	for _, pattern := range patternList {
		re, err := regexp.Compile(luaPatternToRegexp(pattern))
		if err != nil {
			continue
		}
		foundID, foundName, foundVariant, found := "", "", false, false
		for _, table := range []struct {
			gems    map[string]raw2.PoBGem
			variant bool
		}{{raw2.Gems, false}, {raw2.GemVariants, true}} {
			for id, gemData := range table.gems {
				if re.MatchString(" " + gemData.Name) {
					if found {
						return findSkillGemResult{msg: "Ambiguous gem name '" + nameSpec + "': matches '" + foundName + "', '" + gemData.Name + "'"}
					}
					foundID, foundName, foundVariant, found = id, gemData.Name, table.variant, true
				}
			}
		}
		if found {
			return findSkillGemResult{id: foundID, variant: foundVariant}
		}
	}
	return findSkillGemResult{msg: "Unrecognised gem name '" + nameSpec + "'"}
}

func isLuaAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isLuaLower(c byte) bool { return c >= 'a' && c <= 'z' }

func luaGsubLetters(s string, match func(byte) bool, repl func(string) string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if match(s[i]) {
			b.WriteString(repl(s[i : i+1]))
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func luaPatternToRegexp(p string) string {
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch {
		case c == '^' && i == 0, c == '$' && i == len(p)-1, c == '.':
			b.WriteByte(c)
		case c == '%' && i+1 < len(p):
			i++
			switch p[i] {
			case 'l':
				b.WriteString("[a-z]")
			case 'a':
				b.WriteString("[A-Za-z]")
			default:
				b.WriteString(regexp.QuoteMeta(p[i : i+1]))
			}
		case c == '[':
			end := strings.IndexByte(p[i:], ']')
			if end < 0 {
				b.WriteString(regexp.QuoteMeta(p[i:]))
				return b.String()
			}
			b.WriteString(p[i : i+end+1])
			i += end
		default:
			b.WriteString(regexp.QuoteMeta(p[i : i+1]))
		}
		if i+1 < len(p) {
			switch p[i+1] {
			case '*', '+', '?':
				b.WriteByte(p[i+1])
				i++
			case '-':
				b.WriteString("*?")
				i++
			}
		}
	}
	return b.String()
}

//pob:block:src/Modules/CalcSetup.lua:1364-1375,src/Modules/CalcSetup.lua:1397
func itemGrantedSkillNoSupports(env *Environment, socketGroup *pob.Skill) bool {
	if socketGroup == nil || !strings.HasPrefix(socketGroup.Source, "Item:") || len(socketGroup.Gems) == 0 || socketGroup.Gems[0].SkillID == "" {
		return false
	}
	item := env.Player.ItemList[socketGroup.Slot]
	if item == nil {
		return false
	}
	for _, grantedSkill := range item.GrantedSkills {
		if grantedSkill.SkillID == socketGroup.Gems[0].SkillID {
			return grantedSkill.NoSupports
		}
	}
	return false
}

//pob:block:src/Modules/CalcSetup.lua:1364-1398
func itemGrantedSkillTriggered(env *Environment, socketGroup *pob.Skill, gem pob.Gem) bool {
	if socketGroup == nil || !strings.HasPrefix(socketGroup.Source, "Item:") || gem.SkillID == "" {
		return false
	}
	item := env.Player.ItemList[socketGroup.Slot]
	if item == nil {
		return false
	}
	for _, grantedSkill := range item.GrantedSkills {
		if grantedSkill.SkillID == gem.SkillID && grantedSkill.Triggered {
			return true
		}
	}
	return false
}

//pob:block:src/Modules/CalcSetup.lua:1399
func itemGrantedSkillTriggerChance(env *Environment, socketGroup *pob.Skill, gem pob.Gem) *float64 {
	if socketGroup == nil || !strings.HasPrefix(socketGroup.Source, "Item:") || gem.SkillID == "" {
		return nil
	}
	item := env.Player.ItemList[socketGroup.Slot]
	if item == nil {
		return nil
	}
	for _, grantedSkill := range item.GrantedSkills {
		if grantedSkill.SkillID == gem.SkillID && grantedSkill.triggerChance != 0 {
			return utils.Ptr(grantedSkill.triggerChance)
		}
	}
	return nil
}

//pob:block:src/Modules/CalcSetup.lua:701-711,src/Modules/CalcSetup.lua:735-737
func collectItemExplodeSources(env *Environment, treeVersion pob.TreeVersion) []ExplodeSource {
	items := env.Build.Items
	if len(items.ItemSets) == 0 {
		return nil
	}
	set := items.ItemSets[activeItemSetIndex(items)]
	slotsByName := make(map[string]pob.Slot, len(set.Slots))
	for _, slot := range set.Slots {
		slotsByName[slot.Name] = slot
	}
	byID := itemTextByID(env)

	var sources []ExplodeSource
	for _, s := range orderedItemSlots {
		if (s.name == "Graft 1" || s.name == "Graft 2") && !strings.Contains(string(treeVersion), "3_27") {
			continue
		}
		if s.name == "Ring 3" && (env.InitialNodeModDB == nil || !env.InitialNodeModDB.Flag(nil, "AdditionalRingSlot")) {
			continue
		}
		item, ok := peekSlotItem(env, s.name, slotsByName, byID)
		if !ok {
			continue
		}
		sources = appendItemExplodeSource(sources, item)
	}

	specs := env.Build.Tree.Specs
	if len(specs) > 0 {
		idx := env.Build.Tree.ActiveSpec - 1
		if idx < 0 || idx >= len(specs) {
			idx = 0
		}
		specSockets, socketItems := overrideSockets(env, specs[idx].Sockets, byID)
		sockets := append([]pob.Socket(nil), specSockets...)
		sort.SliceStable(sockets, func(a, b int) bool { return sockets[a].NodeID < sockets[b].NodeID })
		tree := data.TreeVersions[data.LatestTreeVersion].Tree()
		for _, socket := range sockets {
			node, ok := tree.Nodes[strconv.Itoa(socket.NodeID)]
			if !ok || node.IsJewelSocket == nil || !*node.IsJewelSocket || socket.ItemID == 0 {
				continue
			}
			if text, ok := socketItems[socket.ItemID]; ok {
				sources = appendItemExplodeSource(sources, peekItemCached(env.Cache, text))
			}
		}
	}
	return sources
}

//pob:block:src/Modules/CalcSetup.lua:1266-1279
func toggleItem(active []*Item, item *Item) []*Item {
	if item == nil {
		return active
	}
	for i, f := range active {
		if f.Raw == item.Raw {
			return slices.Delete(active, i, i+1)
		}
	}
	return append(active, item.Clone())
}

func appendItemExplodeSource(sources []ExplodeSource, item *Item) []ExplodeSource {
	if item == nil {
		return sources
	}
	for _, m := range item.Mods {
		if m.Name() == "CanExplode" && m.Type() == mod.TypeFlag {
			return append(sources, ExplodeSource{ModSource: "Item:-1:" + item.Name, Rarity: item.Rarity})
		}
	}
	return sources
}

//pob:block:src/Modules/CalcSetup.lua:1405-1446,src/Modules/CalcSetup.lua:1448-1460
func processExplodeSocketGroup(build *pob.PathOfBuilding, explodeSources []ExplodeSource) {
	setIndex := activeSkillSetIndex(build.Skills)
	if setIndex < 0 {
		return
	}
	groups := &build.Skills.SkillSets[setIndex].Skills
	groupIndex := -1
	for i := range *groups {
		if (*groups)[i].Source == "Explode" {
			groupIndex = i
			break
		}
	}

	if len(explodeSources) == 0 {
		if groupIndex >= 0 {
			*groups = append((*groups)[:groupIndex], (*groups)[groupIndex+1:]...)
		}
		return
	}

	if groupIndex < 0 {
		source := "Explode"
		*groups = append(*groups, pob.Skill{
			Label:      "On Kill Monster Explosion",
			Enabled:    true,
			SourceAttr: &source,
			Source:     source,
			Gems:       []pob.Gem{},
		})
		groupIndex = len(*groups) - 1
	}
	group := &(*groups)[groupIndex]

	gemsBySource := make(map[string]pob.Gem)
	for _, gem := range group.Gems {
		if source := gem.ExplodeSource(); source != "" {
			gemsBySource[source] = gem
		}
	}
	gems := make([]pob.Gem, 0, len(explodeSources))
	for _, explodeSource := range explodeSources {
		activeGemInstance, ok := gemsBySource[explodeSource.key()]
		if !ok {
			activeGemInstance = pob.Gem{SkillID: "EnemyExplode", Quality: 0, Enabled: true, Level: 1}
			activeGemInstance.SetExplodeSource(explodeSource.key())
		}
		gems = append(gems, activeGemInstance)
	}
	group.Gems = gems
}

//pob:block:src/Modules/CalcSetup.lua:1064-1071
func allocateGrantedPassives(env *Environment, treeVersion pob.TreeVersion) {
	granted := env.ModDB.List(nil, "GrantedPassive")
	if len(granted) == 0 {
		return
	}

	names := make([]string, 0, len(granted))
	for _, value := range granted {
		if name, ok := value.(string); ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	notableMap := data.TreeVersions[treeVersion].NotableMap()
	tree := data.TreeVersions[treeVersion].Tree()

	for _, name := range names {
		nodeID, ok := notableMap[strings.ToLower(name)]
		if !ok {
			continue
		}
		//pob:block:src/Modules/CalcSetup.lua:1291
		if env.override.removesNodeKey(nodeID) {
			continue
		}

		env.GrantedPassives[nodeID] = true

		if _, allocated := env.AllocatedNodes[nodeID]; allocated {
			continue
		}

		env.AllocatedNodes[nodeID] = tree.Nodes[nodeID]
	}
}

//pob:block:src/Modules/CalcSetup.lua:868-883
func allocateGrantedAscendancyNodes(env *Environment, treeVersion pob.TreeVersion) {
	granted := env.ModDB.List(nil, "GrantedAscendancyNode")
	if len(granted) == 0 {
		return
	}

	className := ""
	if env.Spec != nil {
		className = string(env.Spec.ClassName)
	}
	if className == "" ||
		env.ItemClassRestrictions["ForbiddenFlesh"] != className ||
		env.ItemClassRestrictions["ForbiddenFlame"] != className {
		return
	}

	entries := make([]mod.GrantedAscendancyNode, 0, len(granted))
	for _, value := range granted {
		if entry, ok := value.(mod.GrantedAscendancyNode); ok {
			entries = append(entries, entry)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Side < entries[j].Side
	})

	type seen struct {
		side    string
		matched bool
	}
	matchedName := make(map[string]*seen, len(entries))

	ascendancyMap := data.TreeVersions[treeVersion].AscendancyMap()
	tree := data.TreeVersions[treeVersion].Tree()

	for _, entry := range entries {
		name := strings.ToLower(entry.Name)
		prev, ok := matchedName[name]
		if !ok || prev.side == entry.Side || prev.matched {
			if !ok {
				matchedName[name] = &seen{side: entry.Side}
			}
			continue
		}
		prev.matched = true

		nodeID, ok := ascendancyMap[name]
		if !ok {
			nodeID, ok = missingTreeNodeIDByName[name]
			if !ok {
				continue
			}
		}
		//pob:block:src/Modules/CalcSetup.lua:1306
		if env.override.removesNodeKey(nodeID) {
			continue
		}

		env.GrantedPassives[nodeID] = true
		env.GrantedAscendancyNodes[nodeID] = true

		if _, allocated := env.AllocatedNodes[nodeID]; allocated {
			continue
		}

		env.AllocatedNodes[nodeID] = resolveTreeNode(tree, nodeID)
	}
}

// local socketedIn = playerItems[groupCfg.slotName] and playerItems[groupCfg.slotName].sockets
//
//pob:block:src/Modules/CalcSetup.lua:329-336
func applySocketMods(env *Environment, gem *poe.SkillGem, groupCfg *moddb.ListCfg, socketNum int) {
	socketCfg := *groupCfg
	socketCfg.SkillGem = gem
	socketCfg.SocketNum = socketNum

	modSource := groupCfg.SlotName
	if item := env.Player.ItemList[groupCfg.SlotName]; item != nil && item.Name != "" {
		modSource = item.Name
	}

	for _, value := range utils.CastSlice[mod.SocketProperty](env.ModDB.List(&socketCfg, "SocketProperty")) {
		if value.Mod == nil {
			continue
		}
		env.ModDB.AddMod(value.Mod.Clone().Source(mod.Source(modSource)))
	}
}

func gemPropertyModListFor(env *Environment, slotName string, gemIndex int, groupList []gemPropertyValue) []gemPropertyValue {
	item := env.Player.ItemList[slotName]
	if item == nil || gemIndex >= len(item.SocketColors) {
		return groupList
	}
	cfg := &moddb.ListCfg{SlotName: slotName, SocketColor: item.SocketColors[gemIndex]}
	return gemPropertyList(env, cfg)
}

type gemPropertyValue struct {
	mod.GemProperty
	socketed bool
	mod      mod.Mod
}

func gemPropertyList(env *Environment, cfg *moddb.ListCfg) []gemPropertyValue {
	values := utils.CastSlice[mod.GemProperty](env.ModDB.List(cfg, "GemProperty"))
	mods := env.ModDB.Tabulate(mod.TypeList, cfg, "GemProperty")
	out := make([]gemPropertyValue, len(values))
	for i, v := range values {
		out[i].GemProperty = v
		if len(mods) == len(values) {
			out[i].socketed = slices.ContainsFunc(mods[i].Mod.Tags(), func(t mod.Tag) bool { return t.Type() == mod.TypeSocketedIn })
			out[i].mod = mods[i].Mod
		}
	}
	return out
}

//pob:block:src/Modules/CalcSetup.lua:307-322
func applyGemProperty(effect *GemEffect, value gemPropertyValue) {
	switch value.Key {
	case "level":
		effect.Level += int(value.Value)
	case "req":
		req := int(value.Value)
		if effect.req != nil {
			req += *effect.req
		}
		effect.req = &req
	case "quality":
		effect.Quality += int(value.Value)
		if value.socketed {
			effect.ItemQuality += int(value.Value)
		} else {
			effect.GlobalQuality += int(value.Value)
		}
	}
}

//pob:src/Modules/PantheonTools.lua:6-19,src/Modules/CalcSetup.lua:563-571
func applySoulMod(env *Environment, godName string) {
	if godName == "" || godName == "None" {
		return
	}

	god, ok := data.Pantheons[godName]
	if !ok || len(god.Souls) == 0 {
		return
	}

	source := mod.Source("Pantheon:" + god.Souls[0].Name)
	for _, soul := range god.Souls {
		for _, line := range soul.Mods {
			entry := ParseMod(line, false)
			if entry.ModList == nil || strings.TrimSpace(entry.Extra) != "" {
				continue
			}
			for _, m := range entry.ModList {
				env.ModDB.AddMod(m.Source(source))
			}
		}
	}
}

//pob:block:src/Modules/CalcSetup.lua:1502-1630
func collectSlotSupportLists(env *Environment, skills []pob.Skill, indexOrder []int, mainSocketGroup int, activeWeaponSet int, matchesSocket map[int]map[int]bool, imbuedSupportBySlot map[string][]*poe.GrantedEffect) map[string][]*GemEffect {
	bySlot := map[string][]*GemEffect{}
	for _, index := range indexOrder {
		socketGroup := skills[index]
		if socketGroup.Slot == "" {
			continue
		}
		slotEnabled := true
		if ws := slotWeaponSet(socketGroup.Slot); ws != 0 {
			slotEnabled = ws == activeWeaponSet
		}
		if index != mainSocketGroup && !(socketGroup.Enabled && slotEnabled) {
			continue
		}

		slotName := strings.Replace(socketGroup.Slot, " Swap", "", -1)
		cfg := &moddb.ListCfg{SlotName: slotName}
		propertyModList := gemPropertyList(env, cfg)

		//pob:block:src/Modules/CalcSetup.lua:1550-1565
		if !socketGroupIsSourced(&socketGroup) {
			for _, value := range utils.CastSlice[mod.ExtraSupport](env.ModDB.List(cfg, "ExtraSupport")) {
				grantedEffect := poe.GrantedEffectByID(value.SkillID)
				if grantedEffect != nil && !grantedEffect.IsSupport {
					grantedEffect = poe.GrantedEffectByID("Support" + value.SkillID)
				}
				if grantedEffect != nil {
					env.markItemSupport(grantedEffect.ID)
					bySlot[slotName] = addBestSupport(&GemEffect{
						GrantedEffect: &GrantedEffect{Raw: grantedEffect, fromItem: true},
						Level:         value.Level,
					}, bySlot[slotName], env.Mode)
				}
			}
		}
		if socketGroup.ImbuedSupport != nil {
			for _, effect := range imbuedSupportBySlot[slotName] {
				bySlot[slotName] = addBestSupport(&GemEffect{
					GrantedEffect: &GrantedEffect{Raw: effect},
					Level:         1,
				}, bySlot[slotName], env.Mode)
			}
		}

		for gemIndex, gemInstance := range socketGroup.Gems {
			if !gemInstance.Enabled {
				continue
			}
			gemData, bareGrantedEffect := resolveGemInstance(gemInstance)
			if gemData == nil && bareGrantedEffect == nil {
				continue
			}
			socketQuality := 0
			if matchesSocket[index][gemIndex] {
				socketQuality = data.MatchingSocketQualityBonus
			}
			addIfSupport := func(grantedEffect *poe.GrantedEffect) {
				if grantedEffect == nil || !grantedEffect.IsSupport {
					return
				}
				supportEffect := newSupportEffect(env, gemInstance, gemData, grantedEffect, socketQuality,
					gemPropertyModListFor(env, slotName, gemIndex, propertyModList))
				//pob:block:src/Modules/CalcSetup.lua:1592-1595
				if env.Mode == OutputModeMain {
					env.setDisplayEffect(index, gemIndex, supportEffect)
				}
				bySlot[slotName] = addBestSupport(supportEffect, bySlot[slotName], env.Mode)
			}
			if bareGrantedEffect != nil {
				addIfSupport(bareGrantedEffect)
			} else if gemData != nil {
				addIfSupport(primaryGrantedEffectOf(gemData))
			}
			addIfSupport(secondaryGrantedEffectOf(gemData, gemInstance.SkillID, gemInstance.VariantID))
		}
	}
	return bySlot
}

//pob:block:src/Modules/CalcSetup.lua:1596-1620
func newSupportEffect(env *Environment, gemInstance pob.Gem, gemData *poe.SkillGem, grantedEffect *poe.GrantedEffect, socketQuality int, propertyModList []gemPropertyValue) *GemEffect {
	var temp = gemInstance
	supportEffect := &GemEffect{
		GrantedEffect: &GrantedEffect{
			Raw:          grantedEffect,
			itemSupports: env.itemSupportSet(),
		},
		Level:         gemInstance.Level,
		Quality:       gemInstance.Quality + socketQuality,
		SocketQuality: socketQuality,
		QualityID:     gemInstance.QualityID,
		SrcInstance:   &temp,
		GemData:       gemData,
		Superseded:    false,
		IsSupporting:  make(map[*pob.Gem]bool),
		Values:        make(map[string]float64),
	}

	if gemData != nil {
		for _, value := range propertyModList {
			match := true

			effectID := ""
			if grantedEffect != nil {
				effectID = grantedEffect.ID
			}
			if len(value.KeywordList) > 0 {
				for _, keyword := range value.KeywordList {
					if !CalcGemIsTypeVariant(supportEffect.GemData, effectID, keyword) {
						match = false
						break
					}
				}
			} else if !CalcGemIsTypeVariant(supportEffect.GemData, effectID, *value.Keyword) {
				match = false
			}

			if match {
				applyGemProperty(supportEffect, value)
			}
		}
	}

	// Validate support gem level in case there is no active skill (and no full calculation)
	CalcValidateGemLevel(supportEffect)
	return supportEffect
}

//pob:src/Modules/CalcSetup.lua:338-366
func addBestSupport(supportEffect *GemEffect, appliedSupportList []*GemEffect, mode OutputMode) []*GemEffect {
	grantedEffect := supportEffect.GrantedEffect.Raw
	add := true
	for index, otherSupport := range appliedSupportList {
		if grantedEffect == otherSupport.GrantedEffect.Raw {
			add = false
			if supportEffect.Level > otherSupport.Level || (supportEffect.Level == otherSupport.Level && supportEffect.Quality > otherSupport.Quality) {
				if mode == OutputModeMain {
					otherSupport.Superseded = true
				}
				appliedSupportList[index] = supportEffect
			} else {
				supportEffect.Superseded = true
			}
			break
		} else if grantedEffect.PlusVersionOf != nil && *grantedEffect.PlusVersionOf == otherSupport.GrantedEffect.Raw.Key {
			add = false
			if mode == OutputModeMain {
				otherSupport.Superseded = true
			}
			appliedSupportList[index] = supportEffect
		} else if otherSupport.GrantedEffect.Raw.PlusVersionOf != nil && *otherSupport.GrantedEffect.Raw.PlusVersionOf == grantedEffect.Key {
			add = false
			supportEffect.Superseded = true
		}
	}

	if add {
		appliedSupportList = append(appliedSupportList, supportEffect)
	}
	return appliedSupportList
}

//pob:src/Modules/CalcSetup.lua:18-80
func initModDB(env *Environment, modDB *moddb.ModDB) {
	modDB.AddDB(initModDBBase())

	modDB.Conditions["Buffed"] = env.ModeBuffs
	modDB.Conditions["Combat"] = env.ModeCombat
	modDB.Conditions["Effective"] = env.ModeEffective
}

var initModDBBase = sync.OnceValue(func() *moddb.ModDB {
	modDB := moddb.NewModDB()
	initModDBMods(modDB)
	return modDB
})

//pob:block:src/Modules/CalcSetup.lua:19-110
func initModDBMods(modDB *moddb.ModDB) {
	modDB.AddMod(mod.NewFloat("FireResistMax", mod.TypeBase, data.BaseMaximumAllResistancesPercent).Source("Base"))
	modDB.AddMod(mod.NewFloat("ColdResistMax", mod.TypeBase, data.BaseMaximumAllResistancesPercent).Source("Base"))
	modDB.AddMod(mod.NewFloat("LightningResistMax", mod.TypeBase, data.BaseMaximumAllResistancesPercent).Source("Base"))
	modDB.AddMod(mod.NewFloat("ChaosResistMax", mod.TypeBase, data.BaseMaximumAllResistancesPercent).Source("Base"))
	modDB.AddMod(mod.NewFloat("TotemFireResistMax", mod.TypeBase, data.BaseMaximumAllResistancesPercent).Source("Base"))
	modDB.AddMod(mod.NewFloat("TotemColdResistMax", mod.TypeBase, data.BaseMaximumAllResistancesPercent).Source("Base"))
	modDB.AddMod(mod.NewFloat("TotemLightningResistMax", mod.TypeBase, data.BaseMaximumAllResistancesPercent).Source("Base"))
	modDB.AddMod(mod.NewFloat("TotemChaosResistMax", mod.TypeBase, data.BaseMaximumAllResistancesPercent).Source("Base"))
	modDB.AddMod(mod.NewFloat("BlockChanceMax", mod.TypeBase, data.MaximumBlockPercent).Source("Base"))
	modDB.AddMod(mod.NewFloat("SpellBlockChanceMax", mod.TypeBase, data.BaseMaximumSpellBlockPercent).Source("Base"))
	modDB.AddMod(mod.NewFloat("SpellDodgeChanceMax", mod.TypeBase, 75).Source("Base"))
	modDB.AddMod(mod.NewFloat("ChargeDuration", mod.TypeBase, 10).Source("Base"))
	modDB.AddMod(mod.NewFloat("PowerChargesMax", mod.TypeBase, data.MaxPowerCharges).Source("Base"))
	modDB.AddMod(mod.NewFloat("FrenzyChargesMax", mod.TypeBase, data.MaxFrenzyCharges).Source("Base"))
	modDB.AddMod(mod.NewFloat("EnduranceChargesMax", mod.TypeBase, data.MaxEnduranceCharges).Source("Base"))
	modDB.AddMod(mod.NewFloat("SiphoningChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("ChallengerChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("BlitzChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("InspirationChargesMax", mod.TypeBase, data.MaximumRighteousCharges).Source("Base"))
	modDB.AddMod(mod.NewFloat("CrabBarriersMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("BrutalChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("AbsorptionChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("AfflictionChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("BloodChargesMax", mod.TypeBase, data.MaximumBloodScytheCharges).Source("Base"))
	modDB.AddMod(mod.NewFloat("BrineChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("PhysicalDamageGainAsCold", mod.TypeBase, data.PhysicalDamagePercentToAddAsColdPerBrineCharge).
		Source("Base").Tag(mod.Multiplier("BrineCharge")))
	modDB.AddMod(mod.NewFloat("PhysicalDamageGainAsLightning", mod.TypeBase, data.PhysicalDamagePercentToAddAsLightningPerBrineCharge).
		Source("Base").Tag(mod.Multiplier("BrineCharge")))
	modDB.AddMod(mod.NewFloat("MaxLifeLeechRate", mod.TypeBase, data.MaximumLifeLeechRatePercentPerMinute/60).Source("Base"))
	modDB.AddMod(mod.NewFloat("MaxManaLeechRate", mod.TypeBase, data.MaximumManaLeechRatePercentPerMinute/60).Source("Base"))
	modDB.AddMod(mod.NewFloat("ImpaleStacksMax", mod.TypeBase, data.ImpaledDebuffNumberOfReflectedHits).Source("Base"))
	modDB.AddMod(mod.NewFloat("SoulEaterMax", mod.TypeBase, data.SoulEaterMaximumStacks).Source("Base"))
	modDB.AddMod(mod.NewFloat("BleedStacksMax", mod.TypeBase, 1).Source("Base"))
	modDB.AddMod(mod.NewFloat("MaxEnergyShieldLeechRate", mod.TypeBase, 10).Source("Base"))
	modDB.AddMod(mod.NewFloat("MaxLifeLeechInstance", mod.TypeBase, data.MaximumLifeLeechAmountPerLeechPercentMaxLife).Source("Base"))
	modDB.AddMod(mod.NewFloat("MaxManaLeechInstance", mod.TypeBase, data.MaximumManaLeechAmountPerLeechPercentMaxMana).Source("Base"))
	modDB.AddMod(mod.NewFloat("MaxEnergyShieldLeechInstance", mod.TypeBase, data.MaximumEnergyShieldLeechAmountPerLeechPercentMaxEnergyShield).Source("Base"))
	modDB.AddMod(mod.NewFloat("TrapThrowingTime", mod.TypeBase, 0.6).Source("Base"))
	modDB.AddMod(mod.NewFloat("MineLayingTime", mod.TypeBase, 0.3).Source("Base"))
	modDB.AddMod(mod.NewFloat("WarcryCastTime", mod.TypeBase, 0.8).Source("Base"))
	modDB.AddMod(mod.NewFloat("TotemPlacementTime", mod.TypeBase, 0.6).Source("Base"))
	modDB.AddMod(mod.NewFloat("BallistaPlacementTime", mod.TypeBase, 0.5).Source("Base"))
	modDB.AddMod(mod.NewFloat("ActiveTotemLimit", mod.TypeBase, data.BaseNumberOfTotemsAllowed).Source("Base"))
	modDB.AddMod(mod.NewFloat("ShockStacksMax", mod.TypeBase, 1).Source("Base"))
	modDB.AddMod(mod.NewFloat("ScorchStacksMax", mod.TypeBase, 1).Source("Base"))

	modDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, -30).Source("Base").Tag(mod.Condition("Maimed")))

	modDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, -10).Source("Base").Tag(mod.Condition("Debilitated"), mod.GlobalEffect("Debuff").Name("Debilitated")))
	modDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeMore, -20).Source("Base").Tag(mod.Condition("Debilitated"), mod.GlobalEffect("Debuff").Name("Debilitated")))
	modDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, -10).Source("Base").Tag(mod.Condition("MalignantMadness"), mod.GlobalEffect("Debuff").Name("Malignant Madness")))
	modDB.AddMod(mod.NewFloat("ActionSpeed", mod.TypeMore, -10).Source("Base").Tag(mod.Condition("MalignantMadness"), mod.GlobalEffect("Debuff").Name("Malignant Madness")))

	modDB.AddMod(mod.NewFloat("DamageTaken", mod.TypeIncrease, 10).Source("Base").Flag(mod.MFlagAttack).Tag(mod.Condition("Intimidated")))
	modDB.AddMod(mod.NewFloat("DamageTaken", mod.TypeIncrease, 10).Source("Base").Flag(mod.MFlagAttack).
		Tag(mod.Condition("Intimidated").Neg(true), mod.Condition("Party:Intimidated")))
	modDB.AddMod(mod.NewFloat("DamageTaken", mod.TypeIncrease, 10).Source("Base").Flag(mod.MFlagSpell).Tag(mod.Condition("Unnerved")))
	modDB.AddMod(mod.NewFloat("DamageTaken", mod.TypeIncrease, 10).Source("Base").Flag(mod.MFlagSpell).
		Tag(mod.Condition("Unnerved").Neg(true), mod.Condition("Party:Unnerved")))

	modDB.AddMod(mod.NewFlag("Condition:Burning", true).Source("Base").Tag(mod.IgnoreCond()).Tag(mod.Condition("Ignited")))
	modDB.AddMod(mod.NewFlag("Condition:Poisoned", true).Source("Base").Tag(mod.IgnoreCond()).Tag(mod.MultiplierThreshold("PoisonStack").Threshold(1)))

	modDB.AddMod(mod.NewFlag("Blind", true).Source("Base").Tag(mod.Condition("Blinded")))
	modDB.AddMod(mod.NewFlag("Chill", true).Source("Base").Tag(mod.Condition("Chilled")))
	modDB.AddMod(mod.NewFlag("Freeze", true).Source("Base").Tag(mod.Condition("Frozen")))
	modDB.AddMod(mod.NewFlag("Fortify", true).Source("Base").Tag(mod.Condition("Fortify")))
	modDB.AddMod(mod.NewFlag("Fortified", true).Source("Base").Tag(mod.Condition("Fortified")))
	modDB.AddMod(mod.NewFlag("Excommunicated", true).Source("Base").Tag(mod.Condition("Excommunicated")))
	modDB.AddMod(mod.NewFlag("Fanaticism", true).Source("Base").Tag(mod.Condition("Fanaticism")))
	modDB.AddMod(mod.NewFlag("Onslaught", true).Source("Base").Tag(mod.Condition("Onslaught")))
	modDB.AddMod(mod.NewFlag("UnholyMight", true).Source("Base").Tag(mod.Condition("UnholyMight")))
	modDB.AddMod(mod.NewFlag("ChaoticMight", true).Source("Base").Tag(mod.Condition("ChaoticMight")))
	modDB.AddMod(mod.NewFlag("Tailwind", true).Source("Base").Tag(mod.Condition("Tailwind")))
	modDB.AddMod(mod.NewFlag("Adrenaline", true).Source("Base").Tag(mod.Condition("Adrenaline")))

	modDB.AddMod(mod.NewFlag("AccelerationShrine", true).Source("Base").Tag(mod.Condition("AccelerationShrine")))
	modDB.AddMod(mod.NewFlag("BrutalShrine", true).Source("Base").Tag(mod.Condition("BrutalShrine")))
	modDB.AddMod(mod.NewFlag("DiamondShrine", true).Source("Base").Tag(mod.Condition("DiamondShrine")))
	modDB.AddMod(mod.NewFlag("DivineShrine", true).Source("Base").Tag(mod.Condition("DivineShrine")))
	modDB.AddMod(mod.NewFlag("EchoingShrine", true).Source("Base").Tag(mod.Condition("EchoingShrine")))
	modDB.AddMod(mod.NewFlag("GloomShrine", true).Source("Base").Tag(mod.Condition("GloomShrine")))
	modDB.AddMod(mod.NewFlag("GreaterFreezingShrine", true).Source("Base").Tag(mod.Condition("GreaterFreezingShrine")))
	modDB.AddMod(mod.NewFlag("GreaterShockingShrine", true).Source("Base").Tag(mod.Condition("GreaterShockingShrine")))
	modDB.AddMod(mod.NewFlag("GreaterSkeletalShrine", true).Source("Base").Tag(mod.Condition("GreaterSkeletalShrine")))
	modDB.AddMod(mod.NewFlag("ImpenetrableShrine", true).Source("Base").Tag(mod.Condition("ImpenetrableShrine")))
	modDB.AddMod(mod.NewFlag("MassiveShrine", true).Source("Base").Tag(mod.Condition("MassiveShrine")))
	modDB.AddMod(mod.NewFlag("ReplenishingShrine", true).Source("Base").Tag(mod.Condition("ReplenishingShrine")))
	modDB.AddMod(mod.NewFlag("ResistanceShrine", true).Source("Base").Tag(mod.Condition("ResistanceShrine")))
	modDB.AddMod(mod.NewFlag("ResonatingShrine", true).Source("Base").Tag(mod.Condition("ResonatingShrine")))
	modDB.AddMod(mod.NewFlag("LesserAccelerationShrine", true).Source("Base").Tag(mod.Condition("LesserAccelerationShrine"), mod.Condition("AccelerationShrine").Neg(true)))
	modDB.AddMod(mod.NewFlag("LesserBrutalShrine", true).Source("Base").Tag(mod.Condition("LesserBrutalShrine"), mod.Condition("BrutalShrine").Neg(true)))
	modDB.AddMod(mod.NewFlag("LesserImpenetrableShrine", true).Source("Base").Tag(mod.Condition("LesserImpenetrableShrine"), mod.Condition("ImpenetrableShrine").Neg(true)))
	modDB.AddMod(mod.NewFlag("LesserMassiveShrine", true).Source("Base").Tag(mod.Condition("LesserMassiveShrine"), mod.Condition("MassiveShrine").Neg(true)))
	modDB.AddMod(mod.NewFlag("LesserReplenishingShrine", true).Source("Base").Tag(mod.Condition("LesserReplenishingShrine"), mod.Condition("ReplenishingShrine").Neg(true)))
	modDB.AddMod(mod.NewFlag("LesserResistanceShrine", true).Source("Base").Tag(mod.Condition("LesserResistanceShrine"), mod.Condition("ResistanceShrine").Neg(true)))

	modDB.AddMod(mod.NewFlag("AlchemistsGenius", true).Source("Base").Tag(mod.Condition("AlchemistsGenius")))
	modDB.AddMod(mod.NewFlag("LuckyHits", true).Source("Base").Tag(mod.Condition("LuckyHits")))
	modDB.AddMod(mod.NewFlag("Convergence", true).Source("Base").Tag(mod.Condition("Convergence")))

	modDB.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, -15).Source(mod.SourceBase).Tag(mod.Condition("Crushed")))
	modDB.AddMod(mod.NewFloat("CritChanceCap", mod.TypeBase, 100).Source("Base"))
}

//pob:block:src/Modules/CalcSetup.lua:609-639
func buildRadiusJewelList(env *Environment, extraFuncs *moddb.ModList) ([]*radiusJewel, []string) {
	if env.Build == nil {
		return nil, nil
	}

	specs := env.Build.Tree.Specs
	if len(specs) == 0 {
		return nil, nil
	}
	idx := env.Build.Tree.ActiveSpec - 1
	if idx < 0 || idx >= len(specs) {
		idx = 0
	}
	sockets, byID := overrideSockets(env, specs[idx].Sockets, itemTextByID(env))
	if len(sockets) == 0 {
		return nil, nil
	}

	order := make([]int, 0, len(sockets))
	for i := range sockets {
		order = append(order, i)
	}
	sort.SliceStable(order, func(a, b int) bool { return sockets[order[a]].NodeID < sockets[order[b]].NodeID })

	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	if env.Spec != nil {
		tree = env.Spec.Tree()
	}
	radiusNodes := tree.JewelRadiusNodes()

	jewels := make([]*radiusJewel, 0)
	extra := make(map[string]bool)
	hasExtraJewelFuncs := extraFuncs != nil && len(extraFuncs.Mods()) > 0

	limitDisabled := limitDisabledJewelSockets(env)

	for _, i := range order {
		socket := sockets[i]
		socketID := strconv.Itoa(socket.NodeID)
		if _, allocated := env.AllocatedNodes[socketID]; !allocated {
			continue
		}
		if limitDisabled[socket.NodeID] {
			continue
		}
		text, ok := byID[socket.ItemID]
		if !ok {
			continue
		}
		item := peekItemCached(env.Cache, text)
		radiusIndex, hasRadius := itemJewelRadiusIndex(item)
		if !hasRadius && !hasExtraJewelFuncs {
			continue
		}

		nodesForRadius := func(index int) (map[string]bool, bool) {
			if isExpansionSocket(tree, socketID) {
				return map[string]bool{}, true
			}
			perRadius, known := radiusNodes[int64(socket.NodeID)]
			if !known || index < 0 || index >= len(perRadius) {
				return nil, false
			}
			out := make(map[string]bool, len(perRadius[index]))
			for _, id := range perRadius[index] {
				out[strconv.FormatInt(id, 10)] = true
			}
			return out, true
		}

		addEntry := func(nodes map[string]bool, funcType string, fn radiusJewelFunc) {
			jewels = append(jewels, &radiusJewel{
				nodes:    nodes,
				funcType: funcType,
				fn:       fn,
				nodeID:   socketID,
				data:     newRadiusJewelData(mod.Source("Tree:" + socketID)),
			})

			if funcType == "Self" {
				return
			}
			for id := range nodes {
				if _, allocated := env.AllocatedNodes[id]; !allocated {
					extra[id] = true
				}
			}
		}

		if hasRadius {
			if funcs := itemJewelFuncs(item); len(funcs) > 0 {
				if nodes, ok := nodesForRadius(radiusIndex); ok {
					for _, entry := range funcs {
						addEntry(nodes, entry.Type, entry.Func)
					}
				}
			}
		}

		if hasExtraJewelFuncs {
			for _, payload := range extraFuncs.List(&moddb.ListCfg{Item: item}, "ExtraJewelFunc") {
				entry, ok := payload.(extraJewelFuncEntry)
				if !ok {
					continue
				}
				index, known := data.JewelRadiusIndexForLabel(entry.Radius)
				if !known {
					continue
				}
				nodes, ok := nodesForRadius(index)
				if !ok {
					continue
				}
				addEntry(nodes, entry.Type, entry.Func)
			}
		}
	}

	extraIDs := make([]string, 0, len(extra))
	for id := range extra {
		extraIDs = append(extraIDs, id)
	}
	sort.Strings(extraIDs)

	return jewels, extraIDs
}

//pob:block:src/Classes/PassiveSpec.lua:42-49
func isExpansionSocket(tree *data.Tree, nodeID string) bool {
	node, ok := tree.Nodes[nodeID]
	return ok && node.ExpansionJewel != nil && node.ExpansionJewel.Parent != nil
}

//pob:block:src/Classes/Item.lua:236-242,src/Classes/Item.lua:1599-1601
func itemJewelRadiusIndex(item *Item) (int, bool) {
	if index, ok := data.JewelRadiusIndexForLabel(item.JewelRadiusLabel); ok {
		return index, true
	}

	for _, m := range item.Mods {
		if m.Name() != "JewelData" {
			continue
		}
		value, ok := m.Value().List().(mod.JewelData)
		if !ok || value.Key != "radiusIndex" {
			continue
		}
		switch n := value.Value.(type) {
		case int:
			return n - 1, true
		case float64:
			return int(n) - 1, true
		}
	}

	return 0, false
}

//pob:block:src/Classes/Item.lua:2421-2424
func itemJewelFuncs(item *Item) []jewelFuncEntry {
	funcs := make([]jewelFuncEntry, 0)
	for _, m := range item.Mods {
		if m.Name() != "JewelFunc" || m.Type() != mod.TypeList {
			continue
		}
		if entry, ok := m.Value().List().(jewelFuncEntry); ok && entry.Func != nil {
			funcs = append(funcs, entry)
		}
	}
	return funcs
}

var socketColourIndexNames = []string{"R", "G", "B"}

var socketMatchesFollowOverride = false

//pob:block:src/Classes/SkillsTab.lua:1223-1231
func socketGroupSlotItem(env *Environment, slotName string) *ItemData {
	if env == nil || env.Build == nil || slotName == "" {
		return nil
	}
	items := env.Build.Items
	if len(items.ItemSets) == 0 {
		return nil
	}
	set := items.ItemSets[activeItemSetIndex(items)]
	if socketMatchesFollowOverride {
		slotsByName := make(map[string]pob.Slot, len(set.Slots))
		for _, slot := range set.Slots {
			slotsByName[slot.Name] = slot
		}
		byID := make(map[int]string, len(items.Items))
		for _, it := range items.Items {
			byID[it.ID] = it.Text
		}
		if item, ok := peekSlotItem(env, slotName, slotsByName, byID); ok {
			return itemSocketData(item)
		}
		return nil
	}
	for _, slot := range set.Slots {
		if slot.Name != slotName || slot.ItemID == 0 {
			continue
		}
		for _, it := range items.Items {
			if it.ID == slot.ItemID {
				return itemSocketData(peekItemCached(env.Cache, it.Text))
			}
		}
		return nil
	}
	return nil
}

func itemSocketData(item *Item) *ItemData {
	id := &ItemData{SocketColourAlwaysMatches: item.SocketColourAlwaysMatches}
	if len(item.Sockets) > 0 {
		id.SocketColors = make([]string, len(item.Sockets))
		for i, s := range item.Sockets {
			id.SocketColors[i] = s.Color
		}
	}
	return id
}

//pob:src/Classes/SkillsTab.lua:1210-1245
func buildMatchesSocket(env *Environment, socketGroups []pob.Skill) map[int]map[int]bool {
	matches := make(map[int]map[int]bool, len(socketGroups))
	slotSocketedCounts := make(map[string]int, len(socketGroups))

	for groupIndex := range socketGroups {
		socketGroup := socketGroups[groupIndex]
		if socketGroup.Slot == "" {
			continue
		}
		gemOffset := slotSocketedCounts[socketGroup.Slot]
		item := socketGroupSlotItem(env, socketGroup.Slot)

		if item != nil && len(item.SocketColors) > 0 {
			for i := range socketGroup.Gems {
				gemInstance := socketGroup.Gems[i]
				gemData, bareGrantedEffect := resolveGemInstance(gemInstance)
				if gemData == nil && bareGrantedEffect == nil {
					continue
				}
				matched := false
				switch {
				case item.SocketColourAlwaysMatches:
					matched = true
				default:
					gemColour := ""
					if colour := gemPrimaryColour(gemData, bareGrantedEffect); colour >= 1 && colour <= len(socketColourIndexNames) {
						gemColour = socketColourIndexNames[colour-1]
					}
					if gemIdx := gemOffset + i; gemColour != "" && gemIdx < len(item.SocketColors) {
						matched = item.SocketColors[gemIdx] == gemColour
					}
				}
				if matched {
					if matches[groupIndex] == nil {
						matches[groupIndex] = map[int]bool{}
					}
					matches[groupIndex][i] = true
				}
			}
		}
		slotSocketedCounts[socketGroup.Slot] = gemOffset + len(socketGroup.Gems)
	}
	return matches
}

//pob:block:src/Classes/SkillsTab.lua:1221-1222,src/Classes/SkillsTab.lua:1236
func gemPrimaryColour(gemData *poe.SkillGem, bareGrantedEffect *poe.GrantedEffect) int {
	grantedEffect := bareGrantedEffect
	if grantedEffect == nil && gemData != nil {
		grantedEffect = primaryGrantedEffectOf(gemData)
	}
	if grantedEffect == nil {
		return 0
	}
	return raw2.PoBSkills[grantedEffect.ID].Color
}

//pob:src/Classes/SkillsTab.lua:1437-1448
func buildImbuedSupportBySlot(groups []pob.Skill) map[string][]*poe.GrantedEffect {
	out := map[string][]*poe.GrantedEffect{}
	for _, group := range groups {
		if group.Slot == "" || group.ImbuedSupport == nil {
			continue
		}
		gem, ok := gemLookup().gemForBaseName[strings.ToLower(*group.ImbuedSupport)+" support"]
		if !ok {
			continue
		}
		effects := raw2.GrantedEffectsForGem(gem)
		if len(effects) == 0 {
			continue
		}
		kept := []*poe.GrantedEffect{effects[0]}
		for _, secondary := range effects[1:] {
			if skill, known := raw2.PoBSkills[secondary.ID]; known && skill.Support {
				kept = append(kept, secondary)
			}
		}
		out[group.Slot] = kept
	}
	return out
}

//pob:block:src/Classes/ItemsTab.lua:4309-4325
func overrideSockets(env *Environment, sockets []pob.Socket, byID map[int]string) ([]pob.Socket, map[int]string) {
	o := env.override
	if o == nil || o.SocketNodeID == 0 {
		return sockets, byID
	}
	return socketWith(sockets, byID, o.SocketNodeID, o.SocketItem)
}

func socketWith(sockets []pob.Socket, byID map[int]string, socketID int, item *Item) ([]pob.Socket, map[int]string) {
	out := make([]pob.Socket, 0, len(sockets)+1)
	for _, s := range sockets {
		if s.NodeID != socketID {
			out = append(out, s)
		}
	}
	if item == nil {
		return out, byID
	}
	ids := maps.Clone(byID)
	temp := -1
	for {
		if _, taken := ids[temp]; !taken {
			break
		}
		temp--
	}
	ids[temp] = item.Raw
	return append(out, pob.Socket{NodeID: socketID, ItemID: temp}), ids
}
