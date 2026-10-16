import type { Node, Sprite, Tree, TreeClusterNode } from '$lib/skill_tree/types';
import {
  ascendancyGroupPositionOffsets,
  ascendancyGroups,
  ascendancyStartGroups,
  bloodlineSprites,
  classStartGroups,
  drawnGroups,
  drawnNodes,
  inverseSpritesActive,
  inverseSpritesInactive,
  inverseSpritesOther,
  masterySprites,
  tattooSprites
} from '$lib/skill_tree';
import { drawnByPoB, nodeCentre, orbitAngleAt, type Point } from '$lib/skill_tree/geometry';

const DRAW_SCALING = 2.6;

export const Variant = { Plain: 0, Crop: 1, CropDim: 2 } as const;
type Variant = (typeof Variant)[keyof typeof Variant];

// PassiveTreeView.lua:693-699
export const Style = { Base: 0, Hover: 1, Active: 2, ActiveCompared: 3, ActiveOwn: 4, SplitBase: 5, SplitHover: 6, SplitActive: 7, Depends: 8 } as const;

export const STYLE_COLOURS = ['#524518', '#c89c01', '#e9deb6', '#00de00', '#e90000', '#523a04', '#c88300', '#e9ba1e', '#e90000'];
export const STYLE_WIDTHS = [6, 6, 12, 12, 12, 6, 6, 12, 12];

export const SPRITE_FLOATS = 13;

export const packTint = (rgb: [number, number, number] | undefined) => {
  if (!rgb) return 0;
  const c = (v: number) => Math.round(Math.min(Math.max(v, 0), 1) * 255);
  return 1 + ((c(rgb[0]) << 16) | (c(rgb[1]) << 8) | c(rgb[2]));
};

export const unpackTint = (t: number): [number, number, number] | undefined =>
  t > 0 ? [(((t - 1) >> 16) & 255) / 255, (((t - 1) >> 8) & 255) / 255, ((t - 1) & 255) / 255] : undefined;
export const STROKE_FLOATS = 11;

type Connection = { a: number; b: number; geom: number[]; arc: boolean; bounds: number[] };

type Placed = { id: number; node: Node; at: Point };

/* PassiveTreeView.lua:648-656 */
export type ClusterLayer = { groups: { x: number; y: number; orbit: number; compare?: boolean }[]; nodes: Placed[]; connections: Connection[] };

export type SceneState = {
  active: Set<number>;
  hoverPath: Set<number>;
  hovered: number | undefined;
  className: string | undefined;
  ascendancy: string | undefined;
  secondaryAscendancy?: string;
  /* PassiveTreeView.lua:840-846 */
  socketArt?: Map<number, string>;
  cluster?: ClusterLayer;
  /* PassiveTreeView.lua:906-949 */
  tint?: Map<number, [number, number, number]>;
  /* CalcSetup.lua:1285-1315 */
  granted?: Set<number>;
  /* PassiveTreeView.lua:686-699, 811 */
  compared?: Set<number>;
  /* PassiveTreeView.lua:679-681 */
  split?: Set<number>;
  /* PassiveTreeView.lua:831-833 */
  heatMap?: boolean;
  /* PassiveTreeView.lua:847-877 */
  tattoos?: Map<number, { icon: string; effect: string }>;
  /* PassiveTreeView.lua:372-375 */
  depends?: Set<number>;
};

export type Scene = {
  tree: Tree;
  sheets: string[];
  sheetOfUrl: Map<string, number>;
  nodes: { id: number; node: Node; at: Point }[];
  connections: Connection[];
  background: Float32Array;
  strokes: Float32Array;
  sprites: Float32Array;
  counts: { background: number; strokes: number; sprites: number };
};

const assetPath = (url: string) => 'tree/assets/' + new URL(url).pathname.split('/').pop();

const arcBounds = (cx: number, cy: number, r: number, a0: number, sweep: number) => {
  const xs = [Math.cos(a0), Math.cos(a0 + sweep)];
  const ys = [Math.sin(a0), Math.sin(a0 + sweep)];
  for (let k = Math.ceil(a0 / (Math.PI / 2)); k * (Math.PI / 2) <= a0 + sweep; k++) {
    xs.push(Math.cos(k * (Math.PI / 2)));
    ys.push(Math.sin(k * (Math.PI / 2)));
  }
  return [cx + r * Math.min(...xs), cy + r * Math.min(...ys), cx + r * Math.max(...xs), cy + r * Math.max(...ys)];
};

