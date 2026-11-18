package calculator

import (
	"slices"
	"strings"
	"testing"
)

// TestBuildRawBaseLines: Item.lua:1016-1021, 1980-1984. PoB's BuildRaw of its own Allelopathy text keeps both
// variant base lines after the variant selection.
func TestBuildRawBaseLines(t *testing.T) {
	initItemsTabData(t)
	raw := strings.Join([]string{"Rarity: UNIQUE", "Allelopathy", "Satin Gloves", "Variant: Pre 3.19.0", "Variant: Pre 3.26.0",
		"Variant: Current", "Selected Variant: 3", "{variant:2,3}Satin Gloves", "{variant:1}Sorcerer Gloves", "Quality: 0",
		"Sockets: B-B-B-B", "LevelReq: 41", "Implicits: 0", "{variant:1,2}Grants Level 22 Blight", "{variant:3}Grants Level 25 Blight",
		"{range:0.5}(100-120)% increased Energy Shield"}, "\n")
	out := strings.Split(buildRawText(ParseItem(raw)), "\n")
	at := slices.Index(out, "Selected Variant: 3")
	if at < 0 || at+2 >= len(out) || !slices.Equal(out[at+1:at+3], []string{"{variant:2,3}Satin Gloves", "{variant:1}Sorcerer Gloves"}) {
		t.Fatalf("variant base lines lost:\n%s", strings.Join(out, "\n"))
	}
}
