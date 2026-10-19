// @vitest-environment jsdom
import './pointer_event_polyfill';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { fireEvent } from '@testing-library/svelte';
import { dragSource, dropZone, type DropZone } from './reorder';

const el = (parent: HTMLElement = document.body) => parent.appendChild(document.createElement('div'));
const ptr = (x: number, extra: Record<string, unknown> = {}) => ({
  pointerId: 1,
  isPrimary: true,
  button: 0,
  pointerType: 'mouse',
  clientX: x,
  clientY: 0,
  ...extra
});

afterEach(() => {
  document.body.innerHTML = '';
  vi.useRealTimers();
});

const setup = () => {
  const src = el();
  const outer = el();
  const inner = el(outer);
  const calls: string[] = [];
  const zone = (name: string, z: DropZone = {}): DropZone => ({
    over: () => calls.push(`over ${name}`),
    leave: () => calls.push(`leave ${name}`),
    drop: (d) => calls.push(`drop ${name} ${JSON.stringify(d)}`),
    ...z
  });
  const onend = vi.fn();
  const source = dragSource(src, { data: () => ({ n: 1 }), label: 'Thing', onend });
  dropZone(outer, zone('outer'));
  dropZone(inner, zone('inner'));
  return { src, outer, inner, calls, onend, source };
};

describe('dragSource / dropZone', () => {
  it('a mouse press under the threshold is a click, not a drag', async () => {
    const { src, inner, calls, onend } = setup();
    await fireEvent.pointerDown(src, ptr(0));
    await fireEvent.pointerMove(inner, ptr(2));
    await fireEvent.pointerUp(inner, ptr(2));
    expect(calls).toEqual([]);
    expect(onend).not.toHaveBeenCalled();
    expect(document.querySelector('[data-drag-label]')).toBeNull();
  });

  it('drops onto every accepting zone under the pointer, innermost first, with a label following the pointer', async () => {
    const { src, inner, calls, onend } = setup();
    await fireEvent.pointerDown(src, ptr(0));
    await fireEvent.pointerMove(inner, ptr(20));
    const label = document.querySelector<HTMLElement>('[data-drag-label]');
    expect(label?.textContent).toBe('Thing');
    expect(label?.style.left).toBe('20px');
    await fireEvent.pointerUp(inner, ptr(20));
    expect(calls).toEqual(['over inner', 'over outer', 'drop inner {"n":1}', 'drop outer {"n":1}']);
    expect(onend).toHaveBeenCalledOnce();
    expect(document.querySelector('[data-drag-label]')).toBeNull();
  });

  it('leaves a zone the pointer moves out of, and skips zones that refuse the data', async () => {
    const { src, outer, inner, calls } = setup();
    dropZone(inner, { accepts: () => false, over: () => calls.push('over refusing') });
    await fireEvent.pointerDown(src, ptr(0));
    await fireEvent.pointerMove(inner, ptr(20));
    await fireEvent.pointerMove(document.body, ptr(30));
    await fireEvent.pointerUp(document.body, ptr(30));
    expect(calls).toEqual(['over outer', 'leave outer']);
    void outer;
  });

  it('Escape cancels the drag and nothing drops', async () => {
    const { src, inner, calls, onend } = setup();
    await fireEvent.pointerDown(src, ptr(0));
    await fireEvent.pointerMove(inner, ptr(20));
    await fireEvent.keyDown(window, { key: 'Escape' });
    await fireEvent.pointerUp(inner, ptr(20));
    expect(calls).toEqual(['over inner', 'over outer', 'leave inner', 'leave outer']);
    expect(onend).toHaveBeenCalledOnce();
  });

  it('swallows the click that ends a drag', async () => {
    const { src, inner } = setup();
    const click = vi.fn();
    inner.addEventListener('click', click);
    await fireEvent.pointerDown(src, ptr(0));
    await fireEvent.pointerMove(inner, ptr(20));
    await fireEvent.pointerUp(inner, ptr(20));
    await fireEvent.click(inner);
    expect(click).not.toHaveBeenCalled();
  });

  it('on touch, a swipe before the hold scrolls instead of dragging; a hold then a move drags', async () => {
    vi.useFakeTimers();
    const { src, inner, calls } = setup();
    const touch = (x: number) => ptr(x, { pointerType: 'touch' });
    await fireEvent.pointerDown(src, touch(0));
    await fireEvent.pointerMove(inner, touch(20));
    vi.advanceTimersByTime(400);
    await fireEvent.pointerMove(inner, touch(40));
    await fireEvent.pointerUp(inner, touch(40));
    expect(calls).toEqual([]);

    await fireEvent.pointerDown(src, touch(0));
    vi.advanceTimersByTime(400);
    const scroll = new Event('touchmove', { cancelable: true });
    await fireEvent.pointerMove(inner, touch(20));
    window.dispatchEvent(scroll);
    expect(scroll.defaultPrevented, 'the page does not scroll under a drag').toBe(true);
    await fireEvent.pointerUp(inner, touch(20));
    expect(calls).toContain('drop inner {"n":1}');
  });

  it('keeps dragging when the source unmounts mid-drag', async () => {
    const { src, inner, calls, source } = setup();
    await fireEvent.pointerDown(src, ptr(0));
    await fireEvent.pointerMove(inner, ptr(20));
    source.destroy();
    src.remove();
    await fireEvent.pointerUp(inner, ptr(20));
    expect(calls).toContain('drop inner {"n":1}');
  });
});
