package mod

var _ Tag = (*SkillNameTag)(nil)

type SkillNameTag struct {
	TagType       Type
	SkillNameList []string
	Negative      bool
}

func SkillName(names ...string) *SkillNameTag {
	return &SkillNameTag{
		TagType:       TypeSkillName,
		SkillNameList: names,
	}
}

func (t SkillNameTag) Type() Type {
	return t.TagType
}

func (t *SkillNameTag) Neg(negative bool) *SkillNameTag {
	t.Negative = negative
	return t
}
