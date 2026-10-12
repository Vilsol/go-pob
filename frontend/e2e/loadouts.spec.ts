// Build.lua:301-415, 753-878

import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

test('New Loadout, then choosing it, as Go does (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const m = await go.model();
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));

  const select = page.getByRole('combobox', { name: 'Loadout' });
  await expect(select).toHaveValue('Loadouts:');
  await select.selectOption('New Loadout');
  const dialog = page.getByRole('dialog', { name: 'Set Name' });
  await dialog.getByRole('textbox', { name: 'Enter name for this loadout' }).fill('Mapping');
  await app.act(
    'new-loadout',
    () => dialog.getByRole('button', { name: 'Save' }).click(),
    async () => {
      await m.exposition.NewLoadout(build, 'Mapping');
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(select.locator('option')).toContainText(['Loadouts:', 'Mapping', 'New Loadout', 'Sync']);
  await expect(select).toHaveValue('Loadouts:');

  await app.act(
    'select-loadout',
    () => select.selectOption('Mapping'),
    async () => {
      expect(await m.exposition.SelectLoadout(build, 'Mapping')).toBe(true);
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(select).toHaveValue('Mapping');
  await app.goTo('Tree');
  await expect(page.getByRole('combobox', { name: 'Tree set' })).toHaveText('Mapping');
  app.assertNoErrors();
});
