package calculator

import (
	"os"
	"testing"
)

// TestDistanceToClassStartMatchesPerSocketBFS is the control for replacing N per-socket breadth- first searches with ONE from the class start.
func TestDistanceToClassStartMatchesPerSocketBFS(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	_, names := corpusNamesForLoadedData(t)

	var builds, sockets, nonZero int
	for _, name := range names {
		env := corpusEnvForLoadedData(t, name)
		if env == nil || env.Spec == nil || len(env.AllocatedNodes) == 0 {
			continue
		}
		classStart := classStartNodeID(env.Spec.Tree(), env.Spec.ClassID)
		if classStart == "" {
			continue
		}
		builds++

		fast := buildDistanceToClassStart(env.AllocatedNodes, classStart)

		for id, node := range env.AllocatedNodes {
			if node.IsJewelSocket == nil && node.ExpansionJewel == nil {
				continue
			}
			sockets++
			want := setNodeDistanceToClassStart(id, env.AllocatedNodes, classStart)
			if got := fast[id]; got != want {
				t.Fatalf("%s node %s: one-BFS gave %d, per-socket BFS gave %d", name, id, got, want)
			}
			if want > 0 {
				nonZero++
			}
		}

		if builds >= 250 {
			break
		}
	}

	if builds == 0 || sockets == 0 {
		t.Fatal("REFUSING: no build in the sample had an allocated tree with a class start, " +
			"so this test compared nothing")
	}
	if nonZero == 0 {
		t.Fatalf("REFUSING: all %d sockets across %d builds scored 0, so the comparison cannot "+
			"distinguish a correct distance from a constant zero", sockets, builds)
	}
	t.Logf("compared %d sockets across %d builds, %d with a nonzero distance", sockets, builds, nonZero)
}
