package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestFullDPSRollUp pins Calcs.lua:340-385 term by term.
func TestFullDPSRollUp(t *testing.T) {
	cases := []struct {
		name    string
		in      fullDPSBuckets
		base    float64
		want    float64
		wantDot float64
		lua     string
	}{
		{
			name: "corrupting blood is carried", base: 100,
			in:   fullDPSBuckets{CorruptingBloodDPS: 7},
			want: 107, wantDot: 7, lua: ":346-349",
		},
		{
			name: "burning ground is carried", base: 100,
			in:   fullDPSBuckets{BurningGroundDPS: 11},
			want: 111, wantDot: 11, lua: ":354-357",
		},
		{
			name: "caustic ground is carried", base: 100,
			in:   fullDPSBuckets{CausticGroundDPS: 13},
			want: 113, wantDot: 13, lua: ":363-366",
		},
		{
			name: "every bucket at once", base: 1000,
			in: fullDPSBuckets{
				BleedDPS: 1, CorruptingBloodDPS: 2, IgniteDPS: 4, BurningGroundDPS: 8,
				PoisonDPS: 16, CausticGroundDPS: 32, ImpaleDPS: 64, DecayDPS: 128, DotDPS: 256,
			},
			want: 1511, wantDot: 447, lua: ":342-378",
		},
		{
			name: "culling applies to the total INCLUDING the dot sum", base: 100,
			in:   fullDPSBuckets{DotDPS: 100, CullingMulti: 1.5},
			want: 300, wantDot: 100, lua: ":379-385",
		},
		{
			name: "a zero bucket contributes nothing", base: 100,
			in:   fullDPSBuckets{},
			want: 100, wantDot: 0, lua: ":341-380",
		},
		{
			name: "cullingMulti of exactly 0 is not applied", base: 100,
			in:   fullDPSBuckets{CullingMulti: 0},
			want: 100, wantDot: 0, lua: ":381",
		},
		{
			name: "impale is added outside TotalDotDPS", base: 0,
			in:   fullDPSBuckets{ImpaleDPS: 5, DotDPS: 5},
			want: 10, wantDot: 5, lua: ":367-370",
		},
		{
			name: "impale survives a dot sum sitting on the cap", base: 0,
			in:   fullDPSBuckets{ImpaleDPS: 1000, DotDPS: data.DotDpsCap},
			want: data.DotDpsCap + 1000, wantDot: data.DotDpsCap, lua: ":367-370,:379",
		},
		{
			name: "the dot sum is capped at DotDpsCap", base: 0,
			in:   fullDPSBuckets{DotDPS: data.DotDpsCap + 1},
			want: data.DotDpsCap, wantDot: data.DotDpsCap, lua: ":379",
		},
		{
			name: "a dot sum exactly at the cap is unchanged", base: 0,
			in:   fullDPSBuckets{DotDPS: data.DotDpsCap},
			want: data.DotDpsCap, wantDot: data.DotDpsCap, lua: ":379",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, gotDot := c.in.rollUp(c.base, &[]SkillDPS{})
			if got != c.want {
				t.Errorf("rollUp(%v) combined = %v, want %v — Calcs.lua%s", c.base, got, c.want, c.lua)
			}
			if gotDot != c.wantDot {
				t.Errorf("rollUp(%v) TotalDotDPS = %v, want %v — Calcs.lua%s (read out as "+
					"output.FullDotDPS at :430)", c.base, gotDot, c.wantDot, c.lua)
			}
		})
	}
}
