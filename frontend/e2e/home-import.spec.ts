import { test, expect, type Page } from '@playwright/test';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { buildXml } from './support/builds';
import * as go from './support/go_model';

const BUILD = '0280-elementalist-fireball';

test.use({ serviceWorkers: 'block' });

const start = async (page: Page) => {
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await page.context().route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });
  await routeTreeAssets(page);
  await page.goto('/');
  await expect(page.getByRole('button', { name: 'Import build', exact: true })).toBeVisible({ timeout: 180_000 });
  return errors;
};

const slowChunks = (page: Page, ms: number) =>
  page.route('**/_app/immutable/**', async (route) => {
    await new Promise((r) => setTimeout(r, ms));
    await route.continue();
  });

// Import is a dialog over the build list, which stays under it
const expectImportPage = async (page: Page) => {
  const dialog = page.getByRole('dialog', { name: 'Import build' });
  await expect(dialog.getByRole('region', { name: 'Build import' })).toBeVisible({ timeout: 60_000 });
  await page.waitForTimeout(2_000);
  await expect(dialog.getByRole('region', { name: 'Build import' })).toBeVisible();
  await expect(page.getByRole('grid', { name: 'Builds', includeHidden: true })).toHaveCount(1);
};

test('Import build on the homepage opens Import, with slow chunks, once the data is in', async ({ page }) => {
  const errors = await start(page);
  await page.waitForFunction(() => performance.getEntriesByName('pob:data-loaded').length > 0, undefined, { timeout: 180_000 });
  await slowChunks(page, 1_500);
  await page.getByRole('button', { name: 'Import build', exact: true }).click();
  await expectImportPage(page);
  expect(errors).toEqual([]);
});

test('Import build on the homepage opens Import, with slow chunks, while the data loads', { tag: '@smoke' }, async ({ page }) => {
  const errors = await start(page);
  await slowChunks(page, 1_500);
  await page.getByRole('button', { name: 'Import build', exact: true }).click();
  await expectImportPage(page);
  expect(errors).toEqual([]);
});

test('Import build clicked twice on the homepage opens Import', async ({ page }) => {
  const errors = await start(page);
  await page.waitForFunction(() => performance.getEntriesByName('pob:data-loaded').length > 0, undefined, { timeout: 180_000 });
  await slowChunks(page, 1_500);
  const button = page.getByRole('button', { name: 'Import build', exact: true });
  await button.click();
  await button.click({ timeout: 1_000 }).catch(() => {});
  await expectImportPage(page);
  expect(errors).toEqual([]);
});

test('Import build on the build list reached from an open build opens Import', async ({ page }) => {
  const errors = await start(page);
  await page.waitForFunction(() => performance.getEntriesByName('pob:data-loaded').length > 0, undefined, { timeout: 180_000 });
  await page.getByRole('button', { name: 'New build', exact: true }).click();
  await expect(page).toHaveURL(/\/tree$/);
  await page.getByRole('link', { name: 'Back to builds' }).click();
  await slowChunks(page, 1_500);
  await page.getByRole('button', { name: 'Import build', exact: true }).click();
  await expectImportPage(page);
  expect(errors).toEqual([]);
});

test('/import opens the import dialog over the build list, and importing there opens the build on its Tree', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await page.context().route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });
  await routeTreeAssets(page);
  await page.goto('/import');
  const dialog = page.getByRole('dialog', { name: 'Import build' });
  await expect(dialog).toBeVisible({ timeout: 180_000 });
  await expect(page).toHaveURL(/\/tree$/);
  await page.waitForFunction(() => performance.getEntriesByName('pob:data-loaded').length > 0, undefined, { timeout: 180_000 });
  const region = dialog.getByRole('region', { name: 'Build import' });
  const box = region.getByRole('textbox', { name: /^To import a build/ });
  await box.fill(await go.encode(buildXml(BUILD)));
  await expect(box).toHaveAccessibleDescription('Code is valid');
  await region.getByRole('button', { name: 'Import', exact: true }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByRole('main', { name: 'Tree' })).toBeVisible({ timeout: 60_000 });
  await expect(page.getByRole('link', { name: 'Back to builds' })).toBeVisible();
  expect(errors).toEqual([]);
});
