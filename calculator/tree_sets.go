package calculator

import (
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

type TreeSpecRow struct {
	Label           string
	Title           string
	ClassName       string
	AscendClassName string
	TreeVersion     string
	VersionLabel    string
}

type TreeSpecPoints struct {
	Used       int
	AscendUsed int
	Sockets    int
}

//pob:block:src/GameVersions.lua:19-21,src/GameVersions.lua:145-147
func treeVersionDisplay(v pob.TreeVersion) string {
	parts := strings.SplitN(string(v), "_", 3)
	if len(parts) < 2 {
		return string(v)
	}
	out := parts[0] + "." + parts[1]
	if len(parts) == 3 {
		out += " (" + strings.ReplaceAll(parts[2], "_", " ") + ")"
	}
	return out
}

//pob:block:src/Classes/PassiveSpec.lua:824-835,src/Classes/PassiveSpec.lua:279-286
func specAllocCounts(tree *data.Tree, spec pob.Spec) (int, int) {
	selections := make(map[int64]int64)
	for _, m := range masteryEffectsRegex.FindAllStringSubmatch(spec.MasteryEffects, -1) {
		node, errNode := strconv.ParseInt(m[1], 10, 64)
		effect, errEffect := strconv.ParseInt(m[2], 10, 64)
		if errNode == nil && errEffect == nil {
			selections[node] = effect
		}
	}
	used, ascUsed := 0, 0
	seen := make(map[int64]bool)
	for _, id := range pob.SpecNodes(spec) {
		node, ok := tree.Nodes[strconv.FormatInt(id, 10)]
		if seen[id] || !ok {
			continue
		}
		seen[id] = true
		if isMasteryNode(node) {
			if _, ok := selections[id]; !ok {
				continue
			}
		}
		if isClassStartNode(node) || isAscendClassStartNode(node) {
			continue
		}
		if node.AscendancyName != nil && *node.AscendancyName != "" {
			if node.IsMultipleChoiceOption == nil || !*node.IsMultipleChoiceOption {
				ascUsed++
			}
		} else {
			used++
		}
	}
	return used, ascUsed
}

//pob:block:src/Classes/PassiveSpec.lua:836-838
func specSocketCount(tree *data.Tree, spec pob.Spec) int {
	n := 0
	seen := make(map[int64]bool)
	for _, id := range pob.SpecNodes(spec) {
		node, ok := tree.Nodes[strconv.FormatInt(id, 10)]
		if !seen[id] && ok && node.IsJewelSocket != nil && *node.IsJewelSocket {
			n++
		}
		seen[id] = true
	}
	return n
}

//pob:block:src/Classes/TreeTab.lua:486-492,src/Classes/PassiveSpecListControl.lua:72-80
func TreeSpecs(build *pob.PathOfBuilding) []TreeSpecRow {
	rows := make([]TreeSpecRow, 0, len(build.Tree.Specs))
	for _, spec := range build.Tree.Specs {
		title := pob.SpecTitle(spec)
		label := title
		if spec.TreeVersion != data.LatestTreeVersion {
			label = "[" + treeVersionDisplay(spec.TreeVersion) + "] " + title
		}
		rows = append(rows, TreeSpecRow{
			Label:           label,
			Title:           title,
			ClassName:       string(pob.ClassNameByID[spec.ClassID]),
			AscendClassName: string(pob.AscendancyNameOf(spec.ClassID, spec.AscendClassID)),
			TreeVersion:     string(spec.TreeVersion),
			VersionLabel:    treeVersionDisplay(spec.TreeVersion),
		})
	}
	return rows
}

//pob:block:src/Classes/PassiveSpecListControl.lua:74
func TreeSpecsPoints(build *pob.PathOfBuilding) []TreeSpecPoints {
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	out := make([]TreeSpecPoints, 0, len(build.Tree.Specs))
	for _, spec := range build.Tree.Specs {
		used, ascUsed := specAllocCounts(tree, spec)
		out = append(out, TreeSpecPoints{Used: used, AscendUsed: ascUsed, Sockets: specSocketCount(tree, spec)})
	}
	return out
}

//pob:block:src/Classes/PassiveSpec.lua:603-613
func classAscendancyStartNodeID(tree *data.Tree, classID pob.ClassID, name pob.AscendancyName) string {
	if int(classID) < 0 || int(classID) >= len(tree.Classes) {
		return ""
	}
	for _, a := range tree.Classes[classID].Ascendancies {
		if a.Name == name {
			return ascendancyStartNodeID(tree, a.ID)
		}
	}
	return ""
}

//pob:block:src/Classes/PassiveTree.lua:527-530
func ascendancyStartNodeID(tree *data.Tree, name pob.AscendancyName) string {
	for id, node := range tree.Nodes {
		if isAscendClassStartNode(node) && node.AscendancyName != nil && *node.AscendancyName == string(name) {
			return id
		}
	}
	return ""
}

//pob:block:src/Classes/PassiveSpecListControl.lua:36-41,src/Classes/PassiveSpec.lua:577-580,src/Classes/PassiveSpec.lua:608-613
func NewTreeSpec(build *pob.PathOfBuilding, title string) {
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	var classID pob.ClassID
	var ascendClassID pob.AscendancyID
	var secondary *int
	if i := build.ActiveSpecIndex(); i >= 0 {
		cur := build.Tree.Specs[i]
		classID, ascendClassID, secondary = cur.ClassID, cur.AscendClassID, cur.SecondaryAscendClassID
	}
	nodes := []string{}
	if id := classStartNodeID(tree, classID); id != "" {
		nodes = append(nodes, id)
	}
	if name := pob.AscendancyNameOf(classID, ascendClassID); name != "None" {
		if id := classAscendancyStartNodeID(tree, classID, name); id != "" {
			nodes = append(nodes, id)
		}
	}
	two := 2
	build.AddSpec(pob.Spec{
		ClassID:                  classID,
		AscendClassID:            ascendClassID,
		SecondaryAscendClassID:   secondary,
		ClusterHashFormatVersion: &two,
		Title:                    &title,
		TreeVersion:              data.LatestTreeVersion,
		NodesAttr:                strings.Join(nodes, ","),
		Sockets:                  []pob.Socket{},
		Overrides:                []pob.Override{},
	})
}

//pob:block:src/Modules/Build.lua:1327
func CurrentGameVersionLabel() string {
	return treeVersionDisplay(data.LatestTreeVersion)
}
