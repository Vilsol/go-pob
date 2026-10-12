package calculator

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

//pob:block:src/Modules/Calcs.lua:213,src/Modules/Calcs.lua:251,src/Modules/Calcs.lua:288,src/Modules/Calcs.lua:343-383
type SkillDPS struct {
	Name      string
	DPS       float64
	Count     float64
	Trigger   string
	SkillPart string
	Source    string
}

type SidebarWarnings struct {
	EnergyShieldProtectsMana bool

	CostWarning []string

	LifeCostWarningList []string
	ManaCostWarningList []string
	RageCostWarningList []string
	ESCostWarningList   []string

	LifePercentCostPercentCostWarningList []string
	ManaPercentCostPercentCostWarningList []string

	PlayerAspectCount int
	MinionAspectCount int

	JewelLimitWarning    []string
	SocketLimitWarning   []string
	MissingAnointWarning []string
}

var costWarningPools = []struct{ pool, costResource string }{
	{"LifeUnreserved", "LifeCost"},
	{"ManaUnreserved", "ManaCost"},
	{"Rage", "RageCost"},
	{"EnergyShield", "ESCost"},
}

var percentCostWarningPools = []struct{ pool, costResource string }{
	{"LifeUnreservedPercent", "LifePercentCost"},
	{"ManaUnreservedPercent", "ManaPercentCost"},
}

func costOutputs(output map[string]float64) map[string]float64 {
	costs := make(map[string]float64, 6)
	for _, p := range costWarningPools {
		if v, ok := output[p.costResource]; ok {
			costs[p.costResource] = v
		}
	}
	for _, p := range percentCostWarningPools {
		if v, ok := output[p.costResource]; ok {
			costs[p.costResource] = v
		}
	}
	return costs
}

func (env *Environment) SidebarWarnings() *SidebarWarnings {
	return env.NewSidebarWarningsRun().Result()
}

type SidebarWarningsRun struct {
	env  *Environment
	w    *SidebarWarnings
	next int
	done bool

	loopEnv                *Environment
	cachedPlayerDB         moddb.ModStoreFuncs
	cachedEnemyDB          moddb.ModStoreFuncs
	cachedMinionDB         moddb.ModStoreFuncs
	loopEnvUsed, loopEnvOK bool
}

func (env *Environment) NewSidebarWarningsRun() *SidebarWarningsRun {
	r := &SidebarWarningsRun{env: env, w: &SidebarWarnings{}}
	if env == nil || env.Player == nil || env.Player.Output == nil {
		r.done = true
	}
	return r
}

func (r *SidebarWarningsRun) Step() bool {
	if r.done {
		return false
	}
	list := r.env.Player.ActiveSkillList
	for r.next < len(list) {
		index := r.next
		r.next++
		if r.checkSkill(index) {
			return r.next < len(list)
		}
	}
	return false
}

func (r *SidebarWarningsRun) Result() *SidebarWarnings {
	for r.Step() {
	}
	if r.done {
		return r.w
	}
	r.done = true
	env, w := r.env, r.w
	w.PlayerAspectCount = aspectCount(env.Player)
	if env.Minion != nil {
		w.MinionAspectCount = aspectCount(env.Minion.Actor)
	}
	w.JewelLimitWarning = jewelLimitWarning(env)
	w.SocketLimitWarning, w.MissingAnointWarning = equippedItemWarnings(env)
	return w
}

type skillCostEntry struct {
	costs               map[string]float64
	lifeReservedPercent float64
	manaReservedPercent float64
}

func recordFullDPSSkillCosts(env *Environment, index int, fullEnv *Environment) {
	if env == nil || len(fullEnv.Player.ActiveSkillList) != len(env.Player.ActiveSkillList) {
		return
	}
	if env.fullDPSSkillCosts == nil {
		env.fullDPSSkillCosts = make(map[int]skillCostEntry)
	}
	e := skillCostEntry{costs: costOutputs(fullEnv.Player.Output)}
	if sd := fullEnv.Player.MainSkill.SkillData; sd != nil {
		e.lifeReservedPercent, e.manaReservedPercent = sd.LifeReservedPercent, sd.ManaReservedPercent
	}
	env.fullDPSSkillCosts[index] = e
}

