package mod

var _ Tag = (*SkillIDTag)(nil)

type SkillIDTag struct {
	TagType Type
	IDTag   string
}

//nolint:all
func SkillId(id string) *SkillIDTag {
	return &SkillIDTag{
		TagType: TypeSkillID,
		IDTag:   id,
	}
}

func (t SkillIDTag) Type() Type {
	return t.TagType
}

func (t SkillIDTag) ID() string {
	return t.IDTag
}
