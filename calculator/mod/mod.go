package mod

type Mod interface {
	Name() string
	Type() Type
	Source(source Source) Mod
	Flag(flag MFlag) Mod
	KeywordFlag(keywordFlags KeywordFlag) Mod
	Tag(tag Tag) Mod
	Flags() MFlag
	KeywordFlags() KeywordFlag
	GetSource() Source
	Tags() []Tag
	Value() interface{}
}
