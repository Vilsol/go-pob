package poe2

import "sort"

// ModCacheRow is one src/Data/ModCache.lua entry: c[Line] = { mods, extra }. Mods is PoB's mod list as
// JSON ("" when PoB caches nil); a tag's skillType is its SkillType name.
type ModCacheRow struct {
	Line     string
	Mods     string
	Extra    string
	HasExtra bool
}

// ModCacheByLine is modLib.parseModCache[line].
//
//pob2:block:src/Modules/ModParser.lua:7435-7454
func ModCacheByLine(line string) (*ModCacheRow, bool) {
	i := sort.Search(len(ModCacheRows), func(i int) bool { return ModCacheRows[i].Line >= line })
	if i < len(ModCacheRows) && ModCacheRows[i].Line == line {
		return &ModCacheRows[i], true
	}
	return nil, false
}

// QuestReward is one data.questRewards entry; NoConfig is `useConfig = false`.
//
//pob2:src/Data/QuestRewards.lua:1-305
type QuestReward struct {
	Act         float64
	Description string
	Area        string
	Info        string
	AreaLevel   float64
	Stat        string
	Options     []string
	QuestPoints float64
	NoConfig    bool
}
