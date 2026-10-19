// @vitest-environment jsdom
import '$lib/design/pointer_event_polyfill';

import { describe, expect, it, beforeAll, afterEach, onTestFinished, vi, type MockInstance } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { get } from 'svelte/store';
import type { exposition } from '$lib/types';

afterEach(cleanup);

const pageErrors: string[] = [];
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

const nav = vi.hoisted(() => ({ goto: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/navigation', async (original) => ({ ...(await original<object>()), goto: nav.goto }));

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

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
const types = await import('$lib/types');

const BUILD = '0280-elementalist-fireball.xml';

const goSlot = (name: string) => {
  const calc = types.calculator.NewCalculator(workerInstance.currentBuild!)!;
  try {
    const items = calc.PoB!.Items;
    const set = items.ItemSets.find((s) => s.ID === String(items.ActiveItemSet)) ?? items.ItemSets[0];
    return set.Slots?.find((s) => s.Name === name);
  } finally {
    calc.release();
  }
};

let tick: MockInstance<(source: string) => Promise<void>>;
let equipID: MockInstance<typeof workerInstance.EquipItemID>;
let saveItem: MockInstance<typeof workerInstance.SaveItem>;
let replaceItem: MockInstance<typeof workerInstance.ReplaceItem>;

const OLD_GLOVES = 'Rarity: RARE\nOld Grip\nIron Gauntlets';
const NEW_GLOVES = 'Rarity: RARE\nEdited Grip\nIron Gauntlets';

const goItems = () => {
  const calc = types.calculator.NewCalculator(workerInstance.currentBuild!)!;
  try {
    return calc.PoB!.Items.Items ?? [];
  } finally {
    calc.release();
  }
};
const goItemText = (id: number) => goItems().find((x) => x.ID === id)?.Text;

beforeAll(async () => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD);
  tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
  vi.spyOn(workerInstance, 'GetEquippedItems').mockImplementation(() =>
    Promise.resolve(
      ['Gloves', 'Flask 2'].flatMap((name) => {
        const s = goSlot(name);
        return s && s.ItemID > 0
          ? [
              {
                SlotName: s.Name,
                ItemID: s.ItemID,
                Name: `item ${s.ItemID}`,
                Rarity: 'RARE',
                Flask: s.Name.startsWith('Flask'),
                Active: s.Active,
                Text: s.Name === 'Gloves' ? OLD_GLOVES : ''
              }
            ]
          : [];
      })
    )
  );
  vi.spyOn(workerInstance, 'GetItemsTabSlots').mockResolvedValue({ abyssal: [], sockets: [] });
  vi.spyOn(workerInstance, 'GetItemSetTooltips').mockResolvedValue({});
  vi.spyOn(workerInstance, 'GetConditionalSlots').mockResolvedValue([]);
  vi.spyOn(workerInstance, 'GetItemListFilter').mockResolvedValue({ filters: [], items: [] });
  vi.spyOn(workerInstance, 'GetTreeSpecs').mockResolvedValue(undefined);
  vi.spyOn(workerInstance, 'GetSharedItemSets').mockResolvedValue([]);
  vi.spyOn(workerInstance, 'GetSlotItemLists').mockImplementation(() =>
    Promise.resolve({ Gloves: [goSlot('Gloves')?.ItemID ?? 0, goSlot('Flask 2')?.ItemID ?? 0].filter((id) => id > 0) })
  );
  equipID = vi.spyOn(workerInstance, 'EquipItemID').mockImplementation((slot: string, id: number) => {
    workerInstance.currentBuild!.SetSlotItem(slot, id);
    return Promise.resolve(undefined);
  });
  vi.spyOn(workerInstance, 'ParseItem').mockImplementation((text: string) =>
    Promise.resolve(
      text === NEW_GLOVES
        ? {
            item: {
              Name: 'Edited Grip',
              Title: 'Edited Grip',
              BaseName: 'Iron Gauntlets',
              Type: 'Gloves',
              Rarity: 'RARE',
              Flask: false,
              FitSlots: ['Gloves'],
              Tooltip: {}
            } as unknown as exposition.ParsedItem
          }
        : { error: 'no game data in the component harness' }
    )
  );
  vi.spyOn(workerInstance, 'ItemCompare').mockResolvedValue(undefined);
  vi.spyOn(workerInstance, 'PrepareStatCompare').mockResolvedValue(undefined);
  saveItem = vi.spyOn(workerInstance, 'SaveItem').mockResolvedValue({ error: 'SaveItem is not this test’s write' });
  replaceItem = vi.spyOn(workerInstance, 'ReplaceItem').mockImplementation((id: number, text: string) => {
    workerInstance.currentBuild!.SetItemText(id, text);
    return Promise.resolve(undefined);
  });
}, 120_000);

const mount = async () => {
  const Page = (await import('./+page.svelte')).default;
  return render(Page);
};

describe('items page', () => {
  // ItemsTab.lua:203-205
  it('Manage… beside the Passive tree select opens the Tree page’s tree manager', async () => {
    const { treeManageRequest } = await import('$lib/shell/tree_manage');
    const specs = vi.spyOn(workerInstance, 'GetTreeSpecs').mockResolvedValue({ active: 1, specs: [{ Label: 'Main', TreeVersion: '3_29' }] } as never);
    onTestFinished(() => void specs.mockResolvedValue(undefined));
    const view = await mount();
    const manage = await view.findByRole('button', { name: 'Manage passive trees' }, { timeout: 10_000 });
    nav.goto.mockClear();
    await fireEvent.click(manage);
    expect(get(treeManageRequest)).toBe(true);
    expect(nav.goto).toHaveBeenCalledWith('/tree');
    treeManageRequest.set(false);
  });

  it('lists the equipped slots from Go and writes nothing on mount', async () => {
    tick.mockClear();
    const { findByRole } = await mount();
    const board = await findByRole('region', { name: 'Equipped items' });
    const id = goSlot('Gloves')?.ItemID ?? 0;
    expect(id, 'vacuity: 0280 has no gloves to unequip').toBeGreaterThan(0);
    await within(board).findByRole('button', { name: `Gloves: item ${id}` });
    expect(tick, 'mounting the page ticked').not.toHaveBeenCalled();
  });

  it('opens in the Default items view setting: Board unless set to List; the page switch does not change it', async () => {
    const { itemsLayout } = await import('$lib/display/appearance');
    const pressed = (v: { getByRole: (r: string, o: { name: string }) => HTMLElement }, name: string) =>
      v.getByRole('button', { name }).getAttribute('aria-pressed');
    expect(get(itemsLayout)).toBe('board');
    let view = await mount();
    let layout = within(await view.findByRole('group', { name: 'Layout' }, { timeout: 10_000 }));
    expect([pressed(layout, 'Board'), pressed(layout, 'List')]).toEqual(['true', 'false']);
    await fireEvent.click(layout.getByRole('button', { name: 'List' }));
    expect(get(itemsLayout)).toBe('board');
    cleanup();
    itemsLayout.set('list');
    onTestFinished(() => itemsLayout.set('board'));
    view = await mount();
    layout = within(await view.findByRole('group', { name: 'Layout' }, { timeout: 10_000 }));
    expect([pressed(layout, 'Board'), pressed(layout, 'List')]).toEqual(['false', 'true']);
    expect(view.queryByRole('region', { name: 'Equipped items' })).toBeNull();
  });

  it('flask checkbox writes Active to Go, then ticks', async () => {
    const { findByRole } = await mount();
    const box = await findByRole('checkbox', { name: 'Flask 2 active' });
    expect(goSlot('Flask 2')?.Active ?? false, 'vacuity: Flask 2 starts active').toBe(false);
    tick.mockClear();
    await fireEvent.click(box);
    await vi.waitFor(() => expect(goSlot('Flask 2')?.Active).toBe(true));
    expect(tick).toHaveBeenCalledWith('SetFlaskActive');
  });

  it('a slot menu offers the items of Go’s list for the slot; choosing one issues EquipItemID with its id', async () => {
    const { findByRole } = await mount();
    const gloves = goSlot('Gloves')?.ItemID ?? 0;
    const other = goSlot('Flask 2')?.ItemID ?? 0;
    expect(gloves > 0 && other > 0, 'vacuity: 0280 has gloves and a Flask 2 item').toBe(true);
    await fireEvent.click(await findByRole('button', { name: 'Change Gloves' }));
    const menu = await findByRole('menu', { name: 'Gloves' });
    expect(
      within(menu)
        .getAllByRole('menuitem')
        .map((m) => m.textContent)
    ).toEqual(['Unequip', `Equip item ${other}`, 'Delete item…']);
    equipID.mockClear();
    await fireEvent.click(within(menu).getByRole('menuitem', { name: `Equip item ${other}` }));
    await vi.waitFor(() => expect(goSlot('Gloves')?.ItemID).toBe(other));
    expect(equipID.mock.calls).toEqual([['Gloves', other]]);
    workerInstance.currentBuild!.SetSlotItem('Gloves', gloves);
  });

  // ItemSlotControl.lua:12-19
  it('unequip issues EquipItemID with 0, which empties the slot in Go', async () => {
    const { findByRole } = await mount();
    const gloves = goSlot('Gloves')?.ItemID ?? 0;
    await fireEvent.click(await findByRole('button', { name: 'Change Gloves' }));
    const menu = await findByRole('menu', { name: 'Gloves' });
    const unequip = within(menu).getByRole('menuitem', { name: 'Unequip' });
    tick.mockClear();
    equipID.mockClear();
    await fireEvent.click(unequip);
    await vi.waitFor(() => expect(goSlot('Gloves')?.ItemID).toBe(0));
    expect(equipID.mock.calls).toEqual([['Gloves', 0]]);
    expect(tick).not.toHaveBeenCalledWith('UnequipSlot');
    workerInstance.currentBuild!.SetSlotItem('Gloves', gloves);
  });

  // ItemsTab.lua:335-340
  it('Edit as text, then Save, replaces the item under its id; Cancel discards the edit', async () => {
    const { findByRole, getByRole, queryByRole } = await mount();
    const id = goSlot('Gloves')?.ItemID ?? 0;
    expect(id, 'vacuity: 0280 has gloves').toBeGreaterThan(0);
    const count = goItems().length;
    const box = (await findByRole('textbox', { name: 'Item text' })) as HTMLTextAreaElement;

    const openEdit = async () => {
      await fireEvent.click(await findByRole('button', { name: `Gloves: item ${id}` }));
      await fireEvent.click(await findByRole('button', { name: 'Edit as text' }));
      await vi.waitFor(() => expect(box.value).toBe(OLD_GLOVES));
      await findByRole('heading', { name: `Editing item ${id}` });
    };

    await openEdit();
    tick.mockClear();
    await fireEvent.click(getByRole('button', { name: 'Cancel' }));
    await vi.waitFor(() => expect(box.value).toBe(''));
    expect(queryByRole('heading', { name: `Editing item ${id}` }), 'Cancel left the edit open').toBeNull();
    expect(tick, 'Cancel wrote').not.toHaveBeenCalled();

    await openEdit();
    await fireEvent.input(box, { target: { value: NEW_GLOVES } });
    const save = await findByRole('button', { name: 'Save' });
    saveItem.mockClear();
    replaceItem.mockClear();
    await fireEvent.click(save);
    await vi.waitFor(() => expect(replaceItem.mock.calls).toEqual([[id, NEW_GLOVES]]));
    expect(saveItem, 'the edit was added as a new item').not.toHaveBeenCalled();
    expect(goItemText(id)).toBe(NEW_GLOVES);
    expect(goSlot('Gloves')?.ItemID).toBe(id);
    expect(goItems().length).toBe(count);
  });

  it('typing an item issues no write', async () => {
    const { findByRole, queryByRole } = await mount();
    const before = JSON.stringify(goSlot('Helmet'));
    tick.mockClear();
    const box = await findByRole('textbox', { name: 'Item text' });
    await fireEvent.input(box, { target: { value: 'Rarity: Rare\nHarness Crown\nHubris Circlet\n' } });
    await vi.waitFor(() => expect(queryByRole('alert') ?? queryByRole('article')).not.toBeNull());
    expect(tick).not.toHaveBeenCalled();
    expect(JSON.stringify(goSlot('Helmet'))).toBe(before);
  });
});

// ItemsTab.lua:1459-1464, 1481-1484
describe('items page keys', () => {
  it('E over an equipped slot opens its item in the editor', async () => {
    const { findByRole } = await mount();
    const id = goSlot('Gloves')?.ItemID ?? 0;
    expect(id, 'vacuity: 0280 has gloves').toBeGreaterThan(0);
    await fireEvent.mouseEnter(await findByRole('button', { name: `Gloves: item ${id}` }));
    await fireEvent.keyDown(window, { key: 'e' });
    await vi.waitFor(() => expect([...document.querySelectorAll('dialog')].some((d) => d.open && (d.textContent ?? '').includes(`item ${id}`))).toBe(true));
  });

  // ItemsTab.lua:1471-1480
  it('Ctrl+F focuses the unique search, and from there the rare one', async () => {
    await mount();
    await fireEvent.keyDown(window, { key: 'f', ctrlKey: true });
    await vi.waitFor(() => expect(document.activeElement?.getAttribute('aria-label')).toBe('Search uniques'));
    await fireEvent.keyDown(window, { key: 'f', ctrlKey: true });
    await vi.waitFor(() => expect(document.activeElement?.getAttribute('aria-label')).toBe('Search templates'));
  });

  it('Ctrl+D saves the stat differences toggle in Go and ticks', async () => {
    tick.mockClear();
    await mount();
    const before = workerInstance.StatDifferencesShown();
    await fireEvent.keyDown(window, { key: 'd', ctrlKey: true });
    await vi.waitFor(() => expect(workerInstance.StatDifferencesShown()).toBe(!before));
    expect(tick.mock.calls.map((c) => c[0])).toContain('ToggleStatDifferences');
    await fireEvent.keyDown(window, { key: 'd', ctrlKey: true });
    await vi.waitFor(() => expect(workerInstance.StatDifferencesShown()).toBe(before));
  });
});

// ItemListControl.lua:275-284
describe('shared items', () => {
  const ptr = (x: number) => ({ pointerId: 1, isPrimary: true, button: 0, pointerType: 'mouse', clientX: x, clientY: 0 });
  const dragOnto = async (from: HTMLElement, tab: RegExp, region: string, find: (role: string, o: object) => Promise<HTMLElement>) => {
    await fireEvent.pointerDown(from, ptr(0));
    await fireEvent.pointerMove(from, ptr(20));
    await fireEvent.click(await find('tab', { name: tab }));
    const to = await find('region', { name: region });
    await fireEvent.pointerMove(to, ptr(30));
    await fireEvent.pointerUp(to, ptr(30));
    // a browser delivers the drag's own click in the same task as the pointerup
    await new Promise((r) => setTimeout(r, 0));
  };
  const sharedMount = async () => {
    vi.spyOn(workerInstance, 'GetSharedItems').mockResolvedValue([NEW_GLOVES]);
    const del = vi.spyOn(workerInstance, 'DeleteSharedItem').mockResolvedValue(true);
    const share = vi.spyOn(workerInstance, 'ShareItem').mockResolvedValue(true);
    const add = vi.spyOn(workerInstance, 'AddSharedItem').mockResolvedValue(true);
    const r = await mount();
    await fireEvent.click(await r.findByRole('tab', { name: /^Shared/ }));
    return { ...r, del, share, add };
  };

  it('lists the shared items, deletes one after its confirmation and copies its text', async () => {
    const { findByRole, del } = await sharedMount();
    const row = await findByRole('button', { name: 'Edited Grip' });
    const write = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', { value: { writeText: write }, configurable: true });
    await fireEvent.keyDown(row, { key: 'c', ctrlKey: true });
    await vi.waitFor(() => expect(write).toHaveBeenCalledWith(NEW_GLOVES.replace(/\n/g, '\r\n')));
    await fireEvent.click(await findByRole('button', { name: 'Delete Edited Grip' }));
    expect(del, 'deleted before the confirmation').not.toHaveBeenCalled();
    const dialog = await findByRole('dialog', { name: 'Delete Item' });
    expect(dialog.textContent).toContain("Are you sure you want to remove 'Edited Grip' from the shared item list?");
    await fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }));
    await vi.waitFor(() => expect(del).toHaveBeenCalledWith(0));
  });

  it('shares a build item dropped on it, and adds a shared item dropped on the stash', async () => {
    const { findByRole, share, add } = await sharedMount();
    const find = (role: string, o: object) => findByRole(role, o);
    await fireEvent.click(await findByRole('tab', { name: /^Stash/ }));
    const id = goSlot('Gloves')?.ItemID ?? 0;
    const stash = await findByRole('region', { name: 'Stash items' });
    await dragOnto(await within(stash).findByRole('button', { name: `item ${id}` }), /^Shared/, 'Shared items', find);
    await vi.waitFor(() => expect(share).toHaveBeenCalledWith(id, '', '', Number.MAX_SAFE_INTEGER));
    await dragOnto(await findByRole('button', { name: 'Edited Grip' }), /^Stash/, 'Stash items', find);
    await vi.waitFor(() => expect(add).toHaveBeenCalledWith(0, '', 0));
  });
});

