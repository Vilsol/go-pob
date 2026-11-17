package mcp

import (
	"encoding/json"
	"fmt"
	"maps"
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
	capRank    = 3 * 1024
)

type rankIn struct {
	Handle      string         `json:"handle"`
	Kind        string         `json:"kind"`
	Stat        string         `json:"stat"`
	MaxPoints   *int           `json:"max_points"`
	NodeTypes   []string       `json:"node_types"`
	Group       int            `json:"group"`
	Slot        string         `json:"slot"`
	MaxLevelReq *int           `json:"max_level_req"`
	Limit       int            `json:"limit"`
	Watch       []string       `json:"watch"`
	Keep        []constraintIn `json:"keep"`
	WorstCase   bool           `json:"worst_case"`
}

var resistElems = []string{"Fire", "Cold", "Lightning", "Chaos"}

// headroom is each resistance's margin over its cap: ResistOverCap above it, minus MissingResist below it.
func headroom(env *calculator.Environment) map[string]float64 {
	out := make(map[string]float64, len(resistElems))
	//pob:block:src/Modules/CalcDefence.lua:629-631
	for _, el := range resistElems {
		over, _ := statValue(env, el+"ResistOverCap")
		missing, _ := statValue(env, "Missing"+el+"Resist")
		out[el] = round(over - missing)
	}
	return out
}

// watchSnap is the compact per-candidate view a ranking row carries: the watched stats after the change and the
// resistance headroom.
func watchSnap(env *calculator.Environment, watch []string) map[string]any {
	return map[string]any{"after": statsOf(env, watch), "headroom": headroom(env)}
}

func rankWatch(env *calculator.Environment, key string, ps calculator.PowerStat, extra []string) ([]string, []string) {
	watch := []string{"Life", "TotalEHP"}
	if !ps.Combined && !slices.Contains(watch, key) {
		watch = append([]string{key}, watch...)
	}
	var unknown []string
	for _, w := range extra {
		if !knownStat(env, w) {
			unknown = append(unknown, w)
		} else if !slices.Contains(watch, w) {
			watch = append(watch, w)
		}
	}
	return watch, unknown
}

