package mod

import "reflect"

type Cloneable interface {
	Clone(pointer bool) Cloneable
}

type ScalableValue interface {
	ScaleValue(mult, base float64, limitTotal *float64) Cloneable
}

type ScalableNestedMod interface {
	ScaleNestedMod(scale func(Mod, float64) float64) Cloneable
}

func scaleNestedModValue(m Mod, scale func(Mod, float64) float64) Mod {
	scaled := m.Clone()
	if v := scaled.Value(); v != nil && v.Type() == ModValueMultiTypeFloat {
		v.SetFloat(scale(scaled, v.Float()))
	}
	return scaled
}

//pob:src/Modules/ModTools.lua:217-223
func SetSource(m Mod, source Source) Mod {
	m.Source(source)
	if v := m.Value(); v != nil && v.Type() == ModValueMultiTypeList && v.ValueList != nil {
		rv := reflect.Indirect(reflect.ValueOf(v.ValueList))
		if rv.Kind() == reflect.Struct {
			if field := rv.FieldByName("Mod"); field.IsValid() && !field.IsNil() {
				if nested, ok := field.Interface().(Mod); ok {
					nested.Source(source)
				}
			}
		}
	}
	return m
}

type GemProperty struct {
	Key         string
	Value       float64
	KeywordList []string
	Keyword     *string
}

func (g GemProperty) ScaleValue(mult, base float64, limitTotal *float64) Cloneable {
	out := g.Value*mult + base
	if limitTotal != nil {
		out = min(out, *limitTotal)
	}

	scaled := g.Clone(false).(GemProperty)
	scaled.Value = out
	return scaled
}

func (g GemProperty) Clone(pointer bool) Cloneable {
	out := make([]string, len(g.KeywordList))
	copy(out, g.KeywordList)

	obj := GemProperty{
		Key:         g.Key,
		Value:       g.Value,
		KeywordList: out,
		Keyword:     g.Keyword,
	}

	if pointer {
		return &obj
	}

	return obj
}

type SkillData struct {
	Key   string
	Value float64
	Merge string
}

func (s SkillData) ScaleValue(mult, base float64, limitTotal *float64) Cloneable {
	out := s.Value*mult + base
	if limitTotal != nil {
		out = min(out, *limitTotal)
	}

	return SkillData{
		Key:   s.Key,
		Value: out,
		Merge: s.Merge,
	}
}

func (s SkillData) Clone(pointer bool) Cloneable {
	obj := SkillData{
		Key:   s.Key,
		Value: s.Value,
		Merge: s.Merge,
	}

	if pointer {
		return &obj
	}

	return obj
}

type ExtraMinionSkill struct {
	SkillID string
}

func (s ExtraMinionSkill) Clone(pointer bool) Cloneable {
	obj := ExtraMinionSkill{
		SkillID: s.SkillID,
	}

	if pointer {
		return &obj
	}

	return obj
}

type SocketProperty struct {
	Mod Mod
}

func (s SocketProperty) ScaleNestedMod(scale func(Mod, float64) float64) Cloneable {
	return SocketProperty{Mod: scaleNestedModValue(s.Mod, scale)}
}

func (s SocketProperty) Clone(pointer bool) Cloneable {
	obj := SocketProperty{
		Mod: s.Mod.Clone(),
	}

	if pointer {
		return &obj
	}

	return obj
}

type ExtraAuraEffect struct {
	Mod Mod
}

func (s ExtraAuraEffect) ScaleNestedMod(scale func(Mod, float64) float64) Cloneable {
	return ExtraAuraEffect{Mod: scaleNestedModValue(s.Mod, scale)}
}

func (s ExtraAuraEffect) Clone(pointer bool) Cloneable {
	obj := ExtraAuraEffect{
		Mod: s.Mod.Clone(),
	}

	if pointer {
		return &obj
	}

	return obj
}

type ExtraAura struct {
	Mod        Mod
	OnlyAllies bool
	FromAllies bool
}

func (s ExtraAura) ScaleValue(mult, base float64, limitTotal *float64) Cloneable {
	scaled := s.Mod.Clone()
	if v := scaled.Value(); v != nil && v.Type() == ModValueMultiTypeFloat {
		out := v.Float()*mult + base
		if limitTotal != nil {
			out = min(out, *limitTotal)
		}
		v.SetFloat(out)
	}
	return ExtraAura{Mod: scaled, OnlyAllies: s.OnlyAllies, FromAllies: s.FromAllies}
}

