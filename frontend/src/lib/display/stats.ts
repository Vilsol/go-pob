import { colorCodes } from './colors';

export interface Stat {
  label: string;
  stat?: string;
  compPercent?: boolean;
  condFunc?: (v: number, o: { [key: string]: number }) => boolean;
  flag?: string;
  fmt?: string;
  color?: string;
  hideStat?: boolean;
  labelStat?: string;
  overCapStat?: string;
  val?: string;
  warnFunc?: (v: number) => string | false;
  lowerIsBetter?: boolean;
  pc?: boolean;
  mod?: boolean;
}

export const displayStats: Stat[][] = [
  [
    { stat: 'ActiveMinionLimit', label: 'Active Minion Limit', fmt: 'd' },
    { stat: 'AverageHit', label: 'Average Hit', fmt: '.1f', compPercent: true },
    { stat: 'AverageDamage', label: 'Average Damage', fmt: '.1f', compPercent: true, flag: 'attack' },
    {
      stat: 'Speed',
      label: 'Attack Rate',
      fmt: '.2f',
      compPercent: true,
      flag: 'attack',
      condFunc: (v, o) => v > 0 && (o.TriggerTime || 0) == 0
    },
    {
      stat: 'Speed',
      label: 'Cast Rate',
      fmt: '.2f',
      compPercent: true,
      flag: 'spell',
      condFunc: (v, o) => v > 0 && (o.TriggerTime || 0) == 0
    },
    {
      stat: 'ServerTriggerRate',
      label: 'Trigger Rate',
      fmt: '.2f',
      compPercent: true,
      condFunc: (v, o) => (o.TriggerTime || 0) != 0
    },
    {
      stat: 'Speed',
      label: 'Effective Trigger Rate',
      fmt: '.2f',
      compPercent: true,
      condFunc: (v, o) => (o.TriggerTime || 0) != 0 && o.ServerTriggerRate != o.Speed
    },
    { stat: 'WarcryCastTime', label: 'Cast Time', fmt: '.2fs', compPercent: true, lowerIsBetter: true, flag: 'warcry' },
    { stat: 'HitSpeed', label: 'Hit Rate', fmt: '.2f', compPercent: true, condFunc: (v, o) => !o.TriggerTime },
    { stat: 'TrapThrowingTime', label: 'Trap Throwing Time', fmt: '.2fs', compPercent: true, lowerIsBetter: true },
    { stat: 'TrapCooldown', label: 'Trap Cooldown', fmt: '.3fs', lowerIsBetter: true },
    { stat: 'MineLayingTime', label: 'Mine Throwing Time', fmt: '.2fs', compPercent: true, lowerIsBetter: true },
    { stat: 'TotemPlacementTime', label: 'Totem Placement Time', fmt: '.2fs', compPercent: true, lowerIsBetter: true },
    { stat: 'PreEffectiveCritChance', label: 'Crit Chance', fmt: '.2f%%' },
    {
      stat: 'CritChance',
      label: 'Effective Crit Chance',
      fmt: '.2f%%',
      condFunc: (v, o) => v != o.PreEffectiveCritChance
    },
    {
      stat: 'CritMultiplier',
      label: 'Crit Multiplier',
      fmt: 'd%%',
      pc: true,
      condFunc: (v, o) => (o.CritChance || 0) > 0
    },
    { stat: 'HitChance', label: 'Hit Chance', fmt: '.0f%%', flag: 'attack' },
    { stat: 'TotalDPS', label: 'Total DPS', fmt: '.1f', compPercent: true, flag: 'notAverage' },
    {
      stat: 'TotalDPS',
      label: 'Total DPS',
      fmt: '.1f',
      compPercent: true,
      flag: 'showAverage',
      condFunc: (v, o) => (o.TriggerTime || 0) != 0
    },
    { stat: 'TotalDot', label: 'DoT DPS', fmt: '.1f', compPercent: true },
    {
      stat: 'WithDotDPS',
      label: 'Total DPS inc. DoT',
      fmt: '.1f',
      compPercent: true,
      flag: 'notAverage',
      condFunc: (v, o) =>
        v != o.TotalDPS &&
        (o.PoisonDPS || 0) == 0 &&
        (o.IgniteDPS || 0) == 0 &&
        (o.ImpaleDPS || 0) == 0 &&
        (o.BleedDPS || 0) == 0
    },
    { stat: 'BleedDPS', label: 'Bleed DPS', fmt: '.1f', compPercent: true },
    { stat: 'BleedDamage', label: 'Total Damage per Bleed', fmt: '.1f', compPercent: true, flag: 'showAverage' },
    {
      stat: 'WithBleedDPS',
      label: 'Total DPS inc. Bleed',
      fmt: '.1f',
      compPercent: true,
      flag: 'notAverage',
      condFunc: (v, o) =>
        v != o.TotalDPS &&
        (o.TotalDot || 0) == 0 &&
        (o.PoisonDPS || 0) == 0 &&
        (o.ImpaleDPS || 0) == 0 &&
        (o.IgniteDPS || 0) == 0
    },
    { stat: 'IgniteDPS', label: 'Ignite DPS', fmt: '.1f', compPercent: true },
    { stat: 'IgniteDamage', label: 'Total Damage per Ignite', fmt: '.1f', compPercent: true, flag: 'showAverage' },
    {
      stat: 'WithIgniteDPS',
      label: 'Total DPS inc. Ignite',
      fmt: '.1f',
      compPercent: true,
      flag: 'notAverage',
      condFunc: (v, o) =>
        v != o.TotalDPS &&
        (o.TotalDot || 0) == 0 &&
        (o.PoisonDPS || 0) == 0 &&
        (o.ImpaleDPS || 0) == 0 &&
        (o.BleedDPS || 0) == 0
    },
    { stat: 'WithIgniteAverageDamage', label: 'Average Dmg. inc. Ignite', fmt: '.1f', compPercent: true },
    { stat: 'PoisonDPS', label: 'Poison DPS', fmt: '.1f', compPercent: true },
    { stat: 'PoisonDamage', label: 'Total Damage per Poison', fmt: '.1f', compPercent: true },
    {
      stat: 'WithPoisonDPS',
      label: 'Total DPS inc. Poison',
      fmt: '.1f',
      compPercent: true,
      flag: 'notAverage',
      condFunc: (v, o) =>
        v != o.TotalDPS &&
        (o.TotalDot || 0) == 0 &&
        (o.IgniteDPS || 0) == 0 &&
        (o.ImpaleDPS || 0) == 0 &&
        (o.BleedDPS || 0) == 0
    },
    { stat: 'DecayDPS', label: 'Decay DPS', fmt: '.1f', compPercent: true },
    {
      stat: 'TotalDotDPS',
      label: 'Total DoT DPS',
      fmt: '.1f',
      compPercent: true,
      condFunc: (v, o) =>
        v != o.TotalDot &&
        v != o.ImpaleDPS &&
        v != o.TotalPoisonDPS &&
        v != (o.TotalIgniteDPS || o.IgniteDPS) &&
        v != o.BleedDPS
    },
    { stat: 'ImpaleDPS', label: 'Impale Damage', fmt: '.1f', compPercent: true, flag: 'showAverage' },
    {
      stat: 'WithImpaleDPS',
      label: 'Damage inc. Impale',
      fmt: '.1f',
      compPercent: true,
      flag: 'showAverage',
      condFunc: (v, o) =>
        v != o.TotalDPS &&
        (o.TotalDot || 0) == 0 &&
        (o.IgniteDPS || 0) == 0 &&
        (o.PoisonDPS || 0) == 0 &&
        (o.BleedDPS || 0) == 0
    },
    { stat: 'ImpaleDPS', label: 'Impale DPS', fmt: '.1f', compPercent: true, flag: 'notAverage' },
    {
      stat: 'WithImpaleDPS',
      label: 'Total DPS inc. Impale',
      fmt: '.1f',
      compPercent: true,
      flag: 'notAverage',
      condFunc: (v, o) =>
        v != o.TotalDPS &&
        (o.TotalDot || 0) == 0 &&
        (o.IgniteDPS || 0) == 0 &&
        (o.PoisonDPS || 0) == 0 &&
        (o.BleedDPS || 0) == 0
    },
    { stat: 'MirageDPS', label: 'Total Mirage DPS', fmt: '.1f', compPercent: true, condFunc: (v) => v > 0 },
    {
      stat: 'CullingDPS',
      label: 'Culling DPS',
      fmt: '.1f',
      compPercent: true,
      condFunc: (v, o) => (o.CullingDPS || 0) > 0
    },
    {
      stat: 'CombinedDPS',
      label: 'Combined DPS',
      fmt: '.1f',
      compPercent: true,
      flag: 'notAverage',
      condFunc: (v, o) =>
        v != (o.TotalDPS || 0) + (o.TotalDot || 0) &&
        v != o.WithImpaleDPS &&
        v != o.WithPoisonDPS &&
        v != o.WithIgniteDPS &&
        v != o.WithBleedDPS
    },
    {
      stat: 'CombinedAvg',
      label: 'Combined Total Damage',
      fmt: '.1f',
      compPercent: true,
      flag: 'showAverage',
      condFunc: (v, o) =>
        v != o.AverageDamage &&
        (o.TotalDot || 0) == 0 &&
        (v != o.WithPoisonDPS || v != o.WithIgniteDPS || v != o.WithBleedDPS)
    },
    { stat: 'Cooldown', label: 'Skill Cooldown', fmt: '.3fs', lowerIsBetter: true },
    { stat: 'SealCooldown', label: 'Seal Gain Frequency', fmt: '.2fs', lowerIsBetter: true },
    { stat: 'SealMax', label: 'Max Number of Seals', fmt: 'd' },
    { stat: 'TimeMaxSeals', label: 'Time to Gain Max Seals', fmt: '.2fs', lowerIsBetter: true },
    { stat: 'AreaOfEffectRadius', label: 'AoE Radius', fmt: 'd' },
    { stat: 'BrandAttachmentRange', label: 'Attachment Range', fmt: 'd', flag: 'brand' },
    { stat: 'BrandTicks', label: 'Activations per Brand', fmt: 'd', flag: 'brand' },
    {
      stat: 'ManaCost',
      label: 'Mana Cost',
      fmt: 'd',
      color: colorCodes.MANA,
      compPercent: true,
      lowerIsBetter: true,
      condFunc: (v) => v > 0
    },
    {
      stat: 'LifeCost',
      label: 'Life Cost',
      fmt: 'd',
      color: colorCodes.LIFE,
      compPercent: true,
      lowerIsBetter: true,
      condFunc: (v) => v > 0
    },
    {
      stat: 'ESCost',
      label: 'Energy Shield Cost',
      fmt: 'd',
      color: colorCodes.ES,
      compPercent: true,
      lowerIsBetter: true,
      condFunc: (v) => v > 0
    },
    {
      stat: 'RageCost',
      label: 'Rage Cost',
      fmt: 'd',
      color: colorCodes.RAGE,
      compPercent: true,
      lowerIsBetter: true,
      condFunc: (v) => v > 0
    },
    {
      stat: 'ManaPercentCost',
      label: 'Mana Cost',
      fmt: 'd%%',
      color: colorCodes.MANA,
      compPercent: true,
      lowerIsBetter: true,
      condFunc: (v) => v > 0
    },
    {
      stat: 'LifePercentCost',
      label: 'Life Cost',
      fmt: 'd%%',
      color: colorCodes.LIFE,
      compPercent: true,
      lowerIsBetter: true,
      condFunc: (v) => v > 0
    },
    {
      stat: 'ManaPerSecondCost',
      label: 'Mana Cost',
      fmt: '.2f/s',
      color: colorCodes.MANA,
      compPercent: true,
      lowerIsBetter: true,
      condFunc: (v) => v > 0
    },
    {
      stat: 'LifePerSecondCost',
      label: 'Life Cost',
      fmt: '.2f/s',
      color: colorCodes.LIFE,
      compPercent: true,
      lowerIsBetter: true,
      condFunc: (v) => v > 0
    },
    {
      stat: 'ManaPercentPerSecondCost',
      label: 'Mana Cost',
      fmt: '.2f%%/s',
      color: colorCodes.MANA,
      compPercent: true,
      lowerIsBetter: true,
      condFunc: (v) => v > 0
    },
    {
      stat: 'LifePercentPerSecondCost',
      label: 'Life Cost',
      fmt: '.2f%%/s',
      color: colorCodes.LIFE,
      compPercent: true,
      lowerIsBetter: true,
      condFunc: (v) => v > 0
    },
    {
      stat: 'ESPerSecondCost',
      label: 'Energy Shield Cost',
      fmt: '.2f/s',
      color: colorCodes.ES,
      compPercent: true,
      lowerIsBetter: true,
      condFunc: (v) => v > 0
    },
    {
      stat: 'ESPercentPerSecondCost',
      label: 'Energy Shield Cost',
      fmt: '.2f%%/s',
      color: colorCodes.ES,
      compPercent: true,
      lowerIsBetter: true,
      condFunc: (v) => v > 0
    }
  ],
  [
    { stat: 'Str', label: 'Strength', color: colorCodes.STRENGTH, fmt: 'd' },
    {
      stat: 'ReqStr',
      label: 'Strength Required',
      color: colorCodes.STRENGTH,
      fmt: 'd',
      lowerIsBetter: true,
      condFunc: (v, o) => v > o.Str,
      warnFunc: () => 'You do not meet the Strength requirement'
    },
    { stat: 'Dex', label: 'Dexterity', color: colorCodes.DEXTERITY, fmt: 'd' },
    {
      stat: 'ReqDex',
      label: 'Dexterity Required',
      color: colorCodes.DEXTERITY,
      fmt: 'd',
      lowerIsBetter: true,
      condFunc: (v, o) => v > o.Dex,
      warnFunc: () => 'You do not meet the Dexterity requirement'
    },
    { stat: 'Int', label: 'Intelligence', color: colorCodes.INTELLIGENCE, fmt: 'd' },
    {
      stat: 'ReqInt',
      label: 'Intelligence Required',
      color: colorCodes.INTELLIGENCE,
      fmt: 'd',
      lowerIsBetter: true,
      condFunc: (v, o) => v > o.Int,
      warnFunc: () => 'You do not meet the Intelligence requirement'
    },
    { stat: 'Omni', label: 'Omniscience', color: colorCodes.RARE, fmt: 'd' },
    {
      stat: 'ReqOmni',
      label: 'Omniscience Required',
      color: colorCodes.RARE,
      fmt: 'd',
      lowerIsBetter: true,
      condFunc: (v, o) => v > (o.Omni || 0),
      warnFunc: () => 'You do not meet the Omniscience requirement'
    }
  ],
  [{ stat: 'Devotion', label: 'Devotion', color: colorCodes.RARE, fmt: 'd' }],
  [
    { stat: 'TotalEHP', label: 'Effective Hit Pool', fmt: '.0f', compPercent: true },
    { stat: 'PhysicalMaximumHitTaken', label: 'Phys Max Hit', fmt: '.0f', color: colorCodes.PHYS, compPercent: true },
    {
      stat: 'LightningMaximumHitTaken',
      label: 'Elemental Max Hit',
      fmt: '.0f',
      color: colorCodes.LIGHTNING,
      compPercent: true,
      condFunc: (v, o) =>
        o.LightningMaximumHitTaken == o.ColdMaximumHitTaken && o.LightningMaximumHitTaken == o.FireMaximumHitTaken
    },
    {
      stat: 'FireMaximumHitTaken',
      label: 'Fire Max Hit',
      fmt: '.0f',
      color: colorCodes.FIRE,
      compPercent: true,
      condFunc: (v, o) =>
        o.LightningMaximumHitTaken != o.ColdMaximumHitTaken || o.LightningMaximumHitTaken != o.FireMaximumHitTaken
    },
    {
      stat: 'ColdMaximumHitTaken',
      label: 'Cold Max Hit',
      fmt: '.0f',
      color: colorCodes.COLD,
      compPercent: true,
      condFunc: (v, o) =>
        o.LightningMaximumHitTaken != o.ColdMaximumHitTaken || o.LightningMaximumHitTaken != o.FireMaximumHitTaken
    },
    {
      stat: 'LightningMaximumHitTaken',
      label: 'Lightning Max Hit',
      fmt: '.0f',
      color: colorCodes.LIGHTNING,
      compPercent: true,
      condFunc: (v, o) =>
        o.LightningMaximumHitTaken != o.ColdMaximumHitTaken || o.LightningMaximumHitTaken != o.FireMaximumHitTaken
    },
    { stat: 'ChaosMaximumHitTaken', label: 'Chaos Max Hit', fmt: '.0f', color: colorCodes.CHAOS, compPercent: true }
  ],
  [
    { stat: 'Life', label: 'Total Life', fmt: 'd', color: colorCodes.LIFE, compPercent: true },
    {
      stat: 'Spec:LifeInc',
      label: '%Inc Life from Tree',
      fmt: 'd%%',
      color: colorCodes.LIFE,
      condFunc: (v, o) => v > 0 && o.Life > 1
    },
    {
      stat: 'LifeUnreserved',
      label: 'Unreserved Life',
      fmt: 'd',
      color: colorCodes.LIFE,
      condFunc: (v, o) => v < o.Life,
      compPercent: true,
      warnFunc: (v) => v < 0 && 'Your unreserved Life is negative'
    },
    {
      stat: 'LifeRecoverable',
      label: 'Life Recoverable',
      fmt: 'd',
      color: colorCodes.LIFE,
      condFunc: (v, o) => v < o.LifeUnreserved
    },
    {
      stat: 'LifeUnreservedPercent',
      label: 'Unreserved Life',
      fmt: 'd%%',
      color: colorCodes.LIFE,
      condFunc: (v) => v < 100
    },
    { stat: 'LifeRegen', label: 'Life Regen', fmt: '.1f', color: colorCodes.LIFE },
    {
      stat: 'LifeLeechGainRate',
      label: 'Life Leech/On Hit Rate',
      fmt: '.1f',
      color: colorCodes.LIFE,
      compPercent: true
    },
    {
      stat: 'LifeLeechGainPerHit',
      label: 'Life Leech/Gain per Hit',
      fmt: '.1f',
      color: colorCodes.LIFE,
      compPercent: true
    }
  ],
  [
    { stat: 'Mana', label: 'Total Mana', fmt: 'd', color: colorCodes.MANA, compPercent: true },
    { stat: 'Spec:ManaInc', label: '%Inc Mana from Tree', color: colorCodes.MANA, fmt: 'd%%' },
    {
      stat: 'ManaUnreserved',
      label: 'Unreserved Mana',
      fmt: 'd',
      color: colorCodes.MANA,
      condFunc: (v, o) => v < o.Mana,
      compPercent: true,
      warnFunc: (v) => v < 0 && 'Your unreserved Mana is negative'
    },
    {
      stat: 'ManaUnreservedPercent',
      label: 'Unreserved Mana',
      fmt: 'd%%',
      color: colorCodes.MANA,
      condFunc: (v) => v < 100
    },
    { stat: 'ManaRegen', label: 'Mana Regen', fmt: '.1f', color: colorCodes.MANA },
    {
      stat: 'ManaLeechGainRate',
      label: 'Mana Leech/On Hit Rate',
      fmt: '.1f',
      color: colorCodes.MANA,
      compPercent: true
    },
    {
      stat: 'ManaLeechGainPerHit',
      label: 'Mana Leech/Gain per Hit',
      fmt: '.1f',
      color: colorCodes.MANA,
      compPercent: true
    }
  ],
  [
    { stat: 'TotalDegen', label: 'Total Degen', fmt: '.1f', lowerIsBetter: true },
    { stat: 'TotalNetRegen', label: 'Total Net Regen', fmt: '+.1f' },
    { stat: 'NetLifeRegen', label: 'Net Life Regen', fmt: '+.1f', color: colorCodes.LIFE },
    { stat: 'NetManaRegen', label: 'Net Mana Regen', fmt: '+.1f', color: colorCodes.MANA },
    { stat: 'NetEnergyShieldRegen', label: 'Net Energy Shield Regen', fmt: '+.1f', color: colorCodes.ES }
  ],
  [
    { stat: 'Ward', label: 'Ward', fmt: 'd', color: colorCodes.WARD, compPercent: true },
    { stat: 'EnergyShield', label: 'Energy Shield', fmt: 'd', color: colorCodes.ES, compPercent: true },
    {
      stat: 'EnergyShieldRecoveryCap',
      label: 'Recoverable ES',
      color: colorCodes.ES,
      fmt: 'd',
      condFunc: (v) => !!v
    },
    { stat: 'Spec:EnergyShieldInc', label: '%Inc ES from Tree', color: colorCodes.ES, fmt: 'd%%' },

    { stat: 'EnergyShieldRegen', label: 'Energy Shield Regen', color: colorCodes.ES, fmt: '.1f' },
    {
      stat: 'EnergyShieldLeechGainRate',
      label: 'ES Leech/On Hit Rate',
      color: colorCodes.ES,
      fmt: '.1f',
      compPercent: true
    },
    {
      stat: 'EnergyShieldLeechGainPerHit',
      label: 'ES Leech/Gain per Hit',
      color: colorCodes.ES,
      fmt: '.1f',
      compPercent: true
    }
  ],
  [
    { stat: 'Evasion', label: 'Evasion rating', fmt: 'd', color: colorCodes.EVASION, compPercent: true },
    { stat: 'Spec:EvasionInc', label: '%Inc Evasion from Tree', color: colorCodes.EVASION, fmt: 'd%%' },
    {
      stat: 'MeleeEvadeChance',
      label: 'Evade Chance',
      fmt: 'd%%',
      color: colorCodes.EVASION,
      condFunc: (v, o) => v > 0 && o.MeleeEvadeChance == o.ProjectileEvadeChance
    },
    {
      stat: 'MeleeEvadeChance',
      label: 'Melee Evade Chance',
      fmt: 'd%%',
      color: colorCodes.EVASION,
      condFunc: (v, o) => v > 0 && o.MeleeEvadeChance != o.ProjectileEvadeChance
    },
    {
      stat: 'ProjectileEvadeChance',
      label: 'Projectile Evade Chance',
      fmt: 'd%%',
      color: colorCodes.EVASION,
      condFunc: (v, o) => v > 0 && o.MeleeEvadeChance != o.ProjectileEvadeChance
    }
  ],
  [
    { stat: 'Armour', label: 'Armour', fmt: 'd', compPercent: true },
    { stat: 'Spec:ArmourInc', label: '%Inc Armour from Tree', fmt: 'd%%' },
    { stat: 'PhysicalDamageReduction', label: 'Phys. Damage Reduction', fmt: 'd%%' }
  ],
  [
    { stat: 'BlockChance', label: 'Block Chance', fmt: 'd%%', overCapStat: 'BlockChanceOverCap' },
    { stat: 'SpellBlockChance', label: 'Spell Block Chance', fmt: 'd%%', overCapStat: 'SpellBlockChanceOverCap' },
    { stat: 'AttackDodgeChance', label: 'Attack Dodge Chance', fmt: 'd%%', overCapStat: 'AttackDodgeChanceOverCap' },
    { stat: 'SpellDodgeChance', label: 'Spell Dodge Chance', fmt: 'd%%', overCapStat: 'SpellDodgeChanceOverCap' },
    {
      stat: 'SpellSuppressionChance',
      label: 'Spell Suppression Chance',
      fmt: 'd%%',
      overCapStat: 'SpellSuppressionChanceOverCap'
    }
  ],
  [
    {
      stat: 'FireResist',
      label: 'Fire Resistance',
      fmt: 'd%%',
      color: colorCodes.FIRE,
      overCapStat: 'FireResistOverCap'
    },
    { stat: 'FireResistOverCap', label: 'Fire Res. Over Max', fmt: 'd%%', hideStat: true },
    {
      stat: 'ColdResist',
      label: 'Cold Resistance',
      fmt: 'd%%',
      color: colorCodes.COLD,
      overCapStat: 'ColdResistOverCap'
    },
    { stat: 'ColdResistOverCap', label: 'Cold Res. Over Max', fmt: 'd%%', hideStat: true },
    {
      stat: 'LightningResist',
      label: 'Lightning Resistance',
      fmt: 'd%%',
      color: colorCodes.LIGHTNING,
      overCapStat: 'LightningResistOverCap'
    },
    { stat: 'LightningResistOverCap', label: 'Lightning Res. Over Max', fmt: 'd%%', hideStat: true },
    {
      stat: 'ChaosResist',
      label: 'Chaos Resistance',
      fmt: 'd%%',
      color: colorCodes.CHAOS,
      condFunc: (v, o) => !o.ChaosInoculation,
      overCapStat: 'ChaosResistOverCap'
    },
    { stat: 'ChaosResistOverCap', label: 'Chaos Res. Over Max', fmt: 'd%%', hideStat: true },
    {
      label: 'Chaos Resistance',
      val: 'Immune',
      labelStat: 'ChaosResist',
      color: colorCodes.CHAOS,
      condFunc: (v, o) => o.ChaosInoculation > 0
    }
  ],
  [{ stat: 'EffectiveMovementSpeedMod', label: 'Movement Speed Modifier', fmt: '+d%%', mod: true }],
  [{ stat: 'FullDPS', label: 'Full DPS', fmt: '.1f', color: colorCodes.CURRENCY, compPercent: true }],
  [{ stat: 'SkillDPS', label: 'Skill DPS' }]
];
