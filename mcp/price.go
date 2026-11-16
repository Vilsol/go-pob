package mcp

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mcp/prices"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

// CodePricesUnavailable answers a pricing call on a server started without prices.
const CodePricesUnavailable ErrorCode = "PRICES_UNAVAILABLE"

const (
	priceNinja    = "ninja"
	priceEstimate = "estimate"
	priceUnknown  = "unknown"
)

// Price is what one item or gem costs in chaos; Chaos is nil when it cannot be priced, which never counts as free.
type Price struct {
	Chaos  *float64 `json:"chaos"`
	Source string   `json:"price_source"`
	Detail string   `json:"detail,omitempty"`
	// Listings is poe.ninja's listing count behind a ninja price; LowConfidence marks fewer than prices.ThinMarket.
	Listings      *int `json:"listings,omitempty"`
	LowConfidence bool `json:"low_confidence,omitempty"`
}

func (p Price) known() bool { return p.Chaos != nil }

func knownPrice(chaos float64, source, detail string) Price {
	c := math.Round(chaos*10) / 10
	return Price{Chaos: &c, Source: source, Detail: detail}
}

func ninjaPrice(q prices.Quote, detail string) Price {
	p := knownPrice(q.Chaos, priceNinja, detail)
	n := q.Listings
	p.Listings, p.LowConfidence = &n, n < prices.ThinMarket
	return p
}

func unknownPrice(why string) Price {
	return Price{Source: priceUnknown, Detail: why}
}

// PricedPart is one slot's item or one gem in a build's cost.
type PricedPart struct {
	Slot string `json:"slot,omitempty"`
	Name string `json:"name"`
	Price
}

// BuildCost is a build's price: TotalChaos sums the known parts, Unknown counts the rest.
type BuildCost struct {
	TotalChaos  float64      `json:"total_chaos"`
	TotalDivine *float64     `json:"total_divine"`
	DivineChaos *float64     `json:"divine_chaos"`
	DivineFrom  string       `json:"divine_source,omitempty"`
	Unknown     int          `json:"unknown"`
	LowConf     int          `json:"low_confidence"`
	League      string       `json:"league"`
	AgeMinutes  *int         `json:"snapshot_age_min"`
	Items       []PricedPart `json:"items"`
	Gems        []PricedPart `json:"gems"`
}

// PriceBuild prices every item the build wears (the active weapon set and jewels in allocated tree sockets) and every
// enabled gem in enabled groups of the active skill set. Item-granted skills cost nothing.
func PriceBuild(book *prices.Book, b *pob.PathOfBuilding) BuildCost {
	cost := BuildCost{Items: []PricedPart{}, Gems: []PricedPart{}}
	texts := map[int]string{}
	for _, it := range b.Items.Items {
		texts[it.ID] = it.Text
	}
	add := func(into *[]PricedPart, part PricedPart) {
		*into = append(*into, part)
		if part.known() {
			cost.TotalChaos += *part.Chaos
		} else {
			cost.Unknown++
		}
		if part.LowConfidence {
			cost.LowConf++
		}
	}
	for _, e := range equippedItems(b) {
		item := calculator.PeekItem(texts[e.id])
		add(&cost.Items, PricedPart{Slot: e.slot, Name: e.name, Price: PriceItem(book, item)})
	}
	if spec := activeSpec(b); spec != nil {
		allocated := map[int64]bool{}
		for _, n := range pob.SpecNodes(*spec) {
			allocated[n] = true
		}
		for _, s := range spec.Sockets {
			text, ok := texts[s.ItemID]
			if !ok || s.ItemID == 0 || !allocated[int64(s.NodeID)] {
				continue
			}
			item := calculator.PeekItem(text)
			add(&cost.Items, PricedPart{Slot: fmt.Sprintf("Jewel %d", s.NodeID), Name: item.Name, Price: PriceItem(book, item)})
		}
	}
	for _, sk := range activeSkills(b) {
		if !sk.Enabled || sk.Source != "" {
			continue
		}
		for _, g := range sk.Gems {
			if !g.Enabled {
				continue
			}
			add(&cost.Gems, PricedPart{Slot: sk.Slot, Name: fmt.Sprintf("%s %d/%d", g.NameSpec, g.Level, g.Quality), Price: PriceGem(book, g)})
		}
	}
	cost.TotalChaos = math.Round(cost.TotalChaos*10) / 10
	league, at := book.Info()
	cost.League = league
	if !at.IsZero() {
		age := int(time.Since(at).Minutes())
		cost.AgeMinutes = &age
	}
	if div, from, ok := book.DivineRate(); ok {
		d, t := math.Round(div*10)/10, math.Round(cost.TotalChaos/div*100)/100
		cost.DivineChaos, cost.TotalDivine, cost.DivineFrom = &d, &t, from
	}
	return cost
}

