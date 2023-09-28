package mod

var _ Tag = (*GlobalEffectTag)(nil)

type GlobalEffectTag struct {
	TagType          Type
	GlobalEffectList []string
	Negative         bool
}

func GlobalEffect(names ...string) *GlobalEffectTag {
	return &GlobalEffectTag{
		TagType:          TypeGlobalEffect,
		GlobalEffectList: names,
	}
}

func (t GlobalEffectTag) Type() Type {
	return t.TagType
}
