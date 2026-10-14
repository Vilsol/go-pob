import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import { optionControl, unconditionalConfigChecks, unconditionalConfigCounts } from './support/config_spec';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import * as tree from './support/tree_model';
import { framesSince, installTreeProbe } from './support/tree_probe';

const open = async (page: import('@playwright/test').Page, file: string, xml = buildXml(file)) => {
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  const want = async () => expectedLines(await go.outputs(build));
  await app.importCode(await go.encode(xml), want);
  return { app, build, want };
};

test('import a build code: the sidebar shows what Go computes for it', async ({ page }) => {
  const { app } = await open(page, '0280-elementalist-fireball');
  app.assertNoErrors();
});

test('change the character level, up and back down', async ({ page }) => {
  const { app, build, want } = await open(page, '0516-reliquarian-kinetic-fusillade');
  await app.openBuildControls();
  const level = page.getByRole('spinbutton', { name: 'Character level' });
  const original = build.Build.Level;

  for (const n of [Math.max(1, original - 30), original]) {
    await app.act(
      'set-level',
      () => level.fill(String(n)),
      async () => {
        // Build.lua:218-224
        build.SetCharacterLevelAutoMode(false);
        build.SetLevel(n);
        return want();
      }
    );
    await expect(level, 'the level control snapped back to a stale value').toHaveValue(String(n));
  }
  app.assertNoErrors();
});

test('change class, then ascendancy', async ({ page }) => {
  const { app, build, want } = await open(page, '0037-ascendant-fire-trap-of-blasting-2');
  await app.openBuildControls();

  const m = await go.model();
  // Build.lua:263-275
  await page.getByRole('combobox', { name: 'Class', exact: true }).selectOption('Witch');
  const confirm = page.getByRole('dialog', { name: 'Class Change' });
  await expect(confirm).toContainText('Changing class to Witch will reset your passive tree.');
  await app.act(
    'set-class',
    () => confirm.getByRole('button', { name: 'Continue' }).click(),
    async () => {
      expect((await m.exposition.ChangeTreeClass(build, 'Witch', 1)).Changed).toBe(true);
      return want();
    }
  );
  await app.act(
    'set-ascendancy',
    () => page.getByRole('combobox', { name: 'Ascendancy' }).selectOption('Elementalist'),
    async () => {
      await m.exposition.ChangeTreeAscendancy(build, 'Elementalist');
      return want();
    }
  );
  await expect(page.getByRole('combobox', { name: 'Class', exact: true })).toHaveValue('Witch');
  await expect(page.getByRole('combobox', { name: 'Ascendancy' })).toHaveValue('Elementalist');
  app.assertNoErrors();
});

// Build.lua:285-299
// Build.lua:1093-1138, PassiveSpec.lua:619-655
test('choose a bloodline, then none', async ({ page }) => {
  const { app, build, want } = await open(page, '0037-ascendant-fire-trap-of-blasting-2');
  await app.openBuildControls();
  const m = await go.model();
  const select = page.getByRole('combobox', { name: 'Bloodline' });
  await expect(select).toBeEnabled();
  const labels = await select.locator('option').allTextContents();
  expect(labels[0]).toBe('None');
  expect(labels.length).toBeGreaterThan(1);
  expect(labels.slice(1)).toEqual([...labels.slice(1)].sort());
  const id = Number(await select.locator('option').nth(1).getAttribute('value'));
  await app.act(
    'set-bloodline',
    () => select.selectOption(String(id)),
    async () => {
      await m.exposition.ChangeSecondaryAscendancy(build, id);
      return want();
    }
  );
  await expect(select).toHaveValue(String(id));
  await app.act(
    'set-bloodline-none',
    () => select.selectOption('0'),
    async () => {
      await m.exposition.ChangeSecondaryAscendancy(build, 0);
      return want();
    }
  );
  await expect(select).toHaveValue('0');
  app.assertNoErrors();
});

