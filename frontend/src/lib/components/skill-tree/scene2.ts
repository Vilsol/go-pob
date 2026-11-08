// The PoE2 passive tree as vector shapes: a disc and ring per node, a line or arc per connector; with the tree's art
// attached, its sprites draw over them in PassiveTreeView.lua's order and a node with art drops its disc.
import type { ArtSprite, Tree2ArtData, Tree2Atlas } from '$lib/skill_tree/tree2_art';

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
  // node.unlockConstraint.nodes: the view shows the node once all of them are allocated
  unlockConstraint?: number[];
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
  // the art the view draws by name, from the tree's data
  art?: Tree2ArtData;
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
  // spec.curClassName, for the class background art
  className?: string;
  // the icon a switched or attribute-chosen spec node draws in place of the tree's
  icons?: Map<number, string>;
  // each socket's jewel, whose art PassiveTreeView.lua:961-969 draws in it
  jewels?: Map<number, { title: string; baseName: string; unique: boolean }>;
  // the compared spec's allocated nodes (PassiveTreeView.lua:722-734, 932, 1057-1067)
  compared?: Set<number>;
  // while the heat map shows, each node's power colour (PassiveTreeView.lua:1003-1030); a node without one is black
  heat?: Map<number, [number, number, number]>;
};

type NodeArt2 = { icon?: ArtSprite; effect?: ArtSprite; effectR?: number; overlay?: { alloc?: ArtSprite; path?: ArtSprite; unalloc?: ArtSprite } };

type LineState = 'Normal' | 'Intermediate' | 'Active';
// one image quad of a connector: origin, u and v edges, the kite clip of an orbit arc (1 for none), the line's repeats,
// and the radius about the arc's centre corner, as a fraction of the image, inside which the sheet holds other arcs
type ArtQuad = { sprite: ArtSprite; o: [number, number]; u: [number, number]; v: [number, number]; clip: number; repeat: number; inner: number };
// a connector's quads in each line state; an arc wider than 90 degrees is two
type ArtConnector2 = Partial<Record<LineState, ArtQuad[]>>;

export type Scene2Art = {
  // the data version's root the sheets load from
  base: string;
  atlas: Tree2Atlas;
  data: Tree2ArtData;
  nodes: Map<number, NodeArt2>;
  onlyImage: { node: Tree2Node; sprite: ArtSprite }[];
  ascendancyMiddle?: ArtSprite;
  // by scene.connectors index
  connectors: ArtConnector2[];
};

export type Scene2 = {
  tree: Tree2;
  byId: Map<number, Tree2Node>;
  nodes: Placed2[];
  connectors: Connector2[];
  size: number;
  discs: Float32Array;
  strokes: Float32Array;
  art?: Scene2Art;
  // the art drawn below the connectors, the connectors' art, and the art above them
  under: Float32Array;
  lines: Float32Array;
  over: Float32Array;
  // the strokes after the first connectorStrokes draw over the nodes when the tree has art
  counts: { discs: number; strokes: number; under: number; lines: number; over: number; connectorStrokes: number };
};

export const DISC_FLOATS = 11;
export const STROKE2_FLOATS = 11;
// origin x, y; the u and v edges; the source rect x, y, w, h on its sheet; the sheet; tint r, g, b, a; the kite clip
// (an orbit arc's p, 1 for none); the times the image repeats along u; the inner radius an arc's image is cut at
export const SPRITE2_FLOATS = 18;

