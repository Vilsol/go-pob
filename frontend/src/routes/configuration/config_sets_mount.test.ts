// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi, type MockInstance } from 'vitest';
import { render, fireEvent, cleanup, within, screen } from '@testing-library/svelte';
import { get } from 'svelte/store';

afterEach(cleanup);
if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}
const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild, outputs } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');

const BUILD = '0045-hierophant-spark-of-the-nova.xml';
const OPTION = 'buffArcaneSurge';

const goActive = () => {
  const { sets, active } = workerInstance.GetConfigSets();
  return sets[active]?.id;
};
const goHas = (name: string) => name in workerInstance.GetAllConfigOptions();

let tick: MockInstance<typeof workerInstance.Tick>;

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD);
  tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
  vi.spyOn(workerInstance, 'ParseModLines').mockImplementation((lines: string[]) => lines.map(() => ({ Status: 'blank', Extra: '', Mods: [] }) as never));
  outputs.set(undefined);
}, 120_000);

const mount = async (q = 'Arcane Surge') => {
  const Page = (await import('./+page.svelte')).default;
  const view = render(Page);
  await view.findAllByRole('textbox', { name: /Modifier line/ });
  await fireEvent.input(view.getByRole('searchbox', { name: 'Filter options' }), { target: { value: q } });
  await fireEvent.click(view.getByRole('checkbox', { name: /Show options that don’t affect this build/ }));
  return view;
};
const setSelect = () => screen.getByRole<HTMLSelectElement>('combobox', { name: 'Config set' });
const optionBox = (view: { container: HTMLElement }) => {
  const row = [...view.container.querySelectorAll<HTMLElement>(`[data-var="${OPTION}"]`)].find((el) => !el.closest('[aria-label="Most used"]'));
  if (!row) throw new Error(`no ${OPTION} row on the page`);
  return within(row).getByRole<HTMLInputElement>('checkbox');
};

describe('the config-set select', () => {
  it('opens on the active set by id, and is live', async () => {
    expect(goActive()).toBe(3);
    expect(goHas(OPTION), 'the fixture: set 3 carries the option').toBe(true);
    const view = await mount();
    expect(setSelect().value).toBe('0');
    expect(setSelect().disabled).toBe(false);
    await vi.waitFor(() => expect(optionBox(view).checked).toBe(true));
  }, 20_000);

  it('switching writes the chosen set’s ID to Go, Ticks once, and shows that set’s options; switching back restores them', async () => {
    const view = await mount();
    tick.mockClear();

    await fireEvent.change(setSelect(), { target: { value: '1' } });
    await vi.waitFor(() => expect(goActive()).toBe(2));
    expect(goHas(OPTION)).toBe(false);
    expect(tick.mock.calls.map((c) => c[0])).toEqual(['SetActiveConfigSet']);
    await vi.waitFor(() => expect(optionBox(view).checked).toBe(false));
    expect(setSelect().value).toBe('1');

    await fireEvent.change(setSelect(), { target: { value: '0' } });
    await vi.waitFor(() => expect(goActive()).toBe(3));
    expect(goHas(OPTION)).toBe(true);
    await vi.waitFor(() => expect(optionBox(view).checked).toBe(true));
    expect(tick.mock.calls.map((c) => c[0])).toEqual(['SetActiveConfigSet', 'SetActiveConfigSet']);
  }, 20_000);
});

