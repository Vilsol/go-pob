package mod

var _ Tag = (*SocketedInTag)(nil)

type SocketedInTag struct {
	TagType        Type
	SlotName       string
	TagKeyword     string
	TagSocketColor string
	TagSockets     []int
}

func SocketedIn(slotName string) *SocketedInTag {
	return &SocketedInTag{
		TagType:  TypeSocketedIn,
		SlotName: slotName,
	}
}

func (t *SocketedInTag) Type() Type {
	return t.TagType
}

func (t *SocketedInTag) Keyword(keyword string) *SocketedInTag {
	t.TagKeyword = keyword
	return t
}

func (t *SocketedInTag) SocketColor(color string) *SocketedInTag {
	t.TagSocketColor = color
	return t
}

func (t *SocketedInTag) Sockets(sockets ...int) *SocketedInTag {
	t.TagSockets = sockets
	return t
}