// addSnap adds the after-view to a ranking row; headroom only when it differs from base, the current build's.
func addSnap(row map[string]any, env *calculator.Environment, watch []string, base map[string]float64) {
	for k, v := range watchSnap(env, watch) {
		row[k] = v
	}
	if maps.Equal(row["headroom"].(map[string]float64), base) {
		delete(row, "headroom")
	}
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
	if len(in.Keep) > 0 && in.Kind != "remove_nodes" {
		return nil, invalidArg("keep applies only to kind=remove_nodes.", nil)
	}
	extra := slices.Clone(in.Watch)
	for i, c := range in.Keep {
		if !knownStat(env, c.Stat) {
			e := unknownStat(c.Stat)
			e.Hint = fmt.Sprintf("keep[%d]: use a PoB output key.", i)
			return nil, e
		}
		if c.Op != ">=" && c.Op != "<=" {
			return nil, invalidArg(fmt.Sprintf("keep[%d].op must be >= or <=.", i), []string{">=", "<="})
		}
		extra = append(extra, c.Stat)
	}
	start := time.Now()
	watch, unknown := rankWatch(env, key, ps, extra)
	res := map[string]any{"handle": h.id, "revision": h.cur, "stat": key, "base_stats": statsOf(env, watch), "base_headroom": headroom(env)}
	if len(unknown) > 0 {
		res["unknown_watch"] = unknown
	}
	if !ps.Combined {
		res["base"] = round(statOf(env, key))
	}
	var rows []map[string]any
	var considered int
	partial := false
	switch in.Kind {
	case "nodes":
		rows, considered, partial, e = rankNodes(b, ps, key, in, limit, watch)
		res["hint"] = "Allocate with edit_tree op=allocate node=<id>; re-rank after each pick (gains interact)."
		if e == nil && !ps.Combined && (len(in.NodeTypes) == 0 || slices.Contains(in.NodeTypes, "mastery")) {
			sw, cut, se := masterySwitches(b, env, key, start, watch)
			if se != nil {
				return nil, se
			}
			partial = partial || cut
			if len(sw) > 0 {
				res["mastery_switches"] = toAny(sw[:min(3, len(sw))])
				res["hint"] = res["hint"].(string) + " Switch an allocated mastery with edit_tree op=set_mastery node=<id> effect=<effect>."
			}
		}
	case "remove_nodes":
		var dropped int
		rows, considered, dropped, partial = rankRemoveNodes(b, env, key, ps, in.Keep, start, watch)
		if len(in.NodeTypes) > 0 {
			rows = slices.DeleteFunc(rows, func(r map[string]any) bool { return !slices.Contains(in.NodeTypes, r["type"].(string)) })
		}
		if len(in.Keep) > 0 {
			res["dropped"] = dropped
		}
		res["hint"] = "Remove with edit_tree op=deallocate node=<id>."
	case "supports":
		rows, considered, partial, e = rankSupports(b, env, ps, key, in.Group, start, limit, watch)
		res["hint"] = "Apply with edit_skills op=add_gem or op=set_gems."
	case "uniques":
		rows, considered, partial, e = rankUniques(b, key, in, start, limit, watch)
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
	realKey := key
	if ps.Combined {
		realKey = mainDPS
	}
	rz := newRealism(b, env, realKey, time.Now().Add(rankBudget))
	sign := 1.0
	if in.Kind == "remove_nodes" {
		sign = -1
	}
	if rz.annotate(rows, sign, in.WorstCase) {
		partial = true
	}
	if ps.Combined {
		ehpNote(rows, env)
	}
	if in.Kind == "uniques" {
		slot, _ := resolveSlot(in.Slot)
		overfillFlags(rows, b, slot)
	}
	if in.Kind == "uniques" || in.Kind == "remove_nodes" {
		markUnusable(rows)
		markBigLoss(rows)
	}
	if sw, ok := res["mastery_switches"].([]any); ok {
		list := make([]map[string]any, len(sw))
		for i, r := range sw {
			list[i] = r.(map[string]any)
		}
		rz.annotate(list, 1, in.WorstCase)
	}
	if in.WorstCase {
		if !rz.scored {
			res["worst_case"] = "no combat condition or stage count is forced on or assumed off in config and no Full DPS count exceeds its totem or mine limit"
		} else {
			res["worst_case"] = "worst_" + map[bool]string{true: "gain", false: "loss"}[sign > 0] + " and worst_d are go-pob's estimate, not PoB's: forced_conditions off (stage and stack counts halved), assumed_off_conditions on, Full DPS counts held to their totem or mine limit"
			worstKey := "worst_gain"
			if sign < 0 {
				worstKey = "worst_loss"
			}
			sort.SliceStable(rows, func(i, j int) bool {
				a, aok := rows[i][worstKey].(float64)
				c, cok := rows[j][worstKey].(float64)
				if aok != cok {
					return aok
				}
				return sign*a > sign*c
			})
		}
	}
	sortUnusableLast(rows)
	for _, r := range rows {
		delete(r, dirtyKey)
	}
	if len(rz.forced) > 0 {
		res["forced_conditions"] = rz.forced
	}
	if len(rz.off) > 0 {
		res["assumed_off_conditions"] = rz.off
	}
	if c := dpsColumn(realKey); c != mainDPS {
		res["dDPS_stat"] = c
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
	fitBudget(res, "results", budget, rankMoreHint(in.Kind, in.Limit))
	return res, nil
}

// rankMoreHint says how to see the rows a cut ranking left out, with the arguments that kind takes.
func rankMoreHint(kind string, limit int) string {
	narrow := "watch fewer stats"
	switch kind {
	case "nodes", "remove_nodes":
		narrow = "narrow node_types or watch fewer stats"
	case "uniques":
		narrow = "set max_level_req or watch fewer stats"
	}
	if limit <= 8 {
		return "pass limit above 8 for a 3x output budget, or " + narrow
	}
	return narrow
}

func rankNodes(b *pob.PathOfBuilding, ps calculator.PowerStat, key string, in rankIn, limit int, watch []string) ([]map[string]any, int, bool, *ToolError) {
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
		gain, perPoint := r.PathPower*float64(r.PathDist), r.PathPower
		// Offence/Defence is a DPS fraction and a defence score for the node alone, which PoB's heat map spreads
		// over the path (CalcsTab.lua:553-554); both shown in percent, ranked by their sum
		off, def := 100*r.Offence, 100*r.Defence
		if ps.Combined {
			gain = off + def
			perPoint = gain / float64(max(r.PathDist, 1))
		}
		if gain <= 0 {
			continue
		}
		row := map[string]any{"id": r.ID, "name": r.Name, "type": t, "gain": round(gain), "points": r.PathDist, "per_point": round(perPoint)}
		if ps.Combined {
			row["offence_pct"], row["defence_pct"] = round(off), round(def)
		}
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
	// a mastery is one choice: its best effect
	seen := map[int64]bool{}
	rows = slices.DeleteFunc(rows, func(row map[string]any) bool {
		if row["type"] != "mastery" {
			return false
		}
		id := row["id"].(int64)
		dup := seen[id]
		seen[id] = true
		return dup
	})
	spec := &b.Tree.Specs[b.ActiveSpecIndex()]
	baseRoom := headroom(misc.Base())
	for _, row := range rows[:min(limit, len(rows))] {
		p, ok := row["path"].([]int64)
		if !ok {
			continue
		}
		eff, hasEffect := row["effect"].(int64)
		id := row["id"].(int64)
		row[changeKey] = change(func(c *pob.PathOfBuilding) {
			c.AllocateNodes(p)
			if hasEffect {
				c.SetMasteryEffect(id, eff)
			}
		})
		saved := spec.MasteryEffects
		if hasEffect {
			b.SetMasteryEffect(id, eff)
		}
		addSnap(row, misc.Calc(&calculator.CalcOverride{AddNodes: p}), watch, baseRoom)
		spec.MasteryEffects = saved
	}
	return rows, len(report), partial, nil
}

// masterySwitches ranks, best first, the other effects an allocated mastery could take for no points; only those that
// gain. PoB's power report offers effects only for unallocated masteries (CalcsTab.lua:563-571).
func masterySwitches(b *pob.PathOfBuilding, env *calculator.Environment, key string, start time.Time, watch []string) ([]map[string]any, bool, *ToolError) {
	v := specTreeVersion(b)
	base := statOf(env, key)
	var rows []map[string]any
	for _, sel := range pob.SpecMasterySelections(b.Tree.Specs[b.ActiveSpecIndex()]) {
		n, ok := treeNodeByID(v, sel.Node)
		if !ok || !slices.Contains(b.Build.PassiveNodes, sel.Node) {
			continue
		}
		current := ""
		for _, o := range calculator.MasteryOptions(b, sel.Node) {
			if o.Selected {
				current = o.Label
			}
		}
		for _, o := range calculator.MasteryOptions(b, sel.Node) {
			if o.Selected {
				continue
			}
			if time.Since(start) > rankBudget {
				return rows, true, nil
			}
			after, e := evalBuild(b, func(c *pob.PathOfBuilding) { c.SetMasteryEffect(sel.Node, o.Effect) })
			if e != nil {
				return nil, false, e
			}
			gain := statOf(after, key) - base
			if round(gain) <= 0 {
				continue
			}
			row := map[string]any{"id": sel.Node, "name": nodeLabel(n) + ": " + o.Label, "effect": o.Effect, "replaces": current, "gain": round(gain),
				changeKey: change(func(c *pob.PathOfBuilding) { c.SetMasteryEffect(sel.Node, o.Effect) })}
			if base != 0 {
				row["gain_pct"] = round(gain / math.Abs(base) * 100)
			}
			addSnap(row, after, watch, headroom(env))
			rows = append(rows, row)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i]["gain"].(float64) > rows[j]["gain"].(float64) })
	return rows, false, nil
}

// rankRemoveNodes ranks allocated nodes by the stat lost per point freed when each is removed with its dependents, each
// removal a full calculation (the misc calculator's node removal keeps a removed cluster socket's nodes); with keep,
// only removals that hold every keep constraint, most points freed first, and the count it dropped. Ranking FullDPS,
// a removal that lowers a totem or mine limit below its Full DPS count loses the count the limit takes away (countsAfter).
func rankRemoveNodes(b *pob.PathOfBuilding, env *calculator.Environment, key string, ps calculator.PowerStat, keep []constraintIn, start time.Time, watch []string) ([]map[string]any, int, int, bool) {
	v := specTreeVersion(b)
	starts := startNodes(b, v)
	roots := classRoots(b, v)
	targets := make([]float64, len(keep))
	for i, c := range keep {
		targets[i] = c.Value
		if c.Relative {
			targets[i] = c.Value * statOf(env, c.Stat)
		}
	}
	var rows []map[string]any
	considered, dropped := 0, 0
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
		removed := cloneBuild(b)
		removed.DeallocateNodes(cut)
		before, afterEnv := env, calcEnv(removed)
		considered++
		kept := true
		for i, c := range keep {
			if a := statOf(afterEnv, c.Stat); (c.Op == ">=" && a < targets[i]) || (c.Op == "<=" && a > targets[i]) {
				kept = false
			}
		}
		if !kept {
			dropped++
			continue
		}
		loss := statOf(before, key) - statOf(afterEnv, key)*totemShare(key, before, afterEnv)
		var countFlags []string
		if key == "FullDPS" {
			var counted *calculator.Environment
			if counted, countFlags = countsAfter(removed, before, afterEnv); counted != afterEnv {
				loss = statOf(before, key) - statOf(counted, key)
				countFlags = []string{strings.Replace(countFlags[0], "dFullDPS counts", "loss counts", 1)}
			}
		}
		var off, def float64
		if ps.Combined {
			off, def = combinedOffDefStat(before, afterEnv)
			off, def = 100*off, 100*def
			loss = off + def
		}
		freed := chargedPoints(v, cut)
		row := map[string]any{"id": id, "name": n.Name, "type": n.Type, "loss": round(loss), "points_freed": freed,
			"loss_per_point": round(loss / float64(max(freed, 1))), changeKey: change(func(c *pob.PathOfBuilding) { c.DeallocateNodes(cut) })}
		if ps.Combined {
			row["offence_pct"], row["defence_pct"] = round(off), round(def)
		}
		if len(countFlags) > 0 {
			row["flags"] = countFlags
		}
		if asc := len(cut) - freed; asc > 0 {
			row["ascendancy_points_freed"] = asc
			if freed == 0 {
				row["loss_per_point"] = round(loss / float64(asc))
			}
		}
		addSnap(row, afterEnv, watch, headroom(env))
		if also := limitChanges(before, afterEnv); len(also) > 0 {
			row["also"] = also
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if len(keep) > 0 && rows[i]["points_freed"] != rows[j]["points_freed"] {
			return rows[i]["points_freed"].(int) > rows[j]["points_freed"].(int)
		}
		return rows[i]["loss_per_point"].(float64) < rows[j]["loss_per_point"].(float64)
	})
	return rows, considered, dropped, partial
}

