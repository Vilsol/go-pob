// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
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

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild, markBackendAsLoaded } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');
const Page = (await import('./+page.svelte')).default;

const BUILD = '0339-reliquarian-shield-crush-of-the-chieftain-2.xml';

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
  };
}

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD);
  get(markBackendAsLoaded)();
}, 120_000);

describe('calc detail follows the calcs page', () => {
  it('is off until the page mounts, on while it is mounted, and off again after', async () => {
    const tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    expect(workerInstance.calcDetail, 'detail was on before any page asked for it').toBe(false);

    render(Page);
    await vi.waitFor(() => expect(workerInstance.calcDetail).toBe(true), { timeout: 10_000 });
    expect(tick, 'mounting did not recalculate with detail').toHaveBeenCalledWith('setCalcTabElements');

    cleanup();
    await vi.waitFor(() => expect(workerInstance.calcDetail).toBe(false), { timeout: 10_000 });
    tick.mockRestore();
  }, 60_000);
});

// CalcsTab.lua:120-125
describe('the Calculation Mode select', () => {
  it('writes misc_buffMode to Go and Ticks', async () => {
    const { outputs } = await import('$lib/global');
    const { fireEvent, within, screen } = await import('@testing-library/svelte');
    const tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    outputs.set({ Output: {}, SkillFlags: {}, CalcsTab: { Output: {}, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {} } });
    render(Page);
    const mode = await vi.waitFor(() => screen.getByRole<HTMLSelectElement>('combobox', { name: 'Calculation Mode' }), { timeout: 10_000 });
    expect(mode.value).toBe('Effective DPS');
    tick.mockClear();

    await fireEvent.change(mode, { target: { value: 'In Combat' } });
    await vi.waitFor(() => expect(workerInstance.GetCalcsTabInputs().buffMode).toBe('COMBAT'));
    expect(tick.mock.calls.map((c) => c[0])).toContain('SetCalcsBuffMode');
    void within;
    tick.mockRestore();
  }, 60_000);
});

// CalcsTab.lua:87-91
describe('Show Minion Stats', () => {
  it('appears for a minion skill and writes showMinion to Go', async () => {
    const { outputs } = await import('$lib/global');
    const { fireEvent, screen } = await import('@testing-library/svelte');
    const tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    outputs.set({
      Output: {},
      SkillFlags: { haveMinion: true },
      CalcsTab: { Output: {}, OutputTable: {}, SkillFlags: { haveMinion: true }, Breakdown: {}, Calcs: {} }
    });
    render(Page);
    const box = await vi.waitFor(() => screen.getByRole<HTMLInputElement>('checkbox', { name: 'Show Minion Stats' }), { timeout: 10_000 });
    expect(box.checked).toBe(workerInstance.GetCalcsTabInputs().showMinion);
    const before = box.checked;
    tick.mockClear();

    await fireEvent.click(box);
    await vi.waitFor(() => expect(workerInstance.GetCalcsTabInputs().showMinion).toBe(!before));
    expect(tick.mock.calls.map((c) => c[0])).toContain('SetCalcsShowMinion');
    tick.mockRestore();
  }, 60_000);

  it('is not drawn without a minion', async () => {
    const { outputs } = await import('$lib/global');
    const { screen } = await import('@testing-library/svelte');
    vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    outputs.set({ Output: {}, SkillFlags: {}, CalcsTab: { Output: {}, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {} } });
    render(Page);
    await vi.waitFor(() => screen.getByRole('combobox', { name: 'Calculation Mode' }), { timeout: 10_000 });
    expect(screen.queryByRole('checkbox', { name: 'Show Minion Stats' })).toBeNull();
  }, 60_000);
});

