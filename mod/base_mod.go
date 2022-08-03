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
}

func (m *BaseMod) Clone() Mod {
	return &BaseMod{
		parent:          m.parent,
		ModName:         m.ModName,
		ModType:         m.ModType,
		ModSource:       m.ModSource,
		ModFlags:        m.ModFlags,
		ModKeywordFlags: m.ModKeywordFlags,
		ModTags:         m.ModTags,
	}
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

func (m *BaseMod) Value() interface{} {
	panic("should be implemented by extendee")
}

func (m *BaseMod) Name() string {
	return m.ModName
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

func (m *BaseMod) Tag(tag Tag) Mod {
	m.ModTags = append(m.ModTags, tag)
	return m.child
}
