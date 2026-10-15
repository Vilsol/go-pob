package calculator

import (
	"strings"
	"testing"
)

// TestMoltenStrikeTertiaryRadiusBreakdown: CalcOffence.lua:235-294: the tertiary radius breakdown's lines.
func TestMoltenStrikeTertiaryRadiusBreakdown(t *testing.T) {
	for _, c := range []struct {
		in    [6]float64
		lines string
		rows  string
		dist  float64
	}{
		{[6]float64{15, 25, 1.4, 1.1, 1.3, 1.2}, "Molten radius:| = (sqrt(R*R*a - r*r*(a-1)) - r) * s + r| = 36|^8Next AoE breakpoint: 2% increased or 1% more|^8Next Proj. Speed breakpoint: 2% increased or 1% more|^8Previous AoE breakpoint: 7% increased or 5% more|^8Previous Proj. Speed breakpoint: 6% increased or 5% more", "r=15;R=25;a=1.54;s=1.56;", 36},
		{[6]float64{15, 25, 1, 1, 1, 1}, "Molten radius:| = (sqrt(R*R*a - r*r*(a-1)) - r) * s + r| = 25|^8Next AoE breakpoint: 13% increased or 13% more|^8Next Proj. Speed breakpoint: 10% increased or 10% more|^8Previous AoE breakpoint: 1% increased or 1% more|^8Previous Proj. Speed breakpoint: 1% increased or 1% more", "r=15;R=25;a=1.00;s=1.00;", 25},
		{[6]float64{15, 26, 2.05, 0.9, 0.55, 1.15}, "Molten radius:| = (sqrt(R*R*a - r*r*(a-1)) - r) * s + r| = 26|^8Next AoE breakpoint: 26% increased or 13% more|^8Next Proj. Speed breakpoint: 5% increased or 9% more|^8Previous AoE breakpoint: 2% increased or 1% more|^8Previous Proj. Speed breakpoint: 1% increased or 2% more", "r=15;R=26;a=1.85;s=0.63;", 26},
	} {
		b := &Breakdown{Data: map[string]*Entry{}}
		setMoltenStrikeTertiaryRadiusBreakdown(b, c.in[0], c.in[1], "Molten radius:", c.in[2], c.in[3], c.in[4], c.in[5])
		e := b.Data["AreaOfEffectRadiusTertiary"]
		var rows strings.Builder
		for _, r := range e.Rows {
			rows.WriteString(r["name"] + "=" + r["value"] + ";")
		}
		if got := strings.Join(e.Lines, "|"); got != c.lines || rows.String() != c.rows || e.Label != "Inputs" || e.Radius != c.dist || len(e.Columns) != 3 {
			t.Errorf("%v:\n got %s\n     %s %s %v\nwant %s\n     %s %v", c.in, got, rows.String(), e.Label, e.Radius, c.lines, c.rows, c.dist)
		}
	}
}
