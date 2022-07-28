package mod

var _ Mod = (*FloatMod)(nil)

type FloatMod struct {
	BaseMod
	ModValue float64
}

func NewFloat(name string, modType Type, value float64) *FloatMod {
	return &FloatMod{
		ModValue: value,
		BaseMod: BaseMod{
			ModName: name,
			ModType: modType,
		},
	}
}

func (m *FloatMod) Value() interface{} {
	return m.ModValue
}
