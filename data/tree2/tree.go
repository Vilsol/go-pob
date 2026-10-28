// Package tree2 loads the Path of Exile 2 passive tree and a build's allocation on it.
// It holds no global state and imports no PoE1 package.
package tree2

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
)

type Connection struct {
	ID    int `json:"id"`
	Orbit int `json:"orbit"`
}

type UnlockConstraint struct {
	Ascendancy string `json:"ascendancy"`
	Nodes      []int  `json:"nodes"`
}

// AttributeOption is one entry of an attribute node's options list (Strength, Dexterity, Intelligence).
type AttributeOption struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Stats []string `json:"stats"`
	Icon  string   `json:"icon"`
	Dn    string
	Sd    []string
}

// SwitchOption is the per-ascendancy replacement of a switchable node (Lich -> Abyssal Lich).
type SwitchOption struct {
	ID             int      `json:"id"`
	Name           string   `json:"name"`
	Stats          []string `json:"stats"`
	AscendancyName string   `json:"ascendancyName"`
	Dn             string
	Sd             []string
}

type Node struct {
	Skill                  int               `json:"skill"`
	StringID               string            `json:"stringId"`
	GroupRef               *int              `json:"group"`
	Orbit                  int               `json:"orbit"`
	OrbitIndex             int               `json:"orbitIndex"`
	Name                   string            `json:"name"`
	Stats                  []string          `json:"stats"`
	Connections            []Connection      `json:"connections"`
	AscendancyName         string            `json:"ascendancyName"`
	ClassesStart           []string          `json:"classesStart"`
	IsAscendancyStart      bool              `json:"isAscendancyStart"`
	IsSwitchable           bool              `json:"isSwitchable"`
	IsAttribute            bool              `json:"isAttribute"`
	IsOnlyImage            bool              `json:"isOnlyImage"`
	IsJewelSocket          bool              `json:"isJewelSocket"`
	IsKeystone             bool              `json:"isKeystone"`
	IsNotable              bool              `json:"isNotable"`
	IsFreeAllocate         bool              `json:"isFreeAllocate"`
	IsMultipleChoice       bool              `json:"isMultipleChoice"`
	IsMultipleChoiceOption bool              `json:"isMultipleChoiceOption"`
	IsProxy                bool              `json:"isProxy"`
	ContainJewelSocket     bool              `json:"containJewelSocket"`
	NoRadius               bool              `json:"noRadius"`
	ApplyToArmour          bool              `json:"applyToArmour"`
	Sinister               bool              `json:"sinister"`
	AliasPassiveSocket     string            `json:"aliasPassiveSocket"`
	UnlockConstraint       *UnlockConstraint `json:"unlockConstraint"`
	RawOptions             json.RawMessage   `json:"options"`

	ID       int
	GroupID  int
	Dn       string
	Sd       []string
	Type     string
	Group    *Group
	LinkedID []int
	Angle    float64
	X, Y     float64
	// HasPosition is false for a groupless node, whose x/y PoB leaves nil.
	HasPosition bool

	// NodesInRadius is indexed by jewel radius index - 1; set on radius sockets and keystones.
	NodesInRadius []map[int]*Node

	// AttributeOptions is set on isAttribute nodes, SwitchOptions on isSwitchable ones.
	AttributeOptions []AttributeOption
	SwitchOptions    map[string]SwitchOption

	// LegionID is a tree.legion node's string id; tree nodes leave it empty.
	LegionID string
}

type Group struct {
	X                 float64 `json:"x"`
	Y                 float64 `json:"y"`
	Orbits            []int   `json:"orbits"`
	Nodes             []int   `json:"nodes"`
	IsProxy           bool    `json:"isProxy"`
	Oo                map[int]bool
	AscendancyName    string
	IsAscendancyStart bool
}

type Ascendancy struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	InternalID string `json:"internalId"`
	Replace    string `json:"replace"`
	ReplaceBy  string `json:"replaceBy"`
	// StartNodeID is 0 when no node names this ascendancy as its start.
	StartNodeID int
}

