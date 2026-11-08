package exposition

import (
	"github.com/Vilsol/go-pob/calculator2"
)

// The Skills tab's edits over the last calculation's socket group list. Group and gem indexes are 1-based; each
// edit leaves the build to be calculated again.

// edited marks the skills as edited, so a save writes the list the calculation filled in, as PoB-PoE2's does. The
// first edit starts the undo buffers from the state before it.
func (c *Calculator) edited() error {
	if c.env == nil {
		return notCalculatedError{}
	}
	c.skillsEdited = true
	c.skillsUndo()
	return nil
}

// skillsUndoBuffers is the Skills tab's undo buffers; a worker holds one build, so one is kept.
var skillsUndoBuffers struct {
	calc *Calculator
	undo *calculator2.SkillsUndo
}

//pob2:block:src/Classes/SkillsTab.lua:448
func (c *Calculator) skillsUndo() *calculator2.SkillsUndo {
	if skillsUndoBuffers.calc != c {
		skillsUndoBuffers.calc, skillsUndoBuffers.undo = c, &calculator2.SkillsUndo{}
		skillsUndoBuffers.undo.ResetUndo(c.build)
	}
	return skillsUndoBuffers.undo
}

// AddSkillsUndoState is the undo state each Skills tab edit adds once it is made.
func (c *Calculator) AddSkillsUndoState() {
	c.skillsUndo().AddUndoState(c.build, false)
}

// SkillsUndo is Ctrl+Z (or Ctrl+Y with redo) on the Skills tab; it reports whether a state was restored, which the
// build is then calculated with.
//
//pob2:block:src/Classes/SkillsTab.lua:594-599
func (c *Calculator) SkillsUndo(redo bool) (bool, error) {
	if c.env == nil {
		return false, notCalculatedError{}
	}
	h := c.skillsUndo()
	var changed bool
	if redo {
		changed = h.Redo(c.build)
	} else {
		changed = h.Undo(c.build)
	}
	if changed {
		c.skillsEdited = true
	}
	return changed, nil
}

// syncSkills is what an edited build saves: the socket groups the last calculation generated or dropped.
func (c *Calculator) syncSkills() {
	if c.skillsEdited && c.env != nil {
		calculator2.SyncSocketGroups(c.build, c.env)
	}
}

// NewSocketGroup adds an empty socket group and returns its index.
func (c *Calculator) NewSocketGroup() (int, error) {
	if err := c.edited(); err != nil {
		return 0, err
	}
	return calculator2.NewSocketGroup(c.build, c.env), nil
}

func (c *Calculator) DeleteSocketGroup(index int) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.DeleteSocketGroup(c.build, c.env, index)
}

func (c *Calculator) DeleteAllSocketGroups() error {
	if err := c.edited(); err != nil {
		return err
	}
	calculator2.DeleteAllSocketGroups(c.build)
	return nil
}

// MoveSocketGroup moves the group at from to to.
func (c *Calculator) MoveSocketGroup(from, to int) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.MoveSocketGroup(c.build, c.env, from, to)
}

func (c *Calculator) SetSocketGroupLabel(index int, label string) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.SetSocketGroupLabel(c.build, c.env, index, label)
}

func (c *Calculator) SetSocketGroupEnabled(index int, state bool) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.SetSocketGroupEnabled(c.build, c.env, index, state)
}

func (c *Calculator) SetSocketGroupIncludeInFullDPS(index int, state bool) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.SetSocketGroupIncludeInFullDPS(c.build, c.env, index, state)
}

func (c *Calculator) SetSocketGroupCount(index, count int) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.SetSocketGroupCount(c.build, c.env, index, count)
}

// SetSocketGroupWeaponSet sets whether the group is used in weapon set 1 or 2.
func (c *Calculator) SetSocketGroupWeaponSet(index, set int, state bool) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.SetSocketGroupWeaponSet(c.build, c.env, index, set, state)
}

// SetGem puts the gem with data id gemID in a group's gem slot; the slot past the last gem adds one.
func (c *Calculator) SetGem(group, gem int, gemID string) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.SetGem(c.build, c.env, group, gem, gemID)
}

// AddGem sets a gem in the slot past the group's last gem, counted when the edit runs, not when it was asked for.
//
//pob2:block:src/Classes/SkillsTab.lua:815-875
func (c *Calculator) AddGem(group int, gemID string) error {
	gem := 1
	if c.env != nil && group >= 1 && group <= len(c.env.SocketGroupList) {
		gem = len(c.env.SocketGroupList[group-1].GemList) + 1
	}
	return c.SetGem(group, gem, gemID)
}

func (c *Calculator) RemoveGem(group, gem int) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.RemoveGem(c.build, c.env, group, gem)
}

