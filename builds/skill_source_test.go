package builds

import (
	"strings"
	"testing"
)

// TestSkillSourceRoundTripsAbsentAndEmptySeparately pins the write-path half of the most destructive defect the census has found.
func TestSkillSourceRoundTripsAbsentAndEmptySeparately(t *testing.T) {
	const header = `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
		`<Build level="90" className="Witch" ascendClassName="Occultist"/>` +
		`<Tree activeSpec="1"><Spec treeVersion="3_29" nodes=""/></Tree>`

	for _, tc := range []struct {
		name, attr string
		wantInSave bool
		wantSource string
		why        string
	}{
		{
			name: "absent stays absent", attr: ``, wantInSave: false, wantSource: "",
			why: "an ordinary socket group; writing source=\"\" makes CalcSetup.lua:1452 delete it",
		},
		{
			name: "explicit empty survives", attr: ` source=""`, wantInSave: true, wantSource: "",
			why: "PoB writes this on 628 corpus builds; omitempty would drop it",
		},
		{
			name: "a real source survives", attr: ` source="Item:23:Prism Guardian"`, wantInSave: true,
			wantSource: "Item:23:Prism Guardian",
			why:        "the item-granted case the attribute exists for",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build, err := ParseBuildStr(header +
				`<Skills activeSkillSet="1"><SkillSet id="1">` +
				`<Skill enabled="true"` + tc.attr + `><Gem nameSpec="Spark" level="20" quality="0" enabled="true"/></Skill>` +
				`</SkillSet></Skills></PathOfBuilding>`)
			if err != nil {
				t.Fatalf("ParseBuild: %v", err)
			}

			got := build.Skills.SkillSets[0].Skills[0].Source
			if got != tc.wantSource {
				t.Errorf("Skill.Source = %q, want %q — builds.resolveSkillSources did not fill it", got, tc.wantSource)
			}

			out, err := SerializeBuild(build)
			if err != nil {
				t.Fatalf("SerializeBuild: %v", err)
			}
			hasAttr := strings.Contains(string(out), `source=`)
			if hasAttr != tc.wantInSave {
				t.Errorf("saved XML contains a source attribute = %v, want %v (%s)", hasAttr, tc.wantInSave, tc.why)
			}
			if tc.wantInSave && !strings.Contains(string(out), `source="`+tc.wantSource+`"`) {
				t.Errorf("saved XML lost source=%q", tc.wantSource)
			}
		})
	}
}
