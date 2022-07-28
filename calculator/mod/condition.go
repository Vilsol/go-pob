package mod

var _ Tag = (*ConditionTag)(nil)

type ConditionTag struct {
	Variable string
	Negative bool
}

func Condition(variable string) *ConditionTag {
	return &ConditionTag{Variable: variable}
}

func (*ConditionTag) Type() Type {
	return TypeCondition
}

func (m *ConditionTag) Neg(negative bool) *ConditionTag {
	out := *m
	out.Negative = negative
	return &out
}
