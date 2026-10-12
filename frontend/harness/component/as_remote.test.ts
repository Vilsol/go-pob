import { describe, expect, it } from 'vitest';

import { asRemote } from './as_remote';

const flush = () => new Promise((r) => setTimeout(r, 0));

describe('asRemote gives an object Comlink-remote semantics', () => {
  it('turns a plain property read into a thenable', async () => {
    const r = asRemote({ name: 'go-pob' });
    expect(r.name).not.toBe('go-pob');
    expect(await r.name).toBe('go-pob');
  });

  it('keeps methods callable, and typeof reports function', async () => {
    const r = asRemote({ double: (n: number) => n * 2 });
    expect(typeof r.double).toBe('function');
    expect(await r.double(21)).toBe(42);
  });

  it('a void method still returns something with .catch and .finally', async () => {
    const seen: string[] = [];
    const r = asRemote({ set: (v: string) => void seen.push(v) });

    let finallyRan = false;
    await r
      .set('x')
      .catch(() => expect.unreachable('must not reject'))
      .finally(() => (finallyRan = true));

    expect(seen).toEqual(['x']);
    expect(finallyRan).toBe(true);
  });

  it('routes a thrown error into .catch rather than escaping synchronously', async () => {
    const r = asRemote({
      boom: (): void => {
        throw new Error('from go');
      }
    });

    let caught: unknown;
    await r.boom().catch((e: unknown) => (caught = e));
    expect((caught as Error).message).toBe('from go');
  });

  it('resolves a method result as RAW data, not another proxy', async () => {
    const r = asRemote({ list: () => [{ label: 'a' }, { label: 'b' }] });
    const out = await r.list();
    expect(Array.isArray(out)).toBe(true);
    expect(out.map((x) => x.label)).toEqual(['a', 'b']);
  });

  it('chains property access without awaiting each hop', async () => {
    const r = asRemote({ Skills: { SkillSets: [{ Skills: [{ Label: 'deep' }] }] } });
    expect(await r.Skills.SkillSets[0].Skills[0].Label).toBe('deep');
  });

  it('reads undefined through a missing branch instead of throwing', async () => {
    const r = asRemote({} as { missing?: { deeper?: string } });
    expect(await r.missing?.deeper).toBeUndefined();
  });

  it('dispatches a call that is never awaited', async () => {
    let calls = 0;
    const r = asRemote({ bump: () => void calls++ });

    void r.bump();
    expect(calls).toBe(0);

    await flush();
    expect(calls).toBe(1);
  });

  it('dispatches once per call, not once per proxy', async () => {
    let calls = 0;
    const r = asRemote({ bump: () => void calls++ });

    void r.bump();
    void r.bump();
    void r.bump();
    await flush();

    expect(calls).toBe(3);
  });

  it('does not re-run the underlying read for each .then on one property proxy', async () => {
    let reads = 0;
    const r = asRemote({
      get counted() {
        reads++;
        return reads;
      }
    });

    const p = r.counted;
    expect(await p).toBe(1);
    expect(await p).toBe(1);
    expect(reads).toBe(1);
  });

  it('answers symbol probes with undefined', () => {
    const r = asRemote({ a: 1 }) as unknown as Record<symbol, unknown>;
    expect(r[Symbol.iterator]).toBeUndefined();
    expect(r[Symbol.toPrimitive]).toBeUndefined();
  });

  it('is truthy, so `syncWrap?.X` and `if (syncWrap)` take the real path', () => {
    expect(Boolean(asRemote({}))).toBe(true);
  });

  it('does not answer protocol names the runtime probes and then calls', () => {
    const r = asRemote({ real: 1 }) as unknown as Record<string, unknown>;
    for (const key of ['toJSON', 'toString', 'valueOf', 'constructor', 'inspect', 'asymmetricMatch']) {
      expect(r[key], `${key} must not look callable`).toBeUndefined();
    }
  });

  it('survives JSON.stringify without rejecting', () => {
    expect(() => JSON.stringify(asRemote({ a: 1 }))).not.toThrow();
  });

  it('writes a property through to the underlying object', async () => {
    const target = { Enabled: true };
    const r = asRemote(target) as unknown as { Enabled: boolean };

    r.Enabled = false;

    await flush();
    expect(target.Enabled).toBe(false);
  });

  it('writes through a nested path', async () => {
    const target = { Skills: { Sets: [{ Enabled: true }] } };
    const r = asRemote(target) as unknown as { Skills: { Sets: { Enabled: boolean }[] } };

    r.Skills.Sets[0].Enabled = false;

    await flush();
    expect(target.Skills.Sets[0].Enabled).toBe(false);
  });

  it('writes a value that arrives as a promise once it settles', async () => {
    const target = { n: 0 };
    const r = asRemote(target) as unknown as { n: number | Promise<number> };

    r.n = Promise.resolve(42);

    await flush();
    expect(target.n).toBe(42);
  });

  it('names the path when a non-function is called', async () => {
    const r = asRemote({ nested: { notAFunction: 5 } }, 'syncWrap') as unknown as {
      nested: { notAFunction: () => Promise<unknown> };
    };

    await expect(r.nested.notAFunction()).rejects.toThrow('syncWrap.nested.notAFunction()');
  });
});
