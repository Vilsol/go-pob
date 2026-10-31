import 'fake-indexeddb/auto';
import { afterEach, describe, expect, it, vi } from 'vitest';

vi.unmock('@zenfs/core');
vi.unmock('@zenfs/dom');

const { fs } = await import('@zenfs/core');
const { remountStore } = await import('$lib/go/storage_config');
const { TabStorage } = await import('./tabs');

const text = (s: string) => new TextEncoder().encode(s);
const open: InstanceType<typeof TabStorage>[] = [];
let n = 0;

// each tab is a realm with its own zenfs; here each tab mounts the shared database at its own point
const tabs = (store: string, lazy?: number) => {
  const lock = `test-storage-${++n}`;
  const tab = async (name: string) => {
    const point = `/${n}-${name}`;
    await remountStore(point, store);
    const changes: string[] = [];
    const t = new TabStorage(lock, {
      buckets: [store],
      apply: async (w) => {
        if (w.kind === 'set') await fs.promises.writeFile(`${point}/${w.key}`, w.value);
      },
      remount: () => remountStore(point, store),
      changed: (b) => changes.push(b),
      lazy: lazy === undefined ? undefined : { [store]: lazy }
    });
    open.push(t);
    const read = (key: string) => fs.promises.readFile(`${point}/${key}`, 'utf8');
    const set = (key: string, value: string) => t.write({ kind: 'set', bucket: store, key, value: text(value) });
    return { t, point, changes, read, set };
  };
  return { lock, tab };
};

afterEach(() => {
  for (const t of open.splice(0)) t.close();
});

describe('one writer across tabs', () => {
  it('the first tab writes; a second tab forwards its writes and reads both', async () => {
    const { tab } = tabs('one-writer');
    const a = await tab('a');
    const b = await tab('b');
    expect(await a.t.start()).toBe(true);
    expect(await b.t.start()).toBe(false);

    await a.set('first', 'from a');
    await b.set('second', 'from b');

    expect(await a.read('first')).toBe('from a');
    expect(await a.read('second')).toBe('from b');
    expect(await b.read('first')).toBe('from a');
    expect(await b.read('second')).toBe('from b');
    expect(a.changes, "the writer's own UI hears of the forwarded write").toEqual(['one-writer']);
  });

  it("a reading tab sees the writer's change without writing", async () => {
    const { tab } = tabs('broadcast');
    const a = await tab('a');
    const b = await tab('b');
    await a.t.start();
    await b.t.start();

    await a.set('late', 'after b mounted');
    await vi.waitFor(() => expect(b.changes).toEqual(['broadcast']));
    expect(await b.read('late')).toBe('after b mounted');
  });

  it('when the writer closes, the next tab takes over and keeps every file', async () => {
    const { tab } = tabs('handover');
    const a = await tab('a');
    const b = await tab('b');
    await a.t.start();
    await b.t.start();
    await a.set('kept', 'from a');
    await b.set('forwarded', 'from b');

    a.t.close();
    await vi.waitFor(() => expect(b.t.isWriter).toBe(true));
    await b.set('after', 'from b, writing');

    const c = await tab('c');
    expect(await c.read('kept')).toBe('from a');
    expect(await c.read('forwarded')).toBe('from b');
    expect(await c.read('after')).toBe('from b, writing');
  });

  it('a tab that takes over rereads what it had not yet remounted before it writes', async () => {
    const { tab } = tabs('takeover-reread', 60_000);
    const a = await tab('a');
    const b = await tab('b');
    await a.t.start();
    await b.t.start();
    await a.set('unseen', 'from a, not yet remounted by b');

    a.t.close();
    await vi.waitFor(() => expect(b.t.isWriter).toBe(true));
    await b.set('after', 'from b, writing');

    const c = await tab('c');
    expect(await c.read('unseen')).toBe('from a, not yet remounted by b');
    expect(await c.read('after')).toBe('from b, writing');
  });

  it('a write nobody acknowledged goes to the next writer', async () => {
    const { lock, tab } = tabs('pending');
    let release!: () => void;
    // a writer that never answers: its tab is closing
    void navigator.locks.request(lock, () => new Promise<void>((r) => (release = r)));
    await vi.waitFor(() => expect(release).toBeDefined());
    const b = await tab('b');
    const c = await tab('c');
    expect(await b.t.start()).toBe(false);
    expect(await c.t.start()).toBe(false);

    let landed = false;
    const write = c.set('waiting', 'from c').then(() => (landed = true));
    await new Promise((r) => setTimeout(r, 50));
    expect(landed).toBe(false);

    release();
    await write;
    expect(b.t.isWriter).toBe(true);
    expect(await b.read('waiting')).toBe('from c');
    expect(await c.read('waiting')).toBe('from c');
  });

  it('a tab that was waiting applies its own pending write when it becomes the writer', async () => {
    const { lock, tab } = tabs('own-pending');
    let release!: () => void;
    void navigator.locks.request(lock, () => new Promise<void>((r) => (release = r)));
    await vi.waitFor(() => expect(release).toBeDefined());
    const b = await tab('b');
    await b.t.start();

    const write = b.set('mine', 'from b');
    release();
    await write;
    expect(b.t.isWriter).toBe(true);
    expect(await b.read('mine')).toBe('from b');
  });
});
