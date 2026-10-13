package calculator

import (
	"maps"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/calculator/calclib"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

type radiusJewelData struct {
	modSource mod.Source

	stats map[string]float64

	total   float64
	totalOK bool
	modList *moddb.ModList

	funcData map[int]*radiusJewelData
}

//pob:block:src/Modules/CalcSetup.lua:176-177
func newRadiusJewelData(source mod.Source) *radiusJewelData {
	return &radiusJewelData{modSource: source, stats: make(map[string]float64)}
}

type radiusJewelNode struct {
	id       string
	nodeType string
	modList  *moddb.ModList

	conquered bool

	isTattoo bool
}

type radiusJewelFunc func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData)

type jewelFuncEntry struct {
	Type string
	Func radiusJewelFunc
}

func (s jewelFuncEntry) Clone(pointer bool) mod.Cloneable {
	obj := jewelFuncEntry{Type: s.Type, Func: s.Func}
	if pointer {
		return &obj
	}
	return obj
}

type nodeModifier struct {
	Mod mod.Mod
}

func (s nodeModifier) Clone(pointer bool) mod.Cloneable {
	obj := nodeModifier{Mod: s.Mod}
	if pointer {
		return &obj
	}
	return obj
}

type radiusJewel struct {
	nodes    map[string]bool
	funcType string
	fn       radiusJewelFunc
	nodeID   string
	data     *radiusJewelData
}

type extraJewelFuncEntry struct {
	Radius string
	Type   string
	Func   radiusJewelFunc
}

//pob:src/Classes/ModList.lua:69-82
func mergeMod(list *moddb.ModList, m mod.Mod) {
	if t := m.Type(); t == mod.TypeBase || t == mod.TypeIncrease || t == mod.TypeMore {
		mods := list.Mods()
		for i, existing := range mods {
			if !moddb.CompareModParams(existing, m) {
				continue
			}
			ev, nv := existing.Value(), m.Value()
			if ev == nil || nv == nil ||
				ev.Type() != mod.ModValueMultiTypeFloat || nv.Type() != mod.ModValueMultiTypeFloat {
				continue
			}
			merged := existing.Clone()
			merged.Value().SetFloat(ev.Float() + nv.Float())
			mods[i] = merged
			list.SetMods(mods)
			return
		}
	}
	list.AddMod(m)
}

//pob:src/Classes/ModList.lua:92-94
func mergeNewMod(list *moddb.ModList, name string, modType mod.Type, value float64, proto mod.Mod) {
	m := mod.NewFloat(name, modType, value).
		Source(proto.GetSource()).
		Flag(proto.Flags()).
		KeywordFlag(proto.KeywordFlags())
	if tags := proto.Tags(); len(tags) > 0 {
		m.Tag(tags...)
	}
	mergeMod(list, m)
}

//pob:block:src/Classes/PassiveTree.lua:313-361
func nodeTypeOf(node data.Node) string {
	switch {
	case node.ClassStartIndex != nil:
		return "ClassStart"
	case node.IsAscendancyStart != nil && *node.IsAscendancyStart:
		return "AscendClassStart"
	case node.IsMastery != nil && *node.IsMastery:
		return "Mastery"
	case node.IsJewelSocket != nil && *node.IsJewelSocket:
		return "Socket"
	case node.IsKeystone != nil && *node.IsKeystone:
		return "Keystone"
	case node.IsNotable != nil && *node.IsNotable:
		return "Notable"
	}
	return "Normal"
}

func inAnyRadius(jewels []*radiusJewel, nodeID string) bool {
	for _, rad := range jewels {
		if rad.nodes[nodeID] {
			return true
		}
	}
	return false
}

//pob:block:src/Modules/CalcSetup.lua:1323-1333
func buildModListForNodeList(env *Environment, nodes map[string]data.Node, finishJewels bool) (*moddb.ModList, []ExplodeSource) {
	errorsBefore := len(env.DebugErrors)

	modList, explodeSources := buildModListForNodeListPass(env, nodes, finishJewels, nil)

	extraFuncs := collectExtraJewelFuncs(env, modList)
	if extraFuncs == nil {
		return modList, explodeSources
	}

	env.DebugErrors = env.DebugErrors[:errorsBefore]
	return buildModListForNodeListPass(env, nodes, finishJewels, extraFuncs)
}

