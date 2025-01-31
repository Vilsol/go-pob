package mod

var _ Tag = (*DistanceRampTag)(nil)

type DistanceRampTag struct {
	TagType Type
	Ramp    [][]float64
}

func DistanceRamp(ramp [][]float64) *DistanceRampTag {
	return &DistanceRampTag{
		TagType: TypeDistanceRamp,
		Ramp:    ramp,
	}
}

func (m *DistanceRampTag) Type() Type {
	return TypeDistanceRamp
}
