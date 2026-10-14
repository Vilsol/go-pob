package calculator

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/calculator/calclib"
	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

//pob:src/Modules/CalcTriggers.lua:1592-1617

type triggerConfig struct {
	skipEffectiveRate bool
	ignoresTickRate   bool
	overlaps          *float64
	source            *ActiveSkill

	actor *Actor

	triggerSkillCond func(skill *ActiveSkill) bool

	triggerOnUse bool

	sourceWeapon bool

	triggerName       string
	customTriggerName string

	triggeredSkillCond func(skill *ActiveSkill) bool

	comparer triggerComparer

	triggerChance *float64

	trigRate *float64

	ignoreSourceRate bool

	customHandler func(env *Environment)

	useCastRate           bool
	assumingEveryHitKills bool
	sourceRateIsFinal     bool
	sourceName            string
}

type triggerComparer func(cached *cachedTriggerSkill, source *ActiveSkill, triggerRate float64) bool

func grantedEffectFromItem(skill *ActiveSkill) bool {
	if skill == nil || skill.ActiveEffect == nil || skill.ActiveEffect.GrantedEffect == nil {
		return false
	}
	return raw2.PoBSkills[skill.ActiveEffect.GrantedEffect.ID()].FromItem
}

//pob:block:src/Modules/CalcSetup.lua:1392
func srcInstanceFromItem(skill *ActiveSkill) bool {
	if skill == nil {
		return false
	}
	socketGroup, ok := skill.SocketGroup.(*pob.Skill)
	return ok && socketGroup != nil && strings.HasPrefix(socketGroup.Source, "Item:")
}

//pob:src/Modules/CalcTriggers.lua:32-38
func slotMatch(env *Environment, skill *ActiveSkill) bool {
	if env == nil || env.Player == nil || env.Player.MainSkill == nil || skill == nil {
		return false
	}
	mainSkill := env.Player.MainSkill

	mainFromItem := grantedEffectFromItem(mainSkill)
	fromItem := mainFromItem || grantedEffectFromItem(skill) ||
		srcInstanceFromItem(mainSkill) || srcInstanceFromItem(skill)

	if fromItem && skill.SocketGroup != nil && mainSkill.SocketGroup != nil &&
		socketGroupSlot(skill) == socketGroupSlot(mainSkill) {
		return true
	}

	//pob:src/Modules/CalcTriggers.lua:36
	return !mainFromItem && skill.SocketGroup == mainSkill.SocketGroup
}

//pob:src/Modules/CalcTriggers.lua:897-1574
func triggerConfigFor(env *Environment, key string, mainSkill *ActiveSkill) *triggerConfig {
	switch key {
	//pob:src/Modules/CalcTriggers.lua:1290-1293
	case "tempest shield":
		return &triggerConfig{source: mainSkill}

	//pob:src/Modules/CalcTriggers.lua:1175-1189
	case "cast when damage taken":
		if mainSkill.SkillData.TriggeredByDamageTaken == 0 {
			return nil
		}
		return &triggerConfig{source: mainSkill}

	//pob:src/Modules/CalcTriggers.lua:1190-1194
	case "cast when stunned":
		return &triggerConfig{source: mainSkill}

	//pob:src/Modules/CalcTriggers.lua:1195-1199
	case "cast on ward break":
		return &triggerConfig{source: mainSkill}

	//pob:src/Modules/CalcTriggers.lua:1231-1246
	case "automation":
		config := triggerCooldownFromActivePart(env, mainSkill, "Automation", false)
		config.useCastRate = true
		return config

	//pob:src/Modules/CalcTriggers.lua:1247-1268
	case "autoexertion":
		config := triggerCooldownFromActivePart(env, mainSkill, "Autoexertion", true)
		config.useCastRate, config.sourceRateIsFinal = true, true
		return config

	//pob:src/Modules/CalcTriggers.lua:1151-1154
	case "cast on critical strike":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeAttack] && slotMatch(env, skill)
			},
			triggeredSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillData != nil && skill.SkillData.TriggeredByCoC &&
					slotMatch(env, skill)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1321-1341
	case "arcanist brand":
		return arcanistBrandConfig(env, mainSkill)

	//pob:src/Modules/CalcTriggers.lua:1483-1497
	case "intuitive link":
		return intuitiveLinkConfig(env, mainSkill)

	//pob:src/Modules/CalcTriggers.lua:1498-1511
	case "svalinn cast on block", "festering resentment cast on block":
		return &triggerConfig{
			source: mainSkill,
			triggeredSkillCond: func(skill *ActiveSkill) bool {
				return slotMatch(env, skill) && skill.TriggeredBy != nil &&
					CalcCanGrantedEffectSupportActiveSkill(skill.TriggeredBy.GrantedEffect, skill)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1167-1174
	case "nova":
		if env == nil || env.Minion == nil || env.Minion.Actor == nil || env.Minion.MainSkill == nil {
			return nil
		}
		return &triggerConfig{
			actor:       env.Minion.Actor,
			triggerName: "Summon Holy Relic",
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeAttack]
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1272-1288

	//pob:src/Modules/CalcTriggers.lua:1275-1277
	case "mark on hit", "oskarm":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeAttack]
			},
			sourceRateIsFinal: key == "oskarm",
		}

	//pob:src/Modules/CalcTriggers.lua:1278-1283
	case "hextouch", "prismatic burst":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeAttack] && slotMatch(env, skill)
			},
			sourceRateIsFinal: key == "hextouch",
		}

	//pob:src/Modules/CalcTriggers.lua:1361-1369
	case "shockwave", "void shockwave", "falling crystal":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeMelee] && slotMatch(env, skill)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1040-1045
	case "vixen's entrapment":
		return &triggerConfig{
			useCastRate: true,
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeHex]
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1109-1116
	case "mjolner":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				if !skill.SkillTypes[data.SkillTypeDamage] && !skill.SkillTypes[data.SkillTypeAttack] {
					return false
				}
				if skill.SkillCfg == nil || skill.SkillCfg.Flags == nil ||
					*skill.SkillCfg.Flags&(mod.MFlagMace|mod.MFlagWeapon1H) == 0 {
					return false
				}
				return !slotMatch(env, skill)
			},
			triggeredSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillData != nil && skill.SkillData.TriggeredByMjolner &&
					slotMatch(env, skill)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1131-1134
	case "seven teachings":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeMelee] && skill.SkillCfg != nil && skill.SkillCfg.Flags != nil &&
					*skill.SkillCfg.Flags&(mod.MFlagUnarmed|mod.MFlagMelee) == mod.MFlagUnarmed|mod.MFlagMelee
			},
			triggeredSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillData != nil && skill.SkillData.TriggeredBySevenTeachings &&
					slotMatch(env, skill)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1269-1271
	case "seize the flesh":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeWarcry] && slotMatch(env, skill)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1272-1274
	case "fissure":
		return &triggerConfig{
			triggerOnUse: true,
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeSlam] && slotMatch(env, skill)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1530-1538
	case "ghostly artillery":
		return &triggerConfig{
			sourceWeapon: true,
			triggerOnUse: true,
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeAttack]
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1358-1360
	case "voidstorm":
		return &triggerConfig{
			triggerOnUse: true,
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeAttack] &&
					skill.SkillTypes[data.SkillTypeRain] && slotMatch(env, skill)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1378-1382
	case "manaforged arrows":
		return &triggerConfig{
			triggerOnUse: true,
			triggerName:  "Manaforged Arrows",
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeAttack] &&
					skill.SkillCfg != nil && skill.SkillCfg.Flags != nil &&
					*skill.SkillCfg.Flags&mod.MFlagBow > 0
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1475-1482
	case "avenging flame":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillFlags[SkillFlagTotem] && slotMatch(env, skill)
			},
			comparer: func(cached *cachedTriggerSkill, source *ActiveSkill, currentTotemLife float64) bool {
				return (source == nil && cached.totemLifeSet) ||
					(cached.totemLifeSet && cached.TotemLife > currentTotemLife)
			},
			ignoreSourceRate: true,
		}

	//pob:src/Modules/CalcTriggers.lua:1310-1320
	case "battlemage's cry":
		if mainSkill.ActiveEffect.GrantedEffect.Name() == "Battlemage's Cry" {
			return nil
		}
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeMelee]
			},
			comparer: func(cached *cachedTriggerSkill, source *ActiveSkill, triggerRate float64) bool {
				//pob:src/Modules/CalcTriggers.lua:1316
				return defaultTriggerComparer(cached, source, triggerRate) &&
					cached.battlemageUpTimeRatioSet
			},
			triggeredSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillData != nil && skill.SkillData.TriggeredByBattleMageCry &&
					slotMatch(env, skill)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1347-1354
	case "combust":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeMelee]
			},
			comparer: func(cached *cachedTriggerSkill, source *ActiveSkill, triggerRate float64) bool {
				// Skills with no uptime ratio are not exerted by infernal cry so should not be considered.
				return defaultTriggerComparer(cached, source, triggerRate) && cached.infernalUpTimeRatioSet
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1200-1212
	case "spellslinger":
		return &triggerConfig{
			skipEffectiveRate: mainSkill.ActiveEffect.GrantedEffect.Name() == "Spellslinger",
			sourceRateIsFinal: true,
			triggerOnUse:      true,
			triggerName:       "Spellslinger",
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeAttack] &&
					skill.SkillTypes[data.SkillTypeProjectile] &&
					skillCfgHasFlag(skill, mod.MFlagWand)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1383-1392
	case "doom blast":
		if triggerConfigInputString(env, "doomBlastSource") == "replacement" {
			env.Player.ModDB.AddMod(mod.NewFlag("UsesCurseOverlaps", true).Source("Config"))
		}
		var overlaps *float64
		if len(env.Player.ModDB.Tabulate(mod.TypeBase, nil, "Multiplier:CurseOverlaps")) > 0 {
			overlaps = utils.Ptr(math.Max(env.Player.ModDB.Sum(mod.TypeBase, nil, "Multiplier:CurseOverlaps"), 1))
		}
		return &triggerConfig{
			ignoresTickRate:   true,
			useCastRate:       true,
			overlaps:          overlaps,
			customTriggerName: "Doom Blast triggering Hex: ",
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeHex] && slotMatch(env, skill)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:905-923
	case "the rippling thoughts", "the surging thoughts":
		if env.Player.MainSkill == nil || env.Player.MainSkill.ActiveEffect.GrantedEffect.Name() != "Storm Cascade" {
			return nil
		}
		return &triggerConfig{
			triggerOnUse: key == "the surging thoughts",
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeMelee] || skill.SkillTypes[data.SkillTypeAttack]
			},
		}

	//pob:src/Modules/CalcTriggers.lua:979-981
	case "starcaller":
		return &triggerConfig{
			triggerName: "Starfall",
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeMelee] || skill.SkillTypes[data.SkillTypeAttack]
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1370-1377
	case "call the pyre":
		if env.Enemy == nil || env.Enemy.ModDB == nil || !env.Enemy.ModDB.Flag(nil, "Condition:Ignited") {
			return nil
		}
		return &triggerConfig{
			triggerChance: utils.Ptr(float64(50)),
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeMelee] && slotMatch(env, skill)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:1147-1150
	case "kinetic flux":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeAttack] && slotMatch(env, skill)
			},
			triggeredSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillData != nil && skill.SkillData.TriggeredByKineticFlux &&
					slotMatch(env, skill)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:939-946
	case "replica eternity shroud", "shroud of the lightless":
		return &triggerConfig{source: mainSkill}

	//pob:src/Modules/CalcTriggers.lua:1393-1395
	case "cast while channelling":
		return &triggerConfig{customHandler: cwcHandler}

	//pob:src/Modules/CalcTriggers.lua:1396-1398
	case "focus":
		return &triggerConfig{customHandler: helmetFocusHandler}

	case "snipe":
		return snipeTriggerConfig(env, mainSkill)

	//pob:src/Modules/CalcTriggers.lua:1058-1080
	case "trigger craft":
		return triggerCraftConfig(env, mainSkill)

	//pob:src/Modules/CalcTriggers.lua:898-904
	case "law of the wilds":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return !skill.SkillTypes[data.SkillTypeSummonsTotem] &&
					(skill.SkillTypes[data.SkillTypeMelee] || skill.SkillTypes[data.SkillTypeAttack]) &&
					skillCfgHasFlag(skill, mod.MFlagClaw)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:956-967
	case "lioneye's paws", "replica lioneye's paws":
		mainSkill.SkillData.Cooldown = 1
		return &triggerConfig{
			triggerOnUse: true,
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeAttack] && skillCfgHasFlag(skill, mod.MFlagBow)
			},
		}

	//pob:src/Modules/CalcTriggers.lua:968-972
	case "moonbender's wing":
		mainSkill.SkillData.Cooldown = 1
		return &triggerConfig{
			triggerName: "Lightning Warp",
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeMelee] || skill.SkillTypes[data.SkillTypeAttack]
			},
		}

	//pob:src/Modules/CalcTriggers.lua:947-955,src/Modules/CalcTriggers.lua:973-1008
	case "limbsplit", "the cauteriser", "ngamahu's flame":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeMelee] || skill.SkillTypes[data.SkillTypeAttack]
			},
		}
	case "duskblight", "cameria's avarice", "uul-netol's embrace", "rigwald's crest",
		"jorrhast's blacksteel", "ashcaller", "arakaali's fang", "sporeguard", "mark of the elder",
		"mark of the shaper":
		//pob:block:src/Modules/CalcTriggers.lua:985-1008
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return skill.SkillTypes[data.SkillTypeDamage] || skill.SkillTypes[data.SkillTypeAttack]
			},
			triggerName:           map[string]string{"duskblight": "Stalking Pustule", "cameria's avarice": "Icicle Burst", "uul-netol's embrace": "Bone Nova"}[key],
			assumingEveryHitKills: key != "duskblight" && key != "cameria's avarice" && key != "uul-netol's embrace",
			sourceRateIsFinal:     key == "rigwald's crest" || key == "jorrhast's blacksteel" || key == "ashcaller",
		}

	//pob:src/Modules/CalcTriggers.lua:1539-1545
	case "replica gifts from above":
		return &triggerConfig{
			triggerSkillCond: func(skill *ActiveSkill) bool {
				return !skill.SkillTypes[data.SkillTypeSummonsTotem] &&
					skill.SkillTypes[data.SkillTypeAttack]
			},
		}
	}

	return nil
}

