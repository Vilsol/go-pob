package mod

var _ Tag = (*StatThresholdTag)(nil)

type StatThresholdTag struct {
	TagType          Type
	Stat             string
	Threshold        float64
	TagUpper         bool
	TagThresholdStat string
	ThresholdPercent float64
	StatList         []string
}

func StatThreshold(stat string, threshold float64) *StatThresholdTag {
	return &StatThresholdTag{
		TagType:   TypeStatThreshold,
		Stat:      stat,
		Threshold: threshold,
	}
}

func (t *StatThresholdTag) Stats(stats ...string) *StatThresholdTag {
	t.StatList = stats
	return t
}

func StatThresholdStat(stat string, thresholdStat string) *StatThresholdTag {
	return &StatThresholdTag{
		TagType:          TypeStatThreshold,
		Stat:             stat,
		TagThresholdStat: thresholdStat,
	}
}

func (t *StatThresholdTag) Type() Type {
	return t.TagType
}

func (t *StatThresholdTag) Upper(upper bool) *StatThresholdTag {
	t.TagUpper = upper
	return t
}

func (t *StatThresholdTag) Percent(thresholdPercent float64) *StatThresholdTag {
	t.ThresholdPercent = thresholdPercent
	return t
}

func (t *StatThresholdTag) ThresholdStat(thresholdStat string) *StatThresholdTag {
	t.TagThresholdStat = thresholdStat
	return t
}
