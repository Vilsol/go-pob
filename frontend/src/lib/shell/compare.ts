import { displayStats, minionDisplayStats, type Stat, type StatOutput } from '$lib/display/stats';
import { formatNumSep, luaFormat } from '$lib/display/format_stat';
import {
  configurations,
  configRelevanceQuery,
  isRelevant,
  isShowAllExcluded,
  relevanceSets,
  reverseConfigOptions,
  type AllVarTypes
} from '$lib/display/configurations';
import type { CompareStat, CompareItemSide, ConfigKind, GemAttr, Metrics, PowerRow, SkillsFixture } from '$lib/design/fixtures';
import type { CompareGroup, CompareItem, CompareJewel, CompareSideData } from './compare_types';
import type { StatCompareSection } from '$lib/skill_tree/stat_compare';
import { METRICS, type Metric } from '$lib/design/compare';
import type { calculator, exposition, pob } from '$lib/types';
import type { Node, Tree } from '$lib/skill_tree/types';
import { ascendancyOffsets, nodeCentre } from '$lib/skill_tree/geometry';

type SideOutputs = NonNullable<CompareSideData['outputs']>;

export type CompareMetric = Metric & { stat: string };
export const BASE_METRICS: CompareMetric[] = METRICS.map((m, i) => ({ ...m, stat: ['FullDPS', 'TotalEHP', 'Life'][i] }));
export const POWER_STATS = BASE_METRICS.map((m) => m.stat);

/* CompareTab.lua:968-974 */
export const compareMetrics = (stats: { Stat: string; Label: string }[]): CompareMetric[] => [
  ...BASE_METRICS,
  ...stats.filter((s) => s.Stat && !POWER_STATS.includes(s.Stat)).map((s) => ({ id: s.Stat, label: s.Label, stat: s.Stat }))
];

/* Data.lua:179-205 */
// src/Modules/Data.lua:179-205
const statValue = (out: Record<string, number>, minion: Record<string, number>, stat: string) => {
  if (stat === 'FullDPS') return out.FullDPS !== undefined ? out.FullDPS : (out.CombinedDPS ?? 0) + (minion.CombinedDPS ?? 0);
  if (stat.startsWith('Minion')) return minion[stat.slice('Minion'.length)] ?? 0;
  if (stat === 'AverageDamage' || stat === 'TotalDot' || stat.includes('DPS')) return (out[stat] ?? 0) + (minion[stat] ?? 0);
  return out[stat] ?? 0;
};

export const metricsOf = (o: SideOutputs | undefined, metrics: CompareMetric[] = BASE_METRICS): Metrics => {
  const out = o?.Output ?? {};
  const minion = o?.Minion?.Output ?? {};
  return Object.fromEntries(metrics.map((m) => [m.id, statValue(out, minion, m.stat)]));
};

// src/Classes/CompareTab.lua:101-123
const matchFlags = (req: string | undefined, not: string | undefined, flags: Record<string, boolean>) => (!req || !!flags[req]) && (!not || !flags[not]);

const fmtValue = (s: Stat, v: number) => formatNumSep(luaFormat(s.fmt ?? 'd', v * (s.pc || s.mod ? 100 : 1)));

// src/Classes/CompareTab.lua:3328-3341
const diffText = (s: Stat, pv: number, cv: number, scale: number) => {
  const diff = cv - pv;
  if (!(diff > 0.001 || diff < -0.001)) return '';
  const str = formatNumSep(luaFormat('+' + (s.fmt ?? 'd').replace(/^\+/, ''), diff * scale));
  return pv !== 0 ? `${str} (${luaFormat('+.1f%%', (cv / pv) * 100 - 100)})` : str;
};

