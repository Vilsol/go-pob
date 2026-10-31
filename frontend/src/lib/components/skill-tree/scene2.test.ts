import { describe, expect, it } from 'vitest';
import { COLOURS, DISC_FLOATS, STROKE2_FLOATS, buildScene2, fillScene2, nodeAt2, screenToTree, treeToScreen, zoomAbout, type Scene2State } from './scene2';
import { tinyTree } from './tree2_fixture';

const state = (s: Partial<Scene2State> = {}): Scene2State => ({
  alloc: new Set(),
  allocMode: new Map(),
  hovered: undefined,
  hoverPath: new Set(),
  hoverDep: new Set(),
  ascendClassBaseName: '',
  ...s
});

describe('the PoE2 scene (PassiveTree.lua:305-366, 630-744)', () => {
  const scene = buildScene2(tinyTree());
  const conn = (a: number, b: number) => scene.connectors.find((c) => c.a === a && c.b === b);

  it('skips class-start and OnlyImage nodes as connector ends, and OnlyImage as a node', () => {
    expect(conn(1, 2)).toBeUndefined();
    expect(scene.connectors.map((c) => `${c.a}-${c.b}`)).toEqual(['2-3', '3-4', '4-5']);
    expect(scene.nodes.map((p) => p.node.id)).toEqual([1, 2, 3, 4, 5, 7, 8]);
  });

  it('draws two nodes on one orbit of one group as an arc about the group', () => {
    const c = conn(2, 3)!;
    expect(c.kind).toBe(1);
    const [cx, cy, r, a0, sweep] = c.geom;
    expect([cx, cy, r]).toEqual([0, 0, 82]);
    expect(a0).toBeCloseTo(-Math.PI / 2);
    expect(sweep).toBeCloseTo(Math.PI / 2);
  });

  it('draws a connection with an orbit as an arc of that orbit radius through both nodes', () => {
    const c = conn(3, 4)!;
    expect(c.kind).toBe(1);
    const [cx, cy, r, a0, sweep] = c.geom;
    expect(r).toBe(335);
    const ends = [a0, a0 + sweep].map((a) => [cx + r * Math.cos(a), cy + r * Math.sin(a)]);
    const near = (p: number[], x: number, y: number) => Math.hypot(p[0] - x, p[1] - y) < 1e-6;
    expect(ends.some((p) => near(p, 82, 0)) && ends.some((p) => near(p, 500, 0))).toBe(true);
    expect(sweep).toBeLessThan(Math.PI);
  });

  it('draws anything else as a line', () => {
    expect(conn(4, 5)).toMatchObject({ kind: 0, geom: [500, 0, 1000, 0, 0] });
  });
});

describe('hit testing (PassiveTreeView.lua:291-299)', () => {
  const scene = buildScene2(tinyTree());
  it('finds a node within its overlay size, never an ascendancy start', () => {
    expect(nodeAt2(scene, 540, 30)?.id).toBe(4);
    expect(nodeAt2(scene, 590, 0)).toBeUndefined();
    expect(nodeAt2(scene, 0, 600)).toBeUndefined();
  });
});

