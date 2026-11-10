package exposition

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/data/poe2"
)

type buySimilarSearch struct {
	Base  string         `json:"base"`
	Query map[string]any `json:"query"`
}

// buySimilarOracle is oracle/poe2_buy_similar.lua's record of one slot's popup.
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

// decodeBuySimilar splits a trade URL into its path and its gzipped, Base64URL encoded query.
func decodeBuySimilar(t *testing.T, url string) buySimilarSearch {
	t.Helper()
	i := strings.LastIndex(url, "/")
	raw, err := base64.StdEncoding.DecodeString(strings.NewReplacer("-", "+", "_", "/").Replace(url[i+1:]))
	if err != nil {
		t.Fatal(err)
	}
	r, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	text, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	out := buySimilarSearch{Base: url[:i]}
	if err := json.Unmarshal(text, &out.Query); err != nil {
		t.Fatal(err)
	}
	return out
}

// The popup and its URLs are PoB-PoE2's CompareBuySimilar for every item the fixture equips, as it opens, with every
// box set, and with every searchable mod row checked (`mise run buy-similar-poe2`).
func TestBuySimilarMatchesPoB(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_buy_similar.json"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle map[string]buySimilarOracle
	if err := json.Unmarshal(raw, &oracle); err != nil {
		t.Fatal(err)
	}
	xml, err := os.ReadFile(filepath.Join("testdata", "buy_similar.xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	categories := map[string]bool{}
	for slot, want := range oracle {
		t.Run(slot, func(t *testing.T) {
			view, err := calc.BuySimilar(slot)
			if err != nil {
				t.Fatal(err)
			}
			if view.Unique != want.Unique || view.Name != want.Name || view.BaseName != want.BaseName || view.Category != want.Category {
				t.Errorf("popup %+v, PoB-PoE2 unique %v name %q base %q category %q", view, want.Unique, want.Name, want.BaseName, want.Category)
			}
			if len(view.Defences) != len(want.Defences) {
				t.Fatalf("defences %+v, PoB-PoE2 %+v", view.Defences, want.Defences)
			}
			set := calculator2.BuySimilarOptions{League: "Rise of the Abyssal", Listed: 3, BaseType: true, IlvlMin: new(70), IlvlMax: new(85)}
			for i, d := range view.Defences {
				if d.Label != want.Defences[i].Label || strconv.Itoa(int(d.Value)) != want.Defences[i].Min {
					t.Errorf("defence %d %s %v, PoB-PoE2 %s %s", i, d.Label, d.Value, want.Defences[i].Label, want.Defences[i].Min)
				}
				low := int(d.Value)
				f := calculator2.BuySimilarDefenceFilter{Checked: true, Min: &low}
				if i == 0 {
					f.Max = new(5000)
				}
				set.Defences = append(set.Defences, f)
			}
			for _, c := range []struct {
				name string
				o    calculator2.BuySimilarOptions
				want buySimilarSearch
			}{{"default", calculator2.BuySimilarOptions{Listed: 1}, want.Default}, {"set", set, want.Set}} {
				url, err := calc.BuySimilarURL(slot, c.o)
				if err != nil {
					t.Fatal(err)
				}
				got := decodeBuySimilar(t, url)
				// encoding/json reads PoB-PoE2's numbers as float64 too
				if !reflect.DeepEqual(got, c.want) {
					t.Errorf("%s:\n PoB-PoE2 %+v\n ours     %+v", c.name, c.want, got)
				}
			}
			t.Run("mods", func(t *testing.T) {
				if _, err := poe2.TradeStats(); err != nil {
					t.Skipf("SKIP: %v", err)
				}
				if len(view.Mods) != len(want.Mods) {
					t.Fatalf("%d mod rows %+v, PoB-PoE2 %d %+v", len(view.Mods), view.Mods, len(want.Mods), want.Mods)
				}
				o := calculator2.BuySimilarOptions{Listed: 1}
				for i, m := range view.Mods {
					w := want.Mods[i]
					if strings.Join(m.Lines, "\n") != w.Label || m.Type != w.Type || m.CanSearch != w.CanSearch || m.HasBoxes != (w.Min != nil) || (w.Min != nil && m.Min != *w.Min) {
						t.Errorf("mod row %d %+v, PoB-PoE2 %+v", i+1, m, w)
					}
					f := calculator2.BuySimilarModFilter{Checked: w.CanSearch}
					if w.Min != nil {
						if v, err := strconv.ParseFloat(*w.Min, 64); err == nil {
							f.Min = &v
						}
						if w.CanSearch && i%2 == 0 {
							f.Max = new(999.0)
						}
					}
					o.Mods = append(o.Mods, f)
				}
				url, err := calc.BuySimilarURL(slot, o)
				if err != nil {
					t.Fatal(err)
				}
				if got := decodeBuySimilar(t, url); !reflect.DeepEqual(got, want.ModSearch) {
					t.Errorf("mod rows checked:\n PoB-PoE2 %+v\n ours     %+v", want.ModSearch, got)
				}
			})
			if f, ok := want.Default.Query["filters"].(map[string]any); ok {
				if tf, ok := f["type_filters"].(map[string]any); ok {
					categories[tf["filters"].(map[string]any)["category"].(map[string]any)["option"].(string)] = true
				}
			}
		})
	}
	for _, c := range []string{"armour.chest", "armour.focus", "weapon.warstaff", "weapon.wand", "accessory.ring"} {
		if !categories[c] {
			t.Errorf("no fixture item searches category %s", c)
		}
	}
}
