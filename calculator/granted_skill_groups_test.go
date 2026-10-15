package calculator

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

// TestGrantedSkillGroupsFollowTheTree pins CalcSetup.lua:236-249, :1437-1449 and :1550-1562.
func TestGrantedSkillGroupsFollowTheTree(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	corpusDir, _ := corpusNamesForLoadedData(t)
	file, err := os.ReadFile(filepath.Join(corpusDir, "1606-chieftain-treeskill-4.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatal(err)
	}
	groups := &build.Skills.SkillSets[activeSkillSetIndex(build.Skills)].Skills
	sources := func() []string {
		out := make([]string, 0, len(*groups))
		for _, g := range *groups {
			out = append(out, g.Source)
		}
		return out
	}
	if !slices.Contains(sources(), "Tree:10916") {
		t.Fatal("CONTROL FAILED: the fixture has no Tree:10916 group")
	}
	*groups = slices.DeleteFunc(*groups, func(g pob.Skill) bool { return g.Source == "Tree:10916" })
	stale := "Item:999:Nothing"
	*groups = append(*groups, pob.Skill{Enabled: true, Source: stale, SourceAttr: &stale, Gems: []pob.Gem{{SkillID: "WardShatter", Level: 20, Enabled: true}}})

	BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	got := sources()
	if !slices.Contains(got, "Tree:10916") {
		t.Errorf("no Tree:10916 group after a calculation: %q", got)
	}
	if slices.Contains(got, stale) {
		t.Errorf("the unmatched %s group survived: %q", stale, got)
	}
}