// CalcsTab.lua:40-59
describe('the Calcs-local skill controls', () => {
  const controls = (main: number) => ({ SkillSet: 0, SocketGroup: 0, GemIndex: 0, MainActiveSkill: main, Skills: ['First', 'Second'] });

  it('the socket group select writes skill_number, not the main socket group', async () => {
    const { outputs } = await import('$lib/global');
    const { fireEvent, screen } = await import('@testing-library/svelte');
    const tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    outputs.set({ Output: {}, SkillFlags: {}, CalcsTab: { Output: {}, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {} } });
    render(Page);
    const group = await vi.waitFor(() => screen.getByRole<HTMLSelectElement>('combobox', { name: 'Socket Group' }), { timeout: 10_000 });
    await vi.waitFor(() => expect(group.options.length).toBeGreaterThan(2), { timeout: 10_000 });
    const mainBefore = workerInstance.GetBuildInfo()?.mainSocketGroup;
    tick.mockClear();

    await fireEvent.change(group, { target: { value: '2' } });
    await vi.waitFor(() => expect(workerInstance.GetCalcsTabInputs().skillNumber).toBe(3));
    expect(workerInstance.GetBuildInfo()?.mainSocketGroup, 'the sidebar main group moved').toBe(mainBefore);
    expect(tick.mock.calls.map((c) => c[0])).toContain('SetCalcsSkillNumber');
    tick.mockRestore();
  }, 60_000);

  it("Skill Stages and Active Mines write the Calcs tab's own counts to the gem in Go", async () => {
    const { outputs } = await import('$lib/global');
    const { fireEvent, screen } = await import('@testing-library/svelte');
    const tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    const shown = { ...controls(1), StageShown: true, StageCount: '1', MineShown: true, MineCount: '' };
    outputs.set({ Output: {}, SkillFlags: {}, CalcsTab: { Output: {}, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {}, MainSkill: shown } } as never);
    render(Page);
    const gem = () => workerInstance.currentBuild?.Skills?.SkillSets?.[0]?.Skills?.[0]?.Gems?.[0];
    const stages = await vi.waitFor(() => screen.getByRole<HTMLInputElement>('textbox', { name: 'Skill Stages' }), { timeout: 10_000 });
    tick.mockClear();
    await fireEvent.input(stages, { target: { value: '7' } });
    await vi.waitFor(() => expect(gem()?.SkillStageCountCalcs).toBe(7));
    expect(gem()?.SkillStageCount, "the sidebar's count moved").not.toBe(7);
    expect(tick.mock.calls.map((c) => c[0])).toContain('SetGemSkillStageCountCalcs');

    await fireEvent.input(screen.getByRole<HTMLInputElement>('textbox', { name: 'Active Mines' }), { target: { value: '3' } });
    await vi.waitFor(() => expect(gem()?.SkillMineCountCalcs).toBe(3));
    expect(tick.mock.calls.map((c) => c[0])).toContain('SetGemSkillMineCountCalcs');
    tick.mockRestore();
  }, 60_000);

  it('Manage Spectres... opens the spectre library from Go', async () => {
    const { outputs } = await import('$lib/global');
    const { fireEvent, screen } = await import('@testing-library/svelte');
    vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    const library = vi.spyOn(workerInstance, 'GetSpectreLibrary');
    outputs.set({
      Output: {},
      SkillFlags: {},
      CalcsTab: { Output: {}, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {}, MainSkill: { ...controls(1), MinionLibraryShown: true } }
    } as never);
    render(Page);
    const button = await vi.waitFor(() => screen.getByRole('button', { name: 'Manage Spectres...' }), { timeout: 10_000 });
    await fireEvent.click(button);
    await vi.waitFor(() => expect(library).toHaveBeenCalled());
    await vi.waitFor(() => screen.getByRole('listbox', { name: 'Available Spectres' }));
    library.mockRestore();
  }, 60_000);

  it("hovering the socket group select draws that group's tooltip from the worker", async () => {
    const { outputs } = await import('$lib/global');
    const { fireEvent, screen } = await import('@testing-library/svelte');
    vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    const tip = vi.spyOn(workerInstance, 'SocketGroupTooltip').mockResolvedValue([{ Text: '^7Active Skill #1:', Separator: false }]);
    outputs.set({ Output: {}, SkillFlags: {}, CalcsTab: { Output: {}, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {} } });
    render(Page);
    const group = await vi.waitFor(() => screen.getByRole<HTMLSelectElement>('combobox', { name: 'Socket Group' }), { timeout: 10_000 });
    await vi.waitFor(() => expect(group.options.length).toBeGreaterThan(2), { timeout: 10_000 });
    await vi.waitFor(() => expect(Number(group.value)).toBe(workerInstance.GetCalcsTabInputs().skillNumber - 1));
    await fireEvent.mouseEnter(group.closest('div.relative')!);
    await vi.waitFor(() => expect(tip).toHaveBeenCalledWith(Number(group.value)));
    await vi.waitFor(() => expect(screen.getByRole('tooltip', { name: 'Socket group tooltip' }).textContent).toContain('Active Skill #1:'));
    tip.mockRestore();
  }, 60_000);

  it('the socket group tooltip follows a group picked under a still pointer', async () => {
    const { outputs } = await import('$lib/global');
    const { fireEvent, screen } = await import('@testing-library/svelte');
    vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    const tip = vi.spyOn(workerInstance, 'SocketGroupTooltip').mockImplementation((i: number) => Promise.resolve([{ Text: `^7Group ${i}`, Separator: false }]));
    const empty = () => ({ Output: {}, SkillFlags: {}, CalcsTab: { Output: {}, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {} } });
    outputs.set(empty());
    render(Page);
    const group = await vi.waitFor(() => screen.getByRole<HTMLSelectElement>('combobox', { name: 'Socket Group' }), { timeout: 10_000 });
    await vi.waitFor(() => expect(group.options.length).toBeGreaterThan(2), { timeout: 10_000 });
    const from = Number(group.value);
    const to = from === 0 ? 1 : 0;
    await fireEvent.mouseEnter(group.closest('div.relative')!);
    const shown = () => screen.getByRole('tooltip', { name: 'Socket group tooltip' });
    await vi.waitFor(() => expect(shown().textContent).toContain(`Group ${from}`));
    await fireEvent.change(group, { target: { value: String(to) } });
    outputs.set(empty());
    await vi.waitFor(() => expect(shown().textContent).toContain(`Group ${to}`), { timeout: 10_000 });
    await fireEvent.change(group, { target: { value: String(from) } });
    tip.mockRestore();
  }, 60_000);

  it("a row's breakdown asks the worker for its modifier tables (CalcBreakdownControl:AddModSection) and draws them", async () => {
    const { outputs } = await import('$lib/global');
    const { fireEvent, screen, within } = await import('@testing-library/svelte');
    vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    const rows = vi.spyOn(workerInstance, 'CalcsModSection').mockReturnValue([
      {
        Value: '+42 base',
        Name: '',
        Flags: '',
        Tags: '',
        Source: 'Item',
        SourceName: 'Stub Ring',
        Totals: ['Total from Item:', '+42 base'],
        ItemName: 'Stub Ring',
        NodeID: 0
      }
    ]);
    outputs.set({ Output: {}, SkillFlags: {}, CalcsTab: { Output: { Life: 100 }, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {} } } as never);
    render(Page);
    await vi.waitFor(() => expect(document.querySelectorAll('[role="row"][data-key]').length).toBeGreaterThan(0), { timeout: 10_000 });
    for (const row of document.querySelectorAll<HTMLElement>('[role="row"][data-key]')) {
      await fireEvent.mouseEnter(row);
      if (rows.mock.calls.length) break;
    }
    expect(rows).toHaveBeenCalled();
    await vi.waitFor(() => expect(within(screen.getByRole('complementary', { name: 'Breakdown' })).getAllByText('Stub Ring').length).toBeGreaterThan(0));

    // CalcBreakdownControl.lua:400-403
    const itemTip = vi
      .spyOn(workerInstance, 'ItemSourceTooltip')
      .mockResolvedValue({ Influence: [], Title: ['^xAF6025Stub Ring'], Lines: [{ Text: 'Ring of stubs', Separator: false }], Skills: [] } as never);
    const cell = within(screen.getByRole('complementary', { name: 'Breakdown' }))
      .getAllByText('Stub Ring')[0]
      .closest('[role="cell"]')!;
    await fireEvent.mouseEnter(cell);
    await vi.waitFor(() => expect(itemTip).toHaveBeenCalledWith('Stub Ring'));
    await vi.waitFor(() => expect(document.body.textContent).toContain('Ring of stubs'));
    itemTip.mockRestore();
    rows.mockRestore();
  }, 60_000);

  it("a Tree source's name shows the node's place on the tree (CalcBreakdownControl.lua:405-413, 594-612)", async () => {
    const { outputs } = await import('$lib/global');
    const { fireEvent, screen, within } = await import('@testing-library/svelte');
    vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    const rows = vi
      .spyOn(workerInstance, 'CalcsModSection')
      .mockReturnValue([{ Value: '+10', Name: '', Flags: '', Tags: '', Source: 'Tree', SourceName: 'Stub Notable', Totals: [], ItemName: '', NodeID: 1234 }]);
    const version = vi.spyOn(workerInstance, 'GetTreeVersion').mockResolvedValue('');
    outputs.set({ Output: {}, SkillFlags: {}, CalcsTab: { Output: { Life: 100 }, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {} } } as never);
    render(Page);
    await vi.waitFor(() => expect(document.querySelectorAll('[role="row"][data-key]').length).toBeGreaterThan(0), { timeout: 10_000 });
    for (const row of document.querySelectorAll<HTMLElement>('[role="row"][data-key]')) {
      await fireEvent.mouseEnter(row);
      if (rows.mock.calls.length) break;
    }
    const cell = await vi.waitFor(
      () =>
        within(screen.getByRole('complementary', { name: 'Breakdown' }))
          .getAllByText('Stub Notable')[0]
          .closest('[role="cell"]')!
    );
    await fireEvent.mouseEnter(cell);
    await vi.waitFor(() => expect(screen.getByRole('img', { name: 'Node location' }).dataset.node).toBe('1234'));
    await fireEvent.mouseLeave(cell);
    await vi.waitFor(() => expect(screen.queryByRole('img', { name: 'Node location' })).toBeNull());
    version.mockRestore();
    rows.mockRestore();
  }, 60_000);

  it("the active skill select shows and writes the Calcs tab's own control, not the sidebar's", async () => {
    const { outputs } = await import('$lib/global');
    const { fireEvent, screen } = await import('@testing-library/svelte');
    const tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    outputs.set({
      Output: {},
      SkillFlags: {},
      MainSkill: controls(1),
      CalcsTab: { Output: {}, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {}, MainSkill: controls(2) }
    } as never);
    render(Page);
    const active = await vi.waitFor(() => screen.getByRole<HTMLSelectElement>('combobox', { name: 'Active Skill' }), { timeout: 10_000 });
    expect(active.value).toBe('2');
    tick.mockClear();

    await fireEvent.change(active, { target: { value: '1' } });
    await vi.waitFor(() => expect(tick.mock.calls.map((c) => c[0])).toContain('SetMainActiveSkillCalcs'));
    tick.mockRestore();
  }, 60_000);
});

