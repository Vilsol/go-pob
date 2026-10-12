package calculator

import (
	"reflect"
	"testing"

	"github.com/Vilsol/go-pob/moddb"
)

// TestBreakdownMultiChainMatchesPin pins CalcBreakdown.lua:17-42 line for line.
//
//	not chain.noTotal and (lines > 0 and base ~= nil) or (lines > 1 and base == nil)
func TestBreakdownMultiChainMatchesPin(t *testing.T) {
	item := func(v float64) BMultiChainItem { return BMultiChainItem{"%.2f", v} }

	cases := []struct {
		name  string
		chain BMultiChain
		want  []string
	}{
		{
			name:  "label emitted with no surviving multiplier",
			chain: BMultiChain{Label: "L:", Items: []BMultiChainItem{item(1), item(1)}},
			want:  []string{"L:"},
		},
		{
			name:  "base emitted with no surviving multiplier",
			chain: BMultiChain{Base: &BMultiChainItem{"%.2f base", 7}, Items: []BMultiChainItem{item(1)}},
			want:  []string{"7.00 base"},
		},
		{
			name:  "zero base suppresses the multiplier loop but not the base line",
			chain: BMultiChain{Base: &BMultiChainItem{"%.2f base", 0}, Items: []BMultiChainItem{item(2), item(3)}},
			want:  []string{"0.00 base"},
		},
		{
			name: "total emitted once, not once per surviving item",
			chain: BMultiChain{
				Base:  &BMultiChainItem{"%.2f base", 7},
				Total: "= T",
				Items: []BMultiChainItem{item(2), item(3), item(5)},
			},
			want: []string{"7.00 base", "x 2.00", "x 3.00", "x 5.00", "= T"},
		},
		{
			name:  "computed total from base and multipliers",
			chain: BMultiChain{Base: &BMultiChainItem{"%.2f base", 4}, Items: []BMultiChainItem{item(2), item(3)}},
			want:  []string{"4.00 base", "x 2.00", "x 3.00", "= 24.00"},
		},
		{
			name:  "noTotal suppresses the computed total when a base is present",
			chain: BMultiChain{Base: &BMultiChainItem{"%.2f base", 4}, NoTotal: true, Items: []BMultiChainItem{item(2)}},
			want:  []string{"4.00 base", "x 2.00"},
		},
		{
			name:  "one surviving multiplier and no base prints no computed total",
			chain: BMultiChain{Items: []BMultiChainItem{item(2)}},
			want:  []string{"x 2.00"},
		},
		{
			name:  "two surviving multipliers and no base print the computed total",
			chain: BMultiChain{Items: []BMultiChainItem{item(2), item(3)}},
			want:  []string{"x 2.00", "x 3.00", "= 6.00"},
		},
		{
			name:  "noTotal baseless with two multipliers still prints the computed total",
			chain: BMultiChain{NoTotal: true, Items: []BMultiChainItem{item(2), item(3)}},
			want:  []string{"x 2.00", "x 3.00", "= 6.00"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := NewBreakdown(moddb.NewModDB(), map[string]float64{}, nil)
			b.MultiChain("K", c.chain)
			var got []string
			if e, ok := b.GetData()["K"]; ok {
				got = e.Lines
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("lines:\n got %#v\nwant %#v", got, c.want)
			}
		})
	}
}
