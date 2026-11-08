package tree2

// ClickResult says what a click on a node did.
type ClickResult int

const (
	ClickNothing ClickResult = iota
	ClickChanged
	// ClickChooseAttribute is TreeTab:ModifyAttributePopup's cue: the caller asks for an attribute and calls
	// AllocateAttributeNode.
	ClickChooseAttribute
	// ClickConfirmClassChange is the "Class Change" confirm popup's cue: the caller asks, then clicks again
	// with confirmClassChange set.
	ClickConfirmClassChange
)

//pob2:src/Classes/PassiveTreeView.lua:2097-2120
func (s *PassiveSpec) IsConnectedToWeaponSetNodes(node *SpecNode) bool {
	if node.HasPath && len(node.Path) > 1 {
		for _, pathNode := range node.Path[1:] {
			if pathNode.Alloc && pathNode.AllocMode > 0 {
				return true
			}
		}
	}
	for _, linkedNode := range node.Linked {
		if linkedNode.Alloc && linkedNode.AllocMode > 0 {
			return true
		}
	}
	return false
}

func isGlobalNode(node *SpecNode) bool {
	return node.Type == "Keystone" || node.Type == "Socket" || node.ContainJewelSocket
}

//pob2:block:src/Classes/PassiveTreeView.lua:393-408
func (s *PassiveSpec) shouldBlockGlobalNodeAllocation(node *SpecNode) bool {
	if !isGlobalNode(node) || node.Alloc || !node.HasPath {
		return false
	}
	return s.AllocMode > 0 || s.IsConnectedToWeaponSetNodes(node)
}

//pob2:block:src/Classes/PassiveTreeView.lua:410-423
func (s *PassiveSpec) shouldBlockGlobalNodeDeallocation(node *SpecNode) bool {
	if !isGlobalNode(node) || !node.Alloc {
		return false
	}
	return node.AllocMode == 0 && s.AllocMode > 0
}

// processAttributeHotkeys takes the held hotkey as the attribute index it picks (1 Strength, 2 Dexterity,
// 3 Intelligence), 0 for none.
//
//pob2:block:src/Classes/PassiveTreeView.lua:378-389
func (s *PassiveSpec) processAttributeHotkeys(node *SpecNode, hotkey int, switchAttribute bool) {
	if hotkey < 1 || hotkey > 3 {
		return
	}
	s.AttributeIndex = hotkey
	if switchAttribute {
		s.SwitchAttributeNode(node.ID, hotkey)
	}
}

// LeftClickNode is a left click on a node; tracePath is the Shift-traced path, nil when not tracing. With points
// spent off the target class and the tree not connected to it, it returns ClickConfirmClassChange, and
// confirmClassChange is the popup's Continue.
//
//pob2:block:src/Classes/PassiveTreeView.lua:425-551
func (s *PassiveSpec) LeftClickNode(node *SpecNode, hotkey int, confirmClassChange bool, tracePath []*SpecNode) (ClickResult, error) {
	hotkeyPressed := hotkey >= 1 && hotkey <= 3
	if node.Alloc && !s.shouldBlockGlobalNodeDeallocation(node) {
		if node.IsAttribute {
			if hotkeyPressed {
				s.processAttributeHotkeys(node, hotkey, true)
				s.BuildAllDependsAndPaths()
			} else {
				delete(s.HashOverrides, node.ID)
				s.DeallocNode(node)
			}
		} else {
			s.DeallocNode(node)
		}
		return ClickChanged, nil
	}
	result := ClickNothing
	if node.AscendancyName != "" {
		curBase := s.CurAscendClassBaseName
		if s.CurAscendClass.Replace != "" {
			curBase = s.CurAscendClass.Replace
		}
		if s.CurAscendClassID == 0 || node.AscendancyName != curBase {
			targetAscendClassID := -1
			for ascendClassID, ascendClass := range s.CurClass.Classes {
				if ascendClass.ID == node.AscendancyName {
					targetAscendClassID = ascendClassID
					break
				}
			}
			if targetAscendClassID >= 0 {
				s.SelectAscendClass(targetAscendClassID)
				result = ClickChanged
			} else {
				targetBaseClassID := -1
				for _, classID := range sortedKeys(s.Tree.Classes) {
					for ascendClassID, ascendClass := range s.Tree.Classes[classID].Classes {
						if ascendClass.ID == node.AscendancyName {
							targetBaseClassID, targetAscendClassID = classID, ascendClassID
							break
						}
					}
					if targetBaseClassID >= 0 {
						break
					}
				}
				if targetBaseClassID >= 0 {
					used, _, _, _, _, _ := s.CountAllocNodes()
					connected := used == 0 || s.IsClassConnected(targetBaseClassID)
					if !connected && !confirmClassChange {
						return ClickConfirmClassChange, nil
					}
					s.SelectClass(targetBaseClassID)
					s.SelectAscendClass(targetAscendClassID)
					if targetNode, ok := s.Nodes[node.ID]; ok && !targetNode.Alloc {
						if err := s.AllocNode(targetNode, nil); err != nil {
							return ClickChanged, err
						}
					}
					// the popup's callback ends the click
					if !connected {
						return ClickChanged, nil
					}
					result = ClickChanged
				}
			}
		}
	}
	if node.HasPath && !s.shouldBlockGlobalNodeAllocation(node) {
		if node.IsAttribute && !hotkeyPressed {
			return ClickChooseAttribute, nil
		}
		if hotkeyPressed {
			s.processAttributeHotkeys(node, hotkey, node.IsAttribute)
		}
		return ClickChanged, s.AllocNode(node, tracedPath(node, tracePath))
	}
	return result, nil
}

// tracedPath is the trace when it ends at node, the path a click on node allocates, else nil.
//
//pob2:block:src/Classes/PassiveTreeView.lua:546
func tracedPath(node *SpecNode, tracePath []*SpecNode) []*SpecNode {
	if len(tracePath) > 0 && tracePath[len(tracePath)-1] == node {
		return tracePath
	}
	return nil
}

// AllocateAttributeNode is the attribute popup's Allocate button; its Cancel only deallocates the
// unallocated node, which changes nothing.
//
//pob2:block:src/Classes/TreeTab.lua:857-864
func (s *PassiveSpec) AllocateAttributeNode(node *SpecNode, attributeIndex int) error {
	s.SwitchAttributeNode(node.ID, attributeIndex)
	s.AttributeIndex = attributeIndex
	return s.AllocNode(node, nil)
}

// RightClickNode is a right click on a node, less the note popup and the jump to a socket's item slot; tracePath
// is the Shift-traced path, nil when not tracing.
//
//pob2:block:src/Classes/PassiveTreeView.lua:553-594
func (s *PassiveSpec) RightClickNode(node *SpecNode, hotkey int, tracePath []*SpecNode) (ClickResult, error) {
	if node.Alloc && (node.Type == "Socket" || node.ContainJewelSocket) {
		return ClickNothing, nil
	}
	if hotkey >= 1 && hotkey <= 3 {
		s.processAttributeHotkeys(node, hotkey, node.IsAttribute)
	} else if node.IsAttribute {
		switch node.Dn {
		case "Intelligence":
			s.AttributeIndex = 1
		case "Dexterity":
			s.AttributeIndex = 3
		case "Strength":
			s.AttributeIndex = 2
		}
		attributeIndex := s.AttributeIndex
		if attributeIndex == 0 {
			attributeIndex = 1
		}
		s.SwitchAttributeNode(node.ID, attributeIndex)
	}
	return ClickChanged, s.AllocNode(node, tracedPath(node, tracePath))
}
