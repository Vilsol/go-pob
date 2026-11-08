package exposition

import (
	"context"
	"sync"

	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/pob2"
	"github.com/Vilsol/go-pob/storage"
)

// TreeVersion is the passive tree every build is calculated against; go-pob-data publishes 0_5 only.
const TreeVersion = "0_5"

var (
	treeMu sync.Mutex
	tree   *tree2.PassiveTree
)

// loadTree keeps the tree once it loads, and retries a failed fetch on the next call.
func loadTree() (*tree2.PassiveTree, error) {
	treeMu.Lock()
	defer treeMu.Unlock()
	if tree != nil {
		return tree, nil
	}
	t, err := tree2.LoadData(context.Background(), TreeVersion, assetcache.AssetLoaderWrapper{Storage: storage.Get()})
	if err != nil {
		return nil, err
	}
	tree = t
	return tree, nil
}

// Preload loads the passive tree and the game data ahead of the first calculation.
func Preload() error {
	if _, err := loadTree(); err != nil {
		return err
	}
	return calculator2.Preload()
}

// Calculator holds one parsed PoE2 build and its last calculation.
type Calculator struct {
	build *pob2.PathOfBuilding2
	env   *calculator2.Environment
	used  *calculator2.UsedTables
	items *calculator2.ItemsTab
	// undo is the spec's undo handler, reset on the first calculation as PassiveSpec:Load resets it.
	undo *tree2.UndoHandler
	// skillsEdited is set once the Skills tab has edited the build.
	skillsEdited bool
}

// Outputs is the player's output table, the main skill's minion's when it has one, and the build's
// class and level as the calculation resolved them, with what the sidebar and config page read.
type Outputs struct {
	Output          map[string]float64
	Minion          map[string]float64
	ClassName       string
	AscendClassName string
	Level           int
	// SkillFlags is the main skill's activeEffect.statSet.skillFlags; MinionSkillFlags the minion's.
	SkillFlags       map[string]bool
	MinionSkillFlags map[string]bool
	// SkillData is the main skill's skillData keys Lua reads as truthy.
	SkillData    map[string]bool
	SocketGroups []calculator2.SocketGroupView
	// MainSocketGroup and MainActiveSkill are 1-based, as the calculation resolved them.
	MainSocketGroup int
	MainActiveSkill int
	ConfigInputs    map[string]calculator2.ConfigValue
	// SkillDPS is output.SkillDPS, the Full DPS rows.
	SkillDPS []calculator2.SkillDPS
}

func ParseBuild(xml string) (*Calculator, error) {
	build, err := pob2.ParseBuild([]byte(xml))
	if err != nil {
		return nil, err
	}
	return &Calculator{build: build}, nil
}

