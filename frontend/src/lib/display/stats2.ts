// The PoE2 sidebar: PoB-PoE2's src/Modules/BuildDisplayStats.lua and the Build.lua code that lists it.
// The calculator's output table holds numbers only; a boolean or table output reads as nil here.

import { colorCodes } from './colors';
import { formatNumSep, luaFormat, SUFFIX_COLOR, VALUE_COLOR } from './format_stat';
import type { Stat, StatOutput } from './stats';
import type { StatLine } from './stat_list';

export interface Stat2 extends Omit<Stat, 'condFunc' | 'warnFunc'> {
  condFunc?: (v: number, o: StatOutput) => boolean;
  warnFunc?: (v: number, o: StatOutput) => string | false;
  /* BuildDisplayStats.lua:39-40, a label-only row */
  labelStat?: string;
  val?: string;
  labelCondFunc?: (o: StatOutput) => boolean;
}

// src/Data/Global.lua:74
const SPIRIT = '#F5D076';
// src/Modules/Data.lua:269
export const DOT_DPS_CAP2 = 35791394;

const n = (v: number | undefined) => v ?? 0;
const has = (v: number | undefined) => v !== undefined;
const ne = (a: number | undefined, b: number | undefined) => a !== b;
const yes = () => true;
const capWarn = (text: string) => (v: number) => v >= DOT_DPS_CAP2 && text;
const reqWarn = (attr: string) => () => `You do not meet the ${attr} requirement of `;

// src/Modules/BuildDisplayStats.lua:9
const defenseIgnoredSections = ['Base from Gear', 'Inc. from Tree'];

