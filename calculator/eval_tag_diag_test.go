package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestDiagEvalTags is the EVALUATED-TAG diag.
func TestDiagEvalTags(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.SkipNow()
	}
	buildName := os.Getenv("BUILD")
	if buildName == "" {
		t.Skip("BUILD unset")
	}
	names := strings.Split(os.Getenv("NAME"), ",")
	if os.Getenv("NAME") == "" {
		names = []string{"Life"}
	}
	for i := range names {
		names[i] = strings.TrimSpace(names[i])
	}

	file, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), buildName+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatal(err)
	}
	mode := OutputModeMain
	if os.Getenv("MODE") == "calcs" {
		mode = OutputModeCalcs
	}
	env := (&Calculator{PoB: build}).BuildOutput(mode)

	actorName := strings.ToLower(os.Getenv("ACTOR"))
	if actorName == "" {
		actorName = "player"
	}
	actor := diagActor(t, env, actorName)

	skill := actor.MainSkill
	cfgName := strings.ToLower(os.Getenv("CFG"))
	if cfgName == "" {
		cfgName = "skill"
	}
	cfg := diagCfg(t, skill, cfgName)

	levels := diagLevels(t, actor, os.Getenv("LIST"))
	context := levels[0].store

	fmt.Printf("\n=== EVALTAG %s / %s ===\n", buildName, strings.Join(names, ","))
	fmt.Printf("HDR   actor=%s list=%s cfg=%s mode=%v type=%s\n",
		actorName, diagListName(os.Getenv("LIST")), cfgName, mode, orNone(os.Getenv("TYPE")))
	fmt.Printf("CFG   %s\n", describeCfg(cfg))
	if skill != nil {
		fmt.Printf("SKILL %s  part=%d slot=%s\n", diagSkillName(skill), skill.SkillPart, skill.SlotName)
	}

	typeFilter := mod.Type(strings.ToUpper(os.Getenv("TYPE")))

	diagTotals := map[mod.Type]float64{}
	seenTypes := map[mod.Type]bool{}

	for li, lvl := range levels {
		var matching []mod.Mod
		for _, name := range names {
			for _, m := range lvl.mods {
				if m.Name() == name && (typeFilter == "" || m.Type() == typeFilter) {
					matching = append(matching, m)
				}
			}
		}
		fmt.Printf("LVL   %d %-8s %d matching mod(s)\n", li, lvl.label, len(matching))
		for _, m := range matching {
			seenTypes[m.Type()] = true
			diagTotals[m.Type()] += diagReportMod(context, m, cfg, li, lvl.label)
		}
	}

	types := make([]mod.Type, 0, len(seenTypes))
	for ty := range seenTypes {
		types = append(types, ty)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })

	for _, ty := range types {
		engine, ok := diagEngineAggregate(context, ty, cfg, names)
		if !ok {
			fmt.Printf("SUM   %-9s diag=%12.4f engine=<not aggregated>\n", ty, diagTotals[ty])
			continue
		}
		mark := "MATCH"
		diagV := diagTotals[ty]
		if ty == mod.TypeMore {
			diagV = 1
			for _, lvl := range levels {
				for _, name := range names {
					for _, m := range lvl.mods {
						if m.Name() != name || m.Type() != mod.TypeMore {
							continue
						}
						if tr := moddb.TraceEvalMod(context, m, cfg); !tr.Excluded() &&
							tr.Final.Type() == mod.ModValueMultiTypeFloat && diagAdmitted(m, cfg) {
							diagV *= 1 + tr.Final.Float()/100
						}
					}
				}
			}
		}
		if diff := diagV - engine; diff > 1e-9 || diff < -1e-9 {
			mark = "DIFF <- diag drifted from the engine, or a globalLimit clamped the sum"
		}
		fmt.Printf("SUM   %-9s diag=%12.4f engine=%12.4f %s\n", ty, diagV, engine, mark)
	}
}

