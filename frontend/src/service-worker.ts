/// <reference types="@sveltejs/kit" />
/// <reference lib="webworker" />

import { base, build, files, version } from '$service-worker';
import { WASM_FALLBACK, WASM_FILE } from '$lib/go/wasm_file';
import { isCdnAsset } from '$lib/pwa/hosts';

const sw = self as unknown as ServiceWorkerGlobalScope;

const APP_PREFIX = 'pob-app-';
const APP = `${APP_PREFIX}${version}`;
const CDN = 'pob-cdn-v1';

const SHELL = `${base}/`;
const WASM = `${base}${WASM_FILE}`;
const PRECACHE = [SHELL, WASM, ...build, ...files];
const PRECACHED = new Set(PRECACHE);
const FALLBACK = `${base}${WASM_FALLBACK}`;
const RUNTIME = new Set([FALLBACK]);

const usableWasm = async (res: Response | undefined) => {
  const reader = res?.body?.getReader();
  if (!reader) return false;
  const { value } = await reader.read();
  await reader.cancel();
  if (!value || value.length < 4) return false;
  return (value[0] === 0x00 && value[1] === 0x61 && value[2] === 0x73 && value[3] === 0x6d) || (value[0] === 0x1f && value[1] === 0x8b);
};

const FONT_CSS = [
  'https://fonts.googleapis.com/css2?family=Roboto+Flex:opsz@8..144&display=swap',
  'https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500&family=IBM+Plex+Sans:wght@400;500;600&display=swap'
];

const cacheFonts = async () => {
  const cache = await caches.open(CDN);
  for (const href of FONT_CSS) {
    const res = await fetch(href, { mode: 'cors' });
    if (!res.ok) continue;
    const css = await res.clone().text();
    await cache.put(href, res);
    const fonts = [...css.matchAll(/url\((https:\/\/fonts\.gstatic\.com\/[^)]+)\)/g)].map((m) => m[1]);
    await Promise.all(fonts.map(async (f) => (await cache.match(f)) ?? cache.add(new Request(f, { mode: 'cors' }))));
  }
};

const IMMUTABLE = `${base}/_app/immutable/`;
const workerChunks = async (cache: Cache) => {
  const found = new Set<string>();
  for (const url of build) {
    if (!url.endsWith('.js')) continue;
    const text = (await (await cache.match(url))?.text()) ?? '';
    for (const m of text.matchAll(/workers\/([\w.-]+\.js)/g)) found.add(`${IMMUTABLE}workers/${m[1]}`);
  }
  return [...found];
};

sw.addEventListener('install', (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(APP);
      await cache.addAll(PRECACHE.map((url) => new Request(url, { cache: 'reload' })));
      await cache.addAll((await workerChunks(cache)).map((url) => new Request(url, { cache: 'reload' })));
      if (WASM !== FALLBACK && !(await usableWasm(await cache.match(WASM)))) await cache.add(new Request(FALLBACK, { cache: 'reload' }));
      await cacheFonts().catch((e: unknown) => console.warn('service worker: fonts not cached', e));
    })()
  );
});

sw.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      for (const key of await caches.keys()) {
        if (key.startsWith(APP_PREFIX) && key !== APP) await caches.delete(key);
      }
      await sw.clients.claim();
    })()
  );
});

const keepCdn = async (urls: string[]): Promise<string[]> => {
  if (!(await caches.has(APP))) return urls;
  const cache = await caches.open(CDN);
  const missing: string[] = [];
  for (const url of urls) {
    if (!isCdnAsset(new URL(url)) || (await cache.match(url))) continue;
    const res = await fetch(url, { mode: 'cors' }).catch(() => undefined);
    if (res?.ok) await cache.put(url, res);
    else missing.push(url);
  }
  return missing;
};

sw.addEventListener('message', (event) => {
  const data = event.data as { type?: string; urls?: string[]; tag?: string } | null;
  if (data?.type === 'SKIP_WAITING') void sw.skipWaiting();
  if (data?.type === 'KEEP_CDN' && Array.isArray(data.urls)) {
    const source = event.source;
    const { urls, tag } = data;
    event.waitUntil(keepCdn(urls).then((missing) => source?.postMessage({ type: 'KEPT_CDN', tag, count: urls.length, missing })));
  }
});

const cacheFirst = async (cacheName: string, request: Request, key: Request | string = request) => {
  const cache = (await caches.has(APP)) ? await caches.open(cacheName) : undefined;
  const hit = await cache?.match(key);
  if (hit) return hit;
  const res = await fetch(request);
  if (cache && (res.ok || res.type === 'opaque')) await cache.put(key, res.clone());
  return res;
};

sw.addEventListener('fetch', (event) => {
  const { request } = event;
  if (request.method !== 'GET') return;
  const url = new URL(request.url);

  if (url.origin === sw.location.origin) {
    if (request.mode === 'navigate' && url.searchParams.get('sw') === 'off') {
      event.respondWith(fetch(request));
      return;
    }
    if (PRECACHED.has(url.pathname) || RUNTIME.has(url.pathname) || url.pathname.startsWith(IMMUTABLE)) {
      event.respondWith(cacheFirst(APP, request, url.pathname));
      return;
    }
    if (request.mode === 'navigate' && url.pathname.startsWith(SHELL)) {
      event.respondWith(
        (async () => {
          const shell = (await caches.has(APP)) ? await (await caches.open(APP)).match(SHELL) : undefined;
          return shell ?? fetch(request);
        })()
      );
    }
    return;
  }

  if (isCdnAsset(url)) {
    event.respondWith(cacheFirst(CDN, request));
  }
});
