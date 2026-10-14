package mod

var _ Tag = (*KeywordFlagAndTag)(nil)

type KeywordFlagAndTag struct {
	TagType      Type
	KeywordFlags KeywordFlag
}

func KeywordFlagAnd(flags KeywordFlag) *KeywordFlagAndTag {
	return &KeywordFlagAndTag{
		TagType:      TypeKeywordFlagAnd,
		KeywordFlags: flags,
	}
}

func (t KeywordFlagAndTag) Type() Type {
	return t.TagType
}
