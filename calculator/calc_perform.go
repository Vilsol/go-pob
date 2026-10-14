package calculator

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/calculator/calclib"
	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

//pob:block:src/Modules/CalcPerform.lua:2480
var totemAuraResistRe = regexp.MustCompile(`Resist?M?a?x?$`)

//pob:block:src/Modules/Data.lua:322-349
var cursePriority = map[string]float64{
	"Temporal Chains":    1,
	"Enfeeble":           2,
	"Vulnerability":      3,
	"Elemental Weakness": 4,
	"Flammability":       5,
	"Frostbite":          6,
	"Conductivity":       7,
	"Despair":            8,
	"Punishment":         9,
	"Warlord's Mark":     10,
	"Assassin's Mark":    11,
	"Sniper's Mark":      12,
	"Poacher's Mark":     13,
	"SocketPriorityBase": 100,
	"Weapon 1":           1000,
	"Amulet":             2000,
	"Helmet":             3000,
	"Weapon 2":           4000,
	"Body Armour":        5000,
	"Gloves":             6000,
	"Boots":              7000,
	"Ring 1":             8000,
	"Ring 2":             9000,
	"Ring 3":             10000,
	"CurseFromEquipment": 11000,
	"CurseFromAura":      20000,
}

type curseEntry struct {
	name                   string
	modList                *moddb.ModList
	buffModList            *moddb.ModList
	minionBuffModList      *moddb.ModList
	priority               float64
	fromPlayer             bool
	isMark                 bool
	ignoreHexLimit         bool
	socketedCursesHexLimit bool
}

//pob:block:src/Modules/CalcPerform.lua:3091-3097
func extraCurseEffect(value mod.ExtraCurse) *GemEffect {
	id := value.SkillID
	if id == "" && value.SkillName != "" {
		id = extraSkillIDForName(value.SkillName)
	}
	if id == "" {
		return nil
	}
	if _, inPoBData := SkillBaseFlagsFor(id); !inPoBData {
		return nil
	}
	grantedEffect := poe.GrantedEffectByID(id)
	if grantedEffect == nil || grantedEffect.GetActiveSkill() == nil {
		return nil
	}

	skillTypes := grantedEffectSkillTypes(grantedEffect)
	level := value.Level
	if level < 1 {
		level = 1
	}
	return &GemEffect{
		GrantedEffect: &GrantedEffect{
			Raw:        grantedEffect,
			Parts:      skillPartsFor(grantedEffect.ID),
			SkillTypes: skillTypes,
			Funcs:      skillFuncsFor(grantedEffect.ID),
		},
		Level:   level,
		Quality: 0,
	}
}

//pob:src/Modules/CalcPerform.lua:563-593
func determineCursePriority(curseName string, activeSkill *ActiveSkill) float64 {
	source := ""
	slot := ""
	socket := 1.0

	if activeSkill != nil {
		if socketGroup, ok := activeSkill.SocketGroup.(*pob.Skill); ok && socketGroup != nil {
			source = socketGroup.Source
			slot = socketGroup.Slot
			for k, gemInstance := range socketGroup.Gems {
				if gemInstanceName(gemInstance) == curseName {
					socket = math.Min(float64(k+1), 8)
					break
				}
			}
		}
	}

	basePriority := cursePriority[curseName]
	socketPriority := float64(socket * cursePriority["SocketPriorityBase"])
	slotPriority := cursePriority[strings.ReplaceAll(slot, " Swap", "")]
	sourcePriority := 0.0
	if activeSkill != nil && activeSkill.SkillTypes[data.SkillTypeAura] {
		sourcePriority = cursePriority["CurseFromAura"]
	} else if source != "" {
		sourcePriority = cursePriority["CurseFromEquipment"]
	}
	if source != "" && (slotPriority == cursePriority["Ring 2"] || slotPriority == cursePriority["Ring 3"]) {
		slotPriority = cursePriority["Ring 1"]
	}

	return basePriority + socketPriority + slotPriority + sourcePriority
}

//pob:block:src/Modules/CalcPerform.lua:571-572
func gemInstanceName(gemInstance pob.Gem) string {
	gem, _ := resolveGemInstance(gemInstance)
	if gem == nil {
		return ""
	}
	pobGem, ok := pobGemForVariant(gem, gemInstance.SkillID)
	if !ok {
		return ""
	}
	return pobGem.Name
}

var playerActorCfg = &moddb.ListCfg{Actor: "player"}

//pob:src/Modules/CalcPerform.lua:44-66
func mergeBuff(src *moddb.ModList, dest map[string]*moddb.ModList, key string) {
	if src == nil {
		return
	}
	if dest[key] == nil {
		dest[key] = moddb.NewModList()
	}
	destList := dest[key]
	for _, srcMod := range src.Mods() {
		merged := false
		//pob:src/Modules/CalcPerform.lua:51
		if srcMod.Type() == mod.TypeList {
			destList.AddMod(srcMod.Clone())
			continue
		}
		//pob:src/Modules/CalcPerform.lua:52-60
		for index, destMod := range destList.Mods() {
			if moddb.CompareModParams(srcMod, destMod) {
				sv := srcMod.Value()
				dv := destMod.Value()
				if sv != nil && dv != nil &&
					sv.Type() == mod.ModValueMultiTypeFloat && dv.Type() == mod.ModValueMultiTypeFloat {
					if dv.Float() < sv.Float() {
						destList.Mods()[index] = srcMod.Clone()
					}
				}
				merged = true
				break
			}
		}
		if !merged {
			destList.AddMod(srcMod.Clone())
		}
	}
}

func asModList(mods []mod.Mod) *moddb.ModList {
	list := moddb.NewModList()
	list.SetMods(mods)
	return list
}

func mergeFlaskBuff(src *moddb.ModList, dest map[string]*moddb.ModList, order []string, key string) []string {
	if dest[key] == nil {
		order = append(order, key)
	}
	mergeBuff(src, dest, key)
	return order
}

//pob:block:src/Modules/CalcPerform.lua:1555-1563
func flaskModKey(mods []mod.Mod) string {
	var b []byte
	for _, m := range mods {
		b = append(b, m.Name()...)
		b = append(b, '|')
		b = append(b, m.Type()...)
		b = append(b, '|')
		b = strconv.AppendInt(b, int64(m.Flags()), 10)
		b = append(b, '|')
		b = strconv.AppendInt(b, int64(m.KeywordFlags()), 10)
		for _, tag := range m.Tags() {
			b = append(b, '|')
			b = append(b, mod.FormatValue(tag)...)
		}
		b = append(b, '&')
	}
	return string(b)
}

type flaskSlotItem struct {
	item   *Item
	active bool
}

//pob:block:src/Modules/CalcPerform.lua:1489-1501
func orderedFlaskSlots(env *Environment) []flaskSlotItem {
	items := env.Build.Items
	if len(items.ItemSets) == 0 {
		return nil
	}

	setIdx := activeItemSetIndex(items)

	byID := itemTextByID(env)

	slots := make([]pob.Slot, 0, 5)
	for _, slot := range items.ItemSets[setIdx].Slots {
		if slot.ItemID != 0 && flaskSlotNum(slot.Name) > 0 {
			slots = append(slots, slot)
		}
	}
	sort.SliceStable(slots, func(i, j int) bool {
		return flaskSlotNum(slots[i].Name) < flaskSlotNum(slots[j].Name)
	})

	out := make([]flaskSlotItem, 0, len(slots))
	for _, slot := range slots {
		text, ok := byID[slot.ItemID]
		if !ok {
			continue
		}
		item := parseItemCached(env.Cache, text)
		if item.Type == "Flask" {
			out = append(out, flaskSlotItem{item: item, active: slot.Active})
		}
	}
	return out
}

func flaskSlotNum(name string) int {
	rest, ok := strings.CutPrefix(name, "Flask ")
	if !ok {
		return 0
	}
	num, err := strconv.Atoi(rest)
	if err != nil || num < 1 {
		return 0
	}
	return num
}

//pob:src/Modules/CalcPerform.lua:1603-1615
func getFlaskInstantRecovery(env *Environment, item *Item) float64 {
	instantPerc := item.FlaskData["instantPerc"]
	if env.ModDB.Flag(nil, "Condition:LowLife") {
		instantPerc += item.FlaskData["instantLowLifePerc"]
	}
	if item.Base.Flask.Life > 0 {
		instantPerc += env.ModDB.Sum(mod.TypeBase, nil, "LifeFlaskInstantRecovery")
	}
	if item.Base.Flask.Mana > 0 {
		instantPerc += env.ModDB.Sum(mod.TypeBase, nil, "ManaFlaskInstantRecovery")
	}
	return math.Min(100, instantPerc)
}

//pob:src/Modules/CalcPerform.lua:1617-1654
func calcFlaskRecovery(env *Environment, typ string, item *Item, effectInc, flaskDurInc, flaskTotalRateInc float64) []mod.Mod {
	lType := strings.ToLower(typ)

	if item.FlaskData[lType+"EffectNotRemoved"] == 0 && !env.ModDB.Flag(nil, typ+"FlaskEffectNotRemoved") {
		return nil
	}

	name := item.Name
	base := item.FlaskData[lType+"Base"]
	dur := item.FlaskData["duration"]
	instPerc := getFlaskInstantRecovery(env, item)
	flaskRecInc := env.ModDB.Sum(mod.TypeIncrease, nil, "Flask"+typ+"Recovery")
	flaskRecMore := env.ModDB.More(nil, "Flask"+typ+"Recovery")
	flaskRateInc := env.ModDB.Sum(mod.TypeIncrease, nil, "Flask"+typ+"RecoveryRate")
	flaskTotal := base * (1 - instPerc/100) * (1 + flaskRecInc/100) * flaskRecMore * (1 + flaskDurInc/100)
	flaskDur := dur * (1 + flaskDurInc/100) / (1 + flaskTotalRateInc/100) / (1 + flaskRateInc/100)

	lowLifeFlaskRecMore := env.ModDB.More(nil, "FlaskLifeRecoveryLowLife")
	if lowLifeFlaskRecMore > 1 && lType == "life" {
		flaskTotal *= (lowLifeFlaskRecMore-1)/(1+effectInc/100) + 1
	}

	out := []mod.Mod{mod.NewFloat(typ+"Recovery", mod.TypeBase, flaskTotal/flaskDur).Source(mod.Source(name))}

	if env.ModDB.Flag(nil, typ+"FlaskAppliesToEnergyShield") {
		out = append(out, mod.NewFloat("EnergyShieldRecovery", mod.TypeBase, flaskTotal/flaskDur).Source(mod.Source(name)))
	}
	if env.ModDB.Flag(nil, typ+"FlaskAppliesToLife") {
		out = append(out, mod.NewFloat("LifeRecovery", mod.TypeBase, flaskTotal/flaskDur).Source(mod.Source(name)))
	}

	return out
}

//pob:block:src/Modules/CalcPerform.lua:1666-1695
func mergeFlaskRecovery(env *Environment, item *Item, typ, key string,
	effectInc, flaskDurInc, flaskTotalRateInc float64,
	dest map[string]*moddb.ModList, order []string,
) []string {
	recovery := calcFlaskRecovery(env, typ, item, effectInc, flaskDurInc, flaskTotalRateInc)
	if len(recovery) == 0 {
		return order
	}
	effectMod := 1 + (effectInc+item.FlaskData["effectInc"])/100
	srcList := moddb.NewModList()
	srcList.ScaleAddList(asModList(recovery), effectMod)
	return mergeFlaskBuff(srcList, dest, order, key)
}

//pob:src/Modules/CalcPerform.lua:1570-1574
type flaskMinionState struct {
	effectIncNonPlayer      float64
	effectIncMagicNonPlayer float64
	flasksApply             bool
	quickSilverApply        bool
	nonUniqueApply          bool

	buffs        map[string]*moddb.ModList
	order        []string
	perBase      map[string]map[string]*moddb.ModList
	perBaseOrder map[string][]string
}

func newFlaskMinionState(env *Environment) *flaskMinionState {
	s := &flaskMinionState{
		effectIncNonPlayer:      env.ModDB.Sum(mod.TypeIncrease, nil, "FlaskEffect"),
		effectIncMagicNonPlayer: env.ModDB.Sum(mod.TypeIncrease, nil, "MagicUtilityFlaskEffect"),
	}
	if env.Minion != nil {
		var cfg *moddb.ListCfg
		if env.Player.MainSkill != nil {
			cfg = env.Player.MainSkill.SkillCfg
		}
		s.flasksApply = env.ModDB.Flag(cfg, "FlasksApplyToMinion")
		s.quickSilverApply = env.ModDB.Flag(cfg, "QuickSilverAppliesToAllies")
		s.nonUniqueApply = env.Minion.ModDB.Flag(nil, "ParentNonUniqueFlasksAppliedToYou")
	}
	return s
}

func (s *flaskMinionState) reset(env *Environment) {
	s.buffs = make(map[string]*moddb.ModList)
	s.order = nil
	s.perBase = make(map[string]map[string]*moddb.ModList)
	s.perBaseOrder = make(map[string][]string)
	for _, item := range env.Flasks {
		if _, ok := s.perBase[item.BaseName]; !ok {
			s.perBase[item.BaseName] = make(map[string]*moddb.ModList)
		}
	}
}

func (s *flaskMinionState) eligible(item *Item) bool {
	return s.flasksApply || s.quickSilverApply ||
		(s.nonUniqueApply && item.Rarity != RarityUnique && item.Rarity != RarityRelic)
}

func (s *flaskMinionState) effectModNonPlayer(item *Item) float64 {
	inc := s.effectIncNonPlayer + item.FlaskData["effectInc"]
	isLife := item.Base != nil && item.Base.Flask != nil && item.Base.Flask.Life > 0
	isMana := item.Base != nil && item.Base.Flask != nil && item.Base.Flask.Mana > 0
	if item.Rarity == RarityMagic && !(isLife || isMana) {
		inc += s.effectIncMagicNonPlayer
	}
	return 1 + inc/100
}

func (s *flaskMinionState) merge(item *Item, list *moddb.ModList, key string) {
	srcList := moddb.NewModList()
	srcList.ScaleAddList(list, s.effectModNonPlayer(item))
	s.order = mergeFlaskBuff(srcList, s.buffs, s.order, key)
	s.perBaseOrder[item.BaseName] = mergeFlaskBuff(srcList, s.perBase[item.BaseName], s.perBaseOrder[item.BaseName], key)
}

func (s *flaskMinionState) calcFlaskMods(item *Item, baseName string, buffModList, modList []mod.Mod) {
	if !s.eligible(item) {
		return
	}
	if len(buffModList) > 0 {
		s.merge(item, asModList(buffModList), baseName)
	}
	if len(modList) > 0 {
		key := item.Title
		if item.Rarity != RarityUnique && item.Rarity != RarityRelic {
			key = flaskModKey(modList)
		}
		s.merge(item, asModList(modList), key)
	}
}

func flaskConditionNames(env *Environment) []string {
	var names []string
	seen := map[string]bool{}
	add := func(n string) {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	for _, item := range env.Flasks {
		if getFlaskInstantRecovery(env, item) >= 100 {
			continue
		}
		add("UsingFlask")
		add("Using" + strings.Join(strings.Fields(item.BaseName), ""))
		if item.Base != nil && item.Base.Flask != nil {
			if item.Base.Flask.Life > 0 && !env.ModDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
				add("UsingLifeFlask")
			}
			if item.Base.Flask.Mana > 0 {
				add("UsingManaFlask")
			}
		}
	}
	return names
}

//pob:block:src/Modules/CalcPerform.lua:1790-1808
func (s *flaskMinionState) applyToMinion(env *Environment) {
	if env.Minion == nil {
		return
	}
	conds := flaskConditionNames(env)
	minionModDB := env.Minion.ModDB
	if s.flasksApply || s.nonUniqueApply { // nonUniqueAlreadyFiltered
		for _, c := range conds {
			minionModDB.Conditions[c] = true
		}
		for _, key := range s.order {
			minionModDB.AddList(s.buffs[key])
		}
		return
	}
	// Not all flasks apply to minions. Check if some flasks need to be selectively applied
	if qs, ok := s.perBase["Quicksilver Flask"]; s.quickSilverApply && ok {
		has := func(n string) bool { return slices.Contains(conds, n) }
		minionModDB.Conditions["UsingQuicksilverFlask"] = has("UsingQuicksilverFlask")
		minionModDB.Conditions["UsingFlask"] = has("UsingFlask")
		for _, key := range s.perBaseOrder["Quicksilver Flask"] {
			minionModDB.AddList(qs[key])
		}
	}
}

func warcryDisplayName(activeSkill *ActiveSkill) string {
	if as := activeSkill.ActiveEffect.GrantedEffect.Raw.GetActiveSkill(); as != nil {
		return as.DisplayedName
	}
	return ""
}

var rallyingCryDamageTypes = []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"}

//pob:block:src/Modules/CalcPerform.lua:1422-1423
func weaponDataDamage(weaponData *SkillData, damageType string) (float64, float64) {
	if weaponData == nil {
		return 0, 0
	}

	switch damageType {
	case "Physical":
		return weaponData.PhysicalMin, weaponData.PhysicalMax
	case "Lightning":
		return weaponData.LightningMin, weaponData.LightningMax
	case "Cold":
		return weaponData.ColdMin, weaponData.ColdMax
	case "Fire":
		return weaponData.FireMin, weaponData.FireMax
	case "Chaos":
		return weaponData.ChaosMin, weaponData.ChaosMax
	}

	return 0, 0
}

type cachedTriggerSkill struct {
	Speed       float64
	HitSpeed    float64
	hitSpeedSet bool
	HitTime     float64
	ManaCost    float64
	ManaCostRaw float64
	HitChance   float64
	CritChance  float64

	TotalDPS float64

	ProjectileCount float64

	TotemLife    float64
	totemLifeSet bool

	MainHandHitChance  float64
	OffHandHitChance   float64
	MainHandCritChance float64
	OffHandCritChance  float64

	BattlemageUpTimeRatio    float64
	battlemageUpTimeRatioSet bool

	BattleCryExertsCount  float64
	BattleMageCryDuration float64
	BattleMageCryCastTime float64
	BattleMageCryCooldown float64

	InfernalUpTimeRatio    float64
	infernalUpTimeRatioSet bool

	InfernalExertsCount float64
	InfernalCryDuration float64
	InfernalCryCastTime float64
	InfernalCryCooldown float64

	ChanceToTriggerOnCrit float64

	Duration float64

	speedSet bool

	costs               map[string]float64
	lifeReservedPercent float64
	manaReservedPercent float64
}

type triggerSkillCache struct {
	entries map[string]*cachedTriggerSkill

	spare []*triggerSourceEnv

	building map[string]bool
}

//pob:src/Modules/Common.lua:660-675
func cacheSkillUUID(env *Environment, skill *ActiveSkill) string {
	name := ""
	if skill.ActiveEffect != nil && skill.ActiveEffect.GrantedEffect != nil {
		name = skill.ActiveEffect.GrantedEffect.ID()
	}

	slotName := skill.SlotName
	if slotName == "" {
		slotName = "NO_SLOT"
	}

	index := activeSkillIndexInGroup(env, skill)
	if index < 1 {
		index = 1
	}

	return strings.ReplaceAll(name, " ", "") + "_" + strings.ReplaceAll(slotName, " ", "") + "_" + strconv.Itoa(index)
}

//pob:src/Modules/Calcs.lua:387-399, src/Modules/CalcPerform.lua:3131-3133
func buildActiveSkillCache(env *Environment, skill *ActiveSkill) *cachedTriggerSkill {
	uuid := cacheSkillUUID(env, skill)
	if cached, ok := env.triggerCache.entries[uuid]; ok {
		return cached
	}
	if env.triggerCache.building[uuid] {
		return nil
	}

	if env.Build == nil || skill.SocketGroupIndex < 1 || activeSkillIndexInGroup(env, skill) < 1 {
		return nil
	}

	env.triggerCache.building[uuid] = true
	defer delete(env.triggerCache.building, uuid)

	//pob:block:src/Modules/Calcs.lua:392-412
	source := triggerSourceEnvFor(env)
	defer func() { env.triggerCache.spare = append(env.triggerCache.spare, source) }()
	subEnv := source.env
	subEnv.triggerCache = env.triggerCache
	var target *ActiveSkill
	for _, activeSkill := range subEnv.Player.ActiveSkillList {
		if cacheSkillUUID(subEnv, activeSkill) == uuid {
			target = activeSkill
			break
		}
	}
	if target == nil {
		return nil
	}
	subEnv.Player.MainSkill = target
	performCalc(subEnv, true)

	speed, speedSet := subEnv.Player.Output["Speed"]
	hitSpeed, hitSpeedSet := subEnv.Player.Output["HitSpeed"]
	cached := &cachedTriggerSkill{
		Speed:       speed,
		speedSet:    speedSet,
		HitSpeed:    hitSpeed,
		hitSpeedSet: hitSpeedSet,
		HitTime:     subEnv.Player.Output["HitTime"],
		ManaCost:    subEnv.Player.Output["ManaCost"],
		ManaCostRaw: subEnv.Player.Output["ManaCostRaw"],
		HitChance:   subEnv.Player.Output["HitChance"],
		CritChance:  subEnv.Player.Output["CritChance"],
		TotalDPS:    subEnv.Player.Output["TotalDPS"],

		ProjectileCount: subEnv.Player.Output["ProjectileCount"],
		Duration:        subEnv.Player.Output["Duration"],

		MainHandHitChance:  subEnv.Player.OutputTable[OutTableMainHand]["HitChance"],
		OffHandHitChance:   subEnv.Player.OutputTable[OutTableOffHand]["HitChance"],
		MainHandCritChance: subEnv.Player.OutputTable[OutTableMainHand]["CritChance"],
		OffHandCritChance:  subEnv.Player.OutputTable[OutTableOffHand]["CritChance"],

		BattleCryExertsCount:  subEnv.Player.Output["BattleCryExertsCount"],
		BattleMageCryDuration: subEnv.Player.Output["BattleMageCryDuration"],
		BattleMageCryCastTime: subEnv.Player.Output["BattleMageCryCastTime"],
		BattleMageCryCooldown: subEnv.Player.Output["BattleMageCryCooldown"],

		InfernalExertsCount: subEnv.Player.Output["InfernalExertsCount"],
		InfernalCryDuration: subEnv.Player.Output["InfernalCryDuration"],
		InfernalCryCastTime: subEnv.Player.Output["InfernalCryCastTime"],
		InfernalCryCooldown: subEnv.Player.Output["InfernalCryCooldown"],
	}
	cached.BattlemageUpTimeRatio, cached.battlemageUpTimeRatioSet =
		subEnv.Player.Output["BattlemageUpTimeRatio"]
	cached.InfernalUpTimeRatio, cached.infernalUpTimeRatioSet =
		subEnv.Player.Output["InfernalUpTimeRatio"]
	cached.TotemLife, cached.totemLifeSet = subEnv.Player.Output["TotemLife"]
	if subEnv.Player.MainSkill != nil && subEnv.Player.MainSkill.SkillData != nil {
		cached.ChanceToTriggerOnCrit = subEnv.Player.MainSkill.SkillData.ChanceToTriggerOnCrit
		cached.lifeReservedPercent = subEnv.Player.MainSkill.SkillData.LifeReservedPercent
		cached.manaReservedPercent = subEnv.Player.MainSkill.SkillData.ManaReservedPercent
	}
	cached.costs = costOutputs(subEnv.Player.Output)
	env.triggerCache.entries[uuid] = cached

	return cached
}

var reuseTriggerSourceEnvs = true

type triggerSourceEnv struct {
	env                                           *Environment
	override                                      *CalcOverride
	cachedPlayerDB, cachedEnemyDB, cachedMinionDB moddb.ModStoreFuncs
}

func triggerSourceEnvFor(env *Environment) *triggerSourceEnv {
	spare := env.triggerCache.spare
	for i := len(spare) - 1; i >= 0 && reuseTriggerSourceEnvs; i-- {
		source := spare[i]
		if source.env.Build != env.Build || source.env.Mode != env.Mode || source.override != env.override {
			continue
		}
		env.triggerCache.spare = slices.Delete(spare, i, i+1)
		source.env, _, _, _ = initEnvSpec(source.env.Build, env.Cache, env.Mode, false, source.env.override, &calcSpecEnv{
			cachedPlayerDB: source.cachedPlayerDB,
			cachedEnemyDB:  source.cachedEnemyDB,
			cachedMinionDB: source.cachedMinionDB,
			env:            source.env,
			accelerate: calcAccelerate{
				nodeAlloc:         true,
				requirementsItems: true,
				requirementsGems:  true,
				skills:            true,
				everything:        true,
			},
			flatDB: true,
		})
		return source
	}

	source := &triggerSourceEnv{override: env.override}
	source.env, source.cachedPlayerDB, source.cachedEnemyDB, source.cachedMinionDB =
		InitEnvOverride(env.Build, env.Cache, env.Mode, env.override)
	return source
}

//pob:block:src/Modules/CalcActiveSkill.lua:164-181
func mirageSkillCopyEffect(newEnv *Environment, actor *Actor, built *ActiveSkill) *ActiveSkill {
	ae := built.ActiveEffect
	if ae == nil {
		return built
	}

	copied := *ae
	copied.SocketQuality = 0
	copied.gemCfg = nil
	copied.gemPropertyInfo = nil
	if ae.SrcInstance != nil {
		copied.Level = ae.SrcInstance.Level
		copied.Quality = ae.SrcInstance.Quality
	}

	socketGroup, _ := built.SocketGroup.(*pob.Skill)

	newSkill := CreateActiveSkill(&copied, built.SupportList, actor, socketGroup, built.SummonSkill)
	newSkill.SocketGroupIndex = built.SocketGroupIndex
	CalcBuildActiveSkillModList(newEnv, newSkill)

	//pob:block:src/Modules/CalcActiveSkill.lua:545-547
	if ae.SrcInstance != nil && ae.SocketQuality != 0 && newSkill.BaseSkillModList != nil {
		newSkill.BaseSkillModList.AddMod(mod.NewFloat("GemSocketQuality", mod.TypeBase,
			float64(ae.SocketQuality)).Source("Socket Quality"))
	}

	for i, skill := range newEnv.Player.ActiveSkillList {
		if skill == built {
			newEnv.Player.ActiveSkillList[i] = newSkill
			break
		}
	}
	newEnv.Player.MainSkill = newSkill

	return newSkill
}

//pob:block:src/Modules/CalcActiveSkill.lua:181
func mirageSkillCopy(env *Environment, newEnv *Environment, built *ActiveSkill) *ActiveSkill {
	newSkill := mirageSkillCopyEffect(newEnv, env.Player, built)
	newSkill.SkillModList = moddb.NewModList()
	newSkill.SkillModList.Parent = newSkill.BaseSkillModList
	newSkill.mirageCopy = true
	//pob:block:src/Modules/CalcMirages.lua:39
	newSkill.SkillCfg.SkillCond["usedByMirage"] = true
	return newSkill
}

//pob:block:src/Modules/CalcPerform.lua:2422-2499
func performMirageArcher(env *Environment) {
	skill := env.Player.MainSkill
	if skill == nil || skill.SkillData == nil || !skill.SkillData.TriggeredByMirageArcher {
		return
	}
	if skill.SkillFlags[SkillFlagMinion] {
		return
	}

	if skill.Marked {
		return
	}

	groupIndex := skill.SocketGroupIndex
	skillIndex := activeSkillIndexInGroup(env, skill)
	if env.Build == nil || groupIndex < 1 || skillIndex < 1 {
		return
	}

	//pob:block:src/Modules/CalcActiveSkill.lua:179
	newEnv := initEnvOnly(env.Build.WithMainSkill(groupIndex, skillIndex), env.Cache, env.Mode, env.override)
	newEnv.triggerCache = env.triggerCache
	newSkill := newEnv.Player.MainSkill
	if newSkill == nil || newSkill.SkillModList == nil {
		return
	}
	newSkill = mirageSkillCopy(env, newEnv, newSkill)

	//	:40  newSkill.skillFlags.multiPart = nil                  -- HERE
	//	:41  newSkill.skillFlags.haveMinion = nil                 -- HERE
	func() {
		delete(newSkill.SkillFlags, SkillFlagMultiPart)
		delete(newSkill.SkillFlags, SkillFlagHaveMinion)
		if newSkill.SkillTypes == nil {
			newSkill.SkillTypes = map[data.SkillType]bool{}
		}
		newSkill.SkillTypes[data.SkillTypeOtherThingUsesSkill] = true
	}()

	//	:70  moreDamage      = newSkill.skillModList:Sum("BASE", newSkill.skillCfg, ...)
	//	:71  moreAttackSpeed = newSkill.skillModList:Sum("BASE", newSkill.skillCfg, ...)
	//	:72  mirageCount     = newSkill.skillModList:Sum("BASE", env.player.mainSkill.skillCfg, ...)
	//pob:block:src/Modules/CalcMirages.lua:70-72
	moreDamage := newSkill.SkillModList.Sum(mod.TypeBase, newSkill.SkillCfg, "MirageArcherLessDamage")
	moreAttackSpeed := newSkill.SkillModList.Sum(mod.TypeBase, newSkill.SkillCfg, "MirageArcherLessAttackSpeed")
	mirageCount := newSkill.SkillModList.Sum(mod.TypeBase, skill.SkillCfg, "MirageArcherMaxCount")

	newSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, moreDamage).Source("Mirage Archer"))
	newSkill.SkillModList.AddMod(mod.NewFloat("Speed", mod.TypeMore, moreAttackSpeed).Source("Mirage Archer"))

	//pob:block:src/Modules/CalcMirages.lua:87
	newSkill.SkillModList.AddMod(mod.NewFlag("HasNoCost", true).Source("Used by mirage"))

	//pob:block:src/Modules/CalcPerform.lua:2675-2677
	//pob:block:src/Modules/CalcMirages.lua:74-76,src/Modules/CalcMirages.lua:89-95
	skill.Mirage = &MirageResult{Name: newSkill.ActiveEffect.GrantedEffect.Name(), Count: mirageCount, SkillPartName: newSkill.SkillPartName}
	//pob:block:src/Modules/CalcMirages.lua:78-80
	if skill.infoMessage == "" {
		skill.infoMessage = luaNumber(mirageCount) + " Mirage Archers using " + newSkill.ActiveEffect.GrantedEffect.Name()
	}

	//pob:block:src/Modules/CalcPerform.lua:2691
	newSkill.Marked = true
	PerformCalc(newEnv)

	skill.Mirage.Output = newEnv.Player.Output
	skill.Mirage.Output["ManaCost"] = 0
}

//pob:block:src/Modules/CalcMirages.lua:22-56,src/Modules/CalcMirages.lua:118-162
func performSaviourMirage(env *Environment) bool {
	mainSkill := env.Player.MainSkill
	if mainSkill == nil || mainSkill.ActiveEffect == nil || mainSkill.ActiveEffect.GrantedEffect == nil ||
		mainSkill.ActiveEffect.GrantedEffect.Name() != "Reflection" {
		return false
	}
	if mainSkill.SkillCfg.SkillCond["usedByMirage"] || mainSkill.SkillFlags[SkillFlagDisable] {
		return false
	}
	maxMirageWarriors := mainSkill.SkillModList.Sum(mod.TypeBase, mainSkill.SkillCfg, "SaviourMirageWarriorMaxCount")

	var mirageSkill *ActiveSkill
	usedSkillBestDps := float64(0)
	for _, skill := range env.Player.ActiveSkillList {
		if !saviourMirageWarriorCandidate(skill, mainSkill) {
			continue
		}
		cached := buildActiveSkillCache(env, skill)
		if cached != nil && cached.CritChance > 0 && (mirageSkill == nil || cached.TotalDPS > usedSkillBestDps) {
			usedSkillBestDps = cached.TotalDPS
			mirageSkill = skill
		}
	}

	if mirageSkill == nil {
		mainSkill.DisableReason = "No Saviour active skill found"
		mainSkill.SkillFlags[SkillFlagDisable] = true
		return true
	}

	groupIndex := mirageSkill.SocketGroupIndex
	skillIndex := activeSkillIndexInGroup(env, mirageSkill)
	if env.Build == nil || groupIndex < 1 || skillIndex < 1 {
		return true
	}
	newEnv := initEnvOnly(env.Build.WithMainSkill(groupIndex, skillIndex), env.Cache, OutputModeCalculator, env.override)
	newEnv.triggerCache = env.triggerCache
	newSkill := newEnv.Player.MainSkill
	if newSkill == nil || newSkill.SkillModList == nil {
		return true
	}
	newSkill = mirageSkillCopy(env, newEnv, newSkill)
	delete(newSkill.SkillFlags, SkillFlagMultiPart)
	delete(newSkill.SkillFlags, SkillFlagHaveMinion)
	newSkill.SkillTypes[data.SkillTypeOtherThingUsesSkill] = true

	moreDamage := mainSkill.SkillModList.Sum(mod.TypeBase, mainSkill.SkillCfg, "SaviourMirageWarriorLessDamage")
	// Add new modifiers to new skill (which already has all the old skill's modifiers)
	newSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, moreDamage).Source("The Saviour"))
	if w1, w2 := env.Player.ItemList["Weapon 1"], env.Player.ItemList["Weapon 2"]; w1 != nil && w2 != nil && w1.Name == w2.Name {
		maxMirageWarriors /= 2
	}
	newSkill.SkillModList.AddMod(mod.NewFloat("QuantityMultiplier", mod.TypeBase, maxMirageWarriors).Source("The Saviour Mirage Warriors"))
	// Does not use player resources
	newSkill.SkillModList.AddMod(mod.NewFlag("HasNoCost", true).Source("Used by mirage"))

	newEnv.Player.MainSkill = newSkill
	PerformCalc(newEnv)

	env.Player.MainSkill = newSkill
	newSkill.infoMessage = luaNumber(maxMirageWarriors) + " Mirage Warriors using " + newSkill.ActiveEffect.GrantedEffect.Name()
	env.Player.Output = newEnv.Player.Output
	if newEnv.Player.Breakdown != nil {
		env.Player.Breakdown = newEnv.Player.Breakdown
	}
	return true
}

//pob:block:src/Modules/CalcMirages.lua:355-357
func performGeneralsCryMarks(env *Environment) {
	skill := env.Player.MainSkill
	if skill == nil || skill.SkillData == nil || !skill.SkillData.TriggeredByGeneralsCry {
		return
	}
	if skill.SkillTypes == nil {
		skill.SkillTypes = map[data.SkillType]bool{}
	}
	skill.SkillTypes[data.SkillTypeTriggered] = true
	skill.SkillCfg.SkillCond["usedByMirage"] = true
	skill.SkillTypes[data.SkillTypeOtherThingUsesSkill] = true
}

//pob:block:src/Modules/CalcMirages.lua:282-343
func performSacredWisps(env *Environment) {
	skill := env.Player.MainSkill
	if skill == nil || skill.SkillData == nil || !skill.SkillData.TriggeredBySacredWisps {
		return
	}
	if skill.Marked || skill.SkillFlags[SkillFlagDisable] {
		return
	}

	groupIndex := skill.SocketGroupIndex
	skillIndex := activeSkillIndexInGroup(env, skill)
	if env.Build == nil || groupIndex < 1 || skillIndex < 1 {
		return
	}

	var wispsCastChance, wispsMaxCount float64
	found := false
	for _, candidate := range env.Player.ActiveSkillList {
		if candidate.ActiveEffect == nil || candidate.ActiveEffect.GrantedEffect == nil {
			continue
		}
		as := candidate.ActiveEffect.GrantedEffect.Raw.GetActiveSkill()
		if as == nil || as.DisplayedName != "Summon Sacred Wisps" {
			continue
		}
		wispsCastChance = candidate.SkillModList.Sum(mod.TypeBase, skill.SkillCfg, "SacredWispsChance")
		wispsMaxCount = candidate.SkillModList.Sum(mod.TypeBase, skill.SkillCfg, "SacredWispsMaxCount")
		found = true
		break
	}
	if !found {
		return
	}

	//pob:block:src/Modules/CalcActiveSkill.lua:179
	newEnv := initEnvOnly(env.Build.WithMainSkill(groupIndex, skillIndex), env.Cache, env.Mode, env.override)
	newEnv.triggerCache = env.triggerCache
	newSkill := newEnv.Player.MainSkill
	if newSkill == nil || newSkill.SkillModList == nil {
		return
	}
	newSkill = mirageSkillCopy(env, newEnv, newSkill)

	//	:40  newSkill.skillFlags.multiPart = nil                  -- HERE
	//	:41  newSkill.skillFlags.haveMinion = nil                 -- HERE
	func() {
		delete(newSkill.SkillFlags, SkillFlagMultiPart)
		delete(newSkill.SkillFlags, SkillFlagHaveMinion)
		if newSkill.SkillTypes == nil {
			newSkill.SkillTypes = map[data.SkillType]bool{}
		}
		newSkill.SkillTypes[data.SkillTypeOtherThingUsesSkill] = true
	}()

	lessDamage := newSkill.SkillModList.Sum(mod.TypeBase, skill.SkillCfg, "SacredWispsLessDamage")

	newSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, lessDamage).Source("Used by Sacred Wisps"))
	newSkill.SkillModList.AddMod(mod.NewFloat("Speed", mod.TypeMore, wispsCastChance-100).Source("Sacred Wisps cast chance"))
	//pob:block:src/Modules/CalcMirages.lua:313
	newSkill.SkillModList.AddMod(mod.NewFlag("HasNoCost", true).Source("Used by Sacred Wisps"))

	//pob:block:src/Modules/CalcMirages.lua:300-302,src/Modules/CalcMirages.lua:315-321
	skill.Mirage = &MirageResult{Name: newSkill.ActiveEffect.GrantedEffect.Name(), Count: wispsMaxCount, SkillPartName: newSkill.SkillPartName}
	//pob:block:src/Modules/CalcMirages.lua:304-306
	if skill.infoMessage == "" {
		skill.infoMessage = luaNumber(wispsMaxCount) + " Sacred Wisps using " + newSkill.ActiveEffect.GrantedEffect.Name()
	}

	newSkill.Marked = true
	PerformCalc(newEnv)

	//pob:block:src/Modules/CalcMirages.lua:324
	skill.Mirage.Output = newEnv.Player.Output
}

func activeSkillIndexInGroup(env *Environment, skill *ActiveSkill) int {
	index := 0
	for _, candidate := range env.Player.ActiveSkillList {
		if candidate.SocketGroupIndex != skill.SocketGroupIndex {
			continue
		}
		index++
		if candidate == skill {
			return index
		}
	}

	return 0
}

//pob:block:src/Modules/CalcTriggers.lua:69
func triggerSourceSpeed(cached *cachedTriggerSkill) (float64, bool) {
	if cached.hitSpeedSet {
		return cached.HitSpeed, true
	}
	return cached.Speed, cached.speedSet
}

//pob:src/Modules/CalcTriggers.lua:68-71
func defaultTriggerComparer(cached *cachedTriggerSkill, source *ActiveSkill, triggerRate float64) bool {
	speed, speedSet := triggerSourceSpeed(cached)
	return (source == nil && speedSet) || (speedSet && speed > triggerRate)
}

//pob:src/Modules/CalcPerform.lua:21-40
func findTriggerSkill(env *Environment, skill *ActiveSkill, source *ActiveSkill, triggerRate float64, reqManaCost *float64, comparer triggerComparer) (*ActiveSkill, float64) {
	cached := buildActiveSkillCache(env, skill)
	if cached == nil {
		return source, triggerRate
	}

	speed, _ := triggerSourceSpeed(cached)

	if comparer == nil {
		comparer = defaultTriggerComparer
	}
	if comparer(cached, source, triggerRate) {
		if !triggerSkillEligible(skill) {
			return source, triggerRate
		}
		if reqManaCost == nil || cached.ManaCost >= *reqManaCost {
			return skill, speed
		}
	}

	return source, triggerRate
}

//pob:block:src/Modules/CalcTriggers.lua:82
func triggerSkillEligible(skill *ActiveSkill) bool {
	if skill.SkillFlags != nil && skill.SkillFlags[SkillFlagDisable] {
		return false
	}
	if skill.Marked {
		return false
	}
	if skill.SkillCfg != nil && skill.SkillCfg.SkillCond["usedByMirage"] {
		return false
	}
	return !skill.SkillTypes[data.SkillTypeOtherThingUsesSkill]
}

type triggerRotationSkill struct {
	uuid     string
	cd       float64
	nextTrig float64
	count    float64

	cdOverride    *float64
	addsCastTime  float64
	icdr          float64
	addedCooldown float64
}

//pob:src/Modules/CalcTriggers.lua:64-66
func packageSkillDataForSimulation(env *Environment, skill *ActiveSkill) *triggerRotationSkill {
	icdr := calclib.Mod(skill.SkillModList, skill.SkillCfg, "CooldownRecovery")
	if icdr == 0 {
		icdr = 1
	}

	var cdOverride *float64
	if o := skill.SkillModList.Override(skill.SkillCfg, "CooldownRecovery"); o != nil {
		cdOverride = utils.Ptr(utils.Or(o, 0))
	}

	return &triggerRotationSkill{
		uuid:          cacheSkillUUID(env, skill),
		cd:            skill.SkillData.Cooldown,
		cdOverride:    cdOverride,
		addsCastTime:  processAddedCastTime(skill),
		icdr:          icdr,
		addedCooldown: skill.SkillModList.Sum(mod.TypeBase, skill.SkillCfg, "CooldownRecovery"),
	}
}

//pob:src/Modules/Common.lua:951-960
func ceilB(x float64, base float64) float64  { return float64(base * math.Ceil(x/base)) }
func floorB(x float64, base float64) float64 { return float64(base * math.Floor(x/base)) }

//pob:src/Modules/CalcTriggers.lua:90-133
type triggerSimBreakdown struct {
	simTime float64
	rates   []triggerSimRate
}

type triggerSimRate struct {
	name string
	rate float64
}

func calcMultiSpellRotationImpact329(env *Environment, skillRotation []*triggerRotationSkill, sourceRate float64, triggerCD float64, chance float64, actor *Actor) (float64, *triggerSimBreakdown) {
	if sourceRate <= 0 || len(skillRotation) == 0 {
		return 0, nil
	}
	if chance == 0 {
		chance = 100
	}
	if actor == nil {
		actor = env.Player
	}

	triggerIncrement := 1 / sourceRate
	simTime := triggerIncrement * 1000

	for _, skill := range skillRotation {
		icdr := skill.icdr
		if icdr == 0 {
			icdr = 1
		}
		triggered := (skill.cd + skill.addedCooldown) / icdr
		if skill.cdOverride != nil {
			triggered = *skill.cdOverride
		}
		skill.cd = math.Max(triggered, (triggerCD+skill.addsCastTime)/icdr)
		skill.nextTrig = 0
		skill.count = 0
	}

	rotationIndex := 0
	nextTrigger := float64(0)
	for nextTrigger < simTime {
		currentIndex := rotationIndex
		for {
			if skillRotation[currentIndex].nextTrig <= nextTrigger {
				skillRotation[currentIndex].count++
				skillRotation[currentIndex].nextTrig = ceilB(
					floorB(nextTrigger, data.ServerTickTime)+skillRotation[currentIndex].cd,
					data.ServerTickTime)
				break
			}
			currentIndex = (currentIndex + 1) % len(skillRotation)
			if currentIndex == rotationIndex {
				break
			}
		}
		rotationIndex = (rotationIndex + 1) % len(skillRotation)
		nextTrigger += triggerIncrement
	}

	trigRateTable := &triggerSimBreakdown{simTime: simTime}
	mainUUID := cacheSkillUUID(env, actor.MainSkill)
	mainRate := float64(0)
	for _, sd := range skillRotation {
		if sd.count == 0 {
			trigRateTable.rates = append(trigRateTable.rates, triggerSimRate{name: sd.uuid})
			continue
		}
		rate := 1 / (simTime/sd.count + float64(triggerIncrement/chance*100) - triggerIncrement)
		trigRateTable.rates = append(trigRateTable.rates, triggerSimRate{name: sd.uuid, rate: rate})
		if sd.uuid == mainUUID {
			mainRate = rate
		}
	}

	return mainRate, trigRateTable
}

//pob:src/Modules/CalcPerform.lua:161-290
func calcActualTriggerRate(env *Environment, source *ActiveSkill, sourceAPS *float64, spellCount []*triggerRotationSkill, output map[string]float64, triggerChance float64) float64 {
	if sourceAPS == nil && source.SkillTypes[data.SkillTypeChannel] {
		triggerTime := source.SkillData.TriggerTime
		if triggerTime == 0 {
			triggerTime = 1
		}
		triggerRateOfTrigger := 1 / triggerTime

		mainSkill := env.Player.MainSkill
		icdr := calclib.Mod(mainSkill.SkillModList, mainSkill.SkillCfg, "CooldownRecovery")
		if icdr == 0 {
			icdr = 1
		}

		addsCastTime := processAddedCastTime(mainSkill)
		if addsCastTime != 0 {
			output["addsCastTime"] = addsCastTime
		}

		triggeredTotalCooldown := math.Max(mainSkill.SkillData.Cooldown, addsCastTime) / icdr
		if cooldownOverride := mainSkill.SkillModList.Override(mainSkill.SkillCfg, "CooldownRecovery"); cooldownOverride != nil {
			triggeredTotalCooldown = utils.Or(cooldownOverride, 0)
		}

		triggeredCDAdjusted := math.Ceil(triggeredTotalCooldown*data.ServerTickRate) / data.ServerTickRate
		effCDTriggeredSkill := math.Ceil(triggeredCDAdjusted*triggerRateOfTrigger) / triggerRateOfTrigger

		//pob:src/Modules/CalcTriggers.lua:246-263
		output["TriggerRateCap"] = math.Min(1/effCDTriggeredSkill, triggerRateOfTrigger)
	} else {
		actionCooldown, addsCastTime := triggerActionCooldown(env.Player.MainSkill, env.Player.MainSkill, false)
		if addsCastTime != 0 {
			output["addsCastTime"] = addsCastTime
		}
		//pob:block:src/Modules/CalcTriggers.lua:583-586
		output["TriggerRateCap"] = math.Inf(1)
		if actionCooldown != 0 {
			output["TriggerRateCap"] = 1 / actionCooldown
		}
	}

	if sourceAPS != nil {
		output["EffectiveSourceRate"] = *sourceAPS
	}

	var trigRate float64
	skillRotationImpact := float64(len(spellCount))
	if sourceAPS != nil {
		simTriggerCD := triggeredByCooldown(env.Player.MainSkill)
		if simTriggerCD == 0 {
			simTriggerCD = env.Player.MainSkill.SkillData.Cooldown
		}
		output["SourceTriggerRate"], _ = calcMultiSpellRotationImpact329(env, spellCount, *sourceAPS, simTriggerCD, triggerChance, env.Player)
		trigRate = math.Min(output["SourceTriggerRate"], output["TriggerRateCap"])

		if env.ModDB.Flag(nil, "HaveTriggerBots") && env.Player.MainSkill.SkillTypes[data.SkillTypeSpell] {
			trigRate *= 2
		}
	} else {
		output["SourceTriggerRate"] = output["TriggerRateCap"] / skillRotationImpact
		trigRate = output["SourceTriggerRate"]
	}

	if output["SourceTriggerRate"] == 0 {
		return 0
	}

	output["ServerTriggerRate"] = trigRate
	output["SkillTriggerRate"] = trigRate

	return trigRate
}

//pob:src/Modules/CalcTriggers.lua:23-30
func addTriggerIncMoreMods(activeSkill *ActiveSkill, sourceSkill *ActiveSkill) {
	for _, value := range activeSkill.SkillModList.Tabulate(mod.TypeIncrease, sourceSkill.SkillCfg, "TriggeredDamage") {
		activeSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, value.Mod.Value().Float()).
			Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
	}
	for _, value := range activeSkill.SkillModList.Tabulate(mod.TypeMore, sourceSkill.SkillCfg, "TriggeredDamage") {
		activeSkill.SkillModList.AddMod(mod.NewFloat("Damage", mod.TypeMore, value.Mod.Value().Float()).
			Source(value.Mod.GetSource()).Flag(value.Mod.Flags()).KeywordFlag(value.Mod.KeywordFlags()).Tag(value.Mod.Tags()...))
	}
}

//pob:src/Data/Skills/other.lua
var itemGrantedSupportNames = map[string]string{
	"SupportBloodMagicUniquePrismGuardian":                          "Blood Magic",
	"SupportCastOnLifeSpent":                                        "Foulborn Kitava's Thirst",
	"SupportCastOnManaSpent":                                        "Kitava's Thirst",
	"SupportCrabTotem":                                              "Crab Totem",
	"SupportCurseOnTrapTriggered":                                   "Hex on Trap",
	"SupportCursePillarTriggerCurses":                               "Doedre's Effigy",
	"SupportDivineBlessing":                                         "Divine Blessing",
	"SupportEarthbreaker":                                           "Earthbreaker",
	"SupportTriggerBowSkillOnBowAttack":                             "Maloney's Mechanism",
	"SupportTriggerElementalSpellOnBlock":                           "Svalinn Cast on Block",
	"SupportTriggerFireSpellOnHit":                                  "SupportTriggerFireSpellOnHit",
	"SupportTriggerSpellFromHelmet":                                 "Focus",
	"SupportTriggerSpellOnAttack":                                   "Poet's Pen",
	"SupportTriggerSpellOnBowAttack":                                "Asenath's Chant",
	"SupportTriggerSpellOnBowAttackFreezeHit":                       "Wing of the Wyvern",
	"SupportTriggerSpellOnKill":                                     "Squirming Terror",
	"SupportTriggerSpellOnSkillUse":                                 "Trigger Craft",
	"SupportTriggerSpellOnUnarmedMeleeCriticalHit":                  "Seven Teachings",
	"SupportUniqueCastCurseOnCurse":                                 "Vixen's Entrapment",
	"SupportUniqueCosprisMaliceColdSpellsCastOnMeleeCriticalStrike": "Cospri's Malice",
	"SupportUniqueMjolnerLightningSpellsCastOnHit":                  "Mjolner",
	"SupportUniqueVirulenceSpellsCastOnBlock":                       "Festering Resentment Cast on Block",
}

//pob:src/Modules/CalcTriggers.lua:1575-1590
func getUniqueItemTriggerName(skill *ActiveSkill) string {
	for _, gemInstance := range skill.SupportList {
		if gemInstance.GrantedEffect == nil {
			continue
		}
		if name, ok := itemGrantedSupportNames[gemInstance.GrantedEffect.ID()]; ok {
			return name
		}
	}

	return ""
}

//pob:src/Modules/CalcPerform.lua:296-311
func calcDualWieldImpact(env *Environment, sourceAPS float64, skillDoubleHitsWhenDualWielding bool) (float64, bool) {
	dualWield := false
	if env.Player.WeaponData1 != nil && env.Player.WeaponData1.Type != "" &&
		env.Player.WeaponData2 != nil && env.Player.WeaponData2.Type != "" {
		dualWield = true
		sourceAPS *= 0.5
	}

	if dualWield && skillDoubleHitsWhenDualWielding {
		sourceAPS *= 2.0
	}

	return sourceAPS, dualWield
}

//pob:block:src/Modules/CalcPerform.lua:2549
func skillCfgHasFlag(skill *ActiveSkill, flag mod.MFlag) bool {
	return skill.SkillCfg != nil && skill.SkillCfg.Flags != nil && *skill.SkillCfg.Flags&flag > 0
}

//pob:src/Modules/CalcPerform.lua:338-345
func getTriggerDefaultCooldown(supportList []*GemEffect, name string) float64 {
	for _, support := range supportList {
		if support.GrantedEffect != nil && support.GrantedEffect.ID() == name {
			level := raw2.GetCalculatedGrantedEffect(support.GrantedEffect.Raw).GetCalculatedLevels()[support.Level]
			if level != nil && level.Cooldown != nil {
				return *level.Cooldown
			}
			return 0
		}
	}

	return 100
}

//pob:block:src/Modules/CalcPerform.lua:2552
func socketGroupSlot(skill *ActiveSkill) string {
	if socketGroup, ok := skill.SocketGroup.(*pob.Skill); ok && socketGroup != nil {
		return socketGroup.Slot
	}

	return ""
}

// Finalises the environment and performs the stat calculations:
// 1. Merges keystone modifiers
// 2. Initialises minion skills
// 3. Initialises the main skill's minion, if present
// 4. Merges flask effects
// 6. Calculates reservations
// 7. Sets life/mana reservation (doActorLifeManaReservation)
// 8. Processes buffs and debuffs
// 10. Calculates defence and offence stats (calcs.defence, calcs.offence)
//
//pob:src/Modules/CalcPerform.lua:1079-3132
func PerformCalc(env *Environment) {
	performCalc(env, false)
}

func reuseMinionSkills(env *Environment, activeSkill *ActiveSkill) {
	if activeSkill == env.Player.MainSkill {
		CalcCreateMinionSkills(env, activeSkill)
		return
	}
	if env.accelerated && activeSkill.reusableMinionSkills != nil {
		activeSkill.Minion.ActiveSkillList = activeSkill.reusableMinionSkills
		activeSkill.Minion.MainSkill = activeSkill.reusableMinionMain
		return
	}
	CalcCreateMinionSkills(env, activeSkill)
	activeSkill.reusableMinionSkills, activeSkill.reusableMinionMain = activeSkill.Minion.ActiveSkillList, activeSkill.Minion.MainSkill
}

//pob:block:src/Modules/CalcPerform.lua:1252
func performCalc(env *Environment, skipEHP bool) {
	env.skipEHP = skipEHP

	if env.triggerCache == nil {
		env.triggerCache = &triggerSkillCache{
			entries:  make(map[string]*cachedTriggerSkill),
			building: make(map[string]bool),
		}
	}

	// Merge keystone modifiers
	env.KeystonesAdded = make(map[string]interface{}, 0)
	mergeKeystones(env, env.ModDB)

	hasBonechill := false
	hasGuaranteedBonechill := false

	for _, activeSkill := range env.Player.ActiveSkillList {
		if !activeSkill.mirageCopy {
			activeSkill.SkillModList = moddb.NewModList()
			activeSkill.SkillModList.Parent = activeSkill.BaseSkillModList
		}
		if activeSkill.Minion != nil {
			// Build minion skills
			if activeSkill == env.Player.MainSkill {
				activeSkill.Minion.ModDB = moddb.NewModDBSized(192)
			} else {
				activeSkill.Minion.ModDB = moddb.NewModDB()
			}
			activeSkill.Minion.ModDB.Actor = activeSkill.Minion
			reuseMinionSkills(env, activeSkill)
			//pob:block:src/Modules/CalcPerform.lua:1267
			activeSkill.SkillPartName = activeSkill.Minion.MainSkill.ActiveEffect.GrantedEffect.Name()
		}
	}

	env.Player.Output = takeSpareOutput(&env.spare.player)
	env.Player.OutputTable = make(map[OutTable]map[string]float64)
	env.Player.OutputStrings = make(map[string]string)

	env.Enemy.Output = make(map[string]float64)
	env.Enemy.OutputTable = make(map[OutTable]map[string]float64)
	env.Enemy.OutputStrings = make(map[string]string)

	env.Minion = env.Player.MainSkill.Minion
	if env.Minion != nil {
		// Initialise minion modifier database
		env.Player.OutputTable["Minion"] = takeSpareOutput(&env.spare.minion)
		initMinionModDB(env, env.Player.MainSkill, env.Player.OutputTable["Minion"])
	}

	if env.AegisModList != nil {
		env.Player.ItemList["Weapon 2"] = nil
	}

	if env.ModDB.Flag(nil, "AlchemistsGenius") {
		effectMod := 1 + env.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100
		env.ModDB.AddMod(mod.NewFloat("FlaskEffect", mod.TypeIncrease, math.Floor(10*effectMod)).Source("Alchemist's Genius"))
		env.ModDB.AddMod(mod.NewFloat("FlaskChargesGained", mod.TypeIncrease, math.Floor(20*effectMod)).Source("Alchemist's Genius"))
	}

	//pob:block:src/Modules/CalcPerform.lua:1305-1312
	if env.ModDB.Flag(nil, "CryWolfMinimumPower") && env.ModDB.Sum(mod.TypeBase, nil, "WarcryPower") < 10 {
		env.ModDB.AddMod(mod.NewFloat("WarcryPower", "OVERRIDE", 10).Source("Minimum Warcry Power from CryWolf"))
	}
	if env.ModDB.Flag(nil, "WarcryInfinitePower") {
		env.ModDB.AddMod(mod.NewFloat("WarcryPower", "OVERRIDE", 999999).Source("Warcries have infinite power"))
	}
	env.Player.Output["WarcryPower"] = utils.Or(env.ModDB.Override(nil, "WarcryPower"), env.ModDB.Sum(mod.TypeBase, nil, "WarcryPower"))
	env.ModDB.Multipliers["WarcryPower"] = env.Player.Output["WarcryPower"]

	//pob:block:src/Modules/CalcPerform.lua:1314-1318
	applyEnemyModifiers(env.Player, true)
	if env.Minion != nil {
		applyEnemyModifiers(env.Minion.Actor, true)
	}
	applyEnemyModifiers(env.Enemy, true)

	minionCounts := map[string]*minionCountEntry{}
	summonedMinions := float64(0)

	for _, activeSkill := range env.Player.ActiveSkillList {
		if activeSkill.SkillTypes[data.SkillTypeBrand] {
			attachLimit := min(
				activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "BrandsAttachedLimit"),
				activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "ActiveBrandLimit"),
			)
			configured := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:ConfigBrandsAttachedToEnemy")
			actual := attachLimit
			if configured > 0 {
				actual = min(configured, attachLimit)
			}
			activeSkill.SkillData.AttachedBrandCount = actual
			activeBrands := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:ConfigActiveBrands")
			// Cap the number of active brands by the limit, which is 3 by default
			// Also consider increase to number of active brands from other sources (e.g. Foulgrasp Support)
			env.ModDB.Multipliers["ActiveBrand"] = max(
				min(activeBrands, activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "ActiveBrandLimit")),
				env.ModDB.Multipliers["ActiveBrand"],
			)
			env.ModDB.Multipliers["BrandsAttachedToEnemy"] = max(actual, env.ModDB.Multipliers["BrandsAttachedToEnemy"])
			env.EnemyModDB.Multipliers["BrandsAttached"] = max(actual, env.EnemyModDB.Multipliers["BrandsAttached"])
		}

		//pob:block:src/Modules/CalcPerform.lua:1334-1339
		if activeSkill.SkillFlags[SkillFlagTotem] {
			limit := env.Player.MainSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
			env.Player.Output["ActiveTotemLimit"] = max(limit, env.Player.Output["ActiveTotemLimit"])
			env.Player.Output["TotemsSummoned"] = utils.Or(env.ModDB.Override(nil, "TotemsSummoned"), env.Player.Output["ActiveTotemLimit"])
			env.EnemyModDB.Multipliers["TotemsSummoned"] = max(env.Player.Output["TotemsSummoned"], env.EnemyModDB.Multipliers["TotemsSummoned"])
		}

		// The actual hexes as opposed to hex related skills all have the curse flag. TotemCastsWhenNotDetached is to remove blasphemy
		// Note that this doesn't work for triggers yet, insufficient support
		//pob:block:src/Modules/CalcPerform.lua:1342-1351
		if activeSkill.SkillFlags[SkillFlagHex] && activeSkill.SkillFlags[SkillFlagCurse] && !activeSkill.SkillTypes[data.SkillTypeTotemCastsWhenNotDetached] {
			hexDoom := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:HexDoomStack")
			maxDoom := activeSkill.SkillModList.Sum(mod.TypeBase, nil, "MaxDoom")
			doomEffect := activeSkill.SkillModList.More(nil, "DoomEffect")
			// Update the max doom limit
			env.Player.Output["HexDoomLimit"] = max(maxDoom, env.Player.Output["HexDoomLimit"])
			// Update the Hex Doom to apply
			activeSkill.SkillModList.AddMod(mod.NewFloat("CurseEffect", mod.TypeIncrease, min(hexDoom, maxDoom)*doomEffect).Source("Doom"))
			env.ModDB.Multipliers["HexDoom"] = min(max(hexDoom, env.ModDB.Multipliers["HexDoom"]), env.Player.Output["HexDoomLimit"])
		}

		//pob:src/Modules/CalcPerform.lua:1357-1359
		if activeSkill.SkillData.SupportBonechill &&
			(activeSkill.SkillTypes[data.SkillTypeChillingArea] ||
				activeSkill.SkillTypes[data.SkillTypeNonHitChill] ||
				!activeSkill.SkillModList.Flag(nil, "CannotChill")) {
			hasBonechill = true
		}

		if as := activeSkill.ActiveEffect.GrantedEffect.Raw.GetActiveSkill(); as != nil && (as.DisplayedName == "Vaal Lightning Trap" || as.DisplayedName == "Shock Ground") {
			// Shock effect applies to shocked ground
			effect := activeSkill.SkillModList.Sum(mod.TypeBase, nil, "ShockedGroundEffect") *
				(1 + activeSkill.SkillModList.Sum(mod.TypeIncrease, nil, "EnemyShockEffect")/100)
			env.ModDB.AddMod(mod.NewFloat("ShockOverride", mod.TypeBase, effect).Source("Shocked Ground").Tag(mod.ActorCondition("enemy", "OnShockedGround")))
		}

		if as := activeSkill.ActiveEffect.GrantedEffect.Raw.GetActiveSkill(); as != nil && as.DisplayedName == "Summon Skitterbots" {
			skitterbotAilmentEffect := activeSkill.SkillModList.Sum(mod.TypeIncrease, nil, "SkitterbotAilmentEffect")
			if !activeSkill.SkillModList.Flag(nil, "SkitterbotsCannotShock") {
				effect := *data.NonDamagingAilments[data.AilmentShock].Default * (1 + (activeSkill.SkillModList.Sum(mod.TypeIncrease, &moddb.ListCfg{
					Source: utils.Ptr(mod.SourceSkill),
				}, "EnemyShockEffect")+skitterbotAilmentEffect)/100)
				env.ModDB.AddMod(mod.NewFloat("ShockOverride", mod.TypeBase, effect).Source("Summon Skitterbots"))
				env.EnemyModDB.AddMod(mod.NewFlag("Condition:Shocked", true).Source("Summon Skitterbots"))
				if activeSkill.SkillModList.Flag(nil, "SkitterbotAffectPlayer") {
					env.ModDB.AddMod(mod.NewFlag("Shock", true).Source("Summon Skitterbots"))
					env.ModDB.AddMod(mod.NewFloat("SelfShockOverride", mod.TypeBase, effect).Source("Summon Skitterbots"))
				}
			}
			if !activeSkill.SkillModList.Flag(nil, "SkitterbotsCannotChill") {
				effect := *data.NonDamagingAilments[data.AilmentChill].Default * (1 + (activeSkill.SkillModList.Sum(mod.TypeIncrease, &moddb.ListCfg{
					Source: utils.Ptr(mod.SourceSkill),
				}, "EnemyChillEffect")+skitterbotAilmentEffect)/100)
				env.ModDB.AddMod(mod.NewFloat("ChillOverride", mod.TypeBase, effect).Source("Summon Skitterbots"))
				env.EnemyModDB.AddMod(mod.NewFlag("Condition:Chilled", true).Source("Summon Skitterbots"))
				if activeSkill.SkillModList.Flag(nil, "SkitterbotAffectPlayer") {
					env.ModDB.AddMod(mod.NewFlag("Chill", true).Source("Summon Skitterbots"))
					env.ModDB.AddMod(mod.NewFloat("SelfChillOverride", mod.TypeBase, effect).Source("Summon Skitterbots"))
				}
				if activeSkill.SkillData.SupportBonechill {
					hasGuaranteedBonechill = true
					env.ModDB.AddMod(mod.NewFlag("SkitterbotBonechill", true).Source("Summon Skitterbots"))
				}
			}
			if activeSkill.SkillModList.Flag(nil, "ScorchingSkitterbot") {
				effect := *data.NonDamagingAilments[data.AilmentScorch].Default * (1 + (activeSkill.SkillModList.Sum(mod.TypeIncrease, &moddb.ListCfg{
					Source: utils.Ptr(mod.SourceSkill),
				}, "EnemyScorchEffect")+skitterbotAilmentEffect)/100)
				env.ModDB.AddMod(mod.NewFloat("ScorchOverride", mod.TypeBase, effect).Source("Summon Skitterbots"))
				env.EnemyModDB.AddMod(mod.NewFlag("Condition:Scorched", true).Source("Summon Skitterbots"))
				if activeSkill.SkillModList.Flag(nil, "SkitterbotAffectPlayer") {
					env.ModDB.AddMod(mod.NewFlag("Scorch", true).Source("Summon Skitterbots"))
					env.ModDB.AddMod(mod.NewFloat("SelfScorchOverride", mod.TypeBase, effect).Source("Summon Skitterbots"))
				}
			}
		} else if activeSkill.SkillTypes[data.SkillTypeChillingArea] ||
			(activeSkill.SkillTypes[data.SkillTypeNonHitChill] && !activeSkill.SkillModList.Flag(nil, "CannotChill")) {
			effect := *data.NonDamagingAilments[data.AilmentChill].Default *
				calclib.Mod(activeSkill.SkillModList, activeSkill.SkillCfg, "EnemyChillEffect")
			source := mod.Source("")
			if as := activeSkill.ActiveEffect.GrantedEffect.Raw.GetActiveSkill(); as != nil {
				source = mod.Source(as.DisplayedName)
			}
			env.ModDB.AddMod(mod.NewFloat("ChillOverride", mod.TypeBase, effect).Source(source))
			env.EnemyModDB.AddMod(mod.NewFlag("Condition:Chilled", true).Source(source))
			if activeSkill.SkillData.SupportBonechill {
				hasGuaranteedBonechill = true
			}
		}
		if env.ModDB.Flag(nil, "Condition:BannerPlanted") && env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:BannerValour") == 0 {
			maxValour := env.ModDB.Sum(mod.TypeBase, nil, "MaximumValour")
			stacks := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:ValourStacks")
			env.ModDB.AddMod(mod.NewFloat("Multiplier:BannerValour", mod.TypeBase, math.Min(stacks, maxValour)).Source("Base"))
		}
		if env.ModeBuffs && activeSkill.SkillFlags[SkillFlagWarcry] {
			warcryName := warcryDisplayName(activeSkill)
			if warcryName == "Rallying Cry" && !activeSkill.SkillModList.Flag(nil, "CannotShareWarcryBuffs") && !env.ModDB.Flag(nil, "RallyingActive") {
				env.Player.ModDB.AddMod(mod.NewFloat("RallyingExertMoreDamagePerAlly", mod.TypeBase, activeSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "RallyingCryExertDamageBonus")))
				env.ModDB.AddMod(mod.NewFlag("RallyingActive", true)) // Prevents effect from applying multiple times
			} else if warcryName == "Seismic Cry" && !env.ModDB.Flag(nil, "SeismicActive") {
				env.Player.ModDB.AddMod(mod.NewFloat("SeismicMoreAoE", mod.TypeBase, activeSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "SeismicAoEMoreMultiplier")))
				env.ModDB.AddMod(mod.NewFlag("SeismicActive", true)) // Prevents effect from applying multiple times
			}
		}

		if activeSkill.SkillData.TriggeredOnDeath && !activeSkill.SkillFlags[SkillFlagMinion] {
			activeSkill.SkillData.Triggered = true
			addTriggerIncMoreMods(activeSkill, env.Player.MainSkill)
			// Set trigger time to 1 min in ms ( == 6000 ). Technically any large value would do.
			activeSkill.SkillData.TriggerTime = 60 * 1000
		}
		//pob:block:src/Modules/CalcPerform.lua:1449-1453
		if activeSkill.ActiveEffect.GrantedEffect.Name() == "Reflection" || activeSkill.SkillData.TriggeredBySaviour {
			activeSkill.infoMessage = "Triggered by a Crit from The Saviour"
			activeSkill.infoTrigger = "Saviour"
		}

		if !activeSkill.SkillFlags[SkillFlagDisable] && len(activeSkill.MinionList) > 0 {
			for _, minionType := range activeSkill.MinionList {
				minionData := raw2.Minions[minionType]
				if minionData == nil || minionData.Hostile {
					continue
				}
				key := activeSkill.ActiveEffect.GrantedEffect.Raw.ID + ":" + minionType
				count := float64(1)
				if minionData.Limit != "" {
					key = minionData.Limit
					if o := env.ModDB.Override(nil, minionData.Limit); o != nil {
						count = math.Floor(o.Float())
					} else {
						count = math.Floor(calclib.Val(activeSkill.SkillModList, minionData.Limit) *
							activeSkill.SkillModList.More(activeSkill.SkillCfg, "ActiveMinionLimit"))
					}
					env.Player.Output[minionData.Limit] = max(count, env.Player.Output[minionData.Limit])
				}
				c := minionCounts[key]
				if c == nil {
					c = &minionCountEntry{}
					minionCounts[key] = c
				}
				c.total = max(count, c.total)
				if !activeSkill.SkillTypes[data.SkillTypeVaal] {
					c.nonVaal = max(count, c.nonVaal)
					c.hasNonVaal = true
				}
				if activeSkill.SkillFlags[SkillFlagPermanentMinion] {
					c.permanent = max(count, c.permanent)
					c.hasPermanent = true
				}
			}
		}

		//pob:src/Modules/CalcPerform.lua:1426-1428
		if activeSkill.SkillTypes[data.SkillTypeCreatesMinion] &&
			!activeSkill.SkillTypes[data.SkillTypeMinionsAreUndamagable] {
			env.ModDB.AddMod(mod.NewFlag("Condition:HaveDamageableMinion", true))
		}
	}

	minionCountKeys := make([]string, 0, len(minionCounts))
	for k := range minionCounts {
		minionCountKeys = append(minionCountKeys, k)
	}
	sort.Strings(minionCountKeys)
	for _, k := range minionCountKeys {
		c := minionCounts[k]
		env.ModDB.AddMod(mod.NewFloat("Multiplier:SummonedMinion", mod.TypeBase, c.total).
			Source("Config").Tag(mod.Condition("Combat")))
		if c.hasNonVaal {
			env.ModDB.AddMod(mod.NewFloat("Multiplier:NonVaalSummonedMinion", mod.TypeBase, c.nonVaal).
				Source("Config").Tag(mod.Condition("Combat")))
		}
		if c.hasPermanent {
			env.ModDB.AddMod(mod.NewFloat("Multiplier:PermanentMinion", mod.TypeBase, c.permanent).
				Source("Config").Tag(mod.Condition("Combat")))
		}
		summonedMinions += c.total
	}
	summonedMinionOverride := env.Build.GetNumberOption("multiplierSummonedMinion")
	if summonedMinionOverride != 0 {
		summonedMinions = summonedMinionOverride
	}
	env.ModDB.Conditions["OnlyMinion"] = summonedMinions == 1

	//pob:block:src/Modules/CalcPerform.lua:1479-1487
	var breakdown *Breakdown
	if env.Mode != OutputModeCalculator {
		breakdown = NewBreakdown(env.ModDB, env.Player.Output, env.Player)
	}
	env.Player.Breakdown = breakdown
	if env.Minion != nil {
		env.Minion.Breakdown = nil
		if breakdown != nil {
			env.Minion.Breakdown = NewBreakdown(env.Minion.ModDB, env.Minion.Output, env.Minion.Actor)
		}
	}

	//pob:block:src/Modules/CalcPerform.lua:1489-1503
	if env.ModDB.Flag(nil, "ConvertArmourESToLife") {
		for _, slot := range []string{"Helmet", "Gloves", "Boots", "Body Armour", "Weapon 2", "Weapon 3"} {
			item := env.Player.ItemList[slot]
			if item == nil || item.ArmourData == nil {
				continue
			}
			if energyShieldBase := item.ArmourData.EnergyShield; energyShieldBase > 0 {
				env.ModDB.AddMod(mod.NewFloat("Life", mod.TypeBase, energyShieldBase).
					Source(mod.Source(slot + " ES to Life Conversion")))
			}
		}
	}

	//pob:block:src/Modules/CalcPerform.lua:1505-1529
	for _, element := range []string{"Lightning", "Fire", "Cold", "Chaos", "Physical"} {
		if !env.ModDB.Flag(nil, element+"DamageAppliesTo"+element+"AuraEffect") {
			continue
		}
		multiplier := env.ModDB.Sum(mod.TypeBase, nil, "Improved"+element+"DamageAppliesTo"+element+"AuraEffect") / 100
		limit := env.ModDB.Max(nil, element+"DamageAppliesTo"+element+"AuraEffectLimit")
		totalConverted := float64(0)
		for _, value := range env.ModDB.Tabulate(mod.TypeIncrease, &moddb.ListCfg{}, element+"Damage") {
			m := value.Mod
			modifiers := calclib.GetConvertedModTags(m, multiplier, true)
			converted := math.Floor(m.Value().Float() * multiplier)
			if limit > 0 && converted > 0 {
				remaining := limit - totalConverted
				if remaining <= 0 {
					converted = 0
				} else {
					converted = math.Min(converted, remaining)
					totalConverted += converted
				}
			}
			if converted != 0 {
				env.ModDB.AddMod(mod.NewFloat("AuraEffect", m.Type(), converted).
					Source(m.GetSource()).Flag(m.Flags()).
					KeywordFlag(elementKeywordFlag[element]).Tag(modifiers...))
			}
		}
	}

	//pob:block:src/Modules/CalcPerform.lua:1531-1541
	if env.ModDB.Flag(nil, "MinionLifeAppliesToPlayer") {
		// Minion Life conversion from Rigwald's Hunt
		multiplier := utils.OrF(env.ModDB.Max(nil, "ImprovedMinionLifeAppliesToPlayer"), 100) / 100
		for _, value := range env.ModDB.List(nil, "MinionModifier") {
			cValue, ok := value.(mod.MinionModifier)
			if !ok || cValue.Mod == nil {
				continue
			}
			if cValue.Mod.Name() != "Life" || cValue.Mod.Type() != mod.TypeIncrease {
				continue
			}
			m := cValue.Mod
			modifiers := calclib.GetConvertedModTags(m, multiplier, true)
			env.ModDB.AddMod(mod.NewFloat("Life", mod.TypeIncrease, math.Floor(m.Value().Float()*multiplier)).
				Source(m.GetSource()).Flag(m.Flags()).KeywordFlag(m.KeywordFlags()).Tag(modifiers...))
		}
	}

	//pob:block:src/Modules/CalcPerform.lua:1543-1565
	maxLeftActiveMagicUtility := env.ModDB.Sum(mod.TypeBase, nil, "LeftActiveMagicUtilityFlasks")
	maxRightActiveMagicUtility := env.ModDB.Sum(mod.TypeBase, nil, "RightActiveMagicUtilityFlasks")
	if maxLeftActiveMagicUtility > 0 || maxRightActiveMagicUtility > 0 {
		magicUtility := make([]flaskSlotItem, 0, 5)
		for _, slot := range orderedFlaskSlots(env) {
			item := slot.item
			if item.Rarity == RarityMagic && !(strings.Contains(item.BaseName, "Life Flask") ||
				strings.Contains(item.BaseName, "Mana Flask") || strings.Contains(item.BaseName, "Hybrid Flask")) {
				magicUtility = append(magicUtility, slot)
			}
		}

		activated := make(map[int]bool, len(magicUtility))
		activate := func(i int) {
			slot := magicUtility[i]
			if slot.active || activated[i] {
				return
			}
			activated[i] = true
			env.Flasks = append(env.Flasks, slot.item)
		}

		for i := 0; i < min(int(maxLeftActiveMagicUtility), len(magicUtility)); i++ {
			activate(i)
		}
		for i := len(magicUtility) - 1; i >= max(len(magicUtility)-int(maxRightActiveMagicUtility), 0); i-- {
			activate(i)
		}
	}

	flaskMinion := newFlaskMinionState(env)

	//pob:src/Modules/CalcPerform.lua:1812-1814
	tinctureEffectInc := env.ModDB.Sum(mod.TypeIncrease, playerActorCfg, "TinctureEffect")
	tinctureEffectIncMagic := env.ModDB.Sum(mod.TypeIncrease, playerActorCfg, "MagicTinctureEffect")
	tinctureLimit := env.ModDB.Sum(mod.TypeBase, nil, "TinctureLimit")

	// Merge flask modifiers
	if env.ModeCombat {
		flaskMinion.reset(env)
		effectInc := env.ModDB.Sum(mod.TypeIncrease, playerActorCfg, "FlaskEffect")
		flaskBuffs := make(map[string]*moddb.ModList)
		flaskBuffOrder := make([]string, 0, len(env.Flasks)*2)
		flaskBuffsNonUtility := make(map[string]*moddb.ModList)
		flaskBuffNonUtilityOrder := make([]string, 0, len(env.Flasks))
		usingFlask := false
		usingLifeFlask := false
		usingManaFlask := false
		usingNonUtilityFlask := false
		for _, item := range env.Flasks {
			isLifeFlask := item.Base != nil && item.Base.Flask != nil && item.Base.Flask.Life > 0
			isManaFlask := item.Base != nil && item.Base.Flask != nil && item.Base.Flask.Mana > 0

			if getFlaskInstantRecovery(env, item) < 100 {
				usingFlask = true
				if isLifeFlask || isManaFlask {
					usingNonUtilityFlask = true
				}
				if isLifeFlask && !env.ModDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
					usingLifeFlask = true
				}
				if isManaFlask {
					usingManaFlask = true
				}
			}

			flaskEffectInc := item.FlaskData["effectInc"]
			if item.Rarity == RarityMagic && !(isLifeFlask || isManaFlask) {
				flaskEffectInc += env.ModDB.Sum(mod.TypeIncrease, playerActorCfg, "MagicUtilityFlaskEffect")
			}

			// Avert thine eyes, lest they be forever scarred
			// I have no idea how to determine which buff is applied by a given flask,
			// so utility flasks are grouped by base, unique flasks are grouped by name, and magic flasks by their modifiers
			effectMod := 1 + (effectInc+flaskEffectInc)/100
			if len(item.BuffModList) > 0 {
				srcList := moddb.NewModList()
				srcList.ScaleAddList(asModList(item.BuffModList), effectMod)
				flaskBuffOrder = mergeFlaskBuff(srcList, flaskBuffs, flaskBuffOrder, item.BaseName)
			}
			if len(item.Mods) > 0 {
				srcList := moddb.NewModList()
				srcList.ScaleAddList(asModList(item.Mods), effectMod)
				key := item.Title
				if item.Rarity != RarityUnique && item.Rarity != RarityRelic {
					key = flaskModKey(item.Mods)
				}
				flaskBuffOrder = mergeFlaskBuff(srcList, flaskBuffs, flaskBuffOrder, key)
				//pob:src/Modules/CalcPerform.lua:1719
				flaskBuffNonUtilityOrder = mergeFlaskBuff(srcList, flaskBuffsNonUtility, flaskBuffNonUtilityOrder, key)
			}
			//pob:block:src/Modules/CalcPerform.lua:1696-1701
			flaskMinion.calcFlaskMods(item, item.BaseName, item.BuffModList, item.Mods)
		}
		//pob:block:src/Modules/CalcPerform.lua:1775-1789
		if env.ModDB.Flag(nil, "UtilityFlasksDoNotApplyToPlayer") {
			env.ModDB.Conditions["UsingFlask"] = usingNonUtilityFlask
			env.ModDB.Conditions["UsingLifeFlask"] = usingLifeFlask
			env.ModDB.Conditions["UsingManaFlask"] = usingManaFlask
			for _, key := range flaskBuffNonUtilityOrder {
				env.ModDB.AddList(flaskBuffsNonUtility[key])
			}
		} else if !env.ModDB.Flag(nil, "FlasksDoNotApplyToPlayer") {
			env.ModDB.Conditions["UsingFlask"] = usingFlask
			env.ModDB.Conditions["UsingLifeFlask"] = usingLifeFlask
			env.ModDB.Conditions["UsingManaFlask"] = usingManaFlask
			for _, key := range flaskBuffOrder {
				env.ModDB.AddList(flaskBuffs[key])
			}
		}
		flaskMinion.applyToMinion(env)
	}

	// Merge keystones again to catch any that were added by flasks
	mergeKeystones(env, env.ModDB)

	// Calculate attributes and life/mana pools
	doActorAttribsPoolsConditions(env, env.Player)

	//pob:src/Modules/CalcPerform.lua:1609-1621
	if env.Minion != nil {
		for _, value := range env.Player.MainSkill.SkillModList.List(env.Player.MainSkill.SkillCfg, "MinionModifier") {
			minionMod, ok := value.(mod.MinionModifier)
			if !ok {
				continue
			}
			env.Minion.ModDB.AddMod(minionMod.Mod)
		}
		for _, value := range env.Minion.ModDB.List(nil, "Keystone") {
			name, ok := value.(string)
			if !ok {
				continue
			}
			if list := keystoneModList(env, env.Spec.TreeVersion, name); list != nil {
				env.Minion.ModDB.AddList(list)
			}
		}
		doActorAttribsPoolsConditions(env, env.Minion.Actor)
	}

	env.Player.Output["reserved_LifeBase"] = 0
	env.Player.Output["reserved_LifePercent"] = env.ModDB.Sum(mod.TypeBase, nil, "ExtraLifeReserved")
	env.Player.Output["reserved_ManaBase"] = 0
	env.Player.Output["reserved_ManaPercent"] = 0
	env.Player.Output["uncancellable_LifeReservation"] = env.ModDB.Sum(mod.TypeBase, nil, "ExtraLifeReserved")
	env.Player.Output["uncancellable_ManaReservation"] = env.ModDB.Sum(mod.TypeBase, nil, "ExtraManaReserved")
	for _, activeSkill := range env.Player.ActiveSkillList {
		if (activeSkill.SkillTypes[data.SkillTypeHasReservation] || activeSkill.SkillData.TriggeredByAutoexertion) &&
			!activeSkill.SkillTypes[data.SkillTypeReservationBecomesCost] {
			skillModList := activeSkill.SkillModList
			skillCfg := activeSkill.SkillCfg
			mult := skillModList.More(skillCfg, "SupportManaMultiplier")
			pool := map[string]map[string]float64{
				"Mana": {},
				"Life": {},
			}
			pool["Mana"]["baseFlat"] = utils.OrNil(activeSkill.SkillData.ManaReservationFlat, utils.OrNil(activeSkill.ActiveEffect.GrantedEffectLevel.ManaReservationFlat, 0))
			if skillModList.Flag(skillCfg, "ManaCostGainAsReservation") && activeSkill.ActiveEffect.GrantedEffectLevel.Cost != nil {
				pool["Mana"]["baseFlat"] = skillModList.Sum(mod.TypeBase, skillCfg, "ManaCostBase") + float64(activeSkill.ActiveEffect.GrantedEffectLevel.Cost["Mana"])
			}
			pool["Mana"]["basePercent"] = utils.OrNil(activeSkill.SkillData.ManaReservationPercent, utils.OrNil(activeSkill.ActiveEffect.GrantedEffectLevel.ManaReservationPercent, 0))
			pool["Life"]["baseFlat"] = utils.OrNil(activeSkill.SkillData.LifeReservationFlat, utils.OrNil(activeSkill.ActiveEffect.GrantedEffectLevel.LifeReservationFlat, 0))
			if skillModList.Flag(skillCfg, "LifeCostGainAsReservation") && activeSkill.ActiveEffect.GrantedEffectLevel.Cost != nil {
				pool["Life"]["baseFlat"] = skillModList.Sum(mod.TypeBase, skillCfg, "LifeCostBase") + float64(activeSkill.ActiveEffect.GrantedEffectLevel.Cost["Life"])
			}
			pool["Life"]["basePercent"] = utils.OrNil(activeSkill.SkillData.LifeReservationPercent, utils.OrNil(activeSkill.ActiveEffect.GrantedEffectLevel.LifeReservationPercent, 0))
			if skillModList.Flag(skillCfg, "BloodMagicReserved") {
				pool["Life"]["baseFlat"] = pool["Life"]["baseFlat"] + pool["Mana"]["baseFlat"]
				pool["Mana"]["baseFlat"] = 0
				activeSkill.SkillData.LifeReservationFlatForced = activeSkill.SkillData.ManaReservationFlatForced
				activeSkill.SkillData.ManaReservationFlatForced = nil
				pool["Life"]["basePercent"] = pool["Life"]["basePercent"] + pool["Mana"]["basePercent"]
				pool["Mana"]["basePercent"] = 0
				activeSkill.SkillData.LifeReservationPercentForced = activeSkill.SkillData.ManaReservationPercentForced
				activeSkill.SkillData.ManaReservationPercentForced = nil
			}
			for name, values := range pool {
				rn := reservationNames[name]
				values["more"] = skillModList.More(skillCfg, rn.reserved, "Reserved")
				values["inc"] = skillModList.Sum(mod.TypeIncrease, skillCfg, rn.reserved, "Reserved")
				values["efficiency"] = max(skillModList.Sum(mod.TypeIncrease, skillCfg, rn.efficiency, "ReservationEfficiency"), -100)
				values["efficiencyMore"] = skillModList.More(skillCfg, rn.efficiency, "ReservationEfficiency")
				// used for Arcane Cloak calculations in ModStore.GetStat
				env.Player.Output[rn.outEfficiency] = values["efficiency"]
				if utils.GetOr[*float64](activeSkill.SkillData, rn.flatForced, nil) != nil {
					values["reservedFlat"] = *utils.GetOr[*float64](activeSkill.SkillData, rn.flatForced, nil)
				} else {
					baseFlatVal := math.Floor(values["baseFlat"] * mult)
					values["reservedFlat"] = 0
					if values["more"] > 0 && values["inc"] > -100 && baseFlatVal != 0 {
						values["reservedFlat"] = max(utils.RoundTo(baseFlatVal*(100+values["inc"])/100*values["more"]/(1+values["efficiency"]/100)/values["efficiencyMore"], 0), 0)
					}
				}
				if utils.GetOr[*float64](activeSkill.SkillData, rn.percentForced, nil) != nil {
					values["reservedPercent"] = *utils.GetOr[*float64](activeSkill.SkillData, rn.percentForced, nil)
				} else {
					basePercentVal := values["basePercent"] * mult
					values["reservedPercent"] = 0
					if values["more"] > 0 && values["inc"] > -100 && basePercentVal != 0 {
						values["reservedPercent"] = max(utils.RoundTo(basePercentVal*(100+values["inc"])/100*values["more"]/(1+values["efficiency"]/100)/values["efficiencyMore"], 2), 0)
					}
				}
				if activeSkill.ActiveMineCount != nil {
					values["reservedFlat"] = values["reservedFlat"] * *activeSkill.ActiveMineCount
					values["reservedPercent"] = values["reservedPercent"] * *activeSkill.ActiveMineCount
				}
				//pob:block:src/Modules/CalcPerform.lua:1994-1998
				if activeSkill.SkillCfg.SkillName == "Blood Sacrament" {
					values["reservedFlat"] = values["reservedFlat"] * (activeSkill.ActiveStageCount + 1)
					values["reservedPercent"] = values["reservedPercent"] * (activeSkill.ActiveStageCount + 1)
				}
				if values["reservedFlat"] != 0 {
					utils.Set(activeSkill.SkillData, rn.reservedBase, values["reservedFlat"])
					env.Player.Output[rn.reservedBaseOut] = env.Player.Output[rn.reservedBaseOut] + values["reservedFlat"]
					if breakdown != nil {
						breakdown.Reservation(rn.reserved, BReservation{
							SkillName:      activeSkill.ActiveEffect.GrantedEffect.Name(),
							Base:           fmt.Sprint(values["baseFlat"]),
							Mult:           utils.Ternary(mult != 1, utils.Ptr("x "+fmt.Sprint(mult)), nil),
							More:           utils.Ternary(values["more"] != 1, utils.Ptr("x "+fmt.Sprint(values["more"])), nil),
							Inc:            utils.Ternary(values["inc"] != 0, utils.Ptr("x "+fmt.Sprint(1+values["inc"]/100)), nil),
							Efficiency:     utils.Ternary(values["efficiency"] != 0, utils.Ptr("x "+fmt.Sprint(utils.RoundTo(100/(100+values["efficiency"]), 4))), nil),
							EfficiencyMore: utils.Ternary(values["efficiencyMore"] != 1, utils.Ptr("x "+fmt.Sprint(values["efficiencyMore"])), nil),
							Total:          fmt.Sprint(values["reservedFlat"]),
						})
					}
				}
				if values["reservedPercent"] != 0 {
					utils.Set(activeSkill.SkillData, rn.reservedPercent, values["reservedPercent"])
					utils.Set(activeSkill.SkillData, rn.reservedBase, (utils.GetOr(activeSkill.SkillData, rn.reservedBase, float64(0)))+math.Ceil(env.Player.Output[name]*values["reservedPercent"]/100))
					env.Player.Output[rn.reservedPercentOut] = env.Player.Output[rn.reservedPercentOut] + values["reservedPercent"]
					if breakdown != nil {
						breakdown.Reservation(rn.reserved, BReservation{
							SkillName:      activeSkill.ActiveEffect.GrantedEffect.Name(),
							Base:           fmt.Sprint(values["basePercent"]) + "%",
							Mult:           utils.Ternary(mult != 1, utils.Ptr("x "+fmt.Sprint(mult)), nil),
							More:           utils.Ternary(values["more"] != 1, utils.Ptr("x "+fmt.Sprint(values["more"])), nil),
							Inc:            utils.Ternary(values["inc"] != 0, utils.Ptr("x "+fmt.Sprint(1+values["inc"]/100)), nil),
							Efficiency:     utils.Ternary(values["efficiency"] != 0, utils.Ptr("x "+fmt.Sprint(utils.RoundTo(100/(100+values["efficiency"]), 4))), nil),
							EfficiencyMore: utils.Ternary(values["efficiencyMore"] != 1, utils.Ptr("x "+fmt.Sprint(values["efficiencyMore"])), nil),
							Total:          fmt.Sprint(values["reservedPercent"]) + "%",
						})
					}
				}
				if skillModList.Flag(skillCfg, "HasUncancellableReservation") {
					env.Player.Output[rn.uncancellable] = env.Player.Output[rn.uncancellable] + values["reservedPercent"]
				}
			}
		}
	}

	// Set the life/mana reservations (hold off on GrantReserved<pool>AsAura)
	//pob:src/Modules/CalcPerform.lua:2039-2040
	doActorLifeManaReservation(env.Player, !env.ModDB.Flag(nil, "ManaIncreasedByOvercappedLightningRes"))
	if env.Minion != nil {
		doActorLifeManaReservation(env.Minion.Actor, false)
	}

	//pob:src/Modules/CalcPerform.lua:2042-2044
	if env.ModeCombat {
		mergeTinctures(env, tinctureEffectInc, tinctureEffectIncMagic, tinctureLimit)
	}

	// Process attribute requirements
	reqMult := calclib.Mod(env.ModDB, nil, "GlobalAttributeRequirements")
	attrTable := utils.Ternary(env.ModDB.Flag(nil, "OmniscienceRequirements"), []string{"Omni", "Str", "Dex", "Int"}, []string{"Str", "Dex", "Int"})
	for _, attr := range attrTable {
		breakdownAttr := attr
		if env.ModDB.Flag(nil, "OmniscienceRequirements") {
			breakdownAttr = "Omni"
		}
		if breakdown != nil {
			breakdown.AddCol("Req"+attr,
				BCol{Label: attr, Key: "req"},
				BCol{Label: "Source", Key: "source"},
				BCol{Label: "Source Name", Key: "sourceName"},
			)
		}
		out := float64(0)
		var outSource *RequirementsTable
		for _, reqSource := range env.RequirementsTable {
			attrVal := float64(utils.GetOr(reqSource, attr, 0))
			if attrVal > 0 {
				req := math.Floor(attrVal * reqMult)
				if env.ModDB.Flag(nil, "OmniscienceRequirements") {
					omniReqMult := 1 / (calclib.Mod(env.ModDB, nil, "OmniAttributeRequirements") - 1)
					attributereq := math.Floor(attrVal * reqMult)
					req = math.Floor(attributereq * omniReqMult)
				}
				//pob:block:src/Modules/CalcPerform.lua:2074-2077
				if req > out {
					outSource = reqSource
				}
				out = max(out, req)
				if breakdown != nil {
					row := map[string]string{
						"req":    utils.Ternary(req > env.Player.Output[breakdownAttr], "^#DD0022"+fmt.Sprint(req), fmt.Sprint(req)),
						"reqNum": fmt.Sprint(req),
						"source": reqSource.Source,
					}
					if reqSource.Source == "Item" {
						row["sourceName"] = fmt.Sprint(reqSource.SourceItem)
					} else if reqSource.Source == "Gem" {
						row["sourceName"] = fmt.Sprintf("%s %d/%d", reqSource.SourceGem.NameSpec, reqSource.SourceGem.Level, reqSource.SourceGem.Quality)
					}
					breakdown.AddRow("Req"+breakdownAttr, row)
				}
			}
		}
		if env.ModDB.Flag(nil, "IgnoreAttributeRequirements") {
			out = 0
		}
		env.Player.Output["Req"+attr+"String"] = 0
		if out > (env.Player.Output["Req"+breakdownAttr]) {
			env.Player.Output["Req"+breakdownAttr+"String"] = out
			env.Player.Output["Req"+breakdownAttr] = out
			//pob:block:src/Modules/CalcPerform.lua:2104
			env.Player.OutputStrings["Req"+breakdownAttr+"Item"] = requirementSourceName(outSource)
			/*
				TODO Breakdown
				if breakdown != nil {
					actor.Output["Req"+breakdownAttr+"String"] = out > (actor.Output[breakdownAttr] or 0) and colorCodes.NEGATIVE+out or out
				}
			*/
		}
	}
	/*
		TODO Breakdown
		if breakdown and breakdown["ReqOmni"] {
			table.sort(breakdown["ReqOmni"].rowList, function(a, b)
				if a.reqNum != b.reqNum {
					return a.reqNum > b.reqNum
				} else if a.source != b.source {
					return a.source < b.source
				} else {
					return a.sourceName < b.sourceName
				}
			end)
		}
	*/

	//pob:src/Modules/CalcPerform.lua:1804-1814
	if env.ModeBuffs {
		heraldList := make(map[string]bool)
		auraList := make(map[string]bool)
		for _, activeSkill := range env.Player.ActiveSkillList {
			switch {
			case activeSkill.SkillTypes[data.SkillTypeHerald] && !heraldList[activeSkill.SkillCfg.SkillName]:
				heraldList[activeSkill.SkillCfg.SkillName] = true
				env.ModDB.Multipliers["Herald"]++
				env.ModDB.Conditions["AffectedByHerald"] = true
			case activeSkill.SkillTypes[data.SkillTypeAura] &&
				!activeSkill.SkillTypes[data.SkillTypeAuraAffectsEnemies] &&
				!activeSkill.SkillData.AuraCannotAffectSelf &&
				!auraList[activeSkill.SkillCfg.SkillName]:
				auraList[activeSkill.SkillCfg.SkillName] = true
				env.ModDB.Multipliers["AuraAffectingSelf"]++
			}
		}
	}

	//pob:src/Modules/CalcPerform.lua:2128-2136
	if env.ModDB.Flag(nil, "ManaAppliesToShockEffect") {
		// Maximum Mana conversion from Lightning Mastery
		multiplier := utils.OrF(env.ModDB.Max(nil, "ImprovedManaAppliesToShockEffect"), 100) / 100
		for _, value := range env.ModDB.Tabulate(mod.TypeIncrease, nil, "Mana") {
			m := value.Mod
			modifiers := calclib.GetConvertedModTags(m, multiplier, false)
			env.ModDB.AddMod(mod.NewFloat("EnemyShockEffect", mod.TypeIncrease, math.Floor(m.Value().Float()*multiplier)).
				Source(m.GetSource()).Flag(m.Flags()).KeywordFlag(m.KeywordFlags()).Tag(modifiers...))
		}
	}

	//pob:src/Modules/CalcPerform.lua:2138-2139
	DoActorCharges(env, env.Player)

	buffs := make(map[string]*moddb.ModList)
	env.Buffs = buffs
	notBuff := make(map[string]bool)
	debuffs := make(map[string]*moddb.ModList)
	env.Debuffs = debuffs
	minionBuffs := make(map[string]*moddb.ModList)
	env.MinionBuffs = minionBuffs
	curses := make([]*curseEntry, 0)
	var linkSkills []string
	exports := newPartyBuffExports(env)
	minionCurses := make([]*curseEntry, 0)
	minionCurseLimit := 1
	//pob:block:src/Modules/CalcPerform.lua:1275
	partyMembers := env.partyMembers()
	//pob:block:src/Modules/CalcPerform.lua:2154
	allyBuffs := partyMembers.Aura
	hasActiveSpectreSkill := false
	activeSpectreTypes := make(map[string]bool)
	for _, activeSkill := range env.Player.ActiveSkillList {
		if activeSkill.SkillFlags[SkillFlagDisable] {
			continue
		}
		if activeSkill.ActiveEffect == nil || activeSkill.ActiveEffect.GrantedEffect == nil {
			continue
		}
		if !strings.HasPrefix(activeSkill.ActiveEffect.GrantedEffect.ID(), "RaiseSpectre") {
			continue
		}
		hasActiveSpectreSkill = true
		if activeSkill.Minion != nil && activeSkill.Minion.Type != "" {
			activeSpectreTypes[activeSkill.Minion.Type] = true
		}
	}
	if hasActiveSpectreSkill {
		for _, spectreID := range spectreList(env) {
			if !activeSpectreTypes[spectreID] {
				continue
			}
			spectreData := raw2.Minions[spectreID]
			if spectreData == nil {
				continue
			}
			if !env.ModDB.Conditions["HaveBeastSpectre"] {
				for _, tagName := range spectreData.MonsterTags {
					if tagName == "beast" {
						env.ModDB.Conditions["HaveBeastSpectre"] = true
						break
					}
				}
			}
			//pob:src/Modules/CalcPerform.lua:2189-2218
			for _, modData := range spectreData.ModList {
				name := modData.Name()
				if name == "EnemyCurseLimit" {
					minionCurseLimit = int(modData.Value().Float()) + 1
					break
				}
				if name != "AllyModifier" && name != "MinionModifier" && name != "PlayerModifier" {
					continue
				}
				if modData.Value().Type() != mod.ModValueMultiTypeList {
					continue
				}
				nested, ok := modData.Value().List().(mod.MinionModifier)
				if !ok || nested.Mod == nil {
					continue
				}
				copied := nested.Mod.Clone().Source(mod.Source("Spectre:" + spectreData.Name))
				if name == "AllyModifier" || name == "PlayerModifier" {
					if buffs["Spectre"] == nil {
						buffs["Spectre"] = moddb.NewModList()
					}
					buffs["Spectre"].AddMod(copied)
				}
				if name == "AllyModifier" || name == "MinionModifier" {
					if minionBuffs["Spectre"] == nil {
						minionBuffs["Spectre"] = moddb.NewModList()
					}
					minionBuffs["Spectre"].AddMod(nested.Mod.Clone().Source(mod.Source("Spectre:" + spectreData.Name)))
				}
			}
		}
	}
	//pob:block:src/Modules/CalcPerform.lua:2226-2252
	for _, activeSkill := range env.Player.ActiveSkillList {
		if activeSkill.SkillFlags[SkillFlagDisable] {
			continue
		}
		name := activeSkill.ActiveEffect.GrantedEffect.Name()
		key := strings.Join(strings.Fields(name), "")
		setStages := func(v float64) {
			activeSkill.SkillModList.AddMod(mod.NewFloat("Multiplier:"+key+"MaxStages", mod.TypeBase, v).Source("Base"))
			activeSkill.SkillModList.AddMod(mod.NewFloat("Multiplier:"+key+"StageAfterFirst", mod.TypeBase, v).Source("Base"))
		}
		switch {
		case (name == "Blight" || name == "Blight of Contagion" || name == "Blight of Atrophy") && activeSkill.SkillPart == 2:
			if env.triggerCache.building[cacheSkillUUID(env, activeSkill)] {
				continue
			}
			if cached := buildActiveSkillCache(env, activeSkill); cached != nil {
				baseMaxStages := activeSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "BlightBaseMaxStages")
				setStages(min(math.Floor(cached.Speed*cached.Duration)-1, baseMaxStages-1))
				continue
			}
			rate := (1 / activeSkill.ActiveEffect.GrantedEffect.CastTime()) * calclib.Mod(activeSkill.SkillModList, activeSkill.SkillCfg, "Speed") * CalcActionSpeedMod(env.Player)
			duration := calcSkillDuration(activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData, env, env.EnemyModDB)
			baseMaxStages := activeSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "BlightBaseMaxStages")
			setStages(min(math.Floor(rate*duration)-1, baseMaxStages-1))
		case name == "Penance Brand of Dissipation" && activeSkill.SkillPart == 2:
			if env.triggerCache.building[cacheSkillUUID(env, activeSkill)] {
				continue
			}
			if cached := buildActiveSkillCache(env, activeSkill); cached != nil {
				setStages(math.Max(min(math.Floor(cached.HitSpeed*cached.Duration)-1, 19), 0))
				continue
			}
			rate := 1 / (activeSkill.SkillData.RepeatFrequency / (1 + env.Player.MainSkill.SkillModList.Sum(mod.TypeIncrease, env.Player.MainSkill.SkillCfg, "Speed", "BrandActivationFrequency")/100) / activeSkill.SkillModList.More(activeSkill.SkillCfg, "BrandActivationFrequency"))
			duration := calcSkillDuration(activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData, env, env.EnemyModDB)
			setStages(math.Max(min(math.Floor(rate*duration)-1, 19), 0))
		case (name == "Scorching Ray" || name == "Scorching Ray of Immolation") && activeSkill.SkillPart == 2:
			setStages(7)
		case name == "Earthquake of Amplification" && activeSkill.SkillPart == 2:
			duration := calcSkillDuration(activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData, env, env.EnemyModDB)
			activeSkill.SkillModList.AddMod(mod.NewFloat("Multiplier:100msEarthquakeDuration", mod.TypeBase, math.Floor(duration*10)).Source("Skill:EarthquakeAltX"))
		}
	}

	guards := make(map[string]*moddb.ModList)
	guardOrder := make([]string, 0)
	affectedByAura := make(map[*Actor]bool)
	affectedByCurse := make(map[*Actor]bool)
	warcryList := make(map[string]bool)
	//pob:block:src/Modules/CalcPerform.lua:2254
	appliedCombustion := false
	for _, activeSkill := range env.Player.ActiveSkillList {
		skillModList := activeSkill.SkillModList
		skillCfg := activeSkill.SkillCfg
		for _, buff := range activeSkill.BuffList {
			if buff.Cond != "" {
				if cond, _ := skillModList.GetCondition(buff.Cond, skillCfg, false); !cond {
					continue
				}
			}
			if buff.EnemyCond != "" {
				if cond, _ := env.EnemyModDB.GetCondition(buff.EnemyCond, nil, false); !cond {
					continue
				}
			}

			buffNameKey := strings.ReplaceAll(buff.Name, " ", "")
			if buff.Type == "Buff" {
				if env.ModeBuffs && (!activeSkill.SkillFlags[SkillFlagTotem] || buff.AllowTotemBuff) {
					var cfg *moddb.ListCfg
					var modStore moddb.ModStoreFuncs = env.ModDB
					if buff.ActiveSkillBuff {
						cfg = skillCfg
						modStore = skillModList
					}
					if !buff.ApplyNotPlayer {
						activeSkill.BuffSkill = true
						env.ModDB.Conditions["AffectedBy"+buffNameKey] = true
						inc := modStore.Sum(mod.TypeIncrease, cfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer") +
							skillModList.Sum(mod.TypeIncrease, cfg, buffNameKey+"Effect")
						more := modStore.More(cfg, "BuffEffect", "BuffEffectOnSelf")
						srcList := moddb.NewModList()
						srcList.ScaleAddList(buff.ModList, (1+inc/100)*more)
						mergeBuff(srcList, buffs, buff.Name)
						mergeBuff(buff.UnscalableModList, buffs, buff.Name)
						//pob:src/Modules/CalcPerform.lua:2278-2280
						if activeSkill.SkillData.ThisIsNotABuff {
							notBuff[buff.Name] = true
						}
					}
					//pob:block:src/Modules/CalcPerform.lua:2282-2290
					if env.Minion != nil && (env.Minion.MinionData == nil || !env.Minion.MinionData.Hostile) &&
						(buff.ApplyMinions || buff.ApplyAllies || skillModList.Flag(nil, "BuffAppliesToAllies")) {
						activeSkill.MinionBuffSkill = true
						env.Minion.ModDB.Conditions["AffectedBy"+buffNameKey] = true
						inc := modStore.Sum(mod.TypeIncrease, cfg, "BuffEffect") +
							env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
						more := modStore.More(cfg, "BuffEffect") *
							env.Minion.ModDB.More(nil, "BuffEffectOnSelf")
						srcList := moddb.NewModList()
						srcList.ScaleAddList(buff.ModList, (1+inc/100)*more)
						mergeBuff(srcList, minionBuffs, buff.Name)
						mergeBuff(buff.UnscalableModList, minionBuffs, buff.Name)
					}
					//pob:block:src/Modules/CalcPerform.lua:2291-2296
					if exports != nil && (buff.ApplyAllies || skillModList.Flag(nil, "BuffAppliesToAllies") || skillModList.Flag(nil, "BuffAppliesToPartyMembers")) {
						inc := modStore.Sum(mod.TypeIncrease, skillCfg, "BuffEffect") + skillModList.Sum(mod.TypeIncrease, skillCfg, buffNameKey+"Effect")
						more := modStore.More(skillCfg, "BuffEffect")
						exports.otherEffects[buff.Name] = &partyBuffExport{effectMult: (1 + inc/100) * more, modList: joinModLists(buff.ModList, buff.UnscalableModList)}
					}
				}
			} else if buff.Type == "Guard" {
				if env.ModeBuffs && (!activeSkill.SkillFlags[SkillFlagTotem] || buff.AllowTotemBuff) && !buff.ApplyNotPlayer {
					var cfg *moddb.ListCfg
					var modStore moddb.ModStoreFuncs = env.ModDB
					if buff.ActiveSkillBuff {
						cfg = skillCfg
						modStore = skillModList
					}
					activeSkill.BuffSkill = true
					inc := modStore.Sum(mod.TypeIncrease, cfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer")
					more := modStore.More(cfg, "BuffEffect", "BuffEffectOnSelf")
					srcList := moddb.NewModList()
					srcList.ScaleAddList(buff.ModList, (1+inc/100)*more)
					guardOrder = mergeFlaskBuff(srcList, guards, guardOrder, buff.Name)
					guardOrder = mergeFlaskBuff(buff.UnscalableModList, guards, guardOrder, buff.Name)
				}
			} else if buff.Type == "Aura" {
				if env.ModeBuffs {
					//pob:src/Modules/CalcPerform.lua:2394-2408
					extraAuraModList := moddb.NewModList()
					for _, value := range env.ModDB.List(skillCfg, "ExtraAuraEffect") {
						extra, ok := value.(mod.ExtraAuraEffect)
						if !ok {
							continue
						}
						add := true
						for _, existing := range extraAuraModList.Mods() {
							if moddb.CompareModParams(existing, extra.Mod) {
								existing.Value().SetFloat(existing.Value().Float() + extra.Mod.Value().Float())
								add = false
								break
							}
						}
						if add {
							extraAuraModList.AddMod(extra.Mod.Clone())
						}
					}
					if !activeSkill.SkillData.AuraCannotAffectSelf {
						inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
						more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
						mult := (1 + inc/100) * more
						//pob:block:src/Modules/CalcPerform.lua:2413
						if env.ModDB.Flag(nil, "AlliesAurasCannotAffectSelf") || !allyBuffs.strongerThan("Aura", buff.Name, mult) {
							activeSkill.BuffSkill = true
							affectedByAura[env.Player] = true
							//pob:block:src/Modules/CalcPerform.lua:1933-1935
							if strings.HasPrefix(buff.Name, "Vaal") {
								base := ""
								if len(buff.Name) > 5 {
									base = strings.ReplaceAll(buff.Name[5:], " ", "")
								}
								env.ModDB.Conditions["AffectedBy"+base] = true
							}
							env.ModDB.Conditions["AffectedBy"+buffNameKey] = true
							srcList := moddb.NewModList()
							srcList.ScaleAddList(buff.ModList, mult)
							//pob:src/Modules/CalcPerform.lua:2143
							srcList.ScaleAddList(extraAuraModList, mult)
							mergeBuff(srcList, buffs, buff.Name)
							mergeBuff(buff.UnscalableModList, buffs, buff.Name)
						}
					}
					//pob:src/Modules/CalcPerform.lua:2426-2430
					if env.Minion != nil && !(env.ModDB.Flag(nil, "SelfAurasCannotAffectAllies") ||
						env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") ||
						env.ModDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies")) {
						inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect") +
							env.Minion.ModDB.Sum(mod.TypeIncrease, skillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
						more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect") *
							env.Minion.ModDB.More(skillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
						mult := (1 + inc/100) * more
						//pob:block:src/Modules/CalcPerform.lua:2431
						if !allyBuffs.strongerThan("Aura", buff.Name, mult) {
							activeSkill.MinionBuffSkill = true
							affectedByAura[env.Minion.Actor] = true
							env.Minion.ModDB.Conditions["AffectedBy"+buffNameKey] = true
							srcList := moddb.NewModList()
							srcList.ScaleAddList(buff.ModList, mult)
							srcList.ScaleAddList(extraAuraModList, mult)
							mergeBuff(srcList, minionBuffs, buff.Name)
							mergeBuff(buff.UnscalableModList, minionBuffs, buff.Name)
						}
					}
					//pob:block:src/Modules/CalcPerform.lua:2441-2450
					if exports != nil && !(env.ModDB.Flag(nil, "SelfAurasCannotAffectAllies") ||
						env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") ||
						env.ModDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies")) {
						inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect")
						more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect")
						exports.setAura(buff.Name, (1+inc/100)*more, joinModLists(buff.ModList, buff.UnscalableModList, extraAuraModList))
					}
					//pob:block:src/Modules/CalcPerform.lua:2451-2464
					if !(env.ModDB.Flag(nil, "SelfAurasCannotAffectAllies") ||
						env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") ||
						env.ModDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies")) &&
						env.ModDB.Flag(nil, "AurasAffectEnemies") &&
						!skillModList.Flag(skillCfg, "SelfAurasAffectYouAndLinkedTarget") {
						inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect")
						more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect")
						mult := (1 + inc/100) * more
						//pob:src/Modules/CalcPerform.lua:2460
						if exports != nil {
							exports.setAuraDebuff(buff.Name, mult, joinModLists(buff.ModList, buff.UnscalableModList, extraAuraModList))
						}
						srcList := moddb.NewModList()
						srcList.ScaleAddList(buff.ModList, mult)
						srcList.ScaleAddList(extraAuraModList, mult)
						mergeBuff(srcList, debuffs, buff.Name)
						mergeBuff(buff.UnscalableModList, debuffs, buff.Name)
					}
					//pob:block:src/Modules/CalcPerform.lua:2466-2495
					if env.Player.MainSkill.SkillFlags[SkillFlagTotem] &&
						!(env.ModDB.Flag(nil, "SelfAurasCannotAffectAllies") ||
							env.ModDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies")) {
						moddb.SetCondition(env.Player.MainSkill.SkillModList.ModStore, "AffectedBy"+buffNameKey, true)
						moddb.SetCondition(env.Player.MainSkill.SkillModList.ModStore, "AffectedByAura", true)

						inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
						more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
						scale := math.Max((1+inc/100)*more, 0)

						renamed := moddb.NewModList()
						for _, list := range []*moddb.ModList{extraAuraModList, buff.ModList, buff.UnscalableModList} {
							if list == nil {
								continue
							}
							for _, m := range list.Mods() {
								name := m.Name()
								if name != "EnergyShield" && name != "Armour" && name != "Evasion" &&
									!totemAuraResistRe.MatchString(name) {
									continue
								}
								renamed.AddMod(m.Clone().SetName("Totem" + name))
							}
						}
						srcList := moddb.NewModList()
						srcList.ScaleAddList(renamed, scale)
						mergeBuff(srcList, buffs, "Totem "+buff.Name)
					}
				}
			} else if buff.Type == "Warcry" {
				if env.ModeBuffs {
					warcryKey := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(buff.Name, " Cry", ""), "'s", ""), " ", "")

					baseExerts := activeSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, warcryKey+"ExertedAttacks")
					if baseExerts > 0 {
						extraExertions := activeSkill.SkillModList.Sum(mod.TypeBase, nil, "ExtraExertedAttacks")
						exertMultiplier := activeSkill.SkillModList.More(nil, "ExtraExertedAttacks")
						env.Player.ModDB.AddMod(mod.NewFloat("Num"+warcryKey+"Exerts", mod.TypeBase, math.Floor((baseExerts+extraExertions)*exertMultiplier)))
						if !warcryList[buff.Name] {
							env.Player.ModDB.AddMod(mod.NewFloat("Multiplier:ExertingWarcryCount", mod.TypeBase, 1).Source(mod.Source(buff.Name)))
							warcryList[buff.Name] = true
						}
					}

					//pob:src/Modules/CalcPerform.lua:2326
					if !activeSkill.SkillModList.Flag(nil, "CannotShareWarcryBuffs") {
						warcryPower := utils.Or(
							env.ModDB.Override(nil, "WarcryPower"),
							math.Max(
								env.ModDB.Sum(mod.TypeBase, nil, "WarcryPower")*(1+env.ModDB.Sum(mod.TypeIncrease, nil, "WarcryPower")/100),
								env.ModDB.Sum(mod.TypeBase, nil, "MinimumWarcryPower"),
							),
						)

						fullDuration := calcSkillDuration(activeSkill.SkillModList, skillCfg, activeSkill.SkillData, env, env.EnemyModDB)
						actualCooldown := utils.Or(
							activeSkill.SkillModList.Override(skillCfg, "CooldownRecovery"),
							(activeSkill.SkillData.Cooldown+activeSkill.SkillModList.Sum(mod.TypeBase, skillCfg, "CooldownRecovery"))/calclib.Mod(activeSkill.SkillModList, skillCfg, "CooldownRecovery"),
						)
						uptime := min(fullDuration/actualCooldown, 1)
						if env.ModDB.Flag(nil, "Condition:WarcryMaxHit") {
							uptime = 1
						}

						//pob:src/Modules/CalcPerform.lua:2338-2351
						if !env.ModDB.Flag(nil, "CannotGainWarcryBuffs") && !buff.ApplyNotPlayer {
							activeSkill.BuffSkill = true
							env.ModDB.Conditions["AffectedBy"+warcryKey] = true
							inc := activeSkill.SkillModList.Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer")
							more := activeSkill.SkillModList.More(skillCfg, "BuffEffect", "BuffEffectOnSelf")
							srcList := moddb.NewModList()
							for _, warcryBuff := range buff.ModList.Mods() {
								srcList.ScaleAddList(singleModList(warcryBuff), (1+inc/100)*more*warcryPowerBonus(warcryBuff, warcryPower)*uptime)
							}
							mergeBuff(srcList, buffs, buff.Name)
							mergeBuff(buff.UnscalableModList, buffs, buff.Name)
						}

						//pob:src/Modules/CalcPerform.lua:2352-2361
						if env.Minion != nil {
							activeSkill.MinionBuffSkill = true
							env.Minion.ModDB.Conditions["AffectedBy"+warcryKey] = true
							inc := activeSkill.SkillModList.Sum(mod.TypeIncrease, skillCfg, "BuffEffect") +
								env.Minion.ModDB.Sum(mod.TypeIncrease, skillCfg, "BuffEffectOnSelf")
							more := activeSkill.SkillModList.More(skillCfg, "BuffEffect") *
								env.Minion.ModDB.More(skillCfg, "BuffEffectOnSelf")
							srcList := moddb.NewModList()
							for _, warcryBuff := range buff.ModList.Mods() {
								srcList.ScaleAddList(singleModList(warcryBuff), (1+inc/100)*more*warcryPowerBonus(warcryBuff, warcryPower)*uptime)
							}
							if warcryDisplayName(activeSkill) == "Rallying Cry" {
								powerBonus := math.Floor(math.Min(warcryPower, 30) / 5)
								rallyingWeaponEffect := math.Floor(activeSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "RallyingCryAllyDamageBonusPer5Power") * powerBonus)
								rallyingBonusMoreMultiplier := 1 + activeSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "RallyingCryMinionDamageBonusMultiplier")
								extraWarcryModList := moddb.NewModList()
								for _, damageType := range rallyingCryDamageTypes {
									minBase, maxBase := weaponDataDamage(env.Player.WeaponData1, damageType)
									if minBase != 0 {
										extraWarcryModList.AddMod(mod.NewFloat(damageType+"Min", mod.TypeBase, minBase*rallyingWeaponEffect/100).Source("Rallying Cry").KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Warcry").Div(5).Limit(30)))
									}
									if maxBase != 0 {
										extraWarcryModList.AddMod(mod.NewFloat(damageType+"Max", mod.TypeBase, maxBase*rallyingWeaponEffect/100).Source("Rallying Cry").KeywordFlag(mod.KeywordFlagAttack).Tag(mod.GlobalEffect("Warcry").Div(5).Limit(30)))
									}
								}
								srcList.ScaleAddList(extraWarcryModList, (1+inc/100)*rallyingBonusMoreMultiplier*uptime)
							}
							mergeBuff(srcList, minionBuffs, buff.Name)
						}
						//pob:block:src/Modules/CalcPerform.lua:2382-2389
						if exports != nil {
							inc := activeSkill.SkillModList.Sum(mod.TypeIncrease, skillCfg, "BuffEffect")
							more := activeSkill.SkillModList.More(skillCfg, "BuffEffect")
							mods := joinModLists(buff.ModList, buff.UnscalableModList)
							exports.warcry[buff.Name] = &partyBuffExport{effectMult: (1 + inc/100) * more * uptime, modList: mods, warcryPowerBonus: warcryPowerBonuses(mods, warcryPower)}
						}
					}
				}
			} else if buff.Type == "Debuff" || buff.Type == "AuraDebuff" {
				var stackCount float64
				if buff.StackVar != "" {
					stackCount = skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.StackVar)
					if buff.StackLimit > 0 {
						stackCount = math.Min(stackCount, float64(buff.StackLimit))
					} else if buff.StackLimitVar != "" {
						stackCount = math.Min(stackCount, skillModList.Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.StackLimitVar))
					}
				} else {
					if activeSkill.SkillData.StackCount != nil {
						stackCount = *activeSkill.SkillData.StackCount
					} else {
						stackCount = 1
					}
				}

				if env.ModeEffective && stackCount > 0 {
					activeSkill.DebuffSkill = true
					env.EnemyModDB.Conditions["AffectedBy"+buffNameKey] = true
					env.ModDB.Conditions["AffectedBy"+buffNameKey] = true
					mult := float64(1)
					if buff.Type == "AuraDebuff" {
						mult = 0
						if !env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") {
							inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
							more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
							mult = (1 + inc/100) * more
							//pob:src/Modules/CalcPerform.lua:2583
							if exports != nil {
								exports.setAuraDebuff(buff.Name, mult, joinModLists(buff.ModList, buff.UnscalableModList))
							}
							//pob:block:src/Modules/CalcPerform.lua:2584-2586
							if allyBuffs.strongerThan("AuraDebuff", buff.Name, mult) {
								mult = 0
							}
						}
					}
					if buff.Type == "Debuff" {
						inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "DebuffEffect")
						more := skillModList.More(skillCfg, "DebuffEffect")
						mult = (1 + inc/100) * more
					}
					srcList := moddb.NewModList()
					srcList.ScaleAddList(buff.ModList, mult*stackCount)
					if activeSkill.SkillData.StackCount != nil || buff.StackVar != "" {
						srcList.NewMod("Multiplier:"+buff.Name+"Stack", mod.TypeBase, stackCount, mod.Source(buff.Name))
					}
					mergeBuff(srcList, debuffs, buff.Name)
				}
			} else if buff.Type == "Curse" || buff.Type == "CurseBuff" {
				mark := activeSkill.SkillTypes[data.SkillTypeMark]
				env.ModDB.Conditions["SelfCast"+buffNameKey] =
					!(activeSkill.SkillTypes[data.SkillTypeTriggered] || activeSkill.SkillTypes[data.SkillTypeAura])
				skipCurse := !mark && balanceOfTerrorSelfOnly(env, buffNameKey)
				if !skipCurse && (env.ModeEffective && (!env.EnemyModDB.Flag(nil, "Hexproof") || env.ModDB.Flag(nil, "CursesIgnoreHexproof")) || mark) {
					curse := &curseEntry{
						name:                   buff.Name,
						fromPlayer:             true,
						priority:               determineCursePriority(buff.Name, activeSkill),
						isMark:                 mark,
						ignoreHexLimit:         env.ModDB.Flag(activeSkill.SkillCfg, "CursesIgnoreHexLimit") && !mark,
						socketedCursesHexLimit: env.ModDB.Flag(activeSkill.SkillCfg, "SocketedCursesAdditionalLimit"),
					}
					inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "CurseEffect") + env.EnemyModDB.Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf")
					//pob:src/Modules/CalcPerform.lua:2620
					if activeSkill.SkillTypes[data.SkillTypeAura] && !activeSkill.SkillTypes[data.SkillTypeRemoteMined] {
						inc += skillModList.Sum(mod.TypeIncrease, skillCfg, "AuraEffect")
					}
					more := skillModList.More(skillCfg, "CurseEffect")
					moreMark := more
					// This is non-ideal, but the only More for enemy is the boss effect
					if !curse.isMark {
						more *= env.EnemyModDB.More(nil, "CurseEffectOnSelf")
					}
					mult := 0.0
					//pob:src/Modules/CalcPerform.lua:2630
					if !((env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") ||
						skillModList.Flag(skillCfg, "SelfAurasAffectYouAndLinkedTarget")) &&
						activeSkill.SkillTypes[data.SkillTypeAura]) {
						mult = (1 + inc/100) * more
					}
					if buff.Type == "Curse" {
						curse.modList = moddb.NewModList()
						curse.modList.ScaleAddList(buff.ModList, mult)
						//pob:block:src/Modules/CalcPerform.lua:2636-2638
						if exports != nil {
							effectMult := (1 + inc/100) * moreMark
							if curse.isMark {
								effectMult = mult
							}
							exports.curse[buff.Name] = &partyBuffExport{isMark: curse.isMark, effectMult: effectMult, modList: joinModLists(buff.ModList, buff.UnscalableModList)}
						}
					} else {
						// Curse applies a buff; scale by curse effect, then buff effect.
						temp := moddb.NewModList()
						temp.ScaleAddList(buff.ModList, mult)
						curse.buffModList = moddb.NewModList()
						buffInc := env.ModDB.Sum(mod.TypeIncrease, skillCfg, "BuffEffectOnSelf")
						buffMore := env.ModDB.More(skillCfg, "BuffEffectOnSelf")
						curse.buffModList.ScaleAddList(temp, (1+buffInc/100)*buffMore)
						//pob:src/Modules/CalcPerform.lua:2647-2652
						if env.Minion != nil {
							curse.minionBuffModList = moddb.NewModList()
							minionBuffInc := env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
							minionBuffMore := env.Minion.ModDB.More(nil, "BuffEffectOnSelf")
							curse.minionBuffModList.ScaleAddList(temp, (1+minionBuffInc/100)*minionBuffMore)
						}
					}
					curses = append(curses, curse)
				}
			} else if buff.Type == "Link" {
				//pob:block:src/Modules/CalcPerform.lua:2656-2702
				summonUndamageable := false
				if env.Minion != nil && env.Minion.MainSkill != nil && env.Minion.MainSkill.SummonSkill != nil {
					summonUndamageable = env.Minion.MainSkill.SummonSkill.SkillTypes[data.SkillTypeMinionsAreUndamagable]
				}
				minionResolvable := env.Minion != nil && env.Minion.MainSkill != nil && env.Minion.MainSkill.SummonSkill != nil
				linksApplyToMinions := minionResolvable &&
					env.ModDB.Flag(nil, "Condition:CanLinkToMinions") &&
					env.ModDB.Flag(nil, "Condition:LinkedToMinion") &&
					!env.Minion.ModDB.Flag(nil, "Condition:CannotBeDamaged") &&
					!summonUndamageable

				if env.ModeBuffs && len(linkSkills) < 1 && (exports != nil || linksApplyToMinions) {
					//pob:block:src/Modules/CalcPerform.lua:2660-2678
					extraLinkModList := moddb.NewModList()
					for _, value := range utils.CastSlice[mod.MinionModifier](env.ModDB.List(skillCfg, "ExtraLinkEffect")) {
						add := true
						for _, existing := range extraLinkModList.Mods() {
							if moddb.CompareModParams(existing, value.Mod) {
								ev, sv := existing.Value(), value.Mod.Value()
								if ev != nil && sv != nil && ev.Type() == mod.ModValueMultiTypeFloat && sv.Type() == mod.ModValueMultiTypeFloat {
									ev.SetFloat(ev.Float() + sv.Float())
								}
								add = false
								break
							}
						}
						if add {
							extraLinkModList.AddMod(value.Mod.Clone())
							// special handling to add this early
							if value.Mod.Name() == "ParentNonUniqueFlasksAppliedToYou" {
								flaskMinion.nonUniqueApply = true
							}
						}
					}
					inc := skillModList.Sum(mod.TypeIncrease, skillCfg, "LinkEffect", "BuffEffect")
					more := skillModList.More(skillCfg, "LinkEffect", "BuffEffect")
					//pob:block:src/Modules/CalcPerform.lua:2682-2687
					if exports != nil {
						exports.link[buff.Name] = &partyBuffExport{effectMult: (1 + inc/100) * more, modList: joinModLists(buff.ModList, buff.UnscalableModList, extraLinkModList)}
					}

					if linksApplyToMinions {
						activeSkill.MinionBuffSkill = true
						env.Minion.ModDB.Conditions["AffectedBy"+buffNameKey] = true
						env.Minion.ModDB.Conditions["AffectedByLink"] = true
						inc += env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "LinkEffectOnSelf")
						more *= env.Minion.ModDB.More(nil, "BuffEffectOnSelf", "LinkEffectOnSelf")
						mult := (1 + inc/100) * more
						srcList := moddb.NewModList()
						srcList.ScaleAddList(buff.ModList, mult)
						srcList.ScaleAddList(extraLinkModList, mult)
						mergeBuff(srcList, minionBuffs, buff.Name)
						linkSkills = append(linkSkills, buff.Name)
					}
				}
			}
		}

		//pob:block:src/Modules/CalcPerform.lua:2704-2706
		if activeSkill.SkillModList.Flag(nil, "Condition:CanWither") ||
			(activeSkill.Minion != nil && env.Minion != nil && env.Minion.ModDB != nil &&
				env.Minion.ModDB.Flag(nil, "Condition:CanWither")) {
			witherInc := "WitherEffect"
			if activeSkill.Minion != nil {
				witherInc = "MinionWitherEffect"
			}
			effect := math.Floor(6 * (1 + env.ModDB.Sum(mod.TypeIncrease, nil, witherInc)/100))
			env.ModDB.AddMod(mod.NewFloat("WitherEffectStack", mod.TypeMAX, effect))
		}
		//pob:block:src/Modules/CalcPerform.lua:2708-2720
		if env.EnemyModDB.Flag(nil, "Condition:Ignited") && (activeSkill.SkillTypes[data.SkillTypeDamage] || activeSkill.SkillTypes[data.SkillTypeAttack]) && !appliedCombustion {
			for _, support := range activeSkill.SupportList {
				if support.GrantedEffect != nil && support.GrantedEffect.Name() == "Combustion" {
					if !activeSkill.SkillModList.Flag(activeSkill.SkillCfg, "CannotIgnite") {
						value := activeSkill.SkillModList.Sum(mod.TypeBase, activeSkill.SkillCfg, "CombustionFireResist")
						env.EnemyModDB.AddMod(mod.NewFloat("FireResist", mod.TypeBase, value).Source("Combustion").
							Tag(mod.GlobalEffect("Debuff").Name("Combustion"), mod.Condition("Ignited")))
						appliedCombustion = true
					}
					break
				}
			}
		}

		//pob:src/Modules/CalcPerform.lua:1945-2029
		if activeSkill.Minion != nil && activeSkill.Minion.ActiveSkillList != nil {
			castingMinion := activeSkill.Minion
			//pob:block:src/Modules/CalcPerform.lua:2724-2736
			setSpectreSource := func(modList *moddb.ModList, sourceSkill string) {
				if !activeSkill.SkillFlags[SkillFlagSpectre] {
					return
				}
				source := "Spectre:"
				if sourceSkill != "" {
					source += sourceSkill + " - " + castingMinion.MinionData.Name
				} else {
					source += castingMinion.MinionData.Name
				}
				mods := modList.Mods()
				for i, m := range mods {
					mods[i] = m.Clone().Source(mod.Source(source))
				}
			}
			for _, minionSkill := range castingMinion.ActiveSkillList {
				minionSkillModList := minionSkill.SkillModList
				minionSkillCfg := minionSkill.SkillCfg
				for _, buff := range minionSkill.BuffList {
					minionBuffNameKey := strings.ReplaceAll(buff.Name, " ", "")
					if buff.Type == "Buff" {
						//pob:src/Modules/CalcPerform.lua:1951-1973
						if env.ModeBuffs && minionSkill.SkillData.Enable {
							var cfg *moddb.ListCfg
							var modStore moddb.ModStoreFuncs = castingMinion.ModDB
							if buff.ActiveSkillBuff {
								cfg = minionSkillCfg
								modStore = minionSkillModList
							}
							if buff.ApplyAllies {
								minionSkill.BuffSkill = true
								env.ModDB.Conditions["AffectedBy"+minionBuffNameKey] = true
								inc := modStore.Sum(mod.TypeIncrease, cfg, "BuffEffect", "BuffEffectOnPlayer") +
									env.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
								more := modStore.More(cfg, "BuffEffect", "BuffEffectOnPlayer") * env.ModDB.More(nil, "BuffEffectOnSelf")
								srcList := moddb.NewModList()
								srcList.ScaleAddList(buff.ModList, (1+inc/100)*more)
								mergeBuff(srcList, buffs, buff.Name)
								//pob:block:src/Modules/CalcPerform.lua:2752
								mergeBuff(buff.ModList, buffs, buff.Name)
								mergeBuff(buff.UnscalableModList, buffs, buff.Name)
								//pob:block:src/Modules/CalcPerform.lua:2753-2761
								if minionSkill.SkillData.ThisIsNotABuff {
									notBuff[buff.Name] = true
								}
								if exports != nil {
									inc := modStore.Sum(mod.TypeIncrease, cfg, "BuffEffect")
									more := modStore.More(cfg, "BuffEffect")
									exports.otherEffects[buff.Name] = &partyBuffExport{effectMult: (1 + inc/100) * more, modList: joinModLists(buff.ModList, buff.UnscalableModList)}
								}
							}
							envMinionCheck := env.Minion != nil && (env.Minion == castingMinion || buff.ApplyAllies)
							if buff.ApplyMinions || envMinionCheck {
								minionSkill.MinionBuffSkill = true
								if envMinionCheck {
									env.Minion.ModDB.Conditions["AffectedBy"+minionBuffNameKey] = true
								} else {
									castingMinion.ModDB.Conditions["AffectedBy"+minionBuffNameKey] = true
								}
								names := []string{"BuffEffect"}
								if env.Minion == castingMinion {
									names = append(names, "BuffEffectOnSelf")
								}
								inc := modStore.Sum(mod.TypeIncrease, cfg, names...)
								more := modStore.More(cfg, names...)
								srcList := moddb.NewModList()
								srcList.ScaleAddList(buff.ModList, (1+inc/100)*more)
								mergeBuff(srcList, minionBuffs, buff.Name)
								//pob:block:src/Modules/CalcPerform.lua:2776
								mergeBuff(buff.ModList, minionBuffs, buff.Name)
								mergeBuff(buff.UnscalableModList, minionBuffs, buff.Name)
								//pob:block:src/Modules/CalcPerform.lua:2777-2779
								if minionSkill.SkillData.ThisIsNotABuff {
									notBuff[buff.Name] = true
								}
							}
						}
					} else if buff.Type == "Aura" {
						//pob:src/Modules/CalcPerform.lua:2102-2119
						//pob:block:src/Modules/CalcPerform.lua:2782-2876
						if env.ModeBuffs && minionSkill.SkillData.Enable {
							extraAuraModList := moddb.NewModList()
							for _, value := range castingMinion.ModDB.List(minionSkillCfg, "ExtraAuraEffect") {
								extra, ok := value.(mod.ExtraAuraEffect)
								if !ok {
									continue
								}
								add := true
								for _, existing := range extraAuraModList.Mods() {
									if moddb.CompareModParams(existing, extra.Mod) {
										existing.Value().SetFloat(existing.Value().Float() + extra.Mod.Value().Float())
										add = false
										break
									}
								}
								if add {
									extraAuraModList.AddMod(extra.Mod.Clone())
								}
							}
							if !(castingMinion.ModDB.Flag(nil, "SelfAurasCannotAffectAllies") ||
								castingMinion.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") ||
								castingMinion.ModDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies") ||
								minionSkillModList.Flag(minionSkillCfg, "SelfAurasAffectYouAndLinkedTarget")) {
								if !env.ModDB.Flag(nil, "AlliesAurasCannotAffectSelf") && !env.ModDB.Conditions["AffectedBy"+minionBuffNameKey] {
									inc := minionSkillModList.Sum(mod.TypeIncrease, minionSkillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnPlayer", "AuraBuffEffect") +
										env.ModDB.Sum(mod.TypeIncrease, minionSkillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect") *
										env.ModDB.More(minionSkillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
									mult := (1 + inc/100) * more
									//pob:block:src/Modules/CalcPerform.lua:2804
									if !allyBuffs.strongerThan("Aura", buff.Name, mult) {
										minionSkill.BuffSkill = true
										env.ModDB.Conditions["AffectedByAura"] = true
										if strings.HasPrefix(buff.Name, "Vaal") && len(buff.Name) > 5 {
											env.ModDB.Conditions["AffectedBy"+strings.ReplaceAll(buff.Name[5:], " ", "")] = true
										}
										env.ModDB.Conditions["AffectedBy"+minionBuffNameKey] = true
										srcList := moddb.NewModList()
										srcList.ScaleAddList(buff.ModList, mult)
										srcList.ScaleAddList(extraAuraModList, mult)
										setSpectreSource(srcList, buff.Name)
										mergeBuff(srcList, buffs, buff.Name)
									}
								}
								if env.Minion != nil && !env.Minion.ModDB.Conditions["AffectedBy"+minionBuffNameKey] &&
									(env.Minion != castingMinion || !activeSkill.SkillData.AuraCannotAffectSelf) {
									inc := minionSkillModList.Sum(mod.TypeIncrease, minionSkillCfg, "AuraEffect", "BuffEffect") +
										env.Minion.ModDB.Sum(mod.TypeIncrease, minionSkillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect") *
										env.Minion.ModDB.More(minionSkillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
									mult := (1 + inc/100) * more
									//pob:block:src/Modules/CalcPerform.lua:2822
									if !allyBuffs.strongerThan("Aura", buff.Name, mult) {
										minionSkill.MinionBuffSkill = true
										env.Minion.ModDB.Conditions["AffectedBy"+minionBuffNameKey] = true
										env.Minion.ModDB.Conditions["AffectedByAura"] = true
										srcList := moddb.NewModList()
										srcList.ScaleAddList(buff.ModList, mult)
										srcList.ScaleAddList(extraAuraModList, mult)
										setSpectreSource(srcList, buff.Name)
										mergeBuff(srcList, minionBuffs, buff.Name)
									}
								}
								//pob:block:src/Modules/CalcPerform.lua:2833-2843
								if exports != nil {
									inc := minionSkillModList.Sum(mod.TypeIncrease, minionSkillCfg, "AuraEffect", "BuffEffect")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect")
									newModList := moddb.NewModList()
									newModList.AddList(buff.ModList)
									newModList.AddList(buff.UnscalableModList)
									newModList.AddList(extraAuraModList)
									setSpectreSource(newModList, buff.Name)
									exports.setAura(buff.Name, (1+inc/100)*more, newModList.Mods())
								}
								if env.Player.MainSkill.SkillFlags[SkillFlagTotem] &&
									!env.Player.MainSkill.SkillModList.Conditions["AffectedBy"+minionBuffNameKey] {
									moddb.SetCondition(env.Player.MainSkill.SkillModList.ModStore, "AffectedBy"+minionBuffNameKey, true)
									moddb.SetCondition(env.Player.MainSkill.SkillModList.ModStore, "AffectedByAura", true)
									inc := minionSkillModList.Sum(mod.TypeIncrease, minionSkillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
									scale := math.Max((1+inc/100)*more, 0)
									renamed := moddb.NewModList()
									for _, list := range []*moddb.ModList{extraAuraModList, buff.ModList, buff.UnscalableModList} {
										if list == nil {
											continue
										}
										for _, m := range list.Mods() {
											name := m.Name()
											if name != "EnergyShield" && name != "Armour" && name != "Evasion" &&
												!totemAuraResistRe.MatchString(name) {
												continue
											}
											renamed.AddMod(m.Clone().SetName("Totem" + name))
										}
									}
									srcList := moddb.NewModList()
									srcList.ScaleAddList(renamed, scale)
									setSpectreSource(srcList, "")
									mergeBuff(srcList, buffs, "Totem "+buff.Name)
								}
							}
						}
					} else if buff.Type == "Curse" {
						//pob:src/Modules/CalcPerform.lua:2120-2132
						if env.ModeEffective && minionSkill.SkillData.Enable &&
							(!env.EnemyModDB.Flag(nil, "Hexproof") || minionSkill.SkillTypes[data.SkillTypeMark]) {
							curse := &curseEntry{
								name:     buff.Name,
								priority: determineCursePriority(buff.Name, minionSkill),
							}
							inc := minionSkillModList.Sum(mod.TypeIncrease, minionSkillCfg, "CurseEffect") +
								env.EnemyModDB.Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf")
							more := minionSkillModList.More(minionSkillCfg, "CurseEffect") *
								env.EnemyModDB.More(nil, "CurseEffectOnSelf")
							curse.modList = moddb.NewModList()
							curse.modList.ScaleAddList(buff.ModList, (1+inc/100)*more)
							minionCurses = append(minionCurses, curse)
						}
					} else if buff.Type == "Debuff" || buff.Type == "AuraDebuff" {
						//pob:block:src/Modules/CalcPerform.lua:2889-2935
						var stackCount float64
						if buff.StackVar != "" {
							stackCount = minionSkillModList.Sum(mod.TypeBase, minionSkillCfg, "Multiplier:"+buff.StackVar)
							if buff.StackLimit > 0 {
								stackCount = math.Min(stackCount, float64(buff.StackLimit))
							} else if buff.StackLimitVar != "" {
								stackCount = math.Min(stackCount, minionSkillModList.Sum(mod.TypeBase, minionSkillCfg, "Multiplier:"+buff.StackLimitVar))
							}
						} else {
							if minionSkill.SkillData.StackCount != nil {
								stackCount = *minionSkill.SkillData.StackCount
							} else {
								stackCount = 1
							}
						}
						if env.ModeEffective && stackCount > 0 {
							minionSkill.DebuffSkill = true
							srcList := moddb.NewModList()
							mult := 1.0
							if buff.Type == "AuraDebuff" {
								mult = 0
								if !minionSkillModList.Flag(nil, "SelfAurasOnlyAffectYou") || minionSkillModList.Flag(minionSkillCfg, "SelfAurasAffectYouAndLinkedTarget") {
									inc := minionSkillModList.Sum(mod.TypeIncrease, minionSkillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
									mult = (1 + inc/100) * more
									if env.EnemyModDB.Conditions["AffectedBy"+minionBuffNameKey] {
										mult = 0
									} else if exports != nil {
										//pob:block:src/Modules/CalcPerform.lua:2912
										exports.setAuraDebuff(buff.Name, mult, joinModLists(buff.ModList, buff.UnscalableModList))
									}
									if mult != 0 && allyBuffs.strongerThan("AuraDebuff", buff.Name, mult) {
										//pob:block:src/Modules/CalcPerform.lua:2913-2915
										mult = 0
									}
								}
							}
							env.EnemyModDB.Conditions["AffectedBy"+minionBuffNameKey] = true
							if env.Minion != nil && env.Minion == castingMinion {
								env.Minion.ModDB.Conditions["AffectedBy"+minionBuffNameKey] = true
							}
							if buff.Type == "Debuff" {
								inc := minionSkillModList.Sum(mod.TypeIncrease, minionSkillCfg, "DebuffEffect")
								more := minionSkillModList.More(minionSkillCfg, "DebuffEffect")
								mult = (1 + inc/100) * more
							}
							srcList.ScaleAddList(buff.ModList, mult*stackCount)
							if minionSkill.SkillData.StackCount != nil {
								srcList.NewMod("Multiplier:"+buff.Name+"Stack", mod.TypeBase, *minionSkill.SkillData.StackCount, mod.Source(buff.Name))
							}
							mergeBuff(srcList, debuffs, buff.Name)
						}
					}
				}
			}
		}
	}

	//pob:block:src/Modules/CalcPerform.lua:2941-2958
	if others := allyBuffs.byType["otherEffects"]; others != nil {
		for _, buffName := range sortedPartyBuffNames(others.buffs) {
			buff := others.buffs[buffName]
			buffNameKey := strings.ReplaceAll(buffName, " ", "")
			env.ModDB.Conditions["AffectedBy"+buffNameKey] = true
			inc := env.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			more := env.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			srcList := moddb.NewModList()
			srcList.ScaleAddList(buff.modList, (buff.effectMult+inc)/100*more)
			mergeBuff(srcList, buffs, buffName)
			if env.Minion != nil {
				env.Minion.ModDB.Conditions["AffectedBy"+buffNameKey] = true
				inc := env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				more := env.Minion.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				srcList := moddb.NewModList()
				srcList.ScaleAddList(buff.modList, (buff.effectMult+inc)/100*more)
				mergeBuff(srcList, minionBuffs, buffName)
			}
		}
	}
	//pob:block:src/Modules/CalcPerform.lua:2959-2999
	if auras := allyBuffs.byType["Aura"]; auras != nil {
		for _, vaal := range []bool{false, true} {
			list := auras.buffs
			if vaal {
				list = auras.vaal
			}
			for _, auraName := range sortedPartyBuffNames(list) {
				aura := list[auraName]
				auraNameCompressed := strings.ReplaceAll(auraName, " ", "")
				if !env.ModDB.Flag(nil, "AlliesAurasCannotAffectSelf") && !env.ModDB.Conditions["AffectedBy"+auraNameCompressed] {
					env.ModDB.Conditions["AffectedByAura"] = true
					if vaal {
						base := ""
						if len(auraName) > 5 {
							base = strings.ReplaceAll(auraName[5:], " ", "")
						}
						env.ModDB.Conditions["AffectedBy"+base] = true
					}
					env.ModDB.Conditions["AffectedBy"+auraNameCompressed] = true
					srcList := moddb.NewModList()
					srcList.ScaleAddList(aura.modList, aura.effectMult/100)
					mergeBuff(srcList, buffs, auraName)
				}
				if env.Minion != nil && !env.Minion.ModDB.Conditions["AffectedBy"+auraNameCompressed] {
					env.Minion.ModDB.Conditions["AffectedByAura"] = true
					env.Minion.ModDB.Conditions["AffectedBy"+auraNameCompressed] = true
					srcList := moddb.NewModList()
					srcList.ScaleAddList(aura.modList, aura.effectMult/100)
					mergeBuff(srcList, minionBuffs, auraName)
				}
			}
		}
	}
	//pob:block:src/Modules/CalcPerform.lua:3000-3025
	if auraDebuffs := allyBuffs.byType["AuraDebuff"]; auraDebuffs != nil && env.ModeEffective {
		for _, list := range []map[string]*partyBuff{auraDebuffs.buffs, auraDebuffs.vaal} {
			for _, auraName := range sortedPartyBuffNames(list) {
				aura := list[auraName]
				auraNameCompressed := strings.ReplaceAll(auraName, " ", "")
				if !env.EnemyModDB.Conditions["AffectedBy"+auraNameCompressed] {
					env.EnemyModDB.Conditions["AffectedBy"+auraNameCompressed] = true
					env.ModDB.Conditions["AffectedBy"+auraNameCompressed] = true
					srcList := moddb.NewModList()
					srcList.ScaleAddList(aura.modList, aura.effectMult/100)
					mergeBuff(srcList, debuffs, auraName)
				}
			}
		}
	}
	//pob:block:src/Modules/CalcPerform.lua:3026-3048
	if warcries := partyMembers.Warcry.byType["Warcry"]; warcries != nil {
		for _, warcryName := range sortedPartyBuffNames(warcries.buffs) {
			warcry := warcries.buffs[warcryName]
			warcryNameCompressed := strings.ReplaceAll(warcryName, " ", "")
			scaled := func() *moddb.ModList {
				srcList := moddb.NewModList()
				for _, warcryBuff := range warcry.modList.Mods() {
					bonus, ok := warcry.warcryPowerBonus[warcryBuff]
					if !ok {
						bonus = 1
					}
					srcList.ScaleAddList(singleModList(warcryBuff), warcry.effectMult/100*bonus)
				}
				return srcList
			}
			if !env.ModDB.Conditions["AffectedBy"+warcryNameCompressed] {
				env.ModDB.Conditions["AffectedByWarcry"] = true
				env.ModDB.Conditions["AffectedBy"+warcryNameCompressed] = true
				mergeBuff(scaled(), buffs, warcryName)
			}
			if env.Minion != nil && !env.Minion.ModDB.Conditions["AffectedBy"+warcryNameCompressed] {
				env.Minion.ModDB.Conditions["AffectedByWarcry"] = true
				env.Minion.ModDB.Conditions["AffectedBy"+warcryNameCompressed] = true
				mergeBuff(scaled(), minionBuffs, warcryName)
			}
		}
	}
	//pob:block:src/Modules/CalcPerform.lua:3049-3060
	if links := partyMembers.Link.byType["Link"]; links != nil {
		for _, linkName := range sortedPartyBuffNames(links.buffs) {
			link := links.buffs[linkName]
			linkNameCompressed := strings.ReplaceAll(linkName, " ", "")
			if !env.ModDB.Conditions["AffectedBy"+linkNameCompressed] {
				env.ModDB.Conditions["AffectedByLink"] = true
				env.ModDB.Conditions["AffectedBy"+linkNameCompressed] = true
				srcList := moddb.NewModList()
				srcList.ScaleAddList(link.modList, link.effectMult/100)
				mergeBuff(srcList, buffs, linkName)
			}
		}
	}

	//pob:block:src/Modules/CalcPerform.lua:3062-3066
	if env.ModeCombat {
		//pob:src/Modules/CalcPerform.lua:1567-1576
		effectInc := env.ModDB.Sum(mod.TypeIncrease, playerActorCfg, "FlaskEffect")
		flaskTotalRateInc := env.ModDB.Sum(mod.TypeIncrease, nil, "FlaskRecoveryRate")
		flaskDurInc := env.ModDB.Sum(mod.TypeIncrease, nil, "FlaskDuration")

		recoveryBuffs := make(map[string]*moddb.ModList)
		recoveryOrder := make([]string, 0, 2)
		flaskMinion.reset(env)
		checkNonRecoveryFlasksForMinions := flaskMinion.nonUniqueApply

		//pob:block:src/Modules/CalcPerform.lua:1761-1767
		for _, item := range env.Flasks {
			if item == nil || item.Base == nil || item.Base.Flask == nil {
				continue
			}
			if item.Base.Flask.Life > 0 && !env.ModDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
				recoveryOrder = mergeFlaskRecovery(env, item, "Life", "LifeFlask",
					effectInc, flaskDurInc, flaskTotalRateInc, recoveryBuffs, recoveryOrder)
				if checkNonRecoveryFlasksForMinions {
					flaskMinion.calcFlaskMods(item, "LifeFlask",
						calcFlaskRecovery(env, "Life", item, effectInc, flaskDurInc, flaskTotalRateInc), nil)
				}
			}
			if item.Base.Flask.Mana > 0 {
				recoveryOrder = mergeFlaskRecovery(env, item, "Mana", "ManaFlask",
					effectInc, flaskDurInc, flaskTotalRateInc, recoveryBuffs, recoveryOrder)
				if checkNonRecoveryFlasksForMinions {
					flaskMinion.calcFlaskMods(item, "ManaFlask",
						calcFlaskRecovery(env, "Mana", item, effectInc, flaskDurInc, flaskTotalRateInc), nil)
				}
			}
			//pob:block:src/Modules/CalcPerform.lua:1768-1770
			if checkNonRecoveryFlasksForMinions {
				flaskMinion.calcFlaskMods(item, item.BaseName, item.BuffModList, item.Mods)
			}
		}

		//pob:block:src/Modules/CalcPerform.lua:1775-1789
		if !env.ModDB.Flag(nil, "UtilityFlasksDoNotApplyToPlayer") && !env.ModDB.Flag(nil, "FlasksDoNotApplyToPlayer") {
			for _, key := range recoveryOrder {
				env.ModDB.AddList(recoveryBuffs[key])
			}
		}
		flaskMinion.applyToMinion(env)
	}

	//pob:block:src/Modules/CalcPerform.lua:3068-3085
	if env.ModeCombat {
		if env.ModDB.Flag(nil, "UseEnduranceCharges") && env.ModDB.Flag(nil, "EnduranceChargesConvertToBrutalCharges") {
			tripleDmgChancePerEndurance := env.ModDB.Sum(mod.TypeBase, nil, "PerBrutalTripleDamageChance")
			env.ModDB.AddMod(mod.NewFloat("TripleDamageChance", "BASE", tripleDmgChancePerEndurance).Tag(mod.Multiplier("BrutalCharge")))
		}
		if env.ModDB.Flag(nil, "UseFrenzyCharges") && env.ModDB.Flag(nil, "FrenzyChargesConvertToAfflictionCharges") {
			dmgPerAffliction := env.ModDB.Sum(mod.TypeBase, nil, "PerAfflictionAilmentDamage")
			effectPerAffliction := env.ModDB.Sum(mod.TypeBase, nil, "PerAfflictionNonDamageEffect")
			env.ModDB.AddMod(mod.NewFloat("Damage", "MORE", dmgPerAffliction).Source("Affliction Charges").KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("AfflictionCharge")))
			env.ModDB.AddMod(mod.NewFloat("EnemyChillEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
			env.ModDB.AddMod(mod.NewFloat("EnemyShockEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
			env.ModDB.AddMod(mod.NewFloat("EnemyFreezeEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
			env.ModDB.AddMod(mod.NewFloat("EnemyScorchEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
			env.ModDB.AddMod(mod.NewFloat("EnemyBrittleEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
			env.ModDB.AddMod(mod.NewFloat("EnemySapEffect", "MORE", effectPerAffliction).Source("Affliction Charges").Tag(mod.Multiplier("AfflictionCharge")))
		}
	}

	//pob:block:src/Modules/CalcPerform.lua:2200-2243
	extraCurseSources := []struct {
		dest  *[]*curseEntry
		store *moddb.ModDB
	}{{&curses, env.ModDB}}
	if env.Minion != nil && env.Minion.ModDB != nil {
		extraCurseSources = append(extraCurseSources, struct {
			dest  *[]*curseEntry
			store *moddb.ModDB
		}{&minionCurses, env.Minion.ModDB})
	}
	for _, source := range extraCurseSources {
		for _, value := range utils.CastSlice[mod.ExtraCurse](source.store.List(nil, "ExtraCurse")) {
			effect := extraCurseEffect(value)
			if effect == nil {
				continue
			}
			curseName := effect.GrantedEffect.Raw.GetActiveSkill().DisplayedName

			gemModList := moddb.NewModList()
			CalcMergeSkillInstanceMods(env, gemModList, effect, nil)

			curseModList := moddb.NewModList()
			for _, m := range gemModList.Mods() {
				for _, tag := range m.Tags() {
					get, isGlobal := tag.(*mod.GlobalEffectTag)
					if isGlobal && len(get.GlobalEffectList) > 0 && get.GlobalEffectList[0] == "Curse" {
						curseModList.AddMod(m)
						break
					}
				}
			}

			if value.ApplyToPlayer {
				if source.store.Sum(mod.TypeBase, nil, "AvoidCurse") < 100 {
					source.store.Conditions["Cursed"] = true
					source.store.Multipliers["CurseOnSelf"] = source.store.Multipliers["CurseOnSelf"] + 1
					source.store.Conditions["AffectedBy"+strings.ReplaceAll(curseName, " ", "")] = true
					cfg := &moddb.ListCfg{SkillName: curseName}
					inc := source.store.Sum(mod.TypeIncrease, cfg, "CurseEffectOnSelf") +
						gemModList.Sum(mod.TypeIncrease, nil, "CurseEffectAgainstPlayer")
					more := source.store.More(cfg, "CurseEffectOnSelf") *
						gemModList.More(nil, "CurseEffectAgainstPlayer")
					//pob:block:src/Modules/CalcPerform.lua:3116
					source.store.ScaleAddList(curseModList, max((1+inc/100)*more, 0))
				}
				continue
			}

			if env.EnemyModDB.Flag(nil, "Hexproof") && !source.store.Flag(nil, "CursesIgnoreHexproof") {
				continue
			}
			curse := &curseEntry{
				name:       curseName,
				fromPlayer: source.store == env.ModDB,
				priority:   determineCursePriority(curseName, nil),
				modList:    moddb.NewModList(),
			}
			curse.modList.ScaleAddList(curseModList,
				(1+env.EnemyModDB.Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf")/100)*
					env.EnemyModDB.More(nil, "CurseEffectOnSelf"))
			*source.dest = append(*source.dest, curse)
		}
	}

	//pob:block:src/Modules/CalcPerform.lua:3131-3152
	allyCurses := make([]*curseEntry, 0)
	allyCurseLimit := 0
	if allyPartyCurses := partyMembers.Curse.byType["Curse"]; allyPartyCurses != nil {
		allyCurseLimit = int(partyMembers.Curse.limit)
		for _, curseName := range sortedPartyBuffNames(allyPartyCurses.buffs) {
			curse := allyPartyCurses.buffs[curseName]
			newCurse := &curseEntry{
				name:     curseName,
				priority: 0,
				modList:  moddb.NewModList(),
			}
			mult := curse.effectMult / 100
			if curse.isMark {
				newCurse.isMark = true
			} else {
				mult *= env.EnemyModDB.More(nil, "CurseEffectOnSelf")
			}
			newCurse.modList.ScaleAddList(curse.modList, mult)
			allyCurses = append(allyCurses, newCurse)
		}
	}
	env.allyCurseCandidates = make([]string, 0, len(allyCurses))
	for _, c := range allyCurses {
		env.allyCurseCandidates = append(env.allyCurseCandidates, c.name)
	}
	if env.Cache != nil {
		allyCurses = pinPairsOrder(allyCurses, env.Cache.pairsPicks[pairsPickPartyCurses], func(c *curseEntry) string { return c.name })
	}

	if env.ModDB.Flag(nil, "CurseLimitIsMaximumPowerCharges") {
		env.Player.Output["EnemyCurseLimit"] = env.Player.Output["PowerChargesMax"]
	} else {
		env.Player.Output["EnemyCurseLimit"] = env.ModDB.Sum(mod.TypeBase, nil, "EnemyCurseLimit")
	}
	//pob:src/Modules/CalcPerform.lua:3158
	if exports != nil {
		exports.curseLimit = env.Player.Output["EnemyCurseLimit"]
	}

	curseSlots := make([]*curseEntry, 0)
	type curseCandidate struct {
		curse *curseEntry
		limit int
	}
	curseCandidates := make([]curseCandidate, 0, len(curses)+len(minionCurses))
	playerCurseLimit := int(env.Player.Output["EnemyCurseLimit"])
	for _, curse := range curses {
		curseCandidates = append(curseCandidates, curseCandidate{curse, playerCurseLimit})
	}
	for _, curse := range minionCurses {
		curseCandidates = append(curseCandidates, curseCandidate{curse, minionCurseLimit})
	}
	//pob:block:src/Modules/CalcPerform.lua:3164
	for _, curse := range allyCurses {
		curseCandidates = append(curseCandidates, curseCandidate{curse, allyCurseLimit})
	}
	// Currently assume only 1 mark is possible
	markSlotted := false
	for _, candidate := range curseCandidates {
		curse, limit := candidate.curse, candidate.limit
		// Calculate curses that ignore hex limit after
		if curse.ignoreHexLimit || curse.socketedCursesHexLimit {
			continue
		}

		slot := -1
		skipAddingCurse := false
		// Check if we need to disable a certain curse aura.
		for _, activeSkill := range env.Player.ActiveSkillList {
			if len(activeSkill.BuffList) > 0 && curse.name == activeSkill.BuffList[0].Name && activeSkill.SkillTypes[data.SkillTypeAura] {
				//pob:src/Modules/CalcPerform.lua:3173
				if env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") ||
					activeSkill.SkillModList.Flag(env.Player.MainSkill.SkillCfg, "SelfAurasAffectYouAndLinkedTarget") {
					skipAddingCurse = true
					break
				}
				if activeSkill.SkillData.ManaReservedBase > env.Player.Output["Mana"] ||
					activeSkill.SkillData.LifeReservedBase > env.Player.Output["Life"] {
					skipAddingCurse = true
				}
				break
			}
		}
		for i := 0; i < limit; i++ {
			// Prevent multiple marks from being considered
			if curse.isMark && markSlotted {
				slot = -1
				break
			}
			if i >= len(curseSlots) {
				slot = i
				break
			} else if curseSlots[i].name == curse.name {
				if curseSlots[i].priority < curse.priority {
					slot = i
				} else {
					slot = -1
				}
				break
			} else if curseSlots[i].priority < curse.priority {
				slot = i
			}
		}
		if slot >= 0 {
			if slot < len(curseSlots) && curseSlots[slot].isMark {
				markSlotted = false
			}
			if !skipAddingCurse {
				if slot < len(curseSlots) {
					curseSlots[slot] = curse
				} else {
					curseSlots = append(curseSlots, curse)
				}
			}
			if curse.isMark {
				markSlotted = true
			}
		}
	}

	//pob:src/Modules/CalcPerform.lua:2184
	for _, candidate := range curseCandidates {
		curse := candidate.curse
		if curse.ignoreHexLimit {
			skipAddingCurse := false
			for i := range curseSlots {
				if curseSlots[i].name == curse.name {
					if curseSlots[i].priority < curse.priority {
						curseSlots[i] = curse
					}
					skipAddingCurse = true
					break
				}
			}
			if !skipAddingCurse {
				curseSlots = append(curseSlots, curse)
			}
		}
		if curse.socketedCursesHexLimit {
			socketedCursesHexLimitValue := env.ModDB.Sum(mod.TypeBase, nil, "SocketedCursesHexLimitValue")
			skipAddingCurse := false
			for i := range curseSlots {
				if curseSlots[i].name == curse.name {
					if curseSlots[i].priority < curse.priority {
						curseSlots[i] = curse
					}
					skipAddingCurse = true
					break
				}
				if float64(i+1) >= socketedCursesHexLimitValue {
					skipAddingCurse = true
				}
			}
			if !skipAddingCurse {
				curseSlots = append(curseSlots, curse)
			}
		}
	}

	type guardSlot struct {
		name    string
		modList *moddb.ModList
	}
	guardSlots := make([]guardSlot, 0)
	nonVaal := false
	env.guardCandidates = guardOrder
	walk := guardOrder
	if env.Cache != nil && env.Cache.guardPick != "" {
		for i, name := range guardOrder {
			if !strings.HasPrefix(name, "Vaal") && strings.ReplaceAll(name, " ", "") == env.Cache.guardPick {
				walk = append([]string{name}, slices.Delete(slices.Clone(guardOrder), i, i+1)...)
				break
			}
		}
	}
	for _, name := range walk {
		if name == "Vaal Molten Shell" {
			guardSlots = guardSlots[:0]
			nonVaal = false
			guardSlots = append(guardSlots, guardSlot{name: name, modList: guards[name]})
			break
		} else if strings.HasPrefix(name, "Vaal") {
			guardSlots = append(guardSlots, guardSlot{name: name, modList: guards[name]})
		} else if !nonVaal {
			guardSlots = append(guardSlots, guardSlot{name: name, modList: guards[name]})
			nonVaal = true
		}
	}
	if nonVaal {
		env.ModDB.Conditions["AffectedByNonVaalGuardSkill"] = true
	}
	for _, guard := range guardSlots {
		env.ModDB.Conditions["AffectedByGuardSkill"] = true
		env.ModDB.Conditions["AffectedBy"+strings.ReplaceAll(guard.name, " ", "")] = true
		mergeBuff(guard.modList, buffs, guard.name)
	}
	//pob:block:src/Modules/CalcPerform.lua:3281
	env.Player.Output["GuardSkillActive"] = utils.Ternary(env.ModDB.Conditions["AffectedByGuardSkill"], 1.0, 0.0)

	for _, name := range sortedModListKeys(buffs) {
		modList := buffs[name]
		env.ModDB.AddList(modList)
		//pob:src/Modules/CalcPerform.lua:3283-3288
		if !notBuff[name] {
			env.ModDB.Multipliers["BuffOnSelf"]++
		}
		//pob:src/Modules/CalcPerform.lua:3289-3291
		if env.Minion != nil {
			for _, value := range modList.List(env.Player.MainSkill.SkillCfg, "MinionModifier") {
				minionMod, ok := value.(mod.MinionModifier)
				if !ok {
					continue
				}
				env.Minion.ModDB.AddMod(minionMod.Mod)
			}
		}
	}

	//pob:src/Modules/CalcPerform.lua:3293-3297
	if env.Minion != nil {
		for _, name := range sortedModListKeys(minionBuffs) {
			env.Minion.ModDB.AddList(minionBuffs[name])
		}
	}

	for _, name := range sortedModListKeys(debuffs) {
		env.EnemyModDB.AddList(debuffs[name])
	}

	env.ModDB.Multipliers["CurseOnEnemy"] = float64(len(curseSlots))
	env.curseSlots = curseSlots
	for _, slot := range curseSlots {
		env.EnemyModDB.Conditions["Cursed"] = true
		if slot.isMark {
			env.EnemyModDB.Conditions["Marked"] = true
		}
		if slot.fromPlayer {
			affectedByCurse[env.Enemy] = true
		}
		if slot.modList != nil {
			env.EnemyModDB.AddList(slot.modList)
		}
		if slot.buffModList != nil {
			env.ModDB.AddList(slot.buffModList)
		}
		if slot.minionBuffModList != nil && env.Minion != nil {
			env.Minion.ModDB.AddList(slot.minionBuffModList)
		}
	}

	/*
		TODO // Combine buffs/debuffs -- CalcPerform.lua:1738-1919.
		POSITION VERIFIED (relocation audit): the live port at :1023-1369 above sits at the
		same point in the order as the Lua -- reservations (:1619) -> attribute requirements
		(:1625) -> heralds (:1698) -> auras affecting self (:1710) -> consecrated ground
		(:1721) -> this loop (:1738). Same file, same neighbours, same sequence.

		The SELF path of every branch (Buff, Guard, Aura, Debuff, AuraDebuff) is live. What
		is still unported is interleaved through the loop rather than contiguous, which is
		why this block is kept whole as the worklist instead of being cut down:
		  - the minion arms -- :1436-1445 (Buff), :1493-1504 (Aura), and minionBuffs;
		  - ExtraAuraEffect -- :1464-1477 and the ScaleAddList calls at :1490 and :1502;
		  - minionCurses and the spectre EnemyCurseLimit scan at :1394-1403.
		buffs := { }
		env.buffs = buffs
		guards := { }
		minionBuffs := { }
		env.minionBuffs = minionBuffs
		debuffs := { }
		env.debuffs = debuffs
		curses := { }
		minionCurses := {
			limit = 1,
		}
		for spectreId = 1, #env.spec.build.spectreList {
			spectreData := data.minions[env.spec.build.spectreList[spectreId]]
			for modId = 1, #spectreData.modList {
				modData := spectreData.modList[modId]
				if modData.name == "EnemyCurseLimit" {
					minionCurses.limit = modData.value + 1
					break
				}
			}
		}
		affectedByAura := { }
		for _, activeSkill := range (env.Player.activeSkillList) {
			skillModList := activeSkill.skillModList
			skillCfg := activeSkill.skillCfg
			for _, buff := range (activeSkill.buffList) {
				//Skip adding buff if reservation exceeds maximum
				for _, value := range ({"Mana", "Life"}) {
					if activeSkill.skillData[value+"ReservedBase"] and activeSkill.skillData[value+"ReservedBase"] > env.Player.output[value] {
						goto disableAura
					}
				}
				if buff.cond and not skillModList:GetCondition(buff.cond, skillCfg) {
					// Nothing!
				} else if buff.enemyCond and not enemyDB:GetCondition(buff.enemyCond) {
					// Also nothing :/
				} else if buff.type == "Buff" {
					if env.mode_buffs and (not activeSkill.skillFlags.totem or buff.allowTotemBuff) {
						skillCfg := buff.activeSkillBuff and skillCfg
						modStore := buff.activeSkillBuff and skillModList or modDB
					 	if not buff.applyNotPlayer {
							activeSkill.buffSkill = true
							env.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
							srcList := new("ModList")
							inc := modStore:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer") + skillModList:Sum(mod.TypeIncrease, skillCfg, buff.name:gsub(" ", "")+"Effect")
							more := modStore:More(skillCfg, "BuffEffect", "BuffEffectOnSelf")
							srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
							mergeBuff(srcList, buffs, buff.name)
							mergeBuff(buff.unscalableModList, buffs, buff.name)
							if activeSkill.skillData.thisIsNotABuff {
								buffs[buff.name].notBuff = true
							}
						}
						if env.Minion and (buff.applyMinions or buff.applyAllies) {
							activeSkill.minionBuffSkill = true
							env.Minion.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
							srcList := new("ModList")
							inc := modStore:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnMinion") + env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
							more := modStore:More(skillCfg, "BuffEffect", "BuffEffectOnMinion") * env.Minion.ModDB.More(nil, "BuffEffectOnSelf")
							srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
							mergeBuff(srcList, minionBuffs, buff.name)
							mergeBuff(buff.unscalableModList, minionBuffs, buff.name)
						}
					}
				} else if buff.type == "Guard" {
					if env.mode_buffs and (not activeSkill.skillFlags.totem or buff.allowTotemBuff) {
						skillCfg := buff.activeSkillBuff and skillCfg
						modStore := buff.activeSkillBuff and skillModList or modDB
					 	if not buff.applyNotPlayer {
							activeSkill.buffSkill = true
							srcList := new("ModList")
							inc := modStore:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer")
							more := modStore:More(skillCfg, "BuffEffect", "BuffEffectOnSelf")
							srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
							mergeBuff(srcList, guards, buff.name)
							mergeBuff(buff.unscalableModList, guards, buff.name)
						}
					}
				} else if buff.type == "Aura" {
					if env.mode_buffs {
						// Check for extra modifiers to apply to aura skills
						extraAuraModList := { }
						for _, value := range (env.ModDB.List(skillCfg, "ExtraAuraEffect")) {
							add := true
							for _, mod := range (extraAuraModList) {
								if modLib.compareModParams(mod, value.mod) {
									mod.value = mod.value + value.mod.value
									add = false
									break
								}
							}
							if add {
								t_insert(extraAuraModList, copyTable(value.mod, true))
							}
						}
						if not activeSkill.skillData.auraCannotAffectSelf {
							activeSkill.buffSkill = true
							affectedByAura[env.Player] = true
							if buff.name:sub(1,4) == "Vaal" {
								env.ModDB.conditions["AffectedBy"+buff.name:sub(6):gsub(" ","")] = true
							}
							env.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
							srcList := new("ModList")
							inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
							more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
							mult := (1 + inc / 100) * more
							srcList:ScaleAddList(buff.modList, mult)
							srcList:ScaleAddList(extraAuraModList, mult)
							mergeBuff(srcList, buffs, buff.name)
						}
						if env.Minion and not (env.ModDB.Flag(nil, "SelfAurasCannotAffectAllies") or env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") or env.ModDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies")) {
							activeSkill.minionBuffSkill = true
							affectedByAura[env.Minion] = true
							env.Minion.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
							srcList := new("ModList")
							inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect") + env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
							more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect") * env.Minion.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
							mult := (1 + inc / 100) * more
							srcList:ScaleAddList(buff.modList, mult)
							srcList:ScaleAddList(extraAuraModList, mult)
							mergeBuff(srcList, minionBuffs, buff.name)
						}
					}
				} else if buff.type == "Debuff" or buff.type == "AuraDebuff" {
					local stackCount
					if buff.stackVar {
						stackCount = skillModList:Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.stackVar)
						if buff.stackLimit {
							stackCount = min(stackCount, buff.stackLimit)
						} else if buff.stackLimitVar {
							stackCount = min(stackCount, skillModList:Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.stackLimitVar))
						}
					} else {
						stackCount = activeSkill.skillData.stackCount or 1
					}
					if env.mode_effective and stackCount > 0 {
						activeSkill.debuffSkill = true
						env.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
						srcList := new("ModList")
						mult := 1
						if buff.type == "AuraDebuff" {
							mult = 0
							if not env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") {
								inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
								more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
								mult = (1 + inc / 100) * more
							}
						}
						if buff.type == "Debuff" {
							inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "DebuffEffect")
							more := skillModList:More(skillCfg, "DebuffEffect")
							mult = (1 + inc / 100) * more
						}
						srcList:ScaleAddList(buff.modList, mult * stackCount)
						if activeSkill.skillData.stackCount or buff.stackVar {
							srcList:NewMod("Multiplier:"+buff.name+"Stack", "BASE", stackCount, buff.name)
						}
						mergeBuff(srcList, debuffs, buff.name)
					}
				} else if buff.type == "Curse" or buff.type == "CurseBuff" {
					mark := activeSkill.skillTypes[SkillType.Mark]
					if env.mode_effective and (not enemyDB:Flag(nil, "Hexproof") or env.ModDB.Flag(nil, "CursesIgnoreHexproof")) or mark {
						curse := {
							name = buff.name,
							fromPlayer = true,
							priority = determineCursePriority(buff.name, activeSkill),
							isMark = mark,
							ignoreHexLimit = env.ModDB.Flag(activeSkill.skillCfg, "CursesIgnoreHexLimit") and not mark or false,
							socketedCursesHexLimit = env.ModDB.Flag(activeSkill.skillCfg, "SocketedCursesAdditionalLimit")
						}
						inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "CurseEffect") + enemyDB:Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf")
						if activeSkill.skillTypes[SkillType.Aura] {
							inc = inc + skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect")
						}
						more := skillModList:More(skillCfg, "CurseEffect")
						// This is non-ideal, but the only More for enemy is the boss effect
						if not curse.isMark {
							more = more * enemyDB:More(nil, "CurseEffectOnSelf")
						}
						mult := 0
						if not (env.ModDB.Flag(nil, "SelfAurasOnlyAffectYou") and activeSkill.skillTypes[SkillType.Aura]) then //If your aura only effect you blasphemy does nothing
							mult = (1 + inc / 100) * more
						}
						if buff.type == "Curse" {
							curse.modList = new("ModList")
							curse.modList:ScaleAddList(buff.modList, mult)
						} else {
							// Curse applies a buff; scale by curse effect, then buff effect
							temp := new("ModList")
							temp:ScaleAddList(buff.modList, mult)
							curse.buffModList = new("ModList")
							buffInc := env.ModDB.Sum(mod.TypeIncrease, skillCfg, "BuffEffectOnSelf")
							buffMore := env.ModDB.More(skillCfg, "BuffEffectOnSelf")
							curse.buffModList:ScaleAddList(temp, (1 + buffInc / 100) * buffMore)
							if env.Minion {
								curse.minionBuffModList = new("ModList")
								buffInc := env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
								buffMore := env.Minion.ModDB.More(nil, "BuffEffectOnSelf")
								curse.minionBuffModList:ScaleAddList(temp, (1 + buffInc / 100) * buffMore)
							}
						}
						t_insert(curses, curse)
					}
				}
				::disableAura::
			}
			if activeSkill.minion and activeSkill.minion.activeSkillList {
				castingMinion := activeSkill.minion
				for _, activeSkill := range (activeSkill.minion.activeSkillList) {
					skillModList := activeSkill.skillModList
					skillCfg := activeSkill.skillCfg
					for _, buff := range (activeSkill.buffList) {
						if buff.type == "Buff" {
							if env.mode_buffs and activeSkill.skillData.enable {
								skillCfg := buff.activeSkillBuff and skillCfg
								modStore := buff.activeSkillBuff and skillModList or castingMinion.modDB
								if buff.applyAllies {
									env.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
									srcList := new("ModList")
									inc := modStore:Sum(mod.TypeIncrease, skillCfg, "BuffEffect") + env.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")
									more := modStore:More(skillCfg, "BuffEffect") * env.ModDB.More(nil, "BuffEffectOnSelf")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, buffs, buff.name)
									mergeBuff(buff.unscalableModList, buffs, buff.name)
								}
								if env.Minion and (env.Minion == castingMinion or buff.applyAllies) {
					 				env.Minion.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
									srcList := new("ModList")
									inc := modStore:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnSelf")
									more := modStore:More(skillCfg, "BuffEffect", "BuffEffectOnSelf")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, minionBuffs, buff.name)
									mergeBuff(buff.unscalableModList, minionBuffs, buff.name)
								}
							}
						} else if buff.type == "Aura" {
							if env.mode_buffs and activeSkill.skillData.enable {
								if not env.ModDB.Flag(nil, "AlliesAurasCannotAffectSelf") {
									srcList := new("ModList")
									inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect") + env.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
									more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect") * env.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, buffs, buff.name)
								}
								if env.Minion and (env.Minion != activeSkill.minion or not activeSkill.skillData.auraCannotAffectSelf) {
									srcList := new("ModList")
									inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "AuraEffect", "BuffEffect") + env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
									more := skillModList:More(skillCfg, "AuraEffect", "BuffEffect") * env.Minion.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, minionBuffs, buff.name)
								}
							}
						} else if buff.type == "Curse" {
							if env.mode_effective and activeSkill.skillData.enable and (not enemyDB:Flag(nil, "Hexproof") or activeSkill.skillTypes[SkillType.Mark]) {
								curse := {
									name = buff.name,
									priority = determineCursePriority(buff.name, activeSkill),
								}
								inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "CurseEffect") + enemyDB:Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf")
								more := skillModList:More(skillCfg, "CurseEffect") * enemyDB:More(nil, "CurseEffectOnSelf")
								curse.modList = new("ModList")
								curse.modList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
								t_insert(minionCurses, curse)
							}
						} else if buff.type == "Debuff" {
							local stackCount
							if buff.stackVar {
								stackCount = env.ModDB.Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.stackVar)
								if buff.stackLimit {
									stackCount = min(stackCount, buff.stackLimit)
								} else if buff.stackLimitVar {
									stackCount = min(stackCount, env.ModDB.Sum(mod.TypeBase, skillCfg, "Multiplier:"+buff.stackLimitVar))
								}
							} else {
								stackCount = activeSkill.skillData.stackCount or 1
							}
							if env.mode_effective and stackCount > 0 {
								activeSkill.debuffSkill = true
								srcList := new("ModList")
								srcList:ScaleAddList(buff.modList, stackCount)
								if activeSkill.skillData.stackCount {
									srcList:NewMod("Multiplier:"+buff.name+"Stack", "BASE", activeSkill.skillData.stackCount, buff.name)
								}
								mergeBuff(srcList, debuffs, buff.name)
							}
						}
					}
				}
			}
		}
	*/

	/*
		TODO // Limited support for handling buffs originating from Spectres
		for _, activeSkill := range (env.Player.activeSkillList) {
			if activeSkill.minion {
				for _, activeMinionSkill := range (activeSkill.minion.activeSkillList) {
					if activeMinionSkill.skillData.enable {
						skillModList := activeMinionSkill.skillModList
						skillCfg := activeMinionSkill.skillCfg
						for _, buff := range (activeMinionSkill.buffList) {
							if buff.type == "Buff" {
								if buff.applyAllies {
									activeMinionSkill.buffSkill = true
									env.ModDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
									srcList := new("ModList")
									inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnPlayer")
									more := skillModList:More(skillCfg, "BuffEffect", "BuffEffectOnPlayer")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, buffs, buff.name)
									mergeBuff(buff.modList, buffs, buff.name)
									if activeMinionSkill.skillData.thisIsNotABuff {
										buffs[buff.name].notBuff = true
									}
								}
								if buff.applyMinions {
									activeMinionSkill.minionBuffSkill = true
									activeSkill.minion.modDB.conditions["AffectedBy"+buff.name:gsub(" ","")] = true
									srcList := new("ModList")
									inc := skillModList:Sum(mod.TypeIncrease, skillCfg, "BuffEffect", "BuffEffectOnMinion")
									more := skillModList:More(skillCfg, "BuffEffect", "BuffEffectOnMinion")
									srcList:ScaleAddList(buff.modList, (1 + inc / 100) * more)
									mergeBuff(srcList, minionBuffs, buff.name)
									mergeBuff(buff.modList, minionBuffs, buff.name)
									if activeMinionSkill.skillData.thisIsNotABuff {
										buffs[buff.name].notBuff = true
									}
								}
							}
						}
					}
				}
			}
		}
	*/

	/*
		TODO // Check for extra curses
		for dest, modDB := range ({[curses] = modDB, [minionCurses] = env.Minion and env.Minion.ModDB}) {
			for _, value := range (env.ModDB.List(nil, "ExtraCurse")) {
				gemModList := new("ModList")
				grantedEffect := env.data.skills[value.skillId]
				if grantedEffect {
					calcs.mergeSkillInstanceMods(env, gemModList, {
						grantedEffect = grantedEffect,
						level = value.level,
						quality = 0,
					})
					curseModList := { }
					for _, mod := range (gemModList) {
						for _, tag := range (mod) {
							if tag.type == "GlobalEffect" and tag.effectType == "Curse" {
								t_insert(curseModList, mod)
								break
							}
						}
					}
					if value.applyToPlayer {
						// Sources for curses on the player don't usually respect any kind of limit, so there's little point bothering with slots
						if env.ModDB.Sum(mod.TypeBase, nil, "AvoidCurse") < 100 {
							env.ModDB.conditions["Cursed"] = true
							env.ModDB.multipliers["CurseOnSelf"] = (env.ModDB.multipliers["CurseOnSelf"] or 0) + 1
							env.ModDB.conditions["AffectedBy"+grantedEffect.name:gsub(" ","")] = true
							cfg := { skillName = grantedEffect.name }
							inc := env.ModDB.Sum(mod.TypeIncrease, cfg, "CurseEffectOnSelf") + gemModList:Sum(mod.TypeIncrease, nil, "CurseEffectAgainstPlayer")
							more := env.ModDB.More(cfg, "CurseEffectOnSelf") * gemModList:More(nil, "CurseEffectAgainstPlayer")
							env.ModDB.ScaleAddList(curseModList, (1 + inc / 100) * more)
						}
					} else if not enemyDB:Flag(nil, "Hexproof") or env.ModDB.Flag(nil, "CursesIgnoreHexproof") {
						curse := {
							name = grantedEffect.name,
							fromPlayer = (dest == curses),
							priority = determineCursePriority(grantedEffect.name),
						}
						curse.modList = new("ModList")
						curse.modList:ScaleAddList(curseModList, (1 + enemyDB:Sum(mod.TypeIncrease, nil, "CurseEffectOnSelf") / 100) * enemyDB:More(nil, "CurseEffectOnSelf"))
						t_insert(dest, curse)
					}
				}
			}
		}
	*/

	/*
		TODO // Process Triggered Skill and Set Trigger Conditions
		// Cospri's Malice
		if env.Player.MainSkill.skillData.triggeredByCospris and not env.Player.MainSkill.skillFlags.minion {
			spellCount := {}
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill := range (env.Player.activeSkillList) {
				if skill.skillTypes[SkillType.Melee] and band(skill.skillCfg.flags, bor(ModFlag.Sword, ModFlag.Weapon1H)) > 0 and skill != env.Player.MainSkill {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredByCospris and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot {
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = skill.skillData.cooldown / icdr, next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByCospris = nil
				env.Player.MainSkill.infoMessage = "No Cospri Triggering Skill Found"
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed
				dualWield := false

				sourceAPS, dualWield = calcDualWieldImpact(env, sourceAPS, source.skillData.doubleHitsWhenDualWielding)

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown, dualWield)

				// Account for chance to hit/crit
				sourceCritChance := GlobalCache.cachedData["CACHE"][uuid].CritChance
				trigRate = trigRate * sourceCritChance / 100
				if breakdown != nil {
					breakdown.Speed = {
						fmt.Sprintf("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						fmt.Sprintf("x %.2f%% ^8(%s effective crit chance)", sourceCritChance, source.activeEffect.grantedEffect.name),
						fmt.Sprintf("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = trigRate
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "Cospri Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.Player.MainSkill.infoTrigger = "Cospri"
			}
		}
	*/
	/*
		TODO // Mjolner
		if env.Player.MainSkill.skillData.triggeredByMjolner and not env.Player.MainSkill.skillFlags.minion {
			spellCount := {}
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill := range (env.Player.activeSkillList) {
				if (skill.skillTypes[SkillType.Damage] or skill.skillTypes[SkillType.Attack]) and band(skill.skillCfg.flags, bor(ModFlag.Mace, ModFlag.Weapon1H)) > 0 and skill != env.Player.MainSkill {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredByMjolner and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot {
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = skill.skillData.cooldown / icdr, next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByMjolner = nil
				env.Player.MainSkill.infoMessage = "No Mjolner Triggering Skill Found"
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed
				dualWield := false

				sourceAPS, dualWield = calcDualWieldImpact(env, sourceAPS, source.skillData.doubleHitsWhenDualWielding)

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown, dualWield)

				// Account for chance to hit/crit
				sourceHitChance := GlobalCache.cachedData["CACHE"][uuid].HitChance
				trigRate = trigRate * sourceHitChance / 100
				if breakdown != nil {
					breakdown.Speed = {
						fmt.Sprintf("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						fmt.Sprintf("x %.0f%% ^8(%s hit chance)", sourceHitChance, source.activeEffect.grantedEffect.name),
						fmt.Sprintf("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = trigRate
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "Mjolner Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.Player.MainSkill.infoTrigger = "Mjolner"
			}
		}
	*/
	//pob:src/Modules/CalcTriggers.lua:1176-1178
	if ms := env.Player.MainSkill; ms != nil && ms.SkillData != nil && ms.SkillData.TriggeredByDamageTaken != 0 {
		env.Player.Output["CWDTThreshold"] = ms.SkillData.TriggeredByDamageTaken *
			calclib.Mod(ms.SkillModList, nil, "CWDTThreshold")
	}

	/*
		TODO // Kitava's Thirst
		if env.Player.MainSkill.skillData.triggeredByManaSpent and not env.Player.MainSkill.skillFlags.minion {
			triggerName := "Kitava"
			spellCount := 0
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "CooldownRecovery")
			reqManaCost := env.Player.modDB.Sum(mod.TypeBase, nil, "KitavaRequiredManaCost")
			trigRate := 0
			source := nil
			for _, skill := range (env.Player.activeSkillList) {
				if not skill.skillTypes[SkillType.Triggered] and skill != env.Player.MainSkill and not skill.skillData.triggeredByManaSpent {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate, reqManaCost, nil)
				}
				if skill.skillData.triggeredByManaSpent and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot {
					spellCount = spellCount + 1
				}
			}

			if not source or spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByManaSpent = nil
				env.Player.MainSkill.infoMessage = fmt.Sprintf("No %s Triggering Skill Found", triggerName)
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true

				actor.Output["ActionTriggerRate"] = getTriggerActionTriggerRate(env.Player.MainSkill.skillData.cooldown, env, breakdown)

				// Get action trigger rate
				kitavaCD := getTriggerDefaultCooldown(env.Player.MainSkill.supportList, "SupportCastOnManaSpent")

				trigRate = icdr / kitavaCD
				actor.Output["SourceTriggerRate"] = trigRate
				actor.Output["ServerTriggerRate"] = min(actor.Output["SourceTriggerRate"], actor.Output["ActionTriggerRate"])
				if breakdown != nil {
					modActionCooldown := kitavaCD / icdr
					rateCapAdjusted := m_ceil(modActionCooldown * data.misc.ServerTickRate) / data.misc.ServerTickRate
					extraICDRNeeded := m_ceil((modActionCooldown - rateCapAdjusted + data.misc.ServerTickTime) * icdr * 1000)
					breakdown.SimData = {
						fmt.Sprintf("%.2f ^8(base cooldown of kitava's trigger)", kitavaCD),
						fmt.Sprintf("/ %.2f ^8(increased/reduced cooldown recovery)", icdr),
						fmt.Sprintf("= %.4f ^8(final cooldown of trigger)", modActionCooldown),
						fmt.Sprintf(""),
						fmt.Sprintf("%.3f ^8(adjusted for server tick rate)", rateCapAdjusted),
						fmt.Sprintf("^8(extra ICDR of %d%% would reach next breakpoint)", extraICDRNeeded),
						fmt.Sprintf(""),
						fmt.Sprintf("Trigger rate:"),
						fmt.Sprintf("1 / %.3f", rateCapAdjusted),
						fmt.Sprintf("= %.2f ^8per second", 1 / rateCapAdjusted),
					}
					breakdown.ServerTriggerRate = {
						fmt.Sprintf("%.2f ^8(smaller of 'cap' and 'skill' trigger rates)", actor.Output["ServerTriggerRate"]),
					}
				}

				// Account for chance to trigger
				kitavaTriggerChance := env.Player.modDB.Sum(mod.TypeBase, nil, "KitavaTriggerChance")
				trigRate = actor.Output["ServerTriggerRate"] * kitavaTriggerChance / 100
				if breakdown != nil {
					breakdown.Speed = {
						fmt.Sprintf("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						fmt.Sprintf("x %.2f%% ^8(kitava's trigger chance)", kitavaTriggerChance),
						fmt.Sprintf("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = trigRate
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "Kitava's Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.Player.MainSkill.infoTrigger = triggerName
			}
		}
	*/
	/*
		TODO // Helmet Focus Trigger
		if env.Player.MainSkill.skillData.triggeredByFocus and not env.Player.MainSkill.skillFlags.minion {
			triggerName := "Focus"
			spellCount := 0
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "FocusCooldownRecovery")
			trigRate := 0
			source := env.Player.modDB.Flag(nil, "Condition:Focused")
			for _, skill := range (env.Player.activeSkillList) {
				if skill.skillData.triggeredByFocus and env.Player.MainSkill.socketGroup.slot == skill.socketGroup.slot {
					spellCount = spellCount + 1
				}
			}
			if not source or spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByFocus = nil
				env.Player.MainSkill.infoMessage = fmt.Sprintf("No %s Triggering Skill Found", triggerName)
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true

				actor.Output["ActionTriggerRate"] = getTriggerActionTriggerRate(env.Player.MainSkill.skillData.cooldown, env, breakdown, true)

				// Get action trigger rate
				skillFocus := env.data.skills["Focus"]
				focusCD := skillFocus.levels[1].cooldown

				trigRate = icdr / focusCD
				actor.Output["SourceTriggerRate"] = trigRate
				actor.Output["ServerTriggerRate"] = min(actor.Output["SourceTriggerRate"], actor.Output["ActionTriggerRate"])
				if breakdown != nil {
					modActionCooldown := focusCD / icdr
					rateCapAdjusted := m_ceil(modActionCooldown * data.misc.ServerTickRate) / data.misc.ServerTickRate
					breakdown.SimData = {
						fmt.Sprintf("%.2f ^8(base cooldown of focus trigger)", focusCD),
						fmt.Sprintf("/ %.2f ^8(increased/reduced cooldown recovery)", icdr),
						fmt.Sprintf("= %.4f ^8(final cooldown of trigger)", modActionCooldown),
						fmt.Sprintf(""),
						fmt.Sprintf("%.3f ^8(adjusted for server tick rate)", rateCapAdjusted),
						fmt.Sprintf(""),
						fmt.Sprintf("Trigger rate:"),
						fmt.Sprintf("1 / %.3f", rateCapAdjusted),
						fmt.Sprintf("= %.2f ^8per second", 1 / rateCapAdjusted),
					}
					breakdown.ServerTriggerRate = {
						fmt.Sprintf("%.2f ^8(smaller of 'cap' and 'skill' trigger rates)", actor.Output["ServerTriggerRate"]),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = actor.Output["ServerTriggerRate"]
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "Focus Triggering Skill Found"
				env.Player.MainSkill.infoTrigger = triggerName
				env.Player.MainSkill.skillFlags.dontDisplay = true
			}
		}
	*/
	if env.Player.MainSkill.SkillData.TriggeredByUnique && !env.Player.MainSkill.SkillFlags[SkillFlagMinion] {
		uniqueTriggerName := getUniqueItemTriggerName(env.Player.MainSkill)
		spellCount := make([]*triggerRotationSkill, 0)
		trigRate := float64(0)
		var source *ActiveSkill
		for _, skill := range env.Player.ActiveSkillList {
			//pob:src/Modules/CalcTriggers.lua:64-66
			rotationEntry := func() *triggerRotationSkill {
				return packageSkillDataForSimulation(env, skill)
			}
			sameSlot := socketGroupSlot(env.Player.MainSkill) == socketGroupSlot(skill)
			switch uniqueTriggerName {
			case "Poet's Pen":
				if (skill.SkillTypes[data.SkillTypeDamage] || skill.SkillTypes[data.SkillTypeAttack]) && skillCfgHasFlag(skill, mod.MFlagWand) && skill != env.Player.MainSkill && !skill.SkillData.TriggeredByUnique {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate, nil, nil)
				}
				if skill.SkillData.TriggeredByUnique && sameSlot && skill.SkillTypes[data.SkillTypeSpell] {
					spellCount = append(spellCount, rotationEntry())
				}
			case "Maloney's Mechanism":
				if skill.SkillTypes[data.SkillTypeAttack] && skillCfgHasFlag(skill, mod.MFlagBow) && skill != env.Player.MainSkill && !skill.SkillData.TriggeredByUnique {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate, nil, nil)
				}
				if skill.SkillData.TriggeredByUnique && sameSlot && skill.SkillTypes[data.SkillTypeRangedAttack] {
					spellCount = append(spellCount, rotationEntry())
				}
			case "Asenath's Chant":
				if (skill.SkillTypes[data.SkillTypeDamage] || skill.SkillTypes[data.SkillTypeAttack]) && skillCfgHasFlag(skill, mod.MFlagBow) && skill != env.Player.MainSkill && !skill.SkillData.TriggeredByUnique {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate, nil, nil)
				}
				if skill.SkillData.TriggeredByUnique && sameSlot && skill.SkillTypes[data.SkillTypeSpell] {
					spellCount = append(spellCount, rotationEntry())
				}
			}
		}
		if source == nil || len(spellCount) < 1 {
			env.Player.MainSkill.SkillData.TriggeredByUnique = false
		} else {
			env.Player.MainSkill.SkillData.Triggered = true
			cached := buildActiveSkillCache(env, source)
			sourceAPS := cached.Speed

			sourceAPS, _ = calcDualWieldImpact(env, sourceAPS, source.SkillData.DoubleHitsWhenDualWielding)

			trigRate = calcActualTriggerRate(env, source, &sourceAPS, spellCount, env.Player.Output, 100)

			// Account for Trigger-related INC/MORE modifiers
			addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)

			env.Player.MainSkill.SkillData.TriggerRate = utils.Ptr(trigRate)
		}
	}
	/*
		TODO // Cast On Melee Kill Support (CoMK)
		if env.Player.MainSkill.skillData.triggeredByMeleeKill and not env.Player.MainSkill.skillFlags.minion and env.ModDB.Flag(nil, "Condition:KilledRecently") {
			spellCount := {}
			icdr := calcLib.mod(env.Player.MainSkill.skillModList, env.Player.MainSkill.skillCfg, "CooldownRecovery")
			trigRate := 0
			source := nil
			for _, skill := range (env.Player.activeSkillList) {
				match1 := env.Player.MainSkill.activeEffect.grantedEffect.fromItem and skill.socketGroup.slot == env.Player.MainSkill.socketGroup.slot
				match2 := (not env.Player.MainSkill.activeEffect.grantedEffect.fromItem) and skill.socketGroup == env.Player.MainSkill.socketGroup
				if skill.skillTypes[SkillType.Attack] and skill.skillTypes[SkillType.Melee] and skill != env.Player.MainSkill and (match1 or match2) {
					source, trigRate = findTriggerSkill(env, skill, source, trigRate)
				}
				if skill.skillData.triggeredByMeleeKill and (match1 or match2) {
					cooldownOverride := skill.skillModList:Override(env.Player.MainSkill.skillCfg, "CooldownRecovery")
					t_insert(spellCount, { uuid = cacheSkillUUID(skill), cd = cooldownOverride or (skill.skillData.cooldown / icdr), next_trig = 0, count = 0 })
				}
			}
			if not source or #spellCount < 1 {
				env.Player.MainSkill.skillData.triggeredByMeleeKill = nil
				env.Player.MainSkill.infoMessage = "No CoMK Triggering Skill Found"
				env.Player.MainSkill.infoMessage2 = "DPS reported assuming Self-Cast"
				env.Player.MainSkill.infoTrigger = ""
			} else {
				env.Player.MainSkill.skillData.triggered = true
				uuid := cacheSkillUUID(source)
				sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed

				// Get action trigger rate
				trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, output, breakdown)

				// Account for chance to trigger on Melee Kill
				trigRate = trigRate * source.skillData.chanceToTriggerOnMeleeKill / 100

				if breakdown != nil {
					breakdown.Speed = {
						fmt.Sprintf("%.2fs ^8(adjusted trigger rate)", actor.Output["ServerTriggerRate"]),
						fmt.Sprintf("x %.2f%% ^8(chance to trigger on melee kill)", source.skillData.chanceToTriggerOnMeleeKill),
						fmt.Sprintf("= %.2f ^8per second", trigRate),
					}
				}

				// Account for Trigger-related INC/MORE modifiers
				addTriggerIncMoreMods(env.Player.MainSkill, env.Player.MainSkill)
				env.Player.MainSkill.skillData.triggerRate = trigRate
				env.Player.MainSkill.skillData.triggerSource = source
				env.Player.MainSkill.infoMessage = "CoMK Triggering Skill: " + source.activeEffect.grantedEffect.name
				env.Player.MainSkill.infoTrigger = "CoMK"
			}
		}
	*/
	/*
		TODO // Triggered by parent attack
		if env.Minion and env.Player.MainSkill.minion {
			if env.Minion.mainSkill.skillData.triggeredByParentAttack {
				spellCount := {}
				trigRate := 0
				source := nil
				for _, skill := range (env.Player.activeSkillList) {
					if skill.skillTypes[SkillType.Attack] and skill != env.Player.MainSkill {
						source, trigRate = findTriggerSkill(env, skill, source, trigRate)
					}
				}

				icdr := calcLib.mod(env.Minion.mainSkill.skillModList, env.Minion.mainSkill.skillCfg, "CooldownRecovery")
				t_insert(spellCount, { uuid = cacheSkillUUID(env.Minion.mainSkill), cd = env.Minion.mainSkill.skillData.cooldown / icdr, next_trig = 0, count = 0 })

				if not source {
					env.Minion.mainSkill.skillData.triggeredByParentAttack = nil
					env.Minion.mainSkill.infoMessage = "No triggering Skill Found"
					env.Minion.mainSkill.infoMessage2 = "DPS reported assuming regular cast"
					env.Minion.mainSkill.infoTrigger = ""
				} else {
					env.Minion.mainSkill.skillData.triggered = true
					uuid := cacheSkillUUID(source)

					sourceAPS := GlobalCache.cachedData["CACHE"][uuid].Speed

					// Get action trigger rate
					trigRate = calcActualTriggerRate(env, source, sourceAPS, spellCount, env.Minion.output, env.Minion.breakdown, false, true)

					// Account for chance to hit
					sourceHitChance := GlobalCache.cachedData["CACHE"][uuid].HitChance
					trigRate = trigRate * sourceHitChance / 100
					if env.Minion.breakdown {
						env.Minion.breakdown.Speed = {
							fmt.Sprintf("%.2fs ^8(adjusted trigger rate)", env.Minion.output.ServerTriggerRate),
							fmt.Sprintf("x %.2f%% ^8(%s Hit chance)", sourceHitChance, source.activeEffect.grantedEffect.name),
							fmt.Sprintf("= %.2f ^8per second", trigRate),
						}
					}

					// Account for Trigger-related INC/MORE modifiers
					addTriggerIncMoreMods(env.Minion.mainSkill, env.Minion.mainSkill)
					env.Minion.mainSkill.skillData.triggerRate = trigRate
					env.Minion.mainSkill.skillData.triggerSource = source
					env.Minion.mainSkill.infoMessage = "Triggering Skill: " + source.activeEffect.grantedEffect.name
					env.Minion.mainSkill.infoTrigger = "Parent attack"
				}
			}
		}
	*/

	// 		If the config is missing (blank), then use the maximum number of stacks
	//		If the config is larger than the maximum number of stacks, replace it with the correct maximum
	maxImpaleStacks := impaleStacksCap(env.ModDB)
	if !env.EnemyModDB.HasMod(mod.TypeBase, nil, "Multiplier:ImpaleStacks") {
		env.EnemyModDB.AddMod(mod.NewFloat("Multiplier:ImpaleStacks", mod.TypeBase, maxImpaleStacks).Source("Config").Tag(mod.Condition("Combat")))
	} else if env.EnemyModDB.Sum(mod.TypeBase, nil, "Multiplier:ImpaleStacks") > maxImpaleStacks {
		env.EnemyModDB.ReplaceMod(mod.NewFloat("Multiplier:ImpaleStacks", mod.TypeBase, maxImpaleStacks).Source("Config").Tag(mod.Condition("Combat")))
	}

	//pob:block:src/Modules/CalcPerform.lua:3328-3343
	if env.ModDB.Flag(nil, "ManaIncreasedByOvercappedLightningRes") {
		tempModDB := moddb.NewModDB()
		tempModDB.Parent = env.ModDB
		tempResistActor := &Actor{
			ModDB:           tempModDB,
			Output:          env.Player.Output,
			ActiveSkillList: env.Player.ActiveSkillList,
			Enemy:           env.Player.Enemy,
			Breakdown:       env.Player.Breakdown,
			ItemList:        env.Player.ItemList,
		}
		tempResistActor.Player = tempResistActor
		calcResistances(tempResistActor, tempModDB)
		doActorLifeMana(env.Player)
		doActorLifeManaReservation(env.Player, true)
	}

	//pob:src/Modules/CalcPerform.lua:3352
	if exports != nil {
		exports.aura["extraAura"] = &partyBuffExport{effectMult: 1}
	}
	for _, value := range env.ModDB.List(nil, "ExtraAura") {
		extraAura := value.(mod.ExtraAura)
		modList := asModList([]mod.Mod{extraAura.Mod})
		//pob:block:src/Modules/CalcPerform.lua:3363-3369
		if exports != nil && (extraAura.FromAllies || !env.ModDB.Flag(nil, "SelfAurasCannotAffectAllies")) {
			exports.aura["extraAura"].modList = append(exports.aura["extraAura"].modList, extraAura.Mod)
		}

		if !extraAura.OnlyAllies && !(extraAura.FromAllies && env.ModDB.Flag(nil, "AlliesAurasCannotAffectSelf")) {
			inc := env.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			more := env.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			scaled := moddb.NewModList()
			scaled.ScaleAddList(modList, (1+inc/100)*more)
			env.ModDB.AddList(scaled)
			env.ModDB.Multipliers["BuffOnSelf"]++
		}

		if env.Minion != nil && (extraAura.FromAllies || !env.ModDB.Flag(nil, "SelfAurasCannotAffectAllies")) {
			inc := env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			more := env.Minion.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			scaled := moddb.NewModList()
			scaled.ScaleAddList(modList, (1+inc/100)*more)
			env.Minion.ModDB.AddList(scaled)
		}

		//pob:block:src/Modules/CalcPerform.lua:3290-3299
		totemModBlacklist := extraAura.Mod.Name() == "Speed" ||
			extraAura.Mod.Name() == "CritMultiplier" ||
			extraAura.Mod.Name() == "CritChance"
		if env.Player.MainSkill.SkillFlags[SkillFlagTotem] &&
			(extraAura.FromAllies || !env.ModDB.Flag(nil, "SelfAurasCannotAffectAllies")) && !totemModBlacklist {
			totemMod := extraAura.Mod.Clone()
			name := totemMod.Name()
			if strings.Contains(name, "Condition:") {
				name = strings.ReplaceAll(name, "Condition:", "Condition:Totem")
			} else {
				name = "Totem" + name
			}
			env.ModDB.AddMod(totemMod.SetName(name))
		}
	}

	//pob:block:src/Modules/CalcPerform.lua:3382-3394
	if extra := allyBuffs.byType["extraAura"]; extra != nil {
		for _, name := range sortedPartyBuffNames(extra.buffs) {
			modList := extra.buffs[name].modList
			inc := env.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			more := env.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			env.ModDB.ScaleAddList(modList, (1+inc/100)*more)
			if env.Minion != nil {
				inc := env.Minion.ModDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				more := env.Minion.ModDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				env.Minion.ModDB.ScaleAddList(modList, (1+inc/100)*more)
			}
		}
	}

	// Check for modifiers to apply to actors affected by player auras or curses
	for _, value := range env.ModDB.List(nil, "AffectedByAuraMod") {
		for actor := range affectedByAura {
			actor.ModDB.AddMod(value.(mod.AffectedByAuraMod).Mod)
		}
	}
	for _, value := range env.ModDB.List(nil, "AffectedByCurseMod") {
		for actor := range affectedByCurse {
			actor.ModDB.AddMod(value.(mod.AffectedByCurseMod).Mod)
		}
	}

	// Merge keystones again to catch any that were added by buffs
	mergeKeystones(env, env.ModDB)

	//pob:block:src/Modules/CalcPerform.lua:3305-3314
	if env.ModDB.Flag(nil, "DisableWeapons") {
		env.Player.WeaponData1 = unarmedWeaponData(env.Spec.ClassID)
		env.ModDB.Conditions["Unarmed"] = true
		env.ModDB.Conditions["Unencumbered"] = true
	} else if env.WeaponModList1 != nil {
		env.ModDB.AddList(env.WeaponModList1)
	}

	// Process prerequisites for conditionals.
	//pob:block:src/Modules/CalcPerform.lua:3417-3421
	DefenceForConditionals(env.Player)
	if env.Minion != nil {
		DefenceForConditionals(env.Minion.Actor)
	}

	// Process misc buffs/modifiers
	//pob:src/Modules/CalcPerform.lua:3423-3429
	DoActorCharges(env, env.Player)
	DoActorMisc(env, env.Player)
	if env.Minion != nil {
		DoActorCharges(env, env.Minion.Actor)
		DoActorMisc(env, env.Minion.Actor)
	}

	// Calculate maximum and apply the strongest non-damaging ailments.
	//pob:block:src/Modules/CalcPerform.lua:3431-3556
	nonDamagingAilments := []struct {
		ailment   data.Ailment
		condition string
		mods      func(num float64) []mod.Mod
	}{
		{data.AilmentChill, "Chilled", func(num float64) []mod.Mod {
			mods := []mod.Mod{
				mod.NewFloat("ActionSpeed", mod.TypeIncrease, -num).Source("Chill").Tag(mod.Condition("Chilled")),
			}
			switch {
			case env.ModDB.Flag(nil, "ChillEffectIncDamageTaken"):
				mods = append(mods, mod.NewFloat("DamageTaken", mod.TypeIncrease, num).
					Source("Ahuana's Bite").Tag(mod.Condition("Chilled")))
			case env.ModDB.Flag(nil, "ChillEffectIncColdDamageTaken"):
				mods = append(mods, mod.NewFloat("ColdDamageTaken", mod.TypeIncrease, num).
					Source("Chilled by Hits").Tag(mod.Condition("Chilled")))
			case env.ModDB.Flag(nil, "ChillingAreaIncColdDamageTaken"):
				mods = append(mods, mod.NewFloat("ColdDamageTaken", mod.TypeIncrease, num).
					Source("Chilling Area").Tag(mod.Condition("Chilled")))
			case hasBonechill && (hasGuaranteedBonechill || env.EnemyModDB.Sum(mod.TypeBase, nil, "ChillVal") > 0):
				mods = append(mods, mod.NewFloat("ColdDamageTaken", mod.TypeIncrease, num).
					Source("Bonechill").Tag(mod.Condition("Chilled")))
			}
			if env.ModDB.Flag(nil, "ChillEffectLessDamageDealt") {
				mods = append(mods, mod.NewFloat("Damage", mod.TypeMore, -num/2).
					Source("Shaper of Winter").Tag(mod.Condition("Chilled")))
			}
			return mods
		}},
		{data.AilmentShock, "Shocked", func(num float64) []mod.Mod {
			if !env.ModDB.Flag(nil, "ShockCanStack") {
				return []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeIncrease, num).Source("Shock").Tag(mod.Condition("Shocked"))}
			}
			limit := ailmentStacksMax(env.ModDB, "ShockStacksMax")
			env.Player.Output["CurrentShock"] = num * min(env.EnemyModDB.Sum(mod.TypeBase, nil, "Multiplier:ShockStacks"), limit)
			return []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeIncrease, num).Source("Shock").
				Tag(mod.Condition("Shocked"), mod.Multiplier("ShockStacks").Limit(limit))}
		}},
		{data.AilmentScorch, "Scorched", func(num float64) []mod.Mod {
			if !env.ModDB.Flag(nil, "ScorchCanStack") {
				return []mod.Mod{mod.NewFloat("ElementalResist", mod.TypeBase, -num).Source("Scorch").Tag(mod.Condition("Scorched"))}
			}
			limit := ailmentStacksMax(env.ModDB, "ScorchStacksMax")
			env.Player.Output["CurrentScorch"] = num * min(env.EnemyModDB.Sum(mod.TypeBase, nil, "Multiplier:ScorchStacks"), limit)
			return []mod.Mod{mod.NewFloat("ElementalResist", mod.TypeBase, -num).Source("Scorch").
				Tag(mod.Condition("Scorched"), mod.Multiplier("ScorchStacks").Limit(limit))}
		}},
		{data.AilmentBrittle, "Brittle", func(num float64) []mod.Mod {
			return []mod.Mod{mod.NewFloat("SelfCritChance", mod.TypeBase, num).Source("Brittle").Tag(mod.Condition("Brittle"))}
		}},
		{data.AilmentSap, "Sapped", func(num float64) []mod.Mod {
			return []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, -num).Source("Sap").Tag(mod.Condition("Sapped"))}
		}},
	}

	for _, val := range nonDamagingAilments {
		ailment := string(val.ailment)
		ailmentData := data.NonDamagingAilments[val.ailment]

		if env.EnemyModDB.Flag(nil, "Condition:Already"+val.condition) ||
			env.EnemyModDB.Flag(nil, ailment+"Immune", "ElementalAilmentImmune") ||
			env.EnemyModDB.Sum(mod.TypeBase, nil, "Avoid"+ailment, "AvoidAilments", "AvoidElementalAilments") >= 100 {
			continue
		}

		override := float64(0)
		minimum := float64(0)
		for _, value := range env.ModDB.Tabulate(mod.TypeBase, nil, ailment+"Base", ailment+"Override", ailment+"Minimum") {
			srcMod := value.Mod
			effect := srcMod.Value().Float()
			scalesWithSource := srcMod.Name() == ailment+"Base" || srcMod.Name() == ailment+"Minimum"
			if srcMod.Name() == ailment+"Override" {
				env.EnemyModDB.AddMod(mod.NewFlag("Condition:"+val.condition, true).Source(srcMod.GetSource()))
			}
			if scalesWithSource {
				effect *= calclib.Mod(env.ModDB, nil, "Enemy"+ailment+"Effect")
			}
			//pob:src/Modules/CalcPerform.lua:3523
			effect = float64(effect * calclib.Mod(env.EnemyModDB, nil, "Self"+ailment+"Effect"))

			if scalesWithSource {
				env.ModDB.AddMod(mod.NewFloat(ailment+"Override", mod.TypeBase, effect).
					Source(srcMod.GetSource()).
					Flag(srcMod.Flags()).
					KeywordFlag(srcMod.KeywordFlags()).
					Tag(srcMod.Tags()...))
				if srcMod.Name() == ailment+"Minimum" {
					minimum += effect
				}
			}

			override = max(max(override, effect), minimum)
		}

		maxAilment := float64(0)
		ailmentMax := ailment + "Max"
		if override := env.ModDB.Override(nil, ailmentMax); override != nil {
			maxAilment = utils.Or(override, 0)
		} else {
			for _, skill := range env.Player.ActiveSkillList {
				if skill == nil || skill.BaseSkillModList == nil {
					continue
				}
				skillMax := ailmentData.Max + skill.BaseSkillModList.Sum(mod.TypeBase, nil, ailmentMax)
				if skillMax > maxAilment {
					maxAilment = skillMax
				}
			}
		}
		env.Player.Output["Maximum"+ailment] = maxAilment
		precision := math.Pow(10, ailmentData.Precision)
		env.Player.Output["Current"+ailment] = math.Floor(min(max(override,
			env.EnemyModDB.Sum(mod.TypeBase, nil, ailment+"Val")),
			env.Player.Output["Maximum"+ailment])*precision) / precision

		for _, m := range val.mods(env.Player.Output["Current"+ailment]) {
			env.EnemyModDB.AddMod(m)
		}
		env.EnemyModDB.AddMod(mod.NewFlag("Condition:Already"+val.condition, true).Tag(mod.Condition(val.condition)))
	}

	for _, m := range []struct{ ailment, multiplier string }{
		{"Chill", "Multiplier:ChillEffect"},
		{"Shock", "Multiplier:ShockEffect"},
	} {
		current := env.Player.Output["Current"+m.ailment]
		if existing := env.EnemyModDB.Sum(mod.TypeBase, nil, m.multiplier); existing < current {
			env.EnemyModDB.AddMod(mod.NewFloat(m.multiplier, mod.TypeBase, current-existing))
		}
	}

	DoActorCharges(env, env.Enemy)
	DoActorMisc(env, env.Enemy)

	// Apply exposures
	for _, element := range []string{"Fire", "Cold", "Lightning"} {
		if !env.Player.ModDB.Flag(nil, "ElementalEquilibrium") || // if Elemental Equilibrium isn't active we just process Exposure normally
			element == "Fire" && !env.EnemyModDB.Flag(nil, "Condition:HitByFireDamage") ||
			element == "Cold" && !env.EnemyModDB.Flag(nil, "Condition:HitByColdDamage") ||
			element == "Lightning" && !env.EnemyModDB.Flag(nil, "Condition:HitByLightningDamage") {
			Min := math.MaxFloat64
			source := mod.Source("")
			for _, Mod := range env.EnemyModDB.Tabulate("BASE", nil, element+"Exposure") {
				if Mod.Value < Min {
					Min = Mod.Value
					source = Mod.Mod.GetSource()
				}
			}
			if Min != math.MaxFloat64 {
				// Modify the magnitude of all exposures
				for _, Mod := range env.Player.ModDB.Tabulate("BASE", nil, "ExtraExposure", "Extra"+element+"Exposure") {
					Min = Min + Mod.Value
				}
				env.EnemyModDB.AddMod(mod.NewFlag("Condition:Has"+element+"Exposure", true))
				env.EnemyModDB.AddMod(mod.NewFloat(element+"Resist", "BASE", min(Min, utils.Or(env.Player.ModDB.Override(nil, "ExposureMin"), 0))).Source(source))
				env.Player.ModDB.AddMod(mod.NewFlag("Condition:AppliedExposureRecently", true))
			}
		}
	}

	// Handle consecrated ground effects on enemies
	if env.EnemyModDB.Flag(nil, "Condition:OnConsecratedGround") {
		effect := 1 + env.Player.ModDB.Sum(mod.TypeIncrease, nil, "ConsecratedGroundEffect")/100
		env.EnemyModDB.AddMod(mod.NewFloat("DamageTaken", "INC", math.Floor(env.EnemyModDB.Sum(mod.TypeIncrease, nil, "DamageTakenConsecratedGround")*effect)).Source("Consecrated Ground"))
	}

	//pob:block:src/Modules/CalcPerform.lua:3596-3599
	needsAllyLife := !env.skipEHP && env.ModDB.Sum(mod.TypeBase, nil, allyLifeRedirects...) != 0
	if needsAllyLife {
		spectreLimit := env.Player.Output["ActiveSpectreLimit"]
		if override := env.Build.GetNumberOption("multiplierSummonedMinion"); override != 0 {
			spectreLimit = math.Min(spectreLimit, override)
		}
		spectreCount := float64(0)
		firstSpectreLife := float64(0)
		spectrePool := allyLifePoolByKey("spectres")
		spectrePool.source = "Spectres"
		calculatedLifePool := map[string]bool{}

		for _, activeSkill := range env.Player.ActiveSkillList {
			if activeSkill.Minion == nil || activeSkill.SkillFlags[SkillFlagDisable] ||
				activeSkill.SkillTypes[data.SkillTypeMinionsAreUndamagable] {
				continue
			}
			minionLife := float64(0)
			if activeSkill.Minion == env.Minion {
				minionLife = env.Minion.Output["Life"]
			} else {
				prepareAllyMinion(env, activeSkill)
				minionLife = activeSkill.Minion.Output["Life"]
			}
			if minionLife <= 0 {
				continue
			}
			var pools []allyLifePool
			//pob:src/Modules/CalcPerform.lua:3607
			var skillPool allyLifePool
			var hasSkillPool bool
			isSpectre := activeSkill.SkillFlags[SkillFlagSpectre]
			if isSpectre {
				skillPool, hasSkillPool = spectrePool, true
			} else if p, ok := minionLifePoolBySkill[activeSkill.ActiveEffect.GrantedEffect.Name()]; ok {
				skillPool, hasSkillPool = p, true
			}
			if hasSkillPool {
				pools = append(pools, skillPool)
			}
			for _, buff := range activeSkill.BuffList {
				var found bool
				for _, list := range []*moddb.ModList{buff.ModList, buff.UnscalableModList} {
					if list == nil {
						continue
					}
					for _, m := range list.Mods() {
						if m.Name() == "takenFromMinionBeforeYou" {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if found {
					companionshipPool := allyLifePoolByKey("minion")
					pools = append(pools, companionshipPool)
					if hasSkillPool &&
						env.ModDB.Sum(mod.TypeBase, nil, skillPool.redirect) != 0 &&
						env.ModDB.Sum(mod.TypeBase, nil, companionshipPool.redirect) != 0 {
						pools = []allyLifePool{companionshipPool}
						env.ModDB.AddMod(mod.NewFlag("MinionLifeShares"+skillPool.life, true).Source("Companionship"))
					}
					break
				}
			}
			for _, pool := range pools {
				if env.ModDB.Sum(mod.TypeBase, nil, pool.redirect) == 0 || env.ModDB.Override(nil, pool.life) != nil {
					continue
				}
				if pool.key != spectrePool.key {
					if calculatedLifePool[pool.life] {
						continue
					}
					calculatedLifePool[pool.life] = true
				}
				//pob:src/Modules/CalcPerform.lua:3634
				//pob:src/Modules/CalcPerform.lua:3662-3665
				//pob:src/Modules/CalcPerform.lua:3672
				if pool.key == spectrePool.key {
					if spectreCount >= spectreLimit {
						continue
					}
					spectreCount++
					if firstSpectreLife == 0 {
						firstSpectreLife = minionLife
					}
					env.ModDB.AddMod(mod.NewFloat(pool.life, mod.TypeBase, minionLife).Source(mod.Source(pool.source)))
					continue
				}
				count := float64(1)
				if pool.limit != "" {
					if v := env.Player.Output[pool.limit]; v != 0 {
						count = v
					} else {
						count = calclib.Val(activeSkill.SkillModList, pool.limit)
					}
				}
				env.ModDB.AddMod(mod.NewFloat(pool.life, mod.TypeBase, minionLife*count).Source(mod.Source(pool.source)))
			}
		}

		if firstSpectreLife > 0 && spectreCount < spectreLimit {
			env.ModDB.AddMod(mod.NewFloat(spectrePool.life, mod.TypeBase,
				firstSpectreLife*(spectreLimit-spectreCount)).Source(mod.Source(spectrePool.source)))
		}
	}

	if needsAllyLife {
		var normalTotem, vaalTotem *ActiveSkill
		var normalLife, vaalLife float64
		for _, activeSkill := range env.Player.ActiveSkillList {
			if !activeSkill.SkillFlags[SkillFlagTotem] || activeSkill.SkillFlags[SkillFlagDisable] ||
				activeSkill.SkillData.TotemLevel == 0 || activeSkill.SkillTotemId == 0 {
				continue
			}
			life, _ := CalcTotemLife(activeSkill)
			if activeSkill.ActiveEffect.GrantedEffect.Name() == "Vaal Rejuvenation Totem" {
				if vaalTotem == nil || life > vaalLife {
					vaalTotem, vaalLife = activeSkill, life
				}
			} else if normalTotem == nil || life > normalLife {
				normalTotem, normalLife = activeSkill, life
			}
		}
		totemOverride := env.ModDB.Override(nil, "TotalTotemLife")
		totemRedirect := env.ModDB.Sum(mod.TypeBase, nil, "takenFromTotemsBeforeYou")
		nearestIsVaal := vaalTotem != nil && (normalTotem == nil || vaalLife > normalLife) && totemOverride == nil
		if normalTotem != nil && !nearestIsVaal && totemRedirect != 0 && totemOverride == nil {
			env.ModDB.AddMod(mod.NewFloat("TotalTotemLife", mod.TypeBase, normalLife).Source("Totem"))
		}
		if vaalTotem != nil && env.ModDB.Override(nil, "TotalVaalRejuvenationTotemLife") == nil &&
			(env.ModDB.Sum(mod.TypeBase, nil, "takenFromVaalRejuvenationTotemsBeforeYou") != 0 || (nearestIsVaal && totemRedirect != 0)) {
			env.ModDB.AddMod(mod.NewFloat("TotalVaalRejuvenationTotemLife", mod.TypeBase, vaalLife).Source("Vaal Rejuvenation Totem"))
		}
	}

	// Defence/offence calculations
	CalculateDefence(env, env.Player)

	//pob:src/Modules/CalcPerform.lua:3726
	calcsTriggers(env, env.Player)

	performMirageArcher(env)
	performSacredWisps(env)
	performGeneralsCryMarks(env)

	//pob:block:src/Modules/CalcPerform.lua:3727-3729
	if !performSaviourMirage(env) {
		CalculateOffence(env, env.Player, env.Player.MainSkill)
	}

	//pob:block:src/Modules/CalcPerform.lua:3864-3898
	if ms := env.Player.MainSkill; ms != nil && ms.ActiveEffect != nil && ms.ActiveEffect.SrcInstance != nil {
		sml, cfg := ms.SkillModList, ms.SkillCfg
		baseLevel := sml.Sum(mod.TypeBase, cfg, "GemLevel")
		totalItemLevel := sml.Sum(mod.TypeBase, cfg, "GemItemLevel")
		totalSupportLevel := sml.Sum(mod.TypeBase, cfg, "GemSupportLevel")
		env.Player.Output["GemHasLevel"] = 1
		env.Player.Output["GemLevel"] = baseLevel + totalSupportLevel + totalItemLevel
		//pob:block:src/Modules/CalcPerform.lua:3877-3887
		if bd := env.Player.Breakdown; bd != nil {
			bd.Data["GemLevel"] = &Entry{Lines: []string{fmt.Sprintf("%d ^8(level from gem)", luaInt(baseLevel))}}
			if totalSupportLevel > 0 {
				bd.AddLine("GemLevel", fmt.Sprintf("+ %d ^8(level from support)", luaInt(totalSupportLevel)))
			}
			if totalItemLevel > 0 {
				bd.AddLine("GemLevel", fmt.Sprintf("+ %d ^8(level from items)", luaInt(totalItemLevel)))
			}
			bd.AddLine("GemLevel", fmt.Sprintf("= %d", luaInt(env.Player.Output["GemLevel"])))
		}
		baseQuality := sml.Sum(mod.TypeBase, cfg, "GemQuality")
		totalItemQuality := sml.Sum(mod.TypeBase, cfg, "GemItemQuality")
		totalSupportQuality := sml.Sum(mod.TypeBase, cfg, "GemSupportQuality")
		socketQuality := sml.Sum(mod.TypeBase, cfg, "GemSocketQuality")
		env.Player.Output["GemHasQuality"] = 1
		env.Player.Output["GemQuality"] = baseQuality + totalSupportQuality + totalItemQuality + socketQuality
		//pob:block:src/Modules/CalcPerform.lua:3900-3913
		if bd := env.Player.Breakdown; bd != nil {
			bd.Data["GemQuality"] = &Entry{Lines: []string{fmt.Sprintf("%d ^8(quality from gem)", luaInt(baseQuality))}}
			if totalSupportQuality > 0 {
				bd.AddLine("GemQuality", fmt.Sprintf("+ %d ^8(quality from support)", luaInt(totalSupportQuality)))
			}
			if totalItemQuality > 0 {
				bd.AddLine("GemQuality", fmt.Sprintf("+ %d ^8(quality from items)", luaInt(totalItemQuality)))
			}
			if socketQuality > 0 {
				bd.AddLine("GemQuality", fmt.Sprintf("+ %d ^8(quality from socket)", luaInt(socketQuality)))
			}
			bd.AddLine("GemQuality", fmt.Sprintf("= %d", luaInt(env.Player.Output["GemQuality"])))
		}
	}

	if env.Minion != nil {
		CalculateDefence(env, env.Minion.Actor)

		//pob:src/Modules/CalcPerform.lua:3736
		calcsTriggers(env, env.Minion.Actor)

		CalculateOffence(env, env.Minion.Actor, env.Minion.MainSkill)
	}

	//pob:block:src/Modules/CalcPerform.lua:3741-3860
	if exports != nil {
		exports.collectEnemyAndPlayer(env, env.Player.Output)
		env.partyExports = exports
		env.Build.SetExportedBuffs(exports.texts())
	}

	/*
		TODO Cache Data
		uuid := cacheSkillUUID(env.Player.MainSkill)
		if not env.dontCache {
			cacheData(uuid, env)
		}
	*/
}

// Calculate life/mana reservation
//
//pob:src/Modules/CalcPerform.lua:526-560
func doActorLifeManaReservation(actor *Actor, addAura bool) {
	for _, pool := range []string{"Life", "Mana"} {
		pn := poolReservationNames[pool]
		Max := actor.Output[pool]
		reserved := float64(0)
		if Max > 0 {
			lowPerc := actor.ModDB.Sum(mod.TypeBase, nil, pn.lowPercentage)
			reserved = actor.Output[pn.reservedBase] + math.Ceil(Max*actor.Output[pn.reservedPercentIn]/100)
			uncancellableReservation := actor.Output[pn.uncancellableIn]
			actor.Output[pn.reserved] = min(reserved, Max)
			actor.Output[pn.reservedPercent] = min(reserved/Max*100, 100)
			actor.Output[pn.unreserved] = Max - reserved
			actor.Output[pn.unreservedPercent] = (Max - reserved) / Max * 100
			actor.Output[pn.uncancellable] = min(uncancellableReservation, 0)
			actor.Output[pn.cancellable] = 100 - uncancellableReservation
			threshold := data.LowPoolThreshold
			if lowPerc > 0 {
				threshold = lowPerc
			}
			if (Max-reserved)/Max <= threshold {
				actor.ModDB.Conditions[pn.low] = true
			}
		}
		if !addAura {
			continue
		}
		for _, value := range actor.ModDB.List(nil, "GrantReserved"+pool+"AsAura") {
			auraMod := value.(mod.GrantReservedPoolAsAura).Mod.Clone()
			auraMod.Value().SetFloat(math.Floor(auraMod.Value().Float() * min(reserved, Max)))
			actor.ModDB.AddMod(mod.NewList("ExtraAura", mod.ExtraAura{
				Mod: auraMod,
			}))
		}
	}
}

//pob:block:src/Modules/CalcPerform.lua:402-413
func setAttributeRankConditions(actor *Actor) {
	str, dex, intel := actor.Output["Str"], actor.Output["Dex"], actor.Output["Int"]
	actor.ModDB.Conditions["IntHigherThanDex"] = intel > dex
	actor.ModDB.Conditions["DexHigherThanStr"] = dex > str
	actor.ModDB.Conditions["StrHighestAttribute"] = str >= dex && str >= intel
	actor.ModDB.Conditions["IntHighestAttribute"] = intel >= str && intel >= dex
	actor.ModDB.Conditions["DexHighestAttribute"] = dex >= str && dex >= intel
	actor.ModDB.Conditions["IntSingleHighestAttribute"] = intel > str && intel > dex
	actor.ModDB.Conditions["DexSingleHighestAttribute"] = dex > str && dex > intel
}

func setAllOneHandedCondition(actor *Actor, cond string, added **bool, extra bool) {
	*added = utils.Ptr(extra || !actor.ModDB.Conditions[cond])
	actor.ModDB.Conditions[cond] = true
}

//pob:src/Modules/CalcPerform.lua:135-524
func doActorAttribsPoolsConditions(env *Environment, actor *Actor) {
	/*
		local modDB = actor.modDB
		local output = actor.output
		local breakdown = actor.breakdown
		local condList = modDB.conditions
	*/
	// Set conditions
	if (actor.ItemList["Weapon 2"] != nil && actor.ItemList["Weapon 2"].Type == "Shield") || (actor == env.Player && env.AegisModList != nil) {
		actor.ModDB.Conditions["UsingShield"] = true
	} else if actor.ItemList["Weapon 2"] == nil {
		actor.ModDB.Conditions["OffHandIsEmpty"] = true
	}
	if actor.WeaponData1.Type == "None" {
		actor.ModDB.Conditions["Unarmed"] = true
		if actor.ItemList["Weapon 2"] == nil && actor.ItemList["Gloves"] == nil {
			actor.ModDB.Conditions["Unencumbered"] = true
		}
	} else {
		info := data.WeaponTypes[actor.WeaponData1.Type]
		actor.ModDB.Conditions["Using"+info.Flag] = true
		if actor.WeaponData1.CountsAsAll1H {
			varunastra1 := actor.ItemList["Weapon 1"] != nil &&
				strings.Contains(actor.ItemList["Weapon 1"].Name, "Varunastra")
			setAllOneHandedCondition(actor, "UsingAxe", &actor.WeaponData1.AddedUsingAxe, false)
			setAllOneHandedCondition(actor, "UsingSword", &actor.WeaponData1.AddedUsingSword, varunastra1)
			setAllOneHandedCondition(actor, "UsingDagger", &actor.WeaponData1.AddedUsingDagger, false)
			setAllOneHandedCondition(actor, "UsingMace", &actor.WeaponData1.AddedUsingMace, false)
			setAllOneHandedCondition(actor, "UsingClaw", &actor.WeaponData1.AddedUsingClaw, false)
			// GGG stated that a single Varunastra satisfied requirement for wielding two different weapons
			actor.ModDB.Conditions["WieldingDifferentWeaponTypes"] = true
		}
		if info.Melee {
			actor.ModDB.Conditions["UsingMeleeWeapon"] = true
		}
		if info.OneHand {
			actor.ModDB.Conditions["UsingOneHandedWeapon"] = true
		} else {
			actor.ModDB.Conditions["UsingTwoHandedWeapon"] = true
		}
	}
	for _, slotName := range []string{"Helmet", "Body Armour", "Gloves", "Boots"} {
		if actor.ItemList[slotName] != nil {
			actor.ModDB.Conditions["Using"+slotName] = true
		}
	}
	if actor.WeaponData2.Type != "" {
		info := data.WeaponTypes[actor.WeaponData2.Type]
		actor.ModDB.Conditions["Using"+info.Flag] = true
		if actor.WeaponData2.CountsAsAll1H {
			varunastra2 := actor.ItemList["Weapon 2"] != nil &&
				strings.Contains(actor.ItemList["Weapon 2"].Name, "Varunastra")
			setAllOneHandedCondition(actor, "UsingAxe", &actor.WeaponData2.AddedUsingAxe, false)
			setAllOneHandedCondition(actor, "UsingSword", &actor.WeaponData2.AddedUsingSword, varunastra2)
			setAllOneHandedCondition(actor, "UsingDagger", &actor.WeaponData2.AddedUsingDagger, false)
			setAllOneHandedCondition(actor, "UsingMace", &actor.WeaponData2.AddedUsingMace, false)
			setAllOneHandedCondition(actor, "UsingClaw", &actor.WeaponData2.AddedUsingClaw, false)
			// GGG stated that a single Varunastra satisfied requirement for wielding two different weapons
			actor.ModDB.Conditions["WieldingDifferentWeaponTypes"] = true
		}
		if info.Melee {
			actor.ModDB.Conditions["UsingMeleeWeapon"] = true
		}
		if info.OneHand {
			actor.ModDB.Conditions["UsingOneHandedWeapon"] = true
		} else {
			actor.ModDB.Conditions["UsingTwoHandedWeapon"] = true
		}
	}
	if actor.WeaponData1.Type != "" && actor.WeaponData2.Type != "" {
		actor.ModDB.Conditions["DualWielding"] = true
		if (actor.WeaponData1.Type == "Claw" || actor.WeaponData1.CountsAsAll1H) && (actor.WeaponData2.Type == "Claw" || actor.WeaponData2.CountsAsAll1H) {
			actor.ModDB.Conditions["DualWieldingClaws"] = true
		}
		if (actor.WeaponData1.Type == "Dagger" || actor.WeaponData1.CountsAsAll1H) && (actor.WeaponData2.Type == "Dagger" || actor.WeaponData2.CountsAsAll1H) {
			actor.ModDB.Conditions["DualWieldingDaggers"] = true
		}
		if utils.OrS(data.WeaponTypes[actor.WeaponData1.Type].Label, string(actor.WeaponData1.Type)) != utils.OrS(data.WeaponTypes[actor.WeaponData2.Type].Label, string(actor.WeaponData2.Type)) {
			info1 := data.WeaponTypes[actor.WeaponData1.Type]
			info2 := data.WeaponTypes[actor.WeaponData2.Type]
			if info1.OneHand && info2.OneHand {
				actor.ModDB.Conditions["WieldingDifferentWeaponTypes"] = true
			}
		}
	}
	if env.ModeCombat {

		if !actor.MainSkill.SkillData.Triggered && !actor.MainSkill.SkillFlags[SkillFlagTrap] && !actor.MainSkill.SkillFlags[SkillFlagMine] && !actor.MainSkill.SkillFlags[SkillFlagTotem] {
			if actor.MainSkill.SkillFlags[SkillFlagAttack] {
				actor.ModDB.Conditions["AttackedRecently"] = true
			} else if actor.MainSkill.SkillFlags[SkillFlagSpell] {
				actor.ModDB.Conditions["CastSpellRecently"] = true
			}
			if actor.MainSkill.SkillTypes[data.SkillTypeMovement] {
				actor.ModDB.Conditions["UsedMovementSkillRecently"] = true
			}
			if actor.MainSkill.SkillFlags[SkillFlagMinion] && !actor.MainSkill.SkillFlags[SkillFlagPermanentMinion] {
				actor.ModDB.Conditions["UsedMinionSkillRecently"] = true
			}
			if actor.MainSkill.SkillTypes[data.SkillTypeVaal] {
				actor.ModDB.Conditions["UsedVaalSkillRecently"] = true
			}
			if actor.MainSkill.SkillTypes[data.SkillTypeChannel] {
				actor.ModDB.Conditions["Channelling"] = true
			}
		}
		if actor.MainSkill.SkillFlags[SkillFlagHit] && !actor.MainSkill.SkillFlags[SkillFlagTrap] && !actor.MainSkill.SkillFlags[SkillFlagMine] && !actor.MainSkill.SkillFlags[SkillFlagTotem] {
			actor.ModDB.Conditions["HitRecently"] = true
			if actor.MainSkill.SkillFlags[SkillFlagSpell] {
				actor.ModDB.Conditions["HitSpellRecently"] = true
			}
		}
		if actor.MainSkill.SkillFlags[SkillFlagTotem] {
			actor.ModDB.Conditions["HaveTotem"] = true
			actor.ModDB.Conditions["SummonedTotemRecently"] = true
			if actor.MainSkill.SkillFlags[SkillFlagHit] {
				actor.ModDB.Conditions["TotemsHitRecently"] = true
				if actor.MainSkill.SkillFlags[SkillFlagSpell] {
					actor.ModDB.Conditions["TotemsSpellHitRecently"] = true
				}
			}
		}
		if actor.MainSkill.SkillFlags[SkillFlagMine] {
			actor.ModDB.Conditions["DetonatedMinesRecently"] = true
		}
		if actor.MainSkill.SkillFlags[SkillFlagTrap] {
			actor.ModDB.Conditions["TriggeredTrapsRecently"] = true
		}
		if actor.ModDB.Sum(mod.TypeBase, nil, "EnemyScorchChance") > 0 || actor.ModDB.Flag(nil, "CritAlwaysAltAilments") && !actor.ModDB.Flag(env.Player.MainSkill.SkillCfg, "NeverCrit") || actor.ModDB.Flag(nil, "IgniteCanScorch") {
			actor.ModDB.Conditions["CanInflictScorch"] = true
		}
		if actor.ModDB.Sum(mod.TypeBase, nil, "EnemyBrittleChance") > 0 || actor.ModDB.Flag(nil, "CritAlwaysAltAilments") && !actor.ModDB.Flag(env.Player.MainSkill.SkillCfg, "NeverCrit") {
			actor.ModDB.Conditions["CanInflictBrittle"] = true
		}
		if actor.ModDB.Sum(mod.TypeBase, nil, "EnemySapChance") > 0 || actor.ModDB.Flag(nil, "CritAlwaysAltAilments") && !actor.ModDB.Flag(env.Player.MainSkill.SkillCfg, "NeverCrit") {
			actor.ModDB.Conditions["CanInflictSap"] = true
		}
		applyShrineBuffs(actor)
	}
	if env.ModeEffective {
		if env.Player.MainSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "FireExposureChance") > 0 || actor.ModDB.Sum(mod.TypeBase, nil, "FireExposureChance") > 0 {
			actor.ModDB.Conditions["CanApplyFireExposure"] = true
		}
		if env.Player.MainSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "ColdExposureChance") > 0 || actor.ModDB.Sum(mod.TypeBase, nil, "ColdExposureChance") > 0 {
			actor.ModDB.Conditions["CanApplyColdExposure"] = true
		}
		if env.Player.MainSkill.SkillModList.Sum(mod.TypeBase, env.Player.MainSkill.SkillCfg, "LightningExposureChance") > 0 || actor.ModDB.Sum(mod.TypeBase, nil, "LightningExposureChance") > 0 {
			actor.ModDB.Conditions["CanApplyLightningExposure"] = true
		}
	}

	calculateAttributes := func() {
		for p := 1; p <= 2; p++ {
			for _, stat := range []string{"Str", "Dex", "Int"} {
				actor.Output[stat] = math.Max(math.Round(CalcVal(actor.ModDB, stat, nil)), 0)
				if actor.Breakdown != nil {
					actor.Breakdown.Simple(nil, nil, actor.Output[stat], stat)
				}
			}

			stats := []float64{actor.Output["Str"], actor.Output["Dex"], actor.Output["Int"]}
			sort.Float64s(stats)
			actor.Output["LowestAttribute"] = stats[0]
			actor.ModDB.Conditions["TwoHighestAttributesEqual"] = stats[1] == stats[2]

			actor.ModDB.Conditions["DexHigherThanInt"] = actor.Output["Dex"] > actor.Output["Int"]
			actor.ModDB.Conditions["StrHigherThanDex"] = actor.Output["Str"] > actor.Output["Dex"]
			actor.ModDB.Conditions["IntHigherThanStr"] = actor.Output["Int"] > actor.Output["Str"]
			actor.ModDB.Conditions["StrHigherThanInt"] = actor.Output["Str"] > actor.Output["Int"]
			setAttributeRankConditions(actor)
		}
	}
	calculateOmniscience := func() {
		classStats := env.Spec.Class()
		statBase := map[string]float64{
			"Str": float64(classStats.BaseStr),
			"Dex": float64(classStats.BaseDex),
			"Int": float64(classStats.BaseInt),
		}

		for pass := 1; pass <= 2; pass++ {
			if pass != 1 {
				for _, stat := range []string{"Str", "Dex", "Int"} {
					base := statBase[stat]
					actor.Output[stat] = math.Min(math.Round(CalcVal(actor.ModDB, stat, nil)), base)
					if actor.Breakdown != nil {
						actor.Breakdown.Simple(nil, nil, actor.Output[stat], stat)
					}

					actor.ModDB.AddMod(mod.NewFloat("Omni", mod.TypeBase, actor.ModDB.Sum(mod.TypeBase, nil, stat)-base).Source(mod.Source(stat + " conversion Omniscience")))
					actor.ModDB.AddMod(mod.NewFloat("Omni", mod.TypeIncrease, actor.ModDB.Sum(mod.TypeIncrease, nil, stat)).Source("Omniscience"))
					actor.ModDB.AddMod(mod.NewFloat("Omni", mod.TypeMore, actor.ModDB.Sum(mod.TypeMore, nil, stat)).Source("Omniscience"))
				}
			}

			if pass != 2 {
				for _, modType := range []mod.Type{mod.TypeBase, mod.TypeIncrease, mod.TypeMore} {
					reduction := actor.ModDB.Sum(modType, nil, "StrDex") +
						actor.ModDB.Sum(modType, nil, "StrInt") +
						actor.ModDB.Sum(modType, nil, "DexInt") +
						2*actor.ModDB.Sum(modType, nil, "All")
					actor.ModDB.AddMod(mod.NewFloat("Omni", modType, -reduction).Source("Reduction from Double/Triple Dipped attributes to Omniscience"))
				}
			}

			for _, stat := range []string{"Str", "Dex", "Int"} {
				actor.Output[stat] = statBase[stat]
			}

			actor.Output["Omni"] = math.Max(math.Round(CalcVal(actor.ModDB, "Omni", nil)), 0)
			if actor.Breakdown != nil {
				actor.Breakdown.Simple(nil, nil, actor.Output["Omni"], "Omni")
			}

			stats := []float64{actor.Output["Str"], actor.Output["Dex"], actor.Output["Int"]}
			sort.Float64s(stats)
			actor.Output["LowestAttribute"] = stats[0]
			actor.ModDB.Conditions["TwoHighestAttributesEqual"] = stats[1] == stats[2]

			actor.Output["LowestAttribute"] = math.Min(actor.Output["Str"], math.Min(actor.Output["Dex"], actor.Output["Int"]))
			actor.ModDB.Conditions["DexHigherThanInt"] = actor.Output["Dex"] > actor.Output["Int"]
			actor.ModDB.Conditions["StrHigherThanDex"] = actor.Output["Str"] > actor.Output["Dex"]
			actor.ModDB.Conditions["IntHigherThanStr"] = actor.Output["Int"] > actor.Output["Str"]
			actor.ModDB.Conditions["StrHigherThanInt"] = actor.Output["Str"] > actor.Output["Int"]
			setAttributeRankConditions(actor)
		}
	}

	if actor.ModDB.Flag(nil, "Omniscience") {
		calculateOmniscience()
	} else {
		calculateAttributes()
	}

	// Calculate total attributes
	actor.Output["TotalAttr"] = actor.Output["Str"] + actor.Output["Dex"] + actor.Output["Int"]

	// Special case for Devotion
	actor.Output["Devotion"] = actor.ModDB.Sum(mod.TypeBase, nil, "Devotion")

	//pob:src/Modules/CalcPerform.lua:630-658
	if !actor.ModDB.Flag(nil, "NoAttributeBonuses") {
		if !actor.ModDB.Flag(nil, "NoStrengthAttributeBonuses") {
			if !actor.ModDB.Flag(nil, "NoStrBonusToLife") {
				actor.ModDB.AddMod(mod.NewFloat("Life", mod.TypeBase, math.Floor(actor.Output["Str"]/2)).Source("Strength"))
			}
			strDmgBonusRatioOverride := actor.ModDB.Sum(mod.TypeBase, nil, "StrDmgBonusRatioOverride")
			if strDmgBonusRatioOverride > 0 {
				actor.StrDmgBonus = math.Floor((actor.Output["Str"] + actor.ModDB.Sum(mod.TypeBase, nil, "DexIntToMeleeBonus")) * strDmgBonusRatioOverride)
			} else {
				actor.StrDmgBonus = math.Floor((actor.Output["Str"] + actor.ModDB.Sum(mod.TypeBase, nil, "DexIntToMeleeBonus")) / 5)
			}
			actor.ModDB.AddMod(mod.NewFloat("PhysicalDamage", mod.TypeIncrease, actor.StrDmgBonus).Source("Strength").Flag(mod.MFlagMelee))
		}

		if !actor.ModDB.Flag(nil, "NoDexterityAttributeBonuses") {
			accuracyMult := data.AccuracyPerDexBase
			DexAccBonusOverride := actor.ModDB.Override(nil, "DexAccBonusOverride")
			if DexAccBonusOverride != nil {
				accuracyMult = DexAccBonusOverride.Float()
			}

			actor.ModDB.AddMod(mod.NewFloat("Accuracy", mod.TypeBase, actor.Output["Dex"]*accuracyMult).Source("Dexterity"))
			if !actor.ModDB.Flag(nil, "NoDexBonusToEvasion") {
				actor.ModDB.AddMod(mod.NewFloat("Evasion", mod.TypeIncrease, math.Floor(actor.Output["Dex"]/5)).Source("Dexterity"))
			}
		}

		if !actor.ModDB.Flag(nil, "NoIntelligenceAttributeBonuses") {
			if !actor.ModDB.Flag(nil, "NoIntBonusToMana") {
				actor.ModDB.AddMod(mod.NewFloat("Mana", mod.TypeBase, math.Floor(actor.Output["Int"]/2)).Source("Intelligence"))
			}

			if !actor.ModDB.Flag(nil, "NoIntBonusToES") {
				actor.ModDB.AddMod(mod.NewFloat("EnergyShield", mod.TypeIncrease, math.Floor(actor.Output["Int"]/10)).Source("Intelligence"))
			}
		}
	}

	doActorLifeMana(actor)
}

//pob:src/Modules/CalcPerform.lua:68-130
func doActorLifeMana(actor *Actor) {
	//pob:block:src/Modules/CalcPerform.lua:74-77
	lowLifePerc := actor.ModDB.Sum(mod.TypeBase, nil, "LowLifePercentage")
	actor.Output["LowLifePercentage"] = 100 * utils.Ternary(lowLifePerc > 0, lowLifePerc, data.LowPoolThreshold)
	fullLifePerc := actor.ModDB.Sum(mod.TypeBase, nil, "FullLifePercentage")
	actor.Output["FullLifePercentage"] = 100 * utils.Ternary(fullLifePerc > 0, fullLifePerc, float64(1))

	actor.Output["ChaosInoculation"] = utils.Ternary[float64](actor.ModDB.Flag(nil, "ChaosInoculation"), 1, 0)

	// Life/mana pools
	if actor.Output["ChaosInoculation"] > 0 {
		actor.Output["Life"] = 1
		actor.ModDB.Conditions["FullLife"] = true
	} else {
		base := actor.ModDB.Sum(mod.TypeBase, nil, "Life")
		inc := actor.ModDB.Sum(mod.TypeIncrease, nil, "Life")
		more := actor.ModDB.More(nil, "Life")
		conv := actor.ModDB.Sum(mod.TypeBase, nil, "LifeConvertToEnergyShield")
		override := actor.ModDB.Override(nil, "Life")
		if override != nil {
			actor.Output["Life"] = override.Float()
		} else {
			actor.Output["Life"] = max(utils.RoundTo(base*(1+inc/100)*more*(1-conv/100), 0), 1)
		}
		//pob:block:src/Modules/CalcPerform.lua:91-106
		if actor.Breakdown != nil {
			if inc != 0 || more != 1 || conv != 0 || override != nil {
				actor.Breakdown.Data["Life"] = &Entry{Lines: []string{fmt.Sprintf("%.6g ^8(base)", base)}}
				if inc != 0 {
					actor.Breakdown.AddLine("Life", fmt.Sprintf("x %.2f ^8(increased/reduced)", 1+inc/100))
				}
				if more != 1 {
					actor.Breakdown.AddLine("Life", fmt.Sprintf("x %.2f ^8(more/less)", more))
				}
				if override != nil {
					actor.Breakdown.AddLine("Life", fmt.Sprintf("= %.6g ^8(life override)", override.Float()))
				}
				actor.Breakdown.AddLine("Life", fmt.Sprintf("= %.6g", actor.Output["Life"]))
			}
		}
	}
	manaConv := actor.ModDB.Sum(mod.TypeBase, nil, "ManaConvertToArmour")
	actor.Output["Mana"] = utils.RoundTo(calclib.Val(actor.ModDB, "Mana")*(1-manaConv/100), 0)
	base := actor.ModDB.Sum(mod.TypeBase, nil, "Mana")
	inc := actor.ModDB.Sum(mod.TypeIncrease, nil, "Mana")
	more := actor.ModDB.More(nil, "Mana")
	//pob:block:src/Modules/CalcPerform.lua:113-128
	if actor.Breakdown != nil {
		if inc != 0 || more != 1 || manaConv != 0 {
			actor.Breakdown.Data["Mana"] = &Entry{Lines: []string{fmt.Sprintf("%.6g ^8(base)", base)}}
			if inc != 0 {
				actor.Breakdown.AddLine("Mana", fmt.Sprintf("x %.2f ^8(increased/reduced)", 1+inc/100))
			}
			if more != 1 {
				actor.Breakdown.AddLine("Mana", fmt.Sprintf("x %.2f ^8(more/less)", more))
			}
			if manaConv != 0 {
				actor.Breakdown.AddLine("Mana", fmt.Sprintf("x %.2f ^8(converted to Armour)", 1-manaConv/100))
			}
			actor.Breakdown.AddLine("Mana", fmt.Sprintf("= %.6g", actor.Output["Mana"]))
		}
	}
	actor.Output["LowestOfMaximumLifeAndMaximumMana"] = min(actor.Output["Life"], actor.Output["Mana"])
}

//pob:block:src/Modules/ModTools.lua:226-237
func mergeKeystones(env *Environment, modDB interface {
	Tabulate(mod.Type, *moddb.ListCfg, ...string) []moddb.ModResult
	AddMod(mod.Mod)
	AddList(*moddb.ModList)
}) {
	if env.KeystonesAdded == nil {
		env.KeystonesAdded = make(map[string]interface{})
	}
	for _, res := range modDB.Tabulate(mod.TypeList, nil, "Keystone") {
		name, ok := res.Mod.Value().ValueList.(string)
		if !ok {
			continue
		}
		if _, added := env.KeystonesAdded[name]; added {
			continue
		}
		list := keystoneModList(env, env.Spec.TreeVersion, name)
		if list == nil {
			continue
		}
		env.KeystonesAdded[name] = true

		src := res.Mod.GetSource()
		if src != "" && !strings.Contains(strings.ToLower(string(src)), "tree") {
			for _, m := range list.Mods() {
				modDB.AddMod(m.Clone().Source(src))
			}
			continue
		}
		modDB.AddList(list)
	}
}

//pob:src/Modules/CalcPerform.lua:1107-1117
func CalcActionSpeedMod(actor *Actor) float64 {
	minimumActionSpeed := actor.ModDB.Max(nil, "MinimumActionSpeed")
	actionSpeedMod := 1 + (math.Max(-data.TemporalChainsEffectCap, actor.ModDB.Sum(mod.TypeIncrease, nil, "TemporalChainsActionSpeed"))+actor.ModDB.Sum(mod.TypeIncrease, nil, "ActionSpeed"))/100
	actionSpeedMod = math.Max(minimumActionSpeed/100, actionSpeedMod)
	if maximumActionSpeedReduction := actor.ModDB.Max(nil, "MaximumActionSpeedReduction"); maximumActionSpeedReduction > 0 {
		actionSpeedMod = math.Min((100-maximumActionSpeedReduction)/100, actionSpeedMod)
	}
	return actionSpeedMod
}

//pob:src/Modules/CalcPerform.lua:947-1105
func DoActorCharges(env *Environment, actor *Actor) {
	modDB := actor.ModDB

	// Calculate current and maximum charges.
	//pob:block:src/Modules/CalcPerform.lua:951-983
	actor.Output["PowerChargesMin"] = max(modDB.Sum(mod.TypeBase, nil, "PowerChargesMin"), 0)
	actor.Output["PowerChargesMax"] = chargesMax(modDB, "PowerChargesMax", modDB.Sum(mod.TypeBase, nil, "PowerChargesMax"))
	//pob:block:src/Modules/CalcPerform.lua:954
	actor.Output["PowerChargesDuration"] = math.Floor(modDB.Sum(mod.TypeBase, nil, "ChargeDuration") * calclib.Mod(modDB, nil, "PowerChargesDuration", "ChargeDuration"))
	replaceChargesMaxFromFlag(modDB, "MaximumFrenzyChargesIsMaximumPowerCharges", "FrenzyChargesMax", actor.Output["PowerChargesMax"])
	actor.Output["FrenzyChargesMin"] = max(modDB.Sum(mod.TypeBase, nil, "FrenzyChargesMin"), 0)
	actor.Output["FrenzyChargesMax"] = chargesMax(modDB, "FrenzyChargesMax",
		utils.Ternary(modDB.Flag(nil, "MaximumFrenzyChargesIsMaximumPowerCharges"), actor.Output["PowerChargesMax"], modDB.Sum(mod.TypeBase, nil, "FrenzyChargesMax")))
	//pob:block:src/Modules/CalcPerform.lua:961
	actor.Output["FrenzyChargesDuration"] = math.Floor(modDB.Sum(mod.TypeBase, nil, "ChargeDuration") * calclib.Mod(modDB, nil, "FrenzyChargesDuration", "ChargeDuration"))
	replaceChargesMaxFromFlag(modDB, "MaximumEnduranceChargesIsMaximumFrenzyCharges", "EnduranceChargesMax", actor.Output["FrenzyChargesMax"])
	actor.Output["EnduranceChargesMin"] = max(modDB.Sum(mod.TypeBase, nil, "EnduranceChargesMin"), 0)
	actor.Output["EnduranceChargesMax"] = chargesMax(modDB, "EnduranceChargesMax",
		utils.Ternary(modDB.Flag(nil, "MaximumEnduranceChargesIsMaximumFrenzyCharges"), actor.Output["FrenzyChargesMax"], modDB.Sum(mod.TypeBase, nil, "EnduranceChargesMax")))
	//pob:block:src/Modules/CalcPerform.lua:968
	actor.Output["EnduranceChargesDuration"] = math.Floor(modDB.Sum(mod.TypeBase, nil, "ChargeDuration") * calclib.Mod(modDB, nil, "EnduranceChargesDuration", "ChargeDuration"))
	actor.Output["SiphoningChargesMax"] = max(modDB.Sum(mod.TypeBase, nil, "SiphoningChargesMax"), 0)
	actor.Output["ChallengerChargesMax"] = max(modDB.Sum(mod.TypeBase, nil, "ChallengerChargesMax"), 0)
	actor.Output["BlitzChargesMax"] = max(modDB.Sum(mod.TypeBase, nil, "BlitzChargesMax"), 0)
	actor.Output["InspirationChargesMax"] = max(modDB.Sum(mod.TypeBase, nil, "InspirationChargesMax"), 0)
	actor.Output["CrabBarriersMax"] = max(modDB.Sum(mod.TypeBase, nil, "CrabBarriersMax"), 0)
	actor.Output["BrutalChargesMin"] = max(utils.Ternary(modDB.Flag(nil, "MinimumEnduranceChargesEqualsMinimumBrutalCharges"),
		utils.Ternary(modDB.Flag(nil, "MinimumEnduranceChargesIsMaximumEnduranceCharges"), actor.Output["EnduranceChargesMax"], actor.Output["EnduranceChargesMin"]), 0), 0)
	actor.Output["BrutalChargesMax"] = max(utils.Ternary(modDB.Flag(nil, "MaximumEnduranceChargesEqualsMaximumBrutalCharges"), actor.Output["EnduranceChargesMax"], 0), 0)
	//pob:block:src/Modules/CalcPerform.lua:976
	actor.Output["BrineChargesMax"] = max(actor.Output["EnduranceChargesMax"], 0)
	actor.Output["AbsorptionChargesMin"] = max(utils.Ternary(modDB.Flag(nil, "MinimumPowerChargesEqualsMinimumAbsorptionCharges"),
		utils.Ternary(modDB.Flag(nil, "MinimumPowerChargesIsMaximumPowerCharges"), actor.Output["PowerChargesMax"], actor.Output["PowerChargesMin"]), 0), 0)
	actor.Output["AbsorptionChargesMax"] = max(utils.Ternary(modDB.Flag(nil, "MaximumPowerChargesEqualsMaximumAbsorptionCharges"), actor.Output["PowerChargesMax"], 0), 0)
	actor.Output["AfflictionChargesMin"] = max(utils.Ternary(modDB.Flag(nil, "MinimumFrenzyChargesEqualsMinimumAfflictionCharges"),
		utils.Ternary(modDB.Flag(nil, "MinimumFrenzyChargesIsMaximumFrenzyCharges"), actor.Output["FrenzyChargesMax"], actor.Output["FrenzyChargesMin"]), 0), 0)
	actor.Output["AfflictionChargesMax"] = max(utils.Ternary(modDB.Flag(nil, "MaximumFrenzyChargesEqualsMaximumAfflictionCharges"), actor.Output["FrenzyChargesMax"], 0), 0)
	actor.Output["BloodChargesMax"] = max(modDB.Sum(mod.TypeBase, nil, "BloodChargesMax"), 0)
	actor.Output["SpiritChargesMax"] = max(modDB.Sum(mod.TypeBase, nil, "SpiritChargesMax"), 0)
	actor.Output["SpiritInfusionsMax"] = modDB.Sum(mod.TypeBase, nil, "SpiritInfusionsMax") +
		utils.Ternary(modDB.Flag(nil, "Condition:CanGainSpiritInfusion"), 10.0, 0.0)

	// Initialize Charges
	actor.Output["PowerCharges"] = 0
	actor.Output["FrenzyCharges"] = 0
	actor.Output["EnduranceCharges"] = 0
	actor.Output["SiphoningCharges"] = 0
	actor.Output["ChallengerCharges"] = 0
	actor.Output["BlitzCharges"] = 0
	actor.Output["InspirationCharges"] = 0
	actor.Output["GhostShrouds"] = 0
	actor.Output["BrutalCharges"] = 0
	actor.Output["BrineCharges"] = 0
	actor.Output["AbsorptionCharges"] = 0
	actor.Output["AfflictionCharges"] = 0
	actor.Output["BloodCharges"] = 0
	actor.Output["SpiritCharges"] = 0
	actor.Output["SpiritInfusions"] = 0

	// Conditionally over-write Charge values
	if modDB.Flag(nil, "UsePowerCharges") {
		actor.Output["PowerCharges"] = utils.Or(modDB.Override(nil, "PowerCharges"), actor.Output["PowerChargesMax"])
	}
	if modDB.Flag(nil, "PowerChargesConvertToAbsorptionCharges") {
		// we max with possible Power Charge Override from Config since Absorption Charges won't have their own config entry
		// and are converted from Power Charges
		actor.Output["AbsorptionCharges"] = max(actor.Output["PowerCharges"], min(actor.Output["AbsorptionChargesMax"], actor.Output["AbsorptionChargesMin"]))
		actor.Output["PowerCharges"] = 0
	} else {
		actor.Output["PowerCharges"] = max(actor.Output["PowerCharges"], min(actor.Output["PowerChargesMax"], actor.Output["PowerChargesMin"]))
	}
	actor.Output["RemovablePowerCharges"] = max(actor.Output["PowerCharges"]-actor.Output["PowerChargesMin"], 0)
	if modDB.Flag(nil, "UseFrenzyCharges") {
		actor.Output["FrenzyCharges"] = utils.Or(modDB.Override(nil, "FrenzyCharges"), actor.Output["FrenzyChargesMax"])
	}
	if modDB.Flag(nil, "FrenzyChargesConvertToAfflictionCharges") {
		// we max with possible Power Charge Override from Config since Absorption Charges won't have their own config entry
		// and are converted from Power Charges
		actor.Output["AfflictionCharges"] = max(actor.Output["FrenzyCharges"], min(actor.Output["AfflictionChargesMax"], actor.Output["AfflictionChargesMin"]))
		actor.Output["FrenzyCharges"] = 0
	} else {
		actor.Output["FrenzyCharges"] = max(actor.Output["FrenzyCharges"], min(actor.Output["FrenzyChargesMax"], actor.Output["FrenzyChargesMin"]))
	}
	actor.Output["RemovableFrenzyCharges"] = max(actor.Output["FrenzyCharges"]-actor.Output["FrenzyChargesMin"], 0)
	if modDB.Flag(nil, "UseEnduranceCharges") {
		actor.Output["EnduranceCharges"] = utils.Or(modDB.Override(nil, "EnduranceCharges"), actor.Output["EnduranceChargesMax"])
		//pob:block:src/Modules/CalcPerform.lua:1038-1040
		if modDB.Flag(nil, "CanGainBrineCharges") {
			actor.Output["BrineCharges"] = actor.Output["BrineChargesMax"]
		}
	}
	if modDB.Flag(nil, "EnduranceChargesConvertToBrutalCharges") {
		// we max with possible Endurance Charge Override from Config since Brutal Charges won't have their own config entry
		// and are converted from Endurance Charges
		actor.Output["BrutalCharges"] = max(actor.Output["EnduranceCharges"], min(actor.Output["BrutalChargesMax"], actor.Output["BrutalChargesMin"]))
		actor.Output["EnduranceCharges"] = 0
	} else {
		actor.Output["EnduranceCharges"] = max(actor.Output["EnduranceCharges"], min(actor.Output["EnduranceChargesMax"], actor.Output["EnduranceChargesMin"]))
	}
	actor.Output["RemovableEnduranceCharges"] = max(actor.Output["EnduranceCharges"]-actor.Output["EnduranceChargesMin"], 0)
	if modDB.Flag(nil, "UseSiphoningCharges") {
		actor.Output["SiphoningCharges"] = utils.Or(modDB.Override(nil, "SiphoningCharges"), actor.Output["SiphoningChargesMax"])
	}
	if modDB.Flag(nil, "UseChallengerCharges") {
		actor.Output["ChallengerCharges"] = utils.Or(modDB.Override(nil, "ChallengerCharges"), actor.Output["ChallengerChargesMax"])
	}
	if modDB.Flag(nil, "UseBlitzCharges") {
		actor.Output["BlitzCharges"] = utils.Or(modDB.Override(nil, "BlitzCharges"), actor.Output["BlitzChargesMax"])
	}
	if actor == env.Player {
		actor.Output["InspirationCharges"] = utils.Or(modDB.Override(nil, "InspirationCharges"), actor.Output["InspirationChargesMax"])
	}
	if modDB.Flag(nil, "UseGhostShrouds") {
		actor.Output["GhostShrouds"] = utils.Or(modDB.Override(nil, "GhostShrouds"), 3)
	}
	actor.Output["BloodCharges"] = min(utils.Or(modDB.Override(nil, "BloodCharges"), actor.Output["BloodChargesMax"]), actor.Output["BloodChargesMax"])
	actor.Output["SpiritCharges"] = min(utils.Or(modDB.Override(nil, "SpiritCharges"), 0), actor.Output["SpiritChargesMax"])
	actor.Output["SpiritInfusions"] = min(utils.Or(modDB.Override(nil, "SpiritInfusion"), 0), actor.Output["SpiritInfusionsMax"])

	actor.Output["CrabBarriers"] = min(utils.Or(modDB.Override(nil, "CrabBarriers"), actor.Output["CrabBarriersMax"]), actor.Output["CrabBarriersMax"])
	//pob:block:src/Modules/CalcPerform.lua:1072-1080
	if modDB.Flag(nil, "HaveMaximumPowerCharges") {
		actor.Output["PowerCharges"] = actor.Output["PowerChargesMax"]
	}
	if modDB.Flag(nil, "HaveMaximumFrenzyCharges") {
		actor.Output["FrenzyCharges"] = actor.Output["FrenzyChargesMax"]
	}
	if modDB.Flag(nil, "HaveMaximumEnduranceCharges") {
		actor.Output["EnduranceCharges"] = actor.Output["EnduranceChargesMax"]
	}
	actor.Output["TotalCharges"] = actor.Output["PowerCharges"] + actor.Output["FrenzyCharges"] + actor.Output["EnduranceCharges"]
	actor.Output["RemovableTotalCharges"] = actor.Output["RemovableEnduranceCharges"] + actor.Output["RemovableFrenzyCharges"] + actor.Output["RemovablePowerCharges"]
	modDB.Multipliers["PowerCharge"] = actor.Output["PowerCharges"]
	modDB.Multipliers["PowerChargeMax"] = actor.Output["PowerChargesMax"]
	modDB.Multipliers["RemovablePowerCharge"] = actor.Output["RemovablePowerCharges"]
	modDB.Multipliers["FrenzyCharge"] = actor.Output["FrenzyCharges"]
	modDB.Multipliers["RemovableFrenzyCharge"] = actor.Output["RemovableFrenzyCharges"]
	modDB.Multipliers["EnduranceCharge"] = actor.Output["EnduranceCharges"]
	modDB.Multipliers["RemovableEnduranceCharge"] = actor.Output["RemovableEnduranceCharges"]
	modDB.Multipliers["TotalCharges"] = actor.Output["TotalCharges"]
	modDB.Multipliers["RemovableTotalCharges"] = actor.Output["RemovableTotalCharges"]
	modDB.Multipliers["SiphoningCharge"] = actor.Output["SiphoningCharges"]
	modDB.Multipliers["ChallengerCharge"] = actor.Output["ChallengerCharges"]
	modDB.Multipliers["BlitzCharge"] = actor.Output["BlitzCharges"]
	modDB.Multipliers["InspirationCharge"] = actor.Output["InspirationCharges"]
	modDB.Multipliers["GhostShroud"] = actor.Output["GhostShrouds"]
	modDB.Multipliers["CrabBarrier"] = actor.Output["CrabBarriers"]
	modDB.Multipliers["BrutalCharge"] = actor.Output["BrutalCharges"]
	modDB.Multipliers["BrineCharge"] = actor.Output["BrineCharges"]
	modDB.Multipliers["AbsorptionCharge"] = actor.Output["AbsorptionCharges"]
	modDB.Multipliers["AfflictionCharge"] = actor.Output["AfflictionCharges"]
	modDB.Multipliers["BloodCharge"] = actor.Output["BloodCharges"]
	modDB.Multipliers["SpiritCharge"] = actor.Output["SpiritCharges"]
	modDB.Multipliers["SpiritInfusion"] = actor.Output["SpiritInfusions"]
}

//pob:src/Modules/CalcPerform.lua:612-944
func DoActorMisc(env *Environment, actor *Actor) {
	modDB := actor.ModDB
	enemyDB := actor.Enemy.ModDB

	// Add misc buffs/debuffs
	if env.ModeCombat {
		if env.Player.MainSkill.BaseSkillModList.Flag(nil, "Cruelty") {
			modDB.Multipliers["Cruelty"] = utils.Or(modDB.Override(nil, "Cruelty"), 40)
		}
		alliedFortify := float64(0)
		if modDB.Flag(nil, "YourFortifyEqualToParent") && actor.Parent != nil {
			alliedFortify = actor.Parent.Output["FortificationStacks"]
		}
		if modDB.Sum(mod.TypeBase, nil, "MinimumFortification") > 0 || alliedFortify > 0 {
			modDB.Conditions["Fortified"] = true
		}
		//pob:block:src/Modules/CalcPerform.lua:632-655
		if modDB.Flag(nil, "Fortified") || modDB.Sum(mod.TypeBase, nil, "Multiplier:Fortification") > 0 {
			var skillModList moddb.ModStoreFuncs = actor.ModDB
			var skillCfg *moddb.ListCfg
			if actor.MainSkill != nil {
				skillModList = actor.MainSkill.SkillModList
				skillCfg = actor.MainSkill.SkillCfg
			}
			maxStacks := max(utils.Or(modDB.Override(nil, "MaximumFortification"), modDB.Sum(mod.TypeBase, skillCfg, "MaximumFortification")), alliedFortify)
			minStacks := modDB.Sum(mod.TypeBase, nil, "MinimumFortification")
			if modDB.Flag(nil, "Condition:HaveMaxFortification") {
				minStacks = maxStacks
			}
			minStacks = min(minStacks, maxStacks)
			stacks := maxStacks
			if minStacks > 0 {
				stacks = minStacks
			}
			stacks = min(utils.Or(modDB.Override(nil, "FortificationStacks"), stacks), maxStacks)
			increasedDuration := skillModList.Sum(mod.TypeIncrease, nil, "FortifyDuration")

			actor.Output["MaximumFortification"] = maxStacks
			actor.Output["MinimumFortification"] = minStacks
			actor.Output["RemovableFortification"] = min(maxStacks-minStacks, utils.Or(modDB.Override(nil, "FortificationStacks"), maxStacks)-minStacks)
			actor.Output["FortificationStacks"] = stacks
			actor.Output["FortificationStacksOver20"] = min(max(0, stacks-20), maxStacks-20)
			actor.Output["FortifyDuration"] = utils.Or(skillModList.Override(nil, "FortifyDuration"), data.FortifyBaseDuration) * (1 + increasedDuration/100)
			actor.Output["FortificationEffect"] = 0
			if !modDB.Flag(nil, "Condition:NoFortificationMitigation") {
				actor.Output["FortificationEffect"] = stacks
				modDB.AddMod(mod.NewFloat("DamageTakenWhenHit", mod.TypeMore, -stacks).Source("Fortification"))
			}
			if stacks >= maxStacks {
				modDB.AddMod(mod.NewFlag("Condition:HaveMaximumFortification", true))
			}
			modDB.Multipliers["BuffOnSelf"] = (modDB.Multipliers["BuffOnSelf"]) + 1
		}

		if modDB.Flag(nil, "Onslaught") {
			onslaughtFromFlask := false
			flaskEffectInc := float64(-100)
			for _, item := range env.Flasks {
				if item == nil || !strings.Contains(item.BaseName, "Silver Flask") {
					continue
				}
				onslaughtFromFlask = true
				curFlaskEffectInc := item.FlaskData["effectInc"] + modDB.Sum(mod.TypeIncrease, playerActorCfg, "FlaskEffect")
				if item.Rarity == RarityMagic {
					curFlaskEffectInc += modDB.Sum(mod.TypeIncrease, playerActorCfg, "MagicUtilityFlaskEffect")
				}
				if flaskEffectInc < curFlaskEffectInc/100 {
					flaskEffectInc = curFlaskEffectInc / 100
				}
			}
			onslaughtEffectInc := modDB.Sum(mod.TypeIncrease, nil, "OnslaughtEffect", "BuffEffectOnSelf") / 100
			effect := math.Floor(20 * (1 + onslaughtEffectInc))
			if onslaughtFromFlask {
				effect = math.Floor(20 * (1 + flaskEffectInc + onslaughtEffectInc))
			}
			modDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, effect).Source("Onslaught").Flag(mod.MFlagAttack))
			modDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, effect).Source("Onslaught").Flag(mod.MFlagCast))
			modDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, effect).Source("Onslaught"))
		}

		//pob:block:src/Modules/CalcPerform.lua:686-704
		if modDB.Conditions["AffectedByArcaneSurge"] || modDB.Flag(nil, "Condition:ArcaneSurge") {
			modDB.Conditions["AffectedByArcaneSurge"] = true
			effect := 1 + modDB.Sum(mod.TypeIncrease, nil, "ArcaneSurgeEffect", "BuffEffectOnSelf")/100
			effect += modDB.Sum(mod.TypeIncrease, playerActorCfg, "FlaskEffect") / 100 *
				modDB.Sum(mod.TypeBase, nil, "FlaskEffectToArcaneSurgeEffect") / 100

			manaRegen := utils.OrF(modDB.Max(nil, "ArcaneSurgeManaRegen"), 30)
			modDB.AddMod(mod.NewFloat("ManaRegen", mod.TypeIncrease, manaRegen*effect).Source("Arcane Surge"))

			castSpeed := utils.OrF(modDB.Max(nil, "ArcaneSurgeCastSpeed"), 20) * effect
			modDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, castSpeed).Source("Arcane Surge").Flag(mod.MFlagCast))
			if modDB.Flag(nil, "ArcaneSurgeCastSpeedToMovementSpeed") {
				modDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, castSpeed).Source("Arcane Surge"))
			}

			if damage := modDB.Max(nil, "ArcaneSurgeDamage"); damage != 0 {
				modDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, damage*effect).Source("Arcane Surge").Flag(mod.MFlagSpell))
			}
			if lifeRegen := modDB.Sum(mod.TypeBase, nil, "ArcaneSurgeAlsoLifeRegen"); lifeRegen > 0 {
				modDB.AddMod(mod.NewFloat("LifeRegen", mod.TypeIncrease, lifeRegen*effect).Source("Arcane Surge"))
			}
		}

		if modDB.Flag(nil, "Fanaticism") && actor.MainSkill != nil && actor.MainSkill.SkillFlags[SkillFlagSelfCast] {
			effect := math.Floor(75 * (1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(mod.NewFloat("Speed", mod.TypeMore, effect).Source("Fanaticism").Flag(mod.MFlagCast))
			modDB.AddMod(mod.NewFloat("Cost", mod.TypeMore, -effect).Source("Fanaticism").Flag(mod.MFlagCast))
			modDB.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeIncrease, effect).Source("Fanaticism").Flag(mod.MFlagCast))
		}
		//pob:src/Modules/CalcPerform.lua:711-719
		if modDB.Flag(nil, "Condition:CanGainSpiritInfusion") {
			globalEffect := func() *mod.GlobalEffectTag {
				return mod.GlobalEffect("Buff").Name("Spirit Infusion").Unscalable(true)
			}
			modDB.AddMod(mod.NewFloat("EnergyShieldRechargeFaster", mod.TypeIncrease, 15).Source("Spirit Infusion").
				Tag(mod.Multiplier("SpiritInfusion"), globalEffect()))
			modDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, 5).Source("Spirit Infusion").KeywordFlag(mod.KeywordFlagSpell).
				Tag(mod.SkillType(string(data.SkillTypeChannel)), mod.Multiplier("SpiritInfusion"), globalEffect()))
			modDB.AddMod(mod.NewFloat("Cost", mod.TypeMore, 10).Source("Spirit Infusion").KeywordFlag(mod.KeywordFlagSpell).
				Tag(mod.SkillType(string(data.SkillTypeChannel)), mod.Multiplier("SpiritInfusion"), globalEffect()))
		}
		//pob:src/Modules/CalcPerform.lua:720-724
		if modDB.Flag(nil, "UnholyMight") {
			effect := 1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100
			modDB.AddMod(mod.NewFloat("PhysicalDamageConvertToChaos", mod.TypeBase, math.Floor(100*effect)).Source("Unholy Might"))
			modDB.AddMod(mod.NewFlag("Condition:CanWither", true).Source("Unholy Might"))
		}
		if modDB.Flag(nil, "ShepherdOfSouls") {
			modDB.AddMod(mod.NewFloat("SoulCost", mod.TypeMore, -80).Source("Shepherd of Souls").
				Tag(mod.SkillType(string(data.SkillTypeVaal)), mod.SkillType(string(data.SkillTypeAura)).Neg(true)))
			modDB.AddMod(mod.NewFloat("SoulCost", mod.TypeIncrease, 100).Source("Shepherd of Souls").
				Tag(mod.SkillType(string(data.SkillTypeVaal)), mod.SkillType(string(data.SkillTypeAura)).Neg(true), mod.Multiplier("VaalSkillsUsedInPast8Seconds")))
		}
		if modDB.Flag(nil, "ChaoticMight") {
			effect := math.Floor(30 * (1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(mod.NewFloat("PhysicalDamageGainAsChaos", mod.TypeBase, effect).Source("Chaotic Might"))
		}
		if modDB.Flag(nil, "Tailwind") {
			effect := math.Floor(8 * (1 + modDB.Sum(mod.TypeIncrease, nil, "TailwindEffectOnSelf", "BuffEffectOnSelf")/100))
			modDB.AddMod(mod.NewFloat("ActionSpeed", mod.TypeIncrease, effect).Source("Tailwind"))
		}
		//pob:block:src/Modules/CalcPerform.lua:737-739
		if modDB.Flag(nil, "Condition:TotemTailwind") {
			modDB.AddMod(mod.NewFloat("TotemActionSpeed", mod.TypeIncrease, 8).Source("Tailwind"))
		}
		if modDB.Flag(nil, "Adrenaline") {
			effectMod := 1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100
			modDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, math.Floor(100*effectMod)).Source("Adrenaline"))
			modDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, math.Floor(25*effectMod)).Source("Adrenaline").Flag(mod.MFlagAttack))
			modDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, math.Floor(25*effectMod)).Source("Adrenaline").Flag(mod.MFlagCast))
			modDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, math.Floor(25*effectMod)).Source("Adrenaline"))
			modDB.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, math.Floor(10*effectMod)).Source("Adrenaline"))
		}
		if modDB.Flag(nil, "Condition:WildSavagery") && modDB.Flag(nil, "WildSavagery") {
			modDB.AddMod(mod.NewFloat("PhysicalDamage", mod.TypeIncrease, 100).Source("Wild Savagery"))
			modDB.AddMod(mod.NewFloat("ActionSpeed", mod.TypeIncrease, 10).Source("Wild Savagery"))
			modDB.AddMod(mod.NewFlag("IgnoreEnemyPhysicalDamageReduction", true).Source("Wild Savagery"))
			modDB.AddMod(mod.NewFlag("StunImmune", true).Source("Wild Savagery"))
		}
		if modDB.Flag(nil, "Convergence") {
			effect := math.Floor(30 * (1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf")/100))
			modDB.AddMod(mod.NewFloat("ElementalDamage", mod.TypeMore, effect).Source("Convergence"))
		}
		if modDB.Flag(nil, "HerEmbrace") {
			modDB.Conditions["HerEmbrace"] = true
			modDB.AddMod(mod.NewFloat("AvoidStun", mod.TypeBase, 100).Source("Her Embrace"))
			modDB.AddMod(mod.NewFloat("PhysicalDamageGainAsFire", mod.TypeBase, 123).Source("Her Embrace").Flag(mod.MFlagSword))
			modDB.AddMod(mod.NewFloat("AvoidFreeze", mod.TypeBase, 100).Source("Her Embrace"))
			modDB.AddMod(mod.NewFloat("AvoidChill", mod.TypeBase, 100).Source("Her Embrace"))
			modDB.AddMod(mod.NewFloat("AvoidIgnite", mod.TypeBase, 100).Source("Her Embrace"))
			modDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 20).Source("Her Embrace").Flag(mod.MFlagAttack))
			modDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 20).Source("Her Embrace").Flag(mod.MFlagCast))
			modDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, 20).Source("Her Embrace"))
		}
		//pob:block:src/Modules/CalcPerform.lua:770-775
		if modDB.Flag(nil, "Condition:OnConsecratedGround") {
			effect := 1 + modDB.Sum(mod.TypeIncrease, nil, "ConsecratedGroundEffect")/100
			modDB.AddMod(mod.NewFloat("LifeRegenPercent", mod.TypeBase, 5*effect).Source("Consecrated Ground"))
			modDB.AddMod(mod.NewFloat("CurseEffectOnSelf", mod.TypeIncrease, -50*effect).Source("Consecrated Ground"))
			modDB.AddMod(mod.NewFloat("Accuracy", mod.TypeIncrease,
				math.Floor(modDB.Sum(mod.TypeIncrease, nil, "ConsecratedGroundAlsoAccuracy")*effect)).
				Source("Consecrated Ground"))
		}
		if modDB.Flag(nil, "Condition:PhantasmalMight") {
			modDB.Multipliers["BuffOnSelf"] = (modDB.Multipliers["BuffOnSelf"]) + (utils.OrDefault(actor.Output["ActivePhantasmLimit"], 1)) - 1 // slight hack to not double count the initial buff
		}
		if modDB.Flag(nil, "Elusive") {
			maxSkillInc := modDB.Max(&moddb.ListCfg{
				Source: utils.Ptr(mod.SourceSkill),
			}, "ElusiveEffect")

			inc := modDB.Sum(mod.TypeIncrease, nil, "ElusiveEffect", "BuffEffectOnSelf")
			if actor.MainSkill.SkillModList.Flag(nil, "SupportedByNightblade") {
				inc = inc + modDB.Sum(mod.TypeIncrease, nil, "NightbladeSupportedElusiveEffect")
			}
			inc = inc + maxSkillInc
			elusiveEffectMod := float64((1 + inc/100) * modDB.More(nil, "ElusiveEffect", "BuffEffectOnSelf") * 100)

			elusiveEffectMinThreshold := utils.Or(modDB.Override(nil, "ElusiveEffectMinThreshold"), 0)
			elusiveEffectIncreaseDuration := modDB.Sum(mod.TypeBase, nil, "ElusiveEffectIncreaseDuration")
			peakElusiveEffect := elusiveEffectMod
			if elusiveEffectIncreaseDuration > 0 {
				elusiveEffectChangeRate := 20 / (1 + modDB.Sum(mod.TypeIncrease, nil, "ElusiveEffectLossSlower")/100)
				peakElusiveEffect = elusiveEffectMod + float64(elusiveEffectChangeRate*elusiveEffectIncreaseDuration)
				elusiveEffectDecreaseDuration := (peakElusiveEffect - elusiveEffectMinThreshold) / elusiveEffectChangeRate
				totalElusiveEffectDuration := elusiveEffectIncreaseDuration + elusiveEffectDecreaseDuration
				averageIncreaseEffect := (elusiveEffectMod + peakElusiveEffect) / 2
				averageDecreaseEffect := (peakElusiveEffect + elusiveEffectMinThreshold) / 2
				//pob:src/Modules/CalcPerform.lua:797
				actor.Output["ElusiveEffectMod"] = (float64(averageIncreaseEffect*elusiveEffectIncreaseDuration) +
					float64(averageDecreaseEffect*elusiveEffectDecreaseDuration)) / totalElusiveEffectDuration
			} else {
				//pob:src/Modules/CalcPerform.lua:799
				actor.Output["ElusiveEffectMod"] = (elusiveEffectMod + elusiveEffectMinThreshold) / 2
			}
			// if we want the max skill to not be noted as its own breakdown table entry, comment out below
			modDB.AddMod(mod.NewFloat("ElusiveEffect", mod.TypeIncrease, maxSkillInc).Source("Max Skill Effect"))
			if elusiveOverride := modDB.Override(nil, "ElusiveEffect"); elusiveOverride != nil {
				actor.Output["ElusiveEffectMod"] = min(elusiveOverride.Float(), peakElusiveEffect)
			}
			effect := actor.Output["ElusiveEffectMod"] / 100
			modDB.Conditions["Elusive"] = true
			modDB.AddMod(mod.NewFloat("AvoidAllDamageFromHitsChance", mod.TypeBase, math.Floor(15*effect)).Source("Elusive"))
			modDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, math.Floor(30*effect)).Source("Elusive"))
		}
		WitherEffectStack := modDB.Max(nil, "WitherEffectStack")
		if WitherEffectStack > 0 {
			modDB.AddMod(mod.NewFlag("Condition:CanWither", true).Source("Config"))
			enemyDB.AddMod(mod.NewFloat("ChaosDamageTaken", mod.TypeIncrease, WitherEffectStack).Source("Withered").Tag(mod.Multiplier("WitheredStack").Limit(15)))
		}
		if modDB.Flag(nil, "Condition:CanBeWithered") {
			effect := 6 * (100 + modDB.Sum(mod.TypeIncrease, nil, "WitherEffectOnSelf")) / 100 * modDB.More(nil, "WitherEffectOnSelf")
			modDB.AddMod(mod.NewFloat("ChaosDamageTaken", mod.TypeIncrease, effect).Source("Withered").Tag(mod.Multiplier("WitheredStack").Limit(15)))
		}
		if modDB.Flag(nil, "Excommunicated") {
			modDB.AddMod(mod.NewFloat("ChaosDamage", mod.TypeMore, -100).Source("Excommunicated"))
		}
		if modDB.Flag(nil, "Blind") && !modDB.Flag(nil, "CannotBeBlinded") {
			if !modDB.Flag(nil, "UnaffectedByBlind") {
				effect := 1 + modDB.Sum(mod.TypeIncrease, nil, "BlindEffect", "BuffEffectOnSelf")/100
				// Override Blind effect if set.
				if blindOverride := modDB.Override(nil, "BlindEffect"); blindOverride != nil {
					effect = min(blindOverride.Float()/100, effect)
				}
				modDB.AddMod(mod.NewFloat("Accuracy", mod.TypeMore, math.Floor(-20*effect)).Source("Blind"))
				modDB.AddMod(mod.NewFloat("Evasion", mod.TypeMore, math.Floor(-20*effect)).Source("Blind"))
			}
		}
		if modDB.Flag(nil, "Chill") {
			//pob:block:src/Modules/CalcPerform.lua:835-848
			//	local chillValue = m_max(modDB:Sum("BASE", nil, "SelfChillOverride"),
			//	                         modDB:Override(nil, "ChillVal")) or ailmentData.Chill.default
			chillValue := max(modDB.Sum(mod.TypeBase, nil, "SelfChillOverride"),
				utils.Or(modDB.Override(nil, "ChillVal"), 0))

			totalChillSelfEffect := calclib.Mod(modDB, nil, "SelfChillEffect")

			avoidChill := float64(0)
			if modDB.Flag(nil, "ChillImmune") || modDB.Flag(nil, "ElementalAilmentImmune") {
				avoidChill = 100
			} else {
				avoid := modDB.Sum(mod.TypeBase, nil, "AvoidChill", "AvoidAilments", "AvoidElementalAilments")
				if modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
					avoid += modDB.Sum(mod.TypeBase, nil, "AvoidShock")
				}
				if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
					avoid += modDB.Sum(mod.TypeBase, nil, "SpellSuppressionChance") / 2
				}
				avoidChill = math.Floor(min(avoid, 100))
			}

			effect := float64(0)
			if avoidChill != 100 {
				effect = min(max(math.Floor(chillValue*totalChillSelfEffect), 0),
					utils.Or(modDB.Override(nil, "ChillMax"), data.NonDamagingAilments[data.AilmentChill].Max))
			}

			if modDB.Flag(nil, "SkitterbotBonechill") {
				modDB.AddMod(mod.NewFloat("ColdDamageTaken", mod.TypeIncrease,
					effect*utils.Ternary[float64](modDB.Flag(nil, "SelfChillEffectIsReversed"), -1, 1)).Source("Bonechill"))
			}
			modDB.AddMod(mod.NewFloat("ActionSpeed", mod.TypeIncrease, effect*utils.Ternary[float64](modDB.Flag(nil, "SelfChillEffectIsReversed"), 1, -1)).Source("Chill"))
		}
		//pob:block:src/Modules/CalcPerform.lua:849-858
		if modDB.Flag(nil, "Shock") {
			shockValue := max(modDB.Sum(mod.TypeBase, nil, "SelfShockOverride"),
				utils.Or(modDB.Override(nil, "ShockVal"), 0))
			totalShockSelfEffect := calclib.Mod(modDB, nil, "SelfShockEffect")

			avoidShock := float64(0)
			if modDB.Flag(nil, "ShockImmune") || modDB.Flag(nil, "ElementalAilmentImmune") {
				avoidShock = 100
			} else {
				avoid := modDB.Sum(mod.TypeBase, nil, "AvoidShock", "AvoidAilments", "AvoidElementalAilments")
				if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
					avoid += modDB.Sum(mod.TypeBase, nil, "SpellSuppressionChance") / 2
				}
				avoidShock = math.Floor(min(avoid, 100))
			}

			if avoidShock != 100 {
				effect := min(max(math.Floor(shockValue*totalShockSelfEffect), 0),
					utils.Or(modDB.Override(nil, "ShockMax"), data.NonDamagingAilments[data.AilmentShock].Max))
				modDB.AddMod(mod.NewFloat("DamageTaken", mod.TypeIncrease, effect).Source("Shock"))
			}
		}
		//pob:block:src/Modules/CalcPerform.lua:859-868
		if modDB.Flag(nil, "Scorch") {
			scorchValue := max(modDB.Sum(mod.TypeBase, nil, "SelfScorchOverride"),
				utils.Or(modDB.Override(nil, "ScorchVal"), 0))
			totalScorchSelfEffect := calclib.Mod(modDB, nil, "SelfScorchEffect")

			avoidScorch := float64(0)
			if modDB.Flag(nil, "ScorchImmune") || modDB.Flag(nil, "ElementalAilmentImmune") {
				avoidScorch = 100
			} else {
				avoid := modDB.Sum(mod.TypeBase, nil, "AvoidScorch", "AvoidAilments", "AvoidElementalAilments")
				if modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
					avoid += modDB.Sum(mod.TypeBase, nil, "AvoidShock")
				}
				if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
					avoid += modDB.Sum(mod.TypeBase, nil, "SpellSuppressionChance") / 2
				}
				avoidScorch = math.Floor(min(avoid, 100))
			}

			if avoidScorch != 100 {
				effect := min(max(math.Floor(scorchValue*totalScorchSelfEffect), 0),
					utils.Or(modDB.Override(nil, "ScorchMax"), data.NonDamagingAilments[data.AilmentScorch].Max))
				modDB.AddMod(mod.NewFloat("ElementalResist", mod.TypeBase, -effect).Source("Scorch"))
			}
		}
		if modDB.Flag(nil, "Freeze") {
			effect := max(math.Floor(70*calclib.Mod(modDB, nil, "SelfChillEffect")), 0)
			modDB.AddMod(mod.NewFloat("ActionSpeed", mod.TypeIncrease, -effect).Source("Freeze"))
		}
		if modDB.Flag(nil, "CanLeechLifeOnFullLife") && !modDB.Flag(nil, "GhostReaver") {
			modDB.Conditions["Leeching"] = true
			modDB.Conditions["LeechingLife"] = true
		}
		//pob:src/Modules/CalcPerform.lua:877-880
		if modDB.Flag(nil, "CanLeechEnergyShieldOnFullEnergyShield") {
			modDB.Conditions["Leeching"] = true
			modDB.Conditions["LeechingEnergyShield"] = true
		}
		if modDB.Flag(nil, "Condition:CanGainRage") || modDB.Sum(mod.TypeBase, nil, "RageRegen") > 0 {
			maxRageStacks := math.Floor(modDB.Sum(mod.TypeBase, nil, "MaximumRage") *
				modDB.More(nil, "MaximumRage"))
			minRageStacks := min(modDB.Sum(mod.TypeBase, nil, "MinimumRage"), maxRageStacks)
			rageConfig := modDB.Sum(mod.TypeBase, nil, "Multiplier:RageStack")
			rageFloor := float64(0)
			if minRageStacks > 0 {
				rageFloor = minRageStacks
			}
			rageStacks := max(min(rageConfig, maxRageStacks), rageFloor)

			actor.Output["MaximumRage"] = maxRageStacks
			actor.Output["Rage"] = rageStacks
			modDB.AddMod(mod.NewFloat("Multiplier:Rage", mod.TypeBase, rageStacks).Source("Base"))
			//pob:block:src/Modules/CalcPerform.lua:886-895
			actor.Output["RageEffect"] = math.Floor(actor.Output["Rage"] * calclib.Mod(modDB, nil, "RageEffect"))
			modDB.AddMod(mod.NewFloat("Multiplier:RageEffect", mod.TypeBase, actor.Output["RageEffect"]).Source("Base"))
			if modDB.Flag(nil, "Condition:RageSpellDamage") {
				modDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, actor.Output["RageEffect"]).Source("Rage").Flag(mod.MFlagSpell))
			} else {
				modDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, actor.Output["RageEffect"]).Source("Rage").Flag(mod.MFlagAttack))
			}
			if rageStacks == maxRageStacks {
				modDB.AddMod(mod.NewFlag("Condition:HaveMaximumRage", true))
			}
			//pob:block:src/Modules/CalcPerform.lua:899-900
			actor.Output["InherentRageLossDelay"] = 2 + modDB.Sum(mod.TypeBase, nil, "InherentRageLossDelay")
			actor.Output["InherentRageLoss"] = 0
			if !modDB.Flag(nil, "InherentRageLossIsPrevented") {
				actor.Output["InherentRageLoss"] = 10 * (1 + modDB.Sum(mod.TypeIncrease, nil, "InherentRageLoss")/100)
			}
		}
		//pob:block:src/Modules/CalcPerform.lua:902-915
		if manaBurnConfig := env.Build.GetNumberOption("multiplierManaBurnStacks"); manaBurnConfig > 0 {
			maxManaBurn := modDB.Sum(mod.TypeBase, nil, "MaxManaBurnStacks")
			if maxManaBurn == 0 {
				maxManaBurn = 9999
			}
			manaBurnStacks := math.Min(manaBurnConfig, maxManaBurn)
			modDB.AddMod(mod.NewFloat("Multiplier:ManaBurnStacks", mod.TypeBase, manaBurnStacks).Source("Config"))
			manaBurnStacks += modDB.Sum(mod.TypeBase, playerActorCfg, "EffectiveManaBurnStacks")
			if modDB.Flag(nil, "Condition:WeepingWoundsInsteadOfManaBurn") {
				modDB.AddMod(mod.NewFloat("Multiplier:WeepingWoundsStacks", mod.TypeBase, manaBurnStacks).Source("Config"))
			} else {
				modDB.AddMod(mod.NewFloat("Multiplier:EffectiveManaBurnStacks", mod.TypeBase, manaBurnStacks).Source("Config"))
			}
		}
		if modDB.Sum(mod.TypeBase, nil, "CoveredInAshEffect") > 0 {
			effect := modDB.Sum(mod.TypeBase, nil, "CoveredInAshEffect")
			enemyDB.AddMod(mod.NewFloat("FireDamageTaken", mod.TypeIncrease, min(effect, 20)).Source("Covered in Ash"))
		}
		if modDB.Sum(mod.TypeBase, nil, "CoveredInFrostEffect") > 0 {
			effect := modDB.Sum(mod.TypeBase, nil, "CoveredInFrostEffect")
			enemyDB.AddMod(mod.NewFloat("ColdDamageTaken", mod.TypeIncrease, min(effect, 20)).Source("Covered in Frost"))
		}
		if modDB.Flag(nil, "HasMalediction") {
			modDB.AddMod(mod.NewFloat("DamageTaken", mod.TypeIncrease, 10).Source("Malediction"))
			modDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, -10).Source("Malediction"))
		}
		if modDB.Flag(nil, "HasMaddeningPresence") {
			modDB.AddMod(mod.NewFloat("ActionSpeed", mod.TypeIncrease, -10).Source("Maddening Presence"))
			modDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, -10).Source("Maddening Presence"))
		}
		if modDB.Flag(nil, "HasShapersPresence") {
			modDB.AddMod(mod.NewFloat("BuffExpireFaster", mod.TypeMore, -20).Source("Shapers Presence"))
		}
		//pob:block:src/Modules/CalcPerform.lua:935-939
		if modDB.Flag(nil, "Condition:CanHaveSoulEater") {
			soulEaterMax := modDB.Sum(mod.TypeBase, nil, "SoulEaterMax")
			if o := modDB.Override(nil, "SoulEaterMax"); o != nil {
				soulEaterMax = o.Float()
			}
			for _, flag := range []mod.MFlag{mod.MFlagAttack, mod.MFlagCast} {
				modDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 5).Source("Base").Flag(flag).
					Tag(mod.Multiplier("SoulEaterStack").Limit(soulEaterMax)))
			}
		}
	}

	//pob:block:src/Modules/CalcPerform.lua:942-943
	applyEnemyModifiers(actor, false)
}

//pob:src/Modules/CalcPerform.lua:595-609
func applyEnemyModifiers(actor *Actor, clearCache bool) {
	if clearCache || actor.appliedEnemyModifiers == nil {
		actor.appliedEnemyModifiers = make(map[mod.Mod]int)
	}
	cache := actor.appliedEnemyModifiers
	seen := make(map[mod.Mod]int)
	enemyDB := actor.Enemy.ModDB
	for _, value := range actor.ModDB.Tabulate("", nil, "EnemyModifier") {
		if value.Mod.Value() == nil || value.Mod.Value().Type() != mod.ModValueMultiTypeList {
			continue
		}
		payload, ok := value.Mod.Value().List().(mod.EnemyModifier)
		if !ok || payload.Mod == nil {
			continue
		}
		seen[value.Mod]++
		if seen[value.Mod] <= cache[value.Mod] {
			continue
		}
		source := payload.Mod.GetSource()
		if source == "" {
			source = value.Mod.GetSource()
		}
		enemyDB.AddMod(payload.Mod.Clone().Source(source))
		cache[value.Mod]++
	}
}

func sortedModListKeys(m map[string]*moddb.ModList) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type minionCountEntry struct {
	total        float64
	nonVaal      float64
	hasNonVaal   bool
	permanent    float64
	hasPermanent bool
}

func initMinionModDB(env *Environment, activeSkill *ActiveSkill, output map[string]float64) {
	minion := activeSkill.Minion
	if minion == nil {
		return
	}
	if output == nil {
		output = make(map[string]float64)
	}
	minion.Output = output
	minion.OutputTable = make(map[OutTable]map[string]float64)
	minion.OutputStrings = make(map[string]string)
	minion.ModDB.Multipliers["Level"] = float64(minion.Level)
	initModDB(env, minion.ModDB)
	mapLevelLifeMult := 1.0
	if minion.MinionData.Hostile {
		if v, ok := data.MapLevelLifeMult[env.EnemyLevel]; ok {
			mapLevelLifeMult = v
		}
	}

	baseLife := float64(minion.LifeTable[minion.Level]) * minion.MinionData.Life
	if minion.MinionData.Hostile {
		baseLife *= mapLevelLifeMult
	}
	minion.ModDB.AddMod(mod.NewFloat("Life", mod.TypeBase, math.Floor(baseLife)).Source("Base"))
	if minion.MinionData.EnergyShield != 0 {
		esTable := data.MonsterAllyLifeTable
		if minion.MinionData.Hostile {
			esTable = minion.LifeTable
		}
		baseES := float64(esTable[minion.Level]) * minion.MinionData.Life * minion.MinionData.EnergyShield
		if minion.MinionData.Hostile {
			baseES *= mapLevelLifeMult
		}
		minion.ModDB.AddMod(mod.NewFloat("EnergyShield", mod.TypeBase, math.Floor(baseES)).Source("Base"))
	}
	armourMult := minion.MinionData.Armour
	if armourMult == 0 {
		armourMult = 1
	}
	minion.ModDB.AddMod(mod.NewFloat("Armour", mod.TypeBase, utils.RoundTo(data.MonsterArmourTable[minion.Level]*armourMult, 0)).Source("Base"))
	evasionMult := minion.MinionData.Evasion
	if evasionMult == 0 {
		evasionMult = 1
	}
	minion.ModDB.AddMod(mod.NewFloat("Evasion", mod.TypeBase, utils.RoundTo(data.MonsterEvasionTable[minion.Level]*evasionMult, 0)).Source("Base"))
	//pob:block:src/Modules/CalcPerform.lua:1141-1145
	if env.ModDB.Flag(nil, "MinionAccuracyEqualsAccuracy") {
		dexBonus := env.ModDB.Override(nil, "DexAccBonusOverride")
		perDex := data.AccuracyPerDexBase
		if dexBonus != nil {
			perDex = dexBonus.Float()
		}
		minion.ModDB.AddMod(mod.NewFloat("Accuracy", mod.TypeBase,
			calclib.Val(env.ModDB, "Accuracy")+float64(calclib.Val(env.ModDB, "Dex")*perDex)).Source("Player"))
	} else {
		minion.ModDB.AddMod(mod.NewFlag("CannotBeEvaded", true).Source("Minion Attacks always hit"))
	}
	minion.ModDB.AddMod(mod.NewFloat("CritMultiplier", mod.TypeBase, 30).Source("Base"))
	//pob:block:src/Modules/CalcPerform.lua:1147
	minion.ModDB.AddMod(mod.NewFloat("DotMultiplier", mod.TypeBase, 30).Source("Base").Tag(mod.Condition("CriticalStrike")))
	minion.ModDB.AddMod(mod.NewFloat("FireResist", mod.TypeBase, minion.MinionData.FireResist).Source("Base"))
	minion.ModDB.AddMod(mod.NewFloat("ColdResist", mod.TypeBase, minion.MinionData.ColdResist).Source("Base"))
	minion.ModDB.AddMod(mod.NewFloat("LightningResist", mod.TypeBase, minion.MinionData.LightningResist).Source("Base"))
	minion.ModDB.AddMod(mod.NewFloat("ChaosResist", mod.TypeBase, minion.MinionData.ChaosResist).Source("Base"))
	//pob:block:src/Modules/CalcPerform.lua:1152
	minion.ModDB.AddMod(mod.NewFloat("CritChance", mod.TypeIncrease, data.CriticalStrikeChancePlusPercentPerPowerCharge).Source("Base").Tag(mod.Multiplier("PowerCharge")))
	minion.ModDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 4).Source("Base").Flag(mod.MFlagAttack).Tag(mod.Multiplier("FrenzyCharge")))
	minion.ModDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 4).Source("Base").Flag(mod.MFlagCast).Tag(mod.Multiplier("FrenzyCharge")))
	minion.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, 4).Source("Base").Tag(mod.Multiplier("FrenzyCharge")))
	//pob:block:src/Modules/CalcPerform.lua:1156-1157
	minion.ModDB.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, data.PhysicalDamageReductionPercentPerEnduranceCharge).Source("Base").Tag(mod.Multiplier("EnduranceCharge")))
	minion.ModDB.AddMod(mod.NewFloat("ElementalDamageReduction", mod.TypeBase, data.ElementalDamageReductionPercentPerEnduranceCharge).Source("Base").Tag(mod.Multiplier("EnduranceCharge")))
	minion.ModDB.AddMod(mod.NewFloat("ProjectileCount", mod.TypeBase, 1).Source("Base"))
	minion.ModDB.AddMod(mod.NewFloat("MaximumFortification", mod.TypeBase, 20).Source("Base"))
	minion.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, 200).Source("Base").KeywordFlag(mod.KeywordFlagBleed).Tag(mod.ActorCondition("enemy", "Moving")))
	for _, Mod := range minion.MinionData.ModList {
		minion.ModDB.AddMod(Mod)
	}
	for _, Mod := range activeSkill.ExtraSkillModList {
		minion.ModDB.AddMod(Mod)
	}
	if env.AegisModList != nil {
		minion.ItemList["Weapon 3"] = env.Player.ItemList["Weapon 2"]
		minion.ModDB.AddList(env.AegisModList)
	}
	if env.TheIronMass != nil && minion.Type == "RaisedSkeleton" {
		minion.ModDB.AddList(env.TheIronMass)
	}
	if activeSkill.SkillData.MinionUseBowAndQuiver {
		if env.Player.WeaponData1 != nil && env.Player.WeaponData1.Type == "Bow" {
			if weapon1 := env.Player.ItemList["Weapon 1"]; weapon1 != nil && weapon1.SlotModList[1] != nil {
				minion.ModDB.AddList(weapon1.SlotModList[1])
			}
		}
		if weapon2 := env.Player.ItemList["Weapon 2"]; weapon2 != nil && weapon2.Type == "Quiver" && weapon2.ModList != nil {
			minion.ModDB.ScaleAddList(weapon2.ModList, math.Max(env.ModDB.Sum(mod.TypeBase, nil, "WidowHailMultiplier"), 1))
		}
		if env.ModDB.Flag(nil, "BlinkAndMirrorUseGloves") {
			if gloves := env.Player.ItemList["Gloves"]; gloves != nil && gloves.ModList != nil {
				minion.ModDB.AddList(gloves.ModList)
			}
		}
	}
	if activeSkill.SkillData.MinionUseMainHandWeapon {
		if weapon1 := env.Player.ItemList["Weapon 1"]; weapon1 != nil && weapon1.SlotModList[1] != nil {
			minion.ModDB.AddList(weapon1.SlotModList[1])
		}
	}
	//pob:block:src/Modules/CalcPerform.lua:1224-1242
	if minion.ItemSet != nil || minion.Uses != nil {
		for _, slot := range orderedItemSlots {
			if !minion.Uses[slot.name] {
				continue
			}

			if minion.ItemSet != nil {
				slotName := slot.name
				if slot.weaponSet == 1 {
					slotName = weaponSetSlotName(minion.ItemSet, slotName)
				}
				item := itemSetSlotItem(env, minion.ItemSet, slotName)
				if item == nil {
					continue
				}
				minion.ItemList[slotName] = itemToItemData(item)
				minion.ModDB.AddList(asModList(item.modListForSlot(slot.slotNum)))
				continue
			}

			item := env.Player.ItemList[slot.name]
			if item == nil {
				continue
			}
			minion.ItemList[slot.name] = item
			srcList := item.ModList
			if srcList == nil {
				srcList = item.SlotModList[slot.slotNum]
			}
			if srcList != nil {
				minion.ModDB.AddList(srcList)
			}
		}
	}
	strAdded := env.ModDB.Sum(mod.TypeBase, nil, "StrengthAddedToMinions")
	if strAdded == 0 {
		if env.ModDB.Flag(nil, "StrengthAddedToMinions") {
			strAdded = 100
		} else if env.ModDB.Flag(nil, "HalfStrengthAddedToMinions") {
			strAdded = 50
		}
	}
	if strAdded > 0 {
		minion.ModDB.AddMod(mod.NewFloat("Str", mod.TypeBase, utils.RoundTo(calclib.Val(env.ModDB, "Str")*strAdded/100, 0)).Source("Player"))
	}
}

//pob:block:src/Modules/CalcPerform.lua:1213-1219
func addMinionModifiers(modList *moddb.ModList, skillCfg *moddb.ListCfg, minion *ActorMinion) {
	if modList == nil {
		return
	}
	for _, value := range modList.List(skillCfg, "MinionModifier") {
		minionMod, ok := value.(mod.MinionModifier)
		if !ok {
			continue
		}
		minion.ModDB.AddMod(minionMod.Mod)
	}
}

func prepareAllyMinion(env *Environment, activeSkill *ActiveSkill) {
	minion := activeSkill.Minion
	if minion == nil {
		return
	}
	initMinionModDB(env, activeSkill, nil)
	addMinionModifiers(activeSkill.SkillModList, activeSkill.SkillCfg, minion)
	for _, name := range sortedModListKeys(env.Buffs) {
		addMinionModifiers(env.Buffs[name], activeSkill.SkillCfg, minion)
	}
	for _, name := range sortedModListKeys(env.MinionBuffs) {
		minion.ModDB.AddList(env.MinionBuffs[name])
	}
	for _, value := range minion.ModDB.List(nil, "Keystone") {
		name, ok := value.(string)
		if !ok {
			continue
		}
		if list := keystoneModList(env, env.Spec.TreeVersion, name); list != nil {
			minion.ModDB.AddList(list)
		}
	}
	doActorAttribsPoolsConditions(env, minion.Actor)
}

//pob:block:src/Modules/CalcPerform.lua:2328-2332
func warcryPowerBonus(m mod.Mod, warcryPower float64) float64 {
	for _, tag := range m.Tags() {
		get, ok := tag.(*mod.GlobalEffectTag)
		if !ok || len(get.GlobalEffectList) == 0 || get.GlobalEffectList[0] != "Warcry" || !get.DivSet {
			continue
		}
		power := warcryPower
		if get.LimitSet {
			power = min(power, get.LimitTag)
		}
		return math.Floor(power / get.DivTag)
	}
	return 1
}

func singleModList(m mod.Mod) *moddb.ModList {
	list := moddb.NewModList()
	list.AddMod(m)
	return list
}

//pob:block:src/Modules/CalcPerform.lua:3458
func ailmentStacksMax(modDB *moddb.ModDB, name string) float64 {
	return utils.Or(modDB.Override(nil, name), modDB.Sum(mod.TypeBase, nil, name))
}

func chargesMax(modDB *moddb.ModDB, name string, fallback float64) float64 {
	return utils.Or(modDB.Override(nil, name), math.Max(fallback, 0))
}

//pob:block:src/Modules/CalcPerform.lua:955-958
func replaceChargesMaxFromFlag(modDB *moddb.ModDB, flagName string, chargeName string, value float64) {
	if !modDB.Flag(nil, flagName) {
		return
	}
	source := mod.Source("")
	if mods := modDB.Mods[flagName]; len(mods) > 0 {
		source = mods[0].GetSource()
	}
	modDB.ReplaceMod(mod.NewFloat(chargeName, mod.TypeOverride, value).Source(source))
}

//pob:block:src/Modules/CalcPerform.lua:3321
func impaleStacksCap(modDB *moddb.ModDB) float64 {
	return modDB.Sum(mod.TypeBase, nil, "ImpaleStacksMax") * (1 + modDB.Sum(mod.TypeBase, nil, "ImpaleAdditionalDurationChance")/100)
}

//pob:block:src/Modules/CalcPerform.lua:290-374
func applyShrineBuffs(actor *Actor) {
	modDB := actor.ModDB
	shrineEffectMod := 1 + modDB.Sum(mod.TypeIncrease, nil, "BuffEffectOnSelf", "ShrineBuffEffect")/100
	floorEffect := func(base float64) float64 { return math.Floor(base * shrineEffectMod) }

	if modDB.Flag(nil, "AccelerationShrine") {
		modDB.AddMod(mod.NewFloat("ActionSpeed", mod.TypeIncrease, floorEffect(50)).Source("Acceleration Shrine"))
		modDB.AddMod(mod.NewFloat("ProjectileSpeed", mod.TypeIncrease, floorEffect(80)).Source("Acceleration Shrine"))
	}
	if modDB.Flag(nil, "BrutalShrine") {
		modDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, floorEffect(50)).Source("Brutal Shrine"))
		modDB.AddMod(mod.NewFloat("EnemyStunDuration", mod.TypeIncrease, floorEffect(30)).Source("Brutal Shrine"))
		modDB.AddMod(mod.NewFloat("EnemyKnockbackChance", mod.TypeIncrease, 100).Source("Brutal Shrine"))
	}
	if modDB.Flag(nil, "DiamondShrine") {
		modDB.AddMod(mod.NewFloat("CritChance", mod.TypeOverride, 100).Source("Diamond Shrine"))
	}
	if modDB.Flag(nil, "DivineShrine") {
		modDB.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -100).Source("Divine Shrine"))
	}
	if modDB.Flag(nil, "EchoingShrine") {
		modDB.AddMod(mod.NewFloat("Speed", mod.TypeMore, floorEffect(100)).Source("Echoing Shrine").Flag(mod.MFlagAttack))
		modDB.AddMod(mod.NewFloat("Speed", mod.TypeMore, floorEffect(100)).Source("Echoing Shrine").Flag(mod.MFlagCast))
		modDB.AddMod(mod.NewFloat("RepeatCount", mod.TypeBase, floorEffect(1)).Source("Echoing Shrine"))
	}
	if modDB.Flag(nil, "GloomShrine") {
		modDB.AddMod(mod.NewFloat("NonChaosDamageGainAsChaos", mod.TypeBase, floorEffect(10)).Source("Gloom Shrine"))
	}
	if modDB.Flag(nil, "GreaterFreezingShrine") {
		modDB.AddMod(mod.NewFloat("PhysicalDamageGainAsCold", mod.TypeBase, floorEffect(30)).Source("Greater Freezing Shrine"))
	}
	if modDB.Flag(nil, "GreaterShockingShrine") {
		modDB.AddMod(mod.NewFloat("PhysicalDamageGainAsLightning", mod.TypeBase, floorEffect(30)).Source("Greater Shocking Shrine"))
	}
	if modDB.Flag(nil, "GreaterSkeletalShrine") {
		modDB.AddMod(mod.NewFloat("PhysicalDamageGainAsChaos", mod.TypeBase, floorEffect(30)).Source("Greater Skeletal Shrine"))
	}
	if modDB.Flag(nil, "ImpenetrableShrine") {
		modDB.AddMod(mod.NewFloat("Armour", mod.TypeIncrease, floorEffect(100)).Source("Impenetrable Shrine"))
		modDB.AddMod(mod.NewFloat("Evasion", mod.TypeIncrease, floorEffect(100)).Source("Impenetrable Shrine"))
		modDB.AddMod(mod.NewFloat("EnergyShield", mod.TypeIncrease, floorEffect(100)).Source("Impenetrable Shrine"))
	}
	if modDB.Flag(nil, "MassiveShrine") {
		modDB.AddMod(mod.NewFloat("Life", mod.TypeIncrease, floorEffect(40)).Source("Massive Shrine"))
		modDB.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeIncrease, floorEffect(40)).Source("Massive Shrine"))
	}
	if modDB.Flag(nil, "ReplenishingShrine") {
		modDB.AddMod(mod.NewFloat("ManaRegen", mod.TypeIncrease, 200*shrineEffectMod).Source("Replenishing Shrine"))
		modDB.AddMod(mod.NewFloat("LifeRegenPercent", mod.TypeBase, 6.7*shrineEffectMod).Source("Replenishing Shrine"))
	}
	if modDB.Flag(nil, "ResistanceShrine") {
		modDB.AddMod(mod.NewFloat("ElementalResist", mod.TypeBase, floorEffect(50)).Source("Resistance Shrine"))
		modDB.AddMod(mod.NewFloat("ElementalResistMax", mod.TypeBase, floorEffect(10)).Source("Resistance Shrine"))
		modDB.AddMod(mod.NewFloat("ChaosResistMax", mod.TypeBase, floorEffect(10)).Source("Resistance Shrine"))
	}
	if modDB.Flag(nil, "ResonatingShrine") {
		modDB.AddMod(mod.NewFloat("PowerChargesMax", mod.TypeBase, floorEffect(1)).Source("Resonating Shrine"))
		modDB.AddMod(mod.NewFloat("FrenzyChargesMax", mod.TypeBase, floorEffect(1)).Source("Resonating Shrine"))
		modDB.AddMod(mod.NewFloat("EnduranceChargesMax", mod.TypeBase, floorEffect(1)).Source("Resonating Shrine"))
	}
	if modDB.Flag(nil, "LesserAccelerationShrine") && !modDB.Flag(nil, "AccelerationShrine") {
		modDB.AddMod(mod.NewFloat("ActionSpeed", mod.TypeIncrease, floorEffect(10)).Source("Lesser Acceleration Shrine"))
		modDB.AddMod(mod.NewFloat("ProjectileSpeed", mod.TypeIncrease, floorEffect(30)).Source("Lesser Acceleration Shrine"))
	}
	if modDB.Flag(nil, "LesserBrutalShrine") {
		modDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, floorEffect(20)).Source("Lesser Brutal Shrine"))
		modDB.AddMod(mod.NewFloat("EnemyStunDuration", mod.TypeIncrease, floorEffect(20)).Source("Lesser Brutal Shrine"))
		modDB.AddMod(mod.NewFloat("EnemyKnockbackChance", mod.TypeIncrease, 100).Source("Lesser Brutal Shrine"))
	}
	if modDB.Flag(nil, "LesserImpenetrableShrine") {
		modDB.AddMod(mod.NewFloat("Armour", mod.TypeIncrease, floorEffect(50)).Source("Lesser Impenetrable Shrine"))
		modDB.AddMod(mod.NewFloat("Evasion", mod.TypeIncrease, floorEffect(50)).Source("Lesser Impenetrable Shrine"))
		modDB.AddMod(mod.NewFloat("EnergyShield", mod.TypeIncrease, floorEffect(50)).Source("Lesser Impenetrable Shrine"))
	}
	if modDB.Flag(nil, "LesserMassiveShrine") {
		modDB.AddMod(mod.NewFloat("Life", mod.TypeIncrease, floorEffect(20)).Source("Lesser Massive Shrine"))
		modDB.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeIncrease, floorEffect(20)).Source("Lesser Massive Shrine"))
	}
	if modDB.Flag(nil, "LesserReplenishingShrine") {
		modDB.AddMod(mod.NewFloat("ManaRegen", mod.TypeIncrease, 100*shrineEffectMod).Source("Lesser Replenishing Shrine"))
		modDB.AddMod(mod.NewFloat("LifeRegenPercent", mod.TypeBase, 3.3*shrineEffectMod).Source("Lesser Replenishing Shrine"))
	}
	if modDB.Flag(nil, "LesserResistanceShrine") {
		modDB.AddMod(mod.NewFloat("ElementalResist", mod.TypeBase, floorEffect(25)).Source("Lesser Resistance Shrine"))
		modDB.AddMod(mod.NewFloat("ElementalResistMax", mod.TypeBase, floorEffect(2)).Source("Lesser Resistance Shrine"))
		modDB.AddMod(mod.NewFloat("ChaosResistMax", mod.TypeBase, floorEffect(2)).Source("Lesser Resistance Shrine"))
	}
}

//pob:block:src/Modules/CalcPerform.lua:2605-2609
func balanceOfTerrorSelfOnly(env *Environment, buffNameKey string) bool {
	if env.Build == nil {
		return false
	}
	name := "balanceOfTerrorSelfCast" + buffNameKey
	for _, input := range env.Build.Config.Inputs {
		if input.Name == name && input.Boolean != nil {
			return *input.Boolean
		}
	}
	return false
}

//pob:src/Modules/CalcPerform.lua:1812-1893
func mergeTinctures(env *Environment, effectInc, effectIncMagic, tinctureLimit float64) {
	//pob:src/Modules/CalcPerform.lua:1824-1828
	tincturesNotInflictManaBurn := math.Min(env.ModDB.Sum(mod.TypeBase, nil, "TincturesNotInflictManaBurn"), 100)
	canGainRequiredBurn := env.ModDB.Flag(nil, "Condition:WeepingWoundsInsteadOfManaBurn") ||
		(tincturesNotInflictManaBurn < 100 && env.Player.Output["ManaUnreserved"] > 0)
	if !canGainRequiredBurn {
		return
	}

	tinctureBuffs := make(map[string]*moddb.ModList)
	tinctureBuffOrder := make([]string, 0, len(env.Tinctures)*2)
	tinctureConditions := make(map[string]bool)

	for _, item := range env.Tinctures {
		if tinctureLimit <= 0 {
			break
		}
		tinctureLimit--

		tinctureConditions["UsingTincture"] = true
		tinctureConditions["Using"+strings.Join(strings.Fields(item.BaseName), "")] = true

		//pob:src/Modules/CalcPerform.lua:1833-1841
		tinctureEffectInc := effectInc + item.TinctureData["effectInc"]
		if item.Rarity == RarityMagic {
			tinctureEffectInc += effectIncMagic
		}
		quality := 0.0
		if item.Quality != nil {
			quality = float64(*item.Quality)
		}
		effectMod := math.Floor((1+tinctureEffectInc/100)*(1+quality/100)*100) / 100

		if len(item.BuffModList) > 0 {
			srcList := moddb.NewModList()
			srcList.ScaleAddList(asModList(item.BuffModList), effectMod)
			tinctureBuffOrder = mergeFlaskBuff(srcList, tinctureBuffs, tinctureBuffOrder, item.BaseName)
		}
		if len(item.Mods) > 0 {
			srcList := moddb.NewModList()
			srcList.ScaleAddList(asModList(item.Mods), effectMod)
			key := item.Title
			if item.Rarity != RarityUnique {
				key = flaskModKey(item.Mods)
			}
			tinctureBuffOrder = mergeFlaskBuff(srcList, tinctureBuffs, tinctureBuffOrder, key)
		}
	}

	//pob:src/Modules/CalcPerform.lua:1876-1881
	for cond, status := range tinctureConditions {
		env.ModDB.Conditions[cond] = status
	}
	for _, key := range tinctureBuffOrder {
		env.ModDB.AddList(tinctureBuffs[key])
	}

	//pob:src/Modules/CalcPerform.lua:1882-1892
	if env.ModDB.Flag(nil, "TinctureRangedWeapons") {
		for _, key := range tinctureBuffOrder {
			for _, buff := range tinctureBuffs[key].Mods() {
				if buff.Flags()&mod.MFlagWeaponMelee == mod.MFlagWeaponMelee {
					newMod := buff.Clone()
					if base, ok := newMod.(*mod.BaseMod); ok {
						base.ModFlags = (base.ModFlags &^ mod.MFlagWeaponMelee) | mod.MFlagWeaponRanged
					}
					env.ModDB.AddMod(newMod)
				}
			}
		}
	}
}

//pob:block:src/Modules/CalcTriggers.lua:737-747
func dualWieldChance(env *Environment, source *ActiveSkill, combined, mainHand, offHand float64) float64 {
	if source.SkillData == nil || !source.SkillData.DoubleHitsWhenDualWielding {
		return combined
	}
	if env.Player.WeaponData1 == nil || env.Player.WeaponData2 == nil ||
		env.Player.WeaponData1.Type == "" || env.Player.WeaponData2.Type == "" {
		return combined
	}
	return mainHand*offHand/100 + mainHand*(100-offHand)/100 + (100-mainHand)*offHand/100
}

var elementKeywordFlag = map[string]mod.KeywordFlag{
	"Lightning": mod.KeywordFlagLightning,
	"Fire":      mod.KeywordFlagFire,
	"Cold":      mod.KeywordFlagCold,
	"Chaos":     mod.KeywordFlagChaos,
	"Physical":  mod.KeywordFlagPhysical,
}

var reservationNames = func() map[string]*struct {
	reserved, efficiency, outEfficiency, flatForced, percentForced, reservedPercent, reservedBase string
	reservedBaseOut, reservedPercentOut, uncancellable                                            string
} {
	out := map[string]*struct {
		reserved, efficiency, outEfficiency, flatForced, percentForced, reservedPercent, reservedBase string
		reservedBaseOut, reservedPercentOut, uncancellable                                            string
	}{}
	for _, name := range []string{"Mana", "Life"} {
		out[name] = &struct {
			reserved, efficiency, outEfficiency, flatForced, percentForced, reservedPercent, reservedBase string
			reservedBaseOut, reservedPercentOut, uncancellable                                            string
		}{name + "Reserved", name + "ReservationEfficiency", name + "Efficiency", name + "ReservationFlatForced", name + "ReservationPercentForced",
			name + "ReservedPercent", name + "ReservedBase", "reserved_" + name + "Base", "reserved_" + name + "Percent", "uncancellable_" + name + "Reservation"}
	}
	return out
}()

var poolReservationNames = func() map[string]*struct {
	lowPercentage, reservedBase, reservedPercentIn, uncancellableIn, reserved, reservedPercent string
	unreserved, unreservedPercent, uncancellable, cancellable, low                             string
} {
	out := map[string]*struct {
		lowPercentage, reservedBase, reservedPercentIn, uncancellableIn, reserved, reservedPercent string
		unreserved, unreservedPercent, uncancellable, cancellable, low                             string
	}{}
	for _, pool := range []string{"Life", "Mana"} {
		out[pool] = &struct {
			lowPercentage, reservedBase, reservedPercentIn, uncancellableIn, reserved, reservedPercent string
			unreserved, unreservedPercent, uncancellable, cancellable, low                             string
		}{"Low" + pool + "Percentage", "reserved_" + pool + "Base", "reserved_" + pool + "Percent", "uncancellable_" + pool + "Reservation",
			pool + "Reserved", pool + "ReservedPercent", pool + "Unreserved", pool + "UnreservedPercent", pool + "UncancellableReservation",
			pool + "CancellableReservation", "Low" + pool}
	}
	return out
}()
