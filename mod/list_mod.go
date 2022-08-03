package mod

type ListMod struct {
	*BaseMod
	ModValue interface{}
}

func NewList(name string, value interface{}) *ListMod {
	self := &ListMod{
		ModValue: value,
		BaseMod: &BaseMod{
			ModName: name,
			ModType: TypeList,
		},
	}
	self.child = self
	return self
}

func (m *ListMod) Value() interface{} {
	return m.ModValue
}

func (m *ListMod) Clone() Mod {
	out := &ListMod{
		BaseMod:  m.BaseMod.Clone().(*BaseMod),
		ModValue: m.ModValue, // TODO Copy Value
	}
	out.child = out
	return out
}