func diagReportMod(context moddb.ModStoreFuncs, m mod.Mod, cfg *moddb.ListCfg, level int, label string) float64 {
	head := fmt.Sprintf("L%d %-6s %-9s %-10s %-44s flags=%#x kw=%#x",
		level, label, m.Type(), fmtModValue(m.Value()),
		normalizeSource(string(m.GetSource())), int(m.Flags()), int(m.KeywordFlags()))

	if flagsOK, kwOK, srcOK := moddb.CfgAdmits(m, cfg); !flagsOK || !kwOK || !srcOK {
		var why []string
		if !flagsOK {
			why = append(why, fmt.Sprintf("flags(cfg %#x does not cover mod %#x)", int(cfgFlags(cfg)), int(m.Flags())))
		}
		if !kwOK {
			why = append(why, fmt.Sprintf("keywordFlags(cfg %#x vs mod %#x)", int(cfgKeywordFlags(cfg)), int(m.KeywordFlags())))
		}
		if !srcOK {
			why = append(why, fmt.Sprintf("source(cfg %q vs mod %q)", string(*cfg.Source), string(m.GetSource())))
		}
		fmt.Printf("MOD   [EXCL-CFG] %s\n", head)
		fmt.Printf("  WHY %s\n", strings.Join(why, " "))
		return 0
	}

	trace := moddb.TraceEvalMod(context, m, cfg)

	verdict := "[INCL]    "
	if trace.Excluded() {
		verdict = fmt.Sprintf("[EXCL-TAG%d]", len(trace.Steps))
	}
	fmt.Printf("MOD   %s %s -> %s\n", verdict, head, fmtModValue(trace.Final))

	for i, step := range trace.Steps {
		gate := "pass"
		if step.After == nil {
			gate = "GATED OUT"
		}
		fmt.Printf("  TAG %d %-20s %-9s %s -> %s | %s\n",
			i+1, step.Tag.Type(), gate,
			fmtModValue(step.Before), fmtModValue(step.After),
			describeTag(context, step.Tag, cfg))
	}

	if trace.Excluded() || trace.Final.Type() != mod.ModValueMultiTypeFloat {
		return 0
	}
	return trace.Final.Float()
}

