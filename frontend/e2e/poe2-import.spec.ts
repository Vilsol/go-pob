// BuildSiteTools.lua:19-35, 88-113, BuildListHelpers.lua:17-51 (PoB-PoE2)

import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync, inflateSync } from 'node:zlib';

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
  await expect(view).toContainText('Sorceress');
  await expect(view.getByRole('spinbutton', { name: 'Character level' })).toHaveValue('40');
  expect(errors).toEqual([]);
});

// ImportTab.lua:135-137, 263 (PoB-PoE2): the Export tab's code is the build as Save writes it, and imports as that build
test("a PoE2 build's exported code imports back as a new build with the same XML", async ({ page }) => {
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
  await page.route('**/api/build?*', (route) => route.fulfill({ status: 200, contentType: 'text/plain; charset=utf-8', body: code() }));
  await page.goto(`/?import=${encodeURIComponent('https://poe.ninja/poe2/pob/Abc1')}`);
  await expect(poe2Stat(page, 'Life')).toHaveText(POB2.Life, { timeout: 180_000 });

  const exportCode = async () => {
    await page.getByRole('link', { name: 'Import / Export' }).click();
    const dialog = page.getByRole('dialog', { name: 'Import / Export' });
    await dialog.getByRole('tab', { name: 'Export' }).click();
    const box = dialog.getByRole('region', { name: 'Build sharing' }).getByRole('textbox', { name: 'Build code' });
    let out = '';
    await expect.poll(async () => (out = await box.inputValue())).not.toBe('');
    return { dialog, out };
  };
  const decode = (c: string) => inflateSync(Buffer.from(c.replaceAll('-', '+').replaceAll('_', '/'), 'base64')).toString('utf8');

  const first = await exportCode();
  const xml = decode(first.out);
  expect(xml).toMatch(/^<\?xml[^>]*\?>\s*<PathOfBuilding2>/);
  expect(xml).toContain(`<PlayerStat stat="Life" value="${POB2.Life}"/>`);
  await expect(first.dialog.getByRole('combobox', { name: 'Share via' }).locator('option')).toHaveText([
    'Maxroll (PoE2)',
    'pobb.in',
    'poe.ninja (PoE2)',
    'poe2db.tw'
  ]);

  await first.dialog.getByRole('tab', { name: 'Import' }).click();
  const region = first.dialog.getByRole('region', { name: 'Build import' });
  await region.getByRole('textbox', { name: /^To import a build/ }).fill(first.out);
  await expect(region.getByRole('textbox', { name: /^To import a build/ })).toHaveAccessibleDescription('Code is valid');
  await region.getByRole('button', { name: 'Import', exact: true }).click();
  const prompt = page.getByRole('dialog', { name: 'Save Changes' });
  await expect.poll(async () => (await prompt.isVisible()) || !(await first.dialog.isVisible())).toBe(true);
  if (await prompt.isVisible()) await prompt.getByRole('button', { name: "Don't Save" }).click();
  await expect(first.dialog).toBeHidden();
  await expect(poe2Stat(page, 'Life')).toHaveText(POB2.Life, { timeout: 60_000 });

  expect(decode((await exportCode()).out)).toBe(xml);
  expect(errors).toEqual([]);
});

// ImportTab.lua:286-294, 312 (PoB-PoE2): a code imported as a comparison opens beside the open build on the Compare page
test('a PoE2 code imported as a comparison opens on the Compare page beside the open build', async ({ page }) => {
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
  await expect(poe2Stat(page, 'Life')).toHaveText(POB2.Life, { timeout: 180_000 });

  await page.getByRole('link', { name: 'Import / Export' }).click();
  const region = page.getByRole('dialog', { name: 'Import / Export' }).getByRole('region', { name: 'Build import' });
  await region.getByRole('textbox', { name: /^To import a build/ }).fill(code());
  await region.getByRole('radio', { name: /Import as a comparison/ }).click();
  await region.getByRole('button', { name: 'Import', exact: true }).click();
  await page.waitForURL((u) => u.pathname.endsWith('/compare'));
  const rows = page.getByRole('region', { name: 'Stat comparison' });
  await expect(rows).toBeVisible({ timeout: 60_000 });
  await expect(rows.getByRole('group', { name: new RegExp(`^Total Life: A ${POB2.Life}, B ${POB2.Life}(,|$)`) })).toHaveCount(1);
  await expect(poe2Stat(page, 'Life')).toHaveText(POB2.Life);
  expect(errors).toEqual([]);
});
