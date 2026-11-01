// @vitest-environment jsdom

import { describe, expect, it, vi } from 'vitest';

vi.mock('$app/environment', () => ({ browser: true, dev: false, building: false, version: '' }));
vi.mock('$lib/go/worker', () => ({ syncWrap: undefined, syncWorker: undefined }));
vi.mock('$lib/go/worker2', () => ({ poe2Worker: vi.fn() }));

const { currentBuild, poe2Build } = await import('$lib/global');
const { LAST_GAME_KEY, lastGame } = await import('./engine');

const VIEW = { className: 'Sorceress', ascendClassName: 'None', level: 1, output: {} };

describe('the last-used game', () => {
  it('follows the build last opened, and outlives closing it', () => {
    expect(lastGame()).toBeUndefined();
    poe2Build.set(VIEW);
    expect(localStorage.getItem(LAST_GAME_KEY)).toBe('poe2');
    expect(lastGame()).toBe('poe2');
    poe2Build.set(undefined);
    expect(lastGame()).toBe('poe2');
    currentBuild.set({} as never);
    expect(lastGame()).toBe('poe1');
    currentBuild.set(undefined);
    expect(lastGame()).toBe('poe1');
  });

  it('reads nothing it did not write', () => {
    localStorage.setItem(LAST_GAME_KEY, 'poe3');
    expect(lastGame()).toBeUndefined();
  });
});
