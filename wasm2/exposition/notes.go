package exposition

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/Vilsol/go-pob/pob2"
)

// The Notes page's rich document is kept beside the notes text in a RichNotes element PoB-PoE2 does not read, as
// the PoE1 build file keeps it; plainHash says which notes text it was written with.
const (
	richNotesElem    = "RichNotes"
	richNotesFormat  = "tiptap-json"
	richNotesVersion = "1"
	// xml.lua's ParseXML trims a text node of these, so notes saved with them would not reopen as written
	notesEdgeSpace = " \t\n\v\f\r"
)

// Notes is the build's notes text.
//
//pob2:src/Classes/NotesTab.lua:75-82
func (c *Calculator) Notes() string {
	return c.build.Notes.Text
}

func notesHash(notes string) string {
	sum := sha256.Sum256([]byte(notes))
	return hex.EncodeToString(sum[:])
}

func (c *Calculator) richNotes() *pob2.Node {
	for _, kid := range c.build.ExtraKids {
		if kid.Elem == richNotesElem {
			return kid
		}
	}
	return nil
}

// SetRichNotes sets the notes text Save writes, and the rich document it was made from.
//
//pob2:src/Classes/NotesTab.lua:84-88
func (c *Calculator) SetRichNotes(plain string, doc string) {
	plain = strings.Trim(plain, notesEdgeSpace)
	c.build.Notes.Text = plain
	n := c.richNotes()
	if n == nil {
		n = &pob2.Node{Elem: richNotesElem}
		c.build.ExtraKids = append(c.build.ExtraKids, n)
	}
	n.Attrib = map[string]string{"format": richNotesFormat, "v": richNotesVersion, "plainHash": notesHash(plain)}
	n.Kids = []*pob2.Node{{Text: doc}}
}

// RichNotesStatus is "none" without a rich document, "mismatch" when the notes text changed since it was written,
// and "match" otherwise.
func (c *Calculator) RichNotesStatus() string {
	n := c.richNotes()
	if n == nil || n.Attrib["format"] != richNotesFormat || n.Attrib["v"] != richNotesVersion || c.RichNotesDoc() == "" {
		return "none"
	}
	if n.Attrib["plainHash"] != notesHash(c.build.Notes.Text) {
		return "mismatch"
	}
	return "match"
}

func (c *Calculator) RichNotesDoc() string {
	n := c.richNotes()
	if n == nil {
		return ""
	}
	var doc strings.Builder
	for _, kid := range n.Kids {
		if kid.Elem == "" {
			doc.WriteString(kid.Text)
		}
	}
	return doc.String()
}
