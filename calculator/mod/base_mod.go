package mod

import "go-pob/data"

var _ Mod = (*BaseMod)(nil)

type BaseMod struct {
	parent Mod

	ModName         string
	ModType         Type
	ModSource       Source
	ModFlags        data.ModFlag
	ModKeywordFlags KeywordFlag
	ModTags         []Tag
}

func (m *BaseMod) Flags() data.ModFlag {
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
	return m.parent
}

func (m *BaseMod) Flag(flag data.ModFlag) Mod {
	m.ModFlags |= flag
	return m.parent
}

func (m *BaseMod) KeywordFlag(keywordFlag KeywordFlag) Mod {
	m.ModKeywordFlags |= keywordFlag
	return m.parent
}

func (m *BaseMod) Tag(tag Tag) Mod {
	m.ModTags = append(m.ModTags, tag)
	return m.parent
}
