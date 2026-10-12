package mod

var _ Mod = (*BaseMod)(nil)

type BaseMod struct {
	parent Mod
	child  Mod

	ModName         string
	ModType         Type
	ModSource       Source
	ModFlags        MFlag
	ModKeywordFlags KeywordFlag
	ModTags         []Tag
	ModValue        *ModValueMulti

	ModReplaced bool
}

func (m *BaseMod) Replaced() bool {
	return m.ModReplaced
}

func (m *BaseMod) SetReplaced(replaced bool) {
	m.ModReplaced = replaced
}

func (m *BaseMod) Name() string {
	return m.ModName
}

func (m *BaseMod) SetName(name string) Mod {
	m.ModName = name
	return m
}

func (m *BaseMod) Type() Type {
	return m.ModType
}

func (m *BaseMod) Source(source Source) Mod {
	m.ModSource = source
	return m.child
}

func (m *BaseMod) Flag(flag MFlag) Mod {
	m.ModFlags |= flag
	return m.child
}

func (m *BaseMod) KeywordFlag(keywordFlag KeywordFlag) Mod {
	m.ModKeywordFlags |= keywordFlag
	return m.child
}

func (m *BaseMod) Tag(tag ...Tag) Mod {
	if tag == nil {
		return m.child
	}

	m.ModTags = append(m.ModTags, tag...)
	return m.child
}

func (m *BaseMod) Flags() MFlag {
	return m.ModFlags
}

func (m *BaseMod) KeywordFlags() KeywordFlag {
	return m.ModKeywordFlags
}

func (m *BaseMod) GetSource() Source {
	return m.ModSource
}

func (m *BaseMod) Tags() []Tag {
	return m.ModTags
}

func (m *BaseMod) Value() *ModValueMulti {
	return m.ModValue
}

func (m *BaseMod) SetValue(value *ModValueMulti) Mod {
	m.ModValue = value
	return m
}

func (m *BaseMod) Clone() Mod {
	var tags []Tag
	if m.ModTags != nil {
		tags = make([]Tag, len(m.ModTags))
		copy(tags, m.ModTags)
	}

	c := &BaseMod{
		parent:          m.parent,
		ModName:         m.ModName,
		ModType:         m.ModType,
		ModSource:       m.ModSource,
		ModFlags:        m.ModFlags,
		ModKeywordFlags: m.ModKeywordFlags,
		ModTags:         tags,
		ModValue:        m.ModValue.Clone(),
		ModReplaced:     m.ModReplaced,
	}
	c.child = c
	return c
}

func CloneSlice(mods []Mod) []Mod {
	if mods == nil {
		return nil
	}

	out := make([]Mod, len(mods))
	for i, m := range mods {
		out[i] = m.Clone()
	}

	return out
}

func (m *BaseMod) ClearTags() {
	m.ModTags = nil
}

func NewFloat(name string, modType Type, value float64) Mod {
	self := &BaseMod{
		ModName:  name,
		ModType:  modType,
		ModValue: NewModValueFloat(value),
	}
	self.child = self
	return self
}

func NewFlag(name string, value bool) Mod {
	self := &BaseMod{
		ModName:  name,
		ModType:  TypeFlag,
		ModValue: NewModValueFlag(value),
	}
	self.child = self
	return self
}

func NewList(name string, value any) Mod {
	self := &BaseMod{
		ModName:  name,
		ModType:  TypeList,
		ModValue: NewModValueList(value),
	}
	self.child = self
	return self
}
