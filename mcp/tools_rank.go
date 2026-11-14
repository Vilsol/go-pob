package mcp

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

const (
	rankBudget = 30 * time.Second
	capRank    = 2 * 1024
)

type rankIn struct {
	Handle      string   `json:"handle"`
	Kind        string   `json:"kind"`
	Stat        string   `json:"stat"`
	MaxPoints   *int     `json:"max_points"`
	NodeTypes   []string `json:"node_types"`
	Group       int      `json:"group"`
	Slot        string   `json:"slot"`
	MaxLevelReq *int     `json:"max_level_req"`
	Limit       int      `json:"limit"`
}

// powerStat maps a stat key to PoB's power stat; MainDPS is PoB's CombinedDPS entry, which already adds minion DPS.
func powerStat(env *calculator.Environment, key string) (calculator.PowerStat, *ToolError) {
	if key == "" || key == mainDPS {
		return calculator.PowerStat{Stat: "CombinedDPS", Label: "Main DPS"}, nil
	}
	if norm(key) == norm("OffenceDefence") || norm(key) == norm("Offence/Defence") {
		return calculator.PowerStat{Label: "Offence/Defence", Combined: true}, nil
	}
	for _, p := range calculator.PowerStats() {
		if p.Stat == key {
			return p, nil
		}
	}
	if knownStat(env, key) {
		return calculator.PowerStat{Stat: key, Label: key}, nil
	}
	return calculator.PowerStat{}, unknownStat(key)
}

func unknownStat(key string) *ToolError {
	e := errf(CodeUnknownStat, "No stat key %q.", key)
	e.DidYouMean = suggest(key, statKeys(), 5)
	e.Hint = "Stat keys are PoB output keys; search_data kind=stat lists them."
	return e
}

// statOf reads a stat for ranking: the power stat's own value (negated stats negated back), or MainDPS.
func statOf(env *calculator.Environment, key string) float64 {
	v, _ := statValue(env, key)
	return v
}

func (s *Server) rankCandidates(args json.RawMessage) (any, *ToolError) {
	var in rankIn
	if e := decode(args, schemaRankCandidates, &in); e != nil {
		return nil, e
	}
	h, e := s.get(in.Handle)
	if e != nil {
		return nil, e
	}
	b, e := h.parseRev(h.cur)
	if e != nil {
		return nil, e
	}
	env, e := h.envAt(h.cur)
	if e != nil {
		return nil, e
	}
	key := in.Stat
	if key == "" {
		key = mainDPS
	}
	ps, e := powerStat(env, key)
	if e != nil {
		return nil, e
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 8
	}
	limit = min(limit, 25)
	start := time.Now()
	res := map[string]any{"handle": h.id, "revision": h.cur, "stat": key}
	if !ps.Combined {
		res["base"] = round(statOf(env, key))
	}
	var rows []map[string]any
	var considered int
	partial := false
	switch in.Kind {
	case "nodes":
		rows, considered, partial, e = rankNodes(b, ps, key, in)
		res["hint"] = "Allocate with edit_tree op=allocate node=<id>; re-rank after each pick (gains interact)."
	case "remove_nodes":
		rows, considered, partial = rankRemoveNodes(b, env, key, start)
		res["hint"] = "Remove with edit_tree op=deallocate node=<id>."
	case "supports":
		rows, considered, partial, e = rankSupports(b, env, key, in.Group, start)
		res["hint"] = "Apply with edit_skills op=add_gem or op=set_gems."
	case "uniques":
		rows, considered, partial, e = rankUniques(b, key, in, start)
		res["hint"] = "Equip with edit_items op=equip_unique name=<name> slot=<slot>."
	default:
		return nil, invalidArg(fmt.Sprintf("Unknown kind %q.", in.Kind), []string{"nodes", "remove_nodes", "supports", "uniques"})
	}
	if e != nil {
		return nil, e
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	res["results"] = toAny(nonNil(rows))
	res["considered"] = considered
	res["elapsed_ms"] = time.Since(start).Milliseconds()
	if partial {
		res["partial"] = true
	}
	budget := capRank
	if in.Limit > 8 {
		budget = capRank * 3
	}
	fitBudget(res, "results", budget, "lower limit or narrow node_types")
	return res, nil
}

func rankNodes(b *pob.PathOfBuilding, ps calculator.PowerStat, key string, in rankIn) ([]map[string]any, int, bool, *ToolError) {
	v := specTreeVersion(b)
	if v != data.LatestTreeVersion {
		e := errf(CodeTreeNotLatest, "Node ranking needs the %s tree; this build uses %s.", data.LatestTreeVersion, v)
		e.Hint = "Re-import with import_build convert_tree:true."
		return nil, 0, false, e
	}
	maxPoints := 5
	if in.MaxPoints != nil {
		maxPoints = max(*in.MaxPoints, 1)
	}
	misc := calculator.GetMiscCalculator(b)
	pb := misc.NewPowerBuilder(ps, maxPoints)
	partial := pb.Step(rankBudget) < 1
	base := statOf(misc.Base(), key)
	plan := planPaths(b, v)
	var rows []map[string]any
	report := pb.Report()
	for _, r := range report {
		if r.Allocated || r.Cluster || r.PathDist > maxPoints {
			continue
		}
		t := "small"
		if n, ok := treeNodeByID(v, r.ID); ok {
			t = n.Type
		}
		if len(in.NodeTypes) > 0 && !slices.Contains(in.NodeTypes, t) {
			continue
		}
		gain := r.PathPower * float64(r.PathDist)
		if ps.Combined {
			gain = r.Power
		}
		if gain <= 0 {
			continue
		}
		row := map[string]any{"id": r.ID, "name": r.Name, "type": t, "gain": round(gain), "points": r.PathDist, "per_point": round(r.PathPower)}
		if !ps.Combined && base != 0 {
			row["gain_pct"] = round(gain / math.Abs(base) * 100)
		}
		if r.Effect != 0 {
			row["effect"] = r.Effect
		}
		if p := plan.path(r.ID); p != nil {
			slices.Reverse(p)
			row["path"] = p
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i]["per_point"].(float64) > rows[j]["per_point"].(float64) })
	return rows, len(report), partial, nil
}

