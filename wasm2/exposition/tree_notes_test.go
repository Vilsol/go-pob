package exposition

import (
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
