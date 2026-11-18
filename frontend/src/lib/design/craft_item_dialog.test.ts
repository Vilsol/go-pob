// @vitest-environment jsdom
// ItemsTab.lua:2519-2603

import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, within } from '@testing-library/svelte';
import CraftItemDialog from './CraftItemDialog.svelte';

afterEach(cleanup);

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
  };
}

const groups = [
  { type: 'Amulet', bases: ['Coral Amulet', 'Onyx Amulet'] },
  { type: 'Helmet', bases: ['Iron Hat', 'Hubris Circlet', 'Lion Pelt'] }
];

describe('CraftItemDialog', () => {
  it('picks the type, then a base typed into its filter; hovering a base previews it (:2582-2587); Create hands it back', async () => {
    const onpreview = vi.fn();
    const oncreate = vi.fn();
    render(CraftItemDialog, { open: true, groups, onpreview, oncreate });
    await vi.waitFor(() => expect(onpreview).toHaveBeenLastCalledWith({ rarity: 2, title: '', base: 'Coral Amulet' }));

    const type = screen.getByRole('combobox', { name: 'Type' });
    for (const k of 'helm') await fireEvent.keyDown(type, { key: k });
    await fireEvent.keyDown(type, { key: 'Enter' });
    await vi.waitFor(() => expect(onpreview).toHaveBeenLastCalledWith({ rarity: 2, title: '', base: 'Iron Hat' }));

    const base = screen.getByRole('combobox', { name: 'Base' });
    for (const k of 'circ') await fireEvent.keyDown(base, { key: k });
    const list = () => within(screen.getByRole('listbox', { name: 'Base' }));
    expect(
      list()
        .getAllByRole('option')
        .map((o) => o.textContent?.trim())
    ).toEqual(['Hubris Circlet']);
    await vi.waitFor(() => expect(onpreview).toHaveBeenLastCalledWith({ rarity: 2, title: '', base: 'Hubris Circlet' }));
    await fireEvent.keyDown(base, { key: 'Backspace' });
    await fireEvent.keyDown(base, { key: 'Backspace' });
    await fireEvent.keyDown(base, { key: 'Backspace' });
    await fireEvent.keyDown(base, { key: 'Backspace' });
    await fireEvent.mouseEnter(list().getByRole('option', { name: 'Lion Pelt' }));
    await vi.waitFor(() => expect(onpreview).toHaveBeenLastCalledWith({ rarity: 2, title: '', base: 'Lion Pelt' }));
    await fireEvent.click(list().getByRole('option', { name: 'Hubris Circlet' }));
    await vi.waitFor(() => expect(onpreview).toHaveBeenLastCalledWith({ rarity: 2, title: '', base: 'Hubris Circlet' }));

    await fireEvent.click(screen.getByRole('button', { name: 'Create' }));
    expect(oncreate).toHaveBeenCalledWith({ rarity: 2, title: '', base: 'Hubris Circlet' });
  });

  // Data.lua:1256-1281, DropDownControl.lua:24-33
  it('finds a base by its implicit, enchant or flask buff, not only its name', async () => {
    const searchable = [
      {
        type: 'Amulet: Talisman',
        bases: ['Chieftain Talisman', 'Ape Talisman'],
        searchFilters: ['Chieftain Talisman 16% increased Area of Effect', 'Ape Talisman +1 to Minimum Endurance, Frenzy and Power Charges']
      }
    ];
    render(CraftItemDialog, { open: true, groups: searchable, onpreview: vi.fn(), oncreate: vi.fn() });
    const base = screen.getByRole('combobox', { name: 'Base' });
    for (const k of 'area') await fireEvent.keyDown(base, { key: k });
    expect(
      within(screen.getByRole('listbox', { name: 'Base' }))
        .getAllByRole('option')
        .map((o) => o.textContent?.trim())
    ).toEqual(['Chieftain Talisman']);
  });
});
