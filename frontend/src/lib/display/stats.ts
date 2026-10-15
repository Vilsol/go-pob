import { colorCodes } from './colors';

export type StatOutput = Record<string, number | undefined>;

export interface Stat {
  stat?: string;
  childStat?: string;
  label?: string;
  fmt?: string;
  flag?: string;
  notFlag?: string;
  color?: string;
  overCapStat?: string;
  suffix?: string | ((v: number, o: StatOutput) => string | false | undefined);
  pc?: boolean;
  mod?: boolean;
  hideStat?: boolean;
  warnColor?: boolean;
  chaosInoc?: boolean;
  compactValue?: boolean;
  compPercent?: boolean;
  lowerIsBetter?: boolean;
  pool?: string;
  breakdown?: string;
  modNames?: string[];
  ignoredSections?: string[];
  condFunc?: (v: number, o: StatOutput) => boolean;
  warnFunc?: (v: number, o: StatOutput, strings?: Record<string, string | undefined>) => string | false;
  suffixCondFunc?: (v: number, o: StatOutput) => boolean;
  overCapStatCondFunc?: (v: number, o: StatOutput) => boolean;
}

export const DOT_DPS_CAP = 35791394;

const n = (v: number | undefined) => v ?? 0;
const truthy = (v: number | undefined) => v !== undefined && v !== 0;

// src/Modules/BuildDisplayStats.lua:12-14
const maxHitSuffix = (value: number, output: StatOutput) => (value === Infinity ? 'Immune' : truthy(output.GuardSkillActive) && 'Guard');

const reqWarn = (attr: string, key: string) => (_v: number, _o: StatOutput, strings?: Record<string, string | undefined>) =>
  `You do not meet the ${attr} requirement of ${strings?.[key] ?? ''}`;

// src/Modules/BuildDisplayStats.lua:16
const defenseIgnoredSections = ['Base from Gear', 'Inc. from Tree'];

