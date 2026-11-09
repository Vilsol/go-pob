package calculator2

import (
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob2"
)

type partyBuffExport struct {
	effectMult       float64
	isMark           bool
	modList          []mod.Mod
	warcryPowerBonus map[mod.Mod]float64
}

// partyBuffExports is buffExports; Link is absent, as no PoE2 skill carries a Link buff.
//
//pob2:block:src/Modules/CalcPerform.lua:2048
type partyBuffExports struct {
	aura, curse, warcry map[string]*partyBuffExport
	otherEffects        map[string]*partyBuffExport
	enemyMods           map[string][]mod.Mod
	enemyConditions     map[string]bool
	playerMods          map[string]bool
	curseLimit          float64
}

// newPartyBuffExports is nil unless the Import tab's "Export support" is on; every buffExports write
// below is skipped with it, as the export reads none of them then.
//
//pob2:block:src/Modules/CalcPerform.lua:1221
func newPartyBuffExports(env *Environment) *partyBuffExports {
	if env.Build == nil || !env.Build.Import.ExportParty || env.Mode == "CALCULATOR" {
		return nil
	}
	return &partyBuffExports{
		aura: map[string]*partyBuffExport{}, curse: map[string]*partyBuffExport{}, warcry: map[string]*partyBuffExport{},
		otherEffects: map[string]*partyBuffExport{}, enemyMods: map[string][]mod.Mod{},
		enemyConditions: map[string]bool{}, playerMods: map[string]bool{},
	}
}

func joinMods(lists ...[]mod.Mod) []mod.Mod {
	var out []mod.Mod
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

//pob2:block:src/Modules/CalcPerform.lua:2343-2346
func (e *partyBuffExports) setAura(name string, mult float64, mods []mod.Mod) {
	if old := e.aura[name]; old != nil {
		e.aura[name+"_Debuff"] = old
	}
	e.aura[name] = &partyBuffExport{effectMult: mult, modList: mods}
}

//pob2:block:src/Modules/CalcPerform.lua:2414
func (e *partyBuffExports) setAuraDebuff(name string, mult float64, mods []mod.Mod) {
	if e.aura[name] != nil {
		name += "_Debuff"
	}
	e.aura[name] = &partyBuffExport{effectMult: mult, modList: mods}
}

// warcryPowerBonuses is the warcryPowerBonus each Warcry mod's first tag carries.
//
//pob2:block:src/Modules/CalcPerform.lua:2225-2229
func warcryPowerBonuses(mods []mod.Mod, bonus []float64) map[mod.Mod]float64 {
	out := map[mod.Mod]float64{}
	for i, m := range mods {
		if tags := m.Tags(); len(tags) > 0 {
			if ge, ok := tags[0].(*mod.GlobalEffectTag); ok && len(ge.GlobalEffectList) > 0 && ge.GlobalEffectList[0] == "Warcry" && ge.DivSet {
				out[m] = bonus[i]
			}
		}
	}
	return out
}

// collectEnemyAndPlayer exports the enemy's conditions and mods and the stats party members copy. The
// Link loop (3625-3648) is absent with Link.
//
//pob2:block:src/Modules/CalcPerform.lua:3530-3623
func (e *partyBuffExports) collectEnemyAndPlayer(env *Environment, output map[string]float64) {
	enemyDB := env.Enemy.ModDB
	for _, k := range sortedExportNames(enemyDB.Mods) {
		mods := enemyDB.Mods[k]
		if strings.Contains(k, "Condition") && !strings.Contains(k, "Party") {
			e.enemyConditions[k] = true
			continue
		}
		if !(strings.Contains(k, "Resist") && !strings.Contains(k, "Totem") && !strings.Contains(k, "Max") ||
			strings.Contains(k, "Damage") || strings.Contains(k, "ActionSpeed") || strings.Contains(k, "SelfCrit") ||
			strings.Contains(k, "Multiplier") && !strings.Contains(k, "Max") && !strings.Contains(k, "Impale")) {
			continue
		}
		for _, v := range mods {
			src := string(v.GetSource())
			if v.Value() != nil && v.Value().Type() == mod.ModValueMultiTypeFloat && v.Value().Float() == 0 || src == "EnemyConfig" || src == "Base" ||
				strings.Contains(src, "Delirious") || strings.HasPrefix(src, "Party") {
				continue
			}
			skip := false
			for _, tag := range v.Tags() {
				if ge, ok := tag.(*mod.GlobalEffectTag); ok && len(ge.GlobalEffectList) > 0 &&
					(ge.GlobalEffectList[0] == "Curse" || ge.GlobalEffectList[0] == "AuraDebuff") {
					skip = true
					break
				}
			}
			if skip || !enemyTagHolds(enemyDB, v) {
				continue
			}
			e.enemyMods[k] = append(e.enemyMods[k], v)
		}
	}

	for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
		if r, ok := output[damageType+"Resist"]; ok && env.ModDB.Flag(nil, "Enemy"+damageType+"ResistEqualToYours") {
			e.playerMods["Enemy"+damageType+"ResistEqualToYours"] = true
			e.playerMods[damageType+"Resist="+luaToString(r)] = true
		}
	}
	if v, ok := output["EnduranceChargesMax"]; ok && env.ModDB.Flag(nil, "PartyMemberMaximumEnduranceChargesEqualToYours") {
		e.playerMods["PartyMemberMaximumEnduranceChargesEqualToYours"] = true
		e.playerMods["EnduranceChargesMax="+luaToString(v)] = true
	}
	e.playerMods["MovementSpeedMod|percent|max="+luaToString(output["MovementSpeedMod"]*100)] = true
	for _, m := range e.aura["extraAura"].modList {
		if m.Name() == "YourFortifyEqualToParent" {
			e.playerMods["PartyMemberFortifyEqualToYours"] = true
			e.playerMods["FortificationStacks="+luaToString(output["FortificationStacks"])] = true
		}
	}

	// preStack Mine auras
	for _, auraName := range sortedExportNames(e.aura) {
		if !strings.Contains(auraName, "Mine") || strings.Contains(auraName, " Limit") {
			continue
		}
		stackCount := 0.0
		// a MultiStat entry has no value
		if stack := e.enemyMods["Multiplier:"+auraName+"Stack"]; len(stack) == 1 {
			if v := stack[0].Value(); v != nil && v.Type() == mod.ModValueMultiTypeFloat {
				stackCount = v.Float()
			}
		}
		delete(e.enemyMods, "Multiplier:"+auraName+"Stack")
		for _, name := range []string{auraName, auraName + " Limit"} {
			aura := e.aura[name]
			if aura == nil {
				continue
			}
			if stackCount == 0 {
				delete(e.aura, name)
				continue
			}
			e.aura[name] = &partyBuffExport{effectMult: aura.effectMult, isMark: aura.isMark, modList: stackMineMods(aura.modList, stackCount)}
		}
	}
}

