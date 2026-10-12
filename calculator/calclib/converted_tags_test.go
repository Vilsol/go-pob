package calclib

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestGetConvertedModTags_LimitlessTagStaysLimitless: PoB CalcTools.lua:206 gates the limit rewrite on `value.limitTotal`.
func TestGetConvertedModTags_LimitlessTagStaysLimitless(t *testing.T) {
	t.Run("multiplier", func(t *testing.T) {
		m := mod.NewFloat("Damage", mod.TypeIncrease, 10).Tag(mod.Multiplier("M"))
		tags := GetConvertedModTags(m, 0.5, false)

		out, ok := tags[0].(*mod.MultiplierTag)
		testza.AssertTrue(t, ok)
		testza.AssertNil(t, out.TagLimit)
	})

	t.Run("per stat", func(t *testing.T) {
		m := mod.NewFloat("Damage", mod.TypeIncrease, 10).Tag(mod.PerStat(10, "Strength"))
		tags := GetConvertedModTags(m, 0.5, false)

		out, ok := tags[0].(*mod.PerStatTag)
		testza.AssertTrue(t, ok)
		testza.AssertNil(t, out.TagLimit)
	})

	t.Run("per stat with limitVar keeps the limitVar path", func(t *testing.T) {
		m := mod.NewFloat("Damage", mod.TypeIncrease, 10).
			Tag(mod.PerStat(10, "Strength").LimitVar("SomeVar"))
		tags := GetConvertedModTags(m, 0.5, false)

		out, ok := tags[0].(*mod.PerStatTag)
		testza.AssertTrue(t, ok)
		testza.AssertNil(t, out.TagLimit)
		testza.AssertNotNil(t, out.TagLimitVariable)
		testza.AssertEqual(t, "SomeVar", *out.TagLimitVariable)
	})
}

// TestGetConvertedModTags_LimitScaling: Only `limitTotal` tags get `copy.limit = copy.limit * multiplier`.
func TestGetConvertedModTags_LimitScaling(t *testing.T) {
	t.Run("limitTotal limit is scaled", func(t *testing.T) {
		m := mod.NewFloat("Damage", mod.TypeIncrease, 10).
			Tag(mod.Multiplier("M").Limit(5).LimitTotal(true))
		tags := GetConvertedModTags(m, 0.5, false)

		out := tags[0].(*mod.MultiplierTag)
		testza.AssertNotNil(t, out.TagLimit)
		testza.AssertEqual(t, float64(2.5), *out.TagLimit)
	})

	t.Run("plain limit is copied unscaled", func(t *testing.T) {
		m := mod.NewFloat("Damage", mod.TypeIncrease, 10).
			Tag(mod.Multiplier("M").Limit(5))
		tags := GetConvertedModTags(m, 0.5, false)

		out := tags[0].(*mod.MultiplierTag)
		testza.AssertNotNil(t, out.TagLimit)
		testza.AssertEqual(t, float64(5), *out.TagLimit)
	})

	t.Run("source tag is not mutated", func(t *testing.T) {
		tag := mod.Multiplier("M").Limit(5).LimitTotal(true)
		m := mod.NewFloat("Damage", mod.TypeIncrease, 10).Tag(tag)
		GetConvertedModTags(m, 0.5, false)

		testza.AssertEqual(t, float64(5), *tag.TagLimit)
	})
}

// TestGetConvertedModTags_ConvertedModIsNotZeroed: End to end: a converted mod carrying a limitless Multiplier must still evaluate.
func TestGetConvertedModTags_ConvertedModIsNotZeroed(t *testing.T) {
	db := moddb.NewModDB()
	db.Multipliers["M"] = 3

	source := mod.NewFloat("Damage", mod.TypeIncrease, 10).Tag(mod.Multiplier("M"))
	converted := mod.NewFloat("Damage", mod.TypeIncrease, 10*0.5).
		Tag(GetConvertedModTags(source, 0.5, false)...)
	db.AddMod(converted)

	testza.AssertEqual(t, float64(15), db.Sum(mod.TypeIncrease, nil, "Damage"))
}
