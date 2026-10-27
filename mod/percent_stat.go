package mod

var _ Tag = (*PercentStatTag)(nil)

type PercentStatTag struct {
	TagType    Type
	Stat       string
	Percent    float64
	PercentVar string
	TagActor   string
	TagFloor   bool
	StatList   []string
}

func (t *PercentStatTag) Actor(actor string) *PercentStatTag {
	t.TagActor = actor
	return t
}

func (t *PercentStatTag) Floor() *PercentStatTag {
	t.TagFloor = true
	return t
}

func PercentStatVar(stat string, percentVar string) *PercentStatTag {
	return &PercentStatTag{
		TagType:    TypePercentStat,
		Stat:       stat,
		PercentVar: percentVar,
	}
}

func PercentStatList(percent float64, stats ...string) *PercentStatTag {
	return &PercentStatTag{
		TagType:  TypePercentStat,
		StatList: stats,
		Percent:  percent,
	}
}

func PercentStat(stat string, percent float64) *PercentStatTag {
	return &PercentStatTag{
		TagType: TypePercentStat,
		Stat:    stat,
		Percent: percent,
	}
}

func (t *PercentStatTag) Type() Type {
	return t.TagType
}
