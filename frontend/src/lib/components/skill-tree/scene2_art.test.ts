import { describe, expect, it } from 'vitest';
import { buildAtlas2, type Tree2ArtData } from '$lib/skill_tree/tree2_art';
import { COLOURS, SPRITE2_FLOATS, STROKE2_FLOATS, attachArt2, buildScene2, fillScene2, type Scene2, type Scene2State } from './scene2';
import { tinyTree } from './tree2_fixture';
import type { Tree2 } from './scene2';

const state = (s: Partial<Scene2State> = {}): Scene2State => ({
  alloc: new Set(),
  allocMode: new Map(),
  hovered: undefined,
  hoverPath: new Set(),
  hoverDep: new Set(),
  ascendClassBaseName: '',
  ...s
});

const at = (x: number, y: number, w: number, h: number) => ({ x, y, w, h });
// two sheets at zoom 0.5 (the sizes in the source art are twice the sheet's), an Inactive copy that must lose, and a smaller zoom
const art = (): Tree2ArtData => ({
  nodes: {
    '4': { icon: 'Icon4', activeEffectImage: 'Effect4' },
    '6': { activeEffectImage: 'GroupArt' },
    '8': {}
  },
  nodeOverlay: {
    Notable: { alloc: 'NotableAlloc', path: 'NotablePath', unalloc: 'NotableUnalloc' },
    Socket: { alloc: 'JewelAlloc', path: 'JewelPath', unalloc: 'JewelUnalloc' }
  },
  classes: [
    {
      name: 'Witch',
      background: { image: 'ClassesWitch', x: 0, y: 0, width: 1500, height: 1500, active: { width: 2000, height: 2000 }, bg: { width: 2000, height: 2000 } },
      ascendancies: [
        { id: 'Lich', name: 'Lich', background: { image: 'ClassesLich', x: 0, y: 600, width: 300, height: 300 } },
        { id: 'Blood Mage', name: 'Blood Mage', background: { image: 'ClassesBloodMage', x: 900, y: 600, width: 300, height: 300 } }
      ]
    }
  ],
  sprites: {
    normalInactive: { '0.5': { filename: 'https://x/a.png', w: 64, h: 64, coords: { Icon4: at(60, 0, 10, 10) } } },
    normalActive: {
      '0.25': { filename: 'https://x/small.png', w: 64, h: 64, coords: { Icon4: at(0, 0, 5, 5) } },
      '0.5': { filename: 'https://x/a.png', w: 64, h: 64, coords: { Icon4: at(0, 0, 10, 10), Effect4: at(10, 0, 20, 20), GroupArt: at(30, 0, 8, 8) } }
    },
    frame: {
      '0.5': {
        filename: 'https://x/b.png',
        w: 64,
        h: 64,
        coords: Object.fromEntries(
          [
            'NotableAlloc',
            'NotablePath',
            'NotableUnalloc',
            'JewelAlloc',
            'JewelPath',
            'JewelUnalloc',
            'ClassesWitch',
            'ClassesLich',
            'ClassesBloodMage',
            'BGTree',
            'BGTreeActive'
          ].map((n, i) => [n, at(i * 4, 0, 4, 4)])
        )
      }
    }
  }
});

const sprites = (scene: Scene2, layer: 'under' | 'lines' | 'over') =>
  Array.from({ length: scene.counts[layer] }, (_, i) => Array.from(scene[layer].subarray(i * SPRITE2_FLOATS, (i + 1) * SPRITE2_FLOATS)));
const atlas = buildAtlas2(art().sprites);
const named = (sx: number, sheet: number) => [...atlas.byName].find(([, s]) => s.sheet === sheet && s.x === sx)![0];
const describeSprite = (v: number[]) => ({
  name: named(v[6], v[10]),
  o: [v[0], v[1]],
  u: [v[2], v[3]],
  v: [v[4], v[5]],
  tint: v.slice(11, 15).map((t) => +t.toFixed(4))
});

describe('the PoE2 tree atlas (PassiveTree.lua:144-181, 812-819)', () => {
  it('names every sprite once, at the largest zoom, never from an Inactive sheet, sized in the source art', () => {
    expect(atlas.sheets).toEqual(['https://x/a.png', 'https://x/b.png']);
    expect(atlas.byName.get('Icon4')).toEqual({ sheet: 0, x: 0, y: 0, w: 10, h: 10, width: 20, height: 20 });
  });
});

