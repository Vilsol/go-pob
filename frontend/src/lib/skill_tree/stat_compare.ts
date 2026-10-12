// src/Modules/Build.lua:1877-1936

import { printf } from 'fast-printf';
import { displayStats, type Stat } from '../display/stats';

export type CompareOutputs = {
  Output: Record<string, number>;
  Minion?: Record<string, number>;
};

export type StatCompareLine = {
  label: string;
  value: string;
  percent?: string;
  perPoint?: string;
  better: boolean;
  color?: string;
};

export type StatCompareSection = { actor?: 'Minion' | 'Player'; lines: StatCompareLine[] };

type CompareStat = Stat & { notFlag?: string };

// src/Modules/BuildDisplayStats.lua:209-238
export const minionDisplayStats: CompareStat[] = [
  { stat: 'AverageDamage', label: 'Average Damage', fmt: '.1f', compPercent: true },
  { stat: 'Speed', label: 'Attack/Cast Rate', fmt: '.2f', compPercent: true, condFunc: (v, o) => v > 0 && (o.TriggerTime || 0) == 0 },
  { stat: 'HitSpeed', label: 'Hit Rate', fmt: '.2f' },
  {
    stat: 'Speed',
    label: 'Effective Trigger Rate',
    fmt: '.2f',
    compPercent: true,
    notFlag: 'skipEffectiveRate',
    condFunc: (v, o) => (o.TriggerTime || 0) != 0
  },
  { stat: 'TotalDPS', label: 'Hit DPS', fmt: '.1f', compPercent: true },
  { stat: 'TotalDot', label: 'DoT DPS', fmt: '.1f', compPercent: true },
  {
    stat: 'WithDotDPS',
    label: 'Total DPS inc. DoT',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v != o.TotalDPS && (o.PoisonDPS || 0) == 0 && (o.IgniteDPS || 0) == 0 && (o.ImpaleDPS || 0) == 0 && (o.BleedDPS || 0) == 0
  },
  { stat: 'BleedDPS', label: 'Bleed DPS', fmt: '.1f', compPercent: true },
  {
    stat: 'WithBleedDPS',
    label: 'Total DPS inc. Bleed',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v != o.TotalDPS && (o.TotalDot || 0) == 0 && (o.PoisonDPS || 0) == 0 && (o.ImpaleDPS || 0) == 0 && (o.IgniteDPS || 0) == 0
  },
  { stat: 'IgniteDPS', label: 'Ignite DPS', fmt: '.1f', compPercent: true },
  {
    stat: 'WithIgniteDPS',
    label: 'Total DPS inc. Ignite',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v != o.TotalDPS && (o.TotalDot || 0) == 0 && (o.PoisonDPS || 0) == 0 && (o.ImpaleDPS || 0) == 0 && (o.BleedDPS || 0) == 0
  },
  { stat: 'PoisonDPS', label: 'Poison DPS', fmt: '.1f', compPercent: true },
  { stat: 'PoisonDamage', label: 'Total Damage per Poison', fmt: '.1f', compPercent: true },
  {
    stat: 'WithPoisonDPS',
    label: 'Total DPS inc. Poison',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v != o.TotalDPS && (o.TotalDot || 0) == 0 && (o.IgniteDPS || 0) == 0 && (o.ImpaleDPS || 0) == 0 && (o.BleedDPS || 0) == 0
  },
  { stat: 'DecayDPS', label: 'Decay DPS', fmt: '.1f', compPercent: true },
  {
    stat: 'TotalDotDPS',
    label: 'Total DoT DPS',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) => v != o.TotalDot && v != o.ImpaleDPS && v != o.TotalPoisonDPS && v != (o.TotalIgniteDPS ?? o.IgniteDPS) && v != o.BleedDPS
  },
  { stat: 'ImpaleDPS', label: 'Impale DPS', fmt: '.1f', compPercent: true, flag: 'impale' },
  {
    stat: 'WithImpaleDPS',
    label: 'Total DPS inc. Impale',
    fmt: '.1f',
    compPercent: true,
    flag: 'impale',
    condFunc: (v, o) => v != o.TotalDPS && (o.TotalDot || 0) == 0 && (o.IgniteDPS || 0) == 0 && (o.PoisonDPS || 0) == 0 && (o.BleedDPS || 0) == 0
  },
  { stat: 'CullingDPS', label: 'Culling DPS', fmt: '.1f', compPercent: true, condFunc: (v, o) => (o.CullingDPS || 0) > 0 },
  { stat: 'ReservationDPS', label: 'Reservation DPS', fmt: '.1f', compPercent: true, condFunc: (v, o) => (o.ReservationDPS || 0) > 0 },
  {
    stat: 'CombinedDPS',
    label: 'Combined DPS',
    fmt: '.1f',
    compPercent: true,
    condFunc: (v, o) =>
      v != (o.TotalDPS || 0) + (o.TotalDot || 0) && v != o.WithImpaleDPS && v != o.WithPoisonDPS && v != o.WithIgniteDPS && v != o.WithBleedDPS
  },
  { stat: 'Cooldown', label: 'Skill Cooldown', fmt: '.3fs', notFlag: 'skipEffectiveRate', lowerIsBetter: true },
  { stat: 'Life', label: 'Total Life', fmt: '.1f', compPercent: true },
  { stat: 'LifeRegenRecovery', label: 'Life Recovery', fmt: '.1f' },
  { stat: 'LifeLeechGainRate', label: 'Life Leech/On Hit Rate', fmt: '.1f', compPercent: true },
  { stat: 'EnergyShield', label: 'Energy Shield', fmt: 'd', compPercent: true },
  { stat: 'EnergyShieldRegenRecovery', label: 'ES Recovery', fmt: '.1f' },
  { stat: 'EnergyShieldLeechGainRate', label: 'ES Leech/On Hit Rate', fmt: '.1f', compPercent: true }
];

