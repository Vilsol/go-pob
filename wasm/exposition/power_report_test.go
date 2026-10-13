package exposition

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
)

// TestPowerShardsOnTheSavedBuildMergeToTheReport: The power workers' path: shard 0 on the open build.
func TestPowerShardsOnTheSavedBuildMergeToTheReport(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true")
	}
	initItemData(t)
	const count = 3
	for _, file := range []string{"0280-elementalist-fireball.xml", "0369-necromancer-automation.xml"} {
		t.Run(file, func(t *testing.T) {
			build := corpusBuild(t, file)
			rows := func(r []calculator.PowerReportEntry) []string {
				out := make([]string, len(r))
				for i, e := range r {
					out[i] = fmt.Sprintf("%d %q %x %x %d %v %v %x %x", e.ID, e.Name, math.Float64bits(e.Power), math.Float64bits(e.PathPower), e.PathDist,
						e.Allocated, e.Cluster, math.Float64bits(e.Offence), math.Float64bits(e.Defence))
				}
				return out
			}
			run := func(p *PowerBuilder) {
				for p.Step(1000) < 1 {
				}
			}
			single := NewPowerBuilder(GetMiscCalculator(build), "Life", 2)
			run(single)
			want := rows(single.Report())

			xml, err := builds.SerializeBuild(build)
			if err != nil {
				t.Fatal(err)
			}
			shards := make([]calculator.PowerShardRows, count)
			for i := range count {
				b := build
				if i > 0 {
					if b, err = builds.ParseBuild(xml); err != nil {
						t.Fatal(err)
					}
				}
				p := NewPowerBuilderShard(GetMiscCalculator(b), "Life", 2, i, count)
				run(p)
				shards[i] = p.Shard()
			}
			got := rows(MergePowerReports(shards))
			if len(want) < 100 || len(got) != len(want) {
				t.Fatalf("%d rows merged, %d in the report", len(got), len(want))
			}
			bad := 0
			for i := range want {
				if got[i] != want[i] {
					if bad < 5 {
						t.Errorf("row %d:\n  want %s\n  got  %s", i, want[i], got[i])
					}
					bad++
				}
			}
			if bad > 0 {
				t.Errorf("%d of %d rows differ", bad, len(want))
			}
		})
	}
}
