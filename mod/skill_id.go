package mod

var _ Tag = (*SkillIDTag)(nil)

type SkillIDTag struct {
	TagType Type
	ID      string
}

//nolint:all
func SkillId(id string) *SkillIDTag {
	return &SkillIDTag{
		TagType: TypeSkillID,
		ID:      id,
	}
}

func (t SkillIDTag) Type() Type {
	return t.TagType
}
