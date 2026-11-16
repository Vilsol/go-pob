package mcp

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

const (
	capCompare      = 3 * 1024
	capCompareSteps = 6 * 1024
)

type compareIn struct {
	A         string   `json:"a"`
	B         string   `json:"b"`
	Stats     []string `json:"stats"`
	Structure *bool    `json:"structure"`
	Steps     bool     `json:"steps"`
}

type ref struct {
	h   *handle
	rev int
}

func (s *Server) parseRef(r string) (ref, *ToolError) {
	id, revText, hasRev := strings.Cut(strings.TrimSpace(r), "@")
	h, e := s.get(id)
	if e != nil {
		return ref{}, e
	}
	rev := h.cur
	if hasRev {
		n, err := strconv.Atoi(revText)
		if err != nil {
			return ref{}, invalidArg(fmt.Sprintf("%q is not handle@revision.", r), nil)
		}
		rev = n
	}
	if _, e := h.parseRev(rev); e != nil {
		return ref{}, e
	}
	return ref{h, rev}, nil
}

func (s *Server) compareBuilds(args json.RawMessage) (any, *ToolError) {
	var in compareIn
	if e := decode(args, schemaCompareBuilds, &in); e != nil {
		return nil, e
	}
	a, e := s.parseRef(in.A)
	if e != nil {
		return nil, e
	}
	b, e := s.parseRef(in.B)
	if e != nil {
		return nil, e
	}
	if in.Steps {
		return s.compareSteps(a, b, in.Stats)
	}
	ea, e := a.h.envAt(a.rev)
	if e != nil {
		return nil, e
	}
	eb, e := b.h.envAt(b.rev)
	if e != nil {
		return nil, e
	}
	keys := in.Stats
	if len(keys) == 0 {
		keys = summaryKeys
	}
	var rows []map[string]any
	for _, k := range keys {
		va, _ := statValue(ea, k)
		vb, _ := statValue(eb, k)
		if round(va) == round(vb) && len(in.Stats) == 0 {
			continue
		}
		rows = append(rows, map[string]any{"stat": k, "a": round(va), "b": round(vb), "pct": pct(va, vb)})
	}
	res := map[string]any{"a": fmt.Sprintf("%s@%d", a.h.id, a.rev), "b": fmt.Sprintf("%s@%d", b.h.id, b.rev), "stats": nonNil(rows)}
	if in.Structure == nil || *in.Structure {
		ba, _ := a.h.parseRev(a.rev)
		bb, _ := b.h.parseRev(b.rev)
		res["structure"] = structureDiff(ba, bb)
	}
	if out, _ := json.Marshal(res); len(out) > capCompare {
		st, _ := res["structure"].(map[string]any)
		for _, k := range []string{"config_changed", "gems_changed", "items_changed", "nodes_removed", "nodes_added"} {
			if out, _ := json.Marshal(res); len(out) <= capCompare || st == nil {
				break
			}
			if list, ok := st[k].([]string); ok && len(list) > 10 {
				st[k] = append(list[:10], fmt.Sprintf("... %d more", len(list)-10))
			}
		}
		res["truncated"] = true
	}
	return res, nil
}

const maxSteps = 25

