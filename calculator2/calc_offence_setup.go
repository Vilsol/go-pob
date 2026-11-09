package calculator2

import (
	"math"
	"strconv"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/poe2skills"
	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// offenceStatConversions applies the "X applies to Y" stat conversions. The Law of the Wilds claw
// conversions to minions (CalcOffence.lua:1031-1048) are absent: no PoE2 mod sets their flags.
//
//pob2:block:src/Modules/CalcOffence.lua:874-1106
func offenceStatConversions(env *Environment, actor *Actor, activeSkill *ActiveSkill, skillFlags map[string]bool, skillCfg *moddb.ListCfg) {
	skillModList := activeSkill.SkillModList
	modDB := actor.ModDB
	output := actor.Output
	addCopy := func(name string, modType mod.Type, value float64, m mod.Mod, flags mod.MFlag, keywordFlags mod.KeywordFlag) {
		skillModList.AddMod(copyMod(name, modType, value, m.GetSource(), flags, keywordFlags, m.Tags()))
	}
	if skillModList.Flag(nil, "MinionDamageAppliesToPlayer") || skillModList.Flag(skillCfg, "MinionDamageAppliesToPlayer") {
		// Minion Damage conversion from Spiritual Aid and The Scourge
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedMinionDamageAppliesToPlayer") / 100
		for _, value := range skillModList.List(skillCfg, "MinionModifier") {
			m := minionModifierMod(value)
			if m != nil && m.Name() == "Damage" && m.Type() == mod.TypeIncrease {
				skillModList.AddMod(copyMod("Damage", mod.TypeIncrease, m.Value().Float()*multiplier, m.GetSource(), m.Flags(), m.KeywordFlags(), getConvertedModTagsMinion(m, multiplier)))
			}
		}
	}
	if skillModList.Flag(nil, "CompanionDamageAppliesToPlayer") {
		// Companion Damage conversion from Inspiring Ally
		// copyTable(skillCfg, true) is shallow: the tag lands in the skill's own skillTypes and stays there.
		tempCfg := *skillCfg
		tempCfg.SkillTypes["CreatesCompanion"] = true // Add companion skill tag to cfg so it doesn't fail
		activeSkill.SkillTypes["CreatesCompanion"] = true
		for _, value := range skillModList.Tabulate(mod.TypeList, &tempCfg, "MinionModifier") {
			m := minionModifierMod(value.Mod.Value().List())
			if modHasSkillType(value.Mod, "CreatesCompanion") && m != nil && m.Name() == "Damage" && m.Type() == mod.TypeIncrease {
				addCopy("Damage", mod.TypeIncrease, m.Value().Float(), m, m.Flags(), m.KeywordFlags())
			}
		}
	}
	if skillModList.Flag(nil, "MinionAttackSpeedAppliesToPlayer") {
		// Minion Damage conversion from Spiritual Command
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedMinionAttackSpeedAppliesToPlayer") / 100
		// Minion Attack Speed conversion from Spiritual Command
		for _, value := range skillModList.List(skillCfg, "MinionModifier") {
			m := minionModifierMod(value)
			if m != nil && m.Name() == "Speed" && m.Type() == mod.TypeIncrease && (m.Flags() == 0 || m.Flags()&ModFlagAttack != 0) {
				skillModList.AddMod(copyMod("Speed", mod.TypeIncrease, m.Value().Float()*multiplier, m.GetSource(), ModFlagAttack, m.KeywordFlags(), getConvertedModTagsMinion(m, multiplier)))
			}
		}
	}
	if skillModList.Flag(nil, "SpellDamageAppliesToAttacks") {
		// Spell Damage conversion from Crown of Eyes, Kinetic Bolt, and the Wandslinger notable
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedSpellDamageAppliesToAttacks") / 100
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, flagsCfg(ModFlagSpell), "Damage") {
			m := value.Mod
			if m.Flags()&ModFlagSpell != 0 {
				skillModList.AddMod(copyMod("Damage", mod.TypeIncrease, m.Value().Float()*multiplier, m.GetSource(), (m.Flags()&^ModFlagSpell)|ModFlagAttack, m.KeywordFlags(), getConvertedModTags(m, multiplier)))
				if m.GetSource() == "Strength" { // Prevent double-dipping from converted strength's damage bonus
					skillModList.ReplaceMod(newMod("PhysicalDamage", mod.TypeIncrease, 0, "Strength").Flag(ModFlagMelee))
					skillModList.ReplaceMod(newMod("PhysicalDamage", mod.TypeIncrease, 0, "Strength").Flag(ModFlagAttack | ModFlagProjectile))
				}
			}
		}
	}

	// Apply thorns-derived modifiers to hits
	if skillModList.Flag(nil, "ThornsDamageApplyToAttackDamage") {
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, flagsCfg(ModFlagThorns), "Damage") {
			m := value.Mod
			addCopy("Damage", mod.TypeIncrease, m.Value().Float(), m, (m.Flags()&^ModFlagThorns)|ModFlagAttack, m.KeywordFlags())
		}
	}

	if skillModList.Flag(nil, "CastSpeedAppliesToAttacks") {
		// Get all increases for this; assumption is that multiple sources would not stack, so find the max
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedCastSpeedAppliesToAttacks") / 100
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, flagsCfg(ModFlagCast), "Speed") {
			m := value.Mod
			// Add a new mod for all mods that are cast only
			// Replace this with a single mod for the sum?
			if m.Flags()&ModFlagCast != 0 {
				skillModList.AddMod(copyMod("Speed", mod.TypeIncrease, m.Value().Float()*multiplier, m.GetSource(), (m.Flags()&^ModFlagCast)|ModFlagAttack, m.KeywordFlags(), getConvertedModTags(m, multiplier)))
			}
		}
	}
	if skillModList.Flag(nil, "CastSpeedAppliesToProjectileSpeed") {
		// Bow mastery projectile speed to damage with bows conversion
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedCastSpeedAppliesToProjectileSpeed") / 100
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, flagsCfg(ModFlagCast), "Speed") {
			m := value.Mod
			skillModList.AddMod(copyMod("ProjectileSpeed", m.Type(), m.Value().Float()*multiplier, m.GetSource(), m.Flags()&^ModFlagCast, m.KeywordFlags(), getConvertedModTags(m, multiplier)))
		}
	}
	if skillModList.Flag(nil, "ProjectileSpeedAppliesToBowDamage") {
		// Bow mastery projectile speed to damage with bows conversion
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, flagsCfg(ModFlagBow), "ProjectileSpeed") {
			m := value.Mod
			addCopy("Damage", m.Type(), m.Value().Float(), m, ModFlagBow|ModFlagHit, m.KeywordFlags())
		}
	}
	if skillModList.Flag(nil, "WarcryDamageAppliesToSkill") {
		// Fortifying Cry mod
		for _, modType := range []mod.Type{mod.TypeIncrease, mod.TypeMore} {
			for _, value := range skillModList.Tabulate(modType, keywordFlagsCfg(KeywordFlagWarcry), "Damage") {
				m := value.Mod
				if m.KeywordFlags()&KeywordFlagWarcry != 0 {
					addCopy("Damage", m.Type(), m.Value().Float(), m, m.Flags(), m.KeywordFlags()&^KeywordFlagWarcry)
				}
			}
		}
	}
	if skillModList.Flag(nil, "PinBuildupInsteadOfHeavyStunBuildup") {
		// Fortifying Cry mod
		for _, modType := range []mod.Type{mod.TypeIncrease, mod.TypeMore} {
			for _, value := range skillModList.Tabulate(modType, &moddb.ListCfg{}, "EnemyHeavyStunBuildup") {
				m := value.Mod
				addCopy("EnemyPinBuildup", m.Type(), m.Value().Float(), m, m.Flags(), m.KeywordFlags())
			}
		}
	}
	if skillModList.Flag(nil, "FreezeBuildupInsteadOfStunBuildup") {
		// Kaltenhalt
		for _, modType := range []mod.Type{mod.TypeIncrease, mod.TypeMore} {
			for _, value := range skillModList.Tabulate(modType, &moddb.ListCfg{}, "EnemyHeavyStunBuildup") {
				m := value.Mod
				addCopy("EnemyFreezeBuildup", m.Type(), m.Value().Float(), m, m.Flags(), m.KeywordFlags())
			}
		}
	}
	if skillModList.Flag(nil, "ProjectileSpeedAppliesToProjectileDamage") {
		// Projectile speed to projectile damage conversion
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, &moddb.ListCfg{}, "ProjectileSpeed") {
			m := value.Mod
			addCopy("Damage", m.Type(), m.Value().Float(), m, ModFlagProjectile, m.KeywordFlags())
		}
	}
	if skillModList.Flag(nil, "ClawDamageAppliesToUnarmed") {
		// Claw Damage conversion from Rigwald's Curse
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, flagsKeywordCfg(ModFlagClaw|ModFlagHit, KeywordFlagHit), "Damage") {
			m := value.Mod
			if m.Flags()&ModFlagClaw != 0 {
				addCopy("Damage", m.Type(), m.Value().Float(), m, (m.Flags()&^ModFlagClaw)|ModFlagUnarmed|ModFlagMelee, m.KeywordFlags())
			}
		}
	}
	if skillModList.Flag(nil, "ClawAttackSpeedAppliesToUnarmed") {
		// Claw Attack Speed conversion from Rigwald's Curse
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, flagsCfg(ModFlagClaw|ModFlagAttack|ModFlagHit), "Speed") {
			m := value.Mod
			if m.Flags()&ModFlagClaw != 0 && m.Flags()&ModFlagAttack != 0 {
				addCopy("Speed", m.Type(), m.Value().Float(), m, (m.Flags()&^ModFlagClaw)|ModFlagUnarmed, m.KeywordFlags())
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritChanceAppliesToUnarmed") {
		// Claw Crit Chance conversion from Rigwald's Curse
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, flagsCfg(ModFlagClaw|ModFlagHit), "CritChance") {
			m := value.Mod
			if m.Flags()&ModFlagClaw != 0 {
				addCopy("CritChance", m.Type(), m.Value().Float(), m, (m.Flags()&^ModFlagClaw)|ModFlagUnarmed, m.KeywordFlags())
			}
		}
	}
	for _, crit := range []struct{ flag, outputKey string }{
		{"CritChanceIncreasedByUncappedLightningRes", "LightningResistTotal"},
		{"CritChanceIncreasedByLightningRes", "LightningResist"},
		{"CritChanceIncreasedByOvercappedLightningRes", "LightningResistOverCap"},
		{"CritChanceIncreasedBySpellSuppressChance", "SpellSuppressionChance"},
	} {
		if skillModList.Flag(nil, crit.flag) {
			for _, value := range modDB.Tabulate(mod.TypeFlag, nil, crit.flag) {
				skillModList.AddMod(newMod("CritChance", mod.TypeIncrease, output[crit.outputKey], value.Mod.GetSource()))
				break
			}
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToAccuracy") {
		// Light Radius conversion from Corona Solaris
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, &moddb.ListCfg{}, "LightRadius") {
			m := value.Mod
			addCopy("Accuracy", mod.TypeIncrease, m.Value().Float(), m, m.Flags(), m.KeywordFlags())
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToAreaOfEffect") {
		// Light Radius conversion from Wreath of Phrecia
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, &moddb.ListCfg{}, "LightRadius") {
			m := value.Mod
			addCopy("AreaOfEffect", mod.TypeIncrease, math.Floor(m.Value().Float()/2), m, m.Flags(), m.KeywordFlags())
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToDamage") {
		// Light Radius conversion from Wreath of Phrecia
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, &moddb.ListCfg{}, "LightRadius") {
			m := value.Mod
			addCopy("Damage", mod.TypeIncrease, m.Value().Float(), m, m.Flags(), m.KeywordFlags())
		}
	}
	if skillModList.Flag(nil, "CastSpeedAppliesToTrapThrowingSpeed") {
		// Cast Speed conversion from Slavedriver's Hand
		for _, value := range skillModList.Tabulate(mod.TypeIncrease, flagsCfg(ModFlagCast), "Speed") {
			m := value.Mod
			if m.Flags() == 0 || m.Flags()&ModFlagCast != 0 {
				addCopy("TrapThrowingSpeed", mod.TypeIncrease, m.Value().Float(), m, m.Flags()&^ModFlagCast&^ModFlagAttack, m.KeywordFlags())
			}
		}
	}
}

