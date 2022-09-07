package mod

var _ Tag = (*PerStatTag)(nil)

type PerStatTag struct {
	TagType          Type
	StatList         []string
	Divide           float64
	TagLimit         *float64
	TagLimitVariable *string
	LimitTotal       bool
	Base             float64
	TagActor         string
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
