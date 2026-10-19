// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi, type MockInstance } from 'vitest';
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

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild, outputs } = await import('$lib/global');
const { browser } = await import('$app/environment');
const { workerInstance } = await import('$lib/go/sync_worker');
const { configurations, configRelevanceQuery } = await import('$lib/display/configurations');
const { formatColors } = await import('$lib/display/colors');

const BUILD = '0008-necromancer-vaal-summon-skeletons.xml';

const plain = (s: string) =>
  formatColors(s)
    .replace(/<[^>]*>/g, '')
    .trim();

const check = configurations[0].variables.find((v) => v.type === 'check' && !Object.keys(v).some((k) => k.startsWith('if')) && v.defaultState === undefined)!;

const build = () => workerInstance.currentBuild!;
const inputs = () => Object.keys(workerInstance.GetAllConfigOptions()).sort();
const blocks = () => workerInstance.GetCustomModifierBlocks().map((b) => ({ Title: b.Title, Enabled: b.Enabled, Text: b.Text }));

let tick: MockInstance;
const stubStatus = (line: string) =>
  line.trim() === '' ? 'blank' : line.trim().startsWith('#') ? 'comment' : line.includes('Life') ? 'parsed' : 'unsupported';

const mount = async () => {
  const Page = (await import('./+page.svelte')).default;
  const view = render(Page);
  await view.findAllByRole('textbox', { name: /Modifier line/ });
  return view;
};

const statusOf = (cell: HTMLElement) => cell.closest<HTMLElement>('.mod-cell')!.dataset.k;
const row = (view: { container: HTMLElement }, v: string) =>
  [...view.container.querySelectorAll(`[data-var="${v}"]`)].find((el) => !el.closest('[aria-label="Most used"]')) ?? null;

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
  vi.spyOn(workerInstance, 'ParseModLines').mockImplementation((lines: string[]) =>
    lines.map((l) => ({ Status: stubStatus(l), Extra: '', Mods: [] }) as never)
  );
  outputs.set(undefined);
}, 120_000);

describe('the harness is not vacuous', () => {
  it('runs the browser path against a real build', () => {
    expect(browser).toBe(true);
    expect(workerInstance.booted).toBe(true);
    expect(typeof build().SetCustomModifierBlocks).toBe('function');
    expect(get(currentBuild)).toBeDefined();
    expect(check, 'no ungated General checkbox to drive').toBeDefined();
    expect(blocks()[0].Text.trim().length, `${BUILD} has no custom modifier text`).toBeGreaterThan(0);
  });
});

