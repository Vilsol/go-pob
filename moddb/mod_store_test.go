package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

func TestClone(t *testing.T) {
	m := NewModStore(nil)
	m.Conditions["test condition"] = true
	m.Multipliers["test multiplier"] = 42.0
	r := m.Clone()
	testza.AssertEqual(t, m, r)
}

func TestMultiplier(t *testing.T) {
	tc := []struct {
		name        string
		mod         mod.Mod
		storeMods   []mod.Mod
		cfg         *ListCfg
		tag         *mod.MultiplierTag
		multipliers map[string]float64
		expected    *mod.ModValueMulti
	}{
		{
			name: "single mod",
			mod:  mod.NewFloat("testMod0", mod.TypeIncrease, 10),
			tag: &mod.MultiplierTag{
				Division:     5,
				VariableList: []string{"FullLife"},
			},
			multipliers: map[string]float64{"FullLife": 200},
			expected:    mod.NewModValueFloat(400),
		},
		{
			name: "limited mod",
			mod:  mod.NewFloat("testMod0", mod.TypeIncrease, 10),
			tag: &mod.MultiplierTag{
				Division:         5,
				VariableList:     []string{"FullLife"},
				TagLimit:         utils.Ptr(float64(15)),
				TagLimitVariable: utils.Ptr("FullLife"),
			},
			multipliers: map[string]float64{"FullLife": 200},
			expected:    mod.NewModValueFloat(150),
		},
	}

	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			m := NewModList()
			for _, tm := range test.storeMods {
				m.AddMod(tm)
			}
			m.Multipliers = test.multipliers
			got := m.evalMultiplier(test.mod.Value(), test.cfg, test.tag)
			testza.AssertEqual(t, test.expected, got)
		})
	}
}
