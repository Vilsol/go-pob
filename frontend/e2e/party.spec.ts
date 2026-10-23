import { test, expect } from './support/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

const BUILD = '0381-assassin-soulrend-of-reaping-2';
const EXPORTER = '0160-elementalist-winter-orb-10';

test(`party import, disable and rebuild on ${BUILD}`, { tag: '@smoke' }, async ({ page }) => {
  const source = buildXml(BUILD);
  const model = await go.loadXml(source);
  const app = await App.open(page, BUILD);
  await app.importCode(await go.encode(source), async () => expectedLines(await go.outputs(model)));
  await app.goTo('Party');

  const exporter = buildXml(EXPORTER);
  const imp = page.getByRole('region', { name: 'Party import' });
  await imp.getByRole('textbox', { name: /Enter a build code/ }).fill(await go.encode(exporter));
  const importButton = imp.getByRole('button', { name: 'Import' });
  await expect(importButton).toBeEnabled();
  model.ImportParty(exporter, 'All', false).unwrap();
  await app.act(
    'import a party member',
    () => importButton.click(),
    async () => expectedLines(await go.outputs(model))
  );
  await expect(page.getByRole('region', { name: 'Curses' }).locator('[data-entry]')).toHaveText([/^Frostbite/]);

  model.DisablePartyEffects();
  await app.act(
    'disable party effects',
    () => imp.getByRole('checkbox', { name: 'Disable Party Effects' }).check(),
    async () => expectedLines(await go.outputs(model))
  );
  await expect(page.getByRole('group', { name: 'Effect of the party' })).toContainText('Party effects disabled');

  model.RebuildParty();
  await app.act(
    'rebuild the party',
    () => imp.getByRole('button', { name: 'Rebuild All' }).click(),
    async () => expectedLines(await go.outputs(model))
  );

  const saved = await go.loadXml(await go.decode(await app.exportCode()));
  expect(saved.PartyBuffers()).toEqual(model.PartyBuffers());
  app.assertNoErrors();
});
