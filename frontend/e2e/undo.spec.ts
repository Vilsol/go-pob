import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import { optionControl, unconditionalConfigChecks } from './support/config_spec';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

const FILE = '0516-reliquarian-kinetic-fusillade';

test('change something on two pages, undo both in order, redo one', async ({ page }) => {
  const xml0 = buildXml(FILE);
  const build = await go.loadXml(xml0);
  const app = await App.open(page, FILE);
  const at = (xml: string) => async () => expectedLines(await go.outputs(await go.loadXml(xml)));
  await app.importCode(await go.encode(xml0), at(xml0));

  const undo = (name: string) => page.getByRole('button', { name: `Undo: ${name} (Ctrl+Z)` });
  const redo = (name: string) => page.getByRole('button', { name: `Redo: ${name} (Ctrl+Y / Ctrl+Shift+Z)` });
  await expect(page.getByRole('button', { name: 'Nothing to undo' })).toBeDisabled();

  await app.openBuildControls();
  const level = page.getByRole('spinbutton', { name: 'Character level' });
  const level0 = build.Build.Level;
  const level1 = Math.max(1, level0 - 30);
  const start = await go.serialize(build);
  await app.act(
    'set-level',
    () => level.fill(String(level1)),
    async () => {
      // Build.lua:218-224
      build.SetCharacterLevelAutoMode(false);
      build.SetLevel(level1);
      return expectedLines(await go.outputs(build));
    }
  );
  const afterLevel = await go.serializeSaved(build);
  await expect(undo('change level')).toBeEnabled();

  await app.goTo('Config');
  const check = unconditionalConfigChecks()[0];
  const box = optionControl(page, 'checkbox', check.name);
  const was = await box.isChecked();
  await app.act(
    'toggle-config',
    () => box.setChecked(!was),
    async () => {
      go.setConfig(build, check.key, !was);
      return expectedLines(await go.outputs(build));
    }
  );
  const afterConfig = await go.serializeSaved(build);
  await expect(undo('change a config option')).toBeEnabled();

  await page.getByRole('button', { name: /^Undo history/ }).click();
  const menu = page.getByRole('menu', { name: 'Undo history' });
  await expect(menu.getByRole('menuitem')).toHaveText([/change a config option.*Config/, /change level.*Tree/]);
  await page.keyboard.press('Escape');

  await app.act('undo', () => undo('change a config option').click(), at(afterLevel));
  await expect(box, 'the Config page kept the undone value').toBeChecked({ checked: was });
  await app.act('undo', () => undo('change level').click(), at(start));
  await expect(page.getByRole('button', { name: 'Nothing to undo' })).toBeDisabled();

  await page.locator('body').click({ position: { x: 1, y: 1 } });
  await app.act('redo', () => page.keyboard.press('Control+Shift+Z'), at(afterLevel));
  await expect(undo('change level')).toBeEnabled();
  await expect(redo('change a config option')).toBeEnabled();
  await app.openBuildControls();
  await expect(level, 'the Tree page kept the undone level').toHaveValue(String(level1));

  expect(await go.decode(await app.exportCode())).toBe(afterLevel);
  expect(afterConfig).not.toBe(afterLevel);
  app.assertNoErrors();
});
