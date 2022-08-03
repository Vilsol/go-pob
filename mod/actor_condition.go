package mod

var _ Tag = (*ActorConditionTag)(nil)

type ActorConditionTag struct {
	TagType  Type
	Actor    *string
	Variable string
	Negative bool
}

func ActorCondition(actor string, variable string) *ActorConditionTag {
	return &ActorConditionTag{
		TagType:  TypeActorCondition,
		Actor:    &actor,
		Variable: variable,
	}
}

func (m *ActorConditionTag) Type() Type {
	return m.TagType
}

func (m *ActorConditionTag) Neg(negative bool) *ActorConditionTag {
	m.Negative = negative
	return m
}
