package mod

var _ Tag = (*GlobalTag)(nil)

type GlobalTag struct {
	TagType  Type
	Negative bool
}

func Global() *GlobalTag {
	return &GlobalTag{
		TagType: TypeGlobal,
	}
}

func (t GlobalTag) Type() Type {
	return t.TagType
}

func (t *GlobalTag) Neg(negative bool) *GlobalTag {
	t.Negative = negative
	return t
}
