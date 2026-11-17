package mcp

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/utils"
)

var (
	affixRangeRe  = regexp.MustCompile(`\(-?\d+(?:\.\d+)?-\d+(?:\.\d+)?\)`)
	affixNumberRe = regexp.MustCompile(`\d+(?:\.\d+)?`)
)

func affixTemplate(line string) string {
	return norm(affixNumberRe.ReplaceAllString(affixRangeRe.ReplaceAllString(line, "#"), "#"))
}

// affixPool is the base's prefix and suffix table (Item.lua's affix list for the base type) by mod id.
func affixPool(item *calculator.Item) []data.AffixMod {
	baseType, subType := item.Base.Type, item.Base.SubType
	if pb, ok := data.CraftBaseKnown(item.Base.Name); ok {
		baseType, subType = pb.Type, pb.SubType
	}
	if subType != "" {
		if t, ok := data.AffixTable(baseType + subType); ok {
			return t
		}
	}
	if t, ok := data.AffixTable(baseType); ok {
		return t
	}
	t, _ := data.AffixTable("Item")
	return t
}

// delveMods are the base's Delve drop-restricted mods, PoB's "Delve Drop-restricted" modifier source.
//
//pob:block:src/Classes/ItemsTab.lua:3685-3708
func delveMods(item *calculator.Item) []data.AffixMod {
	sub := ""
	if item.Base.SubType != "" {
		sub = item.Base.Type + ": " + item.Base.SubType
	}
	var out []data.AffixMod
	for _, d := range data.DropRestricted("DELVE") {
		if slices.ContainsFunc(d.Categories, func(c string) bool { return c == item.Base.Type || c == sub }) {
			out = append(out, d.AffixMod)
		}
	}
	return out
}

// fossilMods are the ids of the item's fossil-only mods, PoB's Subterranean and of the Underground affixes: its prefix and
// suffix lists offer them, but only a fossil craft rolls them.
//
//pob:block:src/Classes/ItemsTab.lua:3710-3720,src/Classes/Item.lua:1767-1769
func fossilMods(choices calculator.CustomModChoices) map[string]bool {
	out := map[string]bool{}
	for _, m := range choices.Mods["FOSSIL"] {
		out[m.ID] = true
	}
	return out
}

const multiCraftLine = "Can have up to 3 Crafted Modifiers"

var affixRecordLineRe = regexp.MustCompile(`^(Prefix|Suffix): (?:\{fractured\})?(?:\{range:[^}]*\})?(.+)$`)

// recordedAffixes are a PoB-crafted item's Prefix:/Suffix: records that name a mod of the base's pool, by mod id or
// else by affix name; "None" and unknown records hold nothing.
//
//pob:block:src/Classes/Item.lua:810-831,src/Classes/Item.lua:1512-1527
func recordedAffixes(item *calculator.Item) []data.AffixMod {
	if !item.Crafted {
		return nil
	}
	pool := affixPool(item)
	var out []data.AffixMod
	for _, l := range strings.Split(item.Raw, "\n") {
		m := affixRecordLineRe.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil || m[2] == "None" {
			continue
		}
		i := slices.IndexFunc(pool, func(a data.AffixMod) bool { return a.ID == m[2] })
		if i < 0 {
			i = slices.IndexFunc(pool, func(a data.AffixMod) bool { return a.Affix == m[2] })
		}
		if i >= 0 && pool[i].Type == m[1] {
			out = append(out, pool[i])
		}
	}
	return out
}

// itemAffixes is what an item's explicit mods already use: prefixes and suffixes (crafted mods included), crafted
// mods, and the mod groups present; the game allows one mod per group.
type itemAffixes struct {
	pre, suf   int
	crafted    int
	craftLimit int
	limit      int
	ilvl       int
	ilvlKnown  bool
	exact      bool
	groups     map[string]bool
}

func (a itemAffixes) open(typ string) bool {
	if typ == "Prefix" {
		return a.pre < a.limit
	}
	return a.suf < a.limit
}

