package mod

var _ Tag = (*ItemConditionTag)(nil)

type ItemConditionTag struct {
	TagType Type

	ItemSlot      string
	SearchCond    string
	RarityCond    string
	CorruptedCond *bool
	NameCond      string
	ShaperCond    bool
	ElderCond     bool

	AllSlots    bool
	BothSlots   bool
	ExcludeSelf bool

	Neg bool
}

func ItemCondition(itemSlot string) *ItemConditionTag {
	return &ItemConditionTag{TagType: TypeItemCondition, ItemSlot: itemSlot}
}

func (t *ItemConditionTag) Search(cond string) *ItemConditionTag {
	t.SearchCond = cond
	return t
}

func (t *ItemConditionTag) Name(name string) *ItemConditionTag {
	t.NameCond = name
	return t
}

func (t *ItemConditionTag) Rarity(rarity string) *ItemConditionTag {
	t.RarityCond = rarity
	return t
}

func (t *ItemConditionTag) Corrupted(corrupted bool) *ItemConditionTag {
	t.CorruptedCond = &corrupted
	return t
}

func (t *ItemConditionTag) Shaper() *ItemConditionTag {
	t.ShaperCond = true
	return t
}

func (t *ItemConditionTag) Elder() *ItemConditionTag {
	t.ElderCond = true
	return t
}

func (t *ItemConditionTag) Negate(neg bool) *ItemConditionTag {
	t.Neg = neg
	return t
}

func (t *ItemConditionTag) WithAllSlots() *ItemConditionTag {
	t.AllSlots = true
	return t
}

func (t *ItemConditionTag) WithBothSlots() *ItemConditionTag {
	t.BothSlots = true
	return t
}

func (t *ItemConditionTag) WithExcludeSelf() *ItemConditionTag {
	t.ExcludeSelf = true
	return t
}

func (t *ItemConditionTag) Type() Type {
	return t.TagType
}
