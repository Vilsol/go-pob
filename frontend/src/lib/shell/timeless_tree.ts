import { calculateAllocationPath, type PrecalculatedAllocationPaths } from '$lib/components/skill-tree/paths';
import type { TreeFixture, TreeNodeFx } from '$lib/design/fixtures';
import type { TimelessSocket } from '$lib/design/timeless';
import { ascendancyOffsets, nodeCentre } from '$lib/skill_tree/geometry';
import type { Node, Tree } from '$lib/skill_tree/types';
import type { calculator } from '$lib/types';

// TreeTab.lua:1295-1336

// Data.lua:609-613
const LARGE_RADIUS = 1800;
const ASPECT = 1.3;
const MIN_HEIGHT = 3400;
const MIN_SOCKETS = 3;

const round = (v: number) => Math.round(v * 1000) / 1000;
const kindOf = (n: Node): TreeNodeFx['k'] =>
  n.classStartIndex !== undefined ? 'c' : n.isKeystone ? 'k' : n.isMastery ? 'm' : n.isJewelSocket ? 'j' : n.isNotable ? 'n' : 's';

export function timelessTree(
  version: string,
  tree: Tree,
  active: number[],
  offered: calculator.TimelessSocket[],
  paths: PrecalculatedAllocationPaths,
  jewels: Record<number, string> = {}
): { tree: TreeFixture; sockets: TimelessSocket[] } {
  const offsets = ascendancyOffsets(tree);
  const allocated = new Set(active);
  const main = Object.values(tree.nodes).filter((n) => n.skill !== undefined && !n.ascendancyName && !n.isProxy && n.group !== undefined);
  const at = new Map(main.flatMap((n) => (nodeCentre(tree, offsets, n) ? [[n.skill!, nodeCentre(tree, offsets, n)!] as const] : [])));
  const mine = active.flatMap((id) => at.get(id) ?? []);
  const cx = mine.length ? mine.reduce((s, p) => s + p.x, 0) / mine.length : 0;
  const cy = mine.length ? mine.reduce((s, p) => s + p.y, 0) / mine.length : 0;

  let h = MIN_HEIGHT;
  const inBox = (id: number, w: number, hh: number) => {
    const p = at.get(id);
    return !!p && Math.abs(p.x - cx) <= w / 2 && Math.abs(p.y - cy) <= hh / 2;
  };
  const socketsIn = (hh: number) => offered.filter((s) => s.ID !== -1 && inBox(s.ID, hh * ASPECT, hh));
  while (socketsIn(h).length < MIN_SOCKETS && h < 20000) h += 400;
  const w = h * ASPECT;
  const x0 = cx - w / 2;
  const y0 = cy - h / 2;
  const view = main.filter((n) => inBox(n.skill!, w, h)).sort((a, b) => a.skill! - b.skill!);
  const nodes: TreeNodeFx[] = view.map((n) => {
    const p = at.get(n.skill!)!;
    return {
      id: n.skill!,
      k: kindOf(n),
      x: round((p.x - x0) / w),
      y: round((p.y - y0) / h),
      name: n.name ?? '',
      ...(allocated.has(n.skill!) ? { a: 1 as const } : {})
    };
  });
  const index = new Map(nodes.map((n, i) => [n.id, i]));
  const edges: [number, number][] = [];
  for (const n of view) {
    for (const o of n.out ?? []) {
      const j = index.get(Number(o));
      if (j !== undefined && tree.nodes[o]?.classStartIndex === undefined && n.classStartIndex === undefined) edges.push([index.get(n.skill!)!, j]);
    }
  }

  const sockets = offered
    .filter((s) => s.ID !== -1 && at.has(s.ID))
    .map((s) => {
      const c = at.get(s.ID)!;
      const d2 = (id: number) => (at.get(id)!.x - c.x) ** 2 + (at.get(id)!.y - c.y) ** 2;
      const near = main.filter((n) => n.skill !== s.ID && at.has(n.skill!) && d2(n.skill!) <= LARGE_RADIUS ** 2);
      return {
        id: s.ID,
        label: s.Label,
        pts: allocated.has(s.ID) ? 0 : (calculateAllocationPath(paths, s.ID)?.length ?? -1),
        notables: near.filter((n) => kindOf(n) === 'n').map((n) => n.name ?? ''),
        keystones: near.filter((n) => kindOf(n) === 'k').map((n) => n.name ?? ''),
        smalls: near.filter((n) => kindOf(n) === 's').length,
        ...(jewels[s.ID] ? { jewel: jewels[s.ID] } : {})
      };
    });

  return {
    tree: {
      version,
      classes: [],
      ascendancies: [],
      specs: [],
      activeSpec: 0,
      aspect: ASPECT,
      radius: round(LARGE_RADIUS / w),
      nodes,
      edges,
      tips: {},
      sockets: sockets.filter((s) => index.has(s.id))
    },
    sockets
  };
}
