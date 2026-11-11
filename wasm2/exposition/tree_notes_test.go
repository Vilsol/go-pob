package exposition

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTreeNodeNotesSaveAsNoteElements(t *testing.T) {
	calc, _ := openFixture(t, "tree_specs.xml")
	calc.SetTreeNodeNote(47175, "first\nsecond line")
	calc.SetTreeNodeNote(3936, "an earlier node")
	if got := calc.TreeNodeNote(47175); got != "first\nsecond line" {
		t.Fatalf("note %q", got)
	}
	xml, err := calc.Serialize()
	mustOK(t, err)
	reopened, err := ParseBuild(xml)
	mustOK(t, err)
	if got := reopened.TreeNodeNote(47175); got != "first\nsecond line" {
		t.Fatalf("reopened note %q", got)
	}
	if strings.Index(xml, `nodeId="3936"`) > strings.Index(xml, `nodeId="47175"`) {
		t.Fatalf("notes not in node order:\n%s", xml)
	}
	calc.SetTreeNodeNote(47175, "")
	calc.SetTreeNodeNote(3936, "")
	xml, err = calc.Serialize()
	mustOK(t, err)
	if strings.Contains(xml, "<Note nodeId") {
		t.Fatalf("a removed note still saves:\n%s", xml)
	}
	// the other spec's notes are its own
	calc.SetActiveTreeSpec(2)
	if got := calc.TreeNodeNote(47175); got != "" {
		t.Fatalf("spec 2 shows spec 1's note %q", got)
	}
}

// PassiveTreeView.lua:556-564, BuildExportPoE2.lua:71-89 (PoB-PoE2): a note set from the tree reaches the .build export
// as its node's additional_text
func TestTreeNodeNoteReachesBuildPlannerExport(t *testing.T) {
	calc, _ := openFixture(t, "tree_specs.xml")
	calc.SetTreeNodeNote(47175, "first\nsecond line")
	_, err := calc.Calculate()
	mustOK(t, err)
	file, err := calc.BuildPlannerExport(buildPlannerMetas[0], 1, 1, 1)
	mustOK(t, err)
	var got struct {
		Passives []map[string]any `json:"passives"`
	}
	mustOK(t, json.Unmarshal([]byte(file.JSON), &got))
	spec, err := calc.spec()
	mustOK(t, err)
	want := spec.Nodes[47175].Node.StringID
	var notes []any
	for _, p := range got.Passives {
		if text, ok := p["additional_text"]; ok {
			if p["id"] != want {
				t.Fatalf("a note on %v", p["id"])
			}
			notes = append(notes, text)
		}
	}
	if len(notes) != 1 || notes[0] != "first\nsecond line" {
		t.Fatalf("notes in the export %v", notes)
	}
}
