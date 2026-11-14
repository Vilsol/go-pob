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
	"github.com/Vilsol/go-pob/pob"
)

const (
	exhaustiveLimit = 500
	spaceLimit      = 1_000_000
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

// choice is one option for one slot: a label for the output and the edit that makes it.
type choice struct {
	label any
	apply func(c *pob.PathOfBuilding)
}

type optSlot struct {
	name    string
	choices []choice
	current int
	sizeMsg string
	// tooMany is the choice count of a slot too large to enumerate
	tooMany float64
}

func (sl optSlot) count() float64 {
	if sl.tooMany > 0 {
		return sl.tooMany
	}
	return float64(len(sl.choices))
}

type evaluation struct {
	state    []int
	vals     map[string]float64
	points   int
	score    float64
	feasible bool
	checks   []map[string]any
}

const pointsKey = "points"
const levelKey = "level_needed"

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
	for i, c := range in.Constraints {
		if c.Stat != pointsKey && c.Stat != levelKey && !knownStat(baseEnv, c.Stat) {
			e := unknownStat(c.Stat)
			e.Hint = fmt.Sprintf("constraints[%d]: use a PoB output key, %q or %q.", i, pointsKey, levelKey)
			return nil, e
		}
		if c.Op != ">=" && c.Op != "<=" && c.Op != "==" {
			return nil, invalidArg(fmt.Sprintf("constraints[%d].op must be >=, <= or ==.", i), []string{">=", "<=", "=="})
		}
	}
	budget := 300
	if in.Budget != nil {
		budget = min(max(*in.Budget, 1), 2000)
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

	slots, e := buildSlots(base, baseEnv, in.Slots)
	if e != nil {
		return nil, e
	}
	size := 1.0
	sweep := 0
	var perSlot []any
	for _, sl := range slots {
		size *= sl.count()
		sweep += int(min(sl.count(), spaceLimit))
		perSlot = append(perSlot, map[string]any{"slot": sl.name, "choices": sl.count(), "how": sl.sizeMsg})
	}
	if size > spaceLimit || (size > exhaustiveLimit && budget < sweep) {
		e := errf(CodeSpaceTooLarge, "The search space has %.0f combinations.", size)
		e.SizePerSlot = perSlot
		for _, sl := range slots {
			if sl.count() > 20 {
				e.Suggestions = append(e.Suggestions, fmt.Sprintf("%s: %s; shrink it (pass the top candidates from rank_candidates, or lower k)", sl.name, sl.sizeMsg))
			}
		}
		if size <= spaceLimit {
			e.Suggestions = append(e.Suggestions, fmt.Sprintf("raise budget to >= %d", sweep))
		}
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
		if c.Stat != pointsKey && c.Stat != levelKey {
			watch = append(watch, c.Stat)
		}
	}

	start := time.Now()
	deadline := start.Add(time.Duration(maxMs) * time.Millisecond)
	memo := map[string]*evaluation{}
	used := 0
	stopped := ""
	baseXML, e := serialize(base)
	if e != nil {
		return nil, e
	}
	keyOf := func(st []int) string {
		parts := make([]string, len(st))
		for i, v := range st {
			parts[i] = strconv.Itoa(v)
		}
		return strings.Join(parts, ",")
	}
	eval := func(st []int) *evaluation {
		k := keyOf(st)
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
		c, perr := parseXML(baseXML)
		if perr != nil {
			return nil
		}
		for i, sl := range slots {
			sl.choices[st[i]].apply(c)
		}
		env := calcEnv(c)
		ev := &evaluation{state: slices.Clone(st), vals: map[string]float64{}}
		for _, w := range watch {
			ev.vals[w], _ = statValue(env, w)
		}
		pi := pointsOf(env)
		ev.points = pi.Used - basePoints.Used
		ev.vals[pointsKey] = float64(ev.points)
		ev.vals[levelKey] = float64(pi.LevelNeeded)
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
	if size <= exhaustiveLimit && size <= float64(budget) {
		st := make([]int, len(slots))
		for {
			if eval(st) == nil {
				break
			}
			i := 0
			for i < len(st) {
				st[i]++
				if st[i] < len(slots[i].choices) {
					break
				}
				st[i] = 0
				i++
			}
			if i == len(st) {
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
	baseVals[pointsKey] = 0
	baseVals[levelKey] = float64(basePoints.LevelNeeded)
	must := []string{in.Objective.Stat}
	for _, c := range in.Constraints {
		must = append(must, c.Stat)
	}
	var results []any
	for i, ev := range evals[:min(top, len(evals))] {
		var picks []map[string]any
		for si, sl := range slots {
			picks = append(picks, map[string]any{"slot": sl.name, "choice": sl.choices[ev.state[si]].label})
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
		"space":          map[string]any{"size": size, "method": method, "per_slot": perSlot},
		"proven_optimal": proven,
		"evaluations":    map[string]any{"used": used, "budget": budget},
		"elapsed_ms":     time.Since(start).Milliseconds(),
		"results":        nonNil(results),
	}
	if stopped != "" {
		res["stopped"] = stopped
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
			c, perr := parseXML(baseXML)
			if perr != nil {
				return nil, perr
			}
			for i, sl := range slots {
				sl.choices[best.state[i]].apply(c)
			}
			rev, e := s.commit(h, c)
			if e != nil {
				return nil, e
			}
			d := deltaResult(h, rev, baseEnv, h.revs[rev].env, false, []string{"applied the best feasible result"}, nil)
			res["applied"] = map[string]any{"revision": rev}
			res["revision"] = rev
			res["delta"] = d["delta"]
		}
	} else {
		res["hint"] = "Re-run with apply:true, or apply a result with edit_skills/edit_items/edit_tree using its choices."
	}
	limit := capOptimize
	if top > 3 {
		limit = capOptimizeMax
	}
	fitBudget(res, "results", limit, "lower top")
	return res, nil
}

func parseXML(xml []byte) (*pob.PathOfBuilding, *ToolError) {
	h := &handle{revs: []*revision{{xml: xml}}}
	return h.parseRev(0)
}

func constraintCurrent(env *calculator.Environment, stat string, p pointsInfo) float64 {
	switch stat {
	case pointsKey:
		return 0
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
	return slices.Compare(a.state, b.state) < 0
}

// coordinateAscent keeps the best beamWidth states and, each sweep, re-chooses one slot at a time for every state in
// the beam; it starts from the current build plus restarts seeded random states and stops when a sweep finds no
// better state or eval refuses (budget or time).
func coordinateAscent(slots []optSlot, seed int64, eval func([]int) *evaluation) {
	rng := rand.New(rand.NewPCG(uint64(seed), 0x9e3779b97f4a7c15)) //nolint:gosec // restarts must repeat for a seed
	cur := make([]int, len(slots))
	for i, sl := range slots {
		cur[i] = sl.current
	}
	var beam []*evaluation
	add := func(ev *evaluation) {
		if ev != nil && !slices.ContainsFunc(beam, func(b *evaluation) bool { return slices.Equal(b.state, ev.state) }) {
			beam = append(beam, ev)
		}
	}
	add(eval(cur))
	for range restarts {
		st := make([]int, len(slots))
		for i, sl := range slots {
			st[i] = rng.IntN(len(sl.choices))
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
		for _, st := range beam {
			for i, sl := range slots {
				for c := range sl.choices {
					if c == st.state[i] {
						continue
					}
					s := slices.Clone(st.state)
					s[i] = c
					ev := eval(s)
					if ev == nil {
						halted = true
						break
					}
					if !slices.ContainsFunc(next, func(b *evaluation) bool { return slices.Equal(b.state, ev.state) }) {
						next = append(next, ev)
					}
				}
				if halted {
					break
				}
			}
			if halted {
				break
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

func buildSlots(b *pob.PathOfBuilding, env *calculator.Environment, in []optSlotIn) ([]optSlot, *ToolError) {
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
		if sl.count() == 0 {
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
	sl := optSlot{name: slot}
	if in.KeepCurrent == nil || *in.KeepCurrent {
		cur := slotItemName(b, slot)
		if cur == "" {
			cur = "(empty)"
		}
		sl.choices = append(sl.choices, choice{label: "(current) " + cur, apply: func(*pob.PathOfBuilding) {}})
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
		sl.choices = append(sl.choices, choice{label: label, apply: func(c *pob.PathOfBuilding) {
			calculator.EquipItem(c, text, slot)
		}})
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
	sl.sizeMsg = fmt.Sprintf("%d items", len(sl.choices))
	return sl, nil
}

func combinations(n, k int) [][]int {
	var out [][]int
	var rec func(start int, cur []int)
	rec = func(start int, cur []int) {
		if len(out) > spaceLimit {
			return
		}
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
	sl := optSlot{name: fmt.Sprintf("supports g%d", group)}
	n := binom(len(pool), k)
	sl.sizeMsg = fmt.Sprintf("%d candidates choose %d = %.0f", len(pool), k, n)
	if n > spaceLimit {
		sl.tooMany = n
		return sl, nil
	}
	fixed := []pob.Gem{}
	for i, gem := range g.Gems {
		if !slices.Contains(replaceIdx, i) {
			fixed = append(fixed, gem)
		}
	}
	for ci, combo := range combinations(len(pool), k) {
		var names []string
		set := slices.Clone(fixed)
		for _, p := range combo {
			names = append(names, pool[p].name)
			set = append(set, pool[p].gem)
		}
		if keep && k == len(replaceIdx) && slices.Equal(combo, seq(k)) {
			sl.current = ci
		}
		gemsCopy := set
		sl.choices = append(sl.choices, choice{label: names, apply: func(c *pob.PathOfBuilding) {
			(*groupsOf(c))[group-1].Gems = slices.Clone(gemsCopy)
		}})
	}
	return sl, nil
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
	var nodes []treeNode
	for _, c := range in.Candidates {
		n, e := resolveNode(b, v, json.RawMessage(strconv.Quote(c)), &plan)
		if e != nil {
			return optSlot{}, e
		}
		if !slices.ContainsFunc(nodes, func(x treeNode) bool { return x.ID == n.ID }) {
			nodes = append(nodes, n)
		}
	}
	k := len(nodes)
	if in.K != nil {
		k = *in.K
	}
	if k < 0 || k > len(nodes) {
		return optSlot{}, invalidArg(fmt.Sprintf("k=%d but only %d nodes are listed.", k, len(nodes)), nil)
	}
	withPath := in.Path == nil || *in.Path
	sl := optSlot{name: "nodes"}
	total := 0.0
	for size := 0; size <= k; size++ {
		total += binom(len(nodes), size)
	}
	sl.sizeMsg = fmt.Sprintf("subsets of up to %d of %d nodes = %.0f", k, len(nodes), total)
	if total > spaceLimit {
		sl.tooMany = total
		return sl, nil
	}
	for size := 0; size <= k; size++ {
		for _, combo := range combinations(len(nodes), size) {
			var names []string
			var picked []treeNode
			for _, p := range combo {
				names = append(names, nodes[p].Name)
				picked = append(picked, nodes[p])
			}
			sl.choices = append(sl.choices, choice{label: nonNil(names), apply: func(c *pob.PathOfBuilding) {
				for _, n := range picked {
					if slices.Contains(c.Build.PassiveNodes, n.ID) {
						continue
					}
					if !withPath {
						c.AllocateNodes([]int64{n.ID})
						continue
					}
					if p := planPaths(c, v).path(n.ID); len(p) > 0 {
						c.AllocateNodes(p)
					}
				}
			}})
		}
	}
	return sl, nil
}
