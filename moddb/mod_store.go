package moddb

import (
	"maps"
	"math"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
	raw "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
)

type ListCfg struct {
	Actor        string
	Flags        *mod.MFlag
	KeywordFlags *mod.KeywordFlag
	Source       *mod.Source
	SkillStats   map[string]float64
	SkillCond    map[string]bool
	SlotName     string
	Item         CfgItem

	SkillName          string
	SummonSkillName    string
	SkillGem           *poe.SkillGem
	SkillGrantedEffect any
	SkillPart          int
	SkillTypes         map[data.SkillType]bool
	SkillDist          float64
	SkillDistSet       bool
	SocketColor        string
	SocketNum          int
}

var GemIsType func(gem *poe.SkillGem, keyword string) bool

type ModStoreFuncs interface {
	List(cfg *ListCfg, names ...string) []interface{}
	Sum(modType mod.Type, cfg *ListCfg, names ...string) float64
	More(cfg *ListCfg, names ...string) float64
	Flag(cfg *ListCfg, names ...string) bool
	Override(cfg *ListCfg, names ...string) *mod.ModValueMulti
	GetMultiplier(variable string, cfg *ListCfg, noMod bool) float64
	GetCondition(variable string, cfg *ListCfg, noMod bool) (bool, bool)
	GetActor() Actor
	Clone() ModStoreFuncs
	TabulateInternal(context ModStoreFuncs, result *[]ModResult, modType mod.Type,
		cfg *ListCfg, flags mod.MFlag, keywordFlags mod.KeywordFlag, source mod.Source, modNames ...string)
	Combine(modType mod.Type, cfg *ListCfg, modNames ...string) *mod.ModValueMulti

	SumInternal(context ModStoreFuncs, modType mod.Type, cfg *ListCfg, names ...string) float64
	MoreInternal(context ModStoreFuncs, cfg *ListCfg, names ...string) float64
	FlagInternal(context ModStoreFuncs, cfg *ListCfg, names ...string) bool
	OverrideInternal(context ModStoreFuncs, cfg *ListCfg, names ...string) *mod.ModValueMulti
	ListInternal(context ModStoreFuncs, result *[]interface{}, cfg *ListCfg, names ...string)

	evalMod(m mod.Mod, cfg *ListCfg) *mod.ModValueMulti

	getStat(stat string, cfg *ListCfg) float64
}

func heapNames(names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = strings.Clone(n)
	}
	return out
}

func sumIn(s, context ModStoreFuncs, modType mod.Type, cfg *ListCfg, names ...string) float64 {
	switch s := s.(type) {
	case *ModDB:
		return s.SumInternal(context, modType, cfg, names...)
	case *ModList:
		return s.SumInternal(context, modType, cfg, names...)
	}
	return s.SumInternal(context, modType, cfg, heapNames(names)...)
}

func moreIn(s, context ModStoreFuncs, cfg *ListCfg, names ...string) float64 {
	switch s := s.(type) {
	case *ModDB:
		return s.MoreInternal(context, cfg, names...)
	case *ModList:
		return s.MoreInternal(context, cfg, names...)
	}
	return s.MoreInternal(context, cfg, heapNames(names)...)
}

func flagIn(s, context ModStoreFuncs, cfg *ListCfg, names ...string) bool {
	switch s := s.(type) {
	case *ModDB:
		return s.FlagInternal(context, cfg, names...)
	case *ModList:
		return s.FlagInternal(context, cfg, names...)
	}
	return s.FlagInternal(context, cfg, heapNames(names)...)
}

func overrideIn(s, context ModStoreFuncs, cfg *ListCfg, names ...string) *mod.ModValueMulti {
	switch s := s.(type) {
	case *ModDB:
		return s.OverrideInternal(context, cfg, names...)
	case *ModList:
		return s.OverrideInternal(context, cfg, names...)
	}
	return s.OverrideInternal(context, cfg, heapNames(names)...)
}

func SumOf(s ModStoreFuncs, modType mod.Type, cfg *ListCfg, names ...string) float64 {
	return sumIn(s, s, modType, cfg, names...)
}

func MoreOf(s ModStoreFuncs, cfg *ListCfg, names ...string) float64 {
	return moreIn(s, s, cfg, names...)
}

func listIn(s, context ModStoreFuncs, result *[]interface{}, cfg *ListCfg, names ...string) {
	switch s := s.(type) {
	case *ModDB:
		s.ListInternal(context, result, cfg, names...)
	case *ModList:
		s.ListInternal(context, result, cfg, names...)
	default:
		s.ListInternal(context, result, cfg, heapNames(names)...)
	}
}

func tabulateIn(s, context ModStoreFuncs, result *[]ModResult, modType mod.Type, cfg *ListCfg, flags mod.MFlag,
	keywordFlags mod.KeywordFlag, source mod.Source, names ...string) {
	switch s := s.(type) {
	case *ModDB:
		s.TabulateInternal(context, result, modType, cfg, flags, keywordFlags, source, names...)
	case *ModList:
		s.TabulateInternal(context, result, modType, cfg, flags, keywordFlags, source, names...)
	default:
		s.TabulateInternal(context, result, modType, cfg, flags, keywordFlags, source, heapNames(names)...)
	}
}

