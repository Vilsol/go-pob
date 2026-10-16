package calculator

import (
	"sort"
	"strconv"
	"sync"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

// 0-3:   node index (0-11)
// 4-5:   group size (0-2)
// 6-8:   large index (0-5)
// 9-10:  medium index (0-2)
// 11-15: unused
const clusterNodeIDBase = 0x10000

const maxClusterNesting = 4

func isClusterNodeID(nodeID string) bool {
	id, err := strconv.Atoi(nodeID)
	return err == nil && id >= clusterNodeIDBase
}

var clusterNodeMaps sync.Map

//pob:block:src/Classes/PassiveTree.lua:619-621
func clusterNodeMap(version pob.TreeVersion) map[string]data.Node {
	if cached, ok := clusterNodeMaps.Load(version); ok {
		return cached.(map[string]data.Node)
	}

	tree := data.TreeVersions[version].Tree()
	nodeMap := make(map[string]data.Node)
	for _, node := range tree.Nodes {
		if node.Group != nil || node.Name == nil {
			continue
		}
		if (node.IsNotable != nil && *node.IsNotable) || (node.IsKeystone != nil && *node.IsKeystone) {
			nodeMap[*node.Name] = node
		}
	}

	clusterNodeMaps.Store(version, nodeMap)
	return nodeMap
}

type clusterJewelData struct {
	clusterJewel data.ClusterJewel

	skill      string
	keystone   string
	notables   []string
	addedMods  []string
	incEffect  float64
	sizeIsSet  bool
	nodeCount  int
	socketSet  bool
	socket     int
	overrideOK bool
	override   int
	nothingSet bool
	nothing    int

	smallsAreNothingness bool
}

//pob:block:src/Classes/Item.lua:2514-2516
func (c *clusterJewelData) valid() bool {
	if c.keystone != "" {
		return true
	}
	if (c.skill != "" || c.smallsAreNothingness) && c.sizeIsSet {
		return true
	}
	return c.overrideOK && c.nothingSet
}

//pob:block:src/Classes/Item.lua:1267-1293
func readClusterJewelData(item *Item) *clusterJewelData {
	if item == nil || item.Type != "Jewel" {
		return nil
	}
	jewel, ok := data.ClusterJewels.Jewels[item.BaseName]
	if !ok {
		return nil
	}

	out := &clusterJewelData{clusterJewel: jewel}
	for _, m := range item.Mods {
		switch m.Name() {
		case "JewelData":
			kv, isKV := m.Value().ValueList.(mod.JewelData)
			if !isKV {
				continue
			}
			switch kv.Key {
			case "clusterJewelSkill":
				out.skill, _ = kv.Value.(string)
			case "clusterJewelKeystone":
				out.keystone, _ = kv.Value.(string)
			case "clusterJewelNodeCount":
				out.nodeCount, out.sizeIsSet = jewelDataInt(kv.Value)
			case "clusterJewelSocketCount":
				out.socket, out.socketSet = jewelDataInt(kv.Value)
			case "clusterJewelSocketCountOverride":
				out.override, out.overrideOK = jewelDataInt(kv.Value)
			case "clusterJewelNothingnessCount":
				out.nothing, out.nothingSet = jewelDataInt(kv.Value)
			case "clusterJewelSmallsAreNothingness":
				b, _ := kv.Value.(bool)
				out.smallsAreNothingness = b
			case "clusterJewelIncEffect":
				if v, isNum := jewelDataInt(kv.Value); isNum {
					out.incEffect = float64(v)
				}
			}
		case "ClusterJewelNotable":
			if name, isStr := m.Value().ValueList.(string); isStr {
				out.notables = append(out.notables, name)
			}
		case "AddToClusterJewelNode":
			if line, isStr := m.Value().ValueList.(string); isStr {
				out.addedMods = append(out.addedMods, line)
			}
		}
	}

	if out.skill == "affliction_curse_effect" && out.sizeIsSet && out.nodeCount < 4 {
		out.skill = "affliction_curse_effect_small"
	}

	if out.sizeIsSet {
		out.nodeCount = min(max(out.nodeCount, jewel.MinNodes), jewel.MaxNodes)
	}
	if out.skill != "" {
		if _, known := jewel.Skills[out.skill]; !known {
			out.skill = ""
		}
	}

	return out
}

func jewelDataInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	}
	return 0, false
}

