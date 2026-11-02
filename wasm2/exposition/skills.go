package exposition

import (
	"github.com/Vilsol/go-pob/calculator2"
)

// The Skills tab's edits over the last calculation's socket group list. Group and gem indexes are 1-based; each
// edit leaves the build to be calculated again.

// edited marks the skills as edited, so a save writes the list the calculation filled in, as PoB-PoE2's does.
func (c *Calculator) edited() error {
	if c.env == nil {
		return notCalculatedError{}
	}
	c.skillsEdited = true
	return nil
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
