import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Nav, poe2Page } from './support/poe2';
import { displayStats2 } from '../src/lib/display/stats2';
import { formatNumSep, luaFormat } from '../src/lib/display/format_stat';

test.use({ serviceWorkers: 'block' });

const FIXTURES = resolve(import.meta.dirname, '../../wasm2/exposition/testdata');
const ORACLE = resolve(import.meta.dirname, '../../testdata/oracle');
const code = (name: string) =>
  deflateSync(readFileSync(resolve(FIXTURES, `${name}.xml`)))
    .toString('base64')
    .replaceAll('+', '-')
    .replaceAll('/', '_');
// PoB-PoE2's own calculation of each build (oracle/poe2_sidebar.lua)
const pob2 = (name: string) => (JSON.parse(readFileSync(resolve(ORACLE, `poe2_sidebar_${name}.json`), 'utf8')) as { output: Record<string, number> }).output;
const fmt = (stat: string, v: number) => {
  const s = displayStats2.find((d) => d.stat === stat)!;
  return formatNumSep(luaFormat(s.fmt ?? 'd', v * (s.pc || s.mod ? 100 : 1)));
};

const open = async (page: Page, name: string) => {
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
  await page.goto(`/?import=${encodeURIComponent(code(name))}`);
  await expect(poe2Page(page)).toBeVisible({ timeout: 180_000 });
  return errors;
};

test("a PoE2 build compared with another: each side shows PoB-PoE2's numbers, and B's item and config reach A", async ({ page }) => {
  const errors = await open(page, 'sorceress_spark');
  await poe2Nav(page, 'Compare').click();
  const choose = page.getByRole('region', { name: 'Choose a build to compare' });
  await choose.getByRole('textbox', { name: 'Comparison build code' }).fill(code('warrior_dual_mace'));
  await choose.getByRole('button', { name: 'Compare' }).click();

  const rows = page.getByRole('region', { name: 'Stat comparison' });
  await expect(rows).toBeVisible({ timeout: 60_000 });
  const [a, b] = [pob2('sorceress_spark'), pob2('warrior_dual_mace')];
  for (const [label, stat] of [
    ['Total Life', 'Life'],
    ['Effective Hit Pool', 'TotalEHP'],
    ['Strength', 'Str']
  ] as const) {
    await expect(rows.getByRole('group', { name: new RegExp(`^${label}: A ${fmt(stat, a[stat])}, B ${fmt(stat, b[stat])}(,|$)`) })).toHaveCount(1);
  }

  await page.getByRole('tab', { name: 'Items' }).click();
  await expect(page.getByRole('button', { name: 'Equip' }).first()).toBeVisible();
  await expect(page.locator('body')).not.toContainText(/\^(x[0-9A-Fa-f]{6}|\d)/, { useInnerText: true });
  await page.getByRole('button', { name: 'Equip' }).first().click();
  await expect(page.getByRole('button', { name: 'Save (unsaved changes)' })).toBeVisible();

  await page.getByRole('tab', { name: 'Config' }).click();
  await page.getByRole('combobox', { name: 'Elemental Resistance penalty: (B)' }).selectOption({ label: 'Act 1 (0%)' });
  const row = page.getByRole('row').filter({ has: page.getByRole('combobox', { name: 'Elemental Resistance penalty: (A)' }) });
  await row.getByRole('button', { name: 'Use B’s value' }).click();
  await expect(page.getByRole('combobox', { name: 'Elemental Resistance penalty: (A)' })).toHaveValue('0');

  expect(errors).toEqual([]);
});

test("the power report and the config impact column carry PoB-PoE2's values, and the Items view lists its jewel rows", async ({ page }) => {
  // PoB-PoE2's compare power report for this pair (oracle/poe2_compare_power.lua)
  const pob = JSON.parse(readFileSync(resolve(ORACLE, 'poe2_compare_power.json'), 'utf8')) as {
    FullDPS: { category: string; name: string; var?: string; impact: number }[];
  };
  const boss = pob.FullDPS.find((r) => r.var === 'enemyIsBoss')!;
  const signed = (v: number) => (v > 0 ? '+' : '−') + Math.round(Math.abs(v)).toLocaleString('en-US');

  const errors = await open(page, 'compare_power_primary');
  await poe2Nav(page, 'Compare').click();
  const choose = page.getByRole('region', { name: 'Choose a build to compare' });
  await choose.getByRole('textbox', { name: 'Comparison build code' }).fill(code('compare_power_compared'));
  await choose.getByRole('button', { name: 'Compare' }).click();
  await expect(page.getByRole('region', { name: 'Stat comparison' })).toBeVisible({ timeout: 60_000 });

  await page.getByRole('tab', { name: 'Power report' }).click();
  const report = page.getByRole('table', { name: 'Power report rows' });
  const bossRow = report.getByRole('row').filter({ hasText: 'Is the enemy a Boss?' });
  await expect(bossRow).toContainText(signed(boss.impact), { timeout: 60_000 });
  await expect(report.getByRole('row').filter({ hasText: 'Take Spell Damage' })).toHaveCount(1);

  await page.getByRole('tab', { name: 'Config' }).click();
  const cfgRow = page.getByRole('row').filter({ has: page.getByRole('combobox', { name: 'Is the enemy a Boss? (A)' }) });
  await expect(cfgRow).toContainText(signed(boss.impact));

  await page.getByRole('tab', { name: 'Items' }).click();
  const items = page.getByRole('region', { name: 'Item differences' });
  await expect(items.getByRole('row').filter({ hasText: 'Sapphire of Thought' })).toContainText('Socket');

  expect(errors).toEqual([]);
});