//pob2:block:src/Modules/CalcOffence.lua:708-724
func modHasSkillType(m mod.Mod, skillType string) bool {
	for _, tag := range m.Tags() {
		if st, ok := tag.(*mod.SkillTypeTag); ok {
			for _, listed := range st.SkillTypeList {
				if listed == skillType {
					return true
				}
			}
		}
	}
	return false
}

//pob2:block:src/Modules/CalcOffence.lua:1129-1233
func offenceRepeats(env *Environment, activeSkill *ActiveSkill, skillFlags map[string]bool, output map[string]float64) {
	skillModList := activeSkill.SkillModList
	skillCfg := activeSkill.SkillCfg
	repeatSkillTypesCheck := func(activeSkillTypes map[string]bool) bool {
		for _, typ := range []string{"Instant", "Channel", "Triggered", "Retaliation", "NonRepeatable"} {
			if activeSkillTypes[typ] {
				return false
			}
		}
		return !skillModList.Flag(nil, "CannotRepeat") && (activeSkillTypes["Attack"] || activeSkillTypes["Spell"])
	}
	output["Repeats"] = 1
	if repeatSkillTypesCheck(activeSkill.SkillTypes) {
		output["Repeats"] = 1 + skillModList.Sum(mod.TypeBase, skillCfg, "RepeatCount")
	}
	if output["Repeats"] <= 1 {
		return
	}
	repeats := output["Repeats"]
	output["RepeatCount"] = repeats
	// handle all the multipliers from Repeats
	repeatMode, _ := configListValue(env, "repeatMode")
	if repeatMode == "NONE" {
		return
	}
	average := repeatMode == "AVERAGE"
	add := func(name string, modType mod.Type, value float64, m mod.Mod) {
		skillModList.AddMod(copyMod(name, modType, value, m.GetSource(), m.Flags(), m.KeywordFlags(), m.Tags()))
	}
	for _, value := range skillModList.Tabulate(mod.TypeIncrease, skillCfg, "RepeatAreaOfEffect") {
		modValue := value.Mod.Value().Float()
		if average {
			modValue = modValue * (repeats - 1) / repeats
		}
		add("AreaOfEffect", mod.TypeIncrease, modValue, value.Mod)
	}
	for _, value := range skillModList.Tabulate(mod.TypeIncrease, skillCfg, "RepeatFinalAreaOfEffect") {
		modValue := value.Mod.Value().Float()
		if average {
			modValue /= repeats
		}
		add("AreaOfEffect", mod.TypeIncrease, modValue, value.Mod)
	}
	for _, value := range skillModList.Tabulate(mod.TypeIncrease, skillCfg, "RepeatPerRepeatAreaOfEffect") {
		modValue := value.Mod.Value().Float() * (repeats - 1)
		if average {
			modValue /= 2
		}
		add("AreaOfEffect", mod.TypeIncrease, modValue, value.Mod)
	}
	for _, value := range skillModList.Tabulate(mod.TypeBase, skillCfg, "RepeatFinalDoubleDamageChance") {
		modValue := value.Mod.Value().Float()
		if average {
			modValue /= repeats
		}
		add("DoubleDamageChance", mod.TypeBase, modValue, value.Mod)
	}
	damageFinalMoreValueTotal := 1.0
	damageMoreValueTotal := 0.0
	for _, value := range skillModList.Tabulate(mod.TypeMore, skillCfg, "RepeatFinalDamage") {
		modValue := value.Mod.Value().Float()
		damageFinalMoreValueTotal *= 1 + modValue/100
		damageMoreValueTotal += modValue
		if average && !skillModList.Flag(nil, "OnlyFinalRepeat") {
			modValue /= repeats
		}
		add("Damage", mod.TypeMore, modValue, value.Mod)
	}
	for _, value := range skillModList.Tabulate(mod.TypeMore, skillCfg, "RepeatPerRepeatDamage") {
		v := value.Mod.Value().Float()
		modValue := v * (repeats - 1)
		if average {
			if damageFinalMoreValueTotal != 1 {
				// sum from 0 to num Repeats the damage each one does, multiplied by the other repeat multipliers,
				// divide the total by the average other repeat multipliers and divide by number of repeats
				// eg greater echo with 20Q div echo is (100 + 130 + 160 + 190*1.6)/1.15/4 - 100 = 50.87% more damage
				modValue = (float64((100+v*(repeats-2)/2)*(repeats-1))+float64((100+float64(v*(repeats-1)))*damageFinalMoreValueTotal))/(repeats+damageMoreValueTotal/100) - 100
			} else {
				modValue /= 2
			}
		}
		add("Damage", mod.TypeMore, modValue, value.Mod)
	}

	var lastMod mod.Mod
	damageFinalMoreValueTotal = damageMoreValueTotal
	for _, repeatCount := range []struct {
		n    float64
		name string
	}{{2, "One"}, {3, "Two"}, {4, "Three"}} {
		if repeatCount.n > repeats {
			break
		} else if average {
			for _, value := range skillModList.Tabulate(mod.TypeMore, skillCfg, "Repeat"+repeatCount.name+"Damage") {
				damageMoreValueTotal += value.Mod.Value().Float()
				lastMod = value.Mod
			}
		} else if repeatCount.n == repeats {
			for _, value := range skillModList.Tabulate(mod.TypeMore, skillCfg, "Repeat"+repeatCount.name+"Damage") {
				add("Damage", mod.TypeMore, value.Mod.Value().Float(), value.Mod)
			}
		}
	}
	if average && lastMod != nil {
		add("Damage", mod.TypeMore, (damageMoreValueTotal/repeats+100)/(1+damageFinalMoreValueTotal/repeats/100)-100, lastMod)
	}
	if skillModList.Flag(nil, "FinalRepeatSumsDamage") {
		for _, value := range skillModList.Tabulate(mod.TypeFlag, skillCfg, "FinalRepeatSumsDamage") {
			add("Damage", mod.TypeMore, (float64(100*repeats)+damageFinalMoreValueTotal)/(1+damageFinalMoreValueTotal/100)-100, value.Mod)
		}
	}
	if skillFlags["trap"] || skillFlags["mine"] {
		skillModList.AddMod(newMod("DPS", mod.TypeMore, (repeats-1)*100, "Repeat Count"))
	}
}