const connect = (tree: Tree, aId: number, node: Node, p: Point, bId: number, targetNode: Node, q: Point): Connection => {
  if (node.group != targetNode.group || node.orbit != targetNode.orbit) {
    return { a: aId, b: bId, geom: [p.x, p.y, q.x, q.y], arc: false, bounds: [0, 0, 0, 0] };
  }
  const angle = orbitAngleAt(tree, node.orbit!, node.orbitIndex!);
  const targetAngle = orbitAngleAt(tree, targetNode.orbit!, targetNode.orbitIndex!);
  let a = Math.PI / 180 - (Math.PI / 180) * angle;
  let b = Math.PI / 180 - (Math.PI / 180) * targetAngle;
  a -= Math.PI / 2;
  b -= Math.PI / 2;
  const diff = Math.abs(Math.max(a, b) - Math.min(a, b));
  const finalA = diff > Math.PI ? Math.max(a, b) : Math.min(a, b);
  const finalB = diff > Math.PI ? Math.min(a, b) : Math.max(a, b);
  const group = drawnGroups.get(node.group!) ?? tree.groups[node.group!];
  const cx = ((node.ascendancyName && ascendancyGroupPositionOffsets[node.ascendancyName]?.x) || 0) + group.x;
  const cy = ((node.ascendancyName && ascendancyGroupPositionOffsets[node.ascendancyName]?.y) || 0) + group.y;
  const r = tree.constants.orbitRadii[node.orbit!];
  const sweep = (((finalB - finalA) % (2 * Math.PI)) + 2 * Math.PI) % (2 * Math.PI);
  return { a: aId, b: bId, geom: [cx, cy, r, finalA, finalB], arc: true, bounds: arcBounds(cx, cy, r, finalA, sweep) };
};

export const clusterNodeAsTreeNode = (c: TreeClusterNode): Node => ({
  skill: c.id,
  name: c.name,
  icon: c.icon,
  stats: c.stats,
  group: c.group,
  orbit: c.orbit,
  orbitIndex: c.orbitIndex,
  out: c.linked.map(String),
  isNotable: c.type === 'Notable' || undefined,
  isKeystone: c.type === 'Keystone' || undefined,
  isJewelSocket: c.type === 'Socket' || undefined,
  expansionJewel: c.type === 'Socket' ? { size: 0, index: 0, proxy: '' } : undefined,
  isMastery: c.type === 'Mastery' || undefined
});

/* PassiveSpec.lua:1885-2271, PassiveTreeView.lua:645-647 */
export const buildClusterLayer = (
  tree: Tree,
  cluster: TreeClusterNode[],
  treeAt: (id: number) => Placed | undefined,
  compareOnly: Set<number> = new Set()
): ClusterLayer => {
  const nodes = cluster.map((c) => ({ id: c.id, node: clusterNodeAsTreeNode(c), at: { x: c.x, y: c.y } }));
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const groups = new Map<number, { x: number; y: number; orbit: number; compare?: boolean }>();
  for (const c of cluster) {
    if (c.type === 'Mastery' || c.orbit === 0) continue;
    const g = tree.groups[c.group];
    if (!g) continue;
    const was = groups.get(c.group);
    const compare = (was?.compare ?? true) && compareOnly.has(c.id);
    groups.set(c.group, { x: g.x, y: g.y, orbit: Math.max(c.orbit, was?.orbit ?? 0), ...(compare && { compare }) });
  }
  const connections: Connection[] = [];
  const seen = new Set<string>();
  for (const n of nodes) {
    for (const oid of cluster.find((c) => c.id === n.id)!.linked) {
      const other = byId.get(oid) ?? treeAt(oid);
      const joined = Math.min(oid, n.id) + ':' + Math.max(oid, n.id);
      if (!other || seen.has(joined)) continue;
      seen.add(joined);
      connections.push(connect(tree, n.id, n.node, n.at, other.id, other.node, other.at));
    }
  }
  return { groups: [...groups.values()], nodes, connections };
};

