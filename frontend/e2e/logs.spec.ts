import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

test('the Logs panel lists a page error, the worker’s slow-calculation timings, and copies the log for a bug report', async ({ page, context }) => {
  const file = '1591-necromancer-summon-holy-relic-of-conviction-15';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));

  await page.evaluate(() =>
    setTimeout(() => {
      throw new Error('provoked for the log');
    })
  );
  await expect.poll(() => app.errors.length).toBe(1);

  await page.getByRole('button', { name: /^Logs/ }).click();
  const panel = page.getByRole('dialog', { name: 'Logs' });
  const log = panel.getByRole('list', { name: 'Application log' });
  await expect(log.getByRole('listitem').filter({ hasText: 'provoked for the log' })).toHaveText(/ERRORpageError: provoked for the log$/);
  await expect(
    log
      .getByRole('listitem')
      .filter({ hasText: /calculated in .*\(ImportCode\)/ })
      .last(),
    'the calculation worker’s line reached the page'
  ).toHaveText(/INFOcalccalculated in \d+ ms \(ImportCode\)$/);
  await expect(page.getByRole('button', { name: 'Logs, 1 error' })).toBeVisible();

  await panel.getByRole('button', { name: 'Error', pressed: false }).click();
  await expect(log.getByRole('listitem')).toHaveCount(1);
  await panel.getByRole('button', { name: /^All / }).click();

  await panel.getByRole('button', { name: 'Copy for bug reports' }).click();
  await expect(panel.getByRole('button', { name: 'Copied' })).toBeVisible();
  const report = await page.evaluate(() => navigator.clipboard.readText());
  expect(report).toMatch(/^go-pob \S+/);
  expect(report).toContain(`browser: ${await page.evaluate(() => navigator.userAgent)}`);
  expect(report).toMatch(/ERROR \[page\] Error: provoked for the log/);
  expect(report).toMatch(/INFO {2}\[calc\] calculated in \d+ ms \(ImportCode\)/);
});