type clusterBuilder struct {
	tree       *data.Tree
	nodeMap    map[string]data.Node
	jewels     map[string]*clusterJewelData
	sortOrde   map[string]int
	nodes      map[string]data.Node
	legacyHash bool
	layout     map[string]*clusterNodeLayout
	allocated  map[string]bool
}

type clusterNodeLayout struct {
	kind       string
	group      int64
	orbit      int
	orbitIndex int
	mastery    *data.Node
}

//pob:src/Classes/PassiveSpec.lua:1806-1882
func buildClusterJewelGraphs(env *Environment, version pob.TreeVersion) map[string]data.Node {
	nodes, _ := buildClusterJewelLayout(env, version, false)
	return nodes
}

func buildClusterJewelLayout(env *Environment, version pob.TreeVersion, withLayout bool) (map[string]data.Node, map[string]*clusterNodeLayout) {
	jewels := socketedClusterJewels(env)
	if len(jewels) == 0 {
		return nil, nil
	}

	b := &clusterBuilder{
		tree:     data.TreeVersions[version].Tree(),
		nodeMap:  clusterNodeMap(version),
		jewels:   jewels,
		sortOrde: data.ClusterJewels.NotableSortOrder,
		nodes:    make(map[string]data.Node),
	}
	if withLayout {
		b.layout = make(map[string]*clusterNodeLayout)
	}
	//pob:block:src/Classes/PassiveSpec.lua:106,src/Classes/PassiveSpec.lua:1735
	if specs := env.Build.Tree.Specs; len(specs) > 0 {
		idx := env.Build.Tree.ActiveSpec - 1
		if idx < 0 || idx >= len(specs) {
			idx = 0
		}
		spec := specs[idx]
		b.allocated = make(map[string]bool)
		for _, id := range pob.SpecNodes(spec) {
			b.allocated[strconv.FormatInt(id, 10)] = true
		}
		if spec.ClusterHashFormatVersion != nil {
			b.legacyHash = *spec.ClusterHashFormatVersion < 2
		} else {
			b.legacyHash = spec.NodesAttr != ""
		}
	}

	for _, slot := range b.tree.JewelSlots {
		socketID := strconv.FormatInt(slot, 10)
		socket, ok := b.tree.Nodes[socketID]
		//pob:block:src/Classes/PassiveSpec.lua:1849
		if !ok || !b.allocated[socketID] || socket.ExpansionJewel == nil || socket.ExpansionJewel.Size != 2 {
			continue
		}
		jewel, socketed := jewels[socketID]
		if !socketed || !jewel.valid() {
			continue
		}
		b.buildSubgraph(jewel, socketID, socket, clusterNodeIDBase, 0)
	}

	return b.nodes, b.layout
}

//pob:src/Classes/PassiveSpec.lua:901-910
func socketedClusterJewels(env *Environment) map[string]*clusterJewelData {
	specs := env.Build.Tree.Specs
	if len(specs) == 0 {
		return nil
	}
	idx := env.Build.Tree.ActiveSpec - 1
	if idx < 0 || idx >= len(specs) {
		idx = 0
	}

	byID := itemTextByID(env)

	var jewels map[string]*clusterJewelData
	for _, socket := range specs[idx].Sockets {
		text, ok := byID[socket.ItemID]
		if !ok {
			continue
		}
		jewelData := readClusterJewelData(peekItemCached(env.Cache, text))
		if jewelData == nil {
			continue
		}
		if jewels == nil {
			jewels = make(map[string]*clusterJewelData)
		}
		jewels[strconv.Itoa(socket.NodeID)] = jewelData
	}

	return jewels
}

