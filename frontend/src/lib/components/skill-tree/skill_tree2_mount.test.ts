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
  const onclick = vi.fn((...call: [number, boolean, number, boolean, number]) => Promise.resolve(call.length ? results.shift() : undefined));
  const onattribute = vi.fn();
  const onhover = vi.fn((...[id]: [number, number]) => Promise.resolve({ path: [id, 3], depends: [] as number[] }));
  const onunhover = vi.fn();
  const onconnect = vi.fn(() => Promise.resolve(1));
  const view = render(SkillTree2, {
    data: tree,
    spec: {
      alloc: [1, 2, 3],
      allocMode: {},
      text: { 4: { name: 'Replaced Four', stats: ['20% increased Life'] } },
      ascendClassBaseName: '',
      notes: { 4: 'take it early' }
    },
    onclick,
    onattribute,
    onhover,
    onunhover,
    onconnect,
    compared: new Set([1, 2])
  });
  const canvas = view.container.querySelector('canvas')!;
  return {
    canvas,
    onclick,
    onattribute,
    onhover,
    onunhover,
    onconnect,
    component: view.component as unknown as { zoom: (how: 'in' | 'out' | 'fit') => void; project: (id: number) => { x: number; y: number } | undefined }
  };
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
    expect(onhover).toHaveBeenCalledWith(4, 0);
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
    expect(onclick.mock.calls).toEqual([[5, false, 0, false, 0]]);
    const dialog = await screen.findByRole('dialog', { name: 'Choose Attribute' });
    await fireEvent.click(Array.from(dialog.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Dexterity')!);
    expect(onattribute.mock.calls).toEqual([[5, 2, 0]]);
    expect(screen.queryByRole('dialog', { name: 'Choose Attribute' })).toBeNull();
  });

  it('confirms a class change with a second click, and does not click after a drag', async () => {
    const { canvas, onclick } = mount([3, 1]);
    await vi.waitFor(() => expect(views.at(-1)?.width).toBe(W));
    await press(canvas, at(4));
    const dialog = await screen.findByRole('dialog', { name: 'Class Change' });
    await fireEvent.click(Array.from(dialog.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Continue')!);
    expect(onclick.mock.calls).toEqual([
      [4, false, 0, false, 0],
      [4, false, 0, true, 0]
    ]);

    const p = at(4);
    await fireEvent.pointerMove(canvas, { clientX: p.x, clientY: p.y });
    await fireEvent.pointerDown(canvas, { clientX: p.x, clientY: p.y, button: 0 });
    await fireEvent.pointerMove(canvas, { clientX: p.x + 20, clientY: p.y });
    await fireEvent.pointerUp(window, { clientX: p.x + 20, clientY: p.y, button: 0 });
    expect(onclick).toHaveBeenCalledTimes(2);
    expect(views.at(-1)?.zoomX).toBe(20);
  });

  it('highlights the nodes the search matches, reading the spec text (PassiveTreeView.lua:859-882)', async () => {
    mount([]);
    await vi.waitFor(() => expect(views.at(-1)?.width).toBe(W));
    const field = screen.getByRole('searchbox', { name: 'Search passive nodes' });
    await fireEvent.input(field, { target: { value: 'replaced' } });
    await vi.waitFor(() => expect([...(states.at(-1)?.search ?? [])]).toEqual([4]));
    await fireEvent.input(field, { target: { value: '"notable four"' } });
    await vi.waitFor(() => expect([...(states.at(-1)?.search ?? [])]).toEqual([]));
    await fireEvent.input(field, { target: { value: 'attribute' } });
    await vi.waitFor(() => expect([...(states.at(-1)?.search ?? [])]).toEqual([5]));
    expect(states.at(-1)?.searchR).toBeCloseTo(140 / 1.2 ** (3 * 0.2));
  });

  it('switches the weapon-set allocation mode with Alt and the wheel, and clicks and hovers in it (PassiveTreeView.lua:190-195)', async () => {
    const { canvas, onclick, onhover } = mount([1]);
    await vi.waitFor(() => expect(views.at(-1)?.width).toBe(W));
    const zoom = views.at(-1)!.zoomLevel;
    await fireEvent.pointerMove(canvas, { clientX: at(4).x, clientY: at(4).y });
    await fireEvent.wheel(canvas, { deltaY: -100, altKey: true });
    await vi.waitFor(() => expect(states.at(-1)?.specAllocMode).toBe(1));
    expect(views.at(-1)!.zoomLevel).toBe(zoom);
    expect(screen.getByRole('status', { name: 'Weapon set allocation mode' }).textContent).toContain('Allocating Weapon set 1 Mode');
    expect(onhover.mock.calls.at(-1)).toEqual([4, 1]);
    for (let i = 0; i < 3; i++) await fireEvent.wheel(canvas, { deltaY: -100, altKey: true });
    await vi.waitFor(() => expect(states.at(-1)?.specAllocMode).toBe(2));
    await press(canvas, at(4));
    expect(onclick.mock.calls).toEqual([[4, false, 0, false, 2]]);
    for (let i = 0; i < 3; i++) await fireEvent.wheel(canvas, { deltaY: 100, altKey: true });
    await vi.waitFor(() => expect(states.at(-1)?.specAllocMode).toBe(0));
    expect(screen.queryByRole('status', { name: 'Weapon set allocation mode' })).toBeNull();
  });

  it('zooms by 3 with Shift and the wheel, also when the browser turns it into a horizontal scroll (PassiveTreeView.lua:222-225)', async () => {
    const { canvas } = mount([]);
    await vi.waitFor(() => expect(views.at(-1)?.width).toBe(W));
    const zoom = views.at(-1)!.zoomLevel;
    await fireEvent.wheel(canvas, { deltaY: -100 });
    await vi.waitFor(() => expect(views.at(-1)!.zoomLevel).toBe(zoom + 1));
    await fireEvent.wheel(canvas, { deltaY: -100, shiftKey: true });
    await vi.waitFor(() => expect(views.at(-1)!.zoomLevel).toBe(zoom + 4));
    await fireEvent.wheel(canvas, { deltaX: 100, shiftKey: true });
    await vi.waitFor(() => expect(views.at(-1)!.zoomLevel).toBe(zoom + 1));
  });

  it('does not take S, D or I typed into the search as an attribute hotkey (PassiveTreeView.lua:378-391)', async () => {
    const { canvas, onclick } = mount([1]);
    await vi.waitFor(() => expect(views.at(-1)?.width).toBe(W));
    const field = screen.getByRole('searchbox', { name: 'Search passive nodes' });
    await fireEvent.keyDown(field, { key: 's' });
    await press(canvas, at(5));
    expect(onclick.mock.calls).toEqual([[5, false, 0, false, 0]]);
  });

  it("reports leaving a node for the empty canvas, shows the node's note, and hands the compared spec to the renderer", async () => {
    const { canvas, onunhover } = mount([]);
    await vi.waitFor(() => expect(views.at(-1)?.width).toBe(W));
    await fireEvent.pointerMove(canvas, { clientX: at(4).x, clientY: at(4).y });
    // PassiveTreeView.lua:2086-2093
    expect((await screen.findByRole('tooltip', { name: 'Node tooltip' })).textContent).toContain('Note: take it early');
    expect([...(states.at(-1)?.compared ?? [])]).toEqual([1, 2]);
    await fireEvent.pointerMove(canvas, { clientX: 5, clientY: 5 });
    expect(onunhover).toHaveBeenCalledTimes(1);
  });

  it('zooms by 2 on a Ctrl+click and out on a Ctrl+right-click, allocating nothing (PassiveTreeView.lua:251-255)', async () => {
    const { canvas, onclick } = mount([1]);
    await vi.waitFor(() => expect(views.at(-1)?.width).toBe(W));
    const zoom = views.at(-1)!.zoomLevel;
    const p = at(4);
    await fireEvent.pointerMove(canvas, { clientX: p.x, clientY: p.y });
    await fireEvent.pointerDown(canvas, { clientX: p.x, clientY: p.y, button: 0, ctrlKey: true });
    await fireEvent.pointerUp(window, { clientX: p.x, clientY: p.y, button: 0, ctrlKey: true });
    await vi.waitFor(() => expect(views.at(-1)!.zoomLevel).toBe(zoom + 2));
    await fireEvent.contextMenu(canvas, { clientX: p.x, clientY: p.y, ctrlKey: true });
    await vi.waitFor(() => expect(views.at(-1)!.zoomLevel).toBe(zoom));
    expect(onclick).not.toHaveBeenCalled();
  });

  it('zooms a step at a time for the page, and projects a node onto the canvas (PassiveTreeView.lua:206-209)', async () => {
    const { component } = mount([]);
    await vi.waitFor(() => expect(views.at(-1)?.width).toBe(W));
    const zoom = views.at(-1)!.zoomLevel;
    component.zoom('in');
    await vi.waitFor(() => expect(views.at(-1)!.zoomLevel).toBe(zoom + 1));
    component.zoom('out');
    component.zoom('out');
    await vi.waitFor(() => expect(views.at(-1)!.zoomLevel).toBe(zoom - 1));
    expect(component.project(4)).toEqual(at(4));
  });

  it('offers Connect Path in the Class Change popup, which connects instead of resetting (PassiveTreeView.lua:513-529)', async () => {
    const { canvas, onclick, onconnect } = mount([3]);
    await vi.waitFor(() => expect(views.at(-1)?.width).toBe(W));
    await press(canvas, at(4));
    const dialog = await screen.findByRole('dialog', { name: 'Class Change' });
    expect(dialog.textContent).toContain('This can be avoided by connecting one of the');
    await fireEvent.click(Array.from(dialog.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Connect Path')!);
    expect(onconnect.mock.calls).toEqual([[4, 0, 0]]);
    expect(onclick).toHaveBeenCalledTimes(1);
  });

  it('undoes with Ctrl+Z and redoes with Ctrl+Y, not while typing in the search (TreeTab.lua:355-364)', async () => {
    const onundo = vi.fn();
    const onredo = vi.fn();
    render(SkillTree2, {
      data: tree,
      spec: { alloc: [1], allocMode: {}, text: {}, ascendClassBaseName: '' },
      onclick: vi.fn(),
      onattribute: vi.fn(),
      onhover: vi.fn(() => Promise.resolve({ path: [], depends: [] })),
      onundo,
      onredo
    });
    await fireEvent.keyDown(window, { key: 'z', ctrlKey: true });
    await fireEvent.keyDown(window, { key: 'y', ctrlKey: true });
    await fireEvent.keyDown(window, { key: 'z' });
    await fireEvent.keyDown(screen.getByRole('searchbox', { name: 'Search passive nodes' }), { key: 'z', ctrlKey: true });
    expect(onundo).toHaveBeenCalledTimes(1);
    expect(onredo).toHaveBeenCalledTimes(1);
  });
});