describe('the PoE2 tree art (PassiveTreeView.lua:602-655, 930-1180)', () => {
  const scene = buildScene2(tinyTree());
  attachArt2(scene, art(), atlas, 'https://data/0.5');

  it('draws an unallocated notable as its effect at 15% under the connectors, its icon at half gray and its unallocated frame over them', () => {
    fillScene2(scene, state({ alloc: new Set([1]) }));
    const under = sprites(scene, 'under').map(describeSprite);
    const over = sprites(scene, 'over').map(describeSprite);
    expect(under).toContainEqual({ name: 'Effect4', o: [500 - 380, -380], u: [760, 0], v: [0, 760], tint: [1, 1, 1, 0.15] });
    expect(over.filter((s) => s.o[0] === 500 - 54 || s.o[0] === 500 - 80)).toEqual([
      { name: 'Icon4', o: [500 - 54, -54], u: [108, 0], v: [0, 108], tint: [0.502, 0.502, 0.502, 1] },
      { name: 'NotableUnalloc', o: [500 - 80, -80], u: [160, 0], v: [0, 160], tint: [1, 1, 1, 1] }
    ]);
    expect(scene.counts.discs).toBe(scene.nodes.length - 3);
  });

  it('draws an allocated weapon-set notable full bright, with its allocated frame in the weapon set colour', () => {
    fillScene2(scene, state({ alloc: new Set([1, 4]), allocMode: new Map([[4, 1]]) }));
    const all = [...sprites(scene, 'under'), ...sprites(scene, 'over')].map(describeSprite);
    expect(all.find((s) => s.name === 'Effect4')!.tint).toEqual([1, 1, 1, 1]);
    expect(all.find((s) => s.name === 'Icon4')!.tint).toEqual([1, 1, 1, 1]);
    expect(all.find((s) => s.name === 'NotableAlloc')!.tint).toEqual([...COLOURS.weaponSet1.map((c) => +c.toFixed(4)), 1]);
  });

  it('draws the heat map: an unallocated node in its power colour, frame included, in the allocated state; no effect art, no half gray (PassiveTreeView.lua:948, 1003-1030, 1074, 1108-1115)', () => {
    fillScene2(scene, state({ alloc: new Set([1, 5]), heat: new Map([[4, [0.5, 0.25, 0]]]) }));
    const all = [...sprites(scene, 'under'), ...sprites(scene, 'over')].map(describeSprite);
    expect(all.find((s) => s.name === 'Effect4')).toBeUndefined();
    expect(all.find((s) => s.name === 'Icon4')!.tint).toEqual([0.5, 0.25, 0, 1]);
    expect(all.find((s) => s.name === 'NotableAlloc')!.tint).toEqual([0.5, 0.25, 0, 1]);
    expect(all.find((s) => s.name === 'NotableUnalloc')).toBeUndefined();
    // a node the power run has not reached is black, an allocated one white
    expect(all.find((s) => s.name === 'JewelAlloc')!.tint).toEqual([0, 0, 0, 1]);
  });

  it("draws a jewel socket's frame as its base, red when it depends on the hovered node", () => {
    fillScene2(scene, state({ alloc: new Set([1]), hovered: 2, hoverDep: new Set([8]) }));
    expect(
      sprites(scene, 'over')
        .map(describeSprite)
        .find((s) => s.name === 'JewelUnalloc')!.tint
    ).toEqual([1, 0, 0, 1]);
  });

  it("draws the class's background, the start's rotated active art and the tree centre; the other ascendancies at half", () => {
    fillScene2(scene, state({ alloc: new Set([1]), ascendClassBaseName: 'Lich' }));
    const under = sprites(scene, 'under').map(describeSprite);
    expect(under.slice(0, 3).map((s) => s.name)).toEqual(['ClassesLich', 'BGTreeActive', 'BGTree']);
    expect(under.find((s) => s.name === 'ClassesBloodMage')!.tint).toEqual([0.5, 0.5, 0.5, 1]);
    expect(under.find((s) => s.name === 'ClassesLich' && s.o[1] === 300)!.tint).toEqual([1, 1, 1, 1]);
    // the class start draws nothing of its own once the centre art is there
    expect(scene.counts.discs).toBe(scene.nodes.length - 3);
  });

  it('keeps the vector shape of a node whose sheet has not loaded', () => {
    fillScene2(scene, state(), (sheet) => sheet !== 0);
    expect(
      sprites(scene, 'over')
        .map(describeSprite)
        .some((s) => s.name === 'Icon4')
    ).toBe(false);
    expect(
      sprites(scene, 'over')
        .map(describeSprite)
        .some((s) => s.name === 'NotableUnalloc')
    ).toBe(true);
    fillScene2(scene, state(), () => false);
    expect([scene.counts.under, scene.counts.over, scene.counts.discs]).toEqual([0, 0, scene.nodes.length]);
  });
});