test('change the main skill from the sidebar', async ({ page }) => {
  const { app, build, want } = await open(page, '0339-reliquarian-shield-crush-of-the-chieftain-2');
  await go.outputs(build);
  const set = build.ActiveSkillSetIndex();
  const labels = build.Skills.SkillSets[set].Skills.map((g) => g.DisplayLabel || g.Label || '<No active skills>');
  const main = build.Build.MainSocketGroup - 1;
  const target = labels.findIndex((l, i) => i !== main && l !== '<No active skills>' && labels.indexOf(l) === labels.lastIndexOf(l));
  expect(target, 'the build needs a second, uniquely-labelled socket group').toBeGreaterThanOrEqual(0);

  const select = page.getByRole('combobox', { name: 'Main skill' });
  await expect(select.getByRole('option')).toHaveCount(labels.length);
  await app.act(
    'set-main-skill',
    () => select.selectOption({ label: labels[target] }),
    async () => {
      build.SetMainSocketGroup(target + 1);
      return want();
    }
  );
  app.assertNoErrors();
});

test('toggle a config checkbox on and off', async ({ page }) => {
  const { app, build, want } = await open(page, '0444-reliquarian-static-strike-2');
  await app.goTo('Config');

  const checks = unconditionalConfigChecks();
  expect(checks.length).toBeGreaterThan(0);
  let toggled = 0;
  for (const c of checks.slice(0, 3)) {
    const box = optionControl(page, 'checkbox', c.name);
    await expect(box, `config option ${c.key} lost its accessible name`).toBeVisible();
    const was = await box.isChecked();
    for (const v of [!was, was]) {
      await app.act(
        'toggle-config',
        () => box.setChecked(v),
        async () => {
          go.setConfig(build, c.key, v);
          return want();
        }
      );
      toggled++;
    }
  }
  expect(toggled).toBe(6);
  app.assertNoErrors();
});

test('set a numeric config option', async ({ page }) => {
  const { app, build, want } = await open(page, '0009-trickster-cobra-lash');
  await app.goTo('Config');

  const counts = unconditionalConfigCounts();
  expect(counts.length).toBeGreaterThan(0);
  const c = counts[0];
  const box = optionControl(page, 'spinbutton', c.name);
  await expect(box, `config option ${c.key} lost its accessible name`).toBeVisible();
  const n = Number((await box.inputValue()) || c.min) + 3;
  await app.act(
    'set-config-number',
    async () => {
      await box.fill(String(n));
      await box.press('Enter');
    },
    async () => {
      go.setConfig(build, c.key, n);
      return want();
    }
  );
  app.assertNoErrors();
});

// SkillsTab.lua:1462-1464
test('the model edits gems in the active skill set, whatever position it has (0280)', async () => {
  const build = await go.loadXml(buildXml('0280-elementalist-fireball'));
  expect(
    go.fresh(build, (b) => [b.Skills.ActiveSkillSet, b.Skills.SkillSets.map((s) => s.ID)]),
    'vacuity: 0280 keeps set 1 active and last'
  ).toEqual([1, [2, 3, 4, 1]]);
  const level = (set: number) => go.fresh(build, (b) => b.Skills.SkillSets[set].Skills[0].Gems[0].Level);
  const [first, active] = [level(0), level(3)];
  go.setGemField(build, 0, 0, 'Level', active + 1);
  expect(level(3), 'the active set (id 1) was not edited').toBe(active + 1);
  expect(level(0), 'the first set in order (id 2) was edited').toBe(first);
});