// CalcsTab.lua:135-137, Calcs.lua:634-828
describe('the buff, combat and curse lists', () => {
  it("show the Calcs pass's lists by the Calculation Mode", async () => {
    const { outputs } = await import('$lib/global');
    const { screen, within, fireEvent } = await import('@testing-library/svelte');
    const tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    const strings = { BuffList: 'Herald of Ash, Wrath', CombatList: '3 Power Charges, Onslaught', CurseList: 'Flammability' };
    outputs.set({ Output: {}, SkillFlags: {}, CalcsTab: { Output: {}, OutputTable: {}, OutputStrings: strings, SkillFlags: {}, Breakdown: {}, Calcs: {} } });
    workerInstance.SetCalcsBuffMode('EFFECTIVE');
    render(Page);
    const dl = await vi.waitFor(() => document.querySelector<HTMLElement>('dl[aria-label="Buffs and debuffs"]')!, { timeout: 10_000 });
    expect(dl).not.toBeNull();
    const rows = () =>
      [...within(document.querySelector<HTMLElement>('dl[aria-label="Buffs and debuffs"]') ?? dl).queryAllByRole('term')].map(
        (t) => `${t.textContent}: ${t.nextElementSibling?.textContent}`
      );
    expect(rows()).toEqual(['Aura and Buff Skills: Herald of Ash, Wrath', 'Combat Buffs: 3 Power Charges, Onslaught', 'Curses and Debuffs: Flammability']);
    await new Promise((r) => setTimeout(r, 200));
    await fireEvent.change(screen.getByRole<HTMLSelectElement>('combobox', { name: 'Calculation Mode' }), { target: { value: 'Buffed' } });
    await vi.waitFor(() => expect(rows()).toEqual(['Aura and Buff Skills: Herald of Ash, Wrath']));
    workerInstance.SetCalcsBuffMode('EFFECTIVE');
    tick.mockRestore();
  }, 60_000);

  // CalcsTab.lua:135, 137
  // CalcBreakdownControl.lua:281-282
  it('a list line shows its breakdown modifier table', async () => {
    const { outputs } = await import('$lib/global');
    const { screen, within, fireEvent } = await import('@testing-library/svelte');
    const tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    const rows = vi
      .spyOn(workerInstance, 'CalcsBreakdownModList')
      .mockReturnValue([
        { Value: '40% increased', Name: 'Fire Damage', Flags: '', Tags: '', Source: 'Skill', SourceName: 'Anger', Totals: [], ItemName: '', NodeID: 0 }
      ]);
    const strings = { BuffList: 'Anger', CombatList: '', CurseList: 'Flammability' };
    outputs.set({ Output: {}, SkillFlags: {}, CalcsTab: { Output: {}, OutputTable: {}, OutputStrings: strings, SkillFlags: {}, Breakdown: {}, Calcs: {} } });
    workerInstance.SetCalcsBuffMode('EFFECTIVE');
    render(Page);
    const line = await vi.waitFor(() => screen.getByRole('button', { name: 'Aura and Buff Skills' }), { timeout: 10_000 });
    expect(screen.queryByRole('button', { name: 'Combat Buffs' }), 'Combat Buffs has no breakdown').toBeNull();
    await fireEvent.mouseEnter(line);
    await vi.waitFor(() => expect(rows).toHaveBeenCalledWith('SkillBuffs'));
    const panel = screen.getByRole('complementary', { name: 'Breakdown' });
    await vi.waitFor(() => expect(within(panel).getByText('Fire Damage')).toBeTruthy());
    expect(within(panel).getByText('40% increased')).toBeTruthy();
    rows.mockRestore();
    tick.mockRestore();
  }, 60_000);
});

