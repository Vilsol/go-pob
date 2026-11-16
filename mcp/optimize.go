package mcp

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

const (
	maxBudget      = 2000
	beamWidth      = 3
	restarts       = 3
	capOptimize    = 3 * 1024
	capOptimizeMax = 5 * 1024
)

type optSlotIn struct {
	Kind        string   `json:"kind"`
	Slot        string   `json:"slot"`
	Uniques     []string `json:"uniques"`
	Texts       []string `json:"texts"`
	KeepCurrent *bool    `json:"keep_current"`
	Group       int      `json:"group"`
	Candidates  []string `json:"candidates"`
	Replace     []string `json:"replace"`
	Remove      []string `json:"remove"`
	K           *int     `json:"k"`
	Path        *bool    `json:"path"`
	Gem         string   `json:"gem"`
	Levels      []int    `json:"levels"`
	Qualities   []int    `json:"qualities"`
	Corrupted   bool     `json:"corrupted"`
}

type objectiveIn struct {
	Stat string `json:"stat"`
	Goal string `json:"goal"`
}

type constraintIn struct {
	Stat     string  `json:"stat"`
	Op       string  `json:"op"`
	Value    float64 `json:"value"`
	Relative bool    `json:"relative"`
}

type optimizeIn struct {
	Handle      string         `json:"handle"`
	Slots       []optSlotIn    `json:"slots"`
	Objective   objectiveIn    `json:"objective"`
	Constraints []constraintIn `json:"constraints"`
	Budget      *int           `json:"budget"`
	MaxMs       *int           `json:"max_ms"`
	Seed        *int64         `json:"seed"`
	Top         *int           `json:"top"`
	Apply       bool           `json:"apply"`
	Pareto      bool           `json:"pareto"`
	budgetIn
}

const (
	slotOne   = iota // one of n choices
	slotExact        // exactly k of n
	slotUpTo         // up to k of the first addN, plus any of the rest
)

// optSlot is one dimension of the search; a selection is a sorted list of indices into its pool.
type optSlot struct {
	name string
	kind int
	n    int
	k    int
	addN int
	// group marks pool entries that exclude each other (the effects of one mastery); -1 for none
	group   []int
	start   []int
	apply   func(c *pob.PathOfBuilding, sel []int)
	label   func(sel []int) any
	sizeMsg string
	// moveMsg explains how many moves one sweep of this slot costs
	moveMsg string
}

func (sl optSlot) size() float64 {
	switch sl.kind {
	case slotExact:
		return binom(sl.n, sl.k)
	case slotUpTo:
		adds := 0.0
		for a := 0; a <= min(sl.k, sl.addN); a++ {
			adds += binom(sl.addN, a)
		}
		return adds * math.Pow(2, float64(sl.n-sl.addN))
	}
	return float64(sl.n)
}

// count is the number of valid selections: size without the add subsets that pick two effects of one mastery,
// counted while the raw subsets are few enough to walk and the valid ones could fit the budget cap.
func (sl optSlot) count() float64 {
	raw := sl.size()
	mult := math.Pow(2, float64(sl.n-sl.addN))
	if sl.kind != slotUpTo || sl.group == nil || raw/mult > 1e5 {
		return raw
	}
	adds := 0.0
	for a := 0; a <= min(sl.k, sl.addN); a++ {
		for _, combo := range combinations(sl.addN, a) {
			if sl.compatible(combo) {
				adds++
			}
		}
		if adds*mult > maxBudget {
			return raw
		}
	}
	return adds * mult
}

// enumerate lists every selection; only called on small spaces. The start selection sorts first.
func (sl optSlot) enumerate() [][]int {
	switch sl.kind {
	case slotExact:
		return combinations(sl.n, sl.k)
	case slotUpTo:
		var out [][]int
		nr := sl.n - sl.addN
		for a := 0; a <= min(sl.k, sl.addN); a++ {
			for _, combo := range combinations(sl.addN, a) {
				if !sl.compatible(combo) {
					continue
				}
				for mask := range 1 << nr {
					sel := slices.Clone(combo)
					for r := range nr {
						if mask&(1<<r) != 0 {
							sel = append(sel, sl.addN+r)
						}
					}
					out = append(out, sel)
				}
			}
		}
		return out
	}
	out := make([][]int, sl.n)
	for i := range out {
		out[i] = []int{i}
	}
	return out
}

// clashes reports whether adding x to sel picks a second entry of one exclusive group.
func (sl optSlot) clashes(sel []int, x int) bool {
	if sl.group == nil || sl.group[x] < 0 {
		return false
	}
	return slices.ContainsFunc(sel, func(m int) bool { return m != x && sl.group[m] == sl.group[x] })
}

// valid reports whether sel is a selection the slot can make.
func (sl optSlot) valid(sel []int) bool {
	switch sl.kind {
	case slotOne:
		return len(sel) == 1
	case slotExact:
		return len(sel) == sl.k
	}
	adds := 0
	for _, x := range sel {
		if x < sl.addN {
			adds++
		}
	}
	return adds <= sl.k && sl.compatible(sel)
}

func (sl optSlot) compatible(sel []int) bool {
	for i, x := range sel {
		if sl.clashes(sel[:i], x) {
			return false
		}
	}
	return true
}

// neighbours are the selections one move away: another choice, a swap, an add, a removal, or a removal plus an add
// (the only point-neutral move at full points).
func (sl optSlot) neighbours(sel []int) [][]int {
	var out [][]int
	in := make([]bool, sl.n)
	for _, x := range sel {
		in[x] = true
	}
	with := func(drop, add int) []int {
		s := make([]int, 0, len(sel)+1)
		for _, x := range sel {
			if x != drop {
				s = append(s, x)
			}
		}
		if add >= 0 {
			s = append(s, add)
		}
		slices.Sort(s)
		return s
	}
	switch sl.kind {
	case slotOne:
		for i := range sl.n {
			if i != sel[0] {
				out = append(out, []int{i})
			}
		}
	case slotExact:
		for _, m := range sel {
			for x := range sl.n {
				if !in[x] {
					out = append(out, with(m, x))
				}
			}
		}
	case slotUpTo:
		adds := 0
		for _, x := range sel {
			if x < sl.addN {
				adds++
			}
		}
		for x := range sl.n {
			if !in[x] && (x >= sl.addN || adds < sl.k) && !sl.clashes(sel, x) {
				out = append(out, with(-1, x))
			}
		}
		for _, m := range sel {
			out = append(out, with(m, -1))
		}
		if adds < sl.k {
			for r := sl.addN; r < sl.n; r++ {
				if in[r] {
					continue
				}
				for x := range sl.addN {
					if !in[x] && !sl.clashes(sel, x) {
						s := append(with(-1, r), x)
						slices.Sort(s)
						out = append(out, s)
					}
				}
			}
		}
		for _, m := range sel {
			if m >= sl.addN {
				continue
			}
			for x := range sl.addN {
				if !in[x] {
					if s := with(m, x); sl.compatible(s) {
						out = append(out, s)
					}
				}
			}
		}
	}
	return out
}

// refills are the free-and-refill moves of an add slot: drop two or three of the selected adds together and add one
// other, which frees the points a longer path needs; one-change moves cannot reach that at full points.
func (sl optSlot) refills(sel []int) [][]int {
	if sl.kind != slotUpTo {
		return nil
	}
	var picked []int
	for _, x := range sel {
		if x < sl.addN {
			picked = append(picked, x)
		}
	}
	var out [][]int
	for k := 2; k <= min(3, len(picked)); k++ {
		for _, drop := range combinations(len(picked), k) {
			kept := slices.DeleteFunc(slices.Clone(sel), func(x int) bool {
				return slices.ContainsFunc(drop, func(d int) bool { return picked[d] == x })
			})
			for a := range sl.addN {
				if !slices.Contains(sel, a) && !sl.clashes(kept, a) {
					s := append(slices.Clone(kept), a)
					slices.Sort(s)
					out = append(out, s)
				}
			}
		}
	}
	return out
}