type ExplodeSource struct {
	ModSource string
	NodeID    string
	Rarity    ItemRarity
}

//pob:block:src/Modules/CalcSetup.lua:1424
func (s ExplodeSource) key() string {
	if s.ModSource != "" {
		return s.ModSource
	}
	return s.NodeID
}

//pob:block:src/Data/Skills/other.lua:6081
func (s ExplodeSource) activeEffectSource() string {
	if s.ModSource != "" {
		return s.ModSource
	}
	return "Tree:" + s.NodeID
}

//pob:block:src/Modules/CalcSetup.lua:1323-1329
func collectExtraJewelFuncs(env *Environment, nodeMods *moddb.ModList) *moddb.ModList {
	prevActor := nodeMods.Actor
	nodeMods.Actor = env.Player
	rows := env.ModDB.Tabulate(mod.TypeList, nil, "ExtraJewelFunc")
	rows = append(rows, nodeMods.Tabulate(mod.TypeList, nil, "ExtraJewelFunc")...)
	nodeMods.Actor = prevActor
	if len(rows) == 0 {
		return nil
	}

	out := moddb.NewModList()
	out.Actor = env.Player
	for _, row := range rows {
		out.AddMod(row.Mod)
	}
	return out
}

//pob:src/Modules/CalcSetup.lua:140-179
func buildModListForNodeListPass(env *Environment, nodes map[string]data.Node, finishJewels bool, extraFuncs *moddb.ModList) (*moddb.ModList, []ExplodeSource) {
	jewels, extraRadiusNodeIDs := buildRadiusJewelList(env, extraFuncs)

	// Add node modifiers
	var modList = moddb.NewModList()
	var explodeSources []ExplodeSource
	nodeIds := make([]string, 0, len(nodes))
	for nodeId := range nodes {
		nodeIds = append(nodeIds, nodeId)
	}
	sort.Strings(nodeIds)

	conquered := nodeRewrites(env)

	for _, nodeId := range nodeIds {
		node := nodes[nodeId]
		isMastery := node.IsMastery != nil && *node.IsMastery
		transform, isConquered := conquered[nodeId]
		uncacheable := isMastery || isClusterNodeID(nodeId) || isConquered || inAnyRadius(jewels, nodeId)
		cachedModList, isCached := env.Cache.nodeMods(nodeId)
		var nodeModList *moddb.ModList
		if isCached && !uncacheable {
			nodeModList = &cachedModList
		} else {
			nodeModList = buildModListForConqueredNode(env, nodeId, node, transform, jewels)
			if !uncacheable {
				env.Cache.storeNodeMods(nodeId, *nodeModList)
			}
		}
		modList.AddDB(nodeModList)

		//pob:block:src/Modules/CalcSetup.lua:169,src/Modules/CalcSetup.lua:184-185
		if nodeModList.Flag(nil, "CanExplode") {
			explodeSources = append(explodeSources, ExplodeSource{NodeID: nodeId})
		}
	}

	if finishJewels {
		if len(extraRadiusNodeIDs) > 0 {
			tree := data.TreeVersions[data.LatestTreeVersion].Tree()
			if env.Spec != nil {
				tree = env.Spec.Tree()
			}
			for _, nodeId := range extraRadiusNodeIDs {
				node, exists := tree.Nodes[nodeId]
				if !exists {
					continue
				}
				buildModListForConqueredNode(env, nodeId, node, conquered[nodeId], jewels)
			}
		}

		for _, rad := range jewels {
			rad.fn(nil, modList, rad.data)
			//pob:block:src/Modules/CalcSetup.lua:204-209
			if env.Mode == OutputModeMain || env.Mode == OutputModeCalculator {
				if env.JewelRadiusData == nil {
					env.JewelRadiusData = map[string]map[string]float64{}
				}
				env.JewelRadiusData[rad.nodeID] = maps.Clone(rad.data.stats)
			}
		}
	}

	return modList, explodeSources
}

