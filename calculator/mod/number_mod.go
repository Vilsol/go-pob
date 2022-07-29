package mod

var _ Mod = (*FloatMod)(nil)

type FloatMod struct {
	*BaseMod
	ModValue float64
}

func NewFloat(name string, modType Type, value float64) *FloatMod {
	base := &BaseMod{
		ModName: name,
		ModType: modType,
	}

	self := &FloatMod{
		ModValue: value,
		BaseMod:  base,
	}

	base.parent = self

	return self
}

func (m *FloatMod) Value() interface{} {
	return m.ModValue
}
