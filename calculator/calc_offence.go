package calculator

import (
	"fmt"
	"maps"
	"math"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/calculator/calclib"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

func calcDamage(activeSkill *ActiveSkill, output map[string]float64, cfg *moddb.ListCfg, breakdown *Breakdown, breakdownKey string, damageType data.DamageType, typeFlags int, convDst *data.DamageType) (float64, float64) {
	typeFlags = typeFlags | data.DamageTypeFlags[damageType]

	// Calculate conversions
	addMin := float64(0)
	addMax := float64(0)

	for _, otherType := range data.DamageType("").Values() {
		if otherType == damageType {
			// Damage can only be converted from damage types that precede this one in the conversion sequence, so stop here
			break
		}

		convMult := activeSkill.ConversionTable[otherType].Targets[damageType]
		if convMult > 0 {
			// Damage is being converted/gained from the other damage type
			minDamage, maxDamage := calcDamage(activeSkill, output, cfg, breakdown, breakdownKey, otherType, typeFlags, &damageType)
			addMin += minDamage * convMult
			addMax += maxDamage * convMult
		}
	}

	if addMin != 0 && addMax != 0 {
		addMin = math.Round(addMin)
		addMax = math.Round(addMax)
	}

	baseMin := output[string(damageType)+"MinBase"]
	baseMax := output[string(damageType)+"MaxBase"]

	if baseMin == 0 && baseMax == 0 {
		// No base damage for this type, don't need to calculate modifiers
		if breakdown != nil && (addMin != 0 || addMax != 0) {
			breakdown.DamageType(breakdownKey, BDamageType{
				Source:  string(damageType),
				ConvSrc: utils.Ternary(addMin != 0 || addMax != 0, utils.Ptr(fmt.Sprint(addMin)+" to "+fmt.Sprint(addMax)), nil),
				Total:   fmt.Sprint(addMin) + " to " + fmt.Sprint(addMax),
				// TODO Fix nil error
				//ConvDst: utils.Ternary(convDst != nil, utils.Ptr(fmt.Sprintf("%.2f%% to %s", activeSkill.ConversionTable[damageType].Targets[*convDst]*100, *convDst)), nil),
			})
		}
		return addMin, addMax
	}

	// Combine modifiers
	modNames := data.DamageStatsForType(typeFlags)
	inc := 1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, cfg, modNames...)/100
	more := math.Floor(activeSkill.SkillModList.More(cfg, modNames...)*100+0.50000001) / 100
	moreMinDamage := activeSkill.SkillModList.More(cfg, "Min"+string(damageType)+"Damage")
	moreMaxDamage := activeSkill.SkillModList.More(cfg, "Max"+string(damageType)+"Damage")

	if breakdown != nil {
		breakdown.DamageType(breakdownKey, BDamageType{
			Source:  string(damageType),
			Base:    utils.Ptr(fmt.Sprint(baseMin) + " to " + fmt.Sprint(baseMax)),
			Inc:     utils.Ternary(inc != 1, utils.Ptr("x "+fmt.Sprint(inc)), nil),
			More:    utils.Ternary(more != 1, utils.Ptr("x "+fmt.Sprint(more)), nil),
			ConvSrc: utils.Ternary(addMin != 0 || addMax != 0, utils.Ptr(fmt.Sprint(addMin)+" to "+fmt.Sprint(addMax)), nil),
			Total:   fmt.Sprint(utils.RoundTo(baseMin*inc*more, 0)+addMin) + " to " + fmt.Sprint(utils.RoundTo(baseMax*inc*more, 0)+addMax),
			// TODO Fix nil error
			//ConvDst: utils.Ternary(convDst != nil && activeSkill.ConversionTable[damageType].Targets[*convDst] > 0, utils.Ptr(fmt.Sprintf("%.2f%% to %s", activeSkill.ConversionTable[damageType].Targets[*convDst]*100, *convDst)), nil),
		})
	}

	return math.Round(((baseMin * inc * more) + addMin) * moreMinDamage),
		math.Round(((baseMax * inc * more) + addMax) * moreMaxDamage)
}

func calcAilmentSourceDamage(activeSkill *ActiveSkill, output map[string]float64, cfg *moddb.ListCfg, breakdown *Breakdown, breakdownKey string, damageType data.DamageType, typeFlags int) (float64, float64) {
	minDamage, maxDamage := calcDamage(activeSkill, output, cfg, breakdown, breakdownKey, damageType, typeFlags, nil)
	convMult := activeSkill.ConversionTable[damageType].Mult
	if breakdown != nil && convMult != 1 {
		breakdown.AddLine(breakdownKey, "Source damage:")
		breakdown.AddLine(breakdownKey, fmt.Sprintf("%.2f to %.2f ^8(total damage)", minDamage, maxDamage))
		breakdown.AddLine(breakdownKey, fmt.Sprintf("x %g ^8(%g%% converted to other damage types)", convMult, (1-convMult)*100))
		breakdown.AddLine(breakdownKey, fmt.Sprintf("= %.2f to %.2f", minDamage*convMult, maxDamage*convMult))
	}
	return minDamage * convMult, maxDamage * convMult
}