//pob2:block:src/Modules/CalcOffence.lua:1234-1271
func offenceWeaponSpellDamage(env *Environment, actor *Actor, activeSkill *ActiveSkill) {
	skillModList := activeSkill.SkillModList
	skillData := activeSkill.SkillData
	weaponData1 := weaponData(env, actor, 1)
	weaponData2 := weaponData(env, actor, 2)
	for _, gain := range []struct{ key, weaponType, source string }{
		{"gainPercentBaseWandDamageToSpells", "Wand", "Spellslinger"},
		{"gainPercentBaseDaggerDamageToSpells", "Dagger", "Blade Blast of Dagger Detonation"},
	} {
		percent, ok := sdNum(skillData, gain.key)
		if !ok {
			continue
		}
		mult := percent / 100
		source := mod.Source(gain.source)
		addBound := func(bound string, value float64) {
			skillModList.AddMod(newMod(bound, mod.TypeBase, math.Floor(value), source).Flag(ModFlagSpell))
		}
		switch {
		case actor.WeaponData1Type == gain.weaponType && actor.WeaponData2Type == gain.weaponType:
			for _, damageType := range dmgTypeList {
				addBound(damageType+"Min", (weaponData1[damageType+"Min"]+weaponData2[damageType+"Min"])/2*mult)
				addBound(damageType+"Max", (weaponData1[damageType+"Max"]+weaponData2[damageType+"Max"])/2*mult)
			}
		case actor.WeaponData1Type == gain.weaponType:
			for _, damageType := range dmgTypeList {
				addBound(damageType+"Min", weaponData1[damageType+"Min"]*mult)
				addBound(damageType+"Max", weaponData1[damageType+"Max"]*mult)
			}
		case actor.WeaponData2Type == gain.weaponType:
			for _, damageType := range dmgTypeList {
				addBound(damageType+"Min", weaponData2[damageType+"Min"]*mult)
				addBound(damageType+"Max", weaponData2[damageType+"Max"]*mult)
			}
		}
	}
}

