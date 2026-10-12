const BREAKDOWN_ROWS = 12;
const CONFIG_IMPACTS = 10;

export const MOD_SAMPLES = [
  '10% increased Damage',
  '+50 to maximum Life',
  'Minions deal 15% more Damage',
  'Nearby Enemies take 10% increased Physical Damage',
  '80% increased global Accuracy Rating',
  'Crit Immune',
  'Totally not a modifier',
  '# boss uptime assumptions'
];

const settle = async (worker) => {
  while (worker.ticksInFlight > 0) await new Promise((r) => setTimeout(r, 5));
};

export const verdictOf = (m, line) => ({
  k: { parsed: 'ok', unsupported: 'bad', comment: 'comment' }[line?.Status] ?? 'empty',
  parsed: line ? m.custom.modSummary(line) : ''
});

export function configFixture(m, worker, out) {
  const cfg = worker.currentBuild.Config;
  const blocks = worker.GetCustomModifierBlocks();
  const custom = blocks.map((b) => {
    const cells = m.custom.textToCells(b.Text ?? '').cells.filter((c) => c.trim() !== '');
    const verdicts = cells.length ? worker.ParseModLines(cells) : [];
    return {
      title: b.Title ?? 'Default',
      enabled: b.Enabled === undefined || b.Enabled === 'true',
      cells: cells.map((text, i) => ({ text, ...verdictOf(m, verdicts[i]) }))
    };
  });
  const sort = (o) => Object.fromEntries(Object.entries(o).sort(([a], [b]) => a.localeCompare(b)));
  return {
    sets: (cfg.ConfigSets ?? []).map((s, i) => s.Title || (i === 0 ? 'Default' : `Config set ${s.ID}`)),
    activeSet: worker.currentBuild.ActiveConfigSetIndex(),
    inputs: sort(worker.GetAllConfigOptions()),
    placeholders: sort(worker.GetConfigPlaceholders()),
    relevance: out.ConfigRelevance,
    custom
  };
}

export async function calcsFixture(m, worker, getLast) {
  const { calcSections } = m.sections;
  worker.setCalcTabElements(m.calcs.calcElements(calcSections));
  await worker.Tick('design-fixtures: calcs');
  await settle(worker);
  const out = getLast()?.CalcsTab;
  worker.clearCalcTabElements();
  if (!out?.Breakdown) throw new Error('the calcs Tick carried no CalcsTab Breakdown');
  return m.calcs.calcsView(calcSections, out, BREAKDOWN_ROWS);
}

// Build.lua:1689-1777
function statRows(m, output, flags, list, actor) {
  const groups = [];
  let group = [];
  for (const s of list) {
    if ((s.flag && flags[s.flag] !== true) || (s.notFlag && flags[s.notFlag] === true)) continue;
    if (!s.stat) {
      if (group.length) groups.push(group);
      group = [];
      continue;
    }
    if (s.stat === 'SkillDPS' || s.childStat) continue;
    const v = output[s.stat];
    if (v === undefined || !isFinite(v)) continue;
    const shown = s.condFunc ? s.condFunc(v, output) : v !== 0;
    if (!shown || s.hideStat || group.some((r) => r.key === `${actor}:${s.stat}`)) continue;
    group.push({ key: `${actor}:${s.stat}`, label: s.label, v, text: m.fmtStat.formatStat(s, v).value, lower: !!s.lowerIsBetter, pc: !!s.pc });
  }
  if (group.length) groups.push(group);
  return groups;
}

const metricsOf = (out) => ({
  dps: out.Output?.FullDPS || out.Output?.CombinedDPS || out.Minion?.Output?.CombinedDPS || 0,
  ehp: out.Output?.TotalEHP ?? 0,
  life: out.Output?.Life ?? 0
});

let treeCache = new Map();
export async function treeOf(worker) {
  const v = worker.GetTreeVersion();
  if (!treeCache.has(v)) treeCache.set(v, JSON.parse(await worker.GetTree(v)));
  return treeCache.get(v);
}

