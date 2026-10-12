// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi, type MockInstance } from 'vitest';
import { render, fireEvent, cleanup, within, screen } from '@testing-library/svelte';

afterEach(cleanup);
const pageErrors: string[] = [];
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});
vi.mock('$lib/cache', () => ({ GetSkillGems: () => Promise.resolve([]) }));

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
  };
}

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');

const BUILD = '0045-hierophant-spark-of-the-nova.xml';

const goSkills = () => {
  const skills = workerInstance.currentBuild?.Skills;
  if (!skills) throw new Error('REFUSING: no build loaded');
  return skills;
};
const goSetLabels = (pos: number) => (goSkills().SkillSets?.[pos]?.Skills ?? []).map((g) => g.Label);

let tick: MockInstance<typeof workerInstance.Tick>;

beforeAll(async () => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD);
  tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
});

const list = () => screen.getByRole('region', { name: 'Socket groups' });
const cards = () =>
  within(list())
    .getAllByRole('button')
    .filter((b) => b.hasAttribute('aria-current'));
const setSelect = () => within(list()).getByRole<HTMLSelectElement>('combobox', { name: 'Skill set' });

const mountSkills = async () => {
  const Page = (await import('./+page.svelte')).default;
  const rendered = render(Page);
  await vi.waitFor(() => expect(cards().length).toBeGreaterThan(0), { timeout: 10_000 });
  return rendered;
};

describe('the skill-set select', () => {
  it('opens on the active set by id, not on SkillSets[id - 1]', async () => {
    expect(goSkills().ActiveSkillSet).toBe(3);
    expect(goSkills().SkillSets?.[0]?.ID, 'the fixture must keep set 3 first').toBe(3);
    await mountSkills();
    expect(setSelect().value).toBe('0');
    expect(setSelect().disabled, 'the select is live').toBe(false);
    expect(cards().length).toBe(goSetLabels(0).length);
  }, 20_000);

  it('switching sets writes the chosen set’s ID to Go, Ticks once, and lists that set’s groups', async () => {
    const sets = goSkills().SkillSets ?? [];
    sets[0].Skills[0].Label = 'ONLY IN SET 3';
    sets[1].Skills[0].Label = 'ONLY IN SET 2';
    expect(goSetLabels(0)[0]).toBe('ONLY IN SET 3');
    await mountSkills();
    const shows = (label: string) => cards().some((c) => c.textContent?.includes(label));
    await vi.waitFor(() => expect(shows('ONLY IN SET 3')).toBe(true));
    tick.mockClear();

    await fireEvent.change(setSelect(), { target: { value: '1' } });
    await vi.waitFor(() => expect(goSkills().ActiveSkillSet).toBe(2));
    expect(tick.mock.calls.map((c) => c[0])).toEqual(['SetActiveSkillSet']);
    await vi.waitFor(() => expect(shows('ONLY IN SET 2')).toBe(true));
    expect(shows('ONLY IN SET 3')).toBe(false);
    expect(setSelect().value).toBe('1');

    await fireEvent.change(setSelect(), { target: { value: '0' } });
    await vi.waitFor(() => expect(goSkills().ActiveSkillSet).toBe(3));
    await vi.waitFor(() => expect(shows('ONLY IN SET 3')).toBe(true));
    expect(tick.mock.calls.map((c) => c[0])).toEqual(['SetActiveSkillSet', 'SetActiveSkillSet']);
  }, 20_000);

  it('a new group lands in the active set (by id), not in SkillSets[id - 1]', async () => {
    await mountSkills();
    expect(goSkills().ActiveSkillSet).toBe(3);
    const before = [goSetLabels(0).length, goSetLabels(1).length];
    await fireEvent.click(within(list()).getByRole('button', { name: '+ New group' }));
    await vi.waitFor(() => expect(goSetLabels(0).length).toBe(before[0] + 1));
    expect(goSetLabels(1).length).toBe(before[1]);
  }, 20_000);
});

