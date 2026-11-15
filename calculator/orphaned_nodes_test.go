package calculator

import (
	"bytes"
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestOrphanedNodesLeaveTheCalculation: the fixture is a 3.29 tree with Life node 6712 removed, which cuts 36949,
// 10031 and mastery 41415 off the class start. The pinned PoB prunes them at load: Life 5661, the same as with the
// three removed by hand; with 6712 kept it is 6633.
func TestOrphanedNodesLeaveTheCalculation(t *testing.T) {
	requireGameData(t, "3.29")
	file, err := os.ReadFile("testdata/orphaned-nodes/life-node-removed.xml")
	if err != nil {
		t.Fatal(err)
	}
	connected := bytes.Replace(file, []byte(`nodes="`), []byte(`nodes="6712,`), 1)
	for _, c := range []struct {
		name string
		xml  []byte
		want float64
	}{{"orphaned", file, 5661}, {"connected", connected, 6633}} {
		build, err := builds.ParseBuild(c.xml)
		if err != nil {
			t.Fatal(err)
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if got := env.Player.Output["Life"]; math.Abs(got-c.want) > 1e-9*c.want {
			t.Errorf("%s: Life %v, PoB %v", c.name, got, c.want)
		}
	}
}
