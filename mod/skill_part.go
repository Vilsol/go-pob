package mod

var _ Tag = (*SkillPartTag)(nil)

type SkillPartTag struct {
	TagType  Type
	Part     int
	PartList []int
	Negative bool
}

//nolint:all
func SkillPart(part int) *SkillPartTag {
	return &SkillPartTag{
		TagType: TypeSkillPart,
		Part:    part,
	}
}

//nolint:all
func SkillPartList(parts ...int) *SkillPartTag {
	return &SkillPartTag{
		TagType:  TypeSkillPart,
		PartList: parts,
	}
}

func (t SkillPartTag) Type() Type {
	return t.TagType
}
