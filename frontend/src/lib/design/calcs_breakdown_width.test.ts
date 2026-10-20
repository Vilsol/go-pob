// @vitest-environment jsdom

import { describe, expect, it, afterEach, beforeEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import CalcsBreakdown from './CalcsBreakdown.svelte';

const KEY = 'go-pob:calcs-breakdown-width';
afterEach(cleanup);
const innerWidth = window.innerWidth;
afterEach(() => (window.innerWidth = innerWidth));
beforeEach(() => localStorage.removeItem(KEY));

const mount = () => render(CalcsBreakdown, { props: { id: 'bd', parts: [], pinned: false, onunpin: () => {} } });
const parts = (r: ReturnType<typeof mount>) => ({
  aside: r.getByRole('complementary', { name: 'Breakdown' }),
  handle: r.getByRole('separator', { name: 'Resize breakdown' })
});

describe('calcs breakdown width', () => {
  it('opens at 400 px on a wide window when no width was kept', () => {
    window.innerWidth = 1920;
    const { aside, handle } = parts(mount());
    expect(aside.style.width).toBe('400px');
    expect(handle.getAttribute('aria-valuenow')).toBe('400');
  });

  it('opens at 320 px on a 960 px window, as before', () => {
    window.innerWidth = 960;
    expect(parts(mount()).aside.style.width).toBe('320px');
  });

  it('opens at a kept width, and a resize is kept for the next mount', async () => {
    localStorage.setItem(KEY, '320');
    const r = mount();
    const { aside, handle } = parts(r);
    expect(aside.style.width).toBe('320px');
    await fireEvent.keyDown(handle, { key: 'ArrowLeft' });
    expect(aside.style.width).toBe('336px');
    r.unmount();
    expect(parts(mount()).aside.style.width).toBe('336px');
  });
});
