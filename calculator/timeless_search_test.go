package calculator

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

type timelessSearchCase struct {
	Name      string
	Build     string
	Type      int
	Conqueror int
	Socket    int64
	List      string
	Fallback  string
	Filter    bool
	Distance  int
	MinTotal  *float64
	Protected []string
	Sockets   []struct {
		ID       int64
		Label    string
		Keystone string
	}
	Mods []struct {
		ID    string
		Label string
	}
	Desired []struct {
		ID   string
		Name string
		Idx  int
	}
	Results [][3]float64
	Rows    []struct {
		Seed   int
		Socket int64
		Label  string
		Nodes  map[string][]string
	}
}

func loadTimelessSearchCases(t *testing.T) []timelessSearchCase {
	t.Helper()
	f, err := os.Open("../testdata/tree-ui/timeless_search.json.gz")
	if err != nil {
		t.Fatalf("REFUSING: testdata/tree-ui/timeless_search.json.gz: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var cases []timelessSearchCase
	if err := json.NewDecoder(zr).Decode(&cases); err != nil || len(cases) < 6 {
		t.Fatalf("REFUSING: the fixture does not decode: %v", err)
	}
	return cases
}

func (c timelessSearchCase) query() TimelessSearchQuery {
	q := TimelessSearchQuery{
		JewelType: c.Type, ConquerorType: c.Conqueror, SocketID: c.Socket, SocketFilter: c.Filter,
		SocketFilterDistance: c.Distance, ProtectedNodes: c.Protected, SearchList: c.List, SearchListFallback: c.Fallback,
	}
	if c.MinTotal != nil {
		q.TotalMinimumWeight, q.HasTotalMinimumWeight = *c.MinTotal, true
	}
	return q
}

// TestTimelessSearchAgainstPoB grades Find a Timeless Jewel's socket list.
func TestTimelessSearchAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	graded := 0
	for _, c := range loadTimelessSearchCases(t) {
		t.Run(c.Name, func(t *testing.T) {
			build := loadCorpusBuild(t, c.Build)
			sockets := TimelessSockets(build)
			if len(sockets) != len(c.Sockets) {
				t.Errorf("sockets: ours %d, the pin's %d", len(sockets), len(c.Sockets))
			}
			for i := range min(len(sockets), len(c.Sockets)) {
				w := c.Sockets[i]
				if g := sockets[i]; g.ID != w.ID || g.Label != w.Label || g.Keystone != w.Keystone {
					t.Errorf("socket %d: ours %+v, the pin's %+v", i, g, w)
				}
			}
			mods := TimelessMods(c.Type)
			if len(mods) != len(c.Mods) {
				t.Errorf("mods: ours %d, the pin's %d", len(mods), len(c.Mods))
			}
			for i := range min(len(mods), len(c.Mods)) {
				if mods[i].ID != c.Mods[i].ID || mods[i].Label != c.Mods[i].Label {
					t.Errorf("mod %d: ours %q %q, the pin's %q %q", i, mods[i].ID, mods[i].Label, c.Mods[i].ID, c.Mods[i].Label)
				}
			}
			if len(c.Protected) > 0 {
				prot := GetMiscCalculator(build).TimelessProtectableNodes(c.Socket)
				var names []string
				for _, p := range prot {
					names = append(names, p.Label)
				}
				for _, p := range c.Protected {
					if !slices.Contains(names, p) {
						t.Errorf("protectable: the pin's %q is not in ours %v", p, names)
					}
				}
			}

			got := GetMiscCalculator(build).TimelessSearch(c.query())
			var desired []string
			for _, d := range got.Desired {
				desired = append(desired, fmt.Sprintf("%s|%s|%d", d.ID, d.DisplayName, d.DesiredIdx))
			}
			var wantDesired []string
			for _, d := range c.Desired {
				wantDesired = append(wantDesired, fmt.Sprintf("%s|%s|%d", d.ID, d.Name, d.Idx))
			}
			if !slices.Equal(desired, wantDesired) {
				t.Errorf("desired: ours %v, the pin's %v", desired, wantDesired)
			}

			type key struct {
				seed   int
				socket int64
			}
			ours := map[key]TimelessSearchResult{}
			for _, r := range got.Results {
				s := r.SocketID
				if c.Socket != -1 {
					s = c.Socket
				}
				ours[key{r.Seed, s}] = r
			}
			if len(got.Results) != len(c.Results) {
				t.Errorf("results: ours %d, the pin's %d", len(got.Results), len(c.Results))
			}
			bad := 0
			for _, w := range c.Results {
				k := key{int(w[0]), int64(w[2])}
				g, ok := ours[k]
				switch {
				case !ok:
					bad++
					if bad <= 5 {
						t.Errorf("seed %d socket %d: the pin's total %v, ours has no row", k.seed, k.socket, w[1])
					}
				case math.Abs(g.Total-w[1]) > 1e-9*math.Max(1, math.Abs(w[1])):
					bad++
					if bad <= 5 {
						t.Errorf("seed %d socket %d: ours %v, the pin's %v", k.seed, k.socket, g.Total, w[1])
					}
				}
				delete(ours, k)
			}
			for k, g := range ours {
				bad++
				if bad <= 10 {
					t.Errorf("seed %d socket %d: ours %v, the pin has no row", k.seed, k.socket, g.Total)
				}
			}
			for i := 1; i < len(got.Results); i++ {
				if got.Results[i].Total > got.Results[i-1].Total {
					t.Fatalf("rows are not sorted by total at %d", i)
				}
			}
			byKey := map[key]TimelessSearchResult{}
			for _, r := range got.Results {
				s := r.SocketID
				if c.Socket != -1 {
					s = c.Socket
				}
				byKey[key{r.Seed, s}] = r
			}
			for _, w := range c.Rows {
				g, ok := byKey[key{w.Seed, w.Socket}]
				if !ok {
					continue
				}
				graded++
				if g.Label != w.Label {
					t.Errorf("seed %d label: ours %q, the pin's %q", w.Seed, g.Label, w.Label)
				}
				nodes := map[string][]string{}
				for _, n := range g.Nodes {
					names := slices.Clone(n.TargetNodeNames)
					slices.Sort(names)
					nodes[n.ID] = names
				}
				for id, names := range w.Nodes {
					if !slices.Equal(nodes[id], names) {
						t.Errorf("seed %d %s: ours %v, the pin's %v", w.Seed, id, nodes[id], names)
					}
				}
				if len(nodes) != len(w.Nodes) {
					t.Errorf("seed %d: ours hits %d desired nodes, the pin %d", w.Seed, len(nodes), len(w.Nodes))
				}
			}
		})
	}
	if graded < 500 {
		t.Fatalf("graded only %d sampled rows", graded)
	}
}

// TestTimelessTradeAndItemAgainstPoB grades Open Trade URL (clicked twice) over the pin's own row order, and the item a double click on the first row adds.
func TestTimelessTradeAndItemAgainstPoB(t *testing.T) {
	cases := loadTimelessSearchCasesTrade(t)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			if len(c.Results) == 0 {
				t.Fatalf("REFUSING: no rows")
			}
			seeds := make([]int, len(c.Results))
			for i, r := range c.Results {
				seeds[i] = int(r[0])
			}
			q := TimelessTradeQuery{
				JewelType: c.Type, ConquerorType: c.Conqueror, DevotionVariant1: c.Devotion[0], DevotionVariant2: c.Devotion[1],
				Seeds: seeds, SelIndex: 1, SearchMore: c.SearchMore, TradeTypeIndex: c.TradeType,
				Realm: TimelessRealms[c.Realm-1], League: "Settlers",
			}
			if c.Highlight[0] > 0 {
				q.SelIndex, q.HighlightIndex = c.Highlight[0], c.Highlight[1]
			}
			if len(c.URLs) != 2 {
				t.Fatalf("REFUSING: the pin opened %d urls", len(c.URLs))
			}
			for i, want := range c.URLs {
				got := TimelessTradeURL(q)
				gp, gq, _ := strings.Cut(got.URL, "?q=")
				wp, wq, _ := strings.Cut(want, "?q=")
				if gp != wp {
					t.Errorf("click %d path: ours %q, the pin's %q", i+1, gp, wp)
				}
				if dg, dw := decodeTradeQuery(t, gq), decodeTradeQuery(t, wq); !reflect.DeepEqual(dg, dw) {
					t.Errorf("click %d query:\nours    %v\nthe pin %v", i+1, dg, dw)
				}
				q.SelIndex, q.HighlightIndex = got.SelIndex, got.HighlightIndex
				q.LastSearchStart, q.LastSearchCount, q.LastSearchPresent = got.LastSearchStart, got.LastSearchCount, got.LastSearchPresent
			}
			first := c.Results[0]
			socketLabel := ""
			for _, s := range c.Sockets {
				if s.ID == c.Socket {
					socketLabel = s.Keystone
				}
				if c.Socket == -1 && s.ID == int64(first[2]) {
					socketLabel = s.Label
				}
			}
			if got := TimelessResultItem(c.Type, c.Conqueror, c.Devotion[0], c.Devotion[1], int(first[0]), first[1], socketLabel); got != c.Item {
				t.Errorf("item:\nours    %q\nthe pin %q", got, c.Item)
			}
		})
	}
}

