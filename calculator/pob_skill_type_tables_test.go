package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
)

// TestSupportExcludeTypesComeFromThePin pins the consumer of raw2.SkillExcludeSkillTypes.
func TestSupportExcludeTypesComeFromThePin(t *testing.T) {
	requireGameData(t, "3.29")

	ruthless := &GrantedEffect{Raw: poe.GrantedEffectByID("SupportRuthless")}
	if ruthless.Raw == nil {
		t.Fatal("SupportRuthless missing from go-pob-data")
	}
	active := func(types ...data.SkillType) *ActiveSkill {
		st := map[data.SkillType]bool{}
		for _, ty := range types {
			st[ty] = true
		}
		return &ActiveSkill{
			ActiveEffect: &GemEffect{GrantedEffect: &GrantedEffect{Raw: poe.GrantedEffectByID("Cleave")}},
			SkillTypes:   st,
		}
	}
	if !CalcCanGrantedEffectSupportActiveSkill(ruthless, active(data.SkillTypeAttack, data.SkillTypeMelee, data.SkillTypeMeleeSingleTarget), false) {
		t.Fatal("CONTROL: Ruthless must support an untriggered melee single-target attack")
	}
	if CalcCanGrantedEffectSupportActiveSkill(ruthless, active(data.SkillTypeAttack, data.SkillTypeMelee, data.SkillTypeMeleeSingleTarget, data.SkillTypeTriggered), false) {
		t.Error("Ruthless supported a Triggered skill; PoB's excludeSkillTypes carries Triggered")
	}
}

// TestActiveSkillTypesComeFromThePin pins grantedEffectSkillTypes.
func TestActiveSkillTypesComeFromThePin(t *testing.T) {
	requireGameData(t, "3.29")
	types := grantedEffectSkillTypes(poe.GrantedEffectByID("GSAncestralDruidFlaskExplode"))
	if !types[data.SkillTypeProjectile] {
		t.Errorf("GSAncestralDruidFlaskExplode types %v lack Projectile; PoB declares it", types)
	}
}

// TestAnimateGuardiansWeaponMinionRefusesRuthless is the corpus effect of the exclude fix.
func TestAnimateGuardiansWeaponMinionRefusesRuthless(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1262-necromancer-tempest-shield-3.xml")
	if env.Minion == nil {
		t.Fatal("no minion")
	}
	if got := env.Minion.OutputTable[OutTableMainHand]["RuthlessBlowChance"]; got != 0 {
		t.Errorf("Minion.MainHand.RuthlessBlowChance = %v, PoB 0", got)
	}
}
