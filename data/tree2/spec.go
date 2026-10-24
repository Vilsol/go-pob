package tree2

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SpecNode is PoB's per-spec node: setmetatable({linked, power}, treeNode) plus what ReplaceNode writes.
type SpecNode struct {
	*Node
	Dn        string
	Sd        []string
	Linked    []*SpecNode
	Alloc     bool
	AllocMode int

	Depends                     []*SpecNode
	IntuitiveLeapLikesAffecting []*SpecNode
	Visited                     bool
	ConnectedToStart            bool
	IsGrantedPassive            bool
	// IsFreeAllocate shadows the tree node's flag, which SetGrantedPassiveNodes overwrites per spec.
	IsFreeAllocate bool
}

// Item is the part of build.itemsTab.items[id] the spec reads.
type Item struct {
	JewelRadiusIndex int
	JewelData        *JewelData
}

// JewelData is item.jewelData as Item:BuildModList fills it from the jewel's JewelData,
// AlternateClassStart and FromNothingKeystones mods.
type JewelData struct {
	IntuitiveLeapLike    bool
	FromNothingKeystone  string
	FromNothingKeystones map[string]bool
	AlternateClassStart  string
	LimitDisabled        bool
}

// NodeRadius is an AllocateFromNodeRadius LIST mod's value.
type NodeRadius struct {
	From        string
	RadiusIndex int
	To          []string
}

// SpecInput carries the <Spec> element's attributes and children as PoB reads them; the XML decoding
// itself belongs to the build-file package.
type SpecInput struct {
	Title                  string
	TreeVersion            string
	Nodes                  *string
	ClassID                *string
	AscendClassID          *string
	ClassInternalID        *string
	AscendancyInternalID   *string
	SecondaryAscendClassID *string
	MasteryEffects         *string
	URL                    *string
	// WeaponSets maps the N of a <WeaponSetN nodes="..."> child to its nodes attribute.
	WeaponSets map[int]string
	Sockets    []SocketInput
	// AttributeOverrides holds each <Overrides><AttributeOverride> child, in document order.
	AttributeOverrides []AttributeOverrideInput
}

type SocketInput struct {
	NodeID string
	ItemID string
}

type AttributeOverrideInput struct {
	StrNodes string
	DexNodes string
	IntNodes string
}

type PassiveSpec struct {
	Tree        *PassiveTree
	TreeVersion string
	Title       string

	Nodes              map[int]*SpecNode
	AllocNodes         map[int]*SpecNode
	AllocSubgraphNodes []int
	AllocExtendedNodes []int
	Jewels             map[int]int
	MasterySelections  map[int]int
	HashOverrides      map[int]*SpecNode
	SwitchableNodes    map[int]*SpecNode
	IgnoredNodes       map[int]*SpecNode

	AllocMode int

	// AllocateFromNodeRadius stands in for node.modList:List(nil, "AllocateFromNodeRadius") on an
	// allocated ascendancy node; the mod parse is the calc's. Nil means no node carries the mod.
	AllocateFromNodeRadius func(node *SpecNode) []NodeRadius
	intuitiveLeapLikeNodes []NodeRadius
	// Items stands in for build.itemsTab.items; nil means the build has no items.
	Items func(itemID int) *Item

	CurClassID             int
	CurClass               *Class
	CurClassName           string
	CurAscendClassID       int
	CurAscendClass         *Ascendancy
	CurAscendClassName     string
	CurAscendClassBaseName string

	hasCurClass       bool
	hasCurAscendClass bool
}

var (
	digitsRE  = regexp.MustCompile(`\d+`)
	masteryRE = regexp.MustCompile(`\{(\d+),(\d+)\}`)
	//pob2:block:src/Classes/PassiveSpec.lua:19-23
	legacyClassIDMap = map[string]map[int]int{
		"0_1": {0: 2, 1: 6, 2: 9, 3: 1, 4: 7, 5: 10},
		"0_2": {0: 2, 1: 8, 2: 6, 3: 9, 4: 1, 5: 7, 6: 10},
		"0_3": {0: 2, 1: 8, 2: 6, 3: 9, 4: 1, 5: 7, 6: 10},
	}
)

func digits(s string) []int {
	var out []int
	for _, m := range digitsRE.FindAllString(s, -1) {
		v, err := strconv.Atoi(m)
		if err == nil {
			out = append(out, v)
		}
	}
	return out
}

