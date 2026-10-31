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

test("a set shared in one tab shows in another tab's shared item sets without a reload", async ({ context }) => {
  const a = await context.newPage();
  const b = await context.newPage();
  const appA = await App.open(a, 'shared sets: sharing');
  const appB = await App.open(b, 'shared sets: other');
  const manage = async (app: App) => {
    await app.goTo('Items');
    await app.page.getByRole('button', { name: 'Manage…' }).click();
    return app.page.getByRole('dialog', { name: 'Manage item sets' }).getByRole('region', { name: 'Shared item sets' });
  };
  const regionA = await manage(appA);
  const regionB = await manage(appB);
  const sharedB = regionB.getByRole('listbox', { name: 'Shared sets' }).getByRole('option');
  await expect(sharedB).toHaveCount(0);

  await regionA.getByRole('button', { name: 'Share set' }).click();
  await expect(regionA.getByRole('listbox', { name: 'Shared sets' }).getByRole('option')).toHaveCount(1);
  await expect(sharedB, "the other tab's list shows the shared set").toHaveCount(1);

  await sharedB.first().click();
  await regionB.getByRole('button', { name: 'Remove' }).click();
  await regionB.getByRole('group', { name: 'Confirm shared delete' }).getByRole('button', { name: 'Delete' }).click();
  await expect(sharedB).toHaveCount(0);
  await expect(regionA.getByRole('listbox', { name: 'Shared sets' }).getByRole('option'), "the sharing tab's list drops the removed set").toHaveCount(0);

  appA.assertNoErrors();
  appB.assertNoErrors();
});

test("a build saved in one tab shows in another tab's Compare picker and Save As folders without a reload", async ({ context }) => {
  const a = await context.newPage();
  const b = await context.newPage();
  const appA = await App.open(a, 'live pickers: open');
  const appB = await App.open(b, 'live pickers: saving');

  await appA.goTo('Compare');
  await a.getByRole('button', { name: 'Choose a saved build…' }).click();
  const picked = a.getByRole('dialog', { name: 'Compare with a saved build' }).getByRole('option').filter({ hasText: 'e2e-live-compare' });
  await expect(a.getByRole('dialog', { name: 'Compare with a saved build' })).toBeVisible();
  await expect(picked).toHaveCount(0);
  await setLevel(appB, 27);
  await save(b, 'e2e-live-compare');
  await expect(picked, "the Compare picker lists the other tab's save").toHaveCount(1);
  await a.getByRole('dialog', { name: 'Compare with a saved build' }).getByRole('button', { name: 'Cancel' }).click();

  await a.getByRole('button', { name: /^Save \(/ }).click();
  const folders = a.getByRole('dialog', { name: 'Save build as' }).getByRole('combobox', { name: 'Folder' });
  await expect(folders).toBeVisible();
  await expect(folders.getByRole('option', { name: /e2e-live-folder/ })).toHaveCount(0);
  await b.getByRole('button', { name: 'Save As' }).click();
  const dialogB = b.getByRole('dialog', { name: 'Save build as' });
  await dialogB.getByRole('button', { name: 'New folder' }).click();
  const prompt = b.getByRole('dialog', { name: 'New folder' });
  await prompt.getByRole('textbox', { name: 'Folder name' }).fill('e2e-live-folder');
  await prompt.getByRole('button', { name: 'Create' }).click();
  await expect(folders.getByRole('option', { name: /e2e-live-folder/ }), "the Save As folders show the other tab's new folder").toHaveCount(1);

  appA.assertNoErrors();
  appB.assertNoErrors();
});

test('the build sort, the share site and the panel widths another tab stores reach this tab without a reload', async ({ context }) => {
  const a = await context.newPage();
  const b = await context.newPage();
  const appA = await App.open(a, 'layout: storing tab');
  const appB = await App.open(b, 'layout: other tab');
  const store = (key: string, value: string) => a.evaluate(([k, v]) => localStorage.setItem(k, v), [key, value]);

  await appB.goTo('Notes');
  const sidebar = b.getByRole('separator', { name: 'Resize the sidebar' });
  await expect(sidebar).toHaveAttribute('aria-valuenow', '320');
  await store('go-pob:sidebar-width', '400');
  await expect(sidebar, "the sidebar takes the other tab's width").toHaveAttribute('aria-valuenow', '400');

  await appB.goTo('Tree');
  await b.getByPlaceholder('Search nodes and stats').fill('life');
  const details = b.getByRole('separator', { name: 'Resize tree details' });
  await expect(details).toHaveAttribute('aria-valuenow', '360');
  await store('go-pob:tree-details-width', '500');
  await expect(details, "the tree details take the other tab's width").toHaveAttribute('aria-valuenow', '500');

  await appB.goTo('Import / Export');
  const site = b.getByRole('region', { name: 'Build sharing' }).getByRole('combobox');
  const sites = await site.locator('option').allTextContents();
  expect(sites.length, 'more than one share site to choose from').toBeGreaterThan(1);
  await expect(site).toHaveValue(sites[0]);
  await store('options:lastExportWebsite', sites.at(-1)!);
  await expect(site, "the share site follows the other tab's last share").toHaveValue(sites.at(-1)!);

  await toList(b);
  const level = b.getByRole('grid', { name: 'Builds' }).getByRole('columnheader', { name: /^LEVEL/ });
  await store('options:buildSortMode', JSON.stringify({ key: 'level', dir: 'asc' }));
  await expect(level, "the build list takes the other tab's sort").toHaveAttribute('aria-sort', 'ascending');

  appA.assertNoErrors();
  appB.assertNoErrors();
});
