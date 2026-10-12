import { displayStats } from './stats';

export type GaugeTone = 'life' | 'mana' | 'es' | 'fire' | 'cold' | 'light' | 'chaos';

export interface Gauge {
  pct: number;
  cap?: number;
  res?: number;
  seg?: number;
  warn?: boolean;
  tone?: GaugeTone;
}

type Output = Record<string, number | undefined>;

const MISC_CAPS: [RegExp, number][] = [
  [/EvadeChance$/, 95],
  [/SpellSuppressionChance$/, 100],
  [/^Avoid\w+Chance$/, 75]
];

const ELEMENT_TONES: Record<string, GaugeTone> = { Fire: 'fire', Cold: 'cold', Lightning: 'light', Chaos: 'chaos' };
const POOL_TONES: Record<string, GaugeTone> = { Life: 'life', Mana: 'mana', EnergyShield: 'es', ES: 'es' };

const COST_POOLS = Object.fromEntries(
  displayStats.filter((s) => s.stat && /^(Life|Mana|EnergyShield)/.test(s.pool ?? '') && !s.stat.includes('PerSecond')).map((s) => [s.stat!, s.pool!])
);

const RESIST = /^(Totem)?(Fire|Cold|Lightning|Chaos)Resist$/;
const SHARE = /^(Life|Mana|EnergyShield)(Un)?[Rr]eserved(Percent)?$/;
const BOUNDED = /(Chance|DamageReduction)$/;

const clamp = (v: number) => Math.max(0, Math.min(100, v));

export function statCap(stat: string, output: Output, breakdown?: string[]): number | undefined {
  const line = (breakdown ?? []).map((l) => /^Max: (-?\d+(?:\.\d+)?)%$/.exec(l)).find(Boolean);
  if (line) return Number(line[1]);
  const resist = RESIST.exec(stat);
  if (resist && output[stat] !== undefined && output[`Missing${stat}`] !== undefined) return output[stat] + output[`Missing${stat}`]!;
  const base = stat.replace(/^Effective/, '');
  if (/Chance$/.test(base) && typeof output[`${base}Max`] === 'number') return output[`${base}Max`];
  if (/^(Physical|Fire|Cold|Lightning|Chaos)DamageReduction$/.test(stat) && typeof output.DamageReductionMax === 'number') return output.DamageReductionMax;
  if (/Chance$/.test(base) && (output[`${base}OverCap`] ?? 0) > 0) return output[stat];
  return MISC_CAPS.find(([re]) => re.test(stat))?.[1];
}

export function statGauge(stat: string | undefined, output: Output, breakdown?: string[]): Gauge | undefined {
  if (!stat) return undefined;
  const v = output[stat];
  if (typeof v !== 'number') return undefined;
  const share = SHARE.exec(stat);
  if (share) {
    const [, pool, un] = share;
    const tone = POOL_TONES[pool];
    if (un) return { pct: clamp(output[`${pool}UnreservedPercent`] ?? 0), res: clamp(output[`${pool}ReservedPercent`] ?? 0), tone };
    return { pct: clamp(output[`${pool}ReservedPercent`] ?? 0), tone };
  }
  const pool = COST_POOLS[stat];
  if (pool) {
    const max = output[pool] ?? 0;
    const tone = POOL_TONES[/^(Life|Mana|ES)/.exec(stat)![1]];
    return { pct: max > 0 ? (v / max) * 100 : v > 0 ? Infinity : 0, warn: true, tone };
  }
  const resist = RESIST.exec(stat);
  if (resist || BOUNDED.test(stat)) return { pct: clamp(v), cap: statCap(stat, output, breakdown), tone: resist ? ELEMENT_TONES[resist[2]] : undefined };
  return undefined;
}