//pob:block:src/Modules/CalcTriggers.lua:1231-1246
func triggerCooldownFromActivePart(env *Environment, mainSkill *ActiveSkill, name string, ignoresTickRate bool) *triggerConfig {
	if mainSkill.ActiveEffect.GrantedEffect.Name() == name {
		return &triggerConfig{skipEffectiveRate: true, ignoresTickRate: ignoresTickRate, source: mainSkill}
	}

	if env != nil && env.Player != nil && mainSkill.TriggeredBy != nil {
		for _, skill := range env.Player.ActiveSkillList {
			if skill.ActiveEffect == nil || skill.ActiveEffect.GrantedEffect == nil {
				continue
			}
			if skill.ActiveEffect.GrantedEffect.Name() == name {
				mainSkill.TriggeredBy.GrantedEffect = skill.ActiveEffect.GrantedEffect
				break
			}
		}
	}

	return &triggerConfig{ignoresTickRate: ignoresTickRate, source: mainSkill}
}

//pob:block:src/Modules/CalcTriggers.lua:1602-1606
func triggerConfigForSkill(env *Environment, mainSkill *ActiveSkill) *triggerConfig {
	grantedEffect := mainSkill.ActiveEffect.GrantedEffect

	if config := triggerConfigFor(env, grantedEffect.ID(), mainSkill); config != nil {
		return config
	}
	if config := triggerConfigFor(env, strings.ToLower(grantedEffect.Name()), mainSkill); config != nil {
		return config
	}

	if mainSkill.TriggeredBy == nil || mainSkill.TriggeredBy.GrantedEffect == nil {
		return nil
	}

	triggerName := strings.ToLower(mainSkill.TriggeredBy.GrantedEffect.Name())
	if config := triggerConfigFor(env, triggerName, mainSkill); config != nil {
		return config
	}
	return triggerConfigFor(env, strings.TrimPrefix(triggerName, "awakened "), mainSkill)
}

//pob:block:src/Modules/CalcTriggers.lua:1596
func uniqueTriggerRung(env *Environment, mainSkill *ActiveSkill) *triggerConfig {
	if !isTriggered(mainSkill) {
		return nil
	}
	name := uniqueItemTriggerName(mainSkill)
	if name == "" {
		return nil
	}
	return triggerConfigFor(env, strings.ToLower(name), mainSkill)
}

//pob:src/Modules/CalcTriggers.lua:1575-1590
func uniqueItemTriggerName(skill *ActiveSkill) string {
	for _, gemInstance := range skill.SupportList {
		if gemInstance == nil || gemInstance.GrantedEffect == nil || gemInstance.GrantedEffect.Raw == nil {
			continue
		}
		if raw2.PoBSkills[gemInstance.GrantedEffect.ID()].FromItem && !gemInstance.GrantedEffect.Raw.IsSupport {
			return gemInstance.GrantedEffect.Name()
		}
	}
	if socketGroup, ok := skill.SocketGroup.(*pob.Skill); ok && socketGroup != nil && socketGroup.Source != "" {
		if m := uniqueTriggerSourceRe.FindStringSubmatch(socketGroup.Source); m != nil {
			return m[1]
		}
	}
	return ""
}

var uniqueTriggerSourceRe = regexp.MustCompile(`^.*:.*:(.*),.*$`)

