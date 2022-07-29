package mod

type FlagMod struct {
	*BaseMod
	ModValue bool
}

func NewFlag(name string, value bool) *FlagMod {
	base := &BaseMod{
		ModName: name,
		ModType: TypeFlag,
	}

	self := &FlagMod{
		ModValue: value,
		BaseMod:  base,
	}

	base.parent = self

	return self
}

func (m *FlagMod) Value() interface{} {
	return m.ModValue
}
