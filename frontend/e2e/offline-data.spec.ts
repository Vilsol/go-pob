import { test, expect, type Locator, type Page } from '@playwright/test';

import { DATA_HOST, cachedAsset } from './support/data_cache';

test.use({ serviceWorkers: 'block' });

const alert = (page: Page) => page.getByRole('alert').filter({ hasText: 'The game data could not be loaded' });

// the floating activity indicator and the notice share the bottom right corner: neither may cover the other
const expectApart = async (page: Page, notice: Locator) => {
  const activity = page.getByRole('button', { name: /^Activity/ });
  await expect(activity).toBeVisible();
  const a = (await notice.boundingBox())!;
  const b = (await activity.boundingBox())!;
  const apart = a.x + a.width <= b.x || b.x + b.width <= a.x || a.y + a.height <= b.y || b.y + b.height <= a.y;
  expect(apart, `the notice ${JSON.stringify(a)} and the activity indicator ${JSON.stringify(b)} overlap`).toBe(true);
};

test('game data the browser cannot fetch and has not kept shows the reason, with a retry', async ({ page, context }) => {
  const refused: string[] = [];
  await context.route(`${DATA_HOST}**`, (route) => {
    refused.push(route.request().url());
    return route.abort('internetdisconnected');
  });
  await page.goto('/');

  await expect(alert(page)).toBeVisible({ timeout: 60_000 });
  await expect(alert(page)).toContainText('failed to fetch url');
  expect(await page.evaluate(() => performance.getEntriesByName('pob:data-loaded').length), 'data-loaded was marked for data that did not load').toBe(0);
  expect(refused.length, 'the test did not reach the data host, so it refused nothing').toBeGreaterThan(0);
  await expectApart(page, alert(page));

  const reloaded = page.waitForEvent('framenavigated');
  await alert(page).getByRole('button', { name: 'Retry' }).click();
  await reloaded;
});

test('PoE2 game data the browser cannot fetch and has not kept shows the reason, with a retry', async ({ page, context }) => {
  await context.addInitScript(() => {
    localStorage.setItem('options:poe2', 'true');
    localStorage.setItem('lastGame', 'poe2');
  });
  const refused: string[] = [];
  await context.route(`${DATA_HOST}**`, (route) => {
    const url = route.request().url();
    if (/\/data\/0\.\d+\//.test(url)) {
      refused.push(url);
      return route.abort('internetdisconnected');
    }
    const asset = cachedAsset(url);
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });
  await page.goto('/');

  const poe2Alert = page.getByRole('alert').filter({ hasText: 'The PoE2 game data could not be loaded' });
  await expect(poe2Alert).toBeVisible({ timeout: 120_000 });
  await expect(poe2Alert).toContainText('tree2: fetching');
  await expectApart(page, poe2Alert);
  expect(refused.length, 'the test did not reach the PoE2 data, so it refused nothing').toBeGreaterThan(0);
  await expect(alert(page), 'the PoE1 game data was served, so it has no notice').toHaveCount(0);

  const reloaded = page.waitForEvent('framenavigated');
  await poe2Alert.getByRole('button', { name: 'Retry' }).click();
  await reloaded;
});

test('a wasm runtime the browser cannot fetch says so, with a retry', async ({ page }) => {
  const refused: string[] = [];
  await page.route(/\/go-pob\.wasm/, (route) => {
    refused.push(route.request().url());
    return route.abort('internetdisconnected');
  });
  await page.goto('/');

  const status = page.getByRole('status').filter({ hasText: 'Failed to load wasm runtime' });
  await expect(status).toBeVisible({ timeout: 60_000 });
  expect(refused.length, 'the test did not reach the wasm, so it refused nothing').toBeGreaterThan(0);

  const reloaded = page.waitForEvent('framenavigated');
  await status.getByRole('button', { name: 'Retry' }).click();
  await reloaded;
});
