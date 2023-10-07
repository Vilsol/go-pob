package mod

var _ Tag = (*ModFlagTag)(nil)

//nolint:all
type ModFlagTag struct {
	TagType Type
	Flag    MFlag
}

//nolint:all
func ModFlag(flag MFlag) *ModFlagTag {
	return &ModFlagTag{
		TagType: TypeModFlag,
		Flag:    flag,
	}
}

func (t ModFlagTag) Type() Type {
	return t.TagType
}
