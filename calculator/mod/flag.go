package mod

var _ Tag = (*FlagTag)(nil)

type FlagTag struct {
	Value bool
}

func Flag(value bool) *FlagTag {
	return &FlagTag{Value: value}
}

func (FlagTag) Type() Type {
	return TypeFlag
}