// src/Modules/BuildDisplayStats.lua:43-261
export const displayStats2: Stat2[] = [
  { stat: 'ActiveMinionLimit', label: 'Active Minion Limit', fmt: 'd' },
  { stat: 'AverageHit', label: 'Average Hit', fmt: '.1f', compPercent: true },
  { stat: 'PvpAverageHit', label: 'PvP Average Hit', fmt: '.1f', compPercent: true, flag: 'isPvP' },
  { stat: 'AverageDamage', label: 'Average Damage', fmt: '.1f', compPercent: true, flag: 'attack' },
  { stat: 'AverageDamage', label: 'Average Damage', fmt: '.1f', compPercent: true, flag: 'monsterExplode', condFunc: (v, o) => o.HitChance !== 100 },
  {
    stat: 'AverageBurstDamage',
    label: 'Average Burst Damage',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => has(o.AverageBurstHits) && n(o.AverageBurstHits) > 1 && v > 0
  },
  { stat: 'PvpAverageDamage', label: 'PvP Average Damage', fmt: '.1f', compPercent: true, flag: 'attackPvP' },
  {
    stat: 'Speed',
    label: 'Attack Rate',
    fmt: '.2f',
    compPercent: true,
    flag: 'attack',
    condFunc: (v, o) => v > 0 && !has(o.ChannelTime) && n(o.TriggerTime) === 0,
    breakdown: 'Speed',
    modNames: ['Speed'],
    ignoredSections: ['Inc. Cast Speed', 'More Cast Speed']
  },
  {
    stat: 'Speed',
    label: 'Cast Rate',
    fmt: '.2f',
    compPercent: true,
    flag: 'spell',
    condFunc: (v, o) => v > 0 && !has(o.ChannelTime) && n(o.TriggerTime) === 0,
    modNames: ['Speed']
  },
  { stat: 'Speed', label: 'Effective Trigger Rate', fmt: '.2f', compPercent: true, condFunc: (v, o) => n(o.TriggerTime) !== 0 },
  { stat: 'WarcryCastTime', label: 'Cast Time', fmt: '.2fs', compPercent: true, lowerIsBetter: true, flag: 'warcry' },
  {
    stat: 'ChannelTime',
    label: 'Channel Time',
    fmt: '.2fs',
    compPercent: true,
    flag: 'channelRelease',
    lowerIsBetter: true,
    condFunc: (v, o) => !has(o.TriggerTime)
  },
  { stat: 'HitSpeed', label: 'Hit Rate', fmt: '.2f', compPercent: true, condFunc: (v, o) => !has(o.TriggerTime) },
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
    condFunc: (v, o) => !has(o.TriggerTime)
  },
  { stat: 'FiringRate', label: 'Firing Rate', fmt: '.2f', compPercent: true, condFunc: (v, o) => has(o.FiringRate) },
  { stat: 'ReloadTime', label: 'Reload Time', fmt: '.2fs', compPercent: true, lowerIsBetter: true, condFunc: (v, o) => has(o.ReloadTime) },
  { stat: 'PreEffectiveCritChance', label: 'Crit Chance', fmt: '.2f%%', flag: 'hit', breakdown: 'PreEffectiveCritChance' },
  {
    stat: 'CritChance',
    label: 'Effective Crit Chance',
    fmt: '.2f%%',
    flag: 'hit',
    condFunc: (v, o) => v !== o.PreEffectiveCritChance,
    breakdown: 'CritChance'
  },
  { stat: 'CritBifurcates', label: 'Crit Bifurcate Chance', fmt: '.2f%%', flag: 'hit', condFunc: (v, o) => n(o.CritForks) > 0 },
  { stat: 'CritMultiplier', label: 'Crit Multiplier', fmt: 'd%%', pc: true, condFunc: (v, o) => n(o.CritChance) > 0, breakdown: 'CritMultiplier' },
  { stat: 'HitChance', label: 'Hit Chance', fmt: '.0f%%', flag: 'attack', breakdown: 'HitChance' },
  { stat: 'HitChance', label: 'Hit Chance', fmt: '.0f%%', condFunc: (v, o) => has(o.enemyHasSpellBlock) },
  {
    stat: 'AccuracyHitChanceUncapped',
    label: 'Uncap. Hit Chance',
    fmt: '.0f%%',
    flag: 'attack',
    condFunc: (v, o) => has(o.AccuracyHitChanceUncapped) && n(o.AccuracyHitChanceUncapped) > 100
  },
  { stat: 'TotalDPS', label: 'Hit DPS', fmt: '.1f', compPercent: true, flag: 'notAverage' },
  { stat: 'PvpTotalDPS', label: 'PvP Hit DPS', fmt: '.1f', compPercent: true, flag: 'notAveragePvP' },
  { stat: 'TotalDPS', label: 'Hit DPS', fmt: '.1f', compPercent: true, flag: 'showAverage', condFunc: (v, o) => n(o.TriggerTime) !== 0 },
  { stat: 'TotalDot', label: 'DoT DPS', fmt: '.1f', compPercent: true },
  {
    stat: 'WithDotDPS',
    label: 'Total DPS inc. DoT',
    fmt: '.1f',
    compPercent: true,
    flag: 'notAverage',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.PoisonDPS) === 0 && n(o.IgniteDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'BleedDPS', label: 'Bleed DPS', fmt: '.1f', compPercent: true, warnFunc: capWarn('Bleed DPS exceeds in game limit') },
  {
    stat: 'CorruptingBloodDPS',
    label: 'Corrupting Blood DPS',
    fmt: '.1f',
    compPercent: true,
    warnFunc: capWarn('Corrupting Blood DPS exceeds in game limit')
  },
  { stat: 'BleedDamage', label: 'Total Damage per Bleed', fmt: '.1f', compPercent: true, flag: 'showAverage' },
  {
    stat: 'WithBleedDPS',
    label: 'Total DPS inc. Bleed',
    fmt: '.1f',
    compPercent: true,
    flag: 'notAverage',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.PoisonDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.IgniteDPS) === 0
  },
  { stat: 'IgniteDPS', label: 'Ignite DPS', fmt: '.1f', compPercent: true, warnFunc: capWarn('Ignite DPS exceeds in game limit') },
  { stat: 'IgniteDamage', label: 'Total Damage per Ignite', fmt: '.1f', compPercent: true, flag: 'showAverage' },
  {
    stat: 'BurningGroundDPS',
    label: 'Burning Ground DPS',
    fmt: '.1f',
    compPercent: true,
    warnFunc: capWarn('Burning Ground DPS exceeds in game limit')
  },
  {
    stat: 'MirageBurningGroundDPS',
    label: 'Mirage Burning Ground DPS',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v !== o.BurningGroundDPS,
    warnFunc: capWarn('Mirage Burning Ground DPS exceeds in game limit')
  },
  {
    stat: 'WithIgniteDPS',
    label: 'Total DPS inc. Ignite',
    fmt: '.1f',
    compPercent: true,
    flag: 'notAverage',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.PoisonDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'PoisonDPS', label: 'Poison DPS', fmt: '.1f', compPercent: true, warnFunc: capWarn('Poison DPS exceeds in game limit') },
  {
    stat: 'CausticGroundDPS',
    label: 'Caustic Ground DPS',
    fmt: '.1f',
    compPercent: true,
    warnFunc: capWarn('Caustic Ground DPS exceeds in game limit')
  },
  {
    stat: 'MirageCausticGroundDPS',
    label: 'Mirage Caustic Ground DPS',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v !== o.CausticGroundDPS,
    warnFunc: capWarn('Mirage Caustic Ground DPS exceeds in game limit')
  },
  { stat: 'PoisonDamage', label: 'Total Damage per Poison', fmt: '.1f', compPercent: true, flag: 'showAverage' },
  {
    stat: 'WithPoisonDPS',
    label: 'Total DPS inc. Poison',
    fmt: '.1f',
    compPercent: true,
    flag: 'notAverage',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.IgniteDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'DecayDPS', label: 'Decay DPS', fmt: '.1f', compPercent: true },
  {
    stat: 'TotalDotDPS',
    label: 'Total DoT DPS',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) =>
      has(o.showTotalDotDPS) ||
      (ne(v, o.TotalDot) &&
        ne(v, o.TotalPoisonDPS) &&
        ne(v, o.CausticGroundDPS) &&
        ne(v, o.TotalIgniteDPS ?? o.IgniteDPS) &&
        ne(v, o.BurningGroundDPS) &&
        ne(v, o.BleedDPS) &&
        ne(v, o.CorruptingBloodDPS) &&
        ne(v, o.MirageCausticGroundDPS) &&
        ne(v, o.MirageBurningGroundDPS)),
    warnFunc: capWarn('DoT DPS exceeds in game limit')
  },
  // BuildDisplayStats.lua:94-97 give flag twice; the second, showAverage or notAverage, wins
  { stat: 'ImpaleDPS', label: 'Impale Damage', fmt: '.1f', compPercent: true, flag: 'showAverage' },
  {
    stat: 'WithImpaleDPS',
    label: 'Damage inc. Impale',
    fmt: '.1f',
    compPercent: true,
    flag: 'showAverage',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.IgniteDPS) === 0 && n(o.PoisonDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'ImpaleDPS', label: 'Impale DPS', fmt: '.1f', compPercent: true, flag: 'notAverage' },
  {
    stat: 'WithImpaleDPS',
    label: 'Total DPS inc. Impale',
    fmt: '.1f',
    compPercent: true,
    flag: 'notAverage',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.IgniteDPS) === 0 && n(o.PoisonDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'MirageDPS', label: 'Total Mirage DPS', fmt: '.1f', compPercent: true, flag: 'mirageArcher', condFunc: (v) => v > 0 },
  { stat: 'MirageDPS', label: 'Total Wisp DPS', fmt: '.1f', compPercent: true, flag: 'wisp', condFunc: (v) => v > 0 },
  { stat: 'CullingDPS', label: 'Culling DPS', fmt: '.1f', compPercent: true, condFunc: (v, o) => n(o.CullingDPS) > 0 },
  { stat: 'ReservationDPS', label: 'Reservation DPS', fmt: '.1f', compPercent: true, condFunc: (v, o) => n(o.ReservationDPS) > 0 },
  {
    stat: 'CombinedDPS',
    label: 'Combined DPS',
    fmt: '.1f',
    compPercent: true,
    flag: 'notAverage',
    condFunc: (v, o) =>
      v !== n(o.TotalDPS) + n(o.TotalDot) &&
      ne(v, o.WithImpaleDPS) &&
      (has(o.showTotalDotDPS) || (ne(v, o.WithPoisonDPS) && ne(v, o.WithIgniteDPS) && ne(v, o.WithBleedDPS)))
  },
  {
    stat: 'CombinedAvg',
    label: 'Combined Total Damage',
    fmt: '.1f',
    compPercent: true,
    flag: 'showAverage',
    condFunc: (v, o) => v !== o.AverageDamage && n(o.TotalDot) === 0 && (ne(v, o.WithPoisonDPS) || ne(v, o.WithIgniteDPS) || ne(v, o.WithBleedDPS))
  },
  { stat: 'ExplodeChance', label: 'Total Explode Chance', fmt: '.0f%%' },
  { stat: 'CombinedAvgToMonsterLife', label: 'Enemy Life Equivalent', fmt: '.1f%%' },
  { stat: 'Cooldown', label: 'Skill Cooldown', fmt: '.3fs', lowerIsBetter: true },
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
    condFunc: (v, o) => has(o.ManaHasCost)
  },
  {
    stat: 'ManaPercentCost',
    label: 'Mana Cost',
    fmt: 'd%%',
    color: colorCodes.MANA,
    pool: 'ManaUnreservedPercent',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.ManaPercentHasCost)
  },
  {
    stat: 'ManaPerSecondCost',
    label: 'Mana Cost per second',
    fmt: '.2f',
    color: colorCodes.MANA,
    pool: 'ManaUnreserved',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.ManaPerSecondHasCost)
  },
  {
    stat: 'ManaPercentPerSecondCost',
    label: 'Mana Cost per second',
    fmt: '.2f%%',
    color: colorCodes.MANA,
    pool: 'ManaUnreservedPercent',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.ManaPercentPerSecondHasCost)
  },
  {
    stat: 'LifeCost',
    label: 'Life Cost',
    fmt: 'd',
    color: colorCodes.LIFE,
    pool: 'LifeUnreserved',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.LifeHasCost)
  },
  {
    stat: 'LifePercentCost',
    label: 'Life Cost',
    fmt: 'd%%',
    color: colorCodes.LIFE,
    pool: 'LifeUnreservedPercent',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.LifePercentHasCost)
  },
  {
    stat: 'LifePerSecondCost',
    label: 'Life Cost per second',
    fmt: '.2f',
    color: colorCodes.LIFE,
    pool: 'LifeUnreserved',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.LifePerSecondHasCost)
  },
  {
    stat: 'LifePercentPerSecondCost',
    label: 'Life Cost per second',
    fmt: '.2f%%',
    color: colorCodes.LIFE,
    pool: 'LifeUnreservedPercent',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.LifePercentPerSecondHasCost)
  },
  {
    stat: 'ESCost',
    label: 'Energy Shield Cost',
    fmt: 'd',
    color: colorCodes.ES,
    pool: 'EnergyShield',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.ESHasCost)
  },
  {
    stat: 'ESPerSecondCost',
    label: 'ES Cost per second',
    fmt: '.2f',
    color: colorCodes.ES,
    pool: 'EnergyShield',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.ESPerSecondHasCost)
  },
  {
    stat: 'ESPercentPerSecondCost',
    label: 'ES Cost per second',
    fmt: '.2f%%',
    color: colorCodes.ES,
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.ESPercentPerSecondHasCost)
  },
  {
    stat: 'WardCost',
    label: 'Runic Ward Cost',
    fmt: 'd',
    color: colorCodes.WARD,
    pool: 'Ward',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.WardHasCost)
  },
  {
    stat: 'WardPercentCost',
    label: 'Runic Ward Cost',
    fmt: 'd%%',
    color: colorCodes.WARD,
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.WardPercentHasCost)
  },
  {
    stat: 'WardPerSecondCost',
    label: 'Runic Ward Cost / s',
    fmt: '.2f',
    color: colorCodes.WARD,
    pool: 'Ward',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.WardPerSecondHasCost)
  },
  {
    stat: 'RageCost',
    label: 'Rage Cost',
    fmt: 'd',
    color: colorCodes.RAGE,
    pool: 'Rage',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.RageHasCost)
  },
  {
    stat: 'RagePerSecondCost',
    label: 'Rage Cost per second',
    fmt: '.2f',
    color: colorCodes.RAGE,
    pool: 'Rage',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.RagePerSecondHasCost)
  },
  {
    stat: 'SoulCost',
    label: 'Soul Cost',
    fmt: 'd',
    color: colorCodes.RAGE,
    pool: 'Soul',
    compPercent: true,
    lowerIsBetter: true,
    condFunc: (v, o) => has(o.SoulHasCost)
  },
  {},
  { stat: 'Str', label: 'Strength', color: colorCodes.STRENGTH, fmt: 'd', breakdown: 'Str', modNames: [''] },
  {
    stat: 'ReqStr',
    label: 'Strength Required',
    color: colorCodes.STRENGTH,
    fmt: 'd',
    lowerIsBetter: true,
    condFunc: (v, o) => v > n(o.Str),
    warnFunc: reqWarn('Strength')
  },
  { stat: 'Dex', label: 'Dexterity', color: colorCodes.DEXTERITY, fmt: 'd', breakdown: 'Dex', modNames: [''] },
  {
    stat: 'ReqDex',
    label: 'Dexterity Required',
    color: colorCodes.DEXTERITY,
    fmt: 'd',
    lowerIsBetter: true,
    condFunc: (v, o) => v > n(o.Dex),
    warnFunc: reqWarn('Dexterity')
  },
  { stat: 'Int', label: 'Intelligence', color: colorCodes.INTELLIGENCE, fmt: 'd', breakdown: 'Int', modNames: [''] },
  {
    stat: 'ReqInt',
    label: 'Intelligence Required',
    color: colorCodes.INTELLIGENCE,
    fmt: 'd',
    lowerIsBetter: true,
    condFunc: (v, o) => v > n(o.Int),
    warnFunc: reqWarn('Intelligence')
  },
  {},
  { stat: 'Devotion', label: 'Devotion', color: colorCodes.RARE, fmt: 'd' },
  { stat: 'Tribute', label: 'Tribute', color: colorCodes.RARE, fmt: 'd' },
  {},
  { stat: 'TotalEHP', label: 'Effective Hit Pool', fmt: '.0f', compPercent: true },
  { stat: 'PvPTotalTakenHit', label: 'PvP Hit Taken', fmt: '.1f', flag: 'isPvP', lowerIsBetter: true },
  { stat: 'PhysicalMaximumHitTaken', label: 'Phys Max Hit', fmt: '.0f', color: colorCodes.PHYS, compPercent: true },
  {
    stat: 'LightningMaximumHitTaken',
    label: 'Elemental Max Hit',
    fmt: '.0f',
    color: colorCodes.LIGHTNING,
    compPercent: true,
    condFunc: (v, o) => o.LightningMaximumHitTaken === o.ColdMaximumHitTaken && o.LightningMaximumHitTaken === o.FireMaximumHitTaken
  },
  {
    stat: 'FireMaximumHitTaken',
    label: 'Fire Max Hit',
    fmt: '.0f',
    color: colorCodes.FIRE,
    compPercent: true,
    condFunc: (v, o) => o.LightningMaximumHitTaken !== o.ColdMaximumHitTaken || o.LightningMaximumHitTaken !== o.FireMaximumHitTaken
  },
  {
    stat: 'ColdMaximumHitTaken',
    label: 'Cold Max Hit',
    fmt: '.0f',
    color: colorCodes.COLD,
    compPercent: true,
    condFunc: (v, o) => o.LightningMaximumHitTaken !== o.ColdMaximumHitTaken || o.LightningMaximumHitTaken !== o.FireMaximumHitTaken
  },
  {
    stat: 'LightningMaximumHitTaken',
    label: 'Lightning Max Hit',
    fmt: '.0f',
    color: colorCodes.LIGHTNING,
    compPercent: true,
    condFunc: (v, o) => o.LightningMaximumHitTaken !== o.ColdMaximumHitTaken || o.LightningMaximumHitTaken !== o.FireMaximumHitTaken
  },
  { stat: 'ChaosMaximumHitTaken', label: 'Chaos Max Hit', fmt: '.0f', color: colorCodes.CHAOS, compPercent: true },
  {},
  {
    stat: 'MainHand',
    childStat: 'Accuracy',
    label: 'MH Accuracy',
    fmt: 'd',
    condFunc: (v, o) => has(o.PreciseTechnique),
    warnFunc: (v, o) => v < n(o.Life) && 'You do not have enough Accuracy for Precise Technique',
    warnColor: true
  },
  {
    stat: 'OffHand',
    childStat: 'Accuracy',
    label: 'OH Accuracy',
    fmt: 'd',
    condFunc: (v, o) => has(o.PreciseTechnique),
    warnFunc: (v, o) => v < n(o.Life) && 'You do not have enough Accuracy for Precise Technique',
    warnColor: true
  },
  {},
  { stat: 'Darkness', label: 'Total Darkness', fmt: 'd', color: colorCodes.CHAOS },
  { stat: 'ReservedDarkness', label: 'Reserved Darkness', fmt: 'd', color: colorCodes.CHAOS },
  {},
  { stat: 'Spirit', label: 'Total Spirit', fmt: 'd', color: SPIRIT, compPercent: true, ignoredSections: defenseIgnoredSections, modNames: ['Spirit'] },
  {
    stat: 'SpiritUnreserved',
    label: 'Unreserved Spirit',
    fmt: 'd',
    color: SPIRIT,
    condFunc: (v, o) => v < n(o.Spirit),
    compPercent: true,
    warnFunc: (v) => v < 0 && 'Your unreserved Spirit is negative',
    breakdown: 'SpiritReserved'
  },
  { stat: 'SpiritUnreservedPercent', label: 'Unreserved Spirit', fmt: 'd%%', color: SPIRIT, condFunc: (v) => v < 100, breakdown: 'SpiritReserved' },
  {},
  {
    stat: 'Life',
    label: 'Total Life',
    fmt: 'd',
    color: colorCodes.LIFE,
    compPercent: true,
    modNames: ['Life'],
    ignoredSections: ['Base from Gear', 'Inc. from Tree']
  },
  { stat: 'Spec:LifeInc', label: '%Inc Life', fmt: 'd%%', color: colorCodes.LIFE, condFunc: (v, o) => v > 0 && n(o.Life) > 1 },
  {
    stat: 'LifeUnreserved',
    label: 'Unreserved Life',
    fmt: 'd',
    color: colorCodes.LIFE,
    condFunc: (v, o) => v < n(o.Life),
    compPercent: true,
    warnFunc: (v) => v <= 0 && 'Your unreserved Life is below 1',
    breakdown: 'LifeReserved',
    modNames: ['']
  },
  { stat: 'LifeRecoverable', label: 'Life Recoverable', fmt: 'd', color: colorCodes.LIFE, condFunc: (v, o) => v < n(o.LifeUnreserved) },
  {
    stat: 'LifeUnreservedPercent',
    label: 'Unreserved Life',
    fmt: 'd%%',
    color: colorCodes.LIFE,
    condFunc: (v) => v < 100,
    breakdown: 'LifeReserved',
    modNames: ['']
  },
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
  { stat: 'LifeLeechGainRate', label: 'Life Leech/On Hit Rate', fmt: '.1f', color: colorCodes.LIFE, compPercent: true, breakdown: 'LifeLeech' },
  { stat: 'LifeLeechGainPerHit', label: 'Life Leech/Gain per Hit', fmt: '.1f', color: colorCodes.LIFE, compPercent: true },
  {},
  {
    stat: 'Mana',
    label: 'Total Mana',
    fmt: 'd',
    color: colorCodes.MANA,
    compPercent: true,
    modNames: ['Mana'],
    ignoredSections: ['Base from Gear', 'Inc. from Tree']
  },
  { stat: 'Spec:ManaInc', label: '%Inc Mana from Tree', color: colorCodes.MANA, fmt: 'd%%' },
  {
    stat: 'ManaUnreserved',
    label: 'Unreserved Mana',
    fmt: 'd',
    color: colorCodes.MANA,
    condFunc: (v, o) => v < n(o.Mana),
    compPercent: true,
    warnFunc: (v) => v < 0 && 'Your unreserved Mana is negative',
    breakdown: 'ManaReserved'
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
  { stat: 'ManaLeechGainRate', label: 'Mana Leech/On Hit Rate', fmt: '.1f', color: colorCodes.MANA, compPercent: true, breakdown: 'ManaLeech' },
  { stat: 'ManaLeechGainPerHit', label: 'Mana Leech/Gain per Hit', fmt: '.1f', color: colorCodes.MANA, compPercent: true },
  {},
  {
    stat: 'EnergyShield',
    label: 'Energy Shield',
    fmt: 'd',
    color: colorCodes.ES,
    compPercent: true,
    modNames: ['EnergyShield', 'Defences'],
    ignoredSections: defenseIgnoredSections
  },
  { stat: 'EnergyShieldRecoveryCap', label: 'Recoverable ES', color: colorCodes.ES, fmt: 'd', condFunc: (v, o) => has(o.CappingES) },
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
  {
    stat: 'EnergyShieldLeechGainRate',
    label: 'ES Leech/On Hit Rate',
    color: colorCodes.ES,
    fmt: '.1f',
    compPercent: true,
    breakdown: 'EnergyShieldLeech'
  },
  { stat: 'EnergyShieldLeechGainPerHit', label: 'ES Leech/Gain per Hit', color: colorCodes.ES, fmt: '.1f', compPercent: true },
  {},
  { stat: 'Ward', label: 'Runic Ward', fmt: 'd', color: colorCodes.WARD, compPercent: true },
  {
    stat: 'WardRegenRecovery',
    label: 'Runic Ward Regen',
    fmt: '.1f',
    color: colorCodes.WARD,
    condFunc: (v, o) => n(o.WardRecovery) <= 0 && o.WardRegenRecovery !== 0
  },
  {
    stat: 'WardRegenRecovery',
    label: 'Runic Ward Recovery',
    fmt: '.1f',
    color: colorCodes.WARD,
    condFunc: (v, o) => n(o.WardRecovery) > 0 && o.WardRegenRecovery !== 0
  },
  {},
  { stat: 'Rage', label: 'Rage', fmt: 'd', color: colorCodes.RAGE, compPercent: true },
  { stat: 'RageRegenRecovery', label: 'Rage Regen', fmt: '.1f', color: colorCodes.RAGE, compPercent: true },
  {},
  { stat: 'TotalBuildDegen', label: 'Total Degen', fmt: '.1f', lowerIsBetter: true },
  { stat: 'TotalNetRegen', label: 'Total Net Recovery', fmt: '+.1f' },
  { stat: 'NetLifeRegen', label: 'Net Life Recovery', fmt: '+.1f', color: colorCodes.LIFE },
  { stat: 'NetManaRegen', label: 'Net Mana Recovery', fmt: '+.1f', color: colorCodes.MANA },
  { stat: 'NetWardRegen', label: 'Net Runic Ward Recovery', fmt: '+.1f', color: colorCodes.WARD },
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
  { stat: 'EvadeChance', label: 'Evade Chance', fmt: 'd%%', color: colorCodes.EVASION, condFunc: (v, o) => v > 0 && has(o.noSplitEvade) },
  { stat: 'MeleeEvadeChance', label: 'Melee Evade Chance', fmt: 'd%%', color: colorCodes.EVASION, condFunc: (v, o) => v > 0 && has(o.splitEvade) },
  {
    stat: 'ProjectileEvadeChance',
    label: 'Projectile Evade Chance',
    fmt: 'd%%',
    color: colorCodes.EVASION,
    condFunc: (v, o) => v > 0 && has(o.splitEvade)
  },
  { stat: 'SpellEvadeChance', label: 'Spell Evade Chance', fmt: 'd%%', color: colorCodes.EVASION, condFunc: (v, o) => v > 0 && has(o.splitEvade) },
  {
    stat: 'SpellProjectileEvadeChance',
    label: 'Spell Proj. Evade Chance',
    fmt: 'd%%',
    color: colorCodes.EVASION,
    condFunc: (v, o) => v > 0 && has(o.splitEvade)
  },
  {
    stat: 'DeflectionRating',
    label: 'Deflection Rating',
    fmt: 'd',
    color: colorCodes.EVASION,
    compPercent: true,
    modNames: ['EvasionGainAsDeflection', 'ArmourGainAsDeflection', 'DeflectionRating'],
    breakdown: ''
  },
  { stat: 'DeflectChance', label: 'Deflect Chance', fmt: 'd%%', color: colorCodes.EVASION, compPercent: true },
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
  { stat: 'PhysicalDamageReduction', label: 'Phys. Damage Reduction', fmt: 'd%%', condFunc: yes },
  {},
  { stat: 'EffectiveBlockChance', label: 'Block Chance', fmt: '.3f%%', overCapStat: 'BlockChanceOverCap', breakdown: 'BlockChance' },
  {
    stat: 'EffectiveSpellBlockChance',
    label: 'Spell Block Chance',
    fmt: '.3f%%',
    overCapStat: 'SpellBlockChanceOverCap',
    breakdown: 'SpellBlockChance'
  },
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
  { stat: 'FireResist', label: 'Fire Resistance', fmt: 'd%%', color: colorCodes.FIRE, condFunc: yes, overCapStat: 'FireResistOverCap' },
  { stat: 'FireResistOverCap', label: 'Fire Res. Over Max', fmt: 'd%%', hideStat: true },
  { stat: 'ColdResist', label: 'Cold Resistance', fmt: 'd%%', color: colorCodes.COLD, condFunc: yes, overCapStat: 'ColdResistOverCap' },
  { stat: 'ColdResistOverCap', label: 'Cold Res. Over Max', fmt: 'd%%', hideStat: true },
  {
    stat: 'LightningResist',
    label: 'Lightning Resistance',
    fmt: 'd%%',
    color: colorCodes.LIGHTNING,
    condFunc: yes,
    overCapStat: 'LightningResistOverCap'
  },
  { stat: 'LightningResistOverCap', label: 'Lightning Res. Over Max', fmt: 'd%%', hideStat: true },
  {
    stat: 'ChaosResist',
    label: 'Chaos Resistance',
    fmt: 'd%%',
    color: colorCodes.CHAOS,
    condFunc: (v, o) => !has(o.ChaosInoculation),
    overCapStat: 'ChaosResistOverCap'
  },
  { stat: 'ChaosResistOverCap', label: 'Chaos Res. Over Max', fmt: 'd%%', hideStat: true },
  { label: 'Chaos Resistance', val: 'Immune', labelStat: 'ChaosResist', color: colorCodes.CHAOS, labelCondFunc: (o) => has(o.ChaosInoculation) },
  {},
  { stat: 'EffectiveMovementSpeedMod', label: 'Movement Speed Modifier', fmt: '+.1f%%', mod: true, condFunc: yes },
  { stat: 'MovementSpeedWhileUsingSkill', label: 'Skill Movement Speed', fmt: '+.1f%%', mod: true, condFunc: yes },
  {},
  { stat: 'PresenceRadiusMetres', label: 'Presence Radius', fmt: '.1fm', compPercent: true },
  {},
  { stat: 'FullDPS', label: 'Full DPS', fmt: '.1f', color: colorCodes.CURRENCY, compPercent: true },
  {
    stat: 'FullDotDPS',
    label: 'Full Dot DPS',
    fmt: '.1f',
    color: colorCodes.CURRENCY,
    compPercent: true,
    condFunc: (v) => v >= DOT_DPS_CAP2,
    warnFunc: () => 'Full Dot DPS exceeds in game limit'
  },
  {},
  { stat: 'SkillDPS', label: 'Skill DPS', condFunc: yes }
];

