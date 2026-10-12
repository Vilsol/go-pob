import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

test('the item DBs are parsed before Browse items first opens', async ({ page }) => {
  test.setTimeout(300_000);
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Items');
  await app.idleJobDone();
  expect(await page.workers()[0].evaluate(() => performance.getEntriesByName('pob:item-db-warmed').length)).toBe(1);

  const t0 = Date.now();
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Browse items' }).click();
  const list = page.getByRole('table', { name: 'Unique list' });
  await expect.poll(async () => list.locator('[role=row]').count(), { timeout: 60_000 }).toBeGreaterThan(1000);
  console.log(`browse open: ${Date.now() - t0} ms`);
});