func decodeTradeQuery(t *testing.T, enc string) any {
	t.Helper()
	s, err := url.QueryUnescape(enc)
	if err != nil {
		t.Fatal(err)
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("query does not decode: %v", err)
	}
	return v
}

type timelessTradeCase struct {
	timelessSearchCase
	URLs       []string
	Item       string
	Devotion   [2]int
	Realm      int
	TradeType  int
	SearchMore bool
	Highlight  [2]int
}

func loadTimelessSearchCasesTrade(t *testing.T) []timelessTradeCase {
	t.Helper()
	f, err := os.Open("../testdata/tree-ui/timeless_search.json.gz")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var cases []timelessTradeCase
	if err := json.NewDecoder(zr).Decode(&cases); err != nil || len(cases) < 6 {
		t.Fatalf("REFUSING: the fixture does not decode: %v", err)
	}
	return cases
}

// TestTimelessFallbackWeightsAgainstPoB grades Generate's fallback list.
func TestTimelessFallbackWeightsAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	raw, err := os.ReadFile("../testdata/tree-ui/timeless_fallback.json")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var cases []struct {
		Build string
		Type  int
		Stat  string
		List  string
		Modes []struct{ Label, Stat string }
	}
	if err := json.Unmarshal(raw, &cases); err != nil || len(cases) < 4 {
		t.Fatalf("REFUSING: the fixture does not decode: %v", err)
	}
	modes := TimelessFallbackModes()
	if len(modes) != len(cases[0].Modes) {
		t.Errorf("modes: ours %d, the pin's %d", len(modes), len(cases[0].Modes))
	}
	for i := range min(len(modes), len(cases[0].Modes)) {
		if "Sort by "+modes[i].Label != cases[0].Modes[i].Label || modes[i].Stat != cases[0].Modes[i].Stat {
			t.Errorf("mode %d: ours %q %q, the pin's %+v", i, modes[i].Label, modes[i].Stat, cases[0].Modes[i])
		}
	}
	for _, c := range cases {
		got, ok := GetMiscCalculator(loadCorpusBuild(t, c.Build)).TimelessFallbackWeights(c.Type, c.Stat)
		if !ok {
			t.Errorf("%s type %d %s: refused", c.Build, c.Type, c.Stat)
			continue
		}
		ours := strings.Split(got, "\n")
		var kept []string
		for _, row := range strings.Split(c.List, "\n") {
			id, _, _ := strings.Cut(row, ",")
			if !slices.ContainsFunc(ours, func(r string) bool { return strings.HasPrefix(r, id+",") }) && timelessNodeUnparsed(id) {
				t.Logf("%s: %s left out: a line Go's parser cannot read", c.Build, id)
				continue
			}
			kept = append(kept, row)
		}
		want := strings.Join(kept, "\n")
		if got != want {
			t.Errorf("%s type %d %s:\nours    %q\nthe pin %q", c.Build, c.Type, c.Stat, got, want)
		}
	}
}

func timelessNodeUnparsed(id string) bool {
	node, ok := data.LegionNodes[id]
	if !ok {
		node, ok = data.LegionAdditions[id]
	}
	if !ok {
		return false
	}
	for _, l := range node.Stats {
		if e := ParseMod(l, false); e == nil || strings.TrimSpace(e.Extra) != "" {
			return true
		}
	}
	return false
}
