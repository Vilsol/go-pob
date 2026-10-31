// The PoE2 passive tree as vector shapes: a disc and ring per node, a line or arc per connector.

export type Tree2Connection = { id: number; orbit: number };

export type Tree2Node = {
  id: number;
  x: number;
  y: number;
  angle: number;
  group: number;
  orbit: number;
  type: string;
  name: string;
  stats: string[] | null;
  ascendancy?: string;
  isAttribute?: boolean;
  isAscendancyStart?: boolean;
  containJewelSocket?: boolean;
  // a radius socket's socket.nodesInRadius (PassiveTree.lua:369-396), by radius index - 1
  nodesInRadius?: number[][];
  classesStart?: string[];
  connections: Tree2Connection[] | null;
};

export type Tree2 = {
  nodes: Tree2Node[];
  groups: { id: number; x: number; y: number }[];
  orbitRadii: number[];
  minX: number;
  minY: number;
  maxX: number;
  maxY: number;
  // data.jewelRadius with PassiveTreeJewelDistanceMultiplier applied to inner and outer; col is "#rrggbb"
  jewelRadius?: { inner: number; outer: number; col: string }[];
};

export type Placed2 = { node: Tree2Node; r: number; rsq: number };

// kind 0: a line x0, y0, x1, y1; kind 1: an arc about cx, cy of radius r from angle a0 sweeping clockwise
export type Connector2 = { a: number; b: number; kind: 0 | 1; geom: [number, number, number, number, number]; ascendancy?: string };

export type Scene2State = {
  alloc: Set<number>;
  allocMode: Map<number, number>;
  hovered: number | undefined;
  hoverPath: Set<number>;
  hoverDep: Set<number>;
  ascendClassBaseName: string;
  // the nodes the search matches, and the highlight's radius in tree units
  search?: Set<number>;
  searchR?: number;
  // spec.allocMode: the weapon set a click allocates to, 0 for none
  specAllocMode?: number;
  // the jewelRadiusIndex of the jewel in each allocated socket that has one
  jewelRadiusIndex?: Map<number, number>;
};

export type Scene2 = {
  tree: Tree2;
  byId: Map<number, Tree2Node>;
  nodes: Placed2[];
  connectors: Connector2[];
  size: number;
  discs: Float32Array;
  strokes: Float32Array;
  counts: { discs: number; strokes: number };
};

export const DISC_FLOATS = 11;
export const STROKE2_FLOATS = 11;

/* PassiveTree.lua:821-879: the base art's half width, and the overlay's, which PassiveTreeView.lua:291-299 hit-tests */
const targetSize = (n: Tree2Node): { base: number; overlay: number } | undefined => {
  if (n.isAscendancyStart) return { base: 16, overlay: 50 };
  if (n.type === 'Normal' && n.ascendancy) return { base: 37, overlay: 80 };
  if (n.containJewelSocket) return { base: 80, overlay: 80 };
  if (n.ascendancy) return { base: 54, overlay: 100 };
  switch (n.type) {
    case 'Notable':
      return { base: 54, overlay: 80 };
    case 'AscendClassStart':
      return { base: 16, overlay: 24 };
    case 'Keystone':
      return { base: 82, overlay: 120 };
    case 'Normal':
      return { base: 37, overlay: 54 };
    case 'Socket':
      return { base: 76, overlay: 76 };
    case 'ClassStart':
      return { base: 37, overlay: 1 };
  }
  return undefined;
};

/* PassiveTree.lua:630-744, as one vector line or arc per connection */
const buildConnector = (tree: Tree2, groups: Map<number, { x: number; y: number }>, n1: Tree2Node, n2: Tree2Node, connection: Tree2Connection): Connector2 => {
  const base = { a: n1.id, b: n2.id, ascendancy: n1.ascendancy };
  const orbitR = tree.orbitRadii[Math.abs(connection.orbit)];
  if (connection.orbit !== 0 && orbitR !== undefined) {
    const r = orbitR;
    const dx = n2.x - n1.x;
    const dy = n2.y - n1.y;
    const dist = Math.sqrt(dx * dx + dy * dy);
    if (dist < r * 2) {
      const perp = Math.sqrt(r * r - (dist * dist) / 4) * (connection.orbit > 0 ? 1 : -1);
      const cx = n1.x + dx / 2 + perp * (dy / dist);
      const cy = n1.y + dy / 2 - perp * (dx / dist);
      let a1 = Math.atan2(n1.y - cy, n1.x - cx);
      let a2 = Math.atan2(n2.y - cy, n2.x - cx);
      if (a1 > a2) [a1, a2] = [a2, a1];
      let arc = a2 - a1;
      if (arc >= Math.PI) {
        [a1, a2] = [a2, a1];
        arc = Math.PI * 2 - arc;
      }
      if (arc <= Math.PI) return { ...base, kind: 1, geom: [cx, cy, r, a1, arc] };
    }
  } else if (n1.group === n2.group && n1.orbit === n2.orbit && connection.orbit === 0) {
    let [p, q] = n1.angle > n2.angle ? [n2, n1] : [n1, n2];
    let arc = q.angle - p.angle;
    if (arc >= Math.PI) {
      [p, q] = [q, p];
      arc = Math.PI * 2 - arc;
    }
    if (arc <= Math.PI) {
      const group = groups.get(p.group)!;
      // the tree's angles run clockwise from north; the arc's from east
      return { ...base, kind: 1, geom: [group.x, group.y, tree.orbitRadii[p.orbit], p.angle - Math.PI / 2, arc] };
    }
  }
  return { ...base, kind: 0, geom: [n1.x, n1.y, n2.x, n2.y, 0] };
};

