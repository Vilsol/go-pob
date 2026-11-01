import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { PHONE, bar, openMore, routeOf } from './support/phone';
import { poe2Nav, poe2Page, poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

// two socket groups, the first with two active skills
const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_three_spells.xml');
const code = () => deflateSync(readFileSync(FIXTURE)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');
// PoB-PoE2's Hit DPS with Spark and with Fireball as the main active skill (oracle/poe2_sidebar.lua)
const POB2 = { spark: '24.9', fireball: '32.8' };

const PAGES: [string, string][] = [
  ['Tree', 'tree'],
  ['Skills', 'skills'],
  ['Items', 'items'],
  ['Calcs', 'calcs'],
  ['Config', 'configuration']
];

const open = async (page: Page) => {
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
  return errors;
};

const activeSkill = (page: Page) => page.getByRole('complementary', { name: 'Main skill and stats' }).getByRole('combobox', { name: 'Active skill' });

test('a PoE2 build runs in the app frame: every PoE2 page from the top bar, the main skill from the sidebar, saved and reopened', async ({ page }) => {
  const errors = await open(page);
  const tabs = page.locator('nav a.tab');
  await expect(tabs).toHaveCount(PAGES.length);
  for (const [label, route] of PAGES) {
    await poe2Nav(page, label).click();
    await expect.poll(() => routeOf(page)).toBe(route);
    await expect(poe2Page(page)).toBeVisible();
    await expect(page.getByRole('status').filter({ hasText: 'not available' })).toHaveCount(0);
  }
  // a direct URL to a page PoE2 has no body for
  await page.goto(`/notes?import=${encodeURIComponent(code())}`);
  await expect(page.getByRole('status').filter({ hasText: 'The Notes page is not available for Path of Exile 2 builds yet.' })).toBeVisible({
    timeout: 180_000
  });

  await page.goto(`/calcs?import=${encodeURIComponent(code())}`);
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(POB2.spark, { timeout: 180_000 });
  await activeSkill(page).selectOption({ label: 'Fireball' });
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(POB2.fireball);
  await expect(page.getByRole('button', { name: 'Save (unsaved changes)' })).toBeVisible();

  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog', { name: 'Save build as' });
  await dialog.getByRole('textbox', { name: 'Build name' }).fill('Three spells');
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByRole('button', { name: 'Save (no unsaved changes)' })).toBeVisible();
  await page.getByRole('link', { name: 'Back to builds' }).click();

  await page.getByRole('link', { name: 'Three spells', exact: true }).click();
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(POB2.fireball, { timeout: 60_000 });
  await expect(activeSkill(page)).toHaveValue('2');
  expect(errors).toEqual([]);
});

test.describe('390px wide', () => {
  test.use(PHONE);
  test('a PoE2 build on a phone: the bottom bar reaches every PoE2 page and More lists no PoE1-only page', async ({ page }) => {
    const errors = await open(page);
    await expect(bar(page).getByRole('link')).toHaveText(PAGES.map(([label]) => label));
    for (const [label, route] of PAGES) {
      await bar(page).getByRole('link', { name: label, exact: true }).click();
      await expect.poll(() => routeOf(page)).toBe(route);
      await expect(poe2Page(page)).toBeVisible();
    }
    const more = await openMore(page);
    await expect(more.getByRole('navigation', { name: 'More pages' })).toHaveCount(0);
    await expect(more.getByRole('menuitem', { name: 'Settings' })).toBeVisible();
    expect(errors).toEqual([]);
  });
});
