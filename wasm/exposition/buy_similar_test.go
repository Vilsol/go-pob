package exposition

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

type buySimilarSearch struct {
	Base  string         `json:"base"`
	Query map[string]any `json:"query"`
}

// buySimilarOracle is oracle/buy_similar.lua's record of one slot's popup.
type buySimilarOracle struct {
	Name     string `json:"name"`
	BaseName string `json:"baseName"`
	Category string `json:"category"`
	Defences []struct {
		Label string `json:"label"`
		Min   string `json:"min"`
	} `json:"defences"`
	Default   buySimilarSearch `json:"default"`
	Set       buySimilarSearch `json:"set"`
	ModSearch buySimilarSearch `json:"modSearch"`
	Mods      []struct {
		Min       *string `json:"min"`
		Label     string  `json:"label"`
		Type      string  `json:"type"`
		CanSearch bool    `json:"canSearch"`
	} `json:"mods"`
	Unique bool `json:"unique"`
}

// decodeBuySimilar splits a trade URL into its path and its URL encoded query.
func decodeBuySimilar(t *testing.T, u string) buySimilarSearch {
	t.Helper()
	base, q, ok := strings.Cut(u, "?q=")
	if !ok {
		t.Fatalf("no query in %s", u)
	}
	text, err := url.PathUnescape(q)
	if err != nil {
		t.Fatal(err)
	}
	out := buySimilarSearch{Base: base}
	if err := json.Unmarshal([]byte(text), &out.Query); err != nil {
		t.Fatal(err)
	}
	return out
}

// buySimilarTexts is the item text in each slot the build equips, its tree's jewel sockets included.
func buySimilarTexts(build *pob.PathOfBuilding) map[string]string {
	texts := map[int]string{}
	for _, it := range build.Items.Items {
		texts[it.ID] = it.Text
	}
	out := map[string]string{}
	for _, e := range EquippedItems(build) {
		out[e.SlotName] = e.Text
	}
	if spec := build.Tree.ActiveSpec - 1; spec >= 0 && spec < len(build.Tree.Specs) {
		for _, s := range build.Tree.Specs[spec].Sockets {
			if text, ok := texts[s.ItemID]; ok {
				out[fmt.Sprintf("Jewel %d", s.NodeID)] = text
			}
		}
	}
	return out
}

// The popup and its URLs are PoB's CompareBuySimilar for every item two fixture builds equip, as it opens, with every
// box set, and with every searchable mod row checked (`mise run buy-similar`).
func TestBuySimilarMatchesPoB(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, assetcache.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "buy_similar.json"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle map[string]map[string]buySimilarOracle
	if err := json.Unmarshal(b, &oracle); err != nil {
		t.Fatal(err)
	}
	fixtures := map[string]string{"7.xml": "many-builds", "Fireball-full.xml": "builds"}
	searched := 0
	for file, slots := range oracle {
		xml, err := os.ReadFile(filepath.Join("..", "..", "testdata", fixtures[file], file))
		if err != nil {
			t.Fatal(err)
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			t.Fatal(err)
		}
		texts := buySimilarTexts(build)
		for slot, want := range slots {
			t.Run(file+"/"+slot, func(t *testing.T) {
				text, ok := texts[slot]
				if !ok {
					t.Fatalf("the fixture equips nothing in %s", slot)
				}
				view, err := BuySimilar(build, text, slot)
				if err != nil {
					t.Fatal(err)
				}
				if view.Unique != want.Unique || view.Name != want.Name || view.BaseName != want.BaseName || view.Category != want.Category {
					t.Errorf("popup %+v, PoB unique %v name %q base %q category %q", view, want.Unique, want.Name, want.BaseName, want.Category)
				}
				if len(view.Defences) != len(want.Defences) {
					t.Fatalf("defences %+v, PoB %+v", view.Defences, want.Defences)
				}
				set := calculator.BuySimilarOptions{Realm: "xbox", League: "Mirage Ruthless", Listed: 3, BaseType: true, IlvlMin: new(70), IlvlMax: new(85)}
				for i, d := range view.Defences {
					low := int(math.Floor(d.Value))
					if d.Label != want.Defences[i].Label || strconv.Itoa(low) != want.Defences[i].Min {
						t.Errorf("defence %d %s %v, PoB %s %s", i, d.Label, d.Value, want.Defences[i].Label, want.Defences[i].Min)
					}
					f := calculator.BuySimilarDefenceFilter{Checked: true, Min: &low}
					if i == 0 {
						f.Max = new(5000)
					}
					set.Defences = append(set.Defences, f)
				}
				if len(view.Mods) != len(want.Mods) {
					t.Fatalf("%d mod rows %+v, PoB %d %+v", len(view.Mods), view.Mods, len(want.Mods), want.Mods)
				}
				mods := calculator.BuySimilarOptions{Listed: 1}
				for i, m := range view.Mods {
					w := want.Mods[i]
					if strings.Join(m.Lines, "\n") != w.Label || m.Type != w.Type || m.CanSearch != w.CanSearch || m.HasBoxes != (w.Min != nil) || (w.Min != nil && m.Min != *w.Min) {
						t.Errorf("mod row %d %+v, PoB %+v", i+1, m, w)
					}
					f := calculator.BuySimilarModFilter{Checked: w.CanSearch}
					if w.Min != nil {
						if v, err := strconv.ParseFloat(*w.Min, 64); err == nil {
							f.Min = &v
						}
						if w.CanSearch && i%2 == 0 {
							f.Max = new(999.0)
						}
					}
					mods.Mods = append(mods.Mods, f)
				}
				for _, c := range []struct {
					name string
					o    calculator.BuySimilarOptions
					want buySimilarSearch
				}{{"default", calculator.BuySimilarOptions{Listed: 1}, want.Default}, {"set", set, want.Set}, {"mods", mods, want.ModSearch}} {
					u, err := BuySimilarURL(build, text, slot, c.o)
					if err != nil {
						t.Fatal(err)
					}
					// encoding/json reads PoB's numbers as float64 too
					if got := decodeBuySimilar(t, u); !reflect.DeepEqual(got, c.want) {
						t.Errorf("%s:\n PoB  %+v\n ours %+v", c.name, c.want, got)
					}
				}
				searched++
			})
		}
	}
	if searched == 0 {
		t.Error("the oracle has no items")
	}
}
