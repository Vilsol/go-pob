package mod

var _ Tag = (*PerStatTag)(nil)

type PerStatTag struct {
	TagType          Type
	Stat             string
	Divide           float64
	TagLimit         *float64
	TagLimitVariable *string
	LimitTotal       bool
	Base             float64
}

func PerStat(stat string, divide float64) *PerStatTag {
	return &PerStatTag{
		TagType: TypePerStat,
		Stat:    stat,
		Divide:  divide,
	}
}

func (t PerStatTag) Type() Type {
	return t.TagType
}
