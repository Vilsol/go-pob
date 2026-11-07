import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Page, poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
// PoB-PoE2's values for the fixture, as wasm2/exposition's TestFixtureBuildMatchesPoB asserts them
const POB2 = { Life: '683', Mana: '231' };

const code = () => deflateSync(readFileSync(FIXTURE)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

// game_route.ts POE2_DISABLED
const POE2_DISABLED = 'This is a Path of Exile 2 build. Path of Exile 2 builds are not supported yet.';

const SECOND_WORKER = /go-pob2\.wasm|sync_worker2/;
// go-pob-data's PoE2 game version and tree version
const POE2_DATA = /\/0[._]5\//;

const start = async (page: Page, poe2: boolean) => {
  const errors: string[] = [];
  const requests: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  page.on('request', (r) => requests.push(r.url()));
  await page.addInitScript((on) => localStorage.setItem('options:poe2', String(on)), poe2);
  await page.context().route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });
  await routeTreeAssets(page);
  await page.goto(`/?import=${encodeURIComponent(code())}`);
  return { errors, requests };
};

test('with the PoE2 flag off a PoE2 build is refused and the second worker never loads', { tag: '@smoke' }, async ({ page }) => {
  const { errors, requests } = await start(page, false);
  await expect(page.getByRole('alert')).toContainText(POE2_DISABLED, { timeout: 180_000 });
  await page.getByRole('button', { name: 'OK' }).click();
  await expect(page.getByRole('button', { name: 'Import build', exact: true })).toBeVisible();
  expect(requests.filter((u) => SECOND_WORKER.test(u))).toEqual([]);
  expect(requests.some((u) => /go-pob\.wasm/.test(u))).toBe(true);
  expect(errors).toEqual([]);
});

test('with the PoE2 flag on a PoE2 build opens in the second worker with PoB-PoE2 values', async ({ page }) => {
  const { errors, requests } = await start(page, true);
  const view = poe2Page(page);
  await expect(view).toBeVisible({ timeout: 180_000 });
  await expect(view).toContainText('Sorceress');
  await expect(view.getByRole('spinbutton', { name: 'Character level' })).toHaveValue('40');
  await expect(poe2Stat(page, 'Life')).toHaveText(POB2.Life);
  await expect(poe2Stat(page, 'Mana')).toHaveText(POB2.Mana);
  expect(requests.filter((u) => /go-pob2\.wasm/.test(u))).toHaveLength(1);
  expect(errors).toEqual([]);
});

test('a second session opens a PoE2 build from the storage cache, with no game data fetched', async ({ page }) => {
  const first = await start(page, true);
  await expect(poe2Stat(page, 'Life')).toHaveText(POB2.Life, { timeout: 180_000 });
  const poe2Data = (urls: string[]) => urls.filter((u) => u.startsWith(DATA_HOST) && POE2_DATA.test(u));
  expect(poe2Data(first.requests), 'the first session fetched no PoE2 data, so the second proves nothing').not.toEqual([]);

  first.requests.length = 0;
  await page.goto(`/?import=${encodeURIComponent(code())}`);
  await expect(poe2Stat(page, 'Life')).toHaveText(POB2.Life, { timeout: 180_000 });
  await expect(poe2Stat(page, 'Mana')).toHaveText(POB2.Mana);
  expect(
    first.requests.filter((u) => /go-pob2\.wasm/.test(u)),
    'the second session did not boot the PoE2 worker'
  ).toHaveLength(1);
  expect(poe2Data(first.requests)).toEqual([]);
  expect(first.errors).toEqual([]);
});

test.describe('with the service worker on', () => {
  test.use({ serviceWorkers: 'allow' });

  const cachedUrls = (page: Page) =>
    page.evaluate(async () => {
      await navigator.serviceWorker.ready;
      const urls: string[] = [];
      for (const key of await caches.keys()) for (const req of await (await caches.open(key)).keys()) urls.push(req.url);
      return urls;
    });

  test('with the PoE2 flag off a cold load neither fetches nor precaches any PoE2 bytes', { tag: '@smoke' }, async ({ page, context }) => {
    const all: string[] = [];
    context.on('request', (r) => all.push(r.url()));
    const { errors } = await start(page, false);
    await expect(page.getByRole('alert')).toContainText(POE2_DISABLED, { timeout: 180_000 });
    const cached = await cachedUrls(page);
    expect(
      cached.some((u) => /\/go-pob\.wasm/.test(u)),
      'the PoE1 wasm is not in the cache, so the probe sees nothing'
    ).toBe(true);
    expect(cached.filter((u) => SECOND_WORKER.test(u))).toEqual([]);
    expect(all.filter((u) => SECOND_WORKER.test(u))).toEqual([]);
    expect(errors).toEqual([]);
  });

  test('with the PoE2 flag on the PoE2 wasm is cached the first time a PoE2 build asks for it', async ({ page }) => {
    const { errors } = await start(page, true);
    await expect(poe2Stat(page, 'Life')).toHaveText(POB2.Life, { timeout: 180_000 });
    await expect.poll(async () => (await cachedUrls(page)).filter((u) => /\/go-pob2\.wasm/.test(u)), { timeout: 60_000 }).not.toEqual([]);
    expect(errors).toEqual([]);
  });
});
