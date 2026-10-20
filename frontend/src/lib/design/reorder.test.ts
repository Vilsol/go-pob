// @vitest-environment jsdom
import './pointer_event_polyfill';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent } from '@testing-library/svelte';
import { Reorder, reorderState } from './reorder';

const ptr = (y: number, pointerType = 'mouse') => ({ pointerId: 1, isPrimary: true, button: 0, pointerType, clientX: 10, clientY: y });

const setup = () => {
  const box = document.body.appendChild(document.createElement('div'));
  const moves: [number, number][] = [];
  const drag = new Reorder(reorderState(), { box: () => box, name: (i) => `row ${i}`, onmove: (f, t) => moves.push([f, t]) });
  const rows = [0, 1, 2].map((i) => {
    const row = box.appendChild(document.createElement('div'));
    const { onpointerdown, onkeydown, ...attrs } = drag.row(i);
    for (const [k, v] of Object.entries(attrs)) if (v !== undefined) row.setAttribute(k, String(v));
    row.addEventListener('pointerdown', onpointerdown as EventListener);
    row.addEventListener('keydown', onkeydown as EventListener);
    row.getBoundingClientRect = () => ({ top: i * 30, bottom: i * 30 + 28, left: 0, right: 100, width: 100, height: 28, x: 0, y: i * 30, toJSON: () => ({}) });
    return row;
  });
  return { rows, moves };
};

const scroll = () => {
  const e = new Event('touchmove', { cancelable: true });
  window.dispatchEvent(e);
  return e.defaultPrevented;
};

beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal('matchMedia', () => ({ matches: true }));
});
afterEach(() => {
  document.body.innerHTML = '';
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('Reorder pointer drag', () => {
  it('a mouse drag starts as soon as the pointer moves', async () => {
    const { rows, moves } = setup();
    await fireEvent.pointerDown(rows[0], ptr(5));
    await fireEvent.pointerMove(window, ptr(80));
    expect(document.querySelector('[data-reorder-ghost]')).not.toBeNull();
    await fireEvent.pointerUp(window, ptr(80));
    vi.runAllTimers();
    expect(moves).toEqual([[0, 2]]);
  });

  it('on touch, a swipe before the hold scrolls and moves nothing', async () => {
    const { rows, moves } = setup();
    await fireEvent.pointerDown(rows[0], ptr(5, 'touch'));
    await fireEvent.pointerMove(window, ptr(25, 'touch'));
    expect(scroll(), 'the list scrolls under a swipe').toBe(false);
    vi.advanceTimersByTime(400);
    await fireEvent.pointerMove(window, ptr(80, 'touch'));
    await fireEvent.pointerUp(window, ptr(80, 'touch'));
    vi.runAllTimers();
    expect(document.querySelector('[data-reorder-ghost]')).toBeNull();
    expect(moves).toEqual([]);
  });

  it('on touch, a hold then a move drags, and the page does not scroll under it', async () => {
    const { rows, moves } = setup();
    await fireEvent.pointerDown(rows[0], ptr(5, 'touch'));
    await fireEvent.pointerMove(window, ptr(80, 'touch'));
    expect(document.querySelector('[data-reorder-ghost]'), 'no drag before the hold').toBeNull();
    await fireEvent.pointerUp(window, ptr(80, 'touch'));

    await fireEvent.pointerDown(rows[0], ptr(5, 'touch'));
    vi.advanceTimersByTime(400);
    expect(scroll(), 'a held touch stops the page scrolling').toBe(true);
    await fireEvent.pointerMove(window, ptr(80, 'touch'));
    expect(document.querySelector('[data-reorder-ghost]')).not.toBeNull();
    await fireEvent.pointerUp(window, ptr(80, 'touch'));
    vi.runAllTimers();
    expect(moves).toEqual([[0, 2]]);
    expect(scroll(), 'the block ends with the drag').toBe(false);
  });
});