//pob:src/Modules/CalcTriggers.lua:1321-1341
func arcanistBrandConfig(env *Environment, mainSkill *ActiveSkill) *triggerConfig {
	if mainSkill.ActiveEffect == nil || mainSkill.ActiveEffect.GrantedEffect == nil ||
		mainSkill.ActiveEffect.GrantedEffect.Name() == "Arcanist Brand" {
		return nil
	}
	if env == nil || env.Player == nil || env.Player.MainSkill == nil {
		return nil
	}
	triggeredBy := env.Player.MainSkill.TriggeredBy
	if triggeredBy == nil {
		return nil
	}

	//pob:src/Modules/CalcTriggers.lua:1324-1329
	for _, skill := range env.Player.ActiveSkillList {
		if skill.ActiveEffect != nil && skill.ActiveEffect.GrantedEffect != nil &&
			skill.ActiveEffect.GrantedEffect.Name() == "Arcanist Brand" {
			triggeredBy.MainSkill = skill
			break
		}
	}
	brand := triggeredBy.MainSkill
	if brand == nil || brand.SkillData == nil || brand.SkillModList == nil {
		return nil
	}

	//pob:src/Modules/CalcTriggers.lua:1331-1336
	activationFreqInc := (100 + brand.SkillModList.Sum(mod.TypeIncrease, brand.SkillCfg,
		"Speed", "BrandActivationFrequency")) / 100
	activationFreqMore := brand.SkillModList.More(brand.SkillCfg, "BrandActivationFrequency")
	triggeredBy.ActivationFreqInc = activationFreqInc
	triggeredBy.ActivationFreqMore = activationFreqMore
	triggeredBy.AttachedBrandCount = brand.SkillData.AttachedBrandCount
	triggeredBy.IgnoresTickRate = true

	//pob:src/Modules/CalcTriggers.lua:1337-1339
	trigRate := brand.SkillData.RepeatFrequency * activationFreqInc * activationFreqMore *
		triggeredBy.AttachedBrandCount
	return &triggerConfig{
		trigRate:          utils.Ptr(trigRate),
		source:            brand,
		sourceRateIsFinal: true,
		triggeredSkillCond: func(skill *ActiveSkill) bool {
			return skill.SkillData != nil && skill.SkillData.TriggeredByBrand && slotMatch(env, skill)
		},
	}
}

//pob:src/Modules/CalcTriggers.lua:1058-1080
func triggerCraftConfig(env *Environment, mainSkill *ActiveSkill) *triggerConfig {
	if !mainSkill.SkillData.TriggeredByCraft {
		return nil
	}
	var source *ActiveSkill
	var trigRate float64
	useCastRate := false
	for _, skill := range env.Player.ActiveSkillList {
		if (skill.SkillTypes[data.SkillTypeDamage] || skill.SkillTypes[data.SkillTypeAttack] || skill.SkillTypes[data.SkillTypeSpell]) &&
			!skill.SkillFlags[SkillFlag("aura")] && skill != mainSkill && !skill.SkillData.TriggeredByCraft &&
			!grantedEffectFromItem(skill) && !isTriggered(skill) {
			source, trigRate = findTriggerSkill(env, skill, source, trigRate, nil, nil)
			//pob:src/Modules/CalcTriggers.lua:1065-1072
			if skill.SkillFlags[SkillFlagTotem] || skill.SkillFlags[SkillFlag("golem")] ||
				skill.SkillFlags[SkillFlag("banner")] || skill.SkillFlags[SkillFlagBallista] {
				grantedEffect := skill.ActiveEffect.GrantedEffect
				if grantedEffect != nil && grantedEffect.Raw != nil {
					cooldown := float64(0)
					if level, ok := raw2.GetCalculatedGrantedEffect(grantedEffect.Raw).GetCalculatedLevels()[skill.ActiveEffect.Level]; ok && level != nil && level.Cooldown != nil {
						cooldown = *level.Cooldown
					}
					trigRate = 1 / (grantedEffect.CastTime() + cooldown)
				}
				useCastRate = true
			}
		}
	}
	if source == nil {
		mainSkill.SkillData.Triggered = false
		return nil
	}
	return &triggerConfig{source: source, trigRate: utils.Ptr(trigRate), useCastRate: useCastRate}
}

//pob:src/Modules/CalcTriggers.lua:135-217
func helmetFocusHandler(env *Environment) {
	mainSkill := env.Player.MainSkill
	if mainSkill == nil || mainSkill.SkillFlags[SkillFlagMinion] || mainSkill.SkillFlags[SkillFlagDisable] ||
		mainSkill.TriggeredBy == nil {
		return
	}
	skillFocus := poe.GrantedEffectByID("Focus")
	if skillFocus == nil {
		return
	}
	calculatedFocus := raw2.GetCalculatedGrantedEffect(skillFocus)
	focusLevel, ok := calculatedFocus.GetCalculatedLevels()[1]
	if !ok || focusLevel == nil || focusLevel.Cooldown == nil {
		return
	}
	mainSkill.SkillData.Triggered = true
	output := env.Player.Output

	triggerCD := triggeredByCooldown(mainSkill)
	triggeredCD := mainSkill.SkillData.Cooldown

	icdrFocus := calclib.Mod(mainSkill.SkillModList, mainSkill.SkillCfg, "FocusCooldownRecovery")
	icdrSkill := calclib.Mod(mainSkill.SkillModList, mainSkill.SkillCfg, "CooldownRecovery")

	focusDuration := calculatedFocus.GetCalculatedConstantStats()["base_skill_effect_duration"] / 1000
	focusCD := *focusLevel.Cooldown / icdrFocus
	focusTotalCD := focusDuration + focusCD

	//pob:src/Modules/CalcTriggers.lua:156-163
	modActionCooldown := math.Max(triggeredCD, triggerCD/icdrSkill)
	rateCapAdjusted := math.Ceil(modActionCooldown*data.ServerTickRate) / data.ServerTickRate
	if rateCapAdjusted != 0 {
		output["TriggerRateCap"] = 1 / rateCapAdjusted
	}
	//pob:src/Modules/CalcTriggers.lua:164
	output["SkillTriggerRate"] = 1 / focusTotalCD

	//pob:src/Modules/CalcTriggers.lua:211-215
	addTriggerIncMoreMods(mainSkill, mainSkill)
	mainSkill.infoMessage = "Assuming perfect focus Re-Use"
	mainSkill.infoTrigger = "Focus"
	mainSkill.SkillData.TriggerRate = utils.Ptr(output["SkillTriggerRate"])
}

//pob:src/Modules/CalcTriggers.lua:219-387
func cwcHandler(env *Environment) {
	mainSkill := env.Player.MainSkill
	if mainSkill.SkillFlags[SkillFlagMinion] || mainSkill.SkillFlags[SkillFlagDisable] {
		return
	}
	var triggeredSkills []*triggerRotationSkill
	var source *ActiveSkill
	triggerName := "Cast While Channeling"
	output := env.Player.Output
	for _, skill := range env.Player.ActiveSkillList {
		slotMatch := slotMatch(env, skill)
		if source == nil {
			canSupport := false
			if mainSkill.TriggeredBy != nil && mainSkill.TriggeredBy.GemData != nil {
				effect := mainSkill.TriggeredBy.GemData.GetGrantedEffect()
				canSupport = CalcCanGrantedEffectSupportActiveSkill(&GrantedEffect{Raw: effect, fromItem: env.itemSupports[effect.ID]}, skill)
			}
			if skill.SkillData.TriggerTime != 0 && canSupport && skill != mainSkill && slotMatch && !isTriggered(skill) {
				source = skill
			}
		}
		if skill.SkillData.TriggeredWhileChannelling && slotMatch {
			triggeredSkills = append(triggeredSkills, packageSkillDataForSimulation(env, skill))
		}
	}
	if source == nil || len(triggeredSkills) < 1 {
		mainSkill.SkillData.Triggered = false
		mainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
		mainSkill.infoMessage = fmt.Sprintf("No %s Triggering Skill Found", triggerName)
		mainSkill.infoTrigger = ""
		return
	}

	if addsCastTime := processAddedCastTime(mainSkill); addsCastTime != 0 {
		output["addsCastTime"] = addsCastTime
	}

	icdr := calclib.Mod(mainSkill.SkillModList, mainSkill.SkillCfg, "CooldownRecovery")
	triggerInterval := source.SkillData.TriggerTime
	triggerRateOfTrigger := 1 / triggerInterval
	triggeredCD := mainSkill.SkillData.Cooldown
	cooldownOverride := mainSkill.SkillModList.Override(mainSkill.SkillCfg, "CooldownRecovery")

	triggeredTotalCooldown := math.Max(triggeredCD, output["addsCastTime"]) / icdr
	if cooldownOverride != nil {
		triggeredTotalCooldown = utils.Or(cooldownOverride, 0)
	}
	triggeredCDAdjusted := math.Ceil(triggeredTotalCooldown*data.ServerTickRate) / data.ServerTickRate
	effCDTriggeredSkill := math.Ceil(triggeredCDAdjusted*triggerRateOfTrigger) / triggerRateOfTrigger

	output["TriggerRateCap"] = math.Min(1/effCDTriggeredSkill, triggerRateOfTrigger)
	output["SkillTriggerRate"], _ = calcMultiSpellRotationImpact329(env, triggeredSkills, triggerRateOfTrigger, 0, 0, nil)
	if env.Player.ModDB.Flag(nil, "HaveTriggerBots") && mainSkill.SkillTypes[data.SkillTypeSpell] {
		output["SkillTriggerRate"] = 2 * output["SkillTriggerRate"]
	}

	// Account for Trigger-related INC/MORE modifiers
	addTriggerIncMoreMods(mainSkill, mainSkill)
	output["ChannelTimeToTrigger"] = triggerInterval
	mainSkill.SkillData.Triggered = true
	mainSkill.SkillData.TriggerRate = utils.Ptr(output["SkillTriggerRate"])
	mainSkill.infoMessage = triggerName + "'s Trigger: " + source.ActiveEffect.GrantedEffect.Name()
}