export const formatSigned = (fmt: string, v: number) => {
  const s = printf('%' + fmt.replace(/^\+/, ''), v);
  return s.startsWith('-') ? s : '+' + s;
};

// src/Modules/Common.lua:741-770
const formatNumSep = (s: string) => s.replace(/\d+/, (int) => int.replace(/\B(?=(\d{3})+(?!\d))/g, ','));

const matchFlags = (stat: CompareStat, flags: Record<string, boolean>) =>
  (!stat.flag || flags[stat.flag] === true) && (!stat.notFlag || flags[stat.notFlag] !== true);

export const compareStatList = (
  statList: CompareStat[],
  skillFlags: Record<string, boolean>,
  baseOutput: Record<string, number>,
  compareOutput: Record<string, number>,
  nodeCount?: number
): StatCompareLine[] => {
  const lines: StatCompareLine[] = [];
  for (const statData of statList) {
    if (!statData.stat || !matchFlags(statData, skillFlags) || statData.childStat || statData.stat === 'SkillDPS') {
      continue;
    }
    const statVal1 = compareOutput[statData.stat] ?? 0;
    const statVal2 = baseOutput[statData.stat] ?? 0;
    let diff = statVal1 - statVal2;
    if (statData.stat === 'FullDPS' && compareOutput[statData.stat] === undefined) {
      diff = 0;
    }
    if ((diff > 0.001 || diff < -0.001) && (!statData.condFunc || statData.condFunc(statVal1, compareOutput) || statData.condFunc(statVal2, baseOutput))) {
      const fmt = statData.fmt ?? 'd';
      const scale = statData.pc || statData.mod ? 100 : 1;
      let value = formatSigned(fmt, diff * scale);
      const m = /^([+-]?\d+\.\d+)(\D*)$/.exec(value);
      if (m) {
        value = m[1].replace(/0+$/, '').replace(/\.$/, '') + m[2];
      }
      value = formatNumSep(value);

      const line: StatCompareLine = {
        label: statData.label ?? statData.stat,
        value,
        better: (statData.lowerIsBetter && diff < 0) || (!statData.lowerIsBetter && diff > 0),
        ...(statData.color && { color: statData.color })
      };
      let pcPerPt = '';
      if (statData.compPercent && statVal1 !== 0 && statVal2 !== 0) {
        const pc = (statVal1 / statVal2) * 100 - 100;
        line.percent = `(${formatSigned('.1f', pc)}%)`;
        if (nodeCount) {
          pcPerPt = ` (${formatSigned('.1f', pc / nodeCount)}%)`;
        }
      }
      if (nodeCount) {
        line.perPoint = `[${formatSigned(fmt, (diff * scale) / nodeCount)}${pcPerPt} pp]`;
      }
      lines.push(line);
    }
  }
  return lines;
};

export const lineText = (line: StatCompareLine) => `${line.value} ${line.label}${line.percent ? ' ' + line.percent : ''}`;

export type StatDiffBlock = { header: string; sections: StatCompareSection[]; readyAt: number; calcMs: number };

/* PassiveTreeView.lua:1658-1706 */
export type NodeStatDiff = {
  node: number;
  blocks: StatDiffBlock[];
  pending: boolean;
  empty?: string;
};

export const addStatCompares = (
  skillFlags: Record<string, boolean>,
  base: CompareOutputs,
  compare: CompareOutputs,
  nodeCount?: number
): StatCompareSection[] => {
  const sections: StatCompareSection[] = [];
  const minion = base.Minion && compare.Minion;
  if (minion) {
    const lines = compareStatList(minionDisplayStats, skillFlags, base.Minion!, compare.Minion!, nodeCount);
    if (lines.length) sections.push({ actor: 'Minion', lines });
  }
  const lines = compareStatList(displayStats.flat(), skillFlags, base.Output, compare.Output, nodeCount);
  if (lines.length) sections.push({ actor: minion ? 'Player' : undefined, lines });
  return sections;
};
