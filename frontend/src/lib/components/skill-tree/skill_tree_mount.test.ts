// @vitest-environment jsdom
// PassiveTreeView.lua:221-257

import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render } from '@testing-library/svelte';
import { brotliDecompressSync } from 'node:zlib';

import { cachedAsset } from '../../../../e2e/support/data_cache';
import type { Node, Tree } from '$lib/skill_tree/types';
import type { SceneState } from './scene';

afterEach(cleanup);

const rawJson = (v: string) => {
  const a = cachedAsset(`https://data.pob.gg/data/${v.replace('_', '.')}/tree/data.json.br`);
  if (a.status !== 200) throw new Error(`REFUSING: no ${v} tree in the bundle cache`);
  return brotliDecompressSync(a.body).toString('utf8');
};

const worker = vi.hoisted(() => ({
  granted: [] as number[],
  paths: {} as Record<number, number>,
  active: [] as number[],
  depends: [] as number[],
  calls: [] as string[]
}));

vi.mock('$lib/go/worker', () => {
  const known: Record<string, (...a: unknown[]) => unknown> = {
    GetTree: (v) => rawJson(v as string),
    GetBuildInfo: () => ({ className: 'Witch', ascendClassName: 'None', secondaryAscendClassId: 0 }),
    GetPassiveNodes: () => worker.active,
    CalculateDependentNodesWith: () => worker.depends,
    GetLeapHeldNodes: () => [],
    CalculateAllocationPaths: () => worker.paths,
    GetGrantedPassives: () => worker.granted
  };
  const syncWrap = new Proxy(
    {},
    {
      get:
        (_, name: string) =>
        (...args: unknown[]) => {
          worker.calls.push(name);
          return Promise.resolve(known[name]?.(...args));
        }
    }
  );
  return { syncWrap, syncWorker: undefined };
});

const states: SceneState[] = [];
vi.mock('./renderer', () => ({
  createRenderer: () => ({
    kind: 'canvas',
    setView: () => {},
    setState: (s: SceneState) => states.push(s),
    complete: () => true,
    drawListArt: () => undefined,
    destroy: () => {}
  })
}));

const st = await import('$lib/skill_tree');
const { currentBuild } = await import('$lib/global');
const SkillTree = (await import('./SkillTree.svelte')).default;

globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
};
globalThis.PointerEvent ??= class extends MouseEvent {} as typeof PointerEvent;

const W = 800;
const H = 600;

let tree: Tree;
beforeAll(async () => {
  await st.loadSkillTree('3_29');
  st.skillTree.subscribe((t) => (tree = t!))();
  Object.defineProperty(HTMLElement.prototype, 'offsetWidth', { configurable: true, get: () => W });
  Object.defineProperty(HTMLElement.prototype, 'offsetHeight', { configurable: true, get: () => H });
  currentBuild.set({} as never);
}, 60_000);

const mount = async (props: Record<string, unknown> = {}) => {
  const view = render(SkillTree, { skillTree: tree, skillTreeVersion: '3_29', chrome: false, ...props });
  const canvas = await vi.waitFor(() => {
    const c = view.container.querySelector('canvas');
    if (!c) throw new Error('no canvas yet');
    return c;
  });
  return { view, canvas, component: view.component as unknown as { focus: (id: number) => void } };
};

const someNotable = () => Object.values(tree.nodes).find((n) => n.isNotable && !n.ascendancyName && n.group !== undefined)!;

describe('a press on a node (PassiveTreeView.lua:221-257)', () => {
  const pressAt = async (canvas: HTMLCanvasElement, path: [number, number][]) => {
    const [x0, y0] = path[0];
    await fireEvent.pointerMove(canvas, { clientX: x0, clientY: y0 });
    await fireEvent.pointerDown(canvas, { clientX: x0, clientY: y0, button: 0 });
    for (const [x, y] of path.slice(1)) await fireEvent.pointerMove(canvas, { clientX: x, clientY: y });
    const [x1, y1] = path[path.length - 1];
    await fireEvent.pointerUp(canvas, { clientX: x1, clientY: y1, button: 0 });
  };

  it('picks the node on release when the pointer stayed within 5 px', async () => {
    const onpick = vi.fn<(n: Node) => void>();
    const onhover = vi.fn<(n: Node | undefined) => void>();
    const { canvas, component } = await mount({ onpick, onhover });
    const node = someNotable();
    component.focus(node.skill!);
    await pressAt(canvas, [
      [W / 2, H / 2],
      [W / 2 + 5, H / 2 - 5]
    ]);
    expect(onhover.mock.calls.at(-1)?.[0]?.skill, 'the pointer is over the node').toBe(node.skill);
    expect(onpick.mock.calls.map((c) => c[0].skill)).toEqual([node.skill]);
  });

  it('picks nothing while pressed, nor after a drag of more than 5 px', async () => {
    const onpick = vi.fn<(n: Node) => void>();
    const onhover = vi.fn<(n: Node | undefined) => void>();
    const { canvas, component } = await mount({ onpick, onhover });
    component.focus(someNotable().skill!);
    await fireEvent.pointerMove(canvas, { clientX: W / 2, clientY: H / 2 });
    expect(onhover.mock.calls.at(-1)?.[0]?.skill, 'the pointer is over the node').toBe(someNotable().skill);
    await fireEvent.pointerDown(canvas, { clientX: W / 2, clientY: H / 2, button: 0 });
    expect(onpick).not.toHaveBeenCalled();
    await fireEvent.pointerMove(canvas, { clientX: W / 2 + 6, clientY: H / 2 });
    await fireEvent.pointerMove(canvas, { clientX: W / 2, clientY: H / 2 });
    await fireEvent.pointerUp(canvas, { clientX: W / 2, clientY: H / 2, button: 0 });
    expect(onpick).not.toHaveBeenCalled();
  });
});

