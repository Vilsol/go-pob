package raw

import (
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

// SkillStatMaps is PoB's hand-written per-skill `statMap` table (src/Data/Skills/*.lua),
// keyed by granted-effect id then stat key. calcs.mergeSkillInstanceMods
// (src/Modules/CalcActiveSkill.lua:62) looks a skill's own stat up here first; only when a
// skill declares nothing for that stat does data.skillStatMapMeta.__index
// (src/Modules/Data.lua:668) fall the lookup through to the global map, which go-pob ports
// separately as SkillStatMap. GetStatMapEntry (granted_effects.go) mirrors that order.
//
// The distinction matters because the same raw stat maps differently per skill: PoB tags
// `base_physical_damage_reduction_rating` with GlobalEffect/Aura on Determination so it
// reaches the player ModDB as an aura, while the global map leaves it skill-local.
//
// An empty Mods list means one of TWO things, and this header used to claim only the first.
//
// 245 entries are empty because PoB DECLARES the stat with no mods (42 of them suppress a real
// global entry). Those are deliberate: emitting the entry keeps the fallback suppressed the way
// PoB does, and dropping it would apply a global mapping PoB never uses.
//
// 2 are empty because every mod in them was OMITTED below (0 of them suppress a real global
// entry). Those are LOSSES, not decisions — PoB applies something there and we apply nothing.
// The split is mechanical, not a judgement: the dump says how many mods the entry had.
//
// ⚠️ THEY ARE STILL EMITTED, AND THAT IS MEASURED RATHER THAN ASSUMED. It reads as though
// dropping a lost entry would be better, because the global map would then be consulted. It is
// not: dropping all of them scored 7115624 / 7310135 against a base of 7115624 / 7310135 — a
// MEASURED ZERO on an unmoved denominator — and the population is NOT empty (237 corpus builds
// carry SupportSummonPhantasm), so this is inert, not vacuous. The reason is the distinction this
// header opens with: the GLOBAL entry a lost one shadows is usually skill-local, so consulting it
// would reach nobody either. Emitting the empty entry costs nothing and is the more faithful
// half. Re-open this if a global entry a loss shadows gains a GlobalEffect tag.
//
// ⚠️ THE GROUP FORM WAS ONE OF THESE LOSSES AND IS NOT ANY MORE — read Groups on raw.StatMap
// before re-deriving it. PoB's `{ mod(...), div = 60 }` attaches a scalar to ONE mod rather than
// to the entry (src/Modules/CalcActiveSkill.lua:64-73), and RejuvenationTotem's two groups divide
// by 60 and by 400 (src/Data/Skills/act_str.lua:9329-9332), so no single entry-level Div could
// carry them. Both halves were missing at once: oracle/skill_stat_map.lua ran a group through
// encodeMod, which dumped the scalar's NAME and not its value, and raw.StatMap had nowhere to put
// one. Both are fixed, and the dump now carries a `groupCount` this emitter REQUIRES, so a stale
// dump refuses instead of quietly emptying the two entries again.
//
// Generated from the PINNED PoB (717 skills, 1661 statMap entries, 1514 mods) by
// oracle/skill_stat_map.lua + gopob's skill-stat-maps emitter. The pin's identity is in
// scripts/pob_pin.json; this header does not restate it, because the per-skill dump does
// not carry pobVersion and a hardcoded version is how this line came to claim "v2.18.1"
// long after the pin moved to 3.29. 1512 mods are emitted;
// 2 are not representable and are omitted rather than mapped lossily:
//   - 2 nested mod: name/type live inside tags[0], which mod/ has no constructor for [go-absent:mod.NestedMod]
//
// SkillPart entries follow the precedent set by SkillBaseMods: every part is carried, and
// so is PoB's `skillPartList` form. moddb evaluates the tag (ModStore.lua:529-549 ->
// moddb/mod_store.go evalSkillPartTag) with PoB's drop-when-not-matching semantics, so a
// row written for part 2 reaches only a build on part 2. The earlier revision kept part 1
// alone and dropped 35 part-2, 7 part-3 and 16 part-LIST mods because the tag was then a
// TODO and keeping them would have applied every part of a skill at once. All 58 are now
// back: the last 2 (SupportStormBarrier, which is Infused Channelling) also carried a
// GlobalEffect effectCond, and that is no longer a reason to omit.
//
// This table replaces the hand-written SkillStatMapOverrides that the aura, guard, Blood
// Rage / Righteous Fire and Defiance Banner batches grew one skill at a time. 41 of its 42
// entries are reproduced here byte for byte. The 42nd is not, deliberately: the hand-written
// Discipline entry gave `energy_shield_delay_-%` a GlobalEffect/Aura tag, but PoB passes
// that tag in mod()'s `flags` argument slot (src/Data/Skills/act_int.lua:2621), so upstream
// the mod ends up with NO tags, never gets promoted out of the skill's local mod list, and
// never reaches the player. Emitting the aura would beat PoB's own numbers rather than match
// them, so the mod is omitted and the entry left empty.
var SkillStatMaps = buildSkillStatMaps()

func buildSkillStatMaps() map[string]map[string]*StatMap {
	m := make(map[string]map[string]*StatMap, 717)
	skillStatMapsPart0(m)
	skillStatMapsPart1(m)
	skillStatMapsPart2(m)
	skillStatMapsPart3(m)
	skillStatMapsPart4(m)
	skillStatMapsPart5(m)
	return m
}

func skillStatMapsPart0(m map[string]map[string]*StatMap) {
	// src/Data/Skills/spectre.lua:8375
	m["ABTTAzmeriBasaliskShroud"] = map[string]*StatMap{
		"base_chaos_damage_taken_per_minute": {
			Mods: []mod.Mod{skill("ChaosDot", 0)},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/spectre.lua:9166
	m["ABTTAzmeriGoddessAura"] = map[string]*StatMap{
		"attack_maximum_added_chaos_damage": {
			Mods: []mod.Mod{mod.NewFloat("ChaosMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Buff").Name("Judgemental Spirit"), mod.MonsterTag("Skeleton"))},
		},
		"attack_maximum_added_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Buff").Name("Judgemental Spirit"), mod.MonsterTag("Skeleton"))},
		},
		"attack_minimum_added_chaos_damage": {
			Mods: []mod.Mod{mod.NewFloat("ChaosMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Buff").Name("Judgemental Spirit"), mod.MonsterTag("Skeleton"))},
		},
		"attack_minimum_added_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Buff").Name("Judgemental Spirit"), mod.MonsterTag("Skeleton"))},
		},
	}
	// src/Data/Skills/spectre.lua:9971
	m["ABTTAzmeriSpiderLeaderAura"] = map[string]*StatMap{
		"attack_maximum_added_chaos_damage": {
			Mods: []mod.Mod{mod.NewFloat("ChaosMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Buff").Name("Judgemental Spirit"), mod.MonsterTag("Spider"))},
		},
		"attack_minimum_added_chaos_damage": {
			Mods: []mod.Mod{mod.NewFloat("ChaosMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Buff").Name("Judgemental Spirit"), mod.MonsterTag("Spider"))},
		},
	}
	// src/Data/Skills/act_str.lua:23
	m["Absolution"] = map[string]*StatMap{
		"skill_physical_damage_%_to_convert_to_lightning": {
			Mods: []mod.Mod{
				mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0),
				mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0)}),
			},
		},
	}
	// src/Data/Skills/act_str.lua:119
	m["AbsolutionAltX"] = map[string]*StatMap{
		"skill_physical_damage_%_to_convert_to_lightning": {
			Mods: []mod.Mod{
				mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0),
				mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0)}),
			},
		},
	}
	// src/Data/Skills/minion.lua:1632
	m["AbyssalFiendEnfeebleAura"] = map[string]*StatMap{
		"accuracy_rating_+%": {
			Mods: []mod.Mod{mod.NewFloat("Accuracy", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"enfeeble_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique").Neg(true))},
		},
		"enfeeble_damage_+%_vs_rare_or_unique_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique"))},
		},
	}
	// src/Data/Skills/minion.lua:1718
	m["AbyssalFiendPunishmentAura"] = map[string]*StatMap{
		"damage_taken_+%_on_low_life": {
			Mods: []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("LowLife"))},
		},
	}
	// src/Data/Skills/minion.lua:1675
	m["AbyssalFiendTemporalChainsAura"] = map[string]*StatMap{
		"buff_time_passed_+%_other_than_temporal_chains": {
			Mods: []mod.Mod{mod.NewFloat("BuffExpireFaster", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"temporal_chains_action_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("TemporalChainsActionSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique").Neg(true))},
		},
		"temporal_chains_action_speed_+%_vs_rare_or_unique_final": {
			Mods: []mod.Mod{mod.NewFloat("TemporalChainsActionSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique"))},
		},
	}
	// src/Data/Skills/act_dex.lua:18
	m["AlchemistsMark"] = map[string]*StatMap{
		"alchemists_mark_grant_attacker_x_flask_charges_when_hit_once_per_3s": {
			Mods: []mod.Mod{mod.NewFloat("FlaskChargesGenerated", mod.TypeBase, 0)},
			Div:  utils.Ptr(float64(3)),
		},
		"alchemists_mark_igniter_creates_burning_ground_%_ignite_damage": {
			Mods: []mod.Mod{mod.NewFloat("IgniteDpsAsBurningGround", mod.TypeMAX, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"alchemists_mark_poisoner_creates_caustic_ground_%_poison_damage": {
			Mods: []mod.Mod{mod.NewFloat("PoisonDpsAsCausticGround", mod.TypeMAX, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_dex.lua:107
	m["Ambush"] = map[string]*StatMap{
		"ambush_additional_critical_strike_chance_permyriad": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeBase, 0).Flag(mod.MFlagMelee).Tag(mod.GlobalEffect("Buff").Name("Ambush"))},
			Div:  utils.Ptr(float64(100)),
		},
		"vanishing_ambush_critical_strike_multiplier_+": {
			Mods: []mod.Mod{mod.NewFloat("CritMultiplier", mod.TypeBase, 0).Flag(mod.MFlagMelee).Tag(mod.GlobalEffect("Buff").Name("Ambush"))},
		},
	}
	// src/Data/Skills/act_str.lua:289
	m["AncestralCry"] = map[string]*StatMap{
		"ancestral_cry_elemental_resist_%_per_5_power_up_to_cap": {
			Mods: []mod.Mod{mod.NewFloat("ElementalResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Warcry").Div(5).Limit(30))},
		},
		"ancestral_cry_maximum_elemental_resist_%_per_10_power_up_to_cap": {
			Mods: []mod.Mod{mod.NewFloat("ElementalResistMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Warcry").Div(10).Limit(30))},
		},
		"skill_empower_limitation_specifier_for_stat_description": {
			Mods: []mod.Mod{},
		},
		"skill_empowers_next_x_melee_attacks": {
			Mods: []mod.Mod{mod.NewFloat("AncestralExertedAttacks", mod.TypeBase, 0).Tag(mod.SkillType("MeleeSingleTarget"))},
		},
	}
	// src/Data/Skills/act_str.lua:386
	m["Anger"] = map[string]*StatMap{
		"attack_maximum_added_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"attack_minimum_added_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"spell_maximum_added_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagSpell).Tag(mod.GlobalEffect("Aura"))},
		},
		"spell_minimum_added_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagSpell).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/other.lua:84
	m["AnimateGuardiansWeapon"] = map[string]*StatMap{
		"active_skill_damage_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)})},
		},
		"attack_maximum_added_physical_damage": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)})},
		},
		"attack_minimum_added_physical_damage": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)})},
		},
		"number_of_animated_weapons_allowed": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:AnimatedWeapon", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:197
	m["AnimateWeapon"] = map[string]*StatMap{
		"attack_maximum_added_physical_damage": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)})},
		},
		"attack_maximum_added_physical_damage_for_ethereal_blades": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.ActorCondition("parent", "AnimatingLingeringBlades"))})},
		},
		"attack_minimum_added_physical_damage": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)})},
		},
		"attack_minimum_added_physical_damage_for_ethereal_blades": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.ActorCondition("parent", "AnimatingLingeringBlades"))})},
		},
	}
	// src/Data/Skills/act_dex.lua:397
	m["AnimateWeaponAltY"] = map[string]*StatMap{
		"attack_maximum_added_physical_damage": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)})},
		},
		"attack_maximum_added_physical_damage_for_ethereal_blades": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.ActorCondition("parent", "AnimatingLingeringBlades"))})},
		},
		"attack_minimum_added_physical_damage": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)})},
		},
		"attack_minimum_added_physical_damage_for_ethereal_blades": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.ActorCondition("parent", "AnimatingLingeringBlades"))})},
		},
	}
	// src/Data/Skills/other.lua:5157
	m["AnimosityUniqueSkill"] = map[string]*StatMap{
		"base_physical_damage_to_deal_per_minute": {
			Mods: []mod.Mod{skill("PhysicalDot", 0).Tag(mod.Multiplier("WeepingBlackStacks").Limit(5))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/other.lua:130
	m["ApproachingFlames"] = map[string]*StatMap{
		"display_touch_of_fire": {
			Mods: []mod.Mod{mod.NewFlag("FlameTouched", true).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_int.lua:19
	m["Arc"] = map[string]*StatMap{
		"arc_damage_+%_final_for_each_remaining_chain": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.PerStat(1, "ChainRemaining"))},
		},
	}
	// src/Data/Skills/act_int.lua:174
	m["ArcAltY"] = map[string]*StatMap{
		"arc_damage_+%_final_for_each_remaining_chain": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.PerStat(1, "ChainRemaining"))},
		},
	}
	// src/Data/Skills/act_int.lua:340
	m["ArcaneCloak"] = map[string]*StatMap{
		"arcane_cloak_consume_%_of_mana": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ArcaneCloakConsumedMana", mod.TypeBase, 0).Tag(mod.GlobalEffect("Guard").Unscalable(true))},
			Div:  utils.Ptr(float64(100)),
		},
		"arcane_cloak_damage_absorbed_%": {
			Mods: []mod.Mod{mod.NewFloat("GuardAbsorbRate", mod.TypeBase, 0).Tag(mod.GlobalEffect("Guard").Unscalable(true))},
		},
		"arcane_cloak_gain_%_of_consumed_mana_as_lightning_damage": {
			Mods: []mod.Mod{
				mod.NewFloat("LightningMin", mod.TypeBase, 0).Tag(mod.PercentStatVar("ManaUnreserved", "ArcaneCloakConsumedMana"), mod.GlobalEffect("Buff")),
				mod.NewFloat("LightningMax", mod.TypeBase, 0).Tag(mod.PercentStatVar("ManaUnreserved", "ArcaneCloakConsumedMana"), mod.GlobalEffect("Buff")),
			},
		},
	}
	// src/Data/Skills/act_dex.lua:587
	m["ArcticArmour"] = map[string]*StatMap{
		"base_immune_to_freeze": {
			Mods: []mod.Mod{mod.NewFlag("FreezeImmune", true).Tag(mod.GlobalEffect("Buff"))},
		},
		"new_arctic_armour_fire_damage_taken_when_hit_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("FireDamageTakenWhenHit", mod.TypeMore, 0).Tag(mod.Condition("Stationary"), mod.GlobalEffect("Buff"))},
		},
		"new_arctic_armour_physical_damage_taken_when_hit_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTakenWhenHit", mod.TypeMore, 0).Tag(mod.Condition("Stationary"), mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_int.lua:719
	m["ArmageddonBrand"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:811
	m["ArmageddonBrandAltX"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:899
	m["ArmageddonBrandAltY"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/other.lua:238
	m["AspectOfTheAvian"] = map[string]*StatMap{
		"base_movement_velocity_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff").Name("Avian's Flight").EffectCond("AviansFlightActive"))},
		},
		"chance_to_deal_double_damage_%": {
			Mods: []mod.Mod{mod.NewFloat("DoubleDamageChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Avian's Might").EffectCond("AviansMightActive"))},
		},
		"minion_chance_to_deal_double_damage_%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("DoubleDamageChance", mod.TypeBase, 0)}).Tag(mod.GlobalEffect("Buff").Name("Avian's Might").EffectCond("AviansMightActive"))},
		},
		"minion_movement_speed_+%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0)}).Tag(mod.GlobalEffect("Buff").Name("Avian's Flight").EffectCond("AviansFlightActive"))},
		},
	}
	// src/Data/Skills/other.lua:281
	m["AspectOfTheCat"] = map[string]*StatMap{
		"attack_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Buff").Name("Cat's Agility").EffectCond("CatsAgilityActive"))},
		},
		"avoid_damage_%": {
			Mods: []mod.Mod{mod.NewFloat("AvoidAllDamageFromHitsChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Cat's Stealth").EffectCond("CatsStealthActive"))},
		},
		"cast_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagCast).Tag(mod.GlobalEffect("Buff").Name("Cat's Agility").EffectCond("CatsAgilityActive"))},
		},
		"skill_buff_grants_critical_strike_chance_+%": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff").Name("Cat's Stealth").EffectCond("CatsStealthActive"))},
		},
	}
	// src/Data/Skills/other.lua:325
	m["AspectOfTheCrab"] = map[string]*StatMap{
		"max_crab_aspect_stacks": {
			Mods: []mod.Mod{mod.NewFloat("CrabBarriersMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"physical_damage_reduction_%_per_crab_aspect_stack": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, 0).Tag(mod.Multiplier("CrabBarrier"), mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/other.lua:356
	m["AspectOfTheSpider"] = map[string]*StatMap{
		"base_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Debuff").Name("Spider's Web"))},
		},
		"spider_aspect_max_web_count": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:SpiderWebApplyStackMax", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:989
	m["AssassinsMark"] = map[string]*StatMap{
		"enemy_additional_critical_strike_chance_against_self": {
			Mods: []mod.Mod{mod.NewFloat("SelfCritChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
			Div:  utils.Ptr(float64(100)),
		},
		"enemy_additional_critical_strike_multiplier_against_self": {
			Mods: []mod.Mod{mod.NewFloat("SelfCritMultiplier", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"life_granted_when_killed": {
			Mods: []mod.Mod{mod.NewFloat("SelfLifeOnKill", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"mana_granted_when_killed": {
			Mods: []mod.Mod{mod.NewFloat("SelfManaOnKill", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_str.lua:646
	m["Autoexertion"] = map[string]*StatMap{
		"call_to_arms_behaviour": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:425
	m["Automation"] = map[string]*StatMap{
		"automation_behaviour": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_str.lua:2548
	m["AvengingFlame"] = map[string]*StatMap{
		"active_skill_base_area_of_effect_radius": {
			Mods: []mod.Mod{skill("radius", 0)},
		},
		"infernal_bolt_base_fire_damage_%_maximum_life": {
			Mods: []mod.Mod{skill("lifeDealtAsFire", 0)},
		},
		"infernal_bolt_triggered_when_totem_with_this_skill_hit_by_enemy": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/spectre.lua:1978
	m["AxisEnfeeble"] = map[string]*StatMap{
		"accuracy_rating_+%": {
			Mods: []mod.Mod{mod.NewFloat("Accuracy", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"enfeeble_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique").Neg(true))},
		},
		"enfeeble_damage_+%_vs_rare_or_unique_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique"))},
		},
	}
	// src/Data/Skills/spectre.lua:3684
	m["AxisTemporalChains"] = map[string]*StatMap{
		"buff_time_passed_+%_other_than_temporal_chains": {
			Mods: []mod.Mod{mod.NewFloat("BuffExpireFaster", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"temporal_chains_action_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("TemporalChainsActionSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique").Neg(true))},
		},
		"temporal_chains_action_speed_+%_vs_rare_or_unique_final": {
			Mods: []mod.Mod{mod.NewFloat("TemporalChainsActionSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique"))},
		},
	}
	// src/Data/Skills/spectre.lua:7796
	m["AzmeriAdmiralPrecision"] = map[string]*StatMap{
		"accuracy_rating": {
			Mods: []mod.Mod{mod.NewFloat("Accuracy", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"skill_buff_grants_critical_strike_chance_+%": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/spectre.lua:8013
	m["AzmeriBarrageDemonSpinestorm"] = map[string]*StatMap{
		"fire_storm_fireball_delay_ms": {
			Mods: []mod.Mod{skill("hitTimeOverride", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"firestorm_base_area_of_effect_+%": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffectPrimary", mod.TypeIncrease, 0)},
		},
	}
	// src/Data/Skills/spectre.lua:9368
	m["AzmeriBirdGrace"] = map[string]*StatMap{
		"base_evasion_rating": {
			Mods: []mod.Mod{mod.NewFloat("Evasion", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"grace_aura_evasion_rating_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Evasion", mod.TypeMore, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/spectre.lua:9305
	m["AzmeriBirdScreechExposure"] = map[string]*StatMap{
		"lightning_exposure_on_hit_magnitude": {
			Mods: []mod.Mod{mod.NewFloat("LightningExposure", mod.TypeBase, 0).Tag(mod.GlobalEffect("Debuff").Name("Lightning Exposure"))},
		},
	}
	// src/Data/Skills/spectre.lua:8500
	m["AzmeriCasterDemonSpellDamageAura"] = map[string]*StatMap{
		"spell_critical_strike_chance_+%": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeIncrease, 0).Flag(mod.MFlagSpell).Tag(mod.GlobalEffect("Aura"))},
		},
		"spell_damage_aura_spell_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/spectre.lua:11058
	m["AzmeriDemonPhysicalDamageAura"] = map[string]*StatMap{
		"physical_damage_aura_nearby_enemies_physical_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeMore, 0).Tag(mod.GlobalEffect("AuraDebuff").ModCond("PrideMinEffect"))},
		},
		"physical_damage_aura_nearby_enemies_physical_damage_taken_+%_max": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeMore, 0).Tag(mod.GlobalEffect("AuraDebuff"))},
		},
	}
	// src/Data/Skills/spectre.lua:8671
	m["AzmeriDualStrikeDemonFireEnrage"] = map[string]*StatMap{
		"attack_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff").Name("Enrage"))},
		},
		"base_movement_velocity_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff").Name("Enrage"))},
		},
		"maximum_fire_damage_to_return_when_hit": {
			Mods: []mod.Mod{mod.NewFloat("FireMax", mod.TypeBase, 0)},
		},
		"minimum_fire_damage_to_return_when_hit": {
			Mods: []mod.Mod{mod.NewFloat("FireMin", mod.TypeBase, 0)},
		},
		"physical_damage_%_to_add_as_fire": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsFire", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Enrage"))},
		},
	}
	// src/Data/Skills/spectre.lua:7175
	m["AzmeriFirefuryCremation"] = map[string]*StatMap{
		"cremation_base_fires_projectile_every_x_ms": {
			Mods: []mod.Mod{skill("hitTimeOverride", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/spectre.lua:7216
	m["AzmeriFirefuryFireResistAura"] = map[string]*StatMap{
		"base_fire_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("FireResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_maximum_fire_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("FireResistMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/spectre.lua:9000
	m["AzmeriGeofriSmite"] = map[string]*StatMap{
		"maximum_added_lightning_damage_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"), mod.Condition("AffectedByVaalSmite").Neg(true))},
		},
		"minimum_added_lightning_damage_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"), mod.Condition("AffectedByVaalSmite").Neg(true))},
		},
	}
	// src/Data/Skills/spectre.lua:9208
	m["AzmeriGoddessDiscipline"] = map[string]*StatMap{
		"base_maximum_energy_shield": {
			Mods: []mod.Mod{mod.NewFloat("EnergyShield", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"energy_shield_recharge_rate_+%": {
			Mods: []mod.Mod{mod.NewFloat("EnergyShieldRecharge", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/spectre.lua:7012
	m["AzmeriHailrakeColdResistAura"] = map[string]*StatMap{
		"base_cold_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("ColdResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_maximum_cold_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("ColdResistMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/spectre.lua:7331
	m["AzmeriHydraHatred"] = map[string]*StatMap{
		"hatred_aura_cold_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ColdDamage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"physical_damage_%_to_add_as_cold": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsCold", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/spectre.lua:7567
	m["AzmeriKudukuWrath"] = map[string]*StatMap{
		"attack_maximum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"attack_minimum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"wrath_aura_spell_lightning_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("LightningDamage", mod.TypeMore, 0).Flag(mod.MFlagSpell).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/spectre.lua:9656
	m["AzmeriOakVitality"] = map[string]*StatMap{
		"base_life_regeneration_rate_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/spectre.lua:10395
	m["AzmeriOversoulRocksTriggered"] = map[string]*StatMap{
		"fire_storm_fireball_delay_ms": {
			Mods: []mod.Mod{skill("hitTimeOverride", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"firestorm_base_area_of_effect_+%": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffectPrimary", mod.TypeIncrease, 0)},
		},
	}
	// src/Data/Skills/spectre.lua:9497
	m["AzmeriPhantasmClarity"] = map[string]*StatMap{
		"base_mana_regeneration_rate_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("ManaRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/spectre.lua:9461
	m["AzmeriPhantasmExplodeSap"] = map[string]*StatMap{
		"chance_to_inflict_sapped_%": {
			Mods: []mod.Mod{mod.NewFloat("EnemySapChance", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/spectre.lua:7905
	m["AzmeriSwordStormCascade"] = map[string]*StatMap{
		"fire_storm_fireball_delay_ms": {
			Mods: []mod.Mod{skill("hitTimeOverride", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"firestorm_base_area_of_effect_+%": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffectPrimary", mod.TypeIncrease, 0)},
		},
	}
	// src/Data/Skills/spectre.lua:10150
	m["AzmeriTentacleMinionLightningResistAura"] = map[string]*StatMap{
		"base_lightning_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("LightningResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_maximum_lightning_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("LightningResistMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/spectre.lua:10293
	m["AzmeriTigerHaste"] = map[string]*StatMap{
		"attack_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_movement_velocity_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"cast_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagCast).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/spectre.lua:10357
	m["AzmeriTurtleDetermination"] = map[string]*StatMap{
		"base_physical_damage_reduction_rating": {
			Mods: []mod.Mod{mod.NewFloat("Armour", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"determination_aura_armour_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Armour", mod.TypeMore, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_int.lua:1168
	m["BallLightning"] = map[string]*StatMap{
		"ball_lightning_projectile_speed_and_hit_frequency_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileSpeed", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:1287
	m["BallLightningAltX"] = map[string]*StatMap{
		"ball_lightning_projectile_speed_and_hit_frequency_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileSpeed", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/spectre.lua:99
	m["BanditExplosiveArrowAtAnimationSpeed"] = map[string]*StatMap{
		"fuse_arrow_explosion_radius_+_per_fuse_arrow_orb": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(mod.Multiplier("ExplosiveArrowFuse"))},
		},
		"maximum_fire_damage_per_fuse_arrow_orb": {
			Mods: []mod.Mod{skill("FireMax", 0).Tag(mod.Multiplier("ExplosiveArrowFuse"))},
		},
		"minimum_fire_damage_per_fuse_arrow_orb": {
			Mods: []mod.Mod{skill("FireMin", 0).Tag(mod.Multiplier("ExplosiveArrowFuse"))},
		},
	}
	// src/Data/Skills/act_int.lua:1482
	m["Bane"] = map[string]*StatMap{
		"apply_linked_curses_with_dark_ritual": {
			Mods: []mod.Mod{},
		},
		"cannot_cast_curses": {
			Mods: []mod.Mod{},
		},
		"dark_ritual_damage_+%_final_per_curse_applied": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("CurseApplied"))},
		},
		"dark_ritual_skill_effect_duration_+%_per_curse_applied": {
			Mods: []mod.Mod{mod.NewFloat("Duration", mod.TypeIncrease, 0).Tag(mod.Multiplier("CurseApplied"))},
		},
		"display_linked_curse_effect_+%": {
			Mods: []mod.Mod{},
		},
		"display_linked_curse_effect_+%_final": {
			Mods: []mod.Mod{},
		},
		"support_bane_curse_effect_+%_final": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:1659
	m["BaneAltX"] = map[string]*StatMap{
		"apply_linked_curses_with_dark_ritual": {
			Mods: []mod.Mod{},
		},
		"cannot_cast_curses": {
			Mods: []mod.Mod{},
		},
		"dark_ritual_damage_+%_final_per_curse_applied": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("CurseApplied"))},
		},
		"dark_ritual_skill_effect_duration_+%_per_curse_applied": {
			Mods: []mod.Mod{mod.NewFloat("Duration", mod.TypeIncrease, 0).Tag(mod.Multiplier("CurseApplied"))},
		},
		"support_bane_curse_effect_+%_final": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/other.lua:173
	m["Barkskin"] = map[string]*StatMap{
		"chance_to_evade_attacks_+%_final_per_missing_barkskin_stack": {
			Mods: []mod.Mod{mod.NewFloat("EvadeChance", mod.TypeMore, 0).Tag(mod.Multiplier("MissingBarkskinStacks"), mod.GlobalEffect("Buff").Name("Barkskin"))},
		},
		"max_barkskin_stacks": {
			Mods: []mod.Mod{
				mod.NewFloat("Multiplier:BarkskinMaxStacks", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true)),
				mod.NewFloat("Multiplier:MissingBarkskinStacks", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true)),
			},
		},
		"physical_attack_damage_taken_+_per_barkskin_stack": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTakenFromAttacks", mod.TypeBase, 0).Tag(multiplierLimitVar("BarkskinStacks", "BarkskinMaxStacks"), mod.GlobalEffect("Buff").Name("Barkskin"))},
		},
	}
	// src/Data/Skills/other.lua:396
	m["BarnacleSnap"] = map[string]*StatMap{
		"barnacle_snap_base_activate_rate_ms": {
			Mods: []mod.Mod{skill("hitTimeOverride", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/act_dex.lua:1031
	m["Barrage"] = map[string]*StatMap{
		"projectiles_barrage": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:1119
	m["BarrageAltX"] = map[string]*StatMap{
		"projectiles_barrage": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:785
	m["BattlemagesCry"] = map[string]*StatMap{
		"display_battlemage_cry_exerted_attacks_trigger_supported_spell": {
			Mods: []mod.Mod{},
		},
		"divine_cry_additional_base_critical_strike_chance_per_5_power_up_to_cap": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Warcry").Div(5).Limit(25))},
			Div:  utils.Ptr(float64(100)),
		},
		"skill_empowers_next_x_melee_attacks": {
			Mods: []mod.Mod{mod.NewFloat("BattlemageExertedAttacks", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_str.lua:879
	m["BattlemagesCrySupport"] = map[string]*StatMap{
		"support_divine_cry_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
		"triggered_by_divine_cry": {
			Mods: []mod.Mod{skill("triggeredByBattleMageCry", 1)},
		},
	}
	// src/Data/Skills/act_dex.lua:1193
	m["BearTrap"] = map[string]*StatMap{
		"bear_trap_damage_taken_+%_from_traps_and_mines": {
			Mods: []mod.Mod{mod.NewFloat("TrapMineDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Debuff"))},
		},
	}
	// src/Data/Skills/act_str.lua:943
	m["Berserk"] = map[string]*StatMap{
		"berserk_rage_effect_+%": {
			Mods: []mod.Mod{mod.NewFloat("RageEffect", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:1472
	m["BladeBlastAltX"] = map[string]*StatMap{
		"blade_burst_area_of_effect_+%_final_per_blade_vortex_blade_detonated": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.Multiplier("BladeBlastofUnloadingStage"))},
		},
		"blade_vortex_damage_+%_per_blade_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("BladeBlastofUnloadingStage"))},
		},
		"damage_per_blade_vortex_blade_description_mode": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:1560
	m["BladeBlastAltY"] = map[string]*StatMap{
		"gain_%_of_base_dagger_damage_as_added_spell_damage": {
			Mods: []mod.Mod{skill("gainPercentBaseDaggerDamageToSpells", 0)},
		},
	}
	// src/Data/Skills/act_dex.lua:1917
	m["BladeFlurry"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"blade_flurry_damage_+%_final_while_at_max_stages": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.MultiplierThreshold("BladeFlurryStage").ThresholdVar("BladeFlurryMaxStages"))},
		},
		"charged_attack_damage_per_stack_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("BladeFlurryStage"))},
		},
		"display_max_charged_attack_stats": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:BladeFlurryMaxStages", mod.TypeBase, 0)},
		},
		"quality_display_charged_attack_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:2034
	m["BladeFlurryAltX"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"blade_flurry_critical_strike_chance_per_stage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeMore, 0).Tag(mod.Multiplier("BladeFlurryofIncisionStage"))},
		},
		"blade_flurry_critical_strike_multiplier_+_while_at_max_stages": {
			Mods: []mod.Mod{mod.NewFloat("CritMultiplier", mod.TypeBase, 0).Tag(mod.MultiplierThreshold("BladeFlurryofIncisionStage").ThresholdVar("BladeFlurryofIncisionMaxStages"))},
		},
		"display_max_charged_attack_stats": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:BladeFlurryofIncisionMaxStages", mod.TypeBase, 0)},
		},
		"quality_display_charged_attack_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:1806
	m["BladeTrapAltY"] = map[string]*StatMap{
		"quality_display_active_skill_bleed_damage_final_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:2223
	m["BladeVortex"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"blade_vortex_ailment_damage_+%_per_blade_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("BladeVortexBlade"))},
		},
		"blade_vortex_damage_+%_per_blade_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("BladeVortexBlade"))},
		},
		"blade_vortex_hit_rate_+%_per_blade": {
			Mods: []mod.Mod{skill("hitFrequencyPerBlade", 0)},
			Div:  utils.Ptr(float64(100)),
		},
		"maximum_number_of_spinning_blades": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:BladeVortexMaxStages", mod.TypeBase, 0)},
		},
		"quality_display_blade_vortex_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:2334
	m["BladeVortexAltX"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"quality_display_blade_vortex_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:2530
	m["Bladefall"] = map[string]*StatMap{
		"quality_display_bladefall_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:2617
	m["BladefallAltX"] = map[string]*StatMap{
		"bladefall_critical_strike_chance_+%_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeIncrease, 0).Tag(mod.Multiplier("BladefallofVolleysStage"))},
		},
		"bladefall_damage_per_stage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("BladefallofVolleysStage"))},
		},
		"bladefall_number_of_volleys": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:BladefallofVolleysMaxStages", mod.TypeBase, 0)},
		},
		"quality_display_bladefall_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:2706
	m["BladefallAltY"] = map[string]*StatMap{
		"quality_display_bladefall_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:2790
	m["BladefallAltZ"] = map[string]*StatMap{
		"bladefall_base_volley_frequency_ms": {
			Mods: []mod.Mod{skill("hitFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"bladefall_volley_frequency_+%_per_100_maximum_mana": {
			Mods: []mod.Mod{skill("incVolleyFrequency", 0).Tag(mod.PerStat(100, "Mana"))},
		},
		"quality_display_bladefall_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:1030
	m["Bladestorm"] = map[string]*StatMap{
		"bladestorm_attack_speed_+%_final_while_in_bloodstorm": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.Condition("BladestormInBloodstorm"))},
		},
		"bladestorm_movement_speed_+%_while_in_sandstorm": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff").Name("Sandstorm").EffectCond("BladestormInSandstorm"))},
		},
		"bladestorm_storm_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_str.lua:1138
	m["BladestormAltX"] = map[string]*StatMap{
		"bladestorm_attack_speed_+%_final_while_in_bloodstorm": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.Condition("BladestormInBloodstorm"))},
		},
		"bladestorm_movement_speed_+%_while_in_sandstorm": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff").Name("Sandstorm").EffectCond("BladestormInSandstorm"))},
		},
		"bladestorm_storm_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_int.lua:1916
	m["Blight"] = map[string]*StatMap{
		"display_max_blight_stacks": {
			Mods: []mod.Mod{
				mod.NewFloat("Multiplier:BlightMaxStages", mod.TypeBase, 0).Tag(mod.SkillPart(1)),
				mod.NewFloat("BlightBaseMaxStages", mod.TypeBase, 0).Tag(mod.SkillPart(2)),
			},
		},
	}
	// src/Data/Skills/act_int.lua:2012
	m["BlightAltX"] = map[string]*StatMap{
		"display_max_blight_stacks": {
			Mods: []mod.Mod{
				mod.NewFloat("Multiplier:BlightofContagionMaxStages", mod.TypeBase, 0).Tag(mod.SkillPart(1)),
				mod.NewFloat("BlightBaseMaxStages", mod.TypeBase, 0).Tag(mod.SkillPart(2)),
			},
		},
	}
	// src/Data/Skills/act_int.lua:2108
	m["BlightAltY"] = map[string]*StatMap{
		"display_max_blight_stacks": {
			Mods: []mod.Mod{
				mod.NewFloat("Multiplier:BlightofAtrophyMaxStages", mod.TypeBase, 0).Tag(mod.SkillPart(1)),
				mod.NewFloat("BlightBaseMaxStages", mod.TypeBase, 0).Tag(mod.SkillPart(2)),
			},
		},
	}
	// src/Data/Skills/act_str.lua:1235
	m["BloodAndSand"] = map[string]*StatMap{
		"blood_sand_stance_melee_skills_area_damage_+%_final_in_blood_stance": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagArea).Tag(mod.GlobalEffect("Buff").Name("Blood").EffectCond("BloodStance"), mod.SkillType("Melee"))},
		},
		"blood_sand_stance_melee_skills_area_damage_+%_final_in_sand_stance": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagArea).Tag(mod.GlobalEffect("Buff").Name("Sand").EffectCond("SandStance"), mod.SkillType("Melee"))},
		},
		"blood_sand_stance_melee_skills_area_of_effect_+%_final_in_blood_stance": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff").Name("Blood").EffectCond("BloodStance"), mod.SkillType("Melee"))},
		},
		"blood_sand_stance_melee_skills_area_of_effect_+%_final_in_sand_stance": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff").Name("Sand").EffectCond("SandStance"), mod.SkillType("Melee"))},
		},
	}
	// src/Data/Skills/other.lua:488
	m["BloodOffering"] = map[string]*StatMap{
		"blood_offering_%_of_life_to_lose": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:BloodOfferingLifeSacrificed", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.PerStat(1, "LifeUnreserved").Actor("parent"))},
			Div:  utils.Ptr(float64(100)),
		},
		"blood_offering_%_of_lost_life_to_regenerate_as_life_per_second": {
			Mods: []mod.Mod{
				mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"), mod.Multiplier("BloodOfferingLifeSacrificed")),
				mod.NewFloat("Multiplier:BloodOfferingBaseRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("BloodOfferingLifeSacrificed")),
			},
			Div: utils.Ptr(float64(100)),
		},
		"blood_offering_life_regenerated_+%_final_per_corpse": {
			Mods: []mod.Mod{
				mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"), mod.Multiplier("BloodOfferingBaseRegen"), mod.Multiplier("CorpseConsumedRecently").Limit(5)),
				mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"), mod.Multiplier("BloodOfferingBaseRegen"), mod.Multiplier("CorpseConsumedRecently").Limit(5).Actor("parent")),
			},
			Div: utils.Ptr(float64(100)),
		},
		"damage_+%": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_dex.lua:3314
	m["BloodRage"] = map[string]*StatMap{
		"attack_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Buff"))},
		},
		"base_physical_damage_%_of_maximum_energy_shield_to_deal_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDegen", mod.TypeBase, 0).Tag(mod.PerStat(1, "EnergyShield"), mod.GlobalEffect("Buff"))},
			Div:  utils.Ptr(float64(6000)),
		},
		"base_physical_damage_%_of_maximum_life_to_deal_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDegen", mod.TypeBase, 0).Tag(mod.PerStat(1, "Life"), mod.GlobalEffect("Buff"))},
			Div:  utils.Ptr(float64(6000)),
		},
		"life_leech_from_physical_attack_damage_permyriad": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageLifeLeech", mod.TypeBase, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Buff"))},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/other.lua:550
	m["BloodSacramentUnique"] = map[string]*StatMap{
		"flameblast_ailment_damage_+%_final_per_10_life_reserved": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagAilment).Tag(mod.Multiplier("ChannelledLifeReservedPerStage").Div(10))},
		},
		"flameblast_damage_+%_final_per_10_life_reserved": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("ChannelledLifeReservedPerStage").Div(10))},
		},
		"flameblast_hundred_times_radius_+_per_1%_life_reserved": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(mod.Multiplier("ChannelledLifeReservedPercentPerStage"))},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/act_int.lua:2310
	m["Bodyswap"] = map[string]*StatMap{
		"corpse_warp_area_of_effect_+%_final_when_consuming_corpse": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"spell_maximum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMax", 0).Tag(mod.SkillPart(1))},
		},
		"spell_minimum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMin", 0).Tag(mod.SkillPart(1))},
		},
	}
	// src/Data/Skills/act_int.lua:2425
	m["BodyswapAltX"] = map[string]*StatMap{
		"corpse_warp_area_of_effect_+%_final_when_consuming_minion": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"skill_minion_explosion_life_%": {
			Mods: []mod.Mod{skill("selfFireExplosionLifeMultiplier", 0).Tag(mod.SkillPart(2))},
			Div:  utils.Ptr(float64(100)),
		},
		"spell_base_fire_damage_%_maximum_life": {
			Mods: []mod.Mod{skill("selfFireExplosionLifeMultiplier", 0).Tag(mod.SkillPart(1))},
			Div:  utils.Ptr(float64(100)),
		},
		"spell_maximum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMax", 0).Tag(mod.SkillPart(1))},
		},
		"spell_minimum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMin", 0).Tag(mod.SkillPart(1))},
		},
	}
	// src/Data/Skills/other.lua:602
	m["BoneArmour"] = map[string]*StatMap{
		"quick_guard_damage_absorb_limit": {
			Mods: []mod.Mod{mod.NewFloat("GuardAbsorbLimit", mod.TypeBase, 0).Tag(mod.GlobalEffect("Guard").Unscalable(true))},
		},
		"quick_guard_damage_absorbed_%": {
			Mods: []mod.Mod{mod.NewFloat("GuardAbsorbRate", mod.TypeBase, 0).Tag(mod.GlobalEffect("Guard").Unscalable(true))},
		},
	}
	// src/Data/Skills/minion.lua:1472
	m["BoneGolemMultiAttack"] = map[string]*StatMap{
		"bone_golem_attack_speed_+%_final_after_third_combo_index": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAttack)},
			Div:  utils.Ptr(float64(1.5)),
		},
		"bone_golem_damage_+%_final_after_third_combo_index": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
			Div:  utils.Ptr(float64(1.5)),
		},
	}
	// src/Data/Skills/act_int.lua:2523
	m["BoneOffering"] = map[string]*StatMap{
		"base_spell_block_%": {
			Mods: []mod.Mod{mod.NewFloat("SpellBlockChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"monster_base_block_%": {
			Mods: []mod.Mod{mod.NewFloat("BlockChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_str.lua:1326
	m["Boneshatter"] = map[string]*StatMap{
		"boneshatter_trauma_base_duration_ms": {
			Mods: []mod.Mod{mod.NewFloat("TraumaDuration", mod.TypeBase, 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"quality_display_boneshatter_is_gem": {
			Mods: []mod.Mod{},
		},
		"trauma_strike_damage_+%_final_per_trauma": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("TraumaStacks"))},
		},
	}
	// src/Data/Skills/act_str.lua:1436
	m["BoneshatterAltX"] = map[string]*StatMap{
		"boneshatter_trauma_base_duration_ms": {
			Mods: []mod.Mod{mod.NewFloat("TraumaDuration", mod.TypeBase, 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"lose_all_trauma_at_X_trauma": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:TraumaStacksMax", mod.TypeBase, 0)},
		},
		"quality_display_boneshatter_is_gem": {
			Mods: []mod.Mod{},
		},
		"trauma_strike_damage_+%_final_per_trauma_capped": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(multiplierLimitVar("TraumaStacks", "TraumaStacksMax"))},
		},
	}
	// src/Data/Skills/act_int.lua:2603
	m["BrandRecall"] = map[string]*StatMap{
		"recall_sigil_target_search_range_+%": {
			Mods: []mod.Mod{mod.NewFloat("BrandAttachmentRange", mod.TypeIncrease, 0).Tag(mod.Condition("CannotRecallBrand").Neg(true), mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/spectre.lua:6527
	m["BreachBlizzardSpectre"] = map[string]*StatMap{
		"avoid_damage_%": {
			Mods: []mod.Mod{
				mod.NewFloat("AvoidPhysicalDamageChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Snow Cloak")),
				mod.NewFloat("AvoidLightningDamageChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Snow Cloak")),
				mod.NewFloat("AvoidColdDamageChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Snow Cloak")),
				mod.NewFloat("AvoidFireDamageChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Snow Cloak")),
				mod.NewFloat("AvoidChaosDamageChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Snow Cloak")),
			},
		},
	}
	// src/Data/Skills/spectre.lua:285
	m["BreachCleave"] = map[string]*StatMap{
		"active_skill_merged_damage_+%_final_while_dual_wielding": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Condition("DualWielding"))},
		},
	}
	// src/Data/Skills/other.lua:5418
	m["BreachHandTrap"] = map[string]*StatMap{
		"active_skill_hit_damage_+%_final_per_curse_on_target": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("CurseOnEnemy"))},
		},
		"gain_%_of_phys_as_extra_chaos_per_white_socket_on_gloves": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsChaos", mod.TypeBase, 0).Tag(mod.Multiplier("WhiteSocketInGloves"))},
		},
		"gain_%_of_phys_as_extra_cold_per_green_socket_on_gloves": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsCold", mod.TypeBase, 0).Tag(mod.Multiplier("GreenSocketInGloves"))},
		},
		"gain_%_of_phys_as_extra_fire_per_red_socket_on_gloves": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsFire", mod.TypeBase, 0).Tag(mod.Multiplier("RedSocketInGloves"))},
		},
		"gain_%_of_phys_as_extra_lightning_per_blue_socket_on_gloves": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsLightning", mod.TypeBase, 0).Tag(mod.Multiplier("BlueSocketInGloves"))},
		},
	}
	// src/Data/Skills/act_dex.lua:3477
	m["BurningArrowAltX"] = map[string]*StatMap{
		"added_fire_damage_to_attacks_equal_to_%_maximum_life": {
			Mods: []mod.Mod{
				mod.NewFloat("FireMin", mod.TypeBase, 0).Flag(mod.MFlagAttack).Tag(mod.PercentStat("Life", 1)),
				mod.NewFloat("FireMax", mod.TypeBase, 0).Flag(mod.MFlagAttack).Tag(mod.PercentStat("Life", 1)),
			},
		},
	}
	// src/Data/Skills/other.lua:2352
	m["CatsStealthTriggeredIntimidatingCry"] = map[string]*StatMap{
		"intimidating_cry_movement_speed_+%_per_5_power_up_to_cap": {
			Mods: []mod.Mod{mod.NewFloat("IntimidatingMovementSpeedPer5MP", mod.TypeBase, 0)},
		},
		"skill_empowers_next_x_melee_attacks": {
			Mods: []mod.Mod{mod.NewFloat("IntimidatingExertedAttacks", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_dex.lua:3632
	m["CausticArrow"] = map[string]*StatMap{
		"caustic_arrow_explode_on_hit_base_area_of_effect_radius": {
			Mods: []mod.Mod{skill("radius", 0)},
		},
	}
	// src/Data/Skills/act_dex.lua:3722
	m["CausticArrowAltX"] = map[string]*StatMap{
		"caustic_arrow_explode_on_hit_base_area_of_effect_radius": {
			Mods: []mod.Mod{skill("radius", 0)},
		},
	}
	// src/Data/Skills/act_str.lua:1626
	m["ChainHook"] = map[string]*StatMap{
		"chain_strike_cone_radius_+_per_x_rage": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeBase, 0).Tag(mod.Multiplier("Rage").Div(5))},
			Div:  utils.Ptr(float64(5)),
		},
		"quality_display_chain_hook_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:1711
	m["ChainHookAltY"] = map[string]*StatMap{
		"chain_hook_attachment_rage_to_gain_per_hit": {
			Mods: []mod.Mod{mod.NewFlag("Condition:CanGainRage", true).Tag(mod.GlobalEffect("Buff").Name("Rage"))},
		},
		"chain_hook_attachment_range": {
			Mods: []mod.Mod{skill("radiusSecondary", 0)},
		},
		"chain_strike_cone_radius_+_per_x_rage": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeBase, 0).Tag(mod.Multiplier("Rage").Div(5))},
			Div:  utils.Ptr(float64(5)),
		},
	}
	// src/Data/Skills/act_dex.lua:3945
	m["ChargedDash"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"charged_dash_channelling_damage_at_full_stacks_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"charged_dash_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("chargedDashFinalDamageModifier", mod.TypeIncrease, 0).Tag(mod.SkillPart(3))},
		},
		"charged_dash_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("chargedDashFinalDamageModifier", mod.TypeIncrease, 0).Tag(mod.Multiplier("ChargedDashStage"), mod.SkillPart(3))},
		},
	}
	// src/Data/Skills/act_dex.lua:4070
	m["ChargedDashAltX"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"charged_dash_channelling_damage_at_full_stacks_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"charged_dash_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("chargedDashFinalDamageModifier", mod.TypeIncrease, 0).Tag(mod.SkillPart(3))},
		},
		"charged_dash_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("chargedDashFinalDamageModifier", mod.TypeIncrease, 0).Tag(mod.Multiplier("ChargedDashofProjectionStage"), mod.SkillPart(3))},
		},
	}
	// src/Data/Skills/act_int.lua:2662
	m["Clarity"] = map[string]*StatMap{
		"base_mana_regeneration_rate_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("ManaRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/act_str.lua:1804
	m["Cleave"] = map[string]*StatMap{
		"cleave_+1_base_radius_per_nearby_enemy_up_to_10": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeBase, 0).Tag(mod.Multiplier("NearbyEnemies").Limit(10).LimitTotal(true))},
		},
	}
	// src/Data/Skills/act_str.lua:1888
	m["CleaveAltX"] = map[string]*StatMap{
		"chain_strike_cone_radius_+_per_x_rage": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeBase, 0).Tag(mod.Multiplier("Rage").Div(5))},
			Div:  utils.Ptr(float64(5)),
		},
		"quality_display_chain_hook_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:4168
	m["CobraLash"] = map[string]*StatMap{
		"cobra_lash_hit_and_ailment_damage_+%_final_for_each_remaining_chain": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit|mod.KeywordFlagAilment).Tag(mod.PerStat(1, "ChainRemaining"), mod.SkillType("Projectile"))},
		},
	}
	// src/Data/Skills/other.lua:797
	m["ColdAegis"] = map[string]*StatMap{
		"aegis_unique_shield_max_value": {
			Mods: []mod.Mod{mod.NewFloat("ColdAegisValue", mod.TypeMAX, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_str.lua:6969
	m["Combust"] = map[string]*StatMap{
		"triggered_by_infernal_cry": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:3077
	m["Conductivity"] = map[string]*StatMap{
		"base_lightning_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("LightningResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"base_self_shock_duration_-%": {
			Mods: []mod.Mod{mod.NewFloat("SelfShockDuration", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
			Div:  utils.Ptr(float64(-1)),
		},
		"chance_to_be_shocked_%": {
			Mods: []mod.Mod{mod.NewFloat("SelfShockChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_dex.lua:4257
	m["Conflagration"] = map[string]*StatMap{
		"base_fire_damage_to_deal_per_minute": {
			Mods: []mod.Mod{skill("FireDot", 0).Tag(mod.SkillPart(2))},
			Div:  utils.Ptr(float64(60)),
		},
		"napalm_arrow_detonation_hit_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_str.lua:2085
	m["ConsecratedPath"] = map[string]*StatMap{
		"groundslam_damage_to_close_targets_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Condition("AtCloseRange"))},
		},
		"quality_display_groundslam_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:2179
	m["ConsecratedPathAltX"] = map[string]*StatMap{
		"groundslam_damage_to_close_targets_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Condition("AtCloseRange"))},
		},
		"quality_display_groundslam_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/other.lua:915
	m["Convocation"] = map[string]*StatMap{
		"base_life_regeneration_rate_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/act_int.lua:3465
	m["CracklingLance"] = map[string]*StatMap{
		"disintegrate_base_radius_+_per_intensify": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(multiplierLimitVar("Intensity", "IntensityLimit"))},
		},
		"disintegrate_damage_+%_final_per_intensity": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(multiplierLimitVar("Intensity", "IntensityLimit"))},
		},
		"quality_display_disintegrate_is_gem": {
			Mods: []mod.Mod{},
		},
	}
}

func skillStatMapsPart1(m map[string]map[string]*StatMap) {
	// src/Data/Skills/act_dex.lua:4371
	m["Cremation"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"cremation_base_fires_projectile_every_x_ms": {
			Mods: []mod.Mod{skill("cremationFireRate", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"cremation_fires_projectiles_faster_+%_final": {
			Mods: []mod.Mod{skill("cremationFireRateIncrease", 0)},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/act_dex.lua:4489
	m["CremationAltX"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"cremation_base_fires_projectile_every_x_ms": {
			Mods: []mod.Mod{skill("cremationFireRate", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"cremation_fires_projectiles_faster_+%_final": {
			Mods: []mod.Mod{skill("cremationFireRateIncrease", 0)},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/act_dex.lua:4593
	m["CremationAltY"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"cremation_base_fires_projectile_every_x_ms": {
			Mods: []mod.Mod{skill("cremationFireRate", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"cremation_fires_projectiles_faster_+%_final": {
			Mods: []mod.Mod{skill("cremationFireRateIncrease", 0)},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/spectre.lua:6764
	m["CrucibleIceStormTrap"] = map[string]*StatMap{
		"fire_storm_fireball_delay_ms": {
			Mods: []mod.Mod{skill("hitTimeOverride", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"firestorm_base_area_of_effect_+%": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffectPrimary", mod.TypeIncrease, 0)},
		},
	}
	// src/Data/Skills/act_dex.lua:4715
	m["Cyclone"] = map[string]*StatMap{
		"cyclone_area_of_effect_+%_per_additional_melee_range": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0).Tag(mod.Multiplier("AdditionalMeleeRange"))},
		},
		"cyclone_movement_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeMore, 0).Tag(mod.Condition("ChannellingCyclone"), mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:4821
	m["CycloneAltX"] = map[string]*StatMap{
		"cyclone_area_of_effect_+%_per_additional_melee_range": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0).Tag(mod.Multiplier("AdditionalMeleeRange"))},
		},
		"cyclone_attack_speed_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.Multiplier("CycloneofTumultStage"))},
		},
		"cyclone_max_number_of_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:CycloneofTumultMaxStages", mod.TypeBase, 0)},
		},
		"cyclone_melee_weapon_range_+_per_stage": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(mod.Multiplier("CycloneofTumultStage"))},
		},
	}
	// src/Data/Skills/spectre.lua:6244
	m["DTTHellscapeStabWeb"] = map[string]*StatMap{
		"action_speed_-%": {
			Mods: []mod.Mod{mod.NewFloat("ActionSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Debuff").Name("Thunder Web"))},
		},
		"base_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Debuff").Name("Thunder Web"))},
		},
	}
	// src/Data/Skills/act_int.lua:3904
	m["DarkPact"] = map[string]*StatMap{
		"quality_display_dark_pact_is_gem": {
			Mods: []mod.Mod{},
		},
		"skeletal_chains_aoe_%_health_dealt_as_chaos_damage": {
			Mods: []mod.Mod{skill("lifeDealtAsChaos", 0)},
		},
		"skeletal_chains_no_minions_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.SkillPart(1))},
		},
		"skeletal_chains_no_minions_radius_+": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(mod.SkillPart(1))},
		},
	}
	// src/Data/Skills/act_int.lua:4023
	m["DarkPactAltX"] = map[string]*StatMap{
		"dark_pact_chance_to_gain_an_additional_ruin_%": {
			Mods: []mod.Mod{skill("additionalRuinChance", 0)},
		},
		"dark_pact_chaos_damage_added_from_sacrifice_+%": {
			Mods: []mod.Mod{skill("percentageSacrificedDealtAsChaos", 0)},
		},
		"dark_pact_sacrifice_%_life_from_max_ruin": {
			Mods: []mod.Mod{skill("percentageLifeSacrificed", 0)},
		},
	}
	// src/Data/Skills/other.lua:1161
	m["DeathWish"] = map[string]*StatMap{
		"death_wish_attack_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Buff"))},
		},
		"death_wish_cast_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagCast).Tag(mod.GlobalEffect("Buff"))},
		},
		"death_wish_hit_and_ailment_damage_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit|mod.KeywordFlagAilment).Tag(mod.Multiplier("DeathWishStage"), mod.SkillPart(2))},
		},
		"death_wish_max_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:DeathWishMaxStages", mod.TypeBase, 0).Tag(mod.SkillPart(2))},
		},
		"death_wish_movement_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"spell_maximum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMax", 0).Tag(mod.SkillPart(2))},
		},
		"spell_minimum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMin", 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_str.lua:2500
	m["DefianceBanner"] = map[string]*StatMap{
		"base_movement_velocity_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_spell_suppression_chance_%": {
			Mods: []mod.Mod{mod.NewFloat("SpellSuppressionChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/spectre.lua:6678
	m["DelveMonsterEnfeebleOnHit"] = map[string]*StatMap{
		"accuracy_rating_+%": {
			Mods: []mod.Mod{mod.NewFloat("Accuracy", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"enfeeble_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique").Neg(true))},
		},
		"enfeeble_damage_+%_vs_rare_or_unique_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique"))},
		},
	}
	// src/Data/Skills/act_int.lua:4110
	m["Despair"] = map[string]*StatMap{
		"base_chaos_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("ChaosResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"maximum_added_chaos_damage_taken": {
			Mods: []mod.Mod{mod.NewFloat("SelfChaosMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"minimum_added_chaos_damage_taken": {
			Mods: []mod.Mod{mod.NewFloat("SelfChaosMin", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_int.lua:4196
	m["DestructiveLink"] = map[string]*StatMap{
		"critical_link_grants_accuracy_rating_+%": {
			Mods: []mod.Mod{mod.NewFloat("Accuracy", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Link"))},
		},
		"critical_link_grants_base_critical_strike_multiplier_+": {
			Mods: []mod.Mod{mod.NewFloat("CritMultiplier", mod.TypeBase, 0).Tag(mod.GlobalEffect("Link"))},
		},
		"display_critical_link_overrides_main_hand_critical_strike_chance": {
			Mods: []mod.Mod{mod.NewFlag("MainHandCritIsEqualToParent", true).Tag(mod.GlobalEffect("Link"), mod.Condition("MainHandAttack"))},
		},
	}
	// src/Data/Skills/act_str.lua:2585
	m["Determination"] = map[string]*StatMap{
		"base_physical_damage_reduction_rating": {
			Mods: []mod.Mod{mod.NewFloat("Armour", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"determination_aura_armour_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Armour", mod.TypeMore, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_dex.lua:5180
	m["DetonateDead"] = map[string]*StatMap{
		"spell_maximum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMax", 0).Tag(mod.SkillPart(1))},
		},
		"spell_minimum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMin", 0).Tag(mod.SkillPart(1))},
		},
	}
	// src/Data/Skills/act_dex.lua:5348
	m["DetonateDeadAltY"] = map[string]*StatMap{
		"spell_maximum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMax", 0).Tag(mod.SkillPart(1))},
		},
		"spell_minimum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMin", 0).Tag(mod.SkillPart(1))},
		},
	}
	// src/Data/Skills/act_str.lua:2667
	m["DevouringTotem"] = map[string]*StatMap{
		"corpse_consumption_life_to_gain": {
			Mods: []mod.Mod{skill("lifeLeechPerUse", 0)},
		},
		"corpse_consumption_mana_to_gain": {
			Mods: []mod.Mod{skill("manaLeechPerUse", 0)},
		},
	}
	// src/Data/Skills/act_int.lua:4439
	m["Discipline"] = map[string]*StatMap{
		"base_maximum_energy_shield": {
			Mods: []mod.Mod{mod.NewFloat("EnergyShield", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"energy_shield_recharge_rate_+%": {
			Mods: []mod.Mod{mod.NewFloat("EnergyShieldRecharge", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_str.lua:2756
	m["DivineBlast"] = map[string]*StatMap{
		"discus_slam_damage_+%_final_for_shockwave": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_int.lua:4619
	m["DivineIre"] = map[string]*StatMap{
		"divine_tempest_ailment_damage_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("DivineIreStageAfterFirst"))},
		},
		"divine_tempest_damage_+%_final_while_channelling": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(1))},
		},
		"divine_tempest_hit_damage_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("DivineIreStageAfterFirst"))},
		},
	}
	// src/Data/Skills/act_int.lua:4729
	m["DivineIreAltX"] = map[string]*StatMap{
		"divine_tempest_ailment_damage_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("DivineIreofHolyLightningStageAfterFirst"))},
		},
		"divine_tempest_hit_damage_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("DivineIreofHolyLightningStageAfterFirst"))},
		},
		"divine_tempest_no_beam": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:4826
	m["DivineIreAltY"] = map[string]*StatMap{
		"divine_tempest_ailment_damage_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("DivineIreofDisintegrationStageAfterFirst"))},
		},
		"divine_tempest_hit_damage_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("DivineIreofDisintegrationStageAfterFirst"))},
		},
	}
	// src/Data/Skills/act_dex.lua:5705
	m["DoubleStrikeAltY"] = map[string]*StatMap{
		"double_strike_attack_speed_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.Multiplier("DoubleStrikeofMomentumStage"))},
		},
		"double_strike_max_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:DoubleStrikeofMomentumMaxStages", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_str.lua:3205
	m["DreadBanner"] = map[string]*StatMap{
		"additional_block_%": {
			Mods: []mod.Mod{mod.NewFloat("BlockChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"recover_X_life_on_block": {
			Mods: []mod.Mod{mod.NewFloat("LifeOnBlock", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/minion.lua:1437
	m["DropBearSummonedRallyingCry"] = map[string]*StatMap{
		"damage_+%": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_dex.lua:5880
	m["DualStrike"] = map[string]*StatMap{
		"dual_strike_critical_strike_chance_+%_final_against_enemies_on_full_life": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeMore, 0).Tag(mod.ActorCondition("enemy", "FullLife"))},
		},
		"dual_strike_damage_+%_final_against_enemies_on_full_life": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "FullLife"))},
		},
		"quality_display_dual_strike_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:5968
	m["DualStrikeAltX"] = map[string]*StatMap{
		"dual_strike_off_hand_weapon_determines_attack_time": {
			Mods: []mod.Mod{mod.NewFlag("UseOffhandAttackSpeed", true)},
		},
	}
	// src/Data/Skills/act_str.lua:3312
	m["Earthquake"] = map[string]*StatMap{
		"quake_slam_fully_charged_explosion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.SkillPart(2))},
		},
		"quality_display_earthquake_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:3414
	m["EarthquakeAltX"] = map[string]*StatMap{
		"earthquake_skill_aftershock_ailment_damage_+%_final_per_100ms_duration": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.SkillPart(2), mod.Multiplier("100msEarthquakeDuration"))},
		},
		"earthquake_skill_aftershock_area_of_effect_+%_final_per_100ms_duration": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.SkillPart(2), mod.Multiplier("100msEarthquakeDuration"))},
		},
		"earthquake_skill_aftershock_hit_damage_+%_final_per_100ms_duration": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit).Tag(mod.SkillPart(2), mod.Multiplier("100msEarthquakeDuration"))},
		},
	}
	// src/Data/Skills/act_str.lua:3625
	m["Earthshatter"] = map[string]*StatMap{
		"active_skill_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0)},
		},
		"quality_display_earthshatter_spike_damage_is_gem": {
			Mods: []mod.Mod{},
		},
		"spike_slam_explosion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"spike_slam_spike_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_str.lua:3736
	m["EarthshatterAltX"] = map[string]*StatMap{
		"quality_display_earthshatter_spike_damage_is_gem": {
			Mods: []mod.Mod{},
		},
		"spike_slam_explosion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"spike_slam_spike_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_str.lua:3844
	m["EarthshatterAltY"] = map[string]*StatMap{
		"earthshatter_spike_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"quality_display_earthshatter_spike_damage_is_gem": {
			Mods: []mod.Mod{},
		},
		"spike_slam_explosion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/other.lua:1445
	m["ElementalAegis"] = map[string]*StatMap{
		"aegis_unique_shield_max_value": {
			Mods: []mod.Mod{mod.NewFloat("ElementalAegisValue", mod.TypeMAX, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:6080
	m["ElementalHit"] = map[string]*StatMap{
		"active_skill_base_area_of_effect_radius": {
			Mods: []mod.Mod{skill("radius", 0).Tag(mod.SkillPartList(2, 4, 6))},
		},
		"attack_maximum_added_cold_damage_for_elemental_hit": {
			Mods: []mod.Mod{mod.NewFloat("ColdMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
		"attack_maximum_added_fire_damage_for_elemental_hit": {
			Mods: []mod.Mod{mod.NewFloat("FireMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
		"attack_maximum_added_lightning_damage_for_elemental_hit": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
		"attack_minimum_added_cold_damage_for_elemental_hit": {
			Mods: []mod.Mod{mod.NewFloat("ColdMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
		"attack_minimum_added_fire_damage_for_elemental_hit": {
			Mods: []mod.Mod{mod.NewFloat("FireMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
		"attack_minimum_added_lightning_damage_for_elemental_hit": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
		"deal_no_non_elemental_damage": {
			Mods: []mod.Mod{},
		},
		"elemental_hit_area_of_effect_+100%_final_vs_enemy_with_associated_ailment": {
			Mods: []mod.Mod{},
		},
		"elemental_hit_damage_+%_final_per_enemy_elemental_ailment": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("ElementalHitAilmentOnEnemy"))},
		},
		"elemental_hit_no_damage_of_unchosen_elemental_type": {
			Mods: []mod.Mod{},
		},
		"elemental_hit_no_physical_chaos_damage": {
			Mods: []mod.Mod{
				mod.NewFlag("DealNoPhysical", true),
				mod.NewFlag("DealNoChaos", true),
				mod.NewFlag("DealNoFire", true).Tag(mod.SkillPartList(3, 4)),
				mod.NewFlag("DealNoFire", true).Tag(mod.SkillPartList(5, 6)),
				mod.NewFlag("DealNoCold", true).Tag(mod.SkillPartList(1, 2)),
				mod.NewFlag("DealNoCold", true).Tag(mod.SkillPartList(5, 6)),
				mod.NewFlag("DealNoLightning", true).Tag(mod.SkillPartList(1, 2)),
				mod.NewFlag("DealNoLightning", true).Tag(mod.SkillPartList(3, 4)),
			},
		},
		"quality_display_elemental_hit_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:6236
	m["ElementalHitAltX"] = map[string]*StatMap{
		"active_skill_base_area_of_effect_radius": {
			Mods: []mod.Mod{skill("radius", 0).Tag(mod.SkillPart(2))},
		},
		"attack_maximum_added_cold_damage_for_elemental_hit": {
			Mods: []mod.Mod{mod.NewFloat("ColdMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
		"attack_maximum_added_fire_damage_for_elemental_hit": {
			Mods: []mod.Mod{mod.NewFloat("FireMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
		"attack_maximum_added_lightning_damage_for_elemental_hit": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
		"attack_minimum_added_cold_damage_for_elemental_hit": {
			Mods: []mod.Mod{mod.NewFloat("ColdMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
		"attack_minimum_added_fire_damage_for_elemental_hit": {
			Mods: []mod.Mod{mod.NewFloat("FireMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
		"attack_minimum_added_lightning_damage_for_elemental_hit": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
		"deal_no_non_elemental_damage": {
			Mods: []mod.Mod{},
		},
		"elemental_hit_damage_+%_final_per_enemy_elemental_ailment": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("ElementalHitAilmentOnEnemy"))},
		},
		"elemental_hit_no_physical_chaos_damage": {
			Mods: []mod.Mod{
				mod.NewFlag("DealNoPhysical", true),
				mod.NewFlag("DealNoChaos", true),
			},
		},
		"quality_display_elemental_hit_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:4996
	m["ElementalWeakness"] = map[string]*StatMap{
		"base_resist_all_elements_%": {
			Mods: []mod.Mod{mod.NewFloat("ElementalResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"self_elemental_status_duration_-%": {
			Mods: []mod.Mod{mod.NewFloat("SelfElementalAilmentDuration", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
			Mult: utils.Ptr(float64(-1)),
		},
	}
	// src/Data/Skills/spectre.lua:11477
	m["EmpoweringHowlSpectre"] = map[string]*StatMap{
		"empowering_cry_physical_damage_added_as_fire_damage_%": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsFire", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Empowering Howl"))},
		},
	}
	// src/Data/Skills/spectre.lua:6485
	m["EmptyActionSpellWarlordGrandmaster"] = map[string]*StatMap{
		"attack_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Buff").Name("Arena Master's Presence"))},
		},
		"auras_grant_damage_+%_to_you_and_your_allies": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff").Name("Arena Master's Presence"))},
		},
		"base_movement_velocity_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff").Name("Arena Master's Presence"))},
		},
		"cast_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagCast).Tag(mod.GlobalEffect("Buff").Name("Arena Master's Presence"))},
		},
	}
	// src/Data/Skills/act_str.lua:3932
	m["EnduringCry"] = map[string]*StatMap{
		"enduring_cry_life_regeneration_rate_per_minute_%_per_5_power_up_to_cap": {
			Mods: []mod.Mod{mod.NewFloat("LifeRegenPercent", mod.TypeBase, 0).Tag(mod.GlobalEffect("Warcry").Div(5).Limit(25))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/act_int.lua:5087
	m["EnergyBlade"] = map[string]*StatMap{
		"storm_blade_damage_+%_final_with_two_hand_weapon": {
			Mods: []mod.Mod{mod.NewFloat("EnergyBladeDamage", mod.TypeMore, 0).Tag(mod.Condition("UsingTwoHandedWeapon"), mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"storm_blade_energy_shield_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("EnergyShield", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"storm_blade_global_attack_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("EnergyBladeAttackSpeed", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"storm_blade_maximum_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("EnergyBladeMaxLightning", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"storm_blade_maximum_lightning_damage_from_es_%": {
			Mods: []mod.Mod{mod.NewFloat("EnergyBladeMaxLightning", mod.TypeBase, 0).Tag(mod.PercentStat("EnergyShield", 1), mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"storm_blade_minimum_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("EnergyBladeMinLightning", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"storm_blade_minimum_lightning_damage_from_es_%": {
			Mods: []mod.Mod{mod.NewFloat("EnergyBladeMinLightning", mod.TypeBase, 0).Tag(mod.PercentStat("EnergyShield", 1), mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_int.lua:5179
	m["Enfeeble"] = map[string]*StatMap{
		"accuracy_rating_+%": {
			Mods: []mod.Mod{mod.NewFloat("Accuracy", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"enfeeble_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique").Neg(true))},
		},
		"enfeeble_damage_+%_vs_rare_or_unique_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique"))},
		},
	}
	// src/Data/Skills/act_dex.lua:6373
	m["EnsnaringArrow"] = map[string]*StatMap{
		"tethered_enemies_take_attack_projectile_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileAttackDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Debuff").Name("Ensnared").EffectStackVar("EnsnareStackCount").EffectStackLimit(1))},
		},
	}
	// src/Data/Skills/other.lua:1741
	m["Envy"] = map[string]*StatMap{
		"attack_maximum_added_chaos_damage": {
			Mods: []mod.Mod{mod.NewFloat("ChaosMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"attack_minimum_added_chaos_damage": {
			Mods: []mod.Mod{mod.NewFloat("ChaosMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"spell_maximum_added_chaos_damage": {
			Mods: []mod.Mod{mod.NewFloat("ChaosMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagSpell).Tag(mod.GlobalEffect("Aura"))},
		},
		"spell_minimum_added_chaos_damage": {
			Mods: []mod.Mod{mod.NewFloat("ChaosMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagSpell).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_dex.lua:6776
	m["ExplosiveArrow"] = map[string]*StatMap{
		"explosive_arrow_ailment_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.SkillPartList(1, 2), mod.Multiplier("ExplosiveArrowStage"))},
		},
		"explosive_arrow_explosion_base_damage_+permyriad": {
			Mods: []mod.Mod{skill("baseMultiplier", 0).Tag(mod.SkillPartList(1, 2))},
			Div:  utils.Ptr(float64(-10000)),
		},
		"explosive_arrow_explosion_maximum_added_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMax", mod.TypeBase, 0).Tag(mod.SkillPartList(1, 2))},
		},
		"explosive_arrow_explosion_minimum_added_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMin", mod.TypeBase, 0).Tag(mod.SkillPartList(1, 2))},
		},
		"explosive_arrow_hit_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit).Tag(mod.SkillPartList(1, 2), mod.Multiplier("ExplosiveArrowStage"))},
		},
		"explosive_arrow_maximum_bonus_explosion_radius": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ExplosiveArrowMaxBonusRadius", mod.TypeBase, 0)},
		},
		"explosive_arrow_stack_limit": {
			Mods: []mod.Mod{
				mod.NewFloat("Multiplier:ExplosiveArrowMaxStages", mod.TypeBase, 0).Tag(mod.SkillPart(1)),
				mod.NewFloat("ExplosiveArrowMaxFuseCount", mod.TypeBase, 0),
			},
		},
		"fuse_arrow_explosion_radius_+_per_fuse_arrow_orb": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(multiplierLimitVar("ExplosiveArrowStage", "ExplosiveArrowMaxBonusRadius").LimitTotal(true))},
		},
		"quality_display_explosive_arrow_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:6919
	m["ExplosiveConcoction"] = map[string]*StatMap{
		"flask_throw_maximum_cold_damage_if_used_sapphire_flask": {
			Mods: []mod.Mod{mod.NewFloat("ColdMax", mod.TypeBase, 0).Tag(mod.SkillPartList(2, 5, 6, 8))},
		},
		"flask_throw_maximum_lightning_damage_if_used_topaz_flask": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).Tag(mod.SkillPartList(3, 5, 7, 8))},
		},
		"flask_throw_minimum_cold_damage_if_used_sapphire_flask": {
			Mods: []mod.Mod{mod.NewFloat("ColdMin", mod.TypeBase, 0).Tag(mod.SkillPartList(2, 5, 6, 8))},
		},
		"flask_throw_minimum_lightning_damage_if_used_topaz_flask": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).Tag(mod.SkillPartList(3, 5, 7, 8))},
		},
		"flask_throw_ruby_flask_ignite_dot_multiplier_+": {
			Mods: []mod.Mod{mod.NewFloat("FireDotMultiplier", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagIgnite).Tag(mod.SkillPartList(4, 6, 7, 8))},
		},
	}
	// src/Data/Skills/act_dex.lua:7045
	m["ExplosiveConcoctionAltX"] = map[string]*StatMap{
		"flask_throw_maximum_cold_damage_if_used_sapphire_flask": {
			Mods: []mod.Mod{mod.NewFloat("ColdMax", mod.TypeBase, 0).Tag(mod.SkillPartList(2, 5, 6, 8))},
		},
		"flask_throw_maximum_lightning_damage_if_used_topaz_flask": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).Tag(mod.SkillPartList(3, 5, 7, 8))},
		},
		"flask_throw_minimum_cold_damage_if_used_sapphire_flask": {
			Mods: []mod.Mod{mod.NewFloat("ColdMin", mod.TypeBase, 0).Tag(mod.SkillPartList(2, 5, 6, 8))},
		},
		"flask_throw_minimum_lightning_damage_if_used_topaz_flask": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).Tag(mod.SkillPartList(3, 5, 7, 8))},
		},
		"flask_throw_ruby_flask_critical_strike_multiplier_+": {
			Mods: []mod.Mod{mod.NewFloat("CritMultiplier", mod.TypeBase, 0).Tag(mod.SkillPartList(4, 6, 7, 8))},
		},
	}
	// src/Data/Skills/act_dex.lua:7219
	m["ExplosiveTrap"] = map[string]*StatMap{
		"quality_display_explosive_trap_is_gem": {
			Mods: []mod.Mod{},
		},
		"shrapnel_trap_number_of_secondary_explosions": {
			Mods: []mod.Mod{mod.NewFloat("SmallExplosions", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills
	m["ExplosiveTrapAltX"] = map[string]*StatMap{
		"quality_display_explosive_trap_is_gem": {
			Mods: []mod.Mod{},
		},
		"shrapnel_trap_number_of_secondary_explosions": {
			Mods: []mod.Mod{mod.NewFloat("SmallExplosions", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_str.lua:4110
	m["Exsanguinate"] = map[string]*StatMap{
		"base_physical_damage_to_deal_per_minute": {
			Mods: []mod.Mod{
				skill("PhysicalDot", 0).Tag(mod.Condition("ExsanguinateDebuffIsFireDamage").Neg(true)),
				skill("FireDot", 0).Tag(mod.Condition("ExsanguinateDebuffIsFireDamage")),
			},
			Div: utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/act_str.lua:4199
	m["ExsanguinateAltX"] = map[string]*StatMap{
		"base_physical_damage_to_deal_per_minute": {
			Mods: []mod.Mod{
				skill("PhysicalDot", 0).Tag(mod.Condition("ExsanguinateDebuffIsFireDamage").Neg(true)),
				skill("FireDot", 0).Tag(mod.Condition("ExsanguinateDebuffIsFireDamage")),
			},
			Div: utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/act_int.lua:5521
	m["EyeOfWinter"] = map[string]*StatMap{
		"freezing_pulse_damage_+%_final_at_long_range": {
			Mods: []mod.Mod{mod.NewFloat("EyeOfWinterRamp", mod.TypeBase, 0)},
		},
		"quality_display_eye_of_winter_is_gem": {
			Mods: []mod.Mod{},
		},
		"quality_display_freezing_pulse_damage_at_long_range_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:5614
	m["EyeOfWinterAltX"] = map[string]*StatMap{
		"freezing_pulse_damage_+%_final_at_long_range": {
			Mods: []mod.Mod{mod.NewFloat("EyeOfWinterRamp", mod.TypeBase, 0)},
		},
		"quality_display_eye_of_winter_is_gem": {
			Mods: []mod.Mod{},
		},
		"quality_display_freezing_pulse_damage_at_long_range_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:5707
	m["EyeOfWinterAltY"] = map[string]*StatMap{
		"freezing_pulse_damage_+%_final_at_long_range": {
			Mods: []mod.Mod{mod.NewFloat("EyeOfWinterRamp", mod.TypeBase, 0)},
		},
		"quality_display_eye_of_winter_is_gem": {
			Mods: []mod.Mod{},
		},
		"quality_display_freezing_pulse_damage_at_long_range_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/spectre.lua:11730
	m["FaridunCasterUndeadDamageOverTimeAura"] = map[string]*StatMap{
		"delirium_aura_damage_over_time_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagDot).Tag(mod.GlobalEffect("Aura"))},
		},
		"delirium_skill_effect_duration_+%": {
			Mods: []mod.Mod{mod.NewFloat("Duration", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/spectre.lua:7092
	m["FemaleCannibalBossFireStorm"] = map[string]*StatMap{
		"fire_storm_fireball_delay_ms": {
			Mods: []mod.Mod{skill("hitTimeOverride", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"firestorm_base_area_of_effect_+%": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffectPrimary", mod.TypeIncrease, 0)},
		},
	}
	// src/Data/Skills/other.lua:1817
	m["FireAegis"] = map[string]*StatMap{
		"aegis_unique_shield_max_value": {
			Mods: []mod.Mod{mod.NewFloat("FireAegisValue", mod.TypeMAX, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/spectre.lua:789
	m["FireMonsterWhirlingBlades"] = map[string]*StatMap{
		"whirling_blades_base_ground_fire_damage_to_deal_per_minute": {
			Mods: []mod.Mod{skill("FireDot", 0)},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/act_int.lua:5978
	m["Firestorm"] = map[string]*StatMap{
		"firestorm_initial_impact_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.SkillPart(1))},
		},
		"firestorm_initial_impact_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.SkillPart(1))},
		},
	}
	// src/Data/Skills/act_str.lua:4293
	m["FlameLink"] = map[string]*StatMap{
		"flame_link_added_fire_damage_from_life_%": {
			Mods: []mod.Mod{
				mod.NewFloat("FireMin", mod.TypeBase, 0).Tag(mod.GlobalEffect("Link"), mod.PercentStat("Life", 1).Actor("parent")),
				mod.NewFloat("FireMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Link"), mod.PercentStat("Life", 1).Actor("parent")),
			},
		},
		"flame_link_maximum_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Link"))},
		},
		"flame_link_minimum_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMin", mod.TypeBase, 0).Tag(mod.GlobalEffect("Link"))},
		},
	}
	// src/Data/Skills/act_int.lua:6613
	m["FlameSurge"] = map[string]*StatMap{
		"active_skill_base_area_length_+": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeBase, 0)},
		},
		"flame_surge_ignite_damage_as_burning_ground_damage_%": {
			Mods: []mod.Mod{mod.NewFloat("IgniteDpsAsBurningGround", mod.TypeMAX, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"flame_whip_damage_+%_final_vs_burning_enemies": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.ActorCondition("enemy", "Burning"))},
		},
		"quality_display_flame_whip_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:6709
	m["FlameSurgeAltX"] = map[string]*StatMap{
		"active_skill_base_area_length_+": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeBase, 0)},
		},
		"flame_surge_burning_ground_on_ignite_damage_%": {
			Mods: []mod.Mod{mod.NewFloat("IgniteDpsAsBurningGround", mod.TypeMAX, 0)},
		},
		"quality_display_alt_flame_whip_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:6499
	m["FlameWall"] = map[string]*StatMap{
		"base_fire_damage_to_deal_per_minute": {
			Mods: []mod.Mod{skill("FireDot", 0).Tag(mod.SkillPart(1))},
			Div:  utils.Ptr(float64(60)),
		},
		"flame_wall_maximum_added_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMax", mod.TypeBase, 0).Flag(mod.MFlagProjectile).Tag(mod.GlobalEffect("Buff").Name("Flame Wall").EffectCond("FlameWallAddedDamage"))},
		},
		"flame_wall_minimum_added_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMin", mod.TypeBase, 0).Flag(mod.MFlagProjectile).Tag(mod.GlobalEffect("Buff").Name("Flame Wall").EffectCond("FlameWallAddedDamage"))},
		},
		"quality_display_firewall_is_gem": {
			Mods: []mod.Mod{},
		},
		"quality_display_wall_length_is_gem": {
			Mods: []mod.Mod{},
		},
		"secondary_base_fire_damage_to_deal_per_minute": {
			Mods: []mod.Mod{skill("FireDot", 0).Tag(mod.SkillPart(2))},
			Div:  utils.Ptr(float64(60)),
		},
		"wall_maximum_length": {
			Mods: []mod.Mod{skill("radius", 0)},
		},
	}
	// src/Data/Skills/act_int.lua:6803
	m["Flameblast"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"charged_blast_spell_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("FlameblastStage"))},
		},
		"flameblast_ailment_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("FlameblastStage"))},
		},
		"flameblast_maximum_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:FlameblastMaxStages", mod.TypeBase, 0)},
		},
		"quality_display_flameblast_is_gem": {
			Mods: []mod.Mod{},
		},
		"vaal_flameblast_radius_+_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeBase, 0).Tag(mod.Multiplier("FlameblastStageAfterFirst"))},
		},
	}
	// src/Data/Skills/act_int.lua:6908
	m["FlameblastAltX"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"charged_blast_spell_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("FlameblastofCelerityStage"))},
		},
		"flameblast_base_radius_override": {
			Mods: []mod.Mod{skill("radius", 0)},
		},
		"flameblast_maximum_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:FlameblastofCelerityMaxStages", mod.TypeBase, 0)},
		},
		"quality_display_flameblast_is_gem": {
			Mods: []mod.Mod{},
		},
		"vaal_flameblast_radius_+_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeBase, 0).Tag(mod.Multiplier("FlameblastofCelerityStageAfterFirst"))},
		},
	}
	// src/Data/Skills/act_int.lua:7011
	m["FlameblastAltY"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"charged_blast_spell_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("FlameblastofContractionStage"))},
		},
		"flameblast_ailment_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("FlameblastofContractionStage"))},
		},
		"flameblast_base_radius_override": {
			Mods: []mod.Mod{skill("radius", 0)},
		},
		"flameblast_maximum_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:FlameblastofContractionMaxStages", mod.TypeBase, 0)},
		},
		"quality_display_flameblast_is_gem": {
			Mods: []mod.Mod{},
		},
		"vaal_flameblast_radius_+_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeBase, 0).Tag(mod.Multiplier("FlameblastofContractionStageAfterFirst"))},
		},
	}
	// src/Data/Skills/act_dex.lua:7715
	m["FlamethrowerTrap"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"flamethrower_trap_damage_+%_final_vs_burning_enemies": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.ActorCondition("enemy", "Burning"))},
		},
	}
	// src/Data/Skills/act_int.lua:7207
	m["Flammability"] = map[string]*StatMap{
		"base_fire_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("FireResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"base_self_ignite_duration_-%": {
			Mods: []mod.Mod{mod.NewFloat("SelfIgniteDuration", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
			Mult: utils.Ptr(float64(-1)),
		},
		"chance_to_be_ignited_%": {
			Mods: []mod.Mod{mod.NewFloat("SelfIgniteChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_str.lua:4376
	m["FleshAndStone"] = map[string]*StatMap{
		"flesh_stone_blood_stance_enemies_physical_damage_taken_when_hit_+%_final_from_player_distance": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.GlobalEffect("AuraDebuff").EffectCond("BloodStance"))},
		},
		"flesh_stone_sand_stance_damage_taken_+%_final_from_distance_from_enemy_hits": {
			Mods: []mod.Mod{mod.NewFloat("DamageTakenWhenHit", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff").EffectCond("SandStance"))},
		},
	}
	// src/Data/Skills/act_int.lua:7297
	m["FleshOffering"] = map[string]*StatMap{
		"attack_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Buff"))},
		},
		"base_movement_velocity_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"cast_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagCast).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_dex.lua:7920
	m["FlickerStrike"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"flicker_strike_buff_movement_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff").Name("Flick Strike"))},
		},
	}
	// src/Data/Skills/act_dex.lua:8011
	m["FlickerStrikeAltX"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"flicker_strike_buff_movement_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff").Name("Flick Strike"))},
		},
	}
	// src/Data/Skills/act_int.lua:7435
	m["ForbiddenRite"] = map[string]*StatMap{
		"base_skill_area_of_effect_+%": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0)},
		},
		"skill_base_chaos_damage_%_maximum_energy_shield": {
			Mods: []mod.Mod{skill("energyShieldDealtAsChaos", 0)},
			Div:  utils.Ptr(float64(100)),
		},
		"skill_base_chaos_damage_%_maximum_life": {
			Mods: []mod.Mod{skill("lifeDealtAsChaos", 0)},
			Div:  utils.Ptr(float64(100)),
		},
		"soulfeast_take_%_maximum_energy_shield_as_chaos_damage": {
			Mods: []mod.Mod{skill("SelfDamageTakenES", 0)},
			Div:  utils.Ptr(float64(100)),
		},
		"soulfeast_take_%_maximum_life_as_chaos_damage": {
			Mods: []mod.Mod{skill("SelfDamageTakenLife", 0)},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/act_int.lua:7590
	m["ForbiddenRiteAltX"] = map[string]*StatMap{
		"base_skill_area_of_effect_+%": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0)},
		},
		"skill_base_chaos_damage_%_maximum_energy_shield": {
			Mods: []mod.Mod{skill("energyShieldDealtAsChaos", 0)},
			Div:  utils.Ptr(float64(100)),
		},
		"soulfeast_take_%_maximum_energy_shield_as_chaos_damage": {
			Mods: []mod.Mod{skill("SelfDamageTakenES", 0)},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/act_int.lua:7690
	m["FreezingPulse"] = map[string]*StatMap{
		"display_what_freezing_pulse_does": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:8193
	m["Frenzy"] = map[string]*StatMap{
		"frenzy_skill_attack_damage_+%_final_per_frenzy_charge": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.Multiplier("FrenzyCharge"))},
		},
		"frenzy_skill_attack_speed_+%_final_per_frenzy_charge": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.Multiplier("FrenzyCharge"))},
		},
		"quality_display_active_skill_attack_speed_per_frenzy_is_gem": {
			Mods: []mod.Mod{},
		},
		"quality_display_frenzy_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:8292
	m["FrenzyAltX"] = map[string]*StatMap{
		"frenzy_skill_attack_damage_+%_final_per_frenzy_charge": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.Multiplier("FrenzyCharge"))},
		},
		"frenzy_skill_attack_speed_+%_final_per_frenzy_charge": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.Multiplier("FrenzyCharge"))},
		},
		"quality_display_active_skill_attack_speed_per_frenzy_is_gem": {
			Mods: []mod.Mod{},
		},
		"quality_display_frenzy_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:8400
	m["FrostBlades"] = map[string]*StatMap{
		"active_skill_hit_ailment_damage_with_projectile_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_dex.lua:8508
	m["FrostBladesAltX"] = map[string]*StatMap{
		"base_cold_damage_to_deal_per_minute": {
			Mods: []mod.Mod{skill("ColdDot", 0).Tag(mod.SkillPart(2))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/act_int.lua:7766
	m["FrostBomb"] = map[string]*StatMap{
		"base_cold_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("ColdExposure", mod.TypeBase, 0).Tag(mod.GlobalEffect("Debuff").Name("Frost Bomb"))},
		},
		"life_regeneration_rate_+%": {
			Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Debuff").Name("Frost Bomb"))},
		},
	}
	// src/Data/Skills/act_int.lua:7937
	m["FrostBombAltY"] = map[string]*StatMap{
		"active_skill_ailment_damage_+%_final_per_100ms_duration": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("100msFrostBombDuration"))},
		},
		"active_skill_hit_damage_+%_final_per_100ms_duration": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit).Tag(mod.Multiplier("100msFrostBombDuration"))},
		},
	}
	// src/Data/Skills/act_int.lua:8022
	m["FrostShield"] = map[string]*StatMap{
		"energy_shield_lost_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("EnergyShieldDegen", mod.TypeBase, 0).Tag(mod.MultiplierThreshold("FrostShieldStage").Threshold(1), mod.GlobalEffect("Buff").Name("Frost Shield"))},
			Div:  utils.Ptr(float64(60)),
		},
		"frost_globe_absorb_damage_%_enemy_in_bubble": {
			Mods: []mod.Mod{mod.NewFloat("FrostGlobeDamageMitigation", mod.TypeBase, 0).Tag(mod.ActorCondition("enemy", "EnemyInFrostGlobe"), mod.GlobalEffect("Buff").Name("Frost Shield"))},
		},
		"frost_globe_absorb_damage_%_enemy_outside_bubble": {
			Mods: []mod.Mod{mod.NewFloat("FrostGlobeDamageMitigation", mod.TypeBase, 0).Tag(mod.ActorCondition("enemy", "EnemyInFrostGlobe").Neg(true), mod.GlobalEffect("Buff").Name("Frost Shield"))},
		},
		"frost_globe_additional_spell_base_critical_strike_chance_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeBase, 0).Flag(mod.MFlagSpell).Tag(multiplierLimitVar("FrostShieldStage", "FrostShieldMaxStages"), mod.GlobalEffect("Buff").Name("Frost Shield"))},
			Div:  utils.Ptr(float64(100)),
		},
		"frost_globe_health_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("FrostGlobeHealth", mod.TypeBase, 0).Tag(multiplierLimitVar("FrostShieldStage", "FrostShieldMaxStages"), mod.GlobalEffect("Buff").Name("Frost Shield"))},
		},
		"frost_globe_max_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:FrostShieldMaxStages", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_int.lua:8120
	m["FrostWall"] = map[string]*StatMap{
		"quality_display_wall_length_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:8199
	m["Frostbite"] = map[string]*StatMap{
		"base_cold_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("ColdResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"base_self_freeze_duration_-%": {
			Mods: []mod.Mod{mod.NewFloat("SelfFreezeDuration", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
			Mult: utils.Ptr(float64(-1)),
		},
		"chance_to_be_frozen_%": {
			Mods: []mod.Mod{mod.NewFloat("SelfFreezeChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_int.lua:8291
	m["Frostblink"] = map[string]*StatMap{
		"ice_dash_cooldown_recovery_per_nearby_normal_or_magic_enemy": {
			Mods: []mod.Mod{mod.NewFloat("CooldownRecovery", mod.TypeIncrease, 0).Tag(mod.ActorCondition("enemy", "RareOrUnique").Neg(true))},
		},
		"ice_dash_cooldown_recovery_per_nearby_rare_or_unique_enemy": {
			Mods: []mod.Mod{mod.NewFloat("CooldownRecovery", mod.TypeIncrease, 0).Tag(mod.ActorCondition("enemy", "RareOrUnique"))},
		},
	}
	// src/Data/Skills/act_int.lua:8384
	m["FrostblinkAltX"] = map[string]*StatMap{
		"frostblink_damage_+%_final_per_5%_chill_effect_on_target": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("ChillEffect").Div(5).Actor("enemy"))},
		},
	}
	// src/Data/Skills/act_int.lua:8554
	m["GalvanicField"] = map[string]*StatMap{
		"base_chance_to_shock_%_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("EnemyShockChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Galvanic Field"))},
		},
		"base_galvanic_field_beam_delay_ms": {
			Mods: []mod.Mod{skill("repeatFrequency", 0).Tag(mod.SkillPart(2))},
			Div:  utils.Ptr(float64(1000)),
		},
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"galvanic_field_damage_+%_final_per_5%_increased_damage_taken_from_shock": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("ShockEffect").Div(5).Actor("enemy"))},
		},
		"galvanic_field_radius_+_per_10%_increased_damage_taken_from_shock": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(mod.Multiplier("ShockEffect").Div(10).Actor("enemy"))},
		},
		"galvanic_field_retargeting_delay_ms": {
			Mods: []mod.Mod{skill("repeatFrequency", 0).Tag(mod.SkillPart(1))},
			Div:  utils.Ptr(float64(1000)),
		},
		"quality_display_shock_chance_from_skill_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:8674
	m["GalvanicFieldAltX"] = map[string]*StatMap{
		"base_chance_to_shock_%_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("EnemyShockChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Galvanic Field of Intensity"))},
		},
		"base_galvanic_field_beam_delay_ms": {
			Mods: []mod.Mod{skill("repeatFrequency", 0).Tag(mod.SkillPart(2))},
			Div:  utils.Ptr(float64(1000)),
		},
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"galvanic_field_retargeting_delay_ms": {
			Mods: []mod.Mod{skill("repeatFrequency", 0).Tag(mod.SkillPart(1))},
			Div:  utils.Ptr(float64(1000)),
		},
		"quality_display_shock_chance_from_skill_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:4836
	m["GeneralsCry"] = map[string]*StatMap{
		"maximum_number_of_spiritual_cry_warriors": {
			Mods: []mod.Mod{mod.NewFloat("GeneralsCryDoubleMaxCount", mod.TypeBase, 0)},
		},
		"quality_display_generals_cry_is_gem": {
			Mods: []mod.Mod{},
		},
		"spiritual_cry_doubles_summoned_per_5_MP": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:4931
	m["GeneralsCrySupport"] = map[string]*StatMap{
		"number_of_warcries_exerting_this_action": {
			Mods: []mod.Mod{},
		},
		"support_spiritual_cry_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:8783
	m["GlacialCascade"] = map[string]*StatMap{
		"glacial_cascade_final_spike_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"quality_display_glacial_cascade_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:5005
	m["GlacialHammer"] = map[string]*StatMap{
		"glacial_hammer_third_hit_freeze_as_though_dealt_damage_+%": {
			Mods: []mod.Mod{mod.NewFloat("FreezeAsThoughDealing", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/act_str.lua:5086
	m["GlacialHammerAltX"] = map[string]*StatMap{
		"glacial_hammer_third_hit_always_crits": {
			Mods: []mod.Mod{mod.NewFlag("Every3UseCrit", true)},
		},
	}
	// src/Data/Skills/other.lua:2057
	m["GluttonyOfElements"] = map[string]*StatMap{
		"immune_to_curses": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:8927
	m["Grace"] = map[string]*StatMap{
		"base_evasion_rating": {
			Mods: []mod.Mod{mod.NewFloat("Evasion", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"grace_aura_evasion_rating_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Evasion", mod.TypeMore, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_str.lua:5254
	m["GroundSlam"] = map[string]*StatMap{
		"groundslam_damage_to_close_targets_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Condition("AtCloseRange"))},
		},
		"quality_display_groundslam_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:5344
	m["GroundSlamAltX"] = map[string]*StatMap{
		"groundslam_damage_to_close_targets_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Condition("AtCloseRange"))},
		},
		"quality_display_groundslam_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/spectre.lua:4841
	m["HarvestNessaCrabScreechDebuff"] = map[string]*StatMap{
		"frigid_roar_cold_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("ColdDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Debuff").Name("Frigid Roar"))},
		},
	}
	// src/Data/Skills/spectre.lua:4965
	m["HarvestRhexScreechDebuff"] = map[string]*StatMap{
		"thunderous_roar_lightning_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("LightningDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Debuff").Name("Thunderous Roar"))},
		},
	}
	// src/Data/Skills/act_dex.lua:9088
	m["Haste"] = map[string]*StatMap{
		"attack_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_movement_velocity_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"cast_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagCast).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_dex.lua:9262
	m["Hatred"] = map[string]*StatMap{
		"physical_damage_%_to_add_as_cold": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsCold", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_str.lua:5540
	m["HeavyStrike"] = map[string]*StatMap{
		"chance_to_deal_double_damage_%": {
			Mods: []mod.Mod{mod.NewFloat("DoubleDamageChance", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/spectre.lua:5215
	m["HeistThugRangedExplosiveArrow"] = map[string]*StatMap{
		"explosive_arrow_explosion_base_damage_+permyriad": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment)},
			Div:  utils.Ptr(float64(100)),
		},
		"explosive_arrow_explosion_maximum_added_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMax", mod.TypeBase, 0)},
		},
		"explosive_arrow_explosion_minimum_added_fire_damage": {
			Mods: []mod.Mod{mod.NewFloat("FireMin", mod.TypeBase, 0)},
		},
		"explosive_arrow_hit_and_ailment_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("ExplosiveArrowFuse"))},
		},
		"fuse_arrow_explosion_radius_+_per_fuse_arrow_orb": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(mod.Multiplier("ExplosiveArrowFuse"))},
		},
	}
	// src/Data/Skills/act_dex.lua:9343
	m["HeraldOfAgony"] = map[string]*StatMap{
		"herald_of_agony_poison_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagPoison).Tag(mod.GlobalEffect("Buff"))},
		},
		"maximum_virulence_stacks": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:VirulenceStacksMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura").Name("Virulence Stack Limit").Unscalable(true))},
		},
		"quality_display_herald_of_agony_is_gem": {
			Mods: []mod.Mod{},
		},
		"scorpion_minion_attack_speed_+%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Speed", mod.TypeIncrease, 0).Tag(multiplierLimitVar("VirulenceStack", "VirulenceStacksMax").Actor("parent").LimitActor("parent"))})},
		},
		"scorpion_minion_maximum_added_physical_damage": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMax", mod.TypeBase, 0).Tag(multiplierLimitVar("VirulenceStack", "VirulenceStacksMax").Actor("parent").LimitActor("parent"))})},
		},
		"scorpion_minion_minimum_added_physical_damage": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMin", mod.TypeBase, 0).Tag(multiplierLimitVar("VirulenceStack", "VirulenceStacksMax").Actor("parent").LimitActor("parent"))})},
		},
		"scorpion_minion_physical_damage_+%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDamage", mod.TypeIncrease, 0).Tag(multiplierLimitVar("VirulenceStack", "VirulenceStacksMax").Actor("parent").LimitActor("parent"))})},
		},
		"skill_buff_grants_chance_to_poison_%": {
			Mods: []mod.Mod{mod.NewFloat("PoisonChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_str.lua:5683
	m["HeraldOfAsh"] = map[string]*StatMap{
		"herald_of_ash_burning_%_overkill_damage_per_minute": {
			Mods: []mod.Mod{skill("hoaOverkillPercent", 0)},
			Div:  utils.Ptr(float64(6000)),
		},
		"herald_of_ash_burning_damage_+%_final": {
			Mods: []mod.Mod{skill("hoaMoreBurn", 0)},
		},
		"physical_damage_%_to_add_as_fire": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsFire", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"quality_display_herald_of_ash_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:9446
	m["HeraldOfIce"] = map[string]*StatMap{
		"attack_maximum_added_cold_damage": {
			Mods: []mod.Mod{mod.NewFloat("ColdMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Buff"))},
		},
		"attack_minimum_added_cold_damage": {
			Mods: []mod.Mod{mod.NewFloat("ColdMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Buff"))},
		},
		"spell_maximum_added_cold_damage": {
			Mods: []mod.Mod{mod.NewFloat("ColdMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagSpell).Tag(mod.GlobalEffect("Buff"))},
		},
		"spell_minimum_added_cold_damage": {
			Mods: []mod.Mod{mod.NewFloat("ColdMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagSpell).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_str.lua:5774
	m["HeraldOfPurity"] = map[string]*StatMap{
		"base_number_of_champions_of_light_allowed": {
			Mods: []mod.Mod{mod.NewFloat("ActiveSentinelOfPurityLimit", mod.TypeBase, 0)},
		},
		"herald_of_purity_physical_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/other.lua:2133
	m["HeraldOfTheBreach"] = map[string]*StatMap{
		"base_herald_of_the_breach_pulse_delay_ms": {
			Mods: []mod.Mod{skill("repeatFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"herald_of_the_breach_add_stack_on_inflicting_ailment": {
			Mods: []mod.Mod{mod.NewFloat("OtherworldlyPressure", mod.TypeBase, 1).Tag(mod.Condition("InflictedAilmentRecently"))},
		},
		"herald_of_the_breach_pulse_frequency_+%_per_otherworldly_pressure": {
			Mods: []mod.Mod{mod.NewFloat("PulseFrequencyPerPressure", mod.TypeIncrease, 0).Tag(multiplierLimitVar("OtherworldlyPressure", "OtherworldlyPressureMax"))},
		},
		"maximum_otherwordly_pressure_stacks": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:OtherworldlyPressureMax", mod.TypeBase, 0)},
		},
		"skill_buff_grants_damage_+%_final_per_herald_skill_affecting_you": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("Herald"), mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_int.lua:9386
	m["HeraldOfThunder"] = map[string]*StatMap{
		"attack_maximum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Buff"))},
		},
		"attack_minimum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Buff"))},
		},
		"herald_of_thunder_bolt_base_frequency": {
			Mods: []mod.Mod{skill("repeatFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"herald_of_thunder_bolt_frequency_+%": {
			Mods: []mod.Mod{mod.NewFloat("HeraldStormFrequency", mod.TypeIncrease, 0)},
		},
		"spell_maximum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagSpell).Tag(mod.GlobalEffect("Buff"))},
		},
		"spell_minimum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagSpell).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_int.lua:9104
	m["Hexblast"] = map[string]*StatMap{
		"chaos_damage_resisted_by_lowest_resistance": {
			Mods: []mod.Mod{mod.NewFlag("ChaosDamageUsesLowestResistance", true)},
		},
		"hexblast_ailment_damage_+%_final_if_hexed": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "Cursed"))},
		},
		"hexblast_display_innate_remove_hex_100%_chance": {
			Mods: []mod.Mod{},
		},
		"hexblast_hit_damage_+%_final_if_hexed": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.ActorCondition("enemy", "Cursed"))},
		},
	}
	// src/Data/Skills/act_int.lua:9207
	m["HexblastAltX"] = map[string]*StatMap{
		"chaos_damage_resisted_by_highest_resistance": {
			Mods: []mod.Mod{mod.NewFlag("ChaosDamageUsesHighestResistance", true)},
		},
		"hexblast_ailment_damage_+%_final_if_hexed": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "Cursed"))},
		},
		"hexblast_display_innate_remove_hex_100%_chance": {
			Mods: []mod.Mod{},
		},
		"hexblast_hit_damage_+%_final_if_hexed": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.ActorCondition("enemy", "Cursed"))},
		},
	}
	// src/Data/Skills/act_str.lua:6051
	m["HolyHammers"] = map[string]*StatMap{
		"holy_hammers_damage_+%_final_for_initial_hammer_in_cascade": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"holy_hammers_damage_+%_final_if_consuming_power_charge": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("RemovablePowerCharge").Limit(1))},
		},
	}
	// src/Data/Skills/act_str.lua:6137
	m["HolyHammersAltX"] = map[string]*StatMap{
		"holy_hammers_damage_+%_final_per_power_charge_consumed": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("RemovablePowerCharge"))},
		},
	}
	// src/Data/Skills/act_str.lua:6228
	m["HolyStrike"] = map[string]*StatMap{
		"skill_physical_damage_%_to_convert_to_lightning": {
			Mods: []mod.Mod{
				mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0),
				mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0)}),
			},
		},
	}
	// src/Data/Skills/act_int.lua:8977
	m["Hydrosphere"] = map[string]*StatMap{
		"hydro_sphere_base_pulse_frequency_ms": {
			Mods: []mod.Mod{skill("repeatFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"hydro_sphere_pulse_frequency_+%": {
			Mods: []mod.Mod{mod.NewFloat("HydroSphereFrequency", mod.TypeIncrease, 0)},
		},
		"quality_display_hydrosphere_is_gem": {
			Mods: []mod.Mod{},
		},
		"skill_physical_damage_%_to_convert_to_cold": {
			Mods: []mod.Mod{
				mod.NewFloat("SkillPhysicalDamageConvertToCold", mod.TypeBase, 0).Tag(mod.SkillPart(1)),
				mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0).Tag(mod.SkillPart(2)),
				mod.NewFloat("SkillPhysicalDamageConvertToCold", mod.TypeBase, 0).Tag(mod.SkillPart(3)),
				mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0).Tag(mod.SkillPart(3)),
				mod.NewFloat("SkillPhysicalDamageConvertToCold", mod.TypeBase, 0).Tag(mod.SkillPart(4)),
				mod.NewFloat("SkillPhysicalDamageConvertToCold", mod.TypeBase, 0).Tag(mod.SkillPart(5)),
				mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0).Tag(mod.SkillPart(6)),
				mod.NewFloat("SkillPhysicalDamageConvertToCold", mod.TypeBase, 0).Tag(mod.SkillPart(7)),
				mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0).Tag(mod.SkillPart(7)),
			},
		},
		"water_sphere_cold_lightning_exposure_%": {
			Mods: []mod.Mod{
				mod.NewFloat("ColdExposure", mod.TypeBase, 0).Tag(mod.GlobalEffect("Debuff")),
				mod.NewFloat("LightningExposure", mod.TypeBase, 0).Tag(mod.GlobalEffect("Debuff")),
			},
		},
		"water_sphere_does_weird_conversion_stuff": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:6529
	m["IceCrash"] = map[string]*StatMap{
		"ice_crash_second_hit_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"ice_crash_third_hit_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(3))},
		},
		"quality_display_ice_crash_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:9567
	m["IceNovaAltX"] = map[string]*StatMap{
		"ice_nova_damage_when_cast_on_frostbolt_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Condition("CastOnFrostbolt"))},
		},
	}
	// src/Data/Skills/act_int.lua:9648
	m["IceNovaAltY"] = map[string]*StatMap{
		"ice_nova_freeze_as_though_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("FreezeAsThoughDealing", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:9824
	m["IceSpear"] = map[string]*StatMap{
		"ice_spear_second_form_critical_strike_chance_+%": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeIncrease, 0).Tag(mod.SkillPartList(2, 4))},
		},
		"ice_spear_second_form_critical_strike_multiplier_+": {
			Mods: []mod.Mod{mod.NewFloat("CritMultiplier", mod.TypeBase, 0).Tag(mod.SkillPartList(2, 4))},
		},
		"ice_spear_second_form_projectile_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileSpeed", mod.TypeMore, 0).Tag(mod.SkillPartList(2, 4))},
		},
	}
	// src/Data/Skills/act_int.lua:9934
	m["IceSpearAltX"] = map[string]*StatMap{
		"ice_spear_second_form_projectile_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileSpeed", mod.TypeMore, 0).Tag(mod.SkillPartList(2, 4))},
		},
	}
}

