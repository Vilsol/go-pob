// Package prices reads item prices from poe.ninja's economy overviews for the MCP server. Prices are market data:
// they are fetched at run time or read from a snapshot file the caller supplies, never bundled.
package prices

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

// Category is a poe.ninja overview type.
type Category string

const (
	Currency        Category = "Currency"
	UniqueWeapon    Category = "UniqueWeapon"
	UniqueArmour    Category = "UniqueArmour"
	UniqueAccessory Category = "UniqueAccessory"
	UniqueFlask     Category = "UniqueFlask"
	UniqueJewel     Category = "UniqueJewel"
	SkillGem        Category = "SkillGem"
	BaseType        Category = "BaseType"
	ClusterJewel    Category = "ClusterJewel"
	// Exchange is the in-game Currency Exchange's currency overview; currency rates come from it first.
	Exchange Category = "Exchange"
)

// Categories is every overview a build's price can need.
var Categories = []Category{Exchange, Currency, UniqueWeapon, UniqueArmour, UniqueAccessory, UniqueFlask, UniqueJewel, SkillGem, BaseType, ClusterJewel}

var uniqueCategories = []Category{UniqueWeapon, UniqueArmour, UniqueAccessory, UniqueFlask, UniqueJewel}

// Line is one overview row; JSON nulls leave fields at zero.
type Line struct {
	Name          string  `json:"name"`
	BaseType      string  `json:"baseType"`
	Variant       string  `json:"variant"`
	Links         int     `json:"links"`
	ItemClass     int     `json:"itemClass"`
	LevelRequired int     `json:"levelRequired"`
	GemLevel      int     `json:"gemLevel"`
	GemQuality    int     `json:"gemQuality"`
	Corrupted     bool    `json:"corrupted"`
	ChaosValue    float64 `json:"chaosValue"`
	DivineValue   float64 `json:"divineValue"`
	ListingCount  int     `json:"listingCount"`

	CurrencyTypeName string        `json:"currencyTypeName"`
	ChaosEquivalent  float64       `json:"chaosEquivalent"`
	Pay              *CurrencySide `json:"pay"`
	Receive          *CurrencySide `json:"receive"`
}

// CurrencySide is one direction of a stash currency line: pay's value is units of the currency per chaos, receive's is
// chaos per unit.
type CurrencySide struct {
	Value        float64 `json:"value"`
	ListingCount int     `json:"listing_count"`
}

type overview struct {
	Lines []Line `json:"lines"`
}

