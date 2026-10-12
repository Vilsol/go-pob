package calculator

import (
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

type ComparePowerCategories struct {
	TreeNodes   bool
	Items       bool
	SkillGems   bool
	SupportGems bool
	Config      bool
}

type ComparePowerEntry struct {
	Category string
	Name     string
	NodeID   int64
	SlotName string
	Var      string
	Rarity   string
	PathDist int
	Impact   []float64
}

type ComparePowerBuilder struct {
	primary     *pob.PathOfBuilding
	compare     *pob.PathOfBuilding
	baseEnv     *Environment
	usedFullDPS bool
	stats       []PowerStat
	base        []float64

	results []ComparePowerEntry
	tasks   []func()
	next    int
}

//pob:src/Classes/CompareTab.lua:20
const clusterNodeOffset = 65536

//pob:block:src/Classes/CompareTab.lua:2689
var compareBaseSlots = []string{"Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring 1", "Ring 2", "Belt",
	"Flask 1", "Flask 2", "Flask 3", "Flask 4", "Flask 5"}

//pob:block:src/Classes/CompareTab.lua:2487-2508,src/Classes/CompareTab.lua:2594-2595
func NewComparePowerBuilder(primary, compare *pob.PathOfBuilding, stats []string, categories ComparePowerCategories, config []pob.Input) *ComparePowerBuilder {
	b := &ComparePowerBuilder{primary: primary, compare: compare}
	b.baseEnv, b.usedFullDPS = compareMiscBase(primary)
	known := PowerStats()
	for _, name := range stats {
		for _, s := range known {
			if s.Stat == name {
				b.stats = append(b.stats, s)
				break
			}
		}
	}
	b.base = b.valuesOf(b.baseEnv)
	if categories.TreeNodes {
		b.treeTasks()
	}
	if categories.Items {
		b.tasks = append(b.tasks, b.itemTasks)
	}
	if categories.SkillGems {
		b.skillGemTasks()
	}
	if categories.SupportGems {
		b.supportGemTasks()
	}
	if categories.Config {
		for _, in := range config {
			b.tasks = append(b.tasks, func() { b.configTask(in) })
		}
	}
	return b
}

//pob:block:src/Modules/Calcs.lua:123-133
func compareMiscBase(build *pob.PathOfBuilding) (*Environment, bool) {
	env, player, enemy, minion := InitEnvOverride(build, envCache, OutputModeCalculator, nil)
	PerformCalc(env)
	fullDPS, fullDotDPS, skills := calcFullDPS(&Calculator{PoB: build}, env, OutputModeCalculator, envCache, nil,
		&calcSpecEnv{cachedPlayerDB: player, cachedEnemyDB: enemy, cachedMinionDB: minion})
	if len(skills) == 0 {
		return env, false
	}
	env.Player.Output["FullDPS"] = fullDPS
	env.Player.Output["FullDotDPS"] = fullDotDPS
	return env, true
}

func compareBaseEnv(build *pob.PathOfBuilding) *Environment {
	env, _ := compareMiscBase(build)
	return env
}

//pob:block:src/Modules/Calcs.lua:134-147
func (b *ComparePowerBuilder) calc(override *CalcOverride) *Environment {
	env, player, enemy, minion := InitEnvOverride(b.primary, envCache, OutputModeCalculator, override)
	PerformCalc(env)
	if b.usedFullDPS {
		fullDPS, fullDotDPS, _ := calcFullDPS(&Calculator{PoB: b.primary}, env, OutputModeCalculator, envCache, env.override,
			&calcSpecEnv{cachedPlayerDB: player, cachedEnemyDB: enemy, cachedMinionDB: minion})
		env.Player.Output["FullDPS"] = fullDPS
		env.Player.Output["FullDotDPS"] = fullDotDPS
	}
	return env
}

func (b *ComparePowerBuilder) Stats() []PowerStat {
	return b.stats
}

func (b *ComparePowerBuilder) valuesOf(env *Environment) []float64 {
	out := make([]float64, len(b.stats))
	for i, s := range b.stats {
		out[i] = powerStatValue(env, s)
	}
	return out
}

//pob:src/Classes/CalcsTab.lua:735-739
func (b *ComparePowerBuilder) impactOf(env *Environment) []float64 {
	out := b.valuesOf(env)
	for i := range out {
		out[i] -= b.base[i]
	}
	return out
}

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

//pob:block:src/Classes/CompareTab.lua:2630-2683
func (b *ComparePowerBuilder) treeTasks() {
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	primaryNodes := map[int64]bool{}
	for _, id := range b.primary.Build.PassiveNodes {
		primaryNodes[id] = true
	}
	var roots []int64
	if classID, ok := pob.ClassIDByName(b.primary.Build.ClassName); ok {
		start := tree.Nodes[classStartNodeID(tree, classID)]
		for _, id := range linkedNodeIDs(start) {
			if n, ok := tree.Nodes[id]; ok && n.Skill != nil && n.AscendancyName == nil {
				roots = append(roots, *n.Skill)
			}
		}
	}
	paths := data.TreeVersions[data.LatestTreeVersion].CalculateAllocationPaths(b.primary.Build.PassiveNodes, roots)
	ids := slices.Clone(b.compare.Build.PassiveNodes)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	cache := map[string][]float64{}
	for _, id := range ids {
		if id >= clusterNodeOffset || primaryNodes[id] {
			continue
		}
		n, ok := tree.Nodes[strconv.FormatInt(id, 10)]
		if !ok || !specNode(tree, n) {
			continue
		}
		switch powerNodeType(n) {
		case "Normal", "Notable", "Keystone":
		default:
			continue
		}
		key := statsModKey(n.Stats)
		if (n.AscendancyName != nil && *n.AscendancyName != "") || key == "" {
			continue
		}
		b.tasks = append(b.tasks, func() {
			impact, ok := cache[key]
			if !ok {
				impact = b.impactOf(b.calc(&CalcOverride{AddNodes: []int64{id}}))
				cache[key] = impact
			}
			pathDist := 1000
			if p := allocationPath(paths, id); p != nil {
				pathDist = len(p)
			}
			name := ""
			if n.Name != nil {
				name = *n.Name
			}
			b.results = append(b.results, ComparePowerEntry{Category: "Tree", Name: name, NodeID: id, PathDist: pathDist, Impact: impact})
		})
	}
}

//pob:src/Classes/CompareTab.lua:3639-3645
func shouldShowRing3(primary, compare *Environment) bool {
	has := func(env *Environment) bool {
		return env != nil && env.ModDB != nil && env.ModDB.Flag(nil, "AdditionalRingSlot")
	}
	return has(primary) || has(compare)
}

//pob:src/Classes/CompareTab.lua:3650-3664
func compareAbyssSockets(primary, compare *pob.PathOfBuilding) []string {
	equipment := []string{"Weapon 1", "Weapon 2", "Weapon 1 Swap", "Weapon 2 Swap", "Helmet", "Body Armour", "Gloves", "Boots", "Belt"}
	var out []string
	for _, parent := range equipment {
		for _, s := range orderedItemSlots {
			if s.parentSlot == parent && slotShown(primary, s, false) && slotShown(compare, s, false) {
				out = append(out, s.name)
			}
		}
	}
	return out
}

//pob:block:src/Classes/CompareTab.lua:2688-2783
func (b *ComparePowerBuilder) itemTasks() {
	slots := slices.Clone(compareBaseSlots)
	if shouldShowRing3(b.baseEnv, compareBaseEnv(b.compare)) {
		slots = slices.Insert(slots, 9, "Ring 3")
	}
	slots = append(slots, compareAbyssSockets(b.primary, b.compare)...)
	cSet, cByID := setItems(b.compare)
	pSet, pByID := setItems(b.primary)
	for _, slotName := range slots {
		cText, ok := cByID[setSlotItemID(cSet, slotName)]
		if !ok || strings.TrimSpace(cText) == "" {
			continue
		}
		cItem := parseItemGlobal(cText)
		if pItem := setSlotItem(pSet, pByID, slotName); pItem != nil && pItem.Name == cItem.Name {
			continue
		}
		b.tasks = append(b.tasks, func() {
			newItem := ParseItem(cText)
			newItem.NormaliseQuality()
			jewels, restore := b.equipCompareAbyssJewels(slotName, newItem)
			impact := b.impactOf(b.calc(&CalcOverride{RepSlotName: slotName, RepItem: newItem}))
			restore()
			b.results = append(b.results, ComparePowerEntry{Category: "Item", Name: cItem.Name + ", " + slotName + abyssJewelText(jewels), SlotName: slotName,
				Rarity: string(cItem.Rarity), Impact: impact})
		})
	}
}

//pob:src/Classes/CompareTab.lua:2439-2444
func compareGemEffect(gem pob.Gem) *poe.GrantedEffect {
	gemData, effect := resolveGemInstance(gem)
	if gemData != nil {
		if e := primaryGrantedEffectOf(gemData); e != nil {
			return e
		}
	}
	return effect
}

//pob:block:src/Classes/SkillsTab.lua:1141-1174
func compareGemName(gem pob.Gem) (string, *poe.GrantedEffect) {
	gemData, effect := resolveGemInstance(gem)
	if gemData != nil {
		if p, ok := pobGemFor(gemData); ok && p.Name != "" {
			return p.Name, nil
		}
		return gem.NameSpec, nil
	}
	if effect != nil {
		return (&GrantedEffect{Raw: effect}).Name(), effect
	}
	return gem.NameSpec, nil
}

//pob:src/Classes/CompareTab.lua:2447-2457
func CompareSocketGroupSignature(group pob.Skill) string {
	names := make([]string, 0, len(group.Gems))
	for _, gem := range group.Gems {
		if name, _ := compareGemName(gem); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return strings.Join(names, "+")
}

//pob:src/Classes/CompareTab.lua:2460-2484
func CompareSocketGroupLabel(group pob.Skill) string {
	var names []string
	for _, gem := range group.Gems {
		if name, e := compareGemName(gem); name != "" && (e == nil || !e.IsSupport) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		for _, gem := range group.Gems {
			if name, _ := compareGemName(gem); name != "" {
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 {
		return "(empty group)"
	}
	return strings.Join(names, " + ")
}

func activeSocketGroups(build *pob.PathOfBuilding) *[]pob.Skill {
	if len(build.Skills.SkillSets) == 0 {
		return nil
	}
	return &build.Skills.SkillSets[build.ActiveSkillSetIndex()].Skills
}

func cloneSocketGroup(group pob.Skill) pob.Skill {
	group.Gems = slices.Clone(group.Gems)
	return group
}

//pob:block:src/Classes/CompareTab.lua:2873-2931
func (b *ComparePowerBuilder) skillGemTasks() {
	cGroups := activeSocketGroups(b.compare)
	if cGroups == nil {
		return
	}
	pSignatures := map[string]bool{}
	if pGroups := activeSocketGroups(b.primary); pGroups != nil {
		for _, g := range *pGroups {
			pSignatures[CompareSocketGroupSignature(g)] = true
		}
	}
	for _, cGroup := range *cGroups {
		sig := CompareSocketGroupSignature(cGroup)
		if sig == "" || pSignatures[sig] {
			continue
		}
		group := cloneSocketGroup(cGroup)
		b.tasks = append(b.tasks, func() {
			pGroups := activeSocketGroups(b.primary)
			if pGroups == nil {
				return
			}
			saved := *pGroups
			*pGroups = append(slices.Clone(saved), cloneSocketGroup(group))
			env := compareBaseEnv(b.primary)
			*pGroups = saved
			b.results = append(b.results, ComparePowerEntry{Category: "Skill gem", Name: CompareSocketGroupLabel(group), Impact: b.impactOf(env)})
		})
	}
}

func mainSocketGroup(build *pob.PathOfBuilding) (*pob.Skill, int) {
	groups := activeSocketGroups(build)
	i := build.Build.MainSocketGroup - 1
	if groups == nil || i < 0 || i >= len(*groups) {
		return nil, -1
	}
	return &(*groups)[i], i
}

//pob:block:src/Classes/CompareTab.lua:2936-3014
func (b *ComparePowerBuilder) supportGemTasks() {
	cMain, _ := mainSocketGroup(b.compare)
	pMain, _ := mainSocketGroup(b.primary)
	if cMain == nil || pMain == nil {
		return
	}
	pSupportNames := map[string]bool{}
	supportName := func(gem pob.Gem) string {
		e := compareGemEffect(gem)
		if e == nil || !e.IsSupport {
			return ""
		}
		if gemData, _ := resolveGemInstance(gem); gemData != nil {
			if p, ok := pobGemFor(gemData); ok && p.Name != "" {
				return p.Name
			}
		}
		if gem.NameSpec != "" {
			return gem.NameSpec
		}
		return (&GrantedEffect{Raw: e}).Name()
	}
	for _, gem := range pMain.Gems {
		if name := supportName(gem); name != "" {
			pSupportNames[name] = true
		}
	}
	for _, cGem := range cMain.Gems {
		name := supportName(cGem)
		if name == "" || pSupportNames[name] {
			continue
		}
		gem := cGem
		b.tasks = append(b.tasks, func() {
			group, i := mainSocketGroup(b.primary)
			if group == nil {
				return
			}
			groups := activeSocketGroups(b.primary)
			saved := (*groups)[i].Gems
			(*groups)[i].Gems = append(slices.Clone(saved), gem)
			env := compareBaseEnv(b.primary)
			(*groups)[i].Gems = saved
			b.results = append(b.results, ComparePowerEntry{Category: "Support gem", Name: name, Impact: b.impactOf(env)})
		})
	}
}

//pob:block:src/Classes/CompareTab.lua:3019-3093
func (b *ComparePowerBuilder) configTask(in pob.Input) {
	saved := b.primary.Config.Inputs
	b.primary.Config.Inputs = cloneInputs(saved)
	if in.Boolean == nil && in.Number == nil && in.String == nil {
		b.primary.RemoveConfigOption(in.Name)
	} else {
		b.primary.SetConfigOption(in)
	}
	env := compareBaseEnv(b.primary)
	b.primary.Config.Inputs = saved
	b.results = append(b.results, ComparePowerEntry{Category: "Config", Var: in.Name, Impact: b.impactOf(env)})
}
