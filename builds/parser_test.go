package builds

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestParseBuild(t *testing.T) {
	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	_, err = ParseBuild(file)
	testza.AssertNoError(t, err)
}

// TestParseBuildOldSkillLayout: PoB's SkillsTab:Load accepts <Skill> nodes sitting directly under <Skills>.
func TestParseBuildOldSkillLayout(t *testing.T) {
	file, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), "0001-juggernaut-molten-strike.xml"))
	if err != nil {
		t.Skipf("corpus unavailable: %v", err)
	}

	build, err := ParseBuild(file)
	testza.AssertNoError(t, err)

	testza.AssertEqual(t, 1, len(build.Skills.SkillSets))
	testza.AssertEqual(t, 1, build.Skills.SkillSets[0].ID)
	testza.AssertEqual(t, 1, build.Skills.ActiveSkillSet)
	testza.AssertGreater(t, len(build.Skills.SkillSets[0].Skills), 0)

	gems := 0
	for _, skill := range build.Skills.SkillSets[0].Skills {
		gems += len(skill.Gems)
	}
	testza.AssertGreater(t, gems, 0)

	raw, err := SerializeBuild(build)
	testza.AssertNoError(t, err)

	again, err := ParseBuild(raw)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, build.Skills.SkillSets, again.Skills.SkillSets)
}

func TestParseBuildLooseSkillsMergeIntoSetOne(t *testing.T) {
	build, err := ParseBuildStr(`<PathOfBuilding>
	<Build level="1" targetVersion="3_0" className="Scion"/>
	<Tree activeSpec="1"><Spec treeVersion="3_18"/></Tree>
	<Skills activeSkillSet="2">
		<Skill enabled="true" label="loose"><Gem nameSpec="Cyclone" level="20"/></Skill>
		<SkillSet id="1"><Skill enabled="true" label="in-set-1"/></SkillSet>
		<SkillSet id="2"><Skill enabled="true" label="in-set-2"/></SkillSet>
	</Skills>
</PathOfBuilding>`)
	testza.AssertNoError(t, err)

	testza.AssertEqual(t, 2, len(build.Skills.SkillSets))
	testza.AssertEqual(t, 2, build.Skills.ActiveSkillSet)
	testza.AssertEqual(t, 2, len(build.Skills.SkillSets[0].Skills))
	testza.AssertEqual(t, "in-set-1", build.Skills.SkillSets[0].Skills[0].Label)
	testza.AssertEqual(t, "loose", build.Skills.SkillSets[0].Skills[1].Label)
	testza.AssertEqual(t, 1, len(build.Skills.SkillSets[1].Skills))
}

var nilCleanupRegex = regexp.MustCompile(`\w+?="nil"`)

func TestStripNilAttrsMatchesRegex(t *testing.T) {
	cases := []string{
		``,
		`nothing to do here`,
		`<Build a="nil"/>`,
		`<Build a="nil" b="1" c="nil"/>`,
		`="nil"`,
		`=="nil"`,
		`_x9="nil"`,
		`a="nil"="nil"`,
		`a="nil"b="nil"`,
		`"nil"`,
		`ä="nil"`,
		`<Build className="nil" ascendClassName="nil" level="nil"`,
		`trailing text after a="nil" and more`,
	}

	for _, c := range cases {
		testza.AssertEqual(t, string(nilCleanupRegex.ReplaceAllLiteral([]byte(c), []byte{})), string(stripNilAttrs([]byte(c))), "input %q", c)
	}
}

func TestStripNilAttrsMatchesRegexOnCorpus(t *testing.T) {
	corpusDir := corpusDirForLoadedData(t)
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Skipf("corpus unavailable: %v", err)
	}

	checked := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".xml" {
			continue
		}

		raw, err := os.ReadFile(filepath.Join(corpusDir, e.Name()))
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, string(nilCleanupRegex.ReplaceAllLiteral(raw, []byte{})), string(stripNilAttrs(raw)), e.Name())

		checked++
		if checked >= 200 {
			break
		}
	}

	testza.AssertGreater(t, checked, 0)
}

// TestParseBuildTreeViewStatDifferences: PassiveTreeView.lua:79, 96-98: the tree's stat differences are on unless the build's <TreeView> turns them off.
func TestParseBuildTreeViewStatDifferences(t *testing.T) {
	for _, c := range []struct {
		attr string
		want bool
	}{{"", true}, {` showStatDifferences="false"`, false}, {` showStatDifferences="true"`, true}} {
		file, err := os.ReadFile("../testdata/builds/Fireball.xml")
		testza.AssertNoError(t, err)
		build, err := ParseBuild(regexp.MustCompile(`<TreeView[^>]*>`).ReplaceAll(file, []byte(`<TreeView zoomLevel="0"`+c.attr+`/>`)))
		testza.AssertNoError(t, err)
		if build.TreeView.ShowStatDifferences != c.want {
			t.Errorf("%q: %v", c.attr, build.TreeView.ShowStatDifferences)
		}
	}
}