// src/Classes/CompareTab.lua:3133-3189, src/Classes/CompareTab.lua:3280-3369
export const summaryStats = (a: SideOutputs | undefined, b: SideOutputs | undefined) => {
  const useMinion = !!a?.Minion?.Output || !!b?.Minion?.Output;
  const list = useMinion ? minionDisplayStats : displayStats;
  const actor = useMinion ? 'Minion' : 'Player';
  const pOut: StatOutput = (a?.Minion?.Output ?? a?.Output ?? {}) as StatOutput;
  const cOut: StatOutput = (b?.Minion?.Output ?? b?.Output ?? {}) as StatOutput;
  const pFlags = (a?.Minion?.Output ? a.Minion.SkillFlags : a?.SkillFlags) ?? {};
  const cFlags = (b?.Minion?.Output ? b.Minion.SkillFlags : b?.SkillFlags) ?? {};
  const A: CompareStat[][] = [];
  const B: CompareStat[][] = [];
  let ga: CompareStat[] = [];
  let gb: CompareStat[] = [];
  const close = () => {
    if (ga.length) {
      A.push(ga);
      B.push(gb);
    }
    ga = [];
    gb = [];
  };
  list.forEach((s, i) => {
    if (!s.stat && !s.label) return close();
    if (s.stat === 'SkillDPS' || s.hideStat) return;
    if (!matchFlags(s.flag, s.notFlag, pFlags) && !matchFlags(s.flag, s.notFlag, cFlags)) return;
    const key = `${actor}:${s.stat ?? s.label}#${i}`;
    const label = s.label ?? s.stat ?? '';
    if (s.stat) {
      const pv = s.childStat ? 0 : (pOut[s.stat] ?? 0);
      const cv = s.childStat ? 0 : (cOut[s.stat] ?? 0);
      if (!Number.isFinite(pv) || !Number.isFinite(cv)) return;
      if ((pv === 0 && cv === 0) || (s.condFunc && !s.condFunc(pv, pOut) && !s.condFunc(cv, cOut))) return;
      const scale = s.pc || s.mod ? 100 : 1;
      const base = { key, label, lower: !!s.lowerIsBetter, pc: !!(s.pc || s.mod), diff: diffText(s, pv, cv, scale) };
      ga.push({ ...base, v: pv * scale, text: fmtValue(s, pv) });
      gb.push({ ...base, v: cv * scale, text: fmtValue(s, cv) });
    } else if (s.condFunc) {
      const cond = s.condFunc as unknown as (o: StatOutput) => boolean;
      const [ps, cs] = [cond(pOut), cond(cOut)];
      if (!ps && !cs) return;
      const val = (s as Stat & { val?: string }).val ?? '';
      const base = { key, label, lower: false, pc: false };
      ga.push({ ...base, v: ps ? 1 : 0, text: ps ? val : '-' });
      gb.push({ ...base, v: cs ? 1 : 0, text: cs ? val : '-' });
    }
  });
  close();
  return { a: A, b: B };
};

const SKIPPED_SECTIONS = new Set(['Custom Modifiers', 'Map Modifiers and Player Debuffs']);
type Value = boolean | number | string | undefined;

// src/Classes/CompareTab.lua:1118-1125
export const normalizeConfigVals = (v: AllVarTypes, p: Value, c: Value): [Value, Value] => {
  if (v.type === 'check') return [p ?? false, c ?? false];
  if (v.type === 'count' || v.type === 'integer' || v.type === 'float') return [p ?? 0, c ?? 0];
  return [p, c];
};

// src/Classes/CompareTab.lua:1100-1114
export const formatConfigValue = (v: AllVarTypes, val: Value): string => {
  if (val === undefined) return '(not set)';
  if (v.type === 'check') return val ? 'Yes' : 'No';
  if (v.type === 'list') return String(v.list.find((l) => l.value === val)?.label ?? val).replace(/\^x[0-9A-Fa-f]{6}|\^\d/g, '');
  return String(val);
};

