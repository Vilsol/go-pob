// @vitest-environment jsdom
// Build.lua:89-100

import { expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, cleanup, waitFor, fireEvent } from '@testing-library/svelte';
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
globalThis.matchMedia = ((q: string) => ({
  matches: false,
  media: q,
  addEventListener: () => {},
  removeEventListener: () => {}
})) as unknown as typeof matchMedia;

const worker = vi.hoisted(() => {
  let n = 0;
  const open = async () => (await import('$lib/global')).currentBuild.set({ n: ++n } as never);
  return {
    ImportCode: vi.fn((code: string) => open().then(() => void code)),
    OpenBuild: vi.fn((path: string) => open().then(() => void path)),
    NewBuild: vi.fn(() => open()),
    ListBuilds: vi.fn((dir: string) =>
      Promise.resolve(
        dir === '' ? [{ Name: 'Saved', Type: 'file', Class: 'Witch', LastEdit: new Date().toISOString(), Level: 90, Ascendancy: '', MainSkill: '' }] : []
      )
    ),
    GetUndoHistoryWhenRecorded: vi.fn(() => Promise.resolve({ steps: [], undone: 0, max: 50 })),
    BuildInfo: vi.fn(() => Promise.resolve({ Main: { Version: 'test' } })),
    GetLoadouts: vi.fn(() => Promise.resolve({ loadouts: [], selected: -1 })),
    GameDataVersion: vi.fn(() => Promise.resolve('3.29')),
    GetTreeVersion: vi.fn(() => Promise.resolve('3_29'))
  };
});
const goto = vi.hoisted(() => vi.fn(() => Promise.resolve()));
const syncWrap = new Proxy(worker, { get: (t, k: string) => (k in t ? t[k as keyof typeof t] : () => Promise.resolve(undefined)) });
vi.mock('$lib/go/worker', () => ({ syncWrap, syncWorker: undefined }));
vi.mock('$app/navigation', () => ({ goto, pushState: vi.fn() }));
vi.mock('$app/state', () => ({ page: { state: {}, url: new URL('http://localhost/') } }));

vi.stubEnv('VITE_DESKTOP', '1');
const { openImportLink } = await import('./link_import');
const AppTopBar = (await import('./AppTopBar.svelte')).default;
const AppBuildList = (await import('./AppBuildList.svelte')).default;
const { currentBuild, currentBuildPath, markBackendAsLoaded } = await import('$lib/global');
get(markBackendAsLoaded)();
const { opening } = await import('./build_name.svelte');

vi.stubGlobal(
  'fetch',
  vi.fn(() => Promise.resolve(new Response('CODE', { status: 200, headers: { 'Content-Type': 'text/plain' } })))
);

afterEach(() => {
  for (const f of Object.values(worker)) f.mockClear();
  currentBuild.set(undefined);
  currentBuildPath.current = undefined;
  opening.modFlag = false;
});

const UNSAVED = 'Save (unsaved changes)';
const SAVED = 'Save (no unsaved changes)';
const bar = () => {
  expect(get(currentBuild), 'no build opened').toBeTruthy();
  return render(AppTopBar);
};
const importFirst = async () => {
  await openImportLink('gopob-file:T');
  bar();
  await waitFor(() => expect(document.querySelector(`button[aria-label="${UNSAVED}"]`)).not.toBeNull());
  cleanup();
  currentBuild.set(undefined);
};
const fromList = async () => {
  const view = render(AppBuildList);
  await waitFor(() => expect(view.container.querySelector('[data-build="Saved"]')).not.toBeNull());
  return view;
};

it('a pob:// link or ?import= opens the build unsaved (Main.lua:71-80)', async () => {
  await openImportLink('gopob-file:T');
  expect(worker.ImportCode).toHaveBeenCalledWith('CODE');
  const view = bar();
  await view.findByRole('button', { name: UNSAVED });
  await new Promise((r) => setTimeout(r, 50));
  view.getByRole('button', { name: UNSAVED });
});

it('a file the desktop host hands over opens unsaved, over a saved build too', async () => {
  currentBuildPath.current = 'mine';
  await worker.OpenBuild('mine');
  const view = bar();
  await view.findByRole('button', { name: SAVED });
  window.dispatchEvent(new CustomEvent('pob:desktop-import', { detail: 'gopob-file:T' }));
  await waitFor(() => expect(worker.ImportCode).toHaveBeenCalledWith('CODE'));
  await view.findByRole('button', { name: UNSAVED });
});

it('a build opened from the list opens saved, and a change then leaves it unsaved', async () => {
  await importFirst();
  const list = await fromList();
  await fireEvent.dblClick(list.container.querySelector('[data-build="Saved"]')!);
  expect(worker.OpenBuild).toHaveBeenCalledWith('/Saved');
  await waitFor(() => expect(get(currentBuild)).toBeTruthy());
  list.unmount();
  const view = bar();
  await view.findByRole('button', { name: SAVED });
  await new Promise((r) => setTimeout(r, 50));
  view.getByRole('button', { name: SAVED });
  currentBuild.set(get(currentBuild));
  await view.findByRole('button', { name: UNSAVED });
});

it('a new build opens saved (BuildList.lua:39-41)', async () => {
  await importFirst();
  const list = await fromList();
  await fireEvent.click(list.getByRole('button', { name: 'New build' }));
  expect(worker.NewBuild).toHaveBeenCalled();
  await waitFor(() => expect(get(currentBuild)).toBeTruthy());
  list.unmount();
  const view = bar();
  await view.findByRole('button', { name: SAVED });
});
