package mod

var _ Tag = (*IgnoreCondTag)(nil)

type IgnoreCondTag struct {
}

func IgnoreCond() *IgnoreCondTag {
	return &IgnoreCondTag{}
}

func (IgnoreCondTag) Type() Type {
	return TypeIgnoreCond
}