//pob:src/Modules/CalcTriggers.lua:1483-1497
func intuitiveLinkConfig(env *Environment, mainSkill *ActiveSkill) *triggerConfig {
	if mainSkill.ActiveEffect == nil || mainSkill.ActiveEffect.GrantedEffect == nil ||
		mainSkill.ActiveEffect.GrantedEffect.Name() == "Intuitive Link" {
		return nil
	}
	if env == nil || env.Player == nil || env.Player.MainSkill == nil || env.Player.MainSkill.TriggeredBy == nil {
		return nil
	}
	triggeredBy := env.Player.MainSkill.TriggeredBy
	//pob:src/Modules/CalcTriggers.lua:1485-1490
	for _, skill := range env.Player.ActiveSkillList {
		if skill.ActiveEffect != nil && skill.ActiveEffect.GrantedEffect != nil &&
			skill.ActiveEffect.GrantedEffect.Name() == "Intuitive Link" {
			triggeredBy.MainSkill = skill
			break
		}
	}
	link := triggeredBy.MainSkill
	return &triggerConfig{
		triggeredSkillCond: func(skill *ActiveSkill) bool {
			return skill.SkillTypes[data.SkillTypeSpell] && slotMatch(env, skill) && skill != link
		},
		trigRate:    utils.Ptr(env.Player.ModDB.Sum(mod.TypeBase, nil, "IntuitiveLinkSourceRate")),
		source:      link,
		sourceName:  "Custom source",
		useCastRate: true,
	}
}

//pob:block:src/Modules/CalcTriggers.lua:562
func triggeredByCooldown(mainSkill *ActiveSkill) float64 {
	triggeredBy := mainSkill.TriggeredBy
	if triggeredBy == nil || triggeredBy.GrantedEffect == nil || triggeredBy.GrantedEffect.Raw == nil {
		return 0
	}

	levels := raw2.GetCalculatedGrantedEffect(triggeredBy.GrantedEffect.Raw).GetCalculatedLevels()
	level, ok := levels[triggeredBy.Level]
	if !ok || level == nil || level.Cooldown == nil {
		return 0
	}

	return *level.Cooldown
}

//pob:src/Modules/CalcTriggers.lua:44-61
func processAddedCastTime(skill *ActiveSkill) float64 {
	if !skill.SkillModList.Flag(skill.SkillCfg, "SpellCastTimeAddedToCooldownIfTriggered") {
		return 0
	}

	baseCastTime := skill.SkillData.CastTimeOverride
	if baseCastTime == 0 {
		baseCastTime = skill.ActiveEffect.GrantedEffect.CastTime()
	}
	if baseCastTime == 0 {
		baseCastTime = 1
	}

	inc := skill.SkillModList.Sum(mod.TypeIncrease, skill.SkillCfg, "Speed")
	more := skill.SkillModList.More(skill.SkillCfg, "Speed")
	csi := math.Round((1+inc/100)*more*100) / 100
	if csi == 0 {
		return 0
	}

	return baseCastTime / csi
}

//pob:src/Modules/CalcTriggers.lua:40-42
func isTriggered(skill *ActiveSkill) bool {
	if skill == nil {
		return false
	}
	if skill.SkillData != nil && (skill.SkillData.TriggeredByUnique || skill.SkillData.Triggered) {
		return true
	}
	return skill.SkillTypes[data.SkillTypeInbuiltTrigger] || skill.SkillTypes[data.SkillTypeTriggered] ||
		(skill.ActiveEffect != nil && skill.ActiveEffect.CostEmptied)
}

func triggerActionCooldown(mainSkill *ActiveSkill, castTimeSkill *ActiveSkill, configIgnoresTickRate bool) (float64, float64) {
	cd := triggerCooldowns(mainSkill, castTimeSkill, configIgnoresTickRate)
	return cd.actionCooldownTickRounded, cd.addsCastTime
}

//pob:block:src/Modules/CalcTriggers.lua:571,src/Modules/CalcTriggers.lua:616-698
func triggerRateCapBreakdown(env *Environment, actor *Actor, mainSkill *ActiveSkill, breakdown *Breakdown, cd triggerCooldownParts, triggerName string, triggerRateCap float64) {
	isMinion := env.Minion != nil && actor == env.Minion.Actor
	triggeredName := "Triggered skill"
	if mainSkill.ActiveEffect.GrantedEffect != nil && !isMinion {
		triggeredName = mainSkill.ActiveEffect.GrantedEffect.Name()
	}
	key := "TriggerRateCap"
	breakdown.Data[key] = &Entry{}
	add := func(lines ...string) { breakdown.AddLine(key, lines...) }

	switch {
	case cd.cooldownOverride != nil:
		add(fmt.Sprintf("%.2f ^8(hard override of cooldown of %s)", *cd.cooldownOverride, triggeredName))
	case cd.triggeredCDAdjusted == 0:
		add(triggeredName + " has no base cooldown or cooldown override")
	default:
		if cd.triggeredCD != 0 {
			add(fmt.Sprintf("%.2f ^8(base cooldown of triggered skill)", cd.triggeredCD))
		} else {
			add(triggeredName + " has no base cooldown or cooldown override")
		}
		if cd.addedCooldown != 0 {
			add(fmt.Sprintf("+ %.2f ^8(flat added cooldown)", cd.addedCooldown))
		}
		add(fmt.Sprintf("/ %.2f ^8(increased/reduced cooldown recovery)", cd.icdr),
			fmt.Sprintf("= %.4f ^8(final cooldown of triggered skill)", cd.triggeredCDAdjusted))
	}

	add("")

	if !isMinion {
		if cd.triggerCDAdjusted == 0 {
			add(fmt.Sprintf("Trigger rate based on %s cooldown", triggeredName))
		} else {
			if cd.triggerCD != 0 {
				add(fmt.Sprintf("%.2f ^8(base cooldown of %s)", cd.triggerCD, triggerName))
			} else {
				add(triggerName + " has no base cooldown")
			}
			if cd.addsCastTime != 0 {
				add(fmt.Sprintf("+ %.2f ^8(this skill adds cast time to cooldown when triggered)", cd.addsCastTime))
			}
			add(fmt.Sprintf("/ %.2f ^8(increased/reduced cooldown recovery)", cd.icdr),
				fmt.Sprintf("= %.4f ^8(final cooldown of trigger)", cd.triggerCDAdjusted))
		}
	}

	add("")

	if cd.triggeredCDAdjusted != 0 && cd.triggerCDAdjusted != 0 {
		add(fmt.Sprintf("%.3f ^8(biggest of trigger cooldown and triggered skill cooldown)", cd.actionCooldownAdjusted))
	}

	displayCooldownRounding := (!cd.ignoresTickRate && cd.triggeredCDAdjusted != 0) ||
		(mainSkill.TriggeredBy != nil && !mainSkill.TriggeredBy.IgnoresTickRate && cd.triggerCDAdjusted != 0)
	if displayCooldownRounding {
		add(fmt.Sprintf("%.3f ^8(adjusted for server tick rate)", cd.actionCooldownTickRounded))
	}

	extraIncreaseNeeded := func(affectedCD float64) (float64, bool) {
		if cd.cooldownOverride == nil {
			nextBreakpoint := cd.actionCooldownTickRounded - data.ServerTickTime
			timeOverBreakpoint := cd.actionCooldownAdjusted - nextBreakpoint
			alreadyReducedTime := cd.actionCooldown - cd.actionCooldownAdjusted
			if timeOverBreakpoint < affectedCD {
				divNeeded := affectedCD / (affectedCD - timeOverBreakpoint - alreadyReducedTime)
				incTotal := math.Ceil((divNeeded - 1) * 100)
				return incTotal - float64((cd.icdr-1)*100), true
			}
		}
		return 0, false
	}

	if extraICDRNeeded, ok := extraIncreaseNeeded(cd.actionCooldown); ok {
		add(fmt.Sprintf("^8(extra ICDR of %d%% would reach next breakpoint)", luaInt(extraICDRNeeded)))
	}
	if cd.addsCastTime != 0 {
		if extraCSIncNeeded, ok := extraIncreaseNeeded(cd.addsCastTime); ok {
			add(fmt.Sprintf("^8(extra ICS  of %d%% would reach next breakpoint)", luaInt(extraCSIncNeeded)))
		}
	}

	add("")

	if cd.triggeredCD == 0 && cd.triggerCD == 0 && cd.cooldownOverride == nil {
		add("Assuming cast on every kill/attack/hit")
	} else {
		add("Trigger rate:",
			fmt.Sprintf("1 / %.3f", cd.actionCooldownTickRounded),
			fmt.Sprintf("= %.2f ^8per second", triggerRateCap))
	}
}

type triggerCooldownParts struct {
	icdr, addedCooldown, triggerCD, triggeredCD, addsCastTime         float64
	cooldownOverride                                                  *float64
	triggeredCDAdjusted, triggerCDAdjusted                            float64
	actionCooldown, actionCooldownAdjusted, actionCooldownTickRounded float64
	ignoresTickRate                                                   bool
}

