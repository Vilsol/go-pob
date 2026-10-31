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
	ConqueredBy    *ConqueredBy

	// Path is nil where PoB's node.path is nil; HasPath tells an empty root path from none.
	Path                 []*SpecNode
	HasPath              bool
	PathDist             int
	PathRoot             *SpecNode
	DistanceToClassStart int
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
	ConqueredBy          *ConqueredBy
}

// ConqueredBy is jewelData.conqueredBy: a timeless jewel's seed and conqueror.
type ConqueredBy struct {
	ID        int
	Conqueror Conqueror
}

// Conqueror is a ModParser conquerorList entry.
type Conqueror struct {
	ID   string
	Type string
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
	// AttributeIndex is spec.attributeIndex; 0 stands for nil, which AllocNode reads as 1.
	AttributeIndex int

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

// SpecSave is the <Spec> PassiveSpec:Save writes, less the URL and the notes; the XML encoding belongs to
// the build-file package.
type SpecSave struct {
	TreeVersion          string
	ClassID              int
	AscendClassID        int
	ClassInternalID      int
	AscendancyInternalID string
	Nodes                string
	MasteryEffects       string
	// WeaponSets maps N to the nodes of a <WeaponSetN>; Sockets is the <Sockets> children in order.
	WeaponSets map[int]string
	Sockets    []SocketInput
	StrNodes   string
	DexNodes   string
	IntNodes   string
}

func joinInts(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}

//pob2:src/Classes/PassiveSpec.lua:252-366
func (s *PassiveSpec) Save() SpecSave {
	var allocNodeIDList []int
	weaponSets := map[int][]int{}
	for _, nodeID := range sortedKeys(s.AllocNodes) {
		node := s.AllocNodes[nodeID]
		if !(node.IsGrantedPassive && node.IsFreeAllocate) {
			allocNodeIDList = append(allocNodeIDList, nodeID)
		}
		if n, ok := s.Nodes[nodeID]; ok && n.AllocMode != 0 {
			weaponSets[n.AllocMode] = append(weaponSets[n.AllocMode], nodeID)
		}
	}
	masterySelections := make([]string, 0, len(s.MasterySelections))
	for _, mastery := range sortedKeys(s.MasterySelections) {
		masterySelections = append(masterySelections, fmt.Sprintf("{%d,%d}", mastery, s.MasterySelections[mastery]))
	}
	out := SpecSave{
		TreeVersion:          s.TreeVersion,
		ClassID:              s.CurClassID,
		AscendClassID:        s.CurAscendClassID,
		ClassInternalID:      s.Tree.Classes[s.CurClassID].IntegerID,
		AscendancyInternalID: s.ascendClassOrNone(s.CurAscendClassID).InternalID,
		Nodes:                joinInts(allocNodeIDList),
		MasteryEffects:       strings.Join(masterySelections, ","),
		WeaponSets:           map[int]string{},
	}
	for set, ids := range weaponSets {
		out.WeaponSets[set] = joinInts(ids)
	}
	for _, nodeID := range sortedKeys(s.Jewels) {
		if itemID := s.Jewels[nodeID]; itemID > 0 {
			out.Sockets = append(out.Sockets, SocketInput{NodeID: strconv.Itoa(nodeID), ItemID: strconv.Itoa(itemID)})
		}
	}
	var strList, dexList, intList []int
	for _, nodeID := range sortedKeys(s.HashOverrides) {
		node := s.HashOverrides[nodeID]
		if node.IsAttribute {
			switch node.Dn {
			case "Strength":
				strList = append(strList, nodeID)
			case "Dexterity":
				dexList = append(dexList, nodeID)
			case "Intelligence":
				intList = append(intList, nodeID)
			}
		}
	}
	out.StrNodes, out.DexNodes, out.IntNodes = joinInts(strList), joinInts(dexList), joinInts(intList)
	return out
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
// attribute overrides), the dependency walk and orphan pruning, the mastery/ignored deallocation and
// the intuitive-leap dependency resolution and the node paths.
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
		node.ConqueredBy = nil
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
			//pob2:block:src/Classes/PassiveSpec.lua:1530-1559
			for _, nodeID := range jewelIDs {
				itemID := s.Jewels[nodeID]
				item := s.item(itemID)
				if item == nil || item.JewelRadiusIndex == 0 || s.AllocNodes[nodeID] == nil || item.JewelData == nil || item.JewelData.LimitDisabled {
					continue
				}
				radiusIndex := item.JewelRadiusIndex
				if inRadius(s.Nodes[nodeID].Node, radiusIndex, node.ID) && itemID != 0 {
					if item.JewelData.IntuitiveLeapLike {
						node.IntuitiveLeapLikesAffecting = append(node.IntuitiveLeapLikesAffecting, s.Nodes[nodeID])
					}
					if item.JewelData.ConqueredBy != nil {
						node.ConqueredBy = item.JewelData.ConqueredBy
					}
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
		if node.ConqueredBy != nil && node.Type != "Socket" {
			s.conquerNode(node)
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
	potentialDeps := map[int][]*SpecNode{}
	intuitiveLeaps := map[int][]*SpecNode{}
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
					} else {
						if len(n.IntuitiveLeapLikesAffecting) > 0 {
							potentialDeps[n.ID] = append(potentialDeps[n.ID], node)
						} else {
							node.Depends = append(node.Depends, n)
						}
						// The intuitiveLeapLikeNodes loop (1907-1935) walks n.nodesInRadius, the tree's own nodes, whose
						// alloc is always nil, so it never inserts.
						intuitiveLeaps[node.ID] = append(intuitiveLeaps[node.ID], s.NodesInIntuitiveLeapLikeRadius(n)...)
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
						intuitiveLeaps[nodeID] = append(intuitiveLeaps[nodeID], depNode)
						break
					}
				}
				for _, intuitiveLeapMap := range s.intuitiveLeapLikeNodes {
					if !containsString(intuitiveLeapMap.To, depNode.Type) || intuitiveLeapMap.From != "Keystone" {
						continue
					}
					// Any keystone with the node in radius holds it, allocated or not.
					for _, keyName := range keystoneNames {
						if keyNode := s.Tree.KeystoneMap[keyName]; inRadius(keyNode, intuitiveLeapMap.RadiusIndex, depNode.ID) {
							prune = false
							intuitiveLeaps[keyNode.ID] = append(intuitiveLeaps[keyNode.ID], depNode)
						}
					}
				}
				if prune {
					s.DeallocSingleNode(depNode)
				}
			}
		}
	}

	//pob2:block:src/Classes/PassiveSpec.lua:2015-2050
	for _, id := range sortedKeys(potentialDeps) {
		potentialNode := s.Nodes[id]
		seen := map[int]bool{}
		for _, node := range potentialDeps[id] {
			allDep := true
			for _, intuitiveLeapLikeProvider := range potentialNode.IntuitiveLeapLikesAffecting {
				if !containsNode(node.Depends, intuitiveLeapLikeProvider) {
					allDep = false
				}
			}
			if allDep && !seen[node.ID] {
				node.Depends = append(node.Depends, potentialNode)
				seen[node.ID] = true
			}
		}
	}
	for _, id := range sortedKeys(intuitiveLeaps) {
		node, ok := s.Nodes[id]
		if !ok {
			continue
		}
		seen := map[int]bool{}
		for _, dep := range intuitiveLeaps[id] {
			if dep.ConnectedToStart {
				continue
			}
			allDep := true
			for _, intuitiveDep := range dep.IntuitiveLeapLikesAffecting {
				if !containsNode(node.Depends, intuitiveDep) {
					allDep = false
					break
				}
			}
			if allDep && !seen[dep.ID] {
				node.Depends = append(node.Depends, dep)
				seen[dep.ID] = true
			}
		}
	}

	//pob2:block:src/Classes/PassiveSpec.lua:2052-2077
	for _, id := range ids {
		node := s.Nodes[id]
		node.PathDist = 1000
		if node.Alloc && len(node.IntuitiveLeapLikesAffecting) == 0 {
			node.PathDist = 0
		}
		node.Path, node.HasPath = nil, false
		node.PathRoot = nil
		if node.IsJewelSocket {
			node.DistanceToClassStart = 0
		}
	}
	var rootList []*SpecNode
	for _, id := range sortedKeys(s.AllocNodes) {
		if node := s.AllocNodes[id]; len(node.IntuitiveLeapLikesAffecting) == 0 || node.ConnectedToStart {
			rootList = append(rootList, node)
		}
	}
	s.BuildNodePathsToRootNodes(rootList)
	for _, node := range rootList {
		if node.IsJewelSocket {
			s.SetNodeDistanceToClassStart(node)
		}
	}
	alternateClassStartNodesArray := make([]*SpecNode, 0, len(alternateClassStartNodes))
	for _, id := range sortedKeys(alternateClassStartNodes) {
		alternateClassStartNodesArray = append(alternateClassStartNodesArray, alternateClassStartNodes[id])
	}
	s.BuildNodePathsToRootNodes(alternateClassStartNodesArray)
}