// countAffixes counts an item's explicit mods, a two-line mod once: a PoB-crafted item's affix records count as their
// mods and hold the lines worded or grouped as them; for the other lines a {prefix}/{suffix} tag decides a mod's type,
// else its lines are matched against the base's affixes and the crafting bench's (a crafted line against the bench's
// first), keeping the mods whose rolls hold the line's values when any do; two lines are one mod when a two-line mod's
// rolls hold them or either has no one-line match, and both are crafted or neither is. A mod whose match has no single type is not counted and makes the
// count inexact.
func countAffixes(item *calculator.Item) itemAffixes {
	a := itemAffixes{craftLimit: 1, limit: 3, ilvl: 100, exact: true, groups: map[string]bool{}}
	//pob:block:src/Classes/Item.lua:1490-1503
	if item.Rarity == calculator.RarityMagic {
		a.limit = 1
	} else if item.Base.Type == "Jewel" {
		a.limit = 2
	}
	if item.ItemLevel != nil {
		a.ilvl, a.ilvlKnown = *item.ItemLevel, true
	}
	pool, bench := map[string][]data.AffixMod{}, map[string][]data.AffixMod{}
	add := func(into map[string][]data.AffixMod, m data.AffixMod) {
		if m.Type == "Prefix" || m.Type == "Suffix" {
			k := affixTemplate(strings.Join(m.Lines, "\n"))
			into[k] = append(into[k], m)
		}
	}
	for _, m := range slices.Concat(affixPool(item), delveMods(item)) {
		add(pool, m)
	}
	for _, m := range data.MasterMods() {
		if slices.Contains(m.Types, item.Base.Type) {
			add(bench, m.AffixMod)
		}
	}
	match := func(key string, crafted bool) ([]data.AffixMod, bool) {
		first, second := pool, bench
		if crafted {
			first, second = bench, pool
		}
		if mods, ok := first[key]; ok {
			return mods, true
		}
		mods, ok := second[key]
		return mods, ok
	}
	type record struct {
		mod   data.AffixMod
		lines map[string]bool
		left  int
	}
	recs := recordedAffixes(item)
	records := make([]*record, 0, len(recs))
	for _, m := range recs {
		r := &record{mod: m, lines: map[string]bool{}, left: len(m.Lines)}
		for _, l := range m.Lines {
			r.lines[affixTemplate(l)] = true
		}
		records = append(records, r)
		a.groups[m.Group] = true
		if m.Type == "Prefix" {
			a.pre++
		} else {
			a.suf++
		}
	}
	// recorded is the record a line belongs to: one with lines left whose wording or group the line has
	recorded := func(line string) bool {
		key := affixTemplate(line)
		mods, _ := match(key, false)
		for _, r := range records {
			if r.left > 0 && (r.lines[key] || slices.ContainsFunc(mods, func(m data.AffixMod) bool { return m.Group == r.mod.Group })) {
				r.left--
				return true
			}
		}
		return false
	}
	lines := item.ExplicitModLines
	for i := 0; i < len(lines); i++ {
		ml := lines[i]
		if !ml.Crafted && recorded(ml.Line) {
			continue
		}
		var mods []data.AffixMod
		if i+1 < len(lines) && lines[i+1].Crafted == ml.Crafted {
			if m, ok := match(affixTemplate(ml.Line+"\n"+lines[i+1].Line), ml.Crafted); ok {
				_, one := match(affixTemplate(ml.Line), ml.Crafted)
				_, two := match(affixTemplate(lines[i+1].Line), lines[i+1].Crafted)
				if fit := rolledAs(m, rolledLine(ml)+"\n"+rolledLine(lines[i+1])); len(fit) > 0 {
					mods = fit
					i++
				} else if !one || !two {
					mods = m
					i++
				}
			}
		}
		if mods == nil {
			mods, _ = match(affixTemplate(ml.Line), ml.Crafted)
			if fit := rolledAs(mods, rolledLine(ml)); len(fit) > 0 {
				mods = fit
			}
		}
		types := map[string]bool{}
		for _, m := range mods {
			types[m.Type] = true
			a.groups[m.Group] = true
		}
		if ml.Crafted {
			a.crafted++
			if strings.TrimSpace(ml.Line) == multiCraftLine {
				a.craftLimit = 3
			}
		}
		switch {
		case ml.Prefix:
			a.pre++
		case ml.Suffix:
			a.suf++
		case len(types) != 1:
			a.exact = false
		case types["Prefix"]:
			a.pre++
			a.exact = false
		default:
			a.suf++
			a.exact = false
		}
	}
	// a magic item's name carries its affixes: a word before the base is its prefix, "of ..." after it its suffix
	//pob:block:src/Classes/Item.lua:990-992
	if item.Rarity == calculator.RarityMagic {
		if strings.TrimSpace(item.NamePrefix) != "" {
			a.pre = max(a.pre, 1)
		}
		if strings.HasPrefix(strings.TrimSpace(item.NameSuffix), "of ") {
			a.suf = max(a.suf, 1)
		}
	}
	return a
}

