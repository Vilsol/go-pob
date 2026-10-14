// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within, waitFor, screen } from '@testing-library/svelte';
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

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}

type Entry = { Name: string; Type: 'file' | 'dir'; Class: string; LastEdit: string; Level: number; Ascendancy: string; MainSkill: string };
type Saved = { Class: string; Level: number; Ascendancy?: string; MainSkill?: string };
const files = new Map<string, Saved>();
const dirs = new Set<string>();
const seed = () => {
  files.clear();
  dirs.clear();
  dirs.add('/Leagues');
  dirs.add('/Leagues/Old');
  files.set('/Top build', { Class: 'Witch', Level: 90, Ascendancy: 'Occultist', MainSkill: 'Bane, Contagion' });
  files.set('/Leagues/Spark', { Class: 'Templar', Level: 95 });
  files.set('/Leagues/Old/Spark', { Class: 'Templar', Level: 80 });
};
const parent = (p: string) => p.slice(0, p.lastIndexOf('/'));
const calls: [string, ...unknown[]][] = [];
const fake = {
  ListBuilds: vi.fn((dir: string) => {
    calls.push(['ListBuilds', dir]);
    const out: Entry[] = [];
    for (const d of dirs)
      if (parent(d) === dir)
        out.push({ Name: d.slice(dir.length + 1), Type: 'dir', Class: '', LastEdit: new Date().toISOString(), Level: 0, Ascendancy: '', MainSkill: '' });
    for (const [f, v] of files)
      if (parent(f) === dir)
        out.push({
          Name: f.slice(dir.length + 1),
          Type: 'file',
          Class: v.Class,
          LastEdit: new Date(Date.now() - 3 * 3600_000).toISOString(),
          Level: v.Level,
          Ascendancy: v.Ascendancy ?? '',
          MainSkill: v.MainSkill ?? ''
        });
    return Promise.resolve(out);
  }),
  NewFolder: vi.fn((p: string) => {
    calls.push(['NewFolder', p]);
    dirs.add(p);
    return Promise.resolve();
  }),
  Copy: vi.fn((from: string, to: string) => {
    calls.push(['Copy', from, to]);
    files.set(to, files.get(from)!);
    return Promise.resolve();
  }),
  Rename: vi.fn((from: string, to: string) => {
    calls.push(['Rename', from, to]);
    files.set(to, files.get(from)!);
    files.delete(from);
    return Promise.resolve();
  }),
  CopyFolder: vi.fn((from: string, to: string) => {
    calls.push(['CopyFolder', from, to]);
    return Promise.resolve();
  }),
  MoveFolder: vi.fn((from: string, to: string) => {
    calls.push(['MoveFolder', from, to]);
    return Promise.resolve();
  }),
  Delete: vi.fn((p: string) => {
    calls.push(['Delete', p]);
    files.delete(p);
    return Promise.resolve();
  }),
  OpenBuild: vi.fn((p: string) => {
    calls.push(['OpenBuild', p]);
    return Promise.resolve();
  }),
  SaveBuildAs: vi.fn((p: string) => {
    calls.push(['SaveBuildAs', p]);
    files.set(p, { Class: 'Witch', Level: 1 });
    return Promise.resolve();
  }),
  NewBuild: vi.fn(() => {
    calls.push(['NewBuild']);
    return Promise.resolve();
  }),
  SetShowFlavourText: vi.fn(() => Promise.resolve()),
  SetDefaultItemAffixQuality: vi.fn(() => Promise.resolve()),
  SetItemModDebug: vi.fn(() => Promise.resolve()),
  GameDataVersion: vi.fn(() => Promise.resolve('3.28')),
  GetTreeVersion: vi.fn(() => Promise.resolve('3_29')),
  BuildInfo: vi.fn(() =>
    Promise.resolve({
      GoVersion: 'go1.26.2',
      Main: { Path: 'github.com/Vilsol/go-pob', Version: 'v1.2.3' },
      Settings: [{ Key: 'GOOS', Value: 'js' }],
      Deps: { 0: { Path: 'github.com/Vilsol/go-pob-data', Version: 'v0.9.0' }, __crystallineHandle: 7 }
    })
  )
};

