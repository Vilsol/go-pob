package mod

var _ Tag = (*FlagTag)(nil)

type FlagTag struct {
	TagType Type
	Value   bool
}

func Flag(value bool) *FlagTag {
	return &FlagTag{
		TagType: TypeFlag,
		Value:   value,
	}
}

func (m FlagTag) Type() Type {
	return m.TagType
}
