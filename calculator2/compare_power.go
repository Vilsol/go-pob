package calculator2

import (
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/pob2"
)

// ComparePowerCategories is CompareTab's comparePowerCategories.
type ComparePowerCategories struct {
	TreeNodes   bool
	Items       bool
	SkillGems   bool
	SupportGems bool
	Config      bool
}

// ComparePowerEntry is one comparePowerResults row with its impact on each requested stat, before the display
// stat's scale; Var is the config option of a Config row, SlotName the slot of an equipment Item row.
type ComparePowerEntry struct {
	Category string
	Name     string
	NodeID   int
	SlotName string
	Var      string
	Rarity   string
	PathDist int
	Impact   []float64
}

// ComparePowerSide is one build as the power builder reads it: the build, its last main calculation and its items.
type ComparePowerSide struct {
	Build *pob2.PathOfBuilding2
	Env   *Environment
	Items *ItemsTab
}

// ComparePowerBuilder is CompareTab:ComparePowerBuilder run in steps over several power stats at once: each entry's
// impact is CalculatePowerStat of every stat. The primary build's skill and config entries calculate a copy of it.
type ComparePowerBuilder struct {
	primary    ComparePowerSide
	compare    ComparePowerSide
	misc       *MiscCalculator
	tree       *tree2.PassiveTree
	stats      []*PowerStat
	useFullDPS bool
	base       PowerOutput
	copy       *pob2.PathOfBuilding2

	results []ComparePowerEntry
	tasks   []func()
	next    int
}

//pob2:src/Classes/CompareTab.lua:20
const clusterNodeOffset = 65536

//pob2:block:src/Classes/CompareTab.lua:2571-2573
//pob2:block:src/Classes/CompareTab.lua:2675
func NewComparePowerBuilder(primary, compare ComparePowerSide, misc *MiscCalculator, tree *tree2.PassiveTree, stats []string, categories ComparePowerCategories) (*ComparePowerBuilder, error) {
	b := &ComparePowerBuilder{primary: primary, compare: compare, misc: misc, tree: tree, base: powerOutput(misc.Base)}
	for _, name := range stats {
		for i := range PowerStatList {
			if s := &PowerStatList[i]; s.Stat != "" && s.Stat == name {
				b.stats = append(b.stats, s)
				b.useFullDPS = b.useFullDPS || s.Stat == "FullDPS"
				break
			}
		}
	}
	if categories.SkillGems || categories.SupportGems || categories.Config {
		text, err := pob2.SerializeBuild(primary.Build)
		if err != nil {
			return nil, err
		}
		if b.copy, err = pob2.ParseBuild(text); err != nil {
			return nil, err
		}
	}
	if categories.TreeNodes {
		b.treeTasks()
	}
	if categories.Items {
		b.itemTasks()
		b.jewelTasks()
	}
	if categories.SkillGems {
		b.skillGemTasks()
	}
	if categories.SupportGems {
		b.supportGemTasks()
	}
	if categories.Config {
		b.configTasks()
	}
	return b, nil
}

//pob2:block:src/Classes/CalcsTab.lua:711-715
func (b *ComparePowerBuilder) impactOf(o PowerOutput) []float64 {
	out := make([]float64, len(b.stats))
	for i, s := range b.stats {
		out[i] = calculatePowerStat(s, o, b.base)
	}
	return out
}

func (b *ComparePowerBuilder) calcFunc(o calcOverride) PowerOutput {
	return powerOutput(b.misc.calc(o, b.useFullDPS))
}

// recalc is getMiscCalculator over the primary build's copy after an edit: its base pass.
func (b *ComparePowerBuilder) recalc() (PowerOutput, error) {
	env, err := BuildOutput(b.copy, Options{Tree: b.tree, Items: NewItemsTab(b.copy, b.tree, nil)})
	if err != nil {
		return PowerOutput{}, err
	}
	return powerOutput(NewMiscCalculator(env).Base), nil
}