// CalcsTab.lua:174-189, 208-216
describe('section collapse', () => {
  it('a toggle reaches Go, and a saved state is drawn on mount', async () => {
    const { outputs } = await import('$lib/global');
    const { screen, fireEvent } = await import('@testing-library/svelte');
    const tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    outputs.set({ Output: {}, SkillFlags: {}, CalcsTab: { Output: {}, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {} } });
    const first = render(Page);
    await new Promise((r) => setTimeout(r, 200));
    const toggle = await vi.waitFor(
      () => screen.getAllByRole<HTMLButtonElement>('button', { expanded: true }).find((b) => !b.disabled && b.closest('section')?.getAttribute('aria-label'))!,
      { timeout: 10_000 }
    );
    const label = toggle.closest('section')!.getAttribute('aria-label')!;
    await fireEvent.click(toggle);
    const { calcSections } = await import('$lib/calcs/calc_sections');
    const keys = calcSections.map((s) => [s[1], (s[4][0]?.label ?? '').replace(/[^A-Za-z0-9]/g, '')] as [string, string]);
    const id = calcSections.find((s) => s[4][0]?.label === label)![1];
    await vi.waitFor(() => expect(workerInstance.GetCalcsSectionsCollapsed(keys)[keys.findIndex(([k]) => k === id)]).toBe(true));
    expect(toggle.getAttribute('aria-expanded')).toBe('false');
    first.unmount();

    render(Page);
    await vi.waitFor(() => {
      const again = screen.getByRole('region', { name: label }).querySelector<HTMLButtonElement>('h2 button')!;
      expect(again.getAttribute('aria-expanded')).toBe('false');
    });
    tick.mockRestore();
  }, 60_000);
});

