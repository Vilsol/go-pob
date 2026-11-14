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
	exhaustiveLimit = 500
	maxBudget       = 2000
	beamWidth       = 3
	restarts        = 3
	capOptimize     = 3 * 1024
	capOptimizeMax  = 5 * 1024
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

func (sl optSlot) compatible(sel []int) bool {
	for i, x := range sel {
		if sl.clashes(sel[:i], x) {
			return false
		}
	}
	return true
}

// neighbours are the selections one move away: another choice, a swap, an add or a removal.
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
		return adds + sl.n - sl.addN
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
	vals     map[string]float64
	points   int
	score    float64
	feasible bool
	checks   []map[string]any
}

const (
	pointsAddedKey = "points_added"
	pointsTotalKey = "points_total"
	levelKey       = "level_needed"
	// pointsAlias is v1's name for points_added
	pointsAlias = "points"
)

func syntheticStat(s string) bool {
	return s == pointsAddedKey || s == pointsTotalKey || s == levelKey
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
	if !knownStat(baseEnv, in.Objective.Stat) {
		return nil, unknownStat(in.Objective.Stat)
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
			e.Hint = fmt.Sprintf("constraints[%d]: use a PoB output key, %q, %q or %q.", i, pointsAddedKey, pointsTotalKey, levelKey)
			return nil, e
		}
		if c.Op != ">=" && c.Op != "<=" && c.Op != "==" {
			return nil, invalidArg(fmt.Sprintf("constraints[%d].op must be >=, <= or ==.", i), []string{">=", "<=", "=="})
		}
	}
	budget := 300
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
	size := 1.0
	sweep := 0
	var perSlot []any
	for _, sl := range slots {
		size *= sl.size()
		moves := sl.movesFor(sl.n)
		sweep += moves
		perSlot = append(perSlot, map[string]any{"slot": sl.name, "choices": math.Min(sl.size(), 1e18), "moves": moves, "how": sl.sizeMsg})
	}
	size = math.Min(size, 1e18)
	exhaustive := size <= exhaustiveLimit && size <= float64(budget)
	if !exhaustive && sweep > maxBudget {
		e := errf(CodeSpaceTooLarge, "One sweep of single changes over these slots needs %d calculations; the budget cap is %d.", sweep, maxBudget)
		e.SizePerSlot = perSlot
		e.Suggestions = shrinkSuggestions(slots, sweep)
		return nil, e
	}

	objCur, _ := statValue(baseEnv, in.Objective.Stat)
	sign := 1.0
	if goal == "minimize" {
		sign = -1
	}
	lambda := math.Max(10*math.Abs(objCur), 1)
	targets := make([]float64, len(in.Constraints))
	for i, c := range in.Constraints {
		cur := constraintCurrent(baseEnv, c.Stat, basePoints)
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

	start := time.Now()
	deadline := start.Add(time.Duration(maxMs) * time.Millisecond)
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
	eval := func(st state) *evaluation {
		k := st.key()
		if ev, ok := memo[k]; ok {
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
		c := build(st)
		ev := &evaluation{state: st.clone(), vals: map[string]float64{}}
		ev.points = chargedPoints(v, c.Build.PassiveNodes) - chargedPoints(v, baseNodes)
		env := calcEnv(c)
		for _, w := range watch {
			ev.vals[w], _ = statValue(env, w)
		}
		ev.vals[pointsAddedKey] = float64(ev.points)
		ev.vals[pointsTotalKey] = float64(basePoints.Used + ev.points)
		ev.vals[levelKey] = float64(pointsOf(env).LevelNeeded)
		ev.score = sign * ev.vals[in.Objective.Stat]
		ev.feasible = true
		for i, cn := range in.Constraints {
			actual := ev.vals[cn.Stat]
			short := 0.0
			switch cn.Op {
			case ">=":
				short = math.Max(0, targets[i]-actual)
			case "<=":
				short = math.Max(0, actual-targets[i])
			case "==":
				short = math.Abs(actual - targets[i])
			}
			scale := math.Max(math.Abs(constraintCurrent(baseEnv, cn.Stat, basePoints)), 1)
			ev.score -= lambda * short / scale
			check := map[string]any{"stat": cn.Stat, "op": cn.Op, "value": round(targets[i]), "actual": round(actual), "ok": short == 0}
			if short > 0 {
				ev.feasible = false
				check["short"] = round(short)
			}
			ev.checks = append(ev.checks, check)
		}
		memo[k] = ev
		return ev
	}

	method := "exhaustive"
	proven := false
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
	} else {
		method = "coordinate_ascent"
		coordinateAscent(slots, seed, eval)
	}

	evals := make([]*evaluation, 0, len(memo))
	for _, ev := range memo {
		evals = append(evals, ev)
	}
	sort.Slice(evals, func(i, j int) bool { return better(evals[i], evals[j]) })
	baseVals := map[string]float64{}
	for _, w := range watch {
		baseVals[w], _ = statValue(baseEnv, w)
	}
	baseVals[pointsAddedKey] = 0
	baseVals[pointsTotalKey] = float64(basePoints.Used)
	baseVals[levelKey] = float64(basePoints.LevelNeeded)
	must := []string{in.Objective.Stat}
	for _, c := range in.Constraints {
		must = append(must, c.Stat)
	}
	var results []any
	for i, ev := range evals[:min(top, len(evals))] {
		var picks []map[string]any
		for si, sl := range slots {
			picks = append(picks, map[string]any{"slot": sl.name, "choice": sl.label(ev.state[si])})
		}
		row := map[string]any{"rank": i + 1, "feasible": ev.feasible, "score": round(ev.score), "choices": picks,
			"delta": mapDelta(baseVals, ev.vals, watch, must, 6), "points_added": ev.points}
		if len(ev.checks) > 0 {
			row["constraints"] = ev.checks
		}
		results = append(results, row)
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
	if stopped != "" {
		res["stopped"] = stopped
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
		res["hint"] = hint
	}
	limit := capOptimize
	if top > 3 {
		limit = capOptimizeMax
	}
	fitBudget(res, "results", limit, "lower top")
	return res, nil
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

// better orders results: feasible first, then score, then the state's choice order.
func better(a, b *evaluation) bool {
	if a.feasible != b.feasible {
		return a.feasible
	}
	if a.score != b.score {
		return a.score > b.score
	}
	return slices.Compare(a.state.flat(), b.state.flat()) < 0
}

// coordinateAscent keeps the best beamWidth states and, each sweep, tries every single move (another item, a
// support swap, a node add/remove/swap) of every slot for every state in the beam; it starts from the current build
// plus restarts seeded random states and stops when a sweep finds no better state or eval refuses (budget or time).
func coordinateAscent(slots []optSlot, seed int64, eval func(state) *evaluation) {
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
		if halted || !better(beam[0], best) {
			return
		}
	}
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
		default:
			e = invalidArg(fmt.Sprintf("Unknown slot kind %q.", sin.Kind), []string{"item", "supports", "nodes"})
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
	}
	var pool []pooled
	keep := in.KeepCurrent == nil || *in.KeepCurrent
	if keep {
		for _, i := range replaceIdx {
			pool = append(pool, pooled{g.Gems[i].NameSpec, g.Gems[i]})
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
		pool = append(pool, pooled{ge.Name, newGem(ge, 20, 20)})
	}
	k := len(replaceIdx)
	if in.K != nil {
		k = *in.K
	}
	if k < 0 || k > len(pool) {
		return optSlot{}, invalidArg(fmt.Sprintf("k=%d but only %d supports are available to choose from.", k, len(pool)), nil)
	}
	var fixed []pob.Gem
	for i, gem := range g.Gems {
		if !slices.Contains(replaceIdx, i) {
			fixed = append(fixed, gem)
		}
	}
	n := len(pool)
	return optSlot{
		name: fmt.Sprintf("supports g%d", group), kind: slotExact, n: n, k: k, start: seq(k),
		apply: func(c *pob.PathOfBuilding, sel []int) {
			set := slices.Clone(fixed)
			for _, p := range sel {
				set = append(set, pool[p].gem)
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
			adds = append(adds, addNode{node: n, label: n.Name, group: -1})
			continue
		}
		for _, o := range calculator.MasteryOptions(b, n.ID) {
			if !o.Selected {
				adds = append(adds, addNode{node: n, effect: o.Effect, label: n.Name + ": " + o.Label, group: nodesListed})
			}
		}
	}
	for _, c := range in.Remove {
		n, e := resolveNode(b, v, json.RawMessage(strconv.Quote(c)), nil)
		if e != nil {
			return optSlot{}, e
		}
		if !slices.Contains(b.Build.PassiveNodes, n.ID) {
			return optSlot{}, errf(CodeInvalidValue, "%s (id %d) is not allocated, so it cannot be removed.", n.Name, n.ID)
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
	sl.sizeMsg = fmt.Sprintf("subsets of up to %d of %d nodes to add, any of %d to remove = %.0f", k, len(adds), len(removes), math.Min(sl.size(), 1e18))
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
		var add, rem []string
		for _, p := range sel {
			if p < len(adds) {
				add = append(add, adds[p].label)
			} else {
				rem = append(rem, pool[p].Name)
			}
		}
		out["add"] = nonNil(add)
		if len(removes) > 0 {
			out["remove"] = nonNil(rem)
		}
		return out
	}
	return sl, nil
}