// rankGain is a change's gain in the ranked stat from base to after, counting only the totems after still summons
// when the stat is per totem; for Offence/Defence the DPS fraction and PoB's defence score of applying it, in percent,
// as node rows take them.
func rankGain(ps calculator.PowerStat, key string, base, after *calculator.Environment) (float64, float64, float64) {
	if !ps.Combined {
		return statOf(after, key)*totemShare(key, base, after) - statOf(base, key), 0, 0
	}
	off, def := combinedOffDefStat(after, base)
	return 100 * (off + def), 100 * off, 100 * def
}

// putOffDef adds the Offence/Defence halves to a row ranked by it.
func putOffDef(row map[string]any, ps calculator.PowerStat, off, def float64) {
	if ps.Combined {
		row["offence_pct"], row["defence_pct"] = round(off), round(def)
	}
}

// combinedOffDefStat is PoB's Offence/Defence power of modified over original: the DPS fraction and the defence
// score; a removal's loss is the power of adding the node back.
//
//pob:src/Classes/CalcsTab.lua:748-758
func combinedOffDefStat(original, modified *calculator.Environment) (float64, float64) {
	o, m := original.Player.Output, modified.Player.Output
	es := func(out map[string]float64) float64 {
		if v, ok := out["EnergyShieldRecoveryCap"]; ok {
			return v
		}
		return out["EnergyShield"]
	}
	defence := (o["LifeUnreserved"]-m["LifeUnreserved"])/math.Max(3000, m["Life"]) +
		(o["Armour"]-m["Armour"])/math.Max(10000, m["Armour"]) +
		(es(o)-es(m))/math.Max(3000, es(m)) +
		(o["Evasion"]-m["Evasion"])/math.Max(10000, m["Evasion"]) +
		(o["LifeRegenRecovery"]-m["LifeRegenRecovery"])/500 +
		(o["EnergyShieldRegenRecovery"]-m["EnergyShieldRegenRecovery"])/1000
	modifiedDps, _ := statValue(modified, mainDPS)
	originalDps, _ := statValue(original, mainDPS)
	return (originalDps - modifiedDps) / modifiedDps, defence
}