/* CompareTab.lua:1128-1160 */
export const configControl = (key: string): { kind?: ConfigKind; options?: { value: string | number; label: string }[] } => {
  const v = reverseConfigOptions[key];
  if (!v) return {};
  if (v.type === 'check') return { kind: 'check' };
  if (v.type === 'count' || v.type === 'integer' || v.type === 'countAllowZero' || v.type === 'float') return { kind: 'number' };
  if (v.type === 'list') return { kind: 'list', options: v.list.map((l) => ({ value: l.value as string | number, label: String(l.label) })) };
  return {};
};

/* CompareTab.lua:2111-2117 */
export type ConfigDiff = { key: string; section: string; label: string; a: string; b: string; aValue: Value; bValue: Value; diff?: boolean };

// src/Classes/CompareTab.lua:1186-1222, src/Classes/CompareTab.lua:2099-2110
export const configDiffs = (a: Record<string, Value>, b: Record<string, Value>): ConfigDiff[] => {
  const out: ConfigDiff[] = [];
  for (const section of configurations) {
    if (SKIPPED_SECTIONS.has(section.name)) continue;
    for (const v of section.variables) {
      if (v.type === 'text') continue;
      const [p, c] = normalizeConfigVals(v, a[v.var], b[v.var]);
      if (String(p) === String(c)) continue;
      out.push({
        key: v.var,
        section: section.name,
        label: v.label,
        a: formatConfigValue(v, a[v.var]),
        b: formatConfigValue(v, b[v.var]),
        aValue: a[v.var],
        bValue: b[v.var]
      });
    }
  }
  return out;
};

// src/Classes/CompareTab.lua:2099-2124
export const configRows = (
  a: Record<string, Value>,
  b: Record<string, Value>,
  relA: calculator.ConfigRelevance | undefined,
  relB: calculator.ConfigRelevance | undefined,
  showAll = false
): ConfigDiff[] => {
  const empty = relevanceSets(Object.fromEntries(Object.keys(configRelevanceQuery).map((k) => [k, []])) as unknown as calculator.ConfigRelevance)!;
  const setsA = relevanceSets(relA) ?? empty;
  const setsB = relevanceSets(relB) ?? empty;
  const out: ConfigDiff[] = [];
  for (const section of configurations) {
    if (SKIPPED_SECTIONS.has(section.name)) continue;
    const diffs: ConfigDiff[] = [];
    const commons: ConfigDiff[] = [];
    for (const v of section.variables) {
      if (v.type === 'text') continue;
      const [p, c] = normalizeConfigVals(v, a[v.var], b[v.var]);
      const diff = String(p) !== String(c);
      const row = {
        key: v.var,
        section: section.name,
        label: v.label,
        a: formatConfigValue(v, a[v.var]),
        b: formatConfigValue(v, b[v.var]),
        aValue: a[v.var],
        bValue: b[v.var],
        diff
      };
      if (diff) diffs.push(row);
      // CompareTab.lua:2113-2117
      else if (isRelevant(v, a, setsA) || isRelevant(v, b, setsB) || (showAll && !isShowAllExcluded(v))) commons.push(row);
    }
    out.push(...diffs, ...commons);
  }
  return out;
};

export const inputOf = (key: string, v: Value): pob.Input =>
  typeof v === 'boolean'
    ? { Name: key, Boolean: v }
    : typeof v === 'number'
      ? { Name: key, Number: v }
      : typeof v === 'string'
        ? { Name: key, String: v }
        : { Name: key };

// src/Classes/CompareTab.lua:2494-2504, src/Classes/CompareTab.lua:2598-2602
const shownImpact = (impact: number[] = [], metrics: CompareMetric[]) =>
  metrics.map((m, i) => {
    const ds = displayStats.find((d) => d.stat === m.stat);
    const v = (impact[i] ?? 0) * (ds?.pc || ds?.mod ? 100 : 1);
    return Number(luaFormat(ds?.fmt ?? '.1f', v)) === 0 ? 0 : (impact[i] ?? 0);
  });

