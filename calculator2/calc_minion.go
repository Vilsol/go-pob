package calculator2

import (
	"math"
	"slices"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/poe2minions"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// Minion is activeSkill.minion: the actor a minion skill creates; Actor.ActiveSkillList and
// Actor.MainSkill hold its skills.
type Minion struct {
	Actor
	MinionData *poe2minions.Minion
	Uses       map[string]bool
	// ItemSet is minion.itemSet: the item set a minionHasItemSet skill's minion takes its items from.
	ItemSet   *ItemSet
	LifeTable []float64
	Type      string

	HiddenDamageFixup float64
	Hostile           bool
}

// loadMinionData is data.minions, loaded once from go-pob-data and the templates.
var loadMinionData = sync.OnceValues(func() (poe2minions.Minions, error) {
	return poe2minions.Load(gameDataContext(), skillDataVersion, gameDataCache())
})

// minionData is data.minions; a failed load leaves it empty, so no minion is created.
func minionData() poe2minions.Minions {
	minions, err := loadMinionData()
	if err != nil {
		return nil
	}
	return minions
}

// spectreList and beastList are build.spectreList and build.beastList: the saved ids data.minions has.
//
//pob2:block:src/Modules/Build.lua:1146-1156
func (env *Environment) minionIDList(ids []string) []string {
	minions := minionData()
	var out []string
	for _, id := range ids {
		if id != "" && minions[id] != nil {
			out = append(out, id)
		}
	}
	return out
}

func (env *Environment) spectreList() []string {
	ids := make([]string, len(env.Build.Build.Spectres))
	for i, m := range env.Build.Build.Spectres {
		ids[i] = m.ID
	}
	return env.minionIDList(ids)
}

func (env *Environment) beastList() []string {
	ids := make([]string, len(env.Build.Build.BeastCompanions))
	for i, m := range env.Build.Build.BeastCompanions {
		ids[i] = m.ID
	}
	return env.minionIDList(ids)
}

// minionLevelTable is data.minionLevelTable[level], false for nil.
func minionLevelTable(level int) (float64, bool) {
	if level < 1 || level > len(poe2.MinionLevelTable) {
		return 0, false
	}
	return poe2.MinionLevelTable[level-1], true
}

func skillDataNumber(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case bool:
		return 0, x
	}
	return 0, false
}

