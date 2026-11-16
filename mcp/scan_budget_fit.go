package mcp

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

const (
	// fitRounds is how many sets fitPlan chooses and checks before it gives up
	fitRounds = 12
	// fitPickLoss is the loss each pick adds while choosing, so of two sets that lose the same the smaller wins
	fitPickLoss = 0.01
)

// fitKeys are what a savings row changes, of which a set takes at most one change each: its item slot, and the gem
// (group|gem) a gem row or a links row changes.
func fitKeys(r map[string]any) []string {
	id, _ := r["id"].(string)
	p := strings.Split(id, "|")
	switch {
	case (p[0] == "gemlevel" || p[0] == "gemquality") && len(p) >= 3:
		return []string{"gem " + p[1] + "|" + p[2]}
	case p[0] == "links" && len(p) == 4:
		return []string{"slot " + p[1], "gem " + p[2] + "|" + p[3]}
	}
	slot, _ := r["slot"].(string)
	return []string{"slot " + slot}
}

// fitChoose is the set of rows, at most one per first fitKey, whose savings cover need and whose loss_pct sum (plus
// fitPickLoss a pick) is least; nil when no set covers need.
func fitChoose(rows []map[string]any, need float64) []map[string]any {
	unit := math.Max(0.1, need/100000)
	n := int(math.Ceil(need/unit - 1e-9))
	byKey := map[string][]int{}
	var order []string
	for i, r := range rows {
		k := fitKeys(r)[0]
		if byKey[k] == nil {
			order = append(order, k)
		}
		byKey[k] = append(byKey[k], i)
	}
	dp := make([]float64, n+1)
	for s := 1; s <= n; s++ {
		dp[s] = math.Inf(1)
	}
	type step struct{ row, from int }
	choice := make([][]step, len(order))
	for gi, k := range order {
		next := slices.Clone(dp)
		ch := make([]step, n+1)
		for s := range ch {
			ch[s] = step{-1, s}
		}
		for _, i := range byKey[k] {
			save := int(math.Floor(-rows[i]["cost"].(float64)/unit + 1e-9))
			if save <= 0 {
				continue
			}
			loss, _ := rows[i]["loss_pct"].(float64)
			loss += fitPickLoss
			for s := 0; s <= n; s++ {
				if math.IsInf(dp[s], 1) {
					continue
				}
				t := min(n, s+save)
				if v := dp[s] + loss; v < next[t] {
					next[t], ch[t] = v, step{i, s}
				}
			}
		}
		dp, choice[gi] = next, ch
	}
	if math.IsInf(dp[n], 1) {
		return nil
	}
	var out []map[string]any
	s := n
	for gi := len(order) - 1; gi >= 0; gi-- {
		st := choice[gi][s]
		if st.row >= 0 {
			out = append(out, rows[st.row])
		}
		s = st.from
	}
	return out
}

// fitMost is the most the rows save together, the largest saving of each first fitKey.
func fitMost(rows []map[string]any) float64 {
	best := map[string]float64{}
	for _, r := range rows {
		k := fitKeys(r)[0]
		best[k] = math.Max(best[k], -r["cost"].(float64))
	}
	out := 0.0
	for _, v := range best {
		out += v
	}
	return out
}

// fitOrder is the order a set's picks apply in: the least loss first, links rows last, as they remove a gem and
// renumber the gems after it.
func fitOrder(set []map[string]any) []map[string]any {
	out := slices.Clone(set)
	sort.SliceStable(out, func(i, j int) bool {
		li, lj := strings.HasPrefix(out[i]["id"].(string), "links|"), strings.HasPrefix(out[j]["id"].(string), "links|")
		if li != lj {
			return lj
		}
		a, _ := out[i]["loss_pct"].(float64)
		b, _ := out[j]["loss_pct"].(float64)
		if a != b {
			return a < b
		}
		return out[i]["id"].(string) < out[j]["id"].(string)
	})
	return out
}

// fitVerdict says why a set cannot stand ("set_big_loss", "set_resist_gap"), empty when it can, and the resistance
// gap it leaves with the open suffixes that refill it (nil without a gap).
type fitVerdict func(c *pob.PathOfBuilding, after *calculator.Environment) (string, any)

// fitCheck judges a set applied to c: big_loss against the current build, or a resistance gap no open suffix of the
// set's own items refills, so a refill never lands on an item the set replaces.
func (sc *scanCtx) fitCheck(c *pob.PathOfBuilding, after *calculator.Environment) (string, any) {
	if bl := bigLosses(sc.key, sc.env, after, changeColumns(sc.env, after, dpsColumn(sc.key))); len(bl) > 0 {
		return "set_big_loss", nil
	}
	gap := resistGap(sc.room, headroom(after))
	if len(gap) == 0 {
		return "", nil
	}
	r := refill(resistItems(c, sc.roll.rng), sc.room, map[string]any{"resist_gap": gap}, sc.roll)
	if _, no := r.(string); no {
		return "set_resist_gap", nil
	}
	return "", map[string]any{"resist_gap": gap, "refill": r}
}