export type ComparePowerRow = PowerRow & { nodeId?: number; slot?: string; key?: string };

export const powerRows = (
  entries: calculator.ComparePowerEntry[],
  giverItems: CompareItem[],
  receiverItems: CompareItem[],
  config: ConfigDiff[],
  receiver: 'a' | 'b',
  metrics: CompareMetric[] = BASE_METRICS
): ComparePowerRow[] =>
  entries.map((e): ComparePowerRow => {
    const impact = shownImpact(e.Impact, metrics);
    switch (e.Category) {
      case 'Tree':
        return {
          cat: 'tree',
          what: `Take ${e.Name}`,
          detail: e.PathDist >= 1000 ? 'no path reaches it' : `${e.PathDist} point path`,
          tone: 'text',
          points: e.PathDist >= 1000 ? undefined : e.PathDist,
          impact,
          nodeId: e.NodeID
        };
      case 'Item': {
        const giving = giverItems.find((i) => i.slot === e.SlotName);
        const replacing = receiverItems.find((i) => i.slot === e.SlotName);
        return {
          cat: 'items',
          what: e.Name,
          // CompareTab.lua:2848
          detail: !e.SlotName ? 'in a tree socket' : replacing ? `replaces ${replacing.name}` : 'slot empty here',
          tone: (giving?.rarity ?? e.Rarity).toLowerCase(),
          impact,
          slot: e.SlotName
        };
      }
      case 'Skill gem':
        return { cat: 'gems', what: e.Name, detail: 'socket group only in the other build', tone: 'gem', impact };
      case 'Support gem':
        return { cat: 'supports', what: e.Name, detail: 'support of the other build’s main skill', tone: 'gem', impact };
      default: {
        const d = config.find((c) => c.key === e.Var);
        const [from, to] = receiver === 'a' ? [d?.a, d?.b] : [d?.b, d?.a];
        const label = (d?.label ?? e.Var).replace(/:$/, '');
        return { cat: 'config', what: `${label}  (${from} -> ${to})`, detail: d?.section ?? '', tone: 'text', impact, key: e.Var };
      }
    }
  });

export const itemRows = (
  a: CompareItem[],
  b: CompareItem[],
  impactA: Record<string, StatCompareSection[] | undefined>,
  impactB: Record<string, StatCompareSection[] | undefined>
) => {
  const slots = [...new Set([...a.map((i) => i.slot), ...b.map((i) => i.slot)])];
  const brief = (it: CompareItem | undefined): CompareItemSide =>
    it ? { name: it.name, rarity: it.rarity.toLowerCase() as NonNullable<CompareItemSide>['rarity'], line: it.line, base: it.base } : null;
  return slots.map((slot) => ({
    slot,
    a: brief(a.find((i) => i.slot === slot)),
    b: brief(b.find((i) => i.slot === slot)),
    impactA: impactA[slot],
    impactB: impactB[slot]
  }));
};

// CompareTab.lua:3933-3938
export const jewelRows = (jewels: CompareJewel[]) => {
  const brief = (it: CompareItem | undefined): CompareItemSide =>
    it ? { name: it.name, rarity: it.rarity.toLowerCase() as NonNullable<CompareItemSide>['rarity'], line: it.line, base: it.base } : null;
  return jewels.map((j) => ({
    slot: j.slot,
    label: j.label,
    node: j.node,
    a: brief(j.a),
    b: brief(j.b),
    aWarn: !!j.a && !j.aAllocated,
    bWarn: !!j.b && !j.bAllocated
  }));
};

export const itemsInto = (receiver: CompareItem[], giver: CompareItem[]) => giver.filter((g) => receiver.find((r) => r.slot === g.slot)?.name !== g.name);

const ATTR: Record<string, GemAttr> = { STR: 'str', DEX: 'dex', INT: 'int', NONE: 'white' };

