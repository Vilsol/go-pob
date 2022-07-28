package mod

var _ Tag = (*MultiplierTag)(nil)

type MultiplierTag struct {
	Variable         string
	Base             float64
	Division         float64
	TagLimit         *float64
	TagLimitVariable *string
	LimitTotal       bool
}

func Multiplier(variable string, base float64) *MultiplierTag {
	return &MultiplierTag{
		Variable: variable,
		Base:     base,
		Division: 1,
	}
}

func (*MultiplierTag) Type() Type {
	return TypeMultiplier
}

func (m *MultiplierTag) Div(div float64) *MultiplierTag {
	out := *m
	out.Division = div
	return &out
}

func (m *MultiplierTag) Limit(limit float64) *MultiplierTag {
	out := *m
	out.TagLimit = &limit
	return &out
}
