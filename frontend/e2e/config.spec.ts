import { test, expect, type Page, type Locator } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import { optionControl, unconditionalConfigChecks } from './support/config_spec';
import * as go from './support/go_model';
import { record } from './support/perf';
import { expectedLines } from './support/stats';
import { cellsToText, textToCells } from '../src/lib/display/custom_mods';
import type * as Types from '../src/lib/types';

type Build = Types.pob.PathOfBuilding;

const BUILDS = ['0008-necromancer-vaal-summon-skeletons', '0009-trickster-cobra-lash', '0280-elementalist-fireball'];

const LIFE_LINE = '+1000 to maximum Life';
const GARBAGE_LINE = 'this line is not a modifier at all';

const open = async (page: Page, file: string) => {
  const build = await go.loadXml(buildXml(file));
  const app = await App.open(page, file);
  const want = async () => expectedLines(await go.outputs(build));
  await app.importCode(await go.encode(buildXml(file)), want);
  await app.goTo('Config');
  return { app, build, want };
};

const appendToModel = (build: Build, line: string) => {
  const blocks = build.GetCustomModifierBlocks()!.map((b) => ({ Title: b.Title ?? 'Default', Enabled: b.Enabled ?? 'true', Text: b.Text }));
  const cells = textToCells(blocks[0].Text);
  if (cells.cells.length === 1 && cells.cells[0] === '') cells.cells = [line];
  else cells.cells.push(line);
  blocks[0] = { ...blocks[0], Text: cellsToText(cells) };
  build.SetCustomModifierBlocks(blocks);
};

const customPanel = (page: Page) => page.getByRole('region', { name: 'Custom modifiers' });

const nextCell = async (page: Page): Promise<Locator> => {
  const cells = customPanel(page).getByRole('textbox', { name: /^Modifier line/ });
  const last = cells.last();
  if ((await last.inputValue()) === '') return last;
  await customPanel(page).getByRole('button', { name: '+ Add modifier' }).click();
  return cells.last();
};

const typeLine = async (app: App, cell: Locator, line: string, verdict: 'ok' | 'bad') => {
  const t0 = await app.now();
  await cell.fill(line);
  await expect(cell.locator('xpath=..')).toHaveAttribute('data-k', verdict);
  record({ kind: 'interaction', build: app.build, phase: 'config', name: 'custom-mod-live-verdict', durationMs: +((await app.now()) - t0).toFixed(1) });
};

const totalModified = async (page: Page) =>
  Number(/· (\d+) modified ·/.exec(await page.getByText(/^\d+ shown · \d+ modified · \d+ in total$/).innerText())![1]);

const sectionOptions = (page: Page) => page.locator('section[aria-label]:not([aria-label="Most used"]) label[data-var]');

for (const file of BUILDS) {
  test(`custom modifier cells and option markers on ${file}`, async ({ page }) => {
    const { app, build, want } = await open(page, file);

    const lifeBefore = (await go.outputs(build)).Output.Life;
    const cell = await nextCell(page);
    await typeLine(app, cell, LIFE_LINE, 'ok');
    await app.act(
      'config-add-custom-mod',
      () => cell.press('Enter'),
      async () => {
        appendToModel(build, LIFE_LINE);
        return want();
      }
    );
    expect((await go.outputs(build)).Output.Life, 'the model did not move Life, so the check above is vacuous').toBeGreaterThan(lifeBefore);

    const before = (await go.outputs(build)).Output;
    const junk = await nextCell(page);
    await typeLine(app, junk, GARBAGE_LINE, 'bad');
    await app.act(
      'config-add-garbage-mod',
      () => junk.press('Enter'),
      async () => {
        appendToModel(build, GARBAGE_LINE);
        return want();
      }
    );
    expect((await go.outputs(build)).Output).toEqual(before);

    expect(go.withoutView(await go.decode(await app.exportCode())), 'the exported build differs from the model').toBe(
      go.withoutView(await go.serializeSaved(build))
    );
    await app.goTo('Config');

    const c = unconditionalConfigChecks()[0];
    const box = optionControl(page, 'checkbox', c.name);
    const was = await box.isChecked();
    const count = await totalModified(page);
    await app.act(
      'toggle-config',
      () => box.setChecked(!was),
      async () => {
        go.setConfig(build, c.key, !was);
        return want();
      }
    );
    const nowModified = !was !== c.defaultState;
    if (nowModified) await expect(box).toHaveAccessibleName(/\(changed\)$/);
    else await expect(box).not.toHaveAccessibleName(/\(changed\)$/);
    await expect.poll(() => totalModified(page)).toBe(count + (nowModified ? 1 : -1));
    await expect(page.getByRole('region', { name: 'General' })).toBeVisible();

    await page.getByRole('checkbox', { name: 'Modified only' }).check();
    const shown = sectionOptions(page);
    const n = await shown.count();
    expect(n, 'Modified only shows as many options as are counted').toBe(await totalModified(page));
    for (let i = 0; i < n; i++) await expect(shown.nth(i)).toContainText('(changed)');
    await page.getByRole('checkbox', { name: 'Modified only' }).uncheck();

    if ((await totalModified(page)) > 0) {
      await page.getByRole('button', { name: 'Reset to defaults' }).click();
      await app.act(
        'config-reset',
        () => page.getByRole('button', { name: 'Confirm reset' }).click(),
        async () => {
          build.ResetConfigOptions();
          return want();
        }
      );
      await expect.poll(() => totalModified(page)).toBe(0);
    }
    app.assertNoErrors();
  });
}

test('switch the config set and back (0063)', async ({ page }) => {
  const { app, build, want } = await open(page, '0063-occultist-wintertide-brand');
  await page.getByRole('checkbox', { name: 'Show options that don’t affect this build' }).check();
  const select = page.getByRole('combobox', { name: 'Config set' });
  const surge = page.locator('[data-var="buffArcaneSurge"]').last().getByRole('checkbox');
  await expect(select).toBeEnabled();
  await expect(select).toHaveValue('0');
  await expect(surge).toBeChecked();

  await app.act(
    'switch-config-set',
    () => select.selectOption({ label: 'Jewel swap' }),
    async () => {
      build.SetActiveConfigSet(2);
      return want();
    }
  );
  await expect(select).toHaveValue('1');
  await expect(surge).not.toBeChecked();

  await app.act(
    'switch-config-set-back',
    () => select.selectOption({ label: 'Default' }),
    async () => {
      build.SetActiveConfigSet(1);
      return want();
    }
  );
  await expect(select).toHaveValue('0');
  await expect(surge).toBeChecked();
  app.assertNoErrors();
});

test('the Mod Browser finds a misspelt search (0280)', async ({ page }) => {
  const { app } = await open(page, '0280-elementalist-fireball');
  await page.getByRole('button', { name: 'Add Mod', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Mod Browser' });
  await dialog.getByRole('searchbox', { name: 'Search modifiers' }).fill('lightnign res');
  const first = dialog.getByRole('option').first();
  await expect(first).toContainText('Lightning');
  await expect(dialog.getByRole('option', { name: /Lightning Resistance/ }).first()).toBeVisible();
  app.assertNoErrors();
});
