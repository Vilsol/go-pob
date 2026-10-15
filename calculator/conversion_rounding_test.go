package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestScaledConversionRoundsToWholePercent pins CalcOffence.lua:1973-1982.
func TestScaledConversionRoundsToWholePercent(t *testing.T) {
	list := moddb.NewModList()
	for _, name := range []string{"PhysicalDamageConvertToLightning", "PhysicalDamageConvertToCold", "PhysicalDamageConvertToFire"} {
		list.AddMod(mod.NewFloat(name, mod.TypeBase, 40))
	}
	phys := buildConversionTable(list, nil)[data.DamageTypePhysical]
	for _, dst := range []data.DamageType{data.DamageTypeLightning, data.DamageTypeCold, data.DamageTypeFire} {
		if got := phys.Conversion[dst]; got != 0.33 {
			t.Errorf("conversion to %s = %v, want 0.33", dst, got)
		}
	}
	if got := phys.Mult; got < 0.01-1e-6 || got > 0.01+1e-6 {
		t.Errorf("mult = %v, want 0.01", got)
	}
}

// TestConvertedDamageRoundedAtFinalType pins CalcOffence.lua:104-107 and :134-140 (#10263): a conversion chain stays fractional until its final type.
func TestConvertedDamageRoundedAtFinalType(t *testing.T) {
	list := moddb.NewModList()
	list.AddMod(mod.NewFloat("PhysicalDamageConvertToLightning", mod.TypeBase, 50))
	list.AddMod(mod.NewFloat("LightningDamageConvertToCold", mod.TypeBase, 50))
	skill := &ActiveSkill{SkillModList: list, ConversionTable: buildConversionTable(list, nil)}
	output := map[string]float64{"PhysicalMinBase": 5, "PhysicalMaxBase": 5}
	minDamage, maxDamage := calcDamage(skill, output, nil, nil, "", data.DamageTypeCold, 0, nil)
	if minDamage != 1 || maxDamage != 1 {
		t.Errorf("cold = %v to %v, want 1 to 1", minDamage, maxDamage)
	}
}