describe('configuration/+page.svelte', () => {
  it('mounting alone writes nothing and does not Tick', async () => {
    const before = { inputs: inputs(), blocks: blocks() };
    tick.mockClear();
    await mount();
    await new Promise((r) => setTimeout(r, 50));
    expect({ inputs: inputs(), blocks: blocks() }).toEqual(before);
    expect(tick).not.toHaveBeenCalled();
  }, 20_000);

  it('shows each non-blank line of the group as a cell, with its verdict', async () => {
    const view = await mount();
    const cells = await view.findAllByRole('textbox', { name: /Modifier line/ });
    const text = blocks()[0].Text;
    const lines = text.split('\n');
    const first = lines.findIndex((l) => l.trim() !== '');
    const last = lines.length - 1 - [...lines].reverse().findIndex((l) => l.trim() !== '');
    expect(cells.map((c) => (c as HTMLInputElement).value)).toEqual(lines.slice(first, last + 1));
    await vi.waitFor(() =>
      expect(cells.map(statusOf)).toEqual(
        lines.slice(first, last + 1).map((l) => ({ parsed: 'ok', unsupported: 'bad', comment: 'comment', blank: 'empty' })[stubStatus(l)])
      )
    );
  });

  it('an edited cell reaches Go as the block text, changing only that line, then Ticks', async () => {
    const before = blocks()[0].Text;
    const view = await mount();
    const cell = (await view.findByRole('textbox', { name: 'Modifier line 2' })) as HTMLInputElement;
    const old = cell.value;
    await fireEvent.input(cell, { target: { value: '+50 to maximum Life' } });
    await vi.waitFor(() => expect(statusOf(cell)).toBe('ok'));
    expect(blocks()[0].Text, 'typing alone must not write').toBe(before);
    tick.mockClear();
    await fireEvent.change(cell, { target: { value: '+50 to maximum Life' } });
    await vi.waitFor(() => expect(blocks()[0].Text).toBe(before.replace(old, '+50 to maximum Life')));
    expect(build().GetStringOption('customMods')).toBe('');
    await vi.waitFor(() => expect(tick).toHaveBeenCalledWith('SetCustomModifierBlocks'));
  });

  it('adding a group, disabling it and removing a line reach Go', async () => {
    const view = await mount();
    const groupsBefore = blocks().length;
    await fireEvent.click(view.getByRole('button', { name: '+ New group' }));
    await vi.waitFor(() => expect(blocks().length).toBe(groupsBefore + 1));
    const list = within(view.getByRole('list', { name: 'Mod groups' }));
    const added = list.getAllByRole('listitem').at(-1)!;
    await fireEvent.click(within(added).getByRole('checkbox', { name: /^Enable / }));
    await vi.waitFor(() => expect(blocks().at(-1)!.Enabled).toBe('false'));

    const lineCount = blocks()[0].Text.split('\n').length;
    await fireEvent.click(within(list.getAllByRole('listitem')[0]).getByRole('button'));
    await fireEvent.click(view.getByRole('button', { name: 'Remove line 1' }));
    await vi.waitFor(() => expect(blocks()[0].Text.split('\n').length).toBe(lineCount - 1));
  });

  it('a toggled option is marked, counted, filterable and reset', async () => {
    build().ResetConfigOptions();
    const view = await mount();
    const box = (await view.findAllByRole('checkbox', { name: plain(check.label) }))[0] as HTMLInputElement;

    tick.mockClear();
    await fireEvent.click(box);
    await vi.waitFor(() => expect(build().GetBooleanOption(check.var)).toBe(true));
    await vi.waitFor(() => expect(tick).toHaveBeenCalled());
    await vi.waitFor(() => expect(view.getAllByRole('checkbox', { name: `${plain(check.label)} (changed)` }).length).toBeGreaterThan(0));
    expect(within(view.getByRole('region', { name: 'General' })).getByText(/· 1 modified$/)).toBeDefined();
    expect(view.getByText(/· 1 modified ·/, { selector: 'span[aria-live]' })).toBeDefined();

    await fireEvent.click(view.getByRole('checkbox', { name: 'Modified only' }));
    await vi.waitFor(() => expect(row(view, 'resistancePenalty')).toBeNull());
    expect(row(view, check.var)).not.toBeNull();

    tick.mockClear();
    await fireEvent.click(view.getByRole('button', { name: 'Reset to defaults' }));
    await fireEvent.click(view.getByRole('button', { name: 'Confirm reset' }));
    await vi.waitFor(() => expect(build().GetBooleanOption(check.var)).toBe(false));
    expect(inputs().filter((n) => n !== 'customMods')).toEqual([]);
    expect(tick).toHaveBeenCalledWith('ResetConfigOptions');
    await vi.waitFor(() => expect(view.getByText(/· 0 modified ·/, { selector: 'span[aria-live]' })).toBeDefined());
  });

  it('shows a gated option only once Go reports it relevant', async () => {
    const gated = configurations
      .flatMap((s) => s.variables)
      .find(
        (v) => v.type === 'check' && typeof v.ifCond === 'string' && Object.keys(v).filter((k) => k.startsWith('if') || k.startsWith('imply')).length === 1
      )!;
    expect(gated).toBeDefined();
    outputs.set(undefined);
    const view = await mount();
    expect(row(view, gated.var)).toBeNull();
    const none = Object.fromEntries(Object.keys(configRelevanceQuery).map((k) => [k, []]));
    outputs.set({ ConfigRelevance: { ...none, Conds: [gated.ifCond as string] } as never });
    await vi.waitFor(() => expect(row(view, gated.var)).not.toBeNull());
    outputs.set(undefined);
  });

  it('emptying a number option drops it from Go', async () => {
    build().ResetConfigOptions();
    workerInstance.SetConfigOption('enemyLevel', 70);
    expect(inputs()).toContain('enemyLevel');
    const view = await mount();
    const box = (await view.findAllByRole('spinbutton', { name: /^Enemy Level/ }))[0] as HTMLInputElement;
    await vi.waitFor(() => expect(box.value).toBe('70'));
    tick.mockClear();
    await fireEvent.input(box, { target: { value: '' } });
    await fireEvent.blur(box);
    await vi.waitFor(() => expect(inputs()).not.toContain('enemyLevel'));
    await vi.waitFor(() => expect(tick).toHaveBeenCalled());
  });

  // ConfigOptions.lua:135
  it('a list choice that is not the default index survives, and choosing the default drops it', async () => {
    build().ResetConfigOptions();
    const view = await mount();
    const select = (await view.findAllByRole('combobox', { name: 'Resistance penalty:' }))[0];
    expect(select.textContent).toBe('Act 10 (-60%)');
    const choose = async (i: number) => {
      await fireEvent.click(select);
      await fireEvent.click(within(view.getByRole('listbox')).getAllByRole('option')[i]);
    };

    await choose(0);
    await vi.waitFor(() => expect(build().GetNumberOptionOK('resistancePenalty')).toEqual([0, true]));

    await choose(2);
    await vi.waitFor(() => expect(build().GetNumberOptionOK('resistancePenalty')).toEqual([0, false]));
  });
});

