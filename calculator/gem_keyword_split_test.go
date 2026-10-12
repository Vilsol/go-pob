package calculator

import (
	"reflect"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestGemKeywordListSplitsOnWhitespace pins ModParser.lua:3232-3236 — `for tag in skill:gmatch("%S+")`, NON-SPACE runs, not `%w+`.
func TestGemKeywordListSplitsOnWhitespace(t *testing.T) {
	requireGameData(t, "3.29")

	if got := gemPropertyKeywords(t, "+1 to Level of all Elemental Support Gems"); !reflect.DeepEqual(got, []string{"elemental", "support"}) {
		t.Fatalf("REFUSING: control keyword list = %v, want [elemental support]; the row itself is broken", got)
	}

	got := gemPropertyKeywords(t, "+1 to Level of all non-Exceptional Support Gems")
	want := []string{"non-exceptional", "support"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("keyword list = %v, want %v (ModParser.lua:3233 splits on %%S+)", got, want)
	}
}

func gemPropertyKeywords(t *testing.T, line string) []string {
	t.Helper()
	e := ParseMod(line, false)
	if e == nil {
		t.Fatalf("no entry: %s", line)
	}
	for _, m := range e.ModList {
		if m.Name() != "GemProperty" {
			continue
		}
		if gp, ok := m.Value().ValueList.(mod.GemProperty); ok {
			return gp.KeywordList
		}
	}
	t.Fatalf("no GemProperty produced: %s", line)
	return nil
}
