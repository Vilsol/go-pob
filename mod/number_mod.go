package mod

var _ Mod = (*FloatMod)(nil)

type FloatMod struct {
	*BaseMod
	ModValue float64
}

func NewFloat(name string, modType Type, value float64) *FloatMod {
	self := &FloatMod{
		ModValue: value,
		BaseMod: &BaseMod{
			ModName: name,
			ModType: modType,
		},
	}
	self.child = self
	return self
}

func (m *FloatMod) Value() interface{} {
	return m.ModValue
}

func (m *FloatMod) Clone() Mod {
	out := &FloatMod{
		BaseMod:  m.BaseMod.Clone().(*BaseMod),
		ModValue: m.ModValue,
	}
	out.child = out
	return out
}
