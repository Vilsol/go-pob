package mod

var _ Tag = (*ConditionTag)(nil)

type ConditionTag struct {
	TagType  Type
	Variable string
	Negative bool
}

func Condition(variable string) *ConditionTag {
	return &ConditionTag{
		TagType:  TypeCondition,
		Variable: variable,
	}
}

func (m *ConditionTag) Type() Type {
	return TypeCondition
}

func (m *ConditionTag) Neg(negative bool) *ConditionTag {
	m.Negative = negative
	return m
}