/* PassiveTree.lua:821-879: the base art's half width, and the overlay's, which PassiveTreeView.lua:291-299 hit-tests */
const targetSize = (n: Tree2Node): { base: number; overlay: number; effect?: number } | undefined => {
  if (n.isAscendancyStart) return { base: 16, overlay: 50 };
  if (n.type === 'Normal' && n.ascendancy) return { base: 37, overlay: 80 };
  if (n.containJewelSocket) return { base: 80, overlay: 80 };
  if (n.ascendancy) return { base: 54, overlay: 100 };
  switch (n.type) {
    case 'Notable':
      return { base: 54, overlay: 80, effect: 380 };
    case 'AscendClassStart':
      return { base: 16, overlay: 24, effect: 380 };
    case 'OnlyImage':
      return { base: 380, overlay: 0 };
    case 'Keystone':
      return { base: 82, overlay: 120, effect: 380 };
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
    under: new Float32Array(0),
    lines: new Float32Array(0),
    over: new Float32Array(0),
    counts: { discs: 0, strokes: 0, under: 0, lines: 0, over: 0, connectorStrokes: 0 }
  };
};

const LINE_STATES: LineState[] = ['Normal', 'Intermediate', 'Active'];

/* PassiveTree.lua:746-786: an orbit's quarter-arc image on a kite-shaped quad about the orbit's centre, clipped at both
   ends to arcAngle; the quad's corners are an affine image of the texture's, so it is the image's square with the
   clip p */
const buildArc = (
  get: (name: string) => ArtSprite | undefined,
  inner: (sp: ArtSprite) => number,
  art: string,
  arcAngle: number,
  orbit: number,
  xScale: number,
  yScale: number,
  angle0: number,
  mirrored: boolean
) => {
  const clipAngle = Math.PI / 4 - arcAngle / 2;
  const p = 1 - Math.max(Math.tan(clipAngle), 0);
  let angle = angle0 - clipAngle;
  if (mirrored) angle += arcAngle;
  const out: Partial<Record<LineState, ArtQuad>> = {};
  for (const state of LINE_STATES) {
    const sprite = get(`${art}Orbit${orbit}${state}`);
    if (!sprite) continue;
    const size = sprite.width;
    const oX = size * Math.SQRT2 * Math.sin(angle + Math.PI / 4);
    const oY = size * Math.SQRT2 * -Math.cos(angle + Math.PI / 4);
    const e1: [number, number] = [size * Math.cos(angle) - oX, size * Math.sin(angle) - oY];
    const e2: [number, number] = [size * Math.sin(angle) - oX, size * -Math.cos(angle) - oY];
    // a mirrored arc swaps the quad's two clipped corners, which swaps the edges
    out[state] = { sprite, o: [xScale + oX, yScale + oY], u: mirrored ? e2 : e1, v: mirrored ? e1 : e2, clip: p, repeat: 1, inner: inner(sprite) };
  }
  return out;
};

/* PassiveTree.lua:630-744: the connector's image quads, by line state */
const buildArtConnector = (
  tree: Tree2,
  groups: Map<number, { x: number; y: number }>,
  get: (name: string) => ArtSprite | undefined,
  inner: (sp: ArtSprite) => number,
  art: string,
  n1: Tree2Node,
  n2: Tree2Node,
  connection: Tree2Connection
): ArtConnector2 => {
  const arcs = (arcAngle: number, orbit: number, x: number, y: number, angle: number): ArtConnector2 => {
    const pair = arcAngle > Math.PI / 2;
    if (pair) arcAngle /= 2;
    const first = buildArc(get, inner, art, arcAngle, orbit, x, y, angle, false);
    const second = pair ? buildArc(get, inner, art, arcAngle, orbit, x, y, angle, true) : undefined;
    const out: ArtConnector2 = {};
    for (const state of LINE_STATES) {
      const q = [first[state], second?.[state]].filter((e): e is ArtQuad => !!e);
      if (q.length === (pair ? 2 : 1)) out[state] = q;
    }
    return out;
  };
  const orbitR = tree.orbitRadii[Math.abs(connection.orbit)];
  if (connection.orbit !== 0 && orbitR !== undefined) {
    const orbit = Math.abs(connection.orbit);
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
      a1 += Math.PI / 2;
      if (arc <= Math.PI) return arcs(arc, orbit, cx, cy, a1);
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
      return arcs(arc, p.orbit, group.x, group.y, p.angle);
    }
  }
  /* PassiveTree.lua:726-743, each state at its own art's height: the sheets crop each state's image to its art, where
     PoB's images of the three states share one size */
  const out: ArtConnector2 = {};
  const vX = n2.x - n1.x;
  const vY = n2.y - n1.y;
  const dist = Math.sqrt(vX * vX + vY * vY);
  for (const state of LINE_STATES) {
    const sprite = get(`${art}LineConnector${state}`);
    if (!sprite || !dist) continue;
    const scale = (sprite.height * 0.5) / dist;
    const [nX, nY] = [vX * scale, vY * scale];
    out[state] = [{ sprite, o: [n1.x + nY, n1.y - nX], u: [vX, vY], v: [-2 * nY, 2 * nX], clip: 1, repeat: dist / sprite.width, inner: 0 }];
  }
  return out;
};

