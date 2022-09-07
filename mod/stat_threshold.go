package mod

var _ Tag = (*StatThresholdTag)(nil)

type StatThresholdTag struct {
	TagType          Type
	Stat             string
	Threshold        float64
	TagUpper         bool
	TagThresholdStat string
}

func StatThreshold(stat string, threshold float64) *StatThresholdTag {
	return &StatThresholdTag{
		TagType:   TypeStatThreshold,
		Stat:      stat,
		Threshold: threshold,
	}
}

func (t StatThresholdTag) Type() Type {
	return t.TagType
}

func (t *StatThresholdTag) Upper(upper bool) *StatThresholdTag {
	t.TagUpper = upper
	return t
}

func (t *StatThresholdTag) ThresholdStat(thresholdStat string) *StatThresholdTag {
	t.TagThresholdStat = thresholdStat
	return t
}
