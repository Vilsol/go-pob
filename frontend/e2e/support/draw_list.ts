import type { RecordedCall } from './tree_probe';
import type { TreeView } from '../../src/lib/components/skill-tree/probe';

export type SpriteItem = { k: 'sprite'; src: string; s: number[]; v: '' | 'crop' | 'crop-dim'; q: number[] };
export type LineItem = { k: 'line'; p: number[]; style: string; w: number };
export type ArcItem = { k: 'arc'; c: number[]; r: number; a: number[]; style: string; w: number };
export type Item = SpriteItem | LineItem | ArcItem;

export type DrawList = { view: TreeView; min: { x: number; y: number }; items: Item[] };

const TAU = Math.PI * 2;

export const assetKey = (url: string) => {
  const p = new URL(url).pathname;
  return p.replace(/^\/data\/[^/]+\//, '');
};

export const fromCanvasCalls = (calls: RecordedCall[], view: TreeView, min: { x: number; y: number }, dpr: number): DrawList => {
  const toTree = (t: number[], x: number, y: number) => {
    const cx = (t[0] * x + t[2] * y + t[4]) / dpr;
    const cy = (t[1] * x + t[3] * y + t[5]) / dpr;
    return [cx * view.scaling - Math.abs(min.x) - view.offsetX, cy * view.scaling - Math.abs(min.y) - view.offsetY];
  };
  const items: Item[] = [];
  let path: RecordedCall[] = [];
  for (const c of calls) {
    if (c.op === 'img') {
      let [dx, dy, dw, dh] = c.d;
      if (dw < 0) [dx, dw] = [dx + dw, -dw];
      if (dh < 0) [dy, dh] = [dy + dh, -dh];
      const o = toTree(c.t, dx, dy);
      const u = toTree(c.t, dx + dw, dy);
      const v = toTree(c.t, dx, dy + dh);
      const q = [o[0], o[1], u[0] - o[0], u[1] - o[1], v[0] - o[0], v[1] - o[1]];
      if (c.crop) {
        items.push({ k: 'sprite', src: assetKey(c.crop.src), s: c.crop.s, v: c.crop.filter.includes('brightness') ? 'crop-dim' : 'crop', q });
      } else {
        items.push({ k: 'sprite', src: assetKey(c.src), s: c.s, v: '', q });
      }
    } else if (c.op === 'begin') {
      path = [];
    } else if (c.op === 'stroke') {
      const w = ((c.width * Math.hypot(c.t[0], c.t[1])) / dpr) * view.scaling;
      let from: number[] | undefined;
      for (const p of path) {
        if (p.op === 'move') {
          from = toTree(p.t, p.p[0], p.p[1]);
        } else if (p.op === 'line' && from) {
          const to = toTree(p.t, p.p[0], p.p[1]);
          items.push({ k: 'line', p: [...from, ...to], style: c.style, w });
          from = to;
        } else if (p.op === 'arc') {
          const [x, y, r, a0, a1] = p.a;
          items.push({ k: 'arc', c: toTree(p.t, x, y), r: ((r * Math.hypot(p.t[0], p.t[1])) / dpr) * view.scaling, a: [a0, a1], style: c.style, w });
        }
      }
    } else {
      path.push(c);
    }
  }
  return { view, min, items };
};

const inView = (list: DrawList, x: number, y: number) => {
  const { view, min } = list;
  const sx = (Math.abs(min.x) + x + view.offsetX) / view.scaling;
  const sy = (Math.abs(min.y) + y + view.offsetY) / view.scaling;
  return sx >= 0 && sx <= view.width && sy >= 0 && sy <= view.height;
};

const arcEnds = (i: ArcItem) => i.a.map((a) => [i.c[0] + i.r * Math.cos(a), i.c[1] + i.r * Math.sin(a)]);

const visible = (list: DrawList) =>
  list.items
    .map((item, order) => ({ item, order }))
    .filter(({ item: i }) => {
      if (i.k === 'sprite') return inView(list, i.q[0] + (i.q[2] + i.q[4]) / 2, i.q[1] + (i.q[3] + i.q[5]) / 2);
      if (i.k === 'line') return inView(list, i.p[0], i.p[1]) || inView(list, i.p[2], i.p[3]);
      return arcEnds(i).some(([x, y]) => inView(list, x, y));
    });

const r2 = (n: number) => Math.round(n * 100) / 100;

const EPS = 0.05;

const keyOf = (i: Item) => {
  if (i.k === 'sprite') return `sprite ${i.src} [${i.s.join(',')}] ${i.v}`;
  return `${i.k} ${i.style} w=${r2(i.w)}`;
};

const geomOf = (i: Item): number[] => {
  if (i.k === 'sprite') return i.q;
  if (i.k === 'line') {
    const [a, b, c, d] = i.p;
    return (Math.abs(a - c) > EPS ? a < c : b <= d) ? [a, b, c, d] : [c, d, a, b];
  }
  const a0 = ((i.a[0] % TAU) + TAU) % TAU;
  const sweep = (((i.a[1] - i.a[0]) % TAU) + TAU) % TAU;
  return [i.c[0], i.c[1], i.r, a0 * 1000, sweep * 1000];
};

const close = (a: number[], b: number[]) => a.length === b.length && a.every((x, j) => Math.abs(x - b[j]) <= EPS);

const bbox = (i: Item): number[] => {
  if (i.k === 'sprite') {
    const xs = [i.q[0], i.q[0] + i.q[2], i.q[0] + i.q[4], i.q[0] + i.q[2] + i.q[4]];
    const ys = [i.q[1], i.q[1] + i.q[3], i.q[1] + i.q[5], i.q[1] + i.q[3] + i.q[5]];
    return [Math.min(...xs), Math.min(...ys), Math.max(...xs), Math.max(...ys)];
  }
  if (i.k === 'line') {
    return [Math.min(i.p[0], i.p[2]) - i.w / 2, Math.min(i.p[1], i.p[3]) - i.w / 2, Math.max(i.p[0], i.p[2]) + i.w / 2, Math.max(i.p[1], i.p[3]) + i.w / 2];
  }
  return [i.c[0] - i.r - i.w, i.c[1] - i.r - i.w, i.c[0] + i.r + i.w, i.c[1] + i.r + i.w];
};

export type Comparison = { compared: number; missing: string[]; extra: string[]; misordered: string[]; overlappingPairs: number };

const describe = (i: Item) =>
  `${keyOf(i)} @ ${geomOf(i)
    .map((n) => n.toFixed(1))
    .join(',')}`;

export const compareDrawLists = (want: DrawList, got: DrawList): Comparison => {
  const a = visible(want);
  const b = visible(got);
  const pool = new Map<string, typeof b>();
  for (const e of b) pool.set(keyOf(e.item), [...(pool.get(keyOf(e.item)) ?? []), e]);

  const missing: string[] = [];
  const pairs: { wantOrder: number; gotOrder: number; item: Item }[] = [];
  for (const e of a) {
    const cands = pool.get(keyOf(e.item)) ?? [];
    const g = geomOf(e.item);
    const j = cands.findIndex((c) => close(g, geomOf(c.item)));
    if (j < 0) {
      missing.push(describe(e.item));
      continue;
    }
    pairs.push({ wantOrder: e.order, gotOrder: cands[j].order, item: e.item });
    cands.splice(j, 1);
  }
  const extra = [...pool.values()].flat().map((e) => describe(e.item));

  const CELL = 512;
  const grid = new Map<string, number[]>();
  const boxes = pairs.map((p) => bbox(p.item));
  boxes.forEach((bb, idx) => {
    for (let x = Math.floor(bb[0] / CELL); x <= Math.floor(bb[2] / CELL); x++)
      for (let y = Math.floor(bb[1] / CELL); y <= Math.floor(bb[3] / CELL); y++) grid.set(`${x},${y}`, [...(grid.get(`${x},${y}`) ?? []), idx]);
  });
  const seen = new Set<string>();
  const misordered: string[] = [];
  for (const cell of grid.values()) {
    for (let m = 0; m < cell.length; m++) {
      for (let n = m + 1; n < cell.length; n++) {
        const [i, j] = [cell[m], cell[n]];
        const id = i < j ? `${i}:${j}` : `${j}:${i}`;
        if (seen.has(id)) continue;
        const [p, q] = [boxes[i], boxes[j]];
        if (p[0] >= q[2] || q[0] >= p[2] || p[1] >= q[3] || q[1] >= p[3]) continue;
        seen.add(id);
        const before = pairs[i].wantOrder < pairs[j].wantOrder;
        if (before !== pairs[i].gotOrder < pairs[j].gotOrder) {
          misordered.push(`${describe(pairs[i].item)} ${before ? 'below' : 'above'} ${describe(pairs[j].item)}`);
        }
      }
    }
  }
  return { compared: a.length, missing, extra, misordered, overlappingPairs: seen.size };
};
