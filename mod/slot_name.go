package mod

var _ Tag = (*SlotNameTag)(nil)

type SlotNameTag struct {
	TagType      Type
	SlotNameList []string
}

func SlotName(names ...string) *SlotNameTag {
	return &SlotNameTag{
		TagType:      TypeSlotName,
		SlotNameList: names,
	}
}

func (t SlotNameTag) Type() Type {
	return t.TagType
}
