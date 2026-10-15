package calculator

import (
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

type partyBuffExport struct {
	effectMult       float64
	isMark           bool
	modList          []mod.Mod
	warcryPowerBonus map[mod.Mod]float64
}

//pob:block:src/Modules/CalcPerform.lua:2380-2384
func warcryPowerBonuses(mods []mod.Mod, warcryPower float64) map[mod.Mod]float64 {
	out := map[mod.Mod]float64{}
	for _, m := range mods {
		if tags := m.Tags(); len(tags) > 0 {
			if ge, ok := tags[0].(*mod.GlobalEffectTag); ok && len(ge.GlobalEffectList) > 0 && ge.GlobalEffectList[0] == "Warcry" && ge.DivSet {
				out[m] = warcryPowerBonus(m, warcryPower)
			}
		}
	}
	return out
}

//pob:src/Modules/CalcPerform.lua:2207
type partyBuffExports struct {
	aura, curse, warcry, link map[string]*partyBuffExport
	otherEffects              map[string]*partyBuffExport
	enemyMods                 map[string][]mod.Mod
	enemyConditions           map[string]bool
	playerMods                map[string]bool
	curseLimit                float64
}

//pob:block:src/Modules/CalcPerform.lua:1312
func newPartyBuffExports(env *Environment) *partyBuffExports {
	if env.Build == nil || !env.Build.ExportParty() || env.Mode == OutputModeCalculator {
		return nil
	}
	return &partyBuffExports{
		aura: map[string]*partyBuffExport{}, curse: map[string]*partyBuffExport{}, warcry: map[string]*partyBuffExport{},
		link: map[string]*partyBuffExport{}, otherEffects: map[string]*partyBuffExport{}, enemyMods: map[string][]mod.Mod{},
		enemyConditions: map[string]bool{}, playerMods: map[string]bool{},
	}
}

func joinModLists(lists ...*moddb.ModList) []mod.Mod {
	var out []mod.Mod
	for _, l := range lists {
		if l != nil {
			out = append(out, l.Mods()...)
		}
	}
	return out
}

//pob:block:src/Modules/CalcPerform.lua:2499-2502
func (e *partyBuffExports) setAura(name string, mult float64, mods []mod.Mod) {
	if old := e.aura[name]; old != nil {
		e.aura[name+"_Debuff"] = old
	}
	e.aura[name] = &partyBuffExport{effectMult: mult, modList: mods}
}

//pob:block:src/Modules/CalcPerform.lua:2512
func (e *partyBuffExports) setAuraDebuff(name string, mult float64, mods []mod.Mod) {
	if e.aura[name] != nil {
		name += "_Debuff"
	}
	e.aura[name] = &partyBuffExport{effectMult: mult, modList: mods}
}

//pob:block:src/Modules/CalcPerform.lua:3812-3930
func (e *partyBuffExports) collectEnemyAndPlayer(env *Environment, output map[string]float64) {
	enemyDB := env.EnemyModDB
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
			e.playerMods[damageType+"Resist="+luaNumber(r)] = true
		}
	}
	if v, ok := output["EnduranceChargesMax"]; ok && env.ModDB.Flag(nil, "PartyMemberMaximumEnduranceChargesEqualToYours") {
		e.playerMods["PartyMemberMaximumEnduranceChargesEqualToYours"] = true
		e.playerMods["EnduranceChargesMax="+luaNumber(v)] = true
	}
	e.playerMods["MovementSpeedMod|percent|max="+luaNumber(output["MovementSpeedMod"]*100)] = true
	if extra := e.aura["extraAura"]; extra != nil {
		for _, m := range extra.modList {
			if m.Name() == "YourFortifyEqualToParent" {
				e.playerMods["PartyMemberFortifyEqualToYours"] = true
				e.playerMods["FortificationStacks="+luaNumber(output["FortificationStacks"])] = true
			}
		}
	}

	// preStack Mine auras
	for _, auraName := range sortedExportNames(e.aura) {
		if !strings.Contains(auraName, "Mine") || strings.Contains(auraName, " Limit") {
			continue
		}
		stackCount := 0.0
		if stack := e.enemyMods["Multiplier:"+auraName+"Stack"]; len(stack) > 0 {
			stackCount = partyModValue(stack[0])
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

	for _, linkName := range sortedExportNames(e.link) {
		if linkName == "Flame Link" {
			e.playerMods["Life="+luaNumber(output["Life"])] = true
		}
		for _, m := range e.link[linkName].modList {
			switch m.Name() {
			case "MaximumBlockAttackChanceIsEqualToParent":
				e.playerMods["BlockChanceMax="+luaNumber(output["BlockChanceMax"])] = true
			case "BlockAttackChanceIsEqualToParent":
				e.playerMods["BlockChance="+luaNumber(output["BlockChance"])] = true
			case "MaximumLifeLeechIsEqualToPartyMember":
				e.playerMods["MaxLifeLeechRatePercent="+luaNumber(output["MaxLifeLeechRatePercent"])] = true
			case "TakenFromParentESBeforeYou":
				e.playerMods["EnergyShieldRecoveryCap="+luaNumber(output["EnergyShieldRecoveryCap"])] = true
			case "MainHandCritIsEqualToParent":
				if env.Player.OutputTable[OutTableMainHand] != nil {
					e.playerMods["MainHand.CritChance="+luaNumber(env.Player.OutputTable[OutTableMainHand]["CritChance"])] = true
				} else if env.Player.WeaponData1 != nil {
					e.playerMods["MainHand.CritChance="+luaNumber(env.Player.WeaponData1.CritChance)] = true
				}
			}
		}
	}
}

//pob:block:src/Modules/CalcPerform.lua:3826-3828
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
		if len(t.VarList) == 1 {
			return has("Condition:" + t.VarList[0])
		}
		return len(t.VarList) > 1 && has("Condition:"+t.VarList[0])
	case *mod.MultiplierTag:
		return len(t.VariableList) > 0 && has("Multiplier:"+t.VariableList[0])
	}
	return true
}

