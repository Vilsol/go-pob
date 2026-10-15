package data

import "strconv"

//pob:block:src/Classes/PassiveTree.lua:438-459
func (t *Tree) NodesInRadiusOf(centre int64, radiusIndex int) map[int64]bool {
	out := map[int64]bool{}
	if radiusIndex < 0 || radiusIndex >= len(JewelRadius) {
		return out
	}
	positions := t.geometry().positions
	cp := positions[centre]
	if !cp.valid {
		return out
	}
	r := JewelRadius[radiusIndex]
	for key, node := range t.Nodes {
		if !t.includedInRadius(key, node) {
			continue
		}
		id, err := strconv.ParseInt(key, 10, 64)
		if err != nil || id == centre {
			continue
		}
		p := positions[id]
		if !p.valid {
			continue
		}
		vx, vy := p.x-cp.x, p.y-cp.y
		if d2 := float64(vx*vx) + float64(vy*vy); r.Inner*r.Inner <= d2 && d2 <= r.Outer*r.Outer {
			out[id] = true
		}
	}
	return out
}