func describeTag(context moddb.ModStoreFuncs, raw mod.Tag, cfg *moddb.ListCfg) string {
	switch tag := raw.(type) {
	case *mod.MultiplierTag:
		target, note := diagTarget(context, tag.TagActor)
		if target == nil {
			return fmt.Sprintf("vars=%v actor=%q %s", tag.VariableList, tag.TagActor, note)
		}
		parts := make([]string, 0, len(tag.VariableList))
		base := 0.0
		for _, v := range tag.VariableList {
			got := target.GetMultiplier(v, cfg, false)
			base += got
			parts = append(parts, fmt.Sprintf("%s=%g", v, got))
		}
		out := fmt.Sprintf("mult[%s] base=%g div=%g%s", strings.Join(parts, " "), base, tagDivOf(tag.Division), note)
		if tag.TagBase != 0 {
			out += fmt.Sprintf(" tagBase=%g", tag.TagBase)
		}
		if tag.TagLimit != nil {
			out += fmt.Sprintf(" limit=%g", *tag.TagLimit)
		}
		if tag.TagLimitVariable != nil {
			lt, ln := diagTarget(context, tag.TagLimitActor)
			lim := 0.0
			if lt != nil {
				lim = lt.GetMultiplier(*tag.TagLimitVariable, cfg, false)
			}
			out += fmt.Sprintf(" limitVar=%s=%g%s", *tag.TagLimitVariable, lim, ln)
		}
		if tag.TagLimitTotal {
			out += " limitTotal"
		}
		return out

	case *mod.MultiplierThresholdTag:
		target, note := diagTarget(context, tag.TagActor)
		if target == nil {
			return fmt.Sprintf("var=%s actor=%q %s", tag.Variable, tag.TagActor, note)
		}
		got := target.GetMultiplier(tag.Variable, cfg, false)
		thr := 0.0
		thrDesc := "?"
		if tag.TagThreshold != nil {
			thr = *tag.TagThreshold
			thrDesc = fmt.Sprintf("%g", thr)
		} else if tag.ThresholdVariable != nil {
			thr = target.GetMultiplier(*tag.ThresholdVariable, cfg, false)
			thrDesc = fmt.Sprintf("%s=%g", *tag.ThresholdVariable, thr)
		}
		return fmt.Sprintf("%s=%g threshold=%s upper=%v%s", tag.Variable, got, thrDesc, tag.TagUpper, note)

	case *mod.PerStatTag:
		target, note := diagTarget(context, tag.TagActor)
		if target == nil {
			target, note = context, fmt.Sprintf(" [actor %q absent, PoB falls back to self]", tag.TagActor)
		}
		parts := make([]string, 0, len(tag.StatList))
		base := 0.0
		for _, st := range tag.StatList {
			got := moddb.GetStat(target, st, cfg)
			base += got
			parts = append(parts, fmt.Sprintf("%s=%g", st, got))
		}
		return fmt.Sprintf("stats[%s] base=%g div=%g base+=%g%s",
			strings.Join(parts, " "), base, tagDivOf(tag.Divide), tag.Base, note)

	case *mod.ConditionTag:
		parts := make([]string, 0, len(tag.VarList))
		for _, v := range tag.VarList {
			store, _ := context.GetCondition(v, cfg, false)
			skillCond := cfg != nil && cfg.SkillCond != nil && cfg.SkillCond[v]
			parts = append(parts, fmt.Sprintf("%s: store=%v skillCond=%v", v, store, skillCond))
		}
		return fmt.Sprintf("cond[%s] neg=%v", strings.Join(parts, " | "), tag.Negative)

	case *mod.ActorConditionTag:
		name := ""
		if tag.Actor != nil {
			name = *tag.Actor
		}
		target, note := diagTarget(context, name)
		if target == nil {
			return fmt.Sprintf("actor=%q vars=%v neg=%v %s", name, tag.VariableList, tag.Negative, note)
		}
		parts := make([]string, 0, len(tag.VariableList))
		for _, v := range tag.VariableList {
			c, _ := target.GetCondition(v, cfg, false)
			parts = append(parts, fmt.Sprintf("%s=%v", v, c))
		}
		return fmt.Sprintf("actor=%q cond[%s] neg=%v%s", name, strings.Join(parts, " "), tag.Negative, note)

	case *mod.SkillNameTag:
		return fmt.Sprintf("want=%v neg=%v cfg.SkillName=%q", tag.SkillNameList, tag.Negative, cfgSkillName(cfg))

	case *mod.SkillIDTag:
		got := "<nil>"
		if cfg != nil && cfg.SkillGrantedEffect != nil {
			if ge, ok := cfg.SkillGrantedEffect.(interface{ ID() string }); ok {
				got = ge.ID()
			} else {
				got = "<no ID()>"
			}
		}
		return fmt.Sprintf("want=%q cfg.SkillGrantedEffect.ID=%q", tag.ID(), got)

	case *mod.SkillPartTag:
		want := fmt.Sprintf("%v", tag.Part)
		if tag.PartList != nil {
			want = fmt.Sprintf("%v", tag.PartList)
		}
		return fmt.Sprintf("want=%s neg=%v cfg.SkillPart=%d", want, tag.Negative, cfgSkillPart(cfg))

	case *mod.SkillTypeTag:
		have := make([]string, 0, len(tag.SkillTypeList))
		for _, st := range tag.SkillTypeList {
			on := cfg != nil && cfg.SkillTypes != nil && cfg.SkillTypes[data.SkillType(st)]
			have = append(have, fmt.Sprintf("%v=%v", st, on))
		}
		return fmt.Sprintf("types[%s] neg=%v", strings.Join(have, " "), tag.Negative)

	case *mod.SlotNameTag:
		return fmt.Sprintf("want=%v cfg.SlotName=%q", tag.SlotNameList, cfgSlotName(cfg))

	case *mod.SocketedInTag:
		gem := "<nil>"
		if cfg != nil && cfg.SkillGem != nil {
			gem = "<set>"
		}
		return fmt.Sprintf("slot=%q cfg.SlotName=%q keyword=%q cfg.SkillGem=%s",
			tag.SlotName, cfgSlotName(cfg), tag.TagKeyword, gem)

	case *mod.ModFlagOrTag:
		return fmt.Sprintf("want any of %#x cfg.Flags=%#x", int(tag.Flag), int(cfgFlags(cfg)))

	case *mod.StatThresholdTag:
		got := moddb.GetStat(context, tag.Stat, cfg)
		thr := tag.Threshold
		thrDesc := fmt.Sprintf("%g", thr)
		if tag.TagThresholdStat != "" {
			thr = moddb.GetStat(context, tag.TagThresholdStat, cfg)
			thrDesc = fmt.Sprintf("%s=%g", tag.TagThresholdStat, thr)
		}
		return fmt.Sprintf("%s=%g threshold=%s upper=%v", tag.Stat, got, thrDesc, tag.TagUpper)

	case *mod.LimitTag:
		lim := tag.TagLimit
		desc := fmt.Sprintf("%g", lim)
		if tag.LimitVar != "" {
			lim = context.GetMultiplier(tag.LimitVar, cfg, false)
			desc = fmt.Sprintf("%s=%g", tag.LimitVar, lim)
		}
		return "cap=" + desc

	case *mod.PercentStatTag:
		return fmt.Sprintf("%s=%g percent=%g", tag.Stat, moddb.GetStat(context, tag.Stat, cfg), tag.Percent)

	case *mod.DistanceRampTag:
		return fmt.Sprintf("cfg.SkillDist=%g steps=%d", cfgSkillDist(cfg), len(tag.Ramp))

	case *mod.MeleeProximityTag:
		return fmt.Sprintf("cfg.SkillDist=%g ramp=%v", cfgSkillDist(cfg), tag.Ramp)
	}

	return "(not evaluated by EvalMod)"
}