// movesFor is the neighbour count of the start selection for a pool of n (n-addN fixed for removals).
func (sl optSlot) movesFor(n int) int {
	switch sl.kind {
	case slotExact:
		return sl.k * (n - sl.k)
	case slotUpTo:
		adds := n - (sl.n - sl.addN)
		if sl.k == 0 {
			adds = 0
		}
		return adds + (sl.n-sl.addN)*(1+adds)
	}
	return n - 1
}

func (sl optSlot) random(rng *rand.Rand) []int {
	switch sl.kind {
	case slotExact:
		s := rng.Perm(sl.n)[:sl.k]
		slices.Sort(s)
		return s
	case slotUpTo:
		want := rng.IntN(min(sl.k, sl.addN) + 1)
		var s []int
		for _, x := range rng.Perm(sl.addN) {
			if len(s) < want && !sl.clashes(s, x) {
				s = append(s, x)
			}
		}
		for r := sl.addN; r < sl.n; r++ {
			if rng.IntN(4) == 0 {
				s = append(s, r)
			}
		}
		slices.Sort(s)
		return s
	}
	return []int{rng.IntN(sl.n)}
}

type state [][]int

func (st state) key() string {
	var b strings.Builder
	for i, sel := range st {
		if i > 0 {
			b.WriteByte('|')
		}
		for j, x := range sel {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.Itoa(x))
		}
	}
	return b.String()
}

func (st state) flat() []int {
	var out []int
	for _, sel := range st {
		out = append(append(out, sel...), -1)
	}
	return out
}

func (st state) clone() state {
	out := make(state, len(st))
	for i, sel := range st {
		out[i] = slices.Clone(sel)
	}
	return out
}

type evaluation struct {
	state    state
	changes  int
	vals     map[string]float64
	points   int
	score    float64
	feasible bool
	checks   []map[string]any
	// cost is the chaos the state adds to the build's price; unpriced names what it adds of unknown price
	cost     float64
	unpriced []string
	// priceOut marks a state over cost_budget, or of unknown price without allow_unknown_price
	priceOut bool
}

const (
	pointsAddedKey = "points_added"
	pointsTotalKey = "points_total"
	levelKey       = "level_needed"
	// ascKey counts the ascendancy points a result uses; PoB warns past ascMax
	ascKey = "ascendancy_points"
	ascMax = 8
	// pointsAlias is v1's name for points_added
	pointsAlias = "points"
)

func syntheticStat(s string) bool {
	return s == pointsAddedKey || s == pointsTotalKey || s == levelKey || s == buildCostKey
}

