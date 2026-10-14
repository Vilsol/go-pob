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

const gemFixture: { entries: unknown[] } = { entries: [] };
vi.mock('$lib/cache', () => ({
  GetSkillGems: () => Promise.resolve(gemFixture.entries)
}));

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { readGate } = await import('../../../harness/component/as_remote');
const { get } = await import('svelte/store');
const { currentBuild, outputs } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');

const BUILD = '0008-necromancer-vaal-summon-skeletons.xml';
const GROUP = 0;
const EXTRA = { id: 'Metadata/Items/Gems/SkillGemFireball', name: 'Fireball' };
const VAAL = 'Metadata/Items/Gems/SkillGemVaalSummonSkeletons';

const spySetSocketGroupGems = () => {
  const build = workerInstance.currentBuild;
  if (!build) {
    throw new Error('REFUSING: no build loaded, so there is nothing to spy on');
  }
  return vi.spyOn(build, 'SetSocketGroupGems');
};

const goGroups = () => workerInstance.currentBuild?.Skills?.SkillSets?.[0]?.Skills ?? [];
const goGroup = (i = GROUP) => goGroups()[i];
const goGems = () => goGroup()?.Gems ?? [];
const goGemLevels = () => goGems().map((g) => g.Level);
const goGemQualities = () => goGems().map((g) => g.Quality);
const goGemIds = () => goGems().map((g) => g.GemID);

let tick: MockInstance<typeof workerInstance.Tick>;

beforeAll(async () => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));

  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD);

  tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);

  const entry = (id: string, name: string) => ({
    ID: id,
    GemType: 'INT',
    MaxLevel: 40,
    Support: false,
    Base: { Name: name }
  });
  gemFixture.entries = [...goGems().map((g) => entry(g.GemID, g.NameSpec)), entry(EXTRA.id, EXTRA.name)];
  // SkillsTab.lua:1021-1048
  for (const e of gemFixture.entries as { ID: string; GlobalEffects?: string[] }[]) {
    if (e.ID === VAAL) e.GlobalEffects = ['Vaal Summon Skeletons', 'Summon Skeletons'];
  }
});

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const editor = () => screen.getByRole('region', { name: 'Selected group' });
const gemBox = () => within(editor()).getByRole('group', { name: 'Gems' });
const spins = (suffix: 'level' | 'quality' | 'count') => within(gemBox()).getAllByRole<HTMLInputElement>('spinbutton', { name: new RegExp(` ${suffix}$`) });

const mountSkills = async () => {
  const Page = (await import('./+page.svelte')).default;
  const rendered = render(Page);
  await vi.waitFor(() => expect(spins('level').length).toBeGreaterThan(0), { timeout: 10_000 });
  return rendered;
};

const commitNumber = async (input: HTMLInputElement, target: number) => {
  await fireEvent.input(input, { target: { value: String(target) } });
  await fireEvent.keyDown(input, { key: 'Enter' });
  expect(Number(input.value), 'the input did not take the edit').toBe(target);
  await sleep(300);
};

describe('the skills page mounts with the build’s gems', () => {
  it('renders one row per gem in the build, not an empty list', async () => {
    await mountSkills();
    expect(goGemLevels().length).toBe(6);
    expect(spins('level').length).toBe(6);
    expect(spins('quality').length).toBe(6);
    expect(spins('count').length).toBe(6);
  });

  it('shows the levels Go actually holds', async () => {
    await mountSkills();
    expect(spins('level').map((el) => Number(el.value))).toEqual(goGemLevels());
  });

  it('lists every socket group and marks the main one', async () => {
    await mountSkills();
    const list = screen.getByRole('region', { name: 'Socket groups' });
    const cards = within(list)
      .getAllByRole('button')
      .filter((b) => b.hasAttribute('aria-current'));
    expect(cards.length).toBe(goGroups().length);
    expect(cards[0].getAttribute('aria-current')).toBe('true');
    expect(cards[0].textContent).toContain('MAIN');
  });
});

describe('CONTROL: a socket-group field reaches Go', () => {
  it('toggling the socket-group Enabled checkbox flips the Go field, and Ticks', async () => {
    await mountSkills();
    expect(goGroup()?.Enabled, 'the build should start with group 0 enabled').toBe(true);
    const ticks = tick.mock.calls.length;

    await fireEvent.click(within(editor()).getByRole('checkbox', { name: 'Enabled' }));

    await vi.waitFor(() => expect(goGroup()?.Enabled).toBe(false), { timeout: 5000 });
    expect(tick.mock.calls.slice(ticks).map((c) => c[0])).toContain('currentGemGroup');

    await fireEvent.click(within(editor()).getByRole('checkbox', { name: 'Enabled' }));
    await vi.waitFor(() => expect(goGroup()?.Enabled).toBe(true), { timeout: 5000 });
  });
});

