package calclib

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

func TestVal(t *testing.T) {
	tc := []struct {
		name     string
		mods     []mod.Mod
		expected float64
	}{
		{
			name: "base of exactly 1 keeps its increase/more",
			mods: []mod.Mod{
				mod.NewFloat("testStat", mod.TypeBase, 1),
				mod.NewFloat("testStat", mod.TypeIncrease, 50),
			},
			expected: 1.5,
		},
		{
			name: "base of zero returns zero",
			mods: []mod.Mod{
				mod.NewFloat("testStat", mod.TypeBase, 0),
				mod.NewFloat("testStat", mod.TypeIncrease, 50),
			},
			expected: 0,
		},
		{
			name: "non-unit base applies increase/more",
			mods: []mod.Mod{
				mod.NewFloat("testStat", mod.TypeBase, 4),
				mod.NewFloat("testStat", mod.TypeIncrease, 50),
			},
			expected: 6,
		},
	}

	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			m := moddb.NewModList()
			for _, tm := range test.mods {
				m.AddMod(tm)
			}
			testza.AssertEqual(t, test.expected, Val(m, "testStat"))
		})
	}
}