func (s *Server) optimize(args json.RawMessage) (any, *ToolError) {
	var in optimizeIn
	if e := decode(args, schemaOptimize, &in); e != nil {
		return nil, e
	}
	h, e := s.get(in.Handle)
	if e != nil {
		return nil, e
	}
	base, e := h.parseRev(h.cur)
	if e != nil {
		return nil, e
	}
	baseEnv, e := h.envAt(h.cur)
	if e != nil {
		return nil, e
	}
	if len(in.Slots) == 0 || len(in.Slots) > 6 {
		return nil, invalidArg("slots must hold 1-6 entries.", nil)
	}
	if in.Objective.Stat == "" {
		return nil, invalidArg("objective.stat is required.", nil)
	}
	if !knownStat(baseEnv, in.Objective.Stat) && in.Objective.Stat != buildCostKey {
		return nil, unknownStat(in.Objective.Stat)
	}
	if in.RankBy == rankBySavings || in.FitBudget {
		return nil, invalidArg("optimize takes savings as BuildCost, the build's price in chaos: objective {\"stat\":\"BuildCost\",\"goal\":\"minimize\"} with constraints on what to keep, or a {\"stat\":\"BuildCost\",\"op\":\"<=\",\"value\":<chaos>} constraint.", nil)
	}
	goal := in.Objective.Goal
	if goal == "" {
		goal = "maximize"
	}
	if goal != "maximize" && goal != "minimize" {
		return nil, invalidArg("objective.goal must be maximize or minimize.", []string{"maximize", "minimize"})
	}
	basePoints := pointsOf(baseEnv)
	for i := range in.Constraints {
		c := &in.Constraints[i]
		if c.Stat == pointsAlias {
			c.Stat = pointsAddedKey
		}
		if !syntheticStat(c.Stat) && !knownStat(baseEnv, c.Stat) {
			e := unknownStat(c.Stat)
			e.Hint = fmt.Sprintf("constraints[%d]: use a PoB output key, %q, %q, %q or %q.", i, pointsAddedKey, pointsTotalKey, levelKey, buildCostKey)
			return nil, e
		}
		if c.Op != ">=" && c.Op != "<=" && c.Op != "==" {
			return nil, invalidArg(fmt.Sprintf("constraints[%d].op must be >=, <= or ==.", i), []string{">=", "<=", "=="})
		}
	}
	budget := maxBudget
	if in.Budget != nil {
		budget = min(max(*in.Budget, 1), maxBudget)
	}
	maxMs := 20000
	if in.MaxMs != nil {
		maxMs = min(max(*in.MaxMs, 1), 60000)
	}
	seed := int64(1)
	if in.Seed != nil {
		seed = *in.Seed
	}
	top := 3
	if in.Top != nil {
		top = min(max(*in.Top, 1), 5)
	}

	slots, e := buildSlots(base, in.Slots)
	if e != nil {
		return nil, e
	}
	usesCost := in.Objective.Stat == buildCostKey || slices.ContainsFunc(in.Constraints, func(c constraintIn) bool { return c.Stat == buildCostKey })
	cb, e := s.newCostBudget(in.budgetIn, base, usesCost)
	if e != nil {
		return nil, e
	}
	// current is a constraint's or the objective's value on the build
	current := func(stat string) float64 {
		if stat == buildCostKey {
			return cb.baseTotal
		}
		return constraintCurrent(baseEnv, stat, basePoints)
	}
	// a start over the level's passive points is no answer: unless a constraint already bounds points, results must
	// fit them, which repairs the start
	var startNote map[string]any
	if over := basePoints.Used - basePoints.Available; over > 0 && slices.ContainsFunc(slots, func(sl optSlot) bool { return sl.kind == slotUpTo }) &&
		!slices.ContainsFunc(in.Constraints, func(c constraintIn) bool { return c.Stat == pointsTotalKey || c.Stat == pointsAddedKey }) {
		in.Constraints = append(in.Constraints, constraintIn{Stat: pointsTotalKey, Op: "<=", Value: float64(basePoints.Available)})
		startNote = map[string]any{"over_budget": over, "note": fmt.Sprintf("the build uses %d passive points more than its level grants; results must fit points_total <= %d", over, basePoints.Available)}
	}
	size := 1.0
	sweep := 0
	var perSlot []any
	for _, sl := range slots {
		n := sl.count()
		size *= n
		moves := sl.movesFor(sl.n)
		sweep += moves
		perSlot = append(perSlot, map[string]any{"slot": sl.name, "choices": math.Min(n, 1e18), "moves": moves, "how": sl.sizeMsg})
	}
	size = math.Min(size, 1e18)
	exhaustive := size <= float64(budget)
	if !exhaustive && sweep > maxBudget {
		e := errf(CodeSpaceTooLarge, "One sweep of single changes over these slots needs %d calculations; the budget cap is %d.", sweep, maxBudget)
		e.SizePerSlot = perSlot
		e.Suggestions = shrinkSuggestions(slots, sweep)
		return nil, e
	}

	objCur := current(in.Objective.Stat)
	sign := 1.0
	if goal == "minimize" {
		sign = -1
	}
	lambda := math.Max(10*math.Abs(objCur), 1)
	targets := make([]float64, len(in.Constraints))
	for i, c := range in.Constraints {
		cur := current(c.Stat)
		targets[i] = c.Value
		if c.Relative {
			targets[i] = c.Value * cur
		}
	}
	watch := slices.Clone(summaryKeys)
	watch = append(watch, in.Objective.Stat)
	for _, c := range in.Constraints {
		if !syntheticStat(c.Stat) {
			watch = append(watch, c.Stat)
		}
	}

	scoreVals := func(vals map[string]float64) (float64, bool, []map[string]any) {
		score, feasible := sign*vals[in.Objective.Stat], true
		checks := make([]map[string]any, 0, len(in.Constraints))
		for i, cn := range in.Constraints {
			actual := vals[cn.Stat]
			short := 0.0
			switch cn.Op {
			case ">=":
				short = math.Max(0, targets[i]-actual)
			case "<=":
				short = math.Max(0, actual-targets[i])
			case "==":
				short = math.Abs(actual - targets[i])
			}
			scale := math.Max(math.Abs(current(cn.Stat)), 1)
			score -= lambda * short / scale
			check := map[string]any{"stat": cn.Stat, "op": cn.Op, "value": round(targets[i]), "actual": round(actual), "ok": short == 0}
			if short > 0 {
				feasible = false
				check["short"] = round(short)
			}
			checks = append(checks, check)
		}
		// a result that spends more ascendancy points than PoB allows, and than the build already does, is no answer
		//pob:block:src/Modules/Build.lua:902,src/Modules/Build.lua:926
		if asc := vals[ascKey]; asc > ascMax && asc > float64(basePoints.AscUsed) {
			feasible = false
			checks = append(checks, map[string]any{"stat": ascKey, "op": "<=", "value": ascMax, "actual": asc, "ok": false, "short": asc - ascMax})
		}
		return score, feasible, checks
	}
	baseVals := map[string]float64{}
	for _, w := range watch {
		baseVals[w], _ = statValue(baseEnv, w)
	}
	baseVals[pointsAddedKey] = 0
	baseVals[pointsTotalKey] = float64(basePoints.Used)
	baseVals[levelKey] = float64(basePoints.LevelNeeded)
	baseVals[ascKey] = float64(basePoints.AscUsed)
	if cb != nil {
		baseVals[buildCostKey] = cb.baseTotal
	}
	curScore, curFeasible, _ := scoreVals(baseVals)
	// worse reports whether applying ev would leave a feasible current build worse off
	worse := func(ev *evaluation) bool {
		return curFeasible && (!ev.feasible || round(ev.score) < round(curScore))
	}

	start := time.Now()
	deadline := start.Add(time.Duration(maxMs) * time.Millisecond)
	skipped := ""
	memo := map[string]*evaluation{}
	used := 0
	stopped := ""
	baseNodes := slices.Clone(base.Build.PassiveNodes)
	v := specTreeVersion(base)
	build := func(st state) *pob.PathOfBuilding {
		c := cloneBuild(base)
		for i, sl := range slots {
			sl.apply(c, st[i])
		}
		return c
	}
	// built maps what a state builds to its evaluation: a removal's dependents, or an add on another add's path,
	// change nothing, so such states share one row, labelled by the state with the fewest picks
	built := map[string]*evaluation{}
	eval := func(st state) *evaluation {
		k := st.key()
		if ev, ok := memo[k]; ok {
			return ev
		}
		c := build(st)
		fp := builtKey(slots, st, c)
		if ev, ok := built[fp]; ok {
			if len(st.flat()) < len(ev.state.flat()) {
				ev.state = st.clone()
				ev.changes = changesOf(slots, st)
			}
			memo[k] = ev
			return ev
		}
		if used >= budget {
			stopped = "budget"
			return nil
		}
		if time.Now().After(deadline) {
			stopped = "time"
			return nil
		}
		used++
		ev := &evaluation{state: st.clone(), changes: changesOf(slots, st), vals: map[string]float64{}}
		ev.points = chargedPoints(v, c.Build.PassiveNodes) - chargedPoints(v, baseNodes)
		env := calcEnv(c)
		for _, w := range watch {
			ev.vals[w], _ = statValue(env, w)
		}
		ev.vals[pointsAddedKey] = float64(ev.points)
		ev.vals[pointsTotalKey] = float64(basePoints.Used + ev.points)
		p := pointsOf(env)
		ev.vals[levelKey] = float64(p.LevelNeeded)
		ev.vals[ascKey] = float64(p.AscUsed)
		if cb != nil {
			cb.price(ev, c)
		}
		ev.score, ev.feasible, ev.checks = scoreVals(ev.vals)
		if cb != nil {
			cb.priceOut(ev, lambda)
		}
		memo[k] = ev
		built[fp] = ev
		return ev
	}

	method := "exhaustive"
	proven := false
	if exhaustive {
		// a search the time limit would cut is no proof, and its enumeration order is no ranking: time the start and
		// take the local search when the whole space would not fit
		cur := make(state, len(slots))
		for i, sl := range slots {
			cur[i] = sl.start
		}
		t0 := time.Now()
		eval(cur)
		if need := time.Duration(size) * time.Since(t0); time.Now().Add(need).After(deadline) {
			exhaustive = false
			skipped = fmt.Sprintf("searching all %.0f states needs about %ds; max_ms allows %ds", size, int(need.Seconds())+1, maxMs/1000)
		}
	}
	if exhaustive {
		lists := make([][][]int, len(slots))
		for i, sl := range slots {
			lists[i] = sl.enumerate()
		}
		idx := make([]int, len(slots))
		for {
			st := make(state, len(slots))
			for i := range slots {
				st[i] = lists[i][idx[i]]
			}
			if eval(st) == nil {
				break
			}
			i := 0
			for i < len(idx) {
				idx[i]++
				if idx[i] < len(lists[i]) {
					break
				}
				idx[i] = 0
				i++
			}
			if i == len(idx) {
				break
			}
		}
		proven = stopped == ""
		if proven {
			stopped = "space"
		}
	} else {
		method = "coordinate_ascent"
		if coordinateAscent(slots, seed, eval) {
			stopped = "converged"
		}
	}

	evals := make([]*evaluation, 0, len(built))
	for _, ev := range built {
		evals = append(evals, ev)
	}
	sort.Slice(evals, func(i, j int) bool { return better(evals[i], evals[j]) })
	gainOf := func(ev *evaluation) float64 { return sign * (ev.vals[in.Objective.Stat] - objCur) }
	if cb != nil {
		evals = slices.DeleteFunc(evals, func(ev *evaluation) bool {
			if ev.priceOut {
				if len(ev.unpriced) > 0 && !cb.allowUnknown {
					cb.unpriced++
				} else {
					cb.over++
				}
			}
			return ev.priceOut
		})
		if cb.rankBy == rankByGainPerDiv {
			sort.SliceStable(evals, func(i, j int) bool {
				a, b := evals[i], evals[j]
				if a.feasible != b.feasible {
					return a.feasible
				}
				pa, pb := cb.evalPerDiv(a, gainOf(a)), cb.evalPerDiv(b, gainOf(b))
				if math.IsNaN(pa) != math.IsNaN(pb) {
					return !math.IsNaN(pa)
				}
				return pa > pb
			})
		}
	}
	must := []string{in.Objective.Stat}
	for _, c := range in.Constraints {
		must = append(must, c.Stat)
	}
	picksOf := func(ev *evaluation) []map[string]any {
		picks := make([]map[string]any, 0, len(slots))
		for si, sl := range slots {
			picks = append(picks, map[string]any{"slot": sl.name, "choice": sl.label(ev.state[si])})
		}
		return picks
	}
	var results []any
	var shown []*evaluation
	var resultRows []map[string]any
	add := func(ev *evaluation) map[string]any {
		row := map[string]any{"rank": len(shown) + 1, "feasible": ev.feasible, "score": round(ev.score), "choices": picksOf(ev),
			"delta": mapDelta(baseVals, ev.vals, watch, must, 6), "points_added": ev.points, "near_duplicates": 0}
		if len(ev.checks) > 0 {
			row["constraints"] = ev.checks
		}
		if worse(ev) {
			row["worse_than_current"] = true
		}
		if cb != nil {
			if len(ev.unpriced) > 0 {
				row["cost_unknown"] = ev.unpriced
			} else {
				row["cost"] = ev.cost
			}
			switch v := cb.evalPerDiv(ev, gainOf(ev)); {
			case math.IsNaN(v):
			case math.IsInf(v, 1):
				row["pays_for_itself"], row["gain_per_div_of"] = true, "gain"
			default:
				row["gain_per_div"], row["gain_per_div_of"] = round(v), "gain"
			}
		}
		st := ev.state
		row[changeKey] = change(func(c *pob.PathOfBuilding) {
			for i, sl := range slots {
				sl.apply(c, st[i])
			}
		})
		shown = append(shown, ev)
		resultRows = append(resultRows, row)
		results = append(results, row)
		return row
	}
	// dups are the near duplicates folded into a shown row, by its index; they fill the rows top asks for when fewer
	// distinct states exist
	type dup struct {
		ev *evaluation
		of int
	}
	var dups []dup
	for _, ev := range evals {
		if i := slices.IndexFunc(shown, func(o *evaluation) bool { return nearDuplicate(o, ev, watch) }); i >= 0 {
			resultRows[i]["near_duplicates"] = resultRows[i]["near_duplicates"].(int) + 1
			dups = append(dups, dup{ev, i})
			continue
		}
		if len(shown) < top {
			add(ev)
		}
	}
	for _, d := range dups {
		if len(shown) == top {
			break
		}
		// a pool that lists one choice twice builds the same state: no new row
		if slices.ContainsFunc(shown, func(o *evaluation) bool { return fmt.Sprint(picksOf(o)) == fmt.Sprint(picksOf(d.ev)) }) {
			continue
		}
		resultRows[d.of]["near_duplicates"] = resultRows[d.of]["near_duplicates"].(int) - 1
		add(d.ev)["near_duplicate_of"] = d.of + 1
	}
	newRealism(base, baseEnv, in.Objective.Stat, time.Now().Add(rankBudget)).annotate(resultRows, 1, false)
	for _, row := range resultRows {
		if row["near_duplicates"] == 0 {
			delete(row, "near_duplicates")
		}
	}
	objective := map[string]any{"stat": in.Objective.Stat, "goal": goal, "current": round(objCur)}
	if objCur == 0 {
		objective["note"] = "stat is 0 on this build"
	}
	res := map[string]any{
		"handle": h.id, "revision": h.cur, "objective": objective,
		"space":          map[string]any{"size": size, "method": method, "moves_per_sweep": sweep, "per_slot": perSlot},
		"proven_optimal": proven,
		"evaluations":    map[string]any{"used": used, "budget": budget},
		"points":         map[string]any{"used": basePoints.Used, "available": basePoints.Available},
		"elapsed_ms":     time.Since(start).Milliseconds(),
		"results":        nonNil(results),
	}
	if skipped != "" {
		res["exhaustive_skipped"] = skipped
	}
	res["stopped"] = stopped
	if in.Pareto {
		res["pareto"] = paretoFront(evals, picksOf)
	}
	if startNote != nil {
		res["start"] = startNote
	}
	if cb != nil {
		res["cost_budget"] = cb.summary()
	}
	hint := "Re-run with apply:true, or apply a result with edit_skills/edit_items/edit_tree using its choices."
	if !exhaustive && budget < sweep {
		hint = fmt.Sprintf("budget %d covers part of one sweep (%d moves); raise budget to >= %d for a full sweep.", budget, sweep, sweep)
	}
	if in.Apply {
		var best *evaluation
		for _, ev := range evals {
			if ev.feasible {
				best = ev
				break
			}
		}
		if best != nil && worse(best) {
			best = nil
			res["hint"] = "Nothing applied: no result beats the current build."
		}
		if best == nil {
			res["applied"] = nil
		} else {
			rev, e := s.commit(h, build(best.state))
			if e != nil {
				return nil, e
			}
			d := deltaResult(h, rev, baseEnv, h.revs[rev].env, false, []string{"applied the best feasible result"}, nil)
			res["applied"] = map[string]any{"revision": rev}
			res["revision"] = rev
			res["delta"] = d["delta"]
		}
	} else {
		if keep := keepHint(resultRows, baseEnv, in.Constraints); keep != "" {
			hint += " " + keep
		}
		res["hint"] = hint
	}
	limit := capOptimize
	if top > 3 {
		limit = capOptimizeMax
	}
	fitBudget(res, "results", limit, optimizeMoreHint(top))
	return res, nil
}