// src/Modules/BuildDisplayStats.lua:50-248
export const displayStats: Stat[] = [
  { stat: 'ActiveMinionLimit', label: 'Active Minion Limit', fmt: 'd' },
  { stat: 'AverageHit', label: 'Average Hit', fmt: '.1f', compactValue: true, compPercent: true },
  { stat: 'PvpAverageHit', label: 'PvP Average Hit', fmt: '.1f', compactValue: true, compPercent: true, flag: 'isPvP' },
  { stat: 'AverageDamage', label: 'Average Damage', fmt: '.1f', compactValue: true, compPercent: true, flag: 'attack' },
  {
    stat: 'AverageDamage',
    label: 'Average Damage',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    flag: 'monsterExplode',
    condFunc: (v, o) => o.HitChance !== 100
  },
  {
    stat: 'AverageBurstDamage',
    label: 'Average Burst Damage',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    condFunc: (v, o) => truthy(o.AverageBurstHits) && n(o.AverageBurstHits) > 1 && v > 0
  },
  { stat: 'PvpAverageDamage', label: 'PvP Average Damage', fmt: '.1f', compactValue: true, compPercent: true, flag: 'attackPvP' },
  {
    stat: 'Speed',
    label: 'Attack Rate',
    fmt: '.2f',
    compPercent: true,
    flag: 'attack',
    condFunc: (v, o) => v > 0 && n(o.TriggerTime) === 0,
    breakdown: 'Speed',
    modNames: ['Speed'],
    ignoredSections: ['Inc. Cast Speed', 'More Cast Speed']
  },
  { stat: 'KineticFusilladeMaxEffectiveAPS', label: 'Max Eff. KF Attack Rate', fmt: '.2f', compPercent: true, flag: 'attack', condFunc: (v) => v > 0 },
  { stat: 'Speed', label: 'Cast Rate', fmt: '.2f', compPercent: true, flag: 'spell', condFunc: (v, o) => v > 0 && n(o.TriggerTime) === 0, modNames: ['Speed'] },
  { stat: 'Speed', label: 'Effective Trigger Rate', fmt: '.2f', compPercent: true, notFlag: 'skipEffectiveRate', condFunc: (v, o) => n(o.TriggerTime) !== 0 },
  { stat: 'WarcryCastTime', label: 'Cast Time', fmt: '.2fs', compPercent: true, lowerIsBetter: true, flag: 'warcry' },
  { stat: 'HitSpeed', label: 'Hit Rate', fmt: '.2f', compPercent: true, condFunc: (v, o) => o.TriggerTime === undefined },
  {
    stat: 'HitTime',
    label: 'Channel Time',
    fmt: '.2fs',
    compPercent: true,
    flag: 'channelRelease',
    lowerIsBetter: true,
    condFunc: (v, o) => o.TriggerTime === undefined
  },
  { stat: 'ChannelTimeToTrigger', label: 'Channel Time', fmt: '.2fs', compPercent: true, lowerIsBetter: true },
  { stat: 'TrapThrowingTime', label: 'Trap Throwing Time', fmt: '.2fs', compPercent: true, lowerIsBetter: true },
  { stat: 'TrapCooldown', label: 'Trap Cooldown', fmt: '.3fs', lowerIsBetter: true },
  { stat: 'MineLayingTime', label: 'Mine Throwing Time', fmt: '.2fs', compPercent: true, lowerIsBetter: true },
  { stat: 'TrapThrowCount', label: 'Avg. Traps per Throw', fmt: '.2f' },
  { stat: 'MineThrowCount', label: 'Avg. Mines per Throw', fmt: '.2f' },
  {
    stat: 'TotemPlacementTime',
    label: 'Totem Placement Time',
    fmt: '.2fs',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => o.TriggerTime === undefined
  },
  { stat: 'PreEffectiveCritChance', label: 'Crit Chance', fmt: '.2f%%', flag: 'hit', breakdown: 'PreEffectiveCritChance' },
  {
    stat: 'CritChance',
    label: 'Effective Crit Chance',
    fmt: '.2f%%',
    flag: 'hit',
    condFunc: (v, o) => v !== o.PreEffectiveCritChance,
    breakdown: 'CritChance'
  },
  { stat: 'CritMultiplier', label: 'Crit Multiplier', fmt: 'd%%', pc: true, condFunc: (v, o) => n(o.CritChance) > 0, breakdown: 'CritMultiplier' },
  { stat: 'HitChance', label: 'Hit Chance', fmt: '.0f%%', flag: 'attack', breakdown: 'HitChance' },
  { stat: 'HitChance', label: 'Hit Chance', fmt: '.0f%%', condFunc: (v, o) => truthy(o.enemyHasSpellBlock) },
  { stat: 'TotalDPS', label: 'Hit DPS', fmt: '.1f', compactValue: true, compPercent: true, flag: 'notAverage' },
  { stat: 'PvpTotalDPS', label: 'PvP Hit DPS', fmt: '.1f', compactValue: true, compPercent: true, flag: 'notAveragePvP' },
  { stat: 'TotalDPS', label: 'Hit DPS', fmt: '.1f', compactValue: true, compPercent: true, flag: 'showAverage', condFunc: (v, o) => n(o.TriggerTime) !== 0 },
  { stat: 'TotalDot', label: 'DoT DPS', fmt: '.1f', compactValue: true, compPercent: true },
  {
    stat: 'WithDotDPS',
    label: 'Total DPS inc. DoT',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    flag: 'notAverage',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.PoisonDPS) === 0 && n(o.IgniteDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.BleedDPS) === 0
  },
  {
    stat: 'BleedDPS',
    label: 'Bleed DPS',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    warnFunc: (v) => v >= DOT_DPS_CAP && 'Bleed DPS exceeds in game limit'
  },
  {
    stat: 'CorruptingBloodDPS',
    label: 'Corrupting Blood DPS',
    compactValue: true,
    fmt: '.1f',
    compPercent: true,
    warnFunc: (v) => v >= DOT_DPS_CAP && 'Corrupting Blood DPS exceeds in game limit'
  },
  { stat: 'BleedDamage', label: 'Total Damage per Bleed', fmt: '.1f', compactValue: true, compPercent: true, flag: 'showAverage' },
  {
    stat: 'WithBleedDPS',
    label: 'Total DPS inc. Bleed',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    flag: 'notAverage',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.PoisonDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.IgniteDPS) === 0
  },
  {
    stat: 'IgniteDPS',
    label: 'Ignite DPS',
    fmt: '.1f',
    compPercent: true,
    compactValue: true,
    warnFunc: (v) => v >= DOT_DPS_CAP && 'Ignite DPS exceeds in game limit'
  },
  { stat: 'IgniteDamage', label: 'Total Damage per Ignite', fmt: '.1f', compactValue: true, compPercent: true, flag: 'showAverage' },
  {
    stat: 'BurningGroundDPS',
    label: 'Burning Ground DPS',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    warnFunc: (v) => v >= DOT_DPS_CAP && 'Burning Ground DPS exceeds in game limit'
  },
  {
    stat: 'MirageBurningGroundDPS',
    label: 'Mirage Burning Ground DPS',
    compactValue: true,
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v !== o.BurningGroundDPS,
    warnFunc: (v) => v >= DOT_DPS_CAP && 'Mirage Burning Ground DPS exceeds in game limit'
  },
  {
    stat: 'WithIgniteDPS',
    label: 'Total DPS inc. Ignite',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    flag: 'notAverage',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.PoisonDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'WithIgniteAverageDamage', label: 'Average Dmg. inc. Ignite', compactValue: true, fmt: '.1f', compPercent: true },
  {
    stat: 'PoisonDPS',
    label: 'Single Poison DPS',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    warnFunc: (v) => v >= DOT_DPS_CAP && 'Poison DPS exceeds in game limit'
  },
  {
    stat: 'CausticGroundDPS',
    label: 'Caustic Ground DPS',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    warnFunc: (v) => v >= DOT_DPS_CAP && 'Caustic Ground DPS exceeds in game limit'
  },
  {
    stat: 'MirageCausticGroundDPS',
    label: 'Mirage Caustic Ground DPS',
    compactValue: true,
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v !== o.CausticGroundDPS,
    warnFunc: (v) => v >= DOT_DPS_CAP && 'Mirage Caustic Ground DPS exceeds in game limit'
  },
  { stat: 'PoisonDamage', label: 'Total Damage per Poison', fmt: '.1f', compactValue: true, compPercent: true },
  {
    stat: 'WithPoisonDPS',
    label: 'Total DPS inc. Poison',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    flag: 'notAverage',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.IgniteDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'DecayDPS', label: 'Decay DPS', fmt: '.1f', compactValue: true, compPercent: true },
  {
    stat: 'TotalDotDPS',
    label: 'Total DoT DPS',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    condFunc: (v, o) =>
      truthy(o.showTotalDotDPS) ||
      (v !== o.TotalDot &&
        v !== o.TotalPoisonDPS &&
        v !== o.CausticGroundDPS &&
        v !== (o.TotalIgniteDPS ?? o.IgniteDPS) &&
        v !== o.BurningGroundDPS &&
        v !== o.BleedDPS &&
        v !== o.CorruptingBloodDPS &&
        v !== o.MirageCausticGroundDPS &&
        v !== o.MirageBurningGroundDPS),
    warnFunc: (v) => v >= DOT_DPS_CAP && 'DoT DPS exceeds in game limit'
  },
  { stat: 'ImpaleDPS', label: 'Impale Damage', fmt: '.1f', compactValue: true, compPercent: true, flag: 'showAverage' },
  {
    stat: 'WithImpaleDPS',
    label: 'Damage inc. Impale',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    flag: 'showAverage',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.IgniteDPS) === 0 && n(o.PoisonDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'ImpaleDPS', label: 'Impale DPS', fmt: '.1f', compactValue: true, compPercent: true, flag: 'notAverage' },
  {
    stat: 'WithImpaleDPS',
    label: 'Total DPS inc. Impale',
    compactValue: true,
    fmt: '.1f',
    compPercent: true,
    flag: 'notAverage',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.IgniteDPS) === 0 && n(o.PoisonDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'MirageDPS', label: 'Total Mirage DPS', fmt: '.1f', compactValue: true, compPercent: true, flag: 'mirageArcher', condFunc: (v) => v > 0 },
  { stat: 'MirageDPS', label: 'Total Wisp DPS', fmt: '.1f', compactValue: true, compPercent: true, flag: 'wisp', condFunc: (v) => v > 0 },
  { stat: 'CullingDPS', label: 'Culling DPS', fmt: '.1f', compactValue: true, compPercent: true, condFunc: (v, o) => n(o.CullingDPS) > 0 },
  { stat: 'ReservationDPS', label: 'Reservation DPS', fmt: '.1f', compactValue: true, compPercent: true, condFunc: (v, o) => n(o.ReservationDPS) > 0 },
  {
    stat: 'CombinedDPS',
    label: 'Combined DPS',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    flag: 'notAverage',
    condFunc: (v, o) =>
      v !== n(o.TotalDPS) + n(o.TotalDot) &&
      v !== o.WithImpaleDPS &&
      (truthy(o.showTotalDotDPS) || (v !== o.WithPoisonDPS && v !== o.WithIgniteDPS && v !== o.WithBleedDPS))
  },
  {
    stat: 'CombinedAvg',
    label: 'Combined Total Damage',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    flag: 'showAverage',
    condFunc: (v, o) => v !== o.AverageDamage && n(o.TotalDot) === 0 && (v !== o.WithPoisonDPS || v !== o.WithIgniteDPS || v !== o.WithBleedDPS)
  },
  { stat: 'ExplodeChance', label: 'Total Explode Chance', fmt: '.0f%%' },
  { stat: 'CombinedAvgToMonsterLife', label: 'Enemy Life Equivalent', fmt: '.1f%%' },
  { stat: 'Cooldown', label: 'Skill Cooldown', fmt: '.3fs', notFlag: 'skipEffectiveRate', lowerIsBetter: true },
  { stat: 'SealCooldown', label: 'Seal Gain Frequency', fmt: '.2fs', lowerIsBetter: true },
  { stat: 'SealMax', label: 'Max Number of Seals', fmt: 'd' },
  { stat: 'TimeMaxSeals', label: 'Time to Gain Max Seals', fmt: '.2fs', lowerIsBetter: true },
  { stat: 'AreaOfEffectRadiusMetres', label: 'AoE Radius', fmt: '.1fm', breakdown: 'AreaOfEffectRadius', modNames: ['AreaOfEffect'] },
  { stat: 'BrandAttachmentRangeMetre', label: 'Attachment Range', fmt: '.1fm', flag: 'brand' },
  { stat: 'BrandTicks', label: 'Activations per Brand', fmt: 'd', flag: 'brand' },
  {
    stat: 'ManaCost',
    label: 'Mana Cost',
    fmt: 'd',
    color: colorCodes.MANA,
    pool: 'ManaUnreserved',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.ManaHasCost)
  },
  {
    stat: 'ManaPercentCost',
    label: 'Mana Cost',
    fmt: 'd%%',
    color: colorCodes.MANA,
    pool: 'ManaUnreservedPercent',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.ManaPercentHasCost)
  },
  {
    stat: 'ManaPerSecondCost',
    label: 'Mana Cost per second',
    fmt: '.2f',
    color: colorCodes.MANA,
    pool: 'ManaUnreserved',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.ManaPerSecondHasCost)
  },
  {
    stat: 'ManaPercentPerSecondCost',
    label: 'Mana Cost per second',
    fmt: '.2f%%',
    color: colorCodes.MANA,
    pool: 'ManaUnreservedPercent',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.ManaPercentPerSecondHasCost)
  },
  {
    stat: 'LifeCost',
    label: 'Life Cost',
    fmt: 'd',
    color: colorCodes.LIFE,
    pool: 'LifeUnreserved',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.LifeHasCost)
  },
  {
    stat: 'LifePercentCost',
    label: 'Life Cost',
    fmt: 'd%%',
    color: colorCodes.LIFE,
    pool: 'LifeUnreservedPercent',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.LifePercentHasCost)
  },
  {
    stat: 'LifePerSecondCost',
    label: 'Life Cost per second',
    fmt: '.2f',
    color: colorCodes.LIFE,
    pool: 'LifeUnreserved',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.LifePerSecondHasCost)
  },
  {
    stat: 'LifePercentPerSecondCost',
    label: 'Life Cost per second',
    fmt: '.2f%%',
    color: colorCodes.LIFE,
    pool: 'LifeUnreservedPercent',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.LifePercentPerSecondHasCost)
  },
  {
    stat: 'ESCost',
    label: 'Energy Shield Cost',
    fmt: 'd',
    color: colorCodes.ES,
    pool: 'EnergyShield',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.ESHasCost)
  },
  {
    stat: 'ESPerSecondCost',
    label: 'ES Cost per second',
    fmt: '.2f',
    color: colorCodes.ES,
    pool: 'EnergyShield',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.ESPerSecondHasCost)
  },
  {
    stat: 'ESPercentPerSecondCost',
    label: 'ES Cost per second',
    fmt: '.2f%%',
    color: colorCodes.ES,
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.ESPercentPerSecondHasCost)
  },
  {
    stat: 'RageCost',
    label: 'Rage Cost',
    fmt: 'd',
    color: colorCodes.RAGE,
    pool: 'Rage',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.RageHasCost)
  },
  {
    stat: 'RagePerSecondCost',
    label: 'Rage Cost per second',
    fmt: '.2f',
    color: colorCodes.RAGE,
    pool: 'Rage',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.RagePerSecondHasCost)
  },
  {
    stat: 'SoulCost',
    label: 'Soul Cost',
    fmt: 'd',
    color: colorCodes.RAGE,
    pool: 'Soul',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => truthy(o.SoulHasCost)
  },
  {},
  { stat: 'Str', label: 'Strength', color: colorCodes.STRENGTH, fmt: 'd', breakdown: 'Str' },
  {
    stat: 'ReqStr',
    label: 'Strength Required',
    color: colorCodes.STRENGTH,
    fmt: 'd',
    lowerIsBetter: true,
    condFunc: (v, o) => v > n(o.Str),
    warnFunc: reqWarn('Strength', 'ReqStrItem')
  },
  { stat: 'Dex', label: 'Dexterity', color: colorCodes.DEXTERITY, fmt: 'd', breakdown: 'Dex' },
  {
    stat: 'ReqDex',
    label: 'Dexterity Required',
    color: colorCodes.DEXTERITY,
    fmt: 'd',
    lowerIsBetter: true,
    condFunc: (v, o) => v > n(o.Dex),
    warnFunc: reqWarn('Dexterity', 'ReqDexItem')
  },
  { stat: 'Int', label: 'Intelligence', color: colorCodes.INTELLIGENCE, fmt: 'd', breakdown: 'Int' },
  {
    stat: 'ReqInt',
    label: 'Intelligence Required',
    color: colorCodes.INTELLIGENCE,
    fmt: 'd',
    lowerIsBetter: true,
    condFunc: (v, o) => v > n(o.Int),
    warnFunc: reqWarn('Intelligence', 'ReqIntItem')
  },
  { stat: 'Omni', label: 'Omniscience', color: colorCodes.RARE, fmt: 'd' },
  {
    stat: 'ReqOmni',
    label: 'Omniscience Required',
    color: colorCodes.RARE,
    fmt: 'd',
    lowerIsBetter: true,
    condFunc: (v, o) => v > n(o.Omni),
    warnFunc: reqWarn('Omniscience', 'ReqOmniItem')
  },
  {},
  { stat: 'Devotion', label: 'Devotion', color: colorCodes.RARE, fmt: 'd' },
  {},
  {
    stat: 'TotalEHP',
    label: 'Effective Hit Pool',
    fmt: '.0f',
    suffix: 'Guard',
    compactValue: true,
    compPercent: true,
    suffixCondFunc: (v, o) => truthy(o.GuardSkillActive)
  },
  { stat: 'PvPTotalTakenHit', label: 'PvP Hit Taken', fmt: '.1f', flag: 'isPvP', compactValue: true, lowerIsBetter: true },
  { stat: 'PhysicalMaximumHitTaken', label: 'Phys Max Hit', fmt: '.0f', compactValue: true, color: colorCodes.PHYS, compPercent: true, suffix: maxHitSuffix },
  {
    stat: 'LightningMaximumHitTaken',
    label: 'Elemental Max Hit',
    fmt: '.0f',
    compactValue: true,
    color: colorCodes.LIGHTNING,
    compPercent: true,
    suffix: maxHitSuffix,
    condFunc: (v, o) => o.LightningMaximumHitTaken === o.ColdMaximumHitTaken && o.LightningMaximumHitTaken === o.FireMaximumHitTaken
  },
  {
    stat: 'FireMaximumHitTaken',
    label: 'Fire Max Hit',
    fmt: '.0f',
    compactValue: true,
    color: colorCodes.FIRE,
    compPercent: true,
    suffix: maxHitSuffix,
    condFunc: (v, o) => o.LightningMaximumHitTaken !== o.ColdMaximumHitTaken || o.LightningMaximumHitTaken !== o.FireMaximumHitTaken
  },
  {
    stat: 'ColdMaximumHitTaken',
    label: 'Cold Max Hit',
    fmt: '.0f',
    compactValue: true,
    color: colorCodes.COLD,
    compPercent: true,
    suffix: maxHitSuffix,
    condFunc: (v, o) => o.LightningMaximumHitTaken !== o.ColdMaximumHitTaken || o.LightningMaximumHitTaken !== o.FireMaximumHitTaken
  },
  {
    stat: 'LightningMaximumHitTaken',
    label: 'Lightning Max Hit',
    fmt: '.0f',
    compactValue: true,
    color: colorCodes.LIGHTNING,
    compPercent: true,
    suffix: maxHitSuffix,
    condFunc: (v, o) => o.LightningMaximumHitTaken !== o.ColdMaximumHitTaken || o.LightningMaximumHitTaken !== o.FireMaximumHitTaken
  },
  { stat: 'ChaosMaximumHitTaken', label: 'Chaos Max Hit', fmt: '.0f', compactValue: true, color: colorCodes.CHAOS, compPercent: true, suffix: maxHitSuffix },
  {},
  {
    stat: 'MainHand',
    childStat: 'Accuracy',
    label: 'MH Accuracy',
    fmt: 'd',
    condFunc: (v, o) => truthy(o.PreciseTechnique),
    warnFunc: (v, o) => v < n(o.Life) && 'You do not have enough Accuracy for Precise Technique',
    warnColor: true
  },
  {
    stat: 'OffHand',
    childStat: 'Accuracy',
    label: 'OH Accuracy',
    fmt: 'd',
    condFunc: (v, o) => truthy(o.PreciseTechnique),
    warnFunc: (v, o) => v < n(o.Life) && 'You do not have enough Accuracy for Precise Technique',
    warnColor: true
  },
  {
    stat: 'Life',
    label: 'Total Life',
    fmt: 'd',
    compactValue: true,
    color: colorCodes.LIFE,
    compPercent: true,
    modNames: ['Life'],
    ignoredSections: defenseIgnoredSections
  },
  { stat: 'Spec:LifeInc', label: '%Inc Life from Tree', fmt: 'd%%', color: colorCodes.LIFE, condFunc: (v, o) => v > 0 && n(o.Life) > 1 },
  {
    stat: 'LifeUnreserved',
    label: 'Unreserved Life',
    fmt: 'd',
    color: colorCodes.LIFE,
    condFunc: (v, o) => v < n(o.Life),
    compPercent: true,
    warnFunc: (v) => v <= 0 && 'Your unreserved Life is below 1'
  },
  { stat: 'LifeRecoverable', label: 'Life Recoverable', fmt: 'd', color: colorCodes.LIFE, condFunc: (v, o) => v < n(o.LifeUnreserved) },
  { stat: 'LifeUnreservedPercent', label: 'Unreserved Life', fmt: 'd%%', color: colorCodes.LIFE, condFunc: (v) => v < 100 },
  {
    stat: 'LifeRegenRecovery',
    label: 'Life Regen',
    fmt: '.1f',
    color: colorCodes.LIFE,
    condFunc: (v, o) => n(o.LifeRecovery) <= 0 && o.LifeRegenRecovery !== 0,
    breakdown: 'LifeRegenRecovery'
  },
  {
    stat: 'LifeRegenRecovery',
    label: 'Life Recovery',
    fmt: '.1f',
    color: colorCodes.LIFE,
    condFunc: (v, o) => n(o.LifeRecovery) > 0 && o.LifeRegenRecovery !== 0,
    breakdown: 'LifeRegenRecovery'
  },
  { stat: 'LifeRecharge', label: 'Life Recharge', fmt: '.1f', color: colorCodes.LIFE, condFunc: (v) => v > 0 },
  { stat: 'LifeLeechGainRate', label: 'Life Leech/On Hit Rate', fmt: '.1f', color: colorCodes.LIFE, compPercent: true, breakdown: 'LifeLeech' },
  { stat: 'LifeLeechGainPerHit', label: 'Life Leech/Gain per Hit', fmt: '.1f', color: colorCodes.LIFE, compPercent: true },
  {},
  {
    stat: 'Mana',
    label: 'Total Mana',
    fmt: 'd',
    compactValue: true,
    color: colorCodes.MANA,
    compPercent: true,
    modNames: ['Mana'],
    ignoredSections: defenseIgnoredSections
  },
  { stat: 'Spec:ManaInc', label: '%Inc Mana from Tree', color: colorCodes.MANA, fmt: 'd%%' },
  {
    stat: 'ManaUnreserved',
    label: 'Unreserved Mana',
    fmt: 'd',
    color: colorCodes.MANA,
    condFunc: (v, o) => v < n(o.Mana),
    compPercent: true,
    warnFunc: (v) => v < 0 && 'Your unreserved Mana is negative'
  },
  { stat: 'ManaUnreservedPercent', label: 'Unreserved Mana', fmt: 'd%%', color: colorCodes.MANA, condFunc: (v) => v < 100, breakdown: 'ManaReserved' },
  {
    stat: 'ManaRegenRecovery',
    label: 'Mana Regen',
    fmt: '.1f',
    color: colorCodes.MANA,
    condFunc: (v, o) => n(o.ManaRecovery) <= 0 && o.ManaRegenRecovery !== 0
  },
  {
    stat: 'ManaRegenRecovery',
    label: 'Mana Recovery',
    fmt: '.1f',
    color: colorCodes.MANA,
    condFunc: (v, o) => n(o.ManaRecovery) > 0 && o.ManaRegenRecovery !== 0
  },
  { stat: 'ManaLeechGainRate', label: 'Mana Leech/On Hit Rate', fmt: '.1f', color: colorCodes.MANA, compPercent: true },
  { stat: 'ManaLeechGainPerHit', label: 'Mana Leech/Gain per Hit', fmt: '.1f', color: colorCodes.MANA, compPercent: true },
  {},
  {
    stat: 'EnergyShield',
    label: 'Energy Shield',
    fmt: 'd',
    compactValue: true,
    color: colorCodes.ES,
    compPercent: true,
    modNames: ['EnergyShield', 'Defences'],
    ignoredSections: defenseIgnoredSections
  },
  { stat: 'EnergyShieldRecoveryCap', label: 'Recoverable ES', color: colorCodes.ES, fmt: 'd', condFunc: (v, o) => truthy(o.CappingES) },
  { stat: 'Spec:EnergyShieldInc', label: '%Inc ES from Tree', color: colorCodes.ES, fmt: 'd%%' },
  {
    stat: 'EnergyShieldRegenRecovery',
    label: 'ES Regen',
    color: colorCodes.ES,
    fmt: '.1f',
    condFunc: (v, o) => n(o.EnergyShieldRecovery) <= 0 && o.EnergyShieldRegenRecovery !== 0
  },
  {
    stat: 'EnergyShieldRegenRecovery',
    label: 'ES Recovery',
    color: colorCodes.ES,
    fmt: '.1f',
    condFunc: (v, o) => n(o.EnergyShieldRecovery) > 0 && o.EnergyShieldRegenRecovery !== 0
  },
  { stat: 'EnergyShieldLeechGainRate', label: 'ES Leech/On Hit Rate', color: colorCodes.ES, fmt: '.1f', compPercent: true },
  { stat: 'EnergyShieldLeechGainPerHit', label: 'ES Leech/Gain per Hit', color: colorCodes.ES, fmt: '.1f', compPercent: true },
  {},
  {
    stat: 'Ward',
    label: 'Ward',
    fmt: 'd',
    compactValue: true,
    color: colorCodes.WARD,
    compPercent: true,
    modNames: ['Ward'],
    ignoredSections: defenseIgnoredSections
  },
  {},
  { stat: 'Rage', label: 'Rage', fmt: 'd', color: colorCodes.RAGE, compPercent: true },
  { stat: 'RageRegenRecovery', label: 'Rage Regen', fmt: '.1f', color: colorCodes.RAGE, compPercent: true },
  {},
  { stat: 'TotalBuildDegen', label: 'Total Degen', fmt: '.1f', lowerIsBetter: true },
  { stat: 'TotalNetRegen', label: 'Total Net Recovery', fmt: '+.1f' },
  { stat: 'NetLifeRegen', label: 'Net Life Recovery', fmt: '+.1f', color: colorCodes.LIFE },
  { stat: 'NetManaRegen', label: 'Net Mana Recovery', fmt: '+.1f', color: colorCodes.MANA },
  { stat: 'NetEnergyShieldRegen', label: 'Net ES Recovery', fmt: '+.1f', color: colorCodes.ES },
  {},
  {
    stat: 'Evasion',
    label: 'Evasion rating',
    fmt: 'd',
    color: colorCodes.EVASION,
    compPercent: true,
    modNames: ['Evasion', 'ArmourAndEvasion', 'Defences'],
    ignoredSections: defenseIgnoredSections
  },
  { stat: 'Spec:EvasionInc', label: '%Inc Evasion from Tree', color: colorCodes.EVASION, fmt: 'd%%' },
  {
    stat: 'MeleeEvadeChance',
    label: 'Evade Chance',
    fmt: 'd%%',
    color: colorCodes.EVASION,
    condFunc: (v, o) => v > 0 && o.MeleeEvadeChance === o.ProjectileEvadeChance
  },
  {
    stat: 'MeleeEvadeChance',
    label: 'Melee Evade Chance',
    fmt: 'd%%',
    color: colorCodes.EVASION,
    condFunc: (v, o) => v > 0 && o.MeleeEvadeChance !== o.ProjectileEvadeChance
  },
  {
    stat: 'ProjectileEvadeChance',
    label: 'Projectile Evade Chance',
    fmt: 'd%%',
    color: colorCodes.EVASION,
    condFunc: (v, o) => v > 0 && o.MeleeEvadeChance !== o.ProjectileEvadeChance
  },
  {},
  {
    stat: 'Armour',
    label: 'Armour',
    fmt: 'd',
    compPercent: true,
    modNames: ['Armour', 'ArmourAndEvasion', 'Defences', 'ArmourDefense'],
    ignoredSections: defenseIgnoredSections
  },
  { stat: 'Spec:ArmourInc', label: '%Inc Armour from Tree', fmt: 'd%%' },
  { stat: 'PhysicalDamageReduction', label: 'Phys. Damage Reduction', fmt: 'd%%', condFunc: () => true },
  {},
  { stat: 'EffectiveBlockChance', label: 'Block Chance', fmt: '.3f%%', overCapStat: 'BlockChanceOverCap', breakdown: 'BlockChance' },
  { stat: 'EffectiveSpellBlockChance', label: 'Spell Block Chance', fmt: '.3f%%', overCapStat: 'SpellBlockChanceOverCap', breakdown: 'SpellBlockChance' },
  { stat: 'AttackDodgeChance', label: 'Attack Dodge Chance', fmt: 'd%%', overCapStat: 'AttackDodgeChanceOverCap' },
  { stat: 'SpellDodgeChance', label: 'Spell Dodge Chance', fmt: 'd%%', overCapStat: 'SpellDodgeChanceOverCap' },
  {
    stat: 'EffectiveSpellSuppressionChance',
    label: 'Spell Suppression Chance',
    fmt: '.2f%%',
    overCapStat: 'SpellSuppressionChanceOverCap',
    breakdown: '',
    modNames: ['SpellSuppressionChance']
  },
  {},
  { stat: 'FireResist', label: 'Fire Resistance', fmt: 'd%%', color: colorCodes.FIRE, condFunc: () => true, overCapStat: 'FireResistOverCap' },
  { stat: 'FireResistOverCap', label: 'Fire Res. Over Max', fmt: 'd%%', hideStat: true },
  { stat: 'ColdResist', label: 'Cold Resistance', fmt: 'd%%', color: colorCodes.COLD, condFunc: () => true, overCapStat: 'ColdResistOverCap' },
  { stat: 'ColdResistOverCap', label: 'Cold Res. Over Max', fmt: 'd%%', hideStat: true },
  {
    stat: 'LightningResist',
    label: 'Lightning Resistance',
    fmt: 'd%%',
    color: colorCodes.LIGHTNING,
    condFunc: () => true,
    overCapStat: 'LightningResistOverCap'
  },
  { stat: 'LightningResistOverCap', label: 'Lightning Res. Over Max', fmt: 'd%%', hideStat: true },
  {
    stat: 'ChaosResist',
    label: 'Chaos Resistance',
    fmt: 'd%%',
    color: colorCodes.CHAOS,
    overCapStat: 'ChaosResistOverCap',
    overCapStatCondFunc: (v, o) => !truthy(o.ChaosInoculation),
    suffix: 'Immune',
    suffixCondFunc: (v, o) => truthy(o.ChaosInoculation)
  },
  { stat: 'ChaosResistOverCap', label: 'Chaos Res. Over Max', fmt: 'd%%', hideStat: true },
  {},
  { stat: 'EffectiveMovementSpeedMod', label: 'Movement Speed Modifier', fmt: '+d%%', mod: true, condFunc: () => true },
  {},
  { stat: 'FullDPS', label: 'Full DPS', fmt: '.1f', compactValue: true, color: colorCodes.CURRENCY, compPercent: true },
  {
    stat: 'FullDotDPS',
    label: 'Full Dot DPS',
    fmt: '.1f',
    compactValue: true,
    color: colorCodes.CURRENCY,
    compPercent: true,
    condFunc: (v) => v >= DOT_DPS_CAP,
    warnFunc: () => 'Full Dot DPS exceeds in game limit'
  },
  {},
  { stat: 'SkillDPS', label: 'Skill DPS', compactValue: true, condFunc: () => true }
];

