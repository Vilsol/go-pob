package mod

var _ Tag = (*PerStatTag)(nil)

type PerStatTag struct {
	TagType           Type
	StatList          []string
	Divide            float64
	TagLimit          *float64
	TagLimitVariable  *string
	TagLimitTotal     bool
	Base              float64
	TagActor          string
	TagGlobalLimit    float64
	TagGlobalLimitKey string
}

func PerStat(divide float64, stats ...string) *PerStatTag {
	return &PerStatTag{
		TagType:  TypePerStat,
		StatList: stats,
		Divide:   divide,
	}
}

func (t PerStatTag) Type() Type {
	return t.TagType
}

func (t *PerStatTag) Actor(actor string) *PerStatTag {
	t.TagActor = actor
	return t
}

func (t *PerStatTag) Limit(limit float64) *PerStatTag {
	t.TagLimit = &limit
	return t
}

func (t *PerStatTag) LimitTotal(limitTotal bool) *PerStatTag {
	t.TagLimitTotal = limitTotal
	return t
}

func (t *PerStatTag) GlobalLimit(globalLimit float64) *PerStatTag {
	t.TagGlobalLimit = globalLimit
	return t
}

func (t *PerStatTag) GlobalLimitKey(globalLimitKey string) *PerStatTag {
	t.TagGlobalLimitKey = globalLimitKey
	return t
}
