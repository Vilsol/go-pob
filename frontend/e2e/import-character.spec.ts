// ImportTab.lua:609-628, 701-705

import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

test("a pasted character JSON imports its items, skills and passive tree into this build as Go does (PoB's sample character)", async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const json = readFileSync(resolve(import.meta.dirname, '../../testdata/import/sample_character.json'), 'utf8');
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));

  await app.goTo('Import / Export');
  const region = page.getByRole('region', { name: 'Build import' });
  const box = region.getByRole('textbox', { name: /^To import a build/ });
  await box.fill(json);
  await expect(box).toHaveAccessibleDescription('JSON is valid');
  await app.act(
    'import a character JSON',
    () => region.getByRole('button', { name: 'Import', exact: true }).click(),
    async () => {
      const m = await go.model();
      expect((await m.exposition.ImportItemsAndSkills(build, json, true, true, false)).unwrap()).toBe('Items and skills successfully imported.');
      expect((await m.exposition.ImportPassiveTreeAndJewels(build, json, true)).unwrap()).toBe('Passive tree and jewels successfully imported.');
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(region.getByRole('status')).toHaveText('Items and skills successfully imported. Passive tree and jewels successfully imported.');
  // ImportTab.lua:1396-1402, 1424-1460
  await expect(page.getByRole('combobox', { name: 'Main skill' }).locator('option').first()).toHaveText('Kinetic Blast');
  await app.goTo('Items');
  await expect(page.getByText('Chimeric Guardian').first()).toBeVisible();
  app.assertNoErrors();
});
