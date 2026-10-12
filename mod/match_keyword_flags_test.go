package mod

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

// TestMatchKeywordFlagsMatchAll: Data/Global.lua:146-148 reads the MatchAll bit BEFORE masking it out.
//
//	local matchAll = band(modKeywordFlags, KeywordFlag.MatchAll) ~= 0
func TestMatchKeywordFlagsMatchAll(t *testing.T) {
	const both = KeywordFlagCurse | KeywordFlagAura
	modFlags := both | KeywordFlagMatchAll

	testza.AssertTrue(t, MatchKeywordFlags(both, modFlags))
	testza.AssertTrue(t, MatchKeywordFlags(both|KeywordFlagSpell, modFlags))

	testza.AssertFalse(t, MatchKeywordFlags(KeywordFlagCurse, modFlags))
	testza.AssertFalse(t, MatchKeywordFlags(KeywordFlagAura, modFlags))

	testza.AssertFalse(t, MatchKeywordFlags(KeywordFlagSpell, modFlags))

	testza.AssertTrue(t, MatchKeywordFlags(both|KeywordFlagMatchAll, modFlags))
}

// TestMatchKeywordFlagsAnyOf: Without the MatchAll bit the rule is ANY-of, and an unflagged mod matches anything.
func TestMatchKeywordFlagsAnyOf(t *testing.T) {
	const both = KeywordFlagCurse | KeywordFlagAura

	testza.AssertTrue(t, MatchKeywordFlags(KeywordFlagCurse, both))
	testza.AssertTrue(t, MatchKeywordFlags(KeywordFlagAura, both))
	testza.AssertFalse(t, MatchKeywordFlags(KeywordFlagSpell, both))

	testza.AssertTrue(t, MatchKeywordFlags(0, 0))
	testza.AssertTrue(t, MatchKeywordFlags(KeywordFlagSpell, 0))

	testza.AssertFalse(t, MatchKeywordFlags(0, KeywordFlagTotem))
	testza.AssertFalse(t, MatchKeywordFlags(0, both|KeywordFlagMatchAll))
}