export const buildScene2 = (tree: Tree2): Scene2 => {
  const byId = new Map(tree.nodes.map((n) => [n.id, n]));
  const nodes: Placed2[] = [];
  for (const node of tree.nodes) {
    if (node.type === 'OnlyImage') continue;
    const size = targetSize(node);
    if (!size) continue;
    nodes.push({ node, r: node.type === 'ClassStart' ? 120 : size.base, rsq: size.overlay * size.overlay });
  }
  /* PassiveTree.lua:305-366 */
  const connectors: Connector2[] = [];
  const groups = new Map(tree.groups.map((g) => [g.id, g]));
  for (const node of tree.nodes) {
    for (const connection of node.connections ?? []) {
      const other = byId.get(connection.id);
      if (!other || node.type === 'OnlyImage' || other.type === 'OnlyImage' || node.id === other.id) continue;
      if (node.ascendancy !== other.ascendancy || node.classesStart || other.classesStart) continue;
      connectors.push(buildConnector(tree, groups, node, other, connection));
    }
  }
  const size = Math.min(tree.maxX - tree.minX, tree.maxY - tree.minY) * 1.1;
  return {
    tree,
    byId,
    nodes,
    connectors,
    size,
    discs: new Float32Array(nodes.length * DISC_FLOATS),
    strokes: new Float32Array(connectors.length * STROKE2_FLOATS),
    counts: { discs: 0, strokes: 0 }
  };
};

/* PassiveTreeView.lua:291-299; the nearest node rather than the first in pairs() order */
export const nodeAt2 = (scene: Scene2, x: number, y: number): Tree2Node | undefined => {
  let best: Tree2Node | undefined;
  let bestD = Infinity;
  for (const p of scene.nodes) {
    if (p.node.isAscendancyStart) continue;
    const d = (x - p.node.x) ** 2 + (y - p.node.y) ** 2;
    if (d <= p.rsq && d < bestD) [best, bestD] = [p.node, d];
  }
  return best;
};

type RGB = [number, number, number];
const hex = (h: string): RGB => [parseInt(h.slice(1, 3), 16) / 255, parseInt(h.slice(3, 5), 16) / 255, parseInt(h.slice(5, 7), 16) / 255];

export const COLOURS = {
  connector: hex('#524518'),
  intermediate: hex('#c89c01'),
  active: hex('#e9deb6'),
  depends: hex('#e90000'),
  otherAscendancy: hex('#5f5f5f'),
  weaponSet1: hex('#dd0022'),
  weaponSet2: hex('#33ff77'),
  ringUnalloc: hex('#524518'),
  ringPath: hex('#c89c01'),
  ringAlloc: hex('#e9deb6'),
  normal: hex('#8a8170'),
  attribute: hex('#6f8f7a'),
  notable: hex('#c9a046'),
  keystone: hex('#d2693c'),
  socket: hex('#5a86c8'),
  search: hex('#ff0000'),
  /* PassiveTreeView.lua:21, at alpha 0.7 */
  jewelRadius: hex('#ffffff'),
  ascendancy: hex('#9a72b8'),
  classStart: hex('#c89c01')
};

const fillOf = (n: Tree2Node): RGB => {
  if (n.type === 'ClassStart' || n.type === 'AscendClassStart') return COLOURS.classStart;
  if (n.type === 'Socket' || n.containJewelSocket) return COLOURS.socket;
  if (n.ascendancy) return COLOURS.ascendancy;
  if (n.type === 'Keystone') return COLOURS.keystone;
  if (n.type === 'Notable') return COLOURS.notable;
  if (n.isAttribute) return COLOURS.attribute;
  return COLOURS.normal;
};

const STROKE_WIDTH = { normal: 10, active: 18 };

