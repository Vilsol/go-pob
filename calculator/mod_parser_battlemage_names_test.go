package calculator

import (
	"strings"
	"testing"
)

// TestBattlemageAndMinionAttackSpeedRowsMatchThePin pins two rows this tree emitted differently from the pin, both routed as "census hygiene, expect +0".
//
//pob:block:src/Modules/ModParser.lua:3450-3454
//pob:src/Modules/ModParser.lua:5542
func TestBattlemageAndMinionAttackSpeedRowsMatchThePin(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		name string
		line string
		want []string
	}{
		{"battlemage", "Battlemage", []string{"Battlemage", "MainHandWeaponDamageAppliesToSpells"}},
		{"minionAttackSpeed", "Increases and Reductions to Minion Attack Speed also affect you",
			[]string{"MinionAttackSpeedAppliesToPlayer"}},
		{"nb_minionDamage", "Increases and Reductions to Minion Damage also affect you at 100% of their value",
			[]string{"MinionDamageAppliesToPlayer", "ImprovedMinionDamageAppliesToPlayer"}},
		{"nb_minionLife", "Increases and Reductions to Minion maximum Life also apply to you at 50% of their value",
			[]string{"MinionLifeAppliesToPlayer", "ImprovedMinionLifeAppliesToPlayer"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) == 0 {
				t.Fatalf("no mods at all: %s", tc.line)
			}
			if extra := strings.TrimSpace(modExtra(e)); extra != "" {
				t.Fatalf("leftover %q: %s", extra, tc.line)
			}
			got := make([]string, 0, len(e.ModList))
			for _, m := range e.ModList {
				got = append(got, m.Name())
			}
			if len(got) != len(tc.want) {
				t.Fatalf("emitted %v (%d mods), want exactly %v (%d) — an EXTRA mod here is as much "+
					"a divergence as a missing one, and it is the direction no missing-thing "+
					"instrument looks: %s", got, len(got), tc.want, len(tc.want), tc.line)
			}
			for _, w := range tc.want {
				if !contains(got, w) {
					t.Fatalf("emitted %v, want %q: %s", got, w, tc.line)
				}
			}
		})
	}
}