describe('SUBJECT: a gem field edit reaching Go (r30 §7.1)', () => {
  it('ISSUES SetSocketGroupGems for the edited group', async () => {
    await mountSkills();
    const spy = spySetSocketGroupGems();
    try {
      await commitNumber(spins('level')[0], 13);
      expect(spy, 'the page never called SetSocketGroupGems').toHaveBeenCalled();
      expect(spy.mock.calls.at(-1)?.slice(0, 2)).toEqual([0, GROUP]);
    } finally {
      spy.mockRestore();
    }
  });

  it('sends pob.Gem fields only', async () => {
    await mountSkills();
    const spy = spySetSocketGroupGems();
    try {
      await commitNumber(spins('level')[0], 14);
      const payload = spy.mock.calls.at(-1)?.[2] as Record<string, unknown>[] | undefined;
      expect(payload, 'no payload captured').toBeDefined();
      expect(Object.keys(payload![0])).not.toContain('GemListValue');
      expect(Object.keys(payload![0]).sort()).toEqual(Object.keys(goGems()[0]).sort());
    } finally {
      spy.mockRestore();
    }
  });

  it('lands a new gem Level in Go', async () => {
    await mountSkills();
    const before = goGemLevels();
    const target = before[0] === 7 ? 8 : 7;

    await commitNumber(spins('level')[0], target);

    const after = goGemLevels();
    expect(after.length).toBe(before.length);
    expect(after[0], `gem 0 Level in Go: before=${before[0]} after=${after[0]} (edited the input to ${target})`).toBe(target);
    expect(after.slice(1)).toEqual(before.slice(1));
  });

  it('lands a new gem Quality in Go', async () => {
    await mountSkills();
    const before = goGemQualities();
    const target = before[0] === 11 ? 12 : 11;

    await commitNumber(spins('quality')[0], target);

    expect(goGemQualities()[0], `gem 0 Quality in Go: before=${before[0]} (edited to ${target})`).toBe(target);
  });

  it('lands a gem Count in Go', async () => {
    await mountSkills();
    const target = (goGems()[1].Count ?? 1) === 3 ? 4 : 3;

    await commitNumber(spins('count')[1], target);

    expect(goGems()[1].Count).toBe(target);
  });

  it('lands a gem Enabled in Go', async () => {
    await mountSkills();
    const name = within(gemBox()).getAllByRole('checkbox', { name: / enabled$/ })[2];
    const before = goGems()[2].Enabled;

    await fireEvent.click(name);
    await vi.waitFor(() => expect(goGems()[2].Enabled).toBe(!before), { timeout: 5000 });

    await fireEvent.click(within(gemBox()).getAllByRole('checkbox', { name: / enabled$/ })[2]);
    await vi.waitFor(() => expect(goGems()[2].Enabled).toBe(before), { timeout: 5000 });
  });

  // SkillsTab.lua:1015-1020
  it('lands a Vaal gem enableGlobal2 in Go, and Ticks', async () => {
    await mountSkills();
    const at = goGems().findIndex((g) => g.GemID === VAAL);
    expect(at, 'the build carries no Vaal Summon Skeletons in this group').toBeGreaterThanOrEqual(0);
    const before = goGems()[at].EnableGlobal2;
    const box = within(gemBox()).getByRole<HTMLInputElement>('checkbox', { name: 'Enable Summon Skeletons' });
    expect(box.checked).toBe(before);
    const ticks = tick.mock.calls.length;

    await fireEvent.click(box);
    await vi.waitFor(() => expect(goGems()[at].EnableGlobal2).toBe(!before), { timeout: 5000 });
    expect(tick.mock.calls.length, 'the toggle did not recalculate').toBeGreaterThan(ticks);

    await fireEvent.click(within(gemBox()).getByRole('checkbox', { name: 'Enable Summon Skeletons' }));
    await vi.waitFor(() => expect(goGems()[at].EnableGlobal2).toBe(before), { timeout: 5000 });
  });

  it('preserves gem fields the page never displays', async () => {
    await mountSkills();
    const gemsBefore = goGems().map((g) => ({ id: g.GemID, nameSpec: g.NameSpec, skillId: g.SkillID, part: g.SkillPart, minion: g.SkillMinion }));

    await commitNumber(spins('level')[0], 12);

    const gemsAfter = goGems().map((g) => ({ id: g.GemID, nameSpec: g.NameSpec, skillId: g.SkillID, part: g.SkillPart, minion: g.SkillMinion }));
    expect(gemsAfter).toEqual(gemsBefore);
  });
});

describe('reordering gems writes the new order', () => {
  it('Alt+↓ on a gem row moves it in Go, Alt+↑ moves it back', async () => {
    await mountSkills();
    const before = goGemIds();
    const rows = () =>
      within(gemBox())
        .getAllByRole('group')
        .filter((r) => r.hasAttribute('data-reorder-item'));

    await fireEvent.keyDown(rows()[0], { key: 'ArrowDown', altKey: true });
    await vi.waitFor(() => expect(goGemIds()).toEqual([before[1], before[0], ...before.slice(2)]), { timeout: 5000 });

    await fireEvent.keyDown(rows()[1], { key: 'ArrowUp', altKey: true });
    await vi.waitFor(() => expect(goGemIds()).toEqual(before), { timeout: 5000 });
  });
});

