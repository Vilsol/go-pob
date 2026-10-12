package data

type LegionNode struct {
	ID    string
	Stats []string

	Rolls []LegionStat

	DN string
}

type LegionStat struct {
	Key string

	Fmt string

	Index int

	Min float64
	Max float64
}

func JewelRadiusIndexForLabel(label string) (int, bool) {
	if label == "" || label == "Variable" {
		return 0, false
	}
	for index, radius := range JewelRadius {
		if radius.Label == label {
			return index, true
		}
	}
	return 0, false
}

//	local matchStr = conqueredBy.conqueror.type .. "_keystone_" .. conqueredBy.conqueror.id
//	for _, legionNode in ipairs(legionNodes) do
//		if legionNode.id == matchStr then self:ReplaceNode(node, legionNode) break end
//	end
