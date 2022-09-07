package mod

var _ Tag = (*SlotNumberTag)(nil)

type SlotNumberTag struct {
	TagType Type
	N       int
}

func SlotNumber(n int) *SlotNumberTag {
	return &SlotNumberTag{
		TagType: TypeSlotNumber,
		N:       n,
	}
}

func (t SlotNumberTag) Type() Type {
	return t.TagType
}