func (r *SidebarWarningsRun) skillCosts(index int, skill *ActiveSkill) (map[string]float64, float64, float64, bool, bool) {
	env := r.env
	if env.triggerCache != nil {
		if cached, ok := env.triggerCache.entries[cacheSkillUUID(env, skill)]; ok {
			return cached.costs, cached.lifeReservedPercent, cached.manaReservedPercent, true, false
		}
	}
	if e, ok := env.fullDPSSkillCosts[index]; ok {
		return e.costs, e.lifeReservedPercent, e.manaReservedPercent, true, false
	}
	if env.Build == nil {
		return nil, 0, 0, false, false
	}
	switch {
	case r.loopEnv == nil:
		r.loopEnv, r.cachedPlayerDB, r.cachedEnemyDB, r.cachedMinionDB = initEnvSpec(env.Build, env.Cache, OutputModeCalculator, false, env.override, nil)
		r.loopEnv.triggerCache = env.triggerCache
		r.loopEnvOK = len(r.loopEnv.Player.ActiveSkillList) == len(env.Player.ActiveSkillList)
	case r.loopEnvUsed:
		r.loopEnv = reinitFullDPSEnv(r.loopEnv, env.Cache, OutputModeCalculator, r.cachedPlayerDB, r.cachedEnemyDB, r.cachedMinionDB)
	}
	if !r.loopEnvOK {
		return nil, 0, 0, false, true
	}
	loopSkill := r.loopEnv.Player.ActiveSkillList[index]
	r.loopEnv.Player.MainSkill = loopSkill
	performCalc(r.loopEnv, true)
	r.loopEnvUsed = true
	var lifeReserved, manaReserved float64
	if loopSkill.SkillData != nil {
		lifeReserved, manaReserved = loopSkill.SkillData.LifeReservedPercent, loopSkill.SkillData.ManaReservedPercent
	}
	return costOutputs(r.loopEnv.Player.Output), lifeReserved, manaReserved, true, true
}

//pob:block:src/Modules/Calcs.lua:432-468
func (r *SidebarWarningsRun) checkSkill(index int) bool {
	env, w := r.env, r.w
	output := env.Player.Output
	mainSkill := env.Player.MainSkill
	skill := env.Player.ActiveSkillList[index]
	if skill.ActiveEffect == nil || skill.ActiveEffect.GrantedEffect == nil {
		return false
	}
	var costs map[string]float64
	var lifeReserved, manaReserved float64
	calculated := false
	if skill == mainSkill {
		costs = costOutputs(output)
		if skill.SkillData != nil {
			lifeReserved, manaReserved = skill.SkillData.LifeReservedPercent, skill.SkillData.ManaReservedPercent
		}
	} else {
		var ok bool
		if costs, lifeReserved, manaReserved, ok, calculated = r.skillCosts(index, skill); !ok {
			return calculated
		}
	}
	w.EnergyShieldProtectsMana = env.ModDB.Flag(nil, "EnergyShieldProtectsMana")
	name := skill.ActiveEffect.GrantedEffect.Name()
	for _, p := range costWarningPools {
		cachedCost, ok := costs[p.costResource]
		if !ok {
			continue
		}
		totalPool := output[p.pool]
		if w.EnergyShieldProtectsMana && p.costResource == "ManaCost" {
			totalPool += output["EnergyShield"]
		}
		if totalPool >= cachedCost {
			continue
		}
		rawPool := strings.TrimSuffix(p.pool, "Unreserved")
		reservation := 0.0
		switch rawPool {
		case "Life":
			reservation = lifeReserved
		case "Mana":
			reservation = manaReserved
		}
		if reservation != 0 && totalPool+math.Ceil(output[rawPool]*reservation/100) >= cachedCost {
			continue
		}
		if mainSkill != nil && mainSkill.ActiveEffect != nil && mainSkill.ActiveEffect.GrantedEffect != nil &&
			mainSkill.ActiveEffect.GrantedEffect.Name() == name {
			if output[p.costResource+"Warning"] == 0 {
				w.CostWarning = append(w.CostWarning, p.costResource)
			}
			output[p.costResource+"Warning"] = 1
		}
		switch p.costResource {
		case "LifeCost":
			w.LifeCostWarningList = append(w.LifeCostWarningList, name)
		case "ManaCost":
			w.ManaCostWarningList = append(w.ManaCostWarningList, name)
		case "RageCost":
			w.RageCostWarningList = append(w.RageCostWarningList, name)
		case "ESCost":
			w.ESCostWarningList = append(w.ESCostWarningList, name)
		}
	}
	for _, p := range percentCostWarningPools {
		cachedCost, ok := costs[p.costResource]
		if !ok || output[p.pool] >= cachedCost {
			continue
		}
		if p.costResource == "LifePercentCost" {
			w.LifePercentCostPercentCostWarningList = append(w.LifePercentCostPercentCostWarningList, name)
		} else {
			w.ManaPercentCostPercentCostWarningList = append(w.ManaPercentCostPercentCostWarningList, name)
		}
	}
	return calculated
}

