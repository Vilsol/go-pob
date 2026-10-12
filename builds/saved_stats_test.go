package builds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestSerializeBuildWithStatsWritesTheGivenStats: Build.lua:1014-1061: a save writes the stats of its last calculation, not the ones it was loaded with.
func TestSerializeBuildWithStatsWritesTheGivenStats(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), "0280-elementalist-fireball.xml"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	build, err := ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	before := len(build.Build.PlayerStats)
	out, err := SerializeBuildWithStats(build,
		[]pob.PlayerStat{{Stat: "Life", Value: 1234}, {Stat: "PowerChargesMax", Value: 4}},
		[]pob.FullDPSSkill{{Stat: "2x Fireball", Value: "21", Source: "Item"}},
		[]pob.PlayerStat{{Stat: "Life", Value: 500}})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		`<PlayerStat value="1234" stat="Life"`, `<PlayerStat value="4" stat="PowerChargesMax"`,
		`<FullDPSSkill stat="2x Fireball" value="21" skillPart="" source="Item"`, `<MinionStat value="500" stat="Life"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Count(s, "<PlayerStat ") != 2 {
		t.Errorf("the loaded stats were kept: %d PlayerStat elements", strings.Count(s, "<PlayerStat "))
	}
	if len(build.Build.PlayerStats) != before {
		t.Error("the build itself was changed")
	}
}
