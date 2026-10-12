package calculator

import (
	"strings"

	"github.com/Vilsol/go-pob/data"
)

func missingNode(id int64, name, ascendancy string, group int64, stats ...string) data.Node {
	notable := true
	return data.Node{
		Skill:          &id,
		Name:           &name,
		IsNotable:      &notable,
		AscendancyName: &ascendancy,
		Group:          &group,
		Stats:          stats,
	}
}

var missingTreeNodeIDByName = func() map[string]string {
	byName := make(map[string]string, len(missingTreeNodes))
	for id, node := range missingTreeNodes {
		byName[strings.ToLower(*node.Name)] = id
	}
	return byName
}()

func resolveTreeNode(tree *data.Tree, id string) data.Node {
	if node, ok := tree.Nodes[id]; ok {
		return node
	}
	return missingTreeNodes[id]
}
