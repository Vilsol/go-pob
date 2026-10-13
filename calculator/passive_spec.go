package calculator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	tjcalc "github.com/Vilsol/timeless-jewels/calculator"
	tjdata "github.com/Vilsol/timeless-jewels/data"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

var masteryEffectsRegex = regexp.MustCompile(`\{(\d+),(\d+)\}`)

type PassiveSpec struct {
	Build       *pob.PathOfBuilding
	TreeVersion pob.TreeVersion

	Nodes              map[string]interface{}
	AllocNodes         map[string]data.Node
	AllocSubgraphNodes map[string]interface{}
	AllocExtendedNodes map[string]interface{}
	Jewels             map[string]interface{}
	SubGraphs          map[string]interface{}
	MasterySelections  map[int64]int64

	ClassID        pob.ClassID
	ClassName      pob.ClassName
	AscendancyID   pob.AscendancyID
	AscendancyName pob.AscendancyName

	AllocatedNotableCount int
	AllocatedMasteryCount int

	DistanceToClassStart map[string]int
}

//pob:src/Classes/PassiveSpec.lua:19-28,src/Classes/PassiveSpec.lua:30-90
func NewPassiveSpec(build *pob.PathOfBuilding, treeVersion pob.TreeVersion) *PassiveSpec {
	passiveSpec := &PassiveSpec{
		Build:       build,
		TreeVersion: treeVersion,
	}

	classID := pob.Scion
	passiveSpec.MasterySelections = make(map[int64]int64)
	if specs := build.Tree.Specs; len(specs) > 0 {
		idx := build.Tree.ActiveSpec - 1
		if idx < 0 || idx >= len(specs) {
			idx = 0
		}
		classID = specs[idx].ClassID

		for _, match := range masteryEffectsRegex.FindAllStringSubmatch(specs[idx].MasteryEffects, -1) {
			nodeID, errNode := strconv.ParseInt(match[1], 10, 64)
			effectID, errEffect := strconv.ParseInt(match[2], 10, 64)
			if errNode == nil && errEffect == nil {
				passiveSpec.MasterySelections[nodeID] = effectID
			}
		}
	}
	passiveSpec.SelectClass(classID)

	return passiveSpec
}

//pob:block:src/Classes/PassiveSpec.lua:32
func (p *PassiveSpec) Tree() *data.Tree {
	return data.TreeVersions[p.TreeVersion].Tree()
}

//pob:block:src/Classes/PassiveSpec.lua:573-574
func (p *PassiveSpec) Class() data.Class {
	return p.Tree().Classes[p.ClassID]
}

//pob:src/Classes/PassiveSpec.lua:562-585
func (p *PassiveSpec) SelectClass(classID pob.ClassID) {
	/*
		TODO Implement
		if self.curClassId then
			-- Deallocate the current class's starting node
			local oldStartNodeId = self.curClass.startNodeId
			self.nodes[oldStartNodeId].alloc = false
			self.allocNodes[oldStartNodeId] = nil
		end
	*/

	p.ClassID = classID
	p.ClassName = pob.ClassNameByID[classID]

	/*
		TODO Implement
		-- Allocate the new class's starting node
		local startNode = self.nodes[class.startNodeId]
		startNode.alloc = true
		self.allocNodes[startNode.id] = startNode
	*/

	p.SelectAscendancyClass(pob.ClassAscendancies[p.ClassID][0])
}

//pob:src/Classes/PassiveSpec.lua:599-617
func (p *PassiveSpec) SelectAscendancyClass(ascendancyID pob.AscendancyID) {
	p.AscendancyID = ascendancyID
	p.AscendancyName = pob.AscendancyNameByID[ascendancyID]

	/*
		TODO Implement
		-- Deallocate any allocated ascendancy nodes that don't belong to the new ascendancy class
		for id, node in pairs(self.allocNodes) do
			if node.ascendancyName and node.ascendancyName ~= ascendClass.name then
				node.alloc = false
				self.allocNodes[id] = nil
			end
		end
	*/

	/*
		TODO Implement
		if ascendClass.startNodeId then
			-- Allocate the new ascendancy class's start node
			local startNode = self.nodes[ascendClass.startNodeId]
			startNode.alloc = true
			self.allocNodes[startNode.id] = startNode
		end
	*/

	/*
		TODO Implement
		self:BuildAllDependsAndPaths()
	*/
}

