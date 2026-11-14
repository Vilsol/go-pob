package mcp

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

const capSearch = 2560

type searchIn struct {
	Kind        string   `json:"kind"`
	Query       string   `json:"query"`
	Handle      string   `json:"handle"`
	Slot        string   `json:"slot"`
	Base        string   `json:"base"`
	Tags        []string `json:"tags"`
	MaxLevelReq *int     `json:"max_level_req"`
	Limit       int      `json:"limit"`
	Cursor      string   `json:"cursor"`
}

var searchKinds = []string{"gem", "support", "unique", "base", "node", "keystone", "mastery", "affix", "config", "stat"}

type hit struct {
	rank matchRank
	name string
	row  func() map[string]any
}

func (s *Server) searchData(args json.RawMessage) (any, *ToolError) {
	var in searchIn
	if e := decode(args, schemaSearchData, &in); e != nil {
		return nil, e
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	limit = min(limit, 50)
	offset := 0
	if in.Cursor != "" {
		n, err := strconv.Atoi(strings.TrimPrefix(in.Cursor, "c"))
		if err != nil || n < 0 {
			return nil, invalidArg("cursor must be a next_cursor value from a previous page.", nil)
		}
		offset = n
	}
	var b *pob.PathOfBuilding
	if in.Handle != "" {
		h, e := s.get(in.Handle)
		if e != nil {
			return nil, e
		}
		if b, e = h.parseRev(h.cur); e != nil {
			return nil, e
		}
	}
	var hits []hit
	var e *ToolError
	switch in.Kind {
	case "gem", "support":
		var env *calculator.Environment
		group := 0
		if in.Handle != "" && in.Kind == "support" {
			h := s.handles[in.Handle]
			env, _ = h.envAt(h.cur)
			group = b.Build.MainSocketGroup
		}
		hits = searchGems(in, env, group)
	case "unique":
		hits, e = searchUniques(in, b)
	case "base":
		hits, e = searchBases(in, b)
	case "node", "keystone", "mastery":
		hits, e = searchNodes(in, b)
	case "affix":
		hits, e = searchAffixes(in)
	case "config":
		hits = searchConfig(in)
	case "stat":
		hits = searchStats(in)
	default:
		return nil, invalidArg(fmt.Sprintf("Unknown kind %q.", in.Kind), searchKinds)
	}
	if e != nil {
		return nil, e
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].rank != hits[j].rank {
			return hits[i].rank < hits[j].rank
		}
		return hits[i].name < hits[j].name
	})
	res := map[string]any{"kind": in.Kind, "total": len(hits)}
	var rows []any
	for i := offset; i < len(hits) && len(rows) < limit; i++ {
		row := hits[i].row()
		if in.Query != "" {
			row["match"] = hits[i].rank.label()
		}
		rows = append(rows, row)
	}
	res["results"] = nonNil(rows)
	if offset+len(rows) < len(hits) {
		res["next_cursor"] = "c" + strconv.Itoa(offset+len(rows))
	}
	budget := capSearch
	if in.Limit > 10 {
		budget = capSearch * 5
	}
	before := len(rows)
	fitBudget(res, "results", budget, "page with cursor")
	if after := len(res["results"].([]any)); after < before {
		res["next_cursor"] = "c" + strconv.Itoa(offset+after)
	}
	if len(hits) == 0 && in.Query != "" {
		res["hint"] = "No match; try fewer or shorter words."
	}
	return res, nil
}

func matchQuery(q, name, text string) (matchRank, bool) {
	r := rankMatch(q, name, text)
	return r, r != rankNone
}

