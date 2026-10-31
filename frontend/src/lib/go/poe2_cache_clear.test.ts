import 'fake-indexeddb/auto';
import { describe, expect, it, vi } from 'vitest';

vi.unmock('@zenfs/core');
vi.unmock('@zenfs/dom');

const remote = vi.hoisted(() => ({ boot: vi.fn(() => Promise.resolve()), ClearDataCache: vi.fn(() => Promise.resolve(4)) }));
vi.mock('comlink', () => ({ wrap: () => remote, transfer: (d: unknown) => d, expose: () => {} }));
vi.mock('./sync_worker2?worker', () => ({ default: class {} }));
vi.mock('$lib/app_log', async (orig) => ({ ...(await orig<object>()), forwardWorkerLog: () => {} }));
vi.mock('../types2', () => ({
  boot: () => Promise.resolve(),
  config: { InitLogging: () => {} },
  storage: { InitializeStorage: () => {} },
  exposition: {}
}));

const { fs } = await import('@zenfs/core');

const databases = async () => (await indexedDB.databases()).map((d) => d.name);

describe('Clear cached data on the PoE2 cache', () => {
  it('deletes the PoE2 database whole when no PoE2 worker is running', async () => {
    await new Promise<void>((done) => {
      const req = indexedDB.open('cache2');
      req.onupgradeneeded = () => req.result.createObjectStore('cache2').put(new Uint8Array([1]), '2');
      req.onsuccess = () => {
        req.result.close();
        done();
      };
    });
    expect(await databases()).toContain('cache2');

    const { clearPoe2DataCache } = await import('./worker2');
    expect(await clearPoe2DataCache()).toBe(0);
    expect(await databases()).not.toContain('cache2');
    expect(remote.ClearDataCache).not.toHaveBeenCalled();
  });

  it('asks a running PoE2 worker to clear through its own mount', async () => {
    vi.stubGlobal('fetch', () => Promise.resolve(new Response(new Uint8Array([0x00, 0x61, 0x73, 0x6d]))));
    const { clearPoe2DataCache, poe2Worker } = await import('./worker2');
    vi.spyOn(console, 'log').mockImplementation(() => {});
    await poe2Worker();
    expect(await clearPoe2DataCache()).toBe(4);
  });

  it('the PoE2 worker removes every file of its cache mount and counts them', async () => {
    const { worker2Instance } = await import('./sync_worker2');
    await worker2Instance.boot(new ArrayBuffer(0));
    for (const k of ['a', 'b', 'c']) await fs.promises.writeFile(`/cache/${k}`, k);

    expect(await worker2Instance.ClearDataCache()).toBe(3);
    expect(await fs.promises.readdir('/cache')).toEqual([]);
  });
});
