package data

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestLargeClusterAnchorMatchesScan(t *testing.T) {
	for version, tv := range TreeVersions {
		tree := tv.Tree()
		if tree == nil {
			t.Fatalf("%s: tree not loaded", version)
		}
		found := 0
		for large := int64(0); large < 8; large++ {
			var matches []int64
			for _, node := range tree.Nodes {
				if j := node.ExpansionJewel; j != nil && j.Parent == nil && j.Size == 2 && j.Index == large && node.Skill != nil {
					matches = append(matches, *node.Skill)
				}
			}
			testza.AssertTrue(t, len(matches) <= 1, "%s: large index %d has %d sockets %v", version, large, len(matches), matches)
			got, ok := tree.LargeClusterAnchor(large)
			if len(matches) == 0 {
				testza.AssertFalse(t, ok, "%s: large index %d has no socket but the table answers %d", version, large, got)
				continue
			}
			found++
			testza.AssertTrue(t, ok, "%s: large index %d missing from the table", version, large)
			testza.AssertEqual(t, matches[0], got, "%s: large index %d", version, large)
		}
		testza.AssertTrue(t, found > 0, "%s: no large cluster sockets", version)
	}
}
