// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, onTestFinished, vi, type MockInstance } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { get } from 'svelte/store';

afterEach(cleanup);

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
vi.mock('$lib/skill_tree', async (original) => ({ ...(await original<typeof import('$lib/skill_tree')>()), loadSkillTree: vi.fn(async () => {}) }));

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild, outputs } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');
const types = await import('$lib/types');

const BUILD = '0280-elementalist-fireball.xml';

const info = () => {
  const i = workerInstance.GetBuildInfo()!;
  return { className: i.className, ascendClassName: i.ascendClassName, level: i.level };
};

let tick: MockInstance;

globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD);
  tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
  vi.spyOn(workerInstance, 'GetTreeSpecsPoints').mockResolvedValue([]);
  outputs.set(undefined);
}, 120_000);

const mount = async () => {
  const Page = (await import('./+page.svelte')).default;
  const view = render(Page);
  const start = info();
  await vi.waitFor(() => expect((view.getByRole('combobox', { name: 'Class' }) as HTMLSelectElement).value).toBe(start.className), { timeout: 10_000 });
  return view;
};

const treeEntries = async (box: HTMLElement) => {
  await fireEvent.click(box);
  const names = within(box.parentElement!)
    .queryAllByRole('option')
    .map((o) => o.textContent!.trim());
  await fireEvent.keyDown(box, { key: 'Escape' });
  return names;
};
const chooseTree = async (box: HTMLElement, i: number) => {
  await fireEvent.click(box);
  await fireEvent.click(within(box.parentElement!).getAllByRole('option')[i]);
};

