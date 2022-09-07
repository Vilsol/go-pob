package mod

var _ Tag = (*ActorConditionTag)(nil)

type ActorConditionTag struct {
	TagType      Type
	Actor        *string
	VariableList []string
	Negative     bool
}

func ActorCondition(actor string, variables ...string) *ActorConditionTag {
	return &ActorConditionTag{
		TagType:      TypeActorCondition,
		Actor:        &actor,
		VariableList: variables,
	}
}

func (m *ActorConditionTag) Type() Type {
	return m.TagType
}

func (m *ActorConditionTag) Neg(negative bool) *ActorConditionTag {
	m.Negative = negative
	return m
}
