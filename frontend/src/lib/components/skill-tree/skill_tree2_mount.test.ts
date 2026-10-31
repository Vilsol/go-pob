// @vitest-environment jsdom
// PassiveTreeView.lua:244-251, 291-299, 425-594 and TreeTab.lua:851-876 (PoB-PoE2)

import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';

import { buildScene2, treeToScreen, type Scene2State, type View2 } from './scene2';
import { tinyTree } from './tree2_fixture';

afterEach(cleanup);

const states: Scene2State[] = [];
const views: View2[] = [];
vi.mock('./gl_renderer2', () => ({
  createTree2Renderer: () => ({
    kind: 'canvas',
    setView: (v: View2) => views.push(v),
    setState: (s: Scene2State) => states.push(s),
    destroy: () => {}
  })
}));

const SkillTree2 = (await import('./SkillTree2.svelte')).default;

const W = 800;
const H = 600;
const errors: string[] = [];
beforeAll(() => {
  globalThis.ResizeObserver ??= class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
  globalThis.PointerEvent ??= class extends MouseEvent {} as typeof PointerEvent;
  Element.prototype.getBoundingClientRect = () => ({ left: 0, top: 0, right: W, bottom: H, width: W, height: H, x: 0, y: 0, toJSON: () => ({}) });
  window.addEventListener('error', (e) => errors.push(e.message));
});

const tree = tinyTree();
const scene = buildScene2(tree);
const at = (id: number) => {
  const n = tree.nodes.find((x) => x.id === id)!;
  return treeToScreen(scene, views.at(-1)!, n.x, n.y);
};

const mount = (results: number[]) => {
  const onclick = vi.fn((...call: [number, boolean, number, boolean]) => Promise.resolve(call.length ? results.shift() : undefined));
  const onattribute = vi.fn();
  const onhover = vi.fn((id: number) => Promise.resolve({ path: [id, 3], depends: [] as number[] }));
  const view = render(SkillTree2, {
    data: tree,
    spec: { alloc: [1, 2, 3], allocMode: {}, text: { 4: { name: 'Replaced Four', stats: ['20% increased Life'] } }, ascendClassBaseName: '' },
    onclick,
    onattribute,
    onhover
  });
  const canvas = view.container.querySelector('canvas')!;
  return { canvas, onclick, onattribute, onhover };
};

const press = async (canvas: HTMLCanvasElement, p: { x: number; y: number }) => {
  await fireEvent.pointerMove(canvas, { clientX: p.x, clientY: p.y });
  await fireEvent.pointerDown(canvas, { clientX: p.x, clientY: p.y, button: 0 });
  await fireEvent.pointerUp(window, { clientX: p.x, clientY: p.y, button: 0 });
};

describe('the PoE2 tree view', () => {
  it('hovers a node: the tooltip shows the spec text, the hover path reaches the renderer', async () => {
    errors.length = 0;
    const { canvas, onhover } = mount([]);
    await vi.waitFor(() => expect(views.at(-1)?.width).toBe(W));
    await fireEvent.pointerMove(canvas, { clientX: at(4).x, clientY: at(4).y });
    expect(onhover).toHaveBeenCalledWith(4);
    const tip = await screen.findByRole('tooltip', { name: 'Node tooltip' });
    expect(tip.textContent).toContain('Replaced Four');
    expect(tip.textContent).toContain('20% increased Life');
    await vi.waitFor(() => expect([...(states.at(-1)?.hoverPath ?? [])]).toEqual([4, 3]));
    expect(states.at(-1)?.hovered).toBe(4);
    expect([...states.at(-1)!.alloc]).toEqual([1, 2, 3]);
    expect(errors).toEqual([]);
  });

  it('clicks a node, and asks for the attribute when the click says so', async () => {
    const { canvas, onclick, onattribute } = mount([2]);
    await vi.waitFor(() => expect(views.at(-1)?.width).toBe(W));
    await press(canvas, at(5));
    expect(onclick.mock.calls).toEqual([[5, false, 0, false]]);
    const dialog = await screen.findByRole('dialog', { name: 'Choose Attribute' });
    await fireEvent.click(Array.from(dialog.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Dexterity')!);
    expect(onattribute.mock.calls).toEqual([[5, 2]]);
    expect(screen.queryByRole('dialog', { name: 'Choose Attribute' })).toBeNull();
  });

  it('confirms a class change with a second click, and does not click after a drag', async () => {
    const { canvas, onclick } = mount([3, 1]);
    await vi.waitFor(() => expect(views.at(-1)?.width).toBe(W));
    await press(canvas, at(4));
    const dialog = await screen.findByRole('dialog', { name: 'Class Change' });
    await fireEvent.click(Array.from(dialog.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Continue')!);
    expect(onclick.mock.calls).toEqual([
      [4, false, 0, false],
      [4, false, 0, true]
    ]);

    const p = at(4);
    await fireEvent.pointerMove(canvas, { clientX: p.x, clientY: p.y });
    await fireEvent.pointerDown(canvas, { clientX: p.x, clientY: p.y, button: 0 });
    await fireEvent.pointerMove(canvas, { clientX: p.x + 20, clientY: p.y });
    await fireEvent.pointerUp(window, { clientX: p.x + 20, clientY: p.y, button: 0 });
    expect(onclick).toHaveBeenCalledTimes(2);
    expect(views.at(-1)?.zoomX).toBe(20);
  });
});
