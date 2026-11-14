package mcp

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

const (
	capSection       = 3 * 1024
	capSectionDetail = 12 * 1024
	capStats         = 2 * 1024
	capStatsDetail   = 6 * 1024
)

// fitBudget drops trailing entries of res[key] until res marshals within limit, and marks the result truncated.
func fitBudget(res map[string]any, key string, limit int, how string) {
	size := func() int {
		b, _ := json.Marshal(res)
		return len(b)
	}
	if size() <= limit {
		return
	}
	res["truncated"] = true
	res["how_to_get_more"] = how
	switch list := res[key].(type) {
	case []any:
		for len(list) > 0 && size() > limit {
			list = list[:len(list)-1]
			res[key] = list
		}
	case []map[string]any:
		for len(list) > 0 && size() > limit {
			list = list[:len(list)-1]
			res[key] = list
		}
	}
}

type getBuildIn struct {
	Handle  string `json:"handle"`
	Section string `json:"section"`
	Detail  bool   `json:"detail"`
}

var sections = []string{"summary", "tree", "items", "skills", "config", "notes", "warnings"}

func (s *Server) getBuild(args json.RawMessage) (any, *ToolError) {
	var in getBuildIn
	if e := decode(args, schemaGetBuild, &in); e != nil {
		return nil, e
	}
	if in.Handle == "" {
		var list []map[string]any
		for _, id := range s.order {
			h := s.handles[id]
			b, e := h.parseRev(h.cur)
			if e != nil {
				return nil, e
			}
			list = append(list, map[string]any{"handle": id, "name": h.name, "revision": h.cur, "latest_revision": h.latest(),
				"class": b.Build.ClassName, "ascendancy": ascendancyName(b), "level": b.Build.Level, "exported": h.exported})
		}
		return map[string]any{"handles": nonNil(list)}, nil
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
	limit := capSection
	if in.Detail {
		limit = capSectionDetail
	}
	section := in.Section
	if section == "" {
		section = "summary"
	}
	res := map[string]any{"handle": h.id, "revision": h.cur, "section": section}
	switch section {
	case "summary":
		sum, e := summaryOf(h, h.cur)
		if e != nil {
			return nil, e
		}
		for k, v := range sum {
			res[k] = v
		}
		res["bandit"] = b.Build.Bandit
		res["pantheon"] = map[string]string{"major": b.Build.PantheonMajorGod, "minor": b.Build.PantheonMinorGod}
		res["item_count"] = len(equippedItems(b))
		res["group_count"] = len(activeSkills(b))
		res["versions"] = h.versions
		fitBudget(res, "warnings", limit, "get_build section=warnings")
	case "tree":
		treeSection(res, b, env, in.Detail)
		fitBudget(res, "notables", limit, "get_build section=tree detail=true")
	case "items":
		itemsSection(res, b, in.Detail)
		fitBudget(res, "items", limit, "get_build section=items detail=true")
	case "skills":
		res["groups"] = skillsSection(b)
		fitBudget(res, "groups", limit, "get_build section=skills detail=true")
	case "config":
		res["config"] = configSection(b)
		fitBudget(res, "config", limit, "get_build section=config detail=true")
	case "notes":
		notes := b.Notes
		if len(notes) > limit-200 {
			notes = notes[:limit-200]
			res["truncated"] = true
			if !in.Detail {
				res["how_to_get_more"] = "get_build section=notes detail=true"
			}
		}
		res["notes"] = notes
	case "warnings":
		res["warnings"] = toAny(nonNil(cachedWarnings(env)))
		fitBudget(res, "warnings", limit, "get_build section=warnings detail=true")
	default:
		return nil, invalidArg(fmt.Sprintf("Unknown section %q.", section), sections)
	}
	return res, nil
}

func treeSection(res map[string]any, b *pob.PathOfBuilding, env *calculator.Environment, detail bool) {
	v := specTreeVersion(b)
	selected := map[int64]int64{}
	if spec := activeSpec(b); spec != nil {
		for _, m := range pob.SpecMasterySelections(*spec) {
			selected[m.Node] = m.Effect
		}
	}
	sockets := map[int64]int{}
	if spec := activeSpec(b); spec != nil {
		for _, sk := range spec.Sockets {
			sockets[int64(sk.NodeID)] = sk.ItemID
		}
	}
	texts := map[int]string{}
	for _, it := range b.Items.Items {
		texts[it.ID] = it.Text
	}
	var keystones, notables, masteries, asc, jewels, all []any
	small := 0
	ids := slices.Clone(b.Build.PassiveNodes)
	slices.Sort(ids)
	for _, id := range ids {
		n, ok := treeNodeByID(v, id)
		if !ok {
			continue
		}
		all = append(all, id)
		row := map[string]any{"id": id, "name": n.Name}
		switch n.Type {
		case "keystone":
			keystones = append(keystones, row)
		case "notable":
			if detail {
				row["stats"] = n.Stats
			}
			notables = append(notables, row)
		case "mastery":
			if eff, ok := selected[id]; ok {
				for _, m := range n.Node.MasteryEffects {
					if m.Effect == eff {
						row["effect"] = strings.Join(m.Stats, " / ")
					}
				}
			}
			masteries = append(masteries, row)
		case "ascendancy":
			asc = append(asc, row)
		case "jewel":
			if itemID, ok := sockets[id]; ok && itemID != 0 {
				row["item"] = calculator.PeekItem(texts[itemID]).Name
			}
			delete(row, "name")
			jewels = append(jewels, row)
		default:
			small++
		}
	}
	res["class"] = b.Build.ClassName
	res["ascendancy"] = ascendancyName(b)
	res["points"] = pointsOf(env)
	res["keystones"] = nonNil(keystones)
	res["notables"] = nonNil(notables)
	res["masteries"] = nonNil(masteries)
	res["ascendancy_nodes"] = nonNil(asc)
	res["jewel_sockets"] = nonNil(jewels)
	res["small_node_count"] = small
	res["tree_version"] = string(v)
	if detail {
		res["node_ids"] = all
	}
}

var keyModWords = []string{"maximum life", "resistance", "to all attributes", "to strength", "to dexterity", "to intelligence", "energy shield", "damage", "maximum mana", "speed", "critical"}

// keyMods ranks an item's visible mod lines life, resistances, attributes, then damage, and keeps max of them.
func keyMods(item *calculator.Item, maxLines int) []string {
	type ranked struct {
		line string
		rank int
		pos  int
	}
	var lines []ranked
	pos := 0
	for _, list := range [][]calculator.ModLine{item.ImplicitModLines, item.ExplicitModLines} {
		for _, ml := range list {
			if !item.ItemModLineVisible(ml) {
				continue
			}
			text, _, ok := calculator.FormatModLine(ml)
			if !ok || text == "" {
				text = ml.Line
			}
			r := len(keyModWords)
			lower := strings.ToLower(text)
			for i, w := range keyModWords {
				if strings.Contains(lower, w) {
					r = i
					break
				}
			}
			lines = append(lines, ranked{text, r, pos})
			pos++
		}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].rank < lines[j].rank })
	var out []string
	for _, l := range lines {
		if len(out) == maxLines {
			break
		}
		out = append(out, l.line)
	}
	return out
}