var (
	modValueRe  = regexp.MustCompile(`\((\d+(?:\.\d+)?)-(\d+(?:\.\d+)?)\)|(\d+(?:\.\d+)?)`)
	lineValueRe = regexp.MustCompile(`\d+(?:\.\d+)?`)
)

// rolledLine is a mod line's text with its range rolled as the item has it.
func rolledLine(ml calculator.ModLine) string {
	r := 0.5
	if ml.Range != nil {
		r = *ml.Range
	}
	return data.ApplyRange(ml.Line, r, nil, nil)
}

// rolledAs keeps the mods whose rolls can give text's values: each number within its line's range.
func rolledAs(mods []data.AffixMod, text string) []data.AffixMod {
	values := lineValueRe.FindAllString(text, -1)
	var out []data.AffixMod
	for _, m := range mods {
		bounds := modValueRe.FindAllStringSubmatch(strings.Join(m.Lines, "\n"), -1)
		fits := len(bounds) == len(values)
		for k := 0; fits && k < len(bounds); k++ {
			v, _ := strconv.ParseFloat(values[k], 64)
			lo, hi := bounds[k][3], bounds[k][3]
			if lo == "" {
				lo, hi = bounds[k][1], bounds[k][2]
			}
			a, _ := strconv.ParseFloat(lo, 64)
			b, _ := strconv.ParseFloat(hi, 64)
			fits = v >= min(a, b) && v <= max(a, b)
		}
		if fits {
			out = append(out, m)
		}
	}
	return out
}

type affixCandidate struct {
	id    string
	label string
	typ   string
	text  string
	group string
	lines []string
}

// affixRoll is the roll a scan gives an added mod: rng across each line's range (0 lowest, 1 highest) of tier tier
// (1 the base's best; 0 the best the item level allows).
type affixRoll struct {
	rng   float64
	tier  int
	delve bool
}

// defaultRoll is PoB's own for a mod line without a stored roll (main.defaultItemAffixQuality, Item.lua:1327) at the
// best tier the item level allows.
var defaultRoll = affixRoll{rng: 0.5}

func (r affixRoll) label(lines []string, tier int) string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = data.ApplyRange(l, r.rng, nil, nil)
	}
	return strings.Join(out, " / ") + fmt.Sprintf(" (T%d)", tier)
}

// insertedLines locates what after adds to before when it keeps every line of before unchanged and adds one contiguous
// run of lines: the lines with the run in before's order, and the run's start and length. Blank lines at the end do not
// count.
func insertedLines(before, after string) ([]string, int, int, bool) {
	trim := func(lines []string) []string {
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		return lines
	}
	all := strings.Split(after, "\n")
	old, next := trim(strings.Split(before, "\n")), trim(slices.Clone(all))
	n := len(next) - len(old)
	if n <= 0 {
		return nil, 0, 0, false
	}
	at := 0
	for at < len(old) && old[at] == next[at] {
		at++
	}
	if slices.Equal(old[at:], next[at+n:]) {
		return all, at, n, true
	}
	return movedInsert(strings.Split(before, "\n"), old, next, n)
}

