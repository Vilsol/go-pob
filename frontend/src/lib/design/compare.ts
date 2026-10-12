import type { calculator } from '$lib/types';
import type { StatCompareSection } from '$lib/skill_tree/stat_compare';
import type { CompareFixture, CompareSide, CompareStat, PowerRow, SkillsFixture, SocketGroupFixture } from './fixtures';

export type MetricId = string;
export type Metric = { id: MetricId; label: string };
export type CompareView = 'summary' | 'power' | 'tree' | 'items' | 'skills' | 'calcs' | 'config';
export const METRICS: Metric[] = [
  { id: 'dps', label: 'Full DPS' },
  { id: 'ehp', label: 'Effective Hit Pool' },
  { id: 'life', label: 'Life' }
];
export const metricIndex = (m: MetricId, list: Metric[] = METRICS) =>
  Math.max(
    0,
    list.findIndex((x) => x.id === m)
  );

const nf = new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 });
export const num = (v: number) => nf.format(Math.round(v));
export const signed = (v: number) => (v > 0 ? '+' : v < 0 ? '−' : '') + num(Math.abs(v));
export const pctOf = (v: number, base: number) => (v > 0 ? '+' : v < 0 ? '−' : '') + (base ? ((Math.abs(v) / Math.abs(base)) * 100).toFixed(1) : '0.0') + '%';
export const arrowed = (v: number, text: string) => (v > 0 ? `▲ ${text}` : v < 0 ? `▼ ${text}` : '= no change');
export const dirTone = (v: number) => (v > 0 ? 'up' : v < 0 ? 'down' : 'muted');

type Side = CompareSide & { skills: SkillsFixture; power: PowerRow[] };
type ItemRow = { slot: string; a: CompareFixture['items'][number]['a']; b: CompareFixture['items'][number]['b']; impact?: StatCompareSection[] };

const SLOT_ORDER = [
  'Weapon 1',
  'Weapon 2',
  'Helmet',
  'Body Armour',
  'Gloves',
  'Boots',
  'Amulet',
  'Ring 1',
  'Ring 2',
  'Belt',
  'Flask 1',
  'Flask 2',
  'Flask 3',
  'Flask 4',
  'Flask 5',
  'Weapon 1 Swap',
  'Weapon 2 Swap'
];
const slotRank = (s: string) => {
  const i = SLOT_ORDER.indexOf(s);
  return i < 0 ? SLOT_ORDER.length : i;
};

export function sidesOf(fx: CompareFixture, aSkills: SkillsFixture, swapped: boolean) {
  const a: Side = { ...fx.a, skills: aSkills };
  const b: Side = fx.b;
  const [A, B] = swapped ? [b, a] : [a, b];
  const items: ItemRow[] = [...fx.items]
    .sort((x, y) => slotRank(x.slot) - slotRank(y.slot) || x.slot.localeCompare(y.slot))
    .map((r) => (swapped ? { slot: r.slot, a: r.b, b: r.a, impact: r.impactB } : { slot: r.slot, a: r.a, b: r.b, impact: r.impactA }));
  const jewels = (fx.jewels ?? []).map((j) => (swapped ? { ...j, a: j.b, b: j.a, aWarn: j.bWarn, bWarn: j.aWarn } : j));
  const config = fx.config.map((c) => (swapped ? { ...c, a: c.b, b: c.a, aValue: c.bValue, bValue: c.aValue, impactA: c.impactB, impactB: c.impactA } : c));
  const flip = { a: 'b', b: 'a', both: 'both' } as const;
  const nodes = fx.tree.nodes.map((n) => (swapped ? { ...n, owner: flip[n.owner] } : n));
  return { A, B, items, jewels, config, tree: { nodes, edges: fx.tree.edges } };
}

export type SummaryRow = {
  key: string;
  label: string;
  a: string;
  b: string;
  winner?: 'a' | 'b';
  lead: string;
  pA: number;
  pB: number;
  diff?: string;
  diffBetter?: boolean;
};

const GROUPS: [RegExp, string][] = [
  [/^Minion:/, 'MINION'],
  [/DPS|AverageHit|AverageDamage|Speed|Crit/, 'OFFENCE'],
  [/Str|Dex|Int|Omni/, 'ATTRIBUTES'],
  [/EHP|MaximumHitTaken/, 'EFFECTIVE HIT POOL'],
  [/Life/, 'LIFE'],
  [/Mana/, 'MANA'],
  [/EnergyShield|Ward/, 'ENERGY SHIELD AND WARD'],
  [/Evasion|Armour|Evade|PhysicalDamageReduction/, 'ARMOUR AND EVASION'],
  [/Block|Suppression|Dodge/, 'AVOIDANCE'],
  [/Resist/, 'RESISTANCES']
];
const groupTitle = (keys: string[]) => GROUPS.find(([re]) => keys.some((k) => re.test(k)))?.[1] ?? 'OTHER';

