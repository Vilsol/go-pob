// @vitest-environment jsdom

import { describe, expect, it, beforeAll, vi, type MockInstance } from 'vitest';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

vi.mock('@zenfs/core', async () => (await import(/* @vite-ignore */ `${process.cwd()}/harness/design_fixtures/zenfs_node.js`)) as Record<string, unknown>);

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, recordedOutputs } = await import('../../../harness/component/go_runtime');
const { currentBuild } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');
const types = await import('$lib/types');

const BUILD = '0008-necromancer-vaal-summon-skeletons.xml';

type Internals = { ticksInFlight: number; computeOutputs: () => Promise<unknown> };
const internals = workerInstance as unknown as Internals;

let clock = 1_800_000_000_000;
const later = (ms = 10_000) => (clock += ms);

const settle = async () => {
  await vi.waitFor(() => expect(internals.ticksInFlight).toBe(0));
  await workerInstance.GetSidebarWarnings();
  await workerInstance.GetUndoHistoryWhenRecorded();
};

const serialize = () => new TextDecoder().decode(types.builds.SerializeBuild(workerInstance.currentBuild).unwrap());

const pageView = async () => {
  const warnings = await workerInstance.GetSidebarWarnings();
  return {
    outputs: recordedOutputs.at(-1),
    warnings: warnings && { warnings: warnings.warnings, costWarning: warnings.costWarning },
    groups: workerInstance.GetSocketGroupSummaries(workerInstance.currentBuild!.ActiveSkillSetIndex()),
    info: workerInstance.GetBuildInfo(),
    xml: serialize()
  };
};

const freshView = async () => {
  await workerInstance.Tick('Redo');
  await settle();
  return pageView();
};

let calc: MockInstance<() => Promise<unknown>>;

beforeAll(async () => {
  vi.spyOn(console, 'log').mockImplementation(() => {});
  await bootWorker(currentBuild);
  await workerInstance.loadData(async () => {});
  calc = vi.spyOn(internals, 'computeOutputs');
  vi.spyOn(Date, 'now').mockImplementation(() => clock);
}, 300_000);

describe('a restore to a state calculated before reuses that calculation', () => {
  it('undo and redo show what a fresh calculation of each restored state shows, without calculating it', async () => {
    const xml = await readFile(resolve(process.cwd(), '../testdata/corpus-3.29', BUILD), 'utf8');
    await workerInstance.ImportCode(types.pob.CompressEncode(xml).unwrap());
    await settle();
    const level0 = workerInstance.GetBuildInfo()!.level;

    later();
    workerInstance.SetLevel(level0 - 10);
    await settle();
    later();
    workerInstance.SetConfigOption('enemyLevel', 70);
    await settle();
    later();
    const group = workerInstance.currentBuild!.Skills.SkillSets[workerInstance.currentBuild!.ActiveSkillSetIndex()].Skills[0];
    group.Enabled = !group.Enabled;
    await workerInstance.Tick('currentGemGroup');
    await settle();
    expect(workerInstance.GetUndoHistory().steps).toHaveLength(3);

    expect(
      workerInstance.GetSocketGroupSummaries(workerInstance.currentBuild!.ActiveSkillSetIndex()).some((g) => g.label !== '<No active skills>'),
      'the build has a labelled group to lose'
    ).toBe(true);

    const restores: [string, () => unknown][] = [
      ['undo', () => workerInstance.Undo()],
      ['undo', () => workerInstance.Undo()],
      ['undo', () => workerInstance.Undo()],
      ['redo', () => workerInstance.Redo()],
      ['redo', () => workerInstance.Redo()],
      ['go to the newest step', () => workerInstance.UndoTo(0)],
      ['go to the base', () => workerInstance.UndoTo(3)],
      ['redo', () => workerInstance.Redo()]
    ];
    for (const [i, [name, restore]] of restores.entries()) {
      const calcs = calc.mock.calls.length;
      restore();
      await settle();
      expect(calc.mock.calls.length, `restore ${i} (${name}) calculated`).toBe(calcs);
      const shown = await pageView();
      const fresh = await freshView();
      expect(shown.outputs, `restore ${i} (${name}): the outputs`).toEqual(fresh.outputs);
      expect(shown.warnings, `restore ${i} (${name}): the warnings`).toEqual(fresh.warnings);
      expect(shown.groups, `restore ${i} (${name}): the socket groups`).toEqual(fresh.groups);
      expect(shown.info, `restore ${i} (${name}): the build info`).toEqual(fresh.info);
      expect(shown.xml, `restore ${i} (${name}): the build Go holds`).toBe(fresh.xml);
    }
  }, 300_000);
});
