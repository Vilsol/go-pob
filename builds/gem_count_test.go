package builds

import (
	"strings"
	"testing"
)

// TestGemCountRoundTripsAbsentAndZeroSeparately pins the WRITE half of `<Gem count>`.
func TestGemCountRoundTripsAbsentAndZeroSeparately(t *testing.T) {
	const header = `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
		`<Build level="90" className="Witch" ascendClassName="Occultist"/>` +
		`<Tree activeSpec="1"><Spec treeVersion="3_29" nodes=""/></Tree>`

	for _, tc := range []struct {
		name, attr string
		wantNil    bool
		wantValue  int
		wantInSave string
		why        string
	}{
		{
			name: "absent stays absent", attr: ``, wantNil: true, wantInSave: "",
			why: `writing count="0" makes PoB count the gem zero times`,
		},
		{
			name: `PoB's "nil" spelling stays absent`, attr: ` count="nil"`, wantNil: true, wantInSave: "",
			why: `the parser strips ="nil"; PoB's loader maps it and absent to the same 1`,
		},
		{
			name: "explicit zero survives", attr: ` count="0"`, wantValue: 0, wantInSave: `count="0"`,
			why: "120 gems over 18 corpus builds carry one; omitempty on an int would drop it",
		},
		{
			name: "a real count survives", attr: ` count="3"`, wantValue: 3, wantInSave: `count="3"`,
			why: "the ordinary case the attribute exists for",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build, err := ParseBuildStr(header +
				`<Skills activeSkillSet="1"><SkillSet id="1"><Skill enabled="true">` +
				`<Gem nameSpec="Spark" level="20" quality="0" enabled="true"` + tc.attr + `/>` +
				`</Skill></SkillSet></Skills></PathOfBuilding>`)
			if err != nil {
				t.Fatalf("ParseBuild: %v", err)
			}

			got := build.Skills.SkillSets[0].Skills[0].Gems[0].Count
			if tc.wantNil {
				if got != nil {
					t.Errorf("Gem.Count = %d, want nil — absent and 0 must stay distinct (%s)", *got, tc.why)
				}
			} else {
				if got == nil {
					t.Fatalf("Gem.Count = nil, want %d (%s)", tc.wantValue, tc.why)
				}
				if *got != tc.wantValue {
					t.Errorf("Gem.Count = %d, want %d", *got, tc.wantValue)
				}
			}

			out, err := SerializeBuild(build)
			if err != nil {
				t.Fatalf("SerializeBuild: %v", err)
			}
			hasCount := strings.Contains(string(out), "count=")
			if tc.wantInSave == "" {
				if hasCount {
					t.Errorf("saved XML carries a count attribute and must not (%s)", tc.why)
				}
				return
			}
			if !strings.Contains(string(out), tc.wantInSave) {
				t.Errorf("saved XML lost %s (%s)", tc.wantInSave, tc.why)
			}
		})
	}
}
