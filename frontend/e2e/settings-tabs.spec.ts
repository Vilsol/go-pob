// Main.lua:1056-1079, 1080-1082

import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

test('a setting changed in one tab reaches another open tab without a reload (0280)', async ({ context }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const code = await go.encode(xml);
  const out = await go.outputs(build);
  const value = out.Output?.AverageHit ?? 0;
  expect(value, 'the fixture shows Average Hit over 10,000').toBeGreaterThan(10_000);
  const compact = value >= 1e9 ? `${(value / 1e9).toFixed(1)}B` : value >= 1e6 ? `${(value / 1e6).toFixed(1)}M` : `${(value / 1e3).toFixed(1)}K`;

  const a = await context.newPage();
  const b = await context.newPage();
  const appA = await App.open(a, 'settings: changing tab');
  const appB = await App.open(b, 'settings: other tab');
  await appA.importCode(code, async () => expectedLines(out));
  await appB.importCode(code, async () => expectedLines(out));
  const rowB = appB.statsRegion().getByRole('button').filter({ hasText: 'Average Hit' }).first();
  await expect(rowB).toContainText(',');
  await expect(b).toHaveTitle('Unnamed build (Elementalist) - go-pob');

  await a.getByRole('button', { name: 'Settings' }).click();
  const sheet = a.getByRole('dialog', { name: 'Settings' });
  await sheet.getByRole('combobox', { name: 'Theme' }).selectOption('Light');
  await sheet.getByRole('checkbox', { name: 'Show build name in window title' }).uncheck();
  await sheet.getByRole('tab', { name: /^Numbers/ }).click();
  await sheet.getByRole('checkbox', { name: 'Compact large numbers (e.g. 12.3K)' }).check();
  await expect(a.locator('html')).toHaveAttribute('data-theme', 'light');

  await expect(b.locator('html')).toHaveAttribute('data-theme', 'light');
  await expect(b).toHaveTitle('go-pob');
  await expect(rowB).toContainText(compact);

  await b.getByRole('button', { name: 'Settings' }).click();
  const sheetB = b.getByRole('dialog', { name: 'Settings' });
  await expect(sheetB.getByRole('combobox', { name: 'Theme' })).toHaveValue('Light');
  await expect(sheetB.getByRole('checkbox', { name: 'Show build name in window title' })).not.toBeChecked();
  appA.assertNoErrors();
  appB.assertNoErrors();
});
