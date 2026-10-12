// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi, type MockInstance } from 'vitest';
import { render, fireEvent, cleanup } from '@testing-library/svelte';
import type { calculator } from '$lib/types';
import type { MainSkillField } from './main_skill';

afterEach(cleanup);

const pageErrors: string[] = [];
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted component threw').toEqual([]);
});

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');

const BUILD = '0001-necromancer-unknown.xml';
const GEM = 0;
let GROUP = -1;

const goGroup = () => workerInstance.currentBuild?.Skills?.SkillSets?.[0]?.Skills?.[GROUP];
const goGem = () => goGroup()?.Gems?.[GEM];

let tick: MockInstance<(source: string) => Promise<void>>;

beforeAll(async () => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD);
  GROUP = (workerInstance.currentBuild?.Skills?.SkillSets?.[0]?.Skills ?? []).findIndex((g) => (g.Gems?.length ?? 0) > 0);
  tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
}, 120_000);

const controls = (): calculator.MainSkillControls =>
  ({
    SkillSet: 0,
    SocketGroup: GROUP,
    GemIndex: GEM,
    Skills: ['First', 'Second'],
    SkillEnabled: true,
    MainActiveSkill: 1,
    PartShown: true,
    Parts: ['Part A', 'Part B'],
    SkillPart: 1,
    StageShown: true,
    StageCount: '5',
    MineShown: true,
    MineCount: '',
    MinionShown: true,
    MinionEnabled: true,
    MinionLibraryShown: false,
    Minions: [
      { Label: 'Zombie', MinionID: 'RaisedZombie', ItemSetID: 0 },
      { Label: 'Spectre', MinionID: 'SomeSpectre', ItemSetID: 0 }
    ],
    Minion: 1,
    MinionSkillShown: true,
    MinionSkillEnabled: true,
    MinionSkills: ['Default Attack', 'Slam'],
    MinionSkill: 1
  }) as unknown as calculator.MainSkillControls;

const render_ = async (c: calculator.MainSkillControls) => {
  const Sidebar = (await import('$lib/design/Sidebar.svelte')).default;
  const { writeMainSkill } = await import('./main_skill');
  return render(Sidebar, {
    props: {
      sidebar: { sections: [], warnings: [] },
      socketGroups: ['Group'],
      mainSocketGroup: 0,
      controls: c,
      oncontrol: (field: MainSkillField, v: number | undefined) => writeMainSkill(c, field, v)
    }
  });
};
const mount = () => render_(controls());

const ticked = (source: string) => tick.mock.calls.some((c) => c[0] === source);

describe('the sidebar main-skill writes land in Go and each is followed by a Tick', () => {
  it('the harness holds the gem the controls address', () => {
    expect(goGem(), `${BUILD} has no gem ${GEM} in group ${GROUP}`).toBeDefined();
  });

  it('mounting alone writes nothing and does not Tick', async () => {
    const before = JSON.stringify(goGem());
    tick.mockClear();
    await mount();
    await new Promise((r) => setTimeout(r, 50));
    expect(JSON.stringify(goGem())).toBe(before);
    expect(tick).not.toHaveBeenCalled();
  });

  it('active skill', async () => {
    const { getByRole } = await mount();
    await fireEvent.change(getByRole('combobox', { name: 'Active skill' }), { target: { value: '2' } });
    await vi.waitFor(() => expect(goGroup()?.MainActiveSkill).toBe(2));
    await vi.waitFor(() => expect(ticked('SetMainActiveSkill')).toBe(true));
  });

  it('skill part', async () => {
    const { getByRole } = await mount();
    await fireEvent.change(getByRole('combobox', { name: 'Skill part' }), { target: { value: '2' } });
    await vi.waitFor(() => expect(goGem()?.SkillPart).toBe(2));
    await vi.waitFor(() => expect(ticked('SetGemSkillPart')).toBe(true));
  });

  it('stages, and an empty box clears them', async () => {
    const { getByRole } = await mount();
    const box = getByRole('textbox', { name: 'Stages' });
    await fireEvent.input(box, { target: { value: '3' } });
    await vi.waitFor(() => expect(goGem()?.SkillStageCount).toBe(3));
    await fireEvent.input(box, { target: { value: '' } });
    await vi.waitFor(() => expect(goGem()?.SkillStageCount).toBe(0));
    await vi.waitFor(() => expect(ticked('SetGemSkillStageCount')).toBe(true));
  });

  it('active mines, and an empty box clears them', async () => {
    const { getByRole } = await mount();
    const box = getByRole('textbox', { name: 'Active mines' });
    await fireEvent.input(box, { target: { value: '4' } });
    await vi.waitFor(() => expect(goGem()?.SkillMineCount).toBe(4));
    await fireEvent.input(box, { target: { value: '' } });
    await vi.waitFor(() => expect(goGem()?.SkillMineCount).toBeNull());
    await vi.waitFor(() => expect(ticked('SetGemSkillMineCount')).toBe(true));
  });

  it('minion', async () => {
    const { getByRole } = await mount();
    await fireEvent.change(getByRole('combobox', { name: 'Minion' }), { target: { value: '2' } });
    await vi.waitFor(() => expect(goGem()?.SkillMinion).toBe('SomeSpectre'));
    await vi.waitFor(() => expect(ticked('SetGemSkillMinion')).toBe(true));
  });

  it('minion item set', async () => {
    const c = controls();
    c.Minions = [
      { Label: 'Default Item Set', MinionID: '', ItemSetID: 1 },
      { Label: 'Guardian', MinionID: '', ItemSetID: 7 }
    ] as unknown as calculator.MainSkillControls['Minions'];
    const { getByRole } = await render_(c);
    await fireEvent.change(getByRole('combobox', { name: 'Minion' }), { target: { value: '2' } });
    await vi.waitFor(() => expect(goGem()?.SkillMinionItemSet).toBe(7));
    await vi.waitFor(() => expect(ticked('SetGemSkillMinionItemSet')).toBe(true));
  });

  it('minion skill', async () => {
    const { getByRole } = await mount();
    await fireEvent.change(getByRole('combobox', { name: 'Minion skill' }), { target: { value: '2' } });
    await vi.waitFor(() => expect(goGem()?.SkillMinionSkill).toBe(2));
    await vi.waitFor(() => expect(ticked('SetGemSkillMinionSkill')).toBe(true));
  });
});