func diagTarget(context moddb.ModStoreFuncs, name string) (moddb.ModStoreFuncs, string) {
	if name == "" {
		return context, ""
	}
	db := moddb.RelatedModDB(context, name)
	if db == nil {
		return nil, fmt.Sprintf("[actor %q NOT LINKED -> mod dropped]", name)
	}
	return db, fmt.Sprintf(" [via actor %q]", name)
}

type diagLevel struct {
	label string
	store moddb.ModStoreFuncs
	mods  []mod.Mod
}

func diagLevels(t *testing.T, actor *Actor, list string) []diagLevel {
	t.Helper()

	var head moddb.ModStoreFuncs
	if strings.ToLower(list) == "skill" {
		skill := actor.MainSkill
		if skill == nil || skill.SkillModList == nil {
			t.Fatal("LIST=skill but the actor has no main skill mod list")
		}
		head = skill.SkillModList
	} else {
		if actor.ModDB == nil {
			t.Fatal("actor has no ModDB")
		}
		head = actor.ModDB
	}

	labels := map[moddb.ModStoreFuncs]string{actor.ModDB: "actorDB"}
	if skill := actor.MainSkill; skill != nil {
		if skill.BaseSkillModList != nil {
			labels[skill.BaseSkillModList] = "base"
		}
		if skill.SkillModList != nil {
			labels[skill.SkillModList] = "skill"
		}
	}
	if actor.Enemy != nil && actor.Enemy.ModDB != nil {
		labels[actor.Enemy.ModDB] = "enemyDB"
	}
	if actor.Parent != nil && actor.Parent.ModDB != nil {
		labels[actor.Parent.ModDB] = "parentDB"
	}

	var levels []diagLevel
	seen := map[moddb.ModStoreFuncs]bool{}
	for s := head; s != nil && !seen[s]; s = diagParentOf(s) {
		seen[s] = true
		label := labels[s]
		if label == "" {
			label = "store"
		}
		levels = append(levels, diagLevel{label, s, diagModsOf(s)})
	}
	return levels
}

func diagParentOf(s moddb.ModStoreFuncs) moddb.ModStoreFuncs {
	switch v := s.(type) {
	case *moddb.ModDB:
		return v.Parent
	case *moddb.ModList:
		return v.Parent
	}
	return nil
}

func diagModsOf(s moddb.ModStoreFuncs) []mod.Mod {
	switch v := s.(type) {
	case *moddb.ModDB:
		keys := make([]string, 0, len(v.Mods))
		for k := range v.Mods {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]mod.Mod, 0, len(v.Mods))
		for _, k := range keys {
			out = append(out, v.Mods[k]...)
		}
		return out
	case *moddb.ModList:
		return v.Mods()
	}
	return nil
}

func diagActor(t *testing.T, env *Environment, name string) *Actor {
	t.Helper()
	switch name {
	case "player":
		return env.Player
	case "enemy":
		if env.Enemy == nil {
			t.Fatal("no enemy actor")
		}
		return env.Enemy
	case "minion":
		if env.Minion == nil {
			t.Fatal("this build's main skill grants no minion")
		}
		return env.Minion.Actor
	}
	t.Fatalf("ACTOR must be player, minion or enemy (got %q)", name)
	return nil
}

