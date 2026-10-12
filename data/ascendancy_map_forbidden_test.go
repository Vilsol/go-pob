package data

import "testing"

// TestAscendancyMapCarriesForbiddenNodes pins PassiveTree.lua:563/578 for the Forbidden Flame/Flesh nodes.
func TestAscendancyMapCarriesForbiddenNodes(t *testing.T) {
	version := TreeVersions[LatestTreeVersion]
	if version == nil {
		t.Skipf("NOT-RUN: no tree registered for %q", LatestTreeVersion)
	}
	byName := version.AscendancyMap()
	if len(byName) < 50 {
		t.Fatalf("REFUSING: %d ascendancyMap entries; the tree did not load", len(byName))
	}
	want := map[string]string{
		"nine lives":          "27602",
		"searing purity":      "57568",
		"indomitable resolve": "52435",
		"unleashed potential": "19355",
		"fatal flourish":      "42469",
		"fury of nature":      "18054",
	}
	for name, id := range want {
		if got := byName[name]; got != id {
			t.Errorf("ascendancyMap[%q] = %q, want %q", name, got, id)
		}
	}
}