/* PassiveTree.lua:144-181, 547-568, 630-633: each node's and connector's art by the names PassiveTreeView.lua draws
   it by */
export const attachArt2 = (scene: Scene2, data: Tree2ArtData, atlas: Tree2Atlas, base: string) => {
  const get = (name: string | undefined) => (name ? atlas.byName.get(name) : undefined);
  const nodes = new Map<number, NodeArt2>();
  const onlyImage: Scene2Art['onlyImage'] = [];
  for (const node of scene.tree.nodes) {
    const d = data.nodes[String(node.id)];
    if (!d) continue;
    if (node.type === 'OnlyImage') {
      const sprite = get(d.activeEffectImage);
      if (sprite) onlyImage.push({ node, sprite });
      continue;
    }
    const overlay = d.nodeOverlay ?? data.nodeOverlay?.[node.type];
    nodes.set(node.id, {
      icon: get(d.icon),
      effect: get(d.activeEffectImage),
      effectR: targetSize(node)?.effect,
      overlay: overlay && { alloc: get(overlay.alloc), path: get(overlay.path), unalloc: get(overlay.unalloc) }
    });
  }
  const groups = new Map(scene.tree.groups.map((g) => [g.id, g]));
  /* the sheet packs a line state's orbit images into one quarter, sharing the centre corner, so an orbit's image holds
     the smaller orbits' arcs; each draws only outside the next smaller image */
  const sizes = new Map<string, number[]>();
  const cornerOf = (sp: ArtSprite) => `${sp.sheet}:${sp.x + sp.w}:${sp.y + sp.h}`;
  for (const [name, sp] of atlas.byName) {
    if (!/Orbit\d+(Normal|Intermediate|Active)$/.test(name)) continue;
    const k = cornerOf(sp);
    sizes.set(k, [...(sizes.get(k) ?? []), sp.w]);
  }
  const inner = (sp: ArtSprite) => Math.max(0, ...(sizes.get(cornerOf(sp)) ?? []).filter((w) => w < sp.w)) / sp.w;
  const connectors = scene.connectors.map((c) => {
    const n1 = scene.byId.get(c.a)!;
    const n2 = scene.byId.get(c.b)!;
    const connection = n1.connections!.find((x) => x.id === c.b)!;
    const art =
      data.nodes[String(n1.id)]?.connectionArt ?? data.nodes[String(n2.id)]?.connectionArt ?? data.connectionArt?.[n1.ascendancy ? 'ascendancy' : 'default'];
    return art ? buildArtConnector(scene.tree, groups, get, inner, art, n1, n2, connection) : {};
  });
  scene.art = { base, atlas, data, nodes, onlyImage, ascendancyMiddle: get('AscendancyMiddle'), connectors };
};

const UNSEEN_PATH = 5571;
/* PassiveTreeView.lua:2179-2191 */
const unlocked = (s: Pick<Scene2State, 'alloc' | 'hovered'>, n: Tree2Node | undefined) => {
  const nodes = n?.unlockConstraint;
  if (!nodes) return true;
  // PassiveTreeView.lua:317: hovering an unallocated Unseen Path shows the nodes it unlocks
  if (s.hovered === UNSEEN_PATH && !s.alloc.has(UNSEEN_PATH) && nodes[0] === UNSEEN_PATH) return true;
  return nodes.every((id) => s.alloc.has(id));
};
export const nodeShown2 = unlocked;

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

