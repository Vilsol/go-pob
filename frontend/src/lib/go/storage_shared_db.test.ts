import 'fake-indexeddb/auto';
import { describe, expect, it, vi } from 'vitest';

vi.unmock('@zenfs/core');
vi.unmock('@zenfs/dom');

type Callbacks = [unknown, (bucket: string, key: string) => Promise<Uint8Array>, (bucket: string, key: string, value?: Uint8Array) => Promise<void>];
const go = vi.hoisted(() => ({ callbacks: undefined as unknown }));
vi.mock('../types', () => ({ storage: { InitializeStorage: (...cbs: unknown[]) => void (go.callbacks = cbs) }, builds: {} }));

const { configure, fs } = await import('@zenfs/core');
const { IndexedDB } = await import('@zenfs/dom');
const { configureStorage } = await import('./storage_config');
const { initializeStorage } = await import('./storage_init');

const text = (b: Uint8Array) => new TextDecoder().decode(b);

// each worker is its own realm with its own zenfs; two mounts of one database in one realm are the same thing
describe('two zenfs instances on one IndexedDB database', () => {
  it('hand out the same inode id, so the second write replaces the first file', async () => {
    const [main, power] = await Promise.all([IndexedDB.create({ storeName: 'shared' }), IndexedDB.create({ storeName: 'shared' })]);
    await configure({ mounts: { '/main': main, '/power': power } });

    await fs.promises.writeFile('/main/written-by-main', 'main');
    await fs.promises.writeFile('/power/written-by-power', 'power');

    expect(await fs.promises.readFile('/main/written-by-main', 'utf8')).toBe('power');
  });

  it("a power worker reads the main worker's cache and writes none of it", async () => {
    await configure({ mounts: { '/main-cache': await IndexedDB.create({ storeName: 'cache' }) } });
    await fs.promises.writeFile('/main-cache/before', 'main, before the power worker');

    vi.spyOn(console, 'log').mockImplementation(() => {});
    await configureStorage();
    await initializeStorage(true);
    const [, get, write] = go.callbacks as Callbacks;
    expect(text(await get('cache', 'before'))).toBe('main, before the power worker');

    await fs.promises.writeFile('/main-cache/after', 'main, after the power worker');
    await write('cache', 'missed', new TextEncoder().encode('power'));

    expect(await fs.promises.readFile('/main-cache/after', 'utf8')).toBe('main, after the power worker');
    expect(fs.existsSync('/cache/missed')).toBe(false);
  });
});
