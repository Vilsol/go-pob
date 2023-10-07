package mod

var _ Tag = (*MeleeProximityTag)(nil)

type MeleeProximityTag struct {
	TagType Type
	Ramp    []int
}

func MeleeProximity(ramp []int) *MeleeProximityTag {
	return &MeleeProximityTag{
		TagType: TypeMeleeProximity,
		Ramp:    ramp,
	}
}

func (m *MeleeProximityTag) Type() Type {
	return TypeMeleeProximity
}
