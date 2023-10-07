package mod

var _ Tag = (*GlobalEffectTag)(nil)

type GlobalEffectTag struct {
	TagType          Type
	GlobalEffectList []string
	Negative         bool
	UnscalableTag    bool
	NameTag          string
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

func (t *GlobalEffectTag) Unscalable(unscalable bool) *GlobalEffectTag {
	t.UnscalableTag = unscalable
	return t
}

func (t *GlobalEffectTag) Name(name string) *GlobalEffectTag {
	t.NameTag = name
	return t
}
