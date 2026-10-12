package data

func (t *Tree) NodePosition(id int64) (float64, float64, bool) {
	p := t.geometry().positions[id]
	return p.x, p.y, p.valid
}
