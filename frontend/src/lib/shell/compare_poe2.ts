// CompareTab.lua (PoB-PoE2): the Compare page's parts that read a PoE2 build

import { displayStats2, minionDisplayStats2 } from '$lib/display/stats2';
import type { Stat } from '$lib/display/stats';
import { configurations2, currentValue, isVariable2, type ConfigValue2, type Variable2 } from '$lib/display/configurations2';
import type { CompareFixture, CompareItemSide, ConfigKind, GemAttr, ItemTip, SkillsFixture } from '$lib/design/fixtures';
import type { CompareSets } from '$lib/design/compare';
import type { Tree2 } from '$lib/components/skill-tree/scene2';
import type { Poe2Calc, Poe2CalcsPage, Poe2CompareItemRow, Poe2CompareParts, Poe2Item } from '$lib/go/sync_worker2';
import type { CalcsActor2 } from '$lib/display/calcs2';
import type { CalcSectionEntry2, CalcSectionRow2 } from '$lib/calcs/calc_section_types2';
import type { CompareCalcSection } from '$lib/display/compare_calcs';
import { metricsOf, powerRows, summaryStats, type CompareMetric, type ComparePowerRow } from './compare';
import type { calculator2 } from '$lib/types2';

type Value = ConfigValue2 | undefined;

const plain = (s: string) => s.replace(/\^x[0-9A-Fa-f]{6}|\^\d/g, '');

export const sideOutputs2 = (v: Poe2Calc) => ({
  Output: v.output,
  SkillFlags: v.skillFlags,
  Minion: v.minion ? { Output: v.minion, SkillFlags: v.minionSkillFlags ?? {} } : undefined
});

const LISTS2 = { player: displayStats2 as Stat[], minion: minionDisplayStats2 as Stat[] };

// src/Classes/CompareTab.lua:3162-3397 (PoB-PoE2)
export const summaryStats2 = (a: Poe2Calc, b: Poe2Calc) => summaryStats(sideOutputs2(a) as never, sideOutputs2(b) as never, LISTS2);

// src/Classes/CompareTab.lua:1152-1166 (PoB-PoE2)
export const formatConfigValue2 = (v: Variable2, val: Value): string => {
  if (val === undefined) return '(not set)';
  if (v.type === 'check') return val ? 'Yes' : 'No';
  if (v.type === 'list' && v.list) return plain(String(v.list.find((l) => l.val === val)?.label ?? val));
  return String(val);
};

// src/Classes/CompareTab.lua:1170-1177 (PoB-PoE2)
const normalizeConfigVals2 = (v: Variable2, p: Value, c: Value): [Value, Value] => {
  if (v.type === 'check') return [p ?? false, c ?? false];
  if (v.type === 'count' || v.type === 'integer' || v.type === 'float') return [p ?? 0, c ?? 0];
  return [p, c];
};

const EXCLUDED_WORDS = ['recently', 'in the last', 'in the past', 'in last', 'in past', 'pvp'];

// src/Classes/CompareTab.lua:1263-1288 (PoB-PoE2)
const visibility2 = (v: Variable2) => {
  const isHardConditional = !!(v.ifOption || v.ifSkill || v.ifSkillData || v.ifSkillFlag || v.legacy);
  const label = v.label.toLowerCase();
  const isKeywordExcluded = EXCLUDED_WORDS.some((kw) => label.includes(kw));
  const hasAnyCondition = !!(
    v.ifCond ||
    v.ifOption ||
    v.ifSkill ||
    v.ifSkillFlag ||
    v.ifSkillData ||
    v.ifMod ||
    v.ifMult ||
    v.ifEnemyStat ||
    v.ifEnemyCond ||
    v.legacy
  );
  return { alwaysShow: !hasAnyCondition && !isKeywordExcluded, showWithToggle: !isHardConditional && !isKeywordExcluded };
};

/* CompareTab.lua:1180-1224 (PoB-PoE2) */
const configControl2 = (v: Variable2): { kind?: ConfigKind; options?: { value: string | number; label: string }[] } => {
  if (v.type === 'check') return { kind: 'check' };
  if (v.type === 'list') return { kind: 'list', options: (v.list ?? []).map((l) => ({ value: l.val as string | number, label: plain(l.label) })) };
  return { kind: 'number' };
};