/* PassiveTree.lua:630-786 run under LuaJIT on this file's tree (node 9 added at orbit 1, angle pi) with these line
   sizes: each connector's quad corners in tree space, by connector, quad and line state; uv (1,1), (0,p), (0,0), (p,0)
   for an arc, (0,1), (0,0), (endS,0), (endS,1) for a line */
const POB_QUADS: Record<string, number[]> = {
  '2-3/1/Normal': [0.0, 0.0, 0.0, -90.0, 90.0, -90.0, 90.0, 0.0],
  '2-3/1/Intermediate': [0.0, 0.0, 0.0, -92.0, 92.0, -92.0, 92.0, 0.0],
  '2-3/1/Active': [0.0, 0.0, 0.0, -96.0, 96.0, -96.0, 96.0, 0.0],
  '3-4/1/Normal': [291.0, -261.809090751, 504.449607742, 5.573912714, 291.0, 219.023520456, 77.550392258, 5.573912714],
  '3-4/1/Intermediate': [291.0, -261.809090751, 506.960779598, 8.719595107, 291.0, 224.680374705, 75.039220402, 8.719595107],
  '3-4/1/Active': [291.0, -261.809090751, 510.727537381, 13.438118698, 291.0, 233.165656079, 71.272462619, 13.438118698],
  '4-5/1/Normal': [500.0, 7.0, 500.0, -7.0, 1000.0, -7.0, 1000.0, 7.0],
  '4-5/1/Intermediate': [500.0, 7.0, 500.0, -7.0, 1000.0, -7.0, 1000.0, 7.0],
  '4-5/1/Active': [500.0, 7.0, 500.0, -7.0, 1000.0, -7.0, 1000.0, 7.0],
  '2-9/1/Normal': [0.0, 0.0, 0.0, 90.0, -90.0, 90.0, -90.0, 0.0],
  '2-9/1/Intermediate': [0.0, 0.0, 0.0, 92.0, -92.0, 92.0, -92.0, 0.0],
  '2-9/1/Active': [0.0, 0.0, 0.0, 96.0, -96.0, 96.0, -96.0, 0.0],
  '2-9/2/Normal': [0.0, 0.0, -0.0, -90.0, -90.0, -90.0, -90.0, 0.0],
  '2-9/2/Intermediate': [0.0, 0.0, -0.0, -92.0, -92.0, -92.0, -92.0, 0.0],
  '2-9/2/Active': [0.0, 0.0, -0.0, -96.0, -96.0, -96.0, -96.0, 0.0]
};