// Step runs tasks until budget has passed and reports the share done.
func (b *ComparePowerBuilder) Step(budget time.Duration) float64 {
	start := time.Now()
	for b.next < len(b.tasks) {
		b.tasks[b.next]()
		b.next++
		if time.Since(start) > budget {
			break
		}
	}
	return b.Progress()
}

func (b *ComparePowerBuilder) StepMs(budgetMs int) float64 {
	return b.Step(time.Duration(budgetMs) * time.Millisecond)
}

func (b *ComparePowerBuilder) Progress() float64 {
	if len(b.tasks) == 0 {
		return 1
	}
	return float64(b.next) / float64(len(b.tasks))
}

func (b *ComparePowerBuilder) Report() []ComparePowerEntry {
	return slices.Clone(b.results)
}

// PoB walks compareEntry.spec.allocNodes with pairs(); ids are walked sorted here.
//
//pob2:block:src/Classes/CompareTab.lua:2710-2763
func (b *ComparePowerBuilder) treeTasks() {
	pSpec := b.primary.Env.Spec
	keys := &PowerBuilder{m: b.misc, modKeys: map[int]string{}}
	cache := map[string]PowerOutput{}
	for _, nodeID := range sortedNodeIDs(b.compare.Env.Spec.AllocNodes) {
		if nodeID >= clusterNodeOffset || pSpec.AllocNodes[nodeID] != nil {
			continue
		}
		pNode := pSpec.Nodes[nodeID]
		if pNode == nil || (pNode.Type != "Normal" && pNode.Type != "Notable" && pNode.Type != "Keystone") || pNode.AscendancyName != "" {
			continue
		}
		b.tasks = append(b.tasks, func() {
			modKey := keys.nodeModKey(pNode)
			if modKey == "" {
				return
			}
			output, ok := cache[modKey]
			if !ok {
				output = b.calcFunc(calcOverride{AddNodes: map[*tree2.SpecNode]bool{pNode: true}})
				cache[modKey] = output
			}
			pathDist := pNode.PathDist
			if pathDist == 0 {
				pathDist = max(len(pNode.Path), 1)
			}
			b.results = append(b.results, ComparePowerEntry{Category: "Tree", Name: pNode.Dn, NodeID: nodeID, PathDist: pathDist, Impact: b.impactOf(output)})
		})
	}
}

//pob2:src/Classes/CompareTab.lua:3661-3667
func shouldShowRing3(primary, compare *Environment) bool {
	return primary.ModDB.Flag(nil, "AdditionalRingSlot") || compare.ModDB.Flag(nil, "AdditionalRingSlot")
}

func selItem(tab *ItemsTab, slotName string) *Item {
	if tab == nil || tab.Slots[slotName] == nil {
		return nil
	}
	return tab.Items[tab.Slots[slotName].SelItemID]
}

func compareItemCopy(raw string) *Item {
	newItem := ParseItem(raw, ItemModSource(WithParser(ParseMod)))
	newItem.NormaliseQuality()
	return newItem
}

//pob2:block:src/Classes/CompareTab.lua:2768-2813
func (b *ComparePowerBuilder) itemTasks() {
	baseSlots := []string{"Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring 1", "Ring 2", "Belt", "Flask 1", "Flask 2", "Charm 1", "Charm 2", "Charm 3"}
	if shouldShowRing3(b.primary.Env, b.compare.Env) {
		baseSlots = slices.Insert(baseSlots, 9, "Ring 3")
	}
	for _, slotName := range baseSlots {
		cItem := selItem(b.compare.Items, slotName)
		pItem := selItem(b.primary.Items, slotName)
		if cItem == nil || cItem.Raw == "" || (pItem != nil && pItem.Name == cItem.Name) {
			continue
		}
		b.tasks = append(b.tasks, func() {
			output := b.calcFunc(calcOverride{RepSlotName: slotName, RepItem: compareItemCopy(cItem.Raw)})
			b.results = append(b.results, ComparePowerEntry{Category: "Item", Name: cItem.Name + ", " + slotName, SlotName: slotName, Rarity: cItem.Rarity, Impact: b.impactOf(output)})
		})
	}
}