// NewPassiveSpec is PassiveSpec(build, treeVersion) without the build: Init, then SelectClass(DexClass).
//
//pob2:src/Classes/PassiveSpec.lua:33-43
func NewPassiveSpec(tree *PassiveTree) *PassiveSpec {
	s := &PassiveSpec{}
	s.Init(tree)
	s.SelectClass(tree.Constants.Classes["DexClass"])
	return s
}

// Init skips the convert path: go-pob loads a build on the tree version it names.
//
//pob2:src/Classes/PassiveSpec.lua:45-115
func (s *PassiveSpec) Init(tree *PassiveTree) {
	s.TreeVersion = tree.TreeVersion
	s.Tree = tree
	s.IgnoredNodes = map[int]*SpecNode{}
	s.AllocMode = 0

	s.Nodes = map[int]*SpecNode{}
	for _, id := range tree.sortedNodeIDs() {
		treeNode := tree.Nodes[id]
		// tree.json 0_5 carries no expansionJewel, so only the proxy half of the filter can bite.
		if treeNode.Group != nil && !treeNode.IsProxy && !treeNode.Group.IsProxy {
			s.Nodes[treeNode.ID] = &SpecNode{Node: treeNode, Dn: treeNode.Dn, Sd: treeNode.Sd, IsFreeAllocate: treeNode.IsFreeAllocate}
		}
	}
	for _, id := range sortedKeys(s.Nodes) {
		node := s.Nodes[id]
		node.AllocMode = 0
		for _, otherID := range node.LinkedID {
			// t_insert(t, nil) is a no-op, so a link to a filtered node is dropped.
			if other, ok := s.Nodes[otherID]; ok {
				node.Linked = append(node.Linked, other)
			}
		}
	}

	s.AllocNodes = map[int]*SpecNode{}
	s.AllocSubgraphNodes = nil
	s.AllocExtendedNodes = nil
	s.Jewels = map[int]int{}
	s.MasterySelections = map[int]int{}
	s.HashOverrides = map[int]*SpecNode{}
}

// Load reads a <Spec>; a socket keeps its jewel only when s.Items knows the item.
// Notes and the legacy URL decode are not ported.
//
//pob2:src/Classes/PassiveSpec.lua:117-250
func (s *PassiveSpec) Load(in SpecInput) error {
	s.Title = in.Title
	weaponSets := map[int]int{}
	for _, child := range in.Sockets {
		if child.NodeID == "" {
			return fmt.Errorf("tree2: 'Socket' element missing 'nodeId' attribute")
		}
		if child.ItemID == "" {
			return fmt.Errorf("tree2: 'Socket' element missing 'itemId' attribute")
		}
		jewelIDNum, err1 := strconv.Atoi(child.ItemID)
		nodeIDNum, err2 := strconv.Atoi(child.NodeID)
		if err1 == nil && err2 == nil && jewelIDNum > 0 && s.item(jewelIDNum) != nil {
			s.Jewels[nodeIDNum] = jewelIDNum
		}
	}
	for _, set := range sortedKeys(in.WeaponSets) {
		for _, nodeID := range digits(in.WeaponSets[set]) {
			weaponSets[nodeID] = set
		}
	}
	if in.Nodes == nil {
		if in.URL != nil {
			return fmt.Errorf("tree2: a <Spec> with only a URL is not supported")
		}
		return nil
	}

	classID, ascendClassID := -1, -1
	if in.ClassID != nil {
		classID = luaToNumber(*in.ClassID, -1)
	}
	if in.AscendClassID != nil {
		ascendClassID = luaToNumber(*in.AscendClassID, -1)
	}
	if in.ClassInternalID != nil {
		if mapped, ok := s.Tree.ClassIntegerIDMap[luaToNumber(*in.ClassInternalID, -1)]; ok {
			classID = mapped
		}
	} else if legacy, ok := legacyClassIDMap[s.TreeVersion]; classID != -1 && ok {
		if mapped, ok := legacy[classID]; ok {
			classID = mapped
		}
	}
	if in.AscendancyInternalID != nil {
		if *in.AscendancyInternalID == "" {
			ascendClassID = 0
		} else if ref, ok := s.Tree.InternalAscendNameMap[*in.AscendancyInternalID]; ok {
			ascendClassID = ref.AscendClassID
		}
	}
	if classID == -1 {
		return fmt.Errorf("tree2: 'Spec' element missing 'classId' / 'classInternalId' attribute")
	}
	if ascendClassID == -1 {
		return fmt.Errorf("tree2: 'Spec' element missing 'ascendClassId' / 'ascendancyInternalId' attribute")
	}
	if _, ok := s.Tree.Classes[classID]; !ok {
		return fmt.Errorf("tree2: class %d is not on tree %s", classID, s.TreeVersion)
	}

	hashList := digits(*in.Nodes)
	masteryEffects := map[int]int{}
	if in.MasteryEffects != nil {
		for _, m := range masteryRE.FindAllStringSubmatch(*in.MasteryEffects, -1) {
			mastery, _ := strconv.Atoi(m[1])
			effect, _ := strconv.Atoi(m[2])
			masteryEffects[mastery] = effect
		}
	}
	for _, child := range in.AttributeOverrides {
		for _, id := range digits(child.StrNodes) {
			s.SwitchAttributeNode(id, 1)
		}
		for _, id := range digits(child.DexNodes) {
			s.SwitchAttributeNode(id, 2)
		}
		for _, id := range digits(child.IntNodes) {
			s.SwitchAttributeNode(id, 3)
		}
	}
	secondary := 0
	if in.SecondaryAscendClassID != nil {
		secondary = luaToNumber(*in.SecondaryAscendClassID, 0)
	}
	overrides := make(map[int]*SpecNode, len(s.HashOverrides))
	for id, n := range s.HashOverrides {
		overrides[id] = n
	}
	s.ImportFromNodeList(classID, ascendClassID, secondary, hashList, weaponSets, overrides, masteryEffects)
	return nil
}

