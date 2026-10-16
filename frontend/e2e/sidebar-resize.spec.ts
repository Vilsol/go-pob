import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { layoutFindings } from './support/layout';
import { expectedLines } from './support/stats';

test.use({ viewport: { width: 960, height: 1080 } });

const BUILD = '0339-reliquarian-shield-crush-of-the-chieftain-2';
const NAME = `sidebar resize ${BUILD}`;

const rows = (page: Page) =>
  page.getByRole('region', { name: 'Build stats' }).evaluate((root) =>
    [...root.querySelectorAll('.stat-row')].map((row) => {
      const [label, value] = [row.querySelector('.stat-label')!, row.querySelector('.stat-value')!].map((e) => e.getBoundingClientRect());
      const r = row.getBoundingClientRect();
      return {
        text: row.textContent?.trim() ?? '',
        oneLine: Math.abs(label.top - value.top) < 4 && value.left >= label.right - 1,
        stacked: value.bottom <= label.top + 1,
        inside: label.left >= r.left - 1 && value.right <= r.right + 1 && label.right <= r.right + 1
      };
    })
  );

test('sidebar: dragged narrower and wider, rows stack per row, the width survives a reload', async ({ page }) => {
  test.setTimeout(10 * 60_000);
  const xml = buildXml(BUILD);
  const build = await go.loadXml(xml);
  const app = await App.open(page, BUILD);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('textbox', { name: 'Build name' }).fill(NAME);
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
  await app.goTo('Notes');

  const aside = page.getByRole('complementary', { name: 'Main skill and stats' });
  const handle = page.getByRole('separator', { name: 'Resize the sidebar' });
  const width = async () => (await aside.boundingBox())!.width;
  expect(await width()).toBe(320);
  const wide = await rows(page);
  expect(wide.length).toBeGreaterThan(20);
  expect(
    wide.filter((r) => !r.oneLine),
    'at 320 px every row of this build fits one line'
  ).toEqual([]);

  const box = (await handle.boundingBox())!;
  await page.mouse.move(box.x + box.width / 2, box.y + 200);
  await page.mouse.down();
  await page.mouse.move(box.x - 300, box.y + 200, { steps: 8 });
  await page.mouse.up();
  expect(await width()).toBe(240);
  const narrow = await rows(page);
  const stacked = narrow.filter((r) => r.stacked);
  expect(stacked.length, 'some rows stack at 240 px').toBeGreaterThan(0);
  expect(stacked.length, 'not every row stacks: the choice is per row').toBeLessThan(narrow.length);
  expect(
    narrow.filter((r) => !r.oneLine && !r.stacked),
    'a row is either one line or stacked'
  ).toEqual([]);
  expect(
    narrow.filter((r) => !r.inside),
    'no row spills out of the sidebar'
  ).toEqual([]);
  expect(await layoutFindings(page, 'aside', [{ selector: '[aria-label="Build stats"]', axis: 'y', why: 'the stat list' }])).toEqual([]);

  await handle.focus();
  await page.keyboard.press('ArrowRight');
  expect(await width()).toBe(256);
  await expect(handle).toHaveAttribute('aria-valuenow', '256');

  await page.reload();
  await app.savedBuild(NAME).waitFor({ timeout: 180_000 });
  await app.openSaved(NAME);
  await expect(aside).toBeVisible({ timeout: 180_000 });
  expect(await width(), 'the width is kept across a reload').toBe(256);
  app.assertNoErrors();
});
