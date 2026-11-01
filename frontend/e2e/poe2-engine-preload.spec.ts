import { test, expect, type BrowserContext, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
const code = () => deflateSync(readFileSync(FIXTURE)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

// from Import to the PoE2 page, warm, once the preloaded game data is in: at most 1.6 s with the PoE2 engine booted at
// startup and at least 4.2 s when the import has to boot it, on a machine at load 25-55
const MAX_IMPORT_MS = 3_000;

const workerProbe = () => {
  const log = console.log;
  console.log = (...args: unknown[]) => {
    const g = globalThis as unknown as { __poe2Worker?: number };
    if (args[0] === 'Creating PoE2 sync worker') g.__poe2Worker ??= performance.now();
    log(...args);
  };
};

const prepared = new WeakSet<BrowserContext>();

const prepare = async (context: BrowserContext, lastGame: string, poe2: boolean, both: boolean) => {
  prepared.add(context);
  if (poe2) await context.addInitScript(() => localStorage.setItem('options:poe2', 'true'));
  if (both) await context.addInitScript(() => localStorage.setItem('options:engineBoot', 'both'));
  await context.addInitScript((g) => localStorage.setItem('lastGame', g), lastGame);
  await context.addInitScript(workerProbe);
  await context.route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });
};

const start = async (context: BrowserContext, page: Page, lastGame: string, poe2 = true, both = false) => {
  if (!prepared.has(context)) await prepare(context, lastGame, poe2, both);
  await routeTreeAssets(page);
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await page.goto('/');
  await expect(page.getByRole('button', { name: 'Import build', exact: true })).toBeVisible({ timeout: 180_000 });
  await page.waitForFunction(() => performance.getEntriesByName('pob:data-loaded').length > 0, undefined, { timeout: 180_000 });
  return errors;
};

const poe2Worker = (page: Page) => page.evaluate(() => (globalThis as unknown as { __poe2Worker?: number }).__poe2Worker);

// the build list's Import, with the PoE2 code pasted; the time from Import to the PoE2 page's stats
const importPoe2 = async (page: Page) => {
  await page.getByRole('button', { name: 'Import build', exact: true }).click();
  const region = page.getByRole('dialog', { name: 'Import build' }).getByRole('region', { name: 'Build import' });
  const box = region.getByRole('textbox', { name: /^To import a build/ });
  await box.fill(code());
  await expect(box).toHaveAccessibleDescription('Code is valid');
  const started = Date.now();
  await region.getByRole('button', { name: 'Import', exact: true }).click();
  await expect(poe2Stat(page, 'Life')).toHaveText('683', { timeout: 180_000 });
  return Date.now() - started;
};

test('with a PoE2 build used last, the PoE2 engine boots at startup and a PoE2 import opens at once', async ({ context, page }) => {
  // the first visit fills the game data caches, so the second boots warm
  const coldErrors = await start(context, page, 'poe2');
  await importPoe2(page);
  await page.waitForTimeout(3_000);
  await page.close();
  expect(coldErrors).toEqual([]);

  const warm = await context.newPage();
  const preloaded = warm.waitForEvent('console', { predicate: (m) => m.text().includes('PoE2 game data preloaded'), timeout: 180_000 });
  const errors = await start(context, warm, 'poe2');
  expect(await poe2Worker(warm), 'no PoE2 worker at startup').toBeDefined();
  await preloaded;
  const ms = await importPoe2(warm);
  console.log(`PoE2 import with the PoE2 engine booted at startup: ${ms} ms`);
  expect(ms).toBeLessThan(MAX_IMPORT_MS);
  expect(errors).toEqual([]);
});

test('with a PoE1 build used last, the PoE2 engine waits for a PoE2 build', async ({ context, page }) => {
  const errors = await start(context, page, 'poe1');
  await page.waitForTimeout(1_000);
  expect(await poe2Worker(page), 'a PoE2 worker before any PoE2 build').toBeUndefined();
  console.log(`PoE2 import that boots the PoE2 engine: ${await importPoe2(page)} ms`);
  expect(await poe2Worker(page)).toBeDefined();
  expect(await page.evaluate(() => localStorage.getItem('lastGame'))).toBe('poe2');
  expect(errors).toEqual([]);
});

test('with both engines asked for, the PoE2 engine boots at startup after a PoE1 build', async ({ context, page }) => {
  const errors = await start(context, page, 'poe1', true, true);
  await expect.poll(() => poe2Worker(page), { message: 'no PoE2 worker at startup' }).toBeDefined();
  expect(errors).toEqual([]);
});

test('with PoE2 locked, neither a PoE2 build used last nor both engines asked for boots a PoE2 engine', async ({ context, page }) => {
  const errors = await start(context, page, 'poe2', false, true);
  await page.waitForTimeout(1_000);
  expect(await poe2Worker(page)).toBeUndefined();
  expect(errors).toEqual([]);
});
