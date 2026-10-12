package mod

var _ Tag = (*MonsterTagTag)(nil)

type MonsterTagTag struct {
	TagType        Type
	MonsterTagList []string
	Negative       bool
}

func MonsterTag(tags ...string) *MonsterTagTag {
	return &MonsterTagTag{
		TagType:        TypeMonsterTag,
		MonsterTagList: tags,
	}
}

func (t *MonsterTagTag) Type() Type {
	return t.TagType
}

func (t *MonsterTagTag) Neg(negative bool) *MonsterTagTag {
	t.Negative = negative
	return t
}
