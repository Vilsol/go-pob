package calculator

import (
	"sort"
	"strings"
	"testing"
)

// TestHardCodedGemLevelRowsAgreeWithTheCatchAll is the adjudication of audit row ed078a2d's second half.
func TestHardCodedGemLevelRowsAgreeWithTheCatchAll(t *testing.T) {
	requireGameData(t, "3.29")

	phrases := []string{
		"minion skill",
		"spell skill",
		"physical spell skill",
		"physical skill",
		"lightning spell skill",
		"lightning skill",
		"cold spell skill",
		"cold skill",
		"fire spell skill",
		"fire skill",
		"chaos spell skill",
		"chaos skill",
		"strength skill",
		"dexterity skill",
		"intelligence skill",
	}

	if got, want := asSet(gemPropertyKeywords(t, "+1 to Level of all Elemental Skill Gems")), asSet(gemKeywordList("elemental skill")); !equalSets(got, want) {
		t.Fatalf("REFUSING: control (catch-all) keywords = %v, want %v", got, want)
	}

	for _, phrase := range phrases {
		t.Run(phrase, func(t *testing.T) {
			line := "+1 to Level of all " + strings.Title(phrase) + " Gems" //nolint:staticcheck // ASCII phrase list
			got := asSet(gemPropertyKeywords(t, line))
			want := asSet(gemKeywordList(phrase))
			if len(want) == 0 || len(got) == 0 {
				t.Fatalf("REFUSING: %q produced keywords %v against expected %v; an empty set on either side makes the comparison vacuous", line, got, want)
			}
			if !equalSets(got, want) {
				t.Errorf("%q -> keywords %v, but the catch-all (ModParser.lua:3228) derives %v from the same phrase", line, got, want)
			}
		})
	}
}

func asSet(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func equalSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
