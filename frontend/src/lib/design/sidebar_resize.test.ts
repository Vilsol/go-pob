// @vitest-environment jsdom

import { describe, expect, it, afterEach, beforeEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import type { StatLine } from '$lib/display/stat_list';
import Sidebar from './Sidebar.svelte';

globalThis.PointerEvent ??= class extends MouseEvent {
  pointerId: number;
  constructor(type: string, init: PointerEventInit = {}) {
    super(type, init);
    this.pointerId = init.pointerId ?? 0;
  }
} as unknown as typeof PointerEvent;

const KEY = 'go-pob:sidebar-width';
afterEach(cleanup);
beforeEach(() => localStorage.removeItem(KEY));

const line = { label: 'Total Life', labelColor: '', value: '5,123', valueColor: '' } as StatLine;
const mount = () => render(Sidebar, { props: { sidebar: { sections: [{ groups: [[line]] }], warnings: [] }, socketGroups: ['Group'], mainSocketGroup: 0 } });
const parts = (r: ReturnType<typeof mount>) => ({
  aside: r.getByRole('complementary', { name: 'Main skill and stats' }),
  handle: r.getByRole('separator', { name: 'Resize the sidebar' })
});

describe('sidebar width', () => {
  it('opens at 320 px with a keyboard-reachable handle', () => {
    const { aside, handle } = parts(mount());
    expect(aside.style.width).toBe('320px');
    expect(handle.tabIndex).toBe(0);
    expect(handle.getAttribute('aria-valuenow')).toBe('320');
    expect([handle.getAttribute('aria-valuemin'), handle.getAttribute('aria-valuemax')]).toEqual(['240', '640']);
  });

  it('moves with the arrow keys, clamped, and the next mount opens at the kept width', async () => {
    const r = mount();
    const { aside, handle } = parts(r);
    await fireEvent.keyDown(handle, { key: 'ArrowRight' });
    expect(aside.style.width).toBe('336px');
    for (let i = 0; i < 40; i++) await fireEvent.keyDown(handle, { key: 'ArrowLeft' });
    expect(aside.style.width).toBe('240px');
    expect(handle.getAttribute('aria-valuenow')).toBe('240');
    expect(localStorage.getItem(KEY)).toBe('240');
    cleanup();
    expect(parts(mount()).aside.style.width).toBe('240px');
  });

  it('follows a drag of its edge and stops at the maximum', async () => {
    const { aside, handle } = parts(mount());
    await fireEvent.pointerDown(handle, { button: 0, clientX: 320, pointerId: 1 });
    await fireEvent.pointerMove(handle, { clientX: 400, pointerId: 1 });
    expect(aside.style.width).toBe('400px');
    await fireEvent.pointerMove(handle, { clientX: 2000, pointerId: 1 });
    await fireEvent.pointerUp(handle, { pointerId: 1 });
    expect(aside.style.width).toBe('640px');
    await fireEvent.pointerMove(handle, { clientX: 300, pointerId: 1 });
    expect(aside.style.width).toBe('640px');
  });

  it('opens at the default when the kept value is not a width', () => {
    localStorage.setItem(KEY, 'wide');
    expect(parts(mount()).aside.style.width).toBe('320px');
  });
});