// src/Modules/BuildDisplayStats.lua:263-292
export const minionDisplayStats2: Stat2[] = [
  { stat: 'AverageDamage', label: 'Average Damage', fmt: '.1f', compPercent: true },
  { stat: 'Speed', label: 'Attack/Cast Rate', fmt: '.2f', compPercent: true, condFunc: (v, o) => v > 0 && n(o.TriggerTime) === 0 },
  { stat: 'HitSpeed', label: 'Hit Rate', fmt: '.2f' },
  { stat: 'Speed', label: 'Effective Trigger Rate', fmt: '.2f', compPercent: true, condFunc: (v, o) => n(o.TriggerTime) !== 0 },
  { stat: 'TotalDPS', label: 'Hit DPS', fmt: '.1f', compPercent: true },
  { stat: 'TotalDot', label: 'DoT DPS', fmt: '.1f', compPercent: true },
  {
    stat: 'WithDotDPS',
    label: 'Total DPS inc. DoT',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v !== o.TotalDPS && n(o.PoisonDPS) === 0 && n(o.IgniteDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'BleedDPS', label: 'Bleed DPS', fmt: '.1f', compPercent: true, warnFunc: capWarn('Minion Bleed DPS exceeds in game limit') },
  {
    stat: 'WithBleedDPS',
    label: 'Total DPS inc. Bleed',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.PoisonDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.IgniteDPS) === 0
  },
  { stat: 'IgniteDPS', label: 'Ignite DPS', fmt: '.1f', compPercent: true, warnFunc: capWarn('Minion Ignite DPS exceeds in game limit') },
  {
    stat: 'WithIgniteDPS',
    label: 'Total DPS inc. Ignite',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.PoisonDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'PoisonDPS', label: 'Poison DPS', fmt: '.1f', compPercent: true, warnFunc: capWarn('Minion Poison dps exceeds in game limit') },
  { stat: 'PoisonDamage', label: 'Total Damage per Poison', fmt: '.1f', compPercent: true },
  {
    stat: 'WithPoisonDPS',
    label: 'Total DPS inc. Poison',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.IgniteDPS) === 0 && n(o.ImpaleDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'DecayDPS', label: 'Decay DPS', fmt: '.1f', compPercent: true },
  {
    stat: 'TotalDotDPS',
    label: 'Total DoT DPS',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => ne(v, o.TotalDot) && ne(v, o.ImpaleDPS) && ne(v, o.TotalPoisonDPS) && ne(v, o.TotalIgniteDPS ?? o.IgniteDPS) && ne(v, o.BleedDPS),
    warnFunc: capWarn('Minion DoT DPS exceeds in game limit')
  },
  { stat: 'ImpaleDPS', label: 'Impale DPS', fmt: '.1f', compPercent: true, flag: 'impale' },
  {
    stat: 'WithImpaleDPS',
    label: 'Total DPS inc. Impale',
    fmt: '.1f',
    compPercent: true,
    flag: 'impale',
    condFunc: (v, o) => v !== o.TotalDPS && n(o.TotalDot) === 0 && n(o.IgniteDPS) === 0 && n(o.PoisonDPS) === 0 && n(o.BleedDPS) === 0
  },
  { stat: 'CullingDPS', label: 'Culling DPS', fmt: '.1f', compPercent: true, condFunc: (v, o) => n(o.CullingDPS) > 0 },
  { stat: 'ReservationDPS', label: 'Reservation DPS', fmt: '.1f', compPercent: true, condFunc: (v, o) => n(o.ReservationDPS) > 0 },
  {
    stat: 'CombinedDPS',
    label: 'Combined DPS',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) =>
      v !== n(o.TotalDPS) + n(o.TotalDot) && ne(v, o.WithImpaleDPS) && ne(v, o.WithPoisonDPS) && ne(v, o.WithIgniteDPS) && ne(v, o.WithBleedDPS)
  },
  { stat: 'Cooldown', label: 'Skill Cooldown', fmt: '.3fs', lowerIsBetter: true },
  {
    stat: 'Life',
    label: 'Total Life',
    fmt: '.1f',
    color: colorCodes.LIFE,
    compPercent: true,
    modNames: ['Life'],
    ignoredSections: ['Base from Gear', 'Inc. from Tree']
  },
  { stat: 'LifeRegenRecovery', label: 'Life Recovery', fmt: '.1f', color: colorCodes.LIFE },
  { stat: 'LifeLeechGainRate', label: 'Life Leech/On Hit Rate', fmt: '.1f', color: colorCodes.LIFE, compPercent: true },
  {
    stat: 'EnergyShield',
    label: 'Energy Shield',
    fmt: 'd',
    color: colorCodes.ES,
    compPercent: true,
    modNames: ['EnergyShield', 'Defences'],
    ignoredSections: defenseIgnoredSections
  },
  { stat: 'EnergyShieldRegenRecovery', label: 'ES Recovery', fmt: '.1f', color: colorCodes.ES },
  { stat: 'EnergyShieldLeechGainRate', label: 'ES Leech/On Hit Rate', fmt: '.1f', color: colorCodes.ES, compPercent: true }
];

