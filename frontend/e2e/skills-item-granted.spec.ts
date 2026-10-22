// SkillsTab.lua:383-417, CalcSetup.lua:1794-1813

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { test, expect } from '@playwright/test';

import { App } from './support/app';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

const FIXTURE = resolve(import.meta.dirname, '../../calculator/testdata/item-granted-supports/dancing-dervish.xml');

test('skills: an item-granted group takes its supports from the item’s other groups and offers no gem picker', async ({ page }) => {
  test.setTimeout(5 * 60_000);
  const xml = readFileSync(FIXTURE, 'utf8');
  const build = await go.loadXml(xml);
  const app = await App.open(page, 'dancing-dervish');
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Skills');

  const groups = page.getByRole('region', { name: 'Socket groups' });
  const editor = page.getByRole('region', { name: 'Selected group' });
  await groups
    .getByRole('button', { name: /Minion Damage/ })
    .first()
    .click();
  await expect(editor.getByRole('combobox', { name: 'Add a gem' })).toHaveCount(1);
  const set = build.ActiveSkillSetIndex();
  const toggle = editor.getByRole('checkbox', { name: 'Enabled', exact: true });
  const times: number[] = [];
  for (const on of [false, true, false, true]) {
    const before = await app.seq();
    const t0 = await app.now();
    await app.act(
      on ? 'enable-support-group' : 'disable-support-group',
      () => toggle.click(),
      async () => {
        build.Skills.SkillSets[set].Skills[2].Enabled = on;
        return expectedLines(await go.outputs(build));
      }
    );
    const ticks = await app.timingsSince(before);
    times.push(ticks[0].renderedAt - t0);
    console.log(
      `toggle ${on}: first ${(ticks[0].renderedAt - t0).toFixed(0)} last ${(ticks.at(-1)!.renderedAt - t0).toFixed(0)} ticks ${ticks.length} calc ${ticks.map((t) => t.calcMs.toFixed(0)).join('/')}`
    );
  }
  console.log(`support-group toggle settle ms: ${times.map((t) => t.toFixed(0)).join(', ')}`);
  await groups
    .getByRole('button', { name: /Manifest Dancing Dervishes/ })
    .first()
    .click();
  await expect(editor.getByText('You cannot add support gems to this group')).toBeVisible();
  await expect(editor.getByRole('combobox', { name: 'Add a gem' })).toHaveCount(0);
  await expect(editor.getByRole('button', { name: /^Remove / })).toHaveCount(0);
  app.assertNoErrors();
});