describe('the Tree page edits the build', () => {
  it('shows what Go holds and writes nothing on mount', async () => {
    const before = info();
    tick.mockClear();
    const view = await mount();
    expect((view.getByRole('combobox', { name: 'Ascendancy' }) as HTMLSelectElement).value).toBe(before.ascendClassName);
    expect((view.getByRole('spinbutton', { name: 'Character level' }) as HTMLInputElement).value).toBe(String(before.level));
    await new Promise((r) => setTimeout(r, 200));
    expect(info(), 'mounting changed the build').toEqual(before);
    expect(tick, 'mounting recalculated').not.toHaveBeenCalled();
  }, 60_000);

  it('the level is a number input right of the ascendancy', async () => {
    const read = vi
      .spyOn(workerInstance, 'GetSecondaryAscendancies')
      .mockResolvedValue({ Entries: [{ Label: 'None', AscendClassID: 0 }], Selected: 0, Enabled: false } as never);
    onTestFinished(() => read.mockRestore());
    const view = await mount();
    const asc = view.getByRole('combobox', { name: 'Ascendancy' });
    const box = view.getByRole('spinbutton', { name: 'Character level' });
    expect(box.closest('.number-input'), 'not the design system NumberInput').not.toBeNull();
    const bar = asc.closest('[data-toolbar]')!;
    const items = [...bar.children];
    const at = (el: Element) => items.findIndex((c) => c.contains(el));
    const grow = items.findIndex((c) => c.classList.contains('grow'));
    // Build.lua:285
    expect(at(box)).toBe(at(await view.findByRole('combobox', { name: 'Bloodline' })) + 1);
    expect(at(box)).toBeGreaterThan(at(asc));
    expect(at(box)).toBeLessThan(grow);
  }, 60_000);

  it('the right sidebar is resizable', async () => {
    const view = await mount();
    await fireEvent.input(view.getByRole('searchbox', { name: 'Search the tree' }), { target: { value: 'life' } });
    const aside = view.getByRole('complementary', { name: 'Tree details' });
    const handle = view.getByRole('separator', { name: 'Resize tree details' });
    const width = () => parseFloat(aside.style.width);
    const w0 = width();
    expect(w0).toBeGreaterThan(0);
    await fireEvent.keyDown(handle, { key: 'ArrowLeft' });
    expect(width()).toBeGreaterThan(w0);
    const w1 = width();
    if (!('PointerEvent' in window)) Object.assign(window, { PointerEvent: MouseEvent });
    await fireEvent.pointerDown(handle, { clientX: 500, pointerId: 1, button: 0 });
    await fireEvent.pointerMove(handle, { clientX: 560, pointerId: 1 });
    await fireEvent.pointerUp(handle, { clientX: 560, pointerId: 1 });
    expect(width()).toBe(w1 - 60);
    expect(handle.getAttribute('aria-valuenow')).toBe(String(w1 - 60));
  }, 60_000);

  it('the details sidebar hides with an empty search and the power report closed', async () => {
    vi.spyOn(workerInstance, 'GetPowerStats').mockReturnValue([{ Stat: 'Life', Label: 'Life' }] as never);
    const view = await mount();
    const aside = () => view.queryByRole('complementary', { name: 'Tree details' });
    const handle = () => view.queryByRole('separator', { name: 'Resize tree details' });
    expect(aside()).toBeNull();
    expect(handle()).toBeNull();
    const search = view.getByRole('searchbox', { name: 'Search the tree' });
    await fireEvent.input(search, { target: { value: 'life' } });
    await fireEvent.keyDown(handle()!, { key: 'ArrowLeft' });
    const w = aside()!.style.width;
    await fireEvent.input(search, { target: { value: '  ' } });
    expect(aside()).toBeNull();
    await fireEvent.click(view.getByRole('button', { name: /^Power/ }));
    expect(aside()!.style.width).toBe(w);
    await fireEvent.click(view.getByRole('button', { name: 'Close power report' }));
    expect(aside()).toBeNull();
    vi.restoreAllMocks();
    tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    vi.spyOn(workerInstance, 'GetTreeSpecsPoints').mockResolvedValue([]);
  }, 60_000);

  it('a build-info read that started before a level edit does not overwrite it', async () => {
    const view = await mount();
    const box = view.getByRole('spinbutton', { name: 'Character level' }) as HTMLInputElement;
    const stale = workerInstance.GetBuildInfo()!;
    let release!: () => void;
    const spy = vi.spyOn(workerInstance, 'GetBuildInfo').mockImplementationOnce(() => new Promise((resolve) => (release = () => resolve(stale))) as never);
    outputs.set({} as never);
    await vi.waitFor(() => expect(spy).toHaveBeenCalled());
    const n = stale.level === 37 ? 38 : 37;
    await fireEvent.input(box, { target: { value: String(n) } });
    await vi.waitFor(() => expect(info().level).toBe(n), { timeout: 10_000 });
    await vi.waitFor(() => expect(box.value).toBe(String(n)));
    release();
    await new Promise((r) => setTimeout(r, 100));
    expect(box.value, 'a read from before the edit put the old level back').toBe(String(n));
    spy.mockRestore();
    outputs.set(undefined);
  }, 60_000);

  it('class, then ascendancy, then level reach Go', async () => {
    const changeClass = vi.spyOn(workerInstance, 'ChangeTreeClass').mockImplementation((c: string) => {
      workerInstance.SetClass(c);
      workerInstance.SetAscendancy('None');
      return Promise.resolve({ changed: true, needsConfirm: false, className: '', handled: false });
    });
    const changeAscendancy = vi
      .spyOn(workerInstance, 'ChangeTreeAscendancy')
      .mockImplementation((a: string) => Promise.resolve(workerInstance.SetAscendancy(a)));
    const view = await mount();
    const start = info();
    const cls = start.className === 'Marauder' ? 'Ranger' : 'Marauder';

    await fireEvent.change(view.getByRole('combobox', { name: 'Class' }), { target: { value: cls } });
    await vi.waitFor(() => expect(info()).toEqual({ ...start, className: cls, ascendClassName: 'None' }), { timeout: 10_000 });
    expect(changeClass).toHaveBeenCalledWith(cls);

    const asc = cls === 'Marauder' ? 'Berserker' : 'Deadeye';
    const select = view.getByRole('combobox', { name: 'Ascendancy' }) as HTMLSelectElement;
    await vi.waitFor(() => expect([...select.options].map((o) => o.value)).toContain(asc));
    await fireEvent.change(select, { target: { value: asc } });
    await vi.waitFor(() => expect(info()).toEqual({ ...start, className: cls, ascendClassName: asc }), { timeout: 10_000 });
    expect(changeAscendancy).toHaveBeenCalledWith(asc);
    changeClass.mockRestore();
    changeAscendancy.mockRestore();

    const level = view.getByRole('spinbutton', { name: 'Character level' });
    const n = start.level === 42 ? 43 : 42;
    await fireEvent.input(level, { target: { value: String(n) } });
    await vi.waitFor(() => expect(info()).toEqual({ className: cls, ascendClassName: asc, level: n }), { timeout: 10_000 });
    await fireEvent.input(level, { target: { value: '250' } });
    await vi.waitFor(() => expect(info().level).toBe(100), { timeout: 10_000 });
  }, 60_000);
});

