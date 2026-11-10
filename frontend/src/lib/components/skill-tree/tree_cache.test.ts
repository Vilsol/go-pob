// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest';

import { objectId, park, unpark, type Parked } from './tree_cache';

const entry = (key: string): Parked<{ destroy: () => void }, object> => {
  const canvas = document.createElement('canvas');
  document.body.append(canvas);
  return { key, canvas, renderer: { destroy: vi.fn() }, scene: {}, onDraw: vi.fn() };
};

describe('the kept tree renderer', () => {
  it('comes back for the same key, out of the document and drawing nowhere', () => {
    const e = entry('a');
    park(e);
    expect(e.canvas.isConnected).toBe(false);
    e.onDraw(1);
    const back = unpark('a');
    expect(back).toBe(e);
    expect(e.renderer.destroy).not.toHaveBeenCalled();
    expect(unpark('a'), 'a kept renderer comes back once').toBeUndefined();
  });

  it('is released by another key, by another park and by its context being lost', () => {
    const a = entry('a');
    park(a);
    expect(unpark('b')).toBeUndefined();
    expect(a.renderer.destroy).toHaveBeenCalledOnce();

    const b = entry('b');
    const c = entry('c');
    park(b);
    park(c);
    expect(b.renderer.destroy).toHaveBeenCalledOnce();
    expect(c.renderer.destroy).not.toHaveBeenCalled();

    c.canvas.dispatchEvent(new Event('webglcontextlost'));
    expect(c.renderer.destroy).toHaveBeenCalledOnce();
    expect(unpark('c')).toBeUndefined();
  });

  it('numbers each object once', () => {
    const [x, y] = [{}, {}];
    expect(objectId(x)).toBe(objectId(x));
    expect(objectId(x)).not.toBe(objectId(y));
  });
});
