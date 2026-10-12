package calculator

import (
	"testing"
)

// TestClickBloodlineNode: A click on another bloodline's node selects that bloodline.
func TestClickBloodlineNode(t *testing.T) {
	tree := specTree()
	altIndex := func(id string) int {
		for i, a := range tree.AlternateAscendancies {
			if string(a.ID) == id {
				return i + 1
			}
		}
		t.Fatalf("no bloodline %s", id)
		return 0
	}
	pick := func(bloodline string) int64 {
		for _, n := range tree.Nodes {
			if n.AscendancyName != nil && *n.AscendancyName == bloodline && n.IsBloodline != nil && *n.IsBloodline &&
				!isAscendClassStartNode(n) && n.Skill != nil && n.IsNotable != nil && *n.IsNotable &&
				(n.IsMultipleChoiceOption == nil || !*n.IsMultipleChoiceOption) {
				return *n.Skill
			}
		}
		t.Fatalf("no %s notable", bloodline)
		return 0
	}
	build := parseCorpusBuild(t, "0243-necromancer-raise-spectre.xml")
	bloodlineNodes := func(bloodline string) int {
		n := 0
		for _, id := range build.Build.PassiveNodes {
			if node, ok := tree.Nodes[nodeKey(id)]; ok && node.AscendancyName != nil && *node.AscendancyName == bloodline {
				n++
			}
		}
		return n
	}

	used := CountAllocNodes(build)
	spec := func() *int { return build.Tree.Specs[build.ActiveSpecIndex()].SecondaryAscendClassID }

	oshabi := pick("Oshabi")
	if got := ClickAscendancyNode(build, oshabi, ClassAsk); !got.Changed || !got.Handled {
		t.Fatalf("bloodline click: %+v", got)
	}
	if s := spec(); s == nil || *s != altIndex("Oshabi") {
		t.Fatalf("secondary ascendancy %v, want %d", s, altIndex("Oshabi"))
	}
	if !hasNode(build, oshabi) || bloodlineNodes("Oshabi") < 2 || CountAllocNodes(build) != used {
		t.Fatalf("clicked allocated %v, Oshabi nodes %d, points %d / %d", hasNode(build, oshabi), bloodlineNodes("Oshabi"), CountAllocNodes(build), used)
	}
	if build.Build.AscendClassName != "Necromancer" {
		t.Fatalf("primary ascendancy changed to %s", build.Build.AscendClassName)
	}

	aul := pick("Aul")
	if got := ClickAscendancyNode(build, aul, ClassAsk); !got.Changed || !got.Handled {
		t.Fatalf("second bloodline click: %+v", got)
	}
	if s := spec(); s == nil || *s != altIndex("Aul") || !hasNode(build, aul) || bloodlineNodes("Oshabi") != 0 {
		t.Fatalf("secondary %v, Aul allocated %v, Oshabi nodes left %d", s, hasNode(build, aul), bloodlineNodes("Oshabi"))
	}

	for _, n := range tree.Nodes {
		if n.AscendancyName != nil && *n.AscendancyName == "Aul" && n.Skill != nil && !hasNode(build, *n.Skill) && !isAscendClassStartNode(n) {
			if got := ClickAscendancyNode(build, *n.Skill, ClassAsk); got.Handled {
				t.Fatalf("current bloodline node handled: %+v", got)
			}
			break
		}
	}
}