var legionAttributeNodes = map[string]bool{
	"Dexterity":    true,
	"Intelligence": true,
	"Strength":     true,
}

type nodeRewrite struct {
	tattoo  *data.LegionNode
	stamped bool

	addStats []string

	replaceWith *data.LegionNode

	reparsed bool
}

func (r nodeRewrite) effectiveReplacement() *data.LegionNode {
	if r.replaceWith != nil {
		return r.replaceWith
	}
	return r.tattoo
}

func tattooedNodes(env *Environment) map[string]*data.LegionNode {
	specs := env.Build.Tree.Specs
	if len(specs) == 0 {
		return nil
	}
	idx := env.Build.Tree.ActiveSpec - 1
	if idx < 0 || idx >= len(specs) {
		idx = 0
	}

	overrides := specs[idx].Overrides
	if len(overrides) == 0 {
		return nil
	}

	tattoos := make(map[string]*data.LegionNode, len(overrides))
	for _, o := range overrides {
		row, found := data.TattooNodes[o.DN]
		if !found {
			continue
		}
		replacement := row
		tattoos[strconv.FormatInt(int64(o.NodeID), 10)] = &replacement
	}
	return tattoos
}

type socketedLegionJewel struct {
	socketNodeID int64
	radiusIndex  int
	jewel        mod.LegionJewel
}

//pob:block:src/Classes/PassiveSpec.lua:1146-1148
func socketedLegionJewels(env *Environment) []socketedLegionJewel {
	specs := env.Build.Tree.Specs
	if len(specs) == 0 {
		return nil
	}
	idx := env.Build.Tree.ActiveSpec - 1
	if idx < 0 || idx >= len(specs) {
		idx = 0
	}

	sockets, byID := overrideSockets(env, specs[idx].Sockets, itemTextByID(env))
	if len(sockets) == 0 {
		return nil
	}

	order := make([]int, 0, len(sockets))
	for i := range sockets {
		order = append(order, i)
	}
	sort.SliceStable(order, func(a, b int) bool { return sockets[order[a]].NodeID < sockets[order[b]].NodeID })

	jewels := make([]socketedLegionJewel, 0, 1)
	for _, i := range order {
		socket := sockets[i]
		if _, allocated := env.AllocatedNodes[strconv.Itoa(socket.NodeID)]; !allocated {
			continue
		}
		text, ok := byID[socket.ItemID]
		if !ok {
			continue
		}
		item := peekItemCached(env.Cache, text)
		if item.Type != "Jewel" {
			continue
		}
		conqueredBy, ok := itemConqueredBy(item)
		if !ok {
			continue
		}
		radiusIndex, ok := data.JewelRadiusIndexForLabel(item.JewelRadiusLabel)
		if !ok {
			continue
		}
		jewels = append(jewels, socketedLegionJewel{
			socketNodeID: int64(socket.NodeID),
			radiusIndex:  radiusIndex,
			jewel:        conqueredBy,
		})
	}
	return jewels
}

//pob:block:src/Classes/Item.lua:2425-2427
func itemConqueredBy(item *Item) (mod.LegionJewel, bool) {
	for _, m := range item.Mods {
		if m.Name() != "JewelData" {
			continue
		}
		value, ok := m.Value().List().(mod.JewelData)
		if !ok || value.Key != "conqueredBy" {
			continue
		}
		if jewel, isLegion := value.Value.(mod.LegionJewel); isLegion {
			return jewel, true
		}
	}
	return mod.LegionJewel{}, false
}

