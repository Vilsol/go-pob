package calculator

import (
	"fmt"
	"maps"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func powerShardRowLines(rows []PowerReportEntry) []string {
	f := func(v float64) string { return strconv.FormatUint(math.Float64bits(v), 16) }
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, fmt.Sprintf("%d\t%q\t%s\t%s\t%s\t%d\t%v\t%v\t%s\t%s\t%q", r.ID, r.Name, r.Type, f(r.Power), f(r.PathPower), r.PathDist,
			r.Allocated, r.Cluster, f(r.Offence), f(r.Defence), r.Stats))
	}
	return out
}

var powerShardVarietyBuilds = []string{
	"0210-occultist-death-aura-4", "0610-assassin-icicle-mine-of-fanning-3", "0365-luminary-vaal-cold-snap",
	"0221-reliquarian-ethereal-knives", "1111-necromancer-triggered-summon-phantasm-2", "0043-pathfinder-toxic-rain",
	"2312-ladder-godhatemee_2544.Evilcloudv", "0429-slayer-precision", "0332-chieftain-righteous-fire-5",
	"0436-inquisitor-eye-of-winter", "0006-elementalist-storm-burst", "0933-assassin-viper-strike-of-the-mamba-12",
	"0714-luminary-perforate",
}

var powerShardWideBuilds = []string{
	"0007-elementalist-storm-burst-of-repulsion", "0053-juggernaut-unknown", "0058-saboteur-cyclone-3",
	"0116-reliquarian-blade-blast", "0134-chieftain-detonate-dead", "0140-necromancer-summon-holy-relic-of-conviction",
	"0209-elementalist-lightning-tendrils-of-eccentricity", "0242-occultist-cyclone", "0269-juggernaut-determination",
	"0289-necromancer-animate-weapon-of-ranged-arms-5", "0359-hierophant-divine-retribution", "0382-reliquarian-animate-guardian",
	"0397-hierophant-bladefall-of-impaling", "0423-assassin-lancing-steel-of-spraying-3", "0482-luminary-intuitive-link",
	"0487-juggernaut-double-strike-of-momentum", "0503-reliquarian-vaal-blade-vortex", "0530-hierophant-kinetic-fusillade-7",
	"0532-necromancer-summon-holy-relic-of-conviction-7", "0549-guardian-tempest-shield", "0567-hierophant-storm-burst-of-repulsion-4",
	"0621-assassin-bladefall-3", "0636-hierophant-storm-burst-6", "0702-necromancer-summon-holy-relic-of-conviction-9",
	"0756-occultist-winter-orb-17", "0828-assassin-crackling-lance", "0901-reliquarian-volatile-dead-2",
	"10041-ladder-FrozenBlobman_2931.BottingMelon", "10121-ladder-GizmoSaveTheWorld_2311.АуработКинулМеня", "10160-ladder-Gres0125_1532.GresDontStandInMercFace",
	"13785-ladder-idklbcool_4254.Rokuthump", "3371-ladder-ZechsMerq_1777.ChaoticSea", "4777-ladder-Gardisau_5730.ALLFLAME_GARDISAU",
	"6515-ladder-WaisoAllflame_7715.WaisoAllflame",
}

// TestPowerShardsMergeToTheReport: K shards of a report, merged, are the unsharded report bit for bit.
func TestPowerShardsMergeToTheReport(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true")
	}
	casesFor := map[string][]string{}
	for _, name := range append([]string{"0018-berserker-heavy-strike", "0040-trickster-exsanguinate-of-transmission", "0019-champion-herald-of-agony"},
		powerBenchBuilds...) {
		casesFor[name] = []string{"Life:2", "OffDef:2", "CombinedDPS:2", "FullDPS:2"}
	}
	variety := powerShardVarietyBuilds
	if os.Getenv("POWER_SHARD_WIDE") == "1" {
		variety = append(slices.Clone(variety), powerShardWideBuilds...)
	}
	for _, name := range variety {
		casesFor[name] = []string{"Life:2", "CombinedDPS:2"}
	}
	if v := os.Getenv("POWER_SHARD_BUILDS"); v != "" {
		casesFor = map[string][]string{}
		for _, name := range strings.Fields(v) {
			casesFor[name] = []string{"Life:2", "OffDef:2", "CombinedDPS:2", "FullDPS:2"}
		}
	}
	if v := os.Getenv("POWER_SHARD_CASES"); v != "" {
		for name := range casesFor {
			casesFor[name] = strings.Fields(v)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(casesFor)) {
		for _, c := range casesFor[name] {
			t.Run(name+"/"+c, func(t *testing.T) {
				t.Parallel()
				stat, depthStr, _ := strings.Cut(c, ":")
				depth, err := strconv.Atoi(depthStr)
				if err != nil {
					t.Fatalf("bad case %q", c)
				}
				run := func(index, count int) PowerShardRows {
					b := GetMiscCalculator(parseCorpusBuild(t, name+".xml")).NewPowerBuilder(powerOracleStat(stat), depth)
					var owned []int64
					if count > 0 {
						owned = b.Shard(index, count)
					}
					for b.Progress() < 1 {
						b.Step(time.Second)
					}
					return PowerShardRows{Rows: b.Report(), Owned: owned, Ascendancy: b.AscendancyPower()}
				}
				single := run(0, 0)
				want := powerShardRowLines(single.Rows)
				wantAsc := powerShardRowLines(single.Ascendancy)
				if len(wantAsc) == 0 {
					t.Fatal("REFUSING: no ascendancy node")
				}
				if len(want) < 100 {
					t.Fatalf("REFUSING: %d rows", len(want))
				}
				for k := 1; k <= 4; k++ {
					shards := make([]PowerShardRows, k)
					for i := range k {
						shards[i] = run(i, k)
						if k > 1 && len(shards[i].Owned) == 0 {
							t.Fatalf("K=%d: shard %d owns no node", k, i)
						}
					}
					if got := powerShardRowLines(MergePowerAscendancy(shards)); !slices.Equal(got, wantAsc) {
						t.Errorf("K=%d: ascendancy entries differ (%d merged, %d in the report)", k, len(got), len(wantAsc))
					}
					got := powerShardRowLines(MergePowerReports(shards))
					var bad []string
					for i := range max(len(want), len(got)) {
						var w, g string
						if i < len(want) {
							w = want[i]
						}
						if i < len(got) {
							g = got[i]
						}
						if w != g {
							bad = append(bad, fmt.Sprintf("line %d:\n  want %s\n  got  %s", i+1, w, g))
						}
					}
					if len(bad) > 0 {
						t.Errorf("K=%d: %d of %d rows differ:\n%s", k, len(bad), len(want), strings.Join(bad[:min(5, len(bad))], "\n"))
					}
				}
			})
		}
	}
}
