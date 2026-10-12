package calculator

import (
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

type ClassDecision int

const (
	ClassAsk ClassDecision = iota
	ClassContinue
	ClassConnect
)

type ClassChange struct {
	Changed      bool
	NeedsConfirm bool
	ClassName    string
	Handled      bool
}

func specTree() *data.Tree {
	return data.TreeVersions[data.LatestTreeVersion].Tree()
}

func nodeKey(id int64) string {
	return strconv.FormatInt(id, 10)
}

func allocSet(build *pob.PathOfBuilding, tree *data.Tree) map[int64]bool {
	alloc := make(map[int64]bool, len(build.Build.PassiveNodes)+2)
	for _, id := range build.Build.PassiveNodes {
		alloc[id] = true
	}
	for _, id := range startNodeIDs(build, tree) {
		alloc[id] = true
	}
	return alloc
}

//pob:block:src/Classes/PassiveSpec.lua:565,src/Classes/PassiveSpec.lua:590-591,src/Classes/PassiveSpec.lua:626-628
func startNodeIDs(build *pob.PathOfBuilding, tree *data.Tree) []int64 {
	classID, ok := pob.ClassIDByName(build.Build.ClassName)
	if !ok {
		return nil
	}
	out := make([]int64, 0, 2)
	if id, err := strconv.ParseInt(classStartNodeID(tree, classID), 10, 64); err == nil {
		out = append(out, id)
	}
	if name := build.Build.AscendClassName; name != "" && name != "None" {
		if id, err := strconv.ParseInt(classAscendancyStartNodeID(tree, classID, name), 10, 64); err == nil {
			out = append(out, id)
		}
	}
	if i := build.ActiveSpecIndex(); i >= 0 {
		if sec := build.Tree.Specs[i].SecondaryAscendClassID; sec != nil && *sec > 0 && *sec <= len(tree.AlternateAscendancies) &&
			!legacyAlternateAscendancies[string(tree.AlternateAscendancies[*sec-1].ID)] {
			if id, err := strconv.ParseInt(ascendancyStartNodeID(tree, tree.AlternateAscendancies[*sec-1].ID), 10, 64); err == nil {
				out = append(out, id)
			}
		}
	}
	return out
}

func walksOutward(from, to data.Node) bool {
	return (from.AscendancyName != nil && *from.AscendancyName != "") || to.AscendancyName == nil || *to.AscendancyName == ""
}

//pob:block:src/Classes/PassiveSpec.lua:846-871
func connectedToStart(tree *data.Tree, alloc map[int64]bool, starts []int64) map[int64]bool {
	seen := make(map[int64]bool, len(alloc))
	queue := make([]int64, 0, len(alloc))
	for _, s := range starts {
		if alloc[s] && !seen[s] {
			seen[s] = true
			queue = append(queue, s)
		}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		node, ok := tree.Nodes[nodeKey(id)]
		if !ok || isMasteryNode(node) {
			continue
		}
		for _, otherKey := range linkedNodeIDs(node) {
			other, ok := tree.Nodes[otherKey]
			if !ok || other.Skill == nil {
				continue
			}
			oid := *other.Skill
			if !alloc[oid] || seen[oid] || !walksOutward(node, other) {
				continue
			}
			seen[oid] = true
			queue = append(queue, oid)
		}
	}
	return seen
}

//pob:block:src/Classes/PassiveSpec.lua:822-835
func CountAllocNodes(build *pob.PathOfBuilding) int {
	tree := specTree()
	used := 0
	for id := range allocSet(build, tree) {
		node, ok := tree.Nodes[nodeKey(id)]
		if ok && (isClassStartNode(node) || isAscendClassStartNode(node) || (node.AscendancyName != nil && *node.AscendancyName != "")) {
			continue
		}
		used++
	}
	return used
}

//pob:src/Classes/PassiveSpec.lua:657-680
func IsClassConnected(build *pob.PathOfBuilding, classID pob.ClassID) bool {
	tree := specTree()
	start, ok := tree.Nodes[classStartNodeID(tree, classID)]
	if !ok {
		return false
	}
	alloc := allocSet(build, tree)
	var current []int64
	if cur, ok := pob.ClassIDByName(build.Build.ClassName); ok {
		if id, err := strconv.ParseInt(classStartNodeID(tree, cur), 10, 64); err == nil {
			current = append(current, id)
		}
	}
	reached := connectedToStart(tree, alloc, current)
	for _, key := range linkedNodeIDs(start) {
		if other, ok := tree.Nodes[key]; ok && other.Skill != nil && alloc[*other.Skill] && reached[*other.Skill] {
			return true
		}
	}
	return false
}

//pob:block:src/Classes/PassiveSpec.lua:1897-1910
func clusterAnchor(tree *data.Tree, id int64) (int64, bool) {
	if node, ok := tree.Nodes[nodeKey(id)]; ok {
		if node.ExpansionJewel == nil || node.ExpansionJewel.Parent == nil {
			return 0, false
		}
		for node.ExpansionJewel != nil && node.ExpansionJewel.Parent != nil {
			parent, ok := tree.Nodes[*node.ExpansionJewel.Parent]
			if !ok || parent.Skill == nil {
				return 0, false
			}
			node = parent
		}
		return *node.Skill, true
	}
	if id < 0x10000 {
		return 0, false
	}
	large := (id >> 6) & 7
	for _, node := range tree.Nodes {
		if j := node.ExpansionJewel; j != nil && j.Parent == nil && j.Size == 2 && j.Index == large && node.Skill != nil {
			return *node.Skill, true
		}
	}
	return 0, false
}

//pob:block:src/Classes/PassiveSpec.lua:1521-1535,src/Classes/PassiveSpec.lua:2397-2407
func leapHeld(build *pob.PathOfBuilding, tree *data.Tree, alloc map[int64]bool) map[int64]bool {
	i := build.ActiveSpecIndex()
	if i < 0 {
		return map[int64]bool{}
	}
	texts := make(map[int]string, len(build.Items.Items))
	for _, it := range build.Items.Items {
		texts[it.ID] = it.Text
	}
	return leapHeldBy(tree, build.Tree.Specs[i].Sockets, texts, alloc)
}

func leapHeldBy(tree *data.Tree, sockets []pob.Socket, texts map[int]string, alloc map[int64]bool) map[int64]bool {
	held := map[int64]bool{}
	radius := tree.JewelRadiusNodes()
	for _, socket := range sockets {
		text, ok := texts[socket.ItemID]
		if socket.ItemID == 0 || !ok || !alloc[int64(socket.NodeID)] {
			continue
		}
		item := PeekItem(text)
		index, hasRadius := itemJewelRadiusIndex(item)
		if !hasRadius {
			continue
		}
		leap := false
		keystones := map[string]bool{}
		for _, m := range item.Mods {
			if m.Name() != "JewelData" {
				continue
			}
			value, ok := m.Value().List().(mod.JewelData)
			if !ok {
				continue
			}
			switch value.Key {
			case "intuitiveLeapLike":
				leap = true
			case "impossibleEscapeKeystone":
				if name, ok := value.Value.(string); ok {
					keystones[strings.ToLower(name)] = true
				}
			}
		}
		if leap {
			if per := radius[int64(socket.NodeID)]; index < len(per) {
				for _, id := range per[index] {
					held[id] = true
				}
			}
		}
		if len(keystones) > 0 {
			for key, node := range tree.Nodes {
				if node.Name == nil || node.IsKeystone == nil || !*node.IsKeystone || !keystones[strings.ToLower(*node.Name)] {
					continue
				}
				if id, err := strconv.ParseInt(key, 10, 64); err == nil {
					for n := range tree.NodesInRadiusOf(id, index) {
						held[n] = true
					}
				}
			}
		}
	}
	return held
}

//pob:block:src/Classes/PassiveSpec.lua:856-857
func allocatedStarts(tree *data.Tree, alloc map[int64]bool) []int64 {
	var out []int64
	for id := range alloc {
		if node, ok := tree.Nodes[nodeKey(id)]; ok && (isClassStartNode(node) || isAscendClassStartNode(node)) {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

//pob:block:src/Classes/PassiveSpec.lua:1516-1550
func pruneOrphans(build *pob.PathOfBuilding) {
	pruneOrphansOn(build, specTree())
}

func pruneOrphansOn(build *pob.PathOfBuilding, tree *data.Tree) {
	alloc := allocSet(build, tree)
	reached := connectedToStart(tree, alloc, allocatedStarts(tree, alloc))
	held := leapHeld(build, tree, alloc)
	var orphans []int64
	for _, id := range build.Build.PassiveNodes {
		at := id
		if anchor, ok := clusterAnchor(tree, id); ok {
			at = anchor
		} else if _, known := tree.Nodes[nodeKey(id)]; !known {
			continue
		}
		if !reached[at] && !held[at] {
			orphans = append(orphans, id)
		}
	}
	if len(orphans) > 0 {
		build.DeallocateNodes(orphans)
	}
}

func setStartNodes(build *pob.PathOfBuilding, tree *data.Tree, old []int64) {
	if len(old) > 0 {
		build.DeallocateNodes(old)
	}
	var add []int64
	for _, id := range startNodeIDs(build, tree) {
		if !slices.Contains(build.Build.PassiveNodes, id) {
			add = append(add, id)
		}
	}
	if len(add) > 0 {
		build.AllocateNodes(add)
	}
}

//pob:src/Classes/PassiveSpec.lua:562-585
func SelectClass(build *pob.PathOfBuilding, className pob.ClassName) {
	tree := specTree()
	old := startNodeIDs(build, tree)
	build.SetClass(className)
	build.SetAscendancy("None")
	setStartNodes(build, tree, old)
	pruneOrphans(build)
}

//pob:src/Classes/PassiveSpec.lua:599-617
func SelectAscendClass(build *pob.PathOfBuilding, name pob.AscendancyName) {
	tree := specTree()
	old := startNodeIDs(build, tree)
	build.SetAscendancy(name)
	setStartNodes(build, tree, old)
	pruneOrphans(build)
}

//pob:src/Classes/PassiveSpec.lua:619-655
func SelectSecondaryAscendClass(build *pob.PathOfBuilding, ascendClassID int) {
	tree := specTree()
	i := build.ActiveSpecIndex()
	if len(tree.AlternateAscendancies) == 0 || i < 0 {
		return
	}
	old := startNodeIDs(build, tree)
	build.Tree.Specs[i].SecondaryAscendClassID = &ascendClassID
	setStartNodes(build, tree, old)
	pruneOrphans(build)
}

func bloodlineIndex(tree *data.Tree, id string) int {
	for i, a := range tree.AlternateAscendancies {
		if string(a.ID) == id && !legacyAlternateAscendancies[id] {
			return i + 1
		}
	}
	return 0
}

func isMainTreeNode(node data.Node) bool {
	return (node.IsProxy == nil || !*node.IsProxy) &&
		(node.AscendancyName == nil || *node.AscendancyName == "") &&
		!isClassStartNode(node) && !isAscendClassStartNode(node)
}

//pob:src/Classes/PassiveSpec.lua:682-754
func ConnectToClass(build *pob.PathOfBuilding, classID pob.ClassID) bool {
	tree := specTree()
	targetKey := classStartNodeID(tree, classID)
	target, ok := tree.Nodes[targetKey]
	if !ok || target.Skill == nil {
		return false
	}
	alloc := allocSet(build, tree)
	connected := connectedToStart(tree, alloc, startNodeIDs(build, tree))

	visited := map[string]bool{targetKey: true}
	prev := map[string]string{}
	queue := []string{targetKey}
	found := ""
	for len(queue) > 0 && found == "" {
		key := queue[0]
		queue = queue[1:]
		node := tree.Nodes[key]
		if key != targetKey && node.Skill != nil && alloc[*node.Skill] && connected[*node.Skill] &&
			!isClassStartNode(node) && !isAscendClassStartNode(node) {
			found = key
			break
		}
		for _, linked := range linkedNodeIDs(node) {
			other, ok := tree.Nodes[linked]
			if ok && isMainTreeNode(other) && !visited[linked] {
				visited[linked] = true
				prev[linked] = key
				queue = append(queue, linked)
			}
		}
	}
	if found == "" {
		return false
	}

	var path []int64
	for cur := prev[found]; cur != "" && cur != targetKey; cur = prev[cur] {
		if n := tree.Nodes[cur]; n.Skill != nil && !alloc[*n.Skill] {
			path = append(path, *n.Skill)
		}
	}
	if len(path) > 0 {
		build.AllocateNodes(path)
	}
	return true
}

//pob:block:src/Classes/PassiveSpec.lua:770-800
func allocPathTo(build *pob.PathOfBuilding, nodeID int64) {
	tree := specTree()
	alloc := allocSet(build, tree)
	if alloc[nodeID] {
		return
	}
	active := make([]int64, 0, len(alloc))
	for id := range alloc {
		active = append(active, id)
	}
	slices.Sort(active)
	hops := data.TreeVersions[data.LatestTreeVersion].CalculateAllocationPaths(active, nil)
	var path []int64
	for cur := nodeID; !alloc[cur]; {
		next, ok := hops[cur]
		if !ok || next == -1 {
			return
		}
		path = append(path, cur)
		cur = next
	}
	build.AllocateNodes(path)
}

//pob:block:src/Modules/Build.lua:256-276
func ChangeClass(build *pob.PathOfBuilding, className pob.ClassName, decision ClassDecision) ClassChange {
	classID, ok := pob.ClassIDByName(className)
	if !ok || className == build.Build.ClassName {
		return ClassChange{}
	}
	switch {
	case decision == ClassConnect:
		if !ConnectToClass(build, classID) {
			return ClassChange{}
		}
	case decision == ClassContinue:
	case CountAllocNodes(build) == 0 || IsClassConnected(build, classID):
	default:
		return ClassChange{NeedsConfirm: true, ClassName: string(className)}
	}
	SelectClass(build, className)
	return ClassChange{Changed: true}
}

func ascendancyOwner(tree *data.Tree, ascendancyID string) (pob.ClassID, pob.AscendancyName, bool) {
	for classID, class := range tree.Classes {
		for _, a := range class.Ascendancies {
			if string(a.ID) == ascendancyID {
				return pob.ClassID(classID), a.Name, true
			}
		}
	}
	return 0, "", false
}

//pob:block:src/Classes/PassiveTreeView.lua:390-497
func ClickAscendancyNode(build *pob.PathOfBuilding, nodeID int64, decision ClassDecision) ClassChange {
	tree := specTree()
	node, ok := tree.Nodes[nodeKey(nodeID)]
	if !ok || node.AscendancyName == nil || *node.AscendancyName == "" || allocSet(build, tree)[nodeID] {
		return ClassChange{}
	}
	if node.IsBloodline != nil && *node.IsBloodline {
		cur := 0
		if i := build.ActiveSpecIndex(); i >= 0 && build.Tree.Specs[i].SecondaryAscendClassID != nil {
			cur = *build.Tree.Specs[i].SecondaryAscendClassID
		}
		target := bloodlineIndex(tree, *node.AscendancyName)
		if target == 0 || target == cur {
			return ClassChange{}
		}
		SelectSecondaryAscendClass(build, target)
		allocPathTo(build, nodeID)
		return ClassChange{Changed: true, Handled: true}
	}
	classID, name, ok := ascendancyOwner(tree, *node.AscendancyName)
	if !ok {
		return ClassChange{}
	}
	if name == build.Build.AscendClassName {
		return ClassChange{}
	}
	if pob.ClassNameByID[classID] == build.Build.ClassName {
		SelectAscendClass(build, name)
		allocPathTo(build, nodeID)
		return ClassChange{Changed: true, Handled: true}
	}
	className := pob.ClassNameByID[classID]
	switch {
	case decision == ClassConnect:
		if !ConnectToClass(build, classID) {
			return ClassChange{Handled: true}
		}
	case decision == ClassContinue:
	case CountAllocNodes(build) == 0 || IsClassConnected(build, classID):
	default:
		return ClassChange{NeedsConfirm: true, ClassName: string(className), Handled: true}
	}
	SelectClass(build, className)
	SelectAscendClass(build, name)
	allocPathTo(build, nodeID)
	return ClassChange{Changed: true, Handled: true}
}

//pob:block:src/Classes/ItemsTab.lua:4309-4337,src/Classes/PassiveSpec.lua:1516-1550
func socketOrphans(build *pob.PathOfBuilding, socketID int, item *Item) []int64 {
	i := build.ActiveSpecIndex()
	if i < 0 {
		return nil
	}
	tree := specTree()
	texts := make(map[int]string, len(build.Items.Items))
	for _, it := range build.Items.Items {
		texts[it.ID] = it.Text
	}
	sockets, texts := socketWith(build.Tree.Specs[i].Sockets, texts, socketID, item)
	alloc := allocSet(build, tree)
	reached := connectedToStart(tree, alloc, allocatedStarts(tree, alloc))
	held := leapHeldBy(tree, sockets, texts, alloc)
	var orphans []int64
	for _, id := range build.Build.PassiveNodes {
		at := id
		if anchor, ok := clusterAnchor(tree, id); ok {
			at = anchor
		} else if _, known := tree.Nodes[nodeKey(id)]; !known {
			continue
		}
		if !reached[at] && !held[at] {
			orphans = append(orphans, id)
		}
	}
	return orphans
}

//pob:block:src/Classes/PassiveSpec.lua:1136-1172
func LeapHeldNodes(build *pob.PathOfBuilding) []int64 {
	tree := specTree()
	held := leapHeld(build, tree, allocSet(build, tree))
	out := make([]int64, 0, len(held))
	for id := range held {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}