//pob2:block:src/Modules/CalcPerform.lua:3544-3546
func enemyTagHolds(enemyDB *moddb.ModDB, v mod.Mod) bool {
	tags := v.Tags()
	if len(tags) == 0 {
		return true
	}
	has := func(name string) bool {
		mods := enemyDB.Mods[name]
		return len(mods) > 0 && partyModTruthy(mods[0])
	}
	switch t := tags[0].(type) {
	case *mod.ConditionTag:
		return len(t.VarList) > 0 && has("Condition:"+t.VarList[0])
	case *mod.MultiplierTag:
		return len(t.VariableList) > 0 && has("Multiplier:"+t.VariableList[0])
	}
	return true
}

//pob2:block:src/Modules/CalcPerform.lua:3595-3603
func stackMineMods(mods []mod.Mod, stackCount float64) []mod.Mod {
	out := make([]mod.Mod, len(mods))
	for i, m := range mods {
		out[i] = m
		for _, tag := range m.Tags() {
			if ge, ok := tag.(*mod.GlobalEffectTag); ok && ge.EffectStackVarTag == "ActiveMineCount" {
				c := m.Clone()
				for j, ct := range c.Tags() {
					if cge, ok := ct.(*mod.GlobalEffectTag); ok && cge.EffectStackVarTag == "ActiveMineCount" {
						cp := *cge
						cp.EffectStackVarTag = ""
						c.Tags()[j] = &cp
					}
				}
				if val := c.Value(); val != nil && val.Type() == mod.ModValueMultiTypeFloat {
					val.SetFloat(val.Float() * stackCount)
				}
				out[i] = c
				break
			}
		}
	}
	return out
}

// partyModTruthy is a Lua truth test of a mod's value: only false and nil fail.
func partyModTruthy(m mod.Mod) bool {
	v := m.Value()
	if v == nil {
		return false
	}
	if v.Type() == mod.ModValueMultiTypeFlag {
		return v.ValueFlag
	}
	return true
}

func sortedExportNames[T any](m map[string]T) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	slices.Sort(names)
	return names
}

// partyExportNames is each export's buffType and the name Save gives its ExportedBuffs element.
//
//pob2:src/Classes/PartyTab.lua:612-653
var partyExportNames = [][2]string{
	{"PlayerMods", "PartyMemberStats"}, {"Aura", "Aura"}, {"Curse", "Curse"}, {"Warcry", "Warcry Skills"},
	{"Link", "Link Skills"}, {"EnemyConditions", "EnemyConditions"}, {"EnemyMods", "EnemyMods"},
}

