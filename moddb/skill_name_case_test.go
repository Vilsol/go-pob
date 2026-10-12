package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

//	local matchName = tag.summonSkill and (cfg and cfg.summonSkillName or "") or (cfg and cfg.skillName or "")
//	... if name:lower() == matchName then match = true
//	skillName = string.gsub(" "..skill, "%W%l", string.upper):sub(2)

func TestSkillNameTagComparesCaseInsensitively(t *testing.T) {
	skillNameCfg := func(name string) *ListCfg {
		return &ListCfg{SkillName: name}
	}

	t.Run("a lowercase connective still matches", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("CurseEffect", mod.TypeIncrease, 20).
			Tag(mod.SkillName("Herald Of Ash")))

		testza.AssertEqual(t, 20.0,
			db.Sum(mod.TypeIncrease, skillNameCfg("Herald of Ash"), "CurseEffect"))
	})

	t.Run("an exact match still matches", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("CurseEffect", mod.TypeIncrease, 20).
			Tag(mod.SkillName("Temporal Chains")))

		testza.AssertEqual(t, 20.0,
			db.Sum(mod.TypeIncrease, skillNameCfg("Temporal Chains"), "CurseEffect"))
	})

	t.Run("a different skill still does not match", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("CurseEffect", mod.TypeIncrease, 20).
			Tag(mod.SkillName("Temporal Chains")))

		testza.AssertEqual(t, 0.0,
			db.Sum(mod.TypeIncrease, skillNameCfg("Enfeeble"), "CurseEffect"))
	})

	t.Run("a negated tag folds case as well", func(t *testing.T) {
		db := NewModDB()
		db.AddMod(mod.NewFloat("CurseEffect", mod.TypeIncrease, 20).
			Tag(mod.SkillName("Herald Of Ash").Neg(true)))

		testza.AssertEqual(t, 0.0,
			db.Sum(mod.TypeIncrease, skillNameCfg("Herald of Ash"), "CurseEffect"))
	})
}