func skillStatMapsPart2(m map[string]map[string]*StatMap) {
	// src/Data/Skills/other.lua:2251
	m["Icestorm"] = map[string]*StatMap{
		"skill_effect_duration_per_100_int": {
			Mods: []mod.Mod{mod.NewFloat("Duration", mod.TypeBase, 0).Tag(mod.PerStat(100, "Int"))},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/act_int.lua:10018
	m["IcicleMine"] = map[string]*StatMap{
		"cold_projectile_mine_enemy_critical_strike_chance_+%_against_self": {
			Mods: []mod.Mod{mod.NewFloat("SelfCritChance", mod.TypeIncrease, 0).Tag(mod.Limit(500), mod.GlobalEffect("AuraDebuff").EffectStackVar("ActiveMineCount"))},
		},
	}
	// src/Data/Skills/act_int.lua:10107
	m["IcicleMineAltX"] = map[string]*StatMap{
		"cold_projectile_mine_enemy_critical_strike_chance_+%_against_self": {
			Mods: []mod.Mod{mod.NewFloat("SelfCritChance", mod.TypeIncrease, 0).Tag(mod.Limit(500), mod.GlobalEffect("AuraDebuff").EffectStackVar("ActiveMineCount"))},
		},
	}
	// src/Data/Skills/act_int.lua:10195
	m["IcicleMineAltY"] = map[string]*StatMap{
		"cold_projectile_mine_enemy_critical_strike_chance_+%_against_self": {
			Mods: []mod.Mod{mod.NewFloat("SelfCritChance", mod.TypeIncrease, 0).Tag(mod.Limit(500), mod.GlobalEffect("AuraDebuff").EffectStackVar("ActiveMineCount"))},
		},
	}
	// src/Data/Skills/act_str.lua:6704
	m["ImmortalCall"] = map[string]*StatMap{
		"mortal_call_elemental_damage_taken_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ElementalDamageTaken", mod.TypeMore, 0).Tag(mod.GlobalEffect("Guard"))},
		},
		"mortal_call_physical_damage_taken_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeMore, 0).Tag(mod.GlobalEffect("Guard"))},
		},
		"mortal_call_physical_damage_taken_per_endurance_charge_consumed_final_permyriad": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeMore, 0).Tag(mod.GlobalEffect("Guard"), mod.Multiplier("RemovableEnduranceCharge").Limit(5))},
			Div:  utils.Ptr(float64(100)),
		},
		"quality_display_immortal_call_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:10300
	m["Incinerate"] = map[string]*StatMap{
		"expanding_fire_cone_final_wave_always_ignite": {
			Mods:  []mod.Mod{mod.NewFloat("EnemyIgniteChance", mod.TypeBase, 0).Tag(mod.SkillPart(2))},
			Value: utils.Ptr(float64(100)),
		},
		"expanding_fire_cone_maximum_number_of_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:IncinerateMaxStages", mod.TypeBase, 0)},
		},
		"expanding_fire_cone_radius_+_per_stage": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(multiplierLimitVar("IncinerateStage", "IncinerateRadiusLimit").LimitTotal(true))},
		},
		"expanding_fire_cone_radius_limit": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:IncinerateRadiusLimit", mod.TypeBase, 0)},
		},
		"expanding_fire_cone_release_hit_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.SkillPart(2))},
		},
		"flamethrower_damage_+%_per_stage_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("IncinerateStage"))},
		},
		"grant_expanding_fire_cone_release_ignite_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagIgnite).Tag(mod.SkillPart(2))},
		},
		"quality_display_incinerate_is_gem_hit": {
			Mods: []mod.Mod{},
		},
		"quality_display_incinerate_is_gem_ingite": {
			Mods: []mod.Mod{},
		},
		"quality_display_incinerate_is_gem_stages": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:10438
	m["IncinerateAltX"] = map[string]*StatMap{
		"expanding_fire_cone_final_wave_always_ignite": {
			Mods:  []mod.Mod{mod.NewFloat("EnemyIgniteChance", mod.TypeBase, 0).Tag(mod.SkillPart(2))},
			Value: utils.Ptr(float64(100)),
		},
		"expanding_fire_cone_maximum_number_of_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:IncinerateofExpanseMaxStages", mod.TypeBase, 0)},
		},
		"expanding_fire_cone_radius_+_per_stage": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(multiplierLimitVar("IncinerateofExpanseStage", "IncinerateofExpanseRadiusLimit").LimitTotal(true))},
		},
		"expanding_fire_cone_radius_limit": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:IncinerateofExpanseRadiusLimit", mod.TypeBase, 0)},
		},
		"expanding_fire_cone_release_hit_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.SkillPart(2))},
		},
		"flamethrower_damage_+%_per_stage_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("IncinerateofExpanseStage"))},
		},
		"grant_expanding_fire_cone_release_ignite_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagIgnite).Tag(mod.SkillPart(2))},
		},
		"quality_display_incinerate_is_gem_hit": {
			Mods: []mod.Mod{},
		},
		"quality_display_incinerate_is_gem_ingite": {
			Mods: []mod.Mod{},
		},
		"quality_display_incinerate_is_gem_stages": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:10566
	m["IncinerateAltY"] = map[string]*StatMap{
		"expanding_fire_cone_maximum_number_of_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:IncinerateofVentingMaxStages", mod.TypeBase, 0)},
		},
		"expanding_fire_cone_radius_+_per_stage": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(multiplierLimitVar("IncinerateofVentingStage", "IncinerateofVentingRadiusLimit").LimitTotal(true))},
		},
		"expanding_fire_cone_radius_limit": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:IncinerateofVentingRadiusLimit", mod.TypeBase, 0)},
		},
		"flamethrower_damage_+%_per_stage_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("IncinerateofVentingStage"))},
		},
		"quality_display_incinerate_is_gem_hit": {
			Mods: []mod.Mod{},
		},
		"quality_display_incinerate_is_gem_ingite": {
			Mods: []mod.Mod{},
		},
		"quality_display_incinerate_is_gem_stages": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:7075
	m["InfernalBlow"] = map[string]*StatMap{
		"active_skill_base_area_of_effect_radius": {
			Mods: []mod.Mod{skill("radius", 0).Tag(mod.SkillPart(1))},
		},
		"active_skill_base_secondary_area_of_effect_radius": {
			Mods: []mod.Mod{skill("radius", 0).Tag(mod.SkillPartList(2, 3))},
		},
		"infernal_blow_explosion_damage_%_of_total_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("DebuffEffect", mod.TypeBase, 0)},
		},
		"quality_display_infernal_blow_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:7201
	m["InfernalBlowAltX"] = map[string]*StatMap{
		"infernal_blow_explosion_damage_%_of_total_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("DebuffEffect", mod.TypeBase, 0)},
		},
		"quality_display_infernal_blow_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:6861
	m["InfernalCry"] = map[string]*StatMap{
		"infernal_cry_empowered_attacks_trigger_combust_display": {
			Mods: []mod.Mod{},
		},
		"infernal_cry_physical_damage_%_to_add_as_fire_per_5_power_up_to_cap": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsFire", mod.TypeBase, 0).Tag(mod.GlobalEffect("Warcry").Div(5).Limit(25))},
		},
		"skill_empowers_next_x_melee_attacks": {
			Mods: []mod.Mod{mod.NewFloat("InfernalExertedAttacks", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_str.lua:7282
	m["IntimidatingCry"] = map[string]*StatMap{
		"intimidating_cry_empowerd_attacks_deal_double_damage_display": {
			Mods: []mod.Mod{},
		},
		"intimidating_cry_movement_speed_+%_per_5_power_up_to_cap": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Warcry").Div(5).Limit(30))},
		},
		"skill_empowers_next_x_melee_attacks": {
			Mods: []mod.Mod{mod.NewFloat("IntimidatingExertedAttacks", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_dex.lua:18236
	m["IntuitiveLink"] = map[string]*StatMap{
		"display_trigger_link": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/spectre.lua:1376
	m["KaomFireBeamTotemSpectre"] = map[string]*StatMap{
		"fire_beam_enemy_fire_resistance_%_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("FireResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Debuff").EffectStackVar("KaomFireBeamTotemStage").EffectStackLimit(24))},
		},
	}
	// src/Data/Skills/act_int.lua:10772
	m["KineticBlastAltX"] = map[string]*StatMap{
		"added_physical_damage_to_attacks_equal_to_%_maximum_mana": {
			Mods: []mod.Mod{
				mod.NewFloat("PhysicalMin", mod.TypeBase, 0).Flag(mod.MFlagAttack).Tag(mod.PercentStat("Mana", 1)),
				mod.NewFloat("PhysicalMax", mod.TypeBase, 0).Flag(mod.MFlagAttack).Tag(mod.PercentStat("Mana", 1)),
			},
		},
		"kinetic_blast_modifiers_to_number_of_projectiles_instead_apply_to_number_of_clusters": {
			Mods: []mod.Mod{mod.NewFlag("NoAdditionalProjectiles", true)},
		},
	}
	// src/Data/Skills/act_int.lua:11124
	m["KineticFusillade"] = map[string]*StatMap{
		"kinetic_fusillade_base_delay_between_projectiles_ms": {
			Mods: []mod.Mod{skill("delayPerProjectile", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"kinetic_fusillade_damage_+%_final_per_projectile_fired": {
			Mods: []mod.Mod{skill("damagePerProjectile", 0)},
		},
		"kinetic_fusillade_maximum_floating_projectiles": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileCountMaximum", mod.TypeOverride, 0)},
		},
		"quality_display_kinetic_fusillade_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:11342
	m["KineticFusilladeAltX"] = map[string]*StatMap{
		"kinetic_fusillade_base_delay_between_projectiles_ms": {
			Mods: []mod.Mod{skill("delayPerProjectile", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"kinetic_fusillade_damage_+%_final_per_projectile_fired": {
			Mods: []mod.Mod{skill("damagePerProjectile", 0)},
		},
		"kinetic_fusillade_maximum_floating_projectiles": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileCountMaximum", mod.TypeOverride, 0)},
		},
		"quality_display_kinetic_fusillade_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:9968
	m["Lacerate"] = map[string]*StatMap{
		"attack_speed_+%_if_changed_stance_recently": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.Condition("ChangedStanceRecently"))},
		},
	}
	// src/Data/Skills/act_dex.lua:10247
	m["LancingSteel"] = map[string]*StatMap{
		"lancing_steel_damage_+%_at_close_range": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeIncrease, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.DistanceRamp([][]float64{{10, 1}, {70, 0}}))},
		},
		"lancing_steel_damage_+%_final_after_first_hit_on_target": {
			Mods: []mod.Mod{mod.NewFloat("LancingSteelSubsequentDamage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"number_of_projectiles_to_fire_+%_final_per_steel_ammo_consumed": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileCount", mod.TypeMore, 0).Tag(mod.Multiplier("SteelShardConsumed").Limit(4))},
		},
	}
	// src/Data/Skills/act_dex.lua:10350
	m["LancingSteelAltX"] = map[string]*StatMap{
		"lancing_steel_damage_+%_final_after_first_hit_on_target": {
			Mods: []mod.Mod{mod.NewFloat("LancingSteelSubsequentDamage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/other.lua:2435
	m["LightningAegis"] = map[string]*StatMap{
		"aegis_unique_shield_max_value": {
			Mods: []mod.Mod{mod.NewFloat("LightningAegisValue", mod.TypeMAX, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:10497
	m["LightningArrowAltX"] = map[string]*StatMap{
		"lightning_arrow_alt_strike_frequency_ms": {
			Mods: []mod.Mod{skill("hitFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"lightning_arrow_stack_limit": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:LightningArrowofElectrocutionMaxStages", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:12382
	m["LightningConduit"] = map[string]*StatMap{
		"energy_release_damage_+%_final_per_5%_increased_damage_taken_from_shock_on_target": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit).Tag(mod.Multiplier("ShockEffect").Div(5).Actor("enemy"))},
		},
		"quality_display_lightning_conduit_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/minion.lua:459
	m["LightningGolemWrath"] = map[string]*StatMap{
		"attack_maximum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"attack_minimum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"spell_maximum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagSpell).Tag(mod.GlobalEffect("Aura"))},
		},
		"spell_minimum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagSpell).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_int.lua:11809
	m["LightningSpireTrap"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"lightning_tower_trap_base_interval_duration_ms": {
			Mods: []mod.Mod{skill("repeatInterval", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"lightning_tower_trap_number_of_beams": {
			Mods: []mod.Mod{mod.NewFloat("MaximumWaves", mod.TypeBase, 0)},
		},
		"quality_display_lightning_tower_trap_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:12049
	m["LightningSpireTrapAltX"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"lightning_tower_trap_base_interval_duration_ms": {
			Mods: []mod.Mod{skill("repeatInterval", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"lightning_tower_trap_number_of_beams": {
			Mods: []mod.Mod{mod.NewFloat("MaximumWaves", mod.TypeBase, 0)},
		},
		"quality_display_lightning_tower_trap_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:12281
	m["LightningSpireTrapAltY"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"lightning_tower_trap_base_interval_duration_ms": {
			Mods: []mod.Mod{skill("repeatInterval", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"lightning_tower_trap_number_of_beams": {
			Mods: []mod.Mod{mod.NewFloat("MaximumWaves", mod.TypeBase, 0)},
		},
		"quality_display_lightning_tower_trap_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:10706
	m["LightningStrike"] = map[string]*StatMap{
		"active_skill_hit_ailment_damage_with_projectile_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_dex.lua:10807
	m["LightningStrikeAltX"] = map[string]*StatMap{
		"active_skill_hit_ailment_damage_with_projectile_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"skill_damage_+%_final_per_chain_from_skill_specific_stat": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.PerStat(1, "Chain"))},
		},
	}
	// src/Data/Skills/act_int.lua:12550
	m["LightningTendrils"] = map[string]*StatMap{
		"lightning_tendrils_channelled_larger_pulse_always_crit": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeOverride, 0).Tag(mod.SkillPart(3))},
			Base: utils.Ptr(float64(100)),
		},
		"lightning_tendrils_channelled_larger_pulse_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.SkillPart(3))},
		},
		"lightning_tendrils_channelled_larger_pulse_interval": {
			Mods: []mod.Mod{mod.NewFlag("Every3UseCrit", true).Tag(mod.SkillPart(1))},
		},
	}
	// src/Data/Skills/act_int.lua:12674
	m["LightningTendrilsAltX"] = map[string]*StatMap{
		"lightning_tendrils_channelled_larger_pulse_always_crit": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeOverride, 0).Tag(mod.SkillPart(3))},
			Base: utils.Ptr(float64(100)),
		},
		"lightning_tendrils_channelled_larger_pulse_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.SkillPart(3))},
		},
		"lightning_tendrils_channelled_larger_pulse_damage_+%_final": {
			Mods: []mod.Mod{
				mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.SkillPart(3)),
				skill("pulseDamage", 0),
			},
		},
		"lightning_tendrils_channelled_larger_pulse_interval": {
			Mods: []mod.Mod{
				mod.NewFlag("Every5UseCrit", true).Tag(mod.SkillPart(1)),
				skill("pulseInterval", 0),
			},
		},
	}
	// src/Data/Skills/act_int.lua:12770
	m["LightningTendrilsAltY"] = map[string]*StatMap{
		"lightning_tendrils_channelled_base_radius_+_per_second_while_channelling": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(mod.Multiplier("ChannellingTime").Limit(2))},
		},
	}
	// src/Data/Skills/act_int.lua:13355
	m["Malevolence"] = map[string]*StatMap{
		"delirium_aura_damage_over_time_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagDot).Tag(mod.GlobalEffect("Aura"))},
		},
		"delirium_skill_effect_duration_+%": {
			Mods: []mod.Mod{mod.NewFloat("Duration", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_int.lua:13434
	m["ManaInfusedStaff"] = map[string]*StatMap{
		"active_skill_always_reserves_unmodifiable_X%_of_mana": {
			Mods: []mod.Mod{skill("ManaReservationPercentForced", 0)},
		},
		"active_skill_buff_block_chance_+%_final_to_grant": {
			Mods: []mod.Mod{
				mod.NewFloat("BlockChance", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff")),
				mod.NewFloat("SpellBlockChance", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff")),
			},
		},
		"active_skill_buff_mana_cost_+%_final_to_grant": {
			Mods: []mod.Mod{mod.NewFloat("ManaCost", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"mana_infused_staff_base_trigger_interval_ms": {
			Mods: []mod.Mod{skill("hitTimeOverride", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"skill_has_added_lightning_damage_equal_to_%_of_maximum_mana": {
			Mods: []mod.Mod{
				skill("LightningMin", 0).Tag(mod.PercentStat("Mana", 1)),
				skill("LightningMax", 0).Tag(mod.PercentStat("Mana", 1)),
			},
		},
	}
	// src/Data/Skills/act_int.lua:13533
	m["Manabond"] = map[string]*StatMap{
		"mana_void_gain_%_missing_unreserved_mana_as_base_lightning_damage": {
			Mods: []mod.Mod{skill("ManabondMissingManaGainPercent", 0)},
		},
		"quality_display_manabond_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/other.lua:2555
	m["ManifestDancingDervishes"] = map[string]*StatMap{
		"attack_damage_+%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeIncrease, 0).Flag(mod.MFlagAttack)})},
		},
		"attack_speed_+%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack)})},
		},
	}
	// src/Data/Skills/act_str.lua:7541
	m["MoltenShell"] = map[string]*StatMap{
		"base_physical_damage_reduction_rating": {
			Mods: []mod.Mod{mod.NewFloat("Armour", mod.TypeBase, 0).Tag(mod.GlobalEffect("Guard"))},
		},
		"molten_shell_%_of_absorbed_damage_dealt_as_reflected_fire": {
			Mods: []mod.Mod{skill("moltenShellReflect", 0)},
		},
		"molten_shell_damage_absorb_limit_%_of_armour": {
			Mods: []mod.Mod{mod.NewFloat("GuardAbsorbLimit", mod.TypeBase, 0).Tag(mod.PerStat(1, "Armour").LimitVar("GuardAbsorbMax").LimitTotal(true), mod.GlobalEffect("Guard").Unscalable(true))},
			Div:  utils.Ptr(float64(100)),
		},
		"molten_shell_damage_absorbed_%": {
			Mods: []mod.Mod{mod.NewFloat("GuardAbsorbRate", mod.TypeBase, 0).Tag(mod.GlobalEffect("Guard").Unscalable(true))},
		},
		"molten_shell_max_damage_absorbed": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:GuardAbsorbMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Guard").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_str.lua:7820
	m["MoltenStrike"] = map[string]*StatMap{
		"active_skill_hit_ailment_damage_with_projectile_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPartList(2, 3))},
		},
	}
	// src/Data/Skills/act_str.lua:8020
	m["MoltenStrikeAltX"] = map[string]*StatMap{
		"active_skill_hit_ailment_damage_with_projectile_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPartList(2, 3, 4, 5, 6))},
		},
		"molten_strike_every_5th_attack_fire_X_additional_projectiles": {
			Mods: []mod.Mod{
				mod.NewFloat("ProjectileCount", mod.TypeBase, 0).Tag(mod.SkillPartList(4, 5)),
				skill("FifthStrikeProjectiles", 0),
			},
		},
		"molten_strike_every_5th_attack_projectiles_damage_+%_final": {
			Mods: []mod.Mod{
				mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPartList(4, 5)),
				skill("FifthStrikeDamage", 0),
			},
		},
	}
	// src/Data/Skills/spectre.lua:6593
	m["MonsterChanceToTemporalChainsOnHit1"] = map[string]*StatMap{
		"buff_time_passed_+%_other_than_temporal_chains": {
			Mods: []mod.Mod{mod.NewFloat("BuffExpireFaster", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"curse_effect_+%_final_vs_players": {
			Mods: []mod.Mod{mod.NewFloat("CurseEffectAgainstPlayer", mod.TypeMore, 0)},
		},
		"temporal_chains_action_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("TemporalChainsActionSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique").Neg(true))},
		},
		"temporal_chains_action_speed_+%_vs_rare_or_unique_final": {
			Mods: []mod.Mod{mod.NewFloat("TemporalChainsActionSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique"))},
		},
	}
	// src/Data/Skills/spectre.lua:2526
	m["MonsterProjectileWeakness"] = map[string]*StatMap{
		"projectile_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills
	m["MonsterVulnerabilityOnHit1"] = map[string]*StatMap{
		"physical_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"receive_bleeding_chance_%_when_hit_by_attack": {
			Mods: []mod.Mod{mod.NewFloat("SelfBleedChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/spectre.lua:2826
	m["MonsterWarlordsMark"] = map[string]*StatMap{
		"enemy_rage_regeneration_on_stun": {
			Mods: []mod.Mod{mod.NewFlag("Condition:CanGainRage", true).Tag(mod.GlobalEffect("Buff"))},
		},
		"life_leech_on_any_damage_when_hit_by_attack_permyriad": {
			Mods: []mod.Mod{mod.NewFloat("SelfDamageLifeLeech", mod.TypeBase, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Curse"))},
		},
		"mana_leech_on_any_damage_when_hit_by_attack_permyriad": {
			Mods: []mod.Mod{mod.NewFloat("SelfDamageManaLeech", mod.TypeBase, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/spectre.lua:2901
	m["NecromancerConductivity"] = map[string]*StatMap{
		"base_lightning_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("LightningResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"chance_to_be_shocked_%": {
			Mods: []mod.Mod{mod.NewFloat("SelfShockChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/spectre.lua:2939
	m["NecromancerElementalWeakness"] = map[string]*StatMap{
		"base_resist_all_elements_%": {
			Mods: []mod.Mod{mod.NewFloat("ElementalResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/spectre.lua:2973
	m["NecromancerEnfeeble"] = map[string]*StatMap{
		"accuracy_rating_+%": {
			Mods: []mod.Mod{mod.NewFloat("Accuracy", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"enfeeble_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique").Neg(true))},
		},
		"enfeeble_damage_+%_vs_rare_or_unique_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique"))},
		},
	}
	// src/Data/Skills/spectre.lua:3012
	m["NecromancerFlammability"] = map[string]*StatMap{
		"base_fire_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("FireResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"chance_to_be_ignited_%": {
			Mods: []mod.Mod{mod.NewFloat("SelfIgniteChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/spectre.lua:3050
	m["NecromancerFrostbite"] = map[string]*StatMap{
		"base_cold_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("ColdResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"chance_to_be_frozen_%": {
			Mods: []mod.Mod{mod.NewFloat("SelfFreezeChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/spectre.lua:3088
	m["NecromancerProjectileWeakness"] = map[string]*StatMap{
		"projectile_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/spectre.lua:3152
	m["NecromancerVulnerability"] = map[string]*StatMap{
		"physical_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"receive_bleeding_chance_%_when_hit_by_attack": {
			Mods: []mod.Mod{mod.NewFloat("SelfBleedChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_int.lua:13624
	m["OrbOfStorms"] = map[string]*StatMap{
		"orb_of_storms_base_bolt_frequency_ms": {
			Mods: []mod.Mod{skill("hitFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/act_int.lua:13721
	m["OrbOfStormsAltX"] = map[string]*StatMap{
		"orb_of_storms_base_channelling_bolt_frequency_ms": {
			Mods: []mod.Mod{skill("hitTimeOverride", 0).Tag(mod.SkillPart(2))},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/other.lua:2782
	m["PactOfBeidat"] = map[string]*StatMap{
		"pact_skill_additional_beam_only_chains": {
			Mods: []mod.Mod{mod.NewFloat("BeidatAdditionalBeamChains", mod.TypeBase, 0)},
		},
		"pact_skill_damage_+%_final_with_hits_and_ailments_to_grant": {
			Mods: []mod.Mod{mod.NewList("BeidatPactDamage", mod.PactDamage{Mod: mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment)})},
		},
		"pact_skill_grant_x_additional_projectiles_fired_in_nova": {
			Mods: []mod.Mod{mod.NewFloat("BeidatAdditionalProjectiles", mod.TypeBase, 0)},
		},
		"pact_skill_grant_x_cascades_to_do_in_spiral": {
			Mods: []mod.Mod{mod.NewFloat("BeidatAdditionalCascades", mod.TypeBase, 0)},
		},
		"skill_empowers_next_x_spells_cast": {
			Mods: []mod.Mod{mod.NewFloat("BeidatEmpoweredSpells", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/other.lua:2872
	m["PactOfGhorr"] = map[string]*StatMap{
		"pact_skill_grant_damage_over_time_+%_final_to_spells": {
			Mods: []mod.Mod{mod.NewList("GhorrPactDamage", mod.PactDamage{Mod: mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagDot)})},
		},
		"skill_empowers_next_x_spells_cast": {
			Mods: []mod.Mod{mod.NewFloat("GhorrEmpoweredSpells", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/other.lua:3027
	m["PactOfKtash"] = map[string]*StatMap{
		"active_skill_osm_vaal_skill_soul_gain_prevention_+%_final_to_grant": {
			Mods: []mod.Mod{mod.NewFloat("KtashPactSoulGainPrevention", mod.TypeBase, 0)},
		},
		"active_skill_osm_vaal_skill_soul_refund_chance_%_to_grant": {
			Mods: []mod.Mod{mod.NewFloat("KtashPactSoulRefundChance", mod.TypeBase, 0)},
		},
		"pact_skill_grant_damage_+%_final_to_exerted_skills": {
			Mods: []mod.Mod{mod.NewList("KtashPactDamage", mod.PactDamage{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)})},
		},
		"skill_empowers_next_x_spells_cast": {
			Mods: []mod.Mod{mod.NewFloat("KtashEmpoweredSpells", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/other.lua:3113
	m["PactOfLycia"] = map[string]*StatMap{
		"pact_skill_grant_damage_+%_final_to_exerted_skills": {
			Mods: []mod.Mod{mod.NewList("LyciaPactDamage", mod.PactDamage{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)})},
		},
		"skill_empowers_next_x_spells_cast": {
			Mods: []mod.Mod{mod.NewFloat("LyciaEmpoweredSpells", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:13812
	m["PenanceBrand"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:13913
	m["PenanceBrandAltX"] = map[string]*StatMap{
		"magma_brand_ailment_damage_+%_final_per_additional_pustule": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("PenanceBrandofDissipationStageAfterFirst"))},
		},
		"magma_brand_hit_damage_+%_final_per_additional_pustule": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("PenanceBrandofDissipationStageAfterFirst"))},
		},
		"penance_brand_pulses_instead_of_explode": {
			Mods:  []mod.Mod{skill("radiusExtra", 0).Tag(mod.Multiplier("PenanceBrandofDissipationStageAfterFirst"))},
			Value: utils.Ptr(float64(1)),
		},
	}
	// src/Data/Skills/other.lua:2696
	m["PenanceMark"] = map[string]*StatMap{
		"penance_mark_summon_phantasms_when_hit": {
			Mods: []mod.Mod{mod.NewFloat("SummonPhantasmWhenHit", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_str.lua:8244
	m["Perforate"] = map[string]*StatMap{
		"blood_spears_base_number_of_spears": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:PerforateMaxSpikes", mod.TypeBase, 0)},
		},
		"blood_spears_damage_+%_final_in_blood_stance": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Condition("BloodStance"))},
		},
		"quality_display_perforate_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:8340
	m["PerforateAltX"] = map[string]*StatMap{
		"blood_spears_additional_number_of_spears_if_changed_stance_recently": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:PerforateofDualityMaxSpikes", mod.TypeBase, 0).Tag(mod.Condition("ChangedStanceRecently"))},
		},
		"blood_spears_base_number_of_spears": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:PerforateofDualityMaxSpikes", mod.TypeBase, 0)},
		},
		"blood_spears_damage_+%_final_in_blood_stance": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Condition("BloodStance"))},
		},
		"quality_display_perforate_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:8441
	m["PerforateAltY"] = map[string]*StatMap{
		"blood_spears_base_number_of_spears": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:PerforateofBloodshedMaxSpikes", mod.TypeBase, 0)},
		},
		"quality_display_perforate_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:8527
	m["PetrifiedBlood"] = map[string]*StatMap{
		"cannot_recover_above_low_life_except_flasks": {
			Mods: []mod.Mod{},
		},
		"petrified_blood_%_life_loss_below_half_from_hit_to_prevent": {
			Mods: []mod.Mod{mod.NewFloat("LifeLossBelowHalfPrevented", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Petrified Blood"))},
		},
		"petrified_blood_%_prevented_life_loss_to_lose_over_time": {
			Mods: []mod.Mod{mod.NewFloat("LifeLossBelowHalfLost", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Petrified Blood"))},
		},
		"skill_grants_life_cost_%_mana_cost_while_not_on_low_life": {
			Mods: []mod.Mod{mod.NewFloat("ManaCostAsLifeCost", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Petrified Blood").Unscalable(true), mod.Condition("LowLife").Neg(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:11324
	m["PhaseRun"] = map[string]*StatMap{
		"base_movement_velocity_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"phase_run_melee_physical_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0).Flag(mod.MFlagMelee).Tag(mod.SkillType("SummonsTotem").Neg(true), mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/other.lua:2732
	m["PhysicalAegis"] = map[string]*StatMap{
		"aegis_unique_shield_max_value": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalAegisValue", mod.TypeMAX, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:11404
	m["PlagueBearer"] = map[string]*StatMap{
		"corrosive_shroud_poison_damage_+%_final_while_accumulating_poison": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagPoison).Tag(mod.GlobalEffect("Buff").ModCond("PlagueBearerIncubating"))},
		},
	}
	// src/Data/Skills/act_dex.lua:11486
	m["PoachersMark"] = map[string]*StatMap{
		"base_additional_physical_damage_reduction_%": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"life_granted_when_hit_by_attacks": {
			Mods: []mod.Mod{mod.NewFloat("SelfLifeOnHit", mod.TypeBase, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Curse"))},
		},
		"mana_granted_when_hit_by_attacks": {
			Mods: []mod.Mod{mod.NewFloat("SelfManaOnHit", mod.TypeBase, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Curse"))},
		},
		"maximum_added_physical_damage_taken": {
			Mods: []mod.Mod{mod.NewFloat("SelfPhysicalMax", mod.TypeBase, 0).Flag(mod.MFlagHit).Tag(mod.GlobalEffect("Curse"))},
		},
		"minimum_added_physical_damage_taken": {
			Mods: []mod.Mod{mod.NewFloat("SelfPhysicalMin", mod.TypeBase, 0).Flag(mod.MFlagHit).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_dex.lua:17426
	m["PoisonousConcoction"] = map[string]*StatMap{
		"flask_throw_added_chaos_damage_%_of_flask_life_to_recover": {
			Mods: []mod.Mod{mod.NewFloat("ChaosPerLifeFlaskPercent", mod.TypeBase, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_dex.lua:17527
	m["PoisonousConcoctionAltX"] = map[string]*StatMap{
		"flask_throw_added_chaos_damage_%_of_flask_life_to_recover": {
			Mods: []mod.Mod{mod.NewFloat("ChaosPerLifeFlaskPercent", mod.TypeBase, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_int.lua:14108
	m["PowerSiphon"] = map[string]*StatMap{
		"critical_strike_chance_+%_final_per_power_charge_from_power_siphon": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeMore, 0).Tag(mod.Multiplier("PowerCharge"))},
		},
		"power_siphon_base_fire_at_x_targets": {
			Mods: []mod.Mod{mod.NewFlag("OneShotProj", true)},
		},
	}
	// src/Data/Skills/act_int.lua:14192
	m["PowerSiphonAltX"] = map[string]*StatMap{
		"power_siphon_base_fire_at_x_targets": {
			Mods: []mod.Mod{mod.NewFlag("OneShotProj", true)},
		},
	}
	// src/Data/Skills/act_dex.lua:11577
	m["Precision"] = map[string]*StatMap{
		"accuracy_rating": {
			Mods: []mod.Mod{mod.NewFloat("Accuracy", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"skill_buff_grants_critical_strike_chance_+%": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_str.lua:8605
	m["Pride"] = map[string]*StatMap{
		"physical_damage_aura_nearby_enemies_physical_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeMore, 0).Tag(mod.GlobalEffect("AuraDebuff").ModCond("PrideMinEffect"))},
		},
		"physical_damage_aura_nearby_enemies_physical_damage_taken_+%_max": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeMore, 0).Tag(mod.GlobalEffect("AuraDebuff").ModCond("PrideMaxEffect"))},
		},
	}
	// src/Data/Skills/sup_int.lua:5426
	m["PrismaticBurst"] = map[string]*StatMap{
		"prismatic_burst_unchosen_type_damage_-100%_final": {
			Mods: []mod.Mod{
				mod.NewFloat("FireDamage", mod.TypeMore, 0).Tag(mod.SkillPartList(2, 3)),
				mod.NewFloat("ColdDamage", mod.TypeMore, 0).Tag(mod.SkillPartList(1, 3)),
				mod.NewFloat("LightningDamage", mod.TypeMore, 0).Tag(mod.SkillPartList(1, 2)),
			},
			Mult: utils.Ptr(float64(-100)),
		},
	}
	// src/Data/Skills/act_str.lua:8684
	m["ProtectiveLink"] = map[string]*StatMap{
		"bulwark_link_grants_recover_X_life_on_block": {
			Mods: []mod.Mod{mod.NewFloat("LifeOnBlock", mod.TypeBase, 0).Tag(mod.GlobalEffect("Link"))},
		},
		"display_bulwark_link_overrides_attack_block_and_maximum_attack_block": {
			Mods: []mod.Mod{
				mod.NewFlag("BlockAttackChanceIsEqualToParent", true).Tag(mod.GlobalEffect("Link")),
				mod.NewFlag("MaximumBlockAttackChanceIsEqualToParent", true).Tag(mod.GlobalEffect("Link")),
			},
		},
	}
	// src/Data/Skills/act_dex.lua:12064
	m["Puncture"] = map[string]*StatMap{
		"quality_display_active_skill_bleed_damage_final_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:12152
	m["PunctureAltX"] = map[string]*StatMap{
		"quality_display_active_skill_bleed_damage_final_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:8762
	m["Punishment"] = map[string]*StatMap{
		"damage_taken_+%_on_low_life": {
			Mods: []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("LowLife"))},
		},
	}
	// src/Data/Skills/act_int.lua:14367
	m["PurifyingFlame"] = map[string]*StatMap{
		"quality_display_sanctify_is_gem": {
			Mods: []mod.Mod{},
		},
		"sanctify_wave_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_int.lua:14471
	m["PurifyingFlameAltX"] = map[string]*StatMap{
		"quality_display_sanctify_is_gem": {
			Mods: []mod.Mod{},
		},
		"sanctify_wave_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_int.lua:14562
	m["PurityOfElements"] = map[string]*StatMap{
		"base_resist_all_elements_%": {
			Mods: []mod.Mod{mod.NewFloat("ElementalResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"immune_to_status_ailments": {
			Mods: []mod.Mod{mod.NewFlag("ElementalAilmentImmune", true).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_str.lua:8847
	m["PurityOfFire"] = map[string]*StatMap{
		"base_fire_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("FireResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_maximum_fire_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("FireResistMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_dex.lua:12232
	m["PurityOfIce"] = map[string]*StatMap{
		"base_cold_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("ColdResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_maximum_cold_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("ColdResistMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_int.lua:14641
	m["PurityOfLightning"] = map[string]*StatMap{
		"base_lightning_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("LightningResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_maximum_lightning_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("LightningResistMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_int.lua:14808
	m["PyroclastMine"] = map[string]*StatMap{
		"pyroclast_mine_aura_fire_exposure_%_to_apply": {
			Mods: []mod.Mod{mod.NewFloat("FireExposure", mod.TypeBase, 0).Tag(mod.GlobalEffect("AuraDebuff").Name("Pyroclast Mine Fire Exposure"))},
		},
	}
	// src/Data/Skills/act_int.lua:14906
	m["PyroclastMineAltX"] = map[string]*StatMap{
		"pyroclast_mine_aura_fire_exposure_%_to_apply": {
			Mods: []mod.Mod{mod.NewFloat("FireExposure", mod.TypeBase, 0).Tag(mod.GlobalEffect("AuraDebuff").Name("Pyroclast Mine Fire Exposure"))},
		},
	}
	// src/Data/Skills/act_str.lua:10951
	m["QuickGuard"] = map[string]*StatMap{
		"quick_guard_damage_absorb_limit": {
			Mods: []mod.Mod{mod.NewFloat("GuardAbsorbLimit", mod.TypeBase, 0).Tag(mod.GlobalEffect("Guard").Unscalable(true))},
		},
		"quick_guard_damage_absorbed_%": {
			Mods: []mod.Mod{mod.NewFloat("GuardAbsorbRate", mod.TypeBase, 0).Tag(mod.GlobalEffect("Guard").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_str.lua:9037
	m["RageVortex"] = map[string]*StatMap{
		"quality_display_rage_vortex_is_gem": {
			Mods: []mod.Mod{},
		},
		"rage_slash_damage_+%_final_per_amount_of_rage_sacrificed": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("RageSacrificed").DivVar("RagePerDamage"))},
		},
		"rage_slash_radius_+_per_amount_of_rage_sacrificed": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeBase, 0).Tag(mod.Multiplier("RageSacrificed").DivVar("RagePerRadius"))},
		},
		"rage_slash_rage_sacrifice_per_damage_bonus": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:RagePerDamage", mod.TypeBase, 0)},
		},
		"rage_slash_rage_sacrifice_per_radius_bonus": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:RagePerRadius", mod.TypeBase, 0)},
		},
		"rage_slash_sacrifice_rage_%": {
			Mods: []mod.Mod{skill("MaxRageVortexSacrificePercentage", 0)},
		},
		"rage_slash_vortex_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_str.lua:9152
	m["RageVortexAltX"] = map[string]*StatMap{
		"rage_slash_vortex_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_int.lua:15010
	m["RaiseSpectre"] = map[string]*StatMap{
		"raised_spectre_level": {
			Mods: []mod.Mod{skill("minionLevel", 0)},
		},
	}
	// src/Data/Skills/act_int.lua:15090
	m["RaiseSpectreAltX"] = map[string]*StatMap{
		"raised_spectre_level": {
			Mods: []mod.Mod{skill("minionLevel", 0)},
		},
		"spectres_have_base_duration_ms": {
			Mods: []mod.Mod{skill("duration", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/other.lua:3410
	m["RaiseSpiders"] = map[string]*StatMap{
		"summoned_spider_grants_attack_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.Multiplier("RaisedSpider"), mod.GlobalEffect("Buff").Name("Raised Spider"))},
		},
		"summoned_spider_grants_poison_damage_+%": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeIncrease, 0).KeywordFlag(mod.KeywordFlagPoison).Tag(mod.Multiplier("RaisedSpider"), mod.GlobalEffect("Buff").Name("Raised Spider"))},
		},
	}
	// src/Data/Skills/act_int.lua:15176
	m["RaiseZombie"] = map[string]*StatMap{
		"quality_display_raise_zombie_is_gem+%": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:15258
	m["RaiseZombieAltX"] = map[string]*StatMap{
		"quality_display_raise_zombie_is_gem+%": {
			Mods: []mod.Mod{},
		},
		"zombie_slam_area_of_effect_+%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0).Tag(mod.SkillId("ZombieSlam"))})},
		},
		"zombie_slam_cooldown_speed_+%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("CooldownRecovery", mod.TypeIncrease, 0).Tag(mod.SkillId("ZombieSlam"))})},
		},
	}
	// src/Data/Skills/act_int.lua:15345
	m["RaiseZombieAltY"] = map[string]*StatMap{
		"minion_global_always_hit": {
			Mods: []mod.Mod{},
		},
		"quality_display_raise_zombie_is_gem+%": {
			Mods: []mod.Mod{},
		},
		"raise_zombie_does_not_use_corpses": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:9229
	m["RallyingCry"] = map[string]*StatMap{
		"rallying_cry_buff_effect_on_minions_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("RallyingCryMinionDamageBonusMultiplier", mod.TypeBase, 0)},
			Div:  utils.Ptr(float64(100)),
		},
		"rallying_cry_damage_+%_final_from_osm_per_nearby_ally": {
			Mods: []mod.Mod{mod.NewFloat("RallyingCryExertDamageBonus", mod.TypeBase, 0)},
		},
		"rallying_cry_weapon_damage_%_for_allies_per_5_monster_power": {
			Mods: []mod.Mod{
				mod.NewFloat("RallyingCryAllyDamageBonusPer5Power", mod.TypeBase, 0),
				mod.NewFloat("Dummy", mod.TypeDummy, 1).Tag(mod.GlobalEffect("Warcry"), mod.Multiplier("NearbyAlly")),
			},
		},
		"skill_empowers_next_x_melee_attacks": {
			Mods: []mod.Mod{mod.NewFloat("RallyingExertedAttacks", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/other.lua:3374
	m["Ravenous"] = map[string]*StatMap{
		"ravenous_buff_magnitude": {
			Groups: []*StatMap{
				{
					Mods: []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff").EffectCond("RavenousCorpseConsumed"))},
					Mult: utils.Ptr(float64(-1)),
				},
				{
					Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff").EffectCond("RavenousCorpseConsumed"))},
				},
			},
		},
	}
	// src/Data/Skills/act_str.lua:11032
	m["Reap"] = map[string]*StatMap{
		"base_physical_damage_to_deal_per_minute": {
			Mods: []mod.Mod{
				skill("PhysicalDot", 0).Tag(mod.Condition("ReapDebuffIsFireDamage").Neg(true)),
				skill("FireDot", 0).Tag(mod.Condition("ReapDebuffIsFireDamage")),
			},
			Div: utils.Ptr(float64(60)),
		},
		"blood_scythe_cost_+%_final_per_charge": {
			Mods: []mod.Mod{mod.NewFloat("LifeCost", mod.TypeMore, 0).Tag(mod.Multiplier("BloodCharge"))},
		},
		"blood_scythe_damage_+%_final_per_charge": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("BloodCharge"))},
		},
		"quality_display_reap_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:11132
	m["ReapAltX"] = map[string]*StatMap{
		"base_physical_damage_to_deal_per_minute": {
			Mods: []mod.Mod{
				skill("PhysicalDot", 0).Tag(mod.Condition("ReapDebuffIsFireDamage").Neg(true)),
				skill("FireDot", 0).Tag(mod.Condition("ReapDebuffIsFireDamage")),
			},
			Div: utils.Ptr(float64(60)),
		},
		"blood_scythe_cost_+%_final_per_charge": {
			Mods: []mod.Mod{mod.NewFloat("LifeCost", mod.TypeMore, 0).Tag(mod.Multiplier("BloodCharge"))},
		},
		"blood_scythe_damage_+%_final_per_charge": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("BloodCharge"))},
		},
		"quality_display_reap_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:12716
	m["Reave"] = map[string]*StatMap{
		"reave_additional_max_stacks": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ReaveMaxStages", mod.TypeBase, 0)},
		},
		"reave_area_of_effect_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.Multiplier("ReaveStage"))},
		},
	}
	// src/Data/Skills/act_dex.lua:12805
	m["ReaveAltX"] = map[string]*StatMap{
		"reave_additional_max_stacks": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ReaveofRefractionMaxStages", mod.TypeBase, 0)},
		},
		"reave_area_of_effect_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.Multiplier("ReaveofRefractionStage"))},
		},
	}
	// src/Data/Skills/act_str.lua:9327
	m["RejuvenationTotem"] = map[string]*StatMap{
		"base_life_regeneration_rate_per_minute": {
			Groups: []*StatMap{
				{
					Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
					Div:  utils.Ptr(float64(60)),
				},
				{
					Mods: []mod.Mod{mod.NewFloat("ManaRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"), mod.Condition("RejuvenationTotemManaRegen"))},
					Div:  utils.Ptr(float64(400)),
				},
			},
		},
	}
	// src/Data/Skills/other.lua:764
	m["ResentmentUniqueSkill"] = map[string]*StatMap{
		"base_fire_damage_to_deal_per_minute": {
			Mods: []mod.Mod{skill("FireDot", 0).Tag(mod.Multiplier("CinderflameStacks").Limit(10))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/act_int.lua:15435
	m["RighteousFire"] = map[string]*StatMap{
		"base_nonlethal_fire_damage_%_of_maximum_energy_shield_taken_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("FireDegen", mod.TypeBase, 0).Tag(mod.PerStat(1, "EnergyShield"), mod.GlobalEffect("Buff"), mod.StatThreshold("LifeUnreserved", 2))},
			Div:  utils.Ptr(float64(6000)),
		},
		"base_nonlethal_fire_damage_%_of_maximum_life_taken_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("FireDegen", mod.TypeBase, 0).Tag(mod.PerStat(1, "Life"), mod.GlobalEffect("Buff"), mod.StatThreshold("LifeUnreserved", 2))},
			Div:  utils.Ptr(float64(6000)),
		},
		"base_righteous_fire_%_of_max_energy_shield_to_deal_to_nearby_per_minute": {
			Mods: []mod.Mod{skill("RFESMultiplier", 0)},
			Div:  utils.Ptr(float64(6000)),
		},
		"base_righteous_fire_%_of_max_life_to_deal_to_nearby_per_minute": {
			Mods: []mod.Mod{skill("RFLifeMultiplier", 0)},
			Div:  utils.Ptr(float64(6000)),
		},
		"righteous_fire_spell_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell).Tag(mod.GlobalEffect("Buff"), mod.StatThreshold("LifeUnreserved", 2))},
		},
	}
	// src/Data/Skills/act_int.lua:15538
	m["RighteousFireAltX"] = map[string]*StatMap{
		"base_nonlethal_fire_damage_%_of_maximum_mana_taken_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("FireDegen", mod.TypeBase, 0).Tag(mod.PerStat(1, "Mana"), mod.GlobalEffect("Buff"), mod.StatThreshold("LifeUnreserved", 2), mod.StatThreshold("Mana", 1))},
			Div:  utils.Ptr(float64(6000)),
		},
		"base_righteous_fire_%_of_max_mana_to_deal_to_nearby_per_minute": {
			Mods: []mod.Mod{skill("RFManaMultiplier", 0)},
			Div:  utils.Ptr(float64(6000)),
		},
		"righteous_fire_cast_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagCast).Tag(mod.GlobalEffect("Buff"), mod.StatThreshold("LifeUnreserved", 2), mod.StatThreshold("Mana", 1))},
		},
	}
	// src/Data/Skills/act_int.lua:15731
	m["ScorchingRay"] = map[string]*StatMap{
		"base_fire_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("FireExposure", mod.TypeBase, 0).Tag(mod.GlobalEffect("Debuff").Name("Fire Exposure").EffectCond("ScorchingRayMaxStages"))},
		},
		"display_max_fire_beam_stacks": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ScorchingRayMaxStages", mod.TypeBase, 0).Tag(mod.SkillPart(1))},
		},
		"fire_beam_additional_stack_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("ScorchingRayStageAfterFirst"))},
			Base: utils.Ptr(float64(100)),
		},
		"quality_display_firebeam_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:15833
	m["ScorchingRayAltX"] = map[string]*StatMap{
		"display_max_fire_beam_stacks": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ScorchingRayofImmolationMaxStages", mod.TypeBase, 0).Tag(mod.SkillPart(1))},
		},
		"fire_beam_additional_stack_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("ScorchingRayofImmolationStageAfterFirst"))},
			Base: utils.Ptr(float64(100)),
		},
		"quality_display_firebeam_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:13000
	m["ScourgeArrow"] = map[string]*StatMap{
		"base_arrows_always_pierce": {
			Mods: []mod.Mod{mod.NewFlag("PierceAllTargets", true)},
		},
		"virulent_arrow_damage_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("ScourgeArrowStage"))},
		},
		"virulent_arrow_maximum_number_of_stacks": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ScourgeArrowMaxStages", mod.TypeBase, 0)},
		},
		"virulent_arrow_pod_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_dex.lua:13097
	m["ScourgeArrowAltX"] = map[string]*StatMap{
		"base_arrows_always_pierce": {
			Mods: []mod.Mod{mod.NewFlag("PierceAllTargets", true)},
		},
		"virulent_arrow_damage_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("ScourgeArrowStage"))},
		},
		"virulent_arrow_pod_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_str.lua:9589
	m["SearingBondAltX"] = map[string]*StatMap{
		"number_of_additional_totems_allowed_per_maximum_power_charge": {
			Mods: []mod.Mod{mod.NewFloat("ActiveTotemLimit", mod.TypeBase, 0).Tag(mod.PerStat(1, "PowerChargesMax"))},
		},
		"searing_bond_totems_detonation_damage_over_time_+%_final_per_active_totem": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagDot).Tag(mod.PerStat(1, "TotemsSummoned"), mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/spectre.lua:3382
	m["SeawitchVulnerability"] = map[string]*StatMap{
		"physical_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"receive_bleeding_chance_%_when_hit_by_attack": {
			Mods: []mod.Mod{mod.NewFloat("SelfBleedChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_str.lua:9675
	m["SeismicCry"] = map[string]*StatMap{
		"seismic_cry_armour_+%_final_per_5_power_up_to_cap": {
			Mods: []mod.Mod{mod.NewFloat("Armour", mod.TypeMore, 0).Tag(mod.GlobalEffect("Warcry").Div(5).Limit(25))},
		},
		"seismic_cry_base_slam_skill_area_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("SeismicAoEMoreMultiplier", mod.TypeBase, 0)},
		},
		"seismic_cry_stun_threshold_+%_per_5_power_up_to_cap": {
			Mods: []mod.Mod{mod.NewFloat("StunThreshold", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Warcry").Div(5).Limit(25))},
		},
		"skill_empower_limitation_specifier_for_stat_description": {
			Mods: []mod.Mod{},
		},
		"skill_empowers_next_x_melee_attacks": {
			Mods: []mod.Mod{mod.NewFloat("SeismicExertedAttacks", mod.TypeBase, 0).Tag(mod.SkillType("Slam"))},
		},
	}
	// src/Data/Skills/act_dex.lua:13514
	m["SeismicTrap"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"phys_cascade_trap_base_interval_duration_ms": {
			Mods: []mod.Mod{skill("repeatInterval", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"phys_cascade_trap_number_of_cascades": {
			Mods: []mod.Mod{mod.NewFloat("MaximumWaves", mod.TypeBase, 0)},
		},
		"quality_display_phys_cascade_trap_is_gem": {
			Mods: []mod.Mod{},
		},
		"seismic_trap_frequency_+%": {
			Mods: []mod.Mod{mod.NewFloat("SeismicPulseFrequency", mod.TypeIncrease, 0)},
		},
	}
	// src/Data/Skills/act_dex.lua:13680
	m["SeismicTrapAltX"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"phys_cascade_trap_number_of_cascades": {
			Mods: []mod.Mod{mod.NewFloat("MaximumWaves", mod.TypeBase, 0)},
		},
		"quality_display_phys_cascade_trap_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:13284
	m["ShatteringSteelAltX"] = map[string]*StatMap{
		"additional_block_chance_against_projectiles_%_per_steel_charge": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileBlockChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("SteelWardCount").Limit(6))},
		},
		"fires_1_projectile_if_no_steel_ammo": {
			Mods: []mod.Mod{mod.NewFlag("NoAdditionalProjectiles", true).Tag(mod.MultiplierThreshold("SteelShardConsumed").Threshold(0).Upper(true))},
		},
		"shattering_steel_hit_damage_+%_final_scaled_by_projectile_distance_per_ammo_consumed": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(multiplierLimitVar("SteelShardConsumed", "MaxSteelShardsConsumed"), mod.DistanceRamp([][]float64{{10, 1}, {70, 0}}))},
		},
		"steel_ammo_consumed_per_use": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:MaxSteelShardsConsumed", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_str.lua:9874
	m["ShieldCrush"] = map[string]*StatMap{
		"shield_crush_damage_+%_final_from_distance": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.MeleeProximity([]int{1, 0}))},
		},
	}
	// src/Data/Skills/act_str.lua:9968
	m["ShieldCrushAltX"] = map[string]*StatMap{
		"shield_crush_damage_+%_final_from_distance": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.MeleeProximity([]int{1, 0}))},
		},
	}
	// src/Data/Skills/other.lua:3668
	m["ShieldShatter"] = map[string]*StatMap{
		"buff_added_spell_maximum_base_physical_damage_per_shield_quality": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMax", mod.TypeBase, 0).Flag(mod.MFlagSpell).Tag(mod.Multiplier("QualityOnWeapon 2"), mod.Condition("BlockedRecently"), mod.GlobalEffect("Buff"))},
		},
		"buff_added_spell_minimum_base_physical_damage_per_shield_quality": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMin", mod.TypeBase, 0).Flag(mod.MFlagSpell).Tag(mod.Multiplier("QualityOnWeapon 2"), mod.Condition("BlockedRecently"), mod.GlobalEffect("Buff"))},
		},
		"spell_maximum_base_physical_damage_per_shield_quality": {
			Mods: []mod.Mod{skill("PhysicalMax", 0).Tag(mod.Multiplier("QualityOnWeapon 2"))},
		},
		"spell_minimum_base_physical_damage_per_shield_quality": {
			Mods: []mod.Mod{skill("PhysicalMin", 0).Tag(mod.Multiplier("QualityOnWeapon 2"))},
		},
	}
	// src/Data/Skills/act_int.lua:15933
	m["ShockNova"] = map[string]*StatMap{
		"shock_nova_ring_chance_to_shock_+%": {
			Mods: []mod.Mod{mod.NewFloat("EnemyShockChance", mod.TypeBase, 0).Tag(mod.SkillPartList(1, 2))},
		},
	}
	// src/Data/Skills/act_dex.lua:13873
	m["ShrapnelBallistaAltX"] = map[string]*StatMap{
		"fires_1_projectile_if_no_steel_ammo": {
			Mods: []mod.Mod{mod.NewFlag("NoAdditionalProjectiles", true).Tag(mod.MultiplierThreshold("SteelShardConsumed").Threshold(0).Upper(true))},
		},
	}
}

func skillStatMapsPart3(m map[string]map[string]*StatMap) {
	// src/Data/Skills/act_dex.lua:13961
	m["SiegeBallista"] = map[string]*StatMap{
		"attack_speed_+%_per_maximum_totem": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.PerStat(1, "ActiveTotemLimit"))},
		},
	}
	// src/Data/Skills/act_dex.lua:14127
	m["SiegeBallistaAltY"] = map[string]*StatMap{
		"active_skill_summon_X_totems_in_ring_override": {
			Mods: []mod.Mod{skill("dpsMultiplier", 0)},
		},
		"totem_fire_in_formation": {
			Mods: []mod.Mod{skill("totemFireOnce", 1)},
		},
	}
	// src/Data/Skills/act_int.lua:16101
	m["SigilOfPower"] = map[string]*StatMap{
		"circle_of_power_enemy_damage_+%_final_at_max_stages": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.MultiplierThreshold("SigilOfPowerStage").ThresholdVar("SigilOfPowerMaxStages").Actor("enemy"), mod.GlobalEffect("Debuff").Name("Sigil of Power"))},
		},
		"circle_of_power_max_added_lightning_per_stage": {
			Mods: []mod.Mod{
				mod.NewFloat("LightningMax", mod.TypeBase, 0).Tag(multiplierLimitVar("SigilOfPowerStage", "SigilOfPowerMaxStages"), mod.GlobalEffect("Buff").Name("Sigil of Power")),
				mod.NewFloat("LightningMax", mod.TypeBase, 0).Tag(multiplierLimitVar("SigilOfPowerStage", "SigilOfPowerMaxStages").Actor("parent"), mod.GlobalEffect("Buff").Name("Sigil of Power")),
			},
		},
		"circle_of_power_max_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:SigilOfPowerMaxStages", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"circle_of_power_min_added_lightning_per_stage": {
			Mods: []mod.Mod{
				mod.NewFloat("LightningMin", mod.TypeBase, 0).Tag(multiplierLimitVar("SigilOfPowerStage", "SigilOfPowerMaxStages"), mod.GlobalEffect("Buff").Name("Sigil of Power")),
				mod.NewFloat("LightningMin", mod.TypeBase, 0).Tag(multiplierLimitVar("SigilOfPowerStage", "SigilOfPowerMaxStages").Actor("parent"), mod.GlobalEffect("Buff").Name("Sigil of Power")),
			},
		},
		"quality_display_circle_of_power_damage_is_gem": {
			Mods: []mod.Mod{},
		},
		"quality_display_circle_of_power_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:16201
	m["SiphoningTrap"] = map[string]*StatMap{
		"skill_life_regeneration_per_minute_per_affected_enemy": {
			Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").EffectCond("SiphoningTrapSiphoning"), mod.Multiplier("EnemyAffectedBySiphoningTrap").Limit(10))},
			Div:  utils.Ptr(float64(60)),
		},
		"skill_life_regeneration_per_minute_with_at_least_1_affected_enemy": {
			Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").EffectCond("SiphoningTrapSiphoning"), mod.MultiplierThreshold("EnemyAffectedBySiphoningTrap").Threshold(1))},
			Div:  utils.Ptr(float64(60)),
		},
		"skill_mana_regeneration_per_minute_per_affected_enemy": {
			Mods: []mod.Mod{mod.NewFloat("ManaRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").EffectCond("SiphoningTrapSiphoning"), mod.Multiplier("EnemyAffectedBySiphoningTrap").Limit(10))},
			Div:  utils.Ptr(float64(60)),
		},
		"skill_mana_regeneration_per_minute_with_at_least_1_affected_enemy": {
			Mods: []mod.Mod{mod.NewFloat("ManaRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").EffectCond("SiphoningTrapSiphoning"), mod.MultiplierThreshold("EnemyAffectedBySiphoningTrap").Threshold(1))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/spectre.lua:3723
	m["SkeletonVulnerability"] = map[string]*StatMap{
		"physical_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"receive_bleeding_chance_%_when_hit_by_attack": {
			Mods: []mod.Mod{mod.NewFloat("SelfBleedChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_str.lua:10325
	m["Smite"] = map[string]*StatMap{
		"base_chance_to_shock_%_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("EnemyShockChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"), mod.Condition("AffectedByVaalSmite").Neg(true))},
		},
		"maximum_added_lightning_damage_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"), mod.Condition("AffectedByVaalSmite").Neg(true))},
		},
		"minimum_added_lightning_damage_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"), mod.Condition("AffectedByVaalSmite").Neg(true))},
		},
		"quality_display_active_skill_area_damage_quality_negated_from_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:14213
	m["SmokeMine"] = map[string]*StatMap{
		"base_movement_velocity_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_dex.lua:17617
	m["Snipe"] = map[string]*StatMap{
		"quality_display_snipe_is_gem": {
			Mods: []mod.Mod{},
		},
		"snipe_max_stacks": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:SnipeStagesMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:11657
	m["SnipersMark"] = map[string]*StatMap{
		"projectile_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"projectiles_hitting_self_split_into_x": {
			Mods: []mod.Mod{mod.NewFloat("SelfSplitCount", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_int.lua:17211
	m["SoulLink"] = map[string]*StatMap{
		"soul_link_grants_damage_taken_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeMore, 0).Tag(mod.GlobalEffect("Link"))},
		},
		"soul_link_grants_mana_regeneration_+%": {
			Mods: []mod.Mod{mod.NewFloat("ManaRegen", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Link"))},
		},
		"soul_link_grants_take_%_of_hit_damage_from_soul_link_source_energy_shield_before_you": {
			Mods: []mod.Mod{mod.NewFloat("TakenFromParentESBeforeYou", mod.TypeBase, 0).Tag(mod.GlobalEffect("Link"))},
		},
	}
	// src/Data/Skills/act_dex.lua:15217
	m["SpectralHelix"] = map[string]*StatMap{
		"spectral_spiral_weapon_base_number_of_bounces": {
			Mods: []mod.Mod{mod.NewFloat("BounceCount", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_dex.lua:15308
	m["SpectralHelixAltY"] = map[string]*StatMap{
		"spectral_spiral_weapon_base_number_of_bounces": {
			Mods: []mod.Mod{mod.NewFloat("BounceCount", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_dex.lua:14310
	m["SpectralShieldThrow"] = map[string]*StatMap{
		"off_hand_maximum_added_physical_damage_per_15_shield_armour_and_evasion_rating": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMax", mod.TypeBase, 0).Tag(mod.Condition("OffHandAttack"), mod.Condition("ShieldThrowCrushNoArmourEvasion").Neg(true), mod.PerStat(15, "ArmourOnWeapon 2", "EvasionOnWeapon 2"))},
		},
		"off_hand_minimum_added_physical_damage_per_15_shield_armour_and_evasion_rating": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMin", mod.TypeBase, 0).Tag(mod.Condition("OffHandAttack"), mod.Condition("ShieldThrowCrushNoArmourEvasion").Neg(true), mod.PerStat(15, "ArmourOnWeapon 2", "EvasionOnWeapon 2"))},
		},
		"primary_projectile_chains_+": {
			Mods: []mod.Mod{mod.NewFloat("ChainCountMax", mod.TypeBase, 0).Tag(mod.SkillPart(1))},
		},
		"thrown_shield_secondary_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_dex.lua:14410
	m["SpectralShieldThrowAltX"] = map[string]*StatMap{
		"thrown_shield_secondary_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_dex.lua:14497
	m["SpectralShieldThrowAltY"] = map[string]*StatMap{
		"stationary_shield_throw_firing_interval_ms": {
			Mods: []mod.Mod{skill("hitTimeOverride", 0).Tag(mod.SkillPart(2))},
			Div:  utils.Ptr(float64(1000)),
		},
		"thrown_shield_secondary_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_int.lua:17060
	m["Spellslinger"] = map[string]*StatMap{
		"spellslinger_mana_reservation": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:17292
	m["SpiritOffering"] = map[string]*StatMap{
		"spirit_offering_critical_strike_chance_+%": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"spirit_offering_critical_strike_multiplier_+": {
			Mods: []mod.Mod{mod.NewFloat("CritMultiplier", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/other.lua:3845
	m["StalkingPustule"] = map[string]*StatMap{
		"volatile_dead_core_explosion_radius_+": {
			Mods: []mod.Mod{skill("radius", 0)},
		},
	}
	// src/Data/Skills/act_str.lua:10678
	m["StaticStrike"] = map[string]*StatMap{
		"static_strike_base_zap_frequency_ms": {
			Mods: []mod.Mod{skill("repeatFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"static_strike_beam_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
		"static_strike_zap_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("StaticStrikeFrequency", mod.TypeIncrease, 0)},
		},
	}
	// src/Data/Skills/act_str.lua:10777
	m["StaticStrikeAltX"] = map[string]*StatMap{
		"hit_and_ailment_damage_+%_final_per_gathering_lightning": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("StaticStrikeofGatheringLightningStage"))},
		},
		"static_strike_maximum_gathering_lightning_stacks": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:StaticStrikeofGatheringLightningMaxStages", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:17376
	m["StormBrand"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:17473
	m["StormBrandAltX"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:18049
	m["StormBurst"] = map[string]*StatMap{
		"display_storm_burst_jump_time_ms": {
			Mods: []mod.Mod{skill("repeatFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"storm_burst_new_damage_+%_final_per_remaining_teleport_zap": {
			Mods: []mod.Mod{skill("remainingDurationDamage", 0)},
		},
	}
	// src/Data/Skills/act_int.lua:18158
	m["StormBurstAltX"] = map[string]*StatMap{
		"display_storm_burst_jump_time_ms": {
			Mods: []mod.Mod{skill("repeatFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"storm_burst_new_damage_+%_final_per_remaining_teleport_zap": {
			Mods: []mod.Mod{skill("remainingDurationDamage", 0)},
		},
	}
	// src/Data/Skills/act_int.lua:18252
	m["StormCall"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:18336
	m["StormCallAltX"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:11750
	m["StormRain"] = map[string]*StatMap{
		"number_of_allowed_storm_arrows": {
			Mods: []mod.Mod{},
		},
		"prismatic_rain_beam_base_frequency_ms": {
			Mods: []mod.Mod{skill("hitFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"prismatic_rain_beam_frequency_+%": {
			Mods: []mod.Mod{mod.NewFloat("StormRainBeamFrequency", mod.TypeIncrease, 0)},
		},
		"quality_display_storm_rain_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:11862
	m["StormRainAltX"] = map[string]*StatMap{
		"number_of_allowed_storm_arrows": {
			Mods: []mod.Mod{mod.NewFloat("StormRainAllowedStormArrows", mod.TypeBase, 0)},
		},
		"prismatic_rain_beam_base_frequency_ms": {
			Mods: []mod.Mod{skill("hitFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"prismatic_rain_beam_frequency_+%": {
			Mods: []mod.Mod{mod.NewFloat("StormRainBeamFrequency", mod.TypeIncrease, 0)},
		},
		"quality_display_storm_rain_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:11970
	m["StormRainAltY"] = map[string]*StatMap{
		"number_of_allowed_storm_arrows": {
			Mods: []mod.Mod{mod.NewFloat("StormRainAllowedStormArrows", mod.TypeBase, 0)},
		},
		"prismatic_rain_beam_base_frequency_ms": {
			Mods: []mod.Mod{skill("hitFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"prismatic_rain_beam_frequency_+%": {
			Mods: []mod.Mod{mod.NewFloat("StormRainBeamFrequency", mod.TypeIncrease, 0)},
		},
		"quality_display_storm_rain_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:17673
	m["Stormbind"] = map[string]*StatMap{
		"quality_display_rune_paint_area_is_gem": {
			Mods: []mod.Mod{},
		},
		"quality_display_rune_paint_is_gem": {
			Mods: []mod.Mod{},
		},
		"rune_paint_area_of_effect_+%_final_per_rune_level": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.Multiplier("RuneLevel"))},
		},
		"rune_paint_area_of_effect_+%_per_rune_level": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0).Tag(mod.Multiplier("RuneLevel"))},
		},
		"rune_paint_damage_+%_final_per_rune_level": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("RuneLevel"))},
		},
		"rune_paint_max_rune_level": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:17855
	m["StormbindAltX"] = map[string]*StatMap{
		"quality_display_rune_paint_area_is_gem": {
			Mods: []mod.Mod{},
		},
		"quality_display_rune_paint_is_gem": {
			Mods: []mod.Mod{},
		},
		"rune_paint_area_of_effect_+%_final_per_rune_level": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.Multiplier("RuneLevel"))},
		},
		"rune_paint_area_of_effect_+%_per_rune_level": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0).Tag(mod.Multiplier("RuneLevel"))},
		},
		"rune_paint_damage_+%_final_per_rune_level": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("RuneLevel"))},
		},
		"rune_paint_max_rune_level": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:17567
	m["StormblastMine"] = map[string]*StatMap{
		"lightning_explosion_mine_aura_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeIncrease, 0).Tag(mod.Limit(150), mod.GlobalEffect("AuraDebuff").EffectStackVar("ActiveMineCount"))},
		},
		"quality_display_stormblast_mine_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/other.lua:4086
	m["SummonBeastialRhoa"] = map[string]*StatMap{
		"base_number_of_beast_minions_allowed": {
			Mods: []mod.Mod{mod.NewFloat("ActiveBeastMinionLimit", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/other.lua:4125
	m["SummonBeastialSnake"] = map[string]*StatMap{
		"base_number_of_beast_minions_allowed": {
			Mods: []mod.Mod{mod.NewFloat("ActiveBeastMinionLimit", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/other.lua:4165
	m["SummonBeastialUrsa"] = map[string]*StatMap{
		"base_number_of_beast_minions_allowed": {
			Mods: []mod.Mod{mod.NewFloat("ActiveBeastMinionLimit", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:18513
	m["SummonCarrionGolem"] = map[string]*StatMap{
		"bone_golem_damage_+%_final_per_non_golem_minion_nearby": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0).Tag(multiplierLimitVar("NearbyNonGolemMinion", "MaxNearbyGolemDamage").LimitTotal(true).Actor("parent"))})},
		},
		"bone_golem_damage_per_non_golem_minion_nearby_maximum_%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Multiplier:MaxNearbyGolemDamage", mod.TypeBase, 0)})},
		},
		"bone_golem_grants_minion_maximum_added_physical_damage": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMax", mod.TypeBase, 0)}).Tag(mod.SkillType("Golem").Neg(true), mod.GlobalEffect("Buff"))},
		},
		"bone_golem_grants_minion_minimum_added_physical_damage": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalMin", mod.TypeBase, 0)}).Tag(mod.SkillType("Golem").Neg(true), mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_int.lua:18690
	m["SummonCarrionGolemAltY"] = map[string]*StatMap{
		"bone_golem_damage_+%_final_per_non_golem_minion_nearby": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0).Tag(multiplierLimitVar("NearbyNonGolemMinion", "MaxNearbyGolemDamage").LimitTotal(true).Actor("parent"))})},
		},
		"bone_golem_damage_per_non_golem_minion_nearby_maximum_%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Multiplier:MaxNearbyGolemDamage", mod.TypeBase, 0)})},
		},
	}
	// src/Data/Skills/act_int.lua:18779
	m["SummonChaosGolem"] = map[string]*StatMap{
		"chaos_golem_grants_chaos_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("ChaosResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"chaos_golem_grants_dot_multiplier_+": {
			Mods: []mod.Mod{mod.NewFloat("DotMultiplier", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_str.lua:11332
	m["SummonFlameGolem"] = map[string]*StatMap{
		"fire_golem_grants_area_of_effect_+%": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"fire_golem_grants_damage_+%": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_int.lua:19030
	m["SummonHolyRelic"] = map[string]*StatMap{
		"holy_relic_nova_life_regeneration_rate_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Holy Relic's Boon").EffectCond("HolyRelicBoonActive"))},
			Div:  utils.Ptr(float64(60)),
		},
		"holy_relic_nova_minion_life_regeneration_rate_per_second": {
			Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Holy Relic's Minion Boon").EffectCond("HolyRelicBoonActive").ApplyNotPlayer(true).ApplyMinions(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:15702
	m["SummonIceGolem"] = map[string]*StatMap{
		"ice_golem_grants_accuracy_rating_+": {
			Mods: []mod.Mod{mod.NewFloat("Accuracy", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"ice_golem_grants_critical_strike_chance_+%": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_int.lua:19195
	m["SummonLightningGolem"] = map[string]*StatMap{
		"lightning_golem_grants_attack_and_cast_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"lightning_golem_grants_base_mana_regeneration_rate_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("ManaRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/other.lua:4948
	m["SummonMirageChieftain"] = map[string]*StatMap{
		"skill_used_by_mirage_chieftain_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ChieftainMirageChieftainMoreDamage", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:19512
	m["SummonReaper"] = map[string]*StatMap{
		"active_skill_non_reaper_minion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)}).Tag(mod.SkillName("Summon Reaper").Neg(true).Transfigured(true), mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"active_skill_non_reaper_minion_maximum_life_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Life", mod.TypeMore, 0)}).Tag(mod.SkillName("Summon Reaper").Neg(true).Transfigured(true), mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"bleed_on_hit_with_attacks_%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("BleedChance", mod.TypeBase, 0).Flag(mod.MFlagAttack)})},
		},
	}
	// src/Data/Skills/act_int.lua:19600
	m["SummonReaperAltX"] = map[string]*StatMap{
		"active_skill_non_reaper_minion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)}).Tag(mod.SkillName("Summon Reaper").Neg(true).Transfigured(true), mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"active_skill_non_reaper_minion_maximum_life_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Life", mod.TypeMore, 0)}).Tag(mod.SkillName("Summon Reaper").Neg(true).Transfigured(true), mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_int.lua:19683
	m["SummonReaperAltY"] = map[string]*StatMap{
		"active_skill_non_reaper_minion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)}).Tag(mod.SkillName("Summon Reaper").Neg(true).Transfigured(true), mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"active_skill_non_reaper_minion_maximum_life_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Life", mod.TypeMore, 0)}).Tag(mod.SkillName("Summon Reaper").Neg(true).Transfigured(true), mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"bleed_on_hit_with_attacks_%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("BleedChance", mod.TypeBase, 0).Flag(mod.MFlagAttack)})},
		},
	}
	// src/Data/Skills/sup_int.lua:5821
	m["SummonSacredWisp"] = map[string]*StatMap{
		"base_number_of_sacred_wisps_allowed": {
			Mods: []mod.Mod{mod.NewFloat("SacredWispsMaxCount", mod.TypeBase, 2)},
		},
		"support_sacred_wisps_wisp_%_chance_to_attack": {
			Mods: []mod.Mod{mod.NewFloat("SacredWispsChance", mod.TypeBase, 25)},
		},
		"support_sacred_wisps_wisp_additional_%_chance_to_attack_when_rare_or_unique_enemy_in_presence": {
			Mods: []mod.Mod{mod.NewFloat("SacredWispsChance", mod.TypeBase, 25).Tag(mod.ActorCondition("enemy", "RareOrUnique"))},
		},
	}
	// src/Data/Skills/other.lua:4696
	m["SummonSentinelOfRadiance"] = map[string]*StatMap{
		"radiant_sentinel_minion_burning_effect_radius": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Multiplier:GuardianSentinelFireAuraRadius", mod.TypeBase, 0)})},
		},
		"radiant_sentinel_minion_fire_%_of_life_to_deal_nearby_per_minute": {
			Mods: []mod.Mod{
				mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Multiplier:GuardianSentinelFireAuraBaseDamage", mod.TypeBase, 0)}),
				mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "GuardianSentinelFireAura"}),
			},
			Div: utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/act_int.lua:19774
	m["SummonSkeletons"] = map[string]*StatMap{
		"quality_display_summon_skeleton_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:19853
	m["SummonSkeletonsAltX"] = map[string]*StatMap{
		"quality_display_summon_skeleton_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:19932
	m["SummonSkeletonsAltY"] = map[string]*StatMap{
		"quality_display_summon_skeleton_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:20098
	m["SummonSkitterbots"] = map[string]*StatMap{
		"skitterbots_trap_mine_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagTrap | mod.KeywordFlagMine).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/other.lua:4737
	m["SummonSpectralTiger"] = map[string]*StatMap{
		"modifiers_to_claw_critical_strike_chance_apply_minion_critical_strike_chance": {
			Mods: []mod.Mod{mod.NewFlag("ClawCritChanceAppliesToMinions", true)},
		},
		"modifiers_to_claw_critical_strike_multiplier_apply_minion_critical_strike_multiplier": {
			Mods: []mod.Mod{mod.NewFlag("ClawCritMultiplierAppliesToMinions", true)},
		},
		"spectral_tiger_grants_attack_and_cast_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Tag(mod.Multiplier("SpectralTigerCount"), mod.GlobalEffect("Buff").Name("Spectral Tiger"))},
		},
		"spectral_tiger_grants_critical_strike_chance_+%": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeIncrease, 0).Tag(mod.Multiplier("SpectralTigerCount"), mod.GlobalEffect("Buff").Name("Spectral Tiger"))},
		},
	}
	// src/Data/Skills/other.lua:4787
	m["SummonSpectralWolf"] = map[string]*StatMap{
		"modifiers_to_claw_critical_strike_chance_apply_minion_critical_strike_chance": {
			Mods: []mod.Mod{mod.NewFlag("ClawCritChanceAppliesToMinions", true)},
		},
		"modifiers_to_claw_critical_strike_multiplier_apply_minion_critical_strike_multiplier": {
			Mods: []mod.Mod{mod.NewFlag("ClawCritMultiplierAppliesToMinions", true)},
		},
		"spectral_wolf_grants_attack_maximum_added_physical_damage": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.Multiplier("SpectralWolfCount"), mod.GlobalEffect("Buff").Name("Spectral Wolf"))},
		},
		"spectral_wolf_grants_attack_minimum_added_physical_damage": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.Multiplier("SpectralWolfCount"), mod.GlobalEffect("Buff").Name("Spectral Wolf"))},
		},
	}
	// src/Data/Skills/act_str.lua:11578
	m["SummonStoneGolem"] = map[string]*StatMap{
		"stone_golem_grants_base_life_regeneration_rate_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
			Div:  utils.Ptr(float64(60)),
		},
		"stone_golem_grants_defences_+%": {
			Mods: []mod.Mod{mod.NewFloat("Defences", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/other.lua:4872
	m["SummonVoidSpawn"] = map[string]*StatMap{
		"void_spawn_withered_on_hit_chance_%": {
			Mods: []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
		},
	}
	// src/Data/Skills/act_str.lua:11840
	m["Sunder"] = map[string]*StatMap{
		"active_skill_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0)},
		},
		"shockwave_slam_explosion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(3))},
		},
		"sunder_wave_radius_+_per_step": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(mod.Multiplier("SunderWaveArea"))},
		},
	}
	// src/Data/Skills/act_str.lua:11948
	m["SunderAltX"] = map[string]*StatMap{
		"active_skill_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0)},
		},
		"shockwave_slam_explosion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_str.lua:12053
	m["SunderAltY"] = map[string]*StatMap{
		"shockwave_slam_explosion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(3))},
		},
		"sunder_number_of_fake_chains": {
			Mods: []mod.Mod{},
		},
		"sunder_wave_max_steps": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:SunderWaveArea", mod.TypeBase, 0).Tag(mod.SkillPart(2))},
			Base: utils.Ptr(float64(-1)),
		},
		"sunder_wave_radius_+_per_step": {
			Mods: []mod.Mod{skill("radiusExtra", 0).Tag(mod.Multiplier("SunderWaveArea"))},
		},
	}
	// src/Data/Skills/sup_str.lua:119
	m["SupportAncestralCall"] = map[string]*StatMap{
		"support_spirit_strike_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:228
	m["SupportArcaneSurge"] = map[string]*StatMap{
		"support_arcane_surge_cast_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("ArcaneSurgeCastSpeed", mod.TypeMAX, 0).Tag(mod.GlobalEffect("Buff").Name("Arcane Surge").Unscalable(true))},
		},
		"support_arcane_surge_mana_regeneration_rate_+%": {
			Mods: []mod.Mod{mod.NewFloat("ArcaneSurgeManaRegen", mod.TypeMAX, 0).Tag(mod.GlobalEffect("Buff").Name("Arcane Surge").Unscalable(true))},
		},
		"support_arcane_surge_spell_damage_+%_final_while_you_have_arcane_surge": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell).Tag(mod.Condition("AffectedByArcaneSurge"))},
		},
	}
	// src/Data/Skills/sup_int.lua:304
	m["SupportArchmage"] = map[string]*StatMap{
		"archmage_gain_lightning_damage_%_of_max_unreserved_mana": {
			Mods: []mod.Mod{
				mod.NewFloat("LightningMin", mod.TypeBase, 0).Tag(mod.PercentStat("ManaUnreserved", 1)),
				mod.NewFloat("LightningMax", mod.TypeBase, 0).Tag(mod.PercentStat("ManaUnreserved", 1)),
			},
		},
		"manaweave_cost_equals_%_unreserved_mana": {
			Mods: []mod.Mod{mod.NewFloat("ManaCostBase", mod.TypeBase, 0).Tag(mod.PercentStat("ManaUnreserved", 1))},
		},
	}
	// src/Data/Skills/sup_dex.lua:187
	m["SupportArrowNova"] = map[string]*StatMap{
		"support_rain_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/act_str.lua:712
	m["SupportAutoexertion"] = map[string]*StatMap{
		"support_autoexertion_base_mana_cost_override": {
			Mods: []mod.Mod{mod.NewFloat("BaseManaCostOverride", mod.TypeOverride, 0)},
		},
		"warcries_do_not_apply_buffs_to_self_or_allies": {
			Mods: []mod.Mod{mod.NewFlag("CannotShareWarcryBuffs", true)},
		},
	}
	// src/Data/Skills/sup_str.lua:230
	m["SupportAwakenedAncestralCall"] = map[string]*StatMap{
		"support_spirit_strike_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:264
	m["SupportAwakenedArrowNova"] = map[string]*StatMap{
		"support_rain_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/sup_int.lua:448
	m["SupportAwakenedBlasphemy"] = map[string]*StatMap{
		"curse_apply_as_aura": {
			Mods: []mod.Mod{},
		},
		"support_blasphemy_curse_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("CurseEffect", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:868
	m["SupportAwakenedBrutality"] = map[string]*StatMap{
		"support_brutality_physical_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:979
	m["SupportAwakenedBurningDamage"] = map[string]*StatMap{
		"support_burning_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("FireDamage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagFireDot)},
		},
	}
	// src/Data/Skills/sup_dex.lua:665
	m["SupportAwakenedCastOnCriticalStrike"] = map[string]*StatMap{
		"support_cast_on_crit_spell_damage_+%_final": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_int.lua:827
	m["SupportAwakenedCastWhileChannelling"] = map[string]*StatMap{
		"cast_while_channelling_time_ms": {
			Mods: []mod.Mod{skill("triggerTime", 0).Tag(mod.SkillType("Channel"))},
			Div:  utils.Ptr(float64(1000)),
		},
		"support_cast_while_channelling_triggered_skill_damage_+%_final": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_dex.lua:953
	m["SupportAwakenedChain"] = map[string]*StatMap{
		"support_chain_hit_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit)},
		},
	}
	// src/Data/Skills/sup_int.lua:1240
	m["SupportAwakenedControlledDestruction"] = map[string]*StatMap{
		"support_controlled_destruction_critical_strike_chance_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeMore, 0)},
		},
		"support_controlled_destruction_spell_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell)},
		},
	}
	// src/Data/Skills/sup_dex.lua:1690
	m["SupportAwakenedDeadlyAilments"] = map[string]*StatMap{
		"support_better_ailments_ailment_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagPoison | mod.KeywordFlagBleed | mod.KeywordFlagIgnite)},
		},
		"support_better_ailments_hit_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit)},
		},
	}
	// src/Data/Skills/sup_str.lua:1891
	m["SupportAwakenedElementalDamageWithAttacks"] = map[string]*StatMap{
		"support_attack_skills_elemental_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ElementalDamage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
	}
	// src/Data/Skills/sup_int.lua:2116
	m["SupportAwakenedElementalFocus"] = map[string]*StatMap{
		"support_gem_elemental_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ElementalDamage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:2070
	m["SupportAwakenedFork"] = map[string]*StatMap{
		"support_fork_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/sup_str.lua:2771
	m["SupportAwakenedGenerosity"] = map[string]*StatMap{
		"aura_cannot_affect_self": {
			Mods: []mod.Mod{skill("auraCannotAffectSelf", 1)},
		},
	}
	// src/Data/Skills/sup_dex.lua:2229
	m["SupportAwakenedGreaterMultipleProjectiles"] = map[string]*StatMap{
		"support_multiple_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/sup_int.lua:1573
	m["SupportAwakenedHextouch"] = map[string]*StatMap{
		"apply_linked_curses_on_hit_%": {
			Mods: []mod.Mod{},
		},
		"hextouch_support_curse_duration_+%_final": {
			Mods: []mod.Mod{},
		},
		"support_hextouch_curse_effect_+%_final": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_int.lua:3650
	m["SupportAwakenedIncreasedAreaofEffect"] = map[string]*StatMap{
		"inc_aoe_plus_more_area_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagArea)},
		},
	}
	// src/Data/Skills/sup_str.lua:3731
	m["SupportAwakenedMeleePhysicalDamage"] = map[string]*StatMap{
		"support_melee_physical_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0).Flag(mod.MFlagMelee)},
		},
		"support_melee_physical_damage_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack)},
		},
		"support_melee_physical_damage_poison_and_bleeding_damage_+%_final_from_melee_hits": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagMeleeHit).KeywordFlag(mod.KeywordFlagPoison | mod.KeywordFlagBleed)},
		},
	}
	// src/Data/Skills/sup_int.lua:4880
	m["SupportAwakenedMinionDamage"] = map[string]*StatMap{
		"support_minion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)})},
		},
	}
	// src/Data/Skills/sup_str.lua:4043
	m["SupportAwakenedMultistrike"] = map[string]*StatMap{
		"multistrike_damage_+%_final_on_first_repeat": {
			Mods: []mod.Mod{
				mod.NewFloat("RepeatOneDamage", mod.TypeMore, 0).Tag(mod.ModFlagOr(mod.MFlagUnarmed|mod.MFlagWeaponMelee), mod.Condition("averageRepeat", "alwaysFinalRepeat")),
				mod.NewFloat("RepeatOneDamage", mod.TypeMore, 0).Tag(mod.SkillType("RequiresShield"), mod.Condition("averageRepeat", "alwaysFinalRepeat")),
			},
		},
		"multistrike_damage_+%_final_on_second_repeat": {
			Mods: []mod.Mod{
				mod.NewFloat("RepeatTwoDamage", mod.TypeMore, 0).Tag(mod.ModFlagOr(mod.MFlagUnarmed | mod.MFlagWeaponMelee)),
				mod.NewFloat("RepeatTwoDamage", mod.TypeMore, 0).Tag(mod.SkillType("RequiresShield")),
			},
		},
		"multistrike_damage_+%_final_on_third_repeat": {
			Mods: []mod.Mod{
				mod.NewFloat("RepeatThreeDamage", mod.TypeMore, 0).Tag(mod.ModFlagOr(mod.MFlagUnarmed | mod.MFlagWeaponMelee)),
				mod.NewFloat("RepeatThreeDamage", mod.TypeMore, 0).Tag(mod.SkillType("RequiresShield")),
			},
		},
		"support_multiple_attack_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagAttack)},
		},
		"support_multiple_attacks_melee_attack_speed_+%_final": {
			Mods: []mod.Mod{
				mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.ModFlagOr(mod.MFlagUnarmed | mod.MFlagWeaponMelee)),
				mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.SkillType("RequiresShield")),
			},
		},
	}
	// src/Data/Skills/sup_int.lua:6154
	m["SupportAwakenedSpellCascade"] = map[string]*StatMap{
		"support_spell_cascade_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0)},
		},
		"support_spell_cascade_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:6385
	m["SupportAwakenedSpellEcho"] = map[string]*StatMap{
		"spell_echo_plus_chance_double_damage_%_final": {
			Mods: []mod.Mod{mod.NewFloat("RepeatFinalDoubleDamageChance", mod.TypeBase, 0).Flag(mod.MFlagSpell).Tag(mod.Condition("CastOnFrostbolt").Neg(true), mod.Condition("averageRepeat", "alwaysFinalRepeat"))},
		},
		"support_echo_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
		"support_multicast_cast_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagCast)},
		},
	}
	// src/Data/Skills/sup_dex.lua:3981
	m["SupportAwakenedSwiftAffliction"] = map[string]*StatMap{
		"support_rapid_decay_damage_over_time_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagDot)},
		},
	}
	// src/Data/Skills/sup_int.lua:6731
	m["SupportAwakenedUnboundAilments"] = map[string]*StatMap{
		"support_unbound_ailments_ailment_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagPoison | mod.KeywordFlagBleed | mod.KeywordFlagIgnite)},
		},
	}
	// src/Data/Skills/sup_int.lua:6858
	m["SupportAwakenedUnleash"] = map[string]*StatMap{
		"support_anticipation_charge_gain_interval_ms": {
			Mods: []mod.Mod{mod.NewFloat("SealGainFrequency", mod.TypeBase, 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"support_anticipation_rapid_fire_count": {
			Mods: []mod.Mod{mod.NewFloat("SealCount", mod.TypeBase, 0)},
		},
		"support_spell_rapid_fire_repeat_use_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("SealRepeatPenalty", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:4439
	m["SupportAwakenedViciousProjectiles"] = map[string]*StatMap{
		"support_phys_chaos_projectile_chaos_damage_over_time_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ChaosDamage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagChaosDot)},
		},
		"support_phys_chaos_projectile_physical_damage_over_time_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagPhysicalDot)},
		},
		"support_projectile_attack_physical_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0).Flag(mod.MFlagAttack | mod.MFlagProjectile)},
		},
		"support_projectile_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack | mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/sup_dex.lua:4635
	m["SupportAwakenedVoidManipulation"] = map[string]*StatMap{
		"support_void_manipulation_chaos_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ChaosDamage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:4531
	m["SupportBallistaTotem"] = map[string]*StatMap{
		"support_attack_totem_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack)},
		},
		"support_totem_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:320
	m["SupportBarrage"] = map[string]*StatMap{
		"number_of_additional_projectiles": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileCount", mod.TypeBase, 0).Tag(mod.Condition("UsingBow", "UsingWand"))},
		},
		"projectiles_barrage": {
			Mods: []mod.Mod{mod.NewFlag("SequentialProjectiles", true).Tag(mod.Condition("UsingBow", "UsingWand"))},
		},
		"skill_can_only_use_non_melee_weapons": {
			Mods: []mod.Mod{},
		},
		"support_barrage_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Condition("UsingBow", "UsingWand"))},
		},
	}
	// src/Data/Skills/sup_str.lua:369
	m["SupportBehead"] = map[string]*StatMap{
		"support_executioner_damage_vs_enemies_on_low_life_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "LowLife"))},
		},
	}
	// src/Data/Skills/sup_int.lua:375
	m["SupportBlasphemy"] = map[string]*StatMap{
		"curse_apply_as_aura": {
			Mods: []mod.Mod{},
		},
		"support_blasphemy_curse_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("CurseEffect", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:5672
	m["SupportBlastchainMine"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{skill("showAverage", 1).Tag(mod.SkillType("Mineable"))},
		},
		"support_gem_mine_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:437
	m["SupportBlessedCall"] = map[string]*StatMap{
		"consecrated_cry_warcry_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagWarcry)},
		},
	}
	// src/Data/Skills/sup_dex.lua:459
	m["SupportBlockChanceReduction"] = map[string]*StatMap{
		"global_reduce_enemy_block_%": {
			Mods: []mod.Mod{mod.NewFloat("reduceEnemyBlock", mod.TypeBase, 0)},
		},
		"support_reduce_enemy_block_and_spell_block_%": {
			Mods: []mod.Mod{mod.NewFloat("reduceEnemyBlock", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:504
	m["SupportBloodlust"] = map[string]*StatMap{
		"support_bloodlust_melee_physical_damage_+%_final_vs_bleeding_enemies": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0).Flag(mod.MFlagMelee).Tag(mod.ActorCondition("enemy", "Bleeding"))},
		},
	}
	// src/Data/Skills/sup_str.lua:572
	m["SupportBloodsoakedBanner"] = map[string]*StatMap{
		"support_bloodstained_banner_apply_corrupted_blood_duration_ms": {
			Mods: []mod.Mod{skill("durationSecondary", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"support_bloodstained_banner_corrupted_blood_base_physical_damage_to_deal_per_minute_per_valour": {
			Mods: []mod.Mod{skill("PhysicalDot", 0).Tag(mod.Multiplier("ValourStacks"))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/sup_str.lua:632
	m["SupportBloodthirst"] = map[string]*StatMap{
		"blood_price_gain_%_maximum_life_as_added_physical_damage_with_weapons_while_on_low_life": {
			Mods: []mod.Mod{
				mod.NewFloat("PhysicalMin", mod.TypeBase, 0).Flag(mod.MFlagWeapon).Tag(mod.PercentStat("Life", 1), mod.Condition("LowLife")),
				mod.NewFloat("PhysicalMax", mod.TypeBase, 0).Flag(mod.MFlagWeapon).Tag(mod.PercentStat("Life", 1), mod.Condition("LowLife")),
			},
		},
		"support_blood_thirst_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:551
	m["SupportBonechill"] = map[string]*StatMap{
		"support_bonechill_cold_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ColdDamage", mod.TypeMore, 0)},
		},
		"support_chilling_areas_also_grant_cold_damage_taken_+%_equal_to_slow_amount": {
			Mods: []mod.Mod{},
		},
		"support_chills_also_grant_cold_damage_taken_+%_equal_to_slow_amount": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_str.lua:704
	m["SupportBonespire"] = map[string]*StatMap{
		"support_trigger_seize_the_flesh_on_warcry_chance_%": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:634
	m["SupportBrandSupport"] = map[string]*StatMap{
		"support_brand_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0)},
		},
		"support_brand_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("TriggeredDamage", mod.TypeMore, 0)},
		},
		"trigger_brand_support_hit_damage_+%_final_vs_branded_enemy": {
			Mods: []mod.Mod{mod.NewFloat("TriggeredDamage", mod.TypeMore, 0).Tag(mod.Condition("TargetingBrandedEnemy"))},
		},
	}
	// src/Data/Skills/sup_str.lua:801
	m["SupportBrutality"] = map[string]*StatMap{
		"support_brutality_physical_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:914
	m["SupportBurningDamage"] = map[string]*StatMap{
		"support_burning_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("FireDamage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagFireDot)},
		},
	}
	// src/Data/Skills/sup_dex.lua:596
	m["SupportCastOnCritTriggered"] = map[string]*StatMap{
		"support_cast_on_crit_spell_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell)},
		},
	}
	// src/Data/Skills/sup_dex.lua:712
	m["SupportCastOnCritTriggeredPlus"] = map[string]*StatMap{
		"support_cast_on_crit_spell_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell)},
		},
	}
	// src/Data/Skills/sup_dex.lua:530
	m["SupportCastOnCriticalStrike"] = map[string]*StatMap{
		"support_cast_on_crit_spell_damage_+%_final": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_dex.lua:760
	m["SupportCastOnDeath"] = map[string]*StatMap{
		"area_of_effect_+%_while_dead": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0)},
		},
		"cast_on_death_damage_+%_final_while_dead": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
		"no_cost": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_str.lua:1024
	m["SupportCastOnMeleeKill"] = map[string]*StatMap{
		"support_cast_on_melee_kill_spell_damage_+%_final": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_str.lua:1090
	m["SupportCastOnMeleeKillTriggered"] = map[string]*StatMap{
		"support_cast_on_melee_kill_spell_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell)},
		},
	}
	// src/Data/Skills/sup_dex.lua:840
	m["SupportCastOnWardBreak"] = map[string]*StatMap{
		"cast_on_ward_break_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell)},
		},
	}
	// src/Data/Skills/sup_str.lua:1157
	m["SupportCastWhenDamageTaken"] = map[string]*StatMap{
		"cast_on_damage_taken_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
		"cast_when_damage_taken_trigger_threshold_+%": {
			Mods: []mod.Mod{mod.NewFloat("CWDTThreshold", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:688
	m["SupportCastWhileChannelling"] = map[string]*StatMap{
		"cast_while_channelling_time_ms": {
			Mods: []mod.Mod{skill("triggerTime", 0).Tag(mod.SkillType("Channel"))},
			Div:  utils.Ptr(float64(1000)),
		},
		"support_cast_while_channelling_triggered_skill_damage_+%_final": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_int.lua:758
	m["SupportCastWhileChannellingTriggered"] = map[string]*StatMap{
		"support_cast_while_channelling_triggered_skill_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("TriggeredDamage", mod.TypeMore, 0).Tag(mod.SkillType("Triggered"))},
		},
	}
	// src/Data/Skills/sup_int.lua:879
	m["SupportCastWhileChannellingTriggeredPlus"] = map[string]*StatMap{
		"support_cast_while_channelling_triggered_skill_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("TriggeredDamage", mod.TypeMore, 0).Tag(mod.SkillType("Triggered"))},
		},
	}
	// src/Data/Skills/sup_dex.lua:884
	m["SupportChain"] = map[string]*StatMap{
		"support_chain_hit_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit)},
		},
	}
	// src/Data/Skills/sup_str.lua:1233
	m["SupportChanceToBleed"] = map[string]*StatMap{
		"support_chance_to_bleed_bleeding_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagBleed)},
		},
	}
	// src/Data/Skills/sup_dex.lua:1098
	m["SupportChargedTraps"] = map[string]*StatMap{
		"trap_trigger_radius_": {
			Mods: []mod.Mod{mod.NewFloat("TrapTriggerAreaOfEffect", mod.TypeIncrease, 0).Tag(mod.Multiplier("PowerCharge"))},
		},
	}
}

func skillStatMapsPart4(m map[string]map[string]*StatMap) {
	// src/Data/Skills/sup_dex.lua:1174
	m["SupportCloseCombat"] = map[string]*StatMap{
		"close_combat_damage_to_close_range_+%": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeIncrease, 0).Flag(mod.MFlagAttack | mod.MFlagMelee).Tag(mod.Condition("AtCloseRange"))},
		},
		"support_slashing_buff_attack_speed_+%_final_to_grant": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff").Name("Combat Rush").EffectCond("CombatRushActive"), mod.Condition("SupportedByCloseCombat").Neg(true), mod.SkillType("Travel"))},
		},
		"support_slashing_damage_+%_final_from_distance": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagAttack | mod.MFlagMelee).Tag(mod.MeleeProximity([]int{1, 0}))},
		},
		"supported_skill_can_only_use_axe_and_sword": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_dex.lua:1254
	m["SupportClusterTraps"] = map[string]*StatMap{
		"support_clustertrap_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:927
	m["SupportCombustion"] = map[string]*StatMap{
		"ignites_apply_fire_resistance_+": {
			Mods: []mod.Mod{mod.NewFloat("CombustionFireResist", mod.TypeBase, 0)},
		},
		"support_chance_to_ignite_fire_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("FireDamage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:1423
	m["SupportCompanionship"] = map[string]*StatMap{
		"damage_removed_from_minions_before_life_or_es_%_if_only_one_minion": {
			Mods: []mod.Mod{mod.NewFloat("takenFromMinionBeforeYou", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Condition("OnlyMinion"))},
		},
		"support_companionship_minion_damage_taken_+%_final_if_at_most_one_minion": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeMore, 0)}).Tag(mod.Condition("OnlyMinion"))},
		},
	}
	// src/Data/Skills/sup_int.lua:1100
	m["SupportConcentratedEffect"] = map[string]*StatMap{
		"support_area_concentrate_area_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagArea)},
		},
		"support_concentrated_effect_skill_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:1061
	m["SupportCongregation"] = map[string]*StatMap{
		"support_undead_army_minion_maximum_count_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ActiveMinionLimit", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:1361
	m["SupportControlledBlaze"] = map[string]*StatMap{
		"support_recent_ignites_damage_per_recent_ignite_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(multiplierLimitVar("IgniteAppliedRecently", "ControlledBlazeRecentIgniteLimitMin").LimitNegTotal())},
		},
		"support_recent_ignites_damage_per_recent_ignite_+%_final_minimum": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ControlledBlazeRecentIgniteLimitMin", mod.TypeBase, 0)},
		},
		"support_recent_ignites_ignite_damage_per_recent_ignite_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagIgnite).Tag(multiplierLimitVar("IgniteAppliedRecently", "ControlledBlazeRecentIgniteLimitMax").LimitTotal(true))},
		},
		"support_recent_ignites_ignite_damage_per_recent_ignite_+%_final_maximum": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ControlledBlazeRecentIgniteLimitMax", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:1169
	m["SupportControlledDestruction"] = map[string]*StatMap{
		"support_controlled_destruction_critical_strike_chance_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeMore, 0)},
		},
		"support_controlled_destruction_spell_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell)},
		},
	}
	// src/Data/Skills/sup_str.lua:1440
	m["SupportCorruptingCry"] = map[string]*StatMap{
		"support_corrupting_cry_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0).KeywordFlag(mod.KeywordFlagWarcry)},
		},
		"support_corrupting_cry_corrupted_blood_base_physical_damage_to_deal_per_minute": {
			Mods: []mod.Mod{skill("PhysicalDot", 0)},
			Div:  utils.Ptr(float64(60)),
		},
		"support_corrupting_cry_exerted_attack_applies_X_stacks_of_corrupted_blood_on_first_hit": {
			Mods: []mod.Mod{},
		},
		"support_corrupting_cry_warcry_and_first_exerted_attack_applies_corrupted_blood_for_X_ms": {
			Mods: []mod.Mod{skill("durationSecondary", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"support_corrupting_cry_warcry_applies_X_stacks_of_corrupted_blood": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/other.lua:1010
	m["SupportCrabTotem"] = map[string]*StatMap{
		"support_crab_totem_cast_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagCast)},
		},
		"support_crab_totem_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:1531
	m["SupportCruelty"] = map[string]*StatMap{
		"hits_grant_cruelty": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 1).Flag(mod.MFlagDot).Tag(mod.Multiplier("Cruelty"))},
		},
		"support_cruelty_hit_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit)},
		},
	}
	// src/Data/Skills/sup_str.lua:1622
	m["SupportCrystalfall"] = map[string]*StatMap{
		"support_crystalfall_chance_to_trigger_crystalfall_on_hit_%": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_int.lua:1495
	m["SupportCurseOnHitCurse"] = map[string]*StatMap{
		"curse_triggered_by_hextouch": {
			Mods: []mod.Mod{
				skill("triggeredByCurseOnHit", 1),
				mod.NewFlag("HasNoCost", true),
			},
		},
		"hextouch_support_curse_duration_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Duration", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagCurse)},
		},
		"support_hextouch_curse_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("CurseEffect", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:1622
	m["SupportCurseOnHitCursePlus"] = map[string]*StatMap{
		"curse_triggered_by_hextouch": {
			Mods: []mod.Mod{
				skill("triggeredByCurseOnHit", 1),
				mod.NewFlag("HasNoCost", true),
			},
		},
		"hextouch_support_curse_duration_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Duration", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagCurse)},
		},
		"support_hextouch_curse_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("CurseEffect", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:1715
	m["SupportDamageOnFullLife"] = map[string]*StatMap{
		"support_damage_while_on_full_life_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Condition("FullLife"))},
		},
	}
	// src/Data/Skills/act_int.lua:1575
	m["SupportDarkRitual"] = map[string]*StatMap{
		"apply_linked_curses_with_dark_ritual": {
			Mods: []mod.Mod{mod.NewFlag("Condition:AppliedByBane", true)},
		},
		"support_bane_curse_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("CurseEffect", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:1746
	m["SupportDarkRitualAltX"] = map[string]*StatMap{
		"apply_linked_curses_with_dark_ritual": {
			Mods: []mod.Mod{mod.NewFlag("Condition:AppliedByBane", true)},
		},
		"support_bane_curse_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("CurseEffect", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:1619
	m["SupportDeadlyAilments"] = map[string]*StatMap{
		"support_better_ailments_ailment_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagPoison | mod.KeywordFlagBleed | mod.KeywordFlagIgnite)},
		},
		"support_better_ailments_hit_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit)},
		},
	}
	// src/Data/Skills/sup_int.lua:1866
	m["SupportDevour"] = map[string]*StatMap{
		"killing_blow_consumes_corpse_restore_x_life": {
			Mods: []mod.Mod{mod.NewFloat("LifeOnKill", mod.TypeBase, 0)},
		},
		"killing_blow_consumes_corpse_restore_x_mana": {
			Mods: []mod.Mod{mod.NewFloat("ManaOnKill", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/other.lua:1305
	m["SupportDivineBlessing"] = map[string]*StatMap{
		"support_aura_duration_base_buff_duration": {
			Mods: []mod.Mod{skill("auraDuration", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/sup_str.lua:1779
	m["SupportDivineSentinel"] = map[string]*StatMap{
		"summoned_sentinels_have_random_templar_aura": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/other.lua:1482
	m["SupportEarthbreaker"] = map[string]*StatMap{
		"ancestor_totem_grants_owner_area_of_effect_+%_with_melee_skills": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0).Flag(mod.MFlagMelee)},
		},
		"support_ancestor_slam_totem_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack)},
		},
		"support_ancestor_slam_totem_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:1983
	m["SupportEfficacy"] = map[string]*StatMap{
		"support_efficacy_damage_over_time_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagDot)},
		},
		"support_efficacy_spell_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell)},
		},
	}
	// src/Data/Skills/sup_int.lua:502
	m["SupportEldritchBlasphemy"] = map[string]*StatMap{
		"curse_apply_as_aura": {
			Mods: []mod.Mod{},
		},
		"support_blasphemy_curse_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("CurseEffect", mod.TypeMore, 0)},
		},
		"support_blasphemy_enemies_gain_malignant_madness_if_in_aura_for_x_seconds": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)}).Tag(mod.Condition("enemyMalignantMadness"))},
		},
	}
	// src/Data/Skills/sup_int.lua:5098
	m["SupportElementalArmy"] = map[string]*StatMap{
		"minions_inflict_exposure_on_hit_%_chance": {
			Mods: []mod.Mod{},
		},
		"support_minion_totem_resistance_elemental_damage_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("ElementalDamage", mod.TypeMore, 0)})},
		},
	}
	// src/Data/Skills/sup_str.lua:1826
	m["SupportElementalDamageWithAttacks"] = map[string]*StatMap{
		"support_attack_skills_elemental_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ElementalDamage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAttack)},
		},
	}
	// src/Data/Skills/sup_int.lua:2050
	m["SupportElementalFocus"] = map[string]*StatMap{
		"support_gem_elemental_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ElementalDamage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:2004
	m["SupportEnduranceChargeOnMeleeStun"] = map[string]*StatMap{
		"endurance_charge_on_melee_stun_damage_+%_final_per_endurance_charge": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("EnduranceCharge"))},
		},
	}
	// src/Data/Skills/sup_int.lua:2225
	m["SupportEnergyLeech"] = map[string]*StatMap{
		"support_energy_shield_leech_damage_+%_on_full_energy_shield_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Condition("FullEnergyShield"))},
		},
	}
	// src/Data/Skills/sup_str.lua:2071
	m["SupportEternalBlessing"] = map[string]*StatMap{
		"base_skill_no_reservation": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_str.lua:2143
	m["SupportExcommunicate"] = map[string]*StatMap{
		"support_excommunicate_on_melee_attack_hit_for_X_ms": {
			Mods: []mod.Mod{mod.NewFlag("Condition:CanExcommunicate", true)},
		},
	}
	// src/Data/Skills/sup_int.lua:2357
	m["SupportExemplar"] = map[string]*StatMap{
		"minion_critical_strike_chance_+%_per_attack_crit_recently_up_to_100": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("CritChance", mod.TypeIncrease, 0)}).Tag(mod.Multiplier("CritRecently").Limit(200).LimitTotal(true))},
		},
		"minion_critical_strike_multiplier_+_per_attack_crit_recently_up_to_100": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("CritMultiplier", mod.TypeBase, 0)}).Tag(mod.Multiplier("CritRecently").Limit(100).LimitTotal(true))},
		},
	}
	// src/Data/Skills/sup_str.lua:5494
	m["SupportExpertRetaliation"] = map[string]*StatMap{
		"support_expert_retaliation_cooldown_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("CooldownRecovery", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:2483
	m["SupportFeedingFrenzy"] = map[string]*StatMap{
		"support_minion_offensive_stance_minion_damage_+%_final_while_you_have_puppet_master": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)}).Tag(mod.Condition("FeedingFrenzyActive"))},
		},
	}
	// src/Data/Skills/sup_str.lua:2311
	m["SupportFissure"] = map[string]*StatMap{
		"support_fissure_trigger_fissure_on_slam_skill_impact_chance_%": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_str.lua:2401
	m["SupportFistofWar"] = map[string]*StatMap{
		"ancestral_slam_interval_duration": {
			Mods: []mod.Mod{mod.NewFloat("FistOfWarCooldown", mod.TypeBase, 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"support_ancestral_slam_big_hit_area_+%": {
			Mods: []mod.Mod{mod.NewFloat("FistOfWarIncAoE", mod.TypeBase, 0)},
		},
		"support_ancestral_slam_big_hit_damage_with_hits_and_ailments_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("FistOfWarDamageMultiplier", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:2477
	m["SupportFlamewood"] = map[string]*StatMap{
		"support_flamewood_totems_trigger_infernal_bolt_when_hit": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_dex.lua:1928
	m["SupportFocusedBallista"] = map[string]*StatMap{
		"support_focused_ballista_totem_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack)},
		},
		"support_focused_ballista_totem_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:2549
	m["SupportFocusedChannelling"] = map[string]*StatMap{
		"support_focus_channel_cost_+%_final_per_second_channelling_up_to_100%": {
			Mods: []mod.Mod{mod.NewFloat("Cost", mod.TypeMore, 0).Tag(mod.Multiplier("ChannellingTime").Limit(100).LimitTotal(true))},
		},
		"support_focus_channel_damage_+%_final_per_second_channelling_up_to_60%": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("ChannellingTime").Limit(60).LimitTotal(true))},
		},
	}
	// src/Data/Skills/sup_dex.lua:2001
	m["SupportFork"] = map[string]*StatMap{
		"support_fork_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/sup_str.lua:2630
	m["SupportFortify"] = map[string]*StatMap{
		"fortify_effect_+%": {
			Mods: []mod.Mod{mod.NewFloat("FortifyEffectOnSelf", mod.TypeIncrease, 0)},
		},
		"support_fortify_ailment_damage_+%_final_from_melee_hits": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagAilment | mod.MFlagMeleeHit)},
		},
		"support_fortify_melee_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagMelee)},
		},
	}
	// src/Data/Skills/sup_int.lua:2623
	m["SupportFoulgrasp"] = map[string]*StatMap{
		"base_total_number_of_sigils_allowed": {
			Mods: []mod.Mod{mod.NewFloat("ActiveBrandLimit", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:2715
	m["SupportFreshMeat"] = map[string]*StatMap{
		"support_recent_minions_additional_critical_strike_chance_from_wakened_fury": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("CritChance", mod.TypeBase, 0)}).Tag(mod.Condition("FreshMeatActive"))},
			Div:  utils.Ptr(float64(100)),
		},
		"support_recent_minions_additional_critical_strike_multiplier_from_wakened_fury": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("CritMultiplier", mod.TypeBase, 0)}).Tag(mod.Condition("FreshMeatActive"))},
		},
	}
	// src/Data/Skills/sup_int.lua:2792
	m["SupportFrigidBond"] = map[string]*StatMap{
		"support_damaging_links_base_duration_is_gem": {
			Mods: []mod.Mod{},
		},
		"support_damaging_links_base_duration_ms": {
			Mods: []mod.Mod{mod.NewFloat("SecondaryDuration", mod.TypeBase, 0)},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/sup_int.lua:2867
	m["SupportFrostmage"] = map[string]*StatMap{
		"frostmage_cost_equals_%_reserved_mana": {
			Mods: []mod.Mod{mod.NewFloat("ManaCostBase", mod.TypeBase, 0).Tag(mod.PercentStat("ManaReserved", 1))},
		},
		"frostmage_gain_cold_damage_%_of_max_reserved_mana": {
			Mods: []mod.Mod{
				mod.NewFloat("ColdMin", mod.TypeBase, 0).Tag(mod.PercentStat("ManaReserved", 1)),
				mod.NewFloat("ColdMax", mod.TypeBase, 0).Tag(mod.PercentStat("ManaReserved", 1)),
			},
		},
	}
	// src/Data/Skills/sup_str.lua:2704
	m["SupportGenerosity"] = map[string]*StatMap{
		"aura_cannot_affect_self": {
			Mods: []mod.Mod{skill("auraCannotAffectSelf", 1)},
		},
	}
	// src/Data/Skills/sup_int.lua:2917
	m["SupportGluttony"] = map[string]*StatMap{
		"support_gluttony_fire_x_additional_projectiles_in_nova_while_affected_by_disgorge": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileCount", mod.TypeBase, 0).Tag(mod.Condition("Disgorged"))},
		},
	}
	// src/Data/Skills/sup_int.lua:1935
	m["SupportGreaterDevour"] = map[string]*StatMap{
		"killing_blow_consumes_corpse_chance_to_gain_soul_per_power_permillage": {
			Mods: []mod.Mod{
				mod.NewFlag("Condition:CanHaveSoulEater", true).Tag(mod.GlobalEffect("Buff")),
				mod.NewFlag("Condition:MinionCanHaveSoulEater", true).Tag(mod.GlobalEffect("Buff")),
				mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:CanHaveSoulEater", true)}),
			},
		},
		"killing_blow_consumes_corpse_restore_x_life": {
			Mods: []mod.Mod{mod.NewFloat("LifeOnKill", mod.TypeBase, 0)},
		},
		"killing_blow_consumes_corpse_restore_x_mana": {
			Mods: []mod.Mod{mod.NewFloat("ManaOnKill", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:2160
	m["SupportGreaterMultipleProjectiles"] = map[string]*StatMap{
		"support_multiple_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/sup_str.lua:4112
	m["SupportGreaterMultistrike"] = map[string]*StatMap{
		"multistrike_damage_+%_final_on_first_repeat": {
			Mods: []mod.Mod{
				mod.NewFloat("RepeatOneDamage", mod.TypeMore, 0).Tag(mod.ModFlagOr(mod.MFlagUnarmed|mod.MFlagWeaponMelee), mod.Condition("averageRepeat", "alwaysFinalRepeat")),
				mod.NewFloat("RepeatOneDamage", mod.TypeMore, 0).Tag(mod.SkillType("RequiresShield"), mod.Condition("averageRepeat", "alwaysFinalRepeat")),
			},
		},
		"multistrike_damage_+%_final_on_second_repeat": {
			Mods: []mod.Mod{
				mod.NewFloat("RepeatTwoDamage", mod.TypeMore, 0).Tag(mod.ModFlagOr(mod.MFlagUnarmed | mod.MFlagWeaponMelee)),
				mod.NewFloat("RepeatTwoDamage", mod.TypeMore, 0).Tag(mod.SkillType("RequiresShield")),
			},
		},
		"multistrike_damage_+%_final_on_third_repeat": {
			Mods: []mod.Mod{
				mod.NewFloat("RepeatThreeDamage", mod.TypeMore, 0).Tag(mod.ModFlagOr(mod.MFlagUnarmed | mod.MFlagWeaponMelee)),
				mod.NewFloat("RepeatThreeDamage", mod.TypeMore, 0).Tag(mod.SkillType("RequiresShield")),
			},
		},
		"support_multiple_attack_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagAttack)},
		},
		"support_multiple_attacks_melee_attack_speed_+%_final": {
			Mods: []mod.Mod{
				mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.ModFlagOr(mod.MFlagUnarmed | mod.MFlagWeaponMelee)),
				mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.SkillType("RequiresShield")),
			},
		},
	}
	// src/Data/Skills/sup_int.lua:6212
	m["SupportGreaterSpellCascade"] = map[string]*StatMap{
		"support_spell_cascade_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0)},
		},
		"support_spell_cascade_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:6334
	m["SupportGreaterSpellEcho"] = map[string]*StatMap{
		"support_greater_spell_echo_area_of_effect_+%_per_repeat": {
			Mods: []mod.Mod{mod.NewFloat("RepeatPerRepeatAreaOfEffect", mod.TypeIncrease, 0).Tag(mod.Condition("CastOnFrostbolt").Neg(true))},
		},
		"support_greater_spell_echo_spell_damage_+%_final_per_repeat": {
			Mods: []mod.Mod{mod.NewFloat("RepeatPerRepeatDamage", mod.TypeMore, 0).Tag(mod.Condition("HaveBladeVortex").Neg(true), mod.Condition("CastOnFrostbolt").Neg(true), mod.Condition("averageRepeat", "alwaysFinalRepeat"))},
		},
		"support_multicast_cast_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagCast)},
		},
	}
	// src/Data/Skills/sup_int.lua:6919
	m["SupportGreaterUnleash"] = map[string]*StatMap{
		"support_anticipation_charge_gain_interval_ms": {
			Mods: []mod.Mod{mod.NewFloat("SealGainFrequency", mod.TypeBase, 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"support_anticipation_rapid_fire_count": {
			Mods: []mod.Mod{mod.NewFloat("SealCount", mod.TypeBase, 0)},
		},
		"support_spell_rapid_fire_repeat_use_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("SealRepeatPenalty", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:2277
	m["SupportGreaterVolley"] = map[string]*StatMap{
		"support_greater_volley_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/sup_str.lua:2816
	m["SupportGuardiansBlessing"] = map[string]*StatMap{
		"aura_skill_no_reservation": {
			Mods: []mod.Mod{},
		},
		"support_guardians_blessing_minion_physical_damage_%_of_maximum_life_and_ES_taken_per_minute": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_str.lua:2889
	m["SupportGuardiansBlessingMinion"] = map[string]*StatMap{
		"support_guardians_blessing_minion_physical_damage_%_of_maximum_life_and_ES_taken_per_minute": {
			Mods: []mod.Mod{
				mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDegen", mod.TypeBase, 0).Tag(mod.PercentStat("Life", 1))}),
				mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDegen", mod.TypeBase, 0).Tag(mod.PercentStat("EnergyShield", 1))}),
			},
			Div: utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/sup_str.lua:2953
	m["SupportHallow"] = map[string]*StatMap{
		"hallowing_flame_magnitude_+%": {
			Mods: []mod.Mod{mod.NewFloat("HallowingFlameMagnitude", mod.TypeIncrease, 0)},
		},
		"support_hallow_inflict_hallowing_flame_on_melee_hit": {
			Mods: []mod.Mod{mod.NewFlag("Condition:CanInflictHallowingFlame", true).Tag(mod.GlobalEffect("GlobalDB"))},
		},
	}
	// src/Data/Skills/sup_int.lua:3025
	m["SupportHexpass"] = map[string]*StatMap{
		"chance_to_ignore_hexproof_%": {
			Mods: []mod.Mod{mod.NewFlag("CursesIgnoreHexproof", true)},
		},
		"hexed_enemies_cannot_deal_critical_strikes": {
			Mods: []mod.Mod{
				mod.NewFlag("NeverCrit", true).Tag(mod.GlobalEffect("Curse")),
				mod.NewFlag("Condition:NeverCrit", true).Tag(mod.GlobalEffect("Curse")),
			},
		},
	}
	// src/Data/Skills/sup_int.lua:3074
	m["SupportHextoad"] = map[string]*StatMap{
		"hex_zone_trigger_hextoad_every_x_ms": {
			Mods: []mod.Mod{skill("hextoadTriggerInterval", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/sup_int.lua:1427
	m["SupportHextouch"] = map[string]*StatMap{
		"apply_linked_curses_on_hit_%": {
			Mods: []mod.Mod{},
		},
		"hextouch_support_curse_duration_+%_final": {
			Mods: []mod.Mod{},
		},
		"support_hextouch_curse_effect_+%_final": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_int.lua:3168
	m["SupportHighImpactMine"] = map[string]*StatMap{
		"support_remote_mine_2_chance_to_deal_double_damage_%_against_enemies_near_mines": {
			Mods: []mod.Mod{mod.NewFloat("SelfDoubleDamageChance", mod.TypeBase, 0).Tag(mod.Limit(100), mod.GlobalEffect("AuraDebuff").Name("High-Impact Mine").EffectStackVar("ActiveMineCount"))},
		},
		"support_remote_mine_2_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:2345
	m["SupportHypothermia"] = map[string]*StatMap{
		"support_hypothermia_cold_damage_over_time_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ColdDamage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagColdDot)},
		},
		"support_hypothermia_damage_+%_vs_chilled_enemies_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "Chilled"))},
		},
	}
	// src/Data/Skills/sup_int.lua:3331
	m["SupportIgniteProliferation"] = map[string]*StatMap{
		"support_ignite_prolif_ignite_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagIgnite)},
		},
	}
	// src/Data/Skills/sup_dex.lua:2482
	m["SupportImpale"] = map[string]*StatMap{
		"impale_phys_reduction_%_penalty": {
			Mods: []mod.Mod{mod.NewFloat("EnemyImpalePhysicalDamageReduction", mod.TypeBase, 0)},
			Mult: utils.Ptr(float64(-1)),
		},
		"impale_support_physical_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:3456
	m["SupportImpendingDoom"] = map[string]*StatMap{
		"trigger_vicious_hex_explosion_when_curse_ends": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_int.lua:3817
	m["SupportInfernalLegion"] = map[string]*StatMap{
		"minion_fire_damage_%_of_maximum_life_taken_per_minute": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("FireDegen", mod.TypeBase, 0).Tag(mod.PerStat(1, "Life"), mod.GlobalEffect("Buff"))})},
			Div:  utils.Ptr(float64(6000)),
		},
		"support_minion_instability_minion_base_fire_area_damage_per_minute": {
			Mods: []mod.Mod{
				mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Multiplier:InfernalLegionBaseDamage", mod.TypeBase, 0)}),
				mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "InfernalLegion"}),
			},
			Div: utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/sup_int.lua:3891
	m["SupportInfusedChannelling"] = map[string]*StatMap{
		"support_storm_barrier_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
		"support_storm_barrier_damage_taken_when_hit_+%_final_while_channelling": {
			Mods: []mod.Mod{mod.NewFloat("DamageTakenWhenHit", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff").Name("Infused Channelling").ModCond("InfusionActive"))},
		},
	}
	// src/Data/Skills/sup_int.lua:3965
	m["SupportInnervate"] = map[string]*StatMap{
		"support_innervate_maximum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Innervation").EffectCond("InnervationActive"))},
		},
		"support_innervate_minimum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Innervation").EffectCond("InnervationActive"))},
		},
	}
	// src/Data/Skills/sup_str.lua:4613
	m["SupportInspiration"] = map[string]*StatMap{
		"critical_strike_chance_+%_per_righteous_charge": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeIncrease, 0).Tag(mod.Multiplier("InspirationCharge"))},
		},
		"elemental_damage_+%_final_per_righteous_charge": {
			Mods: []mod.Mod{mod.NewFloat("ElementalDamage", mod.TypeMore, 0).Tag(mod.Multiplier("InspirationCharge"))},
		},
		"support_inspiration_mana_cost_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ManaCost", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:4039
	m["SupportIntensify"] = map[string]*StatMap{
		"support_spell_boost_area_damage_+%_final_per_charge": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagArea).Tag(multiplierLimitVar("Intensity", "IntensityLimit"))},
		},
		"support_spell_boost_area_of_effect_+%_final_per_charge": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(multiplierLimitVar("Intensity", "IntensityLimit"))},
		},
	}
	// src/Data/Skills/act_dex.lua:18310
	m["SupportIntuitiveLink"] = map[string]*StatMap{
		"support_trigger_link_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
		"trigger_on_trigger_link_target_hit": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_dex.lua:2553
	m["SupportInvention"] = map[string]*StatMap{
		"support_invention_trap_and_mine_damage_+%_final_per_second_placed_up_to_100": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagTrap | mod.KeywordFlagMine).Tag(mod.Multiplier("PlacedDuration").Limit(100).LimitTotal(true))},
		},
		"trap_and_mine_area_of_effect_+%_per_second_placed_up_to_100": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0).KeywordFlag(mod.KeywordFlagTrap | mod.KeywordFlagMine).Tag(mod.Multiplier("PlacedDuration").Limit(100).LimitTotal(true))},
		},
	}
	// src/Data/Skills/sup_str.lua:3327
	m["SupportLessDuration"] = map[string]*StatMap{
		"support_reduced_duration_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
		"support_reduced_duration_skill_effect_duration_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Duration", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:3205
	m["SupportLesserMultipleProjectiles"] = map[string]*StatMap{
		"support_lesser_multiple_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/sup_dex.lua:2597
	m["SupportLethalDose"] = map[string]*StatMap{
		"additional_poisons_+_to_apply_vs_non_poisoned_enemies": {
			Mods: []mod.Mod{mod.NewFloat("AdditionalPoisonStacks", mod.TypeBase, 0).Tag(mod.Condition("NonPoisonedOnly"))},
		},
		"support_lethal_dose_poison_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagPoison)},
		},
	}
	// src/Data/Skills/sup_str.lua:3512
	m["SupportLifetap"] = map[string]*StatMap{
		"support_lifetap_damage_+%_final_while_buffed": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Condition("Lifetap"))},
		},
	}
	// src/Data/Skills/sup_dex.lua:2645
	m["SupportLocusMine"] = map[string]*StatMap{
		"number_of_projectiles_+%_final_from_locus_mine_support": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileCount", mod.TypeMore, 0)},
		},
		"support_locus_mine_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:2729
	m["SupportMachinations"] = map[string]*StatMap{
		"physical_damage_%_to_add_as_chaos_per_keystone": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsChaos", mod.TypeBase, 0).Tag(mod.Multiplier("AllocatedKeystone"))},
		},
		"skill_cost_+%_per_keystone": {
			Mods: []mod.Mod{mod.NewFloat("Cost", mod.TypeIncrease, 0).Tag(mod.Multiplier("AllocatedKeystone"))},
		},
	}
	// src/Data/Skills/sup_str.lua:3585
	m["SupportMaim"] = map[string]*StatMap{
		"support_maim_chance_physical_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0)},
		},
		"support_maimed_enemies_physical_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Debuff").Name("Maim"), mod.Condition("Maimed"))},
		},
	}
	// src/Data/Skills/sup_dex.lua:2904
	m["SupportManaforgedArrows"] = map[string]*StatMap{
		"support_manaforged_arrows_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
		"support_manaforged_arrows_damage_+%_final_per_mana_spent": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.PerStat(1, "ManaCost"))},
		},
		"support_manaforged_arrows_mana_cost_%_threshold": {
			Mods: []mod.Mod{
				skill("triggeredByManaforged", 1).Tag(mod.SkillType("Triggerable")),
				skill("ManaForgedArrowsPercentThreshold", 0),
			},
			Div: utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/sup_int.lua:4668
	m["SupportMeatShield"] = map[string]*StatMap{
		"minion_block_%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("BlockChance", mod.TypeBase, 0)})},
		},
		"support_minion_defensive_stance_minion_damage_+%_final_against_enemies_near_you": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)}).Tag(mod.Condition("MeatShieldEnemyNearYou"))},
		},
		"support_minion_defensive_stance_minion_damage_taken_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeMore, 0)})},
		},
	}
	// src/Data/Skills/sup_str.lua:3656
	m["SupportMeleePhysicalDamage"] = map[string]*StatMap{
		"support_melee_physical_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0).Flag(mod.MFlagMelee)},
		},
		"support_melee_physical_damage_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack)},
		},
		"support_melee_physical_damage_poison_and_bleeding_damage_+%_final_from_melee_hits": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagMeleeHit).KeywordFlag(mod.KeywordFlagPoison | mod.KeywordFlagBleed)},
		},
	}
	// src/Data/Skills/sup_int.lua:4743
	m["SupportMinefield"] = map[string]*StatMap{
		"support_minefield_mine_throwing_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("MineLayingSpeed", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:4812
	m["SupportMinionDamage"] = map[string]*StatMap{
		"support_minion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)})},
		},
	}
	// src/Data/Skills/sup_int.lua:4928
	m["SupportMinionLife"] = map[string]*StatMap{
		"support_minion_maximum_life_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Life", mod.TypeMore, 0)})},
		},
	}
	// src/Data/Skills/sup_int.lua:4995
	m["SupportMinionPact"] = map[string]*StatMap{
		"spell_maximum_added_physical_damage_per_active_permanent_minion": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMax", mod.TypeBase, 0).Tag(mod.Multiplier("PermanentMinion"))},
		},
		"spell_minimum_added_physical_damage_per_active_permanent_minion": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMin", mod.TypeBase, 0).Tag(mod.Multiplier("PermanentMinion"))},
		},
	}
	// src/Data/Skills/sup_dex.lua:3041
	m["SupportMirageArcher"] = map[string]*StatMap{
		"summon_mirage_archer_on_hit": {
			Mods: []mod.Mod{mod.NewFloat("MirageArcherMaxCount", mod.TypeBase, 1)},
		},
		"support_mirage_archer_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("MirageArcherLessAttackSpeed", mod.TypeBase, 0)},
		},
		"support_mirage_archer_base_duration": {
			Mods: []mod.Mod{mod.NewFloat("MirageArcherDuration", mod.TypeBase, 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"support_mirage_archer_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("MirageArcherLessDamage", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:3122
	m["SupportMomentum"] = map[string]*StatMap{
		"count_as_momentum_+_if_not_channelled": {
			Mods: []mod.Mod{mod.NewFloat("MomentumStacksExtra", mod.TypeBase, 0)},
		},
		"support_momentum_attack_speed_+%_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.Multiplier("MomentumStacks"))},
		},
		"support_momentum_max_stacks": {
			Mods: []mod.Mod{mod.NewFloat("MomentumStacksMax", mod.TypeBase, 0)},
		},
		"support_momentum_movement_speed_+%_per_stack_removed": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.Multiplier("MomentumStacksRemoved"), mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/sup_str.lua:3024
	m["SupportMoreDuration"] = map[string]*StatMap{
		"support_more_duration_skill_effect_duration_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Duration", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:3892
	m["SupportMultipleTotems"] = map[string]*StatMap{
		"support_additional_totem_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_dex.lua:3272
	m["SupportMultipleTraps"] = map[string]*StatMap{
		"support_multithrow_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:3959
	m["SupportMultistrike"] = map[string]*StatMap{
		"multistrike_damage_+%_final_on_first_repeat": {
			Mods: []mod.Mod{
				mod.NewFloat("RepeatOneDamage", mod.TypeMore, 0).Tag(mod.ModFlagOr(mod.MFlagUnarmed|mod.MFlagWeaponMelee), mod.Condition("averageRepeat", "alwaysFinalRepeat")),
				mod.NewFloat("RepeatOneDamage", mod.TypeMore, 0).Tag(mod.SkillType("RequiresShield"), mod.Condition("averageRepeat", "alwaysFinalRepeat")),
			},
		},
		"multistrike_damage_+%_final_on_second_repeat": {
			Mods: []mod.Mod{
				mod.NewFloat("RepeatTwoDamage", mod.TypeMore, 0).Tag(mod.ModFlagOr(mod.MFlagUnarmed | mod.MFlagWeaponMelee)),
				mod.NewFloat("RepeatTwoDamage", mod.TypeMore, 0).Tag(mod.SkillType("RequiresShield")),
			},
		},
		"support_multiple_attack_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagAttack)},
		},
		"support_multiple_attacks_melee_attack_speed_+%_final": {
			Mods: []mod.Mod{
				mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.ModFlagOr(mod.MFlagUnarmed | mod.MFlagWeaponMelee)),
				mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.SkillType("RequiresShield")),
			},
		},
	}
	// src/Data/Skills/sup_dex.lua:3345
	m["SupportNightblade"] = map[string]*StatMap{
		"supported_skill_can_only_use_dagger_and_claw": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_int.lua:5168
	m["SupportOvercharge"] = map[string]*StatMap{
		"support_pure_shock_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
		"support_pure_shock_shock_as_though_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ShockAsThoughDealing", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:4175
	m["SupportOverexertion"] = map[string]*StatMap{
		"empowered_attack_damage_+%": {
			Mods: []mod.Mod{mod.NewFloat("ExertIncrease", mod.TypeIncrease, 0)},
		},
		"support_overexertion_damage_+%_final_if_exerted": {
			Mods: []mod.Mod{mod.NewFloat("ExertIncrease", mod.TypeMore, 0)},
		},
		"support_overexertion_damage_+%_final_per_warcry_exerting_action": {
			Mods: []mod.Mod{mod.NewFloat("OverexertionExertAverageIncrease", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:4247
	m["SupportOverheat"] = map[string]*StatMap{
		"support_overheat_ailment_damage_+%_final_per_3_fortification_consumed": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagAilment).Tag(mod.PerStat(3, "RemovableFortification"))},
		},
	}
	// src/Data/Skills/sup_int.lua:5240
	m["SupportOverloadedIntensity"] = map[string]*StatMap{
		"additional_critical_strike_chance_per_overloaded_intensity": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeBase, 0).Tag(mod.Multiplier("OverloadedIntensity"))},
			Div:  utils.Ptr(float64(100)),
		},
		"critical_strike_multiplier_+_per_overloaded_intensity": {
			Mods: []mod.Mod{mod.NewFloat("CritMultiplier", mod.TypeBase, 0).Tag(mod.Multiplier("OverloadedIntensity"))},
		},
	}
	// src/Data/Skills/sup_dex.lua:3454
	m["SupportPierce"] = map[string]*StatMap{
		"support_pierce_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/sup_int.lua:5520
	m["SupportPinpoint"] = map[string]*StatMap{
		"additional_projectiles_per_intensity": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileCount", mod.TypeBase, 0).Tag(multiplierLimitVar("Intensity", "IntensityLimit"))},
		},
		"greater_projectile_intensity_projectile_damage_+%_final_per_intensity": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile).Tag(multiplierLimitVar("Intensity", "IntensityLimit"))},
		},
		"minimum_number_of_projectiles_to_fire_is_1": {
			Mods:  []mod.Mod{mod.NewFloat("ProjectileCountMinimum", mod.TypeBase, 0)},
			Value: utils.Ptr(float64(1)),
		},
		"support_greater_projectile_intensity_projectile_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/sup_dex.lua:3518
	m["SupportPointBlank"] = map[string]*StatMap{
		"keystone_point_blank": {
			Mods: []mod.Mod{mod.NewFlag("PointBlank", true)},
		},
		"knockback_chance_%_at_close_range": {
			Mods: []mod.Mod{mod.NewFloat("EnemyKnockbackChance", mod.TypeBase, 0).Flag(mod.MFlagHit)},
		},
	}
	// src/Data/Skills/sup_int.lua:5603
	m["SupportPowerChargeOnCritical"] = map[string]*StatMap{
		"support_power_charge_on_crit_damage_+%_final_per_power_charge": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("PowerCharge"))},
		},
	}
	// src/Data/Skills/sup_int.lua:1679
	m["SupportPredator"] = map[string]*StatMap{
		"deathmark_minion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)})},
		},
		"support_minion_focus_fire_damage_+%_final_vs_focussed_target": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, 0)}).Tag(mod.Condition("EnemyHasDeathmark"))},
		},
	}
	// src/Data/Skills/sup_int.lua:5352
	m["SupportPrismaticBurst"] = map[string]*StatMap{
		"trigger_prismatic_burst_on_hit_%_chance": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_str.lua:4288
	m["SupportPulverise"] = map[string]*StatMap{
		"support_pulverise_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack)},
		},
		"support_pulverise_melee_area_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagMelee | mod.MFlagArea)},
		},
	}
	// src/Data/Skills/sup_str.lua:4360
	m["SupportPyre"] = map[string]*StatMap{
		"support_call_the_pyre_chance_to_trigger_call_the_pyre_on_melee_hit_vs_ignited_enemies_%": {
			Mods: []mod.Mod{skill("chanceToTriggerCallThePyreOnHit", 0).Tag(mod.SkillType("Melee"), mod.ActorCondition("enemy", "Ignited"))},
		},
	}
	// src/Data/Skills/sup_str.lua:4456
	m["SupportRage"] = map[string]*StatMap{
		"attack_speed_+%_with_at_least_10_rage": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.MultiplierThreshold("Rage").Threshold(10))},
		},
	}
	// src/Data/Skills/sup_dex.lua:3585
	m["SupportRupture"] = map[string]*StatMap{
		"critical_strikes_that_inflict_bleeding_also_rupture": {
			Mods: []mod.Mod{mod.NewFlag("Condition:CanInflictRupture", true).Tag(mod.GlobalEffect("Buff"))},
		},
		"support_rupture_bleeding_damage_taken_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagBleed).Tag(mod.GlobalEffect("Debuff").Name("Rupture"), mod.Multiplier("RuptureStack").Limit(4))},
		},
		"support_rupture_bleeding_time_passed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("BleedExpireRate", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagBleed).Tag(mod.GlobalEffect("Debuff").Name("Rupture"), mod.Multiplier("RuptureStack").Limit(4))},
		},
	}
	// src/Data/Skills/sup_str.lua:4694
	m["SupportRuthless"] = map[string]*StatMap{
		"support_ruthless_big_hit_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("RuthlessBlowHitMultiplier", mod.TypeBase, 0).Flag(mod.MFlagMelee)},
		},
		"support_ruthless_big_hit_max_count": {
			Mods: []mod.Mod{mod.NewFloat("RuthlessBlowMaxCount", mod.TypeBase, 0)},
		},
		"support_ruthless_big_hit_stun_base_duration_override_ms": {
			Mods: []mod.Mod{skill("baseStunDuration", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"support_ruthless_blow_ailment_damage_from_melee_hits_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("RuthlessBlowAilmentMultiplier", mod.TypeBase, 0).Flag(mod.MFlagMelee)},
		},
	}
	// src/Data/Skills/sup_int.lua:5754
	m["SupportSacredWisps"] = map[string]*StatMap{
		"skill_used_by_sacred_wisp_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("SacredWispsLessDamage", mod.TypeBase, 0)},
		},
		"supported_skill_can_only_use_wand": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_int.lua:5895
	m["SupportSacrifice"] = map[string]*StatMap{
		"support_sacrifice_gain_%_of_sacrificed_life_as_added_chaos_damage": {
			Mods: []mod.Mod{
				mod.NewFloat("ChaosMin", mod.TypeBase, 0).Tag(mod.Multiplier("SacrificePercent")),
				mod.NewFloat("ChaosMax", mod.TypeBase, 0).Tag(mod.Multiplier("SacrificePercent")),
			},
			Div: utils.Ptr(float64(100)),
		},
		"support_sacrifice_sacrifice_%_of_current_life": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:SacrificePercent", mod.TypeBase, 0).Tag(mod.PercentStat("LifeUnreserved", 1).Actor("parent"))},
		},
	}
	// src/Data/Skills/sup_dex.lua:3658
	m["SupportSadism"] = map[string]*StatMap{
		"support_faster_ailments_ailment_duration_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("EnemyAilmentDuration", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:5969
	m["SupportScornfulHerald"] = map[string]*StatMap{
		"herald_no_buff_effect": {
			Mods:  []mod.Mod{mod.NewFloat("BuffEffect", mod.TypeMore, 0).Tag(mod.SkillType("Herald"))},
			Value: utils.Ptr(float64(-100)),
		},
	}
	// src/Data/Skills/sup_str.lua:4779
	m["SupportShockwave"] = map[string]*StatMap{
		"support_blunt_chance_to_trigger_shockwave_on_hit_%": {
			Mods: []mod.Mod{},
		},
		"supported_skill_can_only_use_mace_and_staff": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_dex.lua:3786
	m["SupportSlowerProjectiles"] = map[string]*StatMap{
		"support_slower_projectiles_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile)},
		},
		"support_slower_projectiles_projectile_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileSpeed", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:6077
	m["SupportSpellCascade"] = map[string]*StatMap{
		"support_spell_cascade_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0)},
		},
		"support_spell_cascade_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:6262
	m["SupportSpellEcho"] = map[string]*StatMap{
		"support_echo_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
		"support_multicast_cast_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagCast)},
		},
	}
	// src/Data/Skills/sup_str.lua:5047
	m["SupportSpellTotem"] = map[string]*StatMap{
		"support_spell_totem_cast_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagCast)},
		},
		"support_totem_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:6010
	m["SupportSpellblade"] = map[string]*StatMap{
		"support_spell_weapon_damage_gain_%_of_one_hand_melee_weapon_damage_as_added_spell_damage": {
			Mods: []mod.Mod{mod.NewFloat("OneHandWeaponDamageAppliesToSpells", mod.TypeMAX, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:17126
	m["SupportSpellslinger"] = map[string]*StatMap{
		"gain_%_of_base_wand_damage_as_added_spell_damage": {
			Mods: []mod.Mod{skill("gainPercentBaseWandDamageToSpells", 0)},
		},
		"quality_display_wand_damage_as_added_spell_damage_is_gem": {
			Mods: []mod.Mod{},
		},
		"spellslinger_trigger_on_wand_attack_%": {
			Mods: []mod.Mod{skill("triggeredBySpellSlinger", 0).Tag(mod.SkillType("Triggerable"), mod.SkillType("Spell"))},
		},
		"support_spellslinger_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment)},
		},
	}
	// src/Data/Skills/sup_int.lua:6442
	m["SupportSummonPhantasm"] = map[string]*StatMap{
		"phantasm_maximum_added_physical_damage_to_grant": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMax", mod.TypeBase, 0).Flag(mod.MFlagSpell).Tag(mod.PerStat(1, "ActivePhantasmLimit"), mod.GlobalEffect("Buff").Name("Phantasmal Might").EffectCond("PhantasmalMight").AllowTotemBuff(true))},
		},
		"phantasm_minimum_added_physical_damage_to_grant": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMin", mod.TypeBase, 0).Flag(mod.MFlagSpell).Tag(mod.PerStat(1, "ActivePhantasmLimit"), mod.GlobalEffect("Buff").Name("Phantasmal Might").EffectCond("PhantasmalMight").AllowTotemBuff(true))},
		},
	}
}

func skillStatMapsPart5(m map[string]map[string]*StatMap) {
	// src/Data/Skills/sup_dex.lua:3853
	m["SupportSwiftAffliction"] = map[string]*StatMap{
		"support_rapid_decay_damage_over_time_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagDot)},
		},
	}
	// src/Data/Skills/sup_int.lua:6516
	m["SupportSwiftbrand"] = map[string]*StatMap{
		"support_rapid_activation_brand_activation_rate_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("BrandActivationFrequency", mod.TypeMore, 0)},
		},
		"support_rapid_activation_brand_skill_only_primary_duration_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PrimaryDuration", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagBrand)},
		},
		"support_rapid_activation_brand_skill_only_secondary_duration_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("SecondaryDuration", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagBrand)},
		},
	}
	// src/Data/Skills/sup_str.lua:5185
	m["SupportTransfusion"] = map[string]*StatMap{
		"random_nearby_minion_gain_a_soul_eater_soul_on_X_life_spent": {
			Mods: []mod.Mod{
				mod.NewFlag("Condition:MinionCanHaveSoulEater", true).Tag(mod.GlobalEffect("Buff").Name("Transfusion")),
				mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFlag("Condition:CanHaveSoulEater", true)}).Tag(mod.GlobalEffect("Buff").Name("Transfusion")),
			},
		},
	}
	// src/Data/Skills/sup_dex.lua:4096
	m["SupportTrap"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{skill("showAverage", 1).Tag(mod.SkillType("Trappable"))},
		},
	}
	// src/Data/Skills/sup_dex.lua:4231
	m["SupportTrapAndMineDamage"] = map[string]*StatMap{
		"support_trap_and_mine_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagTrap | mod.KeywordFlagMine)},
		},
		"support_trap_and_mine_damage_mine_throwing_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("MineLayingSpeed", mod.TypeMore, 0)},
		},
		"support_trap_and_mine_damage_trap_throwing_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("TrapThrowingSpeed", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:5239
	m["SupportTrauma"] = map[string]*StatMap{
		"attack_maximum_added_physical_damage_with_weapons_per_trauma": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMax", mod.TypeBase, 0).Flag(mod.MFlagAttack | mod.MFlagWeapon).Tag(mod.Multiplier("TraumaStacks"))},
		},
		"attack_minimum_added_physical_damage_with_weapons_per_trauma": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMin", mod.TypeBase, 0).Flag(mod.MFlagAttack | mod.MFlagWeapon).Tag(mod.Multiplier("TraumaStacks"))},
		},
		"support_trauma_base_duration_ms": {
			Mods: []mod.Mod{mod.NewFloat("TraumaDuration", mod.TypeBase, 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"support_trauma_stun_duration_+%_per_trauma": {
			Mods: []mod.Mod{mod.NewFloat("EnemyStunDuration", mod.TypeIncrease, 0).Tag(mod.Multiplier("TraumaStacks"))},
		},
		"supported_skill_can_only_use_axe_mace_and_staff": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/other.lua:6047
	m["SupportTriggerFireSpellOnHit"] = map[string]*StatMap{
		"spell_is_triggered_on_hit": {
			Mods: []mod.Mod{skill("triggeredBySettlersEnchantTrigger", 1)},
		},
	}
	// src/Data/Skills/other.lua:3491
	m["SupportTriggerSpellOnUnarmedMeleeCriticalHit"] = map[string]*StatMap{
		"spell_is_triggered_on_hit": {
			Mods: []mod.Mod{
				skill("triggeredBySevenTeachings", 1).Tag(mod.SkillType("Triggerable"), mod.SkillType("Spell")),
				skill("triggerOnCrit", 1).Tag(mod.SkillType("Triggerable"), mod.SkillType("Spell")),
			},
		},
	}
	// src/Data/Skills/sup_int.lua:6591
	m["SupportTrinity"] = map[string]*StatMap{
		"damage_penetrates_%_elemental_resistances_while_all_resonance_is_25": {
			Mods: []mod.Mod{mod.NewFloat("ElementalPenetration", mod.TypeBase, 0).Tag(mod.MultiplierThreshold("ResonanceCount").Threshold(25))},
		},
		"elemental_damage_+%_final_per_5_lowest_resonance": {
			Mods: []mod.Mod{mod.NewFloat("ElementalDamage", mod.TypeMore, 0).Tag(mod.Multiplier("ResonanceCount").Div(5))},
		},
	}
	// src/Data/Skills/sup_int.lua:6664
	m["SupportUnboundAilments"] = map[string]*StatMap{
		"support_unbound_ailments_ailment_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagPoison | mod.KeywordFlagBleed | mod.KeywordFlagIgnite)},
		},
	}
	// src/Data/Skills/sup_dex.lua:4307
	m["SupportUnholyTrinity"] = map[string]*StatMap{
		"damage_+%_final_per_10_lowest_unholy_resonance": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("UnholyResonanceCount").Div(10))},
		},
		"hits_ignore_enemy_monster_lightning_and_chaos_resistance_%_chance_while_all_unholy_resonance_is_25": {
			Mods: []mod.Mod{
				mod.NewFloat("ChanceToIgnoreChaosResistance", mod.TypeBase, 0).Tag(mod.MultiplierThreshold("UnholyResonanceCount").Threshold(25)),
				mod.NewFloat("ChanceToIgnoreLightningResistance", mod.TypeBase, 0).Tag(mod.MultiplierThreshold("UnholyResonanceCount").Threshold(25)),
			},
		},
		"hits_ignore_enemy_monster_physical_damage_reduction_%_chance_while_all_unholy_resonance_is_25": {
			Mods: []mod.Mod{mod.NewFloat("ChanceToIgnoreEnemyPhysicalDamageReduction", mod.TypeBase, 0).Tag(mod.MultiplierThreshold("UnholyResonanceCount").Threshold(25))},
		},
	}
	// src/Data/Skills/sup_int.lua:6776
	m["SupportUnleash"] = map[string]*StatMap{
		"support_anticipation_charge_gain_interval_ms": {
			Mods: []mod.Mod{mod.NewFloat("SealGainFrequency", mod.TypeBase, 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"support_anticipation_rapid_fire_count": {
			Mods: []mod.Mod{mod.NewFloat("SealCount", mod.TypeBase, 0)},
		},
		"support_spell_rapid_fire_repeat_use_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("SealRepeatPenalty", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_int.lua:6975
	m["SupportVaalSacrifice"] = map[string]*StatMap{
		"support_multicast_cast_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagCast)},
		},
	}
	// src/Data/Skills/sup_str.lua:5388
	m["SupportVaalTemptation"] = map[string]*StatMap{
		"support_vaal_temptation_physical_damage_taken_per_minute_instead_of_soul_gain_prevention": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/sup_dex.lua:4360
	m["SupportViciousProjectiles"] = map[string]*StatMap{
		"support_phys_chaos_projectile_chaos_damage_over_time_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ChaosDamage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagChaosDot)},
		},
		"support_phys_chaos_projectile_physical_damage_over_time_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagPhysicalDot)},
		},
		"support_projectile_attack_physical_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0).Flag(mod.MFlagAttack | mod.MFlagProjectile)},
		},
		"support_projectile_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack | mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/sup_dex.lua:4496
	m["SupportVileToxins"] = map[string]*StatMap{
		"support_debilitate_hit_damage_+%_final_per_poison_stack": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(multiplierLimitVar("PoisonStack", "VileToxinsPoisonLimit").Actor("enemy"))},
		},
		"support_debilitate_hit_damage_max_poison_stacks": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:VileToxinsPoisonLimit", mod.TypeBase, 0)},
		},
		"support_debilitate_poison_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagPoison)},
		},
	}
	// src/Data/Skills/sup_dex.lua:4569
	m["SupportVoidManipulation"] = map[string]*StatMap{
		"support_void_manipulation_chaos_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("ChaosDamage", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/sup_str.lua:4931
	m["SupportVoidShockwave"] = map[string]*StatMap{
		"support_blunt_chance_to_trigger_shockwave_on_hit_%": {
			Mods: []mod.Mod{},
		},
		"supported_skill_can_only_use_axe_mace_and_staff": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_dex.lua:4686
	m["SupportVoidstorm"] = map[string]*StatMap{
		"support_voidstorm_trigger_voidstorm_on_rain_skill_impact": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/sup_str.lua:5427
	m["SupportVolatility"] = map[string]*StatMap{
		"maximum_attack_damage_+%_final_from_volatility_support": {
			Mods: []mod.Mod{mod.NewFloat("MaxDamage", mod.TypeMore, 0).Tag(mod.SkillType("Attack"))},
		},
		"minimum_attack_damage_+%_final_from_volatility_support": {
			Mods: []mod.Mod{mod.NewFloat("MinDamage", mod.TypeMore, 0).Tag(mod.SkillType("Attack"))},
		},
	}
	// src/Data/Skills/sup_dex.lua:4790
	m["SupportVolley"] = map[string]*StatMap{
		"support_parallel_projectiles_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagProjectile)},
		},
	}
	// src/Data/Skills/act_str.lua:6322
	m["Sweep"] = map[string]*StatMap{
		"holy_sweep_hammerfall_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_str.lua:6416
	m["SweepAltX"] = map[string]*StatMap{
		"holy_sweep_hammerfall_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_str.lua:10869
	m["Swordstorm"] = map[string]*StatMap{
		"swordstorm_num_hits": {
			Mods: []mod.Mod{skill("dpsMultiplier", 0)},
		},
	}
	// src/Data/Skills/spectre.lua:8941
	m["TalismanT1TemporalChains"] = map[string]*StatMap{
		"buff_time_passed_+%_other_than_temporal_chains": {
			Mods: []mod.Mod{mod.NewFloat("BuffExpireFaster", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"temporal_chains_action_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("TemporalChainsActionSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique").Neg(true))},
		},
		"temporal_chains_action_speed_+%_vs_rare_or_unique_final": {
			Mods: []mod.Mod{mod.NewFloat("TemporalChainsActionSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique"))},
		},
	}
	// src/Data/Skills/spectre.lua:8901
	m["TalismanT1Vulnerability"] = map[string]*StatMap{
		"physical_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"receive_bleeding_chance_%_when_hit_by_attack": {
			Mods: []mod.Mod{mod.NewFloat("SelfBleedChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/spectre.lua:8855
	m["TalismanT2EnfeebleAura"] = map[string]*StatMap{
		"accuracy_rating_+%": {
			Mods: []mod.Mod{mod.NewFloat("Accuracy", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"enfeeble_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique").Neg(true))},
		},
		"enfeeble_damage_+%_vs_rare_or_unique_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique"))},
		},
	}
	// src/Data/Skills/act_str.lua:12248
	m["TectonicSlamAltX"] = map[string]*StatMap{
		"endurance_charge_slam_damage_+%_final_per_endurance_charge_consumed": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("RemovableEnduranceCharge"))},
		},
		"quality_display_alternate_tectonic_slam_is_gem": {
			Mods: []mod.Mod{},
		},
		"tectonic_slam_area_of_effect_+%_final_per_endurance_charge_consumed": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.Multiplier("RemovableEnduranceCharge"))},
		},
	}
	// src/Data/Skills/act_int.lua:20179
	m["TempestShield"] = map[string]*StatMap{
		"shield_spell_block_%": {
			Mods: []mod.Mod{mod.NewFloat("SpellBlockChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
		},
		"skill_display_buff_grants_shock_immunity": {
			Mods: []mod.Mod{mod.NewFlag("ShockImmune", true).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_dex.lua:15943
	m["TemporalChains"] = map[string]*StatMap{
		"buff_time_passed_+%_other_than_temporal_chains": {
			Mods: []mod.Mod{mod.NewFloat("BuffExpireFaster", mod.TypeMore, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"curse_effect_+%_final_vs_players": {
			Mods: []mod.Mod{mod.NewFloat("CurseEffectAgainstPlayer", mod.TypeMore, 0)},
		},
		"temporal_chains_action_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("TemporalChainsActionSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique").Neg(true))},
		},
		"temporal_chains_action_speed_+%_vs_rare_or_unique_final": {
			Mods: []mod.Mod{mod.NewFloat("TemporalChainsActionSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"), mod.Condition("RareOrUnique"))},
		},
	}
	// src/Data/Skills/act_dex.lua:17764
	m["TemporalRift"] = map[string]*StatMap{
		"debuff_time_passed_+%": {
			Mods: []mod.Mod{mod.NewFloat("SelfDebuffExpirationRate", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/other.lua:5002
	m["ThirstForBlood"] = map[string]*StatMap{
		"skill_life_regeneration_per_minute_per_affected_enemy": {
			Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(multiplierLimitVar("NearbyBleedingEnemies", "ThirstForBloodMaxTargets"), mod.GlobalEffect("Buff"))},
			Div:  utils.Ptr(float64(60)),
		},
		"vampiric_icon_bleeding_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagBleed).Tag(mod.GlobalEffect("Buff"))},
		},
		"vampiric_icon_max_bleeding_beam_targets": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ThirstForBloodMaxTargets", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:17848
	m["Thunderstorm"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{skill("showAverage", 1).Tag(mod.SkillPart(2))},
		},
		"base_windstorm_storm_stage_gained_per_X_ms": {
			Mods: []mod.Mod{skill("hitTimeOverride", 0).Tag(mod.Multiplier("ThunderstormStage"), mod.SkillPart(2))},
			Div:  utils.Ptr(float64(1000)),
		},
		"windstorm_storm_area_of_effect_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.Multiplier("ThunderstormStage"), mod.SkillPart(2))},
		},
		"windstorm_storm_hit_damage_+%_final_on_detonation": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.SkillPart(2))},
		},
		"windstorm_storm_maximum_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ThunderstormMaxStages", mod.TypeBase, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_dex.lua:18034
	m["Tornado"] = map[string]*StatMap{
		"tornado_base_damage_interval_ms": {
			Mods: []mod.Mod{skill("damageInterval", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/act_dex.lua:18142
	m["TornadoAltY"] = map[string]*StatMap{
		"elemental_strike_physical_damage_%_to_convert": {
			Mods: []mod.Mod{
				mod.NewFloat("SkillPhysicalDamageConvertToFire", mod.TypeBase, 0).Tag(mod.SkillPart(1)),
				mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0).Tag(mod.SkillPart(2)),
				mod.NewFloat("SkillPhysicalDamageConvertToCold", mod.TypeBase, 0).Tag(mod.SkillPart(3)),
			},
		},
		"number_of_tornados_allowed": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:TornadoofElementalTurbulenceMaxStages", mod.TypeBase, 0)},
		},
		"tornado_base_damage_interval_ms": {
			Mods: []mod.Mod{skill("damageInterval", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/act_dex.lua:16060
	m["TornadoShot"] = map[string]*StatMap{
		"tornado_shot_num_of_secondary_projectiles": {
			Mods: []mod.Mod{mod.NewFloat("tornadoShotSecondaryProjectiles", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_dex.lua:15116
	m["ToxicRainAltY"] = map[string]*StatMap{
		"toxic_rain_spores_apply_withered": {
			Mods: []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
		},
	}
	// src/Data/Skills/other.lua:5483
	m["TriggerSummonedAncestorAhuana"] = map[string]*StatMap{
		"ancestral_buff_base_spell_damage_%_suppressed": {
			Mods: []mod.Mod{mod.NewFloat("SpellSuppressionEffect", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Spirit of Ahuana"), mod.Multiplier("RamakoTattooEffect").NoFloor())},
		},
		"ancestral_embrace_effect_+%_per_ramako_tattoo": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:RamakoTattooEffect", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("RamakoTattoo").Base(1))},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/other.lua:5529
	m["TriggerSummonedAncestorAkoya"] = map[string]*StatMap{
		"ancestral_buff_base_rage_regeneration_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("RageRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Spirit of Akoya"), mod.Multiplier("TukohamaTattooEffect").NoFloor())},
			Div:  utils.Ptr(float64(60)),
		},
		"ancestral_embrace_effect_+%_per_tukohama_tattoo": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:TukohamaTattooEffect", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("TukohamaTattoo").Base(1))},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/other.lua:5576
	m["TriggerSummonedAncestorIkiaho"] = map[string]*StatMap{
		"ancestral_buff_travel_skill_cooldown_speed_+%": {
			Mods: []mod.Mod{mod.NewFloat("CooldownRecovery", mod.TypeIncrease, 0).Tag(mod.SkillType("Travel"), mod.GlobalEffect("Buff").Name("Spirit of Ikiaho"), mod.Multiplier("ArohonguiTattooEffect").NoFloor())},
		},
		"ancestral_embrace_effect_+%_per_arohongui_tattoo": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ArohonguiTattooEffect", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("ArohonguiTattoo").Base(1))},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/other.lua:5622
	m["TriggerSummonedAncestorKahuturoa"] = map[string]*StatMap{
		"ancestral_buff_armour_%_applies_to_all_damage_from_hits": {
			Mods: []mod.Mod{
				mod.NewFloat("ArmourAppliesToFireDamageTaken", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Spirit of Kahuturoa"), mod.Multiplier("RongokuraiTattooEffect").NoFloor()),
				mod.NewFloat("ArmourAppliesToColdDamageTaken", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Spirit of Kahuturoa"), mod.Multiplier("RongokuraiTattooEffect").NoFloor()),
				mod.NewFloat("ArmourAppliesToLightningDamageTaken", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Spirit of Kahuturoa"), mod.Multiplier("RongokuraiTattooEffect").NoFloor()),
				mod.NewFloat("ArmourAppliesToChaosDamageTaken", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Spirit of Kahuturoa"), mod.Multiplier("RongokuraiTattooEffect").NoFloor()),
			},
		},
		"ancestral_embrace_effect_+%_per_rongokurai_tattoo": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:RongokuraiTattooEffect", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("RongokuraiTattoo").Base(1))},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/other.lua:5671
	m["TriggerSummonedAncestorKaom"] = map[string]*StatMap{
		"ancestral_buff_damage_removed_from_your_nearest_totem_before_life_or_es_%": {
			Mods: []mod.Mod{mod.NewFloat("takenFromTotemsBeforeYou", mod.TypeBase, 0).Tag(mod.Condition("HaveTotem"), mod.GlobalEffect("Buff").Name("Spirit of Kaom"), mod.Multiplier("NgamahuTattooEffect").NoFloor())},
		},
		"ancestral_embrace_effect_+%_per_ngamahu_tattoo": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:NgamahuTattooEffect", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("NgamahuTattoo").Base(1))},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/other.lua:5717
	m["TriggerSummonedAncestorKiloava"] = map[string]*StatMap{
		"ancestral_buff_recover_%_of_maximum_life_mana_energy_shield_on_block": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:KiloavaRecoverOnBlock", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("ValakoTattooEffect").NoFloor())},
		},
		"ancestral_embrace_effect_+%_per_valako_tattoo": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ValakoTattooEffect", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("ValakoTattoo").Base(1))},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/other.lua:5768
	m["TriggerSummonedAncestorMaata"] = map[string]*StatMap{
		"ancestral_buff_flask_charges_recovered_per_3_seconds": {
			Mods: []mod.Mod{mod.NewFloat("FlaskChargesGenerated", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Spirit of Maata"), mod.Multiplier("TawhoaTattooEffect").NoFloor())},
			Div:  utils.Ptr(float64(3)),
		},
		"ancestral_embrace_effect_+%_per_tawhoa_tattoo": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:TawhoaTattooEffect", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("TawhoaTattoo").Base(1))},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/other.lua:5815
	m["TriggerSummonedAncestorRakiata"] = map[string]*StatMap{
		"ancestral_buff_action_speed_+%_minimum_value": {
			Mods: []mod.Mod{mod.NewFloat("MinimumActionSpeed", mod.TypeMAX, 0).Tag(mod.GlobalEffect("Buff").Name("Spirit of Rakiata"), mod.Multiplier("TasalioTattooEffect").Base(100).NoFloor())},
		},
		"ancestral_embrace_effect_+%_per_tasalio_tattoo": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:TasalioTattooEffect", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("TasalioTattoo").Base(1))},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/other.lua:5861
	m["TriggerSummonedAncestorTawhanuku"] = map[string]*StatMap{
		"ancestral_buff_base_mana_regeneration_rate_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("ManaRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Spirit of Tawhanuku"), mod.Multiplier("HinekoraTattooEffect").NoFloor())},
			Div:  utils.Ptr(float64(60)),
		},
		"ancestral_embrace_effect_+%_per_hinekora_tattoo": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:HinekoraTattooEffect", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("HinekoraTattoo").Base(1))},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/other.lua:5908
	m["TriggerSummonedAncestorUtula"] = map[string]*StatMap{
		"ancestral_buff_leech_%_is_instant": {
			Mods: []mod.Mod{
				mod.NewFloat("InstantLifeLeech", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Spirit of Utula"), mod.Multiplier("KitavaTattooEffect").NoFloor()),
				mod.NewFloat("InstantManaLeech", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Spirit of Utula"), mod.Multiplier("KitavaTattooEffect").NoFloor()),
				mod.NewFloat("InstantEnergyShieldLeech", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Name("Spirit of Utula"), mod.Multiplier("KitavaTattooEffect").NoFloor()),
			},
		},
		"ancestral_embrace_effect_+%_per_kitava_tattoo": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:KitavaTattooEffect", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true), mod.Multiplier("KitavaTattoo").Base(1))},
			Div:  utils.Ptr(float64(100)),
		},
	}
	// src/Data/Skills/other.lua:4656
	m["TriggeredSummonGhostOnKill"] = map[string]*StatMap{
		"phantasm_maximum_added_physical_damage_to_grant": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMax", mod.TypeBase, 0).Flag(mod.MFlagSpell).Tag(mod.PerStat(1, "ActivePhantasmLimit"), mod.GlobalEffect("Buff").Name("Phantasmal Might").EffectCond("PhantasmalMight").AllowTotemBuff(true))},
		},
		"phantasm_minimum_added_physical_damage_to_grant": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalMin", mod.TypeBase, 0).Flag(mod.MFlagSpell).Tag(mod.PerStat(1, "ActivePhantasmLimit"), mod.GlobalEffect("Buff").Name("Phantasmal Might").EffectCond("PhantasmalMight").AllowTotemBuff(true))},
		},
	}
	// src/Data/Skills/sup_int.lua:4550
	m["TriggeredSupportLivingLightning"] = map[string]*StatMap{
		"number_of_chains": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("ChainCountMax", mod.TypeBase, 0)})},
		},
	}
	// src/Data/Skills/sup_str.lua:4989
	m["TriggeredSupportVoidShockwave"] = map[string]*StatMap{
		"shockwave_chained_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/sup_dex.lua:4727
	m["TriggeredSupportVoidstorm"] = map[string]*StatMap{
		"triggered_by_voidstorm_support": {
			Mods: []mod.Mod{},
		},
		"windstorm_storm_area_of_effect_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.Multiplier("VoidstormStageAfterFirst"))},
		},
		"windstorm_storm_hit_damage_+%_final_on_detonation_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("VoidstormStageAfterFirst"))},
		},
		"windstorm_storm_maximum_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:VoidstormMaxStages", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_dex.lua:16205
	m["Unearth"] = map[string]*StatMap{
		"unearth_base_corpse_level": {
			Mods: []mod.Mod{mod.NewFloat("CorpseLevel", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/other.lua:3451
	m["UniqueMirageWarriors"] = map[string]*StatMap{
		"maximum_number_of_mirage_warriors": {
			Mods: []mod.Mod{mod.NewFloat("SaviourMirageWarriorMaxCount", mod.TypeBase, 0)},
		},
		"skill_used_by_mirage_warrior_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("SaviourMirageWarriorLessDamage", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_str.lua:210
	m["VaalAbsolution"] = map[string]*StatMap{
		"vaal_upgrade_minion_damage_+%_final": {
			Mods: []mod.Mod{},
		},
		"vaal_upgrade_minion_damage_taken_+%_final": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:501
	m["VaalAnimateWeapon"] = map[string]*StatMap{
		"base_movement_velocity_+%": {
			Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0)})},
		},
	}
	// src/Data/Skills/act_int.lua:254
	m["VaalArc"] = map[string]*StatMap{
		"arc_damage_+%_final_for_each_remaining_chain": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.PerStat(1, "ChainRemaining"))},
		},
	}
	// src/Data/Skills/act_dex.lua:669
	m["VaalArcticArmour"] = map[string]*StatMap{
		"base_energy_shield_regeneration_rate_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("EnergyShieldRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
			Div:  utils.Ptr(float64(60)),
		},
		"base_mana_regeneration_rate_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("ManaRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff"))},
			Div:  utils.Ptr(float64(60)),
		},
		"vaal_arctic_armour_damage_taken_+%_final_from_hits": {
			Mods: []mod.Mod{mod.NewFloat("VaalArcticArmourMitigation", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
		"vaal_arctic_armour_number_of_hits_absorbed": {
			Mods: []mod.Mod{mod.NewFloat("VaalArcticArmourMaxHits", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:2130
	m["VaalBladeFlurry"] = map[string]*StatMap{
		"vaal_charged_attack_damage_taken_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_dex.lua:2431
	m["VaalBladeVortex"] = map[string]*StatMap{
		"base_blade_vortex_hit_rate_ms": {
			Mods: []mod.Mod{skill("hitFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"blade_vortex_hit_rate_+%_per_blade": {
			Mods: []mod.Mod{skill("hitFrequencyPerBlade", 0)},
			Div:  utils.Ptr(float64(100)),
		},
		"vaal_blade_vortex_has_10_spinning_blades": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:2196
	m["VaalBlight"] = map[string]*StatMap{
		"display_max_blight_stacks": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:BlightMaxStages", mod.TypeBase, 0).Tag(mod.SkillPart(1))},
		},
		"hinder_enemy_chaos_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("ChaosDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Debuff").Name("Hinder"))},
		},
	}
	// src/Data/Skills/act_int.lua:2738
	m["VaalClarity"] = map[string]*StatMap{
		"no_mana_cost": {
			Mods:  []mod.Mod{mod.NewFloat("ManaCost", mod.TypeMore, 0).Tag(mod.GlobalEffect("Aura").Unscalable(true))},
			Value: utils.Ptr(float64(-100)),
		},
	}
	// src/Data/Skills/act_str.lua:1976
	m["VaalCleave"] = map[string]*StatMap{
		"cleave_+1_base_radius_per_nearby_enemy_up_to_10": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeBase, 0).Tag(mod.Multiplier("NearbyEnemies").Limit(10).LimitTotal(true))},
		},
		"vaal_cleave_executioner_area_of_effect_+%": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.SkillName("Cleave").Transfigured(true), mod.GlobalEffect("Buff").Name("Vaal Cleave").Unscalable(true))},
		},
		"vaal_cleave_executioner_damage_against_enemies_on_low_life_+%": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit|mod.KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "LowLife"), mod.SkillName("Cleave").Transfigured(true), mod.GlobalEffect("Buff").Name("Vaal Cleave").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:4927
	m["VaalCyclone"] = map[string]*StatMap{
		"cyclone_area_of_effect_+%_per_additional_melee_range": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 0).Tag(mod.Multiplier("AdditionalMeleeRange"))},
		},
	}
	// src/Data/Skills/act_dex.lua:5445
	m["VaalDetonateDead"] = map[string]*StatMap{
		"spell_maximum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMax", 0).Tag(mod.SkillPart(1))},
		},
		"spell_minimum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMin", 0).Tag(mod.SkillPart(1))},
		},
	}
	// src/Data/Skills/act_int.lua:4520
	m["VaalDiscipline"] = map[string]*StatMap{
		"energy_shield_recharge_not_delayed_by_damage": {
			Mods: []mod.Mod{mod.NewFloat("EnergyShieldRechargeNotDelayedByDamage", mod.TypeDummy, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_str.lua:3126
	m["VaalDomination"] = map[string]*StatMap{
		"vaal_upgrade_minion_damage_+%_final": {
			Mods: []mod.Mod{},
		},
		"vaal_upgrade_minion_damage_taken_+%_final": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:3520
	m["VaalEarthquake"] = map[string]*StatMap{
		"quake_slam_fully_charged_explosion_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.SkillPart(2))},
		},
		"quality_display_earthquake_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:6228
	m["VaalFirestorm"] = map[string]*StatMap{
		"vaal_firestorm_gem_explosion_area_of_effect_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffectSecondary", mod.TypeMore, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:7116
	m["VaalFlameblast"] = map[string]*StatMap{
		"charged_blast_spell_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("VaalFlameblastStage"))},
		},
		"flameblast_ailment_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("VaalFlameblastStage"))},
		},
		"vaal_flameblast_radius_+_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeBase, 0).Tag(mod.Multiplier("VaalFlameblastStageAfterFirst"))},
		},
	}
	// src/Data/Skills/act_dex.lua:8102
	m["VaalFlickerStrike"] = map[string]*StatMap{
		"active_skill_ailment_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagAilment | mod.MFlagMeleeHit)},
		},
		"base_melee_attack_repeat_count": {
			Mods: []mod.Mod{mod.NewFloat("RepeatCount", mod.TypeBase, 0)},
		},
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:9006
	m["VaalGrace"] = map[string]*StatMap{
		"chance_to_evade_attacks_%": {
			Mods: []mod.Mod{mod.NewFloat("EvadeChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_str.lua:5438
	m["VaalGroundSlam"] = map[string]*StatMap{
		"groundslam_damage_to_close_targets_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Condition("AtCloseRange"))},
		},
		"quality_display_groundslam_is_gem": {
			Mods: []mod.Mod{},
		},
		"vaal_skill_exertable": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:9171
	m["VaalHaste"] = map[string]*StatMap{
		"attack_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_movement_velocity_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"cast_speed_+%_granted_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeIncrease, 0).Flag(mod.MFlagCast).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_str.lua:8926
	m["VaalImpurityOfFire"] = map[string]*StatMap{
		"base_immune_to_ignite": {
			Mods: []mod.Mod{mod.NewFlag("IgniteImmune", true).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_maximum_fire_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("FireResistMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"hits_ignore_my_fire_resistance": {
			Mods: []mod.Mod{mod.NewFlag("SelfIgnoreFireResistance", true).Tag(mod.GlobalEffect("AuraDebuff"))},
		},
	}
	// src/Data/Skills/act_dex.lua:12311
	m["VaalImpurityOfIce"] = map[string]*StatMap{
		"base_immune_to_chill": {
			Mods: []mod.Mod{mod.NewFlag("ChillImmune", true).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_immune_to_freeze": {
			Mods: []mod.Mod{mod.NewFlag("FreezeImmune", true).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_maximum_cold_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("ColdResistMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"hits_ignore_my_cold_resistance": {
			Mods: []mod.Mod{mod.NewFlag("SelfIgnoreColdResistance", true).Tag(mod.GlobalEffect("AuraDebuff"))},
		},
	}
	// src/Data/Skills/act_int.lua:14720
	m["VaalImpurityOfLightning"] = map[string]*StatMap{
		"base_immune_to_shock": {
			Mods: []mod.Mod{mod.NewFlag("ShockImmune", true).Tag(mod.GlobalEffect("Aura"))},
		},
		"base_maximum_lightning_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("LightningResistMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"hits_ignore_my_lightning_resistance": {
			Mods: []mod.Mod{mod.NewFlag("SelfIgnoreLightningResistance", true).Tag(mod.GlobalEffect("AuraDebuff"))},
		},
	}
	// src/Data/Skills/spectre.lua:4092
	m["VaalIncursionFirestorm"] = map[string]*StatMap{
		"fire_storm_fireball_delay_ms": {
			Mods: []mod.Mod{skill("hitTimeOverride", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"firestorm_base_area_of_effect_+%": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffectPrimary", mod.TypeIncrease, 0)},
		},
	}
	// src/Data/Skills/act_dex.lua:10592
	m["VaalLightningArrow"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"projectiles_barrage": {
			Mods: []mod.Mod{},
		},
		"quality_display_base_additional_arrows_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:10910
	m["VaalLightningStrike"] = map[string]*StatMap{
		"vaal_lightning_strike_beam_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_int.lua:13019
	m["VaalLightningTrap"] = map[string]*StatMap{
		"shocked_ground_base_magnitude_override": {
			Mods: []mod.Mod{mod.NewFloat("ShockedGroundEffect", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_str.lua:7643
	m["VaalMoltenShell"] = map[string]*StatMap{
		"base_physical_damage_reduction_rating": {
			Mods: []mod.Mod{mod.NewFloat("Armour", mod.TypeBase, 0).Tag(mod.GlobalEffect("Guard"))},
		},
		"molten_shell_%_of_absorbed_damage_dealt_as_reflected_fire": {
			Mods: []mod.Mod{skill("moltenShellReflect", 0)},
		},
		"molten_shell_damage_absorb_limit_%_of_armour": {
			Mods: []mod.Mod{mod.NewFloat("GuardAbsorbLimit", mod.TypeBase, 0).Tag(mod.PerStat(1, "Armour").LimitVar("GuardAbsorbMax").LimitTotal(true), mod.GlobalEffect("Guard").Unscalable(true))},
			Div:  utils.Ptr(float64(100)),
		},
		"molten_shell_damage_absorbed_%": {
			Mods: []mod.Mod{mod.NewFloat("GuardAbsorbRate", mod.TypeBase, 0).Tag(mod.GlobalEffect("Guard").Unscalable(true))},
		},
		"molten_shell_max_damage_absorbed": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:GuardAbsorbMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Guard").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_str.lua:8143
	m["VaalMoltenStrike"] = map[string]*StatMap{
		"active_skill_hit_ailment_damage_with_projectile_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_int.lua:14272
	m["VaalPowerSiphon"] = map[string]*StatMap{
		"critical_strike_chance_+%_final_per_power_charge_from_power_siphon": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeMore, 0).Tag(mod.Multiplier("PowerCharge"))},
		},
		"power_siphon_fire_at_all_targets": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_str.lua:11231
	m["VaalReap"] = map[string]*StatMap{
		"base_physical_damage_to_deal_per_minute": {
			Mods: []mod.Mod{
				skill("PhysicalDot", 0).Tag(mod.Condition("ReapDebuffIsFireDamage").Neg(true)),
				skill("FireDot", 0).Tag(mod.Condition("ReapDebuffIsFireDamage")),
			},
			Div: utils.Ptr(float64(60)),
		},
		"blood_scythe_damage_+%_final_per_charge": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Multiplier("BloodCharge"))},
		},
		"vaal_reap_additional_maximum_blood_charges": {
			Mods: []mod.Mod{mod.NewFloat("BloodChargesMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Buff").Unscalable(true))},
		},
	}
	// src/Data/Skills/act_dex.lua:12895
	m["VaalReave"] = map[string]*StatMap{
		"reave_additional_max_stacks": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:ReaveMaxStages", mod.TypeBase, 0)},
		},
		"reave_area_of_effect_+%_final_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", mod.TypeMore, 0).Tag(mod.Multiplier("ReaveStage"))},
		},
	}
	// src/Data/Skills/act_str.lua:9411
	m["VaalRejuvenationTotem"] = map[string]*StatMap{
		"base_life_regeneration_rate_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
			Div:  utils.Ptr(float64(60)),
		},
		"vaal_rejuvenation_totem_%_damage_taken_applied_to_totem_instead": {
			Mods: []mod.Mod{mod.NewFloat("takenFromVaalRejuvenationTotemsBeforeYou", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_int.lua:15632
	m["VaalRighteousFire"] = map[string]*StatMap{
		"vaal_righteous_fire_life_and_es_%_as_damage_per_second": {
			Mods: []mod.Mod{skill("RFMultiplier", 0)},
			Div:  utils.Ptr(float64(100)),
		},
		"vaal_righteous_fire_life_and_es_%_to_lose_on_use": {
			Mods: []mod.Mod{skill("percentSacrificed", 0)},
			Div:  utils.Ptr(float64(100)),
		},
		"vaal_righteous_fire_spell_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell).Tag(mod.GlobalEffect("Buff"))},
		},
	}
	// src/Data/Skills/act_str.lua:10550
	m["VaalSmite"] = map[string]*StatMap{
		"base_chance_to_shock_%_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("EnemyShockChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"maximum_added_lightning_damage_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"minimum_added_lightning_damage_from_skill": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_dex.lua:14848
	m["VaalSpectralThrow"] = map[string]*StatMap{
		"base_number_of_projectiles_in_spiral_nova": {
			Mods: []mod.Mod{mod.NewFloat("ProjectileCount", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:20013
	m["VaalSummonSkeletons"] = map[string]*StatMap{
		"quality_display_summon_skeleton_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_dex.lua:18373
	m["VampiricLink"] = map[string]*StatMap{
		"life_leech_is_applied_to_remora_link_targets_instead": {
			Mods: []mod.Mod{
				mod.NewFlag("CannotLeechLife", true).Tag(mod.GlobalEffect("Buff")),
				mod.NewFlag("MaximumLifeLeechIsEqualToParent", true).Tag(mod.GlobalEffect("Link")),
			},
		},
		"remora_link_grants_maximum_life_leech_rate_%_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("MaxLifeLeechRate", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Buff"))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/act_str.lua:12335
	m["VengefulCry"] = map[string]*StatMap{
		"display_retaliation_use_requirement_variation": {
			Mods: []mod.Mod{},
		},
		"rage_warcry_gain_X_rage_per_minute_per_5_monster_power_max_25_power": {
			Mods: []mod.Mod{
				mod.NewFloat("RageRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Warcry").Div(5).Limit(25)),
				mod.NewFlag("Condition:CanGainRage", true).Tag(mod.GlobalEffect("Warcry")),
				mod.NewFlag("InherentRageLossIsPrevented", true).Tag(mod.GlobalEffect("Warcry")),
			},
			Div: utils.Ptr(float64(60)),
		},
		"rage_warcry_maximum_rage_+": {
			Mods: []mod.Mod{mod.NewFloat("MaximumRage", mod.TypeBase, 0).Tag(mod.GlobalEffect("Warcry"))},
		},
	}
	// src/Data/Skills/act_dex.lua:16469
	m["ViperStrike"] = map[string]*StatMap{
		"viper_strike_dual_wield_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.Condition("DualWielding"))},
		},
		"viper_strike_dual_wield_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Condition("DualWielding"))},
		},
	}
	// src/Data/Skills/act_dex.lua:16560
	m["ViperStrikeAltX"] = map[string]*StatMap{
		"active_skill_poison_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagPoison)},
		},
		"quality_display_active_skill_poison_damage_final_is_gem": {
			Mods: []mod.Mod{},
		},
		"viper_strike_dual_wield_attack_speed_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Speed", mod.TypeMore, 0).Flag(mod.MFlagAttack).Tag(mod.Condition("DualWielding"))},
		},
		"viper_strike_dual_wield_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagHit).Tag(mod.Condition("DualWielding"))},
		},
	}
	// src/Data/Skills/act_str.lua:12503
	m["Vitality"] = map[string]*StatMap{
		"base_life_regeneration_rate_per_minute": {
			Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 0).Tag(mod.GlobalEffect("Aura"))},
			Div:  utils.Ptr(float64(60)),
		},
	}
	// src/Data/Skills/other.lua:5316
	m["VoidGaze"] = map[string]*StatMap{
		"base_chaos_damage_resistance_%": {
			Mods: []mod.Mod{mod.NewFloat("ChaosResist", mod.TypeBase, 0).Tag(mod.GlobalEffect("Debuff"))},
		},
	}
	// src/Data/Skills/act_int.lua:20270
	m["VoidSphere"] = map[string]*StatMap{
		"base_blackhole_tick_rate_ms": {
			Mods: []mod.Mod{skill("repeatFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/act_int.lua:20359
	m["VoidSphereAltX"] = map[string]*StatMap{
		"base_blackhole_tick_rate_ms": {
			Mods: []mod.Mod{skill("repeatFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
	}
	// src/Data/Skills/act_dex.lua:16666
	m["VolatileDead"] = map[string]*StatMap{
		"spell_maximum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMax", 0).Tag(mod.SkillPart(1))},
		},
		"spell_minimum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMin", 0).Tag(mod.SkillPart(1))},
		},
	}
	// src/Data/Skills/act_dex.lua:16769
	m["VolatileDeadAltX"] = map[string]*StatMap{
		"spell_maximum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMax", 0).Tag(mod.SkillPart(1))},
		},
		"spell_minimum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMin", 0).Tag(mod.SkillPart(1))},
		},
	}
	// src/Data/Skills/act_dex.lua:16873
	m["VolatileDeadAltY"] = map[string]*StatMap{
		"spell_maximum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMax", 0).Tag(mod.SkillPart(1))},
		},
		"spell_minimum_base_fire_damage": {
			Mods: []mod.Mod{skill("FireMin", 0).Tag(mod.SkillPart(1))},
		},
	}
	// src/Data/Skills/act_str.lua:12600
	m["VolcanicFissure"] = map[string]*StatMap{
		"active_skill_hit_ailment_damage_with_projectile_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.SkillPart(2))},
		},
	}
	// src/Data/Skills/act_int.lua:20449
	m["VoltaxicBurst"] = map[string]*StatMap{
		"quality_display_voltaxic_burst_is_gem": {
			Mods: []mod.Mod{},
		},
		"voltaxic_burst_hit_and_ailment_damage_+%_final_per_stack": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("VoltaxicWaitingStages"))},
		},
	}
	// src/Data/Skills/act_int.lua:20632
	m["VortexAltX"] = map[string]*StatMap{
		"active_skill_if_used_through_frostbolt_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Condition("CastOnFrostbolt"))},
		},
	}
	// src/Data/Skills/act_str.lua:12856
	m["Vulnerability"] = map[string]*StatMap{
		"physical_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"receive_bleeding_chance_%_when_hit_by_attack": {
			Mods: []mod.Mod{mod.NewFloat("SelfBleedChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"self_bleed_duration_+%": {
			Mods: []mod.Mod{mod.NewFloat("SelfBleedDuration", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_str.lua:13033
	m["WarBanner"] = map[string]*StatMap{
		"accuracy_rating_+%": {
			Mods: []mod.Mod{mod.NewFloat("Accuracy", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Aura"))},
		},
		"war_banner_physical_damage_+%_final_with_melee_skills": {
			Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", mod.TypeMore, 0).Tag(mod.GlobalEffect("Aura"), mod.SkillType("Melee"))},
		},
	}
	// src/Data/Skills/other.lua:5379
	m["WardShatter"] = map[string]*StatMap{
		"active_skill_damage_+%_final_if_ward_has_not_broken_in_the_past_2_seconds": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Tag(mod.Condition("WardBrokenPast2Seconds").Neg(true))},
		},
		"from_code_active_skill_ailment_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagAilment)},
		},
	}
	// src/Data/Skills/act_str.lua:12946
	m["WarlordsMark"] = map[string]*StatMap{
		"enemy_chance_to_double_stun_duration_%_vs_self": {
			Mods: []mod.Mod{mod.NewFloat("SelfDoubleStunDurationChance", mod.TypeBase, 0).Tag(mod.GlobalEffect("Curse"))},
		},
		"enemy_grant_X_rage_on_stun": {
			Mods: []mod.Mod{mod.NewFlag("Condition:CanGainRage", true).Tag(mod.GlobalEffect("Buff"))},
		},
		"life_leech_on_any_damage_when_hit_by_attack_permyriad": {
			Mods: []mod.Mod{mod.NewFloat("SelfDamageLifeLeech", mod.TypeBase, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Curse"))},
		},
		"mana_leech_on_any_damage_when_hit_by_attack_permyriad": {
			Mods: []mod.Mod{mod.NewFloat("SelfDamageManaLeech", mod.TypeBase, 0).Flag(mod.MFlagAttack).Tag(mod.GlobalEffect("Curse"))},
		},
	}
	// src/Data/Skills/act_int.lua:20801
	m["WaveOfConviction"] = map[string]*StatMap{
		"purge_expose_resist_%_matching_highest_element_damage": {
			Mods: []mod.Mod{
				mod.NewFloat("FireExposure", mod.TypeBase, 0).Tag(mod.GlobalEffect("Debuff").Name("Fire Exposure").EffectCond("WaveOfConvictionFireExposureActive")),
				mod.NewFloat("ColdExposure", mod.TypeBase, 0).Tag(mod.GlobalEffect("Debuff").Name("Cold Exposure").EffectCond("WaveOfConvictionColdExposureActive")),
				mod.NewFloat("LightningExposure", mod.TypeBase, 0).Tag(mod.GlobalEffect("Debuff").Name("Lightning Exposure").EffectCond("WaveOfConvictionLightningExposureActive")),
			},
		},
	}
	// src/Data/Skills/act_dex.lua:17094
	m["WildStrike"] = map[string]*StatMap{
		"elemental_strike_physical_damage_%_to_convert": {
			Mods: []mod.Mod{
				mod.NewFloat("SkillPhysicalDamageConvertToFire", mod.TypeBase, 0).Tag(mod.SkillPartList(1, 2)),
				mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0).Tag(mod.SkillPartList(3, 4)),
				mod.NewFloat("SkillPhysicalDamageConvertToCold", mod.TypeBase, 0).Tag(mod.SkillPartList(5, 6)),
			},
		},
	}
	// src/Data/Skills/act_dex.lua:17235
	m["WildStrikeAltX"] = map[string]*StatMap{
		"elemental_strike_physical_damage_%_to_convert": {
			Mods: []mod.Mod{
				mod.NewFloat("SkillPhysicalDamageConvertToFire", mod.TypeBase, 0).Tag(mod.SkillPartList(1, 2)),
				mod.NewFloat("SkillPhysicalDamageConvertToLightning", mod.TypeBase, 0).Tag(mod.SkillPartList(3, 4)),
				mod.NewFloat("SkillPhysicalDamageConvertToCold", mod.TypeBase, 0).Tag(mod.SkillPartList(5, 6)),
			},
		},
	}
	// src/Data/Skills/act_int.lua:20988
	m["WinterOrb"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"display_frost_fury_additive_cast_speed_modifiers_apply_to_fire_speed": {
			Mods: []mod.Mod{},
		},
		"frost_fury_base_fire_interval_ms": {
			Mods: []mod.Mod{skill("repeatFrequency", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"frost_fury_duration_+%_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("Duration", mod.TypeIncrease, 0).Tag(mod.Multiplier("WinterOrbStageAfterFirst"))},
		},
		"frost_fury_fire_speed_+%_final_while_channelling": {
			Mods: []mod.Mod{mod.NewFloat("HitRate", mod.TypeMore, 0).Tag(mod.SkillPart(1))},
		},
		"frost_fury_fire_speed_+%_per_stage": {
			Mods: []mod.Mod{mod.NewFloat("HitRate", mod.TypeIncrease, 0).Tag(mod.Multiplier("WinterOrbStageAfterFirst"))},
		},
		"frost_fury_max_number_of_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:WinterOrbMaxStages", mod.TypeBase, 0)},
		},
		"quality_display_winter_orb_is_gem": {
			Mods: []mod.Mod{},
		},
	}
	// src/Data/Skills/act_int.lua:21141
	m["WintertideBrand"] = map[string]*StatMap{
		"base_skill_show_average_damage_instead_of_dps": {
			Mods: []mod.Mod{},
		},
		"immolation_brand_burn_damage_+%_final_per_stage": {
			Mods: []mod.Mod{
				mod.NewFloat("Damage", mod.TypeMore, 0).Tag(multiplierLimitVar("WintertideBrandStage", "WintertideBrandMaxStages"), mod.SkillPart(2)),
				mod.NewFloat("Multiplier:WintertideBrandDamagePerStage", mod.TypeBase, 0),
			},
		},
		"quality_display_wintertide_brand_is_gem": {
			Mods: []mod.Mod{},
		},
		"winter_brand_max_number_of_stages": {
			Mods: []mod.Mod{mod.NewFloat("Multiplier:WintertideBrandMaxStages", mod.TypeBase, 0)},
		},
	}
	// src/Data/Skills/act_int.lua:21240
	m["Wither"] = map[string]*StatMap{
		"active_skill_withered_base_duration_ms": {
			Mods: []mod.Mod{skill("duration", 0)},
			Div:  utils.Ptr(float64(1000)),
		},
		"base_movement_velocity_+%": {
			Mods: []mod.Mod{mod.NewFloat("MovementSpeed", mod.TypeIncrease, 0).Tag(mod.GlobalEffect("Debuff").Name("Withered"))},
		},
		"base_skill_effect_duration": {
			Mods: []mod.Mod{},
		},
		"chaos_damage_taken_+%": {
			Mods: []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
		},
	}
	// src/Data/Skills/act_dex.lua:17321
	m["WitheringStep"] = map[string]*StatMap{
		"quality_display_withering_step_is_gem": {
			Mods: []mod.Mod{},
		},
		"slither_wither_stacks": {
			Mods: []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
		},
	}
	// src/Data/Skills/act_int.lua:21328
	m["Wrath"] = map[string]*StatMap{
		"attack_maximum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMax", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"attack_minimum_added_lightning_damage": {
			Mods: []mod.Mod{mod.NewFloat("LightningMin", mod.TypeBase, 0).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Aura"))},
		},
		"wrath_aura_spell_lightning_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("LightningDamage", mod.TypeMore, 0).Flag(mod.MFlagSpell).Tag(mod.GlobalEffect("Aura"))},
		},
	}
	// src/Data/Skills/act_int.lua:21413
	m["Zealotry"] = map[string]*StatMap{
		"spell_critical_strike_chance_+%": {
			Mods: []mod.Mod{mod.NewFloat("CritChance", mod.TypeIncrease, 0).Flag(mod.MFlagSpell).Tag(mod.GlobalEffect("Aura"))},
		},
		"spell_damage_aura_spell_damage_+%_final": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0).Flag(mod.MFlagSpell).Tag(mod.GlobalEffect("Aura"))},
		},
	}
}
