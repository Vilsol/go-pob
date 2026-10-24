package data

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

// TestLatestTreeVersionFollowsGameData pins the PASSIVE TREE to the same version as the GAME DATA.
func TestLatestTreeVersionFollowsGameData(t *testing.T) {
	want := map[string]pob.TreeVersion{
		"3.18": TreeVersion3_18,
		"3.20": TreeVersion3_20,
		"3.29": TreeVersion3_29,
	}[raw.LatestVersion]

	if want == "" {
		t.Skipf("no tree registered for game data %q — add one to tree_versions.go", raw.LatestVersion)
	}

	testza.AssertEqual(t, want, LatestTreeVersion,
		"game data is %q so the passive tree must be %q, not %q. env.go:118 applies "+
			"LatestTreeVersion to every build, so a mismatch grades every build against the "+
			"wrong tree while still producing plausible numbers",
		raw.LatestVersion, want, LatestTreeVersion)
}
