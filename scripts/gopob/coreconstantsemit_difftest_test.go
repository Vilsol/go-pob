package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const ccnPy = "oracle/core_constants_emit.py"

func ccnDump() string {
	return `{
	  "pobVersion":"2.67.2",
	  "tables":{
	    "mapLevelLifeMult":{
	      "exact":{
	        "66":"1.01","67":"1.03","68":"1.05","69":"1.0900000000000001","70":"1.1299999999999999",
	        "71":"1.1899999999999999","72":"1.24","73":"1.3","74":"1.3600000000000001","75":"1.43",
	        "76":"1.49","77":"1.55","78":"1.6100000000000001","79":"1.6799999999999999","80":"1.77",
	        "81":"1.8700000000000001","82":"1.98","83":"2.0899999999999999","84":"2.1699999999999999",
	        "85":"2.25","86":"2.3399999999999999","87":"2.4199999999999999","88":"2.5",
	        "89":"2.5800000000000001","90":"2.6600000000000001"
	      },
	      "values":{
	        "66":1.01,"67":1.03,"68":1.05,"69":1.09,"70":1.13,"71":1.19,"72":1.24,"73":1.3,
	        "74":1.36,"75":1.43,"76":1.49,"77":1.55,"78":1.61,"79":1.68,"80":1.77,"81":1.87,
	        "82":1.98,"83":2.09,"84":2.17,"85":2.25,"86":2.34,"87":2.42,"88":2.5,"89":2.58,
	        "90":2.66
	      }
	    },
	    "bossStats":{
	      "dropped":[],
	      "exact":{
	        "PinnacleArmourMean":"150",
	        "PinnacleEvasionMean":"124.90909090909091",
	        "UberArmourMean":"125",
	        "UberEvasionMean":"116.57142857142857"
	      },
	      "values":{
	        "PinnacleArmourMean":150,
	        "PinnacleEvasionMean":124.90909090909,
	        "UberArmourMean":125,
	        "UberEvasionMean":116.57142857143
	      }
	    },
	    "characterConstants":{
	      "dropped":[],
	      "exact":{
	        "accuracy_rating_per_level":"2",
	        "actor_scale_+%_limit":"100",
	        "additional_insanity_effects_while_delirious":"1",
	        "ailment_damage_+%_final_per_affliction_charge":"8",
	        "base_attack_speed_+%_per_frenzy_charge":"4",
	        "base_cast_speed_+%_per_frenzy_charge":"4",
	        "base_critical_strike_multiplier":"150",
	        "base_evasion_rating":"15",
	        "base_max_fortification":"20",
	        "base_maximum_all_resistances_%":"75",
	        "base_maximum_spell_block_%":"75",
	        "base_number_of_remote_mines_allowed":"15",
	        "base_number_of_totems_allowed":"1",
	        "base_number_of_traps_allowed":"15",
	        "base_presence_radius":"80",
	        "base_speed":"37",
	        "base_total_number_of_sigils_allowed":"3",
	        "chance_to_deal_triple_damage_%_per_brutal_charge":"3",
	        "critical_ailment_dot_multiplier_+":"50",
	        "critical_strike_chance_+%_per_power_charge":"50",
	        "damage_+%_per_10_rampage_stacks":"2",
	        "damage_taken_when_hit_+%_final_per_fortification":"-1",
	        "dexterity_per_level":"0",
	        "dual_wield_inherent_attack_speed_+%_final":"10",
	        "elemental_damage_reduction_%_per_endurance_charge":"4",
	        "elemental_damage_taken_goes_to_energy_shield_over_4_seconds_%_per_absorption_charge":"12",
	        "enable_movement_skill_animation_skipping":"1",
	        "energy_shield_recharge_rate_per_minute_%":"2000",
	        "impaled_debuff_base_duration_ms":"8000",
	        "impaled_debuff_number_of_reflected_hits":"5",
	        "inherent_block_while_dual_wielding_%":"20",
	        "intelligence_per_level":"0",
	        "is_player":"1",
	        "level":"1",
	        "life_per_level":"12",
	        "mana_per_level":"6",
	        "mana_regeneration_rate_per_minute_%":"105",
	        "max_azurite_debuff_stacks":"10",
	        "max_charged_attack_stacks":"6",
	        "max_delve_degen_stacks":"5000",
	        "max_endurance_charges":"3",
	        "max_fire_beam_stacks":"8",
	        "max_frenzy_charges":"3",
	        "max_frost_nova_stacks":"20",
	        "max_fuse_arrow_orbs":"5",
	        "max_power_charges":"3",
	        "max_rampage_stacks":"1000",
	        "max_steel_ammo":"12",
	        "max_talisman_degen_stacks":"20",
	        "max_viper_strike_orbs":"4",
	        "maximum_active_sand_mirage_count":"3",
	        "maximum_block_%":"75",
	        "maximum_blood_scythe_charges":"5",
	        "maximum_energy_shield_leech_amount_per_leech_%_max_energy_shield":"10",
	        "maximum_energy_shield_leech_rate_%_per_minute":"600",
	        "maximum_life_leech_amount_per_leech_%_max_life":"10",
	        "maximum_life_leech_rate_%_per_minute":"1200",
	        "maximum_mana_leech_amount_per_leech_%_max_mana":"10",
	        "maximum_mana_leech_rate_%_per_minute":"1200",
	        "maximum_physical_damage_reduction_%":"90",
	        "maximum_rage":"30",
	        "maximum_righteous_charges":"5",
	        "melee_hit_damage_stun_multiplier_+%":"25",
	        "melee_variation":"1",
	        "mines_invulnerable_for_duration_ms":"2000",
	        "minion_damage_taken_+%_from_arrow_traps_final":"0",
	        "minion_damage_taken_+%_from_guillotine_traps_final":"-90",
	        "minion_damage_taken_+%_from_spike_traps_final":"-90",
	        "minion_global_always_hit":"1",
	        "minions_are_immune_to_labyrinth_degen_effect":"0",
	        "minions_have_labyrinth_trap_degen_effect_+%":"-90",
	        "movement_velocity_+%_per_10_rampage_stacks":"1",
	        "mtx_max_killcounter_stacks":"30000",
	        "non_damaging_ailment_effect_+%_final_per_affliction_charge":"8",
	        "non_melee_hit_damage_stun_multiplier_+%_final":"-25",
	        "non_physical_hit_damage_stun_multiplier_+%":"-25",
	        "object_inherent_damage_+%_final_per_frenzy_charge":"4",
	        "object_inherent_melee_hit_stun_duration_+%_final":"50",
	        "physical_damage_%_to_add_as_cold_per_brine_charge":"4",
	        "physical_damage_%_to_add_as_lightning_per_brine_charge":"4",
	        "physical_damage_reduction_%_per_endurance_charge":"4",
	        "pvp_shield_damage_+%_final":"-15",
	        "rage_loss_delay_ms":"2000",
	        "soul_eater_maximum_stacks":"45",
	        "strength_per_level":"0",
	        "stun_threshold_+%_per_brutal_charge":"10",
	        "traps_explode_on_timeout":"1",
	        "traps_invulnerable_for_duration_ms":"2000",
	        "trigger_attack_graft_each_second_channeling":"1"
	      },
	      "values":{
	        "accuracy_rating_per_level":2,
	        "actor_scale_+%_limit":100,
	        "additional_insanity_effects_while_delirious":1,
	        "ailment_damage_+%_final_per_affliction_charge":8,
	        "base_attack_speed_+%_per_frenzy_charge":4,
	        "base_cast_speed_+%_per_frenzy_charge":4,
	        "base_critical_strike_multiplier":150,
	        "base_evasion_rating":15,
	        "base_max_fortification":20,
	        "base_maximum_all_resistances_%":75,
	        "base_maximum_spell_block_%":75,
	        "base_number_of_remote_mines_allowed":15,
	        "base_number_of_totems_allowed":1,
	        "base_number_of_traps_allowed":15,
	        "base_presence_radius":80,
	        "base_speed":37,
	        "base_total_number_of_sigils_allowed":3,
	        "chance_to_deal_triple_damage_%_per_brutal_charge":3,
	        "critical_ailment_dot_multiplier_+":50,
	        "critical_strike_chance_+%_per_power_charge":50,
	        "damage_+%_per_10_rampage_stacks":2,
	        "damage_taken_when_hit_+%_final_per_fortification":-1,
	        "dexterity_per_level":0,
	        "dual_wield_inherent_attack_speed_+%_final":10,
	        "elemental_damage_reduction_%_per_endurance_charge":4,
	        "elemental_damage_taken_goes_to_energy_shield_over_4_seconds_%_per_absorption_charge":12,
	        "enable_movement_skill_animation_skipping":1,
	        "energy_shield_recharge_rate_per_minute_%":2000,
	        "impaled_debuff_base_duration_ms":8000,
	        "impaled_debuff_number_of_reflected_hits":5,
	        "inherent_block_while_dual_wielding_%":20,
	        "intelligence_per_level":0,
	        "is_player":1,
	        "level":1,
	        "life_per_level":12,
	        "mana_per_level":6,
	        "mana_regeneration_rate_per_minute_%":105,
	        "max_azurite_debuff_stacks":10,
	        "max_charged_attack_stacks":6,
	        "max_delve_degen_stacks":5000,
	        "max_endurance_charges":3,
	        "max_fire_beam_stacks":8,
	        "max_frenzy_charges":3,
	        "max_frost_nova_stacks":20,
	        "max_fuse_arrow_orbs":5,
	        "max_power_charges":3,
	        "max_rampage_stacks":1000,
	        "max_steel_ammo":12,
	        "max_talisman_degen_stacks":20,
	        "max_viper_strike_orbs":4,
	        "maximum_active_sand_mirage_count":3,
	        "maximum_block_%":75,
	        "maximum_blood_scythe_charges":5,
	        "maximum_energy_shield_leech_amount_per_leech_%_max_energy_shield":10,
	        "maximum_energy_shield_leech_rate_%_per_minute":600,
	        "maximum_life_leech_amount_per_leech_%_max_life":10,
	        "maximum_life_leech_rate_%_per_minute":1200,
	        "maximum_mana_leech_amount_per_leech_%_max_mana":10,
	        "maximum_mana_leech_rate_%_per_minute":1200,
	        "maximum_physical_damage_reduction_%":90,
	        "maximum_rage":30,
	        "maximum_righteous_charges":5,
	        "melee_hit_damage_stun_multiplier_+%":25,
	        "melee_variation":1,
	        "mines_invulnerable_for_duration_ms":2000,
	        "minion_damage_taken_+%_from_arrow_traps_final":0,
	        "minion_damage_taken_+%_from_guillotine_traps_final":-90,
	        "minion_damage_taken_+%_from_spike_traps_final":-90,
	        "minion_global_always_hit":1,
	        "minions_are_immune_to_labyrinth_degen_effect":0,
	        "minions_have_labyrinth_trap_degen_effect_+%":-90,
	        "movement_velocity_+%_per_10_rampage_stacks":1,
	        "mtx_max_killcounter_stacks":30000,
	        "non_damaging_ailment_effect_+%_final_per_affliction_charge":8,
	        "non_melee_hit_damage_stun_multiplier_+%_final":-25,
	        "non_physical_hit_damage_stun_multiplier_+%":-25,
	        "object_inherent_damage_+%_final_per_frenzy_charge":4,
	        "object_inherent_melee_hit_stun_duration_+%_final":50,
	        "physical_damage_%_to_add_as_cold_per_brine_charge":4,
	        "physical_damage_%_to_add_as_lightning_per_brine_charge":4,
	        "physical_damage_reduction_%_per_endurance_charge":4,
	        "pvp_shield_damage_+%_final":-15,
	        "rage_loss_delay_ms":2000,
	        "soul_eater_maximum_stacks":45,
	        "strength_per_level":0,
	        "stun_threshold_+%_per_brutal_charge":10,
	        "traps_explode_on_timeout":1,
	        "traps_invulnerable_for_duration_ms":2000,
	        "trigger_attack_graft_each_second_channeling":1
	      }
	    },
	    "jewelRadius":{
	      "rows":[{
	          "dropped":[],
	          "exact":{
	            "inner":"0",
	            "innerSquared":"0",
	            "outer":"960",
	            "outerSquared":"921600"
	          },
	          "values":{
	            "col":"^xBB6600",
	            "inner":0,
	            "innerSquared":0,
	            "label":"Small",
	            "outer":960,
	            "outerSquared":921600
	          }
	        },{
	          "dropped":[],
	          "exact":{
	            "inner":"0",
	            "innerSquared":"0",
	            "outer":"1440",
	            "outerSquared":"2073600"
	          },
	          "values":{
	            "col":"^x66FFCC",
	            "inner":0,
	            "innerSquared":0,
	            "label":"Medium",
	            "outer":1440,
	            "outerSquared":2073600
	          }
	        },{
	          "dropped":[],
	          "exact":{
	            "inner":"0",
	            "innerSquared":"0",
	            "outer":"1800",
	            "outerSquared":"3240000"
	          },
	          "values":{
	            "col":"^x2222CC",
	            "inner":0,
	            "innerSquared":0,
	            "label":"Large",
	            "outer":1800,
	            "outerSquared":3240000
	          }
	        },{
	          "dropped":[],
	          "exact":{
	            "inner":"0",
	            "innerSquared":"0",
	            "outer":"2400",
	            "outerSquared":"5760000"
	          },
	          "values":{
	            "col":"^xC100FF",
	            "inner":0,
	            "innerSquared":0,
	            "label":"Very Large",
	            "outer":2400,
	            "outerSquared":5760000
	          }
	        },{
	          "dropped":[],
	          "exact":{
	            "inner":"0",
	            "innerSquared":"0",
	            "outer":"2880",
	            "outerSquared":"8294400"
	          },
	          "values":{
	            "col":"^x0B9300",
	            "inner":0,
	            "innerSquared":0,
	            "label":"Massive",
	            "outer":2880,
	            "outerSquared":8294400
	          }
	        },{
	          "dropped":[],
	          "exact":{
	            "inner":"960",
	            "innerSquared":"921600",
	            "outer":"1320",
	            "outerSquared":"1742400"
	          },
	          "values":{
	            "col":"^xD35400",
	            "inner":960,
	            "innerSquared":921600,
	            "label":"Variable",
	            "outer":1320,
	            "outerSquared":1742400
	          }
	        },{
	          "dropped":[],
	          "exact":{
	            "inner":"1320",
	            "innerSquared":"1742400",
	            "outer":"1680",
	            "outerSquared":"2822400"
	          },
	          "values":{
	            "col":"^x66FFCC",
	            "inner":1320,
	            "innerSquared":1742400,
	            "label":"Variable",
	            "outer":1680,
	            "outerSquared":2822400
	          }
	        },{
	          "dropped":[],
	          "exact":{
	            "inner":"1680",
	            "innerSquared":"2822400",
	            "outer":"2040",
	            "outerSquared":"4161600"
	          },
	          "values":{
	            "col":"^x2222CC",
	            "inner":1680,
	            "innerSquared":2822400,
	            "label":"Variable",
	            "outer":2040,
	            "outerSquared":4161600
	          }
	        },{
	          "dropped":[],
	          "exact":{
	            "inner":"2040",
	            "innerSquared":"4161600",
	            "outer":"2400",
	            "outerSquared":"5760000"
	          },
	          "values":{
	            "col":"^xC100FF",
	            "inner":2040,
	            "innerSquared":4161600,
	            "label":"Variable",
	            "outer":2400,
	            "outerSquared":5760000
	          }
	        },{
	          "dropped":[],
	          "exact":{
	            "inner":"2400",
	            "innerSquared":"5760000",
	            "outer":"2880",
	            "outerSquared":"8294400"
	          },
	          "values":{
	            "col":"^x0B9300",
	            "inner":2400,
	            "innerSquared":5760000,
	            "label":"Variable",
	            "outer":2880,
	            "outerSquared":8294400
	          }
	        }]
	    },
	    "keystones":{
	      "strings":["Acrobatics","Ancestral Bond","Arrow Dancing","Arsenal of Vengeance","Avatar of Fire","Bitter Frost","Blood Magic","Bloodsoaked Blade","Call to Arms","Chaos Inoculation","Conduit","Corrupted Soul","Crimson Dance","Divine Flesh","Divine Shield","Doomsday","Eldritch Battery","Elemental Equilibrium","Elemental Overload","Eternal Youth","Ghost Dance","Ghost Reaver","Glancing Blows","Hex Master","Imbalanced Guard","Immortal Ambition","Inner Conviction","Iron Grip","Iron Reflexes","Iron Will","Lethe Shade","Magebane","Mind Over Matter","Minion Instability","Mortal Conviction","Necromantic Aegis","Pain Attunement","Perfect Agony","Phase Acrobatics","Point Blank","Power of Purpose","Precise Technique","Resolute Technique","Roiling Tempest","Runebinder","Solipsism","Supreme Decadence","Supreme Ego","The Agnostic","The Impaler","Transcendence","Unwavering Stance","Vaal Pact","Versatile Combatant","Voracious Flame","Wicked Ward","Wind Dancer","Zealot's Oath"]
	    },
	    "misc":{
	      "dropped":[],
	      "exact":{
	        "AccuracyPerDexBase":"2",
	        "AvoidChanceCap":"75",
	        "BleedDurationBase":"5",
	        "BleedPercentBase":"70",
	        "BlockChanceCap":"90",
	        "BrandAttachmentRangeBase":"30",
	        "BuffExpirationSlowCap":"0.25",
	        "DamageReductionCap":"90",
	        "DodgeChanceCap":"75",
	        "DotDpsCap":"35791394",
	        "EnemyMaxResist":"75",
	        "EnemyPhysicalDamageReductionCap":"75",
	        "EnergyShieldRechargeBase":"0.33333333333333337",
	        "EnergyShieldRechargeDelay":"2",
	        "EvadeChanceCap":"95",
	        "FortifyBaseDuration":"6",
	        "IgniteDurationBase":"4",
	        "IgnitePercentBase":"0.90000000000000002",
	        "ImpaleStoredDamageBase":"0.10000000000000001",
	        "LeechRateBase":"0.02",
	        "LowPoolThreshold":"0.5",
	        "ManaRegenBase":"0.017500000000000002",
	        "MatchingSocketQualityBonus":"10",
	        "MaxEnemyLevel":"85",
	        "MaxResistCap":"90",
	        "MinStunChanceNeeded":"20",
	        "MineAuraRadiusBase":"35",
	        "MineDetonationRadiusBase":"60",
	        "PlayerMovementSpeed":"37",
	        "PoisonDurationBase":"2",
	        "PoisonPercentBase":"0.29999999999999999",
	        "ProjectileDistanceCap":"150",
	        "PvpElemental1":"0.55000000000000004",
	        "PvpElemental2":"150",
	        "PvpNonElemental1":"0.56999999999999995",
	        "PvpNonElemental2":"90",
	        "ResistFloor":"-200",
	        "ServerTickRate":"30.303030303030301",
	        "ServerTickTime":"0.033000000000000002",
	        "StunBaseDuration":"0.34999999999999998",
	        "StunBaseMult":"200",
	        "StunNotMeleeDamageMult":"0.75",
	        "SuppressionChanceCap":"100",
	        "SuppressionEffect":"40",
	        "TemporalChainsEffectCap":"75",
	        "Transfiguration":"0.29999999999999999",
	        "TrapTriggerRadiusBase":"10",
	        "WardRechargeDelay":"2",
	        "ehpCalcMaxDamage":"100000000",
	        "ehpCalcMaxIterationsToCalc":"50",
	        "ehpCalcSpeedUp":"8",
	        "experiencePenaltyMultiplier":"0.059999999999999998",
	        "maxExperiencePenaltyFreeAreaLevel":"70",
	        "maxHitSmoothingPasses":"8",
	        "maxStatIncrease":"2",
	        "pinnacleBossDPSMult":"1.8181818181818181",
	        "pinnacleBossPen":"3",
	        "stdBossDPSMult":"0.90909090909090906",
	        "uberBossDPSMult":"2.3529411764705883",
	        "uberBossPen":"8"
	      },
	      "values":{
	        "AccuracyPerDexBase":2,
	        "AvoidChanceCap":75,
	        "BleedDurationBase":5,
	        "BleedPercentBase":70,
	        "BlockChanceCap":90,
	        "BrandAttachmentRangeBase":30,
	        "BuffExpirationSlowCap":0.25,
	        "DamageReductionCap":90,
	        "DodgeChanceCap":75,
	        "DotDpsCap":35791394,
	        "EnemyMaxResist":75,
	        "EnemyPhysicalDamageReductionCap":75,
	        "EnergyShieldRechargeBase":0.33333333333333,
	        "EnergyShieldRechargeDelay":2,
	        "EvadeChanceCap":95,
	        "FortifyBaseDuration":6,
	        "IgniteDurationBase":4,
	        "IgnitePercentBase":0.9,
	        "ImpaleStoredDamageBase":0.1,
	        "LeechRateBase":0.02,
	        "LowPoolThreshold":0.5,
	        "ManaRegenBase":0.0175,
	        "MatchingSocketQualityBonus":10,
	        "MaxEnemyLevel":85,
	        "MaxResistCap":90,
	        "MinStunChanceNeeded":20,
	        "MineAuraRadiusBase":35,
	        "MineDetonationRadiusBase":60,
	        "PlayerMovementSpeed":37,
	        "PoisonDurationBase":2,
	        "PoisonPercentBase":0.3,
	        "ProjectileDistanceCap":150,
	        "PvpElemental1":0.55,
	        "PvpElemental2":150,
	        "PvpNonElemental1":0.57,
	        "PvpNonElemental2":90,
	        "ResistFloor":-200,
	        "ServerTickRate":30.30303030303,
	        "ServerTickTime":0.033,
	        "StunBaseDuration":0.35,
	        "StunBaseMult":200,
	        "StunNotMeleeDamageMult":0.75,
	        "SuppressionChanceCap":100,
	        "SuppressionEffect":40,
	        "TemporalChainsEffectCap":75,
	        "Transfiguration":0.3,
	        "TrapTriggerRadiusBase":10,
	        "WardRechargeDelay":2,
	        "ehpCalcMaxDamage":100000000,
	        "ehpCalcMaxIterationsToCalc":50,
	        "ehpCalcSpeedUp":8,
	        "experiencePenaltyMultiplier":0.06,
	        "maxExperiencePenaltyFreeAreaLevel":70,
	        "maxHitSmoothingPasses":8,
	        "maxStatIncrease":2,
	        "pinnacleBossDPSMult":1.8181818181818,
	        "pinnacleBossPen":3,
	        "stdBossDPSMult":0.90909090909091,
	        "uberBossDPSMult":2.3529411764706,
	        "uberBossPen":8
	      }
	    },
	    "monsterAccuracyTable":{
	      "exact":["14","15","15","16","17","18","19","20","21","23","24","25","26","28","29","31","32","34","35","37","39","41","43","45","47","49","52","54","57","59","62","65","68","71","74","77","81","84","88","92","96","100","105","109","114","119","124","129","135","140","146","152","159","165","172","179","187","195","203","211","220","229","238","247","257","268","279","290","301","314","326","339","352","366","381","396","412","428","444","462","480","499","518","538","559","580","603","626","650","675","701","728","755","784","814","845","877","910","945","980"],
	      "values":[14,15,15,16,17,18,19,20,21,23,24,25,26,28,29,31,32,34,35,37,39,41,43,45,47,49,52,54,57,59,62,65,68,71,74,77,81,84,88,92,96,100,105,109,114,119,124,129,135,140,146,152,159,165,172,179,187,195,203,211,220,229,238,247,257,268,279,290,301,314,326,339,352,366,381,396,412,428,444,462,480,499,518,538,559,580,603,626,650,675,701,728,755,784,814,845,877,910,945,980]
	    },
	    "monsterAllyLifeTable":{
	      "exact":["15","16","18","20","22","24","27","29","32","35","38","41","44","48","52","56","60","65","70","75","81","87","93","99","106","114","121","130","138","148","158","168","179","190","203","215","229","244","259","275","292","310","328","348","369","392","415","440","465","493","522","552","584","618","653","691","730","772","816","862","910","961","1015","1072","1131","1194","1260","1329","1402","1478","1559","1644","1733","1827","1926","2029","2138","2253","2373","2500","2633","2773","2919","3074","3236","3406","3585","3773","3970","4178","4395","4624","4864","5116","5381","5659","5951","6257","6578","6916"],
	      "values":[15,16,18,20,22,24,27,29,32,35,38,41,44,48,52,56,60,65,70,75,81,87,93,99,106,114,121,130,138,148,158,168,179,190,203,215,229,244,259,275,292,310,328,348,369,392,415,440,465,493,522,552,584,618,653,691,730,772,816,862,910,961,1015,1072,1131,1194,1260,1329,1402,1478,1559,1644,1733,1827,1926,2029,2138,2253,2373,2500,2633,2773,2919,3074,3236,3406,3585,3773,3970,4178,4395,4624,4864,5116,5381,5659,5951,6257,6578,6916]
	    },
	    "monsterAllyDamageTable":{
	      "exact":["5.6199998855590998","6.0300002098082999","6.4600000381470002","6.9200000762939","7.4099998474120996","7.9299998283386","8.4799995422362997","9.0600004196166992","9.6800003051758008","10.329999923706","11.029999732971","11.760000228881999","12.539999961853001","13.369999885559","14.25","15.170000076294","16.159999847411999","17.200000762938998","18.299999237061002","19.459999084473001","20.700000762938998","22","23.389999389648001","24.85000038147","26.389999389648001","28.030000686646002","29.760000228881999","31.579999923706001","33.520000457763999","35.560001373291001","37.720001220702997","40","42.409999847412003","44.959999084472997","47.639999389647997","50.490001678467003","53.490001678467003","56.659999847412003","60","63.529998779297003","67.26000213623","71.199996948242003","75.360000610352003","79.73999786377","84.370002746582003","89.25","94.400001525879006","99.839996337890994","105.56999969482","111.62000274658","118","124.73000335693","131.83000183105","139.30999755859","147.19999694824","155.52000427246","164.2799987793","173.5299987793","183.27000427246","193.53999328613","204.36999511719","215.7799987793","227.80000305176","240.46000671387","253.80999755859","267.86999511719","282.69000244141","298.29000854492","314.73001098633","332.04998779297","350.29000854492","369.5","389.73001098633","411.04000854492","433.4700012207","457.08999633789","481.9700012207","508.14999389648","535.71997070312","564.75","595.29998779297","627.46002197266","661.30999755859","696.95001220703","734.45001220703","773.90997314453","815.45001220703","859.15997314453","905.15002441406","953.53997802734","1004.4699707031","1058.0400390625","1114.4100341797","1173.7099609375","1236.0999755859","1301.7299804688","1370.7600097656","1443.3800048828","1519.7600097656","1600.0899658203"],
	      "values":[5.6199998855591,6.0300002098083,6.460000038147,6.9200000762939,7.4099998474121,7.9299998283386,8.4799995422363,9.0600004196167,9.6800003051758,10.329999923706,11.029999732971,11.760000228882,12.539999961853,13.369999885559,14.25,15.170000076294,16.159999847412,17.200000762939,18.299999237061,19.459999084473,20.700000762939,22,23.389999389648,24.85000038147,26.389999389648,28.030000686646,29.760000228882,31.579999923706,33.520000457764,35.560001373291,37.720001220703,40,42.409999847412,44.959999084473,47.639999389648,50.490001678467,53.490001678467,56.659999847412,60,63.529998779297,67.26000213623,71.199996948242,75.360000610352,79.73999786377,84.370002746582,89.25,94.400001525879,99.839996337891,105.56999969482,111.62000274658,118,124.73000335693,131.83000183105,139.30999755859,147.19999694824,155.52000427246,164.2799987793,173.5299987793,183.27000427246,193.53999328613,204.36999511719,215.7799987793,227.80000305176,240.46000671387,253.80999755859,267.86999511719,282.69000244141,298.29000854492,314.73001098633,332.04998779297,350.29000854492,369.5,389.73001098633,411.04000854492,433.4700012207,457.08999633789,481.9700012207,508.14999389648,535.71997070312,564.75,595.29998779297,627.46002197266,661.30999755859,696.95001220703,734.45001220703,773.90997314453,815.45001220703,859.15997314453,905.15002441406,953.53997802734,1004.4699707031,1058.0400390625,1114.4100341797,1173.7099609375,1236.0999755859,1301.7299804688,1370.7600097656,1443.3800048828,1519.7600097656,1600.0899658203]
	    },
	    "monsterArmourTable":{
	      "exact":["12","15","19","23","27","32","37","43","50","57","65","74","83","94","105","118","132","147","164","182","202","224","248","275","303","334","368","405","445","489","537","589","646","707","774","846","925","1010","1103","1204","1313","1432","1560","1700","1850","2014","2191","2383","2591","2815","3059","3322","3607","3915","4248","4608","4997","5418","5873","6365","6896","7469","8089","8757","9480","10259","11101","12009","12989","14047","15188","16419","17747","19178","20722","22387","24182","26117","28203","30451","32873","35483","38296","41326","44591","48107","51894","55973","60365","65095","70188","75670","81573","87926","94765","102125","110047","118571","127744","137613"],
	      "values":[12,15,19,23,27,32,37,43,50,57,65,74,83,94,105,118,132,147,164,182,202,224,248,275,303,334,368,405,445,489,537,589,646,707,774,846,925,1010,1103,1204,1313,1432,1560,1700,1850,2014,2191,2383,2591,2815,3059,3322,3607,3915,4248,4608,4997,5418,5873,6365,6896,7469,8089,8757,9480,10259,11101,12009,12989,14047,15188,16419,17747,19178,20722,22387,24182,26117,28203,30451,32873,35483,38296,41326,44591,48107,51894,55973,60365,65095,70188,75670,81573,87926,94765,102125,110047,118571,127744,137613]
	    },
	    "monsterDamageTable":{
	      "exact":["4.9899997711181996","5.5599999427795002","6.1599998474120996","6.8099999427795002","7.5","8.2299995422362997","9","9.8199996948241992","10.699999809265","11.619999885559","12.60000038147","13.640000343323001","14.739999771118001","15.909999847411999","17.139999389648001","18.450000762938998","19.829999923706001","21.290000915526999","22.840000152588001","24.469999313353998","26.190000534058001","28.010000228881999","29.940000534058001","31.959999084473001","34.110000610352003","36.360000610352003","38.75","41.259998321532997","43.909999847412003","46.700000762938998","49.650001525878999","52.75","56.009998321532997","59.450000762938998","63.080001831055","66.889999389647997","70.910003662109006","75.129997253417997","79.580001831055","84.26000213623","89.180000305175994","94.349998474120994","99.800003051757997","105.51999664307","111.5299987793","117.86000061035","124.5","131.49000549316","138.83000183105","146.5299987793","154.63000488281","163.13999938965","172.07000732422","181.44999694824","191.30000305176","201.63000488281","212.47999572754","223.86999511719","235.83000183105","248.36999511719","261.5299987793","275.32998657227","289.82000732422","305.01000976562","320.94000244141","337.64999389648","355.17999267578","373.54998779297","392.80999755859","413.01000976562","434.17999267578","456.36999511719","479.61999511719","504","529.53997802734","556.29998779297","584.34997558594","613.72998046875","644.5","676.75","710.52001953125","745.89001464844","782.94000244141","821.72998046875","862.35998535156","904.90002441406","949.44000244141","996.07000732422","1044.8900146484","1096","1149.5","1205.5","1264.1099853516","1325.4499511719","1389.6400146484","1456.8199462891","1527.1199951172","1600.6800537109","1677.6400146484","1758.1700439453"],
	      "values":[4.9899997711182,5.5599999427795,6.1599998474121,6.8099999427795,7.5,8.2299995422363,9,9.8199996948242,10.699999809265,11.619999885559,12.60000038147,13.640000343323,14.739999771118,15.909999847412,17.139999389648,18.450000762939,19.829999923706,21.290000915527,22.840000152588,24.469999313354,26.190000534058,28.010000228882,29.940000534058,31.959999084473,34.110000610352,36.360000610352,38.75,41.259998321533,43.909999847412,46.700000762939,49.650001525879,52.75,56.009998321533,59.450000762939,63.080001831055,66.889999389648,70.910003662109,75.129997253418,79.580001831055,84.26000213623,89.180000305176,94.349998474121,99.800003051758,105.51999664307,111.5299987793,117.86000061035,124.5,131.49000549316,138.83000183105,146.5299987793,154.63000488281,163.13999938965,172.07000732422,181.44999694824,191.30000305176,201.63000488281,212.47999572754,223.86999511719,235.83000183105,248.36999511719,261.5299987793,275.32998657227,289.82000732422,305.01000976562,320.94000244141,337.64999389648,355.17999267578,373.54998779297,392.80999755859,413.01000976562,434.17999267578,456.36999511719,479.61999511719,504,529.53997802734,556.29998779297,584.34997558594,613.72998046875,644.5,676.75,710.52001953125,745.89001464844,782.94000244141,821.72998046875,862.35998535156,904.90002441406,949.44000244141,996.07000732422,1044.8900146484,1096,1149.5,1205.5,1264.1099853516,1325.4499511719,1389.6400146484,1456.8199462891,1527.1199951172,1600.6800537109,1677.6400146484,1758.1700439453]
	    },
	    "monsterEvasionTable":{
	      "exact":["67","86","104","124","144","166","188","211","234","259","285","311","339","368","397","428","460","493","527","563","600","638","677","718","760","804","849","896","944","994","1046","1100","1155","1212","1271","1332","1395","1460","1528","1597","1669","1743","1819","1898","1979","2063","2150","2239","2331","2426","2524","2626","2730","2837","2948","3063","3180","3302","3427","3556","3689","3826","3967","4112","4262","4416","4575","4739","4907","5081","5260","5444","5633","5828","6029","6235","6448","6667","6892","7124","7362","7608","7860","8120","8388","8663","8946","9237","9536","9844","10160","10486","10821","11165","11519","11883","12258","12643","13038","13445"],
	      "values":[67,86,104,124,144,166,188,211,234,259,285,311,339,368,397,428,460,493,527,563,600,638,677,718,760,804,849,896,944,994,1046,1100,1155,1212,1271,1332,1395,1460,1528,1597,1669,1743,1819,1898,1979,2063,2150,2239,2331,2426,2524,2626,2730,2837,2948,3063,3180,3302,3427,3556,3689,3826,3967,4112,4262,4416,4575,4739,4907,5081,5260,5444,5633,5828,6029,6235,6448,6667,6892,7124,7362,7608,7860,8120,8388,8663,8946,9237,9536,9844,10160,10486,10821,11165,11519,11883,12258,12643,13038,13445]
	    },
	    "monsterLifeTable":{
	      "exact":["22","26","31","36","42","48","55","62","70","78","87","97","107","119","131","144","158","173","190","207","226","246","267","290","315","341","370","400","432","467","504","543","585","630","678","730","785","843","905","972","1042","1118","1198","1284","1375","1472","1575","1685","1802","1927","2059","2200","2350","2509","2678","2858","3050","3253","3469","3698","3942","4201","4476","4768","5078","5407","5756","6127","6520","6937","7380","7850","8348","8876","9436","10030","10660","11328","12036","12787","13582","14425","15319","16265","17268","18331","19457","20649","21913","23250","24667","26168","27756","29438","31220","33105","35101","37214","39450","41817"],
	      "values":[22,26,31,36,42,48,55,62,70,78,87,97,107,119,131,144,158,173,190,207,226,246,267,290,315,341,370,400,432,467,504,543,585,630,678,730,785,843,905,972,1042,1118,1198,1284,1375,1472,1575,1685,1802,1927,2059,2200,2350,2509,2678,2858,3050,3253,3469,3698,3942,4201,4476,4768,5078,5407,5756,6127,6520,6937,7380,7850,8348,8876,9436,10030,10660,11328,12036,12787,13582,14425,15319,16265,17268,18331,19457,20649,21913,23250,24667,26168,27756,29438,31220,33105,35101,37214,39450,41817]
	    },
	    "monsterLifeTable2":{
	      "exact":["10","12","15","18","21","25","29","34","39","44","50","56","63","70","79","87","97","107","119","131","144","158","174","191","209","228","249","272","296","323","351","382","415","450","489","530","574","621","672","727","786","850","917","990","1069","1153","1243","1339","1443","1554","1673","1800","1937","2083","2240","2408","2587","2780","2986","3206","3442","3694","3963","4252","4560","4890","5243","5620","6022","6453","6913","7404","7929","8490","9089","9729","10412","11141","11920","12751","13638","14585","15596","16675","17825","19053","20363","21760","23250","24840","26535","28343","30270","32326","34517","36853","39343","41997","44826","47841"],
	      "values":[10,12,15,18,21,25,29,34,39,44,50,56,63,70,79,87,97,107,119,131,144,158,174,191,209,228,249,272,296,323,351,382,415,450,489,530,574,621,672,727,786,850,917,990,1069,1153,1243,1339,1443,1554,1673,1800,1937,2083,2240,2408,2587,2780,2986,3206,3442,3694,3963,4252,4560,4890,5243,5620,6022,6453,6913,7404,7929,8490,9089,9729,10412,11141,11920,12751,13638,14585,15596,16675,17825,19053,20363,21760,23250,24840,26535,28343,30270,32326,34517,36853,39343,41997,44826,47841]
	    },
	    "monsterLifeTable3":{
	      "exact":["13","15","18","22","25","29","34","38","44","49","55","62","69","77","86","95","106","117","128","141","155","170","187","204","223","244","266","290","316","344","373","406","440","478","518","561","608","658","712","769","831","898","970","1046","1129","1217","1312","1414","1523","1640","1766","1900","2044","2199","2364","2541","2731","2934","3151","3384","3633","3900","4185","4490","4816","5165","5539","5938","6364","6820","7308","7829","8386","8980","9616","10295","11020","11795","12622","13506","14449","15456","16531","17679","18904","20211","21607","23096","24684","26380","28188","30117","32175","34370","36711","39207","41870","44708","47735","50962"],
	      "values":[13,15,18,22,25,29,34,38,44,49,55,62,69,77,86,95,106,117,128,141,155,170,187,204,223,244,266,290,316,344,373,406,440,478,518,561,608,658,712,769,831,898,970,1046,1129,1217,1312,1414,1523,1640,1766,1900,2044,2199,2364,2541,2731,2934,3151,3384,3633,3900,4185,4490,4816,5165,5539,5938,6364,6820,7308,7829,8386,8980,9616,10295,11020,11795,12622,13506,14449,15456,16531,17679,18904,20211,21607,23096,24684,26380,28188,30117,32175,34370,36711,39207,41870,44708,47735,50962]
	    },
	    "monsterPhysConversionMultiTable":{
      "exact":["0","0","0","0","0","0","0","0","0","0","0","0","0","0","0","0","0","0","0","0","5","10","15","20","25","30","35","40","45","50","55","60","65","70","75","80","85","90","95","100","105","110","115","120","125","130","135","140","145","150","155","160","165","170","175","180","185","190","195","200","205","210","215","220","225","230","235","240","245","250","255","260","265","270","275","280","285","290","295","300","300","300","300","300","300","300","300","300","300","300","300","300","300","300","300","300","300","300","300","300"],
      "values":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,5,10,15,20,25,30,35,40,45,50,55,60,65,70,75,80,85,90,95,100,105,110,115,120,125,130,135,140,145,150,155,160,165,170,175,180,185,190,195,200,205,210,215,220,225,230,235,240,245,250,255,260,265,270,275,280,285,290,295,300,300,300,300,300,300,300,300,300,300,300,300,300,300,300,300,300,300,300,300,300]
    },
    "nonDamagingAilment":{
	      "rows":{
	        "Brittle":{
	          "dropped":[],
	          "exact":{
	            "default":"2",
	            "duration":"4",
	            "max":"6",
	            "min":"0",
	            "precision":"2"
	          },
	          "values":{
	            "alt":true,
	            "associatedType":"Cold",
	            "default":2,
	            "duration":4,
	            "max":6,
	            "min":0,
	            "precision":2
	          }
	        },
	        "Chill":{
	          "dropped":[],
	          "exact":{
	            "default":"10",
	            "duration":"2",
	            "max":"30",
	            "min":"5",
	            "precision":"0"
	          },
	          "values":{
	            "alt":false,
	            "associatedType":"Cold",
	            "default":10,
	            "duration":2,
	            "max":30,
	            "min":5,
	            "precision":0
	          }
	        },
	        "Freeze":{
	          "dropped":[],
	          "exact":{
	            "max":"3",
	            "min":"0.29999999999999999",
	            "precision":"2"
	          },
	          "values":{
	            "alt":false,
	            "associatedType":"Cold",
	            "max":3,
	            "min":0.3,
	            "precision":2
	          }
	        },
	        "Sap":{
	          "dropped":[],
	          "exact":{
	            "default":"6",
	            "duration":"4",
	            "max":"20",
	            "min":"0",
	            "precision":"0"
	          },
	          "values":{
	            "alt":true,
	            "associatedType":"Lightning",
	            "default":6,
	            "duration":4,
	            "max":20,
	            "min":0,
	            "precision":0
	          }
	        },
	        "Scorch":{
	          "dropped":[],
	          "exact":{
	            "default":"10",
	            "duration":"4",
	            "max":"30",
	            "min":"0",
	            "precision":"0"
	          },
	          "values":{
	            "alt":true,
	            "associatedType":"Fire",
	            "default":10,
	            "duration":4,
	            "max":30,
	            "min":0,
	            "precision":0
	          }
	        },
	        "Shock":{
	          "dropped":[],
	          "exact":{
	            "default":"15",
	            "duration":"2",
	            "max":"50",
	            "min":"5",
	            "precision":"0"
	          },
	          "values":{
	            "alt":false,
	            "associatedType":"Lightning",
	            "default":15,
	            "duration":2,
	            "max":50,
	            "min":5,
	            "precision":0
	          }
	        }
	      }
	    },
	    "unarmedWeaponData":{
	      "rows":{
	        "0":{
	          "dropped":[],
	          "exact":{
	            "AttackRate":"1.2",
	            "CritChance":"0",
	            "PhysicalMax":"6",
	            "PhysicalMin":"2"
	          },
	          "values":{
	            "AttackRate":1.2,
	            "CritChance":0,
	            "PhysicalMax":6,
	            "PhysicalMin":2,
	            "type":"None"
	          }
	        },
	        "1":{
	          "dropped":[],
	          "exact":{
	            "AttackRate":"1.2",
	            "CritChance":"0",
	            "PhysicalMax":"8",
	            "PhysicalMin":"2"
	          },
	          "values":{
	            "AttackRate":1.2,
	            "CritChance":0,
	            "PhysicalMax":8,
	            "PhysicalMin":2,
	            "type":"None"
	          }
	        },
	        "2":{
	          "dropped":[],
	          "exact":{
	            "AttackRate":"1.2",
	            "CritChance":"0",
	            "PhysicalMax":"5",
	            "PhysicalMin":"2"
	          },
	          "values":{
	            "AttackRate":1.2,
	            "CritChance":0,
	            "PhysicalMax":5,
	            "PhysicalMin":2,
	            "type":"None"
	          }
	        },
	        "3":{
	          "dropped":[],
	          "exact":{
	            "AttackRate":"1.2",
	            "CritChance":"0",
	            "PhysicalMax":"5",
	            "PhysicalMin":"2"
	          },
	          "values":{
	            "AttackRate":1.2,
	            "CritChance":0,
	            "PhysicalMax":5,
	            "PhysicalMin":2,
	            "type":"None"
	          }
	        },
	        "4":{
	          "dropped":[],
	          "exact":{
	            "AttackRate":"1.2",
	            "CritChance":"0",
	            "PhysicalMax":"6",
	            "PhysicalMin":"2"
	          },
	          "values":{
	            "AttackRate":1.2,
	            "CritChance":0,
	            "PhysicalMax":6,
	            "PhysicalMin":2,
	            "type":"None"
	          }
	        },
	        "5":{
	          "dropped":[],
	          "exact":{
	            "AttackRate":"1.2",
	            "CritChance":"0",
	            "PhysicalMax":"6",
	            "PhysicalMin":"2"
	          },
	          "values":{
	            "AttackRate":1.2,
	            "CritChance":0,
	            "PhysicalMax":6,
	            "PhysicalMin":2,
	            "type":"None"
	          }
	        },
	        "6":{
	          "dropped":[],
	          "exact":{
	            "AttackRate":"1.2",
	            "CritChance":"0",
	            "PhysicalMax":"5",
	            "PhysicalMin":"2"
	          },
	          "values":{
	            "AttackRate":1.2,
	            "CritChance":0,
	            "PhysicalMax":5,
	            "PhysicalMin":2,
	            "type":"None"
	          }
	        }
	      }
	    },
	    "weaponTypeInfo":{
	      "rows":{
	        "Bow":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Bow",
	            "melee":false,
	            "oneHand":false
	          }
	        },
	        "Claw":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Claw",
	            "melee":true,
	            "oneHand":true
	          }
	        },
	        "Dagger":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Dagger",
	            "melee":true,
	            "oneHand":true
	          }
	        },
	        "Fishing Rod":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Fishing",
	            "melee":true,
	            "oneHand":false
	          }
	        },
	        "None":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Unarmed",
	            "melee":true,
	            "oneHand":true
	          }
	        },
	        "One Handed Axe":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Axe",
	            "melee":true,
	            "oneHand":true
	          }
	        },
	        "One Handed Mace":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Mace",
	            "melee":true,
	            "oneHand":true
	          }
	        },
	        "One Handed Sword":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Sword",
	            "melee":true,
	            "oneHand":true
	          }
	        },
	        "Sceptre":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Mace",
	            "label":"Sceptre",
	            "melee":true,
	            "oneHand":true
	          }
	        },
	        "Staff":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Staff",
	            "melee":true,
	            "oneHand":false
	          }
	        },
	        "Thrusting One Handed Sword":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Sword",
	            "label":"One Handed Sword",
	            "melee":true,
	            "oneHand":true
	          }
	        },
	        "Two Handed Axe":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Axe",
	            "melee":true,
	            "oneHand":false
	          }
	        },
	        "Two Handed Mace":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Mace",
	            "melee":true,
	            "oneHand":false
	          }
	        },
	        "Two Handed Sword":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Sword",
	            "melee":true,
	            "oneHand":false
	          }
	        },
	        "Wand":{
	          "dropped":[],
	          "exact":[],
	          "values":{
	            "flag":"Wand",
	            "melee":false,
	            "oneHand":true
	          }
	        }
	      }
	    }
	  },
	  "treeVersion":"3_29"
	}`
}

func ccnSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, ccnPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		for rel, b := range map[string]string{
			ccnPy: string(py), "dump.json": ccnDump(), "adir/keep": "a dir\n",
		} {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(b), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func ccnPut(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func ccnSpec(bin string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool: "core-constants-emit", PythonScript: ccnPy, PythonArgs: args,
		GoBin: bin, GoArgs: append([]string{"core-constants-emit"}, args...),
		Stderr:       difftest.StderrExact,
		Files:        []string{"out.go"},
		RefusalSites: 2,
	}
}

func TestCoreConstantsEmit(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := ccnSandbox(t, repo)
	pySrc := difftest.ReadOriginal(t, repo, ccnPy)

	spec := ccnSpec(bin, "dump.json", "out.go")
	spec.Fixtures = []difftest.Fixture{
		{Name: "happy", Passes: true},

		{Name: "no-args", Refusal: "usage: core_constants_emit.py <dump.json> <out.go>",
			Break: func(string) error { return nil }},
		{Name: "three-args", Refusal: "usage: core_constants_emit.py <dump.json> <out.go>",
			Break: func(string) error { return nil }},

		{Name: "dump-missing",
			Refusal: "REFUSING: the dump (dump.json) is not readable: No such file or directory",
			Break:   func(sb string) error { return os.Remove(filepath.Join(sb, "dump.json")) }},
		{Name: "dump-not-json", Refusal: "REFUSING: the dump (dump.json) is not valid JSON",
			Break: func(sb string) error { return ccnPut(sb, "dump.json", "{ nope") }},
		{Name: "dump-not-an-object", Refusal: "REFUSING: the dump (dump.json) is not a JSON object",
			Break: func(sb string) error { return ccnPut(sb, "dump.json", "[]") }},
		{Name: "tables-is-not-an-object",
			Refusal: "REFUSING: the dump (dump.json)'s 'tables' is not a JSON object",
			Break:   func(sb string) error { return ccnPut(sb, "dump.json", `{"tables": 7}`) }},
		{Name: "no-tables-at-all",
			Refusal: "REFUSING: the dump is missing tables this emitter needs:",
			Break:   func(sb string) error { return ccnPut(sb, "dump.json", `{}`) }},

		{Name: "out-path-is-a-directory",
			Refusal: "REFUSING: adir is not writable: Is a directory",
			Break:   func(string) error { return nil }},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			switch f.Name {
			case "no-args":
				s.PythonArgs = nil
			case "three-args":
				s.PythonArgs = []string{"dump.json", "out.go", "extra"}
			case "out-path-is-a-directory":
				s.PythonArgs = []string{"dump.json", "adir"}
			}
			s.GoArgs = append([]string{"core-constants-emit"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestCoreConstantsEmitKeyDeletion: THE REVERSE DIRECTION: delete each key of the input in turn and require agreement.
func TestCoreConstantsEmitKeyDeletion(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	difftest.AssertKeyDeletionAgrees(t, t.Context(), ccnSpec(bin, "dump.json", "out.go"),
		[]string{"dump.json"}, ccnSandbox(t, repo))
}
