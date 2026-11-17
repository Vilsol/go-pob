package mcp

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

// overLimit is a Full DPS skill counted more times than it can have totems or mines out.
type overLimit struct {
	group int
	name  string
	kind  string
	count int
	limit int
}

// skillCount is the count Full DPS multiplies a skill's DPS by: the group's count, else its gem's.
//
//pob:block:src/Modules/Calcs.lua:123-147
func skillCount(sk *calculator.ActiveSkill, g *pob.Skill) int {
	if g.GroupCount != nil {
		return *g.GroupCount
	}
	if sk.ActiveEffect != nil && sk.ActiveEffect.SrcInstance != nil && sk.ActiveEffect.SrcInstance.Count != nil {
		return *sk.ActiveEffect.SrcInstance.Count
	}
	return 1
}

// countsOverLimit lists the totem and mine skills Full DPS counts more times than their active limit.
func countsOverLimit(env *calculator.Environment) []overLimit {
	return slices.DeleteFunc(countedSkills(env), func(o overLimit) bool { return float64(o.count) <= float64(o.limit) })
}

// countedSkills lists the totem and mine skills Full DPS counts more than once, with their active limit; the main
// skill's limit is its output, another skill's its skill modifiers' sum as CalcOffence takes it.
//
//pob:block:src/Modules/CalcOffence.lua:562,src/Modules/CalcOffence.lua:1185
func countedSkills(env *calculator.Environment) []overLimit {
	var out []overLimit
	for _, sk := range env.Player.ActiveSkillList {
		g, ok := sk.SocketGroup.(*pob.Skill)
		if !ok || g == nil || g.IncludeInFullDPS == nil || !*g.IncludeInFullDPS || sk.SkillModList == nil || sk.ActiveEffect == nil || sk.ActiveEffect.GrantedEffect == nil {
			continue
		}
		count := skillCount(sk, g)
		if count <= 1 {
			continue
		}
		var kind, limitKey string
		var names []string
		switch {
		case sk.SkillFlags[calculator.SkillFlagTotem]:
			kind, limitKey, names = "totem", "ActiveTotemLimit", []string{"ActiveTotemLimit", "ActiveBallistaLimit"}
		case sk.SkillFlags[calculator.SkillFlagMine]:
			kind, limitKey, names = "mine", "ActiveMineLimit", []string{"ActiveMineLimit"}
		default:
			continue
		}
		limit := sk.SkillModList.Sum(mod.TypeBase, sk.SkillCfg, names...)
		if sk == env.Player.MainSkill {
			if v, ok := env.Player.Output[limitKey]; ok {
				limit = v
			}
		}
		out = append(out, overLimit{group: sk.SocketGroupIndex, name: sk.ActiveEffect.GrantedEffect.Name(), kind: kind, count: count, limit: int(limit)})
	}
	return out
}

// totemWarnings names Full DPS counts and a summoned-totems config above the active limit; PoB multiplies Full DPS by
// the stored count (Calcs.lua:253-256) and takes the config as given (CalcOffence.lua:1434), so neither is capped.
func totemWarnings(env *calculator.Environment) []string {
	var out []string
	for _, o := range countsOverLimit(env) {
		out = append(out, fmt.Sprintf("Full DPS counts %s %d times; its active %s limit is %d", o.name, o.count, o.kind, o.limit))
	}
	out = append(out, sharedTotemWarnings(env)...)
	o := env.Player.Output
	s, l := o["TotemsSummoned"], o["ActiveTotemLimit"]
	if s > l && l > 0 {
		out = append(out, fmt.Sprintf("Config sets %v totems summoned; the main skill's active totem limit is %v", s, l))
	}
	if others := otherTotemSkills(env); s >= l && l > 0 && len(others) > 0 {
		out = append(out, fmt.Sprintf("%v totems summoned counts every totem as the main skill's, but %s also summon totems", s, strings.Join(others, ", ")))
	}
	return out
}