//pob:block:src/Classes/PassiveSpec.lua:1185-1192,src/Classes/PassiveSpec.lua:1334-1372
func nodeRewrites(env *Environment) map[string]nodeRewrite {
	tattoos := tattooedNodes(env)

	jewels := socketedLegionJewels(env)

	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	if env.Spec != nil {
		tree = env.Spec.Tree()
	}

	if len(jewels) == 0 {
		if len(tattoos) == 0 && len(abyssSockets(env)) == 0 {
			return nil
		}
		rewrites := make(map[string]nodeRewrite, len(tattoos))
		for nodeID, row := range tattoos {
			rewrites[nodeID] = nodeRewrite{tattoo: row}
		}
		applyAbyssConquests(env, tree, rewrites)
		return rewrites
	}

	radiusNodes := tree.JewelRadiusNodes()

	conqueredBy := make(map[string]mod.LegionJewel)
	for _, j := range jewels {
		if isExpansionSocket(tree, strconv.FormatInt(j.socketNodeID, 10)) {
			continue
		}
		if t := legionJewelType(j.jewel.Conqueror.Type); t >= 7 {
			continue
		}

		perRadius, ok := radiusNodes[j.socketNodeID]
		if !ok || j.radiusIndex < 0 || j.radiusIndex >= len(perRadius) {
			continue
		}
		for _, id := range perRadius[j.radiusIndex] {
			nodeID := strconv.FormatInt(id, 10)
			if node, exists := tree.Nodes[nodeID]; exists {
				switch nodeTypeOf(node) {
				case "ClassStart", "Socket":
					continue
				}
			}
			conqueredBy[nodeID] = j.jewel
		}
	}

	transforms := make(map[string]nodeRewrite, len(conqueredBy)+len(tattoos))
	for nodeID, row := range tattoos {
		transforms[nodeID] = nodeRewrite{tattoo: row}
	}
	for nodeID, jewel := range conqueredBy {
		tattoo := tattoos[nodeID]
		transforms[nodeID] = nodeRewrite{stamped: true, tattoo: tattoo}

		node, ok := tree.Nodes[nodeID]
		if !ok {
			continue
		}

		if nodeTypeOf(node) == "Keystone" {
			matchStr := jewel.Conqueror.Type + "_keystone_" + jewel.Conqueror.ID
			if legionNode, found := data.LegionKeystones[matchStr]; found {
				replacement := legionNode
				transforms[nodeID] = nodeRewrite{stamped: true, tattoo: tattoo, replaceWith: &replacement}
			}
			continue
		}

		if nodeTypeOf(node) == "Notable" {
			if replaceWith, added := legionReplacement(jewel, nodeID); replaceWith != nil || len(added) > 0 {
				transforms[nodeID] = nodeRewrite{
					stamped: true, tattoo: tattoo,
					replaceWith: replaceWith, addStats: added,
				}
			}
			continue
		}

		if !isNormalNode(node) {
			continue
		}

		isAttribute := node.Name != nil && legionAttributeNodes[*node.Name]
		if tattoo != nil {
			isAttribute = true
		}

		switch jewel.Conqueror.Type {
		case "vaal":
			if replaceWith, added := legionReplacement(jewel, nodeID); replaceWith != nil || len(added) > 0 {
				transforms[nodeID] = nodeRewrite{
					stamped: true, tattoo: tattoo,
					replaceWith: replaceWith, addStats: added,
					reparsed: replaceWith != nil,
				}
			}
		case "karui":
			transforms[nodeID] = nodeRewrite{stamped: true, tattoo: tattoo, addStats: []string{legionAttributeLine(isAttribute, "Strength")}}
		case "maraketh":
			transforms[nodeID] = nodeRewrite{stamped: true, tattoo: tattoo, addStats: []string{legionAttributeLine(isAttribute, "Dexterity")}}
		case "templar":
			if isAttribute {
				transforms[nodeID] = nodeRewrite{stamped: true, tattoo: tattoo, replaceWith: &data.LegionTemplarDevotionNode}
			} else {
				transforms[nodeID] = nodeRewrite{stamped: true, tattoo: tattoo, addStats: []string{"+5 to Devotion"}}
			}
		case "kalguur":
			transforms[nodeID] = nodeRewrite{stamped: true, tattoo: tattoo, addStats: []string{legionWardLine(isAttribute)}}
		case "eternal":
			transforms[nodeID] = nodeRewrite{stamped: true, tattoo: tattoo, replaceWith: &data.LegionEternalSmallBlank}
		}
	}

	applyAbyssConquests(env, tree, transforms)

	return transforms
}

