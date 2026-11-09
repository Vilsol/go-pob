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
	Default buySimilarSearch `json:"default"`
	Set     buySimilarSearch `json:"set"`
	Unique  bool             `json:"unique"`
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

// The popup and its URLs are PoB-PoE2's CompareBuySimilar for every item the fixture equips, as it opens and with every
// box set (`mise run buy-similar-poe2`).
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