// PriceGem prices a gem by poe.ninja's listing for its level and quality (prices.Book.GemQuote); a Vaal gem, a gem
// above its natural maximum level and one above 20 quality are corrupted.
func PriceGem(book *prices.Book, g pob.Gem) Price {
	row, ok := gemRow(g)
	if !ok {
		return unknownPrice("gem not in the game data")
	}
	name := row.Name
	if row.Tags["support"] && !strings.HasSuffix(name, " Support") {
		name += " Support"
	}
	corrupted := row.Tags["vaal"] || g.Quality > 20 || (row.NaturalMaxLevel > 0 && g.Level > row.NaturalMaxLevel)
	if vendorGem(g, row) && !corrupted && g.Level < row.NaturalMaxLevel && g.Quality < 20 {
		return knownPrice(0, priceEstimate, vendorGemDetail)
	}
	if q, ok := book.GemQuote(name, g.Level, g.Quality, corrupted); ok {
		switch {
		case q.Corrupted:
			return ninjaPrice(q, "priced as "+q.Variant+" (corrupted listing, cheaper)")
		case q.Variant != "":
			return ninjaPrice(q, "priced as "+q.Variant)
		}
		return ninjaPrice(q, "")
	}
	return unknownPrice("not listed on poe.ninja")
}

const vendorGemDetail = "a quest or vendor gem below its maximum level: bought at level 1 for shards and levelled"

// vendorGem is a gem every class can take as a quest reward or buy from a vendor (Siosa, Lilly Roth): one the game data
// does not mark Vaal, Awakened or exceptional (drop-only) and not a transfigured variant.
func vendorGem(g pob.Gem, row raw.PoBGem) bool {
	if g.VariantID != nil {
		if _, ok := raw.GemVariants[*g.VariantID]; ok {
			return false
		}
	}
	return !row.Tags["vaal"] && !row.Tags["awakened"] && !row.Tags["exceptional"]
}

func gemRow(g pob.Gem) (raw.PoBGem, bool) {
	if g.VariantID != nil {
		if row, ok := raw.GemVariants[*g.VariantID]; ok {
			return row, true
		}
	}
	if row, ok := raw.Gems[g.GemID]; ok {
		return row, true
	}
	return gemRowByName(g.NameSpec)
}

var (
	clusterCountRe   = regexp.MustCompile(`^Adds (\d+) Passive Skills?$`)
	clusterEnchantRe = regexp.MustCompile(`^Added Small Passive Skills grant: (.+)$`)
)

// PriceItem prices one item: a unique (unique flasks included) by poe.ninja's listing at its link count (else the
// unlinked listing plus the bench's link craft), a normal, magic or rare item by the estimate in price_table.go.
func PriceItem(book *prices.Book, item *calculator.Item) Price {
	switch item.Rarity {
	case calculator.RarityUnique:
		links := largestLink(item)
		if q, ok := book.UniqueQuote(item.Title, uniqueVariant(item), links); ok {
			if links >= 5 && q.Links != links {
				c, ok := benchLinkChaos(book, links)
				if !ok {
					return unknownPrice("link craft cost unknown")
				}
				q.Chaos += c
				return ninjaPrice(q, fmt.Sprintf("unlinked + %dL bench %g", links, math.Round(c*10)/10))
			}
			switch {
			case q.Links > 0:
				return ninjaPrice(q, fmt.Sprintf("%dL listing", q.Links))
			case len(item.Sockets) > 0:
				return ninjaPrice(q, "unlinked listing")
			}
			return ninjaPrice(q, "")
		}
		return unknownPrice("not listed on poe.ninja")
	case calculator.RarityRelic:
		return unknownPrice("relics are not priced")
	}
	return estimateItem(book, item)
}

