// SkillsTab.lua:948-960

import { test, expect } from './support/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedBlock, type DiffBlock, type WhatIfOutputs } from './support/stat_diff';
import { expectedLines } from './support/stats';

test('skills: hovering a gem’s enable checkbox shows what switching it would change', async ({ page }) => {
  test.setTimeout(5 * 60_000);
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Skills');

  const set = build.ActiveSkillSetIndex();
  const gem = build.Skills.SkillSets[set].Skills[0].Gems[0];
  const m = await go.model();
  const name = m.exposition.GetSkillGems()!.find((g) => g.ID === gem.GemID)!.Base.Name;
  await page.getByRole('region', { name: 'Socket groups' }).locator('button[aria-current]').first().click();
  const editor = page.getByRole('region', { name: 'Selected group' });
  await editor.getByRole('checkbox', { name: `${name} enabled` }).hover();

  const header = gem.Enabled ? 'Disabling this gem will give you:' : 'Enabling this gem will give you:';
  const status = editor.getByRole('status', { name: 'What-if' });
  await expect(status.locator('[data-header]')).toHaveAttribute('data-header', header, { timeout: 60_000 });
  const shown = await status.evaluate((el) => {
    const text = (e: Element | undefined) => (e?.textContent ?? '').replace(/\s+/g, ' ').trim();
    return [...el.querySelectorAll<HTMLElement>('[data-header]')].map((b) => ({
      header: b.dataset.header!,
      sections: [...b.querySelectorAll<HTMLElement>('[data-section]')].map((sec) => ({
        actor: sec.dataset.actor || undefined,
        lines: [...sec.querySelectorAll('[data-line]')].map((l) => {
          const r = /^(?:[▲▼] )?(\S+)(?: (.*))?$/.exec(text(l.children[1]))!;
          return `${r[1]} ${text(l.children[0])}${r[2] ? ` ${r[2]}` : ''}`;
        })
      }))
    }));
  });

  const misc = (await m.exposition.GetMiscCalculator(build))!;
  let want: DiffBlock;
  try {
    const out = (await m.exposition.CalcGemStats(misc, 0, 0, { ...gem, Enabled: !gem.Enabled })) as WhatIfOutputs;
    want = expectedBlock(header, misc.Base() as WhatIfOutputs, out) ?? { header, sections: [] };
  } finally {
    misc.release();
  }
  expect(want.sections.length, 'switching the main gem changes nothing in the model').toBeGreaterThan(0);
  expect(shown).toEqual([want]);
  app.assertNoErrors();
});