func applyAbyssConquests(env *Environment, tree *data.Tree, transforms map[string]nodeRewrite) {
	sockets := abyssSockets(env)
	if len(sockets) == 0 {
		return
	}

	treeVersion := data.LatestTreeVersion
	if env.Spec != nil {
		treeVersion = env.Spec.TreeVersion
	}

	ascendancies := make([]string, 0, 2)
	for _, node := range env.AllocatedNodes {
		if node.AscendancyName != nil && !slices.Contains(ascendancies, *node.AscendancyName) {
			ascendancies = append(ascendancies, *node.AscendancyName)
		}
	}
	sort.Strings(ascendancies)

	type conquest struct {
		replaceWith *data.LegionNode
		addStats    []string
	}
	conquests := make(map[string]conquest)
	first := map[string]string{}
	if env.Cache != nil {
		for _, pick := range env.Cache.pairsPicks[pairsPickAbyssStats] {
			if legionID, statKey, ok := strings.Cut(pick, "="); ok {
				first[legionID] = statKey
			}
		}
	}
	env.abyssStatTies = map[string]abyssStatTie{}
	render := func(nodeID string) func(data.LegionNode, tjdata.StatRolls) []string {
		return func(node data.LegionNode, rolls tjdata.StatRolls) []string {
			if tie, ok := abyssStatTieOf(node, rolls); ok {
				env.abyssStatTies[nodeID] = tie
			}
			return substituteAbyssRolls(node, rolls, first[node.ID])
		}
	}
	for _, socket := range sockets {
		seed := socket.jewel.ID
		if socket.jewelType != tjdata.AbyssZorath {
			for _, nodeID := range abyssSocketConquests(treeVersion, socket.jewelType, socket.socketID, seed) {
				replaceWith, added, _ := abyssModification(socket.jewelType, seed, nodeID, render(nodeID))
				conquests[nodeID] = conquest{replaceWith: replaceWith, addStats: added}
			}
			continue
		}
		for nodeID := range shortestPathToClassStart(env, tree, socket.socketID) {
			node, ok := tree.Nodes[nodeID]
			if !ok || (node.IsProxy != nil && *node.IsProxy) {
				continue
			}
			switch nodeTypeOf(node) {
			case "ClassStart", "AscendClassStart", "Mastery":
				continue
			}
			if replaceWith, added, ok := abyssModification(tjdata.AbyssZorath, seed, nodeID, render(nodeID)); ok {
				conquests[nodeID] = conquest{replaceWith: replaceWith, addStats: added}
			}
		}
		for _, ascendancy := range ascendancies {
			nodeID, ok := abyssZorathAscendancyNotable(treeVersion, seed, ascendancy)
			if !ok {
				continue
			}
			replaceWith, added, _ := abyssModification(tjdata.AbyssZorath, seed, nodeID, render(nodeID))
			conquests[nodeID] = conquest{replaceWith: replaceWith, addStats: added}
		}
	}

	for nodeID, c := range conquests {
		existing := transforms[nodeID]
		if existing.stamped || env.GrantedAscendancyNodes[nodeID] {
			continue
		}
		rewrite := nodeRewrite{stamped: true, tattoo: existing.tattoo}
		if node, ok := tree.Nodes[nodeID]; ok && nodeTypeOf(node) != "Socket" {
			rewrite.replaceWith = c.replaceWith
			rewrite.addStats = c.addStats
			rewrite.reparsed = c.replaceWith != nil
		}
		transforms[nodeID] = rewrite
	}
}

