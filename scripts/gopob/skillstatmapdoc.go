package main

import "strings"

var ssmHeader = strings.ReplaceAll(ssmHeaderT, "@", "`")

const ssmHeaderT = `package raw

import (
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

// SkillStatMaps is PoB's hand-written per-skill @statMap@ table (src/Data/Skills/*.lua),
// keyed by granted-effect id then stat key. calcs.mergeSkillInstanceMods
// (src/Modules/CalcActiveSkill.lua:62) looks a skill's own stat up here first; only when a
// skill declares nothing for that stat does data.skillStatMapMeta.__index
// (src/Modules/Data.lua:668) fall the lookup through to the global map, which go-pob ports
// separately as SkillStatMap. GetStatMapEntry (granted_effects.go) mirrors that order.
//
// The distinction matters because the same raw stat maps differently per skill: PoB tags
// @base_physical_damage_reduction_rating@ with GlobalEffect/Aura on Determination so it
// reaches the player ModDB as an aura, while the global map leaves it skill-local.
//
// An empty Mods list means one of TWO things, and this header used to claim only the first.
//
// %d entries are empty because PoB DECLARES the stat with no mods (%d of them suppress a real
// global entry). Those are deliberate: emitting the entry keeps the fallback suppressed the way
// PoB does, and dropping it would apply a global mapping PoB never uses.
//
// %d are empty because every mod in them was OMITTED below (%d of them suppress a real global
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
// before re-deriving it. PoB's @{ mod(...), div = 60 }@ attaches a scalar to ONE mod rather than
// to the entry (src/Modules/CalcActiveSkill.lua:64-73), and RejuvenationTotem's two groups divide
// by 60 and by 400 (src/Data/Skills/act_str.lua:9329-9332), so no single entry-level Div could
// carry them. Both halves were missing at once: oracle/skill_stat_map.lua ran a group through
// encodeMod, which dumped the scalar's NAME and not its value, and raw.StatMap had nowhere to put
// one. Both are fixed, and the dump now carries a @groupCount@ this emitter REQUIRES, so a stale
// dump refuses instead of quietly emptying the two entries again.
//
// Generated from the PINNED PoB (%s skills, %s statMap entries, %s mods) by
// oracle/skill_stat_map.lua + gopob's skill-stat-maps emitter. The pin's identity is in
// scripts/pob_pin.json; this header does not restate it, because the per-skill dump does
// not carry pobVersion and a hardcoded version is how this line came to claim "v2.18.1"
// long after the pin moved to 3.29. %d mods are emitted;
// %d are not representable and are omitted rather than mapped lossily:
%s
//
// SkillPart entries follow the precedent set by SkillBaseMods: every part is carried, and
// so is PoB's @skillPartList@ form. moddb evaluates the tag (ModStore.lua:529-549 ->
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
// Discipline entry gave @energy_shield_delay_-%%@ a GlobalEffect/Aura tag, but PoB passes
// that tag in mod()'s @flags@ argument slot (src/Data/Skills/act_int.lua:2621), so upstream
// the mod ends up with NO tags, never gets promoted out of the skill's local mod list, and
// never reaches the player. Emitting the aura would beat PoB's own numbers rather than match
// them, so the mod is omitted and the entry left empty.
var SkillStatMaps = map[string]map[string]*StatMap{
`