// compareSteps is one row per revision of a handle from a to b along the line of edits that led to b, so revisions
// a revert abandoned are left out: what changed against the revision it was edited from and the watched stats that
// moved.
func (s *Server) compareSteps(a, b ref, watch []string) (any, *ToolError) {
	if a.h != b.h {
		return nil, invalidArg("steps compares two revisions of one handle (handle@a and handle@b).", nil)
	}
	from, to := min(a.rev, b.rev), max(a.rev, b.rev)
	line := []int{to}
	for r := to; r > from; {
		r = a.h.parent(r)
		line = append(line, r)
	}
	slices.Reverse(line)
	res := map[string]any{"handle": a.h.id, "from": from, "to": to}
	if line[0] != from {
		res["note"] = fmt.Sprintf("revision %d was reverted away from before %d was made; the steps start at revision %d, where their edits meet", from, to, max(line[0], 0))
		line[0] = max(line[0], 0)
		res["from"] = line[0]
	}
	var reverted []int
	for r := line[0] + 1; r < to; r++ {
		if !slices.Contains(line, r) {
			reverted = append(reverted, r)
		}
	}
	if len(reverted) > 0 {
		res["reverted"] = reverted
	}
	if len(line)-1 > maxSteps {
		res["omitted"] = len(line) - 1 - maxSteps
		res["how_to_get_more"] = fmt.Sprintf("compare_builds steps=true a=%s@%d b=%s@%d", a.h.id, line[maxSteps], a.h.id, to)
		line = line[:maxSteps+1]
	}
	keys := append(slices.Clone(summaryKeys), watch...)
	// envs of older revisions are calculated here without being kept, so a long range does not pin them in memory
	envOf := func(rev int, b *pob.PathOfBuilding) *calculator.Environment {
		if r := a.h.revs[rev]; r.env != nil {
			return r.env
		}
		return calcEnv(b)
	}
	var steps []any
	var pe *calculator.Environment
	for i := 1; i < len(line); i++ {
		prev, rev := line[i-1], line[i]
		pb, e1 := a.h.parseRev(prev)
		cb, e2 := a.h.parseRev(rev)
		if e1 != nil || e2 != nil {
			steps = append(steps, map[string]any{"revision": rev, "changes": "snapshot no longer kept"})
			pe = nil
			continue
		}
		if pe == nil {
			pe = envOf(prev, pb)
		}
		ce := envOf(rev, cb)
		delta := map[string]any{}
		for _, k := range keys {
			va, _ := statValue(pe, k)
			vb, _ := statValue(ce, k)
			if round(va) == round(vb) {
				continue
			}
			if p := pct(va, vb); p != nil {
				delta[k] = fmt.Sprintf("%+g%%", *p)
			} else {
				delta[k] = round(vb - va)
			}
		}
		row := map[string]any{"revision": rev, "changes": changeSummary(structureDiff(pb, cb))}
		if len(delta) > 0 {
			row["delta"] = delta
		}
		steps = append(steps, row)
		pe = ce
	}
	res["steps"] = nonNil(steps)
	if dropped := fitBudget(res, "steps", capCompareSteps, ""); dropped > 0 {
		last := line[len(line)-1-dropped]
		res["how_to_get_more"] = fmt.Sprintf("compare_builds steps=true a=%s@%d b=%s@%d", a.h.id, last, a.h.id, to)
	}
	return res, nil
}

// changeSummary is structureDiff as one line: nodes, masteries, items, gems, config and level.
func changeSummary(d map[string]any) string {
	var parts []string
	names := func(sign string, list []string) {
		if len(list) == 0 {
			return
		}
		shown := list[:min(len(list), 4)]
		p := sign + strings.Join(shown, ", "+sign)
		if len(list) > len(shown) {
			p += fmt.Sprintf(" (+%d more)", len(list)-len(shown))
		}
		parts = append(parts, p)
	}
	names("+", d["nodes_added"].([]string))
	names("-", d["nodes_removed"].([]string))
	for _, m := range d["masteries_changed"].([]map[string]any) {
		parts = append(parts, fmt.Sprintf("%s: %v", m["name"], m["b"]))
	}
	for _, it := range d["items_changed"].([]map[string]any) {
		if it["a"] != it["b"] {
			parts = append(parts, fmt.Sprintf("%s: %s -> %s", it["slot"], orNone(it["a"].(string)), orNone(it["b"].(string))))
			continue
		}
		var lines []string
		for _, l := range []struct {
			sign string
			list any
		}{{"+", it["lines_added"]}, {"-", it["lines_removed"]}} {
			list, _ := l.list.([]string)
			for _, x := range list[:min(len(list), 2)] {
				lines = append(lines, l.sign+x)
			}
			if len(list) > 2 {
				lines = append(lines, fmt.Sprintf("%s%d more", l.sign, len(list)-2))
			}
		}
		if sockets, ok := it["sockets"].(string); ok {
			lines = append([]string{"sockets " + sockets}, lines...)
		}
		if len(lines) == 0 {
			lines = []string{"no mod line changed"}
		}
		parts = append(parts, fmt.Sprintf("%s: %s edited (%s)", it["slot"], orNone(it["a"].(string)), strings.Join(lines, ", ")))
	}
	for _, g := range d["gems_changed"].([]map[string]any) {
		p := fmt.Sprintf("group %v", g["group"])
		switch {
		case g["group_added"] == true:
			p = fmt.Sprintf("new group %v", g["group_b"])
		case g["group_removed"] == true:
			p += " removed:"
		case g["group_b"] != nil:
			p += fmt.Sprintf(" (now %v)", g["group_b"])
		}
		for _, x := range g["added"].([]string) {
			p += " +" + x
		}
		for _, x := range g["removed"].([]string) {
			p += " -" + x
		}
		for _, x := range g["edited"].([]string) {
			p += " " + x
		}
		parts = append(parts, p)
	}
	parts = append(parts, d["config_changed"].([]string)...)
	if lv := d["level"].([]int); lv[0] != lv[1] {
		parts = append(parts, fmt.Sprintf("level %d -> %d", lv[0], lv[1]))
	}
	if len(parts) == 0 {
		return "no structural change"
	}
	return strings.Join(parts, "; ")
}

