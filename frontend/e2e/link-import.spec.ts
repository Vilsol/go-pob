// Main.lua:56-85

import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { diffLines, expectedLines } from './support/stats';

test('a ?import= link opens the build on its Tree as Go does (0280)', { tag: '@smoke' }, async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);

  await page.goto(`/?import=${encodeURIComponent(await go.encode(xml))}`);
  await expect(page).toHaveURL(/\/tree$/, { timeout: 60_000 });
  await expect(page).toHaveTitle(/^Imported build \(Elementalist\)/);
  // Build.lua:89-94
  await expect(page.getByRole('button', { name: 'Save (unsaved changes)' })).toBeVisible();
  const want = expectedLines(await go.outputs(build));
  expect(want.length).toBeGreaterThan(0);
  await expect.poll(async () => diffLines(want, await app.readStats()), { timeout: 60_000 }).toEqual([]);
  app.assertNoErrors();
});