//nolint:gocyclo // Mirrors the Lua function.
//pob:src/Classes/PassiveSpec.lua:1970-2360
func (b *clusterBuilder) buildSubgraph(jewel *clusterJewelData, parentSocketID string, parentSocket data.Node, id int, depth int) {
	if depth >= maxClusterNesting || parentSocket.ExpansionJewel == nil {
		return
	}
	expansionJewel := parentSocket.ExpansionJewel
	clusterJewel := jewel.clusterJewel

	switch expansionJewel.Size {
	case 2:
		id += int(expansionJewel.Index) << 6
	case 1:
		id += int(expansionJewel.Index) << 9
	}
	nodeID := id + clusterJewel.SizeIndex<<4

	// Locate the proxy group.
	proxyNode, ok := b.tree.Nodes[expansionJewel.Proxy]
	if !ok || proxyNode.Group == nil {
		return
	}
	proxyGroup := b.tree.Groups[strconv.FormatInt(*proxyNode.Group, 10)]
	place := func(id, kind string, orbit, orbitIndex int) {
		if b.layout != nil {
			b.layout[id] = &clusterNodeLayout{kind: kind, group: *proxyNode.Group, orbit: orbit, orbitIndex: orbitIndex}
		}
	}

	if jewel.keystone != "" {
		keystoneNode, found := b.nodeMap[jewel.keystone]
		if !found {
			return
		}
		b.nodes[strconv.Itoa(nodeID)] = data.Node{
			Name:       keystoneNode.Name,
			Stats:      keystoneNode.Stats,
			Icon:       keystoneNode.Icon,
			IsKeystone: keystoneNode.IsKeystone,
		}
		place(strconv.Itoa(nodeID), "Keystone", 0, 1)
		//pob:block:src/Classes/PassiveSpec.lua:2094
		b.linkNodes(strconv.Itoa(nodeID), parentSocketID)
		return
	}

	groupSize := int(expansionJewel.Size)
	for b.legacyHash && clusterJewel.SizeIndex < groupSize {
		socket := b.findSocket(proxyGroup, 1)
		if socket == nil {
			socket = b.findSocket(proxyGroup, 0)
		}
		if socket == nil || socket.node.ExpansionJewel == nil {
			return
		}
		next, exists := b.tree.Nodes[socket.node.ExpansionJewel.Proxy]
		if !exists || next.Group == nil {
			return
		}
		proxyGroup = b.tree.Groups[strconv.FormatInt(*next.Group, 10)]
		groupSize = int(socket.node.ExpansionJewel.Size)
	}

	notableList := make([]data.Node, 0, len(jewel.notables))
	for _, name := range jewel.notables {
		baseNode, found := b.nodeMap[name]
		if !found {
			return
		}
		notableList = append(notableList, baseNode)
	}
	sort.SliceStable(notableList, func(i, j int) bool {
		return b.sortOrder(notableList[i]) < b.sortOrder(notableList[j])
	})

	skill, hasSkill := clusterJewel.Skills[jewel.skill]
	if !hasSkill {
		skill = data.ClusterJewelSkill{Name: "Nothingness", Icon: "Art/2DArt/SkillIcons/passives/MasteryBlank.png"}
	}

	socketCount := 0
	switch {
	case jewel.overrideOK:
		socketCount = jewel.override
	case jewel.socketSet:
		socketCount = jewel.socket
	}
	notableCount := len(notableList)
	nodeCount := jewel.nodeCount
	if !jewel.sizeIsSet {
		nodeCount = socketCount + notableCount + jewel.nothing
	}
	smallCount := nodeCount - socketCount - notableCount

	//pob:block:src/Classes/PassiveSpec.lua:2109,src/Classes/PassiveSpec.lua:2138-2151
	nodeOrbit := clusterJewel.SizeIndex + 1
	if skill.MasteryIcon != "" && b.layout != nil {
		masteryID := strconv.Itoa(nodeID + 12)
		place(masteryID, "Mastery", 0, 0)
		name, icon := "Nothingness", skill.MasteryIcon
		b.layout[masteryID].mastery = &data.Node{Name: &name, Icon: &icon}
	}

	indicies := make(map[int]string)

	//pob:src/Classes/PassiveSpec.lua:1334-1353
	sockets := make([]clusterSocket, 0, 3)
	makeJewel := func(nodeIndex, jewelIndex int) {
		socket := b.findSocket(proxyGroup, int64(jewelIndex))
		if socket == nil {
			return
		}
		indicies[nodeIndex] = socket.id
		place(socket.id, "Socket", nodeOrbit, nodeIndex)
		b.nodes[socket.id] = data.Node{
			Name:           socket.node.Name,
			Icon:           socket.node.Icon,
			IsJewelSocket:  socket.node.IsJewelSocket,
			ExpansionJewel: socket.node.ExpansionJewel,
		}
		sockets = append(sockets, *socket)
	}
	if clusterJewel.Size == "Large" && socketCount == 1 {
		// Large clusters always have the single jewel at index 6.
		makeJewel(6, 1)
	} else {
		getJewels := []int{0, 2, 1}
		for i := 1; i <= socketCount && i <= len(clusterJewel.SocketIndicies) && i <= len(getJewels); i++ {
			makeJewel(clusterJewel.SocketIndicies[i-1], getJewels[i-1])
		}
	}

	// Second pass: notables.
	notableIndexList := make([]int, 0, notableCount)
	for _, nodeIndex := range clusterJewel.NotableIndicies {
		if len(notableIndexList) == notableCount {
			break
		}
		if clusterJewel.Size == "Medium" {
			if socketCount == 0 && notableCount == 2 {
				// Special rule for two notables in a Medium cluster
				switch nodeIndex {
				case 6:
					nodeIndex = 4
				case 10:
					nodeIndex = 8
				}
			} else if nodeCount == 4 {
				// Special rule for notables in a 4-node Medium cluster
				switch nodeIndex {
				case 10:
					nodeIndex = 9
				case 2:
					nodeIndex = 3
				}
			}
		}
		if indicies[nodeIndex] == "" {
			notableIndexList = append(notableIndexList, nodeIndex)
		}
	}
	sort.Ints(notableIndexList)

	for index, baseNode := range notableList {
		// "Silently fail to handle cases of jewels with more notables than should be allowed"
		if index >= len(notableIndexList) {
			break
		}
		nodeIndex := notableIndexList[index]
		b.nodes[strconv.Itoa(nodeID+nodeIndex)] = data.Node{
			Name:      baseNode.Name,
			Stats:     baseNode.Stats,
			Icon:      baseNode.Icon,
			IsNotable: baseNode.IsNotable,
		}
		indicies[nodeIndex] = strconv.Itoa(nodeID + nodeIndex)
		place(indicies[nodeIndex], "Notable", nodeOrbit, nodeIndex)
	}

	// Third pass: small fill.
	smallIndexList := make([]int, 0, max(smallCount, 0))
	for _, nodeIndex := range clusterJewel.SmallIndicies {
		if len(smallIndexList) == smallCount {
			break
		}
		if clusterJewel.Size == "Medium" {
			// Special rules for small nodes in Medium clusters
			if nodeCount == 5 && nodeIndex == 4 {
				nodeIndex = 3
			} else if nodeCount == 4 {
				switch nodeIndex {
				case 8:
					nodeIndex = 9
				case 4:
					nodeIndex = 3
				}
			}
		}
		if indicies[nodeIndex] == "" {
			smallIndexList = append(smallIndexList, nodeIndex)
		}
	}

	smallStats := make([]string, 0, len(skill.Stats)+len(jewel.addedMods))
	smallStats = append(smallStats, skill.Stats...)
	smallStats = append(smallStats, jewel.addedMods...)

	for index := 0; index < smallCount; index++ {
		if index >= len(smallIndexList) {
			break
		}
		nodeIndex := smallIndexList[index]
		skillName := skill.Name
		b.nodes[strconv.Itoa(nodeID+nodeIndex)] = data.Node{
			Name:                  &skillName,
			Stats:                 smallStats,
			Icon:                  &skill.Icon,
			ClusterJewelIncEffect: jewel.incEffect,
		}
		indicies[nodeIndex] = strconv.Itoa(nodeID + nodeIndex)
		place(indicies[nodeIndex], "Normal", nodeOrbit, nodeIndex)
	}

	//pob:block:src/Classes/PassiveSpec.lua:2306-2309
	if b.layout != nil && len(b.tree.Constants.SkillsPerOrbit) > nodeOrbit {
		skillsPerOrbit := int(b.tree.Constants.SkillsPerOrbit[nodeOrbit])
		proxyID, _ := strconv.Atoi(expansionJewel.Proxy)
		startOidx := data.ClusterJewels.OrbitOffsets[proxyID][clusterJewel.SizeIndex]
		for _, id := range indicies {
			if l := b.layout[id]; l != nil {
				l.orbitIndex = translateClusterOrbitIndex((l.orbitIndex+startOidx)%clusterJewel.TotalIndicies, clusterJewel.TotalIndicies, skillsPerOrbit)
			}
		}
	}

	//pob:src/Classes/PassiveSpec.lua:2322-2340
	entrance := indicies[0]
	if entrance == "" {
		return
	}
	firstNode, lastNode := "", ""
	for i := 0; i < clusterJewel.TotalIndicies; i++ {
		thisNode := indicies[i]
		if thisNode == "" {
			continue
		}
		if firstNode == "" {
			firstNode = thisNode
		}
		if lastNode != "" {
			b.linkNodes(thisNode, lastNode)
		}
		lastNode = thisNode
	}
	if firstNode != lastNode && clusterJewel.Size != "Small" {
		b.linkNodes(firstNode, lastNode)
	}
	b.linkNodes(entrance, parentSocketID)

	for i := range sockets {
		nested, socketed := b.jewels[sockets[i].id]
		//pob:block:src/Classes/PassiveSpec.lua:2351-2352
		if b.allocated[sockets[i].id] && socketed && nested.valid() {
			b.buildSubgraph(nested, sockets[i].id, b.nodes[sockets[i].id], id, depth+1)
		}
	}
}