func uniqueVariant(item *calculator.Item) string {
	if item.Variant != nil && *item.Variant >= 1 && *item.Variant <= len(item.VariantList) {
		return item.VariantList[*item.Variant-1]
	}
	return ""
}

func largestLink(item *calculator.Item) int {
	groups := map[int]int{}
	best := 0
	for _, s := range item.Sockets {
		groups[s.Group]++
		best = max(best, groups[s.Group])
	}
	return best
}

func influenceName(inf calculator.ItemInfluence) string {
	var out []string
	for _, f := range []struct {
		on   bool
		name string
	}{{inf.Shaper, "Shaper"}, {inf.Elder, "Elder"}, {inf.Crusader, "Crusader"}, {inf.Eyrie, "Redeemer"}, {inf.Basilisk, "Hunter"}, {inf.Adjudicator, "Warlord"}} {
		if f.on {
			out = append(out, f.name)
		}
	}
	return strings.Join(out, "/")
}

// basePrice is the base's price and, when poe.ninja's price does not apply, why: poe.ninja lists bases only at item
// level ninjaBaseMinLevel and up, as crafting bases, so an item with no Item Level line, one below that, or a base it
// does not list costs unlistedBaseChaos.
func basePrice(book *prices.Book, item *calculator.Item) (float64, string) {
	ilvl := 0
	if item.ItemLevel != nil {
		ilvl = *item.ItemLevel
	}
	if strings.HasSuffix(item.BaseName, "Cluster Jewel") {
		passives, enchant := 0, ""
		for _, ml := range slices.Concat(item.EnchantModLines, item.ImplicitModLines) {
			if m := clusterCountRe.FindStringSubmatch(ml.Line); m != nil {
				passives, _ = strconv.Atoi(m[1])
			}
			if m := clusterEnchantRe.FindStringSubmatch(ml.Line); m != nil {
				enchant = m[1]
			}
		}
		p, _ := book.Cluster(item.BaseName, enchant, passives, ilvl)
		return p, ""
	}
	switch {
	case ilvl <= 0:
		return unlistedBaseChaos, fmt.Sprintf("no ilvl, ninja lists %d+ only", ninjaBaseMinLevel)
	case ilvl < ninjaBaseMinLevel:
		return unlistedBaseChaos, fmt.Sprintf("ilvl %d < ninja's %d+", ilvl, ninjaBaseMinLevel)
	}
	if p, ok := book.Base(item.BaseName, ilvl, influenceName(item.Influence)); ok {
		return p, ""
	}
	return unlistedBaseChaos, "unlisted"
}

// estimateItem is the deterministic estimate for a non-unique item: a normal item's base price, a magic item's base
// price plus a fixed magic cost, a rare's base price + affixes by tier + bench costs + rareExtras.
func estimateItem(book *prices.Book, item *calculator.Item) Price {
	if item.Base == nil {
		return unknownPrice("unknown base")
	}
	base, why := basePrice(book, item)
	baseLabel := fmt.Sprintf("base %g", math.Round(base*10)/10)
	if why != "" {
		baseLabel += " (" + why + ")"
	}
	switch item.Rarity {
	case calculator.RarityNormal:
		return knownPrice(base, priceEstimate, baseLabel)
	case calculator.RarityMagic:
		return knownPrice(base+magicItemChaos, priceEstimate, fmt.Sprintf("%s + magic %g", baseLabel, magicItemChaos))
	}
	total := base
	tiers := map[string]int{}
	bench := 0.0
	placed, ok := placeAffixes(item)
	if !ok {
		return unknownPrice("no affix list for the base")
	}
	for _, a := range placed {
		switch {
		case a.master >= 0:
			c, ok := benchChaos(book, a.master)
			if !ok {
				return unknownPrice("bench craft cost unknown")
			}
			bench += c
		case a.tier == 0:
			total += unplacedAffixChaos
			tiers["other"]++
		case a.weaponDamage:
			total += weaponDamageTierChaos(a.tier)
			tiers["dmg "+tierLabel(a.tier)]++
		default:
			total += affixTierChaos(a.tier)
			tiers[tierLabel(a.tier)]++
		}
	}
	total += bench
	parts := []string{baseLabel}
	for _, k := range []string{"dmg T1", "dmg T2", "dmg T3-", "T1", "T2", "T3-"} {
		if n := tiers[k]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s×%d", k, n))
		}
	}
	if n := tiers["other"]; n > 0 {
		parts = append(parts, fmt.Sprintf("other×%d (unplaced: influence/essence/veiled/delve mods, as T1)", n))
	}
	if bench > 0 {
		parts = append(parts, fmt.Sprintf("bench %g", math.Round(bench*10)/10))
	}
	extra, more, ok := rareExtras(book, item)
	if !ok {
		return unknownPrice("link craft cost unknown")
	}
	return knownPrice(total+extra, priceEstimate, strings.Join(append(parts, more...), " + "))
}