// createMinion is buildActiveSkillModList's minion: the list a skill picks its minion from and the
// minion actor.
//
//pob2:block:src/Modules/CalcActiveSkill.lua:923-1051
func createMinion(env *Environment, activeSkill *ActiveSkill) {
	activeEffect := activeSkill.ActiveEffect
	activeGrantedEffect := activeEffect.GrantedEffect
	overlay := activeGrantedEffect.Overlay
	var minionList []string
	monsterDamage := false
	switch {
	case overlay.MinionList != nil && strings.HasPrefix(activeGrantedEffect.Name, "Spectre"):
		minionList = env.spectreList()
		monsterDamage = true
	case overlay.MinionList != nil && strings.HasPrefix(activeGrantedEffect.Name, "Companion"):
		minionList = env.beastList()
		monsterDamage = true
	case len(overlay.MinionList) > 0:
		minionList = slices.Clone(overlay.MinionList)
	}
	activeSkill.MinionList = minionList
	srcInstance := activeEffect.SrcInstance
	if len(minionList) == 0 {
		if srcInstance != nil && !(activeEffect.GemData != nil && len(activeEffect.GemData.AdditionalGrantedEffects) >= 1) {
			srcInstance.SkillMinionCalcs = ""
			srcInstance.SkillMinion = ""
			srcInstance.SkillMinionItemSetCalcs = 0
			srcInstance.SkillMinionItemSet = 0
		}
		return
	}
	var minionType string
	if srcInstance != nil {
		if env.Mode == "CALCS" && activeSkill == env.Player.MainSkill {
			minionType = minionList[max(slices.Index(minionList, srcInstance.SkillMinionCalcs), 0)]
			srcInstance.SkillMinionCalcs = minionType
		} else {
			minionType = minionList[max(slices.Index(minionList, srcInstance.SkillMinion), 0)]
			srcInstance.SkillMinion = minionType
		}
	} else {
		minionType = minionList[0]
	}
	data := minionData()[minionType]
	if data == nil {
		// PoB indexes a nil minionData here and stops; no minion is created.
		return
	}
	minion := &Minion{Type: minionType, MinionData: data}
	activeSkill.Minion = minion
	activeSkill.SkillFlags["haveMinion"] = true
	minion.Parent = env.Player
	minion.Enemy = env.Enemy
	skillData := activeSkill.SkillData
	level, ok := 0.0, false
	if truthy(skillData["minionLevelIsEnemyLevel"]) {
		level, ok = float64(env.EnemyLevel), true
	}
	if !ok && truthy(skillData["minionLevelIsTriggeredSkillLevel"]) && srcInstance != nil && srcInstance.SupportEffect != nil && srcInstance.SupportEffect.ActiveSkillLevel != 0 {
		level, ok = minionLevelTable(srcInstance.SupportEffect.ActiveSkillLevel)
	}
	if !ok {
		if levelCap, isCap := skillDataNumber(skillData["minionLevelIsPlayerLevel"]); isCap {
			level, ok = math.Min(float64(env.Build.Build.Level), levelCap), true
		}
	}
	if !ok {
		level, ok = skillDataNumber(skillData["minionLevel"])
	}
	if !ok {
		level, ok = minionLevelTable(activeEffect.Level)
	}
	if !ok {
		level = 1
	}
	minion.Level = int(math.Min(math.Max(level, 1), 100))
	minion.ItemList = map[string]*Item{}
	if len(overlay.MinionUses) > 0 {
		minion.Uses = make(map[string]bool, len(overlay.MinionUses))
		for _, slot := range overlay.MinionUses {
			minion.Uses[slot] = true
		}
	}
	minion.LifeTable = poe2.MonsterAllyLifeTable
	attackTime := data.AttackTime
	damageTable := poe2.MonsterAllyDamageTable
	if monsterDamage {
		damageTable = poe2.MonsterDamageTable
		minion.HiddenDamageFixup = round(poe2.MonsterAllyDamageTable[minion.Level-1]/damageTable[minion.Level-1]*miscConstant("SpectreBeastDamageFixup"), 2) - 1
	}
	damage := math.Floor(damageTable[minion.Level-1]) * data.Damage
	if !data.BaseDamageIgnoresAttackSpeed { // minions with this flag do not factor attack time into their base damage
		damage = damage * attackTime
	}
	//pob2:block:src/Modules/CalcActiveSkill.lua:990-1005
	if overlay.MinionHasItemSet {
		if tab := env.items; tab != nil && srcInstance != nil {
			if env.Mode == "CALCS" && activeSkill == env.Player.MainSkill {
				if tab.ItemSets[srcInstance.SkillMinionItemSetCalcs] == nil {
					srcInstance.SkillMinionItemSetCalcs = tab.ItemSetOrderList[0]
				}
				minion.ItemSet = tab.ItemSets[srcInstance.SkillMinionItemSetCalcs]
			} else {
				if tab.ItemSets[srcInstance.SkillMinionItemSet] == nil {
					srcInstance.SkillMinionItemSet = tab.ItemSetOrderList[0]
				}
				minion.ItemSet = tab.ItemSets[srcInstance.SkillMinionItemSet]
			}
		}
	} else if srcInstance != nil && !(activeEffect.GemData != nil && len(activeEffect.GemData.AdditionalGrantedEffects) >= 1) {
		srcInstance.SkillMinionItemSetCalcs = 0
		srcInstance.SkillMinionItemSet = 0
	}
	//pob2:block:src/Modules/CalcActiveSkill.lua:1006-1042
	minion.weaponDataSet = true
	switch {
	case truthy(skillData["minionUseBowAndQuiver"]) && env.Player.WeaponData1Type == "Bow",
		env.TheIronMass != nil && minionType == "RaisedSkeleton":
		minion.WeaponData1, minion.WeaponData1Type = env.Player.WeaponData1, env.Player.WeaponData1Type
	default:
		minion.WeaponData1Type = data.WeaponType1
		if minion.WeaponData1Type == "" {
			minion.WeaponData1Type = "None"
		}
		minion.WeaponData1 = map[string]float64{
			"AttackRate":  1 / attackTime,
			"CritChance":  data.CritChance,
			"PhysicalMin": math.Floor(damage * (1 - data.DamageSpread)),
			"PhysicalMax": math.Floor(damage * (1 + data.DamageSpread)),
			"range":       float64(data.AttackRange),
		}
	}
	minion.WeaponData2, minion.WeaponData2Type = map[string]float64{}, ""
	if minion.Uses["Weapon 1"] {
		if minion.ItemSet != nil {
			slotName := "Weapon 1"
			if minion.ItemSet.UseSecondWeaponSet {
				slotName = "Weapon 1 Swap"
			}
			if item := env.items.Items[minion.ItemSet.SelItemID(slotName)]; item != nil && item.WeaponData != nil {
				minion.WeaponData1, minion.WeaponData1Type = item.WeaponData[1], item.Base.Type
			}
		} else {
			minion.WeaponData1, minion.WeaponData1Type = env.Player.WeaponData1, env.Player.WeaponData1Type
		}
	}
	if minion.Uses["Weapon 2"] {
		if minion.ItemSet != nil {
			slotName := "Weapon 2"
			if minion.ItemSet.UseSecondWeaponSet {
				slotName = "Weapon 2 Swap"
			}
			if item := env.items.Items[minion.ItemSet.SelItemID(slotName)]; item != nil && item.WeaponData != nil {
				minion.WeaponData2, minion.WeaponData2Type = item.WeaponData[2], item.Base.Type
			}
		} else {
			minion.WeaponData2, minion.WeaponData2Type = env.Player.WeaponData2, env.Player.WeaponData2Type
		}
	}
}

