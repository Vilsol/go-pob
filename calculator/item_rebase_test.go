package calculator

import (
	"slices"
	"strings"
	"testing"
)

func seriesAt(t *testing.T, c AffixControl, label string) int {
	t.Helper()
	at := slices.IndexFunc(c.Series, func(s AffixSeries) bool { return s.Label == label })
	if at < 0 {
		t.Fatalf("REFUSING: no %q entry", label)
	}
	return at
}

// TestRebaseItemText: a crafted circlet moved onto an armour helmet keeps its life prefix.
func TestRebaseItemText(t *testing.T) {
	initItemsTabData(t)
	circlet, _ := CraftItemText(2, "Doom Crown", "Hubris Circlet")
	c, _ := ItemAffixControls(circlet)
	withLife, _ := AffixEditText(circlet, 0, seriesAt(t, c.Controls[0], "+# to maximum Life"), 0.5)
	withES, _ := AffixEditText(withLife, 1, seriesAt(t, c.Controls[1], "#% increased Energy Shield"), 0.5)

	choices := RebaseChoices(withES)
	if !slices.Contains(choices, "Hubris Circlet") || !slices.Contains(choices, "Royal Burgonet") || slices.Contains(choices, "Vaal Regalia") {
		t.Fatalf("choices %q", choices)
	}
	res, ok := RebaseItemText(withES, "Royal Burgonet")
	if !ok {
		t.Fatalf("rebase refused:\n%s", withES)
	}
	item := ParseItem(res.Text)
	if item.BaseName != "Royal Burgonet" || item.Title != "Doom Crown" {
		t.Fatalf("base %q title %q:\n%s", item.BaseName, item.Title, res.Text)
	}
	if len(res.Dropped) != 1 || !strings.Contains(res.Dropped[0], "increased Energy Shield") {
		t.Fatalf("dropped %q", res.Dropped)
	}
	rc, _ := ItemAffixControls(res.Text)
	if got := rc.Controls[0].Series[rc.Controls[0].Sel].Label; got != "+# to maximum Life" {
		t.Fatalf("first prefix %q:\n%s", got, res.Text)
	}
	if rc.Controls[1].Sel != 0 {
		t.Fatalf("second prefix kept:\n%s", res.Text)
	}
	if lines := item.ExplicitModLines; len(lines) != 1 || !strings.Contains(lines[0].Line, "to maximum Life") {
		t.Fatalf("explicits %v", lines)
	}
	if _, ok := RebaseItemText(res.Text, "Vaal Regalia"); ok {
		t.Fatal("rebased onto a body armour")
	}
}

// TestRebaseItemImplicits: a ring's base implicit is swapped for the new base's; a custom implicit stays.
func TestRebaseItemImplicits(t *testing.T) {
	initItemsTabData(t)
	ring, _ := CraftItemText(2, "", "Two-Stone Ring (Cold/Lightning)")
	ring = strings.Replace(ring, "Implicits: 1\n", "Implicits: 2\n{custom}+5 to Strength\n", 1)
	res, ok := RebaseItemText(ring, "Ruby Ring")
	if !ok {
		t.Fatalf("rebase refused:\n%s", ring)
	}
	lines := ParseItem(res.Text).ImplicitModLines
	got := make([]string, 0, len(lines))
	for _, ml := range lines {
		got = append(got, ml.Line)
	}
	if len(got) != 2 || !strings.Contains(got[0], "Fire Resistance") || got[1] != "+5 to Strength" {
		t.Fatalf("implicits %q:\n%s", got, res.Text)
	}
}
