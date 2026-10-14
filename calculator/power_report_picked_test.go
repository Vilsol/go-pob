package calculator

import (
	"slices"
	"testing"
	"time"

	"github.com/Vilsol/go-pob/data"
)

// TestPowerReportPickedMasteryEffect: an unallocated mastery's picked effect is the one whose power the node takes.
func TestPowerReportPickedMasteryEffect(t *testing.T) {
	t.Parallel()
	m := GetMiscCalculator(parseCorpusBuild(t, "0280-elementalist-fireball.xml"))
	var ps PowerStat
	for _, s := range PowerStats() {
		if s.Stat == "Life" {
			ps = s
		}
	}
	b := m.NewPowerBuilder(ps, 0)
	for b.Progress() < 1 {
		b.Step(time.Second)
	}
	byNode := map[int64][]PowerReportEntry{}
	var order []int64
	for _, r := range b.Report() {
		if r.Type != "Mastery" || r.Cluster {
			continue
		}
		if r.Allocated {
			if r.Picked {
				t.Errorf("node %d: an allocated mastery's row is picked", r.ID)
			}
			continue
		}
		if byNode[r.ID] == nil {
			order = append(order, r.ID)
		}
		byNode[r.ID] = append(byNode[r.ID], r)
	}
	scored := 0
	for _, id := range order {
		rows := byNode[id]
		effects := b.treeNode(id).MasteryEffects
		want, best := int64(0), 0.0
		for _, e := range effects {
			for _, r := range rows {
				if r.Effect == e.Effect && r.Power > best {
					want, best = e.Effect, r.Power
				}
			}
		}
		var got []int64
		for _, r := range rows {
			if !r.Picked {
				continue
			}
			got = append(got, r.Effect)
			k := slices.IndexFunc(effects, func(e data.MasteryEffect) bool { return e.Effect == r.Effect })
			if k < 0 || !slices.Equal(r.Stats, splitStatLines(effects[k].Stats)) {
				t.Errorf("node %d: the picked row's lines %q are not its effect's", id, r.Stats)
			}
		}
		if best <= 0 {
			if len(got) != 0 {
				t.Errorf("node %d: picked %v with no effect of any power", id, got)
			}
			continue
		}
		scored++
		if !slices.Equal(got, []int64{want}) {
			t.Errorf("node %d: picked %v, want %d (power %g)", id, got, want, best)
		}
	}
	t.Logf("%d unallocated masteries, %d with a picked effect", len(order), scored)
	if scored == 0 {
		t.Fatal("REFUSING: no mastery has an effect of any power")
	}
}