// src/Modules/Build.lua (PoB-PoE2) FormatStat: no compact values and no trailing-zero trim, unlike PoE1's
export const formatStat2 = (statData: Stat2, statVal: number, overCapStatVal?: number, colorOverride?: string) => {
  const val = statVal * (statData.pc || statData.mod ? 100 : 1) - (statData.mod ? 100 : 0);
  let color = colorOverride ?? (statVal >= 0 ? VALUE_COLOR : statData.chaosInoc ? SUFFIX_COLOR : colorCodes.NEGATIVE);
  if (statData.label === 'Unreserved Life' && statVal === 0) {
    color = colorCodes.NEGATIVE;
  }
  const value = formatNumSep(luaFormat(statData.fmt ?? 'd', val));
  const overCap = overCapStatVal !== undefined && overCapStatVal > 0 ? ` (+${luaFormat('d', overCapStatVal)}%)` : undefined;
  return { value, color, overCap };
};

export interface StatListInput2 {
  Output?: Record<string, number>;
  SkillFlags?: Record<string, boolean>;
}

// src/Modules/Build.lua:2375-2481 (PoB-PoE2). SkillDPS is a table output the calculator does not hold.
export const buildStatList2 = (input: StatListInput2, statList: Stat2[], warnings: string[] = []): StatLine[][] => {
  const output = (input.Output ?? {}) as StatOutput;
  const flags = input.SkillFlags ?? {};
  const groups: StatLine[][] = [];
  let group: StatLine[] = [];
  const spacer = () => {
    if (group.length > 0) {
      groups.push(group);
      group = [];
    }
  };
  for (const statData of statList) {
    if (statData.flag !== undefined && !flags[statData.flag]) continue;
    if (!statData.stat) {
      if (statData.label && statData.labelCondFunc?.(output)) {
        group.push({
          label: statData.label,
          labelColor: statData.color ?? VALUE_COLOR,
          value: `${output[statData.labelStat!] ?? ''}%`,
          valueColor: VALUE_COLOR,
          note: ` (${statData.val})`
        });
      } else {
        spacer();
      }
      continue;
    }
    if (statData.stat === 'SkillDPS') continue;
    if (statData.childStat) continue;
    const statVal = output[statData.stat];
    if (statVal === undefined) continue;
    const shown = statData.condFunc ? statData.condFunc(statVal, output) : statVal !== 0;
    if (shown && !statData.hideStat) {
      const overCapStatVal = statData.overCapStat !== undefined ? output[statData.overCapStat] : undefined;
      let colorOverride: string | undefined;
      if (output[`${statData.stat}Warning`] !== undefined || (statData.warnFunc && statData.warnFunc(statVal, output) && statData.warnColor)) {
        colorOverride = colorCodes.NEGATIVE;
      }
      const formatted = formatStat2(statData, statVal, overCapStatVal, colorOverride);
      let note = formatted.overCap ?? '';
      if (statData.suffix && (!statData.suffixCondFunc || statData.suffixCondFunc(statVal, output))) {
        const suffix = typeof statData.suffix === 'function' ? statData.suffix(statVal, output) : statData.suffix;
        if (suffix) note += ` (${suffix})`;
      }
      group.push({
        label: statData.label!,
        labelColor: statData.color ?? VALUE_COLOR,
        value: formatted.value,
        valueColor: formatted.color,
        note: note || undefined,
        stat: statData.stat
      });
    }
    if (statData.warnFunc && (!statData.condFunc || shown)) {
      const line = statData.warnFunc(statVal, output);
      if (line && !warnings.includes(line)) warnings.push(line);
    }
  }
  spacer();
  return groups;
};
