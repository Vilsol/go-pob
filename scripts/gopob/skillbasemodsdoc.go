package main

import "strings"

var sbmHeader = strings.ReplaceAll(sbmHeaderT, "@", "`")

var sbmFooter = strings.ReplaceAll(sbmFooterT, "@", "`")

const sbmHeaderT = `package raw

import "github.com/Vilsol/go-pob/mod"

// SkillBaseMods is PoB's hand-written per-skill @baseMods@ table
// (src/Data/Skills/*.lua), keyed by granted-effect id. PoB applies it verbatim at the end
// of calcs.mergeSkillInstanceMods (src/Modules/CalcActiveSkill.lua:69,
// @modList:AddList(grantedEffect.baseMods)@) for every gem effect — active gems and
// supports alike — and go-pob-data carries none of it, so this table is its only source.
//
// A baseMods entry is either the @skill(key, value)@ helper, which
// src/Modules/Data.lua:60 expands to mod("SkillData", "LIST", {key=, value=}), or an
// ordinary mod() with its own type, flags, keywordFlags and tags. Both shapes are here.
// Every mod carries source "Skill:<id>", which src/Modules/Data.lua:64 (processMod) sets
// and which CalcActiveSkill's global-effect pass compares against to decide whether a
// buff inherits the skill's buffNotPlayer/buffMinions/buffAllies/allowTotemBuff config.
//
// Generated from PoB v2.18.1 (%s skills with baseMods, %s mods) by
// oracle/skill_base_mods.lua + oracle/skill_base_mods_emit.py; %d mods are emitted,
// %d are not representable and are omitted:
%s
//
// SkillPart-tagged mods are ALL carried, for every part and for PoB's @skillPartList@
// form: moddb evaluates the tag (ModStore.lua:529-549 -> moddb/mod_store.go
// evalSkillPartTag) with PoB's drop-when-not-matching semantics, so a mod written for
// part 3 reaches only a build on part 3. The earlier revision of this table kept part 1
// alone and dropped 51 part-gated mods (its header said 55, which was its own tally's
// classification, not a count of mods), because the tag was then a TODO and keeping them
// would have applied every part of a skill at once; that reason is gone. It also flattened
// a @skillPartList@ containing 1 down to @mod.SkillPart(1)@ — Magma Sigil's showAverage and
// Wild Strike's fire conversion, both {1, 2} — which silently dropped the second part.
//
// One tag is deliberately not emitted rather than omitted with its mod: ChargedDash's
// hitTimeMultiplier carries @{type = "Skill", skillPartList = {1, 2}}@
// (src/Data/Skills/act_dex.lua:1838), and ModStore.lua has no "Skill" branch, so upstream
// the tag gates nothing and the mod always applies. The mod is emitted untagged to match.
var SkillBaseMods = map[string][]mod.Mod{
`

const sbmFooterT = `
// multiplierLimitVar builds a Multiplier tag limited by another multiplier rather than a
// constant (PoB's @limitVar@). mod.MultiplierTag has the field but no setter.
func multiplierLimitVar(variable string, limitVar string) *mod.MultiplierTag {
	t := mod.Multiplier(variable)
	t.TagLimitVariable = &limitVar
	return t
}

// perStatBase builds a PerStat tag with a flat addend (PoB's @base@). mod.PerStatTag has
// the field but no setter.
func perStatBase(divide float64, base float64, stats ...string) *mod.PerStatTag {
	t := mod.PerStat(divide, stats...)
	t.Base = base
	return t
}
`
