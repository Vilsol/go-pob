package prices

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Hand-written responses in poe.ninja's shape; every value is invented.
var fixtures = map[string]string{
	"leagues": `[{"id":"Testleague","name":"Testleague"},{"id":"Standard","name":"Standard"}]`,
	"Currency": `{"lines":[{"currencyTypeName":"Divine Orb","chaosEquivalent":100},
		{"currencyTypeName":"Orb of Alteration","chaosEquivalent":0.25}],"currencyDetails":[]}`,
	"Exchange": `{"core":{"items":[],"rates":{"divine":0.01},"primary":"chaos","secondary":"divine"},
		"lines":[{"id":"chaos","primaryValue":1},{"id":"divine","primaryValue":100},{"id":"alt","primaryValue":0.25}],
		"items":[{"id":"chaos","name":"Chaos Orb"},{"id":"divine","name":"Divine Orb"},{"id":"alt","name":"Orb of Alteration"}]}`,
	"UniqueArmour": `{"lines":[
		{"id":1,"name":"Test Plate","baseType":"Glorious Plate","itemClass":3,"chaosValue":10,"listingCount":50},
		{"name":"Test Plate","baseType":"Glorious Plate","itemClass":3,"links":6,"chaosValue":90,"listingCount":5},
		{"name":"Test Plate","baseType":"Glorious Plate","itemClass":9,"chaosValue":900,"listingCount":500},
		{"name":"Twin Hood","baseType":"Hubris Circlet","variant":"Cold","itemClass":3,"chaosValue":40,"listingCount":3},
		{"name":"Twin Hood","baseType":"Hubris Circlet","variant":"Fire","itemClass":3,"chaosValue":8,"listingCount":30}]}`,
	"UniqueWeapon":    `{"lines":[]}`,
	"UniqueAccessory": `{"lines":[]}`,
	"UniqueFlask":     `{"lines":[]}`,
	"UniqueJewel":     `{"lines":[]}`,
	"SkillGem": `{"lines":[
		{"name":"Test Bolt","variant":"20/20","gemLevel":20,"gemQuality":20,"chaosValue":12,"listingCount":4},
		{"name":"Test Bolt","variant":"20/20c","gemLevel":20,"gemQuality":20,"corrupted":true,"chaosValue":30,"listingCount":4},
		{"name":"Test Bolt","variant":"21/20c","gemLevel":21,"gemQuality":20,"corrupted":true,"chaosValue":60,"listingCount":4},
		{"name":"Test Bolt","variant":"20","gemLevel":20,"chaosValue":3,"listingCount":9},
		{"name":"Test Bolt","variant":"1","gemLevel":1,"chaosValue":1,"listingCount":99},
		{"name":"Cheap Bolt","variant":"20/20","gemLevel":20,"gemQuality":20,"chaosValue":50,"listingCount":40},
		{"name":"Cheap Bolt","variant":"20/20c","gemLevel":20,"gemQuality":20,"corrupted":true,"chaosValue":30,"listingCount":4},
		{"name":"Cheap Bolt","variant":"21/20c","gemLevel":21,"gemQuality":20,"corrupted":true,"chaosValue":40,"listingCount":4},
		{"name":"Cheap Bolt","gemLevel":19,"gemQuality":20,"corrupted":true,"chaosValue":5,"listingCount":4},
		{"name":"Cheap Bolt","variant":"20","gemLevel":20,"chaosValue":20,"listingCount":40},
		{"name":"Cheap Bolt","variant":"21c","gemLevel":21,"corrupted":true,"chaosValue":10,"listingCount":4}]}`,
	"BaseType": `{"lines":[
		{"name":"Test Boots","levelRequired":86,"chaosValue":7,"listingCount":5},
		{"name":"Test Boots","levelRequired":86,"variant":"Hunter","chaosValue":70,"listingCount":5},
		{"name":"Test Boots","levelRequired":84,"chaosValue":2,"listingCount":5}]}`,
	"ClusterJewel": `{"lines":[
		{"name":"12% increased Fire Damage","baseType":"Large Cluster Jewel","variant":"8 passives","levelRequired":84,"chaosValue":30},
		{"name":"12% increased Fire Damage","baseType":"Large Cluster Jewel","variant":"8 passives","levelRequired":75,"chaosValue":9},
		{"name":"12% increased Fire Damage","baseType":"Large Cluster Jewel","variant":"12 passives","levelRequired":84,"chaosValue":50}]}`,
}