export const buildScene = (tree: Tree): Scene => {
  const sheets: string[] = [];
  const sheetOfUrl = new Map<string, number>();
  for (const src of [inverseSpritesInactive, inverseSpritesActive, inverseSpritesOther, masterySprites, tattooSprites, ...Object.values(bloodlineSprites)]) {
    for (const s of Object.values(src)) {
      if (sheetOfUrl.has(s.filename)) continue;
      const path = assetPath(s.filename);
      let i = sheets.indexOf(path);
      if (i < 0) i = sheets.push(path) - 1;
      sheetOfUrl.set(s.filename, i);
    }
  }

  const nodes = [...drawnNodes.entries()]
    .filter(([, node]) => drawnByPoB(tree, node))
    .map(([id, node]) => ({ id, node, at: nodeCentre(tree, ascendancyGroupPositionOffsets, node) ?? { x: 0, y: 0 } }));
  const posOf = new Map(nodes.map((n) => [n.id, n.at]));

  const connections: Connection[] = [];
  const connected: Record<string, boolean> = {};
  for (const [nNodeId, node] of drawnNodes) {
    if (node.classStartIndex !== undefined) continue;
    if (!posOf.has(nNodeId)) continue;
    for (const o of node.out ?? []) {
      const oid = parseInt(o);
      if (!drawnNodes.get(oid) || !posOf.has(oid)) continue;
      const joined = Math.min(oid, nNodeId) + ':' + Math.max(oid, nNodeId);
      if (joined in connected) continue;
      connected[joined] = true;
      const targetNode = drawnNodes.get(oid)!;
      if (targetNode.isMastery || node.ascendancyName !== targetNode.ascendancyName || targetNode.classStartIndex !== undefined) continue;

      connections.push(connect(tree, nNodeId, node, posOf.get(nNodeId)!, oid, targetNode, posOf.get(oid)!));
    }
  }

  return {
    tree,
    sheets,
    sheetOfUrl,
    nodes,
    connections,
    background: new Float32Array((drawnGroups.size + 1 + nodes.filter((n) => n.node.isMastery).length) * 2 * SPRITE_FLOATS),
    strokes: new Float32Array(connections.length * STROKE_FLOATS),
    sprites: new Float32Array(nodes.length * 2 * SPRITE_FLOATS),
    counts: { background: 0, strokes: 0, sprites: 0 }
  };
};

export const classArt = (tree: Tree, className: string | undefined) => {
  const classIndex = tree.classes.findIndex((c) => c.name === className);
  return className && classIndex in tree.extraImages ? tree.extraImages[classIndex] : undefined;
};

// src/Classes/PassiveTree.lua:446-492
// PassiveTreeView.lua:852
const NODE_OVERLAY: Record<string, Record<string, string>> = {
  Normal: {
    alloc: 'PSSkillFrameActive',
    path: 'PSSkillFrameHighlighted',
    unalloc: 'PSSkillFrame',
    allocAscend: 'AscendancyFrameSmallAllocated',
    pathAscend: 'AscendancyFrameSmallCanAllocate',
    unallocAscend: 'AscendancyFrameSmallNormal'
  },
  Notable: {
    alloc: 'NotableFrameAllocated',
    path: 'NotableFrameCanAllocate',
    unalloc: 'NotableFrameUnallocated',
    allocAscend: 'AscendancyFrameLargeAllocated',
    pathAscend: 'AscendancyFrameLargeCanAllocate',
    unallocAscend: 'AscendancyFrameLargeNormal',
    allocBlighted: 'BlightedNotableFrameAllocated',
    pathBlighted: 'BlightedNotableFrameCanAllocate',
    unallocBlighted: 'BlightedNotableFrameUnallocated'
  },
  Keystone: {
    alloc: 'KeystoneFrameAllocated',
    path: 'KeystoneFrameCanAllocate',
    unalloc: 'KeystoneFrameUnallocated',
    allocBlighted: 'KeystoneFrameAllocated',
    pathBlighted: 'KeystoneFrameCanAllocate',
    unallocBlighted: 'KeystoneFrameUnallocated'
  },
  Socket: {
    alloc: 'JewelFrameAllocated',
    path: 'JewelFrameCanAllocate',
    unalloc: 'JewelFrameUnallocated',
    allocAlt: 'JewelSocketAltActive',
    pathAlt: 'JewelSocketAltCanAllocate',
    unallocAlt: 'JewelSocketAltNormal'
  }
};

