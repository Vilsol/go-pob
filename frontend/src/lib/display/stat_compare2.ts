// src/Modules/Build.lua:2585-2640 (PoB-PoE2): the stat differences between two calculations, over the PoE2 sidebar's
// stat lists

import { formatNumSep, luaFormat } from './format_stat';
import { displayStats2, minionDisplayStats2, type Stat2 } from './stats2';
import type { StatCompareLine, StatCompareSection } from '../skill_tree/stat_compare';

export type CompareOutputs2 = { output: Record<string, number>; minion?: Record<string, number> };

export const compareStatList2 = (
  statList: Stat2[],
  skillFlags: Record<string, boolean>,
  baseOutput: Record<string, number>,
  compareOutput: Record<string, number>,
  nodeCount?: number
): StatCompareLine[] => {
  const lines: StatCompareLine[] = [];
  for (const statData of statList) {
    if (!statData.stat || (statData.flag && !skillFlags[statData.flag]) || statData.childStat || statData.stat === 'SkillDPS') continue;
    const statVal1 = compareOutput[statData.stat] ?? 0;
    const statVal2 = baseOutput[statData.stat] ?? 0;
    let diff = statVal1 - statVal2;
    if (statData.stat === 'FullDPS' && compareOutput[statData.stat] === undefined) diff = 0;
    if ((diff > 0.001 || diff < -0.001) && (!statData.condFunc || statData.condFunc(statVal1, compareOutput) || statData.condFunc(statVal2, baseOutput))) {
      const fmt = '+' + (statData.fmt ?? 'd').replace(/^\+/, '');
      const scale = statData.pc || statData.mod ? 100 : 1;
      const line: StatCompareLine = {
        label: statData.label ?? statData.stat,
        value: formatNumSep(luaFormat(fmt, diff * scale)),
        better: (statData.lowerIsBetter && diff < 0) || (!statData.lowerIsBetter && diff > 0)
      };
      let pcPerPt = '';
      if (statData.compPercent && statVal1 !== 0 && statVal2 !== 0) {
        const pc = (statVal1 / statVal2) * 100 - 100;
        line.percent = `(${luaFormat('+.1f', pc)}%)`;
        if (nodeCount) pcPerPt = ` (${luaFormat('+.1f', pc / nodeCount)}%)`;
      }
      if (nodeCount) line.perPoint = `[${luaFormat(fmt, (diff * scale) / nodeCount)}${pcPerPt} per point]`;
      lines.push(line);
    }
  }
  return lines;
};

export const addStatCompares2 = (
  skillFlags: Record<string, boolean>,
  minionSkillFlags: Record<string, boolean>,
  hasMinion: boolean,
  base: CompareOutputs2,
  compare: CompareOutputs2,
  nodeCount?: number
): StatCompareSection[] => {
  const sections: StatCompareSection[] = [];
  const minion = hasMinion && base.minion && compare.minion;
  if (minion) {
    const lines = compareStatList2(minionDisplayStats2, minionSkillFlags, base.minion!, compare.minion!, nodeCount);
    if (lines.length) sections.push({ actor: 'Minion', lines });
  }
  const lines = compareStatList2(displayStats2, skillFlags, base.output, compare.output, nodeCount);
  if (lines.length) sections.push({ actor: minion ? 'Player' : undefined, lines });
  return sections;
};
