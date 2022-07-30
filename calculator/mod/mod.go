package mod

import "go-pob/data"

type Mod interface {
	Name() string
	Type() Type
	Source(source Source) Mod
	Flag(flag data.ModFlag) Mod
	KeywordFlag(keywordFlags KeywordFlag) Mod
	Tag(tag Tag) Mod
	Flags() data.ModFlag
	KeywordFlags() KeywordFlag
	GetSource() Source
	Tags() []Tag
	Value() interface{}
}
