import { type Page } from '@playwright/test';
import { test, expect } from './support/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

// ListControl.lua:158-163, 428-436, ItemListControl.lua:275-288
const BUILD = '0339-reliquarian-shield-crush-of-the-chieftain-2';

const views = (page: Page) => page.getByRole('tablist', { name: 'Item views' });
const names = (page: Page) => page.getByRole('table', { name: 'Stash' }).locator('button[data-pob-color]');

const dragOnto = async (page: Page, from: ReturnType<Page['locator']>, to: () => ReturnType<Page['locator']>, tab?: string) => {
  await from.hover();
  await page.mouse.down();
  if (tab) {
    await views(page).getByRole('tab', { name: tab }).hover();
    await views(page)
      .getByRole('tab', { name: tab })
      .hover({ position: { x: 5, y: 5 } });
  }
  await to().hover({ position: { x: 20, y: 4 } });
  await to().hover({ position: { x: 24, y: 6 } });
  await page.mouse.up();
};

test(`drag reorders the stash and a DB item lands where it is dropped (${BUILD})`, async ({ page }) => {
  test.setTimeout(300_000);
  const xml = buildXml(BUILD);
  const build = await go.loadXml(xml);
  const app = await App.open(page, BUILD);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Items');
  await views(page)
    .getByRole('tab', { name: /^Stash/ })
    .click();
  await expect(names(page).first()).toBeVisible();
  const before = await names(page).allTextContents();
  expect(before.length, 'vacuity: 0339 needs three stash items').toBeGreaterThanOrEqual(3);

  await dragOnto(page, names(page).nth(2), () => names(page).nth(0));
  const moved = [before[2], before[0], before[1], ...before.slice(3)];
  await expect(names(page)).toHaveText(moved);
  await views(page).getByRole('tab', { name: 'Equipped' }).click();
  await views(page)
    .getByRole('tab', { name: /^Stash/ })
    .click();
  await expect(names(page)).toHaveText(moved);

  await views(page).getByRole('tab', { name: 'Browse items' }).click();
  await page.getByRole('complementary', { name: 'Filters' }).getByRole('searchbox', { name: 'Search uniques' }).fill("kaom's heart");
  const source = page.getByRole('table', { name: 'Unique list' }).getByRole('button', { name: "Kaom's Heart", exact: true });
  await dragOnto(page, source, () => names(page).nth(1), 'Stash');
  await expect(names(page)).toHaveText([moved[0], "Kaom's Heart", ...moved.slice(1)]);
  app.assertNoErrors();
});
