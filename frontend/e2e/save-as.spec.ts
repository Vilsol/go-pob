import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

const saveIn = async (page: Page, name: string) => {
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('textbox', { name: 'Build name' }).fill(name);
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
};

test('a saved build can Save As from the top bar (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  // Build.lua:1014-1061
  const canonical = await go.serializeSaved(build);
  const want = async () => expectedLines(await go.outputs(build));

  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), want);
  await expect(page.getByRole('button', { name: /^Save \(/ })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Save As' }), 'an unnamed build offers Save As').toHaveCount(0);

  const first = `e2e-save-as-first-${file}`;
  const second = `e2e-save-as-second-${file}`;
  await page.getByRole('button', { name: /^Save \(/ }).click();
  await saveIn(page, first);
  await expect(page.getByRole('banner').getByText(first, { exact: true })).toBeVisible();

  await page.reload();
  await app.savedBuild(first).waitFor({ timeout: 180_000 });
  await app.act('open-saved', () => app.openSaved(first), want);

  await page.getByRole('button', { name: 'Save As' }).click();
  await saveIn(page, second);
  await expect(page.getByRole('banner').getByText(second, { exact: true }), 'the top bar still names the old build').toBeVisible();

  await page.reload();
  await app.savedBuild(second).waitFor({ timeout: 180_000 });
  await expect(app.savedBuild(first), 'Save As replaced the original').toBeVisible();
  await app.act('open-saved-as', () => app.openSaved(second), want);
  expect(await go.decode(await app.exportCode()), 'the Save As copy differs in Go').toBe(canonical);

  app.assertNoErrors();
});

// Build.lua:1384-1396
test('Save As over an existing name asks before replacing it (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const want = async () => expectedLines(await go.outputs(build));
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), want);

  const taken = `e2e-save-as-taken-${file}`;
  await page.getByRole('button', { name: /^Save \(/ }).click();
  await saveIn(page, taken);
  await expect(page.getByRole('banner').getByText(taken, { exact: true })).toBeVisible();

  await page.getByRole('button', { name: 'Save As' }).click();
  const dialog = page.getByRole('dialog', { name: 'Save build as' });
  const ask = page.getByRole('dialog', { name: `Replace “${taken}”?` });
  await dialog.getByRole('textbox', { name: 'Build name' }).fill(taken);
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(ask).toContainText(`All builds already holds a build named ${taken}.`);
  await ask.getByRole('button', { name: 'Cancel' }).click();
  await expect(ask).toBeHidden();
  await expect(dialog.getByRole('textbox', { name: 'Build name' }), 'Cancel dropped the name').toHaveValue(taken);

  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await ask.getByRole('button', { name: 'Replace' }).click();
  await expect(dialog).toBeHidden();

  await page.reload();
  await app.savedBuild(taken).waitFor({ timeout: 180_000 });
  await expect(app.savedBuild(taken), 'the replaced build is listed once').toHaveCount(1);
  app.assertNoErrors();
});

// Build.lua:1352-1379
// Build.lua:1160-1162
test('leaving a build with unsaved changes asks Save / Don’t Save / Cancel (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const want = async () => expectedLines(await go.outputs(build));
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), want);

  await page.keyboard.press('Control+i');
  await expect(page).toHaveURL(/\/import$/);
  await app.goTo('Tree');

  const back = page.getByRole('link', { name: 'Back to builds' });
  const prompt = page.getByRole('dialog', { name: 'Save Changes' });
  await back.click();
  await expect(prompt).toContainText('This build has unsaved changes.');
  await prompt.getByRole('button', { name: 'Cancel' }).click();
  await expect(prompt).toBeHidden();
  await expect(back, 'Cancel closed the build').toBeVisible();

  const name = `e2e-leave-${file}`;
  await back.click();
  await prompt.getByRole('button', { name: 'Save', exact: true }).click();
  await saveIn(page, name);
  await expect(app.savedBuild(name)).toBeVisible({ timeout: 30_000 });

  await app.act('open-saved', () => app.openSaved(name), want);
  await page.getByRole('link', { name: 'Back to builds' }).click();
  await expect(prompt).toHaveCount(0);
  await expect(app.savedBuild(name)).toBeVisible({ timeout: 30_000 });

  await app.act('open-saved-again', () => app.openSaved(name), want);
  await app.goTo('Tree');
  await page.getByRole('spinbutton', { name: 'Character level' }).fill('37');
  await expect(page.getByRole('button', { name: 'Save (unsaved changes)' })).toBeVisible({ timeout: 10_000 });
  await page.getByRole('link', { name: 'Back to builds' }).click();
  await prompt.getByRole('button', { name: "Don't Save" }).click({ timeout: 10_000 });
  await expect(app.savedBuild(name)).toBeVisible({ timeout: 30_000 });
  app.assertNoErrors();
});
