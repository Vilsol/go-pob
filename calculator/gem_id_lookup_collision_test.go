package calculator

import (
	"sort"
	"strings"
	"testing"

	raw2 "github.com/Vilsol/go-pob/data/raw"
)

// TestGemIDLookupMatchesUpstreamCollisionRule re-implements ModParser.lua:2032-2045 literally and requires our gemIdLookup port to agree on EVERY name.
//
//pob:block:src/Modules/ModParser.lua:2032-2045
func TestGemIDLookupMatchesUpstreamCollisionRule(t *testing.T) {
	requireGameData(t, "3.29")

	ids := make([]string, 0, len(raw2.PoBSkills))
	for id := range raw2.PoBSkills {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	upstream := map[string]string{"power charge on critical strike": "SupportPowerChargeOnCritical"}
	for _, id := range ids {
		s := raw2.PoBSkills[id]
		if s.Hidden && !s.FromItem && !s.FromTree {
			continue
		}
		upstream[strings.ToLower(s.Name)] = id
	}
	if len(upstream) < 500 {
		t.Fatalf("REFUSING: rebuilt only %d names from raw.PoBSkills; the source table is not "+
			"loaded and a comparison against it would be vacuous", len(upstream))
	}

	ours := gemLookup().skillIDForName
	var bad []string
	for name, want := range upstream {
		if got := ours[name]; got != want {
			bad = append(bad, name+": ours="+got+" pin="+want)
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 {
		t.Errorf("%d name(s) resolve to a different skill id than ModParser.lua:2040-2044 gives:\n  %s",
			len(bad), strings.Join(bad, "\n  "))
	}

	for _, tc := range []struct{ name, want string }{
		{"vulnerability", "Vulnerability"},
		{"barrage", "Barrage"},
		{"bane", "SupportDarkRitualAltX"},
	} {
		if got := ours[tc.name]; got != tc.want {
			t.Errorf("gemIdLookup[%q] = %q, want %q", tc.name, got, tc.want)
		}
	}
}