// fitPlan brings the build under its cost cap with a set of ranked savings rows: of the clean ones (none unusable or
// with big_loss), at most one change per item slot and per gem, the set whose savings cover the excess at the least
// summed loss_pct. It applies the set a pick at a time to one copy and calculates it after each: a pick that leaves the
// set with big_loss, or with a resistance gap no open suffix of the set's items refills, is dropped and the set chosen
// again; so is the last pick when the whole set's worst case has big_loss. When no clean set covers the excess it
// picks nothing and says how close the clean rows come.
func (sc *scanCtx) fitPlan(rows []map[string]any, rz *realism) map[string]any {
	return sc.fitSets(rows, rz, sc.fitCheck)
}

func (sc *scanCtx) fitSets(rows []map[string]any, rz *realism, check fitVerdict) map[string]any {
	cb := sc.cb
	need := cb.baseTotal - *cb.limit
	if need <= 0 {
		return map[string]any{"fits": true, "note": "the build already fits the cap"}
	}
	skipped := map[string]int{}
	var pool []map[string]any
	for _, r := range rows {
		switch {
		case r["unusable"] != nil:
			skipped["unusable"]++
		case r["big_loss"] != nil:
			skipped["big_loss"]++
		case r[dirtyKey] != nil:
			skipped["flagged"]++
		case r["cost_unknown"] != nil:
			skipped["unknown_price"]++
		default:
			pool = append(pool, r)
		}
	}
	envs := map[string]*calculator.Environment{}
	want := need
	for range fitRounds {
		set := fitChoose(pool, want)
		if set == nil {
			break
		}
		c := cloneBuild(sc.b)
		var applied []change
		var picks []any
		used := map[string]bool{}
		var refilled any
		var drop map[string]any
		prefix := ""
		for _, r := range fitOrder(set) {
			id := r["id"].(string)
			why := ""
			for _, k := range fitKeys(r) {
				if used[k] {
					why = "same_slot_or_gem"
				}
			}
			var ch change
			if why == "" {
				var e *ToolError
				if ch, _, e = scanChange(c, id); e != nil {
					why = "stale"
				}
			}
			if why == "" {
				ch(c)
				prefix += id + "\n"
				after, ok := envs[prefix]
				if !ok {
					after = calcEnv(c)
					envs[prefix] = after
				}
				why, refilled = check(c, after)
			}
			if why != "" {
				skipped[why]++
				drop = r
				break
			}
			for _, k := range fitKeys(r) {
				used[k] = true
			}
			applied = append(applied, ch)
			picks = append(picks, id)
		}
		if drop == nil {
			cost, unknown := cb.delta(c)
			if len(unknown) == 0 && cb.baseTotal+cost > *cb.limit {
				// the picks' prices together differ from their rows' costs; ask the next set for the shortfall too
				want += cb.baseTotal + cost - *cb.limit
				continue
			}
			out := sc.fitResult(picks, applied, cost, unknown, rz)
			if out["big_loss"] == nil {
				if refilled != nil {
					out["refill"] = refilled
				}
				if len(skipped) > 0 {
					out["skipped"] = skipped
				}
				return out
			}
			skipped["set_big_loss"]++
			last := picks[len(picks)-1]
			drop = set[slices.IndexFunc(set, func(r map[string]any) bool { return r["id"] == last })]
		}
		pool = slices.DeleteFunc(pool, func(r map[string]any) bool { return r["id"] == drop["id"] })
	}
	most := fitMost(pool)
	out := map[string]any{"fits": false, "picks": []any{}, "closest_chaos": round(cb.baseTotal - most),
		"note": fmt.Sprintf("can't fit: the clean savings rows left, at most one change per item slot and per gem, save at most %v of the %v chaos needed (closest_chaos, before checking them together); widen parts, slots or per_slot, or raise the cap", math.Round(most), math.Round(need))}
	if len(skipped) > 0 {
		out["skipped"] = skipped
	}
	return out
}

// fitResult prices and scores a set of picks applied together, its loss_pct on the rows' basis (worst case where the
// realism scores one).
func (sc *scanCtx) fitResult(picks []any, applied []change, cost float64, unknown []string, rz *realism) map[string]any {
	cb := sc.cb
	c := cloneBuild(sc.b)
	for _, ch := range applied {
		ch(c)
	}
	after := calcEnv(c)
	gain, _, _ := rankGain(sc.ps, sc.key, sc.env, after)
	row := map[string]any{"gain": round(gain), changeKey: change(func(b *pob.PathOfBuilding) {
		for _, ch := range applied {
			ch(b)
		}
	})}
	rz.annotate([]map[string]any{row}, 1, !sc.ps.Combined)
	out := map[string]any{"picks": picks, "cost": cost, "build_chaos_after": round(cb.baseTotal + cost), "fits": len(unknown) == 0 && cb.baseTotal+cost <= *cb.limit,
		"gain": round(gain), "loss_pct": round(basisLoss(row, statOf(sc.env, sc.key), sc.env.Player.Output["TotalEHP"], sc.key == "TotalEHP"))}
	_, out["loss_of"] = rowScore(row)
	for _, k := range []string{"worst_gain", "big_loss"} {
		if v, ok := row[k]; ok {
			out[k] = v
		}
	}
	if len(unknown) > 0 {
		out["cost_unknown"] = unknown
		delete(out, "cost")
	}
	out["hint"] = "apply each pick with scan_build apply=<id>, in this order"
	return out
}