// luaToNumber is tonumber(x) with a stand-in for its nil; secondaryAscendClassId="nil" parses to nil.
func luaToNumber(s string, ifNil int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return ifNil
	}
	return v
}

// ImportFromNodeList takes the class by id; the className branch serves the URL importer, which is not ported.
//
//pob2:src/Classes/PassiveSpec.lua:386-440
func (s *PassiveSpec) ImportFromNodeList(classID, ascendClassID, secondaryAscendClassID int, hashList []int, weaponSets map[int]int, hashOverrides map[int]*SpecNode, masteryEffects map[int]int) {
	if hashOverrides == nil {
		hashOverrides = map[int]*SpecNode{}
	}
	s.ResetNodes()
	s.SelectClass(classID)
	s.SelectAscendClass(ascendClassID)
	s.SelectSecondaryAscendClass(secondaryAscendClassID)
	s.HashOverrides = hashOverrides
	s.MasterySelections = map[int]int{}
	for mastery, effect := range masteryEffects {
		if effect < 65536 {
			s.MasterySelections[mastery] = effect
		}
	}
	for _, id := range sortedKeys(hashOverrides) {
		if node, ok := s.Nodes[id]; ok {
			s.ReplaceNode(node, hashOverrides[id].Dn, hashOverrides[id].Sd)
		}
	}
	for _, id := range hashList {
		if node, ok := s.Nodes[id]; ok {
			if _, sel := s.MasterySelections[id]; node.Type != "Mastery" || sel {
				node.Alloc = true
				node.AllocMode = weaponSets[id]
				s.AllocNodes[id] = node
			}
		} else {
			s.AllocSubgraphNodes = append(s.AllocSubgraphNodes, id)
		}
	}
	for _, id := range s.AllocExtendedNodes {
		if node, ok := s.Nodes[id]; ok {
			node.Alloc = true
			node.AllocMode = weaponSets[id]
			s.AllocNodes[id] = node
		}
	}
	s.BuildAllDependsAndPaths()
}

//pob2:src/Classes/PassiveSpec.lua:689-712
func (s *PassiveSpec) SelectClass(classID int) {
	if s.hasCurClass {
		oldStartNodeID := s.CurClass.StartNodeID
		if n, ok := s.Nodes[oldStartNodeID]; ok {
			n.Alloc = false
		}
		delete(s.AllocNodes, oldStartNodeID)
	}
	s.ResetAscendClass()

	s.CurClassID = classID
	class := s.Tree.Classes[classID]
	s.CurClass = class
	s.CurClassName = class.Name
	s.hasCurClass = true

	startNode := s.Nodes[class.StartNodeID]
	startNode.Alloc = true
	s.AllocNodes[startNode.ID] = startNode

	s.SelectAscendClass(0)
}