// ConfigOptions.lua:1457-1589
describe('enemy placeholders', () => {
  it('an Uber boss shows its level and damage placeholders in the enemy boxes', async () => {
    const view = await mount();
    const boss = row(view, 'enemyIsBoss')!;
    const select = within(boss as HTMLElement).getByRole('combobox');
    const level = () => within(row(view, 'enemyLevel') as HTMLElement).getByRole<HTMLInputElement>('spinbutton');
    await vi.waitFor(() => expect(level().placeholder).toBe('84'), { timeout: 5000 });
    await fireEvent.click(select);
    await fireEvent.click(within(boss as HTMLElement).getByRole('option', { name: 'Uber Pinnacle Boss' }));
    await vi.waitFor(() => expect((workerInstance.GetAllConfigOptions() as Record<string, unknown>).enemyIsBoss).toBe('Uber'));
    await vi.waitFor(() => expect(level().placeholder).toBe('85'), { timeout: 5000 });
    const phys = () => within(row(view, 'enemyPhysicalDamage') as HTMLElement).getByRole<HTMLInputElement>('spinbutton');
    expect(Number(phys().placeholder)).toBeGreaterThan(7);
  }, 20_000);
});

describe('collapsible sections', () => {
  it("hides a collapsed section's options until it is expanded or a search is typed", async () => {
    const view = await mount();
    const general = view.getByRole('region', { name: 'General' });
    const toggle = within(general).getByRole('button', { name: 'Collapse General' });
    expect(toggle.getAttribute('aria-expanded')).toBe('true');
    expect(row(view, check.var)).not.toBeNull();

    await fireEvent.click(toggle);
    const expand = within(general).getByRole('button', { name: 'Expand General' });
    expect(expand.getAttribute('aria-expanded')).toBe('false');
    expect(row(view, check.var)).toBeNull();

    const search = view.getByRole('searchbox', { name: 'Filter options' });
    await fireEvent.input(search, { target: { value: plain(check.label) } });
    await vi.waitFor(() => expect(row(view, check.var)).not.toBeNull());
    await fireEvent.input(search, { target: { value: ' ' } });
    await vi.waitFor(() => expect(row(view, check.var)).toBeNull());

    await fireEvent.click(expand);
    expect(row(view, check.var)).not.toBeNull();
  });
});

describe('Ctrl+F (ConfigTab.lua:988-989)', () => {
  it('focuses the filter box', async () => {
    const view = await mount();
    const box = view.getByRole('searchbox', { name: 'Filter options' });
    expect(document.activeElement).not.toBe(box);
    await fireEvent.keyDown(window, { key: 'f', ctrlKey: true });
    expect(document.activeElement).toBe(box);
  });
});