func rankRemoveNodes(b *pob.PathOfBuilding, env *calculator.Environment, key string, start time.Time) ([]map[string]any, int, bool) {
	v := specTreeVersion(b)
	misc := calculator.GetMiscCalculator(b)
	base := statOf(misc.Base(), key)
	starts := startNodes(b, v)
	roots := classRoots(b, v)
	var rows []map[string]any
	considered := 0
	partial := false
	for _, id := range b.Build.PassiveNodes {
		if slices.Contains(starts, id) {
			continue
		}
		n, ok := treeNodeByID(v, id)
		if !ok {
			continue
		}
		if time.Since(start) > rankBudget {
			partial = true
			break
		}
		cut := append([]int64{id}, data.TreeVersions[v].CalculateDependentNodes(slices.Clone(b.Build.PassiveNodes), id, roots)...)
		after := statOf(misc.Calc(&calculator.CalcOverride{RemoveNodes: cut}), key)
		considered++
		loss := base - after
		rows = append(rows, map[string]any{"id": id, "name": n.Name, "type": n.Type, "loss": round(loss), "points_freed": len(cut),
			"loss_per_point": round(loss / float64(len(cut)))})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i]["loss_per_point"].(float64) < rows[j]["loss_per_point"].(float64)
	})
	return rows, considered, partial
}

// supportCompat reports which of gems can support the group's active skills (PoB's GemSelectControl sort cache).
func supportCompat(env *calculator.Environment, group int, list []gemEntry) []bool {
	gemsIn := make([]pob.Gem, len(list))
	for i, g := range list {
		gemsIn[i] = newGem(g, 20, 20)
	}
	rows := env.GemSortCache(group-1, gemsIn)
	out := make([]bool, len(list))
	for i, r := range rows {
		out[i] = r.CanSupport
	}
	return out
}

func mainGroup(b *pob.PathOfBuilding, group int) (int, *ToolError) {
	if group == 0 {
		group = b.Build.MainSocketGroup
	}
	if _, e := groupArg(b, group); e != nil {
		return 0, e
	}
	return group, nil
}

// evalBuild calculates a modified copy of b; mutate edits the copy.
func evalBuild(b *pob.PathOfBuilding, mutate func(c *pob.PathOfBuilding)) (*calculator.Environment, *ToolError) {
	xml, e := serialize(b)
	if e != nil {
		return nil, e
	}
	h := &handle{revs: []*revision{{xml: xml}}}
	c, e := h.parseRev(0)
	if e != nil {
		return nil, e
	}
	mutate(c)
	return calcEnv(c), nil
}