// keepLosses are the stats whose loss a top result should be held to with a constraint, by their change column.
var keepLosses = []struct{ stat, col string }{{"TotalEHP", "dEHP"}, {"TotemLife", "dTotemLife"}}

// keepHint names the constraint that keeps what the top result loses more than a fifth of, unless one bounds it already.
func keepHint(rows []map[string]any, baseEnv *calculator.Environment, cons []constraintIn) string {
	if len(rows) == 0 {
		return ""
	}
	d, _ := rows[0]["d"].(map[string]float64)
	var out []string
	for _, k := range keepLosses {
		base := baseEnv.Player.Output[k.stat]
		if base <= 0 || d[k.col] >= -0.2*base || slices.ContainsFunc(cons, func(c constraintIn) bool { return c.Stat == k.stat }) {
			continue
		}
		out = append(out, fmt.Sprintf("rank 1 loses %.0f%% of %s; keep it with constraints:[{\"stat\":\"%s\",\"op\":\">=\",\"value\":0.9,\"relative\":true}].", -d[k.col]/base*100, k.stat, k.stat))
	}
	return strings.Join(out, " ")
}

// optimizeMoreHint says how to see the results a cut optimize left out.
func optimizeMoreHint(top int) string {
	if top <= 3 {
		return "pass top 4 or 5 for a larger output budget, or apply:true to commit the best result and read it whole"
	}
	return "fewer constraints shorten each row; apply:true commits the best result to read it whole"
}

// nearDuplicate reports whether two evaluations make as many changes for the same points and agree on feasibility and
// every watched stat to within 0.1%.
func nearDuplicate(a, b *evaluation, watch []string) bool {
	if a.feasible != b.feasible || a.points != b.points || a.changes != b.changes {
		return false
	}
	for _, w := range watch {
		if x, y := a.vals[w], b.vals[w]; math.Abs(x-y) > 0.001*math.Max(math.Abs(x), math.Abs(y)) {
			return false
		}
	}
	return true
}

const paretoMax = 8

// paretoFront lists the feasible evaluations no other one beats on both MainDPS and TotalEHP, highest MainDPS first,
// at most paretoMax spread evenly along the front.
func paretoFront(evals []*evaluation, picks func(*evaluation) []map[string]any) []any {
	var front []*evaluation
	for _, a := range evals {
		if !a.feasible {
			continue
		}
		dominated := slices.ContainsFunc(evals, func(b *evaluation) bool {
			return b.feasible && b.vals[mainDPS] >= a.vals[mainDPS] && b.vals["TotalEHP"] >= a.vals["TotalEHP"] &&
				(b.vals[mainDPS] > a.vals[mainDPS] || b.vals["TotalEHP"] > a.vals["TotalEHP"])
		})
		if !dominated && !slices.ContainsFunc(front, func(f *evaluation) bool {
			return f.vals[mainDPS] == a.vals[mainDPS] && f.vals["TotalEHP"] == a.vals["TotalEHP"]
		}) {
			front = append(front, a)
		}
	}
	sort.SliceStable(front, func(i, j int) bool { return front[i].vals[mainDPS] > front[j].vals[mainDPS] })
	if len(front) > paretoMax {
		picked := make([]*evaluation, 0, paretoMax)
		for i := range paretoMax {
			picked = append(picked, front[i*(len(front)-1)/(paretoMax-1)])
		}
		front = picked
	}
	out := make([]any, 0, len(front))
	for _, ev := range front {
		out = append(out, map[string]any{mainDPS: round(ev.vals[mainDPS]), "TotalEHP": round(ev.vals["TotalEHP"]), "choices": picks(ev)})
	}
	return out
}