export function summaryOf(A: CompareSide, B: CompareSide) {
  const bv = new Map(B.stats.flat().map((s) => [s.key, s]));
  const row = (s: CompareStat): SummaryRow | undefined => {
    const o = bv.get(s.key);
    if (!o) return undefined;
    const [x, y] = [s.v, o.v];
    const end = Math.max(Math.abs(x), Math.abs(y)) || 1;
    const winner = x === y ? undefined : x > y !== s.lower ? 'a' : 'b';
    const [w, l] = winner === 'a' ? [x, y] : [y, x];
    const diff = w - l;
    const sign = diff > 0 ? '+' : '−';
    const gap = s.pc ? `${sign}${Math.abs(diff).toFixed(1).replace(/\.0$/, '')} pp` : `${sign}${((Math.abs(diff) / (Math.abs(l) || 1)) * 100).toFixed(1)}%`;
    const label = s.key.startsWith('Minion:') ? `${s.label} (minion)` : s.label;
    return {
      key: s.key,
      label,
      a: s.text,
      b: o.text,
      winner,
      lead: winner ? `${winner.toUpperCase()} ${gap}` : '=',
      pA: (Math.abs(x) / end) * 100,
      pB: (Math.abs(y) / end) * 100,
      diff: s.diff || undefined,
      diffBetter: y - x > 0 !== s.lower
    };
  };
  const groups = A.stats
    .map((g) => ({ title: groupTitle(g.map((s) => s.key)), rows: g.map(row).filter((r): r is SummaryRow => !!r) }))
    .filter((g) => g.rows.length);
  const rows = groups.flatMap((g) => g.rows);
  const aw = rows.filter((r) => r.winner === 'a').length;
  const bw = rows.filter((r) => r.winner === 'b').length;
  return { groups, tally: `A better on ${aw} · B better on ${bw} · ${rows.length - aw - bw} equal` };
}

/* CompareTab.lua:4139-4157 */
export type GemDiff = {
  status: 'same' | 'level' | 'a' | 'b';
  gem: string;
  a: string;
  b: string;
  ia?: number;
  ib?: number;
  support?: boolean;
  imbued?: boolean;
};
export type GroupPair = { name: string; where: string; gems: GemDiff[]; ga?: number; gb?: number };

const gemNames = (g: SocketGroupFixture) => new Set(g.gems.map((m) => m.name));
const lq = (m: { level: number; quality: number }) => `${m.level}/${m.quality}`;

export function pairSkills(A: SkillsFixture, B: SkillsFixture) {
  const ag = (A.sets[A.activeSet]?.groups ?? []).map((g, gi) => ({ ...g, gi })).filter((g) => g.gems.length);
  const bg = (B.sets[B.activeSet]?.groups ?? []).map((g, gi) => ({ ...g, gi })).filter((g) => g.gems.length);
  const idx = (gs: SocketGroupFixture['gems'], name: string) => {
    const i = gs.findIndex((m) => m.name === name);
    return i < 0 ? undefined : i;
  };
  const kind = (m: SocketGroupFixture['gems'][number]) => ({ support: m.support, imbued: m.imbued });
  const scored: [number, number, number][] = [];
  ag.forEach((x, i) =>
    bg.forEach((y, j) => {
      const [p, q] = [gemNames(x), gemNames(y)];
      const shared = [...p].filter((n) => q.has(n)).length;
      if (shared) scored.push([shared / new Set([...p, ...q]).size, i, j]);
    })
  );
  scored.sort((u, v) => v[0] - u[0] || u[1] - v[1] || u[2] - v[2]);
  const usedA = new Set<number>();
  const usedB = new Set<number>();
  const both: GroupPair[] = [];
  for (const [, i, j] of scored) {
    if (usedA.has(i) || usedB.has(j)) continue;
    usedA.add(i);
    usedB.add(j);
    const [x, y] = [ag[i], bg[j]];
    const bm = new Map(y.gems.map((m) => [m.name, m]));
    const am = new Map(x.gems.map((m) => [m.name, m]));
    const gems: GemDiff[] = [
      ...x.gems.map((m, k): GemDiff => {
        const o = bm.get(m.name);
        return { status: !o ? 'a' : lq(o) === lq(m) ? 'same' : 'level', gem: m.name, a: lq(m), b: o ? lq(o) : '—', ia: k, ib: idx(y.gems, m.name), ...kind(m) };
      }),
      ...y.gems
        .map((m, k) => ({ m, k }))
        .filter(({ m }) => !am.has(m.name))
        .map(({ m, k }): GemDiff => ({ status: 'b', gem: m.name, a: '—', b: lq(m), ib: k, ...kind(m) }))
    ];
    both.push({ name: x.label, where: `A ${x.slot || 'no slot'} · B ${y.slot || 'no slot'}`, gems, ga: x.gi, gb: y.gi });
  }
  const only = (gs: (SocketGroupFixture & { gi: number })[], used: Set<number>, side: 'a' | 'b') =>
    gs
      .filter((_, i) => !used.has(i))
      .map((g) => ({
        name: g.label,
        where: `${side.toUpperCase()} ${g.slot || 'no slot'}`,
        gems: g.gems.map(
          (m, i): GemDiff => ({
            status: side,
            gem: m.name,
            a: side === 'a' ? lq(m) : '—',
            b: side === 'b' ? lq(m) : '—',
            [side === 'a' ? 'ia' : 'ib']: i,
            ...kind(m)
          })
        ),
        [side === 'a' ? 'ga' : 'gb']: g.gi
      }));
  return { both, onlyA: only(ag, usedA, 'a'), onlyB: only(bg, usedB, 'b') };
}

export type CalcsHeaderField = 'group' | 'mode' | 'showMinion' | 'MainActiveSkill' | 'SkillPart' | 'StageCount' | 'MineCount' | 'Minion' | 'MinionSkill';
export type CalcsHeaderSide = {
  name: string;
  groups: string[];
  skillNumber: number;
  buffMode: string;
  showMinion?: boolean;
  controls?: calculator.MainSkillControls;
  info: { label: string; text: string }[];
};
