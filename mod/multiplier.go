package mod

var _ Tag = (*MultiplierTag)(nil)

type MultiplierTag struct {
	TagType          Type
	VariableList     []string
	TagBase          float64
	Division         float64
	TagLimit         *float64
	TagLimitVariable *string
	TagLimitTotal    bool
	TagActor         string
}

func Multiplier(vars ...string) *MultiplierTag {
	return &MultiplierTag{
		TagType:      TypeMultiplier,
		VariableList: vars,
		Division:     1,
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

func (m *MultiplierTag) LimitTotal(limitTotal bool) *MultiplierTag {
	m.TagLimitTotal = limitTotal
	return m
}

func (m *MultiplierTag) Base(base float64) *MultiplierTag {
	m.TagBase = base
	return m
}

func (m *MultiplierTag) Actor(actor string) *MultiplierTag {
	m.TagActor = actor
	return m
}
