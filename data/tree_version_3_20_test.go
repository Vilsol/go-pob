package data

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

// TestTreeVersion3_20IsRegistered pins the 3.20 Forbidden Sanctum passive tree as loadable.
func TestTreeVersion3_20IsRegistered(t *testing.T) {
	v, ok := TreeVersions[TreeVersion3_20]
	testza.AssertTrue(t, ok, "TreeVersion3_20 is not registered in TreeVersions")
	if !ok {
		return
	}

	testza.AssertEqual(t, "3.20", v.Display,
		"Display is what builds the CDN URL — raw.TreeURL(v.Display) in "+
			"tree_versions.go — so it must be the DOTTED form, not the 3_20 key")

	tree := v.Tree()
	testza.AssertNotNil(t, tree, "3.20 tree failed to load")

	testza.AssertTrue(t, len(tree.Nodes) > 1000,
		"3.20 tree has %d nodes, expected >1000 — a short read or an error page would look like this",
		len(tree.Nodes))
}

// TestLatestTreeVersionIsUnchangedBy3_20 guards what the REGISTRATION may do on its own.
func TestLatestTreeVersionIsUnchangedBy3_20(t *testing.T) {
	testza.AssertEqual(t, TreeVersion3_10, DefaultTreeVersion,
		"DefaultTreeVersion is the fallback for a build that names no tree version, and it is "+
			"deliberately ancient. Moving it changes which tree those builds resolve against")
}