test('edit a gem level on the skills page', async ({ page }) => {
  const { app, build, want } = await open(page, '0008-necromancer-vaal-summon-skeletons');
  await app.goTo('Skills');

  const set = build.ActiveSkillSetIndex();
  const gem = build.Skills.SkillSets[set].Skills[0].Gems[0];
  const name = (await go.model()).exposition.GetSkillGems()!.find((g) => g.ID === gem.GemID)!.Base.Name;
  const level = page.getByRole('region', { name: 'Selected group' }).getByRole('spinbutton', { name: `${name} level` });
  await expect(level).toHaveValue(String(gem.Level));

  const n = gem.Level > 1 ? gem.Level - 1 : gem.Level + 1;
  await app.act(
    'set-gem-level',
    async () => {
      await level.fill(String(n));
      await level.press('Enter');
    },
    async () => {
      go.setGemField(build, 0, 0, 'Level', n);
      return want();
    }
  );
  app.assertNoErrors();
});

test('on the skills page: set a main group, disable it, add a gem to it', async ({ page }) => {
  const { app, build, want } = await open(page, '0339-reliquarian-shield-crush-of-the-chieftain-2');
  await go.outputs(build);
  await app.goTo('Skills');
  const set = build.ActiveSkillSetIndex();
  const groups = () => build.Skills.SkillSets[set].Skills;
  const labels = groups().map((g) => g.DisplayLabel || g.Label || '<No active skills>');
  const main = build.Build.MainSocketGroup - 1;
  const target = labels.findIndex((l, i) => i !== main && l !== '<No active skills>' && labels.every((o, j) => j === i || !o.includes(l)));
  expect(target, 'the build needs a second, uniquely-labelled socket group').toBeGreaterThanOrEqual(0);

  const list = page.getByRole('region', { name: 'Socket groups' });
  const card = list.getByRole('button').filter({ hasText: labels[target] });
  await card.click();
  await expect(card).toHaveAttribute('aria-current', 'true');
  await app.act(
    'set-main-group',
    () => list.getByRole('button', { name: 'Set as main' }).click(),
    async () => {
      build.SetMainSocketGroup(target + 1);
      return want();
    }
  );
  await expect(card).toContainText('MAIN');

  const editor = page.getByRole('region', { name: 'Selected group' });
  await app.act(
    'disable-group',
    () => editor.getByRole('checkbox', { name: 'Enabled', exact: true }).click(),
    async () => {
      groups()[target].Enabled = false;
      return want();
    }
  );
  await app.act(
    'enable-group',
    () => editor.getByRole('checkbox', { name: 'Enabled', exact: true }).click(),
    async () => {
      groups()[target].Enabled = true;
      return want();
    }
  );

  const exposition = (await go.model()).exposition;
  const all = exposition.GetSkillGems()!;
  const support = all
    .filter((g) => g.Support && g.Name && !groups()[target].Gems.some((m) => m.GemID === g.ID))
    .filter((g) => all.filter((o) => o.Name?.toLowerCase().includes(g.Name.toLowerCase())).length === 1)
    .filter((g) => !all.some((o) => o.Tags?.includes(g.Name.toLowerCase())))
    .sort((a, b) => a.Name.localeCompare(b.Name))[0];
  expect(support, 'no support gem to add').toBeDefined();
  const search = editor.getByRole('combobox', { name: 'Add a gem' });
  await search.fill(support.Name);
  // SkillsTab.lua:768-800
  const level = exposition.GemDefaultLevel(build, support.ID) || 1;
  await app.act(
    'add-gem',
    () => search.press('Enter'),
    async () => {
      const gems = groups()[target].Gems.map((g) => ({ ...g }));
      gems.push(exposition.DefaultGem(support.ID, support.Base.Name, support.ID, level, go.fresh(build, (b) => b.Skills.DefaultGemQuality) ?? 0));
      build.SetSocketGroupGems(set, target, gems);
      return want();
    }
  );
  await expect(editor.getByRole('spinbutton', { name: `${support.Base.Name} level` })).toHaveValue(String(level));
  app.assertNoErrors();
});