export const kindOf = (n) =>
  n.classStartIndex !== undefined ? 'c' : n.isKeystone ? 'k' : n.isMastery ? 'm' : n.isJewelSocket ? 'j' : n.isNotable ? 'n' : n.ascendancyName ? 'a' : 's';

async function sideOf(m, worker, xmlName, out, itemsFx) {
  const info = worker.GetBuildInfo();
  const cfg = worker.currentBuild.Config;
  const set = itemsFx.sets.find((s) => s.id === itemsFx.activeSet) ?? itemsFx.sets[0];
  return {
    name: xmlName.replace(/\.xml$/, ''),
    className: info.className,
    ascendClassName: info.ascendClassName,
    level: info.level,
    points: out.PointsUsed?.[0] ?? 0,
    metrics: metricsOf(out),
    stats: [
      ...(out.Minion?.Output ? statRows(m, out.Minion.Output, out.Minion.SkillFlags ?? {}, m.stats.minionDisplayStats, 'Minion') : []),
      ...statRows(m, out.Output ?? {}, out.SkillFlags ?? {}, m.stats.displayStats, 'Player')
    ],
    nodes: worker.GetPassiveNodes(),
    slots: Object.fromEntries((set?.slots ?? []).map((x) => [x.slot, x.item])),
    configSet: (cfg.ConfigSets ?? [])[worker.currentBuild.ActiveConfigSetIndex()]?.Title || 'Default',
    inputs: worker.GetAllConfigOptions()
  };
}