// noNetwork fails any request that reaches the default transport: no test in this package may touch poe.ninja.
type noNetwork struct{}

func (noNetwork) RoundTrip(r *http.Request) (*http.Response, error) {
	panic("prices tests must not reach the network: " + r.URL.String())
}

func TestMain(m *testing.M) {
	http.DefaultTransport = noNetwork{}
	os.Exit(m.Run())
}

// fakeNinja serves the fixtures by the request's type parameter and counts requests.
type fakeNinja struct {
	base      string
	requests  atomic.Int32
	down      bool
	etag      string
	notModded atomic.Int32
}

func (f *fakeNinja) RoundTrip(r *http.Request) (*http.Response, error) {
	f.requests.Add(1)
	if f.down {
		return nil, fmt.Errorf("%s", "connection refused")
	}
	if r.Header.Get("User-Agent") != userAgent {
		return nil, fmt.Errorf("%s", "missing user agent")
	}
	base := f.base
	if base == "" {
		base = DefaultURL
	}
	if !strings.HasPrefix(r.URL.String(), base+apiPath+"/") {
		return nil, fmt.Errorf("%s", "unexpected host "+r.URL.String())
	}
	key := r.URL.Query().Get("type")
	switch {
	case strings.HasSuffix(r.URL.Path, "/leagues"):
		key = "leagues"
	case strings.Contains(r.URL.Path, "/exchange/") && key == "Currency":
		key = "Exchange"
	}
	if key != "leagues" && r.URL.Query().Get("league") != "Testleague" {
		return nil, fmt.Errorf("%s", "wrong league "+r.URL.RawQuery)
	}
	if f.etag != "" && r.Header.Get("If-None-Match") == f.etag {
		f.notModded.Add(1)
		return &http.Response{StatusCode: http.StatusNotModified, Status: "304 Not Modified", Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
	}
	body, ok := fixtures[key]
	if !ok {
		return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
	}
	h := http.Header{}
	if f.etag != "" {
		h.Set("ETag", f.etag)
	}
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body)), Header: h}, nil
}

func liveBook(t *testing.T, fake *fakeNinja, now *time.Time) (*Book, *Live) {
	t.Helper()
	l := &Live{Base: DefaultURL, Dir: t.TempDir(), TTL: DefaultTTL, Client: &http.Client{Transport: fake}, Now: func() time.Time { return *now }}
	return newBook(l), l
}

