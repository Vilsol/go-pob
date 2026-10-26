package calculator2

import (
	"math"

	"github.com/Vilsol/go-pob/mod"
)

type nonDamagingAilmentEntry struct {
	name, condition string
	max, precision  float64
	mods            func(num float64) []mod.Mod
}

// The only active skill with skills off is the default MeleeUnarmedPlayer, whose skill modLists read through to
// the player modDB; with no skill ported, they are read as the modDB with no skillCfg, and HasBonechill stays
// unset. The oracle builds the breakdown, so its stack mods always apply.
//
//pob2:block:src/Modules/CalcPerform.lua:3273-3316
//pob2:block:src/Modules/CalcPerform.lua:3327-3380
func applyNonDamagingAilments(env *Environment) {
	modDB := env.ModDB
	enemyDB := env.Enemy.ModDB
	output := env.Player.Output

	stacksMax := func(name string) float64 {
		return overrideOr(modDB, name, modDB.Sum(mod.TypeBase, nil, name))
	}
	ailments := []nonDamagingAilmentEntry{
		{
			name: "Chill", condition: "Chilled", max: gameConstant("ChillMaxEffect"), precision: 0,
			mods: func(num float64) []mod.Mod {
				var mods []mod.Mod
				if modDB.Flag(nil, "ChillCanStack") {
					output["CurrentChill"] = num * math.Min(enemyDB.Sum(mod.TypeBase, nil, "Multiplier:ChillStacks"), stacksMax("ChillStacksMax"))
					mods = append(mods, newMod("ActionSpeed", mod.TypeIncrease, -num, "Chill Stacks", mod.Condition("Chilled"), mod.Multiplier("ShockStacks").Limit(stacksMax("ShockStacksMax"))))
				} else {
					mods = append(mods, newMod("ActionSpeed", mod.TypeIncrease, -num, "Chill", mod.Condition("Chilled")))
				}
				if modDB.Flag(nil, "ChillEffectIncDamageTaken") {
					mods = append(mods, newMod("DamageTaken", mod.TypeIncrease, num, "Asphyxia's Wrath", mod.Condition("Chilled")))
				}
				return mods
			},
		},
		{
			name: "Shock", condition: "Shocked", max: 100, precision: 0,
			mods: func(num float64) []mod.Mod {
				var mods []mod.Mod
				if modDB.Flag(nil, "ShockCanStack") {
					output["ShockStackCount"] = math.Min(enemyDB.Sum(mod.TypeBase, nil, "Multiplier:ShockStacks"), stacksMax("ShockStacksMax"))
					mods = append(mods, newMod("DamageTaken", mod.TypeIncrease, num, "Shock", mod.Condition("Shocked"), mod.Multiplier("ShockStacks").Limit(stacksMax("ShockStacksMax"))))
					output["CurrentShock"] = num * math.Min(enemyDB.Sum(mod.TypeBase, nil, "Multiplier:ShockStacks"), stacksMax("ShockStacksMax"))
					mods = append(mods, newMod("DamageTakenByShock", mod.TypeIncrease, num, "Shock Stacks", mod.Condition("Shocked"), mod.Multiplier("ShockStacks").Limit(stacksMax("ShockStacksMax"))))
				} else {
					mods = append(mods, newMod("DamageTaken", mod.TypeIncrease, num, "Shock", mod.Condition("Shocked")))
				}
				return mods
			},
		},
	}

	for _, val := range ailments {
		ailment := val.name
		// modDB:Sum is 0 when nothing applies, and 0 is truthy in Lua, so this gate only checks the Already condition.
		if !enemyDB.Flag(nil, "Condition:Already"+val.condition) {
			override := 0.0
			minimum := 0.0
			for _, value := range modDB.Tabulate(mod.TypeBase, nil, ailment+"Base", ailment+"Override", ailment+"Minimum") {
				m := value.Mod
				effect := m.Value().Float()
				if m.Name() == ailment+"Override" {
					enemyDB.AddMod(newFlagMod("Condition:"+val.condition, m.GetSource()))
				}
				if m.Name() == ailment+"Base" || m.Name() == ailment+"Minimum" {
					effect = effect * calcLibMod(modDB, nil, "Enemy"+ailment+"Magnitude", "AilmentMagnitude") * calcLibMod(enemyDB, nil, "Self"+ailment+"Magnitude", "AilmentMagnitude")
					c := m.Clone().SetName(ailment + "Override")
					c.SetValue(mod.NewModValueFloat(effect))
					modDB.AddMod(c)
					if m.Name() == ailment+"Minimum" {
						minimum += effect
					}
				}
				override = math.Max(math.Max(override, effect), minimum)
			}
			maxAilment := 0.0
			if o := modDB.Override(nil, ailment+"Max"); o != nil {
				maxAilment = o.Float()
			} else {
				skillMax := val.max + modDB.Sum(mod.TypeBase, nil, ailment+"Max")
				if skillMax > maxAilment {
					maxAilment = skillMax
				}
			}
			output["Maximum"+ailment] = maxAilment
			scale := math.Pow(10, val.precision)
			output["Current"+ailment] = math.Floor(math.Min(math.Max(override, enemyDB.Sum(mod.TypeBase, nil, ailment+"Val")), output["Maximum"+ailment])*scale) / scale
			for _, m := range val.mods(output["Current"+ailment]) {
				enemyDB.AddMod(m)
			}
			// The Lua passes the condition tag as the mod's source, so the flag carries no tag.
			enemyDB.AddMod(newFlagMod("Condition:Already"+val.condition, ""))
		}
	}

	// Update chill and shock multipliers
	if chillEffectMultiplier := enemyDB.Sum(mod.TypeBase, nil, "Multiplier:ChillEffect"); chillEffectMultiplier < output["CurrentChill"] {
		enemyDB.AddMod(newMod("Multiplier:ChillEffect", mod.TypeBase, output["CurrentChill"]-chillEffectMultiplier, ""))
	}
	if shockEffectMultiplier := enemyDB.Sum(mod.TypeBase, nil, "Multiplier:ShockEffect"); shockEffectMultiplier < output["CurrentShock"] {
		enemyDB.AddMod(newMod("Multiplier:ShockEffect", mod.TypeBase, output["CurrentShock"]-shockEffectMultiplier, ""))
	}
}
