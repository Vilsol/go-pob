package calculator

import (
	"sort"
	"testing"

	tjdata "github.com/Vilsol/timeless-jewels/data"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestLegionLibraryIDsAllResolve: The timeless-jewels library speaks the GAME's row ids.
func TestLegionLibraryIDsAllResolve(t *testing.T) {
	if err := raw.EnsureTimelessSources(); err != nil {
		t.Fatalf("EnsureTimelessSources: %v", err)
	}
	if len(tjdata.AlternatePassiveAdditions) == 0 || len(tjdata.AlternatePassiveSkills) == 0 {
		t.Fatal("the timeless-jewels library exposed no rows; this test would be vacuous")
	}

	var unresolved []string
	for _, a := range tjdata.AlternatePassiveAdditions {
		if a == nil {
			continue
		}
		if _, ok := data.LegionAdditions[legionRowID(a.ID)]; !ok {
			unresolved = append(unresolved, "addition "+a.ID)
		}
	}
	for _, s := range tjdata.AlternatePassiveSkills {
		if s == nil {
			continue
		}
		if _, ok := data.LegionNodes[legionNodeRowID(s)]; !ok {
			unresolved = append(unresolved, "node "+s.ID)
		}
	}

	sort.Strings(unresolved)
	if len(unresolved) > 0 {
		t.Fatalf("%d timeless-jewel row ids resolve to no LegionAdditions/LegionNodes row:\n  %v",
			len(unresolved), unresolved)
	}
}

// TestLegionDatErrorAliasesAreLive: Every alias must be live in BOTH directions.
func TestLegionDatErrorAliasesAreLive(t *testing.T) {
	if err := raw.EnsureTimelessSources(); err != nil {
		t.Fatalf("EnsureTimelessSources: %v", err)
	}
	if len(legionDatErrorRenames) == 0 {
		t.Fatal("no aliases declared; this test would be vacuous")
	}
	for from, to := range legionDatErrorRenames {
		_, inAdditions := data.LegionAdditions[from]
		_, inNodes := data.LegionNodes[from]
		if inAdditions || inNodes {
			t.Errorf("alias %q -> %q is dead: the game-side id already resolves", from, to)
		}
		_, toAdditions := data.LegionAdditions[to]
		_, toNodes := data.LegionNodes[to]
		if !toAdditions && !toNodes {
			t.Errorf("alias %q -> %q renames to an id no table carries", from, to)
		}
	}
}

// TestLegionNodeRowIDAppliesNameMatchedSwaps covers the two datErrors entries that match on NAME.
func TestLegionNodeRowIDAppliesNameMatchedSwaps(t *testing.T) {
	if err := raw.EnsureTimelessSources(); err != nil {
		t.Fatalf("EnsureTimelessSources: %v", err)
	}
	want := map[string]string{
		"Powerful Faith": "templar_notable_minimum_power_charge",
		"Frenzied Faith": "templar_notable_minimum_frenzy_charge",
	}
	seen := 0
	for _, s := range tjdata.AlternatePassiveSkills {
		if s == nil {
			continue
		}
		w, ok := want[s.Name]
		if !ok {
			continue
		}
		seen++
		if got := legionNodeRowID(s); got != w {
			t.Errorf("%s (library id %s) resolved to %s, want %s", s.Name, s.ID, got, w)
		}
	}
	if seen != 2 {
		t.Fatalf("found %d of the 2 library rows named Powerful/Frenzied Faith — the fixture premise moved", seen)
	}
}
