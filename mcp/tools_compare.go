package mcp

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/pob"
)

const capCompare = 3 * 1024

type compareIn struct {
	A         string   `json:"a"`
	B         string   `json:"b"`
	Stats     []string `json:"stats"`
	Structure *bool    `json:"structure"`
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
			na, _, _ := strings.Cut(itemsA[slot], "\x00")
			nb, _, _ := strings.Cut(itemsB[slot], "\x00")
			items = append(items, map[string]any{"slot": slot, "a": na, "b": nb})
		}
	}
	ga, gb := activeSkills(a), activeSkills(b)
	var gemsChanged []map[string]any
	for i := range max(len(ga), len(gb)) {
		var na, nb []string
		if i < len(ga) {
			for _, g := range ga[i].Gems {
				na = append(na, g.NameSpec)
			}
		}
		if i < len(gb) {
			for _, g := range gb[i].Gems {
				nb = append(nb, g.NameSpec)
			}
		}
		rem, add := setDiff(na, nb), setDiff(nb, na)
		if len(rem)+len(add) > 0 {
			gemsChanged = append(gemsChanged, map[string]any{"group": i + 1, "removed": rem, "added": add})
		}
	}
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
		"gems_changed": nonNil(gemsChanged), "config_changed": nonNil(cfg), "level": []int{a.Build.Level, b.Build.Level},
	}
}

func orDefault(s string) string {
	if s == "" {
		return "default"
	}
	return s
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