//pob2:block:src/Modules/CalcOffence.lua:1337-1397
func offenceRandomElement(env *Environment, activeSkill *ActiveSkill, skillFlags map[string]bool) {
	skillModList := activeSkill.SkillModList
	skillCfg := activeSkill.SkillCfg
	if skillModList.Sum(mod.TypeBase, skillCfg, "PhysicalDamageGainAsRandom", "PhysicalDamageConvertToRandom", "PhysicalDamageGainAsColdOrLightning", "DamageGainAsRandom") <= 0 {
		return
	}
	skillFlags["randomPhys"] = true
	physMode, ok := configListValue(env, "physMode")
	if !ok {
		physMode = "AVERAGE"
	}
	add := func(name string, value float64, m mod.Mod) {
		skillModList.AddMod(copyMod(name, mod.TypeBase, value, m.GetSource(), m.Flags(), m.KeywordFlags(), m.Tags()))
	}
	for _, conv := range []struct{ from, prefix string }{
		{"DamageGainAsRandom", "DamageGainAs"},
		{"PhysicalDamageGainAsRandom", "PhysicalDamageGainAs"},
		{"PhysicalDamageConvertToRandom", "PhysicalDamageConvertTo"},
	} {
		for _, value := range skillModList.Tabulate(mod.TypeBase, skillCfg, conv.from) {
			m := value.Mod
			v := m.Value().Float()
			switch physMode {
			case "AVERAGE":
				effVal := v / 3
				add(conv.prefix+"Fire", effVal, m)
				add(conv.prefix+"Cold", effVal, m)
				add(conv.prefix+"Lightning", effVal, m)
			case "FIRE":
				add(conv.prefix+"Fire", v, m)
			case "COLD":
				add(conv.prefix+"Cold", v, m)
			case "LIGHTNING":
				add(conv.prefix+"Lightning", v, m)
			}
		}
	}
	for _, value := range skillModList.Tabulate(mod.TypeBase, skillCfg, "PhysicalDamageGainAsColdOrLightning") {
		m := value.Mod
		v := m.Value().Float()
		switch physMode {
		case "AVERAGE", "FIRE":
			effVal := v / 2
			add("PhysicalDamageGainAsCold", effVal, m)
			add("PhysicalDamageGainAsLightning", effVal, m)
		case "COLD":
			add("PhysicalDamageGainAsCold", v, m)
		case "LIGHTNING":
			add("PhysicalDamageGainAsLightning", v, m)
		}
	}
}