// movedInsert places a run of n lines after's re-serialisation added into before's own lines, when after keeps every
// line of before but not their order (PoB files a crafted implicit line as an enchant, Item.lua:1309-1310, and writes
// enchants first): the run goes after the line it follows in after, the same occurrence of it in before.
func movedInsert(all, old, next []string, n int) ([]string, int, int, bool) {
	count := func(lines []string) map[string]int {
		c := map[string]int{}
		for _, l := range lines {
			c[l]++
		}
		return c
	}
	want := count(old)
	for at := 0; at+n <= len(next); at++ {
		if !maps.Equal(count(slices.Concat(next[:at], next[at+n:])), want) {
			continue
		}
		pos := 0
		if at > 0 {
			prev, k := next[at-1], 0
			for _, l := range next[:at] {
				if l == prev {
					k++
				}
			}
			for pos < len(old) && k > 0 {
				if old[pos] == prev {
					k--
				}
				pos++
			}
		}
		return slices.Concat(all[:pos], next[at:at+n], all[pos:]), pos, n, true
	}
	return nil, 0, 0, false
}

var leadingTagsRe = regexp.MustCompile(`^(\{[^}]*\})*`)

// addModText adds a mod to an item through PoB's "Add modifier" (ItemsTab.lua:3799-3817) with its ranged lines rolled
// at rng (0 lowest, 1 highest), refusing a result that changes any line the item already has. A prefix or suffix on a
// PoB-crafted item fills its first empty affix record instead and gets the lines Item:Craft writes for it.
func addModText(text, src, id string, rng float64) (string, bool) {
	next, ok := calculator.CustomModText(text, src, id, "")
	if !ok {
		return "", false
	}
	lines, at, n, ok := insertedLines(text, next)
	if !ok {
		return "", false
	}
	if record := affixRecord(lines, src); record >= 0 {
		mod, ok := poolMod(calculator.ParseItem(text), id)
		if !ok || len(mod.Lines) != n {
			return "", false
		}
		lines[record] = mod.Type + ": {range:" + strconv.FormatFloat(utils.RoundTo(rng, 3), 'f', -1, 64) + "}" + id
		for k, l := range mod.Lines {
			lines[at+k] = craftedLine(mod, data.ApplyRange(l, rng, nil, nil))
		}
		return strings.Join(lines, "\n"), true
	}
	for i := at; i < at+n; i++ {
		if affixRangeRe.MatchString(lines[i]) {
			k := len(leadingTagsRe.FindString(lines[i]))
			lines[i] = lines[i][:k] + "{range:" + strconv.FormatFloat(rng, 'f', -1, 64) + "}" + lines[i][k:]
		}
	}
	return strings.Join(lines, "\n"), true
}

// affixRecord is the line of a PoB-crafted item's first empty record for a PREFIX or SUFFIX mod, else -1.
//
//pob:block:src/Classes/Item.lua:1837-1846,src/Classes/ItemsTab.lua:3873-3876
func affixRecord(lines []string, src string) int {
	empty := map[string]string{"PREFIX": "Prefix: None", "SUFFIX": "Suffix: None"}[src]
	if empty == "" || !slices.ContainsFunc(lines, func(l string) bool { return strings.TrimSpace(l) == "Crafted: true" }) {
		return -1
	}
	return slices.IndexFunc(lines, func(l string) bool { return strings.TrimSpace(l) == empty })
}

//pob:block:src/Classes/Item.lua:2095-2101,src/Classes/Item.lua:1889-1894
func craftedLine(mod data.AffixMod, line string) string {
	line = "{" + strings.ToLower(mod.Type) + "}" + line
	if len(mod.ModTags) > 0 {
		line = "{tags:" + strings.Join(mod.ModTags, ",") + "}" + line
	}
	return line
}

