package exposition

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Vilsol/go-pob/calculator2"
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
	// NodesInRadius is a radius socket's node ids in each jewel radius, by radius index - 1.
	NodesInRadius [][]int `json:"nodesInRadius,omitempty"`
}

type jewelRadiusJSON struct {
	Inner float64 `json:"inner"`
	Outer float64 `json:"outer"`
	Col   string  `json:"col"`
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
	// JewelRadius is data.jewelRadius with PassiveTreeJewelDistanceMultiplier applied, as
	// PassiveTreeView.lua:1321-1323 draws it; Col is "#rrggbb".
	JewelRadius []jewelRadiusJSON `json:"jewelRadius"`
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
	for _, r := range t.JewelRadius {
		out.JewelRadius = append(out.JewelRadius, jewelRadiusJSON{
			Inner: r.Inner * tree2.PassiveTreeJewelDistanceMultiplier, Outer: r.Outer * tree2.PassiveTreeJewelDistanceMultiplier,
			Col: "#" + strings.ToLower(strings.TrimPrefix(r.Col, "^x")),
		})
	}
	ids := make([]int, 0, len(spec.Nodes))
	for id := range spec.Nodes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	groups := map[int]bool{}
	for _, id := range ids {
		n := spec.Nodes[id].Node
		node := treeNodeJSON{
			ID: n.ID, X: n.X, Y: n.Y, Angle: n.Angle, Group: n.GroupID, Orbit: n.Orbit, Type: n.Type, Name: n.Dn, Stats: n.Sd,
			Ascendancy: n.AscendancyName, IsAttribute: n.IsAttribute, IsAscendancyStart: n.IsAscendancyStart, ContainJewel: n.ContainJewelSocket,
			ClassesStart: n.ClassesStart, Connections: n.Connections,
		}
		if n.Type == "Socket" && n.NodesInRadius != nil {
			node.NodesInRadius = make([][]int, len(n.NodesInRadius))
			for i, in := range n.NodesInRadius {
				node.NodesInRadius[i] = make([]int, 0, len(in))
				for nodeID := range in {
					node.NodesInRadius[i] = append(node.NodesInRadius[i], nodeID)
				}
				slices.Sort(node.NodesInRadius[i])
			}
		}
		out.Nodes = append(out.Nodes, node)
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
	// JewelRadiusIndex lists the sockets whose jewel has a radius, as socket id, jewelRadiusIndex pairs.
	JewelRadiusIndex []int
	LevelAutoMode    bool
	Progress         calculator2.PlayerProgress
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
		LevelAutoMode: c.build.Build.CharacterLevelAutoMode, Progress: calculator2.EstimatePlayerProgress(spec, c.env.Player.Output),
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
	//pob2:block:src/Classes/PassiveTreeView.lua:1315
	//pob2:block:src/Classes/PassiveTreeView.lua:1341
	if spec.Items != nil {
		sockets := make([]int, 0, len(spec.Jewels))
		for id := range spec.Jewels {
			sockets = append(sockets, id)
		}
		slices.Sort(sockets)
		for _, id := range sockets {
			if item := spec.Items(spec.Jewels[id]); item != nil && item.JewelRadiusIndex != 0 {
				st.JewelRadiusIndex = append(st.JewelRadiusIndex, id, item.JewelRadiusIndex)
			}
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

// HoverNode takes no trace path; allocMode is spec.allocMode, the weapon set Alt+wheel picked.
//
//pob2:block:src/Classes/PassiveTreeView.lua:356-373
func (c *Calculator) HoverNode(id int, allocMode int) TreeHover {
	spec, err := c.spec()
	if err != nil {
		return TreeHover{}
	}
	spec.AllocMode = allocMode
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
// none; allocMode is spec.allocMode) and, when the click changed the spec, adds an undo state and saves the
// spec into the build. It returns the tree2.ClickResult.
//
//pob2:block:src/Classes/PassiveTreeView.lua:443
//pob2:block:src/Classes/PassiveTreeView.lua:547
//pob2:block:src/Classes/PassiveTreeView.lua:591
func (c *Calculator) ClickNode(id int, right bool, hotkey int, confirmClassChange bool, allocMode int) (int, error) {
	spec, err := c.spec()
	if err != nil {
		return 0, err
	}
	node, ok := spec.Nodes[id]
	if !ok {
		return 0, unknownNodeError(id)
	}
	spec.AllocMode = allocMode
	var result tree2.ClickResult
	if right {
		result, err = spec.RightClickNode(node, hotkey)
	} else {
		result, err = spec.LeftClickNode(node, hotkey, confirmClassChange)
	}
	if result == tree2.ClickChanged {
		c.undo.AddUndoState(spec, false)
		c.saveSpec(spec)
	}
	return int(result), err
}

// AllocateAttributeNode allocates an attribute node as the chosen attribute (1 Strength, 2 Dexterity,
// 3 Intelligence), on the weapon set allocMode names.
//
//pob2:block:src/Classes/TreeTab.lua:857-862
func (c *Calculator) AllocateAttributeNode(id int, attributeIndex int, allocMode int) error {
	spec, err := c.spec()
	if err != nil {
		return err
	}
	node, ok := spec.Nodes[id]
	if !ok {
		return unknownNodeError(id)
	}
	spec.AllocMode = allocMode
	err = spec.AllocateAttributeNode(node, attributeIndex)
	c.undo.AddUndoState(spec, false)
	c.saveSpec(spec)
	return err
}

// Undo is spec:Undo on Ctrl+Z; it reports whether a state was restored, and saves the spec when one was.
//
//pob2:block:src/Classes/TreeTab.lua:357-360
func (c *Calculator) Undo() (bool, error) {
	spec, err := c.spec()
	if err != nil {
		return false, err
	}
	if !c.undo.Undo(spec) {
		return false, nil
	}
	c.saveSpec(spec)
	return true, nil
}

// Redo is spec:Redo on Ctrl+Y.
//
//pob2:block:src/Classes/TreeTab.lua:361-364
func (c *Calculator) Redo() (bool, error) {
	spec, err := c.spec()
	if err != nil {
		return false, err
	}
	if !c.undo.Redo(spec) {
		return false, nil
	}
	c.saveSpec(spec)
	return true, nil
}

// ResetTree is the Reset Tree popup's Reset: attribute nodes go back to "Attribute" and every node but the class
// starts is deallocated.
//
//pob2:block:src/Classes/TreeTab.lua:149-156
func (c *Calculator) ResetTree() error {
	spec, err := c.spec()
	if err != nil {
		return err
	}
	clear(spec.HashOverrides)
	spec.ResetNodes()
	spec.BuildAllDependsAndPaths()
	c.undo.AddUndoState(spec, false)
	c.saveSpec(spec)
	return nil
}

// saveSpec writes the spec back into the active <Spec>, as TreeTab:Save would on the next save.
func (c *Calculator) saveSpec(spec *tree2.PassiveSpec) {
	specs := c.build.Tree.Specs
	idx := c.build.Tree.ActiveSpec
	if idx < 1 || idx > len(specs) {
		idx = 1
	}
	writeSpec(&specs[idx-1], spec)
}

// writeSpec writes the spec's class, nodes, weapon sets and attribute overrides into s.
func writeSpec(s *pob2.Spec, spec *tree2.PassiveSpec) {
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
