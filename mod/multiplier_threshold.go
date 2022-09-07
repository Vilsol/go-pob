package mod

var _ Tag = (*MultiplierThresholdTag)(nil)

type MultiplierThresholdTag struct {
	TagType           Type
	Variable          string
	TagThreshold      *float64
	ThresholdVariable *string
	TagUpper          bool
	TagActor          string
}

func MultiplierThreshold(variable string) *MultiplierThresholdTag {
	return &MultiplierThresholdTag{
		TagType:  TypeMultiplierThreshold,
		Variable: variable,
	}
}

func (m *MultiplierThresholdTag) Type() Type {
	return m.TagType
}

func (m *MultiplierThresholdTag) Threshold(threshold float64) *MultiplierThresholdTag {
	m.TagThreshold = &threshold
	return m
}

func (m *MultiplierThresholdTag) ThresholdVar(thresholdVar string) *MultiplierThresholdTag {
	m.ThresholdVariable = &thresholdVar
	return m
}

func (m *MultiplierThresholdTag) Upper(upper bool) *MultiplierThresholdTag {
	m.TagUpper = upper
	return m
}

func (m *MultiplierThresholdTag) Actor(actor string) *MultiplierThresholdTag {
	m.TagActor = actor
	return m
}