func TestLookups(t *testing.T) {
	now := time.Now()
	b, _ := liveBook(t, &fakeNinja{}, &now)
	check := func(name string, got float64, ok bool, want float64) {
		t.Helper()
		if !ok || got != want {
			t.Errorf("%s = %v (%v), want %v", name, got, ok, want)
		}
	}
	p, ok := b.Unique("Test Plate", "", 0)
	check("unlinked unique (relic row ignored)", p, ok, 10)
	p, ok = b.Unique("Test Plate", "", 6)
	check("6-link unique", p, ok, 90)
	p, ok = b.Unique("Test Plate", "", 5)
	check("5-link unique without a 5-link row", p, ok, 10)
	p, ok = b.Unique("Twin Hood", "Cold", 0)
	check("variant", p, ok, 40)
	p, ok = b.Unique("Twin Hood", "Lightning", 0)
	check("unknown variant takes the most listed", p, ok, 8)
	gem := func(name string, level, quality int, corrupted bool, want float64, wantSub string) {
		t.Helper()
		p, sub, ok := b.Gem("Test Bolt", level, quality, corrupted)
		check(name, p, ok, want)
		if sub != wantSub {
			t.Errorf("%s substitute = %q, want %q", name, sub, wantSub)
		}
	}
	gem("exact gem", 20, 20, false, 12, "")
	gem("corrupted gem", 21, 20, true, 60, "")
	gem("quality below 20 is the 0 bracket, and says so", 20, 13, false, 3, "20")
	gem("a pricier corrupted copy is not taken", 20, 20, false, 12, "")
	cheap := func(name string, level, quality int, want float64, wantSub string) {
		t.Helper()
		q, ok := b.GemQuote("Cheap Bolt", level, quality, false)
		check(name, q.Chaos, ok, want)
		if q.Variant != wantSub || !q.Corrupted {
			t.Errorf("%s = %+v, want a corrupted listing %q", name, q, wantSub)
		}
	}
	cheap("a cheaper corrupted copy of the same gem", 20, 20, 30, "20/20c")
	cheap("the cheapest corrupted copy at or above level and bracket", 20, 0, 10, "21c")
	gem("level 18 takes the nearest lower listed row", 18, 0, false, 1, "1")
	gem("16/20 is never priced as a better row", 16, 20, false, 1, "1")
	gem("an uncorrupted gem never takes a corrupted row", 21, 0, false, 3, "20")
	gem("a corrupted gem takes the nearest lower corrupted row", 22, 23, true, 60, "21/20c")
	p, ok = b.Base("Test Boots", 100, "")
	check("ilvl above 86 counts as 86", p, ok, 7)
	p, ok = b.Base("Test Boots", 85, "")
	check("an unlisted item level takes the highest listed below it", p, ok, 2)
	p, ok = b.Base("Test Boots", 0, "")
	check("no item level takes the lowest listed", p, ok, 2)
	p, ok = b.Base("Test Boots", 86, "Hunter")
	check("influenced base", p, ok, 70)
	p, ok = b.Cluster("Large Cluster Jewel", "12% increased Fire Damage", 8, 80)
	check("cluster band at or below the item level", p, ok, 9)
	p, ok = b.Currency("Orb of Alteration")
	check("currency", p, ok, 0.25)
	p, ok = b.Divine()
	check("divine", p, ok, 100)
	p, ok = b.Currency("Chaos Orb")
	check("chaos", p, ok, 1)
	for name, f := range map[string]func() (float64, bool){
		"unlisted unique": func() (float64, bool) { return b.Unique("Nothing", "", 0) },
		"unlisted gem": func() (float64, bool) {
			p, _, ok := b.Gem("Other Bolt", 20, 20, false)
			return p, ok
		},
		"base below 82":   func() (float64, bool) { return b.Base("Test Boots", 70, "") },
		"cluster too low": func() (float64, bool) { return b.Cluster("Large Cluster Jewel", "12% increased Fire Damage", 8, 60) },
	} {
		if _, ok := f(); ok {
			t.Errorf("%s answered a price", name)
		}
	}
	if league, at := b.Info(); league != "Testleague" || !at.Equal(now) {
		t.Errorf("info = %s %v", league, at)
	}
	if w := b.Warning(); w != "" {
		t.Errorf("warning without a failure: %s", w)
	}
}

func TestLiveFetchesEachCategoryOncePerTTL(t *testing.T) {
	now := time.Now()
	fake := &fakeNinja{etag: `W/"abc"`}
	b, l := liveBook(t, fake, &now)
	b.Unique("Test Plate", "", 0)
	b.Unique("Twin Hood", "", 0)
	if n := fake.requests.Load(); n != 6 {
		t.Fatalf("first lookups made %d requests, want 6 (leagues + 5 unique overviews)", n)
	}
	b2 := newBook(l)
	b2.Unique("Test Plate", "", 0)
	if n := fake.requests.Load(); n != 6 {
		t.Fatalf("a second book inside the TTL refetched: %d requests", n)
	}
	now = now.Add(DefaultTTL + time.Minute)
	b3 := newBook(l)
	if p, ok := b3.Unique("Test Plate", "", 0); !ok || p != 10 {
		t.Fatalf("after a 304 the cached overview answers: %v %v", p, ok)
	}
	if n := fake.notModded.Load(); n != 6 {
		t.Fatalf("expired entries refetched with %d conditional hits, want 6", n)
	}
}

