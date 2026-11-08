package exposition

import (
	"slices"
	"strconv"

	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/pob2"
)

// TreeSpecsView is the tree list: TreeTab:GetSpecList's labels and the active spec's 1-based index.
type TreeSpecsView struct {
	Labels []string
	Active int
}

// treeVersionDisplay is treeVersions[v].display, which GameVersions.lua sets to the version key itself.
//
//pob2:block:src/GameVersions.lua:18-39
func treeVersionDisplay(v string) string { return v }

func specTitle(s *pob2.Spec) string {
	if s.Title == nil {
		return "Default"
	}
	return *s.Title
}

func specTreeVersion(s *pob2.Spec) string {
	if s.TreeVersion == nil {
		return defaultTreeVersion
	}
	return *s.TreeVersion
}

// defaultTreeVersion is the version Load gives a <Spec> without a treeVersion.
//
//pob2:block:src/GameVersions.lua:15
const defaultTreeVersion = "0_1"

//pob2:src/Classes/TreeTab.lua:485-491
func (c *Calculator) TreeSpecs() TreeSpecsView {
	out := TreeSpecsView{Active: c.activeSpecIndex()}
	for i := range c.build.Tree.Specs {
		s := &c.build.Tree.Specs[i]
		label := specTitle(s)
		if v := specTreeVersion(s); v != TreeVersion {
			label = "[" + treeVersionDisplay(v) + "] " + label
		}
		out.Labels = append(out.Labels, label)
	}
	return out
}

func (c *Calculator) activeSpecIndex() int {
	idx := c.build.Tree.ActiveSpec
	if idx < 1 || idx > len(c.build.Tree.Specs) {
		return 1
	}
	return idx
}

// TreeSpecRow is what PassiveSpecListControl's rows and the tree list's tooltip show of one spec.
type TreeSpecRow struct {
	Title           string
	ClassName       string
	AscendClassName string
	Used            int
	Sockets         int
	Version         string
}

// TreeSpecRows loads every spec, as the tree list holds each as a PassiveSpec.
//
//pob2:block:src/Classes/PassiveSpecListControl.lua:77-85
//pob2:block:src/Classes/TreeTab.lua:67-78
func (c *Calculator) TreeSpecRows() ([]TreeSpecRow, error) {
	out := make([]TreeSpecRow, 0, len(c.build.Tree.Specs))
	for i := range c.build.Tree.Specs {
		spec, err := c.specAt(i + 1)
		if err != nil {
			return nil, err
		}
		used, _, _, sockets, _, _ := spec.CountAllocNodes()
		out = append(out, TreeSpecRow{
			Title: specTitle(&c.build.Tree.Specs[i]), ClassName: spec.CurClassName, AscendClassName: spec.CurAscendClassName,
			Used: used, Sockets: sockets, Version: treeVersionDisplay(specTreeVersion(&c.build.Tree.Specs[i])),
		})
	}
	return out, nil
}

// specAt is the PassiveSpec of the idx-th <Spec>: the calculated one for the active spec, else a fresh Load.
func (c *Calculator) specAt(idx int) (*tree2.PassiveSpec, error) {
	if idx == c.activeSpecIndex() && c.env != nil && c.env.Spec != nil {
		return c.env.Spec, nil
	}
	t, err := loadTree()
	if err != nil {
		return nil, err
	}
	spec := tree2.NewPassiveSpec(t)
	if c.items != nil {
		spec.Items = c.items.SpecItem
	}
	if err := spec.Load(specInput(&c.build.Tree.Specs[idx-1])); err != nil {
		return nil, err
	}
	return spec, nil
}

// specInput is the <Spec> as PassiveSpec:Load reads it; calculator2's loadSpec reads the active one the same way.
func specInput(s *pob2.Spec) tree2.SpecInput {
	in := tree2.SpecInput{
		Nodes:                s.Nodes,
		MasteryEffects:       s.MasteryEffects,
		AscendancyInternalID: s.AscendancyInternalID,
		WeaponSets:           map[int]string{},
	}
	if s.Title != nil {
		in.Title = *s.Title
	}
	if s.TreeVersion != nil {
		in.TreeVersion = *s.TreeVersion
	}
	itoa := func(v *int) *string {
		if v == nil {
			return nil
		}
		out := strconv.Itoa(*v)
		return &out
	}
	in.ClassID = itoa(s.ClassID)
	in.AscendClassID = itoa(s.AscendClassID)
	in.ClassInternalID = itoa(s.ClassInternalID)
	if !s.SecondaryAscendClassID.IsNil {
		in.SecondaryAscendClassID = itoa(&s.SecondaryAscendClassID.V)
	}
	if s.URL != nil {
		in.URL = &s.URL.Text
	}
	if s.WeaponSet1 != nil {
		in.WeaponSets[1] = s.WeaponSet1.Nodes
	}
	if s.WeaponSet2 != nil {
		in.WeaponSets[2] = s.WeaponSet2.Nodes
	}
	if s.Sockets != nil {
		for _, sock := range s.Sockets.Sockets {
			in.Sockets = append(in.Sockets, tree2.SocketInput{NodeID: strconv.Itoa(sock.NodeID), ItemID: strconv.Itoa(sock.ItemID)})
		}
	}
	if s.Overrides != nil {
		for _, o := range s.Overrides.AttributeOverrides {
			in.AttributeOverrides = append(in.AttributeOverrides, tree2.AttributeOverrideInput{StrNodes: o.StrNodes, DexNodes: o.DexNodes, IntNodes: o.IntNodes})
		}
	}
	return in
}