func (s ExtraAura) ScaleNestedMod(scale func(Mod, float64) float64) Cloneable {
	return ExtraAura{Mod: scaleNestedModValue(s.Mod, scale), OnlyAllies: s.OnlyAllies, FromAllies: s.FromAllies}
}

func (s ExtraAura) Clone(pointer bool) Cloneable {
	obj := ExtraAura{
		Mod:        s.Mod.Clone(),
		OnlyAllies: s.OnlyAllies,
		FromAllies: s.FromAllies,
	}

	if pointer {
		return &obj
	}

	return obj
}

type AffectedByAuraMod struct {
	Mod Mod
}

func (s AffectedByAuraMod) ScaleNestedMod(scale func(Mod, float64) float64) Cloneable {
	return AffectedByAuraMod{Mod: scaleNestedModValue(s.Mod, scale)}
}

func (s AffectedByAuraMod) Clone(pointer bool) Cloneable {
	obj := AffectedByAuraMod{
		Mod: s.Mod.Clone(),
	}

	if pointer {
		return &obj
	}

	return obj
}

type MinionModifier struct {
	Mod Mod
}

type AllyModifier = MinionModifier

type PlayerModifier = MinionModifier

func (s MinionModifier) ScaleValue(mult, base float64, limitTotal *float64) Cloneable {
	scaled := s.Mod.Clone()
	if v := scaled.Value(); v != nil && v.Type() == ModValueMultiTypeFloat {
		out := v.Float()*mult + base
		if limitTotal != nil {
			out = min(out, *limitTotal)
		}
		v.SetFloat(out)
	}
	return MinionModifier{Mod: scaled}
}

func (s MinionModifier) ScaleNestedMod(scale func(Mod, float64) float64) Cloneable {
	return MinionModifier{Mod: scaleNestedModValue(s.Mod, scale)}
}

func (s MinionModifier) Clone(pointer bool) Cloneable {
	obj := MinionModifier{
		Mod: s.Mod.Clone(),
	}

	if pointer {
		return &obj
	}

	return obj
}

type PactDamage struct {
	Mod Mod
}

func (s PactDamage) ScaleNestedMod(scale func(Mod, float64) float64) Cloneable {
	return PactDamage{Mod: scaleNestedModValue(s.Mod, scale)}
}

func (s PactDamage) Clone(pointer bool) Cloneable {
	obj := PactDamage{
		Mod: s.Mod.Clone(),
	}

	if pointer {
		return &obj
	}

	return obj
}

type ExtraSkillMod struct {
	Mod Mod
}

func (s ExtraSkillMod) ScaleNestedMod(scale func(Mod, float64) float64) Cloneable {
	return ExtraSkillMod{Mod: scaleNestedModValue(s.Mod, scale)}
}

func (s ExtraSkillMod) Clone(pointer bool) Cloneable {
	obj := ExtraSkillMod{
		Mod: s.Mod.Clone(),
	}

	if pointer {
		return &obj
	}

	return obj
}

type EnemyModifier struct {
	Mod Mod
}

func (s EnemyModifier) ScaleNestedMod(scale func(Mod, float64) float64) Cloneable {
	return EnemyModifier{Mod: scaleNestedModValue(s.Mod, scale)}
}

func (s EnemyModifier) Clone(pointer bool) Cloneable {
	obj := EnemyModifier{
		Mod: s.Mod.Clone(),
	}

	if pointer {
		return &obj
	}

	return obj
}

type ExtraSkill struct {
	SkillID       string
	SkillName     string
	Level         int
	NoSupports    bool
	Triggered     bool
	Source        interface{}
	TriggerChance float64
}

func (s ExtraSkill) Clone(pointer bool) Cloneable {
	obj := ExtraSkill{
		SkillID:    s.SkillID,
		SkillName:  s.SkillName,
		Level:      s.Level,
		NoSupports: s.NoSupports,
		Triggered:  s.Triggered,
		Source:     s.Source,

		TriggerChance: s.TriggerChance,
	}

	if pointer {
		return &obj
	}

	return obj
}

type GrantReservedPoolAsAura struct {
	Mod Mod
}

func (s GrantReservedPoolAsAura) ScaleNestedMod(scale func(Mod, float64) float64) Cloneable {
	return GrantReservedPoolAsAura{Mod: scaleNestedModValue(s.Mod, scale)}
}

func (s GrantReservedPoolAsAura) Clone(pointer bool) Cloneable {
	obj := GrantReservedPoolAsAura{
		Mod: s.Mod.Clone(),
	}

	if pointer {
		return &obj
	}

	return obj
}

type AffectedByCurseMod struct {
	Mod Mod
}