test('on the skills page: drag a socket group down, switch the skill set and back', async ({ page }) => {
  const { app, build, want } = await open(page, '0339-reliquarian-shield-crush-of-the-chieftain-2');
  await go.outputs(build);
  await app.goTo('Skills');
  const list = page.getByRole('region', { name: 'Socket groups' });
  const rows = list.locator('[data-reorder-item]');
  const groups = () => build.Skills.SkillSets[build.ActiveSkillSetIndex()].Skills;
  const mainIndex = () => go.fresh(build, (b) => b.Build.MainSocketGroup) - 1;
  const mainLabel = () => {
    const g = groups()[mainIndex()];
    return g.DisplayLabel || g.Label || '<No active skills>';
  };
  await expect(rows, 'the group cards drag').toHaveCount(groups().length);

  // SkillListControl.lua:125-126
  const main = mainIndex();
  const from = main < groups().length - 1 ? main : main - 1;
  const label = mainLabel();
  await app.act(
    'move-socket-group',
    async () => {
      await rows.nth(from).focus();
      await rows.nth(from).press('Alt+ArrowDown');
    },
    async () => {
      build.MoveSocketGroup(from, from + 1);
      return want();
    }
  );
  expect(mainLabel(), 'the model kept the main group on its group').toBe(label);
  await expect(list.locator('button[aria-current]').filter({ hasText: 'MAIN' })).toContainText(label);
  await expect(list.locator('button[aria-current]').nth(from + 1), 'the main card moved down with its group').toContainText('MAIN');

  const select = list.getByRole('combobox', { name: 'Skill set' });
  await expect(select).toBeEnabled();
  await app.act(
    'switch-skill-set',
    () => select.selectOption({ label: 'enmity2' }),
    async () => {
      build.SetActiveSkillSet(3);
      return want();
    }
  );
  await expect(select).toHaveValue('2');
  await app.act(
    'switch-skill-set-back',
    () => select.selectOption({ index: 0 }),
    async () => {
      build.SetActiveSkillSet(1);
      return want();
    }
  );
  await expect(select).toHaveValue('0');
  app.assertNoErrors();
});

test('save a build: the build list shows its ascendancy and main skill', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const { app, build } = await open(page, file);
  const m = await go.model();
  const want = m.builds.SavedBuildSummary(await go.serializeSaved(build)).unwrap();
  expect(want.Ascendancy && want.MainSkill && want.MainSkill !== '<No active skills>', `vacuity: ${JSON.stringify(want)}`).toBeTruthy();
  const name = `e2e-list-columns-${file}`;
  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('textbox', { name: 'Build name' }).fill(name);
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();

  await page.reload();
  const row = app.savedBuild(name);
  await row.waitFor({ timeout: 180_000 });
  await expect(row).toContainText(want.Ascendancy);
  await expect(row).toContainText(want.MainSkill);
  await expect(row).toContainText(String(want.Level));
  app.assertNoErrors();
});

test('export the code, import it back: Go holds the identical build', async ({ page }) => {
  const file = '0977-luminary-purity-of-ice-2';
  const { app, build, want } = await open(page, file);
  const code = await app.exportCode();
  expect(go.withoutView(await go.decode(code)), 'exported build differs from the imported one').toBe(go.withoutView(await go.serializeSaved(build)));

  await app.importCode(code, want);
  expect(go.withoutView(await go.decode(await app.exportCode())), 're-exported build differs').toBe(go.withoutView(await go.serializeSaved(build)));
  app.assertNoErrors();
});