var gemLevelLineRe = regexp.MustCompile(`^\+(\d+) to Level of all (?:[A-Za-z]+ )*Skill Gems$`)

// rareExtras prices what the affix tiers do not see: eldritch implicits by tier, anoints, a 5- or 6-link at the bench's
// link craft cost, and "+N to Level of all ... Skill Gems" lines (price_table.go).
func rareExtras(book *prices.Book, item *calculator.Item) (float64, []string, bool) {
	total := 0.0
	var parts []string
	var sides []string
	eld := 0.0
	for _, side := range []string{"exarch", "eater"} {
		if t := eldritchTier(item, side); t > 0 {
			sides = append(sides, fmt.Sprintf("%s T%d", side, t))
			eld += eldritchTierChaos[t]
		}
	}
	if len(sides) > 0 {
		total += eld
		parts = append(parts, fmt.Sprintf("eldritch %s (%g)", strings.Join(sides, " + "), eld))
	}
	if n := len(calculator.ItemAnoints(item)); n > 0 {
		total += float64(n) * anointChaos
		parts = append(parts, fmt.Sprintf("anoint×%d (%g)", n, float64(n)*anointChaos))
	}
	if links := largestLink(item); links >= 5 {
		c, ok := benchLinkChaos(book, links)
		if !ok {
			return 0, nil, false
		}
		total += c
		parts = append(parts, fmt.Sprintf("%dL %g", links, math.Round(c*10)/10))
	}
	levels, lines := 0, 0
	for _, ml := range item.ExplicitModLines {
		if m := gemLevelLineRe.FindStringSubmatch(rolledLine(ml)); m != nil {
			n, _ := strconv.Atoi(m[1])
			levels += n
			lines++
		}
	}
	if levels > 0 {
		c := float64(levels) * gemLevelChaos
		label := fmt.Sprintf("gem levels +%d", levels)
		if lines >= 2 {
			c *= 2
			label += fmt.Sprintf(" on %d lines ×2", lines)
		}
		total += c
		parts = append(parts, fmt.Sprintf("%s (%g)", label, c))
	}
	return total, parts, true
}

var (
	tagPrefixRe   = regexp.MustCompile(`^\{([^}]*)\}`)
	modTierRe     = regexp.MustCompile(`(\d+)$`)
	eldritchTypes = map[string]string{"exarch": "Exarch", "eater": "Eater"}
)

// eldritchTier is the tier (1 Lesser to 6 Perfect) of the item's Searing Exarch or Eater of Worlds implicit: the best
// tier whose range holds each of its lines, the highest over its lines; 0 when it has none or none places.
func eldritchTier(item *calculator.Item, side string) int {
	best := 0
	for _, raw := range strings.Split(item.Raw, "\n") {
		line, tagged := strings.TrimSpace(raw), false
		for {
			m := tagPrefixRe.FindStringSubmatch(line)
			if m == nil {
				break
			}
			tagged = tagged || m[1] == side
			line = line[len(m[0]):]
		}
		if !tagged {
			continue
		}
		text := line
		for _, ml := range item.ImplicitModLines {
			if ml.Line == line {
				text = rolledLine(ml)
				break
			}
		}
		tpl := affixTemplate(line)
		for _, m := range affixPool(item) {
			if m.Type != eldritchTypes[side] {
				continue
			}
			for _, l := range m.Lines {
				if affixTemplate(l) == tpl && len(rolledAs([]data.AffixMod{{Lines: []string{l}}}, text)) > 0 {
					if n, _ := strconv.Atoi(modTierRe.FindString(m.ID)); n > best && n < len(eldritchTierChaos) {
						best = n
					}
				}
			}
		}
	}
	return best
}

func tierLabel(tier int) string {
	if tier >= 3 {
		return "T3-"
	}
	return "T" + strconv.Itoa(tier)
}