describe('adding a gem sends the pin default, and removing it removes it', () => {
  it('adds Go’s DefaultGem for the picked gem', async () => {
    await mountSkills();
    const spy = spySetSocketGroupGems();
    const before = goGems().length;
    try {
      const box = within(editor()).getByRole('combobox', { name: 'Add a gem' });
      await fireEvent.focus(box);
      await fireEvent.input(box, { target: { value: EXTRA.name } });
      await fireEvent.keyDown(box, { key: 'Enter' });

      await vi.waitFor(() => expect(spy).toHaveBeenCalled(), { timeout: 10_000 });
      const gems = spy.mock.calls.at(-1)![2] ?? [];
      expect(gems.length, 'the pick must ADD a row').toBe(before + 1);
      const gem = gems[gems.length - 1];

      expect(gem.GemID).toBe(EXTRA.id);
      expect(gem.NameSpec).toBe(EXTRA.name);
      expect(gem.SkillMinion, 'SkillMinion was hardcoded to SummonedPhantasm').toBe('');
      expect(gem.SkillMinionItemSet).toBe(0);
      expect(gem.SkillPart).toBe(0);
      expect(gem.SkillPartCalcs).toBe(0);
      expect(gem.SkillMinionSkill).toBe(0);
      expect(gem.SkillStageCount).toBe(0);
      expect(gem.SkillStageCountCalcs).toBe(0);
      // GemSelectControl.lua:587-600
      expect(gem.Enabled).toBe(true);
      expect(gem.EnableGlobal1).toBe(true);
      expect(gem.EnableGlobal2).toBe(true);
      expect(gem.QualityID).toBe('Default');
      expect(gem.Count).toBe(1);
      // SkillsTab.lua:768-800
      expect(gem.Level).toBe(20);
      expect(gem.Quality).toBe(workerInstance.currentBuild?.Skills?.DefaultGemQuality ?? 0);
      await vi.waitFor(() => expect(goGemIds().at(-1)).toBe(EXTRA.id), { timeout: 5000 });
    } finally {
      spy.mockRestore();
    }
  });

  it('removes a gem from Go', async () => {
    await mountSkills();
    expect(goGemIds().at(-1), 'the previous test added the extra gem').toBe(EXTRA.id);
    const before = goGemIds();

    await fireEvent.click(within(editor()).getByRole('button', { name: `Remove ${EXTRA.name}` }));

    await vi.waitFor(() => expect(goGemIds()).toEqual(before.slice(0, -1)), { timeout: 5000 });
  });
});

describe('an edit made while the page re-reads the groups from Go', () => {
  it('lands in Go and stays on the page', async () => {
    await mountSkills();
    await sleep(300);
    const target = goGemLevels()[0] === 9 ? 10 : 9;

    let release!: () => void;
    const released = new Promise<void>((r) => (release = r));
    let held = 0;
    readGate.hold = (path) => {
      if (/\.Skills\.\d+\.(IncludeInFullDPS|Slot)$/.test(path)) {
        held++;
        return released;
      }
      return undefined;
    };
    try {
      currentBuild.set(get(currentBuild));
      await vi.waitFor(() => expect(held, 'the re-read never reached the gated fields').toBeGreaterThan(0), { timeout: 5000 });

      await commitNumber(spins('level')[0], target);
    } finally {
      readGate.hold = undefined;
      release();
    }
    await sleep(300);

    expect(goGemLevels()[0], 'gem 0 Level in Go').toBe(target);
    expect(Number(spins('level')[0].value), 'gem 0 Level on the page').toBe(target);
  });
});

describe('the selected group’s own fields', () => {
  it('writes the Label', async () => {
    await mountSkills();
    const label = within(editor()).getByRole('textbox', { name: 'Label' });
    await fireEvent.change(label, { target: { value: 'Minions' } });
    await vi.waitFor(() => expect(goGroup()?.Label).toBe('Minions'), { timeout: 5000 });
    await fireEvent.change(within(editor()).getByRole('textbox', { name: 'Label' }), { target: { value: '' } });
    await vi.waitFor(() => expect(goGroup()?.Label).toBe(''), { timeout: 5000 });
  });

  it('writes Include in Full DPS', async () => {
    await mountSkills();
    expect(goGroup()?.IncludeInFullDPS).toBe(true);
    await fireEvent.click(within(editor()).getByRole('checkbox', { name: 'Include in Full DPS' }));
    await vi.waitFor(() => expect(goGroup()?.IncludeInFullDPS).toBe(false), { timeout: 5000 });
    await fireEvent.click(within(editor()).getByRole('checkbox', { name: 'Include in Full DPS' }));
    await vi.waitFor(() => expect(goGroup()?.IncludeInFullDPS).toBe(true), { timeout: 5000 });
  });

  // SkillsTab.lua:13-28, 185-198
  it('lists PoB’s slots, each naming the item socketed there', async () => {
    await mountSkills();
    const select = within(editor()).getByRole('combobox', { name: 'Socketed in' });
    await fireEvent.click(select);
    const option = (label: string) => within(editor()).queryByRole('option', { name: label });
    expect(option('Ring 3'), 'Ring 3 is in groupSlotDropList').not.toBeNull();
    expect(option('Ring 3')?.title).toBe('No item is equipped in this slot.');
    expect(select.title).toMatch(/^Select the item in which this skill is socketed\./);
    await fireEvent.keyDown(select, { key: 'Escape' });
  });

  // SkillsTab.lua:190-198
  it("shows a slot entry's item tooltip", async () => {
    const names = vi.spyOn(workerInstance, 'GetSocketedInItemNames').mockResolvedValue({ 'Body Armour': 'Stub Plate' });
    await mountSkills();
    const tip = vi
      .spyOn(workerInstance, 'ItemSourceTooltip')
      .mockResolvedValue({ Influence: [], Title: ['^xAF6025Stub Plate'], Lines: [{ Text: 'Plate of stubs', Separator: false }], Skills: [] } as never);
    try {
      await fireEvent.click(within(editor()).getByRole('combobox', { name: 'Socketed in' }));
      await fireEvent.mouseEnter(within(editor()).getByRole('option', { name: 'Body Armour' }));
      await vi.waitFor(() => expect(tip).toHaveBeenCalledWith('Stub Plate'));
      await vi.waitFor(() => expect(document.body.textContent).toContain('Plate of stubs'));
    } finally {
      tip.mockRestore();
      names.mockRestore();
    }
  });

  it('writes the slot it is socketed in', async () => {
    await mountSkills();
    expect(goGroup()?.Slot).toBe('Body Armour');
    const choose = async (slot: string) => {
      await fireEvent.click(within(editor()).getByRole('combobox', { name: 'Socketed in' }));
      await fireEvent.click(within(editor()).getByRole('option', { name: slot }));
    };
    await choose('Helmet');
    await vi.waitFor(() => expect(goGroup()?.Slot).toBe('Helmet'), { timeout: 5000 });
    await choose('Body Armour');
    await vi.waitFor(() => expect(goGroup()?.Slot).toBe('Body Armour'), { timeout: 5000 });
  });
});

