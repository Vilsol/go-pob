package calculator

//pob:block:src/Classes/CalcsTab.lua:530-549
func (b *PowerBuilder) AscendancyPower() []PowerReportEntry {
	var out []PowerReportEntry
	for _, id := range b.nodes {
		n := b.treeNode(id)
		if n.AscendancyName == nil || *n.AscendancyName == "" {
			continue
		}
		p := b.power[id]
		if p == nil {
			continue
		}
		out = append(out, PowerReportEntry{ID: id, Type: powerNodeType(n), Power: p.single,
			Allocated: b.alloc[id] || b.granted[id], Offence: p.single, Defence: p.defence})
	}
	return out
}
