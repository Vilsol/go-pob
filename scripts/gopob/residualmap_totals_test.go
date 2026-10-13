package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResidualMapReadsTotalsWhenPresent pins the fix for the 1.755x pp inflation.
func TestResidualMapReadsTotalsWhenPresent(t *testing.T) {
	dir := t.TempDir()

	if _, _, ok := rmReadTotals(dir); ok {
		t.Fatalf("rmReadTotals reported ok with no totals.tsv — the absent branch is what keeps " +
			"the difftest byte-identical")
	}

	body := "scope\tseen\tok\tmiss\tnan\n" +
		"ALL\t7310135\t7236176\t73959\t0\n" +
		"Enemy.\t389160\t389160\t0\t0\n" +
		"MainHand.\t252334\t246059\t6275\t0\n" +
		"player\t5882986\t5826855\t56131\t0\n"
	if err := os.WriteFile(filepath.Join(dir, "totals.tsv"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	graded, byActor, ok := rmReadTotals(dir)
	if !ok {
		t.Fatalf("rmReadTotals did not read a well-formed totals.tsv")
	}
	if graded != 7310135 {
		t.Errorf("graded = %d, want 7310135 (the ALL row's seen)", graded)
	}
	for k, want := range map[string]int{
		"player (unprefixed)": 5882986,
		"MainHand.":           252334,
		"Enemy.":              389160,
	} {
		if got := byActor[k]; got != want {
			t.Errorf("byActor[%q] = %d, want %d", k, got, want)
		}
	}
	if _, bad := byActor["player"]; bad {
		t.Errorf(`byActor still carries the raw "player" key; the BY ACTOR TABLE never looks it up`)
	}
}
