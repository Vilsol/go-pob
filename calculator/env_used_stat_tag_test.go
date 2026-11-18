package calculator

import (
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestStatTagsRecordOnlyASingleStat: Calcs.lua:507-515's addStatTag reads statList only behind `tag.varList`, which a
// PerStat or StatThreshold tag never carries, so only a single `stat` is recorded. The want is the pinned Lua
// function run under luajit on the same three tags: it records "Life" alone.
//
//pob:src/Modules/Calcs.lua:507-515
func TestStatTagsRecordOnlyASingleStat(t *testing.T) {
	env := &Environment{}
	env.initUsedTables()
	for _, tag := range []mod.Tag{
		mod.PerStat(250, "Str", "Dex", "Int"),
		mod.PerStat(10, "Life"),
		mod.StatThreshold("", 10).Stats("Str", "Dex"),
	} {
		env.addModTagsUsed(true, mod.NewFloat("Damage", mod.TypeIncrease, 1).Source("Item").Tag(tag))
	}
	got := make([]string, 0, len(env.PerStatsUsed))
	for k := range env.PerStatsUsed {
		got = append(got, k)
	}
	slices.Sort(got)
	if !slices.Equal(got, []string{"Life"}) {
		t.Errorf("perStatsUsed = %v, PoB [Life]", got)
	}
}