//pob2:src/Classes/PassiveSpec.lua:1369-1400
func (s *PassiveSpec) NodesInIntuitiveLeapLikeRadius(node *SpecNode) []*SpecNode {
	var result []*SpecNode
	itemID := s.Jewels[node.ID]
	if itemID <= 0 {
		return result
	}
	item := s.GetJewel(itemID)
	if item == nil {
		return result
	}
	radiusIndex := item.JewelRadiusIndex
	allocated := func(radius *Node) {
		if radius.NodesInRadius == nil || radiusIndex < 1 || radiusIndex > len(radius.NodesInRadius) {
			return
		}
		for _, affectedNodeID := range sortedKeys(radius.NodesInRadius[radiusIndex-1]) {
			if n, ok := s.Nodes[affectedNodeID]; ok && n.Alloc {
				result = append(result, n)
			}
		}
	}
	if item.JewelData.IntuitiveLeapLike {
		allocated(s.Nodes[node.ID].Node)
	}
	if item.JewelData.FromNothingKeystone != "" {
		for _, keyName := range sortedStringKeys(item.JewelData.FromNothingKeystones) {
			if keyNode, ok := s.Tree.KeystoneMap[keyName]; ok {
				allocated(keyNode)
			}
		}
	}
	return result
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

// unlocked is the unlockConstraint check GetAllocationPath and BuildNodePathsToRootNodes repeat.
func (s *PassiveSpec) unlocked(node *SpecNode) bool {
	for _, nodeID := range node.UnlockConstraint.Nodes {
		if n, ok := s.Nodes[nodeID]; !ok || !n.Alloc {
			return false
		}
	}
	return true
}

// GetAllocationPath returns the path and its root; ok is false where the Lua returns nil.
//
//pob2:src/Classes/PassiveSpec.lua:903-959
func (s *PassiveSpec) GetAllocationPath(target *SpecNode, allocMode int, allocatedOnly bool, skipRoot *SpecNode) ([]*SpecNode, *SpecNode, bool) {
	visited := map[*SpecNode]bool{}
	prev := map[*SpecNode]*SpecNode{}
	var queue []*SpecNode
	rootMode := allocMode
	if allocatedOnly {
		rootMode = 0
	}
	for _, id := range sortedKeys(s.AllocNodes) {
		node := s.AllocNodes[id]
		if node != skipRoot && s.CanPathThroughAllocMode(rootMode, node) {
			visited[node] = true
			queue = append(queue, node)
		}
	}
	for o := 0; o < len(queue); o++ {
		node := queue[o]
		if node == target {
			var path []*SpecNode
			current := target
			for current != nil && prev[current] != nil {
				path = append(path, current)
				current = prev[current]
			}
			return path, current, true
		}
		if node.UnlockConstraint != nil && !s.unlocked(node) {
			continue
		}
		for _, other := range node.Linked {
			canPath := other.UnlockConstraint == nil || s.unlocked(other)
			otherMode := other.AllocMode
			canVisit := allocatedOnly && other.Alloc && s.CanPathThroughAllocMode(allocMode, other) && (otherMode > 0 || other.Type != "ClassStart" && other.Type != "AscendClassStart") ||
				!allocatedOnly && (!other.Alloc || s.CanPathThroughAllocMode(allocMode, other))
			if canPath && canVisit && !visited[other] && node.Type != "Mastery" && other.Type != "ClassStart" && other.Type != "AscendClassStart" &&
				(node.AscendancyName == other.AscendancyName || (prev[node] == nil && other.AscendancyName == "")) {
				visited[other] = true
				prev[other] = node
				queue = append(queue, other)
			}
		}
	}
	return nil, nil, false
}

//pob2:src/Classes/PassiveSpec.lua:961-986
func (s *PassiveSpec) GetEffectiveAllocationPath(node *SpecNode, altPath []*SpecNode) ([]*SpecNode, bool) {
	path, ok := node.Path, node.HasPath
	if altPath != nil {
		path, ok = altPath, true
	}
	if altPath != nil || len(node.IntuitiveLeapLikesAffecting) > 0 {
		return path, ok
	}
	pathRoot := node.PathRoot
	pathRootMode := 0
	if pathRoot != nil {
		pathRootMode = pathRoot.AllocMode
	}
	if pathRootMode == 0 {
		return path, ok
	}
	if s.AllocMode == 0 && pathRoot.Alloc {
		path = append([]*SpecNode(nil), path...)
		rootPath, _, found := s.GetAllocationPath(pathRoot, pathRootMode, true, pathRoot)
		if !found {
			rootPath = []*SpecNode{pathRoot}
		}
		for _, pathNode := range rootPath {
			if !containsNode(path, pathNode) {
				path = append(path, pathNode)
			}
		}
		return path, true
	} else if s.AllocMode > 0 && pathRootMode != s.AllocMode {
		path, _, ok = s.GetAllocationPath(node, s.AllocMode, false, nil)
	}
	return path, ok
}

// AllocNode returns an error where the Lua's multiple-choice assert fails.
//
//pob2:src/Classes/PassiveSpec.lua:988-1035
func (s *PassiveSpec) AllocNode(node *SpecNode, altPath []*SpecNode) error {
	if !node.HasPath {
		return nil
	}
	path, ok := s.GetEffectiveAllocationPath(node, altPath)
	if !ok {
		return nil
	}
	allocModeOf := func(n *SpecNode) int {
		if node.AscendancyName != "" || n.Type == "Keystone" || n.Type == "Socket" || n.ContainJewelSocket {
			return 0
		}
		return s.AllocMode
	}
	if len(node.IntuitiveLeapLikesAffecting) > 0 {
		node.Alloc = true
		node.AllocMode = allocModeOf(node)
		s.AllocNodes[node.ID] = node
	} else {
		attributeIndex := s.AttributeIndex
		if attributeIndex == 0 {
			attributeIndex = 1
		}
		for _, pathNode := range path {
			pathNode.Alloc = true
			pathNode.AllocMode = allocModeOf(pathNode)
			if pathNode.IsAttribute {
				s.SwitchAttributeNode(pathNode.ID, attributeIndex)
			}
			s.AllocNodes[pathNode.ID] = pathNode
		}
	}

	if node.IsMultipleChoiceOption {
		var parent *SpecNode
		for _, possibleParent := range node.Linked {
			if possibleParent.IsMultipleChoice && possibleParent.Alloc && possibleParent != node {
				parent = possibleParent
				break
			}
		}
		if parent == nil {
			return fmt.Errorf("tree2: multiple choice option %d has no allocated parent", node.ID)
		}
		for _, optNode := range parent.Linked {
			if optNode.IsMultipleChoiceOption && optNode.Alloc && optNode != node {
				s.DeallocSingleNode(optNode)
			}
		}
	}

	s.BuildAllDependsAndPaths()
	return nil
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

//pob2:src/Classes/PassiveSpec.lua:1047-1060
func (s *PassiveSpec) DeallocNode(node *SpecNode) {
	for _, depNode := range node.Depends {
		if depNode.IsAttribute {
			delete(s.HashOverrides, depNode.ID)
		}
		s.DeallocSingleNode(depNode)
	}
	s.BuildAllDependsAndPaths()
}

//pob2:src/Classes/PassiveSpec.lua:1311-1350
func (s *PassiveSpec) SetNodeDistanceToClassStart(root *SpecNode) {
	root.DistanceToClassStart = 0
	if !root.Alloc || !root.ConnectedToStart {
		return
	}
	targetNodeID := s.CurClass.StartNodeID
	nodeDistanceToRoot := map[int]int{root.ID: 0}
	queue := []*SpecNode{root}
	for o := 0; o < len(queue); o++ {
		node := queue[o]
		curDist := nodeDistanceToRoot[node.ID] + 1
		for _, other := range node.Linked {
			if other.ID == targetNodeID {
				root.DistanceToClassStart = curDist - 1
				return
			}
			if _, seen := nodeDistanceToRoot[other.ID]; other.Alloc && node.Type != "Mastery" && other.Type != "ClassStart" && other.Type != "AscendClassStart" && !seen {
				nodeDistanceToRoot[other.ID] = curDist
				queue = append(queue, other)
			}
		}
	}
}

// BuildNodePathsToRootNodes keeps the Lua's deque in a map, since a push to the front can run below index 1.
//
//pob2:src/Classes/PassiveSpec.lua:1402-1473
func (s *PassiveSpec) BuildNodePathsToRootNodes(roots []*SpecNode) {
	q := make(map[int]*SpecNode, len(roots))
	for i, node := range roots {
		node.PathDist = 0
		node.Path, node.HasPath = []*SpecNode{}, true
		node.PathRoot = node
		q[i+1] = node
	}
	qStart, qLen := 1, len(roots)
	for qStart <= qLen {
		node := q[qStart]
		qStart++
		if node.UnlockConstraint != nil && !s.unlocked(node) {
			continue
		}
		nodeDist := node.PathDist
		nodePath := node.Path
		for _, other := range node.Linked {
			weight := 1
			if other.Alloc {
				weight = 0
			}
			distViaNode := nodeDist + weight
			otherDist := other.PathDist
			preferNormalRoot := distViaNode == otherDist && node.PathRoot.AllocMode == 0 && other.PathRoot != nil && other.PathRoot.AllocMode != 0
			canPath := other.UnlockConstraint == nil || s.unlocked(other)
			if (distViaNode < otherDist || preferNormalRoot) &&
				node.Type != "Mastery" && other.Type != "ClassStart" && other.Type != "AscendClassStart" && (!other.Alloc || s.CanPathThroughAllocMode(node.AllocMode, other)) &&
				(node.AscendancyName == other.AscendancyName || (nodeDist == 0 && other.AscendancyName == "")) && canPath {
				if weight == 0 {
					qStart--
					q[qStart] = other
				} else {
					qLen++
					q[qLen] = other
				}
				other.PathDist = distViaNode
				path := make([]*SpecNode, 0, len(nodePath)+1)
				path = append(path, other)
				path = append(path, nodePath...)
				other.Path, other.HasPath = path, true
				other.PathRoot = node.PathRoot
			}
		}
	}
}

// ReplaceNode keeps the name and stat lines; overlay, icon and the mod list are the view's and the calc's.
//
//pob2:src/Classes/PassiveSpec.lua:2080-2096
func (s *PassiveSpec) ReplaceNode(old *SpecNode, dn string, sd []string) {
	old.Dn = dn
	old.Sd = sd
}

// conquerNode is the conquered-node branch of BuildAllDependsAndPaths. PoB-PoE2 has no seed data
// yet, so only keystones and the abyss's small passives change; ReconnectNodeToClassStart's
// Condition:ConnectedTo<Class>Start mod is not ported, as for unconquered nodes.
//
//pob2:block:src/Classes/PassiveSpec.lua:1478
//pob2:block:src/Classes/PassiveSpec.lua:1608-1789
func (s *PassiveSpec) conquerNode(node *SpecNode) {
	conqueredBy := node.ConqueredBy
	legionNodes := s.Tree.Legion
	switch node.Type {
	case "Keystone":
		matchStr := conqueredBy.Conqueror.Type + "_keystone_" + conqueredBy.Conqueror.ID
		for _, legionNode := range legionNodes {
			if legionNode.LegionID == matchStr {
				s.ReplaceNode(node, legionNode.Dn, legionNode.Sd)
				break
			}
		}
	case "Normal":
		if conqueredBy.Conqueror.Type == "abyss" {
			if containsString([]string{"Dexterity", "Intelligence", "Strength", "Attribute"}, node.Dn) {
				s.nodeAdditionOrReplacementFromString(node, " \n+3 to Tribute", false)
			} else if len(legionNodes) >= 201 {
				legionNode := legionNodes[200] // abyss_small_tribute
				s.ReplaceNode(node, legionNode.Dn, legionNode.Sd)
			}
		}
	}
}

// nodeAdditionOrReplacementFromString splits sd into lines and adds them to the node's, or replaces
// them; the node's mod list is built from its lines where the calc reads it.
//
//pob2:src/Classes/PassiveSpec.lua:2656-2737
func (s *PassiveSpec) nodeAdditionOrReplacementFromString(node *SpecNode, sd string, replacement bool) {
	var lines []string
	for _, l := range strings.Split(sd, "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	if replacement {
		node.Sd = lines
		return
	}
	node.Sd = append(append([]string(nil), node.Sd...), lines...)
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

// UndoState is what CreateUndoState records; WeaponSets maps a node id to its allocMode. The node notes
// are not ported.
type UndoState struct {
	ClassInternalID        int
	AscendancyInternalID   string
	ClassID                int
	AscendClassID          int
	SecondaryAscendClassID int
	HashList               []int
	WeaponSets             map[int]int
	HashOverrides          map[int]*SpecNode
	MasteryEffects         map[int]int
	TreeVersion            string
}

//pob2:src/Classes/PassiveSpec.lua:2597-2630
func (s *PassiveSpec) CreateUndoState() *UndoState {
	state := &UndoState{
		ClassInternalID: s.Tree.Classes[s.CurClassID].IntegerID,
		ClassID:         s.CurClassID,
		AscendClassID:   s.CurAscendClassID,
		HashList:        sortedKeys(s.AllocNodes),
		WeaponSets:      map[int]int{},
		HashOverrides:   make(map[int]*SpecNode, len(s.HashOverrides)),
		MasteryEffects:  make(map[int]int, len(s.MasterySelections)),
		TreeVersion:     s.TreeVersion,
	}
	for _, nodeID := range state.HashList {
		if n, ok := s.Nodes[nodeID]; ok && n.AllocMode != 0 {
			state.WeaponSets[nodeID] = n.AllocMode
		}
	}
	for mastery, effect := range s.MasterySelections {
		state.MasteryEffects[mastery] = effect
	}
	if s.CurAscendClassID >= 0 && s.CurAscendClassID < len(s.CurClass.Classes) {
		state.AscendancyInternalID = s.CurClass.Classes[s.CurAscendClassID].InternalID
	}
	for id, n := range s.HashOverrides {
		state.HashOverrides[id] = n
	}
	return state
}

// RestoreUndoState takes no tree version: go-pob converts no spec between versions.
//
//pob2:src/Classes/PassiveSpec.lua:2632-2646
func (s *PassiveSpec) RestoreUndoState(state *UndoState) {
	s.ImportFromNodeList(state.ClassID, state.AscendClassID, state.SecondaryAscendClassID, state.HashList, state.WeaponSets, state.HashOverrides, state.MasteryEffects)
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
