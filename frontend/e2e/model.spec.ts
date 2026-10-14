import { test, expect } from '@playwright/test';
import fc from 'fast-check';

import { App } from './support/app';
import { buildXml } from './support/builds';
import { optionControl, unconditionalConfigChecks } from './support/config_spec';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import * as tree from './support/tree_model';

type Build = Awaited<ReturnType<typeof go.loadXml>>;
type Model = { build: Build };
type Real = { app: App };

const SEQUENCES = Number(process.env.SEQUENCES || 6);
const SEED = Number(process.env.SEED || 20260926);

const ASCENDANCIES: Record<string, string[]> = {
  Scion: ['Ascendant'],
  Marauder: ['Juggernaut', 'Berserker', 'Chieftain'],
  Ranger: ['Raider', 'Deadeye', 'Pathfinder'],
  Witch: ['Occultist', 'Elementalist', 'Necromancer'],
  Duelist: ['Slayer', 'Gladiator', 'Champion'],
  Templar: ['Inquisitor', 'Hierophant', 'Guardian'],
  Shadow: ['Assassin', 'Trickster', 'Saboteur']
};
const CHECKS = unconditionalConfigChecks().slice(0, 8);

let executed = 0;
const want = (m: Model) => async () => {
  executed++;
  return expectedLines(await go.outputs(m.build));
};

class SetLevel implements fc.AsyncCommand<Model, Real> {
  constructor(readonly n: number) {}
  check = (m: Readonly<Model>) => go.fresh(m.build, (b) => b.Build.Level) !== this.n;
  async run(m: Model, r: Real) {
    await r.app.openBuildControls();
    await r.app.act(
      'set-level',
      () => r.app.page.getByRole('spinbutton', { name: 'Character level' }).fill(String(this.n)),
      async () => {
        // Build.lua:218-224
        m.build.SetCharacterLevelAutoMode(false);
        m.build.SetLevel(this.n);
        return want(m)();
      }
    );
  }
  toString = () => `SetLevel(${this.n})`;
}

class SetClass implements fc.AsyncCommand<Model, Real> {
  constructor(readonly cls: string) {}
  check = (m: Readonly<Model>) => go.fresh(m.build, (b) => b.Build.ClassName) !== this.cls;
  async run(m: Model, r: Real) {
    await r.app.openBuildControls();
    await r.app.act(
      'set-class',
      () => r.app.page.getByRole('combobox', { name: 'Class', exact: true }).selectOption(this.cls),
      async () => {
        m.build.SetClass(this.cls);
        m.build.SetAscendancy('None');
        return want(m)();
      }
    );
  }
  toString = () => `SetClass(${this.cls})`;
}

class SetAscendancy implements fc.AsyncCommand<Model, Real> {
  constructor(readonly pick: number) {}
  private target = (m: Readonly<Model>) => (ASCENDANCIES[go.fresh(m.build, (b) => b.Build.ClassName)] ?? [])[this.pick % 3];
  check = (m: Readonly<Model>) => this.target(m) !== undefined && go.fresh(m.build, (b) => b.Build.AscendClassName) !== this.target(m);
  async run(m: Model, r: Real) {
    const asc = this.target(m);
    await r.app.openBuildControls();
    await r.app.act(
      'set-ascendancy',
      () => r.app.page.getByRole('combobox', { name: 'Ascendancy' }).selectOption(asc),
      async () => {
        await go.changeAscendancy(m.build, asc);
        return want(m)();
      }
    );
  }
  toString = () => `SetAscendancy(#${this.pick})`;
}

class SetMainSkill implements fc.AsyncCommand<Model, Real> {
  constructor(readonly k: number) {}
  private labels = (m: Readonly<Model>) => {
    const set = m.build.ActiveSkillSetIndex();
    return m.build.Skills.SkillSets[set].Skills.map((g) => g.DisplayLabel || g.Label || '<No active skills>');
  };
  check = (m: Readonly<Model>) => {
    const l = this.labels(m);
    return this.k < l.length && this.k !== go.fresh(m.build, (b) => b.Build.MainSocketGroup) - 1 && l.indexOf(l[this.k]) === l.lastIndexOf(l[this.k]);
  };
  async run(m: Model, r: Real) {
    const label = this.labels(m)[this.k];
    await r.app.act(
      'set-main-skill',
      () => r.app.page.getByRole('combobox', { name: 'Main skill' }).selectOption({ label }),
      async () => {
        m.build.SetMainSocketGroup(this.k + 1);
        return want(m)();
      }
    );
  }
  toString = () => `SetMainSkill(${this.k})`;
}

