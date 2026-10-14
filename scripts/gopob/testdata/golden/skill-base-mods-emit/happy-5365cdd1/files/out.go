package raw

import "github.com/Vilsol/go-pob/mod"

// SkillBaseMods is PoB's hand-written per-skill `baseMods` table
// (src/Data/Skills/*.lua), keyed by granted-effect id. PoB applies it verbatim at the end
// of calcs.mergeSkillInstanceMods (src/Modules/CalcActiveSkill.lua:69,
// `modList:AddList(grantedEffect.baseMods)`) for every gem effect — active gems and
// supports alike — and go-pob-data carries none of it, so this table is its only source.
//
// A baseMods entry is either the `skill(key, value)` helper, which
// src/Modules/Data.lua:60 expands to mod("SkillData", "LIST", {key=, value=}), or an
// ordinary mod() with its own type, flags, keywordFlags and tags. Both shapes are here.
// Every mod carries source "Skill:<id>", which src/Modules/Data.lua:64 (processMod) sets
// and which CalcActiveSkill's global-effect pass compares against to decide whether a
// buff inherits the skill's buffNotPlayer/buffMinions/buffAllies/allowTotemBuff config.
//
// Generated from PoB v2.18.1 (9 skills with baseMods, 18 mods) by
// oracle/skill_base_mods.lua + oracle/skill_base_mods_emit.py; 20 mods are emitted,
// 27 are not representable and are omitted:
//   - 4 unhandled mod type HEAVY
//   - 2 unhandled mod type MID
//   - 1 MinionModifier value carries no nested mod table
//   - 1 unhandled mod type WEIRD
//   - 1 keywordFlags 16 are not a union of named KeywordFlag bits
//   - 1 flags 32 are not a union of named MFlag bits
//   - 1 string-valued skill() entries (triggerSource, corpseMonsterVariety) whose key has
//     no string field on calculator.SkillData
//   - 1 skill() entry whose key has no field on calculator.SkillData
//     (noSuchField), which utils.Set drops as an invalid key rather than storing
//   - 1 unhandled mod type T01
//   - 1 unhandled mod type T02
//   - 1 unhandled mod type T03
//   - 1 unhandled mod type T04
//   - 1 unhandled mod type T05
//   - 1 unhandled mod type T06
//   - 1 unhandled mod type T07
//   - 1 unhandled mod type T08
//   - 1 unhandled mod type T09
//   - 1 unhandled mod type T10
//   - 1 unhandled mod type T11
//   - 1 unhandled mod type T12
//   - 1 unhandled mod type T13
//   - 1 unhandled mod type T14
//   - 1 flags 1073741824 are not a union of named MFlag bits
//
// SkillPart-tagged mods are ALL carried, for every part and for PoB's `skillPartList`
// form: moddb evaluates the tag (ModStore.lua:529-549 -> moddb/mod_store.go
// evalSkillPartTag) with PoB's drop-when-not-matching semantics, so a mod written for
// part 3 reaches only a build on part 3. The earlier revision of this table kept part 1
// alone and dropped 51 part-gated mods (its header said 55, which was its own tally's
// classification, not a count of mods), because the tag was then a TODO and keeping them
// would have applied every part of a skill at once; that reason is gone. It also flattened
// a `skillPartList` containing 1 down to `mod.SkillPart(1)` — Magma Sigil's showAverage and
// Wild Strike's fire conversion, both {1, 2} — which silently dropped the second part.
//
// One tag is deliberately not emitted rather than omitted with its mod: ChargedDash's
// hitTimeMultiplier carries `{type = "Skill", skillPartList = {1, 2}}`
// (src/Data/Skills/act_dex.lua:1838), and ModStore.lua has no "Skill" branch, so upstream
// the tag gates nothing and the mod always applies. The mod is emitted untagged to match.
var SkillBaseMods = map[string][]mod.Mod{
	// src/Data/Skills/act_str.lua:100
	"AlphaSkill": {
		mod.NewFloat("Damage", mod.TypeIncrease, 25).Flag(mod.MFlagAttack | mod.MFlagMelee).KeywordFlag(mod.KeywordFlagAttack).Source("Skill:AlphaSkill"),
		mod.NewList("SkillData", mod.SkillData{Key: "CritChance", Value: 6.5}).Source("Skill:AlphaSkill"),
		mod.NewList("SkillData", mod.SkillData{Key: "CannotBeEvaded", Value: 1}).Source("Skill:AlphaSkill"),
	},
	// src/Data/Skills/act_dex.lua:200
	"BetaSkill": {
		mod.NewFloat("Speed", mod.TypeMore, 10).Tag(mod.Multiplier("Frenzy").Limit(3).Actor("parent")).Source("Skill:BetaSkill"),
		mod.NewFloat("Life", mod.TypeBase, 40).Tag(perStatBase(5, 2, "Str")).Source("Skill:BetaSkill"),
		mod.NewFloat("Armour", mod.TypeIncrease, 15).Tag(mod.Condition("Blocking").Neg(true)).Source("Skill:BetaSkill"),
	},
	// src/Data/Skills/other.lua:400
	"DeltaSkill": {
		mod.NewFloat("MinionDamage", mod.TypeIncrease, 30).Tag(mod.ActorCondition("enemy", "Shocked")).Source("Skill:DeltaSkill"),
		mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "MinionInstability"}).Source("Skill:DeltaSkill"),
		mod.NewFloat("HitTime", mod.TypeBase, 1).Source("Skill:DeltaSkill"),
	},
	// src/Data/Skills
	"EmptyLoc": {
		mod.NewFloat("Kept", mod.TypeBase, 2).Source("Skill:EmptyLoc"),
	},
	// src/Data/Skills/act_int.lua:300
	"GammaSkill": {
		mod.NewFloat("Damage", mod.TypeMAX, 3).Tag(mod.SkillType("Trappable")).Source("Skill:GammaSkill"),
		mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 20).Tag(mod.SkillPartList(1, 2)).Source("Skill:GammaSkill"),
		mod.NewFlag("Buff", true).Tag(mod.GlobalEffect("Buff").Name("Fortify")).Source("Skill:GammaSkill"),
	},
	// src/Data/Skills/other.lua:900
	"MaskBits": {
		mod.NewFloat("AreaDamage", mod.TypeIncrease, 7).Flag(mod.MFlagArea).Source("Skill:MaskBits"),
	},
	// src/Data/Skills
	"NoLoc": {
		mod.NewFloat("Life", mod.TypeBase, 5).Tag(mod.MultiplierThreshold("Rage").Threshold(10).Upper(true)).Source("Skill:NoLoc"),
	},
	// src/Data/Skills/other.lua:600
	"OmitTieA": {
		mod.NewFloat("Kept", mod.TypeBase, 1).Source("Skill:OmitTieA"),
	},
	// src/Data/Skills/other.lua:700
	"OmitTieB": {
		mod.NewFloat("Kept", mod.TypeBase, 1).Source("Skill:OmitTieB"),
	},
	// src/Data/Skills/other.lua:1000
	"TieFarm": {
		mod.NewFloat("Kept", mod.TypeBase, 1).Source("Skill:TieFarm"),
	},
	// src/Data/Skills/other.lua:800
	"UnnamedBit": {
		mod.NewFloat("Kept", mod.TypeBase, 1).Source("Skill:UnnamedBit"),
	},
	// src/Data/Skills/other.lua:1100
	"ZHeavy": {
		mod.NewFloat("Kept", mod.TypeBase, 1).Source("Skill:ZHeavy"),
	},
}

// multiplierLimitVar builds a Multiplier tag limited by another multiplier rather than a
// constant (PoB's `limitVar`). mod.MultiplierTag has the field but no setter.
func multiplierLimitVar(variable string, limitVar string) *mod.MultiplierTag {
	t := mod.Multiplier(variable)
	t.TagLimitVariable = &limitVar
	return t
}

// perStatBase builds a PerStat tag with a flat addend (PoB's `base`). mod.PerStatTag has
// the field but no setter.
func perStatBase(divide float64, base float64, stats ...string) *mod.PerStatTag {
	t := mod.PerStat(divide, stats...)
	t.Base = base
	return t
}
