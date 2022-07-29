package mod

var _ Tag = (*IgnoreCondTag)(nil)

type IgnoreCondTag struct {
	TagType Type
}

func IgnoreCond() *IgnoreCondTag {
	return &IgnoreCondTag{
		TagType: TypeIgnoreCond,
	}
}

func (m IgnoreCondTag) Type() Type {
	return m.TagType
}