// ehpNote adds to Offence/Defence rows the EHP change applying them gives, and flags a row whose defence score and EHP
// move apart: PoB's counts life, armour, energy shield, evasion and regeneration, not EHP.
//
//pob:block:src/Classes/CalcsTab.lua:749-754
func ehpNote(rows []map[string]any, env *calculator.Environment) {
	base := env.Player.Output["TotalEHP"]
	if base <= 0 {
		return
	}
	for _, row := range rows {
		d, _ := row["d"].(map[string]float64)
		pct := round(d["dEHP"] / base * 100)
		if pct != 0 {
			row["ehp_pct"] = pct
		}
		def, _ := row["defence_pct"].(float64)
		if (math.Abs(pct) >= 5 && (math.Abs(def) < 1 || (def > 0) != (pct > 0))) || (math.Abs(def) >= 5 && math.Abs(pct) < 1) {
			flags, _ := row["flags"].([]string)
			row["flags"] = append(flags, fmt.Sprintf("defence: PoB's score %v%% counts life, armour, energy shield, evasion and regeneration, not EHP, which moves %v%%", def, pct))
		}
	}
}

// totemShare is the fraction of before's summoned totems that after still summons, when key is the DPS of a totem main
// skill: PoB's totem DPS is per totem, and it counts summoned totems as the override or the limit
// (CalcOffence.lua:1433-1434); a config above the limit counts as the limit, as no more totems can be out. Otherwise 1.
func totemShare(key string, before, after *calculator.Environment) float64 {
	if key != mainDPS && key != "CombinedDPS" && key != "TotalDPS" {
		return 1
	}
	ms := before.Player.MainSkill
	if ms == nil || !ms.SkillFlags[calculator.SkillFlagTotem] {
		return 1
	}
	out := func(env *calculator.Environment) float64 {
		s, l := env.Player.Output["TotemsSummoned"], env.Player.Output["ActiveTotemLimit"]
		if l > 0 && s > l {
			return l
		}
		return s
	}
	b, a := out(before), out(after)
	if b <= 0 || a >= b {
		return 1
	}
	return a / b
}