type Class struct {
	Name        string        `json:"name"`
	IntegerID   int           `json:"integerId"`
	BaseStr     int           `json:"base_str"`
	BaseDex     int           `json:"base_dex"`
	BaseInt     int           `json:"base_int"`
	Ascendancy  []*Ascendancy `json:"ascendancies"`
	StartNodeID int
	// Classes is PoB's class.classes: index 0 is the "None" ascendancy, 1.. are Ascendancy in order.
	Classes []*Ascendancy
}

type Constants struct {
	Classes              map[string]int `json:"classes"`
	CharacterAttributes  map[string]int `json:"characterAttributes"`
	SkillsPerOrbit       []int          `json:"skillsPerOrbit"`
	OrbitRadii           []float64      `json:"orbitRadii"`
	OrbitAnglesByOrbit   [][]float64    `json:"orbitAnglesByOrbit"`
	PSSCentreInnerRadius float64        `json:"PSSCentreInnerRadius"`
}

type AscendRef struct {
	ClassID       int
	Class         *Class
	AscendClassID int
	AscendClass   *Ascendancy
}

type PassiveTree struct {
	TreeVersion string
	Tree        string
	Classes     map[int]*Class
	// Groups is keyed by PoB's 1-based group index; tree.json's null holes are absent.
	Groups     map[int]*Group
	Nodes      map[int]*Node
	JewelSlots []int
	Constants  Constants
	MinX, MinY float64
	MaxX, MaxY float64
	Size       float64

	ClassNameMap          map[string]int
	AscendNameMap         map[string]AscendRef
	ClassIntegerIDMap     map[int]int
	InternalAscendNameMap map[string]AscendRef
	ClassStartNodeNameMap map[string]int
	ClassNotables         map[string][]string
	AscendancyMap         map[string]*Node
	KeystoneMap           map[string]*Node
	NotableMap            map[string]*Node
	ClusterNodeMap        map[string]*Node
	Sockets               map[int]*Node

	SkillsPerOrbit     []int
	OrbitRadii         []float64
	OrbitAnglesByOrbit [][]float64

	// JewelRadius and MaxJewelRadius are data.jewelRadius / data.maxJewelRadius as main:LoadTree sets them.
	JewelRadius    []JewelRadius
	MaxJewelRadius float64

	// MissingConnections lists connection targets absent from the node table, which PoB prints and skips.
	MissingConnections []int

	// Legion is tree.legion.nodes in PoB's order: Legion[i] is the Lua's legion.nodes[i+1].
	Legion []*Node
}

type treeJSON struct {
	Classes    []*Class         `json:"classes"`
	Constants  Constants        `json:"constants"`
	Groups     []*Group         `json:"groups"`
	JewelSlots []int            `json:"jewelSlots"`
	Nodes      map[string]*Node `json:"nodes"`
	Tree       string           `json:"tree"`
	MinX       float64          `json:"min_x"`
	MinY       float64          `json:"min_y"`
	MaxX       float64          `json:"max_x"`
	MaxY       float64          `json:"max_y"`
}

type JewelRadius struct {
	Inner, Outer               float64
	Label                      string
	InnerSquared, OuterSquared float64
}

// jewelRadii is data.jewelRadii; every 0_x tree up to 0_5 resolves to the 0_1 entry.
//
//pob2:block:src/Modules/Data.lua:668-684
var jewelRadii = map[string][]JewelRadius{
	"0_1": {
		{Inner: 0, Outer: 1000, Label: "Small"},
		{Inner: 0, Outer: 1150, Label: "Medium"},
		{Inner: 0, Outer: 1300, Label: "Large"},
		{Inner: 0, Outer: 1500, Label: "Very Large"},

		{Inner: 650, Outer: 950, Label: "Variable"},
		{Inner: 800, Outer: 1100, Label: "Variable"},
		{Inner: 950, Outer: 1250, Label: "Variable"},
		{Inner: 1100, Outer: 1400, Label: "Variable"},
		{Inner: 1250, Outer: 1550, Label: "Variable"},
		{Inner: 1400, Outer: 1700, Label: "Variable"},
		{Inner: 1650, Outer: 1950, Label: "Variable"},
		{Inner: 1800, Outer: 2100, Label: "Variable"},
	},
}

