import { afterEach, describe, expect, it, vi } from 'vitest';

import { dump } from './type_utils';

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('dump handles an unclonable object as an expected condition', () => {
  it('does not log when structuredClone refuses a nested function', () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => {});
    const trace = vi.spyOn(console, 'trace').mockImplementation(() => {});

    const out = dump({ keep: 1, release: () => {} });

    expect(out).toEqual({ keep: 1 });
    expect(error).not.toHaveBeenCalled();
    expect(trace).not.toHaveBeenCalled();
  });

  it('still returns the clonable fields, including nested objects', () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.spyOn(console, 'trace').mockImplementation(() => {});

    const out = dump({
      n: 7,
      s: 'x',
      nested: { deep: 1, release: () => {} },
      release: () => {}
    });

    expect(out).toEqual({ n: 7, s: 'x', nested: { deep: 1 } });
  });

  it('still logs an error that is not a DataCloneError', () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.stubGlobal('structuredClone', () => {
      throw new TypeError('something genuinely unexpected');
    });

    const out = dump({ keep: 1 });

    expect(error).toHaveBeenCalled();
    expect(out).toEqual({ keep: 1 });
  });

  it('the V8 DataCloneError message is not a fixed string', () => {
    let caught: unknown;
    try {
      structuredClone({ f: () => {} });
    } catch (e) {
      caught = e;
    }

    expect(caught).toBeInstanceOf(DOMException);
    expect((caught as DOMException).name).toBe('DataCloneError');
    expect((caught as DOMException).message).not.toBe('Function object could not be cloned.');
    expect((caught as DOMException).message).toContain('could not be cloned');
  });

  it('with release, frees every wrapper it walked, after reading it', () => {
    const released: string[] = [];
    const wrapper = (name: string, fields: object) => ({
      ...fields,
      release: () => {
        released.push(name);
      }
    });
    const inner = wrapper('inner', { deep: 2 });
    const top = wrapper('top', { n: 1, inner, map: { a: wrapper('item', { k: 3 }) } });

    expect(dump(top, true)).toEqual({ n: 1, inner: { deep: 2 }, map: { a: { k: 3 } } });
    expect(released.sort()).toEqual(['inner', 'item', 'top']);

    released.length = 0;
    dump(top);
    expect(released, 'a plain dump must not release').toEqual([]);
  });

  it('with release, frees wrappers from getters that marshal afresh on each read', () => {
    let made = 0;
    let released = 0;
    const fresh = () => {
      made++;
      return {
        k: 1,
        release: () => {
          released++;
        }
      };
    };
    const top = {
      get items() {
        return [fresh(), fresh()];
      },
      release: () => {
        released++;
      }
    };
    made++;

    const out = dump(top, true);
    expect(released, 'wrappers made but never released').toBe(made);
    expect(out).toEqual({ items: [{ k: 1 }, { k: 1 }] });
  });

  it('drops the hidden Go handle a crystalline wrapper carries (a stale number, ~7 KB per detail tick)', () => {
    const wrapper = { a: 1, nested: { b: 2 } };
    Object.defineProperty(wrapper, '__crystallineHandle', { value: 42, enumerable: false });
    Object.defineProperty(wrapper.nested, '__crystallineHandle', { value: 43, enumerable: false });
    const out = dump(wrapper);
    expect(out).toEqual({ a: 1, nested: { b: 2 } });
    expect(Object.getOwnPropertyNames(out)).not.toContain('__crystallineHandle');
    expect(Object.getOwnPropertyNames(out.nested)).not.toContain('__crystallineHandle');
  });

  it('passes through non-objects untouched', () => {
    expect(dump(5)).toBe(5);
    expect(dump('x')).toBe('x');
    expect(dump(null)).toBe(null);
  });
});