// buffers is the ExportedBuffs PartyTab:Save writes, in its order, leaving out an empty one.
func (e *partyBuffExports) buffers() []pob2.PartyBuffs {
	var out []pob2.PartyBuffs
	for _, n := range partyExportNames {
		if t := e.exportBuffs(n[0]); t != "" {
			name := n[1]
			out = append(out, pob2.PartyBuffs{Name: &name, Text: t})
		}
	}
	return out
}

//pob2:src/Classes/PartyTab.lua:992-1043
func (e *partyBuffExports) exportBuffs(buffType string) string {
	var b strings.Builder
	if buffType == "Curse" {
		b.WriteString("--- Curse Limit ---\n" + luaToString(e.curseLimit))
	}
	line := func(s string) {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(s)
	}
	mods := func(list []mod.Mod, bonus map[mod.Mod]float64) {
		for _, m := range list {
			var first map[string]string
			if v, ok := bonus[m]; ok {
				first = map[string]string{"warcryPowerBonus": luaToString(v)}
			}
			b.WriteString("\n" + formatSourceMod(m, first))
		}
	}
	switch buffType {
	case "PlayerMods":
		for _, k := range sortedExportNames(e.playerMods) {
			line(k)
		}
	case "EnemyConditions":
		for _, k := range sortedExportNames(e.enemyConditions) {
			line(k)
		}
	case "EnemyMods":
		for _, k := range sortedExportNames(e.enemyMods) {
			line(k)
			mods(e.enemyMods[k], nil)
		}
	case "Aura":
		names := sortedExportNames(e.aura)
		if len(e.otherEffects) > 0 {
			names = append(names, "otherEffects")
			slices.Sort(names)
		}
		for _, name := range names {
			if name == "otherEffects" {
				line(name)
				for _, inner := range sortedExportNames(e.otherEffects) {
					buff := e.otherEffects[inner]
					for _, m := range buff.modList {
						b.WriteString("\n" + inner + "|" + luaToString(buff.effectMult*100) + "|" + formatSourceMod(m, nil))
					}
				}
				b.WriteString("\n---")
				continue
			}
			buff := e.aura[name]
			if name == "extraAura" && len(buff.modList) == 0 {
				continue
			}
			line(name)
			if name != "extraAura" {
				b.WriteString("\n" + luaToString(buff.effectMult*100))
			}
			mods(buff.modList, nil)
			b.WriteString("\n---")
		}
	case "Curse", "Warcry":
		list := e.curse
		if buffType == "Warcry" {
			list = e.warcry
		}
		for _, name := range sortedExportNames(list) {
			buff := list[name]
			line(name)
			b.WriteString("\n" + luaToString(buff.effectMult*100))
			if buffType == "Curse" {
				b.WriteString("\n" + strconv.FormatBool(buff.isMark))
			}
			mods(buff.modList, buff.warcryPowerBonus)
			b.WriteString("\n---")
		}
	}
	return b.String()
}

//pob2:src/Modules/ModTools.lua:273-275
func formatSourceMod(m mod.Mod, firstTagExtra map[string]string) string {
	return formatPartyValue(m.Value()) + "|" + string(m.GetSource()) + "|" + formatPartyModParams(m, firstTagExtra)
}

//pob2:src/Modules/ModTools.lua:232-234
func formatPartyValue(v *mod.ModValueMulti) string {
	if v == nil {
		return "nil"
	}
	switch v.Type() {
	case mod.ModValueMultiTypeFlag:
		// a numeric FLAG keeps its number
		if v.ValueFloat != 0 {
			return luaToString(v.ValueFloat)
		}
		return strconv.FormatBool(v.ValueFlag)
	case mod.ModValueMultiTypeList:
		return mod.FormatValue(v.ValueList)
	}
	return luaToString(v.ValueFloat)
}

//pob2:src/Modules/ModTools.lua:265-267
func formatPartyModParams(m mod.Mod, firstTagExtra map[string]string) string {
	return m.Name() + "|" + string(m.Type()) + "|" + formatPartyFlags(int64(m.Flags()), modFlagByName) + "|" +
		formatPartyFlags(int64(m.KeywordFlags()), mod.KeywordFlagsByName) + "|" + formatPartyTags(m.Tags(), firstTagExtra)
}

