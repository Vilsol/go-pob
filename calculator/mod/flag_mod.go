package mod

type FlagMod struct {
	BaseMod
	ModValue bool
}

func NewFlag(name string, value bool) *FlagMod {
	return &FlagMod{
		ModValue: value,
		BaseMod: BaseMod{
			ModName: name,
			ModType: TypeFlag,
		},
	}
}

func (m *FlagMod) Value() interface{} {
	return m.ModValue
}