func poolMod(item *calculator.Item, id string) (data.AffixMod, bool) {
	pool := affixPool(item)
	i := slices.IndexFunc(pool, func(m data.AffixMod) bool { return m.ID == id })
	if i < 0 {
		return data.AffixMod{}, false
	}
	return pool[i], true
}

// modChoicesText is the item as PoB's "Add modifier" lists prefixes and suffixes for: a PoB-crafted item gets its
// affix controls there instead, so its Crafted flag is dropped for the listing; the mod is added to the item as is.
func modChoicesText(text string) string {
	lines := strings.Split(text, "\n")
	return strings.Join(slices.DeleteFunc(lines, func(l string) bool { return strings.TrimSpace(l) == "Crafted: true" }), "\n")
}

// affixCandidates are the mods an equipped item can still take, each added to its text without touching its other
// lines and rolled as roll says: PoB's "Add modifier" prefix, suffix and Delve drop-restricted lists
// (ItemsTab.lua:3612-3749) for its open affix types, one mod per group it does not have, at the roll's tier or the highest its item level allows; and the
// crafting bench's mods the item level allows while it has room for another crafted mod.
func affixCandidates(text string, item *calculator.Item, a itemAffixes, roll affixRoll) []affixCandidate {
	var out []affixCandidate
	byID := map[string]data.AffixMod{}
	for _, m := range affixPool(item) {
		byID[m.ID] = m
	}
	delve := map[string]data.AffixMod{}
	for _, m := range delveMods(item) {
		delve[m.ID] = m
	}
	choices, ok := calculator.ItemCustomModChoices(modChoicesText(text))
	if !ok {
		return nil
	}
	rng := strconv.FormatFloat(roll.rng, 'f', -1, 64)
	fossil := fossilMods(choices)
	for _, src := range []string{"PREFIX", "SUFFIX"} {
		typ := map[string]string{"PREFIX": "Prefix", "SUFFIX": "Suffix"}[src]
		if !a.open(typ) {
			continue
		}
		type key struct{ src, group string }
		tiers := map[key][]data.AffixMod{}
		var order []key
		froms := []string{src}
		if roll.delve {
			froms = append(froms, "DELVE")
		}
		for _, from := range froms {
			mods := byID
			if from == "DELVE" {
				mods = delve
			}
			for _, cm := range choices.Mods[from] {
				m, ok := mods[cm.ID]
				if !ok || m.Type != typ || a.groups[m.Group] || (fossil[m.ID] && !roll.delve) {
					continue
				}
				k := key{from, m.Group}
				if _, seen := tiers[k]; !seen {
					order = append(order, k)
				}
				tiers[k] = append(tiers[k], m)
			}
		}
		for _, g := range order {
			list := tiers[g]
			slices.SortStableFunc(list, func(x, y data.AffixMod) int { return y.Level - x.Level })
			k := slices.IndexFunc(list, func(m data.AffixMod) bool { return m.Level <= a.ilvl })
			if roll.tier > 0 {
				k = roll.tier - 1
			}
			if k < 0 || k >= len(list) || list[k].Level > a.ilvl {
				continue
			}
			m := list[k]
			label := roll.label(m.Lines, k+1)
			if g.src == "DELVE" {
				label += " Delve drop-restricted"
			} else if fossil[m.ID] {
				label += " Delve fossil"
			}
			if t, ok := addModText(text, g.src, m.ID, roll.rng); ok {
				out = append(out, affixCandidate{g.src + ":" + m.ID + "|" + rng, label, typ, t, g.group, m.Lines})
			}
		}
	}
	if a.crafted >= a.craftLimit {
		return out
	}
	masters := data.MasterMods()
	bench := map[string]int{}
	var order []string
	for _, cm := range choices.Mods["MASTER"] {
		i, err := strconv.Atoi(strings.TrimPrefix(cm.ID, "master:"))
		if err != nil || i < 0 || i >= len(masters) {
			continue
		}
		m := masters[i]
		if !a.open(m.Type) || a.groups[m.Group] || m.Level > a.ilvl {
			continue
		}
		if cur, seen := bench[m.Group]; !seen || m.Level >= masters[cur].Level {
			if !seen {
				order = append(order, m.Group)
			}
			bench[m.Group] = i
		}
	}
	for _, g := range order {
		m := masters[bench[g]]
		id := "master:" + strconv.Itoa(bench[g])
		if t, ok := addModText(text, "MASTER", id, roll.rng); ok {
			out = append(out, affixCandidate{"MASTER:" + id + "|" + rng, roll.label(m.Lines, 1) + " bench", m.Type, t, g, m.Lines})
		}
	}
	return out
}