// Build.lua:285-299
// Build.lua:1093-1138
describe('the secondary ascendancy select', () => {
  const list = (selected: number, enabled = true) => ({
    Entries: [
      { Label: 'None', AscendClassID: 0 },
      { Label: 'Aul', AscendClassID: 4 },
      { Label: 'Oshabi', AscendClassID: 2 }
    ],
    Selected: selected,
    Enabled: enabled
  });

  it('sits right of the ascendancy, lists Go’s entries and writes the chosen id', async () => {
    let selected = 0;
    const read = vi.spyOn(workerInstance, 'GetSecondaryAscendancies').mockImplementation(() => Promise.resolve(list(selected) as never));
    const change = vi.spyOn(workerInstance, 'ChangeSecondaryAscendancy').mockImplementation((id: number) => {
      selected = id;
      return Promise.resolve();
    });
    try {
      const view = await mount();
      const asc = view.getByRole('combobox', { name: 'Ascendancy' });
      const sec = (await view.findByRole('combobox', { name: 'Bloodline' })) as HTMLSelectElement;
      const items = [...asc.closest('[data-toolbar]')!.children];
      const at = (el: Element) => items.findIndex((c) => c.contains(el));
      expect(at(sec)).toBe(at(asc) + 1);
      await vi.waitFor(() => expect([...sec.options].map((o) => o.textContent)).toEqual(['None', 'Aul', 'Oshabi']));
      expect(sec.disabled).toBe(false);
      expect(sec.value).toBe('0');

      await fireEvent.change(sec, { target: { value: '2' } });
      await vi.waitFor(() => expect(change).toHaveBeenCalledWith(2));
      await vi.waitFor(() => expect(sec.value).toBe('2'));
      expect(change).toHaveBeenCalledTimes(1);
    } finally {
      read.mockRestore();
      change.mockRestore();
    }
  }, 60_000);

  it('is disabled while the tree has none', async () => {
    const read = vi
      .spyOn(workerInstance, 'GetSecondaryAscendancies')
      .mockResolvedValue({ Entries: [{ Label: 'None', AscendClassID: 0 }], Selected: 0, Enabled: false } as never);
    try {
      const view = await mount();
      const sec = (await view.findByRole('combobox', { name: 'Bloodline' })) as HTMLSelectElement;
      await vi.waitFor(() => expect(sec.disabled).toBe(true));
    } finally {
      read.mockRestore();
    }
  }, 60_000);
});

// Build.lua:263-275
describe('the Tree page asks before a class change resets the tree', () => {
  const asked = { changed: false, needsConfirm: true, className: 'Shadow', handled: false };
  const done = { changed: true, needsConfirm: false, className: '', handled: false };

  it('Cancel writes nothing; Connect Path answers 2; a connected change does not ask', async () => {
    await loadCorpusBuild(BUILD);
    const change = vi.spyOn(workerInstance, 'ChangeTreeClass').mockImplementation((_c: string, decision = 0) => Promise.resolve(decision === 0 ? asked : done));
    const view = await mount();
    const start = info();
    const select = view.getByRole('combobox', { name: 'Class' }) as HTMLSelectElement;

    await fireEvent.change(select, { target: { value: 'Shadow' } });
    let confirm = await view.findByRole('dialog', { name: 'Class Change' }, { timeout: 10_000 });
    expect(confirm.textContent).toContain('Changing class to Shadow will reset your passive tree.');
    expect(confirm.textContent).toContain('This can be avoided by connecting one of the Shadow starting nodes to your tree.');
    await vi.waitFor(() => expect(select.value, 'the select shows the class the build has').toBe(start.className));
    await fireEvent.click(within(confirm).getByRole('button', { name: 'Cancel' }));
    await new Promise((r) => setTimeout(r, 100));
    expect(change.mock.calls).toEqual([['Shadow']]);
    expect(view.queryByRole('dialog', { name: 'Class Change' })).toBeNull();

    change.mockClear();
    await fireEvent.change(select, { target: { value: 'Shadow' } });
    confirm = await view.findByRole('dialog', { name: 'Class Change' }, { timeout: 10_000 });
    await fireEvent.click(within(confirm).getByRole('button', { name: 'Connect Path' }));
    await vi.waitFor(() => expect(change.mock.calls).toEqual([['Shadow'], ['Shadow', 2]]));

    change.mockClear();
    await fireEvent.change(select, { target: { value: 'Shadow' } });
    confirm = await view.findByRole('dialog', { name: 'Class Change' }, { timeout: 10_000 });
    await fireEvent.click(within(confirm).getByRole('button', { name: 'Continue' }));
    await vi.waitFor(() => expect(change.mock.calls).toEqual([['Shadow'], ['Shadow', 1]]));

    change.mockClear();
    change.mockResolvedValue(done);
    await fireEvent.change(select, { target: { value: 'Templar' } });
    await vi.waitFor(() => expect(change.mock.calls).toEqual([['Templar']]));
    await new Promise((r) => setTimeout(r, 100));
    expect(view.queryByRole('dialog', { name: 'Class Change' }), 'a change Go made at once asked').toBeNull();
    change.mockRestore();
    expect(info(), 'the mocked changes wrote to Go').toEqual(start);
  }, 60_000);
});

