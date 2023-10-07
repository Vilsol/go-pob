package mod

import "github.com/Vilsol/go-pob-data/poe"

var _ Tag = (*SkillIDTag)(nil)

type SkillIDTag struct {
	TagType Type
	IDTag   string
	Name    string
}

//nolint:all
func SkillId(id string) *SkillIDTag {
	return &SkillIDTag{
		TagType: TypeSkillID,
		IDTag:   id,
	}
}

func SkillIdByName(name string) *SkillIDTag {
	return &SkillIDTag{
		TagType: TypeSkillID,
		Name:    name,
	}
}

func (t SkillIDTag) Type() Type {
	return t.TagType
}

func (t SkillIDTag) ID() string {
	if t.IDTag == "" {
		return poe.ActiveSkillsByDisplayName[t.Name].SkillID
	}
	return t.IDTag
}