// newSpecElement is PassiveSpec:Save of a spec the tree list made, into a fresh <Spec>.
func newSpecElement(spec *tree2.PassiveSpec, title string, notes *pob2.NodeNotes) pob2.Spec {
	version := spec.TreeVersion
	out := pob2.Spec{Title: &title, TreeVersion: &version, SecondaryAscendClassID: pob2.Nil[int]{IsNil: true}, NodeNotes: notes}
	writeSpec(&out, spec)
	save := spec.Save()
	out.Sockets = &pob2.Sockets{}
	for _, sock := range save.Sockets {
		node, _ := strconv.Atoi(sock.NodeID)
		item, _ := strconv.Atoi(sock.ItemID)
		out.Sockets.Sockets = append(out.Sockets.Sockets, pob2.Socket{NodeID: node, ItemID: item})
	}
	return out
}

// SetActiveTreeSpec is the tree list's selection; the build recalculates against the spec.
//
//pob2:src/Classes/TreeTab.lua:541-579
func (c *Calculator) SetActiveTreeSpec(specID int) {
	c.build.Tree.ActiveSpec = max(1, min(specID, len(c.build.Tree.Specs)))
	// Go must differ: the undo history is one handler, so a switch starts the spec's afresh
	c.undo = nil
}

// NewTreeSpec is the tree list's New: a spec of the current class and ascendancy, added at the end.
//
//pob2:src/Classes/PassiveSpecListControl.lua:39-46
//pob2:block:src/Classes/PassiveSpecListControl.lua:57-67
func (c *Calculator) NewTreeSpec(title string) error {
	t, err := loadTree()
	if err != nil {
		return err
	}
	cur, err := c.spec()
	if err != nil {
		return err
	}
	spec := tree2.NewPassiveSpec(t)
	spec.SelectClass(cur.CurClassID)
	spec.SelectAscendClass(cur.CurAscendClassID)
	spec.SelectSecondaryAscendClass(0)
	c.build.Tree.Specs = append(c.build.Tree.Specs, newSpecElement(spec, title, nil))
	return nil
}

// CopyTreeSpec is the tree list's Copy of the specID-th spec, added at the end under title.
//
//pob2:src/Classes/PassiveSpecListControl.lua:16-23
//pob2:block:src/Classes/PassiveSpecListControl.lua:57-67
func (c *Calculator) CopyTreeSpec(specID int, title string) error {
	if specID < 1 || specID > len(c.build.Tree.Specs) {
		return nil
	}
	t, err := loadTree()
	if err != nil {
		return err
	}
	from, err := c.specAt(specID)
	if err != nil {
		return err
	}
	spec := tree2.NewPassiveSpec(t)
	spec.Items = from.Items
	for node, item := range from.Jewels {
		spec.Jewels[node] = item
	}
	spec.RestoreUndoState(from.CreateUndoState())
	var notes *pob2.NodeNotes
	if n := c.build.Tree.Specs[specID-1].NodeNotes; n != nil {
		notes = &pob2.NodeNotes{Notes: slices.Clone(n.Notes)}
	}
	c.build.Tree.Specs = append(c.build.Tree.Specs, newSpecElement(spec, title, notes))
	return nil
}

// RenameTreeSpec is the rename popup's Save.
//
//pob2:block:src/Classes/PassiveSpecListControl.lua:57-58
func (c *Calculator) RenameTreeSpec(specID int, title string) {
	if specID >= 1 && specID <= len(c.build.Tree.Specs) {
		c.build.Tree.Specs[specID-1].Title = &title
	}
}

// DeleteTreeSpec is the tree list's Delete; the last spec is never deleted.
//
//pob2:src/Classes/PassiveSpecListControl.lua:100-116
func (c *Calculator) DeleteTreeSpec(specID int) {
	specs := c.build.Tree.Specs
	if len(specs) <= 1 || specID < 1 || specID > len(specs) {
		return
	}
	active := c.activeSpecIndex()
	c.build.Tree.Specs = slices.Delete(specs, specID-1, specID)
	switch {
	case specID == active:
		c.SetActiveTreeSpec(max(1, specID-1))
	case specID < active:
		c.build.Tree.ActiveSpec = active - 1
	}
}

// MoveTreeSpec is a drag in the tree list: the spec at from (1-based) ends up at to.
//
//pob2:block:src/Classes/ListControl.lua:440-448
//pob2:src/Classes/PassiveSpecListControl.lua:87-92
func (c *Calculator) MoveTreeSpec(from, to int) {
	specs := c.build.Tree.Specs
	if from == to || from < 1 || to < 1 || from > len(specs) || to > len(specs) {
		return
	}
	active := c.activeSpecIndex()
	order := make([]int, len(specs))
	for i := range order {
		order[i] = i + 1
	}
	order = slices.Insert(slices.Delete(order, from-1, from), to-1, from)
	moved := make([]pob2.Spec, len(specs))
	for i, was := range order {
		moved[i] = specs[was-1]
		if was == active {
			c.build.Tree.ActiveSpec = i + 1
		}
	}
	c.build.Tree.Specs = moved
}