func structureDiff(a, b *pob.PathOfBuilding) map[string]any {
	va, vb := specTreeVersion(a), specTreeVersion(b)
	inA, inB := map[int64]bool{}, map[int64]bool{}
	for _, id := range a.Build.PassiveNodes {
		inA[id] = true
	}
	for _, id := range b.Build.PassiveNodes {
		inB[id] = true
	}
	var added, removed []string
	for _, id := range b.Build.PassiveNodes {
		if !inA[id] {
			if n, ok := treeNodeByID(vb, id); ok {
				added = append(added, n.Name)
			}
		}
	}
	for _, id := range a.Build.PassiveNodes {
		if !inB[id] {
			if n, ok := treeNodeByID(va, id); ok {
				removed = append(removed, n.Name)
			}
		}
	}
	slices.Sort(added)
	slices.Sort(removed)
	itemsA, itemsB := map[string]string{}, map[string]string{}
	for _, e := range equippedItems(a) {
		itemsA[e.slot] = e.name + "\x00" + e.text
	}
	for _, e := range equippedItems(b) {
		itemsB[e.slot] = e.name + "\x00" + e.text
	}
	var items []map[string]any
	for _, slot := range slotNames() {
		if itemsA[slot] != itemsB[slot] {
			na, ta, _ := strings.Cut(itemsA[slot], "\x00")
			nb, tb, _ := strings.Cut(itemsB[slot], "\x00")
			row := map[string]any{"slot": slot, "a": na, "b": nb}
			if na == nb {
				la, lb := itemTextLines(ta), itemTextLines(tb)
				row["lines_added"], row["lines_removed"] = setDiff(lb, la), setDiff(la, lb)
				if sa, sb := itemSocketsLine(ta), itemSocketsLine(tb); sa != sb {
					row["sockets"] = orNone(sa) + " -> " + orNone(sb)
				}
			}
			items = append(items, row)
		}
	}
	ga, gb := activeSkills(a), activeSkills(b)
	var gemsChanged []map[string]any
	names := func(g pob.Skill) []string {
		out := make([]string, 0, len(g.Gems))
		for _, x := range g.Gems {
			out = append(out, x.NameSpec)
		}
		return out
	}
	for _, p := range matchGroups(ga, gb) {
		var na, nb []string
		row := map[string]any{}
		if p.a >= 0 {
			na = names(ga[p.a])
			row["group"] = p.a + 1
		}
		if p.b >= 0 {
			nb = names(gb[p.b])
			if p.b != p.a {
				row["group_b"] = p.b + 1
			}
		}
		switch {
		case p.b < 0:
			row["group_removed"] = true
		case p.a < 0:
			row["group_added"] = true
		}
		rem, add := gemNameDiff(na, nb), gemNameDiff(nb, na)
		var edited []string
		if p.a >= 0 && p.b >= 0 {
			edited = gemEdits(ga[p.a].Gems, gb[p.b].Gems)
		}
		if len(rem)+len(add)+len(edited) > 0 {
			row["removed"], row["added"], row["edited"] = rem, add, nonNil(edited)
			gemsChanged = append(gemsChanged, row)
		}
	}
	masteries := masteryChanges(a, b)
	cfgA, cfgB := map[string]string{}, map[string]string{}
	for _, row := range configSection(a) {
		m := row.(map[string]any)
		cfgA[m["var"].(string)] = fmt.Sprint(m["value"])
	}
	for _, row := range configSection(b) {
		m := row.(map[string]any)
		cfgB[m["var"].(string)] = fmt.Sprint(m["value"])
	}
	var cfg []string
	for k, v := range cfgB {
		if cfgA[k] != v {
			cfg = append(cfg, fmt.Sprintf("%s: %s -> %s", k, orDefault(cfgA[k]), v))
		}
	}
	for k, v := range cfgA {
		if _, ok := cfgB[k]; !ok {
			cfg = append(cfg, fmt.Sprintf("%s: %s -> default", k, v))
		}
	}
	slices.Sort(cfg)
	return map[string]any{
		"nodes_added": nonNil(added), "nodes_removed": nonNil(removed), "items_changed": nonNil(items),
		"masteries_changed": nonNil(masteries), "gems_changed": nonNil(gemsChanged), "config_changed": nonNil(cfg), "level": []int{a.Build.Level, b.Build.Level},
	}
}

