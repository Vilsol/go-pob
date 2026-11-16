package mcp

import (
	"encoding/json"
	"slices"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

var warnCache = map[*calculator.Environment][]string{}

// cachedWarnings memoises warningsOf per environment; callers hold the server lock.
func cachedWarnings(env *calculator.Environment) []string {
	if w, ok := warnCache[env]; ok {
		return w
	}
	if len(warnCache) > 64 {
		warnCache = map[*calculator.Environment][]string{}
	}
	w := warningsOf(env)
	warnCache[env] = w
	return w
}

func diffStrings(a, b []string) []string {
	in := map[string]bool{}
	for _, x := range a {
		in[x] = true
	}
	out := []string{}
	for _, x := range b {
		if !in[x] {
			out = append(out, x)
		}
	}
	return out
}

// deltaResult is the stat delta every write and dry run returns (spec §2).
// MainDPS (after FullDPS on a build with Full DPS skills), the watch keys, TotalEHP and Life always appear: as a delta
// row when they moved, else in "unchanged" with their value; capWriteResult trims every other row before them.
// Every resistance always appears with its total and headroom (margin over the cap, negative below it).
func deltaResult(h *handle, rev int, before, after *calculator.Environment, dryRun bool, applied []string, watch []string) map[string]any {
	must := []string{mainDPS}
	if before.Player.Output["FullDPS"] > 0 {
		must = []string{"FullDPS", mainDPS}
	}
	for _, w := range append(slices.Clone(watch), "TotalEHP", "Life") {
		if knownStat(after, w) && !slices.Contains(must, w) {
			must = append(must, w)
		}
	}
	all, more := statDelta(before, after, append(append([]string{}, summaryKeys...), watch...), must, 12)
	rows := []deltaRow{}
	unchanged := map[string]float64{}
	for _, r := range all {
		if slices.Contains(must, r.Stat) && r.Before == r.After {
			unchanged[r.Stat] = r.After
			continue
		}
		rows = append(rows, r)
	}
	wb, wa := cachedWarnings(before), cachedWarnings(after)
	res := map[string]any{
		"handle": h.id, "revision": rev, "dry_run": dryRun,
		"applied":          nonNil(applied),
		"delta":            rows,
		"resists":          resistBlock(before, after),
		"points":           pointsOf(after),
		"warnings_added":   diffStrings(wb, wa),
		"warnings_removed": diffStrings(wa, wb),
	}
	if len(unchanged) > 0 {
		res["unchanged"] = unchanged
	}
	if more > 0 {
		res["more"] = more
	}
	return res
}

const capWrite = 1536

type resistRow struct {
	Resist         float64  `json:"resist"`
	Total          float64  `json:"total"`
	Headroom       float64  `json:"headroom"`
	HeadroomChange *float64 `json:"headroom_change,omitempty"`
	UnderCap       bool     `json:"under_cap,omitempty"`
}

func resistBlock(before, after *calculator.Environment) map[string]resistRow {
	hb, ha := headroom(before), headroom(after)
	out := make(map[string]resistRow, len(resistElems))
	for _, el := range resistElems {
		r, _ := statValue(after, el+"Resist")
		total, _ := statValue(after, el+"ResistTotal")
		row := resistRow{Resist: round(r), Total: round(total), Headroom: ha[el], UnderCap: ha[el] < 0}
		if d := round(ha[el] - hb[el]); d != 0 {
			row.HeadroomChange = &d
		}
		out[el] = row
	}
	return out
}

func shorten(list []string, n int) []string {
	out := make([]string, len(list))
	for i, s := range list {
		if len(s) > n {
			s = s[:n] + "..."
		}
		out[i] = s
	}
	return out
}

// capWriteResult keeps a write result within its budget, ignoring the dry-run item text the caller asked for:
// warnings are shortened first, then the smallest delta rows the caller did not ask for dropped.
func capWriteResult(res map[string]any) {
	size := func() int {
		item := res["item"]
		delete(res, "item")
		b, _ := json.Marshal(res)
		if item != nil {
			res["item"] = item
		}
		return len(b)
	}
	if size() <= capWrite {
		return
	}
	for _, k := range []string{"warnings_added", "warnings_removed"} {
		if l, ok := res[k].([]string); ok {
			res[k] = shorten(l, 120)
		}
	}
	rows, _ := res["delta"].([]deltaRow)
	more, _ := res["more"].(int)
	for size() > capWrite {
		i := len(rows) - 1
		for i >= 0 && rows[i].keep {
			i--
		}
		if i < 0 {
			break
		}
		rows = slices.Delete(rows, i, i+1)
		more++
		res["delta"] = rows
		res["more"] = more
	}
	for _, k := range []string{"warnings_removed", "warnings_added"} {
		for size() > capWrite {
			l, _ := res[k].([]string)
			if len(l) == 0 {
				break
			}
			res[k] = l[:len(l)-1]
			res["truncated"] = true
		}
	}
}

type editCtx struct {
	s       *Server
	h       *handle
	b       *pob.PathOfBuilding
	applied []string
	extra   map[string]any
	hints   []string
	op      int
	// after runs on the calculated result, for checks that need the new environment
	after []func(env *calculator.Environment)
	// verify runs on the calculated result before it is committed; an error refuses the whole batch
	verify []func(env *calculator.Environment) *ToolError
	// groupMap maps the socket group indexes a batch's ops give (the groups before the batch, then the ones add_group
	// creates) to their index now, 0 once removed; origGroups holds the groups as they were before the batch
	groupMap   []int
	origGroups []pob.Skill
}

func (ec *editCtx) note(s string) {
	ec.applied = append(ec.applied, s)
}

func (ec *editCtx) add(key string, v any) {
	if ec.extra == nil {
		ec.extra = map[string]any{}
	}
	if list, ok := ec.extra[key].([]any); ok {
		ec.extra[key] = append(list, v)
		return
	}
	ec.extra[key] = []any{v}
}

// runEdit applies n ops in order to a copy of the handle's current revision; any failing op aborts the batch.
func (s *Server) runEdit(handleID string, dryRun bool, watch []string, n int, apply func(ec *editCtx, i int) *ToolError) (any, *ToolError) {
	h, e := s.get(handleID)
	if e != nil {
		return nil, e
	}
	b, e := h.parseRev(h.cur)
	if e != nil {
		return nil, e
	}
	before, e := h.envAt(h.cur)
	if e != nil {
		return nil, e
	}
	ec := &editCtx{s: s, h: h, b: b}
	var sources map[string]bool
	if len(forcedConditions(b)) > 0 {
		sources = sourceLines(b)
	}
	baseNodes := slices.Clone(b.Build.PassiveNodes)
	for i := range n {
		ec.op = i
		if e := apply(ec, i); e != nil {
			e.atOp(i)
			if e.Hint == "" {
				e.Hint = "Nothing was applied; fix this op and resend the batch."
			}
			return nil, e
		}
	}
	var unknown []string
	for _, w := range watch {
		if !knownStat(before, w) {
			unknown = append(unknown, w)
		}
	}
	rev := h.cur
	xml, after, e := calculate(b)
	if e != nil {
		return nil, e
	}
	for _, f := range ec.verify {
		if e := f(after); e != nil {
			if e.Hint == "" {
				e.Hint = "Nothing was applied; fix this op and resend the batch."
			}
			return nil, e
		}
	}
	if !dryRun {
		if rev, e = s.commitCalculated(h, xml, after); e != nil {
			return nil, e
		}
	}
	for _, f := range ec.after {
		f(after)
	}
	res := deltaResult(h, rev, before, after, dryRun, ec.applied, watch)
	if sources != nil {
		res["warnings_added"] = append(res["warnings_added"].([]string), sourceFlags(forcedConditions(b), sources, sourceLines(b))...)
	}
	res["warnings_added"] = append(res["warnings_added"].([]string), customModsFlags(baseNodes, b)...)
	res["warnings_added"] = append(res["warnings_added"].([]string), slowerFlag(before, after, mainDPS)...)
	res["warnings_added"] = append(res["warnings_added"].([]string), fullDPSFlag(before, after, mainDPS)...)
	res["warnings_added"] = append(res["warnings_added"].([]string), areaFlag(before, after, mainDPS)...)
	res["warnings_added"] = append(res["warnings_added"].([]string), lossFlags(before, after)...)
	for k, v := range ec.extra {
		res[k] = v
	}
	if len(unknown) > 0 {
		res["unknown_watch"] = unknown
	}
	if len(ec.hints) > 0 {
		res["hint"] = ec.hints[0]
	}
	if u := unsocketedList(b); len(u) > 0 {
		res["unsocketed"] = u
	}
	capWriteResult(res)
	return res, nil
}
