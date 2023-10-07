package mod

var _ Tag = (*DistanceRampTag)(nil)

type DistanceRampTag struct {
	TagType Type
	Ramp    [][]int
}

func DistanceRamp(ramp [][]int) *DistanceRampTag {
	return &DistanceRampTag{
		TagType: TypeDistanceRamp,
		Ramp:    ramp,
	}
}

func (m *DistanceRampTag) Type() Type {
	return TypeDistanceRamp
}