// CalcSectionControl.lua:56-61, 223-356
describe('section pop-out', () => {
  it('pops a section out of the columns, raises and closes it', async () => {
    const { outputs } = await import('$lib/global');
    const { screen, fireEvent } = await import('@testing-library/svelte');
    const tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    outputs.set({ Output: {}, SkillFlags: {}, CalcsTab: { Output: {}, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {} } });
    render(Page);
    const pops = await vi.waitFor(() => {
      const b = screen.getAllByRole('button', { name: /^Pop out / });
      expect(b.length).toBeGreaterThan(1);
      return b;
    });
    const title = pops[0].getAttribute('aria-label')!.replace(/^Pop out /, '');
    const second = pops[1].getAttribute('aria-label')!.replace(/^Pop out /, '');
    await fireEvent.click(pops[0]);
    const pane = await screen.findByRole('dialog', { name: `${title} (popped out)` });
    expect(screen.queryByRole('region', { name: title }), 'the section stayed in its column').toBeNull();
    await fireEvent.click(screen.getByRole('button', { name: `Pop out ${second}` }));
    const panes = () => screen.getAllByRole('dialog').map((d) => d.getAttribute('aria-label'));
    expect(panes()).toEqual([`${title} (popped out)`, `${second} (popped out)`]);
    await fireEvent.pointerDown(pane);
    expect(panes()).toEqual([`${second} (popped out)`, `${title} (popped out)`]);
    await fireEvent.click(screen.getByRole('button', { name: `Close ${title}` }));
    await vi.waitFor(() => expect(screen.getByRole('region', { name: title })).toBeTruthy());
    expect(panes()).toEqual([`${second} (popped out)`]);
    tick.mockRestore();
  }, 60_000);
});

