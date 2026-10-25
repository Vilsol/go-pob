package mod

var _ Tag = (*LimitTag)(nil)

//pob:src/Classes/ModStore.lua:631-632
//pob2:src/Classes/ModStore.lua:739-741
type LimitTag struct {
	TagType  Type
	TagLimit float64
	LimitVar string
	Negative bool
}

func Limit(limit float64) *LimitTag {
	return &LimitTag{
		TagType:  TypeLimit,
		TagLimit: limit,
	}
}

func LimitVar(limitVar string) *LimitTag {
	return &LimitTag{
		TagType:  TypeLimit,
		LimitVar: limitVar,
	}
}

func (t *LimitTag) Type() Type {
	return t.TagType
}

func (t *LimitTag) Neg(negative bool) *LimitTag {
	t.Negative = negative
	return t
}
