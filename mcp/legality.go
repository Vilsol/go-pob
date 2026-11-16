package mcp

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

// itemFlag is an equipped item the game could not produce, with the rule it breaks and the item sets that equip it.
type itemFlag struct {
	Sets []string `json:"sets"`
	Slot string   `json:"slot"`
	Item string   `json:"item"`
	Rule string   `json:"rule"`
	Note string   `json:"note"`
}

// itemLegalityFlags lists the items any item set equips, or any passive tree's jewel sockets hold, whose affixes, crafts, values or implicits the game does not
// allow: more prefixes or suffixes than the rarity holds (Item.lua:1478-1491), more explicit modifiers than both
// together hold, more crafted modifiers than the item allows, an explicit value above every mod it can be, or an
// implicit outside its base's range. Each set and tree is named as its title, the active one marked. Socket colours are never
// illegal: any gem fits any socket, a matching colour only adds quality (scan_build's sockets part ranks recolouring).
func itemLegalityFlags(b *pob.PathOfBuilding) []itemFlag {
	var out []itemFlag
	audit := func(name string, items []equipped) {
		for _, e := range items {
			for _, f := range itemFlagsOf(e) {
				i := slices.IndexFunc(out, func(o itemFlag) bool {
					return o.Slot == f.Slot && o.Item == f.Item && o.Rule == f.Rule && o.Note == f.Note
				})
				if i < 0 {
					f.Sets = []string{name}
					out = append(out, f)
				} else if !slices.Contains(out[i].Sets, name) {
					out[i].Sets = append(out[i].Sets, name)
				}
			}
		}
	}
	setName := func(kind string, title *string, active bool) string {
		t := "Default"
		if title != nil && *title != "" {
			t = *title
		}
		name := fmt.Sprintf("%s %q", kind, t)
		if active {
			name += " (active)"
		}
		return name
	}
	for _, set := range b.Items.ItemSets {
		audit(setName("item set", set.Title, set.ID == fmt.Sprint(b.Items.ActiveItemSet)), setItems(b, set))
	}
	texts := make(map[int]string, len(b.Items.Items))
	for _, it := range b.Items.Items {
		texts[it.ID] = it.Text
	}
	for i, spec := range b.Tree.Specs {
		var jewels []equipped
		for _, sk := range spec.Sockets {
			if text, ok := texts[sk.ItemID]; ok && sk.ItemID != 0 {
				jewels = append(jewels, equipped{slot: "jewel socket " + strconv.Itoa(sk.NodeID), id: sk.ItemID, text: calculator.PlainItemText(text)})
			}
		}
		audit(setName("tree", spec.Title, i == b.ActiveSpecIndex()), jewels)
	}
	return out
}

func itemFlagsOf(e equipped) []itemFlag {
	var out []itemFlag
	item := calculator.ParseItem(e.text)
	if item.Base == nil {
		return nil
	}
	add := func(rule, note string) {
		out = append(out, itemFlag{Slot: e.slot, Item: item.Name, Rule: rule, Note: note})
	}
	if item.Rarity == calculator.RarityMagic || item.Rarity == calculator.RarityRare {
		a := countAffixes(item)
		lb := affixLowerBound(e.text, item)
		rarity := strings.ToLower(string(item.Rarity))
		if lb.pre > a.limit {
			add("prefixes", fmt.Sprintf("at least %d prefixes; a %s item holds %d", lb.pre, rarity, a.limit))
		}
		if lb.suf > a.limit {
			add("suffixes", fmt.Sprintf("at least %d suffixes; a %s item holds %d", lb.suf, rarity, a.limit))
		}
		if n := lb.pre + lb.suf + lb.untyped; n > 2*a.limit && lb.pre <= a.limit && lb.suf <= a.limit {
			add("modifiers", fmt.Sprintf("at least %d explicit modifiers; a %s item holds %d", n, rarity, 2*a.limit))
		}
		if n := benchCrafts(item); n > a.craftLimit {
			note := fmt.Sprintf("%d crafted modifiers; the bench allows one", n)
			if a.craftLimit > 1 {
				note = fmt.Sprintf("%d crafted modifiers; %q allows %d", n, multiCraftLine, a.craftLimit)
			}
			add("crafted", note)
		}
		for _, n := range explicitRangeNotes(e.text, item) {
			add("value", n)
		}
	}
	if !item.IsUnique {
		for _, n := range implicitRangeNotes(item) {
			add("implicit", n)
		}
	}
	return out
}

