// @vitest-environment jsdom
// CompareTab.lua:1529-1640

import { describe, expect, it, afterEach, beforeAll } from 'vitest';
import { render, cleanup, fireEvent, within } from '@testing-library/svelte';
import BuildPicker from './BuildPicker.svelte';
import { buildList } from './fixtures/app';
import type { SavedBuild } from './fixtures/app';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted picker threw').toEqual([]);
});

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}

const id = (b: SavedBuild) => (b.folder ? `${b.folder}/${b.name}` : b.name);
const open = (builds: SavedBuild[] | null = buildList.builds) => {
  const picked: SavedBuild[] = [];
  const view = render(BuildPicker, {
    open: true,
    builds: builds ?? undefined,
    folders: buildList.folders,
    action: 'Compare',
    onpick: (b: SavedBuild) => picked.push(b)
  });
  const d = within(view.getByRole('dialog', { name: 'Choose a build' }));
  const list = () => d.getByRole('listbox');
  const options = () =>
    within(list())
      .getAllByRole('option')
      .map((o) => o.dataset.folder ?? `build:${o.dataset.build}`);
  return { d, list, options, picked, dialog: () => view.getByRole('dialog', { hidden: true }) as HTMLDialogElement };
};

describe('BuildPicker', () => {
  it('lists the top level’s folders, then its builds by the sort; a folder click goes in, the breadcrumb comes back', async () => {
    const { d, list, options } = open();
    const top = buildList.builds;
    expect(options()).toEqual(['League starters', 'Bossers', 'Imported', ...[...top].sort((a, b) => a.edited - b.edited).map((b) => `build:${id(b)}`)]);
    await fireEvent.change(d.getByRole('combobox', { name: 'Sort by' }), { target: { value: 'name' } });
    expect(options().slice(3)).toEqual([...top].sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase())).map((b) => `build:${id(b)}`));

    await fireEvent.click(within(list()).getByText('Bossers'));
    expect(list().getAttribute('aria-label')).toBe('Builds in Bossers');
    expect(options()).toEqual(['Bossers/Uber', 'build:Bossers/Stormblast Mine Saboteur']);
    await fireEvent.click(within(d.getByRole('navigation', { name: 'Folder' })).getByRole('button', { name: 'All builds' }));
    expect(list().getAttribute('aria-label')).toBe('Builds in all folders');
  });

  it('search walks the open folder and its subfolders', async () => {
    const { d, options } = open();
    await fireEvent.input(d.getByRole('searchbox', { name: 'Search builds' }), { target: { value: 'necro' } });
    expect(options()).toEqual(['build:League starters/Necromancer/Spectre Necro – mapping', 'build:League starters/Necromancer/Budget/Spectre Necro – budget']);
  });

  it('keyboard: ↓ / ↑ / End move, Enter opens a folder, Backspace goes up, Enter picks a build and closes', async () => {
    const { d, list, options, picked, dialog } = open();
    const box = list();
    const active = () => box.getAttribute('aria-activedescendant') && document.getElementById(box.getAttribute('aria-activedescendant')!);
    expect(active()).toHaveProperty('dataset.folder', 'League starters');
    await fireEvent.keyDown(box, { key: 'ArrowDown' });
    await fireEvent.keyDown(box, { key: 'ArrowDown' });
    expect(active()).toHaveProperty('dataset.folder', 'Imported');
    await fireEvent.keyDown(box, { key: 'ArrowUp' });
    await fireEvent.keyDown(box, { key: 'Enter' });
    expect(options()[0]).toBe('Bossers/Uber');
    await fireEvent.keyDown(box, { key: 'Backspace' });
    expect(options()[0]).toBe('League starters');
    expect(d.getByRole<HTMLButtonElement>('button', { name: 'Compare' }).disabled).toBe(true);
    await fireEvent.keyDown(box, { key: 'End' });
    const last = active() as HTMLElement;
    expect(last.dataset.build).toBeTruthy();
    expect(d.getByRole<HTMLButtonElement>('button', { name: 'Compare' }).disabled).toBe(false);
    await fireEvent.keyDown(box, { key: 'Enter' });
    expect(picked.map(id)).toEqual([last.dataset.build]);
    expect(dialog().open).toBe(false);
  });

  it('mouse: a click selects a build, Compare picks it; a double-click picks at once', async () => {
    let { d, list, picked } = open();
    const row = (name: string) => within(list()).getByText(name).closest<HTMLElement>('[role="option"]')!;
    await fireEvent.click(row('Ice Spear Totems'));
    expect(picked).toEqual([]);
    expect(row('Ice Spear Totems').getAttribute('aria-selected')).toBe('true');
    await fireEvent.click(d.getByRole('button', { name: 'Compare' }));
    expect(picked.map((b) => b.name)).toEqual(['Ice Spear Totems']);
    cleanup();
    ({ d, list, picked } = open());
    await fireEvent.dblClick(row('Ice Spear Totems'));
    expect(picked.map((b) => `${b.folder}/${b.name}`)).toEqual(['Imported/pobb.in/Ice Spear Totems']);
  });

  it('says it is loading until the builds arrive', () => {
    const { list } = open(null);
    expect(list().getAttribute('aria-busy')).toBe('true');
    expect(list().textContent).toContain('Loading');
  });
});