describe('the custom group enable checkbox tooltip (ConfigTab.lua:53-68)', () => {
  it("asks the worker for the group switched the other way and shows it under PoB's header", async () => {
    const view = await mount();
    const compare = vi.spyOn(workerInstance, 'CustomModGroupCompare').mockResolvedValue([]);
    try {
      const box = within(view.getByRole('list', { name: 'Mod groups' })).getAllByRole('checkbox')[0];
      await fireEvent.mouseOver(box);
      await vi.waitFor(() => expect(compare).toHaveBeenCalledWith(0));
      const enabled = (box as HTMLInputElement).checked;
      await vi.waitFor(() =>
        expect(view.getByRole('tooltip', { name: 'Group toggle what-if' }).textContent).toContain(
          enabled ? 'Disabling this group will give you:' : 'Enabling this group will give you:'
        )
      );
      await fireEvent.mouseOut(box);
      await vi.waitFor(() => expect(view.queryByRole('tooltip', { name: 'Group toggle what-if' })).toBeNull());
    } finally {
      compare.mockRestore();
    }
  });
});

describe('the custom group enable checkbox tooltip under a still pointer', () => {
  it('follows the toggle: the header at once, the what-if once the toggle is calculated', async () => {
    const view = await mount();
    let calculated = false;
    const compare = vi
      .spyOn(workerInstance, 'CustomModGroupCompare')
      .mockImplementation(() => Promise.resolve([{ lines: [{ label: calculated ? 'After the toggle' : 'Before the toggle', value: '+1', better: true }] }]));
    const box = () => within(view.getByRole('list', { name: 'Mod groups' })).getAllByRole<HTMLInputElement>('checkbox')[0];
    const shown = () => view.getByRole('tooltip', { name: 'Group toggle what-if' }).textContent;
    const header = (on: boolean) => (on ? 'Disabling this group will give you:' : 'Enabling this group will give you:');
    const before = get(outputs);
    const on = box().checked;
    try {
      await fireEvent.mouseOver(box());
      await vi.waitFor(() => expect(shown()).toContain('Before the toggle'));
      expect(shown()).toContain(header(on));
      await fireEvent.click(box());
      await vi.waitFor(() => expect(blocks()[0].Enabled).toBe(on ? 'false' : 'true'));
      await vi.waitFor(() => expect(shown()).toContain(header(!on)));
      calculated = true;
      outputs.set({ ...before } as NonNullable<typeof before>);
      await vi.waitFor(() => expect(shown()).toContain('After the toggle'));
      await fireEvent.click(box());
      await vi.waitFor(() => expect(blocks()[0].Enabled).toBe(on ? 'true' : 'false'));
    } finally {
      compare.mockRestore();
      outputs.set(before);
    }
  });
});

// ConfigTab.lua:42-44, 1305-1550
describe('the Mod browser', () => {
  it("searches, shows a line's sources and appends the chosen line to the group", async () => {
    const view = await mount();
    const MASTERY = 'All Damage with Maces and Sceptres inflicts Chill';
    const CRAFTED = '2% additional Physical Damage Reduction';
    const list = vi
      .spyOn(workerInstance, 'ModBrowserList')
      .mockImplementation((s: string) => (s ? [MASTERY, CRAFTED] : [MASTERY, CRAFTED, '1% increased Area of Effect']));
    try {
      const before = blocks()[0].Text;
      await fireEvent.click(view.getByRole('button', { name: 'Mod browser' }));
      const dialog = within(view.getByRole('dialog', { name: 'Mod browser' }));
      await vi.waitFor(() => expect(dialog.getAllByRole('option')).toHaveLength(3));
      await fireEvent.input(dialog.getByRole('searchbox', { name: 'Search modifiers' }), { target: { value: 'chill' } });
      await vi.waitFor(() => expect(list).toHaveBeenLastCalledWith('chill'));
      await vi.waitFor(() => expect(dialog.getAllByRole('option')).toHaveLength(2));
      await fireEvent.mouseEnter(dialog.getByRole('option', { name: CRAFTED }));
      await vi.waitFor(() => expect(dialog.getByRole('status').textContent).toContain('Crafted mod · Item mod'));
      await fireEvent.mouseEnter(dialog.getByRole('option', { name: MASTERY }));
      await vi.waitFor(() => expect(dialog.getByRole('status').textContent).toContain('Mastery Node'));
      expect(dialog.getByRole('option', { selected: true }).textContent).toBe(MASTERY);
      await fireEvent.click(dialog.getByRole('button', { name: 'Add' }));
      const lines = (t: string) =>
        t
          .split('\n')
          .map((l) => l.trim())
          .filter(Boolean);
      await vi.waitFor(() => expect(lines(blocks()[0].Text)).toEqual([...lines(before), MASTERY]));
    } finally {
      list.mockRestore();
    }
  });
});