// affixAllowed reports whether the item can take the mod a scan row names: its affix type open, its group absent, its
// level within the item level, and a bench mod only with room for another crafted mod.
func affixAllowed(item *calculator.Item, src, id string) bool {
	if item.Base == nil {
		return false
	}
	a := countAffixes(item)
	var m data.AffixMod
	switch src {
	case "PREFIX", "SUFFIX", "DELVE":
		pool := affixPool(item)
		if src == "DELVE" {
			pool = delveMods(item)
		}
		i := slices.IndexFunc(pool, func(m data.AffixMod) bool { return m.ID == id })
		if i < 0 {
			return false
		}
		m = pool[i]
	case "MASTER":
		masters := data.MasterMods()
		i, err := strconv.Atoi(strings.TrimPrefix(id, "master:"))
		if err != nil || i < 0 || i >= len(masters) || a.crafted >= a.craftLimit {
			return false
		}
		m = masters[i].AffixMod
	default:
		return false
	}
	return a.open(m.Type) && !a.groups[m.Group] && m.Level <= a.ilvl
}

// openEntry is an item's open_affixes entry: the prefixes and suffixes it can still take and what it uses.
func openEntry(slot string, a itemAffixes) map[string]any {
	entry := map[string]any{
		"slot":          slot,
		"open_prefixes": max(0, a.limit-a.pre), "open_suffixes": max(0, a.limit-a.suf),
		"used": fmt.Sprintf("%dP/%dS of %d each, %d/%d crafted", a.pre, a.suf, a.limit, a.crafted, a.craftLimit),
	}
	if !a.exact {
		entry["estimated"] = true
	}
	if !a.ilvlKnown {
		entry["item_level"] = "unknown, tiers up to 100"
	}
	return entry
}

func (sc *scanCtx) affixes(slots []string, deadline time.Time) ([]map[string]any, []any) {
	var rows []map[string]any
	var openList []any
	var items []equipped
	for _, e := range equippedItems(sc.b) {
		if slices.Contains(slots, e.slot) && (e.rarity == string(calculator.RarityRare) || e.rarity == string(calculator.RarityMagic)) {
			items = append(items, e)
		}
	}
	for i, e := range items {
		item := calculator.ParseItem(e.text)
		if item.Base == nil {
			continue
		}
		a := countAffixes(item)
		openList = append(openList, openEntry(e.slot, a))
		if !a.open("Prefix") && !a.open("Suffix") {
			continue
		}
		itemDeadline := time.Now().Add(time.Until(deadline) / time.Duration(len(items)-i))
		var itemRows []map[string]any
		for _, c := range affixCandidates(e.text, item, a, sc.roll) {
			if time.Now().After(itemDeadline) {
				sc.partial = true
				break
			}
			after := sc.misc.Calc(&calculator.CalcOverride{RepSlotName: e.slot, RepItem: calculator.ParseItem(c.text)})
			row := sc.row("affix|"+e.slot+"|"+c.id, e.slot, c.label, after, uniqueChange(c.text, e.slot))
			row["type"] = c.typ
			if sc.keep(row) {
				itemRows = append(itemRows, row)
			}
		}
		rows = append(rows, bestPerSlot(itemRows, sc.in.PerSlot)...)
	}
	return rows, openList
}
