// ImportTab.lua:609-628, 701-705

import { test, expect } from './support/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

test("a pasted character JSON opens as a new build with its items, skills and passive tree as Go imports them (PoB's sample character)", async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const json = readFileSync(resolve(import.meta.dirname, '../../testdata/import/sample_character.json'), 'utf8');
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));

  // an import never writes into the open build
  const made = await go.emptyBuild();
  const region = (await app.openImport()).getByRole('region', { name: 'Build import' });
  const box = region.getByRole('textbox', { name: /^To import a build/ });
  await box.fill(json);
  await expect(box).toHaveAccessibleDescription('JSON is valid');
  await app.act(
    'import a character JSON',
    async () => {
      await region.getByRole('button', { name: 'Import', exact: true }).click();
      // Build.lua:1421-1448: the imported build's unsaved changes are asked about first
      await page.getByRole('dialog', { name: 'Save Changes' }).getByRole('button', { name: "Don't Save" }).click();
    },
    async () => {
      const m = await go.model();
      expect((await m.exposition.ImportItemsAndSkills(made, json, true, true, false)).unwrap()).toBe('Items and skills successfully imported.');
      expect((await m.exposition.ImportPassiveTreeAndJewels(made, json, true)).unwrap()).toBe('Passive tree and jewels successfully imported.');
      return expectedLines(await go.outputs(made));
    }
  );
  await expect(region.getByRole('status')).toHaveText('Items and skills successfully imported. Passive tree and jewels successfully imported.');
  // a character import stays in the dialog, which shows its result
  await page.keyboard.press('Escape');
  // ImportTab.lua:1396-1402, 1424-1460
  await expect(page.getByRole('combobox', { name: 'Main skill' }).locator('option').first()).toHaveText('Kinetic Blast');
  await app.goTo('Items');
  // the board shows item names on hover; the slot's accessible name always carries it
  await expect(page.getByLabel(/^[^:]+: Chimeric Guardian$/).first()).toBeVisible();
  app.assertNoErrors();
});