// placedAffix is one explicit mod: its tier on the base (1 the base's best, 0 when the base's table cannot place it)
// or, for a bench craft, its index in data.MasterMods (-1 otherwise).
type placedAffix struct {
	tier   int
	master int
	// weaponDamage marks a weapon's local physical, elemental or chaos damage affix, its roll inside the mod's range
	weaponDamage bool
}

// placeAffixes places an item's explicit lines among the affixes PoB's "Add modifier" lists for it and the crafting
// bench. A line goes to the mods holding a line of its form whose range holds its value, those whose every line the
// item has first, at the best tier among them; lines placed in one mod group are one affix, since an item carries one
// mod per group, at the worst of its lines' tiers (a line every tier shares cannot raise it). A crafted line goes to the bench mod of its form whose roll holds it, the highest level first.
func placeAffixes(item *calculator.Item) ([]placedAffix, bool) {
	choices, ok := calculator.ItemCustomModChoices(modChoicesText(item.Raw))
	if !ok {
		return nil, len(item.ExplicitModLines) == 0
	}
	allowed := map[string]bool{}
	for _, src := range []string{"PREFIX", "SUFFIX"} {
		for _, cm := range choices.Mods[src] {
			allowed[cm.ID] = true
		}
	}
	byLine := map[string][]data.AffixMod{}
	byGroup := map[string][]data.AffixMod{}
	for _, m := range affixPool(item) {
		if allowed[m.ID] && (m.Type == "Prefix" || m.Type == "Suffix") {
			for _, l := range m.Lines {
				byLine[affixTemplate(l)] = append(byLine[affixTemplate(l)], m)
			}
			byGroup[m.Type+"|"+m.Group] = append(byGroup[m.Type+"|"+m.Group], m)
		}
	}
	tierOf := map[string]int{}
	for _, mods := range byGroup {
		for _, m := range mods {
			t := 1
			for _, o := range mods {
				if o.Level > m.Level {
					t++
				}
			}
			tierOf[m.ID] = t
		}
	}
	present := map[string]bool{}
	for _, ml := range item.ExplicitModLines {
		if !ml.Crafted {
			present[affixTemplate(ml.Line)] = true
		}
	}
	complete := func(m data.AffixMod) bool {
		return !slices.ContainsFunc(m.Lines, func(l string) bool { return !present[affixTemplate(l)] })
	}
	holds := func(m data.AffixMod, tpl, text string) bool {
		return slices.ContainsFunc(m.Lines, func(l string) bool {
			return affixTemplate(l) == tpl && len(rolledAs([]data.AffixMod{{Lines: []string{l}}}, text)) > 0
		})
	}
	masters := data.MasterMods()
	bench := map[string][]int{}
	for i, m := range masters {
		if slices.Contains(m.Types, item.Base.Type) {
			k := affixTemplate(strings.Join(m.Lines, "\n"))
			bench[k] = append(bench[k], i)
		}
	}
	bestMaster := func(idx []int, text string) int {
		pick, fits := idx[0], false
		for _, j := range idx {
			fit := len(rolledAs([]data.AffixMod{masters[j].AffixMod}, text)) > 0
			if (fit && !fits) || (fit == fits && masters[j].Level > masters[pick].Level) {
				pick, fits = j, fit
			}
		}
		return pick
	}
	var out []placedAffix
	groupTier := map[string]int{}
	groupDamage := map[string]bool{}
	var groupOrder []string
	lines := item.ExplicitModLines
	for i := 0; i < len(lines); i++ {
		ml := lines[i]
		text, tpl := rolledLine(ml), affixTemplate(ml.Line)
		if ml.Crafted {
			if i+1 < len(lines) && lines[i+1].Crafted {
				if idx, ok := bench[affixTemplate(ml.Line+"\n"+lines[i+1].Line)]; ok {
					out = append(out, placedAffix{master: bestMaster(idx, text+"\n"+rolledLine(lines[i+1]))})
					i++
					continue
				}
			}
			if idx, ok := bench[tpl]; ok {
				out = append(out, placedAffix{master: bestMaster(idx, text)})
			} else {
				out = append(out, placedAffix{master: -1})
			}
			continue
		}
		cands := byLine[tpl]
		for _, keep := range []func(data.AffixMod) bool{func(m data.AffixMod) bool { return holds(m, tpl, text) }, complete} {
			if kept := slices.DeleteFunc(slices.Clone(cands), func(m data.AffixMod) bool { return !keep(m) }); len(kept) > 0 {
				cands = kept
			}
		}
		if len(cands) == 0 {
			out = append(out, placedAffix{master: -1})
			continue
		}
		pick := cands[0]
		for _, m := range cands {
			if tierOf[m.ID] < tierOf[pick.ID] {
				pick = m
			}
		}
		g := pick.Type + "|" + pick.Group
		if t, seen := groupTier[g]; !seen || tierOf[pick.ID] > t {
			if !seen {
				groupOrder = append(groupOrder, g)
			}
			groupTier[g] = tierOf[pick.ID]
		}
		groupDamage[g] = groupDamage[g] || (item.Base.Weapon != nil && strings.HasPrefix(pick.ID, "Local") &&
			slices.ContainsFunc(pick.ModTags, func(t string) bool { return strings.HasSuffix(t, "_damage") }) && holds(pick, tpl, text))
	}
	for _, g := range groupOrder {
		out = append(out, placedAffix{tier: groupTier[g], master: -1, weaponDamage: groupDamage[g]})
	}
	return out, true
}

