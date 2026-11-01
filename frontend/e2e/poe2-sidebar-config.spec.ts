import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Nav, poe2Page, poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
const code = () => deflateSync(readFileSync(FIXTURE)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

// PoB-PoE2's sidebar for the fixture at the default penalty and at Act 1, from oracle/dump_build.lua
test('with the PoE2 flag on, editing a config option recalculates the PoE2 sidebar', async ({ page }) => {
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
  await page.goto(`/?import=${encodeURIComponent(code())}`);

  await expect(poe2Page(page)).toBeVisible({ timeout: 180_000 });
  await expect(poe2Stat(page, 'FireResist')).toHaveText('-15%');
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText('24.9');

  await poe2Nav(page, 'Config').click();
  const penalty = page.locator('[data-config="resistancePenalty"]');
  await expect(penalty).toHaveValue('6');
  await penalty.selectOption({ label: 'Act 1 (0%)' });
  await expect(poe2Stat(page, 'FireResist')).toHaveText('45%');
  await expect(poe2Stat(page, 'LightningResist')).toHaveText('10%');

  await penalty.selectOption({ label: 'Endgame (-60%)' });
  await expect(poe2Stat(page, 'FireResist')).toHaveText('-15%');
  expect(errors).toEqual([]);
});