// preDamageSkill is the activeSkill a grantedEffect preDamageFunc reads and writes.
type preDamageSkill struct {
	activeSkill *ActiveSkill
}

var _ poe2skills.PreDamageSkill = preDamageSkill{}

func (s preDamageSkill) SkillData(key string) (float64, bool) {
	return sdNum(s.activeSkill.SkillData, key)
}

func (s preDamageSkill) SetSkillData(key string, v float64, ok bool) {
	if ok {
		s.activeSkill.SkillData[key] = v
	} else {
		delete(s.activeSkill.SkillData, key)
	}
}

func (s preDamageSkill) Sum(modType, name string) float64 {
	return s.activeSkill.SkillModList.Sum(mod.Type(modType), s.activeSkill.SkillCfg, name)
}

func (s preDamageSkill) NewMod(name, modType string, value any, source string) {
	switch v := value.(type) {
	case bool:
		s.activeSkill.SkillModList.AddMod(mod.NewFlag(name, v).Source(mod.Source(source)))
	case float64:
		s.activeSkill.SkillModList.AddMod(newMod(name, mod.Type(modType), v, mod.Source(source)))
	}
}

func (s preDamageSkill) StatSetIndex() int {
	return s.activeSkill.ActiveEffect.StatSet.Index
}

func explodeModValue(v any) (ExplodeMod, bool) {
	switch x := v.(type) {
	case ExplodeMod:
		return x, true
	case *ExplodeMod:
		return *x, true
	}
	return ExplodeMod{}, false
}