describe('the scene state (PassiveTreeView.lua:675-769)', () => {
  const ring = (scene: ReturnType<typeof buildScene2>, id: number) => {
    const i = scene.nodes.findIndex((p) => p.node.id === id);
    return Array.from(scene.discs.subarray(i * DISC_FLOATS + 7, i * DISC_FLOATS + 10));
  };
  const stroke = (scene: ReturnType<typeof buildScene2>, a: number) => {
    const i = scene.connectors.findIndex((c) => c.a === a);
    return { width: scene.strokes[i * STROKE2_FLOATS + 6], colour: Array.from(scene.strokes.subarray(i * STROKE2_FLOATS + 7, i * STROKE2_FLOATS + 10)) };
  };
  const close = (a: number[], b: number[]) => a.every((v, i) => Math.abs(v - b[i]) < 1e-6);

  it('colours allocated nodes and the connector between two of them active', () => {
    const scene = buildScene2(tinyTree());
    fillScene2(scene, state({ alloc: new Set([2, 3]) }));
    expect(close(ring(scene, 2), COLOURS.ringAlloc)).toBe(true);
    expect(close(ring(scene, 4), COLOURS.ringUnalloc)).toBe(true);
    expect(close(stroke(scene, 2).colour, COLOURS.active)).toBe(true);
    expect(close(stroke(scene, 3).colour, COLOURS.connector)).toBe(true);
  });

  it('marks the hover path intermediate and what depends on the hovered node red', () => {
    const scene = buildScene2(tinyTree());
    fillScene2(scene, state({ alloc: new Set([2, 3]), hovered: 5, hoverPath: new Set([4, 5]) }));
    expect(close(ring(scene, 4), COLOURS.ringPath)).toBe(true);
    expect(close(stroke(scene, 3).colour, COLOURS.intermediate)).toBe(true);
    fillScene2(scene, state({ alloc: new Set([2, 3]), hovered: 2, hoverDep: new Set([2, 3]) }));
    expect(close(ring(scene, 3), COLOURS.depends)).toBe(true);
    expect(close(stroke(scene, 2).colour, COLOURS.depends)).toBe(true);
  });

  // the states and tints are PoB-PoE2's getState and allocModeColor (PassiveTreeView.lua:675-706, 936-937) run under LuaJIT
  it('previews a weapon-set allocation in that set colour', () => {
    const scene = buildScene2(tinyTree());
    fillScene2(scene, state({ alloc: new Set([2, 3]), hovered: 5, hoverPath: new Set([4, 5]), specAllocMode: 1 }));
    expect(close(stroke(scene, 2).colour, COLOURS.active)).toBe(true);
    expect(close(stroke(scene, 3).colour, COLOURS.weaponSet1)).toBe(true);
    expect(close(stroke(scene, 4).colour, COLOURS.weaponSet1)).toBe(true);
    expect(close(ring(scene, 3), COLOURS.ringAlloc)).toBe(true);
    expect(close(ring(scene, 4), COLOURS.weaponSet1)).toBe(true);
    expect(close(ring(scene, 5), COLOURS.weaponSet1)).toBe(true);
  });

  it('previews weapon-set nodes on the path of a normal allocation as promoted back', () => {
    const scene = buildScene2(tinyTree());
    const allocMode = new Map([
      [3, 1],
      [4, 1]
    ]);
    fillScene2(scene, state({ alloc: new Set([2, 3, 4]), allocMode, hovered: 5, hoverPath: new Set([3, 4, 5]) }));
    expect(close(stroke(scene, 3).colour, COLOURS.intermediate)).toBe(true);
    expect(close(stroke(scene, 4).colour, COLOURS.intermediate)).toBe(true);
    expect(close(ring(scene, 3), COLOURS.ringAlloc)).toBe(true);
    expect(close(ring(scene, 4), COLOURS.ringAlloc)).toBe(true);
    fillScene2(scene, state({ alloc: new Set([2, 3, 4]), allocMode }));
    expect(close(stroke(scene, 3).colour, COLOURS.weaponSet1)).toBe(true);
    expect(close(ring(scene, 3), COLOURS.weaponSet1)).toBe(true);
  });
});

describe('the view (PassiveTreeView.lua:257-274, 1500-1514)', () => {
  it('keeps the point under the cursor in place while zooming', () => {
    const scene = buildScene2(tinyTree());
    const v = { zoomLevel: 3, zoomX: 0, zoomY: 0, width: 800, height: 600, dpr: 1 };
    const before = screenToTree(scene, v, 650, 250);
    const z = zoomAbout(v, 4, 650, 250);
    expect(z.zoomLevel).toBe(7);
    const after = treeToScreen(scene, z, before.x, before.y);
    expect(after.x).toBeCloseTo(650);
    expect(after.y).toBeCloseTo(250);
    expect(zoomAbout(v, 30, 0, 0).zoomLevel).toBe(20);
  });
});

