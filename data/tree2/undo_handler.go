package tree2

// UndoHandler keeps PassiveSpec's undo and redo buffers. The spec is rebuilt from the build on every
// calculation, so the buffers live outside it and each call takes the spec they apply to.
type UndoHandler struct {
	undo []*UndoState
	redo []*UndoState
}

//pob2:src/Classes/UndoHandler.lua:23-27
func (h *UndoHandler) ResetUndo(s *PassiveSpec) {
	h.undo = []*UndoState{s.CreateUndoState()}
	h.redo = nil
}

//pob2:src/Classes/UndoHandler.lua:31-38
func (h *UndoHandler) AddUndoState(s *PassiveSpec, noClearRedo bool) {
	h.undo = append([]*UndoState{s.CreateUndoState()}, h.undo...)
	if len(h.undo) > 101 {
		h.undo = h.undo[:101]
	}
	if !noClearRedo {
		h.redo = nil
	}
}

// Undo reports whether it restored a state.
//
//pob2:src/Classes/UndoHandler.lua:41-47
func (h *UndoHandler) Undo(s *PassiveSpec) bool {
	if len(h.undo) < 2 {
		return false
	}
	h.redo = append([]*UndoState{h.undo[0]}, h.redo...)
	state := h.undo[1]
	h.undo = h.undo[2:]
	s.RestoreUndoState(state)
	h.AddUndoState(s, true)
	return true
}

// Redo reports whether it restored a state.
//
//pob2:src/Classes/UndoHandler.lua:50-55
func (h *UndoHandler) Redo(s *PassiveSpec) bool {
	if len(h.redo) == 0 {
		return false
	}
	state := h.redo[0]
	h.redo = h.redo[1:]
	s.RestoreUndoState(state)
	h.AddUndoState(s, true)
	return true
}