// benchCrafts is the fewest bench crafts the item's crafted explicit lines can be: the lines a multi-line bench mod for
// the base has, in any order and at any roll (an older craft's values), count as one craft, every other crafted line
// as one.
func benchCrafts(item *calculator.Item) int {
	var lines []string
	for _, ml := range item.ExplicitModLines {
		if ml.Crafted {
			lines = append(lines, affixTemplate(ml.Line))
		}
	}
	var hybrids [][]string
	for _, m := range data.MasterMods() {
		if len(m.Lines) > 1 && slices.Contains(m.Types, item.Base.Type) {
			h := make([]string, len(m.Lines))
			for k, l := range m.Lines {
				h[k] = affixTemplate(l)
			}
			hybrids = append(hybrids, h)
		}
	}
	slices.SortStableFunc(hybrids, func(a, b []string) int { return len(b) - len(a) })
	n := 0
	for _, h := range hybrids {
		for {
			left := slices.Clone(lines)
			for _, t := range h {
				if k := slices.Index(left, t); k >= 0 {
					left = slices.Delete(left, k, k+1)
				}
			}
			if len(lines)-len(left) != len(h) {
				break
			}
			lines = left
			n++
		}
	}
	return n + len(lines)
}

// implicitRangeNotes names the item's implicit lines with a value above the range its base's implicit rolls in the
// current game data. A ring, amulet or belt with no catalyst recorded may hold catalyst-scaled values, so up to 20%
// above is allowed there; corruption-scaled lines are left out.
func implicitRangeNotes(item *calculator.Item) []string {
	base, ok := data.CraftBaseKnown(item.Base.Name)
	if !ok || base.Implicit == "" {
		return nil
	}
	allow := 1.0
	if item.CatalystQuality == nil && slices.Contains([]string{"Ring", "Amulet", "Belt"}, item.Base.Type) {
		allow = 1.2
	}
	var out []string
	for _, line := range strings.Split(base.Implicit, "\n") {
		t := affixTemplate(line)
		bounds := modValueRe.FindAllStringSubmatch(line, -1)
		for _, ml := range item.ImplicitModLines {
			if affixTemplate(ml.Line) != t || ml.CorruptedRange != nil {
				continue
			}
			text := rolledLine(ml)
			values := lineValueRe.FindAllString(text, -1)
			if len(values) != len(bounds) {
				continue
			}
			for k, b := range bounds {
				hi := b[3]
				if hi == "" {
					hi = b[2]
				}
				v, _ := strconv.ParseFloat(values[k], 64)
				h, _ := strconv.ParseFloat(hi, 64)
				if v > h*allow {
					out = append(out, fmt.Sprintf("implicit %q is above the base's %q", text, line))
					break
				}
			}
		}
	}
	return out
}

