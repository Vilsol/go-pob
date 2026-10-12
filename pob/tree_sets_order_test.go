package pob

import (
	"slices"
	"testing"
)

func titles(b *PathOfBuilding) []string {
	out := make([]string, 0, len(b.Tree.Specs))
	for _, s := range b.Tree.Specs {
		out = append(out, SpecTitle(s))
	}
	return out
}

// TestMoveSpecKeepsTheActiveTree: Dragging a tree moves it and keeps the active tree active, whichever way it moves.
func TestMoveSpecKeepsTheActiveTree(t *testing.T) {
	mk := func(active int) *PathOfBuilding {
		b := &PathOfBuilding{}
		for _, title := range []string{"A", "B", "C", "D"} {
			b.Tree.Specs = append(b.Tree.Specs, Spec{Title: &title})
		}
		b.Tree.ActiveSpec = active
		return b
	}
	cases := []struct {
		active, from, to int
		order            []string
		wantActive       int
	}{
		{2, 2, 4, []string{"A", "C", "D", "B"}, 4},
		{2, 4, 1, []string{"D", "A", "B", "C"}, 3},
		{3, 1, 4, []string{"B", "C", "D", "A"}, 2},
		{1, 3, 2, []string{"A", "C", "B", "D"}, 1},
		{4, 1, 2, []string{"B", "A", "C", "D"}, 4},
		{2, 5, 1, []string{"A", "B", "C", "D"}, 2},
	}
	for _, c := range cases {
		b := mk(c.active)
		b.MoveSpec(c.from, c.to)
		if got := titles(b); !slices.Equal(got, c.order) || b.Tree.ActiveSpec != c.wantActive {
			t.Errorf("active %d, %d -> %d: %v active %d, want %v active %d", c.active, c.from, c.to, got, b.Tree.ActiveSpec, c.order, c.wantActive)
		}
	}
}