describe('the main socket group', () => {
  it('right click on a group sets it as the main group through the worker', async () => {
    await mountSkills();
    const spy = vi.spyOn(workerInstance, 'SetMainSocketGroup');
    try {
      const list = screen.getByRole('region', { name: 'Socket groups' });
      const cards = within(list)
        .getAllByRole('button')
        .filter((b) => b.hasAttribute('aria-current'));
      await fireEvent.contextMenu(cards[1]);
      await vi.waitFor(() => expect(spy).toHaveBeenCalledWith(2), { timeout: 5000 });
      await fireEvent.click(cards[0]);
      await fireEvent.click(within(list).getByRole('button', { name: 'Set as main' }));
      await vi.waitFor(() => expect(spy).toHaveBeenLastCalledWith(1), { timeout: 5000 });
    } finally {
      spy.mockRestore();
    }
  });
});

// SkillListControl.lua:192-227, SkillsTab.lua:628-629, 667-713
describe('the group list shortcuts', () => {
  const cards = () =>
    within(screen.getByRole('region', { name: 'Socket groups' }))
      .getAllByRole('button')
      .filter((b) => b.hasAttribute('aria-current'));

  it('Ctrl+click toggles a group’s Enabled and Ctrl+right click its Full DPS, in Go', async () => {
    await mountSkills();
    const enabled = goGroup(1)?.Enabled;
    await fireEvent.click(cards()[1], { ctrlKey: true });
    await vi.waitFor(() => expect(goGroup(1)?.Enabled).toBe(!enabled), { timeout: 5000 });
    await fireEvent.click(cards()[1], { ctrlKey: true });
    await vi.waitFor(() => expect(goGroup(1)?.Enabled).toBe(enabled), { timeout: 5000 });

    const full = !!goGroup(1)?.IncludeInFullDPS;
    const main = workerInstance.currentBuild?.Build?.MainSocketGroup;
    await fireEvent.contextMenu(cards()[1], { ctrlKey: true });
    await vi.waitFor(() => expect(!!goGroup(1)?.IncludeInFullDPS).toBe(!full), { timeout: 5000 });
    expect(workerInstance.currentBuild?.Build?.MainSocketGroup, 'Ctrl+right click is not "set as main"').toBe(main);
    await fireEvent.contextMenu(cards()[1], { ctrlKey: true });
    await vi.waitFor(() => expect(!!goGroup(1)?.IncludeInFullDPS).toBe(full), { timeout: 5000 });
  });

  it('Ctrl+C copies the group as PoB’s text, and pasting it adds the group in Go', async () => {
    await mountSkills();
    let copied = '';
    const clip = { writeText: vi.fn((t: string) => Promise.resolve(void (copied = t))) };
    Object.defineProperty(navigator, 'clipboard', { value: clip, configurable: true });
    await fireEvent.keyDown(cards()[0], { key: 'c', ctrlKey: true });
    await vi.waitFor(() => expect(clip.writeText).toHaveBeenCalled(), { timeout: 5000 });
    const first = goGems()[0];
    expect(copied).toContain(`${first.NameSpec} ${first.Level}/${first.Quality}`);

    const n = goGroups().length;
    const paste = new Event('paste') as ClipboardEvent;
    Object.defineProperty(paste, 'clipboardData', { value: { getData: () => copied } });
    window.dispatchEvent(paste);
    await vi.waitFor(() => expect(goGroups().length).toBe(n + 1), { timeout: 5000 });
    expect(goGroups()[n].Gems?.map((g) => g.NameSpec)).toEqual(goGems().map((g) => g.NameSpec));
    await fireEvent.click(within(screen.getByRole('region', { name: 'Socket groups' })).getByRole('button', { name: 'Delete group' }));
    await vi.waitFor(() => expect(goGroups().length).toBe(n), { timeout: 5000 });
  });

  it('hovering a listed gem and a quality field asks Go for their what-ifs', async () => {
    await mountSkills();
    const pick = vi.spyOn(workerInstance, 'GemSelectCompare').mockResolvedValue([]);
    const quality = vi.spyOn(workerInstance, 'GemQualityCompare').mockResolvedValue([]);
    try {
      const box = within(editor()).getByRole('combobox', { name: 'Add a gem' });
      await fireEvent.focus(box);
      await fireEvent.input(box, { target: { value: EXTRA.name } });
      await fireEvent.mouseEnter(within(editor()).getByRole('option', { name: new RegExp(EXTRA.name) }));
      await vi.waitFor(() => expect(pick).toHaveBeenCalledWith(GROUP, EXTRA.id, EXTRA.name), { timeout: 5000 });
      await vi.waitFor(() => expect(within(editor()).getByRole('status', { name: 'What-if' }).textContent).toContain('Selecting this gem will give you:'));
      await fireEvent.blur(box);

      await fireEvent.mouseEnter(spins('quality')[1].closest('span.col-span-4')!);
      await vi.waitFor(() => expect(quality).toHaveBeenCalledWith(GROUP, 1), { timeout: 5000 });
      await vi.waitFor(() => expect(within(editor()).getByRole('status', { name: 'What-if' }).textContent).toContain('Setting to 20 quality will give you:'));
    } finally {
      pick.mockRestore();
      quality.mockRestore();
    }
  });

  // SkillsTab.lua:948-960
  it("hovering a gem's enable checkbox asks for its what-if, worded by the gem's state", async () => {
    await mountSkills();
    const compare = vi.spyOn(workerInstance, 'GemEnableCompare').mockResolvedValue([]);
    const box = () => within(editor()).getAllByRole('checkbox', { name: / enabled$/ })[0] as HTMLInputElement;
    const shown = () => within(editor()).getByRole('status', { name: 'What-if' });
    try {
      expect(box().checked, 'the first gem starts enabled').toBe(true);
      await fireEvent.mouseEnter(box());
      await vi.waitFor(() => expect(compare).toHaveBeenCalledWith(GROUP, 0), { timeout: 5000 });
      await vi.waitFor(() => expect(shown().textContent).toContain('Disabling this gem will give you:'));
      await fireEvent.mouseLeave(box());
      await vi.waitFor(() => expect(within(editor()).queryByRole('status', { name: 'What-if' })).toBeNull());
      await fireEvent.click(box());
      await vi.waitFor(() => expect(goGems()[0].Enabled).toBe(false), { timeout: 5000 });
      await fireEvent.focus(box());
      await vi.waitFor(() => expect(shown().textContent).toContain('Enabling this gem will give you:'));
      expect(compare).toHaveBeenCalledTimes(2);
      await fireEvent.click(box());
      await vi.waitFor(() => expect(goGems()[0].Enabled).toBe(true), { timeout: 5000 });
    } finally {
      compare.mockRestore();
    }
  });
  it('the gem enable what-if is asked again once the toggle is calculated, without a re-hover', async () => {
    await mountSkills();
    let calculated = false;
    const life = { label: 'Life', value: '+10', better: true };
    const compare = vi.spyOn(workerInstance, 'GemEnableCompare').mockImplementation(() => Promise.resolve(calculated ? [{ lines: [life] }] : []));
    const box = () => within(editor()).getAllByRole('checkbox', { name: / enabled$/ })[0] as HTMLInputElement;
    const shown = () => within(editor()).getByRole('status', { name: 'What-if' });
    const before = get(outputs);
    try {
      await fireEvent.mouseEnter(box());
      await vi.waitFor(() => expect(shown().textContent).toContain('Disabling this gem will give you:'));
      await fireEvent.click(box());
      await vi.waitFor(() => expect(goGems()[0].Enabled).toBe(false), { timeout: 5000 });
      await vi.waitFor(() => expect(shown().textContent).toContain('Enabling this gem will give you:'));
      calculated = true;
      outputs.set({ ...before } as NonNullable<typeof before>);
      await vi.waitFor(() => expect(shown().textContent).toContain('Life'));
      expect(shown().textContent).toContain('Enabling this gem will give you:');
      await fireEvent.click(box());
      await vi.waitFor(() => expect(goGems()[0].Enabled).toBe(true), { timeout: 5000 });
    } finally {
      compare.mockRestore();
      outputs.set(before);
    }
  });
});