func TestFailureIsUnknownWithOneWarning(t *testing.T) {
	now := time.Now()
	b, _ := liveBook(t, &fakeNinja{down: true}, &now)
	if _, ok := b.Unique("Test Plate", "", 0); ok {
		t.Fatal("priced while poe.ninja is down")
	}
	if _, ok := b.Divine(); ok {
		t.Fatal("divine priced while poe.ninja is down")
	}
	if w := b.Warning(); !strings.Contains(w, "connection refused") {
		t.Fatalf("warning = %q", w)
	}
	if w := b.Warning(); w != "" {
		t.Fatalf("second warning = %q", w)
	}
}

func TestStaleCacheAnswersWhenRefetchFails(t *testing.T) {
	now := time.Now()
	fake := &fakeNinja{}
	b, l := liveBook(t, fake, &now)
	b.Divine()
	now = now.Add(DefaultTTL * 2)
	fake.down = true
	if p, ok := newBook(l).Divine(); !ok || p != 100 {
		t.Fatalf("stale divine = %v %v", p, ok)
	}
}

func TestSnapshotRoundTripNeverFetches(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	fake := &fakeNinja{}
	b, _ := liveBook(t, fake, &now)
	var buf bytes.Buffer
	if err := b.WriteSnapshot(&buf); err != nil {
		t.Fatal(err)
	}
	before := fake.requests.Load()
	snap, err := FromSnapshot(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if p, _, ok := snap.Gem("Test Bolt", 20, 20, false); !ok || p != 12 {
		t.Fatalf("snapshot gem = %v %v", p, ok)
	}
	if league, at := snap.Info(); league != "Testleague" || !at.Equal(now) {
		t.Fatalf("snapshot info = %s %v", league, at)
	}
	if fake.requests.Load() != before {
		t.Fatal("a snapshot book fetched")
	}
}

func TestSnapshotMissingCategoryIsUnknown(t *testing.T) {
	b, err := FromSnapshot(strings.NewReader(`{"league":"L","fetched_at":"2026-01-01T00:00:00Z","responses":{"Currency":` + fixtures["Currency"] + `}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := b.Gem("Test Bolt", 20, 20, false); ok {
		t.Fatal("priced a gem the snapshot lacks")
	}
	if p, ok := b.Divine(); !ok || p != 100 {
		t.Fatalf("divine = %v %v", p, ok)
	}
	if w := b.Warning(); !strings.Contains(w, "SkillGem") {
		t.Fatalf("warning = %q", w)
	}
}

// TestServiceURL: requests go to the price service, prices.pob.gg unless another is named, under poe.ninja's paths,
// each service's responses cached apart.
func TestServiceURL(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct{ arg, want string }{
		{"", "https://prices.pob.gg"},
		{"http://localhost:8787/", "http://localhost:8787"},
	} {
		b, err := NewLive(dir, tc.arg)
		if err != nil {
			t.Fatal(err)
		}
		fake := &fakeNinja{base: tc.want}
		l := b.src.(*Live)
		l.Client = &http.Client{Transport: fake}
		if _, ok := b.Currency("Divine Orb"); !ok {
			t.Fatalf("%q: no Divine Orb price: %s", tc.arg, b.Warning())
		}
		if fake.requests.Load() != 2 {
			t.Errorf("%q: %d requests, want 2 (leagues, exchange)", tc.arg, fake.requests.Load())
		}
		if l.Base != tc.want {
			t.Errorf("%q: base %q, want %q", tc.arg, l.Base, tc.want)
		}
	}
	for _, bad := range []string{"https://poe.ninja?x=1", "ftp://host", "prices.pob.gg"} {
		if _, err := NewLive(dir, bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

// divineSnapshot is a book over the named responses; the stash Divine Orb line reproduces a one-data-point pay side
// (0.8 divines per chaos, 1.25c) beside a receive side of 100.17c, whose average is 50.71c.
func divineSnapshot(t *testing.T, responses map[string]string) *Book {
	t.Helper()
	parts := make([]string, 0, len(responses))
	for k, v := range responses {
		parts = append(parts, fmt.Sprintf("%q:%s", k, v))
	}
	b, err := FromSnapshot(strings.NewReader(`{"league":"L","fetched_at":"2026-01-01T00:00:00Z","responses":{` + strings.Join(parts, ",") + `}}`))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

const (
	lopsidedDivine = `{"lines":[{"currencyTypeName":"Divine Orb","chaosEquivalent":50.71,
		"pay":{"value":0.8,"data_point_count":1,"listing_count":1},"receive":{"value":100.17,"data_point_count":1,"listing_count":40}},
		{"currencyTypeName":"Orb of Fusing","chaosEquivalent":0.5,"receive":{"value":0.5,"listing_count":90}}]}`
	exchangeDivine = `{"core":{"items":[],"rates":{"divine":0.006566},"primary":"chaos","secondary":"divine"},
		"lines":[{"id":"chaos","primaryValue":1,"volumePrimaryValue":1000},{"id":"divine","primaryValue":152.3,"volumePrimaryValue":1000},
		{"id":"fusing","primaryValue":0.4,"volumePrimaryValue":300}],
		"items":[{"id":"chaos","name":"Chaos Orb"},{"id":"divine","name":"Divine Orb"},{"id":"fusing","name":"Orb of Fusing"}]}`
	divineValuedUniques = `{"lines":[
		{"name":"A","itemClass":3,"chaosValue":1523,"divineValue":10,"listingCount":9},
		{"name":"B","itemClass":3,"chaosValue":758,"divineValue":5,"listingCount":9},
		{"name":"C","itemClass":3,"chaosValue":306,"divineValue":2,"listingCount":9},
		{"name":"D","itemClass":3,"chaosValue":5,"listingCount":9}]}`
)

func TestDivineRate(t *testing.T) {
	for _, tc := range []struct {
		name      string
		responses map[string]string
		want      float64
		source    string
	}{
		{"the exchange answers", map[string]string{"Currency": lopsidedDivine, "Exchange": exchangeDivine, "UniqueArmour": divineValuedUniques}, 152.3, DivineFromExchange},
		{"without the exchange, the item lines' median", map[string]string{"Currency": lopsidedDivine, "UniqueArmour": divineValuedUniques}, 152.3, DivineFromItems},
		{"a stash line whose sides agree", map[string]string{"Currency": fixtures["Currency"]}, 100, DivineFromStash},
	} {
		b := divineSnapshot(t, tc.responses)
		if got, ok := b.Divine(); !ok || got != tc.want {
			t.Errorf("%s: divine = %v (%v), want %v", tc.name, got, ok, tc.want)
		}
		if got, src, ok := b.DivineRate(); !ok || got != tc.want || src != tc.source {
			t.Errorf("%s: rate = %v %q (%v), want %v %q", tc.name, got, src, ok, tc.want, tc.source)
		}
	}
	if got, ok := divineSnapshot(t, map[string]string{"Currency": lopsidedDivine}).Divine(); ok {
		t.Errorf("a stash line whose sides disagree 80x answered %v", got)
	}
	b := divineSnapshot(t, map[string]string{"Currency": lopsidedDivine, "Exchange": exchangeDivine})
	if got, ok := b.Currency("Orb of Fusing"); !ok || got != 0.4 {
		t.Errorf("currency from the exchange = %v (%v), want 0.4", got, ok)
	}
}