func socketString(item *calculator.Item) string {
	var b strings.Builder
	for i, s := range item.Sockets {
		if i > 0 {
			if s.Group == item.Sockets[i-1].Group {
				b.WriteByte('-')
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteString(s.Color)
	}
	return b.String()
}

func itemRow(slot string, id int, text string, detail bool) map[string]any {
	item := calculator.ParseItem(text)
	row := map[string]any{"slot": slot, "item_id": id, "name": item.Name, "base": item.BaseName, "rarity": item.Rarity}
	if detail {
		row["text"] = text
	} else {
		row["key_mods"] = nonNil(keyMods(item, 4))
	}
	if s := socketString(item); s != "" {
		row["sockets"] = s
	}
	return row
}

func itemsSection(res map[string]any, b *pob.PathOfBuilding, detail bool) {
	list := equippedItems(b)
	rows := make([]any, 0, len(list))
	used := map[string]bool{}
	equippedIDs := map[int]bool{}
	for _, e := range list {
		row := itemRow(e.slot, e.id, e.text, detail)
		if strings.HasPrefix(e.slot, "Flask") {
			row["active"] = e.active
		}
		rows = append(rows, row)
		used[e.slot] = true
		equippedIDs[e.id] = true
	}
	var empty []string
	for _, s := range []string{"Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring 1", "Ring 2", "Belt"} {
		if !used[s] {
			empty = append(empty, s)
		}
	}
	res["items"] = nonNil(rows)
	res["empty_slots"] = nonNil(empty)
	if detail {
		var spare []any
		for _, it := range b.Items.Items {
			if !equippedIDs[it.ID] {
				spare = append(spare, map[string]any{"item_id": it.ID, "name": calculator.PeekItem(it.Text).Name})
			}
		}
		res["unequipped"] = nonNil(spare)
	}
}

func skillsSection(b *pob.PathOfBuilding) []any {
	groups := activeSkills(b)
	out := make([]any, 0, len(groups))
	for i, g := range groups {
		var gems []any
		for k, gem := range g.Gems {
			gems = append(gems, map[string]any{"index": k + 1, "name": gem.NameSpec, "level": gem.Level, "quality": gem.Quality,
				"support": gemIsSupport(gem), "enabled": gem.Enabled})
		}
		row := map[string]any{"group": i + 1, "enabled": g.Enabled, "main": b.Build.MainSocketGroup == i+1, "gems": nonNil(gems),
			"in_full_dps": g.IncludeInFullDPS != nil && *g.IncludeInFullDPS}
		if g.Slot != "" {
			row["slot"] = g.Slot
		}
		if g.Label != "" {
			row["label"] = g.Label
		}
		out = append(out, row)
	}
	return nonNil(out)
}

func configSection(b *pob.PathOfBuilding) []any {
	labels := map[string]string{}
	for _, o := range configOptions() {
		labels[o.Var] = o.cleanLabel()
	}
	out := make([]any, 0, len(b.Config.Inputs))
	for _, in := range b.Config.Inputs {
		var v any
		switch {
		case in.Boolean != nil:
			v = *in.Boolean
		case in.Number != nil:
			v = *in.Number
		case in.String != nil:
			v = *in.String
		}
		out = append(out, map[string]any{"var": in.Name, "label": labels[in.Name], "value": v})
	}
	return nonNil(out)
}

type getStatsIn struct {
	Handle   string   `json:"handle"`
	Stats    []string `json:"stats"`
	Groups   []string `json:"groups"`
	Explain  string   `json:"explain"`
	Revision *int     `json:"revision"`
}

func (s *Server) getStats(args json.RawMessage) (any, *ToolError) {
	var in getStatsIn
	if e := decode(args, schemaGetStats, &in); e != nil {
		return nil, e
	}
	h, e := s.get(in.Handle)
	if e != nil {
		return nil, e
	}
	rev := h.cur
	if in.Revision != nil {
		rev = *in.Revision
	}
	env, e := h.envAt(rev)
	if e != nil {
		return nil, e
	}
	b, e := h.parseRev(rev)
	if e != nil {
		return nil, e
	}
	keys := slices.Clone(in.Stats)
	for _, g := range in.Groups {
		list, ok := statGroups[g]
		if !ok {
			return nil, invalidArg(fmt.Sprintf("Unknown group %q.", g), groupNames)
		}
		keys = append(keys, list...)
	}
	if len(keys) == 0 && in.Explain == "" {
		keys = summaryKeys
	}
	stats := map[string]float64{}
	var unknown []map[string]any
	for _, k := range keys {
		if v, ok := statValue(env, k); ok {
			stats[k] = round(v)
		} else if _, known := statTable()[k]; known {
			stats[k] = 0
		} else {
			unknown = append(unknown, map[string]any{"key": k, "did_you_mean": suggest(k, statKeys(), 3)})
		}
	}
	res := map[string]any{"handle": h.id, "revision": rev, "stats": stats}
	if ms := mainSkillOf(b); ms != nil {
		res["main_skill"] = ms.Name
	}
	if len(unknown) > 0 {
		res["unknown"] = unknown
	}
	if in.Explain != "" {
		res["explain"] = explainStat(env, in.Explain)
	}
	limit := capStats
	if len(in.Stats) > 0 || len(in.Groups) > 1 {
		limit = capStatsDetail
	}
	if b, _ := json.Marshal(res); len(b) > limit {
		trimStats(res, limit)
	}
	return res, nil
}

func trimStats(res map[string]any, limit int) {
	stats := res["stats"].(map[string]float64)
	keys := make([]string, 0, len(stats))
	for k := range stats {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for len(keys) > 0 {
		if b, _ := json.Marshal(res); len(b) <= limit {
			break
		}
		delete(stats, keys[len(keys)-1])
		keys = keys[:len(keys)-1]
	}
	res["truncated"] = true
	res["how_to_get_more"] = "ask for fewer stats or groups per call"
}

// explainStat summarises the stat's mod sources as PoB's breakdown tabulates them: base, inc and more totals and
// the eight largest sources.
func explainStat(env *calculator.Environment, key string) map[string]any {
	v, _ := statValue(env, key)
	out := map[string]any{"stat": key, "value": round(v)}
	modNames := []string{strings.TrimPrefix(key, "Minion")}
	if info, ok := statTable()[key]; ok && len(info.ModNames) > 0 {
		modNames = info.ModNames
	}
	if key == mainDPS {
		out["note"] = "MainDPS is CombinedDPS of the main skill plus its minions' CombinedDPS; explain Damage or Speed for sources."
		return out
	}
	minion := strings.HasPrefix(key, "Minion") && env.Minion != nil
	type src struct {
		source string
		text   string
		mag    float64
	}
	var sources []src
	var base, inc float64
	more := 1.0
	for _, t := range []string{"BASE", "INC", "MORE"} {
		rows := env.CalcsModSection(calculator.ModSectionData{ModName: modNames, ModType: t}, minion)
		for _, r := range rows {
			var val float64
			_, _ = fmt.Sscan(strings.TrimPrefix(r.Value, "+"), &val)
			switch t {
			case "BASE":
				base += val
			case "INC":
				inc += val
			case "MORE":
				more *= 1 + val/100
			}
			name := stripColor(r.SourceName)
			if name == "" {
				name = r.Source
			}
			if r.Source != "" && name != r.Source && !strings.HasPrefix(name, r.Source) {
				name = r.Source + ": " + name
			}
			unit := map[string]string{"BASE": " base", "INC": "% inc", "MORE": "% more"}[t]
			text := r.Value + unit
			if r.Name != "" && len(modNames) > 1 {
				text += " (" + r.Name + ")"
			}
			sources = append(sources, src{name, text, math.Abs(val) * map[string]float64{"BASE": 1, "INC": 10, "MORE": 30}[t]})
		}
	}
	sort.SliceStable(sources, func(i, j int) bool { return sources[i].mag > sources[j].mag })
	var list []map[string]string
	for _, s := range sources[:min(len(sources), 8)] {
		list = append(list, map[string]string{"source": s.source, "value": s.text})
	}
	out["base"] = round(base)
	out["inc"] = round(inc)
	out["more"] = math.Round(more*1000) / 1000
	out["sources"] = nonNil(list)
	if len(sources) > 8 {
		out["more_sources"] = len(sources) - 8
	}
	return out
}
