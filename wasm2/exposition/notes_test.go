package exposition

import (
	"strings"
	"testing"
)

const notesBuild = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<PathOfBuilding2>\n\t<Build level=\"1\" className=\"Ranger\"/>\n" +
	"\t<Notes>\n^xE05030Fire^7 notes &amp; more\n\t</Notes>\n\t<Unknown keep=\"1\"/>\n</PathOfBuilding2>\n"

// NotesTab.lua:75-82 (PoB-PoE2) reads the Notes element's text, which xml.lua's ParseXML has trimmed.
func TestNotesAreTheNotesElementsText(t *testing.T) {
	calc, err := ParseBuild(notesBuild)
	if err != nil {
		t.Fatal(err)
	}
	if got := calc.Notes(); got != "^xE05030Fire^7 notes & more" {
		t.Errorf("Notes() = %q", got)
	}
	if got := calc.RichNotesStatus(); got != "none" {
		t.Errorf("a build PoB-PoE2 wrote has rich notes status %q, want none", got)
	}
}

// NotesTab.lua:84-88 (PoB-PoE2) writes the edit buffer as the Notes element's text; the rich document reopens with
// it as long as that text is what it was written with.
func TestRichNotesReopenWithTheBuild(t *testing.T) {
	calc, err := ParseBuild(notesBuild)
	if err != nil {
		t.Fatal(err)
	}
	doc := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"a < b & \"c\""}]}]}`
	calc.SetRichNotes("\n  Plan: a < b & \"c\"\n", doc)
	calc.SetRichNotes("Plan: a < b & \"c\"\n\n", doc)
	saved, err := calc.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(saved, "<RichNotes") != 1 || !strings.Contains(saved, `<Unknown keep="1"/>`) {
		t.Fatalf("saved build:\n%s", saved)
	}
	reopened, err := ParseBuild(saved)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Notes(); got != `Plan: a < b & "c"` {
		t.Errorf("reopened notes %q", got)
	}
	if got := reopened.RichNotesStatus(); got != "match" {
		t.Errorf("reopened rich notes status %q, want match", got)
	}
	if got := reopened.RichNotesDoc(); got != doc {
		t.Errorf("reopened rich doc %q", got)
	}
	reopened.build.Notes.Text = "edited elsewhere"
	if got := reopened.RichNotesStatus(); got != "mismatch" {
		t.Errorf("rich notes status after the text changed %q, want mismatch", got)
	}
}
