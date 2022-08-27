import { colorCodes } from './colors';

type TooltipFunc = (mode: 'BODY' | 'HOVER', value: unknown) => string;

interface Variable {
  label: string;
  var: string;
  type?: 'list' | 'count' | 'check' | 'integer' | 'countAllowZero';
  defaultPlaceholderState?: number;
  defaultState?: boolean;
  ifCond?: string;
  ifEnemyCond?: string;
  ifEnemyMult?: string;
  ifFlag?: string;
  ifMinionCond?: string;
  ifMult?: string;
  ifOption?: string;
  ifSkill?: string;
  ifSkillFlag?: string;
  ifSkillList?: string[];
  implyCond?: string;
  implyCondList?: string[];
  tooltip?: string;
  tooltipFunc?: TooltipFunc;
}

interface VariableList extends Variable {
  type: 'list';
  list: {
    label: string;
    value: unknown;
  }[];
}

interface VariableCount extends Variable {
  type: 'count';
}

interface VariableInteger extends Variable {
  type: 'integer';
}

interface VariableCountAllowZero extends Variable {
  type: 'countAllowZero';
}

interface VariableCheck extends Variable {
  type: 'check';
}

type AllVarTypes = VariableList | VariableCount | VariableCheck | VariableInteger | VariableCountAllowZero;

export interface ConfigSection {
  name: string;
  variables: AllVarTypes[];
}

const applyPantheonDescription = (): string =>
  /*
    TODO applyPantheonDescription
  	tooltip:Clear()
    if value.val == "None" then
      return
    end
    local applyModes = { BODY = true, HOVER = true }
    if applyModes[mode] then
      local god = data.pantheons[value.val]
      for _, soul in ipairs(god.souls) do
        local name = soul.name
        local lines = { }
        for _, mod in ipairs(soul.mods) do
          table.insert(lines, mod.line)
        end
        tooltip:AddLine(20, '^8'+name)
        tooltip:AddLine(14, '^6'+table.concat(lines, '\n'))
        tooltip:AddSeparator(10)
      end
    end
   */
  '';

const banditTooltip = (): string =>
  /*
    TODO banditTooltip
    local banditBenefits = {
      ["None"] = "Grants 2 Passive Skill Points",
      ["Oak"] = "Regenerate 1% of Life per second\n2% additional Physical Damage Reduction\n20% icreased Physical Damage",
      ["Kraityn"] = "6% increased Attack and Cast Speed\n10% chance to Avoid Elemental Ailments\n6% increased Movement Speed",
      ["Alira"] = "Regenerate 5 Mana per second\n+20% to Critical Strike Multiplier\n+15% to all Elemental Resistances",
    }
    local applyModes = { BODY = true, HOVER = true }
    tooltip:Clear()
    if applyModes[mode] then
      tooltip:AddLine(14, '^8'+banditBenefits[value.val])
    end
   */
  '';

