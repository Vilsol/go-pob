package poe2skills

import (
	"errors"
	"fmt"
	"math"
)

// PreDamageSkill is what a preDamageFunc reads and writes of its activeSkill. A false ok is Lua's nil.
type PreDamageSkill interface {
	// SkillData is activeSkill.skillData[key].
	SkillData(key string) (float64, bool)
	// SetSkillData is activeSkill.skillData[key] = v, or nil when ok is false.
	SetSkillData(key string, v float64, ok bool)
	// Sum is activeSkill.skillModList:Sum(modType, activeSkill.skillCfg, name).
	Sum(modType, name string) float64
	// NewMod is activeSkill.skillModList:NewMod(name, modType, value, source); "" is no source.
	NewMod(name, modType string, value any, source string)
	// StatSetIndex is (activeSkill.activeEffect.statSet or activeSkill.activeEffect.statSetCalcs).index.
	StatSetIndex() int
	// ExplodeMods is activeSkill.skillModList:List(activeSkill.skillCfg, "ExplodeMod").
	ExplodeMods() []ExplodeMod
	// TabulateExplodeMods is activeSkill.skillModList:Tabulate("LIST", activeSkill.skillCfg, "ExplodeMod").
	TabulateExplodeMods() []TabulatedExplodeMod
	// ExplodeSource is activeSkill.activeEffect.srcInstance.explodeSource: its modSource (ok false when
	// nil) and its id.
	ExplodeSource() (modSource string, ok bool, id string)
}

// PreDamageOutput is the output table a preDamageFunc reads and writes.
type PreDamageOutput interface {
	Get(key string) (float64, bool)
	Set(key string, v float64)
}

// ExplodeMod is an ExplodeMod list value: { type, amount, value }.
type ExplodeMod struct {
	Type   string
	Amount float64
	Value  float64
}

// TabulatedExplodeMod is one Tabulate row: the value and its mod's source.
type TabulatedExplodeMod struct {
	Source string
	Value  ExplodeMod
}

// ErrNilArithmetic is the Lua error a preDamageFunc raises doing arithmetic on a nil.
var ErrNilArithmetic = errors.New("attempt to perform arithmetic on a nil value")

// PreDamageFunc is grantedEffect.preDamageFunc(activeSkill, output).
type PreDamageFunc func(activeSkill PreDamageSkill, output PreDamageOutput) error

// PreDamageFunc is the effect's preDamageFunc, nil when it has none.
func (ge *GrantedEffect) PreDamageFunc() PreDamageFunc {
	return preDamageFuncs[ge.ID]
}

func copyChannelPercent(activeSkill PreDamageSkill) error {
	v, ok := activeSkill.SkillData("channelPercentOfAttackTime")
	activeSkill.SetSkillData("hitTimeMultiplier", v, ok)
	activeSkill.SetSkillData("channelTimeMultiplier", v, ok)
	return nil
}

var preDamageFuncs = map[string]PreDamageFunc{
	"DetonatingArrowPlayer":    detonatingArrowPreDamage,
	"SnipePlayer":              snipePreDamage,
	"BonestormPlayer":          bonestormPreDamage,
	"FlameblastPlayer":         flameblastPreDamage,
	"FlickerStrikePlayer":      flickerStrikePreDamage,
	"GatheringStormPlayer":     gatheringStormPreDamage,
	"IncineratePlayer":         incineratePreDamage,
	"PlasmaBlastPlayer":        plasmaBlastPreDamage,
	"PerfectStrikePlayer":      perfectStrikePreDamage,
	"ShieldChargePlayer":       shieldChargePreDamage,
	"SuperchargedSlamPlayer":   superchargedSlamPreDamage,
	"MinionInstability":        minionInstabilityPreDamage,
	"TemperWeaponPlayer":       temperWeaponPreDamage,
	"EnemyExplode":             enemyExplodePreDamage,
	"ChaosFrogExplosionPlayer": chaosFrogExplosionPreDamage,
}

//pob2:src/Data/Skills/act_dex.lua:1186-1189
func detonatingArrowPreDamage(activeSkill PreDamageSkill, _ PreDamageOutput) error {
	activeSkill.SetSkillData("hitTimeMultiplier", activeSkill.Sum("BASE", "Multiplier:DetonatingArrowStage"), true)
	activeSkill.SetSkillData("channelTimeMultiplier", activeSkill.Sum("BASE", "Multiplier:DetonatingArrowStage"), true)
	return nil
}