describe("the PoE2 tree's connector art (PassiveTree.lua:630-786, PassiveTreeView.lua:781-822)", () => {
  const lineArt = (): Tree2ArtData => {
    const coords: Record<string, ReturnType<typeof at>> = {};
    const add = (name: string, w: number, h: number) => (coords[name] = at(Object.keys(coords).length * 200, 0, w, h));
    for (const [orbit, sizes] of [
      [1, [45, 46, 48]],
      [3, [170, 172, 175]]
    ] as const)
      ['Normal', 'Intermediate', 'Active'].forEach((st, i) => add(`CharacterOrbit${orbit}${st}`, sizes[i], sizes[i]));
    add('CharacterLineConnectorNormal', 700, 7);
    add('CharacterLineConnectorIntermediate', 700, 12);
    add('CharacterLineConnectorActive', 700, 18);
    return {
      nodes: {},
      connectionArt: { default: 'Character', ascendancy: 'CharacterAscendancy' },
      sprites: { line: { '0.5': { filename: 'https://x/line.png', w: 4096, h: 512, coords } } }
    };
  };
  const tree = (): Tree2 => {
    const t = tinyTree();
    t.nodes.find((n) => n.id === 2)!.connections!.push({ id: 9, orbit: 0 });
    t.nodes.push({ id: 9, x: 0, y: 82, angle: Math.PI, group: 1, orbit: 1, type: 'Normal', name: 'n9', stats: [], connections: [] });
    return t;
  };
  const scene = buildScene2(tree());
  const lineAtlas = buildAtlas2(lineArt().sprites);
  attachArt2(scene, lineArt(), lineAtlas, 'https://data/0.5');
  const corner = (q: { o: number[]; u: number[]; v: number[] }, u: number, v: number) => [q.o[0] + u * q.u[0] + v * q.v[0], q.o[1] + u * q.u[1] + v * q.v[1]];

  it("places every arc's quad where PoB does, in each line state, a wide arc as two", () => {
    let checked = 0;
    scene.connectors.forEach((c, i) => {
      for (const st of ['Normal', 'Intermediate', 'Active'] as const) {
        const quads = scene.art!.connectors[i][st]!;
        quads.forEach((q, k) => {
          const pob = POB_QUADS[`${c.a}-${c.b}/${k + 1}/${st}`];
          expect(pob, `${c.a}-${c.b}/${k + 1}/${st}`).toBeDefined();
          const uv =
            q.clip < 1 || c.kind === 1
              ? [
                  [1, 1],
                  [0, q.clip],
                  [0, 0],
                  [q.clip, 0]
                ]
              : [
                  [0, 1],
                  [0, 0],
                  [1, 0],
                  [1, 1]
                ];
          if (q.repeat !== 1 && st !== 'Normal') return;
          uv.forEach(([u, v], j) => {
            const [x, y] = corner(q, u, v);
            expect(x).toBeCloseTo(pob[j * 2], 6);
            expect(y).toBeCloseTo(pob[j * 2 + 1], 6);
          });
          checked++;
        });
      }
    });
    // 2-3, 3-4, 2-9 (two quads) in three states, 4-5's Normal line
    expect(checked).toBe(3 * 4 + 1);
    // PassiveTree.lua:733: the line repeats its image along its length
    const line = scene.art!.connectors[scene.connectors.findIndex((c) => c.a === 4)].Normal![0];
    expect(line.repeat).toBeCloseTo(500 / 1400, 9);
  });

  it('draws the connectors as their art in the state the spec gives them, and no vector stroke for them', () => {
    fillScene2(scene, state({ alloc: new Set([1, 2, 3]) }));
    expect(scene.counts.connectorStrokes).toBe(0);
    // 2-9 is two quads
    expect(scene.counts.lines).toBe(scene.connectors.length + 1);
    const lines = sprites(scene, 'lines');
    const name = (v: number[]) => [...lineAtlas.byName].find(([, s]) => s.x === v[6])![0];
    expect(lines.map(name)).toContain('CharacterOrbit1Active');
    expect(lines.map(name)).toContain('CharacterOrbit3Normal');
  });

  it('keeps a vector stroke for a connector whose line sheet has not loaded', () => {
    fillScene2(scene, state(), () => false);
    expect([scene.counts.lines, scene.counts.connectorStrokes]).toEqual([0, scene.connectors.length]);
  });
});

