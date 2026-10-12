// Main.lua:1056-1079, Common.lua:741-766, Build.lua:1658-1667

import { readFileSync } from 'node:fs';

import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

test('number options rewrite the sidebar and are kept; the window title names the build (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));

  const out = await go.outputs(build);
  const value = out.Output?.AverageHit ?? 0;
  expect(value, 'the fixture shows Average Hit over 10,000').toBeGreaterThan(10_000);
  const row = app.statsRegion().getByRole('button').filter({ hasText: 'Average Hit' }).first();
  await expect(row).toContainText(',');

  await page.getByRole('button', { name: 'Settings' }).click();
  const sheet = page.getByRole('dialog', { name: 'Settings' });
  await sheet.getByRole('tab', { name: /^Numbers/ }).click();
  await sheet.getByRole('checkbox', { name: 'Compact large numbers (e.g. 12.3K)' }).check();
  const compact = value >= 1e9 ? `${(value / 1e9).toFixed(1)}B` : value >= 1e6 ? `${(value / 1e6).toFixed(1)}M` : `${(value / 1e3).toFixed(1)}K`;
  await expect(row).toContainText(compact);
  await sheet.getByRole('combobox', { name: 'Decimal separator' }).selectOption(',');
  await expect(row).toContainText(compact.replace('.', ','));

  // Main.lua:1750-1756
  await expect(page).toHaveTitle('Unnamed build (Elementalist) - go-pob');
  await sheet.getByRole('tab', { name: /^General/ }).click();
  await sheet.getByRole('checkbox', { name: 'Show build name in window title' }).uncheck();
  await expect(page).toHaveTitle('go-pob');
  await sheet.getByRole('checkbox', { name: 'Show build name in window title' }).check();

  await page.reload();
  await page.getByRole('button', { name: 'Settings' }).click();
  await page
    .getByRole('dialog', { name: 'Settings' })
    .getByRole('tab', { name: /^Numbers/ })
    .click();
  await expect(page.getByRole('dialog', { name: 'Settings' }).getByRole('checkbox', { name: 'Compact large numbers (e.g. 12.3K)' })).toBeChecked();
  await expect(page.getByRole('dialog', { name: 'Settings' }).getByRole('combobox', { name: 'Decimal separator' })).toHaveValue(',');
  app.assertNoErrors();
});

test('appearance options are kept; Export all builds saves a .zip; Clear cached data asks first (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  const name = 'e2e-zip-export';
  await page.getByRole('button', { name: /^Save \(/ }).click();
  const saveAs = page.getByRole('dialog');
  await saveAs.getByRole('textbox', { name: 'Build name' }).fill(name);
  await saveAs.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(saveAs).toBeHidden();

  const html = page.locator('html');
  await page.getByRole('button', { name: 'Settings' }).click();
  const sheet = page.getByRole('dialog', { name: 'Settings' });
  await sheet.getByRole('combobox', { name: 'Theme' }).selectOption('Light');
  await sheet.getByRole('slider', { name: 'Font scaling' }).fill('15');
  await sheet.getByRole('checkbox', { name: 'Show animations' }).uncheck();
  await expect(html).toHaveAttribute('data-theme', 'light');
  await expect(html).toHaveAttribute('data-motion', 'off');
  await expect(html).toHaveCSS('font-size', '20px');
  await expect(page.locator('body')).toHaveCSS('background-color', 'rgb(236, 234, 228)');

  await sheet.getByRole('tab', { name: /^Data and storage/ }).click();
  await expect(sheet.locator('[data-setting="dataVersion"]')).toContainText(/\d+\.\d+ · go-pob-data v\d/);
  await expect(sheet.locator('[data-setting="cache"]')).toContainText(/\d+\.\d MB/);
  const download = page.waitForEvent('download');
  await sheet.getByRole('button', { name: 'Export all builds' }).click();
  const zip = await download;
  expect(zip.suggestedFilename()).toMatch(/^go-pob-builds-\d{4}-\d{2}-\d{2}\.zip$/);
  const bytes = readFileSync(await zip.path());
  expect([...bytes.subarray(0, 4)]).toEqual([0x50, 0x4b, 0x03, 0x04]);
  expect(bytes.toString('latin1')).toContain(`${name}.xml`);
  expect(bytes.toString('latin1'), 'the saved build itself').toContain('<PathOfBuilding');
  await sheet.getByRole('button', { name: 'Save' }).click();

  await page.reload();
  expect(await page.evaluate(() => [document.documentElement.dataset.theme, document.documentElement.style.fontSize])).toEqual(['light', '20px']);
  await expect(html).toHaveAttribute('data-motion', 'off');
  await app.savedBuild(name).waitFor({ timeout: 180_000 });

  await page.getByRole('button', { name: 'Settings', exact: true }).click();
  await sheet.getByRole('tab', { name: /^Data and storage/ }).click();
  await sheet.getByRole('button', { name: 'Clear cached data' }).click();
  await expect(sheet.getByRole('alert')).toHaveText('Clear the cached game data? Builds are kept.');
  await sheet.getByRole('button', { name: 'Keep' }).click();
  await expect(sheet.getByRole('alert')).toHaveCount(0);
  await sheet.getByRole('button', { name: 'Clear cached data' }).click();
  await sheet.getByRole('button', { name: 'Clear cached data' }).click();
  await expect(page.getByText(/^Cached data cleared/)).toBeVisible();
  await sheet.getByRole('button', { name: 'Reset all settings' }).click();
  await sheet.getByRole('button', { name: 'Reset all settings' }).click();
  await expect(html).toHaveAttribute('data-theme', 'dark');
  await expect(html).toHaveCSS('font-size', '16px');
  await expect(html).not.toHaveAttribute('data-motion', 'off');
  await sheet.getByRole('button', { name: 'Save' }).click();
  await expect(app.savedBuild(name)).toBeVisible();
  app.assertNoErrors();
});