// masteryChanges lists the masteries whose selected effect differs between a and b, by node id.
func masteryChanges(a, b *pob.PathOfBuilding) []map[string]any {
	effects := func(p *pob.PathOfBuilding) map[int64]string {
		out := map[int64]string{}
		spec := activeSpec(p)
		if spec == nil {
			return out
		}
		v := specTreeVersion(p)
		for _, m := range pob.SpecMasterySelections(*spec) {
			if n, ok := treeNodeByID(v, m.Node); ok {
				for _, e := range n.Node.MasteryEffects {
					if e.Effect == m.Effect {
						out[m.Node] = strings.Join(e.Stats, " / ")
					}
				}
			}
		}
		return out
	}
	ea, eb := effects(a), effects(b)
	var ids []int64
	for id := range ea {
		if ea[id] != eb[id] {
			ids = append(ids, id)
		}
	}
	for id := range eb {
		if _, ok := ea[id]; !ok {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	out := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		name := ""
		if n, ok := treeNodeByID(specTreeVersion(b), id); ok {
			name = n.Name
		} else if n, ok := treeNodeByID(specTreeVersion(a), id); ok {
			name = n.Name
		}
		out = append(out, map[string]any{"id": id, "name": name, "a": orNone(ea[id]), "b": orNone(eb[id])})
	}
	return out
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func orDefault(s string) string {
	if s == "" {
		return "default"
	}
	return s
}

// gemEdits names the level, quality and enabled changes of gems present by name in both lists.
type groupPair struct{ a, b int }

// matchGroups pairs the socket groups of two builds by content rather than index: identical groups (slot, label,
// source and gems) first, then the most alike by shared gems, slot and label. Unpaired groups have -1 on the other
// side; pairs come in a's order, then b's unpaired groups.
func matchGroups(ga, gb []pob.Skill) []groupPair {
	pa, pb := make([]int, len(ga)), make([]int, len(gb))
	for i := range pa {
		pa[i] = -1
	}
	for j := range pb {
		pb[j] = -1
	}
	key := func(g pob.Skill) string {
		k := g.Slot + "\x00" + g.Label + "\x00" + g.Source
		for _, x := range g.Gems {
			k += "\x00" + x.NameSpec
		}
		return k
	}
	for i := range ga {
		ki := key(ga[i])
		best := -1
		for j := range gb {
			if pb[j] < 0 && key(gb[j]) == ki && (best < 0 || abs(j-i) < abs(best-i)) {
				best = j
			}
		}
		if best >= 0 {
			pa[i], pb[best] = best, i
		}
	}
	type cand struct{ i, j, score int }
	var cands []cand
	for i := range ga {
		if pa[i] >= 0 {
			continue
		}
		for j := range gb {
			if pb[j] >= 0 || ga[i].Source != gb[j].Source {
				continue
			}
			shared := 0
			for _, x := range ga[i].Gems {
				if slices.ContainsFunc(gb[j].Gems, func(y pob.Gem) bool { return y.NameSpec == x.NameSpec }) {
					shared++
				}
			}
			label := ga[i].Label != "" && ga[i].Label == gb[j].Label
			if shared == 0 && !label {
				continue
			}
			score := shared * 4
			if label {
				score += 2
			}
			if ga[i].Slot == gb[j].Slot {
				score++
			}
			cands = append(cands, cand{i, j, score})
		}
	}
	slices.SortStableFunc(cands, func(x, y cand) int {
		if x.score != y.score {
			return y.score - x.score
		}
		return abs(x.j-x.i) - abs(y.j-y.i)
	})
	for _, c := range cands {
		if pa[c.i] < 0 && pb[c.j] < 0 {
			pa[c.i], pb[c.j] = c.j, c.i
		}
	}
	out := make([]groupPair, 0, len(ga)+len(gb))
	for i, j := range pa {
		out = append(out, groupPair{i, j})
	}
	for j, i := range pb {
		if i < 0 {
			out = append(out, groupPair{-1, j})
		}
	}
	return out
}

// gemEdits pairs a gem of b with an identical one of a first, then with the next unpaired one of its name, so copies
// of one gem are not edits of each other.
func gemEdits(a, b []pob.Gem) []string {
	used := make([]bool, len(a))
	free := func(match func(pob.Gem) bool) int {
		for i, g := range a {
			if !used[i] && match(g) {
				return i
			}
		}
		return -1
	}
	pair := make([]int, len(b))
	for j, gb := range b {
		pair[j] = free(func(g pob.Gem) bool {
			return g.NameSpec == gb.NameSpec && g.Level == gb.Level && g.Quality == gb.Quality && g.Enabled == gb.Enabled
		})
		if pair[j] >= 0 {
			used[pair[j]] = true
		}
	}
	var out []string
	for j, gb := range b {
		if pair[j] >= 0 {
			continue
		}
		i := free(func(g pob.Gem) bool { return g.NameSpec == gb.NameSpec })
		if i < 0 {
			continue
		}
		used[i] = true
		ga := a[i]
		var p []string
		if ga.Level != gb.Level {
			p = append(p, fmt.Sprintf("level %d -> %d", ga.Level, gb.Level))
		}
		if ga.Quality != gb.Quality {
			p = append(p, fmt.Sprintf("quality %d -> %d", ga.Quality, gb.Quality))
		}
		if ga.Enabled != gb.Enabled {
			p = append(p, map[bool]string{true: "enabled", false: "disabled"}[gb.Enabled])
		}
		if len(p) > 0 {
			out = append(out, gb.NameSpec+" "+strings.Join(p, ", "))
		}
	}
	return out
}

// itemSocketsLine is an item's sockets as its Sockets: line writes them, so a recolour is a change of its own.
func itemSocketsLine(text string) string {
	for _, l := range strings.Split(text, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(l), "Sockets: "); ok {
			return v
		}
	}
	return ""
}

// itemTextLines are an item's visible mod lines, for a line diff of two versions of one item.
func itemTextLines(text string) []string {
	item := calculator.ParseItem(text)
	var out []string
	for _, list := range [][]calculator.ModLine{item.EnchantModLines, item.ScourgeModLines, item.ImplicitModLines, item.ExplicitModLines, item.CrucibleModLines} {
		for _, ml := range list {
			if item.ItemModLineVisible(ml) {
				out = append(out, ml.Line)
			}
		}
	}
	return out
}

// gemNameDiff are the gems of a that b has fewer copies of, once per missing copy.
func gemNameDiff(a, b []string) []string {
	left := slices.Clone(b)
	out := []string{}
	for _, x := range a {
		if i := slices.Index(left, x); i >= 0 {
			left = slices.Delete(left, i, i+1)
		} else {
			out = append(out, x)
		}
	}
	return out
}

func setDiff(a, b []string) []string {
	out := []string{}
	for _, x := range a {
		if !slices.Contains(b, x) {
			out = append(out, x)
		}
	}
	return out
}