//pob:block:src/Modules/CalcSetup.lua:760-771
func jewelLimitWarning(env *Environment) []string {
	if env.Build == nil || env.Build.GetBooleanOption("ignoreJewelLimits") {
		return nil
	}
	specs := env.Build.Tree.Specs
	if len(specs) == 0 {
		return nil
	}
	idx := env.Build.Tree.ActiveSpec - 1
	if idx < 0 || idx >= len(specs) {
		idx = 0
	}
	sockets := specs[idx].Sockets

	byID := make(map[int]string, len(env.Build.Items.Items))
	for _, it := range env.Build.Items.Items {
		byID[it.ID] = it.Text
	}

	order := make([]int, 0, len(sockets))
	for i := range sockets {
		order = append(order, i)
	}
	sort.SliceStable(order, func(a, b int) bool { return sockets[order[a]].NodeID < sockets[order[b]].NodeID })

	counts := make(map[string]int)
	var warnings []string
	for _, i := range order {
		socket := sockets[i]
		if _, allocated := env.AllocatedNodes[strconv.Itoa(socket.NodeID)]; !allocated {
			continue
		}
		text, ok := byID[socket.ItemID]
		if !ok {
			continue
		}
		item := parseItemCached(env.Cache, text)
		if item.Limit == nil || *item.Limit <= 0 {
			continue
		}
		key := item.Title
		if item.BaseName == "Timeless Jewel" {
			key = "Historic"
		}
		if counts[key] >= *item.Limit {
			warnings = append(warnings, key)
			continue
		}
		counts[key]++
	}
	return warnings
}

//pob:block:src/Modules/CalcSetup.lua:907-918,src/Modules/CalcSetup.lua:1206-1257
func equippedItemWarnings(env *Environment) ([]string, []string) {
	if env.Build == nil || len(env.Build.Items.ItemSets) == 0 {
		return nil, nil
	}
	var socketLimit, missingAnoint []string
	items := env.Build.Items
	set := items.ItemSets[activeItemSetIndex(items)]
	activeWeaponSet := 1
	if set.UseSecondWeaponSet != nil && *set.UseSecondWeaponSet {
		activeWeaponSet = 2
	}
	byID := make(map[int]string, len(items.Items))
	for _, it := range items.Items {
		byID[it.ID] = it.Text
	}
	slotsByName := make(map[string]pob.Slot, len(set.Slots))
	for _, slot := range set.Slots {
		slotsByName[slot.Name] = slot
	}

	equipped := make(map[string]*Item, len(set.Slots))
	for _, s := range orderedItemSlots {
		if slotName, item, ok := resolveEquippedSlot(env, s, slotsByName, byID, activeWeaponSet, false); ok {
			equipped[slotName] = item
		}
	}
	disabled := disabledItemSlots(env, slotsByName, byID, activeWeaponSet)
	for slotName := range disabled {
		delete(equipped, slotName)
	}

	for _, s := range orderedItemSlots {
		if count := missingAnointCount(equipped[s.name]); count > 0 {
			label := s.name
			if count > 1 {
				label += " (" + strconv.Itoa(count) + " missing)"
			}
			missingAnoint = append(missingAnoint, label)
		}
	}

	for _, s := range orderedItemSlots {
		slotName, item, ok := resolveEquippedSlot(env, s, slotsByName, byID, activeWeaponSet, false)
		if !ok || disabled[slotName] {
			continue
		}
		switch item.Type {
		case "Flask", "Tincture", "Jewel", "Graft":
			continue
		}
		gemSockets := 0
		for _, socket := range item.Sockets {
			switch socket.Color {
			case "R", "G", "B", "W":
				gemSockets++
			}
		}
		if socketedGemCount(env, slotName) > gemSockets {
			socketLimit = append(socketLimit, slotName)
		}
	}
	return socketLimit, missingAnoint
}

