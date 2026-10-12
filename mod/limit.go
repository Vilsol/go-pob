package mod

var _ Tag = (*LimitTag)(nil)

//pob:src/Classes/ModStore.lua:588-589
type LimitTag struct {
	TagType  Type
	TagLimit float64
	LimitVar string
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
