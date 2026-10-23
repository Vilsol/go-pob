// Build.lua:486-561

import { type Page } from '@playwright/test';
import { test, expect } from './support/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import type * as Types from '../src/lib/types';

type Build = Types.pob.PathOfBuilding;

const open = async (page: Page, file: string) => {
  const build = await go.loadXml(buildXml(file));
  const app = await App.open(page, file);
  const want = async () => expectedLines(await go.outputs(build));
  await app.importCode(await go.encode(buildXml(file)), want);
  return { app, build, want };
};

const controls = async (build: Build) => {
  const c = (await go.outputs(build)).MainSkill;
  expect(c, 'the model has no main-skill controls for this build').toBeDefined();
  return c!;
};

const expectSelected = async (page: Page, name: string, labels: string[] | undefined, index: number) => {
  const select = page.getByRole('combobox', { name, exact: true });
  await expect(select.getByRole('option')).toHaveText(labels ?? []);
  await expect(select, `${name} shows another selection than Go holds`).toHaveValue(String(index));
};

const saveAndReload = async (app: App, build: Build, want: () => Promise<{ label: string; value: string }[]>) => {
  const code = await app.exportCode();
  expect(go.withoutView(await go.decode(code)), 'the exported build differs from the model').toBe(go.withoutView(await go.serializeSaved(build)));
  await app.importCode(code, want);
};

test('multi-part skill with stages: switch the part, set the stages, survive save/reload', async ({ page }) => {
  const { app, build, want } = await open(page, '0021-assassin-blade-flurry');
  const c = await controls(build);
  expect(c.PartShown && c.StageShown, 'the build needs a multi-part staged main skill').toBe(true);
  await expectSelected(page, 'Skill part', c.Parts, c.SkillPart);

  const part = c.SkillPart === 1 ? 2 : 1;
  await app.act(
    'set-skill-part',
    () => page.getByRole('combobox', { name: 'Skill part', exact: true }).selectOption(String(part)),
    async () => {
      build.SetGemSkillPart(c.SkillSet, c.SocketGroup, c.GemIndex, part);
      return want();
    }
  );
  await expectSelected(page, 'Skill part', c.Parts, part);

  await app.act(
    'set-skill-part',
    () => page.getByRole('combobox', { name: 'Skill part', exact: true }).selectOption(String(c.SkillPart)),
    async () => {
      build.SetGemSkillPart(c.SkillSet, c.SocketGroup, c.GemIndex, c.SkillPart);
      return want();
    }
  );
  const stages = page.getByRole('textbox', { name: 'Stages' });
  await expect(stages).toHaveValue(c.StageCount);
  await app.act(
    'set-stage-count',
    () => stages.fill('3'),
    async () => {
      build.SetGemSkillStageCount(c.SkillSet, c.SocketGroup, c.GemIndex, 3);
      return want();
    }
  );

  await saveAndReload(app, build, want);
  await expectSelected(page, 'Skill part', c.Parts, c.SkillPart);
  await expect(page.getByRole('textbox', { name: 'Stages' })).toHaveValue('3');
  app.assertNoErrors();
});

test('mine skill: set the active mine count, clear it, survive save/reload', async ({ page }) => {
  const { app, build, want } = await open(page, '0344-assassin-icicle-mine-of-fanning');
  const c = await controls(build);
  expect(c.MineShown, 'the build needs a mine main skill').toBe(true);
  const mines = page.getByRole('textbox', { name: 'Active mines' });
  await expect(mines).toHaveValue(c.MineCount);

  await app.act(
    'set-mine-count',
    () => mines.fill('2'),
    async () => {
      build.SetGemSkillMineCount(c.SkillSet, c.SocketGroup, c.GemIndex, 2);
      return want();
    }
  );
  await app.act(
    'clear-mine-count',
    () => mines.fill(''),
    async () => {
      build.SetGemSkillMineCount(c.SkillSet, c.SocketGroup, c.GemIndex, -1);
      return want();
    }
  );
  await app.act(
    'set-mine-count',
    () => mines.fill('1'),
    async () => {
      build.SetGemSkillMineCount(c.SkillSet, c.SocketGroup, c.GemIndex, 1);
      return want();
    }
  );

  await saveAndReload(app, build, want);
  await expect(page.getByRole('textbox', { name: 'Active mines' })).toHaveValue('1');
  app.assertNoErrors();
});

