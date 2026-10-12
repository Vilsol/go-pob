package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestInstantLeechRows pins ModParser.lua:3135 and :3137-3141, two absent rows in a block whose four neighbours (:3132-3134, :3136) this tree already carries.
func TestInstantLeechRows(t *testing.T) {
	requireGameData(t, "3.29")

	if e := ParseMod("Life Leech from Hits with this Weapon is Instant", false); e == nil ||
		len(e.ModList) == 0 || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: an already-ported instant-leech row no longer parses")
	}

	t.Run("vaalPact", func(t *testing.T) {
		const line = "Life Leech from Melee Damage is Instant"
		e := ParseMod(line, false)
		if e == nil || len(e.ModList) == 0 {
			t.Fatalf("no mods at all (the line is DROPPED WHOLE): %s", line)
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Fatalf("leftover %q — an unparsed remainder discards the WHOLE line: %s", extra, line)
		}
		if len(e.ModList) != 1 || e.ModList[0].Name() != "InstantLifeLeech" {
			got := make([]string, 0, len(e.ModList))
			for _, m := range e.ModList {
				got = append(got, m.Name())
			}
			t.Fatalf("mods %v, want exactly [InstantLifeLeech]", got)
		}
		if f := e.ModList[0].Flags(); f != mod.MFlagMelee {
			t.Fatalf("flags %#x, want MFlagMelee %#x — unflagged, Vaal Pact would make NON-melee "+
				"leech instant too, which no value-based instrument here can see", f, mod.MFlagMelee)
		}
		if tags := collectCanonTags(e.ModList); len(tags) != 0 {
			t.Fatalf("tags %v, want none — ModParser.lua:3135 carries no tag, only the ModFlag", tags)
		}
	})

	t.Run("perEnemyPower", func(t *testing.T) {
		const line = "5% of Leech from Hits with this Weapon is Instant per Enemy Power"
		e := ParseMod(line, false)
		if e == nil || len(e.ModList) == 0 {
			t.Fatalf("no mods at all (the line is DROPPED WHOLE): %s", line)
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Fatalf("leftover %q — an unparsed remainder discards the WHOLE line: %s", extra, line)
		}
		got := make([]string, 0, len(e.ModList))
		for _, m := range e.ModList {
			got = append(got, m.Name())
			if f := m.Flags(); f != mod.MFlagHit {
				t.Fatalf("%s flags %#x, want MFlagHit %#x", m.Name(), f, mod.MFlagHit)
			}
		}
		for _, w := range []string{"InstantLifeLeech", "InstantManaLeech", "InstantEnergyShieldLeech"} {
			if !contains(got, w) {
				t.Fatalf("mods %v, want %q", got, w)
			}
		}
		tags := collectCanonTags(e.ModList)
		for _, w := range []string{
			"Condition/vars={Hand}Attack",
			"SkillType/skillTypes=1",
			"Multiplier/vars=EnemyPower",
		} {
			if !contains(tags, w) {
				t.Fatalf("tags %v, want %q — a tag dropped here applies the mod unconditionally", tags, w)
			}
		}
	})

	for _, tc := range []struct {
		name string
		line string
		want string
	}{
		{"nb_appliesInstantly", "Life Leech from Hits with this Weapon applies instantly", "InstantLifeLeech"},
		{"nb_hitsWeaponLife", "Life Leech from Hits with this Weapon is Instant", "InstantLifeLeech"},
		{"nb_hitsWeaponMana", "Mana Leech from Hits with this Weapon is Instant", "InstantManaLeech"},
		{"nb_gainInstantly", "Gain Life from Leech Instantly from Hits with this Weapon", "InstantLifeLeech"},
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
			if !contains(got, tc.want) {
				t.Fatalf("mods %v, want %q — a new row swallowed this neighbour: %s", got, tc.want, tc.line)
			}
			for _, m := range e.ModList {
				if m.Flags()&mod.MFlagMelee != 0 {
					t.Fatalf("%s picked up MFlagMelee — the Vaal Pact row swallowed this neighbour: %s",
						m.Name(), tc.line)
				}
			}
		})
	}
}
