package calculator2

import (
	"math"

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
// breakdowns and is not kept.
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
			}
		}
		damageTakenThatCanBeRecouped[damageType] += damageRemainder
		if aegis[damageType] > 0 {
			tempDamage := math.Min(damageRemainder, aegis[damageType])
			aegis[damageType] -= tempDamage
			damageRemainder -= tempDamage
		}
		if isElemental[damageType] && aegis["sharedElemental"] > 0 {
			tempDamage := math.Min(damageRemainder, aegis["sharedElemental"])
			aegis["sharedElemental"] -= tempDamage
			damageRemainder -= tempDamage
		}
		if aegis["shared"] > 0 {
			tempDamage := math.Min(damageRemainder, aegis["shared"])
			aegis["shared"] -= tempDamage
			damageRemainder -= tempDamage
		}
		if guard[damageType] > 0 {
			tempDamage := math.Min(damageRemainder*output[damageType+"GuardAbsorbRate"]/100, guard[damageType])
			guard[damageType] -= tempDamage
			damageRemainder -= tempDamage
		}
		if guard["shared"] > 0 {
			tempDamage := math.Min(damageRemainder*output["sharedGuardAbsorbRate"]/100, guard["shared"])
			guard["shared"] -= tempDamage
			damageRemainder -= tempDamage
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
		} else if energyShield > 0 && esBypass < 1 {
			MoMEBPool := energyShield
			if esBypass > 0 {
				MoMEBPool = math.Min((MoMPool+lifeHitPool)/esBypass*esDamageTypeMultiplier-(MoMPool+lifeHitPool), energyShield)
			}
			tempDamage := math.Min(damageRemainder*(1-esBypass), MoMEBPool/esDamageTypeMultiplier)
			esPoolRemaining = math.Min(esPoolRemaining, MoMEBPool-tempDamage*esDamageTypeMultiplier)
			energyShield -= tempDamage * esDamageTypeMultiplier
			damageRemainder -= tempDamage
		}
		if MoMEffect > 0 && mana > 0 {
			MoMDamage := damageRemainder * MoMEffect
			tempDamage := math.Min(MoMDamage, MoMPool)
			MoMPoolRemaining = math.Min(MoMPoolRemaining, MoMPool-tempDamage)
			mana -= tempDamage
			damageRemainder -= tempDamage
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
				if life <= halfLife {
					unspecificallyLowLifePreventedDamage := damageRemainder * preventPercent
					lifeLossLostOverTime += unspecificallyLowLifePreventedDamage
					damageRemainder -= unspecificallyLowLifePreventedDamage
					specificallyLowLifePreventedDamage := damageRemainder * preventBelowHalfPercent
					lifeBelowHalfLossLostOverTime += specificallyLowLifePreventedDamage
					damageRemainder -= specificallyLowLifePreventedDamage
				}
			} else {
				tempDamage := damageRemainder * output["preventedLifeLoss"] / 100
				lifeLossLostOverTime += tempDamage
				damageRemainder -= tempDamage
			}
		}
		if life > 0 {
			tempDamage := math.Min(damageRemainder, life-1)
			life -= tempDamage
			damageRemainder -= tempDamage
			if ward > 0 && wardApplies {
				tempDamage := math.Min(damageRemainder, ward)
				ward -= tempDamage
				damageRemainder -= tempDamage
			}
			if damageRemainder > 0 {
				tempDamage := math.Min(damageRemainder, life)
				life -= tempDamage
				damageRemainder -= tempDamage
			}
		}
		overkillDamage += damageRemainder
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

