// @vitest-environment jsdom

import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import { brotliDecompressSync } from 'node:zlib';

import { cachedAsset } from '../../../e2e/support/data_cache';
import type { Tree } from '$lib/skill_tree/types';

afterEach(cleanup);

const rawJson = (v: string) => {
  const a = cachedAsset(`https://data.pob.gg/data/${v.replace('_', '.')}/tree/data.json.br`);
  if (a.status !== 200) throw new Error(`REFUSING: no ${v} tree in the bundle cache`);
  return brotliDecompressSync(a.body).toString('utf8');
};

const calls = vi.hoisted(() => [] as string[]);
vi.mock('$lib/go/worker', () => {
  const known: Record<string, (...a: unknown[]) => unknown> = {
    GetTree: (v) => rawJson(v as string),
    GetBuildInfo: () => ({ className: 'Witch', ascendClassName: 'None', secondaryAscendClassId: 0 }),
    GetPassiveNodes: () => [],
    GetLeapHeldNodes: () => [],
    CalculateAllocationPaths: () => ({}),
    GetGrantedPassives: () => []
  };
  const syncWrap = new Proxy(
    {},
    {
      get:
        (_, name: string) =>
        (...args: unknown[]) => {
          calls.push(name);
          return Promise.resolve(known[name]?.(...args));
        }
    }
  );
  return { syncWrap, syncWorker: undefined };
});
vi.mock('$lib/components/skill-tree/renderer', () => ({
  createRenderer: () => ({ kind: 'canvas', setView: () => {}, setState: () => {}, complete: () => true, drawListArt: () => undefined, destroy: () => {} })
}));

const st = await import('$lib/skill_tree');
const { currentBuild } = await import('$lib/global');
const TimelessTreeView = (await import('./TimelessTreeView.svelte')).default;

globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
};
globalThis.PointerEvent ??= class extends MouseEvent {} as typeof PointerEvent;

const W = 800;
const H = 560;

let tree: Tree;
beforeAll(async () => {
  await st.loadSkillTree('3_29');
  st.skillTree.subscribe((t) => (tree = t!))();
  Object.defineProperty(HTMLElement.prototype, 'offsetWidth', { configurable: true, get: () => W });
  Object.defineProperty(HTMLElement.prototype, 'offsetHeight', { configurable: true, get: () => H });
  currentBuild.set({} as never);
}, 60_000);

const text = (s: string) => createRawSnippet(() => ({ render: () => `<span>${s}</span>` }));

describe('the timeless wizard’s live tree', () => {
  it('centres the socket with its radius in view, hovers and picks sockets only, zooms, asks no what-if', async () => {
    const socket = Object.values(tree.nodes).find((n) => n.isJewelSocket && !n.expansionJewel && !n.ascendancyName && n.group !== undefined)!.skill!;
    const enter = vi.fn();
    const leave = vi.fn();
    const pick = vi.fn();
    calls.length = 0;
    const v = render(TimelessTreeView, {
      tree,
      version: '3_29',
      rings: [{ id: socket }],
      picked: new Set([socket]),
      centre: socket,
      hovered: null,
      sockets: new Set([socket]),
      enter,
      leave,
      pick,
      tip: createRawSnippet((id: () => number) => ({ render: () => `<span>socket ${id()}</span>` })),
      legend: text('legend')
    });
    const ring = await vi.waitFor(() => v.container.querySelector<HTMLElement>(`.radius-ring[data-ring="${socket}"]`)!);
    const x = parseFloat(ring.style.getPropertyValue('--x'));
    const y = parseFloat(ring.style.getPropertyValue('--y'));
    const z = parseFloat(ring.style.getPropertyValue('--z'));
    expect(Math.abs(x + z / 2 - W / 2)).toBeLessThan(1);
    expect(Math.abs(y + z / 2 - H / 2)).toBeLessThan(1);
    expect(z).toBeLessThan(Math.min(W, H));
    expect(z).toBeGreaterThan(Math.min(W, H) / 2);
    expect(v.container.querySelectorAll('[data-conq]').length).toBeGreaterThan(0);

    const canvas = v.container.querySelector('canvas')!;
    await fireEvent.pointerMove(canvas, { clientX: W / 2, clientY: H / 2 });
    await vi.waitFor(() => expect(enter).toHaveBeenCalledWith(socket));
    await fireEvent.pointerDown(canvas, { clientX: W / 2, clientY: H / 2, button: 0 });
    await fireEvent.pointerUp(canvas, { clientX: W / 2, clientY: H / 2, button: 0 });
    expect(pick).toHaveBeenCalledWith(socket);
    await v.rerender({ hovered: socket });
    expect(v.getByRole('tooltip').textContent).toContain(`socket ${socket}`);

    pick.mockClear();
    await fireEvent.pointerMove(canvas, { clientX: W / 2 + z / 2 - 40, clientY: H / 2 });
    await fireEvent.pointerMove(canvas, { clientX: 5, clientY: 5 });
    await vi.waitFor(() => expect(leave).toHaveBeenCalled());

    await fireEvent.wheel(canvas, { deltaY: 300, clientX: W / 2, clientY: H / 2 });
    await vi.waitFor(() => expect(parseFloat(v.container.querySelector<HTMLElement>('.radius-ring')!.style.getPropertyValue('--z'))).toBeLessThan(z));
    expect(calls).not.toContain('StatCompareNodes');
    expect(calls).not.toContain('PrepareStatCompare');
  });
});
