import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import { optionControl, unconditionalConfigChecks } from './support/config_spec';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import { editDiff } from '../src/lib/display/stat_delta';

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
  const levelStep = `Set the character level to ${level1}`;
  await expect(undo(levelStep)).toBeEnabled();

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
  const name = check.name.replace(/[\s:?]+$/, '');
  const configStep = `${was ? 'Turned off' : 'Turned on'} ${name}`;
  await expect(undo(configStep)).toBeEnabled();

  await page.getByRole('button', { name: /^Undo history/ }).click();
  const menu = page.getByRole('menu', { name: 'Undo history' });
  await expect(menu.getByRole('menuitem')).toHaveText([
    new RegExp(`${configStep.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}.*Config`),
    new RegExp(`${levelStep}.*Tree`)
  ]);
  await page.keyboard.press('Escape');

  await app.act('undo', () => undo(configStep).click(), at(afterLevel));
  await expect(box, 'the Config page kept the undone value').toBeChecked({ checked: was });
  await app.act('undo', () => undo(levelStep).click(), at(start));
  await expect(page.getByRole('button', { name: 'Nothing to undo' })).toBeDisabled();

  await page.locator('body').click({ position: { x: 1, y: 1 } });
  await app.act('redo', () => page.keyboard.press('Control+Shift+Z'), at(afterLevel));
  await expect(undo(levelStep)).toBeEnabled();
  await expect(redo(configStep)).toBeEnabled();
  await app.openBuildControls();
  await expect(level, 'the Tree page kept the undone level').toHaveValue(String(level1));

  expect(await go.decode(await app.exportCode())).toBe(afterLevel);
  expect(afterConfig).not.toBe(afterLevel);
  app.assertNoErrors();
});

test('jump to a history entry from the keyboard: the build is Go’s after the same undo count, and each entry shows Go’s stat change', async ({ page }) => {
  const xml0 = buildXml(FILE);
  const build = await go.loadXml(xml0);
  const app = await App.open(page, FILE);
  await app.importCode(await go.encode(xml0), async () => expectedLines(await go.outputs(build)));
  const outs = [await go.outputs(build)];
  const saved: string[] = [];

  await app.openBuildControls();
  const level = page.getByRole('spinbutton', { name: 'Character level' });
  const level0 = build.Build.Level;
  const setLevel = async (to: number) => {
    await app.act(
      'set-level',
      () => level.fill(String(to)),
      async () => {
        // Build.lua:218-224
        build.SetCharacterLevelAutoMode(false);
        build.SetLevel(to);
        return expectedLines(await go.outputs(build));
      }
    );
    outs.push(await go.outputs(build));
    saved.push(await go.serializeSaved(build));
  };
  await setLevel(Math.max(1, level0 - 30));

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
  outs.push(await go.outputs(build));
  saved.push(await go.serializeSaved(build));

  await app.openBuildControls();
  await setLevel(Math.max(1, level0 - 10));

  const historyButton = page.getByRole('button', { name: /^Undo history/ });
  await historyButton.focus();
  await page.keyboard.press('ArrowDown');
  const menu = page.getByRole('menu', { name: 'Undo history' });
  const entries = menu.getByRole('menuitem');
  await expect(entries).toHaveCount(3);
  await expect(entries.nth(0)).toBeFocused();

  // newest first: entry i is the state after write 3 - i; its change is outs[2 - i] → outs[3 - i]
  const tip = page.getByRole('tooltip', { name: 'Stat change' });
  const shown = (before: go.ModelOutputs, after: go.ModelOutputs) => {
    const lines = editDiff(before, after).map((d) => `${d.section === 'Minion' ? 'Minion: ' : ''}${d.label}${d.value}`);
    return lines.length > 8 ? [...lines.slice(0, 8), `+${lines.length - 8} more`] : lines;
  };
  for (const i of [0, 1, 2]) {
    if (i) await page.keyboard.press('ArrowDown');
    await expect(entries.nth(i)).toBeFocused();
    const want = shown(outs[2 - i], outs[3 - i]);
    if (want.length) await expect(tip.getByRole('listitem'), `entry ${i}: Go’s change`).toHaveText(want);
    else await expect(tip, `entry ${i}: no headline stat changed`).toHaveText(/No headline stat changed/);
  }

  await app.act(
    'jump',
    () => page.keyboard.press('Enter'),
    async () => expectedLines(await go.outputs(await go.loadXml(saved[0])))
  );
  await expect(menu).toBeHidden();
  await expect(page.getByRole('button', { name: /^Redo: Turned (on|off) / })).toBeEnabled();
  expect(await go.decode(await app.exportCode()), 'the build after jumping two steps back is Go’s after two undos').toBe(saved[0]);
  app.assertNoErrors();
});