export const configurations: ConfigSection[] = [
  // Section: General options
  {
    name: 'General',
    variables: [
      {
        var: 'resistancePenalty',
        type: 'list',
        label: 'Resistance penalty:',
        list: [
          { value: 0, label: 'None' },
          { value: -30, label: 'Act 5 (-30%)' },
          { value: -60, label: 'Act 10 (-60%)' }
        ]
      },
      {
        var: 'bandit',
        type: 'list',
        label: 'Bandit quest:',
        tooltipFunc: banditTooltip,
        list: [
          { value: 'None', label: 'Kill all' },
          { value: 'Oak', label: 'Help Oak' },
          {
            value: 'Kraityn',
            label: 'Help Kraityn'
          },
          { value: 'Alira', label: 'Help Alira' }
        ]
      },
      {
        var: 'pantheonMajorGod',
        type: 'list',
        label: 'Major God:',
        tooltipFunc: applyPantheonDescription,
        list: [
          { label: 'Nothing', value: 'None' },
          { label: 'Soul of the Brine King', value: 'TheBrineKing' },
          { label: 'Soul of Lunaris', value: 'Lunaris' },
          { label: 'Soul of Solaris', value: 'Solaris' },
          { label: 'Soul of Arakaali', value: 'Arakaali' }
        ]
      },
      {
        var: 'pantheonMinorGod',
        type: 'list',
        label: 'Minor God:',
        tooltipFunc: applyPantheonDescription,
        list: [
          { label: 'Nothing', value: 'None' },
          { label: 'Soul of Gruthkul', value: 'Gruthkul' },
          { label: 'Soul of Yugul', value: 'Yugul' },
          { label: 'Soul of Abberath', value: 'Abberath' },
          { label: 'Soul of Tukohama', value: 'Tukohama' },
          { label: 'Soul of Garukhan', value: 'Garukhan' },
          { label: 'Soul of Ralakesh', value: 'Ralakesh' },
          { label: 'Soul of Ryslatha', value: 'Ryslatha' },
          { label: 'Soul of Shakari', value: 'Shakari' }
        ]
      },
      {
        var: 'detonateDeadCorpseLife',
        type: 'count',
        label: `Enemy Corpse ^${colorCodes.LIFE}Life^#:`,

        // TODO Remove hardcoded monster life table values
        tooltip: `Sets the maximum ^${colorCodes.LIFE}life ^#of the target corpse for Detonate Dead and similar skills.\nFor reference, a level 70 monster has 6937 base ^${colorCodes.LIFE}life^#, and a level 80 monster has 12787.`
      },
      {
        var: 'conditionStationary',
        type: 'count',
        label: 'Time spent stationary',
        ifCond: 'Stationary',
        tooltip: 'Applies mods that use `while stationary` and `per / every second while stationary`'
      },
      { var: 'conditionMoving', type: 'check', label: 'Are you always moving?', ifCond: 'Moving' },
      { var: 'conditionInsane', type: 'check', label: 'Are you insane?', ifCond: 'Insane' },
      {
        var: 'conditionFullLife',
        type: 'check',
        label: `Are you always on Full ^${colorCodes.LIFE}Life^#?`,
        ifCond: 'FullLife',
        tooltip: `You will automatically be considered to be on Full ^${colorCodes.LIFE}Life ^#if you have Chaos Inoculation,\nbut you can use this option to force it if necessary.`
      },
      {
        var: 'conditionLowLife',
        type: 'check',
        label: `Are you always on Low ^${colorCodes.LIFE}Life^#?`,
        ifCond: 'LowLife',
        tooltip: `You will automatically be considered to be on Low ^${colorCodes.LIFE}Life ^#if you have at least 50% ^${colorCodes.LIFE}life ^#reserved,\nbut you can use this option to force it if necessary.`
      },
      {
        var: 'conditionFullMana',
        type: 'check',
        label: `Are you always on Full ^${colorCodes.MANA}Mana^#?`,
        ifCond: 'FullMana'
      },
      {
        var: 'conditionLowMana',
        type: 'check',
        label: `Are you always on Low ^${colorCodes.MANA}Mana^#?`,
        ifCond: 'LowMana',
        tooltip: `You will automatically be considered to be on Low ^${colorCodes.MANA}Mana ^#if you have at least 50% ^${colorCodes.MANA}mana ^#reserved,\nbut you can use this option to force it if necessary.`
      },
      {
        var: 'conditionFullEnergyShield',
        type: 'check',
        label: `Are you always on Full ^${colorCodes.ES}Energy Shield^#?`,
        ifCond: 'FullEnergyShield'
      },
      {
        var: 'conditionLowEnergyShield',
        type: 'check',
        label: `Are you always on Low ^${colorCodes.ES}Energy Shield^#?`,
        ifCond: 'LowEnergyShield',
        tooltip: `You will automatically be considered to be on Low ^${colorCodes.ES}Energy Shield ^#if you have at least 50% ^${colorCodes.ES}ES ^#reserved,\nbut you can use this option to force it if necessary.`
      },
      {
        var: 'conditionHaveEnergyShield',
        type: 'check',
        label: `Do you always have ^${colorCodes.ES}Energy Shield^#?`,
        ifCond: 'HaveEnergyShield'
      },
      {
        var: 'minionsConditionFullLife',
        type: 'check',
        label: `Are your Minions always on Full ^${colorCodes.LIFE}Life^#?`,
        ifMinionCond: 'FullLife'
      },
      {
        var: 'minionsConditionCreatedRecently',
        type: 'check',
        label: 'Have your Minions been created Recently?',
        ifCond: 'MinionsCreatedRecently'
      },
      {
        var: 'igniteMode',
        type: 'list',
        label: 'Ailment calculation mode:',
        tooltip:
          'Controls how the base damage for applying Ailments is calculated:\n\tAverage: damage is based on the average application, including both crits and non-crits\n\tCrits Only: damage is based solely on Ailments inflicted with crits',
        list: [
          { value: 'AVERAGE', label: 'Average' },
          { value: 'CRIT', label: 'Crits Only' }
        ]
      },
      {
        var: 'physMode',
        type: 'list',
        label: 'Random element mode:',
        ifFlag: 'randomPhys',
        tooltip: `Controls how modifiers which choose a random element will function.\n\tAverage: Modifiers will grant one third of their value to ^${colorCodes.FIRE}Fire^#, ^${colorCodes.COLD}Cold^#, and ^${colorCodes.LIGHTNING}Lightning ^#simultaneously\n\t^${colorCodes.FIRE}Fire ^#/ ^${colorCodes.COLD}Cold ^#/ ^${colorCodes.LIGHTNING}Lightning^#: Modifiers will grant their full value as the specified element\nIf a modifier chooses between just two elements, the full value can only be given as those two elements.`,
        list: [
          { value: 'AVERAGE', label: 'Average' },
          { value: 'Fire', label: `^${colorCodes.FIRE}Fire^#` },
          {
            value: 'Cold',
            label: `^${colorCodes.COLD}Cold^#`
          },
          { value: 'Lightning', label: `^${colorCodes.LIGHTNING}Lightning^#` }
        ]
      },
      {
        var: 'lifeRegenMode',
        type: 'list',
        label: `^${colorCodes.LIFE}Life ^#regen calculation mode:`,
        tooltip: `Controls how ^${colorCodes.LIFE}life ^#regeneration is calculated:\n\tMinimum: does not include burst regen\n\tAverage: includes burst regen, averaged based on uptime\n\tBurst: includes full burst regen`,
        list: [
          { value: 'MIN', label: 'Minimum' },
          { value: 'AVERAGE', label: 'Average' },
          { value: 'FULL', label: 'Burst' }
        ]
      },
      {
        var: 'EHPUnluckyWorstOf',
        type: 'list',
        label: 'EHP calc unlucky:',
        tooltip: 'Sets the EHP calc to pretend its unlucky and reduce the effects of random events',
        list: [
          { value: 1, label: 'Average' },
          { value: 2, label: 'Unlucky' },
          { value: 4, label: 'Very Unlucky' }
        ]
      },
      {
        var: 'DisableEHPGainOnBlock',
        type: 'check',
        label: 'Disable EHP gain on block:',
        tooltip: 'Sets the EHP calc to not apply gain on block effects'
      },
      {
        var: 'armourCalculationMode',
        type: 'list',
        label: 'Armour calculation mode:',
        tooltip:
          'Controls how Defending with Double Armour is calculated:\n\tMinimum: never Defend with Double Armour\n\tAverage: Damage Reduction from Defending with Double Armour is proportional to chance\n\tMaximum: always Defend with Double Armour\nThis setting has no effect if you have 100% chance to Defend with Double Armour.',
        list: [
          { value: 'MIN', label: 'Minimum' },
          { value: 'AVERAGE', label: 'Average' },
          { value: 'MAX', label: 'Maximum' }
        ]
      },
      {
        var: 'warcryMode',
        type: 'list',
        label: 'Warcry calculation mode:',
        ifSkillList: ['Infernal Cry', 'Ancestral Cry', 'Enduring Cry', "General's Cry", 'Intimidating Cry', 'Rallying Cry', 'Seismic Cry', "Battlemage's Cry"],
        tooltip:
          'Controls how exerted attacks from Warcries are calculated:\nAverage: Averages out Warcry usage with cast time, attack speed and warcry cooldown.\nMax Hit: Shows maximum hit for lining up all warcries.',
        list: [
          { value: 'AVERAGE', label: 'Average' },
          { value: 'MAX', label: 'Max Hit' }
        ]
      },
      { var: 'EVBypass', type: 'check', label: "Disable Emperor's Vigilance Bypass", ifCond: 'EVBypass' }
    ]
  },
  {
    name: 'Arcanist Brand',
    variables: [
      {
        var: 'targetBrandedEnemy',
        type: 'check',
        label: 'Are skills targeting the Branded enemy?',
        ifSkill: 'Arcanist Brand'
      }
    ]
  },
  {
    name: 'Aspect of the Avian',
    variables: [
      {
        var: 'aspectOfTheAvianAviansMight',
        type: 'check',
        label: "Is Avian's Might active?",
        ifSkill: 'Aspect of the Avian'
      },
      {
        var: 'aspectOfTheAvianAviansFlight',
        type: 'check',
        label: "Is Avian's Flight active?",
        ifSkill: 'Aspect of the Avian'
      }
    ]
  },
  {
    name: 'Aspect of the Cat',
    variables: [
      {
        var: 'aspectOfTheCatCatsStealth',
        type: 'check',
        label: "Is Cat's Stealth active?",
        ifSkill: 'Aspect of the Cat'
      },
      {
        var: 'aspectOfTheCatCatsAgility',
        type: 'check',
        label: "Is Cat's Agility active?",
        ifSkill: 'Aspect of the Cat'
      }
    ]
  },
  {
    name: 'Aspect of the Crab',
    variables: [
      {
        var: 'overrideCrabBarriers',
        type: 'count',
        label: '# of Crab Barriers (if not maximum):',
        ifSkill: 'Aspect of the Crab'
      }
    ]
  },
  {
    name: 'Aspect of the Spider',
    variables: [
      {
        var: 'aspectOfTheSpiderWebStacks',
        type: 'count',
        label: "# of Spider's Web Stacks:",
        ifSkill: 'Aspect of the Spider'
      }
    ]
  },
  {
    name: 'Banner Skills',
    variables: [
      {
        var: 'bannerPlanted',
        type: 'check',
        label: 'Is Banner Planted?',
        ifSkillList: ['Dread Banner', 'War Banner', 'Defiance Banner']
      },
      {
        var: 'bannerStages',
        type: 'count',
        label: 'Banner Stages:',
        ifSkillList: ['Dread Banner', 'War Banner', 'Defiance Banner']
      }
    ]
  },
  {
    name: 'Bladestorm',
    variables: [
      { var: 'bladestormInBloodstorm', type: 'check', label: 'Are you in a Bloodstorm?', ifSkill: 'Bladestorm' },
      { var: 'bladestormInSandstorm', type: 'check', label: 'Are you in a Sandstorm?', ifSkill: 'Bladestorm' }
    ]
  },
  {
    name: 'Bonechill Support',
    variables: [
      {
        var: 'bonechillEffect',
        type: 'count',
        label: 'Effect of Chill:',
        tooltip: `The effect of ^${colorCodes.COLD}Chill ^#is automatically calculated if you have a guaranteed source of ^${colorCodes.COLD}Chill^#,\nbut you can use this to override the effect if necessary.`,
        ifSkill: 'Bonechill'
      }
    ]
  },
  {
    name: 'Boneshatter',
    variables: [
      {
        var: 'boneshatterTraumaStacks',
        type: 'count',
        label: '# of Trauma Stacks:',
        ifSkill: 'Boneshatter'
      }
    ]
  },
  {
    name: 'Brand Skills',
    variables: [
      {
        var: 'ActiveBrands',
        type: 'count',
        label: '# of active Brands:',
        ifSkillList: ['Armageddon Brand', 'Storm Brand', 'Arcanist Brand', 'Penance Brand', 'Wintertide Brand']
      },
      {
        var: 'BrandsAttachedToEnemy',
        type: 'count',
        label: '# of Brands attached to the enemy:',
        ifEnemyMult: 'BrandsAttached'
      },
      {
        var: 'BrandsInLastQuarter',
        type: 'check',
        label: 'Last 25% of Attached Duration?',
        ifCond: 'BrandLastQuarter'
      }
    ]
  },
  {
    name: 'Carrion Golem',
    variables: [
      {
        var: 'carrionGolemNearbyMinion',
        type: 'count',
        label: '# of Nearby Non-Golem Minions:',
        ifSkill: 'Summon Carrion Golem'
      }
    ]
  },
  {
    name: 'Close Combat',
    variables: [
      {
        var: 'closeCombatCombatRush',
        type: 'check',
        label: 'Is Combat Rush active?',
        ifSkill: 'Close Combat',
        tooltip: 'Combat Rush grants 20% more Attack Speed to Travel Skills not Supported by Close Combat.'
      }
    ]
  },
  {
    name: 'Cruelty',
    variables: [
      {
        var: 'overrideCruelty',
        type: 'count',
        label: 'Damage % (if not maximum):',
        ifSkill: 'Cruelty',
        tooltip: 'Cruelty is a buff provided by Cruelty Support which grants\nup to 40% more damage over time to the skills it supports.'
      }
    ]
  },
  {
    name: 'Cyclone',
    variables: [
      {
        var: 'channellingCycloneCheck',
        type: 'check',
        label: 'Are you Channelling Cyclone?',
        ifSkill: 'Cyclone'
      }
    ]
  },
  {
    name: 'Dark Pact',
    variables: [
      {
        var: 'darkPactSkeletonLife',
        type: 'count',
        label: `Skeleton ^${colorCodes.LIFE}Life^#:`,
        ifSkill: 'Dark Pact',
        tooltip: `Sets the maximum ^${colorCodes.LIFE}Life ^#of the Skeleton that is being targeted.`
      }
    ]
  },
  {
    name: 'Predator',
    variables: [
      {
        var: 'deathmarkDeathmarkActive',
        type: 'check',
        label: 'Is the enemy marked with Signal Prey?',
        ifSkill: 'Predator'
      }
    ]
  },
  {
    name: 'Elemental Army',
    variables: [
      {
        var: 'elementalArmyExposureType',
        type: 'list',
        label: 'Exposure Type:',
        ifSkill: 'Elemental Army',
        list: [
          { value: 0, label: 'None' },
          { value: 'Fire', label: `^${colorCodes.FIRE}Fire^#` },
          {
            value: 'Cold',
            label: `^${colorCodes.COLD}Cold^#`
          },
          { value: 'Lightning', label: `^${colorCodes.LIGHTNING}Lightning^#` }
        ]
      }
    ]
  },
  {
    name: 'Energy Blade',
    variables: [
      {
        var: 'energyBladeActive',
        type: 'check',
        label: 'Is Energy Blade active?',
        ifSkill: 'Energy Blade',
        tooltip: 'Energy Blade transforms your weapons into Swords formed from energy'
      }
    ]
  },
  {
    name: 'Embrace Madness',
    variables: [
      {
        var: 'embraceMadnessActive',
        type: 'check',
        label: 'Is Embrace Madness active?',
        ifSkill: 'Embrace Madness'
      }
    ]
  },
  {
    name: 'Feeding Frenzy',
    variables: [
      {
        var: 'feedingFrenzyFeedingFrenzyActive',
        type: 'check',
        label: 'Is Feeding Frenzy active?',
        ifSkill: 'Feeding Frenzy',
        tooltip: 'Feeding Frenzy grants:\n\t10% more Minion Damage\n\t10% increased Minion Movement Speed\n\t10% increased Minion Attack and Cast Speed'
      }
    ]
  },
  {
    name: 'Flame Wall',
    variables: [
      {
        var: 'flameWallAddedDamage',
        type: 'check',
        label: 'Projectile Travelled through Flame Wall?',
        ifSkill: 'Flame Wall'
      }
    ]
  },
  {
    name: 'Frostbolt',
    variables: [{ var: 'frostboltExposure', type: 'check', label: 'Can you apply Exposure?', ifSkill: 'Frostbolt' }]
  },
  {
    name: 'Frost Shield',
    variables: [{ var: 'frostShieldStages', type: 'count', label: 'Stages:', ifSkill: 'Frost Shield' }]
  },
  {
    name: 'Greater Harbinger of Time',
    variables: [
      {
        var: 'greaterHarbingerOfTimeSlipstream',
        type: 'check',
        label: 'Is Slipstream active?:',
        ifSkill: 'Summon Greater Harbinger of Time',
        tooltip:
          'Greater Harbinger of Time Slipstream buff grants:\n10% increased Action Speed\nBuff affects the player and allies\nBuff has a base duration of 8s with a 10s Cooldown'
      }
    ]
  },
  {
    name: 'Harbinger of Time',
    variables: [
      {
        var: 'harbingerOfTimeSlipstream',
        type: 'check',
        label: 'Is Slipstream active?:',
        ifSkill: 'Summon Harbinger of Time',
        tooltip:
          'Harbinger of Time Slipstream buff grants:\n10% increased Action Speed\nBuff affects the player, allies and enemies in a small radius\nBuff has a base duration of 8s with a 20s Cooldown'
      }
    ]
  },
  {
    name: 'Hex',
    variables: [{ var: 'multiplierHexDoom', type: 'count', label: 'Doom on Hex:', ifSkillFlag: 'hex' }]
  },
  {
    name: 'Herald of Agony',
    variables: [
      {
        var: 'heraldOfAgonyVirulenceStack',
        type: 'count',
        label: '# of Virulence Stacks:',
        ifSkill: 'Herald of Agony'
      }
    ]
  },
  {
    name: 'Ice Nova',
    variables: [{ var: 'iceNovaCastOnFrostbolt', type: 'check', label: 'Cast on Frostbolt?', ifSkill: 'Ice Nova' }]
  },
  {
    name: 'Infusion',
    variables: [
      {
        var: 'infusedChannellingInfusion',
        type: 'check',
        label: 'Is Infusion active?',
        ifSkill: 'Infused Channelling'
      }
    ]
  },
  {
    name: 'Innervate',
    variables: [{ var: 'innervateInnervation', type: 'check', label: 'Is Innervation active?', ifSkill: 'Innervate' }]
  },
  {
    name: 'Intensify',
    variables: [
      {
        var: 'intensifyIntensity',
        type: 'count',
        label: '# of Intensity:',
        ifSkillList: ['Intensify', 'Crackling Lance', 'Pinpoint']
      }
    ]
  },
  {
    name: 'Meat Shield',
    variables: [
      {
        var: 'meatShieldEnemyNearYou',
        type: 'check',
        label: 'Is the enemy near you?',
        ifSkill: 'Meat Shield'
      }
    ]
  },
  {
    name: 'Plague Bearer',
    variables: [
      {
        var: 'plagueBearerState',
        type: 'list',
        label: 'State:',
        ifSkill: 'Plague Bearer',
        list: [
          { value: 'INC', label: 'Incubating' },
          { value: 'INF', label: 'Infecting' }
        ]
      }
    ]
  },
  {
    name: 'Perforate',
    variables: [
      {
        var: 'perforateSpikeOverlap',
        type: 'count',
        label: '# of Overlapping Spikes:',
        tooltip: 'Affects the DPS of Perforate in Blood Stance.\nMaximum is limited by the number of Spikes of Perforate.',
        ifSkill: 'Perforate'
      }
    ]
  },
  {
    name: 'Physical Aegis',
    variables: [
      {
        var: 'physicalAegisDepleted',
        type: 'check',
        label: 'Is Physical Aegis depleted?',
        ifSkill: 'Physical Aegis'
      }
    ]
  },
  {
    name: 'Pride',
    variables: [
      {
        var: 'prideEffect',
        type: 'list',
        label: 'Pride Aura Effect:',
        ifSkill: 'Pride',
        list: [
          { value: 'MIN', label: 'Initial effect' },
          { value: 'MAX', label: 'Maximum effect' }
        ]
      }
    ]
  },
  {
    name: 'Rage Vortex',
    variables: [
      {
        var: 'sacrificedRageCount',
        type: 'count',
        label: 'Amount of Rage Sacrificed?',
        ifSkill: 'Rage Vortex'
      }
    ]
  },
  {
    name: 'Raise Spectre',
    variables: [
      {
        var: 'raiseSpectreEnableBuffs',
        type: 'check',
        defaultState: true,
        label: 'Enable buffs:',
        ifSkill: 'Raise Spectre',
        tooltip: 'Enable any buff skills that your spectres have.'
      },
      {
        var: 'raiseSpectreEnableCurses',
        type: 'check',
        defaultState: true,
        label: 'Enable curses:',
        ifSkill: 'Raise Spectre',
        tooltip: 'Enable any curse skills that your spectres have.'
      },
      {
        var: 'raiseSpectreBladeVortexBladeCount',
        type: 'count',
        label: 'Blade Vortex blade count:',
        ifSkillList: ['DemonModularBladeVortexSpectre', 'GhostPirateBladeVortexSpectre'],
        tooltip: 'Sets the blade count for Blade Vortex skills used by spectres.\nDefault is 1; maximum is 5.'
      },
      {
        var: 'raiseSpectreKaomFireBeamTotemStage',
        type: 'count',
        label: 'Scorching Ray Totem stage count:',
        ifSkill: 'KaomFireBeamTotemSpectre'
      },
      {
        var: 'raiseSpectreEnableSummonedUrsaRallyingCry',
        type: 'check',
        label: "Enable Summoned Ursa's Rallying Cry:",
        ifSkill: 'DropBearSummonedRallyingCry'
      }
    ]
  },
  {
    name: 'Raise Spiders',
    variables: [
      { var: 'raiseSpidersSpiderCount', type: 'count', label: '# of Spiders:', ifSkill: 'Raise Spiders' },
      {
        var: 'animateWeaponLingeringBlade',
        type: 'check',
        label: 'Are you animating Lingering Blades?',
        ifSkill: 'Animate Weapon',
        tooltip: 'Enables additional damage given to Lingering Blades\nThe exact weapon is unknown but should be similar to Glass Shank'
      }
    ]
  },
  {
    name: 'Sigil of Power',
    variables: [{ var: 'sigilOfPowerStages', type: 'count', label: 'Stages:', ifSkill: 'Sigil of Power' }]
  },
  {
    name: 'Siphoning Trap',
    variables: [
      {
        var: 'siphoningTrapAffectedEnemies',
        type: 'count',
        label: '# of Enemies affected:',
        ifSkill: 'Siphoning Trap',
        tooltip: 'Sets the number of enemies affected by Siphoning Trap.'
      }
    ]
  },
  {
    name: 'Snipe',
    variables: [
      {
        var: 'configSnipeStages',
        type: 'count',
        label: '# of Snipe stages:',
        ifSkill: 'Snipe',
        tooltip: 'Sets the number of stages reached before releasing Snipe.'
      }
    ]
  },
  {
    name: 'Trinity Support',
    variables: [
      {
        var: 'configResonanceCount',
        type: 'count',
        label: 'Lowest Resonance Count:',
        ifSkill: 'Trinity',
        tooltip: 'Sets the amount of resonance on the lowest element.'
      }
    ]
  },
  {
    name: 'Spectral Wolf',
    variables: [
      {
        var: 'configSpectralWolfCount',
        type: 'count',
        label: '# of Active Spectral Wolves:',
        ifSkill: 'Summon Spectral Wolf',
        tooltip: 'Sets the number of active Spectral Wolves.\nThe maximum number of Spectral Wolves is 10.'
      }
    ]
  },
  {
    name: 'Stance Skills',
    variables: [
      {
        var: 'bloodSandStance',
        type: 'list',
        label: 'Stance:',
        ifSkillList: ['Blood and Sand', 'Flesh and Stone', 'Lacerate', 'Bladestorm', 'Perforate'],
        list: [
          { value: 'BLOOD', label: 'Blood Stance' },
          { value: 'SAND', label: 'Sand Stance' }
        ]
      },
      { var: 'changedStance', type: 'check', label: 'Changed Stance recently?', ifCond: 'ChangedStanceRecently' }
    ]
  },
  {
    name: 'Steel Skills',
    variables: [
      {
        var: 'shardsConsumed',
        type: 'count',
        label: 'Steel Shards consumed:',
        ifSkillList: ['Splitting Steel', 'Shattering Steel', 'Lancing Steel']
      },
      {
        var: 'steelWards',
        type: 'count',
        label: 'Steel Wards:',
        ifSkill: 'Shattering Steel',
        tooltip:
          'Steel Wards are gained from using Shattering Steel with at least 2 Steel Shards.\nYou can have up to 6 Steel Wards, and each grants +4% chance to Block Projectile Attack Damage.'
      }
    ]
  },
  {
    name: 'Storm Rain',
    variables: [
      {
        var: 'stormRainBeamOverlap',
        type: 'count',
        label: '# of Overlapping Beams:',
        ifSkill: 'Storm Rain'
      }
    ]
  },
  {
    name: 'Summon Holy Relic',
    variables: [
      {
        var: 'summonHolyRelicEnableHolyRelicBoon',
        type: 'check',
        label: "Enable Holy Relic's Boon Aura:",
        ifSkill: 'Summon Holy Relic'
      }
    ]
  },
  {
    name: 'Summon Lightning Golem',
    variables: [
      {
        var: 'summonLightningGolemEnableWrath',
        type: 'check',
        label: 'Enable Wrath Aura:',
        ifSkill: 'Summon Lightning Golem'
      }
    ]
  },
  {
    name: 'Thirst for Blood',
    variables: [
      {
        var: 'nearbyBleedingEnemies',
        type: 'count',
        label: '# of Nearby Bleeding Enemies:',
        ifSkill: 'Thirst for Blood'
      }
    ]
  },
  {
    name: 'Toxic Rain',
    variables: [
      {
        var: 'toxicRainPodOverlap',
        type: 'count',
        label: '# of Overlapping Pods:',
        tooltip: 'Maximum is limited by the number of Projectiles.',
        ifSkill: 'Toxic Rain'
      }
    ]
  },
  {
    name: 'Herald of Ash',
    variables: [
      {
        var: 'hoaOverkill',
        type: 'count',
        label: 'Overkill damage:',
        tooltip: `Herald of Ash's base ^${colorCodes.FIRE}Burning ^#damage is equal to 25% of Overkill damage.`,
        ifSkill: 'Herald of Ash'
      }
    ]
  },
  {
    name: 'Voltaxic Burst',
    variables: [
      {
        var: 'voltaxicBurstSpellsQueued',
        type: 'count',
        label: '# of Casts currently waiting:',
        ifSkill: 'Voltaxic Burst'
      }
    ]
  },
  {
    name: 'Vortex',
    variables: [{ var: 'vortexCastOnFrostbolt', type: 'check', label: 'Cast on Frostbolt?', ifSkill: 'Vortex' }]
  },
  {
    name: 'Cold Snap',
    variables: [{ var: 'ColdSnapBypassCD', type: 'check', label: 'Bypass CD?', ifSkill: 'Cold Snap' }]
  },
  {
    name: 'Warcry Skills',
    variables: [
      {
        var: 'multiplierWarcryPower',
        type: 'count',
        label: 'Warcry Power:',
        ifSkillList: ['Infernal Cry', 'Ancestral Cry', 'Enduring Cry', "General's Cry", 'Intimidating Cry', 'Rallying Cry', 'Seismic Cry', "Battlemage's Cry"],
        tooltip:
          'Power determines how strong your Warcry buffs will be, and is based on the total strength of nearby enemies.\nPower is assumed to be 20 if your target is a Boss, but you can override it here if necessary.\n\tEach Normal enemy grants 1 Power\n\tEach Magic enemy grants 2 Power\n\tEach Rare enemy grants 10 Power\n\tEach Unique enemy grants 20 Power'
      }
    ]
  },
  {
    name: 'Wave of Conviction',
    variables: [
      {
        var: 'waveOfConvictionExposureType',
        type: 'list',
        label: 'Exposure Type:',
        ifSkill: 'Wave of Conviction',
        list: [
          { value: 0, label: 'None' },
          { value: 'Fire', label: `^${colorCodes.FIRE}Fire^#` },
          {
            value: 'Cold',
            label: `^${colorCodes.COLD}Cold^#`
          },
          { value: 'Lightning', label: `^${colorCodes.LIGHTNING}Lightning^#` }
        ]
      }
    ]
  },
  {
    name: 'Molten Shell',
    variables: [
      {
        var: 'MoltenShellDamageMitigated',
        type: 'count',
        label: 'Damage mitigated:',
        tooltip: 'Molten Shell reflects damage to the enemy,\nbased on the amount of damage it has mitigated.',
        ifSkill: 'Molten Shell'
      }
    ]
  },
  {
    name: 'Vaal Molten Shell',
    variables: [
      {
        var: 'VaalMoltenShellDamageMitigated',
        type: 'count',
        label: 'Damage mitigated:',
        tooltip: 'Vaal Molten Shell reflects damage to the enemy,\nbased on the amount of damage it has mitigated in the last second.',
        ifSkill: 'Vaal Molten Shell'
      }
    ]
  },
  {
    name: 'Map Prefix Modifiers',
    variables: [
      {
        var: 'enemyHasPhysicalReduction',
        type: 'list',
        label: 'Enemy Physical Damage reduction:',
        tooltip: "'Armoured'",
        list: [
          { value: 0, label: 'None' },
          { value: 20, label: '20% (Low tier)' },
          {
            value: 30,
            label: '30% (Mid tier)'
          },
          { value: 40, label: '40% (High tier)' }
        ]
      },
      { var: 'enemyIsHexproof', type: 'check', label: 'Enemy is Hexproof?', tooltip: "'Hexproof'" },
      {
        var: 'enemyHasLessCurseEffectOnSelf',
        type: 'list',
        label: 'Less effect of Curses on enemy:',
        tooltip: "'Hexwarded'",
        list: [
          { value: 0, label: 'None' },
          { value: 25, label: '25% (Low tier)' },
          {
            value: 40,
            label: '40% (Mid tier)'
          },
          { value: 60, label: '60% (High tier)' }
        ]
      },
      {
        var: 'enemyCanAvoidPoisonBlindBleed',
        type: 'list',
        label: 'Enemy avoid Poison / Blind / Bleed:',
        tooltip: "'Impervious'",
        list: [
          { value: 0, label: 'None' },
          { value: 25, label: '25% (Low tier)' },
          {
            value: 45,
            label: '45% (Mid tier)'
          },
          { value: 65, label: '65% (High tier)' }
        ]
      },
      {
        var: 'enemyHasResistances',
        type: 'list',
        label: `Enemy has Elemental / ^${colorCodes.CHAOS}Chaos ^#Resist:`,
        tooltip: "'Resistant'",
        list: [
          { value: 0, label: 'None' },
          { value: 'LOW', label: '20% / 15% (Low tier)' },
          {
            value: 'MID',
            label: '30% /2 0% (Mid tier)'
          },
          { value: 'HIGH', label: '40% / 25% (High tier)' }
        ]
      }
    ]
  },
  {
    name: 'Map Suffix Modifiers',
    variables: [
      {
        var: 'playerHasElementalEquilibrium',
        type: 'check',
        label: 'Player has Elemental Equilibrium?',
        tooltip: "'of Balance'"
      },
      {
        var: 'playerCannotLeech',
        type: 'check',
        label: `Cannot Leech ^${colorCodes.LIFE}Life ^#/ ^${colorCodes.MANA}Mana^#?`,
        tooltip: "'of Congealment'"
      },
      {
        var: 'playerGainsReducedFlaskCharges',
        type: 'list',
        label: 'Gains reduced Flask Charges:',
        tooltip: "'of Drought'",
        list: [
          { value: 0, label: 'None' },
          { value: 30, label: '30% (Low tier)' },
          {
            value: 40,
            label: '40% (Mid tier)'
          },
          { value: 50, label: '50% (High tier)' }
        ]
      },
      {
        var: 'playerHasMinusMaxResist',
        type: 'count',
        label: '-X% maximum Resistances:',
        tooltip: "'of Exposure'\nMid tier: 5-8%\nHigh tier: 9-12%"
      },
      {
        var: 'playerHasLessAreaOfEffect',
        type: 'list',
        label: 'Less Area of Effect:',
        tooltip: "'of Impotence'",
        list: [
          { value: 0, label: 'None' },
          { value: 15, label: '15% (Low tier)' },
          {
            value: 20,
            label: '20% (Mid tier)'
          },
          { value: 25, label: '25% (High tier)' }
        ]
      },
      {
        var: 'enemyCanAvoidStatusAilment',
        type: 'list',
        label: 'Enemy avoid Elem. Status Ailments:',
        tooltip: "'of Insulation'",
        list: [
          { value: 0, label: 'None' },
          { value: 30, label: '30% (Low tier)' },
          {
            value: 60,
            label: '60% (Mid tier)'
          },
          { value: 90, label: '90% (High tier)' }
        ]
      },
      {
        var: 'enemyHasIncreasedAccuracy',
        type: 'list',
        label: 'Unlucky Dodge / Enemy has inc. Accuracy:',
        tooltip: "'of Miring'",
        list: [
          { value: 0, label: 'None' },
          { value: 30, label: '30% (Low tier)' },
          {
            value: 40,
            label: '40% (Mid tier)'
          },
          { value: 50, label: '50% (High tier)' }
        ]
      },
      {
        var: 'playerHasLessArmourAndBlock',
        type: 'list',
        label: 'Reduced Block Chance / less Armour:',
        tooltip: "'of Rust'",
        list: [
          { value: 0, label: 'None' },
          { value: 'LOW', label: '20% / 20% (Low tier)' },
          {
            value: 'MID',
            label: '30% / 25% (Mid tier)'
          },
          { value: 'HIGH', label: '40% / 30% (High tier)' }
        ]
      },
      { var: 'playerHasPointBlank', type: 'check', label: 'Player has Point Blank?', tooltip: "'of Skirmishing'" },
      {
        var: 'playerHasLessLifeESRecovery',
        type: 'list',
        label: `Less Recovery Rate of ^${colorCodes.LIFE}Life ^#and ^${colorCodes.ES}Energy Shield^#:`,
        tooltip: "'of Smothering'",
        list: [
          { value: 0, label: 'None' },
          { value: 20, label: '20% (Low tier)' },
          {
            value: 40,
            label: '40% (Mid tier)'
          },
          { value: 60, label: '60% (High tier)' }
        ]
      },
      {
        var: 'playerCannotRegenLifeManaEnergyShield',
        type: 'check',
        label: `Cannot Regen ^${colorCodes.LIFE}Life^#, ^${colorCodes.MANA}Mana ^#or ^${colorCodes.ES}ES^#?`,
        tooltip: "'of Stasis'"
      },
      {
        var: 'enemyTakesReducedExtraCritDamage',
        type: 'count',
        label: 'Enemy takes red. Extra Crit Damage:',
        tooltip: "'of Toughness'\nLow tier: 25-30%\nMid tier: 31-35%\nHigh tier: 36-40%"
      },
      { var: 'multiplierSextant', type: 'count', label: '# of Sextants affecting the area', ifMult: 'Sextant' }
    ]
  },
  {
    name: 'Player is cursed by',
    variables: [
      {
        var: 'playerCursedWithAssassinsMark',
        type: 'count',
        label: "Assassin's Mark:",
        tooltip: "Sets the level of Assassin's Mark to apply to the player."
      },
      {
        var: 'playerCursedWithConductivity',
        type: 'count',
        label: 'Conductivity:',
        tooltip: 'Sets the level of Conductivity to apply to the player.'
      },
      {
        var: 'playerCursedWithDespair',
        type: 'count',
        label: 'Despair:',
        tooltip: 'Sets the level of Despair to apply to the player.'
      },
      {
        var: 'playerCursedWithElementalWeakness',
        type: 'count',
        label: 'Elemental Weakness:',
        tooltip:
          "Sets the level of Elemental Weakness to apply to the player.\nIn mid tier maps, 'of Elemental Weakness' applies level 10.\nIn high tier maps, 'of Elemental Weakness' applies level 15."
      },
      {
        var: 'playerCursedWithEnfeeble',
        type: 'count',
        label: 'Enfeeble:',
        tooltip:
          "Sets the level of Enfeeble to apply to the player.\nIn mid tier maps, 'of Enfeeblement' applies level 10.\nIn high tier maps, 'of Enfeeblement' applies level 15."
      },
      {
        var: 'playerCursedWithFlammability',
        type: 'count',
        label: 'Flammability:',
        tooltip: 'Sets the level of Flammability to apply to the player.'
      },
      {
        var: 'playerCursedWithFrostbite',
        type: 'count',
        label: 'Frostbite:',
        tooltip: 'Sets the level of Frostbite to apply to the player.'
      },
      {
        var: 'playerCursedWithPoachersMark',
        type: 'count',
        label: "Poacher's Mark:",
        tooltip: "Sets the level of Poacher's Mark to apply to the player."
      },
      {
        var: 'playerCursedWithProjectileWeakness',
        type: 'count',
        label: 'Projectile Weakness:',
        tooltip: 'Sets the level of Projectile Weakness to apply to the player.'
      },
      {
        var: 'playerCursedWithPunishment',
        type: 'count',
        label: 'Punishment:',
        tooltip: 'Sets the level of Punishment to apply to the player.'
      },
      {
        var: 'playerCursedWithTemporalChains',
        type: 'count',
        label: 'Temporal Chains:',
        tooltip:
          "Sets the level of Temporal Chains to apply to the player.\nIn mid tier maps, 'of Temporal Chains' applies level 10.\nIn high tier maps, 'of Temporal Chains' applies level 15."
      },
      {
        var: 'playerCursedWithVulnerability',
        type: 'count',
        label: 'Vulnerability:',
        tooltip:
          "Sets the level of Vulnerability to apply to the player.\nIn mid tier maps, 'of Vulnerability' applies level 10.\nIn high tier maps, 'of Vulnerability' applies level 15."
      },
      {
        var: 'playerCursedWithWarlordsMark',
        type: 'count',
        label: "Warlord's Mark:",
        tooltip: "Sets the level of Warlord's Mark to apply to the player."
      }
    ]
  },
  {
    name: 'When In Combat',
    variables: [
      { var: 'usePowerCharges', type: 'check', label: 'Do you use Power Charges?' },
      {
        var: 'overridePowerCharges',
        type: 'count',
        label: '# of Power Charges (if not maximum):',
        ifOption: 'usePowerCharges'
      },
      { var: 'useFrenzyCharges', type: 'check', label: 'Do you use Frenzy Charges?' },
      {
        var: 'overrideFrenzyCharges',
        type: 'count',
        label: '# of Frenzy Charges (if not maximum):',
        ifOption: 'useFrenzyCharges'
      },
      { var: 'useEnduranceCharges', type: 'check', label: 'Do you use Endurance Charges?' },
      {
        var: 'overrideEnduranceCharges',
        type: 'count',
        label: '# of Endurance Charges (if not maximum):',
        ifOption: 'useEnduranceCharges'
      },
      { var: 'useSiphoningCharges', type: 'check', label: 'Do you use Siphoning Charges?', ifMult: 'SiphoningCharge' },
      {
        var: 'overrideSiphoningCharges',
        type: 'count',
        label: '# of Siphoning Charges (if not maximum):',
        ifOption: 'useSiphoningCharges'
      },
      {
        var: 'useChallengerCharges',
        type: 'check',
        label: 'Do you use Challenger Charges?',
        ifMult: 'ChallengerCharge'
      },
      {
        var: 'overrideChallengerCharges',
        type: 'count',
        label: '# of Challenger Charges (if not maximum):',
        ifOption: 'useChallengerCharges'
      },
      { var: 'useBlitzCharges', type: 'check', label: 'Do you use Blitz Charges?', ifMult: 'BlitzCharge' },
      {
        var: 'overrideBlitzCharges',
        type: 'count',
        label: '# of Blitz Charges (if not maximum):',
        ifOption: 'useBlitzCharges'
      },
      {
        var: 'multiplierGaleForce',
        type: 'count',
        label: '# of Gale Force:',
        ifFlag: 'Condition:CanGainGaleForce',
        tooltip: 'Base maximum Gale Force is 10.'
      },
      {
        var: 'overrideInspirationCharges',
        type: 'count',
        label: '# of Inspiration Charges (if not maximum):',
        ifMult: 'InspirationCharge'
      },
      { var: 'useGhostShrouds', type: 'check', label: 'Do you use Ghost Shrouds?', ifMult: 'GhostShroud' },
      {
        var: 'overrideGhostShrouds',
        type: 'count',
        label: '# of Ghost Shrouds (if not maximum):',
        ifOption: 'useGhostShrouds'
      },
      { var: 'waitForMaxSeals', type: 'check', label: 'Do you wait for Max Unleash Seals?', ifFlag: 'HasSeals' },
      {
        var: 'overrideBloodCharges',
        type: 'count',
        label: '# of Blood Charges (if not maximum):',
        ifMult: 'BloodCharge'
      },
      {
        var: 'minionsUsePowerCharges',
        type: 'check',
        label: 'Do your Minions use Power Charges?',
        ifFlag: 'haveMinion'
      },
      {
        var: 'minionsUseFrenzyCharges',
        type: 'check',
        label: 'Do your Minions use Frenzy Charges?',
        ifFlag: 'haveMinion'
      },
      {
        var: 'minionsUseEnduranceCharges',
        type: 'check',
        label: 'Do your Minions use Endur. Charges?',
        ifFlag: 'haveMinion'
      },
      {
        var: 'minionsOverridePowerCharges',
        type: 'count',
        label: '# of Power Charges (if not maximum):',
        ifFlag: 'haveMinion',
        ifOption: 'minionsUsePowerCharges'
      },
      {
        var: 'minionsOverrideFrenzyCharges',
        type: 'count',
        label: '# of Frenzy Charges (if not maximum):',
        ifFlag: 'haveMinion',
        ifOption: 'minionsUseFrenzyCharges'
      },
      {
        var: 'minionsOverrideEnduranceCharges',
        type: 'count',
        label: '# of Endurance Charges (if not maximum):',
        ifFlag: 'haveMinion',
        ifOption: 'minionsUseEnduranceCharges'
      },
      {
        var: 'multiplierRampage',
        type: 'count',
        label: '# of Rampage Kills:',
        tooltip:
          'Rampage grants the following, up to 1000 stacks:\n\t1% increased Movement Speed per 20 Rampage\n\t2% increased Damage per 20 Rampage\nYou lose Rampage if you do not get a Kill within 5 seconds.'
      },
      { var: 'conditionFocused', type: 'check', label: 'Are you Focused?', ifCond: 'Focused' },
      { var: 'buffLifetap', type: 'check', label: 'Do you have Lifetap?', ifCond: 'Lifetap' },
      {
        var: 'buffOnslaught',
        type: 'check',
        label: 'Do you have Onslaught?',
        tooltip:
          "In addition to allowing any 'while you have Onslaught' modifiers to apply,\nthis will enable the Onslaught buff itself. (Grants 20% increased Attack, Cast, and Movement Speed)"
      },
      {
        var: 'minionBuffOnslaught',
        type: 'check',
        label: 'Do your minions have Onslaught?',
        ifFlag: 'haveMinion',
        tooltip:
          "In addition to allowing any 'while your minions have Onslaught' modifiers to apply,\nthis will enable the Onslaught buff itself. (Grants 20% increased Attack, Cast, and Movement Speed)"
      },
      {
        var: 'buffUnholyMight',
        type: 'check',
        label: 'Do you have Unholy Might?',
        tooltip: `This will enable the Unholy Might buff. (Grants 30% of Physical Damage as Extra ^${colorCodes.CHAOS}Chaos ^#Damage)`
      },
      {
        var: 'minionbuffUnholyMight',
        type: 'check',
        label: 'Do your minions have Unholy Might?',
        ifFlag: 'haveMinion',
        tooltip: `This will enable the Unholy Might buff on your minions. (Grants 30% of Physical Damage as Extra ^${colorCodes.CHAOS}Chaos ^#Damage)`
      },
      { var: 'buffPhasing', type: 'check', label: 'Do you have Phasing?', ifCond: 'Phasing' },
      { var: 'buffFortification', type: 'check', label: 'Are you Fortified?' },
      {
        var: 'overrideFortification',
        type: 'count',
        label: '# of Fortification Stacks (if not maximum):',
        ifFlag: 'Condition:Fortified',
        tooltip: 'You have 1% less damage taken from hits per stack of fortification:\nHas a default cap of 20 stacks.'
      },
      {
        var: 'buffTailwind',
        type: 'check',
        label: 'Do you have Tailwind?',
        tooltip:
          "In addition to allowing any 'while you have Tailwind' modifiers to apply,\nthis will enable the Tailwind buff itself. (Grants 8% increased Action Speed)"
      },
      {
        var: 'buffAdrenaline',
        type: 'check',
        label: 'Do you have Adrenaline?',
        tooltip:
          'This will enable the Adrenaline buff, which grants:\n\t100% increased Damage\n\t25% increased Attack, Cast and Movement Speed\n\t10% additional Physical Damage Reduction'
      },
      {
        var: 'buffAlchemistsGenius',
        type: 'check',
        label: "Do you have Alchemist's Genius?",
        ifFlag: 'Condition:CanHaveAlchemistGenius',
        tooltip: "This will enable the Alchemist's Genius buff:\n20% increased Flask Charges gained\n10% increased effect of Flasks"
      },
      {
        var: 'buffVaalArcLuckyHits',
        type: 'check',
        label: "Do you have Vaal Arc's Lucky Buff?",
        ifFlag: 'Condition:CanBeLucky',
        tooltip: 'Causes Damage with Arc Hits to be rolled twice, and the maximum roll used.'
      },
      {
        var: 'buffElusive',
        type: 'check',
        label: 'Are you Elusive?',
        ifFlag: 'Condition:CanBeElusive',
        tooltip:
          "In addition to allowing any 'while Elusive' modifiers to apply,\nthis will enable the Elusive buff itself:\n\t15% Chance to Avoid all Damage from Hits\n\t30% increased Movement Speed\nThe effect of Elusive decays over time."
      },
      {
        var: 'overrideBuffElusive',
        type: 'count',
        label: 'Effect of Elusive (if not maximum):',
        ifOption: 'buffElusive',
        tooltip: 'If you have a guaranteed source of Elusive, the strongest one will apply. \nYou can change this to see decaying buff values'
      },
      {
        var: 'buffDivinity',
        type: 'check',
        label: 'Do you have Divinity?',
        ifCond: 'Divinity',
        tooltip: 'This will enable the Divinity buff, which grants:\n\t50% more Elemental Damage\n\t20% less Elemental Damage taken'
      },
      { var: 'multiplierDefiance', type: 'count', label: 'Defiance:', ifMult: 'Defiance' },
      {
        var: 'multiplierRage',
        type: 'count',
        label: 'Rage:',
        ifFlag: 'Condition:CanGainRage',
        tooltip:
          'Base Maximum Rage is 50, and inherently grants the following:\n\t1% increased Attack Damage per 1 Rage\n\t1% increased Attack Speed per 2 Rage\n\t1% increased Movement Speed per 5 Rage\nYou lose 1 Rage every 0.5 seconds if you have not been Hit or gained Rage Recently.'
      },
      {
        var: 'conditionLeeching',
        type: 'check',
        label: 'Are you Leeching?',
        ifCond: 'Leeching',
        tooltip: `You will automatically be considered to be Leeching if you have '^${colorCodes.LIFE}Life ^#Leech effects are not removed at Full ^${colorCodes.LIFE}Life^#',\nbut you can use this option to force it if necessary.`
      },
      {
        var: 'conditionLeechingLife',
        type: 'check',
        label: `Are you Leeching ^${colorCodes.LIFE}Life^#?`,
        ifCond: 'LeechingLife',
        implyCond: 'Leeching'
      },
      {
        var: 'conditionLeechingEnergyShield',
        type: 'check',
        label: `Are you Leeching ^${colorCodes.ES}Energy Shield^#?`,
        ifCond: 'LeechingEnergyShield',
        implyCond: 'Leeching'
      },
      {
        var: 'conditionLeechingMana',
        type: 'check',
        label: `Are you Leeching ^${colorCodes.MANA}Mana^#?`,
        ifCond: 'LeechingMana',
        implyCond: 'Leeching'
      },
      {
        var: 'conditionUsingFlask',
        type: 'check',
        label: 'Do you have a Flask active?',
        ifCond: 'UsingFlask',
        tooltip: 'This is automatically enabled if you have a flask active,\nbut you can use this option to force it if necessary.'
      },
      {
        var: 'conditionHaveTotem',
        type: 'check',
        label: 'Do you have a Totem summoned?',
        ifCond: 'HaveTotem',
        tooltip: 'You will automatically be considered to have a Totem if your main skill is a Totem,\nbut you can use this option to force it if necessary.'
      },
      {
        var: 'conditionSummonedTotemRecently',
        type: 'check',
        label: 'Have you Summoned a Totem Recently?',
        ifCond: 'SummonedTotemRecently',
        tooltip:
          'You will automatically be considered to have Summoned a Totem Recently if your main skill is a Totem,\nbut you can use this option to force it if necessary.'
      },
      {
        var: 'TotemsSummoned',
        type: 'count',
        label: '# of Summoned Totems (if not maximum):',
        ifSkillList: [
          'Spell Totem',
          'Searing Bond',
          'Ballista Totem',
          'Siege Ballista',
          'Artillery Ballista',
          'Shrapnel Ballista',
          'Ancestral Protector',
          'Ancestral Warchief',
          'Vaal Ancestral Warchief',
          'Earthbreaker'
        ],
        tooltip: "This also implies that you have a Totem summoned.\nThis will affect all 'per Summoned Totem' modifiers, even for non-Totem skills."
      },
      {
        var: 'conditionSummonedGolemInPast8Sec',
        type: 'check',
        label: 'Summoned a Golem in the past 8 Seconds?',
        ifCond: 'SummonedGolemInPast8Sec',
        implyCond: 'SummonedGolemInPast10Sec'
      },
      {
        var: 'conditionSummonedGolemInPast10Sec',
        type: 'check',
        label: 'Summoned a Golem in the past 10 Seconds?',
        ifCond: 'SummonedGolemInPast10Sec'
      },
      { var: 'multiplierNearbyAlly', type: 'count', label: '# of Nearby Allies:', ifMult: 'NearbyAlly' },
      { var: 'multiplierNearbyCorpse', type: 'count', label: '# of Nearby Corpses:', ifMult: 'NearbyCorpse' },
      { var: 'multiplierSummonedMinion', type: 'count', label: '# of Summoned Minions:', ifMult: 'SummonedMinion' },
      {
        var: 'conditionOnConsecratedGround',
        type: 'check',
        label: 'Are you on Consecrated Ground?',
        tooltip: `In addition to allowing any 'while on Consecrated Ground' modifiers to apply,\nConsecrated Ground grants 5% ^${colorCodes.LIFE}Life ^#Regeneration to players and allies.`
      },
      {
        var: 'conditionOnFungalGround',
        type: 'check',
        label: 'Are you on Fungal Ground?',
        ifCond: 'OnFungalGround',
        tooltip: `Allies on your Fungal Ground gain 10% of Non-Chaos Damage as extra ^${colorCodes.CHAOS}Chaos ^#Damage.`
      },
      {
        var: 'conditionOnBurningGround',
        type: 'check',
        label: `Are you on ^${colorCodes.FIRE}Burning ^#Ground?`,
        ifCond: 'OnBurningGround',
        implyCond: 'Burning',
        tooltip: `This also implies that you are ^${colorCodes.FIRE}Burning^#.`
      },
      {
        var: 'conditionOnChilledGround',
        type: 'check',
        label: `Are you on ^${colorCodes.COLD}Chilled ^#Ground?`,
        ifCond: 'OnChilledGround',
        implyCond: 'Chilled',
        tooltip: `This also implies that you are ^${colorCodes.COLD}Chilled^#.`
      },
      {
        var: 'conditionOnShockedGround',
        type: 'check',
        label: `Are you on ^${colorCodes.LIGHTNING}Shocked ^#Ground?`,
        ifCond: 'OnShockedGround',
        implyCond: 'Shocked',
        tooltip: `This also implies that you are ^${colorCodes.LIGHTNING}Shocked^#.`
      },
      { var: 'conditionBlinded', type: 'check', label: 'Are you Blinded?', ifCond: 'Blinded' },
      { var: 'conditionBurning', type: 'check', label: `Are you ^${colorCodes.FIRE}Burning^#?`, ifCond: 'Burning' },
      {
        var: 'conditionIgnited',
        type: 'check',
        label: `Are you ^${colorCodes.FIRE}Ignited^#?`,
        ifCond: 'Ignited',
        implyCond: 'Burning',
        tooltip: `This also implies that you are ^${colorCodes.FIRE}Burning^#.`
      },
      { var: 'conditionChilled', type: 'check', label: `Are you ^${colorCodes.COLD}Chilled^#?`, ifCond: 'Chilled' },
      {
        var: 'conditionChilledEffect',
        type: 'count',
        label: `Effect of ^${colorCodes.COLD}Chill^#:`,
        ifOption: 'conditionChilled'
      },
      {
        var: 'conditionSelfChill',
        type: 'check',
        label: `Did you ^${colorCodes.COLD}Chill ^#yourself?`,
        ifOption: 'conditionChilled'
      },
      {
        var: 'conditionFrozen',
        type: 'check',
        label: `Are you ^${colorCodes.COLD}Frozen^#?`,
        ifCond: 'Frozen',
        implyCond: 'Chilled',
        tooltip: `This also implies that you are ^${colorCodes.COLD}Chilled^#.`
      },
      {
        var: 'conditionShocked',
        type: 'check',
        label: `Are you ^${colorCodes.LIGHTNING}Shocked^#?`,
        ifCond: 'Shocked'
      },
      { var: 'conditionBleeding', type: 'check', label: 'Are you Bleeding?', ifCond: 'Bleeding' },
      { var: 'conditionPoisoned', type: 'check', label: 'Are you Poisoned?', ifCond: 'Poisoned' },
      {
        var: 'multiplierPoisonOnSelf',
        type: 'count',
        label: '# of Poison on You:',
        ifMult: 'PoisonStack',
        implyCond: 'Poisoned',
        tooltip: 'This also implies that you are Poisoned.'
      },
      {
        var: 'conditionAgainstDamageOverTime',
        type: 'check',
        label: 'Are you against Damage over Time?',
        ifCond: 'AgainstDamageOverTime'
      },
      { var: 'multiplierNearbyEnemies', type: 'count', label: '# of nearby Enemies:', ifMult: 'NearbyEnemies' },
      {
        var: 'multiplierNearbyRareOrUniqueEnemies',
        type: 'countAllowZero',
        label: '# of nearby Rare or Unique Enemies:',
        ifMult: 'NearbyRareOrUniqueEnemies'
      },
      {
        var: 'conditionHitRecently',
        type: 'check',
        label: 'Have you Hit Recently?',
        ifCond: 'HitRecently',
        tooltip:
          'You will automatically be considered to have Hit Recently if your main skill Hits and is self-cast,\nbut you can use this option to force it if necessary.'
      },
      {
        var: 'conditionCritRecently',
        type: 'check',
        label: 'Have you Crit Recently?',
        ifCond: 'CritRecently',
        implyCond: 'SkillCritRecently',
        tooltip: 'This also implies that your Skills have Crit Recently.'
      },
      {
        var: 'conditionSkillCritRecently',
        type: 'check',
        label: 'Have your Skills Crit Recently?',
        ifCond: 'SkillCritRecently'
      },
      {
        var: 'conditionCritWithHeraldSkillRecently',
        type: 'check',
        label: 'Have your Herald Skills Crit Recently?',
        ifCond: 'CritWithHeraldSkillRecently',
        implyCond: 'SkillCritRecently',
        tooltip: 'This also implies that your Skills have Crit Recently.'
      },
      {
        var: 'LostNonVaalBuffRecently',
        type: 'check',
        label: 'Lost a Non-Vaal Guard Skill buff recently?',
        ifCond: 'LostNonVaalBuffRecently'
      },
      {
        var: 'conditionNonCritRecently',
        type: 'check',
        label: 'Have you dealt a Non-Crit Recently?',
        ifCond: 'NonCritRecently'
      },
      {
        var: 'conditionChannelling',
        type: 'check',
        label: 'Are you Channelling?',
        ifCond: 'Channelling',
        tooltip:
          'You will automatically be considered to be Channeling if your main skill is a channelled skill,\nbut you can use this option to force it if necessary.'
      },
      {
        var: 'conditionHitRecentlyWithWeapon',
        type: 'check',
        label: 'Have you Hit Recently with Your Weapon?',
        ifCond: 'HitRecentlyWithWeapon',
        tooltip: 'This also implies that you have Hit Recently.'
      },
      { var: 'conditionKilledRecently', type: 'check', label: 'Have you Killed Recently?', ifCond: 'KilledRecently' },
      {
        var: 'multiplierKilledRecently',
        type: 'count',
        label: '# of Enemies Killed Recently:',
        ifMult: 'EnemyKilledRecently',
        implyCond: 'KilledRecently',
        tooltip: 'This also implies that you have Killed Recently.'
      },
      {
        var: 'conditionKilledLast3Seconds',
        type: 'check',
        label: 'Have you Killed in the last 3 Seconds?',
        ifCond: 'KilledLast3Seconds',
        implyCond: 'KilledRecently',
        tooltip: 'This also implies that you have Killed Recently.'
      },
      {
        var: 'conditionKilledPosionedLast2Seconds',
        type: 'check',
        label: 'Killed a poisoned enemy in the last 2 Seconds?',
        ifCond: 'KilledPosionedLast2Seconds',
        implyCond: 'KilledRecently',
        tooltip: 'This also implies that you have Killed Recently.'
      },
      {
        var: 'conditionTotemsNotSummonedInPastTwoSeconds',
        type: 'check',
        label: 'No summoned Totems in the past 2 seconds?',
        ifCond: 'NoSummonedTotemsInPastTwoSeconds'
      },
      {
        var: 'conditionTotemsKilledRecently',
        type: 'check',
        label: 'Have your Totems Killed Recently?',
        ifCond: 'TotemsKilledRecently'
      },
      {
        var: 'conditionUsedBrandRecently',
        type: 'check',
        label: 'Have you used a Brand Skill recently?',
        ifCond: 'UsedBrandRecently'
      },
      {
        var: 'multiplierTotemsKilledRecently',
        type: 'count',
        label: '# of Enemies Killed by Totems Recently:',
        ifMult: 'EnemyKilledByTotemsRecently',
        implyCond: 'TotemsKilledRecently',
        tooltip: 'This also implies that your Totems have Killed Recently.'
      },
      {
        var: 'conditionMinionsKilledRecently',
        type: 'check',
        label: 'Have your Minions Killed Recently?',
        ifCond: 'MinionsKilledRecently'
      },
      {
        var: 'conditionMinionsDiedRecently',
        type: 'check',
        label: 'Has a Minion Died Recently?',
        ifCond: 'MinionsDiedRecently'
      },
      {
        var: 'multiplierMinionsKilledRecently',
        type: 'count',
        label: '# of Enemies Killed by Minions Recently:',
        ifMult: 'EnemyKilledByMinionsRecently',
        implyCond: 'MinionsKilledRecently',
        tooltip: 'This also implies that your Minions have Killed Recently.'
      },
      {
        var: 'conditionKilledAffectedByDoT',
        type: 'check',
        label: 'Killed enemy affected by your DoT Recently?',
        ifCond: 'KilledAffectedByDotRecently'
      },
      {
        var: 'multiplierShockedEnemyKilledRecently',
        type: 'count',
        label: `# of ^${colorCodes.LIGHTNING}Shocked ^#Enemies Killed Recently:`,
        ifMult: 'ShockedEnemyKilledRecently'
      },
      {
        var: 'conditionFrozenEnemyRecently',
        type: 'check',
        label: `Have you ^${colorCodes.COLD}Frozen ^#an enemy Recently?`,
        ifCond: 'FrozenEnemyRecently'
      },
      {
        var: 'conditionChilledEnemyRecently',
        type: 'check',
        label: `Have you ^${colorCodes.COLD}Chilled ^#an enemy Recently?`,
        ifCond: 'ChilledEnemyRecently'
      },
      {
        var: 'conditionShatteredEnemyRecently',
        type: 'check',
        label: `Have you ^${colorCodes.COLD}Shattered ^#an enemy Recently?`,
        ifCond: 'ShatteredEnemyRecently'
      },
      {
        var: 'conditionIgnitedEnemyRecently',
        type: 'check',
        label: `Have you ^${colorCodes.FIRE}Ignited ^#an enemy Recently?`,
        ifCond: 'IgnitedEnemyRecently'
      },
      {
        var: 'conditionShockedEnemyRecently',
        type: 'check',
        label: `Have you ^${colorCodes.LIGHTNING}Shocked ^#an enemy Recently?`,
        ifCond: 'ShockedEnemyRecently'
      },
      {
        var: 'conditionStunnedEnemyRecently',
        type: 'check',
        label: 'Have you Stunned an enemy Recently?',
        ifCond: 'StunnedEnemyRecently'
      },
      {
        var: 'multiplierPoisonAppliedRecently',
        type: 'count',
        label: '# of Poisons applied Recently:',
        ifMult: 'PoisonAppliedRecently'
      },
      {
        var: 'multiplierLifeSpentRecently',
        type: 'count',
        label: `# of ^${colorCodes.LIFE}Life ^#spent Recently:`,
        ifMult: 'LifeSpentRecently'
      },
      {
        var: 'multiplierManaSpentRecently',
        type: 'count',
        label: `# of ^${colorCodes.MANA}Mana ^#spent Recently:`,
        ifMult: 'ManaSpentRecently'
      },
      {
        var: 'conditionBeenHitRecently',
        type: 'check',
        label: 'Have you been Hit Recently?',
        ifCond: 'BeenHitRecently'
      },
      {
        var: 'multiplierBeenHitRecently',
        type: 'count',
        label: '# of times you have been Hit Recently:',
        ifMult: 'BeenHitRecently'
      },
      {
        var: 'conditionBeenHitByAttackRecently',
        type: 'check',
        label: 'Have you been Hit by an Attack Recently?',
        ifCond: 'BeenHitByAttackRecently'
      },
      {
        var: 'conditionBeenCritRecently',
        type: 'check',
        label: 'Have you been Crit Recently?',
        ifCond: 'BeenCritRecently'
      },
      {
        var: 'conditionConsumed12SteelShardsRecently',
        type: 'check',
        label: 'Consumed 12 Steel Shards Recently?',
        ifCond: 'Consumed12SteelShardsRecently'
      },
      {
        var: 'conditionGainedPowerChargeRecently',
        type: 'check',
        label: 'Gained a Power Charge Recently?',
        ifCond: 'GainedPowerChargeRecently'
      },
      {
        var: 'conditionGainedFrenzyChargeRecently',
        type: 'check',
        label: 'Gained a Frenzy Charge Recently?',
        ifCond: 'GainedFrenzyChargeRecently'
      },
      {
        var: 'conditionBeenSavageHitRecently',
        type: 'check',
        label: 'Have you taken a Savage Hit Recently?',
        ifCond: 'BeenSavageHitRecently',
        implyCond: 'BeenHitRecently',
        tooltip: 'This also implies that you have been Hit Recently.'
      },
      {
        var: 'conditionHitByFireDamageRecently',
        type: 'check',
        label: `Have you been hit by ^${colorCodes.FIRE}Fire ^#Recently?`,
        ifCond: 'HitByFireDamageRecently',
        implyCond: 'BeenHitRecently',
        tooltip: 'This also implies that you have been Hit Recently.'
      },
      {
        var: 'conditionHitByColdDamageRecently',
        type: 'check',
        label: `Have you been hit by ^${colorCodes.COLD}Cold ^#Recently?`,
        ifCond: 'HitByColdDamageRecently',
        implyCond: 'BeenHitRecently',
        tooltip: 'This also implies that you have been Hit Recently.'
      },
      {
        var: 'conditionHitByLightningDamageRecently',
        type: 'check',
        label: `Have you been hit by ^${colorCodes.LIGHTNING}Light^#. Recently?`,
        ifCond: 'HitByLightningDamageRecently',
        implyCond: 'BeenHitRecently',
        tooltip: 'This also implies that you have been Hit Recently.'
      },
      {
        var: 'conditionHitBySpellDamageRecently',
        type: 'check',
        label: 'Have you taken Spell Damage Recently?',
        ifCond: 'HitBySpellDamageRecently',
        implyCond: 'BeenHitRecently',
        tooltip: 'This also implies that you have been Hit Recently.'
      },
      {
        var: 'conditionTakenFireDamageFromEnemyHitRecently',
        type: 'check',
        label: `Taken ^${colorCodes.FIRE}Fire ^#Damage from enemy Hit Recently?`,
        ifCond: 'TakenFireDamageFromEnemyHitRecently',
        implyCond: 'BeenHitRecently',
        tooltip: 'This also implies that you have been Hit Recently.'
      },
      {
        var: 'conditionBlockedRecently',
        type: 'check',
        label: 'Have you Blocked Recently?',
        ifCond: 'BlockedRecently'
      },
      {
        var: 'conditionBlockedAttackRecently',
        type: 'check',
        label: 'Have you Blocked an Attack Recently?',
        ifCond: 'BlockedAttackRecently',
        implyCond: 'BlockedRecently',
        tooltip: 'This also implies that you have Blocked Recently.'
      },
      {
        var: 'conditionBlockedSpellRecently',
        type: 'check',
        label: 'Have you Blocked a Spell Recently?',
        ifCond: 'BlockedSpellRecently',
        implyCond: 'BlockedRecently',
        tooltip: 'This also implies that you have Blocked Recently.'
      },
      {
        var: 'conditionEnergyShieldRechargeRecently',
        type: 'check',
        label: `^${colorCodes.ES}Energy Shield ^#Recharge started Recently?`,
        ifCond: 'EnergyShieldRechargeRecently'
      },
      {
        var: 'conditionStoppedTakingDamageOverTimeRecently',
        type: 'check',
        label: 'Have you stopped taking DoT recently?',
        ifCond: 'StoppedTakingDamageOverTimeRecently'
      },
      {
        var: 'conditionConvergence',
        type: 'check',
        label: 'Do you have Convergence?',
        ifFlag: 'Condition:CanGainConvergence'
      },
      {
        var: 'buffPendulum',
        type: 'list',
        label: 'Is Pendulum of Destruction active?',
        ifCond: 'PendulumOfDestructionAreaOfEffect',
        list: [
          { value: 0, label: 'None' },
          { value: 'AREA', label: 'Area of Effect' },
          {
            value: 'DAMAGE',
            label: 'Elemental Damage'
          }
        ]
      },
      {
        var: 'buffConflux',
        type: 'list',
        label: 'Conflux Buff:',
        ifCond: 'ChillingConflux',
        list: [
          { value: 0, label: 'None' },
          { value: 'CHILLING', label: `^${colorCodes.COLD}Chilling^#` },
          {
            value: 'SHOCKING',
            label: `^${colorCodes.LIGHTNING}Shocking^#`
          },
          { value: 'IGNITING', label: `^${colorCodes.FIRE}Igniting^#` },
          {
            value: 'ALL',
            label: `^${colorCodes.COLD}Chill ^#+ ^${colorCodes.LIGHTNING}Shock ^#+ ^${colorCodes.FIRE}Ignite^#`
          }
        ]
      },
      { var: 'buffBastionOfHope', type: 'check', label: 'Is Bastion of Hope active?', ifCond: 'BastionOfHopeActive' },
      {
        var: 'buffNgamahuFlamesAdvance',
        type: 'check',
        label: "Is Ngamahu, Flame's Advance active?",
        ifCond: 'NgamahuFlamesAdvance'
      },
      {
        var: 'buffHerEmbrace',
        type: 'check',
        label: 'Are you in Her Embrace?',
        ifCond: 'HerEmbrace',
        tooltip: 'This option is specific to Oni-Goroshi.'
      },
      {
        var: 'conditionUsedSkillRecently',
        type: 'check',
        label: 'Have you used a Skill Recently?',
        ifCond: 'UsedSkillRecently'
      },
      {
        var: 'multiplierSkillUsedRecently',
        type: 'count',
        label: '# of Skills Used Recently:',
        ifMult: 'SkillUsedRecently',
        implyCond: 'UsedSkillRecently'
      },
      {
        var: 'conditionAttackedRecently',
        type: 'check',
        label: 'Have you Attacked Recently?',
        ifCond: 'AttackedRecently',
        implyCond: 'UsedSkillRecently',
        tooltip:
          'This also implies that you have used a Skill Recently.\nYou will automatically be considered to have Attacked Recently if your main skill is an attack,\nbut you can use this option to force it if necessary.'
      },
      {
        var: 'conditionCastSpellRecently',
        type: 'check',
        label: 'Have you Cast a Spell Recently?',
        ifCond: 'CastSpellRecently',
        implyCond: 'UsedSkillRecently',
        tooltip:
          'This also implies that you have used a Skill Recently.\nYou will automatically be considered to have Cast a Spell Recently if your main skill is a spell,\nbut you can use this option to force it if necessary.'
      },
      {
        var: 'conditionCastLast1Seconds',
        type: 'check',
        label: 'Have you Cast a Spell in the last second?',
        ifCond: 'CastLast1Seconds',
        implyCond: 'CastSpellRecently'
      },
      {
        var: 'multiplierCastLast8Seconds',
        type: 'count',
        label: 'How many spells cast in the last 8 seconds?',
        ifMult: 'CastLast8Seconds',
        tooltip: 'Only non-instant spells you cast count'
      },
      {
        var: 'conditionUsedFireSkillRecently',
        type: 'check',
        label: `Have you used a ^${colorCodes.FIRE}Fire ^#Skill Recently?`,
        ifCond: 'UsedFireSkillRecently',
        implyCond: 'UsedSkillRecently',
        tooltip: 'This also implies that you have used a Skill Recently.'
      },
      {
        var: 'conditionUsedColdSkillRecently',
        type: 'check',
        label: `Have you used a ^${colorCodes.COLD}Cold ^#Skill Recently?`,
        ifCond: 'UsedColdSkillRecently',
        implyCond: 'UsedSkillRecently',
        tooltip: 'This also implies that you have used a Skill Recently.'
      },
      {
        var: 'conditionUsedMinionSkillRecently',
        type: 'check',
        label: 'Have you used a Minion Skill Recently?',
        ifCond: 'UsedMinionSkillRecently',
        implyCond: 'UsedSkillRecently',
        tooltip:
          'This also implies that you have used a Skill Recently.\nYou will automatically be considered to have used a Minion skill Recently if your main skill is a Minion skill,\nbut you can use this option to force it if necessary.'
      },
      {
        var: 'conditionUsedTravelSkillRecently',
        type: 'check',
        label: 'Have you used a Travel Skill Recently?',
        ifCond: 'UsedTravelSkillRecently',
        implyCond: 'UsedSkillRecently',
        tooltip: 'This also implies that you have used a Skill Recently+'
      },
      {
        var: 'conditionUsedDashRecently',
        type: 'check',
        label: 'Have you cast Dash Recently?',
        ifCond: 'CastDashRecently',
        implyCondList: ['UsedTravelSkillRecently', 'UsedMovementSkillRecently', 'UsedSkillRecently'],
        tooltip: 'This also implies that you have used a Skill Recently+'
      },
      {
        var: 'conditionUsedMovementSkillRecently',
        type: 'check',
        label: 'Have you used a Movement Skill Recently?',
        ifCond: 'UsedMovementSkillRecently',
        implyCond: 'UsedSkillRecently',
        tooltip:
          'This also implies that you have used a Skill Recently.\nYou will automatically be considered to have used a Movement skill Recently if your main skill is a movement skill,\nbut you can use this option to force it if necessary.'
      },
      {
        var: 'conditionUsedVaalSkillRecently',
        type: 'check',
        label: 'Have you used a Vaal Skill Recently?',
        ifCond: 'UsedVaalSkillRecently',
        implyCond: 'UsedSkillRecently',
        tooltip:
          'This also implies that you have used a Skill Recently.\nYou will automatically be considered to have used a Vaal skill Recently if your main skill is a Vaal skill,\nbut you can use this option to force it if necessary.'
      },
      {
        var: 'conditionSoulGainPrevention',
        type: 'check',
        label: 'Do you have Soul Gain Prevention?',
        ifCond: 'SoulGainPrevention'
      },
      {
        var: 'conditionUsedWarcryRecently',
        type: 'check',
        label: 'Have you used a Warcry Recently?',
        ifCond: 'UsedWarcryRecently',
        implyCondList: ['UsedWarcryInPast8Seconds', 'UsedSkillRecently'],
        tooltip: 'This also implies that you have used a Skill Recently.'
      },
      {
        var: 'conditionUsedWarcryInPast8Seconds',
        type: 'check',
        label: 'Used a Warcry in the past 8 seconds?',
        ifCond: 'UsedWarcryInPast8Seconds'
      },
      {
        var: 'multiplierMineDetonatedRecently',
        type: 'count',
        label: '# of Mines Detonated Recently:',
        ifMult: 'MineDetonatedRecently'
      },
      {
        var: 'multiplierTrapTriggeredRecently',
        type: 'count',
        label: '# of Traps Triggered Recently:',
        ifMult: 'TrapTriggeredRecently'
      },
      {
        var: 'conditionThrownTrapOrMineRecently',
        type: 'check',
        label: 'Have you thrown a Trap or Mine Recently?',
        ifCond: 'TrapOrMineThrownRecently'
      },
      {
        var: 'conditionCursedEnemyRecently',
        type: 'check',
        label: 'Have you Cursed an enemy Recently?',
        ifCond: 'CursedEnemyRecently'
      },
      {
        var: 'conditionCastMarkRecently',
        type: 'check',
        label: 'Have you cast a Mark Spell Recently?',
        ifCond: 'CastMarkRecently'
      },
      {
        var: 'conditionSpawnedCorpseRecently',
        type: 'check',
        label: 'Spawned a corpse Recently?',
        ifCond: 'SpawnedCorpseRecently'
      },
      {
        var: 'conditionConsumedCorpseRecently',
        type: 'check',
        label: 'Consumed a corpse Recently?',
        ifCond: 'ConsumedCorpseRecently'
      },
      {
        var: 'conditionConsumedCorpseInPast2Sec',
        type: 'check',
        label: 'Consumed a corpse in the past 2s?',
        ifCond: 'ConsumedCorpseInPast2Sec',
        implyCond: 'ConsumedCorpseRecently',
        tooltip: "This also implies you have 'Consumed a corpse Recently'"
      },
      {
        var: 'multiplierCorpseConsumedRecently',
        type: 'count',
        label: '# of Corpses Consumed Recently:',
        ifMult: 'CorpseConsumedRecently',
        implyCond: 'ConsumedCorpseRecently'
      },
      {
        var: 'multiplierWarcryUsedRecently',
        type: 'count',
        label: '# of Warcries Used Recently:',
        ifMult: 'WarcryUsedRecently',
        implyCondList: ['UsedWarcryRecently', 'UsedWarcryInPast8Seconds', 'UsedSkillRecently'],
        tooltip: "This also implies you have 'Used a Warcry Recently', 'Used a Warcry in the past 8 seconds', and 'Used a Skill Recently'"
      },
      {
        var: 'conditionTauntedEnemyRecently',
        type: 'check',
        label: 'Taunted an enemy Recently?',
        ifCond: 'TauntedEnemyRecently'
      },
      {
        var: 'conditionLostEnduranceChargeInPast8Sec',
        type: 'check',
        label: 'Lost an Endurance Charge in the past 8s?',
        ifCond: 'LostEnduranceChargeInPast8Sec'
      },
      {
        var: 'multiplierEnduranceChargesLostRecently',
        type: 'count',
        label: '# of Endurance Charges lost Recently:',
        ifMult: 'EnduranceChargesLostRecently',
        implyCond: 'LostEnduranceChargeInPast8Sec'
      },
      {
        var: 'conditionBlockedHitFromUniqueEnemyInPast10Sec',
        type: 'check',
        label: 'Blocked a Hit from a Unique enemy in the past 10s?',
        ifCond: 'BlockedHitFromUniqueEnemyInPast10Sec'
      },
      {
        var: 'BlockedPast10Sec',
        type: 'count',
        label: "Number of times you've Blocked in the past 10s",
        ifCond: 'BlockedHitFromUniqueEnemyInPast10Sec'
      },
      {
        var: 'conditionImpaledRecently',
        type: 'check',
        ifCond: 'ImpaledRecently',
        label: 'Impaled an enemy recently?'
      },
      {
        var: 'multiplierImpalesOnEnemy',
        type: 'countAllowZero',
        label: '# of Impales on enemy (if not maximum):',
        ifFlag: 'impale'
      },
      {
        var: 'multiplierBleedsOnEnemy',
        type: 'count',
        label: '# of Bleeds on enemy (if not maximum):',
        ifFlag: 'Condition:HaveCrimsonDance',
        tooltip: 'Sets current number of Bleeds on the enemy if using the Crimson Dance keystone.\nThis also implies that the enemy is Bleeding.'
      },
      {
        var: 'multiplierFragileRegrowth',
        type: 'count',
        label: '# of Fragile Regrowth Stacks:',
        ifMult: 'FragileRegrowthCount'
      },
      {
        var: 'conditionKilledUniqueEnemy',
        type: 'check',
        label: 'Killed a Rare or Unique enemy Recently?',
        ifCond: 'KilledUniqueEnemy'
      },
      {
        var: 'conditionHaveArborix',
        type: 'check',
        label: 'Do you have Iron Reflexes?',
        ifFlag: 'Condition:HaveArborix',
        tooltip: 'This option is specific to Arborix.'
      },
      {
        var: 'conditionHaveAugyre',
        type: 'list',
        label: 'Augyre rotating buff:',
        ifFlag: 'Condition:HaveAugyre',
        list: [
          { value: 'EleOverload', label: 'Elemental Overload' },
          {
            value: 'ResTechnique',
            label: 'Resolute Technique'
          }
        ],
        tooltip: 'This option is specific to Augyre.'
      },
      {
        var: 'conditionHaveVulconus',
        type: 'check',
        label: 'Do you have Avatar Of Fire?',
        ifFlag: 'Condition:HaveVulconus',
        tooltip: 'This option is specific to Vulconus.'
      },
      {
        var: 'conditionHaveManaStorm',
        type: 'check',
        label: `Do you have Manastorm's ^${colorCodes.LIGHTNING}Lightning ^#Buff?`,
        ifFlag: 'Condition:HaveManaStorm',
        tooltip: `This option enables Manastorm's ^${colorCodes.LIGHTNING}Lightning ^#Damage Buff.\n(When you cast a Spell, Sacrifice all ^${colorCodes.MANA}Mana ^#to gain Added Maximum ^${colorCodes.LIGHTNING}Lightning ^#Damage\nequal to 25% of Sacrificed ^${colorCodes.MANA}Mana ^#for 4 seconds)`
      },
      {
        var: 'buffFanaticism',
        type: 'check',
        label: 'Do you have Fanaticism?',
        ifFlag: 'Condition:CanGainFanaticism',
        tooltip: `This will enable the Fanaticism buff itself. (Grants 75% more cast speed, reduced ^${colorCodes.MANA}mana ^#cost, and increased area of effect)`
      }
    ]
  },
  {
    name: 'For Effective DPS',
    variables: [
      { var: 'critChanceLucky', type: 'check', label: 'Is your Crit Chance Lucky?' },
      { var: 'skillForkCount', type: 'count', label: '# of times Skill has Forked:', ifFlag: 'forking' },
      { var: 'skillChainCount', type: 'count', label: '# of times Skill has Chained:', ifFlag: 'chaining' },
      { var: 'skillPierceCount', type: 'count', label: '# of times Skill has Pierced:', ifFlag: 'piercing' },
      { var: 'meleeDistance', type: 'count', label: 'Melee distance to enemy:', ifFlag: 'melee' },
      { var: 'projectileDistance', type: 'count', label: 'Projectile travel distance:', ifFlag: 'projectile' },
      { var: 'conditionAtCloseRange', type: 'check', label: 'Is the enemy at Close Range?', ifCond: 'AtCloseRange' },
      { var: 'conditionEnemyMoving', type: 'check', label: 'Is the enemy Moving?' },
      {
        var: 'conditionEnemyFullLife',
        type: 'check',
        label: `Is the enemy on Full ^${colorCodes.LIFE}Life^#?`,
        ifEnemyCond: 'FullLife'
      },
      {
        var: 'conditionEnemyLowLife',
        type: 'check',
        label: `Is the enemy on Low ^${colorCodes.LIFE}Life^#?`,
        ifEnemyCond: 'LowLife'
      },
      {
        var: 'conditionEnemyCursed',
        type: 'check',
        label: 'Is the enemy Cursed?',
        ifEnemyCond: 'Cursed',
        tooltip:
          'The enemy will automatically be considered to be Cursed if you have at least one curse enabled,\nbut you can use this option to force it if necessary.'
      },
      { var: 'conditionEnemyBleeding', type: 'check', label: 'Is the enemy Bleeding?' },
      {
        var: 'multiplierRuptureStacks',
        type: 'count',
        label: '# of Rupture stacks?',
        ifFlag: 'Condition:CanInflictRupture',
        tooltip: 'Rupture applies 25% more bleed damage and 25% faster bleeds for 3 seconds, up to 3 stacks'
      },
      { var: 'conditionEnemyPoisoned', type: 'check', label: 'Is the enemy Poisoned?', ifEnemyCond: 'Poisoned' },
      { var: 'multiplierPoisonOnEnemy', type: 'count', label: '# of Poison on enemy:', implyCond: 'Poisoned' },
      {
        var: 'multiplierWitheredStackCount',
        type: 'count',
        label: '# of Withered Stacks:',
        ifFlag: 'Condition:CanWither',
        tooltip: `Withered applies 6% increased ^${colorCodes.CHAOS}Chaos ^#Damage Taken to the enemy, up to 15 stacks.`
      },
      {
        var: 'multiplierCorrosionStackCount',
        type: 'count',
        label: '# of Corrosion Stacks:',
        ifFlag: 'Condition:CanCorrode',
        tooltip: `Each stack of Corrosion applies -5000 to total Armour and -1000 to total ^${colorCodes.POSITIVE}Evasion Rating ^#to the enemy.\nCorrosion lasts 4 seconds and refreshes the duration of existing Corrosion stacks\nCorrosion has no stack limit`
      },
      {
        var: 'multiplierEnsnaredStackCount',
        type: 'count',
        label: '# of Ensnare Stacks:',
        ifSkill: 'Ensnaring Arrow',
        tooltip:
          'While ensnared, enemies take increased Projectile Damage from Attack Hits\nEnsnared enemies always count as moving, and have less movement speed while trying to break the snare.'
      },
      { var: 'conditionEnemyMaimed', type: 'check', label: 'Is the enemy Maimed?' },
      { var: 'conditionEnemyHindered', type: 'check', label: 'Is the enemy Hindered?', ifEnemyCond: 'Hindered' },
      {
        var: 'conditionEnemyBlinded',
        type: 'check',
        label: 'Is the enemy Blinded?',
        tooltip: `In addition to allowing 'against Blinded Enemies' modifiers to apply,\n Blind applies the following effects.\n -20% Accuracy \n -20% ^${colorCodes.POSITIVE}Evasion^#`
      },
      {
        var: 'overrideBuffBlinded',
        type: 'count',
        label: 'Effect of Blind (if not maximum):',
        ifOption: 'conditionEnemyBlinded',
        tooltip: 'If you have a guaranteed source of Blind, the strongest one will apply.'
      },
      { var: 'conditionEnemyTaunted', type: 'check', label: 'Is the enemy Taunted?', ifEnemyCond: 'Taunted' },
      { var: 'conditionEnemyBurning', type: 'check', label: `Is the enemy ^${colorCodes.FIRE}Burning^#?` },
      {
        var: 'conditionEnemyIgnited',
        type: 'check',
        label: `Is the enemy ^${colorCodes.FIRE}Ignited^#?`,
        implyCond: 'Burning',
        tooltip: `This also implies that the enemy is ^${colorCodes.FIRE}Burning^#.`
      },
      {
        var: 'conditionEnemyScorched',
        type: 'check',
        ifFlag: 'inflictScorch',
        label: `Is the enemy ^${colorCodes.FIRE}Scorched^#?`,
        tooltip: `^${colorCodes.FIRE}Scorched ^#enemies have lowered elemental resistances, up to -30%.\nThis option will also allow you to input the effect of ^${colorCodes.FIRE}Scorched^#.`
      },
      {
        var: 'conditionScorchedEffect',
        type: 'count',
        label: `Effect of ^${colorCodes.FIRE}Scorched^#:`,
        ifOption: 'conditionEnemyScorched',
        tooltip: `This effect will only be applied while you can inflict ^${colorCodes.FIRE}Scorched^#.`
      },
      {
        var: 'conditionEnemyOnScorchedGround',
        type: 'check',
        label: `Is the enemy on ^${colorCodes.FIRE}Scorched ^#Ground?`,
        tooltip: `This also implies that the enemy is ^${colorCodes.FIRE}Scorched^#.`,
        ifEnemyCond: 'OnScorchedGround'
      },
      { var: 'conditionEnemyChilled', type: 'check', label: `Is the enemy ^${colorCodes.COLD}Chilled^#?` },
      {
        var: 'conditionEnemyChilledEffect',
        type: 'count',
        label: `Effect of ^${colorCodes.COLD}Chill^#:`,
        ifOption: 'conditionEnemyChilled'
      },
      {
        var: 'conditionEnemyChilledByYourHits',
        type: 'check',
        ifEnemyCond: 'ChilledByYourHits',
        label: `Is the enemy ^${colorCodes.COLD}Chilled ^#by your Hits?`
      },
      {
        var: 'conditionEnemyFrozen',
        type: 'check',
        label: `Is the enemy ^${colorCodes.COLD}Frozen^#?`,
        implyCond: 'Chilled',
        tooltip: `This also implies that the enemy is ^${colorCodes.COLD}Chilled^#.`
      },
      {
        var: 'conditionEnemyBrittle',
        type: 'check',
        ifFlag: 'inflictBrittle',
        label: `Is the enemy ^${colorCodes.COLD}Brittle^#?`,
        tooltip: `Hits against ^${colorCodes.COLD}Brittle ^#enemies have up to +15% Critical Strike Chance.\nThis option will also allow you to input the effect of ^${colorCodes.COLD}Brittle^#.`
      },
      {
        var: 'conditionBrittleEffect',
        type: 'count',
        label: `Effect of ^${colorCodes.COLD}Brittle^#:`,
        ifOption: 'conditionEnemyBrittle',
        tooltip: `This effect will only be applied while you can inflict ^${colorCodes.COLD}Brittle^#.`
      },
      {
        var: 'conditionEnemyOnBrittleGround',
        type: 'check',
        label: `Is the enemy on ^${colorCodes.COLD}Brittle ^#Ground?`,
        tooltip: `This also implies that the enemy is ^${colorCodes.COLD}Brittle^#.`,
        ifEnemyCond: 'OnBrittleGround'
      },
      {
        var: 'conditionEnemyShocked',
        type: 'check',
        label: `Is the enemy ^${colorCodes.LIGHTNING}Shocked^#?`,
        tooltip: `In addition to allowing any 'against ^${colorCodes.LIGHTNING}Shocked ^#Enemies' modifiers to apply,\nthis will allow you to input the effect of the ^${colorCodes.LIGHTNING}Shock ^#applied to the enemy.`
      },
      {
        var: 'conditionShockEffect',
        type: 'count',
        label: `Effect of ^${colorCodes.LIGHTNING}Shock^#:`,
        tooltip: `If you have a guaranteed source of ^${colorCodes.LIGHTNING}Shock^#,\nthe strongest one will apply instead unless this option would apply a stronger ^${colorCodes.LIGHTNING}Shock^#.`
      },
      {
        var: 'conditionEnemyOnShockedGround',
        type: 'check',
        label: `Is the enemy on ^${colorCodes.LIGHTNING}Shocked ^#Ground?`,
        tooltip: `This also implies that the enemy is ^${colorCodes.LIGHTNING}Shocked^#.`,
        ifEnemyCond: 'OnShockedGround'
      },
      {
        var: 'conditionEnemySapped',
        type: 'check',
        ifFlag: 'inflictSap',
        label: `Is the enemy ^${colorCodes.LIGHTNING}Sapped^#?`,
        tooltip: `^${colorCodes.LIGHTNING}Sapped ^#enemies deal less damage, up to 20%.`
      },
      {
        var: 'conditionSapEffect',
        type: 'count',
        label: `Effect of ^${colorCodes.LIGHTNING}Sap^#:`,
        ifOption: 'conditionEnemySapped',
        tooltip: `If you have a guaranteed source of ^${colorCodes.LIGHTNING}Sap^#,\nthe strongest one will apply instead unless this option would apply a stronger ^${colorCodes.LIGHTNING}Sap^#.`
      },
      {
        var: 'conditionEnemyOnSappedGround',
        type: 'check',
        label: `Is the enemy on ^${colorCodes.LIGHTNING}Sapped ^#Ground?`,
        tooltip: `This also implies that the enemy is ^${colorCodes.LIGHTNING}Sapped^#.`,
        ifEnemyCond: 'OnSappedGround'
      },
      {
        var: 'multiplierFreezeShockIgniteOnEnemy',
        type: 'count',
        label: `# of ^${colorCodes.COLD}Freeze ^#/ ^${colorCodes.LIGHTNING}Shock ^#/ ^${colorCodes.FIRE}Ignite ^#on enemy:`,
        ifMult: 'FreezeShockIgniteOnEnemy'
      },
      {
        var: 'conditionEnemyFireExposure',
        type: 'check',
        label: `Is the enemy Exposed to ^${colorCodes.FIRE}Fire^#?`,
        ifFlag: 'applyFireExposure',
        tooltip: `This applies -10% ^${colorCodes.FIRE}Fire Resistance ^#to the enemy.`
      },
      {
        var: 'conditionEnemyColdExposure',
        type: 'check',
        label: `Is the enemy Exposed to ^${colorCodes.COLD}Cold^#?`,
        ifFlag: 'applyColdExposure',
        tooltip: `This applies -10% ^${colorCodes.COLD}Cold Resistance ^#to the enemy.`
      },
      {
        var: 'conditionEnemyLightningExposure',
        type: 'check',
        label: `Is the enemy Exposed to ^${colorCodes.LIGHTNING}Lightning^#?`,
        ifFlag: 'applyLightningExposure',
        tooltip: `This applies -10% ^${colorCodes.LIGHTNING}Lightning Resistance ^#to the enemy.`
      },
      {
        var: 'conditionEnemyIntimidated',
        type: 'check',
        label: 'Is the enemy Intimidated?',
        tooltip: 'Intimidated enemies take 10% increased Attack Damage.'
      },
      {
        var: 'conditionEnemyCrushed',
        type: 'check',
        label: 'Is the enemy Crushed?',
        tooltip: 'Crushed enemies have 15% reduced Physical Damage Reduction.'
      },
      {
        var: 'conditionNearLinkedTarget',
        type: 'check',
        label: 'Is the enemy near you Linked target?',
        ifEnemyCond: 'NearLinkedTarget'
      },
      {
        var: 'conditionEnemyUnnerved',
        type: 'check',
        label: 'Is the enemy Unnerved?',
        tooltip: 'Unnerved enemies take 10% increased Spell Damage.'
      },
      {
        var: 'conditionEnemyCoveredInAsh',
        type: 'check',
        label: 'Is the enemy covered in Ash?',
        tooltip: `Covered in Ash applies the following to the enemy:\n\t20% increased ^${colorCodes.FIRE}Fire ^#Damage taken\n\t20% less Movement Speed`
      },
      {
        var: 'conditionEnemyCoveredInFrost',
        type: 'check',
        label: 'Is the enemy covered in Frost?',
        tooltip: `Covered in Frost applies the following to the enemy:\n\t20% increased ^${colorCodes.COLD}Cold ^#Damage taken\n\t50% less Critical Strike Chance`
      },
      { var: 'conditionEnemyOnConsecratedGround', type: 'check', label: 'Is the enemy on Consecrated Ground?' },
      {
        var: 'conditionEnemyOnProfaneGround',
        type: 'check',
        label: 'Is the enemy on Profane Ground?',
        ifFlag: 'Condition:CreateProfaneGround',
        tooltip: 'Enemies on Profane Ground receive the following modifiers:\n\t-10% to all Resistances\n\t+1% chance to be Critically Hit'
      },
      {
        var: 'multiplierEnemyAffectedByGraspingVines',
        type: 'count',
        label: '# of Grasping Vines affecting enemy:',
        ifMult: 'GraspingVinesAffectingEnemy'
      },
      {
        var: 'conditionEnemyOnFungalGround',
        type: 'check',
        label: 'Is the enemy on Fungal Ground?',
        ifCond: 'OnFungalGround',
        tooltip: 'Enemies on your Fungal Ground deal 10% less Damage.'
      },
      {
        var: 'conditionEnemyInChillingArea',
        type: 'check',
        label: `Is the enemy in a ^${colorCodes.COLD}Chilling ^#area?`,
        ifEnemyCond: 'InChillingArea'
      },
      {
        var: 'conditionEnemyInFrostGlobe',
        type: 'check',
        label: 'Is the enemy in the Frost Shield area?',
        ifEnemyCond: 'EnemyInFrostGlobe'
      },
      {
        var: 'enemyConditionHitByFireDamage',
        type: 'check',
        label: `Enemy was Hit by ^${colorCodes.FIRE}Fire ^#Damage?`,
        ifFlag: 'ElementalEquilibrium'
      },
      {
        var: 'enemyConditionHitByColdDamage',
        type: 'check',
        label: `Enemy was Hit by ^${colorCodes.COLD}Cold ^#Damage?`,
        ifFlag: 'ElementalEquilibrium'
      },
      {
        var: 'enemyConditionHitByLightningDamage',
        type: 'check',
        label: `Enemy was Hit by ^${colorCodes.LIGHTNING}Light^#. Damage?`,
        ifFlag: 'ElementalEquilibrium'
      },
      {
        var: 'EEIgnoreHitDamage',
        type: 'check',
        label: 'Ignore Skill Hit Damage?',
        ifFlag: 'ElementalEquilibrium',
        tooltip: 'This option prevents EE from being reset by the hit damage of your main skill.'
      }
    ]
  },
  {
    name: 'Enemy Stats',
    variables: [
      {
        var: 'enemyLevel',
        type: 'count',
        label: 'Enemy Level:',
        tooltip: `This overrides the default enemy level used to estimate your hit and ^${colorCodes.POSITIVE}evade ^#chances.\nThe default level is your character level, capped at 85, which is the same value\nused in-game to calculate the stats on the character sheet.`
      },
      {
        var: 'conditionEnemyRareOrUnique',
        type: 'check',
        label: 'Is the enemy Rare or Unique?',
        ifEnemyCond: 'EnemyRareOrUnique',
        tooltip: 'The enemy will automatically be considered to be Unique if they are a Boss,\nbut you can use this option to force it if necessary.'
      },
      {
        var: 'enemyIsBoss',
        type: 'list',
        label: 'Is the enemy a Boss?',
        tooltip: `
Bosses' damage is monster damage scaled to an average damage of their attacks
This is divided by 4.25 to represent 4 damage types + some ^${colorCodes.CHAOS}chaos
^#Fill in the exact damage numbers if more precision is needed

Standard Boss adds the following modifiers:
	33% less Effect of your Hexes
	+40% to enemy Elemental Resistances
	+25% to enemy ^${colorCodes.CHAOS}Chaos Resistance
	^#94% of monster damage

Guardian / Pinnacle Boss adds the following modifiers:
	66% less Effect of your Hexes
	+50% to enemy Elemental Resistances
	+30% to enemy ^${colorCodes.CHAOS}Chaos Resistance
	^#+33% to enemy Armour
	188% of monster damage
	5% penetration

Uber Pinnacle Boss adds the following modifiers:
	66% less Effect of your Hexes
	+50% to enemy Elemental Resistances
	+30% to enemy ^${colorCodes.CHAOS}Chaos Resistance
	^#+100% to enemy Armour
	70% less to enemy Damage taken
	235% of monster damage
	8% penetration
	`,
        list: [
          { value: 'None', label: 'No' },
          { value: 'Boss', label: 'Standard Boss' },
          {
            value: 'Pinnacle',
            label: 'Guardian/Pinnacle Boss'
          },
          { value: 'Uber', label: 'Uber Pinnacle Boss' }
        ]
      },
      {
        var: 'deliriousPercentage',
        type: 'list',
        label: 'Delirious Effect:',
        list: [
          { value: 0, label: 'None' },
          { value: '20Percent', label: '20% Delirious' },
          {
            value: '40Percent',
            label: '40% Delirious'
          },
          { value: '60Percent', label: '60% Delirious' },
          {
            value: '80Percent',
            label: '80% Delirious'
          },
          { value: '100Percent', label: '100% Delirious' }
        ],
        tooltip:
          "Delirium scales enemy 'less Damage Taken' as well as enemy 'increased Damage dealt'\nAt 100% effect:\nEnemies Deal 30% Increased Damage\nEnemies take 96% Less Damage"
      },
      { var: 'enemyPhysicalReduction', type: 'integer', label: 'Enemy Phys. Damage Reduction:' },
      { var: 'enemyLightningResist', type: 'integer', label: `Enemy ^${colorCodes.LIGHTNING}Lightning Resistance^#:` },
      { var: 'enemyColdResist', type: 'integer', label: `Enemy ^${colorCodes.COLD}Cold Resistance^#:` },
      { var: 'enemyFireResist', type: 'integer', label: `Enemy ^${colorCodes.FIRE}Fire Resistance^#:` },
      { var: 'enemyChaosResist', type: 'integer', label: `Enemy ^${colorCodes.CHAOS}Chaos Resistance^#:` },
      {
        var: 'presetBossSkills',
        type: 'list',
        label: 'Boss Skill Preset',
        tooltip: `
Used to fill in defaults for specific boss skills if the boss config is not set

Bosses' damage is assumed at a 2/3 roll, with no Atlas passives, at the normal monster level for your character level (capped at 84)
Fill in the exact damage numbers if more precision is needed

Caveats for certain skills are below

Shaper Ball: Allocating Cosmic Wounds increases the penetration to 40% and adds 2 projectiles
Shaper Slam: Cannot be Evaded.  Allocating Cosmic Wounds doubles the damage and cannot be blocked or dodged
Maven Memory Game: Is three separate hits, and has a large DoT effect.  Neither is taken into account here.  i.e. Hits before death should be >: 4 to survive`,
        list: [
          { value: 'None', label: 'None' },
          {
            value: 'Uber Atziri Flameblast',
            label: 'Uber Atziri Flameblast'
          },
          { value: 'Shaper Ball', label: 'Shaper Ball' },
          {
            value: 'Shaper Slam',
            label: 'Shaper Slam'
          },
          { value: 'Maven Memory Game', label: 'Maven Memory Game' }
        ]
      },
      {
        var: 'enemyDamageType',
        type: 'list',
        label: 'Enemy Damage Type:',
        tooltip:
          'Controls which types of damage the EHP calculation uses:\n\tAverage: uses the Average of all damage types\n\nIf a specific damage type is selected, that will be the only type used.',
        list: [
          { value: 'Average', label: 'Average' },
          { value: 'Melee', label: 'Melee' },
          {
            value: 'Projectile',
            label: 'Projectile'
          },
          { value: 'Spell', label: 'Spell' },
          { value: 'SpellProjectile', label: 'Projectile Spell' }
        ]
      },
      { var: 'enemySpeed', type: 'integer', label: 'Enemy attack / cast time in ms:', defaultPlaceholderState: 700 },
      { var: 'enemyCritChance', type: 'integer', label: 'Enemy critical strike chance:', defaultPlaceholderState: 5 },
      {
        var: 'enemyCritDamage',
        type: 'integer',
        label: 'Enemy critical strike multipler:',
        defaultPlaceholderState: 30
      },
      {
        var: 'enemyPhysicalDamage',
        type: 'integer',
        label: 'Enemy Skill Physical Damage:',
        tooltip:
          "This overrides the default damage amount used to estimate your damage reduction from armour.\nThe default is 1.5 times the enemy's base damage, which is the same value\nused in-game to calculate the estimate shown on the character sheet."
      },
      {
        var: 'enemyLightningDamage',
        type: 'integer',
        label: `Enemy Skill ^${colorCodes.LIGHTNING}Lightning Damage^#:`
      },
      { var: 'enemyLightningPen', type: 'integer', label: `Enemy Skill ^${colorCodes.LIGHTNING}Lightning Pen^#:` },
      { var: 'enemyColdDamage', type: 'integer', label: `Enemy Skill ^${colorCodes.COLD}Cold Damage^#:` },
      { var: 'enemyColdPen', type: 'integer', label: `Enemy Skill ^${colorCodes.COLD}Cold Pen^#:` },
      { var: 'enemyFireDamage', type: 'integer', label: `Enemy Skill ^${colorCodes.FIRE}Fire Damage^#:` },
      { var: 'enemyFirePen', type: 'integer', label: `Enemy Skill ^${colorCodes.FIRE}Fire Pen^#:` },
      { var: 'enemyChaosDamage', type: 'integer', label: `Enemy Skill ^${colorCodes.CHAOS}Chaos Damage^#:` }
    ]
  }
];

export const reverseConfigOptions = configurations.reduce(
  (out, val) => ({
    ...out,
    ...val.variables.reduce(
      (vars, v) => ({
        ...vars,
        [v.var]: v
      }),
      {}
    )
  }),
  {} as Record<string, AllVarTypes>
);