describe('the state handed to the renderer', () => {
  // PassiveTreeView.lua:811
  it("carries Go's granted passives and the compared tree's allocation", async () => {
    const node = someNotable();
    worker.granted = [node.skill!];
    states.length = 0;
    await mount({ compared: [1, 2], split: [3] });
    await vi.waitFor(() => expect([...(states.at(-1)?.granted ?? [])]).toEqual([node.skill]));
    expect([...(states.at(-1)?.compared ?? [])]).toEqual([1, 2]);
    expect([...(states.at(-1)?.split ?? [])]).toEqual([3]);
    worker.granted = [];
  });
});

// PassiveTreeView.lua:372-375
describe('hovering an allocated node', () => {
  it('hands the renderer the node and its dependents, and nothing once the pointer leaves', async () => {
    const node = someNotable();
    worker.active = [node.skill!, 11, 12];
    worker.depends = [11, 12];
    states.length = 0;
    const { canvas, component } = await mount();
    await vi.waitFor(() => expect([...(states.at(-1)?.active ?? [])]).toContain(node.skill));
    component.focus(node.skill!);
    await fireEvent.pointerMove(canvas, { clientX: W / 2, clientY: H / 2 });
    await vi.waitFor(() => expect([...(states.at(-1)?.depends ?? [])].sort()).toEqual([11, 12, node.skill!].sort()));
    expect(states.at(-1)?.hovered).toBe(node.skill);
    await fireEvent.pointerMove(canvas, { clientX: 1, clientY: 1 });
    await vi.waitFor(() => expect(states.at(-1)?.depends?.size).toBe(0));
    worker.active = [];
    worker.depends = [];
  });
});

// PassiveTreeView.lua:1715-1716
describe('the Shift trace', () => {
  it('hands the traced path out, and drops it when Shift is released', async () => {
    const node = someNotable();
    worker.paths = { [node.skill!]: -1 };
    const ontrace = vi.fn<(t: number[]) => void>();
    const { canvas, component } = await mount({ ontrace });
    component.focus(node.skill!);
    await fireEvent.keyDown(window, { key: 'Shift' });
    await fireEvent.pointerMove(canvas, { clientX: W / 2, clientY: H / 2, shiftKey: true });
    await vi.waitFor(() => expect(ontrace.mock.calls.at(-1)?.[0]).toEqual([node.skill]));
    await fireEvent.keyUp(window, { key: 'Shift' });
    expect(ontrace.mock.calls.at(-1)?.[0]).toEqual([]);
    worker.paths = {};
  });
});

// PassiveTreeView.lua:1102-1156
describe('the compared tree’s own subgraph nodes', () => {
  it('are drawn and hovered, never weighed', async () => {
    const id = 70000;
    const lone = { id, type: 'Notable', name: 'Lone', icon: '', stats: ['+1 to Strength'], group: 0, orbit: 0, orbitIndex: 0, x: 40000, y: 40000, linked: [] };
    worker.calls.length = 0;
    states.length = 0;
    const onhover = vi.fn<(n: Node | undefined) => void>();
    const { canvas, component } = await mount({ onhover, compareCluster: [lone] });
    await vi.waitFor(() => expect(states.at(-1)?.cluster?.nodes.map((n) => n.id)).toContain(id));
    component.focus(id);
    await fireEvent.pointerMove(canvas, { clientX: W / 2, clientY: H / 2 });
    expect(onhover.mock.calls.at(-1)?.[0]?.skill, 'the pointer is over the node').toBe(id);
    await new Promise((r) => setTimeout(r, 50));
    expect(worker.calls).not.toContain('StatCompareNodes');
  });
});

// PassiveSpec.lua:2043-2064
describe('a cluster subgraph’s centre', () => {
  it('is drawn, never hovered or picked', async () => {
    const id = 70001;
    const centre = { id, type: 'Mastery', name: 'Small Cluster', icon: '', stats: [], group: 0, orbit: 0, orbitIndex: 0, x: 40000, y: 40000, linked: [] };
    states.length = 0;
    const onhover = vi.fn<(n: Node | undefined) => void>();
    const onpick = vi.fn<(n: Node) => void>();
    const { canvas, component } = await mount({ onhover, onpick, cluster: [centre] });
    await vi.waitFor(() => expect(states.at(-1)?.cluster?.nodes.map((n) => n.id)).toContain(id));
    component.focus(id);
    await fireEvent.pointerMove(canvas, { clientX: W / 2, clientY: H / 2 });
    await fireEvent.pointerDown(canvas, { clientX: W / 2, clientY: H / 2, button: 0 });
    await fireEvent.pointerUp(canvas, { clientX: W / 2, clientY: H / 2, button: 0 });
    expect(onhover.mock.calls.map((c) => c[0]?.skill)).not.toContain(id);
    expect(onpick).not.toHaveBeenCalled();
  });
});
