package calculator

import (
	"fmt"
	"math"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

//pob:src/Modules/CalcDefence.lua:44-54
type PoolState struct {
	Remaining float64
	Percent   float64
}

type PoolTable struct {
	AlliesTakenBeforeYou map[string]*PoolState
	Aegis                map[string]float64
	Guard                map[string]float64
	Ward                 *float64
	WardActiveChance     *float64
	EnergyShield         *float64
	Mana                 *float64
	Life                 *float64

	DamageTakenThatCanBeRecouped  map[data.DamageType]float64
	LifeLossLostOverTime          float64
	LifeBelowHalfLossLostOverTime float64

	OverkillDamage   float64
	HitPoolRemaining float64
	ResourcesLost    map[data.DamageType]map[string]float64
}

//pob:src/Modules/CalcDefence.lua:44-54
func buildAllyLifePools(actor *Actor) map[string]*PoolState {
	pools := make(map[string]*PoolState, len(allyLifePoolList))
	for _, ally := range allyLifePoolList {
		life := actor.Output[ally.life]
		mitigation := actor.Output[ally.mitigation]
		if life > 0 && mitigation > 0 {
			pools[ally.key] = &PoolState{Remaining: life, Percent: math.Min(mitigation, 100) / 100}
		}
	}
	return pools
}

func lostTo(m map[data.DamageType]map[string]float64, dt data.DamageType, key string, v float64) {
	if v < 1 {
		return
	}
	if m[dt] == nil {
		m[dt] = make(map[string]float64, 4)
	}
	m[dt][key] += float64(v)
}

func lostSet(m map[data.DamageType]map[string]float64, dt data.DamageType, key string, v float64) {
	if m[dt] == nil {
		m[dt] = make(map[string]float64, 4)
	}
	m[dt][key] = v
}

//pob:block:src/Modules/CalcDefence.lua:188-421
func ReducePoolsByDamage(poolTable *PoolTable, damageTable map[data.DamageType]float64, actor *Actor) *PoolTable {
	output := actor.Output
	modDB := actor.ModDB
	if poolTable == nil {
		poolTable = &PoolTable{}
	}

	damageTotal := float64(0)
	for _, dt := range data.DamageType("").Values() {
		d, ok := damageTable[dt]
		if !ok {
			continue
		}
		if d > 0 {
			damageTable[dt] = math.Ceil(d)
			damageTotal += damageTable[dt]
		} else {
			delete(damageTable, dt)
		}
	}

	alliesTakenBeforeYou := poolTable.AlliesTakenBeforeYou
	if alliesTakenBeforeYou == nil {
		alliesTakenBeforeYou = buildAllyLifePools(actor)
	}

	damageTakenThatCanBeRecouped := poolTable.DamageTakenThatCanBeRecouped
	if damageTakenThatCanBeRecouped == nil {
		damageTakenThatCanBeRecouped = make(map[data.DamageType]float64, 5)
	}

	aegis := poolTable.Aegis
	if aegis == nil {
		aegis = map[string]float64{
			"shared":          output["sharedAegis"],
			"sharedElemental": output["sharedElementalAegis"],
		}
		for dt := range damageTable {
			aegis[string(dt)] = output[string(dt)+"Aegis"]
		}
	}
	guard := poolTable.Guard
	if guard == nil {
		guard = map[string]float64{"shared": output["sharedGuardAbsorb"]}
		for dt := range damageTable {
			guard[string(dt)] = output[string(dt)+"GuardAbsorb"]
		}
	}

	ward := output["Ward"]
	if poolTable.Ward != nil {
		ward = *poolTable.Ward
	}
	wardActiveChance := float64(0)
	if poolTable.WardActiveChance != nil {
		wardActiveChance = *poolTable.WardActiveChance
	} else if ward > 0 {
		wardActiveChance = 1
	}
	wardAvoidBreakChance := math.Min(modDB.Sum(mod.TypeBase, nil, "WardAvoidBreakChance")/100, 1)
	if modDB.Flag(nil, "Condition:WardNotBreak") {
		wardAvoidBreakChance = 1
	}
	wardBypassBelow := modDB.Sum(mod.TypeBase, nil, "WardBypassBelowPercent") / 100

	energyShield := output["EnergyShieldRecoveryCap"]
	if poolTable.EnergyShield != nil {
		energyShield = *poolTable.EnergyShield
	}
	mana := output["ManaUnreserved"]
	if poolTable.Mana != nil {
		mana = *poolTable.Mana
	}
	life := output["LifeRecoverable"]
	if poolTable.Life != nil {
		life = *poolTable.Life
	}
	lifeLossBelowHalfPrevented := modDB.Sum(mod.TypeBase, nil, "LifeLossBelowHalfPrevented")
	lifeLossLostOverTime := poolTable.LifeLossLostOverTime
	lifeBelowHalfLossLostOverTime := poolTable.LifeBelowHalfLossLostOverTime
	overkillDamage := float64(0)

	ward = math.Max(ward, 0)
	wardBeforeHit := ward
	energyShield = math.Max(energyShield, 0)
	mana = math.Max(mana, 0)
	life = math.Max(life, 0)

	resourcesLost := make(map[data.DamageType]map[string]float64, 5)

	lifeHitPoolInitial := calcLifeHitPoolWithLossPrevention(life, output["Life"],
		output["preventedLifeLoss"], lifeLossBelowHalfPrevented)
	momPoolsRemaining := make(map[data.DamageType]float64, 5)
	esPoolsRemaining := make(map[data.DamageType]float64, 5)
	protectsMana := modDB.Flag(nil, "EnergyShieldProtectsMana")
	for dt := range damageTable {
		momEffect := math.Min(output["sharedMindOverMatter"]+output[string(dt)+"MindOverMatter"], 100) / 100
		maxMoM := math.Inf(1)
		momPool := mana
		if momEffect < 1 {
			maxMoM = lifeHitPoolInitial/(1-momEffect) - lifeHitPoolInitial
			momPool = math.Min(maxMoM, mana)
		}
		lifePlusMoMHitPool := lifeHitPoolInitial + momPool
		esBypass := output[string(dt)+"EnergyShieldBypass"] / 100
		esPool := energyShield
		if esBypass > 0 {
			esPool = math.Min(lifePlusMoMHitPool/esBypass-lifePlusMoMHitPool, energyShield)
		}
		if protectsMana && esBypass < 1 {
			esPoolsRemaining[dt] = math.Floor(math.Min(math.Min(maxMoM*(1-esBypass), energyShield),
				lifePlusMoMHitPool/(1-float64((1-esBypass)*momEffect))-lifePlusMoMHitPool))
			momPoolsRemaining[dt] = math.Floor(math.Min(maxMoM-esPoolsRemaining[dt], momPool))
		} else {
			esPoolsRemaining[dt] = math.Floor(esPool)
			momPoolsRemaining[dt] = math.Floor(momPool)
		}
	}

	//pob:block:src/Modules/CalcDefence.lua:263-394
	for _, dt := range data.DamageType("").Values() {
		damageRemainder, ok := damageTable[dt]
		if !ok {
			continue
		}

		for _, ally := range allyLifePoolList {
			av := alliesTakenBeforeYou[ally.key]
			if av == nil || av.Remaining <= 0 {
				continue
			}
			tempDamage := math.Min(damageRemainder*av.Percent, av.Remaining)
			av.Remaining = math.Floor(av.Remaining - tempDamage)
			damageRemainder -= tempDamage
			lostTo(resourcesLost, dt, ally.key, tempDamage)
		}
		damageTakenThatCanBeRecouped[dt] += damageRemainder

		if aegis[string(dt)] > 0 {
			tempDamage := math.Min(damageRemainder, aegis[string(dt)])
			aegis[string(dt)] -= tempDamage
			damageRemainder -= tempDamage
			lostTo(resourcesLost, dt, "aegis", tempDamage)
		}
		if dt.IsElemental() && aegis["sharedElemental"] > 0 {
			tempDamage := math.Min(damageRemainder, aegis["sharedElemental"])
			aegis["sharedElemental"] -= tempDamage
			damageRemainder -= tempDamage
			lostTo(resourcesLost, dt, "sharedElementalAegis", tempDamage)
		}
		if aegis["shared"] > 0 {
			tempDamage := math.Min(damageRemainder, aegis["shared"])
			aegis["shared"] -= tempDamage
			damageRemainder -= tempDamage
			lostTo(resourcesLost, dt, "sharedAegis", tempDamage)
		}
		if guard[string(dt)] > 0 {
			tempDamage := math.Min(damageRemainder*output[string(dt)+"GuardAbsorbRate"]/100, guard[string(dt)])
			guard[string(dt)] = math.Floor(guard[string(dt)] - tempDamage)
			damageRemainder -= tempDamage
			lostTo(resourcesLost, dt, "guard", tempDamage)
		}
		if guard["shared"] > 0 {
			tempDamage := math.Min(damageRemainder*output["sharedGuardAbsorbRate"]/100, guard["shared"])
			guard["shared"] = math.Floor(guard["shared"] - tempDamage)
			damageRemainder -= tempDamage
			lostTo(resourcesLost, dt, "sharedGuard", tempDamage)
		}
		if ward > 0 && (wardBypassBelow == 0 || damageTotal >= wardBeforeHit*wardBypassBelow) {
			tempDamage := math.Min(damageRemainder*(1-modDB.Sum(mod.TypeBase, nil, "WardBypass")/100), ward)
			ward -= tempDamage
			tempDamage = float64(tempDamage * wardActiveChance)
			damageRemainder -= tempDamage
			lostTo(resourcesLost, dt, "ward", tempDamage)
		}

		momEffect := math.Min(output["sharedMindOverMatter"]+output[string(dt)+"MindOverMatter"], 100) / 100
		esBypass := output[string(dt)+"EnergyShieldBypass"] / 100
		if energyShield > 0 && !protectsMana && esBypass < 1 {
			tempDamage := math.Min(damageRemainder*(1-esBypass), esPoolsRemaining[dt])
			for dt2 := range damageTable {
				esPoolsRemaining[dt2] = math.Max(esPoolsRemaining[dt2]-tempDamage, 0)
			}
			energyShield -= tempDamage
			damageRemainder -= tempDamage
			lostTo(resourcesLost, dt, "energyShield", tempDamage)
		}
		if momEffect > 0 {
			momDamage := math.Ceil(damageRemainder * momEffect)
			if protectsMana && energyShield > 0 && esBypass < 1 {
				tempDamage := math.Ceil(math.Min(momDamage*(1-esBypass), esPoolsRemaining[dt]))
				for dt2 := range damageTable {
					esPoolsRemaining[dt2] = math.Floor(math.Max(esPoolsRemaining[dt2]-tempDamage, 0))
				}
				energyShield -= tempDamage
				damageRemainder -= tempDamage
				momDamage -= tempDamage
				lostTo(resourcesLost, dt, "energyShield", tempDamage)
			}
			tempDamage := math.Ceil(math.Min(momDamage, momPoolsRemaining[dt]))
			for dt2 := range damageTable {
				momPoolsRemaining[dt2] = math.Floor(math.Max(momPoolsRemaining[dt2]-tempDamage, 0))
			}
			mana -= tempDamage
			damageRemainder -= tempDamage
			lostTo(resourcesLost, dt, "mana", tempDamage)
		}

		//pob:block:src/Modules/CalcDefence.lua:348-384
		if output["preventedLifeLossTotal"] > 0 {
			halfLife := float64(output["Life"] * 0.5)
			lifeOverHalfLife := math.Max(life-halfLife, 0)
			preventPercent := output["preventedLifeLoss"] / 100
			poolAboveLow := lifeOverHalfLife / (1 - preventPercent)
			preventBelowHalfPercent := lifeLossBelowHalfPrevented / 100
			damageThatLifeCanStillTake := poolAboveLow +
				math.Max(math.Min(life, halfLife), 0)/(1-preventBelowHalfPercent)/(1-preventPercent)
			if damageThatLifeCanStillTake < damageRemainder {
				overkillDamage += damageRemainder - damageThatLifeCanStillTake
				damageRemainder = damageThatLifeCanStillTake
			}
			if output["preventedLifeLossBelowHalf"] != 0 {
				damageToSplit := math.Min(damageRemainder, poolAboveLow)
				lostLife := float64(damageToSplit * (1 - preventPercent))
				preventedLoss := float64(damageToSplit * preventPercent)
				damageRemainder -= damageToSplit
				lifeLossLostOverTime += preventedLoss
				life -= lostLife
				lostSet(resourcesLost, dt, "life", lostLife)
				if life <= halfLife {
					unspecific := float64(damageRemainder * preventPercent)
					lifeLossLostOverTime += unspecific
					damageRemainder -= unspecific
					specific := float64(damageRemainder * preventBelowHalfPercent)
					lifeBelowHalfLossLostOverTime += specific
					damageRemainder -= specific
					preventedLoss = preventedLoss + unspecific + specific
				}
				lostTo(resourcesLost, dt, "lifeLossPrevented", preventedLoss)
			} else {
				tempDamage := damageRemainder * output["preventedLifeLoss"] / 100
				lifeLossLostOverTime += tempDamage
				damageRemainder -= tempDamage
				lostTo(resourcesLost, dt, "lifeLossPrevented", tempDamage)
			}
		}
		if life > 0 {
			tempDamage := math.Min(damageRemainder, life)
			life -= tempDamage
			damageRemainder -= tempDamage
			lostSet(resourcesLost, dt, "life", resourcesLost[dt]["life"]+math.Max(tempDamage, 0))
		}
		overkillDamage += damageRemainder
		lostTo(resourcesLost, dt, "overkill", damageRemainder)
	}

	momPoolRemaining := math.Inf(1)
	esPoolRemaining := math.Inf(1)
	for dt := range damageTable {
		momPoolRemaining = math.Min(momPoolRemaining, momPoolsRemaining[dt])
		esPoolRemaining = math.Min(esPoolRemaining, esPoolsRemaining[dt])
	}
	hitPoolRemaining := float64(0)
	if life >= 1 {
		hitPoolRemaining = calcLifeHitPoolWithLossPrevention(life, output["Life"],
			output["preventedLifeLoss"], lifeLossBelowHalfPrevented)
	}
	if !math.IsInf(momPoolRemaining, 1) {
		hitPoolRemaining += momPoolRemaining
	}
	if !math.IsInf(esPoolRemaining, 1) {
		hitPoolRemaining += esPoolRemaining
	}

	outWard := ward
	outWardActiveChance := wardActiveChance
	if ward < wardBeforeHit {
		outWard = 0
		if wardAvoidBreakChance > 0 {
			outWard = wardBeforeHit
		}
		outWardActiveChance = wardActiveChance * wardAvoidBreakChance
	}

	return &PoolTable{
		AlliesTakenBeforeYou:          alliesTakenBeforeYou,
		Aegis:                         aegis,
		Guard:                         guard,
		Ward:                          ptrF(math.Floor(outWard)),
		WardActiveChance:              ptrF(outWardActiveChance),
		EnergyShield:                  ptrF(math.Floor(energyShield)),
		Mana:                          ptrF(math.Floor(mana)),
		Life:                          ptrF(math.Floor(life)),
		DamageTakenThatCanBeRecouped:  damageTakenThatCanBeRecouped,
		LifeLossLostOverTime:          math.Ceil(lifeLossLostOverTime),
		LifeBelowHalfLossLostOverTime: math.Ceil(lifeBelowHalfLossLostOverTime),
		OverkillDamage:                math.Ceil(overkillDamage),
		HitPoolRemaining:              math.Floor(hitPoolRemaining),
		ResourcesLost:                 resourcesLost,
	}
}

func ptrF(v float64) *float64 { return &v }

//pob:src/Data/Global.lua:70-73
const (
	colorLife = colorStrength
	colorMana = colorIntelligence
	colorES   = colorSource
	colorWard = "^xFFFF77"
)

//pob:src/Modules/CalcDefence.lua:428-475
func incomingDamageBreakdown(breakdownTable []string, poolsRemaining *PoolTable, output map[string]float64) []string {
	for _, ally := range allyLifePoolList {
		if pool := poolsRemaining.AlliesTakenBeforeYou[ally.key]; pool != nil {
			life := output[ally.life]
			breakdownTable = append(breakdownTable, fmt.Sprintf("\t%d "+colorGem+ally.label+" ^7(%d remaining)", luaInt(life-pool.Remaining), luaInt(pool.Remaining)))
		}
	}
	for _, damageType := range data.DamageType("").Values() {
		if lost, ok := poolsRemaining.ResourcesLost[damageType]["aegis"]; ok {
			breakdownTable = append(breakdownTable, fmt.Sprintf("\t%d "+colorGem+"%s Aegis charge ^7(%d remaining)", luaInt(lost), damageType, luaInt(poolsRemaining.Aegis[string(damageType)])))
		}
	}
	if output["sharedElementalAegis"] != 0 && output["sharedElementalAegis"] != poolsRemaining.Aegis["sharedElemental"] {
		breakdownTable = append(breakdownTable, fmt.Sprintf("\t%d "+colorGem+"Elemental Aegis charge ^7(%d remaining)", luaInt(output["sharedElementalAegis"]-poolsRemaining.Aegis["sharedElemental"]), luaInt(poolsRemaining.Aegis["sharedElemental"])))
	}
	if output["sharedAegis"] > 0 {
		breakdownTable = append(breakdownTable, fmt.Sprintf("\t%d "+colorGem+"Shared Aegis charge ^7(%d remaining)", luaInt(output["sharedAegis"]-poolsRemaining.Aegis["shared"]), luaInt(poolsRemaining.Aegis["shared"])))
	}
	for _, damageType := range data.DamageType("").Values() {
		if lost, ok := poolsRemaining.ResourcesLost[damageType]["guard"]; ok {
			breakdownTable = append(breakdownTable, fmt.Sprintf("\n\t%d "+colorScourge+"%s Guard charge ^7(%d remaining)", luaInt(lost), damageType, luaInt(poolsRemaining.Guard[string(damageType)])))
		}
	}
	if output["sharedGuardAbsorb"] > 0 {
		breakdownTable = append(breakdownTable, fmt.Sprintf("\t%d "+colorScourge+"Shared Guard charge ^7(%d remaining)", luaInt(output["sharedGuardAbsorb"]-poolsRemaining.Guard["shared"]), luaInt(poolsRemaining.Guard["shared"])))
	}
	if output["Ward"] > 0 {
		breakdownTable = append(breakdownTable, fmt.Sprintf("\t%d "+colorWard+"Ward", luaInt(output["Ward"])))
	}
	if output["EnergyShieldRecoveryCap"] != *poolsRemaining.EnergyShield && output["EnergyShieldRecoveryCap"] > 0 {
		breakdownTable = append(breakdownTable, fmt.Sprintf("\t%d "+colorES+"Energy Shield ^7(%d remaining)", luaInt(output["EnergyShieldRecoveryCap"]-*poolsRemaining.EnergyShield), luaInt(*poolsRemaining.EnergyShield)))
	}
	if output["ManaUnreserved"] != *poolsRemaining.Mana && output["ManaUnreserved"] > 0 {
		breakdownTable = append(breakdownTable, fmt.Sprintf("\t%d "+colorMana+"Mana ^7(%d remaining)", luaInt(output["ManaUnreserved"]-*poolsRemaining.Mana), luaInt(*poolsRemaining.Mana)))
	}
	if poolsRemaining.LifeLossLostOverTime+poolsRemaining.LifeBelowHalfLossLostOverTime > 0 {
		breakdownTable = append(breakdownTable, fmt.Sprintf("\t%d "+colorLife+"Life ^7Loss Prevented", luaInt(poolsRemaining.LifeLossLostOverTime+poolsRemaining.LifeBelowHalfLossLostOverTime)))
	}
	breakdownTable = append(breakdownTable, fmt.Sprintf("\t%d "+colorLife+"Life ^7(%d remaining)", luaInt(output["LifeRecoverable"]-*poolsRemaining.Life), luaInt(*poolsRemaining.Life)))
	if poolsRemaining.OverkillDamage >= 1 {
		breakdownTable = append(breakdownTable, fmt.Sprintf("\t%d "+colorNegative+"Overkill damage", luaInt(poolsRemaining.OverkillDamage)))
	}
	return breakdownTable
}

//pob:block:src/Modules/CalcDefence.lua:149-174
func TakenHitFromDamage(rawDamage float64, damageType data.DamageType, actor *Actor) (float64, map[data.DamageType]float64) {
	output := actor.Output
	modDB := actor.ModDB

	damageMitigationMultiplierForType := func(damage float64, ofType data.DamageType) float64 {
		totalResistMult := output[string(ofType+"ResistTakenHitMulti")]
		effectiveAppliedArmour := output[string(ofType+"EffectiveAppliedArmour")]
		armourDRPercent := CalcArmourReductionF(effectiveAppliedArmour, damage*totalResistMult)
		flatDRPercent := utils.Ternary(modDB.Flag(nil, string("SelfIgnoreBase"+ofType+"DamageReduction")), 0, output[string("Base"+ofType+"DamageReductionWhenHit")])
		totalDRPercent := math.Min(output["DamageReductionMax"], armourDRPercent+flatDRPercent)
		enemyOverwhelmPercent := utils.Ternary(modDB.Flag(nil, string("SelfIgnore"+ofType+"DamageReduction")), 0, output[string(ofType+"EnemyOverwhelm")])
		totalDRMulti := 1 - math.Max(math.Min(output["DamageReductionMax"], totalDRPercent-enemyOverwhelmPercent), 0)/100
		return totalResistMult * totalDRMulti
	}

	receivedDamageSum := float64(0)
	damages := make(map[data.DamageType]float64, 5)
	shift := actor.DamageShiftTable[damageType]
	for _, damageConvertedType := range data.DamageType("").Values() {
		convertPercent := shift[damageConvertedType]
		takenFlat := output[string(damageConvertedType)+"takenFlat"]
		if convertPercent <= 0 && takenFlat == 0 {
			continue
		}
		convertedDamage := rawDamage * convertPercent / 100
		vaalArctic := math.Min(-modDB.Sum(mod.TypeMore, nil, "VaalArcticArmourMitigation")/100, 1)
		reducedDamage := float64(math.Floor(float64(math.Max(float64(convertedDamage*damageMitigationMultiplierForType(convertedDamage, damageConvertedType))+takenFlat, 0)*output[string(damageConvertedType+"AfterReductionTakenHitMulti")])+0.5) * (1 - vaalArctic))
		receivedDamageSum += reducedDamage
		if reducedDamage > 0 || convertPercent > 0 {
			damages[damageConvertedType] = reducedDamage
		}
	}
	return receivedDamageSum, damages
}