describe('the gem tooltip (GemTooltip.lua)', () => {
  it('hovering a gem row, a listed gem and a quality field asks the worker and draws its lines above the what-if', async () => {
    await mountSkills();
    const line = (Text: string) => ({ Text, Separator: false });
    const tip = vi
      .spyOn(workerInstance, 'GemTooltip')
      .mockResolvedValue([line('^x1AA29BGem Zero'), { Text: '', Separator: true }, line('^x7F7F7FLevel: ^720')]);
    const list = vi.spyOn(workerInstance, 'GemListTooltip').mockReturnValue([line('^x1AA29BListed Fireball')]);
    const quality = vi.spyOn(workerInstance, 'GemQualityTooltip').mockReturnValue([line('^xC8C8C8At +20% Quality:')]);
    const pick = vi.spyOn(workerInstance, 'GemSelectCompare').mockResolvedValue([]);
    const compare = vi.spyOn(workerInstance, 'GemQualityCompare').mockResolvedValue([]);
    const shown = () => within(editor()).getByRole('status', { name: 'What-if' });
    try {
      const name = within(gemBox()).getAllByText(goGems()[0].NameSpec)[0].closest<HTMLElement>('div.relative')!;
      await fireEvent.mouseEnter(name);
      await vi.waitFor(() => expect(tip).toHaveBeenCalledWith(GROUP, 0), { timeout: 5000 });
      await vi.waitFor(() => expect(within(shown()).getByLabelText('Gem tooltip').textContent).toContain('Gem Zero'));
      expect(shown().querySelector('hr'), 'a separator line draws a rule').not.toBeNull();
      expect(shown().querySelector('[data-pob-color]')?.getAttribute('style')).toContain('#1AA29B');
      await fireEvent.mouseLeave(name);
      await vi.waitFor(() => expect(within(editor()).queryByRole('status', { name: 'What-if' })).toBeNull());

      const box = within(editor()).getByRole('combobox', { name: 'Add a gem' });
      await fireEvent.focus(box);
      await fireEvent.input(box, { target: { value: EXTRA.name } });
      await fireEvent.mouseEnter(within(editor()).getByRole('option', { name: new RegExp(EXTRA.name) }));
      await vi.waitFor(() => expect(list).toHaveBeenCalledWith(EXTRA.id, EXTRA.name), { timeout: 5000 });
      await vi.waitFor(() => expect(shown().textContent).toContain('Listed Fireball'));
      expect(shown().textContent).toContain('Selecting this gem will give you:');
      expect(shown().textContent!.indexOf('Listed Fireball')).toBeLessThan(shown().textContent!.indexOf('Selecting this gem'));
      await fireEvent.blur(box);

      await fireEvent.mouseEnter(spins('quality')[1].closest('span.col-span-4')!);
      await vi.waitFor(() => expect(quality).toHaveBeenCalledWith(GROUP, 1), { timeout: 5000 });
      await vi.waitFor(() => expect(shown().textContent).toContain('At +20% Quality:'));
      expect(shown().textContent).toContain('Setting to 20 quality will give you:');
    } finally {
      for (const s of [tip, list, quality, pick, compare]) s.mockRestore();
    }
  });
});

