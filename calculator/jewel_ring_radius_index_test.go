package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestJewelRingRadiusIndexMatchesTheAnnulus pins ModParser.lua:5163-5172 and the ONE reader of the value, env.go:2309-2327.
func TestJewelRingRadiusIndexMatchesTheAnnulus(t *testing.T) {
	for _, tc := range []struct {
		label     string
		wantIndex int
		wantInner float64
		wantOuter float64
	}{
		{"small", 6, 960, 1320},
		{"medium", 7, 1320, 1680},
		{"large", 8, 1680, 2040},
		{"very large", 9, 2040, 2400},
		{"massive", 10, 2400, 2880},
	} {
		t.Run(tc.label, func(t *testing.T) {
			for _, line := range []string{
				"Affects Passives in " + tc.label + " Ring",
				"Only affects Passives in " + tc.label + " Ring",
			} {
				e := ParseMod(line, false)
				if e == nil || e.Extra != "" || len(e.ModList) != 1 {
					t.Fatalf("%q: got %v", line, e)
				}
				jd, ok := e.ModList[0].Value().List().(mod.JewelData)
				if !ok || jd.Key != "radiusIndex" {
					t.Fatalf("%q: value = %v, want JewelData radiusIndex", line, e.ModList[0].Value().List())
				}
				got, ok := jd.Value.(int)
				if !ok || got != tc.wantIndex {
					t.Fatalf("%q: radiusIndex = %v, want %d", line, jd.Value, tc.wantIndex)
				}

				r := data.JewelRadius[got-1]
				if r.Inner != tc.wantInner || r.Outer != tc.wantOuter {
					t.Errorf("%q: JewelRadius[%d] = inner %v outer %v, want inner %v outer %v",
						line, got-1, r.Inner, r.Outer, tc.wantInner, tc.wantOuter)
				}
				if r.Label != "Variable" {
					t.Errorf("%q: JewelRadius[%d].Label = %q, want Variable — a ring is an annulus",
						line, got-1, r.Label)
				}
			}
		})
	}
}