func (s preDamageSkill) ExplodeMods() []poe2skills.ExplodeMod {
	var out []poe2skills.ExplodeMod
	for _, v := range s.activeSkill.SkillModList.List(s.activeSkill.SkillCfg, "ExplodeMod") {
		if e, ok := explodeModValue(v); ok {
			out = append(out, poe2skills.ExplodeMod{Type: e.Type, Amount: e.Amount, Value: e.Value})
		}
	}
	return out
}

func (s preDamageSkill) TabulateExplodeMods() []poe2skills.TabulatedExplodeMod {
	var out []poe2skills.TabulatedExplodeMod
	for _, r := range s.activeSkill.SkillModList.Tabulate(mod.TypeList, s.activeSkill.SkillCfg, "ExplodeMod") {
		if e, ok := explodeModValue(r.Mod.Value().List()); ok {
			out = append(out, poe2skills.TabulatedExplodeMod{Source: string(r.Mod.GetSource()), Value: poe2skills.ExplodeMod{Type: e.Type, Amount: e.Amount, Value: e.Value}})
		}
	}
	return out
}

// ExplodeSource is srcInstance.explodeSource: an item's modSource, or a tree node's id.
func (s preDamageSkill) ExplodeSource() (string, bool, string) {
	switch src := s.activeSkill.ActiveEffect.SrcInstance.ExplodeSource.(type) {
	case *Item:
		return src.modSourceName, true, ""
	case *tree2.SpecNode:
		return "", false, strconv.Itoa(src.ID)
	}
	return "", false, ""
}

