import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Nav, poe2Page, poe2Stat } from './support/poe2';
import { breakdownStep } from '../src/lib/design/breakdown_lines';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
// PoB-PoE2's breakdown panels over the fixture's main calculation (oracle/poe2_breakdowns.lua)
const ORACLE = resolve(import.meta.dirname, '../../testdata/oracle/poe2_breakdowns_sorceress_spark.json');
const code = (path: string) => deflateSync(readFileSync(path)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

// PoB-PoE2's mod tables over the fixture (oracle/poe2_mod_sections.lua)
const MODS = resolve(import.meta.dirname, '../../testdata/oracle/poe2_mod_sections_sorceress_spark.json');
const modValues = (key: string) =>
  ((JSON.parse(readFileSync(MODS, 'utf8')) as Record<string, { rows?: { value: string }[] }>)[key].rows ?? []).map((r) => r.value);

type Section = { type: string; lines?: string[] };
const textOf = (name: string) => (JSON.parse(readFileSync(ORACLE, 'utf8')) as Record<string, Section[]>)[name].find((s) => s.type === 'TEXT')!.lines!;

const plain = (s: string) => s.replace(/\^x[0-9A-Fa-f]{6}|\^\d/g, '').replace(/\s+/g, '');
const shownAs = (line: string) => {
  const st = breakdownStep(line);
  return 'prose' in st ? plain(st.prose) : plain((st.op ?? '') + st.value + (st.note ?? ''));
};

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
  await page.goto(`/?import=${encodeURIComponent(code(FIXTURE))}`);
  await expect(poe2Page(page)).toBeVisible({ timeout: 180_000 });
  return errors;
};

// CalcSections.lua (PoB-PoE2): the Life section's Total row names breakdown "Life", the Fire resistance row "FireResist"
test("the PoE2 Calcs page shows PoB-PoE2's breakdown for a hovered row", async ({ page }) => {
  const errors = await open(page);
  await poe2Nav(page, 'Calcs').click();
  const calcs = page.getByRole('region', { name: 'Calculations' });
  for (const { section, row, breakdown } of [
    { section: 'Life', row: 'Total', breakdown: 'Life' },
    { section: 'Resists', row: 'Fire Resist', breakdown: 'FireResist' }
  ]) {
    const want = textOf(breakdown);
    await calcs
      .getByRole('region', { name: section, exact: true })
      .locator('[data-key]')
      .filter({ has: page.getByRole('button', { name: row, exact: true }) })
      .first()
      .hover();
    const group = page.getByRole('complementary', { name: 'Breakdown' }).getByRole('group', { name: 'Calculation' }).first();
    await expect(group).toBeVisible();
    const shown = (await group.locator('[data-step], .col-span-3').allTextContents()).map(plain);
    expect(shown, `${section} · ${row}`).toEqual(want.map(shownAs));
    await page.mouse.move(0, 0);
  }
  expect(errors).toEqual([]);
});

// CalcSections.lua:1602 and Build.lua:2278-2331 (PoB-PoE2): Life's Total Base row lists the Life BASE mods, and the
// sidebar's Total Life line shows the Life breakdown with that table
test("the PoE2 Calcs page and sidebar show PoB-PoE2's mod table for Life", async ({ page }) => {
  const errors = await open(page);
  const want = modValues('Life|BASE|||false|');
  expect(want.length, 'PoB-PoE2 lists the fixture Life BASE mods').toBeGreaterThan(1);

  await poe2Stat(page, 'Life').first().hover();
  const tip = page.getByRole('tooltip');
  await expect(tip.getByText('Total Base', { exact: true })).toBeVisible();
  await expect(tip.getByText('Base from Gear', { exact: true })).toHaveCount(0);
  await expect(tip.locator('[data-mod-row]')).not.toHaveCount(0);
  await page.mouse.move(0, 0);

  await poe2Nav(page, 'Calcs').click();
  await page
    .getByRole('region', { name: 'Calculations' })
    .getByRole('region', { name: 'Life', exact: true })
    .locator('[data-key]')
    .filter({ has: page.getByRole('button', { name: 'Total Base', exact: true }) })
    .first()
    .hover();
  const table = page.getByRole('complementary', { name: 'Breakdown' }).getByRole('table', { name: 'Modifiers' });
  await expect(table.getByRole('row')).toHaveCount(want.length + 1);
  const shown = await table.getByRole('row').evaluateAll((rows) => rows.slice(1).map((r) => r.querySelector('[role=cell]')?.textContent ?? ''));
  expect(shown.map(plain)).toEqual(want.map(plain));
  expect(errors).toEqual([]);
});
