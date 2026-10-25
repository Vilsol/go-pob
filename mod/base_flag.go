package mod

var _ Tag = (*BaseFlagTag)(nil)

//pob:src/Classes/ModStore.lua:881-889
type BaseFlagTag struct {
	TagType  Type
	BaseFlag string
	Negative bool
}

func BaseFlag(baseFlag string) *BaseFlagTag {
	return &BaseFlagTag{
		TagType:  TypeBaseFlag,
		BaseFlag: baseFlag,
	}
}

func (t *BaseFlagTag) Type() Type {
	return t.TagType
}

func (t *BaseFlagTag) Neg(negative bool) *BaseFlagTag {
	t.Negative = negative
	return t
}
