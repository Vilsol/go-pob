import { test, expect, sharedBoot } from './support/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import { optionControl, unconditionalConfigChecks } from './support/config_spec';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

// The harness behind support/test.ts: a test on a page the previous test left clean starts from what a cold boot
// shows, and a test that changed browser storage hands the next one a fresh page.
test.describe.configure({ mode: 'serial' });

const FILE = '0280-elementalist-fireball';

test('shared boot: a test imports a build, changes it and leaves another page open', async ({ page }) => {
  const xml = buildXml(FILE);
  const build = await go.loadXml(xml);
  const app = await App.open(page, FILE);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Config');
  const check = unconditionalConfigChecks()[0];
  const box = optionControl(page, 'checkbox', check.name);
  const was = await box.isChecked();
  await app.act(
    'toggle-config',
    () => box.setChecked(!was),
    async () => {
      go.setConfig(build, check.key, !was);
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(page.getByRole('button', { name: /^Undo: / })).toBeEnabled();
  await app.goTo('Items');
  app.assertNoErrors();
});

test('shared boot: the next test on that page sees what a cold boot shows, not the previous build', async ({ page, browser, viewport, baseURL }) => {
  expect(sharedBoot.pristine(page), 'the previous test left its page behind, so this one should reuse it').toBeDefined();
  const app = await App.open(page, 'new');
  await expect(page.getByRole('button', { name: 'Nothing to undo' })).toBeDisabled();

  const context = await browser.newContext({ viewport, baseURL });
  const cold = await App.open(await context.newPage(), 'new');
  expect(new URL(page.url()).pathname, 'the reused page shows another route').toBe(new URL(cold.page.url()).pathname);
  expect(await app.readStats(), 'the reused page shows other stats than a cold boot').toEqual(await cold.readStats());
  expect(await page.title()).toBe(await cold.page.title());
  cold.assertNoErrors();
  await context.close();
  app.assertNoErrors();
});

test('shared boot: a test changes a setting', async ({ page }) => {
  const app = await App.open(page, 'new');
  await page.evaluate(() => localStorage.setItem('options:showTitlebarName', 'false'));
  app.assertNoErrors();
});

test('shared boot: the next test gets a fresh page, not one with that setting', async ({ page }) => {
  expect(sharedBoot.pristine(page), 'a page whose storage changed was handed on').toBeUndefined();
  const app = await App.open(page, 'new');
  expect(await page.evaluate(() => localStorage.getItem('options:showTitlebarName'))).not.toBe('false');
  app.assertNoErrors();
});