func abyssSeedInRange(jewelType tjdata.JewelType, seed int) bool {
	r := tjdata.TimelessJewelSeedRanges[jewelType]
	return seed >= int(r.Min) && seed <= int(r.Max)
}

func abyssSocketConquests(treeVersion pob.TreeVersion, jewelType tjdata.JewelType, socketID string, seed int) []string {
	if !abyssSeedInRange(jewelType, seed) {
		return nil
	}
	socket, err := strconv.ParseUint(socketID, 10, 32)
	if err != nil {
		return nil
	}
	abyssTreeMu.Lock()
	defer abyssTreeMu.Unlock()
	installAbyssSkillTree(treeVersion)
	ids, ok := tjcalc.AbyssConqueredNodes(uint32(socket), uint32(seed), jewelType) //nolint:gosec // both guarded above
	if !ok {
		return nil
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = strconv.FormatUint(uint64(id), 10)
	}
	return out
}

//pob:block:src/Classes/PassiveSpec.lua:1222-1238
func abyssModification(jewelType tjdata.JewelType, seed int, graphID string, render func(data.LegionNode, tjdata.StatRolls) []string) (*data.LegionNode, []string, bool) {
	if err := raw.EnsureTimelessSources(); err != nil {
		return nil, nil, false
	}
	id, err := strconv.ParseInt(graphID, 10, 64)
	if err != nil {
		return nil, nil, false
	}
	skill := raw.PassiveSkillByGraphID(id)
	if skill == nil || skill.Key < 0 || !abyssSeedInRange(jewelType, seed) {
		return nil, nil, false
	}
	result := tjcalc.Calculate(uint32(skill.Key), uint32(seed), jewelType, tjdata.Abyss) //nolint:gosec // both guarded above
	if result.AlternatePassiveSkill == nil && len(result.AlternatePassiveAdditionInformations) == 0 {
		return nil, nil, false
	}
	replaceWith, added := renderTimelessResult(result, render)
	return replaceWith, added, true
}

var (
	abyssTreeMu      sync.Mutex
	abyssTreeVersion pob.TreeVersion
)

func abyssZorathAscendancyNotable(treeVersion pob.TreeVersion, seed int, ascendancy string) (string, bool) {
	if !abyssSeedInRange(tjdata.AbyssZorath, seed) {
		return "", false
	}
	abyssTreeMu.Lock()
	defer abyssTreeMu.Unlock()
	installAbyssSkillTree(treeVersion)
	graphID, ok := tjcalc.AbyssAscendancyNotable(uint32(seed), ascendancy) //nolint:gosec // guarded above
	if !ok {
		return "", false
	}
	return strconv.FormatUint(uint64(graphID), 10), true
}

func installAbyssSkillTree(treeVersion pob.TreeVersion) {
	if abyssTreeVersion == treeVersion && len(tjdata.SkillTreeData.Nodes) != 0 {
		return
	}
	var tree struct {
		Nodes map[string]tjdata.Node `json:"nodes"`
	}
	if err := json.Unmarshal(data.TreeVersions[treeVersion].RawTree(), &tree); err != nil {
		panic(fmt.Errorf("failed to decode tree %s for the Abyss walks: %w", treeVersion, err))
	}
	tjdata.SkillTreeData.Nodes = tree.Nodes
	abyssTreeVersion = treeVersion
}

type abyssSocket struct {
	socketID  string
	jewelType tjdata.JewelType
	jewel     mod.LegionJewel
}