// Build.lua:211-217
describe('the Tree page toggles the auto character level', () => {
  it('Manual turns Auto with SetCharacterLevelAutoMode(true), and back', async () => {
    await loadCorpusBuild(BUILD);
    const w = workerInstance as unknown as { SetCharacterLevelAutoMode?: (auto: boolean) => void };
    w.SetCharacterLevelAutoMode ??= () => {};
    const set = vi.spyOn(w as { SetCharacterLevelAutoMode: (auto: boolean) => void }, 'SetCharacterLevelAutoMode');
    const view = await mount();
    const button = await view.findByRole('button', { name: /^(Auto|Manual)$/ });
    const was = button.textContent === 'Auto';
    await fireEvent.click(button);
    await vi.waitFor(() => expect(set).toHaveBeenCalledWith(!was));
    expect(button.textContent).toBe(was ? 'Manual' : 'Auto');
    set.mockRestore();
  }, 60_000);
});

// TreeTab.lua:156-175, 322-340, 588-680
describe('the Tree page converts tree versions', () => {
  it('the select asks first; the older-tree banner converts a copy and converts all', async () => {
    await loadCorpusBuild('0004-trickster-lightning-tendrils-of-eccentricity.xml');
    const convert = vi.spyOn(workerInstance, 'ConvertTreeSpec').mockResolvedValue('');
    const convertAll = vi.spyOn(workerInstance, 'ConvertAllTreeSpecs').mockResolvedValue('');
    vi.spyOn(workerInstance, 'GetTreeVersionOptions').mockReturnValue([
      { value: '3_20', label: '3.20' },
      { value: '3_29', label: '3.29' }
    ]);
    const view = await mount();
    const version = (await view.findByRole('combobox', { name: 'Version' })) as HTMLSelectElement;
    await vi.waitFor(() => expect(version.value).toBe('3_29'));
    expect(view.queryByRole('status', { name: 'Convert older tree' })).toBeNull();

    await fireEvent.change(version, { target: { value: '3_20' } });
    let dialog = await view.findByRole('dialog', { name: 'Convert to Version 3.20' });
    await fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }));
    await vi.waitFor(() => expect(version.value).toBe('3_29'));
    expect(convert).not.toHaveBeenCalled();
    await fireEvent.change(version, { target: { value: '3_20' } });
    dialog = await view.findByRole('dialog', { name: 'Convert to Version 3.20' });
    await fireEvent.click(within(dialog).getByRole('button', { name: 'Copy + Convert' }));
    await vi.waitFor(() => expect(convert).toHaveBeenCalledWith('3_20', false));

    await chooseTree(view.getByRole('combobox', { name: 'Tree set' }), 0);
    const banner = await view.findByRole('status', { name: 'Convert older tree' }, { timeout: 10_000 });
    expect(banner.textContent).toContain('This is an older tree version, which may not be fully compatible with the current game version.');
    convert.mockClear();
    await fireEvent.click(within(banner).getByRole('button', { name: 'Convert to 3.29' }));
    await vi.waitFor(() => expect(convert).toHaveBeenCalledWith('3_29', false));
    const done = await view.findByRole('dialog', { name: 'Tree Converted' });
    expect(done.textContent).toContain('The tree has been converted to 3.29.');
    await fireEvent.click(within(done).getByRole('button', { name: 'OK' }));

    await fireEvent.click(within(banner).getByRole('button', { name: 'Convert all trees to 3.29' }));
    const all = await view.findByRole('dialog', { name: 'Convert all to Version 3.29' });
    expect(convertAll).not.toHaveBeenCalled();
    await fireEvent.click(within(all).getByRole('button', { name: 'Convert' }));
    await vi.waitFor(() => expect(convertAll).toHaveBeenCalledWith('3_29'));
    vi.restoreAllMocks();
    tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    vi.spyOn(workerInstance, 'GetTreeSpecsPoints').mockResolvedValue([]);
  }, 60_000);
});

