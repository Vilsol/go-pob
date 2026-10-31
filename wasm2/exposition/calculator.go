package exposition

import (
	"context"
	"sync"

	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/data/raw"
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
	t, err := tree2.LoadData(context.Background(), TreeVersion, raw.AssetLoaderWrapper{Storage: storage.Get()})
	if err != nil {
		return nil, err
	}
	tree = t
	return tree, nil
}

// Calculator holds one parsed PoE2 build.
type Calculator struct {
	build *pob2.PathOfBuilding2
}

// Outputs is the player's output table, the main skill's minion's when it has one, and the build's
// class and level as the calculation resolved them.
type Outputs struct {
	Output          map[string]float64
	Minion          map[string]float64
	ClassName       string
	AscendClassName string
	Level           int
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

func (c *Calculator) Calculate() (*Outputs, error) {
	t, err := loadTree()
	if err != nil {
		return nil, err
	}
	env, err := calculator2.BuildOutput(c.build, calculator2.Options{Tree: t, Items: calculator2.NewItemsTab(c.build, t, nil)})
	if err != nil {
		return nil, err
	}
	res := &Outputs{
		Output:          env.Player.Output,
		ClassName:       env.Spec.CurClassName,
		AscendClassName: env.Spec.CurAscendClassName,
		Level:           env.Player.Level,
	}
	if main := env.Player.MainSkill; main != nil && main.Minion != nil {
		res.Minion = main.Minion.Output
	}
	return res, nil
}