func CalculateOffence(env *Environment, actor *Actor, activeSkill *ActiveSkill) {
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB
	outputTable := actor.OutputTable
	breakdown := actor.Breakdown
	skillModList := activeSkill.SkillModList
	skillData := activeSkill.SkillData
	skillFlags := activeSkill.SkillFlags
	skillCfg := activeSkill.SkillCfg

	if skillData.ShowAverage {
		skillFlags[SkillFlagShowAverage] = true
	} else {
		skillFlags[SkillFlagNotAverage] = true
	}

	if skillFlags[SkillFlagDisable] {
		// Skill is disabled
		actor.Output["CombinedDPS"] = 0
		return
	}

	/*
		TODO calcAreaOfEffect
		local function calcAreaOfEffect(skillModList, skillCfg, skillData, skillFlags, output, breakdown)
			incArea, moreArea := calclib.Mods(skillModList, skillCfg, "AreaOfEffect")
			actor.Output["AreaOfEffectMod"] = round(round(incArea * moreArea, 10), 2)
			if skillData.radiusIsWeaponRange {
				range := 0
				if skillFlags.weapon1Attack {
					range = max(range, actor.weaponRange1)
				}
				if skillFlags.weapon2Attack {
					range = max(range, actor.weaponRange2)
				}
				skillData.radius = range + 2
			}
			if skillData.radius {
				skillFlags.area = true
				baseRadius := skillData.radius + (skillData.radiusExtra or 0) + skillModList.Sum(mod.TypeBase, skillCfg, "AreaOfEffect")
				actor.Output["AreaOfEffectRadius"] = calcRadius(baseRadius, actor.Output["AreaOfEffectMod"])
				if breakdown != nil {
					incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint := calcRadiusBreakpoints(baseRadius, incArea, moreArea)
					breakdown.AreaOfEffectRadius = breakdown.area(baseRadius, actor.Output["AreaOfEffectMod"], actor.Output["AreaOfEffectRadius"], incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint, skillData.radiusLabel)
				}
				if skillData.radiusSecondary {
					incAreaSecondary, moreAreaSecondary := calclib.Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectSecondary")
					actor.Output["AreaOfEffectModSecondary"] = round(round(incAreaSecondary * moreAreaSecondary, 10), 2)
					baseRadius = skillData.radiusSecondary + (skillData.radiusExtra or 0)
					actor.Output["AreaOfEffectRadiusSecondary"] = calcRadius(baseRadius, actor.Output["AreaOfEffectModSecondary"])
					if breakdown != nil {
						local incAreaBreakpointSecondary, moreAreaBreakpointSecondary, redAreaBreakpointSecondary, lessAreaBreakpointSecondary
						if not skillData.projectileSpeedAppliesToMSAreaOfEffect {
							incAreaBreakpointSecondary, moreAreaBreakpointSecondary, redAreaBreakpointSecondary, lessAreaBreakpointSecondary := calcRadiusBreakpoints(baseRadius, incAreaSecondary, moreAreaSecondary)
						}
						breakdown.AreaOfEffectRadiusSecondary = breakdown.area(baseRadius, actor.Output["AreaOfEffectModSecondary"], actor.Output["AreaOfEffectRadiusSecondary"], incAreaBreakpointSecondary, moreAreaBreakpointSecondary, redAreaBreakpointSecondary, lessAreaBreakpointSecondary, skillData.radiusSecondaryLabel)
					}
				}
				if skillData.radiusTertiary {
					incAreaTertiary, moreAreaTertiary := calclib.Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectTertiary")
					actor.Output["AreaOfEffectModTertiary"] = round(round(incAreaTertiary * moreAreaTertiary, 10), 2)
					baseRadius = skillData.radiusTertiary + (skillData.radiusExtra or 0)
					if skillData.projectileSpeedAppliesToMSAreaOfEffect {
						incSpeedTertiary, moreSpeedTertiary := calclib.Mods(skillModList, skillCfg, "ProjectileSpeed")
						actor.Output["SpeedModTertiary"] = round(round(incSpeedTertiary * moreSpeedTertiary, 10), 2)
						actor.Output["AreaOfEffectRadiusTertiary"] = calcMoltenStrikeTertiaryRadius(baseRadius, skillData.radiusSecondary, actor.Output["AreaOfEffectModTertiary"], actor.Output["SpeedModTertiary"])
						if breakdown != nil {
							setMoltenStrikeTertiaryRadiusBreakdown(
								breakdown, skillData.radiusSecondary, baseRadius, skillData.radiusTertiaryLabel,
								incAreaTertiary, moreAreaTertiary, incSpeedTertiary, moreSpeedTertiary
							)
						}
					} else {
						actor.Output["AreaOfEffectRadiusTertiary"] = calcRadius(baseRadius, actor.Output["AreaOfEffectModTertiary"])
						if breakdown != nil {
							incAreaBreakpointTertiary, moreAreaBreakpointTertiary, redAreaBreakpointTertiary, lessAreaBreakpointTertiary := calcRadiusBreakpoints(baseRadius, incAreaTertiary, moreAreaTertiary)
							breakdown.AreaOfEffectRadiusTertiary = breakdown.area(baseRadius, actor.Output["AreaOfEffectModTertiary"], actor.Output["AreaOfEffectRadiusTertiary"], incAreaBreakpointTertiary, moreAreaBreakpointTertiary, redAreaBreakpointTertiary, lessAreaBreakpointTertiary, skillData.radiusTertiaryLabel)
						}
					}
				}
			}
			if breakdown != nil {
				breakdown.AreaOfEffectMod = { }
				breakdown.multiChain(breakdown.AreaOfEffectMod, {
					{ "%.2f ^8(increased/reduced)", 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "AreaOfEffect") / 100 },
					{ "%.2f ^8(more/less)", skillModList.More(skillCfg, "AreaOfEffect") },
					total = fmt.Sprintf("= %.2f", actor.Output["AreaOfEffectMod"]),
				})
			}
		}
	*/

	runSkillFunc := func(name string) {
		f := activeSkill.ActiveEffect.GrantedEffect.Funcs[name]
		if f != nil {
			f(activeSkill, actor.Output, breakdown)
		}
	}

	runSkillFunc("initialFunc")
	isTriggered := false
	/*
		TODO isTriggered
		isTriggered := skillData.triggeredWhileChannelling or skillData.triggeredByCoC or skillData.triggeredByMeleeKill or skillData.triggeredByCospris or skillData.triggeredByMjolner or skillData.triggeredByUnique or skillData.triggeredByFocus or skillData.triggeredByCraft or skillData.triggeredByManaSpent or skillData.triggeredByParentAttack
		skillCfg.skillCond["SkillIsTriggered"] = skillData.triggered or isTriggered
		if skillCfg.skillCond["SkillIsTriggered"] {
			skillFlags.triggered = true
		}
		skillCfg.skillCond["SkillIsFocused"] = skillData.triggeredByFocus
		if skillCfg.skillCond["SkillIsFocused"] {
			skillFlags.focused = true
		}
	*/
	// Update skill data
	for _, value := range skillModList.List(skillCfg, "SkillData") {
		cVal := value.(*mod.SkillData)
		if cVal.Merge == "MAX" {
			utils.Set(skillData, cVal.Key, max(cVal.Value, utils.GetOr[float64](skillData, cVal.Key, 0)))
		} else {
			utils.Set(skillData, cVal.Key, cVal.Value)
		}
	}
	// Add addition stat bonuses
	if skillModList.Flag(nil, "IronGrip") {
		skillModList.AddMod(mod.NewFloat("PhysicalDamage", "INC", actor.StrDmgBonus).Source("Strength").Flag(mod.MFlagAttack | mod.MFlagProjectile))
	}
	if skillModList.Flag(nil, "IronWill") {
		skillModList.AddMod(mod.NewFloat("Damage", "INC", actor.StrDmgBonus).Source("Strength").Flag(mod.MFlagSpell))
	}

	if skillModList.Flag(nil, "TransfigurationOfBody") {
		skillModList.AddMod(mod.NewFloat("Damage", "INC", math.Floor(skillModList.Sum(mod.TypeIncrease, nil, "Life")*data.Transfiguration)).Source("Transfiguration of Body").Flag(mod.MFlagAttack))
	}
	if skillModList.Flag(nil, "TransfigurationOfMind") {
		skillModList.AddMod(mod.NewFloat("Damage", "INC", math.Floor(skillModList.Sum(mod.TypeIncrease, nil, "Mana")*data.Transfiguration)).Source("Transfiguration of Mind"))
	}
	if skillModList.Flag(nil, "TransfigurationOfSoul") {
		skillModList.AddMod(mod.NewFloat("Damage", "INC", math.Floor(skillModList.Sum(mod.TypeIncrease, nil, "EnergyShield")*data.Transfiguration)).Source("Transfiguration of Soul").Flag(mod.MFlagSpell))
	}

	if modDB.Flag(nil, "Elusive") && skillModList.Flag(nil, "SupportedByNightblade") {
		elusiveEffect := actor.Output["ElusiveEffectMod"] / 100
		nightbladeMulti := skillModList.Sum(mod.TypeBase, nil, "NightbladeElusiveCritMultiplier")
		skillModList.AddMod(mod.NewFloat("CritMultiplier", "BASE", math.Floor(nightbladeMulti*elusiveEffect)).Source("Nightblade"))
	}
	// additional charge based modifiers
	if skillModList.Flag(nil, "UseEnduranceCharges") && skillModList.Flag(nil, "EnduranceChargesConvertToBrutalCharges") {
		tripleDmgChancePerEndurance := modDB.Sum(mod.TypeBase, nil, "PerBrutalTripleDamageChance")
		modDB.AddMod(mod.NewFloat("TripleDamageChance", "BASE", tripleDmgChancePerEndurance).Tag(mod.Multiplier("BrutalCharge")))
	}
	if skillModList.Flag(nil, "UseFrenzyCharges") && skillModList.Flag(nil, "FrenzyChargesConvertToAfflictionCharges") {
		dmgPerAffliction := modDB.Sum(mod.TypeBase, nil, "PerAfflictionAilmentDamage")
		effectPerAffliction := modDB.Sum(mod.TypeBase, nil, "PerAfflictionNonDamageEffect")
		modDB.AddMod(mod.NewFloat("Damage", "MORE", dmgPerAffliction).Source("Affliction Charges").KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("AfflictionCharge")))
		modDB.AddMod(mod.NewFloat("EnemyChillEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
		modDB.AddMod(mod.NewFloat("EnemyShockEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
		modDB.AddMod(mod.NewFloat("EnemyFreezeEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
		modDB.AddMod(mod.NewFloat("EnemyScorchEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
		modDB.AddMod(mod.NewFloat("EnemyBrittleEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
		modDB.AddMod(mod.NewFloat("EnemySapEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
	}
	// set other limits
	actor.Output["ActiveTrapLimit"] = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveTrapLimit")
	actor.Output["ActiveMineLimit"] = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveMineLimit")

	// set flask scaling
	actor.Output["LifeFlaskRecovery"] = env.ItemModDB.Multipliers["LifeFlaskRecovery"]

	if skillModList.Flag(nil, "Condition:EnergyBladeActive") {
		dmgMod := calclib.Mod(skillModList, skillCfg, "EnergyBladeDamage")
		critMod := calclib.Mod(skillModList, skillCfg, "EnergyBladeCritChance")
		speedMod := calclib.Mod(skillModList, skillCfg, "EnergyBladeAttackSpeed")

		// "Weapon 1" : "weaponData1"
		if actor.ItemList["Weapon 1"] != nil && actor.ItemList["Weapon 1"].WeaponData != nil && actor.ItemList["Weapon 1"].WeaponData[1] != nil {
			actor.WeaponData1.CritChance = actor.WeaponData1.CritChance * critMod
			actor.WeaponData1.AttackRate = actor.WeaponData1.AttackRate * speedMod
			for _, damageType := range data.DamageType("").Values() {
				utils.Set(actor.WeaponData1, string(damageType+"Min"), utils.GetOr[float64](actor.WeaponData1, string(damageType+"Min"), 0)+math.Floor(skillModList.Sum(mod.TypeBase, skillCfg, string("EnergyBladeMin"+damageType))*dmgMod))
				utils.Set(actor.WeaponData1, string(damageType+"Max"), utils.GetOr[float64](actor.WeaponData1, string(damageType+"Max"), 0)+math.Floor(skillModList.Sum(mod.TypeBase, skillCfg, string("EnergyBladeMax"+damageType))*dmgMod))
			}
		}

		// "Weapon 2" : "weaponData2"
		if actor.ItemList["Weapon 2"] != nil && actor.ItemList["Weapon 2"].WeaponData != nil && actor.ItemList["Weapon 2"].WeaponData[1] != nil {
			actor.WeaponData2.CritChance = actor.WeaponData2.CritChance * critMod
			actor.WeaponData2.AttackRate = actor.WeaponData2.AttackRate * speedMod
			for _, damageType := range data.DamageType("").Values() {
				utils.Set(actor.WeaponData2, string(damageType+"Min"), utils.GetOr[float64](actor.WeaponData2, string(damageType+"Min"), 0)+math.Floor(skillModList.Sum(mod.TypeBase, skillCfg, string("EnergyBladeMin"+damageType))*dmgMod))
				utils.Set(actor.WeaponData2, string(damageType+"Max"), utils.GetOr[float64](actor.WeaponData2, string(damageType+"Max"), 0)+math.Floor(skillModList.Sum(mod.TypeBase, skillCfg, string("EnergyBladeMax"+damageType))*dmgMod))
			}
		}
	}
	// account for Battlemage
	// Note: we check conditions of Main Hand weapon using actor.itemList as actor.weaponData1 is populated with unarmed values when no weapon slotted.
	if skillModList.Flag(nil, "WeaponDamageAppliesToSpells") && actor.ItemList["Weapon 1"] != nil && actor.ItemList["Weapon 1"].WeaponData != nil && actor.ItemList["Weapon 1"].WeaponData[1] != nil {
		// the multiplier below exist for future possible extension of Battlemage modifiers
		multiplier := utils.OrF(skillModList.Max(skillCfg, "ImprovedWeaponDamageAppliesToSpells"), 100) / 100
		for _, damageType := range data.DamageType("").Values() {
			skillModList.AddMod(mod.NewFloat(string(damageType+"Min"), "BASE", utils.GetOr[float64](actor.WeaponData1, string(damageType+"Min"), 0)*multiplier).Source("Battlemage").Flag(mod.MFlagSpell))
			skillModList.AddMod(mod.NewFloat(string(damageType+"Max"), "BASE", utils.GetOr[float64](actor.WeaponData1, string(damageType+"Max"), 0)*multiplier).Source("Battlemage").Flag(mod.MFlagSpell))
		}
	}
	if skillModList.Flag(nil, "MinionDamageAppliesToPlayer") {
		// Minion Damage conversion from Spiritual Aid and The Scourge
		multiplier := utils.OrF(skillModList.Max(skillCfg, "ImprovedMinionDamageAppliesToPlayer"), 100) / 100
		for _, value := range skillModList.List(skillCfg, "MinionModifier") {
			cValue := value.(mod.MinionModifier)
			if cValue.Mod.Name() == "Damage" && cValue.Mod.Type() == "INC" {
				Mod := cValue.Mod
				modifiers := calclib.GetConvertedModTags(Mod, multiplier, true)
				skillModList.AddMod(mod.NewFloat("Damage", "INC", Mod.Value().Float()*multiplier).Source(Mod.GetSource()).Flag(Mod.Flags()).KeywordFlag(Mod.KeywordFlags()).Tag(modifiers...))
			}
		}
	}
	if skillModList.Flag(nil, "MinionAttackSpeedAppliesToPlayer") {
		// Minion Damage conversion from Spiritual Command
		multiplier := utils.OrF(skillModList.Max(skillCfg, "ImprovedMinionAttackSpeedAppliesToPlayer"), 100) / 100
		// Minion Attack Speed conversion from Spiritual Command
		for _, value := range skillModList.List(skillCfg, "MinionModifier") {
			cValue := value.(mod.MinionModifier)
			if cValue.Mod.Name() == "Speed" && cValue.Mod.Type() == "INC" && (cValue.Mod.Flags() == 0 || (cValue.Mod.Flags()&mod.MFlagAttack) != 0) {
				modifiers := calclib.GetConvertedModTags(cValue.Mod, multiplier, true)
				skillModList.AddMod(mod.NewFloat("Speed", "INC", cValue.Mod.Value().Float()*multiplier).Source(cValue.Mod.GetSource()).Flag(mod.MFlagAttack).KeywordFlag(cValue.Mod.KeywordFlags()).Tag(modifiers...))
			}
		}
	}
	if skillModList.Flag(nil, "SpellDamageAppliesToAttacks") {
		// Spell Damage conversion from Crown of Eyes, Kinetic Bolt, and the Wandslinger notable
		multiplier := utils.OrF(skillModList.Max(skillCfg, "ImprovedSpellDamageAppliesToAttacks"), 100) / 100
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagSpell)}, "Damage") {
			Mod := value.Mod
			if (Mod.Flags() & mod.MFlagSpell) != 0 {
				modifiers := calclib.GetConvertedModTags(Mod, multiplier, false)
				skillModList.AddMod(mod.NewFloat("Damage", "INC", Mod.Value().Float()*multiplier).Source(Mod.GetSource()).Flag((Mod.Flags() & (^mod.MFlagSpell)) | mod.MFlagAttack).KeywordFlag(Mod.KeywordFlags()).Tag(modifiers...))
				if Mod.GetSource() == "Strength" { // Prevent double-dipping from converted strength's damage bonus
					// TODO ReplaceMod
					//skillModList.ReplaceMod("PhysicalDamage", "INC", 0, "Strength", mod.MFlagMelee)
				}
			}
		}
	}
	if skillModList.Flag(nil, "CastSpeedAppliesToAttacks") {
		// Get all increases for this; assumption is that multiple sources would not stack, so find the max
		multiplier := utils.OrF(skillModList.Max(skillCfg, "ImprovedCastSpeedAppliesToAttacks"), 100) / 100
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagCast)}, "Speed") {
			// Add a new mod for all mods that are cast only
			// Replace this with a single mod for the sum?
			if (value.Mod.Flags() & mod.MFlagCast) != 0 {
				modifiers := calclib.GetConvertedModTags(value.Mod, multiplier, false)
				skillModList.AddMod(mod.NewFloat("Speed", "INC", value.Mod.Value().Float()*multiplier).Source(value.Mod.GetSource()).Flag((value.Mod.Flags() & (^mod.MFlagCast)) | mod.MFlagAttack).KeywordFlag(value.Mod.KeywordFlags()).Tag(modifiers...))
			}
		}
	}
	if skillModList.Flag(nil, "ProjectileSpeedAppliesToBowDamage") {
		// Bow mastery projectile speed to damage with bows conversion
		for _, value := range skillModList.Tabulate("INC", nil, "ProjectileSpeed") {
			skillModList.AddMod(mod.NewFloat("Damage", value.Mod.Type(), value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(mod.MFlagBow | mod.MFlagHit).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
		}
	}
	if skillModList.Flag(nil, "ClawDamageAppliesToUnarmed") {
		// Claw Damage conversion from Rigwald's Curse
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagClaw), KeywordFlags: utils.Ptr(mod.KeywordFlagHit)}, "Damage") {
			if (value.Mod.Flags() & mod.MFlagClaw) != 0 {
				skillModList.AddMod(mod.NewFloat("Damage", value.Mod.Type(), value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag((value.Mod.Flags() & (^mod.MFlagClaw)) | mod.MFlagUnarmed | mod.MFlagMelee).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			}
		}
	}
	if skillModList.Flag(nil, "ClawAttackSpeedAppliesToUnarmed") {
		// Claw Attack Speed conversion from Rigwald's Curse
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagClaw | mod.MFlagAttack | mod.MFlagHit)}, "Speed") {
			if (value.Mod.Flags()&mod.MFlagClaw) != 0 && (value.Mod.Flags()&mod.MFlagAttack) != 0 {
				skillModList.AddMod(mod.NewFloat("Speed", value.Mod.Type(), value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag((value.Mod.Flags() & (^mod.MFlagClaw)) | mod.MFlagUnarmed).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritChanceAppliesToUnarmed") {
		// Claw Crit Chance conversion from Rigwald's Curse
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagClaw | mod.MFlagHit)}, "CritChance") {
			if (value.Mod.Flags() & mod.MFlagClaw) != 0 {
				skillModList.AddMod(mod.NewFloat("CritChance", value.Mod.Type(), value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag((value.Mod.Flags() & (^mod.MFlagClaw)) | mod.MFlagUnarmed).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritChanceAppliesToMinions") {
		// Claw Crit Chance conversion from Law of the Wilds
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagClaw | mod.MFlagHit)}, "CritChance") {
			if (value.Mod.Flags() & mod.MFlagClaw) != 0 {
				env.Minion.ModDB.AddMod(mod.NewFloat("CritChance", value.Mod.Type(), value.Mod.Value().Float()).Source(value.Mod.GetSource()))
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritMultiplierAppliesToMinions") {
		// Claw Crit Multi conversion from Law of the Wilds
		for _, value := range skillModList.Tabulate("BASE", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagClaw | mod.MFlagHit)}, "CritMultiplier") {
			if (value.Mod.Flags() & mod.MFlagClaw) != 0 {
				env.Minion.ModDB.AddMod(mod.NewFloat("CritMultiplier", value.Mod.Type(), value.Mod.Value().Float()).Source(value.Mod.GetSource()))
			}
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToAccuracy") {
		// Light Radius conversion from Corona Solaris
		for _, value := range skillModList.Tabulate("INC", nil, "LightRadius") {
			skillModList.AddMod(mod.NewFloat("Accuracy", "INC", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToAreaOfEffect") {
		// Light Radius conversion from Wreath of Phrecia
		for _, value := range skillModList.Tabulate("INC", nil, "LightRadius") {
			skillModList.AddMod(mod.NewFloat("AreaOfEffect", "INC", math.Floor(value.Mod.Value().Float()/2)).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToDamage") {
		// Light Radius conversion from Wreath of Phrecia
		for _, value := range skillModList.Tabulate("INC", nil, "LightRadius") {
			skillModList.AddMod(mod.NewFloat("Damage", "INC", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
		}
	}
	if skillModList.Flag(nil, "CastSpeedAppliesToTrapThrowingSpeed") {
		// Cast Speed conversion from Slavedriver's Hand
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagCast)}, "Speed") {
			if value.Mod.Flags() == 0 || (value.Mod.Flags()&mod.MFlagCast) != 0 {
				skillModList.AddMod(mod.NewFloat("TrapThrowingSpeed", "INC", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags() & (^mod.MFlagCast) & (^mod.MFlagAttack)).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			}
		}
	}
	if skillData.ArrowSpeedAppliesToAreaOfEffect {
		// Arrow Speed conversion for Galvanic Arrow
		for _, value := range skillModList.Tabulate("INC", &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagBow)}, "ProjectileSpeed") {
			skillModList.AddMod(mod.NewFloat("AreaOfEffect", "INC", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
		}
	}
	if skillModList.Flag(nil, "SequentialProjectiles") && !skillModList.Flag(nil, "OneShotProj") && !skillModList.Flag(nil, "NoAdditionalProjectiles") && !skillModList.Flag(nil, "TriggeredBySnipe") {
		// Applies DPS multiplier based on projectile count
		skillData.DpsMultiplier = skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCount")
	}
	if skillData.GainPercentBaseWandDamage != 0 {
		mult := skillData.GainPercentBaseWandDamage / 100
		if actor.WeaponData1.Type == "Wand" && actor.WeaponData2.Type == "Wand" {
			for _, damageType := range data.DamageType("").Values() {
				skillModList.AddMod(mod.NewFloat(string(damageType+"Min"), "BASE", (utils.GetOr[float64](actor.WeaponData1, string(damageType+"Min"), 0)+utils.GetOr[float64](actor.WeaponData2, string(damageType+"Min"), 0))/2*mult).Source("Spellslinger"))
				skillModList.AddMod(mod.NewFloat(string(damageType+"Max"), "BASE", (utils.GetOr[float64](actor.WeaponData1, string(damageType+"Max"), 0)+utils.GetOr[float64](actor.WeaponData2, string(damageType+"Max"), 0))/2*mult).Source("Spellslinger"))
			}
		} else if actor.WeaponData1.Type == "Wand" {
			for _, damageType := range data.DamageType("").Values() {
				skillModList.AddMod(mod.NewFloat(string(damageType+"Min"), "BASE", utils.GetOr[float64](actor.WeaponData1, string(damageType+"Min"), 0)*mult).Source("Spellslinger"))
				skillModList.AddMod(mod.NewFloat(string(damageType+"Max"), "BASE", utils.GetOr[float64](actor.WeaponData1, string(damageType+"Max"), 0)*mult).Source("Spellslinger"))
			}
		} else if actor.WeaponData2.Type == "Wand" {
			for _, damageType := range data.DamageType("").Values() {
				skillModList.AddMod(mod.NewFloat(string(damageType+"Min"), "BASE", utils.GetOr[float64](actor.WeaponData2, string(damageType+"Min"), 0)*mult).Source("Spellslinger"))
				skillModList.AddMod(mod.NewFloat(string(damageType+"Max"), "BASE", utils.GetOr[float64](actor.WeaponData2, string(damageType+"Max"), 0)*mult).Source("Spellslinger"))
			}
		}
	}
	if skillModList.Flag(nil, "TriggeredBySnipe") && activeSkill.SkillTypes[data.SkillTypeTriggerable] {
		skillModList.AddMod(mod.NewFloat("Damage", "MORE", 165).Source("Config").Flag(mod.MFlagHit).Tag(mod.Multiplier("SnipeStage")))
		skillModList.AddMod(mod.NewFloat("Damage", "MORE", 120).Source("Config").Flag(mod.MFlagAilment).Tag(mod.Multiplier("SnipeStage")))
	}
	if skillModList.Sum(mod.TypeBase, nil, "CritMultiplierAppliesToDegen") > 0 {
		for _, value := range skillModList.Tabulate("BASE", skillCfg, "CritMultiplier") {
			if value.Mod.GetSource() != "Base" { // The global base Crit Multi doesn't apply to ailments with Perfect Agony
				skillModList.AddMod(mod.NewFloat("DotMultiplier", "BASE", math.Floor(value.Mod.Value().Float()/2)).Source(value.Mod.GetSource()).Flag(mod.MFlagAilment).Tag(mod.Condition("CriticalStrike")).Tag(value.Mod.Tags()...))
			}
		}
	}
	if skillModList.Flag(nil, "HasSeals") && activeSkill.SkillTypes[data.SkillTypeCanRapidFire] {
		// Applies DPS multiplier based on seals count
		actor.Output["SealCooldown"] = skillModList.Sum(mod.TypeBase, skillCfg, "SealGainFrequency") / calclib.Mod(skillModList, skillCfg, "SealGainFrequency")
		actor.Output["SealMax"] = skillModList.Sum(mod.TypeBase, skillCfg, "SealCount")
		actor.Output["TimeMaxSeals"] = actor.Output["SealCooldown"] * actor.Output["SealMax"]

		if skillData.HitTimeOverride == 0 {
			if skillModList.Flag(nil, "UseMaxUnleash") {
				for _, value := range skillModList.Tabulate("INC", nil, "MaxSealCrit") {
					skillModList.AddMod(mod.NewFloat("CritChance", "INC", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
				}
				env.Player.MainSkill.SkillData.DpsMultiplier = 1 + actor.Output["SealMax"]*calclib.Mod(skillModList, skillCfg, "SealRepeatPenalty")
				env.Player.MainSkill.SkillData.HitTimeOverride = max(actor.Output["TimeMaxSeals"], 1/activeSkill.ActiveEffect.GrantedEffect.CastTime()*1.1*calclib.Mod(skillModList, skillCfg, "Speed")*actor.Output["ActionSpeedMod"])
			} else {
				env.Player.MainSkill.SkillData.DpsMultiplier = 1 + 1/actor.Output["SealCooldown"]/(1/activeSkill.ActiveEffect.GrantedEffect.CastTime()*1.1*calclib.Mod(skillModList, skillCfg, "Speed")*actor.Output["ActionSpeedMod"])*calclib.Mod(skillModList, skillCfg, "SealRepeatPenalty")
			}
		}

		if breakdown != nil {
			breakdown.MultiChain("SealGainTime", BMultiChain{
				Label: "Gain frequency:",
				Base:  fmt.Sprintf("%.2fs ^8(base gain frequency)", skillModList.Sum(mod.TypeBase, skillCfg, "SealGainFrequency")),
				Total: fmt.Sprintf("= %.2fs ^8per Seal", actor.Output["SealCooldown"]),
				Items: []BMultiChainItem{
					{"%.2f ^8(increased/reduced gain frequency)", 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "SealGainFrequency")/100},
					{"%.2f ^8(action speed modifier)", actor.Output["ActionSpeedMod"]},
				},
			})
		}
	}
	if skillModList.Sum(mod.TypeBase, skillCfg, "PhysicalDamageGainAsRandom", "PhysicalDamageConvertToRandom", "PhysicalDamageGainAsColdOrLightning") > 0 {
		skillFlags[SkillFlagRandomPhys] = true
		physMode := utils.OrS(env.Build.GetStringOption("physMode"), "AVERAGE")
		for _, value := range skillModList.Tabulate("BASE", skillCfg, "PhysicalDamageGainAsRandom") {
			effVal := value.Mod.Value().Float() / 3
			if physMode == "AVERAGE" {
				skillModList.AddMod(mod.NewFloat("PhysicalDamageGainAsFire", "BASE", effVal).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
				skillModList.AddMod(mod.NewFloat("PhysicalDamageGainAsCold", "BASE", effVal).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
				skillModList.AddMod(mod.NewFloat("PhysicalDamageGainAsLightning", "BASE", effVal).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			} else if physMode == "FIRE" {
				skillModList.AddMod(mod.NewFloat("PhysicalDamageGainAsFire", "BASE", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			} else if physMode == "COLD" {
				skillModList.AddMod(mod.NewFloat("PhysicalDamageGainAsCold", "BASE", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			} else if physMode == "LIGHTNING" {
				skillModList.AddMod(mod.NewFloat("PhysicalDamageGainAsLightning", "BASE", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			}
		}
		for _, value := range skillModList.Tabulate("BASE", skillCfg, "PhysicalDamageConvertToRandom") {
			effVal := value.Mod.Value().Float() / 3
			if physMode == "AVERAGE" {
				skillModList.AddMod(mod.NewFloat("PhysicalDamageConvertToFire", "BASE", effVal).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
				skillModList.AddMod(mod.NewFloat("PhysicalDamageConvertToCold", "BASE", effVal).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
				skillModList.AddMod(mod.NewFloat("PhysicalDamageConvertToLightning", "BASE", effVal).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			} else if physMode == "FIRE" {
				skillModList.AddMod(mod.NewFloat("PhysicalDamageConvertToFire", "BASE", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			} else if physMode == "COLD" {
				skillModList.AddMod(mod.NewFloat("PhysicalDamageConvertToCold", "BASE", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			} else if physMode == "LIGHTNING" {
				skillModList.AddMod(mod.NewFloat("PhysicalDamageConvertToLightning", "BASE", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			}
		}
		for _, value := range skillModList.Tabulate("BASE", skillCfg, "PhysicalDamageGainAsColdOrLightning") {
			effVal := value.Mod.Value().Float() / 2
			if physMode == "AVERAGE" || physMode == "FIRE" {
				skillModList.AddMod(mod.NewFloat("PhysicalDamageGainAsCold", "BASE", effVal).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
				skillModList.AddMod(mod.NewFloat("PhysicalDamageGainAsLightning", "BASE", effVal).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			} else if physMode == "COLD" {
				skillModList.AddMod(mod.NewFloat("PhysicalDamageGainAsCold", "BASE", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			} else if physMode == "LIGHTNING" {
				skillModList.AddMod(mod.NewFloat("PhysicalDamageGainAsLightning", "BASE", value.Mod.Value().Float()).Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
			}
		}
	}

	runSkillFunc("preSkillTypeFunc")

	isAttack := skillFlags[SkillFlagAttack]

	// Calculate skill type stats
	if skillFlags[SkillFlagMinion] {
		if activeSkill.Minion != nil && activeSkill.Minion.MinionData.Limit != "" {
			actor.Output["ActiveMinionLimit"] = math.Floor(calclib.Val(skillModList, activeSkill.Minion.MinionData.Limit, skillCfg))
		}
	}
	if skillFlags[SkillFlagChaining] {
		if skillModList.Flag(skillCfg, "CannotChain") {
			actor.OutputStrings["ChainMaxString"] = "Cannot chain"
		} else {
			actor.Output["ChainMax"] = skillModList.Sum(mod.TypeBase, skillCfg, "ChainCountMax", utils.Ternary(!skillFlags[SkillFlagProjectile], "BeamChainCountMax", ""))
			actor.OutputStrings["ChainMaxString"] = fmt.Sprint(actor.Output["ChainMax"])
			actor.Output["Chain"] = min(actor.Output["ChainMax"], skillModList.Sum(mod.TypeBase, skillCfg, "ChainCount"))
			actor.Output["ChainRemaining"] = max(0, actor.Output["ChainMax"]-actor.Output["Chain"])
		}
	}
	if skillFlags[SkillFlagProjectile] {
		if skillModList.Flag(nil, "PointBlank") {
			skillModList.AddMod(mod.NewFloat("Damage", "MORE", 30).Source("Point Blank").Flag(mod.MFlagAttack | mod.MFlagProjectile).Tag(mod.DistanceRamp([][]float64{{10, 1}, {35, 0}, {150, -1}})))
		}
		if skillModList.Flag(nil, "FarShot") {
			skillModList.AddMod(mod.NewFloat("Damage", "MORE", 100).Source("Far Shot").Flag(mod.MFlagAttack | mod.MFlagProjectile).Tag(mod.DistanceRamp([][]float64{{10, -0.2}, {35, 0}, {70, 0.6}})))
		}
		if skillModList.Flag(skillCfg, "NoAdditionalProjectiles") {
			actor.Output["ProjectileCount"] = 1
		} else {
			projBase := skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCount")
			projMore := skillModList.More(skillCfg, "ProjectileCount")
			actor.Output["ProjectileCount"] = math.Floor(projBase * projMore)
		}
		if skillModList.Flag(skillCfg, "AdditionalProjectilesAddBouncesInstead") {
			projBase := skillModList.Sum(mod.TypeBase, skillCfg, "ProjectileCount") + skillModList.Sum(mod.TypeBase, skillCfg, "BounceCount") - 1
			projMore := skillModList.More(skillCfg, "ProjectileCount")
			actor.Output["BounceCount"] = math.Floor(projBase * projMore)
		}
		if skillModList.Flag(skillCfg, "CannotFork") {
			actor.OutputStrings["ForkCountString"] = "Cannot fork"
		} else if skillModList.Flag(skillCfg, "ForkOnce") {
			skillFlags[SkillFlagForking] = true
			if skillModList.Flag(skillCfg, "ForkTwice") {
				actor.Output["ForkCountMax"] = min(skillModList.Sum(mod.TypeBase, skillCfg, "ForkCountMax"), 2)
			} else {
				actor.Output["ForkCountMax"] = min(skillModList.Sum(mod.TypeBase, skillCfg, "ForkCountMax"), 1)
			}
			actor.Output["ForkedCount"] = min(actor.Output["ForkCountMax"], skillModList.Sum(mod.TypeBase, skillCfg, "ForkedCount"))
			actor.OutputStrings["ForkCountString"] = fmt.Sprint(actor.Output["ForkCountMax"])
			actor.Output["ForkRemaining"] = max(0, actor.Output["ForkCountMax"]-actor.Output["ForkedCount"])
		} else {
			actor.OutputStrings["ForkCountString"] = "0"
		}
		if skillModList.Flag(skillCfg, "CannotPierce") {
			actor.Output["PierceCount"] = 0
			actor.OutputStrings["PierceCountString"] = "Cannot pierce"
		} else {
			if skillModList.Flag(skillCfg, "PierceAllTargets") || enemyDB.Flag(nil, "AlwaysPierceSelf") {
				actor.Output["PierceCount"] = 100
				actor.OutputStrings["PierceCountString"] = "All targets"
			} else {
				actor.Output["PierceCount"] = skillModList.Sum(mod.TypeBase, skillCfg, "PierceCount")
				actor.OutputStrings["PierceCountString"] = fmt.Sprint(actor.Output["PierceCount"])
			}
			if actor.Output["PierceCount"] > 0 {
				skillFlags[SkillFlagPiercing] = true
			}
			actor.Output["PiercedCount"] = min(actor.Output["PierceCount"], skillModList.Sum(mod.TypeBase, skillCfg, "PiercedCount"))
		}
		actor.Output["ProjectileSpeedMod"] = calclib.Mod(skillModList, skillCfg, "ProjectileSpeed")
		if breakdown != nil {
			breakdown.Mod("ProjectileSpeedMod", skillModList, skillCfg, "ProjectileSpeed")
		}
	}
	/*
		TODO skillFlags.melee
		if skillFlags.melee {
			if skillFlags.weapon1Attack {
				actor.weaponRange1 = (actor.weaponData1.range and actor.weaponData1.range + skillModList.Sum(mod.TypeBase, activeSkill.weapon1Cfg, "MeleeWeaponRange")) or (6 + skillModList.Sum(mod.TypeBase, skillCfg, "UnarmedRange"))
			}
			if skillFlags.weapon2Attack {
				actor.weaponRange2 = (actor.weaponData2.range and actor.weaponData2.range + skillModList.Sum(mod.TypeBase, activeSkill.weapon2Cfg, "MeleeWeaponRange")) or (6 + skillModList.Sum(mod.TypeBase, skillCfg, "UnarmedRange"))
			}
			if activeSkill.skillTypes[SkillType.MeleeSingleTarget] {
				range := 100
				if skillFlags.weapon1Attack {
					range = min(range, actor.weaponRange1)
				}
				if skillFlags.weapon2Attack {
					range = min(range, actor.weaponRange2)
				}
				actor.Output["WeaponRange"] = range + 2
				if breakdown != nil {
					breakdown.WeaponRange = {
						radius = actor.Output["WeaponRange"]
					}
				}
			}
		}
	*/
	/*
		if skillFlags.area or skillData.radius or (skillFlags.mine and activeSkill.skillTypes[SkillType.Aura]) {
			calcAreaOfEffect(skillModList, skillCfg, skillData, skillFlags, output, breakdown)
		}
	*/
	if activeSkill.SkillTypes[data.SkillTypeAura] {
		actor.Output["AuraEffectMod"] = calclib.Mod(skillModList, skillCfg, "AuraEffect")
		if breakdown != nil {
			breakdown.Mod("AuraEffectMod", skillModList, skillCfg, "AuraEffect")
		}
	}
	/*
		TODO SkillType.HasReservation
		if activeSkill.skillTypes[SkillType.HasReservation] and not activeSkill.skillTypes[SkillType.ReservationBecomesCost] {
			for _, pool := range ({"Life", "Mana"}) {
				actor.Output[pool + "ReservedMod"] = 0
				if calclib.Mod(skillModList, skillCfg, "SupportManaMultiplier") > 0 and calclib.Mod(skillModList, skillCfg, pool + "Reserved", "Reserved") > 0 {
					actor.Output[pool + "ReservedMod"] = calclib.Mod(skillModList, skillCfg, pool + "Reserved", "Reserved") * calclib.Mod(skillModList, skillCfg, "SupportManaMultiplier") / max(0, calclib.Mod(skillModList, skillCfg, pool + "ReservationEfficiency", "ReservationEfficiency"))
				}
				if breakdown != nil {
					inc := skillModList.Sum(mod.TypeIncrease, skillCfg, pool + "Reserved", "Reserved", "SupportManaMultiplier")
					more := skillModList.More(skillCfg, pool + "Reserved", "Reserved", "SupportManaMultiplier")
					if inc != 0 and more != 1 {
						breakdown[pool + "ReservedMod"] = {
							fmt.Sprintf("%.2f ^8(increased/reduced)", 1 + inc/100),
							fmt.Sprintf("x %.2f ^8(more/less)", more),
							fmt.Sprintf("/ %.2f ^8(reservation efficiency)", calclib.Mod(skillModList, skillCfg, pool + "ReservationEfficiency", "ReservationEfficiency")),
							fmt.Sprintf("= %.2f", actor.Output[pool + "ReservedMod"]),
						}
					}
				}
			}
		}
	*/
	if activeSkill.SkillTypes[data.SkillTypeHex] || activeSkill.SkillTypes[data.SkillTypeMark] {
		actor.Output["CurseEffectMod"] = calclib.Mod(skillModList, skillCfg, "CurseEffect")
		if breakdown != nil {
			breakdown.Mod("CurseEffectMod", skillModList, skillCfg, "CurseEffect")
		}
	}
	if (skillFlags[SkillFlagTrap] || skillFlags[SkillFlagMine]) && !(skillData.TrapCooldown != 0 || skillData.Cooldown != 0) {
		skillFlags[SkillFlagNotAverage] = true
		skillFlags[SkillFlagShowAverage] = false
		skillData.ShowAverage = false
	}
	/*
		TODO skillFlags.trap
		if skillFlags.trap {
			baseSpeed := 1 / skillModList.Sum(mod.TypeBase, skillCfg, "TrapThrowingTime")
			timeMod := calclib.Mod(skillModList, skillCfg, "SkillTrapThrowingTime")
			if timeMod > 0 {
				baseSpeed = baseSpeed * (1 / timeMod)
			}
			actor.Output["TrapThrowingSpeed"] = baseSpeed * calclib.Mod(skillModList, skillCfg, "TrapThrowingSpeed") * actor.Output["ActionSpeedMod"]
			actor.Output["TrapThrowingSpeed"] = min(actor.Output["TrapThrowingSpeed"], data.ServerTickRate)
			actor.Output["TrapThrowingTime"] = 1 / actor.Output["TrapThrowingSpeed"]
			skillData.timeOverride = actor.Output["TrapThrowingTime"]
			if breakdown != nil {
				breakdown.TrapThrowingSpeed = { }
				breakdown.multiChain(breakdown.TrapThrowingSpeed, {
					label = "Throwing rate:",
					base = fmt.Sprintf("%.2f ^8(base throwing rate)", baseSpeed),
					{ "%.2f ^8(increased/reduced throwing speed)", 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "TrapThrowingSpeed") / 100 },
					{ "%.2f ^8(more/less throwing speed)", skillModList.More(skillCfg, "TrapThrowingSpeed") },
					{ "%.2f ^8(action speed modifier)",  actor.Output["ActionSpeedMod"] },
					total = fmt.Sprintf("= %.2f ^8per second", actor.Output["TrapThrowingSpeed"]),
				})
			}
			if breakdown and timeMod > 0 {
				breakdown.TrapThrowingTime = { }
				breakdown.multiChain(breakdown.TrapThrowingTime, {
					label = "Throwing time:",
					base = fmt.Sprintf("%.2f ^8(base throwing time)", 1 / (actor.Output["TrapThrowingSpeed"] * timeMod)),
					{ "%.2f ^8(total modifier)", timeMod },
					total = fmt.Sprintf("= %.2f ^8seconds per throw", actor.Output["TrapThrowingTime"]),
				})
			}

			baseCooldown := skillData.trapCooldown or skillData.cooldown
			if baseCooldown {
				actor.Output["TrapCooldown"] = baseCooldown / calclib.Mod(skillModList, skillCfg, "CooldownRecovery")
				actor.Output["TrapCooldown"] = m_ceil(actor.Output["TrapCooldown"] * data.ServerTickRate) / data.ServerTickRate
				if breakdown != nil {
					breakdown.TrapCooldown = {
						fmt.Sprintf("%.2fs ^8(base)", skillData.trapCooldown or skillData.cooldown or 4),
						fmt.Sprintf("/ %.2f ^8(increased/reduced cooldown recovery)", 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "CooldownRecovery") / 100),
						fmt.Sprintf("rounded up to nearest server tick"),
						fmt.Sprintf("= %.3fs", actor.Output["TrapCooldown"])
					}
				}
			}
			incArea, moreArea := calclib.Mods(skillModList, skillCfg, "TrapTriggerAreaOfEffect")
			areaMod := round(round(incArea * moreArea, 10), 2)
			actor.Output["TrapTriggerRadius"] = calcRadius(data.TrapTriggerRadiusBase, areaMod)
			if breakdown != nil {
				incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint := calcRadiusBreakpoints(data.TrapTriggerRadiusBase, incArea, moreArea)
				breakdown.TrapTriggerRadius = breakdown.area(data.TrapTriggerRadiusBase, areaMod, actor.Output["TrapTriggerRadius"], incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint)
			}
		} else if skillData.cooldown {
			actor.Output["Cooldown"] = calcSkillCooldown(skillModList, skillCfg, skillData)
			if breakdown != nil {
				breakdown.Cooldown = {
					fmt.Sprintf("%.2fs ^8(base)", skillData.cooldown + skillModList.Sum(mod.TypeBase, skillCfg, "CooldownRecovery")),
					fmt.Sprintf("/ %.2f ^8(increased/reduced cooldown recovery)", 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "CooldownRecovery") / 100),
					fmt.Sprintf("rounded up to nearest server tick"),
					fmt.Sprintf("= %.3fs", actor.Output["Cooldown"])
				}
			}
		}
	*/
	/*
		TODO skillFlags.mine
		if skillFlags.mine {
			baseSpeed := 1 / skillModList.Sum(mod.TypeBase, skillCfg, "MineLayingTime")
			timeMod := calclib.Mod(skillModList, skillCfg, "SkillMineThrowingTime")
			if timeMod > 0 {
				baseSpeed = baseSpeed * (1 / timeMod)
			}
			actor.Output["MineLayingSpeed"] = baseSpeed * calclib.Mod(skillModList, skillCfg, "MineLayingSpeed") * actor.Output["ActionSpeedMod"]
			actor.Output["MineLayingSpeed"] = min(actor.Output["MineLayingSpeed"], data.ServerTickRate)
			actor.Output["MineLayingTime"] = 1 / actor.Output["MineLayingSpeed"]
			skillData.timeOverride = actor.Output["MineLayingTime"]
			if breakdown != nil {
				breakdown.MineLayingTime = { }
				breakdown.multiChain(breakdown.MineLayingTime, {
					label = "Throwing rate:",
					base = fmt.Sprintf("%.2f ^8(base throwing rate)", baseSpeed),
					{ "%.2f ^8(increased/reduced throwing speed)", 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "MineLayingSpeed") / 100 },
					{ "%.2f ^8(more/less throwing speed)", skillModList.More(skillCfg, "MineLayingSpeed") },
					{ "%.2f ^8(action speed modifier)",  actor.Output["ActionSpeedMod"] },
					total = fmt.Sprintf("= %.2f ^8per second", actor.Output["MineLayingSpeed"]),
				})
			}
			if breakdown and timeMod > 0 {
				breakdown.MineThrowingTime = { }
				breakdown.multiChain(breakdown.MineThrowingTime, {
				label = "Throwing time:",
					base = fmt.Sprintf("%.2f ^8(base throwing time)", 1 / (actor.Output["MineLayingSpeed"] * timeMod)),
					{ "%.2f ^8(total modifier)", timeMod },
					total = fmt.Sprintf("= %.2f ^8seconds per throw", actor.Output["MineLayingTime"]),
				})
			}

			incArea, moreArea := calclib.Mods(skillModList, skillCfg, "MineDetonationAreaOfEffect")
			areaMod := round(round(incArea * moreArea, 10), 2)
			actor.Output["MineDetonationRadius"] = calcRadius(data.MineDetonationRadiusBase, areaMod)
			if breakdown != nil {
				incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint := calcRadiusBreakpoints(data.MineDetonationRadiusBase, incArea, moreArea)
				breakdown.MineDetonationRadius = breakdown.area(data.MineDetonationRadiusBase, areaMod, actor.Output["MineDetonationRadius"], incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint)
			}
			if activeSkill.skillTypes[SkillType.Aura] {
				actor.Output["MineAuraRadius"] = calcRadius(data.MineAuraRadiusBase, actor.Output["AreaOfEffectMod"])
				if breakdown != nil {
					incArea, moreArea := calclib.Mods(skillModList, skillCfg, "AreaOfEffect")
					incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint := calcRadiusBreakpoints(data.MineAuraRadiusBase, incArea, moreArea)
					breakdown.MineAuraRadius = breakdown.area(data.MineAuraRadiusBase, actor.Output["AreaOfEffectMod"], actor.Output["MineAuraRadius"], incAreaBreakpoint, moreAreaBreakpoint, redAreaBreakpoint, lessAreaBreakpoint)
				}
			}
		}
	*/
	if skillFlags[SkillFlagTotem] {
		baseSpeed := 1 / skillModList.Sum(mod.TypeBase, skillCfg, "TotemPlacementTime")
		if skillFlags[SkillFlagBallista] {
			baseSpeed = 1 / skillModList.Sum(mod.TypeBase, skillCfg, "BallistaPlacementTime")
		}
		actor.Output["TotemPlacementSpeed"] = baseSpeed * calclib.Mod(skillModList, skillCfg, "TotemPlacementSpeed") * actor.Output["ActionSpeedMod"]
		actor.Output["TotemPlacementTime"] = 1 / actor.Output["TotemPlacementSpeed"]
		if breakdown != nil {
			breakdown.MultiChain("TotemPlacementTime", BMultiChain{
				Label: "Placement speed:",
				Base:  fmt.Sprintf("%.2f ^8(base placement speed)", baseSpeed),
				Total: fmt.Sprintf("= %.2f ^8per second", actor.Output["TotemPlacementSpeed"]),
				Items: []BMultiChainItem{
					{"%.2f ^8(increased/reduced placement speed)", 1 + skillModList.Sum(mod.TypeIncrease, skillCfg, "TotemPlacementSpeed")/100},
					{"%.2f ^8(more/less placement speed)", skillModList.More(skillCfg, "TotemPlacementSpeed")},
					{"%.2f ^8(action speed modifier)", actor.Output["ActionSpeedMod"]},
				},
			})
		}
		actor.Output["ActiveTotemLimit"] = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
		actor.Output["TotemsSummoned"] = utils.Or(env.ModDB.Override(nil, "TotemsSummoned"), actor.Output["ActiveTotemLimit"])
		if breakdown != nil {
			breakdown.AddLine("ActiveTotemLimit",
				"Totems Summoned: "+fmt.Sprint(actor.Output["TotemsSummoned"])+(utils.Ternary(env.Build.GetNumberOption("TotemsSummoned") != 0, " ^8(overridden from the Configuration tab)", " ^8(can be overridden in the Configuration tab)")),
			)
		}
		actor.Output["TotemLifeMod"] = calclib.Mod(skillModList, skillCfg, "TotemLife")
		actor.Output["TotemLife"] = utils.RoundTo(math.Floor(float64(poe.DefaultMonsterStats[skillData.TotemLevel].AllyLife)*(float64(poe.MonsterVarieties[poe.SkillTotemVariations[activeSkill.SkillTotemId].MonsterVarietiesKey].LifeMultiplier)/100))*actor.Output["TotemLifeMod"], 0)
		if breakdown != nil {
			breakdown.Mod("TotemLifeMod", skillModList, skillCfg, "TotemLife")
			/*
				TODO Breakdown
				breakdown.AddLine("TotemLife",
					"Totem level: "+skillData.TotemLevel,
					data.MonsterAllyLifeTable[skillData.TotemLevel]+" ^8(base life for a level "+skillData.TotemLevel+" monster)",
					"x "+env.data.totemLifeMult[activeSkill.skillTotemId]+" ^8(life multiplier for this totem type)",
					"x "+Output["TotemLifeMod"]+" ^8(totem life modifier)",
					"= "+Output["TotemLife"],
				)
			*/
		}
	}
	/*
		TODO skillFlags.brand
		if skillFlags.brand {
			actor.Output["BrandAttachmentRange"] = data.BrandAttachmentRangeBase * calclib.Mod(skillModList, skillCfg, "BrandAttachmentRange")
			actor.Output["ActiveBrandLimit"] = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveBrandLimit")
			if breakdown != nil {
				breakdown.BrandAttachmentRange = { radius = actor.Output["BrandAttachmentRange"] }
			}
		}

	*/
	if skillFlags[SkillFlagWarcry] {
		// TODO Warcry
		//actor.Output["WarcryCastTime"] = CalcWarcryCastTime(skillModList, skillCfg, actor)
	}
	/*
		TODO skillFlags.corpse
		if skillFlags.corpse {
			actor.Output["CorpseLevel"] = skillModList.Sum(mod.TypeBase, skillCfg, "CorpseLevel")
			actor.Output["BaseCorpseLife"] = env.data.monsterLifeTable[actor.Output["CorpseLevel"] or 1] * (env.data.monsterVarietyLifeMult[skillData.corpseMonsterVariety] or 1) * (env.data.mapLevelLifeMult[env.enemyLevel] or 1)
			actor.Output["CorpseLifeInc"] = 1 + (skillModList.Sum(mod.TypeIncrease, skillCfg, "CorpseLife") or 0) / 100
			actor.Output["CorpseLife"] = actor.Output["BaseCorpseLife"] * actor.Output["CorpseLifeInc"]
			if breakdown != nil {
				breakdown.CorpseLife = {
					fmt.Sprintf("%d ^8(base life of a level %d monster)", env.data.monsterLifeTable[actor.Output["CorpseLevel"] or 1], actor.Output["CorpseLevel"] or "n/a"),
					fmt.Sprintf("x %.2f ^8(%s variety multiplier)", env.data.monsterVarietyLifeMult[skillData.corpseMonsterVariety] or 1, skillData.corpseMonsterVariety),
					fmt.Sprintf("x %.2f ^8(map level %d monster life multiplier from config)", env.data.mapLevelLifeMult[env.enemyLevel] or 1, env.enemyLevel),
					fmt.Sprintf(" = %d ^8(base corpse life)", actor.Output["BaseCorpseLife"]),
					fmt.Sprintf(""),
					fmt.Sprintf("x %.2f ^8(corpse maximum life increases)", actor.Output["CorpseLifeInc"]),
					fmt.Sprintf(" = %d", actor.Output["CorpseLife"]),
				}
			}
		}
	*/
	/*
		TODO // General's Cry
		if skillData.triggeredByGeneralsCry {
			mirageActiveSkill := nil

			// Find the active General's Cry gem to get active properties
			for _, skill := range (actor.activeSkillList) {
				if skill.activeEffect.grantedEffect.name == "General's Cry" and actor.mainSkill.socketGroup.slot == activeSkill.socketGroup.slot {
					mirageActiveSkill = skill
					break
				}
			}

			if mirageActiveSkill {
				cooldown := calcSkillCooldown(mirageActiveSkill.skillModList, mirageActiveSkill.skillCfg, mirageActiveSkill.skillData)

				// Non-channelled skills only attack once, disregard attack rate
				if not activeSkill.skillTypes[SkillType.Channel] {
					skillData.timeOverride = 1
				}

				// Supported Attacks Count as Exerted
				for _, value := range (env.modDB.Tabulate("INC", skillCfg, "ExertIncrease")) {
					mod := value.mod
					skillModList.AddMod("Damage", mod.type, mod.value, mod.source, mod.flags, mod.keywordFlags)
				}
				for _, value := range (env.modDB.Tabulate("MORE", skillCfg, "ExertIncrease")) {
					mod := value.mod
					skillModList.AddMod("Damage", mod.type, mod.value, mod.source, mod.flags, mod.keywordFlags)
				}
				for _, value := range (env.modDB.Tabulate("MORE", skillCfg, "ExertAttackIncrease")) {
					mod := value.mod
					skillModList.AddMod("Damage", mod.type, mod.value, mod.source, mod.flags, mod.keywordFlags)
				}
				for _, value := range (env.modDB.Tabulate("BASE", skillCfg, "ExertDoubleDamageChance")) {
					mod := value.mod
					skillModList.AddMod("DoubleDamageChance", mod.type, mod.value, mod.source, mod.flags, mod.keywordFlags)
				}
				maxMirageWarriors := 0
				for _, value := range (mirageActiveSkill.skillModList.Tabulate("BASE", skillCfg, "GeneralsCryDoubleMaxCount")) {
					mod := value.mod
					skillModList.AddMod("QuantityMultiplier", mod.type, mod.value, mod.source, mod.flags, mod.keywordFlags)
					maxMirageWarriors = maxMirageWarriors + mod.value
				}
				env.player.mainSkill.infoMessage = tostring(maxMirageWarriors) + " GC Mirage Warriors using " + activeSkill.activeEffect.grantedEffect.name

				// Scale dps with GC's cooldown
				if skillData.dpsMultiplier {
					skillData.dpsMultiplier = skillData.dpsMultiplier * (1 / cooldown)
				} else {
					skillData.dpsMultiplier = 1 / cooldown
				}
			}
		}
	*/
	debuffDurationMult := float64(1)
	// Skill duration
	if env.ModeEffective {
		debuffDurationMult = 1 / max(data.BuffExpirationSlowCap, calclib.Mod(enemyDB, skillCfg, "BuffExpireFaster"))
	}
	{
		actor.Output["DurationMod"] = calclib.Mod(skillModList, skillCfg, "Duration", "PrimaryDuration", "SkillAndDamagingAilmentDuration", utils.Ternary(skillData.MineDurationAppliesToSkill, "MineDuration", ""))
		if breakdown != nil {
			breakdown.Mod("DurationMod", skillModList, skillCfg, "Duration", "PrimaryDuration", "SkillAndDamagingAilmentDuration", utils.Ternary(skillData.MineDurationAppliesToSkill, "MineDuration", ""))
			if breakdown.Has("DurationMod") && skillData.DurationSecondary != 0 {
				breakdown.AddLine("DurationMod, 1", "Primary duration:")
			}
		}
		durationBase := (skillData.Duration) + skillModList.Sum(mod.TypeBase, skillCfg, "Duration", "PrimaryDuration")
		if durationBase > 0 {
			actor.Output["Duration"] = durationBase * actor.Output["DurationMod"]
			if skillData.Debuff {
				actor.Output["Duration"] = actor.Output["Duration"] * debuffDurationMult
			}
			actor.Output["Duration"] = math.Ceil(actor.Output["Duration"]*data.ServerTickRate) / data.ServerTickRate
			if breakdown != nil && actor.Output["Duration"] != durationBase {
				breakdown.AddLine("Duration",
					fmt.Sprintf("%.2fs ^8(base)", durationBase),
				)
				if actor.Output["DurationMod"] != 1 {
					breakdown.AddLine("Duration", fmt.Sprintf("x %.4f ^8(duration modifier)", actor.Output["DurationMod"]))
				}
				if skillData.Debuff && debuffDurationMult != 1 {
					breakdown.AddLine("Duration", fmt.Sprintf("/ %.3f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
				}
				breakdown.AddLine("Duration", fmt.Sprintf("rounded up to nearest server tick"))
				breakdown.AddLine("Duration", fmt.Sprintf("= %.3fs", actor.Output["Duration"]))
			}
		}
		durationBase = (skillData.DurationSecondary) + skillModList.Sum(mod.TypeBase, skillCfg, "Duration", "SecondaryDuration")
		if durationBase > 0 {
			durationMod := calclib.Mod(skillModList, skillCfg, "Duration", "SecondaryDuration", "SkillAndDamagingAilmentDuration", utils.Ternary(skillData.MineDurationAppliesToSkill, "MineDuration", ""))
			actor.Output["DurationSecondary"] = durationBase * durationMod
			if skillData.DebuffSecondary {
				actor.Output["DurationSecondary"] = actor.Output["DurationSecondary"] * debuffDurationMult
			}
			actor.Output["DurationSecondary"] = math.Ceil(actor.Output["DurationSecondary"]*data.ServerTickRate) / data.ServerTickRate
			if breakdown != nil && actor.Output["DurationSecondary"] != durationBase {
				breakdown.Mod("SecondaryDurationMod", skillModList, skillCfg, "Duration", "SecondaryDuration", "SkillAndDamagingAilmentDuration", utils.Ternary(skillData.MineDurationAppliesToSkill, "MineDuration", ""))
				if breakdown.Has("SecondaryDurationMod") {
					breakdown.AddLine("SecondaryDurationMod, 1", "Secondary duration:")
				}
				breakdown.AddLine("DurationSecondary",
					fmt.Sprintf("%.2fs ^8(base)", durationBase),
				)
				if actor.Output["DurationMod"] != 1 {
					breakdown.AddLine("DurationSecondary", fmt.Sprintf("x %.4f ^8(duration modifier)", durationMod))
				}
				if skillData.DebuffSecondary && debuffDurationMult != 1 {
					breakdown.AddLine("DurationSecondary", fmt.Sprintf("/ %.3f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
				}
				breakdown.AddLine("DurationSecondary", fmt.Sprintf("rounded up to nearest server tick"))
				breakdown.AddLine("DurationSecondary", fmt.Sprintf("= %.3fs", actor.Output["DurationSecondary"]))
			}
		}
		durationBase = skillData.AuraDuration
		if durationBase > 0 {
			durationMod := calclib.Mod(skillModList, skillCfg, "Duration", "SkillAndDamagingAilmentDuration")
			actor.Output["AuraDuration"] = durationBase * durationMod
			actor.Output["AuraDuration"] = math.Ceil(actor.Output["AuraDuration"]*data.ServerTickRate) / data.ServerTickRate
			if breakdown != nil && actor.Output["AuraDuration"] != durationBase {
				breakdown.AddLine("AuraDuration",
					fmt.Sprintf("%.2fs ^8(base)", durationBase),
					fmt.Sprintf("x %.4f ^8(duration modifier)", durationMod),
					fmt.Sprintf("rounded up to nearest server tick"),
					fmt.Sprintf("= %.3fs", actor.Output["AuraDuration"]),
				)
			}
		}
		durationBase = skillData.ReserveDuration
		if durationBase > 0 {
			durationMod := calclib.Mod(skillModList, skillCfg, "Duration", "SkillAndDamagingAilmentDuration")
			actor.Output["ReserveDuration"] = durationBase * durationMod
			actor.Output["ReserveDuration"] = math.Ceil(actor.Output["ReserveDuration"]*data.ServerTickRate) / data.ServerTickRate
			if breakdown != nil && actor.Output["ReserveDuration"] != durationBase {
				breakdown.AddLine("ReserveDuration",
					fmt.Sprintf("%.2fs ^8(base)", durationBase),
					fmt.Sprintf("x %.4f ^8(duration modifier)", durationMod),
					fmt.Sprintf("rounded up to nearest server tick"),
					fmt.Sprintf("= %.3fs", actor.Output["ReserveDuration"]),
				)
			}
		}
	}

	// Calculate costs (may be slightly off due to rounding differences)
	costs := map[string]*struct {
		Type           string
		Upfront        bool
		Percent        bool
		Text           string
		BaseCost       float64
		TotalCost      float64
		BaseCostNoMult float64
	}{
		"Mana":                 {Type: "Mana", Upfront: true, Percent: false, Text: "mana", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"Life":                 {Type: "Life", Upfront: true, Percent: false, Text: "life", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"ES":                   {Type: "ES", Upfront: true, Percent: false, Text: "ES", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"Rage":                 {Type: "Rage", Upfront: true, Percent: false, Text: "rage", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"ManaPercent":          {Type: "Mana", Upfront: true, Percent: true, Text: "mana", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"LifePercent":          {Type: "Life", Upfront: true, Percent: true, Text: "life", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"ManaPerMinute":        {Type: "Mana", Upfront: false, Percent: false, Text: "mana/s", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"LifePerMinute":        {Type: "Life", Upfront: false, Percent: false, Text: "life/s", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"ManaPercentPerMinute": {Type: "Mana", Upfront: false, Percent: true, Text: "mana/s", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"LifePercentPerMinute": {Type: "Life", Upfront: false, Percent: true, Text: "life/s", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"ESPerMinute":          {Type: "ES", Upfront: false, Percent: false, Text: "ES/s", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
		"ESPercentPerMinute":   {Type: "ES", Upfront: false, Percent: true, Text: "ES/s", BaseCost: 0, TotalCost: 0, BaseCostNoMult: 0},
	}

	// First pass to calculate base costs.  Used for cost conversion (e.g. Petrified Blood)
	for resource, val := range costs {
		skillCost := float64(activeSkill.ActiveEffect.GrantedEffectLevel.Cost[resource])
		baseCost := utils.RoundTo(utils.Ternary[float64](skillCost > 0, skillCost/float64(poe.CostTypesByID[resource].Divisor), 0), 2)
		baseCostNoMult := skillModList.Sum(mod.TypeBase, skillCfg, resource+"CostNoMult")
		totalCost := float64(0)
		if val.Upfront {
			baseCost = baseCost + skillModList.Sum(mod.TypeBase, skillCfg, resource+"CostBase")
			if resource == "Mana" && skillData.BaseManaCostIsAtLeastPercentUnreservedMana > 0 {
				baseCost = max(baseCost, math.Floor((actor.Output["ManaUnreserved"])*skillData.BaseManaCostIsAtLeastPercentUnreservedMana/100))
			}
			totalCost = skillModList.Sum(mod.TypeBase, skillCfg, resource+"Cost")
			if activeSkill.SkillTypes[data.SkillTypeReservationBecomesCost] {
				reservedFlat := utils.GetOr(activeSkill.SkillData, val.Text+"ReservationFlat", utils.GetOr(activeSkill.ActiveEffect.GrantedEffectLevel, val.Text+"ReservationFlat", float64(0)))
				baseCost = baseCost + reservedFlat
				reservedPercent := utils.GetOr(activeSkill.SkillData, val.Text+"ReservationPercent", utils.GetOr(activeSkill.ActiveEffect.GrantedEffectLevel, val.Text+"ReservationPercent", float64(0)))
				baseCost = baseCost + (math.Floor((actor.Output[resource]) * reservedPercent / 100))
			}
		}
		if val.Type == "Mana" && skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") {
			target := strings.ReplaceAll(resource, "Mana", "Life")
			costs[target].BaseCost = costs[target].BaseCost + baseCost
			baseCost = 0
			costs[target].TotalCost = costs[target].TotalCost + totalCost
			totalCost = 0
			costs[target].BaseCostNoMult = costs[target].BaseCostNoMult + baseCostNoMult
			baseCostNoMult = 0
		}
		// Extra cost (e.g. Petrified Blood) calculations happen after cost conversion (e.g. Blood Magic)
		if val.Type == "Mana" && skillModList.Sum(mod.TypeBase, skillCfg, "ManaCostAsLifeCost") > 0 {
			target := strings.ReplaceAll(resource, "Mana", "Life")
			costs[target].BaseCost = costs[target].BaseCost + (baseCost+baseCostNoMult)*skillModList.Sum(mod.TypeBase, skillCfg, "ManaCostAsLifeCost")/100
		}
		val.BaseCost = val.BaseCost + baseCost
		val.TotalCost = val.TotalCost + totalCost
		val.BaseCostNoMult = val.BaseCostNoMult + baseCostNoMult
	}
	for resource, val := range costs {
		dec := utils.Ternary(val.Upfront, 0, 2)
		costName := (utils.Ternary(val.Upfront, resource, strings.ReplaceAll(resource, "Minute", "Second"))) + "Cost"
		mult := utils.FloorTo(skillModList.More(skillCfg, "SupportManaMultiplier"), 2)
		more := utils.FloorTo(skillModList.More(skillCfg, val.Type+"Cost", "Cost"), 2)
		inc := skillModList.Sum(mod.TypeIncrease, skillCfg, val.Type+"Cost", "Cost")
		actor.Output[costName] = utils.FloorTo(val.BaseCost*mult+val.BaseCostNoMult, dec)
		actor.Output[costName] = utils.FloorTo(math.Abs(inc/100)*actor.Output[costName], dec)*(utils.Ternary[float64](inc >= 0, 1, -1)) + actor.Output[costName]
		actor.Output[costName] = utils.FloorTo(math.Abs(more-1)*actor.Output[costName], dec)*(utils.Ternary[float64](more >= 1, 1, -1)) + actor.Output[costName]
		actor.Output[costName] = max(0, utils.FloorTo(actor.Output[costName]+val.TotalCost, dec))
		if breakdown != nil && actor.Output[costName] != val.BaseCost {
			breakdown.AddLine(costName,
				fmt.Sprintf("%.2f"+utils.Ternary(val.Percent, "%%", "")+" ^8(base "+val.Text+" cost)", val.BaseCost),
			)
			if mult != 1 {
				breakdown.AddLine(costName, fmt.Sprintf("x %.2f ^8(cost multiplier)", mult))
			}
			if val.BaseCostNoMult != 0 {
				breakdown.AddLine(costName, fmt.Sprintf("+ %d ^8(additional "+val.Text+" cost)", val.BaseCostNoMult))
			}
			if inc != 0 {
				breakdown.AddLine(costName, fmt.Sprintf("x %.2f ^8(increased/reduced "+val.Text+" cost)", 1+inc/100))
			}
			if more != 1 {
				breakdown.AddLine(costName, fmt.Sprintf("x %.2f ^8(more/less "+val.Text+" cost)", more))
			}
			if val.TotalCost != 0 {
				breakdown.AddLine(costName, fmt.Sprintf("%+d ^8(total "+val.Text+" cost)", val.TotalCost))
			}
			breakdown.AddLine(costName, fmt.Sprintf("= %"+utils.Ternary(val.Upfront, "d", ".2f")+utils.Ternary(val.Percent, "%%", ""), actor.Output[costName]))
		}
	}
	/*
		TODO // account for Sacrificial Zeal
		// Note: Sacrificial Zeal grants Added Spell Physical Damage equal to 25% of the Skill's Mana Cost, and causes you to take Physical Damage over Time, for 4 seconds
		if skillModList.Flag(nil, "Condition:SacrificialZeal") {
			multiplier := 0.25
			skillModList.AddMod("PhysicalMin", "BASE", math.Floor(actor.Output["ManaCost"] * multiplier), "Sacrificial Zeal", ModFlag.Spell)
			skillModList.AddMod("PhysicalMax", "BASE", math.Floor(actor.Output["ManaCost"] * multiplier), "Sacrificial Zeal", ModFlag.Spell)
		}
	*/
	runSkillFunc("preDamageFunc")
	/*
		TODO // Handle corpse explosions
		if skillData.explodeCorpse and (skillData.corpseLife or env.enemyLevel) {
			localCorpseLife := skillData.corpseLife or data.monsterLifeTable[env.enemyLevel];
			damageType := skillData.corpseExplosionDamageType or "Fire"
			skillData[damageType+"BonusMin"] = localCorpseLife * ( skillData.corpseExplosionLifeMultiplier or skillData.selfFireExplosionLifeMultiplier )
			skillData[damageType+"BonusMax"] = localCorpseLife * ( skillData.corpseExplosionLifeMultiplier or skillData.selfFireExplosionLifeMultiplier )
		}
	*/

	// Cache global damage disabling flags
	canDeal := make(map[data.DamageType]bool)
	for _, damageType := range data.DamageType("").Values() {
		canDeal[damageType] = !skillModList.Flag(skillCfg, "DealNo"+string(damageType))
	}

	// Calculate damage conversion percentages
	activeSkill.ConversionTable = make(map[data.DamageType]ConversionTable)
	totalDamageTypes := len(data.DamageType("").Values())
	for damageTypeIndex := 0; damageTypeIndex < totalDamageTypes; damageTypeIndex++ {
		damageType := data.DamageType("").Values()[damageTypeIndex]
		globalConv := make(map[data.DamageType]float64)
		skillConv := make(map[data.DamageType]float64)
		add := make(map[data.DamageType]float64)
		globalTotal := float64(0)
		skillTotal := float64(0)

		for otherTypeIndex := damageTypeIndex + 1; otherTypeIndex < totalDamageTypes; otherTypeIndex++ {
			// For all possible destination types, check for global and skill conversions
			otherType := data.DamageType("").Values()[otherTypeIndex]

			globalNames := []string{string(damageType) + "DamageConvertTo" + string(otherType)}
			if damageType.IsElemental() {
				globalNames = append(globalNames, "ElementalDamageConvertTo"+string(otherType))
			}
			if damageType != data.DamageTypeChaos {
				globalNames = append(globalNames, "NonChaosDamageConvertTo"+string(otherType))
			}
			globalConv[otherType] = skillModList.Sum(mod.TypeBase, skillCfg, globalNames...)
			globalTotal += globalConv[otherType]

			skillConv[otherType] = skillModList.Sum(mod.TypeBase, skillCfg, "Skill"+string(damageType)+"DamageConvertTo"+string(otherType))
			skillTotal += skillConv[otherType]

			addNames := []string{string(damageType) + "DamageGainAs" + string(otherType)}
			if damageType.IsElemental() {
				addNames = append(addNames, "ElementalDamageGainAs"+string(otherType))
			}
			if damageType != data.DamageTypeChaos {
				addNames = append(addNames, "NonChaosDamageGainAs"+string(otherType))
			}
			add[otherType] = skillModList.Sum(mod.TypeBase, skillCfg, addNames...)
		}

		if skillTotal > 100 {
			// Skill conversion exceeds 100%, scale it down and remove non-skill conversions
			factor := 100 / skillTotal
			for convType, val := range skillConv {
				// Overconversion is fixed in 3.0, so I finally get to uncomment this line!
				skillConv[convType] = val * factor
			}
			for convType := range globalConv {
				globalConv[convType] = 0
			}
		} else if globalTotal+skillTotal > 100 {
			// Conversion exceeds 100%, scale down non-skill conversions
			factor := (100 - skillTotal) / globalTotal
			for convType, val := range globalConv {
				globalConv[convType] = val * factor
			}
			globalTotal = globalTotal * factor
		}

		dmgTable := ConversionTable{
			Targets: make(map[data.DamageType]float64),
		}
		for convType, val := range globalConv {
			dmgTable.Targets[convType] = (val + skillConv[convType] + add[convType]) / 100
		}

		dmgTable.Mult = 1 - math.Min((globalTotal+skillTotal)/100, 1)
		activeSkill.ConversionTable[damageType] = dmgTable
	}

	activeSkill.ConversionTable[data.DamageTypeChaos] = ConversionTable{
		Mult: 1,
	}

	// Configure damage passes
	passList := make([]*DamagePass, 0)
	if skillFlags[SkillFlagAttack] {
		outputTable[OutTableMainHand] = make(map[string]float64)
		outputTable[OutTableOffHand] = make(map[string]float64)
		critOverride := skillModList.Override(skillCfg, "WeaponBaseCritChance")
		if skillFlags[SkillFlagWeapon1Attack] {
			if breakdown != nil {
				// TODO Breakdown
				// breakdown.MainHand = LoadModule(calcs.breakdownModule, skillModList, actor.Output["MainHand"])
			}
			activeSkill.Weapon1Cfg.SkillStats = outputTable[OutTableMainHand]
			source := actor.WeaponData1 // TODO Copy
			if critOverride != nil && source != nil && source.Type != "None" {
				source.CritChance = critOverride.Float()
			}
			passList = append(passList, &DamagePass{
				Label:     "Main Hand",
				Source:    source,
				Config:    activeSkill.Weapon1Cfg,
				Output:    outputTable[OutTableMainHand],
				Breakdown: breakdown,
			})
		}

		if skillFlags[SkillFlagWeapon2Attack] {
			if breakdown != nil {
				// TODO Breakdown
				// breakdown.OffHand = LoadModule(calcs.breakdownModule, skillModList, actor.Output["OffHand"])
			}
			activeSkill.Weapon2Cfg.SkillStats = outputTable[OutTableOffHand]
			source := actor.WeaponData2 // TODO Copy
			if critOverride != nil && source != nil && source.Type != "None" {
				source.CritChance = critOverride.Float()
			}
			if skillData.CritChance > 0 {
				source.CritChance = skillData.CritChance
			}
			if skillData.SetOffHandPhysicalMin != 0 && skillData.SetOffHandPhysicalMax != 0 {
				source.PhysicalMin = skillData.SetOffHandPhysicalMin
				source.PhysicalMax = skillData.SetOffHandPhysicalMax
			}
			if skillData.AttackTime != 0 {
				source.AttackRate = 1000 / skillData.AttackTime
			}
			passList = append(passList, &DamagePass{
				Label:     "Off Hand",
				Source:    source,
				Config:    activeSkill.Weapon2Cfg,
				Output:    outputTable[OutTableOffHand],
				Breakdown: breakdown,
			})
		}
	} else {
		passList = append(passList, &DamagePass{
			Label:     "Skill",
			Source:    skillData,
			Config:    skillCfg,
			Output:    actor.Output,
			Breakdown: breakdown,
		})
	}

	type CombineMode string
	const (
		ModeOr            = CombineMode("OR")
		ModeAdd           = CombineMode("ADD")
		ModeAverage       = CombineMode("AVERAGE")
		ModeChance        = CombineMode("CHANCE")
		ModeChanceAilment = CombineMode("CHANCE_AILMENT")
		ModeDPS           = CombineMode("DPS")
	)

	// TODO Find what idk is for
	combineStat := func(stat string, mode CombineMode, idk ...string) {
		// Combine stats from Main Hand and Off Hand according to the mode
		if mode == ModeOr || utils.MissingOrFalse(skillFlags, SkillFlagBothWeaponAttack) {
			if utils.Has(outputTable[OutTableMainHand], stat) {
				actor.Output[stat] = outputTable[OutTableMainHand][stat]
			} else {
				actor.Output[stat] = outputTable[OutTableOffHand][stat]
			}
		} else if mode == ModeAdd {
			actor.Output[stat] = outputTable[OutTableMainHand][stat] + outputTable[OutTableOffHand][stat]
		} else if mode == ModeAverage {
			sum := outputTable[OutTableMainHand][stat] + outputTable[OutTableOffHand][stat]
			actor.Output[stat] = sum / 2
		} else if mode == ModeChance {
			if utils.Has(outputTable[OutTableMainHand], stat) && utils.Has(outputTable[OutTableOffHand], stat) {
				mainChance := outputTable[OutTableMainHand][idk[0]] * outputTable[OutTableMainHand]["HitChance"]
				offChance := outputTable[OutTableOffHand][idk[0]] * outputTable[OutTableOffHand]["HitChance"]
				mainPortion := mainChance / (mainChance + offChance)
				offPortion := offChance / (mainChance + offChance)
				actor.Output[stat] = outputTable[OutTableMainHand][stat]*mainPortion + outputTable[OutTableOffHand][stat]*offPortion
				if breakdown != nil {
					breakdown.AddLine(stat, "Contribution from Main Hand:")
					breakdown.AddLine(stat, fmt.Sprintf("%.1f", actor.OutputTable[OutTableMainHand][stat]))
					breakdown.AddLine(stat, fmt.Sprintf("x %.3f ^8(portion of instances created by main hand)", mainPortion))
					breakdown.AddLine(stat, fmt.Sprintf("= %.1f", actor.OutputTable[OutTableMainHand][stat]*mainPortion))
					breakdown.AddLine(stat, "Contribution from Off Hand:")
					breakdown.AddLine(stat, fmt.Sprintf("%.1f", actor.OutputTable[OutTableOffHand][stat]))
					breakdown.AddLine(stat, fmt.Sprintf("x %.3f ^8(portion of instances created by off hand)", offPortion))
					breakdown.AddLine(stat, fmt.Sprintf("= %.1f", actor.OutputTable[OutTableOffHand][stat]*offPortion))
					breakdown.AddLine(stat, "Total:")
					breakdown.AddLine(stat, fmt.Sprintf("%.1f + %.1f", actor.OutputTable[OutTableMainHand][stat]*mainPortion, actor.OutputTable[OutTableOffHand][stat]*offPortion))
					breakdown.AddLine(stat, fmt.Sprintf("= %.1f", actor.Output[stat]))
				}
			} else {
				if utils.Has(outputTable[OutTableMainHand], stat) {
					actor.Output[stat] = outputTable[OutTableMainHand][stat]
				} else {
					actor.Output[stat] = outputTable[OutTableOffHand][stat]
				}
			}
		} else if mode == ModeChanceAilment {
			if utils.Has(outputTable[OutTableMainHand], stat) && utils.Has(outputTable[OutTableOffHand], stat) {
				//mainChance := outputTable[OutTableMainHand][idk[0]] * outputTable[OutTableMainHand]["HitChance"]
				//offChance := outputTable[OutTableOffHand][idk[0]] * outputTable[OutTableOffHand]["HitChance"]
				//mainPortion := mainChance / (mainChance + offChance)
				//offPortion := offChance / (mainChance + offChance)
				maxInstance := max(outputTable[OutTableMainHand][stat], outputTable[OutTableOffHand][stat])
				minInstance := min(outputTable[OutTableMainHand][stat], outputTable[OutTableOffHand][stat])

				// TODO Somehow pass globalOutput ???
				//stackName := strings.ReplaceAll(stat, "DPS", "")+ "Stacks"
				//maxInstanceStacks := min(1, (globalOutput[stackName] or 1) / (globalOutput[stackName+"Max"] or 1))
				maxInstanceStacks := float64(0)

				actor.Output[stat] = maxInstance*maxInstanceStacks + minInstance*(1-maxInstanceStacks)
				/*
					TODO Chance Ailment
				*/
				if breakdown != nil {
					breakdown.AddLine(stat, "")
					breakdown.AddLine(stat, fmt.Sprintf("%.2f%% of ailment stacks use maximum damage", maxInstanceStacks*100))
					breakdown.AddLine(stat, fmt.Sprintf("Max Damage comes from %s", utils.Ternary(actor.OutputTable[OutTableMainHand][stat] >= actor.OutputTable[OutTableOffHand][stat], "Main Hand", "Off Hand")))
					breakdown.AddLine(stat, fmt.Sprintf("= %.1f", maxInstance*maxInstanceStacks))
					if maxInstanceStacks < 1 {
						breakdown.AddLine(stat, fmt.Sprintf("%.2f%% of ailment stacks use non-maximum damage", (1-maxInstanceStacks)*100))
						breakdown.AddLine(stat, fmt.Sprintf("= %.1f", minInstance*(1-maxInstanceStacks)))
					}
					breakdown.AddLine(stat, "")
					breakdown.AddLine(stat, "Total:")
					if maxInstanceStacks < 1 {
						breakdown.AddLine(stat, fmt.Sprintf("%.1f + %.1f", maxInstance*maxInstanceStacks, minInstance*(1-maxInstanceStacks)))
					}
					breakdown.AddLine(stat, fmt.Sprintf("= %.1f", actor.Output[stat]))
				}
			} else {
				if utils.Has(outputTable[OutTableMainHand], stat) {
					actor.Output[stat] = outputTable[OutTableMainHand][stat]
				} else {
					actor.Output[stat] = outputTable[OutTableOffHand][stat]
				}
				if breakdown != nil {
					breakdown.AddLine(stat, fmt.Sprintf("All ailment stacks comes from %s", utils.Ternary(actor.OutputTable[OutTableMainHand][stat] != 0, "Main Hand", "Off Hand")))
				}
			}
		} else if mode == ModeDPS {
			actor.Output[stat] = outputTable[OutTableMainHand][stat] + outputTable[OutTableOffHand][stat]
			if !skillData.DoubleHitsWhenDualWielding {
				actor.Output[stat] = actor.Output[stat] / 2
			}
		}
	}

	// TODO storedMainHandAccuracy
	var storedMainHandAccuracy *float64
	for _, pass := range passList {
		// Calculate hit chance
		pass.Output["Accuracy"] = math.Max(0, CalcVal(skillModList, "Accuracy", pass.Config))
		if breakdown != nil {
			breakdown.Simple(nil, pass.Config, actor.Output["Accuracy"], "Accuracy")
		}

		if skillModList.Flag(nil, "Condition:OffHandAccuracyIsMainHandAccuracy") && pass.Label == "Main Hand" {
			storedMainHandAccuracy = utils.Ptr(pass.Output["Accuracy"])
		} else if skillModList.Flag(nil, "Condition:OffHandAccuracyIsMainHandAccuracy") && pass.Label == "Off Hand" && storedMainHandAccuracy != nil {
			pass.Output["Accuracy"] = *storedMainHandAccuracy
			if breakdown != nil {
				breakdown.AddLine("Accuracy",
					"Using Main Hand Accuracy due to Mastery: "+fmt.Sprint(pass.Output["Accuracy"]),
				)
			}
		}

		if utils.MissingOrFalse(skillFlags, SkillFlagAttack) ||
			skillModList.Flag(pass.Config, "CannotBeEvaded") ||
			skillData.CannotBeEvaded ||
			(env.ModeEffective && enemyDB.Flag(nil, "CannotEvade")) {
			pass.Output["HitChance"] = 100
		} else {
			enemyEvasion := math.Max(math.Round(CalcVal(enemyDB, "Evasion", nil)), 0)
			pass.Output["HitChance"] = CalcHitChance(enemyEvasion, pass.Output["Accuracy"]) * CalcMod(skillModList, pass.Config, "HitChance")
			if breakdown != nil {
				breakdown.AddLine("HitChance",
					"Enemy level: "+fmt.Sprint(env.EnemyLevel)+utils.Ternary(env.Build.GetNumberOption("enemyLevel") != 0, " ^8(overridden from the Configuration tab", " ^8(can be overridden in the Configuration tab)"),
					"Average enemy evasion: "+fmt.Sprint(enemyEvasion),
					"Approximate hit chance: "+fmt.Sprint(pass.Output["HitChance"])+"%",
				)
			}
		}
		/*
			TODO // Check Precise Technique Keystone condition per pass as MH/OH might have different values
			condName := pass.label:gsub(" ", "") + "AccRatingHigherThanMaxLife"
			skillModList.conditions[condName] = actor.Output["Accuracy"] > env.player.Output["Life"]
		*/

		if (activeSkill.ActiveEffect.GrantedEffect.CastTime() == 0) && skillData.CastTimeOverride != 0 {
			pass.Output["Time"] = 0
			pass.Output["Speed"] = 0
		} else if skillData.TimeOverride > 0 {
			pass.Output["Time"] = skillData.TimeOverride
			pass.Output["Speed"] = 1 / pass.Output["Time"]
		} else if skillData.FixedCastTime {
			pass.Output["Time"] = activeSkill.ActiveEffect.GrantedEffect.CastTime()
			pass.Output["Speed"] = 1 / pass.Output["Time"]
		} else if skillData.TriggerTime > 0 && skillData.Triggered {
			activeSkillsLinked := skillModList.Sum(mod.TypeBase, pass.Config, "ActiveSkillsLinkedToTrigger")
			if activeSkillsLinked > 0 {
				pass.Output["Time"] = skillData.TriggerTime / (1 + skillModList.Sum(mod.TypeIncrease, pass.Config, "CooldownRecovery")/100) * activeSkillsLinked
			} else {
				pass.Output["Time"] = skillData.TriggerTime / (1 + skillModList.Sum(mod.TypeIncrease, pass.Config, "CooldownRecovery")/100)
			}
			pass.Output["TriggerTime"] = pass.Output["Time"]
			pass.Output["Speed"] = 1 / pass.Output["Time"]
		} else if skillData.TriggerRate > 0 && skillData.Triggered {
			/*
				TODO // Account for trigger unleash
				if skillData.triggerUnleash {
					// process the source trigger skill to get it's full data
					calcMode := env.mode == "CALCS" and "CALCS" or "MAIN"
					for _, triggerSkill := range (actor.activeSkillList) {
						if cacheSkillUUID(triggerSkill) == skillData.triggerSourceUUID {
							calcs.buildActiveSkill(env, calcMode, triggerSkill)
							break
						}
					}
					cachedSourceSkill := GlobalCache.cachedData[calcMode][skillData.triggerSourceUUID]
					// if properly processed, get it's dpsMultiplier to increase triggerRate
					if cachedSourceSkill {
						skillData.unleashTriggerRate = skillData.triggerRate * (cachedSourceSkill.ActiveSkill.skillData.dpsMultiplier or 1)
						if breakdown != nil {
							breakdown.Speed = {
								fmt.Sprintf("%.2f ^8(trigger rate)", skillData.triggerRate),
								fmt.Sprintf("* %.2f ^8(multiplier from Unleash)", cachedSourceSkill.ActiveSkill.skillData.dpsMultiplier or 1),
								fmt.Sprintf("= %.2f", skillData.unleashTriggerRate),
							}
						}
						// over-write the triggerRate modifier after breakdown as other calcs use it
						skillData.triggerRate = skillData.unleashTriggerRate
					}
					// give this activeSkill "HasSeals" flag so Configuration Option for UseMaxUnleash is available
					activeSkill.skillFlags.HasSeals = true
				}
			*/
			pass.Output["Time"] = 1 / skillData.TriggerRate
			pass.Output["TriggerTime"] = pass.Output["Time"]
			pass.Output["Speed"] = skillData.TriggerRate
			skillData.ShowAverage = false
		} else if skillData.TriggeredByBrand && skillData.Triggered {
			ArcanistSpellsLinked := skillModList.Sum(mod.TypeBase, pass.Config, "ArcanistSpellsLinked")
			if ArcanistSpellsLinked == 0 {
				ArcanistSpellsLinked = 1
			}
			pass.Output["Time"] = 1 / (1 + skillModList.Sum(mod.TypeIncrease, pass.Config, "Speed", "BrandActivationFrequency")/100) / skillModList.More(pass.Config, "BrandActivationFrequency") * ArcanistSpellsLinked
			pass.Output["TriggerTime"] = pass.Output["Time"]
			pass.Output["Speed"] = 1 / pass.Output["Time"]
		} else {
			baseTime := float64(0)
			if skillFlags[SkillFlagAttack] {
				if skillData.CastTimeOverride != 0 {
					// Skill is overriding weapon attack speed
					baseTime = activeSkill.ActiveEffect.GrantedEffect.CastTime() / (1 + (utils.OrDefault(pass.Source.AttackSpeedInc, 0))/100)
				} else if CalcMod(skillModList, skillCfg, "SkillAttackTime") > 0 {
					baseTime = (1/utils.OrDefault(pass.Source.AttackRate, 1) + skillModList.Sum(mod.TypeBase, pass.Config, "Speed")) * CalcMod(skillModList, skillCfg, "SkillAttackTime")
				} else {
					baseTime = 1/utils.OrDefault(pass.Source.AttackRate, 1) + skillModList.Sum(mod.TypeBase, pass.Config, "Speed")
				}
			} else {
				baseTime = 1
				if skillData.CastTimeOverride != 0 {
					baseTime = skillData.CastTimeOverride
				} else if activeSkill.ActiveEffect.GrantedEffect.CastTime() != 0 {
					baseTime = activeSkill.ActiveEffect.GrantedEffect.CastTime()
				}
			}

			inc := skillModList.Sum(mod.TypeIncrease, pass.Config, "Speed")
			more := skillModList.More(pass.Config, "Speed")

			pass.Output["Speed"] = 1 / baseTime * utils.RoundTo((1+inc/100)*more, 2)
			pass.Output["CastRate"] = pass.Output["Speed"]
			pass.Output["Repeats"] = 1 + skillModList.Sum(mod.TypeBase, pass.Config, "RepeatCount")

			if skillFlags[SkillFlagSelfCast] {
				// Self-cast skill; apply action speed
				pass.Output["Speed"] = pass.Output["Speed"] * actor.Output["ActionSpeedMod"]
				pass.Output["CastRate"] = pass.Output["Speed"]
			}

			if utils.Has(pass.Output, "Cooldown") {
				pass.Output["Speed"] = math.Min(pass.Output["Speed"], 1/pass.Output["Cooldown"]*pass.Output["Repeats"])
			}

			if utils.Has(pass.Output, "Cooldown") && skillFlags[SkillFlagSelfCast] {
				skillFlags[SkillFlagNotAverage] = true
				skillFlags[SkillFlagShowAverage] = false
				skillData.ShowAverage = false
			}

			if utils.MissingOrFalse(activeSkill.SkillTypes, data.SkillTypeChannel) {
				pass.Output["Speed"] = math.Min(pass.Output["Speed"], data.ServerTickRate*pass.Output["Repeats"])
			}

			if pass.Output["Speed"] == 0 {
				pass.Output["Time"] = 0
			} else {
				pass.Output["Time"] = 1 / pass.Output["Speed"]
			}

			if breakdown != nil {
				breakdown.MultiChain("Speed", BMultiChain{
					Base:  fmt.Sprintf("%.2f ^8(base)", 1/baseTime),
					Total: fmt.Sprintf("= %.2f ^8casts per second", actor.Output["CastRate"]),
					Items: []BMultiChainItem{
						{"%.2f ^8(increased/reduced)", 1 + inc/100},
						{"%.2f ^8(more/less)", more},
						{"%.2f ^8(action speed modifier)", utils.Ternary(skillFlags[SkillFlagSelfCast], actor.Output["ActionSpeedMod"], float64(1))},
					},
				})
				if actor.Output["Cooldown"] != 0 && (1/actor.Output["Cooldown"]) < actor.Output["CastRate"] {
					breakdown.AddLine("Speed", "\n")
					breakdown.AddLine("Speed", fmt.Sprintf("1 / %.2f ^8(skill cooldown)", actor.Output["Cooldown"]))
					if actor.Output["Repeats"] > 1 {
						breakdown.AddLine("Speed", fmt.Sprintf("x %.2f ^8(repeat count)", actor.Output["Repeats"]))
					}
					breakdown.AddLine("Speed", fmt.Sprintf("= %.2f ^8(casts per second)", actor.Output["Repeats"]/actor.Output["Cooldown"]))
					breakdown.AddLine("Speed", "\n")
					breakdown.AddLine("Speed", fmt.Sprintf("= %.2f ^8(lower of cast rates)", actor.Output["Speed"]))
				}
			}
			if breakdown != nil && calclib.Mod(skillModList, skillCfg, "SkillAttackTime") > 0 {
				breakdown.MultiChain("Time", BMultiChain{
					Base:  fmt.Sprintf("%.2f ^8(base)", 1/(actor.Output["Speed"]*calclib.Mod(skillModList, skillCfg, "SkillAttackTime"))),
					Total: fmt.Sprintf("= %.2f ^8seconds per attack", actor.Output["Time"]),
					Items: []BMultiChainItem{
						{"%.2f ^8(total modifier)", calclib.Mod(skillModList, skillCfg, "SkillAttackTime")},
					},
				})
			}
		}
		/*
			TODO Time override
			if skillData.hitTimeOverride and not skillData.triggeredOnDeath {
				actor.Output["HitTime"] = skillData.hitTimeOverride
				actor.Output["HitSpeed"] = 1 / actor.Output["HitTime"]
				//Brands always have hitTimeOverride
				if skillFlags.brand {
					actor.Output["BrandTicks"] = math.Floor(actor.Output["Duration"] * actor.Output["HitSpeed"])
				}
			} else if skillData.hitTimeMultiplier and actor.Output["Time"] and not skillData.triggeredOnDeath {
				actor.Output["HitTime"] = actor.Output["Time"] * skillData.hitTimeMultiplier
				actor.Output["HitSpeed"] = 1 / actor.Output["HitTime"]
			}
		*/
	}

	if utils.HasTrue(skillFlags, SkillFlagAttack) {
		// Combine hit chance and attack speed
		combineStat("HitChance", ModeAverage)
		combineStat("Speed", ModeAverage)
		combineStat("HitSpeed", ModeOr)

		if actor.Output["Speed"] == 0 {
			actor.Output["Time"] = 0
		} else {
			actor.Output["Time"] = 1 / actor.Output["Speed"]
		}

		if actor.Output["Time"] > 1 {
			modDB.AddMod(mod.NewFlag("Condition:OneSecondAttackTime", true))
		}

		if utils.HasTrue(skillFlags, SkillFlagBothWeaponAttack) {
			if breakdown != nil {
				breakdown.AddLine("Speed",
					"Both weapons:",
					fmt.Sprintf("(%.2f + %.2f) / 2", actor.OutputTable[OutTableMainHand]["Speed"], actor.OutputTable[OutTableOffHand]["Speed"]),
					fmt.Sprintf("= %.2f", actor.Output["Speed"]),
				)
			}
		}
	}

	quantityMultiplier := math.Max(skillModList.Sum(mod.TypeBase, skillCfg, "QuantityMultiplier"), 1)
	if quantityMultiplier > 1 {
		actor.Output["QuantityMultiplier"] = quantityMultiplier
	}

	for _, pass := range passList {
		/*
			TODO Passes
			globalOutput, globalBreakdown = output, breakdown
			source, output, cfg, breakdown := pass.source, pass.output, pass.cfg, pass.breakdown
		*/

		// TODO Use actor.Output instead of globalOutput

		// Exerted Attack members
		actor.Output["OffensiveWarcryEffect"] = 1
		actor.Output["MaxOffensiveWarcryEffect"] = 1
		actor.Output["TheoreticalOffensiveWarcryEffect"] = 1
		actor.Output["TheoreticalMaxOffensiveWarcryEffect"] = 1
		actor.Output["RallyingHitEffect"] = 1
		actor.Output["AilmentWarcryEffect"] = 1

		/*
			exertedDoubleDamage := env.modDB.Sum(mod.TypeBase, cfg, "ExertDoubleDamageChance")
			if env.mode_buffs {
				// Iterative over all the active skills to account for exerted attacks provided by warcries
				if (activeSkill.activeEffect.grantedEffect.name == "Vaal Ground Slam" or not activeSkill.skillTypes[SkillType.Vaal]) and not activeSkill.skillTypes[SkillType.Channel] and not activeSkill.skillModList.Flag(cfg, "SupportedByMultistrike") {
					for index, value := range (actor.activeSkillList) {
						if value.activeEffect.grantedEffect.name == "Ancestral Cry" and activeSkill.skillTypes[SkillType.MeleeSingleTarget] and not globalOutput.AncestralCryCalculated {
							globalOutput.AncestralCryDuration = calcSkillDuration(value.skillModList, value.skillCfg, value.skillData, env, enemyDB)
							globalOutput.AncestralCryCooldown = calcSkillCooldown(value.skillModList, value.skillCfg, value.skillData)
							actor.Output["GlobalWarcryCooldown"] = env.modDB.Sum(mod.TypeBase, nil, "GlobalWarcryCooldown")
							actor.Output["GlobalWarcryCount"] = env.modDB.Sum(mod.TypeBase, nil, "GlobalWarcryCount")
							if modDB.Flag(nil, "WarcryShareCooldown") {
								globalOutput.AncestralCryCooldown = globalOutput.AncestralCryCooldown + (actor.Output["GlobalWarcryCooldown"] - globalOutput.AncestralCryCooldown) / actor.Output["GlobalWarcryCount"]
							}
							globalOutput.AncestralCryCastTime = calcWarcryCastTime(value.skillModList, value.skillCfg, actor)
							globalOutput.AncestralExertsCount = env.modDB.Sum(mod.TypeBase, nil, "NumAncestralExerts") or 0
							baseUptimeRatio := min((globalOutput.AncestralExertsCount / actor.Output["Speed"]) / (globalOutput.AncestralCryCooldown + globalOutput.AncestralCryCastTime), 1) * 100
							additionalCooldownUses := value.skillModList.Sum(mod.TypeBase, value.skillCfg, "AdditionalCooldownUses")
							globalOutput.AncestralUpTimeRatio = min(100, baseUptimeRatio * (additionalCooldownUses + 1))
							if globalBreakdown {
								globalBreakdown.AncestralUpTimeRatio = { }
								t_insert(globalBreakdown.AncestralUpTimeRatio, fmt.Sprintf("(%d ^8(number of exerts)", globalOutput.AncestralExertsCount))
								t_insert(globalBreakdown.AncestralUpTimeRatio, fmt.Sprintf("/ %.2f) ^8(attacks per second)", actor.Output["Speed"]))
								if globalOutput.AncestralCryCastTime > 0 {
									t_insert(globalBreakdown.AncestralUpTimeRatio, fmt.Sprintf("/ (%.2f ^8(warcry cooldown)", globalOutput.AncestralCryCooldown))
									t_insert(globalBreakdown.AncestralUpTimeRatio, fmt.Sprintf("+ %.2f) ^8(warcry casttime)", globalOutput.AncestralCryCastTime))
								} else {
									t_insert(globalBreakdown.AncestralUpTimeRatio, fmt.Sprintf("/ %.2f ^8(average warcry cooldown)", globalOutput.AncestralCryCooldown))
								}
								t_insert(globalBreakdown.AncestralUpTimeRatio, fmt.Sprintf("= %d%%", globalOutput.AncestralUpTimeRatio))
							}
							globalOutput.AncestralCryCalculated = true
						} else if value.activeEffect.grantedEffect.name == "Infernal Cry" and not globalOutput.InfernalCryCalculated {
							globalOutput.InfernalCryDuration = calcSkillDuration(value.skillModList, value.skillCfg, value.skillData, env, enemyDB)
							globalOutput.InfernalCryCooldown = calcSkillCooldown(value.skillModList, value.skillCfg, value.skillData)
							actor.Output["GlobalWarcryCooldown"] = env.modDB.Sum(mod.TypeBase, nil, "GlobalWarcryCooldown")
							actor.Output["GlobalWarcryCount"] = env.modDB.Sum(mod.TypeBase, nil, "GlobalWarcryCount")
							if modDB.Flag(nil, "WarcryShareCooldown") {
								globalOutput.InfernalCryCooldown = globalOutput.InfernalCryCooldown + (actor.Output["GlobalWarcryCooldown"] - globalOutput.InfernalCryCooldown) / actor.Output["GlobalWarcryCount"]
							}
							globalOutput.InfernalCryCastTime = calcWarcryCastTime(value.skillModList, value.skillCfg, actor)
							if activeSkill.skillTypes[SkillType.Melee] {
								globalOutput.InfernalExertsCount = env.modDB.Sum(mod.TypeBase, nil, "NumInfernalExerts") or 0
								baseUptimeRatio := min((globalOutput.InfernalExertsCount / actor.Output["Speed"]) / (globalOutput.InfernalCryCooldown + globalOutput.InfernalCryCastTime), 1) * 100
								additionalCooldownUses := value.skillModList.Sum(mod.TypeBase, value.skillCfg, "AdditionalCooldownUses")
								globalOutput.InfernalUpTimeRatio = min(100, baseUptimeRatio * (additionalCooldownUses + 1))
								if globalBreakdown {
									globalBreakdown.InfernalUpTimeRatio = { }
									t_insert(globalBreakdown.InfernalUpTimeRatio, fmt.Sprintf("(%d ^8(number of exerts)", globalOutput.InfernalExertsCount))
									t_insert(globalBreakdown.InfernalUpTimeRatio, fmt.Sprintf("/ %.2f) ^8(attacks per second)", actor.Output["Speed"]))
									if globalOutput.InfernalCryCastTime > 0 {
										t_insert(globalBreakdown.InfernalUpTimeRatio, fmt.Sprintf("/ (%.2f ^8(warcry cooldown)", globalOutput.InfernalCryCooldown))
										t_insert(globalBreakdown.InfernalUpTimeRatio, fmt.Sprintf("+ %.2f) ^8(warcry casttime)", globalOutput.InfernalCryCastTime))
									} else {
										t_insert(globalBreakdown.InfernalUpTimeRatio, fmt.Sprintf("/ %.2f ^8(average warcry cooldown)", globalOutput.InfernalCryCooldown))
									}
									t_insert(globalBreakdown.InfernalUpTimeRatio, fmt.Sprintf("= %d%%", globalOutput.InfernalUpTimeRatio))
								}
							}
							globalOutput.InfernalCryCalculated = true
						} else if value.activeEffect.grantedEffect.name == "Intimidating Cry" and activeSkill.skillTypes[SkillType.Melee] and not globalOutput.IntimidatingCryCalculated {
							globalOutput.CreateWarcryOffensiveCalcSection = true
							globalOutput.IntimidatingCryDuration = calcSkillDuration(value.skillModList, value.skillCfg, value.skillData, env, enemyDB)
							globalOutput.IntimidatingCryCooldown = calcSkillCooldown(value.skillModList, value.skillCfg, value.skillData)
							actor.Output["GlobalWarcryCooldown"] = env.modDB.Sum(mod.TypeBase, nil, "GlobalWarcryCooldown")
							actor.Output["GlobalWarcryCount"] = env.modDB.Sum(mod.TypeBase, nil, "GlobalWarcryCount")
							if modDB.Flag(nil, "WarcryShareCooldown") {
								globalOutput.IntimidatingCryCooldown = globalOutput.IntimidatingCryCooldown + (actor.Output["GlobalWarcryCooldown"] - globalOutput.IntimidatingCryCooldown) / actor.Output["GlobalWarcryCount"]
							}
							globalOutput.IntimidatingCryCastTime = calcWarcryCastTime(value.skillModList, value.skillCfg, actor)
							globalOutput.IntimidatingExertsCount = env.modDB.Sum(mod.TypeBase, nil, "NumIntimidatingExerts") or 0
							baseUptime := min((globalOutput.IntimidatingExertsCount / actor.Output["Speed"]) / (globalOutput.IntimidatingCryCooldown + globalOutput.IntimidatingCryCastTime), 1) * 100
							additionalCooldownUses := value.skillModList.Sum(mod.TypeBase, value.skillCfg, "AdditionalCooldownUses")
							globalOutput.IntimidatingUpTimeRatio = min(100, baseUptime * (additionalCooldownUses + 1))
							if globalBreakdown {
								globalBreakdown.IntimidatingUpTimeRatio = { }
								t_insert(globalBreakdown.IntimidatingUpTimeRatio, fmt.Sprintf("(%d ^8(number of exerts)", globalOutput.IntimidatingExertsCount))
								t_insert(globalBreakdown.IntimidatingUpTimeRatio, fmt.Sprintf("/ %.2f) ^8(attacks per second)", actor.Output["Speed"]))
								if 	globalOutput.IntimidatingCryCastTime > 0 {
									t_insert(globalBreakdown.IntimidatingUpTimeRatio, fmt.Sprintf("/ (%.2f ^8(warcry cooldown)", globalOutput.IntimidatingCryCooldown))
									t_insert(globalBreakdown.IntimidatingUpTimeRatio, fmt.Sprintf("+ %.2f) ^8(warcry casttime)", globalOutput.IntimidatingCryCastTime))
								} else {
									t_insert(globalBreakdown.IntimidatingUpTimeRatio, fmt.Sprintf("/ %.2f ^8(average warcry cooldown)", globalOutput.IntimidatingCryCooldown))
								}
								t_insert(globalBreakdown.IntimidatingUpTimeRatio, fmt.Sprintf("= %d%%", globalOutput.IntimidatingUpTimeRatio))
							}
							ddChance := min(skillModList.Sum(mod.TypeBase, cfg, "DoubleDamageChance") + (env.mode_effective and enemyDB.Sum(mod.TypeBase, cfg, "SelfDoubleDamageChance") or 0) + exertedDoubleDamage, 100)
							globalOutput.IntimidatingAvgDmg = 2 * (1 - ddChance / 100) // 1
							if globalBreakdown {
								globalBreakdown.IntimidatingAvgDmg = {
									fmt.Sprintf("Average Intimidating Cry Damage:"),
									fmt.Sprintf("%.2f%% ^8(base double damage increase to hit 100%%)", (1 - ddChance / 100) * 100 ),
									fmt.Sprintf("x %d ^8(double damage multiplier)", 2),
									fmt.Sprintf("= %.2f", globalOutput.IntimidatingAvgDmg),
								}
							}
							globalOutput.IntimidatingHitEffect = 1 + globalOutput.IntimidatingAvgDmg * globalOutput.IntimidatingUpTimeRatio / 100
							globalOutput.IntimidatingMaxHitEffect = 1 + globalOutput.IntimidatingAvgDmg
							if globalBreakdown {
								globalBreakdown.IntimidatingHitEffect = {
									fmt.Sprintf("1 + (%.2f ^8(average exerted damage)", globalOutput.IntimidatingAvgDmg),
									fmt.Sprintf("x %.2f) ^8(uptime %%)", globalOutput.IntimidatingUpTimeRatio / 100),
									fmt.Sprintf("= %.2f", globalOutput.IntimidatingHitEffect),
								}
							}

							globalOutput.TheoreticalOffensiveWarcryEffect = globalOutput.TheoreticalOffensiveWarcryEffect * globalOutput.IntimidatingHitEffect
							globalOutput.TheoreticalMaxOffensiveWarcryEffect = globalOutput.TheoreticalMaxOffensiveWarcryEffect * globalOutput.IntimidatingMaxHitEffect
							globalOutput.IntimidatingCryCalculated = true
						} else if value.activeEffect.grantedEffect.name == "Rallying Cry" and activeSkill.skillTypes[SkillType.Melee] and not globalOutput.RallyingCryCalculated {
							globalOutput.CreateWarcryOffensiveCalcSection = true
							globalOutput.RallyingCryDuration = calcSkillDuration(value.skillModList, value.skillCfg, value.skillData, env, enemyDB)
							globalOutput.RallyingCryCooldown = calcSkillCooldown(value.skillModList, value.skillCfg, value.skillData)
							actor.Output["GlobalWarcryCooldown"] = env.modDB.Sum(mod.TypeBase, nil, "GlobalWarcryCooldown")
							actor.Output["GlobalWarcryCount"] = env.modDB.Sum(mod.TypeBase, nil, "GlobalWarcryCount")
							if modDB.Flag(nil, "WarcryShareCooldown") {
								globalOutput.RallyingCryCooldown = globalOutput.RallyingCryCooldown + (actor.Output["GlobalWarcryCooldown"] - globalOutput.RallyingCryCooldown) / actor.Output["GlobalWarcryCount"]
							}
							globalOutput.RallyingCryCastTime = calcWarcryCastTime(value.skillModList, value.skillCfg, actor)
							globalOutput.RallyingExertsCount = env.modDB.Sum(mod.TypeBase, nil, "NumRallyingExerts") or 0
							baseUptimeRatio := min((globalOutput.RallyingExertsCount / actor.Output["Speed"]) / (globalOutput.RallyingCryCooldown + globalOutput.RallyingCryCastTime), 1) * 100
							additionalCooldownUses := value.skillModList.Sum(mod.TypeBase, value.skillCfg, "AdditionalCooldownUses")
							globalOutput.RallyingUpTimeRatio = min(100, baseUptimeRatio * (additionalCooldownUses + 1))
							if globalBreakdown {
								globalBreakdown.RallyingUpTimeRatio = { }
								t_insert(globalBreakdown.RallyingUpTimeRatio, fmt.Sprintf("(%d ^8(number of exerts)", globalOutput.RallyingExertsCount))
								t_insert(globalBreakdown.RallyingUpTimeRatio, fmt.Sprintf("/ %.2f) ^8(attacks per second)", actor.Output["Speed"]))
								if 	globalOutput.RallyingCryCastTime > 0 {
									t_insert(globalBreakdown.RallyingUpTimeRatio, fmt.Sprintf("/ (%.2f ^8(warcry cooldown)", globalOutput.RallyingCryCooldown))
									t_insert(globalBreakdown.RallyingUpTimeRatio, fmt.Sprintf("+ %.2f) ^8(warcry casttime)", globalOutput.RallyingCryCastTime))
								} else {
									t_insert(globalBreakdown.RallyingUpTimeRatio, fmt.Sprintf("/ %.2f ^8(average warcry cooldown)", globalOutput.RallyingCryCooldown))
								}
								t_insert(globalBreakdown.RallyingUpTimeRatio, fmt.Sprintf("= %d%%", globalOutput.RallyingUpTimeRatio))
							}
							globalOutput.RallyingAvgDmg = min(env.modDB.Sum(mod.TypeBase, cfg, "Multiplier:NearbyAlly"), 5) * (env.modDB.Sum(mod.TypeBase, nil, "RallyingExertMoreDamagePerAlly") / 100)
							if globalBreakdown {
								globalBreakdown.RallyingAvgDmg = {
									fmt.Sprintf("Average Rallying Cry Damage:"),
									fmt.Sprintf("%.2f ^8(average damage multiplier per ally)", env.modDB.Sum(mod.TypeBase, nil, "RallyingExertMoreDamagePerAlly") / 100),
									fmt.Sprintf("x %d ^8(number of nearby allies (max=5))", min(env.modDB.Sum(mod.TypeBase, cfg, "Multiplier:NearbyAlly"), 5)),
									fmt.Sprintf("= %.2f", globalOutput.RallyingAvgDmg),
								}
							}
							globalOutput.RallyingHitEffect = 1 + globalOutput.RallyingAvgDmg * globalOutput.RallyingUpTimeRatio / 100
							globalOutput.RallyingMaxHitEffect = 1 + globalOutput.RallyingAvgDmg
							if globalBreakdown {
								globalBreakdown.RallyingHitEffect = {
									fmt.Sprintf("1 + (%.2f ^8(average exerted damage)", globalOutput.RallyingAvgDmg),
									fmt.Sprintf("x %.2f) ^8(uptime %%)", globalOutput.RallyingUpTimeRatio / 100),
									fmt.Sprintf("= %.2f", globalOutput.RallyingHitEffect),
								}
							}
							globalOutput.OffensiveWarcryEffect = globalOutput.OffensiveWarcryEffect * globalOutput.RallyingHitEffect
							globalOutput.MaxOffensiveWarcryEffect = globalOutput.MaxOffensiveWarcryEffect * globalOutput.RallyingMaxHitEffect
							globalOutput.TheoreticalOffensiveWarcryEffect = globalOutput.TheoreticalOffensiveWarcryEffect * globalOutput.RallyingHitEffect
							globalOutput.TheoreticalMaxOffensiveWarcryEffect = globalOutput.TheoreticalMaxOffensiveWarcryEffect * globalOutput.RallyingMaxHitEffect
							globalOutput.RallyingCryCalculated = true

						} else if value.activeEffect.grantedEffect.name == "Seismic Cry" and activeSkill.skillTypes[SkillType.Slam] and not globalOutput.SeismicCryCalculated {
							globalOutput.CreateWarcryOffensiveCalcSection = true
							globalOutput.SeismicCryDuration = calcSkillDuration(value.skillModList, value.skillCfg, value.skillData, env, enemyDB)
							globalOutput.SeismicCryCooldown = calcSkillCooldown(value.skillModList, value.skillCfg, value.skillData)
							actor.Output["GlobalWarcryCooldown"] = env.modDB.Sum(mod.TypeBase, nil, "GlobalWarcryCooldown")
							actor.Output["GlobalWarcryCount"] = env.modDB.Sum(mod.TypeBase, nil, "GlobalWarcryCount")
							if modDB.Flag(nil, "WarcryShareCooldown") {
								globalOutput.SeismicCryCooldown = globalOutput.SeismicCryCooldown + (actor.Output["GlobalWarcryCooldown"] - globalOutput.SeismicCryCooldown) / actor.Output["GlobalWarcryCount"]
							}
							globalOutput.SeismicCryCastTime = calcWarcryCastTime(value.skillModList, value.skillCfg, actor)
							globalOutput.SeismicExertsCount = env.modDB.Sum(mod.TypeBase, nil, "NumSeismicExerts") or 0
							baseUptimeRatio := min((globalOutput.SeismicExertsCount / actor.Output["Speed"]) / (globalOutput.SeismicCryCooldown + globalOutput.SeismicCryCastTime), 1) * 100
							additionalCooldownUses := value.skillModList.Sum(mod.TypeBase, value.skillCfg, "AdditionalCooldownUses")
							globalOutput.SeismicUpTimeRatio = min(100, baseUptimeRatio * (additionalCooldownUses + 1))
							if globalBreakdown {
								globalBreakdown.SeismicUpTimeRatio = { }
								t_insert(globalBreakdown.SeismicUpTimeRatio, fmt.Sprintf("(%d ^8(number of exerts)", globalOutput.SeismicExertsCount))
								t_insert(globalBreakdown.SeismicUpTimeRatio, fmt.Sprintf("/ %.2f) ^8(attacks per second)", actor.Output["Speed"]))
								if 	globalOutput.SeismicCryCastTime > 0 {
									t_insert(globalBreakdown.SeismicUpTimeRatio, fmt.Sprintf("/ (%.2f ^8(warcry cooldown)", globalOutput.SeismicCryCooldown))
									t_insert(globalBreakdown.SeismicUpTimeRatio, fmt.Sprintf("+ %.2f) ^8(warcry casttime)", globalOutput.SeismicCryCastTime))
								} else {
									t_insert(globalBreakdown.SeismicUpTimeRatio, fmt.Sprintf("/ %.2f ^8(average warcry cooldown)", globalOutput.SeismicCryCooldown))
								}
								t_insert(globalBreakdown.SeismicUpTimeRatio, fmt.Sprintf("= %d%%", globalOutput.SeismicUpTimeRatio))
							}
							// calculate the stacking AoE modifier of Seismic slams
							SeismicAoEPerExert := env.modDB.Sum(mod.TypeBase, cfg, "SeismicIncAoEPerExert") / 100
							AoEImpact := 0
							MaxSingleAoEImpact := 0
							for i = 1, globalOutput.SeismicExertsCount {
								AoEImpact = AoEImpact + (i * SeismicAoEPerExert)
								MaxSingleAoEImpact = MaxSingleAoEImpact + SeismicAoEPerExert
							}
							AvgAoEImpact := AoEImpact / globalOutput.SeismicExertsCount

							// account for AoE increase
							if activeSkill.skillModList.Flag(nil, "Condition:WarcryMaxHit") {
								skillModList.AddMod("AreaOfEffect", "INC", MaxSingleAoEImpact * 100, "Max Seismic Exert AoE")
							} else {
								skillModList.AddMod("AreaOfEffect", "INC", math.Floor(AvgAoEImpact * globalOutput.SeismicUpTimeRatio), "Avg Seismic Exert AoE")
							}
							calcAreaOfEffect(skillModList, skillCfg, skillData, skillFlags, globalOutput, globalBreakdown)
							globalOutput.SeismicCryCalculated = true
						}
					}

					if activeSkill.skillModList.Flag(nil, "Condition:WarcryMaxHit") {
						globalOutput.AilmentWarcryEffect = globalOutput.MaxOffensiveWarcryEffect
						skillData.showAverage = true
						skillFlags.showAverage = true
						skillFlags.notAverage = false
					} else {
						globalOutput.AilmentWarcryEffect = globalOutput.OffensiveWarcryEffect
					}

					// Calculate Exerted Attack Uptime
					// There are various strategies a player could use to maximize either warcry effect stacking or staggering
					// 1) they don't pay attention and therefore we calculated exerted attack uptime as just the maximum uptime of any enabled warcries that exert attacks
					globalOutput.ExertedAttackUptimeRatio = max(max(max(globalOutput.AncestralUpTimeRatio or 0, globalOutput.InfernalUpTimeRatio or 0), max(globalOutput.IntimidatingUpTimeRatio or 0, globalOutput.RallyingUpTimeRatio or 0)), globalOutput.SeismicUpTimeRatio or 0)
					if globalBreakdown {
						globalBreakdown.ExertedAttackUptimeRatio = { }
						t_insert(globalBreakdown.ExertedAttackUptimeRatio, fmt.Sprintf("Maximum of:"))
						if globalOutput.AncestralUpTimeRatio {
							t_insert(globalBreakdown.ExertedAttackUptimeRatio, fmt.Sprintf("%d%% ^8(Ancestral Cry Uptime)", globalOutput.AncestralUpTimeRatio or 0))
						}
						if globalOutput.InfernalUpTimeRatio {
							t_insert(globalBreakdown.ExertedAttackUptimeRatio, fmt.Sprintf("%d%% ^8(Infernal Cry Uptime)", globalOutput.InfernalUpTimeRatio or 0))
						}
						if globalOutput.IntimidatingUpTimeRatio {
							t_insert(globalBreakdown.ExertedAttackUptimeRatio, fmt.Sprintf("%d%% ^8(Intimidating Cry Uptime)", globalOutput.IntimidatingUpTimeRatio or 0))
						}
						if globalOutput.RallyingUpTimeRatio {
							t_insert(globalBreakdown.ExertedAttackUptimeRatio, fmt.Sprintf("%d%% ^8(Rallying Cry Uptime)", globalOutput.RallyingUpTimeRatio or 0))
						}
						if globalOutput.SeismicUpTimeRatio {
							t_insert(globalBreakdown.ExertedAttackUptimeRatio, fmt.Sprintf("%d%% ^8(Seismic Cry Uptime)", globalOutput.SeismicUpTimeRatio or 0))
						}
						t_insert(globalBreakdown.ExertedAttackUptimeRatio, fmt.Sprintf("= %d%%", globalOutput.ExertedAttackUptimeRatio))
					}
					if globalOutput.ExertedAttackUptimeRatio > 0 {
						incExertedAttacks := skillModList.Sum(mod.TypeIncrease, cfg, "ExertIncrease")
						moreExertedAttacks := skillModList.Sum("MORE", cfg, "ExertIncrease")
						moreExertedAttackDamage := skillModList.Sum("MORE", cfg, "ExertAttackIncrease")
						if activeSkill.skillModList.Flag(nil, "Condition:WarcryMaxHit") {
							skillModList.AddMod("Damage", "INC", incExertedAttacks, "Exerted Attacks")
							skillModList.AddMod("Damage", "MORE", moreExertedAttacks, "Exerted Attacks")
							skillModList.AddMod("Damage", "MORE", moreExertedAttackDamage, "Exerted Attack Damage", ModFlag.Attack)
						} else {
							skillModList.AddMod("Damage", "INC", incExertedAttacks * globalOutput.ExertedAttackUptimeRatio / 100, "Uptime Scaled Exerted Attacks")
							skillModList.AddMod("Damage", "MORE", moreExertedAttacks * globalOutput.ExertedAttackUptimeRatio / 100, "Uptime Scaled Exerted Attacks")
							skillModList.AddMod("Damage", "MORE", moreExertedAttackDamage * globalOutput.ExertedAttackUptimeRatio / 100, "Uptime Scaled Exerted Attack Damage", ModFlag.Attack)
						}
						globalOutput.ExertedAttackAvgDmg = calclib.Mod(skillModList, skillCfg, "ExertIncrease")
						globalOutput.ExertedAttackAvgDmg = globalOutput.ExertedAttackAvgDmg * calclib.Mod(skillModList, skillCfg, "ExertAttackIncrease")
						globalOutput.ExertedAttackHitEffect = globalOutput.ExertedAttackAvgDmg * globalOutput.ExertedAttackUptimeRatio / 100
						globalOutput.ExertedAttackMaxHitEffect = globalOutput.ExertedAttackAvgDmg
						if globalBreakdown {
							globalBreakdown.ExertedAttackHitEffect = {
								fmt.Sprintf("(%.2f ^8(average exerted damage)", globalOutput.ExertedAttackAvgDmg),
								fmt.Sprintf("x %.2f) ^8(uptime %%)", globalOutput.ExertedAttackUptimeRatio / 100),
								fmt.Sprintf("= %.2f", globalOutput.ExertedAttackHitEffect),
							}
						}
					}
				}
			}
		*/

		pass.Output["RuthlessBlowHitEffect"] = 1
		pass.Output["RuthlessBlowBleedEffect"] = 1
		pass.Output["FistOfWarHitEffect"] = 1
		pass.Output["FistOfWarAilmentEffect"] = 1

		if env.ModeCombat {
			// Calculate Ruthless Blow chance/multipliers + Fist of War multipliers
			actor.Output["RuthlessBlowMaxCount"] = skillModList.Sum(mod.TypeBase, pass.Config, "RuthlessBlowMaxCount")
			if actor.Output["RuthlessBlowMaxCount"] > 0 {
				actor.Output["RuthlessBlowChance"] = utils.RoundTo(100/actor.Output["RuthlessBlowMaxCount"], 0)
			} else {
				actor.Output["RuthlessBlowChance"] = 0
			}
			actor.Output["RuthlessBlowHitMultiplier"] = 1 + skillModList.Sum(mod.TypeBase, pass.Config, "RuthlessBlowHitMultiplier")/100
			actor.Output["RuthlessBlowBleedMultiplier"] = 1 + skillModList.Sum(mod.TypeBase, pass.Config, "RuthlessBlowBleedMultiplier")/100
			actor.Output["RuthlessBlowHitEffect"] = 1 - actor.Output["RuthlessBlowChance"]/100 + actor.Output["RuthlessBlowChance"]/100*actor.Output["RuthlessBlowHitMultiplier"]
			actor.Output["RuthlessBlowBleedEffect"] = 1 - actor.Output["RuthlessBlowChance"]/100 + actor.Output["RuthlessBlowChance"]/100*actor.Output["RuthlessBlowBleedMultiplier"]

			actor.Output["FistOfWarCooldown"] = skillModList.Sum(mod.TypeBase, pass.Config, "FistOfWarCooldown")
			// If Fist of War & Active Skill is a Slam Skill & NOT a Vaal Skill
			if actor.Output["FistOfWarCooldown"] != 0 && activeSkill.SkillTypes[data.SkillTypeSlam] && !activeSkill.SkillTypes[data.SkillTypeVaal] {
				actor.Output["FistOfWarHitMultiplier"] = skillModList.Sum(mod.TypeBase, pass.Config, "FistOfWarHitMultiplier") / 100
				actor.Output["FistOfWarAilmentMultiplier"] = skillModList.Sum(mod.TypeBase, pass.Config, "FistOfWarAilmentMultiplier") / 100
				actor.Output["FistOfWarUptimeRatio"] = min((1/actor.Output["Speed"])/actor.Output["FistOfWarCooldown"], 1) * 100
				if actor.Breakdown != nil {
					actor.Breakdown.AddLine("FistOfWarUptimeRatio",
						fmt.Sprintf("min( (1 / %.2f) ^8(second per attack)", actor.Output["Speed"]),
						fmt.Sprintf("/ %.2f, 1) ^8(fist of war cooldown)", actor.Output["FistOfWarCooldown"]),
						fmt.Sprintf("= %.2f%%", actor.Output["FistOfWarUptimeRatio"]),
					)
				}
				actor.Output["AvgFistOfWarHit"] = actor.Output["FistOfWarHitMultiplier"]
				actor.Output["AvgFistOfWarHitEffect"] = 1 + actor.Output["FistOfWarHitMultiplier"]*(actor.Output["FistOfWarUptimeRatio"]/100)
				if actor.Breakdown != nil {
					actor.Breakdown.AddLine("AvgFistOfWarHitEffect",
						fmt.Sprintf("1 + (%.2f ^8(fist of war hit multiplier)", actor.Output["FistOfWarHitMultiplier"]),
						fmt.Sprintf("x %.2f) ^8(fist of war uptime ratio)", actor.Output["FistOfWarUptimeRatio"]/100),
						fmt.Sprintf("= %.2f", actor.Output["AvgFistOfWarHitEffect"]),
					)
				}
				actor.Output["AvgFistOfWarAilmentEffect"] = 1 + actor.Output["FistOfWarAilmentMultiplier"]*(actor.Output["FistOfWarUptimeRatio"]/100)
				actor.Output["MaxFistOfWarHitEffect"] = 1 + actor.Output["FistOfWarHitMultiplier"]
				actor.Output["MaxFistOfWarAilmentEffect"] = 1 + actor.Output["FistOfWarAilmentMultiplier"]
				if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
					actor.Output["FistOfWarHitEffect"] = actor.Output["MaxFistOfWarHitEffect"]
					actor.Output["FistOfWarAilmentEffect"] = actor.Output["MaxFistOfWarAilmentEffect"]
				} else {
					actor.Output["FistOfWarHitEffect"] = actor.Output["AvgFistOfWarHitEffect"]
					actor.Output["FistOfWarAilmentEffect"] = actor.Output["AvgFistOfWarAilmentEffect"]
				}
				actor.Output["TheoreticalOffensiveWarcryEffect"] = actor.Output["TheoreticalOffensiveWarcryEffect"] * actor.Output["AvgFistOfWarHitEffect"]
				actor.Output["TheoreticalMaxOffensiveWarcryEffect"] = actor.Output["TheoreticalMaxOffensiveWarcryEffect"] * actor.Output["MaxFistOfWarHitEffect"]
			} else {
				actor.Output["FistOfWarHitEffect"] = 1
				actor.Output["FistOfWarAilmentEffect"] = 1
			}
		}

		// Calculate crit chance, crit multiplier, and their combined effect
		if skillModList.Flag(nil, "NeverCrit") {
			pass.Output["PreEffectiveCritChance"] = 0
			pass.Output["CritChance"] = 0
			pass.Output["CritMultiplier"] = 0
			pass.Output["BonusCritDotMultiplier"] = 0
			pass.Output["CritEffect"] = 1
		} else {
			baseCrit := float64(0)

			if pass.Source.CritChance != 0 {
				baseCrit = pass.Source.CritChance
			}

			critOverride := skillModList.Override(pass.Config, "CritChance")
			if critOverride != nil {
				baseCrit = critOverride.Float()
			}

			if baseCrit == 100 {
				pass.Output["PreEffectiveCritChance"] = 100
				pass.Output["CritChance"] = 100
			} else {
				base := float64(0)
				inc := float64(0)
				more := float64(0)
				if critOverride == nil {
					base = skillModList.Sum(mod.TypeBase, pass.Config, "CritChance")
					inc = skillModList.Sum(mod.TypeIncrease, pass.Config, "CritChance")
					more = skillModList.More(pass.Config, "CritChance")

					if env.ModeEffective {
						base += enemyDB.Sum(mod.TypeBase, nil, "SelfCritChance")
						inc += enemyDB.Sum(mod.TypeIncrease, nil, "SelfCritChance")
					}
				}

				pass.Output["CritChance"] = (baseCrit + base) * (1 + inc/100) * more

				preCapCritChance := pass.Output["CritChance"]
				pass.Output["CritChance"] = math.Min(pass.Output["CritChance"], 100)

				if baseCrit+base > 0 {
					pass.Output["CritChance"] = math.Max(pass.Output["CritChance"], 0)
				}

				pass.Output["PreEffectiveCritChance"] = pass.Output["CritChance"]
				preLuckyCritChance := pass.Output["CritChance"]

				if env.ModeEffective && skillModList.Flag(pass.Config, "CritChanceLucky") {
					pass.Output["CritChance"] = (1 - math.Pow(1-pass.Output["CritChance"]/100, 2)) * 100
				}

				preHitCheckCritChance := pass.Output["CritChance"]
				if env.ModeEffective {
					pass.Output["CritChance"] = pass.Output["CritChance"] * pass.Output["HitChance"] / 100
				}

				if breakdown != nil && actor.Output["CritChance"] != baseCrit {
					if base != 0 {
						breakdown.AddLine("CritChance", fmt.Sprintf("(%g + %g) ^8(base)", baseCrit, base))
					} else {
						breakdown.AddLine("CritChance", fmt.Sprintf("%g ^8(base)", baseCrit+base))
					}
					if inc != 0 {
						breakdown.AddLine("CritChance", fmt.Sprintf("x %.2f", 1+inc/100)+" ^8(increased/reduced)")
					}
					if more != 1 {
						breakdown.AddLine("CritChance", fmt.Sprintf("x %.2f", more)+" ^8(more/less)")
					}
					breakdown.AddLine("CritChance", fmt.Sprintf("= %.2f%% ^8(crit chance)", actor.Output["PreEffectiveCritChance"]))
					if preCapCritChance > 100 {
						overCap := preCapCritChance - 100
						breakdown.AddLine("CritChance", fmt.Sprintf("Crit is overcapped by %.2f%% (%.2f%% increased Critical Strike Chance)", overCap, overCap/more/(baseCrit+base)*100))
					}
					if env.ModeEffective && skillModList.Flag(pass.Config, "CritChanceLucky") {
						breakdown.AddLine("CritChance", "Crit Chance is Lucky:")
						breakdown.AddLine("CritChance", fmt.Sprintf("1 - (1 - %.4f) x (1 - %.4f)", preLuckyCritChance/100, preLuckyCritChance/100))
						breakdown.AddLine("CritChance", fmt.Sprintf("= %.2f%%", preHitCheckCritChance))
					}
					if env.ModeEffective && actor.Output["HitChance"] < 100 {
						breakdown.AddLine("CritChance", "Crit confirmation roll:")
						breakdown.AddLine("CritChance", fmt.Sprintf("%.2f%%", preHitCheckCritChance))
						breakdown.AddLine("CritChance", fmt.Sprintf("x %.2f ^8(chance to hit)", actor.Output["HitChance"]/100))
						breakdown.AddLine("CritChance", fmt.Sprintf("= %.2f%%", actor.Output["CritChance"]))
					}
				}
			}

			if skillModList.Flag(pass.Config, "NoCritMultiplier") {
				pass.Output["CritMultiplier"] = 1
			} else {
				extraDamage := skillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplier") / 100
				multiOverride := skillModList.Override(skillCfg, "CritMultiplier")
				if multiOverride != nil {
					extraDamage = (multiOverride.Float() - 100) / 100
				}

				if env.ModeEffective {
					enemyInc := 1 + enemyDB.Sum(mod.TypeIncrease, nil, "SelfCritMultiplier")/100
					extraDamage = extraDamage + enemyDB.Sum(mod.TypeBase, nil, "SelfCritMultiplier")/100
					extraDamage = utils.RoundTo(extraDamage*enemyInc, 2)
					if breakdown != nil && enemyInc != 1 {
						breakdown.AddLine("CritMultiplier",
							fmt.Sprintf("%.2f%% ^8(additional extra damage)", (enemyDB.Sum(mod.TypeBase, nil, "SelfCritMultiplier")+skillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplier"))/100),
							fmt.Sprintf("x %.2f ^8(increased/reduced extra crit damage taken by enemy)", enemyInc),
							fmt.Sprintf("= %.2f%% ^8(extra crit damage)", extraDamage*100),
						)
					}
				}

				pass.Output["CritMultiplier"] = 1 + math.Max(0, extraDamage)
			}

			critChancePercentage := pass.Output["CritChance"] / 100
			pass.Output["CritEffect"] = 1 - critChancePercentage + critChancePercentage*pass.Output["CritMultiplier"]
			pass.Output["CritEffect"] = (skillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplier") - 50) * skillModList.Sum(mod.TypeBase, pass.Config, "CritMultiplierAppliesToDegen") / 1000

			if breakdown != nil && actor.Output["CritEffect"] != 1 {
				breakdown.AddLine("CritEffect",
					fmt.Sprintf("(1 - %.4f) ^8(portion of damage from non-crits)", critChancePercentage),
					fmt.Sprintf("+ [ (%.4f x %g) ^8(portion of damage from crits)", critChancePercentage, actor.Output["CritMultiplier"]),
					fmt.Sprintf("= %.3f", actor.Output["CritEffect"]),
				)
			}
		}

		pass.Output["ScaledDamageEffect"] = 1

		// Calculate chance and multiplier for dealing triple damage on Normal and Crit
		actor.Output["TripleDamageChanceOnCrit"] = min(skillModList.Sum(mod.TypeBase, pass.Config, "TripleDamageChanceOnCrit"), 100)
		actor.Output["TripleDamageChance"] = min(skillModList.Sum(mod.TypeBase, pass.Config, "TripleDamageChance")+(utils.Ternary(env.ModeEffective, enemyDB.Sum(mod.TypeBase, pass.Config, "SelfTripleDamageChance"), 0))+(actor.Output["TripleDamageChanceOnCrit"]*actor.Output["CritChance"]/100), 100)
		actor.Output["TripleDamageEffect"] = 1 + (2 * actor.Output["TripleDamageChance"] / 100)
		actor.Output["ScaledDamageEffect"] = actor.Output["ScaledDamageEffect"] * actor.Output["TripleDamageEffect"]

		// Calculate chance and multiplier for dealing double damage on Normal and Crit
		actor.Output["DoubleDamageChanceOnCrit"] = min(skillModList.Sum(mod.TypeBase, pass.Config, "DoubleDamageChanceOnCrit"), 100)
		actor.Output["DoubleDamageChance"] = min(skillModList.Sum(mod.TypeBase, pass.Config, "DoubleDamageChance")+(utils.Ternary(env.ModeEffective, enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDoubleDamageChance"), 0))+(actor.Output["DoubleDamageChanceOnCrit"]*actor.Output["CritChance"]/100), 100)
		if actor.Output["IntimidatingUpTimeRatio"] != 0 && activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
			actor.Output["DoubleDamageChance"] = 100
		} else if actor.Output["IntimidatingUpTimeRatio"] != 0 {
			actor.Output["DoubleDamageChance"] = min(actor.Output["DoubleDamageChance"]+actor.Output["IntimidatingUpTimeRatio"], 100)
		}

		// Triple Damage overrides Double Damage. If you have both, it's the same as just having Triple
		// We need to subtract the probability of both happening in favor of Triple Damage
		if actor.Output["TripleDamageChance"] > 0 {
			actor.Output["DoubleDamageChance"] = max(actor.Output["DoubleDamageChance"]-actor.Output["TripleDamageChance"]*actor.Output["DoubleDamageChance"]/100, 0)
		}
		actor.Output["DoubleDamageEffect"] = 1 + actor.Output["DoubleDamageChance"]/100
		actor.Output["ScaledDamageEffect"] = actor.Output["ScaledDamageEffect"] * actor.Output["DoubleDamageEffect"]

		// Calculate culling DPS
		criticalCull := skillModList.Max(pass.Config, "CriticalCullPercent")
		if criticalCull > 0 {
			criticalCull = criticalCull * (actor.Output["CritChance"] / 100)
		}
		regularCull := skillModList.Max(pass.Config, "CullPercent")
		maxCullPercent := max(criticalCull, regularCull)
		actor.Output["CullPercent"] = maxCullPercent
		actor.Output["CullMultiplier"] = 100 / (100 - actor.Output["CullPercent"])

		// Calculate base hit damage
		for _, damageType := range data.DamageType("").Values() {
			damageTypeMin := string(damageType) + "Min"
			damageTypeMax := string(damageType) + "Max"

			baseMultiplier := float64(1)
			if activeSkill.ActiveEffect.GrantedEffectLevel.BaseMultiplier != nil {
				baseMultiplier = *activeSkill.ActiveEffect.GrantedEffectLevel.BaseMultiplier
			} else if skillData.BaseMultiplier != 0 {
				baseMultiplier = skillData.BaseMultiplier
			}

			damageEffectiveness := float64(1)
			if activeSkill.ActiveEffect.GrantedEffectLevel.DamageEffectiveness != nil {
				damageEffectiveness = *activeSkill.ActiveEffect.GrantedEffectLevel.DamageEffectiveness
			} else if skillData.DamageEffectiveness != 0 {
				damageEffectiveness = skillData.DamageEffectiveness
			}

			addedMin := skillModList.Sum(mod.TypeBase, pass.Config, damageTypeMin) + enemyDB.Sum(mod.TypeBase, pass.Config, "Self"+damageTypeMin)
			addedMax := skillModList.Sum(mod.TypeBase, pass.Config, damageTypeMax) + enemyDB.Sum(mod.TypeBase, pass.Config, "Self"+damageTypeMax)
			addedMult := CalcMod(skillModList, pass.Config, "Added"+string(damageType)+"Damage", "AddedDamage")

			baseMin := (utils.GetOr(pass.Source, damageTypeMin, utils.Interface(float64(0))).(float64)+
				utils.GetOr(pass.Source, string(damageType)+"BonusMin", utils.Interface(float64(0))).(float64))*
				baseMultiplier + addedMin*damageEffectiveness*addedMult

			baseMax := (utils.GetOr(pass.Source, damageTypeMax, utils.Interface(float64(0))).(float64)+
				utils.GetOr(pass.Source, string(damageType)+"BonusMax", utils.Interface(float64(0))).(float64))*
				baseMultiplier + addedMax*damageEffectiveness*addedMult

			pass.Output[damageTypeMin+"Base"] = baseMin
			pass.Output[damageTypeMax+"Base"] = baseMax

			if breakdown != nil {
				// TODO Breakdown
				// breakdown[damageType] = { damageTypes = { } }
				if baseMin != 0 && baseMax != 0 {
					breakdown.AddLine(string(damageType), "Base damage:")
					plus := ""
					sourceMin := utils.GetOr(pass.Source, damageTypeMin, float64(0))
					sourceMax := utils.GetOr(pass.Source, damageTypeMax, float64(0))
					if sourceMin != 0 || sourceMax != 0 {
						breakdown.AddLine(string(damageType), fmt.Sprintf("%.0f to %.0f ^8(base damage from %s)", sourceMin, sourceMax, utils.Ternary(pass.Source.Type != "", "weapon", "skill")))
						if baseMultiplier != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(base damage multiplier)", baseMultiplier))
						}
						plus = "+ "
					}
					if addedMin != 0 || addedMax != 0 {
						breakdown.AddLine(string(damageType), fmt.Sprintf("%s%.0f to %.0f ^8(added damage)", plus, addedMin, addedMax))
						if damageEffectiveness != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(damage effectiveness)", damageEffectiveness))
						}
						if addedMult != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(added damage multiplier)", addedMult))
						}
					}
					breakdown.AddLine(string(damageType), fmt.Sprintf("= %.1f to %.1f", baseMin, baseMax))
				}
			}
		}

		totalHitMin := float64(0)
		totalHitMax := float64(0)
		totalHitAvg := float64(0)

		totalCritMin := float64(0)
		totalCritMax := float64(0)
		totalCritAvg := float64(0)

		ghostReaver := skillModList.Flag(nil, "GhostReaver")

		pass.Output["LifeLeech"] = 0
		pass.Output["LifeLeechInstant"] = 0
		pass.Output["EnergyShieldLeech"] = 0
		pass.Output["EnergyShieldLeechInstant"] = 0
		pass.Output["ManaLeech"] = 0
		pass.Output["ManaLeechInstant"] = 0
		pass.Output["ImpaleStoredHitAvg"] = 0

		// Calculate hit damage for each damage type
		// TODO(post) Going from 1-2 for legacy Lua reasons. Should probably be changed
		for p := 1; p <= 2; p++ {
			// Pass 1 is critical strike damage, pass 2 is non-critical strike
			pass.Config.SkillCond["CriticalStrike"] = p == 1
			lifeLeechTotal := float64(0)
			energyShieldLeechTotal := float64(0)
			manaLeechTotal := float64(0)

			noLifeLeech := skillModList.Flag(pass.Config, "CannotLeechLife") || enemyDB.Flag(nil, "CannotLeechLifeFromSelf")
			noEnergyShieldLeech := skillModList.Flag(pass.Config, "CannotLeechEnergyShield") || enemyDB.Flag(nil, "CannotLeechEnergyShieldFromSelf")
			noManaLeech := skillModList.Flag(pass.Config, "CannotLeechMana") || enemyDB.Flag(nil, "CannotLeechManaFromSelf")

			for _, damageType := range data.DamageType("").Values() {
				damageTypeHitMin := float64(0)
				damageTypeHitMax := float64(0)
				damageTypeHitAvg := float64(0)
				damageTypeLuckyChance := float64(0)
				damageTypeHitAvgLucky := float64(0)
				damageTypeHitAvgNotLucky := float64(0)

				if utils.HasTrue(skillFlags, SkillFlagHit) && utils.HasTrue(canDeal, damageType) {
					damageTypeHitMin, damageTypeHitMax = calcDamage(activeSkill, pass.Output, pass.Config, breakdown, string(damageType), damageType, 0, nil)
					convMult := activeSkill.ConversionTable[damageType].Mult

					if p == 2 && breakdown != nil {
						breakdown.AddLine(string(damageType), "Hit damage:")
						breakdown.AddLine(string(damageType), fmt.Sprintf("%.0f to %.0f ^8(total damage)", damageTypeHitMin, damageTypeHitMax))
						if convMult != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %g ^8(%g%% converted to other damage types)", convMult, (1-convMult)*100))
						}
						if actor.Output["TripleDamageEffect"] != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(multiplier from %.2f%% chance to deal triple damage)", actor.Output["TripleDamageEffect"], actor.Output["TripleDamageChance"]))
						}
						if actor.Output["DoubleDamageEffect"] != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(multiplier from %.2f%% chance to deal double damage)", actor.Output["DoubleDamageEffect"], actor.Output["DoubleDamageChance"]))
						}
						if actor.Output["RuthlessBlowHitEffect"] != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(ruthless blow effect modifier)", actor.Output["RuthlessBlowHitEffect"]))
						}
						if actor.Output["FistOfWarHitEffect"] != 1 {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(fist of war effect modifier)", actor.Output["FistOfWarHitEffect"]))
						}
						if actor.Output["OffensiveWarcryEffect"] != 1 && !activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(aggregated warcry exerted effect modifier)", actor.Output["OffensiveWarcryEffect"]))
						}
						if actor.Output["MaxOffensiveWarcryEffect"] != 1 && activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.2f ^8(aggregated max warcry exerted effect modifier)", actor.Output["MaxOffensiveWarcryEffect"]))
						}
					}

					if skillModList.Flag(nil, "Condition:WarcryMaxHit") {
						pass.Output["AllMult"] = convMult * pass.Output["ScaledDamageEffect"] * pass.Output["RuthlessBlowHitEffect"] * pass.Output["FistOfWarHitEffect"] * actor.Output["MaxOffensiveWarcryEffect"]
					} else {
						pass.Output["AllMult"] = convMult * pass.Output["ScaledDamageEffect"] * pass.Output["RuthlessBlowHitEffect"] * pass.Output["FistOfWarHitEffect"] * actor.Output["OffensiveWarcryEffect"]
					}

					allMult := pass.Output["AllMult"]
					if p == 1 {
						// Apply crit multiplier
						allMult *= pass.Output["CritMultiplier"]
					}

					damageTypeHitMin *= allMult
					damageTypeHitMax *= allMult

					if skillModList.Flag(skillCfg, "LuckyHits") ||
						(p == 2 && damageType == data.DamageTypeLightning && skillModList.Flag(skillCfg, "LightningNoCritLucky")) ||
						(p == 1 && skillModList.Flag(skillCfg, "CritLucky")) ||
						((damageType == data.DamageTypeLightning || damageType == data.DamageTypeCold || damageType == data.DamageTypeFire) && skillModList.Flag(skillCfg, "ElementalLuckHits")) {
						damageTypeLuckyChance = 1
					} else {
						damageTypeLuckyChance = math.Min(skillModList.Sum(mod.TypeBase, skillCfg, "LuckyHitsChance"), 100) / 100
					}

					damageTypeHitAvgNotLucky = damageTypeHitMin/2 + damageTypeHitMax/2
					damageTypeHitAvgLucky = damageTypeHitMin/3 + 2*damageTypeHitMax/3
					damageTypeHitAvg = damageTypeHitAvgNotLucky*(1-damageTypeLuckyChance) + damageTypeHitAvgLucky*damageTypeLuckyChance

					if (damageTypeHitMin != 0 || damageTypeHitMax != 0) && env.ModeEffective {
						// Apply enemy resistances and damage taken modifiers
						resist := float64(0)
						pen := float64(0)
						sourceRes := data.DamageType("")
						takenInc := enemyDB.Sum(mod.TypeIncrease, pass.Config, "DamageTaken", string(damageType)+"DamageTaken")
						takenMore := enemyDB.More(pass.Config, "DamageTaken", string(damageType)+"DamageTaken")

						// Check if player is supposed to ignore a damage type, or if it's ignored on enemy side
						useThisResist := func(damageType data.DamageType) bool {
							names := []string{"Ignore" + string(damageType) + "Resistance"}
							if damageType.IsElemental() {
								names = append(names, "IgnoreElementalResistances")
							}
							return !skillModList.Flag(pass.Config, names...) && !enemyDB.Flag(nil, "SelfIgnore"+string(damageType)+"Resistance")
						}

						if damageType == data.DamageTypePhysical {
							// store pre-armour physical damage from attacks for impale calculations
							if p == 1 {
								pass.Output["ImpaleStoredHitAvg"] = pass.Output["ImpaleStoredHitAvg"] + damageTypeHitAvg*(pass.Output["CritChance"]/100)
							} else {
								pass.Output["ImpaleStoredHitAvg"] = pass.Output["ImpaleStoredHitAvg"] + damageTypeHitAvg*(1-pass.Output["CritChance"]/100)
							}
							enemyArmour := math.Max(CalcVal(enemyDB, "Armour", nil), 0)
							armourReduction := CalcArmourReductionF(enemyArmour, damageTypeHitAvg)
							if skillModList.Flag(pass.Config, "IgnoreEnemyPhysicalDamageReduction") {
								resist = 0
							} else {
								resist = math.Min(math.Max(0, enemyDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction")+skillModList.Sum(mod.TypeBase, pass.Config, "EnemyPhysicalDamageReduction")+armourReduction), data.DamageReductionCap)
							}
						} else {
							if (skillModList.Flag(pass.Config, "ChaosDamageUsesLowestResistance") && damageType == data.DamageTypeChaos) ||
								(skillModList.Flag(pass.Config, "ElementalDamageUsesLowestResistance") && damageType.IsElemental()) {
								// Default to using the current damage type
								elementUsed := damageType
								if damageType.IsElemental() {
									resist = math.Min(enemyDB.Sum(mod.TypeBase, nil, string(damageType)+"Resist", "ElementalResist")*CalcMod(enemyDB, nil, string(damageType)+"Resist", "ElementalResist"), data.EnemyMaxResist)
									takenInc += enemyDB.Sum(mod.TypeIncrease, pass.Config, "ElementalDamageTaken")
								} else if damageType == data.DamageTypeChaos {
									resist = math.Min(enemyDB.Sum(mod.TypeBase, nil, "ChaosResist")*CalcMod(enemyDB, nil, "ChaosResist"), data.EnemyMaxResist)
								}

								// Find the lowest resist of all the elements and use that if it's lower
								for _, eleDamageType := range data.DamageType("").Values() {
									if eleDamageType.IsElemental() && useThisResist(eleDamageType) && damageType != eleDamageType {
										currentElementResist := math.Min(enemyDB.Sum(mod.TypeBase, nil, string(eleDamageType)+"Resist", "ElementalResist")*CalcMod(enemyDB, nil, string(eleDamageType)+"Resist", "ElementalResist"), data.EnemyMaxResist)
										// If it's explicitly lower, then use the resist and update which element we're using to account for penetration
										if resist > currentElementResist {
											resist = currentElementResist
											elementUsed = eleDamageType
										}
									}
								}

								// Update the penetration based on the element used
								if elementUsed.IsElemental() {
									pen = skillModList.Sum(mod.TypeBase, pass.Config, string(elementUsed)+"Penetration", "ElementalPenetration")
								} else if elementUsed == data.DamageTypeChaos {
									pen = skillModList.Sum(mod.TypeBase, pass.Config, "ChaosPenetration")
								}
								sourceRes = elementUsed
							} else if damageType.IsElemental() {
								resist = enemyDB.Sum(mod.TypeBase, nil, string(damageType)+"Resist")
								if env.ModDB.Flag(nil, "Enemy"+string(damageType)+"ResistEqualToYours") {
									resist = env.Player.Output[string(damageType)+"Resist"]
								} else {
									base := resist + enemyDB.Sum(mod.TypeBase, nil, "ElementalResist")
									resist = base * CalcMod(enemyDB, nil, string(damageType)+"Resist")
								}
								pen = skillModList.Sum(mod.TypeBase, pass.Config, string(damageType)+"Penetration", "ElementalPenetration")
								takenInc += enemyDB.Sum(mod.TypeIncrease, pass.Config, "ElementalDamageTaken")
							} else if damageType == data.DamageTypeChaos {
								resist = enemyDB.Sum(mod.TypeBase, nil, string(damageType)+"Resist")
								pen = skillModList.Sum(mod.TypeBase, pass.Config, "ChaosPenetration")
							}

							resist = math.Max(math.Min(resist, data.EnemyMaxResist), data.ResistFloor)
						}

						if utils.HasTrue(skillFlags, SkillFlagProjectile) {
							takenInc += enemyDB.Sum(mod.TypeIncrease, nil, "ProjectileDamageTaken")
						}

						if utils.HasTrue(skillFlags, SkillFlagProjectile) && utils.HasTrue(skillFlags, SkillFlagAttack) {
							takenInc += enemyDB.Sum(mod.TypeIncrease, nil, "ProjectileAttackDamageTaken")
						}

						if utils.HasTrue(skillFlags, SkillFlagTrap) || utils.HasTrue(skillFlags, SkillFlagMine) {
							takenInc += enemyDB.Sum(mod.TypeIncrease, nil, "TrapMineDamageTaken")
						}

						effMult := (1 + takenInc/100) * takenMore
						useRes := useThisResist(damageType)
						if damageType.IsElemental() && skillModList.Flag(pass.Config, "CannotElePenIgnore") {
							effMult *= 1 - resist/100
						} else {
							effMult *= 1 - (resist-pen)/100
						}

						damageTypeHitMin = damageTypeHitMin * effMult
						damageTypeHitMax = damageTypeHitMax * effMult
						damageTypeHitAvg = damageTypeHitAvg * effMult

						if env.Mode == OutputModeCalcs {
							pass.Output[string(damageType)+"EffMult"] = effMult
						}
						if p == 2 && breakdown != nil && (effMult != 1 || sourceRes != "") && skillModList.Flag(pass.Config, utils.Ternary(isElemental[string(damageType)], "CannotElePenIgnore", "")) {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.3f ^8(effective DPS modifier)", effMult))
							breakdown.EffMult(string(damageType+"EffMult"), damageType, resist, 0, takenInc, effMult, takenMore, sourceRes, useRes)
						} else if p == 2 && breakdown != nil && (effMult != 1 || sourceRes != "") {
							breakdown.AddLine(string(damageType), fmt.Sprintf("x %.3f ^8(effective DPS modifier)", effMult))
							breakdown.EffMult(string(damageType+"EffMult"), damageType, resist, pen, takenInc, effMult, takenMore, sourceRes, useRes)
						}
					}
					if p == 2 && breakdown != nil {
						breakdown.AddLine(string(damageType), fmt.Sprintf("= %.2f to %.2f", damageTypeHitMin, damageTypeHitMax))
					}

					// Beginning of Leech Calculation for this DamageType
					if utils.HasTrue(skillFlags, SkillFlagMine) || utils.HasTrue(skillFlags, SkillFlagTrap) || utils.HasTrue(skillFlags, SkillFlagTotem) {
						if !noLifeLeech {
							lifeLeech := skillModList.Sum(mod.TypeBase, pass.Config, "DamageLifeLeechToPlayer")
							if lifeLeech > 0 {
								lifeLeechTotal += damageTypeHitAvg * lifeLeech / 100
							}
						}
					} else {
						if !noLifeLeech {
							lifeLeech := float64(0)
							if skillModList.Flag(nil, "LifeLeechBasedOnChaosDamage") {
								if damageType == data.DamageTypeChaos {
									lifeLeech = skillModList.Sum(mod.TypeBase, pass.Config, "DamageLeech", "DamageLifeLeech", "PhysicalDamageLifeLeech", "LightningDamageLifeLeech", "ColdDamageLifeLeech", "FireDamageLifeLeech", "ChaosDamageLifeLeech", "ElementalDamageLifeLeech") + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDamageLifeLeech")/100
								} else {
									lifeLeech = 0
								}
							} else {
								names := []string{"DamageLeech", "DamageLifeLeech", string(damageType) + "DamageLifeLeech"}
								if damageType.IsElemental() {
									names = append(names, "ElementalDamageLifeLeech")
								}
								lifeLeech = skillModList.Sum(mod.TypeBase, pass.Config, names...) + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDamageLifeLeech")/100
							}
							if lifeLeech > 0 {
								lifeLeechTotal = lifeLeechTotal + damageTypeHitAvg*lifeLeech/100
							}
						}

						if !noEnergyShieldLeech {
							names := []string{"DamageEnergyShieldLeech", string(damageType) + "DamageEnergyShieldLeech"}
							if damageType.IsElemental() {
								names = append(names, "ElementalDamageEnergyShieldLeech")
							}
							energyShieldLeech := skillModList.Sum(mod.TypeBase, pass.Config, names...) + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDamageEnergyShieldLeech")/100
							if energyShieldLeech > 0 {
								energyShieldLeechTotal = energyShieldLeechTotal + damageTypeHitAvg*energyShieldLeech/100
							}
						}

						if !noManaLeech {
							names := []string{"DamageManaLeech", string(damageType) + "DamageManaLeech"}
							if damageType.IsElemental() {
								names = append(names, "ElementalDamageManaLeech")
							}
							manaLeech := skillModList.Sum(mod.TypeBase, pass.Config, names...) + enemyDB.Sum(mod.TypeBase, pass.Config, "SelfDamageManaLeech")/100
							if manaLeech > 0 {
								manaLeechTotal = manaLeechTotal + damageTypeHitAvg*manaLeech/100
							}
						}
					}
				} else {
					if breakdown != nil {
						breakdown.AddLine(string(damageType), string("You can't deal "+damageType+" damage"))
					}
				}

				if p == 1 {
					pass.Output[string(damageType)+"CritAverage"] = damageTypeHitAvg
					totalCritAvg = totalCritAvg + damageTypeHitAvg
					totalCritMin = totalCritMin + damageTypeHitMin
					totalCritMax = totalCritMax + damageTypeHitMax
				} else {
					pass.Output[string(damageType)+"Min"] = damageTypeHitMin
					pass.Output[string(damageType)+"Max"] = damageTypeHitMax
					pass.Output[string(damageType)+"HitAverage"] = damageTypeHitAvg
					totalHitAvg = totalHitAvg + damageTypeHitAvg
					totalHitMin = totalHitMin + damageTypeHitMin
					totalHitMax = totalHitMax + damageTypeHitMax
				}
			}

			if skillData.LifeLeechPerUse != 0 {
				lifeLeechTotal += skillData.LifeLeechPerUse
			}

			if skillData.ManaLeechPerUse != 0 {
				manaLeechTotal += skillData.ManaLeechPerUse
			}

			portion := 1 - pass.Output["CritChance"]/100
			if p == 1 {
				portion = pass.Output["CritChance"] / 100
			}

			if skillModList.Flag(pass.Config, "InstantLifeLeech") && !ghostReaver {
				pass.Output["LifeLeechInstant"] += lifeLeechTotal * portion
			} else {
				pass.Output["LifeLeech"] += lifeLeechTotal * portion
			}

			if skillModList.Flag(pass.Config, "InstantEnergyShieldLeech") {
				pass.Output["EnergyShieldLeechInstant"] += energyShieldLeechTotal * portion
			} else {
				pass.Output["EnergyShieldLeech"] += energyShieldLeechTotal * portion
			}

			if skillModList.Flag(pass.Config, "InstantManaLeech") {
				pass.Output["ManaLeechInstant"] += manaLeechTotal * portion
			} else {
				pass.Output["ManaLeech"] += manaLeechTotal * portion
			}
		}

		pass.Output["TotalMin"] = totalHitMin
		pass.Output["TotalMax"] = totalHitMax

		/*
			TODO ElementalEquilibrium
			if skillModList.Flag(skillCfg, "ElementalEquilibrium") and not env.configInput.EEIgnoreHitDamage and (actor.Output["FireHitAverage"] + actor.Output["ColdHitAverage"] + actor.Output["LightningHitAverage"] > 0) {
				// Update enemy hit-by-damage-type conditions
				enemyDB.conditions.HitByFireDamage = actor.Output["FireHitAverage"] > 0
				enemyDB.conditions.HitByColdDamage = actor.Output["ColdHitAverage"] > 0
				enemyDB.conditions.HitByLightningDamage = actor.Output["LightningHitAverage"] > 0
			}
		*/
		highestType := "Physical"
		// For each damage type, calculate percentage of total damage. Also tracks the highest damage type and outputs a Condition:TypeIsHighestDamageType flag for whichever the highest type is
		for _, damageType := range dmgTypes {
			if actor.Output[damageType+"HitAverage"] > 0 {
				portion := actor.Output[damageType+"HitAverage"] / totalHitAvg * 100
				highestPortion := actor.Output[highestType+"HitAverage"] / totalHitAvg * 100
				if portion > highestPortion {
					highestType = damageType
				}
				if breakdown != nil {
					breakdown.AddLine(damageType, fmt.Sprintf("Portion of total damage: %.2f%%", portion))
				}
			}
		}
		skillModList.AddMod(mod.NewFlag("Condition:"+highestType+"IsHighestDamageType", true).Source("Config"))

		/*
			hitRate := actor.Output["HitChance"] / 100 * (actor.Output["HitSpeed"] or actor.Output["Speed"]) * (skillData.dpsMultiplier or 1)
			TODO // Calculate leech
			local function getLeechInstances(amount, total)
				if total == 0 {
					return 0, 0
				}
				duration := amount / total / data.LeechRateBase
				return duration, duration * hitRate
			}
			if ghostReaver {
				actor.Output["EnergyShieldLeech"] = actor.Output["EnergyShieldLeech"] + actor.Output["LifeLeech"]
				actor.Output["EnergyShieldLeechInstant"] = actor.Output["EnergyShieldLeechInstant"] + actor.Output["LifeLeechInstant"]
				actor.Output["LifeLeech"] = 0
				actor.Output["LifeLeechInstant"] = 0
			}
			actor.Output["LifeLeech"] = min(actor.Output["LifeLeech"], actor.Output["MaxLifeLeechInstance"])
			actor.Output["LifeLeechDuration"], actor.Output["LifeLeechInstances"] = getLeechInstances(actor.Output["LifeLeech"], actor.Output["Life"])
			actor.Output["LifeLeechInstantRate"] = actor.Output["LifeLeechInstant"] * hitRate
			actor.Output["EnergyShieldLeech"] = min(actor.Output["EnergyShieldLeech"], actor.Output["MaxEnergyShieldLeechInstance"])
			actor.Output["EnergyShieldLeechDuration"], actor.Output["EnergyShieldLeechInstances"] = getLeechInstances(actor.Output["EnergyShieldLeech"], actor.Output["EnergyShield"])
			actor.Output["EnergyShieldLeechInstantRate"] = actor.Output["EnergyShieldLeechInstant"] * hitRate
			actor.Output["ManaLeech"] = min(actor.Output["ManaLeech"], actor.Output["MaxManaLeechInstance"])
			actor.Output["ManaLeechDuration"], actor.Output["ManaLeechInstances"] = getLeechInstances(actor.Output["ManaLeech"], actor.Output["Mana"])
			actor.Output["ManaLeechInstantRate"] = actor.Output["ManaLeechInstant"] * hitRate
		*/
		/*
			TODO // Calculate gain on hit
			if skillFlags.mine or skillFlags.trap or skillFlags.totem {
				actor.Output["LifeOnHit"] = 0
				actor.Output["EnergyShieldOnHit"] = 0
				actor.Output["ManaOnHit"] = 0
			} else {
				actor.Output["LifeOnHit"] = skillModList.Sum(mod.TypeBase, cfg, "LifeOnHit") + enemyDB.Sum(mod.TypeBase, cfg, "SelfLifeOnHit")
				actor.Output["EnergyShieldOnHit"] = skillModList.Sum(mod.TypeBase, cfg, "EnergyShieldOnHit") + enemyDB.Sum(mod.TypeBase, cfg, "SelfEnergyShieldOnHit")
				actor.Output["ManaOnHit"] = skillModList.Sum(mod.TypeBase, cfg, "ManaOnHit") + enemyDB.Sum(mod.TypeBase, cfg, "SelfManaOnHit")
			}
			actor.Output["LifeOnHitRate"] = actor.Output["LifeOnHit"] * hitRate
			actor.Output["EnergyShieldOnHitRate"] = actor.Output["EnergyShieldOnHit"] * hitRate
			actor.Output["ManaOnHitRate"] = actor.Output["ManaOnHit"] * hitRate
		*/

		// Calculate average damage and final DPS
		pass.Output["AverageHit"] = totalHitAvg*(1-pass.Output["CritChance"]/100) + totalCritAvg*pass.Output["CritChance"]/100
		pass.Output["AverageDamage"] = pass.Output["AverageHit"] * pass.Output["HitChance"] / 100

		selectedSpeed, gotSpeed := actor.Output["HitSpeed"]
		if !gotSpeed || selectedSpeed == 0 {
			selectedSpeed = actor.Output["Speed"]
		}

		DpsMultiplier := utils.OrDefault(skillData.DpsMultiplier, 1)
		pass.Output["TotalDPS"] = pass.Output["AverageDamage"] * selectedSpeed * DpsMultiplier * quantityMultiplier
		if breakdown != nil {
			if actor.Output["CritEffect"] != 1 {
				if skillModList.Flag(skillCfg, "LuckyHits") {
					breakdown.AddLine("AverageHit", fmt.Sprintf("(1/3) x %.2f + (2/3) x %.2f = %.1f ^8(average from non-crits)", totalHitMin, totalHitMax, totalHitAvg))
				}
				if skillModList.Flag(skillCfg, "CritLucky") || skillModList.Flag(skillCfg, "LuckyHits") {
					breakdown.AddLine("AverageHit", fmt.Sprintf("(1/3) x %.2f + (2/3) x %.2f = %.1f ^8(average from crits)", totalCritMin, totalCritMax, totalCritAvg))
					breakdown.AddLine("AverageHit", "")
				}
				breakdown.AddLine("AverageHit", fmt.Sprintf("%.1f x (1 - %.4f) ^8(damage from non-crits)", totalHitAvg, actor.Output["CritChance"]/100))
				breakdown.AddLine("AverageHit", fmt.Sprintf("+ %.1f x %.4f ^8(damage from crits)", totalCritAvg, actor.Output["CritChance"]/100))
				breakdown.AddLine("AverageHit", fmt.Sprintf("= %.1f", actor.Output["AverageHit"]))
			}
			if isAttack {
				breakdown.AddLine("AverageDamage", fmt.Sprintf("%s:", pass.Label))
				breakdown.AddLine("AverageDamage", fmt.Sprintf("%.1f ^8(average hit)", actor.Output["AverageHit"]))
				breakdown.AddLine("AverageDamage", fmt.Sprintf("x %.2f ^8(chance to hit)", actor.Output["HitChance"]/100))
				breakdown.AddLine("AverageDamage", fmt.Sprintf("= %.1f", actor.Output["AverageDamage"]))
			}
		}
	}

	if isAttack {
		// Combine crit stats, average damage and DPS
		combineStat("PreEffectiveCritChance", "AVERAGE")
		combineStat("CritChance", "AVERAGE")
		combineStat("CritMultiplier", "AVERAGE")
		combineStat("AverageDamage", "DPS")
		combineStat("TotalDPS", "DPS")
		combineStat("LifeLeechDuration", "DPS")
		combineStat("LifeLeechInstances", "DPS")
		combineStat("LifeLeechInstant", "DPS")
		combineStat("LifeLeechInstantRate", "DPS")
		combineStat("EnergyShieldLeechDuration", "DPS")
		combineStat("EnergyShieldLeechInstances", "DPS")
		combineStat("EnergyShieldLeechInstant", "DPS")
		combineStat("EnergyShieldLeechInstantRate", "DPS")
		combineStat("ManaLeechDuration", "DPS")
		combineStat("ManaLeechInstances", "DPS")
		combineStat("ManaLeechInstant", "DPS")
		combineStat("ManaLeechInstantRate", "DPS")
		combineStat("LifeOnHit", "DPS")
		combineStat("LifeOnHitRate", "DPS")
		combineStat("EnergyShieldOnHit", "DPS")
		combineStat("EnergyShieldOnHitRate", "DPS")
		combineStat("ManaOnHit", "DPS")
		combineStat("ManaOnHitRate", "DPS")

		/*
			TODO bothWeaponAttack
			if skillFlags.bothWeaponAttack {
				if breakdown != nil {
					breakdown.AverageDamage = { }
					breakdown.AddLine("AverageDamage", "Both weapons:")
					if skillData.doubleHitsWhenDualWielding {
						breakdown.AddLine("AverageDamage", fmt.Sprintf("%.1f + %.1f ^8(skill hits with both weapons at once)", actor.Output["MainHand"].AverageDamage, actor.Output["OffHand"].AverageDamage))
					} else {
						breakdown.AddLine("AverageDamage", fmt.Sprintf("(%.1f + %.1f) / 2 ^8(skill alternates weapons)", actor.Output["MainHand"].AverageDamage, actor.Output["OffHand"].AverageDamage))
					}
					breakdown.AddLine("AverageDamage", fmt.Sprintf("= %.1f", actor.Output["AverageDamage"]))
				}
			}
		*/
	}

	/*
		TODO CALCS
		if env.mode == "CALCS" {
			if skillData.ShowAverage {
				actor.Output["DisplayDamage"] = formatNumSep(fmt.Sprintf("%.1f", actor.Output["AverageDamage"])) + " average damage"
			} else {
				actor.Output["DisplayDamage"] = formatNumSep(fmt.Sprintf("%.1f", actor.Output["TotalDPS"])) + " DPS"
			}
		}
	*/
	if breakdown != nil {
		if isAttack {
			breakdown.AddLine("TotalDPS",
				fmt.Sprintf("%.1f ^8(average damage)", actor.Output["AverageDamage"]),
				utils.Ternary(actor.Output["HitSpeed"] != 0, fmt.Sprintf("x %.2f ^8(hit rate)", actor.Output["HitSpeed"]), fmt.Sprintf("x %.2f ^8(attack rate)", actor.Output["Speed"])),
			)
		} else if isTriggered {
			breakdown.AddLine("TotalDPS",
				fmt.Sprintf("%.1f ^8(average damage)", actor.Output["AverageDamage"]),
				utils.Ternary(actor.Output["HitSpeed"] != 0, fmt.Sprintf("x %.2f ^8(hit rate)", actor.Output["HitSpeed"]), fmt.Sprintf("x %.2f ^8(trigger rate)", actor.Output["Speed"])),
			)
		} else {
			breakdown.AddLine("TotalDPS",
				fmt.Sprintf("%.1f ^8(average hit)", actor.Output["AverageDamage"]),
				utils.Ternary(actor.Output["HitSpeed"] != 0, fmt.Sprintf("x %.2f ^8(hit rate)", actor.Output["HitSpeed"]), fmt.Sprintf("x %.2f ^8(cast rate)", actor.Output["Speed"])),
			)
		}
		if skillData.DpsMultiplier != 0 {
			breakdown.AddLine("TotalDPS", fmt.Sprintf("x %g ^8(DPS multiplier for this skill)", skillData.DpsMultiplier))
		}
		if quantityMultiplier > 1 {
			breakdown.AddLine("TotalDPS", fmt.Sprintf("x %g ^8(quantity multiplier for this skill)", quantityMultiplier))
		}
		breakdown.AddLine("TotalDPS", fmt.Sprintf("= %.1f", actor.Output["TotalDPS"]))
	}
	/*
		TODO // Calculate leech rates
		actor.Output["LifeLeechInstanceRate"] = actor.Output["Life"] * data.LeechRateBase * calclib.Mod(skillModList, skillCfg, "LifeLeechRate")
		actor.Output["LifeLeechRate"] = actor.Output["LifeLeechInstantRate"] + min(actor.Output["LifeLeechInstances"] * actor.Output["LifeLeechInstanceRate"], actor.Output["MaxLifeLeechRate"]) * actor.Output["LifeRecoveryRateMod"]
		actor.Output["LifeLeechPerHit"] = actor.Output["LifeLeechInstant"] + min(actor.Output["LifeLeechInstanceRate"], actor.Output["MaxLifeLeechRate"]) * actor.Output["LifeLeechDuration"] * actor.Output["LifeRecoveryRateMod"]
		actor.Output["EnergyShieldLeechInstanceRate"] = actor.Output["EnergyShield"] * data.LeechRateBase * calclib.Mod(skillModList, skillCfg, "EnergyShieldLeechRate")
		actor.Output["EnergyShieldLeechRate"] = actor.Output["EnergyShieldLeechInstantRate"] + min(actor.Output["EnergyShieldLeechInstances"] * actor.Output["EnergyShieldLeechInstanceRate"], actor.Output["MaxEnergyShieldLeechRate"]) * actor.Output["EnergyShieldRecoveryRateMod"]
		actor.Output["EnergyShieldLeechPerHit"] = actor.Output["EnergyShieldLeechInstant"] + min(actor.Output["EnergyShieldLeechInstanceRate"], actor.Output["MaxEnergyShieldLeechRate"]) * actor.Output["EnergyShieldLeechDuration"] * actor.Output["EnergyShieldRecoveryRateMod"]
		actor.Output["ManaLeechInstanceRate"] = actor.Output["Mana"] * data.LeechRateBase * calclib.Mod(skillModList, skillCfg, "ManaLeechRate")
		actor.Output["ManaLeechRate"] = actor.Output["ManaLeechInstantRate"] + min(actor.Output["ManaLeechInstances"] * actor.Output["ManaLeechInstanceRate"], actor.Output["MaxManaLeechRate"]) * actor.Output["ManaRecoveryRateMod"]
		actor.Output["ManaLeechPerHit"] = actor.Output["ManaLeechInstant"] + min(actor.Output["ManaLeechInstanceRate"], actor.Output["MaxManaLeechRate"]) * actor.Output["ManaLeechDuration"] * actor.Output["ManaRecoveryRateMod"]
		// On full life, Immortal Ambition treats life leech as energy shield leech
		if skillModList.Flag(nil, "ImmortalAmbition") {
			actor.Output["EnergyShieldLeechRate"] = actor.Output["EnergyShieldLeechRate"] + actor.Output["LifeLeechRate"]
			actor.Output["EnergyShieldLeechPerHit"] = actor.Output["EnergyShieldLeechPerHit"]  + actor.Output["LifeLeechPerHit"]
			// Clears actor.Output["LifeLeechRate"] to disable leechLife flag
			actor.Output["LifeLeechRate"] = 0
		}
		skillFlags.leechLife = actor.Output["LifeLeechRate"] > 0
		skillFlags.leechES = actor.Output["EnergyShieldLeechRate"] > 0
		skillFlags.leechMana = actor.Output["ManaLeechRate"] > 0
		if skillData.ShowAverage {
			actor.Output["LifeLeechGainPerHit"] = actor.Output["LifeLeechPerHit"] + actor.Output["LifeOnHit"]
			actor.Output["EnergyShieldLeechGainPerHit"] = actor.Output["EnergyShieldLeechPerHit"] + actor.Output["EnergyShieldOnHit"]
			actor.Output["ManaLeechGainPerHit"] = actor.Output["ManaLeechPerHit"] + actor.Output["ManaOnHit"]
		} else {
			actor.Output["LifeLeechGainRate"] = actor.Output["LifeLeechRate"] + actor.Output["LifeOnHitRate"]
			actor.Output["EnergyShieldLeechGainRate"] = actor.Output["EnergyShieldLeechRate"] + actor.Output["EnergyShieldOnHitRate"]
			actor.Output["ManaLeechGainRate"] = actor.Output["ManaLeechRate"] + actor.Output["ManaOnHitRate"]
		}
		if breakdown != nil {
			if skillFlags.leechLife {
				breakdown.LifeLeech = breakdown.leech(actor.Output["LifeLeechInstant"], actor.Output["LifeLeechInstantRate"], actor.Output["LifeLeechInstances"], actor.Output["Life"], "LifeLeechRate", actor.Output["MaxLifeLeechRate"], actor.Output["LifeLeechDuration"])
			}
			if skillFlags.leechES {
				breakdown.EnergyShieldLeech = breakdown.leech(actor.Output["EnergyShieldLeechInstant"], actor.Output["EnergyShieldLeechInstantRate"], actor.Output["EnergyShieldLeechInstances"], actor.Output["EnergyShield"], "EnergyShieldLeechRate", actor.Output["MaxEnergyShieldLeechRate"], actor.Output["EnergyShieldLeechDuration"])
			}
			if skillFlags.leechMana {
				breakdown.ManaLeech = breakdown.leech(actor.Output["ManaLeechInstant"], actor.Output["ManaLeechInstantRate"], actor.Output["ManaLeechInstances"], actor.Output["Mana"], "ManaLeechRate", actor.Output["MaxManaLeechRate"], actor.Output["ManaLeechDuration"])
			}
		}
	*/
	/*
		TODO Calculate Ailments
		ailmentData := data.nonDamagingAilment
		for _, ailment := range (ailmentTypeList) {
			skillFlags[string.lower(ailment)] = false
		}
		skillFlags.igniteCanStack = skillModList.Flag(skillCfg, "IgniteCanStack")
		skillFlags.igniteToChaos = skillModList.Flag(skillCfg, "IgniteToChaos")
		skillFlags.impale = false
	*/

	for _, pass := range passList {
		globalOutput := actor.Output
		// globalBreakdown := breakdown
		// source := pass.Source
		output, cfg := pass.Output, pass.Config
		// breakdown := pass.Breakdown

		// Calculate chance to inflict secondary dots/status effects
		cfg.SkillCond["CriticalStrike"] = true

		if !skillFlags[SkillFlagAttack] || skillModList.Flag(cfg, "CannotBleed") {
			output["BleedChanceOnCrit"] = 0
		} else {
			output["BleedChanceOnCrit"] = min(100, skillModList.Sum(mod.TypeBase, cfg, "BleedChance")+enemyDB.Sum(mod.TypeBase, nil, "SelfBleedChance"))
		}

		if !skillFlags[SkillFlagHit] || skillModList.Flag(cfg, "CannotPoison") {
			output["PoisonChanceOnCrit"] = 0
		} else {
			output["PoisonChanceOnCrit"] = min(100, skillModList.Sum(mod.TypeBase, cfg, "PoisonChance")+enemyDB.Sum(mod.TypeBase, nil, "SelfPoisonChance"))
		}

		if !skillFlags[SkillFlagHit] || skillModList.Flag(cfg, "CannotKnockback") {
			output["KnockbackChanceOnCrit"] = 0
		} else {
			output["KnockbackChanceOnCrit"] = skillModList.Sum(mod.TypeBase, cfg, "EnemyKnockbackChance")
		}

		cfg.SkillCond["CriticalStrike"] = false

		if !skillFlags[SkillFlagAttack] || skillModList.Flag(cfg, "CannotBleed") {
			output["BleedChanceOnHit"] = 0
		} else {
			output["BleedChanceOnHit"] = min(100, skillModList.Sum(mod.TypeBase, cfg, "BleedChance")+enemyDB.Sum(mod.TypeBase, nil, "SelfBleedChance"))
		}

		if !skillFlags[SkillFlagHit] || skillModList.Flag(cfg, "CannotPoison") {
			output["PoisonChanceOnHit"] = 0
			output["ChaosPoisonChance"] = 0
		} else {
			output["PoisonChanceOnHit"] = min(100, skillModList.Sum(mod.TypeBase, cfg, "PoisonChance")+enemyDB.Sum(mod.TypeBase, nil, "SelfPoisonChance"))
			output["ChaosPoisonChance"] = min(100, skillModList.Sum(mod.TypeBase, cfg, "ChaosPoisonChance"))
		}

		for _, ailment := range data.ElementalAilment("").Values() {
			chance := skillModList.Sum(mod.TypeBase, cfg, "Enemy"+string(ailment)+"Chance") + enemyDB.Sum(mod.TypeBase, nil, "Self"+string(ailment)+"Chance")
			if ailment == data.AilmentChill {
				chance = 100
			}

			if skillFlags[SkillFlagHit] && !skillModList.Flag(cfg, "Cannot"+string(ailment)) {
				output[string(ailment)+"ChanceOnHit"] = min(100, chance)

				if skillModList.Flag(cfg, "CritsDontAlways"+string(ailment)) || // e.g. Painseeker
					(ailment.IsNonDamaging() && data.NonDamagingAilments[ailment].Alt && !skillModList.Flag(cfg, "CritAlwaysAltAilments")) { // e.g. Secrets of Suffering
					output[string(ailment)+"ChanceOnCrit"] = output[string(ailment)+"ChanceOnHit"]
				} else {
					output[string(ailment)+"ChanceOnCrit"] = 100
				}
			} else {
				output[string(ailment)+"ChanceOnHit"] = 0
				output[string(ailment)+"ChanceOnCrit"] = 0
			}

			ailmentChance := output[string(ailment)+"ChanceOnHit"]
			if !skillModList.Flag(cfg, "NeverCrit") {
				ailmentChance += output[string(ailment)+"ChanceOnCrit"]
			}

			if ailmentChance > 0 {
				skillFlags[SkillFlag("inflict"+string(ailment))] = true
			}
		}

		if !skillFlags[SkillFlagHit] || skillModList.Flag(cfg, "CannotKnockback") {
			output["KnockbackChanceOnHit"] = 0
		} else {
			output["KnockbackChanceOnHit"] = skillModList.Sum(mod.TypeBase, cfg, "EnemyKnockbackChance")
		}

		output["ImpaleChance"] = min(100, skillModList.Sum(mod.TypeBase, cfg, "ImpaleChance"))

		if skillModList.Sum(mod.TypeBase, cfg, "FireExposureChance") > 0 {
			skillFlags["applyFireExposure"] = true
		}

		if skillModList.Sum(mod.TypeBase, cfg, "ColdExposureChance") > 0 {
			skillFlags["applyColdExposure"] = true
		}

		if skillModList.Sum(mod.TypeBase, cfg, "LightningExposureChance") > 0 {
			skillFlags["applyLightningExposure"] = true
		}

		if env.ModeEffective {
			for _, ailment := range data.Ailment("").Values() {
				mult := 1 - enemyDB.Sum(mod.TypeBase, nil, "Avoid"+string(ailment))/100
				output[string(ailment)+"ChanceOnHit"] = output[string(ailment)+"ChanceOnHit"] * mult
				output[string(ailment)+"ChanceOnCrit"] = output[string(ailment)+"ChanceOnCrit"] * mult
				if ailment == data.AilmentPoison {
					output["ChaosPoisonChance"] = output["ChaosPoisonChance"] * mult
				}
			}
		}

		/*
			TODO igniteMode
			igniteMode := env.configInput.igniteMode or "AVERAGE"
			if igniteMode == "CRIT" {
				for _, ailment := range (ailmentTypeList) {
					output[ailment+"ChanceOnHit"] = 0
				}
			}
		*/
		/*
			TODO calcAverageSourceDamage
			//-Calculates normal and crit damage to be used in non-damaging ailment calculations
			//-@param ailment string
			//-@return number, number @average hit damage, average crit damage
			local function calcAverageSourceDamage(ailment)
				sourceHitDmg, sourceCritDmg := 0, 0
				for _, type := range data.DamageType("").Values() {
					if canDeal[type] and (function()
						if type == ailmentData[ailment].associatedType {
							return not skillModList.Flag(cfg, type+"Cannot"+ailment)
						} else {
							return skillModList.Flag(cfg, type+"Can"+ailment)
						}
					end)() {
						sourceHitDmg = sourceHitDmg + output[type+"HitAverage"]
						sourceCritDmg = sourceCritDmg + output[type+"CritAverage"]
					}
				}
				return sourceHitDmg, sourceCritDmg
			}
		*/

		calcAilmentDamage := func(typee string, sourceHitDmg float64, sourceCritDmg float64) float64 {
			// Calculate the inflict chance and base damage of a secondary effect (bleed/poison/ignite/shock/freeze)
			chanceOnHit, chanceOnCrit := output[typee+"ChanceOnHit"], output[typee+"ChanceOnCrit"]
			chanceFromHit := chanceOnHit * (1 - output["CritChance"]/100)
			chanceFromCrit := chanceOnCrit * output["CritChance"] / 100
			chance := chanceFromHit + chanceFromCrit
			output[typee+"Chance"] = chance
			baseFromHit := sourceHitDmg * chanceFromHit / (chanceFromHit + chanceFromCrit)
			baseFromCrit := sourceCritDmg * chanceFromCrit / (chanceFromHit + chanceFromCrit)
			baseVal := baseFromHit + baseFromCrit
			/*
				TODO Breakdown
				sourceMult := skillModList.More(nil, typee+"AsThoughDealing")
				if breakdown and chance != 0 {
					breakdownChance := breakdown[type+"Chance"] or { }
					breakdown[type+"Chance"] = breakdownChance
					if breakdownChance[1] {
						t_insert(breakdownChance, "")
					}
					if isAttack {
						t_insert(breakdownChance, pass.label+":")
					}
					t_insert(breakdownChance, fmt.Sprintf("Chance on Non-crit: %d%%", chanceOnHit))
					t_insert(breakdownChance, fmt.Sprintf("Chance on Crit: %d%%", chanceOnCrit))
					if chanceOnHit != chanceOnCrit {
						t_insert(breakdownChance, "Combined chance:")
						t_insert(breakdownChance, fmt.Sprintf("%d x (1 - %.4f) ^8(chance from non-crits)", chanceOnHit, actor.Output["CritChance"]/100))
						t_insert(breakdownChance, fmt.Sprintf("+ %d x %.4f ^8(chance from crits)", chanceOnCrit, actor.Output["CritChance"]/100))
						t_insert(breakdownChance, fmt.Sprintf("= %.2f", chance))
					}
				}
				if breakdown and baseVal > 0 {
					breakdownDPS := breakdown[type+"DPS"] or { }
					breakdown[type+"DPS"] = breakdownDPS
					if breakdownDPS[1] {
						t_insert(breakdownDPS, "")
					}
					if isAttack {
						t_insert(breakdownDPS, pass.label+":")
					}
					if sourceHitDmg == sourceCritDmg {
						t_insert(breakdownDPS, "Total damage:")
						t_insert(breakdownDPS, fmt.Sprintf("%.1f ^8(source damage)",sourceHitDmg))
						if sourceMult > 1 {
							t_insert(breakdownDPS, fmt.Sprintf("x %.2f ^8(inflicting as though dealing more damage)", sourceMult))
							t_insert(breakdownDPS, fmt.Sprintf("= %.1f", baseVal * sourceMult))
						}
					} else {
						if baseFromHit > 0 {
							t_insert(breakdownDPS, "Damage from Non-crits:")
							t_insert(breakdownDPS, fmt.Sprintf("%.1f ^8(source damage from non-crits)", sourceHitDmg))
							t_insert(breakdownDPS, fmt.Sprintf("x %.3f ^8(portion of instances created by non-crits)", chanceFromHit / (chanceFromHit + chanceFromCrit)))
							if sourceMult == 1 or baseFromCrit != 0 {
								t_insert(breakdownDPS, fmt.Sprintf("= %.1f", baseFromHit))
							}
						}
						if baseFromCrit > 0 {
							t_insert(breakdownDPS, "Damage from Crits:")
							t_insert(breakdownDPS, fmt.Sprintf("%.1f ^8(source damage from crits)", sourceCritDmg))
							t_insert(breakdownDPS, fmt.Sprintf("x %.3f ^8(portion of instances created by crits)", chanceFromCrit / (chanceFromHit + chanceFromCrit)))
							if sourceMult == 1 or baseFromHit != 0 {
								t_insert(breakdownDPS, fmt.Sprintf("= %.1f", baseFromCrit))
							}
						}
						if baseFromHit > 0 and baseFromCrit > 0 {
							t_insert(breakdownDPS, "Total damage:")
							t_insert(breakdownDPS, fmt.Sprintf("%.1f + %.1f", baseFromHit, baseFromCrit))
							if sourceMult == 1 {
								t_insert(breakdownDPS, fmt.Sprintf("= %.1f", baseVal))
							}
						}
						if sourceMult > 1 {
							t_insert(breakdownDPS, fmt.Sprintf("x %.2f ^8(inflicting as though dealing more damage)", sourceMult))
							t_insert(breakdownDPS, fmt.Sprintf("= %.1f", baseVal * sourceMult))
						}
					}
				}
			*/
			return baseVal
		}

		// Calculate bleeding chance and damage
		if canDeal[data.DamageTypePhysical] && (output["BleedChanceOnHit"]+output["BleedChanceOnCrit"]) > 0 {
			// TODO This is almost definitely a bad idea but I cba to implement a new interface with a struct that references underlying map and returns values so this is what you get
			badIdea := maps.Clone(skillCfg.SkillCond)
			badIdea["CriticalStrike"] = true

			dotCfg := &moddb.ListCfg{
				// TODO SkillName, SkillPart, SkillTypes, SkillDist
				// SkillName: skillCfg.SkillName,
				// SkillPart: skillCfg.SkillPart,
				// SkillTypes: skillCfg.SkillTypes,
				SlotName:     skillCfg.SlotName,
				Flags:        utils.Ptr(mod.MFlagDot | mod.MFlagAilment | (cfg.Flags.Get() & mod.MFlagWeaponMask) | utils.Ternary((cfg.Flags.Get()&mod.MFlagMelee) != 0, mod.MFlagMeleeHit, 0)),
				KeywordFlags: utils.Ptr((cfg.KeywordFlags.Get() & ^mod.KeywordFlagHit) | mod.KeywordFlagBleed | mod.KeywordFlagAilment | mod.KeywordFlagPhysicalDot),
				SkillCond:    badIdea,
				// SkillDist: skillCfg.SkillDist,
			}

			if strings.Contains(pass.Label, "Off Hand") {
				activeSkill.OHBleedCfg = dotCfg
			} else {
				activeSkill.BleedCfg = dotCfg
			}

			sourceHitDmg := float64(0)
			sourceCritDmg := float64(0)
			/*
				TODO Breakdown
				if breakdown != nil {
					breakdown.BleedPhysical = { damageTypes = { } }
				}
			*/

			// For bleeds we will be using a weighted average calculation

			configStacks := enemyDB.Sum(mod.TypeBase, nil, "Multiplier:BleedStacks")
			maxStacks := skillModList.Sum(mod.TypeBase, cfg, "BleedStacksMax")
			if skillModList.Override(cfg, "BleedStacksMax") != nil {
				maxStacks = skillModList.Override(cfg, "BleedStacksMax").Float()
			}
			globalOutput["BleedStacksMax"] = maxStacks
			durationBase := utils.Ternary(skillData.BleedDurationIsSkillDuration, skillData.Duration, data.BleedDurationBase)

			names := []string{"EnemyBleedDuration", "SkillAndDamagingAilmentDuration"}
			if skillData.BleedIsSkillEffect {
				names = append(names, "Duration")
			}
			durationMod := CalcMod(skillModList, dotCfg, names...) * CalcMod(enemyDB, nil, "SelfBleedDuration") / CalcMod(enemyDB, dotCfg, "BleedExpireRate")
			rateMod := CalcMod(skillModList, cfg, "BleedFaster") + enemyDB.Sum(mod.TypeIncrease, nil, "SelfBleedFaster")/100
			globalOutput["BleedDuration"] = durationBase * durationMod / rateMod * debuffDurationMult
			bleedStacks := (output["HitChance"] / 100) * (globalOutput["BleedDuration"] / output["Time"]) / maxStacks
			bleedStacks = utils.Ternary(configStacks > 0, min(bleedStacks, configStacks/maxStacks), bleedStacks)
			globalOutput["BleedStackPotential"] = bleedStacks
			/*
				TODO Breakdown
				if globalBreakdown {
					globalBreakdown.BleedStackPotential = {
						fmt.Sprintf(colorCodes.CUSTOM+"NOTE: Calculation uses new Weighted Avg Ailment formula"),
						fmt.Sprintf(""),
						fmt.Sprintf("%.2f ^8(chance to hit)", actor.Output["HitChance"] / 100),
						fmt.Sprintf("* (%.2f / %.2f) ^8(BleedDuration / Attack Time)", globalOutput.BleedDuration, actor.Output["Time"]),
						fmt.Sprintf("/ %d ^8(max number of stacks)", maxStacks),
						fmt.Sprintf("= %.2f", globalOutput.BleedStackPotential),
					}
				}
			*/

			for _, sub_pass := range []int{1, 2} {
				if skillModList.Flag(dotCfg, "AilmentsAreNeverFromCrit") || sub_pass == 1 {
					dotCfg.SkillCond["CriticalStrike"] = false
				} else {
					dotCfg.SkillCond["CriticalStrike"] = true
				}

				// TODO Breakdown
				// sub_pass == 1 && breakdown && breakdown.BleedPhysical
				output["BleedPhysicalMin"], output["BleedPhysicalMax"] = calcAilmentSourceDamage(activeSkill, output, dotCfg, breakdown, "BleedPhysical", "Physical", 0)

				if sub_pass == 2 {
					output["CritBleedDotMulti"] = 1 + skillModList.Sum(mod.TypeBase, dotCfg, "DotMultiplier", "PhysicalDotMultiplier")/100
					sourceCritDmg = output["BleedPhysicalMin"] + output["BleedPhysicalMax"] - output["BleedPhysicalMin"]/math.Pow(2, float64(1)/(bleedStacks+1))*output["CritBleedDotMulti"]
				} else {
					output["BleedDotMulti"] = 1 + skillModList.Sum(mod.TypeBase, dotCfg, "DotMultiplier", "PhysicalDotMultiplier")/100
					sourceHitDmg = output["BleedPhysicalMin"] + output["BleedPhysicalMax"] - output["BleedPhysicalMin"]/math.Pow(2, float64(1)/(bleedStacks+1))*output["CritBleedDotMulti"]
				}
			}

			/*
				TODO Breakdown
				if globalBreakdown {
					if sourceHitDmg == sourceCritDmg {
						globalBreakdown.BleedDPS = {
							fmt.Sprintf(colorCodes.CUSTOM+"NOTE: Calculation uses new Weighted Avg Ailment formula"),
							fmt.Sprintf(""),
							fmt.Sprintf("Dmg Derivation:"),
							fmt.Sprintf("(%.2f + (%.2f - %.2f) ^8(min source physical + (max source physical - min source physical)", actor.Output["BleedPhysicalMin"], actor.Output["BleedPhysicalMax"], actor.Output["BleedPhysicalMin"]),
							fmt.Sprintf("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", bleedStacks),
							fmt.Sprintf("* %.2f ^8(Bleed DoT Multi)", actor.Output["BleedDotMulti"]),
							fmt.Sprintf("= %.2f", sourceHitDmg),
						}
					} else {
						globalBreakdown.BleedDPS = {
							fmt.Sprintf(colorCodes.CUSTOM+"NOTE: Calculation uses new Weighted Avg Ailment formula"),
							fmt.Sprintf(""),
							fmt.Sprintf("Non-Crit Dmg Derivation:"),
							fmt.Sprintf("(%.2f + (%.2f - %.2f) ^8(min source physical + (max source physical - min source physical)", actor.Output["BleedPhysicalMin"], actor.Output["BleedPhysicalMax"], actor.Output["BleedPhysicalMin"]),
							fmt.Sprintf("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", bleedStacks),
							fmt.Sprintf("* %.2f ^8(Bleed DoT Multi for Non-Crit)", actor.Output["BleedDotMulti"]),
							fmt.Sprintf("= %.2f", sourceHitDmg),
							fmt.Sprintf(""),
							fmt.Sprintf("Crit Dmg Derivation:"),
							fmt.Sprintf("(%.2f + (%.2f - %.2f) ^8(min source physical + (max source physical - min source physical)", actor.Output["BleedPhysicalMin"], actor.Output["BleedPhysicalMax"], actor.Output["BleedPhysicalMin"]),
							fmt.Sprintf("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", bleedStacks),
							fmt.Sprintf("* %.2f ^8(Bleed DoT Multi for Crit)", actor.Output["CritBleedDotMulti"]),
							fmt.Sprintf("= %.2f", sourceCritDmg),
						}
					}
				}
			*/

			basePercent := utils.Ternary(skillData.BleedBasePercent != 0, skillData.BleedBasePercent, data.BleedPercentBase)
			baseVal := calcAilmentDamage("Bleed", sourceHitDmg, sourceCritDmg) * basePercent / 100 * output["RuthlessBlowBleedEffect"] * output["FistOfWarAilmentEffect"] * globalOutput["AilmentWarcryEffect"]
			if baseVal > 0 {
				skillFlags[SkillFlagBleed] = true
				skillFlags[SkillFlagDuration] = true
				effMult := float64(1)
				if env.ModeEffective {
					resist := min(max(0, enemyDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction")), data.DamageReductionCap)
					takenInc := enemyDB.Sum(mod.TypeIncrease, dotCfg, "DamageTaken", "DamageTakenOverTime", "PhysicalDamageTaken", "PhysicalDamageTakenOverTime")
					takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "PhysicalDamageTaken", "PhysicalDamageTakenOverTime")
					effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
					globalOutput["BleedEffMult"] = effMult
					/*
						TODO Breakdown
						if breakdown and effMult != 1 {
							globalBreakdown.BleedEffMult = breakdown.effMult("Physical", resist, 0, takenInc, effMult, takenMore)
						}
					*/
				}

				effectMod := CalcMod(skillModList, dotCfg, "AilmentEffect")
				output["BaseBleedDPS"] = baseVal * effectMod * rateMod * effMult
				bleedStacks = min(maxStacks, (output["HitChance"]/100)*globalOutput["BleedDuration"]/output["Time"])
				chanceToHitInOneSecInterval := 1 - math.Pow(1-(output["HitChance"]/100), output["Speed"])
				output["BleedDPS"] = (baseVal * effectMod * rateMod * effMult) * bleedStacks * chanceToHitInOneSecInterval
				// reset bleed stacks to actual number doing damage after weighted avg DPS calculation is done
				globalOutput["BleedStacks"] = bleedStacks
				globalOutput["BleedDamage"] = output["BaseBleedDPS"] * globalOutput["BleedDuration"]
				/*
					if breakdown != nil {
						if actor.Output["CritBleedDotMulti"] and (actor.Output["CritBleedDotMulti"] != actor.Output["BleedDotMulti"]) {
							chanceFromHit := actor.Output["BleedChanceOnHit"] / 100 * (1 - globalOutput.CritChance / 100)
							chanceFromCrit := actor.Output["BleedChanceOnCrit"] / 100 * actor.Output["CritChance"] / 100
							totalFromHit := chanceFromHit / (chanceFromHit + chanceFromCrit)
							totalFromCrit := chanceFromCrit / (chanceFromHit + chanceFromCrit)
							breakdown.BleedDotMulti = breakdown.critDot(actor.Output["BleedDotMulti"], actor.Output["CritBleedDotMulti"], totalFromHit, totalFromCrit)
							actor.Output["BleedDotMulti"] = (actor.Output["BleedDotMulti"] * totalFromHit) + (actor.Output["CritBleedDotMulti"] * totalFromCrit)
						}
						breakdown.AddLine("BleedDPS", fmt.Sprintf("x %.2f ^8(bleed deals %d%% per second)", basePercent/100, basePercent))
						if effectMod != 1 {
							breakdown.AddLine("BleedDPS", fmt.Sprintf("x %.2f ^8(ailment effect modifier)", effectMod))
						}
						if actor.Output["RuthlessBlowBleedEffect"] != 1 {
							breakdown.AddLine("BleedDPS", fmt.Sprintf("x %.2f ^8(ruthless blow effect modifier)", actor.Output["RuthlessBlowBleedEffect"]))
						}
						if actor.Output["FistOfWarAilmentEffect"] != 1 {
							breakdown.AddLine("BleedDPS", fmt.Sprintf("x %.2f ^8(fist of war effect modifier)", actor.Output["FistOfWarAilmentEffect"]))
						}
						if globalOutput.AilmentWarcryEffect > 1 {
							breakdown.AddLine("BleedDPS", fmt.Sprintf("x %.2f ^8(combined ailment warcry effect modifier)", globalOutput.AilmentWarcryEffect))
						}
						breakdown.AddLine("BleedDPS", fmt.Sprintf("= %.1f", baseVal))
						breakdown.multiChain(breakdown.BleedDPS, {
							label = "Bleed DPS:",
							base = fmt.Sprintf("%.1f ^8(total damage per second)", baseVal),
							{ "%.2f ^8(ailment effect modifier)", effectMod },
							{ "%.2f ^8(damage rate modifier)", rateMod },
							{ "%.3f ^8(effective DPS modifier)", effMult },
							{ "%d ^8(bleed stacks)", globalOutput.BleedStacks },
							{ "%.3f ^8(bleed chance based on chance to hit each second)", chanceToHitInOneSecInterval },
							total = fmt.Sprintf("= %.1f ^8per second", actor.Output["BleedDPS"]),
						})
						if globalOutput.BleedDuration != durationBase {
							globalBreakdown.BleedDuration = {
								fmt.Sprintf("%.2fs ^8(base duration)", durationBase)
							}
							if durationMod != 1 {
								t_insert(globalBreakdown.BleedDuration, fmt.Sprintf("x %.2f ^8(duration modifier)", durationMod))
							}
							if rateMod != 1 {
								t_insert(globalBreakdown.BleedDuration, fmt.Sprintf("/ %.2f ^8(damage rate modifier)", rateMod))
							}
							if debuffDurationMult != 1 {
								t_insert(globalBreakdown.BleedDuration, fmt.Sprintf("/ %.2f ^8(debuff expires slower/faster)", 1 / debuffDurationMult))
							}
							t_insert(globalBreakdown.BleedDuration, fmt.Sprintf("= %.2fs", globalOutput.BleedDuration))
						}
					}
				*/
			}
		}
		/*
			TODO Calculate poison chance and damage
			if canDeal.Chaos and (actor.Output["PoisonChanceOnHit"] + actor.Output["PoisonChanceOnCrit"] + actor.Output["ChaosPoisonChance"]) > 0 {
				activeSkill[pass.label != "Off Hand" and "poisonCfg" or "OHpoisonCfg"] = {
					skillName = skillCfg.skillName,
					skillPart = skillCfg.skillPart,
					skillTypes = skillCfg.skillTypes,
					slotName = skillCfg.slotName,
					flags = bor(ModFlag.Dot, ModFlag.Ailment, band(cfg.flags, ModFlag.WeaponMask), band(cfg.flags, ModFlag.Melee) != 0 and ModFlag.MeleeHit or 0),
					keywordFlags = bor(band(cfg.keywordFlags, bnot(KeywordFlag.Hit)), KeywordFlag.Poison, KeywordFlag.Ailment, KeywordFlag.ChaosDot),
					skillCond = setmetatable({["CriticalStrike"] = true }, { __index = function(table, key) return skillCfg.skillCond[key] or cfg.skillCond[key] end } ),
					skillDist = skillCfg.skillDist,
				}
				dotCfg := pass.label != "Off Hand" and activeSkill.poisonCfg or activeSkill.OHpoisonCfg
				local sourceHitDmg, sourceCritDmg
				if breakdown != nil {
					breakdown.PoisonPhysical = { damageTypes = { } }
					breakdown.PoisonLightning = { damageTypes = { } }
					breakdown.PoisonCold = { damageTypes = { } }
					breakdown.PoisonFire = { damageTypes = { } }
					breakdown.PoisonChaos = { damageTypes = { } }
				}
				for sub_pass = 1, 2 {
					if skillModList.Flag(dotCfg, "AilmentsAreNeverFromCrit") or sub_pass == 1 {
						dotCfg.skillCond["CriticalStrike"] = false
					} else {
						dotCfg.skillCond["CriticalStrike"] = true
					}
					totalMin, totalMax := 0, 0
					{
						min, max := calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.PoisonChaos, "Chaos", 0)
						actor.Output["PoisonChaosMin"] = min
						actor.Output["PoisonChaosMax"] = max
						totalMin = totalMin + min
						totalMax = totalMax + max
					}
					nonChaosMult := 1
					if actor.Output["ChaosPoisonChance"] > 0 and actor.Output["PoisonChaosMax"] > 0 {
						// Additional chance for chaos
						chance := (sub_pass == 2) and "PoisonChanceOnCrit" or "PoisonChanceOnHit"
						chaosChance := min(100, output[chance] + actor.Output["ChaosPoisonChance"])
						nonChaosMult = output[chance] / chaosChance
						output[chance] = chaosChance
					}
					if canDeal.Lightning and skillModList.Flag(cfg, "LightningCanPoison") {
						min, max := calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.PoisonLightning, "Lightning", dmgTypeFlags.Chaos)
						actor.Output["PoisonLightningMin"] = min
						actor.Output["PoisonLightningMax"] = max
						totalMin = totalMin + min * nonChaosMult
						totalMax = totalMax + max * nonChaosMult
					}
					if canDeal.Cold and skillModList.Flag(cfg, "ColdCanPoison") {
						min, max := calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.PoisonCold, "Cold", dmgTypeFlags.Chaos)
						actor.Output["PoisonColdMin"] = min
						actor.Output["PoisonColdMax"] = max
						totalMin = totalMin + min * nonChaosMult
						totalMax = totalMax + max * nonChaosMult
					}
					if canDeal.Fire and skillModList.Flag(cfg, "FireCanPoison") {
						min, max := calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.PoisonFire, "Fire", dmgTypeFlags.Chaos)
						actor.Output["PoisonFireMin"] = min
						actor.Output["PoisonFireMax"] = max
						totalMin = totalMin + min * nonChaosMult
						totalMax = totalMax + max * nonChaosMult
					}
					if canDeal.Physical {
						min, max := calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.PoisonPhysical, "Physical", dmgTypeFlags.Chaos)
						actor.Output["PoisonPhysicalMin"] = min
						actor.Output["PoisonPhysicalMax"] = max
						totalMin = totalMin + min * nonChaosMult
						totalMax = totalMax + max * nonChaosMult
					}
					if sub_pass == 2 {
						actor.Output["CritPoisonDotMulti"] = 1 + skillModList.Sum(mod.TypeBase, dotCfg, "DotMultiplier", "ChaosDotMultiplier") / 100
						sourceCritDmg = (totalMin + totalMax) / 2 * actor.Output["CritPoisonDotMulti"]
					} else {
						actor.Output["PoisonDotMulti"] = 1 + skillModList.Sum(mod.TypeBase, dotCfg, "DotMultiplier", "ChaosDotMultiplier") / 100
						sourceHitDmg = (totalMin + totalMax) / 2 * actor.Output["PoisonDotMulti"]
					}
				}
				if globalBreakdown {
					globalBreakdown.PoisonDPS = {
						fmt.Sprintf("Ailment mode: %s ^8(can be changed in the Configuration tab)", igniteMode == "CRIT" and "Crits Only" or "Average Damage")
					}
				}
				baseVal := calcAilmentDamage("Poison", sourceHitDmg, sourceCritDmg) * data.PoisonPercentBase * actor.Output["FistOfWarAilmentEffect"] * globalOutput.AilmentWarcryEffect
				if baseVal > 0 {
					skillFlags.poison = true
					skillFlags.duration = true
					effMult := 1
					if env.mode_effective {
						resist := min(enemyDB.Sum(mod.TypeBase, nil, "ChaosResist") * calclib.Mod(enemyDB, nil, "ChaosResist"), data.EnemyMaxResist)
						takenInc := enemyDB.Sum(mod.TypeIncrease, dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
						takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
						effMult = (1 - resist / 100) * (1 + takenInc / 100) * takenMore
						globalOutput["PoisonEffMult"] = effMult
						if breakdown and effMult != 1 {
							globalBreakdown.PoisonEffMult = breakdown.effMult("Chaos", resist, 0, takenInc, effMult, takenMore)
						}
					}
					effectMod := calclib.Mod(skillModList, dotCfg, "AilmentEffect")
					rateMod := calclib.Mod(skillModList, cfg, "PoisonFaster") + enemyDB.Sum(mod.TypeIncrease, nil, "SelfPoisonFaster")  / 100
					actor.Output["PoisonDPS"] = baseVal * effectMod * rateMod * effMult
					local durationBase
					if skillData.poisonDurationIsSkillDuration {
						durationBase = skillData.duration
					} else {
						durationBase = data.PoisonDurationBase
					}
					durationMod := calclib.Mod(skillModList, dotCfg, "EnemyPoisonDuration", "SkillAndDamagingAilmentDuration", skillData.poisonIsSkillEffect and "Duration" or nil) * calclib.Mod(enemyDB, nil, "SelfPoisonDuration")
					globalOutput.PoisonDuration = durationBase * durationMod / rateMod * debuffDurationMult
					actor.Output["PoisonDamage"] = actor.Output["PoisonDPS"] * globalOutput.PoisonDuration
					if skillData.ShowAverage {
						actor.Output["TotalPoisonAverageDamage"] = actor.Output["HitChance"] / 100 * actor.Output["PoisonChance"] / 100 * actor.Output["PoisonDamage"]
						actor.Output["TotalPoisonDPS"] = actor.Output["PoisonDPS"]
					} else {
						actor.Output["TotalPoisonStacks"] = actor.Output["HitChance"] / 100 * actor.Output["PoisonChance"] / 100 * globalOutput.PoisonDuration * (globalOutput.HitSpeed or globalOutput.Speed) * (skillData.dpsMultiplier or 1) * (skillData.stackMultiplier or 1) * quantityMultiplier
						actor.Output["TotalPoisonDPS"] = actor.Output["PoisonDPS"] * actor.Output["TotalPoisonStacks"]
					}
					if breakdown != nil {
						if actor.Output["CritPoisonDotMulti"] and (actor.Output["CritPoisonDotMulti"] != actor.Output["PoisonDotMulti"]) {
							chanceFromHit := actor.Output["PoisonChanceOnHit"] / 100 * (1 - globalOutput.CritChance / 100)
							chanceFromCrit := actor.Output["PoisonChanceOnCrit"] / 100 * globalOutput.CritChance / 100
							totalFromHit := chanceFromHit / (chanceFromHit + chanceFromCrit)
							totalFromCrit := chanceFromCrit / (chanceFromHit + chanceFromCrit)
							breakdown.PoisonDotMulti = breakdown.critDot(actor.Output["PoisonDotMulti"], actor.Output["CritPoisonDotMulti"], totalFromHit, totalFromCrit)
							actor.Output["PoisonDotMulti"] = (actor.Output["PoisonDotMulti"] * totalFromHit) + (actor.Output["CritPoisonDotMulti"] * totalFromCrit)
						}
						breakdown.AddLine("PoisonDPS", "x 0.30 ^8(poison deals 30% per second)")
						breakdown.AddLine("PoisonDPS", fmt.Sprintf("= %.1f", baseVal, 1))
						breakdown.multiChain(breakdown.PoisonDPS, {
							label = "Poison DPS:",
							base = fmt.Sprintf("%.1f ^8(total damage per second)", baseVal),
							{ "%.2f ^8(ailment effect modifier)", effectMod },
							{ "%.2f ^8(damage rate modifier)", rateMod },
							{ "%.3f ^8(effective DPS modifier)", effMult },
							total = fmt.Sprintf("= %.1f ^8per second", actor.Output["PoisonDPS"]),
						})
						if globalOutput.PoisonDuration != 2 {
							globalBreakdown.PoisonDuration = {
								fmt.Sprintf("%.2fs ^8(base duration)", durationBase)
							}
							if durationMod != 1 {
								t_insert(globalBreakdown.PoisonDuration, fmt.Sprintf("x %.2f ^8(duration modifier)", durationMod))
							}
							if rateMod != 1 {
								t_insert(globalBreakdown.PoisonDuration, fmt.Sprintf("/ %.2f ^8(damage rate modifier)", rateMod))
							}
							if debuffDurationMult != 1 {
								t_insert(globalBreakdown.PoisonDuration, fmt.Sprintf("/ %.2f ^8(debuff expires slower/faster)", 1 / debuffDurationMult))
							}
							t_insert(globalBreakdown.PoisonDuration, fmt.Sprintf("= %.2fs", globalOutput.PoisonDuration))
						}
						breakdown.PoisonDamage = { }
						if isAttack {
							breakdown.AddLine("PoisonDamage", pass.label+":")
						}
						breakdown.AddLine("PoisonDamage", fmt.Sprintf("%.1f ^8(damage per second)", actor.Output["PoisonDPS"]))
						breakdown.AddLine("PoisonDamage", fmt.Sprintf("x %.2fs ^8(poison duration)", globalOutput.PoisonDuration))
						breakdown.AddLine("PoisonDamage", fmt.Sprintf("= %.1f ^8damage per poison stack", actor.Output["PoisonDamage"]))
						if not skillData.ShowAverage {
							breakdown.TotalPoisonStacks = { }
							if isAttack {
								breakdown.AddLine("TotalPoisonStacks", pass.label+":")
							}
							breakdown.multiChain(breakdown.TotalPoisonStacks, {
								base = fmt.Sprintf("%.2fs ^8(poison duration)", globalOutput.PoisonDuration),
								{ "%.2f ^8(poison chance)", actor.Output["PoisonChance"] / 100 },
								{ "%.2f ^8(hit chance)", actor.Output["HitChance"] / 100 },
								{ "%.2f ^8(hits per second)", globalOutput.HitSpeed or globalOutput.Speed },
								{ "%g ^8(dps multiplier for this skill)", skillData.dpsMultiplier or 1 },
								{ "%g ^8(stack multiplier for this skill)", skillData.stackMultiplier or 1 },
								{ "%g ^8(quantity multiplier for this skill)", quantityMultiplier },
								total = fmt.Sprintf("= %.1f", actor.Output["TotalPoisonStacks"]),
							})
						}
					}
				}
			}
		*/
		/*
			TODO Calculate ignite chance and damage
			if canDeal.Fire and (actor.Output["IgniteChanceOnHit"] + actor.Output["IgniteChanceOnCrit"]) > 0 {
				activeSkill[pass.label != "Off Hand" and "igniteCfg" or "OHigniteCfg"] = {
					skillName = skillCfg.skillName,
					skillPart = skillCfg.skillPart,
					skillTypes = skillCfg.skillTypes,
					slotName = skillCfg.slotName,
					flags = bor(ModFlag.Dot, ModFlag.Ailment, band(cfg.flags, ModFlag.WeaponMask), band(cfg.flags, ModFlag.Melee) != 0 and ModFlag.MeleeHit or 0),
					keywordFlags = bor(band(cfg.keywordFlags, bnot(KeywordFlag.Hit)), KeywordFlag.Ignite, KeywordFlag.Ailment, KeywordFlag.FireDot),
					skillCond = setmetatable({["CriticalStrike"] = true }, { __index = function(table, key) return skillCfg.skillCond[key] or cfg.skillCond[key] end } ),
					skillDist = skillCfg.skillDist,
				}
				dotCfg := pass.label != "Off Hand" and activeSkill.igniteCfg or activeSkill.OHigniteCfg
				local sourceHitDmg, sourceCritDmg
				if breakdown != nil {
					breakdown.IgnitePhysical = { damageTypes = { } }
					breakdown.IgniteLightning = { damageTypes = { } }
					breakdown.IgniteCold = { damageTypes = { } }
					breakdown.IgniteFire = { damageTypes = { } }
					breakdown.IgniteChaos = { damageTypes = { } }
				}

				// For ignites we will be using a weighted average calculation
				maxStacks := 1
				if skillFlags.igniteCanStack {
					maxStacks = maxStacks + skillModList.Sum(mod.TypeBase, cfg, "IgniteStacks")
				}
				globalOutput.IgniteStacksMax = maxStacks

				rateMod := (calclib.Mod(skillModList, cfg, "IgniteBurnFaster") + enemyDB.Sum(mod.TypeIncrease, nil, "SelfIgniteBurnFaster") / 100)  / calclib.Mod(skillModList, cfg, "IgniteBurnSlower")
				durationBase := data.IgniteDurationBase
				durationMod := max(calclib.Mod(skillModList, dotCfg, "EnemyIgniteDuration", "SkillAndDamagingAilmentDuration") * calclib.Mod(enemyDB, nil, "SelfIgniteDuration"), 0)
				globalOutput.IgniteDuration = durationBase * durationMod / rateMod * debuffDurationMult
				globalOutput.IgniteDuration = globalOutput.IgniteDuration > data.IgniteMinDuration and globalOutput.IgniteDuration or 0
				igniteStacks := 1
				if not skillData.triggeredOnDeath {
					igniteStacks = (globalOutput.IgniteDuration / actor.Output["Time"]) / maxStacks
				}
				globalOutput.IgniteStackPotential = igniteStacks
				if globalBreakdown {
					globalBreakdown.IgniteStackPotential = {
						fmt.Sprintf(colorCodes.CUSTOM+"NOTE: Calculation uses new Weighted Avg Ailment formula"),
						fmt.Sprintf(""),
						fmt.Sprintf("(%.2f / %.2f) ^8(IgniteDuration / Cast Time)", globalOutput.IgniteDuration, actor.Output["Time"]),
						fmt.Sprintf("/ %d ^8(max number of stacks)", maxStacks),
						fmt.Sprintf("= %.2f", globalOutput.IgniteStackPotential),
					}
				}

				for sub_pass = 1, 2 {
					if skillModList.Flag(dotCfg, "AilmentsAreNeverFromCrit") or sub_pass == 1 {
						dotCfg.skillCond["CriticalStrike"] = false
					} else {
						dotCfg.skillCond["CriticalStrike"] = true
					}
					totalMin, totalMax := 0, 0
					if canDeal.Physical and skillModList.Flag(cfg, "PhysicalCanIgnite") {
						min, max := calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.IgnitePhysical, "Physical", dmgTypeFlags.Fire)
						actor.Output["IgnitePhysicalMin"] = min
						actor.Output["IgnitePhysicalMax"] = max
						totalMin = totalMin + min
						totalMax = totalMax + max
					}
					if canDeal.Lightning and skillModList.Flag(cfg, "LightningCanIgnite") {
						min, max := calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.IgniteLightning, "Lightning", dmgTypeFlags.Fire)
						actor.Output["IgniteLightningMin"] = min
						actor.Output["IgniteLightningMax"] = max
						totalMin = totalMin + min
						totalMax = totalMax + max
					}
					if canDeal.Cold and skillModList.Flag(cfg, "ColdCanIgnite") {
						min, max := calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.IgniteCold, "Cold", dmgTypeFlags.Fire)
						actor.Output["IgniteColdMin"] = min
						actor.Output["IgniteColdMax"] = max
						totalMin = totalMin + min
						totalMax = totalMax + max
					}
					if canDeal.Fire and not skillModList.Flag(cfg, "FireCannotIgnite") {
						min, max := calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.IgniteFire, "Fire", 0)
						actor.Output["IgniteFireMin"] = min
						actor.Output["IgniteFireMax"] = max
						totalMin = totalMin + min
						totalMax = totalMax + max
					}
					if canDeal.Chaos and skillModList.Flag(cfg, "ChaosCanIgnite") {
						min, max := calcAilmentSourceDamage(activeSkill, output, dotCfg, sub_pass == 1 and breakdown and breakdown.IgniteChaos, "Chaos", dmgTypeFlags.Fire)
						actor.Output["IgniteChaosMin"] = min
						actor.Output["IgniteChaosMax"] = max
						totalMin = totalMin + min
						totalMax = totalMax + max
					}
					if sub_pass == 2 {
						actor.Output["CritIgniteDotMulti"] = 1 + skillModList.Sum(mod.TypeBase, dotCfg, "DotMultiplier", "FireDotMultiplier") / 100
						sourceCritDmg = (totalMin + (totalMax - totalMin) / m_pow(2, 1 / (igniteStacks + 1))) * actor.Output["CritIgniteDotMulti"]
					} else {
						actor.Output["IgniteDotMulti"] = 1 + skillModList.Sum(mod.TypeBase, dotCfg, "DotMultiplier", "FireDotMultiplier") / 100
						sourceHitDmg = (totalMin + (totalMax - totalMin) / m_pow(2, 1 / (igniteStacks + 1))) * actor.Output["IgniteDotMulti"]
					}
					actor.Output["IgniteTotalMin"] = totalMin
					actor.Output["IgniteTotalMax"] = totalMax
				}
				if globalBreakdown {
					if sourceHitDmg == sourceCritDmg {
						globalBreakdown.IgniteDPS = {
							fmt.Sprintf(colorCodes.CUSTOM+"NOTE: Calculation uses new Weighted Avg Ailment formula"),
							fmt.Sprintf(""),
							fmt.Sprintf("Dmg Derivation:"),
							fmt.Sprintf("(%.2f + (%.2f - %.2f) ^8(min combined sources + (max combined sources - min combined sources)", actor.Output["IgniteTotalMin"], actor.Output["IgniteTotalMax"], actor.Output["IgniteTotalMin"]),
							fmt.Sprintf("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", igniteStacks),
							fmt.Sprintf("* %.2f ^8(Ignite DoT Multi)", actor.Output["IgniteDotMulti"]),
							fmt.Sprintf("= %.2f", sourceHitDmg),
						}
					} else {
						globalBreakdown.IgniteDPS = {
							fmt.Sprintf(colorCodes.CUSTOM+"NOTE: Calculation uses new Weighted Avg Ailment formula"),
							fmt.Sprintf(""),
							fmt.Sprintf("Non-Crit Dmg Derivation:"),
							fmt.Sprintf("(%.2f + (%.2f - %.2f) ^8(min combined sources + (max combined sources - min combined sources)", actor.Output["IgniteTotalMin"], actor.Output["IgniteTotalMax"], actor.Output["IgniteTotalMin"]),
							fmt.Sprintf("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", igniteStacks),
							fmt.Sprintf("* %.2f ^8(Ignite DoT Multi for Non-Crit)", actor.Output["IgniteDotMulti"]),
							fmt.Sprintf("= %.2f", sourceHitDmg),
							fmt.Sprintf(""),
							fmt.Sprintf("Crit Dmg Derivation:"),
							fmt.Sprintf("(%.2f + (%.2f - %.2f) ^8(min combined sources + (max combined sources - min combined sources)", actor.Output["IgniteTotalMin"], actor.Output["IgniteTotalMax"], actor.Output["IgniteTotalMin"]),
							fmt.Sprintf("/ 2^(1 / (%.2f + 1))) ^8(/ 2^(1 / (stack potential + 1)))", igniteStacks),
							fmt.Sprintf("* %.2f ^8(Ignite DoT Multi for Crit)", actor.Output["CritIgniteDotMulti"]),
							fmt.Sprintf("= %.2f", sourceCritDmg),
						}
					}
				}
				baseVal := calcAilmentDamage("Ignite", sourceHitDmg, sourceCritDmg) * data.IgnitePercentBase * actor.Output["FistOfWarAilmentEffect"] * globalOutput.AilmentWarcryEffect
				if baseVal > 0 {
					skillFlags.ignite = true
					effMult := 1
					if env.mode_effective {
						if skillModList.Flag(cfg, "IgniteToChaos") {
							resist := min(enemyDB.Sum(mod.TypeBase, nil, "ChaosResist") * calclib.Mod(enemyDB, nil, "ChaosResist"), data.EnemyMaxResist)
							takenInc := enemyDB.Sum(mod.TypeIncrease, dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
							takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
							effMult = (1 - resist / 100) * (1 + takenInc / 100) * takenMore
							globalOutput["IgniteEffMult"] = effMult
							if breakdown and effMult != 1 {
								globalBreakdown.IgniteEffMult = breakdown.effMult("Chaos", resist, 0, takenInc, effMult, takenMore)
							}
						} else {
							resist := min(enemyDB.Sum(mod.TypeBase, nil, "FireResist", "ElementalResist") * calclib.Mod(enemyDB, nil, "FireResist", "ElementalResist"), data.EnemyMaxResist)
							takenInc := enemyDB.Sum(mod.TypeIncrease, dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
							takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
							effMult = (1 - resist / 100) * (1 + takenInc / 100) * takenMore
							globalOutput["IgniteEffMult"] = effMult
							if breakdown and effMult != 1 {
								breakdown.IgniteEffMult = breakdown.effMult("Fire", resist, 0, takenInc, effMult, takenMore)
							}
						}
					}
					effectMod := calclib.Mod(skillModList, dotCfg, "AilmentEffect")
					igniteStacks = 1
					if not skillData.triggeredOnDeath {
						igniteStacks = min(maxStacks, (actor.Output["HitChance"] / 100) * globalOutput.IgniteDuration / actor.Output["Time"])
					}
					actor.Output["IgniteDPS"] = baseVal * effectMod * rateMod * effMult * igniteStacks
					globalOutput.IgniteDamage = actor.Output["IgniteDPS"] * globalOutput.IgniteDuration
					if skillFlags.igniteCanStack {
						actor.Output["IgniteDamage"] = actor.Output["IgniteDPS"] * globalOutput.IgniteDuration
						actor.Output["IgniteStacksMax"] = maxStacks
						actor.Output["TotalIgniteDPS"] = actor.Output["IgniteDPS"]
					}

					if breakdown != nil {
						breakdown.AddLine("IgniteDPS", "x 0.9 ^8(ignite deals 90% per second)")
						breakdown.AddLine("IgniteDPS", fmt.Sprintf("= %.1f", baseVal, 1))
						breakdown.multiChain(breakdown.IgniteDPS, {
							label = "Ignite DPS:",
							base = fmt.Sprintf("%.1f ^8(total damage per second)", baseVal),
							{ "%.2f ^8(ailment effect modifier)", effectMod },
							{ "%.2f ^8(burn rate modifier)", rateMod },
							{ "%.3f ^8(effective DPS modifier)", effMult },
							{ "%d ^8(ignite stacks)", actor.Output["IgniteStacksMax"] },
							total = fmt.Sprintf("= %.1f ^8per second", actor.Output["IgniteDPS"]),
						})
						if actor.Output["CritIgniteDotMulti"] and (actor.Output["CritIgniteDotMulti"] != actor.Output["IgniteDotMulti"]) {
							chanceFromHit := actor.Output["IgniteChanceOnHit"] / 100 * (1 - globalOutput.CritChance / 100)
							chanceFromCrit := actor.Output["IgniteChanceOnCrit"] / 100 * actor.Output["CritChance"] / 100
							totalFromHit := chanceFromHit / (chanceFromHit + chanceFromCrit)
							totalFromCrit := chanceFromCrit / (chanceFromHit + chanceFromCrit)
							breakdown.IgniteDotMulti = breakdown.critDot(actor.Output["IgniteDotMulti"], actor.Output["CritIgniteDotMulti"], totalFromHit, totalFromCrit)
							actor.Output["IgniteDotMulti"] = (actor.Output["IgniteDotMulti"] * totalFromHit) + (actor.Output["CritIgniteDotMulti"] * totalFromCrit)
						}
						if skillFlags.igniteCanStack {
							breakdown.IgniteDamage = { }
							if isAttack {
								breakdown.AddLine("IgniteDamage", pass.label+":")
							}
							breakdown.AddLine("IgniteDamage", fmt.Sprintf("%.1f ^8(damage per second)", actor.Output["IgniteDPS"]))
							breakdown.AddLine("IgniteDamage", fmt.Sprintf("x %.2fs ^8(ignite duration)", globalOutput.IgniteDuration))
							breakdown.AddLine("IgniteDamage", fmt.Sprintf("= %.1f ^8damage per ignite stack", actor.Output["IgniteDamage"]))
						}
						if globalOutput.IgniteDuration != data.IgniteDurationBase {
							globalBreakdown.IgniteDuration = {
								fmt.Sprintf("%.2fs ^8(base duration)", durationBase)
							}
							if durationMod != 1 {
								t_insert(globalBreakdown.IgniteDuration, fmt.Sprintf("x %.2f ^8(duration modifier)", durationMod))
							}
							if rateMod != 1 {
								t_insert(globalBreakdown.IgniteDuration, fmt.Sprintf("/ %.2f ^8(burn rate modifier)", rateMod))
							}
							if debuffDurationMult != 1 {
								t_insert(globalBreakdown.IgniteDuration, fmt.Sprintf("/ %.2f ^8(debuff expires slower/faster)", 1 / debuffDurationMult))
							}
							t_insert(globalBreakdown.IgniteDuration, fmt.Sprintf("= %.2fs", globalOutput.IgniteDuration))
						}
					}
				}
			}
		*/
		/*
			TODO Calculate non-damaging ailments effect and duration modifiers
			isBoss := env.configInput["enemyIsBoss"] != "None"
			enemyBaseLife := data.monsterLifeTable[env.enemyLevel] * enemyDB.More(nil, "Life")
			enemyMapLifeMult := 1
			enemyMapAilmentMult := 1
			if env.enemyLevel >= 66 {
				enemyMapLifeMult = isBoss and data.mapLevelBossLifeMult[env.enemyLevel] or data.mapLevelLifeMult[env.enemyLevel]
				enemyMapAilmentMult = isBoss and data.mapLevelBossAilmentMult[env.enemyLevel] or enemyMapAilmentMult
			}
			enemyTypeMult := isBoss and 7.68 or 1
			enemyThreshold := enemyBaseLife * enemyTypeMult * enemyMapLifeMult * enemyMapAilmentMult * enemyDB.More(nil, "AilmentThreshold")

			bonechill := actor.Output["BonechillEffect"] or enemyDB.Sum(mod.TypeBase, nil, "DesiredBonechillEffect")
			ailments := {
				["Chill"] = {
					effList = { 10, 20 },
					effect = function(damage, effectMod) return 50 * ((damage / enemyThreshold) ^ 0.4) * effectMod end,
					thresh = function(damage, value, effectMod) return damage * ((50 * effectMod / value) ^ 2.5) end,
					ramping = bonechill > 0,
				},
				["Shock"] = {
					effList = { 10, 20, 40 },
					effect = function(damage, effectMod) return 50 * ((damage / enemyThreshold) ^ 0.4) * effectMod end,
					thresh = function(damage, value, effectMod) return damage * ((50 * effectMod / value) ^ 2.5) end,
					ramping = true,
				},
				["Scorch"] = {
					effList = { 5, 10, 20 },
					effect = function(damage, effectMod) return 50 * ((damage / enemyThreshold) ^ 0.4) * effectMod end,
					thresh = function(damage, value, effectMod) return damage * ((50 * effectMod / value) ^ 2.5) end,
					ramping = true,
				},
				["Brittle"] = {
					effList = { 5, 10 },
					effect = function(damage, effectMod) return 25 * ((damage / enemyThreshold) ^ 0.4) * effectMod end,
					thresh = function(damage, value, effectMod) return damage * ((25 * effectMod / value) ^ 2.5) end,
					ramping = true,
				},
				["Sap"] = {
					effList = { 5, 10 },
					effect = function(damage, effectMod) return (100 / 3) * ((damage / enemyThreshold) ^ 0.4) * effectMod end,
					thresh = function(damage, value, effectMod) return damage * ((100 / 3 * effectMod / value) ^ 2.5) end,
					ramping = false,
				},
			}
			if activeSkill.skillTypes[SkillType.ChillingArea] or activeSkill.skillTypes[SkillType.NonHitChill] {
				skillFlags.chill = true
				actor.Output["ChillEffectMod"] = skillModList.Sum(mod.TypeIncrease, cfg, "EnemyChillEffect")
				actor.Output["ChillDurationMod"] = 1 + skillModList.Sum(mod.TypeIncrease, cfg, "EnemyChillDuration") / 100
				actor.Output["ChillSourceEffect"] = min(skillModList:Override(nil, "ChillMax") or ailmentData.Chill.max, math.Floor(ailmentData.Chill.default * (1 + actor.Output["ChillEffectMod"] / 100)))
				if breakdown != nil {
					breakdown.DotChill = { }
					breakdown.multiChain(breakdown.DotChill, {
						label = fmt.Sprintf("Effect of Chill: ^8(capped at %d%%)", skillModList:Override(nil, "ChillMax") or ailmentData.Chill.max),
						base = fmt.Sprintf("%d%% ^8(base)", ailmentData.Chill.default),
						{ "%.2f ^8(increased effect of chill)", 1 + actor.Output["ChillEffectMod"] / 100},
						total = fmt.Sprintf("= %.0f%%", actor.Output["ChillSourceEffect"])
					})
				}
			}
			if (actor.Output["FreezeChanceOnHit"] + actor.Output["FreezeChanceOnCrit"]) > 0 {
				if globalBreakdown {
					globalBreakdown.FreezeDurationMod = {
						fmt.Sprintf("Ailment mode: %s ^8(can be changed in the Configuration tab)", igniteMode == "CRIT" and "Crits Only" or "Average Damage")
					}
				}
				baseVal := calcAilmentDamage("Freeze", calcAverageSourceDamage("Freeze")) * skillModList.More(cfg, "FreezeAsThoughDealing")
				if baseVal > 0 {
					skillFlags.freeze = true
					skillFlags.chill = true
					actor.Output["FreezeDurationMod"] = 1 + skillModList.Sum(mod.TypeIncrease, cfg, "EnemyFreezeDuration") / 100 + enemyDB.Sum(mod.TypeIncrease, nil, "SelfFreezeDuration") / 100
					if breakdown != nil {
						breakdown.AddLine("FreezeDPS", fmt.Sprintf("For freeze to apply for the minimum of 0.3 seconds, target must have no more than %.0f Ailment Threshold.", baseVal * 20 * actor.Output["FreezeDurationMod"]))
						breakdown.AddLine("FreezeDPS", fmt.Sprintf("^8(Ailment Threshold is about equal to Life except on bosses where it is about half of their life)"))
					}
				}
			}
			for ailment, val := range (ailments) {
				if (output[ailment+"ChanceOnHit"] + output[ailment+"ChanceOnCrit"]) > 0 {
					if globalBreakdown {
						globalBreakdown[ailment+"EffectMod"] = {
							fmt.Sprintf("Ailment mode: %s ^8(can be changed in the Configuration tab)", igniteMode == "CRIT" and "Crits Only" or "Average Damage")
						}
					}
					damage := calcAilmentDamage(ailment, calcAverageSourceDamage(ailment)) * skillModList.More(cfg, ailment+"AsThoughDealing")
					if damage > 0 {
						skillFlags[string.lower(ailment)] = true
						incDur := skillModList.Sum(mod.TypeIncrease, cfg, "Enemy"+ailment+"Duration") + enemyDB.Sum(mod.TypeIncrease, nil, "Self"+ailment+"Duration")
						moreDur := skillModList.More(cfg, "Enemy"+ailment+"Duration") * enemyDB.More(nil, "Self"+ailment+"Duration")
						output[ailment+"Duration"] = ailmentData[ailment].duration * (1 + incDur / 100) * moreDur * debuffDurationMult
						output[ailment+"EffectMod"] = calclib.Mod(skillModList, cfg, "Enemy"+ailment+"Effect")
						if breakdown != nil {
							maximum := skillModList:Override(nil, ailment+"Max") or ailmentData[ailment].max
							current := max(min(ailment == "Chill" and bonechill or globalOutput["Current"+ailment] or 0, maximum), 0)
							desired := max(min(enemyDB.Sum(mod.TypeBase, nil, "Desired"+ailment+"Val"), maximum), 0)
							if ailmentData[ailment].min != 0 {
								t_insert(val.effList, ailmentData[ailment].min)
							}
							if enemyThreshold > 0 {
								t_insert(val.effList, val.effect(damage, output[ailment+"EffectMod"]))
							}
							if not isValueInArray(val.effList, maximum) {
								t_insert(val.effList, maximum)
							}
							if current > 0 and not isValueInArray(val.effList, current) {
								t_insert(val.effList, current)
							}
							if desired > 0 and not isValueInArray(val.effList, desired) and current == 0 {
								t_insert(val.effList, desired)
							}
							breakdown[ailment+"DPS"].label = "Resulting ailment effect"+((current > 0 and val.ramping) and fmt.Sprintf(" ^8(with a ^7%s%% ^8%s on the enemy)^7", current, ailment) or "")
							breakdown[ailment+"DPS"].footer = fmt.Sprintf("^8(ailment threshold is about equal to life, except on bosses that have specific ailement thresholds)\n(the above table shows that when the enemy has X ailment threshold, you ^8%s for Y)", ailment:lower())
							breakdown[ailment+"DPS"].rowList = { }
							breakdown[ailment+"DPS"].colList = {
								{ label = "Ailment Threshold", key = "thresh" },
								{ label = ailment+" Effect", key = "effect" },
							}
							table.sort(val.effList)
							for _, value := range (val.effList) {
								thresh := val.thresh(damage, value, output[ailment+"EffectMod"])
								decCheck := value / math.Floor(value)
								precision := ailmentData[ailment].precision
								value = math.Floor(value * (10 ^ precision)) / (10 ^ precision)
								valueFormat := "%."+tostring(precision)+"f%%"
								threshString := fmt.Sprintf("%d", thresh)+(math.Floor(thresh + 0.5) == math.Floor(enemyThreshold + 0.5) and fmt.Sprintf(" ^8(%s)", env.configInput.enemyIsBoss) or "")
								labels := { }
								if decCheck == 1 and value != 0 {
									if ailment == "Chill" and value == bonechill {
										t_insert(labels, "bonechill")
									} else if value == current {
										t_insert(labels, "current")
									}
									if value == desired {
										t_insert(labels, "desired")
									}
									if value == maximum {
										t_insert(labels, "maximum")
									}
									if value == ailmentData[ailment].min {
										t_insert(labels, "minimum")
									}
								}
								t_insert(breakdown[ailment+"DPS"].rowList, {
									effect = fmt.Sprintf(valueFormat, value)+(next(labels) != nil and " ^8("+table.concat(labels, ", ")+")" or ""),
									thresh = threshString,
								})
							}
						}
						if breakdown and output[ailment+"Duration"] != ailmentData[ailment].duration {
							breakdown[ailment+"Duration"] = { }
							if isAttack {
								breakdown.AddLine(ailment+"Duration", pass.label+":")
							}
							breakdown.AddLine(ailment+"Duration", fmt.Sprintf("%.2fs ^8(base duration)", ailmentData[ailment].duration))
							if incDur != 0 {
								breakdown.AddLine(ailment+"Duration", fmt.Sprintf("x %.2f ^8(increased/reduced duration)", 1 + incDur / 100))
							}
							if moreDur != 1 {
								breakdown.AddLine(ailment+"Duration", fmt.Sprintf("x %.2f ^8(more/less duration)", moreDur))
							}
							if debuffDurationMult != 1 {
								breakdown.AddLine(ailment+"Duration", fmt.Sprintf("/ %.2f ^8(debuff expires slower/faster)", 1 / debuffDurationMult))
							}
							breakdown.AddLine(ailment+"Duration", fmt.Sprintf("= %.2fs", output[ailment+"Duration"]))
						}
					}
				}
			}
		*/
		/*
			TODO Calculate knockback chance/distance
			actor.Output["KnockbackChance"] = min(100, actor.Output["KnockbackChanceOnHit"] * (1 - actor.Output["CritChance"] / 100) + actor.Output["KnockbackChanceOnCrit"] * actor.Output["CritChance"] / 100 + enemyDB.Sum(mod.TypeBase, nil, "SelfKnockbackChance"))
			if actor.Output["KnockbackChance"] > 0 {
				actor.Output["KnockbackDistance"] = round(4 * calclib.Mod(skillModList, cfg, "EnemyKnockbackDistance"))
				if breakdown != nil {
					breakdown.KnockbackDistance = {
						radius = actor.Output["KnockbackDistance"],
					}
				}
			}
		*/
		/*
			TODO Calculate enemy stun modifiers
			enemyStunThresholdRed := -skillModList.Sum(mod.TypeIncrease, cfg, "EnemyStunThreshold")
			if enemyStunThresholdRed > 75 {
				actor.Output["EnemyStunThresholdMod"] = 1 - (75 + (enemyStunThresholdRed - 75) * 25 / (enemyStunThresholdRed - 50)) / 100
			} else {
				actor.Output["EnemyStunThresholdMod"] = 1 - enemyStunThresholdRed / 100
			}
			base := skillData.baseStunDuration or 0.35
			incDur := skillModList.Sum(mod.TypeIncrease, cfg, "EnemyStunDuration")
			incRecov := enemyDB.Sum(mod.TypeIncrease, nil, "StunRecovery")
			actor.Output["EnemyStunDuration"] = base * (1 + incDur / 100) / (1 + incRecov / 100)
			if breakdown != nil {
				if actor.Output["EnemyStunDuration"] != base {
					breakdown.EnemyStunDuration = {
						fmt.Sprintf("%.2fs ^8(base duration)", base),
					}
					if incDur != 0 {
						breakdown.AddLine("EnemyStunDuration", fmt.Sprintf("x %.2f ^8(increased/reduced stun duration)", 1 + incDur/100))
					}
					if incRecov != 0 {
						breakdown.AddLine("EnemyStunDuration", fmt.Sprintf("/ %.2f ^8(increased/reduced enemy stun recovery)", 1 + incRecov/100))
					}
					breakdown.AddLine("EnemyStunDuration", fmt.Sprintf("= %.2fs", actor.Output["EnemyStunDuration"]))
				}
			}
		*/
		/*
			TODO Calculate impale chance and modifiers
			if canDeal.Physical and actor.Output["ImpaleChance"] > 0 {
				skillFlags.impale = true
				impaleChance := min(actor.Output["ImpaleChance"]/100, 1)
				maxStacks := skillModList.Sum(mod.TypeBase, cfg, "ImpaleStacksMax") // magic number: base stacks duration
				configStacks := enemyDB.Sum(mod.TypeBase, cfg, "Multiplier:ImpaleStacks")
				impaleStacks := min(maxStacks, configStacks)

				baseStoredDamage := data.ImpaleStoredDamageBase
				storedExpectedDamageIncOnBleed := skillModList.Sum(mod.TypeIncrease, cfg, "ImpaleEffectOnBleed")*skillModList.Sum(mod.TypeBase, cfg, "BleedChance")/100
				storedExpectedDamageInc := (skillModList.Sum(mod.TypeIncrease, cfg, "ImpaleEffect") + storedExpectedDamageIncOnBleed)/100
				storedExpectedDamageMore := round(skillModList.More(cfg, "ImpaleEffect"), 2)
				storedExpectedDamageModifier := (1 + storedExpectedDamageInc) * storedExpectedDamageMore
				impaleStoredDamage := baseStoredDamage * storedExpectedDamageModifier
				impaleHitDamageMod := impaleStoredDamage * impaleStacks  // Source: https://www.reddit.com/r/pathofexile/comments/chgqqt/impale_and_armor_interaction/

				enemyArmour := max(calclib.Val(enemyDB, "Armour"), 0)
				impaleArmourReduction := calcs.armourReductionF(enemyArmour, impaleHitDamageMod * actor.Output["impaleStoredHitAvg"])
				impaleResist := min(max(0, enemyDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction") + skillModList.Sum(mod.TypeBase, cfg, "EnemyImpalePhysicalDamageReduction") + impaleArmourReduction), data.DamageReductionCap)

				impaleDMGModifier := impaleHitDamageMod * (1 - impaleResist / 100) * impaleChance

				globalOutput.ImpaleStacksMax = maxStacks
				globalOutput.ImpaleStacks = impaleStacks
				//ImpaleStoredDamage should be named ImpaleEffect or similar
				//Using the variable name ImpaleEffect breaks the calculations sidebar (?!)
				actor.Output["ImpaleStoredDamage"] = impaleStoredDamage * 100
				actor.Output["ImpaleModifier"] = 1 + impaleDMGModifier

				if breakdown != nil {
					breakdown.ImpaleStoredDamage = {}
					breakdown.AddLine("ImpaleStoredDamage", "10% ^8(base value)")
					breakdown.AddLine("ImpaleStoredDamage", fmt.Sprintf("x %.2f ^8(increased effectiveness)", storedExpectedDamageModifier))
					breakdown.AddLine("ImpaleStoredDamage", fmt.Sprintf("= %.1f%%", actor.Output["ImpaleStoredDamage"]))

					breakdown.ImpaleModifier = {}
					breakdown.AddLine("ImpaleModifier", fmt.Sprintf("%d ^8(number of stacks, can be overridden in the Configuration tab)", impaleStacks))
					breakdown.AddLine("ImpaleModifier", fmt.Sprintf("x %.3f ^8(stored damage)", impaleStoredDamage))
					breakdown.AddLine("ImpaleModifier", fmt.Sprintf("x %.2f ^8(impale chance)", impaleChance))
					breakdown.AddLine("ImpaleModifier", fmt.Sprintf("x %.2f ^8(impale enemy physical damage reduction)", (1 - impaleResist / 100)))
					breakdown.AddLine("ImpaleModifier", fmt.Sprintf("= %.3f ^8(impale damage multiplier)", impaleDMGModifier))
				}
			}
		*/
	}

	// Combine secondary effect stats
	if isAttack {
		combineStat("BleedChance", "AVERAGE")
		combineStat("BleedDPS", "CHANCE_AILMENT", "BleedChance")
		combineStat("PoisonChance", "AVERAGE")
		combineStat("PoisonDPS", "CHANCE", "PoisonChance")
		combineStat("TotalPoisonDPS", "DPS")
		combineStat("PoisonDamage", "CHANCE", "PoisonChance")

		if skillData.ShowAverage {
			combineStat("TotalPoisonAverageDamage", "DPS")
		} else {
			combineStat("TotalPoisonStacks", "DPS")
		}

		combineStat("IgniteChance", "AVERAGE")
		combineStat("IgniteDPS", "CHANCE_AILMENT", "IgniteChance")

		if skillFlags[SkillFlagIgniteCanStack] {
			combineStat("IgniteDamage", "CHANCE", "IgniteChance")

			if skillData.ShowAverage {
				combineStat("TotalIgniteAverageDamage", "DPS")
				combineStat("IgniteStacksMax", "DPS")
				combineStat("TotalIgniteDPS", "DPS")
			} else {
				combineStat("IgniteStacksMax", "DPS")
				combineStat("TotalIgniteDPS", "DPS")
			}
		}

		combineStat("ChillEffectMod", "AVERAGE")
		combineStat("ChillDuration", "AVERAGE")
		combineStat("ShockChance", "AVERAGE")
		combineStat("ShockDuration", "AVERAGE")
		combineStat("ShockEffectMod", "AVERAGE")
		combineStat("FreezeChance", "AVERAGE")
		combineStat("FreezeDurationMod", "AVERAGE")
		combineStat("ScorchChance", "AVERAGE")
		combineStat("ScorchEffectMod", "AVERAGE")
		combineStat("ScorchDuration", "AVERAGE")
		combineStat("BrittleChance", "AVERAGE")
		combineStat("BrittleEffectMod", "AVERAGE")
		combineStat("BrittleDuration", "AVERAGE")
		combineStat("SapChance", "AVERAGE")
		combineStat("SapEffectMod", "AVERAGE")
		combineStat("SapDuration", "AVERAGE")
		combineStat("ImpaleChance", "AVERAGE")
		combineStat("ImpaleStoredDamage", "AVERAGE")
		combineStat("ImpaleModifier", "CHANCE", "ImpaleChance")
	}

	/*
		TODO // Calculate DPS for Essence of Delirium's Decay effect
		if skillFlags.hit and skillData.decay and canDeal.Chaos {
			// Calculate DPS for Essence of Delirium's Decay effect
			skillFlags.decay = true
			activeSkill.decayCfg = {
				skillName = skillCfg.skillName,
				skillPart = skillCfg.skillPart,
				skillTypes = skillCfg.skillTypes,
				slotName = skillCfg.slotName,
				flags = ModFlag.Dot,
				keywordFlags = bor(band(skillCfg.keywordFlags, bnot(KeywordFlag.Hit)), KeywordFlag.ChaosDot),
			}
			dotCfg := activeSkill.decayCfg
			effMult := 1
			if env.mode_effective {
				resist := min(enemyDB.Sum(mod.TypeBase, nil, "ChaosResist") * calclib.Mod(enemyDB, nil, "ChaosResist"), data.EnemyMaxResist)
				takenInc := enemyDB.Sum(mod.TypeIncrease, nil, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
				takenMore := enemyDB.More(nil, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
				effMult = (1 - resist / 100) * (1 + takenInc / 100) * takenMore
				actor.Output["DecayEffMult"] = effMult
				if breakdown and effMult != 1 {
					breakdown.DecayEffMult = breakdown.effMult("Chaos", resist, 0, takenInc, effMult, takenMore)
				}
			}
			inc := skillModList.Sum(mod.TypeIncrease, dotCfg, "Damage", "ChaosDamage")
			more := round(skillModList.More(dotCfg, "Damage", "ChaosDamage"), 2)
			mult := skillModList.Sum(mod.TypeBase, dotTypeCfg, "DotMultiplier", "ChaosDotMultiplier")
			actor.Output["DecayDPS"] = skillData.decay * (1 + inc/100) * more * (1 + mult/100) * effMult
			actor.Output["DecayDuration"] = 8 * debuffDurationMult
			if breakdown != nil {
				breakdown.DecayDPS = { }
				breakdown.dot(breakdown.DecayDPS, skillData.decay, inc, more, mult, nil, nil, effMult, actor.Output["DecayDPS"])
				if actor.Output["DecayDuration"] != 8 {
					breakdown.DecayDuration = {
						fmt.Sprintf("%.2fs ^8(base duration)", 8)
					}
					if debuffDurationMult != 1 {
						breakdown.AddLine("DecayDuration", fmt.Sprintf("/ %.2f ^8(debuff expires slower/faster)", 1 / debuffDurationMult))
					}
					breakdown.AddLine("DecayDuration", fmt.Sprintf("= %.2fs", actor.Output["DecayDuration"]))
				}
			}
		}
	*/
	/*
		TODO // Calculate skill DOT components
		dotCfg := {
			skillName = skillCfg.skillName,
			skillPart = skillCfg.skillPart,
			skillTypes = skillCfg.skillTypes,
			slotName = skillCfg.slotName,
			flags = bor(ModFlag.Dot, skillCfg.flags),
			keywordFlags = band(skillCfg.keywordFlags, bnot(KeywordFlag.Hit)),
		}
		if bor(dotCfg.flags, ModFlag.Area) == dotCfg.flags and not skillData.dotIsArea {
			dotCfg.flags = band(dotCfg.flags, bnot(ModFlag.Area))
		}
		if bor(dotCfg.flags, ModFlag.Projectile) == dotCfg.flags and not skillData.dotIsProjectile {
			dotCfg.flags = band(dotCfg.flags, bnot(ModFlag.Projectile))
		}
		if bor(dotCfg.flags, ModFlag.Spell) == dotCfg.flags and not skillData.dotIsSpell {
			dotCfg.flags = band(dotCfg.flags, bnot(ModFlag.Spell))
		}
		if bor(dotCfg.flags, ModFlag.Attack) == dotCfg.flags and not skillData.dotIsAttack {
			dotCfg.flags = band(dotCfg.flags, bnot(ModFlag.Attack))
		}
		if bor(dotCfg.flags, ModFlag.Hit) == dotCfg.flags and not skillData.dotIsHit {
			dotCfg.flags = band(dotCfg.flags, bnot(ModFlag.Hit))
		}
	*/
	/*
		TODO // spell_damage_modifiers_apply_to_skill_dot does not apply to enemy damage taken
		dotTakenCfg := copyTable(dotCfg, true)
		if (skillData.dotIsSpell) {
			dotTakenCfg.flags = band(dotTakenCfg.flags, bnot(ModFlag.Spell))
		}

		activeSkill.dotCfg = dotCfg
		actor.Output["TotalDotInstance"] = 0

		runSkillFunc("preDotFunc")

		for _, damageType := range data.DamageType("").Values() {
			dotTypeCfg := copyTable(dotCfg, true)
			dotTypeCfg.keywordFlags = bor(dotTypeCfg.keywordFlags, KeywordFlag[damageType+"Dot"])
			activeSkill["dot"+damageType+"Cfg"] = dotTypeCfg
			local baseVal
			if canDeal[damageType] {
				baseVal = skillData[damageType+"Dot"] or 0
			} else {
				baseVal = 0
			}
			if baseVal > 0 or (actor.Output[damageType+"Dot"] or 0) > 0 {
				skillFlags.dot = true
				effMult := 1
				if env.mode_effective {
					resist := 0
					takenInc := enemyDB.Sum(mod.TypeIncrease, dotTakenCfg, "DamageTaken", "DamageTakenOverTime", damageType+"DamageTaken", damageType+"DamageTakenOverTime")
					takenMore := enemyDB.More(dotTakenCfg, "DamageTaken", "DamageTakenOverTime", damageType+"DamageTaken", damageType+"DamageTakenOverTime")
					if damageType == "Physical" {
						resist = max(0, min(enemyDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction"), data.DamageReductionCap))
					} else {
						if env.modDB.Flag(nil, "Enemy"+damageType+"ResistEqualToYours") {
							resist = env.player.output[damageType+"Resist"]
						} else {
							resist = enemyDB.Sum(mod.TypeBase, nil, damageType+"Resist")
							if isElemental[damageType] {
								base := resist + enemyDB.Sum(mod.TypeBase, dotTypeCfg, "ElementalResist")
								resist = base * calclib.Mod(enemyDB, nil, damageType+"Resist")
								takenInc = takenInc + enemyDB.Sum(mod.TypeIncrease, dotTypeCfg, "ElementalDamageTaken")
							}
						}
						resist = min(resist, data.EnemyMaxResist)
					}
					effMult = (1 - resist / 100) * (1 + takenInc / 100) * takenMore
					actor.Output[damageType+"DotEffMult"] = effMult
					if breakdown and effMult != 1 {
						breakdown[damageType+"DotEffMult"] = breakdown.effMult(damageType, resist, 0, takenInc, effMult, takenMore)
					}
				}
				inc := skillModList.Sum(mod.TypeIncrease, dotTypeCfg, "Damage", damageType+"Damage", isElemental[damageType] and "ElementalDamage" or nil)
				more := round(skillModList.More(dotTypeCfg, "Damage", damageType+"Damage", isElemental[damageType] and "ElementalDamage" or nil), 2)
				mult := skillModList.Sum(mod.TypeBase, dotTypeCfg, "DotMultiplier", damageType+"DotMultiplier")
				aura := activeSkill.skillTypes[SkillType.Aura] and not activeSkill.skillTypes[SkillType.RemoteMined] and calclib.Mod(skillModList, dotTypeCfg, "AuraEffect")
				total := baseVal * (1 + inc/100) * more * (1 + mult/100) * (aura or 1) * effMult
				if actor.Output[damageType+"Dot"] == 0 or actor.Output[damageType+"Dot"] == nil {
					actor.Output[damageType+"Dot"] = total
					actor.Output["TotalDotInstance"] = actor.Output["TotalDotInstance"] + total
				} else {
					actor.Output["TotalDotInstance"] = actor.Output["TotalDotInstance"] + total + (actor.Output[damageType+"Dot"] or 0)
				}
				if breakdown != nil {
					breakdown[damageType+"Dot"] = { }
					breakdown.dot(breakdown[damageType+"Dot"], baseVal, inc, more, mult, nil, aura, effMult, total)
				}
			}
		}
		if skillModList.Flag(nil, "DotCanStack") {
			skillFlags.DotCanStack = true
			speed := actor.Output["Speed"]
			// Check if skill is being triggered via Mine (e.g., Blastchain Mine Support) or Trap
			// if "yes", you cannot use actor.Output["Speed"] but rather should use actor.Output["MineLayingSpeed"] or actor.Output["TrapThrowingSpeed"]
			if band(dotCfg.keywordFlags, KeywordFlag.Mine) != 0 {
				speed = actor.Output["MineLayingSpeed"]
			} else if band(dotCfg.keywordFlags, KeywordFlag.Trap) != 0 {
				speed = actor.Output["TrapThrowingSpeed"]
			}
			actor.Output["TotalDot"] = actor.Output["TotalDotInstance"] * speed * actor.Output["Duration"] * (skillData.dpsMultiplier or 1) * quantityMultiplier
			if breakdown != nil {
				breakdown.TotalDot = {
					fmt.Sprintf("%.1f ^8(Damage per Instance)", actor.Output["TotalDotInstance"]),
					fmt.Sprintf("x %.2f ^8(hits per second)", speed),
					fmt.Sprintf("x %.2f ^8(skill duration)", actor.Output["Duration"]),
				}
				if skillData.dpsMultiplier {
					breakdown.AddLine("TotalDot", fmt.Sprintf("x %g ^8(DPS multiplier for this skill)", skillData.dpsMultiplier))
				}
				if quantityMultiplier > 1 {
					breakdown.AddLine("TotalDot", fmt.Sprintf("x %g ^8(quantity multiplier for this skill)", quantityMultiplier))
				}
				breakdown.AddLine("TotalDot", fmt.Sprintf("= %.1f", actor.Output["TotalDot"]))
			}
		} else {
			actor.Output["TotalDot"] = actor.Output["TotalDotInstance"]
		}
	*/
	/*
		TODO // The Saviour
		if activeSkill.activeEffect.grantedEffect.name == "Reflection" {
			usedSkill := nil
			usedSkillBestDps := 0
			calcMode := env.mode == "CALCS" and "CALCS" or "MAIN"
			for _, triggerSkill := range (actor.activeSkillList) {
				if triggerSkill != activeSkill and triggerSkill.skillTypes[SkillType.Attack] and band(triggerSkill.skillCfg.flags, bor(ModFlag.Sword, ModFlag.Weapon1H)) == bor(ModFlag.Sword, ModFlag.Weapon1H) {
					// Grab a fully-processed by calcs.perform() version of the skill that Mirage Warrior(s) will use
					uuid := cacheSkillUUID(triggerSkill)
					if not GlobalCache.cachedData[calcMode][uuid] {
						calcs.buildActiveSkill(env, calcMode, triggerSkill)
						env.dontCache = true
					}
					// We found a skill and it can crit
					if GlobalCache.cachedData[calcMode][uuid] and GlobalCache.cachedData[calcMode][uuid].CritChance and GlobalCache.cachedData[calcMode][uuid].CritChance > 0 {
						if not usedSkill {
							usedSkill = GlobalCache.cachedData[calcMode][uuid].ActiveSkill
							usedSkillBestDps = GlobalCache.cachedData[calcMode][uuid].TotalDPS
						} else {
							if GlobalCache.cachedData[calcMode][uuid].TotalDPS > usedSkillBestDps {
								usedSkill = GlobalCache.cachedData[calcMode][uuid].ActiveSkill
								usedSkillBestDps = GlobalCache.cachedData[calcMode][uuid].TotalDPS
							}
						}
					}
				}
			}

			if usedSkill {
				moreDamage := activeSkill.skillModList.Sum(mod.TypeBase, activeSkill.skillCfg, "SaviourMirageWarriorLessDamage")
				maxMirageWarriors := activeSkill.skillModList.Sum(mod.TypeBase, activeSkill.skillCfg, "SaviourMirageWarriorMaxCount")
				newSkill, newEnv := calcs.copyActiveSkill(env, calcMode, usedSkill)

				// Add new modifiers to new skill (which already has all the old skill's modifiers)
				newSkill.skillModList.AddMod("Damage", "MORE", moreDamage, "The Saviour", activeSkill.ModFlags, activeSkill.KeywordFlags)
				if env.player.itemList["Weapon 1"] and env.player.itemList["Weapon 2"] and env.player.itemList["Weapon 1"].name == env.player.itemList["Weapon 2"].name {
					maxMirageWarriors = maxMirageWarriors / 2
				}
				newSkill.skillModList.AddMod("QuantityMultiplier", "BASE", maxMirageWarriors, "The Saviour Mirage Warriors", activeSkill.ModFlags, activeSkill.KeywordFlags)

				if usedSkill.skillPartName {
					env.player.mainSkill.skillPart = usedSkill.skillPart
					env.player.mainSkill.skillPartName = usedSkill.skillPartName
					env.player.mainSkill.infoMessage2 = usedSkill.activeEffect.grantedEffect.name
				} else {
					env.player.mainSkill.skillPartName = usedSkill.activeEffect.grantedEffect.name
				}

				// Recalculate the offensive/defensive aspects of this new skill
				newEnv.player.mainSkill = newSkill
				calcs.perform(newEnv)
				env.player.mainSkill = newSkill

				env.player.mainSkill.infoMessage = tostring(maxMirageWarriors) + " Mirage Warriors using " + usedSkill.activeEffect.grantedEffect.name

				// Re-link over the output
				env.player.output = newEnv.player.output
				if newSkill.minion {
					env.minion = newEnv.player.mainSkill.minion
					env.minion.output = newEnv.minion.output
				}

				// Make any necessary corrections to output
				env.player.Output["ManaCost"] = 0

				// Re-link over the breakdown (if present)
				if newEnv.player.breakdown {
					env.player.breakdown = newEnv.player.breakdown

					// Make any necessary corrections to breakdown
					env.player.breakdown.ManaCost = nil

					if newSkill.minion {
						env.minion.breakdown = newEnv.minion.breakdown
					}
				}
			} else {
				activeSkill.infoMessage2 = "No Saviour active skill found"
			}
		}
	*/
	// Calculate combined DPS estimate, including DoTs
	baseDPS := actor.Output[utils.Ternary(skillData.ShowAverage, "AverageDamage", "TotalDPS")]
	actor.Output["CombinedDPS"] = baseDPS
	actor.Output["CombinedAvg"] = baseDPS
	if skillFlags[SkillFlagDot] {
		actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + (actor.Output["TotalDot"])
		actor.Output["WithDotDPS"] = baseDPS + (actor.Output["TotalDot"])
	}
	if quantityMultiplier > 1 && actor.Output["TotalPoisonDPS"] > 0 {
		actor.Output["TotalPoisonDPS"] = actor.Output["TotalPoisonDPS"] * quantityMultiplier
	}
	if skillData.ShowAverage {
		actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + (actor.Output["TotalPoisonDPS"])
		actor.Output["CombinedAvg"] = actor.Output["CombinedAvg"] + (actor.Output["PoisonDamage"])
		actor.Output["WithPoisonDPS"] = baseDPS + (actor.Output["TotalPoisonAverageDamage"])
	} else {
		actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + (actor.Output["TotalPoisonDPS"])
		actor.Output["WithPoisonDPS"] = baseDPS + (actor.Output["TotalPoisonDPS"])
	}
	if skillFlags[SkillFlagIgnite] {
		if skillFlags[SkillFlagIgniteCanStack] {
			if skillData.ShowAverage {
				actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + actor.Output["TotalIgniteDPS"]
				actor.Output["CombinedAvg"] = actor.Output["CombinedDPS"] + actor.Output["IgniteDamage"]
			} else {
				actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + actor.Output["TotalIgniteDPS"]
				actor.Output["WithIgniteDPS"] = baseDPS + actor.Output["TotalIgniteDPS"]
			}
		} else if skillData.ShowAverage {
			actor.Output["WithIgniteDPS"] = baseDPS + actor.Output["IgniteDamage"]
			actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + actor.Output["IgniteDPS"]
			actor.Output["CombinedAvg"] = actor.Output["CombinedAvg"] + actor.Output["IgniteDamage"]
		} else {
			actor.Output["WithIgniteDPS"] = baseDPS + actor.Output["IgniteDPS"]
			actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + actor.Output["IgniteDPS"]
		}
	} else {
		actor.Output["WithIgniteDPS"] = baseDPS
	}
	if skillFlags[SkillFlagBleed] {
		if skillData.ShowAverage {
			actor.Output["WithBleedDPS"] = baseDPS + actor.Output["BleedDamage"]
			actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + actor.Output["BleedDPS"]
			actor.Output["CombinedAvg"] = actor.Output["CombinedAvg"] + actor.Output["BleedDamage"]
		} else {
			actor.Output["WithBleedDPS"] = baseDPS + actor.Output["BleedDPS"]
			actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + actor.Output["BleedDPS"]
		}
	} else {
		actor.Output["WithBleedDPS"] = baseDPS
	}
	if skillFlags[SkillFlagDecay] {
		actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + actor.Output["DecayDPS"]
	}
	actor.Output["TotalDotDPS"] = (actor.Output["TotalDot"]) + (actor.Output["TotalPoisonDPS"]) + utils.OrDefault(actor.Output["TotalIgniteDPS"], actor.Output["IgniteDPS"]) + (actor.Output["BleedDPS"]) + (actor.Output["DecayDPS"])
	if skillFlags[SkillFlagImpale] {
		if skillFlags[SkillFlagAttack] {
			actor.Output["ImpaleHit"] = (utils.OrDefault(actor.OutputTable[OutTableMainHand]["PhysicalHitAverage"], actor.OutputTable[OutTableOffHand]["PhysicalHitAverage"])+utils.OrDefault(actor.OutputTable[OutTableOffHand]["PhysicalHitAverage"], actor.OutputTable[OutTableMainHand]["PhysicalHitAverage"]))/2*(1-actor.Output["CritChance"]/100) + (utils.OrDefault(actor.OutputTable[OutTableMainHand]["PhysicalCritAverage"], actor.OutputTable[OutTableOffHand]["PhysicalCritAverage"])+utils.OrDefault(actor.OutputTable[OutTableOffHand]["PhysicalCritAverage"], actor.OutputTable[OutTableMainHand]["PhysicalCritAverage"]))/2*(actor.Output["CritChance"]/100)
			if skillData.DoubleHitsWhenDualWielding && skillFlags[SkillFlagBothWeaponAttack] {
				actor.Output["ImpaleHit"] = actor.Output["ImpaleHit"] * 2
			}
		} else {
			actor.Output["ImpaleHit"] = actor.Output["PhysicalHitAverage"]*(1-actor.Output["CritChance"]/100) + actor.Output["PhysicalCritAverage"]*(actor.Output["CritChance"]/100)
		}
		actor.Output["ImpaleDPS"] = actor.Output["ImpaleHit"] * (utils.OrDefault(actor.Output["ImpaleModifier"], 1) - 1) * actor.Output["HitChance"] / 100 * utils.OrDefault(skillData.DpsMultiplier, 1)
		if skillData.ShowAverage {
			actor.Output["WithImpaleDPS"] = actor.Output["AverageDamage"] + actor.Output["ImpaleDPS"]
			actor.Output["CombinedAvg"] = actor.Output["CombinedAvg"] + actor.Output["ImpaleDPS"]
		} else {
			skillFlags[SkillFlagNotAverage] = true
			actor.Output["ImpaleDPS"] = actor.Output["ImpaleDPS"] * utils.OrDefault(actor.Output["HitSpeed"], actor.Output["Speed"])
			actor.Output["WithImpaleDPS"] = actor.Output["TotalDPS"] + actor.Output["ImpaleDPS"]
		}
		if quantityMultiplier > 1 {
			actor.Output["ImpaleDPS"] = actor.Output["ImpaleDPS"] * quantityMultiplier
		}
		actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + actor.Output["ImpaleDPS"]
		if breakdown != nil {
			breakdown.AddLine("ImpaleDPS", fmt.Sprintf("%.2f ^8(average physical hit)", actor.Output["ImpaleHit"]))
			breakdown.AddLine("ImpaleDPS", fmt.Sprintf("x %.2f ^8(chance to hit)", actor.Output["HitChance"]/100))
			if skillFlags[SkillFlagNotAverage] {
				breakdown.AddLine("ImpaleDPS", utils.Ternary(actor.Output["HitSpeed"] != 0, fmt.Sprintf("x %.2f ^8(hit rate)", actor.Output["HitSpeed"]), fmt.Sprintf("x %.2f ^8(%s rate)", actor.Output["Speed"], utils.Ternary(skillFlags[SkillFlagAttack], "attack", "cast"))))
			}
			breakdown.AddLine("ImpaleDPS", fmt.Sprintf("x %.2f ^8(impale damage multiplier)", utils.OrF(actor.Output["ImpaleModifier"], 1)-1))
			if skillData.DpsMultiplier != 0 {
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("x %g ^8(dps multiplier for this skill)", skillData.DpsMultiplier))
			}
			if quantityMultiplier > 1 {
				breakdown.AddLine("ImpaleDPS", fmt.Sprintf("x %g ^8(quantity multiplier for this skill)", quantityMultiplier))
			}
			breakdown.AddLine("ImpaleDPS", fmt.Sprintf("= %.1f", actor.Output["ImpaleDPS"]))
		}
	}

	bestCull := float64(1)
	/*
		TODO Mirage
		if activeSkill.mirage and activeSkill.mirage.output and activeSkill.mirage.Output["TotalDPS"] {
			mirageCount := activeSkill.mirage.count or 1
			actor.Output["MirageDPS"] = activeSkill.mirage.Output["TotalDPS"] * mirageCount
			actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + activeSkill.mirage.Output["TotalDPS"] * mirageCount

			if activeSkill.mirage.Output["IgniteDPS"] and activeSkill.mirage.Output["IgniteDPS"] > (actor.Output["IgniteDPS"] or 0) {
				actor.Output["MirageDPS"] = actor.Output["MirageDPS"] + activeSkill.mirage.Output["IgniteDPS"]
				actor.Output["IgniteDPS"] = 0
			}
			if activeSkill.mirage.Output["BleedDPS"] and activeSkill.mirage.Output["BleedDPS"] > (actor.Output["BleedDPS"] or 0) {
				actor.Output["MirageDPS"] = actor.Output["MirageDPS"] + activeSkill.mirage.Output["BleedDPS"]
				actor.Output["BleedDPS"] = 0
			}

			if activeSkill.mirage.Output["PoisonDPS"] {
				actor.Output["MirageDPS"] = actor.Output["MirageDPS"] + activeSkill.mirage.Output["PoisonDPS"] * mirageCount
				actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + activeSkill.mirage.Output["PoisonDPS"] * mirageCount
			}
			if activeSkill.mirage.Output["ImpaleDPS"] {
				actor.Output["MirageDPS"] = actor.Output["MirageDPS"] + activeSkill.mirage.Output["ImpaleDPS"] * mirageCount
				actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + activeSkill.mirage.Output["ImpaleDPS"] * mirageCount
			}
			if activeSkill.mirage.Output["DecayDPS"] {
				actor.Output["MirageDPS"] = actor.Output["MirageDPS"] + activeSkill.mirage.Output["DecayDPS"]
				actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + activeSkill.mirage.Output["DecayDPS"]
			}
			if activeSkill.mirage.Output["TotalDot"] and (skillFlags.DotCanStack or not actor.Output["TotalDot"] or actor.Output["TotalDot"] == 0) {
				actor.Output["MirageDPS"] = actor.Output["MirageDPS"] + activeSkill.mirage.Output["TotalDot"] * (skillFlags.DotCanStack and mirageCount or 1)
				actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] + activeSkill.mirage.Output["TotalDot"] * (skillFlags.DotCanStack and mirageCount or 1)
			}
			if activeSkill.mirage.Output["CullMultiplier"] > 1 {
				bestCull = activeSkill.mirage.Output["CullMultiplier"]
			}
		}
	*/

	bestCull = max(bestCull, actor.Output["CullMultiplier"])
	actor.Output["CullingDPS"] = actor.Output["CombinedDPS"] * (bestCull - 1)
	actor.Output["CombinedDPS"] = actor.Output["CombinedDPS"] * bestCull
}
