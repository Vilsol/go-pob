package mcp

import (
	"encoding/json"

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
func deltaResult(h *handle, rev int, before, after *calculator.Environment, dryRun bool, applied []string, watch []string) map[string]any {
	rows, more := statDelta(before, after, append(append([]string{}, summaryKeys...), watch...), nil, 12)
	wb, wa := cachedWarnings(before), cachedWarnings(after)
	res := map[string]any{
		"handle": h.id, "revision": rev, "dry_run": dryRun,
		"applied":          nonNil(applied),
		"delta":            nonNil(rows),
		"points":           pointsOf(after),
		"warnings_added":   diffStrings(wb, wa),
		"warnings_removed": diffStrings(wa, wb),
	}
	if more > 0 {
		res["more"] = more
	}
	return res
}

const capWrite = 1536

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
// warnings are shortened first, then the smallest delta rows dropped.
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
	for len(rows) > 3 && size() > capWrite {
		rows = rows[:len(rows)-1]
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
	// after runs on the calculated result, for checks that need the new environment
	after []func(env *calculator.Environment)
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
	for i := range n {
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
	var after *calculator.Environment
	rev := h.cur
	if dryRun {
		xml, e := serialize(b)
		if e != nil {
			return nil, e
		}
		h2 := &handle{id: h.id, revs: []*revision{{xml: xml}}}
		if after, e = h2.envAt(0); e != nil {
			return nil, e
		}
	} else {
		if rev, e = s.commit(h, b); e != nil {
			return nil, e
		}
		after = h.revs[rev].env
	}
	for _, f := range ec.after {
		f(after)
	}
	res := deltaResult(h, rev, before, after, dryRun, ec.applied, watch)
	for k, v := range ec.extra {
		res[k] = v
	}
	if len(unknown) > 0 {
		res["unknown_watch"] = unknown
	}
	if len(ec.hints) > 0 {
		res["hint"] = ec.hints[0]
	}
	capWriteResult(res)
	return res, nil
}
