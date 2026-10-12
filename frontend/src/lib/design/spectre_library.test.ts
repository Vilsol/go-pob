// @vitest-environment jsdom
// Build.lua:1453-1482, MinionListControl.lua:11-103, MinionSearchListControl.lua:16-67
// Build.lua:552-554, 1621

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent, within } from '@testing-library/svelte';
import SpectreLibrary from './SpectreLibrary.svelte';
import Sidebar from './Sidebar.svelte';
import { spectreMatches, type SpectreRow } from './spectres';
import type { calculator } from '$lib/types';

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

const row = (id: string, name: string, skills: string[]): SpectreRow => ({ id, name, skills, tooltip: [`^7${name}`, '^7Life Multiplier: x1.00'] });
const library = [row('A', 'Arena Master', ['Cyclone']), row('B', 'Bone Husk', ['Fireball', 'Ice Nova']), row('C', 'Cannibal', ['Flame Dash'])];

describe('spectreMatches', () => {
  it('searches names, skills or both, with "|" alternatives, case-insensitively', () => {
    expect(spectreMatches(library[1], 'husk', 1)).toBe(true);
    expect(spectreMatches(library[1], 'fireball', 1)).toBe(false);
    expect(spectreMatches(library[1], 'FIREBALL', 2)).toBe(true);
    expect(spectreMatches(library[1], 'husk', 2)).toBe(false);
    expect(spectreMatches(library[0], 'zzz|arena', 3)).toBe(true);
    expect(spectreMatches(library[2], '  ', 1)).toBe(true);
  });
});

describe('SpectreLibrary', () => {
  it('adds, removes and saves the build’s list; Cancel saves nothing', async () => {
    const saved: string[][] = [];
    const { getByRole } = render(SpectreLibrary, { props: { open: true, library, spectres: [library[2]], onsave: (ids: string[]) => saved.push(ids) } });
    const dest = getByRole('listbox', { name: 'Spectres in Build' });
    const src = getByRole('listbox', { name: 'Available Spectres' });
    expect(
      within(dest)
        .getAllByRole('option')
        .map((o) => o.textContent)
    ).toEqual(['Cannibal']);

    await fireEvent.input(getByRole('textbox', { name: 'Search spectres' }), { target: { value: 'nova' } });
    expect(within(src).queryAllByRole('option')).toHaveLength(0);
    await fireEvent.change(getByRole('combobox', { name: 'Search in' }), { target: { value: '2' } });
    expect(
      within(src)
        .getAllByRole('option')
        .map((o) => o.textContent)
    ).toEqual(['Bone Husk']);
    await fireEvent.dblClick(within(src).getByRole('option', { name: 'Bone Husk' }));
    expect(
      within(dest)
        .getAllByRole('option')
        .map((o) => o.textContent)
    ).toEqual(['Cannibal', 'Bone Husk']);
    await fireEvent.mouseEnter(within(dest).getByRole('option', { name: 'Bone Husk' }));
    expect(getByRole('tooltip').textContent).toContain('Life Multiplier: x1.00');

    await fireEvent.click(within(dest).getByRole('option', { name: 'Cannibal' }));
    await fireEvent.click(getByRole('button', { name: 'Remove' }));
    await fireEvent.click(getByRole('button', { name: 'Save' }));
    expect(saved).toEqual([['B']]);
  });

  // MinionListControl.lua:79-89
  it('drags a spectre into the build’s list and reorders it', async () => {
    const saved: string[][] = [];
    const { getByRole } = render(SpectreLibrary, {
      props: { open: true, library, spectres: [library[0], library[2]], onsave: (ids: string[]) => saved.push(ids) }
    });
    const dest = getByRole('listbox', { name: 'Spectres in Build' });
    const src = getByRole('listbox', { name: 'Available Spectres' });
    const names = () =>
      within(dest)
        .getAllByRole('option')
        .map((o) => o.textContent);
    await fireEvent.dragStart(within(src).getByRole('option', { name: 'Bone Husk' }));
    const target = within(dest).getByRole('option', { name: 'Cannibal' });
    const over = new Event('dragover', { bubbles: true, cancelable: true });
    target.dispatchEvent(over);
    expect(over.defaultPrevented, 'the build list takes a spectre it does not hold').toBe(true);
    await fireEvent.drop(target);
    expect(names()).toEqual(['Arena Master', 'Bone Husk', 'Cannibal']);

    await fireEvent.dragStart(within(src).getByRole('option', { name: 'Cannibal' }));
    const again = new Event('dragover', { bubbles: true, cancelable: true });
    within(dest).getByRole('option', { name: 'Arena Master' }).dispatchEvent(again);
    expect(again.defaultPrevented, 'nor one it holds').toBe(false);

    await fireEvent.keyDown(within(dest).getAllByRole('group')[0], { key: 'ArrowDown', altKey: true });
    expect(names()).toEqual(['Bone Husk', 'Arena Master', 'Cannibal']);
    await fireEvent.click(getByRole('button', { name: 'Save' }));
    expect(saved).toEqual([['B', 'A', 'C']]);
  });
});

describe('Sidebar Manage Spectres', () => {
  const controls = (shown: boolean) =>
    ({
      SkillSet: 0,
      SocketGroup: 0,
      GemIndex: 0,
      Skills: ['Raise Spectre'],
      MinionShown: true,
      MinionLibraryShown: shown,
      Minions: [],
      Minion: 1
    }) as unknown as calculator.MainSkillControls;
  it('shows the button only when the controls ask for it', async () => {
    let opened = 0;
    const props = { sidebar: { sections: [], warnings: [] }, socketGroups: ['G'], mainSocketGroup: 0, onspectres: () => opened++ };
    const a = render(Sidebar, { props: { ...props, controls: controls(true) } });
    await fireEvent.click(a.getByRole('button', { name: 'Manage Spectres...' }));
    expect(opened).toBe(1);
    cleanup();
    const b = render(Sidebar, { props: { ...props, controls: controls(false) } });
    expect(b.queryByRole('button', { name: 'Manage Spectres...' })).toBeNull();
  });
});