describe('the socket group tooltip (SkillsTab:AddSocketGroupTooltip)', () => {
  const cards = () =>
    within(screen.getByRole('region', { name: 'Socket groups' }))
      .getAllByRole('button')
      .filter((b) => b.hasAttribute('aria-current'));

  it('hovering a group in the list draws its tooltip lines from the worker', async () => {
    await mountSkills();
    const tip = vi.spyOn(workerInstance, 'SocketGroupTooltip').mockResolvedValue([{ Text: '^7Inactive Gems:', Separator: false }]);
    try {
      await fireEvent.mouseEnter(cards()[1]);
      await vi.waitFor(() => expect(tip).toHaveBeenCalledWith(1), { timeout: 5000 });
      await vi.waitFor(() => expect(screen.getByRole('tooltip', { name: 'Socket group tooltip' }).textContent).toContain('Inactive Gems:'));
      await fireEvent.mouseLeave(cards()[1]);
      await vi.waitFor(() => expect(screen.queryByRole('tooltip', { name: 'Socket group tooltip' })).toBeNull());
    } finally {
      tip.mockRestore();
    }
  });

  it('a Ctrl+click toggle under a still pointer reads the tooltip again once calculated', async () => {
    await mountSkills();
    let calculated = false;
    const tip = vi
      .spyOn(workerInstance, 'SocketGroupTooltip')
      .mockImplementation(() => Promise.resolve([{ Text: calculated ? '^7After the toggle' : '^7Before the toggle', Separator: false }]));
    const shown = () => screen.getByRole('tooltip', { name: 'Socket group tooltip' });
    const enabled = goGroup(1)?.Enabled;
    const before = get(outputs);
    try {
      await fireEvent.mouseEnter(cards()[1]);
      await vi.waitFor(() => expect(shown().textContent).toContain('Before the toggle'));
      await fireEvent.click(cards()[1], { ctrlKey: true });
      await vi.waitFor(() => expect(goGroup(1)?.Enabled).toBe(!enabled), { timeout: 5000 });
      calculated = true;
      outputs.set({ ...before } as NonNullable<typeof before>);
      await vi.waitFor(() => expect(shown().textContent).toContain('After the toggle'));
      await fireEvent.click(cards()[1], { ctrlKey: true });
      await vi.waitFor(() => expect(goGroup(1)?.Enabled).toBe(enabled), { timeout: 5000 });
    } finally {
      tip.mockRestore();
      outputs.set(before);
    }
  });
});

