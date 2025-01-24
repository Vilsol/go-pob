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
	c := &BaseMod{
		parent:          m.parent,
		ModName:         m.ModName,
		ModType:         m.ModType,
		ModSource:       m.ModSource,
		ModFlags:        m.ModFlags,
		ModKeywordFlags: m.ModKeywordFlags,
		ModTags:         m.ModTags,
		ModValue:        m.ModValue.Clone(),
	}
	c.child = c
	return c
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