func (c *Calculator) Serialize() (string, error) {
	c.syncSkills()
	out, err := pob2.SerializeBuild(c.build)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// SavedStat is one PlayerStat or MinionStat row; SavedFullDPS one FullDPSSkill row.
type SavedStat struct {
	Stat  string
	Value float64
}

type SavedFullDPS struct {
	Stat      string
	Value     float64
	SkillPart string
	Source    string
}

// SerializeWithStats writes the build with the stat rows Save lists from the calculated output in place of the ones
// it was loaded with.
//
//pob2:block:src/Modules/Build.lua:1196-1244
func (c *Calculator) SerializeWithStats(player []SavedStat, fullDPS []SavedFullDPS, minion []SavedStat) (string, error) {
	c.syncSkills()
	out := *c.build
	// Save writes every row afresh, after Spectre and BeastCompanion and before Buffs
	out.Build.Order = nil
	out.Build.PlayerStats = make([]pob2.PlayerStat, len(player))
	for i, s := range player {
		out.Build.PlayerStats[i] = pob2.PlayerStat{Stat: s.Stat, Value: s.Value}
	}
	out.Build.FullDPSSkills = make([]pob2.FullDPSSkill, len(fullDPS))
	for i, s := range fullDPS {
		out.Build.FullDPSSkills[i] = pob2.FullDPSSkill{Stat: s.Stat, Value: s.Value, SkillPart: s.SkillPart, Source: s.Source}
	}
	out.Build.MinionStats = make([]pob2.PlayerStat, len(minion))
	for i, s := range minion {
		out.Build.MinionStats[i] = pob2.PlayerStat{Stat: s.Stat, Value: s.Value}
	}
	text, err := pob2.SerializeBuild(&out)
	if err != nil {
		return "", err
	}
	return string(text), nil
}

func statSetFlags(skill *calculator2.ActiveSkill) map[string]bool {
	out := map[string]bool{}
	if skill == nil || skill.ActiveEffect == nil || skill.ActiveEffect.StatSet == nil {
		return out
	}
	for k, v := range skill.ActiveEffect.StatSet.SkillFlags {
		if v {
			out[k] = true
		}
	}
	return out
}

func truthySkillData(skill *calculator2.ActiveSkill) map[string]bool {
	out := map[string]bool{}
	if skill == nil {
		return out
	}
	for k, v := range skill.SkillData {
		if b, ok := v.(bool); v != nil && (!ok || b) {
			out[k] = true
		}
	}
	return out
}

func (c *Calculator) Calculate() (*Outputs, error) {
	t, err := loadTree()
	if err != nil {
		return nil, err
	}
	items := calculator2.NewItemsTab(c.build, t, nil)
	if c.items != nil {
		items.DisplayItem = c.items.DisplayItem
	}
	env, err := calculator2.BuildOutput(c.build, calculator2.Options{Tree: t, Items: items})
	if err != nil {
		return nil, err
	}
	c.env, c.used, c.items = env, calculator2.BuildOutputUsed(env), items
	// Go must differ: PoB sets the level after the pass and shows that pass's stats; this calculates again at it
	//pob2:block:src/Modules/Build.lua:1038-1042
	if c.build.Build.CharacterLevelAutoMode {
		if level := calculator2.EstimatePlayerProgress(env.Spec, env.Player.Output).Level; level != c.build.Build.Level {
			c.build.Build.Level = level
			env, err = calculator2.BuildOutput(c.build, calculator2.Options{Tree: t, Items: items})
			if err != nil {
				return nil, err
			}
			c.env, c.used = env, calculator2.BuildOutputUsed(env)
		}
	}
	if c.undo == nil {
		c.undo = &tree2.UndoHandler{}
		c.undo.ResetUndo(env.Spec)
	}
	// the sidebar sorts output.SkillDPS in place before Save writes it
	calculator2.SortSkillDPS(env.Player.SkillDPS)
	res := &Outputs{
		Output:          env.Player.Output,
		ClassName:       env.Spec.CurClassName,
		AscendClassName: env.Spec.CurAscendClassName,
		Level:           env.Player.Level,
		SkillFlags:      statSetFlags(env.Player.MainSkill),
		SkillData:       truthySkillData(env.Player.MainSkill),
		SocketGroups:    calculator2.SocketGroupViews(env),
		MainSocketGroup: env.MainSocketGroup,
		ConfigInputs:    calculator2.ConfigInputs(c.build),
		SkillDPS:        env.Player.SkillDPS,
	}
	if i := env.MainSocketGroup; i >= 1 && i <= len(env.SocketGroupList) {
		res.MainActiveSkill = env.SocketGroupList[i-1].MainActiveSkill
	}
	if main := env.Player.MainSkill; main != nil && main.Minion != nil {
		res.Minion = main.Minion.Output
		res.MinionSkillFlags = statSetFlags(main.Minion.MainSkill)
	}
	return res, nil
}

// SetConfigInput sets one config option of the active config set; isDefault drops it instead.
func (c *Calculator) SetConfigInput(name string, value calculator2.ConfigValue, isDefault bool) {
	calculator2.SetConfigInput(c.build, name, value, isDefault)
}

// SetLevel is the level box: it clamps to 1-100 and turns the automatic level off.
//
//pob2:block:src/Modules/Build.lua:209-216
func (c *Calculator) SetLevel(level int) {
	c.build.Build.Level = min(max(level, 1), 100)
	c.build.Build.CharacterLevelAutoMode = false
}

//pob2:block:src/Modules/Build.lua:202-208
func (c *Calculator) SetCharacterLevelAutoMode(auto bool) {
	c.build.Build.CharacterLevelAutoMode = auto
}

// ConfigSetsView is the config set list in its order and the active set's id.
type ConfigSetsView struct {
	Sets   []calculator2.ConfigSetView
	Active int
}

func (c *Calculator) ConfigSets() ConfigSetsView {
	sets, active := calculator2.ConfigSetList(c.build)
	return ConfigSetsView{Sets: sets, Active: active}
}

func (c *Calculator) SetActiveConfigSet(id int) {
	calculator2.SetActiveConfigSet(c.build, id)
}

func (c *Calculator) NewConfigSet(title string) int {
	return calculator2.NewConfigSet(c.build, title)
}

func (c *Calculator) CopyConfigSet(id int, title string) int {
	return calculator2.CopyConfigSet(c.build, id, title)
}

func (c *Calculator) RenameConfigSet(id int, title string) {
	calculator2.RenameConfigSet(c.build, id, title)
}

func (c *Calculator) DeleteConfigSet(id int) {
	calculator2.DeleteConfigSet(c.build, id)
}

func (c *Calculator) MoveConfigSet(from, to int) {
	calculator2.MoveConfigSet(c.build, from, to)
}

func (c *Calculator) SetMainSocketGroup(index int) {
	calculator2.SetMainSocketGroup(c.build, index)
}

func (c *Calculator) SetMainActiveSkill(index int) {
	calculator2.SetMainActiveSkill(c.build, index)
}

type notCalculatedError struct{}

func (notCalculatedError) Error() string { return "the build has not been calculated" }

// ConfigRelevance answers a config visibility query against the last calculation.
func (c *Calculator) ConfigRelevance(query calculator2.ConfigRelevance) (calculator2.ConfigRelevance, error) {
	if c.env == nil {
		return calculator2.ConfigRelevance{}, notCalculatedError{}
	}
	return c.env.ConfigRelevance(c.used, query), nil
}