//pob:src/Classes/PassiveSpec.lua:1916-1933
func translateClusterOrbitIndex(srcOidx, srcNodesPerOrbit, destNodesPerOrbit int) int {
	var table []int
	switch {
	case srcNodesPerOrbit == destNodesPerOrbit:
		return srcOidx
	case srcNodesPerOrbit == 12 && destNodesPerOrbit == 16:
		table = []int{0, 1, 3, 4, 5, 7, 8, 9, 11, 12, 13, 15}
	case srcNodesPerOrbit == 16 && destNodesPerOrbit == 12:
		table = []int{0, 1, 1, 2, 3, 4, 4, 5, 6, 7, 7, 8, 9, 10, 10, 11}
	case srcNodesPerOrbit == 6 && destNodesPerOrbit == 16:
		table = []int{0, 3, 5, 8, 11, 13}
	case srcNodesPerOrbit == 16 && destNodesPerOrbit == 6:
		table = []int{0, 0, 0, 1, 1, 2, 2, 2, 3, 3, 3, 4, 4, 5, 5, 5}
	default:
		return srcOidx * destNodesPerOrbit / srcNodesPerOrbit
	}
	if srcOidx < 0 || srcOidx >= len(table) {
		return srcOidx
	}
	return table[srcOidx]
}

