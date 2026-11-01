import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { PHONE, bar } from './support/phone';
import { poe2Nav, poe2Page } from './support/poe2';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
const code = (path: string) => deflateSync(readFileSync(path)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

const open = async (page: Page, fixture = FIXTURE) => {
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
  await page.goto(`/?import=${encodeURIComponent(code(fixture))}`);
  const view = poe2Page(page);
  await expect(view).toBeVisible({ timeout: 180_000 });
  return { view: page, errors };
};

// PoB-PoE2's Calcs tab for the fixture: testdata/oracle/poe2_calcs_sorceress_spark.json (Life:0:0:0, Life:0:2:0)
test('the PoE2 Calcs page shows PoB-PoE2 values', async ({ page }) => {
  const { view, errors } = await open(page);
  await poe2Nav(page, 'Calcs').click();
  const life = view.getByRole('region', { name: 'Calculations' }).getByRole('region', { name: 'Life', exact: true });
  await expect(life.locator('[data-key]').filter({ hasText: 'Base from Gear' })).toContainText('90');
  await expect(life.locator('[data-key]').filter({ hasText: 'Total Base' })).toContainText('650');
  expect(errors).toEqual([]);
});

const ITEMS_FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_items.xml');

// PoB-PoE2's Skills and Items tabs for the fixture: testdata/oracle/poe2_pages_sorceress_items.json
test('the PoE2 Skills and Items pages show PoB-PoE2 groups and items, and the main group recalculates', async ({ page }) => {
  const { view, errors } = await open(page, ITEMS_FIXTURE);
  await poe2Nav(page, 'Skills').click();
  const groups = view.getByRole('listbox', { name: 'Socket groups' }).getByRole('option');
  await expect(groups).toHaveText(['Spark (Active) (FullDPS)', 'Backup (Disabled)']);
  await expect(view.locator('[data-gem="Rapid Casting I"]')).toContainText('No');
  await groups.nth(1).click();
  await view.getByRole('button', { name: 'Set as main skill' }).click();
  await expect(groups.nth(1)).toHaveText('Backup (Disabled) (Forced Active)');
  // the frame sidebar's picker counts groups from 0
  await expect(page.getByRole('complementary', { name: 'Main skill and stats' }).getByRole('combobox', { name: 'Main skill' })).toHaveValue('1');

  await poe2Nav(page, 'Items').click();
  await expect(view.getByLabel('Weapon set')).toHaveText('Weapon Set 2');
  await expect(view.locator('[data-slot]')).toHaveCount(4);
  await expect(view.locator('[data-slot="Weapon 1 Swap"]')).toContainText('Glyph Pillar');
  await expect(view.locator('[data-slot="Weapon 1"]')).toHaveCount(0);
  await expect(view.locator('[data-slot="Body Armour"]')).toContainText('Bonded: +20 to maximum Life');
  await expect(view.locator('[data-slot="Ring 1"]')).toContainText('Corrupted');
  await expect(view.locator('[data-slot="Jewel 61419"]')).toContainText('12% increased Spell Damage');
  expect(errors).toEqual([]);
});

test.describe('390px wide', () => {
  test.use(PHONE);
  test('the PoE2 Skills, Items and Calcs pages fit a phone', async ({ page }) => {
    const { view, errors } = await open(page, ITEMS_FIXTURE);
    for (const [tab, pane] of [
      ['Skills', view.locator('[data-poe2-page][aria-label="Skills"]')],
      ['Items', view.locator('[data-poe2-page][aria-label="Items"]')],
      ['Calcs', view.getByRole('region', { name: 'Calculations' })]
    ] as const) {
      await bar(page).getByRole('link', { name: tab, exact: true }).click();
      await expect(pane).toBeVisible();
      // the pane sits inside the phone, takes its width below the stats, and its content does not run past it (a Calcs
      // damage table scrolls inside itself)
      const fit = await pane.evaluate((el) => ({
        right: Math.round(el.getBoundingClientRect().right),
        width: el.clientWidth,
        overflow: el.scrollWidth - el.clientWidth
      }));
      const [inner, scroll] = await page.evaluate(() => [window.innerWidth, (document.scrollingElement ?? document.documentElement).scrollWidth]);
      expect([tab, inner, scroll, fit.right <= PHONE.viewport.width, fit.width >= 300, fit.overflow <= 1]).toEqual([
        tab,
        PHONE.viewport.width,
        PHONE.viewport.width,
        true,
        true,
        true
      ]);
    }
    expect(errors).toEqual([]);
  });
});
