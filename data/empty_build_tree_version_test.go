package data

import (
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestEmptyBuildIsOnTheLatestTree: A new build's tree is the latest one: an older version makes the Tree page offer to convert it.
func TestEmptyBuildIsOnTheLatestTree(t *testing.T) {
	got := builds.EmptyBuild().Tree.Specs[0].TreeVersion
	if got != LatestTreeVersion {
		t.Fatalf("a new build's tree is %q, the latest is %q", got, LatestTreeVersion)
	}
}
