package mcp

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

var resistLineRe = regexp.MustCompile(`^\+(\d+(?:\.\d+)?)% to (.+?) Resistances?$`)

// resVec is resistance by element, in resistElems order.
type resVec [4]float64

// resistOf is what a mod's lines, rolled at rng, add to each resistance.
func resistOf(lines []string, rng float64) resVec {
	var v resVec
	for _, l := range lines {
		m := resistLineRe.FindStringSubmatch(data.ApplyRange(l, rng, nil, nil))
		if m == nil {
			continue
		}
		n, _ := strconv.ParseFloat(m[1], 64)
		elems := strings.Split(m[2], " and ")
		if m[2] == "all Elemental" {
			elems = []string{"Fire", "Cold", "Lightning"}
		}
		for _, e := range elems {
			if k := slices.Index(resistElems, e); k >= 0 {
				v[k] += n
			}
		}
	}
	return v
}

type resTier struct {
	src, id string
	lines   []string
	v       resVec
}

// resOption is one resistance group an item can still take, its tiers the item level allows, best first.
type resOption struct {
	group string
	bench bool
	tiers []resTier
}

// resItem is an equipped item's room for resistance suffixes: open suffixes, crafted mods it can still take, and the
// resistance groups it does not have.
type resItem struct {
	slot      string
	text      string
	open      int
	benchRoom int
	opts      []resOption
}

// resistItems are the equipped rare and magic gear items with an open suffix and a resistance suffix to fill it,
// from PoB's "Add modifier" suffix list and the crafting bench's, as affixCandidates offers them.
func resistItems(b *pob.PathOfBuilding, rng float64) []resItem {
	var out []resItem
	for _, e := range equippedItems(b) {
		if !slices.Contains(gearSlots, e.slot) || (e.rarity != string(calculator.RarityRare) && e.rarity != string(calculator.RarityMagic)) {
			continue
		}
		item := calculator.ParseItem(e.text)
		if item.Base == nil {
			continue
		}
		a := countAffixes(item)
		ri := resItem{slot: e.slot, text: e.text, open: a.limit - a.suf, benchRoom: a.craftLimit - a.crafted}
		if ri.open <= 0 {
			continue
		}
		choices, ok := calculator.ItemCustomModChoices(modChoicesText(e.text))
		if !ok {
			continue
		}
		byGroup := map[string]*resOption{}
		var order []string
		add := func(m data.AffixMod, src, id string, bench bool) {
			if m.Type != "Suffix" || a.groups[m.Group] || m.Level > a.ilvl {
				return
			}
			v := resistOf(m.Lines, rng)
			if v == (resVec{}) {
				return
			}
			key := m.Group + map[bool]string{true: " bench"}[bench]
			if byGroup[key] == nil {
				byGroup[key] = &resOption{group: m.Group, bench: bench}
				order = append(order, key)
			}
			byGroup[key].tiers = append(byGroup[key].tiers, resTier{src, id, m.Lines, v})
		}
		pool := map[string]data.AffixMod{}
		for _, m := range affixPool(item) {
			pool[m.ID] = m
		}
		for _, cm := range choices.Mods["SUFFIX"] {
			if m, ok := pool[cm.ID]; ok {
				add(m, "SUFFIX", m.ID, false)
			}
		}
		if ri.benchRoom > 0 {
			masters := data.MasterMods()
			for _, cm := range choices.Mods["MASTER"] {
				if i, err := strconv.Atoi(strings.TrimPrefix(cm.ID, "master:")); err == nil && i >= 0 && i < len(masters) {
					add(masters[i].AffixMod, "MASTER", cm.ID, true)
				}
			}
		}
		for _, k := range order {
			o := byGroup[k]
			slices.SortStableFunc(o.tiers, func(x, y resTier) int { return sumVec(y.v) - sumVec(x.v) })
			ri.opts = append(ri.opts, *o)
		}
		if len(ri.opts) > 0 {
			out = append(out, ri)
		}
	}
	return out
}

func sumVec(v resVec) int {
	return int(v[0] + v[1] + v[2] + v[3])
}

// resPick is one resistance mod of a plan: item, option and tier indexes.
type resPick struct{ item, opt, tier int }

