package calculator2

import (
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// radiusJewel is one entry of env.radiusJewelList.
type radiusJewel struct {
	nodes       map[int]*tree2.Node
	fn          radiusJewelFunc
	typ         string
	item        *Item
	nodeID      int
	data        *radiusJewelData
	jewelHash   string
	effectScale float64
}

// radiusJewelCacheEntry is GlobalCache.cachedData[env.mode].radiusJewelData[nodeId].
type radiusJewelCacheEntry struct {
	hash             string
	smallModList     *moddb.ModList
	attributeModList *moddb.ModList
	notableModList   *moddb.ModList
}

//pob2:src/Modules/CalcSetup.lua:736-743
func defaultRadiusJewelFunc(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
	if node != nil {
		for _, stat := range []string{"Str", "Dex", "Int"} {
			d.stats[stat] += out.Sum(mod.TypeBase, nil, stat)
		}
	}
}

func (env *Environment) radiusNode(node *tree2.SpecNode) *radiusJewelNode {
	return &radiusJewelNode{id: node.ID, nodeType: node.Type, isAttribute: node.IsAttribute, modList: env.nodeModList(node)}
}

// The parsedLine rewrite is display-only and is not ported.
//
//pob2:src/Modules/CalcSetup.lua:128-165
func runRadiusJewelFunc(rad *radiusJewel, node *radiusJewelNode, out *moddb.ModList, data *radiusJewelData) {
	scale := rad.effectScale
	if scale == 0 || scale == 1 {
		rad.fn(node, out, data)
		return
	}

	start := len(out.Mods())
	rad.fn(node, out, data)
	mods := out.Mods()
	if len(mods) == start {
		return
	}

	scaledList := newModList()
	for _, m := range mods[start:] {
		scaledList.AddMod(m)
	}
	out.SetMods(append([]mod.Mod(nil), mods[:start]...))
	out.ScaleAddList(scaledList, scale)
}

//pob2:src/Modules/CalcSetup.lua:167-185
func refreshJewelStatCache(env *Environment) {
	normalNode := &radiusJewelNode{nodeType: "Normal", modList: newModList()}
	attributeNode := &radiusJewelNode{nodeType: "Normal", isAttribute: true, modList: newModList()}
	notableNode := &radiusJewelNode{nodeType: "Notable", modList: newModList()}
	env.radiusJewelData = map[int]*radiusJewelCacheEntry{}

	for _, rad := range env.radiusJewelList {
		cache := env.radiusJewelData[rad.nodeID]
		if cache == nil {
			cache = &radiusJewelCacheEntry{
				hash:             rad.jewelHash,
				smallModList:     newModList(),
				attributeModList: newModList(),
				notableModList:   newModList(),
			}
			env.radiusJewelData[rad.nodeID] = cache
		}
		runRadiusJewelFunc(rad, normalNode, cache.smallModList, rad.data)
		runRadiusJewelFunc(rad, attributeNode, cache.attributeModList, rad.data)
		runRadiusJewelFunc(rad, notableNode, cache.notableModList, rad.data)
	}
}

func copyList(dst, src *moddb.ModList) {
	for _, m := range src.Mods() {
		dst.AddMod(m.Clone())
	}
}

// radiusFirstPass runs the "Other" jewel functions over a node's own mods; Time-Lost and Timeless
// jewels copy the cached prototype lists instead.
//
//pob2:block:src/Modules/CalcSetup.lua:215-238
func radiusFirstPass(env *Environment, node *tree2.SpecNode, modList *moddb.ModList) {
	if len(env.radiusJewelList) > 0 && env.radiusJewelData == nil {
		refreshJewelStatCache(env)
	}
	for _, rad := range env.radiusJewelList {
		inRadius, ok := rad.nodes[node.ID]
		if rad.typ != "Other" || !ok || inRadius.Type == "Mastery" {
			continue
		}
		if !strings.Contains(rad.item.BaseName, "Time-Lost") && !strings.Contains(rad.item.BaseName, "Timeless Jewel") {
			runRadiusJewelFunc(rad, env.radiusNode(node), modList, rad.data)
		} else if node.Type == "Normal" || node.Type == "Notable" {
			// Lua keeps the cache entry it read before a refresh.
			cache := env.radiusJewelData[rad.nodeID]
			if cache == nil || cache.hash != rad.jewelHash {
				refreshJewelStatCache(env)
			}
			switch {
			case node.Type == "Normal" && node.IsAttribute && cache != nil && len(cache.attributeModList.Mods()) > 0:
				copyList(modList, cache.attributeModList)
			case node.Type == "Normal" && !node.IsAttribute && cache != nil && len(cache.smallModList.Mods()) > 0:
				copyList(modList, cache.smallModList)
			case node.Type == "Notable" && cache != nil && len(cache.notableModList.Mods()) > 0:
				copyList(modList, cache.notableModList)
			}
			break
		}
	}
}

// radiusSecondPass reports whether any jewel function ran (PoB's rescan).
//
//pob2:block:src/Modules/CalcSetup.lua:277-288
func radiusSecondPass(env *Environment, node *tree2.SpecNode, modList *moddb.ModList) bool {
	rescan := false
	_, allocated := env.AllocNodes[node.ID]
	for _, rad := range env.radiusJewelList {
		inRadius, ok := rad.nodes[node.ID]
		if ok && inRadius.Type != "Mastery" && (rad.typ == "Threshold" || (rad.typ == "Self" && allocated) || (rad.typ == "SelfUnalloc" && !allocated)) {
			runRadiusJewelFunc(rad, env.radiusNode(node), modList, rad.data)
			rescan = true
		}
	}
	return rescan
}

//pob2:block:src/Modules/CalcSetup.lua:416-420
func initRadiusJewels(env *Environment) {
	for _, rad := range env.radiusJewelList {
		rad.data = newRadiusJewelData(mod.Source("Tree:" + strconv.Itoa(rad.nodeID)))
	}
}

// finishRadiusJewels is the finishJewels block; both of PoB's callers pass finishJewels. A node
// allocated as a granted passive after it joined extraRadiusNodeList is skipped, as
// addGrantedPassiveNode removes it from the list.
//
//pob2:block:src/Modules/CalcSetup.lua:442-461
func finishRadiusJewels(env *Environment, modList *moddb.ModList, inc float64) {
	ids := make([]int, 0, len(env.extraRadiusNodeList))
	for id := range env.extraRadiusNodeList {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if _, ok := env.AllocNodes[id]; ok {
			continue
		}
		buildModListForNode(env, env.extraRadiusNodeList[id], inc, false)
	}

	for _, rad := range env.radiusJewelList {
		runRadiusJewelFunc(rad, nil, modList, rad.data)
	}
}

// addRadiusJewels adds a socketed jewel's functions to env.radiusJewelList. The override's
// extraJewelFuncs are not ported.
//
//pob2:block:src/Modules/CalcSetup.lua:1293-1316
func addRadiusJewels(env *Environment, slotNodeID int, item *Item) {
	node := env.Spec.Nodes[slotNodeID]
	if item == nil || (node != nil && node.Sinister) || item.JewelRadiusIndex == 0 {
		return
	}
	var funcList []jewelFuncEntry
	if fl, ok := item.JewelData["funcList"].([]any); ok {
		for _, v := range fl {
			switch e := v.(type) {
			case jewelFuncEntry:
				funcList = append(funcList, e)
			case *jewelFuncEntry:
				funcList = append(funcList, *e)
			}
		}
	} else {
		funcList = []jewelFuncEntry{{Type: "Self", Func: defaultRadiusJewelFunc}}
	}
	var nodes map[int]*tree2.Node
	if node != nil && node.NodesInRadius != nil && item.JewelRadiusIndex <= len(node.NodesInRadius) {
		nodes = node.NodesInRadius[item.JewelRadiusIndex-1]
	}
	for _, f := range funcList {
		env.radiusJewelList = append(env.radiusJewelList, &radiusJewel{
			nodes:     nodes,
			fn:        f.Func,
			typ:       f.Type,
			item:      item,
			nodeID:    slotNodeID,
			data:      newRadiusJewelData(""),
			jewelHash: item.modSourceName + item.Raw,
		})
		if f.Type != "Self" && nodes != nil {
			for nodeID := range nodes {
				if _, ok := env.AllocNodes[nodeID]; ok {
					continue
				}
				if n := env.Spec.Nodes[nodeID]; n != nil {
					if env.extraRadiusNodeList == nil {
						env.extraRadiusNodeList = map[int]*tree2.SpecNode{}
					}
					env.extraRadiusNodeList[nodeID] = n
				}
			}
		}
	}
}

//pob2:block:src/Modules/CalcSetup.lua:1350-1356
func scaleRadiusJewels(env *Environment) {
	for _, rad := range env.radiusJewelList {
		effect := getCorruptedJewelEffect(env, rad.item, env.Spec.Nodes[rad.nodeID])
		if effect != 0 {
			rad.effectScale = 1 + effect
			rad.jewelHash = rad.jewelHash + ":" + strconv.FormatFloat(rad.effectScale, 'g', -1, 64)
		}
	}
}
