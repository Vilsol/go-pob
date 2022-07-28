package mod

var _ Tag = (*PerStatTag)(nil)

type PerStatTag struct {
	Stat             string
	Divide           float64
	TagLimit         *float64
	TagLimitVariable *string
	LimitTotal       bool
	Base             float64
}

func PerStat(stat string, divide float64) *PerStatTag {
	return &PerStatTag{
		Stat:   stat,
		Divide: divide,
	}
}

func (PerStatTag) Type() Type {
	return TypePerStat
}
