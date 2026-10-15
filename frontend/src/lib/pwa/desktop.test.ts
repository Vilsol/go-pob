// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';

vi.mock('$app/environment', () => ({ browser: true, dev: false, version: 'test' }));
vi.mock('$app/paths', () => ({ base: '' }));

const register = vi.fn(() => Promise.resolve({ addEventListener: () => {}, update: () => Promise.resolve() } as unknown as ServiceWorkerRegistration));
Object.defineProperty(navigator, 'serviceWorker', {
  configurable: true,
  value: { register, addEventListener: () => {}, getRegistrations: () => Promise.resolve([]), controller: null }
});

const load = async (desktop: boolean) => {
  vi.resetModules();
  vi.stubEnv('VITE_DESKTOP', desktop ? '1' : '');
  return { pwa: await import('./index'), toast: await import('$lib/design/toast') };
};

const stubFetch = (answers: Record<string, unknown>) => {
  const fetch = vi.fn<(url: string, init?: RequestInit) => Promise<Response>>((url) => {
    const a = answers[url];
    const [status, body] = Array.isArray(a) ? (a as [number, unknown]) : [a === null ? 202 : 200, a];
    return Promise.resolve(new Response(body === null ? null : JSON.stringify(body), { status }));
  });
  vi.stubGlobal('fetch', fetch);
  return fetch;
};

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
  register.mockClear();
});

describe('desktop mode', () => {
  it('registers no service worker', async () => {
    const { pwa } = await load(true);
    await pwa.startPwa();
    expect(register).not.toHaveBeenCalled();
  });

  it('says when the build has no update source', async () => {
    const { pwa, toast } = await load(true);
    const fetch = stubFetch({ '/desktop/update/check': { enabled: false, install: false } });
    await pwa.checkForUpdate(undefined);
    expect(get(toast.toasts).map((t) => t.message)).toEqual(['Update check failed!\nThis build has no update source; updates come with the next release.']);
    expect(fetch.mock.calls.map((c) => c[0])).toEqual(['/desktop/update/check']);
  });

  it('names a release it cannot install in place', async () => {
    const { pwa, toast } = await load(true);
    stubFetch({ '/desktop/update/check': { enabled: true, current: '1.0.0', version: '1.1.0', page: 'https://example.invalid/v1.1.0', install: false } });
    await pwa.checkForUpdate(undefined);
    expect(get(toast.toasts).map((t) => t.message)).toEqual(['Update Available\ngo-pob 1.1.0 is out (running 1.0.0).\nhttps://example.invalid/v1.1.0']);
    expect(get(pwa.updateAvailable)).toBe(false);
  });

  it('downloads an update it can install, and restarts into it on apply', async () => {
    const { pwa, toast } = await load(true);
    const fetch = stubFetch({
      '/desktop/update/check': { enabled: true, current: '1.0.0', version: '1.1.0', install: true },
      '/desktop/update/install': { ready: true },
      '/desktop/update/restart': null
    });
    await pwa.checkForUpdate(undefined);
    expect(get(pwa.updateAvailable)).toBe(true);
    expect(get(toast.toasts).map((t) => t.message)).toEqual(['Update Available\nAn update has been downloaded and is ready\nto be applied.']);
    pwa.applyUpdate();
    await vi.waitFor(() => expect(fetch).toHaveBeenCalledTimes(3));
    expect(fetch.mock.calls.map((c) => [c[0], c[1]?.method ?? 'GET'])).toEqual([
      ['/desktop/update/check', 'GET'],
      ['/desktop/update/install', 'POST'],
      ['/desktop/update/restart', 'POST']
    ]);
  });

  it('says why a download was refused', async () => {
    const { pwa, toast } = await load(true);
    stubFetch({
      '/desktop/update/check': { enabled: true, version: '1.1.0', install: true },
      '/desktop/update/install': [502, { error: 'updater: ed25519ph signature did not verify' }]
    });
    await pwa.checkForUpdate(undefined);
    expect(get(pwa.updateAvailable)).toBe(false);
    expect(get(toast.toasts).map((t) => t.message)).toEqual(['Update failed!\nupdater: ed25519ph signature did not verify']);
  });

  it('the web build still registers it', async () => {
    const { pwa } = await load(false);
    await pwa.startPwa();
    expect(register).toHaveBeenCalledTimes(1);
  });
});