// initMinionModDB initialises a minion's modifier database with its base stats (life, defences,
// resists), monster type mods and player-granted mods, for the given owning skill. The bow and quiver a
// minion may use (minionUseBowAndQuiver) are not ported.
//
//pob2:src/Modules/CalcPerform.lua:1049-1156
func initMinionModDB(env *Environment, activeSkill *ActiveSkill) {
	skillFlags := statSetSkillFlags(activeSkill)
	modDB := env.ModDB
	minion := activeSkill.Minion
	minion.ModDB.Multipliers["Level"] = float64(minion.Level)
	initModDB(env, minion.ModDB)
	baseLife := minion.LifeTable[minion.Level-1] * minion.MinionData.Life
	minion.ModDB.AddMod(newMod("Life", mod.TypeBase, math.Floor(baseLife), "Base"))
	if minion.MinionData.EnergyShield != 0 {
		minion.ModDB.AddMod(newMod("LifeConvertToEnergyShield", mod.TypeBase, minion.MinionData.EnergyShield*100, "Base"))
	}
	// Armour formula is math.floor((10 + 2 * level) * 1.067 ^ level)
	minion.ModDB.AddMod(newMod("Armour", mod.TypeBase, round(poe2.MonsterArmourTable[minion.Level-1]*orOne(minion.MinionData.Armour)), "Base"))
	// Evasion formula is math.floor((50 + 16 * level + 16 * level * (MonsterType.Evasion / 100)) * (1.0212 ^ level)
	minion.ModDB.AddMod(newMod("Evasion", mod.TypeBase, round(poe2.MonsterEvasionTable[minion.Level-1]*orOne(minion.MinionData.Evasion)), "Base"))
	if modDB.Flag(nil, "MinionAccuracyEqualsAccuracy") {
		dexAcc := miscConstant("AccuracyPerDexBase")
		if o := modDB.Override(nil, "DexAccBonusOverride"); o != nil {
			dexAcc = o.Float()
		}
		minion.ModDB.AddMod(newMod("Accuracy", mod.TypeBase, calcLibVal(modDB, "Accuracy", nil)+float64(calcLibVal(modDB, "Dex", nil)*dexAcc), "Player"))
	} else {
		// Minions no longer need Accuracy as of patch 0.3.0
		minion.ModDB.AddMod(mod.NewFlag("CannotBeEvaded", true).Source("Minion Attacks always hit"))
	}
	minion.ModDB.AddMod(newMod("CritMultiplier", mod.TypeBase, poe2.MonsterConstants["base_critical_hit_damage_bonus"]+poe2.PlayerMinionIntrinsicStats["base_critical_hit_damage_bonus"], "Base"))
	if skillFlags["summonBeast"] {
		minion.ModDB.AddMod(newMod("FireResist", mod.TypeBase, float64(minion.MinionData.CompanionFireResist), "Base"))
		minion.ModDB.AddMod(newMod("ColdResist", mod.TypeBase, float64(minion.MinionData.CompanionColdResist), "Base"))
		minion.ModDB.AddMod(newMod("LightningResist", mod.TypeBase, float64(minion.MinionData.CompanionLightningResist), "Base"))
		minion.ModDB.AddMod(newMod("ChaosResist", mod.TypeBase, float64(minion.MinionData.CompanionChaosResist), "Base"))
	} else {
		minion.ModDB.AddMod(newMod("FireResist", mod.TypeBase, float64(minion.MinionData.FireResist), "Base"))
		minion.ModDB.AddMod(newMod("ColdResist", mod.TypeBase, float64(minion.MinionData.ColdResist), "Base"))
		minion.ModDB.AddMod(newMod("LightningResist", mod.TypeBase, float64(minion.MinionData.LightningResist), "Base"))
		minion.ModDB.AddMod(newMod("ChaosResist", mod.TypeBase, float64(minion.MinionData.ChaosResist), "Base"))
	}
	minion.ModDB.AddMod(newMod("ProjectileCount", mod.TypeBase, 1, "Base"))
	minion.ModDB.AddMod(newMod("PhysicalHeavyStunBuildup", mod.TypeMore, poe2.MonsterConstants["physical_hit_damage_stun_multiplier_+%_final_from_ot"], "Physical Damage"))
	minion.ModDB.AddMod(newMod("EnemyHeavyStunBuildup", mod.TypeMore, poe2.MonsterConstants["melee_hit_damage_stun_multiplier_+%_final_from_ot"], "Melee Damage").Flag(modFlagMelee))
	minion.ModDB.AddMod(newMod("Damage", mod.TypeMore, minion.HiddenDamageFixup*100, "Hidden Level Scaling"))
	for _, m := range minion.MinionData.ModList {
		addSkillModDB(minion.ModDB, m)
	}
	for _, m := range activeSkill.ExtraSkillModList {
		minion.ModDB.AddMod(m)
	}
	if env.TalismanModList != nil {
		// Adding mods provided by "Necromantic Talisman"
		minion.ModDB.AddList(env.TalismanModList)
	}
	if env.TheIronMass != nil && minion.Type == "RaisedSkeleton" {
		minion.ModDB.AddList(env.TheIronMass)
	}
	//pob2:block:src/Modules/CalcPerform.lua:1122-1140
	if (minion.ItemSet != nil || minion.Uses != nil) && env.items != nil {
		for _, slot := range env.items.OrderedSlots {
			slotName := slot.SlotName
			if minion.Uses[slotName] {
				var item *Item
				if minion.ItemSet != nil {
					if slot.WeaponSet == 1 && minion.ItemSet.UseSecondWeaponSet {
						slotName += " Swap"
					}
					item = env.items.Items[minion.ItemSet.SelItemID(slotName)]
				} else {
					item = env.Player.ItemList[slotName]
				}
				if item != nil {
					minion.ItemList[slotName] = item
					list := item.ModList
					if list == nil && slot.SlotNum >= 1 && slot.SlotNum <= len(item.SlotModList) {
						list = item.SlotModList[slot.SlotNum-1]
					}
					for _, m := range list {
						minion.ModDB.AddMod(m)
					}
				}
			}
		}
	}
	if modDB.Flag(nil, "StrengthAddedToMinions") {
		minion.ModDB.AddMod(newMod("Str", mod.TypeBase, round(calcLibVal(modDB, "Str", nil)), "Player"))
	}
	if modDB.Flag(nil, "StrengthAddedToCompanions") && activeSkill.SkillTypes["Companion"] {
		minion.ModDB.AddMod(newMod("Str", mod.TypeBase, round(calcLibVal(modDB, "Str", nil)), "Sturdy Ally"))
	}
	if modDB.Flag(nil, "HalfStrengthAddedToMinions") {
		minion.ModDB.AddMod(newMod("Str", mod.TypeBase, round(calcLibVal(modDB, "Str", nil)*0.5), "Player"))
	}
	if modDB.Flag(nil, "DexterityAddedToMinions") {
		minion.ModDB.AddMod(newMod("Dex", mod.TypeBase, round(calcLibVal(modDB, "Dex", nil)), "Dead can Dance"))
	}
	if modDB.Flag(nil, "DexterityAddedToCompanions") && activeSkill.SkillTypes["Companion"] {
		minion.ModDB.AddMod(newMod("Dex", mod.TypeBase, round(calcLibVal(modDB, "Dex", nil)), "Tandem Assault"))
	}
}

