package mod

var _ Tag = (*SkillTypeTag)(nil)

type SkillTypeTag struct {
	TagType   Type
	SkillType string // TODO Change to data.SkillType
	Negative  bool
}

func SkillType(skillType string) *SkillTypeTag {
	return &SkillTypeTag{
		TagType:   TypeSkillType,
		SkillType: skillType,
	}
}

func (t SkillTypeTag) Type() Type {
	return t.TagType
}

func (t *SkillTypeTag) Neg(negative bool) *SkillTypeTag {
	t.Negative = negative
	return t
}