describe('reordering socket groups', () => {
  it('Alt+↓ moves a group in Go, Ticks, and the main group stays on the group it was', async () => {
    expect(goSkills().ActiveSkillSet).toBe(3);
    const groups = goSkills().SkillSets?.[0]?.Skills ?? [];
    groups.forEach((g, i) => (g.Label = `G${i + 1}`));
    const before = goSetLabels(0);
    const otherBefore = goSetLabels(1);
    const goMain = () => workerInstance.GetBuildInfo()?.mainSocketGroup;
    expect(goMain(), 'the fixture: group 1 is the main group').toBe(1);
    await mountSkills();
    await vi.waitFor(() => expect(cards()[0].textContent).toContain('G1'));
    const rows = () =>
      within(list())
        .getAllByRole('group')
        .filter((r) => r.hasAttribute('data-reorder-item'));
    expect(rows().length, 'the group cards drag').toBe(before.length);
    tick.mockClear();

    await fireEvent.keyDown(rows()[0], { key: 'ArrowDown', altKey: true });
    await vi.waitFor(() => expect(goSetLabels(0)).toEqual([before[1], before[0], ...before.slice(2)]));
    await fireEvent.keyDown(rows()[1], { key: 'ArrowDown', altKey: true });
    await vi.waitFor(() => expect(goSetLabels(0)).toEqual([before[1], before[2], before[0], ...before.slice(3)]));
    expect(tick.mock.calls.map((c) => c[0])).toEqual(['MoveSocketGroup', 'MoveSocketGroup']);
    expect(goMain(), 'SkillListControl.lua:125-126: the main group follows its group').toBe(3);
    await vi.waitFor(() => {
      const main = cards().find((c) => c.textContent?.includes('MAIN'));
      expect(main?.textContent).toContain('G1');
    });
    expect(goSetLabels(1), 'the other set is untouched').toEqual(otherBefore);
  }, 20_000);
});

// SkillsTab.lua:1415-1422, SkillSetListControl.lua:14-110
describe('managing skill sets', () => {
  const goIds = () => (goSkills().SkillSets ?? []).map((s) => s.ID);
  const goTitles = () => (goSkills().SkillSets ?? []).map((s) => s.Title);
  const dialog = () => screen.getByRole('dialog', { name: 'Manage skill sets' });
  const open = async () => {
    await fireEvent.change(setSelect(), { target: { value: '-1' } });
    await vi.waitFor(() => expect(dialog()).toBeTruthy());
  };
  const rows = () => within(dialog()).getAllByRole('option');
  const name = async (title: string) => {
    const box = within(dialog()).getByRole<HTMLInputElement>('textbox', { name: 'Name for this skill set' });
    await fireEvent.input(box, { target: { value: title } });
    await fireEvent.click(within(dialog()).getByRole('button', { name: 'Save' }));
  };

  it('New, Copy, Rename and Delete land in Go', async () => {
    await mountSkills();
    expect(goIds()).toEqual([3, 2]);
    await open();

    await fireEvent.click(within(dialog()).getByRole('button', { name: 'New' }));
    await name('Bossing');
    await vi.waitFor(() => expect(goIds()).toEqual([3, 2, 1]));
    expect(goTitles()[2]).toBe('Bossing');
    await vi.waitFor(() => expect(rows().map((r) => r.textContent)).toEqual(['Finalcurrent', 'Swap on Spark Nova (70 lvl)', 'Bossing']));

    await fireEvent.click(within(dialog()).getByRole('button', { name: 'Copy' }));
    await name('Final copy');
    await vi.waitFor(() => expect(goIds()).toEqual([3, 2, 1, 4]));
    expect(goSetLabels(3)).toEqual(goSetLabels(0));
    await vi.waitFor(() => expect(rows()).toHaveLength(4));

    await fireEvent.click(rows()[2]);
    await fireEvent.click(within(dialog()).getByRole('button', { name: 'Rename' }));
    await name('Bosses');
    await vi.waitFor(() => expect(goTitles()[2]).toBe('Bosses'));
    await vi.waitFor(() => expect(rows()[2].textContent).toBe('Bosses'));

    await fireEvent.click(rows()[2]);
    await fireEvent.click(within(dialog()).getByRole('button', { name: 'Delete' }));
    await fireEvent.click(within(within(dialog()).getByRole('group', { name: 'Confirm delete' })).getByRole('button', { name: 'Delete' }));
    await vi.waitFor(() => expect(goIds()).toEqual([3, 2, 4]));
    expect(goSkills().ActiveSkillSet, 'deleting another set keeps the active one').toBe(3);
  }, 30_000);
});