type preDamageOutput map[string]float64

func (o preDamageOutput) Get(key string) (float64, bool) {
	v, ok := o[key]
	return v, ok
}

func (o preDamageOutput) Set(key string, v float64) {
	o[key] = v
}

// offencePreDamage runs the grantedEffect's preDamageFunc, then handles corpse and enemy explosions.
// A preDamageFunc's Lua error has no counterpart here; the skill's state stays as the error left it.
//
//pob2:block:src/Modules/CalcOffence.lua:2407-2423
func offencePreDamage(env *Environment, activeSkill *ActiveSkill, skillFlags map[string]bool, output map[string]float64) float64 {
	if fn := activeSkill.ActiveEffect.GrantedEffect.PreDamageFunc(); fn != nil {
		_ = fn(preDamageSkill{activeSkill: activeSkill}, preDamageOutput(output))
	}

	skillData := activeSkill.SkillData
	monsterLife, hasCorpseLife := sdNum(skillData, "corpseLife")
	if !hasCorpseLife {
		monsterLife = 100
		if env.EnemyLevel != 0 {
			monsterLife = monsterLifeTable(env.EnemyLevel)
		}
	}
	if truthy(skillData["explodeCorpse"]) && (hasCorpseLife || env.EnemyLevel != 0) {
		damageType := "Physical"
		if v, ok := skillData["corpseExplosionDamageType"].(string); ok {
			damageType = v
		}
		lifeMultiplier, ok := sdNum(skillData, "corpseExplosionLifeMultiplier")
		if !ok {
			lifeMultiplier, _ = sdNum(skillData, "selfFireExplosionLifeMultiplier")
		}
		skillData[damageType+"BonusMin"] = monsterLife * lifeMultiplier
		skillData[damageType+"BonusMax"] = monsterLife * lifeMultiplier
	}
	if skillFlags["monsterExplode"] {
		for _, damageType := range dmgTypeList {
			percentage, _ := sdNum(skillData, damageType+"EffectiveExplodePercentage")
			base := percentage * monsterLife / 100
			skillData[damageType+"Min"] = base
			skillData[damageType+"Max"] = base
		}
	}
	return monsterLife
}

// monsterLifeTable is data.monsterLifeTable[level] or 100.
func monsterLifeTable(level int) float64 {
	if level >= 1 && level <= len(poe2.MonsterLifeTable) {
		return poe2.MonsterLifeTable[level-1]
	}
	return 100
}
