package raw

import (
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

func skill(dataKey string, dataValue float64) mod.Mod {
	return mod.NewList("SkillData", &mod.SkillData{
		Key:   dataKey,
		Value: dataValue,
	})
}

var SkillStatMap = map[string]*StatMap{
	//
	// Skill data modifiers
	//
	"base_skill_effect_duration": {
		Mods: []mod.Mod{skill("duration", 0)},
		Div:  utils.Ptr(float64(1000)),
	},
	"base_secondary_skill_effect_duration": {
		Mods: []mod.Mod{skill("durationSecondary", 0)},
		Div:  utils.Ptr(float64(1000)),
	},
	"spell_minimum_base_physical_damage": {
		Mods: []mod.Mod{skill("PhysicalMin", 0)},
	},
	"secondary_minimum_base_physical_damage": {
		Mods: []mod.Mod{skill("PhysicalMin", 0)},
	},
	"spell_maximum_base_physical_damage": {
		Mods: []mod.Mod{skill("PhysicalMax", 0)},
	},
	"secondary_maximum_base_physical_damage": {
		Mods: []mod.Mod{skill("PhysicalMax", 0)},
	},
	"spell_minimum_base_lightning_damage": {
		Mods: []mod.Mod{skill("LightningMin", 0)},
	},
	"secondary_minimum_base_lightning_damage": {
		Mods: []mod.Mod{skill("LightningMin", 0)},
	},
	"spell_maximum_base_lightning_damage": {
		Mods: []mod.Mod{skill("LightningMax", 0)},
	},
	"secondary_maximum_base_lightning_damage": {
		Mods: []mod.Mod{skill("LightningMax", 0)},
	},
	"spell_minimum_base_cold_damage": {
		Mods: []mod.Mod{skill("ColdMin", 0)},
	},
	"secondary_minimum_base_cold_damage": {
		Mods: []mod.Mod{skill("ColdMin", 0)},
	},
	"spell_maximum_base_cold_damage": {
		Mods: []mod.Mod{skill("ColdMax", 0)},
	},
	"secondary_maximum_base_cold_damage": {
		Mods: []mod.Mod{skill("ColdMax", 0)},
	},
	"spell_minimum_base_fire_damage": {
		Mods: []mod.Mod{skill("FireMin", 0)},
	},
	"secondary_minimum_base_fire_damage": {
		Mods: []mod.Mod{skill("FireMin", 0)},
	},
	"spell_maximum_base_fire_damage": {
		Mods: []mod.Mod{skill("FireMax", 0)},
	},
	"secondary_maximum_base_fire_damage": {
		Mods: []mod.Mod{skill("FireMax", 0)},
	},
	"spell_minimum_base_chaos_damage": {
		Mods: []mod.Mod{skill("ChaosMin", 0)},
	},
	"secondary_minimum_base_chaos_damage": {
		Mods: []mod.Mod{skill("ChaosMin", 0)},
	},
	"spell_maximum_base_chaos_damage": {
		Mods: []mod.Mod{skill("ChaosMax", 0)},
	},
	"secondary_maximum_base_chaos_damage": {
		Mods: []mod.Mod{skill("ChaosMax", 0)},
	},
	"spell_minimum_base_lightning_damage_per_removable_power_charge": {
		Mods: []mod.Mod{skill("LightningMin", 0).Tag(mod.Multiplier("RemovablePowerCharge"))},
	},
	"spell_maximum_base_lightning_damage_per_removable_power_charge": {
		Mods: []mod.Mod{skill("LightningMax", 0).Tag(mod.Multiplier("RemovablePowerCharge"))},
	},
	"spell_minimum_base_fire_damage_per_removable_endurance_charge": {
		Mods: []mod.Mod{skill("FireMin", 0).Tag(mod.Multiplier("RemovableEnduranceCharge"))},
	},
	"spell_maximum_base_fire_damage_per_removable_endurance_charge": {
		Mods: []mod.Mod{skill("FireMax", 0).Tag(mod.Multiplier("RemovableEnduranceCharge"))},
	},
	"spell_minimum_base_cold_damage_per_removable_frenzy_charge": {
		Mods: []mod.Mod{skill("ColdMin", 0).Tag(mod.Multiplier("RemovableFrenzyCharge"))},
	},
	"spell_maximum_base_cold_damage_per_removable_frenzy_charge": {
		Mods: []mod.Mod{skill("ColdMax", 0).Tag(mod.Multiplier("RemovableFrenzyCharge"))},
	},
	"spell_minimum_base_cold_damage_+_per_10_intelligence": {
		Mods: []mod.Mod{skill("ColdMin", 0).Tag(mod.PerStat(10, "Int"))},
	},
	"spell_maximum_base_cold_damage_+_per_10_intelligence": {
		Mods: []mod.Mod{skill("ColdMax", 0).Tag(mod.PerStat(10, "Int"))},
	},
	"base_cold_damage_to_deal_per_minute": {
		Mods: []mod.Mod{skill("ColdDot", 0)},
		Div:  utils.Ptr(float64(60)),
	},
	"base_fire_damage_to_deal_per_minute": {
		Mods: []mod.Mod{skill("FireDot", 0)},
		Div:  utils.Ptr(float64(60)),
	},
	"base_chaos_damage_to_deal_per_minute": {
		Mods: []mod.Mod{skill("ChaosDot", 0)},
		Div:  utils.Ptr(float64(60)),
	},
	"base_physical_damage_to_deal_per_minute": {
		Mods: []mod.Mod{skill("PhysicalDot", 0)},
		Div:  utils.Ptr(float64(60)),
	},
	"critical_ailment_dot_multiplier_+": {
		Mods: []mod.Mod{mod.NewFloat("DotMultiplier", "BASE", 0).Tag(mod.Condition("CriticalStrike"))},
	},
	"base_skill_show_average_damage_instead_of_dps": {
		Mods: []mod.Mod{skill("showAverage", 1)}, // TODO Might be 'true'
	},
	"cast_time_overrides_attack_duration": {
		Mods: []mod.Mod{skill("castTimeOverridesAttackTime", 1)}, // TODO Might be 'true'
	},
	"spell_cast_time_cannot_be_modified": {
		Mods: []mod.Mod{skill("fixedCastTime", 1)}, // TODO Might be 'true'
	},
	"global_always_hit": {
		Mods: []mod.Mod{skill("cannotBeEvaded", 1)}, // TODO Might be 'true'
	},
	"bleed_duration_is_skill_duration": {
		Mods: []mod.Mod{skill("bleedDurationIsSkillDuration", 1)}, // TODO Might be 'true'
	},
	"poison_duration_is_skill_duration": {
		Mods: []mod.Mod{skill("poisonDurationIsSkillDuration", 1)}, // TODO Might be 'true'
	},
	"spell_damage_modifiers_apply_to_skill_dot": {
		Mods: []mod.Mod{skill("dotIsSpell", 1)}, // TODO Might be 'true'
	},
	"projectile_damage_modifiers_apply_to_skill_dot": {
		Mods: []mod.Mod{skill("dotIsProjectile", 1)}, // TODO Might be 'true'
	},
	"additive_mine_duration_modifiers_apply_to_buff_effect_duration": {
		Mods: []mod.Mod{skill("mineDurationAppliesToSkill", 1)}, // TODO Might be 'true'
	},
	"additive_arrow_speed_modifiers_apply_to_area_of_effect": {
		Mods: []mod.Mod{skill("arrowSpeedAppliesToAreaOfEffect", 1)}, // TODO Might be 'true'
	},
	"skill_buff_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("BuffEffect", "INC", 0)},
	},
	"base_skill_reserve_life_instead_of_mana": {
		Mods: []mod.Mod{mod.NewFlag("BloodMagicReserved", true)},
	},
	"base_skill_cost_life_instead_of_mana": {
		Mods: []mod.Mod{mod.NewFlag("CostLifeInsteadOfMana", true)},
	},
	"base_active_skill_totem_level": {
		Mods: []mod.Mod{skill("totemLevel", 0)},
	},
	"totem_support_gem_level": {
		Mods: []mod.Mod{skill("totemLevel", 0)},
	},
	"spell_uncastable_if_triggerable": {
		Mods: []mod.Mod{skill("triggered", 1).Tag(mod.SkillType("Triggerable"))},
	},
	"unique_mjolner_lightning_spells_triggered": {
		Mods: []mod.Mod{skill("triggeredByMjolner", 1).Tag(mod.SkillType("Triggerable"), mod.SkillType("Lightning"))},
	},
	"unique_cospris_malice_cold_spells_triggered": {
		Mods: []mod.Mod{skill("triggeredByCospris", 1).Tag(mod.SkillType("Triggerable"), mod.SkillType("Spell"), mod.SkillType("Cold"))},
	},
	"skill_has_trigger_from_unique_item": {
		Mods: []mod.Mod{skill("triggeredByUnique", 0).Tag(mod.SkillType("Triggerable"))},
	},
	"skill_triggered_when_you_focus_chance_%": {
		Mods: []mod.Mod{skill("triggeredByFocus", 0).Tag(mod.SkillType("Triggerable"), mod.SkillType("Spell"))},
		Div:  utils.Ptr(float64(100)),
	},
	"spell_has_trigger_from_crafted_item_mod": {
		Mods: []mod.Mod{skill("triggeredByCraft", 0).Tag(mod.SkillType("Triggerable"), mod.SkillType("Spell"))},
	},
	"support_cast_on_mana_spent": {
		Mods: []mod.Mod{skill("triggeredByManaSpent", 1).Tag(mod.SkillType("Triggerable"), mod.SkillType("Spell"))},
	},
	"display_mirage_warriors_no_spirit_strikes": {
		Mods: []mod.Mod{skill("triggeredBySaviour", 1).Tag(mod.SkillType("Attack"))},
	},
	"cast_spell_on_linked_attack_crit": {
		Mods: []mod.Mod{skill("triggeredByCoC", 1).Tag(mod.SkillType("Triggerable"), mod.SkillType("Spell"))},
	},
	"cast_linked_spells_on_attack_crit_%": {
		Mods: []mod.Mod{skill("chanceToTriggerOnCrit", 0).Tag(mod.SkillType("Attack"))},
	},
	"cast_spell_on_linked_melee_kill": {
		Mods: []mod.Mod{skill("triggeredByMeleeKill", 1).Tag(mod.SkillType("Triggerable"), mod.SkillType("Spell"))},
	},
	"cast_linked_spells_on_melee_kill_%": {
		Mods: []mod.Mod{skill("chanceToTriggerOnMeleeKill", 0).Tag(mod.SkillType("Attack"), mod.SkillType("Melee"))},
	},
	"cast_spell_while_linked_skill_channelling": {
		Mods: []mod.Mod{skill("triggeredWhileChannelling", 1).Tag(mod.SkillType("Triggerable"), mod.SkillType("Spell"))},
	},
	"skill_triggered_by_snipe": {
		Mods: []mod.Mod{skill("triggered", 1).Tag(mod.SkillType("Triggerable"))},
	},
	"triggered_by_spiritual_cry": {
		Mods: []mod.Mod{skill("triggeredByGeneralsCry", 1).Tag(mod.SkillType("Melee"), mod.SkillType("Attack"))},
	},
	"holy_relic_trigger_on_parent_attack_%": {
		Mods: []mod.Mod{skill("triggeredByParentAttack", 1).Tag(mod.SkillType("Triggerable"))},
	},
	"skill_can_own_mirage_archers": {
		Mods: []mod.Mod{skill("triggeredByMirageArcher", 1).Tag(mod.SkillType("MirageArcherCanUse"))},
	},
	"skill_double_hits_when_dual_wielding": {
		Mods: []mod.Mod{skill("doubleHitsWhenDualWielding", 1)},
	},
	"area_of_effect_+%_while_not_dual_wielding": {
		Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", "INC", 0).Tag(mod.Condition("DualWielding").Neg(true))},
	},
	"base_spell_repeat_count": {
		Mods: []mod.Mod{mod.NewFloat("RepeatCount", "BASE", 0)},
	},
	"base_melee_attack_repeat_count": {
		Mods: []mod.Mod{mod.NewFloat("RepeatCount", "BASE", 0)},
	},
	"display_minion_monster_level": {
		Mods: []mod.Mod{skill("minionLevel", 0)},
	},
	"display_skill_minions_level_is_corpse_level": {
		Mods: []mod.Mod{skill("minionLevelIsEnemyLevel", 1)},
	},
	"active_skill_minion_added_damage_+%_final": {
		Mods: []mod.Mod{skill("minionDamageEffectiveness", 0)},
	},
	"base_bleed_on_hit_still_%_of_physical_damage_to_deal_per_minute": {
		Mods: []mod.Mod{skill("bleedBasePercent", 0)},
		Div:  utils.Ptr(float64(60)),
	},
	"active_skill_base_radius_+": {
		Mods: []mod.Mod{skill("radiusExtra", 0)},
	},
	"corpse_explosion_monster_life_%": {
		Mods: []mod.Mod{skill("corpseExplosionLifeMultiplier", 0)},
		Div:  utils.Ptr(float64(100)),
	},
	"spell_base_fire_damage_%_maximum_life": {
		Mods: []mod.Mod{skill("selfFireExplosionLifeMultiplier", 0)},
		Div:  utils.Ptr(float64(100)),
	},
	// for some reason DeathWish adds another stat with same effect as above
	"skill_minion_explosion_life_%": {
		Mods: []mod.Mod{skill("selfFireExplosionLifeMultiplier", 0)},
		Div:  utils.Ptr(float64(100)),
	},
	"deal_chaos_damage_per_second_for_10_seconds_on_hit": {
		Mods: []mod.Mod{mod.NewList("SkillData", mod.SkillData{
			Key:   "decay",
			Value: 0,
			Merge: "MAX",
		})},
	},
	"base_spell_cast_time_ms_override": {
		Mods: []mod.Mod{skill("castTimeOverride", 0)},
		Div:  utils.Ptr(float64(1000)),
	},
	//
	// Defensive modifiers
	//
	"base_physical_damage_reduction_rating": {
		Mods: []mod.Mod{mod.NewFloat("Armour", "BASE", 0)},
	},
	"base_evasion_rating": {
		Mods: []mod.Mod{mod.NewFloat("Evasion", "BASE", 0)},
	},
	"base_maximum_energy_shield": {
		Mods: []mod.Mod{mod.NewFloat("EnergyShield", "BASE", 0)},
	},
	"base_fire_damage_resistance_%": {
		Mods: []mod.Mod{mod.NewFloat("FireResist", "BASE", 0)},
	},
	"base_cold_damage_resistance_%": {
		Mods: []mod.Mod{mod.NewFloat("ColdResist", "BASE", 0)},
	},
	"base_lightning_damage_resistance_%": {
		Mods: []mod.Mod{mod.NewFloat("LightningResist", "BASE", 0)},
	},
	"base_chaos_damage_resistance_%": {
		Mods: []mod.Mod{mod.NewFloat("ChaosResist", "BASE", 0)},
	},
	"base_resist_all_elements_%": {
		Mods: []mod.Mod{mod.NewFloat("ElementalResist", "BASE", 0)},
	},
	"base_maximum_fire_damage_resistance_%": {
		Mods: []mod.Mod{mod.NewFloat("FireResistMax", "BASE", 0)},
	},
	"base_maximum_cold_damage_resistance_%": {
		Mods: []mod.Mod{mod.NewFloat("ColdResistMax", "BASE", 0)},
	},
	"base_maximum_lightning_damage_resistance_%": {
		Mods: []mod.Mod{mod.NewFloat("LightningResistMax", "BASE", 0)},
	},
	"base_stun_recovery_+%": {
		Mods: []mod.Mod{mod.NewFloat("StunRecovery", "INC", 0)},
	},
	"base_life_gain_per_target": {
		Mods: []mod.Mod{mod.NewFloat("LifeOnHit", "BASE", 0).Flag(mod.MFlagAttack)},
	},
	"base_life_regeneration_rate_per_minute": {
		Mods: []mod.Mod{mod.NewFloat("LifeRegen", "BASE", 0)},
		Div:  utils.Ptr(float64(60)),
	},
	"life_regeneration_rate_per_minute_%": {
		Mods: []mod.Mod{mod.NewFloat("LifeRegenPercent", "BASE", 0)},
		Div:  utils.Ptr(float64(60)),
	},
	"base_mana_regeneration_rate_per_minute": {
		Mods: []mod.Mod{mod.NewFloat("ManaRegen", "BASE", 0)},
		Div:  utils.Ptr(float64(60)),
	},
	"energy_shield_recharge_rate_+%": {
		Mods: []mod.Mod{mod.NewFloat("EnergyShieldRecharge", "INC", 0)},
	},
	"base_mana_cost_-%": {
		Mods: []mod.Mod{mod.NewFloat("ManaCost", "INC", 0)},
		Mult: utils.Ptr(float64(-1)),
	},
	"base_mana_cost_+": {
		Mods: []mod.Mod{mod.NewFloat("ManaCostNoMult", "BASE", 0)},
	},
	"no_mana_cost": {
		Mods:  []mod.Mod{mod.NewFloat("ManaCost", "MORE", 0)},
		Value: utils.Ptr(float64(-1)),
	},
	"base_life_cost_+%": {
		Mods: []mod.Mod{mod.NewFloat("LifeCost", "INC", 0)},
	},
	"flask_mana_to_recover_+%": {
		Mods: []mod.Mod{mod.NewFloat("FlaskManaRecovery", "INC", 0)},
	},
	"base_chance_to_dodge_%": {
		Mods: []mod.Mod{mod.NewFloat("AttackDodgeChance", "BASE", 0)},
	},
	"base_chance_to_dodge_spells_%": {
		Mods: []mod.Mod{mod.NewFloat("SpellDodgeChance", "BASE", 0)},
	},
	"base_movement_velocity_+%": {
		Mods: []mod.Mod{mod.NewFloat("MovementSpeed", "INC", 0)},
	},
	"monster_base_block_%": {
		Mods: []mod.Mod{mod.NewFloat("BlockChance", "BASE", 0)},
	},
	"base_spell_block_%": {
		Mods: []mod.Mod{mod.NewFloat("SpellBlockChance", "BASE", 0)},
	},
	"life_leech_from_any_damage_permyriad": {
		Mods: []mod.Mod{mod.NewFloat("DamageLifeLeech", "BASE", 0)},
		Div:  utils.Ptr(float64(100)),
	},
	"mana_leech_from_any_damage_permyriad": {
		Mods: []mod.Mod{mod.NewFloat("DamageManaLeech", "BASE", 0)},
		Div:  utils.Ptr(float64(100)),
	},
	"attack_skill_mana_leech_from_any_damage_permyriad": {
		Mods: []mod.Mod{mod.NewFloat("DamageManaLeech", "BASE", 0).Flag(mod.MFlagAttack)},
		Div:  utils.Ptr(float64(100)),
	},
	"base_mana_leech_from_elemental_damage_permyriad": {
		Mods: []mod.Mod{mod.NewFloat("ElementalDamageManaLeech", "BASE", 0)},
		Div:  utils.Ptr(float64(100)),
	},
	"base_life_leech_from_attack_damage_permyriad": {
		Mods: []mod.Mod{mod.NewFloat("DamageLifeLeech", "BASE", 0).Flag(mod.MFlagAttack)},
		Div:  utils.Ptr(float64(100)),
	},
	"base_life_leech_from_chaos_damage_permyriad": {
		Mods: []mod.Mod{mod.NewFloat("ChaosDamageLifeLeech", "BASE", 0)},
		Div:  utils.Ptr(float64(100)),
	},
	"energy_shield_leech_from_any_damage_permyriad": {
		Mods: []mod.Mod{mod.NewFloat("DamageEnergyShieldLeech", "BASE", 0)},
		Div:  utils.Ptr(float64(100)),
	},
	"life_leech_from_physical_attack_damage_permyriad": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamageLifeLeech", "BASE", 0).Flag(mod.MFlagAttack)},
		Div:  utils.Ptr(float64(100)),
	},
	"base_energy_shield_leech_from_spell_damage_permyriad": {
		Mods: []mod.Mod{mod.NewFloat("DamageEnergyShieldLeech", "BASE", 0).Flag(mod.MFlagSpell)},
		Div:  utils.Ptr(float64(100)),
	},
	"mana_gain_per_target": {
		Mods: []mod.Mod{mod.NewFloat("ManaOnHit", "BASE", 0)},
	},
	"damage_+%_while_life_leeching": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.Condition("LeechingLife"))},
	},
	"damage_+%_while_mana_leeching": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.Condition("LeechingMana"))},
	},
	"damage_+%_while_es_leeching": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.Condition("LeechingEnergyShield"))},
	},
	"aura_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("AuraEffect", "INC", 0)},
	},
	"elusive_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("ElusiveEffect", "MAX", 0).Tag(mod.GlobalEffect("Buff"))},
	},
	"cannot_be_stunned_while_leeching": {
		Mods: []mod.Mod{mod.NewFloat("AvoidStun", "BASE", 100).Tag(mod.Condition("Leeching"))},
	},
	"base_avoid_stun_%": {
		Mods: []mod.Mod{mod.NewFloat("AvoidStun", "BASE", 0)},
	},
	"base_immune_to_shock": {
		Mods: []mod.Mod{mod.NewFloat("AvoidShock", "BASE", 100)},
	},
	"base_immune_to_chill": {
		Mods: []mod.Mod{mod.NewFloat("AvoidChill", "BASE", 100)},
	},
	"base_immune_to_freeze": {
		Mods: []mod.Mod{mod.NewFloat("AvoidFreeze", "BASE", 100)},
	},
	"base_immune_to_ignite": {
		Mods: []mod.Mod{mod.NewFloat("AvoidIgnite", "BASE", 100)},
	},
	"avoid_interruption_while_using_this_skill_%": {
		Mods: []mod.Mod{mod.NewFloat("AvoidInterruptStun", "BASE", 0)},
	},
	"life_leech_does_not_stop_at_full_life": {
		Mods: []mod.Mod{mod.NewFlag("CanLeechLifeOnFullLife", true)},
	},
	//
	// Offensive modifiers
	//
	// Speed
	"attack_and_cast_speed_+%": {
		Mods: []mod.Mod{mod.NewFloat("Speed", "INC", 0)},
	},
	"cast_speed_+%_granted_from_skill": {
		Mods: []mod.Mod{mod.NewFloat("Speed", "INC", 0).Flag(mod.MFlagCast)},
	},
	"base_cooldown_speed_+%": {
		Mods: []mod.Mod{mod.NewFloat("CooldownRecovery", "INC", 0)},
	},
	"base_spell_cooldown_speed_+%": {
		Mods: []mod.Mod{mod.NewFloat("CooldownRecovery", "INC", 0)},
	},
	"additional_weapon_base_attack_time_ms": {
		Mods: []mod.Mod{mod.NewFloat("Speed", "BASE", 0).Flag(mod.MFlagAttack)},
		Div:  utils.Ptr(float64(1000)),
	},
	"warcry_speed_+%": {
		Mods: []mod.Mod{mod.NewFloat("WarcrySpeed", "INC", 0).KeywordFlag(mod.KeywordFlagWarcry)},
	},
	// AoE
	"base_skill_area_of_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", "INC", 0)},
	},
	"base_aura_area_of_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", "INC", 0).KeywordFlag(mod.KeywordFlagAura)},
	},
	"active_skill_area_of_effect_+%_final_when_cast_on_frostbolt": {
		Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", "MORE", 0).Tag(mod.Condition("CastOnFrostbolt"))},
	},
	"active_skill_area_of_effect_radius_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", "MORE", 0)},
	},
	"active_skill_area_of_effect_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", "MORE", 0)},
	},
	// Critical strikes
	"additional_base_critical_strike_chance": {
		Mods: []mod.Mod{mod.NewFloat("CritChance", "BASE", 0)},
		Div:  utils.Ptr(float64(100)),
	},
	"critical_strike_chance_+%": {
		Mods: []mod.Mod{mod.NewFloat("CritChance", "INC", 0)},
	},
	"spell_critical_strike_chance_+%": {
		Mods: []mod.Mod{mod.NewFloat("CritChance", "INC", 0).Flag(mod.MFlagSpell)},
	},
	"attack_critical_strike_chance_+%": {
		Mods: []mod.Mod{mod.NewFloat("CritChance", "INC", 0).Flag(mod.MFlagAttack)},
	},
	"base_critical_strike_multiplier_+": {
		Mods: []mod.Mod{mod.NewFloat("CritMultiplier", "BASE", 0)},
	},
	"critical_strike_chance_+%_vs_shocked_enemies": {
		Mods: []mod.Mod{mod.NewFloat("CritChance", "INC", 0).Tag(mod.ActorCondition("enemy", "shocked"))},
	},
	"critical_strike_chance_+%_per_power_charge": {
		Mods: []mod.Mod{mod.NewFloat("CritChance", "INC", 0).Tag(mod.Multiplier("PowerCharge"))},
	},
	"critical_strike_multiplier_+_per_power_charge": {
		Mods: []mod.Mod{mod.NewFloat("CritMultiplier", "BASE", 0).Tag(mod.Multiplier("PowerCharge"))},
	},
	"critical_multiplier_+%_per_100_max_es_on_shield": {
		Mods: []mod.Mod{mod.NewFloat("CritMultiplier", "BASE", 0).Tag(mod.PerStat(100, "EnergyShieldOnWeapon 2"))},
	},
	"damage_+%_per_endurance_charge": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.Multiplier("EnduranceCharge"))},
	},
	"damage_+%_per_frenzy_charge": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.Multiplier("FrenzyCharge"))},
	},
	"additional_critical_strike_chance_permyriad_while_affected_by_elusive": {
		Mods: []mod.Mod{mod.NewFloat("CritChance", "BASE", 0).Tag(mod.Condition("Elusive"), mod.Condition("UsingClaw", "UsingDagger"), mod.Condition("UsingSword", "UsingAxe", "UsingMace").Neg(true))},
		Div:  utils.Ptr(float64(100)),
	},
	"nightblade_elusive_grants_critical_strike_multiplier_+_to_supported_skills": {
		Mods: []mod.Mod{mod.NewFloat("NightbladeElusiveCritMultiplier", "BASE", 0).Tag(mod.Condition("UsingClaw", "UsingDagger"), mod.Condition("UsingSword", "UsingAxe", "UsingMace").Neg(true))},
	},
	"critical_strike_chance_against_enemies_on_full_life_+%": {
		Mods: []mod.Mod{mod.NewFloat("CritChance", "INC", 0).Tag(mod.ActorCondition("enemy", "FullLife"))},
	},
	"critical_strike_chance_+%_vs_blinded_enemies": {
		Mods: []mod.Mod{mod.NewFloat("CritChance", "INC", 0).Tag(mod.ActorCondition("enemy", "Blinded"))},
	},
	// Duration
	"buff_effect_duration_+%_per_removable_endurance_charge": {
		Mods: []mod.Mod{mod.NewFloat("Duration", "INC", 0).Tag(mod.Multiplier("RemovableEnduranceCharge"))},
	},
	"buff_effect_duration_+%_per_removable_endurance_charge_limited_to_5": {
		Mods: []mod.Mod{mod.NewFloat("Duration", "INC", 0).Tag(mod.Multiplier("RemovableEnduranceCharge").Limit(5))},
	},
	"skill_effect_duration_+%_per_removable_frenzy_charge": {
		Mods: []mod.Mod{mod.NewFloat("Duration", "INC", 0).Tag(mod.Multiplier("RemovableFrenzyCharge"))},
	},
	"skill_effect_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("Duration", "INC", 0)},
	},
	"secondary_skill_effect_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("SecondaryDuration", "INC", 0)},
	},
	"active_skill_quality_duration_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("Duration", "MORE", 0)},
	},
	"fortify_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("FortifyDuration", "INC", 0)},
	},
	"support_swift_affliction_skill_effect_and_damaging_ailment_duration_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("SkillAndDamagingAilmentDuration", "MORE", 0)},
	},
	"base_bleed_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("BleedDuration", "INC", 0)},
	},
	// Damage
	"damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0)},
	},
	"chance_for_extra_damage_roll_%": {
		Mods: []mod.Mod{mod.NewFloat("LuckyHitsChance", "BASE", 0)},
	},
	"chance_to_deal_double_damage_%": {
		Mods: []mod.Mod{mod.NewFloat("DoubleDamageChance", "BASE", 0)},
	},
	"base_chance_to_deal_triple_damage_%": {
		Mods: []mod.Mod{mod.NewFloat("TripleDamageChance", "BASE", 0)},
	},
	"damage_+%_with_hits_and_ailments": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment)},
	},
	"physical_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", "INC", 0)},
	},
	"lightning_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("LightningDamage", "INC", 0)},
	},
	"cold_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("ColdDamage", "INC", 0)},
	},
	"fire_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("FireDamage", "INC", 0)},
	},
	"chaos_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("ChaosDamage", "INC", 0)},
	},
	"elemental_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("ElementalDamage", "INC", 0)},
	},
	"damage_over_time_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Flag(mod.MFlagDot)},
	},
	"burn_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("FireDamage", "INC", 0).KeywordFlag(mod.KeywordFlagFireDot)},
	},
	"faster_burn_%": {
		Mods: []mod.Mod{mod.NewFloat("IgniteBurnFaster", "INC", 0)},
	},
	"faster_poison_%": {
		Mods: []mod.Mod{mod.NewFloat("PoisonFaster", "INC", 0)},
	},
	"active_skill_damage_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "MORE", 0)},
	},
	"sigil_attached_target_hit_damage_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "MORE", 0).Flag(mod.MFlagHit)},
	},
	"melee_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Flag(mod.MFlagMelee)},
	},
	"melee_physical_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", "INC", 0).Flag(mod.MFlagMelee)},
	},
	"area_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Flag(mod.MFlagArea)},
	},
	"projectile_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Flag(mod.MFlagProjectile)},
	},
	"active_skill_projectile_damage_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "MORE", 0).Flag(mod.MFlagProjectile)},
	},
	"active_skill_area_damage_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "MORE", 0).Flag(mod.MFlagArea)},
	},
	"physical_damage_+%_per_frenzy_charge": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", "INC", 0).Tag(mod.Multiplier("FrenzyCharge"))},
	},
	"melee_damage_vs_bleeding_enemies_+%": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamage", "INC", 0).Flag(mod.MFlagMelee).Tag(mod.ActorCondition("enemy", "Bleeding"))},
	},
	"damage_+%_vs_frozen_enemies": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Flag(mod.MFlagHit).Tag(mod.ActorCondition("enemy", "Frozen"))},
	},
	"base_reduce_enemy_fire_resistance_%": {
		Mods: []mod.Mod{mod.NewFloat("FirePenetration", "BASE", 0)},
	},
	"base_reduce_enemy_cold_resistance_%": {
		Mods: []mod.Mod{mod.NewFloat("ColdPenetration", "BASE", 0)},
	},
	"base_reduce_enemy_lightning_resistance_%": {
		Mods: []mod.Mod{mod.NewFloat("LightningPenetration", "BASE", 0)},
	},
	"reduce_enemy_chaos_resistance_%": {
		Mods: []mod.Mod{mod.NewFloat("ChaosPenetration", "BASE", 0)},
	},
	"reduce_enemy_elemental_resistance_%": {
		Mods: []mod.Mod{mod.NewFloat("ElementalPenetration", "BASE", 0)},
	},
	"global_minimum_added_physical_damage_vs_bleeding_enemies": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalMin", "BASE", 0).Tag(mod.ActorCondition("enemy", "Bleeding"))},
	},
	"global_maximum_added_physical_damage_vs_bleeding_enemies": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalMax", "BASE", 0).Tag(mod.ActorCondition("enemy", "Bleeding"))},
	},
	"global_minimum_added_fire_damage_vs_burning_enemies": {
		Mods: []mod.Mod{mod.NewFloat("FireMin", "BASE", 0).Tag(mod.ActorCondition("enemy", "Burning"))},
	},
	"global_maximum_added_fire_damage_vs_burning_enemies": {
		Mods: []mod.Mod{mod.NewFloat("FireMax", "BASE", 0).Tag(mod.ActorCondition("enemy", "Burning"))},
	},
	"minimum_added_fire_damage_vs_ignited_enemies": {
		Mods: []mod.Mod{mod.NewFloat("FireMin", "BASE", 0).Tag(mod.ActorCondition("enemy", "Ignited"))},
	},
	"maximum_added_fire_damage_vs_ignited_enemies": {
		Mods: []mod.Mod{mod.NewFloat("FireMax", "BASE", 0).Tag(mod.ActorCondition("enemy", "Ignited"))},
	},
	"minimum_added_cold_damage_per_frenzy_charge": {
		Mods: []mod.Mod{mod.NewFloat("ColdMin", "BASE", 0).Tag(mod.Multiplier("FrenzyCharge"))},
	},
	"maximum_added_cold_damage_per_frenzy_charge": {
		Mods: []mod.Mod{mod.NewFloat("ColdMax", "BASE", 0).Tag(mod.Multiplier("FrenzyCharge"))},
	},
	"minimum_added_cold_damage_vs_chilled_enemies": {
		Mods: []mod.Mod{mod.NewFloat("ColdMin", "BASE", 0).Tag(mod.ActorCondition("enemy", "Chilled"))},
	},
	"maximum_added_cold_damage_vs_chilled_enemies": {
		Mods: []mod.Mod{mod.NewFloat("ColdMax", "BASE", 0).Tag(mod.ActorCondition("enemy", "Chilled"))},
	},
	"global_minimum_added_cold_damage": {
		Mods: []mod.Mod{mod.NewFloat("ColdMin", mod.TypeBase, 0)},
	},
	"global_maximum_added_cold_damage": {
		Mods: []mod.Mod{mod.NewFloat("ColdMax", mod.TypeBase, 0)},
	},
	"global_minimum_added_lightning_damage": {
		Mods: []mod.Mod{mod.NewFloat("LightningMin", "BASE", 0)},
	},
	"global_maximum_added_lightning_damage": {
		Mods: []mod.Mod{mod.NewFloat("LightningMax", "BASE", 0)},
	},
	"global_minimum_added_chaos_damage": {
		Mods: []mod.Mod{mod.NewFloat("ChaosMin", "BASE", 0)},
	},
	"global_maximum_added_chaos_damage": {
		Mods: []mod.Mod{mod.NewFloat("ChaosMax", "BASE", 0)},
	},
	"added_damage_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("AddedDamage", "MORE", 0)},
	},
	"active_skill_added_damage_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("AddedDamage", "MORE", 0)},
	},
	"shield_charge_damage_+%_maximum": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "MORE", 0).Flag(mod.MFlagHit).Tag(mod.DistanceRamp([][]int{
			{0, 0},
			{60, 1},
		}))},
	},
	"damage_+%_on_full_energy_shield": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.Condition("FullEnergyShield"))},
	},
	"damage_+%_when_on_low_life": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.Condition("LowLife"))},
	},
	"damage_vs_enemies_on_low_life_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.ActorCondition("enemy", "LowLife"))},
	},
	"damage_+%_when_on_full_life": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.Condition("FullLife"))},
	},
	"damage_+%_vs_enemies_on_full_life": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.ActorCondition("enemy", "FullLife"))},
	},
	"hit_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Flag(mod.MFlagHit)},
	},
	"active_skill_damage_+%_final_when_cast_on_frostbolt": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.Condition("CastOnFrostbolt"))},
	},
	// Conversion
	"physical_damage_%_to_add_as_lightning": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsLightning", "BASE", 0)},
	},
	"physical_damage_%_to_add_as_cold": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsCold", "BASE", 0)},
	},
	"physical_damage_%_to_add_as_fire": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsFire", "BASE", 0)},
	},
	"physical_damage_%_to_add_as_chaos": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamageGainAsChaos", "BASE", 0)},
	},
	"cold_damage_%_to_add_as_fire": {
		Mods: []mod.Mod{mod.NewFloat("ColdDamageGainAsFire", "BASE", 0)},
	},
	"fire_damage_%_to_add_as_chaos": {
		Mods: []mod.Mod{mod.NewFloat("FireDamageGainAsChaos", "BASE", 0)},
	},
	"lightning_damage_%_to_add_as_chaos": {
		Mods: []mod.Mod{mod.NewFloat("LightningDamageGainAsChaos", "BASE", 0)},
	},
	"base_physical_damage_%_to_convert_to_lightning": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamageConvertToLightning", "BASE", 0)},
	},
	"base_physical_damage_%_to_convert_to_cold": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamageConvertToCold", "BASE", 0)},
	},
	"base_physical_damage_%_to_convert_to_fire": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamageConvertToFire", "BASE", 0)},
	},
	"base_physical_damage_%_to_convert_to_chaos": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamageConvertToChaos", "BASE", 0)},
	},
	// Skill Physical
	"skill_physical_damage_%_to_convert_to_lightning": {
		Mods: []mod.Mod{mod.NewFloat("SkillPhysicalDamageConvertToLightning", "BASE", 0)},
	},
	"skill_physical_damage_%_to_convert_to_cold": {
		Mods: []mod.Mod{mod.NewFloat("SkillPhysicalDamageConvertToCold", "BASE", 0)},
	},
	"skill_physical_damage_%_to_convert_to_fire": {
		Mods: []mod.Mod{mod.NewFloat("SkillPhysicalDamageConvertToFire", "BASE", 0)},
	},
	"skill_physical_damage_%_to_convert_to_chaos": {
		Mods: []mod.Mod{mod.NewFloat("SkillPhysicalDamageConvertToChaos", "BASE", 0)},
	},
	// Skill Lightning Conversion
	"skill_lightning_damage_%_to_convert_to_chaos": {
		Mods: []mod.Mod{mod.NewFloat("SkillLightningDamageConvertToChaos", "BASE", 0)},
	},
	"skill_lightning_damage_%_to_convert_to_fire": {
		Mods: []mod.Mod{mod.NewFloat("SkillLightningDamageConvertToFire", "BASE", 0)},
	},
	"skill_lightning_damage_%_to_convert_to_cold": {
		Mods: []mod.Mod{mod.NewFloat("SkillLightningDamageConvertToCold", "BASE", 0)},
	},
	// Skill Cold Conversion
	"skill_cold_damage_%_to_convert_to_fire": {
		Mods: []mod.Mod{mod.NewFloat("SkillColdDamageConvertToFire", "BASE", 0)},
	},
	"skill_cold_damage_%_to_convert_to_chaos": {
		Mods: []mod.Mod{mod.NewFloat("SkillColdDamageConvertToChaos", "BASE", 0)},
	},
	"skill_fire_damage_%_to_convert_to_chaos": {
		Mods: []mod.Mod{mod.NewFloat("SkillFireDamageConvertToChaos", "BASE", 0)},
	},
	"skill_convert_%_physical_damage_to_random_element": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamageConvertToRandom", "BASE", 0)},
	},

	// Ailments
	"bleed_on_hit_with_attacks_%": {
		Mods: []mod.Mod{mod.NewFloat("BleedChance", "BASE", 0).Flag(mod.MFlagAttack)},
	},
	"global_bleed_on_hit": {
		Mods:  []mod.Mod{mod.NewFloat("BleedChance", "BASE", 0)},
		Value: utils.Ptr(float64(100)),
	},
	"bleed_on_melee_attack_chance_%": {
		Mods: []mod.Mod{mod.NewFloat("BleedChance", "BASE", 0).Flag(mod.MFlagMelee)},
	},
	"chance_to_bleed_on_hit_%_chance_in_blood_stance": {
		Mods: []mod.Mod{mod.NewFloat("BleedChance", "BASE", 0).Flag(mod.MFlagAttack).Tag(mod.Condition("BloodStance"))},
	},
	"faster_bleed_%": {
		Mods: []mod.Mod{mod.NewFloat("BleedFaster", "INC", 0)},
	},
	"base_ailment_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).KeywordFlag(mod.KeywordFlagAilment)},
	},
	"base_chance_to_poison_on_hit_%": {
		Mods: []mod.Mod{mod.NewFloat("PoisonChance", "BASE", 0)},
	},
	"global_poison_on_hit": {
		Mods:  []mod.Mod{mod.NewFloat("PoisonChance", "BASE", 0)},
		Value: utils.Ptr(float64(100)),
	},
	"base_chance_to_ignite": {
		Mods: []mod.Mod{mod.NewFloat("EnemyIgniteChance", mod.TypeBase, 0)},
	},
	"base_chance_to_shock_%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyShockChance", "BASE", 0)},
	},
	"base_chance_to_freeze_%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyFreezeChance", "BASE", 0)},
	},
	"chance_to_freeze_shock_ignite_%": {
		Mods: []mod.Mod{
			mod.NewFloat("EnemyFreezeChance", "BASE", 0),
			mod.NewFloat("EnemyShockChance", "BASE", 0),
			mod.NewFloat("EnemyIgniteChance", "BASE", 0),
		},
	},
	"additional_chance_to_freeze_chilled_enemies_%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyFreezeChance", "BASE", 0).Flag(mod.MFlagHit).Tag(mod.ActorCondition("enemy", "Chilled"))},
	},
	"chance_to_scorch_%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyScorchChance", "BASE", 0)},
	},
	"cannot_inflict_status_ailments": {
		Mods: []mod.Mod{
			mod.NewFlag("CannotShock", true),
			mod.NewFlag("CannotChill", true),
			mod.NewFlag("CannotFreeze", true),
			mod.NewFlag("CannotIgnite", true),
			mod.NewFlag("CannotScorch", true),
			mod.NewFlag("CannotBrittle", true),
			mod.NewFlag("CannotSap", true),
		},
	},
	"chill_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyChillEffect", "INC", 0)},
	},
	"shock_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyShockEffect", "INC", 0)},
	},
	"non_damaging_ailment_effect_+%": {
		Mods: []mod.Mod{
			mod.NewFloat("EnemyChillEffect", "INC", 0),
			mod.NewFloat("EnemyShockEffect", "INC", 0),
			mod.NewFloat("EnemyFreezeEffect", "INC", 0),
			mod.NewFloat("EnemyScorchEffect", "INC", 0),
			mod.NewFloat("EnemyBrittleEffect", "INC", 0),
			mod.NewFloat("EnemySapEffect", "INC", 0),
		},
	},
	"lightning_ailment_effect_+%": {
		Mods: []mod.Mod{
			mod.NewFloat("EnemyShockEffect", "INC", 0),
			mod.NewFloat("EnemySapEffect", "INC", 0),
		},
	},
	"cold_ailment_duration_+%": {
		Mods: []mod.Mod{
			mod.NewFloat("EnemyChillDuration", "INC", 0),
			mod.NewFloat("EnemyFreezeDuration", "INC", 0),
			mod.NewFloat("EnemyBrittleDuration", "INC", 0),
		},
	},
	"chill_and_freeze_duration_+%": {
		Mods: []mod.Mod{
			mod.NewFloat("EnemyChillDuration", "INC", 0),
			mod.NewFloat("EnemyFreezeDuration", "INC", 0)},
	},
	"cold_ailment_effect_+%": {
		Mods: []mod.Mod{
			mod.NewFloat("EnemyChillEffect", "INC", 0),
			mod.NewFloat("EnemyFreezeEffect", "INC", 0),
			mod.NewFloat("EnemyBrittleEffect", "INC", 0),
		},
	},
	"base_poison_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyPoisonDuration", "INC", 0)},
	},
	"active_skill_poison_duration_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("EnemyPoisonDuration", "MORE", 0)},
	},
	"ignite_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyIgniteDuration", "INC", 0)},
	},
	"lightning_ailment_duration_+%": {
		Mods: []mod.Mod{
			mod.NewFloat("EnemyShockDuration", "INC", 0),
			mod.NewFloat("EnemySapDuration", "INC", 0),
		},
	},
	"shock_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyShockDuration", "INC", 0)},
	},
	"chill_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyChillDuration", "INC", 0)},
	},
	"freeze_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyFreezeDuration", "INC", 0)},
	},
	"base_elemental_status_ailment_duration_+%": {
		Mods: []mod.Mod{
			mod.NewFloat("EnemyIgniteDuration", "INC", 0),
			mod.NewFloat("EnemyShockDuration", "INC", 0),
			mod.NewFloat("EnemyChillDuration", "INC", 0),
			mod.NewFloat("EnemyFreezeDuration", "INC", 0),
			mod.NewFloat("EnemyScorchDuration", "INC", 0),
			mod.NewFloat("EnemyBrittleDuration", "INC", 0),
			mod.NewFloat("EnemySapDuration", "INC", 0),
		},
	},
	"base_all_ailment_duration_+%": {
		Mods: []mod.Mod{
			mod.NewFloat("EnemyBleedDuration", "INC", 0),
			mod.NewFloat("EnemyPoisonDuration", "INC", 0),
			mod.NewFloat("EnemyIgniteDuration", "INC", 0),
			mod.NewFloat("EnemyShockDuration", "INC", 0),
			mod.NewFloat("EnemyChillDuration", "INC", 0),
			mod.NewFloat("EnemyFreezeDuration", "INC", 0),
			mod.NewFloat("EnemyScorchDuration", "INC", 0),
			mod.NewFloat("EnemyBrittleDuration", "INC", 0),
			mod.NewFloat("EnemySapDuration", "INC", 0),
		},
	},
	"bleeding_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).KeywordFlag(mod.KeywordFlagBleed)},
	},
	"active_skill_bleeding_damage_+%_final_in_blood_stance": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "MORE", 0).KeywordFlag(mod.KeywordFlagBleed).Tag(mod.Condition("BloodStance"))},
	},
	"base_poison_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).KeywordFlag(mod.KeywordFlagPoison)},
	},
	"critical_poison_dot_multiplier_+": {
		Mods: []mod.Mod{mod.NewFloat("DotMultiplier", "BASE", 0).KeywordFlag(mod.KeywordFlagPoison).Tag(mod.Condition("CriticalStrike"))},
	},
	"poison_dot_multiplier_+": {
		Mods: []mod.Mod{mod.NewFloat("DotMultiplier", "BASE", 0).KeywordFlag(mod.KeywordFlagPoison)},
	},
	"dot_multiplier_+": {
		Mods: []mod.Mod{mod.NewFloat("DotMultiplier", "BASE", 0)},
	},
	"fire_dot_multiplier_+": {
		Mods: []mod.Mod{mod.NewFloat("FireDotMultiplier", "BASE", 0)},
	},
	"chaos_dot_multiplier_+": {
		Mods: []mod.Mod{mod.NewFloat("ChaosDotMultiplier", "BASE", 0)},
	},
	"cold_dot_multiplier_+": {
		Mods: []mod.Mod{mod.NewFloat("ColdDotMultiplier", "BASE", 0)},
	},
	"active_skill_ignite_damage_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "MORE", 0).KeywordFlag(mod.KeywordFlagIgnite)},
	},
	"damaging_ailments_deal_damage_+%_faster": {
		Mods: []mod.Mod{
			mod.NewFloat("BleedFaster", "INC", 0),
			mod.NewFloat("PoisonFaster", "INC", 0),
			mod.NewFloat("IgniteBurnFaster", "INC", 0),
		},
	},
	"active_skill_shock_as_though_damage_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("ShockAsThoughDealing", "MORE", 0)},
	},
	"active_skill_chill_as_though_damage_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("ChillAsThoughDealing", "MORE", 0)},
	},
	"ailment_damage_+%_per_frenzy_charge": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).KeywordFlag(mod.KeywordFlagAilment).Tag(mod.Multiplier("FrenzyCharge"))},
	},
	"freeze_as_though_dealt_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("FreezeAsThoughDealing", "MORE", 0)},
	},
	// Global flags
	"never_ignite": {
		Mods: []mod.Mod{mod.NewFlag("CannotIgnite", true)},
	},
	"never_shock": {
		Mods: []mod.Mod{mod.NewFlag("CannotShock", true)},
	},
	"never_freeze": {
		Mods: []mod.Mod{mod.NewFlag("CannotFreeze", true)},
	},
	"cannot_cause_bleeding": {
		Mods: []mod.Mod{mod.NewFlag("CannotBleed", true)},
	},
	"keystone_strong_bowman": {
		Mods: []mod.Mod{mod.NewFlag("IronGrip", true)},
	},
	"strong_casting": {
		Mods: []mod.Mod{mod.NewFlag("IronWill", true)},
	},
	"deal_no_elemental_damage": {
		Mods: []mod.Mod{
			mod.NewFlag("DealNoFire", true),
			mod.NewFlag("DealNoCold", true),
			mod.NewFlag("DealNoLightning", true),
		},
	},
	"base_deal_no_chaos_damage": {
		Mods: []mod.Mod{mod.NewFlag("DealNoChaos", true)},
	},
	"all_damage_can_ignite": {
		Mods: []mod.Mod{
			mod.NewFlag("PhysicalCanIgnite", true),
			mod.NewFlag("LightningCanIgnite", true),
			mod.NewFlag("ColdCanIgnite", true),
			mod.NewFlag("ChaosCanIgnite", true),
		},
	},
	"all_damage_can_freeze": {
		Mods: []mod.Mod{
			mod.NewFlag("PhysicalCanFreeze", true),
			mod.NewFlag("LightningCanFreeze", true),
			mod.NewFlag("FireCanFreeze", true),
			mod.NewFlag("ChaosCanFreeze", true),
		},
	},
	"all_damage_can_shock": {
		Mods: []mod.Mod{
			mod.NewFlag("PhysicalCanShock", true),
			mod.NewFlag("ColdCanShock", true),
			mod.NewFlag("FireCanShock", true),
			mod.NewFlag("ChaosCanShock", true),
		},
	},
	// Other effects
	"enemy_phys_reduction_%_penalty_vs_hit": {
		Mods: []mod.Mod{mod.NewFloat("EnemyPhysicalDamageReduction", "BASE", 0)},
		Mult: utils.Ptr(float64(-1)),
	},
	"base_stun_threshold_reduction_+%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyStunThreshold", "INC", 0)},
		Mult: utils.Ptr(float64(-1)),
	},
	"impale_phys_reduction_%_penalty": {
		Mods: []mod.Mod{mod.NewFloat("EnemyImpalePhysicalDamageReduction", "BASE", 0)},
		Mult: utils.Ptr(float64(-1)),
	},
	"base_stun_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyStunDuration", "INC", 0)},
	},
	"base_killed_monster_dropped_item_quantity_+%": {
		Mods: []mod.Mod{mod.NewFloat("LootQuantity", "INC", 0)},
	},
	"base_killed_monster_dropped_item_rarity_+%": {
		Mods: []mod.Mod{mod.NewFloat("LootRarity", "INC", 0)},
	},
	"global_knockback": {
		Mods:  []mod.Mod{mod.NewFloat("EnemyKnockbackChance", "BASE", 0)},
		Value: utils.Ptr(float64(100)),
	},
	"base_global_chance_to_knockback_%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyKnockbackChance", "BASE", 0)},
	},
	"knockback_distance_+%": {
		Mods: []mod.Mod{mod.NewFloat("EnemyKnockbackDistance", "INC", 0)},
	},
	"chance_to_be_knocked_back_%": {
		Mods: []mod.Mod{mod.NewFloat("SelfKnockbackChance", "BASE", 0)},
	},
	"number_of_additional_curses_allowed": {
		Mods: []mod.Mod{mod.NewFloat("EnemyCurseLimit", "BASE", 0)},
	},
	"consecrated_ground_enemy_damage_taken_+%": {
		Mods: []mod.Mod{mod.NewFloat("DamageTakenConsecratedGround", "INC", 0).Tag(mod.GlobalEffect("Debuff"), mod.Condition("OnConsecratedGround"))},
	},
	"consecrated_ground_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("ConsecratedGroundEffect", "INC", 0).Tag(mod.GlobalEffect("Buff"))},
	},
	"base_inflict_cold_exposure_on_hit_%_chance": {
		Mods: []mod.Mod{mod.NewFloat("ColdExposureChance", "BASE", 0)},
	},
	"base_inflict_lightning_exposure_on_hit_%_chance": {
		Mods: []mod.Mod{mod.NewFloat("LightningExposureChance", "BASE", 0)},
	},
	"base_inflict_fire_exposure_on_hit_%_chance": {
		Mods: []mod.Mod{mod.NewFloat("FireExposureChance", "BASE", 0)},
	},
	// Projectiles
	"base_projectile_speed_+%": {
		Mods: []mod.Mod{mod.NewFloat("ProjectileSpeed", "INC", 0)},
	},
	"base_arrow_speed_+%": {
		Mods: []mod.Mod{mod.NewFloat("ProjectileSpeed", "INC", 0)},
	},
	"projectile_base_number_of_targets_to_pierce": {
		Mods: []mod.Mod{mod.NewFloat("PierceCount", "BASE", 0)},
	},
	"arrow_base_number_of_targets_to_pierce": {
		Mods: []mod.Mod{mod.NewFloat("PierceCount", "BASE", 0).Flag(mod.MFlagAttack)},
	},
	"pierce_%": {
		Mods: []mod.Mod{mod.NewFloat("PierceChance", "BASE", 0)},
	},
	"always_pierce": {
		Mods: []mod.Mod{mod.NewFlag("PierceAllTargets", true)},
	},
	"cannot_pierce": {
		Mods: []mod.Mod{mod.NewFlag("CannotPierce", true)},
	},
	"base_number_of_additional_arrows": {
		Mods: []mod.Mod{mod.NewFloat("ProjectileCount", "BASE", 0)},
	},
	"number_of_additional_projectiles": {
		Mods: []mod.Mod{mod.NewFloat("ProjectileCount", "BASE", 0)},
	},
	"projectile_damage_+%_per_remaining_chain": {
		Mods: []mod.Mod{
			mod.NewFloat("Damage", "INC", 0).Flag(mod.MFlagProjectile).Tag(mod.PerStat(0, "ChainRemaining")),
			mod.NewFloat("Damage", "INC", 0).Flag(mod.MFlagAilment).Tag(mod.PerStat(0, "ChainRemaining")),
		},
	},
	"number_of_chains": {
		Mods: []mod.Mod{mod.NewFloat("ChainCountMax", "BASE", 0)},
	},
	"additional_beam_only_chains": {
		Mods: []mod.Mod{mod.NewFloat("BeamChainCountMax", "BASE", 0)},
	},
	"damage_+%_per_chain": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.PerStat(0, "Chain"))},
	},
	"projectiles_always_pierce_you": {
		Mods: []mod.Mod{mod.NewFlag("AlwaysPierceSelf", true)},
	},
	"projectiles_fork": {
		Mods: []mod.Mod{
			mod.NewFlag("ForkOnce", true),
			mod.NewFloat("ForkCountMax", "BASE", 0),
		},
	},
	"number_of_additional_forks_base": {
		Mods: []mod.Mod{
			mod.NewFlag("ForkTwice", true),
			mod.NewFloat("ForkCountMax", "BASE", 0),
		},
	},
	"active_skill_returning_projectile_damage_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "MORE", 0).Tag(mod.Condition("ReturningProjectile"))},
	},
	"returning_projectiles_always_pierce": {
		Mods: []mod.Mod{mod.NewFlag("PierceAllTargets", true).Tag(mod.Condition("ReturningProjectile"))},
	},
	"support_barrage_attack_time_+%_per_projectile_fired": {
		Mods: []mod.Mod{mod.NewFloat("SkillAttackTime", "MORE", 0).Tag(mod.Condition("UsingBow", "UsingWand"), mod.PerStat(0, "ProjectileCount"))},
	},
	"support_barrage_trap_and_mine_throwing_time_+%_final_per_projectile_fired": {
		Mods: []mod.Mod{
			mod.NewFloat("SkillMineThrowingTime", "MORE", 0).Tag(mod.PerStat(0, "ProjectileCount")),
			mod.NewFloat("SkillTrapThrowingTime", "MORE", 0).Tag(mod.PerStat(0, "ProjectileCount")),
		},
	},
	// Self modifiers
	"chance_to_be_pierced_%": {
		Mods: []mod.Mod{mod.NewFloat("SelfPierceChance", "BASE", 0)},
	},
	"projectile_damage_taken_+%": {
		Mods: []mod.Mod{mod.NewFloat("ProjectileDamageTaken", "INC", 0)},
	},
	"physical_damage_taken_+%": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTaken", "INC", 0)},
	},
	"fire_damage_taken_+%": {
		Mods: []mod.Mod{mod.NewFloat("FireDamageTaken", "INC", 0)},
	},
	"cold_damage_taken_+%": {
		Mods: []mod.Mod{mod.NewFloat("ColdDamageTaken", "INC", 0)},
	},
	"lightning_damage_taken_+%": {
		Mods: []mod.Mod{mod.NewFloat("LightningDamageTaken", "INC", 0)},
	},
	"chaos_damage_taken_+%": {
		Mods: []mod.Mod{mod.NewFloat("ChaosDamageTaken", "INC", 0)},
	},
	"base_physical_damage_over_time_taken_+%": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDamageTakenOverTime", "INC", 0)},
	},
	"degen_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("DamageTakenOverTime", "INC", 0)},
	},
	"buff_time_passed_-%": {
		Mods: []mod.Mod{mod.NewFloat("BuffExpireFaster", "MORE", 0)},
		Mult: utils.Ptr(float64(-1)),
	},
	"additional_chance_to_take_critical_strike_%": {
		Mods: []mod.Mod{mod.NewFloat("SelfExtraCritChance", "BASE", 0)},
	},
	"base_self_critical_strike_multiplier_-%": {
		Mods: []mod.Mod{mod.NewFloat("SelfCritMultiplier", "INC", 0)},
		Mult: utils.Ptr(float64(-1)),
	},
	"chance_to_be_shocked_%": {
		Mods: []mod.Mod{mod.NewFloat("SelfShockChance", "BASE", 0)},
	},
	"chance_to_be_ignited_%": {
		Mods: []mod.Mod{mod.NewFloat("SelfIgniteChance", "BASE", 0)},
	},
	"chance_to_be_frozen_%": {
		Mods: []mod.Mod{mod.NewFloat("SelfFreezeChance", "BASE", 0)},
	},
	"receive_bleeding_chance_%_when_hit_by_attack": {
		Mods: []mod.Mod{mod.NewFloat("SelfBleedChance", "BASE", 0)},
	},
	"base_self_shock_duration_-%": {
		Mods: []mod.Mod{mod.NewFloat("SelfShockDuration", "INC", 0)},
		Mult: utils.Ptr(float64(-1)),
	},
	"base_self_ignite_duration_-%": {
		Mods: []mod.Mod{mod.NewFloat("SelfIgniteDuration", "INC", 0)},
		Mult: utils.Ptr(float64(-1)),
	},
	"base_self_freeze_duration_-%": {
		Mods: []mod.Mod{mod.NewFloat("SelfFreezeDuration", "INC", 0)},
		Mult: utils.Ptr(float64(-1)),
	},
	"life_leech_on_any_damage_when_hit_permyriad": {
		Mods: []mod.Mod{mod.NewFloat("SelfDamageLifeLeech", "BASE", 0)},
	},
	"mana_leech_on_any_damage_when_hit_permyriad": {
		Mods: []mod.Mod{mod.NewFloat("SelfDamageManaLeech", "BASE", 0)},
	},
	"life_granted_when_hit_by_attacks": {
		Mods: []mod.Mod{mod.NewFloat("SelfLifeOnHit", "BASE", 0).Flag(mod.MFlagAttack)},
	},
	"mana_granted_when_hit_by_attacks": {
		Mods: []mod.Mod{mod.NewFloat("SelfManaOnHit", "BASE", 0).Flag(mod.MFlagAttack)},
	},
	"life_granted_when_killed": {
		Mods: []mod.Mod{mod.NewFloat("SelfLifeOnKill", "BASE", 0)},
	},
	"mana_granted_when_killed": {
		Mods: []mod.Mod{mod.NewFloat("SelfManaOnKill", "BASE", 0)},
	},
	// Degen
	"base_physical_damage_%_of_maximum_life_to_deal_per_minute": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDegen", "BASE", 0).Tag(mod.PerStat(1, "Life"))},
		Div:  utils.Ptr(float64(6000)),
	},
	"base_physical_damage_%_of_maximum_energy_shield_to_deal_per_minute": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalDegen", "BASE", 0).Tag(mod.PerStat(1, "EnergyShield"))},
		Div:  utils.Ptr(float64(6000)),
	},
	"base_nonlethal_fire_damage_%_of_maximum_life_taken_per_minute": {
		Mods: []mod.Mod{mod.NewFloat("FireDegen", "BASE", 0).Tag(mod.PerStat(1, "Life"))},
		Div:  utils.Ptr(float64(6000)),
	},
	"base_nonlethal_fire_damage_%_of_maximum_energy_shield_taken_per_minute": {
		Mods: []mod.Mod{mod.NewFloat("FireDegen", "BASE", 0).Tag(mod.PerStat(1, "EnergyShield"))},
		Div:  utils.Ptr(float64(6000)),
	},
	//
	// Attack modifiers
	//
	"attack_speed_+%": {
		Mods: []mod.Mod{mod.NewFloat("Speed", "INC", 0).Flag(mod.MFlagAttack)},
	},
	"active_skill_attack_speed_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("Speed", "MORE", 0).Flag(mod.MFlagAttack)},
	},
	"base_attack_speed_+%_per_frenzy_charge": {
		Mods: []mod.Mod{mod.NewFloat("Speed", "INC", 0).Flag(mod.MFlagAttack).Tag(mod.Multiplier("FrenzyCharge"))},
	},
	"attack_speed_+%_when_on_low_life": {
		Mods: []mod.Mod{mod.NewFloat("Speed", "INC", 0).Flag(mod.MFlagAttack).Tag(mod.Condition("LowLife"))},
	},
	"damage_+%_per_power_charge": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.Multiplier("PowerCharge"))},
	},
	"accuracy_rating": {
		Mods: []mod.Mod{mod.NewFloat("Accuracy", "BASE", 0)},
	},
	"accuracy_rating_+%": {
		Mods: []mod.Mod{mod.NewFloat("Accuracy", "INC", 0)},
	},
	"accuracy_rating_+%_when_on_low_life": {
		Mods: []mod.Mod{mod.NewFloat("Accuracy", "INC", 0).Tag(mod.Condition("LowLife"))},
	},
	"attack_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Flag(mod.MFlagAttack)},
	},
	"elemental_damage_with_attack_skills_+%": {
		Mods: []mod.Mod{mod.NewFloat("ElementalDamage", "INC", 0).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"attack_minimum_added_physical_damage": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalMin", "BASE", 0).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"attack_maximum_added_physical_damage": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalMax", "BASE", 0).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"attack_minimum_added_physical_damage_with_weapons": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalMin", "BASE", 0).Flag(mod.MFlagWeapon).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"attack_maximum_added_physical_damage_with_weapons": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalMax", "BASE", 0).Flag(mod.MFlagWeapon).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"attack_minimum_added_lightning_damage": {
		Mods: []mod.Mod{mod.NewFloat("LightningMin", "BASE", 0).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"attack_maximum_added_lightning_damage": {
		Mods: []mod.Mod{mod.NewFloat("LightningMax", "BASE", 0).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"attack_minimum_added_cold_damage": {
		Mods: []mod.Mod{mod.NewFloat("ColdMin", "BASE", 0).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"attack_maximum_added_cold_damage": {
		Mods: []mod.Mod{mod.NewFloat("ColdMax", "BASE", 0).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"attack_minimum_added_fire_damage": {
		Mods: []mod.Mod{mod.NewFloat("FireMin", "BASE", 0).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"attack_maximum_added_fire_damage": {
		Mods: []mod.Mod{mod.NewFloat("FireMax", "BASE", 0).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"attack_minimum_added_chaos_damage": {
		Mods: []mod.Mod{mod.NewFloat("ChaosMin", "BASE", 0).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"attack_maximum_added_chaos_damage": {
		Mods: []mod.Mod{mod.NewFloat("ChaosMax", "BASE", 0).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"melee_weapon_range_+": {
		Mods: []mod.Mod{mod.NewFloat("MeleeWeaponRange", "BASE", 0)},
	},

	"melee_range_+": {
		Mods: []mod.Mod{
			mod.NewFloat("MeleeWeaponRange", "BASE", 0),
			mod.NewFloat("UnarmedRange", "BASE", 0),
		},
	},
	"override_off_hand_base_critical_strike_chance_to_5%": {
		Mods:  []mod.Mod{skill("setOffHandBaseCritChance", 0)},
		Value: utils.Ptr(float64(5)),
	},
	"off_hand_local_minimum_added_physical_damage": {
		Mods: []mod.Mod{skill("setOffHandPhysicalMin", 0)},
	},
	"off_hand_local_maximum_added_physical_damage": {
		Mods: []mod.Mod{skill("setOffHandPhysicalMax", 0)},
	},
	"off_hand_base_weapon_attack_duration_ms": {
		Mods: []mod.Mod{skill("setOffHandAttackTime", 0)},
	},
	"off_hand_minimum_added_physical_damage_per_15_shield_armour_and_evasion_rating": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalMin", "BASE", 0).Tag(mod.Condition("OffHandAttack"), mod.PerStat(15, "ArmourOnWeapon 2", "EvasionOnWeapon 2"))},
	},
	"off_hand_maximum_added_physical_damage_per_15_shield_armour_and_evasion_rating": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalMax", "BASE", 0).Tag(mod.Condition("OffHandAttack"), mod.PerStat(15, "ArmourOnWeapon 2", "EvasionOnWeapon 2"))},
	},
	"additional_critical_strike_chance_per_10_shield_maximum_energy_shield_permyriad": {
		Mods: []mod.Mod{mod.NewFloat("CritChance", "BASE", 0).Tag(mod.PerStat(10, "EnergyShieldOnWeapon 2"))},
		Div:  utils.Ptr(float64(100)),
	},
	// Impale
	"attacks_impale_on_hit_%_chance": {
		Mods: []mod.Mod{mod.NewFloat("ImpaleChance", "BASE", 0).KeywordFlag(mod.KeywordFlagAttack)},
	},
	"impale_on_hit_%_chance": {
		Mods: []mod.Mod{mod.NewFloat("ImpaleChance", "BASE", 0)},
	},
	"spells_impale_on_hit_%_chance": {
		Mods: []mod.Mod{mod.NewFloat("ImpaleChance", "BASE", 0).KeywordFlag(mod.KeywordFlagSpell)},
	},
	"impale_debuff_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("ImpaleEffect", "INC", 0)},
	},
	//
	// Spell modifiers
	//
	"base_cast_speed_+%": {
		Mods: []mod.Mod{mod.NewFloat("Speed", "INC", 0).Flag(mod.MFlagCast)},
	},
	"active_skill_cast_speed_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("Speed", "MORE", 0).Flag(mod.MFlagCast)},
	},
	"spell_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Flag(mod.MFlagSpell)},
	},
	"spell_minimum_added_physical_damage": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalMin", "BASE", 0).Flag(mod.MFlagSpell)},
	},
	"spell_maximum_added_physical_damage": {
		Mods: []mod.Mod{mod.NewFloat("PhysicalMax", "BASE", 0).Flag(mod.MFlagSpell)},
	},
	"spell_minimum_added_lightning_damage": {
		Mods: []mod.Mod{mod.NewFloat("LightningMin", "BASE", 0).Flag(mod.MFlagSpell)},
	},
	"spell_maximum_added_lightning_damage": {
		Mods: []mod.Mod{mod.NewFloat("LightningMax", "BASE", 0).Flag(mod.MFlagSpell)},
	},
	"spell_minimum_added_cold_damage": {
		Mods: []mod.Mod{mod.NewFloat("ColdMin", "BASE", 0).Flag(mod.MFlagSpell)},
	},
	"spell_maximum_added_cold_damage": {
		Mods: []mod.Mod{mod.NewFloat("ColdMax", "BASE", 0).Flag(mod.MFlagSpell)},
	},
	"spell_minimum_added_fire_damage": {
		Mods: []mod.Mod{mod.NewFloat("FireMin", "BASE", 0).Flag(mod.MFlagSpell)},
	},
	"spell_maximum_added_fire_damage": {
		Mods: []mod.Mod{mod.NewFloat("FireMax", "BASE", 0).Flag(mod.MFlagSpell)},
	},
	"spell_minimum_added_chaos_damage": {
		Mods: []mod.Mod{mod.NewFloat("ChaosMin", "BASE", 0).Flag(mod.MFlagSpell)},
	},
	"spell_maximum_added_chaos_damage": {
		Mods: []mod.Mod{mod.NewFloat("ChaosMax", "BASE", 0).Flag(mod.MFlagSpell)},
	},
	//
	// Skill type modifier
	//
	// Trap
	"support_trap_damage_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "MORE", 0).KeywordFlag(mod.KeywordFlagTrap)},
	},
	"trap_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).KeywordFlag(mod.KeywordFlagTrap)},
	},
	"number_of_additional_traps_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveTrapLimit", "BASE", 0)},
	},
	"trap_throwing_speed_+%": {
		Mods: []mod.Mod{mod.NewFloat("TrapThrowingSpeed", "INC", 0)},
	},
	"trap_throwing_speed_+%_per_frenzy_charge": {
		Mods: []mod.Mod{mod.NewFloat("TrapThrowingSpeed", "INC", 0).Tag(mod.Multiplier("FrenzyCharge"))},
	},
	"trap_critical_strike_multiplier_+_per_power_charge": {
		Mods: []mod.Mod{mod.NewFloat("CritMultiplier", "BASE", 0).KeywordFlag(mod.KeywordFlagTrap).Tag(mod.Multiplier("PowerCharge"))},
	},
	"placing_traps_cooldown_recovery_+%": {
		Mods: []mod.Mod{mod.NewFloat("CooldownRecovery", "INC", 0).KeywordFlag(mod.KeywordFlagTrap)},
	},
	"trap_trigger_radius_+%": {
		Mods: []mod.Mod{mod.NewFloat("TrapTriggerAreaOfEffect", "INC", 0)},
	},
	// Mine
	"number_of_additional_remote_mines_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveMineLimit", "BASE", 0)},
	},
	"mine_laying_speed_+%": {
		Mods: []mod.Mod{mod.NewFloat("MineLayingSpeed", "INC", 0)},
	},
	"mine_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).KeywordFlag(mod.KeywordFlagMine)},
	},
	"mine_detonation_radius_+%": {
		Mods: []mod.Mod{mod.NewFloat("MineDetonationAreaOfEffect", "INC", 0)},
	},
	"mine_throwing_speed_+%_per_frenzy_charge": {
		Mods: []mod.Mod{mod.NewFloat("MineLayingSpeed", "INC", 0).Tag(mod.Multiplier("FrenzyCharge"))},
	},
	"remote_mined_by_support": {
		Mods: []mod.Mod{
			mod.NewFlag("ManaCostGainAsReservation", true),
			mod.NewFlag("LifeCostGainAsReservation", true),
		},
	},
	// Totem
	"totem_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).KeywordFlag(mod.KeywordFlagTotem)},
	},
	"totem_life_+%": {
		Mods: []mod.Mod{mod.NewFloat("TotemLife", "INC", 0)},
	},
	"number_of_additional_totems_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveTotemLimit", "BASE", 0)},
	},
	"attack_skills_additional_ballista_totems_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveTotemLimit", "BASE", 0).Tag(mod.SkillType("RangedAttack"))},
	},
	"base_number_of_totems_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveTotemLimit", "BASE", 0)},
	},
	"summon_totem_cast_speed_+%": {
		Mods: []mod.Mod{mod.NewFloat("TotemPlacementSpeed", "INC", 0)},
	},
	"totems_regenerate_%_life_per_minute": {
		Mods: []mod.Mod{mod.NewFloat("LifeRegenPercent", "BASE", 0).KeywordFlag(mod.KeywordFlagTotem)},
		Div:  utils.Ptr(float64(60)),
	},
	"totem_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("TotemDuration", "INC", 0)},
	},
	// Minion
	"minion_damage_+%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", "INC", 0)})},
	},
	"minion_melee_damage_+%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", "INC", 0).Flag(mod.MFlagMelee)})},
	},
	"active_skill_minion_bleeding_damage_+%_final": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", "MORE", 0).KeywordFlag(mod.KeywordFlagBleed)})},
	},
	"minion_maximum_life_+%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Life", "INC", 0)})},
	},
	"minion_movement_speed_+%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("MovementSpeed", "INC", 0)})},
	},
	"minion_attack_speed_+%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Speed", "INC", 0).Flag(mod.MFlagAttack)})},
	},
	"minion_cast_speed_+%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Speed", "INC", 0).Flag(mod.MFlagCast)})},
	},
	"minion_elemental_resistance_%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("ElementalResist", "BASE", 0)})},
	},
	"minion_elemental_resistance_30%": {
		Mods:  []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("ElementalResist", "BASE", 0)})},
		Value: utils.Ptr(float64(30)),
	},
	"base_minion_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("Duration", "INC", 0).Tag(mod.SkillType("CreatesMinion"))},
	},
	"minion_skill_area_of_effect_+%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("AreaOfEffect", "INC", 0)})},
	},
	"minion_additional_physical_damage_reduction_%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDamageReduction", "BASE", 0)})},
	},
	"summon_fire_resistance_+": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("FireResist", "BASE", 0)})},
	},
	"summon_cold_resistance_+": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("ColdResist", "BASE", 0)})},
	},
	"summon_lightning_resistance_+": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("LightningResist", "BASE", 0)})},
	},
	"minion_maximum_all_elemental_resistances_%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("ElementalResistMax", "BASE", 0)})},
	},
	"minion_cooldown_recovery_+%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("CooldownRecovery", "INC", 0)})},
	},
	"minion_life_regeneration_rate_per_minute_%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("LifeRegenPercent", "BASE", 0)})},
		Div:  utils.Ptr(float64(60)),
	},
	"base_number_of_zombies_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveZombieLimit", "BASE", 0)},
	},
	"base_number_of_skeletons_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveSkeletonLimit", "BASE", 0)},
	},
	"base_number_of_raging_spirits_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveRagingSpiritLimit", "BASE", 0)},
	},
	"base_number_of_golems_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveGolemLimit", "BASE", 0)},
	},
	"base_number_of_champions_of_light_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveSentinelOfPurityLimit", "BASE", 0)},
	},
	"base_number_of_spectres_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveSpectreLimit", "BASE", 0)},
	},
	"number_of_wolves_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveWolfLimit", "BASE", 0)},
	},
	"number_of_spider_minions_allowed": {
		Mods: []mod.Mod{mod.NewFloat("ActiveSpiderLimit", "BASE", 0)},
	},
	"active_skill_minion_damage_+%_final": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", "MORE", 0)})},
	},
	"active_skill_minion_attack_speed_+%_final": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Speed", "MORE", 0).Flag(mod.MFlagAttack)})},
	},
	"active_skill_minion_physical_damage_+%_final": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDamage", "MORE", 0)})},
	},
	"active_skill_minion_life_+%_final": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Life", "MORE", 0)})},
	},
	"support_minion_damage_minion_life_+%_final": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Life", "MORE", 0)})},
	},
	"active_skill_minion_energy_shield_+%_final": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("EnergyShield", "MORE", 0)})},
	},
	"active_skill_minion_movement_velocity_+%_final": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("MovementSpeed", "MORE", 0)})},
	},
	"minions_deal_%_of_physical_damage_as_additional_chaos_damage": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDamageGainAsChaos", "BASE", 0)})},
	},
	"maximum_life_+%_for_corpses_you_create": {
		Mods: []mod.Mod{mod.NewFloat("CorpseLife", "INC", 0)},
	},
	// Golem
	"golem_buff_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("BuffEffect", "INC", 0)},
	},
	"golem_cooldown_recovery_+%": {
		Mods: []mod.Mod{mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("CooldownRecovery", "INC", 0)})},
	},
	// Slam
	"warcry_grant_damage_+%_to_exerted_attacks": {
		Mods: []mod.Mod{mod.NewFloat("ExertIncrease", "INC", 0).Flag(mod.MFlagAttack)},
	},
	// Curse
	"curse_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("CurseEffect", "INC", 0)},
	},
	"curse_effect_+%_vs_players": {
		Mods: []mod.Mod{mod.NewFloat("CurseEffectAgainstPlayer", "INC", 0)},
	},
	"curse_area_of_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", "INC", 0).KeywordFlag(mod.KeywordFlagCurse)},
	},
	"base_curse_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("Duration", "INC", 0).KeywordFlag(mod.KeywordFlagCurse)},
	},
	"curse_skill_effect_duration_+%": {
		Mods: []mod.Mod{mod.NewFloat("Duration", "INC", 0).KeywordFlag(mod.KeywordFlagCurse)},
	},
	"curse_cast_speed_+%": {
		Mods: []mod.Mod{mod.NewFloat("Speed", "INC", 0)},
	},
	// Hex
	"curse_maximum_doom": {
		Mods: []mod.Mod{mod.NewFloat("MaxDoom", "BASE", 0)},
	},
	// Aura
	"non_curse_aura_effect_+%": {
		Mods: []mod.Mod{mod.NewFloat("AuraEffect", "INC", 0).Tag(mod.SkillType("AppliesCurse, neg = true"))},
	},
	"base_mana_reservation_+%": {
		Mods: []mod.Mod{mod.NewFloat("ManaReserved", "INC", 0)},
	},
	"base_life_reservation_+%": {
		Mods: []mod.Mod{mod.NewFloat("LifeReserved", "INC", 0)},
	},
	"base_reservation_+%": {
		Mods: []mod.Mod{mod.NewFloat("Reserved", "INC", 0)},
	},
	"base_mana_reservation_efficiency_+%": {
		Mods: []mod.Mod{mod.NewFloat("ManaReservationEfficiency", "INC", 0)},
	},
	"base_life_reservation_efficiency_+%": {
		Mods: []mod.Mod{mod.NewFloat("LifeReservationEfficiency", "INC", 0)},
	},
	"base_reservation_efficiency_+%": {
		Mods: []mod.Mod{mod.NewFloat("ReservationEfficiency", "INC", 0)},
	},
	// Brand
	"sigil_attached_target_damage_+%_final": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "MORE", 0).Tag(mod.MultiplierThreshold("BrandsAttachedToEnemy").Threshold(1))},
	},
	"base_number_of_sigils_allowed_per_target": {
		Mods: []mod.Mod{mod.NewFloat("BrandsAttachedLimit", "BASE", 0)},
	},
	"base_sigil_repeat_frequency_ms": {
		Mods: []mod.Mod{skill("repeatFrequency", 0)},
		Div:  utils.Ptr(float64(1000)),
	},
	"sigil_repeat_frequency_+%": {
		Mods: []mod.Mod{mod.NewFloat("BrandActivationFrequency", "INC", 0)},
	},
	"additive_cast_speed_modifiers_apply_to_sigil_repeat_frequency": {},
	// Banner
	"banner_buff_effect_+%_per_stage": {
		Mods: []mod.Mod{mod.NewFloat("AuraEffect", "INC", 0).Tag(mod.Multiplier("BannerStage"), mod.Condition("BannerPlanted"))},
	},
	"banner_area_of_effect_+%_per_stage": {
		Mods: []mod.Mod{mod.NewFloat("AreaOfEffect", "INC", 0).Tag(mod.Multiplier("BannerStage"), mod.Condition("BannerPlanted"))},
	},
	// Other
	"triggered_skill_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("TriggeredDamage", "INC", 0).Tag(mod.SkillType("Triggered"))},
	},
	"channelled_skill_damage_+%": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.SkillType("Channel"))},
	},
	"snipe_triggered_skill_hit_damage_+%_final_per_stage": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "MORE", 0).Flag(mod.MFlagHit).Tag(mod.Multiplier("SnipeStage"))},
	},
	"snipe_triggered_skill_ailment_damage_+%_final_per_stage": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "MORE", 0).Flag(mod.MFlagAilment).Tag(mod.Multiplier("SnipeStage"))},
	},
	"withered_on_hit_chance_%": {
		Mods: []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	},
	"withered_on_hit_for_2_seconds_%_chance": {
		Mods: []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	},
	"discharge_damage_+%_if_3_charge_types_removed": {
		Mods: []mod.Mod{mod.NewFloat("Damage", "INC", 0).Tag(mod.Multiplier("RemovableEnduranceCharge").Limit(1), mod.Multiplier("RemovableFrenzyCharge").Limit(1), mod.Multiplier("RemovablePowerCharge").Limit(1))},
	},
	"support_added_cooldown_count_if_not_instant": {
		Mods: []mod.Mod{mod.NewFloat("AdditionalCooldownUses", "BASE", 0)},
	},
	"kill_enemy_on_hit_if_under_10%_life": {
		Mods:  []mod.Mod{mod.NewFloat("CullPercent", "MAX", 0)},
		Value: utils.Ptr(float64(10)),
	},
	//
	// Spectre or Minion-specific stats
	//
	"physical_damage_reduction_rating_+%": {
		Mods: []mod.Mod{mod.NewFloat("Armour", "INC", 0)},
	},
	//
	// Gem Levels
	//
	//Fire
	"supported_fire_skill_gem_level_+": {
		Mods: []mod.Mod{mod.NewList("SupportedGemProperty", mod.SupportedGemProperty{Keyword: "active_skill", Key: "level", Value: 0}).Tag(mod.SkillType("Fire"))},
	},
	//Cold
	"supported_cold_skill_gem_level_+": {
		Mods: []mod.Mod{mod.NewList("SupportedGemProperty", mod.SupportedGemProperty{Keyword: "active_skill", Key: "level", Value: 0}).Tag(mod.SkillType("Cold"))},
	},
	//Lightning
	"supported_lightning_skill_gem_level_+": {
		Mods: []mod.Mod{mod.NewList("SupportedGemProperty", mod.SupportedGemProperty{Keyword: "active_skill", Key: "level", Value: 0}).Tag(mod.SkillType("Lightning"))},
	},
	//Chaos
	"supported_chaos_skill_gem_level_+": {
		Mods: []mod.Mod{mod.NewList("SupportedGemProperty", mod.SupportedGemProperty{Keyword: "active_skill", Key: "level", Value: 0}).Tag(mod.SkillType("Chaos"))},
	},
	//Physical
	"supported_physical_skill_gem_level_+": {
		Mods: []mod.Mod{mod.NewList("SupportedGemProperty", mod.SupportedGemProperty{Keyword: "active_skill", Key: "level", Value: 0}).Tag(mod.SkillType("Physical"))},
	},
	//Active
	"supported_active_skill_gem_level_+": {
		Mods: []mod.Mod{mod.NewList("SupportedGemProperty", mod.SupportedGemProperty{Keyword: "active_skill", Key: "level", Value: 0})},
	},
	//Aura
	"supported_aura_skill_gem_level_+": {
		Mods: []mod.Mod{mod.NewList("SupportedGemProperty", mod.SupportedGemProperty{Keyword: "active_skill", Key: "level", Value: 0}).Tag(mod.SkillType("Aura"))},
	},
	//Curse
	"supported_curse_skill_gem_level_+": {
		Mods: []mod.Mod{mod.NewList("SupportedGemProperty", mod.SupportedGemProperty{Keyword: "active_skill", Key: "level", Value: 0}).KeywordFlag(mod.KeywordFlagCurse)},
	},
	//Strike
	"supported_strike_skill_gem_level_+": {
		Mods: []mod.Mod{mod.NewList("SupportedGemProperty", mod.SupportedGemProperty{Keyword: "active_skill", Key: "level", Value: 0}).Tag(mod.SkillType("MeleeSingleTarget"))},
	},
	//Elemental
	"supported_elemental_skill_gem_level_+": {
		Mods: []mod.Mod{mod.NewList("SupportedGemProperty", mod.SupportedGemProperty{Keyword: "active_skill", Key: "level", Value: 0}).KeywordFlag(mod.KeywordFlagLightning | mod.KeywordFlagCold | mod.KeywordFlagFire)},
	},
	//Minion
	"supported_minion_skill_gem_level_+": {
		Mods: []mod.Mod{mod.NewList("SupportedGemProperty", mod.SupportedGemProperty{Keyword: "active_skill", Key: "level", Value: 0}).Tag(mod.SkillType("Minion"))},
	},
}
