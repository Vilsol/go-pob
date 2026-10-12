// Build.lua:1877-1936

import { displayStats, type Stat } from '../../src/lib/display/stats';
import { minionDisplayStats } from '../../src/lib/skill_tree/stat_compare';

export type WhatIfOutputs = { Output?: Record<string, number>; Minion?: Record<string, number>; SkillFlags?: Record<string, boolean> };

export type DiffBlock = { header: string; sections: { actor?: string; lines: string[] }[] };

const luaSigned = (fmt: string, v: number) => {
  const m = /^\+?(?:\.(\d+))?([df])(.*)$/.exec(fmt);
  if (!m) throw new Error(`unhandled fmt ${fmt}`);
  const n = m[2] === 'd' ? Math.trunc(v) : v;
  const body = m[2] === 'd' ? String(Math.abs(n)) : Math.abs(n).toFixed(Number(m[1] ?? 6));
  const negative = n < 0 || (m[2] === 'f' && Object.is(n, -0));
  return (negative ? '-' : '+') + body + m[3].replace(/%%/g, '%');
};

const thousands = (s: string) => s.replace(/\d+/, (i) => i.replace(/\B(?=(\d{3})+(?!\d))/g, ','));

const compareList = (
  list: (Stat & { notFlag?: string })[],
  flags: Record<string, boolean>,
  base: Record<string, number>,
  cmp: Record<string, number>,
  nodeCount?: number
) => {
  const lines: string[] = [];
  for (const s of list) {
    if (!s.stat || s.stat === 'SkillDPS') continue;
    if (s.flag && flags[s.flag] !== true) continue;
    if (s.notFlag && flags[s.notFlag] === true) continue;
    const v1 = cmp[s.stat] ?? 0;
    const v2 = base[s.stat] ?? 0;
    const diff = s.stat === 'FullDPS' && cmp[s.stat] === undefined ? 0 : v1 - v2;
    if (!(Math.abs(diff) > 0.001)) continue;
    if (s.condFunc && !s.condFunc(v1, cmp) && !s.condFunc(v2, base)) continue;
    const fmt = s.fmt ?? 'd';
    const scale = s.pc || s.mod ? 100 : 1;
    let value = luaSigned(fmt, diff * scale);
    const trimmed = /^([+-]?\d+\.\d+)(\D*)$/.exec(value);
    if (trimmed) value = trimmed[1].replace(/0+$/, '').replace(/\.$/, '') + trimmed[2];
    let line = `${thousands(value)} ${s.label}`;
    let pcPerPt = '';
    if (s.compPercent && v1 !== 0 && v2 !== 0) {
      const pc = (v1 / v2) * 100 - 100;
      line += ` (${luaSigned('.1f', pc)}%)`;
      if (nodeCount) pcPerPt = ` (${luaSigned('.1f', pc / nodeCount)}%)`;
    }
    if (nodeCount) line += ` [${luaSigned(fmt, (diff * scale) / nodeCount)}${pcPerPt} pp]`;
    lines.push(line);
  }
  return lines;
};

export const expectedBlock = (header: string, base: WhatIfOutputs, cmp: WhatIfOutputs, nodeCount?: number): DiffBlock | undefined => {
  const flags = base.SkillFlags ?? {};
  const sections: DiffBlock['sections'] = [];
  const minion = !!(base.Minion && cmp.Minion);
  if (minion) {
    const lines = compareList(minionDisplayStats, flags, base.Minion!, cmp.Minion!, nodeCount);
    if (lines.length) sections.push({ actor: 'Minion', lines });
  }
  const lines = compareList(displayStats.flat(), flags, base.Output ?? {}, cmp.Output ?? {}, nodeCount);
  if (lines.length) sections.push({ actor: minion ? 'Player' : undefined, lines });
  return sections.length ? { header, sections } : undefined;
};
