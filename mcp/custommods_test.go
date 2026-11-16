package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureWithConfig(t *testing.T, configXML string) string {
	t.Helper()
	raw, err := os.ReadFile(fixture("3.xml"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(raw)
	if set, ok := strings.CutPrefix(configXML, "set:"); ok {
		out = strings.Replace(out, "\t<Config>", `\t<Config activeConfigSet="5"><ConfigSet id="5" title="Mapping">`, 1)
		configXML = set + "</ConfigSet>"
	}
	out = strings.Replace(out, "\t</Config>", configXML+"\n\t</Config>", 1)
	path := filepath.Join(t.TempDir(), "custom.xml")
	if err := os.WriteFile(path, []byte(out), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func importedLife(t *testing.T, s *Server, path string) (string, float64) {
	t.Helper()
	res := call(t, s, "import_build", map[string]any{"source": path})
	return res["handle"].(string), stat(res["summary"].(map[string]any), "Life")
}

func TestImportKeepsCustomModifiers(t *testing.T) {
	s := testServer(t)
	_, base := importedLife(t, s, fixture("3.xml"))
	for name, cfg := range map[string]string{
		"legacy input": `<Input string="+1000 to maximum Life" name="customMods"/>`,
		"block":        `<CustomModifierBlock title="Default" enabled="true">+1000 to maximum Life</CustomModifierBlock>`,
		"config set":   `set:<CustomModifierBlock title="Default" enabled="true">+1000 to maximum Life` + "\n" + `+10% to Fire Resistance</CustomModifierBlock>`,
	} {
		h, life := importedLife(t, s, fixtureWithConfig(t, cfg))
		if life <= base+900 {
			t.Fatalf("%s: Life %v, without the custom line %v", name, life, base)
		}
		res := call(t, s, "export_build", map[string]any{"handle": h, "format": "xml"})
		if !strings.Contains(res["xml"].(string), "+1000 to maximum Life") {
			t.Fatalf("%s: the export dropped the custom modifier", name)
		}
	}
}

func TestSetConfigCustomMods(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "customMods",
		"value": "# placeholder resists until the gear has them\n+1000 to maximum Life\nThis line names no modifier at all"})})
	if d := deltaOf(res, "Life"); d < 900 {
		t.Fatalf("Life delta %v: %v", d, res)
	}
	if !hasWarning(res["warnings_added"], "This line names no modifier at all") {
		t.Fatalf("the unparsed custom line was not reported: %v", res["warnings_added"])
	}
	if hasWarning(res["warnings_added"], "placeholder resists") {
		t.Fatalf("a # comment line was reported as unsupported: %v", res["warnings_added"])
	}
	cfg := call(t, s, "get_build", map[string]any{"handle": h, "section": "config"})
	found := false
	for _, r := range asList(cfg["config"]) {
		if row := r.(map[string]any); row["var"] == "customMods" && strings.Contains(row["value"].(string), "+1000 to maximum Life") {
			found = true
		}
	}
	if !found {
		t.Fatalf("config section = %v", cfg["config"])
	}
	res = call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "unset_config", "var": "customMods"})})
	if d := deltaOf(res, "Life"); d > -900 {
		t.Fatalf("unset Life delta %v", d)
	}
}

func deltaOf(res map[string]any, stat string) float64 {
	for _, r := range asList(res["delta"]) {
		if row := r.(map[string]any); row["stat"] == stat {
			return row["after"].(float64) - row["before"].(float64)
		}
	}
	return 0
}

func TestUnparsedNodeLinesAreWarnings(t *testing.T) {
	s := testServer(t)
	res := call(t, s, "import_build", map[string]any{"source": fixture("12.xml")})
	h := res["handle"].(string)
	const want = "Unsupported mod in Passive Node (Disciple of the Slaughter): 8% chance to gain a Frenzy Charge on Kill"
	if w := call(t, s, "get_build", map[string]any{"handle": h, "section": "warnings"}); !hasWarning(w["warnings"], want) {
		t.Fatalf("warnings = %v", w["warnings"])
	}
	edit := call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_level", "level": 95})})
	if hasWarning(edit["warnings_removed"], "Unsupported mod") || hasWarning(edit["warnings_added"], "Unsupported mod") {
		t.Fatalf("an unrelated edit moved the unsupported lines: +%v -%v", edit["warnings_added"], edit["warnings_removed"])
	}
}

func TestUnparsedItemLinesAreWarnings(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	text := "Rarity: RARE\nTest Loop\nTwo-Stone Ring\n+50 to maximum Life\nThis line names no modifier at all"
	res := call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_text", "text": text, "slot": "Ring 1"})})
	if !hasWarning(res["warnings_added"], "Unsupported mod in Ring 1 (Test Loop, Two-Stone Ring): This line names no modifier at all") {
		t.Fatalf("warnings_added = %v", res["warnings_added"])
	}
}

func TestCustomModDefencesWarn(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Witch"})["handle"].(string)
	res := call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "customMods",
		"value": "+40% to Fire Resistance\n+10% to all Elemental Resistances\n+50 to maximum Life\n10% increased maximum Life\n10% increased Spell Damage"})})
	var w string
	for _, l := range asList(res["warnings_added"]) {
		if s, _ := l.(string); strings.HasPrefix(s, "Custom modifiers supply") {
			w = s
		}
	}
	for _, want := range []string{"Fire Res +50%", "Cold Res +10%", "Lightning Res +10%", "Life +50", "Life +10% inc"} {
		if !strings.Contains(w, want) {
			t.Fatalf("custom defence warning %q lacks %q (warnings %v)", w, want, res["warnings_added"])
		}
	}
	if strings.Contains(w, "Spell") || strings.Contains(w, "Chaos") {
		t.Fatalf("custom defence warning names a non-defence or absent stat: %q", w)
	}
}
