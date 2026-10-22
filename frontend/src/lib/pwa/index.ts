import { get, readable, writable, type Readable } from 'svelte/store';
import { base } from '$app/paths';
import { browser, dev, version } from '$app/environment';
import { ART_CACHE, isCdnAsset } from './hosts';
import { DESKTOP } from './desktop';
import { applyDesktopUpdate, checkDesktopUpdate } from './desktop_update';
import { addToast } from '$lib/design/toast';
import { changesSince, parseChangelog, type ChangeVersion } from '$lib/design/help';

const OFF_KEY = 'pob:sw-off';
const CACHE_PREFIX = 'pob-';
const APP_PREFIX = 'pob-app-';
const UPDATE_INTERVAL_MS = 30 * 60_000;

export const online: Readable<boolean> = readable(browser ? navigator.onLine : true, (set) => {
  if (!browser) return;
  const update = () => set(navigator.onLine);
  window.addEventListener('online', update);
  window.addEventListener('offline', update);
  update();
  return () => {
    window.removeEventListener('online', update);
    window.removeEventListener('offline', update);
  };
});

export const updateAvailable = writable(false);

/* Main.lua:1300-1320 */
export const updateChanges = writable<ChangeVersion[]>([]);

const cachedText = async (cache: string, url: string) => (await (await caches.open(cache)).match(url))?.text();

export const readUpdateChanges = async () => {
  const url = `${base}/changelog.txt`;
  const next = (await caches.keys()).find((k) => k.startsWith(APP_PREFIX) && k !== `${APP_PREFIX}${version}`);
  const [running, update] = await Promise.all([cachedText(`${APP_PREFIX}${version}`, url), next ? cachedText(next, url) : undefined]);
  updateChanges.set(running && update ? changesSince(update, parseChangelog(running)[0]?.version ?? '') : []);
};

/* Main.lua:224-229 */
export const updateCheckRunning = writable(false);

export const storagePersistence = writable<'granted' | 'refused' | 'unsupported' | 'unknown'>('unknown');

let registration: ServiceWorkerRegistration | undefined;
let reloading = false;

const storageGet = (key: string) => {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
};

const storageSet = (key: string, value: string | null) => {
  try {
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, value);
  } catch {
    // empty
  }
};

export const clearArtCache = async () => {
  if (typeof caches === 'undefined' || !(await caches.has(ART_CACHE))) return 0;
  const n = (await (await caches.open(ART_CACHE)).keys()).length;
  await caches.delete(ART_CACHE);
  return n;
};

export const disableServiceWorker = async () => {
  storageSet(OFF_KEY, '1');
  const regs = (await navigator.serviceWorker?.getRegistrations()) ?? [];
  await Promise.all(regs.map((r) => r.unregister()));
  for (const key of await caches.keys()) {
    if (key.startsWith(CACHE_PREFIX)) await caches.delete(key);
  }
};

// Main.lua:382-385
let updateAvailableShown = false;

const watch = (reg: ServiceWorkerRegistration) => {
  const check = () => {
    const ready = !!reg.waiting && !!navigator.serviceWorker.controller;
    updateAvailable.set(ready);
    if (ready) readUpdateChanges().catch((e: unknown) => console.warn('the update changelog could not be read', e));
    if (ready && !updateAvailableShown) {
      updateAvailableShown = true;
      addToast('Update Available\nAn update has been downloaded and is ready\nto be applied.');
    }
  };
  check();
  reg.addEventListener('updatefound', () => {
    const installing = reg.installing;
    installing?.addEventListener('statechange', () => {
      if (installing.state === 'installed') check();
    });
  });
};

const wanted = new Map<string, string[]>();

export const keepForOffline = (urls: string[], tag = '') => {
  if (!browser) return;
  const kept = [...new Set(urls.filter((u) => isCdnAsset(new URL(u))))];
  if (tag) wanted.set(tag, [...new Set([...(wanted.get(tag) ?? []), ...kept])]);
  const controller = navigator.serviceWorker?.controller;
  if (controller && kept.length > 0) controller.postMessage({ type: 'KEEP_CDN', urls: kept, tag });
};

const keepCdn = (entries: PerformanceEntry[]) => keepForOffline(entries.map((e) => e.name));

/* Main.lua:374-381 */
export const checkForUpdate = async (reg = registration) => {
  if (get(updateCheckRunning)) return;
  if (!reg && !DESKTOP) {
    addToast('Update check failed!\nThe service worker is off here (?sw=off, or a development server), so no update can be fetched.');
    return;
  }
  updateCheckRunning.set(true);
  try {
    if (!reg) {
      if (await checkDesktopUpdate()) updateAvailable.set(true);
      return;
    }
    await reg.update();
    const installing = reg.installing;
    if (installing) {
      await new Promise<void>((done) => {
        const settled = () => installing.state !== 'installing' && (installing.removeEventListener('statechange', settled), done());
        installing.addEventListener('statechange', settled);
        settled();
      });
    }
    if (!(reg.waiting && navigator.serviceWorker.controller)) addToast('No update available\nYou are running the latest version.');
  } catch (e) {
    addToast(`Update check failed!\n${e instanceof Error ? e.message : String(e)}`);
  } finally {
    updateCheckRunning.set(false);
  }
};

export const applyUpdate = () => {
  if (DESKTOP) {
    void applyDesktopUpdate();
    return;
  }
  const waiting = registration?.waiting;
  if (!waiting) return;
  reloading = true;
  waiting.postMessage({ type: 'SKIP_WAITING' });
};

const requestPersistence = async () => {
  if (!navigator.storage?.persist) {
    storagePersistence.set('unsupported');
    return;
  }
  const granted = (await navigator.storage.persisted()) || (await navigator.storage.persist());
  storagePersistence.set(granted ? 'granted' : 'refused');
  console.info(`persistent storage: ${granted ? 'granted' : 'refused'}`);
};

export const startPwa = async () => {
  if (!browser) return;
  requestPersistence().catch((e: unknown) => console.warn('persistent storage request failed', e));
  if (dev || DESKTOP || !('serviceWorker' in navigator)) return;

  const sw = new URL(location.href).searchParams.get('sw');
  if (sw === 'on') storageSet(OFF_KEY, null);
  if (sw === 'off' || storageGet(OFF_KEY) === '1') {
    await disableServiceWorker();
    console.warn('service worker disabled (?sw=off); open with ?sw=on to enable it again');
    return;
  }

  navigator.serviceWorker.addEventListener('message', (event) => {
    const data = event.data as { type?: string; tag?: string; count?: number; missing?: string[] } | null;
    if (data?.type === 'KEPT_CDN')
      performance.mark('pob:offline-kept', { detail: { tag: data.tag ?? '', count: data.count ?? 0, missing: data.missing ?? [] } });
  });
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    if (!reloading) {
      keepCdn(performance.getEntriesByType('resource'));
      for (const [tag, urls] of wanted) keepForOffline(urls, tag);
      return;
    }
    reloading = false;
    location.reload();
  });
  try {
    new PerformanceObserver((list) => keepCdn(list.getEntries().filter((e) => (e as PerformanceResourceTiming).workerStart === 0))).observe({
      type: 'resource'
    });
  } catch {
    // empty
  }

  registration = await navigator.serviceWorker.register(`${base}/service-worker.js`, { scope: `${base}/`, updateViaCache: 'none' });
  watch(registration);

  const reg = registration;
  const update = () => void reg.update().catch(() => undefined);
  setInterval(update, UPDATE_INTERVAL_MS);
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') update();
  });
};
