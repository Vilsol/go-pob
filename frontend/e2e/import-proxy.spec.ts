// BuildSiteTools.lua:88-113, ImportTab.lua:321-472, 813-1020

import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

test('a pobb.in link is downloaded through /api/build and imported as a new build, as Go imports its code (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const code = await go.encode(xml);
  const asked: string[] = [];
  await page.route('**/api/build?*', (route) => {
    asked.push(new URL(route.request().url()).search);
    return route.fulfill({ status: 200, contentType: 'text/plain; charset=utf-8', body: code });
  });
  const app = await App.open(page, '0005-occultist-vaal-double-strike');
  await app.goTo('Import / Export');
  const region = page.getByRole('region', { name: 'Build import' });
  await region.getByRole('textbox', { name: /^To import a build/ }).fill('https://pobb.in/h-M3H3It2fDw');
  await expect(region.getByRole('textbox', { name: /^To import a build/ })).toHaveAccessibleDescription('URL is valid (pobb.in)');
  await region.getByRole('radio', { name: /Import as a new build/ }).check();
  await app.act(
    'import a pobb.in link',
    () => region.getByRole('button', { name: 'Import', exact: true }).click(),
    async () => expectedLines(await go.outputs(build))
  );
  expect(asked).toEqual(['?site=POBBin&id=h-M3H3It2fDw']);
  app.assertNoErrors();
});

type Sample = { name: string; class: string; league: string; level: number; passives: object; jewels: unknown[]; equipment: unknown[] };

const serveSample = async (page: import('@playwright/test').Page) => {
  const sample = (JSON.parse(readFileSync(resolve(import.meta.dirname, '../../testdata/import/sample_character.json'), 'utf8')) as { character: Sample })
    .character;
  const entry = { name: sample.name, realm: 'pc', class: sample.class, league: sample.league, level: sample.level };
  const passives = JSON.stringify({ ...sample.passives, items: sample.jewels });
  const items = JSON.stringify({ items: sample.equipment });
  const asked: string[] = [];
  await page.route('**/api/poe/*', (route) => {
    const url = new URL(route.request().url());
    asked.push(`${url.pathname}${url.search}`);
    const endpoint = url.pathname.split('/').at(-1);
    if (endpoint === 'characters') return route.fulfill({ json: [entry] });
    if (endpoint === 'profile') return route.fulfill({ json: { accountName: 'Sample-1234' } });
    return route.fulfill({ status: 200, contentType: 'application/json', body: endpoint === 'passives' ? passives : items });
  });
  return { sample, entry, passives, items, asked };
};