// CompareJewelSlot is one GetJewelComparisonSlots entry.
type CompareJewelSlot struct {
	Label          string
	NodeID         int
	PItem, CItem   *Item
	SlotName       string
	PNodeAllocated bool
	CNodeAllocated bool
}

// socketLabel is itemsTab.sockets[nodeId].label after UpdateSockets: "Socket #n" for the nth allocated socket, else
// the control's "Socket".
//
//pob2:block:src/Classes/ItemsTab.lua:342
//pob2:block:src/Classes/ItemsTab.lua:1740-1758
func socketLabel(tab *ItemsTab, spec *tree2.PassiveSpec, nodeID int) (string, bool) {
	if tab == nil || tab.Sockets[nodeID] == nil {
		return "", false
	}
	var active []int
	for id := range tab.Sockets {
		if spec.AllocNodes[id] != nil {
			active = append(active, id)
		}
	}
	sort.Ints(active)
	if i := slices.Index(active, nodeID); i >= 0 {
		return "Socket #" + strconv.Itoa(i+1), true
	}
	return "Socket", true
}

// CompareJewelSlots is GetJewelComparisonSlots: every tree socket either build has a jewel in.
//
//pob2:src/Classes/CompareTab.lua:1427-1491
func CompareJewelSlots(primary, compare ComparePowerSide) []CompareJewelSlot {
	pSpec, cSpec := primary.Env.Spec, compare.Env.Spec
	nodeIDs := map[int]bool{}
	for nodeID, itemID := range pSpec.Jewels {
		if itemID > 0 {
			nodeIDs[nodeID] = true
		}
	}
	for nodeID, itemID := range cSpec.Jewels {
		if itemID > 0 {
			nodeIDs[nodeID] = true
		}
	}
	var result []CompareJewelSlot
	for nodeID := range nodeIDs {
		var pItem, cItem *Item
		if id, ok := pSpec.Jewels[nodeID]; ok && primary.Items != nil {
			pItem = primary.Items.Items[id]
		}
		if id, ok := cSpec.Jewels[nodeID]; ok && compare.Items != nil {
			cItem = compare.Items.Items[id]
		}
		if pItem == nil && cItem == nil {
			continue
		}
		slotName := "Jewel " + strconv.Itoa(nodeID)
		label := slotName
		if l, ok := socketLabel(primary.Items, pSpec, nodeID); ok {
			label = l
		} else if l, ok := socketLabel(compare.Items, cSpec, nodeID); ok {
			label = l
		}
		result = append(result, CompareJewelSlot{
			Label: label, NodeID: nodeID, PItem: pItem, CItem: cItem, SlotName: slotName,
			PNodeAllocated: pSpec.AllocNodes[nodeID] != nil, CNodeAllocated: cSpec.AllocNodes[nodeID] != nil,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].NodeID < result[j].NodeID })
	return result
}