//pob:block:src/Modules/CalcTriggers.lua:558-581
func triggerCooldowns(mainSkill *ActiveSkill, castTimeSkill *ActiveSkill, configIgnoresTickRate bool) triggerCooldownParts {
	icdr := calclib.Mod(mainSkill.SkillModList, mainSkill.SkillCfg, "CooldownRecovery")
	if icdr == 0 {
		icdr = 1
	}
	addedCooldown := mainSkill.SkillModList.Sum(mod.TypeBase, mainSkill.SkillCfg, "CooldownRecovery")
	cooldownOverride := mainSkill.SkillModList.Override(mainSkill.SkillCfg, "CooldownRecovery")

	triggerCD := triggeredByCooldown(mainSkill)
	triggeredCD := mainSkill.SkillData.Cooldown

	//pob:src/Modules/CalcTriggers.lua:566-569
	if mainSkill.SkillData.TriggeredByBrand && mainSkill.TriggeredBy != nil &&
		mainSkill.TriggeredBy.MainSkill != nil && mainSkill.TriggeredBy.MainSkill.SkillData != nil {
		triggeredBy := mainSkill.TriggeredBy
		triggerCD = triggeredBy.MainSkill.SkillData.RepeatFrequency /
			triggeredBy.ActivationFreqMore / triggeredBy.ActivationFreqInc
		triggerCD *= icdr
	}

	addsCastTime := processAddedCastTime(castTimeSkill)

	ignoresTickRate := configIgnoresTickRate ||
		(mainSkill.SkillData.StoredUses != nil && *mainSkill.SkillData.StoredUses > 1)

	triggeredCDAdjusted := (triggeredCD + addedCooldown) / icdr
	triggerCDAdjusted := (triggerCD + addsCastTime) / icdr

	triggeredCDTickRounded := triggeredCDAdjusted
	if !ignoresTickRate {
		triggeredCDTickRounded = math.Ceil(triggeredCDAdjusted*data.ServerTickRate) / data.ServerTickRate
	}
	triggerCDTickRounded := triggerCDAdjusted
	if mainSkill.TriggeredBy == nil || !mainSkill.TriggeredBy.IgnoresTickRate {
		triggerCDTickRounded = math.Ceil(triggerCDAdjusted*data.ServerTickRate) / data.ServerTickRate
	}

	//pob:src/Modules/CalcTriggers.lua:579-581
	actionCooldown := math.Max(triggerCD+addsCastTime, triggeredCD+addedCooldown)
	actionCooldownAdjusted := math.Max(triggerCDAdjusted, triggeredCDAdjusted)
	actionCooldownTickRounded := math.Max(triggerCDTickRounded, triggeredCDTickRounded)
	var override *float64
	if cooldownOverride != nil {
		v := utils.Or(cooldownOverride, 0)
		override = &v
		actionCooldown, actionCooldownAdjusted = v, v
		actionCooldownTickRounded = math.Ceil(v*data.ServerTickRate) / data.ServerTickRate
	}

	return triggerCooldownParts{
		icdr: icdr, addedCooldown: addedCooldown, triggerCD: triggerCD, triggeredCD: triggeredCD, addsCastTime: addsCastTime,
		cooldownOverride: override, triggeredCDAdjusted: triggeredCDAdjusted, triggerCDAdjusted: triggerCDAdjusted,
		actionCooldown: actionCooldown, actionCooldownAdjusted: actionCooldownAdjusted,
		actionCooldownTickRounded: actionCooldownTickRounded, ignoresTickRate: ignoresTickRate,
	}
}