test("the account import lists the account's characters and imports one's items, skills and passive tree as Go does (PoB's sample character)", async ({
  page
}) => {
  const { sample, entry, passives, items, asked } = await serveSample(page);
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Import / Export');
  const chars = page.getByRole('region', { name: 'Character import' });
  await chars.getByRole('textbox', { name: /Account name/ }).fill('sample#1234');
  await chars.getByRole('button', { name: 'Start' }).click();
  await expect(chars.getByRole('status')).toHaveText('Character import status: Character list successfully retrieved.');
  await expect(chars.getByRole('combobox', { name: 'Character' }).locator('option')).toHaveText([`${sample.name} — ${sample.class} lvl ${sample.level}`]);

  const m = await go.model();
  await app.act(
    'import the items and skills',
    () => chars.getByRole('button', { name: 'Items and Skills' }).click(),
    async () => {
      expect((await m.exposition.ImportAccountItems(build, JSON.stringify(entry), items, true, true, false)).unwrap()).toBe(
        'Items and skills successfully imported.'
      );
      return expectedLines(await go.outputs(build));
    }
  );
  await chars.getByRole('button', { name: 'Passive Tree and Jewels' }).click();
  const confirm = page.getByRole('dialog', { name: 'Character Import' });
  await expect(confirm).toContainText('Importing the passive tree will overwrite your current tree.');
  await app.act(
    'import the passive tree and jewels',
    () => confirm.getByRole('button', { name: 'Import' }).click(),
    async () => {
      expect((await m.exposition.ImportAccountPassiveTree(build, JSON.stringify(entry), passives, true)).unwrap()).toBe(
        'Passive tree and jewels successfully imported.'
      );
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(chars.getByRole('status')).toHaveText('Character import status: Passive tree and jewels successfully imported.');
  expect(asked).toEqual([
    '/api/poe/characters?realm=pc&account=sample%231234',
    '/api/poe/profile?realm=pc&account=sample%231234',
    `/api/poe/items?realm=pc&account=Sample%231234&character=${sample.name}`,
    `/api/poe/passives?realm=pc&account=Sample%231234&character=${sample.name}`
  ]);
  app.assertNoErrors();
});

// CompareTab.lua:244-271, 1288-1308
test("Re-import Current on the Compare page re-imports the selected character as Go does (PoB's sample character)", async ({ page }) => {
  const { sample, entry, passives, items } = await serveSample(page);
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Import / Export');
  const chars = page.getByRole('region', { name: 'Character import' });
  await chars.getByRole('textbox', { name: /Account name/ }).fill('sample#1234');
  await chars.getByRole('button', { name: 'Start' }).click();
  await expect(chars.getByRole('status')).toHaveText('Character import status: Character list successfully retrieved.');

  const region = page.getByRole('region', { name: 'Build import' });
  const box = region.getByRole('textbox', { name: /^To import a build/ });
  await box.fill(await go.encode(buildXml('0075-elementalist-winter-orb')));
  await region.getByRole('radio', { name: /Import as a comparison/ }).click();
  await region.getByRole('button', { name: 'Import', exact: true }).click();
  await page.waitForURL((u) => u.pathname.endsWith('/compare'));
  const reimport = page.getByRole('button', { name: 'Re-import Current' });
  await expect(reimport).toBeEnabled({ timeout: 60_000 });
  await expect(reimport).toHaveAttribute(
    'title',
    `Re-import character from the game server:\n${sample.name} (${sample.class}, ${sample.league})\nRefreshes passive tree, jewels, items, and skills.`
  );

  const m = await go.model();
  await app.act(
    're-import: items and skills',
    () => reimport.click(),
    async () => {
      expect((await m.exposition.ImportAccountItems(build, JSON.stringify(entry), items, true, true, false)).unwrap()).toBe(
        'Items and skills successfully imported.'
      );
      return expectedLines(await go.outputs(build));
    }
  );
  const confirm = page.getByRole('dialog', { name: 'Character Import' });
  await expect(confirm).toContainText('Importing the passive tree will overwrite your current tree.');
  await app.act(
    're-import: passive tree and jewels',
    () => confirm.getByRole('button', { name: 'Import' }).click(),
    async () => {
      expect((await m.exposition.ImportAccountPassiveTree(build, JSON.stringify(entry), passives, true)).unwrap()).toBe(
        'Passive tree and jewels successfully imported.'
      );
      return expectedLines(await go.outputs(build));
    }
  );
  app.assertNoErrors();
});

// ImportTab.lua:534-554, BuildSiteTools.lua:47-82
test('Share uploads the build code through /api/share, shows the link and copies it (0280)', async ({ page }) => {
  await page.context().grantPermissions(['clipboard-read', 'clipboard-write']);
  const posted: { search: string; body: string }[] = [];
  await page.route('**/api/share?*', (route) => {
    posted.push({ search: new URL(route.request().url()).search, body: route.request().postData() ?? '' });
    return route.fulfill({ status: 200, contentType: 'text/plain; charset=utf-8', body: 'https://pobb.in/Ab-12' });
  });
  const app = await App.open(page, '0280-elementalist-fireball');
  await app.goTo('Import / Export');
  const sharing = page.getByRole('region', { name: 'Build sharing' });
  await expect(sharing.getByRole('textbox', { name: 'Build code' })).not.toHaveValue('');
  const code = await sharing.getByRole('textbox', { name: 'Build code' }).inputValue();
  await sharing.getByRole('combobox', { name: 'Share via' }).selectOption('pobb.in');
  await sharing.getByRole('button', { name: 'Share' }).click();
  await expect(sharing.getByRole('textbox', { name: 'Share link' })).toHaveValue('https://pobb.in/Ab-12');
  await expect(page.getByRole('status').filter({ hasText: 'Link copied' })).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe('https://pobb.in/Ab-12');
  expect(posted).toEqual([{ search: '?site=POBBin', body: code }]);
  expect(await go.decode(posted[0].body)).toContain('<PathOfBuilding');
  app.assertNoErrors();
});