func (s AffectedByCurseMod) ScaleNestedMod(scale func(Mod, float64) float64) Cloneable {
	return AffectedByCurseMod{Mod: scaleNestedModValue(s.Mod, scale)}
}

func (s AffectedByCurseMod) Clone(pointer bool) Cloneable {
	obj := AffectedByCurseMod{
		Mod: s.Mod.Clone(),
	}

	if pointer {
		return &obj
	}

	return obj
}

type WeaponData struct {
	Key   string
	Value float64
	Unset bool
}

func (s WeaponData) Clone(pointer bool) Cloneable {
	obj := WeaponData{
		Key:   s.Key,
		Value: s.Value,
		Unset: s.Unset,
	}

	if pointer {
		return &obj
	}

	return obj
}

type ArmourData struct {
	Key   string
	Value float64
}

func (s ArmourData) Clone(pointer bool) Cloneable {
	obj := ArmourData{
		Key:   s.Key,
		Value: s.Value,
	}

	if pointer {
		return &obj
	}

	return obj
}

type ExtraSupport struct {
	SkillID string
	Level   int
}

func (s ExtraSupport) Clone(pointer bool) Cloneable {
	obj := ExtraSupport{
		SkillID: s.SkillID,
		Level:   s.Level,
	}

	if pointer {
		return &obj
	}

	return obj
}

type JewelData struct {
	Key   string
	Value any
}

func (s JewelData) Clone(pointer bool) Cloneable {
	obj := JewelData{
		Key:   s.Key,
		Value: s.Value,
	}

	if pointer {
		return &obj
	}

	return obj
}

type ConquerorType struct {
	ID   string
	Type string
}

func (s ConquerorType) Clone(pointer bool) Cloneable {
	obj := ConquerorType{
		ID:   s.ID,
		Type: s.Type,
	}

	if pointer {
		return &obj
	}

	return obj
}

type LegionJewel struct {
	ID        int
	Conqueror ConquerorType
}

func (s LegionJewel) Clone(pointer bool) Cloneable {
	obj := LegionJewel{
		ID:        s.ID,
		Conqueror: s.Conqueror.Clone(false).(ConquerorType),
	}

	if pointer {
		return &obj
	}

	return obj
}

type ImpossibleEscapeKeystones struct {
	Key   string
	Value bool
}

func (s ImpossibleEscapeKeystones) Clone(pointer bool) Cloneable {
	obj := ImpossibleEscapeKeystones{
		Key:   s.Key,
		Value: s.Value,
	}

	if pointer {
		return &obj
	}

	return obj
}

type ExtraCurse struct {
	SkillID       string
	SkillName     string
	Level         int
	ApplyToPlayer bool
}

func (s ExtraCurse) Clone(pointer bool) Cloneable {
	obj := ExtraCurse{
		SkillID:       s.SkillID,
		SkillName:     s.SkillName,
		Level:         s.Level,
		ApplyToPlayer: s.ApplyToPlayer,
	}

	if pointer {
		return &obj
	}

	return obj
}

type LinkedSupport struct {
	TargetSlotName string
}

func (s LinkedSupport) Clone(pointer bool) Cloneable {
	obj := LinkedSupport{
		TargetSlotName: s.TargetSlotName,
	}

	if pointer {
		return &obj
	}

	return obj
}

type GrantedAscendancyNode struct {
	Side string
	Name string
}

func (s GrantedAscendancyNode) Clone(pointer bool) Cloneable {
	obj := GrantedAscendancyNode{
		Side: s.Side,
		Name: s.Name,
	}

	if pointer {
		return &obj
	}

	return obj
}

type ExtraSkillStat struct {
	Key   string
	Value any
}

func (s ExtraSkillStat) Clone(pointer bool) Cloneable {
	obj := ExtraSkillStat{
		Key:   s.Key,
		Value: s.Value,
	}

	if pointer {
		return &obj
	}

	return obj
}

type SupportedGemProperty struct {
	Keyword string
	Key     string
	Value   int
}

func (s SupportedGemProperty) Clone(pointer bool) Cloneable {
	obj := SupportedGemProperty{
		Keyword: s.Keyword,
		Key:     s.Key,
		Value:   s.Value,
	}

	if pointer {
		return &obj
	}

	return obj
}

type SelfDamage struct {
	BaseDamage float64
	DmgMult    float64
	DamageType string
}

func (s SelfDamage) Clone(pointer bool) Cloneable {
	obj := SelfDamage{
		BaseDamage: s.BaseDamage,
		DmgMult:    s.DmgMult,
		DamageType: s.DamageType,
	}

	if pointer {
		return &obj
	}

	return obj
}
