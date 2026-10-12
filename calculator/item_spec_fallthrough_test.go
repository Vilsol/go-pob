package calculator

import (
	"regexp"
	"strings"
	"testing"
)

var specShapedLine = regexp.MustCompile(`^[A-Za-z ()]+:? ?: `)

func leakedSpecLines(i *Item) []string {
	var out []string
	for _, group := range [][]ModLine{i.ImplicitModLines, i.ExplicitModLines, i.EnchantModLines, i.ScourgeModLines} {
		for _, ml := range group {
			if ml.ModList == nil && specShapedLine.MatchString(ml.Line) {
				out = append(out, ml.Line)
			}
		}
	}
	return out
}

// TestUnrecognisedSpecDoesNotOpenTheExplicitSection covers applySpec's default arm.
func TestUnrecognisedSpecDoesNotOpenTheExplicitSection(t *testing.T) {
	requireGameData(t, "3.29")

	const raw = "Rarity: RARE\nHate Carapace\nTwilight Regalia\n" +
		"Energy Shield: 981\nEnergyShieldBasePercentile: 0.9982\n" +
		"Intangibility: 17%\nMemory Strands: 15\n" +
		"Unique ID: 0000\nItem Level: 86\nQuality: 20\nSockets: B-B-W-R\nLevelReq: 84\nImplicits: 1\n" +
		"+2% to maximum Lightning Resistance\n+93 to maximum Energy Shield\n"

	control := strings.Replace(strings.Replace(raw, "Intangibility: 17%\n", "", 1), "Memory Strands: 15\n", "", 1)
	if leaked := leakedSpecLines(ParseItem(control)); len(leaked) != 0 {
		t.Fatalf("control leaked %d spec lines into the mod lists: %q", len(leaked), leaked)
	}

	item := ParseItem(raw)
	if leaked := leakedSpecLines(item); len(leaked) != 0 {
		t.Errorf("an unrecognised spec pushed %d later spec lines into the mod lists: %q\n"+
			"(Item.lua:825-826 and :645-646 have branches for these; the default arm at :838 is for explicits)",
			len(leaked), leaked)
	}
}