/* PassiveTreeView.lua:675-706 */
const nodeIsHoverPathEndpoint = (s: Scene2State, id: number) => {
  if (id === s.hovered || s.hoverPath.has(id)) return true;
  if (s.alloc.has(id)) {
    const mode = s.allocMode.get(id) ?? 0;
    return mode === 0 || mode === (s.specAllocMode ?? 0);
  }
  return false;
};
const nodeWillChangeAllocMode = (s: Scene2State, id: number) =>
  s.hovered !== undefined && s.hoverPath.has(id) && !s.alloc.has(s.hovered) && (s.specAllocMode ?? 0) === 0 && (s.allocMode.get(id) ?? 0) > 0;
const nodeWillAllocateWithAllocMode = (s: Scene2State, n: Tree2Node) =>
  s.hoverPath.has(n.id) &&
  s.hovered !== undefined &&
  !s.alloc.has(s.hovered) &&
  !s.alloc.has(n.id) &&
  (s.specAllocMode ?? 0) > 0 &&
  !n.ascendancy &&
  n.type !== 'Keystone' &&
  n.type !== 'Socket' &&
  !n.containJewelSocket;
const connectorState = (s: Scene2State, a: number, b: number): 'Normal' | 'Intermediate' | 'Active' => {
  const m1 = s.allocMode.get(a) ?? 0;
  const m2 = s.allocMode.get(b) ?? 0;
  const a1 = s.alloc.has(a);
  const a2 = s.alloc.has(b);
  const endpoints = nodeIsHoverPathEndpoint(s, a) && nodeIsHoverPathEndpoint(s, b);
  if (endpoints && s.hoverPath.has(a) && s.hoverPath.has(b) && (nodeWillChangeAllocMode(s, a) || nodeWillChangeAllocMode(s, b))) return 'Intermediate';
  if (a1 && a2 && (m1 === 0 || m2 === 0 || m1 === m2)) return 'Active';
  if (s.hoverPath.size && endpoints && (!a1 || !a2 || (s.hoverPath.has(a) && s.hoverPath.has(b)))) return 'Intermediate';
  return 'Normal';
};