// addSkillModDB is modDB:AddMod for a data mod table; one the mod package cannot express adds nothing.
func addSkillModDB(modDB *moddb.ModDB, m poe2minions.Mod) {
	if _, ok := m["name"].(string); !ok {
		return
	}
	if converted, err := cacheMod(m); err == nil {
		modDB.AddMod(converted)
	}
}

func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}

// addMinionModifiers adds the MinionModifier mods of modList to the minion. No PoE2 MinionModifier
// carries a minion type.
//
//pob2:src/Modules/CalcPerform.lua:1161-1167
func addMinionModifiers(modList moddb.ModStoreFuncs, skillCfg *moddb.ListCfg, minion *Minion) {
	for _, value := range modList.List(skillCfg, "MinionModifier") {
		if m := minionModifierMod(value); m != nil {
			minion.ModDB.AddMod(m)
		}
	}
}

// newMinionModDBs is perform's minion half of building the skill mod lists: each minion gets a fresh
// modifier database and its skills.
//
//pob2:block:src/Modules/CalcPerform.lua:1206-1211
func newMinionModDBs(env *Environment) {
	for _, activeSkill := range env.Player.ActiveSkillList {
		if minion := activeSkill.Minion; minion != nil {
			minion.ModDB = moddb.NewModDB()
			minion.ModDB.Actor = &minion.Actor
			minion.Output = map[string]float64{}
			minion.OutputMainHand, minion.OutputOffHand = nil, nil
			createMinionSkills(env, activeSkill)
			activeSkill.SkillPartName = minion.MainSkill.ActiveEffect.GrantedEffect.Name
		}
	}
}