// solveResists is the fewest resistance suffixes the items' open slots take to cover need, each item taking one mod
// per group, its open suffixes at most and its bench room in crafted mods, each mod at the lowest tier that still
// covers; among plans of that size, the one with the least excess. It reports false when no plan covers need or the
// search ran past its budget.
func solveResists(items []resItem, need resVec) ([]resPick, bool) {
	if need == (resVec{}) {
		return nil, true
	}
	// a type is a group and its best tier's resistances; plans pick types, then assign them to items
	type typ struct {
		group string
		bench bool
		v     resVec
	}
	var types []typ
	for _, it := range items {
		for _, o := range it.opts {
			t := typ{o.group, o.bench, o.tiers[0].v}
			if !slices.Contains(types, t) {
				types = append(types, t)
			}
		}
	}
	offers := func(it resItem, t typ) int {
		return slices.IndexFunc(it.opts, func(o resOption) bool { return o.group == t.group && o.bench == t.bench && o.tiers[0].v == t.v })
	}
	slots := 0
	for _, it := range items {
		slots += it.open
	}
	budget := 2_000_000
	// assign places the counted types on items: open suffixes, bench room and one mod per group per item
	assign := func(counts []int) ([]resPick, bool) {
		var want []int
		for t, n := range counts {
			for range n {
				want = append(want, t)
			}
		}
		used := make([]int, len(items))
		bench := make([]int, len(items))
		groups := make([]map[string]bool, len(items))
		for i := range groups {
			groups[i] = map[string]bool{}
		}
		picks := make([]resPick, len(want))
		var place func(k int) bool
		place = func(k int) bool {
			if k == len(want) {
				return true
			}
			t := types[want[k]]
			for i, it := range items {
				o := offers(it, t)
				if o < 0 || used[i] >= it.open || groups[i][t.group] || (t.bench && bench[i] >= it.benchRoom) {
					continue
				}
				used[i]++
				groups[i][t.group] = true
				if t.bench {
					bench[i]++
				}
				picks[k] = resPick{i, o, 0}
				if place(k + 1) {
					return true
				}
				used[i]--
				delete(groups[i], t.group)
				if t.bench {
					bench[i]--
				}
			}
			return false
		}
		return picks, place(0)
	}
	for size := 1; size <= slots; size++ {
		var best []resPick
		bestExcess := 0.0
		counts := make([]int, len(types))
		var walk func(t, left int, have resVec)
		walk = func(t, left int, have resVec) {
			if budget--; budget < 0 {
				return
			}
			covered := true
			for e := range need {
				if have[e] < need[e] {
					covered = false
				}
			}
			if left == 0 || t == len(types) {
				if !covered || left != 0 {
					return
				}
				excess := 0.0
				for e := range need {
					excess += have[e] - need[e]
				}
				if best != nil && excess >= bestExcess {
					return
				}
				if picks, ok := assign(counts); ok {
					best, bestExcess = picks, excess
				}
				return
			}
			for e := range need {
				reach := 0.0
				for _, o := range types[t:] {
					reach = max(reach, o.v[e])
				}
				if have[e]+reach*float64(left) < need[e] {
					return
				}
			}
			for n := left; n >= 0; n-- {
				counts[t] = n
				next := have
				for e := range next {
					next[e] += float64(n) * types[t].v[e]
				}
				walk(t+1, left-n, next)
			}
			counts[t] = 0
		}
		walk(0, size, resVec{})
		if budget < 0 {
			return nil, false
		}
		if best != nil {
			lowerTiers(items, best, need)
			return best, true
		}
	}
	return nil, false
}

// lowerTiers moves each pick to the lowest tier at which the plan still covers need.
func lowerTiers(items []resItem, picks []resPick, need resVec) {
	total := func() resVec {
		var v resVec
		for _, p := range picks {
			for e, x := range items[p.item].opts[p.opt].tiers[p.tier].v {
				v[e] += x
			}
		}
		return v
	}
	for k := range picks {
		tiers := items[picks[k].item].opts[picks[k].opt].tiers
		for t := len(tiers) - 1; t > picks[k].tier; t-- {
			was := picks[k].tier
			picks[k].tier = t
			v := total()
			if v[0] >= need[0] && v[1] >= need[1] && v[2] >= need[2] && v[3] >= need[3] {
				break
			}
			picks[k].tier = was
		}
	}
}

// resistNeed is the resistance each element is short of its cap by, chaos only with chaos.
func resistNeed(room map[string]float64, chaos bool) resVec {
	var need resVec
	for e, el := range resistElems {
		if el == "Chaos" && !chaos {
			continue
		}
		need[e] = max(0, -room[el])
	}
	return need
}