//pob2:block:src/Data/Misc.lua:37
const passiveTreeJewelDistanceMultiplier = 1.2

func parseTreeVersion(v string) (int, int, bool) {
	major, minor, ok := strings.Cut(v, "_")
	if !ok {
		return 0, 0, false
	}
	a, err1 := strconv.Atoi(major)
	b, err2 := strconv.Atoi(minor)
	return a, b, err1 == nil && err2 == nil
}

// setJewelRadii returns the radius list setJewelRadiiGlobally would install for treeVersion, and maxJewelRadius.
//
//pob2:src/Modules/Data.lua:638-666
func setJewelRadii(treeVersion string) ([]JewelRadius, float64) {
	tMajor, tMinor, ok := parseTreeVersion(treeVersion)
	if !ok {
		return nil, 0
	}
	found := false
	var sMajor, sMinor int
	for version := range jewelRadii {
		jMajor, jMinor, _ := parseTreeVersion(version)
		if jMajor < tMajor || (jMajor == tMajor && jMinor <= tMinor) {
			if !found || jMajor > sMajor || (jMajor == sMajor && jMinor > sMinor) {
				sMajor, sMinor, found = jMajor, jMinor, true
			}
		}
	}
	if !found {
		return nil, 0
	}
	src := jewelRadii[strconv.Itoa(sMajor)+"_"+strconv.Itoa(sMinor)]
	radii := make([]JewelRadius, len(src))
	maxJewelRadius := 0.0
	for i, radiusInfo := range src {
		radiusInfo.OuterSquared = float64(radiusInfo.Outer*radiusInfo.Outer*passiveTreeJewelDistanceMultiplier) * passiveTreeJewelDistanceMultiplier
		radiusInfo.InnerSquared = float64(radiusInfo.Inner*radiusInfo.Inner*passiveTreeJewelDistanceMultiplier) * passiveTreeJewelDistanceMultiplier
		if radiusInfo.Outer > maxJewelRadius {
			maxJewelRadius = radiusInfo.Outer * passiveTreeJewelDistanceMultiplier
		}
		radii[i] = radiusInfo
	}
	return radii, maxJewelRadius
}

