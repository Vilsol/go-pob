import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

// SkillsTab.lua:228-237, 345-391
const BUILD = '0005-occultist-vaal-double-strike';

test(`an item-granted group on ${BUILD}`, async ({ page }) => {
  const source = buildXml(BUILD);
  const model = await go.loadXml(source);
  const app = await App.open(page, BUILD);
  await app.importCode(await go.encode(source), async () => expectedLines(await go.outputs(model)));

  const set = model.Skills.SkillSets[model.ActiveSkillSetIndex()];
  const index = set.Skills.findIndex((g) => g.SourceAttr?.startsWith('Item:'));
  expect(index, 'the build has no item-granted group').toBeGreaterThanOrEqual(0);
  const group = set.Skills[index];
  const itemName = group.SourceAttr!.split(':').slice(2).join(':').split(',')[0];

  await app.goTo('Skills');
  const list = page.getByRole('region', { name: 'Socket groups' });
  await list.locator('button[aria-current]').nth(index).click();
  const editor = page.getByRole('region', { name: 'Selected group' });
  await expect(editor).toContainText('This is a special group created for the');
  await expect(editor).toContainText(`which is being provided by '${itemName}`);

  const count = editor.getByRole('spinbutton', { name: 'Count', exact: true });
  await count.fill('3');
  await count.press('Enter');
  await expect
    .poll(async () => (await go.loadXml(await go.decode(await app.exportCode()))).Skills.SkillSets[model.ActiveSkillSetIndex()].Skills[index].GroupCount)
    .toBe(3);
  app.assertNoErrors();
});
