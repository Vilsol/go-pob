// BuildSiteTools.lua:19-35, 88-113, BuildListHelpers.lua:17-51 (PoB-PoE2)

import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Page, poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
// PoB-PoE2's values for the fixture, as wasm2/exposition's TestFixtureBuildMatchesPoB asserts them
const POB2 = { Life: '683' };

const code = () => deflateSync(readFileSync(FIXTURE)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

test('a poe.ninja PoE2 link imports through /api/build, saves to the build list with a PoE2 badge and reopens', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await page.addInitScript(() => localStorage.setItem('options:poe2', 'true'));
  await page.context().route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });
  await routeTreeAssets(page);
  const asked: string[] = [];
  await page.route('**/api/build?*', (route) => {
    asked.push(new URL(route.request().url()).search);
    return route.fulfill({ status: 200, contentType: 'text/plain; charset=utf-8', body: code() });
  });

  await page.goto(`/?import=${encodeURIComponent('https://poe.ninja/poe2/pob/Abc1')}`);
  const view = poe2Page(page);
  await expect(poe2Stat(page, 'Life')).toHaveText(POB2.Life, { timeout: 180_000 });
  expect(asked).toEqual(['?site=PoeNinja2&id=Abc1']);

  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog', { name: 'Save build as' });
  await dialog.getByRole('textbox', { name: 'Build name' }).fill('Spark sorc');
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
  await page.getByRole('link', { name: 'Back to builds' }).click();

  const link = page.getByRole('link', { name: 'Spark sorc', exact: true });
  const row = page.getByRole('row').filter({ has: link });
  await expect(row.locator('[data-game="poe2"]')).toHaveText('PoE2');
  await expect(row.locator('[data-col="level"]')).toHaveText('40');
  await expect(row.locator('[data-col="skill"]')).toHaveText('Spark');
  await expect(row.locator('[data-col="cls"]')).toHaveText('Sorceress');

  await link.click();
  await expect(poe2Stat(page, 'Life')).toHaveText(POB2.Life, { timeout: 60_000 });
  await expect(view).toContainText('Level 40 Sorceress');
  expect(errors).toEqual([]);
});
