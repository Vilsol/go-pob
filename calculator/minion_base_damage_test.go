package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

func minionSyntheticWeapon(t *testing.T, env *Environment, minionType string) *ActorMinion {
	t.Helper()
	for _, as := range env.Player.ActiveSkillList {
		m := as.Minion
		if m == nil || m.Type != minionType || m.MinionData == nil || m.WeaponData1 == nil {
			continue
		}
		if m.WeaponData1.CritChance == 5 && m.WeaponData1.Range == m.MinionData.AttackRange {
			return m
		}
	}
	t.Fatalf("premise broken: no %s with a synthetic weaponData1 on this build", minionType)
	return nil
}

// TestMinionBaseDamageIgnoresAttackSpeed pins CalcActiveSkill.lua:722-724.
//
//	local damage = damageTable[minion.level] * minion.minionData.damage
//	if not minion.minionData.baseDamageIgnoresAttackSpeed then
//	end
func TestMinionBaseDamageIgnoresAttackSpeed(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0007-elementalist-storm-burst-of-repulsion.xml")
	m := minionSyntheticWeapon(t, env, "Metadata/Monsters/LeagueAzmeri/SpecialCorpses/KudukuHigh")
	md := m.MinionData

	if !md.BaseDamageIgnoresAttackSpeed {
		t.Fatal("premise broken: this fixture's minion no longer carries baseDamageIgnoresAttackSpeed")
	}
	if md.AttackTime == 1 {
		t.Fatal("premise broken: attackTime is 1, so the guard multiplies by the identity and " +
			"a fixture on this minion cannot tell the two implementations apart")
	}

	want := utils.RoundTo(data.MonsterDamageTable[m.Level]*md.Damage*(1+md.DamageSpread), 0)
	if got := m.WeaponData1.PhysicalMax; math.Abs(got-want) > 0.5 {
		t.Errorf("PhysicalMax = %v, want %v (damageTable[%d] * damage, with NO attackTime factor; "+
			"multiplying by attackTime %v gives %v)", got, want, m.Level, md.AttackTime,
			utils.RoundTo(data.MonsterDamageTable[m.Level]*md.Damage*md.AttackTime*(1+md.DamageSpread), 0))
	}
}

// TestMinionDamageFixupIsTwoMods pins CalcActiveSkill.lua:486-491.
//
//	skillModList:NewMod("Damage", "MORE", -100 * damageFixup, "Damage Fixup", ModFlag.Attack)
//	skillModList:NewMod("Speed",  "MORE",  100 * damageFixup, "Damage Fixup", ModFlag.Attack)
func TestMinionDamageFixupIsTwoMods(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0006-elementalist-storm-burst.xml")
	m := minionSyntheticWeapon(t, env, "SummonedFlameGolem")
	md := m.MinionData

	if md.DamageFixup == 0 {
		t.Fatal("premise broken: this fixture's minion no longer carries damageFixup")
	}

	want := 1 / md.AttackTime
	if got := m.WeaponData1.AttackRate; math.Abs(got-want) > 1e-9 {
		t.Errorf("AttackRate = %v, want %v (1 / the RAW attackTime); baking damageFixup into the "+
			"attack time gives %v", got, want, 1/(md.AttackTime*(1-md.DamageFixup)))
	}

	found := 0
	for _, as := range m.ActiveSkillList {
		if as.SkillModList == nil {
			continue
		}
		for _, sm := range as.SkillModList.Mods() {
			if sm.GetSource() == mod.Source("Damage Fixup") {
				found++
			}
		}
	}
	if len(m.ActiveSkillList) == 0 {
		t.Fatal("premise broken: the golem has no active skills, so there is no mod list to check")
	}
	if found == 0 {
		t.Error("no mod sourced \"Damage Fixup\" on any minion-actor skill; " +
			"CalcActiveSkill.lua:486-491 is not running")
	}
}

// TestMinionAllyDamageTableIsTheDefault pins CalcActiveSkill.lua:720-721.
//
//	local damageTable = (isSpectre or minion.minionData.hostile) and env.data.monsterDamageTable
//	local damage = damageTable[minion.level] * minion.minionData.damage
func TestMinionAllyDamageTableIsTheDefault(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0006-elementalist-storm-burst.xml")
	m := minionSyntheticWeapon(t, env, "SummonedFlameGolem")
	md := m.MinionData

	if md.Hostile {
		t.Fatal("premise broken: a hostile minion reads the OTHER table and cannot test this")
	}
	ally, hostile := data.MonsterAllyDamageTable[m.Level], data.MonsterDamageTable[m.Level]
	if ally == hostile {
		t.Fatalf("premise broken: the two damage tables agree at level %d (%v), so this fixture "+
			"cannot separate them", m.Level, ally)
	}

	base := ally * md.Damage
	if !md.BaseDamageIgnoresAttackSpeed {
		base *= md.AttackTime
	}
	want := utils.RoundTo(base*(1+md.DamageSpread), 0)
	if got := m.WeaponData1.PhysicalMax; math.Abs(got-want) > 0.5 {
		t.Errorf("PhysicalMax = %v, want %v (monsterAllyDamageTable[%d] = %v); the HOSTILE table "+
			"reads %v there and would give %v", got, want, m.Level, ally, hostile,
			utils.RoundTo(hostile*md.Damage*md.AttackTime*(1+md.DamageSpread), 0))
	}
}
