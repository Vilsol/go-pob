package mod

var _ Tag = (*SocketedInTag)(nil)

type SocketedInTag struct {
	TagType    Type
	SlotName   string
	TagKeyword string
}

func SocketedIn(slotName string) *SocketedInTag {
	return &SocketedInTag{
		TagType:  TypeSocketedIn,
		SlotName: slotName,
	}
}

func (t SocketedInTag) Type() Type {
	return t.TagType
}

func (t *SocketedInTag) Keyword(keyword string) *SocketedInTag {
	t.TagKeyword = keyword
	return t
}
