package mod

var _ Tag = (*MultiplierTag)(nil)

type MultiplierTag struct {
	TagType          Type
	Variable         string
	Base             float64
	Division         float64
	TagLimit         *float64
	TagLimitVariable *string
	LimitTotal       bool
}

func Multiplier(variable string, base float64) *MultiplierTag {
	return &MultiplierTag{
		TagType:  TypeMultiplier,
		Variable: variable,
		Base:     base,
		Division: 1,
	}
}

func (m *MultiplierTag) Type() Type {
	return m.TagType
}

func (m *MultiplierTag) Div(div float64) *MultiplierTag {
	m.Division = div
	return m
}

func (m *MultiplierTag) Limit(limit float64) *MultiplierTag {
	m.TagLimit = &limit
	return m
}