// affixLowerBound is the fewest prefixes and suffixes the item's explicit lines can take, from the mods PoB's "Add
// modifier" offers the base (ItemsTab.lua:3612-3749: its affixes, essence, veiled, Delve and other mods; the bench's
// for crafted lines). The lines a multi-line mod can hold together count once, then a {prefix} or {suffix} line counts
// by its tag and an untagged line when every one-line mod it can be has one type. A line no mod matches is not
// counted, and grouping only ever lowers the count. A {custom} line counted as neither is untyped: still one modifier.
func affixLowerBound(text string, item *calculator.Item) affixBound {
	type group struct {
		typ   string
		lines []string
	}
	choices, ok := calculator.ItemCustomModChoices(modChoicesText(text))
	if !ok {
		return affixBound{}
	}
	var multi []group
	single := [2]map[string]map[string]bool{{}, {}}
	seen := map[string]bool{}
	for src, mods := range choices.Mods {
		crafted := 0
		if src == "MASTER" {
			crafted = 1
		}
		for _, m := range mods {
			typ := map[string]string{"PREFIX": "Prefix", "SUFFIX": "Suffix"}[src]
			switch {
			case typ != "":
			case strings.HasSuffix(m.Label, "(Prefix)"):
				typ = "Prefix"
			case strings.HasSuffix(m.Label, "(Suffix)"):
				typ = "Suffix"
			default:
				typ = "?"
			}
			if len(m.Lines) == 1 {
				t := affixTemplate(m.Lines[0])
				if single[crafted][t] == nil {
					single[crafted][t] = map[string]bool{}
				}
				single[crafted][t][typ] = true
				continue
			}
			g := group{typ: typ}
			for _, l := range m.Lines {
				g.lines = append(g.lines, affixTemplate(l))
			}
			g.lines = append(g.lines, strconv.Itoa(crafted))
			if key := fmt.Sprint(g); !seen[key] {
				seen[key] = true
				multi = append(multi, g)
			}
		}
	}
	slices.SortFunc(multi, func(a, b group) int {
		if d := len(b.lines) - len(a.lines); d != 0 {
			return d
		}
		return strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
	})
	lines := item.ExplicitModLines
	used := make([]bool, len(lines))
	tag := func(ml calculator.ModLine) string {
		switch {
		case ml.Prefix:
			return "Prefix"
		case ml.Suffix:
			return "Suffix"
		}
		return ""
	}
	count := map[string]int{}
	for _, g := range multi {
		crafted := g.lines[len(g.lines)-1] == "1"
		var pick []int
		for _, t := range g.lines[:len(g.lines)-1] {
			at := -1
			for k, ml := range lines {
				if !used[k] && !slices.Contains(pick, k) && ml.Crafted == crafted && affixTemplate(ml.Line) == t && (tag(ml) == "" || tag(ml) == g.typ) {
					at = k
					break
				}
			}
			if at < 0 {
				pick = nil
				break
			}
			pick = append(pick, at)
		}
		if pick == nil {
			continue
		}
		for _, k := range pick {
			used[k] = true
		}
		count[g.typ]++
	}
	for k, ml := range lines {
		if used[k] {
			continue
		}
		if t := tag(ml); t != "" {
			count[t]++
			continue
		}
		src := single[0]
		if ml.Crafted {
			src = single[1]
		}
		if types := src[affixTemplate(ml.Line)]; len(types) == 1 && !types["?"] {
			for t := range types {
				count[t]++
			}
			continue
		}
		if ml.Custom {
			count["?"]++
		}
	}
	return affixBound{pre: count["Prefix"], suf: count["Suffix"], untyped: count["?"]}
}

type affixBound struct {
	pre, suf, untyped int
}

// explicitRangeNotes names the item's explicit lines with a value above what PoB's "Add modifier" offers the base
// with that wording (ItemsTab.lua:3612-3749, every source): the best one-line mod plus the best multi-line mod, as the
// game adds two mods with one wording into one line. A ring, amulet or belt may hold catalyst-scaled values, so up to
// 20% above is allowed there. Lines no mod words are left out, and so are the lines an item exported from the game
// (it has a Unique ID) or an affix record holds, except {custom} lines.
func explicitRangeNotes(text string, item *calculator.Item) []string {
	choices, ok := calculator.ItemCustomModChoices(modChoicesText(text))
	if !ok {
		return nil
	}
	allow := 1.0
	if slices.Contains([]string{"Ring", "Amulet", "Belt"}, item.Base.Type) {
		allow = 1.2
	}
	// highs[template][multi] is the highest roll of each number on a line with that wording
	highs := map[string]*[2][]float64{}
	for _, mods := range choices.Mods {
		for _, m := range mods {
			multi := min(len(m.Lines)-1, 1)
			for _, l := range m.Lines {
				t := affixTemplate(l)
				if highs[t] == nil {
					highs[t] = &[2][]float64{}
				}
				h := &highs[t][multi]
				for k, b := range modValueRe.FindAllStringSubmatch(l, -1) {
					lo, hi := b[3], b[3]
					if lo == "" {
						lo, hi = b[1], b[2]
					}
					x, _ := strconv.ParseFloat(lo, 64)
					y, _ := strconv.ParseFloat(hi, 64)
					if k == len(*h) {
						*h = append(*h, 0)
					}
					(*h)[k] = max((*h)[k], x, y)
				}
			}
		}
	}
	recorded := map[string]bool{}
	for _, m := range recordedAffixes(item) {
		for _, l := range m.Lines {
			recorded[affixTemplate(l)] = true
		}
	}
	var out []string
	for _, ml := range item.ExplicitModLines {
		t := affixTemplate(ml.Line)
		if h := highs[t]; h == nil || !ml.Custom && (item.UniqueID != "" || recorded[t]) {
			continue
		}
		rolled := rolledLine(ml)
		values := lineValueRe.FindAllString(rolled, -1)
		for k, v := range values {
			x, _ := strconv.ParseFloat(v, 64)
			limit := 0.0
			for _, hs := range highs[t] {
				if k < len(hs) {
					limit += hs[k]
				}
			}
			if x > limit*allow {
				out = append(out, fmt.Sprintf("%q is above the range of every mod with that wording", rolled))
				break
			}
		}
	}
	return out
}