describe('the search highlight (PassiveTreeView.lua:1181-1206)', () => {
  it('rings each matched node after the connectors', () => {
    const scene = buildScene2(tinyTree());
    fillScene2(scene, state({ search: new Set([4, 5]), searchR: 100 }));
    expect(scene.counts.strokes).toBe(scene.connectors.length + 2);
    const at = (i: number) => Array.from(scene.strokes.subarray(i * STROKE2_FLOATS, (i + 1) * STROKE2_FLOATS));
    const ring = at(scene.connectors.length);
    expect(ring.slice(0, 3)).toEqual([500, 0, 100]);
    expect(ring[4]).toBeCloseTo(Math.PI * 2);
    expect(ring[5]).toBe(1);
    expect(ring.slice(7)).toEqual([...COLOURS.search, 1]);
    expect(at(scene.connectors.length + 1).slice(0, 2)).toEqual([1000, 0]);
    fillScene2(scene, state());
    expect(scene.counts.strokes).toBe(scene.connectors.length);
  });
});

describe('the jewel radius rings (PassiveTreeView.lua:1132-1158, 1271-1353)', () => {
  const ringOf = (scene: ReturnType<typeof buildScene2>, id: number) => {
    const i = scene.nodes.findIndex((p) => p.node.id === id);
    return Array.from(scene.discs.subarray(i * DISC_FLOATS + 7, i * DISC_FLOATS + 10));
  };
  const rgb = (h: string) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16) / 255);
  const close = (a: number[], b: number[]) => a.every((v, i) => Math.abs(v - b[i]) < 1e-6);
  const extra = (scene: ReturnType<typeof buildScene2>) =>
    Array.from({ length: scene.counts.strokes - scene.connectors.length }, (_, k) =>
      Array.from(scene.strokes.subarray((scene.connectors.length + k) * STROKE2_FLOATS, (scene.connectors.length + k + 1) * STROKE2_FLOATS))
    );

  it('rings a hovered socket with each full-disc radius and colours the nodes inside the smallest', () => {
    const scene = buildScene2(tinyTree());
    fillScene2(scene, state({ hovered: 8 }));
    const rings = extra(scene);
    expect(rings.map((r) => r.slice(0, 3))).toEqual([
      [500, -400, 1200],
      [500, -400, 1380],
      [500, -400, 1560],
      [500, -400, 1800]
    ]);
    expect(close(rings[0].slice(7, 10), rgb('#bb6600'))).toBe(true);
    expect(close(ringOf(scene, 4), rgb('#bb6600'))).toBe(true);
    expect(close(ringOf(scene, 3), rgb('#66ffcc'))).toBe(true);
    expect(close(ringOf(scene, 2), rgb('#2222cc'))).toBe(true);
    expect(close(ringOf(scene, 5), rgb('#c100ff'))).toBe(true);
    fillScene2(scene, state({ hovered: 8, hoverDep: new Set([4]) }));
    expect(close(ringOf(scene, 4), COLOURS.depends)).toBe(true);
    fillScene2(scene, state({ hovered: 4 }));
    expect(extra(scene)).toEqual([]);
    expect(close(ringOf(scene, 3), COLOURS.ringUnalloc)).toBe(true);
  });

  it('rings an allocated socket by its jewel radius, the inner ring 1.06 times out', () => {
    const scene = buildScene2(tinyTree());
    fillScene2(scene, state({ alloc: new Set([8]), jewelRadiusIndex: new Map([[8, 5]]) }));
    const rings = extra(scene);
    expect(rings.map((r) => r[2])).toEqual([1140, Math.fround(780 * 1.06)]);
    expect(rings[0][10]).toBeCloseTo(0.7);
    fillScene2(scene, state({ jewelRadiusIndex: new Map([[8, 5]]) }));
    expect(extra(scene)).toEqual([]);
  });
});