const pobPlainText = (t) => t.replace(/\^x[0-9A-Fa-f]{6}|\^#[0-9A-Fa-f]{6}|\^\d/g, '');

const optionValue = (v, value) => {
  if (value === undefined || value === null) value = m_default(v);
  if (v.type === 'list') return String(v.list.find((l) => l.value === value)?.label ?? value ?? '').replace(/\^x[0-9A-F]{6}|\^\d/g, '');
  if (v.type === 'check') return value ? 'yes' : 'no';
  return value === undefined || value === null ? '' : String(value);
};
let m_default;

async function configImpact(m, worker, getLast, key, value, back) {
  const base = metricsOf(getLast());
  const v = m.cfg.reverseConfigOptions[key];
  worker.SetConfigOption(key, value ?? (v.type === 'check' ? false : v.type === 'list' ? m.cfg.defaultValue(v) : null));
  await settle(worker);
  const next = metricsOf(getLast());
  worker.SetConfigOption(key, back ?? (v.type === 'check' ? false : v.type === 'list' ? m.cfg.defaultValue(v) : null));
  await settle(worker);
  return [next.dps - base.dps, next.ehp - base.ehp, next.life - base.life].map((d) => Math.round(d * 10) / 10);
}

const round1 = (d) => Math.round(d * 10) / 10;
const rawMetrics = (so) => ({
  dps: so.Output?.FullDPS || so.Output?.CombinedDPS || so.Minion?.CombinedDPS || 0,
  ehp: so.Output?.TotalEHP ?? 0,
  life: so.Output?.Life ?? 0
});
const deltaOf = (next, base) => [next.dps - base.dps, next.ehp - base.ehp, next.life - base.life].map(round1);

const slotsOf = (itemsFx) => {
  const set = itemsFx.sets.find((s) => s.id === itemsFx.activeSet) ?? itemsFx.sets[0];
  return Object.fromEntries((set?.slots ?? []).map((x) => [x.slot, x.item]));
};

const POWER_NODES = 10;

async function powerRows(worker, tree, { items, take, drop, config }) {
  const mc = await worker.miscCalculator();
  const base = rawMetrics(mc.base);
  const rows = [];
  for (const it of items) {
    const slots = (await mc.calc.CompareItem(it.text)).unwrap() ?? [];
    const s = slots.find((x) => x.SlotName === it.slot);
    if (s) {
      const detail = it.replacing ? `replaces ${it.replacing}` : 'slot empty here';
      rows.push({ cat: 'items', what: `${it.slot}: ${it.name}`, detail, tone: it.rarity, impact: deltaOf(rawMetrics(s.Output), base) });
    }
  }
  const node = async (id, add) => {
    const n = tree.nodes[id];
    const out = await mc.calc.CalcNodes(add ? [id] : [], add ? [] : [id]);
    const kind = n.isKeystone ? 'keystone' : 'notable';
    const detail = `${kind} only in ${add ? 'the other build' : 'this build'}`;
    rows.push({ cat: 'tree', what: `${add ? 'Take' : 'Drop'} ${n.name}`, detail, tone: 'text', points: 1, impact: deltaOf(rawMetrics(out), base) });
  };
  for (const id of take.slice(0, POWER_NODES)) await node(id, true);
  for (const id of drop.slice(0, POWER_NODES)) await node(id, false);
  return [...rows, ...config];
}

export async function compareFixture(m, worker, getLast, a, b, { itemsOf, skillsOf }) {
  m_default = m.cfg.defaultValue;
  const tree = await treeOf(worker);
  const offsets = m.geo.ascendancyOffsets(tree);
  const aNodes = worker.GetPassiveNodes();
  const aSlots = slotsOf(a.items);

  worker.currentBuild = b.build;
  await worker.Tick('design-fixtures: compare B');
  await settle(worker);
  b.items = await itemsOf(worker);
  b.texts = Object.fromEntries((worker.GetItems().Items ?? []).map((it) => [it.ID, it.Text]));
  const B = await sideOf(m, worker, b.file, getLast(), b.items);
  B.skills = skillsOf(b.build);
  const cfgOf = (k) => m.cfg.reverseConfigOptions[k];
  const diffKeys = [...new Set([...Object.keys(a.inputs), ...Object.keys(B.inputs)])]
    .filter((k) => cfgOf(k) && optionValue(cfgOf(k), a.inputs[k]) !== optionValue(cfgOf(k), B.inputs[k]))
    .sort()
    .slice(0, CONFIG_IMPACTS);
  const cfgRow = (k, from, to) => {
    const v = cfgOf(k);
    const section = m.cfg.configurations.find((s) => s.variables.includes(v))?.name ?? '';
    return { key: k, section, label: v.label, from: optionValue(v, from), to: optionValue(v, to) };
  };
  const impactsB = {};
  for (const k of diffKeys) impactsB[k] = await configImpact(m, worker, getLast, k, a.inputs[k], B.inputs[k]);

  const slotNames = [...new Set([...Object.keys(aSlots), ...Object.keys(B.slots)])];
  const itemOf = (side, slot) => (side === 'a' ? a.items.items[aSlots[slot]] : b.items.items[B.slots[slot]]);
  const differs = (slot) => itemOf('a', slot)?.name !== itemOf('b', slot)?.name;
  const inA = new Set(aNodes);
  const inB = new Set(B.nodes);
  const notable = (id) => ['n', 'k'].includes(kindOf(tree.nodes[id] ?? {})) && !tree.nodes[id]?.ascendancyName;
  const aOnly = aNodes.filter((id) => !inB.has(id) && notable(id)).sort((x, y) => x - y);
  const bOnly = B.nodes.filter((id) => !inA.has(id) && notable(id)).sort((x, y) => x - y);
  const configPower = (impacts, receiving) =>
    diffKeys.map((k) => {
      const r = receiving === 'a' ? cfgRow(k, a.inputs[k], B.inputs[k]) : cfgRow(k, B.inputs[k], a.inputs[k]);
      return {
        cat: 'config',
        what: `${pobPlainText(r.label)} → ${r.to}`,
        detail: `${receiving === 'a' ? 'A' : 'B'}: ${r.from}`,
        tone: 'text',
        impact: impacts[k]
      };
    });
  const itemsInto = (receiving) => {
    const giver = receiving === 'a' ? 'b' : 'a';
    return slotNames
      .filter((slot) => differs(slot) && itemOf(giver, slot))
      .map((slot) => {
        const give = itemOf(giver, slot);
        const text = receiving === 'a' ? b.texts[B.slots[slot]] : a.texts[aSlots[slot]];
        return { slot, text, name: give.name, rarity: give.rarity, replacing: itemOf(receiving, slot)?.name };
      });
  };
  const itemImpact = async (text, slot) => {
    const r = await worker.ItemCompare(text);
    const blk = r?.blocks.find((x) => x.slot === slot);
    return blk?.sections.map((s) => ({ actor: s.actor, lines: s.lines.slice(0, 3) }));
  };

  await worker.PrepareStatCompare();
  const itemsInB = {};
  for (const it of itemsInto('b')) itemsInB[it.slot] = await itemImpact(it.text, it.slot);
  const powerB = await powerRows(worker, tree, { items: itemsInto('b'), take: aOnly, drop: bOnly, config: configPower(impactsB, 'b') });

  worker.currentBuild = a.build;
  await worker.Tick('design-fixtures: compare A');
  await settle(worker);
  const A = await sideOf(m, worker, a.file, getLast(), a.items);
  const impactsA = {};
  for (const k of diffKeys) impactsA[k] = await configImpact(m, worker, getLast, k, B.inputs[k], a.inputs[k]);
  await worker.PrepareStatCompare();
  const itemsInA = {};
  for (const it of itemsInto('a')) itemsInA[it.slot] = await itemImpact(it.text, it.slot);
  const powerA = await powerRows(worker, tree, { items: itemsInto('a'), take: bOnly, drop: aOnly, config: configPower(impactsA, 'a') });

  const brief = (it) => (it ? { name: it.name, rarity: it.rarity, line: it.explicits[0]?.text ?? it.base } : null);
  const items = slotNames.map((slot) => ({ slot, a: brief(itemOf('a', slot)), b: brief(itemOf('b', slot)), impactA: itemsInA[slot], impactB: itemsInB[slot] }));

  const ids = [...new Set([...A.nodes, ...B.nodes])].sort((x, y) => x - y);
  const placed = ids
    .map((id) => {
      const n = tree.nodes[id];
      const p = n && !n.ascendancyName ? m.geo.nodeCentre(tree, offsets, n) : undefined;
      return { id, n, p };
    })
    .filter((e) => e.n);
  const pts = placed.filter((e) => e.p);
  const [x0, x1] = [Math.min(...pts.map((e) => e.p.x)), Math.max(...pts.map((e) => e.p.x))];
  const [y0, y1] = [Math.min(...pts.map((e) => e.p.y)), Math.max(...pts.map((e) => e.p.y))];
  const span = Math.max(x1 - x0, y1 - y0) || 1;
  const nodes = placed.map(({ id, n, p }) => ({
    id,
    name: n.name ?? String(id),
    kind: kindOf(n),
    owner: inA.has(id) && inB.has(id) ? 'both' : inA.has(id) ? 'a' : 'b',
    x: p ? Math.round(((p.x - x0) / span) * 1000) / 1000 : null,
    y: p ? Math.round(((p.y - y0) / span) * 1000) / 1000 : null
  }));
  const index = new Map(nodes.map((n, i) => [n.id, i]));
  const edges = [];
  for (const [i, e] of nodes.entries()) {
    if (e.x === null) continue;
    for (const o of tree.nodes[e.id].out ?? []) {
      const j = index.get(Number(o));
      if (j !== undefined && nodes[j].x !== null) edges.push([i, j]);
    }
  }

  const config = diffKeys.map((k) => {
    const r = cfgRow(k, a.inputs[k], B.inputs[k]);
    return { key: k, section: r.section, label: r.label, a: r.from, b: r.to, impactA: impactsA[k], impactB: impactsB[k] };
  });

  const strip = ({ inputs: _i, slots: _s, nodes: _n, ...rest }) => rest;
  return { a: { ...strip(A), power: powerA }, b: { ...strip(B), power: powerB }, items, tree: { nodes, edges }, config };
}
