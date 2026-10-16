// @vitest-environment jsdom
// Build.lua:1421-1448

import { expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, cleanup, within, waitFor, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted bar threw').toEqual([]);
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

const worker = vi.hoisted(() => ({
  ImportCode: vi.fn((code: string) => Promise.resolve(void code)),
  SaveBuildAs: vi.fn((path: string) => Promise.resolve(void path)),
  ClearBuild: vi.fn(() => Promise.resolve()),
  GetUndoHistoryWhenRecorded: vi.fn(() => Promise.resolve({ steps: [], undone: 0, max: 50 })),
  BuildInfo: vi.fn(() => Promise.resolve({ Main: { Version: 'test' } })),
  GetLoadouts: vi.fn(() => Promise.resolve({ loadouts: [], selected: -1 })),
  GameDataVersion: vi.fn(() => Promise.resolve('3.29')),
  ListBuilds: vi.fn(() => Promise.resolve([]))
}));
const goto = vi.hoisted(() => vi.fn(() => Promise.resolve()));
const syncWrap = new Proxy(worker, { get: (t, k: string) => (k in t ? t[k as keyof typeof t] : () => Promise.resolve(undefined)) });
vi.mock('$lib/go/worker', () => ({ syncWrap, syncWorker: undefined }));
vi.mock('$app/navigation', () => ({ goto }));

vi.stubEnv('VITE_DESKTOP', '1');
const { beforeReplace } = await import('./link_import');
const AppTopBar = (await import('./AppTopBar.svelte')).default;
const { currentBuild, currentBuildPath, markBackendAsLoaded } = await import('$lib/global');
get(markBackendAsLoaded)();
const { opening } = await import('./build_name.svelte');

vi.stubGlobal(
  'fetch',
  vi.fn(() => Promise.resolve(new Response('CODE', { status: 200, headers: { 'Content-Type': 'text/plain' } })))
);

afterEach(() => {
  for (const f of Object.values(worker)) f.mockClear();
  goto.mockClear();
  currentBuild.set(undefined);
  opening.modFlag = false;
  beforeReplace.reloading = false;
  vi.mocked(fetch).mockClear();
});

const openEdited = async (path: string) => {
  currentBuildPath.current = path;
  currentBuild.set({ n: 1 } as never);
  const view = render(AppTopBar);
  await view.findByRole('button', { name: 'Save (no unsaved changes)' });
  currentBuild.set(get(currentBuild));
  await view.findByRole('button', { name: 'Save (unsaved changes)' });
  return view;
};
const handOver = () => window.dispatchEvent(new CustomEvent('pob:desktop-import', { detail: 'gopob-file:T' }));

it('a build with nothing unsaved is replaced at once', async () => {
  currentBuildPath.current = 'clean';
  currentBuild.set({ n: 1 } as never);
  const view = render(AppTopBar);
  await view.findByRole('button', { name: 'Save (no unsaved changes)' });
  handOver();
  await waitFor(() => expect(worker.ImportCode).toHaveBeenCalledWith('CODE'));
  expect(view.queryByRole('dialog', { name: 'Save Changes' })).toBeNull();
});

it("unsaved changes ask first: Cancel keeps the build, Don't Save opens the other", async () => {
  const view = await openEdited('mine');

  handOver();
  const prompt = within(await view.findByRole('dialog', { name: 'Save Changes' }));
  prompt.getByText(/This build has unsaved changes/);
  await fireEvent.click(prompt.getByRole('button', { name: 'Cancel' }));
  await waitFor(() => expect(view.queryByRole('dialog', { name: 'Save Changes' })).toBeNull());
  await new Promise((r) => setTimeout(r, 50));
  expect(worker.ImportCode, 'Cancel replaced the build').not.toHaveBeenCalled();
  expect(worker.ClearBuild).not.toHaveBeenCalled();
  view.getByRole('button', { name: 'Save (unsaved changes)' });

  handOver();
  await fireEvent.click(within(await view.findByRole('dialog', { name: 'Save Changes' })).getByRole('button', { name: "Don't Save" }));
  await waitFor(() => expect(worker.ImportCode).toHaveBeenCalledWith('CODE'));
  expect(worker.SaveBuildAs).not.toHaveBeenCalled();
  await waitFor(() => expect(goto).toHaveBeenCalledWith('/tree'));
});

it('Save writes the build to its file, then opens the other', async () => {
  const view = await openEdited('mine');
  handOver();
  await fireEvent.click(within(await view.findByRole('dialog', { name: 'Save Changes' })).getByRole('button', { name: 'Save' }));
  await waitFor(() => expect(worker.ImportCode).toHaveBeenCalledWith('CODE'));
  expect(worker.SaveBuildAs).toHaveBeenCalledWith('mine');
  expect(worker.SaveBuildAs.mock.invocationCallOrder[0]).toBeLessThan(worker.ImportCode.mock.invocationCallOrder[0]);
  await new Promise((r) => setTimeout(r, 150));
  expect(worker.ClearBuild, 'Save closed the build instead').not.toHaveBeenCalled();
});

it('Back on unsaved changes, Save: saved, then closed', async () => {
  const view = await openEdited('mine');
  await fireEvent.click(view.getByRole('link', { name: 'Back to builds' }));
  await fireEvent.click(within(await view.findByRole('dialog', { name: 'Save Changes' })).getByRole('button', { name: 'Save' }));
  await waitFor(() => expect(worker.ClearBuild).toHaveBeenCalled());
  expect(worker.SaveBuildAs).toHaveBeenCalledWith('mine');
  expect(worker.ImportCode).not.toHaveBeenCalled();
});

it("Builds folder… over unsaved changes asks first; Don't Save picks the folder and the reload does not ask again", async () => {
  const view = await openEdited('mine');
  const picks = () => vi.mocked(fetch).mock.calls.filter(([url]) => typeof url === 'string' && url.startsWith('/desktop/builds-folder'));
  const unloadAsks = () => !window.dispatchEvent(new Event('beforeunload', { cancelable: true }));
  expect(unloadAsks(), 'the guard is off').toBe(true);

  window.dispatchEvent(new CustomEvent('pob:desktop-builds-folder', { detail: '' }));
  await fireEvent.click(within(await view.findByRole('dialog', { name: 'Save Changes' })).getByRole('button', { name: 'Cancel' }));
  await new Promise((r) => setTimeout(r, 50));
  expect(picks(), 'Cancel picked a folder').toEqual([]);

  window.dispatchEvent(new CustomEvent('pob:desktop-builds-folder', { detail: 'default' }));
  await fireEvent.click(within(await view.findByRole('dialog', { name: 'Save Changes' })).getByRole('button', { name: "Don't Save" }));
  await waitFor(() => expect(picks()).toEqual([['/desktop/builds-folder?default', { method: 'POST' }]]));

  window.dispatchEvent(new CustomEvent('pob:desktop-reload'));
  expect(unloadAsks(), 'the reload asked again').toBe(false);
});