// TreeTab.lua:39-46
describe('the Tree page switches and manages passive trees', () => {
  const specs = async () => {
    const l = (await workerInstance.GetTreeSpecs())!;
    return { active: l.active, titles: l.specs.map((s) => s.Title) };
  };
  const nodes = () => workerInstance.GetPassiveNodes().length;

  it('the select switches the active tree, with its nodes', async () => {
    await loadCorpusBuild(BUILD);
    const view = await mount();
    const select = await view.findByRole('combobox', { name: 'Tree set' }, { timeout: 10_000 });
    await vi.waitFor(async () => expect(await treeEntries(select)).toEqual(['Until Lab', 'Lab 2', 'White maps', 'Manage trees…']), { timeout: 10_000 });
    expect(select.textContent).toBe('Lab 2');
    const before = nodes();
    tick.mockClear();
    await chooseTree(select, 0);
    await vi.waitFor(async () => expect((await specs()).active).toBe(1), { timeout: 10_000 });
    expect(nodes(), 'tree 1 has its own allocation').not.toBe(before);
    expect(tick).toHaveBeenCalledWith('SetActiveTreeSpec');
  }, 60_000);

  // ItemsTab.lua:203-205
  it('a manage request from the Items page opens "Manage passive trees" on mount, once', async () => {
    await loadCorpusBuild(BUILD);
    const { treeManageRequest } = await import('$lib/shell/tree_manage');
    treeManageRequest.set(true);
    const view = await mount();
    const dlg = await view.findByRole('dialog', { name: 'Manage passive trees' }, { timeout: 10_000 });
    expect((dlg as HTMLDialogElement).open).toBe(true);
    await vi.waitFor(() =>
      expect(
        within(dlg)
          .getAllByRole('option')
          .map((o) => o.textContent?.replace(/( \(.*\))?(current)?$/, ''))
      ).toEqual(['Until Lab', 'Lab 2', 'White maps'])
    );
    expect(get(treeManageRequest)).toBe(false);
  }, 60_000);

  // TreeTab.lua:51-104
  it("hovering an entry of the tree select shows that tree's tooltip", async () => {
    await loadCorpusBuild(BUILD);
    const view = await mount();
    const compare = vi.spyOn(workerInstance, 'StatCompareTreeSpec').mockResolvedValue({ sections: [], calcMs: 0 });
    const respec = vi.spyOn(workerInstance, 'TreeSpecRespecCost').mockResolvedValue(undefined as never);
    try {
      const select = await view.findByRole('combobox', { name: 'Tree set' }, { timeout: 10_000 });
      await vi.waitFor(async () => expect(await treeEntries(select)).toHaveLength(4), { timeout: 10_000 });
      await fireEvent.click(select);
      await fireEvent.mouseEnter(within(select.parentElement!).getByRole('option', { name: 'White maps' }));
      await vi.waitFor(() => expect(view.getByLabelText('Tree tooltip').textContent).toContain('White maps'), { timeout: 10_000 });
      await vi.waitFor(() => expect(compare).toHaveBeenCalledWith(3));
      await fireEvent.mouseEnter(within(select.parentElement!).getByRole('option', { name: 'Until Lab' }));
      await vi.waitFor(() => expect(view.getByLabelText('Tree tooltip').textContent).toContain('Until Lab'), { timeout: 10_000 });
      await fireEvent.keyDown(select, { key: 'Escape' });
    } finally {
      compare.mockRestore();
      respec.mockRestore();
    }
  }, 60_000);

  // TreeTab.lua:367-383
  // PassiveSpecListControl.lua:82-87
  it('keys switch trees and open the list; a drag moves a tree', async () => {
    await loadCorpusBuild(BUILD);
    const view = await mount();
    const select = await view.findByRole('combobox', { name: 'Tree set' }, { timeout: 10_000 });
    await vi.waitFor(async () => expect(await treeEntries(select)).toHaveLength(4), { timeout: 10_000 });
    expect((await specs()).active).toBe(2);

    tick.mockClear();
    await fireEvent.keyDown(document.body, { key: 'ArrowDown' });
    await vi.waitFor(async () => expect((await specs()).active).toBe(3), { timeout: 10_000 });
    expect(tick).toHaveBeenCalledWith('SetActiveTreeSpec');
    await fireEvent.keyDown(document.body, { key: 'ArrowDown' });
    await new Promise((r) => setTimeout(r, 100));
    expect((await specs()).active).toBe(3);
    await fireEvent.keyDown(document.body, { key: 'ArrowUp' });
    await vi.waitFor(async () => expect((await specs()).active).toBe(2), { timeout: 10_000 });
    const search = view.getByRole('searchbox', { name: 'Search the tree' });
    await fireEvent.keyDown(search, { key: 'ArrowUp' });
    await new Promise((r) => setTimeout(r, 100));
    expect((await specs()).active).toBe(2);

    await fireEvent.keyDown(document.body, { key: 'f', ctrlKey: true });
    expect(document.activeElement).toBe(search);

    await fireEvent.keyDown(document.body, { key: 'm', ctrlKey: true });
    const list = await view.findByRole('listbox', { name: 'Passive trees' });
    await fireEvent.keyDown(document.body, { key: 'ArrowUp' });
    await new Promise((r) => setTimeout(r, 100));
    expect((await specs()).active).toBe(2);

    const first = within(list).getByRole('group', { name: 'Until Lab' });
    await fireEvent.keyDown(first, { key: 'ArrowDown', altKey: true });
    await vi.waitFor(async () => expect(await specs()).toEqual({ active: 1, titles: ['Lab 2', 'Until Lab', 'White maps'] }), { timeout: 10_000 });
  }, 60_000);

  // PassiveSpecListControl.lua:95-117
  it('Delete and F2 on a row; deleting an earlier tree keeps the current one', async () => {
    await loadCorpusBuild(BUILD);
    const view = await mount();
    const select = await view.findByRole('combobox', { name: 'Tree set' }, { timeout: 10_000 });
    await vi.waitFor(async () => expect(await treeEntries(select)).toHaveLength(4), { timeout: 10_000 });
    await chooseTree(select, 3);
    const list = await view.findByRole('listbox', { name: 'Passive trees' });
    const rowOf = (name: string) => [...list.querySelectorAll<HTMLElement>('[data-reorder-item]')].find((o) => o.textContent!.startsWith(name))!;
    await vi.waitFor(() => expect(rowOf('White maps')).toBeDefined());
    const before = await specs();
    const current = before.titles[before.active - 1];

    await fireEvent.keyDown(rowOf('Until Lab'), { key: 'F2' });
    expect((view.getByRole('textbox', { name: 'Name for this passive tree' }) as HTMLInputElement).value).toBe('Until Lab');
    await fireEvent.click(view.getByRole('button', { name: 'Cancel' }));

    await fireEvent.keyDown(rowOf('Until Lab'), { key: 'Delete' });
    const confirm = view.getByRole('group', { name: 'Confirm delete' });
    expect(confirm.textContent).toContain("'Until Lab'");
    await fireEvent.click(within(confirm).getByRole('button', { name: 'Delete' }));
    await vi.waitFor(async () => expect((await specs()).titles).toEqual(before.titles.filter((t) => t !== 'Until Lab')), { timeout: 10_000 });
    const after = await specs();
    expect(after.titles[after.active - 1], 'the current tree stays current').toBe(current);
  }, 60_000);

  it('new, copy, rename and delete through the manage list', async () => {
    await loadCorpusBuild(BUILD);
    const view = await mount();
    const select = await view.findByRole('combobox', { name: 'Tree set' }, { timeout: 10_000 });
    await vi.waitFor(async () => expect(await treeEntries(select)).toHaveLength(4), { timeout: 10_000 });
    await chooseTree(select, 3);
    const list = await view.findByRole('listbox', { name: 'Passive trees' });
    const row = (name: string) => [...list.querySelectorAll('[role="option"]')].find((o) => o.textContent!.startsWith(name)) as HTMLElement;
    const named = async (button: string, title: string) => {
      await fireEvent.click(view.getByRole('button', { name: button }));
      const box = view.getByRole('textbox', { name: 'Name for this passive tree' });
      await fireEvent.input(box, { target: { value: title } });
      await fireEvent.click(view.getByRole('button', { name: 'Save' }));
    };

    await named('Copy', 'Lab 2 again');
    await vi.waitFor(async () => expect((await specs()).titles).toEqual(['Until Lab', 'Lab 2', 'White maps', 'Lab 2 again']), { timeout: 10_000 });
    const xml = new TextDecoder().decode(types.builds.SerializeBuild(workerInstance.currentBuild).unwrap());
    const specNodes = [...xml.matchAll(/<Spec [^>]*\bnodes="([^"]*)"/g)].map((m) => m[1]);
    expect(specNodes[3], 'the copy has the original tree’s nodes').toBe(specNodes[1]);
    await fireEvent.click(row('Until Lab'));
    await named('Rename', 'Leveling');
    await vi.waitFor(async () => expect((await specs()).titles).toEqual(['Leveling', 'Lab 2', 'White maps', 'Lab 2 again']), { timeout: 10_000 });
    await fireEvent.click(view.getByRole('button', { name: 'Rename' }));
    await fireEvent.input(view.getByRole('textbox', { name: 'Name for this passive tree' }), { target: { value: '  ' } });
    expect((view.getByRole('button', { name: 'Save' }) as HTMLButtonElement).disabled).toBe(true);
    await fireEvent.click(view.getByRole('button', { name: 'Cancel' }));
    tick.mockClear();
    await vi.waitFor(() => expect(row('Lab 2 (')).toBeDefined());
    await fireEvent.click(row('Lab 2 ('));
    await fireEvent.click(view.getByRole('button', { name: 'Delete' }));
    await fireEvent.click(within(view.getByRole('group', { name: 'Confirm delete' })).getByRole('button', { name: 'Delete' }));
    await vi.waitFor(async () => expect(await specs()).toEqual({ active: 1, titles: ['Leveling', 'White maps', 'Lab 2 again'] }), { timeout: 10_000 });
    expect(tick).toHaveBeenCalledWith('DeleteTreeSpec');
  }, 60_000);
});