export type ConfigRow2 = CompareFixture['config'][number];

// src/Classes/CompareTab.lua:1244-1294, 2174-2195 (PoB-PoE2): each section's differing options, then the ones shown
// alike, "Custom Modifiers" left out; an input reads as configTab.input does, with the values CreateConfigSet seeds
export const configRows2 = (aInputs: Record<string, Value>, bInputs: Record<string, Value>, configToggle: boolean): ConfigRow2[] => {
  const seeded = (inputs: Record<string, Value>) =>
    Object.fromEntries(configurations2.flatMap((s) => s.rows.filter(isVariable2).map((v) => [v.var, currentValue(v, inputs)])));
  const [a, b] = [seeded(aInputs), seeded(bInputs)];
  const out: ConfigRow2[] = [];
  for (const section of configurations2) {
    if (section.name === 'Custom Modifiers') continue;
    const diffs: ConfigRow2[] = [];
    const commons: ConfigRow2[] = [];
    for (const v of section.rows) {
      if (!isVariable2(v)) continue;
      const [p, c] = normalizeConfigVals2(v, a[v.var], b[v.var]);
      const diff = String(p) !== String(c);
      const vis = visibility2(v);
      if (!diff && !(vis.alwaysShow || (configToggle && vis.showWithToggle))) continue;
      const row: ConfigRow2 = {
        key: v.var,
        section: section.name,
        label: v.label,
        a: formatConfigValue2(v, a[v.var]),
        b: formatConfigValue2(v, b[v.var]),
        diff,
        impactA: [0, 0, 0],
        impactB: [0, 0, 0],
        ...configControl2(v),
        aValue: a[v.var],
        bValue: b[v.var]
      };
      (diff ? diffs : commons).push(row);
    }
    out.push(...diffs, ...commons);
  }
  return out;
};

const ATTR_BY_CODE: Record<string, GemAttr> = { '^xE05030': 'str', '^x70FF70': 'dex', '^x7070FF': 'int' };

// the active skill set's groups as the shared skills view reads them
export const skillsFixture2 = (p: Poe2CompareParts): SkillsFixture => {
  const active = Math.max(
    0,
    p.skillSets.sets.findIndex((s) => s.id === p.skillSets.active)
  );
  const groups = p.groups.map((g) => ({
    label: g.label,
    slot: g.slot,
    enabled: g.enabled,
    inFull: g.includeInFullDPS,
    count: g.groupCount,
    sockets: '',
    set1: g.set1,
    set2: g.set2,
    gems: g.gems.map((m) => ({
      name: m.name,
      level: m.level,
      quality: m.quality,
      enabled: m.enabled,
      support: m.support,
      attr: ATTR_BY_CODE[m.color] ?? 'white',
      count: m.count
    }))
  }));
  const sets = p.skillSets.sets.length ? p.skillSets.sets : [{ id: 1, title: 'Default' }];
  return { activeSet: active, sets: sets.map((s, i) => ({ title: s.title || 'Default', groups: i === active ? groups : [] })) };
};

const RARITY: Record<string, NonNullable<CompareItemSide>['rarity']> = { NORMAL: 'normal', MAGIC: 'magic', RARE: 'rare', UNIQUE: 'unique', RELIC: 'relic' };

// an item's title lines: a named item's name, then its base
const itemSide = (it: Pick<Poe2Item, 'name' | 'rarity' | 'lines'> | undefined): CompareItemSide => {
  if (!it) return null;
  const [name, base] = it.name.map(plain);
  return {
    name: name ?? '',
    rarity: RARITY[it.rarity] ?? 'normal',
    line: plain(it.lines.find((l) => l.length)?.[0] ?? base ?? name ?? ''),
    base: base ?? name
  };
};

// src/Classes/CompareTab.lua:3856-3862 (PoB-PoE2): every base slot, both empty included
export const itemRows2 = (rows: Poe2CompareItemRow[]) => rows.map((r) => ({ slot: r.slot, a: itemSide(r.a), b: itemSide(r.b) }));

// src/Classes/CompareTab.lua:3894-3907 (PoB-PoE2)
export const jewelRows2 = (rows: Poe2CompareItemRow[]) =>
  rows.map((r) => ({ slot: r.slot, label: r.label, node: r.node, a: itemSide(r.a), b: itemSide(r.b), aWarn: r.aWarn, bWarn: r.bWarn }));