// resistPlanTexts are the item texts a plan gives its slots, its mods added in pick order at rng.
func resistPlanTexts(items []resItem, picks []resPick, rng float64) (map[string]string, bool) {
	texts := map[string]string{}
	for _, p := range picks {
		it := items[p.item]
		text, ok := texts[it.slot]
		if !ok {
			text = it.text
		}
		tier := it.opts[p.opt].tiers[p.tier]
		next, ok := addModText(text, tier.src, tier.id, rng)
		if !ok {
			return nil, false
		}
		texts[it.slot] = next
	}
	return texts, true
}

func resistPlanChange(texts map[string]string) change {
	return func(c *pob.PathOfBuilding) {
		for _, slot := range slices.Sorted(maps.Keys(texts)) {
			calculator.EquipItem(c, texts[slot], slot)
		}
	}
}

// resistPlan builds the cheapest plan that caps the build's resistances from its open suffixes, as items and picks.
func resistPlan(b *pob.PathOfBuilding, room map[string]float64, rng float64, chaos bool) ([]resItem, []resPick, bool) {
	items := resistItems(b, rng)
	picks, ok := solveResists(items, resistNeed(room, chaos))
	return items, picks, ok
}

func resistPlanID(rng float64, chaos bool) string {
	return "resists|" + strconv.FormatFloat(rng, 'f', -1, 64) + "|" + strconv.FormatBool(chaos)
}

// resists reports the plan that caps the build's elemental resistances (chaos too with cap_chaos) with the fewest
// resistance suffixes in its open affix slots, each at the lowest tier that still caps, checked by a calculation:
// needed is the deficit before the plan, short_by what is left after it.
func (sc *scanCtx) resists() map[string]any {
	need := resistNeed(sc.room, sc.in.CapChaos)
	out := map[string]any{"short_by": vecMap(need)}
	if need == (resVec{}) {
		out["note"] = "every resistance asked for is capped"
		return out
	}
	items, picks, ok := resistPlan(sc.b, sc.room, sc.roll.rng, sc.in.CapChaos)
	if !ok {
		open := 0
		for _, it := range items {
			open += it.open
		}
		out["note"] = fmt.Sprintf("no set of resistance suffixes in the %d open suffix slots caps them", open)
		return out
	}
	texts, ok := resistPlanTexts(items, picks, sc.roll.rng)
	if !ok {
		out["note"] = "the plan's mods do not fit the items' texts"
		return out
	}
	ch := resistPlanChange(texts)
	after := evalChange(sc.b, ch)
	var mods []any
	for _, p := range picks {
		mods = append(mods, map[string]any{"slot": items[p.item].slot, "mod": pickLabel(items, p, sc.roll)})
	}
	out["mods"] = mods
	out["suffixes"] = len(picks)
	out["needed"] = out["short_by"]
	out["short_by"] = vecMap(resistNeed(headroom(after), sc.in.CapChaos))
	out["headroom_after"] = headroom(after)
	out["gain"] = round(statOf(after, sc.key) - sc.base)
	out["id"] = resistPlanID(sc.roll.rng, sc.in.CapChaos)
	return out
}

func pickLabel(items []resItem, p resPick, roll affixRoll) string {
	tier := items[p.item].opts[p.opt].tiers[p.tier]
	label := roll.label(tier.lines, p.tier+1)
	if tier.src == "MASTER" {
		label += " bench"
	}
	return label
}

func vecMap(v resVec) map[string]float64 {
	out := map[string]float64{}
	for e, el := range resistElems {
		if v[e] > 0 {
			out[el] = v[e]
		}
	}
	return out
}

// refill is the resistance suffixes in the build's open slots that restore the caps a row's change breaks, rolled as
// roll says; the row's own slot loses its open suffixes to a unique and one to a suffix it adds.
func refill(items []resItem, room map[string]float64, row map[string]any, roll affixRoll) any {
	gap, _ := row["resist_gap"].(map[string]float64)
	after := map[string]float64{}
	for k, v := range room {
		after[k] = v
	}
	for k, v := range gap {
		after[k] = v
	}
	left := make([]resItem, 0, len(items))
	for _, it := range items {
		if it.slot == row["slot"] {
			switch {
			case row["part"] == "uniques":
				continue
			case row["type"] == "Suffix":
				it.open--
			}
		}
		left = append(left, it)
	}
	picks, ok := solveResists(left, resistNeed(after, false))
	if !ok {
		return "not from the open suffix slots"
	}
	mods := make([]string, len(picks))
	for k, p := range picks {
		mods[k] = left[p.item].slot + ": " + pickLabel(left, p, roll)
	}
	return map[string]any{"suffixes": len(picks), "mods": mods}
}
