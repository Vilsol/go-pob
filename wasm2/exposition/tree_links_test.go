package exposition

import (
	"strings"
	"testing"
)

// TreeTab.lua:739-753, 802-807 (PoB-PoE2): a ruthless link and a poeplanner link of a known version name trees PoB-PoE2
// cannot load; it raises there, and the import is refused here, adding no spec.
func TestImportTreeLinkRefusesTreesThatDoNotLoad(t *testing.T) {
	calc, _ := openFixture(t, "tree_specs.xml")
	link, err := calc.ExportTreeLink()
	mustOK(t, err)
	for _, l := range []string{
		"https://www.pathofexile.com/passive-skill-tree/ruthless/" + link,
		"https://www.pathofexile.com/passive-skill-tree/0.20.0/" + link,
		"https://poeplanner.com/AAAAAwAYAAEAAAAAAAAAAAAA",
	} {
		msg, err := calc.ImportTreeLink("x", l)
		mustOK(t, err)
		if !strings.Contains(msg, "is not available") || len(calc.build.Tree.Specs) != 2 {
			t.Fatalf("%s: message %q, %d specs", l, msg, len(calc.build.Tree.Specs))
		}
	}
}
