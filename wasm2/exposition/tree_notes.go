package exposition

import (
	"slices"

	"github.com/Vilsol/go-pob/pob2"
)

// TreeNodeNote is the active spec's author note on a node, empty when it has none.
//
//pob2:block:src/Classes/PassiveSpec.lua:161-171
func (c *Calculator) TreeNodeNote(nodeID int) string {
	s := &c.build.Tree.Specs[c.activeSpecIndex()-1]
	if s.NodeNotes == nil {
		return ""
	}
	for _, n := range s.NodeNotes.Notes {
		if n.NodeID == nodeID {
			if n.Text == "" && n.Attr != nil {
				return *n.Attr
			}
			return n.Text
		}
	}
	return ""
}

// SetTreeNodeNote is the Shift+right-click note popup's Save; an empty note removes it.
//
//pob2:block:src/Classes/PassiveTreeView.lua:556-564
//pob2:block:src/Modules/Main.lua:1818-1823
func (c *Calculator) SetTreeNodeNote(nodeID int, text string) {
	s := &c.build.Tree.Specs[c.activeSpecIndex()-1]
	var notes []pob2.NodeNote
	if s.NodeNotes != nil {
		notes = slices.DeleteFunc(s.NodeNotes.Notes, func(n pob2.NodeNote) bool { return n.NodeID == nodeID })
	}
	if text != "" {
		notes = append(notes, pob2.NodeNote{NodeID: nodeID, Text: text})
		slices.SortFunc(notes, func(a, b pob2.NodeNote) int { return a.NodeID - b.NodeID })
	}
	// PassiveSpec.lua:370-378: Save writes <Notes> only when a note is set
	if len(notes) == 0 {
		s.NodeNotes = nil
		return
	}
	s.NodeNotes = &pob2.NodeNotes{Notes: notes}
}
