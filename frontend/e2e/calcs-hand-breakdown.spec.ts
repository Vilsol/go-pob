// CalcOffence.lua:1942,1959

import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import { breakdownStep } from '../src/lib/design/breakdown_lines';

const FILE = '1291-gladiator-dual-strike-of-ambidexterity-3';
const PIN: { row: string; lines: string[] }[] = [
  { row: 'MH Att. per second', lines: ['0.67 ^8(base)', 'x 2.42 ^8(increased/reduced)', 'x 1.56 ^8(more/less)', '= 2.54 ^8casts per second'] },
  { row: 'OH Att. per second', lines: ['1.42 ^8(base)', 'x 3.32 ^8(increased/reduced)', 'x 1.56 ^8(more/less)', '= 7.36 ^8casts per second'] }
];

const plain = (s: string) => s.replace(/\^x[0-9A-Fa-f]{6}|\^\d/g, '').replace(/\s+/g, '');
const shownAs = (line: string) => {
  const st = breakdownStep(line);
  return 'prose' in st ? plain(st.prose) : plain((st.op ?? '') + st.value + (st.note ?? ''));
};

test('calcs: a dual-wield attack shows each hand its own breakdown', async ({ page }) => {
  test.setTimeout(5 * 60_000);
  const xml = buildXml(FILE);
  const build = await go.loadXml(xml);
  const app = await App.open(page, FILE);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));

  await app.goTo('Calcs');
  const calcs = page.getByRole('region', { name: 'Calculations' });
  for (const c of PIN) {
    await calcs.getByRole('button', { name: c.row, exact: true }).first().hover();
    const group = page.getByRole('group', { name: 'Calculation' }).first();
    await expect(group).toBeVisible();
    const shown = (await group.locator('[data-step], .col-span-3').allTextContents()).map(plain);
    expect(shown, `${c.row}: not the pin's lines`).toEqual(c.lines.map(shownAs));
    await page.mouse.move(0, 0);
  }
  app.assertNoErrors();
});