//pob:block:src/Modules/CalcSetup.lua:1209-1219
func socketedGemCount(env *Environment, slotName string) int {
	setIndex := activeSkillSetIndex(env.Build.Skills)
	if setIndex < 0 {
		return 0
	}
	count := 0
	for _, group := range env.Build.Skills.SkillSets[setIndex].Skills {
		if socketGroupIsSourced(&group) || !group.Enabled || group.Slot == "" || group.Slot != slotName {
			continue
		}
		for _, gem := range group.Gems {
			if gemData, _ := resolveGemInstance(gem); gemData != nil && gem.Enabled {
				count++
			}
		}
	}
	return count
}

//pob:block:src/Export/Scripts/bases.lua:57-58
func isTalismanBase(base *ItemBase) bool {
	if base.Type != "Amulet" {
		return false
	}
	for _, b := range poe.BaseItemTypes {
		if b.Name == base.Name && strings.HasPrefix(b.ID, "Metadata/Items/Amulets/Talismans/Talisman") {
			return true
		}
	}
	return false
}

var anointLineRe = regexp.MustCompile(`Allocates [a-zA-Z ]+`)

//pob:block:src/Classes/ItemsTab.lua:63-67,src/Classes/ItemsTab.lua:2875-2902
func missingAnointCount(item *Item) int {
	if item == nil || item.Base == nil || isTalismanBase(item.Base) || !(item.CanBeAnointed || item.Base.Type == "Amulet") {
		return 0
	}
	maxAnoints := 1
	switch {
	case item.CanHaveFourEnchants:
		maxAnoints = 4
	case item.CanHaveThreeEnchants:
		maxAnoints = 3
	case item.CanHaveTwoEnchants:
		maxAnoints = 2
	}
	anointCount := 0
	for _, lines := range [][]ModLine{item.EnchantModLines, item.ScourgeModLines, item.ImplicitModLines, item.ExplicitModLines} {
		for _, line := range lines {
			if anointLineRe.MatchString(line.Line) {
				anointCount++
			}
		}
	}
	return max(0, maxAnoints-min(anointCount, maxAnoints))
}

//pob:block:src/Modules/Build.lua:1805-1810
func aspectCount(actor *Actor) int {
	if actor == nil || actor.ModDB == nil {
		return 0
	}
	count := 0
	if actor.Output["CrabBarriersMax"] > 0 && actor.Output["CrabBarriers"] > 0 {
		count++
	}
	if count < 2 && actor.ModDB.Flag(nil, "Condition:AspectOfTheSpiderActive") {
		count++
	}
	if count < 2 && (actor.ModDB.Flag(nil, "Condition:CatsAgilityActive") || actor.ModDB.Flag(nil, "Condition:CatsStealthActive")) {
		count++
	}
	if count < 2 && (actor.ModDB.Flag(nil, "Condition:AviansFlightActive") || actor.ModDB.Flag(nil, "Condition:AviansMightActive")) {
		count++
	}
	return count
}

//pob:block:src/Modules/BuildDisplayStats.lua:96
func requirementSourceName(reqSource *RequirementsTable) string {
	if reqSource == nil {
		return ""
	}
	switch reqSource.Source {
	case "Item":
		if item, ok := reqSource.SourceItem.(*Item); ok && item != nil {
			return item.Name
		}
	case "Gem":
		if reqSource.SourceGem != nil {
			return reqSource.SourceGem.NameSpec
		}
	}
	return ""
}