func searchGems(in searchIn, env *calculator.Environment, group int) []hit {
	var out []hit
	compat := map[string]bool{}
	if env != nil && group > 0 {
		var list []gemEntry
		for _, g := range gems() {
			if g.Support {
				list = append(list, g)
			}
		}
		for i, ok := range supportCompat(env, group, list) {
			compat[list[i].Name] = ok
		}
	}
	for _, g := range gems() {
		if (in.Kind == "support") != g.Support {
			continue
		}
		if !hasAllTags(g.Tags, in.Tags) {
			continue
		}
		if in.MaxLevelReq != nil && g.LevelReq() > *in.MaxLevelReq {
			continue
		}
		r, ok := matchQuery(in.Query, g.Name, strings.Join(g.Tags, " "))
		if !ok {
			continue
		}
		out = append(out, hit{r, g.Name, func() map[string]any {
			id := g.GemID
			if g.VariantID != "" {
				id = g.VariantID
			}
			row := map[string]any{"name": g.Name, "id": id, "tags": g.Tags, "level_req": g.LevelReq(), "support": g.Support}
			if env != nil && group > 0 {
				row["compatible"] = compat[g.Name]
			}
			return row
		}})
	}
	return out
}

func hasAllTags(have, want []string) bool {
	for _, w := range want {
		if !slices.Contains(have, strings.ToLower(w)) {
			return false
		}
	}
	return true
}

func slotFilter(b *pob.PathOfBuilding, slot string) (string, *pob.PathOfBuilding, *ToolError) {
	if slot == "" {
		return "", b, nil
	}
	s, e := resolveSlot(slot)
	if e != nil {
		return "", nil, e
	}
	if b == nil {
		b = builds.EmptyBuild()
	}
	return s, b, nil
}

func itemText(item *calculator.Item) string {
	var parts []string
	for _, list := range [][]calculator.ModLine{item.ImplicitModLines, item.ExplicitModLines} {
		for _, ml := range list {
			parts = append(parts, ml.Line)
		}
	}
	return strings.Join(parts, " ")
}

func searchUniques(in searchIn, b *pob.PathOfBuilding) ([]hit, *ToolError) {
	slot, sb, e := slotFilter(b, in.Slot)
	if e != nil {
		return nil, e
	}
	var out []hit
	for _, it := range calculator.ItemDB(calculator.ItemDBUnique) {
		if slot != "" && !calculator.IsItemValidForSlot(sb, it, slot) {
			continue
		}
		lvl := int(it.Requirements["Level"])
		if in.MaxLevelReq != nil && lvl > *in.MaxLevelReq {
			continue
		}
		r, ok := matchQuery(in.Query, it.Name, it.BaseName+" "+itemText(it))
		if !ok {
			continue
		}
		out = append(out, hit{r, it.Name, func() map[string]any {
			return map[string]any{"name": it.Name, "base": it.BaseName, "slot": it.PrimarySlot(), "level_req": lvl, "key_mods": nonNil(keyMods(it, 4))}
		}})
	}
	return out, nil
}

func searchBases(in searchIn, b *pob.PathOfBuilding) ([]hit, *ToolError) {
	slot, sb, e := slotFilter(b, in.Slot)
	if e != nil {
		return nil, e
	}
	var out []hit
	for _, g := range calculator.CraftBaseGroups() {
		for _, name := range g.Bases {
			base := calculator.ItemBaseByName(name)
			if base == nil {
				continue
			}
			if in.MaxLevelReq != nil && base.Req.Level > *in.MaxLevelReq {
				continue
			}
			info := data.CraftBaseInfo(name)
			if slot != "" {
				text, ok := calculator.CraftItemText(0, "", name)
				if !ok || !calculator.IsItemValidForSlot(sb, calculator.ParseItem(text), slot) {
					continue
				}
			}
			r, ok := matchQuery(in.Query, name, g.Type+" "+info.Implicit)
			if !ok {
				continue
			}
			typ := g.Type
			out = append(out, hit{r, name, func() map[string]any {
				row := map[string]any{"name": name, "type": typ, "level_req": base.Req.Level}
				if info.Implicit != "" {
					row["implicits"] = strings.Split(info.Implicit, "\n")
				}
				if base.SocketLimit > 0 {
					row["max_sockets"] = base.SocketLimit
				}
				return row
			}})
		}
	}
	return out, nil
}