describe('the PoE2 tree art by spec state (PassiveTreeView.lua:793-797, 930-1180, 2179-2191)', () => {
  const tree = (): Tree2 => {
    const t = tinyTree();
    // node 5 shows once node 4 is allocated
    t.nodes.find((n) => n.id === 5)!.unlockConstraint = [4];
    return t;
  };
  const data = (): Tree2ArtData => {
    const a = art();
    a.nodes['5'] = { icon: 'IconAttr', options: [{ name: 'Strength', icon: 'IconStr' }] };
    a.nodeOverlay!.Normal = { alloc: 'NotableAlloc', path: 'NotablePath', unalloc: 'NotableUnalloc' };
    const frame = a.sprites!.frame['0.5'].coords;
    Object.assign(frame, { IconAttr: at(60, 8, 4, 4), IconStr: at(56, 8, 4, 4), Ruby: at(52, 8, 4, 4), 'Sol Spark': at(48, 8, 4, 4) });
    return a;
  };
  const scene = buildScene2(tree());
  const a = buildAtlas2(data().sprites);
  attachArt2(scene, data(), a, 'https://data/0.5');
  const names = (layer: 'under' | 'over') =>
    sprites(scene, layer).map((v) => [...a.byName].find(([, sp]) => sp.sheet === v[10] && sp.x === v[6] && sp.y === v[7])?.[0]);

  it('hides a node and its connectors until the nodes its unlock constraint names are allocated', () => {
    fillScene2(scene, state());
    expect(names('over')).not.toContain('IconAttr');
    const lineTo5 = (st: Scene2State) => {
      fillScene2(scene, st);
      return Array.from({ length: scene.counts.connectorStrokes }, (_, k) => scene.strokes[k * STROKE2_FLOATS + 2]).includes(1000);
    };
    expect(lineTo5(state())).toBe(false);
    expect(lineTo5(state({ alloc: new Set([4]) }))).toBe(true);
    expect(names('over')).toContain('IconAttr');
  });

  it("draws an attribute-chosen node's option icon", () => {
    fillScene2(scene, state({ alloc: new Set([4, 5]), icons: new Map([[5, 'IconStr']]) }));
    expect(names('over')).toContain('IconStr');
    expect(names('over')).not.toContain('IconAttr');
  });

  it("draws a socketed jewel's art over an allocated socket: a unique's own art where the sheet has it, else its base", () => {
    fillScene2(scene, state({ alloc: new Set([8]), jewels: new Map([[8, { title: 'Sol Spark', baseName: 'Ruby', unique: false }]]) }));
    expect(names('over')).toContain('Ruby');
    fillScene2(scene, state({ alloc: new Set([8]), jewels: new Map([[8, { title: 'Sol Spark', baseName: 'Ruby', unique: true }]]) }));
    expect(names('over')).toContain('Sol Spark');
    fillScene2(scene, state({ jewels: new Map([[8, { title: 'Sol Spark', baseName: 'Ruby', unique: true }]]) }));
    expect(names('over')).not.toContain('Sol Spark');
  });

  it('tints a node the compared spec has and this one lacks green, and the reverse red (PassiveTreeView.lua:1057-1067)', () => {
    fillScene2(scene, state({ alloc: new Set([1]), compared: new Set([1, 8]) }));
    const over = sprites(scene, 'over');
    const jewelBase = over.find((v) => v[6] === a.byName.get('JewelAlloc')!.x)!;
    expect(jewelBase.slice(11, 15)).toEqual([0, 1, 0, 1]);
    fillScene2(scene, state({ alloc: new Set([1, 4]), compared: new Set([1]) }));
    const icon = sprites(scene, 'over').find((v) => v[6] === a.byName.get('Icon4')!.x)!;
    expect(icon.slice(11, 15)).toEqual([1, 0, 0, 1]);
  });
});

describe("the PoE2 tree's orbit images on a shared quarter", () => {
  it('cut each arc at the next smaller image sharing its centre corner, so it draws only its own arc', () => {
    const data: Tree2ArtData = {
      nodes: {},
      connectionArt: { default: 'Character', ascendancy: 'CharacterAscendancy' },
      sprites: {
        line: {
          '0.5': {
            filename: 'https://x/line.png',
            w: 512,
            h: 512,
            coords: {
              CharacterOrbit1Normal: at(170 - 45, 170 - 45, 45, 45),
              CharacterOrbit3Normal: at(0, 0, 170, 170),
              CharacterLineConnectorNormal: at(0, 200, 400, 7)
            }
          }
        }
      }
    };
    const scene = buildScene2(tinyTree());
    attachArt2(scene, data, buildAtlas2(data.sprites), 'https://data/0.5');
    const arc3 = scene.art!.connectors[scene.connectors.findIndex((c) => c.a === 3)].Normal![0];
    const arc1 = scene.art!.connectors[scene.connectors.findIndex((c) => c.a === 2)].Normal![0];
    expect([arc3.inner, arc1.inner]).toEqual([45 / 170, 0]);
  });
});
