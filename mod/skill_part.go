package mod

var _ Tag = (*SkillPartTag)(nil)

type SkillPartTag struct {
	TagType Type
	Part    int
}

//nolint:all
func SkillPart(part int) *SkillPartTag {
	return &SkillPartTag{
		TagType: TypeSkillPart,
		Part:    part,
	}
}

func (t SkillPartTag) Type() Type {
	return t.TagType
}