// limitChanges lists the counts, caps and avoidances (100 is immunity) that differ between two calculations, as
// [before, after]; a removal can cost one of these while every damage and defence total stays put.
func limitChanges(a, b *calculator.Environment) map[string][2]float64 {
	out := map[string][2]float64{}
	check := func(k string) {
		if !strings.HasSuffix(k, "Limit") && !strings.HasSuffix(k, "ChargesMax") && !strings.HasSuffix(k, "ResistMax") &&
			!strings.HasSuffix(k, "BlockChanceMax") && !strings.HasSuffix(k, "DamageReductionMax") && !strings.Contains(k, "Avoid") {
			return
		}
		va, vb := round(a.Player.Output[k]), round(b.Player.Output[k])
		if va != vb {
			out[k] = [2]float64{va, vb}
		}
	}
	for k := range a.Player.Output {
		check(k)
	}
	for k := range b.Player.Output {
		if _, ok := a.Player.Output[k]; !ok {
			check(k)
		}
	}
	return out
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

// change applies one ranked candidate to a copy of the build.
type change func(c *pob.PathOfBuilding)

func evalChange(b *pob.PathOfBuilding, ch change) *calculator.Environment {
	c := cloneBuild(b)
	ch(c)
	return calcEnv(c)
}

// slotSockets is how many gems the item in slot can hold and how many enabled gems its groups socket there.
//
//pob:block:src/Modules/CalcSetup.lua:1345-1367
func slotSockets(b *pob.PathOfBuilding, slot string) (int, int) {
	sockets, gemsIn := 0, 0
	if slot == "" {
		return 0, 0
	}
	for _, e := range equippedItems(b) {
		if e.slot != slot {
			continue
		}
		for _, sk := range calculator.ParseItem(e.text).Sockets {
			if sk.Color == "R" || sk.Color == "G" || sk.Color == "B" || sk.Color == "W" {
				sockets++
			}
		}
	}
	for _, g := range *groupsOf(b) {
		if g.Source == "" && g.Enabled && g.Slot == slot {
			for _, gem := range g.Gems {
				if gem.Enabled {
					gemsIn++
				}
			}
		}
	}
	return sockets, gemsIn
}

// rankSupports ranks each compatible support by adding it and by swapping it for the weakest support: by the add, its
// change and after values, while the item has a free socket; on a group that fills its item's sockets there is no add,
// rows rank by the swap and the top rows try every support to swap out.
func rankSupports(b *pob.PathOfBuilding, env *calculator.Environment, ps calculator.PowerStat, key string, group int, start time.Time, limit int, watch []string) ([]map[string]any, int, bool, *ToolError) {
	group, e := mainGroup(b, group)
	if e != nil {
		return nil, 0, false, e
	}
	g := (*groupsOf(b))[group-1]
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
		if gain, _, _ := rankGain(ps, key, env, after); -gain < weakestLoss {
			weakest, weakestLoss = i, -gain
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
	sockets, gemsIn := slotSockets(b, g.Slot)
	full := sockets > 0 && gemsIn >= sockets && weakest >= 0
	swapIn := func(ge gemEntry, at int) change {
		return func(c *pob.PathOfBuilding) {
			gg := &(*groupsOf(c))[group-1]
			gg.Gems = slices.Clone(gg.Gems)
			gg.Gems[at] = replacedGem(c, gg.Gems[at], ge, nil, nil)
		}
	}
	swap := func(ge gemEntry, at int) (*calculator.Environment, *ToolError) {
		return evalBuild(b, swapIn(ge, at))
	}
	var rows []map[string]any
	var entries []gemEntry
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
		row := map[string]any{"name": ge.Name}
		if full {
			row["note"] = fmt.Sprintf("group full (%d sockets): best swap", sockets)
		} else {
			gem := addedGem(b, ge, nil, nil)
			add := change(func(c *pob.PathOfBuilding) {
				gg := &(*groupsOf(c))[group-1]
				gg.Gems = append(slices.Clone(gg.Gems), gem)
			})
			row[changeKey] = add
			added, e := evalBuild(b, add)
			if e != nil {
				return nil, 0, false, e
			}
			gain, off, def := rankGain(ps, key, env, added)
			row["gain_added"] = round(gain)
			putOffDef(row, ps, off, def)
			addSnap(row, added, watch, headroom(env))
		}
		if weakest >= 0 {
			swapped, e := swap(ge, weakest)
			if e != nil {
				return nil, 0, false, e
			}
			gain, off, def := rankGain(ps, key, env, swapped)
			row["gain_swapped"] = round(gain)
			row["swaps_out"] = g.Gems[weakest].NameSpec
			if full {
				putOffDef(row, ps, off, def)
				row[changeKey] = swapIn(ge, weakest)
				addSnap(row, swapped, watch, headroom(env))
			}
		}
		rows = append(rows, row)
		entries = append(entries, ge)
	}
	score := func(r map[string]any) float64 {
		if v, ok := r["gain_swapped"].(float64); ok && full {
			return v
		}
		return r["gain_added"].(float64)
	}
	byScore := func() {
		sort.SliceStable(rows, func(i, j int) bool { return score(rows[i]) > score(rows[j]) })
	}
	if !full {
		byScore()
		return rows, considered, partial, nil
	}
	for i, row := range rows {
		row["entry"] = i
	}
	byScore()
	for _, row := range rows[:min(limit, len(rows))] {
		if time.Since(start) > rankBudget {
			partial = true
			break
		}
		ge := entries[row["entry"].(int)]
		for _, i := range supportsIdx {
			if i == weakest {
				continue
			}
			swapped, e := swap(ge, i)
			if e != nil {
				return nil, 0, false, e
			}
			if gain, off, def := rankGain(ps, key, env, swapped); round(gain) > row["gain_swapped"].(float64) {
				row["gain_swapped"] = round(gain)
				putOffDef(row, ps, off, def)
				row["swaps_out"] = g.Gems[i].NameSpec
				row[changeKey] = swapIn(ge, i)
				addSnap(row, swapped, watch, headroom(env))
			}
		}
	}
	for _, row := range rows {
		delete(row, "entry")
	}
	byScore()
	return rows, considered, partial, nil
}

// overfillFlags flags uniques rows whose item has fewer sockets than the slot's groups hold, when the build already
// overfills the slot and so realismFlags, which names only new overflow, does not.
func overfillFlags(rows []map[string]any, b *pob.PathOfBuilding, slot string) {
	_, gemsIn := slotSockets(b, slot)
	for _, row := range rows {
		n, ok := row["sockets"].(int)
		if !ok || n >= gemsIn {
			continue
		}
		flags, _ := row["flags"].([]string)
		if !slices.ContainsFunc(flags, func(f string) bool { return strings.HasPrefix(f, "sockets: ") }) {
			row["flags"] = append(flags, fmt.Sprintf("sockets: %d gems in %d sockets of %s", gemsIn, n, slot))
		}
	}
}

// disabledSkills names the skills enabled on base that after disables, with PoB's reason ("This skill requires a
// Shield").
func disabledSkills(base, after *calculator.Environment) []string {
	name := func(sk *calculator.ActiveSkill) string {
		if sk.ActiveEffect == nil || sk.ActiveEffect.GrantedEffect == nil {
			return ""
		}
		return sk.ActiveEffect.GrantedEffect.Name()
	}
	enabled := map[string]bool{}
	for _, sk := range base.Player.ActiveSkillList {
		if sk.DisableReason == "" {
			enabled[name(sk)] = true
		}
	}
	var out []string
	for _, sk := range after.Player.ActiveSkillList {
		if n := name(sk); n != "" && sk.DisableReason != "" && enabled[n] {
			if l := fmt.Sprintf("%s (%s)", n, sk.DisableReason); !slices.Contains(out, l) {
				out = append(out, l)
			}
		}
	}
	return out
}

// uniqueTitle is an item name without its base type ("Kaom's Heart, Glorious Plate" -> "Kaom's Heart").
func uniqueTitle(name string) string {
	t, _, _ := strings.Cut(name, ",")
	return norm(t)
}

// coloursDiffer reports whether it's sockets take a different colour than old's, or match differently, in a socket the slot's gems fill;
// PoB's item comparison keeps the equipped item's colour matches (SkillsTab.lua:1257-1264), the equipped change does not.
//
//pob:block:src/Classes/SkillsTab.lua:1255-1275
func coloursDiffer(old, it *calculator.Item, gems int) bool {
	if it.SocketColourAlwaysMatches || old.SocketColourAlwaysMatches {
		return it.SocketColourAlwaysMatches != old.SocketColourAlwaysMatches
	}
	for i := range min(gems, max(len(old.Sockets), len(it.Sockets))) {
		if i >= len(old.Sockets) || i >= len(it.Sockets) || old.Sockets[i].Color != it.Sockets[i].Color {
			return true
		}
	}
	return false
}

// dirtyKey marks a uniques row whose gain comes with a cost its flags name; such rows rank after the clean ones.
const dirtyKey = "\x00dirty"

func rankUniques(b *pob.PathOfBuilding, key string, in rankIn, start time.Time, limit int, watch []string) ([]map[string]any, int, bool, *ToolError) {
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
	ps, e := powerStat(misc.Base(), key)
	if e != nil {
		return nil, 0, false, e
	}
	base := statOf(misc.Base(), key)
	baseRoom := headroom(misc.Base())
	equippedNames := map[string]bool{}
	for _, eq := range equippedItems(b) {
		equippedNames[uniqueTitle(eq.name)] = true
	}
	_, gemsIn := slotSockets(b, slot)
	// a flask counts only while active: into an empty slot it is PoB's tooltip toggle (ItemsTab.lua:4866), into an
	// occupied one it is equipped and activated in its place
	flask, emptySlot := strings.HasPrefix(slot, "Flask"), slotItemName(b, slot) == ""
	old := &calculator.Item{}
	for _, eq := range equippedItems(b) {
		if eq.slot == slot {
			old = calculator.ParseItem(eq.text)
		}
	}
	slotGems := 0
	for _, g := range *groupsOf(b) {
		if g.Source == "" && g.Slot == slot {
			slotGems += len(g.Gems)
		}
	}
	offHand := slotItemName(b, "Weapon 2") != ""
	baseOut := misc.Base().Player.Output
	twoHanded := func(env *calculator.Environment) bool { return env.Player.ModDB.Conditions["UsingTwoHandedWeapon"] }
	var rows []map[string]any
	considered := 0
	partial := false
	for _, it := range calculator.ItemDB(calculator.ItemDBUnique) {
		if !calculator.IsItemValidForSlot(b, it, slot) || equippedNames[uniqueTitle(it.Name)] {
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
		override := &calculator.CalcOverride{RepSlotName: slot, RepItem: it}
		if flask && emptySlot {
			text, _ := calculator.ItemDBText(calculator.ItemDBUnique, it.Name)
			item := calculator.ParseItem(text)
			if item.Base == nil || item.Base.Flask == nil {
				continue
			}
			override = &calculator.CalcOverride{ToggleFlask: item}
		}
		after := misc.Calc(override)
		gain, off, def := rankGain(ps, key, misc.Base(), after)
		row := map[string]any{"name": it.Name}
		recoloured := coloursDiffer(old, it, slotGems)
		if flask && !emptySlot {
			text, _ := calculator.ItemDBText(calculator.ItemDBUnique, it.Name)
			after = evalChange(b, flaskChange(text, slot))
			gain, off, def = rankGain(ps, key, misc.Base(), after)
		} else if recoloured || len(it.GrantedSkills) > 0 {
			kept := gain
			text, _ := calculator.ItemDBText(calculator.ItemDBUnique, it.Name)
			after = evalChange(b, func(c *pob.PathOfBuilding) { calculator.EquipItem(c, text, slot) })
			gain, off, def = rankGain(ps, key, misc.Base(), after)
			if recoloured && round(kept) != round(gain) {
				f := fmt.Sprintf("socket colours: gain %v keeping %s's current colours (PoB's item comparison", round(kept), slot)
				if len(it.GrantedSkills) > 0 {
					f += ", which also leaves out the item's granted skill"
				}
				row["flags"] = []string{f + ")"}
			}
		}
		row["gain"] = round(gain)
		putOffDef(row, ps, off, def)
		if base != 0 && !ps.Combined {
			row["gain_pct"] = round(gain / math.Abs(base) * 100)
		}
		addSnap(row, after, watch, baseRoom)
		// a gain that overfills the slot's sockets, drops the off-hand or loses a totem is not a clean one
		if len(it.Sockets) < gemsIn || (offHand && twoHanded(after) && !twoHanded(misc.Base())) ||
			after.Player.Output["ActiveTotemLimit"] < baseOut["ActiveTotemLimit"] || len(disabledSkills(misc.Base(), after)) > 0 {
			row[dirtyKey] = true
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if di, dj := rows[i][dirtyKey] != nil, rows[j][dirtyKey] != nil; di != dj {
			return dj
		}
		return rows[i]["gain"].(float64) > rows[j]["gain"].(float64)
	})
	for _, row := range rows {
		delete(row, dirtyKey)
	}
	for _, row := range rows[:min(limit, len(rows))] {
		text, _ := calculator.ItemDBText(calculator.ItemDBUnique, row["name"].(string))
		row[changeKey] = change(func(c *pob.PathOfBuilding) { calculator.EquipItem(c, text, slot) })
		if flask {
			row[changeKey] = flaskChange(text, slot)
		}
		item := calculator.ParseItem(text)
		row["sockets"] = len(item.Sockets)
		row["links"] = maxLinks(item)
	}
	return rows, considered, partial, nil
}