//pob:block:src/Classes/PassiveSpec.lua:1117-1123
func abyssSockets(env *Environment) []abyssSocket {
	specs := env.Build.Tree.Specs
	if len(specs) == 0 {
		return nil
	}
	idx := env.Build.Tree.ActiveSpec - 1
	if idx < 0 || idx >= len(specs) {
		idx = 0
	}

	sockets, byID := overrideSockets(env, specs[idx].Sockets, itemTextByID(env))

	out := make([]abyssSocket, 0, 1)
	for _, socket := range sockets {
		socketID := strconv.Itoa(socket.NodeID)
		if _, allocated := env.AllocatedNodes[socketID]; !allocated {
			continue
		}
		text, ok := byID[socket.ItemID]
		if !ok {
			continue
		}
		item := peekItemCached(env.Cache, text)
		if item.Type != "Jewel" {
			continue
		}
		conqueredBy, found := itemConqueredBy(item)
		jewelType := legionJewelType(conqueredBy.Conqueror.Type)
		if !found || jewelType < 7 {
			continue
		}
		out = append(out, abyssSocket{socketID: socketID, jewelType: tjdata.JewelType(jewelType), jewel: conqueredBy}) //nolint:gosec // 7..11
	}
	sort.Slice(out, func(a, b int) bool { return out[a].socketID < out[b].socketID })
	return out
}

//pob:block:src/Classes/PassiveSpec.lua:991-1032
func shortestPathToClassStart(env *Environment, tree *data.Tree, rootID string) map[string]bool {
	if _, allocated := env.AllocatedNodes[rootID]; !allocated {
		return nil
	}
	target := classStartNodeID(tree, env.Spec.ClassID)
	if target == "" {
		return nil
	}

	parent := make(map[string]string)
	queue := []string{rootID}
	for o := 0; o < len(queue); o++ {
		id := queue[o]
		node, ok := env.AllocatedNodes[id]
		if !ok {
			node, ok = tree.Nodes[id]
		}
		if !ok {
			continue
		}
		for _, other := range linkedNodeIDs(node) {
			if other == target {
				path := map[string]bool{rootID: true, other: true}
				for cur := id; cur != ""; cur = parent[cur] {
					path[cur] = true
				}
				return path
			}
			otherNode, allocated := env.AllocatedNodes[other]
			if !allocated {
				continue
			}
			if isMasteryNode(node) || isClassStartNode(otherNode) || isAscendClassStartNode(otherNode) {
				continue
			}
			if _, seen := parent[other]; seen || other == rootID {
				continue
			}
			parent[other] = id
			queue = append(queue, other)
		}
	}
	return nil
}

//pob:block:src/Classes/PassiveSpec.lua:1203-1220
func legionStatLine(template string, stat data.LegionStat, value float64) string {
	if stat.Key == "" {
		return template
	}

	if stat.Fmt == "g" {
		switch {
		case strings.Contains(stat.Key, "per_minute"):
			value = utils.RoundTo(value/60, 1)
		case strings.Contains(stat.Key, "permyriad"):
			value /= 100
		case strings.Contains(stat.Key, "_ms"):
			value /= 1000
		}
	}

	if stat.Min != stat.Max {
		return strings.ReplaceAll(template,
			"("+luaNum(stat.Min)+"-"+luaNum(stat.Max)+")", luaNum(value))
	}
	if stat.Min != value {
		return strings.ReplaceAll(template, luaNum(stat.Min), luaNum(value))
	}
	return template
}

func luaNum(v float64) string {
	return fmt.Sprintf("%.14g", v)
}

// local ward = (isValueInArray(attributes, node.dn) or node.isTattoo) and "1" or "2"
//
//pob:block:src/Classes/PassiveSpec.lua:1360-1362
func legionWardLine(isAttribute bool) string {
	if isAttribute {
		return "1% increased Ward"
	}
	return "2% increased Ward"
}

// local str = isValueInArray(attributes, node.dn) and "2" or "4"
//
//pob:block:src/Classes/PassiveSpec.lua:1354-1359
func legionAttributeLine(isAttribute bool, attribute string) string {
	if isAttribute {
		return "+2 to " + attribute
	}
	return "+4 to " + attribute
}