// src/Classes/CompareTab.lua:3921-3930 (PoB-PoE2): the hovered item's tooltip, its title then each group of mod lines
export const itemTip2 = (it: Pick<Poe2Item, 'name' | 'lines'>): ItemTip => ({
  influence: [],
  title: it.name.map(plain),
  lines: it.lines.flatMap((g) => [{ text: '', sep: true }, ...g.map((text) => ({ text, sep: false }))])
});

// the row's item on one side whose shown name is name
export const rowItem2 = (rows: Poe2CompareItemRow[], side: 'a' | 'b', name: string) =>
  rows.map((r) => r[side]).find((it) => it && plain(it.name[0] ?? '') === name);

const KIND: Record<string, string> = { Keystone: 'k', Notable: 'n', Mastery: 'm', Socket: 'j' };

// the nodes either build allocates, placed in the box their non-ascendancy nodes span, and the links between them
export const treeOverlay2 = (tree: Tree2, aNodes: number[], bNodes: number[]): CompareFixture['tree'] => {
  const byId = new Map(tree.nodes.map((n) => [n.id, n]));
  const inA = new Set(aNodes);
  const inB = new Set(bNodes);
  const placed = [...new Set([...aNodes, ...bNodes])]
    .sort((x, y) => x - y)
    .map((id) => byId.get(id))
    .filter((n) => n !== undefined);
  const pts = placed.filter((n) => !n.ascendancy);
  const [x0, x1] = [Math.min(...pts.map((n) => n.x)), Math.max(...pts.map((n) => n.x))];
  const [y0, y1] = [Math.min(...pts.map((n) => n.y)), Math.max(...pts.map((n) => n.y))];
  const span = Math.max(x1 - x0, y1 - y0) || 1;
  const nodes = placed.map((n) => ({
    id: n.id,
    name: n.name || String(n.id),
    kind: n.classesStart?.length ? 'c' : n.ascendancy ? 'a' : (KIND[n.type] ?? 's'),
    owner: inA.has(n.id) && inB.has(n.id) ? ('both' as const) : inA.has(n.id) ? ('a' as const) : ('b' as const),
    x: n.ascendancy ? null : (n.x - x0) / span,
    y: n.ascendancy ? null : (n.y - y0) / span
  }));
  const index = new Map(nodes.map((n, i) => [n.id, i]));
  const edges: [number, number][] = [];
  nodes.forEach((e, i) => {
    if (e.x === null) return;
    for (const c of byId.get(e.id)?.connections ?? []) {
      const j = index.get(c.id);
      if (j !== undefined && j > i && nodes[j].x !== null) edges.push([i, j]);
    }
  });
  return { nodes, edges };
};

// src/Classes/CompareTab.lua:279-389 (PoB-PoE2): the compared build's sets and main skill, 0-based
export const compareSets2 = (b: Poe2CompareParts & { view: Poe2Calc }): CompareSets => {
  const v = b.view;
  const ids = <T extends { id: number }>(sets: T[], active: number) =>
    Math.max(
      0,
      sets.findIndex((s) => s.id === active)
    );
  const group = v.socketGroups[(v.mainSocketGroup || 1) - 1];
  return {
    tree: { labels: v.treeSpecs?.labels ?? [], active: (v.treeSpecs?.active ?? 1) - 1 },
    skill: { labels: b.skillSets.sets.map((s) => s.title || 'Default'), active: ids(b.skillSets.sets, b.skillSets.active) },
    item: { labels: b.itemSets.sets.map((s) => s.title || 'Default'), active: ids(b.itemSets.sets, b.itemSets.active) },
    config: { labels: (v.configSets?.sets ?? []).map((s) => s.title || 'Default'), active: ids(v.configSets?.sets ?? [], v.configSets?.active ?? 1) },
    group: { labels: v.socketGroups.map((g) => g.label), active: (v.mainSocketGroup || 1) - 1 },
    skills: { labels: group?.skills ?? [], active: (v.mainActiveSkill || 1) - 1, enabled: (group?.skills.length ?? 0) > 1 }
  };
};

export type Poe2CompareSides = { a: Poe2CompareParts; b: Poe2CompareParts & { view: Poe2Calc }; bName: string; seq: number };

