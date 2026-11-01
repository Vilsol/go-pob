import { test, expect, type BrowserContext, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Nav, poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
const code = () => deflateSync(readFileSync(FIXTURE)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

// The longest the main thread may go without running a 50 ms interval, past its due time. On this suite's machines a
// warm import's worst stall is the page's own boot, about a second, and opening the Tree page about two under load;
// a freeze of the kind the PoE2 tree data caused in a dev build (tens of seconds) is far past it. Deep $state on the
// tree data costs about a second more in a production build, which this bound does not catch.
const MAX_STALL_MS = 10_000;

type Stall = { max: number; sum: number; reset: () => void };

const stallProbe = () => {
  const probe = { max: 0, sum: 0, reset: () => ((probe.max = 0), (probe.sum = 0)) };
  let last = performance.now();
  setInterval(() => {
    const now = performance.now();
    const late = now - last - 50;
    probe.max = Math.max(probe.max, late);
    probe.sum += Math.max(0, late);
    last = now;
  }, 50);
  (globalThis as unknown as { __stall: typeof probe }).__stall = probe;
};

const maxStall = (page: Page) => page.evaluate(() => (globalThis as unknown as { __stall: Stall }).__stall.max);
const sumStall = (page: Page) => page.evaluate(() => (globalThis as unknown as { __stall: Stall }).__stall.sum);
const resetStall = (page: Page) => page.evaluate(() => (globalThis as unknown as { __stall: Stall }).__stall.reset());

const open = async (context: BrowserContext) => {
  const page = await context.newPage();
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await routeTreeAssets(page);
  const started = Date.now();
  await page.goto(`/?import=${encodeURIComponent(code())}`);
  await expect(poe2Stat(page, 'Life')).toHaveText('683', { timeout: 180_000 });
  return { page, errors, ms: Date.now() - started };
};

test('a warm PoE2 import and opening its passive tree do not stall the main thread', { tag: '@slow' }, async ({ context }) => {
  await context.addInitScript(() => localStorage.setItem('options:poe2', 'true'));
  await context.addInitScript(stallProbe);
  await context.route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });

  const cold = await open(context);
  const coldStall = await maxStall(cold.page);
  // the cache2 database's writes settle before the warm tab reads it
  await cold.page.waitForTimeout(3_000);
  await cold.page.close();
  expect(cold.errors).toEqual([]);

  const warm = await open(context);
  const warmStall = await maxStall(warm.page);
  await resetStall(warm.page);
  const clicked = Date.now();
  await poe2Nav(warm.page, 'Tree').click();
  await expect(warm.page.getByRole('img', { name: 'Passive skill tree' })).toBeVisible({ timeout: 60_000 });
  await warm.page.waitForTimeout(1_000);
  const treeMs = Date.now() - clicked;
  const treeStall = await maxStall(warm.page);
  const treeBlocked = await sumStall(warm.page);
  console.log(
    `PoE2 import: cold ${cold.ms} ms (max stall ${coldStall.toFixed(0)} ms), warm ${warm.ms} ms (max stall ${warmStall.toFixed(0)} ms); ` +
      `Tree page ${treeMs} ms (max stall ${treeStall.toFixed(0)} ms, ${treeBlocked.toFixed(0)} ms late in all)`
  );
  expect(warmStall).toBeLessThan(MAX_STALL_MS);
  expect(treeStall).toBeLessThan(MAX_STALL_MS);
  expect(warm.errors).toEqual([]);
});
