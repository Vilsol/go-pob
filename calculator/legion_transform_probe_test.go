package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
)

// TestLegionReplacementResolves is the control for the per-seed transform.
func TestLegionReplacementResolves(t *testing.T) {
	if err := raw.InitializeAll(raw.LatestVersion, nil); err != nil {
		t.Fatalf("REFUSING: game data did not initialise: %v", err)
	}

	jewel := mod.LegionJewel{
		ID:        7094,
		Conqueror: mod.ConquerorType{ID: "1", Type: "vaal"},
	}

	for _, graphID := range []string{"31875", "34171", "46910", "23027"} {
		replaceWith, added := legionReplacement(jewel, graphID)
		if replaceWith == nil && len(added) == 0 {
			t.Errorf("node %s: legionReplacement returned nothing", graphID)
			continue
		}
		if replaceWith != nil {
			t.Logf("node %-6s -> %-32s %v (+%d additions)", graphID, replaceWith.ID, replaceWith.Stats, len(added))
		} else {
			t.Logf("node %-6s -> no replacement, %d additions: %v", graphID, len(added), added)
		}
	}
}

// TestConqueredNodesReachesTheVaalBranch checks the resolver is actually WIRED, not merely correct.
func TestConqueredNodesReachesTheVaalBranch(t *testing.T) {
	const build = "0012-occultist-unknown.xml"

	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)

	rewrites := nodeRewrites(env)
	if len(rewrites) == 0 {
		t.Fatalf("REFUSING: %s produced no node rewrites at all — it sockets a timeless jewel, so "+
			"this is a stamping failure, not an empty result", build)
	}

	replaced, added, stampedOnly := 0, 0, 0
	for _, r := range rewrites {
		switch {
		case r.replaceWith != nil:
			replaced++
		case len(r.addStats) > 0:
			added++
		default:
			stampedOnly++
		}
	}
	t.Logf("%s: %d rewrites — %d replaced, %d augmented, %d stamped only",
		build, len(rewrites), replaced, added, stampedOnly)

	if replaced == 0 && added == 0 {
		t.Errorf("%s: every one of %d conquered nodes was stamped and rewritten by nothing. The "+
			"per-seed branches are not being reached.", build, len(rewrites))
	}
}