// the shared page's fixture for the open build a against the compared build
export const compareFixture2 = (
  aView: Poe2Calc,
  sides: Poe2CompareSides,
  aName: string,
  metrics: CompareMetric[],
  configToggle: boolean,
  tree: Tree2 | undefined,
  items: { slots: Poe2CompareItemRow[]; jewels: Poe2CompareItemRow[] } = { slots: [], jewels: [] }
): { fixture: CompareFixture; aSkills: SkillsFixture } => {
  const bView = sides.b.view;
  const stats = summaryStats2(aView, bView);
  const configSet = (v: Poe2Calc) => v.configSets?.sets.find((s) => s.id === v.configSets?.active)?.title || 'Default';
  const side = (v: Poe2Calc, name: string, k: 'a' | 'b') => ({
    name,
    className: v.className,
    ascendClassName: v.ascendClassName,
    level: v.level,
    points: v.tree.used,
    metrics: metricsOf(sideOutputs2(v) as never, metrics),
    stats: stats[k],
    configSet: configSet(v),
    power: []
  });
  return {
    fixture: {
      a: side(aView, aName, 'a'),
      b: { ...side(bView, sides.bName, 'b'), skills: skillsFixture2(sides.b) },
      items: itemRows2(items.slots),
      itemsInOrder: true,
      jewels: jewelRows2(items.jewels),
      tree: tree ? treeOverlay2(tree, aView.tree.alloc, bView.tree.alloc) : { nodes: [], edges: [] },
      config: configRows2(aView.configInputs, bView.configInputs, configToggle)
    },
    aSkills: skillsFixture2(sides.a)
  };
};

// one build's Calcs page values as calcs2 reads an actor, as the Calcs page builds it; CompareTab.lua:4584-4586
// (PoB-PoE2) reads each build's calcsEnv
const calcsActor2 = (page: Poe2CalcsPage): CalcsActor2 => {
  const output: CalcsActor2['output'] = { ...page.output, ...page.flags, ...page.strings };
  if (page.mainHand) output.MainHand = page.mainHand;
  if (page.offHand) output.OffHand = page.offHand;
  return { output, skillFlags: page.skillFlags, skillData: page.skillData, mods: page.mods, breakdowns: page.breakdowns };
};

