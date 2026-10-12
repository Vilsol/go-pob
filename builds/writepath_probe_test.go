package builds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func countAll(s, sub string) int { return strings.Count(s, sub) }

// TestRoundTripIsNotIdempotentUnderConfigSets is the probe for the load-time mutation class.
func TestRoundTripIsNotIdempotentUnderConfigSets(t *testing.T) {
	const doc = `<PathOfBuilding>
	<Build level="1" targetVersion="3_0" className="Scion"/>
	<Tree activeSpec="1"><Spec treeVersion="3_18"/></Tree>
	<Skills activeSkillSet="1"><SkillSet id="1"><Skill enabled="true"/></SkillSet></Skills>
	<Config activeConfigSet="1">
		<ConfigSet id="1" title="Default">
			<Input name="enemyIsBoss" string="Boss"/>
			<Input name="conditionStationary" boolean="true"/>
		</ConfigSet>
	</Config>
</PathOfBuilding>`

	inputsAfter := func(gen int, xmlDoc string) (string, int) {
		t.Helper()
		cur := xmlDoc
		for i := 0; i < gen; i++ {
			b, err := ParseBuildStr(cur)
			if err != nil {
				t.Fatalf("generation %d failed to parse: %v", i, err)
			}
			raw, err := SerializeBuild(b)
			if err != nil {
				t.Fatalf("generation %d failed to serialize: %v", i, err)
			}
			cur = string(raw)
		}
		return cur, countAll(cur, "<Input ")
	}

	_, gen0 := inputsAfter(0, doc)
	out1, gen1 := inputsAfter(1, doc)
	_, gen2 := inputsAfter(2, doc)
	_, gen3 := inputsAfter(3, doc)

	t.Logf("<Input> elements after 0/1/2/3 round trips: %d / %d / %d / %d", gen0, gen1, gen2, gen3)
	t.Logf("one round trip produces:\n%s", out1)

	if gen1 != gen0 {
		t.Fatalf("ROUND TRIP IS NOT IDEMPOTENT: %d <Input> elements in, %d out. "+
			"resolveActiveConfigSet copies the active set into Config.Inputs and SerializeBuild "+
			"marshals BOTH, so every save duplicates every config input. After three saves: %d",
			gen0, gen1, gen3)
	}
}

// TestParseBuildDoesNotPanicOnOutOfRangeActiveSpec pins the guard on `build.Tree.Specs[build.Tree.ActiveSpec-1]` (parser.go).
func TestParseBuildDoesNotPanicOnOutOfRangeActiveSpec(t *testing.T) {
	cases := map[string]string{
		"activeSpec absent (parses as 0, indexes Specs[-1])": `<PathOfBuilding>
	<Build level="1" targetVersion="3_0" className="Scion"/>
	<Tree><Spec treeVersion="3_18"/></Tree>
</PathOfBuilding>`,
		"activeSpec past the end": `<PathOfBuilding>
	<Build level="1" targetVersion="3_0" className="Scion"/>
	<Tree activeSpec="5"><Spec treeVersion="3_18"/></Tree>
</PathOfBuilding>`,
		"no Spec elements at all": `<PathOfBuilding>
	<Build level="1" targetVersion="3_0" className="Scion"/>
	<Tree activeSpec="1"/>
</PathOfBuilding>`,
		"no Tree element at all": `<PathOfBuilding>
	<Build level="1" targetVersion="3_0" className="Scion"/>
</PathOfBuilding>`,
	}

	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("ParseBuild PANICKED on %s: %v", name, r)
				}
			}()
			if _, err := ParseBuildStr(doc); err != nil {
				t.Logf("refused (acceptable): %v", err)
			}
		})
	}
}

// TestEveryCorpusBuildCarriesAnInRangeActiveSpec is the CONTROL for the fixture above.
func TestEveryCorpusBuildCarriesAnInRangeActiveSpec(t *testing.T) {
	dir := corpusDirForLoadedData(t)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}

	checked, outOfRange := 0, 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".xml" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("REFUSING: cannot read %s: %v", e.Name(), err)
		}
		if _, err := ParseBuild(raw); err != nil {
			t.Errorf("%s: %v", e.Name(), err)
			outOfRange++
		}
		checked++
	}

	if checked == 0 {
		t.Fatalf("REFUSING: %s holds no .xml build; a clean result here would mean nothing", dir)
	}
	t.Logf("%d builds parsed, %d refused", checked, outOfRange)
}

// TestTreeEditSurvivesASave is the probe for the derived-field class.
func TestTreeEditSurvivesASave(t *testing.T) {
	const doc = `<PathOfBuilding>
	<Build level="1" targetVersion="3_0" className="Scion"/>
	<Tree activeSpec="1"><Spec treeVersion="3_18" nodes="10,20,30"/></Tree>
	<Skills activeSkillSet="1"><SkillSet id="1"><Skill enabled="true"/></SkillSet></Skills>
</PathOfBuilding>`

	build, err := ParseBuildStr(doc)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	build.AllocateNodes([]int64{40})
	build.DeallocateNodes([]int64{20})

	raw, err := SerializeBuild(build)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	saved := string(raw)

	if countAll(saved, "<PassiveNodes>") > 0 {
		t.Errorf("the save carries %d <PassiveNodes> elements; PoB writes none and reads none, "+
			"so this is our data in a form only we can read:\n%s", countAll(saved, "<PassiveNodes>"), saved)
	}
	for _, junk := range []string{"<SlotEnabled>", "<DisplayLabel>"} {
		if countAll(saved, junk) > 0 {
			t.Errorf("the save carries %d %s elements — calculator runtime state, not saved build data",
				countAll(saved, junk), junk)
		}
	}

	again, err := ParseBuild(raw)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}

	got := map[int64]bool{}
	for _, n := range again.Build.PassiveNodes {
		got[n] = true
	}
	if !got[40] || got[20] || !got[10] || !got[30] {
		t.Fatalf("TREE EDIT LOST ON SAVE: allocated 40 and deallocated 20 over {10,20,30}, "+
			"but reloading the saved build gives %v (saved spec nodes=%q)",
			again.Build.PassiveNodes, again.Tree.Specs[0].NodesAttr)
	}
}
