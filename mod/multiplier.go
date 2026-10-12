package mod

var _ Tag = (*MultiplierTag)(nil)

type MultiplierTag struct {
	TagType           Type
	VariableList      []string
	TagBase           float64
	Division          float64
	DivisionSet       bool
	TagLimit          *float64
	TagLimitVariable  *string
	TagLimitTotal     bool
	TagActor          string
	TagLimitActor     string
	TagGlobalLimit    *float64
	TagGlobalLimitKey *string

	TagDivVariable   *string
	TagNoFloor       bool
	TagLimitStat     *string
	TagLimitNegTotal bool
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
	m.DivisionSet = true
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

func (m *MultiplierTag) LimitVar(limitVar string) *MultiplierTag {
	m.TagLimitVariable = &limitVar
	return m
}

func (m *MultiplierTag) Actor(actor string) *MultiplierTag {
	m.TagActor = actor
	return m
}

func (m *MultiplierTag) LimitActor(limitActor string) *MultiplierTag {
	m.TagLimitActor = limitActor
	return m
}

func (m *MultiplierTag) GlobalLimit(globalLimit float64) *MultiplierTag {
	m.TagGlobalLimit = &globalLimit
	return m
}

func (m *MultiplierTag) GlobalLimitKey(globalLimitKey string) *MultiplierTag {
	m.TagGlobalLimitKey = &globalLimitKey
	return m
}

func (m *MultiplierTag) Clone() Tag {
	return &MultiplierTag{
		TagType:           m.TagType,
		VariableList:      m.VariableList,
		TagBase:           m.TagBase,
		Division:          m.Division,
		DivisionSet:       m.DivisionSet,
		TagLimit:          m.TagLimit,
		TagLimitVariable:  m.TagLimitVariable,
		TagLimitTotal:     m.TagLimitTotal,
		TagActor:          m.TagActor,
		TagLimitActor:     m.TagLimitActor,
		TagGlobalLimit:    m.TagGlobalLimit,
		TagGlobalLimitKey: m.TagGlobalLimitKey,
		TagDivVariable:    m.TagDivVariable,
		TagNoFloor:        m.TagNoFloor,
		TagLimitStat:      m.TagLimitStat,
		TagLimitNegTotal:  m.TagLimitNegTotal,
	}
}

func (m *MultiplierTag) DivVar(name string) *MultiplierTag {
	m.TagDivVariable = &name
	return m
}

func (m *MultiplierTag) NoFloor() *MultiplierTag {
	m.TagNoFloor = true
	return m
}

func (m *MultiplierTag) LimitStat(name string) *MultiplierTag {
	m.TagLimitStat = &name
	return m
}

func (m *MultiplierTag) LimitNegTotal() *MultiplierTag {
	m.TagLimitNegTotal = true
	return m
}
