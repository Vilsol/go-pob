// @vitest-environment jsdom
// Main.lua:374-386, Launch.lua:158-161

import { afterEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';

vi.mock('$app/environment', () => ({ browser: true, dev: false }));
vi.mock('$app/paths', () => ({ base: '' }));

const { checkForUpdate, updateCheckRunning } = await import('./index');
const { toasts, clearToasts } = await import('$lib/design/toast');

afterEach(() => clearToasts());

const reg = (o: Partial<ServiceWorkerRegistration>) =>
  ({ update: () => Promise.resolve(), installing: null, waiting: null, ...o }) as unknown as ServiceWorkerRegistration;
const messages = () => get(toasts).map((t) => t.message.split('\n')[0]);

describe('checkForUpdate', () => {
  it('says there is none when the host serves the same worker', async () => {
    await checkForUpdate(reg({}));
    expect(messages()).toEqual(['No update available']);
    expect(get(updateCheckRunning)).toBe(false);
  });

  it('says the check failed, with the reason', async () => {
    await checkForUpdate(reg({ update: () => Promise.reject(new Error('offline')) }));
    expect(get(toasts).map((t) => t.message)).toEqual(['Update check failed!\noffline']);
  });

  it('says the check failed with no service worker', async () => {
    await checkForUpdate(undefined);
    expect(messages()).toEqual(['Update check failed!']);
  });
});