//pob2:src/Classes/PassiveSpec.lua:714-724
func (s *PassiveSpec) ResetAscendClass() {
	if s.hasCurAscendClass {
		ascendClass := s.ascendClassOrNone(s.CurAscendClassID)
		if oldStartNodeID := ascendClass.StartNodeID; oldStartNodeID != 0 {
			if n, ok := s.Nodes[oldStartNodeID]; ok {
				n.Alloc = false
			}
			delete(s.AllocNodes, oldStartNodeID)
		}
	}
}

func (s *PassiveSpec) ascendClassOrNone(id int) *Ascendancy {
	if id >= 0 && id < len(s.CurClass.Classes) {
		return s.CurClass.Classes[id]
	}
	return s.CurClass.Classes[0]
}

//pob2:src/Classes/PassiveSpec.lua:726-744
func (s *PassiveSpec) SelectAscendClass(ascendClassID int) {
	s.ResetAscendClass()

	s.CurAscendClassID = ascendClassID
	s.hasCurAscendClass = true
	ascendClass := s.ascendClassOrNone(ascendClassID)
	s.CurAscendClass = ascendClass
	s.CurAscendClassName = ascendClass.Name
	s.CurAscendClassBaseName = ascendClass.ID

	if ascendClass.StartNodeID != 0 {
		startNode := s.Nodes[ascendClass.StartNodeID]
		startNode.Alloc = true
		s.AllocNodes[startNode.ID] = startNode
	}

	s.BuildAllDependsAndPaths()
}

// SelectSecondaryAscendClass returns at once: tree 0_5 has no alternate_ascendancies.
//
//pob2:src/Classes/PassiveSpec.lua:746-750
func (s *PassiveSpec) SelectSecondaryAscendClass(int) {}

//pob2:src/Classes/PassiveSpec.lua:884-892
func (s *PassiveSpec) ResetNodes() {
	for id, node := range s.Nodes {
		if node.Type != "ClassStart" && node.Type != "AscendClassStart" {
			node.Alloc = false
			delete(s.AllocNodes, id)
		}
	}
	s.MasterySelections = map[int]int{}
}