//pob:src/Classes/PassiveTree.lua:769-846
func processStats(env *Environment, modList *moddb.ModList, nodeId string, what string, stats []string) {
	stats = fixupTreeStatText(stats)

	lines := make([]string, len(stats))
	copy(lines, stats)

	swallowed := make(map[int]bool)

	for i := 0; i < len(lines); i++ {
		if swallowed[i] {
			continue
		}

		if strings.Contains(lines[i], "\n") {
			parts := make([]string, 0, 2)
			for _, part := range strings.Split(lines[i], "\n") {
				if part != "" {
					parts = append(parts, part)
				}
			}
			lines = append(lines[:i:i], append(parts, lines[i+1:]...)...)
			if i >= len(lines) {
				break
			}
		}

		line := lines[i]
		entry := ParseMod(line, false)
		if entry.ModList == nil || strings.Trim(entry.Extra, " ") != "" {
			for endI := i + 1; endI < len(lines); endI++ {
				comb := line
				for ci := i + 1; ci <= endI; ci++ {
					comb += " " + lines[ci]
				}
				combEntry := ParseMod(comb, true)
				if combEntry.ModList != nil && strings.Trim(combEntry.Extra, " ") == "" {
					entry = combEntry
					for ci := i + 1; ci <= endI; ci++ {
						swallowed[ci] = true
					}
					break
				}
			}
		}

		if strings.Trim(entry.Extra, " ") != "" {
			env.DebugErrors = append(env.DebugErrors, "Error parsing "+what+" mod: "+entry.Extra+", stat text: "+line+", with "+strconv.Itoa(len(entry.ModList))+" mods found")
			continue
		}

		for _, m := range entry.ModList {
			modList.AddMod(mod.SetSource(m, mod.Source("Tree:"+nodeId)))
		}
	}
}

var keystoneMaps sync.Map

//pob:block:src/Classes/PassiveTree.lua:549-550
func keystoneMap(version pob.TreeVersion) map[string]data.Node {
	if cached, ok := keystoneMaps.Load(version); ok {
		return cached.(map[string]data.Node)
	}

	nodeMap := make(map[string]data.Node)
	for _, node := range data.TreeVersions[version].Tree().Nodes {
		if node.IsKeystone == nil || !*node.IsKeystone || node.Name == nil {
			continue
		}
		nodeMap[*node.Name] = node
		nodeMap[strings.ToLower(*node.Name)] = node
	}

	keystoneMaps.Store(version, nodeMap)
	return nodeMap
}

var keystoneModLists sync.Map

//pob:block:src/Classes/PassiveTree.lua:707-716
func keystoneModList(env *Environment, version pob.TreeVersion, name string) *moddb.ModList {
	type key struct {
		version pob.TreeVersion
		name    string
	}

	if cached, ok := keystoneModLists.Load(key{version, name}); ok {
		return cached.(*moddb.ModList)
	}

	node, ok := keystoneMap(version)[name]
	if !ok || node.Skill == nil {
		legionNode, isLegion := legionKeystoneByName(name)
		if !isLegion {
			return nil
		}

		modList := moddb.NewModList()
		if len(legionNode.Stats) > 0 {
			processStats(env, modList, legionNode.ID, "Passive Node ("+name+")", legionNode.Stats)
		}

		keystoneModLists.Store(key{version, name}, modList)
		return modList
	}

	nodeId := strconv.FormatInt(*node.Skill, 10)
	modList := moddb.NewModList()
	processStats(env, modList, nodeId, "Passive Node ("+name+")", node.Stats)

	keystoneModLists.Store(key{version, name}, modList)
	return modList
}

var legionKeystonesByName = sync.OnceValue(func() map[string]data.LegionNode {
	byName := make(map[string]data.LegionNode, len(data.LegionKeystones)+4)
	for _, node := range data.LegionKeystones {
		byName[node.DN] = node
	}
	for _, node := range data.LegionNodes {
		if _, seen := byName[node.DN]; seen {
			continue
		}
		for _, roll := range node.Rolls {
			if strings.HasPrefix(roll.Key, "keystone_") {
				byName[node.DN] = node
				break
			}
		}
	}
	return byName
})