vi.mock('$lib/go/worker', () => ({ syncWrap: fake, syncWorker: undefined }));
const nav = vi.hoisted(() => ({ pushState: vi.fn(), goto: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/navigation', () => nav);
vi.mock('$app/state', () => ({ page: { state: {}, url: new URL('http://localhost/') } }));

const { currentBuildPath, devMode, nodePowerTheme, zoomSensitivity } = await import('$lib/global');
const { defaultCharLevel, defaultGemQuality, showFlavourText, showWarnings } = await import('$lib/display/item_options');
const { unsavedName } = await import('./build_name.svelte');

const mount = async () => {
  seed();
  calls.length = 0;
  const Page = (await import('./AppBuildList.svelte')).default;
  const view = render(Page);
  const grid = view.getByRole('grid', { name: 'Builds' });
  await waitFor(() => expect(grid.querySelectorAll('[data-build]').length).toBe(3));
  calls.length = 0;
  return { view, grid };
};
const rowOf = (grid: HTMLElement, name: string) => grid.querySelector<HTMLElement>(`[data-build="${name}"]`)!;

const dialog = () => screen.getByRole('dialog');
const menuItem = async (build: HTMLElement, label: string) => {
  await fireEvent.click(within(build).getByRole('button', { name: /^Actions for / }));
  await fireEvent.click(screen.getByRole('menuitem', { name: label }));
};

describe('AppBuildList', () => {
  it('lists every folder through ListBuilds, top level first, and shows each build with its class and level', async () => {
    seed();
    calls.length = 0;
    const Page = (await import('./AppBuildList.svelte')).default;
    const view = render(Page);
    const grid = view.getByRole('grid', { name: 'Builds' });
    await waitFor(() => expect(grid.querySelectorAll('[data-build]').length).toBe(3));
    expect(calls).toEqual([
      ['ListBuilds', ''],
      ['ListBuilds', '/Leagues'],
      ['ListBuilds', '/Leagues/Old']
    ]);
    expect([...grid.querySelectorAll<HTMLElement>('[data-build]')].map((r) => r.dataset.build).sort()).toEqual(['Spark', 'Spark', 'Top build']);
    expect(within(view.getByRole('tree', { name: 'Folder tree' })).getByRole('button', { name: /^Leagues/ })).toBeTruthy();
    const heads = within(grid)
      .getAllByTitle(/^Sort by /)
      .map((h) => h.title);
    expect(heads).toEqual(['Sort by Name', 'Sort by Class', 'Sort by Ascendancy', 'Sort by Level', 'Sort by Main skill', 'Sort by Modified']);
    const top = rowOf(grid, 'Top build');
    expect(within(top).getByText('Occultist')).toBeTruthy();
    expect(within(top).getByText('Bane, Contagion')).toBeTruthy();
    expect(view.getByText('game data 3.28')).toBeTruthy();
  });

  it('New build makes an unnamed build (NewBuild, no path)', async () => {
    const { view } = await mount();
    currentBuildPath.current = '/somewhere';
    await fireEvent.click(view.getByRole('button', { name: 'New build' }));
    expect(calls).toEqual([['NewBuild']]);
    expect(currentBuildPath.current).toBe('');
  });

  // Main.lua:1085-1113, 1020-1024
  it('the default level and gem quality reach New build; warnings and flavour text toggle', async () => {
    const { view } = await mount();
    await fireEvent.click(view.getByRole('button', { name: 'Settings' }));
    const s = within(view.getByRole('dialog', { name: 'Settings' }));
    await fireEvent.click(s.getByRole('tab', { name: /^General/ }));
    await fireEvent.click(s.getByRole('checkbox', { name: 'Show build warnings' }));
    expect(get(showWarnings)).toBe(false);
    // ItemsTab.lua:5160-5166
    await fireEvent.click(s.getByRole('checkbox', { name: 'Dev mode' }));
    await waitFor(() => expect(fake.SetItemModDebug).toHaveBeenLastCalledWith(true));
    await fireEvent.click(s.getByRole('checkbox', { name: 'Dev mode' }));
    await waitFor(() => expect(fake.SetItemModDebug).toHaveBeenLastCalledWith(false));
    await fireEvent.click(s.getByRole('tab', { name: /^Items/ }));
    await fireEvent.click(s.getByRole('checkbox', { name: 'Styled tooltips with flavour text' }));
    await waitFor(() => expect(fake.SetShowFlavourText).toHaveBeenLastCalledWith(false));
    await fireEvent.click(s.getByRole('button', { name: 'Cancel' }));
    expect([get(showWarnings), get(showFlavourText)]).toEqual([true, true]);
    defaultCharLevel.set(90);
    defaultGemQuality.set(20);
    await fireEvent.click(view.getByRole('button', { name: 'New build' }));
    expect(fake.NewBuild).toHaveBeenLastCalledWith({ level: 90, gemQuality: 20 });
    defaultCharLevel.set(1);
    defaultGemQuality.set(0);
  });

  it('Import build makes a new build and goes to Import / Export', async () => {
    const { view } = await mount();
    nav.goto.mockClear();
    await fireEvent.click(view.getByRole('button', { name: 'Import build' }));
    expect(calls).toEqual([['NewBuild']]);
    expect(nav.goto).toHaveBeenCalledWith('/import');
  });

  it('double-click, Enter and the name open a build by its storage path (OpenBuild) and name the open build', async () => {
    const { view, grid } = await mount();
    const nested = [...grid.querySelectorAll<HTMLElement>('[data-build="Spark"]')].find((r) => r.textContent?.includes('80'))!;
    await fireEvent.dblClick(nested);
    expect(calls).toEqual([['OpenBuild', '/Leagues/Old/Spark']]);
    expect(currentBuildPath.current).toBe('/Leagues/Old/Spark');
    calls.length = 0;
    const top = rowOf(grid, 'Top build');
    await fireEvent.keyDown(top, { key: 'Enter' });
    await fireEvent.click(within(top).getByRole('link', { name: 'Top build' }));
    expect(calls).toEqual([
      ['OpenBuild', '/Top build'],
      ['OpenBuild', '/Top build']
    ]);
    expect(view.queryByRole('dialog')).toBeNull();
  });

  it('Copy asks for a name and copies within the build folder (Copy), then lists again', async () => {
    const { grid } = await mount();
    const nested = [...grid.querySelectorAll<HTMLElement>('[data-build="Spark"]')].find((r) => r.textContent?.includes('95'))!;
    await fireEvent.click(within(nested).getByRole('button', { name: 'Copy Spark' }));
    const box = within(dialog()).getByRole<HTMLInputElement>('textbox', { name: 'Name of the copy' });
    expect(box.value).toBe('Spark (copy)');
    await fireEvent.input(box, { target: { value: 'Spark 2' } });
    await fireEvent.click(within(dialog()).getByRole('button', { name: 'Copy' }));
    await waitFor(() => expect(calls[0]).toEqual(['Copy', '/Leagues/Spark', '/Leagues/Spark 2']));
    await waitFor(() => expect(grid.querySelectorAll('[data-build]').length).toBe(4));
    expect(calls.slice(1).every((c) => c[0] === 'ListBuilds')).toBe(true);
  });

  it('Rename (⋯) renames within the folder (Rename); an empty name cannot be confirmed', async () => {
    const { grid } = await mount();
    await menuItem(rowOf(grid, 'Top build'), 'Rename…');
    const d = within(dialog());
    const box = d.getByRole<HTMLInputElement>('textbox', { name: 'New name' });
    expect(box.value).toBe('Top build');
    await fireEvent.input(box, { target: { value: '' } });
    expect(d.getByRole<HTMLButtonElement>('button', { name: 'Rename' }).disabled).toBe(true);
    await fireEvent.input(box, { target: { value: 'Best build' } });
    await fireEvent.keyDown(box, { key: 'Enter' });
    await waitFor(() => expect(calls[0]).toEqual(['Rename', '/Top build', '/Best build']));
    await waitFor(() => expect(grid.querySelector('[data-build="Best build"]')).not.toBeNull());
  });

  it('Delete (⋯) asks first, and Cancel deletes nothing', async () => {
    const { view, grid } = await mount();
    await menuItem(rowOf(grid, 'Top build'), 'Delete…');
    await fireEvent.click(within(view.getByRole('dialog', { name: 'Delete “Top build”?' })).getByRole('button', { name: 'Cancel' }));
    expect(calls).toEqual([]);
    await menuItem(rowOf(grid, 'Top build'), 'Delete…');
    await fireEvent.click(within(view.getByRole('dialog', { name: 'Delete “Top build”?' })).getByRole('button', { name: 'Delete' }));
    await waitFor(() => expect(calls[0]).toEqual(['Delete', '/Top build']));
    await waitFor(() => expect(grid.querySelector('[data-build="Top build"]')).toBeNull());
  });

  it('Move to… renames into the folder, and asks before replacing a build of that name', async () => {
    const { view, grid } = await mount();
    await menuItem(rowOf(grid, 'Top build'), 'Old');
    await waitFor(() => expect(calls[0]).toEqual(['Rename', '/Top build', '/Leagues/Old/Top build']));
    calls.length = 0;
    const spark = [...grid.querySelectorAll<HTMLElement>('[data-build="Spark"]')].find((r) => r.textContent?.includes('95'))!;
    await menuItem(spark, 'Old');
    const ask = view.getByRole('dialog', { name: 'Replace “Spark”?' });
    expect(calls).toEqual([]);
    await fireEvent.click(within(ask).getByRole('button', { name: 'Replace' }));
    await waitFor(() => expect(calls[0]).toEqual(['Rename', '/Leagues/Spark', '/Leagues/Old/Spark']));
  });

  it('New folder is made in the open folder (NewFolder) and appears in the tree', async () => {
    const { view } = await mount();
    await fireEvent.click(view.getByRole('button', { name: 'New folder' }));
    const box = within(dialog()).getByRole('textbox', { name: 'Folder name' });
    await fireEvent.input(box, { target: { value: 'Bossers' } });
    await fireEvent.click(within(dialog()).getByRole('button', { name: 'Create' }));
    await waitFor(() => expect(calls[0]).toEqual(['NewFolder', '/Bossers']));
    await waitFor(() => expect(within(view.getByRole('tree', { name: 'Folder tree' })).getByRole('button', { name: /^Bossers/ })).toBeTruthy());
  });

  it('picking a folder pushes it as history state (Back walks the breadcrumb)', async () => {
    const { view } = await mount();
    nav.pushState.mockClear();
    await fireEvent.click(within(view.getByRole('tree', { name: 'Folder tree' })).getByRole('button', { name: /^Leagues/ }));
    expect(nav.pushState).toHaveBeenCalledWith('', { folder: 'Leagues' });
  });

  it("Settings change the app's stores at once; About shows Go's build info", async () => {
    const { view } = await mount();
    devMode.set(false);
    zoomSensitivity.set(5);
    await fireEvent.click(view.getByRole('button', { name: 'Settings' }));
    const s = within(view.getByRole('dialog', { name: 'Settings' }));
    await fireEvent.click(s.getByRole('checkbox', { name: 'Dev mode' }));
    expect(get(devMode)).toBe(true);
    await fireEvent.click(s.getByRole('tab', { name: /^Passive tree/ }));
    await fireEvent.input(s.getByRole('slider', { name: 'Zoom sensitivity' }), { target: { value: '7.5' } });
    expect(get(zoomSensitivity)).toBe(7.5);
    // Main.lua:957-961
    await fireEvent.change(s.getByRole('combobox', { name: 'Node Power colours' }), { target: { value: 'GREEN/BLUE' } });
    expect(get(nodePowerTheme)).toBe('GREEN/BLUE');
    expect(localStorage.getItem('options:nodePowerTheme')).toBe('GREEN/BLUE');
    await fireEvent.click(s.getByRole('button', { name: 'Reset this section' }));
    expect(get(zoomSensitivity)).toBe(5);
    expect(get(nodePowerTheme)).toBe('RED/BLUE');
    expect(s.queryByText(/^Preview:/)).toBeNull();
    await fireEvent.click(s.getByRole('button', { name: 'Save' }));
    devMode.set(false);

    // Main.lua:1191-1225
    await fireEvent.click(view.getByRole('button', { name: 'Settings' }));
    const c = within(view.getByRole('dialog', { name: 'Settings' }));
    await fireEvent.click(c.getByRole('tab', { name: /^General/ }));
    await fireEvent.click(c.getByRole('checkbox', { name: 'Dev mode' }));
    await fireEvent.click(c.getByRole('tab', { name: /^Passive tree/ }));
    await fireEvent.input(c.getByRole('slider', { name: 'Zoom sensitivity' }), { target: { value: '3' } });
    expect([get(devMode), get(zoomSensitivity)]).toEqual([true, 3]);
    await fireEvent.click(c.getByRole('button', { name: 'Cancel' }));
    expect([get(devMode), get(zoomSensitivity)]).toEqual([false, 5]);

    await fireEvent.click(view.getByRole('button', { name: 'About go-pob' }));
    const a = within(view.getByRole('dialog', { name: 'About go-pob' }));
    await waitFor(() => expect(a.getAllByText('v1.2.3').length).toBeGreaterThan(0));
    expect(a.getByText('3.28')).toBeTruthy();
    expect(a.getByText('3_29')).toBeTruthy();
    await fireEvent.click(a.getByRole('tab', { name: 'Build settings' }));
    expect(a.getByText('GOOS')).toBeTruthy();
    await fireEvent.click(a.getByRole('tab', { name: 'Dependencies' }));
    expect(a.getByText('github.com/Vilsol/go-pob-data')).toBeTruthy();
    expect(a.getByText('v0.9.0')).toBeTruthy();
    expect(calls).toEqual([]);
  });
});

describe('SaveAsDialog', () => {
  it('saves the open build in the picked folder (SaveBuildAs) and names it; New folder is made inside that folder', async () => {
    seed();
    calls.length = 0;
    currentBuildPath.current = '';
    const Dlg = (await import('./SaveAsDialog.svelte')).default;
    const onsaved = vi.fn();
    render(Dlg, { open: true, onsaved });
    const d = within(screen.getByRole('dialog', { name: 'Save build as' }));
    const folder = d.getByRole<HTMLSelectElement>('combobox', { name: 'Folder' });
    await waitFor(() => expect([...folder.options].map((o) => o.value)).toEqual(['', 'Leagues', 'Leagues/Old']));
    await fireEvent.change(folder, { target: { value: 'Leagues' } });
    await fireEvent.click(d.getByRole('button', { name: 'New folder' }));
    const ask = within(screen.getByRole('dialog', { name: 'New folder' }));
    await fireEvent.input(ask.getByRole('textbox', { name: 'Folder name' }), { target: { value: 'Mapping' } });
    await fireEvent.click(ask.getByRole('button', { name: 'Create' }));
    await waitFor(() => expect(folder.value).toBe('Leagues/Mapping'));
    expect(calls).toContainEqual(['NewFolder', '/Leagues/Mapping']);
    calls.length = 0;
    const name = d.getByRole<HTMLInputElement>('textbox', { name: 'Build name' });
    expect(name.value).toBe('Unnamed build');
    await fireEvent.input(name, { target: { value: 'Spark mapper' } });
    await fireEvent.click(d.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(onsaved).toHaveBeenCalledOnce());
    expect(calls).toEqual([['SaveBuildAs', '/Leagues/Mapping/Spark mapper']]);
    expect(currentBuildPath.current).toBe('/Leagues/Mapping/Spark mapper');
  });

  // Build.lua:1400, ImportTab.lua:1925-1931
  it("starts from the account import's name, else the build's file name", async () => {
    seed();
    const Dlg = (await import('./SaveAsDialog.svelte')).default;
    currentBuildPath.current = '';
    unsavedName.predefined = 'Vilsol - Zed';
    const { unmount } = render(Dlg, { open: true });
    const field = () => within(screen.getByRole('dialog', { name: 'Save build as' })).getByRole<HTMLInputElement>('textbox', { name: 'Build name' });
    expect(field().value).toBe('Vilsol - Zed');
    unmount();
    currentBuildPath.current = '/Leagues/Spark';
    render(Dlg, { open: true });
    expect(field().value).toBe('Spark');
    unsavedName.predefined = undefined;
    currentBuildPath.current = '';
  });

  // Build.lua:1384-1396
  it('asks before saving over a build of that name in that folder', async () => {
    seed();
    calls.length = 0;
    currentBuildPath.current = '';
    const Dlg = (await import('./SaveAsDialog.svelte')).default;
    const onsaved = vi.fn();
    render(Dlg, { open: true, onsaved });
    const d = within(screen.getByRole('dialog', { name: 'Save build as' }));
    const folder = d.getByRole<HTMLSelectElement>('combobox', { name: 'Folder' });
    await waitFor(() => expect(folder.options.length).toBe(3));
    await fireEvent.change(folder, { target: { value: 'Leagues' } });
    const name = d.getByRole<HTMLInputElement>('textbox', { name: 'Build name' });
    await fireEvent.input(name, { target: { value: 'Spark' } });
    await fireEvent.click(d.getByRole('button', { name: 'Save' }));
    const ask = within(screen.getByRole('dialog', { name: 'Replace “Spark”?' }));
    expect(ask.getByText(/Leagues already holds a build named Spark/)).toBeTruthy();
    await fireEvent.click(ask.getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByRole('dialog', { name: 'Replace “Spark”?' })).toBeNull();
    expect(screen.getByRole('dialog', { name: 'Save build as' })).toBeTruthy();
    expect(name.value).toBe('Spark');
    expect(calls.filter((c) => c[0] === 'SaveBuildAs')).toEqual([]);
    expect(onsaved).not.toHaveBeenCalled();

    await fireEvent.click(d.getByRole('button', { name: 'Save' }));
    await fireEvent.click(within(screen.getByRole('dialog', { name: 'Replace “Spark”?' })).getByRole('button', { name: 'Replace' }));
    await waitFor(() => expect(onsaved).toHaveBeenCalledOnce());
    expect(calls.filter((c) => c[0] === 'SaveBuildAs')).toEqual([['SaveBuildAs', '/Leagues/Spark']]);
    currentBuildPath.current = '';
  });

  // Build.lua:2069-2072
  it('shows a save that fails, and stays open', async () => {
    seed();
    const Dlg = (await import('./SaveAsDialog.svelte')).default;
    const onsaved = vi.fn();
    currentBuildPath.current = '/Old';
    const fail = fake.SaveBuildAs.mockRejectedValueOnce(new Error('disk full'));
    render(Dlg, { open: true, onsaved });
    const d = within(screen.getByRole('dialog', { name: 'Save build as' }));
    await fireEvent.input(d.getByRole('textbox', { name: 'Build name' }), { target: { value: 'Broken' } });
    await fireEvent.click(d.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(d.getByRole('alert').textContent).toContain("Couldn't save the build file:"));
    expect(d.getByRole('alert').textContent).toContain('/Broken');
    expect(onsaved).not.toHaveBeenCalled();
    expect(currentBuildPath.current).toBe('/Old');
    expect(fail).toHaveBeenCalled();
  });
});

describe('AppBuildList keys', () => {
  // BuildListControl.lua:260-280, BuildList.lua:135-198
  it('F2, Delete, and Ctrl+X / Ctrl+C then Ctrl+V', async () => {
    const { grid } = await mount();
    const top = rowOf(grid, 'Top build');
    await fireEvent.keyDown(top, { key: 'F2' });
    expect(within(dialog()).getByRole('textbox')).toBeTruthy();
    expect(dialog().textContent).toContain('Rename “Top build”');
    await fireEvent.click(within(dialog()).getByRole('button', { name: 'Cancel' }));
    await fireEvent.keyDown(top, { key: 'Delete' });
    expect(dialog().textContent).toContain('Delete “Top build”?');
    await fireEvent.click(within(dialog()).getByRole('button', { name: 'Cancel' }));
    expect(calls).toEqual([]);

    const spark = (level: string) => [...grid.querySelectorAll<HTMLElement>('[data-build="Spark"]')].find((r) => r.textContent?.includes(level))!;
    await fireEvent.keyDown(spark('80'), { key: 'x', ctrlKey: true });
    expect(spark('80').hasAttribute('data-cut')).toBe(true);
    await fireEvent.keyDown(document.body, { key: 'v', ctrlKey: true });
    await waitFor(() => expect(calls).toContainEqual(['Rename', '/Leagues/Old/Spark', '/Spark']));
    calls.length = 0;
    await waitFor(() => expect(spark('95')).toBeTruthy());
    await fireEvent.keyDown(spark('95'), { key: 'c', ctrlKey: true });
    await fireEvent.keyDown(document.body, { key: 'v', ctrlKey: true });
    await waitFor(() => expect(calls).toContainEqual(['Copy', '/Leagues/Spark', '/Spark[2]']));

    await fireEvent.keyDown(rowOf(grid, 'Top build'), { key: 'c', ctrlKey: true });
    await fireEvent.keyDown(document.body, { key: 'v', ctrlKey: true });
    expect(dialog().textContent).toContain('Copy “Top build”');
  });

  // BuildList.lua:139-160
  // Main.lua:539-541, 747
  it('keeps the sort for the next visit', async () => {
    localStorage.removeItem('options:buildSortMode');
    const first = await mount();
    const level = () => within(first.grid).getByRole('columnheader', { name: /^LEVEL/ });
    await fireEvent.click(within(level()).getByRole('button'));
    const sorted = level().getAttribute('aria-sort');
    expect(sorted).not.toBe('none');
    expect(JSON.parse(localStorage.getItem('options:buildSortMode') ?? '{}')).toMatchObject({ key: 'level' });
    cleanup();
    const again = await mount();
    expect(
      within(again.grid)
        .getByRole('columnheader', { name: /^LEVEL/ })
        .getAttribute('aria-sort')
    ).toBe(sorted);
    localStorage.removeItem('options:buildSortMode');
  });

  it('Ctrl+C / Ctrl+X on a folder, then Ctrl+V', async () => {
    const { grid } = await mount();
    const leagues = () => grid.querySelector<HTMLElement>('[data-folder="Leagues"]')!;
    await fireEvent.keyDown(within(leagues()).getByRole('button'), { key: 'x', ctrlKey: true });
    expect(leagues().hasAttribute('data-cut')).toBe(true);
    await fireEvent.keyDown(document.body, { key: 'v', ctrlKey: true });
    expect(calls.filter((c) => c[0] !== 'ListBuilds')).toEqual([]);

    await fireEvent.keyDown(within(leagues()).getByRole('button'), { key: 'c', ctrlKey: true });
    await fireEvent.keyDown(document.body, { key: 'v', ctrlKey: true });
    expect(dialog().textContent).toContain('Copy “Leagues”');
    await fireEvent.input(within(dialog()).getByRole('textbox'), { target: { value: 'Leagues 2' } });
    await fireEvent.click(within(dialog()).getByRole('button', { name: 'Copy' }));
    await waitFor(() => expect(calls).toContainEqual(['CopyFolder', '/Leagues', '/Leagues 2']));
  });

  it('asks before a folder copy merges into a folder of that name', async () => {
    seed();
    dirs.add('/Archive');
    const Page = (await import('./AppBuildList.svelte')).default;
    const grid = render(Page).getByRole('grid', { name: 'Builds' });
    await waitFor(() => expect(grid.querySelector('[data-folder="Archive"]')).toBeTruthy());
    calls.length = 0;
    const leagues = () => grid.querySelector<HTMLElement>('[data-folder="Leagues"]')!;
    const copyAs = async (name: string) => {
      await fireEvent.keyDown(within(leagues()).getByRole('button'), { key: 'c', ctrlKey: true });
      await fireEvent.keyDown(document.body, { key: 'v', ctrlKey: true });
      await fireEvent.input(within(dialog()).getByRole('textbox'), { target: { value: name } });
      await fireEvent.click(within(dialog()).getByRole('button', { name: 'Copy' }));
    };
    await copyAs('Archive');
    const ask = within(screen.getByRole('dialog', { name: 'Merge into “Archive”?' }));
    expect(ask.getByText(/All builds already holds a folder named Archive/)).toBeTruthy();
    await fireEvent.click(ask.getByRole('button', { name: 'Cancel' }));
    expect(calls.filter((c) => c[0] === 'CopyFolder')).toEqual([]);

    await copyAs('Archive');
    await fireEvent.click(within(screen.getByRole('dialog', { name: 'Merge into “Archive”?' })).getByRole('button', { name: 'Merge' }));
    await waitFor(() => expect(calls.filter((c) => c[0] === 'CopyFolder')).toEqual([['CopyFolder', '/Leagues', '/Archive']]));
  });
});