type Tint = [number, number, number, number];
const WHITE: Tint = [1, 1, 1, 1];
/* PassiveTreeView.lua:929 */
const HALF_GRAY: Tint = [128 / 255, 128 / 255, 128 / 255, 1];
/* PassiveTreeView.lua:711-716: 0.75 gray, colorCodes.HIGHLIGHT, ^x00FF00 */
const INACTIVE_GRAY: Tint = [0xbf / 255, 0xbf / 255, 0xbf / 255, 1];
const BRIGHT_RED: Tint = [1, 0, 0, 1];
const BRIGHT_GREEN: Tint = [0, 1, 0, 1];
const opaque = (c: RGB): Tint => [c[0], c[1], c[2], 1];

const classOf = (scene: Scene2, s: Scene2State) => {
  const classes = scene.art?.data.classes ?? [];
  if (s.className) return classes.find((c) => c.name === s.className);
  const asc = s.ascendClassBaseName && classes.find((c) => c.ascendancies?.some((a) => a.id === s.ascendClassBaseName || a.name === s.ascendClassBaseName));
  if (asc) return asc;
  const start = scene.tree.nodes.find((n) => n.type === 'ClassStart' && s.alloc.has(n.id));
  return start && classes.find((c) => c.name === start.classesStart?.[0]);
};

type Layer = 'under' | 'lines' | 'over';
type Writer = {
  // an image whose u edge runs from (ox, oy) by (ux, uy) and v edge by (vx, vy); false while its sheet loads
  quad: (
    layer: Layer,
    sp: ArtSprite,
    ox: number,
    oy: number,
    ux: number,
    uy: number,
    vx: number,
    vy: number,
    tint: Tint,
    clip?: number,
    repeat?: number,
    inner?: number
  ) => boolean;
  out: Record<Layer, number>;
};

const writer = (scene: Scene2, ready: (sheet: number) => boolean): Writer => {
  const out = { under: 0, lines: 0, over: 0 };
  const quad: Writer['quad'] = (layer, sp, ox, oy, ux, uy, vx, vy, tint, clip = 1, repeat = 1, inner = 0) => {
    if (!ready(sp.sheet)) return false;
    let buf = scene[layer];
    if (out[layer] + SPRITE2_FLOATS > buf.length) {
      const grown = new Float32Array(buf.length * 2 + SPRITE2_FLOATS * 64);
      grown.set(buf);
      buf = scene[layer] = grown;
    }
    buf.set([ox, oy, ux, uy, vx, vy, sp.x, sp.y, sp.w, sp.h, sp.sheet, ...tint, clip, repeat, inner], out[layer]);
    out[layer] += SPRITE2_FLOATS;
    return true;
  };
  return { quad, out };
};

/* PassiveTreeView.lua:717-779: the connector's tint, with its state where a compared spec lifts it to Active */
const connectorTint = (s: Scene2State, c: Connector2, state: LineState): { state: LineState; tint: Tint } => {
  let tint = WHITE;
  let baseState = state;
  if (s.compared) baseState = s.compared.has(c.a) && s.compared.has(c.b) ? 'Active' : 'Normal';
  if (baseState === 'Active' && state !== 'Active') {
    state = 'Active';
    tint = BRIGHT_GREEN;
  }
  if (baseState !== 'Active' && state === 'Active') tint = BRIGHT_RED;
  const specMode = s.specAllocMode ?? 0;
  if (baseState === 'Intermediate' && specMode > 0 && !c.ascendancy) tint = opaque(specMode === 1 ? COLOURS.weaponSet1 : COLOURS.weaponSet2);
  if (baseState === 'Active' && state === 'Active' && !c.ascendancy) {
    const mode = s.allocMode.get(c.a) || s.allocMode.get(c.b) || 0;
    if (mode === 1) tint = opaque(COLOURS.weaponSet1);
    else if (mode === 2) tint = opaque(COLOURS.weaponSet2);
  }
  const unseen = s.hovered === UNSEEN_PATH && (s.hoverDep.has(c.a) || s.hoverDep.has(c.b)) && !c.ascendancy;
  if (unseen && !s.alloc.has(UNSEEN_PATH)) tint = BRIGHT_GREEN;
  else if (unseen) tint = BRIGHT_RED;
  else if (s.hoverDep.has(c.a) && s.hoverDep.has(c.b)) tint = BRIGHT_RED;
  else if (c.ascendancy && c.ascendancy !== s.ascendClassBaseName) tint = INACTIVE_GRAY;
  return { state, tint };
};

