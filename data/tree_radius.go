package data

import (
	"math"
	"strconv"
	"sync"
)

//pob:src/Classes/PassiveTree.lua:1015-1036
func calcOrbitAngles(nodesInOrbit int) []float64 {
	var degrees []float64
	switch nodesInOrbit {
	case 16:
		degrees = []float64{0, 30, 45, 60, 90, 120, 135, 150, 180, 210, 225, 240, 270, 300, 315, 330}
	case 40:
		degrees = []float64{
			0, 10, 20, 30, 40, 45, 50, 60, 70, 80, 90, 100, 110, 120, 130, 135, 140, 150, 160, 170,
			180, 190, 200, 210, 220, 225, 230, 240, 250, 260, 270, 280, 290, 300, 310, 315, 320, 330, 340, 350,
		}
	default:
		degrees = make([]float64, nodesInOrbit+1)
		for i := 0; i <= nodesInOrbit; i++ {
			degrees[i] = 360 * float64(i) / float64(nodesInOrbit)
		}
	}

	angles := make([]float64, len(degrees))
	for i, d := range degrees {
		angles[i] = d * math.Pi / 180
	}
	return angles
}

//pob:block:src/Classes/PassiveTree.lua:871-877
func (t *Tree) OrbitPosition(group int64, orbit, oidx int) (float64, float64, bool) {
	g, ok := t.Groups[strconv.FormatInt(group, 10)]
	if !ok || orbit < 0 || orbit >= len(t.Constants.OrbitRadii) || orbit >= len(t.Constants.SkillsPerOrbit) {
		return 0, 0, false
	}
	angles := calcOrbitAngles(int(t.Constants.SkillsPerOrbit[orbit]))
	if oidx < 0 || oidx >= len(angles) {
		return 0, 0, false
	}
	radius := float64(t.Constants.OrbitRadii[orbit])
	return g.X + math.Sin(angles[oidx])*radius, g.Y - math.Cos(angles[oidx])*radius, true
}

type nodePos struct {
	x, y  float64
	valid bool
}

type treeGeometry struct {
	positions        map[int64]nodePos
	jewelRadiusNodes map[int64][][]int64
}

var (
	treeGeometryOnce sync.Map
	treeGeometryData sync.Map
)

func nodeID(key string, n Node) (int64, bool) {
	if n.Skill != nil {
		return *n.Skill, true
	}
	id, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func boolVal(b *bool) bool { return b != nil && *b }

func (t *Tree) geometry() *treeGeometry {
	onceAny, _ := treeGeometryOnce.LoadOrStore(t, &sync.Once{})
	onceAny.(*sync.Once).Do(func() {
		treeGeometryData.Store(t, t.buildGeometry())
	})
	g, _ := treeGeometryData.Load(t)
	return g.(*treeGeometry)
}

//pob:block:src/Classes/PassiveTree.lua:204-209,src/Classes/PassiveTree.lua:871-877
func (t *Tree) buildGeometry() *treeGeometry {
	orbitRadii := t.Constants.OrbitRadii
	skillsPerOrbit := t.Constants.SkillsPerOrbit

	angleTables := make([]([]float64), len(skillsPerOrbit))
	for orbit, count := range skillsPerOrbit {
		angleTables[orbit] = calcOrbitAngles(int(count))
	}

	positions := make(map[int64]nodePos, len(t.Nodes))
	for key, node := range t.Nodes {
		id, ok := nodeID(key, node)
		if !ok {
			continue
		}

		if node.Group == nil || node.Orbit == nil || node.OrbitIndex == nil {
			positions[id] = nodePos{}
			continue
		}

		group, ok := t.Groups[strconv.FormatInt(*node.Group, 10)]
		if !ok {
			positions[id] = nodePos{}
			continue
		}

		orbit := int(*node.Orbit)
		oidx := int(*node.OrbitIndex)
		if orbit < 0 || orbit >= len(orbitRadii) || orbit >= len(angleTables) {
			positions[id] = nodePos{}
			continue
		}
		angles := angleTables[orbit]
		if oidx < 0 || oidx >= len(angles) {
			positions[id] = nodePos{}
			continue
		}

		angle := angles[oidx]
		radius := float64(orbitRadii[orbit])
		positions[id] = nodePos{
			x:     group.X + math.Sin(angle)*radius,
			y:     group.Y - math.Cos(angle)*radius,
			valid: true,
		}
	}

	return &treeGeometry{
		positions:        positions,
		jewelRadiusNodes: t.buildJewelRadiusNodes(positions),
	}
}

//pob:block:src/Classes/PassiveTree.lua:425
func (t *Tree) includedInRadius(key string, node Node) bool {
	if node.Group == nil || boolVal(node.IsMastery) || boolVal(node.IsProxy) || boolVal(node.IsBlighted) {
		return false
	}
	if node.AscendancyName != nil {
		return false
	}
	group, ok := t.Groups[strconv.FormatInt(*node.Group, 10)]
	if !ok || boolVal(group.IsProxy) {
		return false
	}
	return true
}

//pob:block:src/Classes/PassiveTree.lua:415-436
func (t *Tree) buildJewelRadiusNodes(positions map[int64]nodePos) map[int64][][]int64 {
	type cand struct {
		id   int64
		x, y float64
	}
	candidates := make([]cand, 0, len(t.Nodes))
	for key, node := range t.Nodes {
		if !t.includedInRadius(key, node) {
			continue
		}
		id, ok := nodeID(key, node)
		if !ok {
			continue
		}
		p := positions[id]
		if !p.valid {
			continue
		}
		candidates = append(candidates, cand{id: id, x: p.x, y: p.y})
	}

	radii := make([]struct{ inner2, outer2 float64 }, len(JewelRadius))
	for i, r := range JewelRadius {
		radii[i] = struct{ inner2, outer2 float64 }{r.Inner * r.Inner, r.Outer * r.Outer}
	}

	result := make(map[int64][][]int64)
	for key, node := range t.Nodes {
		if !boolVal(node.IsJewelSocket) {
			continue
		}
		if node.Name != nil && *node.Name == "Charm Socket" {
			continue
		}
		socketID, ok := nodeID(key, node)
		if !ok {
			continue
		}
		sp := positions[socketID]
		if !sp.valid {
			continue
		}

		perRadius := make([][]int64, len(JewelRadius))
		for i := range perRadius {
			perRadius[i] = []int64{}
		}

		for _, c := range candidates {
			if c.id == socketID {
				continue
			}
			vx, vy := c.x-sp.x, c.y-sp.y
			d2 := vx*vx + vy*vy
			for i := range radii {
				if radii[i].inner2 <= d2 && d2 <= radii[i].outer2 {
					perRadius[i] = append(perRadius[i], c.id)
				}
			}
		}

		result[socketID] = perRadius
	}

	return result
}

func (t *Tree) JewelRadiusNodes() map[int64][][]int64 {
	return t.geometry().jewelRadiusNodes
}
