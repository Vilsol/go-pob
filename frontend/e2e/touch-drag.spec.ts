import { test, expect, type Page, type Locator, type CDPSession } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

// Drags by touch, through the browser's real touch input path: HTML5 drag and drop never starts from touch on
// Firefox for Android or Samsung Internet, so these must work on pointer events alone.
const BUILD = '0339-reliquarian-shield-crush-of-the-chieftain-2';

test.use({ hasTouch: true });

const views = (page: Page) => page.getByRole('tablist', { name: 'Item views' });
const names = (page: Page) => page.getByRole('table', { name: 'Stash' }).locator('button[data-pob-color]');

const centre = async (l: Locator) => {
  const b = (await l.boundingBox())!;
  return { x: b.x + b.width / 2, y: b.y + b.height / 2 };
};
const touch = (cdp: CDPSession, type: 'touchStart' | 'touchMove' | 'touchEnd', p?: { x: number; y: number }) =>
  cdp.send('Input.dispatchTouchEvent', { type, touchPoints: p ? [{ x: p.x, y: p.y }] : [] });

const touchDrag = async (page: Page, cdp: CDPSession, from: Locator, to: Locator[], hold = 450) => {
  const a = await centre(from);
  await touch(cdp, 'touchStart', a);
  await page.waitForTimeout(hold);
  let at = a;
  for (const l of to) {
    const b = await centre(l);
    for (let k = 1; k <= 6; k++) await touch(cdp, 'touchMove', { x: at.x + ((b.x - at.x) * k) / 6, y: at.y + ((b.y - at.y) * k) / 6 });
    at = b;
    await page.waitForTimeout(100);
  }
  await touch(cdp, 'touchEnd');
};

test(`touch: a held drag reorders the stash and drops a DB item into it; a quick swipe moves nothing (${BUILD})`, async ({ page }) => {
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
  const cdp = await page.context().newCDPSession(page);

  await touchDrag(page, cdp, names(page).nth(2), [names(page).nth(0)], 50);
  await page.waitForTimeout(300);
  await expect(names(page), 'a swipe without the hold is a scroll, not a drag').toHaveText(before);
  await expect(page.locator('[data-drag-label]')).toHaveCount(0);

  await touchDrag(page, cdp, names(page).nth(2), [names(page).nth(0)]);
  const moved = [before[2], before[0], before[1], ...before.slice(3)];
  await expect(names(page)).toHaveText(moved);

  await views(page).getByRole('tab', { name: 'Browse items' }).click();
  await page.getByRole('complementary', { name: 'Filters' }).getByRole('searchbox', { name: 'Search uniques' }).fill("kaom's heart");
  const source = page.getByRole('table', { name: 'Unique list' }).getByRole('button', { name: "Kaom's Heart", exact: true });
  await expect(source).toBeVisible();
  const stashTab = views(page).getByRole('tab', { name: /^Stash/ });
  await touchDrag(page, cdp, source, [stashTab, names(page).nth(1)]);
  await expect(names(page)).toContainText(["Kaom's Heart"]);
  app.assertNoErrors();
});