test('allocate a passive node from the tree search, then deallocate it', async ({ page }) => {
  const { app, build, want } = await open(page, '0339-reliquarian-shield-crush-of-the-chieftain-2');
  await tree.prepareTree();
  await app.openTree(tree.treeVersion());

  const before = tree.treeState(build);
  const held = before.allocated()[0];
  await expect(await app.treeNode(tree.resultName(held)), 'an allocated node does not read as allocated').toHaveAttribute('aria-pressed', 'true');

  const target = before.adjacent()[0];
  expect(target, 'no unallocated node next to the allocated tree').toBeDefined();
  const button = await app.treeNode(tree.resultName(target));
  await expect(button).toHaveAttribute('aria-pressed', 'false');

  await app.act(
    'allocate-node',
    () => button.click(),
    async () => {
      before.allocate(target.id);
      return want();
    }
  );
  await expect(button, 'the allocated node still reads as unallocated').toHaveAttribute('aria-pressed', 'true');
  expect(
    go.fresh(build, (b) => b.Build.PassiveNodes),
    'the model did not allocate the node'
  ).toContain(target.id);
  expect(go.withoutView(await go.decode(await app.exportCode())), 'after allocating, the build Go holds differs from the model').toBe(
    go.withoutView(await go.serializeSaved(build))
  );
  await app.openTree(tree.treeVersion());
  await expect(await app.treeNode(tree.resultName(target)), 'the reopened tree lost the allocation').toHaveAttribute('aria-pressed', 'true');

  const after = tree.treeState(build, await tree.clusterOf(build));
  await app.act(
    'deallocate-node',
    () => button.click(),
    async () => {
      after.deallocate(target.id);
      return want();
    }
  );
  await expect(button, 'the deallocated node still reads as allocated').toHaveAttribute('aria-pressed', 'false');

  expect(go.withoutView(await go.decode(await app.exportCode())), 'the build Go holds differs from the model').toBe(
    go.withoutView(await go.serializeSaved(build))
  );
  app.assertNoErrors();
});

const recordingTree = (xml: string, version: string) => {
  const active = Number(xml.match(/<Tree\b[^>]*activeSpec="(\d+)"/)?.[1] ?? 1);
  let n = 0;
  return xml.replace(/<Spec\b[^>]*>/g, (spec) => (++n === active ? spec.replace(/treeVersion="[^"]*"/, `treeVersion="${version}"`) : spec));
};

test('the tree draws the tree the build is calculated on, whatever version the build records', async ({ page }) => {
  await installTreeProbe(page);
  await tree.prepareTree();
  const version = tree.treeVersion();
  const shown = `Tree ${version.replace('_', '.')}`;
  const label = page.getByLabel('Tree version');

  const file = '0280-elementalist-fireball';
  const older = recordingTree(buildXml(file), '3_28');
  const { app, build, want } = await open(page, file, older);
  expect(
    go.fresh(build, (b) => b.Tree.Specs[b.Tree.ActiveSpec - 1].TreeVersion),
    'the rewritten build does not record the older version'
  ).toBe('3_28');
  await app.openTree(version);
  await expect(label).toHaveText(`${shown} (build records 3.28)`);

  const state = tree.treeState(build);
  const target = state.adjacent()[0];
  expect(target, 'no unallocated node next to the allocated tree').toBeDefined();
  const button = await app.treeNode(tree.resultName(target));
  await app.act(
    'allocate-node',
    () => button.click(),
    async () => {
      state.allocate(target.id);
      return want();
    }
  );
  await expect(button, 'the allocated node still reads as unallocated').toHaveAttribute('aria-pressed', 'true');
  const after = tree.treeState(build, await tree.clusterOf(build));
  await app.act(
    'deallocate-node',
    () => button.click(),
    async () => {
      after.deallocate(target.id);
      return want();
    }
  );
  await expect(button, 'the deallocated node still reads as allocated').toHaveAttribute('aria-pressed', 'false');

  const current = buildXml('0339-reliquarian-shield-crush-of-the-chieftain-2');
  const next = await go.loadXml(current);
  await app.importCode(await go.encode(current), async () => expectedLines(await go.outputs(next)));
  await app.openTree(version);
  await expect(label).toHaveText(shown);

  const frames = await framesSince(page, 0);
  expect(frames.length, 'the tree drew no frame').toBeGreaterThan(0);
  expect([...new Set(frames.map((f) => f.version))], 'a frame was drawn from another tree version').toEqual([version]);
  app.assertNoErrors();
});
