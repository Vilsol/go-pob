package calculator

import (
	"reflect"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/moddb"
)

// TestBreakdownEffMultMatchesPin pins CalcBreakdown.lua:111-157.
func TestBreakdownEffMultMatchesPin(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	cases := []struct {
		name string
		call func(b *Breakdown)
		want []string
	}{
		{
			name: "resist prints truncated as an integer",
			call: func(b *Breakdown) {
				b.EffMult("K", data.DamageTypeCold, 75.5, 0, 0, 0.245, 1, "", true, 0, nil)
			},
			want: []string{"Enemy resistance: 75%", "Effective DPS modifier:", "x 0.24 ^8(resistance)", "= 0.245"},
		},
		{
			name: "source resistance names its element",
			call: func(b *Breakdown) {
				b.EffMult("K", data.DamageTypeCold, 30.9, 0, 0, 0.691, 1, data.DamageTypeFire, true, 0, nil)
			},
			want: []string{"Enemy resistance: 30% ^8(Fire)", "Effective DPS modifier:", "x 0.69 ^8(resistance)", "= 0.691"},
		},
		{
			name: "inversion prints the weighted average and suppresses the penetration block",
			call: func(b *Breakdown) {
				b.EffMult("K", data.DamageTypeFire, 60, 10, 0, 1, 1, "", true, 0.4, f(22))
			},
			want: []string{
				"Enemy resistance: 60%",
				"Effective resistance:",
				"50% ^8(non-inverted hit after penetration)",
				"-70% ^8(inverted hit after penetration)",
				"= 22% ^8(weighted average from 40% inversion chance)",
				"Effective DPS modifier:",
				"x 0.78 ^8(resistance)",
				"= 1.000",
			},
		},
		{
			name: "penetration block still runs when nothing inverts",
			call: func(b *Breakdown) {
				b.EffMult("K", data.DamageTypeFire, 60, 10, 0, 0.5, 1, "", true, 0, f(50))
			},
			want: []string{
				"Enemy resistance: 60%",
				"Effective resistance:",
				"60% ^8(resistance)",
				"- 10% ^8(penetration)",
				"= 50%",
				"Effective DPS modifier:",
				"x 0.50 ^8(resistance)",
				"= 0.500",
			},
		},
		{
			name: "effectiveResist of exactly 0 is honoured",
			call: func(b *Breakdown) {
				b.EffMult("K", data.DamageTypeFire, 60, 10, 0, 1, 1, "", true, 0, f(0))
			},
			want: []string{
				"Enemy resistance: 60%",
				"Effective resistance:",
				"60% ^8(resistance)",
				"- 10% ^8(penetration)",
				"= 50%",
				"Effective DPS modifier:",
				"= 1.000",
			},
		},
		{
			name: "resistance ignored takes the non-multiChain arm",
			call: func(b *Breakdown) {
				b.EffMult("K", data.DamageTypePhysical, 0, 0, 50, 1.5, 1, "", false, 0, nil)
			},
			want: []string{
				"Effective resistance:",
				"0% ^8(resistance)",
				"x 0% ^8(resistance ignored)",
				"= 0%",
				"Effective DPS modifier:",
				"= 1.500 ^8(increased/reduced damage taken)",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := NewBreakdown(moddb.NewModDB(), map[string]float64{}, nil)
			c.call(b)
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
