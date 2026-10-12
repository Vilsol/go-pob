package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestWardBreakParserRows pins ModParser.lua:2393 and :2394, the two ward rows this tree had a CONSUMER for and no PRODUCER.
func TestWardBreakParserRows(t *testing.T) {
	requireGameData(t, "3.29")

	cases := []struct {
		line string
		name string
		want float64
	}{
		{
			line: "Ward has a 25% chance to not Break",
			name: "WardAvoidBreakChance",
			want: 25,
		},
		{
			line: "Damage taken Bypasses Unbroken Ward if the Hit deals less Damage than 10% of Ward",
			name: "WardBypassBelowPercent",
			want: 10,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := ParseMod(c.line, false)
			if e == nil {
				t.Fatalf("REFUSING: %q produced no parse at all", c.line)
			}
			if rem := strings.TrimSpace(modExtra(e)); rem != "" {
				t.Fatalf("unparsed remainder %q — the whole line is discarded", rem)
			}
			if len(e.ModList) != 1 {
				t.Fatalf("produced %d mods, want exactly 1 (the pin's row emits one)", len(e.ModList))
			}
			m := e.ModList[0]
			if m.Name() != c.name {
				t.Fatalf("produced %q, want %q", m.Name(), c.name)
			}
			if m.Type() != mod.TypeBase {
				t.Fatalf("%s is type %q, want BASE", c.name, m.Type())
			}
			if got := m.Value().Float(); got != c.want {
				t.Fatalf("%s = %v, want %v", c.name, got, c.want)
			}
		})
	}
}