// BuildAllDependsAndPaths ports node replacement (switchable options by class or ascendancy, then
// attribute overrides), the dependency walk and orphan pruning, and the mastery/ignored deallocation.
// Everything a socketed jewel drives (intuitive-leap-like radius, From Nothing, conquest,
// alternate class starts) needs item jewel data and radius lists and is not ported, nor are paths.
//
//pob2:src/Classes/PassiveSpec.lua:1475-2078
func (s *PassiveSpec) BuildAllDependsAndPaths() {
	var visited []*SpecNode
	jewelIDs := sortedKeys(s.Jewels)
	//pob2:block:src/Classes/PassiveSpec.lua:1483-1494
	alternateClassStartNodes := map[int]*SpecNode{}
	for _, socketNodeID := range jewelIDs {
		if _, ok := s.AllocNodes[socketNodeID]; !ok {
			continue
		}
		item := s.GetJewel(s.Jewels[socketNodeID])
		if item == nil {
			continue
		}
		if startNodeID, ok := s.Tree.ClassStartNodeNameMap[item.JewelData.AlternateClassStart]; ok {
			if startNode, ok := s.Nodes[startNodeID]; ok {
				alternateClassStartNodes[startNode.ID] = startNode
			}
		}
	}
	//pob2:block:src/Classes/PassiveSpec.lua:1495-1506
	s.intuitiveLeapLikeNodes = s.intuitiveLeapLikeNodes[:0]
	if s.AllocateFromNodeRadius != nil {
		for _, id := range sortedKeys(s.AllocNodes) {
			if node := s.AllocNodes[id]; node.AscendancyName != "" {
				s.intuitiveLeapLikeNodes = append(s.intuitiveLeapLikeNodes, s.AllocateFromNodeRadius(node)...)
			}
		}
	}

	s.SwitchableNodes = map[int]*SpecNode{}
	ids := sortedKeys(s.Nodes)
	keystoneNames := sortedStringKeys(s.Tree.KeystoneMap)
	//pob2:block:src/Classes/PassiveSpec.lua:1509-1527
	for _, id := range ids {
		node := s.Nodes[id]
		node.Depends = node.Depends[:0]
		node.IntuitiveLeapLikesAffecting = node.IntuitiveLeapLikesAffecting[:0]
		treeNode, ok := s.Tree.Nodes[id]
		if ok {
			dn, sd := treeNode.Dn, treeNode.Sd
			if treeNode.IsSwitchable {
				replaceID := treeNode.ID
				if o, ok := treeNode.SwitchOptions[s.CurClassName]; ok {
					dn, sd, replaceID = o.Dn, o.Sd, o.ID
				} else if o, ok := treeNode.SwitchOptions[s.CurAscendClassName]; ok {
					dn, sd, replaceID = o.Dn, o.Sd, o.ID
				}
				s.SwitchableNodes[replaceID] = node
			}
			s.ReplaceNode(node, dn, sd)
		}
		if node.Type != "ClassStart" && node.Type != "Socket" && node.AscendancyName == "" {
			// conqueredBy (timeless jewels) is not ported.
			//pob2:block:src/Classes/PassiveSpec.lua:1530-1559
			for _, nodeID := range jewelIDs {
				itemID := s.Jewels[nodeID]
				item := s.item(itemID)
				if item == nil || item.JewelRadiusIndex == 0 || s.AllocNodes[nodeID] == nil || item.JewelData == nil || item.JewelData.LimitDisabled {
					continue
				}
				radiusIndex := item.JewelRadiusIndex
				if inRadius(s.Nodes[nodeID].Node, radiusIndex, node.ID) && itemID != 0 && item.JewelData.IntuitiveLeapLike {
					node.IntuitiveLeapLikesAffecting = append(node.IntuitiveLeapLikesAffecting, s.Nodes[nodeID])
				}
				if item.JewelData.FromNothingKeystone != "" {
					for _, keyName := range keystoneNames {
						keyNode := s.Tree.KeystoneMap[keyName]
						if item.JewelData.FromNothingKeystones[keyName] && keyNode.NodesInRadius != nil && inRadius(keyNode, radiusIndex, node.ID) {
							node.IntuitiveLeapLikesAffecting = append(node.IntuitiveLeapLikesAffecting, s.Nodes[nodeID])
						}
					}
				}
			}
			//pob2:block:src/Classes/PassiveSpec.lua:1560-1585
			for _, intuitiveLeapMap := range s.intuitiveLeapLikeNodes {
				if !containsString(intuitiveLeapMap.To, node.Type) || intuitiveLeapMap.From != "Keystone" {
					continue
				}
				// keystoneMap holds each keystone under its name and its lower-cased name, so a
				// keystone usually affects a node twice.
				for _, keyName := range keystoneNames {
					keyNode := s.Tree.KeystoneMap[keyName]
					if _, ok := s.AllocNodes[keyNode.ID]; ok && inRadius(keyNode, intuitiveLeapMap.RadiusIndex, id) {
						node.IntuitiveLeapLikesAffecting = append(node.IntuitiveLeapLikesAffecting, s.Nodes[keyNode.ID])
					}
				}
			}
		}
		//pob2:block:src/Classes/PassiveSpec.lua:1587-1589
		if node.Alloc {
			node.Depends = append(node.Depends, node)
		}
	}
	for _, id := range ids {
		node := s.Nodes[id]
		//pob2:block:src/Classes/PassiveSpec.lua:1595-1601
		if node.UnlockConstraint != nil {
			for _, nodeID := range node.UnlockConstraint.Nodes {
				if dep, ok := s.Nodes[nodeID]; ok {
					dep.Depends = append(dep.Depends, node)
				}
			}
		}
		//pob2:block:src/Classes/PassiveSpec.lua:1604-1606
		if o, ok := s.HashOverrides[id]; ok {
			s.ReplaceNode(node, o.Dn, o.Sd)
		}
	}
	//pob2:block:src/Classes/PassiveSpec.lua:1798-1838
	for _, id := range ids {
		node := s.Nodes[id]
		if ign, ok := s.IgnoredNodes[id]; ok && ign != nil && s.AllocNodes[id] != nil {
			node.Alloc = false
			delete(s.AllocNodes, id)
			delete(s.IgnoredNodes, id)
		} else if _, sel := s.MasterySelections[id]; node.Type == "Mastery" && sel {
			// tree 0_5 has no mastery nodes, so the effect lookup is not ported.
			node.Alloc = false
			delete(s.AllocNodes, id)
			delete(s.MasterySelections, id)
		}
	}

	// pairs(self.allocNodes) while pruning: a node pruned before its turn is skipped.
	//pob2:block:src/Classes/PassiveSpec.lua:1840-2009
	for _, id := range sortedKeys(s.AllocNodes) {
		node, ok := s.AllocNodes[id]
		if !ok {
			continue
		}
		node.Visited = true
		node.ConnectedToStart = false
		anyStartFound := node.Type == "ClassStart" || node.Type == "AscendClassStart" || node.IsFreeAllocate
		if node.IsFreeAllocate {
			node.ConnectedToStart = true
		}
		for _, other := range node.Linked {
			otherAlloc := other.Alloc || alternateClassStartNodes[other.ID] != nil
			if !otherAlloc || !s.CanPathThroughAllocMode(node.AllocMode, other) || containsNode(node.Depends, other) {
				continue
			}
			switch {
			case other.Type == "ClassStart" || other.Type == "AscendClassStart":
				anyStartFound = true
				node.ConnectedToStart = true
			case s.FindStartFromNode(other, &visited, false, node.AllocMode, alternateClassStartNodes):
				anyStartFound = true
				node.ConnectedToStart = true
				for _, n := range visited {
					n.Visited = false
				}
				visited = visited[:0]
			default:
				depIDs := map[int]bool{}
				for _, n := range visited {
					if len(n.IntuitiveLeapLikesAffecting) == 0 {
						depIDs[n.ID] = true
					}
				}
				for _, n := range visited {
					if n.Type == "Mastery" {
						otherPath := false
						allocatedLinkCount := 0
						for _, linkedNode := range n.Linked {
							if linkedNode.Alloc && s.CanPathThroughAllocMode(n.AllocMode, linkedNode) {
								allocatedLinkCount++
							}
						}
						if allocatedLinkCount > 1 {
							for _, linkedNode := range n.Linked {
								if linkedNode.Alloc && s.CanPathThroughAllocMode(n.AllocMode, linkedNode) && !depIDs[linkedNode.ID] {
									otherPath = true
								}
							}
						}
						if !otherPath {
							node.Depends = append(node.Depends, n)
						}
					} else if len(n.IntuitiveLeapLikesAffecting) == 0 {
						// potentialDeps, the intuitiveLeaps bookkeeping and their 2015-2050 resolution only
						// grow depends lists that DeallocNode walks; they never change what is allocated here.
						node.Depends = append(node.Depends, n)
					}
					n.Visited = false
				}
				visited = visited[:0]
			}
		}
		node.Visited = false
		if !anyStartFound {
			for _, depNode := range node.Depends {
				prune := true
				//pob2:block:src/Classes/PassiveSpec.lua:1956-1980
				for _, nodeID := range jewelIDs {
					if s.AllocNodes[nodeID] == nil {
						continue
					}
					itemID := s.Jewels[nodeID]
					item := s.item(itemID)
					if itemID == 0 || item == nil || item.JewelData == nil {
						continue
					}
					jd := item.JewelData
					if (jd.IntuitiveLeapLike && item.JewelRadiusIndex != 0 && inRadius(s.Nodes[nodeID].Node, item.JewelRadiusIndex, depNode.ID)) ||
						(jd.FromNothingKeystones != nil && s.NodeInKeystoneRadius(jd.FromNothingKeystones, depNode.ID, item.JewelRadiusIndex)) {
						prune = false
						break
					}
				}
				for _, intuitiveLeapMap := range s.intuitiveLeapLikeNodes {
					if !containsString(intuitiveLeapMap.To, depNode.Type) || intuitiveLeapMap.From != "Keystone" {
						continue
					}
					// Any keystone with the node in radius holds it, allocated or not.
					for _, keyName := range keystoneNames {
						if inRadius(s.Tree.KeystoneMap[keyName], intuitiveLeapMap.RadiusIndex, depNode.ID) {
							prune = false
						}
					}
				}
				if prune {
					s.DeallocSingleNode(depNode)
				}
			}
		}
	}
}

