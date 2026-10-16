import { describe, expect, it, vi } from 'vitest';
import { brotliDecompressSync } from 'node:zlib';
import { readFileSync } from 'node:fs';

import { cachedAsset } from '../../../../e2e/support/data_cache';
import type { Tree, TreeClusterNode } from '$lib/skill_tree/types';

const rawJson = (v: string) => {
  const a = cachedAsset(`https://data.pob.gg/data/${v.replace('_', '.')}/tree/data.json.br`);
  if (a.status !== 200) throw new Error(`REFUSING: no ${v} tree in the bundle cache`);
  return brotliDecompressSync(a.body).toString('utf8');
};

vi.mock('$lib/go/worker', () => ({ syncWrap: { GetTree: (v: string) => Promise.resolve(rawJson(v)) }, syncWorker: undefined }));

const st = await import('$lib/skill_tree');
const { buildScene, buildClusterLayer, fillScene, sceneDrawList, packTint, unpackTint, SPRITE_FLOATS, STROKE_FLOATS, STYLE_COLOURS, STYLE_WIDTHS, Style } =
  await import('./scene');

type OracleNode = {
  id: number;
  type: TreeClusterNode['type'];
  name: string;
  icon: string;
  o: number;
  oidx: number;
  x: number;
  y: number;
  linked: number[];
  proxy: number;
};
const oracle = readFileSync(new URL('../../../../../testdata/cluster-subgraph/oracle.jsonl', import.meta.url), 'utf8')
  .trim()
  .split('\n')
  .map((l) => JSON.parse(l) as { file: string; nodes: OracleNode[] });

await st.loadSkillTree('3_29');
let tree: Tree | undefined;
st.skillTree.subscribe((t) => (tree = t))();

const asCluster = (t: Tree, n: OracleNode): TreeClusterNode => ({
  id: n.id,
  type: n.type,
  name: n.name,
  icon: n.icon,
  stats: [],
  group: t.nodes[n.proxy].group!,
  orbit: n.o,
  orbitIndex: n.oidx,
  x: n.x,
  y: n.y,
  linked: n.linked
});

