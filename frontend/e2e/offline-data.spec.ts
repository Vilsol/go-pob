import { test, expect } from '@playwright/test';

import { DATA_HOST } from './support/data_cache';

test.use({ serviceWorkers: 'block' });

const alert = (page: import('@playwright/test').Page) => page.getByRole('alert').filter({ hasText: 'The game data could not be loaded' });

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

  const reloaded = page.waitForEvent('framenavigated');
  await alert(page).getByRole('button', { name: 'Retry' }).click();
  await reloaded;
});
