package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestEvalMultiplierDriftedBehaviours: The Multiplier tag gained FOUR behaviours between v2.18.1 and the 3.29 pin, and this port had none of them.
func TestEvalMultiplierDriftedBehaviours(t *testing.T) {
	t.Run("divVar divides by a named multiplier", func(t *testing.T) {
		db := NewModDB()
		db.Multipliers["RageSacrificed"] = 30
		db.Multipliers["RagePerRadius"] = 10
		db.AddMod(mod.NewFloat("AreaOfEffect", mod.TypeBase, 1).
			Tag(mod.Multiplier("RageSacrificed").DivVar("RagePerRadius")))

		testza.AssertEqual(t, float64(3), db.Sum(mod.TypeBase, nil, "AreaOfEffect"))
	})

	t.Run("noFloor keeps the fraction the plain form floors away", func(t *testing.T) {
		db := NewModDB()
		db.Multipliers["RamakoTattooEffect"] = 5

		floored := NewModDB()
		floored.Multipliers["RamakoTattooEffect"] = 5
		floored.AddMod(mod.NewFloat("SpellSuppressionEffect", mod.TypeBase, 1).
			Tag(mod.Multiplier("RamakoTattooEffect").Div(2)))
		testza.AssertEqual(t, float64(2), floored.Sum(mod.TypeBase, nil, "SpellSuppressionEffect"))

		db.AddMod(mod.NewFloat("SpellSuppressionEffect", mod.TypeBase, 1).
			Tag(mod.Multiplier("RamakoTattooEffect").Div(2).NoFloor()))
		testza.AssertEqual(t, float64(2.5), db.Sum(mod.TypeBase, nil, "SpellSuppressionEffect"))
	})

	t.Run("limitStat caps from a stat rather than a multiplier", func(t *testing.T) {
		db := NewModDB()
		db.Multipliers["RaisedSpiderConfig"] = 12
		db.AddMod(mod.NewFloat("Multiplier:RaisedSpider", mod.TypeBase, 1).
			Tag(mod.Multiplier("RaisedSpiderConfig").LimitStat("ActiveSpiderLimit")))

		cfg := &ListCfg{SkillStats: map[string]float64{"ActiveSpiderLimit": 4}}
		testza.AssertEqual(t, float64(4), db.Sum(mod.TypeBase, cfg, "Multiplier:RaisedSpider"))

		testza.AssertEqual(t, float64(0), db.Sum(mod.TypeBase, nil, "Multiplier:RaisedSpider"))
	})

	t.Run("limitNegTotal floors the total instead of capping it", func(t *testing.T) {
		db := NewModDB()
		db.Multipliers["Rage"] = 50
		db.AddMod(mod.NewFloat("DamageTaken", mod.TypeMore, -1).
			Tag(mod.Multiplier("Rage").Limit(-20).LimitNegTotal()))

		testza.AssertEqual(t, float64(-20), db.Sum(mod.TypeMore, nil, "DamageTaken"))
	})
}