func modName(mo mod.Mod) string {
	if bm, ok := mo.(*mod.BaseMod); ok {
		return bm.ModName
	}
	return mo.Name()
}

func modMatches(mo mod.Mod, modType mod.Type, cfg *ListCfg) bool {
	if bm, ok := mo.(*mod.BaseMod); ok {
		return bm.ModType == modType &&
			cfgFlags(cfg)&bm.ModFlags == bm.ModFlags &&
			mod.MatchKeywordFlags(cfgKeywordFlags(cfg), bm.ModKeywordFlags) &&
			(cfg == nil || cfg.Source == nil || sourcePrefixMatch(bm.ModSource, *cfg.Source))
	}
	return mo.Type() == modType &&
		cfgFlags(cfg)&mo.Flags() == mo.Flags() &&
		mod.MatchKeywordFlags(cfgKeywordFlags(cfg), mo.KeywordFlags()) &&
		(cfg == nil || cfg.Source == nil || sourcePrefixMatch(mo.GetSource(), *cfg.Source))
}

//pob:block:src/Classes/ModStore.lua:143-149
func cfgFlags(cfg *ListCfg) mod.MFlag {
	if cfg == nil {
		return 0
	}
	return cfg.Flags.Get()
}

//pob:block:src/Classes/ModStore.lua:143-149
func cfgKeywordFlags(cfg *ListCfg) mod.KeywordFlag {
	if cfg == nil {
		return 0
	}
	return cfg.KeywordFlags.Get()
}

type multiplierSource interface {
	GetMultiplier(variable string, cfg *ListCfg, noMod bool) float64
}

type statSource interface {
	getStat(stat string, cfg *ListCfg) float64
}

type Actor interface {
	GetOutput(string) (float64, bool)
}

type ActorLinks interface {
	RelatedModDB(name string) ModStoreFuncs
}

type MinionMonsterTags interface {
	MonsterTags() (tags []string, ok bool)
}

type ItemQueries interface {
	EquippedSlots() []string
	ItemInSlot(slot string) bool
	ItemModifierMatches(slot, searchCond string) bool
	ItemRarityIs(slot, rarity string) bool
	ItemCorruptedIs(slot string, corrupted bool) bool
	ItemNameIs(slot, name string) bool
}

type ItemInfluenceQueries interface {
	ItemShaperIs(slot string, shaper bool) bool
	ItemElderIs(slot string, elder bool) bool
}

type CfgItemInfluence interface {
	CfgShaperIs(shaper bool) bool
	CfgElderIs(elder bool) bool
}

type AllOneHandedWeapon interface {
	AddedUsingCondition(v string) (bool, bool)
}

type CfgItem interface {
	CfgNameIs(name string) bool
	CfgRarityIs(rarity string) bool
	CfgCorruptedIs(corrupted bool) bool
	CfgModifierMatches(searchCond, slot string) bool
}

type ReservingAuras interface {
	AuraManaReservedBase(skillName, summonSkillName string) (float64, bool)

	AuraLifeReservedBase(skillName, summonSkillName string) (float64, bool)
}

type ModStore struct {
	Parent      ModStoreFuncs
	Child       ModStoreFuncs
	Actor       Actor
	Multipliers map[string]float64
	Conditions  map[string]bool
}

//pob:src/Classes/ModStore.lua:30-35
func NewModStore(parent ModStoreFuncs) *ModStore {
	return &ModStore{
		Parent:      parent,
		Multipliers: make(map[string]float64),
		Conditions:  make(map[string]bool),
	}
}

func SetCondition(s *ModStore, name string, value bool) {
	if s.Conditions == nil {
		s.Conditions = make(map[string]bool)
	}
	s.Conditions[name] = value
}

func (s *ModStore) Clone() *ModStore {
	if s == nil {
		return nil
	}

	var parent ModStoreFuncs
	if s.Parent != nil {
		parent = s.Parent.Clone()
	}

	out := NewModStore(parent)
	out.Actor = s.Actor
	maps.Copy(out.Multipliers, s.Multipliers)
	maps.Copy(out.Conditions, s.Conditions)
	return out
}

//pob:block:src/Classes/ModStore.lua:32
func (s *ModStore) GetActor() Actor {
	if s.Actor != nil {
		return s.Actor
	}

	if s.Parent != nil {
		return s.Parent.GetActor()
	}

	return nil
}

//pob:block:src/Classes/ModStore.lua:37-43
func (s *ModStore) relatedModDB(name string) ModStoreFuncs {
	links, ok := s.GetActor().(ActorLinks)
	if !ok {
		return nil
	}
	return links.RelatedModDB(name)
}

type conditionSource interface {
	GetCondition(variable string, cfg *ListCfg, noMod bool) (bool, bool)
}

//pob:block:src/Classes/ModStore.lua:349
func tagDiv(div float64) float64 {
	if div == 0 {
		return 1
	}
	return div
}