// the art's sprites into scene.under and scene.over; the ids of the nodes it drew, which then draw no disc
const fillArt2 = (scene: Scene2, s: Scene2State, w: Writer, inRadiusColour: Map<number, RGB>): Set<number> => {
  const drawn = new Set<number>();
  const art = scene.art!;
  const { quad } = w;
  /* PassiveTreeView.lua:1374-1396, without isHalf: no PoE2 group has a half image */
  const asset = (layer: Layer, sp: ArtSprite | undefined, x: number, y: number, hw: number, hh: number, tint: Tint) =>
    !!sp && quad(layer, sp, x - hw, y - hh, hw * 2, 0, 0, hh * 2, tint);

  /* PassiveTreeView.lua:602-631 */
  const cls = classOf(scene, s);
  const bg = cls?.background;
  let classArt = false;
  if (cls && bg) {
    const asc = s.ascendClassBaseName ? cls.ascendancies?.find((a) => a.id === s.ascendClassBaseName || a.name === s.ascendClassBaseName) : undefined;
    const image = art.atlas.byName.get(asc?.background?.image ?? bg.image);
    asset('under', image, bg.x, bg.y, bg.width, bg.height, WHITE);
    const start = scene.tree.nodes.find((n) => n.type === 'ClassStart' && n.classesStart?.includes(cls.name));
    const active = art.atlas.byName.get('BGTreeActive');
    if (active && start && bg.active) {
      /* PassiveTreeView.lua:1459-1475 */
      const a = Math.PI / 2 + Math.atan2(start.y - bg.y, start.x - bg.x);
      const [c, sn] = [Math.cos(a), Math.sin(a)];
      const [aw, ah] = [bg.active.width, bg.active.height];
      quad('under', active, bg.x - aw * c + ah * sn, bg.y - aw * sn - ah * c, 2 * aw * c, 2 * aw * sn, -2 * ah * sn, 2 * ah * c, WHITE);
    }
    if (bg.bg) classArt = asset('under', art.atlas.byName.get('BGTree'), bg.x, bg.y, bg.bg.width, bg.bg.height, WHITE);
  }

  /* PassiveTreeView.lua:633-655 */
  for (const c of art.data.classes ?? []) {
    for (const a of c.ascendancies ?? []) {
      const b = a.background;
      if (!b) continue;
      const name = a.id ?? a.name;
      if ((a.replaceBy && a.replaceBy === s.ascendClassBaseName) || (a.replace && name !== s.ascendClassBaseName)) continue;
      const k = name === s.ascendClassBaseName ? 1 : 0.5;
      asset('under', art.atlas.byName.get(b.image), b.x, b.y, b.width, b.height, [k, k, k, 1]);
    }
  }

  const onPath = (id: number) => s.hoverPath.has(id);
  /* PassiveTreeView.lua:970-977, 1103-1105: an OnlyImage's art at layer 15 */
  for (const { node, sprite } of art.onlyImage) if (unlocked(s, node)) asset('under', sprite, node.x, node.y, 380, 380, [1, 1, 1, 0.15]);

  /* PassiveTreeView.lua:930-1180 */
  for (const p of scene.nodes) {
    const n = p.node;
    const na = art.nodes.get(n.id);
    if (n.type === 'ClassStart') {
      if (classArt) drawn.add(n.id);
      continue;
    }
    if (!unlocked(s, n)) {
      drawn.add(n.id);
      continue;
    }
    if (!na) continue;
    const size = targetSize(n)!;
    const alloc = s.alloc.has(n.id);
    const comparedAlloc = s.compared?.has(n.id);
    const isAlloc = alloc || !!comparedAlloc;
    /* PassiveTreeView.lua:948 */
    const state = s.heat || isAlloc || n.id === s.hovered ? 'alloc' : onPath(n.id) ? 'path' : 'unalloc';
    /* PassiveTreeView.lua:1003-1070 */
    let colour: Tint = WHITE;
    if (s.heat && !isAlloc && n.type !== 'AscendClassStart') colour = opaque(s.heat.get(n.id) ?? [0, 0, 0]);
    else if (s.compared) colour = comparedAlloc && !alloc ? BRIGHT_GREEN : !comparedAlloc && alloc ? BRIGHT_RED : WHITE;
    let base: ArtSprite | undefined;
    let overlay: ArtSprite | undefined;
    let overlayR = size.overlay;
    if (n.type === 'AscendClassStart') {
      overlay = art.ascendancyMiddle;
    } else if (n.type === 'Socket' || n.containJewelSocket) {
      base = na.overlay?.[state];
      /* PassiveTreeView.lua:961-969 */
      const jewel = s.jewels?.get(n.id);
      if (isAlloc && jewel) overlay = art.atlas.byName.get(jewel.unique && art.atlas.byName.has(jewel.title) ? jewel.title : jewel.baseName);
      overlayR = size.overlay;
    } else {
      /* PassiveTreeView.lua:1074-1088 */
      if (na.effect && !s.heat) {
        const r = na.effectR ?? na.effect.width;
        const rh = na.effectR ?? na.effect.height;
        asset('under', na.effect, n.x, n.y, r, rh, isAlloc || onPath(n.id) ? WHITE : [1, 1, 1, 0.15]);
      }
      const icon = s.icons?.get(n.id);
      base = (icon && art.atlas.byName.get(icon)) || na.icon;
      overlay = na.overlay?.[state];
    }
    let any = false;
    /* PassiveTreeView.lua:1091-1117 */
    if (base) {
      let tint: Tint = colour;
      if (n.type === 'Socket') tint = s.hoverDep.has(n.id) ? [1, 0, 0, 1] : colour;
      else if (!alloc && !s.heat) tint = HALF_GRAY;
      any = asset('over', base, n.x, n.y, size.base, size.base, tint) || any;
      // PassiveTreeView.lua:1113-1115 sets white back after an unallocated node's half gray
      if (n.type !== 'Socket' && !alloc && !s.heat) colour = WHITE;
    }
    /* PassiveTreeView.lua:1119-1180 */
    if (overlay) {
      const mode = nodeWillAllocateWithAllocMode(s, n) ? (s.specAllocMode ?? 0) : (s.allocMode.get(n.id) ?? 0);
      let tint: Tint = colour;
      /* PassiveTreeView.lua:937 */
      if (!s.heat && !s.compared && mode > 0 && !nodeWillChangeAllocMode(s, n.id)) tint = opaque(mode === 1 ? COLOURS.weaponSet1 : COLOURS.weaponSet2);
      if (n.type !== 'ClassStart' && n.type !== 'AscendClassStart' && s.hovered !== undefined && s.hovered !== n.id) {
        if (s.hoverDep.has(n.id) && s.hovered === UNSEEN_PATH && !s.alloc.has(UNSEEN_PATH)) tint = BRIGHT_GREEN;
        else if (s.hoverDep.has(n.id)) tint = [1, 0, 0, 1];
        else if (inRadiusColour.has(n.id)) tint = opaque(inRadiusColour.get(n.id)!);
      }
      if (!alloc && !s.heat && n.type === 'AscendClassStart') tint = HALF_GRAY;
      any = asset('over', overlay, n.x, n.y, overlayR, overlayR, tint) || any;
    }
    if (any) drawn.add(n.id);
  }
  return drawn;
};

