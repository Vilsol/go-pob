package exposition

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/pob2"
)

type treeNodeJSON struct {
	ID                int                `json:"id"`
	X                 float64            `json:"x"`
	Y                 float64            `json:"y"`
	Angle             float64            `json:"angle"`
	Group             int                `json:"group"`
	Orbit             int                `json:"orbit"`
	Type              string             `json:"type"`
	Name              string             `json:"name"`
	Stats             []string           `json:"stats"`
	Ascendancy        string             `json:"ascendancy,omitempty"`
	IsAttribute       bool               `json:"isAttribute,omitempty"`
	IsAscendancyStart bool               `json:"isAscendancyStart,omitempty"`
	ContainJewel      bool               `json:"containJewelSocket,omitempty"`
	ClassesStart      []string           `json:"classesStart,omitempty"`
	Connections       []tree2.Connection `json:"connections"`
}

type treeGroupJSON struct {
	ID int     `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

type treeJSON struct {
	Nodes      []treeNodeJSON  `json:"nodes"`
	Groups     []treeGroupJSON `json:"groups"`
	OrbitRadii []float64       `json:"orbitRadii"`
	MinX       float64         `json:"minX"`
	MinY       float64         `json:"minY"`
	MaxX       float64         `json:"maxX"`
	MaxY       float64         `json:"maxY"`
}

// TreeClasses is the tree's class names in class id order (PassiveTree.lua:104 keys them by integerId).
func TreeClasses() ([]string, error) {
	t, err := loadTree()
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(t.Classes))
	for id := range t.Classes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = t.Classes[id].Name
	}
	return out, nil
}

// PassiveTreeJSON is the tree the view draws, as one JSON document: the nodes a spec holds (PassiveSpec:Init's
// filter) with their positions, names, stat lines and connections, the group centres and the orbit radii.
func PassiveTreeJSON() (string, error) {
	t, err := loadTree()
	if err != nil {
		return "", err
	}
	spec := tree2.NewPassiveSpec(t)
	out := treeJSON{OrbitRadii: t.OrbitRadii, MinX: t.MinX, MinY: t.MinY, MaxX: t.MaxX, MaxY: t.MaxY}
	ids := make([]int, 0, len(spec.Nodes))
	for id := range spec.Nodes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	groups := map[int]bool{}
	for _, id := range ids {
		n := spec.Nodes[id].Node
		out.Nodes = append(out.Nodes, treeNodeJSON{
			ID: n.ID, X: n.X, Y: n.Y, Angle: n.Angle, Group: n.GroupID, Orbit: n.Orbit, Type: n.Type, Name: n.Dn, Stats: n.Sd,
			Ascendancy: n.AscendancyName, IsAttribute: n.IsAttribute, IsAscendancyStart: n.IsAscendancyStart, ContainJewel: n.ContainJewelSocket,
			ClassesStart: n.ClassesStart, Connections: n.Connections,
		})
		if !groups[n.GroupID] {
			groups[n.GroupID] = true
			out.Groups = append(out.Groups, treeGroupJSON{ID: n.GroupID, X: n.Group.X, Y: n.Group.Y})
		}
	}
	b, err := json.Marshal(out)
	return string(b), err
}

// TreeNodeText is a spec node's name and stat lines where they differ from the tree's (a switched,
// attribute or conquered node).
type TreeNodeText struct {
	ID    int
	Name  string
	Stats []string
}

// TreeState is the spec as the view reads it after the last calculation.
type TreeState struct {
	Alloc []int
	// AllocMode lists the allocated nodes on a weapon set, as id, set pairs.
	AllocMode []int
	Text      []TreeNodeText
	ClassName string
	// AscendClassBaseName is spec.curAscendClassBaseName, which the view greys other ascendancies against.
	AscendClassBaseName string
	Used                int
	AscUsed             int
	AttributeIndex      int
}

func (c *Calculator) spec() (*tree2.PassiveSpec, error) {
	if c.env == nil || c.env.Spec == nil {
		return nil, notCalculatedError{}
	}
	return c.env.Spec, nil
}

func (c *Calculator) TreeState() (*TreeState, error) {
	spec, err := c.spec()
	if err != nil {
		return nil, err
	}
	used, ascUsed, _, _, _, _ := spec.CountAllocNodes()
	st := &TreeState{
		ClassName: spec.CurClassName, AscendClassBaseName: spec.CurAscendClassBaseName,
		Used: used, AscUsed: ascUsed, AttributeIndex: spec.AttributeIndex,
	}
	ids := make([]int, 0, len(spec.Nodes))
	for id := range spec.Nodes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		n := spec.Nodes[id]
		if n.Alloc {
			st.Alloc = append(st.Alloc, id)
			if n.AllocMode != 0 {
				st.AllocMode = append(st.AllocMode, id, n.AllocMode)
			}
		}
		if n.Dn != n.Node.Dn || !slices.Equal(n.Sd, n.Node.Sd) {
			st.Text = append(st.Text, TreeNodeText{ID: id, Name: n.Dn, Stats: n.Sd})
		}
	}
	return st, nil
}

// TreeHover is what the view highlights over a node: the path an allocation would take and the nodes that
// depend on it.
type TreeHover struct {
	Path    []int
	Depends []int
}

func nodeIDs(list []*tree2.SpecNode) []int {
	out := make([]int, 0, len(list))
	for _, n := range list {
		out = append(out, n.ID)
	}
	return out
}

// HoverNode takes no trace path.
//
//pob2:block:src/Classes/PassiveTreeView.lua:356-373
func (c *Calculator) HoverNode(id int) TreeHover {
	spec, err := c.spec()
	if err != nil {
		return TreeHover{}
	}
	node, ok := spec.Nodes[id]
	if !ok || !node.HasPath {
		return TreeHover{}
	}
	var hover TreeHover
	if len(node.IntuitiveLeapLikesAffecting) == 0 {
		path := node.Path
		if !node.Alloc {
			path, _ = spec.GetEffectiveAllocationPath(node, nil)
		}
		hover.Path = nodeIDs(path)
	}
	hover.Depends = nodeIDs(node.Depends)
	return hover
}

type unknownNodeError int

func (e unknownNodeError) Error() string { return fmt.Sprintf("node %d is not on the tree", int(e)) }

// ClickNode clicks a node (right for a right click; hotkey is the attribute index a held key picks, 0 for
// none) and saves the spec into the build when the click changed it. It returns the tree2.ClickResult.
func (c *Calculator) ClickNode(id int, right bool, hotkey int, confirmClassChange bool) (int, error) {
	spec, err := c.spec()
	if err != nil {
		return 0, err
	}
	node, ok := spec.Nodes[id]
	if !ok {
		return 0, unknownNodeError(id)
	}
	result := tree2.ClickChanged
	if right {
		err = spec.RightClickNode(node, hotkey)
	} else {
		result, err = spec.LeftClickNode(node, hotkey, confirmClassChange)
	}
	if result == tree2.ClickChanged {
		c.saveSpec(spec)
	}
	return int(result), err
}

// AllocateAttributeNode allocates an attribute node as the chosen attribute (1 Strength, 2 Dexterity,
// 3 Intelligence).
func (c *Calculator) AllocateAttributeNode(id int, attributeIndex int) error {
	spec, err := c.spec()
	if err != nil {
		return err
	}
	node, ok := spec.Nodes[id]
	if !ok {
		return unknownNodeError(id)
	}
	err = spec.AllocateAttributeNode(node, attributeIndex)
	c.saveSpec(spec)
	return err
}

// saveSpec writes the spec back into the active <Spec>, as TreeTab:Save would on the next save.
func (c *Calculator) saveSpec(spec *tree2.PassiveSpec) {
	specs := c.build.Tree.Specs
	idx := c.build.Tree.ActiveSpec
	if idx < 1 || idx > len(specs) {
		idx = 1
	}
	s := &specs[idx-1]
	save := spec.Save()
	s.ClassID, s.AscendClassID, s.ClassInternalID = &save.ClassID, &save.AscendClassID, &save.ClassInternalID
	s.AscendancyInternalID, s.Nodes, s.MasteryEffects = &save.AscendancyInternalID, &save.Nodes, &save.MasteryEffects
	weaponSet := func(set int) *pob2.WeaponSet {
		if nodes, ok := save.WeaponSets[set]; ok {
			return &pob2.WeaponSet{Nodes: nodes}
		}
		return nil
	}
	s.WeaponSet1, s.WeaponSet2 = weaponSet(1), weaponSet(2)
	if s.Overrides == nil {
		s.Overrides = &pob2.Overrides{}
	}
	s.Overrides.AttributeOverrides = []pob2.AttributeOverride{{StrNodes: save.StrNodes, DexNodes: save.DexNodes, IntNodes: save.IntNodes}}
}
