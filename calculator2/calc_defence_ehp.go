package calculator2

import (
	"math"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

var hitSourceList = []string{"Attack", "Spell"}

//pob2:src/Modules/CalcDefence.lua:57-65
func armourReductionF(armour, raw float64) float64 {
	if armour == 0 && raw == 0 {
		return 0
	} else if armour < 0 {
		armour = -armour
		return -(armour / (armour + raw*miscConstant("ArmourRatio")) * 100)
	}
	return armour / (armour + raw*miscConstant("ArmourRatio")) * 100
}

//pob2:src/Modules/CalcDefence.lua:68-70
func armourReduction(armour, raw float64) float64 {
	return round(armourReductionF(armour, raw))
}

// ehpActor is the actor with the shift tables buildDefenceEstimations stores on it.
type ehpActor struct {
	*Actor
	damageShiftTable         map[string]map[string]float64
	damageOverTimeShiftTable map[string]map[string]float64
}

// outputOr is `output[a] or output[b]`.
func outputOr(output map[string]float64, a, b string) float64 {
	if v, ok := output[a]; ok {
		return v
	}
	return output[b]
}

//pob2:src/Modules/CalcDefence.lua:427-436
func damageMitigationMultiplierForType(output map[string]float64, modDB *moddb.ModDB, damageType string, damage float64, typ string) float64 {
	effectiveAppliedArmour := output[typ+"EffectiveAppliedArmour"]
	armourDRPercent := armourReductionF(effectiveAppliedArmour, damage)
	flatDRPercent := 0.0
	if !modDB.Flag(nil, "SelfIgnoreBase"+typ+"DamageReduction") {
		flatDRPercent = outputOr(output, "Base"+typ+"DamageReductionWhenHit", "Base"+typ+"DamageReduction")
	}
	totalDRPercent := math.Min(output[damageType+"DamageReductionMax"], armourDRPercent+flatDRPercent)
	enemyOverwhelmPercent := 0.0
	if !modDB.Flag(nil, "SelfIgnore"+typ+"DamageReduction") {
		enemyOverwhelmPercent = output[typ+"EnemyOverwhelm"]
	}
	totalDRMulti := 1 - math.Max(math.Min(output[damageType+"DamageReductionMax"], totalDRPercent-enemyOverwhelmPercent), 0)/100
	totalResistMult := output[typ+"ResistTakenHitMulti"]
	return totalResistMult * totalDRMulti
}

// takenHitFromDamage returns the damages as a map holding only the types Lua leaves non-nil.
//
//pob2:src/Modules/CalcDefence.lua:443-460
func takenHitFromDamage(rawDamage float64, damageType string, actor *ehpActor) (float64, map[string]float64) {
	output := actor.Output
	modDB := actor.ModDB
	receivedDamageSum := 0.0
	damages := map[string]float64{}
	for _, damageConvertedType := range dmgTypeList {
		convertPercent := actor.damageShiftTable[damageType][damageConvertedType]
		takenFlat := output[damageConvertedType+"takenFlat"]
		if convertPercent > 0 || takenFlat != 0 {
			convertedDamage := rawDamage * convertPercent / 100
			vaalArctic := math.Min(-modDB.Sum(mod.TypeMore, nil, "VaalArcticArmourMitigation")/100, 1)
			reducedDamage := round(math.Max(convertedDamage*damageMitigationMultiplierForType(output, modDB, damageType, convertedDamage, damageConvertedType)+takenFlat, 0)*output[damageConvertedType+"AfterReductionTakenHitMulti"]) * (1 - vaalArctic)
			receivedDamageSum += reducedDamage
			if reducedDamage > 0 || convertPercent > 0 {
				damages[damageConvertedType] = reducedDamage
			}
		}
	}
	return receivedDamageSum, damages
}

//pob2:src/Modules/CalcDefence.lua:462-466
func calcLifeHitPoolWithLossPrevention(life, maxLife, lifeLossPrevented, lifeLossBelowHalfPrevented float64) float64 {
	halfLife := maxLife * 0.5
	aboveLow := math.Max(life-halfLife, 0)
	return aboveLow/(1-lifeLossPrevented/100) + math.Min(life, halfLife)/(1-lifeLossBelowHalfPrevented/100)/(1-lifeLossPrevented/100)
}

// ehpAlly is one alliesTakenBeforeYou entry.
type ehpAlly struct {
	name      string
	remaining float64
	percent   float64
}

// ehpPools is reducePoolsByDamage's poolTable and its result.
type ehpPools struct {
	alliesTakenBeforeYou          []*ehpAlly
	aegis                         map[string]float64
	guard                         map[string]float64
	ward                          float64
	energyShield                  float64
	mana                          float64
	life                          float64
	damageTakenThatCanBeRecouped  map[string]float64
	lifeLossLostOverTime          float64
	lifeBelowHalfLossLostOverTime float64
	overkillDamage                float64
	hitPoolRemaining              float64
	// resourcesLostToTypeDamage is what each damage type took from each resource; a missing key is the Lua's nil.
	// Only a fresh pool table records it, for a breakdown.
	resourcesLostToTypeDamage map[string]map[string]float64
}

// alliesTakenBeforeYou is kept in the order the Lua inserts the keys; Lua walks the table with pairs().
//
//pob2:block:src/Modules/CalcDefence.lua:481-502
//pob2:block:src/Modules/CalcDefence.lua:2083-2104
func alliesTakenBeforeYou(output map[string]float64) []*ehpAlly {
	var allies []*ehpAlly
	add := func(name, life, mitigation string) {
		if remaining, ok := output[life]; ok {
			allies = append(allies, &ehpAlly{name: name, remaining: remaining, percent: output[mitigation] / 100})
		}
	}
	add("frostShield", "FrostShieldLife", "FrostShieldDamageMitigation")
	add("spectres", "TotalSpectreLife", "SpectreAllyDamageMitigation")
	add("totems", "TotalTotemLife", "TotemAllyDamageMitigation")
	add("vaalRejuvenationTotems", "TotalVaalRejuvenationTotemLife", "VaalRejuvenationTotemAllyDamageMitigation")
	add("radianceSentinel", "TotalRadianceSentinelLife", "RadianceSentinelAllyDamageMitigation")
	add("companion", "TotalCompanionLife", "CompanionAllyDamageMitigation")
	add("soulLink", "AlliedEnergyShield", "SoulLinkMitigation")
	return allies
}

// reducePoolsByDamage takes a nil poolTbl for Lua's nil poolTable. resourcesLostToTypeDamage only feeds
// breakdowns, and is kept only when poolTbl is nil and the actor builds one.
//
//pob2:src/Modules/CalcDefence.lua:473-699
func reducePoolsByDamage(poolTbl *ehpPools, damageTable map[string]float64, actor *ehpActor) *ehpPools {
	output := actor.Output
	modDB := actor.ModDB

	var allies []*ehpAlly
	var damageTakenThatCanBeRecouped, aegis, guard map[string]float64
	var ward, energyShield, mana, life, lifeLossLostOverTime, lifeBelowHalfLossLostOverTime float64
	if poolTbl != nil {
		allies = poolTbl.alliesTakenBeforeYou
		damageTakenThatCanBeRecouped = poolTbl.damageTakenThatCanBeRecouped
		aegis = poolTbl.aegis
		guard = poolTbl.guard
		ward = poolTbl.ward
		energyShield = poolTbl.energyShield
		mana = poolTbl.mana
		life = poolTbl.life
		lifeLossLostOverTime = poolTbl.lifeLossLostOverTime
		lifeBelowHalfLossLostOverTime = poolTbl.lifeBelowHalfLossLostOverTime
	} else {
		allies = alliesTakenBeforeYou(output)
		damageTakenThatCanBeRecouped = map[string]float64{}
		aegis = map[string]float64{"shared": output["sharedAegis"], "sharedElemental": output["sharedElementalAegis"]}
		guard = map[string]float64{"shared": output["sharedGuardAbsorb"]}
		for damageType := range damageTable {
			aegis[damageType] = output[damageType+"Aegis"]
			guard[damageType] = output[damageType+"GuardAbsorb"]
		}
		ward = output["Ward"]
		energyShield = output["EnergyShieldRecoveryCap"]
		mana = output["ManaUnreserved"]
		life = output["LifeRecoverable"]
	}
	wardApplies := modDB.Sum(mod.TypeBase, nil, "WardBypass") == 0
	lifeLossBelowHalfPrevented := modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfPrevented")
	overkillDamage := 0.0
	var lost map[string]map[string]float64
	if poolTbl == nil && actor.Breakdown != nil {
		lost = map[string]map[string]float64{}
		for _, damageType := range dmgTypeList {
			lost[damageType] = map[string]float64{}
		}
	}
	// lose is `resourcesLostToTypeDamage[damageType][key] = v >= 1 and v or nil`
	lose := func(damageType, key string, v float64) {
		if lost == nil {
			return
		}
		if v >= 1 {
			lost[damageType][key] = v
		} else {
			delete(lost[damageType], key)
		}
	}
	MoMPoolRemaining := posInf
	esPoolRemaining := posInf

	damageRemaindersBeforeES := map[string]float64{}
	for _, damageType := range dmgTypeList {
		damageRemainder, ok := damageTable[damageType]
		if !ok {
			continue
		}
		for _, ally := range allies {
			if ally.remaining > 0 {
				tempDamage := math.Min(damageRemainder*ally.percent, ally.remaining)
				ally.remaining -= tempDamage
				damageRemainder -= tempDamage
				lose(damageType, ally.name, tempDamage)
			}
		}
		damageTakenThatCanBeRecouped[damageType] += damageRemainder
		if aegis[damageType] > 0 {
			tempDamage := math.Min(damageRemainder, aegis[damageType])
			aegis[damageType] -= tempDamage
			damageRemainder -= tempDamage
			lose(damageType, "aegis", tempDamage)
		}
		if isElemental[damageType] && aegis["sharedElemental"] > 0 {
			tempDamage := math.Min(damageRemainder, aegis["sharedElemental"])
			aegis["sharedElemental"] -= tempDamage
			damageRemainder -= tempDamage
			lose(damageType, "sharedElementalAegis", tempDamage)
		}
		if aegis["shared"] > 0 {
			tempDamage := math.Min(damageRemainder, aegis["shared"])
			aegis["shared"] -= tempDamage
			damageRemainder -= tempDamage
			lose(damageType, "sharedAegis", tempDamage)
		}
		if guard[damageType] > 0 {
			tempDamage := math.Min(damageRemainder*output[damageType+"GuardAbsorbRate"]/100, guard[damageType])
			guard[damageType] -= tempDamage
			damageRemainder -= tempDamage
			lose(damageType, "guard", tempDamage)
		}
		if guard["shared"] > 0 {
			tempDamage := math.Min(damageRemainder*output["sharedGuardAbsorbRate"]/100, guard["shared"])
			guard["shared"] -= tempDamage
			damageRemainder -= tempDamage
			lose(damageType, "sharedGuard", tempDamage)
		}
		if damageRemainder > 0 {
			damageRemaindersBeforeES[damageType] = damageRemainder
		}
	}

	for i := len(dmgTypeList) - 1; i >= 0; i-- {
		damageType := dmgTypeList[i]
		damageRemainder, ok := damageRemaindersBeforeES[damageType]
		if !ok {
			continue
		}
		esDamageTypeMultiplier := 1.0
		if damageType == "Chaos" && !modDB.Flag(nil, "ChaosNotDoubleESDamage") {
			esDamageTypeMultiplier = 2
		}
		esBypass := output[damageType+"EnergyShieldBypass"] / 100
		lifeHitPool := calcLifeHitPoolWithLossPrevention(life, output["Life"], output["preventedLifeLoss"], lifeLossBelowHalfPrevented)
		MoMEffect := math.Min(output["sharedMindOverMatter"]+output[damageType+"MindOverMatter"], 100) / 100
		MoMPool := mana
		if MoMEffect < 1 {
			MoMPool = math.Min(lifeHitPool/(1-MoMEffect)-lifeHitPool, mana)
		}
		if energyShield > 0 && modDB.Flag(nil, "EternalLife") {
			tempDamage := math.Min(damageRemainder, energyShield/(1-esBypass)/esDamageTypeMultiplier)
			energyShield -= tempDamage * (1 - esBypass) * esDamageTypeMultiplier
			esPoolRemaining = math.Min(esPoolRemaining, energyShield)
			damageRemainder -= tempDamage
			if lost != nil {
				delete(lost[damageType], "energyShield")
				delete(lost[damageType], "eternalLifePrevented")
				if tempDamage >= 1 {
					lost[damageType]["energyShield"] = tempDamage * (1 - esBypass) * esDamageTypeMultiplier
					lost[damageType]["eternalLifePrevented"] = tempDamage * esBypass
				}
			}
		} else if energyShield > 0 && esBypass < 1 {
			MoMEBPool := energyShield
			if esBypass > 0 {
				MoMEBPool = math.Min((MoMPool+lifeHitPool)/esBypass*esDamageTypeMultiplier-(MoMPool+lifeHitPool), energyShield)
			}
			tempDamage := math.Min(damageRemainder*(1-esBypass), MoMEBPool/esDamageTypeMultiplier)
			esPoolRemaining = math.Min(esPoolRemaining, MoMEBPool-tempDamage*esDamageTypeMultiplier)
			energyShield -= tempDamage * esDamageTypeMultiplier
			damageRemainder -= tempDamage
			if lost != nil {
				delete(lost[damageType], "energyShield")
				if tempDamage >= 1 {
					lost[damageType]["energyShield"] = tempDamage * esDamageTypeMultiplier
				}
			}
		}
		if MoMEffect > 0 && mana > 0 {
			MoMDamage := damageRemainder * MoMEffect
			tempDamage := math.Min(MoMDamage, MoMPool)
			MoMPoolRemaining = math.Min(MoMPoolRemaining, MoMPool-tempDamage)
			mana -= tempDamage
			damageRemainder -= tempDamage
			lose(damageType, "mana", tempDamage)
		} else {
			MoMPoolRemaining = 0
		}
		if output["preventedLifeLossTotal"] > 0 {
			halfLife := output["Life"] * 0.5
			lifeOverHalfLife := math.Max(life-halfLife, 0)
			preventPercent := output["preventedLifeLoss"] / 100
			poolAboveLow := lifeOverHalfLife / (1 - preventPercent)
			preventBelowHalfPercent := lifeLossBelowHalfPrevented / 100
			damageThatLifeCanStillTake := poolAboveLow + math.Max(math.Min(life, halfLife), 0)/(1-preventBelowHalfPercent)/(1-output["preventedLifeLoss"]/100)
			if damageThatLifeCanStillTake < damageRemainder {
				overkillDamage += damageRemainder - damageThatLifeCanStillTake
				damageRemainder = damageThatLifeCanStillTake
			}
			if output["preventedLifeLossBelowHalf"] != 0 {
				damageToSplit := math.Min(damageRemainder, poolAboveLow)
				lostLife := damageToSplit * (1 - preventPercent)
				preventedLoss := damageToSplit * preventPercent
				damageRemainder -= damageToSplit
				lifeLossLostOverTime += preventedLoss
				life -= lostLife
				lifeLossPrevented := preventedLoss
				if lost != nil {
					lost[damageType]["life"] = lostLife
				}
				if life <= halfLife {
					unspecificallyLowLifePreventedDamage := damageRemainder * preventPercent
					lifeLossLostOverTime += unspecificallyLowLifePreventedDamage
					damageRemainder -= unspecificallyLowLifePreventedDamage
					specificallyLowLifePreventedDamage := damageRemainder * preventBelowHalfPercent
					lifeBelowHalfLossLostOverTime += specificallyLowLifePreventedDamage
					damageRemainder -= specificallyLowLifePreventedDamage
					lifeLossPrevented += unspecificallyLowLifePreventedDamage + specificallyLowLifePreventedDamage
				}
				lose(damageType, "lifeLossPrevented", lifeLossPrevented)
			} else {
				tempDamage := damageRemainder * output["preventedLifeLoss"] / 100
				lifeLossLostOverTime += tempDamage
				damageRemainder -= tempDamage
				lose(damageType, "lifeLossPrevented", tempDamage)
			}
		}
		if life > 0 {
			tempDamage := math.Min(damageRemainder, life-1)
			life -= tempDamage
			damageRemainder -= tempDamage
			if lost != nil {
				lost[damageType]["life"] += math.Max(tempDamage, 0)
			}
			if ward > 0 && wardApplies {
				tempDamage := math.Min(damageRemainder, ward)
				ward -= tempDamage
				damageRemainder -= tempDamage
				lose(damageType, "runicWard", tempDamage)
			}
			if damageRemainder > 0 {
				tempDamage := math.Min(damageRemainder, life)
				life -= tempDamage
				damageRemainder -= tempDamage
				if lost != nil {
					lost[damageType]["life"] += math.Max(tempDamage, 0)
				}
			}
		}
		overkillDamage += damageRemainder
		lose(damageType, "overkill", damageRemainder)
	}
	hitPoolRemaining := calcLifeHitPoolWithLossPrevention(life, output["Life"], output["preventedLifeLoss"], lifeLossBelowHalfPrevented)
	if !math.IsInf(MoMPoolRemaining, 1) {
		hitPoolRemaining += MoMPoolRemaining
	}
	if !math.IsInf(esPoolRemaining, 1) {
		hitPoolRemaining += esPoolRemaining
	}
	if wardApplies {
		hitPoolRemaining += ward
	}

	return &ehpPools{
		alliesTakenBeforeYou:          allies,
		aegis:                         aegis,
		guard:                         guard,
		ward:                          ward,
		energyShield:                  energyShield,
		mana:                          mana,
		life:                          life,
		damageTakenThatCanBeRecouped:  damageTakenThatCanBeRecouped,
		lifeLossLostOverTime:          lifeLossLostOverTime,
		lifeBelowHalfLossLostOverTime: lifeBelowHalfLossLostOverTime,
		overkillDamage:                overkillDamage,
		hitPoolRemaining:              math.Floor(hitPoolRemaining),
		resourcesLostToTypeDamage:     lost,
	}
}

// ehpDamageIn is numberOfHitsToDie's DamageIn table. Its <type>EnergyShieldBypass keys are written
// but never read, and are not kept.
type ehpDamageIn struct {
	damage                map[string]float64
	cycles                float64
	iterations            float64
	trackRecoupable       bool
	trackLifeLossOverTime bool
	limitEHPSpeedup       bool
	cyclesRan             bool
	gainWhenHit           bool
	lifeWhenHit           float64
	manaWhenHit           float64
	energyShieldWhenHit   float64
	wardWhenHit           float64
}

func newEHPDamageIn() *ehpDamageIn {
	return &ehpDamageIn{damage: map[string]float64{}, cycles: 1}
}

//pob2:src/Modules/CalcDefence.lua:2054-2220
func numberOfHitsToDie(output map[string]float64, actor *ehpActor, DamageIn *ehpDamageIn) float64 {
	numHits := 0.0

	for _, damageType := range dmgTypeList {
		numHits += DamageIn.damage[damageType]
	}
	if numHits == 0 {
		return posInf
	}
	numHits = 0

	ward := output["Ward"]
	if DamageIn.cycles > 1 {
		ward = 0
	}
	aegis := map[string]float64{"shared": output["sharedAegis"], "sharedElemental": output["sharedElementalAegis"]}
	guard := map[string]float64{"shared": output["sharedGuardAbsorb"]}
	for _, damageType := range dmgTypeList {
		aegis[damageType] = output[damageType+"Aegis"]
		guard[damageType] = output[damageType+"GuardAbsorb"]
	}

	poolTable := &ehpPools{
		alliesTakenBeforeYou:          alliesTakenBeforeYou(output),
		aegis:                         aegis,
		guard:                         guard,
		ward:                          ward,
		energyShield:                  output["EnergyShieldRecoveryCap"],
		mana:                          output["ManaUnreserved"],
		life:                          output["LifeRecoverable"],
		lifeLossLostOverTime:          output["LifeLossLostOverTime"],
		lifeBelowHalfLossLostOverTime: output["LifeBelowHalfLossLostOverTime"],
		damageTakenThatCanBeRecouped:  map[string]float64{},
	}

	if DamageIn.cycles != 1 {
		DamageIn.trackRecoupable = false
		DamageIn.trackLifeLossOverTime = false
	}
	VaalArcticArmourHitsLeft := output["VaalArcticArmourLife"]
	if DamageIn.cycles > 1 {
		VaalArcticArmourHitsLeft = 0
	}

	iterationMultiplier := 1.0
	damageTotal := 0.0
	maxDamage := miscConstant("ehpCalcMaxDamage")
	maxIterations := miscConstant("ehpCalcMaxIterationsToCalc")
	for poolTable.life > 0 && DamageIn.iterations < maxIterations {
		DamageIn.iterations++
		Damage := map[string]float64{}
		damageTotal = 0
		VaalArcticArmourMultiplier := 1.0
		if VaalArcticArmourHitsLeft > 0 {
			VaalArcticArmourMultiplier = 1 - output["VaalArcticArmourMitigation"]*math.Min(VaalArcticArmourHitsLeft/iterationMultiplier, 1)
		}
		VaalArcticArmourHitsLeft -= iterationMultiplier
		for _, damageType := range dmgTypeList {
			damage := DamageIn.damage[damageType]
			if damage > 0 {
				Damage[damageType] = damage * iterationMultiplier * VaalArcticArmourMultiplier
			}
			damageTotal += damage
		}
		if DamageIn.gainWhenHit && (iterationMultiplier > 1 || DamageIn.cycles > 1) {
			gainMult := iterationMultiplier * DamageIn.cycles
			poolTable.life = math.Min(poolTable.life+DamageIn.lifeWhenHit*(gainMult-1), gainMult*output["LifeRecoverable"])
			poolTable.mana = math.Min(poolTable.mana+DamageIn.manaWhenHit*(gainMult-1), gainMult*output["ManaUnreserved"])
			poolTable.energyShield = math.Min(poolTable.energyShield+DamageIn.energyShieldWhenHit*(gainMult-1), gainMult*output["EnergyShieldRecoveryCap"])
			poolTable.ward = math.Min(poolTable.ward+DamageIn.wardWhenHit*(gainMult-1), gainMult*output["Ward"])
		}
		poolTable = reducePoolsByDamage(poolTable, Damage, actor)

		if poolTable.life > 0 && damageTotal >= maxDamage {
			return posInf
		}
		if DamageIn.gainWhenHit && poolTable.life > 0 {
			poolTable.life = math.Min(poolTable.life+DamageIn.lifeWhenHit, output["LifeRecoverable"])
			poolTable.mana = math.Min(poolTable.mana+DamageIn.manaWhenHit, output["ManaUnreserved"])
			poolTable.energyShield = math.Min(poolTable.energyShield+DamageIn.energyShieldWhenHit, output["EnergyShieldRecoveryCap"])
			poolTable.ward = math.Min(poolTable.ward+DamageIn.wardWhenHit, output["Ward"])
		}
		iterationMultiplier = 1
		speedUp := miscConstant("ehpCalcSpeedUp")
		if DamageIn.limitEHPSpeedup {
			speedUp = 4
		}
		if !DamageIn.cyclesRan && poolTable.life > 0 && DamageIn.iterations < maxIterations {
			inner := newEHPDamageIn()
			for _, damageType := range dmgTypeList {
				inner.damage[damageType] = DamageIn.damage[damageType] * speedUp
			}
			inner.limitEHPSpeedup = DamageIn.limitEHPSpeedup
			if DamageIn.gainWhenHit {
				inner.gainWhenHit = true
				inner.lifeWhenHit = DamageIn.lifeWhenHit
				inner.manaWhenHit = DamageIn.manaWhenHit
				inner.energyShieldWhenHit = DamageIn.energyShieldWhenHit
				inner.wardWhenHit = DamageIn.wardWhenHit
			}
			inner.cycles = DamageIn.cycles * speedUp
			inner.iterations = DamageIn.iterations
			iterationMultiplier = math.Max((numberOfHitsToDie(output, actor, inner)-1)*speedUp-1, 1)
			if math.IsInf(iterationMultiplier, 1) {
				return posInf
			}
			DamageIn.iterations = inner.iterations
			DamageIn.cyclesRan = true
		}
		numHits += iterationMultiplier
	}
	if DamageIn.trackRecoupable {
		for damageType, recoupable := range poolTable.damageTakenThatCanBeRecouped {
			output[damageType+"RecoupableDamageTaken"] += recoupable
		}
	}
	if DamageIn.trackLifeLossOverTime {
		output["LifeLossLostOverTime"] += poolTable.lifeLossLostOverTime
		output["LifeBelowHalfLossLostOverTime"] += poolTable.lifeBelowHalfLossLostOverTime
	}

	if poolTable.life == 0 && DamageIn.cycles == 1 {
		numHits -= poolTable.overkillDamage / damageTotal
	}
	damageTotal = 0
	for _, damageType := range dmgTypeList {
		damageTotal += DamageIn.damage[damageType] * numHits
	}
	if poolTable.life >= 0 && damageTotal >= maxDamage {
		return posInf
	}
	if math.IsNaN(numHits) {
		return 0
	}
	return math.Max(numHits, 0)
}

// buildDefenceEstimations writes no breakdowns. Its boolean outputs are Actor.OutputFlags, kept as
// locals too where the function reads them back.
//
//pob2:src/Modules/CalcDefence.lua:2222-4389
func buildDefenceEstimations(env *Environment, a *Actor) {
	actor := &ehpActor{Actor: a}
	breakdown := a.Breakdown
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB
	output := actor.Output
	placeholder := enemyConfigPlaceholders(env)

	damageCategoryConfig := "Average"
	if v, ok := env.ConfigInput["enemyDamageType"]; ok && v.IsString {
		damageCategoryConfig = v.String
	}

	elementalName := func(damageType, name string) []string {
		if isElemental[damageType] {
			return []string{name}
		}
		return nil
	}

	//pob2:block:src/Modules/CalcDefence.lua:2235-2255
	if damageCategoryConfig != "DamageOverTime" {
		worstOf := 1.0
		if v, ok := configInputNumber(env, "EHPUnluckyWorstOf"); ok {
			worstOf = v
		}
		specificTypeAvoidance := actor.SpecificTypeAvoidance
		avoidProjectiles := output["AvoidProjectilesChance"]
		if specificTypeAvoidance {
			avoidProjectiles = 0
		}
		output["MeleeNotHitChance"] = 100 - (1-output["MeleeEvadeChance"]/100)*(1-output["EffectiveAttackDodgeChance"]/100)*(1-output["AvoidAllDamageFromHitsChance"]/100)*100
		output["ProjectileNotHitChance"] = 100 - (1-output["ProjectileEvadeChance"]/100)*(1-output["EffectiveAttackDodgeChance"]/100)*(1-output["AvoidAllDamageFromHitsChance"]/100)*(1-avoidProjectiles/100)*100
		output["SpellNotHitChance"] = 100 - (1-output["SpellEvadeChance"]/100)*(1-output["EffectiveSpellDodgeChance"]/100)*(1-output["AvoidAllDamageFromHitsChance"]/100)*100
		output["SpellProjectileNotHitChance"] = 100 - (1-output["SpellProjectileEvadeChance"]/100)*(1-output["EffectiveSpellDodgeChance"]/100)*(1-output["AvoidAllDamageFromHitsChance"]/100)*(1-avoidProjectiles/100)*100
		output["UntypedNotHitChance"] = 100 - (1-output["AvoidAllDamageFromHitsChance"]/100)*100
		output["AverageNotHitChance"] = (output["MeleeNotHitChance"] + output["ProjectileNotHitChance"] + output["SpellNotHitChance"] + output["SpellProjectileNotHitChance"]) / 4
		output["AverageEvadeChance"] = (output["MeleeEvadeChance"] + output["ProjectileEvadeChance"] + output["SpellNotHitChance"] + output["SpellProjectileNotHitChance"]) / 4
		output["ConfiguredNotHitChance"] = output[damageCategoryConfig+"NotHitChance"]
		output["ConfiguredEvadeChance"] = output[damageCategoryConfig+"EvadeChance"]
		if worstOf > 1 {
			output["ConfiguredNotHitChance"] = output["ConfiguredNotHitChance"] / 100 * output["ConfiguredNotHitChance"]
			output["ConfiguredEvadeChance"] = output["ConfiguredEvadeChance"] / 100 * output["ConfiguredEvadeChance"]
			if worstOf == 4 {
				output["ConfiguredNotHitChance"] = output["ConfiguredNotHitChance"] / 100 * output["ConfiguredNotHitChance"]
				output["ConfiguredEvadeChance"] = output["ConfiguredEvadeChance"] / 100 * output["ConfiguredEvadeChance"]
			}
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:2258-2386
	output["totalEnemyDamage"] = 0
	output["totalEnemyDamageIn"] = 0
	if damageCategoryConfig == "DamageOverTime" {
		for _, damageType := range dmgTypeList {
			output[damageType+"EnemyPen"] = 0
			output[damageType+"EnemyDamageMult"] = calcLibMod(enemyDB, nil, append([]string{"Damage", damageType + "Damage"}, elementalName(damageType, "ElementalDamage")...)...)
			output[damageType+"EnemyOverwhelm"] = 0
			output[damageType+"EnemyDamage"] = 0
		}
	} else {
		//pob2:block:src/Modules/CalcDefence.lua:2268-2282
		if breakdown != nil {
			breakdown.setEntry("totalEnemyDamage", &BreakdownEntry{
				Label:   "Total damage from the enemy",
				RowList: []map[string]string{},
				ColList: []BreakdownCol{{"Type", "type"}, {"Value", "value"}, {"Mult", "mult"}, {"Crit", "crit"}, {"Conversion", "convMult"}, {"Final", "final"}, {"From", "from"}},
			})
		}
		enemyCritChance := 0.0
		switch {
		case enemyDB.Flag(nil, "NeverCrit"):
		case enemyDB.Flag(nil, "AlwaysCrit"):
			enemyCritChance = 100
		default:
			base := configNumberOr(env, placeholder, "enemyCritChance", 0)
			if o := modDB.Override(nil, "enemyCritChance"); o != nil {
				base = o.Float()
			}
			enemyCritChance = math.Max(math.Min(base*(1+modDB.Sum(mod.TypeIncrease, nil, "EnemyCritChance")/100+enemyDB.Sum(mod.TypeIncrease, nil, "CritChance")/100)*(1-output["ConfiguredEvadeChance"]/100), 100), 0)
		}
		if modDB.Flag(nil, "EnemyUnluckyCrit") {
			enemyCritChance = enemyCritChance / 100 * enemyCritChance
		}
		output["EnemyCritChance"] = enemyCritChance
		enemyCritDamage := math.Max((configNumberOr(env, placeholder, "enemyCritDamage", 0)+enemyDB.Sum(mod.TypeBase, nil, "CritMultiplier"))*(1+enemyDB.Sum(mod.TypeIncrease, &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagHit)}, "CritMultiplier")/100), 0)
		output["EnemyCritEffect"] = 1 + enemyCritChance/100*(enemyCritDamage/100)*(1-output["CritExtraDamageReduction"]/100)
		enemyCfg := &moddb.ListCfg{KeywordFlags: utils.Ptr(^mod.KeywordFlagMatchAll)}
		enemyDamageConversion := map[string]map[string]float64{}
		for _, damageType := range dmgTypeList {
			enemyDamageMult := calcLibMod(enemyDB, nil, append([]string{"Damage", damageType + "Damage"}, elementalName(damageType, "ElementalDamage")...)...)
			enemyDamage, hasDamage := configInputNumber(env, "enemy"+damageType+"Damage")
			enemyPen, hasPen := configInputNumber(env, "enemy"+damageType+"Pen")
			enemyOverwhelm, hasOverwhelm := configInputNumber(env, "enemy"+damageType+"Overwhelm")
			sourceStr := "Config"
			if !hasDamage {
				sourceStr = "Default"
				enemyDamage = placeholder["enemy"+damageType+"Damage"]
			}
			if !hasPen {
				enemyPen = placeholder["enemy"+damageType+"Pen"]
			}
			if !hasOverwhelm {
				enemyOverwhelm = placeholder["enemy"+damageType+"enemyOverwhelm"]
			}

			enemyDamage += (enemyDB.Sum(mod.TypeBase, enemyCfg, damageType+"Min") + enemyDB.Sum(mod.TypeBase, enemyCfg, damageType+"Max")) / 2

			conversionTotal := 0.0
			if damageType == "Physical" {
				conversions := map[string]float64{"total": 0, "totalSkill": 0}
				for _, damageTypeTo := range dmgTypeList {
					conversions[damageTypeTo+"skill"] = enemyDB.Sum(mod.TypeBase, enemyCfg, damageType+"DamageSkillConvertTo"+damageTypeTo)
					conversions[damageTypeTo] = enemyDB.Sum(mod.TypeBase, enemyCfg, damageType+"DamageConvertTo"+damageTypeTo)
					conversions["totalSkill"] += conversions[damageTypeTo+"skill"]
					conversions["total"] += conversions[damageTypeTo]
				}
				if conversions["totalSkill"] > 100 {
					mult := 100 / conversions["totalSkill"]
					conversions["totalSkill"] *= mult
					conversions["total"] = 0
					for _, damageTypeTo := range dmgTypeList {
						conversions[damageTypeTo+"skill"] *= mult
						conversions[damageTypeTo] = 0
					}
				} else if conversions["total"]+conversions["totalSkill"] > 100 {
					mult := (100 - conversions["totalSkill"]) / conversions["total"]
					conversions["total"] *= mult
					for _, damageTypeTo := range dmgTypeList {
						conversions[damageTypeTo] *= mult
					}
				}
				conversionTotal = conversions["total"] + conversions["totalSkill"]
				for _, damageTypeTo := range dmgTypeList {
					gainAsPercent := (enemyDB.Sum(mod.TypeBase, enemyCfg, damageType+"DamageGainAs"+damageTypeTo, "DamageGainAs"+damageTypeTo) + conversions[damageTypeTo+"skill"] + conversions[damageTypeTo]) / 100
					if gainAsPercent > 0 {
						if enemyDamageConversion[damageTypeTo] == nil {
							enemyDamageConversion[damageTypeTo] = map[string]float64{}
						}
						enemyDamageConversion[damageTypeTo][damageType] = enemyDamage * gainAsPercent
					}
				}
			}

			enemyOverwhelm += enemyDB.Sum(mod.TypeBase, nil, "PhysicalOverwhelm") + modDB.Sum(mod.TypeBase, nil, "EnemyPhysicalOverwhelm")

			output[damageType+"EnemyPen"] = enemyPen
			output[damageType+"EnemyDamageMult"] = enemyDamageMult
			output[damageType+"EnemyOverwhelm"] = enemyOverwhelm
			output["totalEnemyDamageIn"] += enemyDamage
			output[damageType+"EnemyDamage"] = enemyDamage * (1 - conversionTotal/100) * enemyDamageMult * output["EnemyCritEffect"]
			conversionExtra := -float64(enemyDamage*enemyDamageMult*output["EnemyCritEffect"]) + output[damageType+"EnemyDamage"]
			for _, damageTypeFrom := range dmgTypeList {
				enemyDamage, ok := enemyDamageConversion[damageType][damageTypeFrom]
				if !ok {
					continue
				}
				names := make([]string, 0, 5)
				names = append(names, "Damage", damageType+"Damage", damageTypeFrom+"Damage")
				names = append(names, elementalName(damageType, "ElementalDamage")...)
				names = append(names, elementalName(damageTypeFrom, "ElementalDamage")...)
				enemyDamageMult := calcLibMod(enemyDB, nil, names...)
				output[damageType+"EnemyDamage"] += enemyDamage * enemyDamageMult * output["EnemyCritEffect"]
				conversionExtra += float64(enemyDamage * enemyDamageMult * output["EnemyCritEffect"])
			}
			output["totalEnemyDamage"] += output[damageType+"EnemyDamage"]
			//pob2:block:src/Modules/CalcDefence.lua:2365-2384
			if breakdown != nil {
				lines := []string{
					sFormat("from %s: %d", sourceStr, enemyDamage),
					sFormat("* %.2f (modifiers to enemy damage)", enemyDamageMult),
					sFormat("* %.3f (enemy crit effect)", output["EnemyCritEffect"]),
				}
				sign, abs := "-", -conversionExtra
				if conversionExtra > 0 {
					sign = "+"
				}
				if conversionExtra >= 0 {
					abs = conversionExtra
				}
				if conversionExtra != 0 {
					lines = append(lines, sFormat("%s %d (enemy damage conversion)", sign, abs))
				}
				lines = append(lines, sFormat("= %d", output[damageType+"EnemyDamage"]))
				breakdown.set(damageType+"EnemyDamage", lines...)
				convSign := ""
				if conversionExtra > 0 {
					convSign = "+"
				} else if conversionExtra < 0 {
					convSign = "-"
				}
				e := breakdown.Entries["totalEnemyDamage"]
				e.RowList = append(e.RowList, map[string]string{
					"type":     damageType,
					"value":    sFormat("%d", enemyDamage),
					"mult":     sFormat("%.2f", enemyDamageMult),
					"crit":     sFormat("%.2f", output["EnemyCritEffect"]),
					"convMult": sFormat("%s%d", convSign, abs),
					"final":    sFormat("%d", output[damageType+"EnemyDamage"]),
					"from":     sourceStr,
				})
			}
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:2389-2464
	actor.damageShiftTable = map[string]map[string]float64{}
	actor.damageOverTimeShiftTable = map[string]map[string]float64{}
	for _, damageType := range dmgTypeList {
		shiftTable := map[string]float64{}
		dotShiftTable := map[string]float64{}
		destTotal := 0.0
		dotDestinationTotal := 0.0
		for _, destType := range dmgTypeList {
			if destType != damageType {
				dotShiftTable[destType] = modDB.Sum(mod.TypeBase, nil, append([]string{damageType + "DamageTakenAs" + destType}, elementalName(damageType, "ElementalDamageTakenAs"+destType)...)...)
				dotDestinationTotal += dotShiftTable[destType]
				shiftTable[destType] = dotShiftTable[destType] + modDB.Sum(mod.TypeBase, nil, append([]string{damageType + "DamageFromHitsTakenAs" + destType}, elementalName(damageType, "ElementalDamageFromHitsTakenAs"+destType)...)...)
				destTotal += shiftTable[destType]
			}
		}
		dotShiftTable[damageType] = math.Max(100-dotDestinationTotal, 0)
		actor.damageOverTimeShiftTable[damageType] = dotShiftTable
		shiftTable[damageType] = math.Max(100-destTotal, 0)
		actor.damageShiftTable[damageType] = shiftTable

		output[damageType+"TakenDamage"] = output[damageType+"EnemyDamage"] * actor.damageShiftTable[damageType][damageType] / 100
		//pob2:block:src/Modules/CalcDefence.lua:2413-2426
		if breakdown != nil {
			breakdown.setEntry(damageType+"TakenDamage", &BreakdownEntry{
				Label:   "Taken",
				RowList: []map[string]string{{"type": damageType, "value": sFormat("%d", output[damageType+"TakenDamage"])}},
				ColList: []BreakdownCol{{"Type", "type"}, {"Value", "value"}},
			})
		}
	}
	for _, damageType := range dmgTypeList {
		for _, damageConvertedType := range dmgTypeList {
			if damageType != damageConvertedType {
				damage := output[damageType+"EnemyDamage"] * actor.damageShiftTable[damageType][damageConvertedType] / 100
				output[damageConvertedType+"TakenDamage"] += damage
				if breakdown != nil && damage > 0 {
					e := breakdown.Entries[damageConvertedType+"TakenDamage"]
					e.RowList = append(e.RowList, map[string]string{"type": damageType, "value": sFormat("%d", damage)})
				}
			}
		}
	}
	output["totalTakenDamage"] = 0
	//pob2:block:src/Modules/CalcDefence.lua:2445-2463
	var totalTaken *BreakdownEntry
	if breakdown != nil {
		totalTaken = &BreakdownEntry{Label: "Total damage taken from the enemy after taken as", RowList: []map[string]string{}, ColList: []BreakdownCol{{"Type", "type"}, {"Value", "value"}}}
		breakdown.setEntry("totalTakenDamage", totalTaken)
	}
	for _, damageType := range dmgTypeList {
		output["totalTakenDamage"] += output[damageType+"TakenDamage"]
		if totalTaken != nil {
			totalTaken.RowList = append(totalTaken.RowList, map[string]string{"type": damageType, "value": sFormat("%d", output[damageType+"TakenDamage"])})
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:2467-2526
	anyTakenReflect := false
	a.setOutputFlag("AnyTakenReflect", anyTakenReflect)
	for _, damageType := range dmgTypeList {
		baseTakenInc := modDB.Sum(mod.TypeIncrease, nil, "DamageTaken", damageType+"DamageTaken")
		baseTakenMore := modDB.More(nil, "DamageTaken", damageType+"DamageTaken")
		if isElemental[damageType] {
			baseTakenInc += modDB.Sum(mod.TypeIncrease, nil, "ElementalDamageTaken")
			baseTakenMore *= modDB.More(nil, "ElementalDamageTaken")
		}
		{
			takenInc := baseTakenInc + modDB.Sum(mod.TypeIncrease, nil, "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
			takenMore := baseTakenMore * modDB.More(nil, "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
			if isElemental[damageType] {
				takenInc += modDB.Sum(mod.TypeIncrease, nil, "ElementalDamageTakenWhenHit")
				takenMore *= modDB.More(nil, "ElementalDamageTakenWhenHit")
			}
			output[damageType+"TakenHitMult"] = math.Max((1+takenInc/100)*takenMore, 0)

			for _, hitType := range hitSourceList {
				baseTakenIncType := takenInc + modDB.Sum(mod.TypeIncrease, nil, hitType+"DamageTaken")
				baseTakenMoreType := takenMore * modDB.More(nil, hitType+"DamageTaken")
				output[hitType+"TakenHitMult"] = math.Max((1+baseTakenIncType/100)*baseTakenMoreType, 0)
				output[damageType+hitType+"TakenHitMult"] = output[hitType+"TakenHitMult"]
			}
			{
				takenInc += modDB.Sum(mod.TypeIncrease, nil, "ReflectedDamageTaken", damageType+"ReflectedDamageTaken")
				takenMore *= modDB.More(nil, "ReflectedDamageTaken", damageType+"ReflectedDamageTaken")
				if isElemental[damageType] {
					takenInc += modDB.Sum(mod.TypeIncrease, nil, "ElementalReflectedDamageTaken")
					takenMore *= modDB.More(nil, "ElementalReflectedDamageTaken")
				}
				output[damageType+"TakenReflect"] = math.Max((1+takenInc/100)*takenMore, 0)
			}
		}
		{
			takenInc := baseTakenInc + modDB.Sum(mod.TypeIncrease, nil, "DamageTakenOverTime", damageType+"DamageTakenOverTime")
			takenMore := baseTakenMore * modDB.More(nil, "DamageTakenOverTime", damageType+"DamageTakenOverTime")
			if isElemental[damageType] {
				takenInc += modDB.Sum(mod.TypeIncrease, nil, "ElementalDamageTakenOverTime")
				takenMore *= modDB.More(nil, "ElementalDamageTakenOverTime")
			}
			resist := 0.0
			if !modDB.Flag(nil, "SelfIgnore"+damageType+"Resistance") {
				resist = outputOr(output, damageType+"ResistOverTime", damageType+"Resist")
			}
			reduction := 0.0
			if !modDB.Flag(nil, "SelfIgnoreBase"+damageType+"DamageReduction") {
				reduction = output["Base"+damageType+"DamageReduction"]
			}
			output[damageType+"TakenDotMult"] = math.Max((1-resist/100)*(1-reduction/100)*(1+takenInc/100)*takenMore, 0)
			//pob2:block:src/Modules/CalcDefence.lua:2514-2524
			if breakdown != nil {
				reductionLink := link("%.2f ^8(%s damage reduction)", 1-reduction/100)
				reductionLink.Args = []any{strings.ToLower(damageType)}
				lines, _ := breakdown.multiChain(nil, multiChainSpec{
					Label: "DoT Multiplier:",
					Links: []chainLink{link("%.2f ^8(resistance)", 1-resist/100), reductionLink, link("%.2f ^8(increased/reduced damage taken)", 1+takenInc/100), link("%.2f ^8(more/less damage taken)", takenMore)},
					Total: sFormat("= %.2f", output[damageType+"TakenDotMult"]),
				})
				breakdown.set(damageType+"TakenDotMult", lines...)
			}
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:2529-2742
	output["totalTakenHit"] = 0
	if breakdown != nil {
		breakdown.setEntry("totalTakenHit", &BreakdownEntry{
			Label:   "Total damage taken after mitigation",
			RowList: []map[string]string{},
			ColList: []BreakdownCol{{"Type", "type"}, {"Incoming", "incoming"}, {"Mult", "mult"}, {"Value", "value"}},
		})
	}
	impaleFlags := mod.MFlag(0)
	if damageCategoryConfig == "Melee" || damageCategoryConfig == "Projectile" || damageCategoryConfig == "Average" {
		impaleFlags = mod.MFlagAttack
	}
	enemyImpaleChance := enemyDB.Sum(mod.TypeBase, &moddb.ListCfg{Flags: utils.Ptr(impaleFlags), KeywordFlags: utils.Ptr(mod.KeywordFlag(0))}, "ImpaleChance")
	if damageCategoryConfig == "Average" {
		enemyImpaleChance *= 0.5
	}
	enemyImpaleChance *= 1 - output["ImpaleAvoidChance"]
	for _, damageType := range dmgTypeList {
		resist := 0.0
		if !modDB.Flag(nil, "SelfIgnore"+damageType+"Resistance") {
			resist = outputOr(output, damageType+"ResistWhenHit", damageType+"Resist")
		}
		reduction := 0.0
		if !modDB.Flag(nil, "SelfIgnoreBase"+damageType+"DamageReduction") {
			reduction = outputOr(output, "Base"+damageType+"DamageReductionWhenHit", "Base"+damageType+"DamageReduction")
		}
		enemyPen := 0.0
		if !modDB.Flag(nil, "SelfIgnore"+damageType+"Resistance", "EnemyCannotPen"+damageType+"Resistance") {
			enemyPen = output[damageType+"EnemyPen"]
		}
		enemyOverwhelm := 0.0
		if !modDB.Flag(nil, "SelfIgnore"+damageType+"DamageReduction") {
			enemyOverwhelm = output[damageType+"EnemyOverwhelm"]
		}
		damage := output[damageType+"TakenDamage"]
		impaleDamage := 0.0
		if enemyImpaleChance > 0 && damageType == "Physical" {
			impaleDamage = damage * miscConstant("ImpaleStoredDamageBase")
		}

		armourReduct := 0.0
		impaleArmourReduct := 0.0
		percentOfArmourApplies := 0.0
		if !modDB.Flag(nil, "ArmourDoesNotApplyTo"+damageType+"DamageTaken") {
			percentOfArmourApplies = modDB.Sum(mod.TypeBase, nil, "ArmourAppliesTo"+damageType+"DamageTaken")
		}
		if isElemental[damageType] && !modDB.Flag(nil, "ArmourDoesNotApplyToElementalDamageTaken") {
			percentOfArmourApplies += modDB.Sum(mod.TypeBase, nil, "ArmourAppliesToElementalDamageTaken")
		}
		effectiveAppliedArmour := (output["Armour"] * percentOfArmourApplies / 100) * (1 + output["ArmourDefense"])
		effectiveArmourFromArmour := effectiveAppliedArmour
		// effectiveArmourFromOther, in an order of its own: PoB-PoE2 walks it with pairs
		type armourFromOther struct {
			label          string
			percent, value float64
		}
		var fromOther []armourFromOther
		percentOfEvasionApplies := 0.0
		if !modDB.Flag(nil, "EvasionDoesNotApplyTo"+damageType+"DamageTaken") {
			percentOfEvasionApplies = modDB.Sum(mod.TypeBase, nil, "EvasionAppliesTo"+damageType+"DamageTaken")
		}
		percentOfEnergyShieldApplies := 0.0
		if !modDB.Flag(nil, "EnergyShieldDoesNotApplyTo"+damageType+"DamageTaken") {
			percentOfEnergyShieldApplies = modDB.Sum(mod.TypeBase, nil, "EnergyShieldAppliesTo"+damageType+"DamageTaken")
		}
		if percentOfEvasionApplies > 0 {
			fromOther = append(fromOther, armourFromOther{"Evasion", percentOfEvasionApplies, math.Max(output["Evasion"]*percentOfEvasionApplies/100, 0)})
		}
		if percentOfEnergyShieldApplies > 0 {
			fromOther = append(fromOther, armourFromOther{"Energy Shield", percentOfEnergyShieldApplies, math.Max(output["EnergyShield"]*percentOfEnergyShieldApplies/100, 0)})
		}
		for _, o := range fromOther {
			effectiveAppliedArmour += o.value
		}

		resMult := 1 - resist/100
		if resist > 0 {
			resMult = 1 - math.Max(resist-enemyPen, 0)/100
		}
		takenFlat := modDB.Sum(mod.TypeBase, nil, "DamageTaken", damageType+"DamageTaken", "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
		switch damageCategoryConfig {
		case "Melee", "Projectile":
			takenFlat += modDB.Sum(mod.TypeBase, nil, "DamageTakenFromAttacks", damageType+"DamageTakenFromAttacks", damageType+"DamageTakenFrom"+damageCategoryConfig+"Attacks")
		case "Spell", "SpellProjectile":
			takenFlat += modDB.Sum(mod.TypeBase, nil, "DamageTakenFromSpells", damageType+"DamageTakenFromSpells", damageType+"DamageTakenFromSpellProjectiles")
		case "Average":
			takenFlat += modDB.Sum(mod.TypeBase, nil, "DamageTakenFromAttacks", damageType+"DamageTakenFromAttacks")/2 + modDB.Sum(mod.TypeBase, nil, damageType+"DamageTakenFromProjectileAttacks")/4 + modDB.Sum(mod.TypeBase, nil, "DamageTakenFromSpells", damageType+"DamageTakenFromSpells")/2 + modDB.Sum(mod.TypeBase, nil, "DamageTakenFromSpellProjectiles", damageType+"DamageTakenFromSpellProjectiles")/4
		}
		output[damageType+"takenFlat"] = takenFlat
		if effectiveAppliedArmour > 0 {
			armourReduct = armourReduction(effectiveAppliedArmour, damage)
			armourReduct = math.Min(output[damageType+"DamageReductionMax"], armourReduct)
			if impaleDamage > 0 {
				impaleArmourReduct = math.Min(output[damageType+"DamageReductionMax"], armourReduction(effectiveAppliedArmour, impaleDamage))
			}
		}
		totalReduct := math.Min(output[damageType+"DamageReductionMax"], armourReduct+reduction)
		reductMult := 1 - math.Max(math.Min(output[damageType+"DamageReductionMax"], totalReduct-enemyOverwhelm), 0)/100
		output[damageType+"DamageReduction"] = 100 - reductMult*100
		if impaleDamage > 0 {
			impaleDamage = impaleDamage * resMult * (1 - math.Max(math.Min(output[damageType+"DamageReductionMax"], math.Min(output[damageType+"DamageReductionMax"], impaleArmourReduct+reduction)-enemyOverwhelm), 0)/100)
			impaleDamage = impaleDamage * enemyImpaleChance / 100 * 5 * output[damageType+"TakenReflect"]
		}
		physicalArmour := 0.0
		if damageType == "Physical" {
			physicalArmour = 100
		}
		//pob2:block:src/Modules/CalcDefence.lua:2607-2640
		if breakdown != nil {
			var lines []string
			if armourReduct != 0 {
				if percentOfArmourApplies != physicalArmour && percentOfArmourApplies > 0 {
					lines = append(lines, sFormat("%d%% percent of Armour applies", percentOfArmourApplies), sFormat("Armour contributing to reduction: %d", effectiveArmourFromArmour))
				}
				if effectiveArmourFromArmour == effectiveAppliedArmour {
					lines = append(lines, sFormat("Reduction from Armour: %d%%", armourReduct))
				} else {
					for _, o := range fromOther {
						lines = append(lines, sFormat("%d%% percent of %s applies", o.percent, o.label), sFormat("%s contributing to reduction: %d", o.label, o.value))
					}
					lines = append(lines, sFormat("Combined Defence used for reduction: %d", effectiveAppliedArmour), sFormat("Reduction from combined Defence: %d%%", armourReduct))
				}
				if resMult != 1 {
					lines = append(lines, sFormat("Enemy Hit Damage After Resistance: %d ^8(total incoming damage)", damage*resMult))
				} else {
					lines = append(lines, sFormat("Enemy Hit Damage: %d ^8(total incoming damage)", damage))
				}
			}
			if reduction != 0 {
				lines = append(lines, sFormat("Base %s Damage Reduction: %d%%", damageType, reduction))
			}
			if enemyOverwhelm != 0 {
				lines = append(lines, sFormat("Enemy Overwhelm %s Damage: %d%%", damageType, enemyOverwhelm))
			}
			if countTrue(armourReduct != 0, reduction != 0, enemyOverwhelm != 0) >= 2 {
				lines = append(lines, sFormat("Total %s Damage Reduction: %d%%", damageType, 100-reductMult*100))
			}
			breakdown.set(damageType+"DamageReduction", lines...)
		}
		takenMult := output[damageType+"TakenHitMult"]
		spellSuppressMult := 1.0
		switch damageCategoryConfig {
		case "Melee", "Projectile":
			takenMult = output[damageType+"AttackTakenHitMult"]
		case "Spell", "SpellProjectile":
			takenMult = output[damageType+"SpellTakenHitMult"]
			if output["EffectiveSpellSuppressionChance"] == 100 {
				spellSuppressMult = 1 - output["SpellSuppressionEffect"]/100
			}
		case "Average":
			takenMult = (output[damageType+"SpellTakenHitMult"] + output[damageType+"AttackTakenHitMult"]) / 2
			if output["EffectiveSpellSuppressionChance"] == 100 {
				spellSuppressMult = 1 - output["SpellSuppressionEffect"]/100/2
			}
		}
		deflectMulti := 1.0
		if output["DeflectChance"] == 100 {
			deflectMulti = 1 - output["DeflectEffect"]/100
		}
		output[damageType+"EffectiveAppliedArmour"] = effectiveAppliedArmour
		output[damageType+"ResistTakenHitMulti"] = resMult
		afterReductionMulti := takenMult * spellSuppressMult * deflectMulti
		output[damageType+"AfterReductionTakenHitMulti"] = afterReductionMulti
		baseMult := resMult * reductMult
		output[damageType+"BaseTakenHitMult"] = baseMult * afterReductionMulti
		takenMultReflect := output[damageType+"TakenReflect"]
		finalReflect := baseMult * takenMultReflect
		output[damageType+"TakenHit"] = math.Max(damage*baseMult+takenFlat, 0)*afterReductionMulti + impaleDamage
		if damage > 0 {
			output[damageType+"TakenHitMult"] = output[damageType+"TakenHit"] / damage
		} else {
			output[damageType+"TakenHitMult"] = 0
		}
		output["totalTakenHit"] += output[damageType+"TakenHit"]
		if anyTakenReflect {
			output[damageType+"TakenReflectMult"] = finalReflect
		}
		//pob2:block:src/Modules/CalcDefence.lua:2667-2741
		if breakdown != nil {
			var lines []string
			if reduction != 0 {
				lines = append(lines, sFormat("Base %s Damage Taken: %.2f", damageType, 1-reduction/100))
			}
			if armourReduct != 0 {
				if percentOfArmourApplies != physicalArmour && percentOfArmourApplies > 0 {
					lines = append(lines, sFormat("%d%% percent of Armour applies", percentOfArmourApplies))
				}
				if effectiveArmourFromArmour == effectiveAppliedArmour {
					lines = append(lines, sFormat("Reduction from Armour: %.2f", 1-armourReduct/100))
				} else {
					if effectiveArmourFromArmour > 0 {
						lines = append(lines, sFormat("Armour contributing to reduction: %d", effectiveArmourFromArmour))
					}
					for _, o := range fromOther {
						lines = append(lines, sFormat("%d%% percent of %s applies", o.percent, o.label), sFormat("%s contributing to reduction: %d", o.label, o.value))
					}
					lines = append(lines, sFormat("Combined Defence used for reduction: %d", effectiveAppliedArmour), sFormat("Reduction from combined Defence: %.2f", 1-armourReduct/100))
				}
			}
			if enemyOverwhelm != 0 {
				lines = append(lines, sFormat("+ Enemy Overwhelm %s Damage: %.2f", damageType, enemyOverwhelm/100))
			}
			if reduction != 0 || armourReduct != 0 || enemyOverwhelm != 0 {
				lines = append(lines, sFormat("= %.2f", reductMult))
			}
			if resist != 0 {
				lines = append(lines, sFormat("Resistance: %.2f", 1-resist/100))
			}
			if enemyPen != 0 {
				lines = append(lines, sFormat("+ Enemy Pen: %.2f", enemyPen/100))
			}
			switch {
			case resist <= 0 && enemyPen != 0:
				lines = append(lines, sFormat("= %.2f ^8(Negative resistance unaffected by penetration)", resMult))
			case resist-enemyPen < 0:
				lines = append(lines, sFormat("= %.2f ^8(Penetration cannot bring resistances below 0)", resMult))
			case resist != 0:
				lines = append(lines, sFormat("= %.2f", resMult))
			}
			if resMult != 1 && reductMult != 1 {
				lines = append(lines, sFormat("%.2f x %.2f = %.3f", resMult, reductMult, baseMult))
			}
			if takenFlat != 0 {
				lines = append(lines, sFormat("+ Flat: %d", takenFlat))
			}
			if takenMult != 1 {
				lines = append(lines, sFormat("x Taken: %.3f", takenMult))
			}
			if spellSuppressMult != 1 {
				lines = append(lines, sFormat("x Spell Suppression: %.3f", spellSuppressMult))
			}
			if deflectMulti != 1 {
				lines = append(lines, sFormat("x Deflection: %.3f", deflectMulti))
			}
			if impaleDamage != 0 {
				lines = append(lines, sFormat("+ Impale: %.1f", impaleDamage))
			}
			if takenFlat != 0 || afterReductionMulti != 1 || impaleDamage != 0 {
				lines = append(lines, sFormat("= %.3f", output[damageType+"TakenHitMult"]))
			}
			breakdown.set(damageType+"TakenHitMult", lines...)
			if anyTakenReflect {
				reflect := []string{sFormat("Resistance: %.3f", 1-resist/100)}
				if enemyPen > 0 {
					reflect = append(reflect, sFormat("Enemy Pen: %.2f", enemyPen))
				}
				reflect = append(reflect, sFormat("Taken: %.3f", takenMultReflect), sFormat("= %.3f", finalReflect))
				breakdown.set(damageType+"TakenReflectMult", reflect...)
			}
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:2745-2860
	{
		stunThresholdBase := 0.0
		stunThresholdSource := ""
		switch {
		case modDB.Flag(nil, "StunThresholdBasedOnEnergyShieldInsteadOfLife"):
			stunThresholdMult := modDB.Sum(mod.TypeBase, nil, "StunThresholdEnergyShieldPercent")
			stunThresholdBase = output["EnergyShield"] * stunThresholdMult / 100
			stunThresholdSource = luaToString(stunThresholdMult) + "% of Energy Shield"
		case modDB.Flag(nil, "StunThresholdBasedOnManaInsteadOfLife"):
			stunThresholdMult := modDB.Sum(mod.TypeBase, nil, "StunThresholdManaPercent")
			stunThresholdBase = output["Mana"] * stunThresholdMult / 100
			stunThresholdSource = luaToString(stunThresholdMult) + "% of Mana"
		case modDB.Flag(nil, "ChaosInoculation"):
			stunThresholdBase = modDB.Sum(mod.TypeBase, nil, "Life")
			stunThresholdSource = "Life before Chaos Inoculation"
		default:
			stunThresholdBase = output["Life"]
			stunThresholdSource = "Life"
		}
		if modDB.Flag(nil, "AddESToStunThreshold") {
			esMult := modDB.Sum(mod.TypeBase, nil, "ESToStunThresholdPercent")
			stunThresholdBase += output["EnergyShield"] * esMult / 100
			stunThresholdSource += " and " + luaToString(esMult) + "% of Energy Shield"
		}
		StunThresholdInc := 1 + modDB.Sum(mod.TypeIncrease, nil, "StunThreshold")/100
		// PoB passes "INC" as the cfg here; a string cfg carries none of the fields a tag reads.
		StunThresholdMore := modDB.More(nil, "StunThreshold")
		StunThresholdModBase := modDB.Sum(mod.TypeBase, nil, "StunThreshold")
		output["StunThreshold"] = (stunThresholdBase + StunThresholdModBase) * StunThresholdInc * StunThresholdMore

		notAvoidChance := 0.0
		if !modDB.Flag(nil, "StunImmune") {
			notAvoidChance = 100 - math.Min(modDB.Sum(mod.TypeBase, nil, "AvoidStun"), 100)
		}
		if output["EnergyShield"] > output["totalTakenHit"] && !env.ModDB.Flag(nil, "EnergyShieldProtectsMana") {
			notAvoidChance *= 0.5
		}
		output["StunAvoidChance"] = 100 - notAvoidChance
		//pob2:block:src/Modules/CalcDefence.lua:2779-2800
		if breakdown != nil {
			lines := []string{sFormat("%d ^8(base from %s)", stunThresholdBase, stunThresholdSource)}
			for _, source := range []string{"Tree", "Item", "Skill"} {
				src := mod.Source(source)
				if sourceVal := modDB.Sum(mod.TypeBase, &moddb.ListCfg{Source: &src}, "StunThreshold"); sourceVal > 0 {
					lines = append(lines, sFormat("+ %d ^8(base from %s)", sourceVal, source))
				}
			}
			if StunThresholdInc != 1 {
				lines = append(lines, sFormat("* %.2f ^8(increased threshold)", StunThresholdInc))
			}
			if StunThresholdMore != 1 {
				lines = append(lines, sFormat("* %.2f ^8(more threshold)", StunThresholdMore))
			}
			if StunThresholdInc != 1 || StunThresholdMore != 1 {
				lines = append(lines, sFormat("= %d", output["StunThreshold"]))
			}
			breakdown.set("StunThreshold", lines...)
			breakdown.set("StunAvoidChance",
				colorCustom+"NOTE: Having any energy shield when the hit occurs grants 50% chance to avoid stun.",
				colorCustom+"POB only applies this modifier when ES > Total incoming damage.",
			)
		}

		if output["StunAvoidChance"] >= 100 {
			output["StunDuration"] = 0
			output["BlockDuration"] = 0
			if breakdown != nil {
				breakdown.set("StunDuration", "Cannot be Stunned")
				breakdown.set("BlockDuration", "Cannot be Stunned")
			}
		} else {
			stunDuration := 1 + modDB.Sum(mod.TypeIncrease, nil, "StunDuration")/100
			baseStunDuration := miscConstant("StunBaseDuration")
			if actor.Actor == env.minionActor() {
				baseStunDuration = overrideOr(modDB, "StunDuration", miscConstant("MinionBaseStunDuration"))
			}
			stunRecovery := 1 + modDB.Sum(mod.TypeIncrease, nil, "StunRecovery")/100
			stunAndBlockRecovery := 1 + modDB.Sum(mod.TypeIncrease, nil, "StunRecovery", "BlockRecovery")/100
			output["StunDuration"] = math.Ceil(baseStunDuration*stunDuration/stunRecovery*miscConstant("ServerTickRate")) / miscConstant("ServerTickRate")
			output["BlockDuration"] = math.Ceil(baseStunDuration*stunDuration/stunAndBlockRecovery*miscConstant("ServerTickRate")) / miscConstant("ServerTickRate")
			//pob2:block:src/Modules/CalcDefence.lua:2815-2833
			if breakdown != nil {
				stun := []string{sFormat("%.2fs ^8(base)", baseStunDuration)}
				block := []string{sFormat("%.2fs ^8(base)", baseStunDuration)}
				if stunDuration != 1 {
					stun = append(stun, sFormat("* %.2f ^8(increased duration)", stunDuration))
					block = append(block, sFormat("* %.2f ^8(increased duration)", stunDuration))
				}
				if stunRecovery != 1 {
					stun = append(stun, sFormat("/ %.2f ^8(increased/reduced recovery)", stunRecovery))
				}
				if stunAndBlockRecovery != 1 {
					block = append(block, sFormat("/ %.2f ^8(increased/reduced block recovery)", stunAndBlockRecovery))
				}
				breakdown.set("StunDuration", append(stun, "rounded up to nearest server tick", sFormat("= %.2fs", output["StunDuration"]))...)
				breakdown.set("BlockDuration", append(block, "rounded up to nearest server tick", sFormat("= %.2fs", output["BlockDuration"]))...)
			}
		}
		output["InterruptStunAvoidChance"] = math.Min(modDB.Sum(mod.TypeBase, nil, "AvoidInterruptStun"), 100)
		effectiveEnemyDamage := output["totalTakenHit"] + output["PhysicalTakenHit"]*0.25
		if damageCategoryConfig != "Average" {
			effectiveEnemyDamage = effectiveEnemyDamage * miscConstant("PhysicalStunMult") * (1 + miscConstant("MeleeStunMult")*3) / 4
		} else if damageCategoryConfig != "Melee" {
			effectiveEnemyDamage *= miscConstant("PhysicalStunMult")
		}
		baseStunChance := math.Min(miscConstant("StunBaseMult")*effectiveEnemyDamage/output["StunThreshold"], 100)
		if baseStunChance > miscConstant("MinStunChanceNeeded") {
			output["SelfStunChance"] = baseStunChance * notAvoidChance / 100
		} else {
			output["SelfStunChance"] = 0
		}
		//pob2:block:src/Modules/CalcDefence.lua:2844-2859
		if breakdown != nil {
			lines := []string{
				sFormat("%d%% ^8(stun multiplier)", miscConstant("StunBaseMult")),
				sFormat("* %.1f ^8(effective enemy stun damage)", effectiveEnemyDamage),
				sFormat("/ %d ^8(stun threshold)", output["StunThreshold"]),
			}
			if baseStunChance < miscConstant("MinStunChanceNeeded") {
				lines = append(lines, sFormat("= %.2f%%", baseStunChance), sFormat("Stun Chance has to be more than %d%% to stun.", miscConstant("MinStunChanceNeeded")))
			} else {
				if notAvoidChance != 100 {
					lines = append(lines, sFormat("* %.2f ^8(1 - chance to avoid stun)", notAvoidChance/100))
				}
				lines = append(lines, sFormat("= %.2f%%", output["SelfStunChance"]))
			}
			breakdown.set("SelfStunChance", lines...)
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:2863-2876
	conditionLowLife := env.ConfigInput["conditionLowLife"].Boolean
	output["LifeRecoverable"] = output["LifeUnreserved"]
	if conditionLowLife {
		lowLifePercentage, ok := output["LowLifePercentage"]
		if !ok {
			lowLifePercentage = miscConstant("LowPoolThreshold")
		}
		output["LifeRecoverable"] = math.Min(output["Life"]*lowLifePercentage/100, output["LifeUnreserved"])
		if output["LifeRecoverable"] < output["LifeUnreserved"] {
			a.setOutputFlag("CappingLife", true)
		}
	}
	if modDB.Flag(nil, "DamageInsteadReservesLife") {
		output["LifeRecoverable"] = (output["LifeCancellableReservation"] / 100) * output["Life"]
	}
	output["LifeRecoverable"] = math.Max(output["LifeRecoverable"], 1)

	//pob2:block:src/Modules/CalcDefence.lua:2879-2894
	{
		recoverable := output["LifeRecoverable"]
		preventedLifeLoss := math.Min(modDB.Sum(mod.TypeBase, nil, "LifeLossPrevented"), 100)
		output["preventedLifeLoss"] = preventedLifeLoss
		initialLifeLossBelowHalfPrevented := modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfPrevented")
		output["preventedLifeLossBelowHalf"] = (1 - output["preventedLifeLoss"]/100) * initialLifeLossBelowHalfPrevented
		if !conditionLowLife {
			portionLife := math.Min(output["Life"]*0.5/recoverable, 1)
			output["preventedLifeLossTotal"] = output["preventedLifeLoss"] + output["preventedLifeLossBelowHalf"]*portionLife
		} else {
			output["preventedLifeLossTotal"] = output["preventedLifeLoss"] + output["preventedLifeLossBelowHalf"]
		}
		output["LifeHitPool"] = calcLifeHitPoolWithLossPrevention(recoverable, output["Life"], preventedLifeLoss, initialLifeLossBelowHalfPrevented)
	}

	//pob2:block:src/Modules/CalcDefence.lua:2927-2941
	a.setOutputFlag("AnyBypass", false)
	output["MinimumBypass"] = 100
	for _, damageType := range dmgTypeList {
		if modDB.Flag(nil, "UnblockedDamageDoesBypassES") {
			output[damageType+"EnergyShieldBypass"] = 100
			a.setOutputFlag("AnyBypass", true)
		} else {
			if o := modDB.Override(nil, damageType+"EnergyShieldBypass"); o != nil {
				output[damageType+"EnergyShieldBypass"] = o.Float()
			} else {
				output[damageType+"EnergyShieldBypass"] = modDB.Sum(mod.TypeBase, nil, damageType+"EnergyShieldBypass")
			}
			if output[damageType+"EnergyShieldBypass"] != 0 {
				a.setOutputFlag("AnyBypass", true)
			}
		}
		output[damageType+"EnergyShieldBypass"] = math.Max(math.Min(output[damageType+"EnergyShieldBypass"], 100), 0)
		output["MinimumBypass"] = math.Min(output["MinimumBypass"], output[damageType+"EnergyShieldBypass"])
	}

	//pob2:block:src/Modules/CalcDefence.lua:2943-3038
	a.setOutputFlag("ehpSectionAnySpecificTypes", false)
	// Mind over Matter
	a.setOutputFlag("OnlySharedMindOverMatter", false)
	a.setOutputFlag("AnySpecificMindOverMatter", false)
	output["sharedMindOverMatter"] = math.Min(modDB.Sum(mod.TypeBase, nil, "DamageTakenFromManaBeforeLife"), 100)
	if output["sharedMindOverMatter"] > 0 {
		a.setOutputFlag("OnlySharedMindOverMatter", true)
		sourcePool := math.Max(output["ManaUnreserved"], 0)
		sourceHitPool := sourcePool
		esBypass := output["MinimumBypass"] / 100
		if modDB.Flag(nil, "EnergyShieldProtectsMana") && esBypass < 1 {
			if esBypass > 0 {
				manaProtected := math.Min(sourcePool/esBypass-sourcePool, output["EnergyShieldRecoveryCap"])
				sourcePool = math.Max(sourcePool-manaProtected, -output["LifeRecoverable"]) + math.Min(sourcePool+output["LifeRecoverable"], manaProtected)/esBypass
				sourceHitPool = math.Max(sourceHitPool-manaProtected, -output["LifeHitPool"]) + math.Min(sourceHitPool+output["LifeHitPool"], manaProtected)/esBypass
			} else {
				sourcePool += output["EnergyShieldRecoveryCap"]
				sourceHitPool = sourcePool
			}
		}
		poolProtected := sourcePool / (output["sharedMindOverMatter"] / 100) * (1 - output["sharedMindOverMatter"]/100)
		hitPoolProtected := sourceHitPool / (output["sharedMindOverMatter"] / 100) * (1 - output["sharedMindOverMatter"]/100)
		if output["sharedMindOverMatter"] >= 100 {
			output["sharedManaEffectiveLife"] = output["LifeRecoverable"] + sourcePool
			output["sharedMoMHitPool"] = output["LifeHitPool"] + sourceHitPool
		} else {
			output["sharedManaEffectiveLife"] = math.Max(output["LifeRecoverable"]-poolProtected, 0) + math.Min(output["LifeRecoverable"], poolProtected)/(1-output["sharedMindOverMatter"]/100)
			output["sharedMoMHitPool"] = math.Max(output["LifeHitPool"]-hitPoolProtected, 0) + math.Min(output["LifeHitPool"], hitPoolProtected)/(1-output["sharedMindOverMatter"]/100)
		}
	} else {
		output["sharedManaEffectiveLife"] = output["LifeRecoverable"]
		output["sharedMoMHitPool"] = output["LifeHitPool"]
	}
	for _, damageType := range dmgTypeList {
		output[damageType+"MindOverMatter"] = math.Min(modDB.Sum(mod.TypeBase, nil, damageType+"DamageTakenFromManaBeforeLife"), 100-output["sharedMindOverMatter"])
		if output[damageType+"MindOverMatter"] > 0 || (output[damageType+"EnergyShieldBypass"] > output["MinimumBypass"] && output["sharedMindOverMatter"] > 0) {
			MindOverMatter := output[damageType+"MindOverMatter"] + output["sharedMindOverMatter"]
			a.setOutputFlag("ehpSectionAnySpecificTypes", true)
			a.setOutputFlag("AnySpecificMindOverMatter", true)
			a.setOutputFlag("OnlySharedMindOverMatter", false)
			sourcePool := math.Max(output["ManaUnreserved"], 0)
			sourceHitPool := sourcePool
			if modDB.Flag(nil, "EnergyShieldProtectsMana") && output[damageType+"EnergyShieldBypass"] < 100 {
				if output[damageType+"EnergyShieldBypass"] > 0 {
					manaProtected := output["EnergyShieldRecoveryCap"] / (1 - output[damageType+"EnergyShieldBypass"]/100) * (output[damageType+"EnergyShieldBypass"] / 100)
					sourcePool = math.Max(sourcePool-manaProtected, -output["LifeRecoverable"]) + math.Min(sourcePool+output["LifeRecoverable"], manaProtected)/(output[damageType+"EnergyShieldBypass"]/100)
					sourceHitPool = math.Max(sourceHitPool-manaProtected, -output["LifeHitPool"]) + math.Min(sourceHitPool+output["LifeHitPool"], manaProtected)/(output[damageType+"EnergyShieldBypass"]/100)
				} else {
					sourcePool += output["EnergyShieldRecoveryCap"]
					sourceHitPool = sourcePool
				}
			}
			poolProtected := sourcePool / (MindOverMatter / 100) * (1 - MindOverMatter/100)
			hitPoolProtected := sourceHitPool / (MindOverMatter / 100) * (1 - MindOverMatter/100)
			if MindOverMatter >= 100 {
				output[damageType+"ManaEffectiveLife"] = output["LifeRecoverable"] + sourcePool
				output[damageType+"MoMHitPool"] = output["LifeHitPool"] + sourceHitPool
			} else {
				output[damageType+"ManaEffectiveLife"] = math.Max(output["LifeRecoverable"]-poolProtected, 0) + math.Min(output["LifeRecoverable"], poolProtected)/(1-MindOverMatter/100)
				output[damageType+"MoMHitPool"] = math.Max(output["LifeHitPool"]-hitPoolProtected, 0) + math.Min(output["LifeHitPool"], hitPoolProtected)/(1-MindOverMatter/100)
			}
		} else {
			output[damageType+"ManaEffectiveLife"] = output["sharedManaEffectiveLife"]
			output[damageType+"MoMHitPool"] = output["sharedMoMHitPool"]
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:3041-3075
	a.setOutputFlag("AnyGuard", false)
	output["sharedGuardAbsorbRate"] = math.Min(modDB.Sum(mod.TypeBase, nil, "GuardAbsorbRate"), 100)
	if output["sharedGuardAbsorbRate"] > 0 {
		a.setOutputFlag("OnlySharedGuard", true)
		output["sharedGuardAbsorb"] = calcLibVal(modDB, "GuardAbsorbLimit", nil)
	}
	for _, damageType := range dmgTypeList {
		output[damageType+"GuardAbsorbRate"] = math.Min(modDB.Sum(mod.TypeBase, nil, damageType+"GuardAbsorbRate"), 100)
		if output[damageType+"GuardAbsorbRate"] > 0 {
			a.setOutputFlag("ehpSectionAnySpecificTypes", true)
			a.setOutputFlag("AnyGuard", true)
			a.setOutputFlag("OnlySharedGuard", false)
			output[damageType+"GuardAbsorb"] = calcLibVal(modDB, damageType+"GuardAbsorbLimit", nil)
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:3078-3100
	a.setOutputFlag("AnyAegis", false)
	output["sharedAegis"] = modDB.Max(nil, "AegisValue")
	output["sharedElementalAegis"] = modDB.Max(nil, "ElementalAegisValue")
	if output["sharedAegis"] > 0 {
		a.setOutputFlag("AnyAegis", true)
	}
	if output["sharedElementalAegis"] > 0 {
		a.setOutputFlag("ehpSectionAnySpecificTypes", true)
		a.setOutputFlag("AnyAegis", true)
	}
	for _, damageType := range dmgTypeList {
		aegisValue := modDB.Max(nil, damageType+"AegisValue")
		if aegisValue > 0 {
			a.setOutputFlag("ehpSectionAnySpecificTypes", true)
			a.setOutputFlag("AnyAegis", true)
			output[damageType+"Aegis"] = aegisValue
		} else {
			output[damageType+"Aegis"] = 0
		}
		if isElemental[damageType] {
			output[damageType+"AegisDisplay"] = output[damageType+"Aegis"] + output["sharedElementalAegis"]
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:3103-3188
	{
		output["FrostShieldLife"] = modDB.Sum(mod.TypeBase, nil, "FrostGlobeHealth")
		output["FrostShieldDamageMitigation"] = modDB.Sum(mod.TypeBase, nil, "FrostGlobeDamageMitigation")

		output["SpectreAllyDamageMitigation"] = modDB.Sum(mod.TypeBase, nil, "TakenFromSpectresBeforeYou")
		if output["SpectreAllyDamageMitigation"] != 0 {
			if o := modDB.Override(nil, "TotalSpectreLife"); o != nil {
				output["TotalSpectreLife"] = o.Float()
			} else {
				output["TotalSpectreLife"] = modDB.Sum(mod.TypeBase, nil, "TotalSpectreLife")
			}
		}

		output["TotemAllyDamageMitigation"] = modDB.Sum(mod.TypeBase, nil, "TakenFromTotemsBeforeYou")
		if output["TotemAllyDamageMitigation"] != 0 {
			output["TotalTotemLife"] = modDB.Sum(mod.TypeBase, nil, "TotalTotemLife")
		}

		output["VaalRejuvenationTotemAllyDamageMitigation"] = modDB.Sum(mod.TypeBase, nil, "takenFromVaalRejuvenationTotemsBeforeYou") + output["TotemAllyDamageMitigation"]
		if output["VaalRejuvenationTotemAllyDamageMitigation"] != output["TotemAllyDamageMitigation"] {
			output["TotalVaalRejuvenationTotemLife"] = modDB.Sum(mod.TypeBase, nil, "TotalVaalRejuvenationTotemLife")
		}

		output["RadianceSentinelAllyDamageMitigation"] = modDB.Sum(mod.TypeBase, nil, "takenFromRadianceSentinelBeforeYou")
		if output["RadianceSentinelAllyDamageMitigation"] != 0 {
			output["TotalRadianceSentinelLife"] = modDB.Sum(mod.TypeBase, nil, "TotalRadianceSentinelLife")
		}

		companionMitigation := modDB.Sum(mod.TypeBase, nil, "TakenFromCompanionBeforeYou")
		companionMitigationFromDeflected := modDB.Sum(mod.TypeBase, nil, "TakenFromCompanionBeforeYouFromDeflected")
		output["CompanionAllyDamageMitigation"] = companionMitigation + companionMitigationFromDeflected*output["DeflectChance"]/100
		if output["CompanionAllyDamageMitigation"] != 0 {
			if o := modDB.Override(nil, "TotalCompanionLife"); o != nil {
				output["TotalCompanionLife"] = o.Float()
			} else {
				output["TotalCompanionLife"] = modDB.Sum(mod.TypeBase, nil, "TotalCompanionLife")
			}
		}

		// actor.partyMembers is not built, so the party-member branch has no energy shield to read.
		output["SoulLinkMitigation"] = modDB.Sum(mod.TypeBase, nil, "TakenFromParentESBeforeYou")
		if output["SoulLinkMitigation"] != 0 {
			if actor.Parent != nil {
				output["AlliedEnergyShield"] = actor.Parent.Output["EnergyShieldRecoveryCap"]
			}
		} else {
			output["SoulLinkMitigation"] = modDB.Sum(mod.TypeBase, nil, "TakenFromPartyMemberESBeforeYou")
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:3191-3194
	output["VaalArcticArmourLife"] = modDB.Sum(mod.TypeBase, nil, "VaalArcticArmourMaxHits")
	output["VaalArcticArmourMitigation"] = math.Min(-modDB.Sum(mod.TypeMore, nil, "VaalArcticArmourMitigation")/100, 1)

	//pob2:block:src/Modules/CalcDefence.lua:3197-3236
	ward := 0.0
	if modDB.Sum(mod.TypeBase, nil, "WardBypass") == 0 {
		ward = output["Ward"]
	}
	for _, damageType := range dmgTypeList {
		output[damageType+"TotalPool"] = output[damageType+"ManaEffectiveLife"]
		output[damageType+"TotalHitPool"] = output[damageType+"MoMHitPool"]
		esBypass := output[damageType+"EnergyShieldBypass"] / 100
		chaosESMultiplier := 1.0
		if damageType == "Chaos" && !modDB.Flag(nil, "ChaosNotDoubleESDamage") {
			chaosESMultiplier = 2
		}
		if modDB.Flag(nil, "EternalLife") {
			output[damageType+"TotalPool"] += output["EnergyShieldRecoveryCap"] / (1 - esBypass) / chaosESMultiplier
			output[damageType+"TotalHitPool"] += output["EnergyShieldRecoveryCap"] / (1 - esBypass) / chaosESMultiplier
		} else if esBypass < 1 {
			if esBypass > 0 {
				poolProtected := output["EnergyShieldRecoveryCap"] / (1 - esBypass) * esBypass / chaosESMultiplier
				output[damageType+"TotalPool"] = math.Max(output[damageType+"TotalPool"]-poolProtected, 0) + math.Min(output[damageType+"TotalPool"], poolProtected)/esBypass
				output[damageType+"TotalHitPool"] = math.Max(output[damageType+"TotalHitPool"]-poolProtected, 0) + math.Min(output[damageType+"TotalHitPool"], poolProtected)/esBypass
			} else {
				output[damageType+"TotalPool"] += output["EnergyShieldRecoveryCap"] / chaosESMultiplier
				output[damageType+"TotalHitPool"] += output["EnergyShieldRecoveryCap"] / chaosESMultiplier
			}
		}
		poolBeforeWard := output[damageType+"TotalPool"]
		output[damageType+"TotalPool"] = poolBeforeWard + ward
		//pob2:block:src/Modules/CalcDefence.lua:3218-3235
		if breakdown != nil {
			lines := []string{sFormat("Life: %d", output["LifeRecoverable"])}
			if output[damageType+"ManaEffectiveLife"] != output["LifeRecoverable"] {
				lines = append(lines, sFormat("Mana through MoM: %d", output[damageType+"ManaEffectiveLife"]-output["LifeRecoverable"]))
			}
			if modDB.Flag(nil, "EternalLife") {
				// the Lua's `and "..." and not flag or ""` yields true, which %s prints
				note := ""
				if damageType == "Chaos" && !modDB.Flag(nil, "ChaosNotDoubleESDamage") {
					note = "true"
				}
				lines = append(lines, sFormat("Energy Shield: %d%s", output["EnergyShieldRecoveryCap"]/chaosESMultiplier, note),
					sFormat("Life change prevented by Eternal Life: %d", poolBeforeWard-output[damageType+"ManaEffectiveLife"]-output["EnergyShieldRecoveryCap"]/chaosESMultiplier))
			} else if esBypass < 1 {
				lines = append(lines, sFormat("Non-bypassed Energy Shield: %d", poolBeforeWard-output[damageType+"ManaEffectiveLife"]))
			}
			if ward > 0 {
				lines = append(lines, sFormat("Runic Ward: %d", ward))
			}
			breakdown.set(damageType+"TotalPool", append(lines, sFormat("Total Pool: %d", output[damageType+"TotalPool"]))...)
		}
	}

	if damageCategoryConfig != "DamageOverTime" {
		//pob2:block:src/Modules/CalcDefence.lua:3240-3247
		{
			DamageIn := newEHPDamageIn()
			for _, damageType := range dmgTypeList {
				DamageIn.damage[damageType] = output[damageType+"TakenHit"]
			}
			DamageIn.limitEHPSpeedup = output["preventedLifeLossTotal"] > 0
			output["NumberOfDamagingHits"] = numberOfHitsToDie(output, actor, DamageIn)
		}

		//pob2:block:src/Modules/CalcDefence.lua:3250-3361
		{
			DamageIn := newEHPDamageIn()
			BlockChance := output["EffectiveBlockChance"] / 100
			if damageCategoryConfig != "Melee" && damageCategoryConfig != "Untyped" {
				BlockChance = output["Effective"+damageCategoryConfig+"BlockChance"] / 100
			}
			blockEffect := 1 - BlockChance*output["BlockEffect"]/100
			suppressChance := 0.0
			suppressionEffect := 1.0
			ExtraAvoidChance := 0.0
			averageAvoidChance := 0.0
			disableGainOnBlock := env.ConfigInput["DisableEHPGainOnBlock"].Boolean
			if !disableGainOnBlock {
				DamageIn.lifeWhenHit = output["LifeOnBlock"] * BlockChance
				DamageIn.manaWhenHit = output["ManaOnBlock"] * BlockChance
				DamageIn.energyShieldWhenHit = output["EnergyShieldOnBlock"] * BlockChance
				DamageIn.wardWhenHit = output["WardOnBlock"] * BlockChance
				if damageCategoryConfig == "Spell" || damageCategoryConfig == "SpellProjectile" {
					DamageIn.energyShieldWhenHit += output["EnergyShieldOnSpellBlock"] * BlockChance
				} else if damageCategoryConfig == "Average" {
					DamageIn.energyShieldWhenHit += output["EnergyShieldOnSpellBlock"] / 2 * BlockChance
				}
			}
			if damageCategoryConfig == "Spell" || damageCategoryConfig == "SpellProjectile" || damageCategoryConfig == "Average" {
				suppressChance = output["EffectiveSpellSuppressionChance"] / 100
			}
			if suppressChance < 1 {
				if damageCategoryConfig == "Average" {
					suppressChance /= 2
				}
				DamageIn.energyShieldWhenHit += output["EnergyShieldOnSuppress"] * suppressChance
				DamageIn.lifeWhenHit += output["LifeOnSuppress"] * suppressChance
				suppressionEffect = 1 - suppressChance*output["SpellSuppressionEffect"]/100
			} else {
				suppressMult := 1.0
				if damageCategoryConfig == "Average" {
					suppressMult = 0.5
				}
				DamageIn.energyShieldWhenHit += output["EnergyShieldOnSuppress"] * suppressMult
				DamageIn.lifeWhenHit += output["LifeOnSuppress"] * suppressMult
			}
			effectiveDeflectMulti := 1.0
			if output["DeflectChance"] < 100 {
				effectiveDeflectMulti = 1 - output["DeflectChance"]*output["DeflectEffect"]/10000
			}
			if damageCategoryConfig == "Projectile" || damageCategoryConfig == "SpellProjectile" {
				ExtraAvoidChance += output["AvoidProjectilesChance"]
			} else if damageCategoryConfig == "Average" {
				ExtraAvoidChance += output["AvoidProjectilesChance"] / 2
			}
			if !disableGainOnBlock {
				if DamageIn.lifeWhenHit != 0 || DamageIn.manaWhenHit != 0 || DamageIn.energyShieldWhenHit != 0 || DamageIn.wardWhenHit != 0 {
					DamageIn.gainWhenHit = true
				}
			} else {
				DamageIn.lifeWhenHit = 0
				DamageIn.manaWhenHit = 0
				DamageIn.energyShieldWhenHit = 0
				DamageIn.wardWhenHit = 0
			}
			specificTypeAvoidance := actor.SpecificTypeAvoidance
			for _, damageType := range dmgTypeList {
				AvoidChance := 0.0
				if specificTypeAvoidance {
					AvoidChance = math.Min(output["Avoid"+damageType+"DamageChance"]+ExtraAvoidChance, miscConstant("AvoidChanceCap"))
					worstOf := 1.0
					if v, ok := configInputNumber(env, "EHPUnluckyWorstOf"); ok {
						worstOf = v
					}
					if worstOf > 1 {
						AvoidChance = AvoidChance / 100 * AvoidChance
						if worstOf == 4 {
							AvoidChance = AvoidChance / 100 * AvoidChance
						}
					}
					averageAvoidChance += AvoidChance
				}
				DamageIn.damage[damageType] = output[damageType+"TakenHit"] * (blockEffect * suppressionEffect * effectiveDeflectMulti * (1 - AvoidChance/100))
			}
			if output["anyRecoup"] > 0 {
				DamageIn.trackRecoupable = true
				for _, damageType := range dmgTypeList {
					output[damageType+"RecoupableDamageTaken"] = 0
				}
			}
			if output["preventedLifeLossTotal"] > 0 {
				DamageIn.trackLifeLossOverTime = true
				output["LifeLossLostOverTime"] = 0
				output["LifeBelowHalfLossLostOverTime"] = 0
			}
			DamageIn.limitEHPSpeedup = DamageIn.trackRecoupable || DamageIn.trackLifeLossOverTime || DamageIn.gainWhenHit
			averageAvoidChance /= 5
			output["ConfiguredDamageChance"] = 100 * (blockEffect * suppressionEffect * effectiveDeflectMulti * (1 - averageAvoidChance/100))
			if output["ConfiguredDamageChance"] != 100 || DamageIn.trackRecoupable || DamageIn.trackLifeLossOverTime || DamageIn.gainWhenHit {
				output["NumberOfMitigatedDamagingHits"] = numberOfHitsToDie(output, actor, DamageIn)
			} else {
				output["NumberOfMitigatedDamagingHits"] = output["NumberOfDamagingHits"]
			}
			//pob2:block:src/Modules/CalcDefence.lua:3343-3360
			if breakdown != nil {
				lines := []string{sFormat("%.2f ^8(chance for block to fail)", 1-BlockChance)}
				if a.OutputFlags["ShowBlockEffect"] {
					lines = append(lines, sFormat("x %.2f ^8(block effect)", output["BlockEffect"]/100))
				}
				if suppressionEffect != 1 {
					lines = append(lines, sFormat("x %.3f ^8(suppression effect)", suppressionEffect))
				}
				if effectiveDeflectMulti != 1 {
					lines = append(lines, sFormat("x %.3f ^8(deflect effect)", effectiveDeflectMulti))
				}
				if averageAvoidChance > 0 {
					lines = append(lines, sFormat("x %.2f ^8(chance for avoidance to fail)", 1-averageAvoidChance/100))
				}
				lines = append(lines, sFormat("= %.1f%% ^8(of damage taken from a%s hit)", output["ConfiguredDamageChance"], hitArticle(damageCategoryConfig)+damageCategoryConfig))
				breakdown.set("ConfiguredDamageChance", lines...)
			}
		}

		//pob2:block:src/Modules/CalcDefence.lua:3365
		output["TotalNumberOfHits"] = output["NumberOfMitigatedDamagingHits"] / (1 - output["ConfiguredNotHitChance"]/100)
		//pob2:block:src/Modules/CalcDefence.lua:3366-3412
		if breakdown != nil {
			var lines []string
			specificTypeAvoidance := actor.SpecificTypeAvoidance
			switch damageCategoryConfig {
			case "Melee", "Projectile":
				if output[damageCategoryConfig+"EvadeChance"] > 0 {
					lines = append(lines, sFormat("%.2f ^8(chance for evasion to fail)", 1-output[damageCategoryConfig+"EvadeChance"]/100))
				}
				if output["AttackDodgeChance"] > 0 {
					lines = append(lines, sFormat("x %.2f ^8(chance for dodge to fail)", 1-output["AttackDodgeChance"]/100))
				}
				if damageCategoryConfig == "Projectile" && !specificTypeAvoidance && output["AvoidProjectilesChance"] > 0 {
					lines = append(lines, sFormat("x %.2f ^8(chance for avoidance to fail)", 1-output["AvoidProjectilesChance"]/100))
				}
			case "Spell", "SpellProjectile":
				if output[damageCategoryConfig+"EvadeChance"] > 0 {
					lines = append(lines, sFormat("%.2f ^8(chance for evasion to fail)", 1-output[damageCategoryConfig+"EvadeChance"]/100))
				}
				if output["SpellDodgeChance"] > 0 {
					lines = append(lines, sFormat("%.2f ^8(chance for dodge to fail)", 1-output["SpellDodgeChance"]/100))
				}
				if damageCategoryConfig == "SpellProjectile" && !specificTypeAvoidance && output["AvoidProjectilesChance"] > 0 {
					lines = append(lines, sFormat("x %.2f ^8(chance for avoidance to fail)", 1-output["AvoidProjectilesChance"]/100))
				}
			case "Average":
				if output["MeleeEvadeChance"] > 0 || output["ProjectileEvadeChance"] > 0 || output["SpellNotHitChance"] > 0 || output["SpellProjectileNotHitChance"] > 0 {
					lines = append(lines, sFormat("%.2f ^8(chance for evasion to fail)", 1-(output["MeleeEvadeChance"]+output["ProjectileEvadeChance"]+output["SpellNotHitChance"]+output["SpellProjectileNotHitChance"])/4/100))
				}
				if output["AttackDodgeChance"] > 0 || output["SpellDodgeChance"] > 0 {
					lines = append(lines, sFormat("x%.2f ^8(chance for dodge to fail)", 1-(output["AttackDodgeChance"]+output["SpellDodgeChance"])/2/100))
				}
				if !specificTypeAvoidance && output["AvoidProjectilesChance"] > 0 {
					lines = append(lines, sFormat("x %.2f ^8(chance for avoidance to fail)", 1-output["AvoidProjectilesChance"]/2/100))
				}
			}
			if output["AvoidAllDamageFromHitsChance"] > 0 {
				lines = append(lines, sFormat("x %.2f ^8(chance for damage avoidance to fail)", 1-output["AvoidAllDamageFromHitsChance"]/100))
			}
			if worstOf, ok := configInputNumber(env, "EHPUnluckyWorstOf"); ok && worstOf > 1 {
				lines = append(lines, sFormat("unlucky worst of %d", worstOf))
			}
			lines = append(lines, sFormat("= %d%% ^8(chance to be hit by a%s hit)", 100-output["ConfiguredNotHitChance"], hitArticle(damageCategoryConfig)+damageCategoryConfig))
			breakdown.set("ConfiguredNotHitChance", lines...)
			breakdown.set("TotalNumberOfHits",
				sFormat("%.2f ^8(Number of mitigated hits)", output["NumberOfMitigatedDamagingHits"]),
				sFormat("/ %.2f ^8(Chance to even be hit)", 1-output["ConfiguredNotHitChance"]/100),
				sFormat("= %.2f ^8(total average number of hits you can take)", output["TotalNumberOfHits"]),
			)
		}

		//pob2:block:src/Modules/CalcDefence.lua:3416
		output["TotalEHP"] = output["TotalNumberOfHits"] * output["totalEnemyDamageIn"]
		//pob2:block:src/Modules/CalcDefence.lua:3417-3423
		if breakdown != nil {
			breakdown.set("TotalEHP",
				sFormat("%.2f ^8(total average number of hits you can take)", output["TotalNumberOfHits"]),
				sFormat("x %d ^8(total incoming damage)", output["totalEnemyDamageIn"]),
				sFormat("= %d ^8(total damage you can take)", output["TotalEHP"]),
			)
		}

		//pob2:block:src/Modules/CalcDefence.lua:3427-3430
		output["enemySkillTime"] = configNumberOr(env, placeholder, "enemySpeed", 700) / (1 + enemyDB.Sum(mod.TypeIncrease, nil, "Speed")/100)
		enemyActionSpeed := actionSpeedMod(actor.Enemy)
		output["enemySkillTime"] = output["enemySkillTime"] / 1000 / enemyActionSpeed
		output["EHPSurvivalTime"] = output["TotalNumberOfHits"] * output["enemySkillTime"]
		//pob2:block:src/Modules/CalcDefence.lua:3431-3437
		if breakdown != nil {
			breakdown.set("EHPSurvivalTime",
				sFormat("%.2f ^8(total average number of hits you can take)", output["TotalNumberOfHits"]),
				sFormat("x %.2f ^8enemy attack/cast time", output["enemySkillTime"]),
				sFormat("= %.2f seconds ^8(total time it would take to die)", output["EHPSurvivalTime"]),
			)
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:3442-3528
	if output["anyRecoup"] > 0 && damageCategoryConfig != "DamageOverTime" {
		totalDamage := 0.0
		totalElementalDamage := 0.0
		totalPhysicalDamageMitigated := output["NumberOfMitigatedDamagingHits"] * (output["PhysicalTakenDamage"] - output["PhysicalTakenHit"])
		type pseudoRecoup struct{ amount, duration float64 }
		extraPseudoRecoup := map[string]pseudoRecoup{}
		for _, damageType := range dmgTypeList {
			totalDamage += output[damageType+"RecoupableDamageTaken"]
			if isElemental[damageType] {
				totalElementalDamage += output[damageType+"RecoupableDamageTaken"]
			}
		}
		recoupTypeList := []string{"Life", "Mana", "EnergyShield"}
		for i, recoupType := range recoupTypeList {
			recoupTime := 8.0
			if modDB.Flag(nil, "4Second"+recoupType+"Recoup") || modDB.Flag(nil, "4SecondRecoup") {
				recoupTime = 4
			}
			output["Total"+recoupType+"RecoupRecovery"] = output[recoupType+"Recoup"] / 100 * totalDamage
			if output["Elemental"+recoupType+"Recoup"] > 0 && totalElementalDamage > 0 {
				output["Total"+recoupType+"RecoupRecovery"] += output["Elemental"+recoupType+"Recoup"] / 100 * totalElementalDamage
			}
			for _, damageType := range dmgTypeList {
				if output[damageType+recoupType+"Recoup"] > 0 && output[damageType+"RecoupableDamageTaken"] > 0 {
					output["Total"+recoupType+"RecoupRecovery"] += output[damageType+recoupType+"Recoup"] / 100 * output[damageType+"RecoupableDamageTaken"]
				}
			}
			output["Total"+recoupType+"PseudoRecoup"] = output["PhysicalDamageMitigated"+recoupType+"PseudoRecoup"] / 100 * totalPhysicalDamageMitigated
			PseudoRecoupDuration, ok := output["PhysicalDamageMitigated"+recoupType+"PseudoRecoupDuration"]
			if !ok {
				PseudoRecoupDuration = 4
			}
			if output["Total"+recoupType+"PseudoRecoup"] != 0 {
				for _, other := range recoupTypeList[i+1:] {
					if modDB.Flag(nil, recoupType+"RegenerationRecovers"+other) && !modDB.Flag(nil, "UnaffectedBy"+other+"Regen") && !modDB.Flag(nil, "No"+other+"Regen") && !modDB.Flag(nil, "CannotGain"+other) {
						extraPseudoRecoup[other] = pseudoRecoup{output["Total"+recoupType+"PseudoRecoup"] * output[other+"RecoveryRateMod"] / output[recoupType+"RecoveryRateMod"], PseudoRecoupDuration}
					}
				}
			}
			if modDB.Flag(nil, "UnaffectedBy"+recoupType+"Regen") {
				output["Total"+recoupType+"PseudoRecoup"] = 0
			}
			extraMax, extraAvg := 0.0, 0.0
			if extra, ok := extraPseudoRecoup[recoupType]; ok {
				extraMax = extra.amount / extra.duration
				extraAvg = extra.amount / (output["EHPSurvivalTime"] + extra.duration)
			}
			output[recoupType+"RecoupRecoveryMax"] = output["Total"+recoupType+"RecoupRecovery"]/recoupTime + output["Total"+recoupType+"PseudoRecoup"]/PseudoRecoupDuration + extraMax
			output[recoupType+"RecoupRecoveryAvg"] = output["Total"+recoupType+"RecoupRecovery"]/(output["EHPSurvivalTime"]+recoupTime) + output["Total"+recoupType+"PseudoRecoup"]/(output["EHPSurvivalTime"]+PseudoRecoupDuration) + extraAvg
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:3531-3564
	_, hasLifeLossLost := output["LifeLossLostOverTime"]
	_, hasLifeBelowHalfLossLost := output["LifeBelowHalfLossLostOverTime"]
	if output["preventedLifeLossTotal"] > 0 && hasLifeLossLost && hasLifeBelowHalfLossLost {
		LifeLossBelowHalfLost := modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfLost") / 100
		LifeLossLost := modDB.Sum(mod.TypeBase, nil, "LifeLossLost") / 100
		output["LifeLossLostMax"] = (output["LifeLossLostOverTime"]*LifeLossLost + output["LifeBelowHalfLossLostOverTime"]*LifeLossBelowHalfLost) / 4
		output["LifeLossLostAvg"] = (output["LifeLossLostOverTime"]*LifeLossLost + output["LifeBelowHalfLossLostOverTime"]*LifeLossBelowHalfLost) / (output["EHPSurvivalTime"] + 4)
	}

	//pob2:block:src/Modules/CalcDefence.lua:3567-3585
	if output["LifeRecoupRecoveryAvg"] > 0 || output["preventedLifeLossTotal"] > 0 {
		output["netLifeRecoupAndLossLostOverTimeMax"] = output["LifeRecoupRecoveryMax"] - output["LifeLossLostMax"]
		output["netLifeRecoupAndLossLostOverTimeAvg"] = output["LifeRecoupRecoveryAvg"] - output["LifeLossLostAvg"]
		if output["LifeRecoupRecoveryAvg"] > 0 && output["preventedLifeLossTotal"] > 0 {
			a.setOutputFlag("showNetRecoup", true)
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:3588-3633
	if env.ConfigInput["PvpScaling"].Boolean {
		PvpTvalue := output["enemySkillTime"]
		PvpMultiplier := 1.0
		if v, ok := configInputNumber(env, "enemyMultiplierPvpDamage"); ok {
			PvpMultiplier = v / 100
		}
		PvpNonElemental1 := miscConstant("PvpNonElemental1")
		PvpNonElemental2 := miscConstant("PvpNonElemental2")
		PvpElemental1 := miscConstant("PvpElemental1")
		PvpElemental2 := miscConstant("PvpElemental2")

		percentageNonElemental := (output["PhysicalTakenHit"] + output["ChaosTakenHit"]) / output["totalTakenHit"]
		percentageElemental := 1 - percentageNonElemental
		portionNonElemental := math.Pow(output["totalTakenHit"]/PvpTvalue/PvpNonElemental2, PvpNonElemental1) * PvpTvalue * PvpNonElemental2 * percentageNonElemental
		portionElemental := math.Pow(output["totalTakenHit"]/PvpTvalue/PvpElemental2, PvpElemental1) * PvpTvalue * PvpElemental2 * percentageElemental
		output["PvPTotalTakenHit"] = (portionNonElemental + portionElemental) * PvpMultiplier
	}

	//pob2:block:src/Modules/CalcDefence.lua:3636-3860
	{
		protect := func(damageType string, pool, mitigation float64) {
			poolProtected := pool / (mitigation / 100) * (1 - mitigation/100)
			output[damageType+"TotalHitPool"] = math.Max(output[damageType+"TotalHitPool"]-poolProtected, 0) + math.Min(output[damageType+"TotalHitPool"], poolProtected)/(1-mitigation/100)
		}
		for _, damageType := range dmgTypeList {
			if modDB.Sum(mod.TypeBase, nil, "WardBypass") == 0 {
				output[damageType+"TotalHitPool"] += output["Ward"]
			}
			aegisDisplay := 0.0
			if isElemental[damageType] {
				aegisDisplay = output[damageType+"AegisDisplay"]
			}
			output[damageType+"TotalHitPool"] += math.Max(math.Max(output[damageType+"Aegis"], output["sharedAegis"]), aegisDisplay)
			// `a or 0 + b or 0` is `a or (0 + b) or 0`: the shared values win whenever they are set.
			GuardAbsorbRate := output["sharedGuardAbsorbRate"]
			if GuardAbsorbRate > 0 {
				GuardAbsorb, ok := output["sharedGuardAbsorb"]
				if !ok {
					GuardAbsorb = output[damageType+"GuardAbsorb"]
				}
				if GuardAbsorbRate >= 100 {
					output[damageType+"TotalHitPool"] += GuardAbsorb
				} else {
					protect(damageType, GuardAbsorb, GuardAbsorbRate)
				}
			}
			if output["FrostShieldLife"] > 0 {
				protect(damageType, output["FrostShieldLife"], output["FrostShieldDamageMitigation"])
			}
			if output["TotalSpectreLife"] > 0 {
				protect(damageType, output["TotalSpectreLife"], output["SpectreAllyDamageMitigation"])
			}
			if output["TotalTotemLife"] > 0 {
				protect(damageType, output["TotalTotemLife"], output["TotemAllyDamageMitigation"])
			}
			if output["TotalVaalRejuvenationTotemLife"] > 0 {
				protect(damageType, output["TotalVaalRejuvenationTotemLife"], output["VaalRejuvenationTotemAllyDamageMitigation"])
			}
			if output["TotalRadianceSentinelLife"] > 0 {
				protect(damageType, output["TotalRadianceSentinelLife"], output["RadianceSentinelAllyDamageMitigation"])
			}
			if output["TotalCompanionLife"] > 0 {
				if output["CompanionAllyDamageMitigation"] >= 100 {
					output[damageType+"TotalHitPool"] += output["TotalCompanionLife"]
				} else {
					protect(damageType, output["TotalCompanionLife"], output["CompanionAllyDamageMitigation"])
				}
			}
			if output["AlliedEnergyShield"] > 0 {
				protect(damageType, output["AlliedEnergyShield"], output["SoulLinkMitigation"])
			}
		}

		for _, damageType := range dmgTypeList {
			partMin := posInf
			useConversionSmoothing := false
			for _, damageConvertedType := range dmgTypeList {
				convertPercent := actor.damageShiftTable[damageType][damageConvertedType]
				takenFlat := output[damageConvertedType+"takenFlat"]
				if convertPercent > 0 || takenFlat != 0 {
					hitTaken := 0.0
					effectiveAppliedArmour := output[damageConvertedType+"EffectiveAppliedArmour"]
					damageConvertedMulti := convertPercent / 100
					totalHitPool := output[damageConvertedType+"TotalHitPool"]
					totalTakenMulti := output[damageConvertedType+"AfterReductionTakenHitMulti"] * (1 - output["VaalArcticArmourMitigation"])
					if damageConvertedMulti <= 0 {
						takenWithoutIncoming := math.Max(takenFlat, 0) * totalTakenMulti
						if takenWithoutIncoming >= totalHitPool {
							hitTaken = 0
						} else {
							hitTaken = posInf
						}
					} else if effectiveAppliedArmour == 0 && convertPercent == 100 {
						totalResistMult := output[damageConvertedType+"ResistTakenHitMulti"]
						drMulti := totalResistMult * (1 - output[damageConvertedType+"DamageReduction"]/100)
						hitTaken = math.Max(totalHitPool/damageConvertedMulti/drMulti-takenFlat, 0) / totalTakenMulti
					} else {
						totalResistMult := output[damageConvertedType+"ResistTakenHitMulti"]

						reductionPercent := 0.0
						if !modDB.Flag(nil, "SelfIgnoreBase"+damageConvertedType+"DamageReduction") {
							reductionPercent = outputOr(output, "Base"+damageConvertedType+"DamageReductionWhenHit", "Base"+damageConvertedType+"DamageReduction")
						}
						enemyOverwhelmPercent := 0.0
						if !modDB.Flag(nil, "SelfIgnore"+damageConvertedType+"DamageReduction") {
							enemyOverwhelmPercent = output[damageConvertedType+"EnemyOverwhelm"]
						}
						flatDR := reductionPercent / 100

						oneMinusFlatPlusOverwhelm := 1 - flatDR + enemyOverwhelmPercent/100
						HP_tTM_tF_DCM_tRM := (totalHitPool/totalTakenMulti - takenFlat) / (damageConvertedMulti * totalResistMult)

						a := miscConstant("ArmourRatio") * damageConvertedMulti * oneMinusFlatPlusOverwhelm
						b := effectiveAppliedArmour*oneMinusFlatPlusOverwhelm - effectiveAppliedArmour - HP_tTM_tF_DCM_tRM*miscConstant("ArmourRatio")*damageConvertedMulti
						c := -HP_tTM_tF_DCM_tRM * effectiveAppliedArmour

						RAW := posInf
						if a != 0 {
							RAW = (math.Sqrt(math.Max(b*b-4*a*c, 0)) - b) / (2 * a)
						}

						noDRMaxHit := totalHitPool / damageConvertedMulti / totalResistMult / totalTakenMulti * (1 - takenFlat*totalTakenMulti/totalHitPool)
						maxDRMaxHit := noDRMaxHit / (1 - (output[damageType+"DamageReductionMax"]-enemyOverwhelmPercent)/100)
						hitTaken = math.Floor(math.Max(math.Min(RAW, maxDRMaxHit), noDRMaxHit))
						useConversionSmoothing = useConversionSmoothing || convertPercent != 100
					}
					partMin = math.Min(partMin, hitTaken)
				}
			}

			enemyDamageMult := output[damageType+"EnemyDamageMult"]

			var finalMaxHit float64
			switch {
			case math.IsInf(partMin, 1):
				finalMaxHit = posInf
			case useConversionSmoothing:
				passIncomingDamage := partMin
				previousOverkill := 0.0
				hasPreviousOverkill := false
				for n := 1; n <= int(miscConstant("maxHitSmoothingPasses")); n++ {
					_, passDamages := takenHitFromDamage(passIncomingDamage, damageType, actor)
					passPools := reducePoolsByDamage(nil, passDamages, actor)
					passOverkill := passPools.overkillDamage - passPools.hitPoolRemaining
					passRatio := 0.0
					for partType := range passDamages {
						partPool := output[partType+"TotalHitPool"]
						if partPool > 0 {
							passRatio = math.Max(passRatio, (passOverkill+partPool)/partPool)
						}
					}
					if passRatio <= 0 {
						passRatio = 1
					}
					stepSize := 1.0
					if n > 1 && hasPreviousOverkill && previousOverkill != 0 && !math.IsNaN(previousOverkill) {
						stepSize = math.Min(math.Abs((passOverkill-previousOverkill)/previousOverkill), 2)
					}
					stepAdjust := 0.0
					if stepSize > 1 {
						stepAdjust = -passOverkill / stepSize
					} else if n > 1 {
						stepAdjust = -passOverkill * stepSize
					}
					previousOverkill = passOverkill
					hasPreviousOverkill = true
					passIncomingDamage = (passIncomingDamage + stepAdjust) / math.Sqrt(passRatio)
					if passOverkill < 1 && passOverkill > -1 {
						break
					}
				}
				finalMaxHit = round(passIncomingDamage / enemyDamageMult)
			default:
				finalMaxHit = round(partMin / enemyDamageMult)
			}

			output[damageType+"MaximumHitTaken"] = finalMaxHit
			//pob2:block:src/Modules/CalcDefence.lua:3801-3845
			if breakdown != nil {
				key := damageType + "MaximumHitTaken"
				e := &BreakdownEntry{
					Label:   "Maximum hit damage breakdown",
					RowList: []map[string]string{},
					ColList: []BreakdownCol{{"Type", "type"}, {"Pool", "pool"}, {"Incoming", "incoming"}, {"Multi", "multi"}, {"Taken", "taken"}},
				}
				breakdown.setEntry(key, e)
				fullTaken, takenDamages := takenHitFromDamage(finalMaxHit*enemyDamageMult, damageType, actor)
				if math.IsNaN(fullTaken) {
					fullTaken = 0
				}
				// takenDamages in dmgTypeList order: PoB-PoE2 walks it with pairs
				for _, takenType := range dmgTypeList {
					takenAmt, ok := takenDamages[takenType]
					if !ok {
						continue
					}
					conversion := actor.damageShiftTable[damageType][takenType]
					incoming := finalMaxHit * enemyDamageMult * conversion / 100
					if math.IsNaN(takenAmt) {
						takenAmt = 0
					}
					takenDamages[takenType] = takenAmt
					e.RowList = append(e.RowList, map[string]string{
						"type":     sFormat("%d%% as %s", conversion, takenType),
						"pool":     sFormat("%d", output[takenType+"TotalHitPool"]),
						"incoming": sFormat("%.0f", incoming),
						"multi":    sFormat("x%.3f", takenAmt/incoming),
						"taken":    sFormat("%.0f", takenAmt),
					})
				}
				fullMulti := fullTaken / finalMaxHit / enemyDamageMult
				lines := []string{"^8Maximum hit is calculated in reverse -", "^8from health pools, via damage reductions, to the max hit:", sFormat("%d ^8(used pool)", fullTaken)}
				if round(fullMulti, 2) != 1 {
					lines = append(lines, sFormat("/ %.2f ^8(modifiers to damage taken)", fullMulti))
				}
				if enemyDamageMult != 1 {
					lines = append(lines, sFormat("/ %.2f ^8(modifiers to enemy damage)", enemyDamageMult))
				}
				approximate := ""
				if useConversionSmoothing {
					approximate = " (approximate)"
				}
				lines = append(lines, sFormat("= %.0f ^8maximum survivable enemy damage%s", finalMaxHit, approximate), "^8Such a hit would drain the following resources:")
				e.Lines = incomingDamageBreakdown(lines, reducePoolsByDamage(nil, takenDamages, actor), output)
			}
		}

		minimum := posInf
		SecondMinimum := posInf
		for _, damageType := range dmgTypeList {
			if output[damageType+"MaximumHitTaken"] < minimum {
				SecondMinimum = minimum
				minimum = output[damageType+"MaximumHitTaken"]
			} else if output[damageType+"MaximumHitTaken"] < SecondMinimum {
				SecondMinimum = output[damageType+"MaximumHitTaken"]
			}
		}
		output["SecondMinimalMaximumHitTaken"] = SecondMinimum
	}

	//pob2:block:src/Modules/CalcDefence.lua:3863-3872
	for _, damageType := range dmgTypeList {
		output[damageType+"DotEHP"] = output[damageType+"TotalPool"] / output[damageType+"TakenDotMult"]
		if breakdown != nil {
			breakdown.set(damageType+"DotEHP",
				sFormat("Total Pool: %d", output[damageType+"TotalPool"]),
				sFormat("Dot Damage Taken modifier: %.2f", output[damageType+"TakenDotMult"]),
				sFormat("Total Effective Dot Pool: %d", output[damageType+"DotEHP"]),
			)
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:3875-4283
	{
		degenSplit := func(damageType string, typeDegen float64) (float64, float64, float64) {
			var lifeDegen, manaDegen, energyShieldDegen float64
			takenFromMana := output[damageType+"MindOverMatter"] + output["sharedMindOverMatter"]
			if output["EnergyShieldRegenRecovery"] > 0 {
				if modDB.Flag(nil, "EnergyShieldProtectsMana") {
					lifeDegen = typeDegen * (1 - takenFromMana/100)
					energyShieldDegen = typeDegen * (1 - output[damageType+"EnergyShieldBypass"]/100) * (takenFromMana / 100)
				} else {
					lifeDegen = typeDegen * (output[damageType+"EnergyShieldBypass"] / 100) * (1 - takenFromMana/100)
					energyShieldDegen = typeDegen * (1 - output[damageType+"EnergyShieldBypass"]/100)
				}
				manaDegen = typeDegen * (output[damageType+"EnergyShieldBypass"] / 100) * (takenFromMana / 100)
			} else {
				lifeDegen = typeDegen * (1 - takenFromMana/100)
				manaDegen = typeDegen * (takenFromMana / 100)
			}
			return lifeDegen, manaDegen, energyShieldDegen
		}

		output["TotalBuildDegen"] = 0
		for _, damageType := range dmgTypeList {
			baseVal := modDB.Sum(mod.TypeBase, nil, damageType+"Degen")
			if baseVal > 0 {
				for _, damageConvertedType := range dmgTypeList {
					convertPercent := actor.damageOverTimeShiftTable[damageType][damageConvertedType]
					if convertPercent > 0 {
						total := baseVal * (convertPercent / 100) * output[damageConvertedType+"TakenDotMult"]
						output[damageConvertedType+"BuildDegen"] += total
						output["TotalBuildDegen"] += total
					}
				}
			}
		}
		if output["TotalBuildDegen"] == 0 {
			delete(output, "TotalBuildDegen")
		} else {
			output["NetLifeRegen"] = output["LifeRegenRecovery"]
			output["NetManaRegen"] = output["ManaRegenRecovery"]
			output["NetEnergyShieldRegen"] = output["EnergyShieldRegenRecovery"]
			output["NetWardRegen"] = output["WardRegenRecovery"]
			totalLifeDegen := 0.0
			totalManaDegen := 0.0
			totalEnergyShieldDegen := 0.0
			for _, damageType := range dmgTypeList {
				if buildDegen, ok := output[damageType+"BuildDegen"]; ok {
					lifeDegen, manaDegen, energyShieldDegen := degenSplit(damageType, buildDegen)
					totalLifeDegen += lifeDegen
					totalManaDegen += manaDegen
					totalEnergyShieldDegen += energyShieldDegen
				}
			}
			output["NetLifeRegen"] -= totalLifeDegen
			output["NetManaRegen"] -= totalManaDegen
			output["NetEnergyShieldRegen"] -= totalEnergyShieldDegen
			output["TotalNetRegen"] = output["NetLifeRegen"] + output["NetManaRegen"] + output["NetEnergyShieldRegen"] + output["NetWardRegen"]
		}
		enemyCritAilmentEffect := 1 + output["EnemyCritChance"]/100*0.5*(1-output["CritExtraDamageReduction"]/100)
		enemyCritThreshold := 10.1
		enemyBleedChance := 0.0
		enemyIgniteChance := 0.0
		if output["SelfIgniteEffect"] != 0 && output["IgniteAvoidChance"] < 100 && output["SelfIgniteDuration"] != 0 && damageCategoryConfig != "DamageOverTime" {
			enemyIgniteChance = enemyDB.Sum(mod.TypeBase, nil, "IgniteChance", "ElementalAilmentChance", "AilmentChance")
			enemyCritAilmentChance := 0.0
			if !modDB.Flag(nil, "CritsOnYouDontAlwaysApplyElementalAilments") && (output["EnemyCritChance"] > enemyCritThreshold || enemyIgniteChance > 0) {
				enemyCritAilmentChance = output["EnemyCritChance"]
			}
			enemyIgniteChance = (enemyCritAilmentChance + (1-enemyCritAilmentChance/100)*enemyIgniteChance) * (1 - output["IgniteAvoidChance"]/100)
		}
		enemyPoisonChance := 0.0
		if output["SelfPoisonEffect"] != 0 && output["PoisonAvoidChance"] < 100 && output["SelfPoisonDuration"] != 0 && damageCategoryConfig != "DamageOverTime" {
			enemyPoisonChance = enemyDB.Sum(mod.TypeBase, nil, "PoisonChance") * (1 - output["PoisonAvoidChance"]/100)
		}
		if damageCategoryConfig == "DamageOverTime" || (enemyIgniteChance+enemyPoisonChance+enemyBleedChance) > 0 {
			output["TotalDegen"] = output["TotalBuildDegen"]
			addEnemyDegen := func(baseVal float64, damageType string) {
				for _, damageConvertedType := range dmgTypeList {
					convertPercent := actor.damageOverTimeShiftTable[damageType][damageConvertedType]
					if convertPercent > 0 {
						total := baseVal * (convertPercent / 100) * output[damageConvertedType+"TakenDotMult"]
						output[damageConvertedType+"EnemyDegen"] += total
						output["TotalDegen"] += total
					}
				}
			}
			if damageCategoryConfig == "DamageOverTime" {
				for _, damageType := range dmgTypeList {
					baseVal, ok := configInputNumber(env, "enemy"+damageType+"Damage")
					if !ok {
						baseVal = placeholder["enemy"+damageType+"Damage"]
					}
					if baseVal > 0 {
						addEnemyDegen(baseVal, damageType)
					}
				}
			} else {
				type enemyAilment struct {
					name        string
					damageType  string
					sourceTypes []string
				}
				// ailmentList is walked with pairs() in Lua; the order here is Bleed, Ignite, Poison.
				var ailmentList []enemyAilment
				if enemyBleedChance > 0 {
					ailmentList = append(ailmentList, enemyAilment{"Bleed", "Physical", []string{"Physical"}})
				}
				if enemyIgniteChance > 0 {
					sourceTypes := []string{"Fire"}
					if enemyDB.Flag(nil, "AllDamageIgnites") {
						sourceTypes = dmgTypeList
					}
					ailmentList = append(ailmentList, enemyAilment{"Ignite", "Fire", sourceTypes})
				}
				if enemyPoisonChance > 0 {
					ailmentList = append(ailmentList, enemyAilment{"Poison", "Chaos", []string{"Physical", "Chaos"}})
				}
				for _, ailment := range ailmentList {
					baseVal := 0.0
					for _, damageType := range ailment.sourceTypes {
						baseVal += output[damageType+"TakenDamage"]
					}
					baseVal = baseVal * miscConstant(ailment.name+"PercentBase") * (enemyCritAilmentEffect / output["EnemyCritEffect"]) * output["Self"+ailment.name+"Effect"] / 100
					if baseVal > 0 {
						addEnemyDegen(baseVal, ailment.damageType)
					}
				}
			}
			if output["TotalDegen"] == output["TotalBuildDegen"] {
				delete(output, "TotalDegen")
			} else {
				output["ComprehensiveNetLifeRegen"] = output["LifeRegenRecovery"]
				output["ComprehensiveNetManaRegen"] = output["ManaRegenRecovery"]
				output["ComprehensiveNetWardRegen"] = output["WardRegenRecovery"]
				output["ComprehensiveNetEnergyShieldRegen"] = output["EnergyShieldRegenRecovery"]
				totalLifeDegen := 0.0
				totalManaDegen := 0.0
				totalEnergyShieldDegen := 0.0
				for _, damageType := range dmgTypeList {
					typeDegen := output[damageType+"BuildDegen"] + output[damageType+"EnemyDegen"]
					if typeDegen != 0 {
						lifeDegen, manaDegen, energyShieldDegen := degenSplit(damageType, typeDegen)
						totalLifeDegen += lifeDegen
						totalManaDegen += manaDegen
						totalEnergyShieldDegen += energyShieldDegen
					}
				}
				output["ComprehensiveNetLifeRegen"] += output["LifeRecoupRecoveryAvg"] - totalLifeDegen - output["LifeLossLostAvg"]
				output["ComprehensiveNetManaRegen"] += output["ManaRecoupRecoveryAvg"] - totalManaDegen
				output["ComprehensiveNetEnergyShieldRegen"] += output["EnergyShieldRecoupRecoveryAvg"] - totalEnergyShieldDegen
				output["ComprehensiveTotalNetRegen"] = output["ComprehensiveNetLifeRegen"] + output["ComprehensiveNetManaRegen"] + output["ComprehensiveNetWardRegen"] + output["ComprehensiveNetEnergyShieldRegen"]
			}
		}
	}

	if breakdown != nil {
		incomingHitBreakdown(breakdown, actor)
	}
}

//pob2:block:src/Modules/CalcDefence.lua:4286-4387
func incomingHitBreakdown(breakdown *Breakdown, actor *ehpActor) {
	output := actor.Output
	takenHit := map[string]float64{}
	total := breakdown.Entries["totalTakenHit"]
	for _, damageType := range dmgTypeList {
		if output[damageType+"TakenHit"] > 0 {
			takenHit[damageType] = output[damageType+"TakenHit"]
		}
		total.RowList = append(total.RowList, map[string]string{
			"type":     damageType,
			"incoming": sFormat("%.1f incoming damage", output[damageType+"TakenDamage"]),
			"mult":     sFormat("x %.3f damage mult", output[damageType+"TakenHitMult"]),
			"value":    sFormat("%d", math.Ceil(output[damageType+"TakenHit"])),
		})
	}
	poolsRemaining := reducePoolsByDamage(nil, takenHit, actor)
	total.Lines = incomingDamageBreakdown(append(total.Lines, "Such a hit would drain the following resources:"), poolsRemaining, output)

	for _, damageType := range dmgTypeList {
		lost := poolsRemaining.resourcesLostToTypeDamage[damageType]
		sum := 0.0
		lines := []string{
			sFormat("Final %s Damage taken:", damageType),
			sFormat("%.1f incoming damage", output[damageType+"TakenDamage"]),
			sFormat("x %.3f damage mult", output[damageType+"TakenHitMult"]),
			sFormat("= %.1f", output[damageType+"TakenHit"]),
			"This part of the hit drains the following resources:",
		}
		add := func(key, format string, args ...any) {
			if v, ok := lost[key]; ok {
				sum += v
				lines = append(lines, sFormat(format, append([]any{v}, args...)...))
			}
		}
		add("frostShield", "\t%d "+colorGem+"Frost Shield Life")
		add("spectres", "\t%d "+colorGem+"Total Spectre Life")
		add("totems", "\t%d "+colorGem+"Total Totem Life")
		add("vaalRejuvenationTotems", "\t%d "+colorGem+"Total Vaal Rejuvenation Totem Life")
		add("radianceSentinel", "\t%d "+colorGem+"Total Sentinel of Radiance Life")
		add("soulLink", "\t%d "+colorGem+"Total Allied Energy shield")
		add("aegis", "\t%d "+colorGem+"%s Aegis charge", damageType)
		add("sharedElementalAegis", "\t%d "+colorGem+"Elemental Aegis charge")
		add("sharedAegis", "\t%d "+colorGem+"Shared Aegis charge")
		add("guard", "\n\t%d "+colorScourge+"%s Guard charge", damageType)
		add("sharedGuard", "\t%d "+colorScourge+"Shared Guard charge")
		// the Lua's `and "..." and not flag or ""` yields true, which %s prints
		chaosNote := ""
		if damageType == "Chaos" && !actor.ModDB.Flag(nil, "ChaosNotDoubleESDamage") {
			chaosNote = "true"
		}
		add("energyShield", "\t%d "+colorES+"Energy Shield%s", chaosNote)
		if v, ok := lost["eternalLifePrevented"]; ok {
			lines = append(lines, sFormat("\t%d "+colorPositive+"Life change prevented by Eternal Life", v))
		}
		add("mana", "\t%d "+colorMana+"Mana")
		if lost["lifeLossPrevented"] > 0 {
			add("lifeLossPrevented", "\t%d "+colorLife+"Life ^7Loss Prevented")
		}
		if lost["life"] > 0 {
			add("life", "\t%d "+colorLife+"Life")
		}
		add("runicWard", "\t%d "+colorWard+"Runic Ward")
		add("overkill", "\t%d "+colorNegative+"Overkill damage")
		breakdown.set(damageType+"TakenHit", append(lines, sFormat("\t^8(%d total)", sum))...)
	}
}

func countTrue(bs ...bool) int {
	n := 0
	for _, b := range bs {
		if b {
			n++
		}
	}
	return n
}

// colorCustom is colorCodes.CUSTOM.
//
//pob2:src/Data/Global.lua:19
const colorCustom = "^x5CF0BB"

// hitArticle is the Lua's `(cfg == "Average" and "n " or (cfg == "Untyped" and "n " or " "))`.
func hitArticle(damageCategoryConfig string) string {
	if damageCategoryConfig == "Average" || damageCategoryConfig == "Untyped" {
		return "n "
	}
	return " "
}

//pob2:block:src/Data/Global.lua:10-82
const (
	colorGem      = "^x74CABF"
	colorPositive = "^x33FF77"
	colorScourge  = "^xFF6E25"
	colorLife     = "^xE05030"
	colorMana     = "^x7070FF"
	colorES       = "^x88FFFF"
	colorWard     = "^xFFFF77"
)

func (p *ehpPools) ally(name string) *ehpAlly {
	for _, a := range p.alliesTakenBeforeYou {
		if a.name == name {
			return a
		}
	}
	return &ehpAlly{name: name}
}

// incomingDamageBreakdown appends the resources a hit drained.
//
//pob2:src/Modules/CalcDefence.lua:706-772
func incomingDamageBreakdown(lines []string, pools *ehpPools, output map[string]float64) []string {
	for _, a := range []struct{ key, ally, label string }{
		{"FrostShieldLife", "frostShield", "Frost Shield Life"},
		{"TotalSpectreLife", "spectres", "Total Spectre Life"},
		{"TotalTotemLife", "totems", "Total Totem Life"},
		{"TotalVaalRejuvenationTotemLife", "vaalRejuvenationTotems", "Total Vaal Rejuvenation Totem Life"},
		{"TotalRadianceSentinelLife", "radianceSentinel", "Total Sentinel of Radiance Life"},
		{"TotalCompanionLife", "companion", "Total Companion Life"},
		{"AlliedEnergyShield", "soulLink", "Total Allied Energy shield"},
	} {
		if output[a.key] > 0 {
			remaining := pools.ally(a.ally).remaining
			lines = append(lines, sFormat("\t%d "+colorGem+a.label+" ^7(%d remaining)", output[a.key]-remaining, remaining))
		}
	}
	for _, damageType := range dmgTypeList {
		if v, ok := pools.resourcesLostToTypeDamage[damageType]["aegis"]; ok {
			lines = append(lines, sFormat("\t%d "+colorGem+"%s Aegis charge ^7(%d remaining)", v, damageType, pools.aegis[damageType]))
		}
	}
	if v, ok := output["sharedElementalAegis"]; ok && v != pools.aegis["sharedElemental"] {
		lines = append(lines, sFormat("\t%d "+colorGem+"Elemental Aegis charge ^7(%d remaining)", v-pools.aegis["sharedElemental"], pools.aegis["sharedElemental"]))
	}
	if output["sharedAegis"] > 0 {
		lines = append(lines, sFormat("\t%d "+colorGem+"Shared Aegis charge ^7(%d remaining)", output["sharedAegis"]-pools.aegis["shared"], pools.aegis["shared"]))
	}
	eternalLifePrevented := 0.0
	for _, damageType := range dmgTypeList {
		eternalLifePrevented += pools.resourcesLostToTypeDamage[damageType]["eternalLifePrevented"]
		if v, ok := pools.resourcesLostToTypeDamage[damageType]["guard"]; ok {
			lines = append(lines, sFormat("\n\t%d "+colorScourge+"%s Guard charge ^7(%d remaining)", v, damageType, pools.guard[damageType]))
		}
	}
	if output["sharedGuardAbsorb"] > 0 {
		lines = append(lines, sFormat("\t%d "+colorScourge+"Shared Guard charge ^7(%d remaining)", output["sharedGuardAbsorb"]-pools.guard["shared"], pools.guard["shared"]))
	}
	if output["EnergyShieldRecoveryCap"] != pools.energyShield && output["EnergyShieldRecoveryCap"] > 0 {
		lines = append(lines, sFormat("\t%d "+colorES+"Energy Shield ^7(%d remaining)", output["EnergyShieldRecoveryCap"]-pools.energyShield, pools.energyShield))
	}
	if eternalLifePrevented > 0 {
		lines = append(lines, sFormat("\t%d "+colorPositive+"Life change prevented by Eternal Life", eternalLifePrevented))
	}
	if output["ManaUnreserved"] != pools.mana && output["ManaUnreserved"] > 0 {
		lines = append(lines, sFormat("\t%d "+colorMana+"Mana ^7(%d remaining)", output["ManaUnreserved"]-pools.mana, pools.mana))
	}
	if pools.lifeLossLostOverTime+pools.lifeBelowHalfLossLostOverTime > 0 {
		lines = append(lines, sFormat("\t%d "+colorLife+"Life ^7Loss Prevented", pools.lifeLossLostOverTime+pools.lifeBelowHalfLossLostOverTime))
	}
	lines = append(lines, sFormat("\t%d "+colorLife+"Life ^7(%d remaining)", output["LifeRecoverable"]-pools.life, pools.life))
	if output["Ward"] > 0 {
		lines = append(lines, sFormat("\t%d "+colorWard+"Runic Ward ^7(%d remaining)", output["Ward"]-pools.ward, pools.ward))
	}
	if pools.overkillDamage >= 1 {
		lines = append(lines, sFormat("\t%d "+colorNegative+"Overkill damage", pools.overkillDamage))
	}
	return lines
}
