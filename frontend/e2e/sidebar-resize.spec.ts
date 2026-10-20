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
      const [labelEl, valueEl] = [row.querySelector('.stat-label')!, row.querySelector('.stat-value')!];
      const [label, value] = [labelEl, valueEl].map((e) => e.getBoundingClientRect());
      const r = row.getBoundingClientRect();
      return {
        text: row.textContent?.trim() ?? '',
        oneLine: Math.abs(label.top - value.top) < 4 && value.left >= label.right - 1,
        labelCut: labelEl.scrollWidth > labelEl.clientWidth,
        valueCut: valueEl.scrollWidth > valueEl.clientWidth,
        inside: label.left >= r.left - 1 && value.right <= r.right + 1 && label.right <= r.right + 1
      };
    })
  );

test('sidebar: dragged narrower and wider, a value too wide covers the end of its label, the width survives a reload', async ({ page }) => {
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
  expect(
    narrow.filter((r) => !r.oneLine),
    'at 240 px every row still fits one line'
  ).toEqual([]);
  const cut = narrow.filter((r) => r.labelCut);
  expect(cut.length, 'some labels run under their value at 240 px').toBeGreaterThan(0);
  expect(cut.length, 'not every label is cut: the choice is per row').toBeLessThan(narrow.length);
  expect(
    narrow.filter((r) => r.valueCut),
    'no value is cut'
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
