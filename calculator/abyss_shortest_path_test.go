package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

// TestShortestPathToClassStart pins PassiveSpec:GetShortestPathToClassStart.
func TestShortestPathToClassStart(t *testing.T) {
	zero := int64(0)

	tree := &data.Tree{Nodes: map[string]data.Node{
		"0": {ClassStartIndex: &zero, Out: []string{"7", "5"}},
		"5": {Out: []string{"9"}, In: []string{"0"}},
		"6": {Out: []string{"8"}},
		"7": {Out: []string{"8"}, In: []string{"0"}},
		"8": {Out: []string{"9"}, In: []string{"7", "6"}},
		"9": {In: []string{"8", "5"}},
	}}

	alloc := func(ids ...string) map[string]data.Node {
		m := make(map[string]data.Node, len(ids))
		for _, id := range ids {
			m[id] = tree.Nodes[id]
		}
		return m
	}
	envFor := func(ids ...string) *Environment {
		return &Environment{
			AllocatedNodes: alloc(ids...),
			Spec:           &PassiveSpec{TreeVersion: data.LatestTreeVersion, ClassID: pob.ClassID(0)},
		}
	}

	t.Run("walks only allocated nodes", func(t *testing.T) {
		got := shortestPathToClassStart(envFor("9", "8", "7", "6", "0"), tree, "9")
		want := map[string]bool{"9": true, "8": true, "7": true, "0": true}
		if len(got) != len(want) {
			t.Fatalf("path = %v, want %v — 5 is unallocated so the shortcut must not be taken, "+
				"and 6 is a dead end that must not be on the path", got, want)
		}
		for id := range want {
			if !got[id] {
				t.Errorf("path is missing %s: %v — :1012 seeds the table with BOTH the root and "+
					"the class start before walking the parent chain", id, got)
			}
		}
	})

	t.Run("takes the shortcut once it is allocated", func(t *testing.T) {
		got := shortestPathToClassStart(envFor("9", "8", "7", "5", "0"), tree, "9")
		if got["7"] || got["8"] {
			t.Errorf("path = %v, want the 9-5-0 shortcut — a BFS that still returns the long "+
				"way is not finding the SHORTEST path", got)
		}
	})

	t.Run("an unallocated socket has no path", func(t *testing.T) {
		if got := shortestPathToClassStart(envFor("8", "7", "0"), tree, "9"); got != nil {
			t.Errorf("path = %v, want nil — :993 refuses a root that is not allocated, and "+
				"PassiveSpec.lua:1122 gates the whole Abyss read on self.allocNodes[socketId]", got)
		}
	})

	t.Run("no class start for the build's class", func(t *testing.T) {
		env := envFor("9", "8", "7", "0")
		env.Spec.ClassID = pob.ClassID(4)
		if got := shortestPathToClassStart(env, tree, "9"); got != nil {
			t.Errorf("path = %v, want nil — curClass.startNodeId does not resolve", got)
		}
	})
}