// builtKey identifies what a state builds: the other slots' picks, and for a nodes slot the allocated nodes and
// mastery effects it leaves.
func builtKey(slots []optSlot, st state, c *pob.PathOfBuilding) string {
	var b strings.Builder
	nodes := false
	for i, sl := range slots {
		if sl.kind == slotUpTo {
			nodes = true
			continue
		}
		fmt.Fprint(&b, st[i], "|")
	}
	if !nodes {
		return b.String()
	}
	ids := slices.Clone(c.Build.PassiveNodes)
	slices.Sort(ids)
	fmt.Fprint(&b, ids, "|")
	if spec := c.Tree.Specs[c.ActiveSpecIndex()]; spec.MasteryEffects != "" {
		for _, m := range pob.SpecMasterySelections(spec) {
			if _, ok := slices.BinarySearch(ids, m.Node); ok {
				fmt.Fprint(&b, m.Node, ":", m.Effect, ",")
			}
		}
	}
	return b.String()
}

// shrinkSuggestions names, per slot, the largest pool that brings one sweep within the budget cap.
func shrinkSuggestions(slots []optSlot, sweep int) []string {
	var out []string
	for _, sl := range slots {
		cur := sl.movesFor(sl.n)
		if cur <= 1 {
			continue
		}
		rest := sweep - cur
		fixed := sl.n - sl.addN
		if sl.kind != slotUpTo {
			fixed = 0
		}
		best := -1
		for n := sl.n - 1; n > max(sl.k, fixed); n-- {
			if rest+sl.movesFor(n) <= maxBudget {
				best = n
				break
			}
		}
		if best < 0 {
			out = append(out, fmt.Sprintf("%s: %s; lower k or split it into smaller optimize calls", sl.name, sl.moveMsg))
			continue
		}
		out = append(out, fmt.Sprintf("%s: %s; keep at most %d in its pool (%d moves), e.g. the top ones from rank_candidates", sl.name, sl.moveMsg, best, sl.movesFor(best)))
	}
	return out
}

// cloneBuild copies b deeply enough that the slot edits and a calculation on the copy leave b untouched.
func cloneBuild(b *pob.PathOfBuilding) *pob.PathOfBuilding {
	c := *b
	c.Build.PassiveNodes = slices.Clone(b.Build.PassiveNodes)
	c.Tree.Specs = slices.Clone(b.Tree.Specs)
	c.Items.Items = slices.Clone(b.Items.Items)
	c.Items.ItemSets = slices.Clone(b.Items.ItemSets)
	for i := range c.Items.ItemSets {
		c.Items.ItemSets[i].Slots = slices.Clone(c.Items.ItemSets[i].Slots)
	}
	c.Skills.SkillSets = slices.Clone(b.Skills.SkillSets)
	for i := range c.Skills.SkillSets {
		set := &c.Skills.SkillSets[i]
		set.Skills = slices.Clone(set.Skills)
		for j := range set.Skills {
			set.Skills[j].Gems = slices.Clone(set.Skills[j].Gems)
		}
	}
	c.Calcs.Inputs = slices.Clone(b.Calcs.Inputs)
	c.Calcs.Sections = slices.Clone(b.Calcs.Sections)
	c.Config.Inputs = slices.Clone(b.Config.Inputs)
	return &c
}

// chargedPoints counts the main-tree points a node list costs; a mastery costs its point whether or not an effect
// is picked yet, as it does once committed with one.
func chargedPoints(v pob.TreeVersion, nodes []int64) int {
	n := 0
	seen := make(map[int64]bool, len(nodes))
	for _, id := range nodes {
		if seen[id] {
			continue
		}
		seen[id] = true
		if t, ok := treeNodeByID(v, id); ok && t.Ascendancy == "" {
			n++
		}
	}
	return n
}

func constraintCurrent(env *calculator.Environment, stat string, p pointsInfo) float64 {
	switch stat {
	case pointsAddedKey:
		return 0
	case pointsTotalKey:
		return float64(p.Used)
	case levelKey:
		return float64(p.LevelNeeded)
	}
	v, _ := statValue(env, stat)
	return v
}

// changesOf counts the picks a state changes from the current build: another item, a swapped support, a node added or
// removed.
func changesOf(slots []optSlot, st state) int {
	n := 0
	for i, sl := range slots {
		for _, x := range st[i] {
			if !slices.Contains(sl.start, x) {
				n++
			}
		}
	}
	return n
}

// better orders results: feasible first, then score as the output rounds it, then fewer changes, then the state's
// choice order.
func better(a, b *evaluation) bool {
	if a.feasible != b.feasible {
		return a.feasible
	}
	if sa, sb := round(a.score), round(b.score); sa != sb {
		return sa > sb
	}
	if a.changes != b.changes {
		return a.changes < b.changes
	}
	return slices.Compare(a.state.flat(), b.state.flat()) < 0
}

// coordinateAscent keeps the best beamWidth states and, each sweep, tries every single move (another item, a
// support swap, a node add/remove/swap) of every slot for every state in the beam, and once those stop helping the
// free-and-refill moves, k-swaps and compound moves of the best state; it starts from the current build
// plus restarts seeded random states and stops when a sweep finds no better state or eval refuses (budget or time). It
// reports whether it converged.
func coordinateAscent(slots []optSlot, seed int64, eval func(state) *evaluation) bool {
	rng := rand.New(rand.NewPCG(uint64(seed), 0x9e3779b97f4a7c15)) //nolint:gosec // restarts must repeat for a seed
	cur := make(state, len(slots))
	for i, sl := range slots {
		cur[i] = sl.start
	}
	var beam []*evaluation
	has := func(list []*evaluation, ev *evaluation) bool {
		k := ev.state.key()
		return slices.ContainsFunc(list, func(b *evaluation) bool { return b.state.key() == k })
	}
	add := func(ev *evaluation) {
		if ev != nil && !has(beam, ev) {
			beam = append(beam, ev)
		}
	}
	add(eval(cur))
	for range restarts {
		st := make(state, len(slots))
		for i, sl := range slots {
			st[i] = sl.random(rng)
		}
		add(eval(st))
	}
	trim := func() {
		sort.SliceStable(beam, func(i, j int) bool { return better(beam[i], beam[j]) })
		if len(beam) > beamWidth {
			beam = beam[:beamWidth]
		}
	}
	trim()
	for len(beam) > 0 {
		best := beam[0]
		next := slices.Clone(beam)
		halted := false
	sweep:
		for _, st := range beam {
			for i, sl := range slots {
				for _, nb := range sl.neighbours(st.state[i]) {
					s := st.state.clone()
					s[i] = nb
					ev := eval(s)
					if ev == nil {
						halted = true
						break sweep
					}
					if !has(next, ev) {
						next = append(next, ev)
					}
				}
			}
		}
		beam = next
		trim()
		if halted {
			return false
		}
		if better(beam[0], best) {
			continue
		}
		// converged on single moves: from the best state try freeing points and refilling, then k-swaps, then
		// compounds of the best single moves, stopping at the first kind that improves
		kicked := false
		top := beam[0]
		for _, moves := range []func() []state{
			func() []state { return refillStates(slots, top.state) },
			func() []state { return kSwapStates(slots, top.state, eval) },
			func() []state { return compoundStates(slots, top.state, eval) },
		} {
			for _, st := range moves() {
				ev := eval(st)
				if ev == nil {
					return false
				}
				if better(ev, top) && !has(beam, ev) {
					beam = append(beam, ev)
					kicked = true
				}
			}
			if kicked {
				break
			}
		}
		if !kicked {
			return true
		}
		trim()
	}
	return true
}

