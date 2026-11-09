package calculator2

import (
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

//pob2:block:src/Modules/Data.lua:409
var ailmentTypeList = []string{"Bleed", "Poison", "Ignite", "Chill", "Freeze", "Shock"}

// nonDamagingAilment is the part of a data.nonDamagingAilment row the offence calculation reads.
type nonDamagingAilment struct {
	associatedType string
	min, max       float64
	precision      float64
	duration       float64
}

//pob2:block:src/Modules/Data.lua:414-418
func nonDamagingAilmentData(ailment string) (nonDamagingAilment, bool) {
	switch ailment {
	case "Chill":
		return nonDamagingAilment{associatedType: "Cold", min: 30, max: gameConstant("ChillMaxEffect"), duration: gameConstant("BaseChillDuration")}, true
	case "Freeze":
		return nonDamagingAilment{associatedType: "Cold", min: 0.3, max: 3, precision: 2, duration: gameConstant("FreezeDuration")}, true
	case "Shock":
		return nonDamagingAilment{associatedType: "Lightning", min: gameConstant("BaseShockMagnitude"), max: 100, duration: gameConstant("BaseShockDuration")}, true
	}
	return nonDamagingAilment{}, false
}

//pob2:block:src/Modules/Data.lua:420-443
var buildupScalesFrom = map[string]map[string]bool{
	"Electrocute": {},
	"Freeze":      {"Cold": true},
	"HeavyStun":   {"Physical": true, "Fire": true, "Cold": true, "Lightning": true, "Chaos": true},
	"Pin":         {},
}

//pob2:block:src/Modules/Data.lua:445-477
var defaultAilmentScalesFrom = map[string]map[string]bool{
	"Bleed":  {"Physical": true},
	"Poison": {"Physical": true, "Chaos": true},
	"Ignite": {"Fire": true},
	"Shock":  {"Lightning": true},
	"Chill":  {"Cold": true},
}

var defaultAilmentDamageType = map[string]string{"Bleed": "Physical", "Poison": "Chaos", "Ignite": "Fire"}

// ailmentKeywordFlags is KeywordFlag[name] for the names the ailment half indexes.
var ailmentKeywordFlags = map[string]mod.KeywordFlag{
	"Bleed": KeywordFlagBleed, "Poison": KeywordFlagPoison, "Ignite": KeywordFlagIgnite,
	"PhysicalDot": KeywordFlagPhysicalDot, "LightningDot": KeywordFlagLightningDot, "ColdDot": KeywordFlagColdDot,
	"FireDot": KeywordFlagFireDot, "ChaosDot": KeywordFlagChaosDot,
}

// nonDamagingAilmentConfig is a nonDamagingAilmentsConfig entry: the breakdown's effect list and its formulas.
type nonDamagingAilmentConfig struct {
	effList        []float64
	effect, thresh func(damage, x, effectMod float64) float64
	ramping        bool
}

// nonDamagingAilmentsConfigFor is nonDamagingAilmentsConfig over the pass's enemyThreshold; effect takes no value.
//
//pob2:block:src/Modules/CalcOffence.lua:5671-5684
func nonDamagingAilmentsConfigFor(enemyThreshold float64) map[string]nonDamagingAilmentConfig {
	return map[string]nonDamagingAilmentConfig{
		"Chill": {
			effList: []float64{10, 20},
			effect: func(damage, effectMod, _ float64) float64 {
				return gameConstant("ChillEffectMultiplier") * (damage / enemyThreshold) * effectMod
			},
			thresh: func(damage, value, effectMod float64) float64 {
				return float64(damage * (gameConstant("ChillEffectMultiplier") * effectMod / value))
			},
		},
		"Shock": {
			effList: []float64{10, 20, 40},
			effect: func(damage, effectMod, _ float64) float64 {
				return 50 * math.Pow(damage/enemyThreshold, 0.4) * effectMod
			},
			thresh: func(damage, value, effectMod float64) float64 {
				return float64(damage * math.Pow(50*effectMod/value, 2.5))
			},
			ramping: true,
		},
	}
}

// monsterTable is data.<table>[level].
func monsterTable(table []float64, level int) float64 {
	return table[level-1]
}

// newSkillCfg is a cfg table carrying skillCfg's skill fields, as the dot and ailment cfgs are built.
func newSkillCfg(skillCfg *moddb.ListCfg, flags mod.MFlag, keywordFlags mod.KeywordFlag) *moddb.ListCfg {
	return &moddb.ListCfg{
		SkillName:       skillCfg.SkillName,
		SkillPart:       skillCfg.SkillPart,
		SkillTypes:      skillCfg.SkillTypes,
		SummonSkillName: skillCfg.SummonSkillName,
		SlotName:        skillCfg.SlotName,
		Flags:           &flags,
		KeywordFlags:    &keywordFlags,
	}
}

// copyCfgShallow is copyTable(cfg, true): the copy shares skillCond, but owns its flags.
func copyCfgShallow(cfg *moddb.ListCfg) *moddb.ListCfg {
	c := *cfg
	flags := cfg.Flags.Get()
	keywordFlags := cfg.KeywordFlags.Get()
	c.Flags = &flags
	c.KeywordFlags = &keywordFlags
	return &c
}

func outputDefault(output map[string]float64, key string, def float64) float64 {
	if v, ok := output[key]; ok {
		return v
	}
	return def
}

func overrideFloat(v *mod.ModValueMulti) (float64, bool) {
	if v == nil {
		return 0, false
	}
	return v.Float(), true
}

// offenceAilments calculates ailments and debuffs (poison, bleed, ignite, impale, exposure, etc) for
// every pass, then combines them.
//
//pob2:block:src/Modules/CalcOffence.lua:4962-5964
func (h *offenceHit) offenceAilments() {
	env, actor, activeSkill, state, skillFlags := h.env, h.actor, h.activeSkill, h.state, h.skillFlags
	globalOutput := actor.Output
	enemyDB := actor.Enemy.ModDB
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	canDeal := state.canDeal
	isAttack := state.isAttack

	debuffDurationMult := 1.0
	if env.ModeEffective {
		debuffDurationMult = calcBuffExpirationMult(enemyDB, skillCfg)
	}

	for _, ailment := range ailmentTypeList {
		skillFlags[strings.ToLower(ailment)] = false
	}
	skillFlags["impale"] = false

	// Calculate ailment thresholds
	enemyThreshold := float64(monsterTable(poe2.MonsterAilmentThresholdTable, env.EnemyLevel) * calcLibMod(enemyDB, nil, "EnemyAilmentThreshold"))
	globalOutput["EnemyAilmentThreshold"] = enemyThreshold

	// Calculate ailments and debuffs (poison, bleed, ignite, impale, exposure, etc)
	for _, pass := range state.passList {
		output := pass.output
		cfg := pass.cfg
		breakdown, globalBreakdown := pass.breakdown, actor.Breakdown

		// Legacy PoE1 ailments (to be removed later): Scorched, Brittle, Sapped, Impale
		output["ImpaleChance"] = 0
		output["ImpaleChanceOnCrit"] = 0
		output["ScorchChance"] = 0
		output["BrittleChance"] = 0
		output["SappedChance"] = 0
		output["ChaosPoisonChance"] = 0

		// address Weapon1H interaction with Ailment for nodes like Coated Arms (PoE1: Sleight of Hand)
		checkWeapon1HFlags := func(targetCfg *moddb.ListCfg) {
			flags := targetCfg.Flags.Get() | (cfg.Flags.Get() & ModFlagWeapon1H)
			targetCfg.Flags = &flags
		}

		// Check if the skill can inflict a given ailment
		canDoAilment := func(ailmentType, damageType string, defaultDamageTypes map[string]bool) bool {
			if !canDeal[damageType] {
				return false
			}
			// check against input valid types
			data, hasData := nonDamagingAilmentData(ailmentType)
			if defaultDamageTypes[damageType] || (hasData && damageType == data.associatedType) {
				if skillModList.Flag(cfg, damageType+"Cannot"+ailmentType) || skillModList.Flag(cfg, "Cannot"+ailmentType) {
					return false
				}
				return true
			}
			// Process overrides eg. LightningCanFreeze
			return skillModList.Flag(cfg, damageType+"Can"+ailmentType) || skillModList.Flag(cfg, "Can"+ailmentType)
		}

		// Calculates normal and crit damage to be used in non-damaging ailment calculations
		calcAverageUnmitigatedSourceDamage := func(ailment string, defaultDamageTypes map[string]bool) (float64, float64) {
			canCrit := !skillModList.Flag(cfg, "AilmentsAreNeverFromCrit")
			sourceHitDmg, sourceCritDmg := 0.0, 0.0
			for _, dmgType := range dmgTypeList {
				if canDoAilment(ailment, dmgType, defaultDamageTypes) {
					sourceHitDmg += output[dmgType+"HitAverage"]
					if canCrit {
						sourceCritDmg += output[dmgType+"CritAverage"]
					}
				}
			}
			return sourceHitDmg, sourceCritDmg
		}

		// Calculates damage to be used in damaging ailment calculations
		calcMinMaxUnmitigatedAilmentSourceDamage := func(ailment string, defaultDamageTypes map[string]bool) (float64, float64, float64, float64) {
			canCrit := !skillModList.Flag(cfg, "AilmentsAreNeverFromCrit")
			hitMin, hitMax := 0.0, 0.0
			critMin, critMax := 0.0, 0.0
			for _, damageType := range dmgTypeList {
				if canDoAilment(ailment, damageType, defaultDamageTypes) {
					override, hasOverride := overrideFloat(skillModList.Override(cfg, ailment+damageType+"HitDamage"))
					more := skillModList.More(cfg, damageType+ailment+"Buildup")
					ailmentHitMin, ailmentHitMax := override, override
					if !hasOverride {
						ailmentHitMin, ailmentHitMax = output[damageType+"StoredHitMin"], output[damageType+"StoredHitMax"]
					}
					hitMin += float64(ailmentHitMin * more)
					hitMax += float64(ailmentHitMax * more)
					output[ailment+damageType+"Min"] = ailmentHitMin * more
					output[ailment+damageType+"Max"] = ailmentHitMax * more
					if canCrit {
						override, hasOverride = overrideFloat(skillModList.Override(cfg, ailment+damageType+"CritDamage"))
						if hasOverride {
							critMin += float64(override * more)
							critMax += float64(override * more)
						} else {
							critMin += float64(output[damageType+"StoredCritMin"] * more)
							critMax += float64(output[damageType+"StoredCritMax"] * more)
						}
					}
				}
			}
			return hitMin, hitMax, critMin, critMax
		}

		// Calculates damage to be used in poise related ailment calculations; only the averages are
		// returned, as the min / max sums feed the breakdown alone.
		calcMinMaxPoiseSourceDamage := func(ailment string, defaultDamageTypes map[string]bool) (float64, float64, float64, float64, float64, float64, float64) {
			var hitMin, hitMax, hitAvg, critMin, critMax, critAvg, poseDamageAvg float64
			canCrit := !skillModList.Flag(cfg, "AilmentsAreNeverFromCrit")
			critChance := output["CritChance"]
			for _, damageType := range dmgTypeList {
				if canDoAilment(ailment, damageType, defaultDamageTypes) {
					override, hasOverride := overrideFloat(skillModList.Override(cfg, ailment+damageType+"HitDamage"))
					more := skillModList.More(cfg, damageType+ailment+"Buildup")
					ailmentHitMin, ailmentHitMax, ailmentAverage := override, override, override
					if !hasOverride {
						ailmentHitMin, ailmentHitMax, ailmentAverage = output[damageType+"StoredHitMin"], output[damageType+"StoredHitMax"], output[damageType+"HitAverage"]
					}
					effMulti := outputDefault(output, damageType+"EffMult", 1)
					hitMin += float64(ailmentHitMin * more * effMulti)
					hitMax += float64(ailmentHitMax * more * effMulti)
					hitAvg += float64(ailmentAverage * more)
					output[ailment+damageType+"Min"] = ailmentHitMin * more
					output[ailment+damageType+"Max"] = ailmentAverage * more
					if canCrit {
						override, hasOverride = overrideFloat(skillModList.Override(cfg, ailment+damageType+"CritDamage"))
						critMinOf, critMaxOf, critAvgOf := override, override, override
						if !hasOverride {
							critMinOf, critMaxOf, critAvgOf = output[damageType+"StoredCritMin"], output[damageType+"StoredCritMax"], output[damageType+"CritAverage"]
						}
						critMin += float64(critMinOf * more * effMulti)
						critMax += float64(critMaxOf * more * effMulti)
						critAvg += float64(critAvgOf * more)
					}
				}
			}
			if canCrit {
				poseDamageAvg = float64(hitAvg*(1-critChance/100)) + critAvg*critChance/100
			} else {
				poseDamageAvg = hitAvg
			}
			return hitMin, hitMax, hitAvg, critMin, critMax, critAvg, poseDamageAvg
		}

		// Calculate the inflict chance and base damage of a secondary effect (bleed/poison/ignite/shock/freeze)
		//
		//pob2:block:src/Modules/CalcOffence.lua:5111-5189
		calcAilmentDamage := func(ailment string, sourceCritChance, sourceHitDmg, sourceCritDmg float64, hideFromBreakdown bool) float64 {
			chanceOnHit, chanceOnCrit := output[ailment+"ChanceOnHit"], output[ailment+"ChanceOnCrit"]
			// Use sourceCritChance to factor in chance a critical ailment is present
			chanceFromHit := float64(chanceOnHit * (1 - sourceCritChance/100))
			chanceFromCrit := chanceOnCrit * sourceCritChance / 100
			chance := chanceFromHit + chanceFromCrit
			output[ailment+"Chance"] = chance
			baseFromHit := sourceHitDmg * chanceFromHit / (chanceFromHit + chanceFromCrit)
			baseFromCrit := sourceCritDmg * chanceFromCrit / (chanceFromHit + chanceFromCrit)
			baseVal := baseFromHit + baseFromCrit
			sourceMult := skillModList.More(cfg, ailment+"AsThoughDealing")
			if breakdown != nil && chance != 0 && !hideFromBreakdown {
				key := ailment + "Chance"
				if breakdown.Entries[key] == nil {
					breakdown.set(key)
				}
				if len(breakdown.Entries[key].Lines) > 0 {
					breakdown.insert(key, "")
				}
				if isAttack {
					breakdown.insert(key, pass.label+":")
				}
				breakdown.insert(key, sFormat("Chance on Non-crit: %d%%", chanceOnHit), sFormat("Chance on Crit: %d%%", chanceOnCrit))
				if chanceOnHit != chanceOnCrit {
					chancePerHit := float64(chanceOnHit*(1-sourceCritChance/100)) + chanceOnCrit*sourceCritChance/100
					breakdown.insert(key, "Combined chance:",
						sFormat("%d x (1 - %.4f) ^8(chance from non-crits)", chanceOnHit, sourceCritChance/100),
						sFormat("+ %d x %.4f ^8(chance from crits)", chanceOnCrit, sourceCritChance/100),
						sFormat("= %.2f", chancePerHit))
				}
			}
			if breakdown != nil && baseVal > 0 && !hideFromBreakdown {
				key := ailment + "DPS"
				if breakdown.Entries[key] == nil {
					breakdown.set(key)
				}
				if len(breakdown.Entries[key].Lines) > 0 {
					breakdown.insert(key, "")
				}
				if isAttack {
					breakdown.insert(key, pass.label+":")
				}
				if sourceHitDmg == sourceCritDmg || output["CritChance"] == 0 {
					breakdown.insert(key, "Total base DPS per "+ailment+":", sFormat("%.1f ^8(source damage)", sourceHitDmg))
					if sourceMult > 1 {
						breakdown.insert(key, sFormat("x %.2f ^8(inflicting as though dealing more damage)", sourceMult), sFormat("= %.1f", baseVal*sourceMult))
					}
				} else {
					if baseFromHit > 0 {
						breakdown.insert(key, "Damage from Non-crits:",
							sFormat("%.1f ^8(source damage from non-crits)", sourceHitDmg),
							sFormat("x %.3f ^8(portion of instances created by non-crits)", chanceFromHit/(chanceFromHit+chanceFromCrit)))
						if sourceMult == 1 || baseFromCrit != 0 {
							breakdown.insert(key, sFormat("= %.1f", baseFromHit))
						}
					}
					if baseFromCrit > 0 {
						breakdown.insert(key, "Damage from Crits:",
							sFormat("%.1f ^8(source damage from crits)", sourceCritDmg),
							sFormat("x %.3f ^8(portion of instances created by crits)", chanceFromCrit/(chanceFromHit+chanceFromCrit)))
						if sourceMult == 1 || baseFromHit != 0 {
							breakdown.insert(key, sFormat("= %.1f", baseFromCrit))
						}
					}
					if baseFromHit > 0 && baseFromCrit > 0 {
						breakdown.insert(key, "Total base DPS per "+ailment+":", sFormat("%.1f + %.1f", baseFromHit, baseFromCrit))
						if sourceMult == 1 {
							breakdown.insert(key, sFormat("= %.1f", baseVal))
						}
					}
					if sourceMult > 1 {
						breakdown.insert(key, sFormat("x %.2f ^8(inflicting as though dealing more damage)", sourceMult), sFormat("= %.1f", baseVal*sourceMult))
					}
				}
			}
			return baseVal
		}

		// Calculate global / breakdown values for a damaging ailment
		calcDamagingAilmentOutputs := func(ailment, ailmentDamageType string, defaultDamageTypes map[string]bool) {
			if !canDeal[ailmentDamageType] {
				return
			}
			if output[ailment+"ChanceOnHit"]+output[ailment+"ChanceOnCrit"]+output[ailmentDamageType+ailment+"Chance"] <= 0 {
				return
			}

			meleeHit := mod.MFlag(0)
			if cfg.Flags.Get()&ModFlagMelee != 0 {
				meleeHit = ModFlagMeleeHit
			}
			dotCfg := newSkillCfg(skillCfg,
				ModFlagDot|ModFlagAilment|(cfg.Flags.Get()&ModFlagWeaponMask)|meleeHit,
				(cfg.KeywordFlags.Get()&^KeywordFlagHit)|ailmentKeywordFlags[ailment]|KeywordFlagAilment|ailmentKeywordFlags[ailmentDamageType+"Dot"])
			// skillCond's metatable falls back to skillCfg.skillCond, then cfg.skillCond.
			dotCfg.SkillCond = map[string]bool{}
			for k, v := range cfg.SkillCond {
				dotCfg.SkillCond[k] = v
			}
			for k, v := range skillCfg.SkillCond {
				dotCfg.SkillCond[k] = v || cfg.SkillCond[k]
			}
			dotCfg.SkillCond["CriticalStrike"] = true
			dotCfg.SkillDist, dotCfg.SkillDistSet = skillCfg.SkillDist, skillCfg.SkillDistSet
			cfgKey := strings.ToLower(ailment) + "Cfg"
			if pass.label == "Off Hand" {
				cfgKey = "OH" + cfgKey
			}
			if activeSkill.AilmentCfg == nil {
				activeSkill.AilmentCfg = map[string]*moddb.ListCfg{}
			}
			activeSkill.AilmentCfg[cfgKey] = dotCfg
			checkWeapon1HFlags(dotCfg)
			if breakdown != nil {
				for _, damageType := range dmgTypeList {
					breakdown.setEntry(ailment+damageType, &BreakdownEntry{})
				}
			}

			globalOutput[ailment+"ChancePerHit"] = float64(output[ailment+"ChanceOnHit"]*(1-output["CritChance"]/100)) + output[ailment+"ChanceOnCrit"]*output["CritChance"]/100

			// We will be using a weighted average calculation
			maxStacks := 1.0
			if skillModList.Flag(skillCfg, ailment+"CanStack") {
				skillFlags[strings.ToLower(ailment)+"CanStack"] = true
				if o, ok := overrideFloat(skillModList.Override(cfg, ailment+"Stacks")); ok {
					maxStacks = o
				} else {
					maxStacks = float64((maxStacks + skillModList.Sum(mod.TypeBase, cfg, ailment+"Stacks")) * skillModList.More(cfg, ailment+"Stacks"))
				}
			}
			globalOutput[ailment+"StacksMax"] = maxStacks

			// The ailment duration
			ailmentTypeMod := ailmentDamageType
			if ailmentDamageType != "Physical" && ailmentDamageType != "Chaos" {
				ailmentTypeMod = "Elemental"
			}
			rateMod := (calcLibMod(skillModList, cfg, ailment+"Faster") + enemyDB.Sum(mod.TypeIncrease, nil, "Self"+ailment+"Faster")/100) / calcLibMod(skillModList, cfg, ailment+"Slower")
			durationIsSkillDuration := truthy(skillData[strings.ToLower(ailment)+"DurationIsSkillDuration"])
			var durationBase float64
			if d, ok := sdNum(skillData, "duration"); durationIsSkillDuration && ok {
				durationBase = d
			} else if o, ok := overrideFloat(env.ModDB.Override(nil, ailment+"DurationBase")); ok {
				durationBase = o
			} else {
				durationBase = miscConstant(ailment + "DurationBase")
			}
			durationNames := []string{"Enemy" + ailment + "Duration", "EnemyAilmentDuration", "Enemy" + ailmentTypeMod + "AilmentDuration", "DamagingAilmentDuration"}
			if durationIsSkillDuration {
				durationNames = append(durationNames, "Duration")
			}
			durationMod := math.Max(calcLibMod(skillModList, dotCfg, durationNames...)*calcLibMod(enemyDB, nil, "Self"+ailment+"Duration", "SelfAilmentDuration", "Self"+ailmentTypeMod+"AilmentDuration"), 0)
			durationMod = math.Max(durationMod, 0)
			globalOutput[ailment+"Duration"] = durationBase * durationMod / rateMod * debuffDurationMult

			// The chance any given hit applies ailment
			ailmentChance := float64(output[ailment+"ChanceOnHit"]/100*(1-output["CritChance"]/100)) + output[ailment+"ChanceOnCrit"]/100*output["CritChance"]/100

			// The average number of ailment that will be active on the enemy at once
			ailmentStacks := float64(output["HitChance"] / 100 * ailmentChance * output["DpsMultiplier"])
			configStacks := enemyDB.Sum(mod.TypeBase, nil, "Multiplier:"+ailment+"Stacks")
			if !truthy(skillData["triggeredOnDeath"]) {
				hitSpeed := outputDefault(globalOutput, "HitSpeed", globalOutput["Speed"])
				if cooldown, ok := output["Cooldown"]; ok {
					ailmentStacks = ailmentStacks * globalOutput[ailment+"Duration"] / math.Max(cooldown, outputDefault(output, "HitTime", output["Time"]))
				} else if hitSpeed > 0 {
					// assume skills with no cast, attack, or cooldown time are single cast
					ailmentStacks = float64(ailmentStacks * globalOutput[ailment+"Duration"] * hitSpeed)
				}

				activeTotems, ok := overrideFloat(env.ModDB.Override(nil, "TotemsSummoned"))
				if !ok {
					activeTotems = skillModList.Sum(mod.TypeBase, skillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
				}
				if skillFlags["totem"] {
					ailmentStacks *= activeTotems
				}
				if configStacks > 0 {
					ailmentStacks = configStacks
				}
				if ailmentStacks <= 1 {
					skillModList.AddMod(mod.NewFlag("Condition:Single"+ailment, true).Source(mod.Source(strings.ToLower(ailment))))
				}
			}

			// Ratio of ailments applied : max effective ailments
			globalOutput[ailment+"StackPotential"] = ailmentStacks / maxStacks
			globalOutput[ailment+"StackPotentialPercent"] = globalOutput[ailment+"StackPotential"] * 100
			if globalBreakdown != nil {
				key := ailment + "StackPotential"
				globalBreakdown.set(key, sFormat(colorCustom+"The percentage of your max stacks that are applied on average if you are attacking constantly"), sFormat(""))
				if configStacks > 0 {
					globalBreakdown.insert(key, sFormat("%.2f ^8(ailment stacks config override)", configStacks))
				} else {
					globalBreakdown.insert(key, sFormat("%.2f ^8(chance to hit)", output["HitChance"]/100), sFormat("* %.2f ^8(chance to apply)", ailmentChance))
					hitSpeed := outputDefault(globalOutput, "HitSpeed", globalOutput["Speed"])
					if truthy(skillData["triggeredOnDeath"]) {
						globalBreakdown.insert(key, sFormat("* 1 ^8Cast on Death override"))
					} else if cooldown, ok := output["Cooldown"]; ok {
						globalBreakdown.insert(key, sFormat("* (%.2f / max(%.2f, %.2f) ^8(Duration / max(Cooldown, Cast Time))", globalOutput[ailment+"Duration"], cooldown, outputDefault(output, "HitTime", output["Time"])))
					} else if hitSpeed > 0 {
						globalBreakdown.insert(key, sFormat("* (%.2f / %.2f) ^8(Duration / Attack Time)", globalOutput[ailment+"Duration"], outputDefault(globalOutput, "HitTime", output["Time"])))
					}
					if output["DpsMultiplier"] != 1 {
						globalBreakdown.insert(key, sFormat("* %g ^8(DPS multiplier for this skill)", output["DpsMultiplier"]))
					}
				}
				globalBreakdown.insert(key, sFormat("/ %d ^8(max number of stacks)", maxStacks), sFormat("= %.2f", globalOutput[key]))
			}

			// the amount of damage each application does as % maximum
			var ailmentRollAverage float64
			if globalOutput[ailment+"StackPotential"] > 1 {
				// shift damage towards top of range as only top applications apply
				ailmentRollAverage = (ailmentStacks - (maxStacks-1)/2) / (ailmentStacks + 1) * 100
			} else {
				// assume middle of range for hit damage
				ailmentRollAverage = 50
			}
			globalOutput[ailment+"RollAverage"] = ailmentRollAverage

			if globalBreakdown != nil {
				key := ailment + "RollAverage"
				globalBreakdown.set(key,
					sFormat(colorCustom+"This is the average roll of an ailment affecting the enemy if you are constantly attacking"),
					sFormat(colorCustom+"Uses a weighted average formula when stack potential is over 100%%"),
					sFormat(colorCustom+"If hitting constantly, your average strongest ailment currently achieves ^7%.2f%%"+colorCustom+" of its max damage", ailmentRollAverage),
					sFormat(""),
					sFormat("Average Roll:"))
				if ailmentStacks >= maxStacks {
					globalBreakdown.insert(key, sFormat("%.2f - (%.2f - 1)/2", ailmentStacks, maxStacks), sFormat("/ (%.2f + 1)", ailmentStacks), sFormat("= %.2f%%", ailmentRollAverage))
				} else {
					globalBreakdown.insert(key, sFormat("50%% (averaging <= %d applications)", maxStacks))
				}
			}

			hitMin, hitMax, critMin, critMax := calcMinMaxUnmitigatedAilmentSourceDamage(ailment, defaultDamageTypes)
			hitAvg := hitMin + ((hitMax - hitMin) * ailmentRollAverage / 100)
			critAvg := critMin + ((critMax - critMin) * ailmentRollAverage / 100)
			if globalBreakdown != nil {
				key := ailment + "DPS"
				globalBreakdown.set(key,
					sFormat("Non-Crit Dmg Derivation:"),
					sFormat("%.0f + (%.0f - %.0f) * %.2f%%", hitMin, hitMax, hitMin, ailmentRollAverage),
					sFormat("^8min combined sources + (max combined sources - min combined sources) * average roll"),
					sFormat("= %.2f", hitAvg))
				if hitAvg != critAvg || output["CritChance"] > 0 {
					globalBreakdown.insert(key, "", "Crit Dmg Derivation:",
						sFormat("%.0f + (%.0f - %.0f) * %.2f%%", critMin, critMax, critMin, ailmentRollAverage),
						"^8min combined sources + (max combined sources - min combined sources) * average roll",
						sFormat("= %.2f", critAvg))
				}
			}

			// Over-stacking stacks increases the chance a critical is present
			ailmentCritChance := 100 * (1 - math.Pow(1-output["CritChance"]/100, math.Max(globalOutput[ailment+"StackPotential"], 1)))
			globalOutput[ailment+"MagnitudeEffect"] = calcLibMod(skillModList, dotCfg, "AilmentMagnitude")
			ailmentPercentBase := miscConstant(ailment+"PercentBase") * globalOutput[ailment+"MagnitudeEffect"]
			baseMinVal := calcAilmentDamage(ailment, ailmentCritChance, hitMin, 0, true) * ailmentPercentBase
			baseMaxVal := calcAilmentDamage(ailment, 100, hitMax, critMax, true) * ailmentPercentBase
			baseVal := calcAilmentDamage(ailment, ailmentCritChance, hitAvg, critAvg, false) * ailmentPercentBase

			if baseVal > 0 {
				skillFlags[strings.ToLower(ailment)] = true
				effMult := 1.0
				if env.ModeEffective {
					takenType := ailmentDamageType
					elementalDamageTaken := isElemental[ailmentDamageType]
					switch {
					case skillModList.Flag(cfg, ailment+"ToChaos"):
						takenType, elementalDamageTaken = "Chaos", false
					case skillModList.Flag(cfg, ailment+"ToFire"):
						takenType, elementalDamageTaken = "Fire", false
					}
					resist := calcResistForType(env, enemyDB, takenType, dotCfg)
					names := []string{"DamageTaken", "DamageTakenOverTime", takenType + "DamageTaken", takenType + "DamageTakenOverTime"}
					if elementalDamageTaken {
						names = append(names, "ElementalDamageTaken")
					}
					takenInc := enemyDB.Sum(mod.TypeIncrease, dotCfg, names...)
					takenMore := enemyDB.More(dotCfg, names...)
					effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
					globalOutput[ailment+"EffMult"] = effMult
					if breakdown != nil && effMult != 1 {
						sourceRes := takenType
						if env.ModDB.Flag(nil, "Enemy"+takenType+"ResistEqualToYours") {
							sourceRes = "Your " + takenType + " Resistance"
						}
						globalBreakdown.set(ailment+"EffMult", breakdown.effMult(takenType, resist, 0, takenInc, effMult, takenMore, sourceRes, true, 0, 0, calcPenResist(resist, 0, 0), 0, 0)...)
					}
				}

				effectMod := calcLibMod(skillModList, dotCfg, "AilmentEffect")
				activeAilments := math.Min(ailmentStacks, maxStacks)
				ailmentDPSUncapped := float64(baseVal * effectMod * rateMod * activeAilments * effMult)
				ailmentDPSCapped := math.Min(ailmentDPSUncapped, miscConstant("DotDpsCap"))
				minAilmentDPSUncapped := float64(baseMinVal * effectMod * rateMod * activeAilments * effMult)
				minAilmentDPSCapped := luaMin(minAilmentDPSUncapped, miscConstant("DotDpsCap"))
				maxAilmentDPSUncapped := float64(baseMaxVal * effectMod * rateMod * activeAilments * effMult)
				maxAilmentDPSCapped := luaMin(maxAilmentDPSUncapped, miscConstant("DotDpsCap"))
				output[ailment+"DPS"] = ailmentDPSCapped

				switch ailment {
				case "Ignite":
					groundMult := math.Max(skillModList.Max(nil, "IgniteDpsAsBurningGround"), enemyDB.Max(nil, "IgniteDpsAsBurningGround"))
					if groundMult > 0 {
						groundDPSUncapped := baseVal * effectMod * rateMod * effMult * groundMult / 100
						globalOutput["BurningGroundDPS"] = math.Min(groundDPSUncapped, miscConstant("DotDpsCap"))
						h.actor.setOutputFlag("BurningGroundFromIgnite", true)
						if globalBreakdown != nil {
							globalBreakdown.set("BurningGroundDPS",
								sFormat("%.1f ^8(ignite damage per second)", baseVal*effectMod*rateMod),
								sFormat("* %.1f%% ^8(percent as Burning ground)", groundMult),
								sFormat("* %.3f ^8(effective DPS modifier)", effMult),
								sFormat("= %.1f ^8per second", globalOutput["BurningGroundDPS"]))
						}
					}
				case "Poison":
					groundMult := math.Max(skillModList.Max(nil, "PoisonDpsAsCausticGround"), enemyDB.Max(nil, "PoisonDpsAsCausticGround"))
					if groundMult > 0 {
						groundDPSUncapped := baseVal * effectMod * rateMod * effMult * groundMult / 100
						globalOutput["CausticGroundDPS"] = math.Min(groundDPSUncapped, miscConstant("DotDpsCap"))
						if globalBreakdown != nil {
							globalBreakdown.set("CausticGroundDPS",
								sFormat("%.1f ^8(single poison damage per second)", baseVal*effectMod*rateMod),
								sFormat("* %.1f%% ^8(percent as Caustic ground)", groundMult),
								sFormat("* %.3f ^8(effective DPS modifier)", effMult),
								sFormat("= %.1f ^8per second", globalOutput["CausticGroundDPS"]))
						}
					}
				}

				globalOutput[ailment+"Damage"] = output[ailment+"DPS"] * globalOutput[ailment+"Duration"]
				if skillFlags[strings.ToLower(ailment)+"CanStack"] {
					output[ailment+"Damage"] = output[ailment+"DPS"] * globalOutput[ailment+"Duration"]
					output[ailment+"StacksMax"] = maxStacks
					output["Total"+ailment+"DPS"] = output[ailment+"DPS"]
				}

				//pob2:block:src/Modules/CalcOffence.lua:5449-5518
				if breakdown != nil {
					key := ailment + "DPS"
					percentBase := miscConstant(ailment + "PercentBase")
					breakdown.insert(key,
						sFormat("x %.2f ^8(ailment deals %d%% per second)", percentBase, percentBase*100),
						sFormat("x %.2f ^8(ailment magnitude effect)", globalOutput[ailment+"MagnitudeEffect"]),
						sFormat("= %.1f", baseVal, 1),
						"",
						"Average DPS for all "+ailment+"s:")
					if baseVal != ailmentDPSUncapped {
						breakdown.insert(key, sFormat("%.1f ^8(base damage per second)", baseVal))
					}
					if effectMod != 1 {
						breakdown.insert(key, sFormat("x %.2f ^8(ailment effect modifier)", effectMod))
					}
					if rateMod != 1 {
						breakdown.insert(key, sFormat("x %.2f ^8(rate modifier)", rateMod))
					}
					if activeAilments != 1 {
						breakdown.insert(key, sFormat("x %.2f ^8(avg ailment stacks)", activeAilments))
					}
					if effMult != 1 {
						breakdown.insert(key, sFormat("x %.3f ^8(effective DPS modifier from enemy debuffs)", effMult))
					}
					if output[key] != ailmentDPSUncapped {
						breakdown.insert(key,
							sFormat("= %.1f ^8(Uncapped raw ailment DPS)", ailmentDPSUncapped),
							sFormat("^8(Raw ailment DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", ailmentDPSUncapped-ailmentDPSCapped, (ailmentDPSUncapped-ailmentDPSCapped)/ailmentDPSCapped*100),
							sFormat("= %d ^8(Capped ailment DPS)", ailmentDPSCapped))
					} else {
						breakdown.insert(key, sFormat("= %.1f ^8per second", output[key]))
					}
					breakdown.insert(key, sFormat("%.2f%% of Maximum ailment DPS", output[key]/maxAilmentDPSCapped*100), "", "DPS Range:")
					if maxAilmentDPSCapped == maxAilmentDPSUncapped && minAilmentDPSCapped == minAilmentDPSUncapped {
						breakdown.insert(key, sFormat("%.0f to %.0f ^8(ailment DPS Range)", minAilmentDPSUncapped, maxAilmentDPSUncapped))
					} else {
						breakdown.insert(key, sFormat("%.0f to %.0f ^8(Uncapped ailment DPS Range)", minAilmentDPSUncapped, maxAilmentDPSUncapped))
					}
					if minAilmentDPSCapped != minAilmentDPSUncapped {
						breakdown.insert(key, sFormat("^8(Raw Min ailment DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", minAilmentDPSUncapped-minAilmentDPSCapped, (minAilmentDPSUncapped-minAilmentDPSCapped)/minAilmentDPSCapped*100))
					}
					if maxAilmentDPSCapped != maxAilmentDPSUncapped {
						breakdown.insert(key, sFormat("^8(Raw Max ailment DPS is "+colorNegative+"overcapped ^8by^7 %.0f ^8:^7 %.1f%%^8)", maxAilmentDPSUncapped-maxAilmentDPSCapped, (maxAilmentDPSUncapped-maxAilmentDPSCapped)/maxAilmentDPSCapped*100))
					}
					if maxAilmentDPSCapped != maxAilmentDPSUncapped || minAilmentDPSCapped != minAilmentDPSUncapped {
						breakdown.insert(key, sFormat("%.0f to %.0f ^8(Capped ailment DPS Range)", minAilmentDPSCapped, maxAilmentDPSCapped))
					}
					if skillFlags[strings.ToLower(ailment)+"CanStack"] {
						dmgKey := ailment + "Damage"
						breakdown.set(dmgKey)
						if isAttack {
							breakdown.insert(dmgKey, pass.label+":")
						}
						breakdown.insert(dmgKey,
							sFormat("%.1f ^8(DPS of all stacks)", baseVal),
							sFormat("x %.2fs ^8(ailment duration)", globalOutput[ailment+"Duration"]),
							sFormat("= %.1f ^8total damage of all stacks", output[dmgKey]))
					}
					if globalOutput[ailment+"Duration"] != durationBase {
						durKey := ailment + "Duration"
						globalBreakdown.set(durKey, sFormat("%.2fs ^8(base duration)", durationBase))
						if durationMod != 1 {
							globalBreakdown.insert(durKey, sFormat("x %.2f ^8(duration modifier)", durationMod))
						}
						if rateMod != 1 {
							globalBreakdown.insert(durKey, sFormat("/ %.2f ^8(burn rate modifier)", rateMod))
						}
						if debuffDurationMult != 1 {
							globalBreakdown.insert(durKey, sFormat("/ %.2f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
						}
						globalBreakdown.insert(durKey, sFormat("= %.2fs", globalOutput[durKey]))
					}
				}
			}
		}

		// Knockback (not sure why this is in ailment calc, but I'll calculate it anyway)
		output["KnockbackChanceOnHit"] = 0
		output["KnockbackChanceOnCrit"] = 0
		if skillModList.Flag(cfg, "Knockback") { // From what I could see, all skills are 0% or 100%, with no enemy mitigation
			output["KnockbackChanceOnHit"] = 100
			output["KnockbackChanceOnCrit"] = 100
		}

		// Calculate flat chance ailment (Poison, Bleed) chance on crit
		for _, flatAilment := range []string{"Bleed", "Poison"} {
			if !skillFlags["hit"] || skillModList.Flag(cfg, "Cannot"+flatAilment) {
				output[flatAilment+"ChanceOnHit"] = 0
				output[flatAilment+"ChanceOnCrit"] = 0
				skillFlags["inflict"+flatAilment] = false
			} else {
				for _, val := range []string{"OnHit", "OnCrit"} {
					// copyTable(cfg, true) shares cfg.skillCond, so this sets cfg's own CriticalStrike.
					critCfg := copyCfgShallow(cfg)
					critCfg.SkillCond["CriticalStrike"] = val == "OnCrit" // force crit config to be true for "OnCrit" chance calculation
					base := skillModList.Sum(mod.TypeBase, critCfg, flatAilment+"Chance", "AilmentChance") + enemyDB.Sum(mod.TypeBase, nil, "Self"+flatAilment+"Chance")
					inc := skillModList.Sum(mod.TypeIncrease, critCfg, flatAilment+"Chance", "AilmentChance")
					more := skillModList.More(critCfg, flatAilment+"Chance", "AilmentChance")
					chance, ok := overrideFloat(skillModList.Override(critCfg, flatAilment+"Chance"))
					if !ok {
						chance = base * (1 + inc/100) * more
					}
					output[flatAilment+"Chance"+val] = math.Min(100, chance)
				}
				skillFlags["inflict"+flatAilment] = true
			}
		}

		unmitigatedColdDamage, _ := calcAverageUnmitigatedSourceDamage("Chill", defaultAilmentScalesFrom["Chill"])
		chillMinimumThreshold := enemyThreshold / gameConstant("ChillEffectMultiplier")
		output["chillMinimumThreshold"] = chillMinimumThreshold
		if unmitigatedColdDamage > chillMinimumThreshold {
			output["ChillChanceOnHit"] = 100
			output["ChillChanceOnCrit"] = 100
			skillFlags["inflictChill"] = true
		} else {
			output["ChillChanceOnHit"] = 0
			output["ChillChanceOnCrit"] = 0
			skillFlags["inflictChill"] = false
		}

		output["FreezeChanceOnHit"] = 0
		output["FreezeChanceOnCrit"] = 0
		skillFlags["inflictFreeze"] = false
		skillFlags["inflictElectrocute"] = false

		// Calculate poise-related debuffs
		for _, ailment := range []string{"Freeze", "Electrocute", "HeavyStun", "Pin"} {
			thresholdNames := []string{"PoiseThreshold", ailment + "Threshold"}
			if ailment == "HeavyStun" {
				thresholdNames = append(thresholdNames, "EnemyStunThreshold")
			}
			if ailment == "Freeze" || ailment == "Electrocute" {
				thresholdNames = append(thresholdNames, "EnemyAilmentThreshold")
			}
			enemyPoiseThreshold := math.Floor(monsterTable(poe2.MonsterPoiseThresholdTable, env.EnemyLevel) * calcLibMod(enemyDB, nil, thresholdNames...))
			hitMin, hitMax, hitAvg, critMin, critMax, critAvg, poiseAvg := calcMinMaxPoiseSourceDamage(ailment, buildupScalesFrom[ailment])
			// TODO: average for now, can do more complicated calculation later
			incNames := []string{"EnemyImmobilisationBuildup"}
			if !skillModList.Flag(cfg, "PinBuildupInsteadOf"+ailment+"Buildup") {
				incNames = []string{"Enemy" + ailment + "Buildup", "EnemyImmobilisationBuildup"}
			}
			inc := skillModList.Sum(mod.TypeIncrease, cfg, incNames...)
			buildupNames := []string{ailment + "Buildup", "ImmobilisationBuildup"}
			if ailment == "HeavyStun" {
				buildupNames = append(buildupNames, "StunBuildup")
			}
			more := skillModList.More(cfg, "Enemy"+ailment+"Buildup", "EnemyImmobilisationBuildup") * calcLibMod(enemyDB, nil, buildupNames...)
			poiseBuildup := gameConstant(ailment+"DamageScale") / enemyPoiseThreshold * (1 + inc/100) * more * 100
			minHit := hitMin * poiseBuildup
			maxHit := hitMax * poiseBuildup
			hitPoiseBuildup := hitAvg * poiseBuildup
			minCrit := critMin * poiseBuildup
			maxCrit := critMax * poiseBuildup
			critPoiseBuildup := critAvg * poiseBuildup
			totalAvgPoiseBuildup := poiseAvg * poiseBuildup

			if skillFlags["hit"] && !skillModList.Flag(cfg, "Cannot"+ailment) {
				globalOutput[ailment+"BuildupAvg"] = totalAvgPoiseBuildup
			} else {
				globalOutput[ailment+"BuildupAvg"] = 0
			}

			//pob2:block:src/Modules/CalcOffence.lua:5590-5609
			if pass.breakdown != nil {
				overridden := " ^8(can be overridden in the Configuration tab)"
				if _, ok := env.ConfigInput["enemyLevel"]; ok {
					overridden = " ^8(overridden from the Configuration tab"
				}
				actor.Breakdown.set(ailment+"Buildup",
					"Enemy level: "+strconv.Itoa(env.EnemyLevel)+overridden,
					"Enemy poise: "+luaToString(enemyPoiseThreshold),
					"",
					sFormat("Regular Hit "+ailment+" buildup"),
					sFormat("Min: %.1f%%", minHit),
					sFormat("Max: %.1f%%", maxHit),
					sFormat("Avg: %.1f%%", hitPoiseBuildup),
					sFormat(""),
					sFormat("Crit "+ailment+" buildup"),
					sFormat("Crit Min: %.1f%%", minCrit),
					sFormat("Crit Max: %.1f%%", maxCrit),
					sFormat("Crit Avg: %.1f%%", critPoiseBuildup),
					sFormat(""),
					sFormat("Average "+ailment+" buildup"),
					sFormat("= %.1f%%", totalAvgPoiseBuildup),
				)
			}
		}

		// Calculate scaling threshold ailment chance
		for _, ailment := range []string{"Ignite", "Shock"} {
			hitMin, hitMax, critMin, critMax := calcMinMaxUnmitigatedAilmentSourceDamage(ailment, defaultAilmentScalesFrom[ailment])
			// TODO: average for now, can do more complicated calculation later
			hitAvg := hitMin + (hitMax-hitMin)/2
			critAvg := critMin + (critMax-critMin)/2
			base := skillModList.Sum(mod.TypeBase, cfg, "Enemy"+ailment+"Chance", "AilmentChance") + enemyDB.Sum(mod.TypeBase, nil, "Self"+ailment+"Chance")
			inc := skillModList.Sum(mod.TypeIncrease, cfg, "Enemy"+ailment+"Chance", "AilmentChance") + enemyDB.Sum(mod.TypeIncrease, nil, "Self"+ailment+"Chance")
			more := skillModList.More(cfg, "Enemy"+ailment+"Chance", "AilmentChance") * enemyDB.More(nil, "Self"+ailment+"Chance")
			hitElementalAilmentChance := float64(hitAvg / enemyThreshold * gameConstant(ailment+"ChanceMultiplier"))
			hitElementalAilmentChance = float64((hitElementalAilmentChance + base) * (1 + inc/100) * more)
			critElementalAilmentChance := float64(critAvg / enemyThreshold * gameConstant(ailment+"ChanceMultiplier"))
			critElementalAilmentChance = float64((critElementalAilmentChance + base) * (1 + inc/100) * more)

			if skillFlags["hit"] && !skillModList.Flag(cfg, "Cannot"+ailment) {
				output[ailment+"ChanceOnHit"] = math.Min(100, hitElementalAilmentChance)
				output[ailment+"ChanceOnCrit"] = math.Min(100, critElementalAilmentChance)
			} else {
				output[ailment+"ChanceOnHit"] = 0
				output[ailment+"ChanceOnCrit"] = 0
			}

			anyChanceToAilment := output[ailment+"ChanceOnHit"]
			if !skillModList.Flag(cfg, "NeverCrit") {
				anyChanceToAilment += output[ailment+"ChanceOnCrit"]
			}
			if anyChanceToAilment > 0 {
				skillFlags["inflict"+ailment] = true
			}
		}

		// Apply elemental exposure from skill
		for _, element := range []string{"Fire", "Cold", "Lightning"} {
			if skillModList.Sum(mod.TypeBase, cfg, element+"ExposureChance") > 0 || skillModList.Flag(cfg, "InflictExposure") {
				skillFlags["apply"+element+"Exposure"] = true
			}
		}

		// Apply user damage type config
		ailmentMode, ok := configListValue(env, "ailmentMode")
		if !ok {
			ailmentMode = "AVERAGE"
		}
		if ailmentMode == "CRIT" {
			for _, ailment := range ailmentTypeList {
				output[ailment+"ChanceOnHit"] = 0
			}
		}

		// Calculate damaging ailment values
		for _, damagingAilment := range []string{"Bleed", "Poison", "Ignite"} {
			damageType := defaultAilmentDamageType[damagingAilment]
			if !canDeal[damageType] {
				for _, typ := range dmgTypeList {
					if skillModList.Flag(skillCfg, typ+"Can"+damagingAilment) { // e.g. Avatar of Fire + Blistering Bond -> DealNoPhysical + FireCanBleed
						calcDamagingAilmentOutputs(damagingAilment, typ, defaultAilmentScalesFrom[damagingAilment])
					}
				}
			} else {
				calcDamagingAilmentOutputs(damagingAilment, damageType, defaultAilmentScalesFrom[damagingAilment])
			}
		}

		// Calculate non-damaging ailments effect and duration modifiers
		nonDamagingAilmentsConfig := nonDamagingAilmentsConfigFor(enemyThreshold)
		if activeSkill.SkillTypes["ChillingArea"] || activeSkill.SkillTypes["NonHitChill"] {
			skillFlags["chill"] = true
			incChill := skillModList.Sum(mod.TypeIncrease, cfg, "EnemyChillMagnitude", "AilmentMagnitude")
			moreChill := skillModList.More(cfg, "EnemyChillMagnitude")
			output["ChillEffectMod"] = (1 + incChill/100) * moreChill
			output["ChillDurationMod"] = 1 + skillModList.Sum(mod.TypeIncrease, cfg, "EnemyChillDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration")/100
			chillMax, ok := overrideFloat(skillModList.Override(nil, "ChillMax"))
			if !ok {
				chillMax = gameConstant("ChillMaxEffect")
			}
			output["ChillSourceEffect"] = math.Min(chillMax, math.Floor(nonDamagingAilmentChillDefault*output["ChillEffectMod"]))
		}
		ailmentModeText := "Average Damage"
		if ailmentMode == "CRIT" {
			ailmentModeText = "Crits Only"
		}
		if output["FreezeChanceOnHit"]+output["FreezeChanceOnCrit"] > 0 {
			if globalBreakdown != nil {
				globalBreakdown.set("FreezeDurationMod", sFormat("Ailment mode: %s ^8(can be changed in the Configuration tab)", ailmentModeText))
			}
			hitDmg, critDmg := calcAverageUnmitigatedSourceDamage("Freeze", nil)
			baseVal := calcAilmentDamage("Freeze", output["CritChance"], hitDmg, critDmg, false) * skillModList.More(cfg, "FreezeAsThoughDealing")
			if baseVal > 0 {
				skillFlags["freeze"] = true
				output["FreezeDurationMod"] = 1 + skillModList.Sum(mod.TypeIncrease, cfg, "EnemyFreezeDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration")/100 + enemyDB.Sum(mod.TypeIncrease, nil, "SelfFreezeDuration", "SelfElementalAilmentDuration", "SelfAilmentDuration", "HoarfrostFreezeDuration")/100
				if breakdown != nil {
					breakdown.insert("FreezeDPS", sFormat("For freeze to apply for the minimum of 0.3 seconds, target must have no more than %.0f Ailment Threshold.", baseVal*20*output["FreezeDurationMod"]))
				}
			}
		}
		// pairs(nonDamagingAilmentsConfig) visits Chill and Shock; each writes its own keys.
		for _, ailment := range []string{"Chill", "Shock"} {
			if output[ailment+"ChanceOnHit"]+output[ailment+"ChanceOnCrit"] > 0 {
				hitDmg, critDmg := calcAverageUnmitigatedSourceDamage(ailment, defaultAilmentScalesFrom[ailment])
				if globalBreakdown != nil {
					globalBreakdown.set(ailment+"EffectMod", sFormat("Ailment mode: %s ^8(can be changed in the Configuration tab)", ailmentModeText))
				}
				damage := calcAilmentDamage(ailment, output["CritChance"], hitDmg, critDmg, false) * skillModList.More(cfg, ailment+"AsThoughDealing")
				if damage > 0 {
					skillFlags[strings.ToLower(ailment)] = true
					incDur := skillModList.Sum(mod.TypeIncrease, cfg, "Enemy"+ailment+"Duration", "EnemyElementalAilmentDuration", "EnemyAilmentDuration") + enemyDB.Sum(mod.TypeIncrease, nil, "Self"+ailment+"Duration", "SelfElementalAilmentDuration", "SelfAilmentDuration")
					moreDur := skillModList.More(cfg, "Enemy"+ailment+"Duration", "EnemyElementalAilmentDuration", "EnemyAilmentDuration") * enemyDB.More(nil, "Self"+ailment+"Duration", "SelfElementalAilmentDuration", "SelfAilmentDuration")
					data, _ := nonDamagingAilmentData(ailment)
					output[ailment+"Duration"] = data.duration * (1 + incDur/100) * moreDur * debuffDurationMult
					output[ailment+"EffectMod"] = calcLibMod(skillModList, cfg, "Enemy"+ailment+"Magnitude", "AilmentMagnitude") * calcLibMod(enemyDB, cfg, "Self"+ailment+"Magnitude", "AilmentMagnitude")
					//pob2:block:src/Modules/CalcOffence.lua:5732-5802
					if breakdown != nil {
						val := nonDamagingAilmentsConfig[ailment]
						maximum, ok := globalOutput["Maximum"+ailment]
						if !ok {
							maximum = data.max
						}
						current := math.Max(math.Min(globalOutput["Current"+ailment], maximum), 0)
						desired := math.Max(math.Min(enemyDB.Sum(mod.TypeBase, nil, "Desired"+ailment+"Val"), maximum), 0)
						effList := append([]float64(nil), val.effList...)
						if data.min != 0 {
							effList = append(effList, data.min)
						}
						if enemyThreshold > 0 {
							effList = append(effList, val.effect(damage, output[ailment+"EffectMod"], 0))
						}
						if !slices.Contains(effList, maximum) {
							effList = append(effList, maximum)
						}
						if current > 0 && !slices.Contains(effList, current) {
							effList = append(effList, current)
						}
						if desired > 0 && !slices.Contains(effList, desired) && current == 0 {
							effList = append(effList, desired)
						}
						e := breakdown.Entries[ailment+"DPS"]
						e.Label = "Resulting ailment effect"
						if current > 0 && val.ramping {
							e.Label += sFormat(" ^8(with a ^7%s%% ^8%s on the enemy)^7", current, ailment)
						}
						e.RowList = []map[string]string{}
						e.ColList = []BreakdownCol{{"Ailment Threshold", "thresh"}, {ailment + " Effect", "effect"}}
						sort.Float64s(effList)
						boss := "Pinnacle"
						if v, ok := env.ConfigInput["enemyIsBoss"]; ok && v.IsString {
							boss = enemyIsBossInput(v.String)
						}
						for _, value := range effList {
							thresh := val.thresh(damage, value, output[ailment+"EffectMod"])
							decCheck := value / math.Floor(value)
							value = math.Floor(value*math.Pow(10, data.precision)) / math.Pow(10, data.precision)
							valueFormat := "%." + luaToString(data.precision) + "f%%"
							threshString := sFormat("%d", thresh)
							if math.Floor(thresh+0.5) == math.Floor(enemyThreshold+0.5) {
								threshString += sFormat(" ^8(%s)", boss)
							}
							var labels []string
							if decCheck == 1 && value != 0 {
								if value == current {
									labels = append(labels, "current")
								}
								if value == desired {
									labels = append(labels, "desired")
								}
								if value == maximum {
									labels = append(labels, "maximum")
								}
								if value == data.min {
									labels = append(labels, "minimum")
								}
							}
							effect := sFormat(valueFormat, value)
							if len(labels) > 0 {
								effect += " ^8(" + strings.Join(labels, ", ") + ")"
							}
							e.RowList = append(e.RowList, map[string]string{"effect": effect, "thresh": threshString})
						}
					}
					if breakdown != nil && output[ailment+"Duration"] != data.duration {
						durKey := ailment + "Duration"
						breakdown.set(durKey)
						if isAttack {
							breakdown.insert(durKey, pass.label+":")
						}
						breakdown.insert(durKey, sFormat("%.2fs ^8(base duration)", data.duration))
						if incDur != 0 {
							breakdown.insert(durKey, sFormat("x %.2f ^8(increased/reduced duration)", 1+incDur/100))
						}
						if moreDur != 1 {
							breakdown.insert(durKey, sFormat("x %.2f ^8(more/less duration)", moreDur))
						}
						if debuffDurationMult != 1 {
							breakdown.insert(durKey, sFormat("/ %.2f ^8(debuff expires slower/faster)", 1/debuffDurationMult))
						}
						breakdown.insert(durKey, sFormat("= %.2fs", output[durKey]))
					}
				}
			}
		}

		// Calculate knockback chance/distance
		output["KnockbackChance"] = math.Min(100, float64(output["KnockbackChanceOnHit"]*(1-output["CritChance"]/100))+output["KnockbackChanceOnCrit"]*output["CritChance"]/100+enemyDB.Sum(mod.TypeBase, nil, "SelfKnockbackChance"))
		if output["KnockbackChance"] > 0 {
			output["KnockbackDistance"] = round(4 * calcLibMod(skillModList, cfg, "EnemyKnockbackDistance"))
		}

		// Calculate enemy stun modifiers
		enemyStunThresholdRed := -skillModList.Sum(mod.TypeIncrease, cfg, "EnemyStunThreshold")
		if enemyStunThresholdRed > 75 {
			output["EnemyStunThresholdMod"] = 1 - (75+(enemyStunThresholdRed-75)*25/(enemyStunThresholdRed-50))/100
		} else {
			output["EnemyStunThresholdMod"] = 1 - enemyStunThresholdRed/100
		}
		base, ok := sdNum(skillData, "baseStunDuration")
		if !ok {
			base = 0.35
		}
		incDur := skillModList.Sum(mod.TypeIncrease, cfg, "EnemyStunDuration")
		incDurCrit := skillModList.Sum(mod.TypeIncrease, cfg, "EnemyStunDurationOnCrit")
		moreDur := skillModList.More(cfg, "EnemyStunDuration")
		chanceToDouble := math.Min(skillModList.Sum(mod.TypeBase, cfg, "DoubleEnemyStunDurationChance")+enemyDB.Sum(mod.TypeBase, cfg, "SelfDoubleStunDurationChance"), 100)
		incRecov := enemyDB.Sum(mod.TypeIncrease, nil, "StunRecovery")
		minimumStunDuration := base * moreDur / (1 + incRecov/100)
		output["EnemyStunDuration"] = minimumStunDuration
		if incDurCrit != 0 && output["CritChance"] != 0 {
			if output["CritChance"] == 100 {
				minimumStunDuration *= 1 + (incDur+incDurCrit)/100
				output["EnemyStunDuration"] = minimumStunDuration
			} else {
				output["EnemyStunDuration"] *= 1 + (incDur+incDurCrit*output["CritChance"]/100)/100
			}
		} else {
			minimumStunDuration *= 1 + incDur/100
			output["EnemyStunDuration"] = minimumStunDuration
		}
		if chanceToDouble != 0 {
			output["EnemyStunDuration"] *= 1 + chanceToDouble/100
		}

		// Calculate impale chance and modifiers
		if canDeal["Physical"] && output["ImpaleChance"]+output["ImpaleChanceOnCrit"] > 0 {
			skillFlags["impale"] = true
			critChance := output["CritChance"] / 100
			impaleChance := float64(math.Min(output["ImpaleChance"]/100, 1)*(1-critChance)) + float64(math.Min(output["ImpaleChanceOnCrit"]/100, 1)*critChance)
			maxStacks := skillModList.Sum(mod.TypeBase, cfg, "ImpaleStacksMax") * (1 + skillModList.Sum(mod.TypeBase, cfg, "ImpaleAdditionalDurationChance")/100)
			configStacks := enemyDB.Sum(mod.TypeBase, cfg, "Multiplier:ImpaleStacks")
			impaleStacks := math.Min(maxStacks, configStacks)

			baseStoredDamage := miscConstant("ImpaleStoredDamageBase")
			storedExpectedDamageIncOnBleed := skillModList.Sum(mod.TypeIncrease, cfg, "ImpaleEffectOnBleed") * skillModList.Sum(mod.TypeBase, cfg, "BleedChance") / 100
			storedExpectedDamageInc := (skillModList.Sum(mod.TypeIncrease, cfg, "ImpaleEffect") + storedExpectedDamageIncOnBleed) / 100
			storedExpectedDamageMore := round(skillModList.More(cfg, "ImpaleEffect"), 2)
			storedExpectedDamageModifier := (1 + storedExpectedDamageInc) * storedExpectedDamageMore
			impaleStoredDamage := baseStoredDamage * storedExpectedDamageModifier
			impaleHitDamageMod := impaleStoredDamage * impaleStacks // Source: https://www.reddit.com/r/pathofexile/comments/chgqqt/impale_and_armor_interaction/

			enemyArmour := math.Max(calcLibVal(enemyDB, "Armour", nil), 0)
			impaleArmourReduction := armourReductionF(enemyArmour, impaleHitDamageMod*output["PhysicalStoredCombinedAvg"])
			impaleResist := math.Min(math.Max(0, enemyDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction")+skillModList.Sum(mod.TypeBase, cfg, "EnemyImpalePhysicalDamageReduction")+impaleArmourReduction), miscConstant("EnemyPhysicalDamageReductionCap"))
			if skillModList.Flag(cfg, "IgnoreEnemyImpalePhysicalDamageReduction") {
				impaleResist = 0
			}
			impaleTakenCfg := flagsCfg(ModFlagHit)
			impaleTaken := (1 + enemyDB.Sum(mod.TypeIncrease, impaleTakenCfg, "DamageTaken", "PhysicalDamageTaken", "ReflectedDamageTaken")/100) *
				enemyDB.More(impaleTakenCfg, "DamageTaken", "PhysicalDamageTaken", "ReflectedDamageTaken")
			impaleDMGModifier := float64(impaleHitDamageMod * (1 - impaleResist/100) * impaleChance * impaleTaken)

			globalOutput["ImpaleStacksMax"] = maxStacks
			globalOutput["ImpaleStacks"] = impaleStacks
			//ImpaleStoredDamage should be named ImpaleEffect or similar
			//Using the variable name ImpaleEffect breaks the calculations sidebar (?!)
			output["ImpaleStoredDamage"] = impaleStoredDamage * 100
			output["ImpaleModifier"] = 1 + impaleDMGModifier
		}
	}

	// Combine secondary effect stats
	if isAttack {
		output := globalOutput
		combine := func(stat, mode string, extra ...string) {
			state.combineStat(output, actor.Breakdown, skillFlags, skillData, stat, mode, extra...)
		}
		for _, ailment := range []string{"Bleed", "Poison", "Ignite"} {
			combine(ailment+"Chance", "AVERAGE")
			combine(ailment+"DPS", "CHANCE_AILMENT", ailment+"Chance")
			if skillFlags[strings.ToLower(ailment)+"CanStack"] {
				combine(ailment+"Damage", "CHANCE", ailment+"Chance")
				combine("Total"+ailment+"DPS", "DPS")
				combine(ailment+"StacksMax", "DPS")
			}
		}

		combine("ChillEffectMod", "AVERAGE")
		combine("ChillDuration", "AVERAGE")
		combine("ShockChance", "AVERAGE")
		combine("ShockDuration", "AVERAGE")
		combine("ShockEffectMod", "AVERAGE")
		combine("FreezeChance", "AVERAGE")
		combine("FreezeDurationMod", "AVERAGE")
		combine("ImpaleChance", "AVERAGE")
		combine("ImpaleStoredDamage", "AVERAGE")
		combine("ImpaleModifier", "CHANCE", "ImpaleChance")
	}
}

// offenceDots calculates Decay, the burning ground mods drop and the skill's own damage over time.
// The breakdowns are not ported.
//
//pob2:block:src/Modules/CalcOffence.lua:5966-6181
func (h *offenceHit) offenceDots() {
	env, actor, activeSkill, state, skillFlags := h.env, h.actor, h.activeSkill, h.state, h.skillFlags
	output := actor.Output
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	canDeal := state.canDeal
	dotDpsCap := miscConstant("DotDpsCap")

	debuffDurationMult := 1.0
	if env.ModeEffective {
		debuffDurationMult = calcBuffExpirationMult(enemyDB, skillCfg)
	}

	if decay, ok := sdNum(skillData, "decay"); ok && canDeal["Chaos"] {
		// Calculate DPS for Decay effect
		skillFlags["decay"] = true
		dotCfg := newSkillCfg(skillCfg, ModFlagDot, (skillCfg.KeywordFlags.Get()&^KeywordFlagHit)|KeywordFlagChaosDot)
		activeSkill.DecayCfg = dotCfg
		effMult := 1.0
		if env.ModeEffective {
			resist := calcResistForType(env, enemyDB, "Chaos", dotCfg)
			takenInc := enemyDB.Sum(mod.TypeIncrease, nil, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
			takenMore := enemyDB.More(nil, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
			effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			output["DecayEffMult"] = effMult
		}
		inc := skillModList.Sum(mod.TypeIncrease, dotCfg, "Damage", "ChaosDamage")
		more := skillModList.More(dotCfg, "Damage", "ChaosDamage")
		// dotTypeCfg is not a local here, so the Lua reads the nil global.
		mult, ok := overrideFloat(skillModList.Override(nil, "DotMultiplier"))
		if !ok {
			mult = skillModList.Sum(mod.TypeBase, nil, "DotMultiplier") + skillModList.Sum(mod.TypeBase, nil, "ChaosDotMultiplier")
		}
		output["DecayDPS"] = decay * (1 + inc/100) * more * (1 + mult/100) * effMult
		output["DecayDuration"] = 8 * debuffDurationMult
	}

	baseDropsBurningGround := modDB.Sum(mod.TypeBase, nil, "DropsBurningGround")
	if baseDropsBurningGround > 0 && canDeal["Fire"] {
		dotTakenCfg := flagsKeywordCfg(ModFlagDot, 0)
		dotTypeCfg := flagsKeywordCfg(ModFlagDot, KeywordFlagFireDot)
		effMult := 1.0
		if env.ModeEffective {
			resist := calcResistForType(env, enemyDB, "Fire", dotTypeCfg)
			takenInc := enemyDB.Sum(mod.TypeIncrease, dotTakenCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
			takenMore := enemyDB.More(dotTakenCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
			effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
		}
		inc := modDB.Sum(mod.TypeIncrease, dotTypeCfg, "Damage", "FireDamage", "ElementalDamage")
		more := modDB.More(dotTypeCfg, "Damage", "FireDamage", "ElementalDamage")
		mult, ok := overrideFloat(modDB.Override(dotTypeCfg, "DotMultiplier"))
		if !ok {
			mult = modDB.Sum(mod.TypeBase, dotTypeCfg, "DotMultiplier") + modDB.Sum(mod.TypeBase, dotTypeCfg, "FireDotMultiplier")
		}
		total := baseDropsBurningGround * (1 + inc/100) * more * (1 + mult/100) * effMult
		if v, ok := output["BurningGroundDPS"]; !ok || v < total {
			output["BurningGroundDPS"] = total
		}
	}

	// Calculate skill DOT components
	dotFlags := ModFlagDot | skillCfg.Flags.Get()
	if dotFlags&ModFlagArea != 0 && !truthy(skillData["dotIsArea"]) {
		dotFlags &^= ModFlagArea
	}
	if dotFlags&ModFlagProjectile != 0 && !truthy(skillData["dotIsProjectile"]) {
		dotFlags &^= ModFlagProjectile
	}
	if dotFlags&ModFlagSpell != 0 && !truthy(skillData["dotIsSpell"]) {
		dotFlags &^= ModFlagSpell
	}
	if dotFlags&ModFlagAttack != 0 && !truthy(skillData["dotIsAttack"]) {
		dotFlags &^= ModFlagAttack
	}
	if dotFlags&ModFlagHit != 0 && !truthy(skillData["dotIsHit"]) {
		dotFlags &^= ModFlagHit
	}
	dotCfg := newSkillCfg(skillCfg, dotFlags, skillCfg.KeywordFlags.Get()&^KeywordFlagHit)

	// spell_damage_modifiers_apply_to_skill_dot does not apply to enemy damage taken
	dotTakenCfg := copyCfgShallow(dotCfg)
	if truthy(skillData["dotIsSpell"]) {
		flags := dotTakenCfg.Flags.Get() &^ ModFlagSpell
		dotTakenCfg.Flags = &flags
	}

	activeSkill.DotCfg = dotCfg
	output["TotalDotInstance"] = 0

	// runSkillFunc("preDotFunc"): no PoE2 skill defines one.

	// Section Handles Generic Damage over time [DOT]
	for _, damageType := range dmgTypeList {
		dotTypeCfg := copyCfgShallow(dotCfg)
		kw := dotTypeCfg.KeywordFlags.Get() | ailmentKeywordFlags[damageType+"Dot"]
		dotTypeCfg.KeywordFlags = &kw
		if activeSkill.DotTypeCfg == nil {
			activeSkill.DotTypeCfg = map[string]*moddb.ListCfg{}
		}
		activeSkill.DotTypeCfg[damageType] = dotTypeCfg
		baseVal := 0.0
		if canDeal[damageType] {
			baseVal, _ = sdNum(skillData, damageType+"Dot")
		}
		existing, hasExisting := output[damageType+"Dot"]
		if baseVal > 0 || existing > 0 {
			skillFlags["dot"] = true
			effMult := 1.0
			// Section handles Enemy Damage Taken based on Configs
			if env.ModeEffective {
				resist := 0.0
				takenInc := enemyDB.Sum(mod.TypeIncrease, dotTakenCfg, "DamageTaken", "DamageTakenOverTime", damageType+"DamageTaken", damageType+"DamageTakenOverTime")
				takenMore := enemyDB.More(dotTakenCfg, "DamageTaken", "DamageTakenOverTime", damageType+"DamageTaken", damageType+"DamageTakenOverTime")
				if isElemental[damageType] {
					takenInc += enemyDB.Sum(mod.TypeIncrease, dotTakenCfg, "ElementalDamageTaken")
					takenMore *= enemyDB.More(dotTakenCfg, "ElementalDamageTaken")
				}
				if damageType == "Physical" {
					resist = math.Max(0, math.Min(enemyDB.Sum(mod.TypeBase, nil, "PhysicalDamageReduction"), miscConstant("EnemyPhysicalDamageReductionCap")))
				} else {
					resist = calcResistForType(env, enemyDB, damageType, dotTypeCfg)
				}
				effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
				output[damageType+"DotEffMult"] = effMult
			}
			// Variables below calculate DOT damage
			names := []string{"Damage", damageType + "Damage"}
			if isElemental[damageType] {
				names = append(names, "ElementalDamage")
			}
			inc := skillModList.Sum(mod.TypeIncrease, dotTypeCfg, names...)
			if skillModList.Flag(nil, "dotIsHeraldOfAsh") {
				inc -= skillModList.Sum(mod.TypeIncrease, skillCfg, names...)
			}
			more := skillModList.More(dotTypeCfg, names...)
			mult, ok := overrideFloat(skillModList.Override(dotTypeCfg, "DotMultiplier"))
			if !ok {
				mult = skillModList.Sum(mod.TypeBase, dotTypeCfg, "DotMultiplier") + skillModList.Sum(mod.TypeBase, dotTypeCfg, damageType+"DotMultiplier")
			}
			aura := 1.0
			if activeSkill.SkillTypes["Aura"] && !activeSkill.SkillTypes["RemoteMined"] {
				aura = calcLibMod(skillModList, dotTypeCfg, "AuraEffect") * calcLibMod(skillModList, skillCfg, "Magnitude")
			}
			total := float64(baseVal * (1 + inc/100) * more * (1 + mult/100) * aura * effMult)
			if !hasExisting || existing == 0 {
				output[damageType+"Dot"] = total
				output["TotalDotInstance"] = math.Min(output["TotalDotInstance"]+total, dotDpsCap)
			} else {
				output["TotalDotInstance"] = math.Min(output["TotalDotInstance"]+total+existing, dotDpsCap)
			}
		}
	}

	switch {
	case skillModList.Flag(nil, "DotCanStack"):
		skillFlags["DotCanStack"] = true
		speed := output["Speed"]
		// Check if skill is being triggered via Mine (e.g., Blastchain Mine Support) or Trap
		// if "yes", you cannot use output.Speed but rather should use output.MineLayingSpeed or output.TrapThrowingSpeed
		if dotCfg.KeywordFlags.Get()&KeywordFlagMine != 0 {
			speed = output["MineLayingSpeed"]
		} else if dotCfg.KeywordFlags.Get()&KeywordFlagTrap != 0 {
			speed = output["TrapThrowingSpeed"]
		}
		output["TotalDot"] = math.Min(output["TotalDotInstance"]*speed*output["Duration"]*output["DpsMultiplier"]*state.quantityMultiplier, dotDpsCap)
		output["TotalDotCalcSection"] = output["TotalDot"]
	case skillModList.Flag(nil, "dotIsBurningGround"):
		output["TotalDot"] = 0
		output["TotalDotCalcSection"] = output["TotalDotInstance"]
		if v, ok := output["BurningGroundDPS"]; !ok || v < output["TotalDotInstance"] {
			output["BurningGroundDPS"] = math.Max(v, output["TotalDotInstance"])
		}
	case skillModList.Flag(nil, "dotIsCausticGround"):
		output["TotalDot"] = 0
		output["TotalDotCalcSection"] = output["TotalDotInstance"]
		if v, ok := output["CausticGroundDPS"]; !ok || v < output["TotalDotInstance"] {
			output["CausticGroundDPS"] = math.Max(v, output["TotalDotInstance"])
		}
	case skillModList.Flag(nil, "dotIsCorruptingBlood"):
		output["TotalDot"] = 0
		output["TotalDotCalcSection"] = output["TotalDotInstance"]
		if v, ok := output["CorruptingBloodDPS"]; !ok || v < output["TotalDotInstance"] {
			output["CorruptingBloodDPS"] = math.Max(v, output["TotalDotInstance"])
		}
	default:
		if skillModList.Flag(nil, "DotCanStackAsTotems") && skillFlags["totem"] {
			skillFlags["DotCanStack"] = true
		}
		output["TotalDot"] = output["TotalDotInstance"]
		output["TotalDotCalcSection"] = output["TotalDotInstance"]
	}
}

// offenceLinkCosts calculates cost per second for skills that don't already have one (link skills).
// hasCosts is output[resource.."HasCost"] and output[resource.."PerSecondHasCost"].
//
//pob2:block:src/Modules/CalcOffence.lua:6183-6226
func (h *offenceHit) offenceLinkCosts(hasCosts map[string]bool) {
	env, actor, activeSkill, skillFlags := h.env, h.actor, h.activeSkill, h.skillFlags
	output := actor.Output
	skillModList := activeSkill.SkillModList
	for _, resource := range costOrder {
		EB := env.ModDB.Flag(nil, "EnergyShieldProtectsMana")
		// `EB and Sum(...)` is EB: Sum is a number, and every number is truthy.
		_, hasCooldown := output["Cooldown"]
		if costUpfront[resource] && hasCosts[resource] && output[resource+"Cost"] > 0 && !(hasCosts[resource+"PerSecond"] && !EB) && (output["Speed"] > 0 || hasCooldown) {
			usedResource := resource
			if EB && resource == "Mana" {
				usedResource = "ES"
			}

			repeats := outputDefault(output, "Repeats", 1)
			var useSpeed float64
			cooldown, cooldownOK := output["Cooldown"]
			switch {
			case skillFlags["trap"] || skillFlags["mine"]:
				preSpeed := outputDefault(output, "TrapThrowingSpeed", output["MineLayingSpeed"])
				if c, ok := output["TrapCooldown"]; ok {
					cooldown, cooldownOK = c, true
				}
				if cooldownOK && cooldown > 0 {
					useSpeed = 1 / cooldown
				} else {
					useSpeed = preSpeed
				}
			case skillFlags["totem"]:
				if cooldownOK && cooldown > 0 {
					if output["TotemPlacementSpeed"] > 0 {
						useSpeed = output["TotemPlacementSpeed"]
					} else {
						useSpeed = 1 / cooldown
					}
				} else {
					useSpeed = output["TotemPlacementSpeed"]
				}
				useSpeed /= repeats
			case skillModList.Flag(nil, "HasSeals") && skillModList.Flag(nil, "DamageSeal") && skillModList.Flag(nil, "UseMaxUnleash"):
				hitTimeOverride, _ := sdNum(env.Player.MainSkill.SkillData, "hitTimeOverride")
				useSpeed = hitTimeOverride / repeats
			default:
				if reload, ok := output["EffectiveReloadTime"]; ok { // Crossbows: Account for mana cost only happening on reload (once all bolts are fired)
					if _, ok := output["EffectiveBoltCount"]; ok {
						useSpeed = 1 / (output["TotalFiringTime"] + reload)
					}
				} else {
					if cooldownOK && cooldown > 0 {
						if output["Speed"] > 0 {
							useSpeed = output["Speed"]
						} else {
							useSpeed = 1 / cooldown
						}
					} else {
						useSpeed = output["Speed"]
					}
					useSpeed /= repeats
				}
			}

			hasCosts[usedResource+"PerSecond"] = true
			actor.setOutputFlag(usedResource+"PerSecondHasCost", true)
			output[usedResource+"PerSecondCost"] += float64(output[resource+"Cost"] * useSpeed)
		}
	}
}

// offenceCombinedDPS calculates the combined DPS estimate, including DoTs. Self hit damage
// (selfHitHandlers) and mirages are not ported.
//
//pob2:block:src/Modules/CalcOffence.lua:6264-6470
func (h *offenceHit) offenceCombinedDPS() {
	output := h.actor.Output
	state, skillFlags := h.state, h.skillFlags
	skillModList, skillCfg, skillData := h.activeSkill.SkillModList, h.activeSkill.SkillCfg, h.activeSkill.SkillData
	enemyDB := h.actor.Enemy.ModDB
	showAverage := truthy(skillData["showAverage"])
	baseDPS := output["TotalDPS"]
	if showAverage {
		baseDPS = output["AverageDamage"]
	}
	output["CombinedDPS"] = baseDPS
	output["CombinedAvg"] = baseDPS
	if skillFlags["dot"] {
		output["WithDotDPS"] = baseDPS + output["TotalDot"]
	}

	for _, ailment := range []string{"Bleed", "Poison", "Ignite"} {
		switch {
		case !skillFlags[strings.ToLower(ailment)]:
			output["With"+ailment+"DPS"] = baseDPS
		case skillFlags[strings.ToLower(ailment)+"CanStack"]:
			if showAverage {
				output["CombinedAvg"] = output["CombinedDPS"] + output[ailment+"Damage"]
			} else {
				output["With"+ailment+"DPS"] = baseDPS + output["Total"+ailment+"DPS"]
			}
		case showAverage:
			output["With"+ailment+"DPS"] = baseDPS + output[ailment+"Damage"]
			output["CombinedAvg"] += output[ailment+"Damage"]
		default:
			output["With"+ailment+"DPS"] = baseDPS + output[ailment+"DPS"]
		}
	}
	if skillFlags["monsterExplode"] {
		output["CombinedAvgToMonsterLife"] = output["CombinedAvg"] / state.monsterLife * 100
	}
	// Parry Stats
	// NOTE: This section is mainly for skill-specific breakdowns. Actual application of damage modifier is handled in `CalcPerform`
	parryDebuffMagnitudeMod := calcLibMod(skillModList, skillCfg, "ParryDebuffMagnitude")
	if _, ok := skillData["parryDebuffBaseMagnitude"]; ok && parryDebuffMagnitudeMod != 1 {
		output["ParryDebuffMagnitudeMod"] = parryDebuffMagnitudeMod
	}
	if d, ok := sdNum(skillData, "parryDebuffDuration"); ok && d > 0 {
		expirationMult := calcBuffExpirationMult(enemyDB, skillCfg)
		output["ParryDebuffDuration"] = d * calcLibMod(skillModList, skillCfg, "ParryDebuffDuration") * expirationMult
	}
	nonProj, hasNonProj := sdNum(skillData, "parryRangeNonProj")
	proj, hasProj := sdNum(skillData, "parryRangeProj")
	if hasNonProj || hasProj {
		output["ParryRangeNonProj"] = nonProj * calcLibMod(skillModList, skillCfg, "ParryRangeNonProj")
		output["ParryRangeProj"] = proj * calcLibMod(skillModList, skillCfg, "ParryRangeProj")
	}
	if skillFlags["impale"] {
		if skillFlags["attack"] && truthy(skillData["combinesHitsWhenDualWielding"]) && skillFlags["bothWeaponAttack"] {
			// separately combine
			mainHandImpaleDPS := float64(state.mainHand["impaleStoredHitAvg"] * (outputDefault(state.mainHand, "ImpaleModifier", 1) - 1) * state.mainHand["HitChance"] / 100 * output["DpsMultiplier"])
			offHandImpaleDPS := float64(state.offHand["impaleStoredHitAvg"] * (outputDefault(state.offHand, "ImpaleModifier", 1) - 1) * state.offHand["HitChance"] / 100 * output["DpsMultiplier"])
			output["ImpaleDPS"] = mainHandImpaleDPS + offHandImpaleDPS
		} else {
			output["ImpaleDPS"] = output["PhysicalStoredCombinedAvg"] * (outputDefault(output, "ImpaleModifier", 1) - 1) * output["HitChance"] / 100 * output["DpsMultiplier"]
		}
		if showAverage {
			output["WithImpaleDPS"] = output["AverageDamage"] + output["ImpaleDPS"]
			output["CombinedAvg"] += output["ImpaleDPS"]
		} else {
			skillFlags["notAverage"] = true
			output["ImpaleDPS"] *= outputDefault(output, "HitSpeed", output["Speed"])
			output["WithImpaleDPS"] = output["TotalDPS"] + output["ImpaleDPS"]
		}
		if state.quantityMultiplier > 1 {
			output["ImpaleDPS"] *= state.quantityMultiplier
		}
		output["CombinedDPS"] += output["ImpaleDPS"]
	}

	bestCull := 1.0

	totalDotDPS := output["TotalDot"] +
		outputDefault(output, "TotalPoisonDPS", output["PoisonDPS"]) +
		math.Max(output["CausticGroundDPS"], output["MirageCausticGroundDPS"]) +
		outputDefault(output, "TotalIgniteDPS", output["IgniteDPS"]) +
		math.Max(output["BurningGroundDPS"], output["MirageBurningGroundDPS"]) +
		outputDefault(output, "TotalBleedDPS", output["BleedDPS"]) +
		output["CorruptingBloodDPS"] +
		output["DecayDPS"]

	output["TotalDotDPS"] = math.Min(totalDotDPS, miscConstant("DotDpsCap"))
	if output["TotalDotDPS"] != totalDotDPS {
		h.actor.setOutputFlag("showTotalDotDPS", true)
	}
	if !showAverage {
		output["CombinedDPS"] += output["TotalDotDPS"]
	}

	bestCull = math.Max(bestCull, output["CullMultiplier"])
	output["CullingDPS"] = output["CombinedDPS"] * (bestCull - 1)
	output["ReservationDPS"] = output["CombinedDPS"] * (output["ReservationDpsMultiplier"] - 1)
	output["CombinedDPS"] = output["CombinedDPS"] * bestCull * output["ReservationDpsMultiplier"]
}
