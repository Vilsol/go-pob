import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';

test.describe.configure({ mode: 'default' });

const setLevel = async (app: App, n: number) => {
  await app.openBuildControls();
  const before = await app.seq();
  await app.page.getByRole('spinbutton', { name: 'Character level' }).fill(String(n));
  await app.waitSettled(before);
};

const save = async (page: Page, name: string) => {
  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('textbox', { name: 'Build name' }).fill(name);
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
};

const level = (app: App, name: string) => app.savedBuild(name).getByRole('gridcell').nth(4);

const toList = async (page: Page) => {
  await page.getByRole('link', { name: 'Back to builds' }).click();
  const prompt = page.getByRole('dialog', { name: 'Save Changes' });
  await expect(prompt.or(page.getByRole('button', { name: 'New build', exact: true }))).toBeVisible();
  if (await prompt.isVisible()) await prompt.getByRole('button', { name: "Don't Save" }).click();
  await expect(page.getByRole('grid', { name: 'Builds' })).toBeVisible();
};

test('two tabs each save a build and both builds are kept', async ({ context }) => {
  const a = await context.newPage();
  const b = await context.newPage();
  const appA = await App.open(a, 'two tabs: first');
  const appB = await App.open(b, 'two tabs: second');

  await setLevel(appA, 21);
  await save(a, 'e2e-tab-first');
  await setLevel(appB, 42);
  await save(b, 'e2e-tab-second');
  await toList(a);
  await a.getByRole('button', { name: 'New build', exact: true }).click();
  await setLevel(appA, 63);
  await save(a, 'e2e-tab-third');

  await a.reload();
  await appA.savedBuild('e2e-tab-first').or(appA.savedBuild('e2e-tab-second')).first().waitFor({ timeout: 180_000 });
  await expect(level(appA, 'e2e-tab-first'), "the second tab's save replaced the first tab's build").toHaveText('21');
  await expect(level(appA, 'e2e-tab-second'), "a later save replaced the second tab's build").toHaveText('42');
  await expect(level(appA, 'e2e-tab-third')).toHaveText('63');

  appA.assertNoErrors();
  appB.assertNoErrors();
});

test('when the writing tab closes, another tab saves in its place', async ({ context }) => {
  const a = await context.newPage();
  const b = await context.newPage();
  const appA = await App.open(a, 'handover: first');
  const appB = await App.open(b, 'handover: second');

  await setLevel(appA, 21);
  await save(a, 'e2e-handover-first');
  await a.close();
  await setLevel(appB, 42);
  await save(b, 'e2e-handover-second');

  const c = await context.newPage();
  const appC = await App.open(c, 'handover: third');
  await toList(c);
  await expect(level(appC, 'e2e-handover-first')).toHaveText('21');
  await expect(level(appC, 'e2e-handover-second')).toHaveText('42');

  appB.assertNoErrors();
  appC.assertNoErrors();
});

test("a build saved in one tab shows in another tab's build list without a reload", async ({ context }) => {
  const a = await context.newPage();
  const b = await context.newPage();
  const appA = await App.open(a, 'live list: writer');
  const appB = await App.open(b, 'live list: reader');

  await toList(a);
  await setLevel(appB, 33);
  await save(b, 'e2e-live-from-reader');
  await expect(level(appA, 'e2e-live-from-reader'), "the writing tab's list shows the reading tab's save").toHaveText('33');

  await toList(b);
  await a.getByRole('button', { name: 'New build', exact: true }).click();
  await setLevel(appA, 44);
  await save(a, 'e2e-live-from-writer');
  await expect(level(appB, 'e2e-live-from-writer'), "the reading tab's list shows the writing tab's save").toHaveText('44');

  appA.assertNoErrors();
  appB.assertNoErrors();
});