//pob2:src/Data/Skills/act_dex.lua:7276-7279
func snipePreDamage(activeSkill PreDamageSkill, _ PreDamageOutput) error {
	return copyChannelPercent(activeSkill)
}

// `/ output.ProjectileCount or 1` divides first, so the `or 1` never applies.
//
//pob2:src/Data/Skills/act_int.lua:1863-1867
func bonestormPreDamage(activeSkill PreDamageSkill, output PreDamageOutput) error {
	projectileCount, ok := output.Get("ProjectileCount")
	if !ok {
		return fmt.Errorf("output.ProjectileCount: %w", ErrNilArithmetic)
	}
	activeSkill.SetSkillData("hitTimeMultiplier", math.Ceil(activeSkill.Sum("BASE", "Multiplier:BonestormStage")/projectileCount), true)
	activeSkill.SetSkillData("channelTimeMultiplier", math.Ceil(activeSkill.Sum("BASE", "Multiplier:BonestormStage")/projectileCount), true)
	activeSkill.SetSkillData("dpsMultiplier", activeSkill.Sum("BASE", "Multiplier:BonestormStage"), true)
	return nil
}

//pob2:src/Data/Skills/act_int.lua:8517-8519
func flameblastPreDamage(activeSkill PreDamageSkill, _ PreDamageOutput) error {
	activeSkill.SetSkillData("channelTimeMultiplier", activeSkill.Sum("BASE", "Multiplier:FlameblastStage"), true)
	return nil
}

//pob2:src/Data/Skills/act_int.lua:8672-8675
func flickerStrikePreDamage(activeSkill PreDamageSkill, _ PreDamageOutput) error {
	strikesPerCharge := activeSkill.Sum("BASE", "AdditionalFlickersPerPowerCharge")
	activeSkill.SetSkillData("averageBurstHits", 1+strikesPerCharge, true)
	return nil
}

//pob2:src/Data/Skills/act_int.lua:10496-10499
func gatheringStormPreDamage(activeSkill PreDamageSkill, _ PreDamageOutput) error {
	return copyChannelPercent(activeSkill)
}

//pob2:src/Data/Skills/act_int.lua:13623-13625
func incineratePreDamage(activeSkill PreDamageSkill, _ PreDamageOutput) error {
	activeSkill.SetSkillData("channelTimeMultiplier", activeSkill.Sum("BASE", "Multiplier:IncinerateStage"), true)
	return nil
}

//pob2:src/Data/Skills/act_str.lua:11638-11640
func plasmaBlastPreDamage(activeSkill PreDamageSkill, _ PreDamageOutput) error {
	activeSkill.SetSkillData("channelTimeMultiplier", 1, true)
	return nil
}

//pob2:src/Data/Skills/act_str.lua:15069-15072
func perfectStrikePreDamage(activeSkill PreDamageSkill, _ PreDamageOutput) error {
	return copyChannelPercent(activeSkill)
}

//pob2:src/Data/Skills/act_str.lua:17069-17072
func shieldChargePreDamage(activeSkill PreDamageSkill, _ PreDamageOutput) error {
	activeSkill.SetSkillData("hitTimeMultiplier", 1, true)
	activeSkill.SetSkillData("channelTimeMultiplier", 1, true)
	return nil
}

//pob2:src/Data/Skills/act_str.lua:18887-18890
func superchargedSlamPreDamage(activeSkill PreDamageSkill, _ PreDamageOutput) error {
	activeSkill.SetSkillData("hitTimeMultiplier", activeSkill.Sum("BASE", "Multiplier:SuperchargedSlamStage"), true)
	activeSkill.SetSkillData("channelTimeMultiplier", activeSkill.Sum("BASE", "Multiplier:SuperchargedSlamStage"), true)
	return nil
}

//pob2:src/Data/Skills/minion.lua:1108-1112
func minionInstabilityPreDamage(activeSkill PreDamageSkill, output PreDamageOutput) error {
	life, ok := output.Get("Life")
	mult, okMult := activeSkill.SkillData("selfFireExplosionLifeMultiplier")
	if !ok || !okMult {
		return fmt.Errorf("output.Life * skillData.selfFireExplosionLifeMultiplier: %w", ErrNilArithmetic)
	}
	activeSkill.SetSkillData("FireBonusMin", life*mult, true)
	activeSkill.SetSkillData("FireBonusMax", life*mult, true)
	return nil
}