func containsString(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func inRadius(keyNode *Node, radiusIndex, id int) bool {
	if keyNode.NodesInRadius == nil || radiusIndex < 1 || radiusIndex > len(keyNode.NodesInRadius) {
		return false
	}
	_, ok := keyNode.NodesInRadius[radiusIndex-1][id]
	return ok
}

func containsNode(list []*SpecNode, n *SpecNode) bool {
	for _, x := range list {
		if x == n {
			return true
		}
	}
	return false
}

//pob2:src/Classes/PassiveSpec.lua:897-901
func (s *PassiveSpec) CanPathThroughAllocMode(allocMode int, node *SpecNode) bool {
	nodeMode := node.AllocMode
	return nodeMode == 0 || allocMode > 0 && nodeMode == allocMode
}

//pob2:src/Classes/PassiveSpec.lua:1094-1125
func (s *PassiveSpec) FindStartFromNode(node *SpecNode, visited *[]*SpecNode, noAscend bool, allocMode int, alternateClassStartNodes map[int]*SpecNode) bool {
	node.Visited = true
	*visited = append(*visited, node)
	nodeAscendancy := node.AscendancyName != ""
	for _, other := range node.Linked {
		startIndex := len(*visited)
		if otherAlloc := other.Alloc || alternateClassStartNodes[other.ID] != nil; !otherAlloc || !s.CanPathThroughAllocMode(allocMode, other) {
			continue
		}
		otherType := other.Type
		if otherType == "ClassStart" || otherType == "AscendClassStart" ||
			(!other.Visited && node.Type != "Mastery" && s.FindStartFromNode(other, visited, noAscend, allocMode, alternateClassStartNodes)) {
			if nodeAscendancy && other.AscendancyName == "" {
				// Pathing out of Ascendant, un-visit the outside nodes
				for _, v := range (*visited)[startIndex:] {
					v.Visited = false
				}
				*visited = (*visited)[:startIndex]
			} else if !noAscend || otherType != "AscendClassStart" {
				return true
			}
		}
	}
	return false
}

//pob2:src/Classes/PassiveSpec.lua:1037-1045
func (s *PassiveSpec) DeallocSingleNode(node *SpecNode) {
	node.Alloc = false
	node.AllocMode = 0
	delete(s.AllocNodes, node.ID)
	if node.Type == "Mastery" {
		delete(s.MasterySelections, node.ID)
	}
}

// ReplaceNode keeps the name and stat lines; overlay, icon and the mod list are the view's and the calc's.
//
//pob2:src/Classes/PassiveSpec.lua:2080-2096
func (s *PassiveSpec) ReplaceNode(old *SpecNode, dn string, sd []string) {
	old.Dn = dn
	old.Sd = sd
}

//pob2:src/Classes/PassiveSpec.lua:2751-2762
func (s *PassiveSpec) SwitchAttributeNode(nodeID, attributeIndex int) {
	treeNode, ok := s.Tree.Nodes[nodeID]
	if !ok {
		return
	}
	if !treeNode.IsAttribute {
		return
	}
	newNode := &SpecNode{Node: treeNode, Dn: treeNode.Dn, Sd: treeNode.Sd}
	if attributeIndex >= 1 && attributeIndex <= len(treeNode.AttributeOptions) {
		option := treeNode.AttributeOptions[attributeIndex-1]
		s.ReplaceNode(newNode, option.Dn, option.Sd)
	}
	s.HashOverrides[nodeID] = newNode
}

// CountAllocNodes returns used, ascUsed, secondaryAscUsed, sockets, weaponSet1Used, weaponSet2Used.
//
//pob2:src/Classes/PassiveSpec.lua:1063-1090
func (s *PassiveSpec) CountAllocNodes() (int, int, int, int, int, int) {
	used, ascUsed, secondaryAscUsed, sockets, weaponSet1Used, weaponSet2Used := 0, 0, 0, 0, 0, 0
	for _, node := range s.AllocNodes {
		if node.Type != "ClassStart" && node.Type != "AscendClassStart" && !node.IsFreeAllocate {
			if node.AscendancyName != "" {
				if !node.IsMultipleChoiceOption {
					ascUsed++
				}
			} else {
				used++
			}
			if node.Type == "Socket" {
				sockets++
			}
		}
		switch node.AllocMode {
		case 1:
			weaponSet1Used++
		case 2:
			weaponSet2Used++
		}
	}
	return used, ascUsed, secondaryAscUsed, sockets, weaponSet1Used, weaponSet2Used
}

// GrantedPassive is a GrantedPassive mod's value: a node name, or a table such as
// { type = "SinisterJewelSockets", count = n } when Table is set.
type GrantedPassive struct {
	Name  string
	Table bool
	Type  string
	Count int
}

//pob2:src/Classes/PassiveSpec.lua:1138-1140
func normalisePassiveName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

var voicesSinisterSocketAliases = []string{
	"voices_jewel_slot1",
	"voices_jewel_slot2",
	"voices_jewel_slot3__",
	"voices_jewel_slot4",
	"voices_jewel_slot5",
}

func (s *PassiveSpec) specNodeOr(node *Node) *SpecNode {
	if n, ok := s.Nodes[node.ID]; ok {
		return n
	}
	return &SpecNode{Node: node, Dn: node.Dn, Sd: node.Sd, IsFreeAllocate: node.IsFreeAllocate}
}

func (s *PassiveSpec) item(itemID int) *Item {
	if s.Items == nil {
		return nil
	}
	return s.Items(itemID)
}

//pob2:src/Classes/PassiveSpec.lua:1127-1136
func (s *PassiveSpec) GetJewel(itemID int) *Item {
	if itemID == 0 {
		return nil
	}
	item := s.item(itemID)
	if item == nil || item.JewelData == nil {
		return nil
	}
	return item
}

//pob2:src/Classes/PassiveSpec.lua:1150-1194
func (s *PassiveSpec) ResolveGrantedPassiveNodes(passive GrantedPassive) []*SpecNode {
	var out []*SpecNode
	if passive.Table {
		if passive.Type == "SinisterJewelSockets" {
			byAlias := map[string]*Node{}
			for _, id := range sortedKeys(s.Tree.Sockets) {
				node := s.Tree.Sockets[id]
				if node.Sinister && node.AliasPassiveSocket != "" {
					byAlias[node.AliasPassiveSocket] = node
				}
			}
			for i := 0; i < min(passive.Count, len(voicesSinisterSocketAliases)); i++ {
				if node, ok := byAlias[voicesSinisterSocketAliases[i]]; ok {
					out = append(out, s.specNodeOr(node))
				}
			}
		}
		return out
	}

	passiveName := normalisePassiveName(passive.Name)
	if notable, ok := s.Tree.NotableMap[passiveName]; ok {
		return append(out, s.specNodeOr(notable))
	}
	node, ok := s.Tree.KeystoneMap[passiveName]
	if !ok {
		// PoB stops at the first match in pairs() order; ids are walked in order here.
		for _, id := range sortedKeys(s.Tree.Sockets) {
			socket := s.Tree.Sockets[id]
			if normalisePassiveName(socket.Dn) == passiveName {
				node, ok = socket, true
				break
			}
		}
	}
	if ok {
		out = append(out, s.specNodeOr(node))
	}
	return out
}

//pob2:src/Classes/PassiveSpec.lua:2739-2749
func (s *PassiveSpec) NodeInKeystoneRadius(keystoneNames map[string]bool, nodeID, radiusIndex int) bool {
	for _, id := range sortedKeys(s.Nodes) {
		node := s.Nodes[id]
		if node.Name != "" && node.Type == "Keystone" && keystoneNames[strings.ToLower(node.Name)] {
			if inRadius(node.Node, radiusIndex, nodeID) {
				return true
			}
		}
	}
	return false
}

// SetGrantedPassiveNodes takes CollectGrantedPassiveNodesFromItems' result, which the calc builds from
// the equipped items.
//
//pob2:src/Classes/PassiveSpec.lua:1224-1254
func (s *PassiveSpec) SetGrantedPassiveNodes(grantedNodeMap map[int]*SpecNode) bool {
	changed := false
	for _, nodeID := range sortedKeys(s.AllocNodes) {
		node := s.AllocNodes[nodeID]
		if node.IsGrantedPassive && node.IsFreeAllocate && grantedNodeMap[nodeID] == nil {
			node.Alloc = false
			node.IsGrantedPassive = false
			node.IsFreeAllocate = node.Node.IsFreeAllocate
			delete(s.AllocNodes, nodeID)
			changed = true
		}
	}
	for _, nodeID := range sortedKeys(grantedNodeMap) {
		specNode, ok := s.Nodes[nodeID]
		if !ok {
			specNode = grantedNodeMap[nodeID]
		}
		if _, ok := s.AllocNodes[nodeID]; !ok {
			specNode.Alloc = true
			specNode.AllocMode = 0
			specNode.IsGrantedPassive = true
			specNode.IsFreeAllocate = true
			s.AllocNodes[nodeID] = specNode
			changed = true
		}
	}
	if changed {
		s.BuildAllDependsAndPaths()
	}
	return changed
}
