package mod

var _ Tag = (*SocketedInTag)(nil)

type SocketedInTag struct {
	TagType         Type
	SlotName        string
	TagKeyword      string
	TagSocketColor  string
	TagSockets      []int
	TagSocketsAll   bool
	TagSocketsCount float64
	TagNeg          bool
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

func (t *SocketedInTag) SocketsAll() *SocketedInTag {
	t.TagSocketsAll = true
	return t
}

func (t *SocketedInTag) SocketsCount(count float64) *SocketedInTag {
	t.TagSocketsCount = count
	return t
}

func (t *SocketedInTag) Neg(neg bool) *SocketedInTag {
	t.TagNeg = neg
	return t
}