// SetGemLevel sets a gem's level; 0 is an empty box.
func (c *Calculator) SetGemLevel(group, gem, level int) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.SetGemLevel(c.build, c.env, group, gem, level)
}

// SetGemQuality sets a gem's quality; a negative quality is an empty box.
func (c *Calculator) SetGemQuality(group, gem, quality int) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.SetGemQuality(c.build, c.env, group, gem, quality)
}

func (c *Calculator) SetGemEnabled(group, gem int, state bool) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.SetGemEnabled(c.build, c.env, group, gem, state)
}

// SetGemCount sets a gem's count; 0 is an empty box.
func (c *Calculator) SetGemCount(group, gem, count int) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.SetGemCount(c.build, c.env, group, gem, count)
}

// SetGemCorruption picks the Corrupt dropdown's option (1-based) for a gem.
func (c *Calculator) SetGemCorruption(group, gem, option int) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.SetGemCorruption(c.build, c.env, group, gem, option)
}

// GemOptions is the Skills tab's gem options.
func (c *Calculator) GemOptions() calculator2.GemOptions {
	return calculator2.ReadGemOptions(c.build)
}

func (c *Calculator) SetGemOptions(options calculator2.GemOptions) {
	calculator2.SetGemOptions(c.build, options)
}

// GemList is every gem the gem list can offer.
func GemList() ([]calculator2.GemListEntry, error) {
	return calculator2.GemList()
}

// SkillSetsView is the Skill set dropdown: the sets in order and the active set's id.
type SkillSetsView struct {
	Sets   []calculator2.SkillSetView
	Active int
}

func (c *Calculator) SkillSets() SkillSetsView {
	sets, active := calculator2.SkillSetList(c.build)
	return SkillSetsView{Sets: sets, Active: active}
}

func (c *Calculator) SetActiveSkillSet(id int) error {
	if err := c.edited(); err != nil {
		return err
	}
	calculator2.SetActiveSkillSet(c.build, c.env, id)
	return nil
}

// NewSkillSet adds an empty skill set, makes it the active one and returns its id.
func (c *Calculator) NewSkillSet(title string) (int, error) {
	if err := c.edited(); err != nil {
		return 0, err
	}
	return calculator2.NewSkillSet(c.build, c.env, title), nil
}

// CopySkillSet copies a skill set, makes the copy the active one and returns its id.
func (c *Calculator) CopySkillSet(id int, title string) (int, error) {
	if err := c.edited(); err != nil {
		return 0, err
	}
	return calculator2.CopySkillSet(c.build, c.env, id, title)
}

func (c *Calculator) RenameSkillSet(id int, title string) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.RenameSkillSet(c.build, id, title)
}

func (c *Calculator) DeleteSkillSet(id int) error {
	if err := c.edited(); err != nil {
		return err
	}
	return calculator2.DeleteSkillSet(c.build, c.env, id)
}

// GemSortCacheView is a gem list's sort cache: per gem, whether it supports the group and whether its DPS is
// calculated.
type GemSortCacheView struct {
	CanSupport []bool
	Pending    []bool
}

// GemSortCache is the sort cache of a group's gem list over the last calculation.
func (c *Calculator) GemSortCache(group int, gemIDs []string) (*GemSortCacheView, error) {
	canSupport, pending, err := calculator2.GemSortCache(c.env, group, gemIDs)
	if err != nil {
		return nil, err
	}
	return &GemSortCacheView{CanSupport: canSupport, Pending: pending}, nil
}

// GemSortDPSView is the DPS of each gem in a group's new gem slot and the base it is compared with.
type GemSortDPSView struct {
	Base float64
	DPS  []float64
}

// GemSortDPS calculates gems in the slot after a group's last gem, by the sort's DPS field.
func (c *Calculator) GemSortDPS(group int, gemIDs []string, field string) (*GemSortDPSView, error) {
	misc, err := c.miscCalculator()
	if err != nil {
		return nil, err
	}
	index := 1
	if group >= 1 && group <= len(c.env.SocketGroupList) {
		index = len(c.env.SocketGroupList[group-1].GemList) + 1
	}
	base, dps, err := calculator2.GemSortDPS(c.build, c.env, misc, group, index, gemIDs, field)
	if err != nil {
		return nil, err
	}
	return &GemSortDPSView{Base: base, DPS: dps}, nil
}

// CopySocketGroup is the text Ctrl+C copies for a group; "" for a group a source creates.
func (c *Calculator) CopySocketGroup(index int) (string, error) {
	return calculator2.CopySocketGroup(c.env, index)
}

// PasteSocketGroup adds the group the text describes and returns its index, or 0 when the text has no gem.
func (c *Calculator) PasteSocketGroup(text string) (int, error) {
	if err := c.edited(); err != nil {
		return 0, err
	}
	return calculator2.PasteSocketGroup(c.build, c.env, text)
}
