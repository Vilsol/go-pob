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

// Calculator holds one parsed PoE2 build and its last calculation.
type Calculator struct {
	build *pob2.PathOfBuilding2
	env   *calculator2.Environment
	used  *calculator2.UsedTables
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
	SocketGroups     []calculator2.SocketGroupView
	// MainSocketGroup and MainActiveSkill are 1-based, as the calculation resolved them.
	MainSocketGroup int
	MainActiveSkill int
	ConfigInputs    map[string]calculator2.ConfigValue
}

func ParseBuild(xml string) (*Calculator, error) {
	build, err := pob2.ParseBuild([]byte(xml))
	if err != nil {
		return nil, err
	}
	return &Calculator{build: build}, nil
}

func (c *Calculator) Serialize() (string, error) {
	out, err := pob2.SerializeBuild(c.build)
	if err != nil {
		return "", err
	}
	return string(out), nil
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

func (c *Calculator) Calculate() (*Outputs, error) {
	t, err := loadTree()
	if err != nil {
		return nil, err
	}
	env, err := calculator2.BuildOutput(c.build, calculator2.Options{Tree: t, Items: calculator2.NewItemsTab(c.build, t, nil)})
	if err != nil {
		return nil, err
	}
	c.env, c.used = env, calculator2.BuildOutputUsed(env)
	res := &Outputs{
		Output:          env.Player.Output,
		ClassName:       env.Spec.CurClassName,
		AscendClassName: env.Spec.CurAscendClassName,
		Level:           env.Player.Level,
		SkillFlags:      statSetFlags(env.Player.MainSkill),
		SocketGroups:    calculator2.SocketGroupViews(env),
		MainSocketGroup: env.MainSocketGroup,
		ConfigInputs:    calculator2.ConfigInputs(c.build),
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
