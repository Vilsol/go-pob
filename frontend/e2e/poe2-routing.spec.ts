import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
// PoB-PoE2's values for the fixture, as wasm2/exposition's TestFixtureBuildMatchesPoB asserts them
const POB2 = { Life: '683', Mana: '231' };

const code = () => deflateSync(readFileSync(FIXTURE)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

// game_route.ts POE2_DISABLED
const POE2_DISABLED = 'This is a Path of Exile 2 build. Path of Exile 2 builds are not supported yet.';

const SECOND_WORKER = /go-pob2\.wasm|sync_worker2/;

const start = async (page: Page, poe2: boolean) => {
  const errors: string[] = [];
  const requests: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  page.on('request', (r) => requests.push(r.url()));
  if (poe2) await page.addInitScript(() => localStorage.setItem('options:poe2', 'true'));
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
  const view = page.getByRole('region', { name: 'Path of Exile 2 build' });
  await expect(view).toBeVisible({ timeout: 180_000 });
  await expect(view).toContainText('Level 40 Sorceress');
  await expect(view.locator('[data-stat="Life"]')).toHaveText(POB2.Life);
  await expect(view.locator('[data-stat="Mana"]')).toHaveText(POB2.Mana);
  expect(requests.filter((u) => /go-pob2\.wasm/.test(u))).toHaveLength(1);
  expect(errors).toEqual([]);
});