//pob2:src/Data/Skills/other.lua:18232-18234
func temperWeaponPreDamage(activeSkill PreDamageSkill, _ PreDamageOutput) error {
	activeSkill.SetSkillData("channelTimeMultiplier", activeSkill.Sum("BASE", "Multiplier:TemperWeaponStage"), true)
	return nil
}

func addSkillData(activeSkill PreDamageSkill, key string, v float64) {
	cur, _ := activeSkill.SkillData(key)
	activeSkill.SetSkillData(key, cur+v, true)
}

// The per-type sums run over Lua tables with pairs, whose order PoB does not fix; they run here in list
// order.
//
//pob2:src/Data/Skills/other.lua:21146-21195
func enemyExplodePreDamage(activeSkill PreDamageSkill, output PreDamageOutput) error {
	output.Set("ExplodeChance", 0)
	index := activeSkill.StatSetIndex()
	if index != 3 {
		for _, explodeMod := range activeSkill.TabulateExplodeMods() {
			modSource, ok, id := activeSkill.ExplodeSource()
			activeEffectSource := modSource
			if !ok {
				activeEffectSource = "Tree:" + id
			}
			if explodeMod.Source == activeEffectSource {
				v := explodeMod.Value
				if v.Type == "RandomElement" {
					activeSkill.SetSkillData("FireEffectiveExplodePercentage", v.Amount/3, true)
					activeSkill.SetSkillData("ColdEffectiveExplodePercentage", v.Amount/3, true)
					activeSkill.SetSkillData("LightningEffectiveExplodePercentage", v.Amount/3, true)
				} else {
					activeSkill.SetSkillData(v.Type+"EffectiveExplodePercentage", v.Amount, true)
				}
				if index == 2 {
					output.Set("ExplodeChance", 1)
				} else {
					output.Set("ExplodeChance", v.Value/100)
				}
			}
		}
	} else {
		type amountChance struct {
			chance  map[float64]float64
			amounts []float64
		}
		var types []string
		typeAmountChances := map[string]*amountChance{}
		for _, explodeMod := range activeSkill.ExplodeMods() {
			ac := typeAmountChances[explodeMod.Type]
			if ac == nil {
				ac = &amountChance{chance: map[float64]float64{}}
				typeAmountChances[explodeMod.Type] = ac
				types = append(types, explodeMod.Type)
			}
			if _, seen := ac.chance[explodeMod.Amount]; !seen {
				ac.amounts = append(ac.amounts, explodeMod.Amount)
			}
			ac.chance[explodeMod.Amount] += explodeMod.Value / 100
		}
		for _, typ := range types {
			ac := typeAmountChances[typ]
			physExplodeChance := 0.0
			for _, amount := range ac.amounts {
				chance := ac.chance[amount]
				amountXChance := amount * chance
				if typ == "RandomElement" {
					addSkillData(activeSkill, "FireEffectiveExplodePercentage", amountXChance/3)
					addSkillData(activeSkill, "ColdEffectiveExplodePercentage", amountXChance/3)
					addSkillData(activeSkill, "LightningEffectiveExplodePercentage", amountXChance/3)
				} else {
					addSkillData(activeSkill, typ+"EffectiveExplodePercentage", amountXChance)
				}
				if typ == "Physical" {
					physExplodeChance = 1 - float64((1-physExplodeChance)*(1-chance))
				}
				explodeChance, _ := output.Get("ExplodeChance")
				output.Set("ExplodeChance", 1-float64((1-explodeChance)*(1-chance)))
			}
			if typ == "Physical" && physExplodeChance != 0 {
				activeSkill.NewMod("CalcArmourAsThoughDealing", "MORE", 100/math.Min(physExplodeChance, 1)-100, "")
			}
		}
	}
	explodeChance, _ := output.Get("ExplodeChance")
	output.Set("ExplodeChance", math.Min(explodeChance*100, 100))
	return nil
}

//pob2:src/Data/Skills/sup_int.lua:3184-3193
func chaosFrogExplosionPreDamage(activeSkill PreDamageSkill, _ PreDamageOutput) error {
	interval := activeSkill.Sum("BASE", "ToadSpawnInterval")
	if interval == 0 {
		// An inactive support cannot spawn toads or deal damage, including ailments.
		activeSkill.SetSkillData("hitTimeMultiplier", math.Inf(1), true)
		activeSkill.NewMod("DealNoDamage", "FLAG", true, "Doedre's Undoing")
	} else {
		activeSkill.SetSkillData("hitTimeMultiplier", interval/1000, true)
	}
	return nil
}