// minionActor is env.minion's actor, nil when there is no env.minion.
func (env *Environment) minionActor() *Actor {
	if env.Minion == nil {
		return nil
	}
	return &env.Minion.Actor
}

// addMinionKeystones adds the tree keystones the minion's Keystone mods name.
func addMinionKeystones(env *Environment, minion *Minion) {
	for _, modObj := range minion.ModDB.Tabulate(mod.TypeList, nil, "Keystone") {
		name, _ := modObj.Mod.Value().ValueList.(string)
		if node := env.Spec.Tree.KeystoneMap[name]; node != nil {
			minion.ModDB.AddList(processStats(env.modSource, node.ID, node.Type, node.Dn, node.Sd))
		}
	}
}

// calcMinionLifePool is the total life of the player's minions the include function selects; env.minion
// keeps the database perform built for it.
//
//pob2:src/Modules/CalcPerform.lua:3391-3420
func calcMinionLifePool(env *Environment, buffs map[string]*moddb.ModList, includeSkill func(*ActiveSkill, map[string]bool) bool) (float64, []minionLife) {
	totalLife := 0.0
	var lifeList []minionLife
	seenMinions := map[*Minion]bool{}
	for _, activeSkill := range env.Player.ActiveSkillList {
		skillFlags := statSetSkillFlags(activeSkill)
		minion := activeSkill.Minion
		if minion != nil && minion.ModDB != nil && !seenMinions[minion] && !skillFlags["disable"] && includeSkill(activeSkill, skillFlags) {
			seenMinions[minion] = true
			if minion != env.Minion {
				initMinionModDB(env, activeSkill)
				addMinionModifiers(activeSkill.SkillModList, activeSkill.SkillCfg, minion)
				addMinionKeystones(env, minion)
				for _, name := range sortedBuffNames(buffs) {
					addMinionModifiers(buffs[name], activeSkill.SkillCfg, minion)
				}
				doActorAttribsConditions(env, &minion.Actor)
			}
			doActorLifeManaSpirit(&minion.Actor, true)
			totalLife = totalLife + minion.Output["Life"]
			name := ""
			if minion.MinionData != nil {
				name = minion.MinionData.Name
			} else if activeSkill.ActiveEffect != nil && activeSkill.ActiveEffect.GrantedEffect != nil {
				name = activeSkill.ActiveEffect.GrantedEffect.Name
			}
			lifeList = append(lifeList, minionLife{Name: name, Life: minion.Output["Life"]})
		}
	}
	return totalLife, lifeList
}

