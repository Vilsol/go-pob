package mod

var _ Tag = (*GemTagTag)(nil)

//pob2:src/Classes/ModStore.lua:970-988
type GemTagTag struct {
	TagType    Type
	GemTagList []string
	Negative   bool
}

func GemTag(tags ...string) *GemTagTag {
	return &GemTagTag{
		TagType:    TypeGemTag,
		GemTagList: tags,
	}
}

func (t *GemTagTag) Type() Type {
	return t.TagType
}

func (t *GemTagTag) Neg(negative bool) *GemTagTag {
	t.Negative = negative
	return t
}