// exchangeOverview is the Currency Exchange overview: each line's value in the core's primary currency, keyed by an id
// the items list names.
type exchangeOverview struct {
	Core struct {
		Primary string `json:"primary"`
	} `json:"core"`
	Lines []struct {
		ID           string  `json:"id"`
		PrimaryValue float64 `json:"primaryValue"`
	} `json:"lines"`
	Items []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"items"`
}

// source supplies one league's raw overview responses.
type source interface {
	league() (string, error)
	overview(league string, c Category) ([]byte, time.Time, error)
}

// ThinMarket is the listing count below which a poe.ninja price rests on too few offers to trust: low confidence.
const ThinMarket = 10

// Quote is one poe.ninja row's price, its listing count and, for a gem priced by any row but its exact one, that row's
// variant; Corrupted marks an uncorrupted gem priced by a cheaper corrupted listing of the same or a better gem.
type Quote struct {
	Chaos     float64
	Listings  int
	Variant   string
	Corrupted bool
	// Links is the row's link count (0 for an unlinked row).
	Links int
}

func quote(l Line) Quote { return Quote{Chaos: l.ChaosValue, Listings: l.ListingCount, Links: l.Links} }

// Book answers price lookups, loading each category on first need. A category that fails to load answers nothing; the
// failure is reported once through Warning.
type Book struct {
	mu        sync.Mutex
	src       source
	leagueID  string
	leagueErr error
	lines     map[Category][]Line
	exchange  map[string]float64
	fetched   map[Category]time.Time
	failed    map[Category]time.Time
	errs      []string
	warned    bool
	now       func() time.Time
}

// retryFailed is how long a category that failed to load stays unanswered before the next lookup tries again.
const retryFailed = 15 * time.Minute

func newBook(src source) *Book {
	return &Book{src: src, lines: map[Category][]Line{}, fetched: map[Category]time.Time{}, failed: map[Category]time.Time{}, now: time.Now}
}

func (b *Book) league() (string, error) {
	if b.leagueID == "" && b.leagueErr == nil {
		b.leagueID, b.leagueErr = b.src.league()
		if b.leagueErr != nil {
			b.errs = append(b.errs, b.leagueErr.Error())
		}
	}
	return b.leagueID, b.leagueErr
}

func (b *Book) load(c Category) []Line {
	if l, ok := b.lines[c]; ok {
		return l
	}
	if t, ok := b.failed[c]; ok && b.now().Sub(t) < retryFailed {
		return nil
	}
	league, err := b.league()
	if err != nil {
		return nil
	}
	raw, at, err := b.src.overview(league, c)
	var ov overview
	var ex exchangeOverview
	if err == nil && c == Exchange {
		err = json.Unmarshal(raw, &ex)
	} else if err == nil {
		err = json.Unmarshal(raw, &ov)
	}
	if err != nil {
		b.failed[c] = b.now()
		b.errs = append(b.errs, fmt.Sprintf("%s: %v", c, err))
		return nil
	}
	if c == Exchange {
		b.exchange = exchangeRates(ex)
	}
	b.lines[c], b.fetched[c] = ov.Lines, at
	return ov.Lines
}

// Info is the league priced and the time its oldest loaded category was fetched (zero before any loaded).
func (b *Book) Info() (string, time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	league, _ := b.league()
	var oldest time.Time
	for _, t := range b.fetched {
		if oldest.IsZero() || t.Before(oldest) {
			oldest = t
		}
	}
	return league, oldest
}

// Warning returns the load failures not yet reported, once; empty when there are none.
func (b *Book) Warning() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.warned || len(b.errs) == 0 {
		return ""
	}
	b.warned = true
	return "poe.ninja prices unavailable, affected prices are unknown: " + strings.Join(b.errs, "; ")
}

const (
	DivineFromExchange = "poe.ninja currency exchange"
	DivineFromItems    = "median of poe.ninja item lines' chaos/divine values"
	DivineFromStash    = "poe.ninja stash currency line"
)

// maxSideRatio is how far a stash currency line's pay and receive sides may disagree before the line is not used.
const maxSideRatio = 2.0

// exchangeRates is each currency's chaos price by name from an exchange overview, whose values are in its primary
// currency.
func exchangeRates(ov exchangeOverview) map[string]float64 {
	byID := map[string]float64{}
	for _, l := range ov.Lines {
		byID[l.ID] = l.PrimaryValue
	}
	perChaos := 1.0
	if ov.Core.Primary != "chaos" {
		perChaos = byID["chaos"]
	}
	out := map[string]float64{}
	for _, it := range ov.Items {
		if v := byID[it.ID]; v > 0 && perChaos > 0 {
			out[it.Name] = v / perChaos
		}
	}
	return out
}

// stashRate is a stash currency line's chaos price, unless its two sides disagree by more than maxSideRatio.
func stashRate(l Line) (float64, bool) {
	if l.ChaosEquivalent <= 0 {
		return 0, false
	}
	if l.Pay != nil && l.Receive != nil && l.Pay.Value > 0 && l.Receive.Value > 0 {
		pay, recv := 1/l.Pay.Value, l.Receive.Value
		if max(pay, recv) > maxSideRatio*min(pay, recv) {
			return 0, false
		}
	}
	return l.ChaosEquivalent, true
}

func (b *Book) currency(name string) (float64, string, bool) {
	if name == "Chaos Orb" {
		return 1, DivineFromExchange, true
	}
	b.load(Exchange)
	if v, ok := b.exchange[name]; ok {
		return v, DivineFromExchange, true
	}
	if name == "Divine Orb" {
		if v, ok := b.itemLineDivine(); ok {
			return v, DivineFromItems, true
		}
	}
	for _, l := range b.load(Currency) {
		if l.CurrencyTypeName == name {
			if v, ok := stashRate(l); ok {
				return v, DivineFromStash, true
			}
		}
	}
	return 0, "", false
}

// itemLineDivine is the median of chaosValue/divineValue over every loaded item line that carries both.
func (b *Book) itemLineDivine() (float64, bool) {
	var ratios []float64
	for _, c := range Categories {
		if c == Exchange || c == Currency {
			continue
		}
		for _, l := range b.load(c) {
			if l.ChaosValue > 0 && l.DivineValue > 0 {
				ratios = append(ratios, l.ChaosValue/l.DivineValue)
			}
		}
	}
	if len(ratios) == 0 {
		return 0, false
	}
	sort.Float64s(ratios)
	n := len(ratios)
	if n%2 == 1 {
		return ratios[n/2], true
	}
	return (ratios[n/2-1] + ratios[n/2]) / 2, true
}

// Currency is one unit of the named currency in chaos: the Currency Exchange's rate, else the stash currency line's
// when its two sides agree within maxSideRatio.
func (b *Book) Currency(name string) (float64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	v, _, ok := b.currency(name)
	return v, ok
}

// Divine is one Divine Orb in chaos.
func (b *Book) Divine() (float64, bool) {
	v, _, ok := b.DivineRate()
	return v, ok
}

// DivineRate is one Divine Orb in chaos and where the rate came from: the Currency Exchange, else the median
// chaos/divine ratio of poe.ninja's item lines, else the stash currency line when its sides agree.
func (b *Book) DivineRate() (float64, string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.currency("Divine Orb")
}

// Unique prices a unique by name. links (5 or 6) picks the linked listing where one exists; variant picks the row whose
// variant matches it, else the row with the most listings, so the price is the common copy's.
func (b *Book) Unique(name, variant string, links int) (float64, bool) {
	q, ok := b.UniqueQuote(name, variant, links)
	return q.Chaos, ok
}

// UniqueQuote is Unique with the row's listing and link counts.
func (b *Book) UniqueQuote(name, variant string, links int) (Quote, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var rows []Line
	for _, c := range uniqueCategories {
		for _, l := range b.load(c) {
			if l.ItemClass == 3 && strings.EqualFold(l.Name, name) {
				rows = append(rows, l)
			}
		}
	}
	if links < 5 {
		links = 0
	}
	linked := filter(rows, func(l Line) bool { return l.Links == links })
	if len(linked) == 0 && links > 0 {
		linked = filter(rows, func(l Line) bool { return l.Links == 0 })
	}
	if variant != "" {
		if v := filter(linked, func(l Line) bool { return strings.EqualFold(l.Variant, variant) }); len(v) > 0 {
			linked = v
		}
	}
	l, ok := mostListed(linked)
	return quote(l), ok
}

// Gem prices a gem by its market name. It takes the exact level, quality bracket (0, 20 or 23 as poe.ninja lists them)
// and corruption, unless the gem is uncorrupted and a corrupted row at or above its level and bracket is cheaper (the
// same gem for less, it only cannot be improved). Failing an exact row, the nearest listed row no better than it
// (highest level, then quality, at or below the gem's; the same corruption, or for a corrupted gem an uncorrupted row
// when no corrupted one is lower), so a gem is never priced as a better copy. Any row but the exact level and quality
// returns its variant ("1/20"). Nothing lower is unknown.
func (b *Book) Gem(name string, level, quality int, corrupted bool) (float64, string, bool) {
	q, ok := b.GemQuote(name, level, quality, corrupted)
	return q.Chaos, q.Variant, ok
}

func gemVariant(l Line) string {
	if l.Variant != "" {
		return l.Variant
	}
	v := fmt.Sprint(l.GemLevel)
	if l.GemQuality > 0 {
		v += fmt.Sprintf("/%d", l.GemQuality)
	}
	if l.Corrupted {
		v += "c"
	}
	return v
}

// GemQuote is Gem with the row's listing count.
func (b *Book) GemQuote(name string, level, quality int, corrupted bool) (Quote, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	q := 0
	switch {
	case quality >= 23:
		q = 23
	case quality >= 20:
		q = 20
	}
	var rows []Line
	for _, l := range b.load(SkillGem) {
		if strings.EqualFold(l.Name, name) {
			rows = append(rows, l)
		}
	}
	sub := func(l Line) Quote {
		out := quote(l)
		if l.GemLevel != level || l.GemQuality != quality {
			out.Variant = gemVariant(l)
		}
		return out
	}
	if l, ok := mostListed(filter(rows, func(l Line) bool {
		return l.GemLevel == level && l.GemQuality == q && l.Corrupted == corrupted
	})); ok {
		if !corrupted {
			if c, ok := cheapest(filter(rows, func(c Line) bool {
				return c.Corrupted && c.GemLevel >= level && c.GemQuality >= q && c.ChaosValue < l.ChaosValue
			})); ok {
				out := quote(c)
				out.Variant, out.Corrupted = gemVariant(c), true
				return out, true
			}
		}
		return sub(l), true
	}
	lower := func(corrupt bool) []Line {
		return filter(rows, func(l Line) bool { return l.GemLevel <= level && l.GemQuality <= q && l.Corrupted == corrupt })
	}
	near := lower(corrupted)
	if len(near) == 0 && corrupted {
		near = lower(false)
	}
	if len(near) == 0 {
		return Quote{}, false
	}
	sort.SliceStable(near, func(i, j int) bool {
		if near[i].GemLevel != near[j].GemLevel {
			return near[i].GemLevel > near[j].GemLevel
		}
		if near[i].GemQuality != near[j].GemQuality {
			return near[i].GemQuality > near[j].GemQuality
		}
		return near[i].ListingCount > near[j].ListingCount
	})
	out := quote(near[0])
	out.Variant = gemVariant(near[0])
	return out, true
}

func cheapest(rows []Line) (Line, bool) {
	if len(rows) == 0 {
		return Line{}, false
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].ChaosValue != rows[j].ChaosValue {
			return rows[i].ChaosValue < rows[j].ChaosValue
		}
		return rows[i].ListingCount > rows[j].ListingCount
	})
	return rows[0], true
}

// Base prices a base type at its item level and influence ("" for none): the listing at the highest item level
// poe.ninja lists at or below it (it lists 82 to 86). An item level of 0 (the item names none) takes the lowest listed,
// so a missing Item Level line prices the base as its cheapest listed copy. Bases it does not list are not priced.
func (b *Book) Base(name string, ilvl int, influence string) (float64, bool) {
	q, ok := b.BaseQuote(name, ilvl, influence)
	return q.Chaos, ok
}

// BaseQuote is Base with the row's listing count.
func (b *Book) BaseQuote(name string, ilvl int, influence string) (Quote, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var pick Line
	found := false
	for _, l := range b.load(BaseType) {
		if !strings.EqualFold(l.Name, name) || !strings.EqualFold(l.Variant, influence) {
			continue
		}
		better := l.LevelRequired > pick.LevelRequired
		if ilvl <= 0 {
			better = l.LevelRequired < pick.LevelRequired
		} else if l.LevelRequired > ilvl {
			continue
		}
		if !found || better {
			pick, found = l, true
		}
	}
	return quote(pick), found
}

// Cluster prices a cluster jewel base by its size ("Large Cluster Jewel"), its small passives' enchant ("12% increased
// Fire Damage"), its passive count and item level: the listing for the highest item level band at or below it, or the
// lowest band for an item level of 0 (the item names none).
func (b *Book) Cluster(base, enchant string, passives, ilvl int) (float64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	variant := fmt.Sprintf("%d passives", passives)
	band, price, ok := -1, 0.0, false
	for _, l := range b.load(ClusterJewel) {
		if strings.EqualFold(l.BaseType, base) && strings.EqualFold(l.Name, enchant) && l.Variant == variant &&
			((ilvl <= 0 && (!ok || l.LevelRequired < band)) || (ilvl > 0 && l.LevelRequired <= ilvl && l.LevelRequired > band)) {
			band, price, ok = l.LevelRequired, l.ChaosValue, true
		}
	}
	return price, ok
}

func filter(rows []Line, keep func(Line) bool) []Line {
	var out []Line
	for _, l := range rows {
		if keep(l) {
			out = append(out, l)
		}
	}
	return out
}

func mostListed(rows []Line) (Line, bool) {
	if len(rows) == 0 {
		return Line{}, false
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].ListingCount != rows[j].ListingCount {
			return rows[i].ListingCount > rows[j].ListingCount
		}
		return rows[i].ChaosValue < rows[j].ChaosValue
	})
	return rows[0], true
}

// Snapshot is a pinned set of poe.ninja responses: every category's raw overview, the league and when it was fetched.
type Snapshot struct {
	League      string                       `json:"league"`
	FetchedAt   time.Time                    `json:"fetched_at"`
	DivineChaos float64                      `json:"divine_chaos"`
	Responses   map[Category]json.RawMessage `json:"responses"`
}

type snapshotSource struct{ snap Snapshot }

func (s snapshotSource) league() (string, error) {
	if s.snap.League == "" {
		return "", fmt.Errorf("snapshot names no league")
	}
	return s.snap.League, nil
}

func (s snapshotSource) overview(_ string, c Category) ([]byte, time.Time, error) {
	r, ok := s.snap.Responses[c]
	if !ok {
		return nil, time.Time{}, fmt.Errorf("not in the snapshot")
	}
	return r, s.snap.FetchedAt, nil
}

// FromSnapshot reads a snapshot; the book it returns never fetches.
func FromSnapshot(r io.Reader) (*Book, error) {
	var snap Snapshot
	if err := json.NewDecoder(r).Decode(&snap); err != nil {
		return nil, fmt.Errorf("price snapshot: %w", err)
	}
	if snap.League == "" || len(snap.Responses) == 0 {
		return nil, fmt.Errorf("price snapshot: league or responses missing")
	}
	return newBook(snapshotSource{snap}), nil
}

// WriteSnapshot loads every category and writes them as a snapshot; it fails when any category fails to load.
func (b *Book) WriteSnapshot(w io.Writer) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	league, err := b.league()
	if err != nil {
		return err
	}
	snap := Snapshot{League: league, Responses: map[Category]json.RawMessage{}}
	for _, c := range Categories {
		raw, at, err := b.src.overview(league, c)
		if err != nil {
			return fmt.Errorf("%s: %w", c, err)
		}
		snap.Responses[c] = raw
		if snap.FetchedAt.IsZero() || at.Before(snap.FetchedAt) {
			snap.FetchedAt = at
		}
	}
	if v, _, ok := newBook(snapshotSource{snap}).DivineRate(); ok {
		snap.DivineChaos = v
	}
	if err := json.NewEncoder(w).Encode(snap); err != nil {
		return fmt.Errorf("price snapshot: %w", err)
	}
	return nil
}