// The best socket is taken per stat.
//
//pob2:block:src/Classes/CompareTab.lua:2818-2898
func (b *ComparePowerBuilder) jewelTasks() {
	pSpec := b.primary.Env.Spec
	type socketInfo struct {
		slotName  string
		node      *tree2.SpecNode
		allocated bool
	}
	var primaryJewelSockets []socketInfo
	for _, slot := range b.primary.Items.OrderedSlots {
		if slot.NodeID != 0 {
			if node := pSpec.Nodes[slot.NodeID]; node != nil {
				primaryJewelSockets = append(primaryJewelSockets, socketInfo{slotName: slot.SlotName, node: node, allocated: pSpec.AllocNodes[slot.NodeID] != nil})
			}
		}
	}
	for _, jEntry := range CompareJewelSlots(b.primary, b.compare) {
		if jEntry.CItem == nil || jEntry.CItem.Raw == "" || (jEntry.PItem != nil && jEntry.PItem.Name == jEntry.CItem.Name) {
			continue
		}
		b.tasks = append(b.tasks, func() {
			newItem := compareItemCopy(jEntry.CItem.Raw)
			var best []float64
			bestSlotLabel := jEntry.Label
			if jEntry.PNodeAllocated {
				best = b.impactOf(b.calcFunc(calcOverride{RepSlotName: jEntry.SlotName, RepItem: newItem}))
			} else {
				for _, s := range primaryJewelSockets {
					o := calcOverride{RepSlotName: s.slotName, RepItem: newItem}
					if !s.allocated {
						o.AddNodes = map[*tree2.SpecNode]bool{s.node: true}
					}
					impact := b.impactOf(b.calcFunc(o))
					if best == nil {
						best = impact
					} else {
						for k, v := range impact {
							best[k] = max(best[k], v)
						}
					}
					bestSlotLabel = jEntry.Label + " (best socket)"
				}
			}
			if best != nil {
				b.results = append(b.results, ComparePowerEntry{Category: "Item", Name: jEntry.CItem.Name + ", " + bestSlotLabel, Rarity: jEntry.CItem.Rarity, Impact: best})
			}
		})
	}
}

//pob2:src/Classes/CompareTab.lua:2523-2528
func gemGrantedEffectName(gem *GemInstance) (string, bool) {
	if gem.GemData != nil && gem.GemData.GrantedEffect != nil {
		return gem.GemData.GrantedEffect.Name, gem.GemData.GrantedEffect.Support
	}
	if gem.GrantedEffect != nil {
		return gem.GrantedEffect.Name, gem.GrantedEffect.Support
	}
	return "", false
}

// gemName is `gem.grantedEffect and gem.grantedEffect.name or gem.nameSpec`.
func gemName(gem *GemInstance) string {
	if gem.GrantedEffect != nil && gem.GrantedEffect.Name != "" {
		return gem.GrantedEffect.Name
	}
	return gem.NameSpec
}