// Build.lua:534-543
describe('minion item drop', () => {
  it('equips an item whose primary slot the minion uses and refuses one it does not', async () => {
    const slot = vi.spyOn(workerInstance, 'ItemPrimarySlot').mockResolvedValue('Gloves');
    const equip = vi.spyOn(workerInstance, 'EquipItemInSet').mockResolvedValue(true);
    await mount();
    const drop = (uses: string[]) =>
      window.dispatchEvent(new CustomEvent('gopob-minion-item-drop', { detail: { setId: 2, drag: { id: 7 }, second: false, uses } }));
    drop(['Weapon 1', 'Helmet']);
    await vi.waitFor(() => expect(slot).toHaveBeenCalledWith(7, '', ''));
    await new Promise((r) => setTimeout(r, 20));
    expect(equip, 'a glove the minion does not use').not.toHaveBeenCalled();
    drop(['Gloves']);
    await vi.waitFor(() => expect(equip).toHaveBeenCalledWith(2, 7, '', '', false));
  });
});

// ItemsTab.lua:1454-1458, 1859-1869
describe('item paste', () => {
  it('sends the pasted text through Go and shows what comes back', async () => {
    const norm = vi.spyOn(workerInstance, 'NormaliseDisplayItem').mockResolvedValue(NEW_GLOVES);
    const { findByRole } = await mount();
    const box = (await findByRole('textbox', { name: 'Item text' })) as HTMLTextAreaElement;
    await fireEvent.paste(box, { clipboardData: { getData: () => OLD_GLOVES } });
    await vi.waitFor(() => expect(norm).toHaveBeenCalledWith(OLD_GLOVES, true));
    await vi.waitFor(() => expect(box.value).toBe(NEW_GLOVES));
  });
});

