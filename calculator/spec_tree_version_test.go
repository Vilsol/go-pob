package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

// TestEnvUsesTheActiveSpecsTree pins TreeTab.lua:517: a spec is built on the tree of its own treeVersion, so a build
// saved on an older tree keeps that tree's nodes until it is converted. 22628 is a different notable in 3_29 and 3_30.
func TestEnvUsesTheActiveSpecsTree(t *testing.T) {
	for _, version := range []pob.TreeVersion{data.TreeVersion3_29, data.TreeVersion3_30} {
		t.Run(string(version), func(t *testing.T) {
			build, err := builds.ParseBuildStr(`<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
				`<Build level="90" className="Witch" ascendClassName="Necromancer"/>` +
				`<Tree activeSpec="1"><Spec treeVersion="` + string(version) + `" classId="3" ascendClassId="1" nodes="22628,36017"/></Tree>` +
				`<Skills activeSkillSet="1"><SkillSet id="1"/></Skills><Items activeItemSet="1"><ItemSet id="1"/></Items>` +
				`</PathOfBuilding>`)
			if err != nil {
				t.Fatalf("ParseBuild: %v", err)
			}
			env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
			if env.Spec.TreeVersion != version {
				t.Errorf("env.Spec.TreeVersion = %s, want the spec's %s", env.Spec.TreeVersion, version)
			}
			for _, id := range []string{"22628", "36017"} {
				want := data.TreeVersions[version].Tree().Nodes[id]
				got, ok := env.AllocatedNodes[id]
				if !ok {
					t.Fatalf("node %s is not allocated", id)
				}
				if *got.Name != *want.Name || len(got.Stats) != len(want.Stats) || (len(want.Stats) > 0 && got.Stats[len(want.Stats)-1] != want.Stats[len(want.Stats)-1]) {
					t.Errorf("node %s = %q %q, want %s's %q %q", id, *got.Name, got.Stats, version, *want.Name, want.Stats)
				}
			}
		})
	}
}

// TestTreeHelpersReadTheActiveSpecsTree: a skill granted by a tree node is named from the active spec's own tree.
func TestTreeHelpersReadTheActiveSpecsTree(t *testing.T) {
	for _, version := range []pob.TreeVersion{data.TreeVersion3_29, data.TreeVersion3_30} {
		t.Run(string(version), func(t *testing.T) {
			build, err := builds.ParseBuildStr(`<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
				`<Build level="90" className="Witch" ascendClassName="Necromancer"/>` +
				`<Tree activeSpec="1"><Spec treeVersion="` + string(version) + `" classId="3" ascendClassId="1" nodes="22628"/></Tree>` +
				`<Skills activeSkillSet="1"><SkillSet id="1"><Skill source="Tree:22628"/></SkillSet></Skills><Items activeItemSet="1"><ItemSet id="1"/></Items>` +
				`</PathOfBuilding>`)
			if err != nil {
				t.Fatalf("ParseBuild: %v", err)
			}
			want := *data.TreeVersions[version].Tree().Nodes["22628"].Name
			names := SocketGroupSourceNames(build, 0, 0)
			if len(names) != 1 || names[0].Name != want {
				t.Errorf("SocketGroupSourceNames = %v, want %s's %q", names, version, want)
			}
		})
	}
}
