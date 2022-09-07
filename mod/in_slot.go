package mod

var _ Tag = (*InSlotTag)(nil)

type InSlotTag struct {
	TagType Type
	N       int
}

func InSlot(n int) *InSlotTag {
	return &InSlotTag{
		TagType: TypeInSlot,
		N:       n,
	}
}

func (t InSlotTag) Type() Type {
	return t.TagType
}
