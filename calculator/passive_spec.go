package calculator

import (
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

type PassiveSpec struct {
	// TODO UndoHandler

	Build       *pob.PathOfBuilding
	TreeVersion pob.TreeVersion

	Nodes              map[string]interface{} // TODO Implement
	AllocNodes         map[string]data.Node
	AllocSubgraphNodes map[string]interface{} // TODO Implement
	AllocExtendedNodes map[string]interface{} // TODO Implement
	Jewels             map[string]interface{} // TODO Implement
	SubGraphs          map[string]interface{} // TODO Implement
	MasterySelections  map[string]interface{} // TODO Implement

	ClassID        pob.ClassID
	ClassName      pob.ClassName
	AscendancyID   pob.AscendancyID
	AscendancyName pob.AscendancyName

	AllocatedNotableCount int
	AllocatedMasteryCount int
}

func NewPassiveSpec(build *pob.PathOfBuilding, treeVersion pob.TreeVersion) *PassiveSpec {
	passiveSpec := &PassiveSpec{
		Build:       build,
		TreeVersion: treeVersion,
	}

	passiveSpec.SelectClass(pob.Scion)

	return passiveSpec
}

func (p *PassiveSpec) Tree() *data.Tree {
	return data.TreeVersions[p.TreeVersion].Tree()
}

func (p *PassiveSpec) Class() data.Class {
	return p.Tree().Classes[p.ClassID]
}

func (p *PassiveSpec) SelectClass(classID pob.ClassID) {
	/*
		TODO Implement
		if self.curClassId then
			-- Deallocate the current class's starting node
			local oldStartNodeId = self.curClass.startNodeId
			self.nodes[oldStartNodeId].alloc = false
			self.allocNodes[oldStartNodeId] = nil
		end
	*/

	p.ClassID = classID
	p.ClassName = pob.ClassNameByID[classID]

	/*
		TODO Implement
		-- Allocate the new class's starting node
		local startNode = self.nodes[class.startNodeId]
		startNode.alloc = true
		self.allocNodes[startNode.id] = startNode
	*/

	p.SelectAscendancyClass(pob.ClassAscendancies[p.ClassID][0])
}

func (p *PassiveSpec) SelectAscendancyClass(ascendancyID pob.AscendancyID) {
	p.AscendancyID = ascendancyID
	p.AscendancyName = pob.AscendancyNameByID[ascendancyID]

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