//pob:src/Modules/CalcTriggers.lua:390-895
func calcsTriggers(env *Environment, actor *Actor) {
	if actor == nil || actor.MainSkill == nil || actor.MainSkill.SkillData == nil {
		return
	}

	mainSkill := actor.MainSkill
	//pob:block:src/Modules/CalcTriggers.lua:1593
	if mainSkill.SkillFlags[SkillFlagDisable] || mainSkill.mirageCopy {
		return
	}

	if mainSkill.ActiveEffect == nil || mainSkill.ActiveEffect.GrantedEffect == nil ||
		mainSkill.ActiveEffect.GrantedEffect.Raw == nil {
		return
	}

	config := triggerConfigForSkill(env, mainSkill)
	if config == nil {
		config = uniqueTriggerRung(env, mainSkill)
	}
	if config == nil {
		return
	}

	//pob:block:src/Modules/CalcTriggers.lua:1610
	if config.triggerChance == nil {
		config.triggerChance = mainSkill.ActiveEffect.triggerChance
	}

	//pob:src/Modules/CalcTriggers.lua:1611-1612
	if config.customHandler != nil {
		config.customHandler(env)
		return
	}

	//pob:block:src/Modules/CalcTriggers.lua:1594-1596,src/Modules/CalcTriggers.lua:1609
	triggerName := config.triggerName
	if triggerName == "" && mainSkill.TriggeredBy != nil && mainSkill.TriggeredBy.GrantedEffect != nil {
		triggerName = mainSkill.TriggeredBy.GrantedEffect.Name()
	}
	if triggerName == "" {
		triggerName = mainSkill.ActiveEffect.GrantedEffect.Name()
	}

	if config.actor != nil {
		actor = config.actor
		if actor.MainSkill == nil || actor.MainSkill.SkillData == nil ||
			actor.MainSkill.ActiveEffect == nil || actor.MainSkill.ActiveEffect.GrantedEffect == nil {
			return
		}
		mainSkill = actor.MainSkill
	}

	output := actor.Output

	var source *ActiveSkill
	var sourceCached *cachedTriggerSkill
	var trigRate float64
	if config.trigRate != nil {
		source = config.source
		trigRate = *config.trigRate
	}
	var triggeredSkills []*triggerRotationSkill

	var sourceWeaponFlag SkillFlag
	if config.sourceWeapon {
		if slot := socketGroupSlot(mainSkill); slot != "" {
			if strings.HasPrefix(slot, "Weapon 2") {
				sourceWeaponFlag = SkillFlagWeapon2Attack
			} else {
				sourceWeaponFlag = SkillFlagWeapon1Attack
			}
		}
	}

	if config.triggeredSkillCond != nil || config.triggerSkillCond != nil {
		for _, skill := range env.Player.ActiveSkillList {
			if config.triggerSkillCond != nil && skill != mainSkill &&
				config.triggerSkillCond(skill) && !isTriggered(skill) &&
				(sourceWeaponFlag == "" || skill.SkillFlags[sourceWeaponFlag]) {
				source, trigRate = findTriggerSkill(env, skill, source, trigRate, nil, config.comparer)
			}
			//pob:src/Modules/CalcTriggers.lua:408-410
			if config.triggeredSkillCond != nil && config.triggeredSkillCond(skill) {
				triggeredSkills = append(triggeredSkills, packageSkillDataForSimulation(env, skill))
			}
		}
	}

	//pob:src/Modules/CalcTriggers.lua:413
	if len(triggeredSkills) == 0 && config.triggeredSkillCond != nil {
		return
	}

	//pob:block:src/Modules/CalcTriggers.lua:414-418,src/Modules/CalcTriggers.lua:892
	if source == nil && config.source == nil {
		mainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
		mainSkill.infoMessage = fmt.Sprintf("No %s Triggering Skill Found", triggerName)
		mainSkill.infoTrigger = ""
	} else {
		mainSkill.infoTrigger = triggerName
	}

	if config.triggerSkillCond != nil {
		if source == nil {
			mainSkill.SkillData.Triggered = false
			return
		}
		sourceCached = buildActiveSkillCache(env, source)
	}

	//pob:block:src/Modules/CalcTriggers.lua:422-439
	breakdown := actor.Breakdown
	if breakdown != nil {
		if tb := mainSkill.TriggeredBy; mainSkill.SkillData.TriggeredByBrand && tb != nil && tb.MainSkill != nil {
			breakdown.Data["EffectiveSourceRate"] = &Entry{Lines: []string{
				fmt.Sprintf("%.2f ^8(base activation cooldown of %s)", tb.MainSkill.SkillData.RepeatFrequency, triggerName),
				fmt.Sprintf("* %.2f ^8(more activation frequency)", tb.ActivationFreqMore),
				fmt.Sprintf("* %.2f ^8(increased activation frequency)", tb.ActivationFreqInc),
				fmt.Sprintf("* %d ^8(attached Brands)", luaInt(tb.AttachedBrandCount)),
				fmt.Sprintf("= %.2f ^8(activation rate of %s)", trigRate, tb.MainSkill.ActiveEffect.GrantedEffect.Name()),
			}}
		} else {
			breakdown.Data["EffectiveSourceRate"] = &Entry{}
			if source != nil {
				if config.assumingEveryHitKills {
					breakdown.AddLine("EffectiveSourceRate", "Assuming every attack kills")
				}
				sourceName := config.sourceName
				if sourceName == "" {
					sourceName = source.ActiveEffect.GrantedEffect.Name()
				}
				breakdown.AddLine("EffectiveSourceRate", fmt.Sprintf("%.2f ^8(%s %s)", trigRate, sourceName, utils.Ternary(config.useCastRate, "cast rate", "attack rate")))
			}
		}
	}

	//pob:block:src/Modules/CalcTriggers.lua:464-470
	if source != nil && sourceCached != nil && source.SkillFlags[SkillFlagProjectile] &&
		strings.Contains(source.SkillPartName, "All") &&
		strings.Contains(source.SkillPartName, "Projectiles") && sourceCached.ProjectileCount > 0 {
		trigRate *= sourceCached.ProjectileCount
		if breakdown != nil {
			breakdown.AddLine("EffectiveSourceRate", fmt.Sprintf("x %.2f ^8(%d projectiles hit)", sourceCached.ProjectileCount, luaInt(sourceCached.ProjectileCount)))
		}
	}

	//pob:src/Modules/CalcTriggers.lua:483-500
	if mainSkill.SkillData.TriggeredByBattleMageCry && sourceCached != nil && source != nil &&
		source.SkillTypes[data.SkillTypeMelee] {
		battleMageDuration := ceilB(sourceCached.BattleMageCryDuration, data.ServerTickTime)
		battleMageCooldown := ceilB(sourceCached.BattleMageCryCooldown, data.ServerTickTime)
		battleMageHits := math.Max(math.Min(trigRate*battleMageDuration, sourceCached.BattleCryExertsCount), 0)
		if breakdown != nil {
			breakdown.AddLine("EffectiveSourceRate",
				fmt.Sprintf("^8(min(%.2f * %.2f, %d) = %.2f average number of exerted attacks capped by exerted count)", trigRate, battleMageDuration, luaInt(sourceCached.BattleCryExertsCount), battleMageHits),
				fmt.Sprintf("= %.2f / (%.2f + %.2f) ^8(the calculated number of exerted attacks happens every cooldown + duration)", battleMageHits, sourceCached.BattleMageCryCastTime, battleMageCooldown),
			)
		}
		trigRate = battleMageHits / (sourceCached.BattleMageCryCastTime + battleMageCooldown)
	}

	//pob:block:src/Modules/CalcTriggers.lua:502-519
	if mainSkill.ActiveEffect.GrantedEffect.Name() == "Combust" && sourceCached != nil && source != nil &&
		source.SkillTypes[data.SkillTypeMelee] {
		infernalCryDuration := ceilB(sourceCached.InfernalCryDuration, data.ServerTickTime)
		infernalCryCooldown := ceilB(sourceCached.InfernalCryCooldown, data.ServerTickTime)
		// Cap the number of hits that happen during the duration
		infernalCryHits := math.Max(math.Min(trigRate*infernalCryDuration, sourceCached.InfernalExertsCount), 0)
		if breakdown != nil {
			breakdown.AddLine("EffectiveSourceRate",
				fmt.Sprintf("^8(min(%.2f * %.2f, %d) = %.2f average number of exerted attacks capped by exerted count)", trigRate, infernalCryDuration, luaInt(sourceCached.InfernalExertsCount), infernalCryHits),
				fmt.Sprintf("= %.2f / (%.2f + %.2f) ^8(the calculated number of exerted attacks happens every cooldown + duration)", infernalCryHits, sourceCached.InfernalCryCastTime, infernalCryCooldown),
			)
		}
		// The hits happen every Infernal Cry cooldown + duration
		trigRate = infernalCryHits / (sourceCached.InfernalCryCastTime + infernalCryCooldown)
	}

	//pob:block:src/Modules/CalcTriggers.lua:521-556
	if mainSkill.SkillData.TriggeredByManaforged != nil && trigRate > 0 && sourceCached != nil {
		if triggeredCache := buildActiveSkillCache(env, mainSkill); triggeredCache != nil {
			triggeredManaCost := triggeredCache.ManaCostRaw
			if triggeredManaCost > 0 {
				manaSpentThreshold := triggeredManaCost * mainSkill.SkillData.ManaForgedArrowsPercentThreshold
				sourceManaCost := sourceCached.ManaCostRaw
				switch {
				case sourceManaCost <= 0:
					//pob:src/Modules/CalcTriggers.lua:550-553
					if breakdown != nil {
						breakdown.AddLine("EffectiveSourceRate", "Source skill has no mana cost")
					}
					trigRate = 0
				case manaSpentThreshold > 0:
					//pob:src/Modules/CalcTriggers.lua:532-548
					if breakdown != nil {
						first := ""
						if lines := breakdown.Data["EffectiveSourceRate"].Lines; len(lines) > 0 {
							first = lines[0]
						}
						breakdown.Data["EffectiveSourceRate"] = &Entry{Lines: []string{
							fmt.Sprintf("%.4f ^8(Mana cost of trigger source)", sourceManaCost),
							"",
							fmt.Sprintf("%.4f ^8(Mana Cost of triggered)", triggeredManaCost),
							fmt.Sprintf("%.2f ^8(Manaforged threshold multiplier)", mainSkill.SkillData.ManaForgedArrowsPercentThreshold),
							fmt.Sprintf("= %.4f ^8(Manaforged trigger threshold)", manaSpentThreshold),
							"",
							fmt.Sprintf("%.4f ^8(Manaforged trigger threshold)", manaSpentThreshold),
							fmt.Sprintf("/ %.4f ^8(Mana cost of trigger source)", sourceManaCost),
							fmt.Sprintf("= %.2f ^8(Skill usages required)", manaSpentThreshold/sourceManaCost),
							"",
							first,
							fmt.Sprintf("/ ceil(%.2f) ^8(%d Skill usages required)", manaSpentThreshold/sourceManaCost, luaInt(math.Ceil(manaSpentThreshold/sourceManaCost))),
						}}
					}
					trigRate /= math.Ceil(manaSpentThreshold / sourceManaCost)
				}
			}
		}
	}

	cd := triggerCooldowns(mainSkill, env.Player.MainSkill, config.ignoresTickRate)
	actionCooldownTickRounded, addsCastTime := cd.actionCooldownTickRounded, cd.addsCastTime
	if addsCastTime != 0 {
		output["addsCastTime"] = addsCastTime
	}

	// local so the infinity reaches :794's m_min, where it is the identity, without being published:
	triggerRateCap := math.Inf(1)
	if actionCooldownTickRounded != 0 {
		triggerRateCap = 1 / actionCooldownTickRounded
		output["TriggerRateCap"] = triggerRateCap
	}
	//pob:block:src/Modules/CalcTriggers.lua:587-614
	if triggerName == "Doom Blast" && triggerConfigInputString(env, "doomBlastSource") == "expiration" && sourceCached != nil {
		expirationRate := 1 / sourceCached.Duration
		if breakdown != nil && len(breakdown.Data["EffectiveSourceRate"].Lines) > 0 {
			breakdown.Data["EffectiveSourceRate"].Lines[0] = fmt.Sprintf("1 / %.2f ^8(source curse duration)", sourceCached.Duration)
		}
		if expirationRate > trigRate {
			env.Player.ModDB.AddMod(mod.NewFlag("UsesCurseOverlaps", true).Source("Config"))
			if breakdown != nil {
				e := breakdown.Data["EffectiveSourceRate"]
				e.Lines = slices.Insert(e.Lines, min(1, len(e.Lines)),
					fmt.Sprintf("%.2f ^8(%s cast rate)", trigRate, source.ActiveEffect.GrantedEffect.Name()),
					fmt.Sprintf("max(%.2f, %.2f) ^8(If a curse expires instantly curse expiration is equivalent to curse replacement)", expirationRate, trigRate),
				)
			}
		} else {
			trigRate = expirationRate
		}
	} else if triggerName == "Doom Blast" && triggerConfigInputString(env, "doomBlastSource") == "hexblast" {
		var hexBlast *ActiveSkill
		var rate float64
		for _, skill := range env.Player.ActiveSkillList {
			if skill.ActiveEffect.GrantedEffect.Name() == "Hexblast" && !isTriggered(skill) && skill != mainSkill {
				hexBlast, rate = findTriggerSkill(env, skill, hexBlast, rate, nil, nil)
			}
		}
		if hexBlast != nil {
			if breakdown != nil && len(breakdown.Data["EffectiveSourceRate"].Lines) > 0 {
				breakdown.Data["EffectiveSourceRate"].Lines[0] = fmt.Sprintf("1 / (%.2f + %.2f) ^8(sum of triggered curse and hexblast cast time)", 1/trigRate, 1/rate)
			}
			trigRate = 1 / (1/trigRate + 1/rate)
		}
	}

	//pob:block:src/Modules/CalcTriggers.lua:616-698
	if breakdown != nil && breakdown.Data["TriggerRateCap"] == nil {
		triggerRateCapBreakdown(env, actor, mainSkill, breakdown, cd, triggerName, triggerRateCap)
	}

	globalTrigger := source == nil || config.ignoreSourceRate
	if globalTrigger && math.IsInf(triggerRateCap, 1) {
		return
	}
	if env.Player.MainSkill != nil && env.Player.MainSkill.ActiveEffect.GrantedEffect.Name() == "Doom Blast" &&
		triggerConfigInputString(env, "doomBlastSource") == "vixen" {
		//pob:src/Modules/CalcTriggers.lua:700-713
		if gloves := env.Player.ItemList["Gloves"]; gloves == nil || !strings.Contains(gloves.Name, "Vixen's Entrapment") {
			output["VixenModeNoVixenGlovesWarn"] = 1
		}
		env.Player.ModDB.AddMod(mod.NewFlag("UsesCurseOverlaps", true).Source("Config"))
		icdr := calclib.Mod(mainSkill.SkillModList, mainSkill.SkillCfg, "CooldownRecovery")
		if icdr == 0 {
			icdr = 1
		}
		vixensCD := float64(0)
		if vixens := poe.GrantedEffectByID("SupportUniqueCastCurseOnCurse"); vixens != nil {
			if level, ok := raw2.GetCalculatedGrantedEffect(vixens).GetCalculatedLevels()[1]; ok && level != nil && level.Cooldown != nil {
				vixensCD = *level.Cooldown / icdr
			}
		}
		output["EffectiveSourceRate"], _ = calcMultiSpellRotationImpact329(env,
			[]*triggerRotationSkill{{uuid: cacheSkillUUID(env, env.Player.MainSkill), icdr: icdr}},
			trigRate, vixensCD, 100, nil)
		if vixensCD > 1/trigRate {
			output["VixensTooMuchCastSpeedWarn"] = 1
		}
		if breakdown != nil {
			breakdown.AddLine("EffectiveSourceRate",
				fmt.Sprintf("%.2f / %.2f = %.2f ^8(Vixen's trigger cooldown)", vixensCD*icdr, icdr, vixensCD),
				fmt.Sprintf("%.2f ^8(Simulated trigger rate of a curse socketed in Vixen's given ^7%.2f ^8CD and ^7%.2f ^8source rate)", output["EffectiveSourceRate"], vixensCD, trigRate),
			)
		}
	} else if globalTrigger {
		output["EffectiveSourceRate"] = triggerRateCap
	} else {
		//pob:src/Modules/CalcTriggers.lua:715
		output["EffectiveSourceRate"] = trigRate
	}

	//pob:block:src/Modules/CalcTriggers.lua:721-723
	if breakdown != nil && !config.sourceRateIsFinal {
		breakdown.AddLine("EffectiveSourceRate", fmt.Sprintf("= %.2f ^8(Effective source rate)", output["EffectiveSourceRate"]))
	}

	//pob:src/Modules/CalcTriggers.lua:725
	skillName := mainSkill.ActiveEffect.GrantedEffect.Name()
	if source != nil {
		skillName = source.ActiveEffect.GrantedEffect.Name()
	} else if config.source != nil {
		skillName = config.source.ActiveEffect.GrantedEffect.Name()
	} else if mainSkill.TriggeredBy != nil && mainSkill.TriggeredBy.GrantedEffect != nil {
		skillName = mainSkill.TriggeredBy.GrantedEffect.Name()
	}

	if output["EffectiveSourceRate"] != 0 && !config.skipEffectiveRate {
		var triggerChanceBreakdown []string
		triggerChance := triggerSourceChance(env, config, mainSkill, source, sourceCached, &triggerChanceBreakdown)

		ignoresTickRate := (config.ignoresTickRate ||
			(mainSkill.SkillData.StoredUses != nil && *mainSkill.SkillData.StoredUses > 1)) &&
			config.triggeredSkillCond == nil

		switch {
		case ignoresTickRate:
			//pob:src/Modules/CalcTriggers.lua:794
			overlaps := float64(1)
			if config.overlaps != nil {
				overlaps = *config.overlaps
			}
			output["SkillTriggerRate"] = math.Min(triggerRateCap, output["EffectiveSourceRate"]*overlaps)
			//pob:block:src/Modules/CalcTriggers.lua:795-805
			if breakdown != nil {
				if config.overlaps != nil {
					breakdown.Data["SkillTriggerRate"] = &Entry{Lines: []string{fmt.Sprintf("min(%.2f, %.2f *  %d) ^8(%d overlaps)", triggerRateCap, output["EffectiveSourceRate"], luaInt(overlaps), luaInt(overlaps))}}
				} else {
					breakdown.Data["SkillTriggerRate"] = &Entry{Lines: []string{fmt.Sprintf("min(%.2f, %.2f)", triggerRateCap, output["EffectiveSourceRate"])}}
				}
			}

		case globalTrigger && config.triggeredSkillCond == nil:
			output["SkillTriggerRate"] = output["EffectiveSourceRate"]

		default:
			simTriggerCD := triggeredByCooldown(mainSkill)
			if simTriggerCD == 0 {
				simTriggerCD = mainSkill.SkillData.Cooldown
			}
			if mainSkill.SkillData.TriggeredByBrand {
				simTriggerCD = 0
			}
			rotation := triggeredSkills
			if config.triggeredSkillCond == nil {
				rotation = []*triggerRotationSkill{packageSkillDataForSimulation(env, mainSkill)}
			}
			//pob:src/Modules/CalcTriggers.lua:809
			var simBreakdown *triggerSimBreakdown
			output["SkillTriggerRate"], simBreakdown = calcMultiSpellRotationImpact329(env,
				rotation, output["EffectiveSourceRate"], simTriggerCD, triggerChance, actor)

			triggerBotsEffective := actor.ModDB != nil && actor.ModDB.Flag(nil, "HaveTriggerBots") &&
				mainSkill.SkillTypes[data.SkillTypeSpell]
			if triggerBotsEffective {
				output["SkillTriggerRate"] *= 2
			}
			//pob:block:src/Modules/CalcTriggers.lua:818-866
			if breakdown != nil {
				label := "Effective source rate"
				if mainSkill.SkillData.TriggeredByBrand && source != nil {
					label = fmt.Sprintf("%s activations per second", source.ActiveEffect.GrantedEffect.Name())
				} else if source == nil {
					label = fmt.Sprintf("%s triggers per second", skillName)
				}
				lines := []string{
					fmt.Sprintf("%.2f ^8(%s)", output["EffectiveSourceRate"], label),
					fmt.Sprintf("/ %.2f ^8(Estimated impact of skill rotation, cooldown alignment and trigger chance)", math.Max(output["EffectiveSourceRate"]/output["SkillTriggerRate"], 1)),
					fmt.Sprintf("= %.2f ^8per second", output["SkillTriggerRate"]),
				}
				if triggerBotsEffective {
					lines = slices.Insert(lines, 2, "x 2 ^8(Trigger bots effectively cause the skill to trigger twice)")
				}
				if triggerChance != 100 {
					lines = slices.Insert(lines, 0, "")
					lines = slices.Insert(lines, 0, fmt.Sprintf("= %.2f%% ^8(Effective chance to trigger)", triggerChance))
					for _, line := range triggerChanceBreakdown {
						lines = slices.Insert(lines, 0, line)
					}
					lines = slices.Insert(lines, 0, "100% ^8(Base chance)")
				}
				breakdown.Data["SkillTriggerRate"] = &Entry{Lines: lines}
				if simBreakdown != nil {
					simData := &Entry{Columns: []BCol{
						{Label: "Rate", Key: "rate"},
						{Label: "Skill Name", Key: "skillName"},
						{Label: "Slot Name", Key: "slotName"},
						{Label: "Gem Index", Key: "gemIndex"},
					}}
					for _, rateData := range simBreakdown.rates {
						t := strings.Split(rateData.name, "_")
						row := map[string]string{"rate": fmt.Sprintf("%g", luaRound(rateData.rate, 2))}
						for i, k := range []string{"skillName", "slotName", "gemIndex"} {
							if i < len(t) {
								row[k] = t[i]
							}
						}
						simData.Rows = append(simData.Rows, row)
					}
					simData.Lines = append(simData.Lines, fmt.Sprintf("Simulation duration: %.2f ^8(In game source skill usage duration in seconds)", simBreakdown.simTime))
					breakdown.Data["SimData"] = simData
				}
			}
		}
	} else {
		//pob:src/Modules/CalcTriggers.lua:869-874
		if breakdown != nil {
			breakdown.Data["SkillTriggerRate"] = &Entry{Lines: []string{"The trigger needs to be triggered for any skill to be triggered."}}
		}
		output["SkillTriggerRate"] = 0
	}

	mainSkill.SkillData.Triggered = true
	mainSkill.SkillData.TriggerRate = utils.Ptr(output["SkillTriggerRate"])

	//pob:src/Modules/CalcTriggers.lua:880
	if source != nil {
		addTriggerIncMoreMods(mainSkill, source)
	} else {
		addTriggerIncMoreMods(mainSkill, mainSkill)
	}

	//pob:src/Modules/CalcTriggers.lua:881-882
	if source != nil && source != mainSkill {
		mainSkill.SkillData.triggerSource = sourceCached
	}
	if source != nil || config.source != nil {
		mainSkill.infoMessage = triggerInfoMessage(env, actor, config, triggerName, mainSkill, source)
	}
}

