package mcp

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/mcp/prices"
	"github.com/Vilsol/go-pob/pob"
)

// priceSnapshot is a hand-written snapshot in poe.ninja's response shape; every value is invented.
func priceSnapshot(t *testing.T, withCurrency bool) *prices.Book {
	t.Helper()
	responses := map[string]string{
		"UniqueWeapon": `{"lines":[{"name":"The Saviour","baseType":"Legion Sword","itemClass":3,"chaosValue":300,"listingCount":10}]}`,
		"UniqueAccessory": `{"lines":[{"name":"Bear's Girdle","baseType":"Leather Belt","itemClass":3,"chaosValue":5,"listingCount":3},
			{"name":"Kikazaru","baseType":"Topaz Ring","itemClass":3,"chaosValue":2,"listingCount":30},
			{"name":"Foulborn Kikazaru","baseType":"Topaz Ring","variant":"Mana per Level","itemClass":3,"chaosValue":70,"listingCount":12}]}`,
		"UniqueArmour": `{"lines":[{"name":"Belly of the Beast","baseType":"Full Wyrmscale","itemClass":3,"chaosValue":10,"listingCount":40},
			{"name":"Belly of the Beast","baseType":"Full Wyrmscale","links":6,"itemClass":3,"chaosValue":400,"listingCount":8},
			{"name":"Lightning Coil","baseType":"Desert Brigandine","itemClass":3,"chaosValue":2,"listingCount":90}]}`,
		"UniqueFlask":  `{"lines":[]}`,
		"UniqueJewel":  `{"lines":[]}`,
		"BaseType":     `{"lines":[{"name":"Hubris Circlet","levelRequired":82,"chaosValue":5,"listingCount":20},{"name":"Hubris Circlet","levelRequired":86,"chaosValue":9,"listingCount":20}]}`,
		"ClusterJewel": `{"lines":[]}`,
		"SkillGem": `{"lines":[
			{"name":"Brutality Support","gemLevel":20,"chaosValue":2,"listingCount":10},
			{"name":"Berserk","gemLevel":21,"corrupted":true,"chaosValue":15,"listingCount":10},
			{"name":"Precision","variant":"1/20","gemLevel":1,"gemQuality":20,"chaosValue":2,"listingCount":60},
			{"name":"Precision","variant":"20/20","gemLevel":20,"gemQuality":20,"chaosValue":180,"listingCount":30},
			{"name":"Precision","variant":"20/20c","gemLevel":20,"gemQuality":20,"corrupted":true,"chaosValue":110,"listingCount":12},
			{"name":"Precision","variant":"21/20c","gemLevel":21,"gemQuality":20,"corrupted":true,"chaosValue":140,"listingCount":3},
			{"name":"Void Shockwave Support","variant":"3/20","gemLevel":3,"gemQuality":20,"chaosValue":700,"listingCount":4},
			{"name":"Void Shockwave Support","variant":"1","gemLevel":1,"chaosValue":300,"listingCount":50}]}`,
	}
	if withCurrency {
		responses["Currency"] = `{"lines":[{"currencyTypeName":"Divine Orb","chaosEquivalent":100},{"currencyTypeName":"Orb of Transmutation","chaosEquivalent":0.5}]}`
		responses["Exchange"] = `{"core":{"primary":"chaos"},"lines":[{"id":"divine","primaryValue":100},{"id":"transmute","primaryValue":0.5},
			{"id":"fusing","primaryValue":0.5}],
			"items":[{"id":"divine","name":"Divine Orb"},{"id":"transmute","name":"Orb of Transmutation"},{"id":"fusing","name":"Orb of Fusing"}]}`
	}
	snap := map[string]any{"league": "Testleague", "fetched_at": "2026-01-01T00:00:00Z", "divine_chaos": 100, "responses": map[string]json.RawMessage{}}
	for k, v := range responses {
		snap["responses"].(map[string]json.RawMessage)[k] = json.RawMessage(v)
	}
	b, _ := json.Marshal(snap)
	book, err := prices.FromSnapshot(strings.NewReader(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	return book
}

type panicTransport struct{}

func (panicTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	panic("pricing reached the network: " + r.URL.String())
}

// offline fails the test if anything reaches the default HTTP transport while it runs.
func offline(t *testing.T) {
	t.Helper()
	prev := http.DefaultTransport
	http.DefaultTransport = panicTransport{}
	t.Cleanup(func() { http.DefaultTransport = prev })
}

func priced(t *testing.T, parts []any, name string) map[string]any {
	t.Helper()
	for _, p := range parts {
		m := p.(map[string]any)
		if strings.HasPrefix(m["name"].(string), name) {
			return m
		}
	}
	t.Fatalf("%s not priced in %v", name, parts)
	return nil
}

func TestPriceBuild(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = priceSnapshot(t, true)
	h := importFixture(t, s)
	res := assertSize(t, s, "price_build", map[string]any{"handle": h}, 6000)
	cost := res["cost"].(map[string]any)
	items, gems := cost["items"].([]any), cost["gems"].([]any)
	if p := priced(t, items, "The Saviour"); p["chaos"] != 300.0 || p["price_source"] != "ninja" {
		t.Errorf("The Saviour = %v", p)
	}
	if p := priced(t, items, "Bear's Girdle"); p["chaos"] != 5.0 {
		t.Errorf("Bear's Girdle = %v", p)
	}
	if p := priced(t, items, "Lion's Roar"); p["chaos"] != nil || p["price_source"] != "unknown" {
		t.Errorf("an unlisted unique must be unknown, not free: %v", p)
	}
	if p := priced(t, items, "Pain Cloak"); p["price_source"] != "estimate" || p["chaos"] == nil || !strings.HasPrefix(p["detail"].(string), "base 0") {
		t.Errorf("rare = %v", p)
	}
	if p := priced(t, gems, "Brutality 20/11"); p["chaos"] != 2.0 {
		t.Errorf("quality 11 is poe.ninja's no-quality listing: %v", p)
	}
	if p := priced(t, gems, "Berserk 21/7"); p["chaos"] != 15.0 {
		t.Errorf("a level 21 gem is the corrupted listing: %v", p)
	}
	total, unknown := 0.0, 0
	for _, p := range append(items, gems...) {
		if c, ok := p.(map[string]any)["chaos"].(float64); ok {
			total += c
		} else {
			unknown++
		}
	}
	if got := cost["total_chaos"].(float64); got < total-0.5 || got > total+0.5 || cost["unknown"] != float64(unknown) || unknown == 0 {
		t.Errorf("total %v unknown %v, parts sum to %v with %d unknown", got, cost["unknown"], total, unknown)
	}
	if div, _ := cost["total_divine"].(float64); cost["league"] != "Testleague" || cost["divine_chaos"] != 100.0 || cost["divine_source"] != prices.DivineFromExchange || div < total/100-0.01 || div > total/100+0.01 {
		t.Errorf("league / divines = %v %v %v", cost["league"], cost["divine_chaos"], cost["total_divine"])
	}
	if cost["snapshot_age_min"] == nil || res["source"] != "poe.ninja" {
		t.Errorf("age / credit missing: %v", res)
	}
}

func TestRareEstimateByTier(t *testing.T) {
	offline(t)
	testServer(t)
	book := priceSnapshot(t, true)
	ring := func(fire string) string {
		return "Rarity: RARE\nDusk Circle\nRuby Ring\nItem Level: 84\nImplicits: 1\n+30% to Fire Resistance\n" + fire +
			"\n+28 to Intelligence\n{crafted}Adds 7 to 10 Physical Damage to Attacks\n"
	}
	for _, c := range []struct {
		fire   string
		chaos  float64
		detail string
	}{
		// +(46-48)% is the ring's best fire resistance tier, +(42-45)% the next; +28 Intelligence is far below T2; the
		// bench's Adds (5-7) to (8-10) Physical Damage costs 2 Chaos Orbs
		{"+47% to Fire Resistance", 20 + 1 + 2, "base 0 (unlisted) + T1×1 + T3-×1 + bench 2"},
		{"+43% to Fire Resistance", 5 + 1 + 2, "base 0 (unlisted) + T2×1 + T3-×1 + bench 2"},
	} {
		p := PriceItem(book, calculator.PeekItem(ring(c.fire)))
		if p.Chaos == nil || *p.Chaos != c.chaos || p.Detail != c.detail || p.Source != "estimate" {
			t.Errorf("%s: %+v (chaos %v), want %v %q", c.fire, p, deref(p.Chaos), c.chaos, c.detail)
		}
	}
	// a magic item is its base plus a fixed magic cost, whatever its affixes' tiers
	flask := "Rarity: MAGIC\nAbecedarian's Diamond Flask of the Iguana\nItem Level: 84\nImplicits: 0\n38% reduced Duration\n" +
		"37% less Duration\n25% increased effect\nImmunity to Bleeding and Corrupted Blood during Effect\n"
	if p := PriceItem(book, calculator.PeekItem(flask)); p.Chaos == nil || *p.Chaos != magicItemChaos || p.Detail != "base 0 (unlisted) + magic 1" {
		t.Errorf("magic flask: %+v (chaos %v)", p, deref(p.Chaos))
	}
	normal := "Rarity: NORMAL\nDiamond Flask\nItem Level: 84\nImplicits: 0\n"
	if p := PriceItem(book, calculator.PeekItem(normal)); p.Chaos == nil || *p.Chaos != 0 || p.Detail != "base 0 (unlisted)" {
		t.Errorf("normal flask: %+v (chaos %v)", p, deref(p.Chaos))
	}
}

// TestWeaponDamageAffixes: a weapon's local damage affixes are priced by the weapon damage tiers, its other affixes by
// the flat table.
func TestWeaponDamageAffixes(t *testing.T) {
	offline(t)
	testServer(t)
	// (75-79)% increased Physical Damage with +(175-200) Accuracy is the staff's best hybrid tier, Adds (34-47) to
	// (72-84) the second flat tier, 37% increased Critical Strike Chance the best crit tier
	staff := "Rarity: RARE\nStorm Spire\nMaelstrom Staff\nItem Level: 84\nSockets: R B W W W W\nImplicits: 1\n+25% Chance to Block Attack Damage\n" +
		"Adds 37 to 70 Physical Damage\n77% increased Physical Damage\n+188 to Accuracy Rating\n37% increased Critical Strike Chance\n"
	p := PriceItem(priceSnapshot(t, true), calculator.PeekItem(staff))
	want := weaponDamageTier1Chaos + weaponDamageTier2Chaos + tier1Chaos
	if p.Chaos == nil || *p.Chaos != want || p.Detail != "base 0 (unlisted) + dmg T1×1 + dmg T2×1 + T1×1" {
		t.Errorf("staff: %v %q, want %v", deref(p.Chaos), p.Detail, want)
	}
}

func deref(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func TestBenchCraftUnknownWithoutCurrencyPrices(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = priceSnapshot(t, false)
	h := importFixture(t, s)
	res := call(t, s, "price_build", map[string]any{"handle": h})
	ring := "Rarity: RARE\nDusk Circle\nRuby Ring\nItem Level: 84\nImplicits: 1\n+30% to Fire Resistance\n+28 to Intelligence\n" +
		"{crafted}Adds 4 to 7 Physical Damage to Attacks\n"
	// Adds (3-5) to (6-8) Physical Damage to Attacks is the bench's 4 Orbs of Transmutation craft
	if p := PriceItem(s.opts.Prices, calculator.PeekItem(ring)); p.Chaos != nil || p.Source != "unknown" {
		t.Errorf("a craft paid in currency without a price must be unknown: %+v", p)
	}
	if p := PriceItem(priceSnapshot(t, true), calculator.PeekItem(ring)); p.Chaos == nil || *p.Chaos != 1+2 || p.Detail != "base 0 (unlisted) + T3-×1 + bench 2" {
		t.Errorf("with currency prices: %+v (chaos %v)", p, deref(p.Chaos))
	}
	if w, _ := res["server_warnings"].([]any); len(w) != 1 || !strings.Contains(w[0].(string), "Currency") {
		t.Errorf("server_warnings = %v", res["server_warnings"])
	}
	if res = call(t, s, "price_build", map[string]any{"handle": h}); res["server_warnings"] != nil {
		t.Errorf("the warning repeated: %v", res["server_warnings"])
	}
}

func TestPriceBuildWithoutPrices(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	if e := callErr(t, s, "price_build", map[string]any{"handle": h}); e.Code != CodePricesUnavailable {
		t.Fatalf("code = %s", e.Code)
	}
}

func TestThinMarketsAreLowConfidence(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = priceSnapshot(t, true)
	h, _ := s.get(importFixture(t, s))
	b, e := h.parseRev(h.cur)
	if e != nil {
		t.Fatal(e)
	}
	cost := PriceBuild(s.opts.Prices, b)
	thin := 0
	for _, p := range append(cost.Items, cost.Gems...) {
		switch {
		case p.Source != priceNinja:
			if p.Listings != nil || p.LowConfidence {
				t.Errorf("%s: a %s price carries listings %v", p.Name, p.Source, p.Listings)
			}
		case p.Listings == nil:
			t.Errorf("%s: a ninja price without its listing count", p.Name)
		case p.LowConfidence != (*p.Listings < prices.ThinMarket):
			t.Errorf("%s: %d listings, low_confidence %v", p.Name, *p.Listings, p.LowConfidence)
		}
		if p.LowConfidence {
			thin++
		}
	}
	if thin == 0 || cost.LowConf != thin {
		t.Errorf("build low_confidence %d, rows %d (the fixture lists a 3-listing unique)", cost.LowConf, thin)
	}
}

// TestRareEstimateBuckets: each bucket adds its own cost to the same rare and names itself in detail.
func TestRareEstimateBuckets(t *testing.T) {
	offline(t)
	testServer(t)
	book := priceSnapshot(t, true)
	price := func(text string) (float64, string) {
		t.Helper()
		p := PriceItem(book, calculator.PeekItem(text))
		if p.Chaos == nil || p.Source != priceEstimate {
			t.Fatalf("not estimated: %+v\n%s", p, text)
		}
		return *p.Chaos, p.Detail
	}
	armour := func(sockets, implicits string) string {
		n := strings.Count(implicits, "\n")
		return "Rarity: RARE\nDoom Shell\nVaal Regalia\nItem Level: 86\nSockets: " + sockets + "\nImplicits: " + strconv.Itoa(n) + "\n" +
			implicits + "+80 to maximum Energy Shield\n+40% to Cold Resistance\n"
	}
	plain, _ := price(armour("B B B B B B", ""))
	for _, c := range []struct {
		name, text string
		add        float64
		detail     string
	}{
		// 6% Action Speed is the exarch's top two tiers, +(15-16)% all resistances the eater's Perfect tier
		{"two Perfect eldritch implicits", armour("B B B B B B", "{exarch}6% increased Action Speed\n{eater}+16% to all Elemental Resistances\n"),
			2 * eldritchTierChaos[6], "eldritch exarch T6 + eater T6"},
		{"an Exceptional eater implicit", armour("B B B B B B", "{eater}+11% to all Elemental Resistances\n"), eldritchTierChaos[4], "eldritch eater T4"},
		// the bench links six sockets for 1500 Orbs of Fusing and five for 150; the fixture prices a Fusing at 0.5c
		{"a 6-link", armour("B-B-B-B-B-B", ""), 750, "6L 750"},
		{"a 5-link", armour("B-B-B-B-B B", ""), 75, "5L 75"},
	} {
		got, detail := price(c.text)
		if got != plain+c.add || !strings.Contains(detail, c.detail) {
			t.Errorf("%s: %v %q, want %v + %v with %q", c.name, got, detail, plain, c.add, c.detail)
		}
	}
	amulet := func(lines string) string {
		return "Rarity: RARE\nRune Locket\nOnyx Amulet\nItem Level: 84\nImplicits: 2\n{crafted}Allocates High Voltage\n+(10-16) to all Attributes\n" +
			"+70 to maximum Life\n" + lines
	}
	bare, _ := price(strings.Replace(amulet(""), "Implicits: 2\n{crafted}Allocates High Voltage\n", "Implicits: 1\n", 1))
	if got, detail := price(amulet("")); got != bare+anointChaos || !strings.Contains(detail, "anoint×1") {
		t.Errorf("anoint: %v %q, want %v + %v", got, detail, bare, anointChaos)
	}
	// a gem line's own affix tier is priced as any affix's; the gem bucket is on top of it
	for _, c := range []struct {
		name, lines string
		bucket      float64
		detail      string
	}{
		{"one +1 gem prefix", "+1 to Level of all Lightning Skill Gems\n", gemLevelChaos, "gem levels +1 (50)"},
		{"two +1 gem prefixes count double", "+1 to Level of all Lightning Skill Gems\n+1 to Level of all Physical Spell Skill Gems\n",
			2 * 2 * gemLevelChaos, "gem levels +2 on 2 lines ×2 (200)"},
	} {
		if got, detail := price(amulet(c.lines)); !strings.Contains(detail, c.detail) || got < bare+anointChaos+c.bucket {
			t.Errorf("%s: %v %q, want at least %v + %v with %q", c.name, got, detail, bare+anointChaos, c.bucket, c.detail)
		}
	}
	// Frenzy Charges are no body armour affix: an unplaced mod, priced as T1 and named
	if _, detail := price(armour("B B B B B B", "") + "+1 to Maximum Frenzy Charges\n"); !strings.Contains(detail, "other×1 (unplaced: influence/essence/veiled/delve mods, as T1)") {
		t.Errorf("other bucket unexplained: %s", detail)
	}
}

func TestPriceLookups(t *testing.T) {
	offline(t)
	testServer(t)
	book := priceSnapshot(t, true)
	for _, c := range []struct {
		gem    pob.Gem
		chaos  float64
		detail string
	}{
		{pob.Gem{NameSpec: "Precision", GemID: "Metadata/Items/Gems/SkillGemPrecision", Level: 16, Quality: 20}, 2, "priced as 1/20"},
		{pob.Gem{NameSpec: "Void Shockwave", GemID: "Metadata/Items/Gems/SupportGemVoidShockwave", Level: 3, Quality: 20}, 700, ""},
		{pob.Gem{NameSpec: "Void Shockwave", Level: 2, Quality: 0}, 300, "priced as 1"},
		{pob.Gem{NameSpec: "Precision", Level: 20, Quality: 20}, 110, "priced as 20/20c (corrupted listing, cheaper)"},
		{pob.Gem{NameSpec: "Void Shockwave", Level: 3, Quality: 23}, 700, "priced as 3/20"},
		{pob.Gem{NameSpec: "Precision", GemID: "Metadata/Items/Gems/SkillGemPrecision", Level: 16, Quality: 0}, 0, vendorGemDetail},
		{pob.Gem{NameSpec: "Brutality", GemID: "Metadata/Items/Gems/SupportGemBrutality", Level: 1, Quality: 19}, 0, vendorGemDetail},
		{pob.Gem{NameSpec: "Brutality", GemID: "Metadata/Items/Gems/SupportGemBrutality", Level: 20, Quality: 0}, 2, ""},
	} {
		if p := PriceGem(book, c.gem); p.Chaos == nil || *p.Chaos != c.chaos || p.Detail != c.detail {
			t.Errorf("%s %d/%d = %+v (chaos %v), want %v %q", c.gem.NameSpec, c.gem.Level, c.gem.Quality, p, deref(p.Chaos), c.chaos, c.detail)
		}
	}
}

// TestBaseAtItsOwnItemLevel: poe.ninja lists crafting bases at item level 82+ only, so a base with no Item Level line,
// one below 82 or one it does not list takes unlistedBaseChaos and says why, never the 82+ crafting price.
func TestBaseAtItsOwnItemLevel(t *testing.T) {
	offline(t)
	testServer(t)
	book := priceSnapshot(t, true)
	for _, c := range []struct {
		text   string
		chaos  float64
		detail string
	}{
		{"Rarity: NORMAL\nHubris Circlet\nItem Level: 84\nImplicits: 0\n", 5, "base 5"},
		{"Rarity: NORMAL\nHubris Circlet\nItem Level: 86\nImplicits: 0\n", 9, "base 9"},
		{"Rarity: NORMAL\nHubris Circlet\nImplicits: 0\n", unlistedBaseChaos, "base 0 (no ilvl, ninja lists 82+ only)"},
		{"Rarity: NORMAL\nHubris Circlet\nItem Level: 75\nImplicits: 0\n", unlistedBaseChaos, "base 0 (ilvl 75 < ninja's 82+)"},
		{"Rarity: NORMAL\nIron Ring\nItem Level: 84\nImplicits: 0\n", unlistedBaseChaos, "base 0 (unlisted)"},
		{"Rarity: RARE\nDoom Crown\nHubris Circlet\nImplicits: 0\n+80 to maximum Energy Shield\n", unlistedBaseChaos + tier1Chaos,
			"base 0 (no ilvl, ninja lists 82+ only) + T1×1"},
	} {
		if p := PriceItem(book, calculator.PeekItem(c.text)); p.Chaos == nil || *p.Chaos != c.chaos || p.Detail != c.detail {
			t.Errorf("%q: %v %q, want %v %q", c.text, deref(p.Chaos), p.Detail, c.chaos, c.detail)
		}
	}
}

// TestJewelInARemovedSocket: a jewel whose socket node is not allocated (a removed cluster's or an unallocated tree
// socket) is not worn and not priced.
func TestJewelInARemovedSocket(t *testing.T) {
	offline(t)
	s := testServer(t)
	h, _ := s.get(importFixture(t, s))
	b, e := h.parseRev(h.cur)
	if e != nil {
		t.Fatal(e)
	}
	spec := activeSpec(b)
	jewels := func() int {
		n := 0
		for _, p := range PriceBuild(priceSnapshot(t, true), b).Items {
			if strings.HasPrefix(p.Slot, "Jewel ") {
				n++
			}
		}
		return n
	}
	before := jewels()
	if before == 0 || len(spec.Sockets) == 0 {
		t.Fatal("the fixture prices no tree jewel")
	}
	spec.Sockets = append(spec.Sockets, pob.Socket{ItemID: spec.Sockets[0].ItemID, NodeID: 65536 + 12345})
	if got := jewels(); got != before {
		t.Errorf("a jewel in an unallocated socket was priced: %d jewels, want %d", got, before)
	}
}

// TestLinkedUniqueBody: poe.ninja lists a linked unique body as its own row; with no row at the item's link count the
// price is the unlinked row plus the bench's link craft, never the unlinked row alone.
func TestLinkedUniqueBody(t *testing.T) {
	offline(t)
	testServer(t)
	book := priceSnapshot(t, true)
	body := func(name, base, sockets string) *calculator.Item {
		return calculator.PeekItem("Rarity: UNIQUE\n" + name + "\n" + base + "\nSockets: " + sockets + "\nImplicits: 0\n+50 to maximum Life\n")
	}
	for _, c := range []struct {
		name, base, sockets string
		chaos               float64
		detail              string
	}{
		{"Belly of the Beast", "Full Wyrmscale", "R-R-R-R-R-R", 400, "6L listing"},
		{"Belly of the Beast", "Full Wyrmscale", "R R R R R R", 10, "unlinked listing"},
		// the bench links six sockets for 1500 Orbs of Fusing and five for 150; the fixture prices a Fusing at 0.5c
		{"Belly of the Beast", "Full Wyrmscale", "R-R-R-R-R R", 10 + 75, "unlinked + 5L bench 75"},
		{"Lightning Coil", "Desert Brigandine", "R-G-B-R-G-B", 2 + 750, "unlinked + 6L bench 750"},
	} {
		p := PriceItem(book, body(c.name, c.base, c.sockets))
		if p.Chaos == nil || *p.Chaos != c.chaos || p.Detail != c.detail {
			t.Errorf("%s %s = %+v (chaos %v), want %v %q", c.name, c.sockets, p, deref(p.Chaos), c.chaos, c.detail)
		}
	}
}
