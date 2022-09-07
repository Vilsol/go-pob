package mod

var _ Tag = (*ConditionTag)(nil)

type ConditionTag struct {
	TagType  Type
	Negative bool
	VarList  []string
}

func Condition(varList ...string) *ConditionTag {
	return &ConditionTag{
		TagType: TypeCondition,
		VarList: varList,
	}
}

func (m *ConditionTag) Type() Type {
	return TypeCondition
}

func (m *ConditionTag) Neg(negative bool) *ConditionTag {
	m.Negative = negative
	return m
}