// ConfigSetListControl.lua:13-110
// ConfigTab.lua:721-766
describe('Manage config sets', () => {
  const titles = () => workerInstance.GetConfigSets().sets.map((s) => s.title ?? 'Default');
  const open = async () => {
    await fireEvent.change(setSelect(), { target: { value: '-1' } });
    return screen.getByRole('listbox', { name: 'Config sets' });
  };
  const nameAndSave = async (title: string) => {
    await fireEvent.input(screen.getByRole('textbox', { name: 'Name for this config set' }), { target: { value: title } });
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }));
  };

  it('New, Copy, Rename, Move and Delete edit Go’s sets', async () => {
    await mount();
    expect(titles()).toEqual(['Final', 'Swap on Spark Nova (70 lvl)']);
    const list = await open();
    expect(within(list).getAllByRole('option')[0].textContent).toContain('current');

    await fireEvent.click(screen.getByRole('button', { name: 'New' }));
    await nameAndSave('Boss');
    await vi.waitFor(() => expect(titles()).toEqual(['Final', 'Swap on Spark Nova (70 lvl)', 'Boss']));

    await fireEvent.click(within(list).getAllByRole('option')[0]);
    await fireEvent.click(screen.getByRole('button', { name: 'Copy' }));
    await nameAndSave('Final copy');
    await vi.waitFor(() => expect(titles()).toHaveLength(4));
    const copy = workerInstance.GetConfigSets().sets[3];
    expect(copy.title).toBe('Final copy');

    await vi.waitFor(() => expect(within(list).getAllByRole('option')).toHaveLength(4));
    await fireEvent.click(screen.getByRole('button', { name: 'Rename' }));
    await nameAndSave('Final two');
    await vi.waitFor(() => expect(titles()[3]).toBe('Final two'));

    await fireEvent.click(screen.getByRole('button', { name: 'Move up' }));
    await vi.waitFor(() => expect(titles()).toEqual(['Final', 'Swap on Spark Nova (70 lvl)', 'Final two', 'Boss']));
    // ConfigSetListControl.lua:71-110
    const row = within(list).getAllByRole('group')[0];
    await fireEvent.keyDown(row, { key: 'ArrowDown', altKey: true });
    await vi.waitFor(() => expect(titles()).toEqual(['Swap on Spark Nova (70 lvl)', 'Final', 'Final two', 'Boss']));
    await fireEvent.keyDown(within(list).getAllByRole('group')[1], { key: 'ArrowUp', altKey: true });
    await vi.waitFor(() => expect(titles()).toEqual(['Final', 'Swap on Spark Nova (70 lvl)', 'Final two', 'Boss']));

    await vi.waitFor(() => expect(within(list).getAllByRole('option')[2].textContent).toContain('Final two'));
    await fireEvent.dblClick(within(list).getAllByRole('option')[2]);
    await vi.waitFor(() => expect(goActive()).toBe(copy.id));
    expect(goHas(OPTION), 'the copy carries set 3’s options').toBe(true);

    for (const t of ['Final two', 'Boss']) {
      await vi.waitFor(() => expect(within(list).getByRole('option', { name: new RegExp(t) })).toBeDefined());
      await fireEvent.click(within(list).getByRole('option', { name: new RegExp(t) }));
      await fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
      await fireEvent.click(within(screen.getByRole('group', { name: 'Confirm delete' })).getByRole('button', { name: 'Delete' }));
      await vi.waitFor(() => expect(within(list).queryByRole('option', { name: new RegExp(t) })).toBeNull());
    }
    await vi.waitFor(() => expect(titles()).toEqual(['Final', 'Swap on Spark Nova (70 lvl)']));
    expect(goActive()).toBe(2);
    await fireEvent.change(setSelect(), { target: { value: '0' } });
    await vi.waitFor(() => expect(goActive()).toBe(3));
  }, 30_000);

  it('hovering a checkbox asks Go what toggling it gives', async () => {
    const view = await mount();
    const compare = vi.spyOn(workerInstance, 'ConfigCompare').mockResolvedValue([{ lines: [{ label: 'Total DPS', value: '+1', better: true }] }]);
    try {
      await fireEvent.mouseEnter(optionBox(view).closest(`[data-var="${OPTION}"]`)!);
      await vi.waitFor(() => expect(compare).toHaveBeenCalledWith(OPTION, false));
      await vi.waitFor(() => expect(screen.getByRole('tooltip').textContent).toContain('Toggling this option will give you:'));
    } finally {
      compare.mockRestore();
    }
  });

  it("a checkbox's what-if is asked again once its toggle is calculated, without a re-hover", async () => {
    const view = await mount();
    let calculated = false;
    const compare = vi
      .spyOn(workerInstance, 'ConfigCompare')
      .mockImplementation(() => Promise.resolve([{ lines: [{ label: calculated ? 'After the toggle' : 'Before the toggle', value: '+1', better: true }] }]));
    const before = get(outputs);
    const on = optionBox(view).checked;
    try {
      await fireEvent.mouseEnter(optionBox(view).closest(`[data-var="${OPTION}"]`)!);
      await vi.waitFor(() => expect(screen.getByRole('tooltip').textContent).toContain('Before the toggle'));
      await fireEvent.click(optionBox(view));
      await vi.waitFor(() => expect(goHas(OPTION)).toBe(!on));
      calculated = true;
      outputs.set({ ...before } as NonNullable<typeof before>);
      await vi.waitFor(() => expect(screen.getByRole('tooltip').textContent).toContain('After the toggle'));
      await fireEvent.click(optionBox(view));
      await vi.waitFor(() => expect(goHas(OPTION)).toBe(on));
    } finally {
      compare.mockRestore();
      outputs.set(before);
    }
  });

  // ConfigTab.lua:727-758
  it("hovering a dropdown's entry asks Go what selecting it gives", async () => {
    const view = await mount('Boss');
    const compare = vi.spyOn(workerInstance, 'ConfigCompare').mockResolvedValue([{ lines: [{ label: 'Total DPS', value: '+1', better: true }] }]);
    try {
      const row = view.container.querySelector<HTMLElement>('[data-var="enemyIsBoss"]')!;
      await fireEvent.click(within(row).getByRole('combobox'));
      const list = within(row).getByRole('listbox');
      await fireEvent.mouseEnter(within(list).getByRole('option', { name: 'Uber Pinnacle Boss' }));
      await vi.waitFor(() => expect(compare).toHaveBeenCalledWith('enemyIsBoss', 'Uber'));
      await vi.waitFor(() => expect(screen.getByRole('tooltip').textContent).toContain('Selecting this option will give you:'));
      compare.mockClear();
      await fireEvent.mouseEnter(within(list).getByRole('option', { selected: true }));
      await vi.waitFor(() => expect(screen.getByRole('tooltip').textContent).not.toContain('will give you'));
      expect(compare).not.toHaveBeenCalled();
    } finally {
      compare.mockRestore();
    }
  });
});
