package mod

var _ Tag = (*ActorConditionTag)(nil)

type ActorConditionTag struct {
	Actor    *string
	Variable string
	Negative bool
}

func ActorCondition(actor string, variable string) *ActorConditionTag {
	return &ActorConditionTag{
		Actor:    &actor,
		Variable: variable,
	}
}

func (*ActorConditionTag) Type() Type {
	return TypeActorCondition
}

func (m *ActorConditionTag) Neg(negative bool) *ActorConditionTag {
	out := *m
	out.Negative = negative
	return &out
}