//pob:block:src/Modules/CalcTriggers.lua:881-890
func triggerInfoMessage(env *Environment, actor *Actor, config *triggerConfig, triggerName string, mainSkill *ActiveSkill, source *ActiveSkill) string {
	var msg string
	if source != nil && source != mainSkill {
		sourceName := source.ActiveEffect.GrantedEffect.Name()
		name := mainSkill.ActiveEffect.GrantedEffect.Name()
		if triggerName != "" && triggerName != sourceName {
			name = triggerName
		}
		suffix := "'s Trigger: "
		if env.Minion != nil && env.Minion.Actor == actor {
			suffix = "'s attack Trigger: "
		}
		msg = name + suffix + sourceName
		if config.customTriggerName != "" {
			msg = config.customTriggerName + sourceName
		}
	} else if mainSkill.TriggeredBy != nil && mainSkill.TriggeredBy.GrantedEffect != nil {
		msg = mainSkill.TriggeredBy.GrantedEffect.Name()
	} else {
		msg = triggerName + " Trigger"
	}
	if mainSkill.SkillData.TriggeredByBrand && mainSkill.TriggeredBy != nil {
		count := int(mainSkill.TriggeredBy.AttachedBrandCount)
		msg += ":" + fmt.Sprintf("%d attached Brand%s", count, utils.Ternary(count == 1, "", "s"))
	}
	return msg
}

