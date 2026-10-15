import { describe, expect, it, vi } from 'vitest';

type Callbacks = [
  (bucket: string, dir: string) => Promise<{ Name: string; Type: string; Class: string; Level: number }[]>,
  (bucket: string, key: string) => Promise<Uint8Array>,
  (bucket: string, key: string, value?: Uint8Array) => Promise<void>,
  (bucket: string, key: string) => Promise<boolean>,
  (bucket: string, dir: string) => Promise<void>,
  (bucket: string, path: string) => Promise<void>
];
const go = vi.hoisted(() => ({ callbacks: undefined as unknown }));
vi.mock('../types', () => ({
  storage: { InitializeStorage: (...cbs: unknown[]) => void (go.callbacks = cbs) },
  builds: {
    SavedBuildSummary: (xml: string) =>
      xml.includes('Witch') ? { ok: true, value: { ClassName: 'Witch', Level: 90, Ascendancy: 'Elementalist', MainSkill: 'Arc' } } : { ok: false }
  }
}));

vi.stubEnv('VITE_DESKTOP', '1');
const { initializeStorage, storageFs } = await import('./storage_init');

const calls: string[] = [];
const files: Record<string, string> = {
  '/desktop/fs/builds/Leagues': JSON.stringify([
    { name: 'My%build', dir: false, lastEdit: '2026-10-15T00:00:00Z' },
    { name: 'Old', dir: true, lastEdit: '2026-10-14T00:00:00Z' }
  ]),
  '/desktop/fs/builds/Leagues/My%25build': '<Build className="Witch"/>'
};
vi.stubGlobal(
  'fetch',
  vi.fn((url: string, init?: RequestInit) => {
    calls.push(`${init?.method ?? 'GET'} ${url}`);
    const body = files[url];
    if ((init?.method ?? 'GET') === 'GET') return Promise.resolve(new Response(body ?? 'missing', { status: body ? 200 : 404 }));
    return Promise.resolve(new Response(null, { status: url.endsWith('nope') ? 404 : 204 }));
  })
);

describe('desktop storage', () => {
  it('lists a folder with each build summarised, and reads, writes, checks, makes and deletes by path', async () => {
    await initializeStorage();
    const [list, get, write, exists, mkdir, del] = go.callbacks as Callbacks;

    const rows = await list('builds', '/Leagues/');
    expect(rows.map((r) => [r.Name, r.Type, r.Class, r.Level])).toEqual([
      ['My%build', 'file', 'Witch', 90],
      ['Old', 'dir', '', 0]
    ]);
    expect(new TextDecoder().decode(await get('builds', 'Leagues/My%build'))).toBe('<Build className="Witch"/>');
    await expect(get('builds', 'gone')).rejects.toThrow(/read builds\/gone: 404/);

    calls.length = 0;
    await write('builds', '/Leagues/New build', new TextEncoder().encode('<x/>'));
    expect(await exists('cache', 'aGVsbG8')).toBe(true);
    expect(await exists('cache', 'nope')).toBe(false);
    await mkdir('builds', '/Leagues/Sub');
    await del('builds', '/Leagues/New build');
    expect(calls).toEqual([
      'PUT /desktop/fs/builds/Leagues/New%20build',
      'HEAD /desktop/fs/cache/aGVsbG8',
      'HEAD /desktop/fs/cache/nope',
      'POST /desktop/fs/builds/Leagues/Sub',
      'DELETE /desktop/fs/builds/Leagues/New%20build'
    ]);
  });

  it("answers the sync worker's direct reads (Export all builds, Clear cached data) from the same folders", async () => {
    expect(await storageFs.readdir('/builds/Leagues')).toEqual(['My%build', 'Old']);
    expect((await storageFs.stat('/builds/Leagues/Old')).isDirectory()).toBe(true);
    expect((await storageFs.stat('/builds/Leagues/My%build')).isDirectory()).toBe(false);
    expect(new TextDecoder().decode(await storageFs.readFile('/builds/Leagues/My%build'))).toContain('Witch');
    await storageFs.rm('/cache/nope', { force: true });
    await expect(storageFs.rm('/cache/nope')).rejects.toThrow(/404/);
  });
});