func diagCfg(t *testing.T, skill *ActiveSkill, name string) *moddb.ListCfg {
	t.Helper()
	if name == "nil" || name == "none" {
		return nil
	}
	if skill == nil {
		t.Fatalf("CFG=%s but the actor has no main skill; use CFG=nil", name)
	}
	cfgs := map[string]*moddb.ListCfg{
		"skill":    skill.SkillCfg,
		"main":     skill.Weapon1Cfg,
		"off":      skill.Weapon2Cfg,
		"bleed":    skill.BleedCfg,
		"ohbleed":  skill.OHBleedCfg,
		"poison":   skill.PoisonCfg,
		"ohpoison": skill.OHPoisonCfg,
		"ignite":   skill.IgniteCfg,
		"ohignite": skill.OHIgniteCfg,
		"decay":    skill.DecayCfg,
	}
	live := make([]string, 0, len(cfgs))
	for k, c := range cfgs {
		if c != nil {
			live = append(live, k)
		}
	}
	sort.Strings(live)

	cfg, ok := cfgs[name]
	if !ok {
		keys := make([]string, 0, len(cfgs))
		for k := range cfgs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		t.Fatalf("CFG must be nil or one of %v (got %q); this build has %v", keys, name, live)
	}
	if cfg == nil {
		if fallback := cfgs["skill"]; fallback != nil && name != "skill" {
			fmt.Printf("CFG   ** CFG=%s is nil on this build (a spell build has no weapon cfg).\n"+
				"CFG   ** FELL BACK TO CFG=skill. This build has: %v\n", name, live)
			return fallback
		}
		t.Fatalf("CFG=%s is nil on this build's main skill and there is no skill cfg to fall back to; this build has %v", name, live)
	}
	return cfg
}

func diagEngineAggregate(store moddb.ModStoreFuncs, ty mod.Type, cfg *moddb.ListCfg, names []string) (float64, bool) {
	switch ty {
	case mod.TypeBase, mod.TypeIncrease:
		return store.Sum(ty, cfg, names...), true
	case mod.TypeMore:
		return store.More(cfg, names...), true
	case mod.TypeFlag:
		if store.Flag(cfg, names...) {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func diagAdmitted(m mod.Mod, cfg *moddb.ListCfg) bool {
	a, b, c := moddb.CfgAdmits(m, cfg)
	return a && b && c
}

func describeCfg(cfg *moddb.ListCfg) string {
	if cfg == nil {
		return "<nil>"
	}
	source := "<nil>"
	if cfg.Source != nil {
		source = string(*cfg.Source)
	}
	return fmt.Sprintf("flags=%#x keywordFlags=%#x source=%s slot=%q skillName=%q part=%d dist=%g skillCond=%d skillStats=%d",
		int(cfgFlags(cfg)), int(cfg.KeywordFlags.Get()), source, cfg.SlotName, cfg.SkillName,
		cfg.SkillPart, cfg.SkillDist, len(cfg.SkillCond), len(cfg.SkillStats))
}

func cfgFlags(cfg *moddb.ListCfg) mod.MFlag {
	if cfg == nil {
		return 0
	}
	return cfg.Flags.Get()
}

func cfgKeywordFlags(cfg *moddb.ListCfg) mod.KeywordFlag {
	if cfg == nil {
		return 0
	}
	return cfg.KeywordFlags.Get()
}

func cfgSlotName(cfg *moddb.ListCfg) string {
	if cfg == nil {
		return ""
	}
	return cfg.SlotName
}

func cfgSkillName(cfg *moddb.ListCfg) string {
	if cfg == nil {
		return ""
	}
	return cfg.SkillName
}

func cfgSkillPart(cfg *moddb.ListCfg) int {
	if cfg == nil {
		return 0
	}
	return cfg.SkillPart
}

func cfgSkillDist(cfg *moddb.ListCfg) float64 {
	if cfg == nil {
		return 0
	}
	return cfg.SkillDist
}

func fmtModValue(v *mod.ModValueMulti) string {
	if v == nil {
		return "(filtered)"
	}
	switch v.Type() {
	case mod.ModValueMultiTypeFloat:
		return fmt.Sprintf("%10.4f", v.Float())
	case mod.ModValueMultiTypeFlag:
		return fmt.Sprintf("%10v", v.Flag())
	}
	return fmt.Sprintf("%10s", "LIST")
}

func tagDivOf(div float64) float64 {
	if div == 0 {
		return 1
	}
	return div
}

func diagListName(list string) string {
	if strings.ToLower(list) == "skill" {
		return "skill"
	}
	return "db"
}

func diagSkillName(skill *ActiveSkill) string {
	if skill.ActiveEffect != nil && skill.ActiveEffect.GrantedEffect != nil {
		return skill.ActiveEffect.GrantedEffect.ID()
	}
	return "<unnamed>"
}

func orNone(s string) string {
	if s == "" {
		return "<all>"
	}
	return s
}
