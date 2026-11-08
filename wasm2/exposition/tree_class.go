package exposition

import (
	"slices"
	"strings"
)

// TreeClassRow is one class dropdown entry; Ascendancies is the ascendancy dropdown it fills, by ascendClassId.
type TreeClassRow struct {
	Label        string
	ClassID      int
	Ascendancies []string
}

//pob2:src/Modules/Build.lua:1519-1540
func TreeClassList() ([]TreeClassRow, error) {
	t, err := loadTree()
	if err != nil {
		return nil, err
	}
	out := make([]TreeClassRow, 0, len(t.Classes))
	for classID, class := range t.Classes {
		row := TreeClassRow{Label: class.Name, ClassID: classID}
		for _, a := range class.Classes {
			row.Ascendancies = append(row.Ascendancies, a.Name)
		}
		out = append(out, row)
	}
	slices.SortFunc(out, func(a, b TreeClassRow) int { return strings.Compare(a.Label, b.Label) })
	return out, nil
}

// TreeClassChoice answers the Class Change popup: 0 asks it, 1 is Continue, 2 is Connect Path.
const (
	TreeClassAsk = iota
	TreeClassContinue
	TreeClassConnect
)

// ChangeTreeClass is the class dropdown; it reports whether the change waits on the Class Change popup.
//
//pob2:src/Modules/Build.lua:246-272
func (c *Calculator) ChangeTreeClass(classID int, choice int) (bool, error) {
	spec, err := c.spec()
	if err != nil {
		return false, err
	}
	if classID == spec.CurClassID || spec.Tree.Classes[classID] == nil {
		return false, nil
	}
	if used, _, _, _, _, _ := spec.CountAllocNodes(); used != 0 && !spec.IsClassConnected(classID) {
		switch choice {
		case TreeClassAsk:
			return true, nil
		case TreeClassConnect:
			ok, err := spec.ConnectToClass(classID)
			if err != nil || !ok {
				return false, err
			}
		}
	}
	spec.SelectClass(classID)
	c.addTreeUndo()
	c.saveSpec(spec)
	return false, nil
}

// ChangeTreeAscendancy is the ascendancy dropdown; ascendClassID indexes the current class's ascendancies.
//
//pob2:src/Modules/Build.lua:273-278
func (c *Calculator) ChangeTreeAscendancy(ascendClassID int) error {
	spec, err := c.spec()
	if err != nil {
		return err
	}
	spec.SelectAscendClass(ascendClassID)
	c.addTreeUndo()
	c.saveSpec(spec)
	return nil
}

func (c *Calculator) addTreeUndo() {
	if c.undo != nil {
		c.undo.AddUndoState(c.env.Spec, false)
	}
}
