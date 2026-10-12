package calculator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Vilsol/go-pob/builds"
)

// TestPowerBuilderCurrent: The tree shows the node being calculated.
func TestPowerBuilderCurrent(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), "0280-elementalist-fireball.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	var stat PowerStat
	for _, s := range PowerStats() {
		if s.Stat == "Life" {
			stat = s
		}
	}
	b := GetMiscCalculator(build).NewPowerBuilder(stat, 1)
	if len(b.tasks) < 3 {
		t.Fatalf("REFUSING: %d tasks", len(b.tasks))
	}
	seen := map[int64]bool{}
	for b.Progress() < 1 {
		id := b.Current()
		if id == 0 || b.treeNode(id).Skill == nil {
			t.Fatalf("task %d: Current %d is not a tree node", b.next, id)
		}
		seen[id] = true
		b.Step(time.Nanosecond)
	}
	if b.Current() != 0 {
		t.Fatalf("done, but Current is %d", b.Current())
	}
	if len(seen) < 2 {
		t.Fatalf("Current named %d nodes over %d tasks", len(seen), len(b.tasks))
	}
}