// Calcs.lua:37-70
describe('the mod debug dump', () => {
  it('is hidden outside Dev mode and, in it, shows the worker dump on request', async () => {
    const { outputs, devMode } = await import('$lib/global');
    const { screen, fireEvent } = await import('@testing-library/svelte');
    const tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
    const dump = vi.spyOn(workerInstance, 'GetModDebugDump').mockResolvedValue("=== Modifiers ===\n'Life':\n");
    outputs.set({ Output: {}, SkillFlags: {}, CalcsTab: { Output: {}, OutputTable: {}, SkillFlags: {}, Breakdown: {}, Calcs: {} } });
    devMode.set(false);
    render(Page);
    await screen.findByRole('combobox', { name: 'Calculation Mode' });
    expect(screen.queryByRole('button', { name: 'Mod Debug' }), 'drawn outside Dev mode').toBeNull();
    cleanup();

    devMode.set(true);
    render(Page);
    await fireEvent.click(await screen.findByRole('button', { name: 'Mod Debug' }));
    const pre = await screen.findByLabelText('Mod debug');
    expect(pre.textContent).toBe("=== Modifiers ===\n'Life':\n");
    expect(dump).toHaveBeenCalledTimes(1);
    await fireEvent.click(screen.getByRole('button', { name: 'Mod Debug' }));
    expect(screen.queryByLabelText('Mod debug'), 'a second click did not close it').toBeNull();
    devMode.set(false);
    dump.mockRestore();
    tick.mockRestore();
  }, 60_000);
});