// sortedNodeIDs stands in for Lua's pairs() over the node table; PoB's order is unspecified.
func (t *PassiveTree) sortedNodeIDs() []int {
	ids := make([]int, 0, len(t.Nodes))
	for id := range t.Nodes {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// Load builds a PassiveTree from a tree.json in PoB-PoE2's TreeData/<version> layout. Art, connectors and
// jewel radius lists are not built, and stat lines are not parsed.
//
//pob2:src/Classes/PassiveTree.lua:59-302
func Load(treeVersion string, raw []byte) (*PassiveTree, error) {
	var tj treeJSON
	if err := json.Unmarshal(raw, &tj); err != nil {
		return nil, fmt.Errorf("tree2: decoding tree %s: %w", treeVersion, err)
	}
	t := &PassiveTree{
		TreeVersion: treeVersion,
		Tree:        tj.Tree,
		JewelSlots:  tj.JewelSlots,
		Constants:   tj.Constants,
		MinX:        tj.MinX, MinY: tj.MinY, MaxX: tj.MaxX, MaxY: tj.MaxY,
		Groups: make(map[int]*Group, len(tj.Groups)),
		Nodes:  make(map[int]*Node, len(tj.Nodes)),
	}
	for i, g := range tj.Groups {
		if g != nil {
			t.Groups[i+1] = g
		}
	}
	for k, n := range tj.Nodes {
		if k == "root" {
			continue
		}
		id, err := strconv.Atoi(k)
		if err != nil {
			return nil, fmt.Errorf("tree2: node key %q is not a number", k)
		}
		t.Nodes[id] = n
	}

	t.Size = math.Min(t.MaxX-t.MinX, t.MaxY-t.MinY) * 1.1

	t.Classes = make(map[int]*Class, len(tj.Classes))
	for _, class := range tj.Classes {
		t.Classes[class.IntegerID] = class
	}

	t.ClassNameMap = map[string]int{}
	t.AscendNameMap = map[string]AscendRef{}
	t.ClassIntegerIDMap = map[int]int{}
	t.InternalAscendNameMap = map[string]AscendRef{}
	t.ClassStartNodeNameMap = map[string]int{}
	t.ClassNotables = map[string][]string{}

	classIDs := make([]int, 0, len(t.Classes))
	for id := range t.Classes {
		classIDs = append(classIDs, id)
	}
	sort.Ints(classIDs)
	for _, classID := range classIDs {
		class := t.Classes[classID]
		class.Classes = append([]*Ascendancy{{Name: "None"}}, class.Ascendancy...)
		t.ClassNameMap[class.Name] = classID
		t.ClassIntegerIDMap[class.IntegerID] = classID
		for ascendClassID, ascendClass := range class.Classes {
			ref := AscendRef{ClassID: classID, Class: class, AscendClassID: ascendClassID, AscendClass: ascendClass}
			key := ascendClass.ID
			if key == "" {
				key = ascendClass.Name
			}
			t.AscendNameMap[key] = ref
			if ascendClass.InternalID != "" {
				t.InternalAscendNameMap[ascendClass.InternalID] = ref
			}
		}
	}

	t.SkillsPerOrbit = t.Constants.SkillsPerOrbit
	t.OrbitRadii = t.Constants.OrbitRadii
	t.OrbitAnglesByOrbit = t.Constants.OrbitAnglesByOrbit

	for _, group := range t.Groups {
		group.Oo = make(map[int]bool, len(group.Orbits))
		for _, orbit := range group.Orbits {
			group.Oo[orbit] = true
		}
	}

	t.AscendancyMap = map[string]*Node{}
	t.KeystoneMap = map[string]*Node{}
	t.NotableMap = map[string]*Node{}
	t.ClusterNodeMap = map[string]*Node{}
	t.Sockets = map[int]*Node{}

	ids := t.sortedNodeIDs()
	for _, id := range ids {
		node := t.Nodes[id]
		node.ID = node.Skill
		if node.GroupRef != nil {
			node.GroupID = *node.GroupRef
		}
		node.Dn = node.Name
		node.Sd = node.Stats
		if node.Sd == nil {
			node.Sd = []string{}
		}

		switch {
		case node.ClassesStart != nil:
			node.Type = "ClassStart"
			for _, className := range node.ClassesStart {
				t.ClassStartNodeNameMap[className] = node.ID
				if classID, ok := t.ClassNameMap[className]; ok {
					t.Classes[classID].StartNodeID = node.ID
				}
			}
		case node.IsAscendancyStart:
			node.Type = "AscendClassStart"
			ref, ok := t.AscendNameMap[node.AscendancyName]
			if !ok {
				return nil, fmt.Errorf("tree2: ascendancy start %d names unknown ascendancy %q", node.ID, node.AscendancyName)
			}
			ref.AscendClass.StartNodeID = node.ID
			if node.IsSwitchable {
				var opts map[string]json.RawMessage
				if err := json.Unmarshal(node.RawOptions, &opts); err != nil {
					return nil, fmt.Errorf("tree2: node %d switch options: %w", node.ID, err)
				}
				for ascName := range opts {
					if o, ok := t.AscendNameMap[ascName]; ok {
						o.AscendClass.StartNodeID = node.ID
					}
				}
			}
		case node.IsOnlyImage:
			node.Type = "OnlyImage"
		case node.IsJewelSocket:
			node.Type = "Socket"
			t.Sockets[node.ID] = node
		case node.IsKeystone:
			node.Type = "Keystone"
			t.KeystoneMap[node.Dn] = node
			t.KeystoneMap[strings.ToLower(node.Dn)] = node
		case node.IsNotable:
			node.Type = "Notable"
			if node.AscendancyName == "" {
				//pob2:block:src/Classes/PassiveTree.lua:257-264
				lower := strings.ToLower(node.Dn)
				if _, ok := t.NotableMap[lower]; !ok || node.GroupRef != nil {
					t.NotableMap[lower] = node
				}
			} else {
				if node.ContainJewelSocket {
					t.Sockets[node.ID] = node
				}
				t.AscendancyMap[strings.ToLower(node.Dn)] = node
				className := t.AscendNameMap[node.AscendancyName].Class.Name
				if className != "Scion" {
					t.ClassNotables[className] = append(t.ClassNotables[className], node.Dn)
				}
			}
		default:
			node.Type = "Normal"
			if node.AscendancyName == "Ascendant" && !strings.Contains(node.Dn, "Dexterity") && !strings.Contains(node.Dn, "Intelligence") &&
				!strings.Contains(node.Dn, "Strength") && !strings.Contains(node.Dn, "Passive") {
				t.AscendancyMap[strings.ToLower(node.Dn)] = node
				className := t.AscendNameMap[node.AscendancyName].Class.Name
				t.ClassNotables[className] = append(t.ClassNotables[className], node.Dn)
			}
		}

		if group, ok := t.Groups[node.GroupID]; ok {
			node.Group = group
			group.AscendancyName = node.AscendancyName
			if node.IsAscendancyStart {
				group.IsAscendancyStart = true
			}
		} else if node.Type == "Notable" || node.Type == "Keystone" {
			t.ClusterNodeMap[node.Dn] = node
		}

		if err := t.processNode(node); err != nil {
			return nil, err
		}
	}

	//pob2:block:src/Classes/PassiveTree.lua:305-333
	for _, id := range ids {
		node := t.Nodes[id]
		for _, connection := range node.Connections {
			otherID := connection.ID
			other, ok := t.Nodes[otherID]
			if !ok {
				t.MissingConnections = append(t.MissingConnections, otherID)
				continue
			}
			if node.Type == "OnlyImage" || other.Type == "OnlyImage" {
				continue
			}
			if node.ID == otherID {
				continue
			}
			other.LinkedID = append(other.LinkedID, node.ID)
			node.LinkedID = append(node.LinkedID, otherID)
		}
	}

	t.JewelRadius, t.MaxJewelRadius = setJewelRadii(treeVersion)
	t.buildNodesInRadius(ids)
	t.loadLegion()
	return t, nil
}

// buildNodesInRadius leaves out the ConnectedTo<Class>Start flags and the legion nodes; both are mod lists.
//
//pob2:block:src/Classes/PassiveTree.lua:369-426
func (t *PassiveTree) buildNodesInRadius(ids []int) {
	inBox := func(node, centre *Node) bool {
		return node.HasPosition && node.X >= centre.X-t.MaxJewelRadius && node.X <= centre.X+t.MaxJewelRadius &&
			node.Y >= centre.Y-t.MaxJewelRadius && node.Y <= centre.Y+t.MaxJewelRadius
	}
	fill := func(centre, node *Node) {
		vX, vY := node.X-centre.X, node.Y-centre.Y
		distSquared := float64(vX*vX) + float64(vY*vY)
		for radiusIndex, radiusInfo := range t.JewelRadius {
			if distSquared <= radiusInfo.OuterSquared && radiusInfo.InnerSquared <= distSquared {
				centre.NodesInRadius[radiusIndex][node.ID] = node
			}
		}
	}
	newRadius := func() []map[int]*Node {
		r := make([]map[int]*Node, len(t.JewelRadius))
		for i := range r {
			r[i] = map[int]*Node{}
		}
		return r
	}
	for _, socketID := range sortedKeys(t.Sockets) {
		socket := t.Sockets[socketID]
		if socket.Name == "Charm Socket" || socket.ContainJewelSocket || socket.NoRadius {
			continue
		}
		socket.NodesInRadius = newRadius()
		for _, id := range ids {
			node := t.Nodes[id]
			// isBlighted and isMastery are absent from tree 0_5.
			if inBox(node, socket) && node != socket && node.Group != nil && !node.IsProxy && !node.Group.IsProxy {
				fill(socket, node)
			}
		}
	}
	for _, name := range sortedStringKeys(t.KeystoneMap) {
		keystone := t.KeystoneMap[name]
		if keystone.NodesInRadius != nil {
			continue
		}
		keystone.NodesInRadius = newRadius()
		if !keystone.HasPosition {
			continue
		}
		for _, id := range ids {
			node := t.Nodes[id]
			// PoB tests node.OnlyImage and node.isSocket, fields no node carries, so neither excludes anything.
			if inBox(node, keystone) && node != keystone && node.Group != nil && !node.IsProxy && !node.Group.IsProxy {
				fill(keystone, node)
			}
		}
	}
}

func sortedKeys[V any](m map[int]V) []int {
	ids := make([]int, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func sortedStringKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// processNode keeps the position and switch/attribute option parts of ProcessNode; ProcessStats
// (the mod parse) belongs to the calc port.
//
//pob2:src/Classes/PassiveTree.lua:547-614
func (t *PassiveTree) processNode(node *Node) error {
	if node.Group != nil {
		//pob2:block:src/Classes/PassiveTree.lua:571-576
		if node.Orbit < 0 || node.Orbit >= len(t.OrbitAnglesByOrbit) || node.OrbitIndex < 0 || node.OrbitIndex >= len(t.OrbitAnglesByOrbit[node.Orbit]) {
			return fmt.Errorf("tree2: node %d sits at orbit %d index %d outside the tree's orbit table", node.ID, node.Orbit, node.OrbitIndex)
		}
		node.Angle = t.OrbitAnglesByOrbit[node.Orbit][node.OrbitIndex]
		orbitRadius := t.OrbitRadii[node.Orbit]
		node.X = node.Group.X + float64(math.Sin(node.Angle)*orbitRadius)
		node.Y = node.Group.Y - float64(math.Cos(node.Angle)*orbitRadius)
		node.HasPosition = true
	}

	// An option inherits every field it lacks from its node (setmetatable __index = node).
	//pob2:block:src/Classes/PassiveTree.lua:592-613
	if node.IsAttribute {
		if err := json.Unmarshal(node.RawOptions, &node.AttributeOptions); err != nil {
			return fmt.Errorf("tree2: attribute node %d options: %w", node.ID, err)
		}
		for i := range node.AttributeOptions {
			o := &node.AttributeOptions[i]
			o.ID = node.ID
			o.Dn, o.Sd = inherit(o.Name, o.Stats, node)
		}
	} else if node.IsSwitchable {
		if err := json.Unmarshal(node.RawOptions, &node.SwitchOptions); err != nil {
			return fmt.Errorf("tree2: switchable node %d options: %w", node.ID, err)
		}
		for k, o := range node.SwitchOptions {
			o.Dn, o.Sd = inherit(o.Name, o.Stats, node)
			if o.ID == 0 {
				o.ID = node.ID
			}
			node.SwitchOptions[k] = o
		}
	}
	return nil
}

func inherit(name string, stats []string, node *Node) (string, []string) {
	if name == "" {
		name = node.Name
	}
	if stats == nil {
		stats = node.Sd
	}
	return name, stats
}

// loadLegion is PassiveTree's legion load and node typing; the stats are processed where the calc
// reads them, as for tree nodes.
//
//pob2:block:src/Classes/PassiveTree.lua:64
//pob2:block:src/Classes/PassiveTree.lua:438-452
func (t *PassiveTree) loadLegion() {
	t.Legion = make([]*Node, len(poe2.LegionNodes))
	for i, l := range poe2.LegionNodes {
		node := &Node{LegionID: l.ID, Dn: l.Dn, Sd: l.Sd}
		switch {
		case l.M:
			node.Type = "Mastery"
		case l.Ks:
			node.Type = "Keystone"
			if _, ok := t.KeystoneMap[node.Dn]; !ok {
				t.KeystoneMap[node.Dn] = node
			}
		case l.Not:
			node.Type = "Notable"
		default:
			node.Type = "Normal"
		}
		t.Legion[i] = node
	}
}