// ItemsTab.lua:3651-3654, 4368
describe('item popups', () => {
  it("Add modifier's preview shows the item's slot comparison from Go", async () => {
    const spies = [
      vi.spyOn(workerInstance, 'NormaliseDisplayItem').mockResolvedValue(NEW_GLOVES),
      vi.spyOn(workerInstance, 'GetDisplayItem').mockResolvedValue({
        quality: 0,
        hasQuality: false,
        variants: [],
        variant: 0,
        altVariants: [],
        sockets: '',
        socketLimit: 0,
        defaultColor: '',
        canInfluence: false,
        influence: [],
        showsCatalyst: false,
        catalyst: 0,
        catalystQuality: 0,
        mods: []
      }),
      vi.spyOn(workerInstance, 'GetDisplayItemCraft').mockResolvedValue({ addCustom: true }),
      vi.spyOn(workerInstance, 'GetAffixControls').mockResolvedValue(undefined),
      vi.spyOn(workerInstance, 'GetDisplayItemExtras').mockResolvedValue(undefined),
      vi.spyOn(workerInstance, 'RebaseChoices').mockResolvedValue([]),
      vi.spyOn(workerInstance, 'GetCustomModChoices').mockResolvedValue({ sources: [{ id: 'CUSTOM', label: 'Custom' }], mods: {} }),
      vi.spyOn(workerInstance, 'CustomModText').mockResolvedValue(NEW_GLOVES)
    ];
    const compare = vi
      .spyOn(workerInstance, 'ItemCompare')
      .mockImplementation((text: string) =>
        Promise.resolve(
          text === NEW_GLOVES ? { blocks: [{ header: 'Equipping this item in Gloves will give you:', slot: 'Gloves', sections: [] }], calcMs: 0 } : undefined
        )
      );
    onTestFinished(() => {
      spies.forEach((s) => s.mockRestore());
      compare.mockResolvedValue(undefined);
    });
    const { findByRole } = await mount();
    const box = (await findByRole('textbox', { name: 'Item text' })) as HTMLTextAreaElement;
    await fireEvent.paste(box, { clipboardData: { getData: () => NEW_GLOVES } });
    await fireEvent.click(await findByRole('button', { name: 'Add modifier...' }));
    const dialog = within(await findByRole('dialog', { name: 'Add Modifier to Item' }));
    await fireEvent.input(await dialog.findByRole('textbox', { name: 'Custom modifier' }), { target: { value: '+7 to Dexterity' } });
    const preview = within(await dialog.findByRole('group', { name: 'New item' }));
    await vi.waitFor(() =>
      expect(preview.getByRole('group', { name: 'Stat differences' }).textContent).toContain('Equipping this item in Gloves will give you:')
    );
    expect(preview.queryByText('Fits no equipment slot')).toBeNull();
  });
});