//pob:block:src/Classes/PassiveSpec.lua:2018-2027
func (b *clusterBuilder) linkNodes(a, c string) {
	b.addLink(a, c)
	b.addLink(c, a)
}

//pob:block:src/Classes/PassiveSpec.lua:2019-2020
func (b *clusterBuilder) addLink(id, other string) {
	node, ok := b.nodes[id]
	if !ok {
		node, ok = b.tree.Nodes[id]
		if !ok {
			return
		}
		node.In = append([]string(nil), node.In...)
	}
	node.Out = append(append(make([]string, 0, len(node.Out)+1), node.Out...), other)
	b.nodes[id] = node
}

//pob:block:src/Classes/PassiveSpec.lua:2114-2126
func (b *clusterBuilder) sortOrder(node data.Node) int {
	if node.Name == nil {
		return len(b.sortOrde)
	}
	if order, ok := b.sortOrde[*node.Name]; ok {
		return order
	}
	return len(b.sortOrde)
}

type clusterSocket struct {
	id   string
	node data.Node
}

//pob:src/Classes/PassiveSpec.lua:1262-1270
func (b *clusterBuilder) findSocket(group data.Group, index int64) *clusterSocket {
	for _, nodeID := range group.Nodes {
		node, ok := b.tree.Nodes[nodeID]
		if ok && node.ExpansionJewel != nil && node.ExpansionJewel.Index == index {
			return &clusterSocket{id: nodeID, node: node}
		}
	}
	return nil
}
