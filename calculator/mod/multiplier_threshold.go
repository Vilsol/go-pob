package mod

var _ Tag = (*MultiplierThresholdTag)(nil)

type MultiplierThresholdTag struct {
	Variable          string
	TagThreshold      *float64
	ThresholdVariable *string
	TagUpper          bool
}

func MultiplierThreshold(variable string) *MultiplierThresholdTag {
	return &MultiplierThresholdTag{
		Variable: variable,
	}
}

func (*MultiplierThresholdTag) Type() Type {
	return TypeMultiplierThreshold
}

func (m *MultiplierThresholdTag) Threshold(threshold float64) *MultiplierThresholdTag {
	out := *m
	out.TagThreshold = &threshold
	return &out
}

func (m *MultiplierThresholdTag) ThresholdVar(thresholdVar string) *MultiplierThresholdTag {
	out := *m
	out.ThresholdVariable = &thresholdVar
	return &out
}

func (m *MultiplierThresholdTag) Upper(upper bool) *MultiplierThresholdTag {
	out := *m
	out.TagUpper = upper
	return &out
}
