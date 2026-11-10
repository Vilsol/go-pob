import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

type Logged = { kind: string; label: string; state: string; phases: string[] };

// every job the page finished, the fast ones included
const finished = (page: Page, kind: string) =>
  page.evaluate(
    (k) =>
      ((globalThis as unknown as { __pobJobs?: { kind: string; label: string; state: string; phases: { name: string }[] }[] }).__pobJobs ?? [])
        .filter((j) => j.kind === k)
        .map((j): Logged => ({ kind: j.kind, label: j.label, state: j.state, phases: j.phases.map((p) => p.name) })),
    kind
  );

const hold = (ms: number) => new Promise((r) => setTimeout(r, ms));

// the service worker would answer the page's code itself, past the held route
test.describe(() => {
  test.use({ serviceWorkers: 'block' });

  test('a page whose code is still loading shows its job in the top bar and the progress bar until it opens', async ({ page }) => {
    const app = await App.open(page, '0280-elementalist-fireball');
    let held = 0;
    await page.route(/\/_app\/immutable\/nodes\/[^/]+\.js$/, async (route) => {
      held++;
      await hold(2000);
      await route.continue();
    });
    await page.getByRole('navigation', { name: 'Build pages' }).getByRole('link', { name: 'Items' }).click();
    await expect(page.getByRole('button', { name: 'Activity: Opening Items · Loading the page' })).toBeVisible();
    await expect(page.getByRole('progressbar', { name: 'Activity' })).toBeVisible();
    await expect(page.getByRole('main', { name: 'Items' })).toBeVisible();
    await expect(page.getByRole('button', { name: /^Activity/ })).toBeHidden();
    await expect(page.getByRole('progressbar', { name: 'Activity' })).toBeHidden();
    expect(held).toBeGreaterThan(0);
    expect(await finished(page, 'page:items')).toEqual([{ kind: 'page:items', label: 'Opening Items', state: 'ok', phases: ['Loading the page'] }]);
    expect(app.errors).toEqual([]);
  });
});

test('a build link that downloads slowly shows a progress toast, then the import runs as a job with its phases (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const code = await go.encode(xml);
  await page.route('**/api/build?*', async (route) => {
    await hold(2000);
    await route.fulfill({ status: 200, contentType: 'text/plain; charset=utf-8', body: code });
  });
  const app = await App.open(page, '0005-occultist-vaal-double-strike');
  const region = (await app.openImport()).getByRole('region', { name: 'Build import' });
  await region.getByRole('textbox', { name: /^To import a build/ }).fill('https://pobb.in/h-M3H3It2fDw');
  await expect(region.getByRole('textbox', { name: /^To import a build/ })).toHaveAccessibleDescription('URL is valid (pobb.in)');
  await app.act(
    'import a slow pobb.in link',
    async () => {
      await region.getByRole('button', { name: 'Import', exact: true }).click();
      await expect(page.getByRole('status').filter({ hasText: 'Downloading the build' }).first()).toBeVisible();
    },
    async () => expectedLines(await go.outputs(build))
  );
  await page.waitForURL((u) => u.pathname.endsWith('/tree'));
  expect(await finished(page, 'download')).toEqual([{ kind: 'download', label: 'Downloading the build', state: 'ok', phases: [] }]);
  await expect
    .poll(() => finished(page, 'import'))
    .toEqual([
      {
        kind: 'import',
        label: 'Importing build',
        state: 'ok',
        // the calculation's own parts, named by the engine as it reaches them
        phases: [
          'Decoding the code',
          'Importing and calculating',
          'Setting up the items, tree and skills',
          'Calculating the main skill',
          'Calculating Full DPS',
          'Opening the Tree'
        ]
      }
    ]);
  app.assertNoErrors();
});

test('saving and opening a saved build each run as a job with their phases (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));

  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('textbox', { name: 'Build name' }).fill('activity save');
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
  await expect.poll(() => finished(page, 'save')).toEqual([{ kind: 'save', label: 'Saving activity save', state: 'ok', phases: [] }]);

  await page.getByRole('spinbutton', { name: 'Character level' }).fill('42');
  await page.getByRole('button', { name: /^Save \(unsaved/ }).click();
  await expect(page.getByRole('button', { name: /^Save \(unsaved/ })).toBeHidden();
  await expect
    .poll(async () => (await finished(page, 'save')).at(-1))
    .toEqual({ kind: 'save', label: 'Saving activity save', state: 'ok', phases: ['Writing the build'] });

  await page.getByRole('link', { name: 'Back to builds' }).click();
  await expect(page.getByRole('grid', { name: 'Builds' })).toBeVisible();
  await app.openSaved('activity save');
  await expect(page.getByRole('main', { name: 'Tree' })).toBeVisible();
  await expect.poll(() => finished(page, 'open')).toEqual([{ kind: 'open', label: 'Opening activity save', state: 'ok', phases: ['Reading and calculating'] }]);
  app.assertNoErrors();
});