//pob:block:src/Classes/PassiveTree.lua:712-716
func legionKeystoneByName(name string) (data.LegionNode, bool) {
	node, ok := legionKeystonesByName()[name]
	return node, ok
}

var classStartConnectionMaps sync.Map

//pob:block:src/Classes/PassiveTree.lua:313-361
func isNormalNode(node data.Node) bool {
	switch {
	case node.ClassStartIndex != nil,
		node.IsAscendancyStart != nil && *node.IsAscendancyStart,
		node.IsMastery != nil && *node.IsMastery,
		node.IsJewelSocket != nil && *node.IsJewelSocket,
		node.IsKeystone != nil && *node.IsKeystone,
		node.IsNotable != nil && *node.IsNotable:
		return false
	}
	return true
}

// for classId, class in pairs(self.classes) do
//
//	local startNode = nodeMap[class.startNodeId]
//	for _, nodeId in ipairs(startNode.linkedId) do
//		local node = nodeMap[nodeId]
//		if node.type == "Normal" then
//			node.modList:NewMod("Condition:ConnectedTo"..class.name.."Start", "FLAG", true, "Tree:"..nodeId)
//		end
//	end
//
// end
//
//pob:block:src/Classes/PassiveTree.lua:689-697
func classStartConnections(version pob.TreeVersion) map[string][]pob.ClassName {
	if cached, ok := classStartConnectionMaps.Load(version); ok {
		return cached.(map[string][]pob.ClassName)
	}

	tree := data.TreeVersions[version].Tree()

	startNodes := make(map[int64]data.Node, len(tree.Classes))
	for _, node := range tree.Nodes {
		if node.ClassStartIndex != nil {
			startNodes[*node.ClassStartIndex] = node
		}
	}

	connections := make(map[string][]pob.ClassName)
	for classID, class := range tree.Classes {
		startNode, ok := startNodes[int64(classID)]
		if !ok {
			continue
		}
		linked := make([]string, 0, len(startNode.Out)+len(startNode.In))
		linked = append(linked, startNode.Out...)
		linked = append(linked, startNode.In...)
		for _, linkedID := range linked {
			if node, exists := tree.Nodes[linkedID]; exists && isNormalNode(node) {
				connections[linkedID] = append(connections[linkedID], class.Name)
			}
		}
	}

	classStartConnectionMaps.Store(version, connections)
	return connections
}

func masterySelected(env *Environment, nodeId string) bool {
	nodeIDInt, err := strconv.ParseInt(nodeId, 10, 64)
	if err != nil || env.Spec == nil {
		return false
	}
	_, selected := env.Spec.MasterySelections[nodeIDInt]
	return selected
}

//pob:src/Modules/CalcSetup.lua:83-136
func buildModListForNode(env *Environment, nodeId string, node data.Node) *moddb.ModList {
	return buildModListForConqueredNode(env, nodeId, node, nodeRewrite{}, nil)
}