class ToggleConfig implements fc.AsyncCommand<Model, Real> {
  constructor(readonly i: number) {}
  check = () => CHECKS.length > 0;
  async run(m: Model, r: Real) {
    const c = CHECKS[this.i % CHECKS.length];
    const input = go.fresh(m.build, (b) => b.Config.Inputs.find((x) => x.Name === c.key));
    const next = !(input?.Boolean ?? c.defaultState);
    await r.app.goTo('Config');
    const box = optionControl(r.app.page, 'checkbox', c.name);
    await expect(box).toBeChecked({ checked: !next });
    await r.app.act(
      'toggle-config',
      () => box.setChecked(next),
      async () => {
        go.setConfig(m.build, c.key, next);
        return want(m)();
      }
    );
  }
  toString = () => `ToggleConfig(${CHECKS[this.i % CHECKS.length]?.key})`;
}

class AllocateNode implements fc.AsyncCommand<Model, Real> {
  constructor(readonly k: number) {}
  private pick = (m: Readonly<Model>) => {
    const c = tree.treeState(m.build).adjacent();
    return c.length > 0 ? c[this.k % c.length] : undefined;
  };
  check = (m: Readonly<Model>) => this.pick(m) !== undefined;
  async run(m: Model, r: Real) {
    const node = this.pick(m)!;
    const state = tree.treeState(m.build);
    await r.app.openTree(tree.treeVersion());
    const button = await r.app.treeNode(tree.resultName(node));
    await expect(button, 'the page holds a different allocation than the model').toHaveAttribute('aria-pressed', 'false');
    await r.app.act(
      'allocate-node',
      () => button.click(),
      async () => {
        state.allocate(node.id);
        return want(m)();
      }
    );
    await expect(button, 'the allocated node still reads as unallocated').toHaveAttribute('aria-pressed', 'true');
  }
  toString = () => `AllocateNode(#${this.k})`;
}

class DeallocateNode implements fc.AsyncCommand<Model, Real> {
  constructor(readonly k: number) {}
  private pick = (m: Readonly<Model>) => {
    const c = tree.treeState(m.build).allocated();
    return c.length > 0 ? c[this.k % c.length] : undefined;
  };
  check = (m: Readonly<Model>) => this.pick(m) !== undefined;
  async run(m: Model, r: Real) {
    const node = this.pick(m)!;
    const state = tree.treeState(m.build, await tree.clusterOf(m.build));
    await r.app.openTree(tree.treeVersion());
    const button = await r.app.treeNode(tree.resultName(node));
    await expect(button, 'the page holds a different allocation than the model').toHaveAttribute('aria-pressed', 'true');
    await r.app.act(
      'deallocate-node',
      () => button.click(),
      async () => {
        state.deallocate(node.id);
        return want(m)();
      }
    );
    await expect(button, 'the deallocated node still reads as allocated').toHaveAttribute('aria-pressed', 'false');
  }
  toString = () => `DeallocateNode(#${this.k})`;
}

class ExportMatches implements fc.AsyncCommand<Model, Real> {
  check = () => true;
  async run(m: Model, r: Real) {
    expect(go.withoutView(await go.decode(await r.app.exportCode())), 'the build Go holds after this sequence differs from the model').toBe(
      go.withoutView(await go.serializeSaved(m.build))
    );
  }
  toString = () => 'ExportMatches';
}

const LIVE_HEAP_GROWTH_BUDGET_MB = 30;

const commands = [
  fc.integer({ min: 1, max: 100 }).map((n) => new SetLevel(n)),
  fc.constantFrom(...Object.keys(ASCENDANCIES)).map((c) => new SetClass(c)),
  fc.nat(2).map((i) => new SetAscendancy(i)),
  fc.nat(9).map((k) => new SetMainSkill(k)),
  fc.nat(CHECKS.length - 1).map((i) => new ToggleConfig(i)),
  fc.nat(50).map((k) => new AllocateNode(k)),
  fc.nat(200).map((k) => new DeallocateNode(k)),
  fc.constant(new ExportMatches())
];

for (const file of ['0339-reliquarian-shield-crush-of-the-chieftain-2', '0444-reliquarian-static-strike-2']) {
  test(`random action sequences stay equal to the Go model — ${file}`, async ({ page }) => {
    const xml = buildXml(file);
    const code = await go.encode(xml);
    executed = 0;
    await tree.prepareTree();
    const app = await App.open(page, file);
    await app.openTree(tree.treeVersion());
    const before = await app.memory('before-sequences');

    await fc.assert(
      fc.asyncProperty(fc.commands(commands, { maxCommands: 10, size: 'large' }), async (cmds) => {
        const m: Model = { build: await go.loadXml(xml) };
        await app.importCode(code, want(m));
        await go.outputs(m.build);
        await fc.asyncModelRun(() => ({ model: m, real: { app } }), cmds);
      }),
      { numRuns: SEQUENCES, seed: SEED, verbose: 1, endOnFailure: false }
    );

    expect(executed, 'model-based run executed too few actions').toBeGreaterThanOrEqual(SEQUENCES * 4);
    const after = await app.memory('after-sequences');
    expect(before, 'the worker exposed no live-heap reading').not.toBeNull();
    expect(after! - before!, `live Go heap grew ${before} -> ${after} MB over the model run`).toBeLessThanOrEqual(LIVE_HEAP_GROWTH_BUDGET_MB);
    app.assertNoErrors();
  });
}
