// Build.lua:1014-1061
import { displayStats, minionDisplayStats } from './stats';
import { matchFlags, statValue, type StatListInput } from './stat_list';

export type SavedStat = { stat: string; value: string };
export type SavedFullDPS = { stat: string; value: string; skillPart: string; source: string };
export type SavedStats = { player: SavedStat[]; fullDPS: SavedFullDPS[]; minion: SavedStat[] };

// src/Modules/BuildDisplayStats.lua:240-249
const EXTRA_SAVE_STATS = [
  'PowerCharges',
  'PowerChargesMax',
  'FrenzyCharges',
  'FrenzyChargesMax',
  'EnduranceCharges',
  'EnduranceChargesMax',
  'ActiveTotemLimit',
  'ActiveMinionLimit'
];

export const luaToString = (v: number): string => {
  if (Number.isNaN(v)) return 'nan';
  if (!Number.isFinite(v)) return v > 0 ? 'inf' : '-inf';
  if (Number.isInteger(v) && Math.abs(v) < 1e14) return String(v);
  const p = v.toPrecision(14);
  const [mant, exp] = p.split('e');
  const trimmed = mant.includes('.') ? mant.replace(/\.?0+$/, '') : mant;
  if (exp === undefined) return trimmed;
  const e = Number(exp);
  return `${trimmed}e${e < 0 ? '-' : '+'}${String(Math.abs(e)).padStart(2, '0')}`;
};

type SavedInput = StatListInput & { Minion?: { Output?: Record<string, number> } };

// src/Modules/Build.lua:1014-1061
export const savedStats = (out: SavedInput): SavedStats => {
  const flags = out.SkillFlags ?? {};
  const player: SavedStat[] = [];
  const fullDPS: SavedFullDPS[] = [];
  const added = new Set<string>();
  for (const s of displayStats) {
    if (!matchFlags(s.flag, s.notFlag, flags)) continue;
    const name = s.stat && s.stat + (s.childStat ?? '');
    if (!name || added.has(name)) continue;
    if (s.stat === 'SkillDPS') {
      if (!out.SkillDPS) continue;
      for (const d of out.SkillDPS) {
        fullDPS.push({
          stat: d.Count >= 2 ? `${d.Count}x ${d.Name}` : d.Name,
          value: luaToString(d.DPS * d.Count),
          skillPart: d.SkillPart || '',
          source: d.Source || d.Trigger || ''
        });
      }
      added.add(name);
      continue;
    }
    const v = statValue(s, out);
    if (v !== undefined) {
      player.push({ stat: name, value: luaToString(v) });
      added.add(name);
    }
  }
  for (const stat of EXTRA_SAVE_STATS) {
    const v = out.Output?.[stat];
    if (v !== undefined) player.push({ stat, value: luaToString(v) });
  }
  const minion: SavedStat[] = [];
  if (out.Minion) {
    for (const s of minionDisplayStats) {
      const v = s.stat ? out.Minion.Output?.[s.stat] : undefined;
      if (s.stat && v !== undefined) minion.push({ stat: s.stat, value: luaToString(v) });
    }
  }
  return { player, fullDPS, minion };
};