export const fillScene = (scene: Scene, state: SceneState, sheetReady: (i: number) => boolean, art?: { sheet: number; w: number; h: number }) => {
  const { tree } = scene;

  const connections = state.cluster ? [...scene.connections, ...state.cluster.connections] : scene.connections;
  const clusterNodes = state.cluster?.nodes ?? [];
  const backgrounds = drawnGroups.size + 1 + scene.nodes.filter((p) => p.node.isMastery).length + (state.cluster?.groups.length ?? 0);
  if (scene.background.length < backgrounds * 2 * SPRITE_FLOATS) scene.background = new Float32Array(backgrounds * 2 * SPRITE_FLOATS);
  if (scene.strokes.length < connections.length * STROKE_FLOATS) scene.strokes = new Float32Array(connections.length * STROKE_FLOATS);
  if (scene.sprites.length < (scene.nodes.length + clusterNodes.length) * 2 * SPRITE_FLOATS)
    scene.sprites = new Float32Array((scene.nodes.length + clusterNodes.length) * 2 * SPRITE_FLOATS);

  let n = 0;
  const bg = scene.background;
  const put = (arr: Float32Array, sprite: Sprite | undefined, name: string, x: number, y: number, variant: Variant, mirror: boolean, tint = 0) => {
    if (!sprite) return;
    const sheet = scene.sheetOfUrl.get(sprite.filename)!;
    if (!sheetReady(sheet)) return;
    const c = sprite.coords[name];
    const w = c.w * DRAW_SCALING;
    const h = c.h * DRAW_SCALING;
    const quads = mirror
      ? [
          [x - w / 2, y - h, w, 0, 0, h],
          [x + w / 2, y + h, -w, 0, 0, -h]
        ]
      : [[x - w / 2, y - h / 2, w, 0, 0, h]];
    for (const q of quads) {
      const o = n * SPRITE_FLOATS;
      arr.set(q, o);
      arr[o + 6] = c.x;
      arr[o + 7] = c.y;
      arr[o + 8] = c.w;
      arr[o + 9] = c.h;
      arr[o + 10] = sheet;
      arr[o + 11] = variant;
      arr[o + 12] = tint;
      n++;
    }
  };

  const img = classArt(tree, state.className);
  if (img && art) {
    const o = 0;
    bg.set([img.x, img.y, art.w * DRAW_SCALING * 0.5, 0, 0, art.h * DRAW_SCALING * 0.5, 0, 0, art.w, art.h, art.sheet, Variant.Plain, 0], o);
    n = 1;
  }

  const other = (name: string) => inverseSpritesOther[name];
  for (const [groupId, group] of drawnGroups) {
    const posX = ((groupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[groupId]]?.x) || 0) + group.x;
    const posY = ((groupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[groupId]]?.y) || 0) + group.y;
    const maxOrbit = Math.max(...group.orbits);
    // src/Classes/PassiveTreeView.lua:641
    if (group.isProxy) continue;
    if (groupId in classStartGroups) {
      const cls = tree.classes[classStartGroups[groupId]].name;
      if (state.className === cls) {
        const name = 'center' + cls.toLowerCase();
        put(bg, other(name), name, posX, posY, Variant.Plain, false);
      } else {
        put(bg, other('PSStartNodeBackgroundInactive'), 'PSStartNodeBackgroundInactive', posX, posY, Variant.CropDim, false);
      }
    } else if (groupId in ascendancyGroups) {
      if (ascendancyStartGroups.has(groupId)) {
        const name = 'Classes' + ascendancyGroups[groupId];
        // src/Classes/PassiveTreeView.lua:572
        const lit = state.ascendancy === ascendancyGroups[groupId] || state.secondaryAscendancy === ascendancyGroups[groupId];
        put(bg, other(name), name, posX, posY, lit ? Variant.Crop : Variant.CropDim, false);
      }
    } else if (maxOrbit == 1) {
      put(bg, other('PSGroupBackground1'), 'PSGroupBackground1', posX, posY, Variant.Plain, false);
    } else if (maxOrbit == 2) {
      put(bg, other('PSGroupBackground2'), 'PSGroupBackground2', posX, posY, Variant.Plain, false);
    } else if (maxOrbit == 3 || group.orbits.length > 1) {
      put(bg, other('PSGroupBackground3'), 'PSGroupBackground3', posX, posY, Variant.Plain, true);
    }
  }
  // src/Classes/PassiveTreeView.lua:630-636, 645-647
  for (const g of state.cluster?.groups ?? []) {
    const name = g.orbit >= 3 ? 'GroupBackgroundLargeHalfAlt' : g.orbit === 2 ? 'GroupBackgroundMediumAlt' : 'GroupBackgroundSmallAlt';
    // PassiveTreeView.lua:648-656
    put(bg, other(name), name, g.x, g.y, Variant.Plain, g.orbit >= 3, g.compare ? packTint([0, 0.6, 0]) : 0);
  }
  // src/Classes/PassiveTreeView.lua:855-859, 951-956
  // PassiveTreeView.lua:811
  const isAlloc = (id: number) => state.active.has(id) || !!state.granted?.has(id) || !!state.compared?.has(id);
  const tattooArt = (sheet: string, name: string | undefined) => (name && tattooSprites[sheet]?.coords[name] ? tattooSprites[sheet] : undefined);
  for (const { id, node, at } of scene.nodes) {
    const tattoo = state.tattoos?.get(id);
    const effect = tattoo?.effect && other(tattoo.effect)?.coords[tattoo.effect] ? tattoo.effect : node.activeEffectImage;
    if (node.isMastery && effect && isAlloc(id)) put(bg, other(effect), effect, at.x, at.y, Variant.Plain, false, packTint(state.tint?.get(id)));
    else if (!node.isMastery && tattoo)
      put(bg, tattooArt('tattooActiveEffect', tattoo.effect), tattoo.effect, at.x, at.y, Variant.Plain, false, packTint(state.tint?.get(id)));
  }
  scene.counts.background = n;

  const st = scene.strokes;
  connections.forEach((c, i) => {
    let style: number =
      state.active.has(c.a) && state.active.has(c.b) ? Style.Active : state.hoverPath.has(c.a) && state.hoverPath.has(c.b) ? Style.Hover : Style.Base;
    if (state.compared) {
      const baseActive = state.compared.has(c.a) && state.compared.has(c.b);
      if (baseActive && style !== Style.Active) style = Style.ActiveCompared;
      else if (!baseActive && style === Style.Active) style = Style.ActiveOwn;
    }
    if (style <= Style.Active && state.split?.has(c.a) && state.split.has(c.b)) style += Style.SplitBase;
    if (state.depends?.has(c.a) && state.depends.has(c.b)) style = Style.Depends;
    const o = i * STROKE_FLOATS;
    st[o] = c.geom[0];
    st[o + 1] = c.geom[1];
    st[o + 2] = c.geom[2];
    st[o + 3] = c.geom[3];
    st[o + 4] = c.arc ? (((c.geom[4] - c.geom[3]) % (2 * Math.PI)) + 2 * Math.PI) % (2 * Math.PI) : 0;
    st[o + 5] = style;
    st[o + 6] = c.arc ? 1 : 0;
    st.set(c.bounds, o + 7);
  });
  scene.counts.strokes = connections.length;

  n = 0;
  const sp = scene.sprites;
  for (const { id, node, at } of [...scene.nodes, ...clusterNodes]) {
    const active = isAlloc(id);
    const lit = active || state.hoverPath.has(id) || state.hovered === id;
    // PassiveTreeView.lua:840-849
    const overlayState = active || state.hovered === id || state.heatMap ? 'alloc' : state.hoverPath.has(id) ? 'path' : 'unalloc';
    const tint = packTint(state.tint?.get(id));
    // PassiveTreeView.lua:960-963, 972-976
    const frameTint = state.depends?.has(id) && state.hovered !== id ? packTint([1, 0, 0]) : tint;
    const tattoo = state.tattoos?.get(id);
    const icon = () => {
      // PassiveTreeView.lua:877
      const sheet = (node.isKeystone ? 'keystone' : node.isNotable ? 'notable' : 'normal') + (active ? 'Active' : 'Inactive');
      const tattooIcon = tattooArt(sheet, tattoo?.icon);
      if (tattooIcon) return put(sp, tattooIcon, tattoo!.icon, at.x, at.y, Variant.Plain, false, tint);
      const src = active ? inverseSpritesActive : inverseSpritesInactive;
      put(sp, src[node.icon!], node.icon!, at.x, at.y, Variant.Plain, false, tint);
    };
    const frame = (name: string) => put(sp, other(name), name, at.x, at.y, Variant.Plain, false, frameTint);
    // src/Classes/PassiveTreeView.lua:821-828
    const ascFrame = (name: string) => {
      const bloodline = bloodlineSprites[node.ascendancyName!]?.[name];
      put(sp, bloodline ?? other(name), name, at.x, at.y, Variant.Plain, false, frameTint);
    };
    if (node.isAscendancyStart) {
      ascFrame('AscendancyMiddle');
    } else if (node.isKeystone) {
      icon();
      frame(NODE_OVERLAY.Keystone[overlayState + (node.isBlighted ? 'Blighted' : '')]);
    } else if (node.isNotable) {
      icon();
      if (node.ascendancyName) ascFrame(NODE_OVERLAY.Notable[overlayState + 'Ascend']);
      else frame(NODE_OVERLAY.Notable[overlayState + (node.isBlighted ? 'Blighted' : '')]);
    } else if (node.isJewelSocket) {
      frame(NODE_OVERLAY.Socket[overlayState + (node.expansionJewel ? 'Alt' : '')]);
      const jewel = active ? state.socketArt?.get(id) : undefined;
      if (jewel) frame(jewel);
    } else if (node.isMastery && !node.inactiveIcon) {
      // src/Classes/PassiveTreeView.lua:867-870
      put(sp, masterySprites[node.icon!], node.icon!, at.x, at.y, Variant.Plain, false, tint);
    } else if (node.isMastery) {
      // PassiveTreeView.lua:851-866
      const src = lit ? inverseSpritesActive : inverseSpritesInactive;
      const name = tattoo?.icon && src[tattoo.icon]?.coords[tattoo.icon] ? tattoo.icon : lit ? node.activeIcon! : node.inactiveIcon!;
      put(sp, src[name], name, at.x, at.y, Variant.Plain, false, tint);
    } else {
      icon();
      if (node.ascendancyName) ascFrame(NODE_OVERLAY.Normal[overlayState + 'Ascend']);
      else frame(NODE_OVERLAY.Normal[overlayState]);
    }
  }
  scene.counts.sprites = n;
};