describe('the gem picker sort cache (GemSelectControl.lua:255-441)', () => {
  it('opening the picker asks Go which gems support the group and their DPS, and lists by them', async () => {
    await mountSkills();
    const cache = vi.spyOn(workerInstance, 'GemSortCache');
    const dps = vi.spyOn(workerInstance, 'GemSortDPS');
    cache.mockImplementation((_, gems) => Promise.resolve({ canSupport: gems.map(() => false), pending: gems.map((g) => g.id === EXTRA.id) }));
    dps.mockImplementation((_, gems) => Promise.resolve({ base: 100, dps: gems.map(() => 150) }));
    try {
      const box = within(editor()).getByRole('combobox', { name: 'Add a gem' });
      await fireEvent.focus(box);
      await vi.waitFor(() => expect(cache).toHaveBeenCalled(), { timeout: 5000 });
      expect(cache.mock.calls[0][0]).toBe(GROUP);
      await vi.waitFor(() => expect(dps).toHaveBeenCalled(), { timeout: 5000 });
      expect(dps.mock.calls[0][1].map((g) => g.id)).toEqual([EXTRA.id]);
      await fireEvent.input(box, { target: { value: EXTRA.name } });
      await vi.waitFor(() => expect(within(editor()).getByRole('option', { name: new RegExp(EXTRA.name) }).textContent).toContain('▲ +50%'));
    } finally {
      cache.mockRestore();
      dps.mockRestore();
    }
  });

  // SkillsTab.lua:124-129, GemSelectControl.lua:258-264
  it('is rebuilt after the sort stat changes, and after sorting is switched on', async () => {
    await mountSkills();
    const other = { id: goGemIds()[1], name: goGems()[1].NameSpec };
    const cache = vi.spyOn(workerInstance, 'GemSortCache');
    const dps = vi.spyOn(workerInstance, 'GemSortDPS');
    cache.mockImplementation((_, gems) =>
      Promise.resolve({ canSupport: gems.map(() => false), pending: gems.map((g) => g.id === EXTRA.id || g.id === other.id) })
    );
    dps.mockImplementation((_, gems, field) =>
      Promise.resolve({ base: 100, dps: gems.map((g) => ((g.id === EXTRA.id) === (field === 'FullDPS') ? 300 : 200)) })
    );
    const options = screen.getByRole('region', { name: 'Gem options' });
    const sortBy = within(options).getByRole('combobox', { name: 'Sort by' });
    const sortOn = within(options).getByRole<HTMLInputElement>('checkbox', { name: 'Sort gems by DPS' });
    const box = within(editor()).getByRole('combobox', { name: 'Add a gem' });
    const order = () =>
      within(editor())
        .getAllByRole('option')
        .map((o) => o.textContent ?? '')
        .map((t) => (t.includes(EXTRA.name) ? 'extra' : t.includes(other.name) ? 'other' : undefined))
        .filter(Boolean)
        .slice(0, 2);
    const openPicker = async (expected: string[]) => {
      const calls = cache.mock.calls.length;
      await fireEvent.focus(box);
      await vi.waitFor(
        () => {
          expect(cache.mock.calls.length).toBeGreaterThan(calls);
          expect(within(editor()).queryByLabelText('Sort progress')).toBeNull();
          expect(order()).toEqual(expected);
        },
        { timeout: 5000 }
      );
      await sleep(100);
      expect(order()).toEqual(expected);
      await fireEvent.blur(box);
    };
    try {
      await openPicker(['other', 'extra']);
      await fireEvent.change(sortBy, { target: { value: 'Full DPS' } });
      await vi.waitFor(() => expect(workerInstance.currentBuild?.Skills?.SortGemsByDPSField).toBe('FullDPS'), { timeout: 5000 });
      await openPicker(['extra', 'other']);
      expect(dps.mock.calls.at(-1)![2]).toBe('FullDPS');

      await fireEvent.change(sortBy, { target: { value: 'Combined DPS' } });
      await vi.waitFor(() => expect(workerInstance.currentBuild?.Skills?.SortGemsByDPSField).toBe('CombinedDPS'), { timeout: 5000 });
      expect(sortOn.checked).toBe(true);
      await fireEvent.click(sortOn);
      await vi.waitFor(() => expect(workerInstance.currentBuild?.Skills?.SortGemsByDPS).toBe(false), { timeout: 5000 });
      const before = dps.mock.calls.length;
      await openPicker(['extra', 'other']);
      expect(dps.mock.calls.length, 'sorting off asked Go for the DPS').toBe(before);
      await fireEvent.click(sortOn);
      await vi.waitFor(() => expect(workerInstance.currentBuild?.Skills?.SortGemsByDPS).toBe(true), { timeout: 5000 });
      await openPicker(['other', 'extra']);
    } finally {
      cache.mockRestore();
      dps.mockRestore();
      workerInstance.currentBuild?.SetSortGemsByDPSField('CombinedDPS');
      workerInstance.currentBuild?.SetSortGemsByDPS(true);
    }
  });
});