//pob2:src/Classes/CompareTab.lua:2531-2541
func CompareSocketGroupSignature(group *SocketGroup) string {
	var names []string
	for _, gem := range group.GemList {
		if name := gemName(gem); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return strings.Join(names, "+")
}

//pob2:src/Classes/CompareTab.lua:2544-2568
func CompareSocketGroupLabel(group *SocketGroup) string {
	var names []string
	for _, gem := range group.GemList {
		if gem.GrantedEffect != nil && gem.GrantedEffect.Support {
			continue
		}
		if name := gemName(gem); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		for _, gem := range group.GemList {
			if name := gemName(gem); name != "" {
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 {
		return "(empty group)"
	}
	return strings.Join(names, " + ")
}

// The group is added to the copy's saved groups, which the calculation reads as socketGroupList.
//
//pob2:block:src/Classes/CompareTab.lua:2903-2961
func (b *ComparePowerBuilder) skillGemTasks() {
	pSignatures := map[string]bool{}
	for _, group := range b.primary.Env.SocketGroupList {
		pSignatures[CompareSocketGroupSignature(group)] = true
	}
	for _, cGroup := range b.compare.Env.SocketGroupList {
		sig := CompareSocketGroupSignature(cGroup)
		if sig == "" || pSignatures[sig] {
			continue
		}
		saved := saveGeneratedGroup(cGroup)
		b.tasks = append(b.tasks, func() {
			list := savedSkillList(b.copy)
			before := *list
			*list = append(slices.Clone(before), saved)
			output, err := b.recalc()
			*list = before
			if err != nil {
				return
			}
			b.results = append(b.results, ComparePowerEntry{Category: "Skill gem", Name: CompareSocketGroupLabel(cGroup), Impact: b.impactOf(output)})
		})
	}
}

func mainGroupOf(env *Environment) *SocketGroup {
	if i := env.MainSocketGroup; i >= 1 && i <= len(env.SocketGroupList) {
		return env.SocketGroupList[i-1]
	}
	return nil
}

// A main group the calculation generated has no saved entry to add the gem to, and is skipped.
//
//pob2:block:src/Classes/CompareTab.lua:2966-3043
func (b *ComparePowerBuilder) supportGemTasks() {
	cMainGroup, pMainGroup := mainGroupOf(b.compare.Env), mainGroupOf(b.primary.Env)
	if cMainGroup == nil || pMainGroup == nil {
		return
	}
	pSupportNames := map[string]bool{}
	for _, gem := range pMainGroup.GemList {
		if name, support := gemGrantedEffectName(gem); support {
			if name == "" {
				name = gem.NameSpec
			}
			if name != "" {
				pSupportNames[name] = true
			}
		}
	}
	for _, cGem := range cMainGroup.GemList {
		name, support := gemGrantedEffectName(cGem)
		if !support {
			continue
		}
		if name == "" {
			name = cGem.NameSpec
		}
		if name == "" || pSupportNames[name] {
			continue
		}
		tempGem := saveGeneratedGem(cGem, pob2.Gem{})
		tempGem.Corrupted, tempGem.CorruptLevel = pob2.Nil[bool]{}, pob2.Nil[int]{}
		savedIndex := pMainGroup.savedIndex
		b.tasks = append(b.tasks, func() {
			list := savedSkillList(b.copy)
			positions := savedPositions(*list)
			if savedIndex < 1 || savedIndex > len(positions) {
				return
			}
			group := &(*list)[positions[savedIndex-1]]
			before := group.Gems
			group.Gems = append(slices.Clone(before), tempGem)
			output, err := b.recalc()
			group.Gems = before
			if err != nil {
				return
			}
			b.results = append(b.results, ComparePowerEntry{Category: "Support gem", Name: name, Impact: b.impactOf(output)})
		})
	}
}

// seededConfig is configTab.input[var] as CreateConfigSet seeds it: the saved value, else the default state.
func seededConfig(inputs map[string]ConfigValue, v configVar) (ConfigValue, bool) {
	if in, ok := inputs[v.name]; ok {
		return in, true
	}
	if v.defaultState != nil {
		return *v.defaultState, true
	}
	return ConfigValue{}, false
}

//pob2:src/Classes/CompareTab.lua:1170-1177
func normalizeConfigVals(varType string, pVal, cVal ConfigValue, pSet, cSet bool) (ConfigValue, ConfigValue) {
	switch varType {
	case "check":
		if !pSet {
			pVal = ConfigValue{Boolean: false, IsBoolean: true}
		}
		if !cSet {
			cVal = ConfigValue{Boolean: false, IsBoolean: true}
		}
	case "count", "integer", "float":
		if !pSet {
			pVal = ConfigValue{Number: 0, IsNumber: true}
		}
		if !cSet {
			cVal = ConfigValue{Number: 0, IsNumber: true}
		}
	}
	return pVal, cVal
}

//pob2:block:src/Classes/CompareTab.lua:3048-3122
func (b *ComparePowerBuilder) configTasks() {
	pInput, cInput := configInput(b.primary.Build), configInput(b.compare.Build)
	for _, varData := range configVarList {
		if varData.apply == nil || varData.varType == "text" {
			continue
		}
		pVal, pSet := seededConfig(pInput, varData)
		cVal, cSet := seededConfig(cInput, varData)
		pNorm, cNorm := normalizeConfigVals(varData.varType, pVal, cVal, pSet, cSet)
		if pNorm == cNorm {
			continue
		}
		b.tasks = append(b.tasks, func() {
			inputs := activeConfigInputs(b.copy)
			before := *inputs
			*inputs = slices.Clone(before)
			SetConfigInput(b.copy, varData.name, cVal, !cSet)
			output, err := b.recalc()
			*inputs = before
			if err != nil {
				return
			}
			b.results = append(b.results, ComparePowerEntry{Category: "Config", Var: varData.name, Impact: b.impactOf(output)})
		})
	}
}