// minionLife is one lifeList entry calcMinionLifePool returns.
type minionLife struct {
	Name string
	Life float64
}

func sortedBuffNames(buffs map[string]*moddb.ModList) []string {
	names := make([]string, 0, len(buffs))
	for name := range buffs {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// minionLifePools are the spectre and companion life pools that "% of Damage from Hits is taken from your
// Spectres' / Companion's Life before you" read.
//
//pob2:block:src/Modules/CalcPerform.lua:3422-3438
func minionLifePools(env *Environment, buffs map[string]*moddb.ModList) {
	modDB := env.ModDB
	// Total life of spectres, for "% of Damage from Hits is taken from your Spectres' Life before you"
	if modDB.HasMod(mod.TypeBase, nil, "TakenFromSpectresBeforeYou") && modDB.Override(nil, "TotalSpectreLife") == nil {
		totalSpectreLife, spectreLifeList := calcMinionLifePool(env, buffs, func(activeSkill *ActiveSkill, skillFlags map[string]bool) bool {
			return skillFlags["spectre"] && !activeSkill.SkillTypes["MinionsAreUndamagable"]
		})
		modDB.AddMod(newMod("TotalSpectreLife", mod.TypeBase, totalSpectreLife, "Spectres"))
		env.Player.SpectreLifeList = spectreLifeList
	}
	// Total life of damageable companions, for "% of Damage from Hits is taken from your Companion's Life before you"
	if modDB.HasMod(mod.TypeBase, nil, "TakenFromCompanionBeforeYou", "TakenFromCompanionBeforeYouFromDeflected") && modDB.Override(nil, "TotalCompanionLife") == nil {
		totalCompanionLife, companionLifeList := calcMinionLifePool(env, buffs, func(activeSkill *ActiveSkill, _ map[string]bool) bool {
			return activeSkill.SkillTypes["Companion"] && !activeSkill.SkillTypes["MinionsAreUndamagable"]
		})
		modDB.AddMod(newMod("TotalCompanionLife", mod.TypeBase, totalCompanionLife, "Companions"))
		env.Player.CompanionLifeList = companionLifeList
	}
}