// buildDefenceEstimations writes no breakdowns. Its boolean outputs (AnyTakenReflect, CappingLife,
// AnyBypass, the Any*/Only* section flags, showNetRecoup) have no place in the float output table and
// are kept as locals where the function reads them back.
//
//pob2:src/Modules/CalcDefence.lua:2222-4389
func buildDefenceEstimations(env *Environment, a *Actor) {
	actor := &ehpActor{Actor: a}
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
		// output.specificTypeAvoidance is a boolean calcs.defence sets; it is read as its float here.
		specificTypeAvoidance := output["specificTypeAvoidance"] != 0
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
			if !hasDamage {
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
			}
			output["totalEnemyDamage"] += output[damageType+"EnemyDamage"]
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
	}
	for _, damageType := range dmgTypeList {
		for _, damageConvertedType := range dmgTypeList {
			if damageType != damageConvertedType {
				damage := output[damageType+"EnemyDamage"] * actor.damageShiftTable[damageType][damageConvertedType] / 100
				output[damageConvertedType+"TakenDamage"] += damage
			}
		}
	}
	output["totalTakenDamage"] = 0
	for _, damageType := range dmgTypeList {
		output["totalTakenDamage"] += output[damageType+"TakenDamage"]
	}

	//pob2:block:src/Modules/CalcDefence.lua:2467-2526
	anyTakenReflect := false
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
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:2529-2742
	output["totalTakenHit"] = 0
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
		percentOfEvasionApplies := 0.0
		if !modDB.Flag(nil, "EvasionDoesNotApplyTo"+damageType+"DamageTaken") {
			percentOfEvasionApplies = modDB.Sum(mod.TypeBase, nil, "EvasionAppliesTo"+damageType+"DamageTaken")
		}
		percentOfEnergyShieldApplies := 0.0
		if !modDB.Flag(nil, "EnergyShieldDoesNotApplyTo"+damageType+"DamageTaken") {
			percentOfEnergyShieldApplies = modDB.Sum(mod.TypeBase, nil, "EnergyShieldAppliesTo"+damageType+"DamageTaken")
		}
		if percentOfEvasionApplies > 0 {
			effectiveAppliedArmour += math.Max(output["Evasion"]*percentOfEvasionApplies/100, 0)
		}
		if percentOfEnergyShieldApplies > 0 {
			effectiveAppliedArmour += math.Max(output["EnergyShield"]*percentOfEnergyShieldApplies/100, 0)
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
	}

	//pob2:block:src/Modules/CalcDefence.lua:2745-2860
	{
		stunThresholdBase := 0.0
		switch {
		case modDB.Flag(nil, "StunThresholdBasedOnEnergyShieldInsteadOfLife"):
			stunThresholdBase = output["EnergyShield"] * modDB.Sum(mod.TypeBase, nil, "StunThresholdEnergyShieldPercent") / 100
		case modDB.Flag(nil, "StunThresholdBasedOnManaInsteadOfLife"):
			stunThresholdBase = output["Mana"] * modDB.Sum(mod.TypeBase, nil, "StunThresholdManaPercent") / 100
		case modDB.Flag(nil, "ChaosInoculation"):
			stunThresholdBase = modDB.Sum(mod.TypeBase, nil, "Life")
		default:
			stunThresholdBase = output["Life"]
		}
		if modDB.Flag(nil, "AddESToStunThreshold") {
			stunThresholdBase += output["EnergyShield"] * modDB.Sum(mod.TypeBase, nil, "ESToStunThresholdPercent") / 100
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

		if output["StunAvoidChance"] >= 100 {
			output["StunDuration"] = 0
			output["BlockDuration"] = 0
		} else {
			stunDuration := 1 + modDB.Sum(mod.TypeIncrease, nil, "StunDuration")/100
			baseStunDuration := miscConstant("StunBaseDuration")
			stunRecovery := 1 + modDB.Sum(mod.TypeIncrease, nil, "StunRecovery")/100
			stunAndBlockRecovery := 1 + modDB.Sum(mod.TypeIncrease, nil, "StunRecovery", "BlockRecovery")/100
			output["StunDuration"] = math.Ceil(baseStunDuration*stunDuration/stunRecovery*miscConstant("ServerTickRate")) / miscConstant("ServerTickRate")
			output["BlockDuration"] = math.Ceil(baseStunDuration*stunDuration/stunAndBlockRecovery*miscConstant("ServerTickRate")) / miscConstant("ServerTickRate")
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
	output["MinimumBypass"] = 100
	for _, damageType := range dmgTypeList {
		if modDB.Flag(nil, "UnblockedDamageDoesBypassES") {
			output[damageType+"EnergyShieldBypass"] = 100
		} else if o := modDB.Override(nil, damageType+"EnergyShieldBypass"); o != nil {
			output[damageType+"EnergyShieldBypass"] = o.Float()
		} else {
			output[damageType+"EnergyShieldBypass"] = modDB.Sum(mod.TypeBase, nil, damageType+"EnergyShieldBypass")
		}
		output[damageType+"EnergyShieldBypass"] = math.Max(math.Min(output[damageType+"EnergyShieldBypass"], 100), 0)
		output["MinimumBypass"] = math.Min(output["MinimumBypass"], output[damageType+"EnergyShieldBypass"])
	}

	//pob2:block:src/Modules/CalcDefence.lua:2945-3038
	output["sharedMindOverMatter"] = math.Min(modDB.Sum(mod.TypeBase, nil, "DamageTakenFromManaBeforeLife"), 100)
	if output["sharedMindOverMatter"] > 0 {
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
	output["sharedGuardAbsorbRate"] = math.Min(modDB.Sum(mod.TypeBase, nil, "GuardAbsorbRate"), 100)
	if output["sharedGuardAbsorbRate"] > 0 {
		output["sharedGuardAbsorb"] = calcLibVal(modDB, "GuardAbsorbLimit", nil)
	}
	for _, damageType := range dmgTypeList {
		output[damageType+"GuardAbsorbRate"] = math.Min(modDB.Sum(mod.TypeBase, nil, damageType+"GuardAbsorbRate"), 100)
		if output[damageType+"GuardAbsorbRate"] > 0 {
			output[damageType+"GuardAbsorb"] = calcLibVal(modDB, damageType+"GuardAbsorbLimit", nil)
		}
	}

	//pob2:block:src/Modules/CalcDefence.lua:3078-3100
	output["sharedAegis"] = modDB.Max(nil, "AegisValue")
	output["sharedElementalAegis"] = modDB.Max(nil, "ElementalAegisValue")
	for _, damageType := range dmgTypeList {
		aegisValue := modDB.Max(nil, damageType+"AegisValue")
		if aegisValue > 0 {
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
			specificTypeAvoidance := output["specificTypeAvoidance"] != 0
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
		}

		//pob2:block:src/Modules/CalcDefence.lua:3365
		output["TotalNumberOfHits"] = output["NumberOfMitigatedDamagingHits"] / (1 - output["ConfiguredNotHitChance"]/100)

		//pob2:block:src/Modules/CalcDefence.lua:3416
		output["TotalEHP"] = output["TotalNumberOfHits"] * output["totalEnemyDamageIn"]

		//pob2:block:src/Modules/CalcDefence.lua:3427-3430
		output["enemySkillTime"] = configNumberOr(env, placeholder, "enemySpeed", 700) / (1 + enemyDB.Sum(mod.TypeIncrease, nil, "Speed")/100)
		enemyActionSpeed := actionSpeedMod(actor.Enemy)
		output["enemySkillTime"] = output["enemySkillTime"] / 1000 / enemyActionSpeed
		output["EHPSurvivalTime"] = output["TotalNumberOfHits"] * output["enemySkillTime"]
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
}