// TreeTab.lua:1038-1159
describe('the Tree page’s power report', () => {
  it('opens on Life and builds nothing until Start; a run of recalculations restarts it once, after they settle', async () => {
    vi.spyOn(workerInstance, 'GetPowerStats').mockReturnValue([
      { Stat: 'FullDPS', Label: 'Full DPS' },
      { Stat: 'Life', Label: 'Life' }
    ] as never);
    const report = vi.spyOn(workerInstance, 'BuildPowerReportLive').mockImplementation(() => new Promise(() => {}));
    vi.spyOn(workerInstance, 'PowerReportLive').mockReturnValue({ done: 0, current: 0 });
    outputs.set({} as never);
    const view = await mount();
    await fireEvent.click(view.getByRole('button', { name: /^Power/ }));
    const start = await vi.waitFor(() => view.getByRole('button', { name: 'Start' }));
    await new Promise((r) => setTimeout(r, 300));
    expect(report, 'opening the panel built the report').not.toHaveBeenCalled();
    await fireEvent.click(start);
    await vi.waitFor(() => expect(report).toHaveBeenCalledTimes(1));
    expect(report.mock.calls[0][0], 'the stat the panel opens on').toBe('Life');
    await vi.waitFor(() => expect((view.getByRole('combobox', { name: 'Power stat' }) as HTMLSelectElement).value).toBe('Life'));

    for (let i = 0; i < 3; i++) {
      outputs.set({} as never);
      await new Promise((r) => setTimeout(r, 100));
    }
    expect(report, 'restarted before the edits settled').toHaveBeenCalledTimes(1);
    await vi.waitFor(() => expect(report).toHaveBeenCalledTimes(2), { timeout: 3_000 });
    await new Promise((r) => setTimeout(r, 1_500));
    expect(report, 'one restart for the run').toHaveBeenCalledTimes(2);

    await fireEvent.change(view.getByRole('combobox', { name: 'Power stat' }), { target: { value: 'FullDPS' } });
    await vi.waitFor(() => expect(report).toHaveBeenCalledTimes(3), { timeout: 500 });
    expect(report.mock.calls[2][0]).toBe('FullDPS');
    vi.restoreAllMocks();
    tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    vi.spyOn(workerInstance, 'GetTreeSpecsPoints').mockResolvedValue([]);
    outputs.set(undefined);
  }, 60_000);

  it('Show node power, in the panel, offers Offence/Defence, and a Custom depth asks for that distance (TreeTab.lua:207-265)', async () => {
    vi.spyOn(workerInstance, 'GetPowerStats').mockReturnValue([{ Stat: 'Life', Label: 'Life' }] as never);
    const nodeStats = vi.spyOn(workerInstance, 'GetNodePowerStats').mockReturnValue([
      { Stat: '', Label: 'Offence/Defence', Combined: true },
      { Stat: 'Life', Label: 'Life' }
    ] as never);
    const report = vi.spyOn(workerInstance, 'BuildPowerReportLive').mockImplementation(() => new Promise(() => {}));
    vi.spyOn(workerInstance, 'PowerReportLive').mockReturnValue({ done: 0, current: 0 });
    outputs.set({} as never);
    const view = await mount();
    expect(view.queryByRole('checkbox', { name: /Show node power/i })).toBeNull();
    await fireEvent.click(view.getByRole('button', { name: /^Power/ }));
    const panel = within(await vi.waitFor(() => view.getByRole('region', { name: 'Power report' })));
    await fireEvent.click(panel.getByRole('checkbox', { name: 'Show node power' }));
    await fireEvent.click(panel.getByRole('button', { name: 'Start' }));
    await vi.waitFor(() => expect(nodeStats).toHaveBeenCalled());
    const stat = await vi.waitFor(() => {
      const el = view.getByRole('combobox', { name: 'Power stat' }) as HTMLSelectElement;
      expect(el.options.length).toBe(2);
      return el;
    });
    expect([...stat.options].map((o) => o.text)).toEqual(['Offence/Defence', 'Life']);
    await fireEvent.change(stat, { target: { value: '' } });
    await vi.waitFor(() => expect(report.mock.calls.at(-1)?.[0]).toBe(''));

    await fireEvent.change(view.getByRole('combobox', { name: 'Power report depth' }), { target: { value: '-1' } });
    const custom = await vi.waitFor(() => view.getByRole('textbox', { name: 'Custom node power depth' }));
    await fireEvent.change(custom, { target: { value: '7' } });
    await vi.waitFor(() => expect(report.mock.calls.at(-1)).toEqual(['', 7]));
    vi.restoreAllMocks();
    tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    vi.spyOn(workerInstance, 'GetTreeSpecsPoints').mockResolvedValue([]);
    outputs.set(undefined);
  }, 60_000);

  for (const how of ['the close button', 'the close button with Show node power on', 'leaving the page'] as const) {
    it(`${how} stops the report: no Step after it`, async () => {
      vi.spyOn(workerInstance, 'GetPowerStats').mockReturnValue([{ Stat: 'Life', Label: 'Life' }] as never);
      vi.spyOn(workerInstance, 'GetNodePowerStats').mockReturnValue([{ Stat: 'Life', Label: 'Life' }] as never);
      const w = workerInstance as unknown as { misc?: Promise<unknown>; miscCalculator: () => Promise<unknown> };
      const misc = Promise.resolve({ calc: {}, base: {} });
      w.misc = misc;
      vi.spyOn(w, 'miscCalculator').mockReturnValue(misc);
      let steps = 0;
      vi.spyOn(types.exposition, 'NewPowerBuilder').mockReturnValue({
        Step: () => (steps++, 0.001),
        Report: () => [],
        Current: () => 0,
        release: () => {}
      } as never);
      const runs: Promise<unknown>[] = [];
      const real = workerInstance.BuildPowerReportLive.bind(workerInstance);
      vi.spyOn(workerInstance, 'BuildPowerReportLive').mockImplementation((...a) => {
        const r = real(...a);
        runs.push(r);
        return r;
      });
      outputs.set({} as never);
      const view = await mount();
      await fireEvent.click(view.getByRole('button', { name: /^Power/ }));
      const panel = within(await vi.waitFor(() => view.getByRole('region', { name: 'Power report' })));
      if (how === 'the close button with Show node power on') await fireEvent.click(panel.getByRole('checkbox', { name: 'Show node power' }));
      await fireEvent.click(panel.getByRole('button', { name: 'Start' }));
      await vi.waitFor(() => expect(steps, 'the report is stepping').toBeGreaterThan(0));
      if (how === 'leaving the page') view.unmount();
      else await fireEvent.click(panel.getByRole('button', { name: 'Close power report' }));
      const at = steps;
      await new Promise((r) => setTimeout(r, 300));
      expect(steps, 'a Step ran after the close').toBe(at);
      expect(await Promise.all(runs), 'a run finished rather than stopped').toEqual(runs.map(() => undefined));
      w.misc = undefined;
      vi.restoreAllMocks();
      tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
      vi.spyOn(workerInstance, 'GetTreeSpecsPoints').mockResolvedValue([]);
      outputs.set(undefined);
    }, 90_000);
  }
});
