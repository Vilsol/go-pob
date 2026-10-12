package calculator

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestParserGapsAgainstPoB: ModParser.lua:2420, 3166-3175.
func TestParserGapsAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	raw, err := os.ReadFile("../testdata/party-ui/parser_gaps.json")
	if err != nil {
		t.Fatalf("REFUSING: testdata/party-ui/parser_gaps.json: %v", err)
	}
	type pobMod struct {
		Name  string   `json:"name"`
		Type  string   `json:"type"`
		Value float64  `json:"value"`
		Tags  []string `json:"tags"`
	}
	var oracle []struct {
		Line  string   `json:"line"`
		Extra string   `json:"extra"`
		Mods  []pobMod `json:"mods"`
	}
	if err := json.Unmarshal(raw, &oracle); err != nil || len(oracle) < 9 {
		t.Fatalf("REFUSING: %v, %d lines", err, len(oracle))
	}
	for _, o := range oracle {
		e := ParseMod(o.Line, false)
		var got []pobMod
		for _, m := range e.ModList {
			g := pobMod{Name: m.Name(), Type: string(m.Type()), Tags: []string{}}
			switch v := m.Value(); {
			case v.Flag():
				g.Value = 1
			case v.List() != nil:
				if em, ok := v.List().(mod.EnemyModifier); ok {
					g.Value = em.Mod.Value().Float()
				}
			default:
				g.Value = v.Float()
			}
			for _, tag := range m.Tags() {
				g.Tags = append(g.Tags, string(tag.Type()))
			}
			got = append(got, g)
		}
		want := o.Mods
		for k := range want {
			if want[k].Tags == nil {
				want[k].Tags = []string{}
			}
		}
		if modExtra(e) != o.Extra || !slices.EqualFunc(got, want, func(a, b pobMod) bool {
			return a.Name == b.Name && a.Type == b.Type && a.Value == b.Value && slices.Equal(a.Tags, b.Tags)
		}) {
			t.Errorf("%q:\n ours %+v (extra %q)\n PoB  %+v", o.Line, got, modExtra(e), want)
		}
	}
}
