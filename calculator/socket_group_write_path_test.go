package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

func loadCorpusBuild(t *testing.T, name string) *pob.PathOfBuilding {
	t.Helper()
	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name))
	if err != nil {
		t.Fatalf("REFUSING: %v (fetch the corpus with `mise run testdata`)", err)
	}
	build, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	return build
}

// TestSocketGroupWritesReachTheBuild is a WRITE-PATH test, and this repository has none.
func TestSocketGroupWritesReachTheBuild(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	build := loadCorpusBuild(t, "0006-elementalist-storm-burst.xml")
	sel := activeSkillSetIndex(build.Skills)
	if sel < 0 {
		sel = 0
	}

	for i, group := range build.Skills.SkillSets[sel].Skills {
		if group.DisplayLabel != "" || group.SlotEnabled {
			t.Fatalf("CONTROL IS VACUOUS: group[%d] arrives with DisplayLabel %q / SlotEnabled %v "+
				"already set, so this build cannot show that the calculation wrote them",
				i, group.DisplayLabel, group.SlotEnabled)
		}
	}

	(&Calculator{PoB: build}).BuildOutput(OutputModeMain)

	groups := build.Skills.SkillSets[sel].Skills
	labelled, enabled := 0, 0
	for _, group := range groups {
		if group.DisplayLabel != "" {
			labelled++
		}
		if group.SlotEnabled {
			enabled++
		}
	}
	if labelled == 0 {
		t.Errorf("after BuildOutput, NO socket group in build.Skills carries a DisplayLabel — the "+
			"writes at CalcSetup.lua:1793-1801 are not reaching the build (%d groups)", len(groups))
	}
	if enabled == 0 {
		t.Errorf("after BuildOutput, NO socket group in build.Skills has SlotEnabled — the write " +
			"at CalcSetup.lua:1758 is not reaching the build")
	}

	seen := map[string]int{}
	for i, group := range groups {
		if group.DisplayLabel != "" {
			seen[group.DisplayLabel]++
			_ = i
		}
	}
	if labelled > 1 && len(seen) == 1 {
		for label := range seen {
			t.Errorf("all %d labelled groups carry the SAME label %q — the label is being built "+
				"from a constant rather than from each group's own gems", labelled, label)
		}
	}
}

// TestMainActiveSkillStaysOneBased is the half that a "just take a pointer" fix would have broken, and it is the reason the two changes had to land together.
func TestMainActiveSkillStaysOneBased(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	build := loadCorpusBuild(t, "0006-elementalist-storm-burst.xml")
	sel := activeSkillSetIndex(build.Skills)
	if sel < 0 {
		sel = 0
	}

	before := make([]int, 0, len(build.Skills.SkillSets[sel].Skills))
	for _, group := range build.Skills.SkillSets[sel].Skills {
		before = append(before, group.MainActiveSkill)
	}
	if len(before) == 0 || before[0] < 1 {
		t.Fatalf("CONTROL IS VACUOUS: this build's first group arrives with mainActiveSkill %v; "+
			"the fixture must carry a 1-based value for the round-trip check to mean anything", before)
	}

	(&Calculator{PoB: build}).BuildOutput(OutputModeMain)

	for i, group := range build.Skills.SkillSets[sel].Skills {
		if group.MainActiveSkill < 1 {
			t.Errorf("group[%d] mainActiveSkill is %d after BuildOutput — the field is 1-BASED and "+
				"is saved to XML; 0 is the 0-based slice index leaking into build state",
				i, group.MainActiveSkill)
		}
		if group.MainActiveSkillCalcs < 1 {
			t.Errorf("group[%d] mainActiveSkillCalcs is %d after BuildOutput — same 1-based field",
				i, group.MainActiveSkillCalcs)
		}
	}

	for i, group := range build.Skills.SkillSets[sel].Skills {
		if group.MainActiveSkill != before[i] {
			t.Errorf("group[%d] mainActiveSkill moved %d -> %d across one BuildOutput. PoB clamps to "+
				"m_min(#skills, v) and this build's value was already in range, so it must not move",
				i, before[i], group.MainActiveSkill)
		}
	}

	second := loadCorpusBuild(t, "0006-elementalist-storm-burst.xml")
	calc := &Calculator{PoB: second}
	calc.BuildOutput(OutputModeMain)
	afterFirst := second.Skills.SkillSets[sel].Skills[0].MainActiveSkill
	calc.BuildOutput(OutputModeMain)
	if got := second.Skills.SkillSets[sel].Skills[0].MainActiveSkill; got != afterFirst {
		t.Errorf("mainActiveSkill drifted %d -> %d across two BuildOutput calls; the write-back must "+
			"be idempotent because it re-reads its own output", afterFirst, got)
	}
}