/* CompareTab.lua:4032-4052 */
const imbuedGem = (name: string | undefined, gemById: Map<string, exposition.SkillGem>) => {
  if (!name) return [];
  const want = `${name} support`.toLowerCase();
  const def = [...gemById.values()].find((d) => d.Base?.Name?.toLowerCase() === want);
  return [
    { name: def?.Base?.Name || name, level: 1, quality: 0, enabled: true, support: true, attr: (def && ATTR[def.GemType]) ?? 'white', count: 1, imbued: true }
  ];
};

export const skillsFixture = (side: CompareSideData, gemById: Map<string, exposition.SkillGem>): SkillsFixture => ({
  activeSet: side.skills.activeSet,
  sets: side.skills.sets.map((set) => ({
    title: set.title,
    groups: set.groups.map((g: CompareGroup) => ({
      label: g.label,
      slot: g.slot,
      enabled: g.enabled,
      inFull: g.inFull,
      count: g.count,
      sockets: '',
      gems: [
        ...g.gems.map((m) => {
          const def = gemById.get(m.gemId);
          return {
            name: def?.Base?.Name || m.nameSpec || m.skillId,
            level: m.level,
            quality: m.quality,
            enabled: m.enabled,
            support: !!def?.Support,
            attr: (def && ATTR[def.GemType]) ?? 'white',
            count: m.count
          };
        }),
        ...imbuedGem(g.imbued, gemById)
      ]
    }))
  }))
});

export const kindOf = (n: Node) =>
  n.classStartIndex !== undefined ? 'c' : n.isKeystone ? 'k' : n.isMastery ? 'm' : n.isJewelSocket ? 'j' : n.isNotable ? 'n' : n.ascendancyName ? 'a' : 's';

export const treeOverlay = (tree: Tree, aNodes: number[], bNodes: number[]) => {
  const offsets = ascendancyOffsets(tree);
  const inA = new Set(aNodes);
  const inB = new Set(bNodes);
  const placed = [...new Set([...aNodes, ...bNodes])]
    .sort((x, y) => x - y)
    .map((id) => ({ id, n: tree.nodes[id] }))
    .filter((e) => e.n)
    .map(({ id, n }) => ({ id, n, p: n.ascendancyName ? undefined : nodeCentre(tree, offsets, n) }));
  const pts = placed.filter((e) => e.p).map((e) => e.p!);
  const [x0, x1] = [Math.min(...pts.map((p) => p.x)), Math.max(...pts.map((p) => p.x))];
  const [y0, y1] = [Math.min(...pts.map((p) => p.y)), Math.max(...pts.map((p) => p.y))];
  const span = Math.max(x1 - x0, y1 - y0) || 1;
  const nodes = placed.map(({ id, n, p }) => ({
    id,
    name: n.name ?? String(id),
    kind: kindOf(n),
    owner: inA.has(id) && inB.has(id) ? ('both' as const) : inA.has(id) ? ('a' as const) : ('b' as const),
    x: p ? (p.x - x0) / span : null,
    y: p ? (p.y - y0) / span : null
  }));
  const index = new Map(nodes.map((n, i) => [n.id, i]));
  const edges: [number, number][] = [];
  nodes.forEach((e, i) => {
    if (e.x === null) return;
    for (const o of tree.nodes[e.id].out ?? []) {
      const j = index.get(Number(o));
      if (j !== undefined && nodes[j].x !== null) edges.push([i, j]);
    }
  });
  return { nodes, edges };
};

export const classRoots = (tree: Tree, className: string): number[] => {
  const index = tree.classes.findIndex((c) => c.name === className);
  const start = Object.values(tree.nodes).find((n) => n.classStartIndex === index);
  return [...(start?.out ?? []), ...(start?.in ?? [])]
    .map((id) => tree.nodes[id])
    .filter((n) => n?.skill !== undefined && n.ascendancyName === undefined)
    .map((n) => n.skill!);
};