//pob:block:src/Classes/PassiveSpec.lua:993-1008
func buildDistanceToClassStart(allocNodes map[string]data.Node, classStartID string) map[string]int {
	distances := make(map[string]int)
	if classStartID == "" {
		return distances
	}

	hops := make(map[string]int, len(allocNodes))
	queue := make([]string, 0, len(allocNodes))

	for id, node := range allocNodes {
		if isClassStartNode(node) || isAscendClassStartNode(node) {
			continue
		}
		for _, other := range linkedNodeIDs(node) {
			if other == classStartID {
				hops[id] = 1
				queue = append(queue, id)
				break
			}
		}
	}

	for o := 0; o < len(queue); o++ {
		cur := queue[o]
		next := hops[cur] + 1
		for _, other := range linkedNodeIDs(allocNodes[cur]) {
			otherNode, allocated := allocNodes[other]
			if !allocated {
				continue
			}
			if isMasteryNode(otherNode) || isClassStartNode(otherNode) || isAscendClassStartNode(otherNode) {
				continue
			}
			if _, visited := hops[other]; visited {
				continue
			}
			hops[other] = next
			queue = append(queue, other)
		}
	}

	for id, node := range allocNodes {
		if node.IsJewelSocket == nil && node.ExpansionJewel == nil {
			continue
		}
		distances[id] = max(hops[id]-1, 0)
	}

	return distances
}

//pob:src/Classes/PassiveSpec.lua:558-598
func setNodeDistanceToClassStart(root string, allocNodes map[string]data.Node, classStartID string) int {
	if _, allocated := allocNodes[root]; !allocated {
		return 0
	}

	nodeDistanceToRoot := map[string]int{root: 0}
	queueIDs := []string{root}

	for o := 0; o < len(queueIDs); o++ {
		cur := allocNodes[queueIDs[o]]
		curDist := nodeDistanceToRoot[queueIDs[o]] + 1
		for _, other := range linkedNodeIDs(cur) {
			if other == classStartID {
				return curDist - 1
			}
			otherNode, allocated := allocNodes[other]
			if !allocated {
				continue
			}
			if isMasteryNode(cur) || isClassStartNode(otherNode) || isAscendClassStartNode(otherNode) {
				continue
			}
			if _, visited := nodeDistanceToRoot[other]; visited {
				continue
			}
			nodeDistanceToRoot[other] = curDist
			queueIDs = append(queueIDs, other)
		}
	}

	return 0
}

//pob:block:src/Classes/PassiveTree.lua:521-532
func isMasteryNode(node data.Node) bool {
	return node.IsMastery != nil && *node.IsMastery
}

func isClassStartNode(node data.Node) bool {
	return node.ClassStartIndex != nil
}

func isAscendClassStartNode(node data.Node) bool {
	return node.IsAscendancyStart != nil && *node.IsAscendancyStart
}

//pob:block:src/Classes/PassiveTree.lua:592,src/Classes/PassiveTree.lua:395
func linkedNodeIDs(node data.Node) []string {
	linked := make([]string, 0, len(node.Out)+len(node.In))
	linked = append(linked, node.Out...)
	return append(linked, node.In...)
}

var classStartIDs sync.Map

//pob:block:src/Classes/PassiveTree.lua:522-525
func classStartNodeID(tree *data.Tree, classID pob.ClassID) string {
	starts, ok := classStartIDs.Load(tree)
	if !ok {
		starts, _ = classStartIDs.LoadOrStore(tree, uniqueClassStarts(tree))
	}
	if id, unique := starts.(map[int64]string)[int64(classID)]; unique {
		return id
	}
	for id, node := range tree.Nodes {
		if node.ClassStartIndex != nil && *node.ClassStartIndex == int64(classID) {
			return id
		}
	}
	return ""
}

func uniqueClassStarts(tree *data.Tree) map[int64]string {
	starts := map[int64]string{}
	count := map[int64]int{}
	for id, node := range tree.Nodes {
		if node.ClassStartIndex != nil {
			starts[*node.ClassStartIndex] = id
			count[*node.ClassStartIndex]++
		}
	}
	for class, n := range count {
		if n > 1 {
			delete(starts, class)
		}
	}
	return starts
}
