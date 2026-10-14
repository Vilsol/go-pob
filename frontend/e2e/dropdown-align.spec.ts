import { test, expect, type Locator, type Page } from '@playwright/test';

import { App } from './support/app';
import { boxes, type Box } from './support/align';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { setFontScaling } from './support/overflow';
import { expectedLines } from './support/stats';

const BUILD = '0280-elementalist-fireball';

const center = (b: Box) => (b.top + b.bottom) / 2;
const describe = (got: Box[]) => got.map((b) => `${b.what} top ${b.top.toFixed(2)} bottom ${b.bottom.toFixed(2)} baseline ${b.baseline.toFixed(2)}`).join('\n');

const expectCentred = async (where: string, controls: Locator[]) => {
  const got = await boxes(controls);
  for (const b of got.slice(1)) expect.soft(Math.abs(center(b) - center(got[0])), `${where}: ${b.what} is off centre\n${describe(got)}`).toBeLessThanOrEqual(1);
};

const expectSameBox = async (where: string, dropdown: Locator, reference: Locator) => {
  const [d, r] = await boxes([dropdown, reference]);
  const msg = `${where}\n${describe([d, r])}`;
  expect.soft(Math.abs(d.top - r.top), msg).toBeLessThanOrEqual(1);
  expect.soft(Math.abs(d.bottom - r.bottom), msg).toBeLessThanOrEqual(1);
  expect.soft(Math.abs(d.baseline - r.baseline), msg).toBeLessThanOrEqual(1);
};

const open = async (page: Page) => {
  const xml = buildXml(BUILD);
  const build = await go.loadXml(xml);
  const app = await App.open(page, BUILD);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  return app;
};

test('a Dropdown and a select line up with the controls beside them (0280)', async ({ page }) => {
  test.setTimeout(180_000);
  const app = await open(page);
  for (const pt of [12, 18]) {
    await setFontScaling(page, pt);

    const loadout = page.getByRole('combobox', { name: /^Loadout/ });
    await expectCentred(`Top bar ${pt}pt`, [
      loadout,
      loadout.locator('xpath=ancestor::*[contains(concat(" ", @class, " "), " items-center ")][1]').getByRole('button').first()
    ]);

    await app.goTo('Tree');
    const search = page.getByRole('searchbox', { name: 'Search the tree' });
    await expect(search).toBeVisible();
    const filter = page.getByRole('combobox', { name: /^Filter/ });
    const set = page.getByRole('combobox', { name: /^Tree set/ });
    await expectCentred(`Tree ${pt}pt`, [search, filter, set, page.getByRole('button', { name: 'Reset Tree/Tattoos' })]);
    await expectSameBox(`Tree ${pt}pt: Tree set vs Filter`, set, filter);

    await app.goTo('Skills');
    const group = page.getByRole('region', { name: 'Selected group' });
    await expect(group).toBeVisible();
    await expectSameBox(`Skills ${pt}pt`, group.getByRole('combobox', { name: /^Socketed in/ }), group.getByRole('textbox', { name: /^Label/ }));

    await app.goTo('Config');
    const row = page
      .locator('.option-row')
      .filter({ has: page.getByRole('combobox') })
      .first();
    const dropdown = row.locator('button[role=combobox]');
    await row.evaluate(
      (el, label) => {
        el.querySelector('[data-align-probe]')?.remove();
        const s = document.createElement('select');
        s.className = 'select input-xs max-w-37.5 shrink-0';
        s.dataset.alignProbe = '';
        s.append(new Option(label));
        el.append(s);
      },
      (await dropdown.textContent())!.trim()
    );
    await expectSameBox(`Config ${pt}pt`, dropdown, row.locator('[data-align-probe]'));
    await row.evaluate((el) => el.querySelector('[data-align-probe]')?.remove());
  }
  await setFontScaling(page, 12);
  app.assertNoErrors();
});
