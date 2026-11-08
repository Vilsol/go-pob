package exposition

import (
	"maps"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/calculator2"
)

// PoB-PoE2's sidebar for sorceress_spark.xml with node 44871 (+10 to maximum Energy Shield) allocated, from
// oracle/poe2_sidebar.lua: Energy Shield 10 where the build has none. A what-if leaves the build as it was.
func TestNodeCompareAndPowerLeaveTheBuild(t *testing.T) {
	calc, before := openFixture(t, "sorceress_spark.xml")
	want := maps.Clone(before.Output)

	cmp, err := calc.NodeCompare(44871, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(cmp.Blocks) != 1 || cmp.Blocks[0].Header != "Allocating this node will give you:" || cmp.Blocks[0].Compare.Output["EnergyShield"] != 10 {
		t.Fatalf("blocks %+v", cmp.Blocks)
	}
	tip, err := calc.NodeTip(44871, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(tip.PathLines, []string{"1 points to node"}) {
		t.Fatalf("path lines %q", tip.PathLines)
	}

	if _, err := calc.StartNodePower("EnergyShield", 1); err != nil {
		t.Fatal(err)
	}
	for {
		p, err := calc.StepNodePower(50)
		if err != nil {
			t.Fatal(err)
		}
		if p.Finished {
			break
		}
	}
	power, err := calc.NodePower()
	if err != nil {
		t.Fatal(err)
	}
	i := slices.Index(power.IDs, 44871)
	if i < 0 || !power.HasSingle[i] || power.Single[i] != 10 || power.Distance[i] != 1 {
		t.Fatalf("node 44871's power: index %d of %d", i, len(power.IDs))
	}
	rows, err := calc.NodePowerReport()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(rows, func(r calculator2.PowerReportRow) bool { return r.ID == 44871 && r.Power == 10 && r.PathDist == 1 }) {
		t.Fatalf("no report row for node 44871 among %d", len(rows))
	}

	after, err := calc.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(after.Output, want) {
		t.Fatal("the build's output changed after the what-ifs")
	}
	if _, err := calc.NodePower(); err == nil {
		t.Fatal("the power builder still answers after a recalculation")
	}
}
