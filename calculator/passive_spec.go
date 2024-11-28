package calculator

import (
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

type PassiveSpec struct {
	// TODO UndoHandler

	Build       *pob.PathOfBuilding
	TreeVersion data.TreeVersion

	Nodes              map[string]interface{} // TODO Implement
	AllocNodes         map[string]data.Node
	AllocSubgraphNodes map[string]interface{} // TODO Implement
	AllocExtendedNodes map[string]interface{} // TODO Implement
	Jewels             map[string]interface{} // TODO Implement
	SubGraphs          map[string]interface{} // TODO Implement
	MasterySelections  map[string]interface{} // TODO Implement

	ClassName      data.ClassName
	AscendancyName data.AscendancyName

	AllocatedNotableCount int
	AllocatedMasteryCount int
}

func NewPassiveSpec(build *pob.PathOfBuilding, treeVersion data.TreeVersion) *PassiveSpec {
	passiveSpec := &PassiveSpec{
		Build:       build,
		TreeVersion: treeVersion,
	}

	passiveSpec.SelectClass(data.Scion)

	return passiveSpec
}

func (p *PassiveSpec) Tree() *data.Tree {
	return data.TreeVersions[p.TreeVersion].Tree()
}

func (p *PassiveSpec) Class() data.Class {
	return p.Tree().Classes[data.ClassIDs[p.ClassName]]
}

func (p *PassiveSpec) SelectClass(className data.ClassName) {
	/*
		TODO Implement
		if self.curClassId then
			-- Deallocate the current class's starting node
			local oldStartNodeId = self.curClass.startNodeId
			self.nodes[oldStartNodeId].alloc = false
			self.allocNodes[oldStartNodeId] = nil
		end
	*/

	p.ClassName = className

	/*
		TODO Implement
		-- Allocate the new class's starting node
		local startNode = self.nodes[class.startNodeId]
		startNode.alloc = true
		self.allocNodes[startNode.id] = startNode
	*/

	p.SelectAscendancyClass(data.ClassAscendancies[className][0])
}

func (p *PassiveSpec) SelectAscendancyClass(ascendancyName data.AscendancyName) {
	p.AscendancyName = ascendancyName

	/*
		TODO Implement
		-- Deallocate any allocated ascendancy nodes that don't belong to the new ascendancy class
		for id, node in pairs(self.allocNodes) do
			if node.ascendancyName and node.ascendancyName ~= ascendClass.name then
				node.alloc = false
				self.allocNodes[id] = nil
			end
		end
	*/

	/*
		TODO Implement
		if ascendClass.startNodeId then
			-- Allocate the new ascendancy class's start node
			local startNode = self.nodes[ascendClass.startNodeId]
			startNode.alloc = true
			self.allocNodes[startNode.id] = startNode
		end
	*/

	/*
		TODO Implement
		self:BuildAllDependsAndPaths()
	*/
}