//pob:block:src/Classes/ModStore.lua:326-400
func (s *ModStore) evalMultiplier(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.MultiplierTag) *mod.ModValueMulti {
	var target multiplierSource = s
	var limitTarget multiplierSource = s

	// Allow limiting a self multiplier on a parent multiplier (eg. Agony Crawler on player virulence)
	// This explicit target is necessary because even though the GetMultiplier method does call self.parent.GetMultiplier, it does so with noMod = true,
	// disabling the summation (3rd part): (not noMod and self:Sum("BASE", cfg, multiplierName[var]) or 0)
	if tag.TagLimitActor != "" {
		db := s.relatedModDB(tag.TagLimitActor)
		if db == nil {
			return nil
		}
		limitTarget = db
	}

	if tag.TagActor != "" {
		db := s.relatedModDB(tag.TagActor)
		if db == nil {
			return nil
		}
		target = db
	}

	base := 0.0
	for _, v := range tag.VariableList {
		base += target.GetMultiplier(v, cfg, false)
	}

	div := tagDiv(tag.Division)
	if tag.TagDivVariable != nil {
		div = tagDiv(s.GetMultiplier(*tag.TagDivVariable, cfg, false))
	}

	mult := math.Floor(base/div + 0.0001)
	if tag.TagNoFloor {
		mult = base / div
	}

	var limitTotal *float64
	var limitNegTotal *float64
	if tag.TagLimit != nil || tag.TagLimitVariable != nil || tag.TagLimitStat != nil {
		limit := float64(0)
		switch {
		case tag.TagLimit != nil:
			limit = *tag.TagLimit
		case tag.TagLimitVariable != nil:
			limit = limitTarget.GetMultiplier(*tag.TagLimitVariable, cfg, false)
		default:
			if ss, ok := limitTarget.(statSource); ok {
				limit = ss.getStat(*tag.TagLimitStat, cfg)
			}
		}

		switch {
		case tag.TagLimitTotal:
			limitTotal = &limit
		case tag.TagLimitNegTotal:
			limitNegTotal = &limit
		default:
			mult = min(mult, limit)
		}
	}

	if value.Type() == mod.ModValueMultiTypeFloat {
		out := float64(value.Float()*mult) + tag.TagBase
		if limitTotal != nil {
			out = min(out, *limitTotal)
		}
		if limitNegTotal != nil {
			out = max(out, *limitNegTotal)
		}
		value = value.Clone()
		value.ValueFloat = out
	} else if scalable, ok := value.List().(mod.ScalableValue); ok {
		scaled := scalable.ScaleValue(mult, tag.TagBase, limitTotal)

		if reflect.TypeOf(value.List()).Kind() == reflect.Pointer {
			scaled = scaled.Clone(true)
		}

		value = value.Clone()
		value.SetList(scaled)
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:314-334
func (s *ModStore) evalMultiplierThresholdTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.MultiplierThresholdTag) *mod.ModValueMulti {
	var target multiplierSource = s
	if tag.TagActor != "" {
		db := s.relatedModDB(tag.TagActor)
		if db == nil {
			return nil
		}
		target = db
	}

	mult := float64(0)
	if len(tag.VariableList) > 0 {
		for _, v := range tag.VariableList {
			mult += target.GetMultiplier(v, cfg, false)
		}
	} else {
		mult = target.GetMultiplier(tag.Variable, cfg, false)
	}

	threshold := float64(0)
	if tag.TagThreshold != nil {
		threshold = *tag.TagThreshold
	} else {
		threshold = target.GetMultiplier(*tag.ThresholdVariable, cfg, false)
	}

	if (tag.TagUpper && mult > threshold) || (!tag.TagUpper && mult < threshold) {
		return nil
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:335-379
func (s *ModStore) evalPerStatTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.PerStatTag) *mod.ModValueMulti {
	base := float64(0)
	var target statSource = s

	// This functions similar to the above tagTypes in regard to which actor to use, but for PerStat
	// if the actor is 'parent', we don't want to return if we're already using 'parent', just keep using 'self'
	if tag.TagActor != "" {
		if db := s.relatedModDB(tag.TagActor); db != nil {
			target = db
		}
	}

	for _, stat := range tag.StatList {
		base += target.getStat(stat, cfg)
	}

	mult := math.Floor(base/tagDiv(tag.Divide) + 0.0001)
	var limitTotal *float64
	if tag.TagLimit != nil || tag.TagLimitVariable != nil {
		limit := float64(0)
		if tag.TagLimit != nil {
			limit = *tag.TagLimit
		} else {
			limit = s.GetMultiplier(*tag.TagLimitVariable, cfg, false)
		}

		if tag.TagLimitTotal {
			limitTotal = &limit
		} else {
			mult = min(mult, limit)
		}
	}

	if value.Type() == mod.ModValueMultiTypeFloat {
		out := float64(value.Float()*mult) + tag.Base
		if limitTotal != nil {
			out = min(out, *limitTotal)
		}
		value = value.Clone()
		value.ValueFloat = out
	} else if scalable, ok := value.List().(mod.ScalableValue); ok {
		scaled := scalable.ScaleValue(mult, tag.Base, limitTotal)

		if reflect.TypeOf(value.List()).Kind() == reflect.Pointer {
			scaled = scaled.Clone(true)
		}

		value = value.Clone()
		value.SetList(scaled)
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:588-589
func (s *ModStore) evalLimitTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.LimitTag) *mod.ModValueMulti {
	limit := tag.TagLimit
	if tag.LimitVar != "" {
		limit = s.GetMultiplier(tag.LimitVar, cfg, false)
	}

	if value.Type() == mod.ModValueMultiTypeFloat {
		value = value.Clone()
		value.ValueFloat = min(value.ValueFloat, limit)
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:582-620
func (s *ModStore) evalConditionTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.ConditionTag) *mod.ModValueMulti {
	match := false

	allOneH, _ := s.GetActor().(AllOneHandedWeapon)

	for _, v := range tag.VarList {
		if tag.Negative && allOneH != nil {
			if added, present := allOneH.AddedUsingCondition(v); present {
				if !added {
					return nil
				}
				continue
			}
		}
		cond, _ := s.GetCondition(v, cfg, false)
		if !cond && cfg != nil && cfg.SkillCond != nil {
			cond = cfg.SkillCond[v]
		}
		if cond {
			match = true
			break
		}
	}

	if tag.Negative {
		match = !match
	}

	if !match {
		return nil
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:621-641
func (s *ModStore) evalActorConditionTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.ActorConditionTag) *mod.ModValueMulti {
	var target conditionSource = s

	if tag.Actor != nil {
		target = nil
		if db := s.relatedModDB(*tag.Actor); db != nil {
			target = db
		}
	}

	match := false
	switch {
	case target != nil && len(tag.VariableList) > 0:
		for _, v := range tag.VariableList {
			if c, _ := target.GetCondition(v, cfg, false); c {
				match = true
				break
			}
		}
	case tag.Actor != nil && cfg != nil && *tag.Actor == cfg.Actor:
		match = true
	}

	if tag.Negative {
		match = !match
	}

	if !match {
		return nil
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:567-585
func (s *ModStore) evalSkillNameTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.SkillNameTag) *mod.ModValueMulti {
	matchName := ""
	if cfg != nil {
		if tag.TagSummonSkill {
			matchName = cfg.SummonSkillName
		} else {
			matchName = cfg.SkillName
		}
	}

	if tag.IncludeTransfigured {
		matchGameID := gameIDFromGemName(matchName)

		match := false
		if matchGameID != "" {
			for _, name := range tag.SkillNameList {
				if gameIDFromGemName(name) == matchGameID {
					match = true
					break
				}
			}
		}
		if tag.Negative {
			match = !match
		}
		if !match {
			return nil
		}
		return value
	}
	matchName = strings.ToLower(matchName)

	match := false
	for _, name := range tag.SkillNameList {
		if strings.ToLower(name) == matchName {
			match = true
			break
		}
	}
	if tag.Negative {
		match = !match
	}
	if !match {
		return nil
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:795-798
func (s *ModStore) evalSkillIDTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.SkillIDTag) *mod.ModValueMulti {
	if cfg == nil || cfg.SkillGrantedEffect == nil {
		return nil
	}
	ge, ok := cfg.SkillGrantedEffect.(interface{ ID() string })
	if !ok || ge.ID() != tag.ID() {
		return nil
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:799-819
func (s *ModStore) evalSkillPartTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.SkillPartTag) *mod.ModValueMulti {
	if cfg == nil {
		return nil
	}

	match := false
	if tag.PartList != nil {
		for _, part := range tag.PartList {
			if part == cfg.SkillPart {
				match = true
				break
			}
		}
	} else {
		match = tag.Part == cfg.SkillPart
	}

	if tag.Negative {
		match = !match
	}

	if !match {
		return nil
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:820-837
func (s *ModStore) evalSkillTypeTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.SkillTypeTag) *mod.ModValueMulti {
	match := false
	if cfg != nil && cfg.SkillTypes != nil {
		for _, skillType := range tag.SkillTypeList {
			if cfg.SkillTypes[data.SkillType(skillType)] {
				match = true
				break
			}
		}
	}
	if tag.Negative {
		match = !match
	}
	if !match {
		return nil
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:847-867
func (s *ModStore) evalSlotNameTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.SlotNameTag) *mod.ModValueMulti {
	if cfg == nil {
		return nil
	}
	match := false
	for _, slot := range tag.SlotNameList {
		if slot == cfg.SlotName {
			match = true
			break
		}
	}
	if !match {
		return nil
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:648-718
func (s *ModStore) evalItemConditionTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.ItemConditionTag) *mod.ModValueMulti {
	queries, ok := s.GetActor().(ItemQueries)
	if !ok {
		return nil
	}

	itemSlot := titleCaseSlot(tag.ItemSlot)

	var slots []string
	var cfgItem CfgItem
	switch {
	case tag.AllSlots:
		slots = queries.EquippedSlots()
	case tag.BothSlots:
		one, two := itemSlot+" 1", itemSlot+" 2"
		if queries.ItemInSlot(one) && queries.ItemInSlot(two) {
			src1, src2 := one, two
			if queries.ItemNameIs(one, kalandrasTouch) {
				src1 = src2
			}
			if queries.ItemNameIs(two, kalandrasTouch) {
				src2 = src1
			}
			slots = []string{src1, src2}
		}
	default:
		if queries.ItemInSlot(itemSlot) {
			if queries.ItemNameIs(itemSlot, kalandrasTouch) {
				if opposite := oppositeRingSlot(itemSlot); queries.ItemInSlot(opposite) {
					slots = []string{opposite}
				} else if cfg != nil && cfg.Item != nil {
					cfgItem = cfg.Item
				}
			} else {
				slots = []string{itemSlot}
			}
		} else if cfg != nil && cfg.Item != nil {
			cfgItem = cfg.Item
		}
	}

	match := true
	for _, slot := range slots {
		if tag.ExcludeSelf && slot == itemSlot {
			continue
		}
		if tag.SearchCond != "" {
			if queries.ItemModifierMatches(slot, strings.ToLower(tag.SearchCond)) == tag.Neg {
				match = false
				break
			}
		}
		if tag.RarityCond != "" {
			if queries.ItemRarityIs(slot, tag.RarityCond) == tag.Neg {
				match = false
				break
			}
		}
		if tag.CorruptedCond != nil {
			if queries.ItemCorruptedIs(slot, *tag.CorruptedCond) == tag.Neg {
				match = false
				break
			}
		}
		//pob:block:src/Classes/ModStore.lua:688-697
		if tag.ShaperCond || tag.ElderCond {
			influence, ok := queries.(ItemInfluenceQueries)
			if !ok || (tag.ShaperCond && influence.ItemShaperIs(slot, true) == tag.Neg) || (tag.ElderCond && influence.ItemElderIs(slot, true) == tag.Neg) {
				match = false
				break
			}
		}
		if tag.NameCond != "" {
			if queries.ItemNameIs(slot, tag.NameCond) == tag.Neg {
				match = false
				break
			}
		}
	}

	if match && cfgItem != nil {
		if tag.SearchCond != "" && cfgItem.CfgModifierMatches(strings.ToLower(tag.SearchCond), strings.ToLower(itemSlot)) == tag.Neg {
			match = false
		}
		if match && tag.RarityCond != "" && cfgItem.CfgRarityIs(tag.RarityCond) == tag.Neg {
			match = false
		}
		if match && tag.CorruptedCond != nil && cfgItem.CfgCorruptedIs(*tag.CorruptedCond) == tag.Neg {
			match = false
		}
		if match && (tag.ShaperCond || tag.ElderCond) {
			influence, ok := cfgItem.(CfgItemInfluence)
			if !ok || (tag.ShaperCond && influence.CfgShaperIs(true) == tag.Neg) || (tag.ElderCond && influence.CfgElderIs(true) == tag.Neg) {
				match = false
			}
		}
		if match && tag.NameCond != "" && cfgItem.CfgNameIs(tag.NameCond) == tag.Neg {
			match = false
		}
	}

	if !match || (len(slots) == 0 && cfgItem == nil && !tag.Neg) {
		return nil
	}
	return value
}

const kalandrasTouch = "Kalandra's Touch, Ring"

func oppositeRingSlot(slot string) string {
	switch {
	case strings.HasSuffix(slot, "1"):
		return slot[:len(slot)-1] + "2"
	case strings.HasSuffix(slot, "2"):
		return slot[:len(slot)-1] + "1"
	}
	return slot
}

func titleCaseSlot(slot string) string {
	words := strings.Fields(strings.ToLower(slot))
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// elseif tag.socketColor and tag.sockets ~= "all" then
//
//pob:block:src/Classes/ModStore.lua:719-760
func (s *ModStore) evalSocketedInTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.SocketedInTag) *mod.ModValueMulti {
	if cfg == nil || (tag.SlotName == "" && tag.TagKeyword == "" && tag.TagSocketColor == "") {
		return nil
	}
	if tag.SlotName != "" && tag.SlotName != cfg.SlotName {
		return nil
	}
	switch {
	case tag.TagKeyword != "":
		if cfg.SkillGem == nil || GemIsType == nil || !GemIsType(cfg.SkillGem, tag.TagKeyword) {
			return nil
		}
	case tag.TagSocketColor != "":
		if tag.TagSocketColor != cfg.SocketColor {
			return nil
		}
	}
	if len(tag.TagSockets) > 0 {
		if cfg.SocketNum == 0 || !slices.Contains(tag.TagSockets, cfg.SocketNum) {
			return nil
		}
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:868-874
func (s *ModStore) evalModFlagOrTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.ModFlagOrTag) *mod.ModValueMulti {
	if cfg == nil || cfg.Flags == nil {
		return nil
	}
	if (*cfg.Flags)&tag.Flag == 0 {
		return nil
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:875-881
func (s *ModStore) evalKeywordFlagAndTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.KeywordFlagAndTag) *mod.ModValueMulti {
	if cfg == nil || cfg.KeywordFlags == nil {
		return nil
	}
	if (*cfg.KeywordFlags)&tag.KeywordFlags != tag.KeywordFlags {
		return nil
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:419-432
func (s *ModStore) evalStatThresholdTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.StatThresholdTag) *mod.ModValueMulti {
	stat := s.getStat(tag.Stat, cfg)
	threshold := tag.Threshold
	if tag.TagThresholdStat != "" {
		threshold = s.getStat(tag.TagThresholdStat, cfg)
	}
	if tag.ThresholdPercent != 0 {
		threshold *= tag.ThresholdPercent / 100
	}
	if (tag.TagUpper && stat > threshold) || (!tag.TagUpper && stat < threshold) {
		return nil
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:380-418
func (s *ModStore) evalPercentStatTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.PercentStatTag) *mod.ModValueMulti {
	var target statSource = s
	if tag.TagActor != "" {
		if db := s.relatedModDB(tag.TagActor); db != nil {
			target = db
		}
	}

	base := target.getStat(tag.Stat, cfg)
	//	local percent = tag.percent or GetMultiplier(self, tag.percentVar, cfg)
	//	local mult = base * (percent and percent / 100 or 1)
	//pob:block:src/Classes/ModStore.lua:504-505
	percent := tag.Percent
	if percent == 0 && tag.PercentVar != "" {
		percent = s.GetMultiplier(tag.PercentVar, cfg, false)
	}
	mult := base * (percent / 100)
	if tag.TagFloor {
		mult = math.Floor(mult)
	}

	if value.Type() == mod.ModValueMultiTypeFloat {
		value = value.Clone()
		value.ValueFloat = math.Ceil(value.ValueFloat * mult)
	} else if scalable, ok := value.List().(mod.ScalableValue); ok {
		//pob:block:src/Classes/ModStore.lua:518-530
		scaled := ceilScaledValue(scalable.ScaleValue(mult, 0, nil))

		if reflect.TypeOf(value.List()).Kind() == reflect.Pointer {
			scaled = scaled.Clone(true)
		}

		value = value.Clone()
		value.SetList(scaled)
	}
	return value
}

func ceilScaledValue(scaled mod.Cloneable) mod.Cloneable {
	switch v := scaled.(type) {
	case mod.ScalableNestedMod:
		return v.ScaleNestedMod(func(_ mod.Mod, x float64) float64 { return math.Ceil(x) })
	case mod.SkillData:
		v.Value = math.Ceil(v.Value)
		return v
	case mod.GemProperty:
		v.Value = math.Ceil(v.Value)
		return v
	}
	return scaled
}

//pob:block:src/Classes/ModStore.lua:555-571
func (s *ModStore) evalDistanceRampTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.DistanceRampTag) *mod.ModValueMulti {
	if cfg == nil || !cfg.SkillDistSet || len(tag.Ramp) == 0 {
		return nil
	}

	ramp := tag.Ramp
	var mult float64
	switch {
	case cfg.SkillDist <= ramp[0][0]:
		mult = ramp[0][1]
	case cfg.SkillDist >= ramp[len(ramp)-1][0]:
		mult = ramp[len(ramp)-1][1]
	default:
		for i := 0; i < len(ramp)-1; i++ {
			next := ramp[i+1]
			if cfg.SkillDist <= next[0] {
				dat := ramp[i]
				mult = dat[1] + (next[1]-dat[1])*(cfg.SkillDist-dat[0])/(next[0]-dat[0])
				break
			}
		}
	}

	if value.Type() == mod.ModValueMultiTypeFloat {
		value = value.Clone()
		value.ValueFloat = value.ValueFloat * mult
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:575-587
func (s *ModStore) evalMeleeProximityTag(value *mod.ModValueMulti, cfg *ListCfg, tag *mod.MeleeProximityTag) *mod.ModValueMulti {
	if cfg == nil || !cfg.SkillDistSet || len(tag.Ramp) == 0 {
		return nil
	}

	r0 := float64(tag.Ramp[0])
	var mult float64
	mult = 1
	switch {
	case cfg.SkillDist <= 15:
		mult = r0
	case cfg.SkillDist >= 16 && cfg.SkillDist <= 39:
		mult = r0 - float64((r0/25)*(cfg.SkillDist-15))
	case cfg.SkillDist >= 40:
		mult = 0
	}

	if value.Type() == mod.ModValueMultiTypeFloat {
		value = value.Clone()
		value.ValueFloat = value.ValueFloat * mult
	}
	return value
}

//pob:block:src/Classes/ModStore.lua:249-603
func (s *ModStore) evalTag(raw mod.Tag, value *mod.ModValueMulti, cfg *ListCfg) *mod.ModValueMulti {
	switch tag := raw.(type) {
	case *mod.MultiplierTag:
		value = s.evalMultiplier(value, cfg, tag)
	case *mod.MultiplierThresholdTag:
		value = s.evalMultiplierThresholdTag(value, cfg, tag)
	case *mod.PerStatTag:
		value = s.evalPerStatTag(value, cfg, tag)
	case *mod.PercentStatTag:
		value = s.evalPercentStatTag(value, cfg, tag)
	case *mod.DistanceRampTag:
		value = s.evalDistanceRampTag(value, cfg, tag)
	case *mod.MeleeProximityTag:
		value = s.evalMeleeProximityTag(value, cfg, tag)
	case *mod.LimitTag:
		value = s.evalLimitTag(value, cfg, tag)
	case *mod.ConditionTag:
		value = s.evalConditionTag(value, cfg, tag)
	case *mod.ActorConditionTag:
		value = s.evalActorConditionTag(value, cfg, tag)
	case *mod.SkillNameTag:
		value = s.evalSkillNameTag(value, cfg, tag)
	case *mod.SkillIDTag:
		value = s.evalSkillIDTag(value, cfg, tag)
	case *mod.SkillPartTag:
		value = s.evalSkillPartTag(value, cfg, tag)
	case *mod.SkillTypeTag:
		value = s.evalSkillTypeTag(value, cfg, tag)
	case *mod.SlotNameTag:
		value = s.evalSlotNameTag(value, cfg, tag)
	case *mod.ItemConditionTag:
		value = s.evalItemConditionTag(value, cfg, tag)
	case *mod.SocketedInTag:
		value = s.evalSocketedInTag(value, cfg, tag)
	case *mod.ModFlagOrTag:
		value = s.evalModFlagOrTag(value, cfg, tag)
	case *mod.KeywordFlagAndTag:
		value = s.evalKeywordFlagAndTag(value, cfg, tag)
	case *mod.StatThresholdTag:
		value = s.evalStatThresholdTag(value, cfg, tag)
	case *mod.MonsterTagTag:
		value = s.evalMonsterTagTag(value, tag)
	}

	return value
}

//pob:block:src/Classes/ModStore.lua:882-913
func (s *ModStore) evalMonsterTagTag(value *mod.ModValueMulti, tag *mod.MonsterTagTag) *mod.ModValueMulti {
	src, ok := s.GetActor().(MinionMonsterTags)
	if !ok {
		return nil
	}
	monsterTags, ok := src.MonsterTags()
	if !ok {
		return nil
	}
	match := false
	for _, t := range monsterTags {
		matchName := strings.ToLower(t)
		for _, name := range tag.MonsterTagList {
			if strings.ToLower(name) == matchName {
				match = true
				break
			}
		}
		if match {
			break
		}
	}
	if tag.Negative {
		match = !match
	}
	if !match {
		return nil
	}
	return value
}

func (s *ModStore) EvalMod(m mod.Mod, cfg *ListCfg) *mod.ModValueMulti {
	return s.evalMod(m, cfg)
}

//pob:src/Classes/ModStore.lua:246-606
func (s *ModStore) evalMod(m mod.Mod, cfg *ListCfg) *mod.ModValueMulti {
	value := m.Value()

	for _, raw := range m.Tags() {
		value = s.evalTag(raw, value, cfg)
		if value == nil {
			return nil
		}
	}

	return value
}

//pob:block:src/Classes/ModStore.lua:916-926
func tagGlobalLimits(m mod.Mod) []globalLimitPair {
	var out []globalLimitPair

	for _, raw := range m.Tags() {
		switch tag := raw.(type) {
		case *mod.MultiplierTag:
			if tag.TagGlobalLimit != nil && tag.TagGlobalLimitKey != nil {
				out = append(out, globalLimitPair{limit: *tag.TagGlobalLimit, key: *tag.TagGlobalLimitKey})
			}
		case *mod.PerStatTag:
			if tag.TagGlobalLimit != 0 && tag.TagGlobalLimitKey != "" {
				out = append(out, globalLimitPair{limit: tag.TagGlobalLimit, key: tag.TagGlobalLimitKey})
			}
		}
	}

	return out
}

type globalLimitPair struct {
	key   string
	limit float64
}

func applyGlobalLimits(globalLimits *map[string]float64, pairs []globalLimitPair, value float64) float64 {
	if len(pairs) == 0 {
		return value
	}
	if *globalLimits == nil {
		*globalLimits = make(map[string]float64)
	}
	for _, p := range pairs {
		value = applyGlobalLimit(*globalLimits, p.key, p.limit, value)
	}
	return value
}

//pob:block:src/Classes/ModDB.lua:268-273
func applyGlobalLimit(globalLimits map[string]float64, key string, limit float64, value float64) float64 {
	if globalLimits[key]+value > limit {
		value = limit - globalLimits[key]
	}
	globalLimits[key] += value
	return value
}

type nameTable struct {
	prefix string
	names  atomic.Pointer[map[string]string]
	mu     sync.Mutex
}

func (t *nameTable) of(variable string) string {
	if names := t.names.Load(); names != nil {
		if name, ok := (*names)[variable]; ok {
			return name
		}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	old := t.names.Load()
	if old != nil {
		if name, ok := (*old)[variable]; ok {
			return name
		}
	}
	next := make(map[string]string, 1)
	if old != nil {
		next = maps.Clone(*old)
	}
	name := t.prefix + variable
	next[variable] = name
	t.names.Store(&next)
	return name
}

//pob:block:src/Classes/ModStore.lua:21-28
var (
	multiplierName = &nameTable{prefix: "Multiplier:"}
	conditionName  = &nameTable{prefix: "Condition:"}
)

//pob:src/Classes/ModStore.lua:265-267
func (s *ModStore) GetMultiplier(variable string, cfg *ListCfg, noMod bool) float64 {
	//	return (not noMod and self:Override(cfg, multiplierName[var]))
	//	    or (self.multipliers[var] or 0) + (parent...) + (not noMod and self:Sum("BASE", ...) or 0)
	if !noMod {
		if override := overrideIn(s.Child, s.Child, cfg, multiplierName.of(variable)); override != nil {
			return override.Float()
		}
	}

	out := float64(0)

	if mul, ok := s.Multipliers[variable]; ok {
		out += mul
	}

	if s.Parent != nil {
		out += s.Parent.GetMultiplier(variable, cfg, true)
	}

	if !noMod {
		out += sumIn(s.Child, s.Child, mod.TypeBase, cfg, multiplierName.of(variable))
	}

	return out
}

var StatReadProbe func(actor any, stat string, value float64, found bool)

//pob:src/Classes/ModStore.lua:269-318
func (s *ModStore) getStat(stat string, cfg *ListCfg) float64 {
	if stat == "ManaReservedPercent" {
		reserved := float64(0)
		if reserving, ok := s.GetActor().(ReservingAuras); ok && cfg != nil {
			if base, found := reserving.AuraManaReservedBase(cfg.SkillName, cfg.SummonSkillName); found {
				if mana, has := s.GetActor().GetOutput("Mana"); has && mana != 0 {
					reserved = math.Floor(base / mana * 100)
				}
			}
		}
		return math.Min(reserved, 100)
	}

	if stat == "LifeReservedPercent" {
		reserved := float64(0)
		if reserving, ok := s.GetActor().(ReservingAuras); ok && cfg != nil {
			if base, found := reserving.AuraLifeReservedBase(cfg.SkillName, cfg.SummonSkillName); found {
				if life, has := s.GetActor().GetOutput("Life"); has && life != 0 {
					reserved = math.Floor(base / life * 100)
				}
			}
		}
		return math.Min(reserved, 100)
	}

	if actor := s.GetActor(); actor != nil {
		v, ok := actor.GetOutput(stat)
		if StatReadProbe != nil {
			StatReadProbe(actor, stat, v, ok)
		}
		if ok {
			if stat == "ManaUnreserved" && math.IsNaN(v) {
				if mana, ok := actor.GetOutput("Mana"); ok {
					return mana
				}
			}
			if stat == "ManaUnreserved" && v < 0 {
				mana, hasMana := actor.GetOutput("Mana")
				pct, _ := actor.GetOutput("ManaUnreservedPercent")
				eff, _ := actor.GetOutput("ManaEfficiency")
				if hasMana {
					reservedPercentBeforeEfficiency := (math.Abs(pct) + 100) * ((100 + eff) / 100)
					return mana * (math.Ceil(reservedPercentBeforeEfficiency) / 100)
				}
			}
			if v != 0 {
				return v
			}
		}
	}

	if cfg != nil && cfg.SkillStats != nil {
		if v, ok := cfg.SkillStats[stat]; ok {
			return v
		}
	}

	return 0
}

//pob:src/Classes/ModStore.lua:261-263
func (s *ModStore) GetCondition(variable string, cfg *ListCfg, noMod bool) (bool, bool) {
	if cond, ok := s.Conditions[variable]; ok && cond {
		return true, true
	}

	if s.Parent != nil {
		if out, _ := s.Parent.GetCondition(variable, cfg, true); out {
			return true, true
		}
	}

	if !noMod && flagIn(s.Child, s.Child, cfg, conditionName.of(variable)) {
		return true, true
	}

	return false, false
}

var (
	gemGameIDByBaseNameOnce sync.Once
	gemGameIDByBaseName     map[string]string
)

//pob:src/Modules/CalcTools.lua:270-283
func gameIDFromGemName(name string) string {
	if name == "" {
		return ""
	}
	gemGameIDByBaseNameOnce.Do(func() {
		gemGameIDByBaseName = make(map[string]string, len(raw.Gems)+len(raw.GemVariants))
		add := func(gemName, gameID string) {
			key := strings.ToLower(gemName)
			if prev, ok := gemGameIDByBaseName[key]; !ok || gameID < prev {
				gemGameIDByBaseName[key] = gameID
			}
		}
		resolve := func(row raw.PoBGem, gameID string) string {
			if row.Tags["vaal"] && row.SecondaryGrantedEffectID != "" {
				if base := raw.GemGameIDForGrantedEffectID[row.SecondaryGrantedEffectID]; base != "" {
					return base
				}
			}
			return gameID
		}
		for _, gameID := range slices.Sorted(maps.Keys(raw.Gems)) {
			row := raw.Gems[gameID]
			if row.Name != "" {
				add(row.Name, resolve(row, gameID))
			}
		}
		for _, effectID := range slices.Sorted(maps.Keys(raw.GemVariants)) {
			row := raw.GemVariants[effectID]
			if gameID := raw.GemGameIDForGrantedEffectID[effectID]; row.Name != "" && gameID != "" {
				add(row.Name, resolve(row, gameID))
			}
		}
	})
	return gemGameIDByBaseName[strings.ToLower(name)]
}