export const fillScene2 = (scene: Scene2, s: Scene2State, ready: (sheet: number) => boolean = () => true) => {
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
  const w = writer(scene, ready);
  /* PassiveTreeView.lua:781-822: a connector draws its line art where the art has loaded, else a vector stroke */
  scene.connectors.forEach((c, i) => {
    /* PassiveTreeView.lua:793-797 */
    if (!unlocked(s, scene.byId.get(c.a)) || !unlocked(s, scene.byId.get(c.b))) return;
    const own = connectorState(s, c.a, c.b);
    const { state, tint } = connectorTint(s, c, own);
    const quads = scene.art?.connectors[i]?.[state];
    if (quads && quads.every((q) => ready(q.sprite.sheet))) {
      for (const q of quads) w.quad('lines', q.sprite, q.o[0], q.o[1], q.u[0], q.u[1], q.v[0], q.v[1], tint, q.clip, q.repeat, q.inner);
      return;
    }
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
    if (tint === BRIGHT_GREEN || tint === BRIGHT_RED) colour = [tint[0], tint[1], tint[2]];
    if (s.hoverDep.has(c.a) && s.hoverDep.has(c.b)) colour = COLOURS.depends;
    else if (c.ascendancy && c.ascendancy !== s.ascendClassBaseName) colour = state === 'Normal' ? COLOURS.otherAscendancy : colour;
    stroke([...c.geom, c.kind, state === 'Normal' ? STROKE_WIDTH.normal : STROKE_WIDTH.active, ...colour, 1]);
  });
  scene.counts.connectorStrokes = o / STROKE2_FLOATS;
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

  const drawn = scene.art ? fillArt2(scene, s, w, inRadiusColour) : undefined;
  scene.counts.under = w.out.under / SPRITE2_FLOATS;
  scene.counts.lines = w.out.lines / SPRITE2_FLOATS;
  scene.counts.over = w.out.over / SPRITE2_FLOATS;

  o = 0;
  for (const p of scene.nodes) {
    const n = p.node;
    if (drawn?.has(n.id) || !unlocked(s, n)) continue;
    const alloc = s.alloc.has(n.id);
    const shown = alloc || n.id === s.hovered;
    let ring = shown ? COLOURS.ringAlloc : s.hoverPath.has(n.id) ? COLOURS.ringPath : COLOURS.ringUnalloc;
    /* PassiveTreeView.lua:936-937, 1120-1121 */
    const mode = nodeWillAllocateWithAllocMode(s, n) ? (s.specAllocMode ?? 0) : (s.allocMode.get(n.id) ?? 0);
    if (!s.compared && mode > 0 && !nodeWillChangeAllocMode(s, n.id)) ring = mode === 1 ? COLOURS.weaponSet1 : COLOURS.weaponSet2;
    /* PassiveTreeView.lua:1057-1067 */
    if (s.compared?.has(n.id) && !alloc) ring = [0, 1, 0];
    else if (s.compared && !s.compared.has(n.id) && alloc) ring = [1, 0, 0];
    if (s.hovered !== undefined && n.id !== s.hovered && n.type !== 'ClassStart' && n.type !== 'AscendClassStart') {
      if (s.hoverDep.has(n.id)) ring = COLOURS.depends;
      else ring = inRadiusColour.get(n.id) ?? ring;
    }
    const heat = s.heat && !alloc && n.type !== 'AscendClassStart' ? (s.heat.get(n.id) ?? [0, 0, 0]) : undefined;
    const base = heat ?? fillOf(n);
    const k = shown || heat ? 1 : s.hoverPath.has(n.id) ? 0.7 : 0.4;
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