export const fillScene2 = (scene: Scene2, s: Scene2State) => {
  let o = 0;
  const stroke = (v: number[]) => {
    if (o + STROKE2_FLOATS > scene.strokes.length) {
      const grown = new Float32Array(scene.strokes.length * 2 + STROKE2_FLOATS * 16);
      grown.set(scene.strokes);
      scene.strokes = grown;
    }
    scene.strokes.set(v, o);
    o += STROKE2_FLOATS;
  };
  for (const c of scene.connectors) {
    const state = connectorState(s, c.a, c.b);
    let colour = state === 'Active' ? COLOURS.active : state === 'Intermediate' ? COLOURS.intermediate : COLOURS.connector;
    /* PassiveTreeView.lua:735-741 */
    if (state === 'Intermediate' && !c.ascendancy) {
      if (s.specAllocMode === 1) colour = COLOURS.weaponSet1;
      else if (s.specAllocMode === 2) colour = COLOURS.weaponSet2;
    }
    /* PassiveTreeView.lua:742-769 */
    if (state === 'Active' && !c.ascendancy) {
      const mode = s.allocMode.get(c.a) || s.allocMode.get(c.b) || 0;
      if (mode === 1) colour = COLOURS.weaponSet1;
      else if (mode === 2) colour = COLOURS.weaponSet2;
    }
    if (s.hoverDep.has(c.a) && s.hoverDep.has(c.b)) colour = COLOURS.depends;
    else if (c.ascendancy && c.ascendancy !== s.ascendClassBaseName) colour = state === 'Normal' ? COLOURS.otherAscendancy : colour;
    stroke([...c.geom, c.kind, state === 'Normal' ? STROKE_WIDTH.normal : STROKE_WIDTH.active, ...colour, 1]);
  }
  /* PassiveTreeView.lua:1181-1206, a ring stroke for the highlight image and without the edge snap */
  const searchR = s.searchR ?? 140;
  for (const id of s.search ?? []) {
    const n = scene.byId.get(id);
    if (n) stroke([n.x, n.y, searchR, 0, Math.PI * 2, 1, searchR * 0.12, ...COLOURS.search, 1]);
  }
  /* PassiveTreeView.lua:1311-1353, a plain ring for each image, without Thread of Hope's annuli, timeless art or the compare spec */
  const radii = scene.tree.jewelRadius ?? [];
  const hoveredNode = s.hovered === undefined ? undefined : scene.byId.get(s.hovered);
  const radiusSocket = (n: Tree2Node | undefined) => !!n && n.type === 'Socket' && !!n.nodesInRadius;
  if (radiusSocket(hoveredNode)) {
    for (const r of radii) if (r.inner === 0) stroke([hoveredNode!.x, hoveredNode!.y, r.outer, 0, Math.PI * 2, 1, STROKE_WIDTH.normal, ...hex(r.col), 1]);
  }
  for (const [id, index] of s.jewelRadiusIndex ?? []) {
    const n = scene.byId.get(id);
    const r = radii[index - 1];
    if (!r || !radiusSocket(n) || !s.alloc.has(id)) continue;
    stroke([n!.x, n!.y, r.outer, 0, Math.PI * 2, 1, STROKE_WIDTH.active, ...COLOURS.jewelRadius, 0.7]);
    if (r.inner) stroke([n!.x, n!.y, r.inner * 1.06, 0, Math.PI * 2, 1, STROKE_WIDTH.active, ...COLOURS.jewelRadius, 0.7]);
  }
  scene.counts.strokes = o / STROKE2_FLOATS;
  /* PassiveTreeView.lua:1148-1157 */
  const inRadiusColour = new Map<number, RGB>();
  if (radiusSocket(hoveredNode)) {
    radii.forEach((r, i) => {
      if (r.inner !== 0) return;
      for (const id of hoveredNode!.nodesInRadius![i] ?? []) if (!inRadiusColour.has(id)) inRadiusColour.set(id, hex(r.col));
    });
  }

  o = 0;
  for (const p of scene.nodes) {
    const n = p.node;
    const alloc = s.alloc.has(n.id);
    const shown = alloc || n.id === s.hovered;
    let ring = shown ? COLOURS.ringAlloc : s.hoverPath.has(n.id) ? COLOURS.ringPath : COLOURS.ringUnalloc;
    /* PassiveTreeView.lua:936-937, 1120-1121 */
    const mode = nodeWillAllocateWithAllocMode(s, n) ? (s.specAllocMode ?? 0) : (s.allocMode.get(n.id) ?? 0);
    if (mode > 0 && !nodeWillChangeAllocMode(s, n.id)) ring = mode === 1 ? COLOURS.weaponSet1 : COLOURS.weaponSet2;
    if (s.hovered !== undefined && n.id !== s.hovered && n.type !== 'ClassStart' && n.type !== 'AscendClassStart') {
      if (s.hoverDep.has(n.id)) ring = COLOURS.depends;
      else ring = inRadiusColour.get(n.id) ?? ring;
    }
    const base = fillOf(n);
    const k = shown ? 1 : s.hoverPath.has(n.id) ? 0.7 : 0.4;
    const alpha = n.ascendancy && n.ascendancy !== s.ascendClassBaseName && !alloc ? 0.55 : 1;
    if (n.type === 'ClassStart') {
      scene.discs.set([n.x, n.y, p.r, p.r * 0.12, 0, 0, 0, ...(alloc ? COLOURS.classStart : COLOURS.ringUnalloc), 1], o);
    } else {
      scene.discs.set([n.x, n.y, p.r, p.r * 0.22, base[0] * k, base[1] * k, base[2] * k, ...ring, alpha], o);
    }
    o += DISC_FLOATS;
  }
  scene.counts.discs = o / DISC_FLOATS;
};

/* PassiveTreeView.lua:257-274, 1500-1514 */
export type View2 = { zoomLevel: number; zoomX: number; zoomY: number; width: number; height: number; dpr: number };

export const zoomOf = (v: View2) => Math.pow(1.2, v.zoomLevel);
export const scaleOf = (scene: Scene2, v: View2) => (Math.min(v.width, v.height) / scene.size) * zoomOf(v);
export const clampView = (v: View2): View2 => {
  const f = (zoomOf(v) * 2) / 3;
  return { ...v, zoomX: Math.min(Math.max(v.zoomX, -v.width * f), v.width * f), zoomY: Math.min(Math.max(v.zoomY, -v.height * f), v.height * f) };
};
export const treeToScreen = (scene: Scene2, v: View2, x: number, y: number) => {
  const s = scaleOf(scene, v);
  return { x: x * s + v.zoomX + v.width / 2, y: y * s + v.zoomY + v.height / 2 };
};
export const screenToTree = (scene: Scene2, v: View2, x: number, y: number) => {
  const s = scaleOf(scene, v);
  return { x: (x - v.zoomX - v.width / 2) / s, y: (y - v.zoomY - v.height / 2) / s };
};
export const zoomAbout = (v: View2, level: number, cx: number, cy: number): View2 => {
  const zoomLevel = Math.max(0, Math.min(20, v.zoomLevel + level));
  const factor = Math.pow(1.2, zoomLevel) / zoomOf(v);
  const relX = cx - v.width / 2;
  const relY = cy - v.height / 2;
  return { ...v, zoomLevel, zoomX: relX + (v.zoomX - relX) * factor, zoomY: relY + (v.zoomY - relY) * factor };
};