test('spectres: switch the active skill, the spectre and its skill, survive save/reload', async ({ page }) => {
  const { app, build, want } = await open(page, '0031-guardian-raise-spectre');
  let c = await controls(build);
  expect(c.MinionShown && c.MinionSkillShown && (c.Minions?.length ?? 0) > 1, 'the build needs a spectre main skill with several spectres').toBe(true);
  await expectSelected(page, 'Active skill', c.Skills, c.MainActiveSkill);
  await expectSelected(
    page,
    'Minion',
    c.Minions?.map((m) => m.Label),
    c.Minion
  );
  await expectSelected(page, 'Minion skill', c.MinionSkills, c.MinionSkill);

  const minion = c.Minion === 1 ? 2 : 1;
  const minionID = c.Minions![minion - 1].MinionID;
  await app.act(
    'set-minion',
    () => page.getByRole('combobox', { name: 'Minion', exact: true }).selectOption(String(minion)),
    async () => {
      build.SetGemSkillMinion(c.SkillSet, c.SocketGroup, c.GemIndex, minionID);
      return want();
    }
  );
  c = await controls(build);
  await expectSelected(
    page,
    'Minion',
    c.Minions?.map((m) => m.Label),
    minion
  );
  await expectSelected(page, 'Minion skill', c.MinionSkills, c.MinionSkill);

  const minionSkill = (c.MinionSkills?.length ?? 0) > 1 ? 2 : 1;
  await app.act(
    'set-minion-skill',
    () => page.getByRole('combobox', { name: 'Minion skill', exact: true }).selectOption(String(minionSkill)),
    async () => {
      build.SetGemSkillMinionSkill(c.SkillSet, c.SocketGroup, c.GemIndex, minionSkill);
      return want();
    }
  );

  await saveAndReload(app, build, want);
  c = await controls(build);
  await expectSelected(
    page,
    'Minion',
    c.Minions?.map((m) => m.Label),
    minion
  );
  await expectSelected(page, 'Minion skill', c.MinionSkills, minionSkill);

  const other = c.MainActiveSkill === 1 ? 2 : 1;
  await app.act(
    'set-active-skill',
    () => page.getByRole('combobox', { name: 'Active skill', exact: true }).selectOption(String(other)),
    async () => {
      build.SetMainActiveSkill(c.SkillSet, c.SocketGroup, other);
      return want();
    }
  );
  await expectSelected(page, 'Active skill', c.Skills, other);
  app.assertNoErrors();
});

test('minion using items: pick another item set, survive save/reload', async ({ page }) => {
  const { app, build, want } = await open(page, '0342-luminary-animate-guardian');
  const c = await controls(build);
  const sets = c.Minions ?? [];
  expect(sets.length > 1 && sets.every((m) => m.ItemSetID > 0), 'the build needs a minion using item sets').toBe(true);

  const pick = c.Minion === 1 ? 2 : 1;
  await app.act(
    'set-minion-item-set',
    () => page.getByRole('combobox', { name: 'Minion', exact: true }).selectOption(String(pick)),
    async () => {
      build.SetGemSkillMinionItemSet(c.SkillSet, c.SocketGroup, c.GemIndex, sets[pick - 1].ItemSetID);
      return want();
    }
  );

  await saveAndReload(app, build, want);
  await expectSelected(
    page,
    'Minion',
    sets.map((m) => m.Label),
    pick
  );
  app.assertNoErrors();
});

test('point counter: shows what Go counts', async ({ page }) => {
  const { app, build } = await open(page, '0280-elementalist-fireball');
  await app.openBuildControls();
  const [used, ascUsed] = (await go.outputs(build)).PointsUsed;
  const counter = page.getByRole('status', { name: 'Points used' });
  await expect(counter).toContainText(`${used} /`);
  await expect(counter).toContainText(`${ascUsed} / 8`);
});

for (const file of ['0001-necromancer-unknown']) {
  test(`heavy build ${file}: flip every enabled selector and back`, async ({ page }) => {
    const { app, build, want } = await open(page, file);
    const c = await controls(build);
    type Flip = { action: string; name: string; from: number; apply: (v: number) => void };
    const flips: Flip[] = [];
    if (c.SkillEnabled) {
      flips.push({
        action: 'heavy-set-active-skill',
        name: 'Active skill',
        from: c.MainActiveSkill,
        apply: (v) => build.SetMainActiveSkill(c.SkillSet, c.SocketGroup, v)
      });
    }
    if (c.PartShown) {
      flips.push({
        action: 'heavy-set-skill-part',
        name: 'Skill part',
        from: c.SkillPart,
        apply: (v) => build.SetGemSkillPart(c.SkillSet, c.SocketGroup, c.GemIndex, v)
      });
    }
    if (c.MinionSkillShown && c.MinionSkillEnabled) {
      flips.push({
        action: 'heavy-set-minion-skill',
        name: 'Minion skill',
        from: c.MinionSkill,
        apply: (v) => build.SetGemSkillMinionSkill(c.SkillSet, c.SocketGroup, c.GemIndex, v)
      });
    }
    expect(flips.length, `${file} has no enabled main-skill selector to time`).toBeGreaterThan(0);

    for (const f of flips) {
      for (const v of [f.from === 1 ? 2 : 1, f.from]) {
        await app.act(
          f.action,
          () => page.getByRole('combobox', { name: f.name, exact: true }).selectOption(String(v)),
          async () => {
            f.apply(v);
            return want();
          }
        );
      }
    }
    app.assertNoErrors();
  });
}
