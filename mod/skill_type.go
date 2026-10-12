package mod

var _ Tag = (*SkillTypeTag)(nil)

type SkillTypeTag struct {
	TagType       Type
	SkillTypeList []string
	Negative      bool
}

func SkillType(skillTypes ...string) *SkillTypeTag {
	return &SkillTypeTag{
		TagType:       TypeSkillType,
		SkillTypeList: skillTypes,
	}
}

func (t *SkillTypeTag) Type() Type {
	return t.TagType
}

func (t *SkillTypeTag) Neg(negative bool) *SkillTypeTag {
	t.Negative = negative
	return t
}