// src/Classes/CompareTab.lua:4538-4581, 4620-4652 (PoB-PoE2): each section A's flags show, with every row either
// build shows; onlyDifferences leaves out a row whose value, mod rows and breakdown match
export const compareCalcs2 = async (
  a: { view: Poe2Calc; page: Poe2CalcsPage },
  b: { view: Poe2Calc; page: Poe2CalcsPage },
  onlyDifferences: boolean,
  modsEqual: (sds: calculator2.ModSectionData[]) => Promise<boolean[]> = (sds) => Promise.resolve(sds.map(() => true)),
  names: [string, string] = ['A', 'B']
): Promise<CompareCalcSection[]> => {
  const [{ checkFlag2, formatCalcStr2, modSectionData2 }, { calcSections2 }, { cellId, elementsOf }] = await Promise.all([
    import('$lib/display/calcs2'),
    import('$lib/calcs/calc_sections2'),
    import('$lib/display/calcs')
  ]);
  const [pa, ca] = [calcsActor2(a.page), calcsActor2(b.page)];
  const lines = (actor: CalcsActor2, bd: string) => actor.breakdowns?.[bd]?.lines ?? [];
  const sameLines = (bd: string) => {
    const [x, y] = [lines(pa, bd), lines(ca, bd)];
    return x.length === y.length && x.every((l, i) => l === y[i]);
  };
  const modSection = modSectionData2;
  // CompareTab.lua:4545-4561: each mod entry's rows matched between the builds, asked once for the page
  const sds = onlyDifferences
    ? calcSections2.flatMap(([, , , , subs]) =>
        subs.flatMap((sub) =>
          elementsOf<CalcSectionRow2>(sub.data).flatMap((row) =>
            row.label && row[0]?.format
              ? elementsOf<CalcSectionEntry2>(row[0])
                  .filter((e) => e.modName)
                  .map(modSection)
              : []
          )
        )
      )
    : [];
  const equal = new Map((await modsEqual(sds)).map((eq, i) => [JSON.stringify(sds[i]), eq]));
  const sameMods = (e: CalcSectionEntry2) => !e.modName || equal.get(JSON.stringify(modSection(e))) !== false;
  const view: CompareCalcSection[] = [];
  for (const [, id, , color, subs] of calcSections2) {
    const first = subs[0]?.data;
    if (!first || !checkFlag2({ flag: first.flag, notFlag: first.notFlag }, pa)) continue;
    const kept: CompareCalcSection['blocks'] = [];
    subs.forEach((sub, j) => {
      const rows: CompareCalcSection['blocks'][number]['rows'] = [];
      elementsOf<CalcSectionRow2>(sub.data).forEach((row, k) => {
        const col = row[0];
        if (!row.label || !col?.format) return;
        const [shownA, shownB] = [checkFlag2(row, pa), checkFlag2(row, ca)];
        if (!shownA && !shownB) return;
        const cid = cellId(id, j, k, 0);
        const [va, vb] = [formatCalcStr2(col.format, pa, col, cid), formatCalcStr2(col.format, ca, col, cid)];
        const entries = elementsOf<CalcSectionEntry2>(col);
        if (onlyDifferences && shownA && shownB && va === vb && entries.every((e) => sameMods(e) && (!e.breakdown || sameLines(e.breakdown)))) return;
        // CompareCalcsHelpers.lua:179-227 (PoB-PoE2): each build's breakdown lines, then the entries' mod rows
        const tip = [row.label];
        for (const e of entries) {
          if (!e.breakdown) continue;
          const [la, lb] = [lines(pa, e.breakdown), lines(ca, e.breakdown)];
          if (la.length) tip.push(`${names[0]}:`, ...la.map((l) => `  ${l}`));
          if (lb.length) tip.push(`${names[1]}:`, ...lb.map((l) => `  ${l}`));
        }
        const mods = entries.filter((e) => e.modName).map((e) => ({ label: e.label ?? '', sd: modSection(e) }));
        rows.push({ label: row.label, a: va, b: vb, ...(tip.length > 1 ? { tip } : {}), ...(mods.length ? { mods } : {}) });
      });
      const extra = sub.data.extra ? { a: formatCalcStr2(sub.data.extra, pa), b: formatCalcStr2(sub.data.extra, ca) } : undefined;
      if (rows.length || (onlyDifferences && extra && extra.a !== extra.b)) kept.push(extra ? { label: sub.label, extra, rows } : { label: sub.label, rows });
    });
    if (kept.length) view.push({ id, color, blocks: kept });
  }
  return view;
};

export type Poe2Power = { receiver: 'a' | 'b'; rows: ComparePowerRow[]; configBack: Record<string, number[]> };

// src/Classes/CompareTab.lua:2571-3126 (PoB-PoE2): the report's rows as the shared power view reads them; an item row
// is toned by the giver's item and names the receiver's
export const powerRows2 = (
  entries: calculator2.ComparePowerEntry[],
  sides: Pick<Poe2CompareSides, 'a' | 'b'>,
  config: ConfigRow2[],
  receiver: 'a' | 'b',
  metrics: CompareMetric[]
): ComparePowerRow[] => {
  const items = (p: Poe2CompareParts) => p.items.map((i) => ({ slot: i.slot, name: plain(i.name[0] ?? ''), rarity: i.rarity }));
  const giver = receiver === 'a' ? sides.b : sides.a;
  return powerRows(entries, items(giver), items(sides[receiver]), config, receiver, metrics, LISTS2.player);
};

// the receiver's power rows, and each config row's impact into either build: the receiver's from its rows, the other's
// from the config-only report back
export const withPower2 = (f: CompareFixture, p: Poe2Power | undefined): CompareFixture => {
  if (!p) return f;
  const fromRows = Object.fromEntries(p.rows.filter((x) => x.cat === 'config').map((x): [string, number[]] => [x.key ?? '', x.impact]));
  const into = (r: 'a' | 'b') => (p.receiver === r ? fromRows : p.configBack);
  return {
    ...f,
    a: { ...f.a, power: p.receiver === 'a' ? p.rows : [] },
    b: { ...f.b, power: p.receiver === 'b' ? p.rows : [] },
    config: f.config.map((c) => ({ ...c, impactA: into('a')[c.key] ?? c.impactA, impactB: into('b')[c.key] ?? c.impactB }))
  };
};