//pob:src/Modules/CalcSetup.lua:83-136,src/Classes/PassiveSpec.lua:1315-1351
func buildModListForConqueredNode(env *Environment, nodeId string, node data.Node, conquered nodeRewrite, jewels []*radiusJewel) *moddb.ModList {
	var modList = moddb.NewModList()

	//	local modList = new("ModList")
	//	modList:AddList(addition.modList)
	//	if not replacement then modList:AddList(node.modList) end
	for _, line := range conquered.addStats {
		if line == "" {
			continue
		}
		processStats(env, modList, nodeId, "Conquered Passive Node", []string{line})
	}

	stats := node.Stats
	statsSource := nodeId
	replacement := conquered.effectiveReplacement()
	if replacement != nil {
		stats = replacement.Stats
		statsSource = replacement.ID
		if replacement == conquered.replaceWith && conquered.reparsed {
			statsSource = nodeId
		}
		//pob:block:src/Classes/PassiveSpec.lua:1393-1403
		if replacement == conquered.tattoo && node.IsMastery != nil && *node.IsMastery && masterySelected(env, nodeId) {
			statsSource = nodeId
		}
	}

	if node.IsKeystone != nil && *node.IsKeystone {
		nodeName := ""
		if node.Name != nil {
			nodeName = *node.Name
		}
		keystoneSource := nodeId
		if replacement != nil {
			nodeName = replacement.DN
			keystoneSource = replacement.ID
		}
		modList.AddMod(mod.NewList("Keystone", nodeName).Source(mod.Source("Tree" + keystoneSource)))
	} else {
		if len(stats) > 0 {
			nodeName := ""
			if node.Name != nil {
				nodeName = *node.Name
			}
			processStats(env, modList, statsSource, "Passive Node ("+nodeName+")", stats)
		}

		if node.IsMastery != nil && *node.IsMastery && replacement == nil {
			if nodeIDInt, errID := strconv.ParseInt(nodeId, 10, 64); errID == nil {
				if effectID, selected := env.Spec.MasterySelections[nodeIDInt]; selected {
					for _, effect := range node.MasteryEffects {
						if effect.Effect != effectID {
							continue
						}
						processStats(env, modList, nodeId, "Mastery Effect ("+strconv.FormatInt(effectID, 10)+")", effect.Stats)
					}
				} else {
					stats := make([]string, 0, len(node.MasteryEffects))
					for _, effect := range node.MasteryEffects {
						stats = append(stats, effect.Stats...)
					}
					if len(stats) > 0 {
						processStats(env, modList, nodeId, "Mastery (all options, none selected)", stats)
					}
				}
			}
		}
	}

	treeVersion := data.LatestTreeVersion
	if env.Spec != nil {
		treeVersion = env.Spec.TreeVersion
	}
	for _, className := range classStartConnections(treeVersion)[nodeId] {
		modList.AddMod(mod.NewFlag("Condition:ConnectedTo"+string(className)+"Start", true).Source(mod.Source("Tree:" + nodeId)))
	}

	if node.ClusterJewelIncEffect != 0 {
		modList.NewMod("PassiveSkillEffect", mod.TypeIncrease, node.ClusterJewelIncEffect, mod.Source("Tree:"+nodeId))
	}

	var radNode *radiusJewelNode
	if inAnyRadius(jewels, nodeId) {
		snapshot := moddb.NewModList()
		snapshot.AddDB(modList)
		radNode = &radiusJewelNode{
			id:        nodeId,
			nodeType:  nodeTypeOf(node),
			modList:   snapshot,
			conquered: conquered.stamped,
			isTattoo:  conquered.replaceWith == nil && conquered.tattoo != nil,
		}
	}

	if radNode != nil && radNode.nodeType != "Mastery" {
		for _, rad := range jewels {
			if rad.funcType == "Other" && rad.nodes[nodeId] {
				rad.fn(radNode, modList, rad.data)
			}
		}
	}

	_, allocated := env.AllocatedNodes[nodeId]
	if modList.Flag(nil, "PassiveSkillHasNoEffect") ||
		(allocated && modList.Flag(nil, "AllocatedPassiveSkillHasNoEffect")) {
		modList.SetMods(nil)
	}

	if scale := calclib.Mod(modList, nil, "PassiveSkillEffect"); scale != 1 {
		scaledList := moddb.NewModList()
		scaledList.ScaleAddList(modList, scale)
		modList = scaledList
	}

	if radNode != nil && radNode.nodeType != "Mastery" {
		for _, rad := range jewels {
			if !rad.nodes[nodeId] {
				continue
			}
			if rad.funcType == "Threshold" ||
				(rad.funcType == "Self" && allocated) ||
				(rad.funcType == "SelfUnalloc" && !allocated) {
				rad.fn(radNode, modList, rad.data)
			}
		}
	}

	if modList.Flag(nil, "PassiveSkillHasOtherEffect") {
		if replacements := modList.List(nil, "NodeModifier"); len(replacements) > 0 {
			modList.SetMods(nil)
			for _, replacement := range replacements {
				if nm, ok := replacement.(nodeModifier); ok && nm.Mod != nil {
					modList.AddMod(nm.Mod)
				}
			}
		}
	}

	return modList
}
