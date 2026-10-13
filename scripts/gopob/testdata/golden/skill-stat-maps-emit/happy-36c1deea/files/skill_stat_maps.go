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
// 2 entries are empty because PoB DECLARES the stat with no mods (1 of them suppress a real
// global entry). Those are deliberate: emitting the entry keeps the fallback suppressed the way
// PoB does, and dropping it would apply a global mapping PoB never uses.
//
// 14 are empty because every mod in them was OMITTED below (1 of them suppress a real global
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
// Generated from the PINNED PoB (5 skills, 29 statMap entries, 32 mods) by
// oracle/skill_stat_map.lua + gopob's skill-stat-maps emitter. The pin's identity is in
// scripts/pob_pin.json; this header does not restate it, because the per-skill dump does
// not carry pobVersion and a hardcoded version is how this line came to claim "v2.18.1"
// long after the pin moved to 3.29. 26 mods are emitted;
// 14 are not representable and are omitted rather than mapped lossily:
//   - 3 unhandled mod type NOSUCHTYPE [unverifiable: the mod type is data-dependent]
//   - 1 GlobalEffect tag carries nosuchkey, which mod.GlobalEffectTag cannot express [go:mod.GlobalEffectTag]
//   - 1 Multiplier tag carries nosuchkey, which mod.MultiplierTag cannot express [go:mod.MultiplierTag]
//   - 1 mod has no 'value'
//   - 1 MultiplierThreshold tag carries neither threshold nor thresholdVar [pob:src/Classes/ModStore.lua:331]
//   - 1 MultiplierThreshold tag carries varList, which mod.MultiplierThresholdTag cannot express [go:mod.MultiplierThresholdTag]
//   - 1 mod carries 'nosuchmodkey' beyond the known mod fields, which this emitter neither models nor declares [go:mod.Mod]
//   - 1 nested mod group carrying the per-mod scalar 'div', which raw.StatMap cannot express: its Div/Mult are per-ENTRY and PoB writes one per GROUP [go:raw.StatMap]
//   - 1 SkillData LIST value carries nosuchfield, which mod.SkillData cannot express [go:mod.SkillData]
//   - 1 flags 4096 are not a union of named MFlag bits [unverifiable: the flag value is data-dependent; MFLAG_BITS in this file is the named set]
//   - 1 non-dict tag: a bare int in the tags array, from a skill() call written as if it took flags positionally [pob:Modules/Data.lua:52]
//   - 1 unknown tag type NoSuchTagType [unverifiable: a new tag type appears only when PoB adds one]
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
var SkillStatMaps = map[string]map[string]*StatMap{
	// src/Data/Skills/act_str.lua:100
	"AlphaSkill": {
		"energy_shield_delay_-%": {
			Mods: []mod.Mod{mod.NewFloat("EnergyShieldRechargeFaster", mod.TypeIncrease, 1).Tag(mod.GlobalEffect("Aura").Name("Discipline").Unscalable(true).EffectCond("HaveDiscipline").EffectEnemyCond("Shocked").ModCond("Focused").EffectStackVar("Rage").EffectStackLimit(5).EffectStackLimitVar("MaxRage").ApplyNotPlayer(true).ApplyMinions(true))},
			Value: utils.Ptr(float64(1)),
		},
		"globaleffect_unmodelled": {
			Mods: []mod.Mod{},
		},
		"multiplier_stat": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 5).Flag(mod.MFlagAttack | mod.MFlagMelee).KeywordFlag(mod.KeywordFlagAttack).Tag(multiplierLimitVar("Virulence", "MaxV").Div(2).Limit(40).LimitTotal(true).Actor("parent").LimitActor("enemy"))},
			Mult: utils.Ptr(float64(2)),
			Div: utils.Ptr(float64(3)),
			Base: utils.Ptr(float64(4)),
		},
		"multiplier_unmodelled": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeIncrease, 1).Tag(mod.Multiplier("X").NoFloor())},
		},
		"multiplier_unrepresentable": {
			Mods: []mod.Mod{},
		},
		"nameless_mod": {
			Mods: []mod.Mod{},
		},
		"threshold_neither": {
			Mods: []mod.Mod{},
		},
		"threshold_stat": {
			Mods: []mod.Mod{mod.NewFloat("Life", mod.TypeBase, 10).Tag(mod.MultiplierThreshold("Rage").ThresholdVar("MaxRage").Upper(true).Actor("parent"))},
		},
		"threshold_unmodelled": {
			Mods: []mod.Mod{},
		},
		"threshold_zero": {
			Mods: []mod.Mod{mod.NewFloat("Life", mod.TypeBase, 1).Tag(mod.MultiplierThreshold("Rage").Threshold(0))},
		},
	},
	// src/Data/Skills/act_dex.lua:200
	"BetaSkill": {
		"condition_list": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeIncrease, 1).Tag(mod.Condition("A", "B").Neg(true), mod.ActorCondition("enemy", "Shocked"))},
		},
		"percentstat_literal": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeBase, 1).Tag(mod.PercentStat("Mana", 25))},
		},
		"percentstat_var": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeBase, 1).Tag(mod.PercentStatVar("Mana", "ArcaneCloakConsumed"))},
		},
		"perstat_list": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeIncrease, 1).Tag(mod.PerStat(10, "Str", "Dex").LimitVar("Cap").Limit(50).LimitTotal(true).Actor("parent"))},
		},
		"skilltype_absent": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMAX, 1).Tag(mod.SkillType().Neg(true))},
		},
		"skilltype_named": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMAX, 1).Tag(mod.SkillType("Trappable").Neg(true))},
		},
	},
	// src/Data/Skills/other.lua:400
	"DeltaSkill": {
		"declared_empty_stat": {
			Mods: []mod.Mod{},
		},
		"declared_empty_unshadowed": {
			Mods: []mod.Mod{},
		},
		"list_mods": {
			Mods: []mod.Mod{
				mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "MinionInstability"}),
				mod.NewList("SupportedGemProperty", mod.SupportedGemProperty{Keyword: "aura", Key: "level", Value: 0}),
				mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeMore, 0).Tag(mod.GlobalEffect("Buff"))}),
			},
		},
		"named_mod_with_extra_key": {
			Mods: []mod.Mod{},
		},
		"nameless_mod_with_extra_key": {
			Mods: []mod.Mod{},
		},
		"shadowed_empty_stat": {
			Mods: []mod.Mod{},
		},
		"skilldata_absent": {
			Mods: []mod.Mod{skill("levelPlaceholder", 0)},
		},
		"skilldata_false": {
			Mods: []mod.Mod{skill("dotIsSpell", 0)},
		},
		"skilldata_merge": {
			Mods: []mod.Mod{mod.NewList("SkillData", mod.SkillData{Key: "chaosDot", Value: 5, Merge: "MAX"})},
		},
		"skilldata_true": {
			Mods: []mod.Mod{skill("showAverage", 1)},
		},
		"skilldata_unmodelled": {
			Mods: []mod.Mod{},
		},
		"unnamed_bit": {
			Mods: []mod.Mod{},
		},
		"unshadowed_empty_stat": {
			Mods: []mod.Mod{},
		},
	},
	// src/Data/Skills/act_str.lua:500
	"EpsilonSkill": {
		"bare_mod_beside_a_group": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeIncrease, 1)},
			Groups: []*StatMap{
				{
					Mods: []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeMore, 1)},
					Mult: utils.Ptr(float64(-1)),
				},
			},
			Mult: utils.Ptr(float64(7)),
		},
		"group_wholly_omitted": {
			Mods: []mod.Mod{},
		},
		"group_with_no_scalar": {
			Groups: []*StatMap{
				{
					Mods: []mod.Mod{mod.NewFloat("Life", mod.TypeBase, 1)},
				},
			},
		},
		"two_groups_two_scalars": {
			Groups: []*StatMap{
				{
					Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 1)},
					Div: utils.Ptr(float64(60)),
				},
				{
					Mods: []mod.Mod{mod.NewFloat("ManaRegen", mod.TypeBase, 1)},
					Div: utils.Ptr(float64(400)),
					Base: utils.Ptr(float64(2)),
				},
			},
		},
	},
	// src/Data/Skills
	"GammaSkill": {
		"meleeproximity": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 1).Tag(mod.MeleeProximity([]int{10, 20}))},
		},
		"misc_tags": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeIncrease, 1).Tag(mod.SkillName("Cyclone").Neg(true), mod.SkillId("CycloneId"), mod.SkillPartList(1, 2), mod.StatThreshold("Str", 100).Upper(true))},
		},
		"nondict_tag": {
			Mods: []mod.Mod{},
		},
		"ramp_tags": {
			Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 1).Tag(mod.DistanceRamp([][]float64{{0, 1}, {40, 0}}), mod.ModFlagOr(mod.MFlagSpell | mod.MFlagMelee), mod.LimitVar("Cap"))},
		},
		"unknown_tag": {
			Mods: []mod.Mod{},
		},
	},
}
