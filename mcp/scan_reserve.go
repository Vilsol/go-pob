package mcp

import (
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

const manaUnreserved = "ManaUnreserved"

// reservingGroups maps each socket group index (1-based) of b to the reservations its active skills hold, read from
// the per-skill values CalcPerform stores on skillData (CalcPerform.lua:2005-2087).
func reservingGroups(b *pob.PathOfBuilding, env *calculator.Environment) ([]map[string]any, []int) {
	groups := *groupsOf(b)
	var rows []map[string]any
	var idx []int
	for _, sk := range env.Player.ActiveSkillList {
		sd := sk.SkillData
		if sd == nil || (sd.ManaReservedBase == 0 && sd.LifeReservedBase == 0) {
			continue
		}
		g, _ := sk.SocketGroup.(*pob.Skill)
		gi := -1
		if g != nil {
			gi = slices.IndexFunc(groups, func(x pob.Skill) bool { return sameGroup(x, *g) })
		}
		name := ""
		if sk.ActiveEffect != nil && sk.ActiveEffect.GrantedEffect != nil {
			name = sk.ActiveEffect.GrantedEffect.Name()
		}
		row := map[string]any{"skill": name}
		if gi >= 0 {
			row["group"] = gi + 1
			if !slices.Contains(idx, gi+1) {
				idx = append(idx, gi+1)
			}
		}
		if sd.ManaReservedBase != 0 {
			row["mana"] = round(sd.ManaReservedBase)
		}
		if sd.ManaReservedPercent != 0 {
			row["mana_pct"] = round(sd.ManaReservedPercent)
		}
		if sd.LifeReservedBase != 0 {
			row["life"] = round(sd.LifeReservedBase)
		}
		if sd.LifeReservedPercent != 0 {
			row["life_pct"] = round(sd.LifeReservedPercent)
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return num64(rows[i]["mana"])+num64(rows[i]["life"]) > num64(rows[j]["mana"])+num64(rows[j]["life"])
	})
	return rows, idx
}

func num64(v any) float64 {
	f, _ := v.(float64)
	return f
}

func sameGroup(a, b pob.Skill) bool {
	if a.Slot != b.Slot || a.Label != b.Label || len(a.Gems) != len(b.Gems) || a.Source != b.Source {
		return false
	}
	for i := range a.Gems {
		if a.Gems[i].NameSpec != b.Gems[i].NameSpec || a.Gems[i].Level != b.Gems[i].Level {
			return false
		}
	}
	return true
}

// reservation lists each skill's reservation and, with mana_needed, ranks where more unreserved mana comes from:
// tree nodes (PoB's power report on ManaUnreserved), a gem level or quality raise, or a support removed from a
// reserving group.
func (sc *scanCtx) reservation(deadline time.Time) ([]map[string]any, map[string]any) {
	out := sc.env.Player.Output
	skills, groups := reservingGroups(sc.b, sc.env)
	info := map[string]any{"skills": toAny(nonNil(skills))}
	for _, k := range []string{"Mana", "ManaReserved", manaUnreserved, "Life", "LifeReserved", "LifeUnreserved"} {
		if v, ok := out[k]; ok {
			info[k] = round(v)
		}
	}
	if u, ok := out[manaUnreserved]; ok && u < 0 {
		info["over_reserved"] = round(-u)
	}
	if sc.in.ManaNeeded <= 0 {
		return nil, info
	}
	info["mana_needed"] = sc.in.ManaNeeded
	base := out[manaUnreserved]
	var rows []map[string]any
	ps := calculator.PowerStat{Stat: manaUnreserved, Label: manaUnreserved}
	maxPoints := 3
	if sc.in.MaxPoints != nil {
		maxPoints = max(*sc.in.MaxPoints, 1)
	}
	nodes, _, cut, e := rankNodes(sc.b, ps, manaUnreserved, rankIn{MaxPoints: &maxPoints}, 8, nil)
	sc.partial = sc.partial || cut
	if e == nil {
		need, total, points := sc.in.ManaNeeded, 0.0, 0
		var plan []string
		for _, n := range nodes {
			ch, _ := n[changeKey].(change)
			if ch == nil {
				continue
			}
			id := fmt.Sprintf("node|%d", n["id"])
			if eff, ok := n["effect"].(int64); ok {
				id += fmt.Sprintf("|%d", eff)
			}
			row := map[string]any{"id": id, "slot": "tree", "name": n["name"], "gain": n["gain"], "points": n["points"], changeKey: ch}
			rows = append(rows, row)
			if total < need {
				total += n["gain"].(float64)
				points += n["points"].(int)
				plan = append(plan, fmt.Sprint(n["name"]))
			}
		}
		if len(plan) > 0 {
			info["node_plan"] = map[string]any{"nodes": plan, "points": points, "mana": round(total), "reaches_need": total >= need}
		}
	} else {
		info["nodes_skipped"] = e.Message
	}
	for _, gi := range groups {
		g := (*groupsOf(sc.b))[gi-1]
		for j, gem := range g.Gems {
			if time.Now().After(deadline) {
				sc.partial = true
				break
			}
			if !gem.Enabled {
				continue
			}
			var tries []struct {
				id, name string
				ch       change
			}
			if ge, err := resolveGem(gem.NameSpec); err == nil && ge.MaxLevel > 0 && gem.Level < ge.MaxLevel {
				tries = append(tries, struct {
					id, name string
					ch       change
				}{fmt.Sprintf("gemlevel|%d|%d|%d", gi, j+1, gem.Level+1), fmt.Sprintf("%s level %d", gem.NameSpec, gem.Level+1), gemLevelChange(gi, j, gem.Level+1, gem.Quality)})
			}
			if gem.Quality < 20 {
				tries = append(tries, struct {
					id, name string
					ch       change
				}{fmt.Sprintf("gemquality|%d|%d|20", gi, j+1), gem.NameSpec + " quality 20", gemLevelChange(gi, j, gem.Level, 20)})
			}
			if gemIsSupport(gem) {
				tries = append(tries, struct {
					id, name string
					ch       change
				}{fmt.Sprintf("gemremove|%d|%d", gi, j+1), "remove " + gem.NameSpec, gemRemoveChange(gi, j)})
			}
			for _, t := range tries {
				after := evalChange(sc.b, t.ch)
				if gain := round(after.Player.Output[manaUnreserved] - base); gain > 0 {
					rows = append(rows, map[string]any{"id": t.id, "slot": fmt.Sprintf("group %d", gi), "name": t.name, "gain": gain, "points": 0, changeKey: t.ch})
				}
			}
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i]["gain"].(float64) > rows[j]["gain"].(float64) })
	return rows[:min(len(rows), 8)], info
}

func gemLevelChange(group, gem, level, quality int) change {
	return func(c *pob.PathOfBuilding) {
		g := &(*groupsOf(c))[group-1]
		g.Gems[gem].Level = level
		g.Gems[gem].Quality = quality
	}
}

func gemRemoveChange(group, gem int) change {
	return func(c *pob.PathOfBuilding) {
		g := &(*groupsOf(c))[group-1]
		g.Gems = slices.Delete(slices.Clone(g.Gems), gem, gem+1)
	}
}