export const sceneDrawList = (scene: Scene, scaling: number, artPath: string | undefined) => {
  const items: unknown[] = [];
  const sprites = (arr: Float32Array, count: number, first: number) => {
    for (let i = 0; i < count; i++) {
      const o = i * SPRITE_FLOATS;
      const sheet = arr[o + 10];
      const src = i < first ? artPath! : scene.sheets[sheet];
      const v = ['', 'crop', 'crop-dim'][arr[o + 11]];
      const t = unpackTint(arr[o + 12]);
      items.push({ k: 'sprite', src, s: [...arr.subarray(o + 6, o + 10)], v, q: [...arr.subarray(o, o + 6)], ...(t ? { t } : {}) });
    }
  };
  sprites(scene.background, scene.counts.background, artPath ? 1 : 0);
  for (let i = 0; i < scene.counts.strokes; i++) {
    const o = i * STROKE_FLOATS;
    const st = scene.strokes;
    const style = STYLE_COLOURS[st[o + 5]];
    const w = STYLE_WIDTHS[st[o + 5]];
    if (st[o + 6] === 0) items.push({ k: 'line', p: [st[o], st[o + 1], st[o + 2], st[o + 3]], style, w });
    else items.push({ k: 'arc', c: [st[o], st[o + 1]], r: st[o + 2] + scaling, a: [st[o + 3], st[o + 3] + st[o + 4]], style, w });
  }
  sprites(scene.sprites, scene.counts.sprites, 0);
  return items;
};
