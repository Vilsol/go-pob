package calculator

import (
	"slices"
	"strings"
	"testing"
)

func assertNamedRows(t *testing.T, file string, keep func(sourceName string) bool) {
	t.Helper()
	env, cases := oracleModSectionTables(t, file)
	checked := 0
	for _, c := range cases {
		var got []string
		for _, r := range env.CalcsModSection(c.data(), false) {
			got = append(got, r.Value+" | "+r.Name+" | "+r.Source+" | "+r.SourceName)
		}
		for _, r := range c.Rows {
			if !keep(r.SourceName) {
				continue
			}
			checked++
			if want := r.Value + " | " + r.Name + " | " + r.Source + " | " + r.SourceName; !slices.Contains(got, want) {
				t.Errorf("%v: no row %q", c.ModName, want)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no row to check")
	}
}

// TestCalcsModSectionTattooSourceName: A tattoo's mods are sourced "Tree:<tattoo id>".
func TestCalcsModSectionTattooSourceName(t *testing.T) {
	assertNamedRows(t, "0001-necromancer-unknown.xml", func(name string) bool { return strings.HasPrefix(name, "Tattoo of the ") })
}

// TestCalcsModSectionRunegraftSourceName: A runegraft on an allocated mastery with a selection is reprocessed under the mastery's own id.
func TestCalcsModSectionRunegraftSourceName(t *testing.T) {
	assertNamedRows(t, "0031-guardian-raise-spectre.xml", func(name string) bool { return strings.HasPrefix(name, "Runegraft of ") })
}

// TestCalcsModSectionGloriousVanitySmallNodeSourceName: Glorious Vanity re-parses each small node's stat as a replacement under the node's own id.
func TestCalcsModSectionGloriousVanitySmallNodeSourceName(t *testing.T) {
	assertNamedRows(t, "0102-gladiator-raise-spectre.xml", func(name string) bool { return name == "Avoid Elemental Ailments" || name == "Chaos Damage" })
}