func refillStates(slots []optSlot, cur state) []state {
	var out []state
	for i, sl := range slots {
		for _, nb := range sl.refills(cur[i]) {
			st := cur.clone()
			st[i] = nb
			out = append(out, st)
		}
	}
	return out
}

// rankedBy orders xs by the score eval gives the state with with(x) in place of slot i, best first; a state eval
// refuses ranks last.
func rankedBy(cur state, i int, xs []int, with func(x int) []int, eval func(state) *evaluation) []int {
	score := make(map[int]float64, len(xs))
	for _, x := range xs {
		st := cur.clone()
		st[i] = with(x)
		score[x] = math.Inf(-1)
		if ev := eval(st); ev != nil {
			score[x] = ev.score
		}
	}
	out := slices.Clone(xs)
	slices.SortStableFunc(out, func(a, b int) int {
		switch {
		case score[a] > score[b]:
			return -1
		case score[a] < score[b]:
			return 1
		}
		return 0
	})
	return out
}

// kSwapStates are the k-for-j swaps of each add slot, k <= 3 and j <= k: the k removals that cost least and the j adds
// that gain most on their own, taken together; at full points no chain of single moves reaches a path longer than one
// point.
func kSwapStates(slots []optSlot, cur state, eval func(state) *evaluation) []state {
	const topRemovals, topAdds = 4, 3
	var out []state
	for i, sl := range slots {
		if sl.kind != slotUpTo {
			continue
		}
		sel := cur[i]
		plus := func(x int) []int {
			s := append(slices.Clone(sel), x)
			slices.Sort(s)
			return s
		}
		var rems, adds []int
		picked := 0
		for x := range sl.n {
			switch {
			case slices.Contains(sel, x):
				if x < sl.addN {
					picked++
				}
			case x >= sl.addN:
				rems = append(rems, x)
			case !sl.clashes(sel, x):
				adds = append(adds, x)
			}
		}
		rems = rankedBy(cur, i, rems, plus, eval)[:min(topRemovals, len(rems))]
		adds = rankedBy(cur, i, adds, plus, eval)[:min(topAdds, len(adds))]
		for k := 2; k <= min(3, len(rems)); k++ {
			for _, rc := range combinations(len(rems), k) {
				for j := 1; j <= min(k, sl.k-picked, len(adds)); j++ {
					for _, ac := range combinations(len(adds), j) {
						s := slices.Clone(sel)
						for _, r := range rc {
							s = append(s, rems[r])
						}
						for _, a := range ac {
							s = append(s, adds[a])
						}
						slices.Sort(s)
						if sl.compatible(s) {
							st := cur.clone()
							st[i] = s
							out = append(out, st)
						}
					}
				}
			}
		}
	}
	return out
}

// compoundStates combine two or three of the best single moves from cur, across slots or as disjoint changes of one
// slot: moves that each fall short alone can together beat it.
func compoundStates(slots []optSlot, cur state, eval func(state) *evaluation) []state {
	const topMoves = 4
	type move struct {
		slot      int
		add, drop []int
		ev        *evaluation
	}
	var moves []move
	for i, sl := range slots {
		for _, nb := range sl.neighbours(cur[i]) {
			st := cur.clone()
			st[i] = nb
			ev := eval(st)
			if ev == nil {
				continue
			}
			m := move{slot: i, ev: ev}
			for _, x := range nb {
				if !slices.Contains(cur[i], x) {
					m.add = append(m.add, x)
				}
			}
			for _, x := range cur[i] {
				if !slices.Contains(nb, x) {
					m.drop = append(m.drop, x)
				}
			}
			moves = append(moves, m)
		}
	}
	slices.SortStableFunc(moves, func(a, b move) int {
		if better(a.ev, b.ev) {
			return -1
		}
		if better(b.ev, a.ev) {
			return 1
		}
		return 0
	})
	moves = moves[:min(topMoves, len(moves))]
	var out []state
	seen := map[string]bool{}
	for k := 2; k <= min(3, len(moves)); k++ {
	combo:
		for _, c := range combinations(len(moves), k) {
			st := cur.clone()
			touched := map[int][]int{}
			for _, mi := range c {
				m := moves[mi]
				for _, x := range slices.Concat(m.add, m.drop) {
					if slices.Contains(touched[m.slot], x) {
						continue combo
					}
				}
				touched[m.slot] = slices.Concat(touched[m.slot], m.add, m.drop)
				sel := slices.DeleteFunc(st[m.slot], func(x int) bool { return slices.Contains(m.drop, x) })
				sel = append(sel, m.add...)
				slices.Sort(sel)
				st[m.slot] = sel
			}
			for i, sl := range slots {
				if !sl.valid(st[i]) {
					continue combo
				}
			}
			if k := st.key(); !seen[k] {
				seen[k] = true
				out = append(out, st)
			}
		}
	}
	return out
}

