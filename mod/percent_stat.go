package mod

var _ Tag = (*PercentStatTag)(nil)

type PercentStatTag struct {
	TagType Type
	Stat    string
	Percent float64
}

func PercentStat(stat string, percent float64) *PercentStatTag {
	return &PercentStatTag{
		TagType: TypePercentStat,
		Stat:    stat,
		Percent: percent,
	}
}

func (t PercentStatTag) Type() Type {
	return t.TagType
}
