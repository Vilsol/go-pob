import { test as base, type BrowserContext, type Page } from '@playwright/test';
import { appendFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { RESULTS_DIR } from './perf';

// Specs that import `test` from here share one booted app per worker: App.open on a page that already booted resets it
// through the app (Back to builds, New build) instead of reloading. After each test the page is kept only if the test
// passed and left nothing behind (`leftovers`); otherwise the next test boots a fresh context. Specs that need a cold
// boot (load, offline, service worker, persistence across reload, init scripts, their own viewport, context routes)
// import `@playwright/test` instead. E2E_SHARED=0 (CI) gives every test a fresh context.

type Shared = { context?: BrowserContext; page?: Page; hasTouch?: boolean };
export type Pristine = { storage: string; title: string; workers: number };

const booted = new WeakMap<Page, Pristine>();

export const storageFingerprint = (page: Page): Promise<string> =>
  page.evaluate(async () => {
    const digest = (v: unknown): string => {
      if (v instanceof ArrayBuffer || ArrayBuffer.isView(v)) {
        const b = v instanceof ArrayBuffer ? new Uint8Array(v) : new Uint8Array(v.buffer, v.byteOffset, v.byteLength);
        let h = 0;
        for (const x of b) h = (h * 31 + x) | 0;
        return `bytes:${b.length}:${h}`;
      }
      if (v instanceof Blob) return `blob:${v.size}`;
      return JSON.stringify(v) ?? String(v);
    };
    const local = Object.keys(localStorage)
      .sort()
      .map((k) => `${k}=${localStorage.getItem(k)}`);
    const session = Object.keys(sessionStorage)
      .sort()
      .map((k) => `${k}=${sessionStorage.getItem(k)}`);
    const stores: string[] = [];
    for (const { name } of await indexedDB.databases()) {
      if (!name) continue;
      const db = await new Promise<IDBDatabase>((ok, fail) => {
        const r = indexedDB.open(name);
        r.onsuccess = () => ok(r.result);
        r.onerror = () => fail(new Error(r.error?.message ?? `indexedDB.open(${name}) failed`));
      });
      for (const store of [...db.objectStoreNames].sort()) {
        // the game data cache only grows with what the app has loaded so far; it holds no build or setting
        if (/cache/i.test(`${name}/${store}`)) continue;
        const rows = await new Promise<string[]>((ok, fail) => {
          const out: string[] = [];
          const c = db.transaction(store, 'readonly').objectStore(store).openCursor();
          c.onsuccess = () => {
            const cur = c.result;
            if (!cur) return ok(out);
            out.push(`${digest(cur.key)}=${digest(cur.value)}`);
            cur.continue();
          };
          c.onerror = () => fail(new Error(c.error?.message ?? `reading ${name}/${store} failed`));
        });
        stores.push(`${name}/${store}: ${rows.sort().join(', ')}`);
      }
      db.close();
    }
    return JSON.stringify({ local, session, stores, cookie: document.cookie });
  });

export const sharedBoot = {
  // what App.open saw right after this page's first New build, or undefined on a page that has not booted
  pristine: (page: Page) => booted.get(page),
  booted: async (page: Page) => {
    booted.set(page, { storage: await storageFingerprint(page), title: await page.title(), workers: page.workers().length });
  }
};

const leftovers = async (page: Page, pristine: Pristine | undefined): Promise<string> => {
  if (page.isClosed()) return 'the page was closed';
  if (pristine === undefined) return 'the test never opened the app';
  if (page.workers().length !== pristine.workers) return `${page.workers().length} workers run, ${pristine.workers} after boot`;
  await page.mouse.move(0, 0);
  const open = async () => (await page.getByRole('dialog').count()) + (await page.getByRole('menu').count());
  // a user leaving a test's last dialog or menu presses Escape
  for (let i = 0; i < 3 && (await open()) > 0; i++) {
    await page.keyboard.press('Escape');
    await page.waitForTimeout(100);
  }
  const left = await open();
  if (left > 0) return `${left} dialogs or menus stay open after Escape`;
  if ((await storageFingerprint(page)) !== pristine.storage) return 'it changed browser storage (settings or saved builds)';
  return '';
};

const note = (line: string) => {
  try {
    appendFileSync(resolve(RESULTS_DIR, 'shared-boot.log'), `${line}\n`);
  } catch {
    // the results directory is gone between runs
  }
};

export const test = base.extend<object, { shared: Shared }>({
  shared: [
    // eslint-disable-next-line no-empty-pattern -- Playwright reads a fixture's dependencies from this pattern
    async ({}, use) => {
      const s: Shared = {};
      await use(s);
      await s.context?.close();
    },
    { scope: 'worker' }
  ],
  page: async ({ shared, browser, viewport, baseURL, locale, colorScheme, reducedMotion, acceptDownloads, serviceWorkers, hasTouch }, use, testInfo) => {
    if (shared.page && shared.hasTouch !== hasTouch) {
      // touch support is fixed when a context is created
      await shared.context!.close().catch(() => {});
      shared.context = undefined;
      shared.page = undefined;
    }
    if (!shared.page) {
      // the options Playwright's own context fixture would pass for this config
      shared.context = await browser.newContext({ viewport, baseURL, locale, colorScheme, reducedMotion, acceptDownloads, serviceWorkers, hasTouch });
      shared.page = await shared.context.newPage();
      shared.hasTouch = hasTouch;
    }
    const page = shared.page;
    if (viewport) await page.setViewportSize(viewport);
    await use(page);

    const failed = testInfo.status !== testInfo.expectedStatus;
    const why = failed
      ? 'the test failed'
      : process.env.E2E_SHARED === '0'
        ? 'E2E_SHARED=0'
        : await leftovers(page, booted.get(page)).catch((e: unknown) => `the check threw: ${String(e).slice(0, 200)}`);
    // the test's own routes go; the context's are App.open's data and asset routes, kept for the next test
    await page.unrouteAll({ behavior: 'ignoreErrors' }).catch(() => {});
    await shared.context!.clearPermissions().catch(() => {});
    page.removeAllListeners();
    note(`${testInfo.workerIndex}\t${why ? 'recycle' : 'keep'}\t${testInfo.titlePath.slice(1).join(' › ')}${why ? `\t${why}` : ''}`);
    if (why) {
      await shared.context!.close().catch(() => {});
      shared.context = undefined;
      shared.page = undefined;
    }
  }
});

export { expect } from '@playwright/test';