describe('the tree scene', () => {
  it('leaves out proxy groups, proxy nodes and the cluster sockets inside them (PassiveSpec.lua:44)', () => {
    const t = tree!;
    const scene = buildScene(t);
    const hidden = scene.nodes.filter(({ node }) => node.isProxy || t.groups[node.group!]?.isProxy || node.expansionJewel?.parent);
    expect(hidden.map((n) => n.id)).toEqual([]);
    expect(scene.nodes.length).toBeGreaterThan(1000);
    expect(
      [...st.drawnNodes.values()].some((n) => n.isProxy),
      'the 3.29 tree has proxy nodes to leave out'
    ).toBe(true);
  });

  for (const build of oracle) {
    it(`draws ${build.file}'s cluster subgraphs`, () => {
      const t = tree!;
      const scene = buildScene(t);
      const placed = new Map(scene.nodes.map((n) => [n.id, n]));
      const cluster = build.nodes.map((n) => asCluster(t, n));
      const layer = buildClusterLayer(t, cluster, (id) => placed.get(id));

      const links = new Set<string>();
      for (const n of build.nodes) for (const o of n.linked) links.add(Math.min(n.id, o) + ':' + Math.max(n.id, o));
      expect(layer.connections.map((c) => Math.min(c.a, c.b) + ':' + Math.max(c.a, c.b)).sort()).toEqual([...links].sort());
      expect(layer.groups.length).toBe(new Set(build.nodes.filter((n) => n.o > 0).map((n) => n.proxy)).size);

      const ready = () => true;
      fillScene(scene, { active: new Set(), hoverPath: new Set(), hovered: undefined, className: undefined, ascendancy: undefined }, ready);
      const base = { strokes: scene.counts.strokes, sprites: scene.counts.sprites, background: scene.counts.background };
      fillScene(scene, { active: new Set(), hoverPath: new Set(), hovered: undefined, className: undefined, ascendancy: undefined, cluster: layer }, ready);
      expect(scene.counts.strokes - base.strokes).toBe(links.size);
      expect(scene.counts.background - base.background).toBeGreaterThanOrEqual(layer.groups.length);
      const at = new Set<string>();
      for (let i = base.sprites; i < scene.counts.sprites; i++) {
        const o = i * SPRITE_FLOATS;
        at.add(`${Math.round(scene.sprites[o] + scene.sprites[o + 2] / 2)},${Math.round(scene.sprites[o + 1] + scene.sprites[o + 5] / 2)}`);
      }
      const missing = build.nodes.filter((n) => !at.has(`${Math.round(n.x)},${Math.round(n.y)}`)).map((n) => `${n.id} ${n.type} ${n.icon}`);
      expect(missing).toEqual([]);
    });
  }

  // PassiveTreeView.lua:648-656
  it("tints the compared tree's own subgraph backgrounds green", () => {
    const t = tree!;
    const scene = buildScene(t);
    const placed = new Map(scene.nodes.map((n) => [n.id, n]));
    const cluster = oracle[0].nodes.map((n) => asCluster(t, n));
    const state = { active: new Set<number>(), hoverPath: new Set<number>(), hovered: undefined, className: undefined, ascendancy: undefined };
    const tints = (layer: ReturnType<typeof buildClusterLayer>) => {
      fillScene(scene, { ...state, cluster: layer }, () => true);
      return Array.from({ length: scene.counts.background }, (_, i) => scene.background[i * SPRITE_FLOATS + 12]).filter((x) => x > 0);
    };
    const own = buildClusterLayer(t, cluster, (id) => placed.get(id));
    expect(own.groups.length).toBeGreaterThan(0);
    expect(tints(own)).toEqual([]);
    const compared = buildClusterLayer(t, cluster, (id) => placed.get(id), new Set(cluster.map((c) => c.id)));
    const halves = compared.groups.reduce((k, g) => k + (g.orbit >= 3 ? 2 : 1), 0);
    expect(tints(compared)).toEqual(Array.from({ length: halves }, () => packTint([0, 0.6, 0])));
  });

  // PassiveTreeView.lua:906-949
  it('draws a tinted node with its colour on its artwork and frame, every other sprite white', () => {
    const scene = buildScene(tree!);
    const notable = scene.nodes.find((n) => n.node.isNotable && !n.node.ascendancyName)!;
    const state = { active: new Set<number>(), hoverPath: new Set<number>(), hovered: undefined, className: undefined, ascendancy: undefined };
    fillScene(scene, { ...state, tint: new Map([[notable.id, [0, 1, 0] as [number, number, number]]]) }, () => true);
    const tinted = sceneDrawList(scene, 1, undefined).filter((i) => (i as { t?: number[] }).t) as { t: number[]; q: number[] }[];
    expect(tinted.length).toBe(2);
    for (const i of tinted) {
      expect(i.t).toEqual([0, 1, 0]);
      expect(Math.round(i.q[0] + i.q[2] / 2)).toBe(Math.round(notable.at.x));
    }
    fillScene(scene, state, () => true);
    expect(sceneDrawList(scene, 1, undefined).some((i) => (i as { t?: number[] }).t)).toBe(false);
    expect(unpackTint(packTint([1, 0.5, 0]))).toEqual([1, 128 / 255, 0]);
    expect(packTint(undefined)).toBe(0);
  });

  const empty = () => ({ active: new Set<number>(), hoverPath: new Set<number>(), hovered: undefined, className: undefined, ascendancy: undefined });
  const spritesAt = (scene: ReturnType<typeof buildScene>, at: { x: number; y: number }) =>
    (sceneDrawList(scene, 1, undefined) as { k: string; src?: string; s?: number[]; q?: number[] }[])
      .filter((i) => i.k === 'sprite' && Math.round(i.q![0] + i.q![2] / 2) === Math.round(at.x) && Math.round(i.q![1] + i.q![5] / 2) === Math.round(at.y))
      .map((i) => `${i.src} ${i.s!.join(',')}`);

  // PassiveTreeView.lua:811
  it('draws a node an item grants, or the compared tree allocates, as allocated', () => {
    const scene = buildScene(tree!);
    const notable = scene.nodes.find((n) => n.node.isNotable && !n.node.ascendancyName)!;
    fillScene(scene, empty(), () => true);
    const unalloc = spritesAt(scene, notable.at);
    fillScene(scene, { ...empty(), active: new Set([notable.id]) }, () => true);
    const alloc = spritesAt(scene, notable.at);
    expect(alloc, 'control: allocating changes the sprites').not.toEqual(unalloc);
    fillScene(scene, { ...empty(), granted: new Set([notable.id]) }, () => true);
    expect(spritesAt(scene, notable.at)).toEqual(alloc);
    fillScene(scene, { ...empty(), compared: new Set([notable.id]) }, () => true);
    expect(spritesAt(scene, notable.at)).toEqual(alloc);
  });

  // PassiveTreeView.lua:847-877
  it("draws a tattoo's and a runegraft's art in place of the node's", () => {
    const scene = buildScene(tree!);
    expect(scene.sheets.length).toBeLessThan(16);
    const str = scene.nodes.find((n) => n.node.name === 'Strength' && !n.node.ascendancyName)!;
    const tattoo = {
      icon: 'Art/2DArt/SkillIcons/passives/FireResistNode.png',
      effect: 'Art/2DArt/UIImages/InGame/AncestralTrial/PassiveTreeTattoos/NgamahuTribePassiveBG.png'
    };
    fillScene(scene, empty(), () => true);
    const plain = spritesAt(scene, str.at);
    fillScene(scene, { ...empty(), tattoos: new Map([[str.id, tattoo]]) }, () => true);
    const tattooed = spritesAt(scene, str.at);
    expect(tattooed.filter((s) => s.startsWith('tree/assets/tattoo-active-effect'))).toHaveLength(1);
    expect(tattooed.length).toBe(plain.length + 1);
    expect(tattooed.filter((s) => !plain.includes(s))).toHaveLength(2);

    const mastery = scene.nodes.find((n) => n.node.isMastery && n.node.activeEffectImage)!;
    const runegraft = {
      icon: 'Art/2DArt/SkillIcons/passives/AtlasTrees/RuneGraftingIcon20.png',
      effect: 'Art/2DArt/UIImages/InGame/Village/RuneCraftingPassiveBG.png'
    };
    const alloc = { ...empty(), active: new Set([mastery.id]) };
    fillScene(scene, alloc, () => true);
    const own = spritesAt(scene, mastery.at);
    fillScene(scene, { ...alloc, tattoos: new Map([[mastery.id, runegraft]]) }, () => true);
    const grafted = spritesAt(scene, mastery.at);
    expect(grafted.length).toBe(own.length);
    expect(grafted.filter((s) => !own.includes(s))).toHaveLength(2);
  });

  // PassiveTreeView.lua:831-833
  it('draws every frame allocated while the heat map is on', () => {
    const scene = buildScene(tree!);
    const notable = scene.nodes.find((n) => n.node.isNotable && !n.node.ascendancyName)!;
    const mastery = scene.nodes.find((n) => n.node.isMastery && n.node.inactiveIcon)!;
    const frame = (state: Partial<ReturnType<typeof empty>> & { heatMap?: boolean }, at = notable.at) => {
      fillScene(scene, { ...empty(), ...state }, () => true);
      return spritesAt(scene, at);
    };
    const allocFrame = frame({ active: new Set([notable.id]) })[1];
    const [icon, unallocFrame] = frame({});
    expect(allocFrame, 'control: the frames differ').not.toBe(unallocFrame);
    expect(frame({ heatMap: true })).toEqual([icon, allocFrame]);
    expect(frame({ heatMap: true }, mastery.at)).toEqual(frame({}, mastery.at));
  });

  // PassiveTree.lua:446-492, PassiveTreeView.lua:840-852
  it("frames every node with PoB's overlay for its type and state", () => {
    const t = tree!;
    const scene = buildScene(t);
    const overlay: Record<string, Record<string, string>> = {
      Normal: { alloc: 'PSSkillFrameActive', path: 'PSSkillFrameHighlighted', unalloc: 'PSSkillFrame' },
      Notable: { alloc: 'NotableFrameAllocated', path: 'NotableFrameCanAllocate', unalloc: 'NotableFrameUnallocated' },
      Keystone: { alloc: 'KeystoneFrameAllocated', path: 'KeystoneFrameCanAllocate', unalloc: 'KeystoneFrameUnallocated' },
      Socket: { alloc: 'JewelFrameAllocated', path: 'JewelFrameCanAllocate', unalloc: 'JewelFrameUnallocated' }
    };
    for (const s of ['alloc', 'path', 'unalloc']) {
      const cap = s === 'alloc' ? 'Allocated' : s === 'path' ? 'CanAllocate' : 'Normal';
      overlay.Normal[s + 'Ascend'] = 'AscendancyFrameSmall' + cap;
      overlay.Notable[s + 'Ascend'] = 'AscendancyFrameLarge' + cap;
      overlay.Notable[s + 'Blighted'] = 'BlightedNotableFrame' + (s === 'unalloc' ? 'Unallocated' : cap);
      overlay.Keystone[s + 'Blighted'] = overlay.Keystone[s];
      overlay.Socket[s + 'Alt'] = 'JewelSocketAlt' + (s === 'alloc' ? 'Active' : s === 'path' ? 'CanAllocate' : 'Normal');
    }
    const framed = scene.nodes.filter(({ node }) => !node.isMastery && !node.isAscendancyStart && node.classStartIndex === undefined);
    const ids = new Set(framed.map((n) => n.id));
    const wrong: string[] = [];
    for (const s of ['alloc', 'path', 'unalloc']) {
      fillScene(scene, { ...empty(), active: s === 'alloc' ? ids : new Set(), hoverPath: s === 'path' ? ids : new Set() }, () => true);
      const at = new Map<string, { x: number; y: number; sprite: string }[]>();
      for (const i of sceneDrawList(scene, 1, undefined) as { k: string; src: string; s: number[]; q: number[] }[]) {
        if (i.k !== 'sprite') continue;
        const c = { x: i.q[0] + i.q[2] / 2, y: i.q[1] + i.q[5] / 2, sprite: `${i.src} ${i.s.join(',')}` };
        const key = `${Math.round(c.x)},${Math.round(c.y)}`;
        at.set(key, [...(at.get(key) ?? []), c]);
      }
      const drawnAt = (p: { x: number; y: number }) =>
        [-1, 0, 1].flatMap((dx) =>
          [-1, 0, 1].flatMap((dy) =>
            (at.get(`${Math.round(p.x) + dx},${Math.round(p.y) + dy}`) ?? [])
              .filter((c) => Math.abs(c.x - p.x) < 0.1 && Math.abs(c.y - p.y) < 0.1)
              .map((c) => c.sprite)
          )
        );
      for (const { id, node, at: p } of framed) {
        const type = node.isJewelSocket ? 'Socket' : node.isKeystone ? 'Keystone' : node.isNotable ? 'Notable' : 'Normal';
        const key =
          type === 'Socket' ? s + (node.expansionJewel ? 'Alt' : '') : s + (node.ascendancyName ? 'Ascend' : '') + (node.isBlighted ? 'Blighted' : '');
        const name = (node.name === 'Charm Socket' ? 'Azmeri' : '') + overlay[type][key];
        const sprite = (node.ascendancyName && st.bloodlineSprites[node.ascendancyName]?.[name]) || st.inverseSpritesOther[name];
        const c = sprite?.coords[name];
        const want = c && `${scene.sheets[scene.sheetOfUrl.get(sprite.filename)!]} ${[c.x, c.y, c.w, c.h].join(',')}`;
        if (!want || !drawnAt(p).includes(want)) wrong.push(`${s} ${type} ${id} ${node.name}: ${name}`);
      }
    }
    expect(framed.length).toBeGreaterThan(1000);
    expect(wrong).toEqual([]);
  });

  // PassiveTreeView.lua:663-699
  it('styles a connector by both trees', () => {
    const scene = buildScene(tree!);
    const i = scene.connections.findIndex((c) => !tree!.nodes[c.a].ascendancyName && !tree!.nodes[c.a].classStartIndex && !tree!.nodes[c.b].classStartIndex);
    const { a, b } = scene.connections[i];
    const style = (state: Partial<ReturnType<typeof empty>> & { granted?: Set<number>; compared?: Set<number> }) => {
      fillScene(scene, { ...empty(), ...state }, () => true);
      return scene.strokes[i * STROKE_FLOATS + 5];
    };
    const both = new Set([a, b]);
    expect(style({})).toBe(Style.Base);
    expect(style({ active: both })).toBe(Style.Active);
    expect(style({ granted: both })).toBe(Style.Base);
    expect(style({ active: both, compared: both })).toBe(Style.Active);
    expect(style({ compared: both })).toBe(Style.ActiveCompared);
    expect(style({ active: both, compared: new Set([a]) })).toBe(Style.ActiveOwn);
    expect(style({ compared: new Set() })).toBe(Style.Base);
    expect(STYLE_COLOURS[Style.ActiveCompared]).toBe('#00de00');
    expect(STYLE_COLOURS[Style.ActiveOwn]).toBe('#e90000');
    expect(STYLE_WIDTHS[Style.ActiveCompared]).toBe(STYLE_WIDTHS[Style.Active]);
  });

  // PassiveTreeView.lua:708-710, 972-976
  it('draws what the hovered node takes with it in red', () => {
    const scene = buildScene(tree!);
    const i = scene.connections.findIndex((c) => !tree!.nodes[c.a].ascendancyName && !tree!.nodes[c.a].classStartIndex && !tree!.nodes[c.b].classStartIndex);
    const { a, b } = scene.connections[i];
    const both = new Set([a, b]);
    const tints = (state: { hovered?: number; depends?: Set<number> }) => {
      fillScene(scene, { ...empty(), active: both, ...state }, () => true);
      const list = sceneDrawList(scene, 1, undefined) as { k: string; q?: number[]; t?: number[] }[];
      const at = (p: { x: number; y: number }) =>
        list
          .filter((s) => s.k === 'sprite' && Math.round(s.q![0] + s.q![2] / 2) === Math.round(p.x) && Math.round(s.q![1] + s.q![5] / 2) === Math.round(p.y))
          .map((s) => s.t);
      const pos = (id: number) => scene.nodes.find((n) => n.id === id)!.at;
      return { style: scene.strokes[i * STROKE_FLOATS + 5], a: at(pos(a)), b: at(pos(b)) };
    };
    const off = tints({ hovered: a });
    expect(off.style).toBe(Style.Active);
    expect(
      off.b.some((t) => t),
      'control: no tint without dependents'
    ).toBe(false);
    const on = tints({ hovered: a, depends: both });
    expect(on.style).toBe(Style.Depends);
    expect(STYLE_COLOURS[Style.Depends]).toBe('#e90000');
    expect(on.b.at(-1), "the dependent's frame").toEqual([1, 0, 0]);
    expect(on.a, "the hovered node's own sprites").toEqual(off.a);
  });

  // PassiveTreeView.lua:679-683
  it("draws a connector on Split Personality's path in its colour", () => {
    const scene = buildScene(tree!);
    const i = scene.connections.findIndex((c) => !tree!.nodes[c.a].ascendancyName && !tree!.nodes[c.a].classStartIndex && !tree!.nodes[c.b].classStartIndex);
    const { a, b } = scene.connections[i];
    const style = (state: Partial<ReturnType<typeof empty>> & { split?: Set<number>; compared?: Set<number> }) => {
      fillScene(scene, { ...empty(), ...state }, () => true);
      return scene.strokes[i * STROKE_FLOATS + 5];
    };
    const both = new Set([a, b]);
    expect(style({ split: both })).toBe(Style.SplitBase);
    expect(style({ split: both, active: both })).toBe(Style.SplitActive);
    expect(style({ split: both, hoverPath: both })).toBe(Style.SplitHover);
    expect(style({ split: new Set([a]), active: both })).toBe(Style.Active);
    expect(style({ split: both, compared: both })).toBe(Style.ActiveCompared);
    expect([Style.SplitBase, Style.SplitHover, Style.SplitActive].map((s) => STYLE_COLOURS[s])).toEqual(['#523a04', '#c88300', '#e9ba1e']);
    expect(STYLE_WIDTHS[Style.SplitActive]).toBe(STYLE_WIDTHS[Style.Active]);
  });
});