func rankSupports(b *pob.PathOfBuilding, env *calculator.Environment, key string, group int, start time.Time) ([]map[string]any, int, bool, *ToolError) {
	group, e := mainGroup(b, group)
	if e != nil {
		return nil, 0, false, e
	}
	g := (*groupsOf(b))[group-1]
	base := statOf(env, key)
	var supportsIdx []int
	for i, gem := range g.Gems {
		if gemIsSupport(gem) && gem.Enabled {
			supportsIdx = append(supportsIdx, i)
		}
	}
	weakest, weakestLoss := -1, math.Inf(1)
	for _, i := range supportsIdx {
		after, e := evalBuild(b, func(c *pob.PathOfBuilding) {
			gg := &(*groupsOf(c))[group-1]
			gg.Gems = slices.Delete(slices.Clone(gg.Gems), i, i+1)
		})
		if e != nil {
			return nil, 0, false, e
		}
		if loss := base - statOf(after, key); loss < weakestLoss {
			weakest, weakestLoss = i, loss
		}
	}
	var candidates []gemEntry
	for _, ge := range gems() {
		if !ge.Support || slices.ContainsFunc(g.Gems, func(x pob.Gem) bool { return gemMatches(x, ge.Name) }) {
			continue
		}
		candidates = append(candidates, ge)
	}
	compat := supportCompat(env, group, candidates)
	var rows []map[string]any
	considered := 0
	partial := false
	for i, ge := range candidates {
		if !compat[i] {
			continue
		}
		if time.Since(start) > rankBudget {
			partial = true
			break
		}
		considered++
		gem := newGem(ge, 20, 20)
		added, e := evalBuild(b, func(c *pob.PathOfBuilding) {
			gg := &(*groupsOf(c))[group-1]
			gg.Gems = append(slices.Clone(gg.Gems), gem)
		})
		if e != nil {
			return nil, 0, false, e
		}
		row := map[string]any{"name": ge.Name, "gain_added": round(statOf(added, key) - base)}
		if weakest >= 0 {
			swapped, e := evalBuild(b, func(c *pob.PathOfBuilding) {
				gg := &(*groupsOf(c))[group-1]
				gg.Gems = slices.Clone(gg.Gems)
				gg.Gems[weakest] = gem
			})
			if e != nil {
				return nil, 0, false, e
			}
			row["gain_swapped"] = round(statOf(swapped, key) - base)
			row["swaps_out"] = g.Gems[weakest].NameSpec
		}
		rows = append(rows, row)
	}
	score := func(r map[string]any) float64 {
		if v, ok := r["gain_swapped"].(float64); ok {
			return v
		}
		return r["gain_added"].(float64)
	}
	sort.SliceStable(rows, func(i, j int) bool { return score(rows[i]) > score(rows[j]) })
	return rows, considered, partial, nil
}

func rankUniques(b *pob.PathOfBuilding, key string, in rankIn, start time.Time) ([]map[string]any, int, bool, *ToolError) {
	if in.Slot == "" {
		e := invalidArg("kind=uniques needs slot.", nil)
		e.Valid = slotNames()
		return nil, 0, false, e
	}
	slot, e := resolveSlot(in.Slot)
	if e != nil {
		return nil, 0, false, e
	}
	misc := calculator.GetMiscCalculator(b)
	baseEnv := misc.Base()
	base := statOf(baseEnv, key)
	resists := []string{"FireResist", "ColdResist", "LightningResist", "ChaosResist"}
	var rows []map[string]any
	considered := 0
	partial := false
	for _, it := range calculator.ItemDB(calculator.ItemDBUnique) {
		if !calculator.IsItemValidForSlot(b, it, slot) {
			continue
		}
		if in.MaxLevelReq != nil && int(it.Requirements["Level"]) > *in.MaxLevelReq {
			continue
		}
		if time.Since(start) > rankBudget {
			partial = true
			break
		}
		considered++
		after := misc.Calc(&calculator.CalcOverride{RepSlotName: slot, RepItem: it})
		gain := statOf(after, key) - base
		row := map[string]any{"name": it.Name, "gain": round(gain)}
		if base != 0 {
			row["gain_pct"] = round(gain / math.Abs(base) * 100)
		}
		var rd []string
		for _, r := range resists {
			d := statOf(after, r) - statOf(baseEnv, r)
			if math.Abs(d) >= 1 {
				rd = append(rd, fmt.Sprintf("%s %+d", strings.TrimSuffix(r, "Resist"), int(math.Round(d))))
			}
		}
		if len(rd) > 0 {
			row["resists"] = rd
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i]["gain"].(float64) > rows[j]["gain"].(float64) })
	return rows, considered, partial, nil
}