//pob2:src/Modules/ModTools.lua:159-172
func formatPartyFlags[F ~int64 | ~int | ~uint64](flags int64, src map[string]F) string {
	var names []string
	for name, val := range src {
		if flags&int64(val) == int64(val) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	if len(names) == 0 {
		return "-"
	}
	return strings.Join(names, ",")
}

//pob2:src/Modules/ModTools.lua:174-217
func formatPartyTags(tags []mod.Tag, firstTagExtra map[string]string) string {
	if len(tags) == 0 {
		return "-"
	}
	out := make([]string, len(tags))
	for i, tag := range tags {
		params := map[string]string{}
		if i == 0 {
			for k, v := range firstTagExtra {
				params[k] = v
			}
		}
		list := func(k string, vs []string) {
			if len(vs) == 1 {
				params[k] = vs[0]
			} else if len(vs) > 1 {
				params[k+"List"] = "{" + strings.Join(vs, ",") + "}"
			}
		}
		set := func(k string, on bool, v string) {
			if on {
				params[k] = v
			}
		}
		switch t := tag.(type) {
		case *mod.GlobalEffectTag:
			if len(t.GlobalEffectList) > 0 {
				params["effectType"] = t.GlobalEffectList[0]
			}
			set("effectName", t.NameTag != "", t.NameTag)
			set("effectCond", t.EffectCondTag != "", t.EffectCondTag)
			set("effectEnemyCond", t.EffectEnemyCondTag != "", t.EffectEnemyCondTag)
			set("modCond", t.ModCondTag != "", t.ModCondTag)
			set("unscalable", t.UnscalableTag, "true")
			set("div", t.DivSet, luaToString(t.DivTag))
			set("limit", t.LimitSet, luaToString(t.LimitTag))
			set("effectStackVar", t.EffectStackVarTag != "", t.EffectStackVarTag)
			set("effectStackLimit", t.EffectStackLimitTag != 0, luaToString(t.EffectStackLimitTag))
			set("effectStackLimitVar", t.EffectStackLimitVarTag != "", t.EffectStackLimitVarTag)
			set("applyNotPlayer", t.ApplyNotPlayerTag, "true")
			set("applyMinions", t.ApplyMinionsTag, "true")
			set("allowTotemBuff", t.AllowTotemBuffTag, "true")
		case *mod.ConditionTag:
			list("var", t.VarList)
			set("neg", t.Negative, "true")
		case *mod.MultiplierTag:
			list("var", t.VariableList)
			set("base", t.TagBase != 0, luaToString(t.TagBase))
			set("div", t.Division != 1 && t.Division != 0, luaToString(t.Division))
			set("limit", t.TagLimit != nil, luaToString(derefOr(t.TagLimit)))
			set("limitVar", t.TagLimitVariable != nil, derefOr(t.TagLimitVariable))
			set("limitTotal", t.TagLimitTotal, "true")
			set("actor", t.TagActor != "", t.TagActor)
		case *mod.ActorConditionTag:
			if t.Actor != nil {
				set("actor", *t.Actor != "", *t.Actor)
			}
			list("var", t.VariableList)
			set("neg", t.Negative, "true")
		case *mod.PerStatTag:
			list("stat", t.StatList)
			set("div", t.Divide != 1 && t.Divide != 0, luaToString(t.Divide))
			set("divVar", t.TagDivVariable != "", t.TagDivVariable)
			set("base", t.Base != 0, luaToString(t.Base))
			set("limit", t.TagLimit != nil, luaToString(derefOr(t.TagLimit)))
			set("limitVar", t.TagLimitVariable != nil, derefOr(t.TagLimitVariable))
			set("limitTotal", t.TagLimitTotal, "true")
			set("actor", t.TagActor != "", t.TagActor)
			set("globalLimit", t.TagGlobalLimit != 0, luaToString(t.TagGlobalLimit))
			set("globalLimitKey", t.TagGlobalLimitKey != "", t.TagGlobalLimitKey)
		case *mod.LimitTag:
			set("limit", t.LimitVar == "", luaToString(t.TagLimit))
			set("limitVar", t.LimitVar != "", t.LimitVar)
			set("neg", t.Negative, "true")
		case *mod.PercentStatTag:
			set("stat", t.Stat != "", t.Stat)
			set("percent", t.PercentVar == "", luaToString(t.Percent))
			set("percentVar", t.PercentVar != "", t.PercentVar)
			set("actor", t.TagActor != "", t.TagActor)
			set("floor", t.TagFloor, "true")
		}
		names := sortedExportNames(params)
		parts := []string{"type=" + string(tag.Type())}
		for _, k := range names {
			parts = append(parts, k+"="+params[k])
		}
		out[i] = strings.Join(parts, "/")
	}
	return strings.Join(out, ",")
}

func derefOr[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}
