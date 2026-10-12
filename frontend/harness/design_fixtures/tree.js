import { treeOf, kindOf } from './pages.js';

// Data.lua:609-613
const LARGE_RADIUS = 1800;
const ASPECT = 1.3;
const MIN_HEIGHT = 3400;
const MIN_SOCKETS = 3;
const TIPS = 10;
const TIP_LINES = 4;

const round = (v) => Math.round(v * 1000) / 1000;

export async function treeFixture(m, worker, itemsFx) {
  const tree = await treeOf(worker);
  const version = worker.GetTreeVersion();
  const build = worker.currentBuild;
  const info = worker.GetBuildInfo();
  const spec = build.Tree.Specs?.[build.Tree.ActiveSpec - 1] ?? build.Tree.Specs?.[0];
  const offsets = m.geo.ascendancyOffsets(tree);
  const active = worker.GetPassiveNodes();
  const allocated = new Set(active);

  const classIndex = tree.classes.findIndex((c) => c.name === info.className);
  const roots = Object.values(tree.nodes)
    .filter((n) => n.classStartIndex === classIndex)
    .map((n) => n.skill);
  const paths = worker.CalculateAllocationPaths(version, [...active], roots) ?? {};

  const main = Object.values(tree.nodes).filter((n) => n.skill !== undefined && !n.ascendancyName && !n.isProxy && n.group !== undefined);
  const at = new Map(main.map((n) => [n.skill, m.geo.nodeCentre(tree, offsets, n)]).filter(([, p]) => p));
  const mine = active.map((id) => at.get(id)).filter(Boolean);
  const cx = mine.reduce((s, p) => s + p.x, 0) / mine.length;
  const cy = mine.reduce((s, p) => s + p.y, 0) / mine.length;

  let h = MIN_HEIGHT;
  const inBox = (p, w, hh) => Math.abs(p.x - cx) <= w / 2 && Math.abs(p.y - cy) <= hh / 2;
  const socketsIn = (hh) => main.filter((n) => n.isJewelSocket && !n.expansionJewel && inBox(at.get(n.skill), hh * ASPECT, hh));
  while (socketsIn(h).length < MIN_SOCKETS && h < 20000) h += 400;
  const w = h * ASPECT;
  const x0 = cx - w / 2;
  const y0 = cy - h / 2;
  const view = main.filter((n) => at.has(n.skill) && inBox(at.get(n.skill), w, h)).sort((a, b) => a.skill - b.skill);
  const inView = new Set(view.map((n) => n.skill));

  const pathOf = (id) => m.paths.calculateAllocationPath(paths, id) ?? [];
  const loseOf = (id) => worker.CalculateDependentNodes(version, [...active], id, roots) ?? [];

  const chosen = Object.fromEntries([...(spec?.MasteryEffects ?? '').matchAll(/\{(\d+),(\d+)\}/g)].map((x) => [Number(x[1]), Number(x[2])]));

  const nodes = view.map((n) => {
    const p = at.get(n.skill);
    const k = kindOf(n);
    const alloc = allocated.has(n.skill);
    const e = { id: n.skill, k, x: round((p.x - x0) / w), y: round((p.y - y0) / h), name: n.name ?? '' };
    if (alloc) e.a = 1;
    if (n.stats?.length) e.stats = n.stats;
    if (n.recipe?.length) e.recipe = n.recipe;
    if (k === 'm' && alloc && n.masteryEffects?.length) {
      e.effects = n.masteryEffects.map((x) => x.stats);
      const pick = n.masteryEffects.findIndex((x) => x.effect === chosen[n.skill]);
      if (pick >= 0) e.effect = pick;
    }
    if (k !== 'c') {
      if (alloc) {
        const lose = loseOf(n.skill);
        e.lose = lose.length;
        const near = lose.filter((id) => inView.has(id));
        if (near.length) e.loseIds = near;
      } else {
        const path = pathOf(n.skill);
        e.pts = path.length;
        const hop = paths[n.skill];
        if (hop !== undefined && hop !== -1 && inView.has(hop)) e.next = hop;
      }
    }
    return e;
  });
  const index = new Map(nodes.map((n, i) => [n.id, i]));
  const edges = [];
  for (const n of view) {
    for (const o of n.out ?? []) {
      const j = index.get(Number(o));
      if (j !== undefined && !(tree.nodes[o]?.classStartIndex !== undefined || n.classStartIndex !== undefined)) edges.push([index.get(n.skill), j]);
    }
  }

  await worker.PrepareStatCompare();
  const sample = nodes.filter((n) => n.k === 'n' || n.k === 'k');
  const picks = [...sample.filter((n) => n.a).slice(0, TIPS / 2), ...sample.filter((n) => !n.a && n.pts).slice(0, TIPS / 2)];
  const cap = (r) => (r?.sections ?? []).map((s) => ({ actor: s.actor, lines: s.lines.slice(0, TIP_LINES) })).filter((s) => s.lines.length);
  const tips = {};
  for (const n of picks) {
    const blocks = [];
    if (n.a) {
      const one = cap(await worker.StatCompareNodes([], [n.id]));
      if (one.length) blocks.push({ header: 'Unallocating this node will give you:', sections: one });
    } else {
      const one = cap(await worker.StatCompareNodes([n.id], []));
      if (one.length) blocks.push({ header: 'Allocating this node will give you:', sections: one });
      if (n.pts > 1) {
        const all = cap(await worker.StatCompareNodes(pathOf(n.id), [], n.pts));
        if (all.length) blocks.push({ header: 'Allocating this node and all nodes leading to it will give you:', sections: all });
      }
    }
    tips[n.id] = blocks;
  }

  const socketed = Object.fromEntries((spec?.Sockets ?? []).filter((s) => s.ItemID > 0).map((s) => [s.NodeID, s.ItemID]));
  const sockets = socketsIn(h)
    .sort((a, b) => a.skill - b.skill)
    .map((s) => {
      const c = at.get(s.skill);
      const d2 = (n) => (at.get(n.skill).x - c.x) ** 2 + (at.get(n.skill).y - c.y) ** 2;
      // TreeTab.lua:1311-1322
      const near = main.filter((n) => n.skill !== s.skill && d2(n) <= LARGE_RADIUS ** 2).sort((a, b) => d2(a) - d2(b) || a.skill - b.skill);
      const kinds = near.map((n) => [n, kindOf(n)]);
      return {
        id: s.skill,
        pts: allocated.has(s.skill) ? 0 : pathOf(s.skill).length || -1,
        notables: kinds.filter(([, k]) => k === 'n').map(([n]) => n.name),
        keystones: kinds.filter(([, k]) => k === 'k').map(([n]) => n.name),
        smalls: kinds.filter(([, k]) => k === 's').length,
        ...(itemsFx.items[socketed[s.skill]] ? { jewel: itemsFx.items[socketed[s.skill]].name } : {})
      };
    });

  const ascendancies = (tree.classes[classIndex]?.ascendancies ?? []).map((a) => a.name);
  return {
    version,
    classes: tree.classes.map((c) => c.name),
    ascendancies,
    specs: (build.Tree.Specs ?? []).map((s, i) => s.Title || `Tree ${i + 1}`),
    activeSpec: Math.max(1, build.Tree.ActiveSpec ?? 1) - 1,
    aspect: ASPECT,
    radius: round(LARGE_RADIUS / w),
    nodes,
    edges,
    tips,
    sockets
  };
}
