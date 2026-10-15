import { expect, it, vi } from 'vitest';

const zen = vi.hoisted(() => ({ create: vi.fn(() => Promise.resolve({})), configure: vi.fn(() => Promise.resolve()) }));
vi.mock('@zenfs/dom', () => ({ IndexedDB: { create: zen.create } }));
vi.mock('@zenfs/core', () => ({ configure: zen.configure }));

it('opens no IndexedDB store on the desktop, the three on the web', async () => {
  vi.stubEnv('VITE_DESKTOP', '1');
  await (await import('./storage_config')).configureStorage();
  expect(zen.create).not.toHaveBeenCalled();
  expect(zen.configure).not.toHaveBeenCalled();

  vi.resetModules();
  vi.stubEnv('VITE_DESKTOP', '');
  await (await import('./storage_config')).configureStorage();
  expect(zen.create.mock.calls.map((c) => (c as unknown[])[0])).toEqual([{ storeName: 'cache' }, { storeName: 'builds' }, { storeName: 'settings' }]);
});