//pob:block:src/Modules/CalcTriggers.lua:728-789
func triggerSourceChance(env *Environment, config *triggerConfig, mainSkill *ActiveSkill, source *ActiveSkill, cached *cachedTriggerSkill, breakdownLines *[]string) float64 {
	addLine := func(format string, args ...any) {
		if breakdownLines != nil {
			*breakdownLines = append(*breakdownLines, fmt.Sprintf(format, args...))
		}
	}
	triggerChance := 100.0

	if source != nil && cached != nil && !config.triggerOnUse &&
		(source.SkillTypes[data.SkillTypeMelee] || source.SkillTypes[data.SkillTypeAttack]) {

		dualWielding := env.Player.WeaponData1 != nil && env.Player.WeaponData1.Type != "" &&
			env.Player.WeaponData2 != nil && env.Player.WeaponData2.Type != ""
		doubleHits := dualWielding && source.SkillData != nil &&
			source.SkillData.DoubleHitsWhenDualWielding

		if hitChance := cached.HitChance; hitChance != 100 {
			if doubleHits {
				//pob:src/Modules/CalcTriggers.lua:743
				mainHandHit, offHandHit := cached.MainHandHitChance, cached.OffHandHitChance
				effective := mainHandHit*offHandHit/100 +
					mainHandHit*(100-offHandHit)/100 +
					(100-mainHandHit)*offHandHit/100
				triggerChance = triggerChance * effective / 100
				addLine("x %.2f%% ^8(%s effective hit chance for skills that hit with both weapons)", effective, source.ActiveEffect.GrantedEffect.Name())
			} else {
				//pob:src/Modules/CalcTriggers.lua:749
				triggerChance = triggerChance * hitChance / 100
				addLine("x %.2f%% ^8(%s hit chance)", hitChance, source.ActiveEffect.GrantedEffect.Name())
			}
		}

		if mainSkill != nil && mainSkill.SkillData != nil && mainSkill.SkillData.TriggerOnCrit {
			if config.triggerChance == nil {
				chance := mainSkill.SkillData.ChanceToTriggerOnCrit
				if chance == 0 {
					chance = cached.ChanceToTriggerOnCrit
				}
				if chance != 0 {
					config.triggerChance = utils.Ptr(chance)
				}
			}

			if critChance := cached.CritChance; critChance != 100 {
				if doubleHits {
					//pob:src/Modules/CalcTriggers.lua:768
					mainHandCrit, offHandCrit := cached.MainHandCritChance, cached.OffHandCritChance
					effective := mainHandCrit*offHandCrit/100 +
						mainHandCrit*(100-offHandCrit)/100 +
						(100-mainHandCrit)*offHandCrit/100
					triggerChance = triggerChance * effective / 100
					addLine("x %.2f%% ^8(%s effective crit chance for skills that hit with both weapons)", effective, source.ActiveEffect.GrantedEffect.Name())
				} else {
					//pob:src/Modules/CalcTriggers.lua:774
					triggerChance = triggerChance * critChance / 100
					addLine("x %.2f%% ^8(%s crit chance)", critChance, source.ActiveEffect.GrantedEffect.Name())
				}
			}
		}
	}

	//pob:src/Modules/CalcTriggers.lua:784-788
	if config.triggerChance != nil && *config.triggerChance != 100 {
		triggerChance = triggerChance * *config.triggerChance / 100
		addLine("x %.2f%% ^8(chance to trigger)", *config.triggerChance)
	}

	return triggerChance
}

func triggerConfigInputString(env *Environment, name string) string {
	if env.Build == nil {
		return ""
	}
	return resolvedConfigOption(env.Build, name)
}

//pob:src/Modules/CalcTriggers.lua:1399-1474
func snipeTriggerConfig(env *Environment, mainSkill *ActiveSkill) *triggerConfig {
	snipeStages := math.Min(env.Player.ModDB.Sum(mod.TypeBase, nil, "Multiplier:SnipeStage"),
		env.Player.ModDB.Sum(mod.TypeBase, nil, "Multiplier:SnipeStagesMax"))
	snipeHitMulti := mainSkill.SkillModList.Sum(mod.TypeBase, mainSkill.SkillCfg, "snipeHitMulti")
	snipeAilmentMulti := mainSkill.SkillModList.Sum(mod.TypeBase, mainSkill.SkillCfg, "snipeAilmentMulti")
	var triggeredSkills []*ActiveSkill

	for _, skill := range env.Player.ActiveSkillList {
		if skill.SkillData.TriggeredBySnipe && skill.SocketGroup != nil && socketGroupSlot(skill) == socketGroupSlot(mainSkill) {
			triggeredSkills = append(triggeredSkills, skill)
		}
	}

	if mainSkill.ActiveEffect.GrantedEffect.Name() == "Snipe" {
		if mainSkill.mirageCopy {
			// Snipe is being used by some other skill. In this case snipe does not get more damage mods
			snipeStages = 0
		} else {
			// max(1, snipeStages) makes it behave consistently with other channeled ranged skills (scourge arrow)
			mainSkill.SkillData.HitTimeMultiplier = math.Max(1, snipeStages) - 0.5 // First stage takes 0.5x time to channel compared to subsequent stages
		}
		if len(triggeredSkills) < 1 {
			// Snipe is being used as a standalone skill
			mainSkill.SkillModList.AddMod(mod.NewFloat("Multiplier:SnipeStages", mod.TypeBase, snipeStages).Source("Snipe"))
			mainSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, snipeHitMulti).Source("Snipe").Flag(mod.MFlagHit).
				Tag(mod.Multiplier("SnipeStages")))
			mainSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, snipeAilmentMulti).Source("Snipe").Flag(mod.MFlagAilment).
				Tag(mod.Multiplier("SnipeStages")))
		} else {
			// Snipe is being used as a trigger source, it triggers other skills but does no damage it self
			for _, flag := range []string{"DealNoLightning", "DealNoCold", "DealNoFire", "DealNoChaos", "DealNoPhysical"} {
				mainSkill.SkillModList.AddMod(mod.NewFlag(flag, true).Tag(mod.SkillName("Snipe").Transfigured(true)))
			}
		}
		return nil
	}

	currentSkillSnipeIndex := 0
	for index, skill := range triggeredSkills {
		if skill == mainSkill {
			currentSkillSnipeIndex = index + 1
			break
		}
	}

	// Does snipe have enough stages to trigger this skill?
	if currentSkillSnipeIndex == 0 || float64(currentSkillSnipeIndex) > snipeStages {
		mainSkill.SkillData.Triggered = false
		mainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
		mainSkill.infoMessage = "Not enough Snipe stages to trigger this skill"
		mainSkill.infoTrigger = ""
		return nil
	}
	var source *ActiveSkill
	var trigRate *float64
	mainSkill.SkillModList.AddMod(mod.NewFloat("Multiplier:SnipeStages", mod.TypeBase, snipeStages).Source("Snipe"))
	mainSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, snipeAilmentMulti).Source("Snipe").Flag(mod.MFlagAilment).
		Tag(mod.Multiplier("SnipeStages")))
	mainSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, snipeHitMulti).Source("Snipe").Flag(mod.MFlagHit).
		Tag(mod.Multiplier("SnipeStages")))
	for _, skill := range env.Player.ActiveSkillList {
		if skill.ActiveEffect.GrantedEffect.Name() != "Snipe" || skill.SocketGroup == nil || socketGroupSlot(skill) != socketGroupSlot(mainSkill) {
			continue
		}
		skill.SkillData.HitTimeMultiplier = snipeStages - 0.5
		cached := buildActiveSkillCache(env, skill)
		if cached == nil || !cached.hitSpeedSet {
			continue
		}
		if !skill.SkillFlags[SkillFlagDisable] && (skill.SkillCfg != nil && !skill.SkillCfg.SkillCond["usedByMirage"]) &&
			!skill.SkillTypes[data.SkillTypeOtherThingUsesSkill] && (source == nil || cached.HitSpeed > utils.UnwrapOrF(trigRate, 0)) {
			trigRate = utils.Ptr(cached.HitSpeed)
			env.Player.Output["ChannelTimeToTrigger"] = cached.HitTime
			source = skill
		}
	}

	return &triggerConfig{trigRate: trigRate, source: source}
}