//pob:block:src/Modules/CalcPerform.lua:3877-3885
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

func partyModValue(m mod.Mod) float64 {
	if v := m.Value(); v != nil && v.Type() == mod.ModValueMultiTypeFloat {
		return v.Float()
	}
	return 0
}

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

//pob:src/Classes/PartyTab.lua:612-653
var partyExportNames = [][2]string{
	{"PlayerMods", "PartyMemberStats"}, {"Aura", "Aura"}, {"Curse", "Curse"}, {"Warcry", "Warcry Skills"},
	{"Link", "Link Skills"}, {"EnemyConditions", "EnemyConditions"}, {"EnemyMods", "EnemyMods"},
}

func (e *partyBuffExports) texts() map[string]string {
	out := map[string]string{}
	for _, n := range partyExportNames {
		if t := e.exportBuffs(n[0]); t != "" {
			out[n[1]] = t
		}
	}
	return out
}

//pob:src/Classes/PartyTab.lua:992-1043
func (e *partyBuffExports) exportBuffs(buffType string) string {
	var b strings.Builder
	if buffType == "Curse" {
		b.WriteString("--- Curse Limit ---\n" + luaNumber(e.curseLimit))
	}
	line := func(s string) {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(s)
	}
	mods := func(list []mod.Mod) {
		for _, m := range list {
			b.WriteString("\n" + formatSourceMod(m, nil))
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
			mods(e.enemyMods[k])
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
						b.WriteString("\n" + inner + "|" + luaNumber(buff.effectMult*100) + "|" + formatSourceMod(m, nil))
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
				b.WriteString("\n" + luaNumber(buff.effectMult*100))
			}
			mods(buff.modList)
			b.WriteString("\n---")
		}
	case "Curse", "Warcry", "Link":
		list := map[string]map[string]*partyBuffExport{"Curse": e.curse, "Warcry": e.warcry, "Link": e.link}[buffType]
		for _, name := range sortedExportNames(list) {
			buff := list[name]
			line(name)
			b.WriteString("\n" + luaNumber(buff.effectMult*100))
			if buffType == "Curse" {
				b.WriteString("\n" + strconv.FormatBool(buff.isMark))
			}
			for _, m := range buff.modList {
				var first map[string]string
				if bonus, ok := buff.warcryPowerBonus[m]; ok {
					first = map[string]string{"warcryPowerBonus": luaNumber(bonus)}
				}
				b.WriteString("\n" + formatSourceMod(m, first))
			}
			b.WriteString("\n---")
		}
	}
	return b.String()
}

//pob:src/Modules/ModTools.lua:235-237
func formatSourceMod(m mod.Mod, firstTagExtra map[string]string) string {
	return formatPartyValue(m.Value()) + "|" + string(m.GetSource()) + "|" + formatPartyModParams(m, firstTagExtra)
}

//pob:src/Modules/ModTools.lua:196-198
func formatPartyValue(v *mod.ModValueMulti) string {
	if v == nil {
		return "nil"
	}
	switch v.Type() {
	case mod.ModValueMultiTypeFlag:
		return strconv.FormatBool(v.ValueFlag)
	case mod.ModValueMultiTypeList:
		return mod.FormatValue(v.ValueList)
	}
	return luaNumber(v.ValueFloat)
}

//pob:src/Modules/ModTools.lua:227-229
func formatPartyModParams(m mod.Mod, firstTagExtra map[string]string) string {
	flags := map[string]mod.MFlag{}
	for k, v := range partyModFlagByName {
		flags[k] = v
	}
	return m.Name() + "|" + string(m.Type()) + "|" + formatPartyFlags(int64(m.Flags()), flags) + "|" +
		formatPartyFlags(int64(m.KeywordFlags()), mod.KeywordFlagsByName) + "|" + formatPartyTags(m.Tags(), firstTagExtra)
}

//pob:src/Modules/ModTools.lua:136-149
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

//pob:src/Modules/ModTools.lua:151-194
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
			set("div", t.DivSet, luaNumber(t.DivTag))
			set("limit", t.LimitSet, luaNumber(t.LimitTag))
			set("effectStackVar", t.EffectStackVarTag != "", t.EffectStackVarTag)
			set("effectStackLimit", t.EffectStackLimitTag != 0, luaNumber(t.EffectStackLimitTag))
			set("effectStackLimitVar", t.EffectStackLimitVarTag != "", t.EffectStackLimitVarTag)
			set("applyNotPlayer", t.ApplyNotPlayerTag, "true")
			set("applyMinions", t.ApplyMinionsTag, "true")
			set("allowTotemBuff", t.AllowTotemBuffTag, "true")
		case *mod.ConditionTag:
			list("var", t.VarList)
			set("neg", t.Negative, "true")
		case *mod.MultiplierTag:
			list("var", t.VariableList)
			set("base", t.TagBase != 0, luaNumber(t.TagBase))
			set("div", t.Division != 1 && t.Division != 0, luaNumber(t.Division))
			set("limit", t.TagLimit != nil, luaNumber(derefOr(t.TagLimit)))
			set("limitVar", t.TagLimitVariable != nil, derefOr(t.TagLimitVariable))
			set("limitTotal", t.TagLimitTotal, "true")
			set("actor", t.TagActor != "", t.TagActor)
		case *mod.PercentStatTag:
			set("stat", t.Stat != "", t.Stat)
			set("percent", t.PercentVar == "", luaNumber(t.Percent))
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
