package mcp

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// savedCraftedRing is a PoB-crafted ring as PoB saves it: affix records, an XML-escaped name and <ModRange> children.
const savedCraftedRing = `
			Rarity: RARE
Dusk&apos;s Loop
Two-Stone Ring
Crafted: true
Prefix: {range:0.5}IncreasedLife4
Prefix: None
Prefix: None
Suffix: None
Suffix: None
Suffix: None
Item Level: 84
LevelReq: 48
Implicits: 1
{tags:elemental,fire,lightning,resistance}{range:0.5}+(12-16)% to Fire and Lightning Resistances
+62 to maximum Life
			<ModRange id="1" range="0.5" />
			<ModRange id="2" range="0.5" />
		`

// savedCraftedRingText is the same ring as item text.
const savedCraftedRingText = `Rarity: RARE
Dusk's Loop
Two-Stone Ring
Crafted: true
Prefix: {range:0.5}IncreasedLife4
Prefix: None
Prefix: None
Suffix: None
Suffix: None
Suffix: None
Item Level: 84
LevelReq: 48
Implicits: 1
{tags:elemental,fire,lightning,resistance}{range:0.5}+(12-16)% to Fire and Lightning Resistances
+62 to maximum Life`

var ring2ItemRe = regexp.MustCompile(`(?s)(<Item id="13">).*?(</Item>)`)

// importWithRing2 imports the fixture build with Ring 2's saved XML replaced by inner.
func importWithRing2(t *testing.T, s *Server, inner string) string {
	t.Helper()
	raw, err := os.ReadFile(fixture("3.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !ring2ItemRe.Match(raw) {
		t.Fatal("REFUSING: the fixture has no item 13")
	}
	path := filepath.Join(t.TempDir(), "ring.xml")
	if err := os.WriteFile(path, ring2ItemRe.ReplaceAll(raw, []byte("${1}"+inner+"${2}")), 0o600); err != nil {
		t.Fatal(err)
	}
	return call(t, s, "import_build", map[string]any{"source": path})["handle"].(string)
}

// TestScanAffixRoundTripsSavedItem: an affix applied to a PoB-saved crafted item fills one of its affix records and
// adds only that mod's line; export and re-import give the same item and the same stats.
func TestScanAffixRoundTripsSavedItem(t *testing.T) {
	s := testServer(t)
	h := importWithRing2(t, s, savedCraftedRing)
	if got := slotText(t, s, h, "Ring 2"); got != savedCraftedRingText {
		t.Fatalf("Ring 2 reads as\n%s", got)
	}
	list := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"affixes"}, "slots": []string{"Ring 2"}, "stat": "Mana"}))
	if len(list) == 0 || !strings.Contains(list[0]["id"].(string), "|PREFIX:IncreasedMana") {
		t.Fatalf("REFUSING: no mana prefix row: %v", list)
	}
	res := call(t, s, "scan_build", map[string]any{"handle": h, "apply": list[0]["id"]})
	after := slotText(t, s, h, "Ring 2")
	for _, w := range res["warnings_added"].([]any) {
		if strings.Contains(w.(string), "nsupported") {
			t.Fatalf("warning added: %v\n%s", w, after)
		}
	}
	modID := strings.TrimSuffix(strings.SplitN(list[0]["id"].(string), "PREFIX:", 2)[1], "|0.5")
	want := strings.Replace(savedCraftedRingText, "Prefix: None", "Prefix: {range:0.5}"+modID, 1)
	added := addedLines(t, want, after)
	if len(added) != 1 || !regexp.MustCompile(`^\{tags:[^}]*\}\{prefix\}\+\d+ to maximum Mana$`).MatchString(added[0]) {
		t.Fatalf("added %q:\n%s", added, after)
	}

	exported := call(t, s, "export_build", map[string]any{"handle": h, "format": "xml"})["xml"].(string)
	if strings.Contains(exported, "&amp;") || strings.Contains(exported, "&lt;ModRange") {
		t.Fatal("the export double-escapes item text")
	}
	path := filepath.Join(t.TempDir(), "again.xml")
	if err := os.WriteFile(path, []byte(exported), 0o600); err != nil {
		t.Fatal(err)
	}
	h2 := call(t, s, "import_build", map[string]any{"source": path})["handle"].(string)
	if again := slotText(t, s, h2, "Ring 2"); again != after {
		t.Fatalf("re-imported ring\n%s\nwant\n%s", again, after)
	}
	a, b := call(t, s, "get_stats", map[string]any{"handle": h}), call(t, s, "get_stats", map[string]any{"handle": h2})
	if !reflect.DeepEqual(a["stats"], b["stats"]) {
		t.Fatalf("stats differ after the round trip:\n%v\n%v", a["stats"], b["stats"])
	}
}
