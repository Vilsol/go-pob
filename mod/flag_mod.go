package mod

type FlagMod struct {
	*BaseMod
	ModValue bool
}

func NewFlag(name string, value bool) *FlagMod {
	self := &FlagMod{
		ModValue: value,
		BaseMod: &BaseMod{
			ModName: name,
			ModType: TypeFlag,
		},
	}
	self.child = self
	return self
}

func (m *FlagMod) Value() interface{} {
	return m.ModValue
}

func (m *FlagMod) Clone() Mod {
	out := &FlagMod{
		BaseMod:  m.BaseMod.Clone().(*BaseMod),
		ModValue: m.ModValue,
	}
	out.child = out
	return out
}
