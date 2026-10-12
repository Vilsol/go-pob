import 'fake-indexeddb/auto';
import { describe, expect, it, vi } from 'vitest';
import { configureStorage } from './storage_config';

vi.unmock('@zenfs/core');
vi.unmock('@zenfs/dom');

describe('configureStorage', () => {
  it('mounts cache, builds and settings without waiting on the IndexedDB probe', async () => {
    const t0 = performance.now();
    const done = await Promise.race([configureStorage().then(() => true), new Promise((r) => setTimeout(() => r(false), 3000))]);
    expect(done, `configure still pending after ${Math.round(performance.now() - t0)} ms`).toBe(true);
    const { fs } = await import('@zenfs/core');
    for (const dir of ['/cache', '/builds', '/settings']) expect(fs.existsSync(dir)).toBe(true);
  });
});