func mapDelta(a, b map[string]float64, keys, must []string, limit int) []deltaRow {
	var rows []deltaRow
	seen := map[string]bool{}
	mustSet := map[string]bool{}
	for _, k := range must {
		mustSet[k] = true
	}
	for _, k := range slices.Concat(must, keys) {
		if seen[k] {
			continue
		}
		seen[k] = true
		va, vb := a[k], b[k]
		if !mustSet[k] && (round(va) == round(vb) || (!alwaysShown(k) && math.Abs(vb-va) < 0.005*math.Abs(va))) {
			continue
		}
		rows = append(rows, deltaRow{Stat: k, Before: round(va), After: round(vb), Pct: pct(va, vb)})
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows
}

func buildSlots(b *pob.PathOfBuilding, in []optSlotIn) ([]optSlot, *ToolError) {
	var out []optSlot
	itemSlots := map[string]bool{}
	for i, sin := range in {
		var sl optSlot
		var e *ToolError
		switch sin.Kind {
		case "item":
			sl, e = itemSlot(b, sin)
			if e == nil {
				if itemSlots[sl.name] {
					e = invalidArg(fmt.Sprintf("Item slot %s appears twice.", sl.name), nil)
				}
				itemSlots[sl.name] = true
			}
		case "supports":
			sl, e = supportsSlot(b, sin)
		case "nodes":
			sl, e = nodesSlot(b, sin)
		case "gem":
			sl, e = gemSlot(b, sin)
		default:
			e = invalidArg(fmt.Sprintf("Unknown slot kind %q.", sin.Kind), []string{"item", "supports", "nodes", "gem"})
		}
		if e != nil {
			return nil, e.atSlot(i)
		}
		if sl.size() <= 1 && sl.n == 0 {
			return nil, invalidArg(fmt.Sprintf("slots[%d] has no choices.", i), nil).atSlot(i)
		}
		out = append(out, sl)
	}
	return out, nil
}

func itemSlot(b *pob.PathOfBuilding, in optSlotIn) (optSlot, *ToolError) {
	if in.Slot == "" {
		e := invalidArg("An item slot needs slot.", nil)
		e.Valid = slotNames()
		return optSlot{}, e
	}
	slot, e := resolveSlot(in.Slot)
	if e != nil {
		return optSlot{}, e
	}
	type itemChoice struct {
		label string
		text  string
	}
	var choices []itemChoice
	if in.KeepCurrent == nil || *in.KeepCurrent {
		cur := slotItemName(b, slot)
		if cur == "" {
			cur = "(empty)"
		}
		choices = append(choices, itemChoice{label: "(current) " + cur})
	}
	addText := func(label, text string) *ToolError {
		item := calculator.ParseItem(text)
		if item == nil || item.Base == nil {
			e := errf(CodeParseError, "An item text has no known base type.")
			if item != nil {
				e.Line = item.BaseName
			}
			return e
		}
		if !calculator.IsItemValidForSlot(b, item, slot) {
			e := errf(CodeInvalidSlot, "%s cannot go in %s.", item.Name, slot)
			e.ValidSlots = validSlotsFor(b, item)
			return e
		}
		choices = append(choices, itemChoice{label: label, text: text})
		return nil
	}
	for _, u := range in.Uniques {
		name, e := resolveUnique(u)
		if e != nil {
			return optSlot{}, e
		}
		text, _ := calculator.ItemDBText(calculator.ItemDBUnique, name)
		if e := addText(name, text); e != nil {
			return optSlot{}, e
		}
	}
	for i, t := range in.Texts {
		text := strings.ReplaceAll(t, "\r\n", "\n")
		label := fmt.Sprintf("text %d: %s", i+1, calculator.ParseItem(text).Name)
		if e := addText(label, text); e != nil {
			return optSlot{}, e
		}
	}
	return optSlot{
		name: slot, kind: slotOne, n: len(choices), start: []int{0},
		apply: func(c *pob.PathOfBuilding, sel []int) {
			if t := choices[sel[0]].text; t != "" {
				calculator.EquipItem(c, t, slot)
			}
		},
		label:   func(sel []int) any { return choices[sel[0]].label },
		sizeMsg: fmt.Sprintf("%d items", len(choices)),
		moveMsg: fmt.Sprintf("%d items", len(choices)),
	}, nil
}

func combinations(n, k int) [][]int {
	var out [][]int
	var rec func(start int, cur []int)
	rec = func(start int, cur []int) {
		if len(cur) == k {
			out = append(out, slices.Clone(cur))
			return
		}
		for i := start; i < n; i++ {
			rec(i+1, append(cur, i))
		}
	}
	rec(0, nil)
	return out
}

func binom(n, k int) float64 {
	if k < 0 || k > n {
		return 0
	}
	r := 1.0
	for i := 1; i <= k; i++ {
		r = r * float64(n-k+i) / float64(i)
	}
	return r
}

func supportsSlot(b *pob.PathOfBuilding, in optSlotIn) (optSlot, *ToolError) {
	group, e := mainGroup(b, in.Group)
	if e != nil {
		return optSlot{}, e
	}
	g := (*groupsOf(b))[group-1]
	var replaceIdx []int
	if len(in.Replace) == 0 {
		for i, gem := range g.Gems {
			if gemIsSupport(gem) {
				replaceIdx = append(replaceIdx, i)
			}
		}
	} else {
		for _, r := range in.Replace {
			i := slices.IndexFunc(g.Gems, func(x pob.Gem) bool { return gemMatches(x, r) })
			if i < 0 {
				return optSlot{}, gemNotInGroup(&g, r, group)
			}
			replaceIdx = append(replaceIdx, i)
		}
	}
	type pooled struct {
		name string
		gem  pob.Gem
		at   int
	}
	var pool []pooled
	keep := in.KeepCurrent == nil || *in.KeepCurrent
	if keep {
		for _, i := range replaceIdx {
			pool = append(pool, pooled{g.Gems[i].NameSpec, g.Gems[i], i})
		}
	}
	for _, c := range in.Candidates {
		ge, e := resolveGem(c)
		if e != nil {
			return optSlot{}, e
		}
		if !ge.Support {
			return optSlot{}, errf(CodeInvalidValue, "%s is not a support gem.", ge.Name)
		}
		if slices.ContainsFunc(pool, func(p pooled) bool { return p.name == ge.Name }) {
			continue
		}
		pool = append(pool, pooled{ge.Name, addedGem(b, ge, nil, nil), -1})
	}
	k := len(replaceIdx)
	if in.K != nil {
		k = *in.K
	}
	if k < 0 || k > len(pool) {
		return optSlot{}, invalidArg(fmt.Sprintf("k=%d but only %d supports are available to choose from.", k, len(pool)), nil)
	}
	n := len(pool)
	return optSlot{
		name: fmt.Sprintf("supports g%d", group), kind: slotExact, n: n, k: k, start: seq(k),
		// a picked candidate takes the place of a replaced gem that was not picked, keeping its quality as edit_skills set_gems does
		apply: func(c *pob.PathOfBuilding, sel []int) {
			set := slices.Clone(g.Gems)
			var freed []int
			for _, i := range replaceIdx {
				if !slices.ContainsFunc(sel, func(p int) bool { return pool[p].at == i }) {
					freed = append(freed, i)
				}
			}
			for _, p := range sel {
				switch {
				case pool[p].at >= 0:
				case len(freed) > 0:
					set[freed[0]] = keepSlot(pool[p].gem, g.Gems[freed[0]])
					freed = freed[1:]
				default:
					set = append(set, pool[p].gem)
				}
			}
			slices.Sort(freed)
			for _, i := range slices.Backward(freed) {
				set = slices.Delete(set, i, i+1)
			}
			(*groupsOf(c))[group-1].Gems = set
		},
		label: func(sel []int) any {
			names := make([]string, len(sel))
			for i, p := range sel {
				names[i] = pool[p].name
			}
			return names
		},
		sizeMsg: fmt.Sprintf("%d candidates choose %d = %.0f", n, k, binom(n, k)),
		moveMsg: fmt.Sprintf("%d candidates choose %d needs %d swaps per sweep", n, k, k*(n-k)),
	}, nil
}

func seq(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

func nodesSlot(b *pob.PathOfBuilding, in optSlotIn) (optSlot, *ToolError) {
	v := specTreeVersion(b)
	plan := planPaths(b, v)
	// a mastery is one candidate per effect it can still take, so a pick always costs and grants what edit_tree would
	type addNode struct {
		node   treeNode
		effect int64
		label  string
		group  int
		// switch marks another effect for a mastery the build has allocated
		switched bool
	}
	var adds []addNode
	var removes []treeNode
	nodesListed := 0
	for _, c := range in.Candidates {
		n, e := resolveNode(b, v, json.RawMessage(strconv.Quote(c)), &plan)
		if e != nil {
			return optSlot{}, e
		}
		if slices.ContainsFunc(adds, func(x addNode) bool { return x.node.ID == n.ID }) {
			continue
		}
		nodesListed++
		if n.Type != "mastery" {
			adds = append(adds, addNode{node: n, label: nodeLabel(n), group: -1})
			continue
		}
		allocated, current := slices.Contains(b.Build.PassiveNodes, n.ID), ""
		for _, o := range calculator.MasteryOptions(b, n.ID) {
			if o.Selected {
				current = o.Label
			}
		}
		for _, o := range calculator.MasteryOptions(b, n.ID) {
			if !o.Selected && effectHasMods(o.Label) {
				a := addNode{node: n, effect: o.Effect, label: nodeLabel(n) + ": " + o.Label, group: nodesListed, switched: allocated}
				if allocated && current != "" {
					a.label += " (replaces " + current + ")"
				}
				adds = append(adds, a)
			}
		}
	}
	for _, c := range in.Remove {
		n, e := resolveAllocated(b, v, c)
		if e != nil {
			return optSlot{}, e
		}
		if !slices.ContainsFunc(removes, func(x treeNode) bool { return x.ID == n.ID }) {
			removes = append(removes, n)
		}
	}
	k := nodesListed
	if in.K != nil {
		k = *in.K
	}
	if k < 0 || k > nodesListed {
		return optSlot{}, invalidArg(fmt.Sprintf("k=%d but only %d nodes are listed.", k, nodesListed), nil)
	}
	withPath := in.Path == nil || *in.Path
	// with paths, a removal at the end of a dead end is offered with the path nodes that lead only to it too, which
	// frees the points a node further away needs
	branches := map[int][]int64{}
	if withPath {
		var seen [][]int64
		for _, r := range slices.Clone(removes) {
			br := leafBranch(b, v, r.ID)
			if br == nil || slices.ContainsFunc(seen, func(x []int64) bool { return sameNodes(x, br) }) {
				continue
			}
			seen = append(seen, br)
			branches[len(removes)] = br
			removes = append(removes, r)
		}
	}
	group := make([]int, 0, len(adds)+len(removes))
	pool := make([]treeNode, 0, len(adds)+len(removes))
	for _, a := range adds {
		pool = append(pool, a.node)
		group = append(group, a.group)
	}
	for _, r := range removes {
		pool = append(pool, r)
		group = append(group, -1)
	}
	sl := optSlot{name: "nodes", kind: slotUpTo, n: len(pool), k: k, addN: len(adds), group: group, start: []int{}}
	sl.sizeMsg = fmt.Sprintf("subsets of up to %d of %d nodes to add (one effect per mastery), any of %d to remove (%d of them with the dead-end path that leads only to them) = %.0f", k, len(adds), len(removes), len(branches), math.Min(sl.count(), 1e18))
	sl.moveMsg = fmt.Sprintf("%d nodes to add and %d to remove need %d moves per sweep", len(adds), len(removes), sl.movesFor(sl.n))
	sl.apply = func(c *pob.PathOfBuilding, sel []int) {
		for _, p := range sel {
			if p < len(adds) {
				continue
			}
			id := pool[p].ID
			if !slices.Contains(c.Build.PassiveNodes, id) {
				continue
			}
			cut := append([]int64{id}, data.TreeVersions[v].CalculateDependentNodes(slices.Clone(c.Build.PassiveNodes), id, classRoots(c, v))...)
			if _, ok := branches[p-len(adds)]; ok {
				if br := leafBranch(c, v, id); br != nil {
					cut = br
				}
			}
			c.DeallocateNodes(cut)
		}
		for _, p := range sel {
			if p >= len(adds) {
				continue
			}
			n := pool[p]
			if !slices.Contains(c.Build.PassiveNodes, n.ID) {
				if !withPath {
					c.AllocateNodes([]int64{n.ID})
				} else if path := planPaths(c, v).path(n.ID); len(path) > 0 {
					c.AllocateNodes(path)
				}
			}
			if adds[p].effect != 0 && slices.Contains(c.Build.PassiveNodes, n.ID) {
				c.SetMasteryEffect(n.ID, adds[p].effect)
			}
		}
	}
	sl.label = func(sel []int) any {
		out := map[string]any{"add": []string{}}
		var add, sw, rem []string
		for _, p := range sel {
			switch {
			case p >= len(adds):
				if br, ok := branches[p-len(adds)]; ok {
					rem = append(rem, fmt.Sprintf("%s with the path that leads only to it (%d points)", nodeLabel(pool[p]), chargedPoints(v, br)))
				} else {
					rem = append(rem, nodeLabel(pool[p]))
				}
			case adds[p].switched:
				sw = append(sw, adds[p].label)
			default:
				add = append(add, adds[p].label)
			}
		}
		out["add"] = nonNil(add)
		if len(sw) > 0 {
			out["switch"] = sw
		}
		if len(removes) > 0 {
			out["remove"] = nonNil(rem)
		}
		if len(rem) > 0 {
			out["removed_nodes"] = removedNodes(b, v, sl, sel)
		}
		return out
	}
	return sl, nil
}

// removedNodes names every node applying sel to b takes away: the nodes asked for, their dependents and dead-end paths.
func removedNodes(b *pob.PathOfBuilding, v pob.TreeVersion, sl optSlot, sel []int) []string {
	c := cloneBuild(b)
	sl.apply(c, sel)
	var out []string
	for _, id := range b.Build.PassiveNodes {
		if slices.Contains(c.Build.PassiveNodes, id) {
			continue
		}
		if n, ok := treeNodeByID(v, id); ok {
			out = append(out, nodeLabel(n))
		} else {
			out = append(out, strconv.FormatInt(id, 10))
		}
	}
	return out
}

// resolveAllocated resolves a node to remove; a name several nodes share means the allocated ones.
func resolveAllocated(b *pob.PathOfBuilding, v pob.TreeVersion, arg string) (treeNode, *ToolError) {
	n, e := resolveNode(b, v, json.RawMessage(strconv.Quote(arg)), nil)
	if e != nil {
		return treeNode{}, e
	}
	if slices.Contains(b.Build.PassiveNodes, n.ID) {
		return n, nil
	}
	var same []treeNode
	if _, err := strconv.ParseInt(arg, 10, 64); err != nil {
		for _, id := range b.Build.PassiveNodes {
			if t, ok := treeNodeByID(v, id); ok && norm(t.Name) == norm(n.Name) {
				same = append(same, t)
			}
		}
	}
	switch len(same) {
	case 0:
		return treeNode{}, errf(CodeInvalidValue, "%s (id %d) is not allocated, so it cannot be removed.", n.Name, n.ID)
	case 1:
		return same[0], nil
	}
	e = errf(CodeAmbiguous, "%d allocated nodes are named %q.", len(same), n.Name)
	for _, t := range same {
		e.Candidates = append(e.Candidates, map[string]any{"id": t.ID, "name": t.Name, "type": t.Type})
	}
	e.Hint = "Retry with the id of the one to remove."
	return treeNode{}, e
}

// effectHasMods reports whether a mastery effect's lines (joined by " / ") parse to a modifier; PoB's power report
// leaves out effects with none, as they do nothing.
//
//pob:block:src/Classes/CalcsTab.lua:665-675
func effectHasMods(label string) bool {
	for _, line := range strings.Split(label, " / ") {
		for _, part := range strings.Split(line, "\n") {
			if e := calculator.ParseMod(part, false); e != nil && e.Extra == "" && len(e.ModList) > 0 {
				return true
			}
		}
	}
	return false
}

// nodeLabel names a node with its id; names such as "Life" or "Elemental Mastery" are shared by many nodes.
func nodeLabel(n treeNode) string {
	return fmt.Sprintf("%s (%d)", n.Name, n.ID)
}

// maxBranch is the most points a leaf-branch removal frees.
const maxBranch = 5

// leafBranch is the allocated dead end that ends at id: id, its dependents and the path nodes that lead only to them,
// up to maxBranch points; nil when no path node leads only to id.
func leafBranch(b *pob.PathOfBuilding, v pob.TreeVersion, id int64) []int64 {
	tv := data.TreeVersions[v]
	if tv == nil {
		return nil
	}
	roots := classRoots(b, v)
	active := map[int64]bool{}
	for _, n := range b.Build.PassiveNodes {
		active[n] = true
	}
	branch := func(top int64) []int64 {
		return append([]int64{top}, tv.CalculateDependentNodes(slices.Clone(b.Build.PassiveNodes), top, roots)...)
	}
	top, cut := id, branch(id)
	for {
		t, ok := treeNodeByID(v, top)
		if !ok {
			break
		}
		var out []int64
		for _, o := range slices.Concat(t.Node.Out, t.Node.In) {
			n, err := strconv.ParseInt(o, 10, 64)
			if err != nil || !active[n] || slices.Contains(cut, n) {
				continue
			}
			if p, ok := treeNodeByID(v, n); ok && p.Type != "mastery" {
				out = append(out, n)
			}
		}
		if len(out) != 1 || slices.Contains(roots, out[0]) {
			break
		}
		p, _ := treeNodeByID(v, out[0])
		next := branch(out[0])
		if p.Ascendancy != "" || chargedPoints(v, next) > maxBranch {
			break
		}
		top, cut = out[0], next
	}
	if top == id {
		return nil
	}
	return cut
}

// sameNodes reports whether two node lists hold the same nodes.
func sameNodes(a, b []int64) bool {
	return len(a) == len(b) && !slices.ContainsFunc(a, func(x int64) bool { return !slices.Contains(b, x) })
}