func searchNodes(in searchIn, b *pob.PathOfBuilding) ([]hit, *ToolError) {
	v := data.LatestTreeVersion
	if b != nil {
		v = specTreeVersion(b)
	}
	list, e := treeIndex(v)
	if e != nil {
		return nil, e
	}
	var plan *pathPlan
	alloc := map[int64]bool{}
	if b != nil {
		p := planPaths(b, v)
		plan = &p
		for _, id := range b.Build.PassiveNodes {
			alloc[id] = true
		}
	}
	var out []hit
	for _, n := range list {
		switch in.Kind {
		case "keystone", "mastery":
			if n.Type != in.Kind {
				continue
			}
		}
		if b != nil && n.Ascendancy != "" && n.Ascendancy != string(b.Build.AscendClassName) {
			continue
		}
		text := strings.Join(n.Stats, " ")
		for _, m := range n.Node.MasteryEffects {
			text += " " + strings.Join(m.Stats, " ")
		}
		r, ok := matchQuery(in.Query, n.Name, text)
		if !ok {
			continue
		}
		out = append(out, hit{r, n.Name, func() map[string]any {
			row := map[string]any{"id": n.ID, "name": n.Name, "type": n.Type}
			if len(n.Stats) > 0 {
				row["stats"] = n.Stats
			}
			if n.Type == "mastery" {
				var effs []map[string]any
				for _, m := range n.Node.MasteryEffects {
					effs = append(effs, map[string]any{"id": m.Effect, "text": strings.Join(m.Stats, " / ")})
				}
				row["effects"] = effs
			}
			if plan != nil {
				row["allocated"] = alloc[n.ID]
				if !alloc[n.ID] {
					if p := plan.path(n.ID); p != nil {
						row["path_cost"] = len(p)
					}
				}
			}
			return row
		}})
	}
	return out, nil
}

func searchAffixes(in searchIn) ([]hit, *ToolError) {
	if in.Base == "" {
		return nil, invalidArg("kind=affix needs base (an item base, e.g. Astral Plate).", nil)
	}
	base, e := resolveBase(in.Base)
	if e != nil {
		return nil, e
	}
	text, ok := calculator.CraftItemText(2, "", base)
	if !ok {
		return nil, errf(CodeInvalidValue, "%s cannot be crafted.", base)
	}
	ctl, ok := calculator.ItemAffixControls(text)
	if !ok {
		return nil, errf(CodeInvalidValue, "%s takes no crafted affixes.", base)
	}
	seen := map[string]bool{}
	var out []hit
	for _, c := range ctl.Controls {
		for k, ser := range c.Series {
			if k == 0 || seen[c.Type+ser.Label] {
				continue
			}
			seen[c.Type+ser.Label] = true
			r, ok := matchQuery(in.Query, ser.Label, strings.Join(ser.ModList, " "))
			if !ok {
				continue
			}
			typ := c.Type
			out = append(out, hit{r, ser.Label, func() map[string]any {
				return map[string]any{"label": ser.Label, "type": typ, "tiers": len(ser.ModList)}
			}})
		}
	}
	return out, nil
}

func searchConfig(in searchIn) []hit {
	var out []hit
	for _, o := range configOptions() {
		r := min(rankMatch(in.Query, o.cleanLabel(), o.Section), rankMatch(in.Query, o.Var, ""))
		if r == rankNone {
			continue
		}
		out = append(out, hit{r, o.cleanLabel(), func() map[string]any {
			row := map[string]any{"var": o.Var, "label": o.cleanLabel(), "type": o.Type}
			if len(o.List) > 0 {
				var vals []any
				for _, l := range o.List {
					vals = append(vals, l.Val)
				}
				row["values"] = vals
			}
			if d := o.defaultValue(); d != nil {
				row["default"] = d
			}
			return row
		}})
	}
	return out
}

func searchStats(in searchIn) []hit {
	var out []hit
	for _, k := range statKeys() {
		info := statTable()[k]
		r := min(rankMatch(in.Query, k, ""), rankMatch(in.Query, info.Label, ""))
		if r == rankNone {
			continue
		}
		out = append(out, hit{r, k, func() map[string]any {
			row := map[string]any{"key": info.Key, "label": info.Label}
			if info.Group != "" {
				row["group"] = info.Group
			}
			return row
		}})
	}
	return out
}
