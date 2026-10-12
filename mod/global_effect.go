package mod

var _ Tag = (*GlobalEffectTag)(nil)

type GlobalEffectTag struct {
	TagType          Type
	GlobalEffectList []string
	Negative         bool
	UnscalableTag    bool
	NameTag          string

	EffectCondTag      string
	EffectEnemyCondTag string
	ModCondTag         string

	EffectStackVarTag      string
	EffectStackLimitTag    float64
	EffectStackLimitVarTag string

	//	warcryBuff[1].warcryPowerBonus = m_floor((warcryBuff[1].limit and m_min(warcryPower, warcryBuff[1].limit) or warcryPower) / warcryBuff[1].div)
	DivTag   float64
	DivSet   bool
	LimitTag float64
	LimitSet bool

	ApplyNotPlayerTag bool
	ApplyMinionsTag   bool

	AllowTotemBuffTag bool
}

func GlobalEffect(names ...string) *GlobalEffectTag {
	return &GlobalEffectTag{
		TagType:          TypeGlobalEffect,
		GlobalEffectList: names,
	}
}

func (t *GlobalEffectTag) Type() Type {
	return t.TagType
}

func (t *GlobalEffectTag) Unscalable(unscalable bool) *GlobalEffectTag {
	t.UnscalableTag = unscalable
	return t
}

func (t *GlobalEffectTag) Name(name string) *GlobalEffectTag {
	t.NameTag = name
	return t
}

func (t *GlobalEffectTag) EffectCond(cond string) *GlobalEffectTag {
	t.EffectCondTag = cond
	return t
}

func (t *GlobalEffectTag) EffectEnemyCond(cond string) *GlobalEffectTag {
	t.EffectEnemyCondTag = cond
	return t
}

func (t *GlobalEffectTag) ModCond(cond string) *GlobalEffectTag {
	t.ModCondTag = cond
	return t
}

func (t *GlobalEffectTag) EffectStackVar(v string) *GlobalEffectTag {
	t.EffectStackVarTag = v
	return t
}

func (t *GlobalEffectTag) EffectStackLimit(limit float64) *GlobalEffectTag {
	t.EffectStackLimitTag = limit
	return t
}

func (t *GlobalEffectTag) EffectStackLimitVar(v string) *GlobalEffectTag {
	t.EffectStackLimitVarTag = v
	return t
}

func (t *GlobalEffectTag) ApplyNotPlayer(apply bool) *GlobalEffectTag {
	t.ApplyNotPlayerTag = apply
	return t
}

func (t *GlobalEffectTag) ApplyMinions(apply bool) *GlobalEffectTag {
	t.ApplyMinionsTag = apply
	return t
}

func (t *GlobalEffectTag) AllowTotemBuff(allow bool) *GlobalEffectTag {
	t.AllowTotemBuffTag = allow
	return t
}

func (t *GlobalEffectTag) Div(div float64) *GlobalEffectTag {
	t.DivTag, t.DivSet = div, true
	return t
}

func (t *GlobalEffectTag) Limit(limit float64) *GlobalEffectTag {
	t.LimitTag, t.LimitSet = limit, true
	return t
}
