package mod

var _ Tag = (*DisablesItemTag)(nil)

type DisablesItemTag struct {
	TagType            Type
	TagSlotName        string
	TagExcludeItemType string
}

func DisablesItem(slotName string) *DisablesItemTag {
	return &DisablesItemTag{
		TagType:     TypeDisablesItem,
		TagSlotName: slotName,
	}
}

func (t *DisablesItemTag) ExcludeItemType(itemType string) *DisablesItemTag {
	t.TagExcludeItemType = itemType
	return t
}

func (t *DisablesItemTag) Type() Type {
	return t.TagType
}