// src/Modules/BuildDisplayStats.lua:250-279
export const minionDisplayStats: Stat[] = [
  { stat: 'AverageDamage', label: 'Average Damage', fmt: '.1f', compactValue: true, compPercent: true },
  { stat: 'Speed', label: 'Attack/Cast Rate', fmt: '.2f', compPercent: true, condFunc: (v, o) => v > 0 && n(o.TriggerTime) === 0 },
  { stat: 'HitSpeed', label: 'Hit Rate', fmt: '.2f' },
  {
    stat: 'Speed',
    label: 'Effective Trigger Rate',
    fmt: '.2f',
    compPercent: true,
    notFlag: 'skipEffectiveRate',
    condFunc: (v, o) => n(o.TriggerTime) !== 0
  },
  { stat: 'TotalDPS', label: 'Hit DPS', fmt: '.1f', compactValue: true, compPercent: true },
  { stat: 'TotalDot', label: 'DoT DPS', fmt: '.1f', compactValue: true, compPercent: true },
  {
    stat: 'WithDotDPS',
    label: 'Total DPS inc. DoT',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    condFunc: (v, o) => v !== o.TotalDPS && n(o.PoisonDPS) === 0 && n(o.IgniteDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.BleedDPS) === 0
  },
  {
    stat: 'BleedDPS',
    label: 'Bleed DPS',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    warnFunc: (v) => v >= DOT_DPS_CAP && 'Minion Bleed DPS exceeds in game limit'
  },
  {
    stat: 'WithBleedDPS',
    label: 'Total DPS inc. Bleed',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.PoisonDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.IgniteDPS) === 0
  },
  {
    stat: 'IgniteDPS',
    label: 'Ignite DPS',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    warnFunc: (v) => v >= DOT_DPS_CAP && 'Minion Ignite DPS exceeds in game limit'
  },
  {
    stat: 'WithIgniteDPS',
    label: 'Total DPS inc. Ignite',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.PoisonDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.BleedDPS) === 0
  },
  {
    stat: 'PoisonDPS',
    label: 'Poison DPS',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    warnFunc: (v) => v >= DOT_DPS_CAP && 'Minion Poison dps exceeds in game limit'
  },
  { stat: 'PoisonDamage', label: 'Total Damage per Poison', fmt: '.1f', compactValue: true, compPercent: true },
  {
    stat: 'WithPoisonDPS',
    label: 'Total DPS inc. Poison',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.IgniteDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'DecayDPS', label: 'Decay DPS', fmt: '.1f', compactValue: true, compPercent: true },
  {
    stat: 'TotalDotDPS',
    label: 'Total DoT DPS',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    condFunc: (v, o) => v !== o.TotalDot && v !== o.ImpaleDPS && v !== o.TotalPoisonDPS && v !== (o.TotalIgniteDPS ?? o.IgniteDPS) && v !== o.BleedDPS,
    warnFunc: (v) => v >= DOT_DPS_CAP && 'Minion DoT DPS exceeds in game limit'
  },
  { stat: 'ImpaleDPS', label: 'Impale DPS', fmt: '.1f', compactValue: true, compPercent: true, flag: 'impale' },
  {
    stat: 'WithImpaleDPS',
    label: 'Total DPS inc. Impale',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    flag: 'impale',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.IgniteDPS) === 0 && n(o.PoisonDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'CullingDPS', label: 'Culling DPS', fmt: '.1f', compactValue: true, compPercent: true, condFunc: (v, o) => n(o.CullingDPS) > 0 },
  { stat: 'ReservationDPS', label: 'Reservation DPS', fmt: '.1f', compactValue: true, compPercent: true, condFunc: (v, o) => n(o.ReservationDPS) > 0 },
  {
    stat: 'CombinedDPS',
    label: 'Combined DPS',
    fmt: '.1f',
    compactValue: true,
    compPercent: true,
    condFunc: (v, o) => v !== n(o.TotalDPS) + n(o.TotalDot) && v !== o.WithImpaleDPS && v !== o.WithPoisonDPS && v !== o.WithIgniteDPS && v !== o.WithBleedDPS
  },
  { stat: 'Cooldown', label: 'Skill Cooldown', fmt: '.3fs', notFlag: 'skipEffectiveRate', lowerIsBetter: true },
  {
    stat: 'Life',
    label: 'Total Life',
    fmt: '.1f',
    compactValue: true,
    color: colorCodes.LIFE,
    compPercent: true,
    modNames: ['Life'],
    ignoredSections: defenseIgnoredSections
  },
  { stat: 'LifeRegenRecovery', label: 'Life Recovery', fmt: '.1f', color: colorCodes.LIFE },
  { stat: 'LifeLeechGainRate', label: 'Life Leech/On Hit Rate', fmt: '.1f', color: colorCodes.LIFE, compPercent: true },
  {
    stat: 'EnergyShield',
    label: 'Energy Shield',
    fmt: 'd',
    compactValue: true,
    color: colorCodes.ES,
    compPercent: true,
    modNames: ['EnergyShield', 'Defences'],
    ignoredSections: defenseIgnoredSections
  },
  { stat: 'EnergyShieldRegenRecovery', label: 'ES Recovery', fmt: '.1f', color: colorCodes.ES },
  { stat: 'EnergyShieldLeechGainRate', label: 'ES Leech/On Hit Rate', fmt: '.1f', color: colorCodes.ES, compPercent: true }
];
