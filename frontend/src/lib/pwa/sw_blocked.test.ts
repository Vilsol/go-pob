// @vitest-environment jsdom

import { describe, expect, it, vi } from 'vitest';

vi.mock('$app/environment', () => ({ browser: true, dev: false, version: 'test' }));
vi.mock('$app/paths', () => ({ base: '' }));

const { startPwa, updateAvailable } = await import('./index');
const { get } = await import('svelte/store');

describe('startPwa', () => {
  // a browser that blocks service workers (a policy, or Playwright's serviceWorkers: 'block') resolves register with
  // no registration
  it('starts without a registration when the browser blocks service workers', async () => {
    const register = vi.fn(() => Promise.resolve(undefined));
    Object.defineProperty(navigator, 'serviceWorker', {
      configurable: true,
      value: { register, addEventListener: () => {}, controller: null }
    });
    await expect(startPwa()).resolves.toBeUndefined();
    expect(register).toHaveBeenCalledOnce();
    expect(get(updateAvailable)).toBe(false);
  });
});