type benchCurrency struct {
	name  string
	count int
}

var (
	benchCosts   = sync.OnceValue(loadBenchCosts)
	benchOptions = sync.OnceValue(loadBenchOptions)
)

// loadBenchOptions is CraftingBenchOptions with the full column set: the app's trimmed variant drops AddMod and the
// cost columns, and the browser build never needs bench costs.
func loadBenchOptions() []*poe.CraftingBenchOption {
	var options []*poe.CraftingBenchOption
	if err := loader.InitHelper(context.Background(), raw.LatestVersion, "CraftingBenchOptions", &options, nil, assetcache.AssetLoaderWrapper{Storage: storage.Get()}); err != nil {
		return nil
	}
	return options
}

func benchCurrencies(craft *poe.CraftingBenchOption) []benchCurrency {
	var cost []benchCurrency
	for k, item := range craft.CostBaseItemTypes {
		if item < 0 || item >= len(poe.BaseItemTypes) || k >= len(craft.CostValues) {
			return nil
		}
		cost = append(cost, benchCurrency{poe.BaseItemTypes[item].Name, craft.CostValues[k]})
	}
	return cost
}

// benchLinkChaos is the bench's cheapest craft that links this many sockets, in chaos.
func benchLinkChaos(book *prices.Book, links int) (float64, bool) {
	best, ok := 0.0, false
	for _, craft := range benchOptions() {
		if craft.IsDisabled || craft.Links != links {
			continue
		}
		cost := benchCurrencies(craft)
		if cost == nil {
			continue
		}
		total, priced := 0.0, true
		for _, c := range cost {
			p, ok := book.Currency(c.name)
			priced = priced && ok
			total += p * float64(c.count)
		}
		if priced && (!ok || total < best) {
			best, ok = total, true
		}
	}
	return best, ok
}

// loadBenchCosts reads each bench craft's currency cost from CraftingBenchOptions, aligned with data.MasterMods: the
// same options in the same order as the masters export keeps them, a newer game version's options appended after
// them. nil when a craft's mod level disagrees at its index.
func loadBenchCosts() [][]benchCurrency {
	options := benchOptions()
	if options == nil {
		return nil
	}
	masters, mods := data.MasterMods(), raw.Mods()
	var out [][]benchCurrency
	for _, craft := range options {
		if craft.IsDisabled || craft.AddMod == nil {
			continue
		}
		i := len(out)
		if i == len(masters) {
			break
		}
		if *craft.AddMod < 0 || *craft.AddMod >= len(mods) || mods[*craft.AddMod].Level != masters[i].Level {
			return nil
		}
		out = append(out, benchCurrencies(craft))
	}
	if len(out) != len(masters) {
		return nil
	}
	return out
}

func benchChaos(book *prices.Book, master int) (float64, bool) {
	costs := benchCosts()
	if master < 0 || master >= len(costs) || costs[master] == nil {
		return 0, false
	}
	total := 0.0
	for _, c := range costs[master] {
		p, ok := book.Currency(c.name)
		if !ok {
			return 0, false
		}
		total += p * float64(c.count)
	}
	return total, true
}