// otherTotemSkills names the enabled totem skills other than the main skill's.
func otherTotemSkills(env *calculator.Environment) []string {
	main := env.Player.MainSkill
	if main == nil || !main.SkillFlags[calculator.SkillFlagTotem] {
		return nil
	}
	var out []string
	for _, sk := range env.Player.ActiveSkillList {
		if sk == main || !sk.SkillFlags[calculator.SkillFlagTotem] || sk.ActiveEffect == nil || sk.ActiveEffect.GrantedEffect == nil {
			continue
		}
		if g, ok := sk.SocketGroup.(*pob.Skill); ok && g == main.SocketGroup {
			continue
		}
		if name := sk.ActiveEffect.GrantedEffect.Name(); !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
}

// clampCounts lowers each over-limit Full DPS count in b to its limit; it reports whether it changed anything.
func clampCounts(b *pob.PathOfBuilding, env *calculator.Environment) bool {
	over := countsOverLimit(env)
	to := map[int]int{}
	for _, o := range over {
		to[o.group] = max(o.limit, 0)
	}
	setCounts(b, to)
	return len(over) > 0
}

// setCounts stores each group's Full DPS count as to's, on the group or on every gem counted above it.
func setCounts(b *pob.PathOfBuilding, to map[int]int) {
	groups := groupsOf(b)
	for group, n := range to {
		if group < 1 || group > len(*groups) {
			continue
		}
		g := &(*groups)[group-1]
		if g.GroupCount != nil {
			g.GroupCount = &n
			continue
		}
		g.Gems = append([]pob.Gem(nil), g.Gems...)
		for i := range g.Gems {
			if c := g.Gems[i].Count; c != nil && *c > n {
				g.Gems[i].Count = &n
			}
		}
	}
}

// countsAfter calculates after (c's calculation) with each Full DPS totem or mine count lowered by what the change took
// off its limit, keeping any count base already had above it, and names what it lowered; after itself when nothing
// was. PoB's Full DPS keeps the stored count (Calcs.lua:253-256): go-pob's alternative, not PoB's.
func countsAfter(c *pob.PathOfBuilding, base, after *calculator.Environment) (*calculator.Environment, []string) {
	before := countedSkills(base)
	to := map[int]int{}
	var names []string
	for _, o := range countsOverLimit(after) {
		i := slices.IndexFunc(before, func(b overLimit) bool { return b.group == o.group && b.name == o.name })
		if i < 0 {
			continue
		}
		if n := min(o.count, max(o.limit+max(o.count-before[i].limit, 0), 0)); n < o.count {
			to[o.group] = n
			names = append(names, fmt.Sprintf("%s %d -> %d", o.name, o.count, n))
		}
	}
	if len(to) == 0 {
		return after, nil
	}
	cc := cloneBuild(c)
	setCounts(cc, to)
	return calcEnv(cc), []string{fmt.Sprintf("totems: dFullDPS counts %s, the limit after the change (PoB's Full DPS keeps the stored count)", strings.Join(names, ", "))}
}

// clampedEnv calculates b with its Full DPS counts held to their active limits: go-pob's alternative score, not PoB's.
func clampedEnv(b *pob.PathOfBuilding, env *calculator.Environment) *calculator.Environment {
	if len(countsOverLimit(env)) == 0 {
		return env
	}
	c := cloneBuild(b)
	clampCounts(c, env)
	return calcEnv(c)
}

// sharedTotemWarnings names Full DPS totem counts within their limit that, with the totems the build's other enabled
// totem skills summon (a Decoy Totem), exceed it: the active totem limit counts every summoned totem.
func sharedTotemWarnings(env *calculator.Environment) []string {
	type totem struct {
		sk    *calculator.ActiveSkill
		count int
		full  bool
	}
	var list []totem
	seen := map[any]bool{}
	for _, sk := range env.Player.ActiveSkillList {
		g, ok := sk.SocketGroup.(*pob.Skill)
		if !ok || g == nil || !g.Enabled || sk.DisableReason != "" || !sk.SkillFlags[calculator.SkillFlagTotem] || sk.ActiveEffect == nil || sk.ActiveEffect.GrantedEffect == nil || seen[g] {
			continue
		}
		seen[g] = true
		list = append(list, totem{sk, skillCount(sk, g), g.IncludeInFullDPS != nil && *g.IncludeInFullDPS})
	}
	over := countsOverLimit(env)
	var out []string
	for _, t := range list {
		if !t.full || slices.ContainsFunc(over, func(o overLimit) bool { return o.group == t.sk.SocketGroupIndex }) {
			continue
		}
		limit := t.sk.SkillModList.Sum(mod.TypeBase, t.sk.SkillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
		if t.sk == env.Player.MainSkill {
			if v, ok := env.Player.Output["ActiveTotemLimit"]; ok {
				limit = v
			}
		}
		others, names := 0, []string(nil)
		for _, o := range list {
			if o.sk != t.sk {
				others += o.count
				names = append(names, o.sk.ActiveEffect.GrantedEffect.Name())
			}
		}
		if others > 0 && float64(t.count+others) > limit {
			out = append(out, fmt.Sprintf("Full DPS counts %s %d times, plus %d totems from %s; the active totem limit is %v", t.sk.ActiveEffect.GrantedEffect.Name(), t.count, others, strings.Join(names, ", "), limit))
		}
	}
	return out
}
