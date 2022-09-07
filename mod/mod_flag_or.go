package mod

var _ Tag = (*ModFlagOrTag)(nil)

//nolint:all
type ModFlagOrTag struct {
	TagType Type
	Flag    MFlag
}

//nolint:all
func ModFlagOr(flag MFlag) *ModFlagOrTag {
	return &ModFlagOrTag{
		TagType: TypeModFlagOr,
		Flag:    flag,
	}
}

func (t ModFlagOrTag) Type() Type {
	return t.TagType
}