describe('the gem options', () => {
  it('write the default gem quality and the sort field to Go', async () => {
    await mountSkills();
    const options = screen.getByRole('region', { name: 'Gem options' });
    await commitNumber(within(options).getByRole('spinbutton', { name: 'Default gem quality' }), 7);
    await vi.waitFor(() => expect(workerInstance.currentBuild?.Skills?.DefaultGemQuality).toBe(7), { timeout: 5000 });

    await fireEvent.change(within(options).getByRole('combobox', { name: 'Sort by' }), { target: { value: 'Full DPS' } });
    await vi.waitFor(() => expect(workerInstance.currentBuild?.Skills?.SortGemsByDPSField).toBe('FullDPS'), { timeout: 5000 });
    await fireEvent.change(within(options).getByRole('combobox', { name: 'Sort by' }), { target: { value: 'Combined DPS' } });
    await vi.waitFor(() => expect(workerInstance.currentBuild?.Skills?.SortGemsByDPSField).toBe('CombinedDPS'), { timeout: 5000 });
  });

  it('write the default gem level mode and Show legacy gems to Go; a new gem then takes that mode', async () => {
    await mountSkills();
    const options = screen.getByRole('region', { name: 'Gem options' });
    const level = within(options).getByRole<HTMLSelectElement>('combobox', { name: 'Default gem level' });
    expect(level.value).toBe('normalMaximum');
    await fireEvent.change(level, { target: { value: 'levelOne' } });
    await vi.waitFor(() => expect(workerInstance.currentBuild?.Skills?.DefaultGemLevel).toBe('levelOne'), { timeout: 5000 });
    expect(workerInstance.currentBuild?.Skills?.MatchGemLevelToCharacterLevel).toBe(false);

    const legacy = within(options).getByRole('checkbox', { name: 'Show legacy gems' });
    expect((legacy as HTMLInputElement).checked).toBe(false);
    await fireEvent.click(legacy);
    await vi.waitFor(() => expect(workerInstance.currentBuild?.Skills?.ShowLegacyGems).toBe(true), { timeout: 5000 });

    const spy = spySetSocketGroupGems();
    try {
      const box = within(editor()).getByRole('combobox', { name: 'Add a gem' });
      await fireEvent.focus(box);
      await fireEvent.input(box, { target: { value: EXTRA.name } });
      await fireEvent.keyDown(box, { key: 'Enter' });
      await vi.waitFor(() => expect(spy).toHaveBeenCalled(), { timeout: 10_000 });
      expect((spy.mock.calls.at(-1)![2] ?? []).at(-1)?.Level, 'levelOne gives a new gem level 1').toBe(1);
      await vi.waitFor(() => expect(goGemIds().at(-1)).toBe(EXTRA.id), { timeout: 5000 });
    } finally {
      spy.mockRestore();
    }
    const rows = within(gemBox()).getAllByRole('button', { name: `Remove ${EXTRA.name}` });
    await fireEvent.click(rows[rows.length - 1]);
    await vi.waitFor(() => expect(goGemIds().at(-1)).not.toBe(EXTRA.id), { timeout: 5000 });
    await fireEvent.change(level, { target: { value: 'normalMaximum' } });
    await fireEvent.click(legacy);
    await vi.waitFor(() => expect(workerInstance.currentBuild?.Skills?.ShowLegacyGems).toBe(false), { timeout: 5000 });
  });
});

describe('mounting the skills page', () => {
  it('writes nothing back to Go, nor Ticks', async () => {
    const spy = spySetSocketGroupGems();
    const build = workerInstance.currentBuild!;
    const setters = (
      [
        'SetSortGemsByDPS',
        'SetSortGemsByDPSField',
        'SetMatchGemLevelToCharacterLevel',
        'SetDefaultGemLevel',
        'SetDefaultGemLevelMode',
        'SetShowLegacyGems',
        'SetDefaultGemQuality',
        'SetShowSupportGemTypes',
        'SetShowAltQualityGems',
        'SetMainSocketGroup'
      ] as const
    ).map((name) => [name, vi.spyOn(build, name)] as const);
    const ticksBefore = tick.mock.calls.length;
    const skills = build.Skills;
    const before = { level: skills.DefaultGemLevel, quality: skills.DefaultGemQuality, field: skills.SortGemsByDPSField, sort: skills.SortGemsByDPS };
    try {
      await mountSkills();
      await sleep(300);
      expect(spy, 'mount wrote unchanged gems back through SetSocketGroupGems').not.toHaveBeenCalled();
      for (const [name, s] of setters) expect(s, `mount called ${name}`).not.toHaveBeenCalled();
      expect(tick.mock.calls.slice(ticksBefore), 'mount issued a Tick').toEqual([]);
      expect({ level: skills.DefaultGemLevel, quality: skills.DefaultGemQuality, field: skills.SortGemsByDPSField, sort: skills.SortGemsByDPS }).toEqual(
        before
      );
    } finally {
      spy.mockRestore();
      for (const [, s] of setters) s.mockRestore();
    }
  });
});

describe('structural changes', () => {
  it('a new group is added in Go and selected; deleting it removes it', async () => {
    await mountSkills();
    const n = goGroups().length;
    const list = screen.getByRole('region', { name: 'Socket groups' });

    await fireEvent.click(within(list).getByRole('button', { name: '+ New group' }));
    await vi.waitFor(() => expect(goGroups().length).toBe(n + 1), { timeout: 5000 });
    await vi.waitFor(() => expect(within(editor()).queryAllByRole('spinbutton', { name: / level$/ })).toHaveLength(0), { timeout: 5000 });

    await fireEvent.click(within(list).getByRole('button', { name: 'Delete group' }));
    await vi.waitFor(() => expect(goGroups().length).toBe(n), { timeout: 5000 });
    expect(goGemLevels().length, 'group 0 is untouched').toBe(6);
  });
});
